package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
)

const (
	helmChartPath = ".."
	helmChartName = "steward"
)

func render(t *testing.T, template string, values map[string]string) string {
	t.Helper()
	options := &helm.Options{
		SetValues: values,
	}
	templates := []string{template}
	return helm.RenderTemplate(
		t, options, helmChartPath, helmChartName,
		templates)
}
