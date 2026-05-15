package catalog

import "sort"

type FeedSpec struct {
	ID                   string
	Name                 string
	Description          string
	Country              string
	Region               string
	Publisher            string
	License              string
	SourceURL            string
	LocalFileName        string
	Attribution          string
	CenterLatitude       float64
	CenterLongitude      float64
	DefaultDepartureHour *int
	BundledAssetPath     string
}

type NetworkSpec struct {
	ID                   string
	Name                 string
	Description          string
	Publisher            string
	License              string
	SourceURL            string
	Attribution          string
	CenterLatitude       float64
	CenterLongitude      float64
	DefaultDepartureHour *int
	ComponentFeedIDs     []string
}

func hour(value int) *int { return &value }

var Feeds = map[string]FeedSpec{
	"ch-aggregate-2026": {
		ID: "ch-aggregate-2026", Name: "Switzerland national GTFS", Country: "CH", Region: "Nationwide",
		Description:   "Official nationwide Swiss GTFS static timetable for the current 2026 timetable year.",
		SourceURL:     "https://data.opentransportdata.swiss/dataset/timetable-2026-gtfs2020/permalink",
		LocalFileName: "ch-aggregate-2026.zip", Publisher: "Systemaufgaben Kundeninformation SKI+ / opentransportdata.swiss",
		License: "opentransportdata.swiss terms of use", CenterLatitude: 46.8182, CenterLongitude: 8.2275,
		Attribution: "Transit data © opentransportdata.swiss / Systemaufgaben Kundeninformation SKI+, used under the opentransportdata.swiss terms of use.",
	},
	"toei-bus": {
		ID: "toei-bus", Name: "Tokyo Toei Bus", Country: "JP", Region: "Tokyo",
		Description:   "Tokyo Metropolitan Bureau of Transportation municipal bus routes.",
		SourceURL:     "https://api-public.odpt.org/api/v4/files/Toei/data/ToeiBus-GTFS.zip",
		LocalFileName: "ToeiBus-GTFS.zip", Publisher: "Tokyo Metropolitan Bureau of Transportation (東京都交通局)",
		License: "CC-BY-4.0", CenterLatitude: 35.681236, CenterLongitude: 139.767125,
		Attribution: "Transit data © 東京都交通局 (Tokyo Metropolitan Bureau of Transportation), CC-BY 4.0, via the Public Transportation Open Data Center (ODPT).",
	},
	"toei-train": {
		ID: "toei-train", Name: "Tokyo Toei Subway", Country: "JP", Region: "Tokyo",
		Description:   "Toei subway lines (浅草線, 三田線, 新宿線, 大江戸線, 日暮里舎人ライナー, 都電荒川線).",
		SourceURL:     "https://api-public.odpt.org/api/v4/files/Toei/data/Toei-Train-GTFS.zip",
		LocalFileName: "Toei-Train-GTFS.zip", Publisher: "Tokyo Metropolitan Bureau of Transportation (東京都交通局)",
		License: "CC-BY-4.0", CenterLatitude: 35.681236, CenterLongitude: 139.767125,
		DefaultDepartureHour: hour(8), BundledAssetPath: "assets/sample_toei_train/Toei-Train-GTFS.zip",
		Attribution: "Transit data © 東京都交通局 (Tokyo Metropolitan Bureau of Transportation), CC-BY 4.0, via the Public Transportation Open Data Center (ODPT).",
	},
	"kanazawa-flatbus": {
		ID: "kanazawa-flatbus", Name: "Kanazawa Flat Bus", Country: "JP", Region: "Ishikawa",
		Description:   "Kanazawa city bus network, published as GTFS on the city open-data catalog.",
		SourceURL:     "https://catalog-data.city.kanazawa.ishikawa.jp/dataset/1196beb4-f9f9-463c-9723-5b38d8127425/resource/9636cac5-1449-4656-893b-ec98d834eb23/download/flatbus20260401.zip",
		LocalFileName: "flatbus20260401.zip", Publisher: "Kanazawa City, Ishikawa", License: "CC-BY-4.0",
		CenterLatitude: 36.5608, CenterLongitude: 136.6566,
		Attribution: "Transit data © Kanazawa City (Kanazawa-ken Jichitai), CC-BY 4.0.",
	},
	"kanazawa-hakusan-meguru": {
		ID: "kanazawa-hakusan-meguru", Name: "Hakusan Meguru", Country: "JP", Region: "Ishikawa",
		Description:   "Hakusan City Community Bus (\"Meguru\") network GTFS from the municipal open-data portal.",
		SourceURL:     "https://catalog-data.city.kanazawa.ishikawa.jp/dataset/89d93f28-38b4-4971-9988-2ff2d3227f56/resource/50049b19-fe9f-4ca1-9ea9-9d0a24141644/download/172103_bus.zip",
		LocalFileName: "172103_bus.zip", Publisher: "Hakusan City, Ishikawa", License: "CC-BY-4.0",
		CenterLatitude: 36.2581, CenterLongitude: 136.6290, Attribution: "Transit data © Hakusan City (白山市), CC-BY 4.0.",
	},
	"kanazawa-tsubata-bus": {
		ID: "kanazawa-tsubata-bus", Name: "Tsubata Town Bus", Country: "JP", Region: "Ishikawa",
		Description:   "Tsubata Town bus routes on the GSF/GTFS-JP package.",
		SourceURL:     "https://catalog-data.city.kanazawa.ishikawa.jp/dataset/8cd7f0dc-aab0-4bf4-a09d-c1d79faf4512/resource/9565f9b7-3bf7-4937-bee5-789d2aa4bf8a/download/gtfs-jp_tsubata.zip",
		LocalFileName: "gtfs-jp_tsubata.zip", Publisher: "Tsubata Town (Tsubata-chō), Ishikawa", License: "CC-BY-4.0",
		CenterLatitude: 36.7381, CenterLongitude: 136.5596,
		Attribution: "Transit data © Tsubata Town / Kanazawa public transport data, CC-BY 4.0.",
	},
	"kobe-shiokaze":              communityBus("kobe-shiokaze", "Kobe Shiokaze Bus", "Hyogo", "Kobe City (神戸市)", "CC-BY-2.1-JP", "https://api.gtfs-data.jp/v2/organizations/kobecity/feeds/kobe-shiokaze/files/feed.zip?rid=current", "Kobe Shiokaze community bus, via gtfs-data.jp", 34.6901, 135.1955),
	"kobe-satoyama":              communityBus("kobe-satoyama", "Kobe Satoyama Bus", "Hyogo", "Kobe City (神戸市)", "CC-BY-4.0", "https://api.gtfs-data.jp/v2/organizations/kobecity/feeds/kobe-satoyama/files/feed.zip?rid=current", "Kobe Satoyama community bus, via gtfs-data.jp", 34.6901, 135.1955),
	"himeji-ieshima":             communityBus("himeji-ieshima", "Himeji Ieshima Routes", "Hyogo", "Himeji City (姫路市)", "CC-BY-2.1-JP", "https://api.gtfs-data.jp/v2/organizations/himejicity/feeds/ieshima-boze-yukihiko/files/feed.zip?rid=current", "Himeji Ieshima / Boze / Yukihiko routes, via gtfs-data.jp", 34.8151, 134.6854),
	"takarazuka-runrunbus":       communityBus("takarazuka-runrunbus", "Takarazuka Runrun Bus", "Hyogo", "Takarazuka City (宝塚市)", "CC-BY-2.1-JP", "https://api.gtfs-data.jp/v2/organizations/takarazukacity/feeds/runrunbus/files/feed.zip?rid=current", "Takarazuka runrun community bus, via gtfs-data.jp", 34.8114, 135.3407),
	"nishinomiya-sakurayamanami": communityBus("nishinomiya-sakurayamanami", "Nishinomiya Sakurayamanami Bus", "Hyogo", "Nishinomiya City (西宮市)", "CC-BY-2.1-JP", "https://api.gtfs-data.jp/v2/organizations/nishinomiyacity/feeds/sakurayamanami/files/feed.zip?rid=current", "Nishinomiya Sakurayamanami community bus, via gtfs-data.jp", 34.7376, 135.3416),
	"yamatokoriyama-kingyobus":   communityBus("yamatokoriyama-kingyobus", "Yamatokoriyama Kingyo Bus", "Nara", "Yamatokoriyama City (大和郡山市)", "CC-BY-4.0", "https://api.gtfs-data.jp/v2/organizations/yamatokoriyamacity/feeds/kingyobus/files/feed.zip?rid=current", "Yamatokoriyama Kingyo community bus, via gtfs-data.jp", 34.6490, 135.7828),
	"rinkan-koyasan":             communityBus("rinkan-koyasan", "Nankai Rinkan Koyasan Bus", "Wakayama", "Nankai Rinkan Bus (南海りんかんバス)", "CC-BY-4.0", "https://api.gtfs-data.jp/v2/organizations/rinkan/feeds/koyasan/files/feed.zip?rid=current", "Mt. Koya / Koyasan bus network, via gtfs-data.jp", 34.2120, 135.5867),

	"jbda-kaetsunou-kaetsunouippan":                mobilityFeed("jbda-kaetsunou-kaetsunouippan", "Kaetsunou Bus", "Ishikawa Prefecture", "加越能バス", "CC0-1.0", "https://files.mobilitydatabase.org/jbda-kaetsunou-kaetsunouippan/latest.zip", "Kaetsunou general route bus network around Kanazawa, mirrored by Mobility Database.", 36.740017, 136.902600),
	"jbda-nonoichicity-communitybus":               mobilityFeed("jbda-nonoichicity-communitybus", "Nonoichi Community Bus", "Ishikawa Prefecture", "野々市市", "CC-BY-4.0", "https://files.mobilitydatabase.org/jbda-nonoichicity-communitybus/latest.zip", "Nonoichi community bus network, mirrored by Mobility Database.", 36.524932, 136.597140),
	"jbda-uchinadatown-communitybus":               mobilityFeed("jbda-uchinadatown-communitybus", "Uchinada Community Bus", "Ishikawa Prefecture", "内灘町", "CC-BY-4.0", "https://files.mobilitydatabase.org/jbda-uchinadatown-communitybus/latest.zip", "Uchinada community bus network, mirrored by Mobility Database.", 36.660712, 136.652230),
	"jbda-komatsucity-blue":                        mobilityFeed("jbda-komatsucity-blue", "Komatsu North Loop", "Ishikawa Prefecture", "小松市", "CC0-1.0", "https://files.mobilitydatabase.org/jbda-komatsucity-blue/latest.zip", "Komatsu city north loop bus, mirrored by Mobility Database.", 36.414092, 136.459254),
	"jbda-komatsucity-orange":                      mobilityFeed("jbda-komatsucity-orange", "Komatsu South Loop", "Ishikawa Prefecture", "小松市", "CC0-1.0", "https://files.mobilitydatabase.org/jbda-komatsucity-orange/latest.zip", "Komatsu city south loop bus, mirrored by Mobility Database.", 36.395266, 136.470707),
	"jbda-komatsucity-kibagatasen":                 mobilityFeed("jbda-komatsucity-kibagatasen", "Komatsu Kibagata Line", "Ishikawa Prefecture", "小松市", "CC0-1.0", "https://files.mobilitydatabase.org/jbda-komatsucity-kibagatasen/latest.zip", "Komatsu Kibagata bus line, mirrored by Mobility Database.", 36.373056, 136.458775),
	"jbda-chitetsu-chitetsubus":                    mobilityFeed("jbda-chitetsu-chitetsubus", "Toyama Chitetsu Bus", "Toyama Prefecture", "富山地方鉄道", "CC0-1.0", "https://files.mobilitydatabase.org/jbda-chitetsu-chitetsubus/latest.zip", "Toyama Chitetsu bus network, mirrored by Mobility Database.", 36.733815, 137.248690),
	"jbda-chitetsu-chitetsushinaidensha":           mobilityFeed("jbda-chitetsu-chitetsushinaidensha", "Toyama Chitetsu City Tram", "Toyama Prefecture", "富山地方鉄道", "CC0-1.0", "https://files.mobilitydatabase.org/jbda-chitetsu-chitetsushinaidensha/latest.zip", "Toyama Chitetsu city tram network, mirrored by Mobility Database.", 36.715942, 137.212628),
	"jbda-manyosen-manyosen":                       mobilityFeed("jbda-manyosen-manyosen", "Manyosen Tram", "Toyama Prefecture", "万葉線", "CC0-1.0", "https://files.mobilitydatabase.org/jbda-manyosen-manyosen/latest.zip", "Manyosen tram network, mirrored by Mobility Database.", 36.765312, 137.062410),
	"jbda-akashicity-tacobustacobusmini":           mobilityFeed("jbda-akashicity-tacobustacobusmini", "Akashi Taco Bus", "Hyogo Prefecture", "明石市", "CC-BY-4.0", "https://files.mobilitydatabase.org/jbda-akashicity-tacobustacobusmini/latest.zip", "Akashi Taco Bus and Taco Bus Mini, mirrored by Mobility Database.", 34.689950, 134.923916),
	"jbda-kakogawacity-kakobuskakobusmini":         mobilityFeed("jbda-kakogawacity-kakobuskakobusmini", "Kakogawa Kako Bus", "Hyogo Prefecture", "加古川市", "CC-BY-2.1-JP", "https://files.mobilitydatabase.org/jbda-kakogawacity-kakobuskakobusmini/latest.zip", "Kakogawa Kako Bus and Kako Bus Mini, mirrored by Mobility Database.", 34.779365, 134.845272),
	"jbda-takasagocity-jotonbus":                   mobilityFeed("jbda-takasagocity-jotonbus", "Takasago Joton Bus", "Hyogo Prefecture", "高砂市", "CC-BY-2.1-JP", "https://files.mobilitydatabase.org/jbda-takasagocity-jotonbus/latest.zip", "Takasago Joton Bus, mirrored by Mobility Database.", 34.779860, 134.792922),
	"jbda-nishinomiyacity-guruttonamaze":           mobilityFeed("jbda-nishinomiyacity-guruttonamaze", "Nishinomiya Gurutto Namaze", "Hyogo Prefecture", "西宮市", "CC-BY-2.1-JP", "https://files.mobilitydatabase.org/jbda-nishinomiyacity-guruttonamaze/latest.zip", "Nishinomiya Gurutto Namaze community bus, mirrored by Mobility Database.", 34.819407, 135.328093),
	"jbda-nishinomiyacity-koyoen":                  mobilityFeed("jbda-nishinomiyacity-koyoen", "Nishinomiya Koyoen Bus", "Hyogo Prefecture", "西宮市", "CC-BY-4.0", "https://files.mobilitydatabase.org/jbda-nishinomiyacity-koyoen/latest.zip", "Nishinomiya Koyoen community bus, mirrored by Mobility Database.", 34.766085, 135.327792),
	"jbda-andotown-andocombus":                     mobilityFeed("jbda-andotown-andocombus", "Ando Community Bus", "Nara Prefecture", "安堵町", "CC-BY-4.0", "https://files.mobilitydatabase.org/jbda-andotown-andocombus/latest.zip", "Ando community bus, mirrored by Mobility Database.", 34.600310, 135.761048),
	"jbda-nabaricity-communitybus":                 mobilityFeed("jbda-nabaricity-communitybus", "Nabari Community Bus", "Nara Prefecture", "名張市", "CC-BY-4.0", "https://files.mobilitydatabase.org/jbda-nabaricity-communitybus/latest.zip", "Nabari community bus, mirrored by Mobility Database.", 34.622912, 136.115424),
	"jbda-yamatotakadacity-communitybuskibougou":   mobilityFeed("jbda-yamatotakadacity-communitybuskibougou", "Yamatotakada Kibou Bus", "Nara Prefecture", "大和高田市", "CC-BY-4.0", "https://files.mobilitydatabase.org/jbda-yamatotakadacity-communitybuskibougou/latest.zip", "Yamatotakada Kibou community bus, mirrored by Mobility Database.", 34.502817, 135.742643),
	"jbda-higashiomicity-higasiohmisicommunitybus": mobilityFeed("jbda-higashiomicity-higasiohmisicommunitybus", "Higashiomi Chokotto Bus", "Shiga Prefecture", "東近江市", "CC-BY-4.0", "https://files.mobilitydatabase.org/jbda-higashiomicity-higasiohmisicommunitybus/latest.zip", "Higashiomi Chokotto Bus, mirrored by Mobility Database.", 35.129049, 136.248841),
	"jbda-omihachimancity-akakonbus":               mobilityFeed("jbda-omihachimancity-akakonbus", "Omihachiman Akakon Bus", "Shiga Prefecture", "近江八幡市", "CC-BY-4.0", "https://files.mobilitydatabase.org/jbda-omihachimancity-akakonbus/latest.zip", "Omihachiman Akakon Bus, mirrored by Mobility Database.", 35.135921, 136.096512),

	"it-lombardy-trenord":         italyFeed("it-lombardy-trenord", "Lombardy Trenord rail", "Lombardy", "Trenord", "CC-BY-4.0", "https://www.dati.lombardia.it/download/3z4k-mxz9/application%2Fzip", "Regional rail GTFS for Lombardy / Trenord services.", 45.4642, 9.19),
	"it-milan-atm":                italyFeed("it-milan-atm", "Milan urban GTFS", "Lombardy", "Comune di Milano / ATM / AMAT", "CC-BY-4.0", "https://files.mobilitydatabase.org/mdb-2666/latest.zip", "Milan urban public transport GTFS mirrored by Mobility Database because the official direct file can be region-restricted.", 45.4642, 9.19),
	"it-rome":                     italyFeed("it-rome", "Rome public transport GTFS", "Lazio", "Roma Servizi per la Mobilità", "CC-BY-SA", "https://dati.comune.roma.it/catalog/dataset/a7dadb4a-66ae-4eff-8ded-a102064702ba/resource/266d82e1-ba53-4510-8a81-370880c4678f/download/rome_static_gtfs.zip", "Daily static GTFS for Rome public transport.", 41.9028, 12.4964),
	"it-trentino-extraurban":      italyFeed("it-trentino-extraurban", "Trentino extraurban GTFS", "Trentino-Alto Adige", "Trentino Trasporti", "CC-BY-4.0", "https://www.trentinotrasporti.it/opendata/google_transit_extraurbano_tte.zip", "Extraurban GTFS for Trentino public transport.", 46.0667, 11.1211),
	"it-trentino-urban":           italyFeed("it-trentino-urban", "Trentino urban GTFS", "Trentino-Alto Adige", "Trentino Trasporti", "CC-BY-4.0", "https://www.trentinotrasporti.it/opendata/google_transit_urbano_tte.zip", "Urban GTFS for Trentino public transport.", 46.0667, 11.1211),
	"it-tuscany-autolinee":        italyFeed("it-tuscany-autolinee", "Tuscany Autolinee Toscane", "Tuscany", "Regione Toscana / Autolinee Toscane", "CC-BY-4.0", "https://regionetoscana.smartregion.toscana.it/mobility/artifacts/gtfs", "Autolinee Toscane regional bus GTFS from the official Tuscany mobility endpoint.", 43.7711, 11.2486),
	"it-tuscany-colbus-nonschool": italyFeed("it-tuscany-colbus-nonschool", "Tuscany Colbus non-school", "Tuscany", "Regione Toscana / Colbus", "CC-BY-4.0", "https://dati.toscana.it/dataset/8bb8f8fe-fe7d-41d0-90dc-49f2456180d1/resource/61fada72-e2de-4dee-aa23-66629152fa0d/download/02-colbusnonscolastico.gtfs", "Colbus non-school bus GTFS in the Florence metropolitan area.", 43.7711, 11.2486),
	"it-tuscany-colbus-school":    italyFeed("it-tuscany-colbus-school", "Tuscany Colbus school", "Tuscany", "Regione Toscana / Colbus", "CC-BY-4.0", "https://dati.toscana.it/dataset/8bb8f8fe-fe7d-41d0-90dc-49f2456180d1/resource/5fb6d2bd-8146-456a-91fe-23009ffae253/download/01-colbusscolastico.gtfs", "Colbus school bus GTFS in the Florence metropolitan area.", 43.7711, 11.2486),
	"it-tuscany-gest":             italyFeed("it-tuscany-gest", "Florence tram GTFS", "Tuscany", "Regione Toscana / GEST", "CC-BY-4.0", "https://dati.toscana.it/dataset/8bb8f8fe-fe7d-41d0-90dc-49f2456180d1/resource/1f62d551-65f4-49f8-9a99-e19b02077be3/download/gest.gtfs", "Florence tram GTFS from Regione Toscana.", 43.7711, 11.2486),
	"it-tuscany-tft":              italyFeed("it-tuscany-tft", "Tuscany TFT rail GTFS", "Tuscany", "Regione Toscana / Trasporto Ferroviario Toscano", "CC-BY-4.0", "https://dati.toscana.it/dataset/8bb8f8fe-fe7d-41d0-90dc-49f2456180d1/resource/59aeacbc-99ee-410b-ac1e-622b5574a666/download/tft.gtfs", "Trasporto Ferroviario Toscano rail GTFS from Regione Toscana.", 43.4633, 11.8796),
	"it-tuscany-toremar":          italyFeed("it-tuscany-toremar", "Tuscany Toremar ferries GTFS", "Tuscany", "Regione Toscana / Toremar", "CC-BY-4.0", "https://dati.toscana.it/dataset/8bb8f8fe-fe7d-41d0-90dc-49f2456180d1/resource/56539a5a-e0be-49eb-b3ac-052a42ad0de0/download/toremar.gtfs", "Toremar ferry GTFS for Tuscany coastal and island services.", 42.8129, 10.3167),
	"it-tuscany-trenitalia":       italyFeed("it-tuscany-trenitalia", "Tuscany Trenitalia regional rail", "Tuscany", "Regione Toscana / Trenitalia", "CC-BY-4.0", "https://dati.toscana.it/dataset/8bb8f8fe-fe7d-41d0-90dc-49f2456180d1/resource/4f85393b-357d-443d-8378-65de4198505f/download/trenitalia.gtfs", "Trenitalia regional rail GTFS for Tuscany.", 43.7711, 11.2486),
	"it-tuscany-at-nonschool":     italyFeed("it-tuscany-at-nonschool", "Tuscany Autolinee non-school", "Tuscany", "Regione Toscana / Autolinee Toscane", "CC-BY-4.0", "https://dati.toscana.it/dataset/8bb8f8fe-fe7d-41d0-90dc-49f2456180d1/resource/6969571a-96d7-490d-a944-17af386717b6/download/04-atnonscolastico.gtfs", "Autolinee Toscane non-school bus GTFS in the Florence metropolitan area.", 43.7711, 11.2486),
	"it-tuscany-at-school":        italyFeed("it-tuscany-at-school", "Tuscany Autolinee school", "Tuscany", "Regione Toscana / Autolinee Toscane", "CC-BY-4.0", "https://dati.toscana.it/dataset/8bb8f8fe-fe7d-41d0-90dc-49f2456180d1/resource/8b38e763-e349-404c-a274-442312f7e3b2/download/03-atscolastico.gtfs", "Autolinee Toscane school bus GTFS in the Florence metropolitan area.", 43.7711, 11.2486),
}

