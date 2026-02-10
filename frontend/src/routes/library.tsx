import { models } from "../../wailsjs/go/models";
import type { ImportSource } from "../components/modal/GameImportModal";
import { createRoute } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { GetGames } from "../../wailsjs/go/service/GameService";
import { ListTags } from "../../wailsjs/go/service/TagService";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import { FilterBar } from "../components/bar/FilterBar";
import { GameCard } from "../components/card/GameCard";
import { AddGameModal } from "../components/modal/AddGameModal";
import { BatchImportModal } from "../components/modal/BatchImportModal";
import { arrayToMap } from "../components/utils/Utility";
import { GameImportModal } from "../components/modal/GameImportModal";
import { LibrarySkeleton } from "../components/skeleton/LibrarySkeleton";
import { sortOptions, statusOptions } from "../consts/options";
import { Route as rootRoute } from "./__root";
import { TaskPanel } from "../components/panel/TaskPanel";
import { arrayFind, mapToArray } from "../components/utils/Utility";

import { enums, vo } from "../../wailsjs/go/models";
import { BatchUpdateModal } from "../components/modal/BatchUpdateModal";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/library",
  component: LibraryPage,
});



function LibraryPage() {
  const [games, setGames] = useState<models.Game[]>([]);
  const [tagsLoaded, setTagsLoaded] = useState<Map<string, models.Tag[]>>(new Map())
  const [isLoading, setIsLoading] = useState(true);
  const [showSkeleton, setShowSkeleton] = useState(false);
  const [isAddGameModalOpen, setIsAddGameModalOpen] = useState(false);
  const [isBatchImportOpen, setIsBatchImportOpen] = useState(false);
  const [isBatchUpdateOpen, setIsBatchUpdateOpen] = useState(false);
  const [importSource, setImportSource] = useState<ImportSource | null>(null);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState<"name" | "created_at">("created_at");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [tagsFilter, setTags] = useState<string[]>([]);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const gamesForUpdate = useRef(games)
  

  

  
  useEffect(() => {
      const unlistenTaskUpdate = EventsOn("game_updates", (data: any) => {
        
          // 注意：使用正确的语法从data对象获取值
          const task : models.TaskNotice = new models.TaskNotice(data);
          
          // console.log("received taskid:" + task.id + " current taskid:" + taskId + ", item_id:" + task.item_id +  " item_status:" + task.item_status)
          if (task.item_status === enums.TaskStatus.COMPLETED && task.item_id !== "") {
                  const newGame : models.Game = task.item_data as models.Game;
                  console.log("newGame:", newGame)
                  const newGames = [...games]
                  const game = arrayFind(newGames, (it) => it.id === task.item_id)
                  if (game) {
                    const index = newGames.indexOf(game)
                    
                    newGames[index] = newGame
                    setGames(newGames)
                  }
                  
  
              }
          
          
          
      });
  
      return () => {
          if (unlistenTaskUpdate) {
          unlistenTaskUpdate(); // 取消事件监听
          }
      };
      }, [games]);

  
  

  // 延迟显示骨架屏
  useEffect(() => {
    let timer: number;
    if (isLoading) {
      timer = window.setTimeout(() => {
        setShowSkeleton(true);
      }, 300);
    }
    else {
      setShowSkeleton(false);
    }
    return () => clearTimeout(timer);
  }, [isLoading]);

  // 点击外部关闭下拉菜单
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsDropdownOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // 获取URL参数中的标签
  useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search);
    const tagsParam = urlParams.get('tags');
    if (tagsParam) {
      setTags([decodeURIComponent(tagsParam)]);
    }
  }, []);

  const loadGames = async () => {
    try {
      const result = await GetGames();
      setGames(result || []);
      // const tags: string[] = [];
      // result?.forEach((game) => {
      //   const gameTags = game.tags?.split(",") || [];
      //   tags.push(...gameTags);
      // });
      // const uniqueTags : string[] = [...new Set(tags.map(tag => tag.trim()))];
      // setTagsLoaded(uniqueTags);
      const tags = await ListTags();
      const map = arrayToMap(tags, tag => tag.category);
      setTagsLoaded(map);

    }
    catch (error) {
      console.error("Failed to load games:", error);
    }
    finally {
      setIsLoading(false);
    }
  };

  const filteredGames = games
    .filter((game) => {
      // 搜索过滤
      if (searchQuery && !game.name.toLowerCase().includes(searchQuery.toLowerCase())) {
        return false;
      }
      // 状态过滤
      if (statusFilter && game.status !== statusFilter) {
        return false;
      }
      //标签过滤
      if (tagsFilter && tagsFilter.length > 0) {
        const tags = game.tags.split(",");
        if (!tags.some((tag) => tagsFilter.includes(tag))) {
          return false;
        }
      }
      return true;
    })
    .sort((a, b) => {
      let comparison = 0;
      switch (sortBy) {
        case "name":
          comparison = a.name.localeCompare(b.name);
          break;
        case "created_at":
          comparison = String(a.created_at || "").localeCompare(String(b.created_at || ""));
          break;
      }
      return sortOrder === "asc" ? comparison : -comparison;
    });

  useEffect(() => {
    loadGames();
  }, []);

  if (isLoading && games.length === 0) {
    if (!showSkeleton) {
      return null;
    }
    return <LibrarySkeleton />;
  }

  return (
    <div className={`space-y-6 max-w-8xl mx-auto p-8 transition-opacity duration-300 ${isLoading ? "opacity-50 pointer-events-none" : "opacity-100"}`}>
      <div className="flex items-center justify-between">
        <h1 className="text-4xl font-bold text-brand-900 dark:text-white">游戏库</h1>

        <TaskPanel />
      </div>

      <FilterBar
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        searchPlaceholder="搜索游戏..."
        sortBy={sortBy}
        onSortByChange={val => setSortBy(val as "name" | "created_at")}
        sortOptions={sortOptions}
        sortOrder={sortOrder}
        onSortOrderChange={setSortOrder}
        statusFilter={statusFilter}
        onStatusFilterChange={setStatusFilter}
        onTagsFilterChange={setTags}
        tagsLoaded={tagsLoaded}
        tagsFilter={tagsFilter}
        statusOptions={statusOptions}
        storageKey="library"
        actionButton={(
          <div className="relative" ref={dropdownRef}>
            <button
              onClick={() => setIsDropdownOpen(!isDropdownOpen)}
              className="glass-btn-neutral flex items-center rounded-lg bg-neutral-600 px-4 py-2 text-sm font-medium text-white hover:bg-neutral-700 focus:outline-none focus:ring-4 focus:ring-neutral-300 dark:bg-neutral-600 dark:hover:bg-neutral-700 dark:focus:ring-neutral-800"
            >
              <div className="i-mdi-plus mr-2 text-lg" />
              添加游戏
              <div className="i-mdi-chevron-down ml-2 text-lg" />
            </button>

            {/* Dropdown Menu */}
            {isDropdownOpen && (
              <div className="absolute right-0 mt-2 w-56 origin-top-right rounded-lg bg-white shadow-lg ring-1 ring-black/5 dark:bg-brand-700 dark:ring-white/10 z-50">
                <div className="py-1">
                  <button
                    onClick={() => {
                      setIsAddGameModalOpen(true);
                      setIsDropdownOpen(false);
                    }}
                    className="flex w-full items-center px-4 py-3 text-sm text-brand-700 hover:bg-brand-100 dark:text-brand-200 dark:hover:bg-brand-600"
                  >
                    <div className="i-mdi-gamepad-variant mr-3 text-xl text-neutral-500" />
                    <div className="text-left">
                      <div className="font-medium">手动添加</div>
                      <div className="text-xs text-brand-400 dark:text-brand-400">
                        选择可执行文件并搜索元数据
                      </div>
                    </div>
                  </button>
                  <button
                    onClick={() => {
                      setIsBatchImportOpen(true);
                      setIsDropdownOpen(false);
                    }}
                    className="flex w-full items-center px-4 py-3 text-sm text-brand-700 hover:bg-brand-100 dark:text-brand-200 dark:hover:bg-brand-600"
                  >
                    <div className="i-mdi-folder-multiple mr-3 text-xl text-success-500" />
                    <div className="text-left">
                      <div className="font-medium">批量导入</div>
                      <div className="text-xs text-brand-400 dark:text-brand-400">
                        扫描游戏库目录批量添加
                      </div>
                    </div>
                  </button>
                  <div className="border-t border-brand-200 dark:border-brand-600 my-1" />
                  <button
                    onClick={() => {
                      setImportSource("potatovn");
                      setIsDropdownOpen(false);
                    }}
                    className="flex w-full items-center px-4 py-3 text-sm text-brand-700 hover:bg-brand-100 dark:text-brand-200 dark:hover:bg-brand-600"
                  >
                    <div className="i-mdi-database-import mr-3 text-xl text-orange-500" />
                    <div className="text-left">
                      <div className="font-medium">从 PotatoVN 导入</div>
                      <div className="text-xs text-brand-400 dark:text-brand-400">
                        导入 PotatoVN 导出的 ZIP 文件
                      </div>
                    </div>
                  </button>
                  <button
                    onClick={() => {
                      setImportSource("playnite");
                      setIsDropdownOpen(false);
                    }}
                    className="flex w-full items-center px-4 py-3 text-sm text-brand-700 hover:bg-brand-100 dark:text-brand-200 dark:hover:bg-brand-600"
                  >
                    <div className="i-mdi-application-import mr-3 text-xl text-purple-500" />
                    <div className="text-left">
                      <div className="font-medium">从 Playnite 导入</div>
                      <div className="text-xs text-brand-400 dark:text-brand-400">
                        导入 Playnite 导出的 JSON 文件
                      </div>
                    </div>
                  </button>
                  <div className="border-t border-brand-200 dark:border-brand-600 my-1" />
                  <button
                    onClick={() => {
                      gamesForUpdate.current = games;
                      setIsBatchUpdateOpen(true);
                      setIsDropdownOpen(false);
                    }}
                    className="flex w-full items-center px-4 py-3 text-sm text-brand-700 hover:bg-brand-100 dark:text-brand-200 dark:hover:bg-brand-600"
                  >
                    <div className="i-mdi-folder-multiple mr-3 text-xl text-blue-500" />
                    <div className="text-left">
                      <div className="font-medium">更新游戏库</div>
                      <div className="text-xs text-brand-400 dark:text-brand-400">
                        选择数据源更新游戏库
                      </div>
                    </div>
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      />

      {games.length === 0
        ? (
            <div className="flex-1 flex items-center justify-center w-full">
              <div className="flex flex-col items-center justify-center py-20 text-brand-500 dark:text-brand-400">
                <div className="i-mdi-gamepad-variant-outline text-6xl mb-4" />
                <p className="text-xl">暂无游戏</p>
                <p className="text-sm mt-2">添加一些GAL游戏开始吧</p>
                <div className="flex flex-col gap-3 mt-4">
                  <button
                    onClick={() => setImportSource("potatovn")}
                    className="rounded-lg border border-success-600 px-5 py-2.5 text-sm font-medium text-success-600 hover:bg-success-50 focus:outline-none focus:ring-4 focus:ring-success-300 dark:border-success-500 dark:text-success-500 dark:hover:bg-success-900/20"
                  >
                    从 PotatoVN 导入
                  </button>
                  <button
                    onClick={() => setImportSource("playnite")}
                    className="rounded-lg border border-purple-600 px-5 py-2.5 text-sm font-medium text-purple-600 hover:bg-purple-50 focus:outline-none focus:ring-4 focus:ring-purple-300 dark:border-purple-500 dark:text-purple-500 dark:hover:bg-purple-900/20"
                  >
                    从 Playnite 导入
                  </button>
                </div>
              </div>
            </div>
          )
        : filteredGames.length === 0
          ? (
              <div className="flex-1 flex items-center justify-center w-full text-brand-500 dark:text-brand-400">
                <div className="flex flex-col items-center">
                  <div className="i-mdi-magnify text-4xl mb-2" />
                  <p>未找到匹配的游戏</p>
                </div>
              </div>
            )
          : (
              <div className="grid grid-cols-[repeat(auto-fill,minmax(max(8rem,11%),1fr))] gap-3">
                {filteredGames.map(game => (
                  <GameCard key={game.id} game={game} />
                ))}
              </div>
            )}

      <AddGameModal
        isOpen={isAddGameModalOpen}
        onClose={() => setIsAddGameModalOpen(false)}
        onGameAdded={loadGames}
      />

      <GameImportModal
        isOpen={importSource !== null}
        source={importSource || "potatovn"}
        onClose={() => setImportSource(null)}
        onImportComplete={loadGames}
      />

      <BatchImportModal
        isOpen={isBatchImportOpen}
        onClose={() => setIsBatchImportOpen(false)}
        onImportComplete={loadGames}
        onOpenUpdate={(res) => {
          gamesForUpdate.current = res;
          setIsBatchUpdateOpen(true);
          loadGames();
        }}
      />
      <BatchUpdateModal
        isOpen={isBatchUpdateOpen}
        onClose={() => setIsBatchUpdateOpen(false)}
        onUpdateComplete={loadGames}
        games={gamesForUpdate.current}
      />
    </div>
  );
}
