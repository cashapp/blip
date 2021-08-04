package collect

func InternalLevelPlan() Plan {
	return Plan{
		internal: true,
		Name:     "blip",
		Levels: map[string]Level{
			"key-performance-indicators": Level{
				Name: "key-performance-indicators",
				Freq: "1s",
				Collect: map[string]Domain{
					"var.global": {
						Name: "var.global",
						Metrics: []string{
							"read_only",
						},
					},
				},
			},
			"sysvars": Level{
				Name: "sysvars",
				Freq: "5s",
				Collect: map[string]Domain{
					"var.global": {
						Name: "var.global",
						Metrics: []string{
							"max_connections",
						},
					},
				},
			},
		},
	}
}
