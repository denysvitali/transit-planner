// Catalog of known GTFS feeds and merged regional networks for the app.
//
// The app uses one default feed and does not expose feed selection. Broad
// Transitland-discovered coverage must be prebuilt or loaded lazily rather than
// downloading every component feed at startup.
part 'feed_catalog.g.dart';

class TransitFeed {
  const TransitFeed({
    required this.id,
    required this.name,
    required this.description,
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
