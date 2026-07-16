import { useState } from 'react';
import { createPortal } from "react-dom";
import { useTranslation } from 'react-i18next';

export interface SaveResult {
  game_name: string;
  archive_name: string;
  save_path: string;
  source_path: string;
  game_id: string;
  game_path: string;
}

interface OverrideSaveModalProps {
  isOpen: boolean;
  results: SaveResult[];
  onClose: () => void;
  onConfirm: (selectedResults: SaveResult[]) => void;
}

export function OverrideSaveModal({
  isOpen,
  results,
  onClose,
  onConfirm,
}: OverrideSaveModalProps) {
  const { t } = useTranslation();
  const [selectedIds, setSelectedIds] = useState<string[]>(results.map(r => r.game_id));

  const handleSelectAll = () => {
    if (selectedIds.length === results.length) {
      setSelectedIds([]);
    } else {
      setSelectedIds(results.map(r => r.game_id));
    }
  };

  const handleSelectItem = (gameId: string) => {
    if (selectedIds.includes(gameId)) {
      setSelectedIds(selectedIds.filter(id => id !== gameId));
    } else {
      setSelectedIds([...selectedIds, gameId]);
    }
  };

  const handleConfirm = () => {
    const selectedResults = results.filter(r => selectedIds.includes(r.game_id));
    if (selectedResults.length === 0) {
      return;
    }
    onConfirm(selectedResults);
    onClose();
  };

  if (!isOpen)
    return null;

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
      <div className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-brand-800 border border-brand-200 dark:border-brand-700 max-h-[80vh] flex flex-col overflow-hidden">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-xl font-bold text-brand-900 dark:text-white">
            {t("downloadedFiles.overrideSaveConfirm")}
          </h3>
          <button
            onClick={onClose}
            className="text-brand-500 hover:text-brand-700 dark:text-brand-400 dark:hover:text-brand-300"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="flex-1 overflow-auto mb-4">
          <div className="flex items-center gap-2 mb-3">
            <input
              type="checkbox"
              checked={selectedIds.length === results.length}
              onChange={handleSelectAll}
              className="rounded border-brand-300 text-brand-600 focus:ring-neutral-500"
            />
            <span className="text-sm text-brand-600 dark:text-brand-400">
              {t("downloadedFiles.selectAll")}
            </span>
          </div>

          <div className="space-y-2">
            {results.map((result) => (
              <div
                key={result.game_id}
                className={`p-3 border rounded-lg ${
                  selectedIds.includes(result.game_id)
                    ? "border-brand-500 bg-brand-50 dark:bg-brand-700"
                    : "border-brand-200 dark:border-brand-700 bg-white dark:bg-brand-800"
                }`}
              >
                <div className="flex items-start gap-3">
                  <input
                    type="checkbox"
                    checked={selectedIds.includes(result.game_id)}
                    onChange={() => handleSelectItem(result.game_id)}
                    className="mt-1 rounded border-brand-300 text-brand-600 focus:ring-neutral-500"
                  />
                  <div className="flex-1 min-w-0">
                    <div className="font-medium text-brand-900 dark:text-white truncate" title={result.game_name}>
                      {result.game_name}
                    </div>
                    <div className="text-xs text-brand-500 mt-1">
                      <div className="truncate" title={result.archive_name}>
                        {t("downloadedFiles.archiveName")}: {result.archive_name}
                      </div>
                      <div className="truncate mt-1" title={result.save_path}>
                        {t("downloadedFiles.savePath")}: {result.save_path}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="flex justify-end gap-3">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-brand-700 hover:bg-brand-100 rounded-lg dark:text-brand-300 dark:hover:bg-brand-700 transition-colors"
          >
            {t("common.cancel")}
          </button>
          <button
            onClick={handleConfirm}
            disabled={selectedIds.length === 0}
            className="px-4 py-2 text-sm font-medium text-white bg-error-600 hover:bg-error-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {t("downloadedFiles.confirmOverride")}
          </button>
        </div>
      </div>
    </div>,
    document.body,
  );
}