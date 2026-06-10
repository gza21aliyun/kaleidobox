import { createRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useState, useMemo } from "react";
import toast from "react-hot-toast";
import { useTranslation } from 'react-i18next';
import { FilterBar } from "../components/bar/FilterBar";
import { CategoriesSkeleton } from "../components/skeleton/CategoriesSkeleton";
import { CategoryListCard } from "../components/card/CategoryListCard";
import { Route as rootRoute } from "./__root";
import { models, vo, enums } from "../../wailsjs/go/models";
import { GetBrands, GetGenres, GetSeries } from "../../wailsjs/go/service/TagService";
import { GetCategories } from "../../wailsjs/go/service/CategoryService";
import { GetStaffsByRole } from "../../wailsjs/go/service/StaffService";
import { getGameIdsForCategory, type CategoryType } from "../utils/categoryGames";
import { useAppStore } from "../store";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/category_list",
  component: CategoryListPage,
  shouldReload: false,
});

type CategoryItem = {
  id: string;
  name: string;
  type: CategoryType;
  game_count?: number;
  use_count?: number;
  original?: models.Tag | vo.CategoryVO | models.Staff;
  dirPath?: string;
  gameIds?: string[];
};

function CategoryListPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { games, fetchGames } = useAppStore();
  const [favorites, setFavorites] = useState<vo.CategoryVO[]>([]);
  const [brands, setBrands] = useState<models.Tag[]>([]);
  const [series, setSeries] = useState<models.Tag[]>([]);
  const [genres, setGenres] = useState<models.Tag[]>([]);
  const [charaDesigns, setCharaDesigns] = useState<models.Staff[]>([]);
  const [scenearios, setScenearios] = useState<models.Staff[]>([]);
  const [categoryMap, setCategoryMap] = useState<Map<string, string[]>>(new Map());
  const [isLoading, setIsLoading] = useState(true);
  const [showSkeleton, setShowSkeleton] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState<"name" | "use_count" | "game_count">(() => {
    const savedFilter = localStorage.getItem('categoryListSortBy') as "name" | "use_count" | "game_count" | null;
    return savedFilter || "name";
  });
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">(() => {
    const savedSortBy = localStorage.getItem('categoryListSortOrder') as "asc" | "desc" | null;
    return savedSortBy || "asc";
  });
  const [categoryFilter, setCategoryFilter] = useState<string>(() => {
    const savedSortBy = localStorage.getItem('categoryListFilter');
    return savedSortBy || "brand";
  });
  const [viewMode, setViewMode] = useState<"default" | "gallery">(() => {
    const savedViewMode = localStorage.getItem('categoryListViewMode');
    return (savedViewMode as "default" | "gallery") || "default";
  });

  useEffect(() => {
    localStorage.setItem('categoryListFilter', categoryFilter);
  }, [categoryFilter]);

  useEffect(() => {
    localStorage.setItem('categoryListSortBy', sortBy);
  }, [sortBy]);

  useEffect(() => {
    localStorage.setItem('categoryListSortOrder', sortOrder);
  }, [sortOrder]);

  useEffect(() => {
    localStorage.setItem('categoryListViewMode', viewMode);
  }, [viewMode]);

  const categoryOptions = [
    { label: t('categoryList.options.favorite'), value: "favorite" },
    { label: t('categoryList.options.brand'), value: "brand" },
    { label: t('categoryList.options.series'), value: "series" },
    { label: t('categoryList.options.genre'), value: "genre" },
    { label: t('categoryList.options.charaDesign'), value: "chara_design" },
    { label: t('categoryList.options.sceneario'), value: "sceneario" },
    { label: t('categoryList.options.parent1'), value: "parent1" },
    { label: t('categoryList.options.parent2'), value: "parent2" },
  ];

  const loadAllCategories = async () => {
    try {
      setIsLoading(true);
      const [favoritesResult, brandsResult, seriesResult, genresResult, charaDesignsResult, sceneariosResult] = await Promise.all([
        GetCategories(),
        GetBrands(),
        GetSeries(),
        GetGenres(),
        GetStaffsByRole(enums.StaffRole.CHARA_DESIGN),
        GetStaffsByRole(enums.StaffRole.SCENEARIO),
      ]);
      setFavorites(favoritesResult || []);
      setBrands(brandsResult || []);
      setSeries(seriesResult || []);
      setGenres(genresResult || []);
      setCharaDesigns(charaDesignsResult || []);
      setScenearios(sceneariosResult || []);
    }
    catch (error) {
      console.error("Failed to load categories:", error);
      toast.error(t('categoryList.toasts.loadFailed'));
    }
    finally {
      setIsLoading(false);
    }
  };

  const preloadCategoryMap = async (categories: CategoryItem[]) => {
    if (categories.length === 0) return;
    const resultMap = new Map<string, string[]>();
    for (const cat of categories) {
      try {
        const gameIds = await getGameIdsForCategory(
          { type: cat.type, id: cat.id, name: cat.name, dirPath: cat.dirPath, original: cat.original },
          games,
        );
        resultMap.set(cat.id, gameIds);
      } catch (error) {
        console.error(`Failed to preload category ${cat.name}:`, error);
      }
      await new Promise<void>(resolve => setTimeout(resolve, 0));
    }
    setCategoryMap(resultMap);
  };

  useEffect(() => {
    if (games.length === 0) {
      fetchGames();
    }
  }, []);

  const parentDirCategories = useMemo(() => {
    if (categoryFilter !== 'parent1' && categoryFilter !== 'parent2') {
      return [];
    }

    const level = categoryFilter === 'parent1' ? 1 : 2;
    const targetDirSet = new Set<string>();

    games.forEach(game => {
      if (!game.path) return;

      let normalizedPath = game.path.replace(/\\/g, '/');
      const lastSlash = normalizedPath.lastIndexOf('/');
      if (lastSlash !== -1) {
        normalizedPath = normalizedPath.substring(0, lastSlash);
      }

      let targetDir = normalizedPath;
      for (let i = 0; i < level; i++) {
        const slashIndex = targetDir.lastIndexOf('/');
        if (slashIndex !== -1) {
          targetDir = targetDir.substring(0, slashIndex);
        } else {
          break;
        }
      }

      if (targetDir === '') {
        const firstSlash = normalizedPath.indexOf('/');
        if (firstSlash !== -1) {
          targetDir = normalizedPath.substring(0, firstSlash);
        } else {
          targetDir = normalizedPath;
        }
      }

      targetDirSet.add(targetDir);
    });

    const result: CategoryItem[] = [];
    targetDirSet.forEach(dirPath => {
      result.push({
        id: `dir-${dirPath}`,
        name: dirPath,
        type: categoryFilter,
        dirPath: dirPath,
      });
    });

    return result;
  }, [games, categoryFilter]);

  const mergedCategories: CategoryItem[] = [
    ...favorites.map(fav => ({
      id: fav.id,
      name: fav.name,
      type: 'favorite' as const,
      game_count: fav.game_count,
      original: fav,
    })),
    ...brands.map(brand => ({
      id: `brand-${brand.name}`,
      name: brand.name,
      type: 'brand' as const,
      use_count: brand.use_count,
      original: brand,
    })),
    ...series.map(s => ({
      id: `series-${s.name}`,
      name: s.name,
      type: 'series' as const,
      use_count: s.use_count,
      original: s,
    })),
    ...genres.map(g => ({
      id: `genre-${g.name}`,
      name: g.name,
      type: 'genre' as const,
      use_count: g.use_count,
      original: g,
    })),
    ...charaDesigns.map(c => ({
      id: `chara-${c.id}`,
      name: c.name,
      type: 'chara_design' as const,
      original: c,
    })),
    ...scenearios.map(s => ({
      id: `sceneario-${s.id}`,
      name: s.name,
      type: 'sceneario' as const,
      original: s,
    })),
  ];

  const categoriesToFilter = categoryFilter === 'parent1' || categoryFilter === 'parent2'
    ? parentDirCategories
    : mergedCategories;

  const filteredCategories = categoriesToFilter
    .filter((cat) => {
      if (categoryFilter === 'parent1' || categoryFilter === 'parent2') {
        if (!searchQuery) return true;
        return (cat.dirPath || cat.name).toLowerCase().includes(searchQuery.toLowerCase());
      }
      if (categoryFilter && cat.type !== categoryFilter) {
        return false;
      }
      if (!searchQuery)
        return true;
      return cat.name.toLowerCase().includes(searchQuery.toLowerCase());
    })
    .sort((a, b) => {
      let comparison = 0;
      switch (sortBy) {
        case "name":
          comparison = a.name.localeCompare(b.name);
          break;
        case "use_count":
          const countA = a.game_count ?? a.use_count ?? 0;
          const countB = b.game_count ?? b.use_count ?? 0;
          comparison = countB - countA;
          break;
        case "game_count":
          const gcA = categoryMap.get(a.id)?.length ?? 0;
          const gcB = categoryMap.get(b.id)?.length ?? 0;
          comparison = gcB - gcA;
          break;
      }
      return sortOrder === "asc" ? comparison : -comparison;
    });

  const updateCategoryMap = (id: string, gameIds: string[]) => {
    setCategoryMap(prev => {
      const newMap = new Map(prev);
      newMap.set(id, gameIds);
      return newMap;
    });
  };

  useEffect(() => {
    loadAllCategories();
  }, []);

  useEffect(() => {
    let isMounted = true;
    const categoriesToLoad = categoryFilter === 'parent1' || categoryFilter === 'parent2'
      ? parentDirCategories
      : mergedCategories.filter(c => c.type === categoryFilter);
    if (categoriesToLoad.length > 0 && (categoryMap.size === 0 || !categoryMap.has(categoriesToLoad[0]?.id))) {
      preloadCategoryMap(categoriesToLoad);
    }
    return () => {
      isMounted = false;
    };
  }, [categoryFilter, favorites, brands, series, genres, charaDesigns, scenearios, games]);

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

  if (isLoading && mergedCategories.length === 0) {
    if (!showSkeleton) {
      return null;
    }
    return <CategoriesSkeleton />;
  }

  return (
    <div className={`w-full p-8 transition-opacity duration-300 ${isLoading ? "opacity-50 pointer-events-none" : "opacity-100"}`}>
      <div className="flex items-center justify-between">
        <h1 className="text-4xl font-bold text-brand-900 dark:text-white">{t('categoryList.title', { count: filteredCategories.length })}</h1>

        <div className="flex items-center gap-2 bg-neutral-100 dark:bg-brand-700 rounded-lg p-1">
          <button
            onClick={() => setViewMode("default")}
            className={`flex items-center gap-1 px-3 py-1.5 rounded-md text-sm font-medium transition-all ${
              viewMode === "default"
                ? "bg-white dark:bg-brand-600 text-brand-900 dark:text-white shadow-sm"
                : "text-brand-600 dark:text-brand-400 hover:text-brand-800 dark:hover:text-brand-200"
            }`}
            title={t('categoryList.viewMode.list')}
          >
            <div className="i-mdi-view-agenda text-lg" />
            <span>{t('categoryList.viewMode.list')}</span>
          </button>
          <button
            onClick={() => setViewMode("gallery")}
            className={`flex items-center gap-1 px-3 py-1.5 rounded-md text-sm font-medium transition-all ${
              viewMode === "gallery"
                ? "bg-white dark:bg-brand-600 text-brand-900 dark:text-white shadow-sm"
                : "text-brand-600 dark:text-brand-400 hover:text-brand-800 dark:hover:text-brand-200"
            }`}
            title={t('categoryList.viewMode.gallery')}
          >
            <div className="i-mdi-view-grid text-lg" />
            <span>{t('categoryList.viewMode.gallery')}</span>
          </button>
        </div>
      </div>

      <FilterBar
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        searchPlaceholder={t('categoryList.searchPlaceholder')}
        sortBy={sortBy}
        onSortByChange={val => setSortBy(val as "name" | "use_count" | "game_count")}
        sortOptions={[
          { label: t('categoryList.sortOptions.name'), value: "name" },
          { label: t('categoryList.sortOptions.useCount'), value: "use_count" },
          { label: t('categoryList.sortOptions.gameCount'), value: "game_count" },
        ]}
        sortOrder={sortOrder}
        onSortOrderChange={setSortOrder}
        categoryFilter={categoryFilter}
        onCategoryFilterChange={setCategoryFilter}
        categoryOptions={categoryOptions}
      />

      <div className={
        viewMode === "gallery"
          ? "grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-6 mt-6"
          : "grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-4 mt-6"
      }>
        {filteredCategories.map(cat => (
          <CategoryListCard
            key={cat.id}
            id={cat.id}
            name={cat.name}
            type={cat.type}
            game_count={cat.game_count}
            use_count={cat.use_count}
            viewMode={viewMode}
            original={cat.original}
            dirPath={cat.dirPath}
            gameIds={cat.gameIds}
            categoryMap={categoryMap}
            onUpdateCategoryMap={updateCategoryMap}
          />
        ))}
      </div>

      {filteredCategories.length === 0 && !isLoading && (
        <div className="flex flex-col items-center justify-center h-64 text-brand-500 dark:text-brand-400 mt-8">
          <div className="i-mdi-magnify text-6xl mb-4" />
          <p className="text-lg">{t('categoryList.noCategoriesFound')}</p>
        </div>
      )}
    </div>
  );
}
