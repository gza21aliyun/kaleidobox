package service

import (
	"archive/zip"
	"context"
	"crypto/md5"
	"database/sql"
	"fmt"
	"io"
	"lunabox/internal/appconf"
	"lunabox/internal/applog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

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
	ID                string   `json:"id"`
	Name              string   `json:"name"` //单元标题，不能数字，不包含文件后缀
	Path              string   `json:"path"` //全路径，文件的话包含后缀
	BaseName          string   `json:"base_name,omitempty"`
	ExtractedGamePath string   `json:"extracted_game_path,omitempty"` // 解压后的游戏文件夹路径（主路径）
	ExtractedPaths    []string `json:"extracted_paths,omitempty"`     // 所有解压后的文件夹路径
	// IsFolder          bool     `json:"is_folder"`
	Type          int   `json:"type"` // 0:是文件夹没任何压缩包不需解压; 1: 单元文件夹里包含一个或多个压缩包; 2: 游戏下载文件夹下单个压缩包或多了同名文件夹（解压后）结构;
	Size          int64 `json:"size"`
	IsDownloading bool  `json:"is_downloading"` // 是否有未下载完的临时文件
	IsExtracted   bool  `json:"is_extracted"`
	IsInstalled   bool  `json:"is_installed"`
	IsImported    bool  `json:"is_imported"`
	// ContainsISO       bool     `json:"contains_iso"`
	ISOItems []string `json:"iso_items"`
	// ISOCount          int      `json:"iso_count"`
	// ISOFilePath       string   `json:"iso_file_path,omitempty"`
	InnerItems     []string `json:"inner_items"`
	HasNumericName bool     `json:"has_numeric_name"`
	GameName       string   `json:"game_name"` //从单元标题再继续去除描述性前缀和后缀
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
	// ".img": true,
	// ".bin": true,
	// ".cue": true,
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
	var folderItems []DownloadedFile
	var archiveItems []DownloadedFile

	// 第一遍：收集所有条目，分离文件夹和压缩包
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
		item := s.CreateDownloadedFile(itemPath, name, isFolder, info.Size())
		if isFolder {
			folderItems = append(folderItems, item)
		} else {
			archiveItems = append(archiveItems, item)
		}
	}

	// 建立文件夹名称的集合
	folderNames := make(map[string]bool)
	for _, folder := range folderItems {
		folderNames[folder.BaseName] = true
	}

	// 建立压缩包的基础名称映射
	archiveBaseNames := make(map[string]DownloadedFile)
	for _, archive := range archiveItems {
		baseName := strings.TrimSuffix(filepath.Base(archive.Path), filepath.Ext(archive.Path))
		archiveBaseNames[baseName] = archive
	}

	// 记录哪些压缩包已经被显示（因为对应的文件夹有内容）
	displayedArchives := make(map[string]bool)

	// 合并逻辑：文件夹有内容时显示文件夹，没有内容但有同名压缩包时显示压缩包
	for _, folder := range folderItems {
		baseName := strings.TrimSuffix(filepath.Base(folder.Path), filepath.Ext(folder.Path))

		// 检查是否有同名压缩包
		if _, exists := archiveBaseNames[baseName]; exists {
			// 存在同名压缩包
			folder.Type = 2
			folder.IsDownloading = s.checkIsDownloading(folder.Path, 2)
			folder.IsExtracted = s.checkIsExtracted(folder.Path, folder.BaseName, 2)

			// 如果文件夹没有内容，不显示文件夹，后续会显示压缩包
		} else {
			// 没有同名压缩包，: 单个文件夹结构
			judge, hasArchieve, hasInnerFolder := s.hasExtractedContent(folder.Path)
			if false {
				fmt.Printf("hasExtractedContent,name: %s, judge: %v, hasArchieve: %v, hasInnerFolder: %v\n", folder.Name, judge, hasArchieve, hasInnerFolder)
			}
			if hasArchieve {
				// 文件夹有内容，显示文件夹，标记为已解压
				// type=1: 文件夹包含多个压缩包
				folder.Type = 1
				folder.IsDownloading = s.checkIsDownloading(folder.Path, 1)
				folder.IsExtracted = s.checkIsExtracted(folder.Path, folder.BaseName, 1)
			} else {
				folder.Type = 0
				folder.IsDownloading = s.checkIsDownloading(folder.Path, 0)
				folder.IsExtracted = true
			}

		}
		if folder.IsExtracted {
			extractedGamePath, extractedPaths := s.getExtractedPaths(folder.Path, folder.BaseName, folder.Type)
			// fmt.Printf("extractedPath,name: %s, eGamePath: %s, ePaths: %d, path:%s\n", folder.Name, extractedGamePath, len(extractedPaths), folder.Path)
			folder.ExtractedGamePath = extractedGamePath
			folder.ExtractedPaths = extractedPaths
		}

		items = append(items, folder)
		displayedArchives[baseName] = true
	}

	// 添加没有被显示的压缩包
	for _, archive := range archiveItems {
		baseName := strings.TrimSuffix(archive.BaseName, filepath.Ext(archive.BaseName))
		if !displayedArchives[baseName] {
			archive.Type = 2
			archive.IsDownloading = s.checkIsDownloading(archive.Path, 2)
			archive.IsExtracted = s.checkIsExtracted(archive.Path, archive.BaseName, 2)
			items = append(items, archive)
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})

	return items, nil
}

