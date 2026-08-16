package modules

// SettingsStore is the minimal persistence contract State needs — matched
// structurally by *store.SQLiteStore's GetSetting/SetSetting without
// internal/modules importing internal/store (kept decoupled the same way
// internal/notify's ResolveCredentials takes a concrete store instead of
// this package reaching for one itself).
type SettingsStore interface {
	GetSetting(key string) (value string, ok bool, err error)
	SetSetting(key, value string) error
}

// moduleState namespaces a shared SettingsStore under "module.<name>." so
// two modules (or a module and something unrelated) can never collide on a
// key.
type moduleState struct {
	store SettingsStore
	name  string
}

// NewState builds a Module's private State, backed by store and namespaced
// to moduleName.
func NewState(store SettingsStore, moduleName string) State {
	return &moduleState{store: store, name: moduleName}
}

func (s *moduleState) key(k string) string { return "module." + s.name + "." + k }

func (s *moduleState) Get(key string) (string, bool, error) {
	return s.store.GetSetting(s.key(key))
}

func (s *moduleState) Set(key, value string) error {
	return s.store.SetSetting(s.key(key), value)
}
