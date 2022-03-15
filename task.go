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
	done := make(chan *res, 1)
	defer func() {
		if err := recover(); err != nil {
			t.log.Error(t.Info()+" panic: %v", err)
			debug.PrintStack() //堆栈跟踪
			msg := "task panic:" + fmt.Sprintf("%v", err)
			result := &res{
				Code: 500,
				Msg:  msg,
			}
			done <- result
		}
	}()
	// 启动监测协程
	go monitor(done, t, callback)
	msg := t.fn(t.Ext, t.Param)
	result := &res{
		Code: 200,
		Msg:  msg,
	}
	done <- result
}

// monitor 检测任务运行状态,任务执行结果回调
func monitor(done chan *res, task *Task, callback func(code int64, msg string)) {
	select {
	// 判断任务执行是否超时
	case <-task.Ext.Done():
		task.log.Error("time out")
		callback(502, "job execute timeout")
		return
	case result := <-done:
		callback(result.Code, result.Msg.(string))
		return
	}
}

// Info 任务信息
func (t *Task) Info() string {
	return "任务ID[" + Int64ToStr(t.Id) + "]任务名称[" + t.Name + "]参数：" + t.Param.ExecutorParams
}
