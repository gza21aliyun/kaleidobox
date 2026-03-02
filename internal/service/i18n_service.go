package service

import (
	"context"
	"lunabox/internal/applog"
)

type I18nService struct {
	ctx context.Context
}

func NewI18nService() *I18nService {
	return &I18nService{}
}

func (s *I18nService) Init(ctx context.Context) {
	s.ctx = ctx
	applog.LogInfo(ctx, "[I18nService] initialized")
}

// EnumTranslations 枚举翻译结构
type EnumTranslations struct {
	Value       string            `json:"value"`
	Key         string            `json:"key"`
	Translations map[string]string `json:"translations"`
}

// GetAllEnumTranslations 获取所有枚举的多语言翻译
func (s *I18nService) GetAllEnumTranslations() map[string][]EnumTranslations {
	translations := map[string][]EnumTranslations{
		"gameStatus": s.getGameStatusTranslations(),
		"period":     s.getPeriodTranslations(),
		"promptType": s.getPromptTypeTranslations(),
		"sourceType": s.getSourceTypeTranslations(),
		"staffRole":  s.getStaffRoleTranslations(),
		"taskStatus": s.getTaskStatusTranslations(),
		"taskType":   s.getTaskTypeTranslations(),
		"hotkeyAction": s.getHotkeyActionTranslations(),
		"deviceType": s.getDeviceTypeTranslations(),
		"modifierKey": s.getModifierKeyTranslations(),
		"joystickButton": s.getJoystickButtonTranslations(),
	}
	
	applog.LogInfo(s.ctx, "[I18nService] enum translations retrieved")
	return translations
}

// GetEnumTranslationsByType 根据类型获取枚举翻译
func (s *I18nService) GetEnumTranslationsByType(enumType string) []EnumTranslations {
	switch enumType {
	case "gameStatus":
		return s.getGameStatusTranslations()
	case "period":
		return s.getPeriodTranslations()
	case "promptType":
		return s.getPromptTypeTranslations()
	case "sourceType":
		return s.getSourceTypeTranslations()
	case "staffRole":
		return s.getStaffRoleTranslations()
	case "taskStatus":
		return s.getTaskStatusTranslations()
	case "taskType":
		return s.getTaskTypeTranslations()
	case "hotkeyAction":
		return s.getHotkeyActionTranslations()
	case "deviceType":
		return s.getDeviceTypeTranslations()
	case "modifierKey":
		return s.getModifierKeyTranslations()
	case "joystickButton":
		return s.getJoystickButtonTranslations()
	default:
		applog.LogWarningf(s.ctx, "[I18nService] unknown enum type: %s", enumType)
		return []EnumTranslations{}
	}
}

func (s *I18nService) getGameStatusTranslations() []EnumTranslations {
	return []EnumTranslations{
		{
			Value: "not_started",
			Key:   "GAME_STATUS_NOT_STARTED",
			Translations: map[string]string{
				"zh-CN": "未开始",
				"en-US": "Not Started",
				"ja-JP": "未開始",
			},
		},
		{
			Value: "playing",
			Key:   "GAME_STATUS_PLAYING",
			Translations: map[string]string{
				"zh-CN": "游玩中",
				"en-US": "Playing",
				"ja-JP": "プレイ中",
			},
		},
		{
			Value: "completed",
			Key:   "GAME_STATUS_COMPLETED",
			Translations: map[string]string{
				"zh-CN": "已通关",
				"en-US": "Completed",
				"ja-JP": "クリア済み",
			},
		},
		{
			Value: "on_hold",
			Key:   "GAME_STATUS_ON_HOLD",
			Translations: map[string]string{
				"zh-CN": "搁置",
				"en-US": "On Hold",
				"ja-JP": "一時停止",
			},
		},
	}
}

func (s *I18nService) getPeriodTranslations() []EnumTranslations {
	return []EnumTranslations{
		{
			Value: "day",
			Key:   "PERIOD_DAY",
			Translations: map[string]string{
				"zh-CN": "日",
				"en-US": "Day",
				"ja-JP": "日",
			},
		},
		{
			Value: "week",
			Key:   "PERIOD_WEEK",
			Translations: map[string]string{
				"zh-CN": "周",
				"en-US": "Week",
				"ja-JP": "週",
			},
		},
		{
			Value: "month",
			Key:   "PERIOD_MONTH",
			Translations: map[string]string{
				"zh-CN": "月",
				"en-US": "Month",
				"ja-JP": "月",
			},
		},
		{
			Value: "all",
			Key:   "PERIOD_ALL",
			Translations: map[string]string{
				"zh-CN": "全部",
				"en-US": "All",
				"ja-JP": "すべて",
			},
		},
	}
}

