import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:transit_planner/src/app_shell.dart';
import 'package:transit_planner/src/settings_page.dart';

void main() {
  testWidgets('settings shows feeds as read-only attribution', (tester) async {
    await tester.pumpWidget(const MaterialApp(home: SettingsPage()));

    expect(find.text('Transitland coverage'), findsOneWidget);
    expect(find.text('Rome public transport GTFS'), findsOneWidget);
    expect(
      find.textContaining('does not download this whole coverage list'),
      findsOneWidget,
    );
    expect(find.byIcon(Icons.radio_button_checked), findsNothing);
    expect(find.byIcon(Icons.radio_button_off), findsNothing);
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
