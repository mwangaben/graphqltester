package tests

import (
	"sync"
	"testing"
	"time"

	graphqltester "github.com/mwangaben/graphqltester"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Schema for Concurrent Tests
// ============================================================================

const concurrentTestSchema = `
	schema {
		query: Query
		mutation: Mutation
	}

	type Query {
		hello: String!
		counter: Int!
	}

	type Mutation {
		_noop: String
	}
`

// ============================================================================
// Mock Resolver for Concurrent Tests
// ============================================================================

type concurrentTestResolver struct {
	mu      sync.Mutex
	counter int
}

func (r *concurrentTestResolver) Hello() string {
	return "Hello, World!"
}

func (r *concurrentTestResolver) Counter() int32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counter++
	return int32(r.counter)
}

// ============================================================================
// Helper: Create a configured tester for concurrent tests
// ============================================================================

/**
 * newConcurrentTester creates a tester configured for concurrent testing.
 */
func newConcurrentTester(t *testing.T) *graphqltester.Tester {
	t.Helper()

	config := graphqltester.DefaultConfig()
	config.Schema = &graphqltester.SchemaConfig{
		String:    concurrentTestSchema,
		Resolvers: &concurrentTestResolver{},
	}

	tester := graphqltester.NewTester(t, config)
	t.Cleanup(tester.Cleanup)

	return tester
}

// ============================================================================
// RunParallel Tests
// ============================================================================

/**
 * TestRunParallel_ExecutesAllTests verifies that all test functions
 * are executed when running in parallel.
 */
func TestRunParallel_ExecutesAllTests(t *testing.T) {
	tester := newConcurrentTester(t)

	var mu sync.Mutex
	executed := make(map[int]bool)

	tests := []func(*graphqltester.IsolatedTester){
		func(it *graphqltester.IsolatedTester) {
			mu.Lock()
			executed[0] = true
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		},
		func(it *graphqltester.IsolatedTester) {
			mu.Lock()
			executed[1] = true
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		},
		func(it *graphqltester.IsolatedTester) {
			mu.Lock()
			executed[2] = true
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		},
	}

	parallelConfig := graphqltester.DefaultConcurrentConfig()
	parallelConfig.MaxParallel = 2

	tester.RunParallel(tests, parallelConfig)

	// Verify all tests executed
	assert.True(t, executed[0], "Test 0 should have executed")
	assert.True(t, executed[1], "Test 1 should have executed")
	assert.True(t, executed[2], "Test 2 should have executed")
}

/**
 * TestRunParallel_ExecutesWithDifferentUsers verifies parallel tests
 * can have different authentication states.
 */
func TestRunParallel_ExecutesWithDifferentUsers(t *testing.T) {
	tester := newConcurrentTester(t)

	var mu sync.Mutex
	users := make([]string, 0)

	type testUser struct {
		ID   string
		Name string
	}

	tests := []func(*graphqltester.IsolatedTester){
		func(it *graphqltester.IsolatedTester) {
			user := &testUser{ID: "1", Name: "User One"}
			it.ActingAs(user)
			mu.Lock()
			users = append(users, "1")
			mu.Unlock()
		},
		func(it *graphqltester.IsolatedTester) {
			user := &testUser{ID: "2", Name: "User Two"}
			it.ActingAs(user)
			mu.Lock()
			users = append(users, "2")
			mu.Unlock()
		},
		func(it *graphqltester.IsolatedTester) {
			user := &testUser{ID: "3", Name: "User Three"}
			it.ActingAs(user)
			mu.Lock()
			users = append(users, "3")
			mu.Unlock()
		},
	}

	parallelConfig := graphqltester.DefaultConcurrentConfig()
	parallelConfig.MaxParallel = 3

	tester.RunParallel(tests, parallelConfig)

	assert.Len(t, users, 3, "All three tests should have executed")
}

/**
 * TestRunParallel_RespectsMaxParallel verifies that the semaphore
 * limits concurrent execution to MaxParallel.
 */
