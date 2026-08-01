package report

import (
	"fmt"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func addObjectCharts(b *chartBuilder, run *schema.RunResult, compare compareData, lbl WorkloadLabels) {
	if !lbl.IsObject {
		return
	}
	ivs := run.Results.Intervals
	ivLabels := intervalLabels(ivs)

	opTitle := strings.ToUpper(lbl.Operation)
	if opTitle == "" {
		opTitle = "S3"
	}

	b.add("S3 operations", false, ChartPanel{ID: "s3OpsRateChart", Title: lbl.OpsRate + " by node"},
		chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{{Label: lbl.OpsRate, Data: compare.iops}}})
	b.add("S3 operations", false, ChartPanel{ID: "s3ThroughputChart", Title: "S3 throughput (MB/s)"},
		chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{{Label: "MB/s", Data: compare.mbps}}})

	if objectHasPutGet(run, lbl) {
		b.add("S3 operations", false, ChartPanel{ID: "s3PutGetChart", Title: "PUT vs GET (" + lbl.OpsUnit + ")"},
			chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{
				{Label: lbl.WriteOp, Data: compare.writeIOPS},
				{Label: lbl.ReadOp, Data: compare.readIOPS},
			}})
		b.add("S3 operations", false, ChartPanel{ID: "s3PutGetMbpsChart", Title: "PUT vs GET (MB/s)"},
			chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{
				{Label: "PUT MB/s", Data: compare.writeMBps},
				{Label: "GET MB/s", Data: compare.readMBps},
			}})
	} else if lbl.Operation == "put" || run.Results.WriteIOPS > 0 {
		b.add("S3 operations", false, ChartPanel{ID: "s3PutChart", Title: "PUT throughput (" + lbl.OpsUnit + ")"},
			chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{{Label: lbl.WriteOp, Data: compare.writeIOPS}}})
	} else if lbl.Operation == "get" || run.Results.ReadIOPS > 0 {
		b.add("S3 operations", false, ChartPanel{ID: "s3GetChart", Title: "GET throughput (" + lbl.OpsUnit + ")"},
			chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{{Label: lbl.ReadOp, Data: compare.readIOPS}}})
	}

	if compare.hasOps {
		b.add("S3 operations", false, ChartPanel{ID: "s3OpsSecChart", Title: "Object operations/sec"},
			chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{{Label: "Ops/s", Data: compare.ops}}})
	}

	if len(ivs) > 0 {
		b.add("S3 operations — over time", false, ChartPanel{ID: "s3OpsIntervalChart", Title: lbl.OpsRate + " over time"},
			chartSpec{Kind: "line", Labels: ivLabels, Datasets: []chartDataset{
				{Label: lbl.OpsRate, Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.IOPS })},
			}})
		b.add("S3 operations — over time", false, ChartPanel{ID: "s3MbpsIntervalChart", Title: "Throughput (MB/s) over time"},
			chartSpec{Kind: "line", Labels: ivLabels, Datasets: []chartDataset{
				{Label: "MB/s", Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.ThroughputMBps })},
			}})
		if objectHasPutGet(run, lbl) || ivHasReadWrite(ivs) {
			b.add("S3 operations — over time", false, ChartPanel{ID: "s3PutGetIntervalChart", Title: "PUT vs GET over time"},
				chartSpec{Kind: "line", Labels: ivLabels, Datasets: []chartDataset{
					{Label: lbl.WriteOp, Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.WriteIOPS })},
					{Label: lbl.ReadOp, Data: intervalField(ivs, func(iv schema.IntervalSample) float64 { return iv.ReadIOPS })},
				}})
		}
	}

	_ = opTitle
}

func ivHasReadWrite(ivs []schema.IntervalSample) bool {
	for _, iv := range ivs {
		if iv.ReadIOPS > 0 && iv.WriteIOPS > 0 {
			return true
		}
	}
	return false
}

func applyChartLabels(b *chartBuilder, lbl WorkloadLabels) {
	if !lbl.IsObject {
		return
	}
	rename := map[string]string{
		"Throughput (IOPS)":        lbl.OpsRate,
		"Read vs write IOPS":       fmt.Sprintf("%s vs %s", lbl.ReadOp, lbl.WriteOp),
		"IOPS by node":             lbl.OpsRate + " by node",
		"IOPS share (%)":           lbl.OpsRate + " share (%)",
		"Avg IOPS — clients vs targets": "Avg " + lbl.OpsRate + " — clients vs targets",
		"IOPS":                     lbl.OpsRate,
		"Total throughput (records/s)": "Total throughput (" + lbl.ThroughputRec + ")",
	}
	for gi := range b.groups {
		for pi := range b.groups[gi].Panels {
			if t, ok := rename[b.groups[gi].Panels[pi].Title]; ok {
				b.groups[gi].Panels[pi].Title = t
			}
		}
	}
	for id, spec := range b.specs {
		for di := range spec.Datasets {
			switch spec.Datasets[di].Label {
			case "IOPS":
				spec.Datasets[di].Label = lbl.OpsRate
			case "Read IOPS":
				spec.Datasets[di].Label = lbl.ReadOp
			case "Write IOPS":
				spec.Datasets[di].Label = lbl.WriteOp
			case "Total IOPS":
				spec.Datasets[di].Label = "Total " + lbl.OpsRate
			case "Records/s":
				spec.Datasets[di].Label = lbl.ThroughputRec
			case "% of cluster IOPS":
				spec.Datasets[di].Label = "% of cluster " + lbl.OpsRate
			}
		}
		b.specs[id] = spec
	}
}
