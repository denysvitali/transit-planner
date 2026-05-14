// Web fallback for the Go-FFI router. The web build cannot load native
// libraries, so we fall back to the in-Dart mock implementation. The user
// sees the existing Bern mock stops; the About screen notes the real
// feed only works on the mobile/desktop targets.
import 'local_router.dart';

/// True when the current platform supports the Go FFI router. Always false
/// on web; see go_ffi_router_io.dart for the real implementation.
const bool goFfiSupported = false;

/// Returns a [LocalTransitRouter]. On web this is always a
/// [MockTransitRouter].
Future<LocalTransitRouter> openToeiRouter() async {
  return const MockTransitRouter();
}
