package utils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"lunabox/internal/vo"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	"github.com/PuerkitoBio/goquery" // 添加 GoQuery 导入
	"github.com/gocolly/colly/v2"    // 添加 Colly 导入
	"github.com/labstack/gommon/log"
)

const searchBaseUrl = `https://www.dlsite.com/maniax/fsr/=/language/jp/sex_category[0]/male/keyword/%s/ana_flg/all/work_category[0]/doujin/work_category[1]/pc/order/trend/work_type_category[0]/game/options_and_or/and/from/fs.header`

// const searchBaseUrl = `https://210.140.64.86/maniax/fsr/=/language/jp/sex_category[0]/male/keyword/%s/ana_flg/all/work_category[0]/doujin/work_category[1]/pc/order/trend/work_type_category[0]/game/options_and_or/and/from/fs.header`

const apiUrl = `https://www.dlsite.com/%s/api/=/product.json?workno=%s&locale=ja-jp`

// const apiUrl = `https://210.140.64.86/%s/api/=/product.json?workno=%s&locale=ja-jp`

const workUrl = "https://www.dlsite.com/%s/work/=/product_id/%s.html"

// const workUrl = "https://74.86.226.234:443/%s/work/=/product_id/%s.html"

type DlsiteWorkResponse struct {
	AgeCategory       int           `json:"age_category"`
	AgeCategoryString string        `json:"age_category_string"`
	Anime             string        `json:"anime"`
	AutoPlay          string        `json:"auto_play"`
	Bgm               string        `json:"bgm"`
	BgmMode           string        `json:"bgm_mode"`
	BooksID           interface{}   `json:"books_id"`
	BrandID           string        `json:"brand_id"`
	CircleID          interface{}   `json:"circle_id"`
	Coupling          []interface{} `json:"coupling"`
	CPU               string        `json:"cpu"`
	DefaultPoint      int           `json:"default_point"`
	DirectedBy        interface{}   `json:"directed_by"`
	DirectX           string        `json:"directx"`
	// Discount                  *DlsiteDiscount        `json:"discount"`
	DistFlag               int               `json:"dist_flag"`
	DlFormat               int               `json:"dl_format"`
	Etc                    interface{}       `json:"etc"`
	FileDate               interface{}       `json:"file_date"`
	FileSize               interface{}       `json:"file_size"`
	FileType               string            `json:"file_type"`
	FileTypeString         string            `json:"file_type_string"`
	FileTypeSpecial        interface{}       `json:"file_type_special"`
	GalleryMode            string            `json:"gallery_mode"`
	HDD                    interface{}       `json:"hdd"`
	HSceneMode             string            `json:"h_scene_mode"`
	Intro                  interface{}       `json:"intro"`
	IntroMasked            interface{}       `json:"intro_masked"`
	IntroS                 string            `json:"intro_s"`
	IntroSMasked           string            `json:"intro_s_masked"`
	LabelID                interface{}       `json:"label_id"`
	LabelName              interface{}       `json:"label_name"`
	Machine                string            `json:"machine"`
	MachineStringList      map[string]string `json:"machine_string_list"`
	Memory                 string            `json:"memory"`
	MessageSkip            string            `json:"message_skip"`
	MiniResolution         string            `json:"mini_resolution"`
	ModifyFlg              interface{}       `json:"modify_flg"`
	MusicBy                string            `json:"music_by"`
	OnSale                 int               `json:"on_sale"`
	Options                string            `json:"options"`
	OriginalIllust         string            `json:"original_illust"`
	Other                  interface{}       `json:"other"`
	OthersBy               interface{}       `json:"others_by"`
	Pages                  interface{}       `json:"pages"`
	PageNumber             interface{}       `json:"page_number"`
	ProductPoint           interface{}       `json:"product_point"`
	ProductPointEndDate    interface{}       `json:"product_point_end_date"`
	Point                  int               `json:"point"`
	Price                  int               `json:"price"`
	PriceWithoutTax        int               `json:"price_without_tax"`
	PriceEn                float64           `json:"price_en"`
	PriceEur               float64           `json:"price_eur"`
	ProductionWorkno       interface{}       `json:"production_workno"`
	PublisherWorkno        interface{}       `json:"publisher_workno"`
	RegistDate             string            `json:"regist_date"`
	RegularPrice           interface{}       `json:"regular_price"`
	ScenarioBy             string            `json:"scenario_by"`
	ScreenMode             string            `json:"screen_mode"`
	SeriesID               interface{}       `json:"series_id"`
	SeriesName             interface{}       `json:"series_name"`
	SeriesNameMasked       interface{}       `json:"series_name_masked"`
	SexCategory            int               `json:"sex_category"`
	SofrinAppNo            interface{}       `json:"sofrin_app_no"`
	VocalTrack             string            `json:"vocal_track"`
	Voice                  string            `json:"voice"`
	VoiceBy                string            `json:"voice_by"`
	VRAM                   interface{}       `json:"vram"`
	Workno                 string            `json:"workno"`
	WorkName               string            `json:"work_name"`
	WorkNameMasked         string            `json:"work_name_masked"`
	WorkNameKana           string            `json:"work_name_kana"`
	WorkType               string            `json:"work_type"`
	WorkTypeString         string            `json:"work_type_string"`
	WorkTypeSpecial        string            `json:"work_type_special"`
	WorkTypeSpecialMasked  string            `json:"work_type_special_masked"`
	WorkAttributes         string            `json:"work_attributes"`
	ProductID              string            `json:"product_id"`
	BaseProductID          string            `json:"base_product_id"`
	MakerID                string            `json:"maker_id"`
	MakerName              string            `json:"maker_name"`
	MakerNameEn            string            `json:"maker_name_en"`
	AltName                string            `json:"alt_name"`
	AltNameMasked          string            `json:"alt_name_masked"`
	ProductName            string            `json:"product_name"`
	SiteID                 string            `json:"site_id"`
	SiteIDTouch            string            `json:"site_id_touch"`
	IsAna                  bool              `json:"is_ana"`
	WorkCategory           string            `json:"work_category"`
	Platform               []string          `json:"platform"`
	IsPCWork               bool              `json:"is_pc_work"`
	IsSmartphoneWork       bool              `json:"is_smartphone_work"`
	IsAndroidOnlyWork      bool              `json:"is_android_only_work"`
	IsIOSOnlyWork          bool              `json:"is_ios_only_work"`
	IsAndroidOrIOSOnlyWork bool              `json:"is_android_or_ios_only_work"`
	IsDLPlayboxOnlyWork    bool              `json:"is_dlplaybox_only_work"`
	IsAlmightWork          bool              `json:"is_almight_work"`
	IsDLSitePlayWork       bool              `json:"is_dlsiteplay_work"`
	IsDLSitePlayOnlyWork   bool              `json:"is_dlsiteplay_only_work"`
	WorkParts              []interface{}     `json:"work_parts"`
	Introductions          interface{}       `json:"introductions"`
	IntroductionsMasked    interface{}       `json:"introductions_masked"`
	SalesPrice             interface{}       `json:"sales_price"`
	ImageMain              *DlsiteImage      `json:"image_main"`
	ImageThum              *DlsiteImage      `json:"image_thum"`
	ImageThumMini          *DlsiteImage      `json:"image_thum_mini"`
	// ImageThumTouch         *DlsiteImage        `json:"image_thum_touch"`
	ImageThumMiniTouch []DlsiteImageTouch  `json:"image_thum_mini_touch"`
	ImageMini          *DlsiteImage        `json:"image_mini"`
	ImageSamples       []DlsiteImageSample `json:"image_samples"`
	ImageThumb         string              `json:"image_thumb"`
	ImageThumbTouch    string              `json:"image_thumb_touch"`
	// Contents                  []DlsiteContent        `json:"contents"`
	ContentsTouch         interface{} `json:"contents_touch"`
	IsSplitContent        bool        `json:"is_split_content"`
	ContentCount          int         `json:"content_count"`
	ContentCountTouch     int         `json:"content_count_touch"`
	ContentsFileSize      int64       `json:"contents_file_size"`
	ContentsFileSizeTouch int         `json:"contents_file_size_touch"`
	// Trials                bool        `json:"trials"`
	// TrialsTouch           bool        `json:"trials_touch"`
	// Movies                    []DlsiteMovie          `json:"movies"`
	EpubSample              []interface{} `json:"epub_sample"`
	SampleType              string        `json:"sample_type"`
	IsViewableSample        bool          `json:"is_viewable_sample"`
	CampaignID              int           `json:"campaign_id"`
	OfficialPrice           int           `json:"official_price"`
	OfficialPriceWithoutTax int           `json:"official_price_without_tax"`
	OfficialPriceUsd        float64       `json:"official_price_usd"`
	OfficialPriceEur        float64       `json:"official_price_eur"`
	DiscountRate            int           `json:"discount_rate"`
	IsDiscountWork          bool          `json:"is_discount_work"`
	DiscountAccessKey       string        `json:"discount_access_key"`
	DiscountLayout          interface{}   `json:"discount_layout"`
	CampaignStartDate       string        `json:"campaign_start_date"`
	CampaignEndDate         string        `json:"campaign_end_date"`
	IsShowCampaignEndDate   bool          `json:"is_show_campaign_end_date"`
	Chobits                 bool          `json:"chobits"`
	// WorkOptions               map[string]interface{} `json:"work_options"`
	Gift         []interface{} `json:"gift"`
	WorkRentals  []interface{} `json:"work_rentals"`
	IsRentalWork bool          `json:"is_rental_work"`
	// TranslationInfo           DlsiteTranslationInfo  `json:"translation_info"`
	DisplayOrder    int            `json:"display_order"`
	IsOauthWork     bool           `json:"is_oauth_work"`
	IsShowRate      bool           `json:"is_show_rate"`
	RateAverageStar int            `json:"rate_average_star"`
	RateCountDetail map[string]int `json:"rate_count_detail"`
	RankTotal       int            `json:"rank_total"`
	RankTotalDate   string         `json:"rank_total_date"`
	RankYear        int            `json:"rank_year"`
	RankYearDate    int            `json:"rank_year_date"`
	RankMonth       int            `json:"rank_month"`
	RankMonthDate   string         `json:"rank_month_date"`
	RankWeek        int            `json:"rank_week"`
	RankWeekDate    string         `json:"rank_week_date"`
	RankDay         int            `json:"rank_day"`
	RankDayDate     string         `json:"rank_day_date"`
	IsPackChild     bool           `json:"is_pack_child"`
	WorkPackParent  []interface{}  `json:"work_pack_parent"`
	IsPackParent    bool           `json:"is_pack_parent"`
	// WorkPackChildren          []interface{}  `json:"work_pack_children"`
	PackType                  interface{}    `json:"pack_type"`
	IsVoicePack               bool           `json:"is_voice_pack"`
	VoicePackParent           []interface{}  `json:"voice_pack_parent"`
	VoicePackChild            []interface{}  `json:"voice_pack_child"`
	Free                      bool           `json:"free"`
	FreeOnly                  bool           `json:"free_only"`
	FreeEndDate               bool           `json:"free_end_date"`
	HasFreeDownload           bool           `json:"has_free_download"`
	LimitedFreeTerms          []interface{}  `json:"limited_free_terms"`
	LimitedFreeWork           []interface{}  `json:"limited_free_work"`
	Creaters                  DlsiteCreaters `json:"creaters"`
	TitleID                   interface{}    `json:"title_id"`
	TitleName                 interface{}    `json:"title_name"`
	TitleNameMasked           interface{}    `json:"title_name_masked"`
	TitleVolumn               interface{}    `json:"title_volumn"`
	TitleWorkLabeling         interface{}    `json:"title_work_labeling"`
	TitleWorkDisplayOrder     interface{}    `json:"title_work_display_order"`
	TitleWorkCount            interface{}    `json:"title_work_count"`
	IsTitleCompleted          bool           `json:"is_title_completed"`
	TitleLatestWorkno         interface{}    `json:"title_latest_workno"`
	TitlePriceLow             interface{}    `json:"title_price_low"`
	TitlePriceHigh            interface{}    `json:"title_price_high"`
	IsTitlePointup            interface{}    `json:"is_title_pointup"`
	TitlePointRate            interface{}    `json:"title_point_rate"`
	IsTitleDiscount           interface{}    `json:"is_title_discount"`
	IsTitleReserve            interface{}    `json:"is_title_reserve"`
	ReserveWork               interface{}    `json:"reserve_work"`
	IsReserveWork             bool           `json:"is_reserve_work"`
	IsReservable              bool           `json:"is_reservable"`
	IsDownloadableReserveWork bool           `json:"is_downloadable_reserve_work"`
	BonusWorkno               bool           `json:"bonus_workno"`
	BonusWork                 interface{}    `json:"bonus_work"`
	IsBonusWork               bool           `json:"is_bonus_work"`
	IsDownloadableBonusWork   bool           `json:"is_downloadable_bonus_work"`
	ParentReserveWorkno       bool           `json:"parent_reserve_workno"`
	BookType                  interface{}    `json:"book_type"`
	IsBL                      bool           `json:"is_bl"`
	IsTL                      bool           `json:"is_tl"`
	IsDramaWork               bool           `json:"is_drama_work"`
	IsDisplayNotice           bool           `json:"is_display_notice"`
	TouchStyle1               []string       `json:"touch_style1"`
	IsBulkbuy                 bool           `json:"is_bulkbuy"`
	BulkbuyKey                interface{}    `json:"bulkbuy_key"`
	BulkbuyTitle              interface{}    `json:"bulkbuy_title"`
	BulkbuyPerItems           int            `json:"bulkbuy_per_items"`
	BulkbuyStart              interface{}    `json:"bulkbuy_start"`
	BulkbuyEnd                interface{}    `json:"bulkbuy_end"`
	BulkbuyPrice              int            `json:"bulkbuy_price"`
	BulkbuyPriceTax           int            `json:"bulkbuy_price_tax"`
	BulkbuyPriceWithoutTax    int            `json:"bulkbuy_price_without_tax"`
	BulkbuyDiscountRate       int            `json:"bulkbuy_discount_rate"`
	BulkbuyPointRate          int            `json:"bulkbuy_point_rate"`
	BulkbuyPoint              int            `json:"bulkbuy_point"`
	Genres                    []DlsiteGenre  `json:"genres"`
	GenresReplaced            []DlsiteGenre  `json:"genres_replaced"`
	// CustomGenres                      []DlsiteCustomGenre    `json:"custom_genres"`
	Editions          []interface{}       `json:"editions"`
	LanguageEditions  []interface{}       `json:"language_editions"`
	DisplayOptions    []interface{}       `json:"display_options"`
	WorkBrowseSetting DlsiteBrowseSetting `json:"work_browse_setting"`
	IsLimitWork       bool                `json:"is_limit_work"`
	IsLimitSales      bool                `json:"is_limit_sales"`
	IsLimitInStock    bool                `json:"is_limit_in_stock"`
	LimitSaleID       interface{}         `json:"limit_sale_id"`
	LimitStartDate    interface{}         `json:"limit_start_date"`
	LimitEndDate      interface{}         `json:"limit_end_date"`
	LimitDlCount      int                 `json:"limit_dl_count"`
	LimitSoldDlCount  int                 `json:"limit_sold_dl_count"`
	LimitDisplayType  interface{}         `json:"limit_display_type"`
	IsTimesaleWork    bool                `json:"is_timesale_work"`
	// TimesaleDlCount                   int                    `json:"timesale_dl_count"`
	// TimesaleLimitDlCount              interface{}            `json:"timesale_limit_dl_count"`
	// TimesaleStock                     int                    `json:"timesale_stock"`
	// TimesaleStartDate                 string                 `json:"timesale_start_date"`
	// TimesaleEndDate                   string                 `json:"timesale_end_date"`
	// TimesalePrice                     int                    `json:"timesale_price"`
	UpdateDate string `json:"update_date"`
	// LocalePrice                       map[string]float64     `json:"locale_price"`
	// LocaleOfficialPrice               map[string]float64     `json:"locale_official_price"`
	// LocalePriceStr                    map[string]string      `json:"locale_price_str"`
	// LocaleOfficialPriceStr            map[string]string      `json:"locale_official_price_str"`
	// CurrencyPrice                     map[string]float64     `json:"currency_price"`
	// CurrencyOfficialPrice             map[string]float64     `json:"currency_official_price"`
	GivenCouponsByBuying              []interface{} `json:"given_coupons_by_buying"`
	SpecifiedVolumeSets               []interface{} `json:"specified_volume_sets"`
	SpecifiedVolumeSetMaxDiscountRate interface{}   `json:"specified_volume_set_max_discount_rate"`
	HasSpecifiedVolumeSet             bool          `json:"has_specified_volume_set"`
	Rating                            interface{}   `json:"rating"`
	IsGarumaniGeneral                 bool          `json:"is_garumani_general"`
	IsGarumaniGeneralComipoMirror     bool          `json:"is_garumani_general_comipo_mirror"`
	ProductDir                        string        `json:"product_dir"`
}

