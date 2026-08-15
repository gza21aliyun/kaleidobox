package service

import (
	"context"
	"fmt"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"os"
	"path/filepath"
	"strings"

	"encoding/json"

	"github.com/google/uuid"
)

// CheckGamesValidity 检查所有已导入游戏的路径有效性（后台任务）
func (s *GameService) CheckGamesValidity(games []models.Game) error {
	var uuid = uuid.New().String()
	s.taskService.RegisterTaskFunction(uuid, s.createCheckGameValidityTaskFunction())
	taskData := map[string]interface{}{
		"games": games,
	}
	// 传进来的 games 可能来自前端 store，可能为空；这里用 DB 数量作为 total 保证进度正确
	total := len(games)
	if total == 0 {
		if dbGames, err := s.GetGames(); err == nil {
			total = len(dbGames)
		}
	}
	return s.taskService.StartTask("game_updates", uuid, 0, enums.CheckGameValidity, total, taskData)
}

// createCheckGameValidityTaskFunction 创建检查游戏路径有效性的任务函数
func (s *GameService) createCheckGameValidityTaskFunction() TaskFunction {
	return func(ctx context.Context, data string, updateProgress func(completed int, total int,
		workingOn string, warning string, itemId string, itemEvent enums.TaskStatus, resultGames []models.ResultGames, itemData interface{})) error {
		var taskData struct {
			Games []models.Game `json:"games"`
		}

		if err := json.Unmarshal([]byte(data), &taskData); err != nil {
			return fmt.Errorf("解析任务数据失败: %v", err)
		}

		// 前端传进来的 games 可能为空（store 还没加载），兜底从 DB 读取全部
		games := taskData.Games
		if len(games) == 0 {
			dbGames, err := s.GetGames()
			if err != nil {
				return fmt.Errorf("读取游戏列表失败: %v", err)
			}
			games = dbGames
		}

		valid := models.ResultGames{}
		valid.Title = "游戏有效性检查"
		valid.Description = "路径有效的游戏"
		valid.Status = 200
		valid.GameIds = []string{}

		invalid := models.ResultGames{}
		invalid.Description = "路径无效的游戏"
		invalid.Status = 400
		invalid.GameIds = []string{}

		resultGames := []models.ResultGames{invalid, valid}

		total := len(games)
		updateProgress(0, total, "开始检查游戏有效性", "", "", enums.Started, resultGames, nil)

		for index, game := range games {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			updateProgress(index, total, fmt.Sprintf("检查游戏: %s", game.Name),
				"", game.ID, enums.Initial, resultGames, nil)

			if strings.TrimSpace(game.Path) == "" {
				invalid.GameIds = append(invalid.GameIds, game.ID)
				resultGames[0] = invalid
				updateProgress(index, total, fmt.Sprintf("路径为空: %s", game.Name),
					fmt.Sprintf("游戏 %s 的路径为空", game.Name), game.ID, enums.Error, resultGames, nil)
				continue
			}

			if _, err := os.Stat(game.Path); err != nil {
				invalid.GameIds = append(invalid.GameIds, game.ID)
				resultGames[0] = invalid
				updateProgress(index, total, fmt.Sprintf("路径不存在: %s", game.Name),
					fmt.Sprintf("游戏 %s 的路径不存在: %s", game.Name, game.Path), game.ID, enums.Error, resultGames, nil)
				continue
			}

			valid.GameIds = append(valid.GameIds, game.ID)
			resultGames[1] = valid
			updateProgress(index, total, fmt.Sprintf("路径有效: %s", game.Name),
				"", game.ID, enums.Completed, resultGames, game)
		}

		updateProgress(total, total, "所有游戏有效性检查完成", "", "", enums.Completed, resultGames, nil)
		return nil
	}
}

// CheckDirectoryImportState 检测指定目录指定层级的导入状态（后台任务）
// level=0 检查 rootDir 本身; level=N 检查 rootDir 下恰好第 N 层的所有子目录
func (s *GameService) CheckDirectoryImportState(rootDir string, level int) error {
	var uuid = uuid.New().String()
	s.taskService.RegisterTaskFunction(uuid, s.createCheckDirectoryImportStateTaskFunction())
	taskData := map[string]interface{}{
		"rootDir": rootDir,
		"level":   level,
	}
	return s.taskService.StartTask("game_updates", uuid, 0, enums.CheckDirectoryImportState, 0, taskData)
}

