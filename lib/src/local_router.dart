import 'models.dart';

abstract class LocalTransitRouter {
  Future<List<Itinerary>> route(RouteRequest request);
}

class MockTransitRouter implements LocalTransitRouter {
  const MockTransitRouter();

  @override
  Future<List<Itinerary>> route(RouteRequest request) async {
    final depart = request.departure;
    final firstArrival = depart.add(const Duration(minutes: 8));
    final transferDepart = firstArrival.add(const Duration(minutes: 5));
    final finalArrival = transferDepart.add(const Duration(minutes: 14));
    final transferStop = const TransitStop(
      id: 'central',
      name: 'Central Station',
      latitude: 46.948,
      longitude: 7.439,
    );

    return [
      Itinerary(
        transfers: 1,
        walking: const Duration(minutes: 7),
        legs: [
          ItineraryLeg(
            mode: TransitMode.walk,
            from: request.origin,
            to: transferStop,
            departure: depart,
            arrival: firstArrival,
          ),
          ItineraryLeg(
            mode: TransitMode.tram,
            from: transferStop,
            to: request.destination,
            departure: transferDepart,
            arrival: finalArrival,
            routeName: 'T2',
            tripId: 'mock-trip-1',
          ),
        ],
      ),
      Itinerary(
        transfers: 0,
        walking: const Duration(minutes: 4),
        legs: [
          ItineraryLeg(
            mode: TransitMode.bus,
            from: request.origin,
            to: request.destination,
            departure: depart.add(const Duration(minutes: 3)),
            arrival: depart.add(const Duration(minutes: 35)),
            routeName: 'B12',
            tripId: 'mock-trip-2',
          ),
        ],
      ),
    ];
  }
}