type DlsiteDiscount struct {
	ID              string                 `json:"id"`
	Workno          string                 `json:"workno"`
	Status          string                 `json:"status"`
	CampaignID      int                    `json:"campaign_id"`
	StartDate       int64                  `json:"start_date"`
	EndDate         int64                  `json:"end_date"`
	CampaignPrice   int                    `json:"campaign_price"`
	DiscountRate    int                    `json:"discount_rate"`
	RestorePrice    int                    `json:"restore_price"`
	ShowEndDateDays string                 `json:"show_end_date_days"`
	LimitDlCount    interface{}            `json:"limit_dl_count"`
	DelFlg          string                 `json:"del_flg"`
	UpdateDate      string                 `json:"update_date"`
	UpdateID        string                 `json:"update_id"`
	InsertDate      string                 `json:"insert_date"`
	InsertID        string                 `json:"insert_id"`
	AccessKey       string                 `json:"access_key"`
	Title           string                 `json:"title"`
	Options         map[string]interface{} `json:"options"`
}

type DlsiteImage struct {
	Workno       string      `json:"workno"`
	Type         string      `json:"type"`
	FileName     string      `json:"file_name"`
	FileSize     string      `json:"file_size"`
	FileSizeUnit string      `json:"file_size_unit"`
	Width        string      `json:"width"`
	Height       string      `json:"height"`
	Hash         interface{} `json:"hash"`
	DisplayMode  string      `json:"display_mode"`
	UpdateDate   string      `json:"update_date"`
	ID           string      `json:"id"`
	UpperType    string      `json:"upper(work_files.type)"`
	Extension    string      `json:"extension"`
	RelativeURL  string      `json:"relative_url"`
	PathShort    string      `json:"path_short"`
	URL          string      `json:"url"`
	ResizeURL    string      `json:"resize_url"`
}

