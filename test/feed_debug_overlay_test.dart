import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:transit_planner/src/feed_catalog.dart';
import 'package:transit_planner/src/feed_debug_overlay.dart';
import 'package:transit_planner/src/transitland_catalog.dart';

void main() {
  testWidgets(
    'loaded feeds debug view lists active feed names and stop count',
    (tester) async {
      final first = _testFeed(id: 'feed-one', name: 'One City Transit');
      final second = _testFeed(id: 'feed-two', name: 'Two City Transit');
      final third = _testFeed(id: 'feed-three', name: 'Three Rail');
      final feed = TransitFeed(
        id: 'selected-feeds',
        name: 'Selected feeds',
        description: 'Selected runtime feeds',
        publisher: 'Transitland',
        license: 'Mixed',
        sourceUrl: 'https://transit.land/api/v2/rest/feeds',
        localFileName: '',
        attribution: 'Selected feeds',
        centerLatitude: 0,
        centerLongitude: 0,
        componentFeedIds: [first.id, second.id, third.id],
      );
      TransitlandCatalog.instance.replaceForTesting([first, second, third]);
      final feedCount = componentFeedsFor(feed).length;

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: LoadedFeedsDebugView(
              feed: feed,
              stopCount: 1234,
              maxVisibleFeeds: 2,
            ),
          ),
        ),
      );

      expect(
        find.text('Loaded feeds: $feedCount | Stops: 1234'),
        findsOneWidget,
      );
      expect(find.text('One City Transit'), findsOneWidget);
      expect(find.text('Two City Transit'), findsOneWidget);
      expect(find.text('+ ${feedCount - 2} more'), findsOneWidget);
    },
  );
}

TransitFeed _testFeed({required String id, required String name}) {
  return TransitFeed(
    id: id,
    name: name,
    description: '$name description',
    publisher: '$name publisher',
    license: 'License',
    sourceUrl:
        'https://transit.land/api/v2/rest/feeds/$id/download_latest_feed_version',
    localFileName: '$id.zip',
    attribution: '$name attribution',
    centerLatitude: 1,
    centerLongitude: 2,
  );
}
