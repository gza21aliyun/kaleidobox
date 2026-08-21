package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"lunabox/internal/appconf"
	"lunabox/internal/applog"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"lunabox/internal/utils"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type WorkService struct {
	ctx              context.Context
	db               *sql.DB
	config           *appconf.AppConfig
	staffService     *StaffService
	charactorService *CharactorService
	imageService     *ImageService
}

func (s *WorkService) SetServices(staffService *StaffService, charactorService *CharactorService, imageService *ImageService) {
	s.staffService = staffService
	s.charactorService = charactorService
	s.imageService = imageService
}

func NewWorkService() *WorkService {
	return &WorkService{}
}

func (s *WorkService) Init(ctx context.Context, db *sql.DB, config *appconf.AppConfig) {
	s.ctx = ctx
	s.db = db
	s.config = config
}

// CreateWork 创建新的 Work 记录
func (s *WorkService) CreateWork(work models.Work) error {
	query := `
		INSERT INTO works (id, game_id, staff_id, role, charactor_id, charactor_name, staff_name, work_summary, 
		source_type, source_staff_id, source_charactor_id, source_game_id, images, game_name, game_cover, sort)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(s.ctx, query,
		work.Id,
		work.GameId,
		work.StaffId,
		string(work.Role),
		work.CharactorId,
		work.CharactorName,
		work.StaffName,
		work.WorkSummary,
		string(work.SourceType),
		work.SourceStaffId,
		work.SourceCharactorId,
		work.SourceGameId,
		work.Images,
		work.GameName,
		work.CharactorImage,
		work.Sort,
	)
	if err != nil {
		applog.ErrorLogSaveAppLog("创建工作失败 gameId:%s, staffName: %s, charactorName: %s\n",
			err, work.GameId, work.StaffName, work.CharactorName)
	} else {
		// fmt.Printf("创建工作成功 gameId:%s, staffName: %s, charactorName: %s\n",
		// 	work.GameId, work.StaffName, work.CharactorName)
	}
	return err
}

func (s *WorkService) CreateOrUpdateWorkStaffCharactor(work models.Work) error {
	staff := models.Staff{}
	charactor := models.Charactor{}
	var err error = nil
	work.StaffName = strings.ReplaceAll(work.StaffName, `"`, "")
	if work.StaffName != "" {
		fmt.Printf("07 CreateOrUpdateWorkStaffCharactor %s %s %s %s %s\n", work.StaffName, work.CharactorName, string(work.Role), work.GameId, work.Images)
		staff, err = s.staffService.CreateOrUpdateStaff(work.StaffName, work.GameId, work.SourceGameId, work.SourceType,
			work.SourceStaffId, work.Role, work.StaffImage)
		if err != nil {
			fmt.Println("05 CreateOrUpdateWorkStaffCharactor %s, %v ", work.StaffName, err)
			return err
		}
		work.StaffId = staff.Id
		if work.StaffImage != "" {
			err = s.imageService.CreateOrUpdateImageBackup(models.ImageBackup{Url: work.StaffImage, SubjectId: staff.Id,
				SubjectType: 2, CreatedAt: time.Now(), GameId: work.GameId})
		}

	}
	// fmt.Println("11 CreateOrUpdateWorkStaffCharactor")
	if work.CharactorName != "" {
		fmt.Printf("三围022：%s\n", work.Measurements)
		charactor, err = s.charactorService.CreateOrUpdateCharactor(work.CharactorName, work.GameId, work.SourceGameId,
			work.SourceType, work.SourceCharactorId, work.CharactorImage, work.WorkSummary, work.Measurements, work.Height, work.Sort, work.StaffName)
		if err != nil {
			fmt.Println("06 CreateOrUpdateWorkStaffCharactor %s, %v", charactor.Name, err)
			return err
		}
		work.CharactorId = charactor.Id
		if work.CharactorImage != "" {
			err = s.imageService.CreateOrUpdateImageBackup(models.ImageBackup{Url: work.CharactorImage, SubjectId: charactor.Id,
				SubjectType: 1, CreatedAt: time.Now(), GameId: work.GameId})
		}
	}
	// fmt.Println("12 CreateOrUpdateWorkStaffCharactor")
	newWork := models.Work{}
	if work.StaffName != "" && work.CharactorName == "" {
		newWork, err = s.GetWorkByStaff(work.GameId, work.StaffName)
		if err != nil && err != sql.ErrNoRows {
			fmt.Println("获取员工工作出错 CreateOrUpdateWorkStaffCharactor %s, %v", work.StaffName, err)
			return err
		}
	}
	if newWork.Id == "" && work.CharactorName != "" && work.StaffName != "" {
		newWork, err = s.GetWorkByCv(work.GameId, work.StaffName, work.CharactorName)
		if err != nil && err != sql.ErrNoRows {
			fmt.Println("获取员工工作出错2 CreateOrUpdateWorkStaffCharactor %s, %v", work.StaffName, err)
			return err
		}
	}
	if newWork.Id == "" && work.CharactorName != "" {
		newWork, err = s.GetWorkByCharactor(work.GameId, work.CharactorName)
		if err != nil && err != sql.ErrNoRows {
			fmt.Println("获取角色工作出错 CreateOrUpdateWorkStaffCharactor %s, %v", work.StaffName, err)
			return err
		}
	}
	// fmt.Println("14 CreateOrUpdateWorkStaffCharactor")
	if err != nil || newWork.Id == "" {
		// fmt.Println("15 CreateOrUpdateWorkStaffCharactor", err)
		if err == sql.ErrNoRows || newWork.Id == "" {
			work.Id = uuid.New().String()
			err = s.CreateWork(work)
			if err != nil {
				applog.LogErrorf(s.ctx, "创建工作出错 CreateOrUpdateWorkStaffCharactor %s, %v", work.StaffName, err)
			} else {
				err = s.imageService.CreateOrUpdateImageBackup(models.ImageBackup{
					Url: work.CharactorImage, SubjectId: work.Id, SubjectType: 3, ImageType: 0, CreatedAt: time.Now(), GameId: work.GameId})
				for _, image := range strings.Split(work.Images, ",") {
					err = s.imageService.CreateOrUpdateImageBackup(models.ImageBackup{
						Url: image, SubjectId: work.Id, SubjectType: 3, ImageType: 1, CreatedAt: time.Now(), GameId: work.GameId})
				}
			}
			return err
		} else {
			return err
		}
	} else {
		fmt.Printf("16 CreateOrUpdateWorkStaffCharactor:%v, id:%s\n", err, newWork.Id)
		work.Id = newWork.Id
		return s.UpdateWork(work)
	}

}

