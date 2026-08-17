import { createRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from 'react-i18next';
import { models } from "../../wailsjs/go/models";
import { GetGamesByIdsStr, OpenLocalPath, DeleteFolder } from "../../wailsjs/go/service/GameService";
import { LocalSearchModal } from "../components/modal/LocalSearchModal";
import { useAppStore } from "../store";
import { Route as rootRoute } from "./__root";
import toast from "react-hot-toast";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/task_result/$taskId",
  component: TaskResultPage,
});

const PAGE_SIZE = 50;

function TaskResultPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { taskId } = Route.useParams();
  const { tasks, games, fetchGames, setDroppedPaths, setShowDragDropModal } = useAppStore();
  const [task, setTask] = useState<models.TaskNotice | undefined>(undefined);

  // 从 store 中实时获取任务（store 更新时自动重渲染）
  const taskInStore = tasks.find(t => t.id === taskId);

  const resultGames = task?.result_games || [];
  const title = task?.title || '';

  const [gamesMap, setGamesMap] = useState<Record<string, models.Game>>({});
  const [searchTerm, setSearchTerm] = useState("");
  const [limit, setLimit] = useState(PAGE_SIZE);
  const [searchDirPath, setSearchDirPath] = useState<string | null>(null);
  const [deletedPaths, setDeletedPaths] = useState<Set<string>>(new Set());

  // 节流刷新：store 更新很快（如5000个路径任务）时避免每帧重渲染
  const THROTTLE_MS = 250;
  const pendingTaskRef = useRef<models.TaskNotice | undefined>(undefined);
  const throttleTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const initializedRef = useRef(false);

  useEffect(() => {
    if (!games || games.length === 0) {
      fetchGames();
    }
  }, [games]);

  useEffect(() => {
    // 首次进入：立刻同步写入，避免显示"任务不存在"（后续更新才走节流）
    if (!initializedRef.current) {
      initializedRef.current = true;
      setTask(taskInStore);
      // 首次也清一下 pending/timer 状态
      if (throttleTimerRef.current) {
        clearTimeout(throttleTimerRef.current);
        throttleTimerRef.current = null;
      }
      pendingTaskRef.current = undefined;
      return; // 首次不走 timer，也不要返回 cleanup 清下一次的 timer
    }

    // 任务完成/错误/取消：直接刷新（不再节流），避免用户看不到最终结果
    const isTerminal = taskInStore &&
      (taskInStore.status === "完成" || taskInStore.status === "错误" || taskInStore.status === "取消");

    if (isTerminal) {
      if (throttleTimerRef.current) {
        clearTimeout(throttleTimerRef.current);
        throttleTimerRef.current = null;
      }
      pendingTaskRef.current = undefined;
      setTask(taskInStore);
      return;
    }

    pendingTaskRef.current = taskInStore;

    if (throttleTimerRef.current) return; // 已有 timer 等待，排队

    throttleTimerRef.current = setTimeout(() => {
      throttleTimerRef.current = null;
      if (pendingTaskRef.current !== undefined) {
        setTask(pendingTaskRef.current);
        pendingTaskRef.current = undefined;
      }
    }, THROTTLE_MS);

    // 注意：这里不能返回 cleanup 清 timer，否则 effect 重执行时（store 每次更新）
    // 都会把刚启的 1000ms timer 清掉，导致运行中任务永远等不到 fire → setTask 永远不执行
    // → 一直显示"任务不存在"。组件卸载清理放到独立的空依赖 effect 中。
  }, [taskInStore]);

  // 组件卸载：统一清理 timer
  useEffect(() => {
    return () => {
      if (throttleTimerRef.current) {
        clearTimeout(throttleTimerRef.current);
        throttleTimerRef.current = null;
      }
      if (gamesReqTimerRef.current) {
        clearTimeout(gamesReqTimerRef.current);
        gamesReqTimerRef.current = null;
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 对大量路径的过滤 & id 扁平化做 useMemo，避免每次 render 重跑
  const flattenResultIds = useMemo(
    () => (resultGames?.length ? Array.from(new Set(resultGames.flatMap(rg => rg.game_ids || []))) : []),
    [resultGames],
  );

  // 游戏名查询：防抖 + 基于扁平化 ids，避免 5000+ 路径每次刷新都重查
  const gamesReqTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(() => {
    if (flattenResultIds.length === 0) return;

    // 过滤掉路径（只需要查游戏ID的名字），避免传几千条路径给后端
    const gameIds = flattenResultIds.filter(id => !/[\\\/]/.test(id));
    if (gameIds.length === 0) return;

    if (gamesReqTimerRef.current) clearTimeout(gamesReqTimerRef.current);
    gamesReqTimerRef.current = setTimeout(() => {
      gamesReqTimerRef.current = null;
      GetGamesByIdsStr(gameIds.join(','))
        .then(games => {
          const map: Record<string, models.Game> = {};
          (games || []).forEach(g => {
            if (g.id) map[g.id] = g;
          });
          setGamesMap(map);
        })
        .catch(err => {
          console.error("Failed to load games:", err);
        });
    }, 150);

    return () => {
      if (gamesReqTimerRef.current) {
        clearTimeout(gamesReqTimerRef.current);
        gamesReqTimerRef.current = null;
      }
    };
  }, [flattenResultIds]);

  // 搜索词变化时重置分页
  useEffect(() => {
    setLimit(PAGE_SIZE);
  }, [searchTerm]);

  const handleGameClick = (gameId: string, filteredGameIdsStr: string[]) => {
    navigate({
       to: `/game/${gameId}`,
       search: { filteredGameIdsStr }
       });
  };

  const statusConfig: Record<number, { btn: string }> = {
    200: {
      btn: "bg-success-100 text-success-800 hover:bg-success-200 dark:bg-success-900/30 dark:text-success-400 dark:hover:bg-success-800/50",
    },
    400: {
      btn: "bg-error-100 text-error-800 hover:bg-error-200 dark:bg-error-900/30 dark:text-error-400 dark:hover:bg-error-800/50",
    },
    300: {
      btn: "bg-warning-100 text-warning-800 hover:bg-warning-200 dark:bg-warning-900/30 dark:text-warning-400 dark:hover:bg-warning-800/50",
    },
  };
  const getStatusBtn = (status: number) => statusConfig[status]?.btn
    || "bg-blue-100 text-blue-800 hover:bg-blue-200 dark:bg-blue-900/30 dark:text-blue-400 dark:hover:bg-blue-800/50";

  const isPath = (id: string) => /[\\\/]/.test(id);

  const trimSearch = searchTerm.trim().toLowerCase();

  // 预计算每个 resultGroup 的 filtered 列表：依赖变了才重算，避免每帧遍历 5000+ × 3 组
  const filteredByGroup = useMemo(() => {
    return (resultGames || []).map(rg => {
      const ids = rg.game_ids || [];
      if (!trimSearch) return ids;
      return ids.filter(id => {
        if (isPath(id)) {
          return id.toLowerCase().includes(trimSearch);
        }
        const name = gamesMap[id]?.name || "";
        return name.toLowerCase().includes(trimSearch) || id.toLowerCase().includes(trimSearch);
      });
    });
  }, [resultGames, trimSearch, gamesMap]);

  

  if (!task) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center">
          <div className="i-mdi-alert-circle-outline text-6xl text-brand-400 mb-4" />
          <p className="text-xl text-brand-500 dark:text-brand-400">{t('task.result.notFound')}</p>
          <button
            onClick={() => navigate({ to: '/task' })}
            className="mt-4 px-4 py-2 rounded-md bg-blue-600 hover:bg-blue-700 text-white transition-colors"
          >
            {t('task.result.backToTask')}
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full max-w-6xl mx-auto p-6">
      {/* 头部 */}
      <div className="flex items-center justify-between mb-4 shrink-0">
        <div className="flex items-center gap-3">
          <button
            onClick={() => navigate({ to: '/task' })}
            className="p-2 rounded-md text-brand-500 hover:bg-brand-100 dark:hover:bg-brand-700 transition-colors"
            title={t('task.result.backToTask')}
          >
            <div className="i-mdi-arrow-left text-xl" />
          </button>
          <h3 className="text-xl font-bold text-brand-900 dark:text-white">{t('task.result.title')}: {title}</h3>
        </div>
        <div className="flex items-center gap-2 text-sm text-brand-500 dark:text-brand-400">
          <span className={`px-2 py-0.5 rounded-full ${
            task.status === "完成" ? "bg-success-100 text-success-800 dark:bg-success-900/30 dark:text-success-400" :
            task.status === "错误" ? "bg-error-100 text-error-800 dark:bg-error-900/30 dark:text-error-400" :
            "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400"
          }`}>
            {task.status === "完成" ? t('task.status.completed') :
             task.status === "错误" ? t('task.status.error') :
             task.status === "取消" ? t('task.status.canceled') :
             task.status === "暂停" ? t('task.status.paused') :
             task.status === "已开始" ? t('task.status.started') :
             task.status}
          </span>
          <span className="capitalize">{task.type}</span>
          <span>{task.completed}/{task.total}</span>
        </div>
      </div>

      {/* 搜索框 */}
      <div className="mb-4 shrink-0">
        <div className="relative">
          <div className="i-mdi-magnify absolute left-3 top-1/2 -translate-y-1/2 text-brand-400" />
          <input
            type="text"
            value={searchTerm}
            onChange={e => setSearchTerm(e.target.value)}
            placeholder={t('task.result.searchPlaceholder')}
            className="w-full pl-9 pr-3 py-2 text-sm rounded-md border border-brand-200 dark:border-brand-700 bg-white dark:bg-brand-900 text-brand-900 dark:text-white placeholder-brand-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
      </div>

      {/* 内容区 */}
      <div className="flex-1 overflow-y-auto">
        {(!resultGames || resultGames.length === 0) ? (
          <p className="text-brand-500 dark:text-brand-400 text-sm">{t('task.result.empty')}</p>
        ) : (
          <div className="space-y-4">
            {resultGames.map((rg, idx) => {
              const btnClass = getStatusBtn(rg.status);
              const filtered = filteredByGroup[idx] || [];
              const visible = filtered.slice(0, limit);
              const hasMore = filtered.length > visible.length;

              return (
                <div key={idx} className="rounded-lg p-4 border border-brand-200 dark:border-brand-700 bg-white dark:bg-brand-800">
                  <div className="flex items-center justify-between mb-2">
                    <div>
                      {rg.description && (
                        <p className="text-sm text-brand-600 dark:text-brand-400">{rg.description}</p>
                      )}
                    </div>
                    <div className="flex items-center gap-2 shrink-0 ml-2">
                      {rg.status !== 200 && filtered.some(id => isPath(id)) && (
                        <button
                          type="button"
                          onClick={() => {
                            const paths = filtered.filter(id => isPath(id));
                            if (paths.length > 0) {
                              setDroppedPaths(paths);
                              setShowDragDropModal(true);
                            }
                          }}
                          className="flex items-center gap-1 px-2 py-1 text-xs rounded-md bg-primary-500 hover:bg-primary-600 text-white dark:bg-primary-600 dark:hover:bg-primary-500 transition-colors"
                          title={t('task.result.importSelected')}
                        >
                          <div className="i-mdi-import text-sm" />
                          <span>{t('task.result.importSelected')}</span>
                        </button>
                      )}
                      <span className="text-xs text-brand-400 dark:text-brand-500">
                        {filtered.length}{trimSearch ? ` / ${rg.game_ids?.length || 0}` : ""}
                      </span>
                    </div>
                  </div>

                  {visible.length > 0 ? (
                    <>
                      <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 mt-2">
                        {visible.map(gameId => {
                          const isDir = isPath(gameId);
                          const game = !isDir ? gamesMap[gameId] : undefined;
                          const isDeleted = isDir && deletedPaths.has(gameId);
                          return isDir ? (
                            <div
                              key={gameId}
                              className={`flex items-center gap-1 px-2 py-1.5 text-sm rounded-md ${isDeleted ? 'opacity-40 line-through' : ''} ${btnClass}`}
                              title={gameId}
                            >
                              <div className="flex items-center gap-0.5 shrink-0">
                                <button
                                  type="button"
                                  onClick={async () => {
                                    try { await OpenLocalPath(gameId); }
                                    catch (err) { console.error("OpenLocalPath failed:", err); }
                                  }}
                                  className="p-1 rounded hover:bg-black/10 dark:hover:bg-white/10 transition-colors"
                                  title={t('task.result.openFolder')}
                                >
                                  <div className="i-mdi-folder-open-outline text-base" />
                                </button>
                                <button
                                  type="button"
                                  onClick={() => setSearchDirPath(gameId)}
                                  className="p-1 rounded hover:bg-black/10 dark:hover:bg-white/10 transition-colors"
                                  title={t('task.result.searchLocal')}
                                >
                                  <div className="i-mdi-magnify text-base" />
                                </button>
                                <button
                                  type="button"
                                  onClick={async () => {
                                    try {
                                      await DeleteFolder(gameId);
                                      setDeletedPaths(prev => new Set(prev).add(gameId));
                                      toast.success(t('task.result.deleteSuccess'));
                                    } catch (err) {
                                      console.error("DeleteFolder failed:", err);
                                      toast.error(t('task.result.deleteFailed'));
                                    }
                                  }}
                                  className="p-1 rounded hover:bg-red-500/20 transition-colors"
                                  title={t('task.result.deleteFolder')}
                                >
                                  <div className="i-mdi-delete-outline text-base" />
                                </button>
                              </div>
                              {
                                <FoundIcon path={gameId} />
                              }

                              {/* <div className="i-mdi-folder-outline text-base shrink-0" /> */}
                              <span className="truncate">{gameId}</span>
                            </div>
                          ) : (
                            <button
                              key={gameId}
                              onClick={() => handleGameClick(gameId, visible)}
                              className={`flex items-center gap-1 px-3 py-1.5 text-sm rounded-md transition-colors ${btnClass}`}
                              title={game?.name || gameId}
                            >
                              <div className="i-mdi-gamepad-variant-outline text-base shrink-0" />
                              <span className="truncate">{game?.name || gameId}</span>
                            </button>
                          );
                        })}
                      </div>
                      {hasMore && (
                        <button
                          onClick={() => setLimit(l => l + PAGE_SIZE)}
                          className="mt-2 w-full py-1.5 text-xs text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-md transition-colors"
                        >
                          {t('task.result.loadMore', { shown: visible.length, total: filtered.length })}
                        </button>
                      )}
                    </>
                  ) : (
                    <p className="text-xs text-brand-400 dark:text-brand-500 mt-1">
                      {trimSearch ? t('task.result.noMatch') : t('task.result.empty')}
                    </p>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* 本地搜索弹窗 */}
      {searchDirPath && (
        <LocalSearchModal
          itemName={searchDirPath.split(/[\\/]/).pop() || searchDirPath}
          onClose={() => setSearchDirPath(null)}
        />
      )}
    </div>
  );
}

interface FoundIconProps {
  path: string;
}

function FoundIcon({ path }: FoundIconProps) {
  const { games } = useAppStore();

  const foundGamesLength = (p: string) => {
    const query = p.split(/[\\/]/).pop() || "";
    if (query.length === 0) return null;

    const gs = (games || []).filter((game: models.Game) => {
      const name = game?.name?.toLowerCase() || "";
      const searchName = game?.search_name?.toLowerCase() || "";
      const lowerQuery = query?.toLowerCase() || "";
      return name.includes(lowerQuery) || searchName.includes(lowerQuery);
    });
    if (gs.length === 0) return null;

    return gs[0];
  }

  const resultFoundGamesLength = useMemo(() => {
    return foundGamesLength(path);
  }, [path, games]);

  
  if (resultFoundGamesLength) {
    return (
      <div className="i-mdi-folder-outline text-base shrink-0" title={resultFoundGamesLength.path} />
    );
  } else {
    return null;
  }
}