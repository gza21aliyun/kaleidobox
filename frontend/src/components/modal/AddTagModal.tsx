import { useTranslation } from "react-i18next";
import { models } from "../../../wailsjs/go/models";
import { useAppStore } from "../../store";
import { AddTagsForGames } from "../../../wailsjs/go/service/GameService";
import { useEffect, useState } from "react";

interface AddTagModalProps {
  isOpen: boolean;
  games: models.Game[];
  onClose: () => void;
  onConfirm: (gs: models.Game[]) => void;
}

export function AddTagModal({ 
  isOpen, 
  onClose, 
  onConfirm, 
  games 
}: AddTagModalProps) {
  const { t } = useTranslation();
  const { updateGamesInGames } = useAppStore();  
  const [newTagInput, setNewTagInput] = useState("");

  useEffect(() => { 
    setNewTagInput("");
  }, [games]);

  const handleAddTag = async () => {
        if (newTagInput.trim()) {
            try {
                var tags = [newTagInput.trim()]
                const newGames = await AddTagsForGames([...games], tags);
                updateGamesInGames(newGames)
                onConfirm(newGames)
            } catch (error) {
                console.error("Error adding tag:", error);
            }
        }
    };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full">
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-xl font-semibold text-brand-900 dark:text-white">{t('gameInfo.addTag')}</h3>
          <button 
            onClick={onClose}
            className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
          >
            ×
          </button>
        </div>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('gameInfo.tagName')}</label>
            <input
              type="text"
              value={newTagInput}
              onChange={(e) => setNewTagInput(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-brand-900 dark:text-white"
              placeholder={t('gameInfo.enterTagName')}
            />
          </div>
          <div className="flex gap-4 justify-end mt-6">
            <button
              onClick={onClose}
              className="px-4 py-2 bg-gray-300 dark:bg-gray-600 text-brand-900 dark:text-white rounded hover:bg-gray-400 dark:hover:bg-gray-500 transition-colors"
            >
              {t('gameInfo.cancel')}
            </button>
            <button
              onClick={handleAddTag}
              className="px-4 py-2 bg-brand-500 text-white rounded hover:bg-brand-600 transition-colors"
            >
              {t('gameInfo.confirm')}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
