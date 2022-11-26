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
	Contract struct {
		Enrolled  func(labels.Contract) prometheus.Gauge      `name:"enrollment_count" help:"Contract enrollment"`
		Error     func(labels.ContractErr) prometheus.Counter `name:"error_count" help:"Number of contract enrollment errors"`
		Heartbeat func(labels.ContractErr) prometheus.Counter `name:"heartbeat_count" help:"Number of heartbeats sent"`
	} `namespace:"contract"`
	Net struct {
		RemainingCapBytes func(labels.ContractNetCap) prometheus.Gauge `name:"remaining_caplimit_bytes" help:"Remaining capacity bytes"`
		CapLimitsBytes    func(labels.ContractNetCap) prometheus.Gauge `name:"caplimit_bytes" help:"Capacity threshold bytes"`
		CapLimitStatus    func(labels.Contract) prometheus.Gauge       `name:"caplimit_status" help:"Capacity threshold status"`
		TotalBytes        func(labels.Connection) prometheus.Counter   `name:"total_bytes" help:"Number of rerouted bytes"`
	} `namespace:"network"`
	ST struct {
		Scheduled func(labels.Contract) prometheus.Gauge      `name:"scheduled_count" help:"Number of scheduled sharetokens"`
		Submitted func(labels.ContractErr) prometheus.Counter `name:"submitted_count" help:"Number of submitted sharetokens"`
	} `namespace:"sharetoken"`
}
