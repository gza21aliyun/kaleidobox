package service

import (
	"archive/zip"
	"context"
	"crypto/md5"
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"lunabox/internal/appconf"
	"lunabox/internal/applog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
)

type DownloadedFilesService struct {
	ctx    context.Context
	db     *sql.DB
	config *appconf.AppConfig
}

func NewDownloadedFilesService() *DownloadedFilesService {
	return &DownloadedFilesService{}
}

func (s *DownloadedFilesService) Init(ctx context.Context, db *sql.DB, config *appconf.AppConfig) {
	s.ctx = ctx
	s.db = db
	s.config = config
}

type DownloadedFile struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Path           string   `json:"path"`
	IsFolder       bool     `json:"is_folder"`
	Size           int64    `json:"size"`
	IsDownloading  bool     `json:"is_downloading"` // 是否有未下载完的临时文件
	IsExtracted    bool     `json:"is_extracted"`
	IsInstalled    bool     `json:"is_installed"`
	IsImported     bool     `json:"is_imported"`
	ContainsISO    bool     `json:"contains_iso"`
	ISOCount       int      `json:"iso_count"`
	ISOFilePath    string   `json:"iso_file_path,omitempty"`
	InnerItems     []string `json:"inner_items"`
	HasNumericName bool     `json:"has_numeric_name"`
	LongestZipName string   `json:"longest_zip_name"`
}

var compressedExtensions = map[string]bool{
	".zip": true,
	".rar": true,
	".7z":  true,
	".tar": true,
	".gz":  true,
	".bz2": true,
	".xz":  true,
	".z":   true,
	".lz":  true,
}

var imageExtensions = map[string]bool{
	".iso": true,
	".mdf": true,
	".img": true,
	".bin": true,
	".cue": true,
}

func (s *DownloadedFilesService) ListDownloadedFiles() ([]DownloadedFile, error) {
	downloadFolder := s.config.GameDownloadFolder
	if downloadFolder == "" {
		return nil, fmt.Errorf("游戏下载文件夹未配置")
	}

	entries, err := os.ReadDir(downloadFolder)
	if err != nil {
		if os.IsNotExist(err) {
			return []DownloadedFile{}, nil
		}
		return nil, err
	}

	var items []DownloadedFile
	for _, entry := range entries {
		itemPath := filepath.Join(downloadFolder, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		isFolder := entry.IsDir()
		name := entry.Name()
		ext := filepath.Ext(name)
		isCompressed := compressedExtensions[strings.ToLower(ext)]

		if !isFolder && !isCompressed {
			continue
		}

		item := DownloadedFile{
			ID:             uuid.New().String(),
			Name:           name,
			Path:           itemPath,
			IsFolder:       isFolder,
			Size:           info.Size(),
			IsDownloading:  s.checkIsDownloading(itemPath, name, isFolder),
			IsExtracted:    s.checkIsExtracted(itemPath, name, isFolder),
			IsInstalled:    s.checkIsInstalled(itemPath),
			IsImported:     s.checkIsImported(name),
			HasNumericName: s.isNumericName(name),
		}

		if item.HasNumericName && isFolder {
			item.LongestZipName = s.findLongestZipName(itemPath)
		}

		if isFolder {
			item.InnerItems = s.getInnerItems(itemPath)
			isoCount, isoPath := s.countISOFiles(itemPath)
			item.ISOCount = isoCount
			item.ContainsISO = isoCount > 0
			if isoCount == 1 {
				item.ISOFilePath = isoPath
			}
		} else {
			item.InnerItems = s.getArchiveInnerItems(itemPath)
		}

		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})

	return items, nil
}

func (s *DownloadedFilesService) checkIsExtracted(itemPath, name string, isFolder bool) bool {
	downloadFolder := s.config.GameDownloadFolder
	if isFolder {
		baseName := strings.TrimSuffix(name, filepath.Ext(name))
		for ext := range compressedExtensions {
			zipPath := filepath.Join(downloadFolder, baseName+ext)
			if _, err := os.Stat(zipPath); err == nil {
				return true
			}
		}
		return s.hasExtractedContent(itemPath)
	} else {
		baseName := strings.TrimSuffix(name, filepath.Ext(name))
		folderPath := filepath.Join(downloadFolder, baseName)
		if _, err := os.Stat(folderPath); err == nil {
			return true
		}
		return false
	}
}

func (s *DownloadedFilesService) hasExtractedContent(folderPath string) bool {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return false
	}
	return len(entries) > 0
}

