import 'dart:io';

const String _dartDefineTransitlandApiKey = String.fromEnvironment(
  'TRANSITLAND_API_KEY',
);

Future<String> loadTransitlandApiKey() async {
  if (_dartDefineTransitlandApiKey.trim().isNotEmpty) {
    return _dartDefineTransitlandApiKey.trim();
  }

  final environmentValue = Platform.environment['TRANSITLAND_API_KEY'];
  if (environmentValue != null && environmentValue.trim().isNotEmpty) {
    return environmentValue.trim();
  }

  final envFile = File('.env');
  if (!await envFile.exists()) {
    return '';
  }
  final lines = await envFile.readAsLines();
  for (final line in lines) {
    final trimmed = line.trim();
    if (trimmed.isEmpty || trimmed.startsWith('#')) continue;
    final separator = trimmed.indexOf('=');
    if (separator <= 0) continue;
    final key = trimmed.substring(0, separator).trim();
    if (key != 'TRANSITLAND_API_KEY') continue;
    return _stripEnvQuotes(trimmed.substring(separator + 1).trim());
  }
  return '';
}

String _stripEnvQuotes(String value) {
  if (value.length < 2) return value;
  final first = value[0];
  final last = value[value.length - 1];
  if ((first == '"' && last == '"') || (first == "'" && last == "'")) {
    return value.substring(1, value.length - 1);
  }
  return value;
}
