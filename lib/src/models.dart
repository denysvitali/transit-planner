enum TransitMode { walk, bus, tram, rail, subway, ferry }

class TransitStop {
  const TransitStop({
    required this.id,
    required this.name,
    required this.latitude,
    required this.longitude,
  });

  final String id;
  final String name;
  final double latitude;
  final double longitude;
}

/// A trip endpoint — either a GTFS stop the user picked, or an arbitrary
/// geocoded address. In both cases we keep an optional [snappedStop] so the
/// router (which only knows GTFS stop IDs) has something to plan against.
class RoutePoint {
  const RoutePoint({
    required this.name,
    required this.latitude,
    required this.longitude,
    this.snappedStop,
    this.description,
    this.isStop = false,
  });

  /// Construct a [RoutePoint] from an existing GTFS [TransitStop].
  factory RoutePoint.fromStop(TransitStop stop) => RoutePoint(
    name: stop.name,
    latitude: stop.latitude,
    longitude: stop.longitude,
    snappedStop: stop,
    isStop: true,
  );

  final String name;
  final double latitude;
  final double longitude;

  /// The nearest GTFS stop used for actual route planning. For [RoutePoint]s
  /// that originate from a stop pick this is the stop itself; for geocoded
  /// addresses it is the closest stop in the active feed.
  final TransitStop? snappedStop;

  /// Optional secondary line (e.g. for geocoder results the city / country).
  final String? description;

  /// True when this point came directly from a GTFS stop pick.
  final bool isStop;
}

class RouteRequest {
  const RouteRequest({
    required this.origin,
    required this.destination,
    required this.departure,
    required this.modes,
    required this.maxTransfers,
    this.originPoint,
    this.destinationPoint,
  });

  final TransitStop origin;
  final TransitStop destination;
  final DateTime departure;
  final Set<TransitMode> modes;
  final int maxTransfers;
  final RoutePoint? originPoint;
  final RoutePoint? destinationPoint;
}

class ItineraryLeg {
  const ItineraryLeg({
    required this.mode,
    required this.from,
    required this.to,
    required this.departure,
    required this.arrival,
    this.routeName,
    this.tripId,
    this.routeType,
  });

  final TransitMode mode;
  final TransitStop from;
  final TransitStop to;
  final DateTime departure;
  final DateTime arrival;
  final String? routeName;
  final String? tripId;
  final int? routeType;

  Duration get duration => arrival.difference(departure);
}

class Itinerary {
  const Itinerary({
    required this.legs,
    required this.transfers,
    required this.walking,
  });

  final List<ItineraryLeg> legs;
  final int transfers;
  final Duration walking;

  DateTime get departure => legs.first.departure;
  DateTime get arrival => legs.last.arrival;
  Duration get duration => arrival.difference(departure);
}
