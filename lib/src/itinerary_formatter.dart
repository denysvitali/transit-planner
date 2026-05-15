import 'models.dart';

String formatItineraryDetails(Itinerary itinerary) {
  final buffer = StringBuffer()
    ..writeln(
      'Trip ${_clock(itinerary.departure)}-${_clock(itinerary.arrival)}',
    )
    ..writeln('Duration: ${itinerary.duration.inMinutes} min')
    ..writeln(
      'Transfers: ${itinerary.transfers} | Walk: ${itinerary.walking.inMinutes} min',
    )
    ..writeln();

  for (var i = 0; i < itinerary.legs.length; i++) {
    final leg = itinerary.legs[i];
    final label = leg.routeName?.isNotEmpty == true
        ? leg.routeName!
        : _modeLabel(leg.mode);
    buffer
      ..writeln(
        '${i + 1}. ${_clock(leg.departure)}-${_clock(leg.arrival)} $label',
      )
      ..writeln('   ${leg.from.name} -> ${leg.to.name}')
      ..writeln('   ${leg.duration.inMinutes} min');
    if (leg.tripId?.isNotEmpty == true) {
      buffer.writeln('   Trip ID: ${leg.tripId}');
    }
  }

  return buffer.toString().trimRight();
}

String _modeLabel(TransitMode mode) {
  return switch (mode) {
    TransitMode.walk => 'Walk',
    TransitMode.bus => 'Bus',
    TransitMode.tram => 'Tram',
    TransitMode.rail => 'Rail',
    TransitMode.subway => 'Metro',
    TransitMode.ferry => 'Ferry',
  };
}

String _clock(DateTime value) {
  final h = value.hour.toString().padLeft(2, '0');
  final m = value.minute.toString().padLeft(2, '0');
  return '$h:$m';
}
