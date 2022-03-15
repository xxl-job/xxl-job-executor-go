package xxl

import (
	"context"
	"fmt"
	"runtime/debug"
)

// TaskFunc 任务执行函数
type TaskFunc func(cxt context.Context, param *RunReq) string

// Task 任务
type Task struct {
	Id        int64
	Name      string
	Ext       context.Context
	Param     *RunReq
	fn        TaskFunc
	Cancel    context.CancelFunc
	StartTime int64
	EndTime   int64
	//日志
	log Logger
}

// Run 运行任务
func (t *Task) Run(callback func(code int64, msg string)) {
	defer func(cancel func()) {
		if err := recover(); err != nil {
			t.log.Info(t.Info()+" panic: %v", err)
			debug.PrintStack() //堆栈跟踪
			callback(500, "task panic:"+fmt.Sprintf("%v", err))
		}
		cancel()
	}(t.Cancel)
	// 回传任务的执行结果
	done := make(chan string, 1)
	go func(t *Task) {
		msg := t.fn(t.Ext, t.Param)
		done <- msg
	}(t)
	select {
	// 判断任务执行是否超时
	case <-t.Ext.Done():
		t.log.Error("time out")
		callback(502, "job execute timeout")
	case msg := <-done:
		callback(200, msg)
	}
	return
}

// Info 任务信息
func (t *Task) Info() string {
	return "任务ID[" + Int64ToStr(t.Id) + "]任务名称[" + t.Name + "]参数：" + t.Param.ExecutorParams
}
