enum TransitMode {
  walk,
  bus,
  tram,
  rail,
  subway,
  ferry,
}

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

class RouteRequest {
  const RouteRequest({
    required this.origin,
    required this.destination,
    required this.departure,
    required this.modes,
    required this.maxTransfers,
  });

  final TransitStop origin;
  final TransitStop destination;
  final DateTime departure;
  final Set<TransitMode> modes;
  final int maxTransfers;
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
