package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"lunabox/internal/appconf"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"lunabox/internal/utils"
	"lunabox/internal/vo"
	"strings"
	"sync"
	"time"

	"encoding/json"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type GameService struct {
	ctx         context.Context
	db          *sql.DB
	config      *appconf.AppConfig
	taskService *TaskService // 添加任务服务引用
}

func NewGameService() *GameService {
	return &GameService{}
}

func (s *GameService) Init(ctx context.Context, db *sql.DB, config *appconf.AppConfig) {
	s.ctx = ctx
	s.db = db
	s.config = config
}

func (s *GameService) SelectGameExecutable() (string, error) {
	selection, err := runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "Select Game Executable",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Executables",
				Pattern:     "*.exe;*.bat;*.cmd;*.lnk",
			},
			{
				DisplayName: "All Files",
				Pattern:     "*.*",
			},
		},
	})
	if err != nil {
		runtime.LogErrorf(s.ctx, "failed to open file dialog: %v", err)
	}
	return selection, err
}

func (s *GameService) AddGame(game models.Game) error {
	if game.ID == "" {
		game.ID = uuid.New().String()
	}

	if game.CreatedAt.IsZero() {
		game.CreatedAt = time.Now()
	}

	if game.UpdatedAt.IsZero() {
		game.UpdatedAt = time.Now()
	}

	if game.CachedAt.IsZero() {
		game.CachedAt = time.Now()
	}

	// 保存原始封面URL用于后台下载
	originalCoverURL := game.CoverURL

	// 处理临时封面图片
	if strings.Contains(game.CoverURL, "/local/covers/temp_") {
		newCoverURL, err := utils.RenameTempCover(game.CoverURL, game.ID)
		if err != nil {
			runtime.LogWarningf(s.ctx, "AddGame: failed to rename temp cover: %v", err)
		} else {
			game.CoverURL = newCoverURL
			originalCoverURL = ""
		}
	}

	query := `INSERT INTO games (
		id, name, cover_url, company, summary, path, 
		source_type, cached_at, source_id, created_at, updated_at,
		tags, meta_tags, images, bangumi_id, dmm_id, eroscape_id, ymgal_id, charactors, staffs, release_at, related_games, 
		use_locale_emulator, use_magpie
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(s.ctx, query,
		game.ID,
		game.Name,
		game.CoverURL,
		game.Company,
		game.Summary,
		game.Path,
		string(game.SourceType),
		game.CachedAt,
		game.SourceID,
		game.CreatedAt,
		game.UpdatedAt,
		game.Tags,
		game.MetaTags,
		game.Images,
		game.BangumiId,
		game.DmmId,
		game.EroscapeId,
		game.YmgalId,
		game.Charactors,
		game.Staffs,
		game.ReleaseAt,
		game.RelatedGames,

		game.UseLocaleEmulator,
		game.UseMagpie,
	)

	if err != nil {
		runtime.LogErrorf(s.ctx, "AddGame: failed to insert game %s: %v", game.Name, err)
		return err
	}

	// 后台异步下载封面图片（不阻塞添加流程）
	if originalCoverURL != "" {
		go s.asyncDownloadCoverImage(game.ID, game.Name, originalCoverURL)
	}

	return nil
}

// asyncDownloadCoverImage 后台异步下载封面图片并更新数据库
func (s *GameService) asyncDownloadCoverImage(gameID, gameName, coverURL string) {
	// 检查是否为远程URL
	if coverURL == "" || !strings.HasPrefix(coverURL, "http") || strings.Contains(coverURL, "wails.localhost") {
		return
	}

	runtime.LogInfof(s.ctx, "asyncDownloadCoverImage: downloading cover for %s", gameName)

	// 下载并保存图片
	localPath, err := utils.DownloadAndSaveCoverImage(coverURL, gameID)
	if err != nil {
		runtime.LogWarningf(s.ctx, "asyncDownloadCoverImage: failed to download cover for %s: %v", gameName, err)
		return
	}

	// 更新数据库中的封面路径
	if err := s.updateCoverURL(gameID, localPath); err != nil {
		runtime.LogErrorf(s.ctx, "asyncDownloadCoverImage: failed to update cover URL for %s: %v", gameName, err)
		return
	}

	runtime.LogInfof(s.ctx, "asyncDownloadCoverImage: successfully cached cover for %s", gameName)
}

// updateCoverURL 更新游戏的封面URL
func (s *GameService) updateCoverURL(gameID, coverURL string) error {
	query := `UPDATE games SET cover_url = ? WHERE id = ?`
	_, err := s.db.ExecContext(s.ctx, query, coverURL, gameID)
	return err
}

func (s *GameService) DeleteGame(id string) error {
	// 先删除关联的游戏分类记录
	_, err := s.db.ExecContext(s.ctx, "DELETE FROM game_categories WHERE game_id = ?", id)
	if err != nil {
		runtime.LogErrorf(s.ctx, "DeleteGame: failed to delete game_categories for id %s: %v", id, err)
		return fmt.Errorf("failed to delete game categories: %w", err)
	}

	// 删除关联的游玩会话记录
	_, err = s.db.ExecContext(s.ctx, "DELETE FROM play_sessions WHERE game_id = ?", id)
	if err != nil {
		runtime.LogErrorf(s.ctx, "DeleteGame: failed to delete play_sessions for id %s: %v", id, err)
		return fmt.Errorf("failed to delete play sessions: %w", err)
	}
	// 删除游戏记录
	result, err := s.db.ExecContext(s.ctx, "DELETE FROM games WHERE id = ?", id)
	if err != nil {
		runtime.LogErrorf(s.ctx, "DeleteGame: failed to delete game for id %s: %v", id, err)
		return fmt.Errorf("failed to delete game: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		runtime.LogErrorf(s.ctx, "DeleteGame: failed to get rows affected for id %s: %v", id, err)
		return err
	}

	if rowsAffected == 0 {
		runtime.LogWarningf(s.ctx, "DeleteGame: game not found with id: %s", id)
		return fmt.Errorf("game not found with id: %s", id)
	}

	return nil
}

func (s *GameService) GetGames() ([]models.Game, error) {
	query := `SELECT 
		id, name, 
		COALESCE(cover_url, '') as cover_url, 
		COALESCE(company, '') as company, 
		COALESCE(summary, '') as summary, 
		COALESCE(path, '') as path, 
		COALESCE(save_path, '') as save_path,
		COALESCE(status, 'not_started') as status,
		COALESCE(source_type, '') as source_type, 
		cached_at, 
		COALESCE(source_id, '') as source_id, 
		created_at,
		updated_at,
		COALESCE(tags, '') as tags,
		COALESCE(meta_tags, '') as meta_tags,
		COALESCE(images, '') as images,
		COALESCE(bangumi_id, '') as bangumi_id,
		COALESCE(dmm_id, '') as dmm_id,
		COALESCE(eroscape_id, '') as eroscape_id,
		COALESCE(ymgal_id, '') as ymgal_id,
		COALESCE(charactors, '') as charactors,
		COALESCE(staffs, '') as staffs,
		COALESCE(release_at, '') as release_at,
		COALESCE(related_games, '') as related_games,
		COALESCE(use_locale_emulator, FALSE) as use_locale_emulator,
		COALESCE(use_magpie, FALSE) as use_magpie
	FROM games 
	ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(s.ctx, query)
	if err != nil {
		runtime.LogErrorf(s.ctx, "GetGames: failed to query games: %v", err)
		log.Println("GetGames: failed to query games:", err)
		return nil, fmt.Errorf("failed to query games: %w", err)
	}
	defer rows.Close()

	var games []models.Game
	for rows.Next() {
		var game models.Game
		var sourceType string
		var status string

		err := rows.Scan(
			&game.ID,
			&game.Name,
			&game.CoverURL,
			&game.Company,
			&game.Summary,
			&game.Path,
			&game.SavePath,
			&status,
			&sourceType,
			&game.CachedAt,
			&game.SourceID,
			&game.CreatedAt,
			&game.UpdatedAt,
			&game.Tags,
			&game.MetaTags,
			&game.Images,
			&game.BangumiId,
			&game.DmmId,
			&game.EroscapeId,
			&game.YmgalId,
			&game.Charactors,
			&game.Staffs,
			&game.ReleaseAt,
			&game.RelatedGames,
			&game.UseLocaleEmulator,
			&game.UseMagpie,
		)
		if err != nil {
			runtime.LogErrorf(s.ctx, "GetGames: failed to scan game row: %v", err)
			return nil, fmt.Errorf("failed to scan game: %w", err)
		}

		game.SourceType = enums.SourceType(sourceType)
		game.Status = enums.GameStatus(status)
		games = append(games, game)
	}

	if err = rows.Err(); err != nil {
		runtime.LogErrorf(s.ctx, "GetGames: error iterating games: %v", err)
		return nil, fmt.Errorf("error iterating games: %w", err)
	}

	return games, nil
}

