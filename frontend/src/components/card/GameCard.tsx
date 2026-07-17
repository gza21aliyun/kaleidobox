import React, { useEffect, useRef, useState } from "react";
import type { models, vo } from "../../../wailsjs/go/models";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "react-hot-toast";
import { useTranslation } from 'react-i18next';
import { enums } from "../../../wailsjs/go/models";
import { StartGameWithTracking } from "../../../wailsjs/go/service/StartService";
import { GetGamesByTag } from "../../../wailsjs/go/service/GameService";
import { FetchImages } from "../../../wailsjs/go/service/ImageService";
import { ImageCard } from "./ImageCard";
import { formatDurationSimple, formatLastDateText, formatLocalDate, parseTime } from "../../utils/time";
import { useAppStore } from "../../store";

// ── 高亮工具：将文本中匹配 query 的部分高亮显示 ──────────────────────────────
function HighlightText({ text, query }: { text: string; query: string }) {
  if (!query || !text) {
    return <>{text}</>;
  }
  const q = query.toLowerCase();
  const idx = text.toLowerCase().indexOf(q);
  if (idx === -1) {
    return <>{text}</>;
  }
  return (
    <>
      {text.slice(0, idx)}
      <mark className="bg-yellow-300/80 dark:bg-yellow-500/50 text-inherit rounded-[2px] px-[1px]">
        {text.slice(idx, idx + query.length)}
      </mark>
      {text.slice(idx + query.length)}
    </>
  );
}

// ── 懒加载游戏统计信息组件 ──────────────────────────────────────────────────────
function LazyGameStats({ game_id, stats }: { game_id: string, stats: vo.GameDetailStats | undefined }) {
  // const [isVisible, setIsVisible] = useState(false);
  // const containerRef = useRef<HTMLDivElement>(null);


  // useEffect(() => {
  //   const observer = new IntersectionObserver(
  //     ([entry]) => {
  //       if (entry.isIntersecting) {
          
  //         setIsVisible(true);
  //         observer.disconnect();
  //       }
  //     },
  //     { threshold: 0.1 }
  //   );

  //   if (containerRef.current) {
  //     observer.observe(containerRef.current);
  //   }

  //   return () => observer.disconnect();
  // }, []);
  if (!stats) {
    return null;
  }

    const endDate = formatLastDateText(stats.end_date) ?? ""
    return (
      <div className="absolute right-1 bottom-16 z-10 flex flex-col rounded-md bg-black/30 px-2 py-1 text-xs text-white/90 backdrop-blur-sm shadow-lg"> 
        <div className="flex items-center whitespace-nowrap">
          玩过{formatDurationSimple(stats.total_play_time)}
        </div>
        {stats.end_date && (
          <div className="mt-0.5 flex items-center whitespace-nowrap border-t border-white/20 pt-0.5">
            {
            endDate
            // stats.end_date
            }玩过
          </div>
        )}
      </div>
    );
}

// ─────────────────────────────────────────────────────────────────────────────

interface GameCardProps {
  game: models.Game;
  selectionMode?: boolean;
  selected?: boolean;
  onSelectChange?: (selected: boolean, event?: React.MouseEvent) => void;
  /** 当前搜索词，用于高亮游戏名和开发商 */
  searchQuery?: string;
  filteredGameIdsStr?: string[];
  /** 视图模式 */
  viewMode?: "list" | "small" | "large" | "gallery";
  onDelete?: (game: models.Game) => void;
}

