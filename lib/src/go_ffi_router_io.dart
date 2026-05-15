// Native implementation of the LocalTransitRouter that drives the Go
// router via dart:ffi. The C ABI is defined by cmd/libtransitplanner in
// the Go module and lives in libtransit_planner.{so|dylib|dll}.
//
// Public surface (matches go_ffi_router_stub.dart for the conditional
// import barrel in go_ffi_router.dart):
//   - bool   goFfiSupported  — true when the current platform can load the
//                              native library.
//   - Future<LocalTransitRouter> openToeiRouter() — legacy helper that opens
//     the current Transitland feed selection and returns a router whose
//     `route` / `stops` calls go through the Go engine.
import 'dart:convert';
import 'dart:ffi';
import 'dart:io';
import 'dart:isolate';

import 'package:crypto/crypto.dart';
import 'package:ffi/ffi.dart';
import 'package:flutter/services.dart' show rootBundle;
import 'package:path_provider/path_provider.dart';

import 'app_log.dart';
import 'feed_load_progress.dart';
import 'local_router.dart';
import 'models.dart';
import 'feed_catalog.dart';
import 'network_selection.dart';
import 'transitland_api_key.dart';

const String _kFeedCacheDir = 'gtfs';

Future<LocalTransitRouter> openToeiRouter({
  void Function(FeedLoadProgress progress)? onProgress,
}) async =>
    openFeedRouter(NetworkSelection.instance.feed, onProgress: onProgress);

Future<LocalTransitRouter> openFeedRouter(
  TransitFeed feed, {
  void Function(FeedLoadProgress progress)? onProgress,
}) async {
  if (!goFfiSupported) {
    throw UnsupportedError(
      'Go FFI router is not supported on ${Platform.operatingSystem}',
    );
  }
  int? handle;
  try {
    onProgress?.call(
      FeedLoadProgress(
        feed: feed,
        operation: FeedLoadOperation.preparing,
        componentIndex: 1,
        componentCount: 1,
      ),
    );
    final stagedFeeds = await _stageFeeds(feed, onProgress: onProgress);
    onProgress?.call(
      FeedLoadProgress(
        feed: feed,
        operation: FeedLoadOperation.openingRouter,
        componentIndex: stagedFeeds.length,
        componentCount: stagedFeeds.length,
      ),
    );
    final openedHandle = await Isolate.run(() => _openNativeFeed(stagedFeeds));
    handle = openedHandle;
    onProgress?.call(
      FeedLoadProgress(
        feed: feed,
        operation: FeedLoadOperation.loadingStops,
        componentIndex: stagedFeeds.length,
        componentCount: stagedFeeds.length,
      ),
    );
    final stops = await Isolate.run(() => _loadNativeStops(openedHandle));
    return _GoFfiRouter._(handle: openedHandle, stops: stops);
  } catch (error, stack) {
    final failedHandle = handle;
    if (failedHandle != null) {
      await Isolate.run(() => _closeNativeFeed(failedHandle));
    }
    AppLogBuffer.instance.error(
      error,
      stackTrace: stack,
      context: 'Failed to open ${feed.id} via FFI',
    );
    rethrow;
  }
}

bool get goFfiSupported =>
    Platform.isAndroid ||
    Platform.isIOS ||
    Platform.isLinux ||
    Platform.isMacOS ||
    Platform.isWindows;

Future<List<_StagedFeed>> _stageFeeds(
  TransitFeed feed, {
  void Function(FeedLoadProgress progress)? onProgress,
}) async {
  final components = componentFeedsFor(feed);
  if (components.isEmpty) {
    throw StateError('collection feed ${feed.id} has no known components');
  }
  if (components.length == 1 && !feed.isCollection) {
    final path = await _stageFeed(
      components.single,
      componentIndex: 1,
      componentCount: 1,
      onProgress: onProgress,
    );
    return [_StagedFeed(path: path)];
  }

  final staged = <_StagedFeed>[];
  for (var i = 0; i < components.length; i++) {
    final component = components[i];
    final path = await _stageFeed(
      component,
      componentIndex: i + 1,
      componentCount: components.length,
      onProgress: onProgress,
    );
    staged.add(_StagedFeed(prefix: component.id, path: path));
  }
  return staged;
}

