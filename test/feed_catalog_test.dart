import 'package:flutter_test/flutter_test.dart';
import 'package:transit_planner/src/feed_catalog.dart';

void main() {
  test('country network expands to concrete feeds', () {
    final japan = findFeedById('jp-public-no-key');
    final switzerland = findFeedById('ch-national');
    final italy = findFeedById('it-public-regional');

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
    expect(componentFeedsFor(switzerland!).map((feed) => feed.id), [
      'ch-aggregate-2026',
    ]);
    expect(
      componentFeedsFor(italy!).map((feed) => feed.id),
      containsAll([
        'it-rome',
        'it-milan-atm',
        'it-lombardy-trenord',
        'it-tuscany-autolinee',
        'it-trentino-extraurban',
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

  test('Italian regional networks expand to official component feeds', () {
    final tuscany = findFeedById('it-tuscany-public')!;
    final trentino = findFeedById('it-trentino-public')!;

    expect(
      componentFeedsFor(tuscany).map((feed) => feed.id),
      containsAll([
        'it-tuscany-trenitalia',
        'it-tuscany-tft',
        'it-tuscany-toremar',
        'it-tuscany-gest',
        'it-tuscany-at-nonschool',
      ]),
    );
    expect(componentFeedsFor(trentino).map((feed) => feed.id), [
      'it-trentino-urban',
      'it-trentino-extraurban',
    ]);
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
