import { createRoute } from "@tanstack/react-router";
import { useEffect, useState, useCallback, useRef } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-hot-toast";
import { createPortal } from "react-dom";
import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";
import { FetchMonthlyReleases, ClearGetchuTempImages, FetchGetchuImages } from "../../wailsjs/go/service/MonthlyReleaseService";
import { SearchBT, DownloadToQBittorrent } from "../../wailsjs/go/service/BTDownloadService";
import { utils } from "../../wailsjs/go/models";
import { Route as rootRoute } from "./__root";
import { BetterSelect } from "../components/ui/BetterSelect";
import { useAppStore } from "../store";
import { GetchuGameInfoPanel } from "../components/panel/GetchuGameInfoPanel";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/monthly_releases",
  component: MonthlyReleasesPage,
  shouldReload: false,
});

function MonthlyReleasesPage() {
  const { t } = useTranslation();

  // 初始化年月为当前年月
  const now = new Date();
  const [year, setYear] = useState<number>(now.getFullYear());
  const [month, setMonth] = useState<number>(now.getMonth() + 1);

  const [age, setAge] = useState<string>("all");
  const [isLoading, setIsLoading] = useState(false);
  const [result, setResult] = useState<utils.MonthlyReleaseResult | null>(null);

  // 搜索弹窗状态
  const [searchModalOpen, setSearchModalOpen] = useState(false);
  const [searchGame, setSearchGame] = useState<utils.MonthlyReleaseGame | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<any[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [isDownloading, setIsDownloading] = useState(false);

  // 详情弹窗状态
  const [detailModalOpen, setDetailModalOpen] = useState(false);
  const [detailGame, setDetailGame] = useState<utils.MonthlyReleaseGame | null>(null);

  const { config } = useAppStore();

  const loadData = useCallback(async () => {
    setIsLoading(true);
    try {
      // 清空临时文件夹
      await ClearGetchuTempImages();
      const data = await FetchMonthlyReleases(year, month, age);
      setResult(data);
    } catch (error) {
      console.error("Failed to fetch monthly releases:", error);
      toast.error(t("monthlyReleases.toasts.fetchFailed"));
    } finally {
      setIsLoading(false);
    }
  }, [year, month, age]);

  useEffect(() => {
    loadData();
  }, [year, month, age]);

  // 生成可选年份列表 (当前年-3 到 当前年+2)
  const yearOptions = Array.from({ length: 16 }, (_, i) => ({ value: (now.getFullYear() - 15 + i).toString(), label: (now.getFullYear() - 15 + i).toString() }));
  const monthOptions = Array.from({ length: 12 }, (_, i) => i + 1);

  // 上一个月
  const goToPrevMonth = () => {
    if (month === 1) {
      setYear(year - 1);
      setMonth(12);
    } else {
      setMonth(month - 1);
    }
  };

  // 下一个月
  const goToNextMonth = () => {
    if (month === 12) {
      setYear(year + 1);
      setMonth(1);
    } else {
      setMonth(month + 1);
    }
  };

  const totalGames = result?.groups?.reduce((sum, g) => sum + g.games.length, 0) ?? 0;

  // 打开搜索弹窗
  const openSearchModal = (game: utils.MonthlyReleaseGame) => {
    setSearchGame(game);
    setSearchQuery(game.name || "");
    setSearchResults([]);
    setSearchModalOpen(true);
    // 只有当名字不为空时才自动搜索
    if (game.name && game.name.trim()) {
      performSearch(game.name);
    }
  };

  // 关闭搜索弹窗
  const closeSearchModal = () => {
    setSearchModalOpen(false);
    setSearchGame(null);
    setSearchQuery("");
    setSearchResults([]);
  };

  // 执行搜索
  const performSearch = async (searchKey: string) => {
    if (!searchKey.trim() || !config?.rss_url) {
      toast.error(t("btDownload.searchFailed") || "Search failed");
      return;
    }

    setIsSearching(true);
    try {
      const results = await SearchBT(searchKey, config.rss_url);
      setSearchResults(results || []);
      if (!results || results.length === 0) {
        toast.success(t("btDownload.noResults") || "No results found");
      }
    } catch (error) {
      console.error("Search failed:", error);
      toast.error(t("btDownload.searchFailed") || "Search failed");
    } finally {
      setIsSearching(false);
    }
  };

  // 搜索标题（提取主标题）
  const searchByTitle = () => {
    if (!searchGame) return;
    // 使用游戏名的前半部分作为标题
    const name = searchGame.name || "";
    const parts = name.split(/[－\-~～　＝ ・！：─―_!「\[\]]/);
    const title = parts[0]?.trim() || name;
    setSearchQuery(title);
    performSearch(title);
  };

  // 搜索全名
  const searchByFullName = () => {
    if (!searchGame) return;
    const fullName = searchGame.name || "";
    setSearchQuery(fullName);
    performSearch(fullName);
  };

  // 下载选中的资源
  const downloadSelected = async (result: any) => {
    if (!config?.qb_server) {
      toast.error(t("btDownload.notConfigured") || "Please configure qBittorrent settings first");
      return;
    }

    if (!result.link) {
      toast.error("No download link available");
      return;
    }

    setIsDownloading(true);
    try {
      const downloadResult = await DownloadToQBittorrent(
        config.qb_server,
        config.qb_user || "",
        config.qb_password || "",
        config.qb_download_folder || "",
        result.link,
        config.qb_port || 8080
      );

      if (downloadResult.success) {
        toast.success(t("btDownload.downloadStarted") || "Download started");
        closeSearchModal();
      } else {
        toast.error(downloadResult.message || t("btDownload.downloadFailed") || "Download failed");
      }
    } catch (error) {
      console.error("Download failed:", error);
      toast.error(t("btDownload.downloadFailed") || "Download failed");
    } finally {
      setIsDownloading(false);
    }
  };

  // 浏览游戏网页
  const browseGame = (game: utils.MonthlyReleaseGame) => {
    if (game.detail_url) {
      BrowserOpenURL(game.detail_url);
    }
  };

  // 打开详情弹窗
  const openDetailModal = (game: utils.MonthlyReleaseGame) => {
    setDetailGame(game);
    setDetailModalOpen(true);
  };

  // 关闭详情弹窗
  const closeDetailModal = () => {
    setDetailModalOpen(false);
    setDetailGame(null);
  };

  return (
    <div className={`w-full p-8 transition-opacity duration-300 ${isLoading ? "opacity-50 pointer-events-none" : "opacity-100"}`}>
      {/* 标题栏 */}
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-4xl font-bold text-brand-900 dark:text-white">
          {t("monthlyReleases.title")}
          {result && (
            <span className="text-lg font-normal text-brand-500 dark:text-brand-400 ml-3">
              ({totalGames} {t("monthlyReleases.gamesCount")})
            </span>
          )}
        </h1>
        {/* 搜索按钮 */}
        <button
          onClick={() => openSearchModal({ name: "" } as utils.MonthlyReleaseGame)}
          className="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-primary-500 hover:bg-primary-600 text-white text-sm font-medium transition-colors"
        >
          <div className="i-mdi-magnify" />
          <span>{t("monthlyReleases.searchBT") || "搜索BT"}</span>
        </button>
      </div>

      {/* 年月选择器 */}
      <div className="flex items-center gap-3 mb-6 flex-wrap">
        {/* 上月按钮 */}
        <button
          onClick={goToPrevMonth}
          className="flex items-center justify-center w-9 h-9 rounded-lg bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700 hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-700 dark:text-brand-300 transition-colors"
          title={t("monthlyReleases.prevMonth")}
        >
          <div className="i-mdi-chevron-left text-xl" />
        </button>

        {/* 年份选择 */}
        <BetterSelect
          value={year.toString()}
          onChange={(e) => setYear(Number(e))}
          options={yearOptions}
          placeholder={t("monthlyReleases.year") || '请选择年份'}
          className="min-w-[140px]"
        />
        <span className="text-brand-500 dark:text-brand-400 text-sm">{t("monthlyReleases.year")}</span>

        {/* 月份选择 */}
        <BetterSelect
          value={month.toString()}
          onChange={(e) => setMonth(Number(e))}
          options={monthOptions.map((m) => ({ value: m.toString(), label: m.toString() }))}
          placeholder={t("monthlyReleases.month") || '请选择月份'}
          className="min-w-[120px]"
        />
        <span className="text-brand-500 dark:text-brand-400 text-sm">{t("monthlyReleases.month")}</span>

        {/* 下月按钮 */}
        <button
          onClick={goToNextMonth}
          className="flex items-center justify-center w-9 h-9 rounded-lg bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700 hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-700 dark:text-brand-300 transition-colors"
          title={t("monthlyReleases.nextMonth")}
        >
          <div className="i-mdi-chevron-right text-xl" />
        </button>
        <span className="text-brand-500 dark:text-brand-400 text-sm">{t("monthlyReleases.age")}</span>
        <BetterSelect
          value={age}
          onChange={(e) => setAge(e)}
          options={[
            { value: "normal", label: t("monthlyReleases.normal") },
            { value: "all", label: t("monthlyReleases.all") },
            { value: "adult", label: t("monthlyReleases.adult") },
          ]}
          className="min-w-[120px]"
        />
        {/* 刷新按钮 */}
        <button
          onClick={loadData}
          disabled={isLoading}
          className="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-primary-500 hover:bg-primary-600 text-white text-sm font-medium transition-colors disabled:opacity-50"
          title={t("monthlyReleases.refresh")}
        >
          <div className="i-mdi-refresh text-lg" />
          <span>{t("monthlyReleases.refresh")}</span>
        </button>
      </div>

      {/* 加载状态 */}
      {isLoading && !result && (
        <div className="flex items-center justify-center py-20">
          <div className="flex flex-col items-center gap-4">
            <div className="i-mdi-loading text-5xl text-primary-500 animate-spin" />
            <p className="text-brand-500 dark:text-brand-400">{t("monthlyReleases.loading")}</p>
          </div>
        </div>
      )}

      {/* 无数据 */}
      {!isLoading && result?.groups?.length === 0 && (
        <div className="flex items-center justify-center py-20">
          <div className="flex flex-col items-center gap-4">
            <div className="i-mdi-calendar-blank text-5xl text-brand-400" />
            <p className="text-brand-500 dark:text-brand-400">{t("monthlyReleases.empty")}</p>
          </div>
        </div>
      )}

      {/* 游戏列表 */}
      {result?.groups?.map((group) => (
        <div key={group.release_date} className="mb-8">
          {/* 日期标题 */}
          <div className="flex items-center gap-3 mb-4">
            <div className="i-mdi-calendar text-xl text-primary-500" />
            <h2 className="text-xl font-bold text-brand-800 dark:text-brand-200">
              {group.release_date}
            </h2>
            <span className="text-sm text-brand-500 dark:text-brand-400 bg-brand-100 dark:bg-brand-700 px-2 py-0.5 rounded-full">
              {group.games.length} {t("monthlyReleases.gamesCount")}
            </span>
          </div>

          {/* 游戏网格 */}
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 2xl:grid-cols-7 gap-4">
            {group.games.map((game) => (
              <GameItem
                key={game.getchu_id}
                game={game}
                onBrowse={browseGame}
                onSearch={openSearchModal}
                onViewDetail={openDetailModal}
              />
            ))}
          </div>
        </div>
      ))}

      {/* 搜索弹窗 */}
      {searchModalOpen && (
        <BTSearchModal
          game={searchGame}
          searchQuery={searchQuery}
          setSearchQuery={setSearchQuery}
          searchResults={searchResults}
          isSearching={isSearching}
          isDownloading={isDownloading}
          onSearchTitle={searchByTitle}
          onSearchFullName={searchByFullName}
          onSearch={performSearch}
          onDownload={downloadSelected}
          onClose={closeSearchModal}
        />
      )}

      {/* 详情弹窗 */}
      {detailModalOpen && detailGame && (
        <DetailModal
          game={detailGame}
          onClose={closeDetailModal}
        />
      )}
      
    </div>
  );
}

// 将本地文件路径转换为 /local/ URL 供前端显示
function getLocalPath(localPath: string): string {
  if (!localPath) return "";
  const ar = localPath.split("\\");
  if (ar.length < 3) return "";
  return `/local/${ar[ar.length - 3]}/${ar[ar.length - 2]}/${ar[ar.length - 1]}`;
}

// 懒加载封面图组件
interface LazyCoverProps {
  game: utils.MonthlyReleaseGame;
}

function LazyCover({ game }: LazyCoverProps) {
  const [isLoaded, setIsLoaded] = useState(false);
  const [localUrl, setLocalUrl] = useState("");
  const [isError, setIsError] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (isLoaded) return;
    
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting && !isLoaded) {
          loadImage();
        }
      },
      { threshold: 0.01 }
    );

    if (ref.current) {
      observer.observe(ref.current);
    }

    return () => observer.disconnect();
  }, [isLoaded]);

  const loadImage = async () => {
    if (isLoaded) return; // 防止重复加载
    
    try {
      const coverUrl = game.cover_url || "";
      console.log("LazyCover: loading", coverUrl);
      
      if (coverUrl) {
        // 如果 cover_url 是本地路径（Windows路径包含:\），直接转换使用
        if (coverUrl.includes(":\\")) {
          console.log("LazyCover: using local path");
          setLocalUrl(getLocalPath(coverUrl));
        } else {
          // 否则下载图片
          console.log("LazyCover: downloading from", coverUrl);
          const localPaths = await FetchGetchuImages([coverUrl]);
          console.log("LazyCover: result", localPaths);
          if (localPaths.length > 0) {
            console.log("LazyCover: downloaded to", localPaths[0]);
            setLocalUrl(getLocalPath(localPaths[0]));
          } else {
            console.log("LazyCover: download failed, using original url");
            setLocalUrl(coverUrl);
          }
        }
      } else {
        console.log("LazyCover: no cover url");
      }
      setIsLoaded(true);
    } catch (error) {
      console.error("Failed to load cover:", error);
      setIsError(true);
      setIsLoaded(true);
    }
  };

  if (isError) {
    return (
      <div ref={ref} className="absolute inset-0 w-full h-full flex items-center justify-center">
        <div className="i-mdi-image-off text-4xl text-brand-400" />
      </div>
    );
  }

  if (!isLoaded || !localUrl) {
    return (
      <div ref={ref} className="absolute inset-0 w-full h-full flex items-center justify-center">
        <div className="w-6 h-6 border-2 border-brand-300 border-t-brand-500 rounded-full animate-spin" />
      </div>
    );
  }

  return (
    <img
      src={localUrl}
      alt={game.name}
      className="absolute inset-0 w-full h-full object-cover object-center group-hover:scale-105 transition-transform duration-300"
      onError={() => setIsError(true)}
    />
  );
}

