import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre_gl/maplibre_gl.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'app_log.dart';
import 'feed_catalog.dart';
import 'go_ffi_router.dart';
import 'local_router.dart';
import 'location_search_page.dart';
import 'models.dart';
import 'theme.dart';

// OpenFreeMap "Liberty" — community-run free vector basemap with streets,
// labels, transit lines, etc. No API key, no usage limits. CC-BY OSM data.
// https://openfreemap.org/
const _fallbackStyle = 'https://tiles.openfreemap.org/styles/liberty';

const _feedSelectionStorageKey = 'selected_feed_id';

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
  TransitFeed _activeFeed = findFeedById(kDefaultFeedId)!;
  List<TransitStop> _stops = const [];
  RoutePoint? _origin;
  RoutePoint? _destination;
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
  // Null = "depart now" (or the feed's defaultDepartureHour when no clock
  // is selected). When the user picks a time, we anchor planning to that
  // specific TimeOfDay against today's date.
  TimeOfDay? _departureTime;

  MapLibreMapController? _mapController;
  bool _styleLoaded = false;
  final List<Symbol> _markerSymbols = [];
  final List<Line> _routeLines = [];

  @override
  void initState() {
    super.initState();
    _bootstrap();
  }

  Future<void> _bootstrap() async {
    final feed = await _resolveActiveFeed();
    await _openFeed(feed);
  }

  Future<TransitFeed> _resolveActiveFeed() async {
    if (widget.router != null) {
      return _activeFeed;
    }
    final prefs = await SharedPreferences.getInstance();
    final stored = prefs.getString(_feedSelectionStorageKey);
    return findFeedById(stored ?? kDefaultFeedId) ??
        findFeedById(kDefaultFeedId)!;
  }

  Future<void> _openFeed(TransitFeed feed) async {
    final router = widget.router ?? await openFeedRouter(feed);
    final stops = await router.stops();
    if (!mounted) return;
    setState(() {
      _activeFeed = feed;
      _router = router;
      _stops = stops;
      _origin = _pickInitialOrigin(stops);
      _destination = _pickInitialDestination(stops);
      _initializing = false;
      _loading = false;
      _itineraries = const [];
    });
    await _refreshMapOverlays();
    await _plan();
  }

  Future<void> _switchFeed(TransitFeed feed) async {
    if (feed.id == _activeFeed.id) {
      return;
    }
    if (widget.router != null) {
      setState(() {
        _activeFeed = feed;
      });
      return;
    }
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_feedSelectionStorageKey, feed.id);
    // Tear the map down before swapping in the new feed. The Stack/Map
    // gets replaced by _LoadingState while we re-open the router, which
    // disposes the underlying maplibre platform view — keeping the old
    // controller/symbols around would let _refreshMapOverlays() reach a
    // dead method channel and surface as MissingPluginException.
    setState(() {
      _initializing = true;
      _itineraries = const [];
      _mapController = null;
      _styleLoaded = false;
      _markerSymbols.clear();
      _routeLines.clear();
    });
    await _openFeed(feed);
  }

  RoutePoint? _pickInitialOrigin(List<TransitStop> stops) {
    if (stops.isEmpty) return null;
    final preferred = const ['001', '101', 'wankdorf'];
    for (final id in preferred) {
      for (final stop in stops) {
        if (stop.id == id) return RoutePoint.fromStop(stop);
      }
    }
    return RoutePoint.fromStop(stops.first);
  }

  RoutePoint? _pickInitialDestination(List<TransitStop> stops) {
    if (stops.isEmpty) return null;
    final preferred = const ['027', '108', 'bern_bahnhof'];
    for (final id in preferred) {
      for (final stop in stops) {
        if (stop.id == id) return RoutePoint.fromStop(stop);
      }
    }
    return RoutePoint.fromStop(
      stops.length > 1 ? stops[stops.length ~/ 2] : stops.first,
    );
  }

  Future<void> _editOrigin() async {
    final point = await _pickPoint(title: 'Choose origin');
    if (point == null) return;
    setState(() => _origin = point);
    await _refreshMapOverlays();
  }

  Future<void> _editDestination() async {
    final point = await _pickPoint(title: 'Choose destination');
    if (point == null) return;
    setState(() => _destination = point);
    await _refreshMapOverlays();
  }

  Future<RoutePoint?> _pickPoint({required String title}) async {
    return Navigator.of(context).push<RoutePoint>(
      MaterialPageRoute<RoutePoint>(
        builder: (_) => LocationSearchPage(
          title: title,
          stops: _stops,
          feedCenter: (
            latitude: _activeFeed.centerLatitude,
            longitude: _activeFeed.centerLongitude,
          ),
        ),
      ),
    );
  }

  void _swapEndpoints() {
    setState(() {
      final tmp = _origin;
      _origin = _destination;
      _destination = tmp;
    });
    _refreshMapOverlays();
  }

  Future<void> _plan() async {
    final router = _router;
    final origin = _origin;
    final destination = _destination;
    if (router == null || origin == null || destination == null) {
      return;
    }
    final originStop = origin.snappedStop;
    final destinationStop = destination.snappedStop;
    if (originStop == null || destinationStop == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text(
            'Could not find a nearby stop for the selected location.',
          ),
        ),
      );
      return;
    }
    setState(() => _loading = true);
    if (_modes.isEmpty) {
      AppLogBuffer.instance.warning(
        'Route planning requested with no transit modes selected.',
      );
    }
    final request = RouteRequest(
      origin: originStop,
      destination: destinationStop,
      departure: _earliestDepartureForFeed(),
      modes: _modes,
      maxTransfers: _maxTransfers,
    );
    try {
      final itineraries = await router.route(request);
      final filtered = _filterByModes(itineraries);
      if (!mounted) return;
      setState(() {
        _itineraries = filtered;
        _loading = false;
      });
      await _refreshMapOverlays();
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
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Route planning failed')));
    }
  }

  /// The Toei timetable runs ~05:00–24:00 Asia/Tokyo. If the user is in
  /// another timezone or asks at 03:00, "now" returns no trips. For the
  /// initial plan, anchor to a fixed in-service moment so the home page
  /// always has something to show. A user-picked [_departureTime] always
  /// wins over the feed default.
  DateTime _earliestDepartureForFeed() {
    final now = DateTime.now();
    final picked = _departureTime;
    if (picked != null) {
      return DateTime(now.year, now.month, now.day, picked.hour, picked.minute);
    }
    final overrideHour = _activeFeed.defaultDepartureHour;
    if (overrideHour == null) return now;
    return DateTime(now.year, now.month, now.day, overrideHour, 0);
  }

  Future<void> _pickDepartureTime() async {
    final initial = _departureTime ??
        TimeOfDay.fromDateTime(_earliestDepartureForFeed());
    final picked = await showTimePicker(
      context: context,
      initialTime: initial,
      helpText: 'Departure time',
    );
    if (picked == null) return;
    setState(() => _departureTime = picked);
    await _plan();
  }

  void _clearDepartureTime() {
    if (_departureTime == null) return;
    setState(() => _departureTime = null);
    _plan();
  }

  List<Itinerary> _filterByModes(List<Itinerary> itineraries) {
    return itineraries
        .where((itinerary) {
          if (_modes.isEmpty) return false;
          for (final leg in itinerary.legs) {
            if (leg.mode == TransitMode.walk) continue;
            if (!_modes.contains(leg.mode)) {
              return false;
            }
          }
          return true;
        })
        .toList(growable: false);
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

  Future<void> _onMapStyleLoaded() async {
    _styleLoaded = true;
    await _refreshMapOverlays();
  }

  Future<void> _refreshMapOverlays() async {
    final controller = _mapController;
    if (controller == null || !_styleLoaded) return;

    final scheme = Theme.of(context).colorScheme;
    final transitColor = _hexColor(scheme.primary);
    final walkColor = _hexColor(scheme.outline);

    for (final line in _routeLines) {
      try {
        await controller.removeLine(line);
      } catch (_) {}
    }
    _routeLines.clear();
    for (final sym in _markerSymbols) {
      try {
        await controller.removeSymbol(sym);
      } catch (_) {}
    }
    _markerSymbols.clear();

    final origin = _origin;
    final destination = _destination;
    final points = <LatLng>[];

    if (origin != null) {
      points.add(LatLng(origin.latitude, origin.longitude));
      _markerSymbols.add(
        await controller.addSymbol(
          SymbolOptions(
            geometry: LatLng(origin.latitude, origin.longitude),
            iconImage: 'marker-15',
            iconSize: 2.0,
            textField: 'A',
            textOffset: const Offset(0, 1.4),
            textSize: 12,
          ),
        ),
      );
    }
    if (destination != null) {
      points.add(LatLng(destination.latitude, destination.longitude));
      _markerSymbols.add(
        await controller.addSymbol(
          SymbolOptions(
            geometry: LatLng(destination.latitude, destination.longitude),
            iconImage: 'marker-15',
            iconSize: 2.0,
            textField: 'B',
            textOffset: const Offset(0, 1.4),
            textSize: 12,
          ),
        ),
      );
    }

    if (_itineraries.isNotEmpty) {
      final itinerary = _itineraries.first;
      for (final leg in itinerary.legs) {
        final geometry = [
          LatLng(leg.from.latitude, leg.from.longitude),
          LatLng(leg.to.latitude, leg.to.longitude),
        ];
        points.addAll(geometry);
        _routeLines.add(
          await controller.addLine(
            LineOptions(
              geometry: geometry,
              lineColor: leg.mode == TransitMode.walk ? walkColor : transitColor,
              lineWidth: 4.0,
            ),
          ),
        );
      }
    }

    if (points.isNotEmpty) {
      final bounds = _boundsForPoints(points);
      if (bounds != null) {
        try {
          await controller.animateCamera(
            CameraUpdate.newLatLngBounds(
              bounds,
              left: 48,
              right: 48,
              top: 220,
              bottom: 280,
            ),
          );
        } catch (error) {
          // The maplibre platform view can briefly tear down across a feed
          // swap or hot rebuild; swallow the resulting channel miss instead
          // of crashing the plan.
          AppLogBuffer.instance.warning('animateCamera failed: $error');
        }
      }
    }
  }

  LatLngBounds? _boundsForPoints(List<LatLng> points) {
    if (points.isEmpty) return null;
    double minLat = points.first.latitude;
    double maxLat = points.first.latitude;
    double minLng = points.first.longitude;
    double maxLng = points.first.longitude;
    for (final p in points) {
      if (p.latitude < minLat) minLat = p.latitude;
      if (p.latitude > maxLat) maxLat = p.latitude;
      if (p.longitude < minLng) minLng = p.longitude;
      if (p.longitude > maxLng) maxLng = p.longitude;
    }
    if ((maxLat - minLat).abs() < 1e-6 && (maxLng - minLng).abs() < 1e-6) {
      final pad = 0.005;
      return LatLngBounds(
        southwest: LatLng(minLat - pad, minLng - pad),
        northeast: LatLng(maxLat + pad, maxLng + pad),
      );
    }
    return LatLngBounds(
      southwest: LatLng(minLat, minLng),
      northeast: LatLng(maxLat, maxLng),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: _initializing
          ? _LoadingState(feedName: _activeFeed.name)
          : Stack(
              children: [
                Positioned.fill(
                  child: MapLibreMap(
                    styleString: _fallbackStyle,
                    initialCameraPosition: CameraPosition(
                      target: LatLng(
                        _activeFeed.centerLatitude,
                        _activeFeed.centerLongitude,
                      ),
                      zoom: 12,
                    ),
                    myLocationEnabled: false,
                    compassEnabled: true,
                    onMapCreated: (c) => _mapController = c,
                    onStyleLoadedCallback: _onMapStyleLoaded,
                  ),
                ),
                const Positioned(
                  left: AppSpacing.xs,
                  bottom: AppSpacing.xs,
                  child: _MapAttribution(),
                ),
                Positioned(
                  left: 0,
                  right: 0,
                  top: 0,
                  child: SafeArea(
                    bottom: false,
                    child: _SearchHeader(
                      feed: _activeFeed,
                      origin: _origin,
                      destination: _destination,
                      onEditOrigin: _editOrigin,
                      onEditDestination: _editDestination,
                      onSwap: _swapEndpoints,
                      onSettings: () => context.push('/settings'),
                    ),
                  ),
                ),
                _ResultsSheet(
                  feed: _activeFeed,
                  feeds: kTransitFeeds,
                  itineraries: _itineraries,
                  loading: _loading,
                  modes: _modes,
                  maxTransfers: _maxTransfers,
                  departureTime: _departureTime,
                  onFeedChanged: _switchFeed,
                  onModeToggled: _setModeEnabled,
                  onMaxTransfersChanged: _setMaxTransfers,
                  onPlan: _plan,
                  onPickDepartureTime: _pickDepartureTime,
                  onClearDepartureTime: _clearDepartureTime,
                  origin: _origin,
                  destination: _destination,
                ),
              ],
            ),
    );
  }
}

