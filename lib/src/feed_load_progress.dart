import 'feed_catalog.dart';

enum FeedLoadOperation {
  preparing,
  checkingCache,
  copyingBundledFeed,
  downloadingFeed,
  openingRouter,
  loadingStops,
}

class FeedLoadProgress {
  const FeedLoadProgress({
    required this.feed,
    required this.operation,
    required this.componentIndex,
    required this.componentCount,
    this.bytesReceived,
    this.totalBytes,
  });

  final TransitFeed feed;
  final FeedLoadOperation operation;
  final int componentIndex;
  final int componentCount;
  final int? bytesReceived;
  final int? totalBytes;

  double? get fraction {
    final received = bytesReceived;
    final total = totalBytes;
    if (received == null || total == null || total <= 0) {
      return null;
    }
    return (received / total).clamp(0, 1).toDouble();
  }
}
