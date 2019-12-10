package framework

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

func Bar(string) PipelineRunTest {
	return PipelineRunTest{}
}

func Baz(string) PipelineRunTest {
	return PipelineRunTest{}
}

func Test_GetTestPlanName(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		plan         TestPlan
		expectedName string
	}{
		{TestPlan{Name: "foo"}, "foo"},
		{TestPlan{Name: "foo", Count: 1}, "foo"},
		{TestPlan{Name: "foo", Count: 2}, "foo_c2_nodelay"},
		{TestPlan{Name: "foo", Count: 2, ParallelCreation: true}, "foo_c2_parallel"},
		{TestPlan{Name: "foo", Count: 3, CreationDelay: time.Second * 5}, "foo_c3_delay:5.0s"},
		{TestPlan{TestBuilder: Bar, Count: 1}, "Bar"},
		{TestPlan{TestBuilder: Bar, Count: 2}, "Bar_c2_nodelay"},
		{TestPlan{TestBuilder: Bar, Count: 2, ParallelCreation: true}, "Bar_c2_parallel"},
		{TestPlan{TestBuilder: Baz, Count: 3, CreationDelay: time.Second * 5}, "Baz_c3_delay:5.0s"},
		{TestPlan{TestBuilder: Baz, Count: 3, CreationDelay: time.Millisecond * 1500}, "Baz_c3_delay:1.5s"},
	} {
		t.Run(test.expectedName, func(t *testing.T) {
			// EXERCISE
			name := getTestPlanName(test.plan)
			//VERIFY
			assert.Equal(t, test.expectedName, name)
		})
	}
}