type DlsiteImageTouch struct {
	URL string `json:"url"`
}

type DlsiteImageSample struct {
	Workno       string      `json:"workno"`
	Type         string      `json:"type"`
	FileName     string      `json:"file_name"`
	FileSize     string      `json:"file_size"`
	FileSizeUnit string      `json:"file_size_unit"`
	Width        string      `json:"width"`
	Height       string      `json:"height"`
	Hash         interface{} `json:"hash"`
	DisplayMode  string      `json:"display_mode"`
	UpdateDate   string      `json:"update_date"`
	ID           string      `json:"id"`
	UpperType    string      `json:"upper(work_files.type)"`
	Extension    string      `json:"extension"`
	RelativeURL  string      `json:"relative_url"`
	PathShort    string      `json:"path_short"`
	URL          string      `json:"url"`
	ResizeURL    string      `json:"resize_url"`
}

type DlsiteContent struct {
	Workno       string      `json:"workno"`
	Type         string      `json:"type"`
	FileName     string      `json:"file_name"`
	FileSize     string      `json:"file_size"`
	FileSizeUnit string      `json:"file_size_unit"`
	Width        interface{} `json:"width"`
	Height       interface{} `json:"height"`
	Hash         interface{} `json:"hash"`
	DisplayMode  string      `json:"display_mode"`
	UpdateDate   string      `json:"update_date"`
	ID           string      `json:"id"`
	UpperType    string      `json:"upper(work_files.type)"`
	Extension    string      `json:"extension"`
}

