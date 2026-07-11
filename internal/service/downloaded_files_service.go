package service

import (
	"archive/zip"
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"lunabox/internal/appconf"
	"lunabox/internal/applog"
	"lunabox/internal/enums"
	"lunabox/internal/models"
	"lunabox/internal/utils"
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
	ctx         context.Context
	db          *sql.DB
	config      *appconf.AppConfig
	taskService *TaskService
}

func NewDownloadedFilesService() *DownloadedFilesService {
	return &DownloadedFilesService{}
}

func (s *DownloadedFilesService) Init(ctx context.Context, db *sql.DB, config *appconf.AppConfig) {
	s.ctx = ctx
	s.db = db
	s.config = config
}

func (s *DownloadedFilesService) SetTaskService(taskService *TaskService) {
	s.taskService = taskService
}

type DownloadedFile struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"` //单元标题，不能数字，不包含文件后缀
	Path              string   `json:"path"` //全路径，文件的话包含后缀
	BaseName          string   `json:"base_name,omitempty"`
	ExtractedGamePath string   `json:"extracted_game_path,omitempty"` // 解压后的游戏文件夹路径（主路径）
	ExtractedPaths    []string `json:"extracted_paths,omitempty"`     // 所有解压后的文件夹路径
	// IsFolder          bool     `json:"is_folder"`
	Type int `json:"type"` // 0:是文件夹没任何压缩包不需解压; 1: 单元文件夹里包含一个或多个压缩包;
	// 2: 游戏下载文件夹下单个压缩包或多了同名文件夹（解压后）结构; 3：单独安装文件夹内的游戏文件夹，下载文件夹里没关联上也非已导入
	Size int64 `json:"size"`
	// IsDownloading bool      `json:"is_downloading"` // 是否有未下载完的临时文件
	// IsExtracted   bool      `json:"is_extracted"`
	// IsInstalled   bool      `json:"is_installed"`
	// IsImported    bool      `json:"is_imported"`
	ImportedId    string    `json:"imported_id,omitempty"`
	InstalledPath string    `json:"installed_path,omitempty"`
	Time          time.Time `json:"time"`
	// ContainsISO       bool     `json:"contains_iso"`
	ISOItems []string `json:"iso_items"`
	// ISOCount          int      `json:"iso_count"`
	// ISOFilePath       string   `json:"iso_file_path,omitempty"`
	InnerItems     []string `json:"inner_items"`
	HasNumericName bool     `json:"has_numeric_name"`
	GameName       string   `json:"game_name"` //从单元标题再继续去除描述性前缀和后缀
	Status         int      `json:"status"`    //0:下载中、1:已下载、2:已解压、3:已安装 4:已导入
	IsChanged      bool     `json:"is_changed"`
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

