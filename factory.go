package graphqltester

/**
 * Factory Integration for Test Data Generation
 *
 * This file provides integration with your mwangaben/factory package,
 * allowing seamless creation of test data within GraphQL tests.
 *
 * The factory integration follows the same patterns as Laravel's model factories:
 * - Factory("Model").Create() - Creates and persists a model
 * - Factory("Model").Make() - Creates a model without persisting
 * - Factory("Model").CreateMany(5) - Creates multiple models
 *
 * Factory definitions are registered once and can be used throughout tests
 * with override attributes for specific scenarios.
 */

// ============================================================================
// Factory Methods
// ============================================================================

/**
 * Factory returns a factory builder for the specified model.
 *
 * This is the entry point for creating test data. It returns a builder
 * that can be used to create or make model instances.
 *
 * Parameters:
 *   name - The factory definition name (e.g., "User", "Zone", "Post")
 *
 * Returns:
 *   *FactoryBuilder for chaining creation options
 *
 * Example:
 *   // Create a user with default attributes
 *   user := tester.Factory("User").Create()
 *
 *   // Create a user with specific attributes
 *   admin := tester.Factory("User").Create(map[string]interface{}{
 *       "role": "admin",
 *       "email": "admin@example.com",
 *   })
 *
 *   // Make a user without persisting
 *   unsavedUser := tester.Factory("User").Make()
 *
 *   // Create multiple users
 *   users := tester.Factory("User").CreateMany(5)
 *
 * Integration:
 *   Requires mwangaben/factory package to be configured in PackageConfig.
 */
func (tester *Tester) Factory(name string) *FactoryBuilder {
	if tester.config.Packages.Factory == nil {
		tester.t.Fatal("❌ Factory package is not configured. Set it in PackageConfig.")
	}

	return &FactoryBuilder{
		tester: tester,
		name:   name,
		count:  1,
	}
}

/**
 * FactoryBuilder provides a fluent interface for creating test data.
 *
 * It supports creating single or multiple instances, with or without
 * persistence, and with attribute overrides.
 */
type FactoryBuilder struct {
	tester     *Tester
	name       string
	count      int
	attributes map[string]interface{}
	state      string
}

/**
 * Create creates and persists model instances based on the factory definition.
 *
 * This method calls the factory to generate attributes, creates the model
 * in the database, and returns the created instance(s).
 *
 * If a single instance is created (default), returns the model directly.
 * If multiple instances are created (via Times()), returns a slice of models.
 *
 * Parameters:
 *   attrs - Optional attribute overrides for the factory definition
 *
 * Returns:
 *   The created model instance or slice of instances
 *
 * Example:
 *   // Create with default attributes
 *   zone := tester.Factory("Zone").Create().(*models.Zone)
 *
 *   // Create with overrides
 *   zone := tester.Factory("Zone").Create(map[string]interface{}{
 *       "name": "Custom Zone",
 *       "code": "CUS001",
 *   }).(*models.Zone)
 *
 * Note:
 *   Type assertion may be needed depending on your factory's return type.
 */
func (fb *FactoryBuilder) Create(attrs ...map[string]interface{}) interface{} {
	// Merge attributes
	attributes := fb.attributes
	if len(attrs) > 0 && attrs[0] != nil {
		if attributes == nil {
			attributes = make(map[string]interface{})
		}
		for k, v := range attrs[0] {
			attributes[k] = v
		}
	}

	// Use the factory package to create instances
	// This is a placeholder - actual implementation depends on your factory package API

	if fb.tester.config.debugEnabled {
		fb.tester.t.Logf("🏭 Creating %d %s(s) with attributes: %v", fb.count, fb.name, attributes)
	}

	// Placeholder return
	return nil
}

/**
 * Make creates model instances without persisting to the database.
 *
 * This is useful for testing scenarios where you need a model instance
 * but don't want to save it (e.g., testing validation on unsaved models).
 *
 * Parameters:
 *   attrs - Optional attribute overrides
 *
 * Returns:
 *   The created model instance(s) without database persistence
 *
 * Example:
 *   unsavedZone := tester.Factory("Zone").Make().(*models.Zone)
 *   // unsavedZone is not in the database
 */
