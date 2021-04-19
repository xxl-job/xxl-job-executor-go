package xxl

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

//执行器
type Executor interface {
	//初始化
	Init(...Option)
	//日志查询
	LogHandler(handler LogHandler)
	//注册任务
	RegTask(pattern string, task TaskFunc)
	//运行任务
	RunTask(writer http.ResponseWriter, request *http.Request)
	//杀死任务
	KillTask(writer http.ResponseWriter, request *http.Request)
	//任务日志
	TaskLog(writer http.ResponseWriter, request *http.Request)
	//运行服务
	Run() error
	//停止服务
	Stop()
}

//创建执行器
func NewExecutor(opts ...Option) Executor {
	return newExecutor(opts...)
}

func newExecutor(opts ...Option) *executor {
	options := newOptions(opts...)
	executor := &executor{
		opts: options,
	}
	return executor
}

type executor struct {
	opts    Options
	address string
	regList *taskList //注册任务列表
	runList *taskList //正在执行任务列表
	mu      sync.RWMutex
	log     Logger

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

//日志handler
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
	// 创建服务器
	server := &http.Server{
		Addr:         e.address,
		WriteTimeout: time.Second * 3,
		Handler:      mux,
	}
	// 监听端口并提供服务
	e.log.Info("Starting server at " + e.address)
	go server.ListenAndServe()
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	e.registryRemove()
	return nil
}

func (e *executor) Stop() {
	e.registryRemove()
}

//注册任务
func (e *executor) RegTask(pattern string, task TaskFunc) {
	var t = &Task{}
	t.fn = task
	e.regList.Set(pattern, t)
	return
}

//运行一个任务
func (e *executor) runTask(writer http.ResponseWriter, request *http.Request) {
	e.mu.Lock()
	defer e.mu.Unlock()
	req, _ := ioutil.ReadAll(request.Body)
	param := &RunReq{}
	err := json.Unmarshal(req, &param)
	if err != nil {
		_, _ = writer.Write(returnCall(param, 500, "params err"))
		e.log.Error("参数解析错误:" + string(req))
		return
	}
	e.log.Info("任务参数:%v", param)
	if !e.regList.Exists(param.ExecutorHandler) {
		_, _ = writer.Write(returnCall(param, 500, "Task not registered"))
		e.log.Error("任务[" + Int64ToStr(param.JobID) + "]没有注册:" + param.ExecutorHandler)
		return
	}

	//阻塞策略处理
	if e.runList.Exists(Int64ToStr(param.JobID)) {
		if param.ExecutorBlockStrategy == coverEarly { //覆盖之前调度
			oldTask := e.runList.Get(Int64ToStr(param.JobID))
			if oldTask != nil {
				oldTask.Cancel()
				e.runList.Del(Int64ToStr(oldTask.Id))
			}
		} else { //单机串行,丢弃后续调度 都进行阻塞
			_, _ = writer.Write(returnCall(param, 500, "There are tasks running"))
			e.log.Error("任务[" + Int64ToStr(param.JobID) + "]已经在运行了:" + param.ExecutorHandler)
			return
		}
	}

	cxt := context.Background()
	task := e.regList.Get(param.ExecutorHandler)
	if param.ExecutorTimeout > 0 {
		task.Ext, task.Cancel = context.WithTimeout(cxt, time.Duration(param.ExecutorTimeout)*time.Second)
	} else {
		task.Ext, task.Cancel = context.WithCancel(cxt)
	}
	task.Id = param.JobID
	task.Name = param.ExecutorHandler
	task.Param = param
	task.log = e.log

	e.runList.Set(Int64ToStr(task.Id), task)
	go task.Run(func(code int64, msg string) {
		e.callback(task, code, msg)
	})
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
		_, _ = writer.Write(returnKill(param, 500))
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
			if res.Code != 200 {
				e.log.Error("执行器注册失败3:" + string(body))
				return
			}
			e.log.Info("执行器注册成功:" + string(body))
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
	}
	res, err := e.post("/api/registryRemove", string(param))
	if err != nil {
		e.log.Error("执行器摘除失败:" + err.Error())
	}
	body, err := ioutil.ReadAll(res.Body)
	e.log.Info("执行器摘除成功:" + string(body))
	_ = res.Body.Close()
}

//回调任务列表
func (e *executor) callback(task *Task, code int64, msg string) {
	res, err := e.post("/api/callback", string(returnCall(task.Param, code, msg)))
	if err != nil {
		e.log.Error("callback err : ", err.Error())
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		e.log.Error("callback ReadAll err : ", err.Error())
	}
	e.runList.Del(Int64ToStr(task.Id))
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

//runTask
func (e *executor) RunTask(writer http.ResponseWriter, request *http.Request) {
	e.runTask(writer, request)
}

//killTask
func (e *executor) KillTask(writer http.ResponseWriter, request *http.Request) {
	e.killTask(writer, request)
}

//taskLog
func (e *executor) TaskLog(writer http.ResponseWriter, request *http.Request) {
	e.taskLog(writer, request)
}
