# Third-party data attributions

This project bundles and/or downloads transit data published by external
agencies. Each dataset is governed by its own licence; the obligations below
must be honoured by any downstream redistribution (including app stores).

## Toei (都営) — Tokyo Metropolitan Bureau of Transportation

- **Datasets**
  - Static GTFS for Toei subway lines (浅草線, 三田線, 新宿線, 大江戸線,
    日暮里舎人ライナー, 都電荒川線) — vendored under
    [`assets/sample_toei_train/`](assets/sample_toei_train/) as a CI fixture.
  - Static GTFS for the Toei municipal bus network — downloaded on demand by
    [`tool/fetch_gtfs`](tool/fetch_gtfs) into `assets/real_gtfs/toei_bus/`
    (not vendored).
- **Source:** Public Transportation Open Data Center (公共交通オープンデータセンター),
  public-bucket mirror at <https://api-public.odpt.org/>.
- **Publisher:** 東京都交通局 (Tokyo Metropolitan Bureau of Transportation).
- **Licence:** [Creative Commons Attribution 4.0 International (CC-BY-4.0)](https://creativecommons.org/licenses/by/4.0/).
- **Required attribution string:**
  > Transit data © 東京都交通局 (Tokyo Metropolitan Bureau of Transportation),
  > CC-BY 4.0, via the Public Transportation Open Data Center (ODPT).

  This string must appear in the app's About / Credits screen whenever Toei
  data is loaded, and in any derivative redistribution.

No API key, registration, or commercial-use restriction applies to the
public-bucket endpoints. The author of the underlying data warrants neither
its accuracy nor its fitness for any particular purpose.

## Switzerland — opentransportdata.swiss / SKI+

- **Dataset:** Official Swiss nationwide GTFS static timetable for the 2026
  timetable year, downloaded on demand by [`tool/fetch_gtfs`](tool/fetch_gtfs)
  as `ch-aggregate-2026`.
- **Source:** <https://data.opentransportdata.swiss/dataset/timetable-2026-gtfs2020/permalink>
- **Publisher:** Systemaufgaben Kundeninformation SKI+ / opentransportdata.swiss.
- **Licence / terms:** <https://opentransportdata.swiss/en/terms-of-use/>
- **Required attribution string:**
  > Transit data © opentransportdata.swiss / Systemaufgaben Kundeninformation
  > SKI+, used under the opentransportdata.swiss terms of use.

The Swiss feed is the canonical countrywide source in this catalog. It is large
and should be preprocessed before app redistribution.

## Italy — regional and city GTFS feeds

The Italian catalog entries are official no-key regional or city feeds. There
is no single public nationwide GTFS equivalent to the Swiss feed in this repo.

- **Rome (`it-rome`)**
  - **Source:** <https://dati.comune.roma.it/catalog/dataset/a7dadb4a-66ae-4eff-8ded-a102064702ba>
  - **Publisher:** Roma Servizi per la Mobilità.
  - **Licence:** Creative Commons attribution/share-alike terms as declared by
    the official Rome open-data catalog.
- **Milan (`it-milan-atm`)**
  - **Source:** <https://dati.comune.milano.it/dataset/ds929-orari-del-trasporto-pubblico-locale-nel-comune-di-milano-in-formato-gtfs>
  - **Download mirror:** Mobility Database `mdb-2666`, because the official
    direct ZIP can be region-restricted.
  - **Publisher:** Comune di Milano / ATM / AMAT.
  - **Licence:** CC-BY-4.0.
- **Lombardy / Trenord (`it-lombardy-trenord`)**
  - **Source:** <https://www.dati.lombardia.it/d/3z4k-mxz9>
  - **Publisher:** Trenord.
  - **Licence:** CC-BY-4.0.
- **Tuscany (`it-tuscany-*`)**
  - **Source:** <https://dati.toscana.it/it/dataset/rt-oraritb>
  - **Publisher:** Regione Toscana and listed operators.
  - **Licence:** CC-BY-4.0 where declared by the current resource metadata.
- **Trentino (`it-trentino-*`)**
  - **Source:** <https://dati.trentino.it/dataset/trasporti-pubblici-del-trentino-formato-gtfs>
  - **Publisher:** Trentino Trasporti / Provincia autonoma di Trento.
  - **Licence:** CC-BY-4.0.

For complete no-key discovery beyond the curated app catalog, run
`tool/fetch_gtfs -country IT -complete` to read active GTFS entries from
Mobility Database. Treat the current official publisher page as authoritative
when licence metadata differs between portals and aggregators.

## Mobility Database complete-country mode

`tool/fetch_gtfs -country <CC> -complete` reads the active no-key GTFS rows from
Mobility Database and downloads the corresponding `latest.zip` mirror when
available. This is intended for fragmented countries such as Japan and Italy,
where a small curated app catalog cannot represent every municipal or operator
feed. Each generated `MANIFEST.json` records the publisher, source URL, licence
string, fetch time, and hash for downstream attribution review.