func (s *GameService) GetGameByID(id string) (models.Game, error) {
	query := `SELECT 
		id, name, 
		COALESCE(cover_url, '') as cover_url, 
		COALESCE(company, '') as company, 
		COALESCE(summary, '') as summary, 
		COALESCE(path, '') as path, 
		COALESCE(save_path, '') as save_path,
		COALESCE(status, 'not_started') as status,
		COALESCE(source_type, '') as source_type, 
		cached_at, 
		COALESCE(source_id, '') as source_id, 
		created_at,
		updated_at,
		COALESCE(tags, '') as tags, 
		COALESCE(meta_tags, '') as meta_tags, 
		COALESCE(images, '') as images,
		COALESCE(bangumi_id, '') as bangumi_id,
		COALESCE(dmm_id, '') as dmm_id,
		COALESCE(eroscape_id, '') as eroscape_id,
		COALESCE(ymgal_id, '') as ymgal_id,
		COALESCE(charactors, '') as charactors,
		COALESCE(staffs, '') as staffs,
		COALESCE(release_at, '') as release_at,
		COALESCE(related_games, '') as related_games,
		COALESCE(use_locale_emulator, FALSE) as use_locale_emulator,
		COALESCE(use_magpie, FALSE) as use_magpie
	FROM games 
	WHERE id = ?`

	var game models.Game
	var sourceType string
	var status string

	err := s.db.QueryRowContext(s.ctx, query, id).Scan(
		&game.ID,
		&game.Name,
		&game.CoverURL,
		&game.Company,
		&game.Summary,
		&game.Path,
		&game.SavePath,
		&status,
		&sourceType,
		&game.CachedAt,
		&game.SourceID,
		&game.CreatedAt,
		&game.UpdatedAt,
		&game.Tags,
		&game.MetaTags,
		&game.Images,
		&game.BangumiId,
		&game.DmmId,
		&game.EroscapeId,
		&game.YmgalId,
		&game.Charactors,
		&game.Staffs,
		&game.ReleaseAt,
		&game.RelatedGames,
		&game.UseLocaleEmulator,
		&game.UseMagpie,
	)

	if errors.Is(err, sql.ErrNoRows) {
		runtime.LogWarningf(s.ctx, "GetGameByID: game not found with id: %s", id)
		return models.Game{}, fmt.Errorf("game not found with id: %s", id)
	}
	if err != nil {
		runtime.LogErrorf(s.ctx, "GetGameByID: failed to query game %s: %v", id, err)
		return models.Game{}, fmt.Errorf("failed to query game: %w", err)
	}

	game.SourceType = enums.SourceType(sourceType)
	game.Status = enums.GameStatus(status)
	return game, nil
}

