package service

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"lunabox/internal/appconf"
	"lunabox/internal/models"
	"lunabox/internal/utils"
	"strings"
	"unsafe"

	"lunabox/internal/applog"

	"image"
	"image/jpeg"
	"os"

	"github.com/go-vgo/robotgo"

	"github.com/google/uuid"
)

var (
	procGetDIBits = gdi32.NewProc("GetDIBits")
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

	return
}

// 修正后的纯Windows API截图实现
func (s *ImageService) TakeScreenshot(gameId string) {
	if gameId == "" {
		applog.LogErrorf(s.ctx, "Game ID is empty")
		return
	}
	applog.LogInfof(s.ctx, "Taking screenshot using Windows API")

	// 获取前台窗口句柄
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

	if width <= 0 || height <= 0 {
		applog.LogErrorf(s.ctx, "Invalid window dimensions: %dx%d", width, height)
		return
	}

	// 获取屏幕DC
	screenDC, _, _ := procGetDC.Call(0)
	if screenDC == 0 {
		applog.LogErrorf(s.ctx, "Failed to get screen DC")
		return
	}
	defer procReleaseDC.Call(0, screenDC)

	// 创建内存DC
	memDC, _, _ := procCreateCompatibleDC.Call(screenDC)
	if memDC == 0 {
		applog.LogErrorf(s.ctx, "Failed to create memory DC")
		return
	}
	defer procDeleteDC.Call(memDC)

	// 创建位图
	hBitmap, _, _ := procCreateCompatibleBitmap.Call(screenDC, uintptr(width), uintptr(height))
	if hBitmap == 0 {
		applog.LogErrorf(s.ctx, "Failed to create bitmap")
		return
	}
	defer procDeleteObject.Call(hBitmap)

	// 选择位图到内存DC
	oldObj, _, _ := procSelectObject.Call(memDC, hBitmap)
	defer procSelectObject.Call(memDC, oldObj)

	// 复制屏幕内容到位图
	success, _, _ := procBitBlt.Call(
		memDC, 0, 0, uintptr(width), uintptr(height),
		screenDC, uintptr(rect.Left), uintptr(rect.Top), SRCCOPY,
	)

	if success == 0 {
		applog.LogErrorf(s.ctx, "Failed to copy screen to bitmap")
		return
	}

	// 将位图数据提取出来并保存为文件
	name := uuid.New().String()
	fileName, err := s.saveBitmapToFile(hBitmap, width, height, name)
	// fileName, err := s.saveAsBMPDirectly(hBitmap, width, height, `C:\temp\projects\10.bmp`)
	if err != nil {
		applog.LogErrorf(s.ctx, "Failed to save bitmap to file: %v", err)
		return
	}
	applog.LogInfof(s.ctx, "Screenshot saved as %s.jpg (%dx%d)", name, width, height)
	err = s.CreateOrUpdateImageBackup(models.ImageBackup{
		Url:         fileName,
		LocalPath:   fileName,
		SubjectId:   gameId,
		SubjectType: 0,
		ImageType:   3,
	})
	if err != nil {
		applog.LogErrorf(s.ctx, "Failed to create or update image backup: %v", err)
		return
	}
}

