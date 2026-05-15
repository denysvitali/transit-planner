import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:transit_planner/src/feed_catalog.dart';
import 'package:transit_planner/src/transitland_catalog.dart';

void main() {
  test('runtime catalog starts without committed feeds', () {
    replaceTransitFeedsForRuntime(const []);

    expect(kTransitFeeds, isEmpty);
    expect(selectableTransitFeeds(), isEmpty);
  });

  test('Transitland feed JSON becomes a downloadable runtime feed', () {
    final feed = transitFeedFromTransitlandJson({
      'id': 123,
      'onestop_id': 'f-example~jp',
      'name': 'Example Transit',
      'associated_operators': [
        {'name': 'Example Operator'},
      ],
      'license': {
        'spdx_identifier': 'CC-BY-4.0',
        'attribution_text': 'Example attribution',
      },
      'feed_state': {
        'feed_version': {
          'geometry': {
            'type': 'Polygon',
            'coordinates': [
              [
                [136.0, 36.0],
                [138.0, 36.0],
                [138.0, 38.0],
                [136.0, 38.0],
                [136.0, 36.0],
              ],
            ],
          },
        },
      },
    });

    expect(feed, isNotNull);
    expect(feed!.id, 'transitland-f-example-jp');
    expect(feed.country, 'JP');
    expect(feed.publisher, 'Example Operator');
    expect(feed.license, 'CC-BY-4.0');
    expect(feed.attribution, 'Example attribution');
    expect(feed.centerLatitude, 37);
    expect(feed.centerLongitude, 137);
    expect(
      feed.sourceUrl,
      'https://transit.land/api/v2/rest/feeds/f-example~jp/download_latest_feed_version',
    );
  });

  test('Transitland feed JSON falls back to feed key for generic names', () {
    final feed = transitFeedFromTransitlandJson({
      'id': 456,
      'onestop_id': 'f-9q9-caltrain',
      'name': 'Transitland',
      'license': {'spdx_identifier': 'CC-BY-4.0'},
      'feed_state': {
        'feed_version': {
          'geometry': {
            'type': 'Point',
            'coordinates': [-122.4, 37.7],
          },
        },
      },
    });

    expect(feed, isNotNull);
    expect(feed!.name, 'Caltrain');
    expect(feed.publisher, 'Caltrain');
    expect(feed.description, 'Caltrain GTFS feed discovered from Transitland.');
  });

  test(
    'fresh cached Transitland feeds with generic names are repaired',
    () async {
      SharedPreferences.setMockInitialValues({
        'transitland_runtime_feed_catalog_v1': encodeTransitFeeds([
          const TransitFeed(
            id: 'transitland-f-9q9-caltrain',
            name: 'Transitland',
            description: 'Transitland GTFS feed discovered from Transitland.',
            country: 'US',
            region: 'Transitland',
            publisher: 'Transitland',
            license: 'Transitland license metadata',
            sourceUrl:
                'https://transit.land/api/v2/rest/feeds/f-9q9-caltrain/download_latest_feed_version',
            localFileName: 'transitland-f-9q9-caltrain.zip',
            attribution: 'Transit data from Transitland.',
            centerLatitude: 37.7,
            centerLongitude: -122.4,
          ),
        ]),
        'transitland_runtime_feed_catalog_updated_at_v1': DateTime.now()
            .toUtc()
            .toIso8601String(),
      });

      await TransitlandCatalog.instance.load();

      final caltrain = findFeedById('transitland-f-9q9-caltrain');
      expect(caltrain, isNotNull);
      expect(caltrain!.name, 'Caltrain');
      expect(caltrain.description, contains('Caltrain GTFS feed'));
    },
  );

  test('catalog includes supplemental Kanazawa Flat Bus feed', () async {
    SharedPreferences.setMockInitialValues({});

    await TransitlandCatalog.instance.load(
      forceRefresh: true,
      client: TransitlandFeedClient(
        httpClient: MockClient(
          (_) async => http.Response('{"feeds":[],"meta":{"after":0}}', 200),
        ),
      ),
    );

    final feed = findFeedById('supplemental-kanazawa-flatbus');
    expect(feed, isNotNull);
    expect(feed!.name, 'Kanazawa Flat Bus');
    expect(feed.country, 'JP');
    expect(feed.region, 'Ishikawa');
    expect(feed.sourceUrl, contains('flatbus20260401.zip'));
  });

  test('Transitland discovery URI uses license filters', () {
    final uri = transitlandFeedsUri(after: 42);

    expect(uri.path, '/api/v2/rest/feeds');
    expect(uri.queryParameters['spec'], 'gtfs');
    expect(uri.queryParameters['fetch_error'], 'false');
    expect(uri.queryParameters['license_redistribution_allowed'], 'exclude_no');
    expect(uri.queryParameters['license_create_derived_product'], 'exclude_no');
    expect(uri.queryParameters['license_commercial_use_allowed'], 'exclude_no');
    expect(uri.queryParameters['after'], '42');
  });
}
