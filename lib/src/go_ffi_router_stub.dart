// Web fallback for the Go-FFI router. The web build cannot load native
// libraries, so web keeps the in-Dart mock implementation.
import 'feed_load_progress.dart';
import 'local_router.dart';
import 'feed_catalog.dart';

/// True when the current platform supports the Go FFI router. Always false
/// on web; see go_ffi_router_io.dart for the real implementation.
const bool goFfiSupported = false;

/// Returns a [LocalTransitRouter]. On web this is always a
/// [MockTransitRouter].
Future<LocalTransitRouter> openToeiRouter({
  void Function(FeedLoadProgress progress)? onProgress,
}) async {
  return const MockTransitRouter();
}

Future<LocalTransitRouter> openFeedRouter(
  TransitFeed feed, {
  void Function(FeedLoadProgress progress)? onProgress,
}) async => const MockTransitRouter();
