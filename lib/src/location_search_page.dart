import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';

import 'geocoder.dart';
import 'models.dart';
import 'theme.dart';

/// Modal "Google-Maps"-style location picker. Accepts free-form text input
/// and surfaces:
///
/// 1. GTFS stops in the active feed whose name contains the query.
/// 2. Free-form geocoder results (OpenStreetMap / Nominatim).
///
/// The picker returns a [RoutePoint] — either backed by a stop the user
/// tapped, or a geocoded address with its [RoutePoint.snappedStop] set to
/// the nearest GTFS stop in [stops] so the router has something to plan
/// against.
class LocationSearchPage extends StatefulWidget {
  const LocationSearchPage({
    super.key,
    required this.title,
    required this.stops,
    required this.feedCenter,
    this.initialQuery = '',
    this.geocoder,
  });

  final String title;
  final List<TransitStop> stops;
  final ({double latitude, double longitude}) feedCenter;
  final String initialQuery;
  final Geocoder? geocoder;

  @override
  State<LocationSearchPage> createState() => _LocationSearchPageState();
}

class _LocationSearchPageState extends State<LocationSearchPage> {
  late final TextEditingController _controller =
      TextEditingController(text: widget.initialQuery);
  late final Geocoder _geocoder = widget.geocoder ?? NominatimGeocoder();

  Timer? _debounce;
  String _query = '';
  bool _geocoding = false;
  List<GeocodeResult> _geoResults = const [];
  int _requestSeq = 0;

  @override
  void initState() {
    super.initState();
    _query = widget.initialQuery;
    if (_query.trim().length >= 2) {
      _scheduleGeocode();
    }
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _controller.dispose();
    super.dispose();
  }

  void _onChanged(String value) {
    setState(() => _query = value);
    _scheduleGeocode();
  }

  void _scheduleGeocode() {
    _debounce?.cancel();
    final trimmed = _query.trim();
    if (trimmed.length < 2) {
      setState(() {
        _geocoding = false;
        _geoResults = const [];
      });
      return;
    }
    _debounce = Timer(const Duration(milliseconds: 350), _runGeocode);
  }

  Future<void> _runGeocode() async {
    final seq = ++_requestSeq;
    setState(() => _geocoding = true);
    final results = await _geocoder.search(
      _query,
      bias: GeocodeBias(
        centerLat: widget.feedCenter.latitude,
        centerLon: widget.feedCenter.longitude,
      ),
    );
    if (!mounted || seq != _requestSeq) return;
    setState(() {
      _geocoding = false;
      _geoResults = results;
    });
  }

  List<TransitStop> get _matchingStops {
    final q = _query.trim().toLowerCase();
    if (q.isEmpty) return widget.stops.take(20).toList(growable: false);
    return widget.stops
        .where((s) => s.name.toLowerCase().contains(q))
        .take(20)
        .toList(growable: false);
  }

  void _selectStop(TransitStop stop) {
    Navigator.of(context).pop(RoutePoint.fromStop(stop));
  }

  void _selectGeocoded(GeocodeResult result) {
    final snapped = nearestStop(
      widget.stops,
      result.latitude,
      result.longitude,
    );
    final point = RoutePoint(
      name: result.shortName ?? result.displayName,
      description: result.displayName,
      latitude: result.latitude,
      longitude: result.longitude,
      snappedStop: snapped,
    );
    Navigator.of(context).pop(point);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final stops = _matchingStops;
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.title),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(64),
          child: Padding(
            padding: const EdgeInsets.fromLTRB(
              AppSpacing.m,
              0,
              AppSpacing.m,
              AppSpacing.s,
            ),
            child: TextField(
              controller: _controller,
              autofocus: true,
              textInputAction: TextInputAction.search,
              onChanged: _onChanged,
              decoration: InputDecoration(
                hintText: 'Search address, station, place…',
                prefixIcon: const Icon(Icons.search),
                suffixIcon: _query.isEmpty
                    ? null
                    : IconButton(
                        tooltip: 'Clear',
                        icon: const Icon(Icons.clear),
                        onPressed: () {
                          _controller.clear();
                          _onChanged('');
                        },
                      ),
                isDense: true,
              ),
            ),
          ),
        ),
      ),
      body: ListView(
        children: [
          if (stops.isNotEmpty) ...[
            _SectionHeader(label: 'Stops'),
            for (final stop in stops)
              ListTile(
                leading: const Icon(Icons.directions_subway_filled_outlined),
                title: Text(stop.name),
                subtitle: Text(
                  '${stop.latitude.toStringAsFixed(4)}, '
                  '${stop.longitude.toStringAsFixed(4)}',
                ),
                onTap: () => _selectStop(stop),
              ),
          ],
          _SectionHeader(
            label: 'Places',
            trailing: _geocoding
                ? const SizedBox.square(
                    dimension: 16,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : null,
          ),
          if (!_geocoding && _geoResults.isEmpty && _query.trim().length >= 2)
            Padding(
              padding: const EdgeInsets.all(AppSpacing.m),
              child: Text(
                'No matching places',
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ),
          if (_query.trim().length < 2 && _geoResults.isEmpty)
            Padding(
              padding: const EdgeInsets.all(AppSpacing.m),
              child: Text(
                'Type at least 2 characters to search any address.',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ),
          for (final result in _geoResults)
            ListTile(
              leading: const Icon(Icons.place_outlined),
              title: Text(result.shortName ?? result.displayName),
              subtitle: Text(
                result.displayName,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
              onTap: () => _selectGeocoded(result),
            ),
          const SizedBox(height: AppSpacing.xl),
        ],
      ),
    );
  }
}

class _SectionHeader extends StatelessWidget {
  const _SectionHeader({required this.label, this.trailing});

  final String label;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(
        AppSpacing.m,
        AppSpacing.m,
        AppSpacing.m,
        AppSpacing.xs,
      ),
      child: Row(
        children: [
          Expanded(
            child: Text(
              label.toUpperCase(),
              style: theme.textTheme.labelSmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
                letterSpacing: 0.8,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
          ?trailing,
        ],
      ),
    );
  }
}

/// Returns the stop in [stops] closest to ([lat], [lon]) using the haversine
/// distance. Returns null when [stops] is empty.
TransitStop? nearestStop(List<TransitStop> stops, double lat, double lon) {
  if (stops.isEmpty) return null;
  TransitStop? best;
  double bestDist = double.infinity;
  for (final stop in stops) {
    final d = haversineMeters(lat, lon, stop.latitude, stop.longitude);
    if (d < bestDist) {
      bestDist = d;
      best = stop;
    }
  }
  return best;
}

/// Haversine great-circle distance in meters.
double haversineMeters(double lat1, double lon1, double lat2, double lon2) {
  const earthRadius = 6371000.0;
  final dLat = _deg2rad(lat2 - lat1);
  final dLon = _deg2rad(lon2 - lon1);
  final a = math.sin(dLat / 2) * math.sin(dLat / 2) +
      math.cos(_deg2rad(lat1)) *
          math.cos(_deg2rad(lat2)) *
          math.sin(dLon / 2) *
          math.sin(dLon / 2);
  final c = 2 * math.atan2(math.sqrt(a), math.sqrt(1 - a));
  return earthRadius * c;
}

double _deg2rad(double deg) => deg * (math.pi / 180.0);