func (s *ImageService) saveBitmapToFile(hBitmap uintptr, width, height int, name string) (string, error) {
	dataDir, err := utils.GetDataDir()
	if err != nil {
		return "", err

	}
	path := fmt.Sprintf(`%s\%s`, dataDir, "images")
	// path = `C:\temp\projects`
	_, err = os.Stat(path)
	if err != nil {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return "", err
		}
	}

	fileName := fmt.Sprintf(`%s\%s.jpg`, path, name)

	// 获取屏幕设备上下文
	screenDC, _, _ := procGetDC.Call(0)
	if screenDC == 0 {
		applog.LogErrorf(s.ctx, "Failed to get screen DC")
		return "", fmt.Errorf("failed to get screen DC")
	}
	defer procReleaseDC.Call(0, screenDC)

	// 检查位图句柄是否有效
	if hBitmap == 0 {
		applog.LogErrorf(s.ctx, "Invalid bitmap handle: %d", hBitmap)
		return "", fmt.Errorf("invalid bitmap handle")
	}

	// 定义 BITMAPINFO 结构
	type BITMAPINFOHEADER struct {
		BiSize           uint32
		BiWidth          int32
		BiHeight         int32
		BiPlanes         uint16
		BiBitCount       uint16
		BiCompression    uint32
		BiSizeImage      uint32
		BiXPelsPerMeter  int32
		BiYPelsPerimeter int32
		BiClrUsed        uint32
		BiClrImportant   uint32
	}

	bmi := struct {
		Header BITMAPINFOHEADER
	}{
		Header: BITMAPINFOHEADER{
			BiSize:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
			BiWidth:       int32(width),
			BiHeight:      int32(-height), // 负值表示自上而下
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: 0, // BI_RGB
		},
	}

	// 分配像素缓冲区 (BGRA格式)
	bufferSize := width * height * 4
	pixels := make([]byte, bufferSize)

	// 使用 GetDIBits 从 screenDC 获取位图数据
	ret, _, err := procGetDIBits.Call(
		screenDC,                            // hdc
		hBitmap,                             // hbmp
		0,                                   // uStartScan
		uintptr(height),                     // cScanLines
		uintptr(unsafe.Pointer(&pixels[0])), // lpvBits
		uintptr(unsafe.Pointer(&bmi)),       // lpbi
		0,                                   // uUsage (DIB_RGB_COLORS)
	)

	if ret == 0 {
		applog.LogErrorf(s.ctx, "Failed to get DIB bits, error: %v", err)
		applog.LogErrorf(s.ctx, "hBitmap: %d, width: %d, height: %d", hBitmap, width, height)
		applog.LogErrorf(s.ctx, "screenDC: %d", screenDC)
		return s.saveAsBMPDirectly(hBitmap, width, height, fmt.Sprintf("%s\\%s.bmp", path, name))
	}

	// 转换 BGRA 到 RGBA
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcIdx := (y*width + x) * 4
			dstIdx := y*img.Stride + x*4

			// BGRA -> RGBA
			img.Pix[dstIdx+0] = pixels[srcIdx+2] // R
			img.Pix[dstIdx+1] = pixels[srcIdx+1] // G
			img.Pix[dstIdx+2] = pixels[srcIdx+0] // B
			img.Pix[dstIdx+3] = pixels[srcIdx+3] // A
		}
	}

	// 保存为 JPEG
	file, err := os.Create(fileName)
	if err != nil {
		applog.LogErrorf(s.ctx, "Failed to create JPEG file: %v", err)
		return "", fmt.Errorf("failed to create JPEG file")
	}
	defer file.Close()

	options := jpeg.Options{Quality: 90}
	err = jpeg.Encode(file, img, &options)
	if err != nil {
		applog.LogErrorf(s.ctx, "Failed to encode JPEG: %v", err)
		return "", fmt.Errorf("failed to encode JPEG")
	}

	applog.LogInfof(s.ctx, "Screenshot saved as %s (%dx%d)", fileName, width, height)
	return fileName, nil
}

