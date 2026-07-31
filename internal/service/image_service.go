package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"lunabox/internal/appconf"
	"lunabox/internal/models"
	"lunabox/internal/utils"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"lunabox/internal/applog"

	"github.com/go-vgo/robotgo"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/google/uuid"
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")
	// kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	// procGetAsyncKeyState           = user32.NewProc("GetAsyncKeyState")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procGetSystemMetrics         = user32.NewProc("GetSystemMetrics")
	procFindWindowW              = user32.NewProc("FindWindowW")
	procGetPropW                 = user32.NewProc("GetPropW")
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

// getMagpieProp 获取 Magpie 缩放窗口的属性
func getMagpieProp(hwnd uintptr, propName string) int32 {
	propNamePtr, _ := syscall.UTF16PtrFromString(propName)
	val, _, _ := procGetPropW.Call(hwnd, uintptr(unsafe.Pointer(propNamePtr)))
	return int32(val)
}

const (
	SRCCOPY     = 0x00CC0020
	SM_CXSCREEN = 0
	SM_CYSCREEN = 1
)

type ImageService struct {
	ctx    context.Context
	db     *sql.DB
	config *appconf.AppConfig

	// 下载队列相关
	downloadQueue chan downloadTask
	wg            sync.WaitGroup
	mu            sync.Mutex
	maxConcurrent int
}

// 下载任务结构体
type downloadTask struct {
	imageUrl   string
	fileName   string
	result     chan error
	retryCount int // 重试次数
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

	// 初始化下载队列
	s.maxConcurrent = 3                            // 最大并发下载数
	s.downloadQueue = make(chan downloadTask, 100) // 队列容量

	// 启动工作协程
	for i := 0; i < s.maxConcurrent; i++ {
		s.wg.Add(1)
		go s.downloadWorker()
	}

}

