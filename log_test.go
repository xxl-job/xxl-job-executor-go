package xxl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRotateLog(t *testing.T) {

	// 创建测试用例目录
	testCases := []struct {
		dirName string // 目录名
		age     int    // 目录年龄（天）
		expect  bool   // 期望是否被删除
	}{
		{time.Now().AddDate(0, 0, -31).Format("2006-01-02"), 31, true},  // 31天前，应该被删除
		{time.Now().AddDate(0, 0, -15).Format("2006-01-02"), 15, false}, // 15天前，不应该被删除
		{time.Now().Format("2006-01-02"), 0, false},                     // 今天，不应该被删除
	}

	// 创建测试目录
	for _, tc := range testCases {
		dirPath := filepath.Join(logDir, tc.dirName)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("Failed to create test directory %s: %v", dirPath, err)
		}
		// 在目录中创建一个测试文件
		testFile := filepath.Join(dirPath, "test.log")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", testFile, err)
		}
	}

	// 创建上下文和取消函数
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 启动日志轮转（使用较短的时间间隔进行测试）
	go RotateLog(ctx, 500*time.Millisecond, 30) // 30天的保留期

	// 等待日志轮转执行
	time.Sleep(1 * time.Second)

	// 验证结果
	for _, tc := range testCases {
		dirPath := filepath.Join(logDir, tc.dirName)
		_, err := os.Stat(dirPath)
		exists := !os.IsNotExist(err)

		if exists == tc.expect {
			t.Errorf("Directory %s: expected existence = %v, got %v", tc.dirName, !tc.expect, exists)
		}
	}
}

func TestLogf(t *testing.T) {

	// 创建测试用例
	testTime := time.Now()
	logger := NewFileLogger(testTime, 123)
	defer os.RemoveAll(logger.getLogDir())

	tests := []struct {
		name   string
		format string
		args   []interface{}
		want   string
	}{
		{
			name:   "simple string",
			format: "test message",
			args:   nil,
			want:   "test message\n",
		},
		{
			name:   "formatted string",
			format: "test %s %d",
			args:   []interface{}{"message", 123},
			want:   "test message 123\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger.Logf(tt.format, tt.args...)

			// 读取写入的日志文件
			logFile := filepath.Join(logDir, testTime.Format(dirDateLayout), "123.log")
			content, err := os.ReadFile(logFile)
			if err != nil {
				t.Errorf("Failed to read log file: %v", err)
				return
			}

			// 证日志内容
			if !strings.Contains(string(content), tt.want) {
				t.Errorf("Logf() = %v, want %v", string(content), tt.want)
			}
		})
	}

}

func TestFileLogger_readLog(t *testing.T) {
	// 创建临时测试目录
	tmpDir := "./xxl-executor-logs/2024-03-20"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll("./xxl-executor-logs/2024-03-20") // 测试结束后清理

	// 创建测试文件并写入内容
	testLogId := int64(12345)
	testContent := "第1行\n第2行\n第3行\n第4行\n第5行\n"
	testFile := filepath.Join(tmpDir, fmt.Sprintf("%d.log", testLogId))
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	tests := []struct {
		name        string
		logger      *FileLogger
		fromLineNum int
		want        LogResContent
	}{
		{
			name: "正常读取-从第一行开始",
			logger: &FileLogger{
				logId:    testLogId,
				execTime: time.Date(2024, 3, 20, 0, 0, 0, 0, time.Local),
			},
			fromLineNum: 1,
			want: LogResContent{
				FromLineNum: 1,
				ToLineNum:   5,
				LogContent:  testContent,
				IsEnd:       false,
			},
		},
		{
			name: "正常读取-从第三行开始",
			logger: &FileLogger{
				logId:    testLogId,
				execTime: time.Date(2024, 3, 20, 0, 0, 0, 0, time.Local),
			},
			fromLineNum: 3,
			want: LogResContent{
				FromLineNum: 3,
				ToLineNum:   5,
				LogContent:  "第3行\n第4行\n第5行\n",
				IsEnd:       false,
			},
		},
		{
			name: "文件不存在",
			logger: &FileLogger{
				logId:    99999,
				execTime: time.Date(2024, 3, 20, 0, 0, 0, 0, time.Local),
			},
			fromLineNum: 1,
			want: LogResContent{
				FromLineNum: 1,
				ToLineNum:   0,
				LogContent:  "readLog fail, logFile not exists",
				IsEnd:       true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.logger.readLog(tt.fromLineNum)

			if got.FromLineNum != tt.want.FromLineNum {
				t.Errorf("FromLineNum = %v, 期望 %v", got.FromLineNum, tt.want.FromLineNum)
			}
			if got.ToLineNum != tt.want.ToLineNum {
				t.Errorf("ToLineNum = %v, 期望 %v", got.ToLineNum, tt.want.ToLineNum)
			}
			if got.LogContent != tt.want.LogContent {
				t.Errorf("LogContent = %v, 期望 %v", got.LogContent, tt.want.LogContent)
			}
			if got.IsEnd != tt.want.IsEnd {
				t.Errorf("IsEnd = %v, 期望 %v", got.IsEnd, tt.want.IsEnd)
			}
		})
	}
}