// 单个游戏卡片组件
interface GameItemProps {
  game: utils.MonthlyReleaseGame;
  onBrowse: (game: utils.MonthlyReleaseGame) => void;
  onSearch: (game: utils.MonthlyReleaseGame) => void;
  onViewDetail: (game: utils.MonthlyReleaseGame) => void;
}

function GameItem({ game, onBrowse, onSearch, onViewDetail }: GameItemProps) {
  const { t } = useTranslation();
  const [isHovered, setIsHovered] = useState(false);

  return (
    <div
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
      className="group relative cursor-pointer rounded-lg bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700 overflow-hidden hover:shadow-lg hover:border-primary-400 dark:hover:border-primary-500 transition-all duration-200 hover:-translate-y-0.5"
    >
      {/* 封面图 - 懒加载 */}
      <div className="relative aspect-[3/4] bg-brand-100 dark:bg-brand-700 overflow-hidden">
        <LazyCover game={game} />

        {/* 悬停按钮覆盖层 */}
        {isHovered && (
          <div className="absolute inset-0 bg-black/60 flex flex-col items-center justify-center gap-2">
            <button
              onClick={(e) => {
                e.stopPropagation();
                onBrowse(game);
              }}
              className="flex items-center justify-center w-10 h-10 rounded-full bg-white/90 hover:bg-white text-brand-700 transition-colors"
              title={t("monthlyReleases.browse")}
            >
              <div className="i-mdi-web text-xl" />
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation();
                onSearch(game);
              }}
              className="flex items-center justify-center w-10 h-10 rounded-full bg-primary-500 hover:bg-primary-600 text-white transition-colors"
              title={t("monthlyReleases.searchBT")}
            >
              <div className="i-mdi-magnify text-xl" />
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation();
                onViewDetail(game);
              }}
              className="flex items-center justify-center w-10 h-10 rounded-full bg-blue-500 hover:bg-blue-600 text-white transition-colors"
              title={t("monthlyReleases.viewDetail") || "查看详情"}
            >
              <div className="i-mdi-info text-xl" />
            </button>
          </div>
        )}
      </div>

      {/* 信息区 */}
      <div className="p-2.5">
        <p className="text-sm font-medium text-brand-900 dark:text-white line-clamp-2 leading-snug mb-1" title={game.name}>
          {game.name}
        </p>
        {game.company && (
          <p className="text-xs text-brand-500 dark:text-brand-400 truncate" title={game.company}>
            {game.company}
          </p>
        )}
      </div>
    </div>
  );
}

