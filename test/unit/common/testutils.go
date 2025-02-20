// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package common

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsstore "k8s.io/kube-state-metrics/pkg/metrics_store"
)

type GenerateMetricsTestCase struct {
	Name        string
	Obj         interface{}
	MetricNames []string
	Want        string
	Func        func(interface{}) []metricsstore.FamilyByteSlicer
}

func (testCase *GenerateMetricsTestCase) Run() error {
	metricFamilies := testCase.Func(testCase.Obj)
	metricFamilyStrings := []string{}
	for _, f := range metricFamilies {
		metricFamilyStrings = append(metricFamilyStrings, string(f.ByteSlice()))
	}

	metrics := strings.Split(strings.Join(metricFamilyStrings, ""), "\n")

	metrics = filterMetrics(metrics, testCase.MetricNames)

	out := strings.Join(metrics, "\n")

	if err := compareOutput(testCase.Want, out); err != nil {
		return fmt.Errorf("expected wanted output to equal output: %v", err.Error())
	}

	return nil
}

func compareOutput(a, b string) error {
	if a == "" && b == "" {
		return nil
	}

	if (a == "" && b != "") || (a != "" && b == "") {
		return fmt.Errorf("expected a to equal b but got:\n%v\nand:\n%v", a, b)
	}

	entities := []string{a, b}

	// Align a and b
	for i := 0; i < len(entities); i++ {
		for _, f := range []func(string) string{removeUnusedWhitespace, sortLabels, sortByLine} {
			entities[i] = f(entities[i])
		}
	}

	if entities[0] != entities[1] {
		return fmt.Errorf("expected a to equal b but got:\n%v\nand:\n%v", entities[0], entities[1])
	}

	return nil
}

// sortLabels sorts the order of labels in each line of the given metrics. The
// Prometheus exposition format does not force ordering of labels. Hence a test
// should not fail due to different metric orders.
func sortLabels(s string) string {
	// do nothing if the metric has no label
	if !strings.Contains(s, "{") {
		return s
	}
	sorted := []string{}

	for _, line := range strings.Split(s, "\n") {
		split := strings.Split(line, "{")
		if len(split) != 2 {
			panic(fmt.Sprintf("failed to sort labels in \"%v\"", line))
		}
		name := split[0]

		split = strings.Split(split[1], "}")
		value := split[1]

		labels := strings.Split(split[0], ",")
		sort.Strings(labels)

		sorted = append(sorted, fmt.Sprintf("%v{%v}%v", name, strings.Join(labels, ","), value))
	}

	return strings.Join(sorted, "\n")
}

func sortByLine(s string) string {
	split := strings.Split(s, "\n")
	sort.Strings(split)
	return strings.Join(split, "\n")
}

func filterMetrics(ms []string, names []string) []string {
	// In case the test case is based on all returned metrics, MetricNames does
	// not need to me defined.
	if names == nil {
		return ms
	}
	filtered := []string{}

	regexps := []*regexp.Regexp{}
	for _, n := range names {
		regexps = append(regexps, regexp.MustCompile(fmt.Sprintf("^%v", n)))
	}

	for _, m := range ms {
		drop := true
		for _, r := range regexps {
			if r.MatchString(m) {
				drop = false
				break
			}
		}
		if !drop {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func removeUnusedWhitespace(s string) string {
	var (
		trimmedLine  string
		trimmedLines []string
		lines        = strings.Split(s, "\n")
	)

	for _, l := range lines {
		trimmedLine = strings.TrimSpace(l)

		if len(trimmedLine) > 0 {
			trimmedLines = append(trimmedLines, trimmedLine)
		}
	}

	return strings.Join(trimmedLines, "\n")
}

func NewCondition(conditonType string, status metav1.ConditionStatus) metav1.Condition {
	return metav1.Condition{
		Type:   conditonType,
		Status: status,
	}
}

func NewConditionWithTime(conditonType string, status metav1.ConditionStatus, t time.Time) metav1.Condition {
	return metav1.Condition{
		Type:               conditonType,
		Status:             status,
		LastTransitionTime: metav1.NewTime(t),
	}
}
