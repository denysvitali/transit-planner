import 'package:flutter/material.dart';

import 'feed_catalog.dart';
import 'theme.dart';

class LoadedFeedsDebugView extends StatelessWidget {
  const LoadedFeedsDebugView({
    super.key,
    required this.feed,
    required this.stopCount,
    this.maxVisibleFeeds = 4,
  });

  final TransitFeed feed;
  final int stopCount;
  final int maxVisibleFeeds;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final feeds = componentFeedsFor(feed);
    final visibleFeeds = feeds.take(maxVisibleFeeds).toList(growable: false);
    final hiddenCount = feeds.length - visibleFeeds.length;

    return Semantics(
      label: 'Loaded feeds debug view',
      child: Material(
        elevation: 4,
        borderRadius: BorderRadius.circular(AppRadius.s),
        color: theme.colorScheme.surface.withValues(alpha: 0.92),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 280),
          child: Padding(
            padding: const EdgeInsets.all(AppSpacing.s),
            child: DefaultTextStyle(
              style: theme.textTheme.bodySmall!.copyWith(
                color: theme.colorScheme.onSurface,
              ),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Icon(
                        Icons.storage_outlined,
                        size: 16,
                        color: theme.colorScheme.primary,
                      ),
                      const SizedBox(width: AppSpacing.xs),
                      Flexible(
                        child: Text(
                          'Loaded feeds: ${feeds.length} | Stops: $stopCount',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: theme.textTheme.labelMedium?.copyWith(
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: AppSpacing.xs),
                  for (final feed in visibleFeeds)
                    Padding(
                      padding: const EdgeInsets.only(bottom: 2),
                      child: Row(
                        children: [
                          Icon(
                            Icons.check_circle,
                            size: 12,
                            color: theme.colorScheme.primary,
                          ),
                          const SizedBox(width: 6),
                          Expanded(
                            child: Text(
                              feed.name,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                            ),
                          ),
                        ],
                      ),
                    ),
                  if (hiddenCount > 0)
                    Padding(
                      padding: const EdgeInsets.only(top: 2),
                      child: Text(
                        '+ $hiddenCount more',
                        style: theme.textTheme.labelSmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
