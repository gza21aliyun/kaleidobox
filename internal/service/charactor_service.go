package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"lunabox/internal/appconf"
	"lunabox/internal/models"
	"lunabox/internal/utils"
	"strings"

	"lunabox/internal/enums"

	"github.com/dlclark/regexp2"
	"github.com/google/uuid"
)

type CharactorService struct {
	ctx    context.Context
	db     *sql.DB
	config *appconf.AppConfig
}

func NewCharactorService() *CharactorService {
	return &CharactorService{}
}

func (s *CharactorService) Init(ctx context.Context, db *sql.DB, config *appconf.AppConfig) {
	s.ctx = ctx
	s.db = db
	s.config = config
}

// CreateCharactor 创建新的 Charactor 记录
func (s *CharactorService) CreateCharactor(charactor models.Charactor) error {
	query := `
		INSERT INTO charactors (id, name, other_names, image_path, images, 
		source_charactor_id, source_type, game_ids, summary, gender, measurements, height, sort)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(s.ctx, query,
		charactor.Id,
		charactor.Name,
		charactor.OtherNames,
		charactor.ImagePath,
		charactor.Images,
		charactor.SourceCharactorId,
		string(charactor.SourceType),
		charactor.GameIds,
		charactor.Summary,
		charactor.Gender,
		charactor.Measurements,
		charactor.Height,
		charactor.Sort,
	)
	fmt.Printf("创建角色：%s, err:%v\n", charactor.Name, err)
	return err
}

func (s *CharactorService) GetCharactorBySource(sourceType enums.SourceType, sourceCharactorId string) (models.Charactor, error) {
	if sourceCharactorId == "" {
		return models.Charactor{}, errors.New("Invalid source charactor id")
	}
	query := `
		SELECT id, name, other_names, image_path, images, source_charactor_id, 
		source_type, game_ids, summary, gender, measurements, height, sort
		FROM charactors
		WHERE source_charactor_id = ? AND source_type = `
	query += fmt.Sprintf(`'%s'`, string(sourceType))
	return s.GetCharactorByQueryId(sourceCharactorId, "", query)
}

func isUniqueCharactorName(name string) bool {
	if strings.Contains(name, "＝") {
		return true
	}
	pattern := regexp2.MustCompile(`\p{Han}`, 0)
	match, _ := pattern.MatchString(name)
	if match && len([]rune(name)) >= 4 {
		return true
	}
	return false
}

func (s *CharactorService) CreateOrUpdateCharactor(charactorName string, gameId string, sourceId string,
	sourceType enums.SourceType, sourceCharactorId string, charaImage string,
	summary string, mearsurements string, height string, sort int, staffName string) (models.Charactor, error) {
	charactor, err := s.GetCharactorBySource(sourceType, sourceCharactorId)
	if charactor.Id == "" /*&& isUniqueCharactorName(charactorName)*/ {
		charactor, err = s.GetCharactorByBothName(charactorName, staffName)
	}
	if err != nil || charactor.Id == "" {
		if err == sql.ErrNoRows || charactor.Id == "" {
			charactor, err = s.GetCharactorByGameIdAndName(gameId, charactorName)
			if err != nil && err == sql.ErrNoRows || charactor.Id == "" {
				id := uuid.New().String()
				otherNames := charactorName
				if strings.Contains(charactorName, "＝") {
					otherNames = utils.MergeStrings(otherNames, strings.Split(charactorName, "＝")[0])
				}
				charactor = models.Charactor{
					Name:              charactorName,
					OtherNames:        otherNames,
					GameIds:           gameId,
					Id:                id,
					SourceType:        sourceType,
					SourceCharactorId: sourceCharactorId,
					Images:            charaImage,
					ImagePath:         charaImage,
					Summary:           summary,
					Height:            height,
					Measurements:      mearsurements,
					Sort:              sort,
				}
				err = s.CreateCharactor(charactor)
				fmt.Println("CreateOrUpdateCharactor 05 " + charactorName)
				return charactor, err
			}
		} else {
			// return charactor, err
		}
	}
	if charactor.Id != "" {
		charactor.GameIds = utils.MergeStrings(charactor.GameIds, gameId)
		if charaImage != "" {
			charactor.ImagePath = charaImage
		}
		charactor.Images = utils.MergeStrings(charactor.Images, charaImage)
		otherNames := utils.MergeStrings(charactor.OtherNames, charactorName)
		if strings.Contains(charactorName, "＝") {
			otherNames = utils.MergeStrings(otherNames, strings.Split(charactorName, "＝")[0])
			fmt.Println("CreateOrUpdateCharactor 04 " + otherNames)
		}
		charactor.OtherNames = otherNames
		if charactor.Summary == "" {
			charactor.Summary = summary
		}
		err = s.UpdateCharactor(charactor)
		return charactor, err
	}
	return charactor, nil
}

// GetCharactorById 根据 ID 查询 Charactor 记录
func (s *CharactorService) GetCharactorById(id string) (models.Charactor, error) {
	query := `
		SELECT id, name, other_names, image_path, images, source_charactor_id, 
		source_type, game_ids, summary, gender, measurements, height, sort
		FROM charactors
		WHERE id = ?
	`
	return s.GetCharactorByQueryId(id, "", query)
}

func (s *CharactorService) GetCharactorByName(name string) (models.Charactor, error) {
	query := `
		SELECT id, name, other_names, image_path, images, source_charactor_id, 
		source_type, game_ids, summary, gender, measurements, height, sort
		FROM charactors
		WHERE name = ? OR array_contains(string_split(other_names, ','), ?)
	`
	return s.GetCharactorByQueryId(name, name, query)
}

func (s *CharactorService) GetCharactorByBothName(charactorName string, cvName string) (models.Charactor, error) {
	cName := charactorName
	if strings.Contains(charactorName, "＝") {
		cName = strings.Split(charactorName, "＝")[0]
	}
	query := `
		SELECT c.id, c.name, c.other_names, c.image_path, c.images, c.source_charactor_id, 
		c.source_type, c.game_ids, c.summary, c.gender, c.measurements, c.height, c.sort
		FROM charactors c
		JOIN works w ON c.id = w.charactor_id
		WHERE array_contains(string_split(c.other_names, ','), ?) AND w.staff_name = ?
	`
	return s.GetCharactorByQueryId(cName, cvName, query)
}

func (s *CharactorService) GetCharactorByGameIdAndName(gameId, name string) (models.Charactor, error) {
	// query := `
	// 	SELECT id, name, other_names, image_path, images, source_charactor_id,
	// 	source_type, game_ids, summary, gender, measurements, height, sort
	// 	FROM charactors
	// 	WHERE list_contains(string_split(game_ids, ','), ?) AND name = ?
	// `
	query := `
		SELECT id, name, other_names, image_path, images, source_charactor_id, 
		source_type, game_ids, summary, gender, measurements, height, sort
		FROM charactors
		WHERE list_contains(string_split(game_ids, ','), ?) AND list_contains(string_split(other_names, ','), ?)
	`
	return s.GetCharactorByQueryId(gameId, name, query)
}

