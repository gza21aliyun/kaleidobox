import React, { useEffect, useState, useRef } from "react";
import { BetterSelect } from "../ui/BetterSelect";
import { models } from "../../../wailsjs/go/models";
import { arrayContains, mapToArray, tagMapForEach } from "../utils/Utility";
import { ListTags, GetTagListByGroup, UpdateTagsGroup, ListGroups } from "../../../wailsjs/go/service/TagService";
import { FilterChooseTagModal, FilterChooseGroupModal } from "../modal/FilterChooseTagModal";

interface SortOption {
  label: string;
  value: string;
}

interface FilterOption {
  label: string;
  value: string;
}

interface FilterBarProps {
  searchQuery: string;
  onSearchChange: (value: string) => void;
  searchPlaceholder?: string;
  sortBy: string;
  onSortByChange: (value: string) => void;
  sortOptions: SortOption[];
  sortOrder: "asc" | "desc";
  onSortOrderChange: (order: "asc" | "desc") => void;
  // 状态筛选
  statusFilter?: string;
  onStatusFilterChange?: (value: string) => void;
  onTagsFilterChange?: (value: string[]) => void;
  tagsLoaded?: Map<string, models.Tag[]>;
  tagsFilter?: string[];
  statusOptions?: FilterOption[];
  actionButton?: React.ReactNode;
  extraButtons?: React.ReactNode;
  // 持久化存储键，传入后会自动保存和恢复排序设置
  storageKey?: string;
  filterExpanded?: boolean;
  releaseStartDate?: string;
  onReleaseStartDateChange?: (date: string) => void;
  releaseEndDate?: string;
  onReleaseEndDateChange?: (date: string) => void;
}

