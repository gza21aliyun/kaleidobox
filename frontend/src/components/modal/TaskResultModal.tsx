import { createPortal } from "react-dom";
import { useEffect, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from 'react-i18next';
import { models } from "../../../wailsjs/go/models";
import { GetGamesByIdsStr } from "../../../wailsjs/go/service/GameService";

interface TaskResultModalProps {
  isOpen: boolean;
  resultGames: models.ResultGames[];
  onClose: () => void;
}

export function TaskResultModal({ isOpen, resultGames, onClose }: TaskResultModalProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [gamesMap, setGamesMap] = useState<Record<string, models.Game>>({});

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

  if (!isOpen) return null;

  const handleGameClick = (gameId: string) => {
    onClose();
    navigate({ to: `/game/${gameId}` });
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

  return createPortal(
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4"
      onClick={onClose}
    >
      <div
        className="w-full max-w-2xl max-h-[80vh] overflow-y-auto rounded-xl bg-white p-6 shadow-xl dark:bg-brand-800 border border-brand-200 dark:border-brand-700"
        onClick={e => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-xl font-bold text-brand-900 dark:text-white">{t('task.result.title')}</h3>
          <button
            onClick={onClose}
            className="p-1 rounded-md text-brand-500 hover:bg-brand-100 dark:hover:bg-brand-700 transition-colors"
          >
            <div className="i-mdi-close text-xl" />
          </button>
        </div>

        {(!resultGames || resultGames.length === 0) ? (
          <p className="text-brand-500 dark:text-brand-400 text-sm">{t('task.result.empty')}</p>
        ) : (
          <div className="space-y-4">
            {resultGames.map((rg, idx) => {
              const btnClass = getStatusBtn(rg.status);
              return (
                <div key={idx} className="rounded-lg p-4 border border-brand-200 dark:border-brand-700">
                  {rg.title && (
                    <h4 className="font-semibold text-brand-900 dark:text-white mb-1">{rg.title}</h4>
                  )}
                  {rg.description && (
                    <p className="text-sm text-brand-600 dark:text-brand-400 mb-2 whitespace-pre-wrap">{rg.description}</p>
                  )}
                  {rg.game_ids && rg.game_ids.length > 0 && (
                    <div className="flex flex-wrap gap-2 mt-2">
                      {rg.game_ids.map(gameId => {
                        const game = gamesMap[gameId];
                        return (
                          <button
                            key={gameId}
                            onClick={() => handleGameClick(gameId)}
                            className={`flex items-center gap-1 px-3 py-1.5 text-sm rounded-md transition-colors ${btnClass}`}
                            title={game?.name || gameId}
                          >
                            <div className="i-mdi-gamepad-variant-outline text-base" />
                            <span className="max-w-[200px] truncate">{game?.name || gameId}</span>
                          </button>
                        );
                      })}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}

        <div className="flex justify-end mt-6">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-brand-700 hover:bg-brand-100 rounded-lg dark:text-brand-300 dark:hover:bg-brand-700 transition-colors"
          >
            {t('common.close')}
          </button>
        </div>
      </div>
    </div>,
    document.body,
  );
}
