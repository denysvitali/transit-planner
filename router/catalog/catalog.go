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
	Country              string
	Region               string
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

func transitlandFeedURL(feedKey string) string {
	return "https://transit.land/api/v2/rest/feeds/" + feedKey + "/download_latest_feed_version"
}

var Feeds = map[string]FeedSpec{
	"ch-aggregate-2026": {
		ID: "ch-aggregate-2026", Name: "Switzerland national GTFS", Country: "CH", Region: "Nationwide",
		Description:   "Official nationwide Swiss GTFS static timetable for the current 2026 timetable year.",
		SourceURL:     transitlandFeedURL("f-u0-switzerland"),
		LocalFileName: "ch-aggregate-2026.zip", Publisher: "Systemaufgaben Kundeninformation SKI+ / opentransportdata.swiss",
		License: "opentransportdata.swiss terms of use", CenterLatitude: 46.8182, CenterLongitude: 8.2275,
		Attribution: "Transit data © opentransportdata.swiss / Systemaufgaben Kundeninformation SKI+, used under the opentransportdata.swiss terms of use; discovered through Transitland.",
	},
	"toei-bus": {
		ID: "toei-bus", Name: "Tokyo Toei Bus", Country: "JP", Region: "Tokyo",
		Description:   "Tokyo Metropolitan Bureau of Transportation municipal bus routes.",
		SourceURL:     transitlandFeedURL("f-toeibus~gtfs~jp"),
		LocalFileName: "ToeiBus-GTFS.zip", Publisher: "Tokyo Metropolitan Bureau of Transportation (東京都交通局)",
		License: "CC-BY-4.0", CenterLatitude: 35.681236, CenterLongitude: 139.767125,
		Attribution: "Transit data © 東京都交通局 (Tokyo Metropolitan Bureau of Transportation), CC-BY 4.0; discovered through Transitland.",
	},
	"toei-train": {
		ID: "toei-train", Name: "Tokyo Toei Subway", Country: "JP", Region: "Tokyo",
		Description:   "Toei subway lines (浅草線, 三田線, 新宿線, 大江戸線, 日暮里舎人ライナー, 都電荒川線).",
		SourceURL:     transitlandFeedURL("f-toei~data~toei~train~gtfs~jp"),
		LocalFileName: "Toei-Train-GTFS.zip", Publisher: "Tokyo Metropolitan Bureau of Transportation (東京都交通局)",
		License: "CC-BY-4.0", CenterLatitude: 35.681236, CenterLongitude: 139.767125,
		DefaultDepartureHour: hour(8), BundledAssetPath: "assets/sample_toei_train/Toei-Train-GTFS.zip",
		Attribution: "Transit data © 東京都交通局 (Tokyo Metropolitan Bureau of Transportation), CC-BY 4.0; discovered through Transitland.",
	},
	"kanazawa-flatbus": {
		ID: "kanazawa-flatbus", Name: "Kanazawa Flat Bus", Country: "JP", Region: "Ishikawa",
		Description:   "Kanazawa city bus network GTFS discovered through Transitland.",
		SourceURL:     transitlandFeedURL("f-金沢ふらっとバス"),
		LocalFileName: "flatbus20260401.zip", Publisher: "Kanazawa City, Ishikawa", License: "CC-BY-4.0",
		CenterLatitude: 36.5608, CenterLongitude: 136.6566,
		Attribution: "Transit data © Kanazawa City (Kanazawa-ken Jichitai), CC-BY 4.0; discovered through Transitland.",
	},
	"kanazawa-hakusan-meguru": {
		ID: "kanazawa-hakusan-meguru", Name: "Hakusan Meguru", Country: "JP", Region: "Ishikawa",
		Description:   "Hakusan City Community Bus (\"Meguru\") network GTFS discovered through Transitland.",
		SourceURL:     transitlandFeedURL("f-白山市コミュニティバスめぐーる"),
		LocalFileName: "172103_bus.zip", Publisher: "Hakusan City, Ishikawa", License: "CC-BY-4.0",
		CenterLatitude: 36.2581, CenterLongitude: 136.6290, Attribution: "Transit data © Hakusan City (白山市), CC-BY 4.0; discovered through Transitland.",
	},
	"kanazawa-tsubata-bus": {
		ID: "kanazawa-tsubata-bus", Name: "Tsubata Town Bus", Country: "JP", Region: "Ishikawa",
		Description:   "Tsubata Town bus routes discovered through Transitland.",
		SourceURL:     transitlandFeedURL("f-津幡町営バス"),
		LocalFileName: "gtfs-jp_tsubata.zip", Publisher: "Tsubata Town (Tsubata-chō), Ishikawa", License: "CC-BY-4.0",
		CenterLatitude: 36.7381, CenterLongitude: 136.5596,
		Attribution: "Transit data © Tsubata Town / Kanazawa public transport data, CC-BY 4.0; discovered through Transitland.",
	},
	"kobe-shiokaze":              communityBus("kobe-shiokaze", "Kobe Shiokaze Bus", "Hyogo", "Kobe City (神戸市)", "CC-BY-2.1-JP", "f-神戸市垂水区~塩屋コミュニティバスしおかぜ", "Kobe Shiokaze community bus discovered through Transitland.", 34.6901, 135.1955),
	"himeji-ieshima":             communityBus("himeji-ieshima", "Himeji Ieshima Routes", "Hyogo", "Himeji City (姫路市)", "CC-BY-2.1-JP", "f-姫路市コミュニティバス", "Himeji Ieshima / Boze / Yukihiko routes discovered through Transitland.", 34.8151, 134.6854),
	"takarazuka-runrunbus":       communityBus("takarazuka-runrunbus", "Takarazuka Runrun Bus", "Hyogo", "Takarazuka City (宝塚市)", "CC-BY-2.1-JP", "f-宝塚市~ランランバス", "Takarazuka runrun community bus discovered through Transitland.", 34.8114, 135.3407),
	"nishinomiya-sakurayamanami": communityBus("nishinomiya-sakurayamanami", "Nishinomiya Sakurayamanami Bus", "Hyogo", "Nishinomiya City (西宮市)", "CC-BY-2.1-JP", "f-西宮市~さくらやまなみバス", "Nishinomiya Sakurayamanami community bus discovered through Transitland.", 34.7376, 135.3416),
	"yamatokoriyama-kingyobus":   communityBus("yamatokoriyama-kingyobus", "Yamatokoriyama Kingyo Bus", "Nara", "Yamatokoriyama City (大和郡山市)", "CC-BY-4.0", "f-yamatokoriyamacity~kingyobus", "Yamatokoriyama Kingyo community bus discovered through Transitland.", 34.6490, 135.7828),
	"rinkan-koyasan":             communityBus("rinkan-koyasan", "Nankai Rinkan Koyasan Bus", "Wakayama", "Nankai Rinkan Bus (南海りんかんバス)", "CC-BY-4.0", "f-rinkan~koyasan", "Mt. Koya / Koyasan bus network discovered through Transitland.", 34.2120, 135.5867),

	"jbda-kaetsunou-kaetsunouippan":                mobilityFeed("jbda-kaetsunou-kaetsunouippan", "Kaetsunou Bus", "Ishikawa Prefecture", "加越能バス", "CC0-1.0", "f-加越能バス~一般路線", "Kaetsunou general route bus network around Kanazawa discovered through Transitland.", 36.740017, 136.902600),
	"jbda-nonoichicity-communitybus":               mobilityFeed("jbda-nonoichicity-communitybus", "Nonoichi Community Bus", "Ishikawa Prefecture", "野々市市", "CC-BY-4.0", "f-nonoichicity~communitybus", "Nonoichi community bus network discovered through Transitland.", 36.524932, 136.597140),
	"jbda-komatsucity-blue":                        mobilityFeed("jbda-komatsucity-blue", "Komatsu North Loop", "Ishikawa Prefecture", "小松市", "CC0-1.0", "f-komatsucity~blue", "Komatsu city north loop bus discovered through Transitland.", 36.414092, 136.459254),
	"jbda-komatsucity-orange":                      mobilityFeed("jbda-komatsucity-orange", "Komatsu South Loop", "Ishikawa Prefecture", "小松市", "CC0-1.0", "f-komatsucity~orange", "Komatsu city south loop bus discovered through Transitland.", 36.395266, 136.470707),
	"jbda-komatsucity-kibagatasen":                 mobilityFeed("jbda-komatsucity-kibagatasen", "Komatsu Kibagata Line", "Ishikawa Prefecture", "小松市", "CC0-1.0", "f-komatsucity~kibagatasen", "Komatsu Kibagata bus line discovered through Transitland.", 36.373056, 136.458775),
	"jbda-chitetsu-chitetsubus":                    mobilityFeed("jbda-chitetsu-chitetsubus", "Toyama Chitetsu Bus", "Toyama Prefecture", "富山地方鉄道", "CC0-1.0", "f-富山地鉄バス富山地方鉄道", "Toyama Chitetsu bus network discovered through Transitland.", 36.733815, 137.248690),
	"jbda-chitetsu-chitetsushinaidensha":           mobilityFeed("jbda-chitetsu-chitetsushinaidensha", "Toyama Chitetsu City Tram", "Toyama Prefecture", "富山地方鉄道", "CC0-1.0", "f-富山地鉄市内電車", "Toyama Chitetsu city tram network discovered through Transitland.", 36.715942, 137.212628),
	"jbda-manyosen-manyosen":                       mobilityFeed("jbda-manyosen-manyosen", "Manyosen Tram", "Toyama Prefecture", "万葉線", "CC0-1.0", "f-万葉線", "Manyosen tram network discovered through Transitland.", 36.765312, 137.062410),
	"jbda-akashicity-tacobustacobusmini":           mobilityFeed("jbda-akashicity-tacobustacobusmini", "Akashi Taco Bus", "Hyogo Prefecture", "明石市", "CC-BY-4.0", "f-明石市~たこバス", "Akashi Taco Bus and Taco Bus Mini discovered through Transitland.", 34.689950, 134.923916),
	"jbda-kakogawacity-kakobuskakobusmini":         mobilityFeed("jbda-kakogawacity-kakobuskakobusmini", "Kakogawa Kako Bus", "Hyogo Prefecture", "加古川市", "CC-BY-2.1-JP", "f-加古川市~かこバス~かこバスミニ", "Kakogawa Kako Bus and Kako Bus Mini discovered through Transitland.", 34.779365, 134.845272),
	"jbda-takasagocity-jotonbus":                   mobilityFeed("jbda-takasagocity-jotonbus", "Takasago Joton Bus", "Hyogo Prefecture", "高砂市", "CC-BY-2.1-JP", "f-高砂市~じょうとんバス", "Takasago Joton Bus discovered through Transitland.", 34.779860, 134.792922),
	"jbda-nishinomiyacity-guruttonamaze":           mobilityFeed("jbda-nishinomiyacity-guruttonamaze", "Nishinomiya Gurutto Namaze", "Hyogo Prefecture", "西宮市", "CC-BY-2.1-JP", "f-西宮市コミュニティ交通ぐるっと生瀬", "Nishinomiya Gurutto Namaze community bus discovered through Transitland.", 34.819407, 135.328093),
	"jbda-andotown-andocombus":                     mobilityFeed("jbda-andotown-andocombus", "Ando Community Bus", "Nara Prefecture", "安堵町", "CC-BY-4.0", "f-andotown~andocombus", "Ando community bus discovered through Transitland.", 34.600310, 135.761048),
	"jbda-nabaricity-communitybus":                 mobilityFeed("jbda-nabaricity-communitybus", "Nabari Community Bus", "Nara Prefecture", "名張市", "CC-BY-4.0", "f-名張市コミュニティバス", "Nabari community bus discovered through Transitland.", 34.622912, 136.115424),
	"jbda-yamatotakadacity-communitybuskibougou":   mobilityFeed("jbda-yamatotakadacity-communitybuskibougou", "Yamatotakada Kibou Bus", "Nara Prefecture", "大和高田市", "CC-BY-4.0", "f-yamatotakadacity~communitybuskibougou", "Yamatotakada Kibou community bus discovered through Transitland.", 34.502817, 135.742643),
	"jbda-higashiomicity-higasiohmisicommunitybus": mobilityFeed("jbda-higashiomicity-higasiohmisicommunitybus", "Higashiomi Chokotto Bus", "Shiga Prefecture", "東近江市", "CC-BY-4.0", "f-higashiomicity~higasiohmisicommunitybus", "Higashiomi Chokotto Bus discovered through Transitland.", 35.129049, 136.248841),
	"jbda-omihachimancity-akakonbus":               mobilityFeed("jbda-omihachimancity-akakonbus", "Omihachiman Akakon Bus", "Shiga Prefecture", "近江八幡市", "CC-BY-4.0", "f-omihachimancity~akakonbus", "Omihachiman Akakon Bus discovered through Transitland.", 35.135921, 136.096512),

	"it-lombardy-trenord":         italyFeed("it-lombardy-trenord", "Lombardy Trenord rail", "Lombardy", "Trenord", "CC-BY-4.0", "f-u0n-trenord", "Regional rail GTFS for Lombardy / Trenord services discovered through Transitland.", 45.4642, 9.19),
	"it-milan-atm":                italyFeed("it-milan-atm", "Milan urban GTFS", "Lombardy", "Comune di Milano / ATM / AMAT", "CC-BY-4.0", "f-u0nd-comunedimilano", "Milan urban public transport GTFS discovered through Transitland.", 45.4642, 9.19),
	"it-rome":                     italyFeed("it-rome", "Rome public transport GTFS", "Lazio", "Roma Servizi per la Mobilità", "CC-BY-SA", "f-sr-atac~romatpl~trenitalia", "Daily static GTFS for Rome public transport discovered through Transitland.", 41.9028, 12.4964),
	"it-trentino-extraurban":      italyFeed("it-trentino-extraurban", "Trentino extraurban GTFS", "Trentino-Alto Adige", "Trentino Trasporti", "CC-BY-4.0", "f-u0p-trentinotrasportieserciziospa", "Extraurban GTFS for Trentino public transport discovered through Transitland.", 46.0667, 11.1211),
	"it-trentino-urban":           italyFeed("it-trentino-urban", "Trentino urban GTFS", "Trentino-Alto Adige", "Trentino Trasporti", "CC-BY-4.0", "f-u0pv-ttesercizio", "Urban GTFS for Trentino public transport discovered through Transitland.", 46.0667, 11.1211),
	"it-tuscany-autolinee":        italyFeed("it-tuscany-autolinee", "Tuscany Autolinee Toscane", "Tuscany", "Regione Toscana / Autolinee Toscane", "CC-BY-4.0", "f-s-lineeregionali", "Autolinee Toscane regional bus GTFS discovered through Transitland.", 43.7711, 11.2486),
	"it-tuscany-colbus-nonschool": italyFeed("it-tuscany-colbus-nonschool", "Tuscany Colbus non-school", "Tuscany", "Regione Toscana / Colbus", "CC-BY-4.0", "f-s-colbusnonscolastico", "Colbus non-school bus GTFS in the Florence metropolitan area discovered through Transitland.", 43.7711, 11.2486),
	"it-tuscany-colbus-school":    italyFeed("it-tuscany-colbus-school", "Tuscany Colbus school", "Tuscany", "Regione Toscana / Colbus", "CC-BY-4.0", "f-s-colbusscolastico", "Colbus school bus GTFS in the Florence metropolitan area discovered through Transitland.", 43.7711, 11.2486),
	"it-tuscany-gest":             italyFeed("it-tuscany-gest", "Florence tram GTFS", "Tuscany", "Regione Toscana / GEST", "CC-BY-4.0", "f-s-gest", "Florence tram GTFS discovered through Transitland.", 43.7711, 11.2486),
	"it-tuscany-tft":              italyFeed("it-tuscany-tft", "Tuscany TFT rail GTFS", "Tuscany", "Regione Toscana / Trasporto Ferroviario Toscano", "CC-BY-4.0", "f-sr-tft", "Trasporto Ferroviario Toscano rail GTFS discovered through Transitland.", 43.4633, 11.8796),
	"it-tuscany-toremar":          italyFeed("it-tuscany-toremar", "Tuscany Toremar ferries GTFS", "Tuscany", "Regione Toscana / Toremar", "CC-BY-4.0", "f-sp-toremar", "Toremar ferry GTFS for Tuscany coastal and island services discovered through Transitland.", 42.8129, 10.3167),
	"it-tuscany-trenitalia":       italyFeed("it-tuscany-trenitalia", "Tuscany Trenitalia regional rail", "Tuscany", "Regione Toscana / Trenitalia", "CC-BY-4.0", "f-sp-trenitaliaspa", "Trenitalia regional rail GTFS for Tuscany discovered through Transitland.", 43.7711, 11.2486),
	"it-tuscany-at-nonschool":     italyFeed("it-tuscany-at-nonschool", "Tuscany Autolinee non-school", "Tuscany", "Regione Toscana / Autolinee Toscane", "CC-BY-4.0", "f-srb-atnonscolastico", "Autolinee Toscane non-school bus GTFS in the Florence metropolitan area discovered through Transitland.", 43.7711, 11.2486),
	"it-tuscany-at-school":        italyFeed("it-tuscany-at-school", "Tuscany Autolinee school", "Tuscany", "Regione Toscana / Autolinee Toscane", "CC-BY-4.0", "f-s-atscolastico", "Autolinee Toscane school bus GTFS in the Florence metropolitan area discovered through Transitland.", 43.7711, 11.2486),
}

