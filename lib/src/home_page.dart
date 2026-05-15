import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart' show setEquals;
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre_gl/maplibre_gl.dart';

import 'app_log.dart';
import 'feed_catalog.dart';
import 'feed_load_progress.dart';
import 'go_ffi_router.dart';
import 'itinerary_formatter.dart';
import 'local_router.dart';
import 'location_search_page.dart';
import 'models.dart';
import 'network_selection.dart';
import 'theme.dart';

// OpenFreeMap "Liberty" — community-run free vector basemap with streets,
// labels, transit lines, etc. No API key, no usage limits. CC-BY OSM data.
// https://openfreemap.org/
const _fallbackStyle = 'https://tiles.openfreemap.org/styles/liberty';
const _maxStopPins = 240;

class HomePage extends StatefulWidget {
  const HomePage({super.key, this.router});

  /// Optional injected router. When null, the page loads the real Go-FFI
  /// router on init. Tests use this slot to pin a deterministic mock.
  final LocalTransitRouter? router;

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  LocalTransitRouter? _router;
  TransitFeed _activeFeed = NetworkSelection.instance.feed;
  Set<String> _activeFeedIds = NetworkSelection.instance.selectedFeedIds;
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
  int _selectedItineraryIndex = 0;
  // Null = depart at the current date/time. A user-picked value pins both
  // date and clock time until it is cleared.
  DateTime? _departureOverride;
  FeedLoadProgress? _feedProgress;
  String? _loadError;

  MapLibreMapController? _mapController;
  bool _styleLoaded = false;
  bool _locationLayerEnabled = false;
  LocationCoordinate? _lastUserLocation;
  final List<Symbol> _markerSymbols = [];
  final List<Line> _routeLines = [];
  final List<Circle> _stopCircles = [];
  int _feedOpenSeq = 0;
  int _planSeq = 0;

  @override
  void initState() {
    super.initState();
    NetworkSelection.instance.addListener(_handleNetworkSelectionChanged);
    _bootstrap();
  }

  @override
  void dispose() {
    NetworkSelection.instance.removeListener(_handleNetworkSelectionChanged);
    _router?.close();
    super.dispose();
  }

  Future<void> _bootstrap() async {
    final seq = _feedOpenSeq;
    await NetworkSelection.instance.load();
    if (!mounted || _feedOpenSeq != seq) return;
    await _openFeed(NetworkSelection.instance.feed);
  }

  void _handleNetworkSelectionChanged() {
    final feed = NetworkSelection.instance.feed;
    final feedIds = NetworkSelection.instance.selectedFeedIds;
    if (setEquals(feedIds, _activeFeedIds)) return;
    _openFeed(feed);
  }

  Future<void> _openFeed(TransitFeed feed) async {
    final seq = ++_feedOpenSeq;
    _planSeq++;
    if (!NetworkSelection.instance.hasSelectedFeeds) {
      final previousRouter = _router;
      if (mounted) {
        setState(() {
          _activeFeed = feed;
          _activeFeedIds = NetworkSelection.instance.selectedFeedIds;
          _router = null;
          _stops = const [];
          _origin = null;
          _destination = null;
          _initializing = false;
          _loading = false;
          _loadError = 'Select one or more Transitland feeds in Settings.';
          _feedProgress = null;
          _itineraries = const [];
          _selectedItineraryIndex = 0;
          _markerSymbols.clear();
          _routeLines.clear();
          _stopCircles.clear();
        });
      }
      await previousRouter?.close();
      return;
    }
    if (mounted) {
      setState(() {
        _activeFeed = feed;
        _activeFeedIds = NetworkSelection.instance.selectedFeedIds;
        _initializing = true;
        _loadError = null;
        _feedProgress = null;
        _itineraries = const [];
        _selectedItineraryIndex = 0;
        _mapController = null;
        _styleLoaded = false;
        _markerSymbols.clear();
        _routeLines.clear();
        _stopCircles.clear();
      });
    }
    try {
      final router =
          widget.router ??
          await openFeedRouter(
            feed,
            onProgress: (progress) => _handleFeedProgress(seq, progress),
          );
      final stops = await router.stops();
      if (!mounted || seq != _feedOpenSeq) {
        await router.close();
        return;
      }
      final previousRouter = _router;
      setState(() {
        _activeFeed = feed;
        _router = router;
        _stops = stops;
        _origin = _pickInitialOrigin(stops);
        _destination = _pickInitialDestination(stops);
        _initializing = false;
        _loading = false;
        _loadError = null;
        _feedProgress = null;
        _itineraries = const [];
        _selectedItineraryIndex = 0;
      });
      if (previousRouter != null && previousRouter != router) {
        await previousRouter.close();
      }
      await _refreshMapOverlays();
      await _plan();
    } catch (error, stackTrace) {
      if (!mounted || seq != _feedOpenSeq) return;
      AppLogBuffer.instance.error(
        error,
        stackTrace: stackTrace,
        context: 'Failed to load feed ${feed.id}',
      );
      setState(() {
        _router = null;
        _stops = const [];
        _origin = null;
        _destination = null;
        _initializing = false;
        _loading = false;
        _loadError = error.toString();
        _itineraries = const [];
        _selectedItineraryIndex = 0;
      });
    }
  }

