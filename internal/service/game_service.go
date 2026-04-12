package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"lunabox/internal/appconf"
	"lunabox/internal/applog"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"lunabox/internal/utils"
	"lunabox/internal/vo"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"encoding/json"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type GameService struct {
	ctx              context.Context
	db               *sql.DB
	config           *appconf.AppConfig
	taskService      *TaskService // 添加任务服务引用
	staffService     *StaffService
	charactorService *CharactorService
	workService      *WorkService
	tagService       *TagService
	imageService     *ImageService
}

// 设置任务服务引用
func (s *GameService) SetServices(taskService *TaskService, charactorService *CharactorService,
	staffService *StaffService, workService *WorkService, tagService *TagService, imageService *ImageService) {
	s.taskService = taskService
	// 注册游戏更新任务函数
	s.charactorService = charactorService
	s.staffService = staffService
	s.workService = workService
	s.tagService = tagService
	s.imageService = imageService
}

func NewGameService() *GameService {
	return &GameService{}
}

func (s *GameService) Init(ctx context.Context, db *sql.DB, config *appconf.AppConfig) {
	s.ctx = ctx
	s.db = db
	s.config = config
}

func (s *GameService) SelectFile(display, exts string) (string, error) {
	if s.config.NewFolderChooser {
		return s.SelectFile1(display, exts)
	} else {
		return s.SelectFile2(display, exts)
	}
}

func (s *GameService) SelectFile1(display, exts string) (string, error) {
	selection, err := runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "Select Game Executable",
		Filters: []runtime.FileFilter{
			{
				DisplayName: display,
				Pattern:     exts,
			},
			{
				DisplayName: "All Files",
				Pattern:     "*.*",
			},
		},
	})
	if err != nil {
		applog.LogErrorf(s.ctx, "failed to open file dialog: %v", err)
	}
	return selection, err
}

func (s *GameService) SelectFile2(display, exts string) (string, error) {
	psScript := fmt.Sprintf(`
		$PSDefaultParameterValues['Out-File:Encoding'] = 'UTF8'
		[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
		Add-Type -AssemblyName System.Windows.Forms

		$dialog = New-Object System.Windows.Forms.OpenFileDialog
		$dialog.Title = "Select Game Executable"
		$dialog.Filter = "Executables (%s)|%s|All Files (*.*)|*.*"
		$dialog.CheckFileExists = $true
		$dialog.CheckPathExists = $true

		$result = $dialog.ShowDialog()
		if ($result -eq "OK") {
			$path = $dialog.FileName
			# 直接输出路径，避免编码问题
			Write-Host $path
		}
		`, exts, exts)

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-NonInteractive", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("选择文件失败：%v", err)
	}

	selection := strings.TrimSpace(string(output))

	// 清理可能的控制字符
	selection = strings.Map(func(r rune) rune {
		if r >= 32 || r == '\n' || r == '\r' {
			return r
		}
		return -1
	}, selection)

	selection = strings.TrimSpace(selection)

	if selection == "" {
		return "", nil
	}

	return selection, nil
}

