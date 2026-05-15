// Catalog of known GTFS feeds and merged regional networks for the app.
//
// The app starts with one bundled default feed, then lets users opt into
// broader Transitland-discovered networks or individual GTFS feeds.
part 'feed_catalog.g.dart';

class TransitFeed {
  const TransitFeed({
    required this.id,
    required this.name,
    required this.description,
    this.country = '',
    this.region = '',
    required this.publisher,
    required this.license,
    required this.sourceUrl,
    required this.localFileName,
    required this.attribution,
    required this.centerLatitude,
    required this.centerLongitude,
    this.bundledAssetPath,
    this.defaultDepartureHour,
    this.componentFeedIds = const [],
  });

  final String id;
  final String name;
  final String description;
  final String country;
  final String region;
  final String publisher;
  final String license;
  final String sourceUrl;
  final String localFileName;
  final String attribution;
  final String? bundledAssetPath;
  final int? defaultDepartureHour;
  final double centerLatitude;
  final double centerLongitude;
  final List<String> componentFeedIds;

  bool get isBundled => bundledAssetPath != null;
  bool get isCollection => componentFeedIds.isNotEmpty;
}

const String kDefaultFeedId = 'toei-train';

const List<String> kAppNetworkFeedIds = [
  'transitland-coverage',
  'jp-public-no-key',
  'ch-national',
  'it-public-regional',
  'tokyo-toei',
  kDefaultFeedId,
];

TransitFeed? findFeedById(String id) {
  for (final feed in kTransitFeeds) {
    if (feed.id == id) {
      return feed;
    }
  }
  return null;
}

List<TransitFeed> componentFeedsFor(TransitFeed feed) {
  if (!feed.isCollection) {
    return [feed];
  }
  return feed.componentFeedIds
      .map(findFeedById)
      .whereType<TransitFeed>()
      .toList(growable: false);
}

List<TransitFeed> appNetworkFeeds() => kAppNetworkFeedIds
    .map(findFeedById)
    .whereType<TransitFeed>()
    .toList(growable: false);

List<TransitFeed> selectableTransitFeeds() {
  final seen = <String>{};
  final out = <TransitFeed>[];

  void addFeed(TransitFeed feed) {
    if (seen.add(feed.id)) {
      out.add(feed);
    }
  }

  for (final feed in appNetworkFeeds()) {
    addFeed(feed);
  }
  for (final feed in kTransitFeeds) {
    if (!feed.isCollection) {
      addFeed(feed);
    }
  }
  return out;
}
