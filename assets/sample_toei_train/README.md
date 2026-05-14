# Toei Train GTFS — vendored real feed

A frozen copy of the Tokyo Metropolitan Bureau of Transportation (都営地下鉄)
static GTFS-JP timetable, used as the **real-world test fixture** for the Go
router and as a small reference dataset for the Dart/Flutter UI.

This is **not** a synthetic feed — it is the actual production timetable
published by Toei via the Public Transportation Open Data Center (ODPT).

## Source

- Origin: <https://api-public.odpt.org/api/v4/files/Toei/data/Toei-Train-GTFS.zip>
- Publisher: 東京都交通局 (Tokyo Metropolitan Bureau of Transportation)
- Distribution: ODPT Public Transportation Open Data Center
- License: **CC-BY 4.0**. Attribution required — see
  [`LICENSES_THIRD_PARTY.md`](../../LICENSES_THIRD_PARTY.md).
- Effective date: 2026-03-14 timetable revision.

The same URL is what [`tool/fetch_gtfs`](../../tool/fetch_gtfs) downloads at
runtime; this vendored copy exists so CI runs offline against real data
without depending on a network call.

## Contents

| File | Notes |
| --- | --- |
| `agency.txt` | `Asia/Tokyo`, `agency_lang=ja` |
| `routes.txt` | 6 subway lines (浅草, 三田, 新宿, 大江戸, 日暮里・舎人, 都電荒川) |
| `stops.txt` | 149 stations, Japanese names, real lat/lon |
| `trips.txt` | ~5 600 trips |
| `stop_times.txt` | Includes 540 rows with blank arrival/departure (timepoint=0 passing stops) |
| `calendar.txt` / `calendar_dates.txt` | Weekday/weekend service patterns through 2026-12-31 |
| `fare_attributes.txt` / `fare_rules.txt` | Per-line fares (ignored by the current router) |
| `translations.txt` | English/Korean/Chinese station name translations (ignored by the current router) |
| `feed_info.txt` | Publisher metadata |

There is no `transfers.txt` or `shapes.txt` in this feed.

## Updating

```sh
go run ./tool/fetch_gtfs -feed toei-train -out assets/sample_toei_train
```

The zip is committed under git as a single binary file. Toei publishes a new
timetable roughly once per year, so re-vendoring is infrequent. When updating,
keep the file name `Toei-Train-GTFS.zip` so existing test paths continue to
resolve.