func (s *DownloadedFilesService) RefreshDownloadedFile(file DownloadedFile) (DownloadedFile, error) {
	// info, err := os.Stat(itemPath)
	// if err != nil {
	// 	return DownloadedFile{}, err
	// }
	itemPath := file.Path
	ext := filepath.Ext(itemPath)
	path := strings.TrimSuffix(itemPath, ext)
	archive := s.CreateDownloadedFile(path, filepath.Base(path), true, 0)
	if file.Type == 2 {
		if archive.InnerItems != nil && len(archive.InnerItems) > 0 {
			archive.ExtractedGamePath = path
			archive.ExtractedPaths = []string{path}
			return archive, nil
		}
	} else if file.Type == 1 {
		extractedGamePath, extractedPaths := s.getExtractedPaths(path, filepath.Base(path), 1)
		archive.ExtractedGamePath = extractedGamePath
		archive.ExtractedPaths = extractedPaths
	}

	return archive, nil
}

/**
 * itemPath 全路径
 * name 包括后缀的文件名或文件夹名
 */
func (s *DownloadedFilesService) CreateDownloadedFile(itemPath, name string, isFolder bool, size int64) DownloadedFile {
	fileNameWithoutExt := strings.TrimSuffix(name, filepath.Ext(name))
	item := DownloadedFile{
		ID:       uuid.New().String(),
		Name:     fileNameWithoutExt,
		BaseName: name,
		Path:     itemPath,
		ISOItems: []string{},
		// IsFolder:          isFolder,
		Type: 2,
		Size: size,
		// IsDownloading:     s.checkIsDownloading(itemPath, name, itemType),
		// IsExtracted:       s.checkIsExtracted(itemPath, name, itemType),
		IsInstalled:    s.checkIsInstalled(itemPath),
		IsImported:     s.checkIsImported(name),
		HasNumericName: s.isNumericName(name),
		GameName:       s.ExtractGameNameFromDLSite(fileNameWithoutExt),
	}

	if isFolder {
		innerItems, isoItems := s.getInnerItems(itemPath)
		fmt.Printf("InnerItems,name: %s, path:%s, count:%v\n", item.Name, item.Path, innerItems)
		if item.HasNumericName {
			item.Name = s.JudgeGameName(innerItems)
			item.GameName = s.ExtractGameNameFromDLSite(item.Name)
			fmt.Printf("GameName: %s changed from %s\n", item.GameName, item.Name)
		}
		// isoCount, isoPath, innerItems := s.countISOFiles(itemPath)
		// isoCount := len(isoItems)
		item.InnerItems = innerItems
		item.ISOItems = isoItems
		// item.ISOCount = isoCount
		// item.ContainsISO = isoCount > 0
		// if isoCount == 1 {
		// 	item.ISOFilePath = isoItems[0]
		// }
		// type=1: 文件夹包含多个压缩包
		item.Type = 1
	} else {
		item.InnerItems = s.getArchiveInnerItems(itemPath)
		item.ISOItems = []string{}
		// type=2: 压缩包解压后多了同名文件夹结构
		item.Type = 2

	}
	return item
}

/**
 * 有检查有内容的，有检查压缩包的，todo:待改进
 */
