import 'package:flutter/material.dart';

class AppSpacing {
  static const double xs = 8;
  static const double s = 12;
  static const double m = 16;
  static const double l = 24;
  static const double xl = 32;
}

class AppRadius {
  static const double s = 8;
  static const double m = 14;
  static const double l = 22;
}

class AppPalette {
  static const Color green = Color(0xFF0F9F6E);
  static const Color blue = Color(0xFF2563EB);
  static const Color amber = Color(0xFFF59E0B);
  static const Color rose = Color(0xFFE11D48);
  static const Color ink = Color(0xFF111827);
  static const Color surfaceDark = Color(0xFF151A20);
}

ThemeData buildTransitTheme(Brightness brightness) {
  final dark = brightness == Brightness.dark;
  final scheme = ColorScheme.fromSeed(
    seedColor: AppPalette.green,
    brightness: brightness,
    primary: AppPalette.green,
    secondary: AppPalette.blue,
    tertiary: AppPalette.amber,
    error: AppPalette.rose,
  );
  final textTheme = ThemeData(
    useMaterial3: true,
    brightness: brightness,
  ).textTheme.apply(
        bodyColor: dark ? Colors.white : AppPalette.ink,
        displayColor: dark ? Colors.white : AppPalette.ink,
      );

  return ThemeData(
    useMaterial3: true,
    brightness: brightness,
    colorScheme: scheme,
    textTheme: textTheme,
    scaffoldBackgroundColor:
        dark ? const Color(0xFF0E1116) : const Color(0xFFF7F9FB),
    appBarTheme: AppBarTheme(
      elevation: 0,
      scrolledUnderElevation: 0,
      backgroundColor: dark ? const Color(0xFF0E1116) : Colors.white,
      foregroundColor: scheme.onSurface,
      titleTextStyle: textTheme.titleLarge?.copyWith(
        fontWeight: FontWeight.w800,
      ),
    ),
    cardTheme: CardThemeData(
      elevation: 0,
      margin: EdgeInsets.zero,
      color: dark ? AppPalette.surfaceDark : Colors.white,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(AppRadius.s),
        side: BorderSide(color: scheme.outlineVariant),
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(AppRadius.s),
      ),
      filled: true,
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        minimumSize: const Size(0, 48),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppRadius.s),
        ),
      ),
    ),
    segmentedButtonTheme: SegmentedButtonThemeData(
      style: ButtonStyle(
        shape: WidgetStatePropertyAll(
          RoundedRectangleBorder(borderRadius: BorderRadius.circular(AppRadius.s)),
        ),
      ),
    ),
  );
}