func communityBus(id, name, region, publisher, license, feedKey, description string, lat, lon float64) FeedSpec {
	return FeedSpec{
		ID: id, Name: name, Country: "JP", Region: region, Publisher: publisher, License: license,
		SourceURL: transitlandFeedURL(feedKey), LocalFileName: id + ".zip", Description: description,
		CenterLatitude: lat, CenterLongitude: lon,
		Attribution: "Transit data © " + publisher + ", " + license + "; discovered through Transitland.",
	}
}

func mobilityFeed(id, name, region, publisher, license, feedKey, description string, lat, lon float64) FeedSpec {
	return FeedSpec{
		ID: id, Name: name, Country: "JP", Region: region, Publisher: publisher, License: license,
		SourceURL: transitlandFeedURL(feedKey), LocalFileName: id + ".zip", Description: description,
		CenterLatitude: lat, CenterLongitude: lon,
		Attribution: "Transit data © " + publisher + ", " + license + "; discovered through Transitland.",
	}
}

func italyFeed(id, name, region, publisher, license, feedKey, description string, lat, lon float64) FeedSpec {
	return FeedSpec{
		ID: id, Name: name, Country: "IT", Region: region, Publisher: publisher, License: license,
		SourceURL: transitlandFeedURL(feedKey), LocalFileName: id + ".zip", Description: description,
		CenterLatitude: lat, CenterLongitude: lon,
		Attribution: "Transit data © " + publisher + ", " + license + "; discovered through Transitland.",
	}
}

