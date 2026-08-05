package service

import (
	"context"
	"database/sql"
	"fmt"
	"lunabox/internal/appconf"
	"lunabox/internal/applog"
	"lunabox/internal/migrations"
	"lunabox/internal/models"
	"lunabox/internal/service/cloudprovider"
	"lunabox/internal/service/cloudprovider/onedrive"
	"lunabox/internal/utils"
	"lunabox/internal/vo"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type BackupService struct {
	ctx    context.Context
	db     *sql.DB
	config *appconf.AppConfig
}

func NewBackupService() *BackupService {
	return &BackupService{}
}

func (s *BackupService) Init(ctx context.Context, db *sql.DB, config *appconf.AppConfig) {
	s.ctx = ctx
	s.db = db
	s.config = config
}

// getCloudProvider 获取云备份提供商
func (s *BackupService) getCloudProvider() (cloudprovider.CloudStorageProvider, error) {
	return cloudprovider.NewCloudProvider(s.ctx, s.config)
}

// SelectBackupSavePath 选择全量备份保存路径
func (s *BackupService) SelectBackupSavePath() (string, error) {
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	defaultFileName := fmt.Sprintf("kaleidobox_full_%s.zip", timestamp)

	selection, err := runtime.SaveFileDialog(s.ctx, runtime.SaveDialogOptions{
		Title:           "选择全量备份保存位置",
		DefaultFilename: defaultFileName,
		Filters: []runtime.FileFilter{
			{
				DisplayName: "ZIP 压缩包 (*.zip)",
				Pattern:     "*.zip",
			},
		},
	})
	if err != nil {
		applog.LogErrorf(s.ctx, "failed to open save file dialog: %v", err)
	}
	return selection, err
}

// SelectBackupRestorePath 选择要恢复的全量备份文件
func (s *BackupService) SelectBackupRestorePath() (string, error) {
	selection, err := runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "选择要恢复的全量备份文件",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "ZIP 压缩包 (*.zip)",
				Pattern:     "*.zip",
			},
		},
	})
	if err != nil {
		applog.LogErrorf(s.ctx, "failed to open file dialog: %v", err)
	}
	return selection, err
}

// SelectDBBackupRestorePath 选择要恢复的数据库备份文件
func (s *BackupService) SelectDBBackupRestorePath() (string, error) {
	selection, err := runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "选择要恢复的数据库备份文件",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "ZIP 压缩包 (*.zip)",
				Pattern:     "*.zip",
			},
		},
	})
	if err != nil {
		applog.LogErrorf(s.ctx, "failed to open file dialog: %v", err)
	}
	return selection, err
}

// ========== 云备份配置相关方法 ==========

// SetupCloudBackup 设置云备份密码（只能设置一次）
func (s *BackupService) SetupCloudBackup(password string) (string, error) {
	// 检查是否已经设置过密码
	if s.config.BackupPassword != "" {
		applog.LogWarningf(s.ctx, "SetupCloudBackup: backup password already set")
		return "", fmt.Errorf("备份密码已设置，无法修改")
	}

	if password == "" {
		applog.LogWarningf(s.ctx, "SetupCloudBackup: backup password is empty")
		return "", fmt.Errorf("备份密码不能为空")
	}

	// 生成用户ID
	userID := utils.GenerateUserID(password)

	// 更新配置
	s.config.BackupUserID = userID
	s.config.BackupPassword = password

	// 立即保存配置到文件
	if err := appconf.SaveConfig(s.config); err != nil {
		applog.LogErrorf(s.ctx, "SetupCloudBackup: failed to save config: %v", err)
		return "", fmt.Errorf("保存配置失败: %w", err)
	}

	applog.LogInfof(s.ctx, "SetupCloudBackup: backup password set successfully, user_id: %s", userID)
	return userID, nil
}

// TestS3Connection 测试 S3 连接
func (s *BackupService) TestS3Connection(config appconf.AppConfig) error {
	if err := cloudprovider.TestConnection(s.ctx, cloudprovider.ProviderS3, &config); err != nil {
		applog.LogErrorf(s.ctx, "TestS3Connection: connection test failed: %v", err)
		return fmt.Errorf("连接测试失败: %w", err)
	}
	return nil
}

// TestOneDriveConnection 测试 OneDrive 连接
func (s *BackupService) TestOneDriveConnection(config appconf.AppConfig) error {
	return cloudprovider.TestConnection(s.ctx, cloudprovider.ProviderOneDrive, &config)
}

// TestWebDavConnection 测试 WebDav 连接
func (s *BackupService) TestWebDavConnection(config appconf.AppConfig) error {
	return cloudprovider.TestConnection(s.ctx, cloudprovider.ProviderWebDav, &config)
}

// GetOneDriveAuthURL 获取 OneDrive 授权 URL
func (s *BackupService) GetOneDriveAuthURL() string {
	return onedrive.GetOneDriveAuthURL(s.config.OneDriveClientID)
}

// StartOneDriveAuth 启动 OneDrive 授权流程（使用本地回调服务器）
func (s *BackupService) StartOneDriveAuth() (string, error) {
	code, err := onedrive.StartOneDriveAuthServer(s.ctx, 5*time.Minute)
	if err != nil {
		applog.LogErrorf(s.ctx, "StartOneDriveAuth: failed to get auth code: %v", err)
		return "", err
	}
	tokenResp, err := onedrive.ExchangeOneDriveCodeForToken(s.ctx, s.config.OneDriveClientID, code)
	if err != nil {
		applog.LogErrorf(s.ctx, "StartOneDriveAuth: failed to exchange code for token: %v", err)
		return "", err
	}
	return tokenResp.RefreshToken, nil
}

// ExchangeOneDriveCode 用授权码换取 OneDrive token
func (s *BackupService) ExchangeOneDriveCode(code string) (string, error) {
	tokenResp, err := onedrive.ExchangeOneDriveCodeForToken(s.ctx, s.config.OneDriveClientID, code)
	if err != nil {
		applog.LogErrorf(s.ctx, "ExchangeOneDriveCode: failed to exchange code for token: %v", err)
		return "", err
	}
	return tokenResp.RefreshToken, nil
}

// GetCloudBackupStatus 获取云备份状态
func (s *BackupService) GetCloudBackupStatus() vo.CloudBackupStatus {
	return vo.CloudBackupStatus{
		Enabled:    s.config.CloudBackupEnabled,
		Configured: cloudprovider.IsConfigured(s.config),
		UserID:     s.config.BackupUserID,
		Provider:   s.config.CloudBackupProvider,
	}
}

