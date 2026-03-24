import { models } from "../../wailsjs/go/models";
import { createRoute } from "@tanstack/react-router";
import { PauseTask, CancelTask, ResumeTask } from "../../wailsjs/go/service/TaskService";
import { SearchVideoExePaths } from "../../wailsjs/go/service/ImportService";
import { useAppStore } from "../store";
import { Route as rootRoute } from "./__root";
import { useTranslation } from 'react-i18next';

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/task",
  component: TaskPage,
});

function TaskPage() {
  const { t } = useTranslation();
  const { tasks, setTasks } = useAppStore();

  const handlePauseTask = async (taskId: string) => {
    try {
      await PauseTask(taskId);
    } catch (error) {
      console.error("Failed to pause task:", error);
    }
  };

  const handleResumeTask = async (taskId: string) => {
    try {
      await ResumeTask(taskId);
    } catch (error) {
      console.error("Failed to resume task:", error);
    }
  };

  const handleCancelTask = async (taskId: string) => {
    try {
      await CancelTask(taskId);
    } catch (error) {
      console.error("Failed to cancel task:", error);
    }
  };

  const handleDeleteTask = (taskId: string) => {
    setTasks(tasks.filter((t) => t.id !== taskId));
  };

  const handleSearchVideoPaths = async () => {
    try {
      const { games } = useAppStore.getState();
      await SearchVideoExePaths(games);
    } catch (error) {
      console.error("Failed to search video paths:", error);
    }
  };

  return (
    <div className="space-y-6 max-w-8xl mx-auto p-8">
      <div className="flex items-center justify-between">
        <h1 className="text-4xl font-bold text-brand-900 dark:text-white">{t('task.title')}</h1>
        <button
          onClick={handleSearchVideoPaths}
          className="px-4 py-2 rounded-md bg-blue-600 hover:bg-blue-700 text-white transition-colors"
          title={t('task.actions.searchVideo')}
        >
          为游戏搜索视频
        </button>
      </div>

      {tasks.length > 0 ? (
        <div className="task-list flex flex-col gap-4">
          {tasks.map((task) => (
            <div key={task.id} className="bg-white dark:bg-brand-700 rounded-lg border border-brand-200 dark:border-brand-600 p-4 shadow-sm hover:shadow-md transition-shadow">
              <div className="flex justify-between items-start">
                <div className="flex-1 min-w-0">
                  <h4 className="font-medium text-brand-900 dark:text-white truncate">{task.name}</h4>
                  <div className="flex items-center gap-2 mt-1 text-xs text-brand-500 dark:text-brand-400">
                    <span className={`px-2 py-0.5 rounded-full ${
                      task.status === "完成" ? "bg-success-100 text-success-800 dark:bg-success-900/30 dark:text-success-400" :
                      task.status === "错误" ? "bg-error-100 text-error-800 dark:bg-error-900/30 dark:text-error-400" :
                      task.status === "取消" ? "bg-warning-100 text-warning-800 dark:bg-warning-900/30 dark:text-warning-400" :
                      task.status === "暂停" ? "bg-info-100 text-info-800 dark:bg-info-900/30 dark:text-info-400" :
                      "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400"
                    }`}>
                      {task.status}
                    </span>
                    <span className="capitalize">{task.type}</span>
                  </div>
                  
                  {task.working_on && (
                    <p className="text-sm text-brand-600 dark:text-brand-300 mt-2 truncate">{task.working_on}</p>
                  )}
                  
                  {task.warning && (
                    <p className="text-sm text-red-600 dark:text-red-400 mt-1">{task.warning}</p>
                  )}
                  
                  <div className="mt-3">
                    <div className="flex justify-between text-xs text-brand-500 dark:text-brand-400 mb-1">
                      <span>{task.completed}/{task.total}</span>
                      <span>{task.total > 0 ? Math.round((task.completed / task.total) * 100) : 0}%</span>
                    </div>
                    <div className="w-full bg-brand-200 dark:bg-brand-600 rounded-full h-2">
                      <div
                        className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                        style={{ width: `${task.total > 0 ? (task.completed / task.total) * 100 : 0}%` }}
                      ></div>
                    </div>
                  </div>
                  
                  <p className="text-xs text-brand-400 dark:text-brand-500 mt-2 truncate">{task.description}</p>
                </div>
                
                <div className="flex flex-col gap-1 ml-4">
                  {(task.status === "已开始") && (
                    <button
                      onClick={() => handlePauseTask(task.id)}
                      className="p-2 rounded-md bg-yellow-100 hover:bg-yellow-200 text-yellow-800 dark:bg-yellow-900/30 dark:hover:bg-yellow-800/50 dark:text-yellow-400 transition-colors"
                      title={t('task.actions.pause')}
                    >
                      <div className="i-mdi-pause text-lg"></div>
                    </button>
                  )}
                  
                  {(task.status === "暂停") && (
                    <button
                      onClick={() => handleResumeTask(task.id)}
                      className="p-2 rounded-md bg-green-100 hover:bg-green-200 text-green-800 dark:bg-green-900/30 dark:hover:bg-green-800/50 dark:text-green-400 transition-colors"
                      title={t('task.actions.resume')}
                    >
                      <div className="i-mdi-play text-lg"></div>
                    </button>
                  )}
                  
                  {(task.status !== "完成" && task.status !== "错误" && task.status !== "取消") && (
                    <button
                      onClick={() => handleCancelTask(task.id)}
                      className="p-2 rounded-md bg-red-100 hover:bg-red-200 text-red-800 dark:bg-red-900/30 dark:hover:bg-red-800/50 dark:text-red-400 transition-colors"
                      title={t('task.actions.cancel')}
                    >
                      <div className="i-mdi-cancel text-lg"></div>
                    </button>
                  )}
                  {(task.status === "完成") && (
                    <button
                      onClick={() => handleDeleteTask(task.id)}
                      className="p-2 rounded-md bg-gray-100 hover:bg-gray-200 text-gray-800 dark:bg-gray-700 dark:hover:bg-gray-600 dark:text-gray-300 transition-colors"
                      title={t('task.actions.delete')}
                    >
                      <div className="i-mdi-delete text-lg"></div>
                    </button>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="flex-1 flex items-center justify-center w-full">
          <div className="flex flex-col items-center justify-center py-20 text-brand-500 dark:text-brand-400">
            <div className="i-mdi-clipboard-check-outline text-6xl mb-4" />
            <p className="text-xl">{t('task.noRunningTasks')}</p>
          </div>
        </div>
      )}
    </div>
  );
}
