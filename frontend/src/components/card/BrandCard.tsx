import { useEffect, useState, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from "@tanstack/react-router";
import { models } from '../../../wailsjs/go/models';
import { UpdateTag } from '../../../wailsjs/go/service/TagService';
import { GetGamesByBrand } from '../../../wailsjs/go/service/GameService';
import { useAppStore } from '../../store';
import { ImageCard } from './ImageCard';

interface BrandCardProps {
  brand: models.Tag;
  viewMode?: "default" | "gallery";
}

export function BrandCard({ brand, viewMode = "default" }: BrandCardProps) {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { updateTagInTags } = useAppStore();
  const [games, setGames] = useState<models.Game[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [hasLoaded, setHasLoaded] = useState(false);
  const cardRef = useRef<HTMLDivElement>(null);

  const handleViewDetails = () => {
    navigate({ to: `/brand/${encodeURIComponent(brand.name)}` });
    brand.use_count++
    UpdateTag(brand)
    updateTagInTags(brand)
  };

  const loadBrandGames = async () => {
    if (isLoading || hasLoaded) {
      return;
    }

    setIsLoading(true);

    try {
      const result = await GetGamesByBrand(brand.name);
      setGames(result || []);
      setHasLoaded(true);
    } catch (error) {
      console.error(`Failed to load games for brand ${brand.name}:`, error);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    if (viewMode !== "gallery") {
      return;
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          loadBrandGames();
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
  }, [viewMode, brand.name]);

  const previewGames = games.filter(game => game.cover_url).slice(0, 3);

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
              <div className="i-mdi-store text-5xl" />
            </div>
          )}
        </div>

        <div className="p-4">
          <h3 className="font-semibold text-brand-900 dark:text-white group-hover:text-neutral-600 dark:group-hover:text-neutral-400 transition-colors">
            {brand.name}{brand.use_count > 0 ? ` (${brand.use_count})` : ''}
          </h3>
          <p className="text-sm text-brand-500 dark:text-brand-400 mt-1">
            {games.length} {t('brandList.labels.games')}
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
      <div className="p-3 rounded-lg mr-4 bg-neutral-100 text-neutral-600 dark:bg-neutral-900/30 dark:text-neutral-400">
        <div className="text-2xl i-mdi-store" />
      </div>
      <div className="flex-1">
        <h3 className="font-semibold text-brand-900 dark:text-white group-hover:text-neutral-600 dark:group-hover:text-neutral-400 transition-colors">
          {brand.name}{brand.use_count > 0 ? ` (${brand.use_count})` : ''}
        </h3>
      </div>
    </div>
  );
}