create table if not exists feeds (
	id integer primary key,
	code text not null unique,
	name text not null,
	country_code text,
	region text,
	publisher_name text,
	license_name text,
	source_url text,
	attribution_text text,
	created_at text not null default current_timestamp
);

create table if not exists feed_versions (
	id integer primary key,
	feed_id integer not null references feeds(id),
	source_path text not null,
	source_url text,
	sha256 text not null,
	imported_at text not null,
	active integer not null default 0
);

create table if not exists gtfs_files (
	id integer primary key,
	feed_id integer not null references feeds(id),
	feed_version_id integer not null references feed_versions(id),
	filename text not null,
	sha256 text not null,
	header_json text not null,
	row_count integer not null default 0,
	raw_csv blob not null
);

create table if not exists gtfs_rows (
	id integer primary key,
	feed_id integer not null references feeds(id),
	feed_version_id integer not null references feed_versions(id),
	file_id integer not null references gtfs_files(id),
	filename text not null,
	row_index integer not null,
	row_json text not null
);

create table if not exists agencies (
	feed_id integer not null references feeds(id),
	feed_version_id integer not null references feed_versions(id),
	agency_id text not null,
	agency_name text,
	agency_url text,
	agency_timezone text,
	agency_lang text,
	agency_phone text,
	agency_fare_url text,
	agency_email text,
	primary key (feed_version_id, agency_id)
);

create table if not exists stops (
	feed_id integer not null references feeds(id),
	feed_version_id integer not null references feed_versions(id),
	stop_id text not null,
	stop_code text,
	stop_name text,
	stop_desc text,
	stop_lat real,
	stop_lon real,
	zone_id text,
	stop_url text,
	location_type integer,
	parent_station text,
	stop_timezone text,
	wheelchair_boarding text,
	platform_code text,
	primary key (feed_version_id, stop_id)
);

create table if not exists routes (
	feed_id integer not null references feeds(id),
	feed_version_id integer not null references feed_versions(id),
	route_id text not null,
	agency_id text,
	route_short_name text,
	route_long_name text,
	route_desc text,
	route_type integer,
	route_url text,
	route_color text,
	route_text_color text,
	route_sort_order text,
	continuous_pickup text,
	primary key (feed_version_id, route_id)
);

create table if not exists trips (
	feed_id integer not null references feeds(id),
	feed_version_id integer not null references feed_versions(id),
	route_id text not null,
	service_id text not null,
	trip_id text not null,
	trip_headsign text,
	trip_short_name text,
	direction_id integer,
	block_id text,
	shape_id text,
	wheelchair_accessible text,
	bikes_allowed text,
	cars_allowed text,
	primary key (feed_version_id, trip_id)
);

create table if not exists stop_times (
	feed_id integer not null references feeds(id),
	feed_version_id integer not null references feed_versions(id),
	row_index integer not null,
	trip_id text not null,
	arrival_time text,
	departure_time text,
	stop_id text not null,
	stop_sequence integer,
	stop_headsign text,
	pickup_type text,
	drop_off_type text,
	continuous_pickup text,
	continuous_drop_off text,
	shape_dist_traveled real,
	timepoint text,
	primary key (feed_version_id, row_index)
);

create table if not exists calendars (
	feed_id integer not null references feeds(id),
	feed_version_id integer not null references feed_versions(id),
	service_id text not null,
	monday integer,
	tuesday integer,
	wednesday integer,
	thursday integer,
	friday integer,
	saturday integer,
	sunday integer,
	start_date text,
	end_date text,
	primary key (feed_version_id, service_id)
);

create table if not exists calendar_dates (
	feed_id integer not null references feeds(id),
	feed_version_id integer not null references feed_versions(id),
	service_id text not null,
	date text not null,
	exception_type integer,
	primary key (feed_version_id, service_id, date)
);

create table if not exists transfers (
	feed_id integer not null references feeds(id),
	feed_version_id integer not null references feed_versions(id),
	from_stop_id text,
	to_stop_id text,
	from_route_id text,
	to_route_id text,
	from_trip_id text,
	to_trip_id text,
	transfer_type integer,
	min_transfer_time integer
);