func (s *GameService) SelectGameExecutable() (string, error) {
	return s.SelectFile("Executables", "*.exe;*.bat;*.cmd;*.lnk")
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
			applog.LogWarningf(s.ctx, "AddGame: failed to rename temp cover: %v", err)
		} else {
			game.CoverURL = newCoverURL
			originalCoverURL = ""
		}
	}

	query := `INSERT INTO games (
		id, name, cover_url, company, summary, path, 
		source_type, cached_at, source_id, created_at, updated_at,
		tags, arguments, images, bangumi_id, dmm_id, eroscape_id, ymgal_id, search_name, dlsite_id, release_at, related_games, 
		use_locale_emulator, use_magpie, process_name, getchu_id, pv_path, vm_id
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

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
		game.Arguments,
		"", //game.Images,
		game.BangumiId,
		game.DmmId,
		game.EroscapeId,
		game.YmgalId,
		game.SearchName,
		game.DlsiteId,
		game.ReleaseAt,
		game.RelatedGames,

		game.UseLocaleEmulator,
		game.UseMagpie,
		game.ProcessName,
		game.GetchuId,
		game.PvPath,
		game.VmId,
	)

	if err != nil {
		applog.LogErrorf(s.ctx, "AddGame: failed to insert game %s: %v", game.Name, err)
		return err
	}

	// 后台异步下载封面图片（不阻塞添加流程）
	if originalCoverURL != "" {
		//由ImageBackup代替
		// go s.asyncDownloadCoverImage(game.ID, game.Name, originalCoverURL)
	}

	return nil
}

// asyncDownloadCoverImage 后台异步下载封面图片并更新数据库
func (s *GameService) asyncDownloadCoverImage(gameID, gameName, coverURL string) {
	// 检查是否为远程URL
	if coverURL == "" || !strings.HasPrefix(coverURL, "http") || strings.Contains(coverURL, "wails.localhost") {
		return
	}
	fmt.Println("asyncDownloadCoverImage: downloading cover for %s", gameName)

	applog.LogInfof(s.ctx, "asyncDownloadCoverImage: downloading cover for %s", gameName)

	// 下载并保存图片
	localPath, err := utils.DownloadAndSaveCoverImage(coverURL, gameID)
	if err != nil {
		applog.LogWarningf(s.ctx, "asyncDownloadCoverImage: failed to download cover for %s: %v", gameName, err)
		return
	}

	// 更新数据库中的封面路径
	if err := s.updateCoverURL(gameID, localPath); err != nil {
		applog.LogErrorf(s.ctx, "asyncDownloadCoverImage: failed to update cover URL for %s: %v", gameName, err)
		return
	}

	applog.LogInfof(s.ctx, "asyncDownloadCoverImage: successfully cached cover for %s", gameName)
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
		applog.LogErrorf(s.ctx, "DeleteGame: failed to delete game_categories for id %s: %v", id, err)
		return fmt.Errorf("failed to delete game categories: %w", err)
	}

	// 删除关联的游玩会话记录
	_, err = s.db.ExecContext(s.ctx, "DELETE FROM play_sessions WHERE game_id = ?", id)
	if err != nil {
		applog.LogErrorf(s.ctx, "DeleteGame: failed to delete play_sessions for id %s: %v", id, err)
		return fmt.Errorf("failed to delete play sessions: %w", err)
	}
	//删除工作，人物，角色
	s.workService.DeleteWorksForGame(id)
	// 删除游戏记录
	result, err := s.db.ExecContext(s.ctx, "DELETE FROM games WHERE id = ?", id)
	if err != nil {
		applog.LogErrorf(s.ctx, "DeleteGame: failed to delete game for id %s: %v", id, err)
		return fmt.Errorf("failed to delete game: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		applog.LogErrorf(s.ctx, "DeleteGame: failed to get rows affected for id %s: %v", id, err)
		return err
	}

	if rowsAffected == 0 {
		applog.LogWarningf(s.ctx, "DeleteGame: game not found with id: %s", id)
		return fmt.Errorf("game not found with id: %s", id)
	}
	go s.ManageTagsForGames()

	return nil
}

func (s *GameService) DeleteGames(ids []string) error {
	ids = utils.UniqueNonEmptyStrings(ids)
	if len(ids) == 0 {
		return nil
	}

	placeholders := utils.BuildPlaceholders(len(ids))
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}

	tx, err := s.db.Begin()
	if err != nil {
		applog.LogErrorf(s.ctx, "DeleteGames: failed to begin transaction: %v", err)
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(s.ctx, fmt.Sprintf("DELETE FROM game_categories WHERE game_id IN (%s)", placeholders), args...); err != nil {
		applog.LogErrorf(s.ctx, "DeleteGames: failed to delete game_categories: %v", err)
		return fmt.Errorf("failed to delete game categories: %w", err)
	}

	if _, err := tx.ExecContext(s.ctx, fmt.Sprintf("DELETE FROM play_sessions WHERE game_id IN (%s)", placeholders), args...); err != nil {
		applog.LogErrorf(s.ctx, "DeleteGames: failed to delete play_sessions: %v", err)
		return fmt.Errorf("failed to delete play sessions: %w", err)
	}

	//删除工作，人物，角色
	s.workService.DeleteWorksForGames(ids)

	result, err := tx.ExecContext(s.ctx, fmt.Sprintf("DELETE FROM games WHERE id IN (%s)", placeholders), args...)
	if err != nil {
		applog.LogErrorf(s.ctx, "DeleteGames: failed to delete games: %v", err)
		return fmt.Errorf("failed to delete games: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		applog.LogErrorf(s.ctx, "DeleteGames: failed to get rows affected: %v", err)
		return err
	}
	if rowsAffected == 0 {
		applog.LogWarningf(s.ctx, "DeleteGames: no games deleted")
		return fmt.Errorf("no games deleted")
	}

	if err := tx.Commit(); err != nil {
		applog.LogErrorf(s.ctx, "DeleteGames: failed to commit transaction: %v", err)
		return err
	}

	go s.ManageTagsForGames()

	return nil
}

func (s *GameService) GetSimpleGamesByPage(page int, pageSize int) ([]models.Game, error) {
	query := fmt.Sprintf(`SELECT 
		id, name, 
	FROM games 
	ORDER BY created_at DESC
	LIMIT %d OFFSET %d
	`, pageSize, (page-1)*pageSize)
	var games []models.Game = []models.Game{}
	rows, err := s.db.QueryContext(s.ctx, query)
	if err != nil {
		applog.LogErrorf(s.ctx, "GetGames: failed to query games: %v", err)
		return games, fmt.Errorf("failed to query games: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var game models.Game

		err := rows.Scan(
			&game.ID,
			&game.Name,
		)
		if err != nil {
			applog.LogErrorf(s.ctx, "GetGames: failed to scan game row: %v", err)
			// return games, fmt.Errorf("failed to scan game: %w", err)
		}
		games = append(games, game)
	}

	if err = rows.Err(); err != nil {
		applog.LogErrorf(s.ctx, "GetGames: error iterating games: %v", err)
		return games, fmt.Errorf("error iterating games: %w", err)
	}

	return games, nil
}

func (s *GameService) GetGamesByPage(page int, pageSize int) ([]models.Game, error) {
	query := fmt.Sprintf(`SELECT 
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
		COALESCE(arguments, '') as arguments,
		COALESCE(images, '') as images,
		COALESCE(bangumi_id, '') as bangumi_id,
		COALESCE(dmm_id, '') as dmm_id,
		COALESCE(eroscape_id, '') as eroscape_id,
		COALESCE(ymgal_id, '') as ymgal_id,
		COALESCE(search_name, '') as search_name,
		COALESCE(dlsite_id, '') as dlsite_id,
		COALESCE(release_at, '') as release_at,
		COALESCE(related_games, '') as related_games,
		COALESCE(use_locale_emulator, FALSE) as use_locale_emulator,
		COALESCE(use_magpie, FALSE) as use_magpie,
		COALESCE(process_name, '') as process_name,
		COALESCE(getchu_id, '') as getchu_id,
		COALESCE(pv_path, '') as pv_path,
		COALESCE(vm_id, '') as vm_id
	FROM games 
	ORDER BY created_at DESC
	LIMIT %d OFFSET %d
	`, pageSize, (page-1)*pageSize)
	return s.GetGamesByQuery(query)
}

func (s *GameService) GetGamesSendFront(total int) error {
	page := 1
	pageSize := 20
	count := 0
	id := uuid.New().String()
	for {
		games, err := s.GetGamesByPage(page, pageSize)
		if err != nil {
			return err
		}

		// for _, game := range games {
		// 	count++
		// 	taskNotice := models.Task{
		// 		Id:          id,
		// 		Name:        "game_updates",
		// 		ItemData:    game,
		// 		Status:      enums.Completed,
		// 		ItemId:      game.ID,
		// 		Type:        enums.RefreshGames,
		// 		Description: "刷新游戏",
		// 		WorkingOn:   game.Name,
		// 		ItemStatus:  enums.Completed,
		// 		Completed:   count,
		// 		Total:       total,
		// 	}
		// 	runtime.EventsEmit(s.ctx, "game_updates", taskNotice)
		// }
		count += len(games)
		if len(games) > 0 {
			taskNotice := models.Task{
				Id:          id,
				Name:        "game_updates",
				ItemData:    games,
				Status:      enums.Completed,
				ItemId:      games[0].ID,
				Type:        enums.RefreshGames,
				Description: "刷新游戏",
				WorkingOn:   games[0].Name,
				ItemStatus:  enums.Completed,
				Completed:   count,
				Total:       total,
			}
			runtime.EventsEmit(s.ctx, "game_updates", taskNotice)
		}

		if len(games) < pageSize {
			break
		}
		page++
	}
	return nil
}

func (s *GameService) GetGamesByQuery(query string) ([]models.Game, error) {
	// fmt.Printf("GetGamesByQuery 01: query: %s\n", query)
	var games []models.Game = []models.Game{}
	rows, err := s.db.QueryContext(s.ctx, query)
	if err != nil {
		applog.LogErrorf(s.ctx, "GetGames: failed to query games: %v", err)
		return games, fmt.Errorf("failed to query games: %w", err)
	}
	defer rows.Close()

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
			&game.Arguments,
			&game.Images,
			&game.BangumiId,
			&game.DmmId,
			&game.EroscapeId,
			&game.YmgalId,
			&game.SearchName,
			&game.DlsiteId,
			&game.ReleaseAt,
			&game.RelatedGames,
			&game.UseLocaleEmulator,
			&game.UseMagpie,
			&game.ProcessName,
			&game.GetchuId,
			&game.PvPath,
			&game.VmId,
		)
		if err != nil {
			applog.LogErrorf(s.ctx, "GetGames: failed to scan game row: %v", err)
			// return games, fmt.Errorf("failed to scan game: %w", err)
		}

		game.SourceType = enums.SourceType(sourceType)
		game.Status = enums.GameStatus(status)
		games = append(games, game)
	}

	if err = rows.Err(); err != nil {
		applog.LogErrorf(s.ctx, "GetGames: error iterating games: %v", err)
		return games, fmt.Errorf("error iterating games: %w", err)
	}

	return games, nil
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
		COALESCE(arguments, '') as arguments,
		COALESCE(images, '') as images,
		COALESCE(bangumi_id, '') as bangumi_id,
		COALESCE(dmm_id, '') as dmm_id,
		COALESCE(eroscape_id, '') as eroscape_id,
		COALESCE(ymgal_id, '') as ymgal_id,
		COALESCE(search_name, '') as search_name,
		COALESCE(dlsite_id, '') as dlsite_id,
		COALESCE(release_at, '') as release_at,
		COALESCE(related_games, '') as related_games,
		COALESCE(use_locale_emulator, FALSE) as use_locale_emulator,
		COALESCE(use_magpie, FALSE) as use_magpie,
		COALESCE(process_name, '') as process_name,
		COALESCE(getchu_id, '') as getchu_id,
		COALESCE(pv_path, '') as pv_path,
		COALESCE(vm_id, '') as vm_id
	FROM games 
	ORDER BY created_at DESC`

	return s.GetGamesByQuery(query)
}

