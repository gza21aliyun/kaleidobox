import { createRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import toast from "react-hot-toast";
import { useTranslation } from 'react-i18next';
import { GetSeries } from "../../wailsjs/go/service/TagService";
import { FilterBar } from "../components/bar/FilterBar";
import { CategoriesSkeleton } from "../components/skeleton/CategoriesSkeleton";
import { SeriesCard } from "../components/card/SeriesCard";
import { Route as rootRoute } from "./__root";
import { models } from "../../wailsjs/go/models";
import { sortTags } from "../components/utils/Utility";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/series",
  component: SeriesListPage,
  shouldReload: false,
});


function SeriesListPage() {
  const { t } = useTranslation();
  const [series, setSeries] = useState<models.Tag[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showSkeleton, setShowSkeleton] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState<"name" | "use_count">("name");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("asc");
  const [viewMode, setViewMode] = useState<"default" | "gallery">(() => {
    const savedViewMode = localStorage.getItem('seriesViewMode');
    return (savedViewMode as "default" | "gallery") || "default";
  });

  useEffect(() => {
    localStorage.setItem('seriesViewMode', viewMode);
  }, [viewMode]);

  const loadSeries = async () => {
    try {
      const result = await GetSeries();
      setSeries(result ?? []);
    }
    catch (error) {
      console.error("Failed to load series:", error);
      toast.error(t('seriesList.toasts.loadSeriesFailed'));
    }
    finally {
      setIsLoading(false);
    }
  };

  const filteredSeries = series
    .filter((seriesItem) => {
      if (!searchQuery)
        return true;
      return seriesItem.name.toLowerCase().includes(searchQuery.toLowerCase());
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
    loadSeries();
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
        const savedPosition = sessionStorage.getItem('seriesScrollPosition');
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
        sessionStorage.setItem('seriesScrollPosition', mainElement.scrollTop.toString());
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

  if (isLoading && series.length === 0) {
    if (!showSkeleton) {
      return null;
    }
    return <CategoriesSkeleton />;
  }

  return (
    <div className={`w-full p-8 transition-opacity duration-300 ${isLoading ? "opacity-50 pointer-events-none" : "opacity-100"}`}>
      <div className="flex items-center justify-between">
        <h1 className="text-4xl font-bold text-brand-900 dark:text-white">{t('seriesList.title')}</h1>

        <div className="flex items-center gap-2 bg-neutral-100 dark:bg-brand-700 rounded-lg p-1">
          <button
            onClick={() => setViewMode("default")}
            className={`flex items-center gap-1 px-3 py-1.5 rounded-md text-sm font-medium transition-all ${
              viewMode === "default"
                ? "bg-white dark:bg-brand-600 text-brand-900 dark:text-white shadow-sm"
                : "text-brand-600 dark:text-brand-400 hover:text-brand-800 dark:hover:text-brand-200"
            }`}
            title={t('seriesList.viewMode.default')}
          >
            <div className="i-mdi-view-agenda text-lg" />
            <span>{t('seriesList.viewMode.default')}</span>
          </button>
          <button
            onClick={() => setViewMode("gallery")}
            className={`flex items-center gap-1 px-3 py-1.5 rounded-md text-sm font-medium transition-all ${
              viewMode === "gallery"
                ? "bg-white dark:bg-brand-600 text-brand-900 dark:text-white shadow-sm"
                : "text-brand-600 dark:text-brand-400 hover:text-brand-800 dark:hover:text-brand-200"
            }`}
            title={t('seriesList.viewMode.gallery')}
          >
            <div className="i-mdi-view-grid text-lg" />
            <span>{t('seriesList.viewMode.gallery')}</span>
          </button>
        </div>
      </div>

      <FilterBar
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        searchPlaceholder={t('seriesList.searchPlaceholder')}
        sortBy={sortBy}
        onSortByChange={val => setSortBy(val as "name" | "use_count")}
        sortOptions={[
          { label: t('seriesList.sortOptions.name'), value: "name" },
          { label: t('seriesList.sortOptions.use_count'), value: "use_count" },
        ]}
        sortOrder={sortOrder}
        onSortOrderChange={setSortOrder}
      />

      <div className={
        viewMode === "gallery"
          ? "grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-6"
          : "grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-4"
      }>
        {filteredSeries.map(seriesItem => (
          <SeriesCard
            key={seriesItem.name}
            series={seriesItem}
            viewMode={viewMode}
          />
        ))}
      </div>
    </div>
  );
}