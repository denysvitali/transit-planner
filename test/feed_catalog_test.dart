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
        'jbda-kaetsunou-kaetsunouippan',
        'jbda-chitetsu-chitetsushinaidensha',
        'kobe-shiokaze',
        'kobe-satoyama',
        'himeji-ieshima',
        'takarazuka-runrunbus',
        'nishinomiya-sakurayamanami',
        'yamatokoriyama-kingyobus',
        'rinkan-koyasan',
        'jbda-akashicity-tacobustacobusmini',
        'jbda-higashiomicity-higasiohmisicommunitybus',
      ]),
    );
  });

  test('regional networks include Mobility Database mirrors', () {
    final hokuriku = findFeedById('kanazawa-region')!;
    final kansai = findFeedById('kansai-public-no-key')!;

    expect(
      componentFeedsFor(hokuriku).map((feed) => feed.id),
      containsAll([
        'jbda-kaetsunou-kaetsunouippan',
        'jbda-nonoichicity-communitybus',
        'jbda-chitetsu-chitetsubus',
        'jbda-manyosen-manyosen',
      ]),
    );
    expect(
      componentFeedsFor(kansai).map((feed) => feed.id),
      containsAll([
        'jbda-akashicity-tacobustacobusmini',
        'jbda-kakogawacity-kakobuskakobusmini',
        'jbda-andotown-andocombus',
        'jbda-omihachimancity-akakonbus',
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
