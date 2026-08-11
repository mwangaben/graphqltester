package assertions

import (
	"context"

	"github.com/mwangaben/graphqltester/types"
)

/**
 * DatabaseAssertions provides methods for verifying database state.
 *
 * These assertions check that GraphQL operations correctly persist,
 * update, or delete data in the database.
 *
 * Works with types.AssertionResponse and types.TesterInterface
 * to avoid cyclic imports with the main package.
 */
type DatabaseAssertions struct {
	response types.AssertionResponse
	tester   types.TesterInterface
}

/**
 * NewDatabaseAssertions creates database assertion helpers.
 *
 * Parameters:
 *   response - The response to associate with assertions
 *   tester   - The tester instance for database access
 *
 * Returns:
 *   *DatabaseAssertions for chaining
 */
func NewDatabaseAssertions(response types.AssertionResponse, tester types.TesterInterface) *DatabaseAssertions {
	return &DatabaseAssertions{
		response: response,
		tester:   tester,
	}
}

/**
 * AssertDatabaseHas asserts a record exists in the database.
 *
 * Parameters:
 *   table      - The database table name
 *   conditions - Column-value pairs to match
 *
 * Returns:
 *   *DatabaseAssertions for chaining
 */
func (da *DatabaseAssertions) AssertDatabaseHas(table string, conditions map[string]interface{}) *DatabaseAssertions {
	db := da.tester.Database()
	if db == nil {
		da.tester.Fatalf("❌ Database is not configured. Set Database in Config.")
	}

	has, err := db.HasRecord(context.Background(), table, conditions)
	if err != nil {
		da.tester.Fatalf("❌ Database check failed: %v", err)
	}

	if !has {
		da.tester.Errorf("❌ Expected record in '%s' with conditions: %v", table, conditions)
		da.debugTableContents(table)
	}

	return da
}

/**
 * AssertDatabaseMissing asserts a record does NOT exist.
 *
 * Parameters:
 *   table      - The database table name
 *   conditions - Column-value pairs to match
 *
 * Returns:
 *   *DatabaseAssertions for chaining
 */
func (da *DatabaseAssertions) AssertDatabaseMissing(table string, conditions map[string]interface{}) *DatabaseAssertions {
	db := da.tester.Database()
	if db == nil {
		da.tester.Fatalf("❌ Database is not configured")
	}

	has, err := db.HasRecord(context.Background(), table, conditions)
	if err != nil {
		da.tester.Fatalf("❌ Database check failed: %v", err)
	}

	if has {
		da.tester.Errorf("❌ Expected no record in '%s' with conditions: %v", table, conditions)
		da.debugTableContents(table)
	}

	return da
}

/**
 * AssertSoftDeleted asserts a record is soft deleted.
 *
 * Parameters:
 *   table      - The database table name
 *   conditions - Conditions to identify the record
 *
 * Returns:
 *   *DatabaseAssertions for chaining
 */
func (da *DatabaseAssertions) AssertSoftDeleted(table string, conditions map[string]interface{}) *DatabaseAssertions {
	db := da.tester.Database()
	if db == nil {
		da.tester.Fatalf("❌ Database is not configured")
	}

	isDeleted, err := db.IsSoftDeleted(context.Background(), table, conditions)
	if err != nil {
		da.tester.Fatalf("❌ Soft delete check failed: %v", err)
	}

	if !isDeleted {
		da.tester.Errorf("❌ Expected soft-deleted record in '%s' with conditions: %v", table, conditions)
		da.debugTableContents(table)
	}

	return da
}

/**
 * AssertNotSoftDeleted asserts a record is NOT soft deleted.
 *
 * Parameters:
 *   table      - The database table name
 *   conditions - Conditions to identify the record
 *
 * Returns:
 *   *DatabaseAssertions for chaining
 */
