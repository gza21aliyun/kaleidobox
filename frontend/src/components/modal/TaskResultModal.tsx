import { createPortal } from "react-dom";
import { useEffect, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from 'react-i18next';
import { models } from "../../../wailsjs/go/models";
import { GetGamesByIdsStr, OpenLocalPath, DeleteFolder } from "../../../wailsjs/go/service/GameService";
import { LocalSearchModal } from "./LocalSearchModal";
import toast from "react-hot-toast";

interface TaskResultModalProps {
  isOpen: boolean;
  resultGames: models.ResultGames[];
  title: string;
  onClose: () => void;
}

const PAGE_SIZE = 100;

export function TaskResultModal({ isOpen, resultGames, title, onClose }: TaskResultModalProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [gamesMap, setGamesMap] = useState<Record<string, models.Game>>({});
  const [searchTerm, setSearchTerm] = useState("");
  const [limit, setLimit] = useState(PAGE_SIZE);
  const [searchDirPath, setSearchDirPath] = useState<string | null>(null);
  const [deletedPaths, setDeletedPaths] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (!isOpen || !resultGames || resultGames.length === 0) return;

    const allIds = Array.from(new Set(resultGames.flatMap(rg => rg.game_ids || [])));
    if (allIds.length === 0) return;

    GetGamesByIdsStr(allIds.join(','))
      .then(games => {
        const map: Record<string, models.Game> = {};
        (games || []).forEach(g => {
          if (g.id) map[g.id] = g;
        });
        setGamesMap(map);
      })
      .catch(err => {
        console.error("Failed to load games:", err);
      });
  }, [isOpen, resultGames]);

  // 搜索词变化时重置分页
  useEffect(() => {
    setLimit(PAGE_SIZE);
  }, [searchTerm]);

  if (!isOpen) return null;

  const handleGameClick = (gameId: string, filteredGameIdsStr: string[]) => {
    onClose();
    navigate({
       to: `/game/${gameId}`,
       search: { filteredGameIdsStr }
       });
  };

  const statusConfig: Record<number, { btn: string }> = {
    200: {
      btn: "bg-success-100 text-success-800 hover:bg-success-200 dark:bg-success-900/30 dark:text-success-400 dark:hover:bg-success-800/50",
    },
    400: {
      btn: "bg-error-100 text-error-800 hover:bg-error-200 dark:bg-error-900/30 dark:text-error-400 dark:hover:bg-error-800/50",
    },
    300: {
      btn: "bg-warning-100 text-warning-800 hover:bg-warning-200 dark:bg-warning-900/30 dark:text-warning-400 dark:hover:bg-warning-800/50",
    },
  };
  const getStatusBtn = (status: number) => statusConfig[status]?.btn
    || "bg-blue-100 text-blue-800 hover:bg-blue-200 dark:bg-blue-900/30 dark:text-blue-400 dark:hover:bg-blue-800/50";

  const isPath = (id: string) => /[\\\/]/.test(id);

  const trimSearch = searchTerm.trim().toLowerCase();
  const filterIds = (ids: string[] | undefined) => {
    if (!ids || ids.length === 0) return [];
    if (!trimSearch) return ids;
    return ids.filter(id => {
      if (isPath(id)) {
        return id.toLowerCase().includes(trimSearch);
      }
      const name = gamesMap[id]?.name || "";
      return name.toLowerCase().includes(trimSearch) || id.toLowerCase().includes(trimSearch);
    });
  };

  return createPortal(
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4"
      onClick={onClose}
    >
      <div
        className="w-full max-w-3xl max-h-[85vh] flex flex-col rounded-xl bg-white shadow-xl dark:bg-brand-800 border border-brand-200 dark:border-brand-700"
        onClick={e => e.stopPropagation()}
      >
        {/* 头部（固定） */}
        <div className="flex items-center justify-between p-6 pb-3 shrink-0">
          <h3 className="text-xl font-bold text-brand-900 dark:text-white">{t('task.result.title')}:{title}</h3>
          <button
            onClick={onClose}
            className="p-1 rounded-md text-brand-500 hover:bg-brand-100 dark:hover:bg-brand-700 transition-colors"
          >
            <div className="i-mdi-close text-xl" />
          </button>
        </div>

        {/* 搜索框（固定） */}
        <div className="px-6 pb-3 shrink-0">
          <div className="relative">
            <div className="i-mdi-magnify absolute left-3 top-1/2 -translate-y-1/2 text-brand-400" />
            <input
              type="text"
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              placeholder={t('task.result.searchPlaceholder')}
              className="w-full pl-9 pr-3 py-2 text-sm rounded-md border border-brand-200 dark:border-brand-700 bg-white dark:bg-brand-900 text-brand-900 dark:text-white placeholder-brand-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>

        {/* 内容区（滚动） */}
        <div className="flex-1 overflow-y-auto px-6 pb-3">
          {(!resultGames || resultGames.length === 0) ? (
            <p className="text-brand-500 dark:text-brand-400 text-sm">{t('task.result.empty')}</p>
          ) : (
            <div className="space-y-4">
              {resultGames.map((rg, idx) => {
                const btnClass = getStatusBtn(rg.status);
                const filtered = filterIds(rg.game_ids);
                const visible = filtered.slice(0, limit);
                const hasMore = filtered.length > visible.length;

                return (
                  <div key={idx} className="rounded-lg p-4 border border-brand-200 dark:border-brand-700">
                    <div className="flex items-center justify-between mb-2">
                      <div>
                        {/* {rg.title && (
                          <h4 className="font-semibold text-brand-900 dark:text-white">{rg.title}</h4>
                        )} */}
                        {rg.description && (
                          <p className="text-sm text-brand-600 dark:text-brand-400">{rg.description}</p>
                        )}
                      </div>
                      <span className="text-xs text-brand-400 dark:text-brand-500 shrink-0 ml-2">
                        {filtered.length}{trimSearch ? ` / ${rg.game_ids?.length || 0}` : ""}
                      </span>
                    </div>

                    {visible.length > 0 ? (
                      <>
                        <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 mt-2 max-h-[300px] overflow-y-auto">
                          {visible.map(gameId => {
                            const isDir = isPath(gameId);
                            const game = !isDir ? gamesMap[gameId] : undefined;
                            const isDeleted = isDir && deletedPaths.has(gameId);
                            return isDir ? (
                              <div
                                key={gameId}
                                className={`flex items-center gap-1 px-2 py-1.5 text-sm rounded-md ${isDeleted ? 'opacity-40 line-through' : ''} ${btnClass}`}
                                title={gameId}
                              >
                                <div className="flex items-center gap-0.5 shrink-0">
                                  <button
                                    type="button"
                                    onClick={async () => {
                                      try { await OpenLocalPath(gameId); }
                                      catch (err) { console.error("OpenLocalPath failed:", err); }
                                    }}
                                    className="p-1 rounded hover:bg-black/10 dark:hover:bg-white/10 transition-colors"
                                    title={t('task.result.openFolder')}
                                  >
                                    <div className="i-mdi-folder-open-outline text-base" />
                                  </button>
                                  <button
                                    type="button"
                                    onClick={() => setSearchDirPath(gameId)}
                                    className="p-1 rounded hover:bg-black/10 dark:hover:bg-white/10 transition-colors"
                                    title={t('task.result.searchLocal')}
                                  >
                                    <div className="i-mdi-magnify text-base" />
                                  </button>
                                  <button
                                    type="button"
                                    onClick={async () => {
                                      try {
                                        await DeleteFolder(gameId);
                                        setDeletedPaths(prev => new Set(prev).add(gameId));
                                        toast.success(t('task.result.deleteSuccess'));
                                      } catch (err) {
                                        console.error("DeleteFolder failed:", err);
                                        toast.error(t('task.result.deleteFailed'));
                                      }
                                    }}
                                    className="p-1 rounded hover:bg-red-500/20 transition-colors"
                                    title={t('task.result.deleteFolder')}
                                  >
                                    <div className="i-mdi-delete-outline text-base" />
                                  </button>
                                </div>
                                <div className="i-mdi-folder-outline text-base shrink-0" />
                                <span className="truncate">{gameId}</span>
                              </div>
                            ) : (
                              <button
                                key={gameId}
                                onClick={() => handleGameClick(gameId, visible)}
                                className={`flex items-center gap-1 px-3 py-1.5 text-sm rounded-md transition-colors ${btnClass}`}
                                title={game?.name || gameId}
                              >
                                <div className="i-mdi-gamepad-variant-outline text-base shrink-0" />
                                <span className="truncate">{game?.name || gameId}</span>
                              </button>
                            );
                          })}
                        </div>
                        {hasMore && (
                          <button
                            onClick={() => setLimit(l => l + PAGE_SIZE)}
                            className="mt-2 w-full py-1.5 text-xs text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-md transition-colors"
                          >
                            {t('task.result.loadMore', { shown: visible.length, total: filtered.length })}
                          </button>
                        )}
                      </>
                    ) : (
                      <p className="text-xs text-brand-400 dark:text-brand-500 mt-1">
                        {trimSearch ? t('task.result.noMatch') : t('task.result.empty')}
                      </p>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* 底部（固定） */}
        <div className="flex justify-end p-6 pt-3 shrink-0 border-t border-brand-100 dark:border-brand-700">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-brand-700 hover:bg-brand-100 rounded-lg dark:text-brand-300 dark:hover:bg-brand-700 transition-colors"
          >
            {t('common.close')}
          </button>
        </div>
      </div>
      {searchDirPath && (
        <LocalSearchModal
          itemName={searchDirPath.split(/[\\/]/).pop() || searchDirPath}
          onOpenInfo={() => {}}
          onClose={() => setSearchDirPath(null)}
        />
      )}
    </div>,
    document.body,
  );
}