func (s *DownloadedFilesService) checkIsInstalled(itemPath string) bool {
	installFolder := s.config.GameInstallFolder
	if installFolder == "" {
		return false
	}

	// 获取原始文件夹名作为游戏名
	baseName := filepath.Base(itemPath)
	gameName := s.ExtractGameNameFromDLSite(baseName)

	// 检查以游戏名命名的文件夹
	gameNamePath := filepath.Join(installFolder, gameName)
	if _, err := os.Stat(gameNamePath); err == nil {
		return true
	}

	// 检查以 md5 命名的文件夹
	md5Name := s.getGameNameMD5(gameName)
	md5Path := filepath.Join(installFolder, md5Name)
	if _, err := os.Stat(md5Path); err == nil {
		return true
	}

	return false
}

// getGameNameMD5 生成游戏名的 md5 哈希
func (s *DownloadedFilesService) getGameNameMD5(gameName string) string {
	data := []byte(gameName)
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}

func (s *DownloadedFilesService) checkIsImported(name string) bool {
	return false
}

// checkIsDownloading 检查是否有未下载完的临时文件（QBittorrent 的 .!qB 文件，uTorrent 的 .!ut 文件）
func (s *DownloadedFilesService) checkIsDownloading(itemPath, name string, isFolder bool) bool {
	if isFolder {
		// 对于文件夹，检查是否有对应的临时文件
		// .!qB - QBittorrent
		tempFilePath := filepath.Join(filepath.Dir(itemPath), name+".!qB")
		if _, err := os.Stat(tempFilePath); err == nil {
			return true
		}
		// .!ut - uTorrent
		tempFilePath = filepath.Join(filepath.Dir(itemPath), name+".!ut")
		if _, err := os.Stat(tempFilePath); err == nil {
			return true
		}
		// 也检查文件夹内部是否有临时文件
		entries, err := os.ReadDir(itemPath)
		if err != nil {
			return false
		}
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".!qB") || strings.HasSuffix(entry.Name(), ".!ut") {
				return true
			}
		}
		return false
	} else {
		// 对于压缩包，检查是否有临时文件
		baseName := strings.TrimSuffix(name, filepath.Ext(name))
		ext := filepath.Ext(name)
		// .!qB - QBittorrent
		tempFilePath := filepath.Join(filepath.Dir(itemPath), baseName+ext+".!qB")
		if _, err := os.Stat(tempFilePath); err == nil {
			return true
		}
		// .!ut - uTorrent
		tempFilePath = filepath.Join(filepath.Dir(itemPath), baseName+ext+".!ut")
		if _, err := os.Stat(tempFilePath); err == nil {
			return true
		}
		return false
	}
}

func (s *DownloadedFilesService) isNumericName(name string) bool {
	baseName := strings.TrimSuffix(name, filepath.Ext(name))
	re := regexp.MustCompile(`^[\d]+$`)
	return re.MatchString(baseName)
}

func (s *DownloadedFilesService) findLongestZipName(folderPath string) string {
	var longestName string
	maxLen := 0

	err := filepath.WalkDir(folderPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if compressedExtensions[ext] {
				nameWithoutExt := strings.TrimSuffix(d.Name(), ext)
				if len(nameWithoutExt) > maxLen {
					maxLen = len(nameWithoutExt)
					longestName = nameWithoutExt
				}
			}
		}
		return nil
	})

	if err != nil {
		return ""
	}
	return longestName
}

func (s *DownloadedFilesService) getInnerItems(folderPath string) []string {
	items := []string{}
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return items
	}

	for i, entry := range entries {
		if i >= 5 {
			break
		}
		items = append(items, entry.Name())
	}
	return items
}

func (s *DownloadedFilesService) getArchiveInnerItems(archivePath string) []string {
	items := []string{}
	ext := strings.ToLower(filepath.Ext(archivePath))

	if ext == ".zip" {
		r, err := zip.OpenReader(archivePath)
		if err != nil {
			return items
		}
		defer r.Close()

		for i, f := range r.File {
			if i >= 5 {
				break
			}
			items = append(items, f.Name)
		}
	}

	return items
}

func (s *DownloadedFilesService) countISOFiles(folderPath string) (int, string) {
	count := 0
	var isoPath string

	err := filepath.WalkDir(folderPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if imageExtensions[ext] {
				count++
				if isoPath == "" {
					isoPath = path
				}
			}
		}
		return nil
	})

	if err != nil {
		return 0, ""
	}
	return count, isoPath
}

func (s *DownloadedFilesService) ExtractItem(itemPath string) error {
	ext := strings.ToLower(filepath.Ext(itemPath))
	if !compressedExtensions[ext] {
		return fmt.Errorf("不支持的压缩格式: %s", ext)
	}

	downloadFolder := filepath.Dir(itemPath)
	baseName := strings.TrimSuffix(filepath.Base(itemPath), ext)
	targetFolder := filepath.Join(downloadFolder, baseName)

	if err := os.MkdirAll(targetFolder, 0755); err != nil {
		return err
	}

	if ext == ".zip" {
		return s.extractZip(itemPath, targetFolder)
	}

	return s.extractWith7z(itemPath, targetFolder)
}