/// Stages a single feed into app cache.
Future<String> _stageFeed(
  TransitFeed feed, {
  required int componentIndex,
  required int componentCount,
  void Function(FeedLoadProgress progress)? onProgress,
}) async {
  if (feed.isCollection) {
    throw StateError('collection feed ${feed.id} must be expanded first');
  }
  void report(
    FeedLoadOperation operation, {
    int? bytesReceived,
    int? totalBytes,
  }) {
    onProgress?.call(
      FeedLoadProgress(
        feed: feed,
        operation: operation,
        componentIndex: componentIndex,
        componentCount: componentCount,
        bytesReceived: bytesReceived,
        totalBytes: totalBytes,
      ),
    );
  }

  report(FeedLoadOperation.checkingCache);
  final supportDir = await getApplicationSupportDirectory();
  final feedDir = Directory('${supportDir.path}/$_kFeedCacheDir/${feed.id}');
  if (!await feedDir.exists()) {
    await feedDir.create(recursive: true);
  }
  final dst = File('${feedDir.path}/${feed.localFileName}');
  final stamp = File('${dst.path}.sha256');
  final isBundled = feed.isBundled;

  final expectedStamp = isBundled
      ? await _bundledFeedStamp(feed)
      : _cacheStamp(feed);
  final existingStamp = await stamp.exists()
      ? (await stamp.readAsString()).trim()
      : '';
  final hasFreshCache = await dst.exists() && existingStamp == expectedStamp;
  if (hasFreshCache) {
    return dst.path;
  }

  if (isBundled) {
    await _cacheBundledFeed(feed, dst, stamp, onProgress: report);
  } else {
    await _downloadFeed(feed, dst, stamp, onProgress: report);
  }
  return dst.path;
}

Future<void> _cacheBundledFeed(
  TransitFeed feed,
  File dst,
  File stamp, {
  required void Function(
    FeedLoadOperation operation, {
    int? bytesReceived,
    int? totalBytes,
  })
  onProgress,
}) async {
  final assetPath = feed.bundledAssetPath;
  if (assetPath == null) {
    throw StateError('missing bundled asset path for ${feed.id}');
  }
  final assetData = await rootBundle.load(assetPath);
  final bytes = assetData.buffer.asUint8List(
    assetData.offsetInBytes,
    assetData.lengthInBytes,
  );
  final expected = await Isolate.run(() => sha256.convert(bytes).toString());
  onProgress(
    FeedLoadOperation.copyingBundledFeed,
    bytesReceived: bytes.length,
    totalBytes: bytes.length,
  );
  await dst.writeAsBytes(bytes, flush: true);
  await stamp.writeAsString(_cacheStampWithHash(feed, expected));
}

Future<String> _bundledFeedStamp(TransitFeed feed) async {
  final assetPath = feed.bundledAssetPath;
  if (assetPath == null) {
    throw StateError('missing bundled asset path for ${feed.id}');
  }
  final assetData = await rootBundle.load(assetPath);
  final bytes = assetData.buffer.asUint8List(
    assetData.offsetInBytes,
    assetData.lengthInBytes,
  );
  final expected = await Isolate.run(() => sha256.convert(bytes).toString());
  return _cacheStampWithHash(feed, expected);
}

