import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:transit_planner/src/itinerary_detail_page.dart';
import 'package:transit_planner/src/itinerary_formatter.dart';
import 'package:transit_planner/src/models.dart';

void main() {
  Itinerary buildItinerary() {
    final origin = TransitStop(
      id: 'a',
      name: 'Origin Stop',
      latitude: 46.948,
      longitude: 7.439,
    );
    final mid = TransitStop(
      id: 'b',
      name: 'Mid Stop',
      latitude: 46.95,
      longitude: 7.45,
    );
    final destination = TransitStop(
      id: 'c',
      name: 'Destination Stop',
      latitude: 46.96,
      longitude: 7.46,
    );
    final depart = DateTime(2026, 5, 14, 8);
    return Itinerary(
      transfers: 1,
      walking: const Duration(minutes: 3),
      legs: [
        ItineraryLeg(
          mode: TransitMode.walk,
          from: origin,
          to: mid,
          departure: depart,
          arrival: depart.add(const Duration(minutes: 5)),
        ),
        ItineraryLeg(
          mode: TransitMode.bus,
          from: mid,
          to: destination,
          departure: depart.add(const Duration(minutes: 6)),
          arrival: depart.add(const Duration(minutes: 20)),
          routeName: 'Bus 10',
          tripId: 'trip-42',
        ),
      ],
    );
  }

  testWidgets('LegList renders one tile per leg with stop names', (
    tester,
  ) async {
    final itinerary = buildItinerary();
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(body: LegList(itinerary: itinerary)),
      ),
    );

    expect(find.byType(ListTile), findsNWidgets(2));
    expect(find.textContaining('Origin Stop'), findsOneWidget);
    expect(find.textContaining('Mid Stop'), findsWidgets);
    expect(find.textContaining('Destination Stop'), findsOneWidget);
    expect(find.textContaining('Bus 10'), findsOneWidget);
    expect(find.textContaining('trip-42'), findsOneWidget);
    expect(find.textContaining('08:00 → 08:05'), findsOneWidget);
    expect(find.textContaining('08:06 → 08:20'), findsOneWidget);
  });

  testWidgets(
    'ItineraryDetailPage shows total duration and transfer count in app bar',
    (tester) async {
      final itinerary = buildItinerary();
      await tester.pumpWidget(
        MaterialApp(
          home: ItineraryDetailPage(
            itinerary: itinerary,
            mapBuilder: (_) => const SizedBox.shrink(),
          ),
        ),
      );

      expect(find.textContaining('20 min'), findsOneWidget);
      expect(find.textContaining('1 transfer'), findsOneWidget);
      expect(find.byTooltip('Copy trip details'), findsOneWidget);
      // Legs render in the bottom half.
      expect(find.byType(ListTile), findsNWidgets(2));
    },
  );

  test('formatItineraryDetails includes copyable leg details', () {
    final text = formatItineraryDetails(buildItinerary());

    expect(text, contains('Trip 08:00-08:20'));
    expect(text, contains('Duration: 20 min'));
    expect(text, contains('Bus 10'));
    expect(text, contains('Origin Stop -> Mid Stop'));
    expect(text, contains('Trip ID: trip-42'));
  });
}
