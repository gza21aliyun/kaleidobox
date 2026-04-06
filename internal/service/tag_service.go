package service

import (
	"context"
	"database/sql"
	"fmt"
	"lunabox/internal/appconf"
	"lunabox/internal/models"
	"strings"
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
	// fmt.Println("创建标签成功：", tag.Name, " ", tag.Category)
	return err
}

func (s *TagService) CreateOrUpdateTag(newTag models.Tag) error {
	tag, err := s.GetTagByName(newTag.Name)
	cate := newTag.Category
	if cate == "" {
		if tag != nil {
			cate = tag.Category
		}
		if cate == "" {
			cate = models.TagCategoryOther
		}
	}
	if err != nil || tag == nil {
		if err == sql.ErrNoRows || tag == nil {
			tag = &models.Tag{
				Name:        newTag.Name,
				Category:    cate,
				IsH:         newTag.BlockModify,
				IsSpoiler:   newTag.IsSpoiler,
				BlockModify: newTag.BlockModify,
			}
			return s.CreateTag(tag)
		}
		return err
	} else {
		if !newTag.BlockModify {
			newTag.Category = cate
		}
		return s.UpdateTag(&newTag)
	}
	return err
}

func (s *TagService) CreateOrUpdateTagMapArray(tagMapArray map[string][]models.Tag) error {

	for _, names := range tagMapArray {
		for _, name := range names {
			err := s.CreateOrUpdateTag(name)
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

func (s *TagService) ListGroups() ([]string, error) {
	query := `
		SELECT DISTINCT group_name
		FROM tags
		WHERE group_name IS NOT NULL AND group_name != ''
		ORDER BY group_name
	`
	rows, err := s.db.QueryContext(s.ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var group string
		err := rows.Scan(&group)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	// 检查迭代过程中是否有错误
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return groups, nil
}

// UpdateTagGroup 更新标签的分组
func (s *TagService) UpdateTagGroup(tagName string, groupName string) error {
	query := `
		UPDATE tags 
		SET group_name = ? 
		WHERE name = ?
	`
	_, err := s.db.ExecContext(s.ctx, query, groupName, tagName)
	return err
}

// UpdateTagsGroup 批量更新多个标签的分组
func (s *TagService) UpdateTagsGroup(tagNames []string, groupName string) error {
	if len(tagNames) == 0 {
		return nil
	}

	// 构建占位符
	placeholders := make([]string, len(tagNames))
	args := make([]interface{}, len(tagNames)+1)
	args[0] = groupName

	for i, tagName := range tagNames {
		placeholders[i] = "?"
		args[i+1] = tagName
	}

	query := fmt.Sprintf(`
		UPDATE tags 
		SET group_name = ? 
		WHERE name IN (%s)
	`, strings.Join(placeholders, ","))

	_, err := s.db.ExecContext(s.ctx, query, args...)
	return err
}

// DeleteTagGroup 删除标签分组（将该分组下的所有标签的group_name设为空）
func (s *TagService) DeleteTagGroup(groupName string) error {
	query := `
		UPDATE tags 
		SET group_name = '' 
		WHERE group_name = ?
	`
	_, err := s.db.ExecContext(s.ctx, query, groupName)
	return err
}

// ListTags 查询所有 Tag 记录
func (s *TagService) ListTags() ([]*models.Tag, error) {
	fmt.Println("ListTags called")
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
	fmt.Println("ListTags found: ", len(tags))
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

func (s *TagService) GetTagListByGroup(group string) ([]string, error) {
	query := `
		SELECT name
		FROM tags
		WHERE group_name = ?
	`
	rows, err := s.db.QueryContext(s.ctx, query, group)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tagName string
		err := rows.Scan(
			&tagName,
		)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tagName)
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

func (s *TagService) ManageTags(tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	// 使用 array_contains 函数
	query := `
        DELETE FROM tags 
        WHERE NOT array_contains(?, name)
    `

	_, err := s.db.ExecContext(s.ctx, query, tags)
	return err
}

func (s *TagService) GetBrands() ([]string, error) {
	query := `
		SELECT DISTINCT name
		FROM tags
		WHERE category = ?
		ORDER BY name
	`
	rows, err := s.db.QueryContext(s.ctx, query, models.TagCategoryBrand)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var brands []string
	for rows.Next() {
		var group string
		err := rows.Scan(&group)
		if err != nil {
			return nil, err
		}
		brands = append(brands, group)
	}

	// 检查迭代过程中是否有错误
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return brands, nil
}

func (s *TagService) GetSeries() ([]string, error) {
	query := `
		SELECT DISTINCT name
		FROM tags
		WHERE category = ?
		ORDER BY name
	`
	rows, err := s.db.QueryContext(s.ctx, query, models.TagCategorySeries)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var brands []string
	for rows.Next() {
		var group string
		err := rows.Scan(&group)
		if err != nil {
			return nil, err
		}
		brands = append(brands, group)
	}

	// 检查迭代过程中是否有错误
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return brands, nil
}
