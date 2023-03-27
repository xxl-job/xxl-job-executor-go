package xxl

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Executor 执行器
type Executor interface {
	// Init 初始化
	Init(...Option)
	// LogHandler 日志查询
	LogHandler(handler LogHandler)
	// RegTaskNoStorage 注册任务到内存
	RegTaskNoStorage(pattern string)
	// RegTempTask 注册临时性任务
	RegTempTask(pattern string, handlerName string, jobId, expireAt int64)
	// RegPersistenceTask 注册任务
	RegPersistenceTask(pattern, handlerName string, jobId int64)
	// RunTask 运行任务
	RunTask(writer http.ResponseWriter, request *http.Request)
	// KillTask 杀死任务
	KillTask(writer http.ResponseWriter, request *http.Request)
	// TaskLog 任务日志
	TaskLog(writer http.ResponseWriter, request *http.Request)
	// Beat 心跳检测
	Beat(writer http.ResponseWriter, request *http.Request)
	// IdleBeat 忙碌检测
	IdleBeat(writer http.ResponseWriter, request *http.Request)
	// Run 运行服务
	Run() error
	// Stop 停止服务
	Stop()
}

// NewExecutor 创建执行器
func NewExecutor(opts ...Option) Executor {
	return newExecutor(opts...)
}

func newExecutor(opts ...Option) *executor {
	options := newOptions(opts...)
	executor := &executor{
		opts:     options,
		stopChan: make(chan struct{}),
	}
	return executor
}

type executor struct {
	opts     Options
	address  string
	regList  *taskList //注册任务列表
	runList  *taskList //正在执行任务列表
	mu       sync.RWMutex
	log      Logger
	stopChan chan struct{}

	logHandler LogHandler //日志查询handler
}

func (e *executor) Init(opts ...Option) {
	for _, o := range opts {
		o(&e.opts)
	}
	e.log = e.opts.l
	e.regList = &taskList{
		data: make(map[string]*Task),
	}
	e.runList = &taskList{
		data: make(map[string]*Task),
	}
	e.address = e.opts.ExecutorIp + ":" + e.opts.ExecutorPort
	go e.registry()
}

// LogHandler 日志handler
func (e *executor) LogHandler(handler LogHandler) {
	e.logHandler = handler
}

func (e *executor) Run() (err error) {
	// 创建路由器
	mux := http.NewServeMux()
	// 设置路由规则
	mux.HandleFunc("/run", e.runTask)
	mux.HandleFunc("/kill", e.killTask)
	mux.HandleFunc("/log", e.taskLog)
	mux.HandleFunc("/beat", e.beat)
	mux.HandleFunc("/idleBeat", e.idleBeat)
	// 创建服务器
	server := &http.Server{
		Addr:         e.address,
		WriteTimeout: time.Second * 3,
		Handler:      mux,
	}
	// 监听端口并提供服务
	e.log.Debug("Starting server at " + e.address)
	go server.ListenAndServe()
	go e.ScanExpiredTask()
	return nil
}

func (e *executor) Stop() {
	e.stopChan <- struct{}{}
	e.registryRemove()
	e.wait()
}

// Wait 等待所有运行中的任务结束
func (e *executor) wait() {
	for e.runList.Len() > 0 {
		e.log.Info("正在运行任务数: %d\n", e.runList.Len())
		time.Sleep(time.Second)
	}
}

// ScanExpiredTask 扫描过期任务, 将其从内存中删除
func (e *executor) ScanExpiredTask() {
	t := time.NewTicker(time.Hour)
	for {
		select {
		case <-e.stopChan:
			return
		case <-t.C:
			shouldDelete := make([]string, 0)
			for taskName, task := range e.regList.GetAll() {
				storage := e.opts.Storage.Get(taskName)
				if storage == nil {
					shouldDelete = append(shouldDelete, taskName)
				} else if storage.Expired() && !e.runList.Exists(e.opts.GetRunningTaskId(task.Id, task.LogID)) {
					shouldDelete = append(shouldDelete, taskName)
				}
			}
			for _, name := range shouldDelete {
				e.regList.Del(name)
			}
		}
	}
}

// RegTempTask 注册临时性任务
func (e *executor) RegTempTask(pattern string, handlerName string, jobId, expireAt int64) {
	if expireAt <= 0 {
		e.log.Error("请设置有效的过期时间, 过期时间必须大于0s")
		return
	}

	var t = &Task{}
	e.opts.Storage.Set(pattern, handlerName, jobId, expireAt)
	e.regList.Set(pattern, t)
}

// RegPersistenceTask 注册任务
func (e *executor) RegPersistenceTask(pattern, handlerName string, jobId int64) {
	var t = &Task{}
	e.opts.Storage.Set(pattern, handlerName, jobId, Persistence)
	e.regList.Set(pattern, t)
}