func (s *DownloadedFilesService) checkIsExtracted(itemPath, name string, itemType int) bool {
	downloadFolder := s.config.GameDownloadFolder
	if itemType == 0 {
		// 无需解压
		return true
	} else if itemType == 1 {
		// 检查文件夹内是否有真正的解压内容（不是只有压缩包）
		has, _, _ := s.hasExtractedContent(itemPath)
		return has
	} else {
		// 对于压缩包，检查是否有同名的解压文件夹
		baseName := strings.TrimSuffix(name, filepath.Ext(name))
		folderPath := filepath.Join(downloadFolder, baseName)
		// 检查文件夹是否存在且有真正的解压内容
		if info, err := os.Stat(folderPath); err == nil && info.IsDir() {
			has, _, _ := s.hasExtractedContent(itemPath)
			return has
		}
		return false
	}
}

func (s *DownloadedFilesService) hasExtractedContent(folderPath string) (bool, bool, bool) {
	entries, _ := os.ReadDir(folderPath)
	// if err != nil {
	// 	return false
	// }

	hasSubFolder := false
	hasArchive := false

	for _, entry := range entries {
		if entry.IsDir() {
			hasSubFolder = true
			// break
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if compressedExtensions[ext] {
			hasArchive = true
		}
	}

	if hasSubFolder {
		return true, hasArchive, hasSubFolder
	}

	if hasArchive {
		return false, hasArchive, hasSubFolder
	}

	return len(entries) > 0, hasArchive, hasSubFolder
}

func (s *DownloadedFilesService) getExtractedPaths(itemPath, name string, fileType int) (string, []string) {
	downloadFolder := s.config.GameDownloadFolder
	var extractedPaths []string

	if fileType == 2 {
		// 对于压缩包，返回同名文件夹路径
		baseName := strings.TrimSuffix(name, filepath.Ext(name))
		folderPath := filepath.Join(downloadFolder, baseName)
		if info, err := os.Stat(folderPath); err == nil && info.IsDir() {
			return folderPath, []string{folderPath}
		}
		return "", nil
	}
	if fileType == 0 {
		entries, err := os.ReadDir(itemPath)
		if err != nil {
			return itemPath, nil
		}
		if len(entries) == 1 && entries[0].IsDir() {
			return filepath.Join(itemPath, entries[0].Name()), nil
		}
		return itemPath, nil
	}

	// 对于文件夹，检查里面是否有解压后的子目录
	judge, _, _ := s.hasExtractedContent(itemPath)
	if !judge {
		return "", nil
	}

	// 查找所有子文件夹（解压后的游戏目录）
	entries, err := os.ReadDir(itemPath)
	if err != nil {
		return "", nil
	}
	folders := make(map[string]bool)
	files := make(map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() {
			fullPath := filepath.Join(itemPath, entry.Name())
			folders[fullPath] = true
		} else if compressedExtensions[strings.ToLower(filepath.Ext(entry.Name()))] {
			files[filepath.Join(itemPath, entry.Name())] = true
		}
	}

	for path, _ := range files {
		baseName := filepath.Base(path)
		ext := filepath.Ext(baseName)
		folderPath := strings.TrimSuffix(filepath.Join(itemPath, baseName), ext)
		ext2 := filepath.Ext(folderPath)
		folderPath2 := strings.TrimSuffix(folderPath, ext2)
		if folders[folderPath] {
			extractedPaths = append(extractedPaths, folderPath)
		} else if ext2 != "" && folders[folderPath2] {
			extractedPaths = append(extractedPaths, folderPath2)
		}

	}

	if len(extractedPaths) == 0 {
		return "", []string{}
	}

	if len(extractedPaths) == 1 {
		return extractedPaths[0], extractedPaths
	}

	// 多个解压文件夹时，使用 JudgeGameName 判断哪个是真正的游戏文件夹
	var folderNames []string
	for _, path := range extractedPaths {
		folderNames = append(folderNames, filepath.Base(path))
	}

	gameFolderName := s.JudgeGameName(folderNames)
	if gameFolderName != "" {
		for _, path := range extractedPaths {
			if filepath.Base(path) == gameFolderName {
				entries, err := os.ReadDir(path)
				if err == nil && len(entries) == 1 && entries[0].IsDir() {
					return filepath.Join(path, entries[0].Name()), extractedPaths
				}
				return path, extractedPaths
			}
		}
	}

	// 如果 JudgeGameName 没找到，返回第一个
	return extractedPaths[0], extractedPaths
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
func (s *DownloadedFilesService) checkIsDownloading(itemPath string, itemType int) bool {
	if itemType == 1 || itemType == 0 {
		// 对于文件夹，检查是否有对应的临时文件
		// .!qB - QBittorrent
		// tempFilePath := filepath.Join(filepath.Dir(itemPath), name+".!qB")
		// if _, err := os.Stat(tempFilePath); err == nil {
		// 	return true
		// }
		// // .!ut - uTorrent
		// tempFilePath = filepath.Join(filepath.Dir(itemPath), name+".!ut")
		// if _, err := os.Stat(tempFilePath); err == nil {
		// 	return true
		// }
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
		baseName := strings.TrimSuffix(filepath.Base(itemPath), filepath.Ext(itemPath))
		ext := filepath.Ext(itemPath)
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

func (s *DownloadedFilesService) getInnerItems(folderPath string) ([]string, []string) {
	itemsMap := make(map[string]string)
	items := []string{}
	entries, err := os.ReadDir(folderPath)
	isoItems := []string{}
	if err != nil {
		return items, isoItems
	}

	for i, entry := range entries {
		if i >= 5 {
			// break
		}
		filename := entry.Name()
		filebase := strings.TrimSuffix(filename, filepath.Ext(filename))
		// itemsMap[filebase] = filename
		if (strings.Contains(filebase, "iso") || strings.Contains(filebase, "mdf")) && entry.IsDir() {
			subEntries, err := os.ReadDir(filepath.Join(folderPath, filebase))
			if err != nil {
				continue
			}
			for _, subEntry := range subEntries {
				// subFilebase := strings.TrimSuffix(subEntry.Name(), filepath.Ext(subEntry.Name()))
				ext := strings.ToLower(filepath.Ext(subEntry.Name()))
				if imageExtensions[ext] {
					// itemsMap[subFilebase] = subEntry.Name()
					fullpath := filepath.Join(folderPath, filebase, subEntry.Name())
					isoItems = append(isoItems, fullpath)
				}
			}
		}

		if !entry.IsDir() {
			ext := strings.ToLower(filepath.Ext(filename))
			if imageExtensions[ext] {
				fullpath := filepath.Join(folderPath, filename)
				isoItems = append(isoItems, fullpath)
			} else {
				itemsMap[filebase] = filename

			}
		} else {
			itemsMap[filebase] = filename
		}
	}
	for _, v := range itemsMap {
		items = append(items, v)
	}
	sort.Slice(items, func(i, j int) bool {
		return len(items[i]) < len(items[j])
	})
	return items, isoItems
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

// func (s *DownloadedFilesService) countISOFiles(folderPath string) (int, string, []string) {
// 	count := 0
// 	var isoPath string
// 	innerItemsMap := make(map[string]string)

// 	err := filepath.WalkDir(folderPath, func(path string, d fs.DirEntry, err error) error {
// 		if err != nil {
// 			return err
// 		}
// 		relativePath, _ := filepath.Rel(folderPath, path)
// 		level := strings.Count(relativePath, string(filepath.Separator))
// 		if !d.IsDir() {
// 			ext := strings.ToLower(filepath.Ext(d.Name()))
// 			if imageExtensions[ext] {
// 				count++
// 				innerItemsMap[d.Name()] = d.Name()
// 				if isoPath == "" {
// 					isoPath = path
// 				}
// 			} else if level == 0 {
// 				innerItemsMap[d.Name()] = d.Name()
// 			}
// 		}
// 		return nil
// 	})
// 	innerItems := []string{}

// 	for k := range innerItemsMap {
// 		innerItems = append(innerItems, k)
// 	}

// 	if err != nil {
// 		return 0, "", innerItems
// 	}
// 	return count, isoPath, innerItems
// }

func (s *DownloadedFilesService) ExtractItem(itemPath string) error {
	info, err := os.Stat(itemPath)
	fmt.Println("Extracting0:", itemPath)
	if err != nil {
		return err
	}

	fmt.Println("Extracting1:", itemPath)

	if info.IsDir() {
		// 如果是文件夹，解压其中的所有压缩包
		return s.ExtractArchivesInFolder(itemPath)
	}
	fmt.Println("Extracting2:", itemPath)

	// 如果是压缩包文件，直接解压
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

	err = s.extractArchive(itemPath, targetFolder)
	return err
}

// extractArchivesInFolder 解压文件夹内的所有压缩包，处理分段压缩
func (s *DownloadedFilesService) ExtractArchivesInFolder(folderPath string) error {
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}

	var archives []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if compressedExtensions[ext] {

			excludedNames := []string{
				"サウンドトラック", "soundtrack", "mp3", "wav", "flac", "cue",
			}
			shouldExclude := false
			for _, name := range excludedNames {
				if strings.Contains(strings.ToLower(file.Name()), name) {
					shouldExclude = true
					break
				}
			}
			if shouldExclude {
				continue
			}
			archives = append(archives, filepath.Join(folderPath, file.Name()))
		}
	}

	if len(archives) == 0 {
		return fmt.Errorf("文件夹内没有找到压缩包")
	}

	processedBases := make(map[string]bool)

	for _, archivePath := range archives {
		baseName := filepath.Base(archivePath)
		ext := filepath.Ext(baseName)

		baseWithoutExt := strings.TrimSuffix(baseName, ext)
		isPartArchive := false
		partBase := baseWithoutExt
		partNumber := 0

		// 使用正则表达式匹配分段压缩包
		// 匹配 .part1, .part2, .part3 等格式
		if match := regexp.MustCompile(`^(.+)\.part(\d+)$`).FindStringSubmatch(baseWithoutExt); match != nil {
			partBase = match[1]
			partNumber, _ = strconv.Atoi(match[2])
			isPartArchive = true
		} else if match := regexp.MustCompile(`^(.+)\.(\d{3})$`).FindStringSubmatch(baseWithoutExt); match != nil {
			// 匹配 .001, .002, .003 等格式
			partBase = match[1]
			partNumber, _ = strconv.Atoi(match[2])
			isPartArchive = true
		} else if match := regexp.MustCompile(`^(.+)\.(\d+)$`).FindStringSubmatch(baseWithoutExt); match != nil {
			// 匹配 .1, .2, .3 等格式（如 .rar.1, .zip.2）
			// 确保不是纯数字文件名
			if len(match[1]) > 0 {
				partBase = match[1]
				partNumber, _ = strconv.Atoi(match[2])
				isPartArchive = true
			}
		}

		if isPartArchive {
			fmt.Printf("ExtractArchivesInFolder partBase: %s, partNumber: %d, processedBases: %v\n", partBase, partNumber, processedBases)
		}

		if isPartArchive {
			// 只处理第一段（.part1, .001, .1）
			isFirstPart := partNumber == 1

			if !isFirstPart {
				// 不是第一段，跳过
				continue
			}

			if processedBases[partBase] {
				continue
			}
			processedBases[partBase] = true
		} else {
			if processedBases[baseWithoutExt] {
				continue
			}
			processedBases[baseWithoutExt] = true
		}

		targetFolder := filepath.Join(folderPath, partBase)
		if err := os.MkdirAll(targetFolder, 0755); err != nil {
			applog.LogErrorf(s.ctx, "Failed to create folder %s: %v", targetFolder, err)
			continue
		}

		if err := s.extractArchive(archivePath, targetFolder); err != nil {
			applog.LogErrorf(s.ctx, "Failed to extract %s: %v", archivePath, err)
		} else {
			applog.LogInfof(s.ctx, "Extracted %s to %s", archivePath, targetFolder)
		}
	}

	return nil
}

func (s *DownloadedFilesService) extractArchive(archivePath, targetFolder string) error {
	applog.LogInfof(s.ctx, "开始解压: %s -> %s", archivePath, targetFolder)

	// 优先使用配置的7z路径，否则使用系统默认的7z命令
	sevenZipPath := s.config.SevenZipPath
	if sevenZipPath == "" {
		sevenZipPath = "7z"
	}

	cmd := exec.Command(sevenZipPath, "x", archivePath, "-o"+targetFolder, "-y", "-mmt=on")
	output, err := cmd.CombinedOutput()
	if err != nil {
		applog.LogErrorf(s.ctx, "解压失败: %v, 输出: %s", err, string(output))
		return fmt.Errorf("解压失败: %v", err)
	}

	applog.LogInfof(s.ctx, "解压成功: %s", targetFolder)
	return nil
}

func (s *DownloadedFilesService) ExtractFolder(DownloadedFile DownloadedFile) (DownloadedFile, error) {
	folderPath := DownloadedFile.Path

	fmt.Println("Extracting11:", folderPath)
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return DownloadedFile, err
	}
	extractedPaths := []string{}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if compressedExtensions[ext] {
			filePath := filepath.Join(folderPath, file.Name())
			excludedNames := []string{
				"サウンドトラック", "soundtrack", "mp3", "wav",
			}
			shouldExclude := false
			for _, name := range excludedNames {
				if strings.Contains(strings.ToLower(file.Name()), name) {
					shouldExclude = true
					break
				}
			}
			if shouldExclude {
				continue
			}
			if err := s.ExtractItem(filePath); err != nil {
				applog.LogErrorf(s.ctx, "Failed to extract %s: %v", filePath, err)
			} else {
				extractedPaths = append(extractedPaths, filePath)
			}
		}
	}
	DownloadedFile.ExtractedPaths = extractedPaths
	return DownloadedFile, nil
}

func (s *DownloadedFilesService) MountISO(isoPath string) error {
	// 获取挂载前的盘符列表
	output, err := exec.Command("wmic", "logicaldisk", "get", "deviceid").CombinedOutput()
	if err != nil {
		return err
	}
	currentDrivesMap := make(map[string]bool)
	for _, line := range strings.Split(string(output), "\n") {
		drive := strings.TrimSpace(line)
		if drive != "" && drive != "DeviceID" {
			currentDrivesMap[drive] = true
		}
	}

	// 根据文件扩展名选择挂载方式
	ext := strings.ToLower(filepath.Ext(isoPath))

	var newMountedDrive string

	if ext == ".mdf" {
		// 检查是否存在对应的 .mds 文件
		mdsPath := strings.TrimSuffix(isoPath, filepath.Ext(isoPath)) + ".mds"
		if _, err := os.Stat(mdsPath); os.IsNotExist(err) {
			return fmt.Errorf("未找到对应的.mds文件: %s", mdsPath)
		}
		// 使用系统默认方式打开 .mds 文件（假设系统已安装虚拟光驱软件）
		cmd := exec.Command("cmd", "/c", "start", "", mdsPath)
		err := cmd.Start()
		if err != nil {
			return fmt.Errorf("打开MDF文件失败: %v", err)
		}

	} else {
		// 挂载ISO
		cmd := exec.Command("powershell", "-Command", fmt.Sprintf(`Mount-DiskImage -ImagePath "%s"`, isoPath))
		mountOutput, mountErr := cmd.CombinedOutput()
		if mountErr != nil {
			return fmt.Errorf("挂载ISO失败: %v, 输出: %s", mountErr, string(mountOutput))
		}

	}
	// 等待虚拟光驱加载盘符
	time.Sleep(2 * time.Second)

	// 获取挂载后的盘符列表，找出新增盘符
	output, err = exec.Command("wmic", "logicaldisk", "get", "deviceid").CombinedOutput()
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(output), "\n") {
		drive := strings.TrimSpace(line)
		if drive != "" && drive != "DeviceID" && !currentDrivesMap[drive] {
			newMountedDrive = drive
			break
		}
	}

	if newMountedDrive != "" {
		// 模拟双击盘符，触发AutoRun
		drivePath := newMountedDrive + string(filepath.Separator)
		exec.Command("cmd", "/c", "start", "", drivePath).Start()
		return nil
	}

	return fmt.Errorf("未检测到新挂载的盘符")
}

