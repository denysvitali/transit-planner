import 'package:flutter_test/flutter_test.dart';
import 'package:transit_planner/src/app_log.dart';

void main() {
  tearDown(AppLogBuffer.instance.clear);

  test('formats warning and error entries for copying', () {
    AppLogBuffer.instance.warning('No modes selected');
    AppLogBuffer.instance.error(
      StateError('Router unavailable'),
      context: 'Route planning failed',
    );

    final logs = AppLogBuffer.instance.formatted({
      AppLogLevel.warning,
      AppLogLevel.error,
    });

    expect(logs, contains('WARNING No modes selected'));
    expect(
      logs,
      contains('ERROR Route planning failed: Bad state: Router unavailable'),
    );
  });

  test('returns an empty log message when no entries match', () {
    final logs = AppLogBuffer.instance.formatted({
      AppLogLevel.warning,
      AppLogLevel.error,
    });

    expect(logs, 'No warning or error logs recorded.');
  });
}