/// Tiny attribution chip pinned to the bottom-left of the map. OpenFreeMap
/// tiles are built on OpenStreetMap data (ODbL) and crediting both is part
/// of the usage terms.
class _MapAttribution extends StatelessWidget {
  const _MapAttribution();

  @override
  Widget build(BuildContext context) {
    return IgnorePointer(
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: Colors.white.withValues(alpha: 0.7),
          borderRadius: BorderRadius.circular(4),
        ),
        child: const Padding(
          padding: EdgeInsets.symmetric(horizontal: 6, vertical: 2),
          child: Text(
            '© OpenStreetMap · OpenFreeMap',
            style: TextStyle(fontSize: 10, color: Colors.black87),
          ),
        ),
      ),
    );
  }
}

class _LoadingState extends StatelessWidget {
  const _LoadingState({required this.feedName});

  final String feedName;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const CircularProgressIndicator(),
          const SizedBox(height: AppSpacing.m),
          Text(
            'Loading $feedName…',
            style: Theme.of(context).textTheme.bodyMedium,
          ),
        ],
      ),
    );
  }
}

class _SearchHeader extends StatelessWidget {
  const _SearchHeader({
    required this.feed,
    required this.origin,
    required this.destination,
    required this.onEditOrigin,
    required this.onEditDestination,
    required this.onSwap,
    required this.onSettings,
  });