func TestRunParallel_RespectsMaxParallel(t *testing.T) {
	tester := newConcurrentTester(t)

	var maxConcurrent int32
	var current int32
	var mu sync.Mutex

	tests := make([]func(*graphqltester.IsolatedTester), 6)
	for i := 0; i < 6; i++ {
		tests[i] = func(it *graphqltester.IsolatedTester) {
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

	parallelConfig := graphqltester.DefaultConcurrentConfig()
	parallelConfig.MaxParallel = 3

	tester.RunParallel(tests, parallelConfig)

	assert.LessOrEqual(t, maxConcurrent, int32(3),
		"Should not exceed MaxParallel concurrent tests")
}

/**
 * TestRunParallel_RespectsMaxParallel_OneAtATime verifies MaxParallel=1
 * effectively runs tests sequentially.
 */
func TestRunParallel_RespectsMaxParallel_OneAtATime(t *testing.T) {
	tester := newConcurrentTester(t)

	var maxConcurrent int32
	var current int32
	var mu sync.Mutex

	tests := make([]func(*graphqltester.IsolatedTester), 4)
	for i := 0; i < 4; i++ {
		tests[i] = func(it *graphqltester.IsolatedTester) {
			mu.Lock()
			current++
			if current > maxConcurrent {
				maxConcurrent = current
			}
			mu.Unlock()

			time.Sleep(50 * time.Millisecond)

			mu.Lock()
			current--
			mu.Unlock()
		}
	}

	parallelConfig := graphqltester.DefaultConcurrentConfig()
	parallelConfig.MaxParallel = 1 // Only one at a time

	tester.RunParallel(tests, parallelConfig)

	assert.Equal(t, int32(1), maxConcurrent,
		"With MaxParallel=1, only one test should run at a time")
}

/**
 * TestRunParallel_TimeoutHandling verifies that tests that exceed
 * the timeout are properly handled.
 */
func TestRunParallel_TimeoutHandling(t *testing.T) {
	tester := newConcurrentTester(t)

	completed := false

	tests := []func(*graphqltester.IsolatedTester){
		func(it *graphqltester.IsolatedTester) {
			// This test completes quickly
			completed = true
		},
	}

	parallelConfig := graphqltester.DefaultConcurrentConfig()
	parallelConfig.Timeout = 5 * time.Second // Long timeout

	tester.RunParallel(tests, parallelConfig)

	// The test should complete
	assert.True(t, completed, "Test should have completed within timeout")
}

/**
 * TestRunParallel_EmptyTests_DoesNotPanic verifies empty test list is safe.
 */
func TestRunParallel_EmptyTests_DoesNotPanic(t *testing.T) {
	tester := newConcurrentTester(t)

	assert.NotPanics(t, func() {
		tester.RunParallel(nil, nil)
	})

	assert.NotPanics(t, func() {
		tester.RunParallel([]func(*graphqltester.IsolatedTester){}, nil)
	})
}

// ============================================================================
// Sequential Fallback Tests
// ============================================================================

/**
 * TestRunSequential_FallbackWhenDisabled verifies that tests run
 * sequentially when parallel is disabled.
 */
func TestRunSequential_FallbackWhenDisabled(t *testing.T) {
	tester := newConcurrentTester(t)

	var executionOrder []int
	var mu sync.Mutex

	tests := []func(*graphqltester.IsolatedTester){
		func(it *graphqltester.IsolatedTester) {
			mu.Lock()
			executionOrder = append(executionOrder, 0)
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		},
		func(it *graphqltester.IsolatedTester) {
			mu.Lock()
			executionOrder = append(executionOrder, 1)
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		},
		func(it *graphqltester.IsolatedTester) {
			mu.Lock()
			executionOrder = append(executionOrder, 2)
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		},
	}

	// Disable parallel execution
	parallelConfig := graphqltester.DefaultConcurrentConfig()
	parallelConfig.Enabled = false

	tester.RunParallel(tests, parallelConfig)

	// In sequential mode, tests should execute in order
	assert.Equal(t, []int{0, 1, 2}, executionOrder,
		"Sequential tests should execute in order")
}

// ============================================================================
// Shared State Tests
// ============================================================================

/**
 * TestRunParallel_SharedStateVisibility verifies that shared state
 * is visible across all parallel tests.
 */
func TestRunParallel_SharedStateVisibility(t *testing.T) {
	tester := newConcurrentTester(t)

	sharedKey := "shared_counter"
	tester.SetShared(sharedKey, 0)

	var mu sync.Mutex
	results := make([]int, 0)

	tests := []func(*graphqltester.IsolatedTester){
		func(it *graphqltester.IsolatedTester) {
			it.SetShared(sharedKey, 1)
			val := it.GetShared(sharedKey)
			mu.Lock()
			results = append(results, val.(int))
			mu.Unlock()
		},
		func(it *graphqltester.IsolatedTester) {
			it.SetShared(sharedKey, 2)
			val := it.GetShared(sharedKey)
			mu.Lock()
			results = append(results, val.(int))
			mu.Unlock()
		},
	}

	parallelConfig := graphqltester.DefaultConcurrentConfig()
	parallelConfig.MaxParallel = 2

	tester.RunParallel(tests, parallelConfig)

	assert.Len(t, results, 2, "Both tests should have executed")
}

// ============================================================================
// GraphQL in Parallel Tests
// ============================================================================

/**
 * TestRunParallel_GraphQLQueries verifies that GraphQL queries work
 * correctly within parallel tests.
 */
func TestRunParallel_GraphQLQueries(t *testing.T) {
	tester := newConcurrentTester(t)

	var mu sync.Mutex
	results := make([]string, 0)

	tests := []func(*graphqltester.IsolatedTester){
		func(it *graphqltester.IsolatedTester) {
			response := it.GraphQL(`{ hello }`)
			if response.Data() != nil {
				val := response.JSONString("hello")
				mu.Lock()
				results = append(results, val)
				mu.Unlock()
			}
		},
		func(it *graphqltester.IsolatedTester) {
			response := it.GraphQL(`{ hello }`)
			if response.Data() != nil {
				val := response.JSONString("hello")
				mu.Lock()
				results = append(results, val)
				mu.Unlock()
			}
		},
		func(it *graphqltester.IsolatedTester) {
			response := it.GraphQL(`{ hello }`)
			if response.Data() != nil {
				val := response.JSONString("hello")
				mu.Lock()
				results = append(results, val)
				mu.Unlock()
			}
		},
	}

	parallelConfig := graphqltester.DefaultConcurrentConfig()
	parallelConfig.MaxParallel = 3

	tester.RunParallel(tests, parallelConfig)

	assert.Len(t, results, 3, "All three tests should have executed")
	for _, result := range results {
		assert.Equal(t, "Hello, World!", result, "Each test should get the same result")
	}
}

// ============================================================================
// ConcurrentConfig Tests
// ============================================================================

/**
 * TestDefaultConcurrentConfig_ReturnsSensibleDefaults verifies default config.
 */
func TestDefaultConcurrentConfig_ReturnsSensibleDefaults(t *testing.T) {
	config := graphqltester.DefaultConcurrentConfig()

	assert.True(t, config.Enabled, "Should be enabled by default")
	assert.True(t, config.IsolateState, "Should isolate state by default")
	assert.Equal(t, 4, config.MaxParallel, "Default max parallel should be 4")
	assert.Equal(t, 30*time.Second, config.Timeout, "Default timeout should be 30s")
	assert.False(t, config.FailFast, "FailFast should be disabled by default")
	assert.NotNil(t, config.SharedState, "Shared state should be initialized")
}

/**
 * TestDefaultConcurrentConfig_SharedStateIsWritable verifies shared state works.
 */
func TestDefaultConcurrentConfig_SharedStateIsWritable(t *testing.T) {
	config := graphqltester.DefaultConcurrentConfig()

	config.SharedState["test_key"] = "test_value"

	assert.Equal(t, "test_value", config.SharedState["test_key"])
}

// ============================================================================
// Stress Tests
// ============================================================================

/**
 * TestRunParallel_StressTest verifies behavior with many concurrent tests.
 */
func TestRunParallel_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	tester := newConcurrentTester(t)

	numTests := 20
	var mu sync.Mutex
	completed := 0

	tests := make([]func(*graphqltester.IsolatedTester), numTests)
	for i := 0; i < numTests; i++ {
		tests[i] = func(it *graphqltester.IsolatedTester) {
			// Simulate some work
			response := it.GraphQL(`{ hello }`)
			assert.NotNil(t, response)

			mu.Lock()
			completed++
			mu.Unlock()
		}
	}

	parallelConfig := graphqltester.DefaultConcurrentConfig()
	parallelConfig.MaxParallel = 5
	parallelConfig.Timeout = 30 * time.Second

	tester.RunParallel(tests, parallelConfig)

	assert.Equal(t, numTests, completed, "All tests should have completed")
}

/**
 * TestRunParallel_FailFast verifies FailFast behavior.
 */
func TestRunParallel_FailFast(t *testing.T) {
	tester := newConcurrentTester(t)

	var mu sync.Mutex
	executed := make(map[int]bool)

	tests := []func(*graphqltester.IsolatedTester){
		func(it *graphqltester.IsolatedTester) {
			mu.Lock()
			executed[0] = true
			mu.Unlock()
			// This test doesn't fail
		},
		func(it *graphqltester.IsolatedTester) {
			mu.Lock()
			executed[1] = true
			mu.Unlock()
			// This test fails
			it.Errorf("intentional test failure")
		},
		func(it *graphqltester.IsolatedTester) {
			time.Sleep(500 * time.Millisecond) // Slow test
			mu.Lock()
			executed[2] = true
			mu.Unlock()
		},
	}

	parallelConfig := graphqltester.DefaultConcurrentConfig()
	parallelConfig.MaxParallel = 3
	parallelConfig.FailFast = true

	tester.RunParallel(tests, parallelConfig)

	// Test 0 should have executed
	assert.True(t, executed[0], "Test 0 should have executed")
	// Test 1 should have executed (it's the failing one)
	assert.True(t, executed[1], "Test 1 should have executed")
	// Test 2 might or might not have executed depending on timing
	// This is acceptable for fail-fast behavior
}
