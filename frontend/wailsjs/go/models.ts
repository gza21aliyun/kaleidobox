export namespace appconf {
	
	export class AppConfig {
	    access_token?: string;
	    vndb_access_token?: string;
	    theme: string;
	    language: string;
	    sidebar_open: boolean;
	    close_to_tray: boolean;
	    ai_provider?: string;
	    ai_base_url?: string;
	    ai_api_key?: string;
	    ai_model?: string;
	    ai_system_prompt?: string;
	    cloud_backup_enabled: boolean;
	    cloud_backup_provider?: string;
	    backup_password?: string;
	    backup_user_id?: string;
	    s3_endpoint?: string;
	    s3_region?: string;
	    s3_bucket?: string;
	    s3_access_key?: string;
	    s3_secret_key?: string;
	    cloud_backup_retention?: number;
	    onedrive_client_id?: string;
	    onedrive_refresh_token?: string;
	    hikarinagi_refresh_token?: string;
	    webdav_server?: string;
	    webdav_username?: string;
	    webdav_password?: string;
	    last_db_backup_time?: string;
	    pending_db_restore?: string;
	    last_full_backup_time?: string;
	    pending_full_restore?: string;
	    auto_backup_db: boolean;
	    auto_backup_game_save: boolean;
	    auto_upload_to_cloud?: boolean;
	    auto_upload_db_to_cloud: boolean;
	    auto_upload_game_save_to_cloud: boolean;
	    local_backup_retention: number;
	    local_db_backup_retention: number;
	    window_width: number;
	    window_height: number;
	    record_active_time_only: boolean;
	    check_update_on_startup: boolean;
	    update_check_url?: string;
	    last_update_check?: string;
	    skip_version?: string;
	    background_image?: string;
	    background_blur: number;
	    background_opacity: number;
	    background_enabled: boolean;
	    background_hide_game_cover: boolean;
	    background_is_light: boolean;
	    locale_emulator_path?: string;
	    magpie_path?: string;
	    magpie_hotkey?: string;
	    magpie_config_path?: string;
	    magpie_enabled: boolean;
	    magpie_cropping_enabled: boolean;
	    magpie_cropping_left: number;
	    magpie_cropping_top: number;
	    magpie_cropping_right: number;
	    magpie_cropping_bottom: number;
	    ffmpeg_path?: string;
	    auto_detect_game_process: boolean;
	    detect_time: number;
	    time_zone?: string;
	    eroscape_use_mirror: boolean;
	    new_folder_chooser: boolean;
	    joystick_type?: string;
	    search_cn: boolean;
	    auto_download_images: boolean;
	    log_all: boolean;
	    vmrun_path?: string;
	    vm_name?: string;
	    vm_user_name?: string;
	    vm_pass?: string;
	    vm_path?: string;
	    display_name?: string;
	    sidebar_visible_items?: string;
	    rss_url?: string;
	    qb_server?: string;
	    qb_port: number;
	    qb_user?: string;
	    qb_password?: string;
	    qb_download_folder?: string;
	    game_download_folder?: string;
	    game_install_folder?: string;
	    seven_zip_path?: string;
	    hide_old_matching_btns: boolean;
	    delete_confirm: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.access_token = source["access_token"];
	        this.vndb_access_token = source["vndb_access_token"];
	        this.theme = source["theme"];
	        this.language = source["language"];
	        this.sidebar_open = source["sidebar_open"];
	        this.close_to_tray = source["close_to_tray"];
	        this.ai_provider = source["ai_provider"];
	        this.ai_base_url = source["ai_base_url"];
	        this.ai_api_key = source["ai_api_key"];
	        this.ai_model = source["ai_model"];
	        this.ai_system_prompt = source["ai_system_prompt"];
	        this.cloud_backup_enabled = source["cloud_backup_enabled"];
	        this.cloud_backup_provider = source["cloud_backup_provider"];
	        this.backup_password = source["backup_password"];
	        this.backup_user_id = source["backup_user_id"];
	        this.s3_endpoint = source["s3_endpoint"];
	        this.s3_region = source["s3_region"];
	        this.s3_bucket = source["s3_bucket"];
	        this.s3_access_key = source["s3_access_key"];
	        this.s3_secret_key = source["s3_secret_key"];
	        this.cloud_backup_retention = source["cloud_backup_retention"];
	        this.onedrive_client_id = source["onedrive_client_id"];
	        this.onedrive_refresh_token = source["onedrive_refresh_token"];
	        this.hikarinagi_refresh_token = source["hikarinagi_refresh_token"];
	        this.webdav_server = source["webdav_server"];
	        this.webdav_username = source["webdav_username"];
	        this.webdav_password = source["webdav_password"];
	        this.last_db_backup_time = source["last_db_backup_time"];
	        this.pending_db_restore = source["pending_db_restore"];
	        this.last_full_backup_time = source["last_full_backup_time"];
	        this.pending_full_restore = source["pending_full_restore"];
	        this.auto_backup_db = source["auto_backup_db"];
	        this.auto_backup_game_save = source["auto_backup_game_save"];
	        this.auto_upload_to_cloud = source["auto_upload_to_cloud"];
	        this.auto_upload_db_to_cloud = source["auto_upload_db_to_cloud"];
	        this.auto_upload_game_save_to_cloud = source["auto_upload_game_save_to_cloud"];
	        this.local_backup_retention = source["local_backup_retention"];
	        this.local_db_backup_retention = source["local_db_backup_retention"];
	        this.window_width = source["window_width"];
	        this.window_height = source["window_height"];
	        this.record_active_time_only = source["record_active_time_only"];
	        this.check_update_on_startup = source["check_update_on_startup"];
	        this.update_check_url = source["update_check_url"];
	        this.last_update_check = source["last_update_check"];
	        this.skip_version = source["skip_version"];
	        this.background_image = source["background_image"];
	        this.background_blur = source["background_blur"];
	        this.background_opacity = source["background_opacity"];
	        this.background_enabled = source["background_enabled"];
	        this.background_hide_game_cover = source["background_hide_game_cover"];
	        this.background_is_light = source["background_is_light"];
	        this.locale_emulator_path = source["locale_emulator_path"];
	        this.magpie_path = source["magpie_path"];
	        this.magpie_hotkey = source["magpie_hotkey"];
	        this.magpie_config_path = source["magpie_config_path"];
	        this.magpie_enabled = source["magpie_enabled"];
	        this.magpie_cropping_enabled = source["magpie_cropping_enabled"];
	        this.magpie_cropping_left = source["magpie_cropping_left"];
	        this.magpie_cropping_top = source["magpie_cropping_top"];
	        this.magpie_cropping_right = source["magpie_cropping_right"];
	        this.magpie_cropping_bottom = source["magpie_cropping_bottom"];
	        this.ffmpeg_path = source["ffmpeg_path"];
	        this.auto_detect_game_process = source["auto_detect_game_process"];
	        this.detect_time = source["detect_time"];
	        this.time_zone = source["time_zone"];
	        this.eroscape_use_mirror = source["eroscape_use_mirror"];
	        this.new_folder_chooser = source["new_folder_chooser"];
	        this.joystick_type = source["joystick_type"];
	        this.search_cn = source["search_cn"];
	        this.auto_download_images = source["auto_download_images"];
	        this.log_all = source["log_all"];
	        this.vmrun_path = source["vmrun_path"];
	        this.vm_name = source["vm_name"];
	        this.vm_user_name = source["vm_user_name"];
	        this.vm_pass = source["vm_pass"];
	        this.vm_path = source["vm_path"];
	        this.display_name = source["display_name"];
	        this.sidebar_visible_items = source["sidebar_visible_items"];
	        this.rss_url = source["rss_url"];
	        this.qb_server = source["qb_server"];
	        this.qb_port = source["qb_port"];
	        this.qb_user = source["qb_user"];
	        this.qb_password = source["qb_password"];
	        this.qb_download_folder = source["qb_download_folder"];
	        this.game_download_folder = source["game_download_folder"];
	        this.game_install_folder = source["game_install_folder"];
	        this.seven_zip_path = source["seven_zip_path"];
	        this.hide_old_matching_btns = source["hide_old_matching_btns"];
	        this.delete_confirm = source["delete_confirm"];
	    }
	}

}

