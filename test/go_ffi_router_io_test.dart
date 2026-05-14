import 'package:flutter_test/flutter_test.dart';
import 'package:transit_planner/src/go_ffi_router_io.dart';

void main() {
  test('destination unreachable FFI errors are identifiable as no-route', () {
    expect(
      FfiRouterException('destination unreachable').isDestinationUnreachable,
      isTrue,
    );
    expect(
      FfiRouterException('origin stop not found').isDestinationUnreachable,
      isFalse,
    );
  });
}