func (s *GameService) UpdateGame(game models.Game) error {
	query := `UPDATE games SET 
		name = ?,
		cover_url = ?,
		company = ?,
		summary = ?,
		path = ?,
		save_path = ?,
		status = ?,
		source_type = ?,
		cached_at = ?,
		source_id = ?,
		tags = ?,
		images = ?,
		bangumi_id = ?,
		dmm_id = ?,
		eroscape_id = ?,
		ymgal_id = ?,
		charactors = ?,
		staffs = ?,
		release_at = ?,
		related_games = ?,
		use_locale_emulator = ?,
		use_magpie = ?
	WHERE id = ?`

	result, err := s.db.ExecContext(s.ctx, query,
		game.Name,
		game.CoverURL,
		game.Company,
		game.Summary,
		game.Path,
		game.SavePath,
		string(game.Status),
		string(game.SourceType),
		game.CachedAt,
		game.SourceID,
		game.Tags,
		game.Images,
		game.BangumiId,
		game.DmmId,
		game.EroscapeId,
		game.YmgalId,
		game.Charactors,
		game.Staffs,
		game.ReleaseAt,
		game.RelatedGames,
		game.UseLocaleEmulator,
		game.UseMagpie,
		game.ID,
	)

	if err != nil {
		runtime.LogErrorf(s.ctx, "UpdateGame: failed to update game %s: %v", game.ID, err)
		return fmt.Errorf("failed to update game: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		runtime.LogErrorf(s.ctx, "UpdateGame: failed to get rows affected for id %s: %v", game.ID, err)
		return err
	}

	if rowsAffected == 0 {
		runtime.LogWarningf(s.ctx, "UpdateGame: game not found with id: %s", game.ID)
		return fmt.Errorf("game not found with id: %s", game.ID)
	}

	return nil
}

// SelectSaveDirectory 选择存档目录
func (s *GameService) SelectSaveDirectory() (string, error) {
	selection, err := runtime.OpenDirectoryDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "选择存档目录",
	})
	return selection, err
}