export namespace enums {
	
	export enum GuideSource {
	    SEIYA = "seiya",
	    FOOL = "fool",
	}
	export enum SourceType {
	    LOCAL = "local",
	    BANGUMI = "bangumi",
	    VNDB = "vndb",
	    YMGAL = "ymgal",
	    DMM = "dmm",
	    EROSCAPE = "批评空间",
	    DLSITE = "dlsite",
	    GETCHU = "getchu",
	    HIKARINAGI = "hikarinagi",
	    KUNGAL = "kungal",
	}
	export enum GameStatus {
	    NOT_STARTED = "not_started",
	    PLAYING = "playing",
	    COMPLETED = "completed",
	    ON_HOLD = "on_hold",
	}
	export enum TaskStatus {
	    INITIAL = "初始",
	    STARTED = "已开始",
	    PAUSED = "暂停",
	    COMPLETED = "完成",
	    ERROR = "错误",
	    CANCELED = "取消",
	}
	export enum StaffRole {
	    STAFF = "工作人员",
	    CV = "声优",
	    SCENEARIO = "剧本",
	    DIRECTOR = "监督",
	    COMPOSER = "音乐",
	    CHARA_DESIGN = "人设",
	    CHARACTOR = "角色",
	    SINGER = "歌手",
	    ART = "画师",
	}
	export enum DeviceType {
	    KEYBOARD = "keyboard",
	    DUALSENSE = "dualsense",
	    DUALSHOCK4 = "dualshock4",
	    JOYCON = "joyconPair",
	    XINPUT = "xbox360",
	    TOUCH = "touch",
	}
	export enum Period {
	    DAY = "day",
	    WEEK = "week",
	    MONTH = "month",
	    ALL = "all",
	}
	export enum PromptType {
	    DEFAULT_SYSTEM = "你是一个幽默风趣的游戏评论员，擅长用轻松的语气点评玩家的游戏习惯。\n请用轻松幽默的方式点评这位玩家的游戏习惯，可以适当调侃但不要太过分。",
	    MEOW_ZAKO = "你是一个雌小鬼猫娘，根据用户的游戏统计数据对用户进行锐评，语气可爱活泼，不要给用户留脸面偶（=w=）适当加入猫咪的拟声词（如“喵”）和雌小鬼的口癖（如“杂鱼~杂鱼~”），要是能再用上颜文字主人就更高兴了喵。\n\n",
	    STRICT_TUTOR = "你是用户的严厉导师，根据用户的游戏统计数据对用户进行锐评，语气严肃认真，不允许任何调侃和幽默。\n\n",
	}
	export enum TaskType {
	    GAMES = "游戏搜刮更新",
	    CHARACTORS = "角色",
	    STAFFS = "工作人员",
	    IMAGES = "图片",
	    RELATIONS = "关系",
	    VIDEO_PATHS = "视频路径搜索",
	    REFRESH_GAMES = "刷新游戏",
	    DOWNLOAD_FILES = "下载文件处理",
	    CHECK_GAME_VALIDITY = "检查游戏有效性",
	    CHECK_DIRECTORY_IMPORT_STATE = "检测目录导入状态",
	}
	export enum ModifierKey {
	    CTRL = "ctrl",
	    SHIFT = "shift",
	    ALT = "alt",
	    WIN = "win",
	}
	export enum HotkeyActionType {
	    START_GAME = "start_game",
	    STOP_GAME = "stop_game",
	    TOGGLE_PAUSE = "toggle_pause",
	    SCREENSHOT = "screenshot",
	    ARROW_KEYS = "arrow_keys",
	    CUSTOM = "custom",
	}

}

export namespace models {
	
	export class Charactor {
	    id: string;
	    name: string;
	    other_names: string;
	    image_path: string;
	    images: string;
	    source_charactor_id: string;
	    source_type: enums.SourceType;
	    game_ids: string;
	    summary: string;
	    gender: number;
	    measurements: string;
	    height: string;
	    sort: number;
	
	    static createFrom(source: any = {}) {
	        return new Charactor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.other_names = source["other_names"];
	        this.image_path = source["image_path"];
	        this.images = source["images"];
	        this.source_charactor_id = source["source_charactor_id"];
	        this.source_type = source["source_type"];
	        this.game_ids = source["game_ids"];
	        this.summary = source["summary"];
	        this.gender = source["gender"];
	        this.measurements = source["measurements"];
	        this.height = source["height"];
	        this.sort = source["sort"];
	    }
	}
	export class ConnectedDevice {
	    id: string;
	    device_type: enums.DeviceType;
	    device_name: string;
	    device_id: string;
	    is_active: boolean;
	    connected_at: time.Time;
	    last_seen_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new ConnectedDevice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.device_type = source["device_type"];
	        this.device_name = source["device_name"];
	        this.device_id = source["device_id"];
	        this.is_active = source["is_active"];
	        this.connected_at = this.convertValues(source["connected_at"], time.Time);
	        this.last_seen_at = this.convertValues(source["last_seen_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DeviceTypeInfo {
	    type: enums.DeviceType;
	    name: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new DeviceTypeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.name = source["name"];
	        this.description = source["description"];
	    }
	}
	export class Game {
	    id: string;
	    name: string;
	    cover_url: string;
	    company: string;
	    summary: string;
	    path: string;
	    save_path: string;
	    process_name: string;
	    status: enums.GameStatus;
	    source_type: enums.SourceType;
	    cached_at: time.Time;
	    source_id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    tags: string;
	    arguments: string;
	    images: string;
	    source_list: string;
	    search_name: string;
	    release_at: time.Time;
	    related_games: string;
	    use_locale_emulator: boolean;
	    use_magpie: boolean;
	    pv_path: string;
	    vm_id: string;
	
