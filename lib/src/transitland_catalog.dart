import 'dart:async';
import 'dart:convert';
import 'dart:math' as math;

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

import 'app_log.dart';
import 'feed_catalog.dart';
import 'transitland_api_key.dart';

class TransitlandCatalog extends ChangeNotifier {
  TransitlandCatalog._();

  static final TransitlandCatalog instance = TransitlandCatalog._();

  static const String _cacheKey = 'transitland_runtime_feed_catalog_v1';
  static const String _cacheUpdatedKey =
      'transitland_runtime_feed_catalog_updated_at_v1';
  static const Duration _cacheFreshness = Duration(days: 7);

  Future<void>? _loadFuture;
  bool _loading = false;
  bool _loaded = false;
  String? _error;
  DateTime? _updatedAt;

  bool get isLoading => _loading;
  bool get hasLoaded => _loaded;
  String? get error => _error;
  DateTime? get updatedAt => _updatedAt;
  List<TransitFeed> get feeds => kTransitFeeds;

  Future<void> load({
    bool forceRefresh = false,
    TransitlandFeedClient? client,
  }) {
    if (_loadFuture case final future? when !forceRefresh) {
      return future;
    }
    _loadFuture = _load(
      forceRefresh: forceRefresh,
      client: client,
    ).whenComplete(() => _loadFuture = null);
    return _loadFuture!;
  }