// CreateImageBackup 创建新的 ImageBackup 记录
func (s *ImageService) CreateImageBackup(imageBackup models.ImageBackup) error {
	query := `
		INSERT INTO image_backups (url, local_path, subject_id, subject_type, image_type, game_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(s.ctx, query,
		imageBackup.Url,
		imageBackup.LocalPath,
		imageBackup.SubjectId,
		imageBackup.SubjectType,
		imageBackup.ImageType,
		imageBackup.GameId,
		imageBackup.CreatedAt,
	)
	if err == nil {
		fmt.Printf("图库3 成功添加：%s\n", imageBackup.Url)
	}
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
	existing, err := s.GetImageBackupByUrl(url, false)
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
	existing, err := s.GetImageBackupByUrl(imageBackup.Url, false)
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
func (s *ImageService) GetImageBackupByUrl(url string, down bool) (*models.ImageBackup, error) {
	fmt.Printf("GetImageBackupByUrl 01 url: %s\n", url)
	query := `
		SELECT url, local_path, subject_id, subject_type, image_type, game_id, created_at
		FROM image_backups
		WHERE url = ?
	`
	row := s.db.QueryRowContext(s.ctx, query, url)

	var imageBackup models.ImageBackup
	err := row.Scan(
		&imageBackup.Url,
		&imageBackup.LocalPath,
		&imageBackup.SubjectId,
		&imageBackup.SubjectType,
		&imageBackup.ImageType,
		&imageBackup.GameId,
		&imageBackup.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 未找到记录
		}
		return nil, err
	}
	if s.config.AutoDownloadImages && down {
		newList, err := s.DownloadImageBackups([]models.ImageBackup{imageBackup})
		fmt.Printf("GetImageBackupByUrl 04 url: %s\n", url)
		return &newList[0], err
	}
	// else {
	// 	go s.DownloadImageBackups([]models.ImageBackup{imageBackup})
	// }

	return &imageBackup, nil
}

// GetImageBackupByLocalPath 根据 LocalPath 查询 ImageBackup 记录
// func (s *ImageService) GetImageBackupByLocalPath(localPath string) (*models.ImageBackup, error) {
// 	query := `
// 		SELECT url, local_path
// 		FROM image_backups
// 		WHERE local_path = ?
// 	`
// 	row := s.db.QueryRowContext(s.ctx, query, localPath)

// 	var imageBackup models.ImageBackup
// 	err := row.Scan(
// 		&imageBackup.Url,
// 		&imageBackup.LocalPath,
// 	)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil // 未找到记录
// 		}
// 		return nil, err
// 	}
// 	return &imageBackup, nil
// }

// UpdateImageBackup 更新 ImageBackup 记录
func (s *ImageService) UpdateImageBackup(imageBackup *models.ImageBackup) error {
	query := `
		UPDATE image_backups
		SET local_path = ?, subject_id = ?, subject_type = ?, image_type = ?
		WHERE url = ?
	`
	_, err := s.db.ExecContext(s.ctx, query,
		imageBackup.LocalPath,
		imageBackup.SubjectId,
		imageBackup.SubjectType,
		imageBackup.ImageType,
		imageBackup.Url,
	)
	return err
}

// DeleteImageBackup 删除 ImageBackup 记录，同时删除本地图片文件
func (s *ImageService) DeleteImageBackup(url string) error {
	// 先查询记录以获取本地文件路径
	record, err := s.GetImageBackupByUrl(url, false)
	if err != nil {
		return err
	}

	// 删除数据库记录
	query := `DELETE FROM image_backups WHERE url = ?`
	_, err = s.db.ExecContext(s.ctx, query, url)
	if err != nil {
		return err
	}

	// 如果有本地文件，删除它
	if record != nil && record.LocalPath != "" {
		if err := os.Remove(record.LocalPath); err != nil {
			applog.LogWarningf(s.ctx, "删除本地图片文件失败 %s: %v", record.LocalPath, err)
		}
	}

	return nil
}

func (s *ImageService) FetchImages(id string, subjectType int, imageType int, download bool) ([]models.ImageBackup, error) {
	_, err := s.CountImageBackups()
	// applog.LogInfof(s.ctx, "FetchImages start, count:%d, err:%v\n", count, err)
	// query := fmt.Sprintf(`
	// 	SELECT url, local_path, subject_id, subject_type, image_type
	// 	FROM image_backups
	// 	WHERE subject_id = %s AND subject_type = %d AND image_type = %d
	// `, id, subjectType, imageType)
	query := `
		SELECT url, local_path, subject_id, subject_type, image_type, game_id, created_at
		FROM image_backups
		WHERE subject_id = ? AND subject_type = ? AND image_type = ?
	`
	rs, err := s.FetchImageBackups(query, id, subjectType, imageType, download)
	// applog.LogInfof(s.ctx, "FetchImages count:%d\n", len(rs))
	return rs, err
}

func (s *ImageService) FetchImage(id string, subjectType int, imageType int) (models.ImageBackup, error) {
	list, err := s.FetchImages(id, subjectType, imageType, false)
	if err != nil {
		return models.ImageBackup{}, err
	}
	if len(list) == 0 {
		return models.ImageBackup{}, errors.New("not found")
	}
	if s.config.AutoDownloadImages {
		newList, err := s.DownloadImageBackups(list)
		return newList[0], err
	}
	return list[0], nil
}

// FetchGetchuImages 下载 Getchu 图片到临时文件夹，返回本地路径列表
func (s *ImageService) FetchGetchuImages(imageUrls []string) ([]string, error) {
	if len(imageUrls) == 0 {
		return []string{}, nil
	}

	// 准备临时目录
	dataDir, err := utils.GetDataDir()
	if err != nil {
		applog.InfoLogSaveAppLog("FetchGetchuImages: failed to get data dir: %v", err)
		return []string{}, err
	}
	tempDir := filepath.Join(dataDir, "monthly", "temp")
	if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
		applog.InfoLogSaveAppLog("FetchGetchuImages: failed to create temp dir: %v", err)
		return []string{}, err
	}

	// 下载所有图片
	var localPaths []string
	for _, imageUrl := range imageUrls {
		if imageUrl == "" {
			continue
		}
		localPath := filepath.Join(tempDir, fmt.Sprintf("%s.jpg", uuid.New().String()))
		if err := s.downloadImageWithReferer(imageUrl, localPath); err != nil {
			applog.InfoLogSaveAppLog("FetchGetchuImages: failed to download [%s]: %v", imageUrl, err)
			continue
		}
		applog.InfoLogSaveAppLog("FetchGetchuImages: downloaded [%s] to [%s]", imageUrl, localPath)
		localPaths = append(localPaths, localPath)
	}

	return localPaths, nil
}

