import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
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
}
