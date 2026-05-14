import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:transit_planner/src/models.dart';
import 'package:transit_planner/src/stop_search.dart';

void main() {
  testWidgets(
    'StopSearchDelegate shows matching stop in a ListTile after typing',
    (tester) async {
      TransitStop? selected;

      await tester.pumpWidget(
        MaterialApp(
          home: Builder(
            builder: (context) {
              return Scaffold(
                body: Center(
                  child: ElevatedButton(
                    onPressed: () async {
                      selected = await showSearch<TransitStop?>(
                        context: context,
                        delegate: StopSearchDelegate(stops: kMockStops),
                      );
                    },
                    child: const Text('open'),
                  ),
                ),
              );
            },
          ),
        ),
      );

      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();

      // Suggestions should show all stops initially.
      expect(find.byType(ListTile), findsWidgets);

      // Type a query that matches "Wankdorf".
      await tester.enterText(find.byType(TextField), 'wank');
      await tester.pumpAndSettle();

      final tile = find.widgetWithText(ListTile, 'Wankdorf');
      expect(tile, findsOneWidget);

      // Tap the tile and expect it to close with the selected stop.
      await tester.tap(tile);
      await tester.pumpAndSettle();

      expect(selected, isNotNull);
      expect(selected!.name, 'Wankdorf');
    },
  );

  testWidgets(
    'StopSearchDelegate filters out non-matching entries',
    (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Builder(
            builder: (context) {
              return Scaffold(
                body: Center(
                  child: ElevatedButton(
                    onPressed: () {
                      showSearch<TransitStop?>(
                        context: context,
                        delegate: StopSearchDelegate(stops: kMockStops),
                      );
                    },
                    child: const Text('open'),
                  ),
                ),
              );
            },
          ),
        ),
      );

      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();

      await tester.enterText(find.byType(TextField), 'wankdorf');
      await tester.pumpAndSettle();

      expect(find.widgetWithText(ListTile, 'Wankdorf'), findsOneWidget);
      expect(find.widgetWithText(ListTile, 'Bern Bahnhof'), findsNothing);
    },
  );
}
