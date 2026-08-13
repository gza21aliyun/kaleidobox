package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"lunabox/internal/appconf"
	"lunabox/internal/enums"
	"lunabox/internal/models"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// TaskFunction 是任务执行函数的类型定义，接受上下文、任务数据和进度更新函数
type TaskFunction func(ctx context.Context, data string,
	updateProgress func(completed int, total int, workingOn string,
		warning string, itemId string, itemEvent enums.TaskStatus, resultGames []models.ResultGames, itemData interface{})) error

type TaskService struct {
	ctx    context.Context
	db     *sql.DB
	config *appconf.AppConfig

	// 任务管理相关字段
	taskQueue     chan *models.Task
	taskMutex     sync.RWMutex
	activeTask    *models.Task
	taskCancel    context.CancelFunc
	taskWG        sync.WaitGroup
	taskFunctions map[string]TaskFunction // 使用字符串作为键，更灵活
}

func NewTaskService() *TaskService {
	return &TaskService{
		taskQueue:     make(chan *models.Task, 100), // 队列大小可根据需求调整
		taskFunctions: make(map[string]TaskFunction),
	}
}

func (s *TaskService) Init(ctx context.Context, db *sql.DB, config *appconf.AppConfig) {
	s.ctx = ctx
	s.db = db
	s.config = config

	// 启动任务队列处理器
	go s.processTaskQueue()
}

// RegisterTaskFunction 注册任务处理函数，key是任务类型的标识符
func (s *TaskService) RegisterTaskFunction(key string, fn TaskFunction) {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()
	s.taskFunctions[key] = fn
}

// processTaskQueue 处理任务队列
func (s *TaskService) processTaskQueue() {
	for task := range s.taskQueue {
		s.executeTask(task)
	}
}

// executeTask 执行单个任务
func (s *TaskService) executeTask(task *models.Task) {
	s.taskMutex.Lock()
	s.activeTask = task
	s.taskMutex.Unlock()

	// 更新任务状态为开始
	task.Status = enums.Started
	task.WorkingOn = "开始执行任务"
	s.notifyFrontend(task)

	// 创建带取消功能的上下文
	taskCtx, cancel := context.WithCancel(context.Background())
	s.taskMutex.Lock()
	s.taskCancel = cancel
	s.taskMutex.Unlock()

	// 执行任务
	s.runTask(task, taskCtx)

	// 任务完成后更新状态
	s.taskMutex.Lock()
	s.activeTask = nil
	s.taskCancel = nil
	s.taskMutex.Unlock()
}

// runTask 实际执行任务的逻辑
func (s *TaskService) runTask(task *models.Task, ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Task panic recovered: %v", r)
			task.Status = enums.Error
			task.Warning = fmt.Sprintf("任务执行异常: %v", r)
			s.notifyFrontend(task)
		}
	}()

	// 从任务类型字段获取注册的任务函数
	s.taskMutex.RLock()
	taskFn, exists := s.taskFunctions[string(task.Id)]
	s.taskMutex.RUnlock()

	if !exists {
		task.Status = enums.Error
		task.Warning = fmt.Sprintf("未找到类型为 %s 的任务处理函数", task.Type)
		s.notifyFrontend(task)
		return
	}

	// 创建一个更新进度的函数
	updateProgress := func(completed int, total int, workingOn string,
		warning string, itemId string, itemEvent enums.TaskStatus, resultGames []models.ResultGames, itemData interface{}) {
		s.taskMutex.Lock()
		defer s.taskMutex.Unlock()

		if s.activeTask != nil {
			s.activeTask.Completed = completed
			if total > 0 {
				s.activeTask.Total = total
			}
			// if warning != "" {
			// 	s.activeTask.Warning = warning
			// }
			if len(resultGames) > 0 {
				s.activeTask.Title = resultGames[0].Title
			}
			s.activeTask.Warning = warning
			s.activeTask.WorkingOn = workingOn
			s.activeTask.ItemId = itemId
			s.activeTask.ItemStatus = itemEvent
			s.activeTask.ResultGames = resultGames
			s.activeTask.ItemData = itemData
			s.notifyFrontend(s.activeTask)
		}
	}

	// 将任务数据转换为json.RawMessage
	var jsonData string
	if task.Data != nil {
		// var jsontext = "{\"aa\":\"aatext\",\"source\": \"Eroscape\"}"
		// var jsonStruct struct {
		// 	Aa string `json:"aa"`
		// 	Source string `json:"source"`
		// }
		// // 使用 json.Unmarshal 将 JSON 字符串解析到结构体中
		// err := json.Unmarshal([]byte(jsontext), &jsonStruct)

		jsonBytes, err := json.Marshal(task.Data)
		if err != nil {
			task.Status = enums.Error
			task.Warning = fmt.Sprintf("序列化任务数据失败: %v", err)
			s.notifyFrontend(task)
			return
		}
		jsonData = string(jsonBytes)
	}

	// 创建一个goroutine来执行任务函数，这样我们可以监控进度
	done := make(chan error, 1)
	go func() {
		// 执行任务函数
		err := taskFn(ctx, jsonData, updateProgress)
		done <- err
	}()

	// 监控任务进度
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			// 任务完成
			s.taskMutex.Lock()
			defer s.taskMutex.Unlock()

			if s.activeTask != nil {
				if err != nil {
					s.activeTask.Status = enums.Error
					s.activeTask.Warning = err.Error()
				} else {
					// 确保所有子任务都标记为完成
					s.activeTask.Completed = s.activeTask.Total
					s.activeTask.Status = enums.Completed
					s.activeTask.WorkingOn = "任务已完成"
				}
				s.notifyFrontend(s.activeTask)
			}
			return

		case <-ticker.C:
			// 检查是否被取消
			select {
			case <-ctx.Done():
				s.taskMutex.Lock()
				if s.activeTask != nil {
					s.activeTask.Status = enums.Canceled
					s.notifyFrontend(s.activeTask)
				}
				s.taskMutex.Unlock()
				return
			default:
			}

			// 检查是否暂停
			s.taskMutex.Lock()
			if s.activeTask != nil && s.activeTask.Status == enums.Paused {
				// 等待恢复或取消
				for s.activeTask.Status == enums.Paused {
					s.taskMutex.Unlock()
					time.Sleep(100 * time.Millisecond)

					select {
					case <-ctx.Done():
						s.taskMutex.Lock()
						if s.activeTask != nil {
							s.activeTask.Status = enums.Canceled
							s.notifyFrontend(s.activeTask)
						}
						s.taskMutex.Unlock()
						return
					default:
					}

					s.taskMutex.Lock()
					if s.activeTask != nil && s.activeTask.Status == enums.Canceled {
						s.notifyFrontend(s.activeTask)
						s.taskMutex.Unlock()
						return
					}
				}

				// 恢复后继续
				if s.activeTask != nil {
					s.activeTask.Status = enums.Started
				}
			}

			// 发送当前进度更新
			if s.activeTask != nil {
				// s.notifyFrontend(s.activeTask)
			}
			s.taskMutex.Unlock()

		case <-ctx.Done():
			// 上下文被取消
			s.taskMutex.Lock()
			if s.activeTask != nil {
				s.activeTask.Status = enums.Canceled
				s.notifyFrontend(s.activeTask)
			}
			s.taskMutex.Unlock()
			return
		}
	}
}