// ========== 本地备份目录相关方法 ==========

// GetBackupDir 获取备份根目录
func (s *BackupService) GetBackupDir() (string, error) {
	return utils.GetSubDir("backups")
}

// GetDBBackupDir 获取数据库备份目录
func (s *BackupService) GetDBBackupDir() (string, error) {
	return utils.GetSubDir(filepath.Join("backups", "database"))
}

// GetFullBackupDir 获取全量数据备份目录
func (s *BackupService) GetFullBackupDir() (string, error) {
	return utils.GetSubDir(filepath.Join("backups", "full"))
}

// OpenBackupFolder 打开备份文件夹
func (s *BackupService) OpenBackupFolder(gameID string) error {
	backupDir, err := s.GetBackupDir()
	if err != nil {
		return err
	}
	gameBackupDir := filepath.Join(backupDir, gameID)
	return utils.OpenDirectory(gameBackupDir)
}

func (s *BackupService) OpenFolder(path string) error {
	fmt.Printf("openfolder:%s\n", path)
	if path == "" {
		return nil
	}
	return utils.OpenDirectory(path)
}

// ========== 游戏存档本地备份方法 ==========

// GetGameBackups 获取游戏的备份历史（直接读取文件夹，不使用数据库）
func (s *BackupService) GetGameBackups(gameID string) ([]models.GameBackup, error) {
	backupDir, err := s.GetBackupDir()
	if err != nil {
		return nil, err
	}

	gameBackupDir := filepath.Join(backupDir, gameID)
	entries, err := os.ReadDir(gameBackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.GameBackup{}, nil
		}
		return nil, err
	}

	var backups []models.GameBackup
	for _, entry := range entries {
		// 只处理 .zip 文件，跳过目录（如 pre_restore、cloud_download）
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".zip") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, models.GameBackup{
			Path:      filepath.Join(gameBackupDir, entry.Name()),
			Name:      entry.Name(),
			GameID:    gameID,
			Size:      info.Size(),
			CreatedAt: info.ModTime(), // 使用文件修改时间
		})
	}

	// 按创建时间降序排序
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// CreateBackup 创建游戏存档备份
func (s *BackupService) CreateBackup(gameID string) (*models.GameBackup, error) {
	var savePath string
	err := s.db.QueryRowContext(s.ctx, "SELECT COALESCE(save_path, '') FROM games WHERE id = ?", gameID).Scan(&savePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get game: %w", err)
	}
	if savePath == "" {
		return nil, fmt.Errorf("the save path is not set for this game")
	}
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("the save path is not exist: %s", savePath)
	}

	backupDir, err := s.GetBackupDir()
	if err != nil {
		return nil, err
	}
	gameBackupDir := filepath.Join(backupDir, gameID)
	if err := os.MkdirAll(gameBackupDir, 0755); err != nil {
		return nil, err
	}

	timestamp := time.Now().Format("2006-01-02T15-04-05")
	backupFileName := fmt.Sprintf("%s.zip", timestamp)
	backupPath := filepath.Join(gameBackupDir, backupFileName)

	size, err := utils.ZipFileOrDirectory(savePath, backupPath, false)
	if err != nil {
		return nil, fmt.Errorf("fail to backup: %w", err)
	}

	// 获取文件信息以得到准确的修改时间
	stat, err := os.Stat(backupPath)
	if err != nil {
		return nil, err
	}

	backup := &models.GameBackup{
		Path:      backupPath,
		Name:      backupFileName,
		GameID:    gameID,
		Size:      size,
		CreatedAt: stat.ModTime(),
	}

	s.cleanupOldLocalBackups(gameID)

	return backup, nil
}

// cleanupOldLocalBackups 清理旧的本地游戏备份
func (s *BackupService) cleanupOldLocalBackups(gameID string) {
	retention := s.config.LocalBackupRetention
	if retention <= 0 {
		retention = 20
	}

	backups, err := s.GetGameBackups(gameID)
	if err != nil || len(backups) <= retention {
		return
	}

	for i := retention; i < len(backups); i++ {
		s.DeleteBackup(backups[i].Path)
	}
}

// RestoreBackup 恢复备份到指定时间点（参数改为备份路径）
func (s *BackupService) RestoreBackup(backupPath string) error {
	// 从路径中提取 gameID（路径格式: backups/{gameID}/{timestamp}.zip）
	backupDir, err := s.GetBackupDir()
	if err != nil {
		return err
	}

	// 验证备份路径在合法目录下
	if !strings.HasPrefix(backupPath, backupDir) {
		return fmt.Errorf("无效的备份路径")
	}

	// 从路径提取 gameID
	relPath, _ := filepath.Rel(backupDir, backupPath)
	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) < 2 {
		return fmt.Errorf("无效的备份路径格式")
	}
	gameID := parts[0]

	// 检查备份文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupPath)
	}

	var savePath string
	err = s.db.QueryRowContext(s.ctx, "SELECT COALESCE(save_path, '') FROM games WHERE id = ?", gameID).Scan(&savePath)
	if err != nil || savePath == "" {
		return fmt.Errorf("存档路径未设置")
	}

	// 先备份当前存档（恢复前备份）
	if _, err := os.Stat(savePath); err == nil {
		preRestoreDir := filepath.Join(backupDir, gameID, "pre_restore")
		os.MkdirAll(preRestoreDir, 0755)
		preRestorePath := filepath.Join(preRestoreDir, fmt.Sprintf("%s_before_restore.zip", time.Now().Format("2006-01-02T15-04-05")))
		_, err := utils.ZipFileOrDirectory(savePath, preRestorePath, false)
		if err != nil {
			return err
		}
	}

	// 检查原始存档路径是文件还是目录
	// 根据备份前的路径类型来决定恢复方式
	parentDir := filepath.Dir(savePath)
	if err := os.RemoveAll(savePath); err != nil {
		return fmt.Errorf("删除原存档失败: %w", err)
	}

	// 临时解压目录
	tempDir := filepath.Join(backupDir, gameID, "temp_restore")
	os.RemoveAll(tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 解压到临时目录
	if err := utils.UnzipFile(backupPath, tempDir); err != nil {
		return fmt.Errorf("解压备份失败: %w", err)
	}

	// 检查解压后的内容
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return fmt.Errorf("读取临时目录失败: %w", err)
	}

	// 如果只有一个文件且不是目录，说明备份的是单个文件
	if len(entries) == 1 && !entries[0].IsDir() {
		// 恢复单个文件
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("创建父目录失败: %w", err)
		}
		srcFile := filepath.Join(tempDir, entries[0].Name())
		if err := utils.CopyFile(srcFile, savePath); err != nil {
			return fmt.Errorf("恢复文件失败: %w", err)
		}
	} else {
		// 恢复整个目录
		if err := os.MkdirAll(savePath, 0755); err != nil {
			return fmt.Errorf("创建存档目录失败: %w", err)
		}
		if err := utils.CopyDir(tempDir, savePath); err != nil {
			return fmt.Errorf("恢复目录失败: %w", err)
		}
	}

	return nil
}

