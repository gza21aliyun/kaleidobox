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
import { arrayFind, arrayMapString, joinString } from "../components/utils/Utility";
import { formatLocalDate } from "../utils/time";
import { useTranslation } from 'react-i18next';

import { enums, vo } from "../../wailsjs/go/models";
import { BatchUpdateModal } from "../components/modal/BatchUpdateModal";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/library",
  component: LibraryPage,
  // loader: async ({ context }) => {
  //   //只在没有缓存数据时加载
  //   const cachedGames = useAppStore.getState().games;
  //   if (cachedGames && cachedGames.length > 0) {
  //     return { games: cachedGames };
  //   }
  //   const games = await GetGames();
  //   return { games };
  // },
  shouldReload: false, // 关键：禁止自动重新加载
});



function LibraryPage() {
  const { t } = useTranslation();
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
  const [tagsFilter, setTags] = useState<string[]>(() => {
    const savedTagsFilter = localStorage.getItem('libraryTagsFilter');
    return savedTagsFilter ? JSON.parse(savedTagsFilter) : [];
  });
  
  const [filterExpanded, setFilterExpanded] = useState(() => {
    return tagsFilter.length == 0;
  });
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
  // const gamesForUpdate = useRef(games)
  const [releaseStartDate, setReleaseStartDate] = useState<string>("");
  const [releaseEndDate, setReleaseEndDate] = useState<string>("");
  const [viewMode, setViewMode] = useState<"list" | "small" | "large">(() => {
    const savedViewMode = localStorage.getItem('libraryViewMode');
    return (savedViewMode as "list" | "small" | "large") || "small";
  });
  

  

  


  
  

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

  // 保存 viewMode 到 localStorage
  useEffect(() => {
    localStorage.setItem('libraryViewMode', viewMode);
  }, [viewMode]);

  // 保存 tagsFilter 到 localStorage
  useEffect(() => {
    localStorage.setItem('libraryTagsFilter', JSON.stringify(tagsFilter));
  }, [tagsFilter]);

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


  // 获取URL参数中的标签
  useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search);
    const tagsParam = urlParams.get('tags');
    console.log("tagsParam 01", tagsParam);
    if (tagsParam) {
      const tag = decodeURIComponent(tagsParam)
        .replace(/"/g, "");
      console.log("tagsParam 02", tag);
      setTags([tag]);
      setFilterExpanded(false)
    }
  }, []);

  const filterSelected = filteredGames.filter(game => selectedGameIds.includes(game.id))
  const filterSelectedIds = filterSelected.map(game => game.id)

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
    [enums.GameStatus.NOT_STARTED]: { label: t('library.gameStatus.not_started'), icon: "i-mdi-clock-outline", color: "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300" },
    [enums.GameStatus.PLAYING]: { label: t('library.gameStatus.playing'), icon: "i-mdi-gamepad-variant", color: "bg-neutral-100 text-neutral-700 dark:bg-neutral-900 dark:text-neutral-300" },
    [enums.GameStatus.COMPLETED]: { label: t('library.gameStatus.completed'), icon: "i-mdi-trophy", color: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300" },
    [enums.GameStatus.ON_HOLD]: { label: t('library.gameStatus.on_hold'), icon: "i-mdi-pause-circle-outline", color: "bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300" },
  };

  const handleBatchStatusUpdate = async (newStatus: string) => {
    if (filterSelectedIds.length === 0)
      return;
    try {
      await BatchUpdateStatus(filterSelectedIds, newStatus);
      await fetchGames();
      const label = statusConfig[newStatus as keyof typeof statusConfig]?.label ?? newStatus;
      toast.success(t('library.toasts.batchUpdateSuccess', { count: filterSelectedIds.length, label }));
    }
    catch (error) {
      console.error("Failed to batch update status:", error);
      toast.error(t('library.toasts.batchUpdateFailed'));
    }
  };

  const openBatchAddModal = async () => {
    if (filterSelectedIds.length === 0)
      return;
    try {
      const result = await GetCategories();
      setAllCategories(result || []);
      setIsBatchCategoryModalOpen(true);
    }
    catch (error) {
      console.error("Failed to load categories:", error);
      toast.error(t('library.toasts.loadCategoriesFailed'));
    }
  };

  const handleBatchAddToCategory = async (categoryIds: string[]) => {
    if (filterSelectedIds.length === 0 || categoryIds.length === 0)
      return;
    try {
      await AddGamesToCategories(filterSelectedIds, categoryIds);
      toast.success(t('library.toasts.batchAddToCollectionSuccess', { count: filterSelectedIds.length }));
      setSelectedGameIds([]);
      setBatchMode(false);
    }
    catch (error) {
      console.error("Failed to batch add games to category:", error);
      toast.error(t('library.toasts.batchAddFailed'));
    }
  };

  const handleBatchDelete = () => {
    if (filterSelectedIds.length === 0)
      return;
    setConfirmConfig({
      isOpen: true,
      title: t('library.modals.batchDeleteTitle'),
      message: t('library.modals.batchDeleteMessage', { count: filterSelectedIds.length }),
      type: "danger",
      onConfirm: async () => {
        try {
          await DeleteGames(filterSelectedIds);
          await fetchGames();
          setSelectedGameIds([]);
          setBatchMode(false);
          toast.success(t('library.toasts.batchDeleteSuccess'));
        }
        catch (error) {
          console.error("Failed to batch delete games:", error);
          toast.error(t('library.toasts.batchDeleteFailed'));
        }
      },
    });
  };

  const loadGames = async () => {
    try {
      const result = await fetchGames();
      // setGames(result || []);
      // const tags: string[] = [];
      // result?.forEach((game) => {
      //   const gameTags = game.tags?.split(",") || [];
      //   tags.push(...gameTags);
      // });
      // const uniqueTags : string[] = [...new Set(tags.map(tag => tag.trim()))];
      // setTagsLoaded(uniqueTags);
      

    }
    catch (error) {
      console.error("Failed to load games:", error);
    }
    finally {
      // setIsLoading(false);
    }
  };

  useEffect(() => {
    if (games.length === 0) {
      loadGames();
    }
    ListTags().then(tags => {
      const map = arrayToMap(tags, tag => tag.category);
      console.log("loadgames tags", map);
      
      
      setTagsLoaded(map);

    });
      
  }, []);





  if (gamesLoading && games.length === 0) {
    if (!showSkeleton) {
      return null;
    }
    return <LibrarySkeleton />;
  }

  
      console.log("rt tagsFilter:", tagsFilter)

  return (
    <div className={`space-y-6 max-w-8xl mx-auto p-8 transition-opacity duration-300 ${gamesLoading ? "opacity-50 pointer-events-none" : "opacity-100"}`}>
      <div className="flex items-center justify-between">
        <h1 className="text-4xl font-bold text-brand-900 dark:text-white">{t('library.title')}</h1>
      </div>

      <FilterBar
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        searchPlaceholder={t('library.searchPlaceholder')}
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
        selectedCount={filterSelectedIds.length}
        onSelectAll={handleSelectAll}
        onClearSelection={handleClearSelection}
        viewMode={viewMode}
        onViewModeChange={setViewMode}
        batchActions={(
          <>
            {/* 批量更新状态 */}
            <BetterDropdownMenu
              title={t('library.buttons.setStatus')}
              align="end"
              menuWidth="min-w-[130px]"
              disabled={filterSelectedIds.length === 0}
              trigger={(
                <div
                  title={t('library.buttons.batchUpdateStatus')}
                  className={`glass-panel flex items-center gap-2 px-3 py-2 text-sm
                              bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700
                              rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-700 dark:text-brand-300
                              ${filterSelectedIds.length === 0 ? "opacity-50 cursor-not-allowed" : ""}`}
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
              disabled={filterSelectedIds.length === 0}
              title={t('library.buttons.batchAddToCollection')}
              className={`glass-panel flex items-center gap-2 px-3 py-2 text-sm
                          bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700
                          rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-700 dark:text-brand-300
                          ${filterSelectedIds.length === 0 ? "opacity-50 cursor-not-allowed" : ""}`}
            >
              <div className="i-mdi-folder-plus-outline text-lg" />
            </button>
            <button
              type="button"
              onClick={() => {setIsBatchUpdateOpen(true)}}
              disabled={filterSelectedIds.length === 0}
              title={t('library.buttons.updateLibrary')}
              className={`glass-panel flex items-center gap-2 px-3 py-2 text-sm
                          bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700
                          rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-700 dark:text-brand-300
                          ${filterSelectedIds.length === 0 ? "opacity-50 cursor-not-allowed" : ""}`}
            >
              <div className="i-mdi-folder-multiple text-lg" />
            </button>
            {/* 批量删除 */}
            <button
              type="button"
              onClick={handleBatchDelete}
              disabled={filterSelectedIds.length === 0}
              title={t('library.buttons.batchDelete')}
              className={`glass-panel flex items-center gap-2 px-3 py-2 text-sm
                          bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700
                          rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-error-600 dark:text-error-400
                          ${filterSelectedIds.length === 0 ? "opacity-50 cursor-not-allowed" : ""}`}
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
                {t('library.addGame')}
                <div className="i-mdi-chevron-down ml-2 text-lg" />
              </div>
            )}
            items={[
              {
                key: "manual",
                label: t('library.buttons.addManually'),
                description: t('library.descriptions.manualAdd'),
                icon: "i-mdi-gamepad-variant",
                iconColor: "text-neutral-500",
                onClick: () => setIsAddGameModalOpen(true),
              },
              {
                key: "batch",
                label: t('library.buttons.bulkImport'),
                description: t('library.descriptions.bulkImport'),
                icon: "i-mdi-folder-multiple",
                iconColor: "text-success-500",
                onClick: () => setIsBatchImportOpen(true),
              },
              {
                key: "potatovn",
                label: t('library.buttons.importFromPotatoVN'),
                description: t('library.descriptions.importPotatoVN'),
                icon: "i-mdi-database-import",
                iconColor: "text-orange-500",
                dividerBefore: true,
                onClick: () => setImportSource("potatovn"),
              },
              {
                key: "playnite",
                label: t('library.buttons.importFromPlaynite'),
                description: t('library.descriptions.importPlaynite'),
                icon: "i-mdi-application-import",
                iconColor: "text-purple-500",
                onClick: () => setImportSource("playnite"),
              },
              {
                key: "update",
                label: t('library.buttons.updateLibrary'),
                description: t('library.descriptions.updateLibrary'),
                icon: "i-mdi-folder-multiple",
                iconColor: "text-blue-500",
                onClick: () => {
                  if (filterSelectedIds.length === 0 || !batchMode) {
                    setSelectedGameIds(games.map((game) => game.id))
                  }
                  // gamesForUpdate.current = games;
                  setIsBatchUpdateOpen(true);
                },
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
                <p className="text-xl">{t('library.emptyState.noGames')}</p>
                <p className="text-sm mt-2">{t('library.emptyState.addSomeGames')}</p>
                <div className="flex flex-col gap-3 mt-4">
                  <button
                    onClick={() => setImportSource("potatovn")}
                    className="rounded-lg border border-success-600 px-5 py-2.5 text-sm font-medium text-success-600 hover:bg-success-50 focus:outline-none focus:ring-4 focus:ring-success-300 dark:border-success-500 dark:text-success-500 dark:hover:bg-success-900/20"
                  >
                    {t('library.buttons.importFromPotatoVN')}
                  </button>
                  <button
                    onClick={() => setImportSource("playnite")}
                    className="rounded-lg border border-purple-600 px-5 py-2.5 text-sm font-medium text-purple-600 hover:bg-purple-50 focus:outline-none focus:ring-4 focus:ring-purple-300 dark:border-purple-500 dark:text-purple-500 dark:hover:bg-purple-900/20"
                  >
                    {t('library.buttons.importFromPlaynite')}
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
                  <p>{t('library.emptyState.noMatchingGames')}</p>
                </div>
              </div>
            )
          : (
              <div className={
                viewMode === "list" 
                  ? "flex flex-col gap-2"
                  : viewMode === "large"
                    ? "grid grid-cols-[repeat(auto-fill,minmax(19rem,1fr))] gap-4"
                    : "grid grid-cols-[repeat(auto-fill,minmax(8.75rem,1fr))] gap-3"
              }>
                {filteredGames.map(game => (
                  <GameCard
                    key={game.id}
                    game={game}
                    searchQuery={searchQuery}
                    selectionMode={batchMode}
                    selected={selectedGameIds.includes(game.id)}
                    onSelectChange={selected => setGameSelection(game.id, selected)}
                    filteredGameIdsStr={arrayMapString(filteredGames, (game) => game.id)}
                    viewMode={viewMode}
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
          console.log("onOpenUpdate res:", res)
          // gamesForUpdate.current = res;
          fetchGames().then(() => { 
            setSelectedGameIds(res.map((g) => g.id))
            setIsBatchUpdateOpen(true);
          })
          
        }}
      />

      <AddToCategoryModal
        isOpen={isBatchCategoryModalOpen}
        allCategories={allCategories}
        initialSelectedIds={[]}
        onClose={() => setIsBatchCategoryModalOpen(false)}
        onSave={handleBatchAddToCategory}
        title={t('library.modals.batchAddToCollectionTitle')}
        confirmText={t('library.buttons.confirmAdd')}
      />

      <BatchUpdateModal
        isOpen={isBatchUpdateOpen}
        onClose={() => setIsBatchUpdateOpen(false)}
        onUpdateComplete={loadGames}
        games={filterSelected}
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