func communityBus(id, name, region, publisher, license, url, description string, lat, lon float64) FeedSpec {
	return FeedSpec{
		ID: id, Name: name, Country: "JP", Region: region, Publisher: publisher, License: license,
		SourceURL: url, LocalFileName: id + ".zip", Description: description,
		CenterLatitude: lat, CenterLongitude: lon,
		Attribution: "Transit data © " + publisher + ", " + license + ".",
	}
}

func mobilityFeed(id, name, region, publisher, license, url, description string, lat, lon float64) FeedSpec {
	return FeedSpec{
		ID: id, Name: name, Country: "JP", Region: region, Publisher: publisher, License: license,
		SourceURL: url, LocalFileName: id + ".zip", Description: description,
		CenterLatitude: lat, CenterLongitude: lon,
		Attribution: "Transit data © " + publisher + ", " + license + "; mirrored by Mobility Database.",
	}
}

func italyFeed(id, name, region, publisher, license, url, description string, lat, lon float64) FeedSpec {
	return FeedSpec{
		ID: id, Name: name, Country: "IT", Region: region, Publisher: publisher, License: license,
		SourceURL: url, LocalFileName: id + ".zip", Description: description,
		CenterLatitude: lat, CenterLongitude: lon,
		Attribution: "Transit data © " + publisher + ", " + license + ".",
	}
}

