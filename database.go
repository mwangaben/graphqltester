package graphqltester

import (
	"context"
	"fmt"
	"github.com/mwangaben/graphqltester/types"
)

/**
 * Database Management for Test Isolation
 *
 * This file provides database transaction management, migration support,
 * and database state helpers for test isolation and cleanup.
 *
 * The transaction manager wraps each test in a database transaction that
 * is automatically rolled back after the test completes, ensuring that
 * each test starts with a clean database state.
 *
 * Features:
 * - Transaction-based test isolation
 * - Savepoint support for nested operations
 * - Database refresh/reset capabilities
 * - Migration and seeding support
 */

// ============================================================================
// Transaction Manager
// ============================================================================

/**
 * TransactionManager handles database transaction lifecycle for test isolation.
 *
 * It wraps database operations in transactions that can be rolled back,
 * ensuring tests don't affect each other's data. Supports savepoints
 * for nested transaction scenarios.
 *
 * The transaction manager is created automatically by the Tester when
 * database configuration is provided with UseTransactions enabled.
 *
 * Usage:
 *   tm := NewTransactionManager(dbAdapter)
 *   tm.Begin()
 *   defer tm.Rollback()
 *   // ... test operations ...
 *   tm.Commit() // or let defer rollback for test isolation
 */
type TransactionManager struct {
	// adapter is the database adapter for executing operations.
	adapter types.DatabaseAdapter

	// tx holds the current active transaction.
	// The type varies by adapter (e.g., *sql.Tx, *gorm.DB, *sqlx.Tx).
	tx interface{}

	// isActive indicates whether a transaction is currently in progress.
	isActive bool

	// savepoints tracks named savepoints for nested rollbacks.
	savepoints []string

	// debug enables logging of transaction operations.
	debug bool
}

/**
 * NewTransactionManager creates a new transaction manager.
 *
 * Parameters:
 *   adapter - The database adapter to use for transaction operations
 *
 * Returns:
 *   *TransactionManager ready for use
 *
 * Example:
 *   tm := NewTransactionManager(dbAdapter)
 */
func NewTransactionManager(adapter types.DatabaseAdapter) *TransactionManager {
	return &TransactionManager{
		adapter:    adapter,
		savepoints: make([]string, 0),
		debug:      false,
	}
}

/**
 * Begin starts a new database transaction.
 *
 * This is called automatically before each test if UseTransactions is
 * enabled in the database configuration.
 *
 * Only one transaction can be active at a time. Calling Begin while
 * a transaction is already active will return an error.
 *
 * Returns:
 *   error if the transaction cannot be started
 *
 * Example:
 *   if err := tm.Begin(); err != nil {
 *       t.Fatalf("Failed to begin transaction: %v", err)
 *   }
 *   defer tm.Rollback()
 */
func (tm *TransactionManager) Begin() error {
	if tm.isActive {
		return fmt.Errorf("transaction is already active")
	}

	tx, err := tm.adapter.BeginTx(context.Background())
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	tm.tx = tx
	tm.isActive = true

	if tm.debug {
		fmt.Println("📌 Transaction started")
	}

	return nil
}

/**
 * Commit commits the current transaction.
 *
 * Use this when you want to persist changes made during a test.
 * Note: With test isolation, transactions are typically rolled back,
 * not committed.
 *
 * Returns:
 *   error if the commit fails
 *
 * Example:
 *   tm.Begin()
 *   // ... make changes ...
 *   if err := tm.Commit(); err != nil {
 *       t.Fatalf("Failed to commit: %v", err)
 *   }
 */
