package xxl

import (
	"context"
	"fmt"
	"log"
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
func (t *Task) Run(callback func(code int64, msg string)) {
	t.Ext, t.Cancel = context.WithCancel(context.Background())
	defer func(cancel func()) {
		if err := recover(); err != nil {
			log.Println(t.Info()+" panic: ", err)
			callback(500, "task panic:"+fmt.Sprintf("%v", err))
			cancel()
		}
	}(t.Cancel)
	t.fn(t.Ext, t.Param)
	callback(200, "")
	return
}

//任务信息
func (t *Task) Info() string {
	return "任务ID[" + Int64ToStr(t.Id) + "]任务名称[" + t.Name + "]参数：" + t.Param.ExecutorParams
}
