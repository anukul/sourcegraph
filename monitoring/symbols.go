package main

func Symbols() *Container {
	return &Container{
		Name:        "symbols",
		Title:       "Symbols",
		Description: "Handles symbol searches for unindexed branches.",
		Groups: []Group{
			{
				Title: "General",
				Rows: []Row{
					{
						{
							Name:              "store_fetch_failures",
							Description:       "store fetch failures every 5m",
							Query:             `sum(increase(symbols_store_fetch_failed[5m]))`,
							DataMayNotExist:   true,
							Warning:           Alert{GreaterOrEqual: 5},
							PanelOptions:      PanelOptions().LegendFormat("failures"),
							PossibleSolutions: "none",
						},
						{
							Name:              "current_fetch_queue_size",
							Description:       "current fetch queue size",
							Query:             `sum(symbols_store_fetch_queue_size)`,
							DataMayNotExist:   true,
							Warning:           Alert{GreaterOrEqual: 25},
							PanelOptions:      PanelOptions().LegendFormat("size"),
							PossibleSolutions: "none",
						},
					},
					{
						sharedFrontendInternalAPIErrorResponses("symbols"),
					},
				},
			},
			{
				Title:  "Golang runtime monitoring",
				Hidden: true,
				Rows: []Row{
					{
						sharedGoGoroutines("symbols"),
						sharedGoGcDuration("symbols"),
					},
				},
			},
			{
				Title:  "Container monitoring (not available on server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedContainerRestarts("symbols"),
						sharedContainerMemoryUsage("symbols"),
						sharedContainerCPUUsage("symbols"),
					},
				},
			},
			{
				Title:  "Provisioning indicators (not available on server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedProvisioningCPUUsage7d("symbols"),
						sharedProvisioningMemoryUsage7d("symbols"),
					},
					{
						sharedProvisioningCPUUsage5m("symbols"),
						sharedProvisioningMemoryUsage5m("symbols"),
					},
				},
			},
			{
				Title:  "Kubernetes monitoring (only available on k8s)",
				Hidden: true,
				Rows: []Row{
					{
						sharedKubernetesPodsAvailable("symbols"),
					},
				},
			},
		},
	}
}