func (s *I18nService) getPromptTypeTranslations() []EnumTranslations {
	return []EnumTranslations{
		{
			Value: "DEFAULT_SYSTEM",
			Key:   "PROMPT_TYPE_DEFAULT",
			Translations: map[string]string{
				"zh-CN": "默认系统提示",
				"en-US": "Default System Prompt",
				"ja-JP": "デフォルトシステムプロンプト",
			},
		},
		{
			Value: "MEOW_ZAKO",
			Key:   "PROMPT_TYPE_MEOW_ZAKO",
			Translations: map[string]string{
				"zh-CN": "猫娘提示",
				"en-US": "Cat Girl Prompt",
				"ja-JP": "猫娘プロンプト",
			},
		},
		{
			Value: "STRICT_TUTOR",
			Key:   "PROMPT_TYPE_STRICT_TUTOR",
			Translations: map[string]string{
				"zh-CN": "严厉导师提示",
				"en-US": "Strict Tutor Prompt",
				"ja-JP": "厳しいチュータープロンプト",
			},
		},
	}
}

func (s *I18nService) getSourceTypeTranslations() []EnumTranslations {
	return []EnumTranslations{
		{
			Value: "local",
			Key:   "SOURCE_TYPE_LOCAL",
			Translations: map[string]string{
				"zh-CN": "本地",
				"en-US": "Local",
				"ja-JP": "ローカル",
			},
		},
		{
			Value: "bangumi",
			Key:   "SOURCE_TYPE_BANGUMI",
			Translations: map[string]string{
				"zh-CN": "Bangumi",
				"en-US": "Bangumi",
				"ja-JP": "Bangumi",
			},
		},
		{
			Value: "vndb",
			Key:   "SOURCE_TYPE_VNDB",
			Translations: map[string]string{
				"zh-CN": "VNDB",
				"en-US": "VNDB",
				"ja-JP": "VNDB",
			},
		},
		{
			Value: "ymgal",
			Key:   "SOURCE_TYPE_YMGAL",
			Translations: map[string]string{
				"zh-CN": "YM Gal",
				"en-US": "YM Gal",
				"ja-JP": "YM Gal",
			},
		},
		{
			Value: "dmm",
			Key:   "SOURCE_TYPE_DMM",
			Translations: map[string]string{
				"zh-CN": "DMM",
				"en-US": "DMM",
				"ja-JP": "DMM",
			},
		},
		{
			Value: "eroscape",
			Key:   "SOURCE_TYPE_EROSCAPE",
			Translations: map[string]string{
				"zh-CN": "批评空间",
				"en-US": "Eroscrape",
				"ja-JP": "批評空間",
			},
		},
		{
			Value: "dlsite",
			Key:   "SOURCE_TYPE_DLSITE",
			Translations: map[string]string{
				"zh-CN": "Dlsite",
				"en-US": "Dlsite",
				"ja-JP": "Dlsite",
			},
		},
	}
}

