import 'package:flutter_test/flutter_test.dart';
import 'package:transit_planner/src/models.dart';

void main() {
  test('itinerary computes arrival and duration from legs', () {
    final origin = TransitStop(
      id: 'a',
      name: 'A',
      latitude: 46,
      longitude: 7,
    );
    final destination = TransitStop(
      id: 'b',
      name: 'B',
      latitude: 47,
      longitude: 8,
    );
    final depart = DateTime(2026, 5, 14, 8);
    final arrival = depart.add(const Duration(minutes: 24));

    final itinerary = Itinerary(
      transfers: 0,
      walking: const Duration(minutes: 4),
      legs: [
        ItineraryLeg(
          mode: TransitMode.bus,
          from: origin,
          to: destination,
          departure: depart,
          arrival: arrival,
        ),
      ],
    );

    expect(itinerary.departure, depart);
    expect(itinerary.arrival, arrival);
    expect(itinerary.duration, const Duration(minutes: 24));
  });
}
