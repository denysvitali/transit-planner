import 'package:flutter_test/flutter_test.dart';
import 'package:transit_planner/src/geocoder.dart';
import 'package:transit_planner/src/location_search_page.dart';
import 'package:transit_planner/src/models.dart';

void main() {
  group('nearestStop', () {
    const stops = <TransitStop>[
      TransitStop(id: 'a', name: 'A', latitude: 35.6800, longitude: 139.7600),
      TransitStop(id: 'b', name: 'B', latitude: 35.6900, longitude: 139.7800),
      TransitStop(id: 'c', name: 'C', latitude: 35.7000, longitude: 139.8000),
    ];

    test('returns the closest stop to the query point', () {
      final snap = nearestStop(stops, 35.681, 139.761);
      expect(snap, isNotNull);
      expect(snap!.id, 'a');
    });

    test('returns null for an empty list', () {
      expect(nearestStop(const [], 0, 0), isNull);
    });
  });

  test('haversineMeters returns small values for nearby points', () {
    final d = haversineMeters(35.681, 139.767, 35.681, 139.768);
    expect(d, greaterThan(50));
    expect(d, lessThan(150));
  });

  test('stopsSortedByDistance returns nearby stops first', () {
    const stops = <TransitStop>[
      TransitStop(
        id: 'far',
        name: 'Far',
        latitude: 35.7000,
        longitude: 139.8000,
      ),
      TransitStop(
        id: 'near',
        name: 'Near',
        latitude: 35.6810,
        longitude: 139.7610,
      ),
      TransitStop(
        id: 'mid',
        name: 'Mid',
        latitude: 35.6900,
        longitude: 139.7800,
      ),
    ];

    final sorted = stopsSortedByDistance(stops, (
      latitude: 35.6800,
      longitude: 139.7600,
    ));

    expect(sorted.map((s) => s.id), ['near', 'mid', 'far']);
  });

  test('geocodeResultsSortedByDistance returns nearby places first', () {
    const results = <GeocodeResult>[
      GeocodeResult(displayName: 'Far', latitude: 35.7000, longitude: 139.8000),
      GeocodeResult(
        displayName: 'Near',
        latitude: 35.6810,
        longitude: 139.7610,
      ),
      GeocodeResult(displayName: 'Mid', latitude: 35.6900, longitude: 139.7800),
    ];

    final sorted = geocodeResultsSortedByDistance(results, (
      latitude: 35.6800,
      longitude: 139.7600,
    ));

    expect(sorted.map((s) => s.displayName), ['Near', 'Mid', 'Far']);
  });

  test('formatDistance uses meters below one kilometer and km above it', () {
    expect(formatDistance(42.4), '42 m');
    expect(formatDistance(1250), '1.3 km');
    expect(formatDistance(10250), '10 km');
  });

  test('RoutePoint.fromStop sets snappedStop to itself', () {
    const stop = TransitStop(
      id: 'x',
      name: 'X',
      latitude: 35.0,
      longitude: 139.0,
    );
    final point = RoutePoint.fromStop(stop);
    expect(point.snappedStop?.id, 'x');
    expect(point.isStop, isTrue);
    expect(point.name, 'X');
  });
}
