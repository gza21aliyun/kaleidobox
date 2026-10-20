# CLI 命令行

KaleidoBox 自带命令行工具 `kaleidocli`（源码位于 `cmd/lunacli`）。

> **注意**：除 `version` 和彩蛋命令外，所有命令都通过 IPC 转发给**正在运行的 GUI 进程**执行。GUI 未启动时会提示先启动应用。

## 命令一览

| 命令 | 说明 |
|------|------|
| `kaleidocli start <game>` | 启动游戏 |
| `kaleidocli list` | 列出游戏库全部游戏 |
| `kaleidocli detail <game>` | 显示单个游戏详情 |
| `kaleidocli game_entity <game>` | 输出游戏完整实体（含角色、声优、staff）的 JSON |
| `kaleidocli backup --game <game>` | 为指定游戏创建存档备份 |
| `kaleidocli version` | 打印版本号、commit、构建时间、构建模式 |
| `kaleidocli luna-sama` | 彩蛋命令 |

## 游戏参数解析

`<game>` 参数支持多种方式定位游戏（按顺序尝试）：

1. 游戏 ID 精确匹配
2. 游戏 ID 前缀匹配（唯一命中才通过）
3. 名称精确匹配
4. 名称模糊匹配

多匹配时会列出候选列表，需要更精确的输入。

## start

```
kaleidocli start <game> [--le] [--magpie]
```

- `--le` / `-l` —— 使用 Locale Emulator 启动（转区）
- `--magpie` / `-m` —— 使用 Magpie 放大启动

仅显式传入时覆盖游戏默认设置。

## list

```
kaleidocli list
```

以表格输出短 ID + 名称 + 状态图标。

## detail

```
kaleidocli detail <game>
```

显示状态、来源、公司、路径、存档路径、进程名、LE/Magpie 开关、简介等。

## game_entity

```
kaleidocli game_entity <game>
```

输出整个游戏实体的 JSON 字符串（含 WorksMap 中按角色分组的角色/声优/staff 信息），适合脚本处理。

## backup

```
kaleidocli backup --game <game>
```

为指定游戏创建一份存档备份。
