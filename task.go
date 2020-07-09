package xxl

import (
	"context"
)

//任务执行函数
type TaskFunc func(cxt context.Context, param *RunReq)

//任务
type Task struct {
	Id        int64
	Name      string
	Ext       context.Context
	Param     *RunReq
	fn        TaskFunc
	Cancel    context.CancelFunc
	StartTime int64
	EndTime   int64
}

//运行任务
func (t *Task) Run(callback func()) {
	t.Ext, t.Cancel = context.WithCancel(context.Background())
	t.fn(t.Ext, t.Param)
	callback()
	return
}

//任务信息
func (t *Task) Info() string {
	return "任务ID[" + Int64ToStr(t.Id) + "]任务名称[" + t.Name + "]参数：" + t.Param.ExecutorParams
}
