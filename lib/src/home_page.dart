import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:maplibre_gl/maplibre_gl.dart';

import 'app_log.dart';
import 'local_router.dart';
import 'models.dart';
import 'theme.dart';

const _fallbackStyle = 'https://demotiles.maplibre.org/style.json';

class HomePage extends StatefulWidget {
  const HomePage({super.key, this.router = const MockTransitRouter()});

  final LocalTransitRouter router;

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  final _originController = TextEditingController(text: 'Wankdorf');
  final _destinationController = TextEditingController(text: 'Bern Bahnhof');
  final Set<TransitMode> _modes = {
    TransitMode.bus,
    TransitMode.tram,
    TransitMode.rail,
  };

  bool _loading = false;
  int _maxTransfers = 2;
  List<Itinerary> _itineraries = const [];

  @override
  void initState() {
    super.initState();
    _plan();
  }

  @override
  void dispose() {
    _originController.dispose();
    _destinationController.dispose();
    super.dispose();
  }

  Future<void> _plan() async {
    setState(() => _loading = true);
    if (_modes.isEmpty) {
      AppLogBuffer.instance.warning(
        'Route planning requested with no transit modes selected.',
      );
    }
    final request = RouteRequest(
      origin: TransitStop(
        id: 'origin',
        name: _originController.text.trim().isEmpty
            ? 'Origin'
            : _originController.text.trim(),
        latitude: 46.963,
        longitude: 7.465,
      ),
      destination: TransitStop(
        id: 'destination',
        name: _destinationController.text.trim().isEmpty
            ? 'Destination'
            : _destinationController.text.trim(),
        latitude: 46.948,
        longitude: 7.439,
      ),
      departure: DateTime.now(),
      modes: _modes,
      maxTransfers: _maxTransfers,
    );
    try {
      final itineraries = await widget.router.route(request);
      if (!mounted) {
        return;
      }
      setState(() {
        _itineraries = itineraries;
        _loading = false;
      });
    } catch (error, stackTrace) {
      AppLogBuffer.instance.error(
        error,
        stackTrace: stackTrace,
        context: 'Route planning failed',
      );
      if (!mounted) {
        return;
      }
      setState(() => _loading = false);
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Route planning failed')));
    }
  }

  void _setMaxTransfers(double value) {
    setState(() => _maxTransfers = value.round());
  }

  void _setModeEnabled(TransitMode mode, bool selected) {
    setState(() {
      if (selected) {
        _modes.add(mode);
      } else {
        _modes.remove(mode);
      }
    });
  }

  void _showDeveloperOptions() {
    showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder: (context) => const _DeveloperOptionsSheet(),
    );
  }

  @override
  Widget build(BuildContext context) {
    final wide = MediaQuery.sizeOf(context).width >= 860;
    return Scaffold(
      appBar: AppBar(
        title: const Text('Transit Planner'),
        actions: [
          IconButton(
            tooltip: 'Developer options',
            onPressed: _showDeveloperOptions,
            icon: const Icon(Icons.bug_report_outlined),
          ),
          IconButton(
            tooltip: 'Refresh routes',
            onPressed: _loading ? null : _plan,
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: SafeArea(
        child: wide
            ? Row(
                children: [
                  SizedBox(width: 420, child: _PlannerPanel(state: this)),
                  const VerticalDivider(width: 1),
                  const Expanded(child: _TransitMap()),
                ],
              )
            : Column(
                children: [
                  SizedBox(height: 260, child: const _TransitMap()),
                  Expanded(child: _PlannerPanel(state: this)),
                ],
              ),
      ),
    );
  }
}

class _DeveloperOptionsSheet extends StatelessWidget {
  const _DeveloperOptionsSheet();

  static const _logLevels = {AppLogLevel.warning, AppLogLevel.error};

  Future<void> _copyLogs(BuildContext context) async {
    await Clipboard.setData(
      ClipboardData(text: AppLogBuffer.instance.formatted(_logLevels)),
    );
    if (!context.mounted) {
      return;
    }
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('Warning and error logs copied')),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(
          AppSpacing.m,
          0,
          AppSpacing.m,
          AppSpacing.m,
        ),
        child: ListenableBuilder(
          listenable: AppLogBuffer.instance,
          builder: (context, _) {
            final logs = AppLogBuffer.instance.entriesFor(_logLevels);
            return Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        'Developer options',
                        style: theme.textTheme.titleLarge?.copyWith(
                          fontWeight: FontWeight.w800,
                        ),
                      ),
                    ),
                    IconButton(
                      tooltip: 'Copy warnings and errors',
                      onPressed: () => _copyLogs(context),
                      icon: const Icon(Icons.copy_all_outlined),
                    ),
                    IconButton(
                      tooltip: 'Clear logs',
                      onPressed: logs.isEmpty
                          ? null
                          : AppLogBuffer.instance.clear,
                      icon: const Icon(Icons.delete_sweep_outlined),
                    ),
                  ],
                ),
                const SizedBox(height: AppSpacing.s),
                if (logs.isEmpty)
                  const _EmptyLogState()
                else
                  Flexible(
                    child: ListView.separated(
                      shrinkWrap: true,
                      itemCount: logs.length,
                      separatorBuilder: (context, index) =>
                          const Divider(height: AppSpacing.m),
                      itemBuilder: (context, index) {
                        return _LogEntryTile(entry: logs[index]);
                      },
                    ),
                  ),
              ],
            );
          },
        ),
      ),
    );
  }
}