// downloadImageWithReferer 下载图片，带 Referer 绕过防盗链
func (s *ImageService) downloadImageWithReferer(imageUrl, localPath string) error {
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequest("GET", imageUrl, nil)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")

	// 设置 Referer
	referer := "https://www.getchu.com/"
	if strings.HasPrefix(imageUrl, "https://www.getchu.com/") {
		referer = "https://www.getchu.com/"
	} else if strings.HasPrefix(imageUrl, "https://evalidate.getchu.com/") {
		referer = "https://www.getchu.com/"
	}
	req.Header.Set("Referer", referer)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create file failed: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(localPath)
		return fmt.Errorf("write file failed: %w", err)
	}

	return nil
}

// ListImageBackups 查询所有 ImageBackup 记录
func (s *ImageService) ListImageBackups() ([]models.ImageBackup, error) {
	query := `
		SELECT url, local_path, subject_id, subject_type, image_type, game_id, created_at
		FROM image_backups
	`
	return s.FetchImageBackups(query, "", 0, 0, false)
}

func (s *ImageService) FetchImageBackups(query string, id string, subjectType int, imageType int, download bool) ([]models.ImageBackup, error) {
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
			&imageBackup.GameId,
			&imageBackup.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		imageBackups = append(imageBackups, imageBackup)
	}
	if s.config.AutoDownloadImages && download {
		return s.DownloadImageBackups(imageBackups)
	}
	// go s.DownloadImageBackups(imageBackups)
	return imageBackups, nil
}