// DeleteBackup 删除备份（参数改为备份路径）
func (s *BackupService) DeleteBackup(backupPath string) error {
	backupDir, err := s.GetBackupDir()
	if err != nil {
		return err
	}

	// 验证备份路径在合法目录下
	if !strings.HasPrefix(backupPath, backupDir) {
		return fmt.Errorf("无效的备份路径")
	}

	return os.Remove(backupPath)
}

// ========== 游戏存档云备份方法 ==========

// UploadGameBackupToCloud 上传游戏存档到云端（参数改为 backupPath）
func (s *BackupService) UploadGameBackupToCloud(gameID string, backupPath string) error {
	if s.config.BackupUserID == "" {
		return fmt.Errorf("备份用户 ID 未设置")
	}

	provider, err := s.getCloudProvider()
	if err != nil {
		return err
	}

	// 验证备份文件存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupPath)
	}

	timestamp := time.Now().Format("2006-01-02T15-04-05")
	cloudPath := provider.GetCloudPath(s.config.BackupUserID, fmt.Sprintf("saves/%s/%s.zip", gameID, timestamp))

	// 确保文件夹存在 (OneDrive 需要)
	folderPath := provider.GetCloudPath(s.config.BackupUserID, fmt.Sprintf("saves/%s", gameID))
	provider.EnsureDir(s.ctx, folderPath)

	if err := provider.UploadFile(s.ctx, cloudPath, backupPath); err != nil {
		return fmt.Errorf("上传失败: %w", err)
	}

	// 更新 latest
	latestPath := provider.GetCloudPath(s.config.BackupUserID, fmt.Sprintf("saves/%s/latest.zip", gameID))
	provider.UploadFile(s.ctx, latestPath, backupPath)

	s.cleanupOldCloudBackups(gameID)
	return nil
}

// GetCloudGameBackups 获取云端游戏备份列表
func (s *BackupService) GetCloudGameBackups(gameID string) ([]vo.CloudBackupItem, error) {
	if s.config.BackupUserID == "" {
		return nil, fmt.Errorf("备份用户 ID 未设置")
	}

	provider, err := s.getCloudProvider()
	if err != nil {
		return nil, err
	}

	listPath := provider.GetCloudPath(s.config.BackupUserID, fmt.Sprintf("saves/%s/", gameID))
	fmt.Printf("GetCloudGameBackups:  %s\n", listPath)
	keys, err := provider.ListObjects(s.ctx, listPath)
	if err != nil {
		fmt.Printf("GetCloudGameBackups err:%v\n", err)
		return nil, err
	}

	return s.parseCloudBackupItems(keys, ""), nil
}

// DownloadCloudBackup 从云端下载备份
func (s *BackupService) DownloadCloudBackup(cloudKey string, gameID string) (string, error) {
	provider, err := s.getCloudProvider()
	if err != nil {
		return "", err
	}

	backupDir, err := s.GetBackupDir()
	if err != nil {
		return "", err
	}
	cloudDownloadDir := filepath.Join(backupDir, gameID, "cloud_download")
	os.MkdirAll(cloudDownloadDir, 0755)

	destPath := filepath.Join(cloudDownloadDir, filepath.Base(cloudKey))
	if err := provider.DownloadFile(s.ctx, cloudKey, destPath); err != nil {
		return "", fmt.Errorf("下载失败: %w", err)
	}
	return destPath, nil
}

// RestoreFromCloud 从云端恢复备份
func (s *BackupService) RestoreFromCloud(cloudKey string, gameID string) error {
	localPath, err := s.DownloadCloudBackup(cloudKey, gameID)
	if err != nil {
		return err
	}

	var savePath string
	err = s.db.QueryRowContext(s.ctx, "SELECT COALESCE(save_path, '') FROM games WHERE id = ?", gameID).Scan(&savePath)
	if err != nil || savePath == "" {
		return fmt.Errorf("存档路径未设置")
	}

	// 先备份当前存档
	if _, err := os.Stat(savePath); err == nil {
		backupDir, _ := s.GetBackupDir()
		preRestoreDir := filepath.Join(backupDir, gameID, "pre_restore")
		os.MkdirAll(preRestoreDir, 0755)
		preRestorePath := filepath.Join(preRestoreDir, fmt.Sprintf("%s_before_cloud_restore.zip", time.Now().Format("2006-01-02T15-04-05")))
		_, err := utils.ZipFileOrDirectory(savePath, preRestorePath, false)
		if err != nil {
			return err
		}
	}

	// 获取备份目录用于临时解压
	backupDir, _ := s.GetBackupDir()
	parentDir := filepath.Dir(savePath)
	if err := os.RemoveAll(savePath); err != nil {
		return fmt.Errorf("删除原存档失败: %w", err)
	}

	// 临时解压目录
	tempDir := filepath.Join(backupDir, gameID, "temp_restore")
	os.RemoveAll(tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 解压到临时目录
	if err := utils.UnzipFile(localPath, tempDir); err != nil {
		return fmt.Errorf("解压备份失败: %w", err)
	}

	// 检查解压后的内容
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return fmt.Errorf("读取临时目录失败: %w", err)
	}

	// 如果只有一个文件且不是目录，说明备份的是单个文件
	if len(entries) == 1 && !entries[0].IsDir() {
		// 恢复单个文件
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("创建父目录失败: %w", err)
		}
		srcFile := filepath.Join(tempDir, entries[0].Name())
		if err := utils.CopyFile(srcFile, savePath); err != nil {
			return fmt.Errorf("恢复文件失败: %w", err)
		}
	} else {
		// 恢复整个目录
		if err := os.MkdirAll(savePath, 0755); err != nil {
			return fmt.Errorf("创建存档目录失败: %w", err)
		}
		if err := utils.CopyDir(tempDir, savePath); err != nil {
			return fmt.Errorf("恢复目录失败: %w", err)
		}
	}

	return nil
}

