import React, { useEffect, useState } from "react";
import { BetterSelect } from "../ui/BetterSelect";
import { models } from "../../../wailsjs/go/models";
import { arrayFind, mapToArray } from "../utils/Utility";

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
}: FilterBarProps) {
  const [initialized, setInitialized] = useState(false);
  const [expanded, setExpanded] = useState(true);

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
    // 在组件顶部添加状态
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);

  // 计算可用标签（tagsLoaded 中除去 tagsFilter 的标签）
  const availableTags = getMapFromArrayMap(true, tagsFilter || [], tagsLoaded || new Map());
  const selectedTags = getMapFromArrayMap(false, tagsFilter || [], tagsLoaded || new Map());
  // const [availableTags, setAvailableTags] = useState<string[]>([]);
  // setAvailableTags(tagsLoaded?.filter((tag) => !tagsFilter!.includes(tag)) || []);

  // const handleTagsFilterChange = (selectedTags: string[], tag: string) => { 
  //   onTagsFilterChange?.(selectedTags)
  // };
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

              <div className="flex items-center gap-2">
                <div className="font-semibold text-brand-900 dark:text-white">标签</div>
                <div className="relative">
                  <button
                    onClick={() => setIsDropdownOpen(!isDropdownOpen)}
                    className="p-1 rounded-full hover:bg-brand-100 dark:hover:bg-brand-800 transition-colors"
                    aria-label="添加标签"
                  >
                    <div className="i-mdi-plus text-base" />
                  </button>
                  

                  {isDropdownOpen && availableTags && onTagsFilterChange && tagsFilter && (
                    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
                      <div className="bg-white dark:bg-brand-900 rounded-lg shadow-xl w-[1200px] max-w-[95vw] max-h-[90vh] overflow-y-auto">
                        <div className="p-4 border-b border-brand-200 dark:border-brand-700 flex justify-between items-center">
                          <h3 className="text-lg font-semibold text-brand-900 dark:text-white">选择标签</h3>
                          <button 
                            onClick={() => setIsDropdownOpen(false)}
                            className="text-brand-500 hover:text-brand-700 dark:text-brand-400 dark:hover:text-brand-200"
                          >
                            <div className="i-mdi-close text-xl" />
                          </button>
                        </div>


                        <div className="p-4">
                          <div className="flex flex-wrap gap-2">

                              <div className="space-y-4">
                                {Array.from(availableTags.entries()).map(([category, tags]) => (
                                  <div key={category} className="border border-brand-200 dark:border-brand-700 rounded-lg p-4 bg-white dark:bg-brand-800/30">
                                    <h4 className="font-semibold text-brand-800 dark:text-brand-200 mb-3 flex items-center">
                                      <div className="i-mdi-folder-outline mr-2 text-brand-500" />
                                      {category}
                                      <span className="ml-2 text-xs bg-brand-100 dark:bg-brand-700 text-brand-600 dark:text-brand-300 px-2 py-1 rounded-full">
                                        {tags.length}
                                      </span>
                                    </h4>
                                    <div className="flex flex-wrap gap-2">
                                      {tags.map((tag) => (
                                        <button
                                          // key={tag}
                                          onClick={() => onTagsFilterChange([...tagsFilter, tag.name])}
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
                                {availableTags.size === 0 && (
                                  <div className="text-center py-8">
                                    <div className="i-mdi-tag-off text-4xl text-brand-300 dark:text-brand-600 mx-auto mb-3" />
                                    <p className="text-brand-600 dark:text-brand-400">
                                      没有可用的标签分类
                                    </p>
                                  </div>
                                )}
                              </div>

                                {availableTags.size === 0 && (
                                  <p className="text-brand-600 dark:text-brand-400 text-sm">没有可用标签</p>
                                )}
                          </div>
                        </div>



                      </div>
                    </div>
                  )}

                </div>
              </div>
              
              <div className="mt-3 mb-3"></div>
              
              {
                
              tagsFilter && tagsFilter.length > 0 && onTagsFilterChange ? (
                <div className="flex flex-wrap gap-2">
                  {/* {tagsFilter.map((tag, index) => (
                  <button
                      key={index}
                      className="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-[#e0e000] text-brand-800 dark:bg-[#e0e000] dark:text-brand-200 hover:bg-[#d0d000] dark:hover:bg-[#d0d000] transition-colors cursor-pointer relative group"
                      onClick={() => {
                      // 在这里添加点击标签时的处理逻辑
                      // 从 tagsFilter 中移除当前标签
                        const newTagsFilter = tagsFilter.filter(t => t !== tag);
                        onTagsFilterChange(newTagsFilter);
                      }}
                  >
                      <span className="relative z-10">{tag.trim()}</span>
                      <span className="absolute inset-0 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity z-15">
                        <div className="absolute inset-0 bg-white bg-opacity-50 dark:bg-black dark:bg-opacity-40 rounded-full z-1"></div>
                        <div className="relative z-20 i-mdi-close text-xs" />
                      </span>
                  </button>
                  ))} */}
                  {Array.from(selectedTags.entries()).map(([category, tags]) => (
                    <div key={category} className="rounded-lg p-4 bg-transparent dark:bg-transparent border-0 shadow-none">
                      <h4 className="font-semibold text-brand-800 dark:text-brand-200 mb-3 flex items-center">
                        <div className="i-mdi-folder-outline mr-2 text-brand-500" />
                        {category}
                        <span className="ml-2 text-xs bg-brand-100 dark:bg-brand-700 text-brand-600 dark:text-brand-300 px-2 py-1 rounded-full">
                          {tags.length}
                        </span>
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

          <div className="flex flex-col gap-2">
            <div className="flex justify-center">
              <button 
                onClick={() => setExpanded(!expanded)}
                className="focus:outline-none"
              >
                <div className={`i-mdi-chevron-down text-lg text-brand-500 dark:text-brand-400 transition-transform duration-200 ${!expanded ? 'rotate-180' : ''}`} />
              </button>
            </div>
          </div>

        </div>
        
      ) : (
        <div className="flex flex-col gap-2">
          <div className="flex justify-center">
            <button 
              onClick={() => setExpanded(!expanded)}
              className="focus:outline-none"
            >
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
