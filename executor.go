package xxl

import (
	"context"
	"encoding/json"
	"github.com/xxl-job/xxl-job-executor-go/backoff"
	"io"
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

// Executor 执行器
type Executor interface {
	// Init 初始化
	Init(...Option)
	// LogHandler 日志查询
	LogHandler(handler LogHandler)
	// Use 使用中间件
	Use(middlewares ...Middleware)
	// RegTask 注册任务
	RegTask(pattern string, task TaskFunc)
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
	e := &executor{
		opts: options,
	}
	return e
}

type executor struct {
	opts        Options
	address     string
	regList     *taskList //注册任务列表
	runList     *taskList //正在执行任务列表
	mu          sync.RWMutex
	log         Logger
	ctx         context.Context
	logHandler  LogHandler   //日志查询handler
	middlewares []Middleware //中间件
	cancel      context.CancelFunc
}

func (e *executor) Init(opts ...Option) {
	for _, o := range opts {
		o(&e.opts)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	e.log = e.opts.l
	e.ctx = ctx
	e.cancel = cancelFunc

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

func (e *executor) Use(middlewares ...Middleware) {
	e.middlewares = middlewares
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
		Addr:         ":" + e.opts.ExecutorPort,
		WriteTimeout: time.Second * 3,
		Handler:      mux,
	}
	// 监听端口并提供服务
	e.log.Info("Starting server at " + e.address)
	go func() {
		err = server.ListenAndServe()
		log.Panic(err)
	}()
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	e.registryRemove()
	e.cancel()
	// 宽限一段时间再退出
	time.Sleep(e.opts.GracefulShutdownTime)
	return nil
}

func (e *executor) Stop() {
	e.registryRemove()
}

// RegTask 注册任务
func (e *executor) RegTask(pattern string, task TaskFunc) {
	var t = &Task{}
	t.fn = e.chain(task)
	e.regList.Set(pattern, t)
	return
}

// 运行一个任务
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
	e.log.Info("任务参数:%v", param)
	if !e.regList.Exists(param.ExecutorHandler) {
		_, _ = writer.Write(returnCall(param, FailureCode, "Task not registered"))
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
			_, _ = writer.Write(returnCall(param, FailureCode, "There are tasks running"))
			e.log.Error("任务[" + Int64ToStr(param.JobID) + "]已经在运行了:" + param.ExecutorHandler)
			return
		}
	}

	cxt := e.ctx
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

	go func(runTask *Task) {
		runTask.Run(func(code int64, msg string) {
			e.callback(runTask, code, msg)
		})
	}(task)

	e.log.Info("任务[" + Int64ToStr(param.JobID) + "]开始执行:" + param.ExecutorHandler)
	_, _ = writer.Write(returnGeneral())
}

// 删除一个任务
func (e *executor) killTask(writer http.ResponseWriter, request *http.Request) {
	e.mu.Lock()
	defer e.mu.Unlock()
	req, _ := io.ReadAll(request.Body)
	param := &killReq{}
	_ = json.Unmarshal(req, &param)
	if !e.runList.Exists(Int64ToStr(param.JobID)) {
		_, _ = writer.Write(returnKill(param, SuccessCode))
		e.log.Error("任务[" + Int64ToStr(param.JobID) + "]没有运行")
		return
	}
	task := e.runList.Get(Int64ToStr(param.JobID))
	task.Cancel()
	e.runList.Del(Int64ToStr(param.JobID))
	_, _ = writer.Write(returnGeneral())
}

// 任务日志
func (e *executor) taskLog(writer http.ResponseWriter, request *http.Request) {
	var res *LogRes
	data, err := io.ReadAll(request.Body)
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
	//e.log.Info("日志请求参数:%+v", req)
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
	defer request.Body.Close()
	req, _ := io.ReadAll(request.Body)
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
	e.log.Info("忙碌检测任务参数:%v", param)
	_, _ = writer.Write(returnGeneral())
}

// 注册执行器到调度中心
//
//goland:noinspection HttpUrlsUsage
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
	registed := false
	for {
		select {
		case <-t.C:
			t.Reset(time.Second * time.Duration(20)) //20秒心跳防止过期
			func() {
				result, err := e.post("/api/registry", string(param))
				if err != nil {
					registed = false
					e.log.Error("执行器注册失败1:" + err.Error())
					return
				}
				defer result.Body.Close()
				body, err := io.ReadAll(result.Body)
				if err != nil {
					registed = false
					e.log.Error("执行器注册失败2:" + err.Error())
					return
				}
				res := &res{}
				_ = json.Unmarshal(body, &res)
				if res.Code != SuccessCode {
					registed = false
					e.log.Error("执行器注册失败3:" + string(body))
					return
				}
				if !registed {
					e.log.Info("执行器注册成功:" + string(body))
					registed = true
				}
			}()
		case <-e.ctx.Done():
			t.Stop()
			e.log.Info("执行器停止注册")
			return
		}

	}
}

// 执行器注册摘除
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
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	e.log.Info("执行器摘除成功:" + string(body))
}

// 回调任务列表
// callback 是 executor 的一个方法，用于处理任务的回调逻辑。
// 它从 runList 中删除任务，构造回调数据，并尝试通过 HTTP POST 方法将结果回调到某个接口。
// 如果回调失败，它将使用退避策略重试。如果所有重试都失败，它将记录错误并返回。
// 参数:
//
//	task - 执行回调的任务对象。
//	code - 回调的状态码。
//	msg - 回调的消息。
func (e *executor) callback(task *Task, code int64, msg string) {
	// 从 runList 中删除任务。
	e.runList.Del(Int64ToStr(task.Id))

	// 构造回调的数据。
	callbackBody := string(returnCall(task.Param, code, msg))

	// 创建一个退避策略对象，用于处理回调失败时的重试逻辑。
	retry := backoff.New(context.Background(), backoff.Config{
		MinBackoff: 1 * time.Second,
		MaxBackoff: 60 * time.Second,
		MaxRetries: 4,
	})

	// 初始化回调请求的响应体、响应对象和错误对象。
	var (
		body []byte
		resp *http.Response
		err  error
	)

	// 使用退避策略进行回调尝试，直到成功或达到最大重试次数。
	for {
		resp, err = e.post("/api/callback", callbackBody)
		if err != nil {
			// 如果发生错误，则使用退避策略进行等待。
			retry.Wait()
			// 如果退避策略不再进行重试，则记录错误并返回。
			if !retry.Ongoing() {
				e.log.Error("callback err : ", err.Error())
				return
			}
			continue
		}

		// 读取回调响应体。
		body, err = io.ReadAll(resp.Body)
		// 忽略关闭响应体的错误。
		_ = resp.Body.Close()
		if err != nil {
			// 如果发生错误，则使用退避策略进行等待。
			retry.Wait()
			// 如果退避策略不再进行重试，则记录错误并返回。
			if !retry.Ongoing() {
				e.log.Error("callback ReadAll err : ", err.Error())
				return
			}
			continue
		}
		break
	}

	// 回调成功后，记录日志信息。
	e.log.Info("任务ID[%v];名称[%v];LogID[%v] 回调成功:"+string(body), task.Id, task.Name, task.Param.LogID)
}

// post
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
