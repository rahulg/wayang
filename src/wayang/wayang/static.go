package wayang

import (
	"errors"
)

type StaticStore struct {
	StaticData Mock
}

var (
	ErrReadOnly = errors.New("Static store is read-only")
)

func NewStaticStore(mock Mock) (s *StaticStore, err error) {
	s = &StaticStore{}
	s.StaticData = mock
	return s, nil
}

func (s *StaticStore) NewMock(uris Mock) (id string, err error) {
	return "", ErrReadOnly
}

func (s *StaticStore) GetEndpoint(id string, url string) (ep Endpoint, err error) {
	ret, ok := s.StaticData[url]
	if ok {
		return ret, nil
	} else {
		return Endpoint{}, ErrNoSuchEndpoint
	}
}

func (s *StaticStore) Close() {
}

func (s *StaticStore) UpdateEndpoint(id string, uris Mock) (err error) {
	s.StaticData = uris
	return nil
}
