import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:transit_planner/src/app_shell.dart';
import 'package:transit_planner/src/feed_catalog.dart';
import 'package:transit_planner/src/network_selection.dart';
import 'package:transit_planner/src/settings_page.dart';
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

  testWidgets(
    'settings selects runtime Transitland feeds by country and feed',
    (tester) async {
      await tester.pumpWidget(const MaterialApp(home: SettingsPage()));

      expect(find.text('Transitland feeds'), findsOneWidget);
      expect(find.widgetWithText(CheckboxListTile, 'AA'), findsOneWidget);

      await tester.tap(find.widgetWithText(CheckboxListTile, 'AA'));
      await tester.pump();

      expect(NetworkSelection.instance.selectedFeedIds, contains('feed-one'));
      expect(NetworkSelection.instance.selectedFeedIds, contains('feed-two'));

      await tester.scrollUntilVisible(
        find.widgetWithText(CheckboxListTile, 'Two City Transit'),
        500,
        maxScrolls: 50,
      );
      await tester.tap(
        find.widgetWithText(CheckboxListTile, 'Two City Transit'),
      );
      await tester.pump();

      expect(NetworkSelection.instance.selectedFeedIds, contains('feed-one'));
      expect(
        NetworkSelection.instance.selectedFeedIds,
        isNot(contains('feed-two')),
      );
    },
  );

  testWidgets('logs open as a settings sub-view and back returns to settings', (
    tester,
  ) async {
    final router = GoRouter(
      initialLocation: '/settings',
      routes: [
        StatefulShellRoute.indexedStack(
          builder: (context, state, navigationShell) =>
              AppShell(navigationShell: navigationShell),
          branches: [
            StatefulShellBranch(
              routes: [
                GoRoute(
                  path: '/',
                  builder: (context, state) =>
                      const Scaffold(body: Center(child: Text('Route'))),
                ),
              ],
            ),
            StatefulShellBranch(
              routes: [
                GoRoute(
                  path: '/settings',
                  builder: (context, state) => const SettingsPage(),
                  routes: [
                    GoRoute(
                      path: 'logs',
                      builder: (context, state) => const LogsPage(),
                    ),
                  ],
                ),
              ],
            ),
          ],
        ),
      ],
    );
    addTearDown(router.dispose);

    await tester.pumpWidget(MaterialApp.router(routerConfig: router));
    await tester.pumpAndSettle();

    expect(find.widgetWithText(AppBar, 'Settings'), findsOneWidget);

    await tester.scrollUntilVisible(find.widgetWithText(ListTile, 'Logs'), 500);
    await tester.tap(find.widgetWithText(ListTile, 'Logs'));
    await tester.pumpAndSettle();

    expect(find.widgetWithText(AppBar, 'Logs'), findsOneWidget);
    expect(find.text('Copy filtered logs'), findsOneWidget);

    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();

    expect(find.widgetWithText(AppBar, 'Settings'), findsOneWidget);
    expect(find.widgetWithText(ListTile, 'Logs'), findsOneWidget);
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
