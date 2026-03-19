package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io"
	"lunabox/internal/applog"
	"lunabox/internal/cli"
	"lunabox/internal/cli/ipc"
	"lunabox/internal/utils"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

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

var appCtx context.Context

// 用于通知 systray 退出
var systrayQuit chan struct{}

// 用于同步托盘就绪状态
var systrayReady chan struct{}

// 标记是否是从托盘强制退出（绕过 OnBeforeClose 的最小化逻辑）
var forceQuit bool

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
	i18nService := service.NewI18nService()

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
		Title:     "LunaBox",
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
						applog.LogInfof(appCtx, "VideoCleanupHandler: received request for gameID: %s", gameID)

						// 先终止相关的ffmpeg进程
						ffmpegProcessMap := utils.GetFFmpegProcessMap()
						ffmpegProcessMutex := utils.GetFFmpegProcessMutex()
						ffmpegProcessMutex.Lock()
						if cmd, exists := ffmpegProcessMap[gameID]; exists {
							applog.LogInfof(appCtx, "VideoCleanupHandler: terminating ffmpeg process for gameID: %s", gameID)
							if err := cmd.Process.Kill(); err != nil {
								applog.LogErrorf(appCtx, "VideoCleanupHandler: failed to kill ffmpeg process: %v", err)
							} else {
								applog.LogInfof(appCtx, "VideoCleanupHandler: killed ffmpeg process for gameID: %s", gameID)
							}
							// 从映射中删除进程
							delete(ffmpegProcessMap, gameID)
						}
						ffmpegProcessMutex.Unlock()

						// 清理缓存
						videoCacheMutex.Lock()
						cachedVideoPath, exists := videoCacheMap[gameID]
						if exists {
							// 从缓存中删除条目
							delete(videoCacheMap, gameID)
							applog.LogInfof(appCtx, "VideoCleanupHandler: removed gameID from cache: %s", gameID)
						}
						videoCacheMutex.Unlock()

						// 尝试删除缓存文件，添加重试机制
						if exists && cachedVideoPath != "" {
							const maxRetries = 3
							for i := 0; i < maxRetries; i++ {
								if err := os.Remove(cachedVideoPath); err != nil {
									applog.LogErrorf(appCtx, "VideoCleanupHandler: attempt %d failed to remove cached video file: %v", i+1, err)
									// 等待一段时间后重试
									time.Sleep(time.Second)
								} else {
									applog.LogInfof(appCtx, "VideoCleanupHandler: removed cached video file: %s", cachedVideoPath)
									break
								}
							}
						}

						w.WriteHeader(http.StatusOK)
						w.Write([]byte("Video cache cleaned up"))
						return
					}

					// 处理视频流请求
					if strings.HasPrefix(r.URL.Path, "/api/video/") {
						handleVideoStreamRequest(w, r, gameService, appCtx)
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
			if forceQuit {
				return false
			}
			if config.CloseToTray {
				runtime.WindowHide(ctx)
				return true
			}
			return false
		},
		OnStartup: func(ctx context.Context) {
			appCtx = ctx
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
					configService.SafeQuit()
					runtime.Quit(ctx)
					return
				}
			}

			execPath, err := utils.GetDataDir()
			if err != nil {
				appLogger.Fatal(err.Error())
			}
			dbPath := filepath.Join(execPath, "lunabox.db")
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
				forceQuit = true
				runtime.Quit(ctx)
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
			i18nService.Init(ctx)
			workService.Init(ctx, db, config)
			workService.SetServices(staffService, charactorService, imageService)
			gameService.SetServices(taskService, charactorService, staffService, workService, tagService, imageService)
			// 设置 StartService 的 BackupService 依赖
			startService.SetBackupService(backupService)
			startService.SetGameService(gameService)
			startService.SetSessionService(sessionService)
			startService.SetHotkeyService(hotkeyService)

			hotkeyService.SetServices(imageService, startService)

			// 设置 ImportService 的 SessionService 依赖（用于导入游玩记录）
			importService.SetSessionService(sessionService)

			// 启动 IPC Server (用于 CLI 通信)
			// 构造 CLI CoreApp 以共享 GUI 的服务实例
			cliApp := &cli.CoreApp{
				Config:         config,
				DB:             db,
				Ctx:            ctx,
				GameService:    gameService,
				StartService:   startService,
				SessionService: sessionService,
				BackupService:  backupService,
				VersionService: versionService,
			}
			ipc.StartServer(cliApp)

			// 在 Wails 启动后初始化系统托盘
			// TODO: 升级wails v3，使用原生的托盘功能
			systrayQuit = make(chan struct{})
			systrayReady = make(chan struct{})
			go systray.Run(onSystrayReady, onSystrayExit)
			go startService.CleanupSessionsOnStart()

			// 等待托盘初始化完成，避免竞态条件
			<-systrayReady
			appLogger.Info("system tray initialized successfully")
		},
		OnShutdown: func(ctx context.Context) {
			// 关闭系统托盘
			if systrayQuit != nil {
				systray.Quit()
				<-systrayQuit // 等待 systray 完全退出
			}

			// 从 configService 获取最新配置（避免使用启动时的旧配置覆盖文件）
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
		// 确保 appCtx 已经初始化且有效
		if appCtx != nil {
			runtime.WindowShow(appCtx)
		}
	})

	// 双击托盘图标时也显示窗口
	systray.SetOnDClick(func(menu systray.IMenu) {
		// 确保 appCtx 已经初始化且有效
		if appCtx != nil {
			runtime.WindowShow(appCtx)
		}
	})

	mShow := systray.AddMenuItem("显示主窗口", "显示 LunaBox 主窗口")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "退出 LunaBox")

	// energye/systray 使用 Click 方法设置回调，而不是 ClickedCh
	mShow.Click(func() {
		// 确保 appCtx 已经初始化且有效
		if appCtx != nil {
			runtime.WindowShow(appCtx)
		}
	})

	mQuit.Click(func() {
		// 通过托盘退出时，设置强制退出标志，绕过 OnBeforeClose 的最小化逻辑
		forceQuit = true
		if appCtx != nil {
			runtime.Quit(appCtx)
		}
	})

	// 通知主线程托盘已经准备就绪
	if systrayReady != nil {
		close(systrayReady)
	}
}

