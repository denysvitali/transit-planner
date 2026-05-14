import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'app_log.dart';
import 'feed_catalog.dart';
import 'theme.dart';

class SettingsPage extends StatelessWidget {
  const SettingsPage({super.key});

  static const _logLevels = {AppLogLevel.warning, AppLogLevel.error};

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
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(AppSpacing.m),
          children: [
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
            const SizedBox(height: AppSpacing.l),
            const _FeedSection(),
            const SizedBox(height: AppSpacing.l),
            const _AboutSection(),
          ],
        ),
      ),
    );
  }
}

class _FeedSection extends StatefulWidget {
  const _FeedSection();

  @override
  State<_FeedSection> createState() => _FeedSectionState();
}

class _FeedSectionState extends State<_FeedSection> {
  static const _feedSelectionStorageKey = 'selected_feed_id';

  String? _selectedFeedId;

  @override
  void initState() {
    super.initState();
    _loadSelectedFeed();
  }

  Future<void> _loadSelectedFeed() async {
    final prefs = await SharedPreferences.getInstance();
    if (!mounted) return;
    setState(() => _selectedFeedId = prefs.getString(_feedSelectionStorageKey));
  }

  Future<void> _setActiveFeed(String feedId) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_feedSelectionStorageKey, feedId);
    if (!mounted) return;
    setState(() => _selectedFeedId = feedId);
  }

  @override
  Widget build(BuildContext context) {
    final selected = findFeedById(_selectedFeedId ?? kDefaultFeedId);
    final primaryColor = Theme.of(context).colorScheme.primary;
    final mutedColor = Theme.of(context).textTheme.bodySmall?.color;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.m),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Transit feeds',
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: AppSpacing.s),
            for (final feed in kTransitFeeds)
              Padding(
                padding: const EdgeInsets.only(bottom: AppSpacing.s),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    InkWell(
                      onTap: () async => _setActiveFeed(feed.id),
                      child: Row(
                        children: [
                          Icon(
                            feed.id == selected?.id
                                ? Icons.radio_button_checked
                                : Icons.radio_button_off,
                            size: 18,
                            color: feed.id == selected?.id
                                ? primaryColor
                                : mutedColor,
                          ),
                          const SizedBox(width: AppSpacing.s),
                          Expanded(
                            child: Text(
                              feed.name,
                              style: Theme.of(context).textTheme.titleMedium,
                            ),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      feed.description,
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                    if (feed.id == selected?.id)
                      Text(
                        'Set as default for next launch',
                        style: Theme.of(context).textTheme.labelSmall?.copyWith(
                              color: Theme.of(context).colorScheme.primary,
                            ),
                      )
                  ],
                ),
              ),
            Text(
              'Default for next launch: ${selected?.name ?? kDefaultFeedId}',
              style: Theme.of(context).textTheme.titleSmall,
            ),
          ],
        ),
      ),
    );
  }
}

/// About / attributions section.
class _AboutSection extends StatelessWidget {
  const _AboutSection();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
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
        Text('Transit data sources', style: theme.textTheme.titleMedium),
        const SizedBox(height: AppSpacing.xs),
        ...kTransitFeeds
            .map(
              (feed) => Padding(
                padding: const EdgeInsets.only(bottom: AppSpacing.s),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      feed.name,
                      style: theme.textTheme.titleSmall,
                    ),
                    const SizedBox(height: AppSpacing.xs),
                    SelectableText(
                      feed.attribution,
                      style: theme.textTheme.bodySmall,
                    ),
                  ],
                ),
              ),
            )
            .toList(growable: false),
        const SizedBox(height: AppSpacing.xs),
        Text(
          'Licences and terms are listed in the same order as the catalog.',
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