func (s *ImageService) FetchEmptyGalleryGames() ([]string, error) {
	var err error
	var rows *sql.Rows
	query := `
		SELECT g.id FROM games g
		WHERE NOT EXISTS (
			SELECT 1 FROM image_backups ib WHERE ib.subject_type = ? AND ib.subject_id = g.id
		)
	`
	rows, err = s.db.QueryContext(s.ctx, query, 0)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gameIds []string
	for rows.Next() {
		var gameId string
		err = rows.Scan(
			&gameId,
		)
		if err != nil {

			continue
		}
		gameIds = append(gameIds, gameId)
	}
	// go s.DownloadImageBackups(imageBackups)
	fmt.Println("FetchEmptyGalleryGames:", len(gameIds))
	return gameIds, nil
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
// func (s *ImageService) GetImageBackupsByUrls(urls []string) ([]*models.ImageBackup, error) {
// 	if len(urls) == 0 {
// 		return []*models.ImageBackup{}, nil
// 	}

// 	// 构造占位符
// 	placeholders := make([]string, len(urls))
// 	args := make([]interface{}, len(urls))
// 	for i, url := range urls {
// 		placeholders[i] = "?"
// 		args[i] = url
// 	}

// 	query := `
// 		SELECT url, local_path
// 		FROM image_backups
// 		WHERE url IN (` + joinStrings(placeholders, ",") + `)
// 	`

// 	rows, err := s.db.QueryContext(s.ctx, query, args...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var imageBackups []*models.ImageBackup
// 	for rows.Next() {
// 		var imageBackup models.ImageBackup
// 		err := rows.Scan(
// 			&imageBackup.Url,
// 			&imageBackup.LocalPath,
// 		)
// 		if err != nil {
// 			return nil, err
// 		}
// 		imageBackups = append(imageBackups, &imageBackup)
// 	}
// 	return imageBackups, nil
// }

func (s *ImageService) DownloadImageBackups(list []models.ImageBackup) ([]models.ImageBackup, error) {
	newList := []models.ImageBackup{}
	if !s.config.AutoDownloadImages {
		return list, nil
	}
	if len(list) == 0 {
		return list, nil
	}
	for _, imageBackup := range list {
		if imageBackup.Url == "" {
			newList = append(newList, imageBackup)
			continue
		}
		if imageBackup.LocalPath != "" {
			info, err := os.Stat(imageBackup.LocalPath)
			if err != nil {
				imageBackup.LocalPath = ""
			} else if info.Size() > 3000 {
				newList = append(newList, imageBackup)
				continue
			} else {
				os.Remove(imageBackup.LocalPath)
				imageBackup.LocalPath = ""
			}
		}
		path, err := utils.GetDataDir()
		if err != nil {
			newList = append(newList, imageBackup)
			continue
		}
		path = fmt.Sprintf(`%s\images`, path)
		_, err = os.Stat(path)
		if err != nil {
			err := os.MkdirAll(path, os.ModePerm)
			if err != nil {
				newList = append(newList, imageBackup)
				continue
			}
		}
		path = fmt.Sprintf(`%s\%s`, path, imageBackup.GameId)
		_, err = os.Stat(path)
		if err != nil {
			err := os.MkdirAll(path, os.ModePerm)
			if err != nil {
				newList = append(newList, imageBackup)
				continue
			}
		}
		ext := filepath.Ext(imageBackup.Url)
		if queryIndex := strings.Index(ext, "?"); queryIndex != -1 {
			ext = ext[:queryIndex]
		}
		fileName := fmt.Sprintf(`%s\%s%s`, path, uuid.New().String(), ext)
		err = s.DownloadImage(imageBackup.Url, fileName)
		if err != nil {
			applog.LogErrorf(s.ctx, "下载图片3 %s, url:%s, 失败：%v", fileName, imageBackup.Url, err)
			// 清理下载失败的文件
			os.Remove(fileName)
			newList = append(newList, imageBackup)
			continue
		}
		newIb := models.ImageBackup{
			Url:         imageBackup.Url,
			LocalPath:   fileName,
			SubjectId:   imageBackup.SubjectId,
			SubjectType: imageBackup.SubjectType,
			ImageType:   imageBackup.ImageType,
		}
		err = s.UpdateImageBackup(&newIb)
		if err == nil {
			applog.InfoLogSaveAppLog("下载图片 %s 成功：%s", imageBackup.Url, fileName)
		}
		newList = append(newList, newIb)
	}
	return newList, nil
}

func (s *ImageService) DownloadImages() error {
	var count = 0
	query := `
		SELECT url, local_path, subject_id, subject_type, image_type, created_at
		FROM image_backups
		WHERE local_path = '' OR local_path IS NULL
	`
	list, err := s.FetchImageBackups(query, "", 0, 0, false)
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
		path = fmt.Sprintf(`%s\%s`, path, imageBackup.GameId)
		_, err = os.Stat(path)
		if err != nil {
			err := os.MkdirAll(path, os.ModePerm)
			if err != nil {
				continue
			}
		}
		ext := filepath.Ext(imageBackup.Url)
		if queryIndex := strings.Index(ext, "?"); queryIndex != -1 {
			ext = ext[:queryIndex]
		}
		fileName := fmt.Sprintf(`%s\%s.%s`, path, uuid.New().String(), ext)
		err = s.DownloadImage(imageBackup.Url, fileName)
		if err != nil {
			fileName := fmt.Sprintf(`%s\%s.%s`, path, uuid.New().String(), ext)
			applog.LogErrorf(s.ctx, "下载图片2 %s 失败：%v, f:%s", fileName, imageBackup.Url, err)
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

// ioCopyWithContext 带上下文的 io.Copy 函数
func ioCopyWithContext(ctx context.Context, dst io.Writer, src io.Reader) (written int64, err error) {
	type result struct {
		n   int64
		err error
	}

	ch := make(chan result, 1)
	go func() {
		n, err := io.Copy(dst, src)
		ch <- result{n, err}
	}()

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case res := <-ch:
		return res.n, res.err
	}
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

	// 检测 Magpie 缩放窗口是否存在（类名：Window_Magpie_967EB565-6F73-4E94-AE53-00CC42592A22）
	// 只有当这个窗口存在时才表示 Magpie 真正激活了缩放，而不只是进程在运行

	applog.InfoLogSaveAppLog("截图gameId:%s", gameId)
	if gameId == "" {
		return
	}

	magpieWindowClassName, _ := syscall.UTF16PtrFromString("Window_Magpie_967EB565-6F73-4E94-AE53-00CC42592A22")
	hwndMagpie, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(magpieWindowClassName)), 0)

	var x, y int
	var width, height int

	if hwndMagpie != 0 {
		// Magpie 缩放已激活，从缩放窗口属性中获取精确缩放区域
		// 参考: https://github.com/Blinue/Magpie/blob/dev/docs/%E4%BB%A5%E7%BC%96%E7%A8%8B%E6%96%B9%E5%BC%8F%E4%B8%8E%20Magpie%20%E4%BA%A4%E4%BA%92.md
		destLeft := getMagpieProp(hwndMagpie, "Magpie.DestLeft")
		destTop := getMagpieProp(hwndMagpie, "Magpie.DestTop")
		destRight := getMagpieProp(hwndMagpie, "Magpie.DestRight")
		destBottom := getMagpieProp(hwndMagpie, "Magpie.DestBottom")

		x = int(destLeft)
		y = int(destTop)
		width = int(destRight - destLeft)
		height = int(destBottom - destTop)

		if width <= 0 || height <= 0 {
			// fallback: 如果属性获取失败，使用全屏
			screenWidth, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
			screenHeight, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
			x = 0
			y = 0
			width = int(screenWidth)
			height = int(screenHeight)
			applog.LogInfof(s.ctx, "Magpie scaling is active, but dest rect invalid, using fullscreen screenshot: %dx%d", width, height)
		} else {
			applog.LogInfof(s.ctx, "Magpie scaling is active, using dest rect screenshot: (%d,%d) %dx%d", x, y, width, height)
		}
	} else {
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

		x = int(rect.Left)
		y = int(rect.Top)
		width = int(rect.Right - rect.Left)
		height = int(rect.Bottom - rect.Top)
	}

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
	err = robotgo.SaveCapture(fileName, x, y, width, height)
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
		CreatedAt:   time.Now(),
	})

	// 发送事件通知前端刷新截图
	if s.ctx != nil {
		runtime.EventsEmit(s.ctx, "screenshot:saved", gameId)
	}

	// 发送 Windows 通知
	go func() {
		uriPath := strings.ReplaceAll(filepath.ToSlash(fileName), " ", "%20")
		script := fmt.Sprintf(`
			[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
			[Windows.UI.Notifications.ToastNotification, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
			[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
			$template = @"
			<toast activationType="protocol" launch="file:///%s">
				<visual>
					<binding template="ToastGeneric">
						<text>游戏截图已保存</text>
						<image placement="inline" src="file:///%s" />
					</binding>
				</visual>
			</toast>
"@
			$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
			$xml.LoadXml($template)
			$toast = New-Object Windows.UI.Notifications.ToastNotification $xml
			[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("KaleidoBox").Show($toast)
		`, uriPath, uriPath)
		cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: 0x08000000,
		}
		_ = cmd.Start()
		if cmd.Process != nil {
			cmd.Process.Release()
		}
	}()

}