create table if not exists shapes (
	feed_id integer not null references feeds(id),
	feed_version_id integer not null references feed_versions(id),
	shape_id text not null,
	shape_pt_lat real,
	shape_pt_lon real,
	shape_pt_sequence integer,
	primary key (feed_version_id, shape_id, shape_pt_sequence)
);

create table if not exists feed_info (
	feed_id integer not null references feeds(id),
	feed_version_id integer not null references feed_versions(id),
	feed_publisher_name text,
	feed_publisher_url text,
	feed_lang text,
	default_lang text,
	feed_start_date text,
	feed_end_date text,
	feed_version text,
	feed_contact_email text
);

create index if not exists idx_feed_versions_feed_active on feed_versions(feed_id, active);
create index if not exists idx_gtfs_files_feed_filename on gtfs_files(feed_id, filename);
create index if not exists idx_gtfs_rows_feed_filename on gtfs_rows(feed_id, filename);
create index if not exists idx_agencies_feed on agencies(feed_id);
create index if not exists idx_stops_feed on stops(feed_id);
create index if not exists idx_stops_spatial on stops(stop_lat, stop_lon);
create index if not exists idx_routes_feed_route on routes(feed_id, route_id);
create index if not exists idx_trips_feed_trip on trips(feed_id, trip_id);
create index if not exists idx_trips_route_service on trips(feed_version_id, route_id, service_id);
create index if not exists idx_stop_times_feed on stop_times(feed_id);
create index if not exists idx_stop_times_trip on stop_times(feed_version_id, trip_id, stop_sequence);
create index if not exists idx_stop_times_stop_departure on stop_times(feed_version_id, stop_id, departure_time);
create index if not exists idx_calendars_feed on calendars(feed_id);
create index if not exists idx_calendar_dates_feed on calendar_dates(feed_id);
create index if not exists idx_calendar_dates_service_date on calendar_dates(feed_version_id, service_id, date);
create index if not exists idx_transfers_feed on transfers(feed_id);
create index if not exists idx_transfers_from on transfers(feed_version_id, from_stop_id);
create index if not exists idx_shapes_feed on shapes(feed_id);
create index if not exists idx_shapes_shape on shapes(feed_version_id, shape_id, shape_pt_sequence);
create index if not exists idx_feed_info_feed on feed_info(feed_id);

create view if not exists active_agencies as
	select agencies.*
	from agencies
	join feed_versions on feed_versions.id = agencies.feed_version_id
	where feed_versions.active = 1;

create view if not exists active_stops as
	select stops.*
	from stops
	join feed_versions on feed_versions.id = stops.feed_version_id
	where feed_versions.active = 1;

create view if not exists active_routes as
	select routes.*
	from routes
	join feed_versions on feed_versions.id = routes.feed_version_id
	where feed_versions.active = 1;

create view if not exists active_trips as
	select trips.*
	from trips
	join feed_versions on feed_versions.id = trips.feed_version_id
	where feed_versions.active = 1;

create view if not exists active_stop_times as
	select stop_times.*
	from stop_times
	join feed_versions on feed_versions.id = stop_times.feed_version_id
	where feed_versions.active = 1;

create view if not exists active_calendars as
	select calendars.*
	from calendars
	join feed_versions on feed_versions.id = calendars.feed_version_id
	where feed_versions.active = 1;

create view if not exists active_calendar_dates as
	select calendar_dates.*
	from calendar_dates
	join feed_versions on feed_versions.id = calendar_dates.feed_version_id
	where feed_versions.active = 1;

create view if not exists active_transfers as
	select transfers.*
	from transfers
	join feed_versions on feed_versions.id = transfers.feed_version_id
	where feed_versions.active = 1;

create view if not exists active_shapes as
	select shapes.*
	from shapes
	join feed_versions on feed_versions.id = shapes.feed_version_id
	where feed_versions.active = 1;

create view if not exists active_feed_info as
	select feed_info.*
	from feed_info
	join feed_versions on feed_versions.id = feed_info.feed_version_id
	where feed_versions.active = 1;
