package server

import (
    "fmt"
	"time"
    "context"
    "errors"

    "crypto/hmac"
	"crypto/sha256"
	"crypto/rand"
    "golang.org/x/crypto/bcrypt"
    "encoding/base64"

    "go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mopts "go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserDoc struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Username  string             `bson:"username"`
	PassHash  string             `bson:"passHash"`
	CreatedAt time.Time          `bson:"createdAt"`
}

type SessionDoc struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	AuthKey    string             `bson:"authKey"`
	Username   string             `bson:"username"`
	CreatedAt  time.Time          `bson:"createdAt"`
	LastUsedAt time.Time          `bson:"lastUsedAt"`
	ExpiresAt  time.Time          `bson:"expiresAt"`
}

type HistoryDoc struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Username string             `bson:"username"`
	At       time.Time          `bson:"at"`
	Expr     string             `bson:"expr"`
	Result   string             `bson:"result"`
}

type Storage struct {
	db       *mongo.Database
	users    *mongo.Collection
	sessions *mongo.Collection
	history  *mongo.Collection
	secret   []byte
}

func IndexUnique(col *mongo.Collection, ctx context.Context, key string) error {
    _, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
    	Keys:    bson.D{{Key: key, Value: 1}},
    	Options: mopts.Index().SetUnique(true),
    })
    if err != nil {
    	return fmt.Errorf("Couldn't create %s index: %w", key, err)
    }
    return nil
}

func NewStore(ctx context.Context, mongoURI, dbName string,
              secret []byte) (sdb Storage, closeFn func(context.Context) error, err error) {
	var client *mongo.Client
    client, err = mongo.Connect(ctx, mopts.Client().ApplyURI(mongoURI))
	if err != nil {	return }
	closeFn = func(c context.Context) error { return client.Disconnect(c) }

	db := client.Database(dbName)
	sdb = Storage{
		db:       db,
		users:    db.Collection("users"),
		sessions: db.Collection("sessions"),
		history:  db.Collection("history"),
		secret:   secret,
	}

	err = IndexUnique(sdb.users, ctx, "username")
    if err != nil {
        closeFn(ctx)
        return
    }

    err = IndexUnique(sdb.sessions, ctx, "authKey")
    if err != nil {
        closeFn(ctx)
        return
    }

	// Mongo will delete docs when expiresAt is older than "now"
    //   * TTL monitor runs periodically
	_, err = sdb.sessions.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "expiresAt", Value: 1}},
		Options: mopts.Index().
			SetExpireAfterSeconds(0),
	})
	if err != nil {
		closeFn(ctx)
		return
	}

	_, err = sdb.history.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "username", Value: 1},
			{Key: "at", Value: -1},
		},
	})
	if err != nil { closeFn(ctx) }

	return
}

func HashPassword(pass string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}


func (db *Storage) CreateUser(ctx context.Context, username, pass string) error {
	hash, err := HashPassword(pass)
	if err != nil {
		return err
	}
	doc := UserDoc{
		Username:  username,
		PassHash:  hash,
		CreatedAt: time.Now(),
	}
	_, err = db.users.InsertOne(ctx, doc)
	if err != nil {
		var we mongo.WriteException
		if errors.As(err, &we) {
			for _, e := range we.WriteErrors {
				if e.Code == 11000 {
					return fmt.Errorf("username already exists")
				}
			}
		}
		return err
	}
	return nil
}

func (db *Storage) VerifyUser(ctx context.Context, username, pass string) (exists bool, err error) {
	var u UserDoc
    err    = db.users.FindOne(ctx, bson.M{"username": username}).Decode(&u)
    exists = err == nil
    if exists {
        err = bcrypt.CompareHashAndPassword([]byte(u.PassHash), []byte(pass))
        if err == bcrypt.ErrMismatchedHashAndPassword {
            err = fmt.Errorf("Invalid password")
        }
	} else if errors.Is(err, mongo.ErrNoDocuments) {
	    err = nil
    }
	return
}

func (db *Storage) MakeAuthKey(username string) string {
	issued := time.Now().UTC().Format(time.RFC3339Nano)
	nonce := make([]byte, 18)
	rand.Read(nonce)

	msg := username + "|" + issued + "|" + base64.RawURLEncoding.EncodeToString(nonce)
	mac := hmac.New(sha256.New, db.secret)
	mac.Write([]byte(msg))
	sum := mac.Sum(nil)

	return username + "." + base64.RawURLEncoding.EncodeToString(sum)
}

func (db *Storage) CreateSession(
    ctx context.Context,
    username string,
    authTTL time.Duration,
) (authKey string, expiresAt time.Time, err error) {
	authKey   = db.MakeAuthKey(username)
    now      := time.Now()
    expiresAt = now.Add(authTTL)
    _, err = db.sessions.InsertOne(ctx, SessionDoc{
		AuthKey:    authKey,
		Username:   username,
		CreatedAt:  now,
		LastUsedAt: now,
		ExpiresAt:  expiresAt,
	})
	return
}

// Ensure session exists and not expired; update lastUsedAt.
func (db *Storage) TouchSession(ctx context.Context, authKey string) (sess SessionDoc, exists bool, err error) {
	now := time.Now()
	filter := bson.M{
		"authKey":    authKey,
		"expiresAt":  bson.M{"$gt": now},
	}
	update := bson.M{
		"$set": bson.M{"lastUsedAt": now},
	}

	opts  := mopts.FindOneAndUpdate().SetReturnDocument(mopts.After)
	err    = db.sessions.FindOneAndUpdate(ctx, filter, update, opts).Decode(&sess)
	exists = err == nil
    if !exists && errors.Is(err, mongo.ErrNoDocuments) {
		err = nil
	}
	return
}

func (db *Storage) DeleteSession(ctx context.Context, authKey string) error {
	_, err := db.sessions.DeleteOne(ctx, bson.M{"authKey": authKey})
	return err
}

func (db *Storage) AppendHistory(ctx context.Context, username, expr, result string) error {
	_, err := db.history.InsertOne(ctx, HistoryDoc{
		Username: username,
		At:       time.Now(),
		Expr:     expr,
		Result:   result,
	})
	return err
}

func (db *Storage) GetHistory(ctx context.Context, username string,
                              limit int64) (docs []HistoryDoc, err error) {
    var cur *mongo.Cursor
	cur, err = db.history.Find(ctx,
		bson.M{"username": username},
		mopts.Find().SetSort(bson.D{{Key: "at", Value: -1}}).SetLimit(limit),
	)
	if err != nil { return }
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var h HistoryDoc
        err = cur.Decode(&h)
		if err != nil { return }
		docs = append(docs, h)
	}
    err = cur.Err()
	return
}
