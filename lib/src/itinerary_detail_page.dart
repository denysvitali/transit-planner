import 'package:flutter/material.dart';
import 'package:maplibre_gl/maplibre_gl.dart';

import 'models.dart';
import 'theme.dart';

// OpenFreeMap "Liberty" — see lib/src/home_page.dart for rationale.
const _mapStyleUrl = 'https://tiles.openfreemap.org/styles/liberty';

/// Detail view for a single [Itinerary].
///
/// Top half: a MapLibre map that draws each leg as a colored Line.
/// Bottom half: a scrollable list of legs.
class ItineraryDetailPage extends StatelessWidget {
  const ItineraryDetailPage({
    super.key,
    required this.itinerary,
    @visibleForTesting this.mapBuilder,
  });

  final Itinerary itinerary;

  /// Optional override used by tests to avoid spinning up a platform map view.
  final WidgetBuilder? mapBuilder;

  @override
  Widget build(BuildContext context) {
    final transfers = itinerary.transfers;
    final totalMinutes = itinerary.duration.inMinutes;
    return Scaffold(
      appBar: AppBar(
        title: Text(
          '$totalMinutes min · '
          '$transfers ${transfers == 1 ? 'transfer' : 'transfers'}',
        ),
      ),
      body: Column(
        children: [
          Expanded(
            child: mapBuilder != null
                ? mapBuilder!(context)
                : _ItineraryMap(itinerary: itinerary),
          ),
          const Divider(height: 1),
          Expanded(
            child: LegList(itinerary: itinerary),
          ),
        ],
      ),
    );
  }
}

/// Scrollable list of itinerary legs. Extracted so it can be tested in
/// isolation without instantiating the MapLibre platform view.
@visibleForTesting
class LegList extends StatelessWidget {
  const LegList({super.key, required this.itinerary});

  final Itinerary itinerary;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return ListView.separated(
      padding: const EdgeInsets.symmetric(vertical: AppSpacing.xs),
      itemCount: itinerary.legs.length,
      separatorBuilder: (_, _) => const Divider(height: 1),
      itemBuilder: (context, index) {
        final leg = itinerary.legs[index];
        final isTransit = leg.mode != TransitMode.walk;
        final subtitleParts = <String>[
          '${leg.from.name} → ${leg.to.name}',
        ];
        if (isTransit) {
          final tag = <String>[
            if (leg.routeName != null && leg.routeName!.isNotEmpty)
              leg.routeName!,
            if (leg.tripId != null && leg.tripId!.isNotEmpty) leg.tripId!,
          ].join(' · ');
          if (tag.isNotEmpty) subtitleParts.add(tag);
        }
        return ListTile(
          leading: CircleAvatar(
            backgroundColor: isTransit
                ? scheme.primary.withValues(alpha: 0.15)
                : scheme.surfaceContainerHighest,
            foregroundColor:
                isTransit ? scheme.primary : scheme.onSurfaceVariant,
            child: Icon(_modeIcon(leg.mode)),
          ),
          title: Text('${_clock(leg.departure)} → ${_clock(leg.arrival)}'),
          subtitle: Text(subtitleParts.join('\n')),
          isThreeLine: subtitleParts.length > 1,
        );
      },
    );
  }
}

class _ItineraryMap extends StatefulWidget {
  const _ItineraryMap({required this.itinerary});

  final Itinerary itinerary;

  @override
  State<_ItineraryMap> createState() => _ItineraryMapState();
}

class _ItineraryMapState extends State<_ItineraryMap> {
  MapLibreMapController? _controller;

  List<LatLng> _allPoints() {
    final points = <LatLng>[];
    for (final leg in widget.itinerary.legs) {
      points.add(LatLng(leg.from.latitude, leg.from.longitude));
      points.add(LatLng(leg.to.latitude, leg.to.longitude));
    }
    return points;
  }

  CameraPosition _initialCamera() {
    final points = _allPoints();
    if (points.isEmpty) {
      return const CameraPosition(target: LatLng(46.948, 7.439), zoom: 12);
    }
    double sumLat = 0;
    double sumLng = 0;
    for (final p in points) {
      sumLat += p.latitude;
      sumLng += p.longitude;
    }
    return CameraPosition(
      target: LatLng(sumLat / points.length, sumLng / points.length),
      zoom: 12,
    );
  }

  LatLngBounds? _bounds() {
    final points = _allPoints();
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
    return LatLngBounds(
      southwest: LatLng(minLat, minLng),
      northeast: LatLng(maxLat, maxLng),
    );
  }

  Future<void> _onStyleLoaded() async {
    final controller = _controller;
    if (controller == null) return;
    final scheme = Theme.of(context).colorScheme;
    final walkColor = _hexColor(scheme.outline);
    final transitColor = _hexColor(scheme.primary);
    for (final leg in widget.itinerary.legs) {
      final isTransit = leg.mode != TransitMode.walk;
      await controller.addLine(
        LineOptions(
          geometry: [
            LatLng(leg.from.latitude, leg.from.longitude),
            LatLng(leg.to.latitude, leg.to.longitude),
          ],
          lineColor: isTransit ? transitColor : walkColor,
          lineWidth: 4.0,
        ),
      );
    }
    final bounds = _bounds();
    if (bounds != null) {
      await controller.animateCamera(
        CameraUpdate.newLatLngBounds(
          bounds,
          left: 32,
          right: 32,
          top: 32,
          bottom: 32,
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return MapLibreMap(
      styleString: _mapStyleUrl,
      initialCameraPosition: _initialCamera(),
      onMapCreated: (c) => _controller = c,
      onStyleLoadedCallback: _onStyleLoaded,
      compassEnabled: true,
    );
  }
}

String _hexColor(Color color) {
  int channel(double v) => (v * 255.0).round().clamp(0, 255);
  final r = channel(color.r).toRadixString(16).padLeft(2, '0');
  final g = channel(color.g).toRadixString(16).padLeft(2, '0');
  final b = channel(color.b).toRadixString(16).padLeft(2, '0');
  return '#$r$g$b';
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

String _clock(DateTime value) {
  final h = value.hour.toString().padLeft(2, '0');
  final m = value.minute.toString().padLeft(2, '0');
  return '$h:$m';
}
