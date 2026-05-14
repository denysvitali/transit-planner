import 'package:flutter/material.dart';

import 'models.dart';

/// A small in-memory list of plausible stops so that the picker works without
/// a real GTFS feed wired in. Coordinates are around Bern, Switzerland.
const List<TransitStop> kMockStops = <TransitStop>[
  TransitStop(
    id: 'bern_bahnhof',
    name: 'Bern Bahnhof',
    latitude: 46.948,
    longitude: 7.439,
  ),
  TransitStop(
    id: 'wankdorf',
    name: 'Wankdorf',
    latitude: 46.963,
    longitude: 7.465,
  ),
  TransitStop(
    id: 'bern_zytglogge',
    name: 'Bern Zytglogge',
    latitude: 46.948,
    longitude: 7.448,
  ),
  TransitStop(
    id: 'bern_helvetiaplatz',
    name: 'Bern Helvetiaplatz',
    latitude: 46.944,
    longitude: 7.448,
  ),
  TransitStop(
    id: 'bern_bethlehem',
    name: 'Bern Bethlehem',
    latitude: 46.953,
    longitude: 7.391,
  ),
  TransitStop(
    id: 'bern_breitenrain',
    name: 'Bern Breitenrain',
    latitude: 46.957,
    longitude: 7.451,
  ),
  TransitStop(
    id: 'koeniz_zentrum',
    name: 'Köniz Zentrum',
    latitude: 46.924,
    longitude: 7.414,
  ),
  TransitStop(
    id: 'ostermundigen',
    name: 'Ostermundigen Bahnhof',
    latitude: 46.957,
    longitude: 7.485,
  ),
];

/// Search delegate that filters [TransitStop]s by case-insensitive substring
/// match on the stop name. Returns the selected stop, or null when dismissed.
class StopSearchDelegate extends SearchDelegate<TransitStop?> {
  StopSearchDelegate({
    required this.stops,
    String hint = 'Search stops',
  }) : super(searchFieldLabel: hint);

  final List<TransitStop> stops;

  List<TransitStop> _matches(String input) {
    final q = input.trim().toLowerCase();
    if (q.isEmpty) {
      return stops;
    }
    return stops
        .where((stop) => stop.name.toLowerCase().contains(q))
        .toList(growable: false);
  }

  @override
  List<Widget>? buildActions(BuildContext context) {
    return <Widget>[
      if (query.isNotEmpty)
        IconButton(
          tooltip: 'Clear',
          icon: const Icon(Icons.clear),
          onPressed: () => query = '',
        ),
    ];
  }

  @override
  Widget? buildLeading(BuildContext context) {
    return IconButton(
      tooltip: 'Back',
      icon: const Icon(Icons.arrow_back),
      onPressed: () => close(context, null),
    );
  }

  @override
  Widget buildResults(BuildContext context) => _buildList(context);

  @override
  Widget buildSuggestions(BuildContext context) => _buildList(context);

  Widget _buildList(BuildContext context) {
    final results = _matches(query);
    if (results.isEmpty) {
      return const Center(
        child: Padding(
          padding: EdgeInsets.all(24),
          child: Text('No matching stops'),
        ),
      );
    }
    return ListView.builder(
      itemCount: results.length,
      itemBuilder: (context, index) {
        final stop = results[index];
        return ListTile(
          leading: const Icon(Icons.place_outlined),
          title: Text(stop.name),
          subtitle: Text(
            '${stop.latitude.toStringAsFixed(4)}, '
            '${stop.longitude.toStringAsFixed(4)}',
          ),
          onTap: () => close(context, stop),
        );
      },
    );
  }
}

/// A tappable field that displays the currently selected [TransitStop] (or a
/// hint) and, when tapped, opens [StopSearchDelegate] via [showSearch].
class StopPickerField extends StatelessWidget {
  const StopPickerField({
    super.key,
    required this.label,
    required this.icon,
    required this.stop,
    required this.onChanged,
    this.stops = kMockStops,
    this.hint,
  });

  final String label;
  final IconData icon;
  final TransitStop? stop;
  final ValueChanged<TransitStop> onChanged;
  final List<TransitStop> stops;
  final String? hint;

  Future<void> _openSearch(BuildContext context) async {
    final selected = await showSearch<TransitStop?>(
      context: context,
      delegate: StopSearchDelegate(
        stops: stops,
        hint: hint ?? 'Search $label',
      ),
    );
    if (selected != null) {
      onChanged(selected);
    }
  }

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: () => _openSearch(context),
      child: InputDecorator(
        decoration: InputDecoration(
          labelText: label,
          prefixIcon: Icon(icon),
        ),
        child: Text(
          stop?.name ?? (hint ?? 'Tap to choose a stop'),
          style: stop == null
              ? Theme.of(context).textTheme.bodyMedium?.copyWith(
                    color: Theme.of(context).hintColor,
                  )
              : Theme.of(context).textTheme.bodyLarge,
        ),
      ),
    );
  }
}
