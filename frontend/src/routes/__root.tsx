import { createRootRoute, Outlet, useLocation } from "@tanstack/react-router";
import { useEffect, useState, useRef } from "react";
import { OnFileDrop, OnFileDropOff } from "../../wailsjs/runtime/runtime";
import { SideBar } from "../components/bar/SideBar";
import { TopBar } from "../components/bar/TopBar";
import { DragDropImportModal } from "../components/modal/DragDropImportModal";
import { BatchUpdateModal } from "../components/modal/BatchUpdateModal";
import { useAppStore } from "../store";
import { models } from "../../wailsjs/go/models";

function RootLayout() {
  const { config, fetchGames } = useAppStore();
  const [isDragOver, setIsDragOver] = useState(false);
  const [showDragDropModal, setShowDragDropModal] = useState(false);
  const [droppedPaths, setDroppedPaths] = useState<string[]>([]);
  const [isLnk, setIsLnk] = useState(false);
  const [currentDropArea, setCurrentDropArea] = useState<'regular' | 'lnk' | null>(null);
  const currentDropAreaRef = useRef<'regular' | 'lnk' | null>(null);
  const [isBatchUpdateOpen, setIsBatchUpdateOpen] = useState(false);
  const [importedGamesForUpdate, setImportedGamesForUpdate] = useState<models.Game[]>([]);
  const mainContentRef = useRef<HTMLDivElement>(null);
  const location = useLocation();
  const scrollPositions = useRef<Record<string, number>>({});

  // 背景图相关配置
  const bgEnabled = config?.background_enabled && config?.background_image;
  const bgBlur = config?.background_blur ?? 10;
  const bgOpacity = config?.background_opacity ?? 0.85;

  // 设置文件拖拽事件监听
  useEffect(() => {
    // 监听文件拖拽事件
    OnFileDrop((_x: number, _y: number, paths: string[]) => {
      setIsDragOver(false);
      if (paths && paths.length > 0 && currentDropAreaRef.current !== null) {
        // 根据拖拽区域设置 isLnk
        setIsLnk(currentDropAreaRef.current === 'lnk');
        setDroppedPaths(paths);
        setShowDragDropModal(true);
      }
    }, true);

    // 清理函数
    return () => {
      OnFileDropOff();
    };
  }, []);

  // 监听拖拽进入/离开事件（用于视觉反馈）
  useEffect(() => {
    const handleDragOver = (e: DragEvent) => {
      e.preventDefault();

      // 过滤掉图片元素的拖拽（防止误触发）
      const target = e.target as HTMLElement;
      if (target.tagName === "IMG") {
        return;
      }

      // 检查是否包含文件
      if (e.dataTransfer?.types.includes("Files")) {
        setIsDragOver(true);

        // 计算鼠标位置，判断当前在哪个区域
        const regularArea = document.getElementById('regular-drop-area')?.getBoundingClientRect();
        const lnkArea = document.getElementById('lnk-drop-area')?.getBoundingClientRect();
        
        let area: 'regular' | 'lnk' | null = null;
        if (regularArea && e.clientY >= regularArea.top && e.clientY <= regularArea.bottom) {
          area = 'regular';
        } else if (lnkArea && e.clientY >= lnkArea.top && e.clientY <= lnkArea.bottom) {
          area = 'lnk';
        }
        
        setCurrentDropArea(area);
        currentDropAreaRef.current = area;
      }
    };

    const handleDragLeave = (e: DragEvent) => {
      // 只有当离开整个窗口时才隐藏遮罩
      if (e.relatedTarget === null) {
        setIsDragOver(false);
        setCurrentDropArea(null);
        currentDropAreaRef.current = null;
      }
    };

    const handleDrop = (e: DragEvent) => {
      // 过滤掉图片元素的拖拽
      const target = e.target as HTMLElement;
      if (target.tagName === "IMG") {
        e.preventDefault();
        e.stopPropagation();
        return;
      }
      setIsDragOver(false);
      // 保持 currentDropAreaRef 的值不变，因为 OnFileDrop 会使用它
    };

    window.addEventListener("dragover", handleDragOver);
    window.addEventListener("dragleave", handleDragLeave);
    window.addEventListener("drop", handleDrop);

    return () => {
      window.removeEventListener("dragover", handleDragOver);
      window.removeEventListener("dragleave", handleDragLeave);
      window.removeEventListener("drop", handleDrop);
    };
  }, []);

  // 保存和恢复滚动位置
  useEffect(() => {
    // 组件卸载时保存滚动位置
    return () => {
      if (mainContentRef.current) {
        scrollPositions.current[location.pathname] = mainContentRef.current.scrollTop;
      }
    };
  }, [location.pathname]);

  // 延迟恢复滚动位置，确保内容已加载
  useEffect(() => {
    // 恢复新路由的滚动位置
    const savedPosition = scrollPositions.current[location.pathname];
    if (mainContentRef.current && savedPosition !== undefined) {
      // 使用 setTimeout 确保内容已加载
      const scrollDelayStr = localStorage.getItem('scrollDelayMs');
      const scrollDelay = scrollDelayStr ? parseInt(scrollDelayStr) : 300;
      if (scrollDelayStr) {
        localStorage.removeItem('scrollDelayMs');
      }
      const timer = setTimeout(() => {
        mainContentRef.current!.scrollTop = savedPosition;
      }, scrollDelay);

      return () => clearTimeout(timer);
    } else if (mainContentRef.current) {
      // 如果没有保存的位置，滚动到顶部
      mainContentRef.current.scrollTop = 0;
    }
  }, [location.pathname]);

  // 监听滚动事件，实时更新当前路由的滚动位置
  useEffect(() => {
    const handleScroll = () => {
      if (mainContentRef.current) {
        const currentPath = location.pathname;
        scrollPositions.current[currentPath] = mainContentRef.current.scrollTop;
      }
    };

    if (mainContentRef.current) {
      mainContentRef.current.addEventListener('scroll', handleScroll);
    }

    return () => {
      if (mainContentRef.current) {
        mainContentRef.current.removeEventListener('scroll', handleScroll);
      }
    };
  }, [location.pathname]);

  const handleImportComplete = (importedGames: models.Game[], isOpenUpdate: boolean) => {
    fetchGames();
    if (isOpenUpdate && importedGames && importedGames.length > 0) {
      setImportedGamesForUpdate(importedGames);
      setIsBatchUpdateOpen(true);
    }
  };

  const handleCloseDragDropModal = () => {
    setShowDragDropModal(false);
    setDroppedPaths([]);
  };

  return (
    <div
      className="relative h-screen w-full overflow-hidden"
      data-glass={bgEnabled ? "true" : "false"}
      style={{ "--wails-drop-target": "drop" } as React.CSSProperties}
    >
      {/* 背景图层 */}
      {bgEnabled && (
        <div
          key={`bg-${bgBlur}-${config.background_image}`}
          className="absolute inset-0 bg-cover bg-center bg-no-repeat transition-all duration-300"
          style={{
            backgroundImage: `url("${config.background_image}")`,
            filter: `blur(${bgBlur}px)`,
            transform: "scale(1.1)", // 防止模糊边缘出现空白
          }}
        />
      )}

      {/* 主内容容器 */}
      <div className="relative flex h-full w-full flex-col text-brand-900 dark:text-brand-100">
        {/* 顶部栏 */}
        <TopBar />

        {/* 内容区域 */}
        <div className="flex flex-1 overflow-hidden">
          <SideBar bgEnabled={!!bgEnabled} bgOpacity={bgOpacity} />
          <main
            ref={mainContentRef}
            className={`flex-1 overflow-auto ${
              bgEnabled ? "" : "bg-brand-100 dark:bg-brand-900"
            }`}
            style={bgEnabled ? {
              backgroundColor: `rgba(var(--main-bg-rgb), ${bgOpacity})`,
            } : undefined}
          >
            <Outlet />
          </main>
        </div>
      </div>

      {/* 拖拽遮罩层 */}
      {isDragOver && (
        <div className="absolute inset-0 z-50 flex items-center justify-center bg-primary-500/20 backdrop-blur-sm pointer-events-none">
          <div className="flex flex-col items-center gap-8 p-8 rounded-2xl bg-white/90 dark:bg-brand-800/90 shadow-2xl border-2 border-dashed border-primary-500 w-full max-w-2xl">
            <div 
              className={`flex flex-col items-center gap-4 w-full p-6 rounded-xl transition-all duration-200 ${currentDropArea === 'regular' ? 'bg-blue-100/80 dark:bg-blue-900/40 border-2 border-blue-500' : 'border-2 border-transparent'}`}
              id="regular-drop-area"
            >
              <div className="i-mdi-folder-upload text-6xl text-blue-500 animate-bounce" />
              <div className="text-center">
                <p className="text-xl font-bold text-brand-900 dark:text-white">
                  释放游戏文件夹或可执行文件
                </p>
                <p className="text-sm text-brand-500 dark:text-brand-400 mt-1">
                  自动扫描文件夹中的 exe 文件
                </p>
              </div>
            </div>
            <div className="w-full border-t border-brand-200 dark:border-brand-700" />
            <div 
              className={`flex flex-col items-center gap-4 w-full p-6 rounded-xl transition-all duration-200 ${currentDropArea === 'lnk' ? 'bg-green-100/80 dark:bg-green-900/40 border-2 border-green-500' : 'border-2 border-transparent'}`}
              id="lnk-drop-area"
            >
              <div className="i-mdi-folder-upload text-6xl text-green-500 animate-bounce" />
              <div className="text-center">
                <p className="text-xl font-bold text-brand-900 dark:text-white">
                  释放快捷方式文件夹
                </p>
                <p className="text-sm text-brand-500 dark:text-brand-400 mt-1">
                  从快捷方式文件夹导入游戏列表
                </p>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 拖拽导入弹窗 */}
      <DragDropImportModal
        isOpen={showDragDropModal}
        droppedPaths={droppedPaths}
        isLnk={isLnk}
        onClose={handleCloseDragDropModal}
        onImportComplete={handleImportComplete}
      />

      <BatchUpdateModal
        isOpen={isBatchUpdateOpen}
        onClose={() => setIsBatchUpdateOpen(false)}
        onUpdateComplete={() => {
          setIsBatchUpdateOpen(false);
        }}
        games={importedGamesForUpdate || []}
      />
    </div>
  );
}

export const Route = createRootRoute({
  component: RootLayout,
});
