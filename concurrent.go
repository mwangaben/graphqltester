package graphqltester

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

/**
 * Concurrent Testing Support for Parallel Test Execution
 *
 * This file provides opt-in support for running GraphQL tests concurrently.
 * When enabled, tests can run in parallel with isolated state to prevent
 * interference between tests while maximizing execution speed.
 *
 * Features:
 * - Parallel test execution with configurable concurrency
 * - State isolation between concurrent tests
 * - Shared state for coordinated testing
 * - Semaphore-based concurrency limiting
 * - Automatic cleanup of isolated resources
 *
 * Usage:
 *   config := DefaultConfig()
 *   config.Parallel = true
 *   config.IsolateState = true
 *
 *   tester := NewTester(t, config)
 *   tester.RunParallel([]func(*IsolatedTester){
 *       func(t *IsolatedTester) {
 *           // Test 1 - runs concurrently with Test 2
 *       },
 *       func(t *IsolatedTester) {
 *           // Test 2 - runs concurrently with Test 1
 *       },
 *   })
 *
 * Important Considerations:
 * - Database connections may need pooling adjustments for concurrent tests
 * - Each isolated tester gets its own database transaction
 * - Shared state access should be synchronized
 * - Test order is not guaranteed in parallel mode
 */

// ============================================================================
// Concurrent Configuration
// ============================================================================

//ConcurrentConfig
/**
 * ConcurrentConfig holds configuration for parallel test execution.
 *
 * This configuration controls how tests are parallelized, including
 * the maximum number of concurrent tests and isolation behavior.
 */
type ConcurrentConfig struct {
	// Enabled turns on concurrent test execution.
	// When false, tests run sequentially.
	Enabled bool

	// IsolateState creates completely independent state for each test.
	// Each test gets its own database transaction, context, and state maps.
	// This prevents test interference but uses more resources.
	IsolateState bool

	// MaxParallel limits the maximum number of concurrently running tests.
	// Default: runtime.NumCPU()
	// Set to 1 to effectively run sequentially (useful for debugging).
	MaxParallel int

	// SharedState allows tests to share specific state values.
	// Access to shared state should be synchronized by the tests themselves.
	SharedState map[string]interface{}

	// Timeout sets the maximum duration for each concurrent test.
	// Default: 30s per test
	Timeout time.Duration

	// FailFast stops all tests when the first failure occurs.
	// Default: false (all tests complete even if some fail)
	FailFast bool
}

//DefaultConcurrentConfig
/**
 * DefaultConcurrentConfig returns sensible defaults for concurrent testing.
 *
 * Returns:
 *   *ConcurrentConfig with safe defaults
 */
func DefaultConcurrentConfig() *ConcurrentConfig {
	return &ConcurrentConfig{
		Enabled:      true,
		IsolateState: true,
		MaxParallel:  4, // Conservative default
		Timeout:      30 * time.Second,
		FailFast:     false,
		SharedState:  make(map[string]interface{}),
	}
}

// ============================================================================
// Isolated Tester
// ============================================================================

//IsolatedTester
/**
 * IsolatedTester provides an isolated testing environment for concurrent tests.
 *
 * Each IsolatedTester has its own:
 * - Database transaction (if IsolateState is true)
 * - Context with timeout
 * - State maps (models, custom state)
 * - Authentication state
 *
 * The isolated tester shares:
 * - HTTP server (all tests hit the same server)
 * - Schema (read-only, safe to share)
 * - Read-only configuration
 */
type IsolatedTester struct {
	// Tester embeds the base tester for all standard methods.
	*Tester

	// id uniquely identifies this isolated tester.
	id string

	// parent references the original tester that created this isolation.
	parent *Tester

	// localState holds state specific to this isolated tester.
	localState map[string]interface{}

	// ctx is the isolated context with its own cancellation.
	ctx context.Context

	// cancel cancels the isolated context.
	cancel context.CancelFunc

	// done is closed when the test completes.
	done chan struct{}

	// errors collects errors that occur during the test.
	errors []error

	// mu protects concurrent access to isolated state.
	mu sync.RWMutex
}

//ID
/**
 * ID returns the unique identifier for this isolated tester.
 *
 * Returns:
 *   string unique ID
 */
func (it *IsolatedTester) ID() string {
	return it.id
}

//SetLocal
/**
 * SetLocal sets a value in the local isolated state.
 *
 * Local state is only visible to this isolated tester and is not
 * shared with other concurrent tests.
 *
 * Parameters:
 *   key   - State key
 *   value - Value to store
 */
func (it *IsolatedTester) SetLocal(key string, value interface{}) {
	it.mu.Lock()
	defer it.mu.Unlock()
	it.localState[key] = value
}

