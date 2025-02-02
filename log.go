package xxl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LogFunc 应用日志
type LogFunc func(req LogReq, res *LogRes) []byte

// Logger 系统日志
type Logger interface {
	Info(format string, a ...interface{})
	Error(format string, a ...interface{})
}

type logger struct {
}

func (l *logger) Info(format string, a ...interface{}) {
	fmt.Println(fmt.Sprintf(format, a...))
}

func (l *logger) Error(format string, a ...interface{}) {
	log.Println(fmt.Sprintf(format, a...))
}

const (
	// 日志目录
	logDir        = "./xxl-executor-logs/"
	dirDateLayout = "2006-01-02"
)

type FileLogger struct {
	logId    int64
	execTime time.Time
	mu       sync.Mutex
	Prefix   string
}

// RotateLog 日志轮转，
func RotateLog(ctx context.Context, duration time.Duration, rotateDay int) {

	// 定时执行
	ticker := time.NewTicker(duration)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 从logDir 目录下获取所有二级目录
			dirs, err := os.ReadDir(logDir)
			if err != nil {
				log.Printf("error read dir %s; err:%v", logDir, err)
				continue
			}
			for _, dir := range dirs {
				dirName := dir.Name()
				// 判断是否是目录
				if !dir.IsDir() {
					continue
				}
				dirTime, err := time.Parse(dirDateLayout, dirName)
				// 如果dirTime 超过30天，则删除该目录
				if time.Now().Sub(dirTime) > time.Duration(rotateDay)*24*time.Hour {
					// 使用完整路径删除目录
					fullPath := filepath.Join(logDir, dirName)
					if err = os.RemoveAll(fullPath); err != nil {
						log.Printf("error remove dir %s; err:%v", fullPath, err)
					}
				}
			}
		}
	}
}

// NewFileLogger 创建一个新的 FileLogger 实例
//
// 参数：
//
//	datetime time.Time: 日志记录的时间
//	logId int64: 日志的唯一标识符
//
// 返回值：
//
//	*FileLogger: 指向新创建的 FileLogger 实例的指针
func NewFileLogger(datetime time.Time, logId int64) *FileLogger {
	flogger := &FileLogger{
		logId:    logId,
		execTime: datetime,
	}
	flogger.init()
	return flogger
}

// init 方法用于初始化 FileLogger
func (l *FileLogger) init() {
	dir := l.getLogDir()
	// 创建日志目录
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, os.ModePerm); err != nil {
			// 创建目录失败
			panic(err)
		}
	}
}

// getLogDir 获取日志目录
//   - string: 日志目录的路径，格式为 logDir/日期
func (l *FileLogger) getLogDir() string {
	// 日期格式化 作为二级目录， logId 作为文件名
	date := l.execTime.Format(dirDateLayout)

	// 二级目录
	return fmt.Sprintf("%s/%s", logDir, date)
}

// Logf 方法将日志内容写入到文件中。
// 参数 format 是一个格式化字符串，a 是一个可变参数列表，
// 表示要插入到格式化字符串中的参数。
func (l *FileLogger) Logf(format string, a ...interface{}) {
	// 测试用例在 log_test.go 中
	logContent := fmt.Sprintf(format+"\n", a...)

	_, err := l.Write([]byte(logContent))
	if err != nil {
		log.Printf("error write log  err:%v", err)
	}
}

// Write 方法将日志内容写入到文件中。
func (l *FileLogger) Write(buf []byte) (n int, err error) {
	// 加锁
	l.mu.Lock()
	defer l.mu.Unlock()

	// 获取日志文件名
	fileName := l.getLogFileName()

	// 写入文件
	// 打开文件，如果文件不存在则创建，以追加和只写模式打开，权限为0644
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		// 如果打开文件失败，则记录错误日志
		return 0, fmt.Errorf("error open file %s; err:%v", fileName, err)
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file) // 关闭文件

	// 往buf前插入prefix
	buf = append([]byte(l.Prefix), buf...)
	return file.Write(buf)
}

// getLogFileName 函数根据日志记录器实例的时间戳和日志ID生成日志文件名，并返回该文件名。
//
//	string: 生成的日志文件名
func (l *FileLogger) getLogFileName() string {
	dir := l.getLogDir()
	// 文件名
	fileName := fmt.Sprintf("%s/%d.log", dir, l.logId)
	return fileName
}

// readLog 从指定的行号开始读取日志，并返回读取的日志内容
//
// 参数:
//
//	fromLineNum: 从该行号开始读取日志
//
// 返回值:
//
//	LogResContent: 包含读取的日志内容的结构体
//	    FromLineNum: 开始读取的行号
//	    ToLineNum: 读取结束的行号
//	    LogContent: 读取的日志内容
//	    IsEnd: 是否读取到文件末尾
func (l *FileLogger) readLog(fromLineNum int) LogResContent {
	logFileName := l.getLogFileName()
	// 验证文件名
	if logFileName == "" {
		return LogResContent{
			FromLineNum: fromLineNum,
			ToLineNum:   0,
			LogContent:  "readLog fail, logFile not found",
			IsEnd:       true,
		}
	}

	// 检查文件是否存在
	if _, err := os.Stat(logFileName); os.IsNotExist(err) {
		return LogResContent{
			FromLineNum: fromLineNum,
			ToLineNum:   0,
			LogContent:  "readLog fail, logFile not exists",
			IsEnd:       true,
		}
	}

	// 打开文件
	file, err := os.Open(logFileName)
	if err != nil {
		return LogResContent{
			FromLineNum: fromLineNum,
			ToLineNum:   0,
			LogContent:  fmt.Sprintf("readLog fail, %v", err),
			IsEnd:       true,
		}
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	// 创建带缓冲的读取器
	reader := bufio.NewReader(file)
	var contentBuilder strings.Builder
	lineNum := 0

	// 逐行读取
	for {
		line, err := reader.ReadString('\n')

		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("error reading file: %v", err)
			break
		}

		lineNum++
		if lineNum >= fromLineNum {
			contentBuilder.WriteString(line)
		}

	}

	return LogResContent{
		FromLineNum: fromLineNum,
		ToLineNum:   lineNum,
		LogContent:  contentBuilder.String(),
		IsEnd:       false,
	}
}