type DlsiteMovie struct {
	Workno       string      `json:"workno"`
	Type         string      `json:"type"`
	FileName     string      `json:"file_name"`
	FileSize     string      `json:"file_size"`
	FileSizeUnit string      `json:"file_size_unit"`
	Width        interface{} `json:"width"`
	Height       interface{} `json:"height"`
	Hash         interface{} `json:"hash"`
	DisplayMode  string      `json:"display_mode"`
	UpdateDate   string      `json:"update_date"`
	ID           string      `json:"id"`
	UpperType    string      `json:"upper(work_files.type)"`
	Extension    string      `json:"extension"`
	RelativeURL  string      `json:"relative_url"`
	URL          string      `json:"url"`
}

type DlsiteTranslationInfo struct {
	IsTranslationAgree             bool          `json:"is_translation_agree"`
	IsVolunteer                    bool          `json:"is_volunteer"`
	IsOriginal                     bool          `json:"is_original"`
	IsParent                       bool          `json:"is_parent"`
	IsChild                        bool          `json:"is_child"`
	IsTranslationBonusChild        bool          `json:"is_translation_bonus_child"`
	OriginalWorkno                 interface{}   `json:"original_workno"`
	ParentWorkno                   interface{}   `json:"parent_workno"`
	ChildWorknos                   []interface{} `json:"child_worknos"`
	Lang                           interface{}   `json:"lang"`
	TranslationBonusLangs          []interface{} `json:"translation_bonus_langs"`
	TranslationStatusForTranslator []interface{} `json:"translation_status_for_translator"`
}

type DlsiteCreator struct {
	ID                string      `json:"id"`
	Name              string      `json:"name"`
	Classification    string      `json:"classification"`
	SubClassification interface{} `json:"sub_classification"`
}

type DlsiteCreaters struct {
	IllustBy   []DlsiteCreator `json:"illust_by"`
	ScenarioBy []DlsiteCreator `json:"scenario_by"`
	VoiceBy    []DlsiteCreator `json:"voice_by"`
	MusicBy    []DlsiteCreator `json:"music_by"`
}

type DlsiteGenre struct {
	Name      string `json:"name"`
	ID        int    `json:"id"`
	SearchVal string `json:"search_val"`
	NameBase  string `json:"name_base"`
}

type DlsiteCustomGenre struct {
	GenreKey  string      `json:"genre_key"`
	Lang      string      `json:"lang"`
	Name      string      `json:"name"`
	Layout    interface{} `json:"layout"`
	Status    string      `json:"status"`
	IsActive  int         `json:"is_active"`
	StartDate string      `json:"start_date"`
	EndDate   string      `json:"end_date"`
}

