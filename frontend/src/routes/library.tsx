import { models } from "../../wailsjs/go/models";
import type { ImportSource } from "../components/modal/GameImportModal";
import { createRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useRef, useState, useCallback, useMemo } from "react";
import { GetGames } from "../../wailsjs/go/service/GameService";
import { ListTags } from "../../wailsjs/go/service/TagService";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import { toast } from "react-hot-toast";
import { AddGamesToCategories, GetCategories } from "../../wailsjs/go/service/CategoryService";
import { BatchUpdateStatus, DeleteGames } from "../../wailsjs/go/service/GameService";
import { FilterBar } from "../components/bar/FilterBar";
import { GameCard } from "../components/card/GameCard";
import { useVirtualizer } from "@tanstack/react-virtual";
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
import { AddTagModal } from "../components/modal/AddTagModal";
import { v } from "@unocss/preset-wind3/dist/rules-Dd5IWQsx.mjs";
import { FetchEmptyGalleryGames } from "../../wailsjs/go/service/ImageService";

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
  const navigate = useNavigate();
  const { games, gamesLoading, fetchGames, tagsLoaded, gameStats } = useAppStore();
  const [showSkeleton, setShowSkeleton] = useState(false);
  const [isAddGameModalOpen, setIsAddGameModalOpen] = useState(false);
  const [isBatchImportOpen, setIsBatchImportOpen] = useState(false);
  const [isBatchUpdateOpen, setIsBatchUpdateOpen] = useState(false);
  const [importSource, setImportSource] = useState<ImportSource | null>(null);
  const [searchQuery, setSearchQuery] = useState<string>(() => localStorage.getItem('searchQuery') || "");
  const [sortBy, setSortBy] = useState<"name" | "created_at" | "release_at" | "company" | "last_played" | "play_time"> ("created_at");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");
  const [sourceFilter, setSourceFilter] = useState<string>("");
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [includedIds, setIncludedIds] = useState<string[] | null>(null);
  const [tagsFilter, setTags] = useState<string[]>(() => {
    const savedTagsFilter = localStorage.getItem('libraryTagsFilter');
    return savedTagsFilter ? JSON.parse(savedTagsFilter) : [];
  });
  
  const [filterExpanded, setFilterExpanded] = useState(() => {
    return tagsFilter.length == 0;
  });
  const [batchMode, setBatchMode] = useState(false);
  const [selectedGameIds, setSelectedGameIds] = useState<string[]>([]);
  const [lastSelectedGameId, setLastSelectedGameId] = useState<string | null>(null);
  const [allCategories, setAllCategories] = useState<vo.CategoryVO[]>([]);
  const [isBatchCategoryModalOpen, setIsBatchCategoryModalOpen] = useState(false);
  const [isBatchAddTagModalOpen, setIsBatchAddTagModalOpen] = useState(false);
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
  const [tagsIntersectionMode, setTagsIntersectionMode] = useState<boolean>(() => {
    const savedMode = localStorage.getItem('libraryTagsIntersectionMode');
    return savedMode ? JSON.parse(savedMode) : false;
  });
  const [viewMode, setViewMode] = useState<"list" | "small" | "large">(() => {
    const savedViewMode = localStorage.getItem('libraryViewMode');
    return (savedViewMode as "list" | "small" | "large") || "small";
  });
  
  // 选择返回模式
  const [selectMode, setSelectMode] = useState(false);
  const [returnPath, setReturnPath] = useState("/");
  

  

  

  


  
  

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
// 保存标签过滤到本地存储
  useEffect(() => {
    localStorage.setItem('libraryTagsFilter', JSON.stringify(tagsFilter));
  }, [tagsFilter]);

  // 保存标签交集模式到本地存储
  useEffect(() => {
    localStorage.setItem('libraryTagsIntersectionMode', JSON.stringify(tagsIntersectionMode));
  }, [tagsIntersectionMode]);

  const filteredGames = useMemo(() => {
    if (sourceFilter === "emptyGallery") {
      if (includedIds === null) {
        FetchEmptyGalleryGames().then((res) => { 
          setIncludedIds(res);
        });
      }
      
    } else {
      setIncludedIds(null);
    }
    const gs : models.Game[] = games
    .filter((game) => {
      if (includedIds && !includedIds.includes(game.id)) {
        return false;
      }
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
        if (tagsIntersectionMode) {
          // 交集模式：游戏必须包含所有过滤标签
          if (!tagsFilter.every((filterTag) => tags.includes(filterTag))) {
            return false;
          }
        } else {
          // 并集模式：游戏只要包含任意一个过滤标签
          if (!tags.some((tag) => tagsFilter.includes(tag))) {
            return false;
          }
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
      //源匹配过滤
      const sourceValue = game.source_type.toString();
      if (sourceFilter !== "") {
        if (sourceValue === sourceFilter) {
          return true;
        }
        if (sourceFilter === enums.SourceType.DMM.toString() && game.dmm_id && game.dmm_id !== "") {
          return true;
        }
        if (sourceFilter === enums.SourceType.EROSCAPE.toString() && game.eroscape_id && game.eroscape_id !== "") {
          return true;
        }
        if (sourceFilter === enums.SourceType.DLSITE.toString() && game.dlsite_id && game.dlsite_id !== "") {
          return true;
        }
        if (sourceFilter === enums.SourceType.GETCHU.toString() && game.getchu_id && game.getchu_id !== "") {
          return true;
        }
        if (sourceFilter === enums.SourceType.BANGUMI.toString() && game.bangumi_id && game.bangumi_id !== "") {
          return true;
        }
        if (sourceFilter === enums.SourceType.YMGAL.toString() && game.ymgal_id && game.ymgal_id !== "") {
          return true;
        }
        if (sourceFilter === "emptyCover" && (!game.cover_url || game.cover_url === "")) {
          return true;
        }
        if (sourceFilter === "emptyGallery" && (includedIds && includedIds.includes(game.id))) {
          return true;
        }
        return false;
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
          break;
        case "company":
          comparison = String(a.company || "").localeCompare(String(b.company || ""));
          break;
        case "last_played":
          const aEndDate = gameStats.get(a.id)?.end_date || "";
          const bEndDate = gameStats.get(b.id)?.end_date || "";
          comparison = aEndDate.localeCompare(bEndDate);
          break;
        case "play_time":
          const aPlayTime = gameStats.get(a.id)?.total_play_time || 0;
          const bPlayTime = gameStats.get(b.id)?.total_play_time || 0;
          comparison = aPlayTime - bPlayTime;
          break;
      }
      return sortOrder === "asc" ? comparison : -comparison;
    });
    console.log("games:", games);
    console.log("filteredGames:", gs);
    return gs;
  }, [games, sortBy, sortOrder, searchQuery, statusFilter, sourceFilter, tagsFilter, releaseStartDate, releaseEndDate, includedIds, tagsIntersectionMode, gameStats]);

  // const filteredGames = useMemo(() => { 
  //   return filteredGamesList();
  // }, [games, sortBy, sortOrder, searchQuery, statusFilter, sourceFilter, tagsFilter, releaseStartDate, releaseEndDate]);
  
  // 虚拟滚动相关
  const containerRef = useRef<HTMLDivElement>(null);
  
  // 使用 react-virtual 实现虚拟滚动
  const virtual = useVirtualizer({
    count: filteredGames.length,
    getScrollElement: () => containerRef.current,
    estimateSize: () => 100, // 初始估计值，后续会通过 measureElement 校正
    overscan: 3, // 预加载的项目数量
    paddingStart: 0,
    paddingEnd: 0,
  });

  // 获取URL参数中的标签和选择模式
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
    
    // 检查是否为选择模式
    const selectModeParam = urlParams.get('selectMode');
    const returnPathParam = urlParams.get('returnPath');
    if (selectModeParam === 'true' && returnPathParam) {
      // 解码 returnPath
      const decodedReturnPath = decodeURIComponent(returnPathParam);
      console.log("Decoded returnPath:", decodedReturnPath);
      setSelectMode(true);
      setReturnPath(decodedReturnPath);
      setBatchMode(true); // 自动启用批量选择模式
    }
  }, []);
  
  // 监听视图模式变化和窗口大小变化，更新虚拟滚动
  useEffect(() => {
    const handleResize = () => {
      // 窗口大小变化时，虚拟滚动会自动更新
    };
    
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, [viewMode]);

  const filterSelected = filteredGames.filter(game => selectedGameIds.includes(game.id))
  const filterSelectedIds = filterSelected.map(game => game.id)

  const handleBatchModeChange = (enabled: boolean) => {
    setBatchMode(enabled);
    if (!enabled) {
      setSelectedGameIds([]);
    }
  };

  const setGameSelection = (gameId: string, selected: boolean, event?: React.MouseEvent) => {
    setSelectedGameIds((prev) => {
      const currentIndex = filteredGames.findIndex(game => game.id === gameId);
      const lastIndex = lastSelectedGameId ? filteredGames.findIndex(game => game.id === lastSelectedGameId) : -1;
      
      // 处理 Shift 键：反向选择从上次选中到当前的所有游戏
      if (event?.shiftKey && lastSelectedGameId && lastIndex !== -1 && currentIndex !== -1) {
        const startIndex = Math.min(lastIndex, currentIndex);
        const endIndex = Math.max(lastIndex, currentIndex);
        const gamesInRange = filteredGames.slice(startIndex + 1, endIndex + 1).map(game => game.id);
        
        // 反向选择：已选中的变为未选中，未选中的变为选中
        const newSelection = new Set([...prev]);
        gamesInRange.forEach(gameIdInRange => {
          if (newSelection.has(gameIdInRange)) {
            newSelection.delete(gameIdInRange);
          } else {
            newSelection.add(gameIdInRange);
          }
        });
        return Array.from(newSelection);
      }
      
      // 普通点击：添加或移除当前游戏
      else {
        if (selected) {
          return prev.includes(gameId) ? prev : [...prev, gameId];
        }
        return prev.filter(id => id !== gameId);
      }
    });
    
    // 更新上次选中的游戏
    if (selected) {
      setLastSelectedGameId(gameId);
    }
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
  
  // 处理选择完成
  const handleSelectComplete = () => {
    if (selectMode && returnPath) {
      // 将选中的游戏ID作为URL参数传递回原页面
      const selectedGameIdsStr = selectedGameIds.join(',');
      console.log("handleSelectComplete - selectedGameIdsStr:", selectedGameIdsStr);
      console.log("handleSelectComplete - returnPath:", returnPath);
      
      // 确保 returnPath 是有效的URL
      try {
        const returnUrl = new URL(returnPath);
        
        // 使用对象形式的搜索参数
        const searchObj: Record<string, string> = {};
        // 保留原有的搜索参数
        returnUrl.searchParams.forEach((value, key) => {
          searchObj[key] = value;
        });
        // 添加 selectedGameIds 参数
        searchObj.selectedGameIds = selectedGameIdsStr;
        
        console.log("handleSelectComplete - searchObj:", searchObj);
        
        // 使用对象形式的参数调用 navigate
        navigate({
          to: returnUrl.pathname,
          search: searchObj
        });
      } catch (error) {
        console.error("Invalid returnPath:", error);
        // 如果 returnPath 无效，使用当前路径
        navigate({ 
          to: window.location.pathname, 
          search: { selectedGameIds: selectedGameIdsStr } 
        });
      }
    }
  };

  const statusConfig = {
    [enums.GameStatus.NOT_STARTED]: { label: t('library.gameStatus.not_started'), icon: "i-mdi-clock-outline", color: "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300" },
    [enums.GameStatus.PLAYING]: { label: t('library.gameStatus.playing'), icon: "i-mdi-gamepad-variant", color: "bg-neutral-100 text-neutral-700 dark:bg-neutral-900 dark:text-neutral-300" },
    [enums.GameStatus.COMPLETED]: { label: t('library.gameStatus.completed'), icon: "i-mdi-trophy", color: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300" },
    [enums.GameStatus.ON_HOLD]: { label: t('library.gameStatus.on_hold'), icon: "i-mdi-pause-circle-outline", color: "bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300" },
  };

  const sourceConfig = [
    { label: t('sourceType.noFilter'), value: "" },
    { label: t('sourceType.getchu'), value: enums.SourceType.GETCHU.toString() },
    { label: t('sourceType.eroscape'), value: enums.SourceType.EROSCAPE.toString() },
    { label: t('sourceType.dlsite'), value: enums.SourceType.DLSITE.toString() },
    { label: t('sourceType.ymgal'), value: enums.SourceType.YMGAL.toString() },
    { label: t('sourceType.dmm'), value: enums.SourceType.DMM.toString() },
    { label: t('sourceType.bangumi'), value: enums.SourceType.BANGUMI.toString() },
    { label: t('sourceType.vndb'), value: enums.SourceType.VNDB.toString() },
    { label: t('sourceType.local'), value: enums.SourceType.LOCAL.toString() },
    { label: t('sourceType.emptyCover'), value: "emptyCover" },
    { label: t('sourceType.emptyGallery'), value: "emptyGallery" },
  ];

  const handleBatchStatusUpdate = async (newStatus: string) => {
    if (filterSelectedIds.length === 0)
      return;
    try {
      await BatchUpdateStatus(filterSelectedIds, newStatus);
      await loadGames();
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

  const handleBatchAddTag = async (newGames: models.Game[]) => {
    setSelectedGameIds([]);
    setBatchMode(false);
    setIsBatchAddTagModalOpen(false)
    await loadGames();
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
          await loadGames();
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

      // ListTags().then(tags => {
      //   const map = arrayToMap(tags, tag => tag.category);
      //   console.log("loadgames tags", map);
        
        
      //   setTagsLoaded(map);

      // });
      

    }
    catch (error) {
      console.error("Failed to load games:", error);
    }
    finally {
      // setIsLoading(false);
    }
  };

  const filteredGameIdsStr= useMemo(() => arrayMapString(filteredGames, (game) => game.id), [filteredGames])

  useEffect(() => {
    if (games.length === 0) {
      loadGames();
    }
    
      
  }, []);





  if (gamesLoading && games.length === 0) {
    if (!showSkeleton) {
      return null;
    }
    return <LibrarySkeleton />;
  }

  
      // console.log("rt tagsFilter:", tagsFilter)

  return (
    <div className={`space-y-6 max-w-8xl mx-auto p-8 transition-opacity duration-300 ${gamesLoading ? "opacity-50 pointer-events-none" : "opacity-100"}`}>
      <div className="flex items-center justify-between">
        <h1 className="text-4xl font-bold text-brand-900 dark:text-white">{t('library.title')}({filteredGames.length})</h1>
      </div>

      <FilterBar
        searchQuery={searchQuery}
        onSearchChange={(q) => {setSearchQuery(q);localStorage.setItem('searchQuery', q)}}
        searchPlaceholder={t('library.searchPlaceholder')}
        sortBy={sortBy}
        onSortByChange={val => setSortBy(val as "name" | "created_at" | "release_at" | "company" | "last_played" | "play_time")}
        sortOptions={sortOptions}
        sortOrder={sortOrder}
        onSortOrderChange={setSortOrder}
        statusFilter={statusFilter}
        onStatusFilterChange={setStatusFilter}
        sourceFilter={sourceFilter}
        onSourceFilterChange={setSourceFilter}
        sourceOptions={sourceConfig}
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
        games={filteredGames}
        onBatchModeChange={handleBatchModeChange}
        selectedCount={filterSelectedIds.length}
        onSelectAll={handleSelectAll}
        onClearSelection={handleClearSelection}
        viewMode={viewMode}
        onViewModeChange={setViewMode}
        tagsIntersectionMode={tagsIntersectionMode}
        onTagsIntersectionModeChange={setTagsIntersectionMode}
        batchActions={( 
          <>
            {/* 选择模式下的确认和取消按钮 */}
            {selectMode && (
              <>
                <button
                  type="button"
                  onClick={handleSelectComplete}
                  disabled={filterSelectedIds.length === 0}
                  title={t('library.buttons.confirmSelection')}
                  className={`glass-panel flex items-center gap-2 px-3 py-2 text-sm
                              bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700
                              rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-success-600 dark:text-success-400
                              ${filterSelectedIds.length === 0 ? "opacity-50 cursor-not-allowed" : ""}`}
                >
                  <div className="i-mdi-check-circle-outline text-lg" />
                  {t('library.buttons.confirmSelection')}
                </button>
                <button
                  type="button"
                  onClick={() => {
                    console.log("Cancel button clicked, returnPath:", returnPath);
                    if (returnPath) {
                      try {
                        const returnUrl = new URL(returnPath);
                        console.log("Navigate to:", returnUrl.pathname);
                        // 使用对象形式的参数调用 navigate
                        navigate({
                          to: returnUrl.pathname,
                          search: returnUrl.search
                        });
                      } catch (error) {
                        console.error("Invalid returnPath:", error);
                        // 如果 returnPath 无效，使用首页
                        navigate({ to: '/' });
                      }
                    } else {
                      // 如果没有 returnPath，使用首页
                      navigate({ to: '/' });
                    }
                  }}
                  title={t('common.cancel')}
                  className={`glass-panel flex items-center gap-2 px-3 py-2 text-sm
                              bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700
                              rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-error-600 dark:text-error-400`}
                >
                  <div className="i-mdi-close-circle-outline text-lg" />
                  {t('common.cancel')}
                </button>
              </>
            )}
            {/* 非选择模式下的批量操作 */}
            {!selectMode && (
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
                {/* 批量添加标签 */}
                <button
                  type="button"
                  onClick={() => setIsBatchAddTagModalOpen(true)}
                  disabled={filterSelectedIds.length === 0}
                  title={t('library.buttons.batchAddTags')}
                  className={`glass-panel flex items-center gap-2 px-3 py-2 text-sm
                              bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700
                              rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-700 dark:text-brand-300
                              ${filterSelectedIds.length === 0 ? "opacity-50 cursor-not-allowed" : ""}`}
                >
                  <div className="i-mdi-tag-plus-outline text-lg" />
                </button>
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
          : viewMode === "list"
            ? (
                <div 
                  ref={containerRef}
                  className="flex-1 overflow-y-auto"
                  style={{
                    height: '100%',
                    position: 'relative'
                  }}
                >
                  <div 
                    style={{
                      height: `${virtual.getTotalSize()}px`,
                      width: '100%',
                      position: 'relative'
                    }}
                  >
                    {virtual.getVirtualItems().map((virtualItem) => (
                      <div
                        key={filteredGames[virtualItem.index].id}
                        ref={virtual.measureElement}
                        style={{
                          position: 'absolute',
                          top: 0,
                          left: 0,
                          width: '100%',
                          transform: `translateY(${virtualItem.start}px)`
                        }}
                      >
                        <GameCard
                          game={filteredGames[virtualItem.index]}
                          searchQuery={searchQuery}
                          selectionMode={batchMode}
                          selected={selectedGameIds.includes(filteredGames[virtualItem.index].id)}
                          onSelectChange={(selected, event) => setGameSelection(filteredGames[virtualItem.index].id, selected, event)}
                          filteredGameIdsStr={filteredGameIdsStr}
                          viewMode={viewMode}
                        />
                      </div>
                    ))}
                  </div>
                </div>
              )
            : (
                <div 
                  className="flex-1 overflow-y-auto"
                >
                  <div className={
                    viewMode === "large"
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
                        onSelectChange={(selected, event) => setGameSelection(game.id, selected, event)}
                        filteredGameIdsStr={filteredGameIdsStr}
                        viewMode={viewMode}
                      />
                    ))}
                  </div>
                </div>
              )}

      <AddGameModal
        isOpen={isAddGameModalOpen}
        onClose={() => setIsAddGameModalOpen(false)}
        onGameAdded={loadGames}
      />

      <GameImportModal
        isOpen={importSource !== null}
        source={importSource || "potatovn"}
        onClose={() => setImportSource(null)}
        onImportComplete={loadGames}
      />

      <BatchImportModal
        isOpen={isBatchImportOpen}
        onClose={() => setIsBatchImportOpen(false)}
        onImportComplete={loadGames}
        onOpenUpdate={(res) => {
          console.log("onOpenUpdate res:", res)
          // gamesForUpdate.current = res;
          loadGames().then(() => { 
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

      <AddTagModal
        isOpen={isBatchAddTagModalOpen}
        onClose={() => setIsBatchAddTagModalOpen(false)}
        onConfirm={handleBatchAddTag}
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