func (s *I18nService) getStaffRoleTranslations() []EnumTranslations {
	return []EnumTranslations{
		{
			Value: "STAFF",
			Key:   "STAFF_ROLE_STAFF",
			Translations: map[string]string{
				"zh-CN": "工作人员",
				"en-US": "Staff",
				"ja-JP": "スタッフ",
			},
		},
		{
			Value: "CV",
			Key:   "STAFF_ROLE_CV",
			Translations: map[string]string{
				"zh-CN": "声优",
				"en-US": "Voice Actor",
				"ja-JP": "声優",
			},
		},
		{
			Value: "SCENEARIO",
			Key:   "STAFF_ROLE_SCENEARIO",
			Translations: map[string]string{
				"zh-CN": "剧本",
				"en-US": "Scenario",
				"ja-JP": "シナリオ",
			},
		},
		{
			Value: "DIRECTOR",
			Key:   "STAFF_ROLE_DIRECTOR",
			Translations: map[string]string{
				"zh-CN": "监督",
				"en-US": "Director",
				"ja-JP": "監督",
			},
		},
		{
			Value: "COMPOSER",
			Key:   "STAFF_ROLE_COMPOSER",
			Translations: map[string]string{
				"zh-CN": "音乐",
				"en-US": "Music",
				"ja-JP": "音楽",
			},
		},
		{
			Value: "CHARA_DESIGN",
			Key:   "STAFF_ROLE_CHARA_DESIGN",
			Translations: map[string]string{
				"zh-CN": "人设",
				"en-US": "Character Design",
				"ja-JP": "キャラクターデザイン",
			},
		},
		{
			Value: "CHARACTOR",
			Key:   "STAFF_ROLE_CHARACTOR",
			Translations: map[string]string{
				"zh-CN": "角色",
				"en-US": "Character",
				"ja-JP": "キャラクター",
			},
		},
		{
			Value: "SINGER",
			Key:   "STAFF_ROLE_SINGER",
			Translations: map[string]string{
				"zh-CN": "歌手",
				"en-US": "Singer",
				"ja-JP": "歌手",
			},
		},
		{
			Value: "ART",
			Key:   "STAFF_ROLE_ART",
			Translations: map[string]string{
				"zh-CN": "画师",
				"en-US": "Artist",
				"ja-JP": "画家",
			},
		},
	}
}

func (s *I18nService) getTaskStatusTranslations() []EnumTranslations {
	return []EnumTranslations{
		{
			Value: "INITIAL",
			Key:   "TASK_STATUS_INITIAL",
			Translations: map[string]string{
				"zh-CN": "初始",
				"en-US": "Initial",
				"ja-JP": "初期",
			},
		},
		{
			Value: "STARTED",
			Key:   "TASK_STATUS_STARTED",
			Translations: map[string]string{
				"zh-CN": "已开始",
				"en-US": "Started",
				"ja-JP": "開始済み",
			},
		},
		{
			Value: "PAUSED",
			Key:   "TASK_STATUS_PAUSED",
			Translations: map[string]string{
				"zh-CN": "暂停",
				"en-US": "Paused",
				"ja-JP": "一時停止",
			},
		},
		{
			Value: "COMPLETED",
			Key:   "TASK_STATUS_COMPLETED",
			Translations: map[string]string{
				"zh-CN": "完成",
				"en-US": "Completed",
				"ja-JP": "完了",
			},
		},
		{
			Value: "ERROR",
			Key:   "TASK_STATUS_ERROR",
			Translations: map[string]string{
				"zh-CN": "错误",
				"en-US": "Error",
				"ja-JP": "エラー",
			},
		},
		{
			Value: "CANCELED",
			Key:   "TASK_STATUS_CANCELED",
			Translations: map[string]string{
				"zh-CN": "取消",
				"en-US": "Canceled",
				"ja-JP": "キャンセル",
			},
		},
	}
}

func (s *I18nService) getTaskTypeTranslations() []EnumTranslations {
	return []EnumTranslations{
		{
			Value: "GAMES",
			Key:   "TASK_TYPE_GAMES",
			Translations: map[string]string{
				"zh-CN": "游戏",
				"en-US": "Games",
				"ja-JP": "ゲーム",
			},
		},
		{
			Value: "CHARACTORS",
			Key:   "TASK_TYPE_CHARACTORS",
			Translations: map[string]string{
				"zh-CN": "角色",
				"en-US": "Characters",
				"ja-JP": "キャラクター",
			},
		},
		{
			Value: "STAFFS",
			Key:   "TASK_TYPE_STAFFS",
			Translations: map[string]string{
				"zh-CN": "工作人员",
				"en-US": "Staff",
				"ja-JP": "スタッフ",
			},
		},
		{
			Value: "IMAGES",
			Key:   "TASK_TYPE_IMAGES",
			Translations: map[string]string{
				"zh-CN": "图片",
				"en-US": "Images",
				"ja-JP": "画像",
			},
		},
		{
			Value: "RELATIONS",
			Key:   "TASK_TYPE_RELATIONS",
			Translations: map[string]string{
				"zh-CN": "关系",
				"en-US": "Relations",
				"ja-JP": "関係",
			},
		},
	}
}