func TestFileLogger_ConcurrentWrite(t *testing.T) {
	// 创建临时测试目录
	tmpDir := "./xxl-executor-logs/2024-03-20"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll("./xxl-executor-logs/2024-03-20")

	logger := NewFileLogger(time.Date(2024, 3, 20, 0, 0, 0, 0, time.Local), 12345)
	logger.Prefix = "[TEST]"

	// 测试参数
	const (
		goroutineCount = 100  // 并发goroutine数量
		writeCount     = 1000 // 每个goroutine写入次数
	)

	// 使用WaitGroup等待所有goroutine完成
	var wg sync.WaitGroup
	startTime := time.Now()

	// 启动多个goroutine并发写入
	for i := 0; i < goroutineCount; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < writeCount; j++ {
				logger.Logf("Goroutine-%d 写入第 %d 条日志", routineID, j)
			}
		}(i)
	}

	// 等待所有goroutine完成
	wg.Wait()
	duration := time.Since(startTime)

	// 验证写入的日志内容
	logContent := logger.readLog(1)

	// 计算性能指标
	totalWrites := goroutineCount * writeCount
	writePerSecond := float64(totalWrites) / duration.Seconds()

	t.Logf("性能测试结果:")
	t.Logf("总耗时: %v", duration)
	t.Logf("总写入次数: %d", totalWrites)
	t.Logf("每秒写入次数: %.2f", writePerSecond)
	t.Logf("日志行数: %d", logContent.ToLineNum)

	// 验证写入的日志行数是否正确
	expectedLines := goroutineCount * writeCount
	if logContent.ToLineNum != expectedLines {
		t.Errorf("日志行数不匹配, 期望 %d 行, 实际 %d 行", expectedLines, logContent.ToLineNum)
	}
}

// 基准测试
func BenchmarkFileLogger_Write(b *testing.B) {
	// 创建临时测试目录
	tmpDir := "./xxl-executor-logs/2024-03-20"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		b.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll("./xxl-executor-logs/2024-03-20")

	logger := NewFileLogger(time.Date(2024, 3, 20, 0, 0, 0, 0, time.Local), 12345)
	logger.Prefix = "[BENCH]"

	b.ResetTimer() // 重置计时器，不计算设置时间

	// 并发基准测试
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Logf("基准测试日志消息-%d", time.Now().UnixNano())
		}
	})
}

// 测试不同大小的日志消息
func TestFileLogger_DifferentMessageSizes(t *testing.T) {
	// 创建临时测试目录
	tmpDir := "./xxl-executor-logs/2024-03-20"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll("./xxl-executor-logs/2024-03-20")

	logger := NewFileLogger(time.Date(2024, 3, 20, 0, 0, 0, 0, time.Local), 12345)

	tests := []struct {
		name        string
		messageSize int
		writeCount  int
	}{
		{"小消息(100字节)", 100, 1000},
		{"中等消息(1KB)", 1024, 100},
		{"大消息(10KB)", 10 * 1024, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 生成指定大小的消息
			message := strings.Repeat("A", tt.messageSize)

			startTime := time.Now()
			for i := 0; i < tt.writeCount; i++ {
				logger.Logf("%s-%d", message, i)
			}
			duration := time.Since(startTime)

			bytesWritten := tt.messageSize * tt.writeCount
			throughput := float64(bytesWritten) / duration.Seconds() / 1024 / 1024 // MB/s

			t.Logf("消息大小: %d 字节", tt.messageSize)
			t.Logf("写入次数: %d", tt.writeCount)
			t.Logf("总耗时: %v", duration)
			t.Logf("吞吐量: %.2f MB/s", throughput)
		})
	}
}