func (fb *FactoryBuilder) Make(attrs ...map[string]interface{}) interface{} {
	// Similar to Create but doesn't persist
	if fb.tester.config.debugEnabled {
		fb.tester.t.Logf("🏭 Making %d %s(s) (without persisting)", fb.count, fb.name)
	}

	return nil
}

/**
 * Times specifies how many instances to create.
 *
 * Parameters:
 *   count - Number of instances to create
 *
 * Returns:
 *   *FactoryBuilder for chaining
 *
 * Example:
 *   // Create 5 zones
 *   zones := tester.Factory("Zone").Times(5).Create()
 *
 *   // Create 3 users with specific role
 *   users := tester.Factory("User").Times(3).Create(map[string]interface{}{
 *       "role": "editor",
 *   })
 */
func (fb *FactoryBuilder) Times(count int) *FactoryBuilder {
	fb.count = count
	return fb
}

/**
 * State applies a named state transformation to the factory.
 *
 * States are predefined attribute modifications (e.g., "active", "deleted",
 * "verified") that can be applied to factory definitions.
 *
 * Parameters:
 *   state - The state name to apply
 *
 * Returns:
 *   *FactoryBuilder for chaining
 *
 * Example:
 *   // Create a deleted zone
 *   deletedZone := tester.Factory("Zone").State("deleted").Create()
 *
 *   // Create a verified admin user
 *   admin := tester.Factory("User").State("verified").State("admin").Create()
 */
func (fb *FactoryBuilder) State(state string) *FactoryBuilder {
	fb.state = state
	return fb
}

/**
 * WithAttributes sets custom attributes for the factory.
 *
 * Alternative to passing attributes directly to Create/Make.
 * Useful for building up attributes incrementally.
 *
 * Parameters:
 *   attrs - Attribute overrides
 *
 * Returns:
 *   *FactoryBuilder for chaining
 *
 * Example:
 *   tester.Factory("Zone").
 *       WithAttributes(map[string]interface{}{
 *           "status": "ACTIVE",
 *       }).
 *       Create(map[string]interface{}{
 *           "name": "Active Zone",
 *       })
 */
func (fb *FactoryBuilder) WithAttributes(attrs map[string]interface{}) *FactoryBuilder {
	if fb.attributes == nil {
		fb.attributes = make(map[string]interface{})
	}
	for k, v := range attrs {
		fb.attributes[k] = v
	}
	return fb
}

/**
 * CreateMany is a convenience method for creating multiple instances.
 *
 * Equivalent to Times(n).Create(attrs...)
 *
 * Parameters:
 *   count - Number of instances to create
 *   attrs - Optional attribute overrides
 *
 * Returns:
 *   Slice of created model instances
 *
 * Example:
 *   zones := tester.Factory("Zone").CreateMany(5)
 */
func (fb *FactoryBuilder) CreateMany(count int, attrs ...map[string]interface{}) []interface{} {
	return fb.Times(count).Create(attrs...).([]interface{})
}

/**
 * Seed runs a seeder function to populate the database with test data.
 *
 * Useful for setting up complex test scenarios with related data.
 *
 * Parameters:
 *   seeder - Function that creates test data
 *
 * Returns:
 *   *Tester for fluent method chaining
 *
 * Example:
 *   tester.Seed(func() {
 *       tester.Factory("Role").Create(map[string]interface{}{
 *           "name": "admin",
 *       })
 *       tester.Factory("Permission").Create(map[string]interface{}{
 *           "name": "users.manage",
 *       })
 *   })
 */
func (tester *Tester) Seed(seeder func()) *Tester {
	if tester.config.debugEnabled {
		tester.t.Logf("🌱 Running seeder...")
	}

	seeder()

	if tester.config.debugEnabled {
		tester.t.Logf("✅ Seeder completed")
	}

	return tester
}