func (s *DownloadedFilesService) extractTimeFromName(name string, defaultTime time.Time) time.Time {
	re := regexp.MustCompile(`\[(\d{6})\]`)

	matches := re.FindStringSubmatch(name)
	if len(matches) > 1 {
		dateStr := matches[1]
		if len(dateStr) == 6 {
			year, err1 := strconv.Atoi("20" + dateStr[:2])
			month, err2 := strconv.Atoi(dateStr[2:4])
			day, err3 := strconv.Atoi(dateStr[4:6])
			if err1 == nil && err2 == nil && err3 == nil {
				return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
			}
		}
	}
	return defaultTime
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
	// var folderItems []DownloadedFile
	var folderItems2 map[*DownloadedFile]*DownloadedFile = make(map[*DownloadedFile]*DownloadedFile)
	// var archiveItems []DownloadedFile
	var archiveItems2 map[*DownloadedFile]*DownloadedFile = make(map[*DownloadedFile]*DownloadedFile)

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
		fileNameWithoutExt := strings.TrimSuffix(name, filepath.Ext(name))
		fileTime := s.extractTimeFromName(fileNameWithoutExt, info.ModTime())
		item, savedInfo := s.CreateDownloadedFile(itemPath, name, isFolder, info.Size(), fileTime, false)
		if isFolder {
			// folderItems = append(folderItems, item)
			folderItems2[&item] = savedInfo
		} else {
			// archiveItems = append(archiveItems, item)
			archiveItems2[&item] = savedInfo
		}
	}

	// 建立文件夹名称的集合
	folderNames := make(map[string]bool)
	for folder, _ := range folderItems2 {
		folderNames[folder.BaseName] = true
	}

	// 建立压缩包的基础名称映射
	archiveBaseNames := make(map[string]DownloadedFile)
	for archive, _ := range archiveItems2 {
		baseName := strings.TrimSuffix(filepath.Base(archive.Path), filepath.Ext(archive.Path))
		archiveBaseNames[baseName] = *archive
	}

	// 记录哪些压缩包已经被显示（因为对应的文件夹有内容）
	displayedArchives := make(map[string]bool)

	// 合并逻辑：文件夹有内容时显示文件夹，没有内容但有同名压缩包时显示压缩包
	for folder, savedInfo := range folderItems2 {
		baseName := strings.TrimSuffix(filepath.Base(folder.Path), filepath.Ext(folder.Path))
		// savedInfo := s.LoadDownloadInfo(folder.Path)

		// 检查是否有同名压缩包
		if _, exists := archiveBaseNames[baseName]; exists {
			// fmt.Printf("status ？ 01,name: %s, status:%d\n", folder.Name, folder.Status)
			// 存在同名压缩包
			folder.Type = 2
			if s.checkIsDownloading(folder.Path, 2, savedInfo) {
				folder.Status = 0
			} else {
				if folder.Status < 1 {
					folder.Status = 1
				}
				if s.checkIsExtracted(folder.Path, folder.BaseName, 2) && folder.Status < 2 {
					// fmt.Printf("status 2 02,name: %s, status:%d\n", folder.Name, folder.Status)
					folder.Status = 2
				}
			}

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
				if s.checkIsDownloading(folder.Path, 1, savedInfo) {
					folder.Status = 0
				} else {
					if folder.Status < 1 {
						folder.Status = 1
					}
					if s.checkIsExtracted(folder.Path, folder.BaseName, 1) && folder.Status < 2 {
						// fmt.Printf("status 2 01,name: %s, \n", folder.Name)

						folder.Status = 2
					}
				}
			} else {
				folder.Type = 0
				folder.Status = 2
				// fmt.Printf("status 2 03,name: %s, \n", folder.Name)
			}
		}

		if folder.Status >= 2 {
			extractedGamePath, extractedPaths := s.getExtractedPaths(folder.Path, folder.BaseName, folder.Type)
			// fmt.Printf("extractedPath,name: %s, eGamePath: %s, ePaths: %d, path:%s\n", folder.Name, extractedGamePath, len(extractedPaths), folder.Path)
			folder.ExtractedGamePath = extractedGamePath
			folder.ExtractedPaths = extractedPaths
		}
		s.SaveDownloadInfo(folder.Path, *folder)

		items = append(items, *folder)
		displayedArchives[baseName] = true
	}

	// 添加没有被显示的压缩包
	for archive, savedInfo := range archiveItems2 {
		baseName := strings.TrimSuffix(archive.BaseName, filepath.Ext(archive.BaseName))
		// savedInfo := s.LoadDownloadInfo(archive.Path)
		if !displayedArchives[baseName] {
			archive.Type = 2
			if s.checkIsDownloading(archive.Path, 2, savedInfo) {
				archive.Status = 0
			} else {
				if archive.Status < 1 {
					archive.Status = 1
				}
				if s.checkIsExtracted(archive.Path, archive.BaseName, 2) && archive.Status < 2 {
					archive.Status = 2
					// fmt.Printf("status 2 04,name: %s, \n", archive.Name)
				}
			}
			items = append(items, *archive)
		}
	}

	//查找安装文件夹看看有没type3的文件夹
	if s.config.GameInstallFolder != "" {
		installedEntries, err := os.ReadDir(s.config.GameInstallFolder)
		if err == nil {
			for _, entry := range installedEntries {
				itemPath := filepath.Join(s.config.GameInstallFolder, entry.Name())
				isInDownloadFolder := utils.Contains(items, func(d DownloadedFile) bool {
					return d.InstalledPath == itemPath
				})
				// fmt.Printf("ListDown1 itemPath: %s, isInDownloadFolder: %v\n", itemPath, isInDownloadFolder)
				if isInDownloadFolder {
					continue
				}
				if !s.checkPathImported(itemPath) {
					base := filepath.Base(itemPath)
					fileNameWithoutExt := strings.TrimSuffix(base, filepath.Ext(base))
					info, _ := os.Stat(itemPath)
					defaultTime := time.Time{}
					if info != nil {
						defaultTime = info.ModTime()
					}
					fileTime := s.extractTimeFromName(fileNameWithoutExt, defaultTime)
					item, _ := s.CreateDownloadedFile(itemPath, base, true, 0, fileTime, true)
					item.Type = 3
					item.Status = 3
					item.InstalledPath = itemPath
					items = append(items, item)
				}
			}
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
	fileNameWithoutExt := strings.TrimSuffix(filepath.Base(path), filepath.Ext(filepath.Base(path)))
	info, _ := os.Stat(path)
	defaultTime := time.Time{}
	if info != nil {
		defaultTime = info.ModTime()
	}
	fileTime := s.extractTimeFromName(fileNameWithoutExt, defaultTime)
	archive, _ := s.CreateDownloadedFile(path, filepath.Base(path), true, 0, fileTime, true)
	if file.Type == 2 {
		if archive.InnerItems != nil && len(archive.InnerItems) > 0 {
			if len(archive.InnerItems) == 1 {
				gamepath := filepath.Join(path, archive.InnerItems[0])
				archive.ExtractedGamePath = gamepath
			} else {
				archive.ExtractedGamePath = path
			}

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
 * 创建下载文件对象
 * itemPath 全路径
 * name 包括后缀的文件名或文件夹名
 */
func (s *DownloadedFilesService) CreateDownloadedFile(itemPath, name string, isFolder bool, size int64, fileTime time.Time, shouldSave bool) (DownloadedFile, *DownloadedFile) {
	fileNameWithoutExt := strings.TrimSuffix(name, filepath.Ext(name))

	// 读取 download.klb 获取保存的信息
	savedInfo := s.LoadDownloadInfo(itemPath)

	// 检测已导入的ID（优先从保存的信息读取）
	importedID := ""
	if savedInfo != nil {
		savedInfo.IsChanged = false
	}
	status := 0

	if savedInfo != nil && savedInfo.ImportedId != "" {
		status = savedInfo.Status
		// 检查导入ID是否在数据库中真的存在
		if !s.checkImportedIDInDB(savedInfo.ImportedId) && isFolder {
			// 数据库中不存在，清空保存的导入信息
			savedInfo.ImportedId = ""
			if savedInfo.Status > 3 {
				savedInfo.Status = 3
				// fmt.Printf("status 3 01,name: %s, status:%d\n", name, savedInfo.Status)
				status = 3
			}
			savedInfo.IsChanged = true
		} else {
			importedID = savedInfo.ImportedId
			status = 4
		}
	}

	// 使用保存的游戏名（如果有）
	gameName := s.ExtractGameName(fileNameWithoutExt)
	if savedInfo != nil && savedInfo.GameName != "" {
		gameName = savedInfo.GameName
	}

	// 检查安装路径是否准确
	installedPath := ""
	// fmt.Printf("status 3 -2,name: %s, status:%d\n", name, status)
	if savedInfo != nil && savedInfo.InstalledPath != "" {
		status = savedInfo.Status
		// 检查保存的安装路径是否真的存在
		if _, err := os.Stat(savedInfo.InstalledPath); err == nil {
			installedPath = savedInfo.InstalledPath
		} else {
			// 路径不存在，使用原来的逻辑重新获取
			installedPath = s.checkAndGetInstalledPath(itemPath, savedInfo)
			// 更新保存的安装路径
			savedInfo.InstalledPath = installedPath

			savedInfo.IsChanged = true
		}
		// fmt.Printf("status 3 -4,name: %s, status:%d, savedStatus:%d, installedPath:%s\n", name, status, savedInfo.Status, installedPath)
		if installedPath != "" {
			if savedInfo.Status < 3 {
				savedInfo.Status = 3

				// fmt.Printf("status 3 01,name: %s, status:%d\n", name, savedInfo.Status)
				status = 3
				savedInfo.IsChanged = true
			}
		} else if savedInfo.Status > 2 {
			savedInfo.Status = 2
			// fmt.Printf("status 2 05,name: %s, \n", savedInfo.Name)
			status = 2
			savedInfo.IsChanged = true
		}
	} else {
		installedPath = s.checkAndGetInstalledPath(itemPath, savedInfo)
		// fmt.Printf("status 3 -1,name: %s, status:%d, installedPath: %s\n", name, status, installedPath)
		if installedPath != "" {
			if status < 3 {
				status = 3

				// fmt.Printf("status 3 02,name: %s, status:%d\n", name, status)
			}
		} else if status > 2 {
			status = 2
		}
		if savedInfo != nil {
			savedInfo.IsChanged = true
			savedInfo.Status = status
			savedInfo.InstalledPath = installedPath
		}
	}
	if savedInfo != nil && savedInfo.IsChanged && isFolder && shouldSave {
		s.SaveDownloadInfo(itemPath, *savedInfo)
	}

	item := DownloadedFile{
		ID:       uuid.New().String(),
		Name:     fileNameWithoutExt,
		BaseName: name,
		Path:     itemPath,
		ISOItems: []string{},
		// IsFolder:          isFolder,
		Type: 2,
		Size: size,
		Time: fileTime,
		// IsDownloading:     s.checkIsDownloading(itemPath, name, itemType),
		// IsExtracted:       s.checkIsExtracted(itemPath, name, itemType),
		Status:         status,
		ImportedId:     importedID,
		InstalledPath:  installedPath,
		HasNumericName: s.isNumericName(name),
		GameName:       gameName,
	}

	if isFolder {
		innerItems, isoItems := s.getInnerItems(itemPath)
		// fmt.Printf("InnerItems,name: %s, path:%s, count:%v\n", item.Name, item.Path, innerItems)
		if item.HasNumericName {
			item.Name = s.JudgeGameName(innerItems)
			item.GameName = s.ExtractGameName(item.Name)
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
	return item, savedInfo
}

func (s *DownloadedFilesService) DownloadSaves(games []models.Game, isOverride bool) error {
	if len(games) == 0 {
		return nil
	}
	getter := utils.NewSaveInfoGetter()
	return getter.DownloadSavesForGames(games, isOverride)
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
		baseName := strings.ToLower(strings.TrimSuffix(entry.Name(), ext))
		if ext == ".!ut" || ext == ".!qb" {
			ext = strings.ToLower(filepath.Ext(baseName))
		}
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
	// fmt.Printf("getExtractedPaths 01 %s\n", name)

	if fileType == 2 {
		// 对于压缩包，返回同名文件夹路径
		baseName := strings.TrimSuffix(name, filepath.Ext(name))
		folderPath := filepath.Join(downloadFolder, baseName)
		// fmt.Printf("getExtractedPaths 02 %s\n", name)
		if info, err := os.Stat(folderPath); err == nil && info.IsDir() {

			// fmt.Printf("getExtractedPaths 03 %s\n", name)
			subEntries, err := os.ReadDir(folderPath)
			if err == nil {
				folders := []string{}
				for _, entry := range subEntries {
					if entry.IsDir() {
						folders = append(folders, entry.Name())
					}
				}
				if len(folders) == 1 && len(subEntries) <= 2 {
					// fmt.Printf("getExtractedPaths 04 %s\n", name)
					fullpath := filepath.Join(folderPath, folders[0])
					return fullpath, []string{folderPath}
				}

			}
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

// func (s *DownloadedFilesService) checkIsInstalled(itemPath string) bool {
// 	installedPath := s.checkAndGetInstalledPath(itemPath)
// 	return installedPath != ""
// }

const downloadInfoFileName = "download.klb"

// SaveDownloadInfo 保存下载单元信息到 download.klb 文件
func (s *DownloadedFilesService) SaveDownloadInfo(itemPath string, info DownloadedFile) error {
	klbPath := filepath.Join(itemPath, downloadInfoFileName)
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return os.WriteFile(klbPath, data, 0644)
}

// LoadDownloadInfo 从 download.klb 文件加载下载单元信息
func (s *DownloadedFilesService) LoadDownloadInfo(itemPath string) *DownloadedFile {
	klbPath := filepath.Join(itemPath, downloadInfoFileName)
	data, err := os.ReadFile(klbPath)
	if err != nil {
		return nil
	}
	var info DownloadedFile
	if err := json.Unmarshal(data, &info); err != nil {
		return nil
	}
	return &info
}

// checkAndGetInstalledPath 检查是否已安装并返回安装路径
func (s *DownloadedFilesService) checkAndGetInstalledPath(itemPath string, info *DownloadedFile) string {
	// 优先从 download.klb 读取安装路径
	// info := s.LoadDownloadInfo(itemPath)
	if info != nil && info.InstalledPath != "" {
		if _, err := os.Stat(info.InstalledPath); err == nil {
			return info.InstalledPath
		}
	}

	installFolder := s.config.GameInstallFolder
	if installFolder == "" {
		return ""
	}

	// 获取原始文件夹名作为游戏名
	baseName := filepath.Base(itemPath)
	gameName := s.ExtractGameName(baseName)

	// 检查以游戏名命名的文件夹
	gameNamePath := filepath.Join(installFolder, gameName)
	if _, err := os.Stat(gameNamePath); err == nil {
		return gameNamePath
	}

	// 检查以 md5 命名的文件夹
	md5Name := s.getGameNameMD5(gameName)
	md5Path := filepath.Join(installFolder, md5Name)
	if _, err := os.Stat(md5Path); err == nil {
		return md5Path
	}

	return ""
}

// getGameNameMD5 生成游戏名的 md5 哈希
func (s *DownloadedFilesService) getGameNameMD5(gameName string) string {
	data := []byte(gameName)
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}

// GetGameNameMD5 公开的游戏名转MD5函数
func (s *DownloadedFilesService) GetGameNameMD5(gameName string) string {
	return s.getGameNameMD5(gameName)
}

func (s *DownloadedFilesService) checkIsImported(name string) bool {
	return false
}

// checkImportedIDInDB 检查数据库中是否存在该导入ID的游戏
func (s *DownloadedFilesService) checkImportedIDInDB(importedId string) bool {
	if importedId == "" {
		return false
	}
	var exists bool
	err := s.db.QueryRowContext(s.ctx, "SELECT EXISTS(SELECT 1 FROM games WHERE id = ?)", importedId).Scan(&exists)
	if err != nil {
		applog.LogErrorf(s.ctx, "检查导入ID失败: %v", err)
		return false
	}
	return exists
}

// checkPathImported 检查路径是否已导入
func (s *DownloadedFilesService) checkPathImported(path string) bool {
	if path == "" {
		return false
	}
	var exists bool
	err := s.db.QueryRowContext(s.ctx, "SELECT EXISTS(SELECT 1 FROM games WHERE contains(path, ?))", path).Scan(&exists)
	if err != nil {
		applog.LogErrorf(s.ctx, "检查导入ID失败: %v", err)
		return false
	}
	return exists
}

// checkIsDownloading 检查是否有未下载完的临时文件（QBittorrent 的 .!qB 文件，uTorrent 的 .!ut 文件）
func (s *DownloadedFilesService) checkIsDownloading(itemPath string, itemType int, savedInfo *DownloadedFile) bool {
	// savedInfo := s.LoadDownloadInfo(itemPath)
	if savedInfo != nil && savedInfo.Status > 0 {
		return false
	}

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
		time.Sleep(5 * time.Second)

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
	extractedGameName := s.ExtractGameName(baseName)

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
			// 保存安装信息到 download.klb
			s.saveInstallInfo(downloadedFile.Path, targetPath)
			applog.LogInfof(s.ctx, "安装完成: %s", targetPath)
			return targetPath, nil
		}
		applog.LogErrorf(s.ctx, "包含多个镜像文件，无法自动安装")
		return "", fmt.Errorf("包含多个镜像文件，无法自动安装")
	}

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

	// 保存安装信息到 download.klb
	s.saveInstallInfo(downloadedFile.Path, targetPath)

	applog.LogInfof(s.ctx, "安装完成: %s", targetPath)
	return targetPath, nil
}

// saveInstallInfo 保存安装信息到 download.klb
func (s *DownloadedFilesService) saveInstallInfo(itemPath, installedPath string) {
	// 读取现有信息
	info := s.LoadDownloadInfo(itemPath)
	if info == nil {
		info = &DownloadedFile{}
	}
	info.InstalledPath = installedPath
	if info.Status < 3 {
		info.Status = 3
	}
	s.SaveDownloadInfo(itemPath, *info)
}

func (s *DownloadedFilesService) OverwriteInstall(downloadedFile DownloadedFile, exePath string, layers int) error {
	applog.LogInfof(s.ctx, "开始覆盖安装: %s, exePath: %s, layers: %d", downloadedFile.Path, exePath, layers)

	exeFileName := filepath.Base(exePath)
	targetBasePath := filepath.Dir(exePath)

	var sourcePath string
	var extractedExePath string

	if len(downloadedFile.ISOItems) == 0 {
		if downloadedFile.ExtractedGamePath == "" {
			return fmt.Errorf("未找到解压路径")
		}

		var foundExePaths []string
		err := filepath.Walk(downloadedFile.ExtractedGamePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Base(path) == exeFileName {
				foundExePaths = append(foundExePaths, path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("搜索文件失败: %v", err)
		}

		if len(foundExePaths) != 1 {
			return fmt.Errorf("找到 %d 个匹配的执行文件，需要找到恰好1个", len(foundExePaths))
		}

		extractedExePath = foundExePaths[0]
		sourcePath = filepath.Dir(extractedExePath)

		for i := 0; i < layers-1; i++ {
			sourcePath = filepath.Dir(sourcePath)
			targetBasePath = filepath.Dir(targetBasePath)
		}

		applog.LogInfof(s.ctx, "覆盖目录: %s -> %s", sourcePath, targetBasePath)
		if strings.Contains(s.config.GameInstallFolder, targetBasePath) {
			return errors.New("无法覆盖游戏安装根目录或其父目录，请调整层级")
		}
		return copyDirectory(sourcePath, targetBasePath)
	} else if len(downloadedFile.ISOItems) == 1 {
		isoPath := downloadedFile.ISOItems[0]

		tmpDir, err := os.MkdirTemp("", "lunabox-overwrite-*")
		if err != nil {
			return fmt.Errorf("创建临时目录失败: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		applog.LogInfof(s.ctx, "解压ISO到临时目录: %s -> %s", isoPath, tmpDir)
		if err := s.ExtractISO(isoPath, tmpDir); err != nil {
			return fmt.Errorf("解压ISO失败: %v", err)
		}

		var foundExePaths []string
		err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Base(path) == exeFileName {
				foundExePaths = append(foundExePaths, path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("搜索文件失败: %v", err)
		}

		if len(foundExePaths) != 1 {
			return fmt.Errorf("在镜像中找到 %d 个匹配的执行文件，需要找到恰好1个", len(foundExePaths))
		}

		extractedExePath = foundExePaths[0]
		sourcePath = filepath.Dir(extractedExePath)

		for i := 0; i < layers-1; i++ {
			sourcePath = filepath.Dir(sourcePath)
			targetBasePath = filepath.Dir(targetBasePath)
		}
		if strings.Contains(s.config.GameInstallFolder, targetBasePath) {
			return errors.New("无法覆盖游戏安装根目录或其父目录，请调整层级")
		}

		applog.LogInfof(s.ctx, "从临时目录覆盖: %s -> %s", sourcePath, targetBasePath)
		return copyDirectory(sourcePath, targetBasePath)
	} else {
		return fmt.Errorf("包含多个镜像文件，无法自动覆盖安装")
	}
}

// UpdateGameName 更新游戏名并保存到 download.klb
func (s *DownloadedFilesService) UpdateGameName(itemPath, gameName string) error {
	info := s.LoadDownloadInfo(itemPath)
	if info == nil {
		info = &DownloadedFile{}
	}
	info.GameName = gameName
	return s.SaveDownloadInfo(itemPath, *info)
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
		gameName = s.ExtractGameName(baseName)
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

func (s *DownloadedFilesService) DeleteItem(downloadedFile DownloadedFile) error {
	itemPath := downloadedFile.Path
	var err error = nil
	if downloadedFile.Status > 1 && len(downloadedFile.ExtractedPaths) > 0 {
		for _, extractedPath := range downloadedFile.ExtractedPaths {
			err = os.RemoveAll(extractedPath)
		}
	}
	err = os.RemoveAll(itemPath)
	return err
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
			fileTime := s.extractTimeFromName(baseName, info.ModTime())
			// 传入完整的压缩包文件名（带扩展名）
			ai, _ := s.CreateDownloadedFile(archivePath, baseName+ext, false, info.Size(), fileTime, true)
			result.ArchiveItem = &ai
			return result, nil
		}
	}

	return result, nil
}

// SaveImportedID 保存导入的ID到文件
func (s *DownloadedFilesService) SaveImportedID(itemPath, importedID string) error {
	// 对于压缩包，使用同名的解压文件夹路径
	baseName := filepath.Base(itemPath)
	ext := filepath.Ext(baseName)
	if ext != "" {
		// 是压缩包，查找同名文件夹
		folderPath := strings.TrimSuffix(itemPath, ext)
		if info, err := os.Stat(folderPath); err == nil && info.IsDir() {
			itemPath = folderPath
		}
	}

	// 保存到 download.klb
	info := s.LoadDownloadInfo(itemPath)
	if info == nil {
		info = &DownloadedFile{}
	}
	info.ImportedId = importedID
	if info.Status < 4 {
		info.Status = 4
	}
	return s.SaveDownloadInfo(itemPath, *info)
}

// SaveDownloadedFileInfo 保存下载单元信息到 download.klb
func (s *DownloadedFilesService) SaveDownloadedFileInfo(itemPath string, info DownloadedFile) error {
	// 对于压缩包，使用同名的解压文件夹路径
	baseName := filepath.Base(itemPath)
	ext := filepath.Ext(baseName)
	if ext != "" {
		folderPath := strings.TrimSuffix(itemPath, ext)
		if stat, err := os.Stat(folderPath); err == nil && stat.IsDir() {
			itemPath = folderPath
		}
	}
	return s.SaveDownloadInfo(itemPath, info)
}

// ScanFolderForExecutables 扫描文件夹查找可执行文件
func (s *DownloadedFilesService) ScanFolderForExecutables(folderPath string) ([]string, error) {

	// 查找可执行文件
	executables := utils.FindExecutables(folderPath, utils.ExcludeExeKeywords, 2)
	fmt.Printf("ScanFolderForExecutables found executables: %v\n", executables)

	if len(executables) == 0 {
		return nil, nil
	}

	// 选择最佳可执行文件
	// folderName := filepath.Base(folderPath)
	// selectedExe := utils.SelectBestExecutable(executables, folderName)

	// fmt.Printf("ScanFolderForExecutables2 found executables: %v, selectedExe:%s\n", executables, selectedExe)

	return executables, nil
}

func (s *DownloadedFilesService) StartGameTemp(path string) error {
	var cmd *exec.Cmd = exec.Command(path)
	return cmd.Start()
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
func (s *DownloadedFilesService) ExtractGameName(filename string) string {
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

// GetTitlesNum 公开的获取标题和数字的函数
func (s *DownloadedFilesService) GetTitlesNum(searchName string) string {
	mainT, _, _ := utils.GetTitles(searchName)
	return mainT
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
	minusWords := []string{"サウンドトラック", "wav", "mp3", "flac", "cue", "ボイス", "ドラマ", "アップデート", "update", "特典", "Drama", "CD", "part", "00", "download", "klb"}
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

func (s *DownloadedFilesService) ExecuteBatchTask(items []DownloadedFile, showExtract, showInstall, showImport bool, directIsoInstall bool, installMethod string) error {
	if s.taskService == nil {
		return fmt.Errorf("任务服务未初始化")
	}

	taskUUID := uuid.New().String()
	s.taskService.RegisterTaskFunction(taskUUID, s.createBatchProcessTaskFunction(items, showExtract, showInstall, showImport, directIsoInstall, installMethod))

	taskData := map[string]interface{}{
		"items":            items,
		"showExtract":      showExtract,
		"showInstall":      showInstall,
		"showImport":       showImport,
		"directIsoInstall": directIsoInstall,
		"installMethod":    installMethod,
	}

	return s.taskService.StartTask("game_updates", taskUUID, 0, enums.DownloadFiles, len(items), taskData)
}

func (s *DownloadedFilesService) createBatchProcessTaskFunction(items []DownloadedFile, showExtract, showInstall, showImport bool, directIsoInstall bool, installMethod string) TaskFunction {
	return func(ctx context.Context, data string, updateProgress func(completed int, total int,
		workingOn string, warning string, itemId string, itemEvent enums.TaskStatus, itemData interface{})) error {

		updateProgress(0, len(items), "开始批量处理下载文件", "", "", enums.Started, nil)

		for index := range items {
			item := &items[index]
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			updateProgress(index, len(items), fmt.Sprintf("处理文件: %s", item.Name),
				"", item.ID, enums.Initial, *item)

			if showExtract && item.Status == 1 {
				updateProgress(index, len(items), fmt.Sprintf("解压中: %s", item.Name),
					"", item.ID, enums.Started, *item)

				var err error
				if item.Type == 1 {
					err = s.ExtractArchivesInFolder(item.Path)
				} else {
					err = s.ExtractItem(item.Path)
				}

				if err != nil {
					updateProgress(index+1, len(items), fmt.Sprintf("解压失败: %s", item.Name),
						err.Error(), item.ID, enums.Error, *item)
					continue
				}
				fmt.Printf("prerefresh %s, status:%d, type:%d, isos:%d\n", item.Name, item.Status, item.Type, len(item.ISOItems))

				refreshedItem, err := s.RefreshDownloadedFile(*item)
				if err != nil {
					updateProgress(index+1, len(items), fmt.Sprintf("刷新状态失败: %s", item.Name),
						err.Error(), item.ID, enums.Error, *item)
					continue
				}
				refreshedItem.Type = item.Type
				refreshedItem.Status = 2
				*item = refreshedItem

				updateProgress(index, len(items), fmt.Sprintf("解压完成: %s", item.Name),
					"", item.ID, enums.Completed, *item)
			}

			hasIso := len(item.ISOItems) > 0
			shouldInstall := showInstall && item.Status == 2 && item.ExtractedGamePath != "" && (directIsoInstall || !hasIso)
			fmt.Printf("shouldInstall %s: %v, status:%d, type:%d, isos:%d\n", item.Name, shouldInstall, item.Status, item.Type, len(item.ISOItems))

			if shouldInstall {
				updateProgress(index, len(items), fmt.Sprintf("安装中: %s", item.Name),
					"", item.ID, enums.Started, *item)

				installedPath, err := s.InstallGame(*item, installMethod)
				if err != nil {
					updateProgress(index+1, len(items), fmt.Sprintf("安装失败: %s", item.Name),
						err.Error(), item.ID, enums.Error, *item)
					continue
				}

				item.InstalledPath = installedPath
				item.Status = 3

				updateProgress(index+1, len(items), fmt.Sprintf("安装完成: %s", item.Name),
					"", item.ID, enums.Completed, *item)
			} else {
				updateProgress(index+1, len(items), fmt.Sprintf("跳过安装: %s", item.Name),
					"", item.ID, enums.Completed, *item)
			}
		}

		var importItems []DownloadedFile
		if showImport {
			importItems = []DownloadedFile{}
			for _, item := range items {
				if item.Status == 3 {
					importItems = append(importItems, item)
				}
			}
		}

		updateProgress(len(items), len(items), "批量处理完成", "", "", enums.Completed, importItems)
		return nil
	}
}
