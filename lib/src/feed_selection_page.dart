import 'package:flutter/material.dart';

import 'feed_catalog.dart';
import 'network_selection.dart';
import 'theme.dart';
import 'transitland_catalog.dart';

class FeedSelectionPage extends StatefulWidget {
  const FeedSelectionPage({super.key});

  @override
  State<FeedSelectionPage> createState() => _FeedSelectionPageState();
}

class _FeedSelectionPageState extends State<FeedSelectionPage> {
  final Set<String> _expandedCountries = {};
  final TextEditingController _searchController = TextEditingController();
  String _query = '';

  @override
  void initState() {
    super.initState();
    final catalog = TransitlandCatalog.instance;
    if (!catalog.hasLoaded && !catalog.isLoading) {
      catalog.load();
    }
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  void _toggleCountry(String country) {
    setState(() {
      if (_expandedCountries.contains(country)) {
        _expandedCountries.remove(country);
      } else {
        _expandedCountries.add(country);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Select feeds'),
        actions: [
          ListenableBuilder(
            listenable: TransitlandCatalog.instance,
            builder: (context, _) {
              final catalog = TransitlandCatalog.instance;
              return IconButton(
                icon: const Icon(Icons.sync),
                onPressed: catalog.isLoading
                    ? null
                    : () =>
                          TransitlandCatalog.instance.load(forceRefresh: true),
              );
            },
          ),
          ListenableBuilder(
            listenable: NetworkSelection.instance,
            builder: (context, _) {
              final hasSelection =
                  NetworkSelection.instance.selectedFeedIds.isNotEmpty;
              return IconButton(
                tooltip: 'Clear selection',
                icon: const Icon(Icons.clear_all),
                onPressed: hasSelection
                    ? () =>
                          NetworkSelection.instance.setSelectedFeedIds(const [])
                    : null,
              );
            },
          ),
        ],
      ),
      body: SafeArea(
        child: CustomScrollView(
          slivers: [
            SliverPadding(
              padding: const EdgeInsets.symmetric(
                horizontal: AppSpacing.m,
                vertical: AppSpacing.s,
              ),
              sliver: ListenableBuilder(
                listenable: TransitlandCatalog.instance,
                builder: (context, _) {
                  final catalog = TransitlandCatalog.instance;
                  final groupedFeeds = _groupedSelectableFeeds(_query);

                  if (catalog.isLoading && groupedFeeds.isEmpty) {
                    return const SliverToBoxAdapter(
                      child: Padding(
                        padding: EdgeInsets.symmetric(vertical: AppSpacing.m),
                        child: LinearProgressIndicator(),
                      ),
                    );
                  }

                  if (groupedFeeds.isEmpty) {
                    return SliverToBoxAdapter(
                      child: Padding(
                        padding: const EdgeInsets.symmetric(
                          vertical: AppSpacing.m,
                        ),
                        child: Text(
                          'No Transitland feeds are loaded yet.',
                          style: Theme.of(context).textTheme.bodyMedium,
                        ),
                      ),
                    );
                  }

                  return SliverToBoxAdapter(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        Text(
                          _catalogStatus(catalog),
                          style: Theme.of(context).textTheme.bodySmall
                              ?.copyWith(
                                color: Theme.of(
                                  context,
                                ).colorScheme.onSurfaceVariant,
                              ),
                        ),
                        if (catalog.error != null) ...[
                          const SizedBox(height: AppSpacing.xs),
                          Text(
                            catalog.error!,
                            style: Theme.of(context).textTheme.bodySmall
                                ?.copyWith(
                                  color: Theme.of(context).colorScheme.error,
                                ),
                          ),
                        ],
                        const SizedBox(height: AppSpacing.s),
                        TextField(
                          controller: _searchController,
                          decoration: InputDecoration(
                            prefixIcon: const Icon(Icons.search),
                            suffixIcon: _query.isEmpty
                                ? null
                                : IconButton(
                                    tooltip: 'Clear search',
                                    icon: const Icon(Icons.close),
                                    onPressed: () {
                                      _searchController.clear();
                                      setState(() => _query = '');
                                    },
                                  ),
                            hintText: 'Search feeds',
                            border: const OutlineInputBorder(),
                          ),
                          textInputAction: TextInputAction.search,
                          onChanged: (value) {
                            setState(() => _query = value.trim());
                          },
                        ),
                        const SizedBox(height: AppSpacing.s),
                      ],
                    ),
                  );
                },
              ),
            ),
            SliverPadding(
              padding: const EdgeInsets.symmetric(horizontal: AppSpacing.m),
              sliver: ListenableBuilder(
                listenable: TransitlandCatalog.instance,
                builder: (context, _) {
                  final groupedFeeds = _groupedSelectableFeeds(_query);
                  if (groupedFeeds.isEmpty) {
                    return const SliverToBoxAdapter(child: SizedBox.shrink());
                  }
                  return SliverList(
                    delegate: SliverChildBuilderDelegate((context, index) {
                      final country = groupedFeeds.keys.elementAt(index);
                      final regions = groupedFeeds[country]!;
                      final isExpanded = _expandedCountries.contains(country);
                      return _CountrySection(
                        country: country,
                        regions: regions,
                        isExpanded: isExpanded,
                        onToggleExpanded: () => _toggleCountry(country),
                      );
                    }, childCount: groupedFeeds.length),
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _CountrySection extends StatelessWidget {
  const _CountrySection({
    required this.country,
    required this.regions,
    required this.isExpanded,
    required this.onToggleExpanded,
  });

  final String country;
  final Map<String, List<TransitFeed>> regions;
  final bool isExpanded;
  final VoidCallback onToggleExpanded;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        ListenableBuilder(
          listenable: NetworkSelection.instance,
          builder: (context, _) {
            final feedIds = _feedIdsForCountry(regions);
            final value = _selectionValue(
              feedIds,
              NetworkSelection.instance.selectedFeedIds,
            );
            return ListTile(
              contentPadding: EdgeInsets.zero,
              leading: Checkbox(
                tristate: true,
                value: value,
                onChanged: (value) => NetworkSelection.instance
                    .setFeedsSelected(feedIds, value ?? true),
              ),
              title: Text(
                _countryLabel(country),
                style: Theme.of(
                  context,
                ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
              ),
              trailing: IconButton(
                icon: Icon(isExpanded ? Icons.expand_less : Icons.expand_more),
                onPressed: onToggleExpanded,
              ),
              onTap: onToggleExpanded,
            );
          },
        ),
        if (isExpanded)
          for (final regionEntry in regions.entries) ...[
            ListenableBuilder(
              listenable: NetworkSelection.instance,
              builder: (context, _) {
                final feedIds = regionEntry.value.map((f) => f.id);
                final value = _selectionValue(
                  feedIds,
                  NetworkSelection.instance.selectedFeedIds,
                );
                return CheckboxListTile(
                  contentPadding: const EdgeInsets.only(left: AppSpacing.m),
                  tristate: true,
                  dense: true,
                  value: value,
                  title: Text(
                    regionEntry.key,
                    style: Theme.of(context).textTheme.labelLarge?.copyWith(
                      color: Theme.of(context).colorScheme.onSurfaceVariant,
                    ),
                  ),
                  onChanged: (value) => NetworkSelection.instance
                      .setFeedsSelected(feedIds, value ?? true),
                );
              },
            ),
            for (final feed in regionEntry.value)
              ListenableBuilder(
                listenable: NetworkSelection.instance,
                builder: (context, _) {
                  final selected = NetworkSelection.instance.selectedFeedIds
                      .contains(feed.id);
                  return CheckboxListTile(
                    contentPadding: const EdgeInsets.only(left: AppSpacing.xl),
                    value: selected,
                    title: Text(feed.name),
                    subtitle: Text(feed.description),
                    secondary: const Icon(Icons.public),
                    onChanged: (value) => NetworkSelection.instance
                        .setFeedSelected(feed.id, value ?? false),
                  );
                },
              ),
          ],
      ],
    );
  }
}

String _catalogStatus(TransitlandCatalog catalog) {
  final count = catalog.feeds.length;
  if (catalog.isLoading && count == 0) {
    return 'Loading Transitland feed catalog...';
  }
  if (catalog.isLoading) {
    return 'Refreshing $count Transitland feeds...';
  }
  if (count == 0) {
    return 'Loaded feeds: 0';
  }
  final updatedAt = catalog.updatedAt;
  if (updatedAt == null) {
    return 'Loaded feeds: $count';
  }
  return 'Loaded feeds: $count | Updated ${updatedAt.toLocal()}';
}

Map<String, Map<String, List<TransitFeed>>> _groupedSelectableFeeds(
  String query,
) {
  final normalizedQuery = query.trim().toLowerCase();
  final grouped = <String, Map<String, List<TransitFeed>>>{};
  for (final feed in selectableTransitFeeds().where(
    (feed) => !feed.isCollection,
  )) {
    if (normalizedQuery.isNotEmpty && !_matchesFeed(feed, normalizedQuery)) {
      continue;
    }
    final country = feed.country.isEmpty ? 'Other' : feed.country;
    final region = feed.region.isEmpty ? 'Other' : feed.region;
    grouped.putIfAbsent(country, () => <String, List<TransitFeed>>{});
    grouped[country]!.putIfAbsent(region, () => <TransitFeed>[]);
    grouped[country]![region]!.add(feed);
  }

  final countryKeys = grouped.keys.toList()
    ..sort((a, b) {
      if (a == 'Global') return -1;
      if (b == 'Global') return 1;
      return _countryLabel(a).compareTo(_countryLabel(b));
    });
  return {
    for (final country in countryKeys)
      country: {
        for (final region
            in (grouped[country]!.keys.toList()..sort(_compareRegionLabels)))
          region: grouped[country]![region]!,
      },
  };
}

bool _matchesFeed(TransitFeed feed, String query) {
  return feed.name.toLowerCase().contains(query) ||
      feed.description.toLowerCase().contains(query) ||
      feed.region.toLowerCase().contains(query) ||
      feed.country.toLowerCase().contains(query) ||
      feed.id.toLowerCase().contains(query);
}

Iterable<String> _feedIdsForCountry(Map<String, List<TransitFeed>> regions) =>
    regions.values.expand((feeds) => feeds).map((feed) => feed.id);

bool? _selectionValue(Iterable<String> feedIds, Set<String> selectedFeedIds) {
  final ids = feedIds.toList(growable: false);
  final selectedCount = ids.where(selectedFeedIds.contains).length;
  if (selectedCount == 0) return false;
  if (selectedCount == ids.length) return true;
  return null;
}

int _compareRegionLabels(String a, String b) {
  const priority = {'Coverage': 0, 'Country': 1, 'Nationwide': 2};
  final aPriority = priority[a] ?? 10;
  final bPriority = priority[b] ?? 10;
  if (aPriority != bPriority) return aPriority.compareTo(bPriority);
  return a.compareTo(b);
}

String _countryLabel(String country) {
  return switch (country) {
    'CH' => 'Switzerland (CH)',
    'IT' => 'Italy (IT)',
    'JP' => 'Japan (JP)',
    'Global' => 'Global',
    _ => country,
  };
}
