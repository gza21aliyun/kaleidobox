package main

import (
	"context"
	"database/sql"
	"embed"
	"encoding/base64"
	"fmt"
	"io"
	"lunabox/internal/applog"
	"lunabox/internal/utils"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	goruntime "runtime"

	"sync/atomic"

	"lunabox/internal/appconf"
	"lunabox/internal/enums"
	"lunabox/internal/migrations"
	"lunabox/internal/service"

	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/energye/systray"

	_ "github.com/duckdb/duckdb-go/v2"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var icon []byte

var db *sql.DB

var config *appconf.AppConfig

var appState = newLifecycleState()
var ipcHTTPServer *http.Server

type lifecycleState struct {
	ctxMu sync.RWMutex
	ctx   context.Context

	forceQuit    atomic.Bool
	shuttingDown atomic.Bool

	trayReady     chan struct{}
	trayReadyOnce sync.Once
	trayExit      chan struct{}
	trayExitOnce  sync.Once
	trayQuitOnce  sync.Once
}

func newLifecycleState() *lifecycleState {
	return &lifecycleState{
		trayReady: make(chan struct{}),
		trayExit:  make(chan struct{}),
	}
}

func (s *lifecycleState) SetContext(ctx context.Context) {
	s.ctxMu.Lock()
	defer s.ctxMu.Unlock()
	s.ctx = ctx
}

func (s *lifecycleState) Context() context.Context {
	s.ctxMu.RLock()
	defer s.ctxMu.RUnlock()
	return s.ctx
}

func (s *lifecycleState) MarkTrayReady() {
	s.trayReadyOnce.Do(func() {
		close(s.trayReady)
	})
}

func (s *lifecycleState) MarkTrayExit() {
	s.trayExitOnce.Do(func() {
		close(s.trayExit)
	})
}

func (s *lifecycleState) ShouldForceQuit() bool {
	return s.forceQuit.Load() || s.shuttingDown.Load()
}

func (s *lifecycleState) BeginShutdown() {
	s.shuttingDown.Store(true)
}

func (s *lifecycleState) ShowMainWindow() {
	if s.shuttingDown.Load() {
		return
	}

	ctx := s.Context()
	if ctx == nil {
		return
	}

	runtime.WindowShow(ctx)
}

func (s *lifecycleState) QuitApplication() {
	if s.shuttingDown.Load() {
		return
	}

	ctx := s.Context()
	if ctx == nil {
		return
	}

	s.forceQuit.Store(true)
	s.shuttingDown.Store(true)
	s.RequestTrayQuit()
	runtime.Quit(ctx)
}

func (s *lifecycleState) StartTray() {
	go func() {
		goruntime.LockOSThread()
		defer goruntime.UnlockOSThread()
		systray.Run(onSystrayReady, onSystrayExit)
	}()
}

func (s *lifecycleState) RequestTrayQuit() {
	s.trayQuitOnce.Do(func() {
		systray.Quit()
	})
}

func (s *lifecycleState) WaitForTrayExit(timeout time.Duration) bool {
	select {
	case <-s.trayExit:
		return true
	case <-time.After(timeout):
		return false
	}
}

func main() {
	logDir, _ := utils.GetSubDir("logs")
	logFilePath := filepath.Join(logDir, "app.log")
	appLogger := logger.NewFileLogger(logFilePath)
	// 设置 applog 包的日志文件路径
	applog.SetLogFilePath(logFilePath)

	var loadErr error
	config, loadErr = appconf.LoadConfig()
	if loadErr != nil {
		appLogger.Fatal(loadErr.Error())
	}

	gameService := service.NewGameService()
	aiService := service.NewAiService()
	backupService := service.NewBackupService()
	homeService := service.NewHomeService()
	statsService := service.NewStatsService()
	startService := service.NewStartService()
	categoryService := service.NewCategoryService()
	configService := service.NewConfigService()
	importService := service.NewImportService()
	versionService := service.NewVersionService()
	templateService := service.NewTemplateService()
	updateService := service.NewUpdateService()
	sessionService := service.NewSessionService()
	taskService := service.NewTaskService() // 添加任务服务
	staffService := service.NewStaffService()
	charactorService := service.NewCharactorService()
	workService := service.NewWorkService()
	tagService := service.NewTagService()
	imageService := service.NewImageService()
	hotkeyService := service.NewHotkeyService()
	touchMappingService := service.NewTouchMappingService()
	i18nService := service.NewI18nService()
	vmService := service.NewVMService()

	// 创建本地文件处理器
	localFileHandler, err := utils.NewLocalFileHandler()
	if err != nil {
		appLogger.Error("Warning: Failed to create local file handler: " + err.Error())
	}

	// Create application with options
	// 使用配置中保存的窗口尺寸，如果小于最小值则使用最小值
	initWidth := config.WindowWidth
	if initWidth < 970 {
		initWidth = 970
	}
	initHeight := config.WindowHeight
	if initHeight < 563 {
		initHeight = 563
	}

	bootstrapErr := wails.Run(&options.App{
		Title:     "KaleidoBox",
		Logger:    appLogger,
		LogLevel:  logger.INFO,
		Width:     initWidth,
		Height:    initHeight,
		MinWidth:  970,
		MinHeight: 563,
		AssetServer: &assetserver.Options{
			Assets: assets,
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// 跨域处理
					w.Header().Set("Access-Control-Allow-Origin", "*")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

					if r.Method == "OPTIONS" {
						w.WriteHeader(http.StatusOK)
						return
					}

					if strings.HasPrefix(r.URL.Path, "/local/") {
						localFileHandler.ServeHTTP(w, r)
						return
					}

					// 处理视频缓存清理请求
					if strings.HasPrefix(r.URL.Path, "/api/video/cleanup/") {
						gameID := strings.TrimPrefix(r.URL.Path, "/api/video/cleanup/")
						applog.LogInfof(appState.ctx, "VideoCleanupHandler: received request for gameID: %s", gameID)

						// 先终止相关的ffmpeg进程
						ffmpegProcessMap := utils.GetFFmpegProcessMap()
						ffmpegProcessMutex := utils.GetFFmpegProcessMutex()
						ffmpegProcessMutex.Lock()
						// 如果提供了gameID，终止对应的进程
						if gameID != "" {
							if cmd, exists := ffmpegProcessMap[gameID]; exists {
								applog.LogInfof(appState.ctx, "VideoCleanupHandler: terminating ffmpeg process for gameID: %s", gameID)
								if err := cmd.Process.Kill(); err != nil {
									applog.LogErrorf(appState.ctx, "VideoCleanupHandler: failed to kill ffmpeg process: %v", err)
								} else {
									applog.LogInfof(appState.ctx, "VideoCleanupHandler: killed ffmpeg process for gameID: %s", gameID)
								}
								// 从映射中删除进程
								delete(ffmpegProcessMap, gameID)
							}
						} else {
							// 如果没有提供gameID，终止所有进程
							for id, cmd := range ffmpegProcessMap {
								applog.LogInfof(appState.ctx, "VideoCleanupHandler: terminating ffmpeg process for id: %s", id)
								if err := cmd.Process.Kill(); err != nil {
									applog.LogErrorf(appState.ctx, "VideoCleanupHandler: failed to kill ffmpeg process: %v", err)
								} else {
									applog.LogInfof(appState.ctx, "VideoCleanupHandler: killed ffmpeg process for id: %s", id)
								}
							}
							// 清空进程映射
							for id := range ffmpegProcessMap {
								delete(ffmpegProcessMap, id)
							}
						}
						ffmpegProcessMutex.Unlock()

						// 清理缓存
						videoCacheMutex.Lock()
						// 清理游戏ID相关的缓存
						cachedVideoPath, exists := videoCacheMap[gameID]
						if exists {
							// 从缓存中删除条目
							delete(videoCacheMap, gameID)
							applog.LogInfof(appState.ctx, "VideoCleanupHandler: removed gameID from cache: %s", gameID)
						}

						// 清理所有视频路径相关的缓存（path_* 格式的键）
						var pathsToDelete []string
						var keysToDelete []string
						for key, path := range videoCacheMap {
							if strings.HasPrefix(key, "path_") {
								pathsToDelete = append(pathsToDelete, path)
								keysToDelete = append(keysToDelete, key)
							}
						}
						for _, key := range keysToDelete {
							delete(videoCacheMap, key)
						}
						videoCacheMutex.Unlock()

						// 尝试删除缓存文件，添加重试机制
						const maxRetries = 3

						// 删除游戏ID相关的缓存文件
						if exists && cachedVideoPath != "" {
							for i := 0; i < maxRetries; i++ {
								if err := os.Remove(cachedVideoPath); err != nil {
									applog.LogErrorf(appState.ctx, "VideoCleanupHandler: attempt %d failed to remove cached video file: %v", i+1, err)
									// 等待一段时间后重试
									time.Sleep(time.Second)
								} else {
									applog.LogInfof(appState.ctx, "VideoCleanupHandler: removed cached video file: %s", cachedVideoPath)
									break
								}
							}
						}

						// 删除视频路径相关的缓存文件
						for _, path := range pathsToDelete {
							for i := 0; i < maxRetries; i++ {
								if err := os.Remove(path); err != nil {
									applog.LogErrorf(appState.ctx, "VideoCleanupHandler: attempt %d failed to remove video path cached file: %v", i+1, err)
									// 等待一段时间后重试
									time.Sleep(time.Second)
								} else {
									applog.LogInfof(appState.ctx, "VideoCleanupHandler: removed video path cached file: %s", path)
									break
								}
							}
						}

						w.WriteHeader(http.StatusOK)
						w.Write([]byte("Video cache cleaned up"))
						return
					}

					// 处理视频流请求（通过游戏ID）
					if strings.HasPrefix(r.URL.Path, "/api/video/") {
						handleVideoStreamRequest(w, r, gameService, appState.ctx)
						return
					}

					// 处理视频流请求（通过直接视频路径）
					if strings.HasPrefix(r.URL.Path, "/api/videoPath/") {
						// 提取 base64 编码的视频路径
						encodedPath := strings.TrimPrefix(r.URL.Path, "/api/videoPath/")
						// 解码视频路径
						videoPath, err := base64.StdEncoding.DecodeString(encodedPath)
						if err != nil {
							applog.LogErrorf(appState.ctx, "VideoPathHandler: failed to decode video path: %v", err)
							http.Error(w, "Invalid video path", http.StatusBadRequest)
							return
						}
						// 调用新的函数处理视频流
						handleVideoPathStreamRequest(w, r, string(videoPath), appState.ctx)
						return
					}

					next.ServeHTTP(w, r)
				})
			},
		},
		BackgroundColour: &options.RGBA{R: 18, G: 20, B: 22, A: 255},
		StartHidden:      true,
		Frameless:        true, // 启用无边框模式
		// 启用拖拽文件导入功能
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
			CSSDropProperty:    "--wails-drop-target",
			CSSDropValue:       "drop",
		},
		// 样式完全交由wails前端控制
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			BackdropType:         windows.Auto,
			Theme:                windows.SystemDefault,
		},
		// 关闭窗口时的处理
		OnBeforeClose: func(ctx context.Context) bool {
			// 保存当前窗口大小（只在非最大化时）
			if !runtime.WindowIsMaximised(ctx) {
				config.WindowWidth, config.WindowHeight = runtime.WindowGetSize(ctx)
			}

			// 如果是从托盘强制退出，直接允许关闭
			if appState.ShouldForceQuit() {
				return false
			}
			if config.CloseToTray {
				runtime.WindowHide(ctx)
				return true
			}
			return false
		},
		OnStartup: func(ctx context.Context) {
			appState.SetContext(ctx)
			var err error

			// 检查是否有待恢复的全量数据备份（在打开数据库前执行）
			if config.PendingFullRestore != "" {
				restored, restoreErr := service.ExecuteFullDataRestore(config)
				if restoreErr != nil {
					appLogger.Error("fail to restore full data: " + restoreErr.Error())
				} else if restored {
					appLogger.Info("full data restored successfully")
				}
			}

			// 检查是否有待恢复的数据库备份（在打开数据库前执行）
			if config.PendingDBRestore != "" {
				restored, restoreErr := service.ExecuteDBRestore(config)
				if restoreErr != nil {
					appLogger.Error("fail to restore database: " + restoreErr.Error())
					fmt.Printf("fail to restore database: %v\n", restoreErr)
				} else if restored {
					appLogger.Info("database restored successfully")
					runtime.Quit(ctx)
					return
				}
			}

			execPath, err := utils.GetDataDir()
			if err != nil {
				appLogger.Fatal(err.Error())
			}
			dbPath := filepath.Join(execPath, "kaleidobox.db")
			db, err = sql.Open("duckdb", dbPath)
			if err != nil {
				appLogger.Fatal(err.Error())
			}

			// 设置时区为本地时区，确保 TIMESTAMPTZ 的聚合操作使用正确的日界线
			// 这对于按日期统计游戏时长非常重要
			// 注意：需要用户在前端设置时区（使用 Intl.DateTimeFormat().resolvedOptions().timeZone）
			timeZone := config.TimeZone
			if timeZone == "" {
				// 未配置时区，使用 UTC 作为默认值
				timeZone = "UTC"
				appLogger.Warning("TimeZone not configured, using UTC. Please set timezone in settings.")
			}

			_, err = db.Exec(fmt.Sprintf("SET TimeZone = '%s'", timeZone))
			if err != nil {
				appLogger.Warning("Failed to set timezone: " + err.Error())
			} else {
				appLogger.Info("Database timezone set to: " + timeZone)
			}

			if err := migrations.InitSchema(db); err != nil {
				appLogger.Fatal(err.Error())
			}

			// 运行数据库迁移（安全、只执行一次）
			appLogger.Info("Checking for pending database migrations...")
			if err := migrations.Run(ctx, db); err != nil {
				appLogger.Fatal("Database migration failed: " + err.Error())
			}
			appLogger.Info("Database migrations completed")

			configService.Init(ctx, db, config)
			// 设置安全退出回调
			configService.SetQuitHandler(func() {
				appState.QuitApplication()
			})
			applog.SetLogAll(config.LogAll)
			taskService.Init(ctx, db, config) // 初始化任务服务
			gameService.Init(ctx, db, config)
			aiService.Init(ctx, db, config)
			backupService.Init(ctx, db, config)
			homeService.Init(ctx, db, config)
			statsService.Init(ctx, db, config)
			sessionService.Init(ctx, db, config)
			startService.Init(ctx, db, config)
			categoryService.Init(ctx, db, config)
			importService.Init(ctx, db, config, gameService, taskService)
			versionService.Init(ctx)
			templateService.Init(ctx, db, config)
			updateService.Init(ctx, configService)
			staffService.Init(ctx, db, config)
			charactorService.Init(ctx, db, config)
			tagService.Init(ctx, db, config)
			imageService.Init(ctx, db, config)
			hotkeyService.Init(ctx, db, config)
			touchMappingService.Init(ctx, db, config)
			i18nService.Init(ctx)
			workService.Init(ctx, db, config)
			workService.SetServices(staffService, charactorService, imageService)
			gameService.SetServices(taskService, charactorService, staffService, workService, tagService, imageService)
			vmService.Init(ctx, db, config)
			vmService.SetGameService(gameService)
			// 设置 StartService 的 BackupService 依赖
			startService.SetBackupService(backupService)
			startService.SetGameService(gameService)
			startService.SetSessionService(sessionService)
			startService.SetHotkeyService(hotkeyService)
			startService.SetVMService(vmService)
			startService.SetStasService(statsService)

			hotkeyService.SetServices(imageService, startService, touchMappingService)
			touchMappingService.SetHotkeyService(hotkeyService)
			touchMappingService.SetImageService(imageService)

			// 设置 ImportService 的 SessionService 依赖（用于导入游玩记录）
			importService.SetSessionService(sessionService)

			// 启动 IPC Server (用于 CLI 通信)
			// 构造 CLI CoreApp 以共享 GUI 的服务实例
			// cliApp := &cli.CoreApp{
			// 	Config:         config,
			// 	DB:             db,
			// 	Ctx:            ctx,
			// 	GameService:    gameService,
			// 	StartService:   startService,
			// 	SessionService: sessionService,
			// 	BackupService:  backupService,
			// 	VersionService: versionService,
			// }
			// ipc.StartServer(cliApp)

			// 在 Wails 启动后初始化系统托盘
			// TODO: 升级wails v3，使用原生的托盘功能
			appState.StartTray()

			// 等待托盘初始化完成，避免竞态条件
			select {
			case <-appState.trayReady:
				appLogger.Info("system tray initialized successfully")
			case <-time.After(5 * time.Second):
				appLogger.Error("system tray initialization timed out")
			}
		},
		OnShutdown: func(ctx context.Context) {
			appState.BeginShutdown()

			// 先关闭 IPC Server，避免退出过程中还有外部请求进入。
			// if err := ipcserver.ShutdownServer(ipcHTTPServer); err != nil {
			// 	appLogger.Error("failed to shutdown IPC server: " + err.Error())
			// }

			// 关闭系统托盘
			appState.RequestTrayQuit()
			if appState.WaitForTrayExit(1200 * time.Millisecond) {
				appLogger.Info("system tray exited successfully")
			} else {
				appLogger.Warning("system tray exit timed out, continuing shutdown")
			}
			latestConfig, err := configService.GetAppConfig()
			if err != nil {
				appLogger.Error("failed to get latest config: " + err.Error())
			} else {
				// 更新窗口大小到最新配置
				latestConfig.WindowWidth = config.WindowWidth
				latestConfig.WindowHeight = config.WindowHeight
				config = &latestConfig
			}

			// 清理所有待定的进程选择会话（防止遗留临时会话）
			appLogger.Info("cleaning up pending process selections...")
			startService.CleanupPendingSessions()

			// 自动备份数据库（在关闭数据库前）
			if config.AutoBackupDB {
				appLogger.Info("performing automatic database backup...")
				_, err := backupService.CreateAndUploadDBBackup()
				if err != nil {
					appLogger.Error("automatic database backup failed: " + err.Error())
				} else {
					appLogger.Info("automatic database backup succeeded")
				}
			}

			// 关闭数据库连接
			if err := db.Close(); err != nil {
				appLogger.Error("failed to close database: " + err.Error())
			}

			// 保存最终配置
			if err := appconf.SaveConfig(config); err != nil {
				appLogger.Error("failed to save config: " + err.Error())
			}
		},
		Bind: []interface{}{
			gameService,
			aiService,
			backupService,
			homeService,
			statsService,
			startService,
			categoryService,
			configService,
			importService,
			versionService,
			templateService,
			updateService,
			sessionService,
			taskService,
			charactorService,
			staffService,
			tagService,
			imageService,
			hotkeyService,
			i18nService,
			workService,
			vmService,
		},
		EnumBind: []interface{}{
			enums.AllSourceTypes,
			enums.AllPeriodTypes,
			enums.Prompts,
			enums.AllGameStatuses,
			enums.AllTaskStatus,
			enums.AllTaskTypes,
			enums.AllStaffRoles,
			enums.AllDeviceTypes,
			enums.AllModifierKeys,
			enums.AllHotkeyActionTypes,
			enums.AllGuideSources,
		},
	})

	if bootstrapErr != nil {
		appLogger.Fatal(bootstrapErr.Error())
	}
}