type DlsiteBrowseSetting struct {
	Content DlsiteBrowseContent `json:"content"`
	Sample  DlsiteBrowseSample  `json:"sample"`
}

type DlsiteBrowseContent struct {
	PlayEncodeType interface{} `json:"play_encode_type"`
}

type DlsiteBrowseSample struct {
	PlayEncodeType interface{} `json:"play_encode_type"`
}

type DlsiteInfoGetter struct {
	client  *http.Client
	timeout time.Duration
}

func (b DlsiteInfoGetter) FetchMetadataByName(name string) (models.Game, error) {
	game, err := b.FetchByNameImpl(name,
		func(request vo.MetadataRequest) (models.Game, error) {
			fmt.Println("FetchMetadataByName 34")
			gameEntity, err := b.FetchMetadataById2(request)
			return gameEntity.Game, err
		})
	return game, err
}

func (b DlsiteInfoGetter) FetchByNameImpl(searchName string, fn IdFunction) (models.Game, error) {
	// if !dmmIsEnabled {
	// 	return models.Game{}, fmt.Errorf("DMM is not enabled")
	// }
	mainTitle, _, _ := getTitles(searchName)
	var url string = fmt.Sprintf(searchBaseUrl, mainTitle)
	var game = models.Game{}
	// c := CreateCollector2("*dlsite.com")

	var potentialGames []struct {
		Title    string
		Link     string
		Review   string
		CoverUrl string
	}

	rawData, _ := getRawResponse(url)
	document, err := goquery.NewDocumentFromReader(bytes.NewReader(rawData))
	document.Find("ul#search_result_img_box li > dl").Each(func(i int, e *goquery.Selection) {
		// e.Find("")

		link := e.Find("dl > dt > a").AttrOr("href", "")
		// price := e.ChildText(".component-legacy-productTile__review")
		// log.Print("OnHTML 网页列表 ：", e.Text)
		title := e.Find("div.multiline_truncate").Text()
		fmt.Println("link 03:", link)
		fmt.Println("link 034:", title)
		// html, _ := e.Html()
		// fmt.Println("link 033:", html)

		if link != "" {
			potentialGames = append(potentialGames, struct {
				Title    string
				Link     string
				Review   string
				CoverUrl string
			}{
				Title: title,
				Link:  link,
				// Review:   price,
				// CoverUrl: e.ChildAttr("span.component-legacy-productTile__thumbnail img", "src"),
			})
		}

	})
	fmt.Println("link 04:", len(potentialGames))
	gameFound := searchNameByRegex(potentialGames, searchName, []string{"セット"}, func(t1 struct {
		Title    string
		Link     string
		Review   string
		CoverUrl string
	}) string {
		return t1.Title
	})
	if gameFound != nil {
		game.Name = gameFound.Title
		fmt.Println("dlsite详情：" + gameFound.Link)
		linkParts := strings.Split(gameFound.Link, "/")
		id := strings.ReplaceAll(linkParts[len(linkParts)-1], ".html", "")
		fmt.Println("05 id: " + id)
		game.SourceID = id
		game.SourceType = enums.Dlsite
		game.DlsiteId = game.SourceID
		game.CoverURL = gameFound.CoverUrl
	}

	if err != nil {
		return models.Game{}, err
	}

	// 等待收集完成
	// c.Wait()
	if game.SourceID == "" {
		return game, errors.New("游戏未找到")
	}
	game, err = fn(GetReqEntity(&game))

	return game, err
}

