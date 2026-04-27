package gemrouter

type ContextStore struct {
	RequestID string
	UserID    string
	data      map[string]any
}

func (store *ContextStore) Set(key string, val any) {
	if store.data == nil {
		store.data = make(map[string]any, 4)
	}
	store.data[key] = val
}

func (store *ContextStore) Get(key string) (any, bool) {
	if store.data == nil {
		return nil, false
	}
	v, ok := store.data[key]
	return v, ok
}