// 系统托盘初始化
func onSystrayReady() {
	// 先设置托盘的基本属性
	systray.SetIcon(icon)
	systray.SetTitle("LunaBox")
	systray.SetTooltip("LunaBox")

	// 点击托盘图标时显示窗口
	systray.SetOnClick(func(menu systray.IMenu) {
		appState.ShowMainWindow()
	})

	// 双击托盘图标时也显示窗口
	systray.SetOnDClick(func(menu systray.IMenu) {
		appState.ShowMainWindow()
	})

	mShow := systray.AddMenuItem("显示主窗口", "显示 LunaBox 主窗口")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "退出 LunaBox")

	// energye/systray 使用 Click 方法设置回调，而不是 ClickedCh
	mShow.Click(func() {
		appState.ShowMainWindow()
	})

	mQuit.Click(func() {
		appState.QuitApplication()
	})

	// 通知主线程托盘已经准备就绪
	appState.MarkTrayReady()
}

func onSystrayExit() {
	appState.MarkTrayExit()
}

// 用于缓存转换后的视频文件路径
var videoCacheMap = make(map[string]string)
var videoCacheMutex sync.Mutex

// handleVideoPathStreamRequest 处理视频路径的视频流请求
func handleVideoPathStreamRequest(w http.ResponseWriter, r *http.Request, videoPath string, ctx context.Context) {
	// 定义ffmpeg路径
	var ffmpegPath string
	var err error
	if config != nil && config.FfmpegPath != "" {
		ffmpegPath = config.FfmpegPath
	} else {
		var dir string
		dir, err = utils.GetDataDir()
		if err != nil {
			applog.LogErrorf(ctx, "VideoPathStreamHandler: failed to get data dir: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ffmpegPath = fmt.Sprintf(`%s\ffmpeg.exe`, dir)
	}

	applog.LogInfof(ctx, "VideoPathStreamHandler: received request for video path: %s, User-Agent: %s", videoPath, r.UserAgent())

	// 检查文件是否存在
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		applog.LogErrorf(ctx, "VideoPathStreamHandler: video file not found: %s", videoPath)
		http.Error(w, "Video file not found", http.StatusNotFound)
		return
	}

	// 检查文件后缀是否为.mp4
	if strings.ToLower(filepath.Ext(videoPath)) == ".mp4" {
		applog.LogInfof(ctx, "VideoPathStreamHandler: direct serving mp4 file: %s", videoPath)

		// 打开文件
		file, err := os.Open(videoPath)
		if err != nil {
			applog.LogErrorf(ctx, "VideoPathStreamHandler: failed to open mp4 file: %v", err)
			http.Error(w, "Failed to open video file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		// 获取文件信息
		fileInfo, err := file.Stat()
		if err != nil {
			applog.LogErrorf(ctx, "VideoPathStreamHandler: failed to get file info: %v", err)
			http.Error(w, "Failed to get file info", http.StatusInternalServerError)
			return
		}

		// 设置响应头
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Disposition", "inline")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

		// 流式复制文件内容
		if _, err := io.Copy(w, file); err != nil {
			applog.LogErrorf(ctx, "VideoPathStreamHandler: failed to copy file content: %v", err)
			// 不要返回错误，因为可能是客户端断开连接导致的
		}

		applog.LogInfof(ctx, "VideoPathStreamHandler: served mp4 file directly")
		return
	}

	if _, err := os.Stat(ffmpegPath); err != nil {
		//ffmpeg不存在
		applog.LogError(ctx, "VideoPathStreamHandler: ffmpeg not found")
		return

	}

	// 为临时视频生成缓存键
	cacheKey := fmt.Sprintf("path_%s", videoPath)

	// 检查缓存中是否已有转换后的视频
	videoCacheMutex.Lock()
	cachedVideoPath, exists := videoCacheMap[cacheKey]
	videoCacheMutex.Unlock()

	if exists {
		// 检查缓存文件是否存在
		if _, err := os.Stat(cachedVideoPath); err == nil {
			applog.LogInfof(ctx, "VideoPathStreamHandler: using cached video file: %s", cachedVideoPath)
			// 使用http.ServeFile提供缓存的视频文件
			w.Header().Set("Content-Type", "video/mp4")
			w.Header().Set("Content-Disposition", "inline")
			w.Header().Set("Accept-Ranges", "bytes")
			http.ServeFile(w, r, cachedVideoPath)
			applog.LogInfof(ctx, "VideoPathStreamHandler: served cached video file")
			return
		}
		// 缓存文件不存在，删除缓存条目
		videoCacheMutex.Lock()
		delete(videoCacheMap, cacheKey)
		videoCacheMutex.Unlock()
		applog.LogInfof(ctx, "VideoPathStreamHandler: cached video file not found, removing from cache")
	}
	var tempFile *os.File

	// 创建临时文件来存储转换后的视频
	tempFile, err = os.CreateTemp("", "video_*.mp4")
	if err != nil {
		applog.LogErrorf(ctx, "VideoPathStreamHandler: failed to create temp file: %v", err)
		http.Error(w, "Failed to process video", http.StatusInternalServerError)
		return
	}
	tempFilePath := tempFile.Name()
	tempFile.Close()

	// 使用ffmpeg将视频转换为MP4格式并保存到临时文件
	err = utils.ConvertVideoWithFFmpeg(ctx, ffmpegPath, videoPath, tempFilePath, cacheKey)
	if err != nil {
		applog.LogErrorf(ctx, "VideoPathStreamHandler: ffmpeg conversion failed: %v", err)
		// 清理临时文件
		os.Remove(tempFilePath)
		http.Error(w, "Failed to process video", http.StatusInternalServerError)
		return
	}

	applog.LogInfof(ctx, "VideoPathStreamHandler: ffmpeg conversion completed")

	// 将转换后的视频路径加入缓存
	videoCacheMutex.Lock()
	videoCacheMap[cacheKey] = tempFilePath
	videoCacheMutex.Unlock()
	applog.LogInfof(ctx, "VideoPathStreamHandler: added video to cache: %s", tempFilePath)

	// 使用http.ServeFile提供转换后的视频文件
	// 这样浏览器可以一次性获取完整的视频文件，支持进度条拖动
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Accept-Ranges", "bytes")

	applog.LogInfof(ctx, "VideoPathStreamHandler: serving video file")
	http.ServeFile(w, r, tempFilePath)
	applog.LogInfof(ctx, "VideoPathStreamHandler: video file served")

	return
}

// handleVideoStreamRequest 处理视频流请求（通过游戏ID）
func handleVideoStreamRequest(w http.ResponseWriter, r *http.Request, gameService *service.GameService, ctx context.Context) {
	// 提取游戏ID
	gameID := strings.TrimPrefix(r.URL.Path, "/api/video/")
	applog.LogInfof(ctx, "VideoStreamHandler: received request for gameID: %s, User-Agent: %s", gameID, r.UserAgent())

	if gameID != "" && gameService != nil {
		applog.LogInfof(ctx, "VideoStreamHandler: gameService is available, processing request")
		// 获取视频路径
		videoPath, err := gameService.GetVideoStream(gameID)
		if err != nil {
			applog.LogErrorf(ctx, "VideoStreamHandler: failed to get video stream: %v", err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		applog.LogInfof(ctx, "VideoStreamHandler: got video path: %s", videoPath)
		// 调用新的函数处理视频流
		handleVideoPathStreamRequest(w, r, videoPath, ctx)
		return
	} else {
		applog.LogWarningf(ctx, "VideoStreamHandler: invalid request - gameID: %s, gameService: %v", gameID, gameService != nil)
	}
	http.Error(w, "Invalid video request", http.StatusBadRequest)
}