func (s *GameService) GetGamesByRelatedGames(gameIdsStr string) ([]models.Game, error) {
	games := []models.Game{}
	if gameIdsStr == "" {
		return games, nil
	}
	gameStrs := strings.Split(gameIdsStr, ",")
	if len(gameStrs) == 0 {
		return games, nil
	}
	for _, gameStr := range gameStrs {
		strs := strings.Split(gameStr, ":")
		id := strs[1]
		source := strs[0]
		// var game models.Game = models.Game{}
		rs := []models.Game{}
		query := ""
		if source == "local" {
			query = fmt.Sprintf("%s WHERE id = '%s'", s.GetQueryBase(), id)
		} else if source == string(enums.Dmm) {
			query = fmt.Sprintf("%s WHERE dmm_id = '%s'", s.GetQueryBase(), id)
		} else if source == string(enums.Dlsite) {
			query = fmt.Sprintf("%s WHERE dlsite_id = '%s'", s.GetQueryBase(), id)
		} else if source == string(enums.Ymgal) {
			query = fmt.Sprintf("%s WHERE ymgal_id = '%s'", s.GetQueryBase(), id)
		} else if source == string(enums.Eroscape) {
			query = fmt.Sprintf("%s WHERE eroscape_id = '%s'", s.GetQueryBase(), id)
		} else if source == string(enums.Bangumi) {
			query = fmt.Sprintf("%s WHERE bangumi_id = '%s'", s.GetQueryBase(), id)
		} else if source == string(enums.Getchu) {
			query = fmt.Sprintf("%s WHERE getchu_id = '%s'", s.GetQueryBase(), id)
		}
		rs, _ = s.GetGamesByQuery(query)
		games = append(games, rs...)
	}
	return games, nil
}

func (s *GameService) AddRelatedGames(gameIds []string, game models.Game) (models.Game, error) {
	// if game.RelatedGames == "" {
	// 	return s.GetGamesByRelatedGames(game.RelatedGames)
	// }
	gameIdsStr := game.RelatedGames
	oldIds := strings.Split(gameIdsStr, ",")
	idMap := map[string]string{}
	for _, s := range oldIds {
		// gs := strings.Split(s, ":")
		// if len(gs) != 2 {
		// 	continue
		// }
		if s == "" {
			continue
		}
		idMap[s] = s
	}
	for _, g := range gameIds {
		if g == "" {
			continue
		}
		s := string(enums.Local) + ":" + g
		idMap[s] = s
	}
	newIdsStr := ""
	for k, _ := range idMap {
		newIdsStr += fmt.Sprintf("%s,", k)
	}
	newIdsStr = strings.TrimSuffix(newIdsStr, ",")
	game.RelatedGames = newIdsStr

	fmt.Println("AddRelatedGames:", game.RelatedGames)
	s.UpdateGame(game)
	return game, nil
}

func (s *GameService) DeleteRelatedGame(gameToDelete, game models.Game) (models.Game, error) {
	fmt.Printf("before delete: %s\n", game.RelatedGames)
	toRemove := fmt.Sprintf("%s:%s", string(enums.Local), gameToDelete.ID)
	game.RelatedGames = utils.RemoveString(game.RelatedGames, toRemove)
	fmt.Printf("after delete: %s\n", game.RelatedGames)
	err := s.UpdateGame(game)
	return game, err
}

func (s *GameService) GetGamesByBrand(brand string) ([]models.Game, error) {
	games := []models.Game{}
	if brand == "" {
		return games, nil
	}
	query := fmt.Sprintf("%s WHERE company = '%s'", s.GetQueryBase(), brand)
	games, err := s.GetGamesByQuery(query)
	return games, err
}

func (s *GameService) GetQueryBase() string {
	return `SELECT 
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
		COALESCE(arguments, '') as arguments,
		COALESCE(images, '') as images,
		COALESCE(bangumi_id, '') as bangumi_id,
		COALESCE(dmm_id, '') as dmm_id,
		COALESCE(eroscape_id, '') as eroscape_id,
		COALESCE(ymgal_id, '') as ymgal_id,
		COALESCE(search_name, '') as search_name,
		COALESCE(dlsite_id, '') as dlsite_id,
		COALESCE(release_at, '') as release_at,
		COALESCE(related_games, '') as related_games,
		COALESCE(use_locale_emulator, FALSE) as use_locale_emulator,
		COALESCE(use_magpie, FALSE) as use_magpie,
		COALESCE(process_name, '') as process_name,
		COALESCE(getchu_id, '') as getchu_id,
		COALESCE(pv_path, '') as pv_path,
		COALESCE(vm_id, '') as vm_id,
	FROM games`
}