// SelectCoverImage 选择封面图片并保存到 covers 目录
func (s *GameService) SelectCoverImage(gameID string) (string, error) {
	selection, err := runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "选择封面图片",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "图片文件",
				Pattern:     "*.png;*.jpg;*.jpeg;*.gif;*.webp;*.bmp",
			},
		},
	})
	if err != nil {
		runtime.LogErrorf(s.ctx, "failed to open file dialog: %v", err)
		return "", err
	}
	if selection == "" {
		return "", nil
	}

	coverPath, err := utils.SaveCoverImage(selection, gameID)
	if err != nil {
		runtime.LogErrorf(s.ctx, "failed to save cover image: %v", err)
		return "", fmt.Errorf("failed to save cover image: %w", err)
	}

	return coverPath, nil
}

// SelectCoverImageWithTempID 选择封面图片并使用临时ID保存（用于新增游戏时）
func (s *GameService) SelectCoverImageWithTempID() (string, error) {
	selection, err := runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "选择封面图片",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "图片文件",
				Pattern:     "*.png;*.jpg;*.jpeg;*.gif;*.webp;*.bmp",
			},
		},
	})
	if err != nil {
		runtime.LogErrorf(s.ctx, "failed to open file dialog: %v", err)
		return "", err
	}
	if selection == "" {
		return "", nil
	}

	// 使用时间戳作为临时ID
	tempID := fmt.Sprintf("temp_%d", time.Now().UnixNano())
	coverPath, err := utils.SaveCoverImage(selection, tempID)
	if err != nil {
		runtime.LogErrorf(s.ctx, "failed to save cover image: %v", err)
		return "", fmt.Errorf("failed to save cover image: %w", err)
	}

	return coverPath, nil
}