func (s *CharactorService) GetCharactorByQueryId(id1 string, id2, query string) (models.Charactor, error) {
	var row *sql.Row
	if id1 == "" && id2 == "" {
		row = s.db.QueryRowContext(s.ctx, query)
	} else if id2 == "" {
		row = s.db.QueryRowContext(s.ctx, query, id1)
	} else {
		row = s.db.QueryRowContext(s.ctx, query, id1, id2)
	}

	var charactor models.Charactor
	var sourceType string
	err := row.Scan(
		&charactor.Id,
		&charactor.Name,
		&charactor.OtherNames,
		&charactor.ImagePath,
		&charactor.Images,
		&charactor.SourceCharactorId,
		&sourceType,
		&charactor.GameIds,
		&charactor.Summary,
		&charactor.Gender,
		&charactor.Measurements,
		&charactor.Height,
		&charactor.Sort,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return charactor, nil // 未找到记录
		}
		return charactor, err
	}
	charactor.SourceType = enums.SourceType(sourceType)
	return charactor, nil
}

// UpdateCharactor 更新 Charactor 记录
func (s *CharactorService) UpdateCharactor(charactor models.Charactor) error {
	query := `
		UPDATE charactors
		SET name = ?, other_names = ?, image_path = ?, images = ?, source_charactor_id = ?, 
		source_type = ?, game_ids = ?, summary = ?, gender = ?, measurements = ?, height = ?, sort = ?
		WHERE id = ?
	`
	_, err := s.db.ExecContext(s.ctx, query,
		charactor.Name,
		charactor.OtherNames,
		charactor.ImagePath,
		charactor.Images,
		charactor.SourceCharactorId,
		string(charactor.SourceType),
		charactor.GameIds,
		charactor.Summary,
		charactor.Gender,
		charactor.Measurements,
		charactor.Height,
		charactor.Sort,
		charactor.Id,
	)
	return err
}

