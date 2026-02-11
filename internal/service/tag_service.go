package service

import (
	"context"
	"database/sql"
	"fmt"
	"lunabox/internal/appconf"
	"lunabox/internal/models"
)

type TagService struct {
	ctx    context.Context
	db     *sql.DB
	config *appconf.AppConfig
}

func NewTagService() *TagService {
	return &TagService{}
}

func (s *TagService) Init(ctx context.Context, db *sql.DB, config *appconf.AppConfig) {
	s.ctx = ctx
	s.db = db
	s.config = config
}

// CreateTag 创建新的 Tag 记录
func (s *TagService) CreateTag(tag *models.Tag) error {
	if tag == nil {
		return nil
	}
	if tag.Name == "" {
		return nil
	}
	if tag.Category == "" {
		tag.Category = models.TagCategoryBrand
	}
	query := `
		INSERT INTO tags (name, category, group_name, is_h, is_spoiler, block_modify)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(s.ctx, query,
		tag.Name,
		tag.Category,
		tag.Group,
		tag.IsH,
		tag.IsSpoiler,
		tag.BlockModify,
	)
	fmt.Println("创建标签成功：", tag.Name, " ", tag.Category)
	return err
}

func (s *TagService) CreateOrUpdateTag(name string, category string) error {
	tag, err := s.GetTagByName(name)
	cate := category
	if cate == "" {
		cate = models.TagCategoryOther
	}
	if err != nil || tag == nil {
		if err == sql.ErrNoRows || tag == nil {
			tag = &models.Tag{
				Name:        name,
				Category:    cate,
				IsH:         false,
				IsSpoiler:   false,
				BlockModify: false,
			}
			return s.CreateTag(tag)
		}
		return err
	} else {
		if !tag.BlockModify {
			tag.Category = cate
		}
		return s.UpdateTag(tag)
	}
	return err
}

func (s *TagService) CreateOrUpdateTagMapArray(tagMapArray map[string][]models.Tag) error {

	for category, names := range tagMapArray {
		for _, name := range names {
			err := s.CreateOrUpdateTag(name.Name, category)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// GetTagByName 根据 Name 查询 Tag 记录
func (s *TagService) GetTagByName(name string) (*models.Tag, error) {
	query := `
		SELECT name, category, group_name, is_h, is_spoiler, block_modify
		FROM tags
		WHERE name = ?
	`
	row := s.db.QueryRowContext(s.ctx, query, name)

	var tag models.Tag
	err := row.Scan(
		&tag.Name,
		&tag.Category,
		&tag.Group,
		&tag.IsH,
		&tag.IsSpoiler,
		&tag.BlockModify,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 未找到记录
		}
		return nil, err
	}
	return &tag, nil
}

// UpdateTag 更新 Tag 记录
func (s *TagService) UpdateTag(tag *models.Tag) error {
	query := `
		UPDATE tags
		SET category = ?, group_name = ?, is_h = ?, is_spoiler = ?, block_modify = ?
		WHERE name = ?
	`
	_, err := s.db.ExecContext(s.ctx, query,
		tag.Category,
		tag.Group,
		tag.IsH,
		tag.IsSpoiler,
		tag.BlockModify,
		tag.Name,
	)
	return err
}

// DeleteTag 删除 Tag 记录
func (s *TagService) DeleteTag(name string) error {
	query := `DELETE FROM tags WHERE name = ?`
	_, err := s.db.ExecContext(s.ctx, query, name)
	return err
}

// ListTags 查询所有 Tag 记录
func (s *TagService) ListTags() ([]*models.Tag, error) {
	query := `
		SELECT name, category, group_name, is_h, is_spoiler, block_modify
		FROM tags
	`
	rows, err := s.db.QueryContext(s.ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*models.Tag
	for rows.Next() {
		var tag models.Tag
		err := rows.Scan(
			&tag.Name,
			&tag.Category,
			&tag.Group,
			&tag.IsH,
			&tag.IsSpoiler,
			&tag.BlockModify,
		)
		if err != nil {
			return nil, err
		}
		tags = append(tags, &tag)
	}
	return tags, nil
}

func (s *TagService) GetTagListByString(tagString string) ([]models.Tag, error) {
	query := `
		SELECT name, category, group_name, is_h, is_spoiler, block_modify
		FROM tags
		WHERE name IN (
			SELECT unnest(string_to_array(?, ','))
		)
	`
	rows, err := s.db.QueryContext(s.ctx, query, tagString)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		err := rows.Scan(
			&tag.Name,
			&tag.Category,
			&tag.Group,
			&tag.IsH,
			&tag.IsSpoiler,
			&tag.BlockModify,
		)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func (s *TagService) GetTagsMapByString(tagString string) (map[string][]models.Tag, error) {
	tags, err := s.GetTagListByString(tagString)
	if err != nil {
		return nil, err
	}
	var tagsMap map[string][]models.Tag = make(map[string][]models.Tag)
	for _, tag := range tags {
		tagsMap[tag.Category] = append(tagsMap[tag.Category], tag)
	}
	return tagsMap, nil
}

func (s *TagService) GetTagMap() (map[string][]string, error) {
	list, err := s.ListTags()
	if err != nil {
		return nil, err
	}
	tagMap := make(map[string][]string)
	for _, tag := range list {
		tagMap[tag.Category] = append(tagMap[tag.Category], tag.Name)
	}
	return tagMap, nil
}

// CountTags 返回 Tag 表中的记录总数
func (s *TagService) CountTags() (int, error) {
	query := `SELECT COUNT(*) FROM tags`
	row := s.db.QueryRowContext(s.ctx, query)

	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