func (s *DownloadedFilesService) InstallGame(downloadedFile DownloadedFile, installMethod string) (string, error) {
	itemPath := downloadedFile.Path
	installFolder := s.config.GameInstallFolder
	if installFolder == "" {
		return "", fmt.Errorf("游戏安装文件夹未配置")
	}

	applog.LogInfof(s.ctx, "开始安装: %s", itemPath)

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
		applog.LogErrorf(s.ctx, "读取目录失败: %s, 错误: %v", itemPath, err)
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
				applog.LogErrorf(s.ctx, "创建目录失败: %s, 错误: %v", targetPath, err)
				return "", err
			}
			applog.LogInfof(s.ctx, "开始解压ISO: %s -> %s", isoFiles[0], targetPath)
			err := s.ExtractISO(isoFiles[0], targetPath)
			if err != nil {
				applog.LogErrorf(s.ctx, "解压ISO失败: %v", err)
				return "", err
			}
			applog.LogInfof(s.ctx, "安装完成: %s", targetPath)
			return targetPath, nil
		}
		applog.LogErrorf(s.ctx, "包含多个镜像文件，无法自动安装")
		return "", fmt.Errorf("包含多个镜像文件，无法自动安装")
	}

	// 检查文件夹内是否有子文件夹（type=1的情况，解压后的结构）
	// subDirs := []string{}
	// for _, file := range files {
	// 	if file.IsDir() {
	// 		subDirs = append(subDirs, file.Name())
	// 	}
	// }

	// 如果只有一个子文件夹，可能是解压后多了层目录，直接使用子文件夹的内容
	// if len(subDirs) == 1 {
	// 	innerPath := filepath.Join(itemPath, subDirs[0])
	// 	applog.LogInfof(s.ctx, "检测到单层子文件夹，使用内部路径安装: %s", innerPath)
	// 	if err := copyDirectory(innerPath, targetPath); err != nil {
	// 		applog.LogErrorf(s.ctx, "复制目录失败: %v", err)
	// 		return "", err
	// 	}
	// 	applog.LogInfof(s.ctx, "安装完成: %s", targetPath)
	// 	return targetPath, nil
	// }
	sourcePath := itemPath
	if downloadedFile.ExtractedGamePath != "" {
		sourcePath = downloadedFile.ExtractedGamePath
	}

	// 直接复制整个文件夹
	applog.LogInfof(s.ctx, "复制目录: %s -> %s", sourcePath, targetPath)
	if err := copyDirectory(sourcePath, targetPath); err != nil {
		applog.LogErrorf(s.ctx, "复制目录失败: %v", err)
		return "", err
	}

	applog.LogInfof(s.ctx, "安装完成: %s", targetPath)
	return targetPath, nil
}