// 详情弹窗组件
interface DetailModalProps {
  game: utils.MonthlyReleaseGame;
  onClose: () => void;
}

function DetailModal({ game, onClose }: DetailModalProps) {
  const { t } = useTranslation();

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
      <div className="w-full max-w-4xl max-h-[80vh] rounded-xl bg-white dark:bg-brand-800 shadow-xl border border-brand-200 dark:border-brand-700 flex flex-col overflow-hidden">
        {/* 标题栏 */}
        <div className="flex items-center justify-between p-4 border-b border-brand-200 dark:border-brand-700">
          <h3 className="text-lg font-bold text-brand-900 dark:text-white">
            {t("monthlyReleases.gameDetail") || "游戏详情"}
          </h3>
          <button
            onClick={onClose}
            className="p-1 rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-500"
          >
            <div className="i-mdi-close text-xl" />
          </button>
        </div>

        {/* 内容区 */}
        <div className="flex-1 overflow-y-auto p-6">
          <GetchuGameInfoPanel
            getchuId={game.getchu_id || ""}
            gameName={game.name || ""}
            company={game.company || ""}
            coverURL={getLocalPath(game.cover_url) || ""}
            onClose={onClose}
          />
        </div>
      </div>
    </div>,
    document.body
  );
}

// BT搜索弹窗组件
interface BTSearchModalProps {
  game?: utils.MonthlyReleaseGame | null;
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  searchResults: any[];
  isSearching: boolean;
  isDownloading: boolean;
  onSearchTitle: () => void;
  onSearchFullName: () => void;
  onSearch: (query: string) => void;
  onDownload: (result: any) => void;
  onClose: () => void;
}