// saveAsBMPDirectly 实现BMP格式保存
func (s *ImageService) saveAsBMPDirectly(hBitmap uintptr, width, height int, filename string) (string, error) {
	applog.LogInfof(s.ctx, "Saving bitmap as BMP format: %s", filename)

	// 获取设备上下文
	screenDC, _, _ := procGetDC.Call(0)
	if screenDC == 0 {
		applog.LogErrorf(s.ctx, "Failed to get screen DC for BMP saving")
		return "", fmt.Errorf("failed to get screen DC")
	}
	defer procReleaseDC.Call(0, screenDC)

	// 创建BITMAPINFO结构用于获取位图数据
	type BITMAPINFOHEADER struct {
		BiSize          uint32
		BiWidth         int32
		BiHeight        int32
		BiPlanes        uint16
		BiBitCount      uint16
		BiCompression   uint32
		BiSizeImage     uint32
		BiXPelsPerMeter int32
		BiYPelsPerMeter int32
		BiClrUsed       uint32
		BiClrImportant  uint32
	}

	bmi := struct {
		Header BITMAPINFOHEADER
	}{
		Header: BITMAPINFOHEADER{
			BiSize:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
			BiWidth:       int32(width),
			BiHeight:      int32(-height), // 负值表示自上而下
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: 0, // BI_RGB
		},
	}

	// 分配像素缓冲区
	bufferSize := width * height * 4
	pixels := make([]byte, bufferSize)

	// 获取位图数据
	ret, _, _ := procGetDIBits.Call(
		screenDC,
		hBitmap,
		0,
		uintptr(height),
		uintptr(unsafe.Pointer(&pixels[0])),
		uintptr(unsafe.Pointer(&bmi)),
		0, // DIB_RGB_COLORS
	)

	if ret == 0 {
		applog.LogErrorf(s.ctx, "Failed to get DIB bits for BMP saving")
		return "", fmt.Errorf("failed to get DIB bits for BMP saving")
	}

	// 创建BMP文件
	file, err := os.Create(filename)
	if err != nil {
		applog.LogErrorf(s.ctx, "Failed to create BMP file %s: %v", filename, err)
		return "", fmt.Errorf("failed to create BMP file")
	}
	defer file.Close()

	// 计算每行的实际字节数（行对齐）
	rowSize := ((width*32 + 31) / 32) * 4 // 32位位图，每像素4字节
	imageSize := rowSize * height         // 总图像数据大小

	// 写入BMP文件头 (14 bytes)
	fileHeader := make([]byte, 14)
	fileHeader[0] = 'B'
	fileHeader[1] = 'M'
	fileSize := uint32(14 + 40 + imageSize) // 文件头 + 信息头 + 图像数据
	binary.LittleEndian.PutUint32(fileHeader[2:6], fileSize)
	binary.LittleEndian.PutUint32(fileHeader[10:14], 54) // 数据偏移量
	file.Write(fileHeader)

	// 写入BMP信息头 (40 bytes)
	infoHeader := make([]byte, 40)
	binary.LittleEndian.PutUint32(infoHeader[0:4], 40)
	binary.LittleEndian.PutUint32(infoHeader[4:8], uint32(width))
	binary.LittleEndian.PutUint32(infoHeader[8:12], uint32(height))
	binary.LittleEndian.PutUint16(infoHeader[12:14], 1)
	binary.LittleEndian.PutUint16(infoHeader[14:16], 32)
	binary.LittleEndian.PutUint32(infoHeader[16:20], 0) // BI_RGB
	binary.LittleEndian.PutUint32(infoHeader[20:24], uint32(imageSize))
	file.Write(infoHeader)

	// 写入像素数据（BGRA格式，自下而上）
	fileBuffer := make([]byte, imageSize)
	padding := rowSize - width*4 // 每行填充字节数

	for y := 0; y < height; y++ {
		// 从最后一行开始写入（自下而上）
		srcOffset := (height - 1 - y) * width * 4
		dstOffset := y * rowSize

		// 复制一行数据
		copy(fileBuffer[dstOffset:dstOffset+width*4], pixels[srcOffset:srcOffset+width*4])

		// 填充行末对齐字节
		if padding > 0 {
			for i := 0; i < padding; i++ {
				fileBuffer[dstOffset+width*4+i] = 0
			}
		}
	}

	// 写入文件
	file.Write(fileBuffer)

	applog.LogInfof(s.ctx, "BMP screenshot saved as %s (%dx%d)", filename, width, height)
	return filename, nil
}