// StartTask 开始一个新任务
func (s *TaskService) StartTask(name string, id string, delay int, taskType enums.TaskType, total int, data interface{}) error {
	task := &models.Task{
		Id:          id,
		Name:        name,
		Status:      enums.Initial,
		Type:        taskType,
		Completed:   0,
		Total:       total,
		WorkingOn:   "准备开始",
		Description: fmt.Sprintf("任务: %s 总数: %d", taskType, total),
		Warning:     "",
		Deley:       delay,
		Data:        data, // 存储任务数据
	}

	s.notifyFrontend(task)

	// 将任务加入队列
	select {
	case s.taskQueue <- task:
		return nil
	default:
		return fmt.Errorf("任务队列已满")
	}
}

// PauseTask 暂停当前任务
func (s *TaskService) PauseTask(id string) error {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	if s.activeTask == nil {
		return fmt.Errorf("没有活跃的任务可以暂停")
	}

	if s.activeTask.Status == enums.Started {
		s.activeTask.Status = enums.Paused
		s.activeTask.WorkingOn = "任务已暂停"
		s.notifyFrontend(s.activeTask)
		return nil
	}

	return fmt.Errorf("任务不在运行状态，无法暂停")
}

// ResumeTask 恢复暂停的任务
func (s *TaskService) ResumeTask(id string) error {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	if s.activeTask == nil {
		return fmt.Errorf("没有活跃的任务可以恢复")
	}

	if s.activeTask.Status == enums.Paused {
		s.activeTask.Status = enums.Started
		s.activeTask.WorkingOn = "任务已恢复"
		s.notifyFrontend(s.activeTask)
		return nil
	}

	return fmt.Errorf("任务不在暂停状态，无法恢复")
}

// CancelTask 取消当前任务
func (s *TaskService) CancelTask(id string) error {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	if s.activeTask == nil {
		return fmt.Errorf("没有活跃的任务可以取消")
	}

	if s.taskCancel != nil {
		s.taskCancel()
	}

	s.activeTask.Status = enums.Canceled
	s.activeTask.WorkingOn = "任务已取消"
	s.notifyFrontend(s.activeTask)

	return nil
}

// GetActiveTask 获取当前活跃的任务
func (s *TaskService) GetActiveTask() *models.Task {
	s.taskMutex.RLock()
	defer s.taskMutex.RUnlock()

	if s.activeTask != nil {
		taskCopy := *s.activeTask
		return &taskCopy
	}
	return nil
}

func (s *TaskService) GetTaskNotice(task models.Task) models.TaskNotice {
	taskNotice := models.TaskNotice{
		Id:          task.Id,
		Name:        task.Name,
		Status:      task.Status,
		Warning:     task.Warning,
		Total:       task.Total,
		Completed:   task.Completed,
		Title:       task.Title,
		ResultGames: task.ResultGames,
		Description: task.Description,
		WorkingOn:   task.WorkingOn,
		Type:        task.Type,
		ItemId:      task.ItemId,
		ItemStatus:  task.ItemStatus,
		ItemData:    task.ItemData,
	}
	if len(task.ResultGames) > 0 {
		taskNotice.Title = task.ResultGames[0].Title
	}
	return taskNotice
}

// notifyFrontend 向前端发送任务状态更新
func (s *TaskService) notifyFrontend(task *models.Task) {
	if s.ctx != nil {
		// taskNotice := models.TaskNotice{
		// 	Id:          task.Id,
		// 	Name:        task.Name,
		// 	Status:      task.Status,
		// 	Warning:     task.Warning,
		// 	Total:       task.Total,
		// 	Completed:   task.Completed,
		// 	Description: task.Description,
		// 	WorkingOn:   task.WorkingOn,
		// 	Type:        task.Type,
		// 	ItemId:      itemId,
		// }
		taskNotice := s.GetTaskNotice(*task)

		runtime.EventsEmit(s.ctx, task.Name, taskNotice)
	}
}

// Close 关闭任务服务，清理资源
func (s *TaskService) Close() {
	close(s.taskQueue)
	if s.taskCancel != nil {
		s.taskCancel()
	}
	s.taskWG.Wait()
}