var hokurikuPublicFeedIDs = []string{
	"kanazawa-flatbus",
	"kanazawa-hakusan-meguru",
	"kanazawa-tsubata-bus",
	"jbda-kaetsunou-kaetsunouippan",
	"jbda-nonoichicity-communitybus",
	"jbda-komatsucity-blue",
	"jbda-komatsucity-orange",
	"jbda-komatsucity-kibagatasen",
	"jbda-chitetsu-chitetsubus",
	"jbda-chitetsu-chitetsushinaidensha",
	"jbda-manyosen-manyosen",
}

var kansaiPublicFeedIDs = []string{
	"kobe-shiokaze",
	"himeji-ieshima",
	"takarazuka-runrunbus",
	"nishinomiya-sakurayamanami",
	"yamatokoriyama-kingyobus",
	"rinkan-koyasan",
	"jbda-akashicity-tacobustacobusmini",
	"jbda-kakogawacity-kakobuskakobusmini",
	"jbda-takasagocity-jotonbus",
	"jbda-nishinomiyacity-guruttonamaze",
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
		Country: "Global", Region: "Coverage",
		Description: "Single app network for Transitland-sourced coverage across Japan, Switzerland, and Italy. Feed discovery runs in tooling/CI so the app never stores a Transitland API key.",
		Publisher:   "Transitland and source transit-data publishers", License: "Mixed source licences", SourceURL: "https://transit.land/api/v2/rest/feeds",
		Attribution:    "Transit data discovered through Transitland; licences vary by publisher, so show each component feed attribution where relevant.",
		CenterLatitude: 42.5, CenterLongitude: 12.5,
		ComponentFeedIDs: transitlandCoverageFeedIDs,
	},
	{
		ID: "ch-national", Name: "Switzerland - national GTFS",
		Country: "CH", Region: "Nationwide",
		Description: "Official nationwide Swiss static GTFS timetable, suitable as the canonical country source after preprocessing.",
		Publisher:   Feeds["ch-aggregate-2026"].Publisher, License: Feeds["ch-aggregate-2026"].License, SourceURL: Feeds["ch-aggregate-2026"].SourceURL,
		Attribution: Feeds["ch-aggregate-2026"].Attribution, CenterLatitude: 46.8182, CenterLongitude: 8.2275,
		ComponentFeedIDs: []string{"ch-aggregate-2026"},
	},
	{
		ID: "it-public-regional", Name: "Italy - regional and city GTFS",
		Country: "IT", Region: "Country",
		Description: "Transitland-discovered Italian regional and city GTFS feeds currently known to the app: Rome, Milan, Lombardy rail, Tuscany, and Trentino.",
		Publisher:   "Transitland and source transit-data publishers", License: "Mixed source licences", SourceURL: "https://transit.land/api/v2/rest/feeds",
		Attribution:    "Merged network of the Italian feeds listed below. Licences vary by publisher; see each feed attribution.",
		CenterLatitude: 42.5, CenterLongitude: 12.5,
		ComponentFeedIDs: italyPublicFeedIDs,
	},
	{
		ID: "it-tuscany-public", Name: "Tuscany - regional GTFS",
		Country: "IT", Region: "Tuscany",
		Description: "Regione Toscana multimodal GTFS resources for rail, ferries, tram, and bus.",
		Publisher:   "Regione Toscana", License: "CC-BY-4.0", SourceURL: "https://transit.land/api/v2/rest/feeds",
		Attribution:    "Merged network of official Regione Toscana GTFS resources; see each feed attribution below.",
		CenterLatitude: 43.7711, CenterLongitude: 11.2486,
		ComponentFeedIDs: tuscanyPublicFeedIDs,
	},
	{
		ID: "it-trentino-public", Name: "Trentino - public transport GTFS",
		Country: "IT", Region: "Trentino-Alto Adige",
		Description: "Trentino urban and extraurban GTFS resources.",
		Publisher:   "Trentino Trasporti", License: "CC-BY-4.0", SourceURL: "https://transit.land/api/v2/rest/feeds",
		Attribution:    "Merged network of Trentino urban and extraurban GTFS resources; see each feed attribution below.",
		CenterLatitude: 46.0667, CenterLongitude: 11.1211,
		ComponentFeedIDs: trentinoPublicFeedIDs,
	},
	{
		ID: "jp-public-no-key", Name: "Japan - Transitland feeds",
		Country: "JP", Region: "Country",
		Description: "Japanese GTFS feeds currently known to the app through Transitland, merged into one local routing network.",
		Publisher:   "Transitland and source transit-data publishers", License: "Mixed source licences", SourceURL: "https://transit.land/api/v2/rest/feeds",
		Attribution:    "Merged network of the Japanese feeds listed below. Licences vary by publisher; see each feed attribution.",
		CenterLatitude: 35.681236, CenterLongitude: 139.767125, DefaultDepartureHour: hour(8),
		ComponentFeedIDs: japanPublicFeedIDs,
	},
	{
		ID: "tokyo-toei", Name: "Tokyo Toei network",
		Country: "JP", Region: "Tokyo",
		Description: "Transitland-discovered Tokyo Metropolitan Bureau of Transportation subway, tram, liner, and municipal bus feeds merged into one network.",
		Publisher:   "Tokyo Metropolitan Bureau of Transportation (東京都交通局)", License: "CC-BY-4.0", SourceURL: "https://transit.land/api/v2/rest/feeds",
		Attribution: Feeds["toei-train"].Attribution, CenterLatitude: 35.681236, CenterLongitude: 139.767125,
		DefaultDepartureHour: hour(8), ComponentFeedIDs: []string{"toei-train", "toei-bus"},
	},
	{
		ID: "kanazawa-region", Name: "Kanazawa region",
		Country: "JP", Region: "Ishikawa",
		Description: "Kanazawa, Ishikawa, and nearby Hokuriku Transitland-discovered bus and tram feeds merged into one regional network.",
		Publisher:   "Transitland and source transit-data publishers", License: "Mixed source licences", SourceURL: "https://transit.land/api/v2/rest/feeds",
		Attribution:    "Merged network of Kanazawa, Ishikawa, and nearby Hokuriku public GTFS feeds; see each feed attribution below.",
		CenterLatitude: 36.5608, CenterLongitude: 136.6566,
		ComponentFeedIDs: hokurikuPublicFeedIDs,
	},
	{
		ID: "kansai-public-no-key", Name: "Kansai - Transitland feeds",
		Country: "JP", Region: "Kansai",
		Description: "Transitland-discovered Hyogo, Nara, Shiga, and Wakayama feeds. This does not include major JR or private rail.",
		Publisher:   "Transitland and source transit-data publishers", License: "Mixed source licences", SourceURL: "https://transit.land/api/v2/rest/feeds",
		Attribution:    "Merged network of the Kansai-area Transitland feeds listed below. Major rail is not included.",
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
