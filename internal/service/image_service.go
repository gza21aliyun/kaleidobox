package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"lunabox/internal/appconf"
	"lunabox/internal/models"
	"lunabox/internal/utils"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"lunabox/internal/applog"

	"os"

	"github.com/go-vgo/robotgo"

	"github.com/google/uuid"
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")
	// kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	// procGetAsyncKeyState           = user32.NewProc("GetAsyncKeyState")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	// procOpenProcess                = kernel32.NewProc("OpenProcess")
	// procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	// procCloseHandle                = kernel32.NewProc("CloseHandle")

	// procGetDC                  = user32.NewProc("GetDC")
	// procReleaseDC              = user32.NewProc("ReleaseDC")
	// procGetClientRect          = user32.NewProc("GetClientRect")
	// procClientToScreen         = user32.NewProc("ClientToScreen")
	// procBitBlt                 = gdi32.NewProc("BitBlt")
	// procCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	// procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	// procSelectObject           = gdi32.NewProc("SelectObject")
	// procDeleteDC               = gdi32.NewProc("DeleteDC")
	// procDeleteObject           = gdi32.NewProc("DeleteObject")
	// gdi32                      = syscall.NewLazyDLL("gdi32.dll")
	// procKeybdEvent             = user32.NewProc("keybd_event")
)

const (
	SRCCOPY = 0x00CC0020
)

type ImageService struct {
	ctx    context.Context
	db     *sql.DB
	config *appconf.AppConfig
}

// Rect 结构体定义
type Rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

// Point 结构体定义
type Point struct {
	X int32
	Y int32
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
func (s *ImageService) CreateImageBackup(imageBackup models.ImageBackup) error {
	query := `
		INSERT INTO image_backups (url, local_path, subject_id, subject_type, image_type)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(s.ctx, query,
		imageBackup.Url,
		imageBackup.LocalPath,
		imageBackup.SubjectId,
		imageBackup.SubjectType,
		imageBackup.ImageType,
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
		imageBackup := models.ImageBackup{
			Url:       url,
			LocalPath: "",
		}
		return s.CreateImageBackup(imageBackup)
	}

	return nil
}

func (s *ImageService) CreateOrUpdateImageBackup(imageBackup models.ImageBackup) error {
	// 首先尝试查询是否存在该记录
	existing, err := s.GetImageBackupByUrl(imageBackup.Url)
	if err != nil {
		return err
	}

	// 如果不存在则创建新记录
	if existing == nil {
		applog.LogDebugf(s.ctx, "ImageService.CreateOrUpdateImageBackup: existing is nil, create new record %s\n", imageBackup.Url)
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

func (s *ImageService) FetchImages(id string, subjectType int, imageType int) ([]models.ImageBackup, error) {
	count, err := s.CountImageBackups()
	applog.LogInfof(s.ctx, "FetchImages start, count:%d, err:%v\n", count, err)
	// query := fmt.Sprintf(`
	// 	SELECT url, local_path, subject_id, subject_type, image_type
	// 	FROM image_backups
	// 	WHERE subject_id = %s AND subject_type = %d AND image_type = %d
	// `, id, subjectType, imageType)
	query := `
		SELECT url, local_path, subject_id, subject_type, image_type
		FROM image_backups
		WHERE subject_id = ? AND subject_type = ? AND image_type = ?
	`
	rs, err := s.FetchImageBackups(query, id, subjectType, imageType)
	applog.LogInfof(s.ctx, "FetchImages count:%d\n", len(rs))
	return rs, err
}

func (s *ImageService) FetchImage(id string, subjectType int, imageType int) (models.ImageBackup, error) {
	list, err := s.FetchImages(id, subjectType, imageType)
	if err != nil {
		return models.ImageBackup{}, err
	}
	if len(list) == 0 {
		return models.ImageBackup{}, errors.New("not found")
	}
	return list[0], nil
}

// ListImageBackups 查询所有 ImageBackup 记录
func (s *ImageService) ListImageBackups() ([]models.ImageBackup, error) {
	query := `
		SELECT url, local_path, subject_id, subject_type, image_type
		FROM image_backups
	`
	return s.FetchImageBackups(query, "", 0, 0)
}

func (s *ImageService) FetchImageBackups(query string, id string, subjectType int, imageType int) ([]models.ImageBackup, error) {
	var err error
	var rows *sql.Rows
	if id == "" {
		rows, err = s.db.QueryContext(s.ctx, query)
	} else {
		rows, err = s.db.QueryContext(s.ctx, query, id, subjectType, imageType)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var imageBackups []models.ImageBackup
	for rows.Next() {
		var imageBackup models.ImageBackup
		err := rows.Scan(
			&imageBackup.Url,
			&imageBackup.LocalPath,
			&imageBackup.SubjectId,
			&imageBackup.SubjectType,
			&imageBackup.ImageType,
		)
		if err != nil {
			return nil, err
		}
		imageBackups = append(imageBackups, imageBackup)
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

func (s *ImageService) DownloadImages() error {
	var count = 0
	query := `
		SELECT url, local_path, subject_id, subject_type, image_type
		FROM image_backups
		WHERE local_path = '' OR local_path IS NULL
	`
	list, err := s.FetchImageBackups(query, "", 0, 0)
	if err != nil {
		return err
	}

	for _, imageBackup := range list {
		path, err := utils.GetDataDir()
		if err != nil {
			continue
		}
		path = fmt.Sprintf(`%s\images`, path)
		_, err = os.Stat(path)
		if err != nil {
			err := os.MkdirAll(path, os.ModePerm)
			if err != nil {
				continue
			}
		}
		path = fmt.Sprintf(`%s\%s`, path, imageBackup.SubjectId)
		_, err = os.Stat(path)
		if err != nil {
			err := os.MkdirAll(path, os.ModePerm)
			if err != nil {
				continue
			}
		}
		ext := filepath.Ext(imageBackup.Url)
		fileName := fmt.Sprintf(`%s\%s.%s`, path, uuid.New().String(), ext)
		err = DownloadImage(imageBackup.Url, fileName)
		if err != nil {
			applog.LogErrorf(s.ctx, "下载图片 %s 失败：%v", imageBackup.Url, err)
			// 清理下载失败的文件
			os.Remove(fileName)
			continue
		}
		err = s.UpdateImageBackup(&models.ImageBackup{
			Url:         imageBackup.Url,
			LocalPath:   fileName,
			SubjectId:   imageBackup.SubjectId,
			SubjectType: imageBackup.SubjectType,
			ImageType:   imageBackup.ImageType,
		})
		count++
	}

	if count == 0 && len(list) > 0 {
		return errors.New("no images to download")
	}
	return nil
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

func (s *ImageService) TakeScreenshotOfFocusedWindow(gameId string) {
	// pid := getCurrentForegroundProcessId()
	// 1. 获取当前焦点窗口的句柄
	// hwnd := robotgo.GetHWND()
	// if hwnd == 0 {
	// 	return "", fmt.Errorf("failed to get focused window handle")
	// }

	// 2. 获取窗口的位置和尺寸
	// x, y, width, height := robotgo.GetBounds(int(pid))
	// if width <= 0 || height <= 0 {
	// 	return "", fmt.Errorf("invalid window dimensions: %dx%d", width, height)
	// }

	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		applog.LogErrorf(s.ctx, "Failed to get foreground window")
		return
	}

	// 获取窗口位置和大小
	var rect Rect
	ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		applog.LogErrorf(s.ctx, "Failed to get window rectangle")
		return
	}
	rect.Left += 8
	rect.Bottom -= 8
	rect.Right -= 8
	rect.Top += 4

	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)

	name := uuid.New().String()
	dataDir, err := utils.GetDataDir()
	if err != nil {
		return

	}
	path := fmt.Sprintf(`%s\%s`, dataDir, "images")
	// path = `C:\temp\projects`
	_, err = os.Stat(path)
	if err != nil {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return
		}
	}
	path = fmt.Sprintf(`%s\%s`, path, gameId)
	_, err = os.Stat(path)
	if err != nil {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return
		}
	}

	fileName := fmt.Sprintf(`%s\%s.jpeg`, path, name)
	// image, err := robotgo.Capture(x, y, width, height)
	// if err != nil {
	// 	return "", fmt.Errorf("failed to capture: %v", err)
	// }
	// 3. 截图指定区域
	// err = robotgo.SaveJpeg(image, fileName, 90)
	err = robotgo.SaveCapture(fileName, int(rect.Left), int(rect.Top), width, height)
	if err != nil {
		applog.LogInfof(s.ctx, "failed to save screenshot: %v", err)
		return
	}
	err = s.CreateOrUpdateImageBackup(models.ImageBackup{
		Url:         fileName,
		LocalPath:   fileName,
		SubjectId:   gameId,
		SubjectType: 0,
		ImageType:   3,
	})

}

