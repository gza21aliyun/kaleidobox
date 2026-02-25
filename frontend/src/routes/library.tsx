import { models } from "../../wailsjs/go/models";
import type { ImportSource } from "../components/modal/GameImportModal";
import { createRoute } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { GetGames } from "../../wailsjs/go/service/GameService";
import { ListTags } from "../../wailsjs/go/service/TagService";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import { toast } from "react-hot-toast";
import { AddGamesToCategories, GetCategories } from "../../wailsjs/go/service/CategoryService";
import { BatchUpdateStatus, DeleteGames } from "../../wailsjs/go/service/GameService";
import { FilterBar } from "../components/bar/FilterBar";
import { GameCard } from "../components/card/GameCard";
import { AddGameModal } from "../components/modal/AddGameModal";
import { AddToCategoryModal } from "../components/modal/AddToCategoryModal";
import { BatchImportModal } from "../components/modal/BatchImportModal";
import { arrayToMap } from "../components/utils/Utility";
import { ConfirmModal } from "../components/modal/ConfirmModal";
import { GameImportModal } from "../components/modal/GameImportModal";
import { LibrarySkeleton } from "../components/skeleton/LibrarySkeleton";
import { BetterDropdownMenu } from "../components/ui/BetterDropdownMenu";
import { sortOptions, statusOptions } from "../consts/options";
import { useAppStore } from "../store";
import { Route as rootRoute } from "./__root";
import { TaskPanel } from "../components/panel/TaskPanel";
import { arrayFind, mapToArray } from "../components/utils/Utility";
import { formatLocalDate } from "../utils/time";

import { enums, vo } from "../../wailsjs/go/models";
import { BatchUpdateModal } from "../components/modal/BatchUpdateModal";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/library",
  component: LibraryPage,
});



