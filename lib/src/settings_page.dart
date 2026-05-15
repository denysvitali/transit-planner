import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';

import 'app_log.dart';
import 'feed_catalog.dart';
import 'network_selection.dart';
import 'theme.dart';

class SettingsPage extends StatelessWidget {
  const SettingsPage({super.key});

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

class _NetworkSection extends StatelessWidget {
  const _NetworkSection();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return ListenableBuilder(
      listenable: NetworkSelection.instance,
      builder: (context, _) {
        final activeFeed = NetworkSelection.instance.feed;
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              'Network',
              style: theme.textTheme.titleLarge?.copyWith(
                fontWeight: FontWeight.w800,
              ),
            ),
            const SizedBox(height: AppSpacing.s),
            for (final feed in appNetworkFeeds())
              _NetworkOption(
                feed: feed,
                selected: feed.id == activeFeed.id,
                onTap: () => NetworkSelection.instance.select(feed),
              ),
          ],
        );
      },
    );
  }
}

class _NetworkOption extends StatelessWidget {
  const _NetworkOption({
    required this.feed,
    required this.selected,
    required this.onTap,
  });

  final TransitFeed feed;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return ListTile(
      contentPadding: EdgeInsets.zero,
      leading: Icon(
        selected ? Icons.radio_button_checked : Icons.radio_button_off,
        color: selected ? theme.colorScheme.primary : null,
      ),
      title: Text(feed.name),
      subtitle: Text(_networkSubtitle(feed)),
      trailing: feed.id == 'transitland-coverage'
          ? const Icon(Icons.public)
          : null,
      onTap: onTap,
    );
  }
}

String _networkSubtitle(TransitFeed feed) {
  final componentCount = componentFeedsFor(feed).length;
  if (!feed.isCollection) {
    return feed.description;
  }
  return '$componentCount GTFS feeds · ${feed.description}';
}

/// About / attributions section.
class _AboutSection extends StatelessWidget {
  const _AboutSection();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final coverageFeed =
        findFeedById('transitland-coverage') ?? findFeedById(kDefaultFeedId)!;
    final attributionFeeds = componentFeedsFor(coverageFeed);
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
        Text(coverageFeed.name, style: theme.textTheme.titleMedium),
        const SizedBox(height: AppSpacing.xs),
        Text(
          coverageFeed.description,
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: AppSpacing.xs),
        Text(
          'Feeds are shown for attribution and diagnostics. The app does not '
          'download this whole coverage list on startup.',
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: AppSpacing.xs),
        ...attributionFeeds.map(
          (feed) => Padding(
            padding: const EdgeInsets.only(bottom: AppSpacing.s),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(feed.name, style: theme.textTheme.titleSmall),
                const SizedBox(height: AppSpacing.xs),
                SelectableText(
                  feed.attribution,
                  style: theme.textTheme.bodySmall,
                ),
              ],
            ),
          ),
        ),
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
