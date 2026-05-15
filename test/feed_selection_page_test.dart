import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:transit_planner/src/feed_catalog.dart';
import 'package:transit_planner/src/feed_selection_page.dart';
import 'package:transit_planner/src/network_selection.dart';
import 'package:transit_planner/src/transitland_catalog.dart';

void main() {
  setUp(() async {
    SharedPreferences.setMockInitialValues({});
    TransitlandCatalog.instance.replaceForTesting([
      _testFeed(
        id: 'feed-one',
        name: 'One City Transit',
        country: 'AA',
        region: 'Central',
      ),
      _testFeed(
        id: 'feed-two',
        name: 'Two City Transit',
        country: 'AA',
        region: 'Coast',
      ),
      _testFeed(
        id: 'feed-three',
        name: 'Three Rail',
        country: 'BB',
        region: 'Rail',
      ),
    ]);
    await NetworkSelection.instance.setSelectedFeedIds(const []);
  });

  testWidgets('countries are collapsed by default', (tester) async {
    await tester.pumpWidget(const MaterialApp(home: FeedSelectionPage()));
    await tester.pumpAndSettle();

    // Country headers visible
    expect(find.widgetWithText(ListTile, 'AA'), findsOneWidget);
    expect(find.widgetWithText(ListTile, 'BB'), findsOneWidget);

    // Feed checkboxes not visible (collapsed)
    expect(
      find.widgetWithText(CheckboxListTile, 'One City Transit'),
      findsNothing,
    );
    expect(
      find.widgetWithText(CheckboxListTile, 'Two City Transit'),
      findsNothing,
    );
    expect(
      find.widgetWithText(CheckboxListTile, 'Three Rail'),
      findsNothing,
    );
  });

  testWidgets('expanding a country shows region and feed checkboxes', (
    tester,
  ) async {
    await tester.pumpWidget(const MaterialApp(home: FeedSelectionPage()));
    await tester.pumpAndSettle();

    // Tap on AA country header to expand
    await tester.tap(find.widgetWithText(ListTile, 'AA'));
    await tester.pumpAndSettle();

    // Region checkboxes visible
    expect(find.widgetWithText(CheckboxListTile, 'Central'), findsOneWidget);
    expect(find.widgetWithText(CheckboxListTile, 'Coast'), findsOneWidget);

    // Feed checkboxes visible
    expect(
      find.widgetWithText(CheckboxListTile, 'One City Transit'),
      findsOneWidget,
    );
    expect(
      find.widgetWithText(CheckboxListTile, 'Two City Transit'),
      findsOneWidget,
    );

    // BB country still collapsed
    expect(
      find.widgetWithText(CheckboxListTile, 'Three Rail'),
      findsNothing,
    );
  });

  testWidgets('select all by country checkbox then deselect individual feed', (
    tester,
  ) async {
    await NetworkSelection.instance.setSelectedFeedIds(const []);

    await tester.pumpWidget(const MaterialApp(home: FeedSelectionPage()));
    await tester.pumpAndSettle();

    // Tap country checkbox for AA (the Checkbox widget inside the ListTile)
    final aaTile = find.widgetWithText(ListTile, 'AA');
    final aaCheckbox = find.descendant(of: aaTile, matching: find.byType(Checkbox));
    await tester.tap(aaCheckbox);
    await tester.pumpAndSettle();

    expect(
      NetworkSelection.instance.selectedFeedIds,
      contains('feed-one'),
    );
    expect(
      NetworkSelection.instance.selectedFeedIds,
      contains('feed-two'),
    );
    expect(
      NetworkSelection.instance.selectedFeedIds,
      isNot(contains('feed-three')),
    );

    // Expand AA to access individual feed
    await tester.tap(aaTile);
    await tester.pumpAndSettle();

    // Deselect one feed
    await tester.tap(
      find.widgetWithText(CheckboxListTile, 'Two City Transit'),
    );
    await tester.pumpAndSettle();

    expect(
      NetworkSelection.instance.selectedFeedIds,
      contains('feed-one'),
    );
    expect(
      NetworkSelection.instance.selectedFeedIds,
      isNot(contains('feed-two')),
    );
  });
}

TransitFeed _testFeed({
  required String id,
  required String name,
  required String country,
  required String region,
}) {
  return TransitFeed(
    id: id,
    name: name,
    description: '$name description',
    country: country,
    region: region,
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
