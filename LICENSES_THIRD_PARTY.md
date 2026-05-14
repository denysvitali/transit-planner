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
