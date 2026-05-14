import 'package:flutter_test/flutter_test.dart';
import 'package:transit_planner/src/feed_catalog.dart';

void main() {
  test('country network expands to concrete feeds', () {
    final japan = findFeedById('jp-all');

    expect(japan, isNotNull);
    expect(japan!.isCollection, isTrue);
    expect(
      componentFeedsFor(japan).map((feed) => feed.id),
      containsAll([
        'toei-train',
        'toei-bus',
        'kanazawa-flatbus',
        'kanazawa-hakusan-meguru',
        'kanazawa-tsubata-bus',
      ]),
    );
  });

  test('single provider feed expands to itself', () {
    final toeiTrain = findFeedById('toei-train')!;

    expect(componentFeedsFor(toeiTrain), [toeiTrain]);
  });
}