Future<void> _downloadFeed(
  TransitFeed feed,
  File dst,
  File stamp, {
  required void Function(
    FeedLoadOperation operation, {
    int? bytesReceived,
    int? totalBytes,
  })
  onProgress,
}) async {
  final client = HttpClient();
  try {
    final uri = Uri.parse(feed.sourceUrl);
    final request = await client.getUrl(uri);
    if (_isTransitlandDownload(uri)) {
      final transitlandApiKey = await loadTransitlandApiKey();
      if (transitlandApiKey.isEmpty) {
        throw StateError(
          'TRANSITLAND_API_KEY must be provided to download Transitland feeds.',
        );
      }
      request.headers.set('apikey', transitlandApiKey);
    }
    final result = await request.close();
    if (result.statusCode < 200 || result.statusCode >= 300) {
      throw HttpException(
        'failed to download ${feed.id}: HTTP ${result.statusCode}',
        uri: uri,
      );
    }
    final tmp = File('${dst.path}.tmp');
    final sink = tmp.openWrite();
    final totalBytes = result.contentLength >= 0 ? result.contentLength : null;
    var received = 0;
    onProgress(
      FeedLoadOperation.downloadingFeed,
      bytesReceived: received,
      totalBytes: totalBytes,
    );
    try {
      await for (final chunk in result) {
        received += chunk.length;
        sink.add(chunk);
        onProgress(
          FeedLoadOperation.downloadingFeed,
          bytesReceived: received,
          totalBytes: totalBytes,
        );
      }
      await sink.flush();
      await sink.close();
    } catch (_) {
      await sink.close();
      if (await tmp.exists()) {
        await tmp.delete();
      }
      rethrow;
    }
    await tmp.rename(dst.path);
    await stamp.writeAsString(_cacheStamp(feed));
  } finally {
    client.close();
  }
}

bool _isTransitlandDownload(Uri uri) =>
    uri.scheme == 'https' &&
    uri.host == 'transit.land' &&
    uri.path.startsWith('/api/v2/rest/feeds/');

String _cacheStampWithHash(TransitFeed feed, String hash) =>
    '${feed.sourceUrl}\n$hash';

String _cacheStamp(TransitFeed feed) => feed.sourceUrl;

class _StagedFeed {
  const _StagedFeed({required this.path, this.prefix});

  final String path;
  final String? prefix;
}

int _openNativeFeed(List<_StagedFeed> stagedFeeds) =>
    _NativeBindings.instance.open(stagedFeeds);

List<TransitStop> _loadNativeStops(int handle) =>
    _NativeBindings.instance.stops(handle);

void _closeNativeFeed(int handle) => _NativeBindings.instance.close(handle);

Map<String, dynamic> _routeNative(Map<String, dynamic> request) {
  try {
    return {'response': _NativeBindings.instance.route(request)};
  } on FfiRouterException catch (error) {
    return {'error': error.message};
  }
}

/// Thin wrapper that owns the symbols looked up from the Go-built shared
/// library. The C ABI is JSON-in / JSON-out, with the response pointer
/// owned by Go and freed via [TP_Free].
class _NativeBindings {
  _NativeBindings._({
    required this.open_,
    required this.close_,
    required this.stops_,
    required this.route_,
    required this.free_,
  });

  static final _NativeBindings instance = _resolve();

  static _NativeBindings _resolve() {
    final lib = _openLibrary();
    return _NativeBindings._(
      open_: lib.lookupFunction<_NativeStringFn, _DartStringFn>('TP_Open'),
      close_: lib.lookupFunction<_NativeStringFn, _DartStringFn>('TP_Close'),
      stops_: lib.lookupFunction<_NativeStringFn, _DartStringFn>('TP_Stops'),
      route_: lib.lookupFunction<_NativeStringFn, _DartStringFn>('TP_Route'),
      free_: lib.lookupFunction<_NativeFreeFn, _DartFreeFn>('TP_Free'),
    );
  }

  final Pointer<Utf8> Function(Pointer<Utf8>) open_;
  final Pointer<Utf8> Function(Pointer<Utf8>) close_;
  final Pointer<Utf8> Function(Pointer<Utf8>) stops_;
  final Pointer<Utf8> Function(Pointer<Utf8>) route_;
  final void Function(Pointer<Utf8>) free_;

  Map<String, dynamic> _call(
    Pointer<Utf8> Function(Pointer<Utf8>) fn,
    Map<String, dynamic> request,
  ) {
    final reqPtr = jsonEncode(request).toNativeUtf8();
    Pointer<Utf8>? respPtr;
    try {
      respPtr = fn(reqPtr);
      final raw = respPtr.toDartString();
      final decoded = jsonDecode(raw) as Map<String, dynamic>;
      if (decoded['error'] is String &&
          (decoded['error'] as String).isNotEmpty) {
        throw FfiRouterException(decoded['error'] as String);
      }
      return decoded;
    } finally {
      if (respPtr != null) {
        free_(respPtr);
      }
      malloc.free(reqPtr);
    }
  }

