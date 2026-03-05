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
	"strconv"
	"strings"

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
		fmt.Printf("创建工作失败 gameId:%s, staffName: %s, charactorName: %s, err:%v\n",
			work.GameId, work.StaffName, work.CharactorName, err)
	} else {
		fmt.Printf("创建工作成功 gameId:%s, staffName: %s, charactorName: %s\n",
			work.GameId, work.StaffName, work.CharactorName)
	}
	return err
}

func (s *WorkService) CreateOrUpdateWorkStaffCharactor(work models.Work) error {
	staff := models.Staff{}
	charactor := models.Charactor{}
	var err error = nil
	if work.StaffName != "" {
		fmt.Printf("07 CreateOrUpdateWorkStaffCharactor %s %s %s %s %s\n", work.StaffName, work.CharactorName, string(work.Role), work.GameId, work.Images)
		staff, err = s.staffService.CreateOrUpdateStaff(work.StaffName, work.GameId, work.SourceGameId, work.SourceType,
			work.SourceStaffId, work.Role, work.StaffImage)
		if err != nil {
			fmt.Println("05 CreateOrUpdateWorkStaffCharactor %s, %v ", work.StaffName, err)
			return err
		}
		work.StaffId = staff.Id
		err = s.imageService.CreateOrUpdateImageBackup(models.ImageBackup{Url: work.StaffImage, SubjectId: staff.Id, SubjectType: 2})
	}
	// fmt.Println("11 CreateOrUpdateWorkStaffCharactor")
	if work.CharactorName != "" {
		fmt.Printf("三围022：%s\n", work.Measurements)
		charactor, err = s.charactorService.CreateOrUpdateCharactor(work.CharactorName, work.GameId, work.SourceGameId,
			work.SourceType, work.SourceCharactorId, work.CharactorImage, work.WorkSummary, work.Measurements, work.Height, work.Sort)
		if err != nil {
			fmt.Println("06 CreateOrUpdateWorkStaffCharactor %s, %v", charactor.Name, err)
			return err
		}
		work.CharactorId = charactor.Id
		err = s.imageService.CreateOrUpdateImageBackup(models.ImageBackup{Url: work.CharactorImage, SubjectId: charactor.Id, SubjectType: 1})
	}
	// fmt.Println("12 CreateOrUpdateWorkStaffCharactor")
	newWork := models.Work{}
	if work.StaffName != "" {
		newWork, err = s.GetWorkByStaff(work.GameId, work.StaffName)
		if err != nil && err != sql.ErrNoRows {
			fmt.Println("获取员工工作出错 CreateOrUpdateWorkStaffCharactor %s, %v", work.StaffName, err)
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
					Url: work.CharactorImage, SubjectId: work.Id, SubjectType: 3, ImageType: 0})
				for _, image := range strings.Split(work.Images, ",") {
					err = s.imageService.CreateOrUpdateImageBackup(models.ImageBackup{
						Url: image, SubjectId: work.Id, SubjectType: 3, ImageType: 1})
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

func (s *WorkService) GetWorkByStaff(gameId, staffName string) (models.Work, error) {
	query := `
		SELECT id, game_id, staff_id, role, charactor_id, charactor_name, staff_name, work_summary, source_type, 
		source_staff_id, source_charactor_id, source_game_id, images, game_name, game_cover, sort
		FROM works
		WHERE game_id = ? AND staff_name = ?
	`
	return s.GetWorkByQueryIds(query, gameId, staffName)
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

func (s *WorkService) GetWorksByQueryId(query, id1 string) ([]models.Work, error) {
	rows, err := s.db.QueryContext(s.ctx, query, id1)
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
