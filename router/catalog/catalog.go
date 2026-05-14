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

var Networks = []NetworkSpec{
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
