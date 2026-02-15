package lifecycle

import (
	"github.com/aretw0/introspection"

	"github.com/aretw0/lifecycle/pkg/core/signal"
	"github.com/aretw0/lifecycle/pkg/core/worker"
)

// LifecycleDiagramConfig returns an introspection DiagramConfig
// customized for lifecycle's domain terminology and rendering.
//
// This maps lifecycle concepts (Signal Context, Supervision Tree)
// to the generic introspection rendering pipeline by delegating to
// core component stylers and labelers.
func LifecycleDiagramConfig() *introspection.DiagramConfig {
	return &introspection.DiagramConfig{
		PrimaryID:          "S",
		PrimaryLabel:       "Signal Context",
		PrimaryNodeLabel:   "⚡ Lifecycle Controller",
		SecondaryID:        "root",
		SecondaryLabel:     "Supervision Tree",
		ConnectionLabel:    "governs",
		NodeStyler:         worker.NodeStyler,
		NodeLabeler:        worker.NodeLabeler,
		PrimaryNodeStyler:  signal.PrimaryStyler,
		PrimaryNodeLabeler: signal.PrimaryLabeler,
	}
}
