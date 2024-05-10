package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestExporterDataSource(t *testing.T) {
	exporterDataSources := []*exporterDataSourceConfig{
		//{
		//	identifier: "datadog_metric_exporter",
		//	name:       datadogMetricExporterName,
		//},
		//{
		//	identifier: "cloudwatch_metric_exporter",
		//	name:       cloudwatchMetricExporterName,
		//},
		{
			identifier: "cloudwatch_log_exporter",
			id:         cloudwatchLogExporterID,
			name:       cloudwatchLogExporterName,
		},
	}
	testExportersSet := func() []resource.TestCheckFunc {
		checks := make([]resource.TestCheckFunc, len(exporterDataSources))
		for idx, ds := range exporterDataSources {
			checks[idx] = resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrSet(ds.fqid(), "id"),
				resource.TestCheckResourceAttr(ds.fqid(), "name", ds.name),
			)
		}
		return checks
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Steps: []resource.TestStep{
			{
				Config: exporterConfigWithProvider(t, exporterDataSources...),
				Check: resource.ComposeAggregateTestCheckFunc(
					testExportersSet()...,
				),
			},
		},
	})
}
