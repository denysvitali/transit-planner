import 'package:flutter_test/flutter_test.dart';
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
