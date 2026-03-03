import { useTranslation } from 'react-i18next';
import type { models, vo } from "../../wailsjs/go/models";
import { createRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { toast } from "react-hot-toast";
import {
  AddGameToCategory,
  GetCategoryByID,
  GetGamesByCategory,
  RemoveGameFromCategory,
  RemoveGamesFromCategory,
} from "../../wailsjs/go/service/CategoryService";
import { GetGames } from "../../wailsjs/go/service/GameService";
import { FilterBar } from "../components/bar/FilterBar";
import { GameCard } from "../components/card/GameCard";
import { AddGameToCategoryModal } from "../components/modal/AddGameToCategoryModal";
import { CategorySkeleton } from "../components/skeleton/CategorySkeleton";
import { sortOptions, statusOptions } from "../consts/options";
import { Route as rootRoute } from "./__root";

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
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState<"name" | "created_at">("created_at");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [batchMode, setBatchMode] = useState(false);
  const [selectedGameIds, setSelectedGameIds] = useState<string[]>([]);

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
          onSortByChange={val => setSortBy(val as "name" | "created_at")}
          sortOptions={sortOptions}
          sortOrder={sortOrder}
          onSortOrderChange={setSortOrder}
          statusFilter={statusFilter}
          onStatusFilterChange={setStatusFilter}
          statusOptions={statusOptions}
          storageKey="category"
          batchMode={batchMode}
          onBatchModeChange={handleBatchModeChange}
          selectedCount={selectedGameIds.length}
          onSelectAll={handleSelectAll}
          onClearSelection={handleClearSelection}
          batchActions={(
            <button
              type="button"
              onClick={handleBatchRemove}
              disabled={selectedGameIds.length === 0}
              className={`glass-panel flex items-center gap-2 px-3 py-2 text-sm
                          bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700
                          rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-error-600 dark:text-error-400
                          ${selectedGameIds.length === 0 ? "opacity-50 cursor-not-allowed" : ""}`}
            >
              <div className="i-mdi-delete text-lg" />
              {t('category.labels.batchRemove')}
            </button>
          )}
          actionButton={(
            <button
              onClick={openAddGameModal}
              className="glass-btn-neutral flex items-center rounded-lg bg-neutral-600 px-4 py-2 text-sm font-medium text-white hover:bg-neutral-700 focus:outline-none focus:ring-4 focus:ring-neutral-300 dark:bg-neutral-600 dark:hover:bg-neutral-700 dark:focus:ring-neutral-800"
            >
              <div className="i-mdi-plus mr-2 text-lg" />
              {t('common.add')} {t('common.game')}
            </button>
          )}
        />
      </div>

      <div className="mt-6">
        {games.length > 0
          ? (
              filteredGames.length > 0
                ? (
                    <div className="grid grid-cols-[repeat(auto-fill,minmax(8.75rem,1fr))] gap-3">
                      {filteredGames.map(game => (
                        <div key={game.id} className="relative group">
                          <GameCard
                            game={game}
                            searchQuery={searchQuery}
                            selectionMode={batchMode}
                            selected={selectedGameIds.includes(game.id)}
                            onSelectChange={selected => setGameSelection(game.id, selected)}
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
    </div>
  );
}
