package ocextra

import (
	"log"

	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

// PrintExporter is a stats and trace ocextra that logs
// the exported data to the console.
type PrintExporter struct{}

// ExportView logs the view data.
func (e *PrintExporter) ExportView(vd *view.Data) {
	log.Println(vd)
}

// ExportSpan logs the trace span.
func (e *PrintExporter) ExportSpan(vd *trace.SpanData) {
	log.Println(vd)
}