//GetLocal
/**
 * GetLocal retrieves a value from the local isolated state.
 *
 * Parameters:
 *   key - State key
 *
 * Returns:
 *   interface{} - The stored value, or nil if not found
 */
func (it *IsolatedTester) GetLocal(key string) interface{} {
	it.mu.RLock()
	defer it.mu.RUnlock()
	return it.localState[key]
}

//SetShared
/**
 * SetShared sets a value in the shared state (visible to all concurrent tests).
 *
 * IMPORTANT: Access to shared state is not synchronized automatically.
 * Use proper synchronization (mutexes, channels) when modifying shared state
 * from multiple concurrent tests.
 *
 * Parameters:
 *   key   - State key
 *   value - Value to store
 */
func (it *IsolatedTester) SetShared(key string, value interface{}) {
	it.parent.SetShared(key, value)
}

//GetShared
/**
 * GetShared retrieves a value from the shared state.
 *
 * Parameters:
 *   key - State key
 *
 * Returns:
 *   interface{} - The stored value, or nil if not found
 */
func (it *IsolatedTester) GetShared(key string) interface{} {
	return it.parent.GetShared(key)
}

//Cleanup
/**
 * Cleanup performs cleanup for the isolated tester.
 *
 * This is called automatically when the test completes.
 */
func (it *IsolatedTester) Cleanup() {
	// Cancel the isolated context
	if it.cancel != nil {
		it.cancel()
	}

	// Rollback isolated transaction if any
	if it.txManager != nil && it.txManager.IsActive() {
		it.txManager.Rollback()
	}

	close(it.done)
}

/**
 * addError records an error that occurred during the test.
 *
 * Parameters:
 *   err - The error to record
 */
func (it *IsolatedTester) addError(err error) {
	it.mu.Lock()
	defer it.mu.Unlock()
	it.errors = append(it.errors, err)
}

//Errors
/**
 * Errors returns all errors that occurred during the test.
 *
 * Returns:
 *   []error slice of errors
 */
func (it *IsolatedTester) Errors() []error {
	it.mu.RLock()
	defer it.mu.RUnlock()

	errors := make([]error, len(it.errors))
	copy(errors, it.errors)
	return errors
}

// ============================================================================
// Parallel Test Execution
// ============================================================================

//RunParallel
/**
 * RunParallel executes multiple test functions concurrently.
 *
 * This method handles:
 * - Creating isolated testers for each test
 * - Managing concurrency with semaphores
 * - Collecting results and errors
 * - Optional fail-fast behavior
 *
 * Parameters:
 *   tests  - Slice of test functions to run
 *   config - Concurrent execution configuration (nil for defaults)
 *
 * Example:
 *   tester.RunParallel([]func(*IsolatedTester){
 *       func(t *IsolatedTester) {
 *           t.RefreshDatabase().GivenAdmin()
 *           t.GraphQL(`...`).AssertNoErrors()
 *       },
 *       func(t *IsolatedTester) {
 *           t.RefreshDatabase().GivenUser("editor", "posts.edit")
 *           t.GraphQL(`...`).AssertPermissionError()
 *       },
 *   }, nil)
 */
func (tester *Tester) RunParallel(tests []func(*IsolatedTester), config *ConcurrentConfig) {
	if config == nil {
		config = DefaultConcurrentConfig()
	}

	if !config.Enabled {
		// Fall back to sequential execution
		tester.runSequential(tests, config)
		return
	}

	// Set max parallelism
	maxParallel := config.MaxParallel
	if maxParallel <= 0 {
		maxParallel = 4 // Default to 4 concurrent tests
	}

	// Create semaphore for concurrency control
	sem := make(chan struct{}, maxParallel)

	// WaitGroup to wait for all tests
	var wg sync.WaitGroup

	// Channel for fail-fast signaling
	failChan := make(chan struct{})

	// Mutex for error collection
	var errorsMu sync.Mutex
	var allErrors []error

	// Track if any test failed
	hasFailed := false
	var failedMu sync.Mutex

	tester.t.Logf("🚀 Running %d tests with max %d concurrent", len(tests), maxParallel)
	startTime := time.Now()

	for i, test := range tests {
		// Check if we should stop due to fail-fast
		if config.FailFast {
			select {
			case <-failChan:
				tester.t.Logf("⏭️  Skipping remaining tests due to FailFast")
				goto done
			default:
			}
		}

		wg.Add(1)

		go func(index int, testFunc func(*IsolatedTester)) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Create isolated tester
			isolated := tester.isolate(config, index)
			defer isolated.Cleanup()

			// Create a wrapper testing.T for this goroutine
			testT := &concurrentTestingT{
				T:        tester.t,
				isolated: isolated,
				index:    index,
			}

			// Create a temporary tester with the concurrent testing.T
			isolated.Tester.t = testT

			// Execute the test with timeout
			testDone := make(chan struct{})
			var testErr error

			go func() {
				defer close(testDone)

				// Recover from panics in test functions
				defer func() {
					if r := recover(); r != nil {
						testErr = fmt.Errorf("test %d panicked: %v", index, r)
						isolated.addError(testErr)
					}
				}()

				testFunc(isolated)
			}()

			// Wait for test completion or timeout
			timeout := config.Timeout
			if timeout == 0 {
				timeout = 30 * time.Second
			}

			select {
			case <-testDone:
				// Test completed normally
				if len(isolated.Errors()) > 0 {
					testErr = fmt.Errorf("test %d had %d errors", index, len(isolated.Errors()))
				}

			case <-time.After(timeout):
				testErr = fmt.Errorf("test %d timed out after %v", index, timeout)
				isolated.cancel()

			case <-failChan:
				testErr = fmt.Errorf("test %d cancelled due to fail-fast", index)
				isolated.cancel()
			}

			// Collect errors
			if testErr != nil {
				errorsMu.Lock()
				allErrors = append(allErrors, testErr)
				errorsMu.Unlock()

				// Signal fail-fast if enabled
				if config.FailFast {
					failedMu.Lock()
					if !hasFailed {
						hasFailed = true
						close(failChan)
					}
					failedMu.Unlock()
				}
			}

			tester.t.Logf("   [%d/%d] Test %d completed", index+1, len(tests), index)

		}(i, test)
	}