func (s *ImageService) SaveGameImages(gameEntity models.GameEntity) error {
	//game images
	var err error = nil
	for _, image := range strings.Split(gameEntity.Game.Images, ",") {
		backup, _ := s.GetImageBackupByUrl(image)
		if backup != nil && backup.Url != "" {
			continue
		}
		backup = &models.ImageBackup{
			Url:         image,
			LocalPath:   "",
			SubjectId:   gameEntity.Game.ID,
			SubjectType: 0,
			ImageType:   2,
		}
		err = s.CreateImageBackup(*backup)
	}
	cover, _ := s.GetImageBackupByUrl(gameEntity.Game.CoverURL)
	if cover == nil || cover.Url == "" {
		cover = &models.ImageBackup{
			Url:         gameEntity.Game.CoverURL,
			LocalPath:   "",
			SubjectId:   gameEntity.Game.ID,
			SubjectType: 0,
			ImageType:   0,
		}
		err = s.CreateImageBackup(*cover)
	}
	//works
	// for _, work := range utils.MapToArray(gameEntity.WorksMap) {
	// 	staffImage, _ := s.GetImageBackupByUrl(work.StaffImage)
	// 	if staffImage.Url == "" {
	// 		staffImage = &models.ImageBackup{
	// 			Url:         work.StaffImage,
	// 			LocalPath:   "",
	// 			SubjectId:   work.ID,
	// 			SubjectType: 2,
	// 			ImageType:   0,
	// 		}
	// 		err = s.CreateImageBackup(*staffImage)
	// 	}
	// }
	return err
}

func DownloadImage(url, fileName string) error {
	srcFile, err := os.Open(url)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	destFile, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return err
	}
	return nil
}