export function FilterBar({
  searchQuery,
  onSearchChange,
  searchPlaceholder = "搜索...",
  sortBy,
  onSortByChange,
  sortOptions,
  sortOrder,
  onSortOrderChange,
  statusFilter,
  onStatusFilterChange,
  onTagsFilterChange,
  tagsLoaded,
  tagsFilter,
  statusOptions,
  actionButton,
  extraButtons,
  storageKey,
  filterExpanded,
  releaseStartDate,
  onReleaseStartDateChange,
  releaseEndDate,
  onReleaseEndDateChange,
}: FilterBarProps) {
  const [initialized, setInitialized] = useState(false);
  const [expanded, setExpanded] = useState(filterExpanded == undefined ? false : filterExpanded);
  const [isGroupDropdownOpen, setIsGroupDropdownOpen] = useState(false);
    // 在组件顶部添加状态
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);


  useEffect(() => {
    if (filterExpanded != undefined) {
      setExpanded(filterExpanded)
    }
    
  }, [filterExpanded])

  // 初始化时从 localStorage 恢复排序设置
  useEffect(() => {


    console.log("标签2：", tagsLoaded);

    if (storageKey && !initialized) {
      const savedSortBy = localStorage.getItem(`${storageKey}_sortBy`);
      const savedSortOrder = localStorage.getItem(`${storageKey}_sortOrder`);

      // 验证保存的 sortBy 是否在 sortOptions 中
      if (savedSortBy && sortOptions.some(opt => opt.value === savedSortBy)) {
        onSortByChange(savedSortBy);
      }

      if (savedSortOrder === "asc" || savedSortOrder === "desc") {
        onSortOrderChange(savedSortOrder);
      }

      setInitialized(true);
    }
  }, [storageKey, sortOptions, initialized]);

  // 处理排序方式变更
  const handleSortByChange = (value: string) => {
    onSortByChange(value);
    if (storageKey) {
      localStorage.setItem(`${storageKey}_sortBy`, value);
    }
  };

  // 处理排序顺序变更
  const handleSortOrderChange = (order: "asc" | "desc") => {
    onSortOrderChange(order);
    if (storageKey) {
      localStorage.setItem(`${storageKey}_sortOrder`, order);
    }
  };

  
  // const tagGroups = useRef<string[]>([]);
  const getTagGroups = () => {
    tagsLoaded;
    const groupMap : Map<string, string[]> = new Map()
    const tagArray : models.Tag[]= mapToArray(tagsLoaded || new Map())
    for (const tag of tagArray) {
      if (!tag.group || tag.group === "") {
        continue;
      }
      if (groupMap.has(tag.group)) {
        groupMap.set(tag.group, [...(groupMap.get(tag.group) || []), tag.name])
      } else {
        groupMap.set(tag.group, [tag.name])
      }
    }
    return groupMap;
  }

  const handleClearFilters = () => {
    onSearchChange("");
    if (onStatusFilterChange) onStatusFilterChange("");
    if (onTagsFilterChange) onTagsFilterChange([]);
    if (onReleaseStartDateChange) onReleaseStartDateChange("");
    if (onReleaseEndDateChange) onReleaseEndDateChange("");
  };


  // 计算可用标签（tagsLoaded 中除去 tagsFilter 的标签）
  const availableTags = getMapFromArrayMap(true, tagsFilter || [], tagsLoaded || new Map());
  const selectedTags = getMapFromArrayMap(false, tagsFilter || [], tagsLoaded || new Map());
  return (
    <div> 
      <div className="flex flex-wrap items-center justify-between gap-4 my-4">
        <div className="relative flex-1 max-w-md">
          <div className="absolute inset-y-0 left-0 flex items-center pl-3 pointer-events-none">
            <div className="i-mdi-magnify text-brand-500" />
          </div>
          <input
            type="text"
            className="glass-input block w-auto p-2 pl-10 text-sm text-brand-900 dark:text-white
                      bg-white dark:bg-brand-900
                      border border-brand-300 dark:border-brand-700
                      rounded-lg
                      placeholder:text-brand-400 dark:placeholder:text-brand-400
                      focus:ring-neutral-500 focus:border-neutral-500
                      dark:focus:ring-neutral-500 dark:focus:border-neutral-500"
            placeholder={searchPlaceholder}
            value={searchQuery}
            onChange={e => onSearchChange(e.target.value)}
          />
        </div>

        <div className="flex items-center gap-2">

          {true && (
            <button
              type="button"
              onClick={handleClearFilters}
              className="glass-panel p-2
                        text-red-500 dark:text-red-400
                        hover:text-red-700 dark:hover:text-red-300
                        bg-white dark:bg-brand-800
                        border border-red-200 dark:border-red-700
                        rounded-lg
                        hover:bg-red-50 dark:hover:bg-red-900/20
                        transition-colors"
              title="清除所有过滤器"
            >
              <div className="i-mdi-filter-remove text-xl" />
            </button>
          )}



          {/* 状态筛选 */}
          {statusOptions && onStatusFilterChange && (
            <BetterSelect
              value={statusFilter || ""}
              onChange={onStatusFilterChange}
              options={statusOptions}
              className="min-w-[120px]"
            />
          )}

          <BetterSelect
            value={sortBy}
            onChange={handleSortByChange}
            options={sortOptions}
            className="min-w-[120px]"
          />

          <button
            type="button"
            onClick={() => handleSortOrderChange(sortOrder === "asc" ? "desc" : "asc")}
            className="glass-panel p-2
                      text-brand-500 dark:text-brand-400
                      hover:text-brand-900 dark:hover:text-white
                      bg-white dark:bg-brand-800
                      border border-brand-200 dark:border-brand-700
                      rounded-lg
                      hover:bg-brand-100 dark:hover:bg-brand-700"
            title={sortOrder === "asc" ? "升序" : "降序"}
          >
            <div className={sortOrder === "asc" ? "i-mdi-sort-ascending text-xl" : "i-mdi-sort-descending text-xl"} />
          </button>

          {extraButtons}
          {actionButton}
        </div>
      </div>

      {!expanded ? (
        
        <div id="expanded-filter-bar">
          <div className="mt-4">

              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="font-semibold text-brand-900 dark:text-white">标签</div>
                  <div className="relative flex items-center gap-2">
                    <button
                      onClick={() => setIsDropdownOpen(!isDropdownOpen)}
                      className="flex items-center gap-1 px-3 py-1.5 rounded-lg hover:bg-brand-100 dark:hover:bg-brand-800 transition-colors"
                      aria-label="添加标签"
                    >
                      <div className="i-mdi-plus text-base" />
                      <span className="text-sm text-brand-700 dark:text-brand-300 font-medium">
                        选择标签
                      </span>
                    </button>
                    
                    { tagsFilter && onTagsFilterChange && (
                      <FilterChooseTagModal
                        isOpen={isDropdownOpen}
                        onClose={() => setIsDropdownOpen(false)}
                        availableTags={availableTags}
                        tagsFilter={tagsFilter!}
                        onTagsFilterChange={onTagsFilterChange!}
                      />
                    )}

                    

                    <button
                      onClick={() => setIsGroupDropdownOpen(!isGroupDropdownOpen)}
                      className="flex items-center gap-1 px-3 py-1.5 rounded-lg hover:bg-brand-100 dark:hover:bg-brand-800 transition-colors"
                      aria-label="选择标签分组"
                    >
                      <div className="i-mdi-folder-multiple text-base" />
                      <span className="text-sm text-brand-700 dark:text-brand-300 font-medium">
                        选择分组
                      </span>
                    </button>

                    { tagsFilter && onTagsFilterChange && (
                      <FilterChooseGroupModal
                        isOpen={isGroupDropdownOpen}
                        onClose={() => setIsGroupDropdownOpen(false)}
                        onTagsFilterChange={onTagsFilterChange}
                        availableTags={availableTags || new Map()}
                        tagsFilter={tagsFilter}
                      />
                    )}


                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-sm text-brand-700 dark:text-brand-300 whitespace-nowrap">
                    发售日期:
                  </span>
                  
                  {/* 开始日期 */}
                  <div className="relative">
                    <input
                      type="date"
                      value={releaseStartDate || ""}
                      onChange={(e) => onReleaseStartDateChange?.(e.target.value)}
                      className="glass-input px-3 py-2 text-sm text-brand-900 dark:text-white
                                bg-white dark:bg-brand-900
                                border border-brand-300 dark:border-brand-700
                                rounded-lg
                                focus:ring-neutral-500 focus:border-neutral-500
                                dark:focus:ring-neutral-500 dark:focus:border-neutral-500
                                w-32"
                    />
                  </div>
                  
                  <span className="text-brand-500">-</span>
                  
                  {/* 结束日期 */}
                  <div className="relative">
                    <input
                      type="date"
                      value={releaseEndDate || ""}
                      onChange={(e) => onReleaseEndDateChange?.(e.target.value)}
                      className="glass-input px-3 py-2 text-sm text-brand-900 dark:text-white
                                bg-white dark:bg-brand-900
                                border border-brand-300 dark:border-brand-700
                                rounded-lg
                                focus:ring-neutral-500 focus:border-neutral-500
                                dark:focus:ring-neutral-500 dark:focus:border-neutral-500
                                w-32"
                    />
                  </div>
                </div>
              </div>
              
              
              {
                
              tagsFilter && tagsFilter.length > 0 && onTagsFilterChange ? (
                <div className="flex flex-wrap gap-2">
                  
                  {tagMapForEach(selectedTags, (category, tags) => (
                    <div 
                      key={category} 
                      className="rounded-lg p-4 bg-transparent dark:bg-transparent border-0 shadow-none">
                      <h4 className="font-semibold text-brand-800 dark:text-brand-200 mb-3 flex items-center">
                        <div className="i-mdi-folder-outline mr-2 text-brand-500" />
                        {category}
                        <span className="ml-2 text-xs bg-brand-100 dark:bg-brand-700 text-brand-600 dark:text-brand-300 px-2 py-1 rounded-full">
                          {tags.length}
                        </span>
                        <button
                          onClick={(e) => {
                            // e.stopPropagation();
                            const newTagsFilter = tagsFilter.filter(t => !arrayContains(tags, (tag) => {return tag.name == t;}));
                            onTagsFilterChange(newTagsFilter);
                          }}
                          className="p-2 rounded-full hover:bg-red-100 dark:hover:bg-red-900/20 transition-colors"
                          title="删除分类"
                        >
                          <div className="i-mdi-delete text-red-500 hover:text-red-700 dark:hover:text-red-400" />
                        </button>
                      </h4>
                      <div className="flex flex-wrap gap-2">
                        {tags.map((tag) => (
                          <button
                            key={tag.name}
                            onClick={() => {
                              const newTagsFilter = tagsFilter.filter(t => t !== tag.name);
                              onTagsFilterChange(newTagsFilter);
                            }}
                            className="inline-flex items-center px-3 py-1.5 rounded-full text-xs font-medium
                                    bg-gradient-to-r from-blue-500 to-blue-600 
                                    text-white dark:text-white
                                    hover:from-blue-600 hover:to-blue-700
                                    transform hover:-translate-y-0.5
                                    transition-all duration-200 cursor-pointer
                                    border border-blue-400/30
                                    shadow-sm hover:shadow-md"
                          >
                            <div className="i-mdi-tag mr-1 text-xs" />
                            {tag.name}
                          </button>
                        ))}
                        {tags.length === 0 && (
                          <p className="text-brand-500 dark:text-brand-400 text-sm italic">
                            暂无标签
                          </p>
                        )}
                      </div>
                    </div>
                  ))}
                  </div>
              ) : (
              // <p className="text-brand-600 dark:text-brand-400 text-sm">-</p>
              <div></div>
              )}
              
          </div>

          <div className="flex flex-col gap-2 mt-4">
            <div className="flex justify-center">
              <button 
                onClick={() => setExpanded(!expanded)}
                className="flex items-center justify-center gap-2 w-full max-w-xs px-6 py-3 rounded-lg bg-brand-100 dark:bg-brand-800 hover:bg-brand-200 dark:hover:bg-brand-700 focus:outline-none transition-colors"
              >
                <span className="text-brand-700 dark:text-brand-300 font-medium">
                  过滤器
                </span>
                <div className={`i-mdi-chevron-down text-lg text-brand-500 dark:text-brand-400 transition-transform duration-200 ${!expanded ? 'rotate-180' : ''}`} />
              </button>
            </div>
          </div>
        </div>
        
      ) : (
        <div className="flex flex-col gap-2 mt-4">
          <div className="flex justify-center">
            <button 
              onClick={() => setExpanded(!expanded)}
              className="flex items-center justify-center gap-2 w-full max-w-xs px-6 py-3 rounded-lg bg-brand-100 dark:bg-brand-800 hover:bg-brand-200 dark:hover:bg-brand-700 focus:outline-none transition-colors"
            >
              <span className="text-brand-700 dark:text-brand-300 font-medium">
                过滤器
              </span>
              <div className={`i-mdi-chevron-down text-lg text-brand-500 dark:text-brand-400 transition-transform duration-200 ${!expanded ? 'rotate-180' : ''}`} />
            </button>
          </div>
        </div>
      )}
      

      
    </div>
    
  );
}


export function getMapFromArrayMap(isRemain: boolean, array: string[], map: Map<string, models.Tag[]>): Map<string, models.Tag[]> { 
    const newMap = new Map<string, models.Tag[]>();

    for (const [key, tags] of map) {
      const newTags: models.Tag[] = [];
      for (const tag of tags) {
        if ((isRemain && !array.includes(tag.name)) || (!isRemain && array.includes(tag.name))) {
          newTags.push(tag);
        }
      }
      if (newTags.length > 0) {
        newMap.set(key, newTags);
      }
    }
    return newMap;
}
