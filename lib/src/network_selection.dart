import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'app_log.dart';
import 'feed_catalog.dart';
import 'transitland_catalog.dart';

class NetworkSelection extends ChangeNotifier {
  NetworkSelection._() : _selectedFeedIds = {};

  static final NetworkSelection instance = NetworkSelection._();
  static const String _prefsKey = 'selected_transitland_feed_ids';
  static const String _legacyPrefsKey = 'selected_network_feed_id';

  Set<String> _selectedFeedIds;
  Future<void>? _loadFuture;
  bool _loaded = false;

  Set<String> get selectedFeedIds => Set.unmodifiable(_selectedFeedIds);
  bool get hasSelectedFeeds => _selectedFeeds.isNotEmpty;
  bool get hasLoaded => _loaded;

  TransitFeed get feed {
    final feeds = _selectedFeeds;
    if (feeds.isEmpty) {
      return const TransitFeed(
        id: 'no-transitland-feeds-selected',
        name: 'No Transitland feeds selected',
        description: 'Select one or more Transitland feeds in Settings.',
        country: 'Global',
        region: 'Transitland',
        publisher: 'Transitland',
        license: 'Transitland license metadata',
        sourceUrl: kTransitlandRestBaseUrl,
        localFileName: '',
        attribution: 'No Transitland feed is currently selected.',
        centerLatitude: 0,
        centerLongitude: 0,
      );
    }
    if (feeds.length == 1) {
      return feeds.single;
    }
    int? hour;
    for (final feed in feeds) {
      hour ??= feed.defaultDepartureHour;
    }
    final lat =
        feeds.fold<double>(0, (sum, feed) => sum + feed.centerLatitude) /
        feeds.length;
    final lon =
        feeds.fold<double>(0, (sum, feed) => sum + feed.centerLongitude) /
        feeds.length;
    return TransitFeed(
      id: 'selected-transitland-feeds',
      name: 'Selected Transitland feeds',
      description: 'Merged local network from selected Transitland feeds.',
      country: 'Global',
      region: 'Selected',
      publisher: 'Transitland and source transit-data publishers',
      license: 'Mixed source licences',
      sourceUrl: 'https://transit.land/api/v2/rest/feeds',
      localFileName: '',
      attribution:
          'Transit data selected from Transitland-listed feeds; licences vary '
          'by publisher.',
      centerLatitude: lat,
      centerLongitude: lon,
      defaultDepartureHour: hour,
      componentFeedIds: feeds.map((feed) => feed.id).toList(growable: false),
    );
  }

  List<TransitFeed> get _selectedFeeds {
    final feeds = _selectedFeedIds
        .map(findFeedById)
        .whereType<TransitFeed>()
        .where((feed) => !feed.isCollection)
        .toList(growable: false);
    return feeds;
  }

  Future<void> load() {
    return _loadFuture ??= _load().whenComplete(() => _loadFuture = null);
  }

  Future<void> _load() async {
    if (_loaded) return;
    await TransitlandCatalog.instance.load();
    final prefs = await SharedPreferences.getInstance();
    final feedIds = prefs.getStringList(_prefsKey);
    if (feedIds != null && feedIds.isNotEmpty) {
      await _setSelectedFeedIds(feedIds, persist: false);
      _loaded = true;
      return;
    }

    final legacyFeedId = prefs.getString(_legacyPrefsKey);
    final legacyFeed = legacyFeedId == null ? null : findFeedById(legacyFeedId);
    if (legacyFeed != null) {
      await _setSelectedFeedIds(
        componentFeedsFor(legacyFeed).map((feed) => feed.id),
        persist: false,
      );
    }
    _loaded = true;
  }

  Future<void> select(TransitFeed feed) async {
    await setSelectedFeedIds(componentFeedsFor(feed).map((feed) => feed.id));
  }

  Future<void> setFeedSelected(String feedId, bool selected) async {
    final ids = {..._selectedFeedIds};
    if (selected) {
      ids.add(feedId);
    } else {
      ids.remove(feedId);
    }
    await setSelectedFeedIds(ids);
  }

  Future<void> setFeedsSelected(Iterable<String> feedIds, bool selected) async {
    final ids = {..._selectedFeedIds};
    if (selected) {
      ids.addAll(feedIds);
    } else {
      ids.removeAll(feedIds);
    }
    await setSelectedFeedIds(ids);
  }

  Future<void> setSelectedFeedIds(Iterable<String> feedIds) async {
    await _setSelectedFeedIds(feedIds, persist: true);
  }

  Future<void> _setSelectedFeedIds(
    Iterable<String> feedIds, {
    required bool persist,
  }) async {
    final validIds = feedIds
        .map(findFeedById)
        .whereType<TransitFeed>()
        .where((feed) => !feed.isCollection)
        .map((feed) => feed.id)
        .toSet();
    if (setEquals(validIds, _selectedFeedIds)) return;
    _selectedFeedIds = validIds;
    AppLogBuffer.instance.info(
      validIds.isEmpty
          ? 'Feed selection cleared'
          : 'Feed selection: ${validIds.length} feed'
                '${validIds.length == 1 ? '' : 's'} '
                '(${(validIds.toList()..sort()).join(', ')})',
    );
    notifyListeners();
    if (persist) {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setStringList(_prefsKey, validIds.toList()..sort());
    }
  }
}