// DeleteCharactor 删除 Charactor 记录
func (s *CharactorService) DeleteCharactor(id string) error {
	query := `DELETE FROM charactors WHERE id = ?`
	_, err := s.db.ExecContext(s.ctx, query, id)
	return err
}

// ListCharactors 查询所有 Charactor 记录
func (s *CharactorService) ListCharactors() ([]*models.Charactor, error) {
	query := `
		SELECT id, name, other_names, image_path, images, source_charactor_id, source_type, 
		game_ids, summary, gender, measurements, height, sort
		FROM charactors
	`
	rows, err := s.db.QueryContext(s.ctx, query)
	if err != nil {
		fmt.Println("Error querying charactors:", err)
		return nil, err
	}
	defer rows.Close()

	var charactors []*models.Charactor
	for rows.Next() {
		var charactor models.Charactor
		var sourceType string
		err := rows.Scan(
			&charactor.Id,
			&charactor.Name,
			&charactor.OtherNames,
			&charactor.ImagePath,
			&charactor.Images,
			&charactor.SourceCharactorId,
			&sourceType,
			&charactor.GameIds,
			&charactor.Summary,
			&charactor.Gender,
			&charactor.Measurements,
			&charactor.Height,
			&charactor.Sort,
		)
		if err != nil {
			fmt.Println("ListCharactors 01 ", err)
			return nil, err
		}
		charactor.SourceType = enums.SourceType(sourceType)
		charactors = append(charactors, &charactor)
	}
	return charactors, nil
}

func (s *CharactorService) CountCharactors() (int, error) {
	query := `SELECT COUNT(*) FROM charactors`
	row := s.db.QueryRowContext(s.ctx, query)

	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// SearchCharactors 根据关键字模糊搜索角色（按名称或别名匹配），限制返回数量避免内存爆炸
func (s *CharactorService) SearchCharactors(keyword string, limit int) ([]models.Charactor, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, name, other_names, image_path, images, source_charactor_id, source_type,
		game_ids, summary, gender, measurements, height, sort
		FROM charactors
	`
	var rows *sql.Rows
	var err error
	if keyword == "" {
		query += ` ORDER BY sort DESC LIMIT ?`
		rows, err = s.db.QueryContext(s.ctx, query, limit)
	} else {
		query += ` WHERE name LIKE ? OR other_names LIKE ? ORDER BY sort DESC LIMIT ?`
		like := "%" + keyword + "%"
		rows, err = s.db.QueryContext(s.ctx, query, like, like, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var charactors []models.Charactor
	for rows.Next() {
		var charactor models.Charactor
		var sourceType string
		err := rows.Scan(
			&charactor.Id,
			&charactor.Name,
			&charactor.OtherNames,
			&charactor.ImagePath,
			&charactor.Images,
			&charactor.SourceCharactorId,
			&sourceType,
			&charactor.GameIds,
			&charactor.Summary,
			&charactor.Gender,
			&charactor.Measurements,
			&charactor.Height,
			&charactor.Sort,
		)
		if err != nil {
			return nil, err
		}
		charactor.SourceType = enums.SourceType(sourceType)
		charactors = append(charactors, charactor)
	}
	return charactors, nil
}
