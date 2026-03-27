import { useTranslation } from "react-i18next";
import { models } from "../../../wailsjs/go/models";
import { useAppStore } from "../../store";
import { AddTagsForGames } from "../../../wailsjs/go/service/GameService";
import { ListTags, ListGroups } from "../../../wailsjs/go/service/TagService";
import { useEffect, useState } from "react";
import { toast } from "react-hot-toast";
import { arrayToMap } from "../utils/Utility";

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
  games,
}: AddTagModalProps) {
  const { t } = useTranslation();
  const { updateGamesInGames } = useAppStore();
  const [newTagInput, setNewTagInput] = useState("");
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [allTags, setAllTags] = useState<models.Tag[]>([]);
  const [groups, setGroups] = useState<string[]>([]);
  const [activeTab, setActiveTab] = useState<"category" | "group">("category");
  const [searchQuery, setSearchQuery] = useState("");

  useEffect(() => {
    if (isOpen) {
      setNewTagInput("");
      setSelectedTags([]);
      setSearchQuery("");
      loadTagsAndGroups();
    }
  }, [isOpen]);

  const loadTagsAndGroups = async () => {
    try {
      const [tagsResult, groupsResult] = await Promise.all([
        ListTags(),
        ListGroups(),
      ]);
      setAllTags(tagsResult || []);
      setGroups(groupsResult || []);
    } catch (err) {
      console.error("Failed to load tags and groups:", err);
      toast.error(t('tagList.toasts.loadTagGroupsFailed'));
    }
  };

  const handleAddTagToList = () => {
    if (newTagInput.trim() && !selectedTags.includes(newTagInput.trim())) {
      setSelectedTags([...selectedTags, newTagInput.trim()]);
      setNewTagInput("");
    }
  };

  const handleRemoveTag = (tag: string) => {
    setSelectedTags(selectedTags.filter((t) => t !== tag));
  };

  const handleToggleTag = (tagName: string) => {
    if (selectedTags.includes(tagName)) {
      setSelectedTags(selectedTags.filter((t) => t !== tagName));
    } else {
      setSelectedTags([...selectedTags, tagName]);
    }
  };

  const handleConfirm = async () => {
    if (selectedTags.length === 0) {
      toast.error(t('gameInfo.pleaseAddAtLeastOneTag'));
      return;
    }

    if (games.length === 0) {
      toast.error(t('gameInfo.noGamesSelected'));
      return;
    }

    try {
      const newGames = await AddTagsForGames([...games], selectedTags);
      updateGamesInGames(newGames);
      onConfirm(newGames);
      toast.success(
        t('gameInfo.tagsAddedSuccess', {
          count: selectedTags.length,
          gameCount: games.length,
        })
      );
    } catch (error) {
      console.error("Error adding tags:", error);
      toast.error(t('gameInfo.addTagsFailed'));
    }
  };

  // 按类别分组标签
  const tagsByCategory = arrayToMap(allTags, (tag) => tag.category);

  // 按分组组织标签
  const tagsByGroup = allTags.reduce((acc, tag) => {
    const group = tag.group || t('tagList.messages.ungrouped');
    if (!acc[group]) {
      acc[group] = [];
    }
    acc[group].push(tag);
    return acc;
  }, {} as Record<string, models.Tag[]>);

  // 过滤标签
  const filteredTags = allTags.filter(
    (tag) =>
      tag.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      (tag.category &&
        tag.category.toLowerCase().includes(searchQuery.toLowerCase())) ||
      (tag.group &&
        tag.group.toLowerCase().includes(searchQuery.toLowerCase()))
  );

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-4xl w-full max-h-[80vh] flex flex-col">
        {/* Header */}
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-xl font-semibold text-brand-900 dark:text-white">
            {t('gameInfo.addTag')}
            <span className="ml-2 text-sm font-normal text-gray-500 dark:text-gray-400">
              ({games.length} {t('gameInfo.gamesSelected')})
            </span>
          </h3>
          <button
            onClick={onClose}
            className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
          >
            ×
          </button>
        </div>

        {/* Selected Tags */}
        <div className="mb-4">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {t('gameInfo.selectedTags')} ({selectedTags.length})
          </label>
          <div className="min-h-[3rem] p-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 flex flex-wrap gap-2">
            {selectedTags.length === 0 ? (
              <span className="text-gray-400 text-sm">
                {t('gameInfo.noTagsSelected')}
              </span>
            ) : (
              selectedTags.map((tag) => (
                <span
                  key={tag}
                  className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-brand-100 text-brand-800 dark:bg-brand-900 dark:text-brand-200"
                >
                  {tag}
                  <button
                    onClick={() => handleRemoveTag(tag)}
                    className="ml-1 text-brand-600 hover:text-brand-800 dark:text-brand-400 dark:hover:text-brand-200"
                  >
                    ×
                  </button>
                </span>
              ))
            )}
          </div>
        </div>

        {/* Add New Tag Input */}
        <div className="mb-4">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {t('gameInfo.addNewTag')}
          </label>
          <div className="flex gap-2">
            <input
              type="text"
              value={newTagInput}
              onChange={(e) => setNewTagInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault();
                  handleAddTagToList();
                }
              }}
              className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-brand-900 dark:text-white"
              placeholder={t('gameInfo.enterTagName')}
            />
            <button
              onClick={handleAddTagToList}
              disabled={!newTagInput.trim() || selectedTags.includes(newTagInput.trim())}
              className="px-4 py-2 bg-brand-500 text-white rounded hover:bg-brand-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {t('gameInfo.add')}
            </button>
          </div>
        </div>

        {/* Search */}
        <div className="mb-4">
          <div className="relative">
            <input
              type="text"
              placeholder={t('tagList.searchPlaceholder')}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-brand-900 dark:text-white"
            />
            <div className="absolute left-3 top-2.5 text-gray-400">
              <div className="i-mdi-magnify text-lg"></div>
            </div>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-gray-200 dark:border-gray-700 mb-4">
          <button
            onClick={() => setActiveTab("category")}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === "category"
                ? "border-brand-500 text-brand-600 dark:text-brand-400"
                : "border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
            }`}
          >
            {t('tagList.sections.categories')}
          </button>
          <button
            onClick={() => setActiveTab("group")}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === "group"
                ? "border-brand-500 text-brand-600 dark:text-brand-400"
                : "border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
            }`}
          >
            {t('tagList.sections.groups')}
          </button>
        </div>

        {/* Tag List */}
        <div className="flex-1 overflow-y-auto min-h-[300px] max-h-[400px]">
          {searchQuery ? (
            // 搜索结果
            <div className="flex flex-wrap gap-2">
              {filteredTags.map((tag) => (
                <button
                  key={tag.name}
                  onClick={() => handleToggleTag(tag.name)}
                  className={`px-3 py-1 rounded-full text-sm font-medium transition-colors ${
                    selectedTags.includes(tag.name)
                      ? "bg-brand-500 text-white"
                      : "bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600"
                  }`}
                >
                  {tag.name}
                </button>
              ))}
            </div>
          ) : activeTab === "category" ? (
            // 按类别显示
            <div className="space-y-4">
              {Array.from(tagsByCategory.entries()).map(([category, tags]) => (
                <div
                  key={category}
                  className="border border-gray-200 dark:border-gray-700 rounded-lg p-3"
                >
                  <h4 className="font-medium text-brand-800 dark:text-brand-200 mb-2 flex items-center">
                    <div className="i-mdi-folder mr-2 text-brand-500"></div>
                    {category} ({tags.length})
                  </h4>
                  <div className="flex flex-wrap gap-2">
                    {tags.map((tag) => (
                      <button
                        key={tag.name}
                        onClick={() => handleToggleTag(tag.name)}
                        className={`px-3 py-1 rounded-full text-sm font-medium transition-colors ${
                          selectedTags.includes(tag.name)
                            ? "bg-brand-500 text-white"
                            : "bg-[#e0e000] text-brand-800 hover:bg-[#d0d000]"
                        }`}
                      >
                        {tag.name}
                      </button>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            // 按分组显示
            <div className="space-y-4">
              {groups.length === 0 ? (
                <div className="text-center py-8 text-gray-500">
                  {t('tagList.emptyState.noGroups')}
                </div>
              ) : (
                groups.map((groupName) => (
                  <div
                    key={groupName}
                    className="border border-gray-200 dark:border-gray-700 rounded-lg p-3"
                  >
                    <h4 className="font-medium text-brand-800 dark:text-brand-200 mb-2 flex items-center">
                      <div className="i-mdi-folder-multiple mr-2 text-brand-500"></div>
                      {groupName} ({tagsByGroup[groupName]?.length || 0})
                    </h4>
                    <div className="flex flex-wrap gap-2">
                      {tagsByGroup[groupName]?.map((tag) => (
                        <button
                          key={tag.name}
                          onClick={() => handleToggleTag(tag.name)}
                          className={`px-3 py-1 rounded-full text-sm font-medium transition-colors ${
                            selectedTags.includes(tag.name)
                              ? "bg-brand-500 text-white"
                              : "bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600"
                          }`}
                        >
                          {tag.name}
                        </button>
                      ))}
                    </div>
                  </div>
                ))
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex gap-4 justify-end mt-6 pt-4 border-t border-gray-200 dark:border-gray-700">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-gray-300 dark:bg-gray-600 text-brand-900 dark:text-white rounded hover:bg-gray-400 dark:hover:bg-gray-500 transition-colors"
          >
            {t('gameInfo.cancel')}
          </button>
          <button
            onClick={handleConfirm}
            disabled={selectedTags.length === 0}
            className="px-4 py-2 bg-brand-500 text-white rounded hover:bg-brand-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {t('gameInfo.confirm')} ({selectedTags.length})
          </button>
        </div>
      </div>
    </div>
  );
}
