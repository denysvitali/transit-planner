import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'feed_catalog.dart';

class NetworkSelection extends ChangeNotifier {
  NetworkSelection._()
    : _feed = findFeedById(kDefaultFeedId) ?? kTransitFeeds.first;

  static final NetworkSelection instance = NetworkSelection._();
  static const String _prefsKey = 'selected_network_feed_id';

  TransitFeed _feed;

  TransitFeed get feed => _feed;

  Future<void> load() async {
    final prefs = await SharedPreferences.getInstance();
    final feedId = prefs.getString(_prefsKey);
    final feed = feedId == null ? null : findFeedById(feedId);
    if (feed != null && feed.id != _feed.id) {
      _feed = feed;
      notifyListeners();
    }
  }

  Future<void> select(TransitFeed feed) async {
    if (feed.id == _feed.id) return;
    _feed = feed;
    notifyListeners();
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_prefsKey, feed.id);
  }
}
