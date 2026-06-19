import { useEffect, useState, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from "@tanstack/react-router";
import { models, vo, enums } from '../../../wailsjs/go/models';
import { UpdateTag } from '../../../wailsjs/go/service/TagService';
import { GetGamesByTag } from '../../../wailsjs/go/service/GameService';
import { GetGamesByCategory } from '../../../wailsjs/go/service/CategoryService';
import { GetWorksByStaffIdAndRole } from '../../../wailsjs/go/service/WorkService';
import { getGamesForCategory, type CategoryType } from '../../utils/categoryGames';
import { useAppStore } from '../../store';
import { ImageCard } from './ImageCard';

interface CategoryListCardProps {
  id: string;
  name: string;
  type: CategoryType;
  game_count?: number;
  use_count?: number;
  viewMode?: "default" | "gallery";
  original?: models.Tag | vo.CategoryVO | models.Staff;
  dirPath?: string;
  gameIds?: string[];
  categoryMap?: Map<string, string[]>;
  onUpdateCategoryMap?: (id: string, gameIds: string[]) => void;
}

export function CategoryListCard({
  id,
  name,
  type,
  game_count = 0,
  use_count = 0,
  viewMode = "default",
  original,
  dirPath,
  gameIds,
  categoryMap,
  onUpdateCategoryMap,
}: CategoryListCardProps) {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { games: storeGames, updateTagInTags } = useAppStore();
  const [games, setGames] = useState<models.Game[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [hasLoaded, setHasLoaded] = useState(false);
  const cardRef = useRef<HTMLDivElement>(null);

  const isPathCategory = type === 'parent1' || type === 'parent2';
  const isTagCategory = type === 'brand' || type === 'series' || type === 'genre';
  const displayCount = isTagCategory ? use_count : game_count;

  const getPathGameIds = () => {
    if (!isPathCategory || !dirPath) return [];
    const prefix = dirPath + '/';
    return storeGames
      .filter(g => {
        if (!g.path) return false;
        const normalized = g.path.replace(/\\/g, '/');
        return normalized === dirPath || normalized.startsWith(prefix);
      })
      .map(g => g.id);
  };

  const handleViewDetails = () => {
    if (isPathCategory) {
      const pathGameIds = getPathGameIds();
      if (pathGameIds.length > 0) {
        navigate({ 
          to: '/category_games',
          search: {
            selectedGameIds: pathGameIds.join(','),
            title: dirPath || name,
          } as Record<string, string>
        });
      }
      return;
    }
    if (type === 'favorite') {
      navigate({ to: `/favorites/${id}` });
    } else if (type === 'brand') {
      GetGamesByTag(name).then(games => {
        const gameIds = games.map(g => g.id).filter((id): id is string => !!id);
        if (gameIds.length > 0) {
          navigate({
            to: '/category_games',
            search: {
              selectedGameIds: gameIds.join(','),
              title: name,
            } as Record<string, string>
          });
        }
      });
      if (original && 'use_count' in original) {
        original.use_count++;
        UpdateTag(original as models.Tag)
        updateTagInTags(original as models.Tag)
      }
    } else if (type === 'series') {
      GetGamesByTag(name).then(games => {
        const gameIds = games.map(g => g.id).filter((id): id is string => !!id);
        if (gameIds.length > 0) {
          navigate({
            to: '/category_games',
            search: {
              selectedGameIds: gameIds.join(','),
              title: name,
            } as Record<string, string>
          });
        }
      });
      if (original && 'use_count' in original) {
        original.use_count++;
        UpdateTag(original as models.Tag)
        updateTagInTags(original as models.Tag)
      }
    } else if (type === 'genre') {
      const genreGames = storeGames.filter(g => {
        const tags = g.tags?.split(',') || [];
        return tags.includes(name);
      }).map(g => g.id);
      if (genreGames.length > 0) {
        navigate({
          to: '/category_games',
          search: {
            selectedGameIds: genreGames.join(','),
            title: name,
          } as Record<string, string>
        });
      }
      if (original && 'use_count' in original) {
        original.use_count++;
        UpdateTag(original as models.Tag)
        updateTagInTags(original as models.Tag)
      }
    } else if (type === 'chara_design' || type === 'sceneario') {
      const staffModel = original as unknown as models.Staff;
      const role = type === 'chara_design' ? enums.StaffRole.CHARA_DESIGN : enums.StaffRole.SCENEARIO;
      GetWorksByStaffIdAndRole(staffModel.id, role).then(works => {
        const workGameIds = works.map(w => w.game_id).filter((id): id is string => !!id);
        if (workGameIds.length > 0) {
          navigate({
            to: '/category_games',
            search: {
              selectedGameIds: workGameIds.join(','),
              title: name,
            } as Record<string, string>
          });
        }
      });
    }
  };

  const isStaffCategory = type === 'chara_design' || type === 'sceneario';

  const handleStaffTitleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (!isStaffCategory) return;
    const staffModel = original as unknown as models.Staff;
    navigate({ to: `/staff/${staffModel.id}` });
  };

  const loadCategoryGames = async () => {
    if (isLoading || hasLoaded) {
      return;
    }

    const cachedGameIds = categoryMap?.get(id);
    if (cachedGameIds !== undefined) {
      const cachedGames = storeGames.filter(g => cachedGameIds.includes(g.id));
      setGames(cachedGames);
      setHasLoaded(true);
      return;
    }

    setIsLoading(true);

    try {
      const result = await getGamesForCategory({ type, id, name, dirPath, original }, storeGames);
      setGames(result || []);
      setHasLoaded(true);
      const newGameIds = (result || []).map(g => g.id);
      onUpdateCategoryMap?.(id, newGameIds);
    } catch (error) {
      console.error(`Failed to load games for category ${name}:`, error);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          loadCategoryGames();
          observer.disconnect();
        }
      },
      {
        threshold: 0.1,
        rootMargin: '50px'
      }
    );

    if (cardRef.current) {
      observer.observe(cardRef.current);
    }

    return () => {
      observer.disconnect();
    };
  }, [name, id, type]);

  const getTypeIcon = () => {
    switch (type) {
      case 'favorite':
        return "i-mdi-heart";
      case 'brand':
        return "i-mdi-store";
      case 'series':
        return "i-mdi-format-list-bulleted-type";
      case 'parent1':
      case 'parent2':
        return "i-mdi-folder";
      default:
        return "i-mdi-folder";
    }
  };

  const getTypeColorClass = () => {
    switch (type) {
      case 'favorite':
        return "bg-error-100 text-error-600 dark:bg-error-900/30 dark:text-error-400";
      case 'brand':
        return "bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400";
      case 'series':
        return "bg-green-100 text-green-600 dark:bg-green-900/30 dark:text-green-400";
      case 'parent1':
      case 'parent2':
        return "bg-neutral-100 text-neutral-600 dark:bg-neutral-900/30 dark:text-neutral-400";
      default:
        return "bg-neutral-100 text-neutral-600 dark:bg-neutral-900/30 dark:text-neutral-400";
    }
  };

  const cachedIds = categoryMap?.get(id);
  const previewGames = cachedIds && cachedIds.length > 0
    ? storeGames.filter(g => cachedIds.includes(g.id) && g.cover_url).slice(0, 3)
    : games.filter(game => game.cover_url).slice(0, 3);
  const actualGameCount = cachedIds !== undefined ? cachedIds.length : games.length;

  if (viewMode === "gallery") {
    return (
      <div
        ref={cardRef}
        className="glass-card flex flex-col bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700 rounded-xl shadow-sm hover:shadow-md transition-all text-left group cursor-pointer overflow-hidden"
        onClick={handleViewDetails}
      >
        <div className="relative w-full overflow-hidden bg-neutral-100 dark:bg-brand-700" style={{ aspectRatio: '16/9' }}>
          {isLoading ? (
            <div className="flex items-center justify-center h-full">
              <div className="w-8 h-8 border-4 border-brand-200 border-t-brand-500 rounded-full animate-spin"></div>
            </div>
          ) : previewGames.length > 0 ? (
            <>
              {previewGames.length === 1 && (
                <ImageCard
                  url={previewGames[0].cover_url}
                  alt={previewGames[0].name}
                  lazyLoad={true}
                  referrerPolicy="no-referrer"
                  className="absolute inset-0 w-full h-full object-cover object-center"
                  onDragStart={e => e.preventDefault()}
                />
              )}

              {previewGames.length === 2 && (
                <div className="grid grid-cols-2 absolute inset-0">
                  {previewGames.map((game) => (
                    <div key={game.id} className="relative h-full overflow-hidden">
                      <ImageCard
                        url={game.cover_url}
                        alt={game.name}
                        lazyLoad={true}
                        referrerPolicy="no-referrer"
                        className="absolute inset-0 w-full h-full object-cover object-center"
                        onDragStart={e => e.preventDefault()}
                      />
                    </div>
                  ))}
                </div>
              )}

              {previewGames.length >= 3 && (
                <div className="grid grid-cols-3 absolute inset-0">
                  {previewGames.slice(0, 3).map((game) => (
                    <div key={game.id} className="relative h-full overflow-hidden">
                      <ImageCard
                        url={game.cover_url}
                        alt={game.name}
                        lazyLoad={true}
                        referrerPolicy="no-referrer"
                        className="absolute inset-0 w-full h-full object-cover object-center"
                        onDragStart={e => e.preventDefault()}
                      />
                    </div>
                  ))}
                </div>
              )}
            </>
          ) : (
            <div className="flex items-center justify-center h-full text-neutral-400 dark:text-brand-400">
              <div className={`${getTypeIcon()} text-5xl`} />
            </div>
          )}
        </div>

        <div className="p-4">
          <h3 className="font-semibold text-brand-900 dark:text-white group-hover:text-neutral-600 dark:group-hover:text-neutral-400 transition-colors truncate">
            {isStaffCategory ? (
              <button
                onClick={handleStaffTitleClick}
                className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 hover:underline"
              >
                {name}
              </button>
            ) : (
              name
            )}
            {isTagCategory && displayCount > 0 ? ` (点击${displayCount}次)` : ''}
          </h3>
          <p className="text-sm text-brand-500 dark:text-brand-400 mt-1">
            {actualGameCount} {t('brandList.labels.games')}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div
      className="glass-card flex items-center p-4 bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700 rounded-xl shadow-sm hover:shadow-md transition-all text-left group cursor-pointer"
      onClick={handleViewDetails}
    >
      <div className={`p-3 rounded-lg mr-4 ${getTypeColorClass()}`}>
        <div className={`text-2xl ${getTypeIcon()}`} />
      </div>
      <div className="flex-1 min-w-0">
        <h3 className="font-semibold text-brand-900 dark:text-white group-hover:text-neutral-600 dark:group-hover:text-neutral-400 transition-colors truncate">
          {isStaffCategory ? (
            <button
              onClick={handleStaffTitleClick}
              className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 hover:underline"
            >
              {name}
            </button>
          ) : (
            name
          )}
          {isTagCategory && displayCount > 0 ? ` (点击${displayCount}次)` : ''}
        </h3>
        <p className="text-sm text-brand-500 dark:text-brand-400 mt-1">
          {actualGameCount} {t('brandList.labels.games')}
        </p>
      </div>
    </div>
  );
}
