package xxl

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestCtx(t *testing.T) {
	// 创建一个根上下文
	parentCtx, cancel := context.WithCancel(context.Background())

	// 派生子上下文
	childCtx, childCancel := context.WithCancel(parentCtx)
	defer childCancel() // 避免资源泄露

	go func() {
		<-childCtx.Done()
		fmt.Println("子上下文被取消")
	}()

	// 取消父上下文
	time.Sleep(1 * time.Second)
	fmt.Println("取消父上下文")
	cancel()

	// 确认子上下文是否被取消
	time.Sleep(1 * time.Second)
	if err := childCtx.Err(); err != nil {
		fmt.Printf("子上下文错误: %v\n", err)
	}

}
