import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:transit_planner/src/feed_catalog.dart';
import 'package:transit_planner/src/feed_debug_overlay.dart';

void main() {
  testWidgets(
    'loaded feeds debug view lists active feed names and stop count',
    (tester) async {
      final feed = findFeedById('kanazawa-region')!;
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
      expect(find.text('Kanazawa Flat Bus'), findsOneWidget);
      expect(find.text('Hakusan Meguru'), findsOneWidget);
      expect(find.text('+ ${feedCount - 2} more'), findsOneWidget);
    },
  );
}
