package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"lunabox/internal/applog"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"lunabox/internal/utils"
	"lunabox/internal/vo"
	"strings"
	"sync"
	"time"

	"encoding/json"

	"github.com/google/uuid"
)

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

// 创建游戏更新任务函数
func (s *GameService) createGameUpdateTaskFunction() TaskFunction {
	return func(ctx context.Context, data string, updateProgress func(completed int, total int,
		workingOn string, warning string, itemId string, itemEvent enums.TaskStatus, resultGames []models.ResultGames, itemData interface{})) error {
		// 定义结构来解组任务数据
		var taskData struct {
			Games []models.Game      `json:"games"`
			Req   vo.MetadataRequest `json:"req"`
			Delay int64              `json:"delay"`
		}

		if err := json.Unmarshal([]byte(data), &taskData); err != nil {
			return fmt.Errorf("解析任务数据失败: %v", err)
		}
		resultGames := []models.ResultGames{}
		result := models.ResultGames{}
		result.Title = "游戏元数据搜刮更新"
		result.Description = "已更新游戏"
		result.Status = 200
		failed := models.ResultGames{}
		failed.Description = "更新失败"
		failed.Status = 400
		result.GameIds = []string{}
		resultGames = append(resultGames, result)
		resultGames = append(resultGames, failed)

		updateProgress(0, len(taskData.Games), fmt.Sprintf("开始更新游戏: "),
			"", "", enums.Started, resultGames, nil)

		// 实现UpdateGamesBackground的核心逻辑
		for index, ngame := range taskData.Games {
			// 检查是否被取消（通过上下文检查）
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			updateProgress(index, len(taskData.Games), fmt.Sprintf("更新游戏: %s", ngame.Name),
				"", ngame.ID, enums.Initial, resultGames, nil)

			var id = ""
			log.Printf("TaskFunc 00 source: %v\n, game source type: %s, req source: %s\n", ngame, string(ngame.SourceType), string(taskData.Req.Source))

			if taskData.Req.Source == ngame.SourceType && strings.TrimSpace(ngame.SourceID) != "" {
				id = ngame.SourceID
				log.Printf("TaskFunc 01 id found 11 for game %s, id: %s", ngame.Name, id)
			} else if taskData.Req.Source == enums.Eroscape && strings.TrimSpace(ngame.EroscapeId) != "" {
				id = ngame.EroscapeId
				log.Printf("TaskFunc 02 id found 12 for game %s, id: %s", ngame.Name, id)
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
				log.Printf("TaskFunc 13 fetch metadata 03 for game %s, id: %s", ngame.Name, id)
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
				failed.GameIds = append(failed.GameIds, ngame.ID)
				resultGames[1] = failed
				updateProgress(index, len(taskData.Games), "", fmt.Sprintf("Failed to fetch metadata for game %s by ID: %v", ngame.Name, err),
					ngame.ID, enums.Error, resultGames, nil)
				continue
			}
			log.Printf("TaskFunc 31 fetch for game %s, id:%s", ngame.Name, updatedGame.SourceID)
			if updatedGame.SourceID == "" {
				failed.GameIds = append(failed.GameIds, ngame.ID)
				resultGames[1] = failed
				updateProgress(index, len(taskData.Games), "", fmt.Sprintf("Failed to fetch metadata for game %s by ID: %v", ngame.Name, err),
					ngame.ID, enums.Error, resultGames, nil)
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
			result.GameIds = append(result.GameIds, updatedGame.ID)
			resultGames[0] = result
			updateProgress(index, len(taskData.Games), "", fmt.Sprintf("complete for game %s by ID: %v", ngame.Name, err),
				ngame.ID, enums.Completed, resultGames, updatedGame)

			time.Sleep(time.Millisecond * 1000)
		}

		// 标记完成
		updateProgress(len(taskData.Games), len(taskData.Games), "所有游戏更新完成", "", "", enums.Completed, resultGames, nil)
		return nil
	}
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
		warning string, itemId string, itemEvent enums.TaskStatus, resultGames []models.ResultGames, itemData interface{}) {
	})
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

func (s *GameService) UpdateGameByReq(req vo.MetadataRequest, oldGame models.Game) (models.Game, error) {
	req.DbGameId = oldGame.ID
	game, err := s.FetchMetadata(req)
	if err != nil {
		return models.Game{}, fmt.Errorf("failed to fetch metadata from remote: %w", err)
	}
	s.FillGame(&oldGame, &game, req)
	err = s.UpdateGame(game)
	return game, err
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

func (s *GameService) FetchMetadataNotSave(req vo.MetadataRequest) (models.GameEntity, error) {
	var gameEntity models.GameEntity
	var e error

	// if game, e = fetchFromLocal(req.ID); e == nil {
	// 	return gameEntity, nil
	// }
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
		gameEntity, e = vndbGetter.FetchEntity(req, s.config.VNDBAccessToken)
		if e != nil {
			fmt.Printf("Fetching metadata from VNDB error: %v\n", e)
		}
	case enums.Ymgal:
		fmt.Println("Fetching metadata from Ymgal")
		ymgalGetter := utils.NewYmgalInfoGetter(s.config.SearchCn)
		gameEntity, e = ymgalGetter.FetchEntity(req, "")
	case enums.Eroscape:
		fmt.Println("Fetching metadata from Eroscape")
		escGetter := utils.NewEroscapeInfoGetter(s.config.EroscapeUseMirror)
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
		// game.DmmId = req.ID
		gameEntity, e = dmmGetter.FetchMetadataById(req)

	case enums.Dlsite:
		fmt.Println("Fetching metadata from dlsite")
		dlsiteGetter := utils.NewDlsiteInfoGetter()
		// game.DlsiteId = req.ID
		gameEntity, e = dlsiteGetter.FetchMetadataById2(req)

	case enums.Getchu:
		fmt.Println("Fetching metadata from getchu")
		getchuGetter := utils.NewGetchuInfoGetter()
		// game.GetchuId = req.ID
		gameEntity, e = getchuGetter.FetchMetadataById(req)
	}
	s.UnionFetch(&gameEntity, req)
	return gameEntity, e
}

func (s *GameService) FetchMetadata(req vo.MetadataRequest) (models.Game, error) {
	gameEntity, e := s.FetchMetadataNotSave(req)

	game := gameEntity.Game
	if e != nil {
		return game, e
	}
	if req.IsOverwrite && req.ShouldFetchTags {
		s.tagService.CreateOrUpdateTagMapArray(gameEntity.Tags)
	}
	if req.ShouldFetchStaffs || req.ShouldFetchCharactors {
		if req.IsOverwrite {
			s.workService.DeleteWorksForGame(game.ID)
		}
		s.workService.CreateOrUpdateListWorkStaffCharactor(utils.MapToArray(gameEntity.WorksMap))
	}
	if req.ShouldFetchImages {
		s.imageService.SaveGameImages(gameEntity, req.IsOverwrite)
	}
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
		if game.Images == "" {
			game.Images = dmmGameEntity.Game.Images
		}
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
		if game.Images == "" {
			game.Images = getchuGameEntity.Game.Images
		}
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
		if game.Images == "" {
			game.Images = dlsiteGameEntity.Game.Images
		}
		gameEntity.Game = game
		fetched = true
	}

}

func fetchFromLocal(id string) (models.Game, error) {
	// 这个函数暂时返回错误，表示未实现从本地数据库获取
	// 如果需要实现，应该在这里查询数据库
	return models.Game{}, fmt.Errorf("game not found in local cache")
}