func (s *ImageService) SaveGameImages(gameEntity models.GameEntity) error {
	//game images
	var err error = nil
	for _, image := range strings.Split(gameEntity.Game.Images, ",") {
		fmt.Printf("图库2：%s\n", image)
		backup, _ := s.GetImageBackupByUrl(image, false)
		localPath := ""
		if backup != nil {
			localPath = backup.LocalPath
		}
		newBackup := models.ImageBackup{
			Url:         image,
			LocalPath:   localPath,
			SubjectId:   gameEntity.Game.ID,
			SubjectType: 0,
			ImageType:   2,
			CreatedAt:   time.Now(),
			GameId:      gameEntity.Game.ID,
		}
		if backup != nil {
			err = s.UpdateImageBackup(&newBackup)
			continue
		}
		fmt.Printf("图库3：%s\n", image)

		err = s.CreateImageBackup(newBackup)
	}
	cover, _ := s.GetImageBackupByUrl(gameEntity.Game.CoverURL, false)
	localCover := ""
	if cover != nil {
		localCover = cover.LocalPath
	}
	newCover := models.ImageBackup{
		Url:         gameEntity.Game.CoverURL,
		LocalPath:   localCover,
		SubjectId:   gameEntity.Game.ID,
		SubjectType: 0,
		ImageType:   2,
		CreatedAt:   time.Now(),
		GameId:      gameEntity.Game.ID,
	}
	if cover == nil {
		err = s.CreateImageBackup(newCover)
	} else {
		err = s.UpdateImageBackup(&newCover)
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

// // 全局 ImageService 实例，用于处理下载队列
// var globalImageService *ImageService

// // SetGlobalImageService 设置全局 ImageService 实例
// func SetGlobalImageService(service *ImageService) {
// 	globalImageService = service
// }

// DownloadImage 将下载任务加入队列并等待完成
func (s *ImageService) DownloadImage(imageUrl, fileName string) error {

	// 创建下载任务
	task := downloadTask{
		imageUrl:   imageUrl,
		fileName:   fileName,
		result:     make(chan error, 1),
		retryCount: 0, // 初始重试次数为 0
	}

	// 将任务加入队列
	s.downloadQueue <- task

	// 等待任务完成
	err := <-task.result
	return err
}

// determineReferer 根据图片 URL 确定应该使用的 Referer
// 下载工作协程
func (s *ImageService) downloadWorker() {
	defer s.wg.Done()

	for task := range s.downloadQueue {
		// 根据重试次数设置超时时间
		timeout := 5 * time.Second
		if task.retryCount > 0 {
			timeout = 30 * time.Second
		}

		// 创建一个带超时的上下文
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// 使用通道来处理超时
		errChan := make(chan error, 1)
		go func() {
			errChan <- s.downloadImageInternal(ctx, task.imageUrl, task.fileName, timeout)
		}()

		var err error
		select {
		case err = <-errChan:
			if err != nil {
				// 处理下载错误
				fmt.Printf("downloadWorker err: %v\n", err)
			}
			// 下载完成
		case <-ctx.Done():
			// 超时
			err = fmt.Errorf("下载超时")
		}

		// 处理重试逻辑
		if err != nil && task.retryCount == 0 {
			// 第一次超时，将任务重新加入队列
			task.retryCount++
			s.downloadQueue <- task
			continue
		} else {
			// 第二次尝试或下载成功，返回结果
			task.result <- err
			close(task.result)
		}
	}
}

// 内部下载函数
func (s *ImageService) downloadImageInternal(ctx context.Context, imageUrl, fileName string, timeout time.Duration) error {
	fmt.Printf("GetImageBackupByUrl 022 url: %s\n", imageUrl)
	url := imageUrl
	if strings.HasPrefix(url, "//gyutto.com") {
		url = strings.ReplaceAll(url, "//gyutto.com", "https://image.gyutto.com")
	}
	if strings.HasPrefix(url, "//img.dlsite.jp") {
		url = strings.ReplaceAll(url, "//img.dlsite.jp", "https://img.dlsite.jp")
	}
	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败：%w", err)
	}

	// 设置请求头，伪装成从网站页面访问
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ja-JP,ja;q=0.9,en-US;q=0.8")

	// 关键：设置 Referer 来绕过防盗链
	referer := determineReferer(url)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	// 执行请求
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("下载图片失败：%w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("HTTP 403 Forbidden - 可能是防盗链限制，Referer: %s", referer)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP 请求失败，状态码：%d", resp.StatusCode)
	}

	destFile, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("创建文件失败：%w", err)
	}
	defer destFile.Close()

	if _, err := ioCopyWithContext(ctx, destFile, resp.Body); err != nil {
		return fmt.Errorf("写入文件失败：%w", err)
	}
	// 检查文件大小
	fileInfo, err := destFile.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败：%w", err)
	}
	fileSize := fileInfo.Size()

	// 可选：验证文件大小是否合理
	if fileSize < 3000 {
		defer os.Remove(fileName)
		return fmt.Errorf("下载的文件大小小于3000字节，可能不是有效的图片文件")
	}
	fmt.Printf("GetImageBackupByUrl 032 url: %s\n", imageUrl)

	return nil
}

