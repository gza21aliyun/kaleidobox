import { createRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import toast from "react-hot-toast";
import { useTranslation } from 'react-i18next';
import { GetBrands } from "../../wailsjs/go/service/TagService";
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
  shouldReload: false,
});


function BrandsPage() {
  const { t } = useTranslation();
  const [brands, setBrands] = useState<models.Tag[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showSkeleton, setShowSkeleton] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState<"name" | "use_count">("name");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("asc");
  const [viewMode, setViewMode] = useState<"default" | "gallery">(() => {
    const savedViewMode = localStorage.getItem('brandsViewMode');
    return (savedViewMode as "default" | "gallery") || "default";
  });

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
      toast.error(t('brandList.toasts.loadBrandsFailed'));
    }
    finally {
      setIsLoading(false);
    }
  };

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
        case "use_count":
          comparison = (b.use_count || 0) - (a.use_count || 0);
          break;
      }
      return sortOrder === "asc" ? comparison : -comparison;
    });



  useEffect(() => {
    loadBrands();
  }, []);

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

  // 多次尝试恢复滚动位置，确保图片加载后也能正确定位
  useEffect(() => {
    if (!isLoading) {
      const mainElement = document.querySelector('main');
      if (mainElement) {
        const savedPosition = sessionStorage.getItem('brandsScrollPosition');
        if (savedPosition) {
          const attemptRestore = (attempts: number) => {
            if (attempts <= 0) return;
            setTimeout(() => {
              mainElement.scrollTop = parseInt(savedPosition);
              attemptRestore(attempts - 1);
            }, 100);
          };
          attemptRestore(10);
        }
      }
    }
  }, [isLoading]);

  // 保存滚动位置到 sessionStorage
  useEffect(() => {
    const mainElement = document.querySelector('main');
    const handleScroll = () => {
      if (mainElement) {
        sessionStorage.setItem('brandsScrollPosition', mainElement.scrollTop.toString());
      }
    };

    if (mainElement) {
      mainElement.addEventListener('scroll', handleScroll);
    }

    return () => {
      if (mainElement) {
        mainElement.removeEventListener('scroll', handleScroll);
      }
    };
  }, []);

  if (isLoading && brands.length === 0) {
    if (!showSkeleton) {
      return null;
    }
    return <CategoriesSkeleton />;
  }

  return (
    <div className={`w-full p-8 transition-opacity duration-300 ${isLoading ? "opacity-50 pointer-events-none" : "opacity-100"}`}>
      <div className="flex items-center justify-between">
        <h1 className="text-4xl font-bold text-brand-900 dark:text-white">{t('brandList.title')}</h1>

        <div className="flex items-center gap-2 bg-neutral-100 dark:bg-brand-700 rounded-lg p-1">
          <button
            onClick={() => setViewMode("default")}
            className={`flex items-center gap-1 px-3 py-1.5 rounded-md text-sm font-medium transition-all ${
              viewMode === "default"
                ? "bg-white dark:bg-brand-600 text-brand-900 dark:text-white shadow-sm"
                : "text-brand-600 dark:text-brand-400 hover:text-brand-800 dark:hover:text-brand-200"
            }`}
            title={t('brandList.viewMode.default')}
          >
            <div className="i-mdi-view-agenda text-lg" />
            <span>{t('brandList.viewMode.default')}</span>
          </button>
          <button
            onClick={() => setViewMode("gallery")}
            className={`flex items-center gap-1 px-3 py-1.5 rounded-md text-sm font-medium transition-all ${
              viewMode === "gallery"
                ? "bg-white dark:bg-brand-600 text-brand-900 dark:text-white shadow-sm"
                : "text-brand-600 dark:text-brand-400 hover:text-brand-800 dark:hover:text-brand-200"
            }`}
            title={t('brandList.viewMode.gallery')}
          >
            <div className="i-mdi-view-grid text-lg" />
            <span>{t('brandList.viewMode.gallery')}</span>
          </button>
        </div>
      </div>

      <FilterBar
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        searchPlaceholder={t('brandList.searchPlaceholder')}
        sortBy={sortBy}
        onSortByChange={val => setSortBy(val as "name" | "use_count")}
        sortOptions={[
          { label: t('brandList.sortOptions.name'), value: "name" },
          { label: t('brandList.sortOptions.use_count'), value: "use_count" },
        ]}
        sortOrder={sortOrder}
        onSortOrderChange={setSortOrder}
      />

      <div className={
        viewMode === "gallery"
          ? "grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-6"
          : "grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-4"
      }>
        {filteredBrands.map(brand => (
          <BrandCard
            key={brand.name}
            brand={brand}
            viewMode={viewMode}
          />
        ))}
      </div>
    </div>
  );
}