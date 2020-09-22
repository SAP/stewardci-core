package errors

import (
	"errors"
	"fmt"
	"testing"

	"gotest.tools/assert"
)

func Test_Annotate(t *testing.T) {

	assertRecoverability := func(t *testing.T, result error, expected bool) {
		t.Helper()
		if ann, ok := result.(*recoverabilityAnnotation); ok {
			assert.Assert(t, ann.recoverable == expected)
		} else {
			t.Fatal("expected result to be a recoverability annotation but it's not")
		}
	}

	assertResultIsNewOrOriginal := func(t *testing.T, result, orig error, expectNew bool) {
		t.Helper()
		if result == orig && expectNew {
			t.Fatalf("expected to return a new error but got the original one")
		}
		if result != orig && !expectNew {
			t.Fatalf("expected to return the original error but got a new one")
		}
	}

	assertSameMessage := func(t *testing.T, result, orig error) {
		t.Helper()
		if result.Error() != orig.Error() {
			t.Fatalf(
				"Error() should yield the original message %q but returned %q",
				orig.Error(), result.Error(),
			)
		}
	}

	assertUnwrapYieldsOrig := func(t *testing.T, result, orig error) {
		t.Helper()
		if errors.Unwrap(result) != orig {
			t.Fatalf("Unwrap() should yield the original error but did not")
		}
	}

	err1 := fmt.Errorf("err1")

	testcasesRecoverable := []struct {
		name      string
		in        error
		expectNew bool
	}{
		{
			name:      "nil",
			in:        nil,
			expectNew: false,
		},
		{
			name:      "other",
			in:        err1,
			expectNew: true,
		},
		{
			name:      "recoverable",
			in:        Recoverable(err1),
			expectNew: false,
		},
		{
			name:      "non-recoverable",
			in:        NonRecoverable(Recoverable(err1)),
			expectNew: true,
		},
	}

	for _, test := range testcasesRecoverable {
		t.Run("Recoverable/"+test.name, func(t *testing.T) {
			test := test
			t.Parallel()

			// EXCERCISE
			result := Recoverable(test.in)

			// VERIFY
			assertResultIsNewOrOriginal(t, result, test.in, test.expectNew)
			if test.expectNew {
				assertRecoverability(t, result, true)
				assertSameMessage(t, result, test.in)
				assertUnwrapYieldsOrig(t, result, test.in)
			}
			assert.Assert(t, errors.Is(result, test.in) == true)
		})
	}

	testcasesNonRecoverable := []struct {
		name      string
		in        error
		expectNew bool
	}{
		{
			name:      "nil",
			in:        nil,
			expectNew: false,
		},
		{
			name:      "other",
			in:        err1,
			expectNew: false,
		},
		{
			name:      "recoverable",
			in:        Recoverable(err1),
			expectNew: true,
		},
		{
			name:      "non-recoverable",
			in:        NonRecoverable(Recoverable(err1)),
			expectNew: false,
		},
	}

	for _, test := range testcasesNonRecoverable {
		t.Run("NonRecoverable/"+test.name, func(t *testing.T) {
			test := test
			t.Parallel()

			// EXCERCISE
			result := NonRecoverable(test.in)

			// VERIFY
			assertResultIsNewOrOriginal(t, result, test.in, test.expectNew)
			if test.expectNew {
				assertRecoverability(t, result, false)
				assertSameMessage(t, result, test.in)
				assertUnwrapYieldsOrig(t, result, test.in)
			}
			assert.Assert(t, errors.Is(result, test.in) == true)
		})
	}

	for _, test := range testcasesRecoverable {
		t.Run("RecoverableIf/true/"+test.name, func(t *testing.T) {
			test := test
			t.Parallel()

			// EXCERCISE
			result := RecoverableIf(test.in, true)

			// VERIFY
			assertResultIsNewOrOriginal(t, result, test.in, test.expectNew)
			if test.expectNew {
				assertRecoverability(t, result, true)
				assertSameMessage(t, result, test.in)
				assertUnwrapYieldsOrig(t, result, test.in)
			}
			assert.Assert(t, errors.Is(result, test.in) == true)
		})
	}

	for _, test := range testcasesNonRecoverable {
		t.Run("RecoverableIf/false/"+test.name, func(t *testing.T) {
			test := test
			t.Parallel()

			// EXCERCISE
			result := RecoverableIf(test.in, false)

			// VERIFY
			assertResultIsNewOrOriginal(t, result, test.in, test.expectNew)
			if test.expectNew {
				assertRecoverability(t, result, false)
				assertSameMessage(t, result, test.in)
				assertUnwrapYieldsOrig(t, result, test.in)
			}
			assert.Assert(t, errors.Is(result, test.in) == true)
		})
	}
}