  void _handleFeedProgress(int seq, FeedLoadProgress progress) {
    if (!mounted || seq != _feedOpenSeq) return;
    setState(() => _feedProgress = progress);
  }

  void _openSettings() {
    context.go('/settings');
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
    final point = await _pickPoint(
      title: 'Choose origin',
      allowCurrentLocation: true,
    );
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

  Future<RoutePoint?> _pickPoint({
    required String title,
    bool allowCurrentLocation = false,
  }) async {
    return Navigator.of(context).push<RoutePoint>(
      MaterialPageRoute<RoutePoint>(
        builder: (_) => LocationSearchPage(
          title: title,
          stops: _stops,
          feedCenter: (
            latitude: _activeFeed.centerLatitude,
            longitude: _activeFeed.centerLongitude,
          ),
          allowCurrentLocation: allowCurrentLocation,
          currentLocationProvider: allowCurrentLocation
              ? _resolveCurrentLocation
              : null,
          initialNearbyLocation: _lastUserLocation,
        ),
      ),
    );
  }

  Future<LocationCoordinate?> _resolveCurrentLocation() async {
    final cached = _lastUserLocation;
    if (cached != null) return cached;
    final controller = _mapController;
    if (controller == null) {
      AppLogBuffer.instance.warning('Location requested before map is ready.');
      return null;
    }
    if (!_locationLayerEnabled && mounted) {
      setState(() => _locationLayerEnabled = true);
      await Future<void>.delayed(const Duration(milliseconds: 500));
    }
    for (var attempt = 0; attempt < 10; attempt++) {
      LatLng? latLng;
      try {
        latLng = await controller.requestMyLocationLatLng();
      } catch (error) {
        AppLogBuffer.instance.warning('Location lookup failed: $error');
        return null;
      }
      if (latLng != null) {
        final location = (
          latitude: latLng.latitude,
          longitude: latLng.longitude,
        );
        _lastUserLocation = location;
        return location;
      }
      await Future<void>.delayed(const Duration(milliseconds: 300));
    }
    AppLogBuffer.instance.warning(
      'Location lookup timed out. Location permission may be denied.',
    );
    return null;
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
    final seq = ++_planSeq;
    final modes = Set<TransitMode>.of(_modes);
    setState(() => _loading = true);
    if (modes.isEmpty) {
      AppLogBuffer.instance.warning(
        'Route planning requested with no transit modes selected.',
      );
    }
    final request = RouteRequest(
      origin: originStop,
      destination: destinationStop,
      departure: _earliestDepartureForFeed(),
      modes: modes,
      maxTransfers: _maxTransfers,
      originPoint: origin,
      destinationPoint: destination,
    );
    try {
      final itineraries = await router.route(request);
      final filtered = _filterByModes(itineraries, modes);
      AppLogBuffer.instance.info(
        'Planning result: ${itineraries.length} itinerar'
        '${itineraries.length == 1 ? 'y' : 'ies'} '
        '(${filtered.length} after mode filter)',
      );
      if (!mounted || seq != _planSeq) return;
      setState(() {
        _itineraries = filtered;
        _selectedItineraryIndex = 0;
        _loading = false;
      });
      await _refreshMapOverlays();
    } catch (error, stackTrace) {
      if (!mounted || seq != _planSeq) return;
      AppLogBuffer.instance.error(
        error,
        stackTrace: stackTrace,
        context: 'Route planning failed',
      );
      setState(() {
        _itineraries = const [];
        _selectedItineraryIndex = 0;
        _loading = false;
      });
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Route planning failed')));
    }
  }

  DateTime _earliestDepartureForFeed() {
    return _departureOverride ?? DateTime.now();
  }

  Future<void> _pickDepartureDate() async {
    final current = _earliestDepartureForFeed();
    final picked = await showDatePicker(
      context: context,
      initialDate: current,
      firstDate: current.subtract(const Duration(days: 1)),
      lastDate: current.add(const Duration(days: 30)),
      helpText: 'Departure date',
    );
    if (picked == null) return;
    setState(() {
      _departureOverride = DateTime(
        picked.year,
        picked.month,
        picked.day,
        current.hour,
        current.minute,
      );
    });
    await _plan();
  }

  Future<void> _pickDepartureTime() async {
    final current = _earliestDepartureForFeed();
    final initial = TimeOfDay.fromDateTime(current);
    final picked = await showTimePicker(
      context: context,
      initialTime: initial,
      helpText: 'Departure time',
    );
    if (picked == null) return;
    setState(() {
      _departureOverride = DateTime(
        current.year,
        current.month,
        current.day,
        picked.hour,
        picked.minute,
      );
    });
    await _plan();
  }

  void _clearDepartureTime() {
    if (_departureOverride == null) return;
    setState(() => _departureOverride = null);
    _plan();
  }

  void _selectItinerary(int index) {
    if (index == _selectedItineraryIndex) return;
    setState(() => _selectedItineraryIndex = index);
    _refreshMapOverlays();
  }

  Future<void> _copyItinerary(Itinerary itinerary) async {
    await Clipboard.setData(
      ClipboardData(text: formatItineraryDetails(itinerary)),
    );
    if (!mounted) return;
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(const SnackBar(content: Text('Trip details copied')));
  }

  List<Itinerary> _filterByModes(
    List<Itinerary> itineraries,
    Set<TransitMode> modes,
  ) {
    return itineraries
        .where((itinerary) {
          if (modes.isEmpty) return false;
          for (final leg in itinerary.legs) {
            if (leg.mode == TransitMode.walk) continue;
            if (!modes.contains(leg.mode)) {
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
    final stopColor = _hexColor(scheme.secondary);
    final stopStrokeColor = _hexColor(scheme.surface);

    final staleRouteLines = List<Line>.of(_routeLines);
    _routeLines.clear();
    final staleMarkerSymbols = List<Symbol>.of(_markerSymbols);
    _markerSymbols.clear();
    final staleStopCircles = List<Circle>.of(_stopCircles);
    _stopCircles.clear();

    for (final line in staleRouteLines) {
      try {
        await controller.removeLine(line);
      } catch (_) {}
    }
    for (final sym in staleMarkerSymbols) {
      try {
        await controller.removeSymbol(sym);
      } catch (_) {}
    }
    for (final circle in staleStopCircles) {
      try {
        await controller.removeCircle(circle);
      } catch (_) {}
    }

    final origin = _origin;
    final destination = _destination;
    final points = <LatLng>[];
    final anchorPoints = <LocationCoordinate>[];

    if (origin != null) {
      points.add(LatLng(origin.latitude, origin.longitude));
      anchorPoints.add((
        latitude: origin.latitude,
        longitude: origin.longitude,
      ));
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
      anchorPoints.add((
        latitude: destination.latitude,
        longitude: destination.longitude,
      ));
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
      final index = _selectedItineraryIndex.clamp(0, _itineraries.length - 1);
      final itinerary = _itineraries[index];
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
              lineColor: leg.mode == TransitMode.walk
                  ? walkColor
                  : transitColor,
              lineWidth: 4.0,
            ),
          ),
        );
      }
    }

    if (anchorPoints.isEmpty) {
      anchorPoints.add((
        latitude: _activeFeed.centerLatitude,
        longitude: _activeFeed.centerLongitude,
      ));
    }
    final stopPins = _stopsForMap(anchorPoints);
    if (stopPins.isNotEmpty) {
      _stopCircles.addAll(
        await controller.addCircles([
          for (final stop in stopPins)
            CircleOptions(
              geometry: LatLng(stop.latitude, stop.longitude),
              circleRadius: 4.5,
              circleColor: stopColor,
              circleOpacity: 0.75,
              circleStrokeColor: stopStrokeColor,
              circleStrokeWidth: 1.5,
              circleStrokeOpacity: 0.9,
            ),
        ]),
      );
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

  List<TransitStop> _stopsForMap(List<LocationCoordinate> anchors) {
    if (_stops.length <= _maxStopPins) {
      return _stops;
    }
    final nearest = <({TransitStop stop, double distance})>[];
    for (final stop in _stops) {
      final candidate = (
        stop: stop,
        distance: _nearestAnchorDistance(stop, anchors),
      );
      if (nearest.length < _maxStopPins) {
        nearest.add(candidate);
        continue;
      }
      var farthestIndex = 0;
      var farthestDistance = nearest.first.distance;
      for (var i = 1; i < nearest.length; i++) {
        if (nearest[i].distance > farthestDistance) {
          farthestIndex = i;
          farthestDistance = nearest[i].distance;
        }
      }
      if (candidate.distance < farthestDistance) {
        nearest[farthestIndex] = candidate;
      }
    }
    nearest.sort((a, b) => a.distance.compareTo(b.distance));
    return [for (final candidate in nearest) candidate.stop];
  }

  double _nearestAnchorDistance(
    TransitStop stop,
    List<LocationCoordinate> anchors,
  ) {
    var best = double.infinity;
    for (final anchor in anchors) {
      final distance = haversineMeters(
        anchor.latitude,
        anchor.longitude,
        stop.latitude,
        stop.longitude,
      );
      if (distance < best) best = distance;
    }
    return best;
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
      body: _loadError != null
          ? _LoadErrorState(
              feedName: _activeFeed.name,
              message: _loadError!,
              onRetry: () => _openFeed(_activeFeed),
              onSettings: () => _openSettings(),
            )
          : _initializing
          ? _LoadingState(feedName: _activeFeed.name, progress: _feedProgress)
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
                    myLocationEnabled: _locationLayerEnabled,
                    compassEnabled: true,
                    onMapCreated: (c) => _mapController = c,
                    onStyleLoadedCallback: _onMapStyleLoaded,
                    onUserLocationUpdated: (location) {
                      _lastUserLocation = (
                        latitude: location.position.latitude,
                        longitude: location.position.longitude,
                      );
                    },
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
                      origin: _origin,
                      destination: _destination,
                      onEditOrigin: _editOrigin,
                      onEditDestination: _editDestination,
                      onSwap: _swapEndpoints,
                    ),
                  ),
                ),
                _ResultsSheet(
                  itineraries: _itineraries,
                  selectedIndex: _selectedItineraryIndex,
                  loading: _loading,
                  modes: _modes,
                  maxTransfers: _maxTransfers,
                  departure: _earliestDepartureForFeed(),
                  hasDepartureOverride: _departureOverride != null,
                  onModeToggled: _setModeEnabled,
                  onMaxTransfersChanged: _setMaxTransfers,
                  onPlan: _plan,
                  onPickDepartureDate: _pickDepartureDate,
                  onPickDepartureTime: _pickDepartureTime,
                  onClearDepartureTime: _clearDepartureTime,
                  onItinerarySelected: _selectItinerary,
                  onCopyItinerary: _copyItinerary,
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
  const _LoadingState({required this.feedName, required this.progress});

  final String feedName;
  final FeedLoadProgress? progress;

  @override
  Widget build(BuildContext context) {
    final progress = this.progress;
    final value = progress?.fraction;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.l),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 420),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              LinearProgressIndicator(value: value),
              const SizedBox(height: AppSpacing.m),
              Text(
                'Loading $feedName',
                style: Theme.of(context).textTheme.titleMedium,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: AppSpacing.xs),
              Text(
                progress == null ? 'Preparing feed' : _progressText(progress),
                style: Theme.of(context).textTheme.bodySmall,
                textAlign: TextAlign.center,
              ),
              if (progress?.bytesReceived != null) ...[
                const SizedBox(height: AppSpacing.xs),
                Text(
                  _byteProgressText(progress!),
                  style: Theme.of(context).textTheme.labelSmall,
                  textAlign: TextAlign.center,
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _LoadErrorState extends StatelessWidget {
  const _LoadErrorState({
    required this.feedName,
    required this.message,
    required this.onRetry,
    required this.onSettings,
  });

  final String feedName;
  final String message;
  final VoidCallback onRetry;
  final VoidCallback onSettings;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return SafeArea(
      child: Center(
        child: Padding(
          padding: const EdgeInsets.all(AppSpacing.l),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 520),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Icon(
                  Icons.error_outline,
                  color: theme.colorScheme.error,
                  size: 36,
                ),
                const SizedBox(height: AppSpacing.m),
                Text(
                  'Could not load $feedName',
                  style: theme.textTheme.titleLarge,
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: AppSpacing.s),
                Text(
                  message,
                  style: theme.textTheme.bodySmall,
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: AppSpacing.m),
                Wrap(
                  alignment: WrapAlignment.center,
                  spacing: AppSpacing.s,
                  runSpacing: AppSpacing.s,
                  children: [
                    FilledButton.icon(
                      onPressed: onRetry,
                      icon: const Icon(Icons.refresh),
                      label: const Text('Retry'),
                    ),
                    OutlinedButton.icon(
                      onPressed: onSettings,
                      icon: const Icon(Icons.settings_outlined),
                      label: const Text('Settings'),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _SearchHeader extends StatelessWidget {
  const _SearchHeader({
    required this.origin,
    required this.destination,
    required this.onEditOrigin,
    required this.onEditDestination,
    required this.onSwap,
  });

  final RoutePoint? origin;
  final RoutePoint? destination;
  final VoidCallback onEditOrigin;
  final VoidCallback onEditDestination;
  final VoidCallback onSwap;

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
    required this.itineraries,
    required this.selectedIndex,
    required this.loading,
    required this.modes,
    required this.maxTransfers,
    required this.departure,
    required this.hasDepartureOverride,
    required this.onModeToggled,
    required this.onMaxTransfersChanged,
    required this.onPlan,
    required this.onPickDepartureDate,
    required this.onPickDepartureTime,
    required this.onClearDepartureTime,
    required this.onItinerarySelected,
    required this.onCopyItinerary,
    required this.origin,
    required this.destination,
  });

  final List<Itinerary> itineraries;
  final int selectedIndex;
  final bool loading;
  final Set<TransitMode> modes;
  final int maxTransfers;
  final DateTime departure;
  final bool hasDepartureOverride;
  final void Function(TransitMode, bool) onModeToggled;
  final ValueChanged<double> onMaxTransfersChanged;
  final VoidCallback onPlan;
  final VoidCallback onPickDepartureDate;
  final VoidCallback onPickDepartureTime;
  final VoidCallback onClearDepartureTime;
  final ValueChanged<int> onItinerarySelected;
  final ValueChanged<Itinerary> onCopyItinerary;
  final RoutePoint? origin;
  final RoutePoint? destination;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return DraggableScrollableSheet(
      initialChildSize: 0.28,
      minChildSize: 0.16,
      maxChildSize: 0.88,
      snap: true,
      snapSizes: const [0.16, 0.28, 0.88],
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
              if (loading) ...[
                const SizedBox(height: AppSpacing.s),
                const Padding(
                  padding: EdgeInsets.symmetric(horizontal: AppSpacing.m),
                  child: LinearProgressIndicator(),
                ),
              ],
              ExpansionTile(
                tilePadding: const EdgeInsets.symmetric(
                  horizontal: AppSpacing.m,
                ),
                childrenPadding: const EdgeInsets.fromLTRB(
                  AppSpacing.m,
                  0,
                  AppSpacing.m,
                  AppSpacing.s,
                ),
                title: const Text('Options'),
                subtitle: Text(_optionsLabel(departure, hasDepartureOverride)),
                children: [
                  Align(
                    alignment: Alignment.centerLeft,
                    child: Wrap(
                      spacing: AppSpacing.s,
                      runSpacing: AppSpacing.xs,
                      crossAxisAlignment: WrapCrossAlignment.center,
                      children: [
                        ActionChip(
                          avatar: const Icon(Icons.calendar_today, size: 18),
                          label: Text(_dateLabel(departure)),
                          onPressed: onPickDepartureDate,
                        ),
                        ActionChip(
                          avatar: const Icon(Icons.access_time, size: 18),
                          label: Text(
                            hasDepartureOverride
                                ? _clock(departure)
                                : 'Now ${_clock(departure)}',
                          ),
                          onPressed: onPickDepartureTime,
                        ),
                        if (hasDepartureOverride)
                          TextButton.icon(
                            icon: const Icon(Icons.restart_alt),
                            label: const Text('Use current time'),
                            onPressed: onClearDepartureTime,
                          ),
                      ],
                    ),
                  ),
                  const SizedBox(height: AppSpacing.s),
                  Align(
                    alignment: Alignment.centerLeft,
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
                  Row(
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
                ],
              ),
              const Divider(height: 1),
              if (!loading && itineraries.isEmpty)
                _NoItinerariesState(origin: origin, destination: destination),
              for (final entry in itineraries.indexed)
                Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: AppSpacing.m,
                    vertical: AppSpacing.xs,
                  ),
                  child: _ItineraryCard(
                    itinerary: entry.$2,
                    selected: entry.$1 == selectedIndex,
                    onTap: () => onItinerarySelected(entry.$1),
                    onCopy: () => onCopyItinerary(entry.$2),
                    onOpenDetails: () =>
                        context.push('/itinerary', extra: entry.$2),
                  ),
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
  const _ItineraryCard({
    required this.itinerary,
    required this.selected,
    required this.onTap,
    required this.onCopy,
    required this.onOpenDetails,
  });

  final Itinerary itinerary;
  final bool selected;
  final VoidCallback onTap;
  final VoidCallback onCopy;
  final VoidCallback onOpenDetails;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return InkWell(
      onTap: onTap,
      child: Card(
        color: selected
            ? theme.colorScheme.primaryContainer.withValues(alpha: 0.35)
            : null,
        child: Padding(
          padding: const EdgeInsets.all(AppSpacing.m),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  if (selected) ...[
                    Icon(
                      Icons.check_circle,
                      color: theme.colorScheme.primary,
                      size: 20,
                    ),
                    const SizedBox(width: AppSpacing.xs),
                  ],
                  Expanded(
                    child: Text(
                      '${_clock(itinerary.departure)} - ${_clock(itinerary.arrival)}',
                      style: theme.textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w800,
                      ),
                    ),
                  ),
                  Text('${itinerary.duration.inMinutes} min'),
                  PopupMenuButton<VoidCallback>(
                    tooltip: 'Trip actions',
                    icon: const Icon(Icons.more_vert),
                    onSelected: (action) => action(),
                    itemBuilder: (context) => [
                      PopupMenuItem<VoidCallback>(
                        value: onCopy,
                        child: const Row(
                          children: [
                            Icon(Icons.copy_all_outlined),
                            SizedBox(width: AppSpacing.s),
                            Text('Copy details'),
                          ],
                        ),
                      ),
                      PopupMenuItem<VoidCallback>(
                        value: onOpenDetails,
                        child: const Row(
                          children: [
                            Icon(Icons.open_in_full),
                            SizedBox(width: AppSpacing.s),
                            Text('Open details'),
                          ],
                        ),
                      ),
                    ],
                  ),
                ],
              ),
              const SizedBox(height: AppSpacing.xs),
              Text(
                '${itinerary.transfers} transfer${itinerary.transfers == 1 ? '' : 's'} '
                '• ${itinerary.walking.inMinutes} min walk',
                style: theme.textTheme.bodySmall,
              ),
              const SizedBox(height: AppSpacing.xs),
              Row(
                children: [
                  Icon(
                    _modeIcon(_primaryMode(itinerary)),
                    size: 18,
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                  const SizedBox(width: AppSpacing.xs),
                  Expanded(
                    child: Text(
                      _itinerarySummary(itinerary),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: theme.textTheme.bodyMedium,
                    ),
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

TransitMode _primaryMode(Itinerary itinerary) {
  return itinerary.legs
      .map((leg) => leg.mode)
      .firstWhere(
        (mode) => mode != TransitMode.walk,
        orElse: () => TransitMode.walk,
      );
}

String _itinerarySummary(Itinerary itinerary) {
  final labels = <String>[];
  for (final leg in itinerary.legs) {
    if (leg.mode == TransitMode.walk) continue;
    final label = leg.routeName?.isNotEmpty == true
        ? leg.routeName!
        : _modeLabel(leg.mode);
    if (labels.isEmpty || labels.last != label) {
      labels.add(label);
    }
  }
  if (labels.isEmpty) {
    return '${itinerary.legs.first.from.name} -> ${itinerary.legs.last.to.name}';
  }
  final visible = labels.take(3).join(' -> ');
  if (labels.length <= 3) return visible;
  return '$visible +${labels.length - 3} more';
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

String _dateLabel(DateTime value) {
  final now = DateTime.now();
  if (value.year == now.year &&
      value.month == now.month &&
      value.day == now.day) {
    return 'Today';
  }
  final month = value.month.toString().padLeft(2, '0');
  final day = value.day.toString().padLeft(2, '0');
  return '${value.year}-$month-$day';
}

String _optionsLabel(DateTime departure, bool hasDepartureOverride) {
  final time = hasDepartureOverride
      ? _clock(departure)
      : 'Now ${_clock(departure)}';
  return '${_dateLabel(departure)} · $time';
}

String _progressText(FeedLoadProgress progress) {
  final component = progress.componentCount > 1
      ? ' (${progress.componentIndex}/${progress.componentCount})'
      : '';
  final feedName = progress.componentCount > 1 ? ' ${progress.feed.name}' : '';
  return switch (progress.operation) {
    FeedLoadOperation.preparing => 'Preparing feed',
    FeedLoadOperation.checkingCache => 'Checking cache$component$feedName',
    FeedLoadOperation.copyingBundledFeed =>
      'Copying bundled feed$component$feedName',
    FeedLoadOperation.downloadingFeed => 'Downloading feed$component$feedName',
    FeedLoadOperation.openingRouter => 'Opening route engine',
    FeedLoadOperation.loadingStops => 'Loading stops',
  };
}

String _byteProgressText(FeedLoadProgress progress) {
  final received = _formatBytes(progress.bytesReceived ?? 0);
  final total = progress.totalBytes;
  if (total == null || total <= 0) {
    return '$received downloaded';
  }
  return '$received / ${_formatBytes(total)} downloaded';
}

String _formatBytes(int bytes) {
  const units = ['B', 'KB', 'MB', 'GB'];
  var value = bytes.toDouble();
  var unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex++;
  }
  if (unitIndex == 0) {
    return '$bytes ${units[unitIndex]}';
  }
  return '${value.toStringAsFixed(value >= 10 ? 1 : 2)} ${units[unitIndex]}';
}

String _hexColor(Color color) {
  int channel(double v) => (v * 255.0).round().clamp(0, 255);
  final r = channel(color.r).toRadixString(16).padLeft(2, '0');
  final g = channel(color.g).toRadixString(16).padLeft(2, '0');
  final b = channel(color.b).toRadixString(16).padLeft(2, '0');
  return '#$r$g$b';
}
