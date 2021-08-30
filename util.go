package xxl

import (
	"encoding/json"
	"strconv"
)

// Int64ToStr int64 to str
func Int64ToStr(i int64) string {
	return strconv.FormatInt(i, 10)
}

//执行任务回调
func returnCall(req *RunReq, code int64, msg string) []byte {
	data := call{
		&callElement{
			LogID:      req.LogID,
			LogDateTim: req.LogDateTime,
			ExecuteResult: &ExecuteResult{
				Code: code,
				Msg:  msg,
			},
			HandleCode: int(code),
			HandleMsg:  msg,
		},
	}
	str, _ := json.Marshal(data)
	return str
}

//杀死任务返回
func returnKill(req *killReq, code int64) []byte {
	msg := ""
	if code != SuccessCode {
		msg = "Task kill err"
	}
	return generateRetBytes(code, msg)
}

//忙碌返回
func returnIdleBeat(code int64) []byte {
	msg := ""
	if code != SuccessCode {
		msg = "Task is busy"
	}
	return generateRetBytes(code, msg)
}

//通用返回
func returnGeneral() []byte {
	return generateRetBytes(SuccessCode, "")
}

func generateRetBytes(code int64, msg string) []byte {
	data := &res{
		Code: code,
		Msg:  msg,
	}
	bytes, _ := json.Marshal(data)
	return bytes
}
