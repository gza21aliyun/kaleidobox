import { models } from "../../wailsjs/go/models";
import { TagGroupModal } from "../components/modal/TagGroupModal";
import { createRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { ListTags, DeleteTagGroup, UpdateTagsGroup, ListGroups } from "../../wailsjs/go/service/TagService";
import { Route as rootRoute } from "./__root";
import { toast } from "react-hot-toast";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/tag_list",
  component: TagListPage,
});

function TagListPage() {
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
      setError("无法加载标签和分组数据");
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
      setError("无法加载标签列表");
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
  const groupedTags = filteredTags.reduce((acc, tag) => {
    const category = tag.category || "未分类";
    if (!acc[category]) {
      acc[category] = [];
    }
    acc[category].push(tag);
    return acc;
  }, {} as Record<string, models.Tag[]>);

  // 按分组组织标签
  const tagsByGroup = tags.reduce((acc, tag) => {
    const group = tag.group || "未分组";
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
    if (!confirm(`确定要删除分组 "${groupName}" 吗？这将取消该分组下所有标签的分组关联。`)) {
      return;
    }

    try {
      await DeleteTagGroup(groupName);
      await loadTagsAndGroups();
      toast.success('分组删除成功');
    } catch (error) {
      console.error('删除分组失败:', error);
      toast.error('删除分组失败');
    }
  };

  const handleSaveGroup = async (groupName: string, selectedTags: string[]) => {
    try {
      await UpdateTagsGroup(selectedTags, groupName);
      await loadTagsAndGroups();
    } catch (error) {
      console.error('保存分组失败:', error);
      throw error;
    }
  };

  if (loading) {
    return (
      <div className="p-6">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-brand-900 dark:text-white">标签列表</h1>
          <p className="text-brand-600 dark:text-brand-400">加载中...</p>
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
          <h1 className="text-2xl font-bold text-brand-900 dark:text-white">标签列表</h1>
        </div>
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-6">
          <div className="flex items-center">
            <div className="i-mdi-alert-circle text-red-500 text-xl mr-3"></div>
            <div>
              <h3 className="font-medium text-red-800 dark:text-red-200">加载失败</h3>
              <p className="text-red-600 dark:text-red-400 mt-1">{error}</p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-brand-900 dark:text-white">标签列表</h1>
        <p className="text-brand-600 dark:text-brand-400">
          共找到 {filteredTags.length} 个标签，{groups.length} 个分组
        </p>
      </div>

      {/* 搜索框 */}
      <div className="mb-6">
        <div className="relative max-w-md">
          <input
            type="text"
            placeholder="搜索标签..."
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
                {searchQuery ? '未找到匹配的标签' : '暂无标签'}
              </h3>
              <p className="text-yellow-600 dark:text-yellow-400 mt-1">
                {searchQuery ? '请尝试其他关键词' : '还没有添加任何标签'}
              </p>
            </div>
          </div>
        </div>
      ) : (
        <>
          {/* 标签分类展示板块 */}
          <div className="mb-8">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-bold text-brand-900 dark:text-white flex items-center">
                <div className="i-mdi-tag-multiple mr-2 text-brand-600 dark:text-brand-400"></div>
                标签分类
              </h2>
            </div>
            
            <div className="space-y-6">
              {Object.entries(groupedTags).map(([category, categoryTags]) => (
                <div key={category} className="bg-white dark:bg-brand-800/30 rounded-lg p-4 border border-brand-200 dark:border-brand-700">
                  <h3 className="text-lg font-semibold text-brand-900 dark:text-white mb-3 flex items-center">
                    <div className="i-mdi-folder mr-2 text-brand-600 dark:text-brand-400"></div>
                    {category} ({categoryTags.length})
                  </h3>
                  <div className="flex flex-wrap gap-2">
                    {categoryTags.map((tag) => (
                      <span
                        key={tag.name}
                        className={`
                          inline-flex items-center px-3 py-1 rounded-full text-sm font-medium
                          ${tag.is_h
                            ? 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-200 border border-red-200 dark:border-red-800' 
                            : tag.is_spoiler
                            ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-200 border border-yellow-200 dark:border-yellow-800'
                            : 'bg-brand-100 text-brand-800 dark:bg-brand-900/30 dark:text-brand-200 border border-brand-200 dark:border-brand-700'
                          }
                          ${tag.block_modify ? 'opacity-75' : ''}
                        `}
                      >
                        {tag.name}
                        {tag.is_h && (
                          <div className="ml-1 text-xs" title="成人内容">
                            <div className="i-mdi-alert-circle-outline"></div>
                          </div>
                        )}
                        {tag.is_spoiler && (
                          <div className="ml-1 text-xs" title="剧透警告">
                            <div className="i-mdi-eye-off-outline"></div>
                          </div>
                        )}
                        {tag.block_modify && (
                          <div className="ml-1 text-xs" title="禁止修改">
                            <div className="i-mdi-lock-outline"></div>
                          </div>
                        )}
                      </span>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* 标签分组板块 */}
          <div>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-bold text-brand-900 dark:text-white flex items-center">
                <div className="i-mdi-folder-multiple mr-2 text-brand-600 dark:text-brand-400"></div>
                标签分组
              </h2>
              <button
                onClick={handleCreateGroup}
                className="flex items-center px-3 py-1.5 bg-brand-500 text-white rounded-lg hover:bg-brand-600 transition-colors text-sm"
              >
                <div className="i-mdi-plus mr-1"></div>
                添加分组
              </button>
            </div>

            {groups.length === 0 ? (
              <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-6 text-center">
                <div className="i-mdi-folder-outline text-4xl text-yellow-500 mx-auto mb-3"></div>
                <h3 className="font-medium text-yellow-800 dark:text-yellow-200 mb-1">暂无标签分组</h3>
                <p className="text-yellow-600 dark:text-yellow-400">点击上方按钮创建第一个标签分组</p>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {groups.map(groupName => (
                  <div key={groupName} className="bg-white dark:bg-brand-800/30 rounded-lg p-4 border border-brand-200 dark:border-brand-700 hover:shadow-md transition-shadow">
                    <div className="flex items-center justify-between mb-3">
                      <h3 className="font-semibold text-brand-900 dark:text-white flex items-center">
                        <div className="i-mdi-folder mr-2 text-brand-600 dark:text-brand-400"></div>
                        {groupName}
                      </h3>
                      <div className="flex gap-1">
                        <button
                          onClick={() => handleEditGroup(groupName)}
                          className="p-1 text-brand-600 hover:text-brand-800 dark:text-brand-400 dark:hover:text-brand-200"
                          title="编辑分组"
                        >
                          <div className="i-mdi-pencil text-sm"></div>
                        </button>
                        <button
                          onClick={() => handleDeleteGroup(groupName)}
                          className="p-1 text-red-600 hover:text-red-800"
                          title="删除分组"
                        >
                          <div className="i-mdi-delete text-sm"></div>
                        </button>
                      </div>
                    </div>
                    <div className="text-sm text-brand-600 dark:text-brand-400 mb-2">
                      包含 {tagsByGroup[groupName]?.length || 0} 个标签
                    </div>
                    <div className="flex flex-wrap gap-1">
                      {tagsByGroup[groupName]?.slice(0, 5).map(tag => (
                        <span
                          key={tag.name}
                          className="px-2 py-1 bg-brand-100 dark:bg-brand-900/30 text-brand-800 dark:text-brand-200 text-xs rounded-full"
                        >
                          {tag.name}
                        </span>
                      ))}
                      {tagsByGroup[groupName] && tagsByGroup[groupName].length > 5 && (
                        <span className="px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 text-xs rounded-full">
                          +{tagsByGroup[groupName].length - 5} 更多
                        </span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
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
  );
}