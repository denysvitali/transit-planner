import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';

import 'app_log.dart';
import 'feed_catalog.dart';
import 'network_selection.dart';
import 'theme.dart';
import 'transitland_catalog.dart';

class SettingsPage extends StatelessWidget {
  const SettingsPage({super.key});

  Future<void> _copyDiagnostics(BuildContext context) async {
    final selection = NetworkSelection.instance;
    final report = StringBuffer()
      ..writeln('Transit Planner diagnostics')
      ..writeln('Active feed: ${selection.feed.id} (${selection.feed.name})')
      ..writeln('Selected feeds: ${selection.selectedFeedIds.join(', ')}')
      ..writeln()
      ..writeln(AppLogBuffer.instance.formatted(_logLevels));
    await Clipboard.setData(ClipboardData(text: report.toString().trimRight()));
    if (!context.mounted) return;
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(const SnackBar(content: Text('Diagnostic report copied')));
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(AppSpacing.m),
          children: [
            const _NetworkSection(),
            const SizedBox(height: AppSpacing.l),
            Text(
              'Developer options',
              style: theme.textTheme.titleLarge?.copyWith(
                fontWeight: FontWeight.w800,
              ),
            ),
            const SizedBox(height: AppSpacing.s),
            ListenableBuilder(
              listenable: AppLogBuffer.instance,
              builder: (context, _) {
                final logCount = AppLogBuffer.instance
                    .entriesFor(_logLevels)
                    .length;
                return ListTile(
                  leading: const Icon(Icons.article_outlined),
                  title: const Text('Logs'),
                  subtitle: Text(
                    logCount == 0
                        ? 'No warning or error logs'
                        : '$logCount warning or error log'
                              '${logCount == 1 ? '' : 's'}',
                  ),
                  trailing: const Icon(Icons.chevron_right),
                  contentPadding: EdgeInsets.zero,
                  onTap: () => context.push('/settings/logs'),
                );
              },
            ),
            ListTile(
              leading: const Icon(Icons.copy_all_outlined),
              title: const Text('Copy diagnostic report'),
              subtitle: const Text('Active feed, selected networks, and logs'),
              contentPadding: EdgeInsets.zero,
              onTap: () => _copyDiagnostics(context),
            ),
            const SizedBox(height: AppSpacing.l),
            const _AboutSection(),
          ],
        ),
      ),
    );
  }
}

class LogsPage extends StatelessWidget {
  const LogsPage({super.key});

  Future<void> _copyLogs(BuildContext context) async {
    await Clipboard.setData(
      ClipboardData(text: AppLogBuffer.instance.formatted(_logLevels)),
    );
    if (!context.mounted) {
      return;
    }
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('Warning and error logs copied')),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Logs')),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(AppSpacing.m),
          children: [
            ListenableBuilder(
              listenable: AppLogBuffer.instance,
              builder: (context, _) {
                final logs = AppLogBuffer.instance.entriesFor(_logLevels);
                return Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Wrap(
                      spacing: AppSpacing.s,
                      runSpacing: AppSpacing.s,
                      children: [
                        FilledButton.icon(
                          onPressed: () => _copyLogs(context),
                          icon: const Icon(Icons.copy_all_outlined),
                          label: const Text('Copy warnings and errors'),
                        ),
                        OutlinedButton.icon(
                          onPressed: logs.isEmpty
                              ? null
                              : AppLogBuffer.instance.clear,
                          icon: const Icon(Icons.delete_sweep_outlined),
                          label: const Text('Clear logs'),
                        ),
                      ],
                    ),
                    const SizedBox(height: AppSpacing.m),
                    if (logs.isEmpty)
                      const _EmptyLogState()
                    else
                      ...logs.map(
                        (entry) => Padding(
                          padding: const EdgeInsets.only(bottom: AppSpacing.s),
                          child: _LogEntryTile(entry: entry),
                        ),
                      ),
                  ],
                );
              },
            ),
          ],
        ),
      ),
    );
  }
}

const _logLevels = {AppLogLevel.warning, AppLogLevel.error};

class _NetworkSection extends StatefulWidget {
  const _NetworkSection();

  @override
  State<_NetworkSection> createState() => _NetworkSectionState();
}

