# xxl-job go-client
xxl-job golang 客户端(功能完善中...)

## Example
```
package main

import (
	xxl "github.com/xxl-job/go-client"
	"github.com/xxl-job/go-client/example/task"
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
	exec.Run()
}

```
# see
github.com/xxl-job/go-client/example/