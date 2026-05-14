import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre_gl/maplibre_gl.dart';

import 'app_log.dart';
import 'go_ffi_router.dart';
import 'local_router.dart';
import 'models.dart';
import 'stop_search.dart';
import 'theme.dart';

const _fallbackStyle = 'https://demotiles.maplibre.org/style.json';

// Tokyo Station — sensible default focal point now that the bundled feed
// is Toei. Sits between every Toei subway line.
const _tokyoCenter = LatLng(35.681236, 139.767125);
const _bernCenter = LatLng(46.948, 7.439);

class HomePage extends StatefulWidget {
  const HomePage({super.key, this.router});

  /// Optional injected router. When null, the page loads the real Go-FFI
  /// router on init (and falls back to a [MockTransitRouter] if the FFI
  /// is unavailable on the current platform). Tests use this slot to pin
  /// a deterministic mock.
  final LocalTransitRouter? router;

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  LocalTransitRouter? _router;
  List<TransitStop> _stops = const [];
  TransitStop? _origin;
  TransitStop? _destination;
  final Set<TransitMode> _modes = {
    TransitMode.bus,
    TransitMode.tram,
    TransitMode.rail,
    TransitMode.subway,
  };

  bool _initializing = true;
  bool _loading = false;
  int _maxTransfers = 2;
  List<Itinerary> _itineraries = const [];

  @override
  void initState() {
    super.initState();
    _bootstrap();
  }

  Future<void> _bootstrap() async {
    final router = widget.router ?? await openToeiRouter();
    final stops = await router.stops();
    if (!mounted) return;
    setState(() {
      _router = router;
      _stops = stops;
      _origin = _pickInitialOrigin(stops);
      _destination = _pickInitialDestination(stops);
      _initializing = false;
    });
    await _plan();
  }

  /// Picks a sensible default origin. For the Toei feed we want a
  /// recognisable starting station; if the feed is the Bern mock we keep
  /// the existing Wankdorf default.
  TransitStop _pickInitialOrigin(List<TransitStop> stops) {
    if (stops.isEmpty) {
      return const TransitStop(
        id: 'origin',
        name: 'Origin',
        latitude: 35.681,
        longitude: 139.767,
      );
    }
    final preferred = const ['001', '101', 'wankdorf'];
    for (final id in preferred) {
      for (final stop in stops) {
        if (stop.id == id) return stop;
      }
    }
    return stops.first;
  }

  TransitStop _pickInitialDestination(List<TransitStop> stops) {
    if (stops.isEmpty) {
      return const TransitStop(
        id: 'destination',
        name: 'Destination',
        latitude: 35.681,
        longitude: 139.767,
      );
    }
    final preferred = const ['027', '108', 'bern_bahnhof'];
    for (final id in preferred) {
      for (final stop in stops) {
        if (stop.id == id) return stop;
      }
    }
    return stops.length > 1 ? stops[stops.length ~/ 2] : stops.first;
  }

  bool get _isTokyoFeed {
    final first = _stops.isNotEmpty ? _stops.first : null;
    if (first == null) return false;
    // Treat anything inside a coarse Japan bounding box as the Toei feed.
    return first.latitude > 30 &&
        first.latitude < 46 &&
        first.longitude > 128 &&
        first.longitude < 146;
  }

  void _setOrigin(TransitStop stop) {
    setState(() => _origin = stop);
  }

  void _setDestination(TransitStop stop) {
    setState(() => _destination = stop);
  }