func (s *I18nService) getHotkeyActionTranslations() []EnumTranslations {
	return []EnumTranslations{
		{
			Value: "START_GAME",
			Key:   "HOTKEY_ACTION_START_GAME",
			Translations: map[string]string{
				"zh-CN": "开始游戏",
				"en-US": "Start Game",
				"ja-JP": "ゲーム開始",
			},
		},
		{
			Value: "STOP_GAME",
			Key:   "HOTKEY_ACTION_STOP_GAME",
			Translations: map[string]string{
				"zh-CN": "停止游戏",
				"en-US": "Stop Game",
				"ja-JP": "ゲーム停止",
			},
		},
		{
			Value: "TOGGLE_PAUSE",
			Key:   "HOTKEY_ACTION_TOGGLE_PAUSE",
			Translations: map[string]string{
				"zh-CN": "切换暂停",
				"en-US": "Toggle Pause",
				"ja-JP": "一時停止切り替え",
			},
		},
		{
			Value: "SCREENSHOT",
			Key:   "HOTKEY_ACTION_SCREENSHOT",
			Translations: map[string]string{
				"zh-CN": "截图",
				"en-US": "Screenshot",
				"ja-JP": "スクリーンショット",
			},
		},
		{
			Value: "CUSTOM",
			Key:   "HOTKEY_ACTION_CUSTOM",
			Translations: map[string]string{
				"zh-CN": "自定义",
				"en-US": "Custom",
				"ja-JP": "カスタム",
			},
		},
	}
}

func (s *I18nService) getDeviceTypeTranslations() []EnumTranslations {
	return []EnumTranslations{
		{
			Value: "KEYBOARD",
			Key:   "DEVICE_TYPE_KEYBOARD",
			Translations: map[string]string{
				"zh-CN": "键盘",
				"en-US": "Keyboard",
				"ja-JP": "キーボード",
			},
		},
		{
			Value: "DUALSENSE",
			Key:   "DEVICE_TYPE_DUALSENSE",
			Translations: map[string]string{
				"zh-CN": "PlayStation 5 DualSense",
				"en-US": "PlayStation 5 DualSense",
				"ja-JP": "PlayStation 5 DualSense",
			},
		},
		{
			Value: "DUALSHOCK4",
			Key:   "DEVICE_TYPE_DUALSHOCK4",
			Translations: map[string]string{
				"zh-CN": "PlayStation 4 DualShock 4",
				"en-US": "PlayStation 4 DualShock 4",
				"ja-JP": "PlayStation 4 DualShock 4",
			},
		},
		{
			Value: "JOYCON",
			Key:   "DEVICE_TYPE_JOYCON",
			Translations: map[string]string{
				"zh-CN": "Nintendo Switch Joy-Con",
				"en-US": "Nintendo Switch Joy-Con",
				"ja-JP": "Nintendo Switch Joy-Con",
			},
		},
		{
			Value: "XINPUT",
			Key:   "DEVICE_TYPE_XINPUT",
			Translations: map[string]string{
				"zh-CN": "Xbox控制器及兼容设备",
				"en-US": "Xbox Controller and Compatible Devices",
				"ja-JP": "Xboxコントローラーおよび互換機器",
			},
		},
	}
}

func (s *I18nService) getModifierKeyTranslations() []EnumTranslations {
	return []EnumTranslations{
		{
			Value: "CTRL",
			Key:   "MODIFIER_KEY_CTRL",
			Translations: map[string]string{
				"zh-CN": "Ctrl",
				"en-US": "Ctrl",
				"ja-JP": "Ctrl",
			},
		},
		{
			Value: "SHIFT",
			Key:   "MODIFIER_KEY_SHIFT",
			Translations: map[string]string{
				"zh-CN": "Shift",
				"en-US": "Shift",
				"ja-JP": "Shift",
			},
		},
		{
			Value: "ALT",
			Key:   "MODIFIER_KEY_ALT",
			Translations: map[string]string{
				"zh-CN": "Alt",
				"en-US": "Alt",
				"ja-JP": "Alt",
			},
		},
		{
			Value: "WIN",
			Key:   "MODIFIER_KEY_WIN",
			Translations: map[string]string{
				"zh-CN": "Win",
				"en-US": "Win",
				"ja-JP": "Win",
			},
		},
	}
}

