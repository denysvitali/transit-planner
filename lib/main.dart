import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'src/home_page.dart';
import 'src/theme.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const TransitPlannerApp());
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
        GoRoute(
          path: '/',
          builder: (context, state) => const HomePage(),
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
