package xxl

import "time"

//通用响应
type res struct {
	Code int64       `json:"code"` // 200 表示正常、其他失败
	Msg  interface{} `json:"msg"`  // 错误提示消息
}

/*****************  上行参数  *********************/

// Registry 注册参数
type Registry struct {
	RegistryGroup string `json:"registryGroup"`
	RegistryKey   string `json:"registryKey"`
	RegistryValue string `json:"registryValue"`
}

//执行器执行完任务后，回调任务结果时使用
type call []*callElement

type callElement struct {
	LogID         int64          `json:"logId"`
	LogDateTim    int64          `json:"logDateTim"`
	ExecuteResult *ExecuteResult `json:"executeResult"`
	//以下是7.31版本 v2.3.0 Release所使用的字段
	HandleCode int    `json:"handleCode"` //200表示正常,500表示失败
	HandleMsg  string `json:"handleMsg"`
}

// ExecuteResult 任务执行结果 200 表示任务执行正常，500表示失败
type ExecuteResult struct {
	Code int64       `json:"code"`
	Msg  interface{} `json:"msg"`
}

/*****************  下行参数  *********************/

//阻塞处理策略
const (
	serialExecution = "SERIAL_EXECUTION" //单机串行
	discardLater    = "DISCARD_LATER"    //丢弃后续调度
	coverEarly      = "COVER_EARLY"      //覆盖之前调度
)

// RunReq 触发任务请求参数
type RunReq struct {
	JobID                 int64  `json:"jobId"`                 // 任务ID
	ExecutorHandler       string `json:"executorHandler"`       // 任务标识
	ExecutorParams        string `json:"executorParams"`        // 任务参数
	ExecutorBlockStrategy string `json:"executorBlockStrategy"` // 任务阻塞策略
	ExecutorTimeout       int64  `json:"executorTimeout"`       // 任务超时时间，单位秒，大于零时生效
	LogID                 int64  `json:"logId"`                 // 本次调度日志ID
	LogDateTime           int64  `json:"logDateTime"`           // 本次调度日志时间
	GlueType              string `json:"glueType"`              // 任务模式，可选值参考 com.xxl.job.core.glue.GlueTypeEnum
	GlueSource            string `json:"glueSource"`            // GLUE脚本代码
	GlueUpdatetime        int64  `json:"glueUpdatetime"`        // GLUE脚本更新时间，用于判定脚本是否变更以及是否需要刷新
	BroadcastIndex        int64  `json:"broadcastIndex"`        // 分片参数：当前分片
	BroadcastTotal        int64  `json:"broadcastTotal"`        // 分片参数：总分片
}

//终止任务请求参数
type killReq struct {
	JobID int64 `json:"jobId"` // 任务ID
}

//忙碌检测请求参数
type idleBeatReq struct {
	JobID int64 `json:"jobId"` // 任务ID
}

// LogReq 日志请求
type LogReq struct {
	LogDateTim  int64 `json:"logDateTim"`  // 本次调度日志时间
	LogID       int64 `json:"logId"`       // 本次调度日志ID
	FromLineNum int   `json:"fromLineNum"` // 日志开始行号，滚动加载日志
}

// LogRes 日志响应
type LogRes struct {
	Code    int64         `json:"code"`    // 200 表示正常、其他失败
	Msg     string        `json:"msg"`     // 错误提示消息
	Content LogResContent `json:"content"` // 日志响应内容
}

// LogResContent 日志响应内容
type LogResContent struct {
	FromLineNum int    `json:"fromLineNum"` // 本次请求，日志开始行数
	ToLineNum   int    `json:"toLineNum"`   // 本次请求，日志结束行号
	LogContent  string `json:"logContent"`  // 本次请求日志内容
	IsEnd       bool   `json:"isEnd"`       // 日志是否全部加载完
}

/********************任务参数*******************/

