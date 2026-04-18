import { useState } from "react";
import type { models } from "../../../wailsjs/go/models";
import { useTranslation } from 'react-i18next';
import { arrayMapString, mapToArray, tagMapForEach } from "../utils/Utility";
import { UpdateTag } from "../../../wailsjs/go/service/TagService";
import { useAppStore } from "../../store";

interface FilterChooseTagModalProps {
  isOpen: boolean;
  onClose: () => void;
  availableTags: Map<string, models.Tag[]>;
  tagsFilter: string[];
  onTagsFilterChange: (newTags: string[]) => void;
}

export function FilterChooseTagModal({
  isOpen,
  onClose,
  availableTags,
  tagsFilter,
  onTagsFilterChange
}: FilterChooseTagModalProps) {
  const { t } = useTranslation();
  const [expandedCategories, setExpandedCategories] = useState<Record<string, boolean>>({});
  const [selectedTagsInCategory, setSelectedTagsInCategory] = useState<Record<string, string[]>>({});
  const { updateTagInTags } = useAppStore()

  // 处理分类展开/收起
  const toggleCategory = (category: string) => {
    setExpandedCategories(prev => ({
      ...prev,
      [category]: !(prev[category] ?? true)
    }));
  };

  const isAllCategoriesClosed = () => {
    console.log("isAllCategories:start");
    for (const category of availableTags.keys()) {
      if (expandedCategories[category] ?? true) {
        console.log("isAllCategoriesClosed cate " + category + ":", expandedCategories[category]);
        
        return false;
      }
    }
    console.log("isAllCategories:", true);
    return true;
  }

  const toggleAllCategory = (isExpanded: boolean) => {
    console.log("toggleAll:", isExpanded)
    let rs: Record<string, boolean> = {}
    availableTags.keys().forEach(category => {
      rs[category] = isExpanded;
    });
    setExpandedCategories(rs);
  }

  // 全选/取消全选
  const toggleSelectAllInCategory = (category: string, tags: models.Tag[]) => {
    const tagNames = tags.map(tag => tag.name);
    const currentlySelected = selectedTagsInCategory[category] || [];
    
    const allSelected = tagNames.every(tagName => currentlySelected.includes(tagName));
    
    let newSelectedTags: string[];
    if (allSelected) {
      // 取消全选
      newSelectedTags = [];
      const newTagsFilter = tagsFilter.filter(tagName => !tagNames.includes(tagName));
      onTagsFilterChange(newTagsFilter);
    } else {
      // 全选
      newSelectedTags = tagNames;
      const newTagsFilter = [...new Set([...tagsFilter, ...tagNames])];
      onTagsFilterChange(newTagsFilter);
    }
    
    setSelectedTagsInCategory(prev => ({
      ...prev,
      [category]: newSelectedTags
    }));
  };


  // 检查分类是否全选
  const isCategoryFullySelected = (category: string, tags: models.Tag[]) => {
    const tagNames = tags.map(tag => tag.name);
    const selectedInCategory = selectedTagsInCategory[category] || [];
    return tagNames.length > 0 && tagNames.every(tagName => selectedInCategory.includes(tagName));
  };

  if (!isOpen) return null;

  return (
    <div 
      className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50"
      onClick={onClose}
    >
      <div 
        className="bg-white dark:bg-brand-900 rounded-lg shadow-xl w-[1200px] max-w-[95vw] max-h-[90vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-4 border-b border-brand-200 dark:border-brand-700 flex justify-between items-center">
          <div className="flex items-center">
            <h3 className="text-lg font-semibold text-brand-900 dark:text-white">{t('filter.modals.chooseTag.title')}</h3>
            {/* 折叠按钮 */}
            <button
              onClick={() => toggleAllCategory(isAllCategoriesClosed())}
              className="p-1 rounded hover:bg-brand-100 dark:hover:bg-brand-700 transition-colors"
            >
              <div className={`
                i-mdi-chevron-down text-brand-500 transition-transform duration-200
                ${!isAllCategoriesClosed() ? 'rotate-180' : ''}
              `} />
            </button>
          </div>
          
          <button 
            onClick={onClose}
            className="text-brand-500 hover:text-brand-700 dark:text-brand-400 dark:hover:text-brand-200"
          >
            <div className="i-mdi-close text-xl" />
          </button>
        </div>

        <div className="p-4">
          <div className="space-y-4">
            {tagMapForEach(availableTags, (category, tags) => {
              const isExpanded = expandedCategories[category] ?? true;
              const isFullySelected = isCategoryFullySelected(category, tags);
              
              return (
                <div key={category} className="border border-brand-200 dark:border-brand-700 rounded-lg p-4 bg-white dark:bg-brand-800/30">
                  {/* 分类标题栏 */}
                  <div className="flex items-center justify-between mb-3">
                    <h4 className="font-semibold text-brand-800 dark:text-brand-200 flex items-center">
                      <div className="i-mdi-folder-outline mr-2 text-brand-500" />
                      {category}
                      <span className="ml-2 text-xs bg-brand-100 dark:bg-brand-700 text-brand-600 dark:text-brand-300 px-2 py-1 rounded-full">
                        {tags.length}
                      </span>
                    </h4>
                    
                    <div className="flex items-center gap-2">
                      {/* 全选按钮 */}
                      <button
                        onClick={() => toggleSelectAllInCategory(category, tags)}
                        className={`
                          px-2 py-1 text-xs rounded transition-colors
                          ${isFullySelected
                            ? 'bg-brand-500 text-white'
                            : tags.length > 0
                            ? 'bg-brand-100 text-brand-700 dark:bg-brand-700 dark:text-brand-300 hover:bg-brand-200 dark:hover:bg-brand-600'
                            : 'bg-gray-100 text-gray-400 cursor-not-allowed'
                          }
                        `}
                        disabled={tags.length === 0}
                      >
                        {isFullySelected ? t('filter.buttons.deselectAll') : t('filter.buttons.selectAll')}
                      </button>
                      
                      {/* 折叠按钮 */}
                      <button
                        onClick={() => toggleCategory(category)}
                        className="p-1 rounded hover:bg-brand-100 dark:hover:bg-brand-700 transition-colors"
                      >
                        <div className={`
                          i-mdi-chevron-down text-brand-500 transition-transform duration-200
                          ${isExpanded ? 'rotate-180' : ''}
                        `} />
                      </button>
                    </div>
                  </div>
                  
                  {/* 标签列表 - 可折叠 */}
                  {isExpanded && (
                    <div className="flex flex-wrap gap-2">
                      {tags.map((tag) => (
                        <button
                          key={tag.name}
                          onClick={() => {
                            onTagsFilterChange([...tagsFilter, tag.name]);
                            tag.use_count++;
                            UpdateTag(tag)
                            updateTagInTags(tag);
                          }}
                          className="inline-flex items-center px-3 py-1.5 rounded-full text-xs font-medium
                                  bg-gradient-to-r from-[#e0e000] to-[#c0c000] 
                                  text-brand-800 dark:text-brand-900
                                  hover:from-[#d0d000] hover:to-[#b0b000]
                                  shadow-sm hover:shadow-md
                                  transform hover:-translate-y-0.5
                                  transition-all duration-200 cursor-pointer
                                  border border-[#d0d000]/30"
                        >
                          <div className="i-mdi-tag mr-1 text-xs" />
                          {tag.name}{tag.use_count > 0 ? ` (${tag.use_count})` : ''}
                        </button>
                      ))}
                      {tags.length === 0 && (
                        <p className="text-brand-500 dark:text-brand-400 text-sm italic">
                          {t('filter.messages.noTags')}
                        </p>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
            
            {availableTags.size === 0 && (
              <div className="text-center py-8">
                <div className="i-mdi-tag-off text-4xl text-brand-300 dark:text-brand-600 mx-auto mb-3" />
                <p className="text-brand-600 dark:text-brand-400">
                  {t('filter.messages.noAvailableTagCategories')}
                </p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}


interface FilterChooseGroupModalProps {
  isOpen: boolean;
  onClose: () => void;
  onTagsFilterChange: (value: string[]) => void;
  availableTags: Map<string, models.Tag[]>;
  tagsFilter: string[];
}

export function FilterChooseGroupModal({
  isOpen,
  onClose,
  onTagsFilterChange,
  availableTags: availableTags,
  tagsFilter,
}: FilterChooseGroupModalProps) {
  const { t } = useTranslation();
  // 获取标签分组数据
  const [expandedGroups, setExpandedGroups] = useState<Record<string, boolean>>({});
  const { updateTagInTags } = useAppStore();
  const getTagGroups = () => {
    const groupMap: Map<string, models.Tag[]> = new Map();
    const tagArray: models.Tag[] = mapToArray(availableTags || new Map());
    
    for (const tag of tagArray) {
      if (!tag.group || tag.group === "") {
        continue;
      }
      
      if (groupMap.has(tag.group)) {
        const existingTags = groupMap.get(tag.group) || [];
        groupMap.set(tag.group, [...existingTags, tag]);
      } else {
        groupMap.set(tag.group, [tag]);
      }
    }
    
    return groupMap;
  };

  const handleTagsChosen = (tags: string[]) => { 
    var groupTags = [...tagsFilter, ...tags]
    groupTags = [...new Set(groupTags)]
    onTagsFilterChange(groupTags)
  };

  // 处理分组展开/收起
  const toggleGroup = (groupName: string) => {
    setExpandedGroups(prev => ({
      ...prev,
      [groupName]: !prev[groupName]
    }));
  };


//   const [tagGroups] = useState<Map<string, string[]>>(() => getTagGroups());

  if (!isOpen) return null;

  return (
    <div 
      className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50"
      onClick={onClose}
    >
      <div 
        className="bg-white dark:bg-brand-900 rounded-lg shadow-xl w-[600px] max-w-[90vw] max-h-[80vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-4 border-b border-brand-200 dark:border-brand-700 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-brand-900 dark:text-white">{t('filter.modals.chooseTagGroup.title')}</h3>
          <button 
            onClick={onClose}
            className="text-brand-500 hover:text-brand-700 dark:text-brand-400 dark:hover:text-brand-200"
          >
            <div className="i-mdi-close text-xl" />
          </button>
        </div>

        <div className="p-4">
          {getTagGroups().size > 0 ? (
            <div className="space-y-3">
              {Array.from(getTagGroups().entries()).map(([groupName, tags]) => {
                const isExpanded = expandedGroups[groupName] ?? false;
                
                return (
                  <div 
                    key={groupName}
                    className="rounded-lg border border-brand-200 dark:border-brand-700 overflow-hidden"
                  >
                    {/* 分组标题栏 */}
                    <div 
                      className="flex items-center justify-between p-3 bg-white dark:bg-brand-800 cursor-pointer hover:bg-brand-50 dark:hover:bg-brand-700 transition-colors"
                      onClick={() => toggleGroup(groupName)}
                    >
                      <div className="flex items-center">
                        <div className="i-mdi-folder mr-2 text-brand-500" />
                        <div>
                          <span className="font-medium text-brand-900 dark:text-white">
                            {groupName}
                          </span>
                          <div className="text-xs text-brand-600 dark:text-brand-400 mt-1">
                            {t('filter.labels.containsTags', { count: tags.length })}
                          </div>
                        </div>
                      </div>
                      
                      <div className="flex items-center gap-2">
                        {/* 全选按钮 */}
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            const tagsMap = arrayMapString(tags, (tag: models.Tag) => tag.name)
                            const newTagsFilter = [...new Set([...tagsFilter, ...tagsMap])];
                            onTagsFilterChange(newTagsFilter);
                            // onClose();
                          }}
                          className="px-3 py-1 bg-brand-500 text-white rounded-lg hover:bg-brand-600 transition-colors text-sm font-medium"
                        >
                          {t('filter.buttons.selectAll')}
                        </button>
                        
                        {/* 展开/收起按钮 */}
                        <div className={`
                          i-mdi-chevron-down text-brand-500 transition-transform duration-200
                          ${isExpanded ? 'rotate-180' : ''}
                        `} />
                      </div>
                    </div>
                    
                    {/* 标签列表 - 可展开 */}
                    {isExpanded && (
                      <div className="p-3 bg-brand-50 dark:bg-brand-900/20 border-t border-brand-200 dark:border-brand-700">
                        <div className="flex flex-wrap gap-2">
                          {tags.sort((a,b) => a.use_count - b.use_count).map((tag) => (
                            <button
                              key={tag.name}
                              onClick={() => {
                                const newTagsFilter = [...new Set([...tagsFilter, tag.name])];
                                onTagsFilterChange(newTagsFilter);
                                tag.use_count++;
                                UpdateTag(tag).then(() => {})
                                updateTagInTags(tag);
                                // onClose();
                              }}
                              className="px-3 py-1.5 bg-gradient-to-r from-[#e0e000] to-[#c0c000] 
                                       text-brand-800 dark:text-brand-900 text-sm rounded-full
                                       hover:from-[#d0d000] hover:to-[#b0b000]
                                       transform hover:-translate-y-0.5
                                       transition-all duration-200 cursor-pointer
                                       border border-[#d0d000]/30 shadow-sm hover:shadow-md"
                            >
                              <div className="flex items-center">
                                <div className="i-mdi-tag mr-1 text-xs" />
                                {tag.name}{tag.use_count > 0 ? ` (${tag.use_count})` : ''}
                              </div>
                            </button>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          ) : (
            <div className="text-center py-8">
              <div className="i-mdi-folder-outline text-4xl text-brand-300 dark:text-brand-600 mx-auto mb-3" />
              <p className="text-brand-600 dark:text-brand-400">
                {t('filter.messages.noTagGroups')}
              </p>
              <p className="text-brand-500 dark:text-brand-400 text-sm mt-2">
                {t('filter.messages.canCreateTagGroupsFirst')}
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}