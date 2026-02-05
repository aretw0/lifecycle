package mermaid

// Option configures the rendering behavior.
type Option func(*Options)

type Options struct {
	Styles string // Custom Mermaid class definitions
}

// DefaultStyles returns the standard Mermaid class definitions for lifecycle diagrams.
func DefaultStyles() string {
	return `    classDef created fill:#f8f9fa,stroke:#dee2e6,color:#6c757d;
    classDef pending fill:#eef2ff,stroke:#c7d2fe,color:#4338ca;
    classDef starting fill:#cfe2ff,stroke:#b8d4ff,color:#004085;
    classDef running fill:#d1ecf1,stroke:#bee5eb,color:#0c5460;
    classDef suspended fill:#fff3cd,stroke:#ffe69c,color:#856404;
    classDef stopping fill:#f8d7da,stroke:#f5c6cb,color:#721c24;
    classDef stopped fill:#e9ecef,stroke:#adb5bd,color:#495057;
    classDef finished fill:#d4edda,stroke:#c3e6cb,color:#155724;
    classDef failed fill:#f8d7da,stroke:#f5c6cb,color:#721c24;
    classDef container stroke-width:3px,stroke-dasharray: 0;
    classDef process stroke-width:1px;
    classDef goroutine stroke-dasharray: 5 5;
    classDef supervisor stroke-width:2px,stroke-dasharray: 0;
    classDef signal stroke-width:2px,stroke-dasharray: 0;
    classDef active fill:#eef2ff,stroke:#4338ca,stroke-width:2px;
`
}

// WithStyles allows custom Mermaid class definitions.
func WithStyles(styles string) Option {
	return func(o *Options) {
		o.Styles = styles
	}
}