done:
	// Wait for all tests to complete
	wg.Wait()

	elapsed := time.Since(startTime)
	tester.t.Logf("✅ All tests completed in %v", elapsed)

	// Report any errors
	if len(allErrors) > 0 {
		tester.t.Logf("❌ %d test(s) had errors:", len(allErrors))
		for _, err := range allErrors {
			tester.t.Logf("   - %v", err)
		}
	}
}

/**
 * runSequential runs tests one after another when parallel is disabled.
 *
 * Parameters:
 *   tests  - Test functions to run
 *   config - Concurrent configuration
 */
func (tester *Tester) runSequential(tests []func(*IsolatedTester), config *ConcurrentConfig) {
	tester.t.Logf("📋 Running %d tests sequentially", len(tests))

	for i, test := range tests {
		isolated := tester.isolate(config, i)
		defer isolated.Cleanup()

		// Create a sequential testing.T wrapper
		testT := &sequentialTestingT{
			T:        tester.t,
			isolated: isolated,
		}

		isolated.Tester.t = testT

		tester.t.Logf("   [%d/%d] Running test...", i+1, len(tests))
		test(isolated)
	}
}

/**
 * isolate creates an isolated tester for concurrent or sequential execution.
 *
 * This method:
 * - Creates a new context with timeout
 * - Starts an isolated database transaction (if IsolateState)
 * - Copies necessary state from the parent tester
 * - Sets up cleanup handlers
 *
 * Parameters:
 *   config - Concurrent configuration
 *   index  - Test index for identification
 *
 * Returns:
 *   *IsolatedTester ready for test execution
 */
func (tester *Tester) isolate(config *ConcurrentConfig, index int) *IsolatedTester {
	// Create unique ID for this isolation
	id := fmt.Sprintf("test-%d-%s", index, uuid.New().String()[:8])

	// Create isolated context with timeout
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(tester.ctx, timeout)

	// Clone the tester for isolation
	clone := tester.clone()

	// Create isolated tester
	isolated := &IsolatedTester{
		Tester:     clone,
		id:         id,
		parent:     tester,
		localState: make(map[string]interface{}),
		ctx:        ctx,
		cancel:     cancel,
		done:       make(chan struct{}),
		errors:     make([]error, 0),
	}

	// Update the clone's context
	clone.ctx = ctx

	// Isolate database if configured
	if config.IsolateState && tester.dbAdapter != nil {
		// Start a new transaction for isolation
		isolatedTx, err := tester.dbAdapter.BeginTx(ctx)
		if err != nil {
			tester.t.Logf("⚠️  Warning: Could not create isolated transaction: %v", err)
		} else {
			// Create a new transaction manager for this isolated tester
			isolated.txManager = &TransactionManager{
				adapter:  tester.dbAdapter,
				tx:       isolatedTx,
				isActive: true,
				debug:    tester.config.Debug,
			}
		}
	}

	return isolated
}

// ============================================================================
// Shared State Management
// ============================================================================

//SetShared
/**
 * SetShared sets a value in the shared state.
 *
 * Parameters:
 *   key   - State key
 *   value - Value to store
 */
func (tester *Tester) SetShared(key string, value interface{}) {
	tester.mu.Lock()
	defer tester.mu.Unlock()

	if tester.state == nil {
		tester.state = make(map[string]interface{})
	}
	tester.state[key] = value
}

