package xxl

import (
	"context"
	"fmt"
	"runtime/debug"
)

// TaskFunc 任务执行函数
type TaskFunc func(cxt context.Context, param *RunReq) string

type TaskFuncWithError func(cxt context.Context, runReq RunReq) error

// Task 任务
type Task struct {
	Id          int64
	Name        string
	Ext         context.Context
	Param       *RunReq
	fn          TaskFunc
	fnWithError TaskFuncWithError
	Cancel      context.CancelFunc
	StartTime   int64
	EndTime     int64
	//日志
	log Logger
}

// Run 运行任务
func (t *Task) Run(callback func(code int64, msg string)) {
	defer func(cancel func()) {
		if err := recover(); err != nil {
			t.log.Info(t.Info()+" panic: %v", err)
			debug.PrintStack() //堆栈跟踪
			callback(FailureCode, fmt.Sprintf("task panic:%v", err))
			cancel()
		}
	}(t.Cancel)
	if t.fnWithError != nil {
		err := t.fnWithError(t.Ext, *t.Param)
		if err != nil {
			t.log.Error("job execute error, jobName: %s, params: %s, error: %v", t.Name, t.Param.ExecutorParams, err)
			callback(FailureCode, err.Error())
			return
		}
		callback(SuccessCode, "ok")
	} else {
		msg := t.fn(t.Ext, t.Param)
		callback(SuccessCode, msg)
	}
}

// Info 任务信息
func (t *Task) Info() string {
	return fmt.Sprintf("任务ID[%d]任务名称[%s]参数:%s", t.Id, t.Name, t.Param.ExecutorParams)
}
