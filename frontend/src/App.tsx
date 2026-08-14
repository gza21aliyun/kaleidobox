import { createRouter, RouterProvider } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { Toaster } from "react-hot-toast";
import { SafeQuit } from "../wailsjs/go/service/ConfigService";
import { EventsOff, EventsOn, WindowShow } from "../wailsjs/runtime/runtime";
import { ProcessSelectModal } from "./components/modal/ProcessSelectModal";
import { TimezoneSelectModal } from "./components/modal/TimezoneSelectModal";
import { UpdateDialog } from "./components/ui/UpdateDialog";
import { useUpdateCheck } from "./hooks/useUpdateCheck";
import { Route as rootRoute } from "./routes/__root";
import { Route as favoritesListRoute } from "./routes/favorites_list";
import { Route as favoritesRoute } from "./routes/favorites";
import { Route as gameRoute } from "./routes/game";
import { Route as indexRoute } from "./routes/index";
import { Route as libraryRoute } from "./routes/library";
import { Route as settingsRoute } from "./routes/settings";
import { Route as statsRoute } from "./routes/stats";
import { Route as staffRoute } from "./routes/staff";
import { Route as charactorRoute } from "./routes/charactor";
import { Route as charactorListRoute } from "./routes/charactor_list";
import { Route as TagListRoute } from "./routes/tag_list";
import { Route as joystickRoute } from "./routes/joystick";
import { Route as taskRoute } from "./routes/task";
import { Route as taskResultRoute } from "./routes/task_result_page";
import { Route as brandsRoute } from "./routes/brands";
import { Route as brandGamesRoute } from "./routes/brand_games";
import { Route as seriesListRoute } from "./routes/series_list";
import { Route as seriesGamesRoute } from "./routes/series_games";
import { Route as vmRoute } from "./routes/virtual_machines";
import { Route as categoryListRoute } from "./routes/category_list";
import { Route as categoryGamesRoute } from "./routes/category_games";
import { Route as monthlyReleasesRoute } from "./routes/monthly_releases";
import { Route as downloadedFilesRoute } from "./routes/downloaded_files";

import { useAppStore } from "./store";

const routeTree = rootRoute.addChildren([indexRoute, libraryRoute, gameRoute, statsRoute, favoritesListRoute,
  favoritesRoute, settingsRoute, staffRoute, charactorRoute, charactorListRoute, TagListRoute, joystickRoute, taskRoute, taskResultRoute,
  brandsRoute, brandGamesRoute, seriesListRoute, seriesGamesRoute, vmRoute, categoryListRoute, categoryGamesRoute, monthlyReleasesRoute, downloadedFilesRoute]);

const router = createRouter({ routeTree });

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

function App() {
  const { config, fetchConfig, updateConfig } = useAppStore();
  const { updateInfo, showUpdateDialog, setShowUpdateDialog, handleSkipVersion } = useUpdateCheck();
  const [showTimezoneModal, setShowTimezoneModal] = useState(false);
  const [processSelectData, setProcessSelectData] = useState<{
    isOpen: boolean;
    gameID: string;
    launcherExeName: string;
  }>({ isOpen: false, gameID: "", launcherExeName: "" });

  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  // 监听后端发送的进程选择事件
  useEffect(() => {
    const handleProcessSelectRequired = (data: { gameID: string; sessionID: string; launcherExeName: string }) => {
      console.warn("Process select required:", data);

      // 将窗口显示到前台
      WindowShow();

      setProcessSelectData({
        isOpen: true,
        gameID: data.gameID,
        launcherExeName: data.launcherExeName,
      });
    };

    EventsOn("process-select-required", handleProcessSelectRequired);

    return () => {
      EventsOff("process-select-required");
    };
  }, []);

  // 检查时区配置，如果未设置则显示选择弹窗
  useEffect(() => {
    if (config && (!config.time_zone || config.time_zone === "")) {
      setShowTimezoneModal(true);
    }
  }, [config]);

  const handleTimezoneConfirm = async (timezone: string) => {
    if (!config)
      return;

    // 更新配置
    const newConfig = { ...config, time_zone: timezone };
    await updateConfig(newConfig);

    // 关闭弹窗
    setShowTimezoneModal(false);

    // 延迟 500ms 后重启应用
    setTimeout(() => {
      SafeQuit();
    }, 500);
  };

  useEffect(() => {
    if (!config)
      return;

    const root = window.document.documentElement;
    const applyTheme = (theme: string) => {
      // 切换主题时临时禁用所有 transition，避免闪烁
      root.classList.add("theme-transitioning");
      root.classList.remove("light", "dark");
      root.classList.add(theme);

      // 在下一帧移除禁用类，让 hover 等交互恢复 transition
      requestAnimationFrame(() => {
        setTimeout(() => {
          root.classList.remove("theme-transitioning");
        }, 0);
      });
    };

    // 缓存主题设置到 localStorage，供下次启动时预加载
    localStorage.setItem("kaleidobox-theme", config.theme);

    if (config.theme === "system") {
      const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
      applyTheme(mediaQuery.matches ? "dark" : "light");

      const handler = (e: MediaQueryListEvent) => {
        applyTheme(e.matches ? "dark" : "light");
      };

      mediaQuery.addEventListener("change", handler);
      return () => mediaQuery.removeEventListener("change", handler);
    }
    else {
      applyTheme(config.theme);
    }
  }, [config?.theme]);

  // 配置加载完成后显示窗口
  useEffect(() => {
    if (config) {
      // 标记内容已准备好，触发淡入动画
      document.getElementById("root")?.classList.add("ready");
      // 显示窗口
      WindowShow();
    }
  }, [config]);

  return (
    <>
      <RouterProvider router={router} />
      <Toaster
        position="top-center"
        toastOptions={{
          duration: 3000,
          style: {
            background: "var(--toast-bg, #fff)",
            color: "var(--toast-color, #374151)",
          },
          success: {
            iconTheme: {
              primary: "#10b981",
              secondary: "#fff",
            },
          },
          error: {
            iconTheme: {
              primary: "#ef4444",
              secondary: "#fff",
            },
          },
        }}
      />
      {/* {showUpdateDialog && updateInfo && (
        <UpdateDialog
          updateInfo={updateInfo}
          onClose={() => setShowUpdateDialog(false)}
          onSkip={handleSkipVersion}
        />
      )} */}
      <TimezoneSelectModal
        isOpen={showTimezoneModal}
        onConfirm={handleTimezoneConfirm}
      />
      <ProcessSelectModal
        isOpen={processSelectData.isOpen}
        gameID={processSelectData.gameID}
        launcherExeName={processSelectData.launcherExeName}
        onClose={() => setProcessSelectData({ isOpen: false, gameID: "", launcherExeName: "" })}
        onSelected={() => setProcessSelectData({ isOpen: false, gameID: "", launcherExeName: "" })}
      />
    </>
  );
}

export default App;