  final TransitFeed feed;
  final RoutePoint? origin;
  final RoutePoint? destination;
  final VoidCallback onEditOrigin;
  final VoidCallback onEditDestination;
  final VoidCallback onSwap;
  final VoidCallback onSettings;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(
        AppSpacing.s,
        AppSpacing.s,
        AppSpacing.s,
        0,
      ),
      child: Material(
        elevation: 4,
        borderRadius: BorderRadius.circular(AppRadius.m),
        color: theme.colorScheme.surface,
        child: Padding(
          padding: const EdgeInsets.all(AppSpacing.s),
          child: Column(
            children: [
              Row(
                children: [
                  Expanded(
                    child: _EndpointField(
                      icon: Icons.trip_origin,
                      iconColor: theme.colorScheme.primary,
                      label: 'From',
                      point: origin,
                      onTap: onEditOrigin,
                    ),
                  ),
                  IconButton(
                    tooltip: 'Swap',
                    onPressed: onSwap,
                    icon: const Icon(Icons.swap_vert),
                  ),
                ],
              ),
              const Divider(height: 1),
              Row(
                children: [
                  Expanded(
                    child: _EndpointField(
                      icon: Icons.place,
                      iconColor: theme.colorScheme.error,
                      label: 'To',
                      point: destination,
                      onTap: onEditDestination,
                    ),
                  ),
                  IconButton(
                    tooltip: 'Settings',
                    onPressed: onSettings,
                    icon: const Icon(Icons.settings_outlined),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _EndpointField extends StatelessWidget {
  const _EndpointField({
    required this.icon,
    required this.iconColor,
    required this.label,
    required this.point,
    required this.onTap,
  });

  final IconData icon;
  final Color iconColor;
  final String label;
  final RoutePoint? point;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final hasPoint = point != null;
    return InkWell(
      borderRadius: BorderRadius.circular(AppRadius.s),
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(
          horizontal: AppSpacing.s,
          vertical: AppSpacing.s,
        ),
        child: Row(
          children: [
            Icon(icon, color: iconColor, size: 20),
            const SizedBox(width: AppSpacing.s),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    label,
                    style: theme.textTheme.labelSmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                  Text(
                    hasPoint ? point!.name : 'Search location',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: theme.textTheme.bodyLarge?.copyWith(
                      color: hasPoint
                          ? theme.colorScheme.onSurface
                          : theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _ResultsSheet extends StatelessWidget {
  const _ResultsSheet({
    required this.feed,
    required this.feeds,
    required this.itineraries,
    required this.loading,
    required this.modes,
    required this.maxTransfers,
    required this.departureTime,
    required this.onFeedChanged,
    required this.onModeToggled,
    required this.onMaxTransfersChanged,
    required this.onPlan,
    required this.onPickDepartureTime,
    required this.onClearDepartureTime,
    required this.origin,
    required this.destination,
  });

  final TransitFeed feed;
  final List<TransitFeed> feeds;
  final List<Itinerary> itineraries;
  final bool loading;
  final Set<TransitMode> modes;
  final int maxTransfers;
  final TimeOfDay? departureTime;
  final ValueChanged<TransitFeed> onFeedChanged;
  final void Function(TransitMode, bool) onModeToggled;
  final ValueChanged<double> onMaxTransfersChanged;
  final VoidCallback onPlan;
  final VoidCallback onPickDepartureTime;
  final VoidCallback onClearDepartureTime;
  final RoutePoint? origin;
  final RoutePoint? destination;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return DraggableScrollableSheet(
      initialChildSize: 0.32,
      minChildSize: 0.14,
      maxChildSize: 0.92,
      snap: true,
      snapSizes: const [0.14, 0.32, 0.92],
      builder: (context, controller) {
        return Material(
          elevation: 8,
          borderRadius: const BorderRadius.vertical(
            top: Radius.circular(AppRadius.l),
          ),
          color: theme.colorScheme.surface,
          child: ListView(
            controller: controller,
            padding: EdgeInsets.zero,
            children: [
              const SizedBox(height: AppSpacing.xs),
              Center(
                child: Container(
                  width: 40,
                  height: 4,
                  decoration: BoxDecoration(
                    color: theme.colorScheme.outlineVariant,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ),
              const SizedBox(height: AppSpacing.s),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: AppSpacing.m),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(
                        loading
                            ? 'Planning…'
                            : itineraries.isEmpty
                                ? 'No routes yet'
                                : '${itineraries.length} route'
                                    '${itineraries.length == 1 ? '' : 's'}',
                        style: theme.textTheme.titleMedium?.copyWith(
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                    FilledButton.tonalIcon(
                      onPressed: loading ? null : onPlan,
                      icon: loading
                          ? const SizedBox.square(
                              dimension: 16,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Icon(Icons.directions),
                      label: const Text('Find routes'),
                    ),
                  ],
                ),
              ),
              Padding(
                padding: const EdgeInsets.fromLTRB(
                  AppSpacing.m,
                  AppSpacing.s,
                  AppSpacing.m,
                  AppSpacing.xs,
                ),
                child: DropdownButtonFormField<TransitFeed>(
                  isExpanded: true,
                  decoration: const InputDecoration(
                    labelText: 'Feed',
                    isDense: true,
                  ),
                  initialValue: feed,
                  items: feeds
                      .map(
                        (f) => DropdownMenuItem(
                          value: f,
                          child: Text(
                            f.name,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                      )
                      .toList(growable: false),
                  onChanged: (f) {
                    if (f != null) onFeedChanged(f);
                  },
                ),
              ),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: AppSpacing.m),
                child: Wrap(
                  spacing: AppSpacing.xs,
                  runSpacing: AppSpacing.xs,
                  children: [
                    for (final entry in const <(TransitMode, String)>[
                      (TransitMode.bus, 'Bus'),
                      (TransitMode.tram, 'Tram'),
                      (TransitMode.rail, 'Rail'),
                      (TransitMode.subway, 'Metro'),
                      (TransitMode.ferry, 'Ferry'),
                    ])
                      FilterChip(
                        selected: modes.contains(entry.$1),
                        label: Text(entry.$2),
                        avatar: Icon(_modeIcon(entry.$1), size: 18),
                        onSelected: (s) => onModeToggled(entry.$1, s),
                      ),
                  ],
                ),
              ),
              Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: AppSpacing.m,
                ),
                child: Row(
                  children: [
                    const Icon(Icons.transfer_within_a_station),
                    const SizedBox(width: AppSpacing.s),
                    Text('Max transfers', style: theme.textTheme.bodySmall),
                    Expanded(
                      child: Slider(
                        value: maxTransfers.toDouble(),
                        min: 0,
                        max: 5,
                        divisions: 5,
                        label: '$maxTransfers',
                        onChanged: onMaxTransfersChanged,
                      ),
                    ),
                    Text('$maxTransfers'),
                  ],
                ),
              ),
              Padding(
                padding: const EdgeInsets.fromLTRB(
                  AppSpacing.m,
                  0,
                  AppSpacing.m,
                  AppSpacing.s,
                ),
                child: Row(
                  children: [
                    const Icon(Icons.schedule),
                    const SizedBox(width: AppSpacing.s),
                    Text('Depart', style: theme.textTheme.bodySmall),
                    const SizedBox(width: AppSpacing.s),
                    Expanded(
                      child: ActionChip(
                        avatar: const Icon(Icons.access_time, size: 18),
                        label: Text(
                          departureTime == null
                              ? 'Now'
                              : departureTime!.format(context),
                        ),
                        onPressed: onPickDepartureTime,
                      ),
                    ),
                    if (departureTime != null)
                      IconButton(
                        tooltip: 'Reset to default',
                        icon: const Icon(Icons.close),
                        onPressed: onClearDepartureTime,
                      ),
                  ],
                ),
              ),
              const Divider(height: 1),
              if (!loading && itineraries.isEmpty)
                _NoItinerariesState(
                  origin: origin,
                  destination: destination,
                ),
              for (final itinerary in itineraries)
                Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: AppSpacing.m,
                    vertical: AppSpacing.xs,
                  ),
                  child: _ItineraryCard(itinerary: itinerary),
                ),
              const SizedBox(height: AppSpacing.xl),
            ],
          ),
        );
      },
    );
  }
}

class _NoItinerariesState extends StatelessWidget {
  const _NoItinerariesState({required this.origin, required this.destination});

  final RoutePoint? origin;
  final RoutePoint? destination;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final missing = origin == null || destination == null;
    return Padding(
      padding: const EdgeInsets.symmetric(
        vertical: AppSpacing.l,
        horizontal: AppSpacing.m,
      ),
      child: Column(
        children: [
          Icon(
            Icons.directions_subway_filled_outlined,
            color: theme.colorScheme.outline,
            size: 32,
          ),
          const SizedBox(height: AppSpacing.s),
          Text('No itineraries yet', style: theme.textTheme.titleMedium),
          const SizedBox(height: AppSpacing.xs),
          Text(
            missing
                ? 'Pick a start and a destination above, then tap Find routes.'
                : 'Tap Find routes to plan a trip.',
            style: theme.textTheme.bodySmall,
            textAlign: TextAlign.center,
          ),
        ],
      ),
    );
  }
}

class _ItineraryCard extends StatelessWidget {
  const _ItineraryCard({required this.itinerary});

  final Itinerary itinerary;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return InkWell(
      onTap: () => context.push('/itinerary', extra: itinerary),
      child: Card(
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
      ),
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

String _hexColor(Color color) {
  int channel(double v) => (v * 255.0).round().clamp(0, 255);
  final r = channel(color.r).toRadixString(16).padLeft(2, '0');
  final g = channel(color.g).toRadixString(16).padLeft(2, '0');
  final b = channel(color.b).toRadixString(16).padLeft(2, '0');
  return '#$r$g$b';
}
