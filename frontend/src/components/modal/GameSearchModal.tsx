import { useState, useEffect } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { useAppStore } from "../../store";
import { GameCard } from "../card/GameCard";

interface GameSearchModalProps {
  itemName: string;
  onClose: () => void;
}

export function GameSearchModal({ itemName, onClose }: GameSearchModalProps) {
  const { t } = useTranslation();
  const [searchQuery, setSearchQuery] = useState(itemName);
  const { games, fetchGames, gamesLoading } = useAppStore();
  const [filteredGames, setFilteredGames] = useState<any[]>([]);

  useEffect(() => {
    if (games.length === 0) {
      fetchGames();
    }
  }, []);

  useEffect(() => {
    if (!gamesLoading && games.length > 0) {
      setFilteredGames(games);
    }
  }, [gamesLoading, games]);

  useEffect(() => {
    if (searchQuery.trim()) {
      searchGames(searchQuery);
    } else {
      setFilteredGames(games);
    }
  }, [searchQuery, games]);

  const searchGames = (query: string) => {
    if (!query.trim()) {
      setFilteredGames(games);
      return;
    }

    // 前端过滤游戏
    const results = games.filter((game: any) => {
      const name = game.name?.toLowerCase() || "";
      const searchName = game.search_name?.toLowerCase() || "";
      const lowerQuery = query.toLowerCase();
      return name.includes(lowerQuery) || searchName.includes(lowerQuery);
    });
    setFilteredGames(results);
  };

  const handleSearch = () => {
    searchGames(searchQuery);
  };

  const getSimilarityScore = (game: any, query: string): number => {
    const lowerQuery = query.toLowerCase();
    const lowerName = game.name?.toLowerCase() || "";
    const lowerSearchName = game.search_name?.toLowerCase() || "";

    if (lowerName === lowerQuery) return 100;
    if (lowerName.startsWith(lowerQuery)) return 90;
    if (lowerName.includes(lowerQuery)) return 70;
    if (lowerSearchName.includes(lowerQuery)) return 50;
    return 10;
  };

  const sortedGames = [...filteredGames].sort((a, b) => {
    return getSimilarityScore(b, searchQuery) - getSimilarityScore(a, searchQuery);
  });

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
      <div className="w-full max-w-4xl max-h-[80vh] rounded-xl bg-white dark:bg-brand-800 shadow-xl border border-brand-200 dark:border-brand-700 flex flex-col">
        {/* 标题栏 */}
        <div className="flex items-center justify-between p-4 border-b border-brand-200 dark:border-brand-700">
          <h3 className="text-lg font-bold text-brand-900 dark:text-white">
            {t("downloadedFiles.searchTitle")}
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
          <div className="flex gap-2">
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  handleSearch();
                }
              }}
              placeholder={t("downloadedFiles.searchPlaceholder")}
              className="flex-1 px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-lg bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
            <button
              onClick={handleSearch}
              disabled={gamesLoading}
              className="px-4 py-2 bg-brand-600 hover:bg-brand-700 text-white rounded-lg disabled:opacity-50"
            >
              {t("downloadedFiles.search")}
            </button>
          </div>
        </div>

        {/* 游戏列表 */}
        <div className="flex-1 overflow-y-auto p-4">
          {gamesLoading ? (
            <div className="flex items-center justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand-600"></div>
            </div>
          ) : sortedGames.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-8 text-brand-400">
              <div className="i-mdi-magnify text-4xl mb-2" />
              <p>{t("downloadedFiles.noSearchResults")}</p>
            </div>
          ) : (
            <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
              {sortedGames.map((game) => (
                <GameCard
                  key={game.id}
                  game={game}
                  viewMode="small"
                  searchQuery={searchQuery}
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>,
    document.body
  );
}