func (tm *TransactionManager) Commit() error {
	if !tm.isActive {
		return fmt.Errorf("no active transaction to commit")
	}

	if err := tm.adapter.Commit(tm.tx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	tm.isActive = false
	tm.tx = nil

	if tm.debug {
		fmt.Println("✅ Transaction committed")
	}

	return nil
}

/**
 * Rollback rolls back the current transaction.
 *
 * This is the default behavior after each test, ensuring test isolation
 * by discarding all changes made during the test.
 *
 * Returns:
 *   error if the rollback fails
 *
 * Example:
 *   tm.Begin()
 *   defer tm.Rollback() // Automatically rollback after test
 *   // ... test operations ...
 */
func (tm *TransactionManager) Rollback() error {
	if !tm.isActive {
		return fmt.Errorf("no active transaction to rollback")
	}

	if err := tm.adapter.Rollback(tm.tx); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	tm.isActive = false
	tm.tx = nil
	tm.savepoints = nil

	if tm.debug {
		fmt.Println("↩️  Transaction rolled back")
	}

	return nil
}

/**
 * IsActive returns whether a transaction is currently active.
 *
 * Returns:
 *   bool indicating transaction status
 *
 * Example:
 *   if tm.IsActive() {
 *       tm.Rollback()
 *   }
 */
func (tm *TransactionManager) IsActive() bool {
	return tm.isActive
}

/**
 * Savepoint creates a named savepoint within the current transaction.
 *
 * Savepoints allow partial rollbacks within a transaction. This is useful
 * for scenarios where you want to revert part of a test while keeping
 * other changes.
 *
 * Parameters:
 *   name - Unique name for the savepoint
 *
 * Returns:
 *   error if the savepoint cannot be created
 *
 * Example:
 *   tm.Begin()
 *
 *   // Create some data
 *   createInitialData()
 *
 *   // Create a savepoint
 *   tm.Savepoint("before_test")
 *
 *   // Do test operations
 *   testOperations()
 *
 *   // Rollback only test operations, keep initial data
 *   tm.RollbackToSavepoint("before_test")
 */
func (tm *TransactionManager) Savepoint(name string) error {
	if !tm.isActive {
		return fmt.Errorf("no active transaction for savepoint")
	}

	// Check for duplicate savepoint names
	for _, sp := range tm.savepoints {
		if sp == name {
			return fmt.Errorf("savepoint '%s' already exists", name)
		}
	}

	if err := tm.adapter.Exec(fmt.Sprintf("SAVEPOINT %s", name)); err != nil {
		return fmt.Errorf("failed to create savepoint: %w", err)
	}

	tm.savepoints = append(tm.savepoints, name)

	if tm.debug {
		fmt.Printf("💾 Savepoint '%s' created\n", name)
	}

	return nil
}

/**
 * RollbackToSavepoint rolls back to a named savepoint.
 *
 * This discards all changes made after the savepoint while keeping
 * changes made before it.
 *
 * Parameters:
 *   name - The savepoint name to rollback to
 *
 * Returns:
 *   error if the rollback fails
 *
 * Example:
 *   tm.RollbackToSavepoint("before_test")
 */
func (tm *TransactionManager) RollbackToSavepoint(name string) error {
	if !tm.isActive {
		return fmt.Errorf("no active transaction")
	}

	found := false
	for _, sp := range tm.savepoints {
		if sp == name {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("savepoint '%s' not found", name)
	}

	if err := tm.adapter.Exec(fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", name)); err != nil {
		return fmt.Errorf("failed to rollback to savepoint: %w", err)
	}

	// Remove savepoints after the rolled-back one
	newSavepoints := make([]string, 0)
	for _, sp := range tm.savepoints {
		newSavepoints = append(newSavepoints, sp)
		if sp == name {
			break
		}
	}
	tm.savepoints = newSavepoints

	if tm.debug {
		fmt.Printf("↩️  Rolled back to savepoint '%s'\n", name)
	}

	return nil
}

/**
 * SetDebug enables or disables debug logging for transaction operations.
 *
 * Parameters:
 *   debug - Whether to enable debug logging
 */
func (tm *TransactionManager) SetDebug(debug bool) {
	tm.debug = debug
}

/**
 * GetTransaction returns the underlying transaction object.
 *
 * This provides access to the raw transaction for adapter-specific operations.
 * The type varies by adapter (e.g., *sql.Tx, *gorm.DB, *sqlx.Tx).
 *
 * Returns:
 *   interface{} - The transaction object, or nil if no active transaction
 */
func (tm *TransactionManager) GetTransaction() interface{} {
	return tm.tx
}

// ============================================================================
// Database Helper Functions
// ============================================================================

/**
 * databaseConfig is a helper to safely get the database configuration.
 *
 * Returns:
 *   *DatabaseConfig or nil if not configured
 */
func (tester *Tester) databaseConfig() *DatabaseConfig {
	if tester.config == nil {
		return nil
	}
	return tester.config.Database
}

/**
 * isDatabaseConfigured checks if the database is properly configured.
 *
 * Returns:
 *   bool - true if database adapter and config are available
 */
func (tester *Tester) isDatabaseConfigured() bool {
	return tester.dbAdapter != nil && tester.config.Database != nil
}

/**
 * ensureTransaction ensures a transaction is active, starting one if needed.
 *
 * Returns:
 *   error if transaction cannot be started
 */
func (tester *Tester) ensureTransaction() error {
	if tester.txManager == nil {
		return fmt.Errorf("transaction manager not initialized")
	}

	if !tester.txManager.IsActive() {
		return tester.txManager.Begin()
	}

	return nil
}

// Ensure types import is used
var _ types.DatabaseAdapter = nil
