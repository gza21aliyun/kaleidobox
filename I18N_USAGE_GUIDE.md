# LunaBox 多语言使用指南

## 快速开始

### 1. 在组件中使用翻译

```typescript
import { useTranslation } from 'react-i18next';

function MyComponent() {
  const { t } = useTranslation();
  
  return (
    <div>
      <h1>{t('common.confirm')}</h1>
      <p>{t('gameStatus.playing')}</p>
    </div>
  );
}
```

### 2. 切换语言

```typescript
import { useTranslation } from 'react-i18next';

function LanguageSwitcher() {
  const { i18n } = useTranslation();
  
  const changeLanguage = (lng: string) => {
    i18n.changeLanguage(lng);
  };
  
  return (
    <div>
      <button onClick={() => changeLanguage('zh-CN')}>中文</button>
      <button onClick={() => changeLanguage('en-US')}>English</button>
      <button onClick={() => changeLanguage('ja-JP')}>日本語</button>
    </div>
  );
}
```

## 常用翻译键值

### 通用文本
- `common.confirm` - 确认
- `common.cancel` - 取消
- `common.save` - 保存
- `common.delete` - 删除
- `common.edit` - 编辑
- `common.add` - 添加
- `common.close` - 关闭

### 枚举翻译
- `gameStatus.not_started` - 未开始
- `gameStatus.playing` - 游玩中
- `gameStatus.completed` - 已通关
- `gameStatus.on_hold` - 搁置

- `period.day` - 日
- `period.week` - 周
- `period.month` - 月
- `period.all` - 全部

## 最佳实践

### 1. 组件改造模板

```typescript
// 改造前
interface Props {
  title: string;
  confirmText?: string;
  cancelText?: string;
}

function MyModal({ title, confirmText = "确定", cancelText = "取消" }: Props) {
  return (
    <div>
      <h2>{title}</h2>
      <button>{confirmText}</button>
      <button>{cancelText}</button>
    </div>
  );
}

// 改造后
import { useTranslation } from 'react-i18next';

interface Props {
  title: string;
  confirmText?: string;
  cancelText?: string;
}

function MyModal({ title, confirmText, cancelText }: Props) {
  const { t } = useTranslation();
  
  // 使用传入的文本或默认翻译
  const finalConfirmText = confirmText || t('common.confirm');
  const finalCancelText = cancelText || t('common.cancel');
  
  return (
    <div>
      <h2>{title}</h2>
      <button>{finalConfirmText}</button>
      <button>{finalCancelText}</button>
    </div>
  );
}
```

### 2. 条件渲染中的翻译

```typescript
// 改造前
{isLoading ? "加载中..." : "加载完成"}

// 改造后
{isLoading ? t('common.loading') : t('common.success')}
```

### 3. 属性中的翻译

```typescript
// 改造前
<button title="点击查看详情">详情</button>

// 改造后
<button title={t('common.details')}>{t('common.details')}</button>
```

## 测试多语言功能

1. 访问设置页面 -> 语言设置
2. 使用语言切换器切换不同语言
3. 观察界面文本是否正确翻译
4. 检查确认对话框等组件的文本显示

## 添加新翻译

### 1. 在语言包中添加新键值

```json
// frontend/src/i18n/locales/zh-CN.json
{
  "myFeature": {
    "title": "我的功能",
    "description": "这是一个新功能"
  }
}
```

### 2. 在组件中使用

```typescript
const { t } = useTranslation();
return (
  <div>
    <h2>{t('myFeature.title')}</h2>
    <p>{t('myFeature.description')}</p>
  </div>
);
```

## 注意事项

1. **保持键值命名一致性**：使用清晰的命名空间
2. **处理复数形式**：不同语言的复数规则不同
3. **考虑RTL语言**：为从右到左的语言预留样式支持
4. **测试所有语言**：确保每种语言的显示效果都正常
5. **性能优化**：避免在渲染循环中频繁调用翻译函数