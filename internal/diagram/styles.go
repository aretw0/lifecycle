package diagram

import "strings"

// Styles returns the standard Mermaid class definitions for the library.
func Styles() string {
	var sb strings.Builder
	sb.WriteString("    classDef pending fill:#fff3cd,stroke:#ffecb5,color:#856404;\n")
	sb.WriteString("    classDef running fill:#d1ecf1,stroke:#bee5eb,color:#0c5460;\n")
	sb.WriteString("    classDef stopped fill:#d4edda,stroke:#c3e6cb,color:#155724;\n")
	sb.WriteString("    classDef failed fill:#f8d7da,stroke:#f5c6cb,color:#721c24;\n")
	sb.WriteString("    classDef active fill:#d1ecf1,stroke:#0c5460,stroke-width:2px,color:#0c5460;\n")
	sb.WriteString("    classDef container stroke-width:3px,stroke-dasharray: 0;\n")
	sb.WriteString("    classDef process stroke-width:1px;\n")
	sb.WriteString("    classDef goroutine stroke-dasharray: 5 5;\n")
	return sb.String()
}