func determineReferer(imageURL string) string {
	// Getchu
	if strings.Contains(imageURL, "getchu.com") {
		return "https://www.getchu.com/"
	}

	if strings.Contains(imageURL, "gyutto.com") {
		return "https://gyutto.com/"
	}

	// DMM/DMM GAMES
	if strings.Contains(imageURL, "dmm.co.jp") || strings.Contains(imageURL, "dmm.com") {
		return "https://www.dmm.com/"
	}

	// MGStage
	if strings.Contains(imageURL, "mgstage.com") {
		return "https://www.mgstage.com/"
	}

	// Digiket
	if strings.Contains(imageURL, "digiket.com") {
		return "https://www.digiket.com/"
	}

	// DLsite
	if strings.Contains(imageURL, "dl.site") || strings.Contains(imageURL, "dlsite.com") {
		return "https://www.dlsite.com/"
	}

	// FANZA
	if strings.Contains(imageURL, "fanza.jp") || strings.Contains(imageURL, "fanza.com") {
		return "https://www.fanza.jp/"
	}

	// 默认返回一个通用的 Referer（如果是日本网站）
	if strings.Contains(imageURL, ".jp") {
		return "https://www.google.co.jp/"
	}

	// 其他情况不设置 Referer
	return ""
}

// ScaleImageWithMagpie 在当前应用窗口内用 Magpie 裁剪并全屏化图片
// imgLeft/imgTop/imgWidth/imgHeight: 图片在视口中的位置和尺寸（CSS 像素）
// viewportWidth/viewportHeight: 视口尺寸（window.innerWidth/innerHeight）
func (s *ImageService) ScaleImageWithMagpie(imgLeft, imgTop, imgWidth, imgHeight, viewportWidth, viewportHeight int) error {
	if s.config.MagpiePath == "" {
		return fmt.Errorf("Magpie 路径未设置")
	}

	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return fmt.Errorf("无法获取前台窗口")
	}

	procGetClientRect := user32.NewProc("GetClientRect")
	procClientToScreen := user32.NewProc("ClientToScreen")
	procSetForegroundWindow := user32.NewProc("SetForegroundWindow")

	var winRect [4]int32
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&winRect[0])))
	winWidth := int(winRect[2] - winRect[0])
	winHeight := int(winRect[3] - winRect[1])

	var clientRect [4]int32
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&clientRect[0])))
	clientWidth := int(clientRect[2])
	clientHeight := int(clientRect[3])

	var pt [2]int32
	procClientToScreen.Call(hwnd, uintptr(unsafe.Pointer(&pt[0])))
	clientOffsetX := int(pt[0]) - int(winRect[0])
	clientOffsetY := int(pt[1]) - int(winRect[1])

	scaleX := 1.0
	scaleY := 1.0
	if viewportWidth > 0 {
		scaleX = float64(clientWidth) / float64(viewportWidth)
	}
	if viewportHeight > 0 {
		scaleY = float64(clientHeight) / float64(viewportHeight)
	}

	imgXInWindow := clientOffsetX + int(float64(imgLeft)*scaleX)
	imgYInWindow := clientOffsetY + int(float64(imgTop)*scaleY)
	imgWInWindow := int(float64(imgWidth) * scaleX)
	imgHInWindow := int(float64(imgHeight) * scaleY)

	cropLeft := imgXInWindow
	cropTop := imgYInWindow
	cropRight := winWidth - imgXInWindow - imgWInWindow
	cropBottom := winHeight - imgYInWindow - imgHInWindow
	if cropLeft < 0 {
		cropLeft = 0
	}
	if cropTop < 0 {
		cropTop = 0
	}
	if cropRight < 0 {
		cropRight = 0
	}
	if cropBottom < 0 {
		cropBottom = 0
	}

	applog.LogInfof(s.ctx, "Magpie 裁剪: left=%d, top=%d, right=%d, bottom=%d (窗口=%dx%d, 图片=%dx%d@(%d,%d))",
		cropLeft, cropTop, cropRight, cropBottom, winWidth, winHeight, imgWInWindow, imgHInWindow, imgXInWindow, imgYInWindow)

	if err := s.configureMagpieCroppingForImage(cropLeft, cropTop, cropRight, cropBottom); err != nil {
		applog.LogWarningf(s.ctx, "配置 Magpie 裁剪失败: %v", err)
	}

	procSetForegroundWindow.Call(hwnd)
	time.Sleep(300 * time.Millisecond)

	hotkeyStr := s.config.MagpieHotkey
	if hotkeyStr == "" {
		hotkeyStr = "Win+Shift+A"
	}
	utils.SendHotkey(hotkeyStr)

	applog.LogInfof(s.ctx, "已发送 Magpie 缩放快捷键")
	return nil
}