func (e *executor) RegTaskNoStorage(pattern string) {
	var t = &Task{}
	e.regList.Set(pattern, t)
}

func notFoundHandler(cxt context.Context, param *RunReq) string {
	panic(fmt.Sprintf("JobId=%d, ExecutorHandler=%s, 未找到该任务处理器", param.JobID, param.ExecutorHandler))
}

//运行一个任务
func (e *executor) runTask(writer http.ResponseWriter, request *http.Request) {
	e.mu.Lock()
	defer e.mu.Unlock()
	req, _ := ioutil.ReadAll(request.Body)
	param := &RunReq{}
	err := json.Unmarshal(req, &param)
	if err != nil {
		_, _ = writer.Write(returnCall(param, FailureCode, "params err"))
		e.log.Error("参数解析错误:" + string(req))
		return
	}
	e.log.Info("任务参数:%+v", param)
	if !e.regList.Exists(param.ExecutorHandler) {
		if e.opts.Storage.Exists(param.ExecutorHandler) {
			// 因为taskList数据存储在内存, 动态注册的任务时, 除去被注册节点, 其他节点并没有该任务数据
			// 所以需要即时注册该任务
			e.RegTaskNoStorage(param.ExecutorHandler)
		} else {
			_, _ = writer.Write(returnCall(param, FailureCode, "Task not registered"))
			e.log.Error("任务[" + Int64ToStr(param.JobID) + "]没有注册:" + param.ExecutorHandler)
			return
		}
	}

	//阻塞策略处理
	runningTaskId := e.opts.GetRunningTaskId(param.JobID, param.LogID)
	if e.runList.Exists(runningTaskId) {
		if param.ExecutorBlockStrategy == coverEarly { //覆盖之前调度
			oldTask := e.runList.Get(runningTaskId)
			if oldTask != nil {
				oldTask.Cancel()
				e.runList.Del(e.opts.GetRunningTaskId(oldTask.Id, oldTask.LogID))
			}
		} else { //单机串行,丢弃后续调度 都进行阻塞
			_, _ = writer.Write(returnCall(param, FailureCode, "There are tasks running"))
			e.log.Error("任务[" + Int64ToStr(param.JobID) + "]已经在运行了:" + param.ExecutorHandler)
			return
		}
	}

	cxt := context.Background()
	task := &Task{}
	if param.ExecutorTimeout > 0 {
		task.Ext, task.Cancel = context.WithTimeout(cxt, time.Duration(param.ExecutorTimeout)*time.Second)
	} else {
		task.Ext, task.Cancel = context.WithCancel(cxt)
	}
	task.Id = param.JobID
	task.Name = param.ExecutorHandler
	task.Param = param
	task.log = e.log
	task.LogID = param.LogID

	e.runList.Set(runningTaskId, task)
	storage := e.opts.Storage.Get(param.ExecutorHandler)
	var handler TaskFunc = notFoundHandler
	if storage != nil {
		fn, exists := e.opts.HandlerMap[storage.HandleName]
		if exists {
			handler = fn
		}
	}

	go task.Run(func(code int64, msg string) {
		e.callback(task, code, msg)
	}, handler)
	e.log.Info("任务[" + Int64ToStr(param.JobID) + "]开始执行:" + param.ExecutorHandler)
	_, _ = writer.Write(returnGeneral())
}

//删除一个任务
func (e *executor) killTask(writer http.ResponseWriter, request *http.Request) {
	e.mu.Lock()
	defer e.mu.Unlock()
	req, _ := ioutil.ReadAll(request.Body)
	param := &killReq{}
	_ = json.Unmarshal(req, &param)
	if !e.runList.Exists(Int64ToStr(param.JobID)) {
		_, _ = writer.Write(returnKill(param, FailureCode))
		e.log.Error("任务[" + Int64ToStr(param.JobID) + "]没有运行")
		return
	}
	task := e.runList.Get(Int64ToStr(param.JobID))
	task.Cancel()
	e.runList.Del(Int64ToStr(param.JobID))
	_, _ = writer.Write(returnGeneral())
}

//任务日志
func (e *executor) taskLog(writer http.ResponseWriter, request *http.Request) {
	var res *LogRes
	data, err := ioutil.ReadAll(request.Body)
	req := &LogReq{}
	if err != nil {
		e.log.Error("日志请求失败:" + err.Error())
		reqErrLogHandler(writer, req, err)
		return
	}
	err = json.Unmarshal(data, &req)
	if err != nil {
		e.log.Error("日志请求解析失败:" + err.Error())
		reqErrLogHandler(writer, req, err)
		return
	}
	e.log.Info("日志请求参数:%+v", req)
	if e.logHandler != nil {
		res = e.logHandler(req)
	} else {
		res = defaultLogHandler(req)
	}
	str, _ := json.Marshal(res)
	_, _ = writer.Write(str)
}

