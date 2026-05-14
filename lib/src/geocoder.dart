import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

import 'app_log.dart';

/// A single forward-geocoding hit returned by [Geocoder.search].
class GeocodeResult {
  const GeocodeResult({
    required this.displayName,
    required this.latitude,
    required this.longitude,
    this.shortName,
    this.category,
  });

  /// Full address string from the geocoder (e.g. "Tokyo Tower, Minato, …").
  final String displayName;

  /// Short label suitable for compact UI (typically the first segment of
  /// [displayName]).
  final String? shortName;

  final double latitude;
  final double longitude;

  /// Loose category hint from the geocoder ("station", "bus_stop", …).
  final String? category;
}

abstract class Geocoder {
  Future<List<GeocodeResult>> search(String query, {GeocodeBias? bias});
}

/// A geographic hint passed to the geocoder so results near the active feed
/// rank higher than far-away matches with the same name.
class GeocodeBias {
  const GeocodeBias({
    required this.centerLat,
    required this.centerLon,
    this.viewboxRadiusDegrees = 0.6,
  });

  final double centerLat;
  final double centerLon;
  final double viewboxRadiusDegrees;

  String get viewbox {
    final left = centerLon - viewboxRadiusDegrees;
    final right = centerLon + viewboxRadiusDegrees;
    final top = centerLat + viewboxRadiusDegrees;
    final bottom = centerLat - viewboxRadiusDegrees;
    return '$left,$top,$right,$bottom';
  }
}

/// OpenStreetMap Nominatim-backed geocoder. No API key required; usage is
/// rate-limited by their fair-use policy so callers should debounce input
/// before invoking [search].
class NominatimGeocoder implements Geocoder {
  NominatimGeocoder({
    http.Client? client,
    this.endpoint = 'https://nominatim.openstreetmap.org/search',
    this.userAgent = 'transit-planner/1.0 (https://github.com/dvitali)',
  }) : _client = client ?? http.Client();

  final http.Client _client;
  final String endpoint;
  final String userAgent;

  @override
  Future<List<GeocodeResult>> search(String query, {GeocodeBias? bias}) async {
    final q = query.trim();
    if (q.isEmpty) return const [];
    final params = <String, String>{
      'q': q,
      'format': 'jsonv2',
      'addressdetails': '1',
      'limit': '8',
      'accept-language': 'en',
    };
    if (bias != null) {
      params['viewbox'] = bias.viewbox;
      params['bounded'] = '0';
    }
    final uri = Uri.parse(endpoint).replace(queryParameters: params);
    try {
      final response = await _client
          .get(uri, headers: {'User-Agent': userAgent})
          .timeout(const Duration(seconds: 8));
      if (response.statusCode < 200 || response.statusCode >= 300) {
        AppLogBuffer.instance.warning(
          'Geocoder HTTP ${response.statusCode} for query "$q"',
        );
        return const [];
      }
      final decoded = jsonDecode(response.body);
      if (decoded is! List) return const [];
      return decoded
          .whereType<Map<String, dynamic>>()
          .map(_resultFromJson)
          .whereType<GeocodeResult>()
          .toList(growable: false);
    } on TimeoutException {
      AppLogBuffer.instance.warning('Geocoder timed out for "$q"');
      return const [];
    } catch (error, stack) {
      AppLogBuffer.instance.error(
        error,
        stackTrace: stack,
        context: 'Geocoder request failed for "$q"',
      );
      return const [];
    }
  }

  GeocodeResult? _resultFromJson(Map<String, dynamic> json) {
    final lat = double.tryParse(json['lat']?.toString() ?? '');
    final lon = double.tryParse(json['lon']?.toString() ?? '');
    if (lat == null || lon == null) return null;
    final display = (json['display_name'] as String?) ?? '';
    final name = (json['name'] as String?)?.trim();
    final shortName = (name != null && name.isNotEmpty)
        ? name
        : display.split(',').first.trim();
    return GeocodeResult(
      displayName: display,
      shortName: shortName,
      latitude: lat,
      longitude: lon,
      category: json['category'] as String? ?? json['type'] as String?,
    );
  }
}