func (s *DownloadedFilesService) extractZip(zipPath, targetFolder string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(targetFolder, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func (s *DownloadedFilesService) extractWith7z(archivePath, targetFolder string) error {
	cmd := exec.Command("7z", "x", archivePath, "-o"+targetFolder, "-y")
	_, err := cmd.CombinedOutput()
	return err
}

func (s *DownloadedFilesService) ExtractFolder(folderPath string) error {
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if compressedExtensions[ext] {
			filePath := filepath.Join(folderPath, file.Name())
			if err := s.ExtractItem(filePath); err != nil {
				applog.LogErrorf(s.ctx, "Failed to extract %s: %v", filePath, err)
			}
		}
	}
	return nil
}

func (s *DownloadedFilesService) MountISO(isoPath string) error {
	cmd := exec.Command("powershell", "-Command", fmt.Sprintf(`Mount-DiskImage -ImagePath "%s"`, isoPath))
	_, err := cmd.CombinedOutput()
	return err
}

func (s *DownloadedFilesService) InstallGame(itemPath, installMethod string) (string, error) {
	installFolder := s.config.GameInstallFolder
	if installFolder == "" {
		return "", fmt.Errorf("游戏安装文件夹未配置")
	}

	var gameName string
	baseName := filepath.Base(itemPath)

	// 使用 ExtractGameNameFromDLSite 提取游戏名
	extractedGameName := s.ExtractGameNameFromDLSite(baseName)

	if installMethod == "md5" {
		gameName = s.getGameNameMD5(extractedGameName)
	} else {
		gameName = extractedGameName
	}

	targetPath := filepath.Join(installFolder, gameName)

	if _, err := os.Stat(targetPath); err == nil {
		return "", fmt.Errorf("目标路径已存在: %s", targetPath)
	}

	files, err := os.ReadDir(itemPath)
	if err != nil {
		return "", err
	}

	hasISO := false
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if imageExtensions[ext] {
			hasISO = true
			break
		}
	}

	if hasISO {
		isoFiles := []string{}
		for _, file := range files {
			ext := strings.ToLower(filepath.Ext(file.Name()))
			if imageExtensions[ext] {
				isoFiles = append(isoFiles, filepath.Join(itemPath, file.Name()))
			}
		}

		if len(isoFiles) == 1 {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return "", err
			}
			return targetPath, s.ExtractISO(isoFiles[0], targetPath)
		}
		return "", fmt.Errorf("包含多个镜像文件，无法自动安装")
	}

	if err := copyDirectory(itemPath, targetPath); err != nil {
		return "", err
	}

	return targetPath, nil
}

func (s *DownloadedFilesService) ExtractISO(isoPath, targetPath string) error {
	cmd := exec.Command("7z", "x", isoPath, "-o"+targetPath, "-y")
	_, err := cmd.CombinedOutput()
	return err
}

func (s *DownloadedFilesService) OpenFolder(itemPath string) error {
	cmd := exec.Command("explorer.exe", itemPath)
	return cmd.Start()
}

func (s *DownloadedFilesService) DeleteItem(itemPath string) error {
	return os.RemoveAll(itemPath)
}

func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			return err
		}

		return os.Chmod(targetPath, info.Mode())
	})
}

// ExtractGameNameFromDLSite 从 DLSite 风格的文件名中提取游戏名称
// 例如: "[260529][1358608][Whirlpool] Relirium -レリリウム- 遗迹と出逢いと冒険と メモリアル特装版 パッケージ版 (mdf+mds)"
// 返回: "Relirium -レリリウム- 遗迹と出逢いと冒険と メモリアル特装版 パッケージ版"
func (s *DownloadedFilesService) ExtractGameNameFromDLSite(filename string) string {
	// DLSite 格式: (类型) [日期][编号][作者] 游戏名 (文件类型)
	// 或: [日期][编号][作者] 游戏名 (文件类型)

	// 匹配模式: 去除开头的元数据部分，提取游戏名
	// 元数据格式: (xxx) [日期][编号][作者] 或 [日期][编号][作者]
	re := regexp.MustCompile(`^(?:\([^)]+\)\s*)?\[[^\]]+\]\[[^\]]+\]\[[^\]]+\]\s*(.+?)(?:\s*\([^\)]+\))?$`)

	matches := re.FindStringSubmatch(filename)
	if len(matches) < 2 {
		return filename
	}

	return strings.TrimSpace(matches[1])
}