func (s *I18nService) getJoystickButtonTranslations() []EnumTranslations {
	return []EnumTranslations{
		{
			Value: "A",
			Key:   "JOYSTICK_BUTTON_A",
			Translations: map[string]string{
				"zh-CN": "A键",
				"en-US": "A Button",
				"ja-JP": "Aボタン",
			},
		},
		{
			Value: "B",
			Key:   "JOYSTICK_BUTTON_B",
			Translations: map[string]string{
				"zh-CN": "B键",
				"en-US": "B Button",
				"ja-JP": "Bボタン",
			},
		},
		{
			Value: "X",
			Key:   "JOYSTICK_BUTTON_X",
			Translations: map[string]string{
				"zh-CN": "X键",
				"en-US": "X Button",
				"ja-JP": "Xボタン",
			},
		},
		{
			Value: "Y",
			Key:   "JOYSTICK_BUTTON_Y",
			Translations: map[string]string{
				"zh-CN": "Y键",
				"en-US": "Y Button",
				"ja-JP": "Yボタン",
			},
		},
		{
			Value: "LB",
			Key:   "JOYSTICK_BUTTON_LB",
			Translations: map[string]string{
				"zh-CN": "左肩键",
				"en-US": "Left Bumper",
				"ja-JP": "左バンパー",
			},
		},
		{
			Value: "RB",
			Key:   "JOYSTICK_BUTTON_RB",
			Translations: map[string]string{
				"zh-CN": "右肩键",
				"en-US": "Right Bumper",
				"ja-JP": "右バンパー",
			},
		},
		{
			Value: "LT",
			Key:   "JOYSTICK_BUTTON_LT",
			Translations: map[string]string{
				"zh-CN": "左扳机",
				"en-US": "Left Trigger",
				"ja-JP": "左トリガー",
			},
		},
		{
			Value: "RT",
			Key:   "JOYSTICK_BUTTON_RT",
			Translations: map[string]string{
				"zh-CN": "右扳机",
				"en-US": "Right Trigger",
				"ja-JP": "右トリガー",
			},
		},
		{
			Value: "BACK",
			Key:   "JOYSTICK_BUTTON_BACK",
			Translations: map[string]string{
				"zh-CN": "返回键",
				"en-US": "Back Button",
				"ja-JP": "戻るボタン",
			},
		},
		{
			Value: "START",
			Key:   "JOYSTICK_BUTTON_START",
			Translations: map[string]string{
				"zh-CN": "开始键",
				"en-US": "Start Button",
				"ja-JP": "スタートボタン",
			},
		},
		{
			Value: "LS",
			Key:   "JOYSTICK_BUTTON_LS",
			Translations: map[string]string{
				"zh-CN": "左摇杆按下",
				"en-US": "Left Stick Click",
				"ja-JP": "左スティック押し込み",
			},
		},
		{
			Value: "RS",
			Key:   "JOYSTICK_BUTTON_RS",
			Translations: map[string]string{
				"zh-CN": "右摇杆按下",
				"en-US": "Right Stick Click",
				"ja-JP": "右スティック押し込み",
			},
		},
		{
			Value: "DPAD_UP",
			Key:   "JOYSTICK_BUTTON_DPAD_UP",
			Translations: map[string]string{
				"zh-CN": "方向键上",
				"en-US": "D-Pad Up",
				"ja-JP": "方向キー上",
			},
		},
		{
			Value: "DPAD_DOWN",
			Key:   "JOYSTICK_BUTTON_DPAD_DOWN",
			Translations: map[string]string{
				"zh-CN": "方向键下",
				"en-US": "D-Pad Down",
				"ja-JP": "方向キー下",
			},
		},
		{
			Value: "DPAD_LEFT",
			Key:   "JOYSTICK_BUTTON_DPAD_LEFT",
			Translations: map[string]string{
				"zh-CN": "方向键左",
				"en-US": "D-Pad Left",
				"ja-JP": "方向キー左",
			},
		},
		{
			Value: "DPAD_RIGHT",
			Key:   "JOYSTICK_BUTTON_DPAD_RIGHT",
			Translations: map[string]string{
				"zh-CN": "方向键右",
				"en-US": "D-Pad Right",
				"ja-JP": "方向キー右",
			},
		},
		{
			Value: "GUIDE",
			Key:   "JOYSTICK_BUTTON_GUIDE",
			Translations: map[string]string{
				"zh-CN": "主页键",
				"en-US": "Guide Button",
				"ja-JP": "ガイドボタン",
			},
		},
	}
}