// cleanupOldCloudBackups 清理旧的云端备份
func (s *BackupService) cleanupOldCloudBackups(gameID string) {
	retention := s.config.CloudBackupRetention
	if retention <= 0 {
		retention = 20
	}

	items, err := s.GetCloudGameBackups(gameID)
	if err != nil || len(items) <= retention {
		return
	}

	provider, err := s.getCloudProvider()
	if err != nil {
		return
	}

	for i := retention; i < len(items); i++ {
		provider.DeleteObject(s.ctx, items[i].Key)
	}
}

// 下载存档
func (s *BackupService) DownloadSave(game models.Game, isOverride bool) (string, error) {
	if game.SavePath == "" {
		return "", fmt.Errorf("还没设置存档位置")
	}
	getter := utils.NewSaveInfoGetter()
	return getter.FetchSeiyaSave(game.Name, game.SavePath, isOverride)
}

func (s *BackupService) DownloadFileInDownload(url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("还没设置存档位置")
	}
	getter := utils.NewSaveInfoGetter()
	return getter.DownloadFileInFolder(url)
}

func (s *BackupService) DownloadSaveInDownload(saveLink string, game models.Game) (*utils.SaveResult, error) {
	if saveLink == "" {
		return nil, fmt.Errorf("没存档位置")
	}
	getter := utils.NewSaveInfoGetter()
	return getter.DownloadSaveForGame(game, saveLink, true)
}

func (s *BackupService) FetchGameGuide(game models.Game) (models.GuideContent, error) {
	guide := models.GuideContent{}
	if game.Name == "" {
		return guide, fmt.Errorf("还没设置存档位置")
	}
	getter := utils.NewSaveInfoGetter()
	guide, err := getter.FetchSeiyaGuide(game.SearchName, 0)
	return guide, err
}

// ========== 数据库本地备份方法 ==========

// CreateDBBackup 创建数据库备份（包含 covers 文件夹）
func (s *BackupService) CreateDBBackup() (*vo.DBBackupInfo, error) {
	backupDir, err := s.GetDBBackupDir()
	if err != nil {
		return nil, err
	}

	// dataDir, err := utils.GetDataDir()
	// if err != nil {
	// 	return nil, err
	// }

	timestamp := time.Now().Format("2006-01-02T15-04-05")
	// 创建临时打包目录，包含数据库导出和 covers
	packDir := filepath.Join(backupDir, fmt.Sprintf("pack_%s", timestamp))
	dbExportDir := filepath.Join(packDir, "database")
	// coversDestDir := filepath.Join(packDir, "images")

	if err := os.MkdirAll(dbExportDir, 0755); err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 导出数据库
	exportPath := strings.ReplaceAll(dbExportDir, "\\", "/")
	_, err = s.db.ExecContext(s.ctx, fmt.Sprintf("EXPORT DATABASE '%s'", exportPath))
	if err != nil {
		os.RemoveAll(packDir)
		return nil, fmt.Errorf("导出数据库失败: %w", err)
	}

	// 复制 covers 文件夹（如果存在）
	// coversSourceDir := filepath.Join(dataDir, "images")
	// if _, err := os.Stat(coversSourceDir); err == nil {
	// 	if err := utils.CopyDir(coversSourceDir, coversDestDir); err != nil {
	// 		applog.LogWarningf(s.ctx, "CreateDBBackup: failed to copy covers: %v", err)
	// 		// 封面复制失败不影响整体备份，继续执行
	// 	}
	// }

	// 打包整个目录
	backupFileName := fmt.Sprintf("kaleidobox_%s.zip", timestamp)
	backupPath := filepath.Join(backupDir, backupFileName)

	_, err = utils.ZipDirectory(packDir, backupPath, true)
	os.RemoveAll(packDir)
	if err != nil {
		return nil, fmt.Errorf("压缩备份失败: %w", err)
	}

	stat, err := os.Stat(backupPath)
	if err != nil {
		return nil, err
	}

	s.config.LastDBBackupTime = time.Now().Format(time.RFC3339)
	retention := s.config.LocalDBBackupRetention
	if retention <= 0 {
		retention = 10
	}
	s.cleanupOldDBBackups(retention)

	return &vo.DBBackupInfo{
		Path:      backupPath,
		Name:      backupFileName,
		Size:      stat.Size(),
		CreatedAt: time.Now(),
	}, nil
}

// GetDBBackups 获取数据库备份列表
func (s *BackupService) GetDBBackups() (*vo.DBBackupStatus, error) {
	backupDir, err := s.GetDBBackupDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, err
	}

	var backups []vo.DBBackupInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".zip") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, vo.DBBackupInfo{
			Path:      filepath.Join(backupDir, entry.Name()),
			Name:      entry.Name(),
			Size:      info.Size(),
			CreatedAt: info.ModTime(), // 使用本地时间
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return &vo.DBBackupStatus{
		LastBackupTime: s.config.LastDBBackupTime,
		Backups:        backups,
	}, nil
}

// ScheduleDBRestore 安排数据库恢复（下次启动时执行）
func (s *BackupService) ScheduleDBRestore(backupPath string) error {
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupPath)
	}
	s.config.PendingDBRestore = backupPath
	return nil
}

// DeleteDBBackup 删除数据库备份
func (s *BackupService) DeleteDBBackup(backupPath string) error {
	backupDir, err := s.GetDBBackupDir()
	if err != nil {
		return err
	}
	if !strings.HasPrefix(backupPath, backupDir) {
		return fmt.Errorf("无效的备份路径")
	}
	return os.Remove(backupPath)
}

// cleanupOldDBBackups 清理旧的数据库备份
func (s *BackupService) cleanupOldDBBackups(retention int) {
	status, err := s.GetDBBackups()
	if err != nil || len(status.Backups) <= retention {
		return
	}
	for i := retention; i < len(status.Backups); i++ {
		os.Remove(status.Backups[i].Path)
	}
}

// ========== 全量数据本地备份方法 ==========