func (s *WorkService) CreateOrUpdateListWorkStaffCharactor(works []models.Work) {
	fmt.Println("03 CreateOrUpdateListWorkStaffCharactor " + strconv.Itoa(len(works)))
	for _, work := range works {
		if work.Role == enums.Charactor && work.StaffName != "" {
			work.Role = enums.CV
		}
		err := s.CreateOrUpdateWorkStaffCharactor(work)
		if err != nil {
			log.Printf("04 CreateOrUpdateListWorkStaffCharactor %v", err)
		}
	}
}

func (s *WorkService) DeleteWorksForGame(gameId string) {
	s.DeleteWorksForGames([]string{gameId})
}

func (s *WorkService) DeleteWorksForGames(gameIds []string) {
	if len(gameIds) == 0 {
		return
	}

	placeholders := utils.BuildPlaceholders(len(gameIds))
	query := fmt.Sprintf("SELECT id, game_id, staff_id, role, charactor_id, charactor_name, staff_name, work_summary, source_type, source_staff_id, source_charactor_id, source_game_id, images, game_name, game_cover, sort FROM works WHERE game_id IN (%s)", placeholders)

	args := make([]interface{}, len(gameIds))
	for i, v := range gameIds {
		args[i] = v
	}
	works, err := s.GetWorksByQueryId(query, args...)
	if err != nil {
		applog.ErrorLogSaveAppLogs("获取工作失败 gameIds:%v, err: %v\n", gameIds, err)
		return
	}

	staffGameMap := make(map[string]map[string]bool)
	charactorGameMap := make(map[string]map[string]bool)

	for _, work := range works {
		if work.StaffId != "" {
			if staffGameMap[work.StaffId] == nil {
				staffGameMap[work.StaffId] = make(map[string]bool)
			}
			staffGameMap[work.StaffId][work.GameId] = true
		}
		if work.CharactorId != "" {
			if charactorGameMap[work.CharactorId] == nil {
				charactorGameMap[work.CharactorId] = make(map[string]bool)
			}
			charactorGameMap[work.CharactorId][work.GameId] = true
		}
	}

	deleteQuery := fmt.Sprintf("DELETE FROM works WHERE game_id IN (%s)", placeholders)
	_, err = s.db.ExecContext(s.ctx, deleteQuery, args...)
	if err != nil {
		applog.ErrorLogSaveAppLogs("删除工作失败 gameIds:%v, err: %v\n", gameIds, err)
	}

	for _, gameId := range gameIds {
		for staffId, gameIdMap := range staffGameMap {
			if !gameIdMap[gameId] {
				continue
			}
			staff, err := s.staffService.GetStaffById(staffId)
			if err != nil || staff.Id == "" {
				continue
			}
			staff.GameIds = utils.RemoveString(staff.GameIds, gameId)
			if staff.GameIds == "" {
				s.staffService.DeleteStaff(staffId)
			} else {
				s.staffService.UpdateStaff(staff)
			}
		}

		for charactorId, gameIdMap := range charactorGameMap {
			if !gameIdMap[gameId] {
				continue
			}
			charactor, err := s.charactorService.GetCharactorById(charactorId)
			if err != nil || charactor.Id == "" {
				continue
			}
			charactor.GameIds = utils.RemoveString(charactor.GameIds, gameId)
			if charactor.GameIds == "" {
				s.charactorService.DeleteCharactor(charactorId)
			} else {
				s.charactorService.UpdateCharactor(charactor)
			}
		}
	}
}

