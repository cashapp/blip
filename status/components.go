package status

const (
	SERVER = "server"

	MONITOR     = "monitor"
	MONITOR_DSN = "dsn"

	PLAN_CHANGER         = "plan-changer"
	PLAN_CHANGER_STATE   = "state"
	PLAN_CHANGER_PENDING = "state-pending"

	LEVEL_COLLECTOR   = "level-collector"
	LEVEL_PLAN        = "level-plan"
	LEVEL_STATE       = "level-state"
	LEVEL_COLLECT     = "level-collect"
	LEVEL_SINKS       = "level-sinks"
	LEVEL_CHANGE_PLAN = "level-change-plan"

	ENGINE_COLLECT = "engine-collect"
	ENGINE_PREPARE = "engine-prepare"
	ENGINE_PLAN    = "engine-plan"

	HEARTBEAT_READER = "heartbeat-reader"
	HEARTBEAT_WRITER = "heartbeat-writer"
)
