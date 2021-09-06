package event

// Blip events (non-monitor)
const (
	BOOT                = "server-boot"
	BOOT_CONFIG_LOADING = "server-boot-config-loading"
	BOOT_CONFIG_LOADED  = "server-boot-config-loaded"
	BOOT_CONFIG_ERROR   = "server-boot-config-error"
	BOOT_CONFIG_INVALID = "server-boot-config-invalid"

	BOOT_PLANS_LOADING = "server-boot-plans-loading"
	BOOT_PLANS_LOADED  = "server-boot-plans-loaded"
	BOOT_PLANS_ERROR   = "server-boot-plans-error"
	BOOT_PLANS_INVALID = "server-boot-plans-invalid"

	BOOT_MONITORS_ERROR  = "server-boot-monitors-error"
	BOOT_MONITORS_LOADED = "server-boot-monitors-loaded"

	SERVER_RUN      = "server-run"
	SERVER_RUN_WAIT = "server-run-wait"
	SERVER_RUN_STOP = "server-run-stop"

	MONITOR_LOADER_LOADING = "monitor-loader-loading"

	MONITOR_PREPARE_PLAN = "monitor-prepare-plan"

	MONITOR_STOPPED = "monitor-stopped"

	RUN      = "server-run"
	SHUTDOWN = "server-shutting-down"

	REGISTER_SINK    = "register-sink"
	REGISTER_METRICS = "register-metrics"
)

// Monitor events
const (
	MONITOR_CONNECTING = "connecting"
	MONITOR_CONNECTED  = "connected"
	LPC_RUNNING        = "lpc-running"
	CHANGE_PLAN        = "change-plan"
	STATE_CHANGE_BEGIN = "state-change-begin"
	STATE_CHANGE_END   = "state-change-end"
	STATE_CHANGE_ABORT = "state-change-abort"
)