func (s *WorkService) GetWorkByStaff(gameId, staffName string) (models.Work, error) {
	query := `
		SELECT id, game_id, staff_id, role, charactor_id, charactor_name, staff_name, work_summary, source_type, 
		source_staff_id, source_charactor_id, source_game_id, images, game_name, game_cover, sort
		FROM works
		WHERE game_id = ? AND staff_name = ?
	`
	return s.GetWorkByQueryIds(query, gameId, staffName)
}

func (s *WorkService) GetWorkByCv(gameId, staffName string, charactorName string) (models.Work, error) {
	cName := charactorName
	if strings.Contains(charactorName, "＝") {
		cName = strings.Split(charactorName, "＝")[0]
	}
	query := `
		SELECT w.id, w.game_id, w.staff_id, w.role, w.charactor_id, w.charactor_name, w.staff_name, w.work_summary, w.source_type, 
		w.source_staff_id, w.source_charactor_id, w.source_game_id, w.images, w.game_name, w.game_cover, w.sort
		FROM works w
		JOIN charactors c ON w.charactor_id = c.id
		WHERE array_contains(string_split(c.other_names, ','), ?) AND w.staff_name = ? AND w.game_id = ?
	`
	row := s.db.QueryRowContext(s.ctx, query, cName, staffName, gameId)

	var work models.Work = models.Work{}
	var sourceType string
	var role string
	err := row.Scan(
		&work.Id,
		&work.GameId,
		&work.StaffId,
		&role,
		&work.CharactorId,
		&work.CharactorName,
		&work.StaffName,
		&work.WorkSummary,
		&sourceType,
		&work.SourceStaffId,
		&work.SourceCharactorId,
		&work.SourceGameId,
		&work.Images,
		&work.GameName,
		&work.CharactorImage,
		&work.Sort,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return work, err // 未找到记录
		}
		return work, err
	}
	work.SourceType = enums.SourceType(sourceType)
	work.Role = enums.StaffRole(role)
	return work, nil
}

