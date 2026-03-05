import { models } from "../../../wailsjs/go/models";
import { useEffect, useState } from "react";
import { useTranslation } from 'react-i18next';
import { tagMapForEach, workMapForEach, charactorsForEach, arrayToMap } from "../utils/Utility";
import { ListTags, ListGroups, UpdateTagsGroup, DeleteTagGroup } from "../../../wailsjs/go/service/TagService";

import { toast } from "react-hot-toast";

// 标签分组弹窗组件
interface GroupModalProps {
  isOpen: boolean;
  onClose: () => void;
  mode: 'create' | 'edit';
  groupName?: string;
  allTags: models.Tag[];
  existingGroups: string[];
  onSave: (groupName: string, selectedTags: string[]) => Promise<void>;
}

export function TagGroupModal({ isOpen, onClose, mode, groupName, allTags, existingGroups, onSave }: GroupModalProps) {
  const { t } = useTranslation();
  const [newGroupName, setNewGroupName] = useState(groupName || '');
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [isSaving, setIsSaving] = useState(false);
  const [expandedCategories, setExpandedCategories] = useState<Record<string, boolean>>({});

  // 按类别分组标签
  const tagsByCategory = arrayToMap(allTags, tag => tag.category);

  // 编辑模式下初始化选中的标签
  useEffect(() => {
    if (mode === 'edit' && groupName) {
      const tagsInGroup = allTags.filter(tag => tag.group === groupName).map(tag => tag.name);
      setSelectedTags(tagsInGroup);
    } else {
      setSelectedTags([]);
    }
    setNewGroupName(groupName || '');
  }, [mode, groupName, allTags]);

  if (!isOpen) return null;

  const handleSave = async () => {
    if (!newGroupName.trim()) {
      toast.error(t('tagGroup.errors.pleaseEnterGroupName'));
      return;
    }

    if (mode === 'create' && existingGroups.includes(newGroupName)) {
      toast.error(t('tagGroup.errors.groupNameExists'));
      return;
    }

    if (selectedTags.length === 0) {
      toast.error(t('tagGroup.errors.pleaseSelectAtLeastOneTag'));
      return;
    }

    try {
      setIsSaving(true);
      await onSave(newGroupName, selectedTags);
      toast.success(mode === 'create' ? t('tagGroup.toasts.groupCreatedSuccessfully') : t('tagGroup.toasts.groupUpdatedSuccessfully'));
      onClose();
    } catch (error) {
      console.error('保存分组失败:', error);
      toast.error(t('tagGroup.toasts.failedToSaveGroup'));
    } finally {
      setIsSaving(false);
    }
  };

  const handleTagToggle = (tagName: string) => {
    setSelectedTags(prev => 
      prev.includes(tagName) 
        ? prev.filter(name => name !== tagName)
        : [...prev, tagName]
    );
  };

  const toggleCategory = (category: string) => {
    setExpandedCategories(prev => ({
      ...prev,
      [category]: !prev[category]
    }));
  };

  const selectAllInCategory = (category: string) => {
    const categoryTags = (tagsByCategory.get(category) || []).map(tag => tag.name);
    setSelectedTags(prev => {
      // 如果该类别所有标签都已选中，则取消选择；否则全选
      const allSelected = categoryTags.every(tag => prev.includes(tag));
      if (allSelected) {
        return prev.filter(tag => !categoryTags.includes(tag));
      } else {
        return [...new Set([...prev, ...categoryTags])];
      }
    });
  };

  const areAllCategoryTagsSelected = (category: string) => {
    const categoryTags = (tagsByCategory.get(category) || []).map(tag => tag.name);
    return categoryTags.length > 0 && categoryTags.every(tag => selectedTags.includes(tag));
  };

  const areSomeCategoryTagsSelected = (category: string) => {
    const categoryTags = (tagsByCategory.get(category) || []).map(tag => tag.name);
    return categoryTags.some(tag => selectedTags.includes(tag));
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white dark:bg-brand-800 rounded-lg max-w-3xl w-full max-h-[90vh] overflow-hidden">
        <div className="p-6 border-b border-brand-200 dark:border-brand-700">
          <h2 className="text-xl font-bold text-brand-900 dark:text-white">
            {mode === 'create' ? t('tagGroup.modals.createGroup.title') : t('tagGroup.modals.editGroup.title')}
          </h2>
        </div>
        
        <div className="p-6 overflow-y-auto max-h-[70vh]">
          <div className="mb-4">
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
              {t('tagGroup.labels.groupName')}
            </label>
            <input
              type="text"
              value={newGroupName}
              onChange={(e) => setNewGroupName(e.target.value)}
              className="w-full px-3 py-2 border border-brand-200 dark:border-brand-700 rounded-lg bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-500"
              placeholder={t('tagGroup.placeholders.enterGroupName')}
            />
          </div>
          
          <div className="mb-4">
            <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">
              {t('tagGroup.labels.selectTags', { count: selectedTags.length })}
            </label>
            
            <div className="border border-brand-200 dark:border-brand-700 rounded-lg bg-brand-50 dark:bg-brand-900/20 overflow-hidden">
              {tagMapForEach(tagsByCategory, (category, tags) => (
                <div key={category} className="border-b border-brand-200 dark:border-brand-700 last:border-b-0">
                  {/* 类别标题栏 */}
                  <div 
                    className="flex items-center justify-between p-3 bg-white dark:bg-brand-800 cursor-pointer hover:bg-brand-100 dark:hover:bg-brand-700 transition-colors"
                    onClick={() => toggleCategory(category)}
                  >
                    <div className="flex items-center">
                      <div className="i-mdi-folder mr-2 text-brand-600 dark:text-brand-400"></div>
                      <span className="font-medium text-brand-900 dark:text-white">
                        {category} ({tags.length})
                      </span>
                    </div>
                    
                    <div className="flex items-center gap-2">
                      {/* 全选/取消全选按钮 */}
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          selectAllInCategory(category);
                        }}
                        className={`
                          px-2 py-1 text-xs rounded transition-colors
                          ${areAllCategoryTagsSelected(category)
                            ? 'bg-brand-500 text-white'
                            : areSomeCategoryTagsSelected(category)
                            ? 'bg-brand-300 text-white'
                            : 'bg-brand-100 text-brand-700 dark:bg-brand-700 dark:text-brand-300'
                          }
                        `}
                      >
                        {areAllCategoryTagsSelected(category) ? t('tagGroup.buttons.deselectAll') : t('tagGroup.buttons.selectAll')}
                      </button>
                      
                      {/* 展开/收起图标 */}
                      <div className={`
                        transform transition-transform duration-200
                        ${expandedCategories[category] ? 'rotate-90' : ''}
                      `}>
                        <div className="i-mdi-chevron-right text-brand-600 dark:text-brand-400"></div>
                      </div>
                    </div>
                  </div>
                  
                  {/* 标签列表 */}
                  {expandedCategories[category] && (
                    <div className="p-3 bg-brand-50 dark:bg-brand-900/10">
                      <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                        {tags.map(tag => (
                          <div
                            key={tag.name}
                            onClick={() => handleTagToggle(tag.name)}
                            className={`
                              p-2 rounded cursor-pointer transition-colors flex items-center
                              ${selectedTags.includes(tag.name)
                                ? 'bg-brand-500 text-white'
                                : 'bg-white dark:bg-brand-800 text-brand-700 dark:text-brand-300 hover:bg-brand-100 dark:hover:bg-brand-700'
                              }
                            `}
                          >
                            <div className={`
                              w-4 h-4 rounded border mr-2 flex items-center justify-center
                              ${selectedTags.includes(tag.name)
                                ? 'bg-white border-white'
                                : 'border-brand-300 dark:border-brand-600'
                              }
                            `}>
                              {selectedTags.includes(tag.name) && (
                                <div className="i-mdi-check text-xs text-brand-500"></div>
                              )}
                            </div>
                            <span className="truncate text-sm">{tag.name}</span>
                            
                            {/* 标签特殊标识 */}
                            <div className="ml-auto flex gap-1">
                              {tag.is_h && (
                                <div className="text-red-500" title="成人内容">
                                  <div className="i-mdi-alert-circle-outline text-xs"></div>
                                </div>
                              )}
                              {tag.is_spoiler && (
                                <div className="text-yellow-500" title="剧透警告">
                                  <div className="i-mdi-eye-off-outline text-xs"></div>
                                </div>
                              )}
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
        
        <div className="p-6 border-t border-brand-200 dark:border-brand-700 flex justify-end gap-3">
          <button
            onClick={onClose}
            className="px-4 py-2 text-brand-700 dark:text-brand-300 hover:bg-brand-100 dark:hover:bg-brand-700 rounded-lg transition-colors"
            disabled={isSaving}
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSave}
            disabled={isSaving}
            className="px-4 py-2 bg-brand-500 text-white rounded-lg hover:bg-brand-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center"
          >
            {isSaving ? (
              <>
                <div className="i-mdi-loading animate-spin mr-2"></div>
                {t('common.saving')}
              </>
            ) : t('common.save')}
          </button>
        </div>
      </div>
    </div>
  );
}