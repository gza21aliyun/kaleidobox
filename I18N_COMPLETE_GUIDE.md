# LunaBox 多语言化完整指南

## 当前进度

✅ **已完成**
- 基础i18n框架搭建
- 三种语言支持（中文、英文、日文）
- 首页组件多语言化
- 路由组件批量改造
- 基础语言包内容

🔄 **进行中**
- 完善语言包内容
- UI组件多语言化

## 下一步工作

### 1. 手动检查和修复批量改造结果

由于自动脚本可能存在不准确的替换，请手动检查以下文件：
- `frontend/src/routes/library.tsx`
- `frontend/src/routes/stats.tsx` 
- `frontend/src/routes/categories.tsx`
- 等其他路由文件

**常见问题**：
- 属性值中的文本被错误替换：`title="确认"` → `title={t('common.confirm')}`
- JSX表达式中的文本：`{isLoading ? "加载中" : "完成"}` → `{isLoading ? t('common.loading') : t('common.success')}`

### 2. 继续改造UI组件

**优先级高的组件**：
```bash
# 面板组件
frontend/src/components/panel/*.tsx

# 卡片组件  
frontend/src/components/card/*.tsx

# 模态框组件
frontend/src/components/modal/*.tsx

# 导航栏组件
frontend/src/components/bar/*.tsx
```

**改造模板**：
```typescript
// 改造前
import React from 'react';

function MyComponent() {
  return <button>确认</button>;
}

// 改造后
import React from 'react';
import { useTranslation } from 'react-i18next';

function MyComponent() {
  const { t } = useTranslation();
  return <button>{t('common.confirm')}</button>;
}
```

### 3. 完善语言包

根据实际使用情况补充翻译内容：

```json
{
  "componentName": {
    "specificKey": "翻译文本",
    "withVariables": "{{variable}} 的翻译"
  }
}
```

### 4. 测试验证

**测试清单**：
- [ ] 语言切换功能正常
- [ ] 所有页面文本正确翻译
- [ ] 动态内容（如游戏名称）不受影响
- [ ] 表单验证消息正确显示
- [ ] 错误提示信息正确翻译

## 开发工具推荐

### VS Code 插件
- **i18n Ally** - 提供翻译键值的智能提示和管理
- **ES7+ React/Redux/React-Native snippets** - 快速插入useTranslation代码

### 命令行工具
```bash
# 查找未翻译的中文文本
grep -r "['\"'][\u4e00-\u9fa5]+['\"]" frontend/src/

# 查找已有的翻译调用
grep -r "t(" frontend/src/
```

## 最佳实践

### 1. 命名规范
```
namespace.specificKey
├── common.*          # 通用词汇
├── nav.*            # 导航相关
├── home.*           # 首页相关  
├── library.*        # 游戏库相关
└── component.*      # 组件特定
```

### 2. 变量插值
```typescript
// 好的做法
t('game.playTime', { hours: 5 })

// 语言包定义
"playTime": "游玩时间: {{hours}} 小时"
```

### 3. 复数处理
```typescript
// 使用count参数
t('games.count', { count: games.length })
```

## 常见问题解决

### Q: 文本显示为翻译键值而不是翻译内容？
A: 检查语言包中是否存在对应键值，确保i18n配置正确

### Q: 语言切换后部分文本没有更新？
A: 确保组件正确使用了useTranslation hook，检查是否需要强制刷新

### Q: 如何处理从后端获取的动态内容？
A: 动态内容通常不需要翻译，保持原文显示即可

这个多语言化项目为LunaBox提供了完整的国际化支持框架，后续可以根据实际需求逐步完善具体的翻译内容。