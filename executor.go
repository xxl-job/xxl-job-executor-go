package xxl

import (
	"context"
	"encoding/json"
	"fmt"
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
	RegTask(pattern string, task TaskFunc)
	Run() error
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
}

func (e *executor) Init(opts ...Option) {
	for _, o := range opts {
		o(&e.opts)
	}
	e.regList = &taskList{
		data: make(map[string]*Task),
	}
	e.runList = &taskList{
		data: make(map[string]*Task),
	}
	e.address = e.opts.ExecutorIp + ":" + e.opts.ExecutorPort
	go e.registry()
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
	log.Println("Starting server at " + e.address)
	go server.ListenAndServe()
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	e.registryRemove()
	return nil
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
	json.Unmarshal(req, &param)
	if !e.regList.Exists(param.ExecutorHandler) {
		writer.Write(returnCall(param, 500))
		log.Println("任务[" + Int64ToStr(param.JobID) + "]没有注册:" + param.ExecutorHandler)
		return
	}
	task := e.regList.Get(param.ExecutorHandler)
	task.Ext, task.Cancel = context.WithCancel(context.Background())
	task.Id = param.JobID
	task.Name = param.ExecutorHandler
	task.Param = param
	e.runList.Set(Int64ToStr(param.JobID), task)
	go task.Run(func() {
		e.callback(task)
	})
	log.Println("任务[" + Int64ToStr(param.JobID) + "]开始执行:" + param.ExecutorHandler)
	writer.Write(returnGeneral())
}

//删除一个任务
func (e *executor) killTask(writer http.ResponseWriter, request *http.Request) {
	e.mu.Lock()
	defer e.mu.Unlock()
	req, _ := ioutil.ReadAll(request.Body)
	param := &killReq{}
	json.Unmarshal(req, &param)
	if !e.runList.Exists(Int64ToStr(param.JobID)) {
		writer.Write(returnKill(param, 500))
		log.Println("任务[" + Int64ToStr(param.JobID) + "]没有运行")
		return
	}
	task := e.runList.Get(Int64ToStr(param.JobID))
	task.Cancel()
	e.runList.Del(Int64ToStr(param.JobID))
	writer.Write(returnGeneral())
}

//任务日志
func (e *executor) taskLog(writer http.ResponseWriter, request *http.Request) {
	data, _ := ioutil.ReadAll(request.Body)
	req := &logReq{}
	json.Unmarshal(data, &req)

	writer.Write(returnLog(req, 200))
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
		result, err := e.post("/api/registry", string(param))
		if err != nil {
			log.Println("执行器注册失败:" + err.Error())
		}
		body, err := ioutil.ReadAll(result.Body)
		res := &res{}
		json.Unmarshal(body, &res)
		if res.Code != 200 {
			log.Println("执行器注册失败:" + string(body))
		}
		log.Println("执行器注册成功:" + string(body))
		result.Body.Close()
		t.Reset(time.Second * time.Duration(20)) //20秒心跳防止过期
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
		log.Println("执行器摘除失败:" + err.Error())
	}
	res, err := e.post("/api/registryRemove", string(param))
	if err != nil {
		log.Println("执行器摘除失败:" + err.Error())
	}
	body, err := ioutil.ReadAll(res.Body)
	log.Println("执行器摘除成功:" + string(body))
	res.Body.Close()
}

//回调任务列表
func (e *executor) callback(task *Task) {
	res, err := e.post("/api/callback", string(returnCall(task.Param, 200)))
	if err != nil {
		fmt.Println(err)
	}
	body, err := ioutil.ReadAll(res.Body)
	log.Println("任务回调成功:" + string(body))
}

//post
func (e *executor) post(action, body string) (resp *http.Response, err error) {
	request, err := http.NewRequest("POST", e.opts.ServerAddr+action, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json;charset=UTF-8")
	request.Header.Set("XXL-RPC-ACCESS-TOKEN", e.opts.AccessToken)
	client := http.Client{
		Timeout: e.opts.Timeout,
	}
	return client.Do(request)
}