// configureMagpieCroppingForImage 配置 Magpie 裁剪参数并重启 Magpie
func (s *ImageService) configureMagpieCroppingForImage(left, top, right, bottom int) error {
	configPath := s.config.MagpieConfigPath
	if configPath == "" {
		homeDir := os.Getenv("USERPROFILE")
		configPath = filepath.Join(homeDir, "AppData", "Local", "Magpie", "config", "v4", "config.json")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	profiles, ok := config["profiles"].([]interface{})
	if !ok || len(profiles) == 0 {
		return fmt.Errorf("配置文件中没有 profiles 数组")
	}

	profile, ok := profiles[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("profiles[0] 不是 map")
	}

	profile["croppingEnabled"] = true
	profile["cropping"] = map[string]interface{}{
		"left":   left,
		"top":    top,
		"right":  right,
		"bottom": bottom,
	}

	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return err
	}

	return s.restartMagpieForImage()
}

// restartMagpieForImage 重启 Magpie 进程
func (s *ImageService) restartMagpieForImage() error {
	applog.LogInfof(s.ctx, "重启 Magpie...")
	killCmd := exec.Command("taskkill", "/F", "/IM", "Magpie.exe")
	_ = killCmd.Run()
	time.Sleep(1 * time.Second)

	cmd := exec.Command(s.config.MagpiePath, "-t")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000,
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	applog.LogInfof(s.ctx, "Magpie 已重启")
	return nil
}
