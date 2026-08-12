package factory

import (
	"fmt"
	"sync"
)

/**
 * Factory provides Laravel-style model factory functionality.
 *
 * Inspired by Laravel's model factories:
 *   Factory("User").Create()
 *   Factory("User").Create(["name" => "John"])
 *   Factory("User").Times(5).Create()
 *
 * This implementation supports:
 * - In-memory object creation (no database required)
 * - Attribute overrides
 * - Multiple instances via Times()
 * - Named states via State()
 * - Custom definition functions
 */
type Factory struct {
	mu          sync.RWMutex
	definitions map[string]Definition
	states      map[string]map[string]StateFunc
}

/**
 * Definition is a function that generates default attributes for a model.
 * It can optionally receive override attributes.
 */
type Definition func(overrides map[string]interface{}) interface{}

/**
 * StateFunc modifies a model instance for a named state.
 */
type StateFunc func(model interface{}) interface{}

/**
 * NewFactory creates a new factory instance.
 */
func NewFactory() *Factory {
	return &Factory{
		definitions: make(map[string]Definition),
		states:      make(map[string]map[string]StateFunc),
	}
}

/**
 * Define registers a factory definition for a model type.
 *
 * Parameters:
 *   name - The model name (e.g., "User", "Role", "Permission")
 *   fn   - Function that generates default attributes
 *
 * Example:
 *   factory.Define("User", func(overrides map[string]interface{}) interface{} {
 *       return map[string]interface{}{
 *           "name":  faker.Name(),
 *           "email": faker.Email(),
 *       }
 *   })
 */
func (f *Factory) Define(name string, fn Definition) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.definitions[name] = fn
}

/**
 * State registers a state transformation for a model.
 *
 * Parameters:
 *   model - The model name
 *   state - The state name
 *   fn    - Function that applies the state
 *
 * Example:
 *   factory.State("User", "admin", func(model interface{}) interface{} {
 *       m := model.(map[string]interface{})
 *       m["role"] = "admin"
 *       return m
 *   })
 */
func (f *Factory) State(model, state string, fn StateFunc) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.states[model] == nil {
		f.states[model] = make(map[string]StateFunc)
	}
	f.states[model][state] = fn
}

/**
 * Of returns a new FactoryBuilder for the given model.
 *
 * Parameters:
 *   name - The model name
 *
 * Returns:
 *   *FactoryBuilder for chaining
 */
func (f *Factory) Of(name string) *FactoryBuilder {
	f.mu.RLock()
	defer f.mu.RUnlock()

	def, ok := f.definitions[name]
	if !ok {
		panic(fmt.Sprintf("factory: no definition registered for '%s'", name))
	}

	return &FactoryBuilder{
		factory:    f,
		name:       name,
		definition: def,
		count:      1,
		overrides:  make(map[string]interface{}),
		states:     make([]string, 0),
	}
}

/**
 * FactoryBuilder provides a fluent interface for creating model instances.
 */
type FactoryBuilder struct {
	factory    *Factory
	name       string
	definition Definition
	count      int
	overrides  map[string]interface{}
	states     []string
}

/**
 * Times sets the number of instances to create.
 */
func (fb *FactoryBuilder) Times(count int) *FactoryBuilder {
	fb.count = count
	return fb
}

/**
 * Overrides sets custom attribute overrides.
 */
func (fb *FactoryBuilder) Overrides(attrs map[string]interface{}) *FactoryBuilder {
	for k, v := range attrs {
		fb.overrides[k] = v
	}
	return fb
}

/**
 * State applies a named state to the model.
 */
func (fb *FactoryBuilder) State(state string) *FactoryBuilder {
	fb.states = append(fb.states, state)
	return fb
}

/**
 * Create creates and returns model instances.
 *
 * If count is 1, returns a single instance.
 * If count > 1, returns a slice of instances.
 */
func (fb *FactoryBuilder) Create() interface{} {
	if fb.count == 1 {
		model := fb.definition(fb.overrides)
		model = fb.applyStates(model)
		return model
	}

	results := make([]interface{}, fb.count)
	for i := 0; i < fb.count; i++ {
		model := fb.definition(fb.overrides)
		model = fb.applyStates(model)
		results[i] = model
	}
	return results
}

/**
 * Make is an alias for Create (Laravel compatibility).
 */
func (fb *FactoryBuilder) Make() interface{} {
	return fb.Create()
}

/**
 * CreateMany creates multiple instances and returns them as a slice.
 */
func (fb *FactoryBuilder) CreateMany(count int) []interface{} {
	return fb.Times(count).Create().([]interface{})
}

/**
 * applyStates applies all registered states to the model.
 */
func (fb *FactoryBuilder) applyStates(model interface{}) interface{} {
	fb.factory.mu.RLock()
	defer fb.factory.mu.RUnlock()

	modelStates, ok := fb.factory.states[fb.name]
	if !ok {
		return model
	}

	for _, stateName := range fb.states {
		if stateFn, exists := modelStates[stateName]; exists {
			model = stateFn(model)
		}
	}

	return model
}

/**
 * Raw creates a model with the given attributes directly.
 * This bypasses the definition and uses only the provided attributes.
 */
func (fb *FactoryBuilder) Raw(attrs map[string]interface{}) interface{} {
	model := fb.definition(attrs)
	model = fb.applyStates(model)
	return model
}
