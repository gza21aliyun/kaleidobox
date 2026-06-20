import { createRoute } from "@tanstack/react-router";
import { useEffect, useState, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-hot-toast";
import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";
import { FetchMonthlyReleases } from "../../wailsjs/go/service/MonthlyReleaseService";
import { utils } from "../../wailsjs/go/models";
import { Route as rootRoute } from "./__root";
import { BetterSelect } from "../components/ui/BetterSelect";

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
  
  const [age, setAge] = useState<string>("normal");
  const [isLoading, setIsLoading] = useState(false);
  const [result, setResult] = useState<utils.MonthlyReleaseResult | null>(null);

  const loadData = useCallback(async () => {
    setIsLoading(true);
    try {
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
              <GameItem key={game.getchu_id} game={game} />
            ))}
          </div>
        </div>
      ))}
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

// 单个游戏卡片组件
function GameItem({ game }: { game: utils.MonthlyReleaseGame }) {
  const { t } = useTranslation();
  const [imgError, setImgError] = useState(false);

  const handleClick = () => {
    if (game.detail_url) {
      BrowserOpenURL(game.detail_url);
    }
  };

  // 优先用本地路径，失败回退原始 URL
  const getImgSrc = () => {
    if (game.cover_url) {
      const local = getLocalPath(game.cover_url);
      if (local) return local;
      return game.cover_url;
    }
    return "";
  };

  const imgSrc = getImgSrc();

  return (
    <div
      onClick={handleClick}
      className="group cursor-pointer rounded-lg bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700 overflow-hidden hover:shadow-lg hover:border-primary-400 dark:hover:border-primary-500 transition-all duration-200 hover:-translate-y-0.5"
    >
      {/* 封面图 */}
      <div className="relative aspect-[3/4] bg-brand-100 dark:bg-brand-700 overflow-hidden">
        {imgSrc && !imgError ? (
          <img
            src={imgSrc}
            alt={game.name}
            className="absolute inset-0 w-full h-full object-cover object-center group-hover:scale-105 transition-transform duration-300"
            onError={() => setImgError(true)}
            loading="lazy"
          />
        ) : (
          <div className="w-full h-full flex items-center justify-center">
            <div className="i-mdi-image-off text-4xl text-brand-400" />
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