// CreateFullDataBackup 创建全量数据备份（数据库 + 应用设置 + 数据目录）
// savePath: 用户选择的保存路径（完整的 .zip 文件路径）
func (s *BackupService) CreateFullDataBackup(savePath string) error {
	if savePath == "" {
		return fmt.Errorf("保存路径不能为空")
	}
	if !strings.HasSuffix(strings.ToLower(savePath), ".zip") {
		return fmt.Errorf("备份文件必须是 .zip 格式")
	}

	dataDir, err := utils.GetDataDir()
	if err != nil {
		return err
	}
	configDir, err := utils.GetConfigDir()
	if err != nil {
		return err
	}

	// 创建临时打包目录
	tempDir, err := os.MkdirTemp("", "kaleidobox_full_backup_*")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	packDir := filepath.Join(tempDir, "pack")
	dbExportDir := filepath.Join(packDir, "database")

	if err := os.MkdirAll(dbExportDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 先用 DuckDB EXPORT 获取一致性的数据库快照
	exportPath := strings.ReplaceAll(dbExportDir, "\\", "/")
	_, err = s.db.ExecContext(s.ctx, fmt.Sprintf("EXPORT DATABASE '%s'", exportPath))
	if err != nil {
		return fmt.Errorf("导出数据库失败: %w", err)
	}

	// 复制配置文件
	configPath := filepath.Join(configDir, "appconf.json")
	if _, err := os.Stat(configPath); err == nil {
		if err := utils.CopyFile(configPath, filepath.Join(packDir, "appconf.json")); err != nil {
			return fmt.Errorf("复制配置文件失败: %w", err)
		}
	}

	// 复制关键数据目录
	for _, dirName := range []string{"images", "backgrounds", "logs"} {
		srcDir := filepath.Join(dataDir, dirName)
		if _, err := os.Stat(srcDir); err != nil {
			continue
		}
		if err := utils.CopyDir(srcDir, filepath.Join(packDir, dirName)); err != nil {
			return fmt.Errorf("复制目录 %s 失败: %w", dirName, err)
		}
	}

	// 复制 backups 目录（包含游戏存档备份和数据库备份）
	backupsSourceDir := filepath.Join(dataDir, "backups")
	if _, err := os.Stat(backupsSourceDir); err == nil {
		backupsDestDir := filepath.Join(packDir, "backups")
		if err := os.MkdirAll(backupsDestDir, 0755); err != nil {
			return fmt.Errorf("创建备份目录失败: %w", err)
		}
		if err := utils.CopyDir(backupsSourceDir, backupsDestDir); err != nil {
			return fmt.Errorf("复制备份目录失败: %w", err)
		}
	}

	// 打包到用户指定的路径
	_, err = utils.ZipDirectory(packDir, savePath, true)
	if err != nil {
		return fmt.Errorf("压缩全量备份失败: %w", err)
	}

	s.config.LastFullBackupTime = time.Now().Format(time.RFC3339)
	return nil
}

// ScheduleFullDataRestore 安排全量数据恢复（下次启动时执行）
// backupPath: 用户选择的备份文件完整路径
func (s *BackupService) ScheduleFullDataRestore(backupPath string) error {
	if backupPath == "" {
		return fmt.Errorf("备份路径不能为空")
	}
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupPath)
	}
	if !strings.HasSuffix(strings.ToLower(backupPath), ".zip") {
		return fmt.Errorf("备份文件必须是 .zip 格式")
	}

	// 全量恢复包含数据库，清理数据库待恢复任务，避免重复恢复
	s.config.PendingDBRestore = ""
	s.config.PendingFullRestore = backupPath
	return nil
}

// ========== 数据库云备份方法 ==========

// UploadDBBackupToCloud 上传数据库备份到云端
func (s *BackupService) UploadDBBackupToCloud(backupPath string) error {
	if s.config.BackupUserID == "" {
		return fmt.Errorf("备份用户 ID 未设置")
	}

	provider, err := s.getCloudProvider()
	if err != nil {
		return err
	}

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupPath)
	}

	fileName := filepath.Base(backupPath)
	cloudKey := provider.GetCloudPath(s.config.BackupUserID, fmt.Sprintf("database/%s", fileName))

	folderPath := provider.GetCloudPath(s.config.BackupUserID, "database")
	provider.EnsureDir(s.ctx, folderPath)

	if err := provider.UploadFile(s.ctx, cloudKey, backupPath); err != nil {
		return fmt.Errorf("上传失败: %w", err)
	}

	latestKey := provider.GetCloudPath(s.config.BackupUserID, "database/latest.zip")
	provider.UploadFile(s.ctx, latestKey, backupPath)

	s.cleanupOldCloudDBBackups()
	return nil
}

// GetCloudDBBackups 获取云端数据库备份列表
func (s *BackupService) GetCloudDBBackups() ([]vo.CloudBackupItem, error) {
	if s.config.BackupUserID == "" {
		return nil, fmt.Errorf("备份用户 ID 未设置")
	}

	provider, err := s.getCloudProvider()
	if err != nil {
		return nil, err
	}

	prefix := provider.GetCloudPath(s.config.BackupUserID, "database/")
	keys, err := provider.ListObjects(s.ctx, prefix)
	if err != nil {
		return nil, err
	}

	return s.parseCloudBackupItems(keys, "kaleidobox_"), nil
}

// DownloadCloudDBBackup 从云端下载数据库备份
// TODO: 前端提供此功能的按钮
func (s *BackupService) DownloadCloudDBBackup(cloudKey string) (string, error) {
	provider, err := s.getCloudProvider()
	if err != nil {
		return "", err
	}

	backupDir, err := s.GetDBBackupDir()
	if err != nil {
		return "", err
	}
	cloudDownloadDir := filepath.Join(backupDir, "cloud_download")
	os.MkdirAll(cloudDownloadDir, 0755)

	destPath := filepath.Join(cloudDownloadDir, filepath.Base(cloudKey))
	if err := provider.DownloadFile(s.ctx, cloudKey, destPath); err != nil {
		return "", fmt.Errorf("下载失败: %w", err)
	}
	return destPath, nil
}

// ScheduleDBRestoreFromCloud 从云端下载并安排数据库恢复
func (s *BackupService) ScheduleDBRestoreFromCloud(cloudKey string) error {
	localPath, err := s.DownloadCloudDBBackup(cloudKey)
	if err != nil {
		return err
	}
	return s.ScheduleDBRestore(localPath)
}

// cleanupOldCloudDBBackups 清理旧的云端数据库备份
func (s *BackupService) cleanupOldCloudDBBackups() {
	retention := s.config.CloudBackupRetention
	if retention <= 0 {
		retention = 10
	}

	items, err := s.GetCloudDBBackups()
	if err != nil || len(items) <= retention {
		return
	}

	provider, err := s.getCloudProvider()
	if err != nil {
		return
	}

	for i := retention; i < len(items); i++ {
		provider.DeleteObject(s.ctx, items[i].Key)
	}
}

// CreateAndUploadDBBackup 创建数据库备份并上传到云端
func (s *BackupService) CreateAndUploadDBBackup() (*vo.DBBackupInfo, error) {
	backup, err := s.CreateDBBackup()
	if err != nil {
		return nil, err
	}

	// 只有在启用云备份、配置完整且开启数据库自动上传时才上传
	if s.config.CloudBackupEnabled && s.config.BackupUserID != "" && s.config.AutoUploadDBToCloud {
		if err := s.UploadDBBackupToCloud(backup.Path); err != nil {
			return backup, fmt.Errorf("本地备份成功，但云端上传失败: %w", err)
		}
	}
	return backup, nil
}