func (s *GameService) GetGamesByIdsStr(idsStr string) ([]models.Game, error) {
	var games []models.Game = []models.Game{}
	if idsStr == "" {
		return games, nil
	}
	query := s.GetQueryBase() + `
	WHERE LIST_CONTAINS(STRING_SPLIT(?, ','), id)
	ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(s.ctx, query, idsStr)
	if err != nil {
		runtime.LogErrorf(s.ctx, "GetGames: failed to query games: %v", err)
		log.Println("GetGames: failed to query games:", err)
		return nil, fmt.Errorf("failed to query games: %w", err)
	}
	defer rows.Close()

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
			&game.Arguments,
			&game.Images,
			&game.BangumiId,
			&game.DmmId,
			&game.EroscapeId,
			&game.YmgalId,
			&game.SearchName,
			&game.DlsiteId,
			&game.ReleaseAt,
			&game.RelatedGames,
			&game.UseLocaleEmulator,
			&game.UseMagpie,
			&game.ProcessName,
			&game.GetchuId,
			&game.PvPath,
			&game.VmId,
		)
		if err != nil {
			applog.LogErrorf(s.ctx, "GetGames: failed to scan game row: %v", err)
			return nil, fmt.Errorf("failed to scan game: %w", err)
		}

		game.SourceType = enums.SourceType(sourceType)
		game.Status = enums.GameStatus(status)
		games = append(games, game)
	}

	if err = rows.Err(); err != nil {
		applog.LogErrorf(s.ctx, "GetGames: error iterating games: %v", err)
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
		COALESCE(process_name, '') as process_name,
		COALESCE(status, 'not_started') as status,
		COALESCE(source_type, '') as source_type, 
		cached_at, 
		COALESCE(source_id, '') as source_id, 
		created_at,
		updated_at,
		COALESCE(tags, '') as tags, 
		COALESCE(arguments, '') as arguments, 
		COALESCE(images, '') as images,
		COALESCE(bangumi_id, '') as bangumi_id,
		COALESCE(dmm_id, '') as dmm_id,
		COALESCE(eroscape_id, '') as eroscape_id,
		COALESCE(ymgal_id, '') as ymgal_id,
		COALESCE(search_name, '') as search_name,
		COALESCE(dlsite_id, '') as dlsite_id,
		COALESCE(release_at, '') as release_at,
		COALESCE(related_games, '') as related_games,
		COALESCE(use_locale_emulator, FALSE) as use_locale_emulator,
		COALESCE(use_magpie, FALSE) as use_magpie,
		COALESCE(getchu_id, '') as getchu_id,
		COALESCE(pv_path, '') as pv_path,
		COALESCE(vm_id, '') as vm_id
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
		&game.ProcessName,
		&status,
		&sourceType,
		&game.CachedAt,
		&game.SourceID,
		&game.CreatedAt,
		&game.UpdatedAt,
		&game.Tags,
		&game.Arguments,
		&game.Images,
		&game.BangumiId,
		&game.DmmId,
		&game.EroscapeId,
		&game.YmgalId,
		&game.SearchName,
		&game.DlsiteId,
		&game.ReleaseAt,
		&game.RelatedGames,
		&game.UseLocaleEmulator,
		&game.UseMagpie,
		&game.GetchuId,
		&game.PvPath,
		&game.VmId,
	)

	if errors.Is(err, sql.ErrNoRows) {
		applog.LogWarningf(s.ctx, "GetGameByID: game not found with id: %s", id)
		return models.Game{}, fmt.Errorf("game not found with id: %s", id)
	}
	if err != nil {
		applog.LogErrorf(s.ctx, "GetGameByID: failed to query game %s: %v", id, err)
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
		process_name = ?,
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
		search_name = ?,
		dlsite_id = ?,
		release_at = ?,
		related_games = ?,
		use_locale_emulator = ?,
		use_magpie = ?,
		getchu_id = ?,
		pv_path = ?,
		vm_id = ?
	WHERE id = ?`

	result, err := s.db.ExecContext(s.ctx, query,
		game.Name,
		game.CoverURL,
		game.Company,
		game.Summary,
		game.Path,
		game.SavePath,
		game.ProcessName,
		string(game.Status),
		string(game.SourceType),
		game.CachedAt,
		game.SourceID,
		game.Tags,
		"", //game.Images,
		game.BangumiId,
		game.DmmId,
		game.EroscapeId,
		game.YmgalId,
		game.SearchName,
		game.DlsiteId,
		game.ReleaseAt,
		game.RelatedGames,
		game.UseLocaleEmulator,
		game.UseMagpie,
		game.GetchuId,
		game.PvPath,
		game.VmId,
		game.ID,
	)

	if err != nil {
		applog.LogErrorf(s.ctx, "UpdateGame: failed to update game %s: %v", game.ID, err)
		return fmt.Errorf("failed to update game: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		applog.LogErrorf(s.ctx, "UpdateGame: failed to get rows affected for id %s: %v", game.ID, err)
		return err
	}

	if rowsAffected == 0 {
		applog.LogWarningf(s.ctx, "UpdateGame: game not found with id: %s", game.ID)
		return fmt.Errorf("game not found with id: %s", game.ID)
	}
	fmt.Printf("成功更新游戏：%s\n", game.Name)
	jstr, err := json.MarshalIndent(game, "", "  ")
	if err == nil {
		fmt.Printf("game: \n$s\n", string(jstr))
	}

	return nil
}

// SelectSaveFile 选择存档文件
func (s *GameService) SelectSaveFile() (string, error) {
	selection, err := runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "选择存档文件",
	})
	return selection, err
}

// SelectSaveDirectory 选择存档目录
func (s *GameService) SelectSaveDirectory() (string, error) {
	selection, err := runtime.OpenDirectoryDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "选择存档文件夹",
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
		applog.LogErrorf(s.ctx, "failed to open file dialog: %v", err)
		return "", err
	}
	if selection == "" {
		return "", nil
	}

	coverPath, err := utils.SaveCoverImage(selection, gameID)
	imageBackup := models.ImageBackup{Url: coverPath, LocalPath: coverPath, SubjectId: gameID, SubjectType: 0, ImageType: 2}
	s.imageService.CreateOrUpdateImageBackup(imageBackup)
	if err != nil {
		applog.LogErrorf(s.ctx, "failed to save cover image: %v", err)
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
		applog.LogErrorf(s.ctx, "failed to open file dialog: %v", err)
		return "", err
	}
	if selection == "" {
		return "", nil
	}

	// 使用时间戳作为临时ID
	tempID := fmt.Sprintf("temp_%d", time.Now().UnixNano())
	coverPath, err := utils.SaveCoverImage(selection, tempID)
	if err != nil {
		applog.LogErrorf(s.ctx, "failed to save cover image: %v", err)
		return "", fmt.Errorf("failed to save cover image: %w", err)
	}

	return coverPath, nil
}

