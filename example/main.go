package main

import (
	xxl "github.com/xxl-job/go-client"
	"github.com/xxl-job/go-client/example/task"
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
