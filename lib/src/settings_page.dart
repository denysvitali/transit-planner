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
      ..writeln(AppLogBuffer.instance.formatted(AppLogLevel.values.toSet()));
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
                final entries = AppLogBuffer.instance.entries;
                final warnErr = entries
                    .where(
                      (e) =>
                          e.level == AppLogLevel.warning ||
                          e.level == AppLogLevel.error,
                    )
                    .length;
                return ListTile(
                  leading: const Icon(Icons.article_outlined),
                  title: const Text('Logs'),
                  subtitle: Text(
                    entries.isEmpty
                        ? 'No logs yet'
                        : '${entries.length} log entr'
                              '${entries.length == 1 ? 'y' : 'ies'}'
                              '${warnErr == 0 ? '' : ' • $warnErr warning/error'}',
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

class LogsPage extends StatefulWidget {
  const LogsPage({super.key});

  @override
  State<LogsPage> createState() => _LogsPageState();
}

class _LogsPageState extends State<LogsPage> {
  final Set<AppLogLevel> _enabledLevels = {
    AppLogLevel.info,
    AppLogLevel.warning,
    AppLogLevel.error,
  };
  bool _newestFirst = true;

  Future<void> _copyLogs() async {
    await Clipboard.setData(
      ClipboardData(text: AppLogBuffer.instance.formatted(_enabledLevels)),
    );
    if (!mounted) return;
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(const SnackBar(content: Text('Logs copied')));
  }

  void _toggleLevel(AppLogLevel level, bool selected) {
    setState(() {
      if (selected) {
        _enabledLevels.add(level);
      } else {
        _enabledLevels.remove(level);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Logs'),
        actions: [
          IconButton(
            tooltip: _newestFirst ? 'Show oldest first' : 'Show newest first',
            icon: Icon(
              _newestFirst ? Icons.arrow_downward : Icons.arrow_upward,
            ),
            onPressed: () => setState(() => _newestFirst = !_newestFirst),
          ),
        ],
      ),
      body: SafeArea(
        child: ListenableBuilder(
          listenable: AppLogBuffer.instance,
          builder: (context, _) {
            final all = AppLogBuffer.instance.entries;
            final counts = <AppLogLevel, int>{
              for (final level in AppLogLevel.values) level: 0,
            };
            for (final entry in all) {
              counts[entry.level] = (counts[entry.level] ?? 0) + 1;
            }
            final filtered = all
                .where((e) => _enabledLevels.contains(e.level))
                .toList(growable: false);
            final ordered = _newestFirst
                ? filtered.reversed.toList(growable: false)
                : filtered;
            return ListView(
              padding: const EdgeInsets.all(AppSpacing.m),
              children: [
                Wrap(
                  spacing: AppSpacing.xs,
                  runSpacing: AppSpacing.xs,
                  children: [
                    for (final level in AppLogLevel.values)
                      FilterChip(
                        selected: _enabledLevels.contains(level),
                        label: Text('${_levelLabel(level)} (${counts[level]})'),
                        avatar: Icon(_levelIcon(level), size: 18),
                        onSelected: (s) => _toggleLevel(level, s),
                      ),
                  ],
                ),
                const SizedBox(height: AppSpacing.s),
                Wrap(
                  spacing: AppSpacing.s,
                  runSpacing: AppSpacing.s,
                  children: [
                    FilledButton.icon(
                      onPressed: filtered.isEmpty ? null : _copyLogs,
                      icon: const Icon(Icons.copy_all_outlined),
                      label: const Text('Copy filtered logs'),
                    ),
                    OutlinedButton.icon(
                      onPressed: all.isEmpty
                          ? null
                          : AppLogBuffer.instance.clear,
                      icon: const Icon(Icons.delete_sweep_outlined),
                      label: const Text('Clear logs'),
                    ),
                  ],
                ),
                const SizedBox(height: AppSpacing.m),
                if (all.isEmpty)
                  const _EmptyLogState(
                    title: 'No logs yet',
                    detail: 'Activity will appear here as you use the app.',
                  )
                else if (ordered.isEmpty)
                  _EmptyLogState(
                    title: 'No logs match the current filter',
                    detail:
                        '${all.length} entr${all.length == 1 ? 'y is' : 'ies are'} '
                        'hidden. Toggle a chip above to show them.',
                  )
                else
                  ...ordered.map(
                    (entry) => Padding(
                      padding: const EdgeInsets.only(bottom: AppSpacing.s),
                      child: _LogEntryTile(entry: entry),
                    ),
                  ),
              ],
            );
          },
        ),
      ),
    );
  }
}

String _levelLabel(AppLogLevel level) => switch (level) {
  AppLogLevel.debug => 'Debug',
  AppLogLevel.info => 'Info',
  AppLogLevel.warning => 'Warnings',
  AppLogLevel.error => 'Errors',
};

IconData _levelIcon(AppLogLevel level) => switch (level) {
  AppLogLevel.debug => Icons.bug_report_outlined,
  AppLogLevel.info => Icons.info_outline,
  AppLogLevel.warning => Icons.warning_amber_outlined,
  AppLogLevel.error => Icons.error_outline,
};

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
  const _EmptyLogState({required this.title, required this.detail});

  final String title;
  final String detail;

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
          Text(title, style: theme.textTheme.titleMedium),
          const SizedBox(height: AppSpacing.xs),
          Text(
            detail,
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
            textAlign: TextAlign.center,
          ),
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
      AppLogLevel.debug => theme.colorScheme.outline,
      AppLogLevel.info => theme.colorScheme.primary,
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
            Icon(_levelIcon(entry.level), color: color),
            const SizedBox(width: AppSpacing.s),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    '${entry.level.name.toUpperCase()} ${_logClock(entry.timestamp)}',
                    style: theme.textTheme.labelLarge?.copyWith(color: color),
                  ),
                  const SizedBox(height: AppSpacing.xs),
                  SelectableText(entry.message),
                  if (entry.stackTrace != null) ...[
                    const SizedBox(height: AppSpacing.xs),
                    SelectableText(
                      entry.stackTrace.toString(),
                      style: theme.textTheme.bodySmall?.copyWith(
                        fontFamily: 'monospace',
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

String _logClock(DateTime value) {
  final h = value.hour.toString().padLeft(2, '0');
  final m = value.minute.toString().padLeft(2, '0');
  final s = value.second.toString().padLeft(2, '0');
  return '$h:$m:$s';
}
