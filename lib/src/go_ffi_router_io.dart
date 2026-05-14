// Native implementation of the LocalTransitRouter that drives the Go
// router via dart:ffi. The C ABI is defined by cmd/libtransitplanner in
// the Go module and lives in libtransit_planner.{so|dylib|dll}.
//
// Public surface (matches go_ffi_router_stub.dart for the conditional
// import barrel in go_ffi_router.dart):
//   - bool   goFfiSupported  — true when the current platform can load the
//                              native library.
//   - Future<LocalTransitRouter> openToeiRouter() — stages the bundled
//     Toei GTFS zip on disk, opens it through the FFI, and returns a
//     router whose `route` / `stops` calls go through the Go engine.
import 'dart:convert';
import 'dart:ffi';
import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:ffi/ffi.dart';
import 'package:flutter/services.dart' show rootBundle;
import 'package:path_provider/path_provider.dart';

import 'app_log.dart';
import 'local_router.dart';
import 'models.dart';

const String _kFeedAssetPath =
    'assets/sample_toei_train/Toei-Train-GTFS.zip';
const String _kFeedBasename = 'Toei-Train-GTFS.zip';

bool get goFfiSupported =>
    Platform.isAndroid ||
    Platform.isIOS ||
    Platform.isLinux ||
    Platform.isMacOS ||
    Platform.isWindows;

/// Stages the bundled Toei GTFS zip into the app support directory and
/// opens it through the Go FFI surface. Returns a long-lived router whose
/// underlying feed handle stays alive for the lifetime of the process.
///
/// Falls back to [MockTransitRouter] if anything along the way fails — the
/// UI should always remain usable even when the FFI is unavailable.
Future<LocalTransitRouter> openToeiRouter() async {
  if (!goFfiSupported) {
    return const MockTransitRouter();
  }
  try {
    final zipPath = await _stageBundledFeed();
    final native = _NativeBindings.instance;
    final handle = native.open(zipPath);
    final stops = native.stops(handle);
    return _GoFfiRouter._(native: native, handle: handle, stops: stops);
  } catch (error, stack) {
    AppLogBuffer.instance.error(
      error,
      stackTrace: stack,
      context: 'Failed to open Toei GTFS via FFI; falling back to mock',
    );
    return const MockTransitRouter();
  }
}

/// Copies the bundled Toei zip into the platform-appropriate writable
/// directory and returns its absolute path. The bundled bytes' sha256 is
/// recorded in a sidecar file so we only rewrite when the asset changes
/// (e.g. when the developer re-runs `tool/fetch_gtfs`).
Future<String> _stageBundledFeed() async {
  final supportDir = await getApplicationSupportDirectory();
  final feedDir = Directory('${supportDir.path}/gtfs');
  if (!feedDir.existsSync()) {
    feedDir.createSync(recursive: true);
  }
  final dst = File('${feedDir.path}/$_kFeedBasename');
  final stamp = File('${dst.path}.sha256');

  final assetData = await rootBundle.load(_kFeedAssetPath);
  final bytes = assetData.buffer
      .asUint8List(assetData.offsetInBytes, assetData.lengthInBytes);
  final expected = sha256.convert(bytes).toString();

  final fresh = dst.existsSync() &&
      stamp.existsSync() &&
      stamp.readAsStringSync().trim() == expected;
  if (!fresh) {
    dst.writeAsBytesSync(bytes, flush: true);
    stamp.writeAsStringSync(expected);
  }
  return dst.path;
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

  int open(String feedZipPath) {
    final resp = _call(open_, {'feedZip': feedZipPath});
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
  _GoFfiRouter._({
    required _NativeBindings native,
    required int handle,
    required List<TransitStop> stops,
  })  : _native = native,
        _handle = handle,
        _stops = stops,
        _stopsById = {for (final s in stops) s.id: s};

  final _NativeBindings _native;
  final int _handle;
  final List<TransitStop> _stops;
  final Map<String, TransitStop> _stopsById;

  @override
  Future<List<TransitStop>> stops() async => _stops;

  @override
  Future<List<Itinerary>> route(RouteRequest request) async {
    final secondsFromMidnight = request.departure.hour * 3600 +
        request.departure.minute * 60 +
        request.departure.second;
    final Map<String, dynamic> response;
    try {
      response = _native.route({
        'handle': _handle,
        'from': request.origin.id,
        'to': request.destination.id,
        'departure': secondsFromMidnight,
        'maxTransfers': request.maxTransfers,
      });
    } on FfiRouterException catch (error) {
      if (error.isDestinationUnreachable) {
        return const [];
      }
      rethrow;
    }
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
    return [
      Itinerary(legs: legs, transfers: transfers, walking: walking),
    ];
  }

  ItineraryLeg _legFromJson(
    Map<String, dynamic> json,
    DateTime sameDayAnchor,
  ) {
    final mode = _modeFor(json['mode'] as String?);
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
      routeName: json['routeId'] as String?,
      tripId: json['tripId'] as String?,
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

  TransitMode _modeFor(String? raw) {
    switch (raw) {
      case 'walk':
        return TransitMode.walk;
      case 'transit':
        // Today the Toei feed we ship is the subway file. When we add buses
        // or trams the cffi response should include route_type so we can
        // pick the right Material icon; until then default to subway.
        return TransitMode.subway;
      default:
        return TransitMode.subway;
    }
  }
}
