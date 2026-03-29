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
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
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

	// 设置全局 ImageService 实例
	SetGlobalImageService(s)
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

// DeleteImageBackup 删除 ImageBackup 记录
func (s *ImageService) DeleteImageBackup(url string) error {
	query := `DELETE FROM image_backups WHERE url = ?`
	_, err := s.db.ExecContext(s.ctx, query, url)
	return err
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
		fileName := fmt.Sprintf(`%s\%s%s`, path, uuid.New().String(), ext)
		err = DownloadImage(imageBackup.Url, fileName)
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
		fileName := fmt.Sprintf(`%s\%s.%s`, path, uuid.New().String(), ext)
		err = DownloadImage(imageBackup.Url, fileName)
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
		CreatedAt:   time.Now(),
	})

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
		ImageType:   0,
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

// 全局 ImageService 实例，用于处理下载队列
var globalImageService *ImageService

// SetGlobalImageService 设置全局 ImageService 实例
func SetGlobalImageService(service *ImageService) {
	globalImageService = service
}

// DownloadImage 将下载任务加入队列并等待完成
func DownloadImage(imageUrl, fileName string) error {
	// 如果全局 ImageService 未初始化，使用同步下载
	if globalImageService == nil {
		// 创建临时 ImageService 进行同步下载
		tempService := &ImageService{}
		fmt.Printf("GetImageBackupByUrl 021 url: %s\n", imageUrl)
		err := tempService.downloadImageInternal(imageUrl, fileName, 30*time.Second)
		fmt.Printf("GetImageBackupByUrl 031 url: %s\n", imageUrl)
		return err
	}

	// 创建下载任务
	task := downloadTask{
		imageUrl:   imageUrl,
		fileName:   fileName,
		result:     make(chan error, 1),
		retryCount: 0, // 初始重试次数为 0
	}

	// 将任务加入队列
	globalImageService.downloadQueue <- task

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
			errChan <- s.downloadImageInternal(task.imageUrl, task.fileName, timeout)
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
func (s *ImageService) downloadImageInternal(imageUrl, fileName string, timeout time.Duration) error {
	fmt.Printf("GetImageBackupByUrl 022 url: %s\n", imageUrl)
	url := imageUrl
	if strings.HasPrefix(url, "//gyutto.com") {
		url = strings.ReplaceAll(url, "//gyutto.com", "https://image.gyutto.com")
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
	req, err := http.NewRequest("GET", url, nil)
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

	if _, err := io.Copy(destFile, resp.Body); err != nil {
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
