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

func PromPlan() Plan {
	return Plan{
		internal: true,
		Name:     "mysqld_exporter",
		Levels: map[string]Level{
			"all": Level{
				Name: "all",
				Freq: "", // none, pulled/scaped on demand
				Collect: map[string]Domain{
					"status.global": {
						Name: "status.global",
						Options: map[string]string{
							"all": "yes",
						},
					},
					"var.global": {
						Name: "var.global",
						Options: map[string]string{
							"all": "yes",
						},
					},
					"innodb": {
						Name: "innodb",
						Options: map[string]string{
							"all": "enabled",
						},
					},
				},
			},
		},
	}
}