// collectDirsAtLevel 收集 rootDir 下恰好第 targetLevel 层的所有目录
// targetLevel=0 直接返回 [rootDir]
func collectDirsAtLevel(rootDir string, targetLevel int) ([]string, error) {
	cleanRoot := filepath.Clean(rootDir)
	if targetLevel == 0 {
		if info, err := os.Stat(cleanRoot); err != nil || !info.IsDir() {
			return nil, fmt.Errorf("目录不存在或不是文件夹: %s", cleanRoot)
		}
		return []string{cleanRoot}, nil
	}

	dirs := []string{}
	err := filepath.Walk(cleanRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// 忽略访问错误
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(cleanRoot, path)
		if relErr != nil {
			return nil
		}
		if rel == "." {
			return nil
		}
		depth := len(strings.Split(rel, string(os.PathSeparator)))
		if depth == targetLevel {
			subEntries, _ := os.ReadDir(path)
			if len(subEntries) == 1 && subEntries[0].IsDir() {
				subPath := filepath.Join(path, subEntries[0].Name())
				dirs = append(dirs, subPath)
				return filepath.SkipDir
			}
			dirs = append(dirs, path)
			// 不再向下遍历
			return filepath.SkipDir
		}
		if depth > targetLevel {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return dirs, nil
}

// isDirEmpty 判断目录是否为空（既没有文件也没有子目录）
func isDirEmpty(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	if len(entries) == 1 {
		if !entries[0].IsDir() {
			return true
		} else {
			subEntries, err := os.ReadDir(filepath.Join(path, entries[0].Name()))
			if err != nil {
				return true
			}
			return len(subEntries) == 0
		}
	}
	return len(entries) == 0
}

func (s *GameService) createCheckDirectoryImportStateTaskFunction() TaskFunction {
	return func(ctx context.Context, data string, updateProgress func(completed int, total int,
		workingOn string, warning string, itemId string, itemEvent enums.TaskStatus, resultGames []models.ResultGames, itemData interface{})) error {
		var taskData struct {
			RootDir string `json:"rootDir"`
			Level   int    `json:"level"`
		}

		if err := json.Unmarshal([]byte(data), &taskData); err != nil {
			return fmt.Errorf("解析任务数据失败: %v", err)
		}

		fmt.Printf("[CheckDirImport] rootDir=%s, level=%d\n", taskData.RootDir, taskData.Level)

		if strings.TrimSpace(taskData.RootDir) == "" {
			return fmt.Errorf("目录路径为空")
		}
		if taskData.Level < 0 {
			return fmt.Errorf("层级不能小于 0")
		}

		emptyDirs := models.ResultGames{}
		emptyDirs.Title = "检测目录导入状态"
		emptyDirs.Description = "空文件夹"
		emptyDirs.Status = 300
		emptyDirs.GameIds = []string{}

		imported := models.ResultGames{}
		imported.Description = "已导入"
		imported.Status = 200
		imported.GameIds = []string{}

		notImported := models.ResultGames{}
		notImported.Description = "没导入"
		notImported.Status = 400
		notImported.GameIds = []string{}

		resultGames := []models.ResultGames{emptyDirs, notImported, imported}

		updateProgress(0, 0, fmt.Sprintf("扫描目录层级 %d: %s", taskData.Level, taskData.RootDir),
			"", "", enums.Started, resultGames, nil)

		dirs, err := collectDirsAtLevel(taskData.RootDir, taskData.Level)
		if err != nil {
			return err
		}
		total := len(dirs)
		if total == 0 {
			updateProgress(0, 0, "未找到任何目录", "", "", enums.Completed, resultGames, nil)
			return nil
		}

		// 提前从 DB 取全部游戏，用于路径匹配
		allGames, err := s.GetGames()
		if err != nil {
			return fmt.Errorf("读取游戏列表失败: %v", err)
		}

		// 规范化游戏路径以便比较
		type gameInfo struct {
			id        string
			cleanPath string
		}
		gameInfos := make([]gameInfo, 0, len(allGames))
		for _, g := range allGames {
			if strings.TrimSpace(g.Path) == "" {
				continue
			}
			gameInfos = append(gameInfos, gameInfo{id: g.ID, cleanPath: filepath.Clean(g.Path)})
		}

		updateProgress(0, total, fmt.Sprintf("找到 %d 个目录，开始检测", total),
			"", "", enums.Started, resultGames, nil)

		for index, dir := range dirs {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			cleanDir := filepath.Clean(dir)
			updateProgress(index, total, fmt.Sprintf("检查目录: %s", cleanDir),
				"", cleanDir, enums.Initial, resultGames, nil)

			if isDirEmpty(cleanDir) {
				emptyDirs.GameIds = append(emptyDirs.GameIds, cleanDir)
				resultGames[0] = emptyDirs
				updateProgress(index, total, fmt.Sprintf("空文件夹: %s", cleanDir),
					"", cleanDir, enums.Error, resultGames, nil)
				continue
			}

			// 判断该目录下是否有游戏已导入（game.Path 以目录为前缀或相等）
			dirPrefix := cleanDir + string(os.PathSeparator)
			found := false
			for _, gi := range gameInfos {
				if gi.cleanPath == cleanDir || strings.HasPrefix(gi.cleanPath, dirPrefix) {
					imported.GameIds = append(imported.GameIds, gi.id)
					found = true
					// 一个目录可能对应多个游戏，继续收集其他相同目录的游戏
				}
			}

			if found {
				resultGames[2] = imported
				updateProgress(index, total, fmt.Sprintf("已导入: %s", cleanDir),
					"", cleanDir, enums.Completed, resultGames, nil)
			} else {
				notImported.GameIds = append(notImported.GameIds, cleanDir)
				resultGames[1] = notImported
				updateProgress(index, total, fmt.Sprintf("没导入: %s", cleanDir),
					"", cleanDir, enums.Error, resultGames, nil)
			}
		}

		updateProgress(total, total, "目录导入状态检测完成", "", "", enums.Completed, resultGames, nil)
		return nil
	}
}