// DeleteInstalledGame 删除已安装的游戏
func (s *DownloadedFilesService) DeleteInstalledGame(downloadedFile DownloadedFile) error {
	installFolder := s.config.GameInstallFolder
	if installFolder == "" {
		return fmt.Errorf("游戏安装文件夹未配置")
	}

	// 直接使用单元的 gameName 字段，而不是重新计算
	gameName := downloadedFile.GameName
	if gameName == "" {
		// 兼容旧数据，作为后备方案
		baseName := filepath.Base(downloadedFile.Path)
		gameName = s.ExtractGameNameFromDLSite(baseName)
	}

	// 检查以游戏名命名的文件夹
	targetPath := filepath.Join(installFolder, gameName)
	if _, err := os.Stat(targetPath); err == nil {
		applog.LogInfof(s.ctx, "删除安装目录: %s", targetPath)
		if err := os.RemoveAll(targetPath); err != nil {
			applog.LogErrorf(s.ctx, "删除安装目录失败: %v", err)
			return err
		}
		return nil
	}

	// 检查以 md5 命名的文件夹
	md5Name := s.getGameNameMD5(gameName)
	md5Path := filepath.Join(installFolder, md5Name)
	if _, err := os.Stat(md5Path); err == nil {
		applog.LogInfof(s.ctx, "删除安装目录: %s", md5Path)
		if err := os.RemoveAll(md5Path); err != nil {
			applog.LogErrorf(s.ctx, "删除安装目录失败: %v", err)
			return err
		}
		return nil
	}

	return fmt.Errorf("未找到安装目录: %s 或 %s", targetPath, md5Path)
}

