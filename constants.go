package xxl

// 响应码
const (
	SuccessCode = 200
	FailureCode = 500
)

type ScheduleType string

const (
	FixRate ScheduleType = "FIX_RATE"
	Cron    ScheduleType = "CRON"
)

type MisfireStrategy string

const (
	DoNothing   MisfireStrategy = "DO_NOTHING"    //  什么都不做
	FireOnceNow MisfireStrategy = "FIRE_ONCE_NOW" // 立即执行
)

type ExecutorBlockStrategy string

const (
	SerialExecution ExecutorBlockStrategy = "SERIAL_EXECUTION" // 串行
	DiscardLater    ExecutorBlockStrategy = "DISCARD_LATER"    // 丢弃后续调度
	CoverEarly      ExecutorBlockStrategy = "COVER_EARLY"      // 覆盖之前调度
)

type TriggerStatus int32

const (
	Total   TriggerStatus = -1 // 全部
	Stop    TriggerStatus = 0  // 停止
	Running TriggerStatus = 1  // 运行
)

type ExecutorRouteStrategy string

const (
	First  ExecutorRouteStrategy = "FIRST"  // 第一个
	Last   ExecutorBlockStrategy = "LAST"   // 最后一个
	Round  ExecutorBlockStrategy = "ROUND"  // 轮询
	Random ExecutorRouteStrategy = "RANDOM" // 随机
)