func (da *DatabaseAssertions) AssertNotSoftDeleted(table string, conditions map[string]interface{}) *DatabaseAssertions {
	db := da.tester.Database()
	if db == nil {
		da.tester.Fatalf("❌ Database is not configured")
	}

	isDeleted, err := db.IsSoftDeleted(context.Background(), table, conditions)
	if err != nil {
		da.tester.Fatalf("❌ Soft delete check failed: %v", err)
	}

	if isDeleted {
		da.tester.Errorf("❌ Expected non-deleted record in '%s' with conditions: %v", table, conditions)
		da.debugTableContents(table)
	}

	return da
}

/**
 * AssertDatabaseCount asserts the total number of records in a table.
 *
 * Parameters:
 *   table         - The database table name
 *   expectedCount - Expected number of records
 *
 * Returns:
 *   *DatabaseAssertions for chaining
 */
func (da *DatabaseAssertions) AssertDatabaseCount(table string, expectedCount int) *DatabaseAssertions {
	db := da.tester.Database()
	if db == nil {
		da.tester.Fatalf("❌ Database is not configured")
	}

	count, err := db.Count(context.Background(), table, nil)
	if err != nil {
		da.tester.Fatalf("❌ Count query failed: %v", err)
	}

	if count != expectedCount {
		da.tester.Errorf("❌ Expected %d records in '%s', got %d", expectedCount, table, count)
	}

	return da
}

/**
 * AssertDatabaseCountWhere asserts count of records matching conditions.
 *
 * Parameters:
 *   table         - The database table name
 *   conditions    - Filter conditions
 *   expectedCount - Expected number of matching records
 *
 * Returns:
 *   *DatabaseAssertions for chaining
 */
func (da *DatabaseAssertions) AssertDatabaseCountWhere(table string, conditions map[string]interface{}, expectedCount int) *DatabaseAssertions {
	db := da.tester.Database()
	if db == nil {
		da.tester.Fatalf("❌ Database is not configured")
	}

	count, err := db.Count(context.Background(), table, conditions)
	if err != nil {
		da.tester.Fatalf("❌ Count query failed: %v", err)
	}

	if count != expectedCount {
		da.tester.Errorf("❌ Expected %d records in '%s' with conditions %v, got %d",
			expectedCount, table, conditions, count)
	}

	return da
}

/**
 * AssertDatabaseValue asserts a specific column value for a record.
 *
 * Parameters:
 *   table      - The database table name
 *   conditions - Conditions to find the record
 *   column     - The column to check
 *   expected   - The expected value
 *
 * Returns:
 *   *DatabaseAssertions for chaining
 */
func (da *DatabaseAssertions) AssertDatabaseValue(table string, conditions map[string]interface{}, column string, expected interface{}) *DatabaseAssertions {
	db := da.tester.Database()
	if db == nil {
		da.tester.Fatalf("❌ Database is not configured")
	}

	record, err := db.GetRecord(context.Background(), table, conditions)
	if err != nil {
		da.tester.Fatalf("❌ Failed to get record: %v", err)
	}

	if record == nil {
		da.tester.Errorf("❌ No record found in '%s' with conditions: %v", table, conditions)
		return da
	}

	actual, exists := record[column]
	if !exists {
		da.tester.Errorf("❌ Column '%s' not found in record", column)
		return da
	}

	if !deepEqual(actual, expected) {
		da.tester.Errorf("❌ Column '%s': expected %v, got %v", column, expected, actual)
	}

	return da
}

/**
 * debugTableContents logs table contents for debugging failed assertions.
 *
 * Parameters:
 *   table - The table to debug
 */
func (da *DatabaseAssertions) debugTableContents(table string) {
	db := da.tester.Database()
	if db == nil {
		return
	}

	records, err := db.GetRecords(context.Background(), table, nil, 5)
	if err != nil {
		da.tester.Logf("   Could not fetch table contents: %v", err)
		return
	}

	if len(records) == 0 {
		da.tester.Logf("   Table '%s' is empty", table)
		return
	}

	da.tester.Logf("   Table '%s' contains %d record(s):", table, len(records))
	for i, record := range records {
		if i >= 5 {
			da.tester.Logf("   ... and more")
			break
		}
		da.tester.Logf("   [%d] %v", i+1, record)
	}
}
