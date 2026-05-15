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
- **Source:** Transitland feed records for Tokyo Toei GTFS, backed by Tokyo
  Metropolitan Bureau of Transportation data.
- **Publisher:** 東京都交通局 (Tokyo Metropolitan Bureau of Transportation).
- **Licence:** [Creative Commons Attribution 4.0 International (CC-BY-4.0)](https://creativecommons.org/licenses/by/4.0/).
- **Required attribution string:**
  > Transit data © 東京都交通局 (Tokyo Metropolitan Bureau of Transportation),
  > CC-BY 4.0; discovered through Transitland.

  This string must appear in the app's About / Credits screen whenever Toei
  data is loaded, and in any derivative redistribution.

The author of the underlying data warrants neither its accuracy nor its fitness
for any particular purpose.

## Switzerland — opentransportdata.swiss / SKI+

- **Dataset:** Official Swiss nationwide GTFS static timetable for the 2026
  timetable year, downloaded on demand by [`tool/fetch_gtfs`](tool/fetch_gtfs)
  as `ch-aggregate-2026`.
- **Source:** Transitland feed record `f-u0-switzerland`, backed by
  opentransportdata.swiss.
- **Publisher:** Systemaufgaben Kundeninformation SKI+ / opentransportdata.swiss.
- **Licence / terms:** <https://opentransportdata.swiss/en/terms-of-use/>
- **Required attribution string:**
  > Transit data © opentransportdata.swiss / Systemaufgaben Kundeninformation
  > SKI+, used under the opentransportdata.swiss terms of use; discovered
  > through Transitland.

The Swiss feed is the canonical countrywide source in this catalog. It is large
and should be preprocessed before app redistribution.

## Italy — regional and city GTFS feeds

The Italian catalog entries are Transitland-discovered regional or city feeds.
There is no single public nationwide GTFS equivalent to the Swiss feed in this
repo.

- **Rome (`it-rome`)**
  - **Source:** <https://dati.comune.roma.it/catalog/dataset/a7dadb4a-66ae-4eff-8ded-a102064702ba>
  - **Publisher:** Roma Servizi per la Mobilità.
  - **Licence:** Creative Commons attribution/share-alike terms as declared by
    the official Rome open-data catalog.
- **Milan (`it-milan-atm`)**
  - **Source:** Transitland feed record `f-u0nd-comunedimilano`, backed by
    Comune di Milano public transport GTFS.
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

Treat the current official publisher page as authoritative when licence
metadata differs between portals and aggregators.

## Transitland complete-country mode

`tool/fetch_gtfs -country <CC> -complete -complete-source transitland` reads
GTFS feeds from the Transitland REST API. Set `TRANSITLAND_API_KEY` in the
environment; the fetcher sends it as an `apikey` header and never stores it in
download URLs or `MANIFEST.json`.

Transitland discovery is currently bbox-based for `CH`, `IT`, and `JP`, and it
requests feeds with redistribution, derived-product, and commercial-use filters
that exclude feeds explicitly marked "no". Transitland's latest static GTFS
download endpoint can still reject a feed when redistribution is not allowed by
the source licence, so each generated manifest must still be reviewed against
the official publisher terms before redistribution.
