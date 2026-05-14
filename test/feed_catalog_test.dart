import 'package:flutter_test/flutter_test.dart';
import 'package:transit_planner/src/feed_catalog.dart';

void main() {
  test('country network expands to concrete feeds', () {
    final japan = findFeedById('jp-public-no-key');

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
        'kobe-shiokaze',
        'kobe-satoyama',
        'himeji-ieshima',
        'takarazuka-runrunbus',
        'nishinomiya-sakurayamanami',
        'yamatokoriyama-kingyobus',
        'rinkan-koyasan',
      ]),
    );
  });

  test('single provider feed expands to itself', () {
    final toeiTrain = findFeedById('toei-train')!;

    expect(componentFeedsFor(toeiTrain), [toeiTrain]);
  });

  test('Hakusan Meguru feed uses current CKAN resource URL', () {
    final meguru = findFeedById('kanazawa-hakusan-meguru')!;

    expect(meguru.sourceUrl, contains('50049b19-fe9f-4ca1-9ea9-9d0a24141644'));
  });
}