func (s *GameService) FetchMetadataByName(name string) ([]vo.GameMetadataFromWebVO, error) {
	var games []vo.GameMetadataFromWebVO
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 这里暂不处理任何错误，直接尝试从多个来源并发获取数据，空就是网络问题或未找到，不管它
	wg.Add(5)

	go func() {
		defer wg.Done()
		bgmGetter := utils.NewBangumiInfoGetter()
		bgm, _ := bgmGetter.FetchMetadataByName(name, s.config.BangumiAccessToken)
		if bgm != (models.Game{}) {
			mu.Lock()
			games = append(games, vo.GameMetadataFromWebVO{Source: enums.Bangumi, Game: bgm})
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		vndbGetter := utils.NewVNDBInfoGetter()
		vndb, _ := vndbGetter.FetchMetadataByName(name, s.config.VNDBAccessToken)
		if vndb != (models.Game{}) {
			mu.Lock()
			games = append(games, vo.GameMetadataFromWebVO{Source: enums.VNDB, Game: vndb})
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		ymgalGetter := utils.NewYmgalInfoGetter()
		ymgal, _ := ymgalGetter.FetchMetadataByName(name, "")
		if ymgal != (models.Game{}) {
			mu.Lock()
			games = append(games, vo.GameMetadataFromWebVO{Source: enums.Ymgal, Game: ymgal})
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		dmmGetter := utils.NewDmmInfoGetter()
		dmm, _ := dmmGetter.FetchMetadataByName(name, s.config.DmmIsEnabled)
		if dmm != (models.Game{}) {
			mu.Lock()
			games = append(games, vo.GameMetadataFromWebVO{Source: enums.Dmm, Game: dmm})
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		eroscapeGetter := utils.NewEroscapeInfoGetter()
		eroscape, _ := eroscapeGetter.FetchMetadataByName(name, s.config.EroscapeIsEnabled, s.config.EroscapeUseMirror)
		if eroscape != (models.Game{}) {
			mu.Lock()
			games = append(games, vo.GameMetadataFromWebVO{Source: enums.Eroscape, Game: eroscape})
			mu.Unlock()
		}
	}()

	wg.Wait()

	return games, nil
}

func (s *GameService) FetchMetadata(req vo.MetadataRequest) (models.Game, error) {
	var game = models.Game{}
	var e error

	if game, e = fetchFromLocal(req.ID); e == nil {
		return game, nil
	}

	switch req.Source {
	case enums.Bangumi:
		bgmGetter := utils.NewBangumiInfoGetter()
		game, e = bgmGetter.FetchMetadata(req.ID, s.config.BangumiAccessToken)
	case enums.VNDB:
		vndbGetter := utils.NewVNDBInfoGetter()
		game, e = vndbGetter.FetchMetadata(req.ID, s.config.VNDBAccessToken)
	case enums.Ymgal:
		ymgalGetter := utils.NewYmgalInfoGetter()
		game, e = ymgalGetter.FetchMetadata(req.ID, "")
	case enums.Eroscape:
		escGetter := utils.NewEroscapeInfoGetter()
		game.EroscapeId = req.ID
		game, e = escGetter.FetchMetadataById(game, s.config.EroscapeUseMirror)
	case enums.Dmm:
		dmmGetter := utils.NewDmmInfoGetter()
		game.DmmId = req.ID
		game, e = dmmGetter.FetchMetadataById(game)
	}
	return game, e
}

func fetchFromLocal(id string) (models.Game, error) {
	// 这个函数暂时返回错误，表示未实现从本地数据库获取
	// 如果需要实现，应该在这里查询数据库
	return models.Game{}, fmt.Errorf("game not found in local cache")
}

// UpdateGameFromRemote 从远程数据源更新游戏信息
func (s *GameService) UpdateGameFromRemote(gameID string) error {
	// 获取现有游戏信息
	existingGame, err := s.GetGameByID(gameID)
	if err != nil {
		return fmt.Errorf("failed to get game: %w", err)
	}

	if existingGame.SourceType == "" || existingGame.SourceID == "" {
		return fmt.Errorf("游戏缺少数据源信息，无法从远程更新")
	}

	// 从远程获取最新数据
	req := vo.MetadataRequest{
		Source: existingGame.SourceType,
		ID:     existingGame.SourceID,
	}

	remoteGame, err := s.FetchMetadata(req)
	if err != nil {
		return fmt.Errorf("failed to fetch metadata from remote: %w", err)
	}

	// 保留本地重要字段，更新远程可获取的字段
	existingGame.Name = remoteGame.Name
	existingGame.Company = remoteGame.Company
	existingGame.Summary = remoteGame.Summary
	existingGame.CachedAt = time.Now()

	existingGame.CoverURL = remoteGame.CoverURL
	if remoteGame.CoverURL != "" {
		go s.asyncDownloadCoverImage(existingGame.ID, existingGame.Name, remoteGame.CoverURL)
	}

	if err := s.UpdateGame(existingGame); err != nil {
		return fmt.Errorf("failed to update game: %w", err)
	}

	runtime.LogInfof(s.ctx, "UpdateGameFromRemote: successfully updated game %s from %s", existingGame.Name, existingGame.SourceType)
	return nil
}

func (s *GameService) UpdateGamesBackground(games []models.Game, source enums.SourceType, id string) error {
	var uuid = uuid.New().String()
	s.taskService.RegisterTaskFunction(uuid, s.createGameUpdateTaskFunction())
	taskData := map[string]interface{}{
		"games":  games,
		"source": source,
	}
	return s.taskService.StartTask("game_updates", uuid, 1000, enums.Games, len(games), taskData)
	// return nil
}

// 设置任务服务引用
func (s *GameService) SetTaskService(taskService *TaskService) {
	s.taskService = taskService
	// 注册游戏更新任务函数
}

func mergeStrings(tagStr1, tagStr2 string) string {
	// 分割字符串为数组
	tags1 := strings.Split(tagStr1, ",")
	tags2 := strings.Split(tagStr2, ",")

	// 使用map去重
	tagSet := make(map[string]struct{})

	for _, tags := range [][]string{tags1, tags2} {
		for _, tag := range tags {
			trimmedTag := strings.TrimSpace(tag)
			if trimmedTag != "" {
				tagSet[trimmedTag] = struct{}{}
			}
		}
	}

	// 转换为结果数组
	uniqueTags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		uniqueTags = append(uniqueTags, tag)
	}

	// 重新组合为字符串
	return strings.Join(uniqueTags, ",")
}

// 创建游戏更新任务函数
func (s *GameService) createGameUpdateTaskFunction() TaskFunction {
	return func(ctx context.Context, data json.RawMessage, updateProgress func(completed int, total int,
		workingOn string, warning string, itemId string, itemEvent enums.TaskStatus, itemData interface{})) error {
		// 定义结构来解组任务数据
		var taskData struct {
			Games  []models.Game    `json:"games"`
			Source enums.SourceType `json:"source"`
		}

		if err := json.Unmarshal(data, &taskData); err != nil {
			return fmt.Errorf("解析任务数据失败: %v", err)
		}
		updateProgress(0, len(taskData.Games), fmt.Sprintf("开始更新游戏: "),
			"", "", enums.Started, nil)

		// 实现UpdateGamesBackground的核心逻辑
		for index, ngame := range taskData.Games {
			// 检查是否被取消（通过上下文检查）
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			updateProgress(index, len(taskData.Games), fmt.Sprintf("更新游戏: %s", ngame.Name),
				"", ngame.ID, enums.Initial, nil)

			var id = ""

			if taskData.Source == ngame.SourceType {
				id = ngame.SourceID
				log.Printf("TaskFunc 01 id found 11 for game %s, id: %s", ngame.Name, id)
			} else if taskData.Source == enums.Eroscape && strings.TrimSpace(ngame.EroscapeId) != "" {
				id = ngame.EroscapeId
				log.Printf("TaskFunc 02 id found 12 for game %s, id: %s", ngame.Name, id)
			} else if taskData.Source == enums.Ymgal && strings.TrimSpace(ngame.YmgalId) != "" {
				id = ngame.YmgalId
				log.Printf("TaskFunc 03 id found 13 for game %s, id: %s", ngame.Name, id)
			} else if taskData.Source == enums.Dmm && strings.TrimSpace(ngame.DmmId) != "" {
				id = ngame.DmmId
				log.Printf("TaskFunc 04 id found 14 for game %s, id: %s", ngame.Name, id)
			}

			var updatedGame models.Game
			var err error

			log.Printf("TaskFunc 11 fetch metadata 01 for game %s, id: %s, source: %v", ngame.Name, id, taskData.Source)
			if strings.TrimSpace(id) != "" {
				log.Printf("TaskFunc 12 fetch metadata 02 for game %s", ngame.Name)
				// 通过ID获取元数据
				req := vo.MetadataRequest{
					Source: taskData.Source,
					ID:     id,
				}
				updatedGame, err = s.FetchMetadata(req)
				if err != nil {

					log.Printf("Failed to fetch metadata for game %s by ID: %v", ngame.Name, err)
					updateProgress(index, len(taskData.Games), "", fmt.Sprintf("Failed to fetch metadata for game %s by ID: %v", ngame.Name, err),
						ngame.ID, enums.Error, nil)
					continue
				}
			} else {
				log.Printf("TaskFunc 13 fetch metadata 03 for game %s", ngame.Name)
				// 通过名称获取元数据
				if taskData.Source == enums.Bangumi {
					log.Printf("TaskFunc 21 fetch for game %s", ngame.Name)
					bgmGetter := utils.NewBangumiInfoGetter()
					updatedGame, _ = bgmGetter.FetchMetadataByName(ngame.Name, s.config.BangumiAccessToken)
				} else if taskData.Source == enums.VNDB {
					log.Printf("TaskFunc 22 fetch for game %s", ngame.Name)
					vndbGetter := utils.NewVNDBInfoGetter()
					updatedGame, _ = vndbGetter.FetchMetadataByName(ngame.Name, s.config.VNDBAccessToken)
				} else if taskData.Source == enums.Ymgal {
					log.Printf("TaskFunc 23 fetch for game %s", ngame.Name)
					ymgalGetter := utils.NewYmgalInfoGetter()
					updatedGame, _ = ymgalGetter.FetchMetadataByName(ngame.Name, "")
				} else if taskData.Source == enums.Eroscape {
					log.Printf("TaskFunc 24 fetch for game %s", ngame.Name)
					escGetter := utils.NewEroscapeInfoGetter()
					esc, _ := escGetter.FetchMetadataByName(ngame.Name, true, s.config.EroscapeUseMirror)
					updatedGame = esc
				} else if taskData.Source == enums.Dmm {
					log.Printf("TaskFunc 25 fetch for game %s", ngame.Name)
					dmmGetter := utils.NewDmmInfoGetter()
					dmm, _ := dmmGetter.FetchMetadataByName(ngame.Name, true)
					updatedGame = dmm
				} else {
					log.Printf("TaskFunc 26 fetch for game %s", ngame.Name)
					return errors.New("未知的来源 ")
				}
			}
			log.Printf("TaskFunc 31 fetch for game %s, id:%s", ngame.Name, updatedGame.SourceID)

			// updatedGame.ID = ngame.ID
			// updatedGame.Path = ngame.Path
			// if updatedGame.BangumiId == "" {
			// 	updatedGame.BangumiId = ngame.BangumiId
			// }
			// if updatedGame.DmmId == "" {
			// 	updatedGame.DmmId = ngame.DmmId
			// }
			// if updatedGame.EroscapeId == "" {
			// 	updatedGame.EroscapeId = ngame.EroscapeId
			// }

			// updatedGame.CreatedAt = ngame.CreatedAt
			// updatedGame.SourceType = ngame.SourceType
			// updatedGame.SourceID = ngame.SourceID
			// updatedGame.CachedAt = time.Now()
			// updatedGame.Tags = mergeStrings(updatedGame.Tags, ngame.Tags)
			// updatedGame.Charactors = mergeStrings(updatedGame.Charactors, ngame.Charactors)
			// updatedGame.Staffs = mergeStrings(updatedGame.Staffs, ngame.Staffs)
			// updatedGame.Images = mergeStrings(updatedGame.Images, ngame.Images)

			// updatedGame.SavePath = ngame.SavePath
			// updatedGame.ReleaseAt = ngame.ReleaseAt
			// updatedGame.Status = ngame.Status
			// updatedGame.Summary = ngame.Summary
			// updatedGame.UseMagpie = ngame.UseMagpie
			// updatedGame.UseLocaleEmulator = ngame.UseLocaleEmulator
			s.FillGame(&ngame, &updatedGame)

			// 更新游戏
			if err := s.UpdateGame(updatedGame); err != nil {
				log.Printf("Failed to update game %s: %v", updatedGame.Name, err)
				continue
			}
			updateProgress(index, len(taskData.Games), "", fmt.Sprintf("complete for game %s by ID: %v", ngame.Name, err),
				ngame.ID, enums.Completed, updatedGame)

		}

		// 标记完成
		updateProgress(len(taskData.Games), len(taskData.Games), "所有游戏更新完成", "", "", enums.Completed, nil)
		return nil
	}
}

func (s *GameService) FillGame(ngame *models.Game, updatedGame *models.Game) {
	updatedGame.ID = ngame.ID
	updatedGame.Path = ngame.Path
	if updatedGame.BangumiId == "" {
		updatedGame.BangumiId = ngame.BangumiId
	}
	if updatedGame.DmmId == "" {
		updatedGame.DmmId = ngame.DmmId
	}
	if updatedGame.EroscapeId == "" {
		updatedGame.EroscapeId = ngame.EroscapeId
	}
	if updatedGame.YmgalId == "" {
		updatedGame.YmgalId = ngame.YmgalId
	}
	if updatedGame.Name == "" {
		updatedGame.Name = ngame.Name
	}

	updatedGame.CreatedAt = ngame.CreatedAt
	updatedGame.SourceType = ngame.SourceType
	updatedGame.SourceID = ngame.SourceID
	updatedGame.CachedAt = time.Now()
	updatedGame.Tags = mergeStrings(updatedGame.Tags, ngame.Tags)
	updatedGame.Charactors = mergeStrings(updatedGame.Charactors, ngame.Charactors)
	updatedGame.Staffs = mergeStrings(updatedGame.Staffs, ngame.Staffs)
	updatedGame.Images = mergeStrings(updatedGame.Images, ngame.Images)

	updatedGame.SavePath = ngame.SavePath
	updatedGame.ReleaseAt = ngame.ReleaseAt
	updatedGame.Status = ngame.Status
	updatedGame.Summary = ngame.Summary
	updatedGame.UseMagpie = ngame.UseMagpie
	updatedGame.UseLocaleEmulator = ngame.UseLocaleEmulator
}

// 新增方法：启动游戏批量更新任务
// func (s *GameService) StartGameUpdateTask(name string, games []models.Game, sourceText string) error {
// 	var source enums.SourceType = enums.SourceType(sourceText)
// 	id := uuid.New().String()
// 	s.taskService.RegisterTaskFunction(id, s.createGameUpdateTaskFunction())
// 	taskData := map[string]interface{}{
// 		"games":  games,
// 		"source": source,
// 	}

// 	return s.taskService.StartTask(name, id, 1000, enums.Games, len(games), taskData)
// }
