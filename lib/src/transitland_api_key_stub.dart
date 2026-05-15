const String _dartDefineTransitlandApiKey = String.fromEnvironment(
  'TRANSITLAND_API_KEY',
);

Future<String> loadTransitlandApiKey() async => _dartDefineTransitlandApiKey;
