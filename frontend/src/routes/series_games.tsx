import { useTranslation } from 'react-i18next';
import { models, enums } from "../../wailsjs/go/models";
import { createRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { toast } from "react-hot-toast";
import { GetGamesByTag } from "../../wailsjs/go/service/GameService";
import { FilterBar } from "../components/bar/FilterBar";
import { GameCard } from "../components/card/GameCard";
import { sortOptions, statusOptions } from "../consts/options";
import { Route as rootRoute } from "./__root";
import { arrayMapString, arrayToMap } from "../components/utils/Utility";
import { ListTags } from "../../wailsjs/go/service/TagService";
import { formatLocalDate } from '../utils/time';

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/series/$seriesName",
  component: SeriesGamesPage,
});

function SeriesGamesPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { seriesName } = Route.useParams();
  const [games, setGames] = useState<models.Game[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState<"name" | "created_at" | "release_at" | "company"> ("created_at");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc"> ("desc");
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [batchMode, setBatchMode] = useState(false);
  const [selectedGameIds, setSelectedGameIds] = useState<string[]>([]);
  const [filterExpanded, setFilterExpanded] = useState(false);
  const [tagsFilter, setTags] = useState<string[]>(() => {
    const savedTagsFilter = localStorage.getItem('seriesTagsFilter');
    return savedTagsFilter ? JSON.parse(savedTagsFilter) : [];
  });
  const [tagsLoaded, setTagsLoaded] = useState<Map<string, models.Tag[]>>(new Map());
  const [releaseStartDate, setReleaseStartDate] = useState<string>("");
  const [releaseEndDate, setReleaseEndDate] = useState<string>("");
  const [viewMode, setViewMode] = useState<"list" | "small" | "large">(() => {
    const savedViewMode = localStorage.getItem('seriesViewMode');
    return (savedViewMode as "list" | "small" | "large") || "small";
  });

  // 保存 viewMode 到 localStorage
  useEffect(() => {
    localStorage.setItem('seriesViewMode', viewMode);
  }, [viewMode]);

  // 保存 tagsFilter 到 localStorage
  useEffect(() => {
    localStorage.setItem('seriesTagsFilter', JSON.stringify(tagsFilter));
  }, [tagsFilter]);

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

  const loadGames = async (series: string) => {
    try {
      const result = await GetGamesByTag(series);
      setGames(result || []);
    }
    catch (error) {
      console.error("Failed to load games for series:", error);
      toast.error("加载系列游戏失败");
    }
  };

  const onBack = () => {
    navigate({ to: "/series" });
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
      // 标签过滤
      if (tagsFilter && tagsFilter.length > 0) {
        const tags = game.tags ? game.tags.split(",") : [];
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
          break;
        case "company":
          comparison = String(a.company || "").localeCompare(String(b.company || ""));
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

  useEffect(() => {
    if (seriesName) {
      const init = async () => {
        setLoading(true);
        setBatchMode(false);
        setSelectedGameIds([]);
        await loadGames(seriesName);
        setLoading(false);
      };
      init();
    }
  }, [seriesName]);

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
              {seriesName}
            </h1>
            <p className="text-brand-500 dark:text-brand-400 mt-2">
              {games.length} {t('category.labels.games')}
            </p>
          </div>
        </div>

        <FilterBar
          searchQuery={searchQuery}
          onSearchChange={setSearchQuery}
          searchPlaceholder="搜索游戏"
          sortBy={sortBy}
          onSortByChange={val => setSortBy(val as "name" | "created_at" | "release_at" | "company")}
          sortOptions={sortOptions}
          sortOrder={sortOrder}
          onSortOrderChange={setSortOrder}
          statusFilter={statusFilter}
          onStatusFilterChange={setStatusFilter}
          statusOptions={statusOptions}
          filterExpanded={filterExpanded}
          tagsFilter={tagsFilter}
          onTagsFilterChange={setTags}
          tagsLoaded={tagsLoaded}
          releaseStartDate={releaseStartDate}
          onReleaseStartDateChange={setReleaseStartDate}
          releaseEndDate={releaseEndDate}
          onReleaseEndDateChange={setReleaseEndDate}
          storageKey="series"
          batchMode={batchMode}
          onBatchModeChange={handleBatchModeChange}
          selectedCount={selectedGameIds.length}
          onSelectAll={handleSelectAll}
          onClearSelection={handleClearSelection}
          viewMode={viewMode}
          onViewModeChange={setViewMode}
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
                            onSelectChange={selected => setGameSelection(game.id, selected)}
                            filteredGameIdsStr={arrayMapString(filteredGames, (game) => game.id)}
                            viewMode={viewMode}
                          />
                        </div>
                      ))}
                    </div>
                  )
                : (
                    <div className="flex flex-col items-center justify-center h-64 text-brand-500 dark:text-brand-400">
                      <div className="i-mdi-magnify text-6xl mb-4" />
                      <p className="text-lg">未找到游戏</p>
                    </div>
                  )
            )
          : (
              <div className="flex flex-col items-center justify-center h-64 text-brand-500 dark:text-brand-400">
                <div className="i-mdi-gamepad-variant-outline text-6xl mb-4" />
                <p className="text-lg">该系列下暂无游戏</p>
              </div>
            )}
      </div>
    </div>
  );
}