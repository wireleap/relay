package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wireleap/relay/api/labels"
)

// Metrics of the wireleap relay
var Metrics struct {
	Conn struct {
		Open     func(labels.Contract) prometheus.Gauge      `name:"open_count" help:"Number of TCP open connections"`
		Error    func(labels.ContractErr) prometheus.Counter `name:"error_count" help:"Number of failed TCP connections to server"`
		Total    func(labels.Contract) prometheus.Counter    `name:"count" help:"Total number of TCP connections"`
		Lifetime func() TimeHistogram                        `name:"lifetime_seconds" help:"Connection lifetime" buckets:".1,1,5,15,30,60,90,120"`
	} `namespace:"connection"`
	Net struct {
		RemainingCapBytes func(labels.ContractNetCap) prometheus.Gauge `name:"remaining_caplimit_bytes" help:"Remaining capacity bytes"`
		CapLimitsBytes    func(labels.ContractNetCap) prometheus.Gauge `name:"caplimit_bytes" help:"Capacity threshold bytes"`
		CapLimitStatus    func(labels.Contract) prometheus.Gauge       `name:"caplimit_status" help:"Capacity threshold status"`
		TotalBytes        func(labels.Connection) prometheus.Counter   `name:"total_bytes" help:"Number of rerouted bytes"`
	} `namespace:"network"`
}