import { createRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import toast from "react-hot-toast";
import { useTranslation } from 'react-i18next';
import { GetBrands } from "../../wailsjs/go/service/TagService";
import { GetGamesByTag, GetGamesByBrand } from "../../wailsjs/go/service/GameService";
import { FilterBar } from "../components/bar/FilterBar";
import { CategoriesSkeleton } from "../components/skeleton/CategoriesSkeleton";
import { BrandCard } from "../components/card/BrandCard";
import { Route as rootRoute } from "./__root";
import { models } from "../../wailsjs/go/models";
import { sortTags } from "../components/utils/Utility";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/brands",
  component: BrandsPage,
});


function BrandsPage() {
  const { t } = useTranslation();
  const [brands, setBrands] = useState<models.Tag[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showSkeleton, setShowSkeleton] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState<"name"> ("name");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("asc");
  const [viewMode, setViewMode] = useState<"default" | "gallery">(() => {
    const savedViewMode = localStorage.getItem('brandsViewMode');
    return (savedViewMode as "default" | "gallery") || "default";
  });
  const [brandGames, setBrandGames] = useState<Map<string, models.Game[]>>(new Map());
  const [loadingGames, setLoadingGames] = useState<Set<string>>(new Set());

  // 保存 viewMode 到 localStorage
  useEffect(() => {
    localStorage.setItem('brandsViewMode', viewMode);
  }, [viewMode]);

  const loadBrands = async () => {
    try {
      const result = await GetBrands();
      setBrands(result || []);
    }
    catch (error) {
      console.error("Failed to load brands:", error);
      toast.error("加载品牌失败");
    }
    finally {
      setIsLoading(false);
    }
  };

  const loadBrandGames = async (brandName: string) => {
    if (loadingGames.has(brandName) || brandGames.has(brandName)) {
      return;
    }
    
    setLoadingGames(prev => new Set(prev).add(brandName));
    
    try {
      const games = await GetGamesByBrand(brandName);
      setBrandGames(prev => new Map(prev).set(brandName, games || []));
    } catch (error) {
      console.error(`Failed to load games for brand ${brandName}:`, error);
    } finally {
      setLoadingGames(prev => {
        const newSet = new Set(prev);
        newSet.delete(brandName);
        return newSet;
      });
    }
  };

  // 预加载画廊模式下的品牌游戏
  useEffect(() => {
    if (viewMode === "gallery" && brands.length > 0) {
      brands.forEach(brand => {
        loadBrandGames(brand.name);
      });
    }
  }, [viewMode, brands]);

  const filteredBrands = brands
    .filter((brand) => {
      if (!searchQuery)
        return true;
      return brand.name.toLowerCase().includes(searchQuery.toLowerCase());
    })
    .sort((a, b) => {
      let comparison = 0;
      switch (sortBy) {
        case "name":
          comparison = a.name.localeCompare(b.name);
          break;
      }
      return sortOrder === "asc" ? comparison : -comparison;
    });



  useEffect(() => {
    loadBrands();
  }, []);

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

  if (isLoading && brands.length === 0) {
    if (!showSkeleton) {
      return null;
    }
    return <CategoriesSkeleton />;
  }

  return (
    <div className={`h-full w-full overflow-y-auto p-8 transition-opacity duration-300 ${isLoading ? "opacity-50 pointer-events-none" : "opacity-100"}`}>
      <div className="flex items-center justify-between">
        <h1 className="text-4xl font-bold text-brand-900 dark:text-white">品牌</h1>
        
        {/* 视图模式切换按钮 */}
        <div className="flex items-center gap-2 bg-neutral-100 dark:bg-brand-700 rounded-lg p-1">
          <button
            onClick={() => setViewMode("default")}
            className={`flex items-center gap-1 px-3 py-1.5 rounded-md text-sm font-medium transition-all ${
              viewMode === "default"
                ? "bg-white dark:bg-brand-600 text-brand-900 dark:text-white shadow-sm"
                : "text-brand-600 dark:text-brand-400 hover:text-brand-800 dark:hover:text-brand-200"
            }`}
            title="默认视图"
          >
            <div className="i-mdi-view-agenda text-lg" />
            <span>默认</span>
          </button>
          <button
            onClick={() => setViewMode("gallery")}
            className={`flex items-center gap-1 px-3 py-1.5 rounded-md text-sm font-medium transition-all ${
              viewMode === "gallery"
                ? "bg-white dark:bg-brand-600 text-brand-900 dark:text-white shadow-sm"
                : "text-brand-600 dark:text-brand-400 hover:text-brand-800 dark:hover:text-brand-200"
            }`}
            title="画廊视图"
          >
            <div className="i-mdi-view-grid text-lg" />
            <span>画廊</span>
          </button>
        </div>
      </div>

      <FilterBar
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        searchPlaceholder="搜索品牌"
        sortBy={sortBy}
        onSortByChange={val => setSortBy(val as "name")}
        sortOptions={[
          { label: "名称", value: "name" },
        ]}
        sortOrder={sortOrder}
        onSortOrderChange={setSortOrder}
      />

      <div className={
        viewMode === "gallery"
          ? "grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-6"
          : "grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-4"
      }>
        {sortTags(filteredBrands).map(brand => (
          <BrandCard 
            key={brand.name} 
            brand={brand} 
            viewMode={viewMode}
            games={brandGames.get(brand.name) || []}
          />
        ))}
      </div>
    </div>
  );
}