  Future<void> _load({
    required bool forceRefresh,
    TransitlandFeedClient? client,
  }) async {
    final prefs = await SharedPreferences.getInstance();
    if (!forceRefresh) {
      _loadCached(prefs);
      if (_loaded && _updatedAt != null) {
        final age = DateTime.now().toUtc().difference(_updatedAt!);
        if (age < _cacheFreshness) {
          AppLogBuffer.instance.info(
            'Transitland catalog cache hit: ${kTransitFeeds.length} feeds, '
            'age ${age.inHours}h',
          );
          return;
        }
      }
    }

    _loading = true;
    _error = null;
    notifyListeners();
    AppLogBuffer.instance.info(
      forceRefresh
          ? 'Refreshing Transitland catalog from network'
          : 'Loading Transitland catalog from network',
    );

    try {
      final apiKey = await loadTransitlandApiKey();
      if (apiKey.isEmpty) {
        throw StateError(
          'TRANSITLAND_API_KEY is required to discover Transitland feeds.',
        );
      }
      final createdClient = client == null;
      final feedClient = client ?? TransitlandFeedClient();
      try {
        final fetched = await feedClient.fetchFeeds(apiKey: apiKey);
        replaceTransitFeedsForRuntime(fetched);
        _updatedAt = DateTime.now().toUtc();
        _loaded = true;
        await prefs.setString(_cacheKey, encodeTransitFeeds(fetched));
        await prefs.setString(_cacheUpdatedKey, _updatedAt!.toIso8601String());
        AppLogBuffer.instance.info(
          'Transitland catalog loaded: ${fetched.length} feeds',
        );
      } finally {
        if (createdClient) {
          feedClient.close();
        }
      }
    } catch (error, stack) {
      _error = error.toString();
      _loaded = kTransitFeeds.isNotEmpty;
      AppLogBuffer.instance.error(
        error,
        stackTrace: stack,
        context: 'Transitland catalog load failed',
      );
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  void _loadCached(SharedPreferences prefs) {
    final cached = prefs.getString(_cacheKey);
    if (cached == null || cached.isEmpty) return;
    final feeds = decodeTransitFeeds(
      cached,
    ).map(_repairGenericTransitlandFeed).toList(growable: false);
    if (feeds.isEmpty) return;
    replaceTransitFeedsForRuntime(feeds);
    _loaded = true;
    final updated = prefs.getString(_cacheUpdatedKey);
    _updatedAt = updated == null ? null : DateTime.tryParse(updated);
    AppLogBuffer.instance.info(
      'Restored cached Transitland catalog: ${feeds.length} feeds',
    );
    notifyListeners();
  }

  @visibleForTesting
  void replaceForTesting(Iterable<TransitFeed> feeds) {
    replaceTransitFeedsForRuntime(feeds);
    _loaded = true;
    _loading = false;
    _error = null;
    _updatedAt = DateTime.now().toUtc();
    notifyListeners();
  }
}

class TransitlandFeedClient {
  TransitlandFeedClient({
    http.Client? httpClient,
    this.baseUrl = kTransitlandRestBaseUrl,
    this.pageLimit,
  }) : _httpClient = httpClient ?? http.Client();

  final http.Client _httpClient;
  final String baseUrl;
  final int? pageLimit;

  Future<List<TransitFeed>> fetchFeeds({required String apiKey}) async {
    final out = <TransitFeed>[];
    final seen = <String>{};
    var after = 0;
    var pageCount = 0;

    while (true) {
      pageCount++;
      const maxRetries = 3;
      http.Response? response;
      for (var attempt = 0; attempt < maxRetries; attempt++) {
        try {
          response = await _httpClient.get(
            transitlandFeedsUri(baseUrl: baseUrl, after: after),
            headers: {'apikey': apiKey, 'User-Agent': 'transit-planner/0.1'},
          );
          break;
        } on http.ClientException {
          if (attempt == maxRetries - 1) rethrow;
          await Future.delayed(
            Duration(milliseconds: 500 * (attempt + 1)),
          );
        }
      }
      if (response == null) {
        throw StateError('Failed to fetch feeds after $maxRetries attempts');
      }
      if (response.statusCode < 200 || response.statusCode >= 300) {
        throw StateError(
          'Transitland feed discovery failed: HTTP ${response.statusCode}',
        );
      }
      final decoded = jsonDecode(response.body);
      if (decoded is! Map<String, dynamic>) {
        throw StateError('Transitland feed discovery returned invalid JSON.');
      }
      final feeds = decoded['feeds'];
      if (feeds is List) {
        for (final rawFeed in feeds) {
          if (rawFeed is! Map<String, dynamic>) continue;
          final feed = transitFeedFromTransitlandJson(
            rawFeed,
            baseUrl: baseUrl,
          );
          if (feed == null || !seen.add(feed.id)) continue;
          out.add(feed);
        }
      }
      final nextAfter = _nextAfter(decoded);
      if (nextAfter == 0 || nextAfter == after) break;
      if (pageLimit != null && pageCount >= pageLimit!) break;
      after = nextAfter;
    }

    return out;
  }

  void close() => _httpClient.close();
}

Uri transitlandFeedsUri({
  String baseUrl = kTransitlandRestBaseUrl,
  int after = 0,
}) {
  final uri = Uri.parse('${baseUrl.replaceFirst(RegExp(r'/$'), '')}/feeds');
  final query = <String, String>{
    'spec': 'gtfs',
    'fetch_error': 'false',
    'limit': '100',
    'license_redistribution_allowed': 'exclude_no',
    'license_create_derived_product': 'exclude_no',
    'license_commercial_use_allowed': 'exclude_no',
  };
  if (after > 0) {
    query['after'] = after.toString();
  }
  return uri.replace(queryParameters: query);
}

TransitFeed? transitFeedFromTransitlandJson(
  Map<String, dynamic> json, {
  String baseUrl = kTransitlandRestBaseUrl,
}) {
  final idValue = json['id'];
  final onestopId = (json['onestop_id'] as String?)?.trim() ?? '';
  final feedKey = onestopId.isNotEmpty ? onestopId : idValue?.toString() ?? '';
  if (feedKey.isEmpty) return null;

  final publisher = _publisher(json);
  final name = _displayName(json, feedKey, publisher);
  final effectivePublisher = _isGenericTransitlandLabel(publisher)
      ? name
      : publisher;
  final license = _license(json);
  final attribution = _attribution(json, effectivePublisher, license);
  final center = _center(json);
  final runtimeId = transitlandRuntimeFeedId(
    onestopId.isEmpty ? 'id-$feedKey' : feedKey,
  );

  return TransitFeed(
    id: runtimeId,
    name: name,
    description: '$name GTFS feed discovered from Transitland.',
    country: _countryCode(feedKey),
    region: 'Transitland',
    publisher: effectivePublisher,
    license: license,
    sourceUrl: transitlandDownloadUrl(feedKey, baseUrl: baseUrl),
    localFileName: '$runtimeId.zip',
    attribution: attribution,
    centerLatitude: center.latitude,
    centerLongitude: center.longitude,
  );
}

int _nextAfter(Map<String, dynamic> decoded) {
  final meta = decoded['meta'];
  if (meta is! Map<String, dynamic>) return 0;
  return (meta['after'] as num?)?.toInt() ?? 0;
}

String _publisher(Map<String, dynamic> json) {
  final operators = json['associated_operators'];
  if (operators is List) {
    for (final operator in operators) {
      if (operator is! Map<String, dynamic>) continue;
      final name = _stringValue(operator['name']);
      if (name.isNotEmpty) return name;
      final nestedOperator = operator['operator'];
      if (nestedOperator is Map<String, dynamic>) {
        final nestedName = _stringValue(nestedOperator['name']);
        if (nestedName.isNotEmpty) return nestedName;
      }
    }
  }
  final name = _stringValue(json['name']);
  if (_isGenericTransitlandLabel(name)) return '';
  return name;
}

String _displayName(
  Map<String, dynamic> json,
  String feedKey,
  String publisher,
) {
  final name = _stringValue(json['name']);
  if (!_isGenericTransitlandLabel(name)) return name;
  if (!_isGenericTransitlandLabel(publisher)) return publisher;
  return _fallbackFeedName(json, feedKey);
}

String _license(Map<String, dynamic> json) {
  final license = json['license'];
  if (license is Map<String, dynamic>) {
    final spdx = _stringValue(license['spdx_identifier']);
    if (spdx.isNotEmpty) return spdx;
    final url = _stringValue(license['url']);
    if (url.isNotEmpty) return url;
  }
  return 'Transitland license metadata';
}

String _attribution(
  Map<String, dynamic> json,
  String publisher,
  String license,
) {
  final licenseJson = json['license'];
  if (licenseJson is Map<String, dynamic>) {
    final text = _stringValue(licenseJson['attribution_text']);
    if (text.isNotEmpty) return text;
  }
  return 'Transit data from $publisher, $license; discovered through Transitland.';
}

String _countryCode(String feedKey) {
  final matches = RegExp(
    r'(?:^|[-~])([a-z]{2})(?:$|[-~])',
  ).allMatches(feedKey.toLowerCase()).toList(growable: false);
  if (matches.isEmpty) return 'Unknown';
  return matches.last.group(1)!.toUpperCase();
}

({double latitude, double longitude}) _center(Map<String, dynamic> json) {
  final bounds = _geometryBounds(json['feed_state']);
  if (bounds == null) {
    return (latitude: 0, longitude: 0);
  }
  return (
    latitude: (bounds.minLat + bounds.maxLat) / 2,
    longitude: (bounds.minLon + bounds.maxLon) / 2,
  );
}

({double minLon, double minLat, double maxLon, double maxLat})? _geometryBounds(
  Object? feedState,
) {
  if (feedState is! Map<String, dynamic>) return null;
  final feedVersion = feedState['feed_version'];
  if (feedVersion is! Map<String, dynamic>) return null;
  final geometry = feedVersion['geometry'];
  if (geometry is! Map<String, dynamic>) return null;
  final coordinates = geometry['coordinates'];
  var minLon = 180.0;
  var minLat = 90.0;
  var maxLon = -180.0;
  var maxLat = -90.0;
  var anyPoint = false;

  void visit(Object? value) {
    if (value is! List) return;
    if (value.length >= 2 && value[0] is num && value[1] is num) {
      final lon = (value[0] as num).toDouble();
      final lat = (value[1] as num).toDouble();
      minLon = math.min(minLon, lon);
      minLat = math.min(minLat, lat);
      maxLon = math.max(maxLon, lon);
      maxLat = math.max(maxLat, lat);
      anyPoint = true;
      return;
    }
    for (final item in value) {
      visit(item);
    }
  }

  visit(coordinates);
  if (!anyPoint) return null;
  return (minLon: minLon, minLat: minLat, maxLon: maxLon, maxLat: maxLat);
}

String _stringValue(Object? value) => value is String ? value.trim() : '';

bool _isGenericTransitlandLabel(String value) {
  final normalized = value.trim().toLowerCase();
  return normalized.isEmpty || normalized == 'transitland';
}

String _fallbackFeedName(Map<String, dynamic> json, String feedKey) {
  final fromKey = _nameFromFeedKey(feedKey);
  if (!_isGenericTransitlandLabel(fromKey)) return fromKey;

  final urls = json['urls'];
  if (urls is Map<String, dynamic>) {
    for (final key in const ['static_current', 'gbfs_auto_discovery']) {
      final fromUrl = _nameFromUrl(_stringValue(urls[key]));
      if (!_isGenericTransitlandLabel(fromUrl)) return fromUrl;
    }
  }

  return 'Transitland feed $feedKey';
}

String _nameFromFeedKey(String feedKey) {
  final key = feedKey
      .toLowerCase()
      .replaceFirst(RegExp(r'^transitland-'), '')
      .replaceFirst(RegExp(r'^id-'), '')
      .replaceFirst(RegExp(r'^f[-~]'), '');
  var rawParts = key
      .split(RegExp(r'[-~_]+'))
      .where((part) => part.isNotEmpty)
      .toList(growable: false);
  final country = _countryCode(feedKey).toLowerCase();
  if (rawParts.length > 1 && rawParts.last == country) {
    rawParts = rawParts.take(rawParts.length - 1).toList(growable: false);
  }
  final parts = rawParts
      .where((part) => !_looksLikeGeohash(part))
      .toList(growable: false);
  final usefulParts = parts.isEmpty ? rawParts : parts;
  if (usefulParts.isEmpty) return '';
  return usefulParts.map(_titleCaseIdentifier).join(' ');
}

bool _looksLikeGeohash(String value) {
  if (value.length > 8) return false;
  if (!RegExp(r'^[0-9bcdefghjkmnpqrstuvwxyz]+$').hasMatch(value)) {
    return false;
  }
  return value.length >= 5 || RegExp(r'\d').hasMatch(value);
}

String _titleCaseIdentifier(String value) {
  if (value.length <= 4 && RegExp(r'^[a-z0-9]+$').hasMatch(value)) {
    return value.toUpperCase();
  }
  return value
      .split(RegExp(r'[^a-z0-9]+'))
      .where((part) => part.isNotEmpty)
      .map((part) => '${part[0].toUpperCase()}${part.substring(1)}')
      .join(' ');
}

String _nameFromUrl(String value) {
  final uri = Uri.tryParse(value);
  final host = uri?.host ?? '';
  if (host.isEmpty) return '';
  final labels = host
      .replaceFirst(RegExp(r'^www\.'), '')
      .split('.')
      .where((label) => label.isNotEmpty)
      .toList(growable: false);
  if (labels.isEmpty) return '';
  final base = labels.length > 2 ? labels[labels.length - 3] : labels.first;
  return _titleCaseIdentifier(base.replaceAll('-', ' '));
}

TransitFeed _repairGenericTransitlandFeed(TransitFeed feed) {
  if (!_isGenericTransitlandLabel(feed.name)) return feed;

  final feedKey = _feedKeyFromTransitFeed(feed);
  final name = _nameFromFeedKey(feedKey);
  if (_isGenericTransitlandLabel(name)) return feed;

  return TransitFeed(
    id: feed.id,
    name: name,
    description: '$name GTFS feed discovered from Transitland.',
    country: feed.country,
    region: feed.region,
    publisher: _isGenericTransitlandLabel(feed.publisher)
        ? name
        : feed.publisher,
    license: feed.license,
    sourceUrl: feed.sourceUrl,
    localFileName: feed.localFileName,
    attribution: feed.attribution,
    centerLatitude: feed.centerLatitude,
    centerLongitude: feed.centerLongitude,
    bundledAssetPath: feed.bundledAssetPath,
    defaultDepartureHour: feed.defaultDepartureHour,
    componentFeedIds: feed.componentFeedIds,
  );
}

String _feedKeyFromTransitFeed(TransitFeed feed) {
  final sourceUri = Uri.tryParse(feed.sourceUrl);
  final segments = sourceUri?.pathSegments ?? const <String>[];
  final feedSegmentIndex = segments.indexOf('feeds');
  if (feedSegmentIndex >= 0 && feedSegmentIndex + 1 < segments.length) {
    // Path segments from Uri.pathSegments are already percent-decoded.
    return segments[feedSegmentIndex + 1];
  }
  return feed.id.replaceFirst(RegExp(r'^transitland-'), '');
}