export function GameCard({
  game,
  selectionMode = false,
  selected = false,
  onSelectChange,
  searchQuery = "",
  filteredGameIdsStr = [],
  viewMode = "small",
  onDelete,
}: GameCardProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { gameStats } = useAppStore();

  const [galleryImages, setGalleryImages] = useState<models.ImageBackup[]>([]);
  const [isGalleryLoading, setIsGalleryLoading] = useState(false);
  const cardRef = useRef<HTMLDivElement>(null);

  const handleToggleSelect = (e: React.MouseEvent) => {
    e.stopPropagation();
    onSelectChange?.(!selected, e);
  };

  const handleStartGame = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (game.id) {
      try {
        const started = await StartGameWithTracking(game.id);
        if (started) {
          toast.success(t('game.toasts.gameLaunchSuccess', { name: game.name }));
        }
        else {
          toast.error(t('game.toasts.gameLaunchFailed', { name: game.name }));
        }
      }
      catch (error) {
        console.error("Failed to start game:", error);
        toast.error(t('game.toasts.gameLaunchFailedCheckLog', { name: game.name }));
      }
    }
  };

  const handleViewDetails = () => {
    navigate({ 
      to: `/game/${game.id}`,
      search: { filteredGameIdsStr },
     });
  };

  const handleCompanyClick = async (companyName: string) => {
    try {
      const games = await GetGamesByTag(companyName);
      const gameIds = games.map(g => g.id).filter((id): id is string => !!id);
      if (gameIds.length > 0) {
        navigate({
          to: '/category_games',
          search: {
            selectedGameIds: gameIds.join(','),
            title: companyName,
          } as Record<string, string>
        });
      }
    } catch (error) {
      console.error('Failed to load games for company:', error);
      toast.error('Failed to load games for this company');
    }
  };

  const handleReleaseDateClick = () => {
    if (game.release_at) {
      const date = parseTime(game.release_at);
      const year = date.getFullYear();
      const month = date.getMonth() + 1;
      localStorage.setItem('monthlyReleases_year', year.toString());
      localStorage.setItem('monthlyReleases_month', month.toString());
      navigate({
        to: '/monthly_releases',
      });
    }
  };

  useEffect(() => {
    if (viewMode !== "gallery") return;

    const timer = setTimeout(() => {
      loadGalleryImages();
    }, 100);

    return () => clearTimeout(timer);
  }, [viewMode, game.id]);

  const loadGalleryImages = async () => {
    if (!game.id) return;
    setIsGalleryLoading(true);
    try {
      const screenshots = await FetchImages(game.id, 0, 3, false);
      const images = await FetchImages(game.id, 0, 2, false);
      const allImages = [...(screenshots || []), ...(images || [])]
        .filter(img => img.url && img.url.trim() !== "" && !img.url.endsWith("pl.jpg"));
      setGalleryImages(allImages);
    } catch (error) {
      console.error("Failed to load gallery images:", error);
    } finally {
      setIsGalleryLoading(false);
    }
  };

  const isCompleted = game.status === enums.GameStatus.COMPLETED;
  const companyDisplay = game.company || "Unknown Developer";

  const stats = gameStats.get(game.id);
  // console.log('LazyGameStats', game.id, stats, gameStats);

  
  

  if (viewMode === "list") {
    return (
      <div
        className={`glass-card group relative flex w-full items-center gap-4 overflow-hidden rounded-xl border border-brand-100 bg-white p-3 shadow-sm transition-all duration-300 hover:shadow-xl dark:border-brand-700 dark:bg-brand-800 ${selectionMode ? "cursor-pointer" : ""} ${selectionMode && selected ? "ring-2 ring-neutral-500 dark:ring-neutral-400" : ""}`}
        onClick={selectionMode ? handleToggleSelect : undefined}
      >
        {selectionMode && (
          <button
            type="button"
            onClick={handleToggleSelect}
            className={`absolute left-2 top-1/2 z-10 flex h-6 w-6 -translate-y-1/2 items-center justify-center rounded-full border
                        ${selected
              ? "bg-neutral-600 text-white border-neutral-600"
              : "bg-white/90 text-transparent border-brand-300 dark:bg-brand-800/90 dark:border-brand-600"}
                        shadow-sm`}
            title={selected ? t('common.cancelSelection') : t('common.select')}
          >
            <div className="i-mdi-check text-sm" />
          </button>
        )}
        
        <div className="relative h-16 w-12 flex-shrink-0 overflow-hidden rounded-lg bg-brand-200 dark:bg-brand-700">
          {game.cover_url ? (
            <ImageCard
              url={game.cover_url}
              alt={game.name}
              lazyLoad={true}
              referrerPolicy="no-referrer"
              className="absolute inset-0 w-full h-full object-cover object-center"
              onDragStart={e => e.preventDefault()}
            />
          ) : (
            <div className="flex h-full items-center justify-center text-brand-400">
              <div className="i-mdi-image-off text-xl" />
            </div>
          )}
          {isCompleted && (
            <div className="absolute -top-1 -right-1 flex h-4 w-4 items-center justify-center rounded-full bg-yellow-500 shadow-lg">
              <div className="i-mdi-trophy text-[10px] text-white" />
            </div>
          )}
        </div>

        <div className="min-w-0 flex-1 grid gap-x-4 gap-y-1">
          <div className="min-w-0">
            <h3 className="truncate text-sm font-bold text-brand-900 dark:text-white" title={game.name}>
              <HighlightText text={game.name} query={searchQuery} />
            </h3>
            <p className="truncate text-xs text-brand-500 dark:text-brand-400" title={companyDisplay}>
              {game.company ? (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleCompanyClick(game.company);
                  }}
                  className="text-purple-600 hover:text-purple-800 dark:text-purple-400 dark:hover:text-purple-300 hover:underline transition-colors"
                >
                  <HighlightText text={companyDisplay} query={searchQuery} />
                </button>
              ) : (
                <HighlightText text={companyDisplay} query={searchQuery} />
              )}
            </p>
            {game.search_name && (
              <p className="truncate text-xs text-brand-400 dark:text-brand-500" title={game.search_name}>
                <span className="text-brand-300 dark:text-brand-600">Search: </span>
                <HighlightText text={game.search_name} query={searchQuery} />
              </p>
            )}
            {game.path && (
              <p className="truncate text-xs text-brand-400 dark:text-brand-500" title={game.path}>
                <span className="text-brand-300 dark:text-brand-600">Path: </span>
                <HighlightText text={game.path} query={searchQuery} />
              </p>
            )}
          </div>
          {/* {} */}
          
        </div>

        <div className="flex flex-shrink-0 items-center gap-2">
          {game.release_at ? (
            <button
              onClick={(e) => {
                e.stopPropagation();
                handleReleaseDateClick();
              }}
              className="text-xs text-emerald-600 hover:text-emerald-800 dark:text-emerald-400 dark:hover:text-emerald-300 hover:underline transition-colors"
            >
              {formatLocalDate(game.release_at)}
            </button>
          ) : (
            <p className="text-xs text-brand-500 dark:text-brand-400">
              {formatLocalDate(game.release_at)}
            </p>
          )}
          <div className="flex items-center gap-2 ml-2">
            <button
              onClick={handleStartGame}
              className="flex h-8 w-8 items-center justify-center rounded-full bg-neutral-600 text-white shadow-lg transition-transform hover:scale-110 hover:bg-neutral-500 active:scale-95"
              title={t('game.buttons.launchGame')}
            >
              <div className="i-mdi-play text-base" />
            </button>
            <button
              onClick={handleViewDetails}
              className="flex h-8 w-8 items-center justify-center rounded-full bg-brand-500 text-white shadow-lg transition-transform hover:scale-110 hover:bg-brand-400 active:scale-95"
              title={t('common.viewDetails')}
            >
              <div className="i-mdi-information-variant text-base" />
            </button>
          </div>
        </div>
      </div>
    );
  }

  if (viewMode === "gallery") {
    return (
      <div
        ref={cardRef}
        className={`glass-card group relative flex w-full items-stretch gap-4 overflow-hidden rounded-xl border border-brand-100 bg-white p-3 shadow-sm transition-all duration-300 hover:shadow-xl dark:border-brand-700 dark:bg-brand-800 ${selectionMode ? "cursor-pointer" : ""} ${selectionMode && selected ? "ring-2 ring-neutral-500 dark:ring-neutral-400" : ""}`}
        onClick={selectionMode ? handleToggleSelect : undefined}
      >
        {selectionMode && (
          <button
            type="button"
            onClick={handleToggleSelect}
            className={`absolute left-2 top-1/2 z-10 flex h-6 w-6 -translate-y-1/2 items-center justify-center rounded-full border
                        ${selected
              ? "bg-neutral-600 text-white border-neutral-600"
              : "bg-white/90 text-transparent border-brand-300 dark:bg-brand-800/90 dark:border-brand-600"}
                        shadow-sm`}
            title={selected ? t('common.cancelSelection') : t('common.select')}
          >
            <div className="i-mdi-check text-sm" />
          </button>
        )}

        <div className="flex-shrink-0 w-28 flex flex-col">
          <div className="relative aspect-[3/3.6] w-full overflow-hidden rounded-lg bg-brand-200 dark:bg-brand-700">
            {game.cover_url
              ? (
                  <ImageCard
                    url={game.cover_url}
                    alt={game.name}
                    lazyLoad={true}
                    referrerPolicy="no-referrer"
                    className="absolute inset-0 w-full h-full object-cover object-center"
                    onDragStart={e => e.preventDefault()}
                  />
                )
              : (
                  <div className="flex h-full items-center justify-center text-brand-400">
                    <div className="i-mdi-image-off text-4xl" />
                  </div>
                )}

            {isCompleted && (
              <div className="absolute top-1.5 right-1.5 flex h-6 w-6 items-center justify-center rounded-full bg-yellow-500 shadow-lg">
                <div className="i-mdi-trophy text-sm text-white" />
              </div>
            )}

            {!selectionMode && (
              <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-black/40 opacity-0 backdrop-blur-[2px] transition-all duration-300 group-hover:opacity-100">
                <button
                  onClick={handleStartGame}
                  className="flex h-8 w-8 items-center justify-center rounded-full bg-neutral-600 text-white shadow-lg transition-transform hover:scale-110 hover:bg-neutral-500 active:scale-95"
                  title={t('game.buttons.launchGame')}
                >
                  <div className="i-mdi-play text-lg" />
                </button>
                <button
                  onClick={handleViewDetails}
                  className="flex h-8 w-8 items-center justify-center rounded-full bg-white/20 text-white backdrop-blur-md transition-transform hover:scale-110 hover:bg-white/30 active:scale-95"
                  title={t('common.viewDetails')}
                >
                  <div className="i-mdi-information-variant text-lg" />
                </button>
              </div>
            )}
          </div>

          <div className="px-1 mt-1">
            <h3 className="truncate text-sm font-bold text-brand-900 dark:text-white leading-tight" title={game.name}>
              <HighlightText text={game.name} query={searchQuery} />
            </h3>
            <p className="truncate text-xs text-brand-500 dark:text-brand-400 leading-tight" title={companyDisplay}>
              {game.company ? (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleCompanyClick(game.company);
                  }}
                  className="text-purple-600 hover:text-purple-800 dark:text-purple-400 dark:hover:text-purple-300 hover:underline transition-colors"
                >
                  <HighlightText text={companyDisplay} query={searchQuery} />
                </button>
              ) : (
                <HighlightText text={companyDisplay} query={searchQuery} />
              )}
            </p>
            {game.release_at ? (
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  handleReleaseDateClick();
                }}
                className="truncate text-xs text-emerald-600 hover:text-emerald-800 dark:text-emerald-400 dark:hover:text-emerald-300 hover:underline transition-colors leading-tight"
              >
                {formatLocalDate(game.release_at)}
              </button>
            ) : (
              <p className="truncate text-xs text-brand-500 dark:text-brand-400 leading-tight">
                {formatLocalDate(game.release_at)}
              </p>
            )}
          </div>
        </div>

        <div className="flex-1 h-full overflow-hidden">
          <div className="flex h-full overflow-x-auto">
            {isGalleryLoading ? (
              <div className="flex h-full gap-1.5">
                {[1, 2, 3, 4, 5, 6].map(i => (
                  <div
                    key={i}
                    className="h-full w-36 flex-shrink-0 rounded-lg bg-brand-200 dark:bg-brand-700 animate-pulse"
                  />
                ))}
              </div>
            ) : galleryImages.length > 0 ? (
              galleryImages.map((image, index) => (
                <ImageCard
                  key={image.url}
                  url={image.url}
                  alt={`Gallery ${index + 1}`}
                  lazyLoad={true}
                  urls={galleryImages.map((i)=>i.url)}
                  referrerPolicy="no-referrer"
                  className="h-45 aspect-video flex-shrink-0 rounded-lg object-cover object-center hover:opacity-80 transition-opacity cursor-pointer"
                  onDragStart={e => e.preventDefault()}
                />
              ))
            ) : (
              <div className="text-sm text-brand-400 dark:text-brand-500 italic flex items-center h-full px-2">
                No gallery images
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div
      className={`glass-card group relative flex w-full flex-col overflow-hidden rounded-xl border border-brand-100 bg-white shadow-sm transition-all duration-300 hover:shadow-xl dark:border-brand-700 dark:bg-brand-800 ${selectionMode ? "cursor-pointer" : ""} ${selectionMode && selected ? "ring-2 ring-neutral-500 dark:ring-neutral-400" : ""} ${viewMode === "large" ? "aspect-[8/11]" : ""}`}
      onClick={selectionMode ? handleToggleSelect : undefined}
    >
      {selectionMode && (
        <button
          type="button"
          onClick={handleToggleSelect}
          className={`absolute left-2 top-2 z-10 flex h-6 w-6 items-center justify-center rounded-full border
                      ${selected
          ? "bg-neutral-600 text-white border-neutral-600"
          : "bg-white/90 text-transparent border-brand-300 dark:bg-brand-800/90 dark:border-brand-600"}
                      shadow-sm`}
          title={selected ? t('common.cancelSelection') : t('common.select')}
        >
          <div className="i-mdi-check text-sm" />
        </button>
      )}
      <LazyGameStats game_id={game.id} stats={stats} />
      
      <div className="relative aspect-[3/3.6] w-full overflow-hidden bg-brand-200 dark:bg-brand-700">
        {game.cover_url
          ? (
              <ImageCard
                url={game.cover_url}
                alt={game.name}
                lazyLoad={true}
                referrerPolicy="no-referrer"
                className="absolute inset-0 w-full h-full object-cover object-center transition-transform duration-500 group-hover:scale-110"
                onDragStart={e => e.preventDefault()}
              />
            )
          : (
              <div className="flex h-full items-center justify-center text-brand-400">
                <div className="i-mdi-image-off text-4xl" />
              </div>
            )}

        {/* 已通关奖杯标识 */}
        {isCompleted && (
          <div className="absolute top-1.5 right-1.5 flex h-6 w-6 items-center justify-center rounded-full bg-yellow-500 shadow-lg">
            <div className="i-mdi-trophy text-sm text-white" />
          </div>
        )}

        {!selectionMode && (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-black/40 opacity-0 backdrop-blur-[2px] transition-all duration-300 group-hover:opacity-100">
            <button
              onClick={handleStartGame}
              className="flex h-8 w-8 items-center justify-center rounded-full bg-neutral-600 text-white shadow-lg transition-transform hover:scale-110 hover:bg-neutral-500 active:scale-95"
              title={t('game.buttons.launchGame')}
            >
              <div className="i-mdi-play text-lg" />
            </button>
            <button
              onClick={handleViewDetails}
              className="flex h-8 w-8 items-center justify-center rounded-full bg-white/20 text-white backdrop-blur-md transition-transform hover:scale-110 hover:bg-white/30 active:scale-95"
              title={t('common.viewDetails')}
            >
              <div className="i-mdi-information-variant text-lg" />
            </button>
            {onDelete && (
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onDelete(game);
                }}
                className="flex h-8 w-8 items-center justify-center rounded-full bg-red-600/80 text-white backdrop-blur-md transition-transform hover:scale-110 hover:bg-red-500/80 active:scale-95"
                title={t('common.delete')}
              >
                <div className="i-mdi-delete text-lg" />
              </button>
            )}
          </div>
        )}
      </div>

      <div className="px-2 pt-1 pb-2">
        <h3 className="truncate text-sm font-bold text-brand-900 dark:text-white leading-tight" title={game.name}>
          <HighlightText text={game.name} query={searchQuery} />
        </h3>
        <p className="truncate text-xs text-brand-500 dark:text-brand-400 leading-tight" title={companyDisplay}>
          {game.company ? (
            <button
              onClick={(e) => {
                e.stopPropagation();
                handleCompanyClick(game.company);
              }}
              className="text-purple-600 hover:text-purple-800 dark:text-purple-400 dark:hover:text-purple-300 hover:underline transition-colors"
            >
              <HighlightText text={companyDisplay} query={searchQuery} />
            </button>
          ) : (
            <HighlightText text={companyDisplay} query={searchQuery} />
          )}
        </p>
        {game.release_at ? (
          <button
            onClick={(e) => {
              e.stopPropagation();
              handleReleaseDateClick();
            }}
            className="truncate text-xs text-emerald-600 hover:text-emerald-800 dark:text-emerald-400 dark:hover:text-emerald-300 hover:underline transition-colors leading-tight"
          >
            {formatLocalDate(game.release_at)}
          </button>
        ) : (
          <p className="truncate text-xs text-brand-500 dark:text-brand-400 leading-tight">
            {formatLocalDate(game.release_at)}
          </p>
        )}
      </div>
    </div>
  );
}