  Future<void> _plan() async {
    final router = _router;
    final origin = _origin;
    final destination = _destination;
    if (router == null || origin == null || destination == null) {
      return;
    }
    setState(() => _loading = true);
    if (_modes.isEmpty) {
      AppLogBuffer.instance.warning(
        'Route planning requested with no transit modes selected.',
      );
    }
    final request = RouteRequest(
      origin: origin,
      destination: destination,
      departure: _earliestDepartureForFeed(),
      modes: _modes,
      maxTransfers: _maxTransfers,
    );
    try {
      final itineraries = await router.route(request);
      if (!mounted) return;
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
      if (!mounted) return;
      setState(() {
        _itineraries = const [];
        _loading = false;
      });
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Route planning failed')),
      );
    }
  }

  /// The Toei timetable runs ~05:00–24:00 Asia/Tokyo. If the user is in
  /// another timezone or asks at 03:00, "now" returns no trips. For the
  /// initial plan, anchor to a fixed in-service moment so the home page
  /// always has something to show.
  DateTime _earliestDepartureForFeed() {
    if (!_isTokyoFeed) return DateTime.now();
    final now = DateTime.now();
    return DateTime(now.year, now.month, now.day, 8, 0);
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

  @override
  Widget build(BuildContext context) {
    final wide = MediaQuery.sizeOf(context).width >= 860;
    return Scaffold(
      appBar: AppBar(
        title: Text(_isTokyoFeed ? 'Transit Planner — Tokyo' : 'Transit Planner'),
        actions: [
          IconButton(
            tooltip: 'Settings',
            onPressed: () => context.push('/settings'),
            icon: const Icon(Icons.settings_outlined),
          ),
          IconButton(
            tooltip: 'Refresh routes',
            onPressed: _loading || _initializing ? null : _plan,
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: SafeArea(
        child: _initializing
            ? const _LoadingState()
            : wide
                ? Row(
                    children: [
                      SizedBox(width: 420, child: _PlannerPanel(state: this)),
                      const VerticalDivider(width: 1),
                      Expanded(child: _TransitMap(center: _mapCenter)),
                    ],
                  )
                : Column(
                    children: [
                      SizedBox(height: 260, child: _TransitMap(center: _mapCenter)),
                      Expanded(child: _PlannerPanel(state: this)),
                    ],
                  ),
      ),
    );
  }

  LatLng get _mapCenter => _isTokyoFeed ? _tokyoCenter : _bernCenter;
}

class _LoadingState extends StatelessWidget {
  const _LoadingState();

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const CircularProgressIndicator(),
          const SizedBox(height: AppSpacing.m),
          Text(
            'Loading Toei GTFS…',
            style: Theme.of(context).textTheme.bodyMedium,
          ),
        ],
      ),
    );
  }
}

class _PlannerPanel extends StatelessWidget {
  const _PlannerPanel({required this.state});

  final _HomePageState state;

  @override
  Widget build(BuildContext context) {
    final stops = state._stops;
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
                StopPickerField(
                  label: 'Origin',
                  icon: Icons.trip_origin,
                  stop: state._origin,
                  stops: stops.isEmpty ? kMockStops : stops,
                  onChanged: state._setOrigin,
                ),
                const SizedBox(height: AppSpacing.s),
                StopPickerField(
                  label: 'Destination',
                  icon: Icons.place_outlined,
                  stop: state._destination,
                  stops: stops.isEmpty ? kMockStops : stops,
                  onChanged: state._setDestination,
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
        if (state._itineraries.isEmpty && !state._loading)
          const _NoItinerariesState(),
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

class _NoItinerariesState extends StatelessWidget {
  const _NoItinerariesState();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: AppSpacing.l),
      child: Column(
        children: [
          Icon(
            Icons.directions_subway_filled_outlined,
            color: theme.colorScheme.outline,
            size: 32,
          ),
          const SizedBox(height: AppSpacing.s),
          Text(
            'No itineraries yet',
            style: theme.textTheme.titleMedium,
          ),
          const SizedBox(height: AppSpacing.xs),
          Text(
            'Pick origin and destination, then tap Find routes.',
            style: theme.textTheme.bodySmall,
            textAlign: TextAlign.center,
          ),
        ],
      ),
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
  const _TransitMap({required this.center});

  final LatLng center;

  @override
  Widget build(BuildContext context) {
    return MapLibreMap(
      styleString: _fallbackStyle,
      initialCameraPosition: CameraPosition(
        target: center,
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
