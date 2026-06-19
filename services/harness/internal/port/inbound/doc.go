// Package inbound declares the service's driving (inbound) ports: the use-case
// interfaces that driving adapters (CLI, MCP, events) call to enter the
// application. It is empty until the application layer defines its first use
// case — ports are added when a consumer requires them, not speculatively.
package inbound