func (s *WorkService) GetWorkByStaffId(staffId string) (models.Work, error) {
	query := `
		SELECT id, game_id, staff_id, role, charactor_id, charactor_name, staff_name, work_summary, 
		source_type, source_staff_id, source_charactor_id, source_game_id, images, game_name, game_cover, sort
		FROM works
		WHERE staff_id = ?
	`
	return s.GetWorkByQueryIds(query, staffId, "")
}

func (s *WorkService) GetWorkByCharactor(gameId, charactorName string) (models.Work, error) {
	query := `
		SELECT id, game_id, staff_id, role, charactor_id, charactor_name, staff_name, work_summary, source_type,
		 source_staff_id, source_charactor_id, source_game_id, images, game_name, game_cover, sort
		FROM works
		WHERE game_id = ? AND charactor_name = ?
	`
	return s.GetWorkByQueryIds(query, gameId, charactorName)
}

func (s *WorkService) GetWorkByQueryIds(query string, id1 string, id2 string) (models.Work, error) {
	var row *sql.Row
	if id1 == "" && id2 == "" {
		row = s.db.QueryRowContext(s.ctx, query)
	} else if id2 == "" {
		row = s.db.QueryRowContext(s.ctx, query, id1)
	} else {
		row = s.db.QueryRowContext(s.ctx, query, id1, id2)
	}

	var work models.Work = models.Work{}
	var sourceType string
	var role string
	err := row.Scan(
		&work.Id,
		&work.GameId,
		&work.StaffId,
		&role,
		&work.CharactorId,
		&work.CharactorName,
		&work.StaffName,
		&work.WorkSummary,
		&sourceType,
		&work.SourceStaffId,
		&work.SourceCharactorId,
		&work.SourceGameId,
		&work.Images,
		&work.GameName,
		&work.CharactorImage,
		&work.Sort,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return work, err // 未找到记录
		}
		return work, err
	}
	work.SourceType = enums.SourceType(sourceType)
	work.Role = enums.StaffRole(role)
	return work, nil
}

func (s *WorkService) GetWorksByGameId(gameId string) ([]models.Work, error) {
	query := `
		SELECT id, game_id, staff_id, role, charactor_id, charactor_name, staff_name, work_summary, source_type, 
		source_staff_id, source_charactor_id, source_game_id, images, game_name, game_cover, sort 
		FROM works
		WHERE game_id = ?

	`
	works, err := s.GetWorksByQueryId(query, gameId)
	return works, err
}

func (s *WorkService) GetWorksByStaffId(staffId string) ([]models.Work, error) {
	query := `
		SELECT id, game_id, staff_id, role, charactor_id, charactor_name, staff_name, work_summary, 
		source_type, source_staff_id, source_charactor_id, source_game_id, images, game_name, game_cover, sort 
		FROM works
		WHERE staff_id = ?

	`
	works, err := s.GetWorksByQueryId(query, staffId)
	return works, err
}

func (s *WorkService) GetWorksByStaffIdAndRole(staffId string, role enums.StaffRole) ([]models.Work, error) {
	query := `
		SELECT id, game_id, staff_id, role, charactor_id, charactor_name, staff_name, work_summary, 
		source_type, source_staff_id, source_charactor_id, source_game_id, images, game_name, game_cover, sort 
		FROM works
		WHERE staff_id = ? AND role = ?

	`
	works, err := s.GetWorksByQueryId(query, staffId, string(role))
	return works, err
}

