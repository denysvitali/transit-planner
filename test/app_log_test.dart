import 'package:flutter_test/flutter_test.dart';
import 'package:transit_planner/src/app_log.dart';

void main() {
  tearDown(AppLogBuffer.instance.clear);

  test('records entries at every level', () {
    AppLogBuffer.instance.debug('Probe ping');
    AppLogBuffer.instance.info('Feed opened');
    AppLogBuffer.instance.warning('No modes selected');
    AppLogBuffer.instance.error(
      StateError('Router unavailable'),
      context: 'Route planning failed',
    );

    final all = AppLogBuffer.instance.formatted(AppLogLevel.values.toSet());
    expect(all, contains('DEBUG Probe ping'));
    expect(all, contains('INFO Feed opened'));
    expect(all, contains('WARNING No modes selected'));
    expect(
      all,
      contains('ERROR Route planning failed: Bad state: Router unavailable'),
    );
  });

  test('filters entries by selected levels', () {
    AppLogBuffer.instance.info('Feed opened');
    AppLogBuffer.instance.warning('No modes selected');

    final onlyWarnings = AppLogBuffer.instance.formatted({
      AppLogLevel.warning,
    });
    expect(onlyWarnings, contains('WARNING No modes selected'));
    expect(onlyWarnings, isNot(contains('Feed opened')));
  });

  test('returns an empty log message when no entries match', () {
    final logs = AppLogBuffer.instance.formatted(AppLogLevel.values.toSet());
    expect(logs, 'No logs recorded.');
  });
}