// 获取原始响应数据的函数
func getRawResponse(url string) ([]byte, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 设置相同的请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (b DlsiteInfoGetter) FetchMetadataById2(request vo.MetadataRequest) (models.GameEntity, error) {
	var gameEntity models.GameEntity = models.GameEntity{}
	var game models.Game = request.GetGame()

	gameEntity.Game = game

	url := ""
	if strings.Contains(request.ID, "RJ") {
		url = fmt.Sprintf(apiUrl, "maniax", request.ID)
	} else {
		url = fmt.Sprintf(apiUrl, "pro", request.ID)
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("DlsiteInfoGetter FetchMetadata 01 error: %v", err)
		return gameEntity, err
	}

	req.Header.Set("User-Agent", "GameManage/1.0")
	req.Header.Set("Accept", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		fmt.Printf("DlsiteInfoGetter FetchMetadata 02 error: %v\n", err)
		return gameEntity, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("DlsiteInfoGetter FetchMetadata 03 error: %v", err)
			log.Warnf("Error closing response body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return gameEntity, fmt.Errorf("dlsite API returned status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var dlsiteResp []DlsiteWorkResponse
	if err := json.NewDecoder(resp.Body).Decode(&dlsiteResp); err != nil {
		fmt.Println("DlsiteInfoGetter FetchMetadata 04 error: %v", err)
		fmt.Println("DlsiteInfoGetter FetchMetadata 055")
		return gameEntity, err
	}
	data, err := json.MarshalIndent(dlsiteResp, "", "  ")
	fmt.Println("DlsiteInfoGetter FetchMetadata 05")
	fmt.Println(string(data))
	if err != nil {
		return gameEntity, err
	}
	if len(dlsiteResp) == 0 {
		return gameEntity, errors.New("DlsiteInfoGetter FetchMetadata 06 no data")
	}
	dlsiteData := dlsiteResp[0]
	date, err := time.Parse("2006-01-02 15:04:05", dlsiteData.RegistDate)
	game = models.Game{
		ID:        request.DbGameId,
		SourceID:  dlsiteData.Workno,
		Name:      dlsiteData.WorkName,
		DlsiteId:  dlsiteData.Workno,
		ReleaseAt: date,
		CoverURL:  dlsiteData.ImageMain.URL,
		Company:   dlsiteData.MakerName,
		Summary:   dlsiteData.IntroS,
		Images:    JoinString(dlsiteData.ImageSamples, ",", func(img DlsiteImageSample) string { return img.URL }),
	}
	// game.Summary =
	tagList := []models.Tag{}
	for _, tag := range dlsiteData.Genres {
		tagList = append(tagList, models.Tag{
			Name:     tag.Name,
			Category: models.TagCategoryOther,
		})
	}
	tagList = append(tagList, models.Tag{
		Name:        dlsiteData.MakerName,
		Category:    models.TagCategoryBrand,
		BlockModify: true,
	})
	tagList = append(tagList, models.Tag{
		Name:        dlsiteData.WorkType,
		Category:    models.TagCategoryGenre,
		BlockModify: true,
	})
	if dlsiteData.Voice == "あり" {
		tagList = append(tagList, models.Tag{
			Name:     "音声" + dlsiteData.Voice,
			Category: models.TagCategoryOther,
		})
	}
	if dlsiteData.VocalTrack == "あり" {
		tagList = append(tagList, models.Tag{
			Name:     "音楽" + dlsiteData.VocalTrack,
			Category: models.TagCategoryOther,
		})
	}
	if dlsiteData.Anime == "あり" {
		tagList = append(tagList, models.Tag{
			Name:     "動画" + dlsiteData.Anime,
			Category: models.TagCategoryOther,
		})
	}
	if dlsiteData.AutoPlay == "あり" {
		tagList = append(tagList, models.Tag{
			Name:     "自動" + dlsiteData.AutoPlay,
			Category: models.TagCategoryOther,
		})
	}
	if dlsiteData.Bgm == "あり" {
		tagList = append(tagList, models.Tag{
			Name:     "BGM" + dlsiteData.Bgm,
			Category: models.TagCategoryOther,
		})
	}
	if dlsiteData.BgmMode == "あり" {
		tagList = append(tagList, models.Tag{
			Name:     "BGMモード" + dlsiteData.BgmMode,
			Category: models.TagCategoryOther,
		})
	}
	if dlsiteData.GalleryMode == "あり" {
		tagList = append(tagList, models.Tag{
			Name:     "Gallery" + dlsiteData.GalleryMode,
			Category: models.TagCategoryOther,
		})
	}
	if dlsiteData.HSceneMode == "あり" {
		tagList = append(tagList, models.Tag{
			Name:        "HSceneあり",
			Category:    models.TagCategoryProperty,
			BlockModify: true,
		})
	}
	if dlsiteData.MessageSkip == "あり" {
		tagList = append(tagList, models.Tag{
			Name:        "MessageSkip" + dlsiteData.MessageSkip,
			Category:    models.TagCategoryProperty,
			BlockModify: true,
		})
	}
	var worksMap map[enums.StaffRole][]models.Work = make(map[enums.StaffRole][]models.Work)
	for _, work := range dlsiteData.Creaters.IllustBy {
		worksMap[enums.Art] = append(worksMap[enums.Art], models.Work{
			GameId:        game.ID,
			Role:          enums.Art,
			StaffName:     work.Name,
			SourceStaffId: work.ID,
			SourceType:    enums.Dlsite,
		})
	}
	for _, work := range dlsiteData.Creaters.MusicBy {
		worksMap[enums.Composer] = append(worksMap[enums.Composer], models.Work{
			GameId:        game.ID,
			Role:          enums.Composer,
			StaffName:     work.Name,
			SourceStaffId: work.ID,
			SourceType:    enums.Dlsite,
		})
	}
	for _, work := range dlsiteData.Creaters.ScenarioBy {
		worksMap[enums.Sceneario] = append(worksMap[enums.Sceneario], models.Work{
			GameId:        game.ID,
			Role:          enums.Sceneario,
			StaffName:     work.Name,
			SourceStaffId: work.ID,
			SourceType:    enums.Dlsite,
		})
	}
	for _, work := range dlsiteData.Creaters.VoiceBy {
		worksMap[enums.CV] = append(worksMap[enums.CV], models.Work{
			GameId:        game.ID,
			Role:          enums.CV,
			StaffName:     work.Name,
			SourceStaffId: work.ID,
			SourceType:    enums.Dlsite,
		})
	}
	gameEntity.Game = game
	gameEntity.WorksMap = worksMap
	gameEntity.Tags = ArrayToMap(tagList, func(t1 models.Tag) string { return t1.Category })

	return gameEntity, nil
}

func (b DlsiteInfoGetter) FetchMetadataById(request vo.MetadataRequest) (models.GameEntity, error) {

	fmt.Println("开始获取Dlsite游戏信息 36 " + request.Source)
	var game models.Game = request.GetGame()
	var gameEntity models.GameEntity = models.GameEntity{}
	gameEntity.Game = game
	var err error = nil
	if request.ID == "" {
		return gameEntity, fmt.Errorf("dlsite ID is required to fetch metadata by ID")
	}
	fmt.Println("开始获dlsite游戏信息 37 " + request.ID)
	game.DlsiteId = request.ID
	dlsiteUrl := ""
	if strings.Contains(request.ID, "RJ") {
		dlsiteUrl = fmt.Sprintf(workUrl, "maniax", request.ID)
	} else {
		dlsiteUrl = fmt.Sprintf(workUrl, "pro", request.ID)
	}
	c := CreateCollector("*dlsite.com")

	// 处理游戏详情页面
	c.OnHTML("body", func(e *colly.HTMLElement) {
		name := e.ChildText("h1#work_name")
		fmt.Println("开始获取Dlsite游戏信息 ， title:" + e.ChildText("h1"))
		game.Name = name

		// 提取公司信息
		company := e.ChildText("span.maker_name")

		game.Company = company

		tagList := []models.Tag{}
		tagList = append(tagList, models.Tag{Name: company, Category: models.TagCategoryBrand, BlockModify: true})

		e.DOM.Find("div.work_genre a").Each(func(i int, s *goquery.Selection) {
			tag := strings.TrimSpace(s.Text())
			if !strings.Contains(tag, "還元") && !strings.Contains(tag, "クーポン") && !strings.Contains(tag, "セール") &&
				!strings.Contains(tag, "独占販売") {
				tagList = append(tagList, models.Tag{Name: tag, Category: models.TagCategoryGenre})
			}

		})

		// 提取简介
		summary, _ := e.DOM.Find("div.work_parts_area p").Html()
		summary = strings.ReplaceAll(summary, "<br>", "\n")
		game.Summary = summary
		releaseAt := e.ChildText("table#work_outline tbody tr:contains('販売日') td")
		game.ReleaseAt, err = time.Parse("2006年01月02日", releaseAt)
		fmt.Printf("发售日11：%v, %s, %v\n", game.ReleaseAt, releaseAt, err)
		if err != nil {
			return
		}
		// 提取标签
		e.DOM.Find("div.main_genre a").Each(func(i int, s *goquery.Selection) {
			tag := strings.TrimSpace(s.Text())
			if !strings.Contains(tag, "還元") && !strings.Contains(tag, "クーポン") && !strings.Contains(tag, "セール") &&
				!strings.Contains(tag, "独占販売") {
				tagList = append(tagList, models.Tag{Name: tag, Category: models.TagCategoryOther})
			}

		})
		game.Tags = JoinString(tagList, ",", func(tag models.Tag) string { return tag.Name })
		gameEntity.Tags = ArrayToMap(tagList, func(t1 models.Tag) string { return t1.Category })
		// 获取图片
		var images []string
		e.DOM.Find("ul.slider_items li img").Each(func(i int, s *goquery.Selection) {
			image, _ := s.Attr("src")
			if image == "" {
				return
			} else if strings.Contains(image, "main") {
				game.CoverURL = image
			} else {
				images = append(images, image)
			}

		})
		game.Images = strings.Join(images, ",")
		var worksMap map[enums.StaffRole][]models.Work = make(map[enums.StaffRole][]models.Work)
		e.DOM.Find("table#work_outline tbody tr:contains('イラスト') td").Each(func(i int, s *goquery.Selection) {
			staffName := strings.TrimSpace(s.Text())
			StaffUrl := s.AttrOr("href", "")
			if staffName != "" && StaffUrl != "" {
				parsedURL, err := url.Parse(StaffUrl)
				if err != nil {
					fmt.Println("URL 解析失败:", err)
					return
				}
				// 获取查询参数
				queryParams := parsedURL.Query()

				// 提取 character 参数的值
				staffId := queryParams.Get("author")
				sId := "author=" + staffId
				work := models.Work{
					GameId:        game.ID,
					Role:          enums.Art,
					StaffName:     staffName,
					SourceStaffId: sId,
					SourceType:    enums.Dlsite,
					GameName:      game.Name,
				}
				worksMap[work.Role] = append(worksMap[work.Role], work)
			}
		})
		e.DOM.Find("table#work_outline tbody tr:contains('シナリオ') td").Each(func(i int, s *goquery.Selection) {
			staffName := strings.TrimSpace(s.Text())
			StaffUrl := s.AttrOr("href", "")
			if staffName != "" && StaffUrl != "" {
				parsedURL, err := url.Parse(StaffUrl)
				if err != nil {
					fmt.Println("URL 解析失败:", err)
					return
				}
				// 获取查询参数
				queryParams := parsedURL.Query()

				// 提取 character 参数的值
				staffId := queryParams.Get("scenario")
				sId := "scenario=" + staffId
				work := models.Work{
					GameId:        game.ID,
					Role:          enums.Sceneario,
					StaffName:     staffName,
					SourceStaffId: sId,
					SourceType:    enums.Dlsite,
					GameName:      game.Name,
				}
				worksMap[work.Role] = append(worksMap[work.Role], work)
			}
		})
		e.DOM.Find("table#work_outline tbody tr:contains('音楽') td").Each(func(i int, s *goquery.Selection) {
			staffName := strings.TrimSpace(s.Text())
			StaffUrl := s.AttrOr("href", "")
			if staffName != "" && StaffUrl != "" {
				parsedURL, err := url.Parse(StaffUrl)
				if err != nil {
					fmt.Println("URL 解析失败:", err)
					return
				}
				// 获取查询参数
				queryParams := parsedURL.Query()

				// 提取 character 参数的值
				staffId := queryParams.Get("scenario")
				sId := "scenario=" + staffId
				work := models.Work{
					GameId:        game.ID,
					Role:          enums.Composer,
					StaffName:     staffName,
					SourceStaffId: sId,
					SourceType:    enums.Dlsite,
					GameName:      game.Name,
				}
				worksMap[work.Role] = append(worksMap[work.Role], work)
			}
		})
		e.DOM.Find("table#work_outline tbody tr:contains('声優') td").Each(func(i int, s *goquery.Selection) {
			staffName := strings.TrimSpace(s.Text())
			if staffName != "" {
				staffId := staffName
				sId := "voice_actor=" + staffId
				work := models.Work{
					GameId:        game.ID,
					Role:          enums.CV,
					StaffName:     staffName,
					SourceStaffId: sId,
					SourceType:    enums.Dlsite,
					GameName:      game.Name,
				}
				worksMap[work.Role] = append(worksMap[work.Role], work)
			}
		})

		jstr, _ := json.Marshal(worksMap)
		log.Printf("worksMap: " + string(jstr))
		gameEntity.WorksMap = worksMap

	})

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Request error: %s with error: %s\n", r.Request.URL, err)
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Printf("=== 响应调试 ===\n")
		fmt.Printf("URL: %s\n", r.Request.URL)
		fmt.Printf("状态码: %d\n", r.StatusCode)
		fmt.Printf("Content-Type: %s\n", r.Headers.Get("Content-Type"))
		fmt.Printf("响应长度: %d 字节\n", len(r.Body))

		// 显示前几个字节的十六进制
		if len(r.Body) > 0 {
			fmt.Printf("前20字节(hex): ")
			for i := 0; i < min(20, len(r.Body)); i++ {
				fmt.Printf("%02x ", r.Body[i])
			}
			fmt.Println()
		}

		// 解码处理
		decodedBody, err := decodeJapaneseContent(r.Body, r.Headers.Get("Content-Type"))
		if err != nil {
			fmt.Printf("编码处理错误: %v\n", err)
			return
		}

		// 保存解码后的内容供分析
		debugFile := fmt.Sprintf("final_decoded_%s.html", request.ID)
		os.WriteFile(debugFile, r.Body, 0644)
		fmt.Printf("✅ 解码成功，内容已保存到: %s\n", debugFile)

		// 验证解码结果
		if strings.Contains(decodedBody, "[编码检测失败]") {
			fmt.Println("⚠️  编码检测可能失败，请检查生成的文件")
		} else {
			fmt.Printf("解码后内容预览: %.300s...\n", decodedBody)
		}

		// 更新响应体
		r.Body = []byte(decodedBody)
	})

	c.OnScraped(func(r *colly.Response) {
		// fmt.Printf("Scraped:%s\n", string(r.Body))
	})

	// 访问构建的 URL
	err = c.Visit(dlsiteUrl)
	if err != nil {
		fmt.Printf("开始获取Dlsite游戏信息 40 err: %s\n", err)
		return gameEntity, err
	}

	// 等待收集完成
	c.Wait()
	fmt.Println("开始获取Dlsite游戏信息 39 " + game.CoverURL)

	// 检查是否成功获取了数据
	if game.Name == "" {
		fmt.Println("开始获取Dlsite游戏信息 40 " + game.CoverURL)
		return gameEntity, fmt.Errorf("game not found: %s", game.SourceID)
	}
	gameEntity = combineCharacters(gameEntity)
	fmt.Print("角色人数：", len(gameEntity.WorksMap[enums.Charactor]))

	// 设置其他必要字段
	game.SourceType = enums.Dlsite // 假设你有这个枚举
	// game.SourceID = name
	game.CachedAt = time.Now()
	gameEntity.Game = game
	fmt.Println("开始获取Dlsite游戏信息 38 " + game.CoverURL)

	return gameEntity, nil
}

// 编码解码函数
// 改进的编码处理函数
func decodeJapaneseContent(content []byte, contentType string) (string, error) {
	fmt.Printf("原始内容前50字节: %v\n", content[:min(50, len(content))])
	fmt.Printf("Content-Type: %s\n", contentType)

	// 方法1: 直接尝试Shift-JIS解码（最高优先级）
	fmt.Println("🔄 尝试Shift-JIS解码...")

	// 使用更严格的Shift-JIS解码器
	decoder := japanese.ShiftJIS.NewDecoder()
	reader := transform.NewReader(bytes.NewReader(content), decoder)

	decodedBytes, err := io.ReadAll(reader)
	if err != nil {
		fmt.Printf("Shift-JIS解码错误: %v\n", err)
		return "", err
	}

	decodedText := string(decodedBytes)
	fmt.Printf("Shift-JIS解码后前200字符: %.200s\n", decodedText)

	// 验证解码结果的质量
	if isValidJapaneseText(decodedText) {
		fmt.Println("✅ Shift-JIS解码成功且内容有效")
		return decodedText, nil
	}

	fmt.Println("❌ Shift-JIS解码结果质量不佳")

	// 方法2: 尝试UTF-8（尽管Content-Type说UTF-8，但实际可能是Shift-JIS）
	fmt.Println("🔄 尝试UTF-8解码...")
	if utf8.Valid(content) {
		utf8Text := string(content)
		fmt.Printf("UTF-8解码后前200字符: %.200s\n", utf8Text)
		if isValidJapaneseText(utf8Text) {
			fmt.Println("✅ UTF-8解码有效")
			return utf8Text, nil
		}
	}

	// 方法3: 尝试EUC-JP
	fmt.Println("🔄 尝试EUC-JP解码...")
	eucDecoder := japanese.EUCJP.NewDecoder()
	eucReader := transform.NewReader(bytes.NewReader(content), eucDecoder)

	eucBytes, err := io.ReadAll(eucReader)
	if err == nil {
		eucText := string(eucBytes)
		fmt.Printf("EUC-JP解码后前200字符: %.200s\n", eucText)
		if isValidJapaneseText(eucText) {
			fmt.Println("✅ EUC-JP解码成功")
			return eucText, nil
		}
	}

	return "", fmt.Errorf("所有编码尝试都失败")
}

// 验证日文文本质量
func isValidJapaneseText(text string) bool {
	if len(text) < 10 {
		return false
	}

	// 统计日文字符比例
	japaneseCount := 0
	totalCount := 0

	for _, r := range text {
		totalCount++
		// 检查是否为日文字符
		if (r >= 0x3040 && r <= 0x309F) || // 平假名
			(r >= 0x30A0 && r <= 0x30FF) || // 片假名
			(r >= 0x4E00 && r <= 0x9FFF) || // 汉字
			unicode.IsLetter(r) || // 其他字母
			unicode.IsDigit(r) || // 数字
			unicode.IsSpace(r) { // 空白字符
			if r > 127 { // 非ASCII字符
				japaneseCount++
			}
		}
	}

	// 日文字符比例应该超过一定阈值
	ratio := float64(japaneseCount) / float64(totalCount)
	fmt.Printf("日文字符比例: %.2f%% (%d/%d)\n", ratio*100, japaneseCount, totalCount)

	return ratio > 0.3 // 至少30%的日文字符
}

// 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
