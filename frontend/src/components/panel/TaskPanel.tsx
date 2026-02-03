import { appconf, enums, models } from "../../../wailsjs/go/models";
import { EventsOff, EventsOn, EventsOnce, EventsOffAll, EventsOnMultiple } from "../../../wailsjs/runtime";
import { toast } from "react-hot-toast";
import { PauseTask, CancelTask, ResumeTask} from "../../../wailsjs/go/service/TaskService";

import { useEffect, useRef, useState } from "react";

export function TaskPanel() {

    const [tasks, setTasks] = useState<models.TaskNotice[]>([]);

    // 监听任务更新事件
  useEffect(() => {
    const unlistenTaskUpdate = EventsOn("game_updates", (data: any) => {
      console.log("Received task update:", data);
      // setCurrentTask(task);

      // 将接收到的数据转换为Task对象
    // 注意：使用正确的语法从data对象获取值
    const task : models.TaskNotice = new models.TaskNotice(data);
      
      setTasks((prevTasks) => {
        // 检查任务是否已经存在
        const existingTaskIndex = prevTasks.findIndex((t) => t.id === task.id);
        if (existingTaskIndex !== -1) {
          // 如果任务已经存在，则更新其状态
          const updatedTasks = [...prevTasks];
          updatedTasks[existingTaskIndex] = task;
          return updatedTasks;
        } else {
          // 如果任务不存在，则添加到列表中
          return [...prevTasks, task];
        }
      })
    });

    return () => {
      if (unlistenTaskUpdate) {
        unlistenTaskUpdate(); // 取消事件监听
      }
    };
  }, []);
  return (
    <div> 
        {
             tasks.length > 0 ? (
                <div className="task-list flex flex-row gap-3 p-3 max-h-96 overflow-x-auto">
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
                                    
                                    {/* 进度条 */}
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
                                
                                {/* 控制按钮 */}
                                <div className="flex flex-col gap-1 ml-4">
                                    {(task.status === "已开始" ) && (
                                        <button
                                            onClick={() => PauseTask(task.id)}
                                            className="p-2 rounded-md bg-yellow-100 hover:bg-yellow-200 text-yellow-800 dark:bg-yellow-900/30 dark:hover:bg-yellow-800/50 dark:text-yellow-400 transition-colors"
                                            title="暂停任务"
                                        >
                                            <div className="i-mdi-pause text-lg"></div>
                                        </button>
                                    )}
                                    
                                    {(task.status === "暂停" ) && (
                                        <button
                                            onClick={() => ResumeTask(task.id)}
                                            className="p-2 rounded-md bg-green-100 hover:bg-green-200 text-green-800 dark:bg-green-900/30 dark:hover:bg-green-800/50 dark:text-green-400 transition-colors"
                                            title="恢复任务"
                                        >
                                            <div className="i-mdi-play text-lg"></div>
                                        </button>
                                    )}
                                    
                                    {(task.status !== "完成" && task.status !== "错误" && task.status !== "取消") && (
                                        <button
                                            onClick={() => CancelTask(task.id)}
                                            className="p-2 rounded-md bg-red-100 hover:bg-red-200 text-red-800 dark:bg-red-900/30 dark:hover:bg-red-800/50 dark:text-red-400 transition-colors"
                                            title="取消任务"
                                        >
                                            <div className="i-mdi-cancel text-lg"></div>
                                        </button>
                                    )}
                                    {(task.status === "完成" ) && (
                                        <button
                                            onClick={() => setTasks(tasks.filter((t) => t.id !== task.id))}
                                            className="p-2 rounded-md bg-gray-100 hover:bg-gray-200 text-gray-800 dark:bg-gray-700 dark:hover:bg-gray-600 dark:text-gray-300 transition-colors"
                                            title="删除任务"
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
                <div className="text-center py-4 text-brand-500 dark:text-brand-400 text-sm">无运行中的任务</div>
            )
        }
    </div>
  )

}