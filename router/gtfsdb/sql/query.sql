-- name: UpsertFeed :exec
insert into feeds (
	code, name, country_code, region, publisher_name, license_name, source_url, attribution_text
) values (?, ?, ?, ?, ?, ?, ?, ?)
on conflict(code) do update set
	name = excluded.name,
	country_code = excluded.country_code,
	region = excluded.region,
	publisher_name = excluded.publisher_name,
	license_name = excluded.license_name,
	source_url = excluded.source_url,
	attribution_text = excluded.attribution_text;

-- name: GetFeedIDByCode :one
select id from feeds where code = ?;

-- name: DeactivateFeedVersions :exec
update feed_versions set active = 0 where feed_id = ?;

-- name: InsertFeedVersion :execresult
insert into feed_versions (
	feed_id, source_path, source_url, sha256, imported_at, active
) values (?, ?, ?, ?, ?, 1);

-- name: InsertGTFSFile :execresult
insert into gtfs_files (
	feed_id, feed_version_id, filename, sha256, header_json, raw_csv
) values (?, ?, ?, ?, ?, ?);

-- name: UpdateGTFSFileRowCount :exec
update gtfs_files set row_count = ? where id = ?;

-- name: InsertGTFSRow :exec
insert into gtfs_rows (
	feed_id, feed_version_id, file_id, filename, row_index, row_json
) values (?, ?, ?, ?, ?, ?);

-- name: InsertAgency :exec
insert into agencies (
	feed_id, feed_version_id, agency_id, agency_name, agency_url, agency_timezone,
	agency_lang, agency_phone, agency_fare_url, agency_email
) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertStop :exec
insert into stops (
	feed_id, feed_version_id, stop_id, stop_code, stop_name, stop_desc, stop_lat,
	stop_lon, zone_id, stop_url, location_type, parent_station, stop_timezone,
	wheelchair_boarding, platform_code
) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertRoute :exec
insert into routes (
	feed_id, feed_version_id, route_id, agency_id, route_short_name, route_long_name,
	route_desc, route_type, route_url, route_color, route_text_color, route_sort_order,
	continuous_pickup
) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertTrip :exec
insert into trips (
	feed_id, feed_version_id, route_id, service_id, trip_id, trip_headsign,
	trip_short_name, direction_id, block_id, shape_id, wheelchair_accessible,
	bikes_allowed, cars_allowed
) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertStopTime :exec
insert into stop_times (
	feed_id, feed_version_id, row_index, trip_id, arrival_time, departure_time,
	stop_id, stop_sequence, stop_headsign, pickup_type, drop_off_type, continuous_pickup,
	continuous_drop_off, shape_dist_traveled, timepoint
) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertCalendar :exec
insert into calendars (
	feed_id, feed_version_id, service_id, monday, tuesday, wednesday, thursday,
	friday, saturday, sunday, start_date, end_date
) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertCalendarDate :exec
insert into calendar_dates (
	feed_id, feed_version_id, service_id, date, exception_type
) values (?, ?, ?, ?, ?);

-- name: InsertTransfer :exec
insert into transfers (
	feed_id, feed_version_id, from_stop_id, to_stop_id, from_route_id, to_route_id,
	from_trip_id, to_trip_id, transfer_type, min_transfer_time
) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertShape :exec
insert into shapes (
	feed_id, feed_version_id, shape_id, shape_pt_lat, shape_pt_lon, shape_pt_sequence
) values (?, ?, ?, ?, ?, ?);

-- name: InsertFeedInfo :exec
insert into feed_info (
	feed_id, feed_version_id, feed_publisher_name, feed_publisher_url, feed_lang,
	default_lang, feed_start_date, feed_end_date, feed_version, feed_contact_email
) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