// 心跳检测
func (e *executor) beat(writer http.ResponseWriter, request *http.Request) {
	e.log.Info("心跳检测")
	_, _ = writer.Write(returnGeneral())
}

// 忙碌检测
func (e *executor) idleBeat(writer http.ResponseWriter, request *http.Request) {
	e.mu.Lock()
	defer e.mu.Unlock()
	req, _ := ioutil.ReadAll(request.Body)
	param := &idleBeatReq{}
	err := json.Unmarshal(req, &param)
	if err != nil {
		_, _ = writer.Write(returnIdleBeat(FailureCode))
		e.log.Error("参数解析错误:" + string(req))
		return
	}
	if e.runList.Exists(Int64ToStr(param.JobID)) {
		_, _ = writer.Write(returnIdleBeat(FailureCode))
		e.log.Error("idleBeat任务[" + Int64ToStr(param.JobID) + "]正在运行")
		return
	}
	e.log.Debug("忙碌检测任务参数:%v", param)
	_, _ = writer.Write(returnGeneral())
}

//注册执行器到调度中心
func (e *executor) registry() {

	t := time.NewTimer(time.Second * 0) //初始立即执行
	defer t.Stop()
	req := &Registry{
		RegistryGroup: "EXECUTOR",
		RegistryKey:   e.opts.RegistryKey,
		RegistryValue: "http://" + e.address,
	}
	param, err := json.Marshal(req)
	if err != nil {
		log.Fatal("执行器注册信息解析失败:" + err.Error())
	}
	for {
		<-t.C
		t.Reset(time.Second * time.Duration(20)) //20秒心跳防止过期
		func() {
			result, err := e.post("/api/registry", string(param))
			if err != nil {
				e.log.Error("执行器注册失败1:" + err.Error())
				return
			}
			defer result.Body.Close()
			body, err := ioutil.ReadAll(result.Body)
			if err != nil {
				e.log.Error("执行器注册失败2:" + err.Error())
				return
			}
			res := &res{}
			_ = json.Unmarshal(body, &res)
			if res.Code != SuccessCode {
				e.log.Error("执行器注册失败3:" + string(body))
				return
			}
			e.log.Debug("执行器注册成功:" + string(body))
		}()

	}
}

//执行器注册摘除
func (e *executor) registryRemove() {
	t := time.NewTimer(time.Second * 0) //初始立即执行
	defer t.Stop()
	req := &Registry{
		RegistryGroup: "EXECUTOR",
		RegistryKey:   e.opts.RegistryKey,
		RegistryValue: "http://" + e.address,
	}
	param, err := json.Marshal(req)
	if err != nil {
		e.log.Error("执行器摘除失败:" + err.Error())
		return
	}
	res, err := e.post("/api/registryRemove", string(param))
	if err != nil {
		e.log.Error("执行器摘除失败:" + err.Error())
		return
	}
	body, err := ioutil.ReadAll(res.Body)
	e.log.Info("执行器摘除成功:" + string(body))
	_ = res.Body.Close()
}

//回调任务列表
func (e *executor) callback(task *Task, code int64, msg string) {
	runningTaskId := e.opts.GetRunningTaskId(task.Id, task.LogID)
	e.runList.Del(runningTaskId)
	res, err := e.post("/api/callback", string(returnCall(task.Param, code, msg)))
	if err != nil {
		e.log.Error("callback err : ", err.Error())
		return
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		e.log.Error("callback ReadAll err : ", err.Error())
		return
	}
	e.log.Info("任务回调成功:" + string(body))
}

//post
func (e *executor) post(action, body string) (resp *http.Response, err error) {
	request, err := http.NewRequest("POST", e.opts.ServerAddr+action, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json;charset=UTF-8")
	request.Header.Set("XXL-JOB-ACCESS-TOKEN", e.opts.AccessToken)
	client := http.Client{
		Timeout: e.opts.Timeout,
	}
	return client.Do(request)
}

// RunTask 运行任务
func (e *executor) RunTask(writer http.ResponseWriter, request *http.Request) {
	e.runTask(writer, request)
}

// KillTask 删除任务
func (e *executor) KillTask(writer http.ResponseWriter, request *http.Request) {
	e.killTask(writer, request)
}

// TaskLog 任务日志
func (e *executor) TaskLog(writer http.ResponseWriter, request *http.Request) {
	e.taskLog(writer, request)
}

// Beat 心跳检测
func (e *executor) Beat(writer http.ResponseWriter, request *http.Request) {
	e.beat(writer, request)
}

// IdleBeat 忙碌检测
func (e *executor) IdleBeat(writer http.ResponseWriter, request *http.Request) {
	e.idleBeat(writer, request)
}