func (s *DownloadedFilesService) ExtractISO(isoPath, targetPath string) error {
	sevenZipPath := s.config.SevenZipPath
	if sevenZipPath == "" {
		sevenZipPath = "7z"
	}
	cmd := exec.Command(sevenZipPath, "x", isoPath, "-o"+targetPath, "-y")
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

// DeleteExtractedFolderResult 返回删除解压文件夹的结果
type DeleteExtractedFolderResult struct {
	HasArchive  bool            `json:"has_archive"`
	ArchiveItem *DownloadedFile `json:"archive_item"`
}

// DeleteExtractedFolder 删除解压后的文件夹（压缩包解压出来的内容）
func (s *DownloadedFilesService) DeleteExtractedFolder(itemPath, name string, extractedPaths []string) (*DeleteExtractedFolderResult, error) {
	result := &DeleteExtractedFolderResult{}
	fmt.Println("删除解压后的文件夹:", itemPath)
	downloadFolder := s.config.GameDownloadFolder
	// 压缩包名去掉扩展名就是解压文件夹名
	baseName := strings.TrimSuffix(name, filepath.Ext(name))

	// 删除所有解压路径
	if len(extractedPaths) > 0 {
		for _, path := range extractedPaths {
			if err := os.RemoveAll(path); err != nil {
				applog.LogErrorf(s.ctx, "Failed to remove extracted folder %s: %v", path, err)
			}
		}
	} else {
		// 兼容旧版本，使用原有的单一路径删除逻辑
		extractedFolder := filepath.Join(downloadFolder, baseName)
		if err := os.RemoveAll(extractedFolder); err != nil {
			return nil, err
		}
	}

	// 查找是否有同名压缩包
	compressedExtensions := map[string]bool{
		".zip": true, ".rar": true, ".7z": true, ".tar": true,
		".gz": true, ".bz2": true, ".xz": true, ".z": true, ".lz": true,
	}

	for ext := range compressedExtensions {
		archivePath := filepath.Join(downloadFolder, baseName+ext)
		if _, err := os.Stat(archivePath); err == nil {
			// 找到同名压缩包
			if err != nil {
				return nil, err
			}
			result.HasArchive = true
			info, _ := os.Stat(archivePath)
			// 传入完整的压缩包文件名（带扩展名）
			ai := s.CreateDownloadedFile(archivePath, baseName+ext, false, info.Size())
			result.ArchiveItem = &ai
			return result, nil
		}
	}

	return result, nil
}

// getSingleArchiveItem 获取单个压缩包的单元信息
// func (s *DownloadedFilesService) getSingleArchiveItem(archivePath string) ([]DownloadedFile, error) {
// 	entry, err := os.Stat(archivePath)
// 	if err != nil {
// 		return nil, err
// 	}

// 	name := filepath.Base(archivePath)
// 	// ext := filepath.Ext(name)

// 	item := DownloadedFile{
// 		ID:                uuid.New().String(),
// 		Name:              name,
// 		Path:              archivePath,
// 		ExtractedGamePath: "",
// 		ExtractedPaths:    nil,
// 		Type:              2,
// 		Size:              entry.Size(),
// 		ISOItems:          []string{},
// 		IsDownloading:     s.checkIsDownloading(archivePath, 2),
// 		IsExtracted:       false, // 刚删除解压文件夹，肯定没解压
// 		IsInstalled:       s.checkIsInstalled(archivePath),
// 		IsImported:        s.checkIsImported(name),
// 		HasNumericName:    s.isNumericName(name),
// 		InnerItems:        s.getArchiveInnerItems(archivePath),
// 	}

// 	return []DownloadedFile{item}, nil
// }

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
	// 或: [日期][编号][作者] 游戏名 + 附加内容

	// 匹配模式: 去除开头的元数据部分，提取游戏名
	// 元数据格式: (xxx) [内容][内容][内容]... 任意数量的[内容]
	// 游戏名后面可能跟着: 1) 没有后缀 2) (文件类型) 3) + 附加内容
	re := regexp.MustCompile(`^(?:\([^)]+\)\s*)?(?:\[[^\]]+\]\s*)*(.+?)(?:\s*[\(+].*)?$`)

	matches := re.FindStringSubmatch(filename)
	if len(matches) < 2 {
		return filename
	}

	return strings.TrimSpace(matches[1])
}

// JudgeGameName 用于判断文件名是否为游戏名
func (s *DownloadedFilesService) JudgeGameName(filenames []string) string {
	gameName := ""
	if len(filenames) == 0 {
		return gameName
	}
	type GameNameScore struct {
		Name  string
		Score float64
	}
	gameNameScores := []GameNameScore{}
	plusWords := []string{"パッケージ版", "mdf", "mds", "iso"}
	minusWords := []string{"サウンドトラック", "wav", "mp3", "flac", "cue", "ボイス", "ドラマ", "アップデート", "update", "特典", "Drama", "CD", "part", "00"}
	for _, filename := range filenames {
		score := 1.0

		for _, word := range plusWords {
			if strings.Contains(filename, word) {
				score += 0.5
			}
		}
		for _, word := range minusWords {
			if strings.Contains(filename, word) {
				score -= 0.4
			}
		}

		nameScore := GameNameScore{
			Name:  filename,
			Score: score,
		}
		gameNameScores = append(gameNameScores, nameScore)
	}
	sort.Slice(gameNameScores, func(i, j int) bool {
		return gameNameScores[i].Score > gameNameScores[j].Score
	})
	if len(gameNameScores) > 0 {
		gameName = gameNameScores[0].Name
	}

	return gameName
}
