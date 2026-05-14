import 'dart:async';
import 'dart:ui';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'src/app_log.dart';
import 'src/home_page.dart';
import 'src/itinerary_detail_page.dart';
import 'src/models.dart';
import 'src/settings_page.dart';
import 'src/theme.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  final previousFlutterErrorHandler = FlutterError.onError;
  FlutterError.onError = (details) {
    AppLogBuffer.instance.flutterError(details);
    if (previousFlutterErrorHandler != null) {
      previousFlutterErrorHandler(details);
    } else {
      FlutterError.presentError(details);
    }
  };
  PlatformDispatcher.instance.onError = (error, stackTrace) {
    AppLogBuffer.instance.error(error, stackTrace: stackTrace);
    return false;
  };

  runZonedGuarded(() => runApp(const TransitPlannerApp()), (error, stackTrace) {
    AppLogBuffer.instance.error(error, stackTrace: stackTrace);
  });
}

class TransitPlannerApp extends StatefulWidget {
  const TransitPlannerApp({super.key});

  @override
  State<TransitPlannerApp> createState() => _TransitPlannerAppState();
}

class _TransitPlannerAppState extends State<TransitPlannerApp> {
  late final GoRouter _router;

  @override
  void initState() {
    super.initState();
    _router = GoRouter(
      routes: [
        GoRoute(path: '/', builder: (context, state) => const HomePage()),
        GoRoute(
          path: '/settings',
          builder: (context, state) => const SettingsPage(),
        ),
        GoRoute(
          path: '/itinerary',
          builder: (context, state) =>
              ItineraryDetailPage(itinerary: state.extra! as Itinerary),
        ),
      ],
    );
  }

  @override
  void dispose() {
    _router.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'Transit Planner',
      debugShowCheckedModeBanner: false,
      theme: buildTransitTheme(Brightness.light),
      darkTheme: buildTransitTheme(Brightness.dark),
      routerConfig: _router,
    );
  }
}
