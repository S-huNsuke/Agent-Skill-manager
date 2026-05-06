package app

import (
	"sync"
	"time"
)

type scheduledTask struct {
	name     string
	interval time.Duration
	ticker   *time.Ticker
	stopCh   chan struct{}
}

type Scheduler struct {
	mu       sync.Mutex
	tasks    map[string]*scheduledTask
	app      *App
	stopAll  chan struct{}
}

/** 创建新的调度器实例 */
func NewScheduler(app *App) *Scheduler {
	return &Scheduler{
		tasks:   make(map[string]*scheduledTask),
		app:     app,
		stopAll: make(chan struct{}),
	}
}

/** 根据自动化设置启动对应的定时任务 */
func (s *Scheduler) ApplySettings(settings AutomationSettingsViewModel) {
	s.StopAll()

	if settings.AutoSyncCatalog {
		s.Start("sync_catalog", 6*time.Hour, func() {
			s.app.SyncAllSources()
		})
	}

	if settings.AutoCheckUpdates {
		s.Start("check_updates", 24*time.Hour, func() {
			s.app.SyncAllSources()
		})
	}

	switch settings.HealthCheckSchedule {
	case "daily":
		s.Start("health_check", 24*time.Hour, func() {
			s.app.GetDiagnostics()
		})
	case "weekly":
		s.Start("health_check", 7*24*time.Hour, func() {
			s.app.GetDiagnostics()
		})
	}
}

/** 启动一个定时任务 */
func (s *Scheduler) Start(name string, interval time.Duration, fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[name]; exists {
		return
	}

	task := &scheduledTask{
		name:     name,
		interval: interval,
		stopCh:   make(chan struct{}),
	}

	task.ticker = time.NewTicker(interval)
	s.tasks[name] = task

	go func() {
		fn()
		for {
			select {
			case <-task.ticker.C:
				fn()
			case <-task.stopCh:
				return
			case <-s.stopAll:
				return
			}
		}
	}()
}

/** 停止指定名称的定时任务 */
func (s *Scheduler) Stop(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task, exists := s.tasks[name]; exists {
		task.ticker.Stop()
		close(task.stopCh)
		delete(s.tasks, name)
	}
}

/** 停止所有定时任务 */
func (s *Scheduler) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	close(s.stopAll)
	s.stopAll = make(chan struct{})

	for name, task := range s.tasks {
		task.ticker.Stop()
		close(task.stopCh)
		delete(s.tasks, name)
	}
}

/** 返回当前运行中的任务名称列表 */
func (s *Scheduler) RunningTasks() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	names := make([]string, 0, len(s.tasks))
	for name := range s.tasks {
		names = append(names, name)
	}
	return names
}

/** 在后台 goroutine 中执行一次性任务 */
func (s *Scheduler) RunOnce(fn func()) {
	go func() {
		fn()
	}()
}

/** 延迟执行一次性任务 */
func (s *Scheduler) RunAfter(delay time.Duration, fn func()) {
	go func() {
		select {
		case <-time.After(delay):
			fn()
		case <-s.stopAll:
		}
	}()
}

/** 检查调度器是否已初始化（用于 Wails 绑定） */
func (a *App) GetSchedulerStatus() []string {
	if a.scheduler == nil {
		return []string{}
	}
	return a.scheduler.RunningTasks()
}