// ========== 辅助方法 ==========

// parseCloudBackupItems 解析云端备份列表
func (s *BackupService) parseCloudBackupItems(keys []string, prefix string) []vo.CloudBackupItem {
	var items []vo.CloudBackupItem
	for _, key := range keys {
		if strings.HasSuffix(key, "latest.zip") {
			continue
		}
		name := filepath.Base(key)
		displayName := name
		name = strings.TrimPrefix(name, prefix)
		name = strings.TrimSuffix(name, ".zip")
		t, _ := time.Parse("2006-01-02T15-04-05", name)

		items = append(items, vo.CloudBackupItem{
			Key:       key,
			Name:      displayName,
			CreatedAt: t,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items
}

// ========== 全量数据恢复（启动时调用）==========

// ExecuteFullDataRestore 执行全量数据恢复（在 OnStartup 中、打开数据库前调用）
func ExecuteFullDataRestore(config *appconf.AppConfig) (bool, error) {
	if config.PendingFullRestore == "" {
		return false, nil
	}

	backupPath := config.PendingFullRestore
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		config.PendingFullRestore = ""
		appconf.SaveConfig(config)
		return false, fmt.Errorf("备份文件不存在: %s", backupPath)
	}

	dataDir, err := utils.GetDataDir()
	if err != nil {
		return false, err
	}
	configDir, err := utils.GetConfigDir()
	if err != nil {
		return false, err
	}

	tempDir, err := os.MkdirTemp("", "lunabox_full_restore_*")
	if err != nil {
		return false, fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := utils.UnzipForRestore(backupPath, tempDir); err != nil {
		return false, fmt.Errorf("解压全量备份失败: %w", err)
	}

	// 先恢复数据库
	dbPath := filepath.Join(dataDir, "kaleidobox.db")
	dbImportDir := filepath.Join(tempDir, "database")
	rawDBPath := filepath.Join(tempDir, "kaleidobox.db")

	os.Remove(dbPath)
	os.Remove(dbPath + ".wal")

	if _, err := os.Stat(dbImportDir); err == nil {
		db, err := sql.Open("duckdb", dbPath)
		if err != nil {
			return false, fmt.Errorf("打开数据库失败: %w", err)
		}

		importPath := strings.ReplaceAll(dbImportDir, "\\", "/")
		_, err = db.Exec(fmt.Sprintf("IMPORT DATABASE '%s'", importPath))
		db.Close()
		if err != nil {
			return false, fmt.Errorf("导入数据库失败: %w", err)
		}
	} else if _, err := os.Stat(rawDBPath); err == nil {
		if err := utils.CopyFile(rawDBPath, dbPath); err != nil {
			return false, fmt.Errorf("恢复数据库文件失败: %w", err)
		}
	} else {
		return false, fmt.Errorf("全量备份中缺少数据库内容")
	}

	// 恢复应用数据目录
	for _, dirName := range []string{"covers", "backgrounds", "logs", "backups"} {
		srcDir := filepath.Join(tempDir, dirName)
		if _, err := os.Stat(srcDir); err != nil {
			continue
		}

		dstDir := filepath.Join(dataDir, dirName)
		if err := os.RemoveAll(dstDir); err != nil {
			return false, fmt.Errorf("清理目录 %s 失败: %w", dirName, err)
		}
		if err := utils.CopyDir(srcDir, dstDir); err != nil {
			return false, fmt.Errorf("恢复目录 %s 失败: %w", dirName, err)
		}
	}

	// 恢复配置文件
	backupConfigPath := filepath.Join(tempDir, "appconf.json")
	if _, err := os.Stat(backupConfigPath); err == nil {
		configPath := filepath.Join(configDir, "appconf.json")
		if err := utils.CopyFile(backupConfigPath, configPath); err != nil {
			return false, fmt.Errorf("恢复配置文件失败: %w", err)
		}
	}

	// 重新加载配置并清理待恢复标记，避免重复执行
	restoredConfig, err := appconf.LoadConfig()
	if err != nil {
		restoredConfig = config
	}
	restoredConfig.PendingFullRestore = ""
	restoredConfig.PendingDBRestore = ""
	if err := appconf.SaveConfig(restoredConfig); err != nil {
		return false, fmt.Errorf("保存恢复后配置失败: %w", err)
	}
	*config = *restoredConfig

	return true, nil
}

// ========== 数据库恢复（启动时调用）==========

// 字段类型映射
var fieldTypeMap map[string]map[string]string

// initFieldTypeMap 初始化字段类型映射
func initFieldTypeMap() {
	fieldTypeMap = make(map[string]map[string]string)

	// 解析 SchemaQueries() 中的表结构
	queries := migrations.SchemaQueries()
	for _, query := range queries {
		// 提取表名
		tableName := extractTableName(query)
		if tableName == "" {
			continue
		}

		// 提取字段信息
		fields := extractFields(query)
		fieldTypeMap[tableName] = fields
	}
}

// extractTableName 从 CREATE TABLE 语句中提取表名
func extractTableName(query string) string {
	// 简单的正则匹配，提取表名
	re := regexp.MustCompile(`CREATE TABLE IF NOT EXISTS\s+(\w+)\s*\(`)
	matches := re.FindStringSubmatch(query)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractFields 从 CREATE TABLE 语句中提取字段信息
func extractFields(query string) map[string]string {
	fields := make(map[string]string)

	// 提取括号内的内容
	re := regexp.MustCompile(`\(([\s\S]*?)\)\s*$`)
	matches := re.FindStringSubmatch(query)
	if len(matches) <= 1 {
		return fields
	}

	// 分割字段定义
	fieldDefs := strings.Split(matches[1], ",")
	for _, def := range fieldDefs {
		def = strings.TrimSpace(def)
		if def == "" || strings.HasPrefix(def, "PRIMARY KEY") {
			continue
		}

		// 提取字段名和类型
		parts := strings.Fields(def)
		if len(parts) >= 2 {
			fieldName := parts[0]
			fieldType := parts[1]
			// 处理复杂类型，如 TIMESTAMPTZ
			if len(parts) > 2 && strings.HasSuffix(parts[1], "TZ") {
				fieldType = parts[1] + " " + parts[2]
			}
			fields[fieldName] = fieldType
		}
	}

	return fields
}

// migrateData 将数据从临时数据库迁移到新数据库
func migrateData(tempDB, newDB *sql.DB) error {
	// 初始化字段类型映射
	initFieldTypeMap()

	// 迁移 users 表
	if err := migrateTable(tempDB, newDB, "users"); err != nil {
		return err
	}

	// 迁移 categories 表
	if err := migrateTable(tempDB, newDB, "categories"); err != nil {
		return err
	}

	// 迁移 games 表
	if err := migrateTable(tempDB, newDB, "games"); err != nil {
		return err
	}

	// 迁移 game_categories 表
	if err := migrateTable(tempDB, newDB, "game_categories"); err != nil {
		return err
	}

	// 迁移 play_sessions 表
	if err := migrateTable(tempDB, newDB, "play_sessions"); err != nil {
		return err
	}

	// 迁移 tasks 表
	if err := migrateTable(tempDB, newDB, "tasks"); err != nil {
		return err
	}

	// 迁移 charactors 表
	if err := migrateTable(tempDB, newDB, "charactors"); err != nil {
		return err
	}

	// 迁移 staffs 表
	if err := migrateTable(tempDB, newDB, "staffs"); err != nil {
		return err
	}

	// 迁移 works 表
	if err := migrateTable(tempDB, newDB, "works"); err != nil {
		return err
	}

	// 迁移 tags 表
	if err := migrateTable(tempDB, newDB, "tags"); err != nil {
		return err
	}

	// 迁移 image_backups 表
	if err := migrateTable(tempDB, newDB, "image_backups"); err != nil {
		return err
	}

	// 迁移 hotkeys 表
	if err := migrateTable(tempDB, newDB, "hotkeys"); err != nil {
		return err
	}

	// 迁移 connected_devices 表
	if err := migrateTable(tempDB, newDB, "connected_devices"); err != nil {
		return err
	}

	return nil
}

// migrateTable 迁移单个表的数据
func migrateTable(tempDB, newDB *sql.DB, tableName string) error {
	// 获取临时表的列信息
	tempColumns, err := getTableColumns(tempDB, tableName)
	if err != nil {
		// 表不存在，跳过
		fmt.Printf("迁移表 %s: 临时表不存在或查询失败\n", tableName)
		return nil
	}

	// 获取新表的列信息
	newColumns, err := getTableColumns(newDB, tableName)
	if err != nil {
		return fmt.Errorf("获取新表 %s 列信息失败: %w", tableName, err)
	}

	// 找出两个表共有的列
	commonColumns := []string{}
	for _, col := range tempColumns {
		for _, newCol := range newColumns {
			if col == newCol {
				commonColumns = append(commonColumns, col)
				break
			}
		}
	}

	if len(commonColumns) == 0 {
		// 没有共同列，跳过
		fmt.Printf("迁移表 %s: 没有共同列\n", tableName)
		return nil
	}

	// 构建列名字符串
	columnsStr := strings.Join(commonColumns, ", ")
	placeholders := make([]string, len(commonColumns))
	for i := range commonColumns {
		placeholders[i] = "?"
	}
	placeholdersStr := strings.Join(placeholders, ", ")

	// 构建查询语句
	selectQuery := fmt.Sprintf("SELECT %s FROM %s", columnsStr, tableName)
	insertQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableName, columnsStr, placeholdersStr)

	fmt.Printf("迁移表 %s: 开始迁移，共有 %d 个共同列\n", tableName, len(commonColumns))

	// 执行查询
	rows, err := tempDB.Query(selectQuery)
	if err != nil {
		return fmt.Errorf("查询临时表 %s 数据失败: %w", tableName, err)
	}
	defer rows.Close()

	// 准备插入语句
	stmt, err := newDB.Prepare(insertQuery)
	if err != nil {
		return fmt.Errorf("准备插入语句失败: %w", err)
	}
	defer stmt.Close()

	// 遍历结果并插入
	count := 0
	for rows.Next() {
		// 动态创建参数切片
		values := make([]interface{}, len(commonColumns))
		valuePtrs := make([]interface{}, len(commonColumns))

		for i := range commonColumns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("扫描数据失败: %w", err)
		}

		// 处理 NULL 值，根据字段类型设置默认值
		for i, value := range values {
			if value == nil {
				columnName := commonColumns[i]
				// 从字段类型映射中获取字段类型
				fieldType := ""
				if tableFields, ok := fieldTypeMap[tableName]; ok {
					if ft, ok := tableFields[columnName]; ok {
						fieldType = ft
					}
				}

				// 根据字段类型设置默认值
				switch {
				case strings.Contains(strings.ToLower(fieldType), "integer"):
					// 整数类型，设置默认值 0
					values[i] = 0
				case strings.Contains(strings.ToLower(fieldType), "text"):
					// 文本类型，设置默认值 ""
					values[i] = ""
				case strings.Contains(strings.ToLower(fieldType), "boolean"):
					// 布尔类型，设置默认值 false
					values[i] = false
				case strings.Contains(strings.ToLower(fieldType), "timestamp"):
					// 时间戳类型，设置默认值为 nil（保持 NULL）
					values[i] = nil
				default:
					// 其他类型，默认设置为空字符串
					values[i] = ""
				}
			}
		}

		_, err := stmt.Exec(values...)
		if err != nil {
			return fmt.Errorf("插入数据失败: %w", err)
		}

		count++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("遍历结果失败: %w", err)
	}

	fmt.Printf("迁移表 %s: 成功迁移 %d 条记录\n", tableName, count)

	return nil
}

// getTableColumns 获取表的列名
func getTableColumns(db *sql.DB, tableName string) ([]string, error) {
	// 先检查表是否存在
	tableExists := false
	err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", tableName).Scan(&tableExists)
	if err != nil {
		// 尝试另一种方式检查表是否存在
		_, err := db.Exec(fmt.Sprintf("SELECT 1 FROM %s LIMIT 1", tableName))
		if err != nil {
			return nil, fmt.Errorf("表 %s 不存在或无法访问: %w", tableName, err)
		}
	}

	// 尝试使用 PRAGMA table_info 获取列信息
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		// 如果 PRAGMA 失败，尝试使用 SELECT * 来获取列名
		return getColumnsFromSelect(db, tableName)
	}
	defer rows.Close()

	columns := []string{}
	for rows.Next() {
		var cid int
		var name string
		var typeStr string
		var notnull int
		var dfltValue interface{}
		var pk int

		if err := rows.Scan(&cid, &name, &typeStr, &notnull, &dfltValue, &pk); err != nil {
			rows.Close()
			// 扫描失败，尝试使用 SELECT * 来获取列名
			return getColumnsFromSelect(db, tableName)
		}

		columns = append(columns, name)
	}

	if err := rows.Err(); err != nil {
		// 遍历失败，尝试使用 SELECT * 来获取列名
		return getColumnsFromSelect(db, tableName)
	}

	if len(columns) == 0 {
		// 没有获取到列信息，尝试使用 SELECT * 来获取列名
		return getColumnsFromSelect(db, tableName)
	}

	return columns, nil
}

// getColumnsFromSelect 通过执行 SELECT * 来获取表的列名
func getColumnsFromSelect(db *sql.DB, tableName string) ([]string, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT 1", tableName))
	if err != nil {
		return nil, fmt.Errorf("查询表 %s 失败: %w", tableName, err)
	}
	defer rows.Close()

	schema, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取表 %s 列信息失败: %w", tableName, err)
	}

	return schema, nil
}