var hokurikuPublicFeedIDs = []string{
	"kanazawa-flatbus",
	"kanazawa-hakusan-meguru",
	"kanazawa-tsubata-bus",
	"jbda-kaetsunou-kaetsunouippan",
	"jbda-nonoichicity-communitybus",
	"jbda-uchinadatown-communitybus",
	"jbda-komatsucity-blue",
	"jbda-komatsucity-orange",
	"jbda-komatsucity-kibagatasen",
	"jbda-chitetsu-chitetsubus",
	"jbda-chitetsu-chitetsushinaidensha",
	"jbda-manyosen-manyosen",
}

var kansaiPublicFeedIDs = []string{
	"kobe-shiokaze",
	"kobe-satoyama",
	"himeji-ieshima",
	"takarazuka-runrunbus",
	"nishinomiya-sakurayamanami",
	"yamatokoriyama-kingyobus",
	"rinkan-koyasan",
	"jbda-akashicity-tacobustacobusmini",
	"jbda-kakogawacity-kakobuskakobusmini",
	"jbda-takasagocity-jotonbus",
	"jbda-nishinomiyacity-guruttonamaze",
	"jbda-nishinomiyacity-koyoen",
	"jbda-andotown-andocombus",
	"jbda-nabaricity-communitybus",
	"jbda-yamatotakadacity-communitybuskibougou",
	"jbda-higashiomicity-higasiohmisicommunitybus",
	"jbda-omihachimancity-akakonbus",
}