func onSystrayExit() {
	if systrayQuit != nil {
		close(systrayQuit)
	}
}

// 用于缓存转换后的视频文件路径
var videoCacheMap = make(map[string]string)
var videoCacheMutex sync.Mutex

// handleVideoStreamRequest 处理视频流请求
func handleVideoStreamRequest(w http.ResponseWriter, r *http.Request, gameService *service.GameService, ctx context.Context) {
	// 定义ffmpeg路径
	dir, err := utils.GetDataDir()
	if err != nil {
		applog.LogErrorf(ctx, "VideoStreamHandler: failed to get data dir: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ffmpegPath := fmt.Sprintf(`%s\ffmpeg.exe`, dir)

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

		// 检查文件后缀是否为.mp4
		if strings.ToLower(filepath.Ext(videoPath)) == ".mp4" {
			applog.LogInfof(ctx, "VideoStreamHandler: direct serving mp4 file: %s", videoPath)

			// 打开文件
			file, err := os.Open(videoPath)
			if err != nil {
				applog.LogErrorf(ctx, "VideoStreamHandler: failed to open mp4 file: %v", err)
				http.Error(w, "Failed to open video file", http.StatusInternalServerError)
				return
			}
			defer file.Close()

			// 获取文件信息
			fileInfo, err := file.Stat()
			if err != nil {
				applog.LogErrorf(ctx, "VideoStreamHandler: failed to get file info: %v", err)
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
				applog.LogErrorf(ctx, "VideoStreamHandler: failed to copy file content: %v", err)
				// 不要返回错误，因为可能是客户端断开连接导致的
			}

			applog.LogInfof(ctx, "VideoStreamHandler: served mp4 file directly")
			return
		}

		if _, err := os.Stat(ffmpegPath); err != nil {
			//ffmpeg不存在
			applog.LogError(ctx, "VideoStreamHandler: ffmpeg not found")
			return

		}

		// 检查缓存中是否已有转换后的视频
		videoCacheMutex.Lock()
		cachedVideoPath, exists := videoCacheMap[gameID]
		videoCacheMutex.Unlock()

		if exists {
			// 检查缓存文件是否存在
			if _, err := os.Stat(cachedVideoPath); err == nil {
				applog.LogInfof(ctx, "VideoStreamHandler: using cached video file: %s", cachedVideoPath)
				// 使用http.ServeFile提供缓存的视频文件
				w.Header().Set("Content-Type", "video/mp4")
				w.Header().Set("Content-Disposition", "inline")
				w.Header().Set("Accept-Ranges", "bytes")
				http.ServeFile(w, r, cachedVideoPath)
				applog.LogInfof(ctx, "VideoStreamHandler: served cached video file")
				return
			}
			// 缓存文件不存在，删除缓存条目
			videoCacheMutex.Lock()
			delete(videoCacheMap, gameID)
			videoCacheMutex.Unlock()
			applog.LogInfof(ctx, "VideoStreamHandler: cached video file not found, removing from cache")
		}
		var tempFile *os.File

		// 创建临时文件来存储转换后的视频
		tempFile, err = os.CreateTemp("", "video_*.mp4")
		if err != nil {
			applog.LogErrorf(ctx, "VideoStreamHandler: failed to create temp file: %v", err)
			http.Error(w, "Failed to process video", http.StatusInternalServerError)
			return
		}
		tempFilePath := tempFile.Name()
		tempFile.Close()

		// 使用ffmpeg将视频转换为MP4格式并保存到临时文件
		err = utils.ConvertVideoWithFFmpeg(ctx, ffmpegPath, videoPath, tempFilePath, gameID)
		if err != nil {
			applog.LogErrorf(ctx, "VideoStreamHandler: ffmpeg conversion failed: %v", err)
			// 清理临时文件
			os.Remove(tempFilePath)
			http.Error(w, "Failed to process video", http.StatusInternalServerError)
			return
		}

		applog.LogInfof(ctx, "VideoStreamHandler: ffmpeg conversion completed")

		// 将转换后的视频路径加入缓存
		videoCacheMutex.Lock()
		videoCacheMap[gameID] = tempFilePath
		videoCacheMutex.Unlock()
		applog.LogInfof(ctx, "VideoStreamHandler: added video to cache: %s", tempFilePath)

		// 使用http.ServeFile提供转换后的视频文件
		// 这样浏览器可以一次性获取完整的视频文件，支持进度条拖动
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Disposition", "inline")
		w.Header().Set("Accept-Ranges", "bytes")

		applog.LogInfof(ctx, "VideoStreamHandler: serving video file")
		http.ServeFile(w, r, tempFilePath)
		applog.LogInfof(ctx, "VideoStreamHandler: video file served")

		return
	} else {
		applog.LogWarningf(ctx, "VideoStreamHandler: invalid request - gameID: %s, gameService: %v", gameID, gameService != nil)
	}
	http.Error(w, "Invalid video request", http.StatusBadRequest)
}
