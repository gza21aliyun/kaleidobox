package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"lunabox/internal/appconf"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"lunabox/internal/utils"

	"github.com/google/uuid"
)

type StaffService struct {
	ctx    context.Context
	db     *sql.DB
	config *appconf.AppConfig
}

func NewStaffService() *StaffService {
	return &StaffService{}
}

func (s *StaffService) Init(ctx context.Context, db *sql.DB, config *appconf.AppConfig) {
	s.ctx = ctx
	s.db = db
	s.config = config
}

// CreateStaff 创建新的 Staff 记录
func (s *StaffService) CreateStaff(staff models.Staff) error {
	query := `
		INSERT INTO staffs (id, name, other_names, roles, source_staff_id, source_type,
		game_ids, summary, gender, image)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(s.ctx, query,
		staff.Id,
		staff.Name,
		staff.OtherNames,
		staff.Roles,
		staff.SourceStaffId,
		string(staff.SourceType),
		staff.GameIds,
		staff.Summary,
		staff.Gender,
		staff.Image,
	)
	return err
}

func (s *StaffService) CreateOrUpdateStaff(staffName string, gameId string, sourceId string,
	sourceType enums.SourceType, sourceStaffId string, role enums.StaffRole, image string) (models.Staff, error) {
	staff, err := s.GetStaffBySource(sourceType, sourceStaffId)
	if err != nil || staff.Id == "" {
		// fmt.Println("24  staff " + staff.Name)
		if err == sql.ErrNoRows || staff.Id == "" {
			staff, err := s.GetStaffByGameIdAndName(gameId, staffName)
			if err != nil && err == sql.ErrNoRows || staff.Id == "" {
				id := uuid.New().String()
				staff = models.Staff{Name: staffName, GameIds: gameId, Id: id}
				staff.SourceType = sourceType
				staff.Roles = string(role)
				staff.SourceStaffId = sourceStaffId
				staff.Image = image
				staff.OtherNames = staffName
				// fmt.Println("21 create staff " + staff.Name)
				err = s.CreateStaff(staff)
				return staff, err
			}
		} else {
			return staff, err
		}
	}
	if staff.Id != "" {

		// fmt.Println("22 update staff " + staff.Name)
		staff.OtherNames = utils.MergeStrings(staff.OtherNames, staffName)
		staff.GameIds = utils.MergeStrings(staff.GameIds, gameId)
		staff.Roles = utils.MergeStrings(staff.Roles, string(role))
		err = s.UpdateStaff(staff)
		return staff, err
	}
	// fmt.Println("23  staff " + staff.Name)
	return staff, err
}

func (s *StaffService) GetStaffByGameIdAndName(id string, name string) (models.Staff, error) {
	query := `
		SELECT id, name, other_names, roles, source_staff_id, source_type, game_ids, summary, gender, image
		FROM staffs
		WHERE list_contains(string_split(game_ids, ','), ?) AND name = ?
	}
	`
	return s.GetStaffByQueryId(id, name, query)
}

func (s *StaffService) GetStaffsByGameId(id string) ([]models.Staff, error) {
	query := `
		SELECT id, name, other_names, roles, source_staff_id, source_type, game_ids, summary, gender, image
		FROM staffs
		WHERE list_contains(string_split(game_ids, ','), ?)
	`
	rows, err := s.db.QueryContext(s.ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var staffs []models.Staff
	for rows.Next() {
		var staff models.Staff
		var sourceType string
		err := rows.Scan(
			&staff.Id,
			&staff.Name,
			&staff.OtherNames,
			&staff.Roles,
			&staff.SourceStaffId,
			&sourceType,
			&staff.GameIds,
			&staff.Summary,
			&staff.Gender,
			&staff.Image,
		)
		if err != nil {
			return nil, err
		}
		staff.SourceType = enums.SourceType(sourceType)
		staffs = append(staffs, staff)
	}
	return staffs, nil
}

func (s *StaffService) GetStaffBySource(sourceType enums.SourceType, sourceStaffId string) (models.Staff, error) {
	if sourceStaffId == "" {
		return models.Staff{}, errors.New("sourceStaffId is empty")
	}
	query := `
		SELECT id, name, other_names, roles, source_staff_id, source_type,
		game_ids, summary, gender, image
		FROM staffs
		WHERE source_staff_id = ? AND source_type = `
	query += fmt.Sprintf(`'%s'`, string(sourceType))
	staff, err := s.GetStaffByQueryId(sourceStaffId, "", query)
	if err != nil {
		fmt.Printf("GetStaffBySource 04 query:%s error: %v", query, err)
	}
	return staff, err
}

// GetStaffById 根据 ID 查询 Staff 记录
func (s *StaffService) GetStaffById(id string) (models.Staff, error) {
	query := `
		SELECT id, name, other_names, roles, source_staff_id, source_type, 
		game_ids, summary, gender, image
		FROM staffs
		WHERE id = ?
	`
	return s.GetStaffByQueryId(id, "", query)
}

func (s *StaffService) GetFirstStaff() (models.Staff, error) {
	query := `
		SELECT id, name, other_names, roles, source_staff_id, source_type, 
		game_ids, summary, gender, image
		FROM staffs
	`
	return s.GetStaffByQueryId("", "", query)
}

func (s *StaffService) GetStaffByQueryId(id1 string, id2 string, query string) (models.Staff, error) {
	var row *sql.Row
	if id1 != "" && id2 != "" {
		row = s.db.QueryRowContext(s.ctx, query, id1, id2)
	} else if id1 != "" {
		row = s.db.QueryRowContext(s.ctx, query, id1)
	} else {
		row = s.db.QueryRowContext(s.ctx, query)
	}

	var staff models.Staff = models.Staff{}
	var sourceType string
	err := row.Scan(
		&staff.Id,
		&staff.Name,
		&staff.OtherNames,
		&staff.Roles,
		&staff.SourceStaffId,
		&sourceType,
		&staff.GameIds,
		&staff.Summary,
		&staff.Gender,
		&staff.Image,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Staff{}, nil // 未找到记录
		}
		return models.Staff{}, err
	}
	staff.SourceType = enums.SourceType(sourceType)
	return staff, nil
}

// UpdateStaff 更新 Staff 记录
func (s *StaffService) UpdateStaff(staff models.Staff) error {
	query := `
		UPDATE staffs
		SET name = ?, other_names = ?, roles = ?, source_staff_id = ?, source_type = ?, game_ids = ?, summary = ?, gender = ?, image = ?
		WHERE id = ?
	`
	_, err := s.db.ExecContext(s.ctx, query,
		staff.Name,
		staff.OtherNames,
		staff.Roles,
		staff.SourceStaffId,
		string(staff.SourceType),
		staff.GameIds,
		staff.Summary,
		staff.Gender,
		staff.Image,
		staff.Id,
	)
	return err
}

// DeleteStaff 删除 Staff 记录
func (s *StaffService) DeleteStaff(id string) error {
	query := `DELETE FROM staffs WHERE id = ?`
	_, err := s.db.ExecContext(s.ctx, query, id)
	return err
}

// ListStaffs 查询所有 Staff 记录
func (s *StaffService) ListStaffs() ([]models.Staff, error) {
	query := `
		SELECT id, name, other_names, roles, source_staff_id, source_type, game_ids, summary, gender, image
		FROM staffs
	`
	rows, err := s.db.QueryContext(s.ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var staffs []models.Staff
	for rows.Next() {
		var staff models.Staff
		var sourceType string
		err := rows.Scan(
			&staff.Id,
			&staff.Name,
			&staff.OtherNames,
			&staff.Roles,
			&staff.SourceStaffId,
			&sourceType,
			&staff.GameIds,
			&staff.Summary,
			&staff.Gender,
			&staff.Image,
		)
		if err != nil {
			return nil, err
		}
		staff.SourceType = enums.SourceType(sourceType)
		staffs = append(staffs, staff)
	}
	return staffs, nil
}

func (s *StaffService) CountStaffs() (int, error) {
	query := `SELECT COUNT(*) FROM staffs`
	row := s.db.QueryRowContext(s.ctx, query)

	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *StaffService) GetStaffsByRole(role enums.StaffRole) ([]*models.Staff, error) {
	query := `
		SELECT DISTINCT st.id, st.name, st.other_names, st.roles, st.source_staff_id, st.source_type, st.game_ids, st.summary, st.gender, st.image
		FROM works w
		JOIN staffs st ON w.staff_id = st.id
		WHERE w.role = ? AND w.staff_id IS NOT NULL AND w.staff_id != ''
	`
	rows, err := s.db.QueryContext(s.ctx, query, string(role))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var staffs []*models.Staff
	for rows.Next() {
		var staff models.Staff
		var sourceType string
		err := rows.Scan(
			&staff.Id,
			&staff.Name,
			&staff.OtherNames,
			&staff.Roles,
			&staff.SourceStaffId,
			&sourceType,
			&staff.GameIds,
			&staff.Summary,
			&staff.Gender,
			&staff.Image,
		)
		if err != nil {
			continue
		}
		staff.SourceType = enums.SourceType(sourceType)
		staffs = append(staffs, &staff)
	}

	return staffs, nil
}
