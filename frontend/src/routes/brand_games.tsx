import { useTranslation } from 'react-i18next';
import { models, enums } from "../../wailsjs/go/models";
import { createRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useMemo, useState } from "react";
import { toast } from "react-hot-toast";
import { GetGamesByTag, GetGamesByBrand } from "../../wailsjs/go/service/GameService";
import { FilterBar } from "../components/bar/FilterBar";
import { GameCard } from "../components/card/GameCard";
import { sortOptions, statusOptions } from "../consts/options";
import { Route as rootRoute } from "./__root";
import { arrayMapString, arrayToMap } from "../components/utils/Utility";
import { ListTags } from "../../wailsjs/go/service/TagService";
import { formatLocalDate } from '../utils/time';
import { FetchEmptyGalleryGames } from '../../wailsjs/go/service/ImageService';
import { useAppStore } from "../store";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/brand/$brandName",
  component: BrandGamesPage,
  shouldReload: false,
});

function BrandGamesPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { brandName } = Route.useParams();
  const decodedBrandName = decodeURIComponent(brandName);
  const { gameStats, loadStats } = useAppStore();
  const [games, setGames] = useState<models.Game[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState<"name" | "created_at" | "release_at" | "company" | "last_played" | "play_time"> ("created_at");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc"> ("desc");
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [batchMode, setBatchMode] = useState(false);
  const [selectedGameIds, setSelectedGameIds] = useState<string[]>([]);
  const [sourceFilter, setSourceFilter] = useState<string>("");
  const [lastSelectedGameId, setLastSelectedGameId] = useState<string | null>(null);
  const [filterExpanded, setFilterExpanded] = useState(false);
  const [tagsFilter, setTags] = useState<string[]>(() => {
    const savedTagsFilter = localStorage.getItem('brandTagsFilter');
    return savedTagsFilter ? JSON.parse(savedTagsFilter) : [];
  });
  const [tagsLoaded, setTagsLoaded] = useState<Map<string, models.Tag[]>>(new Map());
  const [releaseStartDate, setReleaseStartDate] = useState<string>("");
  const [releaseEndDate, setReleaseEndDate] = useState<string>("");
  const [viewMode, setViewMode] = useState<"list" | "small" | "large">(() => {
    const savedViewMode = localStorage.getItem('brandViewMode');
    return (savedViewMode as "list" | "small" | "large") || "small";
  });
  const [tagsIntersectionMode, setTagsIntersectionMode] = useState<boolean>(() => {
    const savedMode = localStorage.getItem('brandTagsIntersectionMode');
    return savedMode ? JSON.parse(savedMode) : false;
  });

  // 保存 viewMode 到 localStorage
  useEffect(() => {
    localStorage.setItem('brandViewMode', viewMode);
  }, [viewMode]);

  // 保存 tagsFilter 到 localStorage
  useEffect(() => {
    localStorage.setItem('brandTagsFilter', JSON.stringify(tagsFilter));
  }, [tagsFilter]);

  // 保存标签交集模式到本地存储
  useEffect(() => {
    localStorage.setItem('brandTagsIntersectionMode', JSON.stringify(tagsIntersectionMode));
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

  const loadGames = async (brand: string) => {
    try {
      const result = await GetGamesByBrand(brand);
      setGames(result || []);
      loadStats();
    }
    catch (error) {
      console.error("Failed to load games for brand:", error);
      toast.error("加载品牌游戏失败");
    }
  };

  const onBack = () => {
    navigate({ to: "/brands" });
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
  }, [games, searchQuery, statusFilter, tagsFilter, tagsIntersectionMode, releaseStartDate, releaseEndDate, sourceFilter, sortBy, sortOrder, includedIds, gameStats]);

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

  useEffect(() => {
    if (decodedBrandName) {
      const init = async () => {
        setLoading(true);
        setBatchMode(false);
        setSelectedGameIds([]);
        await loadGames(decodedBrandName);
        setLoading(false);
      };
      init();
    }
  }, [decodedBrandName]);

  // 多次尝试恢复滚动位置，确保图片加载后也能正确定位
  useEffect(() => {
    if (!loading) {
      const mainElement = document.querySelector('main');
      if (mainElement) {
        const savedPosition = sessionStorage.getItem('brandGamesScrollPosition');
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
  }, [loading]);

  // 保存滚动位置到 sessionStorage
  useEffect(() => {
    const mainElement = document.querySelector('main');
    const handleScroll = () => {
      if (mainElement) {
        sessionStorage.setItem('brandGamesScrollPosition', mainElement.scrollTop.toString());
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
              {decodedBrandName}
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
          onSortByChange={val => setSortBy(val as "name" | "created_at" | "release_at" | "company" | "last_played" | "play_time")}
          sortOptions={sortOptions}
          sortOrder={sortOrder}
          onSortOrderChange={setSortOrder}
          onSourceFilterChange={setSourceFilter}
          statusFilter={statusFilter}
          games={filteredGames}
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
          storageKey="brand"
          batchMode={batchMode}
          onBatchModeChange={handleBatchModeChange}
          selectedCount={selectedGameIds.length}
          onSelectAll={handleSelectAll}
          onClearSelection={handleClearSelection}
          viewMode={viewMode}
          onViewModeChange={setViewMode}
          tagsIntersectionMode={tagsIntersectionMode}
          onTagsIntersectionModeChange={setTagsIntersectionMode}
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
                <p className="text-lg">该品牌下暂无游戏</p>
              </div>
            )}
      </div>
    </div>
  );
}