import { models, vo } from "../../wailsjs/go/models";
import { createRoute, useNavigate, useSearch } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { toast } from "react-hot-toast";
import { enums } from "../../wailsjs/go/models";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import { AddGameToCategory, GetCategories, GetCategoriesByGame, RemoveGameFromCategory } from "../../wailsjs/go/service/CategoryService";
import { DeleteGame, GetGameByID, SelectCoverImage, SelectGameExecutable, SelectSaveDirectory, SelectSaveFile, UpdateGame, UpdateGameFromRemote } from "../../wailsjs/go/service/GameService";
import { StartGameWithTracking } from "../../wailsjs/go/service/StartService";
import { AddToCategoryModal } from "../components/modal/AddToCategoryModal";
import { ConfirmModal } from "../components/modal/ConfirmModal";
import { GameBackupPanel } from "../components/panel/GameBackupPanel";
import { GameEditPanel } from "../components/panel/GameEditPanel";
import { GameLaunchPanel } from "../components/panel/GameLaunchPanel";
import { Ps4Panel } from "../components/panel/Ps4Panel";
import { GameStatsPanel } from "../components/panel/GameStatsPanel";
import { GameDetailSkeleton } from "../components/skeleton/GameDetailSkeleton";
import { useAppStore } from "../store";
import { formatLocalDate } from "../utils/time";
import { Route as rootRoute } from "./__root";
import { GameInfoPanel } from "../components/panel/GameInfoPanel";
import { GameGalleryPanel } from "../components/panel/GameGalleryPanel"; // 新增导入
import { GameIntroPanel } from "../components/panel/GameIntroPanel";
import { OpenBrowser } from "../../wailsjs/go/service/ImportService"; 
import { useTranslation } from 'react-i18next';

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/game/$gameId",
  component: GameDetailPage,
});

export interface GameSearchParams {
  filteredGameIdsStr?: string[]; // 可选参数
}

function GameDetailPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { gameId } = Route.useParams();
  const [ currentGameId, setCurrentGameId ] = useState(gameId);
  const search = useSearch({ strict: false }) as GameSearchParams; // 获取查询参数
  const filteredGameIds = search.filteredGameIdsStr || [];
  const config = useAppStore(state => state.config);
  const setGames = useAppStore(state => state.setGames);
  const [game, setGame] = useState<models.Game | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [showSkeleton, setShowSkeleton] = useState(false);
  const [activeTab, setActiveTab] = useState("intro");
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [isCategoryModalOpen, setIsCategoryModalOpen] = useState(false);
  const [allCategories, setAllCategories] = useState<vo.CategoryVO[]>([]);
  const [selectedCategoryIds, setSelectedCategoryIds] = useState<string[]>([]);
  const isInitialMount = useRef(true);
  const originalGameData = useRef<models.Game | null>(null);

  console.log("ids:", filteredGameIds)

  useEffect(() => {
        const unlistenTaskUpdate = EventsOn("game_updates", (data: any) => {
          
            // 注意：使用正确的语法从data对象获取值
            const task : models.TaskNotice = new models.TaskNotice(data);
            
            // console.log("received taskid:" + task.id + " current taskid:" + taskId + ", item_id:" + task.item_id +  " item_status:" + task.item_status)
            if (task.item_status === enums.TaskStatus.COMPLETED && task.item_id == currentGameId) {
                    const newGame : models.Game = task.item_data as models.Game;
                    console.log("newGame:", newGame)
                    setGame(newGame)
                    
    
                }
            
            
            
        });
    
        return () => {
            if (unlistenTaskUpdate) {
            unlistenTaskUpdate(); // 取消事件监听
            }
        };
        }, [game]);

  useEffect(() => {
    if (gameId !== currentGameId) {
      setCurrentGameId(gameId);
    }
  }, [gameId]);

  const loadData = async () => {
      try {
        const gameData = await GetGameByID(currentGameId);
        setGame(gameData);
        console.log("gameData", gameData);
        originalGameData.current = gameData;
        isInitialMount.current = false;
      }
      catch (error) {
        console.error("Failed to load game data:", error);
        toast.error(t('game.toasts.loadGameFailed'));
      }
      finally {
        setIsLoading(false);
      }
    };

  useEffect(() => {
    
    loadData();
  }, [currentGameId]);

  

  // 延迟显示骨架屏
  useEffect(() => {
    let timer: number;
    if (isLoading) {
      timer = window.setTimeout(() => {
        setShowSkeleton(true);
      }, 300);
    }
    else {
      setShowSkeleton(false);
    }
    return () => clearTimeout(timer);
  }, [isLoading]);

  // 自动保存
  useEffect(() => {
    if (!game || isInitialMount.current)
      return;

    const hasChanges = JSON.stringify(game) !== JSON.stringify(originalGameData.current);
    if (!hasChanges)
      return;

    const timer = setTimeout(async () => {
      try {
        await UpdateGame(game);
        originalGameData.current = game;
      }
      catch (error) {
        console.error("Failed to auto-save game:", error);
        toast.error(t('game.toasts.saveFailed') + `: ${(error as Error).message}`);
      }
    }, 500);

    return () => clearTimeout(timer);
  }, [game]);

  const currentIndex = filteredGameIds.indexOf(currentGameId);

  // 计算是否可以向左/向右切换
  const canGoPrev = currentIndex > 0;
  const canGoNext = currentIndex < filteredGameIds.length - 1;
  console.log("index:", currentIndex)

  // 切换到上一个游戏
  const goToPrevGame = () => {
    if (canGoPrev) {
      const prevGameId = filteredGameIds[currentIndex - 1];
      // navigate({ to: `/game/${prevGameId}`, search });
      setCurrentGameId(prevGameId);
    }
  };

  // 切换到下一个游戏
  const goToNextGame = () => {
    if (canGoNext) {
      const nextGameId = filteredGameIds[currentIndex + 1];
      // navigate({ to: `/game/${nextGameId}`, search });
      setCurrentGameId(nextGameId);
    }
  };

  useEffect(() => {
    if (!filteredGameIds || filteredGameIds.length == 0) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (!filteredGameIds || filteredGameIds.length == 0) return;

      const activeElement = document.activeElement;
      const isInputFocused =
        activeElement instanceof HTMLInputElement ||
        activeElement instanceof HTMLTextAreaElement;

      // 如果焦点在 input 上，不处理方向键
      if (isInputFocused) {
        return;
      }
      
      
      
      // 获取按下的键
      const key = e.key.toUpperCase();
      let displayKey = key;
      console.log("key:", e.code)
      
      // 特殊键处理
      switch (e.code) {
        case "ArrowLeft":
          e.preventDefault();
          goToPrevGame();
          break;
        case "ArrowRight":
          e.preventDefault();
          goToNextGame();
          break;
        
      }
      
    };

    const handleKeyUp = (e: KeyboardEvent) => {
      // if (!isListening || selectedDeviceType !== enums.DeviceType.KEYBOARD) return;
      // 不处理keyup事件
    };

    if (true) {
      window.addEventListener("keydown", handleKeyDown);
      window.addEventListener("keyup", handleKeyUp);
    }

    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("keyup", handleKeyUp);
    };
  }, [filteredGameIds, currentGameId]);

  if (isLoading && !game) {
    if (!showSkeleton) {
      return null;
    }
    return <GameDetailSkeleton />;
  }

  if (!game) {
    return (
      <div className="flex flex-col items-center justify-center h-full space-y-4 text-brand-500">
        <div className="i-mdi-gamepad-variant-outline text-6xl" />
        <p className="text-xl">{t('game.errors.gameNotFound')}</p>
        <button onClick={() => navigate({ to: "/library" })} className="text-neutral-600 hover:underline">{t('game.buttons.returnToLibrary')}</button>
      </div>
    );
  }

  const handleSelectExecutable = async () => {
    try {
      const path = await SelectGameExecutable();
      if (path && game) {
        setGame({ ...game, path } as models.Game);
      }
    }
    catch (error) {
      console.error("Failed to select executable:", error);
      toast.error(t('game.toasts.selectExecutableFailed'));
    }
  };

  const handleDeleteGame = async () => {
    if (!game)
      return;
    setIsDeleteModalOpen(true);
  };

  const confirmDeleteGame = async () => {
    if (!game)
      return;
    try {
      await DeleteGame(game.id);
      setGames([])
      toast.success(t('common.deleteSuccess'));
      navigate({ to: "/library" });
    }
    catch (error) {
      console.error("Failed to delete game:", error);
      toast.error(t('common.deleteFailed'));
    }
  };

  const handleSelectSaveDirectory = async () => {
    try {
      const path = await SelectSaveDirectory();
      if (path && game) {
        setGame({ ...game, save_path: path } as models.Game);
      }
    }
    catch (error) {
      console.error("Failed to select save directory:", error);
      toast.error(t('game.toasts.selectSavePathFailed'));
    }
  };

  const handleSelectSaveFile = async () => {
    try {
      const path = await SelectSaveFile();
      if (path && game) {
        setGame({ ...game, save_path: path } as models.Game);
      }
    }
    catch (error) {
      console.error("Failed to select save file:", error);
      toast.error(t('game.toasts.selectSavePathFailed'));
    }
  };

  const handleSelectCoverImage = async () => {
    if (!game)
      return;
    try {
      const coverUrl = await SelectCoverImage(game.id);
      if (coverUrl) {
        setGame({ ...game, cover_url: coverUrl } as models.Game);
      }
    }
    catch (error) {
      console.error("Failed to select cover image:", error);
      toast.error(t('game.toasts.selectCoverFailed'));
    }
  };

  const handleUpdateFromRemote = async () => {
    if (!game)
      return;
    try {
      await UpdateGameFromRemote(game.id);
      const updatedGame = await GetGameByID(game.id);
      setGame(updatedGame);
      originalGameData.current = updatedGame;
      toast.success(t('game.toasts.remoteUpdateSuccess'));
    }
    catch (error) {
      console.error("Failed to update from remote:", error);
      toast.error(t('game.toasts.remoteUpdateFailed') + `: ${error}`);
    }
  };

  const statusConfig = {
    [enums.GameStatus.NOT_STARTED]: { label: t('library.gameStatus.not_started'), icon: "i-mdi-clock-outline", color: "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300" },
    [enums.GameStatus.PLAYING]: { label: t('library.gameStatus.playing'), icon: "i-mdi-gamepad-variant", color: "bg-neutral-100 text-neutral-700 dark:bg-neutral-900 dark:text-neutral-300" },
    [enums.GameStatus.COMPLETED]: { label: t('library.gameStatus.completed'), icon: "i-mdi-trophy", color: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300" },
    [enums.GameStatus.ON_HOLD]: { label: t('library.gameStatus.on_hold'), icon: "i-mdi-pause-circle-outline", color: "bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300" },
  };

  const handleStartGame = async () => {
    if (!game || !game.id)
      return;
    try {
      const started = await StartGameWithTracking(game.id);
      if (started) {
        toast.success(t('game.toasts.gameLaunchSuccess', { name: game.name }));
      }
      else {
        toast.error(t('game.toasts.gameLaunchFailed', { name: game.name }));
      }
    }
    catch (error) {
      console.error("Failed to start game:", error);
      toast.error(t('game.toasts.gameLaunchFailedCheckLog', { name: game.name }));
    }
  };

  const handleStatusChange = async (newStatus: string) => {
    if (!game)
      return;
    const updatedGame = { ...game, status: newStatus } as models.Game;
    setGame(updatedGame);
    try {
      await UpdateGame(updatedGame);
      toast.success(t('game.toasts.statusUpdated'));
    }
    catch (error) {
      console.error("Failed to update status:", error);
      toast.error(t('game.toasts.statusUpdateFailed'));
    }
  };

  const handleTagTaps = async (tag: string) => {
    try {
      
    } finally { 

    }
  };

  const openCategoryModal = async () => {
    try {
      const [categories, gameCategories] = await Promise.all([
        GetCategories(),
        GetCategoriesByGame(currentGameId),
      ]);
      setAllCategories(categories || []);
      setSelectedCategoryIds(gameCategories?.map(c => c.id) || []);
      setIsCategoryModalOpen(true);
    }
    catch (error) {
      console.error("Failed to load categories:", error);
      toast.error(t('game.toasts.loadCategoriesFailed'));
    }
  };

  const handleSaveCategories = async (newSelectedIds: string[]) => {
    const currentIds = selectedCategoryIds;

    // 计算需要添加的和移除的
    const toAdd = newSelectedIds.filter(id => !currentIds.includes(id));
    const toRemove = currentIds.filter(id => !newSelectedIds.includes(id));

    try {
      // 执行添加操作
      for (const categoryId of toAdd) {
        await AddGameToCategory(currentGameId, categoryId);
      }
      // 执行移除操作
      for (const categoryId of toRemove) {
        await RemoveGameFromCategory(currentGameId, categoryId);
      }

      setSelectedCategoryIds(newSelectedIds);

      // 刷新所有分类的game_count
      const categories = await GetCategories();
      setAllCategories(categories || []);

      if (toAdd.length > 0 || toRemove.length > 0) {
        toast.success(t('game.toasts.collectionUpdated'));
      }
    }
    catch (error) {
      console.error("Failed to update categories:", error);
      toast.error(t('game.toasts.updateCollectionFailed'));
    }
  };

  const handleSelectProcessExecutable = async () => {
    try {
      const path = await SelectGameExecutable();
      if (path && game) {
        // 从路径中提取文件名
        const filename = path.split(/[\\/]/).pop();
        if (filename) {
          setGame({ ...game, process_name: filename } as models.Game);
        }
      }
    }
    catch (error) {
      console.error("Failed to select executable:", error);
      toast.error(t('game.toasts.selectFileFailed'));
    }
  };

  const getEroscapeUrl = () => {
    if (config?.eroscape_use_mirror || false) {
      return `https://koko.kyara.top/game.php?game=${game.eroscape_id}`
    } else {
      return `https://erogamescape.dyndns.org/~ap2/ero/toukei_kaiseki/game.php?game=${game.eroscape_id}`
    }
  }

  

  return (
    <div className={`space-y-8 max-w-8xl mx-auto p-8 transition-opacity duration-300 ${isLoading ? "opacity-50 pointer-events-none" : "opacity-100"}`}>
      {/* Back Button */}
      <div className="flex justify-between items-center">
        <button
          onClick={() => window.history.back()}
          className="flex rounded-md items-center text-brand-750 hover:text-brand-900 dark:text-brand-400 dark:hover:text-brand-200 transition-colors"
        >
          <div className="i-mdi-arrow-left text-2xl mr-1" />
          <span>{t('common.back')}</span>
        </button>

        {/* Navigation Arrows */}
        {filteredGameIds.length > 0 && (
          <div className="flex gap-2">
            <button
              onClick={goToPrevGame}
              disabled={!canGoPrev}
              className={`flex items-center justify-center w-10 h-10 rounded-full ${
                canGoPrev
                  ? "bg-brand-100 text-brand-700 hover:bg-brand-200 dark:bg-brand-700 dark:text-brand-200 dark:hover:bg-brand-600"
                  : "bg-gray-200 text-gray-400 cursor-not-allowed dark:bg-gray-700 dark:text-gray-500"
              }`}
              title={t('common.previous')}
            >
              <div className="i-mdi-arrow-left text-xl" />
            </button>

            <button
              onClick={goToNextGame}
              disabled={!canGoNext}
              className={`flex items-center justify-center w-10 h-10 rounded-full ${
                canGoNext
                  ? "bg-brand-100 text-brand-700 hover:bg-brand-200 dark:bg-brand-700 dark:text-brand-200 dark:hover:bg-brand-600"
                  : "bg-gray-200 text-gray-400 cursor-not-allowed dark:bg-gray-700 dark:text-gray-500"
              }`}
              title={t('common.next')}
            >
              <div className="i-mdi-arrow-right text-xl" />
            </button>
          </div>
        )}
      </div>

      {/* Header Section */}
      <div className="flex gap-6 items-center">
        <div className="relative w-60 flex-shrink-0 rounded-lg overflow-hidden shadow-lg bg-brand-200 dark:bg-brand-800">
          {game.cover_url
            ? (
                <img
                  src={game.cover_url}
                  alt={game.name}
                  className="w-full h-auto block"
                  referrerPolicy="no-referrer"
                  draggable="false"
                  onDragStart={e => e.preventDefault()}
                />
              )
            : (
                <div className="w-full h-64 flex items-center justify-center text-brand-400">
                  No Cover
                </div>
              )}
        </div>

        <div className="flex-1 space-y-4">
          <div className="flex flex-col gap-3">
            <h1 className="text-4xl font-bold text-brand-900 dark:text-white">{game.name}</h1>
            {/* 操作和状态标签组 */}
            <div className="flex items-center gap-4">
              <button
                onClick={handleStartGame}
                className="flex items-center gap-1.5 rounded-lg bg-neutral-600 text-white shadow-md hover:bg-neutral-700 transition-all duration-300 px-4 py-1.5 text-sm font-medium dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-200"
              >
                <div className="i-mdi-play text-lg" />
                {t('game.buttons.launchGame')}
              </button>

              <div className="h-6 w-px bg-brand-200 dark:bg-brand-700" />
              {" "}
              {/* 分隔线 */}

              <div className="flex gap-1.5">
                {Object.entries(statusConfig).map(([key, config]) => {
                  const isActive = (game.status || enums.GameStatus.NOT_STARTED) === key;
                  return (
                    <button
                      key={key}
                      onClick={() => handleStatusChange(key)}
                      className={`flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium transition-all ${isActive
                        ? `${config.color} ring-2 ring-offset-1 ring-brand-400 dark:ring-offset-brand-900`
                        : "bg-brand-100 text-brand-500 dark:bg-brand-700 dark:text-brand-400 hover:bg-brand-200 dark:hover:bg-brand-600"
                      }`}
                      title={config.label}
                    >
                      <div className={`${config.icon} text-base`} />
                      {isActive && <span>{config.label}</span>}
                    </button>
                  );
                })}
              </div>
            </div>
          </div>

          <div className="grid grid-cols-4 gap-4 text-sm text-brand-750 dark:text-brand-400">
            <div>
              <div className="font-semibold mb-1">{t('game.labels.source')}</div>
              <div>{game.source_type}</div>
            </div>
            <div>
              <div className="font-semibold mb-1">{t('game.labels.developer')}</div>
              <div>{game.company || "-"}</div>
            </div>
            <div>
              <div className="font-semibold mb-1">{t('game.labels.addedAt')}</div>
              <div>{formatLocalDate(game.created_at, config?.time_zone)}</div>
            </div>
            <div>
              <div className="font-semibold mb-1">{t('game.labels.releaseDate')}</div>
              <div>{formatLocalDate(game.release_at, config?.time_zone)}</div>
            </div>
            {/* Placeholders for missing data */}
          </div>

          
          <div className="flex gap-2 mt-4">
            {game.bangumi_id && (
              <button
                onClick={() => window.open(`https://bgm.tv/subject/${game.bangumi_id}`)}
                className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 transition-colors"
              >
                Bangumi
              </button>
            )}
            {game.ymgal_id && (
              <button
                onClick={() => window.open(`https://www.ymgal.games/co/${game.ymgal_id}`)}
                className="px-4 py-2 bg-purple-500 text-white rounded hover:bg-purple-600 transition-colors"
              >
                {t('game.buttons.ymgal')}
              </button>
            )}
            {game.dmm_id && (
              <button
                onClick={() => window.open(`https://dlsoft.dmm.co.jp/detail/${game.dmm_id}/`)}
                className="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600 transition-colors"
              >
                Dmm
              </button>
            )}
            {game.eroscape_id && (
              <button
                onClick={() => OpenBrowser(getEroscapeUrl())}
                className="px-4 py-2 bg-yellow-500 text-white rounded hover:bg-yellow-600 transition-colors"
              >
                {t('game.buttons.eroscape')}
              </button>
            )}
          </div>

        
          
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-brand-200 dark:border-brand-700">
        <div className="flex justify-between items-center">
          <nav className="-mb-px flex space-x-8">
            {["intro","stats", "edit", "launch", "backup", "info", "gallery", "joystick"].map(tab => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={`
                  whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm
                  ${activeTab === tab
                ? "border-neutral-500 text-brand-700 dark:text-neutral-400"
                : "border-transparent text-brand-700 hover:text-brand-750 hover:border-brand-300 dark:text-brand-400 dark:hover:text-brand-300"}
                `}
              >
                {tab === "stats" && t('common.gameStats')}
                {tab === "edit" && t('common.edit')}
                {tab === "launch" && t('common.launchConfig')}
                {tab === "backup" && t('common.backup')}
                {tab === "info" && t('common.gameInfo')}
                {tab === "gallery" && t('common.gallery')}
                {tab === "intro" && t('common.introduction')}
                {tab === "joystick" && t('common.joystick')}
              </button>
            ))}
          </nav>
          <button
            onClick={openCategoryModal}
            className="flex items-center gap-2 px-3 py-2 rounded-lg bg-brand-100 text-brand-750 hover:text-brand-200 dark:bg-brand-900 dark:text-brand-400 dark:hover:text-brand-700 transition-colors"
            title={t('game.buttons.addToCollection')}
          >
            <div className="i-mdi-folder-plus-outline text-lg" />
          </button>
        </div>
      </div>

      {/* Content */}
      {activeTab === "stats" && (
        <GameStatsPanel gameId={currentGameId} />
      )}

      {activeTab === "edit" && game && (
        <GameEditPanel
          game={game}
          onGameChange={setGame}
          onDelete={handleDeleteGame}
          onSelectExecutable={handleSelectExecutable}
          onSelectSaveDirectory={handleSelectSaveDirectory}
          onSelectSaveFile={handleSelectSaveFile}
          onSelectCoverImage={handleSelectCoverImage}
          onUpdateFromRemote={handleUpdateFromRemote}
          onLoadGame={loadData}
        />
      )}

      {activeTab === "launch" && game && (
        <GameLaunchPanel
          game={game}
          config={config || undefined}
          onGameChange={setGame}
          onSelectProcessExecutable={handleSelectProcessExecutable}
        />
      )}


      {activeTab === "backup" && (
        <GameBackupPanel gameId={currentGameId} savePath={game?.save_path} />
      )}

      {activeTab === "info" && game && (
        <GameInfoPanel
          game={game}
          config={config || undefined}
          onTagTaps={handleTagTaps}
        />
      )}

      {activeTab === "gallery" && game && (
        <GameGalleryPanel
          game={game}
        />
      )}

      {activeTab === "intro" && game && (
        <GameIntroPanel
          game={game}
          onTagTaps={handleTagTaps}
        />
      )}

      {activeTab === "joystick" && game && (
        <Ps4Panel
          gameId={game.id}
        />
      )}

      <ConfirmModal
        isOpen={isDeleteModalOpen}
        title={t('common.deleteGame')}
        message={t('game.modals.deleteMessage', { name: game.name })}
        confirmText={t('common.confirmDelete')}
        type="danger"
        onClose={() => setIsDeleteModalOpen(false)}
        onConfirm={confirmDeleteGame}
      />

      <AddToCategoryModal
        isOpen={isCategoryModalOpen}
        allCategories={allCategories}
        initialSelectedIds={selectedCategoryIds}
        onClose={() => setIsCategoryModalOpen(false)}
        onSave={handleSaveCategories}
      />


    </div>
  );
}