class _EmptyLogState extends StatelessWidget {
  const _EmptyLogState();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: AppSpacing.l),
      child: Column(
        children: [
          Icon(
            Icons.check_circle_outline,
            color: theme.colorScheme.primary,
            size: 32,
          ),
          const SizedBox(height: AppSpacing.s),
          Text('No warning or error logs', style: theme.textTheme.titleMedium),
        ],
      ),
    );
  }
}

class _LogEntryTile extends StatelessWidget {
  const _LogEntryTile({required this.entry});

  final AppLogEntry entry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final color = switch (entry.level) {
      AppLogLevel.warning => theme.colorScheme.tertiary,
      AppLogLevel.error => theme.colorScheme.error,
    };
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Icon(
          entry.level == AppLogLevel.warning
              ? Icons.warning_amber_outlined
              : Icons.error_outline,
          color: color,
        ),
        const SizedBox(width: AppSpacing.s),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                '${entry.level.name.toUpperCase()} ${_clock(entry.timestamp)}',
                style: theme.textTheme.labelLarge?.copyWith(color: color),
              ),
              const SizedBox(height: AppSpacing.xs),
              SelectableText(entry.message),
            ],
          ),
        ),
      ],
    );
  }
}

class _PlannerPanel extends StatelessWidget {
  const _PlannerPanel({required this.state});

  final _HomePageState state;

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(AppSpacing.m),
      children: [
        Card(
          child: Padding(
            padding: const EdgeInsets.all(AppSpacing.m),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  'Plan a trip',
                  style: Theme.of(context).textTheme.titleLarge,
                ),
                const SizedBox(height: AppSpacing.m),
                TextField(
                  controller: state._originController,
                  decoration: const InputDecoration(
                    labelText: 'Origin',
                    prefixIcon: Icon(Icons.trip_origin),
                  ),
                ),
                const SizedBox(height: AppSpacing.s),
                TextField(
                  controller: state._destinationController,
                  decoration: const InputDecoration(
                    labelText: 'Destination',
                    prefixIcon: Icon(Icons.place_outlined),
                  ),
                ),
                const SizedBox(height: AppSpacing.m),
                Wrap(
                  spacing: AppSpacing.xs,
                  runSpacing: AppSpacing.xs,
                  children: [
                    _ModeChip(
                      state: state,
                      mode: TransitMode.bus,
                      label: 'Bus',
                    ),
                    _ModeChip(
                      state: state,
                      mode: TransitMode.tram,
                      label: 'Tram',
                    ),
                    _ModeChip(
                      state: state,
                      mode: TransitMode.rail,
                      label: 'Rail',
                    ),
                    _ModeChip(
                      state: state,
                      mode: TransitMode.subway,
                      label: 'Metro',
                    ),
                    _ModeChip(
                      state: state,
                      mode: TransitMode.ferry,
                      label: 'Ferry',
                    ),
                  ],
                ),
                const SizedBox(height: AppSpacing.m),
                Row(
                  children: [
                    const Icon(Icons.transfer_within_a_station),
                    const SizedBox(width: AppSpacing.s),
                    Expanded(
                      child: Slider(
                        value: state._maxTransfers.toDouble(),
                        min: 0,
                        max: 5,
                        divisions: 5,
                        label: '${state._maxTransfers}',
                        onChanged: state._setMaxTransfers,
                      ),
                    ),
                    Text('${state._maxTransfers}'),
                  ],
                ),
                const SizedBox(height: AppSpacing.s),
                FilledButton.icon(
                  onPressed: state._loading ? null : state._plan,
                  icon: state._loading
                      ? const SizedBox.square(
                          dimension: 18,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Icon(Icons.route),
                  label: const Text('Find routes'),
                ),
              ],
            ),
          ),
        ),
        const SizedBox(height: AppSpacing.m),
        ...state._itineraries.map(
          (itinerary) => Padding(
            padding: const EdgeInsets.only(bottom: AppSpacing.s),
            child: _ItineraryCard(itinerary: itinerary),
          ),
        ),
      ],
    );
  }
}

