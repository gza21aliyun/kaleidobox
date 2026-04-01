import { createRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import toast from "react-hot-toast";
import { useTranslation } from 'react-i18next';
import { GetSeries } from "../../wailsjs/go/service/TagService";
import { FilterBar } from "../components/bar/FilterBar";
import { CategoriesSkeleton } from "../components/skeleton/CategoriesSkeleton";
import { SeriesCard } from "../components/card/SeriesCard";
import { Route as rootRoute } from "./__root";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/series",
  component: SeriesListPage,
});

interface SeriesItem {
  name: string;
}

function SeriesListPage() {
  const { t } = useTranslation();
  const [series, setSeries] = useState<SeriesItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showSkeleton, setShowSkeleton] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState<"name"> ("name");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("asc");

  const loadSeries = async () => {
    try {
      const result = await GetSeries();
      setSeries(result ? result.map(name => ({ name })) : []);
    }
    catch (error) {
      console.error("Failed to load series:", error);
      toast.error("加载系列失败");
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
      }
      return sortOrder === "asc" ? comparison : -comparison;
    });

  useEffect(() => {
    loadSeries();
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

  if (isLoading && series.length === 0) {
    if (!showSkeleton) {
      return null;
    }
    return <CategoriesSkeleton />;
  }

  return (
    <div className={`h-full w-full overflow-y-auto p-8 transition-opacity duration-300 ${isLoading ? "opacity-50 pointer-events-none" : "opacity-100"}`}>
      <div className="flex items-center justify-between">
        <h1 className="text-4xl font-bold text-brand-900 dark:text-white">系列</h1>
      </div>

      <FilterBar
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        searchPlaceholder="搜索系列"
        sortBy={sortBy}
        onSortByChange={val => setSortBy(val as "name")}
        sortOptions={[
          { label: "名称", value: "name" },
        ]}
        sortOrder={sortOrder}
        onSortOrderChange={setSortOrder}
      />

      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
        {filteredSeries.map(seriesItem => (
          <SeriesCard key={seriesItem.name} series={seriesItem} />
        ))}
      </div>
    </div>
  );
}