function BTSearchModal({
  game,
  searchQuery,
  setSearchQuery,
  searchResults,
  isSearching,
  isDownloading,
  onSearchTitle,
  onSearchFullName,
  onSearch,
  onDownload,
  onClose,
}: BTSearchModalProps) {
  const { t } = useTranslation();

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
      <div className="w-full max-w-3xl max-h-[80vh] rounded-xl bg-white dark:bg-brand-800 shadow-xl border border-brand-200 dark:border-brand-700 flex flex-col">
        {/* 标题栏 */}
        <div className="flex items-center justify-between p-4 border-b border-brand-200 dark:border-brand-700">
          <h3 className="text-lg font-bold text-brand-900 dark:text-white">
            {t("btDownload.searchTitle") || "搜索BT资源"}
          </h3>
          <button
            onClick={onClose}
            className="p-1 rounded-lg hover:bg-brand-100 dark:hover:bg-brand-700 text-brand-500"
          >
            <div className="i-mdi-close text-xl" />
          </button>
        </div>

        {/* 搜索区域 */}
        <div className="p-4 border-b border-brand-200 dark:border-brand-700">
          {game && game.name && (
            <p className="text-sm text-brand-600 dark:text-brand-400 mb-2">
              {t("btDownload.searchingFor") || "搜索中"}: <span className="font-medium text-brand-800 dark:text-brand-200">{game.name}</span>
            </p>
          )}

          <div className="flex gap-2">
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  onSearch(searchQuery);
                }
              }}
              placeholder={t("btDownload.searchPlaceholder") || "输入搜索关键字"}
              className="flex-1 px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-lg bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </div>

          {/* 搜索按钮组 */}
          <div className="flex gap-2 mt-3">
            {game && game.name && game.name.trim() && (
              <>
                <button
                  onClick={onSearchTitle}
                  disabled={isSearching}
                  className="px-3 py-1.5 text-sm font-medium rounded-lg bg-brand-100 hover:bg-brand-200 dark:bg-brand-700 dark:hover:bg-brand-600 text-brand-700 dark:text-brand-300 transition-colors disabled:opacity-50"
                >
                  {t("btDownload.searchByTitle") || "搜索标题"}
                </button>
                <button
                  onClick={onSearchFullName}
                  disabled={isSearching}
                  className="px-3 py-1.5 text-sm font-medium rounded-lg bg-brand-100 hover:bg-brand-200 dark:bg-brand-700 dark:hover:bg-brand-600 text-brand-700 dark:text-brand-300 transition-colors disabled:opacity-50"
                >
                  {t("btDownload.searchByFullName") || "搜索全名"}
                </button>
              </>
            )}
            <button
              onClick={() => onSearch(searchQuery)}
              disabled={isSearching}
              className="px-3 py-1.5 text-sm font-medium rounded-lg bg-primary-500 hover:bg-primary-600 text-white transition-colors disabled:opacity-50"
            >
              {isSearching ? (
                <div className="i-mdi-loading animate-spin" />
              ) : (
                t("btDownload.search") || "搜索"
              )}
            </button>
          </div>
        </div>

        {/* 搜索结果列表 */}
        <div className="flex-1 overflow-y-auto p-4">
          {searchResults.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-8 text-brand-400">
              <div className="i-mdi-magnify text-4xl mb-2" />
              <p>{t("btDownload.noResults") || "暂无搜索结果"}</p>
            </div>
          ) : (
            <div className="space-y-2">
              {searchResults.map((result, index) => (
                <div
                  key={index}
                  className="p-3 rounded-lg border border-brand-200 dark:border-brand-700 hover:border-primary-400 dark:hover:border-primary-500 cursor-pointer transition-colors"
                  onClick={() => onDownload(result)}
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <span className={`px-1.5 py-0.5 text-xs rounded ${result.link?.startsWith('magnet:') ? 'bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300' : 'bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300'}`}>
                          {result.link?.startsWith('magnet:') ? '磁链' : '种子'}
                        </span>
                        {result.trusted && (
                          <span className="px-1.5 py-0.5 text-xs rounded bg-yellow-100 dark:bg-yellow-900 text-yellow-700 dark:text-yellow-300">
                            受信任
                          </span>
                        )}
                        <span className="flex items-center gap-1 text-xs text-green-600 dark:text-green-400">
                          <div className="i-mdi-arrow-up text-xs" />
                          {result.seeders}
                        </span>
                        <span className="flex items-center gap-1 text-xs text-red-500 dark:text-red-400">
                          <div className="i-mdi-arrow-down text-xs" />
                          {result.leechers}
                        </span>
                      </div>
                      <p className={`text-sm font-medium line-clamp-2 ${result.trusted ? 'text-green-600 dark:text-green-400' : 'text-brand-900 dark:text-white'}`} title={result.title}>
                        {result.title}
                      </p>
                      <p className="text-xs text-brand-500 dark:text-brand-400 truncate mt-1" title={result.link}>
                        {result.link}
                      </p>
                      <div className="flex items-center gap-3 mt-1 text-xs text-brand-500 dark:text-brand-400">
                        {result.size && <span>{result.size}</span>}
                        {result.date && <span>{result.date}</span>}
                      </div>
                    </div>
                    <button
                      disabled={isDownloading}
                      className="px-3 py-1.5 text-sm font-medium rounded-lg bg-primary-500 hover:bg-primary-600 text-white transition-colors disabled:opacity-50"
                    >
                      {isDownloading ? (
                        <div className="i-mdi-loading animate-spin" />
                      ) : (
                        t("btDownload.download") || "下载"
                      )}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>,
    document.body
  );
}
