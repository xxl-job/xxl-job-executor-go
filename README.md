# xxl-job-executor-go
很多公司java与go开发共存，java中有xxl-job做为任务调度引擎，为此也出现了go执行器(客户端)，使用起来比较简单：
# 支持
```	
1.执行器注册
2.耗时任务取消
3.任务注册，像写http.Handler一样方便
4.任务panic处理
5.阻塞策略处理
6.任务完成支持返回执行备注
7.任务超时取消 (单位：秒，0为不限制)
8.失败重试次数(在参数param中，目前由任务自行处理)
9.日志查看（未完成）
```

## Example
```
package main

import (
	xxl "github.com/xxl-job/xxl-job-executor-go"
	"github.com/xxl-job/xxl-job-executor-go/example/task"
	"log"
)

func main() {
	exec := xxl.NewExecutor(
		xxl.ServerAddr("http://127.0.0.1/xxl-job-admin"),
		xxl.AccessToken(""),         //请求令牌(默认为空)
		xxl.ExecutorIp("127.0.0.1"), //可自动获取
		xxl.ExecutorPort("9999"),    //默认9999（非必填）
		xxl.RegistryKey("golang-jobs"),
	)
	exec.Init()
	exec.RegTask("task.test", task.Test)
	exec.RegTask("task.test2", task.Test2)
	exec.RegTask("task.panic", task.Panic)
	log.Fatal(exec.Run())
}
```
# see
github.com/xxl-job/xxl-job-executor-go/example/