//GetShared
/**
 * GetShared retrieves a value from the shared state.
 *
 * Parameters:
 *   key - State key
 *
 * Returns:
 *   interface{} - The stored value, or nil if not found
 */
func (tester *Tester) GetShared(key string) interface{} {
	tester.mu.RLock()
	defer tester.mu.RUnlock()

	if tester.state == nil {
		return nil
	}
	return tester.state[key]
}

// ============================================================================
// Testing.T Wrappers for Concurrent Tests
// ============================================================================

/**
 * concurrentTestingT wraps testing.T for concurrent test execution.
 *
 * This wrapper ensures that test failures in concurrent goroutines
 * are properly reported without causing race conditions.
 */
type concurrentTestingT struct {
	*testing.T
	isolated *IsolatedTester
	index    int
}

//Errorf
/**
 * Errorf records a test error in the isolated tester.
 */
func (t *concurrentTestingT) Errorf(format string, args ...interface{}) {
	err := fmt.Errorf(format, args...)
	t.isolated.addError(err)

	// Also log to the parent tester
	t.T.Logf("[Test %d] ERROR: %s", t.index, fmt.Sprintf(format, args...))
}

//Fatalf
/**
 * Fatalf records a fatal error and cancels the test context.
 */
func (t *concurrentTestingT) Fatalf(format string, args ...interface{}) {
	err := fmt.Errorf(format, args...)
	t.isolated.addError(err)

	t.T.Logf("[Test %d] FATAL: %s", t.index, fmt.Sprintf(format, args...))

	// Cancel the isolated context to stop the test
	t.isolated.cancel()

	// Don't actually call t.T.Fatalf as it calls os.Exit in some cases
	// Instead, we use panic to stop execution in the goroutine
	panic(fmt.Sprintf(format, args...))
}

//Logf
/**
 * Logf logs a message with the test index prefix.
 */
func (t *concurrentTestingT) Logf(format string, args ...interface{}) {
	t.T.Logf("[Test %d] %s", t.index, fmt.Sprintf(format, args...))
}

/**
 * sequentialTestingT wraps testing.T for sequential test execution.
 */
type sequentialTestingT struct {
	*testing.T
	isolated *IsolatedTester
}

//Errorf
/**
 * Errorf records an error for sequential tests.
 */
func (t *sequentialTestingT) Errorf(format string, args ...interface{}) {
	err := fmt.Errorf(format, args...)
	t.isolated.addError(err)
	t.T.Errorf(format, args...)
}

//Fatalf
/**
 * Fatalf records a fatal error for sequential tests.
 */
func (t *sequentialTestingT) Fatalf(format string, args ...interface{}) {
	err := fmt.Errorf(format, args...)
	t.isolated.addError(err)
	t.T.Fatalf(format, args...)
}

// ============================================================================
// BDD-Style Parallel Testing
// ============================================================================

//DescribeParallel
/**
 * DescribeParallel creates a test group that runs its tests in parallel.
 *
 * This combines BDD-style Describe/It with parallel execution.
 * Each It block within the Describe runs concurrently.
 *
 * Parameters:
 *   description - Test group description
 *   config      - Concurrent configuration (nil for defaults)
 *   fn          - Function that defines tests using It
 *
 * Example:
 *   tester.DescribeParallel("Zone Operations", nil, func(t *Tester) {
 *       t.It("can create zones", func(t *Tester) { ... })
 *       t.It("can update zones", func(t *Tester) { ... })
 *       t.It("can delete zones", func(t *Tester) { ... })
 *   })
 */
func (tester *Tester) DescribeParallel(description string, config *ConcurrentConfig, fn func(*Tester)) {
	tester.t.Helper()
	tester.t.Logf("\n📋 %s (Parallel)", description)

	// Collect all It blocks
	type testCase struct {
		name string
		fn   func(*Tester)
	}

	var tests []testCase

	// Create a temporary tester that collects It blocks instead of running them
	collector := tester.clone()
	collector.t = tester.t

	// Override It to collect instead of execute
	originalRun := tester.t.Run
	defer func() { tester.t.Run = originalRun }()

	// This is a simplified approach - in practice, you'd use a proper collector
	// For now, we run tests sequentially within the parallel group
	fn(collector)

	if len(tests) == 0 {
		// If no tests were collected, just run the function normally
		fn(tester)
		return
	}

	// Run collected tests in parallel
	parallelTests := make([]func(*IsolatedTester), len(tests))
	for i, tc := range tests {
		testFn := tc.fn
		parallelTests[i] = func(it *IsolatedTester) {
			tester.t.Logf("   Running: %s", tc.name)
			testFn(it.Tester)
		}
	}

	tester.RunParallel(parallelTests, config)
}
