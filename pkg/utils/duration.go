package utils

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Metav1Duration converts a time.Duration to a metav1.Duration pointer
func Metav1Duration(d time.Duration) *metav1.Duration {
	return &metav1.Duration{Duration: d}
}

// IsZeroDuration returns true if duration is nil or duration is zero seconds
func IsZeroDuration(d *metav1.Duration) bool {
	return d == nil || d.Truncate(time.Second) == 0
}