func Test_IsRecoverable(t *testing.T) {
	err1 := fmt.Errorf("err1")

	assertChainLen := func(t *testing.T, err error, expectedChainLen uint) {
		t.Helper()

		var len uint
		for ; err != nil; err = errors.Unwrap(err) {
			len++
		}

		if len != expectedChainLen {
			t.Fatalf(
				"error chain length is expected to be %d but was %d",
				expectedChainLen, len,
			)
		}
	}

	for _, test := range []struct {
		name     string
		err      error
		chainLen uint
		expected bool
	}{
		{
			name:     "nil",
			err:      nil,
			chainLen: 0,
			expected: false,
		},
		{
			name:     "other error",
			err:      err1,
			chainLen: 1,
			expected: false,
		},
		{
			name:     "direct recoverable",
			err:      Recoverable(err1),
			chainLen: 2,
			expected: true,
		},
		{
			name:     "direct non-recoverable",
			err:      NonRecoverable(Recoverable(err1)),
			chainLen: 3,
			expected: false,
		},
		{
			name:     "1-indirect recoverable",
			err:      fmt.Errorf("%w", Recoverable(err1)),
			chainLen: 3,
			expected: true,
		},
		{
			name:     "1-indirect non-recoverable",
			err:      fmt.Errorf("%w", NonRecoverable(Recoverable(err1))),
			chainLen: 4,
			expected: false,
		},
		{
			name:     "3-indirect recoverable",
			err:      fmt.Errorf("%w", fmt.Errorf("%w", fmt.Errorf("%w", Recoverable(err1)))),
			chainLen: 5,
			expected: true,
		},
		{
			name:     "3-indirect non-recoverable",
			err:      fmt.Errorf("%w", fmt.Errorf("%w", fmt.Errorf("%w", NonRecoverable(Recoverable(err1))))),
			chainLen: 6,
			expected: false,
		},
		{
			name: "multi-reset recoverable",
			err: fmt.Errorf("%w",
				Recoverable(
					fmt.Errorf("%w",
						NonRecoverable(
							fmt.Errorf("%w",
								Recoverable(
									fmt.Errorf("%w",
										NonRecoverable(
											fmt.Errorf("%w",
												Recoverable(err1),
											),
										),
									),
								),
							),
						),
					),
				),
			),
			chainLen: 11,
			expected: true,
		},
		{
			name: "multi-reset non-recoverable",
			err: fmt.Errorf("%w",
				NonRecoverable(
					fmt.Errorf("%w",
						Recoverable(
							fmt.Errorf("%w",
								NonRecoverable(
									fmt.Errorf("%w",
										Recoverable(err1),
									),
								),
							),
						),
					),
				),
			),
			chainLen: 9,
			expected: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test := test
			t.Parallel()

			// SETUP
			assertChainLen(t, test.err, test.chainLen)

			// EXERCISE
			result := IsRecoverable(test.err)

			// VERIFY
			if result != test.expected {
				t.Fatalf(
					"expected %t but got %t",
					test.expected, result,
				)
			}
		})
	}
}