func (s *WorkService) GetWorkGamesByCharactorId(charactorId string) ([]models.Work, error) {
	query := `
		SELECT id, game_id, staff_id, role, charactor_id, charactor_name, staff_name, work_summary, 
		source_type, source_staff_id, source_charactor_id, source_game_id, images, game_name, game_cover, sort 
		FROM works
		WHERE charactor_id = ?

	`
	works, err := s.GetWorksByQueryId(query, charactorId)
	return works, err
}

func (s *WorkService) GetWorkGamesById(id string) (models.Work, error) {
	query := `
		SELECT id, game_id, staff_id, role, charactor_id, charactor_name, staff_name, work_summary, 
		source_type, source_staff_id, source_charactor_id, source_game_id, images, game_name, game_cover, sort 
		FROM works
		WHERE id = ?

	`
	works, err := s.GetWorkByQueryIds(query, id, "")
	return works, err
}

func (s *WorkService) GetWorksByQueryId(query string, args ...interface{}) ([]models.Work, error) {
	rows, err := s.db.QueryContext(s.ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var works []models.Work
	for rows.Next() {
		var work models.Work
		var sourceType string
		var role string
		err := rows.Scan(
			&work.Id,
			&work.GameId,
			&work.StaffId,
			&role,
			&work.CharactorId,
			&work.CharactorName,
			&work.StaffName,
			&work.WorkSummary,
			&sourceType,
			&work.SourceStaffId,
			&work.SourceCharactorId,
			&work.SourceGameId,
			&work.Images,
			&work.GameName,
			&work.CharactorImage,
			&work.Sort,
		)
		if err != nil {
			return nil, err
		}
		work.SourceType = enums.SourceType(sourceType)
		work.Role = enums.StaffRole(role)
		works = append(works, work)
		// fmt.Printf("GetWorksByGameId 01 , staffname: %s, role: %s \n", work.StaffName, string(work.Role))
	}
	return works, nil
}

func (s *WorkService) GetWorksMapByGameId(gameId string) (map[enums.StaffRole][]models.Work, error) {

	works, err := s.GetWorksByGameId(gameId)
	if err != nil {
		return nil, err
	}
	var workMap map[enums.StaffRole][]models.Work = make(map[enums.StaffRole][]models.Work)
	for _, work := range works {
		workMap[work.Role] = append(workMap[work.Role], work)

		fmt.Printf("GetWorksByGameId 02 , staffname: %s, role: %s \n", work.StaffName, string(work.Role))
	}
	return workMap, nil
}

// UpdateWork 更新 Work 记录
func (s *WorkService) UpdateWork(work models.Work) error {
	query := `
		UPDATE works
		SET role = ?, charactor_name = ?, staff_name = ?, work_summary = ?, source_type = ?, source_staff_id = ?, source_charactor_id = ?, source_game_id = ?, images = ?, game_name = ?, game_cover = ?,
		game_id = ?, staff_id = ?, charactor_id = ?, sort = ? 
		WHERE id = ?
	`
	_, err := s.db.ExecContext(s.ctx, query,
		string(work.Role),
		work.CharactorName,
		work.StaffName,
		work.WorkSummary,
		string(work.SourceType),
		work.SourceStaffId,
		work.SourceCharactorId,
		work.SourceGameId,
		work.Images,
		work.GameName,
		work.CharactorImage,
		work.GameId,
		work.StaffId,
		work.CharactorId,
		work.Sort,
		work.Id,
	)
	if err != nil {
		fmt.Printf("更新工作失败 gameId:%s, staffName: %s, charactorName: %s, err: %v\n",
			work.GameId, work.StaffName, work.CharactorName, err)
		return err
	}
	// newWork, _ := s.GetWorkGamesById(work.Id)
	// fmt.Printf("更新工作成功 gameId:%s, staffName: %s, charactorName: %s, images: %v, img:%s\n",
	// 	newWork.GameId, newWork.StaffName, newWork.CharactorName, newWork.Images, work.Images)
	return err
}

// DeleteWork 删除单个 Work 记录，并从关联角色/人员的 game_ids 中移除该游戏，
// 同时删除该 work 及其被级联删除的角色/人员对应的图片备份和本地文件
func (s *WorkService) DeleteWork(work models.Work) error {
	query := `DELETE FROM works WHERE id = ?`
	_, err := s.db.ExecContext(s.ctx, query, work.Id)
	if err != nil {
		applog.ErrorLogSaveAppLog("删除工作失败 id:%s, err: %v", work.Id, err)
		return err
	}

	// 删除该 work 自身的图片备份（作品主图、截图等）
	if work.Id != "" {
		if err := s.imageService.DeleteImageBackupsBySubject(work.Id, 3); err != nil {
			applog.ErrorLogSaveAppLog("删除工作图片失败 id:%s, err: %v", work.Id, err)
		}
	}

	// 从关联角色的 game_ids 中移除该游戏
	if work.CharactorId != "" {
		charactor, err := s.charactorService.GetCharactorById(work.CharactorId)
		if err == nil && charactor.Id != "" {
			charactor.GameIds = utils.RemoveString(charactor.GameIds, work.GameId)
			if charactor.GameIds == "" {
				// 角色已无关联游戏，删除角色前先删除其图片
				if err := s.imageService.DeleteImageBackupsBySubject(charactor.Id, 1); err != nil {
					applog.ErrorLogSaveAppLog("删除角色图片失败 id:%s, err: %v", charactor.Id, err)
				}
				err = s.charactorService.DeleteCharactor(charactor.Id)
			} else {
				err = s.charactorService.UpdateCharactor(charactor)
			}
			if err != nil {
				applog.ErrorLogSaveAppLog("删除工作后更新角色 game_ids 失败 id:%s, err: %v", charactor.Id, err)
			}
		}
	}

	// 从关联人员的 game_ids 中移除该游戏
	if work.StaffId != "" {
		staff, err := s.staffService.GetStaffById(work.StaffId)
		if err == nil && staff.Id != "" {
			staff.GameIds = utils.RemoveString(staff.GameIds, work.GameId)
			if staff.GameIds == "" {
				// 人员已无关联游戏，删除人员前先删除其图片
				if err := s.imageService.DeleteImageBackupsBySubject(staff.Id, 2); err != nil {
					applog.ErrorLogSaveAppLog("删除人员图片失败 id:%s, err: %v", staff.Id, err)
				}
				err = s.staffService.DeleteStaff(staff.Id)
			} else {
				err = s.staffService.UpdateStaff(staff)
			}
			if err != nil {
				applog.ErrorLogSaveAppLog("删除工作后更新人员 game_ids 失败 id:%s, err: %v", staff.Id, err)
			}
		}
	}

	return nil
}

// ListWorks 查询所有 Work 记录
func (s *WorkService) ListWorks() ([]*models.Work, error) {
	query := `
		SELECT id, game_id, staff_id, role, charactor_id, charactor_name, staff_name, work_summary, source_type, 
		source_staff_id, source_charactor_id, source_game_id, images, game_name, game_cover, sort
		FROM works
	`
	rows, err := s.db.QueryContext(s.ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var works []*models.Work
	for rows.Next() {
		var work models.Work
		var sourceType string
		var role string
		err := rows.Scan(
			&work.Id,
			&work.GameId,
			&work.StaffId,
			&role,
			&work.CharactorId,
			&work.CharactorName,
			&work.StaffName,
			&work.WorkSummary,
			&sourceType,
			&work.SourceStaffId,
			&work.SourceCharactorId,
			&work.SourceGameId,
			&work.Images,
			&work.GameName,
			&work.CharactorImage,
			&work.Sort,
		)
		if err != nil {
			return nil, err
		}
		work.SourceType = enums.SourceType(sourceType)
		work.Role = enums.StaffRole(role)
		works = append(works, &work)
	}
	return works, nil
}

// CountWorks 返回 Work 表中的记录总数
func (s *WorkService) CountWorks() (int, error) {
	query := `SELECT COUNT(*) FROM works`
	row := s.db.QueryRowContext(s.ctx, query)

	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