class _NetworkSectionState extends State<_NetworkSection> {
  @override
  void initState() {
    super.initState();
    final catalog = TransitlandCatalog.instance;
    if (!catalog.hasLoaded && !catalog.isLoading) {
      catalog.load();
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return ListenableBuilder(
      listenable: Listenable.merge([
        NetworkSelection.instance,
        TransitlandCatalog.instance,
      ]),
      builder: (context, _) {
        final catalog = TransitlandCatalog.instance;
        final selectedFeedIds = NetworkSelection.instance.selectedFeedIds;
        final groupedFeeds = _groupedSelectableFeeds();
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              'Transitland feeds',
              style: theme.textTheme.titleLarge?.copyWith(
                fontWeight: FontWeight.w800,
              ),
            ),
            const SizedBox(height: AppSpacing.s),
            Row(
              children: [
                Expanded(
                  child: Text(
                    _catalogStatus(catalog),
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ),
                TextButton.icon(
                  onPressed: catalog.isLoading
                      ? null
                      : () => TransitlandCatalog.instance.load(
                          forceRefresh: true,
                        ),
                  icon: const Icon(Icons.sync),
                  label: const Text('Refresh'),
                ),
              ],
            ),
            if (catalog.error != null) ...[
              const SizedBox(height: AppSpacing.xs),
              Text(
                catalog.error!,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.error,
                ),
              ),
            ],
            if (catalog.isLoading && groupedFeeds.isEmpty)
              const Padding(
                padding: EdgeInsets.symmetric(vertical: AppSpacing.m),
                child: LinearProgressIndicator(),
              )
            else if (groupedFeeds.isEmpty)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: AppSpacing.m),
                child: Text(
                  'No Transitland feeds are loaded yet.',
                  style: theme.textTheme.bodyMedium,
                ),
              ),
            for (final country in groupedFeeds.entries) ...[
              CheckboxListTile(
                contentPadding: EdgeInsets.zero,
                tristate: true,
                value: _selectionValue(
                  _feedIdsForCountry(country.value),
                  selectedFeedIds,
                ),
                title: Text(
                  _countryLabel(country.key),
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.w700,
                  ),
                ),
                onChanged: (value) =>
                    NetworkSelection.instance.setFeedsSelected(
                      _feedIdsForCountry(country.value),
                      value ?? true,
                    ),
              ),
              for (final region in country.value.entries) ...[
                CheckboxListTile(
                  contentPadding: const EdgeInsets.only(left: AppSpacing.m),
                  tristate: true,
                  dense: true,
                  value: _selectionValue(
                    region.value.map((feed) => feed.id),
                    selectedFeedIds,
                  ),
                  title: Text(
                    region.key,
                    style: theme.textTheme.labelLarge?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                  onChanged: (value) =>
                      NetworkSelection.instance.setFeedsSelected(
                        region.value.map((feed) => feed.id),
                        value ?? true,
                      ),
                ),
                for (final feed in region.value)
                  _FeedOption(
                    feed: feed,
                    selected: selectedFeedIds.contains(feed.id),
                    onChanged: (selected) => NetworkSelection.instance
                        .setFeedSelected(feed.id, selected),
                  ),
              ],
            ],
          ],
        );
      },
    );
  }
}

class _FeedOption extends StatelessWidget {
  const _FeedOption({
    required this.feed,
    required this.selected,
    required this.onChanged,
  });

  final TransitFeed feed;
  final bool selected;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    return CheckboxListTile(
      contentPadding: const EdgeInsets.only(left: AppSpacing.xl),
      value: selected,
      title: Text(feed.name),
      subtitle: Text(feed.description),
      secondary: const Icon(Icons.public),
      onChanged: (value) => onChanged(value ?? false),
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

Map<String, Map<String, List<TransitFeed>>> _groupedSelectableFeeds() {
  final grouped = <String, Map<String, List<TransitFeed>>>{};
  for (final feed in selectableTransitFeeds().where(
    (feed) => !feed.isCollection,
  )) {
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

/// About / attributions section.
class _AboutSection extends StatelessWidget {
  const _AboutSection();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final loadedCount = TransitlandCatalog.instance.feeds.length;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          'About',
          style: theme.textTheme.titleLarge?.copyWith(
            fontWeight: FontWeight.w800,
          ),
        ),
        const SizedBox(height: AppSpacing.s),
        Text('Transitland runtime catalog', style: theme.textTheme.titleMedium),
        const SizedBox(height: AppSpacing.xs),
        Text(
          'Feeds are discovered from the Transitland REST API at runtime and '
          'cached locally. No app-maintained GTFS feed list is used.',
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: AppSpacing.xs),
        Text(
          'Loaded Transitland feeds: $loadedCount. Feed downloads use '
          'Transitland static GTFS download endpoints.',
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: AppSpacing.xs),
        Text(
          'Licences and attribution come from Transitland feed metadata and '
          'vary by publisher.',
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        Text(
          'See LICENSES_THIRD_PARTY.md in the source tree for the full list '
          'of bundled and downloaded datasets and their licences.',
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }
}

class _EmptyLogState extends StatelessWidget {
  const _EmptyLogState();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: AppSpacing.l),
      child: Column(
        children: [
          Icon(
            Icons.check_circle_outline,
            color: theme.colorScheme.primary,
            size: 32,
          ),
          const SizedBox(height: AppSpacing.s),
          Text('No warning or error logs', style: theme.textTheme.titleMedium),
        ],
      ),
    );
  }
}

class _LogEntryTile extends StatelessWidget {
  const _LogEntryTile({required this.entry});

  final AppLogEntry entry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final color = switch (entry.level) {
      AppLogLevel.warning => theme.colorScheme.tertiary,
      AppLogLevel.error => theme.colorScheme.error,
    };
    return DecoratedBox(
      decoration: BoxDecoration(
        border: Border(
          bottom: BorderSide(color: theme.colorScheme.outlineVariant),
        ),
      ),
      child: Padding(
        padding: const EdgeInsets.only(bottom: AppSpacing.s),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(
              entry.level == AppLogLevel.warning
                  ? Icons.warning_amber_outlined
                  : Icons.error_outline,
              color: color,
            ),
            const SizedBox(width: AppSpacing.s),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    '${entry.level.name.toUpperCase()} ${_clock(entry.timestamp)}',
                    style: theme.textTheme.labelLarge?.copyWith(color: color),
                  ),
                  const SizedBox(height: AppSpacing.xs),
                  SelectableText(entry.message),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

String _clock(DateTime value) {
  final h = value.hour.toString().padLeft(2, '0');
  final m = value.minute.toString().padLeft(2, '0');
  return '$h:$m';
}
