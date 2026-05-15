import 'package:flutter/foundation.dart';

enum AppLogLevel { debug, info, warning, error }

class AppLogEntry {
  const AppLogEntry({
    required this.level,
    required this.message,
    required this.timestamp,
    this.stackTrace,
  });

  final AppLogLevel level;
  final String message;
  final DateTime timestamp;
  final StackTrace? stackTrace;

  String get formatted {
    final buffer = StringBuffer()
      ..write(timestamp.toIso8601String())
      ..write(' ')
      ..write(level.name.toUpperCase())
      ..write(' ')
      ..write(message.trim());
    if (stackTrace != null) {
      buffer
        ..writeln()
        ..write(stackTrace);
    }
    return buffer.toString();
  }
}

class AppLogBuffer extends ChangeNotifier {
  AppLogBuffer._();

  static final AppLogBuffer instance = AppLogBuffer._();

  static const _maxEntries = 500;

  final List<AppLogEntry> _entries = [];

  List<AppLogEntry> get entries => List.unmodifiable(_entries);

  List<AppLogEntry> entriesFor(Set<AppLogLevel> levels) {
    return _entries.where((entry) => levels.contains(entry.level)).toList();
  }

  void debug(String message) {
    _add(
      AppLogEntry(
        level: AppLogLevel.debug,
        message: message,
        timestamp: DateTime.now(),
      ),
    );
  }

  void info(String message) {
    _add(
      AppLogEntry(
        level: AppLogLevel.info,
        message: message,
        timestamp: DateTime.now(),
      ),
    );
  }

  void warning(String message) {
    _add(
      AppLogEntry(
        level: AppLogLevel.warning,
        message: message,
        timestamp: DateTime.now(),
      ),
    );
  }

  void error(Object error, {StackTrace? stackTrace, String? context}) {
    final message = context == null ? '$error' : '$context: $error';
    _add(
      AppLogEntry(
        level: AppLogLevel.error,
        message: message,
        stackTrace: stackTrace,
        timestamp: DateTime.now(),
      ),
    );
  }

  void flutterError(FlutterErrorDetails details) {
    final context = details.context?.toDescription();
    error(details.exception, stackTrace: details.stack, context: context);
  }

  String formatted(Set<AppLogLevel> levels) {
    final selected = entriesFor(levels);
    if (selected.isEmpty) {
      return 'No logs recorded.';
    }
    return selected.map((entry) => entry.formatted).join('\n\n');
  }

  void clear() {
    _entries.clear();
    notifyListeners();
  }

  void _add(AppLogEntry entry) {
    _entries.add(entry);
    if (_entries.length > _maxEntries) {
      _entries.removeRange(0, _entries.length - _maxEntries);
    }
    if (kDebugMode) {
      debugPrint(entry.formatted);
    }
    notifyListeners();
  }
}
