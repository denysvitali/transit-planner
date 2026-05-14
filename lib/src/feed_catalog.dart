// Catalog of known GTFS feeds for the app.
//
// The list is intentionally small and curated so we can keep onboarding
// straightforward while adding more Japanese operators over time.

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

  bool get isBundled => bundledAssetPath != null;
}

const String kDefaultFeedId = 'toei-train';

const List<TransitFeed> kTransitFeeds = [
  TransitFeed(
    id: 'toei-train',
    name: 'Tokyo Toei Subway',
    description:
        'Toei subway lines (浅草線, 三田線, 新宿線, '
        '大江戸線, 日暮里舎人ライナー, 都電荒川線).',
    publisher: 'Tokyo Metropolitan Bureau of Transportation (東京都交通局)',
    license: 'CC-BY-4.0',
    sourceUrl:
        'https://api-public.odpt.org/api/v4/files/Toei/data/Toei-Train-GTFS.zip',
    localFileName: 'Toei-Train-GTFS.zip',
    attribution:
        'Transit data © 東京都交通局 (Tokyo Metropolitan Bureau of '
        'Transportation), CC-BY 4.0, via the Public Transportation Open '
        'Data Center (ODPT).',
    bundledAssetPath: 'assets/sample_toei_train/Toei-Train-GTFS.zip',
    centerLatitude: 35.681236,
    centerLongitude: 139.767125,
    defaultDepartureHour: 8,
  ),
  TransitFeed(
    id: 'toei-bus',
    name: 'Tokyo Toei Bus',
    description:
        'Tokyo Metropolitan Bureau of Transportation municipal bus routes.',
    publisher: 'Tokyo Metropolitan Bureau of Transportation (東京都交通局)',
    license: 'CC-BY-4.0',
    sourceUrl:
        'https://api-public.odpt.org/api/v4/files/Toei/data/ToeiBus-GTFS.zip',
    localFileName: 'ToeiBus-GTFS.zip',
    attribution:
        'Transit data © 東京都交通局 (Tokyo Metropolitan Bureau of '
        'Transportation), CC-BY 4.0, via the Public Transportation Open '
        'Data Center (ODPT).',
    centerLatitude: 35.681236,
    centerLongitude: 139.767125,
  ),
  TransitFeed(
    id: 'kanazawa-flatbus',
    name: 'Kanazawa Flat Bus',
    description:
        'Kanazawa city bus network, published as GTFS on the city open-data '
        'catalog.',
    publisher: 'Kanazawa City, Ishikawa',
    license: 'CC-BY-4.0',
    sourceUrl:
        'https://catalog-data.city.kanazawa.ishikawa.jp/dataset/'
        '1196beb4-f9f9-463c-9723-5b38d8127425/resource/'
        '9636cac5-1449-4656-893b-ec98d834eb23/download/flatbus20260401.zip',
    localFileName: 'flatbus20260401.zip',
    attribution:
        'Transit data © Kanazawa City (Kanazawa-ken Jichitai), CC-BY 4.0.',
    bundledAssetPath:
        'assets/real_gtfs/jp/kanazawa_flatbus/kanazawa-flatbus.zip',
    centerLatitude: 36.5608,
    centerLongitude: 136.6566,
    // Kanazawa Flat Bus runs roughly 08:00–18:00 Asia/Tokyo. Anchor planning
    // to mid-morning so the initial route always returns trips, regardless
    // of the user's local clock.
    defaultDepartureHour: 9,
  ),
  TransitFeed(
    id: 'kanazawa-hakusan-meguru',
    name: 'Hakusan Meguru',
    description:
        'Hakusan City Community Bus ("Meguru") network GTFS from the '
        'municipal open-data portal.',
    publisher: 'Hakusan City, Ishikawa',
    license: 'CC-BY-4.0',
    sourceUrl:
        'https://catalog-data.city.kanazawa.ishikawa.jp/dataset/'
        '89d93f28-38b4-4971-9988-2ff2d3227f56/resource/'
        '50049b19-fe9f-4ca1-9ea9-9d0a24141644/download/172103_bus.zip',
    localFileName: '172103_bus.zip',
    attribution: 'Transit data © Hakusan City (白山市), CC-BY 4.0.',
    centerLatitude: 36.2581,
    centerLongitude: 136.6290,
  ),
  TransitFeed(
    id: 'kanazawa-tsubata-bus',
    name: 'Tsubata Town Bus',
    description: 'Tsubata Town bus routes on the GSF/GTFS-JP package.',
    publisher: 'Tsubata Town (Tsubata-chō), Ishikawa',
    license: 'CC-BY-4.0',
    sourceUrl:
        'https://catalog-data.city.kanazawa.ishikawa.jp/dataset/'
        '8cd7f0dc-aab0-4bf4-a09d-c1d79faf4512/resource/'
        '9565f9b7-3bf7-4937-bee5-789d2aa4bf8a/download/gtfs-jp_tsubata.zip',
    localFileName: 'gtfs-jp_tsubata.zip',
    attribution:
        'Transit data © Tsubata Town / Kanazawa public transport data, CC-BY 4.0.',
    centerLatitude: 36.7381,
    centerLongitude: 136.5596,
  ),
];

TransitFeed? findFeedById(String id) {
  for (final feed in kTransitFeeds) {
    if (feed.id == id) {
      return feed;
    }
  }
  return null;
}
