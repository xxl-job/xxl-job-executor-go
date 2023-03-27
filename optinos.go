package xxl

import (
	"fmt"
	"github.com/go-basic/ipv4"
	"time"
)

type Options struct {
	ServerAddr   string        `json:"server_addr"`   //调度中心地址
	AccessToken  string        `json:"access_token"`  //请求令牌
	Timeout      time.Duration `json:"timeout"`       //接口超时时间
	ExecutorIp   string        `json:"executor_ip"`   //本地(执行器)IP(可自行获取)
	ExecutorPort string        `json:"executor_port"` //本地(执行器)端口
	RegistryKey  string        `json:"registry_key"`  //执行器名称
	LogDir       string        `json:"log_dir"`       //日志目录

	Storage           Storager            // 任务存储
	HandlerMap        map[string]TaskFunc // 任务函数
	l                 Logger              // 日志处理
	ConcurrentExecute bool                // 是否并发执行
}

// GetRunningTaskId 生成运行任务ID
func (o *Options) GetRunningTaskId(jobId, logId int64) string {
	if o.ConcurrentExecute {
		return fmt.Sprintf("%d-%d", jobId, logId)
	}

	return Int64ToStr(jobId)
}

func newOptions(opts ...Option) Options {
	opt := Options{
		ExecutorIp:   ipv4.LocalIP(),
		ExecutorPort: DefaultExecutorPort,
		RegistryKey:  DefaultRegistryKey,
		HandlerMap:   make(map[string]TaskFunc, 0),
		Storage:      NewSessionStorage(),
	}

	for _, o := range opts {
		o(&opt)
	}

	if opt.l == nil {
		opt.l = &logger{}
	}

	return opt
}

type Option func(o *Options)

var (
	DefaultExecutorPort = "9999"
	DefaultRegistryKey  = "golang-jobs"
)

// ServerAddr 设置调度中心地址
func ServerAddr(addr string) Option {
	return func(o *Options) {
		o.ServerAddr = addr
	}
}

// AccessToken 请求令牌
func AccessToken(token string) Option {
	return func(o *Options) {
		o.AccessToken = token
	}
}

// ExecutorIp 设置执行器IP
func ExecutorIp(ip string) Option {
	return func(o *Options) {
		o.ExecutorIp = ip
	}
}

// ExecutorPort 设置执行器端口
func ExecutorPort(port string) Option {
	return func(o *Options) {
		o.ExecutorPort = port
	}
}

// RegistryKey 设置执行器标识
func RegistryKey(registryKey string) Option {
	return func(o *Options) {
		o.RegistryKey = registryKey
	}
}

// SetLogger 设置日志处理器
func SetLogger(l Logger) Option {
	return func(o *Options) {
		o.l = l
	}
}

// SetHandlerMap 设置job处理器
func SetHandlerMap(m map[string]TaskFunc) Option {
	return func(o *Options) {
		o.HandlerMap = m
	}
}

// SetStorage 设置job处理器
func SetStorage(storage Storager) Option {
	return func(o *Options) {
		o.Storage = storage
	}
}

// SetConcurrentExecute 设置是否并发执行
func SetConcurrentExecute(concurrentExecute bool) Option {
	return func(o *Options) {
		o.ConcurrentExecute = concurrentExecute
	}
}
