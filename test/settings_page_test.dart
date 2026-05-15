import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:transit_planner/src/app_shell.dart';
import 'package:transit_planner/src/feed_catalog.dart';
import 'package:transit_planner/src/network_selection.dart';
import 'package:transit_planner/src/settings_page.dart';

void main() {
  setUp(() async {
    SharedPreferences.setMockInitialValues({});
    await NetworkSelection.instance.select(findFeedById(kDefaultFeedId)!);
  });

  testWidgets('settings selects network and shows attribution', (tester) async {
    await tester.pumpWidget(const MaterialApp(home: SettingsPage()));

    expect(
      find.widgetWithText(ListTile, 'Transitland coverage'),
      findsOneWidget,
    );

    await tester.tap(find.widgetWithText(ListTile, 'Transitland coverage'));
    await tester.pump();

    expect(NetworkSelection.instance.feed.id, 'transitland-coverage');

    await tester.scrollUntilVisible(
      find.widgetWithText(ListTile, 'Rome public transport GTFS'),
      500,
      maxScrolls: 50,
    );
    await tester.tap(
      find.widgetWithText(ListTile, 'Rome public transport GTFS'),
    );
    await tester.pump();

    expect(NetworkSelection.instance.feed.id, 'it-rome');
  });

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
    expect(find.text('Copy warnings and errors'), findsOneWidget);

    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();

    expect(find.widgetWithText(AppBar, 'Settings'), findsOneWidget);
    expect(find.widgetWithText(ListTile, 'Logs'), findsOneWidget);
  });
}
