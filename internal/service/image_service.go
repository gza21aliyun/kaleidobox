package service

import (
	"context"
	"database/sql"
	"lunabox/internal/appconf"
	"lunabox/internal/models"
	"strings"
)

type ImageService struct {
	ctx    context.Context
	db     *sql.DB
	config *appconf.AppConfig
}

func NewImageService() *ImageService {
	return &ImageService{}
}

func (s *ImageService) Init(ctx context.Context, db *sql.DB, config *appconf.AppConfig) {
	s.ctx = ctx
	s.db = db
	s.config = config
}

// CreateImageBackup 创建新的 ImageBackup 记录
func (s *ImageService) CreateImageBackup(imageBackup *models.ImageBackup) error {
	query := `
		INSERT INTO image_backups (url, local_path)
		VALUES (?, ?)
	`
	_, err := s.db.ExecContext(s.ctx, query,
		imageBackup.Url,
		imageBackup.LocalPath,
	)
	return err
}

func (s *ImageService) CreateOrUpdateForUrlsStr(urlStr string) error {
	if urlStr == "" {
		return nil
	}

	// 按逗号分割URL字符串
	urls := strings.Split(urlStr, ",")

	// 过滤空字符串
	var validUrls []string
	for _, url := range urls {
		trimmedUrl := strings.TrimSpace(url)
		if trimmedUrl != "" {
			validUrls = append(validUrls, trimmedUrl)
		}
	}

	// 批量处理每个URL
	for _, url := range validUrls {
		err := s.CreateOrUpdateImageUrl(url)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ImageService) CreateOrUpdateImageUrl(url string) error {
	// 首先尝试查询是否存在该记录
	existing, err := s.GetImageBackupByUrl(url)
	if err != nil {
		return err
	}

	// 如果不存在则创建新记录
	if existing == nil {
		imageBackup := &models.ImageBackup{
			Url:       url,
			LocalPath: "",
		}
		return s.CreateImageBackup(imageBackup)
	}

	return nil
}

func (s *ImageService) CreateOrUpdateImageBackup(imageBackup *models.ImageBackup) error {
	// 首先尝试查询是否存在该记录
	existing, err := s.GetImageBackupByUrl(imageBackup.Url)
	if err != nil {
		return err
	}

	// 如果不存在则创建新记录
	if existing == nil {
		return s.CreateImageBackup(imageBackup)
	}

	// 如果存在则更新记录
	// 只有当本地路径不为空且不同时才更新
	if imageBackup.LocalPath != "" && imageBackup.LocalPath != existing.LocalPath {
		existing.LocalPath = imageBackup.LocalPath
		return s.UpdateImageBackup(existing)
	}

	return nil
}

// GetImageBackupByUrl 根据 Url 查询 ImageBackup 记录
func (s *ImageService) GetImageBackupByUrl(url string) (*models.ImageBackup, error) {
	query := `
		SELECT url, local_path
		FROM image_backups
		WHERE url = ?
	`
	row := s.db.QueryRowContext(s.ctx, query, url)

	var imageBackup models.ImageBackup
	err := row.Scan(
		&imageBackup.Url,
		&imageBackup.LocalPath,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 未找到记录
		}
		return nil, err
	}
	return &imageBackup, nil
}

// GetImageBackupByLocalPath 根据 LocalPath 查询 ImageBackup 记录
func (s *ImageService) GetImageBackupByLocalPath(localPath string) (*models.ImageBackup, error) {
	query := `
		SELECT url, local_path
		FROM image_backups
		WHERE local_path = ?
	`
	row := s.db.QueryRowContext(s.ctx, query, localPath)

	var imageBackup models.ImageBackup
	err := row.Scan(
		&imageBackup.Url,
		&imageBackup.LocalPath,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 未找到记录
		}
		return nil, err
	}
	return &imageBackup, nil
}

// UpdateImageBackup 更新 ImageBackup 记录
func (s *ImageService) UpdateImageBackup(imageBackup *models.ImageBackup) error {
	query := `
		UPDATE image_backups
		SET local_path = ?
		WHERE url = ?
	`
	_, err := s.db.ExecContext(s.ctx, query,
		imageBackup.LocalPath,
		imageBackup.Url,
	)
	return err
}

// DeleteImageBackup 删除 ImageBackup 记录
func (s *ImageService) DeleteImageBackup(url string) error {
	query := `DELETE FROM image_backups WHERE url = ?`
	_, err := s.db.ExecContext(s.ctx, query, url)
	return err
}

// ListImageBackups 查询所有 ImageBackup 记录
func (s *ImageService) ListImageBackups() ([]*models.ImageBackup, error) {
	query := `
		SELECT url, local_path
		FROM image_backups
	`
	rows, err := s.db.QueryContext(s.ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var imageBackups []*models.ImageBackup
	for rows.Next() {
		var imageBackup models.ImageBackup
		err := rows.Scan(
			&imageBackup.Url,
			&imageBackup.LocalPath,
		)
		if err != nil {
			return nil, err
		}
		imageBackups = append(imageBackups, &imageBackup)
	}
	return imageBackups, nil
}

// CountImageBackups 返回 ImageBackup 表中的记录总数
func (s *ImageService) CountImageBackups() (int, error) {
	query := `SELECT COUNT(*) FROM image_backups`
	row := s.db.QueryRowContext(s.ctx, query)

	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetImageBackupsByUrls 批量根据 URLs 查询 ImageBackup 记录
func (s *ImageService) GetImageBackupsByUrls(urls []string) ([]*models.ImageBackup, error) {
	if len(urls) == 0 {
		return []*models.ImageBackup{}, nil
	}

	// 构造占位符
	placeholders := make([]string, len(urls))
	args := make([]interface{}, len(urls))
	for i, url := range urls {
		placeholders[i] = "?"
		args[i] = url
	}

	query := `
		SELECT url, local_path
		FROM image_backups
		WHERE url IN (` + joinStrings(placeholders, ",") + `)
	`

	rows, err := s.db.QueryContext(s.ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var imageBackups []*models.ImageBackup
	for rows.Next() {
		var imageBackup models.ImageBackup
		err := rows.Scan(
			&imageBackup.Url,
			&imageBackup.LocalPath,
		)
		if err != nil {
			return nil, err
		}
		imageBackups = append(imageBackups, &imageBackup)
	}
	return imageBackups, nil
}

// 辅助函数：连接字符串切片
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
