package xxl

import "sync"

//任务列表 [JobID]执行函数,并行执行时[+LogID]
type taskList struct {
	mu      sync.RWMutex
	data    map[string]*Task
	runList map[int64]*Task //运行任务列表
}

//设置数据
func (t *taskList) Set(key string, val *Task) {
	t.mu.Lock()
	t.data[key] = val
	t.mu.Unlock()
}

//获取数据
func (t *taskList) Get(key string) *Task {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.data[key]
}

//获取数据
func (t *taskList) GetAll() map[string]*Task {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.data
}

//设置数据
func (t *taskList) Del(key string) {
	t.mu.Lock()
	delete(t.data, key)
	t.mu.Unlock()
}

//长度
func (t *taskList) Len() int {
	return len(t.data)
}

//Key是否存在
func (t *taskList) Exists(key string) bool {
	_, ok := t.data[key]
	return ok
}

//任务是否在运行列表
func (t *taskList) IsRun(key int64) bool {
	_, ok := t.runList[key]
	return ok
}

//获取运行的任务
func (t *taskList) SetRunTask(key int64, val *Task) {
	t.mu.Lock()
	t.runList[key] = val
	t.mu.Unlock()
}

//获取运行的任务
func (t *taskList) GetRunTask(key int64) *Task {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.runList[key]
}

//获取运行的任务
func (t *taskList) DelRunTask(key int64) {
	t.mu.RLock()
	delete(t.runList, key)
	defer t.mu.RUnlock()
}