	    static createFrom(source: any = {}) {
	        return new Game(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.cover_url = source["cover_url"];
	        this.company = source["company"];
	        this.summary = source["summary"];
	        this.path = source["path"];
	        this.save_path = source["save_path"];
	        this.process_name = source["process_name"];
	        this.status = source["status"];
	        this.source_type = source["source_type"];
	        this.cached_at = this.convertValues(source["cached_at"], time.Time);
	        this.source_id = source["source_id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.tags = source["tags"];
	        this.arguments = source["arguments"];
	        this.images = source["images"];
	        this.source_list = source["source_list"];
	        this.search_name = source["search_name"];
	        this.release_at = this.convertValues(source["release_at"], time.Time);
	        this.related_games = source["related_games"];
	        this.use_locale_emulator = source["use_locale_emulator"];
	        this.use_magpie = source["use_magpie"];
	        this.pv_path = source["pv_path"];
	        this.vm_id = source["vm_id"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GameBackup {
	    path: string;
	    name: string;
	    game_id: string;
	    size: number;
	    created_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new GameBackup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.game_id = source["game_id"];
	        this.size = source["size"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GameEntity {
	    game: Game;
	    worksMap: Record<string, Array<Work>>;
	    tags: Record<string, Array<Tag>>;
	
	    static createFrom(source: any = {}) {
	        return new GameEntity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game = this.convertValues(source["game"], Game);
	        this.worksMap = this.convertValues(source["worksMap"], Array<Work>, true);
	        this.tags = this.convertValues(source["tags"], Array<Tag>, true);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Review {
	    id: string;
	    title: string;
	    reviewer: string;
	    content: string;
	    link: string;
	    points: string;
	    total_points: string;
	    date: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new Review(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.reviewer = source["reviewer"];
	        this.content = source["content"];
	        this.link = source["link"];
	        this.points = source["points"];
	        this.total_points = source["total_points"];
	        this.date = this.convertValues(source["date"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GameReview {
	    id: string;
	    points: string;
	    total_points: string;
	    source_type: enums.SourceType;
	    reviews: Review[];
	    has_next: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GameReview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.points = source["points"];
	        this.total_points = source["total_points"];
	        this.source_type = source["source_type"];
	        this.reviews = this.convertValues(source["reviews"], Review);
	        this.has_next = source["has_next"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GuideContent {
	    Name: string;
	    Link: string;
	    Content: string;
	    Text: string;
	    SaveLink: string;
	    Source: enums.GuideSource;
	
	    static createFrom(source: any = {}) {
	        return new GuideContent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Link = source["Link"];
	        this.Content = source["Content"];
	        this.Text = source["Text"];
	        this.SaveLink = source["SaveLink"];
	        this.Source = source["Source"];
	    }
	}
	export class Hotkey {
	    id: string;
	    game_id: string;
	    name: string;
	    device_type: enums.DeviceType;
	    key_code: string;
	    modifiers: string;
	    action_type: enums.HotkeyActionType;
	    action_params: string;
	    is_enabled: boolean;
	    created_at: time.Time;
	    updated_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new Hotkey(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.game_id = source["game_id"];
	        this.name = source["name"];
	        this.device_type = source["device_type"];
	        this.key_code = source["key_code"];
	        this.modifiers = source["modifiers"];
	        this.action_type = source["action_type"];
	        this.action_params = source["action_params"];
	        this.is_enabled = source["is_enabled"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ImageBackup {
	    url: string;
	    local_path: string;
	    subject_id: string;
	    subject_type: number;
	    image_type: number;
	    game_id: string;
	    created_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new ImageBackup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.local_path = source["local_path"];
	        this.subject_id = source["subject_id"];
	        this.subject_type = source["subject_type"];
	        this.image_type = source["image_type"];
	        this.game_id = source["game_id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MonitorInfo {
	    index: number;
	    device_name: string;
	    left: number;
	    top: number;
	    right: number;
	    bottom: number;
	    is_primary: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MonitorInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.device_name = source["device_name"];
	        this.left = source["left"];
	        this.top = source["top"];
	        this.right = source["right"];
	        this.bottom = source["bottom"];
	        this.is_primary = source["is_primary"];
	    }
	}
	export class PlaySession {
	    id: string;
	    game_id: string;
	    start_time: time.Time;
	    end_time: time.Time;
	    duration: number;
	    pid: number;
	    process_name: string;
	
	    static createFrom(source: any = {}) {
	        return new PlaySession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.game_id = source["game_id"];
	        this.start_time = this.convertValues(source["start_time"], time.Time);
	        this.end_time = this.convertValues(source["end_time"], time.Time);
	        this.duration = source["duration"];
	        this.pid = source["pid"];
	        this.process_name = source["process_name"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ResultGames {
	    title: string;
	    description: string;
	    status: number;
	    game_ids: string[];
	
	    static createFrom(source: any = {}) {
	        return new ResultGames(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.description = source["description"];
	        this.status = source["status"];
	        this.game_ids = source["game_ids"];
	    }
	}
	
	export class Staff {
	    id: string;
	    name: string;
	    other_names: string;
	    roles: string;
	    source_staff_id: string;
	    source_type: enums.SourceType;
	    game_ids: string;
	    summary: string;
	    gender: number;
	    image: string;
	
	    static createFrom(source: any = {}) {
	        return new Staff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.other_names = source["other_names"];
	        this.roles = source["roles"];
	        this.source_staff_id = source["source_staff_id"];
	        this.source_type = source["source_type"];
	        this.game_ids = source["game_ids"];
	        this.summary = source["summary"];
	        this.gender = source["gender"];
	        this.image = source["image"];
	    }
	}
	export class Tag {
	    name: string;
	    category: string;
	    group: string;
	    is_h: boolean;
	    is_spoiler: boolean;
	    block_modify: boolean;
	    use_count: number;
	
	    static createFrom(source: any = {}) {
	        return new Tag(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.category = source["category"];
	        this.group = source["group"];
	        this.is_h = source["is_h"];
	        this.is_spoiler = source["is_spoiler"];
	        this.block_modify = source["block_modify"];
	        this.use_count = source["use_count"];
	    }
	}
	export class Task {
	    id: string;
	    name: string;
	    status: enums.TaskStatus;
	    type: enums.TaskType;
	    completed: number;
	    total: number;
	    working_on: string;
	    description: string;
	    title: string;
	    warning: string;
	    deley: number;
	    json_data: string;
	    data: any;
	    item_id: string;
	    item_status: enums.TaskStatus;
	    item_data: any;
	    result_games: ResultGames[];
	
	    static createFrom(source: any = {}) {
	        return new Task(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.type = source["type"];
	        this.completed = source["completed"];
	        this.total = source["total"];
	        this.working_on = source["working_on"];
	        this.description = source["description"];
	        this.title = source["title"];
	        this.warning = source["warning"];
	        this.deley = source["deley"];
	        this.json_data = source["json_data"];
	        this.data = source["data"];
	        this.item_id = source["item_id"];
	        this.item_status = source["item_status"];
	        this.item_data = source["item_data"];
	        this.result_games = this.convertValues(source["result_games"], ResultGames);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TaskNotice {
	    id: string;
	    name: string;
	    status: enums.TaskStatus;
	    type: enums.TaskType;
	    completed: number;
	    total: number;
	    title: string;
	    working_on: string;
	    description: string;
	    warning: string;
	    item_id: string;
	    item_status: enums.TaskStatus;
	    item_data: any;
	    result_games: ResultGames[];
	
	    static createFrom(source: any = {}) {
	        return new TaskNotice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.type = source["type"];
	        this.completed = source["completed"];
	        this.total = source["total"];
	        this.title = source["title"];
	        this.working_on = source["working_on"];
	        this.description = source["description"];
	        this.warning = source["warning"];
	        this.item_id = source["item_id"];
	        this.item_status = source["item_status"];
	        this.item_data = source["item_data"];
	        this.result_games = this.convertValues(source["result_games"], ResultGames);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class User {
	    id: string;
	    created_at: time.Time;
	    default_backup_target: string;
	
	    static createFrom(source: any = {}) {
	        return new User(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.default_backup_target = source["default_backup_target"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Vms {
	    vm_id?: string;
	    vm_name?: string;
	    vm_user_name?: string;
	    vm_pass?: string;
	    vm_path?: string;
	    vm_type?: string;
	    host_url?: string;
	    host_user?: string;
	    host_pass?: string;
	
	    static createFrom(source: any = {}) {
	        return new Vms(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vm_id = source["vm_id"];
	        this.vm_name = source["vm_name"];
	        this.vm_user_name = source["vm_user_name"];
	        this.vm_pass = source["vm_pass"];
	        this.vm_path = source["vm_path"];
	        this.vm_type = source["vm_type"];
	        this.host_url = source["host_url"];
	        this.host_user = source["host_user"];
	        this.host_pass = source["host_pass"];
	    }
	}
	export class Work {
	    id: string;
	    game_id: string;
	    staff_id: string;
	    role: enums.StaffRole;
	    charactor_id: string;
	    charactor_name: string;
	    staff_name: string;
	    work_summary: string;
	    source_type: enums.SourceType;
	    source_staff_id: string;
	    source_charactor_id: string;
	    source_game_id: string;
	    images: string;
	    game_name: string;
	    game_cover: string;
	    measurements: string;
	    height: string;
	    charactor_image: string;
	    sort: number;
	
	    static createFrom(source: any = {}) {
	        return new Work(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.game_id = source["game_id"];
	        this.staff_id = source["staff_id"];
	        this.role = source["role"];
	        this.charactor_id = source["charactor_id"];
	        this.charactor_name = source["charactor_name"];
	        this.staff_name = source["staff_name"];
	        this.work_summary = source["work_summary"];
	        this.source_type = source["source_type"];
	        this.source_staff_id = source["source_staff_id"];
	        this.source_charactor_id = source["source_charactor_id"];
	        this.source_game_id = source["source_game_id"];
	        this.images = source["images"];
	        this.game_name = source["game_name"];
	        this.game_cover = source["game_cover"];
	        this.measurements = source["measurements"];
	        this.height = source["height"];
	        this.charactor_image = source["charactor_image"];
	        this.sort = source["sort"];
	    }
	}
	export class WorkGame {
	    work: Work;
	    game: Game;
	
	    static createFrom(source: any = {}) {
	        return new WorkGame(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.work = this.convertValues(source["work"], Work);
	        this.game = this.convertValues(source["game"], Game);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace object {
	
	export class VirtualMachine {
	    InventoryPath: string;
	
	    static createFrom(source: any = {}) {
	        return new VirtualMachine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.InventoryPath = source["InventoryPath"];
	    }
	}

}

export namespace service {
	
	export class BackupService {
	
	
	    static createFrom(source: any = {}) {
	        return new BackupService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class BtDownloadResult {
	    success: boolean;
	    message: string;
	    task_id: string;
	
	    static createFrom(source: any = {}) {
	        return new BtDownloadResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.task_id = source["task_id"];
	    }
	}
	export class BtSearchResult {
	    title: string;
	    link: string;
	    torrent_url: string;
	    size: string;
	    seeders: number;
	    leechers: number;
	    trusted: boolean;
	    date: string;
	    page_link: string;
	
	    static createFrom(source: any = {}) {
	        return new BtSearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.link = source["link"];
	        this.torrent_url = source["torrent_url"];
	        this.size = source["size"];
	        this.seeders = source["seeders"];
	        this.leechers = source["leechers"];
	        this.trusted = source["trusted"];
	        this.date = source["date"];
	        this.page_link = source["page_link"];
	    }
	}
	export class CharactorService {
	
	
	    static createFrom(source: any = {}) {
	        return new CharactorService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class ConfigService {
	
	
	    static createFrom(source: any = {}) {
	        return new ConfigService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class DownloadedFile {
	    id: string;
	    name: string;
	    path: string;
	    folder_name: string;
	    folder_path: string;
	    base_name?: string;
	    extracted_game_path?: string;
	    extracted_paths?: string[];
	    type: number;
	    size: number;
	    imported_id?: string;
	    installed_path?: string;
	    time: time.Time;
	    iso_items: string[];
	    inner_items: string[];
	    has_numeric_name: boolean;
	    game_name: string;
	    status: number;
	    is_changed: boolean;
	    crack_path?: string;
	    save_status: number;
	    exe_paths: string[];
	
	    static createFrom(source: any = {}) {
	        return new DownloadedFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.folder_name = source["folder_name"];
	        this.folder_path = source["folder_path"];
	        this.base_name = source["base_name"];
	        this.extracted_game_path = source["extracted_game_path"];
	        this.extracted_paths = source["extracted_paths"];
	        this.type = source["type"];
	        this.size = source["size"];
	        this.imported_id = source["imported_id"];
	        this.installed_path = source["installed_path"];
	        this.time = this.convertValues(source["time"], time.Time);
	        this.iso_items = source["iso_items"];
	        this.inner_items = source["inner_items"];
	        this.has_numeric_name = source["has_numeric_name"];
	        this.game_name = source["game_name"];
	        this.status = source["status"];
	        this.is_changed = source["is_changed"];
	        this.crack_path = source["crack_path"];
	        this.save_status = source["save_status"];
	        this.exe_paths = source["exe_paths"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DeleteExtractedFolderResult {
	    has_archive: boolean;
	    archive_item?: DownloadedFile;
	
	    static createFrom(source: any = {}) {
	        return new DeleteExtractedFolderResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.has_archive = source["has_archive"];
	        this.archive_item = this.convertValues(source["archive_item"], DownloadedFile);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class EnumTranslations {
	    value: string;
	    key: string;
	    translations: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new EnumTranslations(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.key = source["key"];
	        this.translations = source["translations"];
	    }
	}
	export class GameService {
	
	
	    static createFrom(source: any = {}) {
	        return new GameService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class HotkeyService {
	
	
	    static createFrom(source: any = {}) {
	        return new HotkeyService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class ImageService {
	
	
	    static createFrom(source: any = {}) {
	        return new ImageService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class ImportResult {
	    success: number;
	    skipped: number;
	    failed: number;
	    failed_names: string[];
	    skipped_names: string[];
	    skipped_games: models.Game[];
	    sessions_imported: number;
	    games: models.Game[];
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.skipped = source["skipped"];
	        this.failed = source["failed"];
	        this.failed_names = source["failed_names"];
	        this.skipped_names = source["skipped_names"];
	        this.skipped_games = this.convertValues(source["skipped_games"], models.Game);
	        this.sessions_imported = source["sessions_imported"];
	        this.games = this.convertValues(source["games"], models.Game);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LaunchOptions {
	    UseLocaleEmulator?: boolean;
	    UseMagpie?: boolean;
	    DetectSavePath?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new LaunchOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.UseLocaleEmulator = source["UseLocaleEmulator"];
	        this.UseMagpie = source["UseMagpie"];
	        this.DetectSavePath = source["DetectSavePath"];
	    }
	}
	export class PreviewGame {
	    name: string;
	    developer: string;
	    source_type: string;
	    exists: boolean;
	    add_time: time.Time;
	    has_path: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PreviewGame(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.developer = source["developer"];
	        this.source_type = source["source_type"];
	        this.exists = source["exists"];
	        this.add_time = this.convertValues(source["add_time"], time.Time);
	        this.has_path = source["has_path"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SessionService {
	
	
	    static createFrom(source: any = {}) {
	        return new SessionService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class StaffService {
	
	
	    static createFrom(source: any = {}) {
	        return new StaffService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class StartService {
	
	
	    static createFrom(source: any = {}) {
	        return new StartService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class StatsService {
	
	
	    static createFrom(source: any = {}) {
	        return new StatsService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class TagService {
	
	
	    static createFrom(source: any = {}) {
	        return new TagService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class TaskService {
	
	
	    static createFrom(source: any = {}) {
	        return new TaskService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class TouchButtonInfo {
	    Index: number;
	    Name: string;
	    VirtualKey: number;
	    X: number;
	    Y: number;
	    ActionType: string;
	    Modifiers: string;
	
	    static createFrom(source: any = {}) {
	        return new TouchButtonInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Index = source["Index"];
	        this.Name = source["Name"];
	        this.VirtualKey = source["VirtualKey"];
	        this.X = source["X"];
	        this.Y = source["Y"];
	        this.ActionType = source["ActionType"];
	        this.Modifiers = source["Modifiers"];
	    }
	}
	export class TouchButtonPosition {
	    Index: number;
	    X: number;
	    Y: number;
	
	    static createFrom(source: any = {}) {
	        return new TouchButtonPosition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Index = source["Index"];
	        this.X = source["X"];
	        this.Y = source["Y"];
	    }
	}
	export class TouchMappingService {
	    ImageService?: ImageService;
	
	    static createFrom(source: any = {}) {
	        return new TouchMappingService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ImageService = this.convertValues(source["ImageService"], ImageService);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UpdateCheckResult {
	    has_update: boolean;
	    current_ver: string;
	    latest_ver: string;
	    release_date: string;
	    changelog: string[];
	    downloads: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.has_update = source["has_update"];
	        this.current_ver = source["current_ver"];
	        this.latest_ver = source["latest_ver"];
	        this.release_date = source["release_date"];
	        this.changelog = source["changelog"];
	        this.downloads = source["downloads"];
	    }
	}
	export class VMService {
	
	
	    static createFrom(source: any = {}) {
	        return new VMService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class WorkService {
	
	
	    static createFrom(source: any = {}) {
	        return new WorkService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace session {
	
	export class Manager {
	
	
	    static createFrom(source: any = {}) {
	        return new Manager(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace sql {
	
	export class DB {
	
	
	    static createFrom(source: any = {}) {
	        return new DB(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace sync {
	
	export class RWMutex {
	
	
	    static createFrom(source: any = {}) {
	        return new RWMutex(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace time {
	
	export class Time {
	
	
	    static createFrom(source: any = {}) {
	        return new Time(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace types {
	
	export class AboutInfo {
	    name: string;
	    fullName: string;
	    vendor: string;
	    version: string;
	    patchLevel?: string;
	    build: string;
	    localeVersion?: string;
	    localeBuild?: string;
	    osType: string;
	    productLineId: string;
	    apiType: string;
	    apiVersion: string;
	    instanceUuid?: string;
	    licenseProductName?: string;
	    licenseProductVersion?: string;
	
	    static createFrom(source: any = {}) {
	        return new AboutInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.fullName = source["fullName"];
	        this.vendor = source["vendor"];
	        this.version = source["version"];
	        this.patchLevel = source["patchLevel"];
	        this.build = source["build"];
	        this.localeVersion = source["localeVersion"];
	        this.localeBuild = source["localeBuild"];
	        this.osType = source["osType"];
	        this.productLineId = source["productLineId"];
	        this.apiType = source["apiType"];
	        this.apiVersion = source["apiVersion"];
	        this.instanceUuid = source["instanceUuid"];
	        this.licenseProductName = source["licenseProductName"];
	        this.licenseProductVersion = source["licenseProductVersion"];
	    }
	}
	export class ManagedObjectReference {
	    type: string;
	    value: string;
	    serverGuid?: string;
	
	    static createFrom(source: any = {}) {
	        return new ManagedObjectReference(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.value = source["value"];
	        this.serverGuid = source["serverGuid"];
	    }
	}
	export class ServiceContent {
	    rootFolder: ManagedObjectReference;
	    propertyCollector: ManagedObjectReference;
	    viewManager?: ManagedObjectReference;
	    about: AboutInfo;
	    setting?: ManagedObjectReference;
	    userDirectory?: ManagedObjectReference;
	    sessionManager?: ManagedObjectReference;
	    authorizationManager?: ManagedObjectReference;
	    serviceManager?: ManagedObjectReference;
	    perfManager?: ManagedObjectReference;
	    scheduledTaskManager?: ManagedObjectReference;
	    alarmManager?: ManagedObjectReference;
	    eventManager?: ManagedObjectReference;
	    taskManager?: ManagedObjectReference;
	    extensionManager?: ManagedObjectReference;
	    customizationSpecManager?: ManagedObjectReference;
	    guestCustomizationManager?: ManagedObjectReference;
	    customFieldsManager?: ManagedObjectReference;
	    accountManager?: ManagedObjectReference;
	    diagnosticManager?: ManagedObjectReference;
	    licenseManager?: ManagedObjectReference;
	    searchIndex?: ManagedObjectReference;
	    fileManager?: ManagedObjectReference;
	    datastoreNamespaceManager?: ManagedObjectReference;
	    virtualDiskManager?: ManagedObjectReference;
	    virtualizationManager?: ManagedObjectReference;
	    snmpSystem?: ManagedObjectReference;
	    vmProvisioningChecker?: ManagedObjectReference;
	    vmCompatibilityChecker?: ManagedObjectReference;
	    ovfManager?: ManagedObjectReference;
	    ipPoolManager?: ManagedObjectReference;
	    dvSwitchManager?: ManagedObjectReference;
	    hostProfileManager?: ManagedObjectReference;
	    clusterProfileManager?: ManagedObjectReference;
	    complianceManager?: ManagedObjectReference;
	    localizationManager?: ManagedObjectReference;
	    storageResourceManager?: ManagedObjectReference;
	    guestOperationsManager?: ManagedObjectReference;
	    overheadMemoryManager?: ManagedObjectReference;
	    certificateManager?: ManagedObjectReference;
	    ioFilterManager?: ManagedObjectReference;
	    vStorageObjectManager?: ManagedObjectReference;
	    hostSpecManager?: ManagedObjectReference;
	    cryptoManager?: ManagedObjectReference;
	    healthUpdateManager?: ManagedObjectReference;
	    failoverClusterConfigurator?: ManagedObjectReference;
	    failoverClusterManager?: ManagedObjectReference;
	    tenantManager?: ManagedObjectReference;
	    siteInfoManager?: ManagedObjectReference;
	    storageQueryManager?: ManagedObjectReference;
	
	    static createFrom(source: any = {}) {
	        return new ServiceContent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootFolder = this.convertValues(source["rootFolder"], ManagedObjectReference);
	        this.propertyCollector = this.convertValues(source["propertyCollector"], ManagedObjectReference);
	        this.viewManager = this.convertValues(source["viewManager"], ManagedObjectReference);
	        this.about = this.convertValues(source["about"], AboutInfo);
	        this.setting = this.convertValues(source["setting"], ManagedObjectReference);
	        this.userDirectory = this.convertValues(source["userDirectory"], ManagedObjectReference);
	        this.sessionManager = this.convertValues(source["sessionManager"], ManagedObjectReference);
	        this.authorizationManager = this.convertValues(source["authorizationManager"], ManagedObjectReference);
	        this.serviceManager = this.convertValues(source["serviceManager"], ManagedObjectReference);
	        this.perfManager = this.convertValues(source["perfManager"], ManagedObjectReference);
	        this.scheduledTaskManager = this.convertValues(source["scheduledTaskManager"], ManagedObjectReference);
	        this.alarmManager = this.convertValues(source["alarmManager"], ManagedObjectReference);
	        this.eventManager = this.convertValues(source["eventManager"], ManagedObjectReference);
	        this.taskManager = this.convertValues(source["taskManager"], ManagedObjectReference);
	        this.extensionManager = this.convertValues(source["extensionManager"], ManagedObjectReference);
	        this.customizationSpecManager = this.convertValues(source["customizationSpecManager"], ManagedObjectReference);
	        this.guestCustomizationManager = this.convertValues(source["guestCustomizationManager"], ManagedObjectReference);
	        this.customFieldsManager = this.convertValues(source["customFieldsManager"], ManagedObjectReference);
	        this.accountManager = this.convertValues(source["accountManager"], ManagedObjectReference);
	        this.diagnosticManager = this.convertValues(source["diagnosticManager"], ManagedObjectReference);
	        this.licenseManager = this.convertValues(source["licenseManager"], ManagedObjectReference);
	        this.searchIndex = this.convertValues(source["searchIndex"], ManagedObjectReference);
	        this.fileManager = this.convertValues(source["fileManager"], ManagedObjectReference);
	        this.datastoreNamespaceManager = this.convertValues(source["datastoreNamespaceManager"], ManagedObjectReference);
	        this.virtualDiskManager = this.convertValues(source["virtualDiskManager"], ManagedObjectReference);
	        this.virtualizationManager = this.convertValues(source["virtualizationManager"], ManagedObjectReference);
	        this.snmpSystem = this.convertValues(source["snmpSystem"], ManagedObjectReference);
	        this.vmProvisioningChecker = this.convertValues(source["vmProvisioningChecker"], ManagedObjectReference);
	        this.vmCompatibilityChecker = this.convertValues(source["vmCompatibilityChecker"], ManagedObjectReference);
	        this.ovfManager = this.convertValues(source["ovfManager"], ManagedObjectReference);
	        this.ipPoolManager = this.convertValues(source["ipPoolManager"], ManagedObjectReference);
	        this.dvSwitchManager = this.convertValues(source["dvSwitchManager"], ManagedObjectReference);
	        this.hostProfileManager = this.convertValues(source["hostProfileManager"], ManagedObjectReference);
	        this.clusterProfileManager = this.convertValues(source["clusterProfileManager"], ManagedObjectReference);
	        this.complianceManager = this.convertValues(source["complianceManager"], ManagedObjectReference);
	        this.localizationManager = this.convertValues(source["localizationManager"], ManagedObjectReference);
	        this.storageResourceManager = this.convertValues(source["storageResourceManager"], ManagedObjectReference);
	        this.guestOperationsManager = this.convertValues(source["guestOperationsManager"], ManagedObjectReference);
	        this.overheadMemoryManager = this.convertValues(source["overheadMemoryManager"], ManagedObjectReference);
	        this.certificateManager = this.convertValues(source["certificateManager"], ManagedObjectReference);
	        this.ioFilterManager = this.convertValues(source["ioFilterManager"], ManagedObjectReference);
	        this.vStorageObjectManager = this.convertValues(source["vStorageObjectManager"], ManagedObjectReference);
	        this.hostSpecManager = this.convertValues(source["hostSpecManager"], ManagedObjectReference);
	        this.cryptoManager = this.convertValues(source["cryptoManager"], ManagedObjectReference);
	        this.healthUpdateManager = this.convertValues(source["healthUpdateManager"], ManagedObjectReference);
	        this.failoverClusterConfigurator = this.convertValues(source["failoverClusterConfigurator"], ManagedObjectReference);
	        this.failoverClusterManager = this.convertValues(source["failoverClusterManager"], ManagedObjectReference);
	        this.tenantManager = this.convertValues(source["tenantManager"], ManagedObjectReference);
	        this.siteInfoManager = this.convertValues(source["siteInfoManager"], ManagedObjectReference);
	        this.storageQueryManager = this.convertValues(source["storageQueryManager"], ManagedObjectReference);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace utils {
	
	export class MonthlyReleaseGame {
	    name: string;
	    getchu_id: string;
	    cover_url: string;
	    release_at: string;
	    company: string;
	    detail_url: string;
	
	    static createFrom(source: any = {}) {
	        return new MonthlyReleaseGame(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.getchu_id = source["getchu_id"];
	        this.cover_url = source["cover_url"];
	        this.release_at = source["release_at"];
	        this.company = source["company"];
	        this.detail_url = source["detail_url"];
	    }
	}
	export class MonthlyReleaseGroup {
	    release_date: string;
	    games: MonthlyReleaseGame[];
	
	    static createFrom(source: any = {}) {
	        return new MonthlyReleaseGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.release_date = source["release_date"];
	        this.games = this.convertValues(source["games"], MonthlyReleaseGame);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MonthlyReleaseResult {
	    year: number;
	    month: number;
	    groups: MonthlyReleaseGroup[];
	
	    static createFrom(source: any = {}) {
	        return new MonthlyReleaseResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year = source["year"];
	        this.month = source["month"];
	        this.groups = this.convertValues(source["groups"], MonthlyReleaseGroup);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProcessInfo {
	    name: string;
	    pid: number;
	
	    static createFrom(source: any = {}) {
	        return new ProcessInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.pid = source["pid"];
	    }
	}
	export class SaveResult {
	    game_name: string;
	    archive_name: string;
	    save_path: string;
	    source_path: string;
	    game_id: string;
	    game_path: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game_name = source["game_name"];
	        this.archive_name = source["archive_name"];
	        this.save_path = source["save_path"];
	        this.source_path = source["source_path"];
	        this.game_id = source["game_id"];
	        this.game_path = source["game_path"];
	    }
	}

}

export namespace vim25 {
	
	export class Client {
	    Transport: any;
	    Jar: any;
	    Timeout: number;
	    namespace: string;
	    version: string;
	    userAgent: string;
	    ServiceContent: types.ServiceContent;
	    RoundTripper: any;
	
	    static createFrom(source: any = {}) {
	        return new Client(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Transport = source["Transport"];
	        this.Jar = source["Jar"];
	        this.Timeout = source["Timeout"];
	        this.namespace = source["namespace"];
	        this.version = source["version"];
	        this.userAgent = source["userAgent"];
	        this.ServiceContent = this.convertValues(source["ServiceContent"], types.ServiceContent);
	        this.RoundTripper = source["RoundTripper"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace vo {
	
	export class AISummaryRequest {
	    dimension: string;
	
	    static createFrom(source: any = {}) {
	        return new AISummaryRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dimension = source["dimension"];
	    }
	}
	export class AISummaryResponse {
	    summary: string;
	    dimension: string;
	
	    static createFrom(source: any = {}) {
	        return new AISummaryResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.summary = source["summary"];
	        this.dimension = source["dimension"];
	    }
	}
	export class BatchImportCandidate {
	    folder_path: string;
	    folder_name: string;
	    executables: string[];
	    selected_exe: string;
	    search_name: string;
	    is_selected: boolean;
	    matched_game?: models.Game;
	    match_source?: enums.SourceType;
	    match_status: string;
	    arguments: string;
	
	    static createFrom(source: any = {}) {
	        return new BatchImportCandidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folder_path = source["folder_path"];
	        this.folder_name = source["folder_name"];
	        this.executables = source["executables"];
	        this.selected_exe = source["selected_exe"];
	        this.search_name = source["search_name"];
	        this.is_selected = source["is_selected"];
	        this.matched_game = this.convertValues(source["matched_game"], models.Game);
	        this.match_source = source["match_source"];
	        this.match_status = source["match_status"];
	        this.arguments = source["arguments"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CategoryVO {
	    id: string;
	    name: string;
	    is_system: boolean;
	    created_at: time.Time;
	    updated_at: time.Time;
	    game_count: number;
	
	    static createFrom(source: any = {}) {
	        return new CategoryVO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.is_system = source["is_system"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.game_count = source["game_count"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CloudBackupItem {
	    key: string;
	    name: string;
	    size: number;
	    created_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new CloudBackupItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CloudBackupStatus {
	    enabled: boolean;
	    configured: boolean;
	    user_id: string;
	    provider: string;
	
	    static createFrom(source: any = {}) {
	        return new CloudBackupStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.configured = source["configured"];
	        this.user_id = source["user_id"];
	        this.provider = source["provider"];
	    }
	}
	export class DBBackupInfo {
	    path: string;
	    name: string;
	    size: number;
	    created_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new DBBackupInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DBBackupStatus {
	    last_backup_time: string;
	    backups: DBBackupInfo[];
	
	    static createFrom(source: any = {}) {
	        return new DBBackupStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.last_backup_time = source["last_backup_time"];
	        this.backups = this.convertValues(source["backups"], DBBackupInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DailyPlayTime {
	    date: string;
	    duration: number;
	
	    static createFrom(source: any = {}) {
	        return new DailyPlayTime(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.duration = source["duration"];
	    }
	}
	export class GameDetailStats {
	    dimension: string;
	    start_date: string;
	    end_date: string;
	    total_play_count: number;
	    total_play_time: number;
	    today_play_time: number;
	    recent_play_history: DailyPlayTime[];
	    game_id: string;
	
	    static createFrom(source: any = {}) {
	        return new GameDetailStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dimension = source["dimension"];
	        this.start_date = source["start_date"];
	        this.end_date = source["end_date"];
	        this.total_play_count = source["total_play_count"];
	        this.total_play_time = source["total_play_time"];
	        this.today_play_time = source["today_play_time"];
	        this.recent_play_history = this.convertValues(source["recent_play_history"], DailyPlayTime);
	        this.game_id = source["game_id"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DataUpdate {
	    Type: string;
	    GameStats: GameDetailStats;
	
	    static createFrom(source: any = {}) {
	        return new DataUpdate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Type = source["Type"];
	        this.GameStats = this.convertValues(source["GameStats"], GameDetailStats);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class GameMetadataFromWebVO {
	    Source: enums.SourceType;
	    Game: models.Game;
	
	    static createFrom(source: any = {}) {
	        return new GameMetadataFromWebVO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Source = source["Source"];
	        this.Game = this.convertValues(source["Game"], models.Game);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GamePlayStats {
	    game_id: string;
	    game_name: string;
	    cover_url: string;
	    total_duration: number;
	
	    static createFrom(source: any = {}) {
	        return new GamePlayStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game_id = source["game_id"];
	        this.game_name = source["game_name"];
	        this.cover_url = source["cover_url"];
	        this.total_duration = source["total_duration"];
	    }
	}
	export class GameStatsRequest {
	    game_id: string;
	    dimension: enums.Period;
	    start_date: string;
	    end_date: string;
	
	    static createFrom(source: any = {}) {
	        return new GameStatsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game_id = source["game_id"];
	        this.dimension = source["dimension"];
	        this.start_date = source["start_date"];
	        this.end_date = source["end_date"];
	    }
	}
	export class TimePoint {
	    label: string;
	    duration: number;
	
	    static createFrom(source: any = {}) {
	        return new TimePoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.duration = source["duration"];
	    }
	}
	export class GameTrendSeries {
	    game_id: string;
	    game_name: string;
	    points: TimePoint[];
	
	    static createFrom(source: any = {}) {
	        return new GameTrendSeries(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game_id = source["game_id"];
	        this.game_name = source["game_name"];
	        this.points = this.convertValues(source["points"], TimePoint);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LastPlayedGame {
	    game: models.Game;
	    last_played_at: time.Time;
	    last_played_dur: number;
	    total_played_dur: number;
	    is_playing: boolean;
	
	    static createFrom(source: any = {}) {
	        return new LastPlayedGame(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game = this.convertValues(source["game"], models.Game);
	        this.last_played_at = this.convertValues(source["last_played_at"], time.Time);
	        this.last_played_dur = source["last_played_dur"];
	        this.total_played_dur = source["total_played_dur"];
	        this.is_playing = source["is_playing"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HomePageData {
	    last_played?: LastPlayedGame;
	    today_play_time_sec: number;
	    weekly_play_time_sec: number;
	
	    static createFrom(source: any = {}) {
	        return new HomePageData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.last_played = this.convertValues(source["last_played"], LastPlayedGame);
	        this.today_play_time_sec = source["today_play_time_sec"];
	        this.weekly_play_time_sec = source["weekly_play_time_sec"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class MetadataRequest {
	    source: enums.SourceType;
	    id: string;
	    should_fetch_staffs: boolean;
	    should_fetch_charactors: boolean;
	    is_overwrite: boolean;
	    should_fetch_images: boolean;
	    should_fetch_tags: boolean;
	    db_game_id: string;
	    should_match_again: boolean;
	    should_union_fetch: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MetadataRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.id = source["id"];
	        this.should_fetch_staffs = source["should_fetch_staffs"];
	        this.should_fetch_charactors = source["should_fetch_charactors"];
	        this.is_overwrite = source["is_overwrite"];
	        this.should_fetch_images = source["should_fetch_images"];
	        this.should_fetch_tags = source["should_fetch_tags"];
	        this.db_game_id = source["db_game_id"];
	        this.should_match_again = source["should_match_again"];
	        this.should_union_fetch = source["should_union_fetch"];
	    }
	}
	export class PeriodStats {
	    dimension: enums.Period;
	    start_date: string;
	    end_date: string;
	    total_play_count: number;
	    total_play_duration: number;
	    total_games_count: number;
	    completed_games_count: number;
	    library_games_count: number;
	    all_sessions_count: number;
	    all_sessions_duration: number;
	    all_completed_games_count: number;
	    play_time_leaderboard: GamePlayStats[];
	    timeline: TimePoint[];
	    leaderboard_series: GameTrendSeries[];
	
	    static createFrom(source: any = {}) {
	        return new PeriodStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dimension = source["dimension"];
	        this.start_date = source["start_date"];
	        this.end_date = source["end_date"];
	        this.total_play_count = source["total_play_count"];
	        this.total_play_duration = source["total_play_duration"];
	        this.total_games_count = source["total_games_count"];
	        this.completed_games_count = source["completed_games_count"];
	        this.library_games_count = source["library_games_count"];
	        this.all_sessions_count = source["all_sessions_count"];
	        this.all_sessions_duration = source["all_sessions_duration"];
	        this.all_completed_games_count = source["all_completed_games_count"];
	        this.play_time_leaderboard = this.convertValues(source["play_time_leaderboard"], GamePlayStats);
	        this.timeline = this.convertValues(source["timeline"], TimePoint);
	        this.leaderboard_series = this.convertValues(source["leaderboard_series"], GameTrendSeries);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PeriodStatsRequest {
	    dimension: enums.Period;
	    start_date: string;
	    end_date: string;
	
	    static createFrom(source: any = {}) {
	        return new PeriodStatsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dimension = source["dimension"];
	        this.start_date = source["start_date"];
	        this.end_date = source["end_date"];
	    }
	}
	export class StatsGameTrend {
	    game_id: string;
	    game_name: string;
	    points: StatsTimePoint[];
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new StatsGameTrend(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game_id = source["game_id"];
	        this.game_name = source["game_name"];
	        this.points = this.convertValues(source["points"], StatsTimePoint);
	        this.color = source["color"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StatsTimePoint {
	    label: string;
	    duration: number;
	    duration_str: string;
	    hours: number;
	
	    static createFrom(source: any = {}) {
	        return new StatsTimePoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.duration = source["duration"];
	        this.duration_str = source["duration_str"];
	        this.hours = source["hours"];
	    }
	}
	export class StatsGameItem {
	    rank: number;
	    game_id: string;
	    game_name: string;
	    cover_url: string;
	    cover_base64: string;
	    total_duration: number;
	    duration_str: string;
	
	    static createFrom(source: any = {}) {
	        return new StatsGameItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rank = source["rank"];
	        this.game_id = source["game_id"];
	        this.game_name = source["game_name"];
	        this.cover_url = source["cover_url"];
	        this.cover_base64 = source["cover_base64"];
	        this.total_duration = source["total_duration"];
	        this.duration_str = source["duration_str"];
	    }
	}
	export class StatsExportData {
	    export_time: string;
	    start_date: string;
	    end_date: string;
	    period: string;
	    total_play_count: number;
	    total_play_duration: number;
	    total_play_time_str: string;
	    leaderboard: StatsGameItem[];
	    timeline: StatsTimePoint[];
	    leaderboard_trend: StatsGameTrend[];
	    chart_labels: string;
	    chart_data: string;
	    game_trend_data: string;
	    ai_summary: string;
	    app_name: string;
	    app_version: string;
	
	    static createFrom(source: any = {}) {
	        return new StatsExportData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.export_time = source["export_time"];
	        this.start_date = source["start_date"];
	        this.end_date = source["end_date"];
	        this.period = source["period"];
	        this.total_play_count = source["total_play_count"];
	        this.total_play_duration = source["total_play_duration"];
	        this.total_play_time_str = source["total_play_time_str"];
	        this.leaderboard = this.convertValues(source["leaderboard"], StatsGameItem);
	        this.timeline = this.convertValues(source["timeline"], StatsTimePoint);
	        this.leaderboard_trend = this.convertValues(source["leaderboard_trend"], StatsGameTrend);
	        this.chart_labels = source["chart_labels"];
	        this.chart_data = source["chart_data"];
	        this.game_trend_data = source["game_trend_data"];
	        this.ai_summary = source["ai_summary"];
	        this.app_name = source["app_name"];
	        this.app_version = source["app_version"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RenderTemplateRequest {
	    template_id: string;
	    data: StatsExportData;
	
	    static createFrom(source: any = {}) {
	        return new RenderTemplateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.template_id = source["template_id"];
	        this.data = this.convertValues(source["data"], StatsExportData);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RenderTemplateResponse {
	    html: string;
	
	    static createFrom(source: any = {}) {
	        return new RenderTemplateResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.html = source["html"];
	    }
	}
	
	
	
	
	export class TemplateInfo {
	    id: string;
	    name: string;
	    description: string;
	    author: string;
	    version: string;
	    preview: string;
	    is_builtin: boolean;
	    file_path: string;
	
	    static createFrom(source: any = {}) {
	        return new TemplateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.author = source["author"];
	        this.version = source["version"];
	        this.preview = source["preview"];
	        this.is_builtin = source["is_builtin"];
	        this.file_path = source["file_path"];
	    }
	}

}

