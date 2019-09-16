package main

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
)

// Model is the database model as well as transfer model
type Model struct {
	ID *primitive.ObjectID `json:"id" bson:"_id,omitempty"`

	Lang      Language  `json:"language,omitempty" bson:"language,omitempty"`
	Source    string    `json:"source,omitempty" bson:"source,omitempty"`
	Date      time.Time `json:"date,omitempty" bson:"date,omitempty"`
	Status    string    `json:"status,omitempty" bson:"status,omitempty"`
	TotalTime uint64    `json:"totalTime,omitempty" bson:"totalTime"`
	MaxMemory uint64    `json:"maxMemory,omitempty" bson:"maxMemory"`
	Results   []Result  `json:"results,omitempty" bson:"results"`
}

// Language defines the way to compile / run
type Language struct {
	Name           string `json:"name" bson:"name"`
	SourceFileName string `json:"sourceFileName" bson:"sourceFileName"`
	CompileCmd     string `json:"compileCmd" bson:"compileCmd"`
	Executables    string `json:"executables" bson:"executables"`
	RunCmd         string `json:"runCmd" bson:"runCmd"`
}

// Result is the judger updates
type Result struct {
	Time   uint64 `json:"time,omitempty" bson:"time,omitempty"`
	Memory uint64 `json:"memory,omitempty" bson:"memory,omitempty"`
	Stdin  string `json:"stdin,omitempty" bson:"stdin,omitempty"`
	Stdout string `json:"stdout,omitempty" bson:"stdout,omitempty"`
	Stderr string `json:"stderr,omitempty" bson:"stderr,omitempty"`
	Log    string `json:"log,omitempty" bson:"log,omitempty"`
}

// JudgerUpdate is judger submitted updates
type JudgerUpdate struct {
	ID *primitive.ObjectID `json:"id" bson:"_id,omitempty"`

	Type     string    `json:"type"`
	Status   string    `json:"status"`
	Date     time.Time `json:"date,omitempty"`
	Language string    `json:"language"`
	Results  []Result  `json:"results,omitempty"`
}

type db struct {
	client   *mongo.Client
	database *mongo.Database
}

const (
	colName         = "submission3"
	defaultURI      = "mongodb://localhost:27017/admin"
	defaultDatabase = "test1"
	envMongoURI     = "MONGODB_URI"
)

func getDB() *db {
	uri := defaultURI
	database := defaultDatabase
	if u := os.Getenv(envMongoURI); u != "" {
		uri = u
		con, err := connstring.Parse(u)
		if err != nil {
			log.Fatal(err)
		}
		database = con.Database
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(uri).SetRetryWrites(false))
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err = client.Connect(ctx)

	if err != nil {
		log.Fatalln(err)
		return nil
	}
	return &db{
		client:   client,
		database: client.Database(database),
	}
}

func (d *db) Add(cs *ClientSubmit) (*Model, error) {
	c := d.database.Collection(colName)
	m := &Model{
		Lang:   cs.Lang,
		Source: cs.Source,
		Date:   time.Now(),
	}
	i, err := c.InsertOne(nil, m)
	if err != nil {
		return nil, err
	}
	id := i.InsertedID.(primitive.ObjectID)
	m.ID = &id
	return m, nil
}

func (d *db) Update(m *JudgerUpdate) (*JudgerUpdate, error) {
	c := d.database.Collection(colName)

	filter := bson.D{{Key: "_id", Value: m.ID}}
	update := bson.D{
		{Key: "status", Value: m.Status},
		{Key: "results", Value: m.Results},
	}
	updateCmd := bson.D{
		{Key: "$set", Value: update},
	}

	_, err := c.UpdateOne(nil, filter, updateCmd)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (d *db) Query(id string) ([]Model, error) {
	c := d.database.Collection(colName)

	findOption := options.Find()
	findOption.SetLimit(10)
	findOption.SetSort(bson.D{{Key: "_id", Value: -1}})

	filter := bson.D{}
	if len(id) > 0 {
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return nil, err
		}
		filter = append(filter, bson.E{
			Key:   "_id",
			Value: bson.D{{Key: "$lt", Value: oid}},
		})
	}

	cursor, err := c.Find(nil, filter, findOption)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(nil)

	rt := make([]Model, 0, 10)
	for cursor.Next(nil) {
		el := Model{}
		if err = cursor.Decode(&el); err != nil {
			return nil, err
		}
		rt = append(rt, el)
	}
	return rt, nil
}