func (s *GameService) FetchMetadataByName(name string) ([]vo.GameMetadataFromWebVO, error) {
	var games []vo.GameMetadataFromWebVO
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 这里暂不处理任何错误，直接尝试从多个来源并发获取数据，空就是网络问题或未找到，不管它
	wg.Add(7)

	go func() {
		defer wg.Done()
		bgmGetter := utils.NewBangumiInfoGetter(s.config.SearchCn)
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
		ymgalGetter := utils.NewYmgalInfoGetter(s.config.SearchCn)
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
		dmm, _ := dmmGetter.FetchMetadataByName2(name)
		if dmm != (models.Game{}) {
			mu.Lock()
			games = append(games, vo.GameMetadataFromWebVO{Source: enums.Dmm, Game: dmm})
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		dlsiteGetter := utils.NewDlsiteInfoGetter()
		dlsite, _ := dlsiteGetter.FetchMetadataByName2(name)
		if dlsite != (models.Game{}) {
			mu.Lock()
			games = append(games, vo.GameMetadataFromWebVO{Source: enums.Dlsite, Game: dlsite})
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		getchuGetter := utils.NewGetchuInfoGetter()
		getchu, _ := getchuGetter.FetchMetadataByName2(name)
		if getchu != (models.Game{}) {
			mu.Lock()
			games = append(games, vo.GameMetadataFromWebVO{Source: enums.Dlsite, Game: getchu})
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		eroscapeGetter := utils.NewEroscapeInfoGetter(s.config.EroscapeUseMirror)
		eroscape, _ := eroscapeGetter.FetchMetadataByName2(name)
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

	var gameEntity models.GameEntity = models.GameEntity{}
	fmt.Printf("Request: source=%s id=%s staffs=%t chars=%t overwrite=%t images=%t\n",
		req.Source, req.ID, req.ShouldFetchStaffs, req.ShouldFetchCharactors, req.IsOverwrite, req.ShouldFetchImages)

	switch req.Source {
	case enums.Bangumi:
		fmt.Println("Fetching metadata from Bangumi Id:" + req.DbGameId)
		bgmGetter := utils.NewBangumiInfoGetter(s.config.SearchCn)
		gameEntity, e = bgmGetter.FetchMetadataReq(req, s.config.BangumiAccessToken)
		gameEntity, e = bgmGetter.FetchWorks(req, gameEntity, s.config.BangumiAccessToken)

	case enums.VNDB:
		fmt.Println("Fetching metadata from VNDB")
		vndbGetter := utils.NewVNDBInfoGetter()
		game, e = vndbGetter.FetchMetadata(req.ID, s.config.VNDBAccessToken)
	case enums.Ymgal:
		fmt.Println("Fetching metadata from Ymgal")
		ymgalGetter := utils.NewYmgalInfoGetter(s.config.SearchCn)
		gameEntity, e = ymgalGetter.FetchEntity(req, "")
		game = gameEntity.Game
	case enums.Eroscape:
		fmt.Println("Fetching metadata from Eroscape")
		escGetter := utils.NewEroscapeInfoGetter(s.config.EroscapeUseMirror)
		game.EroscapeId = req.ID
		gameEntity, e = escGetter.FetchMetadataById(req)
		gameEntity, e = escGetter.FetchCharactors(req, gameEntity)
		gameEntity, e = escGetter.FetchImages(req, gameEntity)
		// if req.ShouldUnionFetch && gameEntity.Game.DmmId != "" {
		// 	newReq := vo.MetadataRequest{DbGameId: game.ID, ID: gameEntity.Game.DmmId, Source: enums.Dmm,
		// 		IsOverwrite: false}
		// 	dmmGetter := utils.NewDmmInfoGetter()
		// 	dmmGameEntity, _ := dmmGetter.FetchMetadataById(newReq)
		// 	game = gameEntity.Game
		// 	game.Summary = dmmGameEntity.Game.Summary
		// 	gameEntity.Game = game
		// }
	case enums.Dmm:
		fmt.Println("Fetching metadata from DMM")
		dmmGetter := utils.NewDmmInfoGetter()
		game.DmmId = req.ID
		gameEntity, e = dmmGetter.FetchMetadataById(req)

	case enums.Dlsite:
		fmt.Println("Fetching metadata from dlsite")
		dlsiteGetter := utils.NewDlsiteInfoGetter()
		game.DlsiteId = req.ID
		gameEntity, e = dlsiteGetter.FetchMetadataById2(req)

	case enums.Getchu:
		fmt.Println("Fetching metadata from getchu")
		getchuGetter := utils.NewGetchuInfoGetter()
		game.GetchuId = req.ID
		gameEntity, e = getchuGetter.FetchMetadataById(req)
	}
	s.UnionFetch(&gameEntity, req)
	game = gameEntity.Game
	if req.IsOverwrite && req.ShouldFetchTags {
		s.tagService.CreateOrUpdateTagMapArray(gameEntity.Tags)
	}
	if req.IsOverwrite && (req.ShouldFetchStaffs || req.ShouldFetchCharactors) {
		s.workService.CreateOrUpdateListWorkStaffCharactor(utils.MapToArray(gameEntity.WorksMap))
	}
	s.imageService.SaveGameImages(gameEntity)
	return game, e
}

func (s *GameService) UnionFetch(gameEntity *models.GameEntity, req vo.MetadataRequest) {
	if !req.ShouldUnionFetch {
		return
	}
	fetched := false
	if req.Source != enums.Dmm && gameEntity.Game.DmmId != "" {
		newReq := vo.MetadataRequest{DbGameId: gameEntity.Game.ID, ID: gameEntity.Game.DmmId, Source: enums.Dmm,
			IsOverwrite: false}
		dmmGetter := utils.NewDmmInfoGetter()
		dmmGameEntity, _ := dmmGetter.FetchMetadataById(newReq)
		game := gameEntity.Game
		game.Summary = dmmGameEntity.Game.Summary
		gameEntity.Game = game
		fetched = true
	}
	if req.Source != enums.Getchu && gameEntity.Game.GetchuId != "" && !fetched {
		newReq := vo.MetadataRequest{DbGameId: gameEntity.Game.ID, ID: gameEntity.Game.GetchuId, Source: enums.Getchu,
			IsOverwrite: false}
		getchuGetter := utils.NewGetchuInfoGetter()
		getchuGameEntity, _ := getchuGetter.FetchMetadataById(newReq)
		game := gameEntity.Game
		game.Summary = getchuGameEntity.Game.Summary
		gameEntity.Game = game
		fetched = true
	}
	if req.Source != enums.Dlsite && gameEntity.Game.DlsiteId != "" && !fetched {
		newReq := vo.MetadataRequest{DbGameId: gameEntity.Game.ID, ID: gameEntity.Game.DlsiteId, Source: enums.Dlsite,
			IsOverwrite: false}
		dlsiteGetter := utils.NewDlsiteInfoGetter()
		dlsiteGameEntity, _ := dlsiteGetter.FetchMetadataById2(newReq)
		game := gameEntity.Game
		game.Summary = dlsiteGameEntity.Game.Summary
		gameEntity.Game = game
		fetched = true
	}

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
		Source:   existingGame.SourceType,
		ID:       existingGame.SourceID,
		DbGameId: existingGame.ID,
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

	applog.LogInfof(s.ctx, "UpdateGameFromRemote: successfully updated game %s from %s", existingGame.Name, existingGame.SourceType)
	return nil
}

// GetRunningProcesses 获取系统中正在运行的进程列表（过滤掉系统进程）
func (s *GameService) GetRunningProcesses() ([]utils.ProcessInfo, error) {
	return utils.GetRunningProcesses()
}

// OpenLocalPath 打开指定的本地文件或目录（通过资源管理器）
func (s *GameService) OpenLocalPath(path string) error {
	err := utils.OpenFileOrFolder(path)
	if err != nil {
		applog.LogErrorf(s.ctx, "OpenLocalPath failed for path %s: %v", path, err)
		return fmt.Errorf("打开路径失败: %w", err)
	}
	return nil
}

// UpdateGameProcessName 更新游戏的进程名
// 当用户选择了实际的游戏进程时调用
func (s *GameService) UpdateGameProcessName(gameID string, processName string) error {
	result, err := s.db.ExecContext(
		s.ctx,
		`UPDATE games SET process_name = ? WHERE id = ?`,
		processName,
		gameID,
	)
	if err != nil {
		applog.LogErrorf(s.ctx, "UpdateGameProcessName: failed to update process_name for game %s: %v", gameID, err)
		return fmt.Errorf("failed to update process_name: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("game not found with id: %s", gameID)
	}

	applog.LogInfof(s.ctx, "UpdateGameProcessName: updated process_name for game %s to %s", gameID, processName)
	return nil
}

// GetVideoStream 获取游戏视频流
func (s *GameService) GetVideoStream(gameID string) (string, error) {
	applog.LogInfof(s.ctx, "GetVideoStream: called with gameID: %s", gameID)

	// 获取游戏信息
	game, err := s.GetGameByID(gameID)
	if err != nil {
		applog.LogErrorf(s.ctx, "GetVideoStream: failed to get game %s: %v", gameID, err)
		return "", fmt.Errorf("failed to get game: %w", err)
	}

	applog.LogInfof(s.ctx, "GetVideoStream: found game: %s, PvPath: %s", game.Name, game.PvPath)

	if game.PvPath == "" {
		applog.LogWarningf(s.ctx, "GetVideoStream: game %s has no video path", game.Name)
		return "", fmt.Errorf("game has no video path")
	}

	// 检查视频文件是否存在
	if _, err := os.Stat(game.PvPath); os.IsNotExist(err) {
		applog.LogErrorf(s.ctx, "GetVideoStream: video file not found: %s", game.PvPath)
		return "", fmt.Errorf("video file not found: %s", game.PvPath)
	}

	applog.LogInfof(s.ctx, "GetVideoStream: video file exists: %s", game.PvPath)

	// 返回视频文件路径
	return game.PvPath, nil
}

// BatchUpdateStatus 批量更新多个游戏的游玩状态
func (s *GameService) BatchUpdateStatus(ids []string, status string) error {
	ids = utils.UniqueNonEmptyStrings(ids)
	if len(ids) == 0 {
		return nil
	}

	placeholders := utils.BuildPlaceholders(len(ids))
	// args: status + all ids
	args := make([]interface{}, 0, 1+len(ids))
	args = append(args, status)
	for _, id := range ids {
		args = append(args, id)
	}

	tx, err := s.db.Begin()
	if err != nil {
		applog.LogErrorf(s.ctx, "BatchUpdateStatus: failed to begin transaction: %v", err)
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(
		s.ctx,
		fmt.Sprintf("UPDATE games SET status = ? WHERE id IN (%s)", placeholders),
		args...,
	)
	if err != nil {
		applog.LogErrorf(s.ctx, "BatchUpdateStatus: failed to update games status: %v", err)
		return fmt.Errorf("failed to batch update status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	applog.LogInfof(s.ctx, "BatchUpdateStatus: updated %d games to status %s", rowsAffected, status)

	return tx.Commit()
}

func (s *GameService) UpdateGamesBackground(games []models.Game, req vo.MetadataRequest, id string) error {
	var uuid = uuid.New().String()
	s.taskService.RegisterTaskFunction(uuid, s.createGameUpdateTaskFunction())
	taskData := map[string]interface{}{
		"games": games,
		"req":   req,
		"delay": 1000,
	}
	return s.taskService.StartTask("game_updates", uuid, 1000, enums.Games, len(games), taskData)
	// return nil
}

func (s *GameService) ExecueteGamesUpdate(games []models.Game, req vo.MetadataRequest) {
	// var uuid = uuid.New().String()
	taskData := map[string]interface{}{
		"games": games,
		"req":   req,
	}
	var jsonData string = ""
	jsonBytes, err := json.Marshal(taskData)
	if err != nil {
		fmt.Sprintf("序列化任务数据失败: %v", err)
	}
	jsonData = string(jsonBytes)
	s.createGameUpdateTaskFunction()(s.ctx, jsonData, func(completed int, total int, workingOn string,
		warning string, itemId string, itemEvent enums.TaskStatus, itemData interface{}) {
	})
}

// 创建游戏更新任务函数
func (s *GameService) createGameUpdateTaskFunction() TaskFunction {
	return func(ctx context.Context, data string, updateProgress func(completed int, total int,
		workingOn string, warning string, itemId string, itemEvent enums.TaskStatus, itemData interface{})) error {
		// 定义结构来解组任务数据
		var taskData struct {
			Games []models.Game      `json:"games"`
			Req   vo.MetadataRequest `json:"req"`
			Delay int64              `json:"delay"`
		}

		if err := json.Unmarshal([]byte(data), &taskData); err != nil {
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

			if taskData.Req.Source == ngame.SourceType {
				id = ngame.SourceID
				// log.Printf("TaskFunc 01 id found 11 for game %s, id: %s", ngame.Name, id)
			} else if taskData.Req.Source == enums.Eroscape && strings.TrimSpace(ngame.EroscapeId) != "" {
				id = ngame.EroscapeId
				// log.Printf("TaskFunc 02 id found 12 for game %s, id: %s", ngame.Name, id)
			} else if taskData.Req.Source == enums.Ymgal && strings.TrimSpace(ngame.YmgalId) != "" {
				id = ngame.YmgalId
				// log.Printf("TaskFunc 03 id found 13 for game %s, id: %s", ngame.Name, id)
			} else if taskData.Req.Source == enums.Dmm && strings.TrimSpace(ngame.DmmId) != "" {
				id = ngame.DmmId
				// log.Printf("TaskFunc 04 id found 14 for game %s, id: %s", ngame.Name, id)
			} else if taskData.Req.Source == enums.Bangumi && strings.TrimSpace(ngame.BangumiId) != "" {
				id = ngame.BangumiId
				// log.Printf("TaskFunc 04 id found 14 for game %s, id: %s", ngame.Name, id)
			} else if taskData.Req.Source == enums.Dlsite && strings.TrimSpace(ngame.DlsiteId) != "" {
				id = ngame.DlsiteId
				// log.Printf("TaskFunc 04 id found 14 for game %s, id: %s", ngame.Name, id)
			} else if taskData.Req.Source == enums.Getchu && strings.TrimSpace(ngame.DlsiteId) != "" {
				id = ngame.GetchuId
				// log.Printf("TaskFunc 04 id found 14 for game %s, id: %s", ngame.Name, id)
			}

			taskData.Req.ID = id
			taskData.Req.DbGameId = ngame.ID

			var updatedGame models.Game
			var err error

			// log.Printf("TaskFunc 11 fetch metadata 01 for game %s, id: %s, source: %v", ngame.Name, id, taskData.Source)
			if strings.TrimSpace(id) != "" && !taskData.Req.ShouldMatchAgain {
				log.Printf("TaskFunc 12 fetch metadata 02 for game %s， id: %s", ngame.Name, id)

				// 通过ID获取元数据
				// req := vo.MetadataRequest{
				// 	Source: taskData.Req.Source,
				// 	ID:     id,
				// }
				updatedGame, err = s.FetchMetadata(taskData.Req)
				fmt.Println("发售日5：", updatedGame.ReleaseAt)

			} else {
				log.Printf("TaskFunc 13 fetch metadata 03 for game %s", ngame.Name)
				// 通过名称获取元数据
				if taskData.Req.Source == enums.Bangumi {
					// log.Printf("TaskFunc 21 fetch for game %s", ngame.Name)
					bgmGetter := utils.NewBangumiInfoGetter(s.config.SearchCn)
					updatedGame, err = bgmGetter.FetchMetadataByName(ngame.SearchName, s.config.BangumiAccessToken)
				} else if taskData.Req.Source == enums.VNDB {
					// log.Printf("TaskFunc 22 fetch for game %s", ngame.Name)
					vndbGetter := utils.NewVNDBInfoGetter()
					updatedGame, err = vndbGetter.FetchMetadataByName(ngame.SearchName, s.config.VNDBAccessToken)
				} else if taskData.Req.Source == enums.Ymgal {
					// log.Printf("TaskFunc 23 fetch for game %s", ngame.Name)
					ymgalGetter := utils.NewYmgalInfoGetter(s.config.SearchCn)
					updatedGame, err = ymgalGetter.FetchMetadataByName(ngame.SearchName, "")
				} else if taskData.Req.Source == enums.Eroscape {
					// log.Printf("TaskFunc 24 fetch for game %s", ngame.Name)
					escGetter := utils.NewEroscapeInfoGetter(s.config.EroscapeUseMirror)
					updatedGame, err = escGetter.FetchMetadataByName2(ngame.SearchName)
					// updatedGame = esc
				} else if taskData.Req.Source == enums.Dmm {
					// log.Printf("TaskFunc 25 fetch for game %s", ngame.Name)
					dmmGetter := utils.NewDmmInfoGetter()
					updatedGame, err = dmmGetter.FetchMetadataByName2(ngame.SearchName)
					// updatedGame = dmm
				} else if taskData.Req.Source == enums.Dlsite {
					// log.Printf("TaskFunc 25 fetch for game %s", ngame.Name)
					dlsiteGetter := utils.NewDlsiteInfoGetter()
					updatedGame, err = dlsiteGetter.FetchMetadataByName2(ngame.SearchName)
					// updatedGame = dmm
				} else if taskData.Req.Source == enums.Getchu {
					getchuGetter := utils.NewGetchuInfoGetter()
					updatedGame, err = getchuGetter.FetchMetadataByName2(ngame.SearchName)

					// log.Printf("TaskFunc 25 fetch for game %s", ngame.Name)

					// updatedGame = dmm
				} else {
					// log.Printf("TaskFunc 26 fetch for game %s", ngame.Name)
					return errors.New("未知的来源 ")
				}
				if updatedGame.SourceID != "" {
					id = updatedGame.SourceID
					req := taskData.Req
					req.ID = id
					fmt.Printf("id 23:%s, sourceId=%s, sourceType1=%s, source2=%s, bangumiId:%s\n", id, updatedGame.SourceID, string(updatedGame.SourceType), string(taskData.Req.Source), updatedGame.BangumiId)
					updatedGame, err = s.FetchMetadata(req)
				}
			}
			if err != nil {

				log.Printf("Failed to fetch metadata for game %s by ID: %s %v", ngame.Name, id, err)
				updateProgress(index, len(taskData.Games), "", fmt.Sprintf("Failed to fetch metadata for game %s by ID: %v", ngame.Name, err),
					ngame.ID, enums.Error, nil)
				continue
			}
			log.Printf("TaskFunc 31 fetch for game %s, id:%s", ngame.Name, updatedGame.SourceID)
			if updatedGame.SourceID == "" {
				updateProgress(index, len(taskData.Games), "", fmt.Sprintf("Failed to fetch metadata for game %s by ID: %v", ngame.Name, err),
					ngame.ID, enums.Error, nil)
				continue
			}
			if updatedGame.Name == "" {
				continue
			}

			s.FillGame(&ngame, &updatedGame, taskData.Req)
			fmt.Println("发售日4：", updatedGame.ReleaseAt)

			// 更新游戏
			if err := s.UpdateGame(updatedGame); err != nil {
				log.Printf("Failed to update game %s: %v", updatedGame.Name, err)
				continue
			}
			updateProgress(index, len(taskData.Games), "", fmt.Sprintf("complete for game %s by ID: %v", ngame.Name, err),
				ngame.ID, enums.Completed, updatedGame)

			time.Sleep(time.Millisecond * 1000)
		}

		// 标记完成
		updateProgress(len(taskData.Games), len(taskData.Games), "所有游戏更新完成", "", "", enums.Completed, nil)
		return nil
	}
}

func (s *GameService) FillGame(ngame *models.Game, updatedGame *models.Game, req vo.MetadataRequest) {
	fmt.Printf("fill game %s, tags:%v, staffs:%v, chara:%v, overwrite:%v, images:%v\n", ngame.Name, req.ShouldFetchTags, req.ShouldFetchStaffs, req.ShouldFetchCharactors,
		req.IsOverwrite, req.ShouldFetchImages)
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
	if updatedGame.DlsiteId == "" {
		updatedGame.DlsiteId = ngame.DlsiteId
	}
	if updatedGame.GetchuId == "" {
		updatedGame.GetchuId = ngame.GetchuId
	}
	if updatedGame.PvPath == "" {
		updatedGame.PvPath = ngame.PvPath
	}
	if ngame.Name != "" && !req.IsOverwrite {
		updatedGame.Name = ngame.Name
	}

	if ngame.Summary != "" && !req.IsOverwrite || updatedGame.Summary == "" {
		updatedGame.Summary = ngame.Summary
	}

	updatedGame.CreatedAt = ngame.CreatedAt
	if !req.IsOverwrite {
		updatedGame.SourceType = ngame.SourceType
		updatedGame.SourceID = ngame.SourceID
		// updatedGame.Name = ngame.Name
		// updatedGame.Summary = ngame.Summary
	}

	if !req.IsOverwrite || !req.ShouldFetchTags {
		updatedGame.Tags = ngame.Tags
	}

	if !req.IsOverwrite || !req.ShouldFetchImages {
		updatedGame.Images = ngame.Images
		updatedGame.CoverURL = ngame.CoverURL
	}

	updatedGame.CachedAt = time.Now()

	// updatedGame.Tags = utils.MergeStrings(updatedGame.Tags, ngame.Tags)

	// updatedGame.Charactors = utils.MergeStrings(updatedGame.Charactors, ngame.Charactors)
	// updatedGame.Staffs = utils.MergeStrings(updatedGame.Staffs, ngame.Staffs)
	// updatedGame.Images = utils.MergeStrings(updatedGame.Images, ngame.Images)

	updatedGame.SavePath = ngame.SavePath
	// updatedGame.ReleaseAt = ngame.ReleaseAt
	updatedGame.Status = ngame.Status

	updatedGame.UseMagpie = ngame.UseMagpie
	updatedGame.Arguments = ngame.Arguments
	updatedGame.SearchName = ngame.SearchName
	updatedGame.UseLocaleEmulator = ngame.UseLocaleEmulator
	updatedGame.VmId = ngame.VmId
	if updatedGame.DmmId == "" {
		updatedGame.DmmId = ngame.DmmId
	}
}

func (s *GameService) GetWorkGamesByStaffId(staffId string) ([]models.WorkGame, error) {
	works, err := s.workService.GetWorksByStaffId(staffId)
	var worksGames []models.WorkGame = []models.WorkGame{}
	if err != nil {
		return worksGames, err
	}

	for _, work := range works {
		game := models.Game{}
		game, err = s.GetGameByID(work.GameId)
		worksGames = append(worksGames, models.WorkGame{Work: work, Game: game})
	}
	return worksGames, err
}

func (s *GameService) GetWorkGamesByCharactorId(staffId string) ([]models.WorkGame, error) {
	works, err := s.workService.GetWorkGamesByCharactorId(staffId)
	var worksGames []models.WorkGame = []models.WorkGame{}
	if err != nil {
		return worksGames, err
	}

	for _, work := range works {
		game := models.Game{}
		game, err = s.GetGameByID(work.GameId)
		worksGames = append(worksGames, models.WorkGame{Work: work, Game: game})
	}
	return worksGames, err
}

func (s *GameService) LoadReviewsForGame(id string, sourceType enums.SourceType, page int) (models.GameReview, error) {
	if sourceType == enums.Eroscape {
		esReviewer := utils.NewEroscapeInfoGetter(s.config.EroscapeUseMirror)
		return esReviewer.FetchReviews(id, "", page)
	} else if sourceType == enums.Dlsite {
		esReviewer := utils.NewDlsiteInfoGetter()
		return esReviewer.FetchReviews(id, "", page)
	} else if sourceType == enums.Dmm {
		dmmReviewer := utils.NewDmmInfoGetter()
		return dmmReviewer.FetchReviews(id, "", page)
	}
	return models.GameReview{}, nil
}

func (s *GameService) LoadDetailReview(review models.Review, gameId string, sourceType enums.SourceType) (models.Review, error) {
	if sourceType == enums.Eroscape {
		esReviewer := utils.NewEroscapeInfoGetter(s.config.EroscapeUseMirror)
		return esReviewer.FetchReviewDetail(review, gameId, "")
	}
	return models.Review{}, nil
}

func (s *GameService) AddTagForGame(game models.Game, tags []string) (models.Game, error) {
	var err error = nil
	for _, tag := range tags {
		t, _ := s.tagService.GetTagByName(tag)
		if t == nil {
			newTag := models.Tag{Name: tag, Category: models.TagCategoryCustom}
			err = s.tagService.CreateTag(&newTag)
		}

		game.Tags = utils.MergeStrings(game.Tags, tag)
	}
	err = s.UpdateGame(game)
	return game, err
}

func (s *GameService) AddTagsForGames(games []models.Game, tags []string) ([]models.Game, error) {
	newGames := []models.Game{}
	for _, game := range games {
		newGame, _ := s.AddTagForGame(game, tags)
		// if err != nil {
		// 	continue
		// }
		newGames = append(newGames, newGame)
	}
	fmt.Printf("AddTagsForGames gamesTags:%s, tags:%v\n", newGames[0].Tags, tags)
	return newGames, nil
}

func (s *GameService) DeleteTagForGame(game models.Game, tag string) (models.Game, error) {
	game.Tags = utils.RemoveString(game.Tags, tag)
	err := s.UpdateGame(game)
	if err != nil {
		return game, err
	}
	go s.ManageTagsForGames()
	return game, nil
}

func (s *GameService) ManageTagsForGames() error {
	games, err := s.GetGames()
	if err != nil {
		applog.LogErrorf(s.ctx, "err:%v\n", err)
		return err
	}
	tags := make(map[string]string)
	for _, game := range games {
		for _, tag := range strings.Split(game.Tags, ",") {
			if tag != "" {
				tags[tag] = tag
			}
		}
	}
	tagAr := []string{}
	for _, tag := range tags {
		tagAr = append(tagAr, tag)
	}
	err = s.tagService.ManageTags(tagAr)
	return err
}

func (s *GameService) GetGamesByTag(tag string) ([]models.Game, error) {
	games := []models.Game{}
	if tag == "" {
		return games, nil
	}
	query := fmt.Sprintf("%s WHERE array_contains(string_split(tags, ','), '%s')", s.GetQueryBase(), tag)
	games, err := s.GetGamesByQuery(query)
	return games, err
}

func (s *GameService) SearchSave(game models.Game) (models.Game, error) {
	return utils.SearchSave(game)
}

func (s *GameService) SearchGameSaves(game []models.Game) error {
	var err error = nil
	for _, g := range game {
		if g.Company == "" {
			continue
		}
		_, err := utils.SearchSave(g)
		if err != nil {
			return err
		}
	}
	return err
}
