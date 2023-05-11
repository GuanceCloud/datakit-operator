package election

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type taskManager struct {
	tasks           tasks
	tasksScoreCount int
	workers         map[string]*worker
	mu              sync.Mutex
}

func newTaskManager() *taskManager {
	return &taskManager{
		workers: make(map[string]*worker),
		mu:      sync.Mutex{},
	}
}

func (t *taskManager) checkList(taskList map[string]int) error {
	if len(t.tasks) == 0 {
		for name, score := range taskList {
			t.tasks = append(t.tasks, newTask(name, score))
		}

		sort.Sort(t.tasks)
		for _, tk := range t.tasks {
			t.tasksScoreCount += tk.score
		}
		return nil
	}

	if len(t.tasks) != len(taskList) {
		return fmt.Errorf("mismatched number of tasks, expect %d, got %d", len(t.tasks), len(taskList))
	}

	for name := range taskList {
		if !t.registedTask(name) {
			return fmt.Errorf("unregistered task: %s, taskList: %#v", name, taskList)
		}
	}

	return nil
}

func (t *taskManager) takeTask(id string) []string {
	if id == "" {
		return nil
	}
	t.cleanInactiveWorkers()

	if wk, ok := t.workers[id]; ok {
		wk.lastTime = time.Now()
		return wk.currentTasks
	}

	t.workers[id] = newWorker()
	t.reassignTasksForWorker()

	return t.workers[id].currentTasks
}

func (t *taskManager) reassignTasksForWorker() {
	for _, wk := range t.workers {
		wk.currentTasks = nil
	}

	plans := t.reassignTasks()

	index := 0
	for _, wk := range t.workers {
		if len(plans) == index {
			break
		}
		wk.currentTasks = plans[index]
		index++
	}
}

func (t *taskManager) reassignTasks() [][]string {
	var plans [][]string
	var plan []string

	workerNum := len(t.workers)
	if workerNum == 0 {
		return nil
	}

	factor := t.tasksScoreCount / workerNum
	scoreCount := 0

	left := 0
	right := len(t.tasks) - 1

	for ; left < workerNum && left <= right; left++ {
		scoreCount += t.tasks[left].score
		plan = append(plan, t.tasks[left].name)

		for ; right > left; right-- {
			if t.tasks[right].score+scoreCount > factor && left != workerNum-1 {
				break
			}
			scoreCount += t.tasks[right].score
			plan = append(plan, t.tasks[right].name)
		}

		plans = append(plans, plan)
		plan = nil
		scoreCount = 0
	}

	return plans
}

func (t *taskManager) cleanInactiveWorkers() {
	oldCount := len(t.workers)

	for id, wk := range t.workers {
		if time.Since(wk.lastTime) > time.Second*10 {
			delete(t.workers, id)
		}
	}

	if len(t.workers) != oldCount {
		t.reassignTasksForWorker()
	}
}

func (t *taskManager) registedTask(name string) bool {
	for _, tk := range t.tasks {
		if tk.name == name {
			return true
		}
	}
	return false
}

type worker struct {
	currentTasks []string
	lastTime     time.Time
}

func newWorker() *worker {
	return &worker{lastTime: time.Now()}
}

type task struct {
	name  string
	score int
}

func newTask(name string, score int) task {
	return task{name: name, score: score}
}

type tasks []task

func (ts tasks) Len() int           { return len(ts) }
func (ts tasks) Swap(i, j int)      { ts[i], ts[j] = ts[j], ts[i] }
func (ts tasks) Less(i, j int) bool { return ts[i].score < ts[j].score }
