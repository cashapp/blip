package collect

func InternalLevelPlan() Plan {
	return Plan{
		internal: true,

		Name: "Internal Level Plan",
		Levels: map[string]Level{
			"key-performance-indicators": Level{
				Name: "key-performance-indicators",
				Freq: "5s",
				Collect: map[string]Domain{
					"global.status": {
						Name: "global.status",
						Metrics: []string{
							"queries",
							"threads_running",
						},
					},
				},
			},
		},
	}
}