  int open(List<_StagedFeed> feeds) {
    if (feeds.length == 1 && feeds.single.prefix == null) {
      final resp = _call(open_, {'feedZip': feeds.single.path});
      return (resp['handle'] as num).toInt();
    }
    final resp = _call(open_, {
      'feeds': [
        for (final feed in feeds) {'prefix': feed.prefix, 'feedZip': feed.path},
      ],
    });
    return (resp['handle'] as num).toInt();
  }

  void close(int handle) {
    // Best-effort: errors here are unrecoverable from the caller's
    // perspective so we just log and continue.
    try {
      _call(close_, {'handle': handle});
    } catch (error) {
      AppLogBuffer.instance.warning(
        'TP_Close failed for handle $handle: $error',
      );
    }
  }

  List<TransitStop> stops(int handle) {
    final resp = _call(stops_, {'handle': handle});
    final raw = (resp['stops'] as List<dynamic>?) ?? const [];
    return raw
        .cast<Map<String, dynamic>>()
        .map(_stopFromJson)
        .toList(growable: false);
  }

  Map<String, dynamic> route(Map<String, dynamic> request) =>
      _call(route_, request);
}

typedef _NativeStringFn = Pointer<Utf8> Function(Pointer<Utf8>);
typedef _DartStringFn = Pointer<Utf8> Function(Pointer<Utf8>);
typedef _NativeFreeFn = Void Function(Pointer<Utf8>);
typedef _DartFreeFn = void Function(Pointer<Utf8>);

DynamicLibrary _openLibrary() {
  if (Platform.isAndroid || Platform.isLinux) {
    return DynamicLibrary.open('libtransit_planner.so');
  }
  if (Platform.isMacOS) {
    return DynamicLibrary.open('libtransit_planner.dylib');
  }
  if (Platform.isWindows) {
    return DynamicLibrary.open('transit_planner.dll');
  }
  if (Platform.isIOS) {
    // iOS bundles the symbols into the Runner executable via the static
    // archive; see cmd/libtransitplanner README for the Xcode wiring.
    return DynamicLibrary.process();
  }
  throw UnsupportedError(
    'Go FFI router is not supported on ${Platform.operatingSystem}',
  );
}

TransitStop _stopFromJson(Map<String, dynamic> json) => TransitStop(
  id: json['id'] as String,
  name: json['name'] as String,
  latitude: (json['lat'] as num).toDouble(),
  longitude: (json['lon'] as num).toDouble(),
);

/// Thrown when the Go side returns a non-empty `error` field.
class FfiRouterException implements Exception {
  FfiRouterException(this.message);
  final String message;

  bool get isDestinationUnreachable => message == 'destination unreachable';

  @override
  String toString() => 'FfiRouterException: $message';
}

class _GoFfiRouter implements LocalTransitRouter {
  _GoFfiRouter._({required int handle, required List<TransitStop> stops})
    : _handle = handle,
      _stops = stops,
      _stopsById = {for (final s in stops) s.id: s};

  final int _handle;
  final List<TransitStop> _stops;
  final Map<String, TransitStop> _stopsById;

  @override
  Future<List<TransitStop>> stops() async => _stops;

