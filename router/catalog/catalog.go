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
}

func communityBus(id, name, region, publisher, license, url, description string, lat, lon float64) FeedSpec {
	return FeedSpec{
		ID: id, Name: name, Country: "JP", Region: region, Publisher: publisher, License: license,
		SourceURL: url, LocalFileName: id + ".zip", Description: description,
		CenterLatitude: lat, CenterLongitude: lon,
		Attribution: "Transit data © " + publisher + ", " + license + ".",
	}
}

var Networks = []NetworkSpec{
	{
		ID: "jp-public-no-key", Name: "Japan - available public feeds",
		Description: "All no-key Japanese GTFS feeds currently known to the app, merged into one local routing network.",
		Publisher:   "Multiple public GTFS publishers", License: "Mixed open-data licences", SourceURL: "Multiple GTFS endpoints",
		Attribution:    "Merged network of the Japanese feeds listed below. Licences vary by publisher; see each feed attribution.",
		CenterLatitude: 35.681236, CenterLongitude: 139.767125, DefaultDepartureHour: hour(8),
		ComponentFeedIDs: []string{"toei-train", "toei-bus", "kanazawa-flatbus", "kanazawa-hakusan-meguru", "kanazawa-tsubata-bus", "kobe-shiokaze", "kobe-satoyama", "himeji-ieshima", "takarazuka-runrunbus", "nishinomiya-sakurayamanami", "yamatokoriyama-kingyobus", "rinkan-koyasan"},
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
		Description: "Kanazawa and nearby Hakusan/Tsubata public bus feeds merged into one regional network.",
		Publisher:   "Multiple Ishikawa public-data publishers", License: "CC-BY-4.0", SourceURL: "Multiple municipal GTFS endpoints",
		Attribution:    "Merged network of Kanazawa, Hakusan, and Tsubata public GTFS feeds; see each feed attribution below.",
		CenterLatitude: 36.5608, CenterLongitude: 136.6566,
		ComponentFeedIDs: []string{"kanazawa-flatbus", "kanazawa-hakusan-meguru", "kanazawa-tsubata-bus"},
	},
	{
		ID: "kansai-public-no-key", Name: "Kansai - available public feeds",
		Description: "Small public no-key Hyogo, Nara, and Wakayama feeds. This does not include major JR or private rail.",
		Publisher:   "Multiple public GTFS publishers", License: "Mixed open-data licences", SourceURL: "Multiple GTFS endpoints",
		Attribution:    "Merged network of the Kansai-area no-key public feeds listed below. Major rail is not included.",
		CenterLatitude: 34.6937, CenterLongitude: 135.5023,
		ComponentFeedIDs: []string{"kobe-shiokaze", "kobe-satoyama", "himeji-ieshima", "takarazuka-runrunbus", "nishinomiya-sakurayamanami", "yamatokoriyama-kingyobus", "rinkan-koyasan"},
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
