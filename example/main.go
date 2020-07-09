package main

import (
	xxl "github.com/xxl-job/go-client"
	"github.com/xxl-job/go-client/example/task"
)

func main() {
	exec := xxl.NewExecutor(
		xxl.ServerAddr("http://127.0.0.1/xxl-job-admin"),
		xxl.ExecutorIp("127.0.0.1"),
		xxl.ExecutorPort("9999"),
	)
	exec.Init()
	exec.RegTask("task.test",task.Test)
	exec.RegTask("task.test2",task.Test2)
	exec.Run()
}
