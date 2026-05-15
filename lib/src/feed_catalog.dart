import 'dart:convert';

const String kTransitlandRestBaseUrl = 'https://transit.land/api/v2/rest';

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

  Map<String, dynamic> toJson() => {
    'id': id,
    'name': name,
    'description': description,
    'country': country,
    'region': region,
    'publisher': publisher,
    'license': license,
    'sourceUrl': sourceUrl,
    'localFileName': localFileName,
    'attribution': attribution,
    'centerLatitude': centerLatitude,
    'centerLongitude': centerLongitude,
    if (bundledAssetPath != null) 'bundledAssetPath': bundledAssetPath,
    if (defaultDepartureHour != null)
      'defaultDepartureHour': defaultDepartureHour,
    if (componentFeedIds.isNotEmpty) 'componentFeedIds': componentFeedIds,
  };

  factory TransitFeed.fromJson(Map<String, dynamic> json) => TransitFeed(
    id: json['id'] as String? ?? '',
    name: json['name'] as String? ?? '',
    description: json['description'] as String? ?? '',
    country: json['country'] as String? ?? '',
    region: json['region'] as String? ?? '',
    publisher: json['publisher'] as String? ?? '',
    license: json['license'] as String? ?? '',
    sourceUrl: json['sourceUrl'] as String? ?? '',
    localFileName: json['localFileName'] as String? ?? '',
    attribution: json['attribution'] as String? ?? '',
    centerLatitude: (json['centerLatitude'] as num?)?.toDouble() ?? 0,
    centerLongitude: (json['centerLongitude'] as num?)?.toDouble() ?? 0,
    bundledAssetPath: json['bundledAssetPath'] as String?,
    defaultDepartureHour: json['defaultDepartureHour'] as int?,
    componentFeedIds:
        (json['componentFeedIds'] as List<dynamic>?)
            ?.whereType<String>()
            .toList(growable: false) ??
        const [],
  );
}

List<TransitFeed> _transitFeeds = const [];

List<TransitFeed> get kTransitFeeds => List.unmodifiable(_transitFeeds);

void replaceTransitFeedsForRuntime(Iterable<TransitFeed> feeds) {
  final sorted = feeds.toList(growable: false)
    ..sort((a, b) {
      final country = a.country.compareTo(b.country);
      if (country != 0) return country;
      final region = a.region.compareTo(b.region);
      if (region != 0) return region;
      return a.name.compareTo(b.name);
    });
  _transitFeeds = sorted;
}

String encodeTransitFeeds(Iterable<TransitFeed> feeds) =>
    jsonEncode(feeds.map((feed) => feed.toJson()).toList(growable: false));

List<TransitFeed> decodeTransitFeeds(String encoded) {
  final decoded = jsonDecode(encoded);
  if (decoded is! List) return const [];
  return decoded
      .whereType<Map<String, dynamic>>()
      .map(TransitFeed.fromJson)
      .where((feed) => feed.id.isNotEmpty)
      .toList(growable: false);
}

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

List<TransitFeed> selectableTransitFeeds() =>
    kTransitFeeds.where((feed) => !feed.isCollection).toList(growable: false);

String transitlandDownloadUrl(
  String feedKey, {
  String baseUrl = kTransitlandRestBaseUrl,
}) {
  final base = Uri.parse(baseUrl);
  final path =
      '${base.path.replaceFirst(RegExp(r'/$'), '')}/feeds/'
      '${Uri.encodeComponent(feedKey)}/download_latest_feed_version';
  final authority = base.hasPort ? '${base.host}:${base.port}' : base.host;
  return Uri.parse('${base.scheme}://$authority$path').toString();
}

String transitlandRuntimeFeedId(String feedKey) =>
    'transitland-${sanitizeTransitlandFeedKey(feedKey)}';

String sanitizeTransitlandFeedKey(String value) {
  final buffer = StringBuffer();
  for (final rune in value.toLowerCase().runes) {
    final char = String.fromCharCode(rune);
    final isAsciiLetter = rune >= 97 && rune <= 122;
    final isDigit = rune >= 48 && rune <= 57;
    if (isAsciiLetter || isDigit) {
      buffer.write(char);
    } else if (buffer.isNotEmpty && !buffer.toString().endsWith('-')) {
      buffer.write('-');
    }
  }
  final out = buffer.toString().replaceAll(RegExp(r'-+$'), '');
  return out.isEmpty ? 'feed' : out;
}
