package store

type Store interface {
	Close() error
}

type NoopStore struct{}

func NewNoopStore() *NoopStore {
	return &NoopStore{}
}

func (s *NoopStore) Close() error {
	return nil
}
