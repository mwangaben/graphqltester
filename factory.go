package graphqltester

import "github.com/mwangaben/graphqltester/pkg/factory"

// ============================================================================
// Factory Operations
// ============================================================================

/**
 * Factory returns a builder for creating model instances.
 *
 * This works like Laravel's model factories:
 *   tester.Factory("User").Create()
 *   tester.Factory("User").Create(map[string]interface{}{"name": "John"})
 *   tester.Factory("User").Times(5).Create()
 *   tester.Factory("Role").Create(map[string]interface{}{"name": "admin"})
 *
 * The factory works WITHOUT a database by default, creating
 * in-memory objects. For database persistence, configure
 * your factory definitions to use GORM or another ORM.
 *
 * Parameters:
 *   name - The model name (e.g., "User", "Role", "Permission")
 *
 * Returns:
 *   *FactoryBuilder for chaining creation options
 *
 * Example:
 *   // Create a user with default attributes
 *   user := tester.Factory("User").Create()
 *
 *   // Create a role with custom name
 *   role := tester.Factory("Role").Create(map[string]interface{}{
 *       "name": "admin",
 *       "guard_name": "api",
 *   })
 *
 *   // Create 5 permissions
 *   permissions := tester.Factory("Permission").Times(5).Create()
 */
func (tester *Tester) Factory(name string) *FactoryBuilder {
	if tester.config.Packages == nil || tester.config.Packages.Factory == nil {
		tester.t.Fatalf("❌ Factory is not configured. Set Packages.Factory in Config.")
	}

	f := tester.config.Packages.Factory

	// Check if it's our Laravel-style factory
	if laravelFactory, ok := f.(*factory.Factory); ok {
		return &FactoryBuilder{
			tester:    tester,
			name:      name,
			factory:   laravelFactory,
			count:     1,
			overrides: make(map[string]interface{}),
		}
	}

	tester.t.Fatalf("❌ Unsupported factory type. Use factory.NewFactory() from pkg/factory.")
	return nil
}

/**
 * FactoryBuilder wraps the factory for Laravel-style fluent API.
 */
type FactoryBuilder struct {
	tester    *Tester
	name      string
	factory   *factory.Factory
	count     int
	overrides map[string]interface{}
	state     string
}

/**
 * Create builds and returns model instances.
 */
func (fb *FactoryBuilder) Create(attrs ...map[string]interface{}) interface{} {
	overrides := fb.overrides
	if overrides == nil {
		overrides = make(map[string]interface{})
	}
	if len(attrs) > 0 && attrs[0] != nil {
		for k, v := range attrs[0] {
			overrides[k] = v
		}
	}

	if fb.tester.config.Debug() {
		fb.tester.t.Logf("🏭 Creating %d %s(s) with overrides: %v", fb.count, fb.name, overrides)
	}

	// Get the factory builder from the underlying factory
	builder := fb.factory.Of(fb.name)

	// Apply state if set
	if fb.state != "" {
		builder.State(fb.state)
	}

	if fb.count == 1 {
		// Apply overrides and create single instance
		return builder.Overrides(overrides).Create()

	}

	// Create multiple instances
	return builder.Overrides(overrides).Times(fb.count).Create()
}

/**
 * Make is an alias for Create.
 */
func (fb *FactoryBuilder) Make(attrs ...map[string]interface{}) interface{} {
	return fb.Create(attrs...)
}

/**
 * Times sets the number of instances to create.
 */
func (fb *FactoryBuilder) Times(count int) *FactoryBuilder {
	fb.count = count
	return fb
}

/**
 * State applies a named state.
 */
func (fb *FactoryBuilder) State(state string) *FactoryBuilder {
	fb.state = state
	return fb
}

/**
 * WithAttributes sets custom attributes.
 */
func (fb *FactoryBuilder) WithAttributes(attrs map[string]interface{}) *FactoryBuilder {
	if fb.overrides == nil {
		fb.overrides = make(map[string]interface{})
	}
	for k, v := range attrs {
		fb.overrides[k] = v
	}
	return fb
}

/**
 * CreateMany creates multiple instances.
 */
func (fb *FactoryBuilder) CreateMany(count int, attrs ...map[string]interface{}) []interface{} {
	return fb.Times(count).Create(attrs...).([]interface{})
}

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
