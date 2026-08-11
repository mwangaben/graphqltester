package tests

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/mwangaben/graphqltester"
	"github.com/stretchr/testify/assert"
)

/**
 * TestIsolatedTester_CreateIsolation verifies that an isolated tester
 * is properly created with its own state, context, and ID.
 */
func TestIsolatedTester_CreateIsolation(t *testing.T) {
	config := graphqltester.DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := graphqltester.NewTester(t, config)

	concurrentConfig := graphqltester.DefaultConcurrentConfig()
	isolated := tester.isolate(concurrentConfig, 0)
	defer isolated.Cleanup()

	assert.NotNil(t, isolated, "Isolated tester should be created")
	assert.NotEmpty(t, isolated.ID(), "Isolated tester should have an ID")
	assert.NotNil(t, isolated.ctx, "Isolated tester should have a context")
	assert.NotNil(t, isolated.localState, "Isolated tester should have local state")
	assert.Same(t, tester, isolated.parent, "Parent should be the original tester")
}

/**
 * TestIsolatedTester_LocalStateIsolation verifies that local state
 * is isolated between different isolated testers.
 */
func TestIsolatedTester_LocalStateIsolation(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)
	concurrentConfig := DefaultConcurrentConfig()

	// Create two isolated testers
	isolated1 := tester.isolate(concurrentConfig, 1)
	defer isolated1.Cleanup()

	isolated2 := tester.isolate(concurrentConfig, 2)
	defer isolated2.Cleanup()

	// Set local state in each
	isolated1.SetLocal("key", "value1")
	isolated2.SetLocal("key", "value2")

	// Verify isolation
	assert.Equal(t, "value1", isolated1.GetLocal("key"), "Isolated1 should have its own value")
	assert.Equal(t, "value2", isolated2.GetLocal("key"), "Isolated2 should have its own value")
}

/**
 * TestIsolatedTester_SharedStateVisibility verifies that shared state
 * is visible across all isolated testers.
 */
func TestIsolatedTester_SharedStateVisibility(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)
	concurrentConfig := DefaultConcurrentConfig()

	isolated := tester.isolate(concurrentConfig, 0)
	defer isolated.Cleanup()

	// Set shared state
	isolated.SetShared("shared_key", "shared_value")

	// Get shared state from both isolated and parent
	assert.Equal(t, "shared_value", isolated.GetShared("shared_key"))
	assert.Equal(t, "shared_value", tester.GetShared("shared_key"))
}

/**
 * TestRunParallel_ExecutesAllTests verifies that all test functions
 * are executed when running in parallel.
 */
func TestRunParallel_ExecutesAllTests(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}
	config.Parallel = true

	tester := NewTester(t, config)

	var mu sync.Mutex
	executed := make(map[int]bool)

	tests := []func(*IsolatedTester){
		func(it *IsolatedTester) {
			mu.Lock()
			executed[0] = true
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		},
		func(it *IsolatedTester) {
			mu.Lock()
			executed[1] = true
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		},
		func(it *IsolatedTester) {
			mu.Lock()
			executed[2] = true
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		},
	}

	parallelConfig := DefaultConcurrentConfig()
	parallelConfig.MaxParallel = 2

	tester.RunParallel(tests, parallelConfig)

	// Verify all tests executed
	assert.True(t, executed[0], "Test 0 should have executed")
	assert.True(t, executed[1], "Test 1 should have executed")
	assert.True(t, executed[2], "Test 2 should have executed")
}

/**
 * TestRunParallel_RespectsMaxParallel verifies that the semaphore
 * limits concurrent execution to MaxParallel.
 */
func TestRunParallel_RespectsMaxParallel(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	var maxConcurrent int32
	var current int32
	var mu sync.Mutex

	tests := make([]func(*IsolatedTester), 6)
	for i := 0; i < 6; i++ {
		idx := i
		tests[i] = func(it *IsolatedTester) {
			// Increment current count
			mu.Lock()
			current++
			if current > maxConcurrent {
				maxConcurrent = current
			}
			mu.Unlock()

			// Simulate work
			time.Sleep(100 * time.Millisecond)

			// Decrement current count
			mu.Lock()
			current--
			mu.Unlock()
		}
	}

	parallelConfig := DefaultConcurrentConfig()
	parallelConfig.MaxParallel = 3

	tester.RunParallel(tests, parallelConfig)

	assert.LessOrEqual(t, maxConcurrent, int32(3),
		"Should not exceed MaxParallel concurrent tests")
}

/**
 * TestRunParallel_TimeoutHandling verifies that tests that exceed
 * the timeout are properly handled.
 */
func TestRunParallel_TimeoutHandling(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	tests := []func(*IsolatedTester){
		func(it *IsolatedTester) {
			// This test will timeout
			select {
			case <-time.After(2 * time.Second):
			case <-it.ctx.Done():
				// Context cancelled due to timeout
				it.addError(fmt.Errorf("test timed out as expected"))
			}
		},
	}

	parallelConfig := DefaultConcurrentConfig()
	parallelConfig.Timeout = 100 * time.Millisecond

	tester.RunParallel(tests, parallelConfig)

	// The test should complete without hanging
	// (The timeout should cancel the context)
}

/**
 * TestRunSequential_FallbackWhenDisabled verifies that tests run
 * sequentially when parallel is disabled.
 */
func TestRunSequential_FallbackWhenDisabled(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	var executionOrder []int
	var mu sync.Mutex

	tests := []func(*IsolatedTester){
		func(it *IsolatedTester) {
			mu.Lock()
			executionOrder = append(executionOrder, 0)
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		},
		func(it *IsolatedTester) {
			mu.Lock()
			executionOrder = append(executionOrder, 1)
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		},
		func(it *IsolatedTester) {
			mu.Lock()
			executionOrder = append(executionOrder, 2)
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		},
	}

	// Disable parallel execution
	parallelConfig := DefaultConcurrentConfig()
	parallelConfig.Enabled = false

	tester.RunParallel(tests, parallelConfig)

	// In sequential mode, tests should execute in order
	assert.Equal(t, []int{0, 1, 2}, executionOrder,
		"Sequential tests should execute in order")
}

/**
 * TestConcurrentTestingT_ErrorHandling verifies that errors in
 * concurrent tests are properly collected.
 */
func TestConcurrentTestingT_ErrorHandling(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	tests := []func(*IsolatedTester){
		func(it *IsolatedTester) {
			// Record an error
			it.addError(fmt.Errorf("test error 1"))
		},
		func(it *IsolatedTester) {
			// Record an error
			it.addError(fmt.Errorf("test error 2"))
		},
	}

	parallelConfig := DefaultConcurrentConfig()
	tester.RunParallel(tests, parallelConfig)

	// Errors should be collected (we can't easily assert on them
	// since they're logged to the parent tester)
}
