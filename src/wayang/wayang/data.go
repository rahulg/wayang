package wayang

type Endpoint map[string]map[string]interface{}
type Mock map[string]Endpoint

type DataStore interface {
	NewMock(Mock) (string, error)
	GetEndpoint(string, string) (Endpoint, error)
	Close()
}