function LibraryPage() {
  const { games, gamesLoading, fetchGames, setGames } = useAppStore();
  const [tagsLoaded, setTagsLoaded] = useState<Map<string, models.Tag[]>>(new Map())
  const [showSkeleton, setShowSkeleton] = useState(false);
  const [isAddGameModalOpen, setIsAddGameModalOpen] = useState(false);
  const [isBatchImportOpen, setIsBatchImportOpen] = useState(false);
  const [isBatchUpdateOpen, setIsBatchUpdateOpen] = useState(false);
  const [importSource, setImportSource] = useState<ImportSource | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState<"name" | "created_at" | "release_at">("created_at");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [tagsFilter, setTags] = useState<string[]>([]);
  const [batchMode, setBatchMode] = useState(false);
  const [selectedGameIds, setSelectedGameIds] = useState<string[]>([]);
  const [allCategories, setAllCategories] = useState<vo.CategoryVO[]>([]);
  const [isBatchCategoryModalOpen, setIsBatchCategoryModalOpen] = useState(false);
  const [confirmConfig, setConfirmConfig] = useState<{
    isOpen: boolean;
    title: string;
    message: string;
    type: "danger" | "info";
    onConfirm: () => void;
  }>({
    isOpen: false,
    title: "",
    message: "",
    type: "info",
    onConfirm: () => { },
  });
  const [filterExpanded, setFilterExpanded] = useState(true);
  const gamesForUpdate = useRef(games)
  const [releaseStartDate, setReleaseStartDate] = useState<string>("");
  const [releaseEndDate, setReleaseEndDate] = useState<string>("");
  

  

  
  useEffect(() => {
      const unlistenTaskUpdate = EventsOn("game_updates", (data: any) => {
        
          // 注意：使用正确的语法从data对象获取值
          const task : models.TaskNotice = new models.TaskNotice(data);
          
          // console.log("received taskid:" + task.id + " current taskid:" + taskId + ", item_id:" + task.item_id +  " item_status:" + task.item_status)
          if (task.item_status === enums.TaskStatus.COMPLETED && task.item_id !== "") {
                  const newGame : models.Game = task.item_data as models.Game;
                  console.log("newGame:", newGame)
                  const newGames = [...games]
                  const game = arrayFind(newGames, (it) => it.id === task.item_id)
                  if (game) {
                    const index = newGames.indexOf(game)
                    
                    newGames[index] = newGame
                    setGames(newGames)
                  }
                  
  
              }
          
          
          
      });
  
      return () => {
          if (unlistenTaskUpdate) {
          unlistenTaskUpdate(); // 取消事件监听
          }
      };
      }, [games]);

  
  

  // 延迟显示骨架屏
  useEffect(() => {
    let timer: number;
    if (gamesLoading) {
      timer = window.setTimeout(() => {
        setShowSkeleton(true);
      }, 300);
    }
    else {
      setShowSkeleton(false);
    }
    return () => clearTimeout(timer);
  }, [gamesLoading]);

  const filteredGames = games
    .filter((game) => {
      // 搜索过滤：同时匹配游戏名和开发商/公司
      if (searchQuery) {
        const q = searchQuery.toLowerCase();
        const matchName = game.name.toLowerCase().includes(q);
        const matchCompany = (game.company || "").toLowerCase().includes(q);
        if (!matchName && !matchCompany)
          return false;
      }
      // 状态过滤
      if (statusFilter && game.status !== statusFilter) {
        return false;
      }
      //标签过滤
      if (tagsFilter && tagsFilter.length > 0) {
        const tags = game.tags.split(",");
        if (!tags.some((tag) => tagsFilter.includes(tag))) {
          return false;
        }
      }
      // 发售日期过滤
      if (releaseStartDate && game.release_at) {
        const gameDate = new Date(formatLocalDate(game.release_at));
        const startDate = new Date(releaseStartDate);
        if (gameDate < startDate) {
          return false;
        }
      }
      if (releaseEndDate && game.release_at) {
        const gameDate = new Date(formatLocalDate(game.release_at));
        const endDate = new Date(releaseEndDate);
        // 将结束日期设置为当天的最后一刻
        endDate.setHours(23, 59, 59, 999);
        if (gameDate > endDate) {
          return false;
        }
      }
      return true;
    })
    .sort((a, b) => {
      let comparison = 0;
      switch (sortBy) {
        case "name":
          comparison = a.name.localeCompare(b.name);
          break;
        case "created_at":
          comparison = String(a.created_at || "").localeCompare(String(b.created_at || ""));
          break;
        case "release_at":
          comparison = String(a.release_at || "").localeCompare(String(b.release_at || ""));
      }
      return sortOrder === "asc" ? comparison : -comparison;
    });

  const handleBatchModeChange = (enabled: boolean) => {
    setBatchMode(enabled);
    if (!enabled) {
      setSelectedGameIds([]);
    }
  };

  const setGameSelection = (gameId: string, selected: boolean) => {
    setSelectedGameIds((prev) => {
      if (selected) {
        return prev.includes(gameId) ? prev : [...prev, gameId];
      }
      return prev.filter(id => id !== gameId);
    });
  };

  const handleSelectAll = () => {
    setSelectedGameIds((prev) => {
      const next = new Set(prev);
      filteredGames.forEach((game) => {
        if (game.id) {
          next.add(game.id);
        }
      });
      return Array.from(next);
    });
  };

  const handleClearSelection = () => {
    setSelectedGameIds([]);
  };

  const statusConfig = {
    [enums.GameStatus.NOT_STARTED]: { label: "未开始", icon: "i-mdi-clock-outline", color: "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300" },
    [enums.GameStatus.PLAYING]: { label: "游玩中", icon: "i-mdi-gamepad-variant", color: "bg-neutral-100 text-neutral-700 dark:bg-neutral-900 dark:text-neutral-300" },
    [enums.GameStatus.COMPLETED]: { label: "已通关", icon: "i-mdi-trophy", color: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300" },
    [enums.GameStatus.ON_HOLD]: { label: "搁置", icon: "i-mdi-pause-circle-outline", color: "bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300" },
  };

  const handleBatchStatusUpdate = async (newStatus: string) => {
    if (selectedGameIds.length === 0)
      return;
    try {
      await BatchUpdateStatus(selectedGameIds, newStatus);
      await fetchGames();
      const label = statusConfig[newStatus as keyof typeof statusConfig]?.label ?? newStatus;
      toast.success(`已将 ${selectedGameIds.length} 个游戏状态更新为「${label}」`);
    }
    catch (error) {
      console.error("Failed to batch update status:", error);
      toast.error("批量更新状态失败");
    }
  };

  const openBatchAddModal = async () => {
    if (selectedGameIds.length === 0)
      return;
    try {
      const result = await GetCategories();
      setAllCategories(result || []);
      setIsBatchCategoryModalOpen(true);
    }
    catch (error) {
      console.error("Failed to load categories:", error);
      toast.error("加载收藏夹失败");
    }
  };

  const handleBatchAddToCategory = async (categoryIds: string[]) => {
    if (selectedGameIds.length === 0 || categoryIds.length === 0)
      return;
    try {
      await AddGamesToCategories(selectedGameIds, categoryIds);
      toast.success(`已添加 ${selectedGameIds.length} 个游戏到收藏`);
      setSelectedGameIds([]);
      setBatchMode(false);
    }
    catch (error) {
      console.error("Failed to batch add games to category:", error);
      toast.error("批量添加失败");
    }
  };

  const handleBatchDelete = () => {
    if (selectedGameIds.length === 0)
      return;
    setConfirmConfig({
      isOpen: true,
      title: "批量删除游戏",
      message: `确定要删除选中的 ${selectedGameIds.length} 个游戏吗？此操作将从库中移除这些游戏，但不会删除本地游戏文件。`,
      type: "danger",
      onConfirm: async () => {
        try {
          await DeleteGames(selectedGameIds);
          await fetchGames();
          setSelectedGameIds([]);
          setBatchMode(false);
          toast.success("批量删除成功");
        }
        catch (error) {
          console.error("Failed to batch delete games:", error);
          toast.error("批量删除失败");
        }
      },
    });
  };

  useEffect(() => {
    fetchGames();
  }, [fetchGames]);

  if (gamesLoading && games.length === 0) {
    if (!showSkeleton) {
      return null;
    }
    return <LibrarySkeleton />;
  }

  return (
    <div className={`space-y-6 max-w-8xl mx-auto p-8 transition-opacity duration-300 ${gamesLoading ? "opacity-50 pointer-events-none" : "opacity-100"}`}>
      <div className="flex items-center justify-between">
        <h1 className="text-4xl font-bold text-brand-900 dark:text-white">游戏库</h1>

        <TaskPanel />
      </div>

      <FilterBar
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        searchPlaceholder="搜索游戏..."
        sortBy={sortBy}
        onSortByChange={val => setSortBy(val as "name" | "created_at" | "release_at")}
        sortOptions={sortOptions}
        sortOrder={sortOrder}
        onSortOrderChange={setSortOrder}
        statusFilter={statusFilter}
        onStatusFilterChange={setStatusFilter}
        onTagsFilterChange={setTags}
        tagsLoaded={tagsLoaded}
        tagsFilter={tagsFilter}
        filterExpanded={filterExpanded}
        statusOptions={statusOptions}
        releaseStartDate={releaseStartDate}
        onReleaseStartDateChange={setReleaseStartDate}
        releaseEndDate={releaseEndDate}
        onReleaseEndDateChange={setReleaseEndDate}
        storageKey="library"
        batchMode={batchMode}
        onBatchModeChange={handleBatchModeChange}
        selectedCount={selectedGameIds.length}
        onSelectAll={handleSelectAll}
        onClearSelection={handleClearSelection}
        batchActions={(
          <>
            {/* 批量更新状态 */}
            <BetterDropdownMenu
              title="设为状态"
              align="end"
              menuWidth="min-w-[130px]"
              disabled={selectedGameIds.length === 0}
              trigger={(
                <div
                  title="批量更新状态"
                  className={`glass-panel flex items-center gap-2 px-3 py-2 text-sm
                              bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700
                              rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-700 dark:text-brand-300
                              ${selectedGameIds.length === 0 ? "opacity-50 cursor-not-allowed" : ""}`}
                >
                  <div className="i-mdi-tag-edit-outline text-lg" />
                </div>
              )}
              items={Object.entries(statusConfig).map(([key, cfg]) => ({
                key,
                label: cfg.label,
                icon: cfg.icon,
                pill: true,
                pillColor: cfg.color,
                onClick: () => handleBatchStatusUpdate(key),
              }))}
            />
            {/* 批量添加到收藏 */}
            <button
              type="button"
              onClick={openBatchAddModal}
              disabled={selectedGameIds.length === 0}
              title="批量添加到收藏"
              className={`glass-panel flex items-center gap-2 px-3 py-2 text-sm
                          bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700
                          rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-700 dark:text-brand-300
                          ${selectedGameIds.length === 0 ? "opacity-50 cursor-not-allowed" : ""}`}
            >
              <div className="i-mdi-folder-plus-outline text-lg" />
            </button>
            {/* 批量删除 */}
            <button
              type="button"
              onClick={handleBatchDelete}
              disabled={selectedGameIds.length === 0}
              title="批量删除"
              className={`glass-panel flex items-center gap-2 px-3 py-2 text-sm
                          bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700
                          rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-error-600 dark:text-error-400
                          ${selectedGameIds.length === 0 ? "opacity-50 cursor-not-allowed" : ""}`}
            >
              <div className="i-mdi-delete text-lg" />
            </button>
          </>
        )}
        actionButton={(
          <BetterDropdownMenu
            align="end"
            menuWidth="min-w-[220px]"
            trigger={(
              <div className="glass-btn-neutral flex items-center rounded-lg bg-neutral-600 px-4 py-2 text-sm font-medium text-white hover:bg-neutral-700 focus:outline-none focus:ring-4 focus:ring-neutral-300 dark:bg-neutral-600 dark:hover:bg-neutral-700 dark:focus:ring-neutral-800">
                <div className="i-mdi-plus mr-2 text-lg" />
                添加游戏
                <div className="i-mdi-chevron-down ml-2 text-lg" />
              </div>
            )}
            items={[
              {
                key: "manual",
                label: "手动添加",
                description: "选择可执行文件并搜索元数据",
                icon: "i-mdi-gamepad-variant",
                iconColor: "text-neutral-500",
                onClick: () => setIsAddGameModalOpen(true),
              },
              {
                key: "batch",
                label: "批量导入",
                description: "扫描游戏库目录批量添加",
                icon: "i-mdi-folder-multiple",
                iconColor: "text-success-500",
                onClick: () => setIsBatchImportOpen(true),
              },
              {
                key: "potatovn",
                label: "从 PotatoVN 导入",
                description: "导入 PotatoVN 导出的 ZIP 文件",
                icon: "i-mdi-database-import",
                iconColor: "text-orange-500",
                dividerBefore: true,
                onClick: () => setImportSource("potatovn"),
              },
              {
                key: "playnite",
                label: "从 Playnite 导入",
                description: "导入 Playnite 导出的 JSON 文件",
                icon: "i-mdi-application-import",
                iconColor: "text-purple-500",
                onClick: () => setImportSource("playnite"),
              },
            ]}
          />
        )}
      />

      {games.length === 0
        ? (
            <div className="flex-1 flex items-center justify-center w-full">
              <div className="flex flex-col items-center justify-center py-20 text-brand-500 dark:text-brand-400">
                <div className="i-mdi-gamepad-variant-outline text-6xl mb-4" />
                <p className="text-xl">暂无游戏</p>
                <p className="text-sm mt-2">添加一些GAL游戏开始吧</p>
                <div className="flex flex-col gap-3 mt-4">
                  <button
                    onClick={() => setImportSource("potatovn")}
                    className="rounded-lg border border-success-600 px-5 py-2.5 text-sm font-medium text-success-600 hover:bg-success-50 focus:outline-none focus:ring-4 focus:ring-success-300 dark:border-success-500 dark:text-success-500 dark:hover:bg-success-900/20"
                  >
                    从 PotatoVN 导入
                  </button>
                  <button
                    onClick={() => setImportSource("playnite")}
                    className="rounded-lg border border-purple-600 px-5 py-2.5 text-sm font-medium text-purple-600 hover:bg-purple-50 focus:outline-none focus:ring-4 focus:ring-purple-300 dark:border-purple-500 dark:text-purple-500 dark:hover:bg-purple-900/20"
                  >
                    从 Playnite 导入
                  </button>
                </div>
              </div>
            </div>
          )
        : filteredGames.length === 0
          ? (
              <div className="flex-1 flex items-center justify-center w-full text-brand-500 dark:text-brand-400">
                <div className="flex flex-col items-center">
                  <div className="i-mdi-magnify text-4xl mb-2" />
                  <p>未找到匹配的游戏</p>
                </div>
              </div>
            )
          : (
              <div className="grid grid-cols-[repeat(auto-fill,minmax(8.75rem,1fr))] gap-3">
                {filteredGames.map(game => (
                  <GameCard
                    key={game.id}
                    game={game}
                    searchQuery={searchQuery}
                    selectionMode={batchMode}
                    selected={selectedGameIds.includes(game.id)}
                    onSelectChange={selected => setGameSelection(game.id, selected)}
                  />
                ))}
              </div>
            )}

      <AddGameModal
        isOpen={isAddGameModalOpen}
        onClose={() => setIsAddGameModalOpen(false)}
        onGameAdded={fetchGames}
      />

      <GameImportModal
        isOpen={importSource !== null}
        source={importSource || "potatovn"}
        onClose={() => setImportSource(null)}
        onImportComplete={fetchGames}
      />

      <BatchImportModal
        isOpen={isBatchImportOpen}
        onClose={() => setIsBatchImportOpen(false)}
        onImportComplete={fetchGames}
        onOpenUpdate={(res) => {
          gamesForUpdate.current = res;
          setIsBatchUpdateOpen(true);
          fetchGames();
        }}
      />

      <AddToCategoryModal
        isOpen={isBatchCategoryModalOpen}
        allCategories={allCategories}
        initialSelectedIds={[]}
        onClose={() => setIsBatchCategoryModalOpen(false)}
        onSave={handleBatchAddToCategory}
        title="批量添加到收藏"
        confirmText="添加"
      />

      <BatchUpdateModal
        isOpen={isBatchUpdateOpen}
        onClose={() => setIsBatchUpdateOpen(false)}
        onUpdateComplete={fetchGames}
        games={gamesForUpdate.current}
      />

      <ConfirmModal
        isOpen={confirmConfig.isOpen}
        title={confirmConfig.title}
        message={confirmConfig.message}
        type={confirmConfig.type}
        onClose={() => setConfirmConfig({ ...confirmConfig, isOpen: false })}
        onConfirm={confirmConfig.onConfirm}
      />
    </div>
  );
}