  @override
  Future<List<Itinerary>> route(RouteRequest request) async {
    final secondsFromMidnight =
        request.departure.hour * 3600 +
        request.departure.minute * 60 +
        request.departure.second;
    final nativeRequest = {
      'handle': _handle,
      'from': request.origin.id,
      'to': request.destination.id,
      ..._endpointPayload('from', request.originPoint),
      ..._endpointPayload('to', request.destinationPoint),
      'departure': secondsFromMidnight,
      'maxTransfers': request.maxTransfers,
      'routeTypes': _routeTypesForModes(request.modes),
    };
    final routeResponse = await Isolate.run(() => _routeNative(nativeRequest));
    final error = routeResponse['error'];
    if (error is String && error.isNotEmpty) {
      final exception = FfiRouterException(error);
      if (exception.isDestinationUnreachable) {
        return const [];
      }
      throw exception;
    }
    final response = routeResponse['response'] as Map<String, dynamic>;
    final legsJson = (response['legs'] as List<dynamic>?) ?? const [];
    if (legsJson.isEmpty) {
      return const [];
    }
    final legs = <ItineraryLeg>[];
    var walking = Duration.zero;
    for (final raw in legsJson.cast<Map<String, dynamic>>()) {
      final leg = _legFromJson(raw, request.departure);
      legs.add(leg);
      if (leg.mode == TransitMode.walk) {
        walking += leg.duration;
      }
    }
    final transfers = (response['transfers'] as num?)?.toInt() ?? 0;
    return [Itinerary(legs: legs, transfers: transfers, walking: walking)];
  }

  ItineraryLeg _legFromJson(Map<String, dynamic> json, DateTime sameDayAnchor) {
    final mode = _modeFor(
      json['mode'] as String?,
      (json['routeType'] as num?)?.toInt(),
    );
    final fromStop = _stopFromLegPayload(json['fromStop']);
    final toStop = _stopFromLegPayload(json['toStop']);
    final dep = _toDateTime(json['departure'] as num, sameDayAnchor);
    final arr = _toDateTime(json['arrival'] as num, sameDayAnchor);
    return ItineraryLeg(
      mode: mode,
      from: fromStop,
      to: toStop,
      departure: dep,
      arrival: arr,
      routeName: (json['routeName'] as String?) ?? (json['routeId'] as String?),
      tripId: json['tripId'] as String?,
      routeType: (json['routeType'] as num?)?.toInt(),
    );
  }

  TransitStop _stopFromLegPayload(dynamic payload) {
    if (payload is! Map) {
      throw FfiRouterException('malformed leg stop payload: $payload');
    }
    // Go uses Capital-cased JSON keys for router.Stop (ID/Name/Lat/Lon).
    final id = payload['ID'] as String;
    return _stopsById[id] ??
        TransitStop(
          id: id,
          name: payload['Name'] as String? ?? id,
          latitude: (payload['Lat'] as num? ?? 0).toDouble(),
          longitude: (payload['Lon'] as num? ?? 0).toDouble(),
        );
  }

  DateTime _toDateTime(num secondsFromMidnight, DateTime anchor) {
    final base = DateTime(anchor.year, anchor.month, anchor.day);
    return base.add(Duration(seconds: secondsFromMidnight.toInt()));
  }

  TransitMode _modeFor(String? raw, int? routeType) {
    switch (raw) {
      case 'walk':
        return TransitMode.walk;
      case 'transit':
        return _transitModeForRouteType(routeType);
      default:
        return TransitMode.subway;
    }
  }

  TransitMode _transitModeForRouteType(int? routeType) {
    switch (routeType) {
      case 0:
        return TransitMode.tram;
      case 1:
        return TransitMode.subway;
      case 2:
        return TransitMode.rail;
      case 3:
        return TransitMode.bus;
      case 4:
        return TransitMode.ferry;
      default:
        return TransitMode.rail;
    }
  }

  List<int> _routeTypesForModes(Set<TransitMode> modes) {
    final routeTypes = <int>{};
    for (final mode in modes) {
      switch (mode) {
        case TransitMode.walk:
          break;
        case TransitMode.tram:
          routeTypes.add(0);
        case TransitMode.subway:
          routeTypes.add(1);
        case TransitMode.rail:
          routeTypes.add(2);
        case TransitMode.bus:
          routeTypes.add(3);
        case TransitMode.ferry:
          routeTypes.add(4);
      }
    }
    return routeTypes.toList(growable: false);
  }

  Map<String, Object> _endpointPayload(String prefix, RoutePoint? point) {
    if (point == null || point.isStop) return const {};
    return {
      '${prefix}Name': point.name,
      '${prefix}Lat': point.latitude,
      '${prefix}Lon': point.longitude,
    };
  }
}