// ExecuteDBRestore 执行数据库恢复（在 OnStartup 中、打开数据库前调用）
// 支持新格式（包含 database/ 和 covers/ 子目录）和旧格式（直接是数据库导出文件）
func ExecuteDBRestore(config *appconf.AppConfig) (bool, error) {
	if config.PendingDBRestore == "" {
		return false, nil
	}
	fmt.Printf("ExecuteDBRestore start\n")

	backupPath := config.PendingDBRestore

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		config.PendingDBRestore = ""
		appconf.SaveConfig(config)
		return false, fmt.Errorf("备份文件不存在: %s", backupPath)
	}

	fmt.Printf("ExecuteDBRestore start 01 \n")

	dataDir, err := utils.GetDataDir()
	if err != nil {
		return false, err
	}
	dbPath := filepath.Join(dataDir, "kaleidobox.db")
	fmt.Printf("ExecuteDBRestore start 02 \n")

	tempDir := filepath.Join(dataDir, "backups", "database", "restore_temp")
	os.RemoveAll(tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return false, fmt.Errorf("创建临时目录失败: %w", err)
	}
	fmt.Printf("ExecuteDBRestore start 03 \n")

	if err := utils.UnzipForRestore(backupPath, tempDir); err != nil {
		os.RemoveAll(tempDir)
		return false, fmt.Errorf("解压备份失败: %w", err)
	}

	fmt.Printf("ExecuteDBRestore start 04 \n")

	// 检测备份格式：新格式有 database/ 子目录，旧格式直接是数据库文件
	dbImportDir := tempDir
	coversBackupDir := ""
	if _, err := os.Stat(filepath.Join(tempDir, "database")); err == nil {
		// 新格式
		dbImportDir = filepath.Join(tempDir, "database")
		coversBackupDir = filepath.Join(tempDir, "images")
	}

	os.Remove(dbPath)
	os.Remove(dbPath + ".wal")

	// 步骤1：创建临时数据库来读取备份数据
	tempDBPath := filepath.Join(tempDir, "temp_backup.db")
	os.Remove(tempDBPath)
	os.Remove(tempDBPath + ".wal")

	tempDB, err := sql.Open("duckdb", tempDBPath)
	if err != nil {
		os.RemoveAll(tempDir)
		return false, fmt.Errorf("打开临时数据库失败: %w", err)
	}

	fmt.Printf("ExecuteDBRestore start 07 \n")

	importPath := strings.ReplaceAll(dbImportDir, "\\", "/")
	_, err = tempDB.Exec(fmt.Sprintf("IMPORT DATABASE '%s'", importPath))
	if err != nil {
		tempDB.Close()
		os.RemoveAll(tempDir)
		return false, fmt.Errorf("导入备份到临时数据库失败: %w", err)
	}

	// 日志：临时数据库导入后的数据量
	fmt.Println("=== 临时数据库导入后的数据量 ===")
	tableNames := []string{"users", "categories", "games", "game_categories", "play_sessions", "tasks", "charactors", "staffs", "works", "tags", "image_backups", "hotkeys", "connected_devices"}
	for _, table := range tableNames {
		var count int
		err := tempDB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err == nil {
			fmt.Printf("临时数据库表 %s: %d 条记录\n", table, count)
		} else {
			fmt.Printf("临时数据库表 %s: 不存在或查询失败\n", table)
		}
	}

	fmt.Printf("ExecuteDBRestore start 08 \n")

	// 步骤2：创建新的数据库结构
	newDB, err := sql.Open("duckdb", dbPath)
	if err != nil {
		tempDB.Close()
		os.RemoveAll(tempDir)
		return false, fmt.Errorf("打开新数据库失败: %w", err)
	}

	fmt.Printf("ExecuteDBRestore start 09 \n")

	// 初始化新的表结构
	if err := migrations.InitSchema(newDB); err != nil {
		tempDB.Close()
		newDB.Close()
		os.RemoveAll(tempDir)
		return false, fmt.Errorf("初始化新数据库结构失败: %w", err)
	}

	fmt.Printf("ExecuteDBRestore start 10 \n")

	// 步骤3：迁移数据
	if err := migrateData(tempDB, newDB); err != nil {
		tempDB.Close()
		newDB.Close()
		os.RemoveAll(tempDir)
		return false, fmt.Errorf("迁移数据失败: %w", err)
	}

	// 日志：正式数据库迁移后的数据量
	fmt.Println("=== 正式数据库迁移后的数据量 ===")
	for _, table := range tableNames {
		var count int
		err := newDB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err == nil {
			fmt.Printf("正式数据库表 %s: %d 条记录\n", table, count)
		} else {
			fmt.Printf("正式数据库表 %s: 查询失败\n", table)
		}
	}

	fmt.Printf("ExecuteDBRestore start 11 \n")

	// 关闭数据库
	tempDB.Close()
	newDB.Close()

	// 恢复 covers 文件夹（如果备份中包含）
	if coversBackupDir != "" {
		if _, err := os.Stat(coversBackupDir); err == nil {
			coversDestDir := filepath.Join(dataDir, "images")
			// 先清空现有 covers 目录
			os.RemoveAll(coversDestDir)
			if err := utils.CopyDir(coversBackupDir, coversDestDir); err != nil {
				// 封面恢复失败不影响整体恢复，只记录警告
				fmt.Printf("警告: 恢复封面图片失败: %v\n", err)
			}
		}
	}
	fmt.Printf("ExecuteDBRestore start 13 \n")

	os.RemoveAll(tempDir)

	config.PendingDBRestore = ""
	appconf.SaveConfig(config)
	return true, nil
}