class _ModeChip extends StatelessWidget {
  const _ModeChip({
    required this.state,
    required this.mode,
    required this.label,
  });

  final _HomePageState state;
  final TransitMode mode;
  final String label;

  @override
  Widget build(BuildContext context) {
    return FilterChip(
      selected: state._modes.contains(mode),
      label: Text(label),
      avatar: Icon(_modeIcon(mode), size: 18),
      onSelected: (selected) => state._setModeEnabled(mode, selected),
    );
  }
}

class _ItineraryCard extends StatelessWidget {
  const _ItineraryCard({required this.itinerary});

  final Itinerary itinerary;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.m),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    '${_clock(itinerary.departure)} - ${_clock(itinerary.arrival)}',
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w800,
                    ),
                  ),
                ),
                Text('${itinerary.duration.inMinutes} min'),
              ],
            ),
            const SizedBox(height: AppSpacing.xs),
            Text(
              '${itinerary.transfers} transfer${itinerary.transfers == 1 ? '' : 's'} '
              '• ${itinerary.walking.inMinutes} min walk',
              style: theme.textTheme.bodySmall,
            ),
            const Divider(height: AppSpacing.l),
            for (final leg in itinerary.legs)
              Padding(
                padding: const EdgeInsets.only(bottom: AppSpacing.s),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Icon(_modeIcon(leg.mode), size: 20),
                    const SizedBox(width: AppSpacing.s),
                    Expanded(
                      child: Text(
                        '${leg.routeName ?? _modeLabel(leg.mode)} '
                        '${leg.from.name} -> ${leg.to.name}',
                      ),
                    ),
                    Text('${leg.duration.inMinutes} min'),
                  ],
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class _TransitMap extends StatelessWidget {
  const _TransitMap();

  @override
  Widget build(BuildContext context) {
    return MapLibreMap(
      styleString: _fallbackStyle,
      initialCameraPosition: const CameraPosition(
        target: LatLng(46.948, 7.439),
        zoom: 12,
      ),
      myLocationEnabled: true,
      compassEnabled: true,
    );
  }
}

IconData _modeIcon(TransitMode mode) {
  return switch (mode) {
    TransitMode.walk => Icons.directions_walk,
    TransitMode.bus => Icons.directions_bus,
    TransitMode.tram => Icons.tram,
    TransitMode.rail => Icons.train,
    TransitMode.subway => Icons.subway,
    TransitMode.ferry => Icons.directions_boat,
  };
}

String _modeLabel(TransitMode mode) {
  return switch (mode) {
    TransitMode.walk => 'Walk',
    TransitMode.bus => 'Bus',
    TransitMode.tram => 'Tram',
    TransitMode.rail => 'Rail',
    TransitMode.subway => 'Metro',
    TransitMode.ferry => 'Ferry',
  };
}

String _clock(DateTime value) {
  final h = value.hour.toString().padLeft(2, '0');
  final m = value.minute.toString().padLeft(2, '0');
  return '$h:$m';
}