type AddJob struct {
	JobGroup               int64  `json:"jobGroup"`                // 执行器ID
	JobDesc                string `json:"jobDesc"`                 // 任务描述
	Author                 string `json:"author"`                  // 作者
	TriggerStatus          int32  `json:"triggerStatus"`           // 任务状态
	AlarmEmail             string `json:"alarmEmail"`              // 多个邮件使用逗号分隔
	ScheduleType           string `json:"scheduleType"`            // 调度类型, CRON 定时, FIX_RATE 固定速率
	ScheduleConf           string `json:"scheduleConf"`            // 必传
	ScheduleConfCron       string `json:"schedule_conf_CRON"`      // 不清楚该字段功能
	CronGenDisplay         string `json:"cronGen_display"`         // CRON类型, 与scheduleConf一致, 否则不传
	ScheduleConfFixDelay   string `json:"schedule_conf_FIX_DELAY"` // 不清楚该字段功能
	ScheduleConfFixRate    string `json:"schedule_conf_FIX_RATE"`  // FIX_RATE类型, 与scheduleConf一致, 否则不传
	GlueType               string `json:"glueType"`                // 运行模式, BEAN
	ExecutorHandler        string `json:"executorHandler"`         // JobHandler, 任务名
	ExecutorParam          string `json:"executorParam"`           // 执行参数
	ExecutorRouteStrategy  string `json:"executorRouteStrategy"`   // 路由策略, ROUND 轮询, FIRST 第一个, LAST 最后一个
	ChildJobId             string `json:"childJobId"`              // 子任务ID, 做个任务使用逗号隔开
	MisfireStrategy        string `json:"misfireStrategy"`         // 调度过期策略, DO_NOTHING 忽略, FIRE_ONCE_NOW 立即执行一次
	ExecutorBlockStrategy  string `json:"executorBlockStrategy"`   // 阻塞处理策略, SERIAL_EXECUTION 串行, DISCARD_LATER 丢弃后续调度, COVER_EARLY 覆盖之前调度
	ExecutorTimeout        int64  `json:"executorTimeout"`         // 执行超时时间, 单位: s
	ExecutorFailRetryCount int32  `json:"executorFailRetryCount"`  // 失败重试次数
	GlueRemark             string `json:"glueRemark"`              // 值为: GLUE代码初始化
	GlueSource             string `json:"glueSource"`              // 源码
}

/********************添加任务响应*******************/

type AddJobResp struct {
	Code    int         `json:"code"`
	Msg     interface{} `json:"msg"`
	Content string      `json:"content"`
}

/********************添加任务参数*******************/

type JobInfo struct {
	Id                     int       `json:"id"`
	JobGroup               int       `json:"jobGroup"`
	JobDesc                string    `json:"jobDesc"`
	AddTime                time.Time `json:"addTime"`
	UpdateTime             time.Time `json:"updateTime"`
	Author                 string    `json:"author"`
	AlarmEmail             string    `json:"alarmEmail"`
	ScheduleType           string    `json:"scheduleType"`
	ScheduleConf           string    `json:"scheduleConf"`
	MisfireStrategy        string    `json:"misfireStrategy"`
	ExecutorRouteStrategy  string    `json:"executorRouteStrategy"`
	ExecutorHandler        string    `json:"executorHandler"`
	ExecutorParam          string    `json:"executorParam"`
	ExecutorBlockStrategy  string    `json:"executorBlockStrategy"`
	ExecutorTimeout        int       `json:"executorTimeout"`
	ExecutorFailRetryCount int       `json:"executorFailRetryCount"`
	GlueType               string    `json:"glueType"`
	GlueSource             string    `json:"glueSource"`
	GlueRemark             string    `json:"glueRemark"`
	GlueUpdatetime         time.Time `json:"glueUpdatetime"`
	ChildJobId             string    `json:"childJobId"`
	TriggerStatus          int       `json:"triggerStatus"`
	TriggerLastTime        int       `json:"triggerLastTime"`
	TriggerNextTime        int       `json:"triggerNextTime"`
}

/********************查询任务参数*******************/

type QueryJob struct {
	JobGroup        int64  `json:"jobGroup"`        // 执行器ID
	TriggerStatus   int32  `json:"triggerStatus"`   // 直接使用-1
	JobDesc         string `json:"jobDesc"`         // 任务描述
	ExecutorHandler string `json:"executorHandler"` // 任务名
	Author          string `json:"author"`          // 作者
	Start           int32  `json:"start"`           // 偏移量, 默认是0
	Length          int32  `json:"length"`          // 每页数量
}

/********************运行任务参数*******************/

type RunJob struct {
	Id int64 `json:"id"`
}

/********************查询任务响应*******************/

type JobList struct {
	RecordsFiltered int       `json:"recordsFiltered"`
	Data            []JobInfo `json:"data"`
	RecordsTotal    int       `json:"recordsTotal"`
}

/********************删除任务参数*******************/

type DeleteJob struct {
	Id int64 `json:"id"`
}
