package wayang

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"time"
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
	m.collection.Find(bson.M{"_id": bson.ObjectIdHex(id)}).Select(bson.M{"uris": 1}).One(&doc)
	return doc.URIs[url], nil
}

func (m *MongoStore) Close() {
	m.session.Close()
}
