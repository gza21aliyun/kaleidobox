import { models } from "../../wailsjs/go/models";
import { TagGroupModal } from "../components/modal/TagGroupModal";
import { DraggableTag, DroppableGroup } from "../components/card/DraggableTag";
import { createRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { ListTags, DeleteTagGroup, UpdateTagsGroup, ListGroups } from "../../wailsjs/go/service/TagService";
import { Route as rootRoute } from "./__root";
import { toast } from "react-hot-toast";
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { tagMapForEach, arrayToMap } from "../components/utils/Utility";
import { useTranslation } from 'react-i18next';

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/tag_list",
  component: TagListPage,
});

function TagListPage() {
  const { t } = useTranslation();
  const [tags, setTags] = useState<models.Tag[]>([]);
  const [groups, setGroups] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [isGroupModalOpen, setIsGroupModalOpen] = useState(false);
  const [groupModalMode, setGroupModalMode] = useState<'create' | 'edit'>('create');
  const [editingGroupName, setEditingGroupName] = useState<string | undefined>(undefined);


  useEffect(() => {
    loadTagsAndGroups();
  }, []);

  const loadTagsAndGroups = async () => {
    try {
      setLoading(true);
      setError(null);
      const [tagsResult, groupsResult] = await Promise.all([
        ListTags(),
        ListGroups()
      ]);
      setTags(tagsResult || []);
      setGroups(groupsResult || []);
    } catch (err) {
      console.error("Failed to load data:", err);
      setError(t('tagList.toasts.loadTagGroupsFailed'));
    } finally {
      setLoading(false);
    }
  };

  const loadTags = async () => {
    try {
      setLoading(true);
      setError(null);
      const result = await ListTags();
      setTags(result || []);
    } catch (err) {
      console.error("Failed to load tags:", err);
      setError(t('tagList.toasts.loadTagsFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 过滤标签
  const filteredTags = tags.filter(tag => 
    tag.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    (tag.category && tag.category.toLowerCase().includes(searchQuery.toLowerCase())) ||
    (tag.group && tag.group.toLowerCase().includes(searchQuery.toLowerCase()))
  );

  // 按类别分组标签
  const groupedTags = arrayToMap(tags, tag => tag.category);

  // 按分组组织标签
  const tagsByGroup = tags.reduce((acc, tag) => {
    const group = tag.group || t('tagList.messages.ungrouped');
    if (!acc[group]) {
      acc[group] = [];
    }
    acc[group].push(tag);
    return acc;
  }, {} as Record<string, models.Tag[]>);

  const handleCreateGroup = () => {
    setGroupModalMode('create');
    setEditingGroupName(undefined);
    setIsGroupModalOpen(true);
  };

  const handleEditGroup = (groupName: string) => {
    setGroupModalMode('edit');
    setEditingGroupName(groupName);
    setIsGroupModalOpen(true);
  };

  const handleDeleteGroup = async (groupName: string) => {
    if (!confirm(t('tagList.modals.deleteGroupMessage', { name: groupName }))) {
      return;
    }

    try {
      await DeleteTagGroup(groupName);
      await loadTagsAndGroups();
      toast.success(t('tagList.toasts.groupDeleteSuccess'));
    } catch (error) {
      console.error(t('tagList.toasts.groupDeleteFailed') + ':', error);
      toast.error(t('tagList.toasts.groupDeleteFailed'));
    }
  };

  const handleSaveGroup = async (groupName: string, selectedTags: string[]) => {
    try {
      await UpdateTagsGroup(selectedTags, groupName);
      await loadTagsAndGroups();
    } catch (error) {
      console.error(t('tagList.toasts.saveGroupFailed') + ':', error);
      throw error;
    }
  };

  

    const handleTagDrop = async (tagName: string, targetGroup: string) => {
    try {
        await UpdateTagsGroup([tagName], targetGroup);
        
        // 局部更新状态而不是重新加载所有数据
        setTags(prevTags => 
        prevTags.map(tag => 
            tag.name === tagName 
            ? { ...tag, group: targetGroup }
            : tag
        )
        );
        
        // 更新分组统计
        setGroups(prevGroups => {
        // 如果是新分组，添加到分组列表
        if (!prevGroups.includes(targetGroup)) {
            return [...prevGroups, targetGroup];
        }
        return prevGroups;
        });
        
        toast.success(t('tagList.toasts.tagAddedToGroup', { tagName, targetGroup }));
    } catch (error) {
        console.error(t('tagList.toasts.dragAddTagFailed') + ':', error);
        toast.error(t('tagList.toasts.addTagToGroupFailed'));
    }
    };

  if (loading) {
    return (
      <div className="p-6">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-brand-900 dark:text-white">{t('tagList.title')}</h1>
          <p className="text-brand-600 dark:text-brand-400">{t('common.loading')}</p>
        </div>
        <div className="animate-pulse">
          <div className="space-y-4">
            {[...Array(5)].map((_, index) => (
              <div key={index} className="bg-white dark:bg-brand-800/30 rounded-lg p-4 border border-brand-200 dark:border-brand-700">
                <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/4 mb-3"></div>
                <div className="flex flex-wrap gap-2">
                  {[...Array(6)].map((_, tagIndex) => (
                    <div key={tagIndex} className="h-6 bg-gray-200 dark:bg-gray-700 rounded-full px-3"></div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-brand-900 dark:text-white">{t('tagList.title')}</h1>
        </div>
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-6">
          <div className="flex items-center">
            <div className="i-mdi-alert-circle text-red-500 text-xl mr-3"></div>
            <div>
              <h3 className="font-medium text-red-800 dark:text-red-200">{t('tagList.errors.loadFailed')}</h3>
              <p className="text-red-600 dark:text-red-400 mt-1">{error}</p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <DndProvider backend={HTML5Backend}>
      <div className="p-6 h-full">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-brand-900 dark:text-white">{t('tagList.title')}</h1>
          <p className="text-brand-600 dark:text-brand-400">
            {t('tagList.stats.found', { tags: filteredTags.length, groups: groups.length })}
            <span className="ml-2 text-sm text-brand-500">{t('tagList.hints.dragToAdd')}</span>
          </p>
        </div>

        {/* 搜索框 */}
        <div className="mb-6">
          <div className="relative max-w-md">
            <input
              type="text"
              placeholder={t('tagList.searchPlaceholder')}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border border-brand-200 dark:border-brand-700 rounded-lg bg-white dark:bg-brand-800 text-brand-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-500"
            />
            <div className="absolute left-3 top-2.5 text-brand-400 dark:text-brand-500">
              <div className="i-mdi-magnify text-lg"></div>
            </div>
          </div>
        </div>

        {filteredTags.length === 0 ? (
          <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-6">
            <div className="flex items-center">
              <div className="i-mdi-information text-yellow-500 text-xl mr-3"></div>
              <div>
                <h3 className="font-medium text-yellow-800 dark:text-yellow-200">
                  {searchQuery ? t('tagList.messages.noMatchingTags') : t('tagList.messages.noTags')}
                </h3>
                <p className="text-yellow-600 dark:text-yellow-400 mt-1">
                  {searchQuery ? t('tagList.messages.tryOtherKeywords') : t('tagList.messages.noTagsAdded')}
                </p>
              </div>
            </div>
          </div>
        ) : (
          <div className="flex gap-6 h-[calc(100vh-200px)]">
            {/* 标签板块 (4/5 宽度) */}
            <div className="w-3/4 flex flex-col">
              <div className="flex items-center justify-between mb-4 flex-shrink-0">
                <h2 className="text-xl font-bold text-brand-900 dark:text-white flex items-center">
                  <div className="i-mdi-tag-multiple mr-2 text-brand-600 dark:text-brand-400"></div>
                  {t('tagList.sections.categories')}
                </h2>
              </div>
              
              <div className="flex-1 overflow-y-auto pr-2">
                <div className="space-y-6">
                  {tagMapForEach(groupedTags, (category, categoryTags) => (
                    <div key={category} className="bg-white dark:bg-brand-800/30 rounded-lg p-4 border border-brand-200 dark:border-brand-700">
                      <h3 className="text-lg font-semibold text-brand-900 dark:text-white mb-3 flex items-center">
                        <div className="i-mdi-folder mr-2 text-brand-600 dark:text-brand-400"></div>
                        {category} ({categoryTags.length})
                      </h3>
                      <div className="flex flex-wrap gap-2">
                        {categoryTags.map((tag) => (
                          <DraggableTag key={tag.name} tag={tag} />
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* 标签分组板块 (1/5 宽度) */}
            <div className="w-1/4 flex flex-col">
              <div className="flex items-center justify-between mb-4 flex-shrink-0">
                <h2 className="text-xl font-bold text-brand-900 dark:text-white flex items-center">
                  <div className="i-mdi-folder-multiple mr-2 text-brand-600 dark:text-brand-400"></div>
                  {t('tagList.sections.groups')}
                </h2>
                <button
                  onClick={handleCreateGroup}
                  className="flex items-center px-2 py-1 bg-brand-500 text-white rounded-lg hover:bg-brand-600 transition-colors text-sm"
                >
                  <div className="i-mdi-plus text-sm"></div>
                </button>
              </div>

              <div className="flex-1 overflow-y-auto pr-2">
                {groups.length === 0 ? (
                  <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4 text-center h-full flex items-center justify-center">
                    <div>
                      <div className="i-mdi-folder-outline text-3xl text-yellow-500 mx-auto mb-2"></div>
                      <h3 className="font-medium text-yellow-800 dark:text-yellow-200 text-sm mb-1">{t('tagList.emptyState.noGroups')}</h3>
                      <p className="text-yellow-600 dark:text-yellow-400 text-xs">{t('tagList.emptyState.clickToCreate')}</p>
                    </div>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {groups.map(groupName => (
                      <DroppableGroup
                        key={groupName}
                        groupName={groupName}
                        tagCount={tagsByGroup[groupName]?.length || 0}
                        tags={tagsByGroup[groupName]}
                        onDrop={handleTagDrop}
                        onEdit={handleEditGroup}
                        onDelete={handleDeleteGroup}
                      />
                    ))}
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* 分组弹窗 */}
        <TagGroupModal
          isOpen={isGroupModalOpen}
          onClose={() => setIsGroupModalOpen(false)}
          mode={groupModalMode}
          groupName={editingGroupName}
          allTags={tags}
          existingGroups={groups}
          onSave={handleSaveGroup}
        />
      </div>
    </DndProvider>
  );
}