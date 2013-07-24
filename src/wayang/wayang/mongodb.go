package wayang

import (
	"errors"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"time"
)

var (
	ErrNotImplemented = errors.New("Method not implemented")
)

type MongoStore struct {
	session    *mgo.Session
	db         *mgo.Database
	collection *mgo.Collection
}

type MongoDocument struct {
	ID        bson.ObjectId `bson:"_id"`
	Timestamp time.Time     `bson:"timestamp"`
	URIs      Mock          `bson:"uris"`
}

var expiry mgo.Index

func init() {
	expiry = mgo.Index{
		Key:         []string{"timestamp"},
		ExpireAfter: 24 * time.Hour,
	}
}

func NewMongoStore(addr string) (m *MongoStore, err error) {
	m = &MongoStore{}
	m.session, _ = mgo.Dial(addr)
	m.session.SetMode(mgo.Monotonic, true)
	m.db = m.session.DB("wayang")
	m.collection = m.db.C("endpoints")
	return m, nil
}

func (m *MongoStore) NewMock(uris Mock) (id string, err error) {
	doc := MongoDocument{
		bson.NewObjectId(),
		time.Now(),
		uris,
	}
	m.collection.Insert(&doc)
	m.collection.EnsureIndex(expiry)
	return doc.ID.Hex(), nil
}

func (m *MongoStore) GetEndpoint(id string, url string) (ep Endpoint, err error) {
	doc := MongoDocument{}
	err = m.collection.Find(bson.M{"_id": bson.ObjectIdHex(id)}).Select(bson.M{"uris": 1}).One(&doc)
	if err != nil {
		return Endpoint{}, err
	} else {
		return doc.URIs[url], nil
	}
}

func (m *MongoStore) UpdateEndpoint(id string, uris Mock) error {
	return ErrNotImplemented
}

func (m *MongoStore) Close() {
	m.session.Close()
}
