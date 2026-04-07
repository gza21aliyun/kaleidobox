import { useTranslation } from 'react-i18next';
import { models, vo, enums } from "../../wailsjs/go/models";
import { createRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useMemo, useState } from "react";
import { toast } from "react-hot-toast";
import {
  AddGameToCategory,
  AddGamesToCategories,
  GetCategories,
  GetCategoryByID,
  GetGamesByCategory,
  RemoveGameFromCategory,
  RemoveGamesFromCategory,
} from "../../wailsjs/go/service/CategoryService";
import { GetGames, BatchUpdateStatus, DeleteGames } from "../../wailsjs/go/service/GameService";
import { ListTags } from "../../wailsjs/go/service/TagService";
import { FilterBar } from "../components/bar/FilterBar";
import { GameCard } from "../components/card/GameCard";
import { AddGameToCategoryModal } from "../components/modal/AddGameToCategoryModal";
import { AddToCategoryModal } from "../components/modal/AddToCategoryModal";
import { AddTagModal } from "../components/modal/AddTagModal";
import { BatchUpdateModal } from "../components/modal/BatchUpdateModal";
import { ConfirmModal } from "../components/modal/ConfirmModal";
import { CategorySkeleton } from "../components/skeleton/CategorySkeleton";
import { BetterDropdownMenu } from "../components/ui/BetterDropdownMenu";
import { sortOptions, statusOptions } from "../consts/options";
import { Route as rootRoute } from "./__root";
import { arrayToMap } from "../components/utils/Utility";
import { formatLocalDate } from '../utils/time';
import { arrayFind, arrayMapString, joinString } from "../components/utils/Utility";
import { FetchEmptyGalleryGames } from '../../wailsjs/go/service/ImageService';

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/categories/$categoryId",
  component: CategoryDetailPage,
});

function CategoryDetailPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { categoryId } = Route.useParams();
  const [category, setCategory] = useState<vo.CategoryVO | null>(null);
  const [games, setGames] = useState<models.Game[]>([]);
  const [loading, setLoading] = useState(true);
  const [showSkeleton, setShowSkeleton] = useState(false);
  const [isAddGameModalOpen, setIsAddGameModalOpen] = useState(false);
  const [allGames, setAllGames] = useState<models.Game[]>([]);
  const [sourceFilter, setSourceFilter] = useState<string>("");
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState<"name" | "created_at" | "release_at" | "company"> ("created_at");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc"> ("desc");
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [batchMode, setBatchMode] = useState(false);
  const [selectedGameIds, setSelectedGameIds] = useState<string[]>([]);
  const [lastSelectedGameId, setLastSelectedGameId] = useState<string | null>(null);
  const [filterExpanded, setFilterExpanded] = useState(false);
  const [tagsFilter, setTags] = useState<string[]>(() => {
    const savedTagsFilter = localStorage.getItem('categoryTagsFilter');
    return savedTagsFilter ? JSON.parse(savedTagsFilter) : [];
  });
  const [tagsLoaded, setTagsLoaded] = useState<Map<string, models.Tag[]>>(new Map());
  const [releaseStartDate, setReleaseStartDate] = useState<string>("");
  const [releaseEndDate, setReleaseEndDate] = useState<string>("");
  const [viewMode, setViewMode] = useState<"list" | "small" | "large">(() => {
    const savedViewMode = localStorage.getItem('categoryViewMode');
    return (savedViewMode as "list" | "small" | "large") || "small";
  });
  const [tagsIntersectionMode, setTagsIntersectionMode] = useState<boolean>(() => {
    const savedMode = localStorage.getItem('categoryTagsIntersectionMode');
    return savedMode ? JSON.parse(savedMode) : false;
  });
  
  // 批量操作相关状态
  const [allCategories, setAllCategories] = useState<vo.CategoryVO[]>([]);
  const [isBatchCategoryModalOpen, setIsBatchCategoryModalOpen] = useState(false);
  const [isBatchAddTagModalOpen, setIsBatchAddTagModalOpen] = useState(false);
  const [isBatchUpdateOpen, setIsBatchUpdateOpen] = useState(false);
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
  
  // 选择模式相关
  const [selectMode, setSelectMode] = useState(false);
  const [returnPath, setReturnPath] = useState("/");

  // 延迟显示骨架屏
  useEffect(() => {
    let timer: number;
    if (loading) {
      timer = window.setTimeout(() => {
        setShowSkeleton(true);
      }, 300);
    }
    else {
      setShowSkeleton(false);
    }
    return () => clearTimeout(timer);
  }, [loading]);

  // 保存 viewMode 到 localStorage
  useEffect(() => {
    localStorage.setItem('categoryViewMode', viewMode);
  }, [viewMode]);

  // 保存 tagsFilter 到 localStorage
  useEffect(() => {
    localStorage.setItem('categoryTagsFilter', JSON.stringify(tagsFilter));
  }, [tagsFilter]);

  // 保存标签交集模式到本地存储
  useEffect(() => {
    localStorage.setItem('categoryTagsIntersectionMode', JSON.stringify(tagsIntersectionMode));
  }, [tagsIntersectionMode]);

  // 加载标签数据
  useEffect(() => {
    ListTags().then(tags => {
      var tagSet = new Set<string>();
      games.forEach(game => {
        game.tags.split(",").forEach(t1 => {
          tagSet.add(t1);
        });
      });
      var newTags: models.Tag[] = [];
      tags.forEach(tag => {
        if (tagSet.has(tag.name)) {
          newTags.push(tag);
        }
      });
      const map = arrayToMap(newTags, tag => tag.category);
      setTagsLoaded(map);
    });
  }, [games]);

  // 获取URL参数中的选择模式
  useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search);
    
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

  const loadCategory = async (id: string) => {
    try {
      const result = await GetCategoryByID(id);
      setCategory(result);
    }
    catch (error) {
      console.error("Failed to load category:", error);
      toast.error(t('category.toasts.loadCategoriesFailed'));
    }
  };

  const loadGames = async (id: string) => {
    try {
      const result = await GetGamesByCategory(id);
      setGames(result || []);
    }
    catch (error) {
      console.error("Failed to load games for category:", error);
      toast.error(t('category.toasts.loadFolderGamesFailed'));
    }
  };

  const onBack = () => {
    navigate({ to: "/categories" });
  };

  const handleRemoveGame = async (gameId: string) => {
    if (!category)
      return;
    try {
      await RemoveGameFromCategory(gameId, category.id);
      await loadGames(category.id);
      await loadCategory(category.id);
    }
    catch (error) {
      console.error("Failed to remove game from category:", error);
      toast.error(t('category.toasts.removeFromCategoryFailed'));
    }
  };

  const openAddGameModal = async () => {
    try {
      const result = await GetGames();
      const currentGameIds = new Set(games.map(g => g.id));
      setAllGames(result.filter(g => !currentGameIds.has(g.id)) || []);
      setIsAddGameModalOpen(true);
    }
    catch (error) {
      console.error("Failed to load all games:", error);
      toast.error(t('category.toasts.loadLibraryGamesFailed'));
    }
  };

  const handleAddGameToCategory = async (gameId: string) => {
    if (!category)
      return;
    try {
      await AddGameToCategory(gameId, category.id);
      setAllGames(prev => prev.filter(g => g.id !== gameId));
      await loadGames(category.id);
      await loadCategory(category.id);
    }
    catch (error) {
      console.error("Failed to add game to category:", error);
      toast.error(t('category.toasts.addGameToCategoryFailed'));
    }
  };

  const [includedIds, setIncludedIds] = useState<string[] | null>(null);

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
    return games
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
      // 标签过滤
      if (tagsFilter && tagsFilter.length > 0) {
        const tags = game.tags ? game.tags.split(",") : [];
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
      // 源匹配过滤
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
      }
      return sortOrder === "asc" ? comparison : -comparison;
    });
  }, [games, searchQuery, statusFilter, tagsFilter, tagsIntersectionMode, releaseStartDate, releaseEndDate, sourceFilter, sortBy, sortOrder, includedIds]);

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

  const handleBatchRemove = async () => {
    if (!category || selectedGameIds.length === 0)
      return;
    try {
      await RemoveGamesFromCategory(selectedGameIds, category.id);
      await Promise.all([loadGames(category.id), loadCategory(category.id)]);
      toast.success(t('category.toasts.batchRemoveSuccess', { count: selectedGameIds.length }));
      setSelectedGameIds([]);
      setBatchMode(false);
    }
    catch (error) {
      console.error("Failed to batch remove games:", error);
      toast.error(t('category.toasts.batchRemoveFailed'));
    }
  };

  const statusConfig = {
    [enums.GameStatus.NOT_STARTED]: { label: t('library.gameStatus.not_started'), icon: "i-mdi-clock-outline", color: "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300" },
    [enums.GameStatus.PLAYING]: { label: t('library.gameStatus.playing'), icon: "i-mdi-gamepad-variant", color: "bg-neutral-100 text-neutral-700 dark:bg-neutral-900 dark:text-neutral-300" },
    [enums.GameStatus.COMPLETED]: { label: t('library.gameStatus.completed'), icon: "i-mdi-trophy", color: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300" },
    [enums.GameStatus.ON_HOLD]: { label: t('library.gameStatus.on_hold'), icon: "i-mdi-pause-circle-outline", color: "bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300" },
  };

  const handleBatchStatusUpdate = async (newStatus: string) => {
    if (selectedGameIds.length === 0)
      return;
    try {
      await BatchUpdateStatus(selectedGameIds, newStatus);
      await loadGames(categoryId);
      const label = statusConfig[newStatus as keyof typeof statusConfig]?.label ?? newStatus;
      toast.success(t('library.toasts.batchUpdateSuccess', { count: selectedGameIds.length, label }));
    }
    catch (error) {
      console.error("Failed to batch update status:", error);
      toast.error(t('library.toasts.batchUpdateFailed'));
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
      toast.error(t('library.toasts.loadCategoriesFailed'));
    }
  };

  const handleBatchAddToCategory = async (categoryIds: string[]) => {
    if (selectedGameIds.length === 0 || categoryIds.length === 0)
      return;
    try {
      await AddGamesToCategories(selectedGameIds, categoryIds);
      toast.success(t('library.toasts.batchAddToCollectionSuccess', { count: selectedGameIds.length }));
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
    setIsBatchAddTagModalOpen(false);
    await loadGames(categoryId);
  };

  const handleBatchDelete = () => {
    if (selectedGameIds.length === 0)
      return;
    setConfirmConfig({
      isOpen: true,
      title: t('library.modals.batchDeleteTitle'),
      message: t('library.modals.batchDeleteMessage', { count: selectedGameIds.length }),
      type: "danger",
      onConfirm: async () => {
        try {
          await DeleteGames(selectedGameIds);
          await loadGames(categoryId);
          await loadCategory(categoryId);
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

  useEffect(() => {
    if (categoryId) {
      const init = async () => {
        setLoading(true);
        setBatchMode(false);
        setSelectedGameIds([]);
        await Promise.all([loadCategory(categoryId), loadGames(categoryId)]);
        setLoading(false);
      };
      init();
    }
  }, [categoryId]);

  if (loading && !category) {
    if (!showSkeleton) {
      return null;
    }
    return <CategorySkeleton />;
  }

  if (!category) {
    return (
      <div className="flex flex-col items-center justify-center h-full space-y-4 text-brand-500">
        <div className="i-mdi-alert-circle-outline text-6xl" />
        <p className="text-xl">{t('category.messages.categoryNotFound', { category: t('nav.categories') })}</p>
        <button onClick={onBack} className="text-neutral-600 hover:underline">{t('category.labels.returnToList')}</button>
      </div>
    );
  }

  return (
    <div className={`h-full w-full overflow-y-auto p-8 transition-opacity duration-300 ${loading ? "opacity-50 pointer-events-none" : "opacity-100"}`}>
      {/* Back Button */}
      <button
        onClick={onBack}
        className="flex rounded-md items-center text-brand-600 hover:text-brand-900 dark:text-brand-400 dark:hover:text-brand-200 transition-colors mb-6"
      >
        <div className="i-mdi-arrow-left text-2xl mr-1" />
        <span>{t('common.back')}</span>
      </button>

      <div className="flex flex-col gap-6">
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-4xl font-bold text-brand-900 dark:text-white flex items-center gap-3">
              {category.name}
              {category.is_system && <span className="text-sm bg-neutral-100 text-neutral-800 px-2 py-1 rounded-md dark:bg-neutral-900 dark:text-neutral-300 align-middle">{t('category.labels.system')}</span>}
            </h1>
            <p className="text-brand-500 dark:text-brand-400 mt-2">
              {games.length} {t('category.labels.games')}
            </p>
          </div>
        </div>

        <FilterBar
          searchQuery={searchQuery}
          onSearchChange={setSearchQuery}
          searchPlaceholder={t('category.messages.searchPlaceholder')}
          sortBy={sortBy}
          onSortByChange={val => setSortBy(val as "name" | "created_at" | "release_at" | "company")}
          sortOptions={sortOptions}
          sortOrder={sortOrder}
          onSortOrderChange={setSortOrder}
          statusFilter={statusFilter}
          onStatusFilterChange={setStatusFilter}
          statusOptions={statusOptions}
          filterExpanded={filterExpanded}
          games={filteredGames}
          tagsFilter={tagsFilter}
          onTagsFilterChange={setTags}
          tagsLoaded={tagsLoaded}
          onSourceFilterChange={setSourceFilter}
          releaseStartDate={releaseStartDate}
          onReleaseStartDateChange={setReleaseStartDate}
          releaseEndDate={releaseEndDate}
          onReleaseEndDateChange={setReleaseEndDate}
          storageKey="category"
          batchMode={batchMode}
          onBatchModeChange={handleBatchModeChange}
          selectedCount={selectedGameIds.length}
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
                    disabled={selectedGameIds.length === 0}
                    title={t('library.buttons.confirmSelection')}
                    className={`glass-panel flex items-center gap-2 px-3 py-2 text-sm
                                bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700
                                rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-success-600 dark:text-success-400
                                ${selectedGameIds.length === 0 ? "opacity-50 cursor-not-allowed" : ""}`}
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
                    disabled={selectedGameIds.length === 0}
                    trigger={( 
                      <div
                        title={t('library.buttons.batchUpdateStatus')}
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
                  {/* 批量添加标签 */}
                  <button
                    type="button"
                    onClick={() => setIsBatchAddTagModalOpen(true)}
                    disabled={selectedGameIds.length === 0}
                    title={t('library.buttons.batchAddTags')}
                    className={`glass-panel flex items-center gap-2 px-3 py-2 text-sm
                                bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700
                                rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-700 dark:text-brand-300
                                ${selectedGameIds.length === 0 ? "opacity-50 cursor-not-allowed" : ""}`}
                  >
                    <div className="i-mdi-tag-plus-outline text-lg" />
                  </button>
                  {/* 批量添加到收藏 */}
                  <button
                    type="button"
                    onClick={openBatchAddModal}
                    disabled={selectedGameIds.length === 0}
                    title={t('library.buttons.batchAddToCollection')}
                    className={`glass-panel flex items-center gap-2 px-3 py-2 text-sm
                                bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700
                                rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-700 dark:text-brand-300
                                ${selectedGameIds.length === 0 ? "opacity-50 cursor-not-allowed" : ""}`}
                  >
                    <div className="i-mdi-folder-plus-outline text-lg" />
                  </button>
                  <button
                    type="button"
                    onClick={() => {setIsBatchUpdateOpen(true)}}
                    disabled={selectedGameIds.length === 0}
                    title={t('library.buttons.updateLibrary')}
                    className={`glass-panel flex items-center gap-2 px-3 py-2 text-sm
                                bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700
                                rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-700 dark:text-brand-300
                                ${selectedGameIds.length === 0 ? "opacity-50 cursor-not-allowed" : ""}`}
                  >
                    <div className="i-mdi-folder-multiple text-lg" />
                  </button>
                  {/* 批量删除 */}
                  <button
                    type="button"
                    onClick={handleBatchDelete}
                    disabled={selectedGameIds.length === 0}
                    title={t('library.buttons.batchDelete')}
                    className={`glass-panel flex items-center gap-2 px-3 py-2 text-sm
                                bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700
                                rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-error-600 dark:text-error-400
                                ${selectedGameIds.length === 0 ? "opacity-50 cursor-not-allowed" : ""}`}
                  >
                    <div className="i-mdi-delete text-lg" />
                  </button>
                  {/* 从当前分类移除 */}
                  <button
                    type="button"
                    onClick={handleBatchRemove}
                    disabled={selectedGameIds.length === 0}
                    title={t('category.labels.batchRemove')}
                    className={`glass-panel flex items-center gap-2 px-3 py-2 text-sm
                                bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700
                                rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-error-600 dark:text-error-400
                                ${selectedGameIds.length === 0 ? "opacity-50 cursor-not-allowed" : ""}`}
                  >
                    <div className="i-mdi-folder-remove-outline text-lg" />
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
                  {t('common.add')} {t('common.game')}
                  <div className="i-mdi-chevron-down ml-2 text-lg" />
                </div>
              )}
              items={[
                {
                  key: "add",
                  label: t('common.add') + " " + t('common.game'),
                  description: t('category.descriptions.addGameToCategory'),
                  icon: "i-mdi-gamepad-variant",
                  iconColor: "text-neutral-500",
                  onClick: () => setIsAddGameModalOpen(true),
                },
              ]}
            />
          )}
        />
      </div>

      <div className="mt-6">
        {games.length > 0
          ? (
              filteredGames.length > 0
                ? (
                    <div className={
                      viewMode === "list" 
                        ? "flex flex-col gap-2"
                        : viewMode === "large"
                          ? "grid grid-cols-[repeat(auto-fill,minmax(19rem,1fr))] gap-4"
                          : "grid grid-cols-[repeat(auto-fill,minmax(8.75rem,1fr))] gap-3"
                    }>
                      {filteredGames.map(game => (
                        <div key={game.id} className="relative group">
                          <GameCard
                            game={game}
                            searchQuery={searchQuery}
                            selectionMode={batchMode}
                            selected={selectedGameIds.includes(game.id)}
                            onSelectChange={(selected, event) => setGameSelection(game.id, selected, event)}
                            filteredGameIdsStr={arrayMapString(filteredGames, (game) => game.id)}
                            viewMode={viewMode}
                          />
                          {!batchMode && (
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                handleRemoveGame(game.id);
                              }}
                              className="absolute top-2 right-2 p-1 bg-error-500 text-white rounded-full opacity-0 group-hover:opacity-100 transition-opacity shadow-md hover:bg-error-600"
                              title={t('category.messages.removeFromCategory')}
                            >
                              <div className="i-mdi-close text-sm" />
                            </button>
                          )}
                        </div>
                      ))}
                    </div>
                  )
                : (
                    <div className="flex flex-col items-center justify-center h-64 text-brand-500 dark:text-brand-400">
                      <div className="i-mdi-magnify text-6xl mb-4" />
                      <p className="text-lg">{t('category.messages.noGamesFound')}</p>
                    </div>
                  )
            )
          : (
              <div className="flex flex-col items-center justify-center h-64 text-brand-500 dark:text-brand-400">
                <div className="i-mdi-gamepad-variant-outline text-6xl mb-4" />
                <p className="text-lg">{t('category.messages.noGamesInCategory')}</p>
                <button
                  onClick={openAddGameModal}
                  className="mt-4 text-neutral-600 hover:underline dark:text-neutral-400"
                >
                  {t('common.add')} {t('common.game')}
                </button>
              </div>
            )}
      </div>

      <AddGameToCategoryModal
        isOpen={isAddGameModalOpen}
        allGames={allGames}
        onClose={() => setIsAddGameModalOpen(false)}
        onAddGame={handleAddGameToCategory}
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
        onUpdateComplete={() => loadGames(categoryId)}
        games={filteredGames.filter(game => selectedGameIds.includes(game.id))}
      />

      <AddTagModal
        isOpen={isBatchAddTagModalOpen}
        onClose={() => setIsBatchAddTagModalOpen(false)}
        onConfirm={handleBatchAddTag}
        games={filteredGames.filter(game => selectedGameIds.includes(game.id))}
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