var japanPublicFeedIDs = append(append([]string{
	"toei-train",
	"toei-bus",
}, hokurikuPublicFeedIDs...), kansaiPublicFeedIDs...)

var tuscanyPublicFeedIDs = []string{
	"it-tuscany-autolinee",
	"it-tuscany-trenitalia",
	"it-tuscany-tft",
	"it-tuscany-toremar",
	"it-tuscany-gest",
	"it-tuscany-colbus-school",
	"it-tuscany-colbus-nonschool",
	"it-tuscany-at-school",
	"it-tuscany-at-nonschool",
}

var trentinoPublicFeedIDs = []string{
	"it-trentino-urban",
	"it-trentino-extraurban",
}

var italyPublicFeedIDs = append(append([]string{
	"it-rome",
	"it-milan-atm",
	"it-lombardy-trenord",
}, tuscanyPublicFeedIDs...), trentinoPublicFeedIDs...)

var transitlandCoverageFeedIDs = append(append([]string{
	"ch-aggregate-2026",
}, italyPublicFeedIDs...), japanPublicFeedIDs...)

var Networks = []NetworkSpec{
	{
		ID: "transitland-coverage", Name: "Transitland coverage",
		Description: "Single app network for Transitland-sourced coverage across Japan, Switzerland, and Italy. Feed discovery runs in tooling/CI so the app never stores a Transitland API key.",
		Publisher:   "Transitland and source transit-data publishers", License: "Mixed source licences", SourceURL: "https://transit.land/api/v2/rest/feeds",
		Attribution:    "Transit data discovered through Transitland; licences vary by publisher, so show each component feed attribution where relevant.",
		CenterLatitude: 42.5, CenterLongitude: 12.5,
		ComponentFeedIDs: transitlandCoverageFeedIDs,
	},
	{
		ID: "ch-national", Name: "Switzerland - national GTFS",
		Description: "Official nationwide Swiss static GTFS timetable, suitable as the canonical country source after preprocessing.",
		Publisher:   Feeds["ch-aggregate-2026"].Publisher, License: Feeds["ch-aggregate-2026"].License, SourceURL: Feeds["ch-aggregate-2026"].SourceURL,
		Attribution: Feeds["ch-aggregate-2026"].Attribution, CenterLatitude: 46.8182, CenterLongitude: 8.2275,
		ComponentFeedIDs: []string{"ch-aggregate-2026"},
	},
	{
		ID: "it-public-regional", Name: "Italy - regional and city GTFS",
		Description: "Official no-key Italian regional and city GTFS feeds currently known to the app: Rome, Milan, Lombardy rail, Tuscany, and Trentino.",
		Publisher:   "Multiple Italian public-data publishers", License: "Mixed open-data licences", SourceURL: "Multiple GTFS endpoints",
		Attribution:    "Merged network of the Italian feeds listed below. Licences vary by publisher; see each feed attribution.",
		CenterLatitude: 42.5, CenterLongitude: 12.5,
		ComponentFeedIDs: italyPublicFeedIDs,
	},
	{
		ID: "it-tuscany-public", Name: "Tuscany - regional GTFS",
		Description: "Regione Toscana multimodal GTFS resources for rail, ferries, tram, and bus.",
		Publisher:   "Regione Toscana", License: "CC-BY-4.0", SourceURL: "Multiple Regione Toscana GTFS endpoints",
		Attribution:    "Merged network of official Regione Toscana GTFS resources; see each feed attribution below.",
		CenterLatitude: 43.7711, CenterLongitude: 11.2486,
		ComponentFeedIDs: tuscanyPublicFeedIDs,
	},
	{
		ID: "it-trentino-public", Name: "Trentino - public transport GTFS",
		Description: "Trentino urban and extraurban GTFS resources.",
		Publisher:   "Trentino Trasporti", License: "CC-BY-4.0", SourceURL: "Multiple Trentino Trasporti GTFS endpoints",
		Attribution:    "Merged network of Trentino urban and extraurban GTFS resources; see each feed attribution below.",
		CenterLatitude: 46.0667, CenterLongitude: 11.1211,
		ComponentFeedIDs: trentinoPublicFeedIDs,
	},
	{
		ID: "jp-public-no-key", Name: "Japan - available public feeds",
		Description: "All no-key Japanese GTFS feeds currently known to the app, merged into one local routing network.",
		Publisher:   "Multiple public GTFS publishers", License: "Mixed open-data licences", SourceURL: "Multiple GTFS endpoints",
		Attribution:    "Merged network of the Japanese feeds listed below. Licences vary by publisher; see each feed attribution.",
		CenterLatitude: 35.681236, CenterLongitude: 139.767125, DefaultDepartureHour: hour(8),
		ComponentFeedIDs: japanPublicFeedIDs,
	},
	{
		ID: "tokyo-toei", Name: "Tokyo Toei network",
		Description: "Tokyo Metropolitan Bureau of Transportation subway, tram, liner, and municipal bus feeds merged into one network.",
		Publisher:   "Tokyo Metropolitan Bureau of Transportation (東京都交通局)", License: "CC-BY-4.0", SourceURL: "Multiple ODPT GTFS endpoints",
		Attribution: Feeds["toei-train"].Attribution, CenterLatitude: 35.681236, CenterLongitude: 139.767125,
		DefaultDepartureHour: hour(8), ComponentFeedIDs: []string{"toei-train", "toei-bus"},
	},
	{
		ID: "kanazawa-region", Name: "Kanazawa region",
		Description: "Kanazawa, Ishikawa, and nearby Hokuriku public bus and tram feeds merged into one regional network.",
		Publisher:   "Multiple Hokuriku public-data publishers", License: "Mixed open-data licences", SourceURL: "Multiple GTFS endpoints",
		Attribution:    "Merged network of Kanazawa, Ishikawa, and nearby Hokuriku public GTFS feeds; see each feed attribution below.",
		CenterLatitude: 36.5608, CenterLongitude: 136.6566,
		ComponentFeedIDs: hokurikuPublicFeedIDs,
	},
	{
		ID: "kansai-public-no-key", Name: "Kansai - available public feeds",
		Description: "Public no-key Hyogo, Nara, Shiga, and Wakayama feeds. This does not include major JR or private rail.",
		Publisher:   "Multiple public GTFS publishers", License: "Mixed open-data licences", SourceURL: "Multiple GTFS endpoints",
		Attribution:    "Merged network of the Kansai-area no-key public feeds listed below. Major rail is not included.",
		CenterLatitude: 34.6937, CenterLongitude: 135.5023,
		ComponentFeedIDs: kansaiPublicFeedIDs,
	},
}

func SortedFeeds() []FeedSpec {
	out := make([]FeedSpec, 0, len(Feeds))
	for _, feed := range Feeds {
		out = append(out, feed)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Country != out[j].Country {
			return out[i].Country < out[j].Country
		}
		return out[i].ID < out[j].ID
	})
	return out
}
