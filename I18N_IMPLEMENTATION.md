# LunaBox 多语言国际化(i18n)改造方案

## 已完成工作

### 1. 前端i18n基础设施
- ✅ 创建了完整的i18n目录结构 (`frontend/src/i18n/`)
- ✅ 实现了三种语言的语言包：
  - 中文 (zh-CN)
  - 英文 (en-US) 
  - 日文 (ja-JP)
- ✅ 集成了i18next和react-i18next库
- ✅ 实现了自动语言检测和持久化存储
- ✅ 创建了语言切换组件和设置面板

### 2. 枚举多语言支持
- ✅ 分析了所有需要国际化的枚举类型
- ✅ 创建了后端多语言服务，提供所有枚举值的翻译
- ✅ 实现了Wails绑定，前端可调用获取枚举翻译

### 3. 用户界面集成
- ✅ 在设置页面添加了语言设置区域
- ✅ 创建了多语言测试组件用于验证功能
- ✅ 实现了实时语言切换功能

## 待完成工作

### 1. 组件国际化改造
需要逐步改造现有React组件，将硬编码文本替换为多语言键值：

```typescript
// 改造前
<span>游戏状态</span>

// 改造后
import { useTranslation } from 'react-i18next';
const { t } = useTranslation();
<span>{t('common.gameStatus')}</span>
```

### 2. 错误消息和提示文本国际化
将所有用户可见的错误消息、提示文本等改为多语言支持。

### 3. 表单标签和占位符国际化
表单字段的标签、占位符文本需要国际化。

### 4. 动态内容国际化
对于从后端API获取的动态内容，需要建立对应的翻译机制。

## 枚举类型清单

已完成多语言支持的枚举类型：

1. **GameStatus** (游戏状态)
   - not_started, playing, completed, on_hold

2. **Period** (时间段)  
   - day, week, month, all

3. **PromptType** (提示类型)
   - DEFAULT_SYSTEM, MEOW_ZAKO, STRICT_TUTOR

4. **SourceType** (来源类型)
   - local, bangumi, vndb, ymgal, dmm, eroscape, dlsite

5. **StaffRole** (工作人员角色)
   - STAFF, CV, SCENEARIO, DIRECTOR, COMPOSER, CHARA_DESIGN, CHARACTOR, SINGER, ART

6. **TaskStatus** (任务状态)
   - INITIAL, STARTED, PAUSED, COMPLETED, ERROR, CANCELED

7. **TaskType** (任务类型)
   - GAMES, CHARACTORS, STAFFS, IMAGES, RELATIONS

8. **HotkeyAction** (热键动作)
   - START_GAME, STOP_GAME, TOGGLE_PAUSE, SCREENSHOT, CUSTOM

9. **DeviceType** (设备类型)
   - KEYBOARD, DUALSENSE, DUALSHOCK4, JOYCON, XINPUT

10. **ModifierKey** (修饰键)
    - CTRL, SHIFT, ALT, WIN

11. **JoystickButton** (手柄按钮)
    - A, B, X, Y, LB, RB, LT, RT, BACK, START, LS, RS, DPAD_UP/DOWN/LEFT/RIGHT, GUIDE

## 技术实现细节

### 前端架构
```
frontend/src/
├── i18n/
│   ├── locales/
│   │   ├── zh-CN.json  # 中文语言包
│   │   ├── en-US.json  # 英文语言包
│   │   └── ja-JP.json  # 日文语言包
│   └── i18n.ts         # i18n配置文件
├── components/
│   └── ui/
│       ├── LanguageSwitcher.tsx    # 语言切换组件
│       └── I18nTestComponent.tsx   # 多语言测试组件
└── routes/
    └── settings.tsx                # 包含语言设置面板
```

### 后端架构
```
internal/service/
└── i18n_service.go     # 提供枚举翻译的Wails服务
```

### 使用方法

1. **在组件中使用翻译**：
```typescript
import { useTranslation } from 'react-i18next';

function MyComponent() {
  const { t } = useTranslation();
  
  return <span>{t('gameStatus.playing')}</span>;
}
```

2. **切换语言**：
```typescript
import { useTranslation } from 'react-i18next';

function LanguageSelector() {
  const { i18n } = useTranslation();
  
  const changeLanguage = (lng: string) => {
    i18n.changeLanguage(lng);
  };
  
  return (
    <select onChange={(e) => changeLanguage(e.target.value)} value={i18n.language}>
      <option value="zh-CN">中文</option>
      <option value="en-US">English</option>
      <option value="ja-JP">日本語</option>
    </select>
  );
}
```

3. **获取枚举翻译**（通过Wails）：
```typescript
import { GetAllEnumTranslations } from "../../wailsjs/go/service/I18nService";

// 获取所有枚举翻译
const translations = await GetAllEnumTranslations();

// 获取特定类型的枚举翻译
const gameStatusTranslations = await GetEnumTranslationsByType("gameStatus");
```

## 下一步建议

1. **优先级排序**：先改造高频使用的UI组件
2. **逐步推进**：每次改造少量组件，确保质量
3. **测试验证**：每种语言都要进行充分测试
4. **用户体验**：保持界面一致性和流畅性

这个多语言框架为LunaBox提供了完整的国际化支持，可以根据项目需求逐步完善具体的翻译内容。