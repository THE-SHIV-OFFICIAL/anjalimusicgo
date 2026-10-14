/*
 * TgMusicBot - Telegram Music Bot
 *  Licensed under GNU GPL v3
 */

package db

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Clone describes one cloned bot. The document _id is the Telegram bot ID
// (the number in front of ':' in the bot token).
type Clone struct {
	ID        int64     `bson:"_id"`
	Token     string    `bson:"token"`
	Username  string    `bson:"username"`
	Name      string    `bson:"name"`
	OwnerID   int64     `bson:"owner_id"`
	CreatedAt time.Time `bson:"created_at"`
}

// AddClone inserts or updates a clone record.
func (db *Database) AddClone(c Clone) error {
	ctx, cancel := db.ctx()
	defer cancel()

	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}

	_, err := db.cloneDB.UpdateOne(ctx,
		bson.M{"_id": c.ID},
		bson.M{"$set": bson.M{
			"token":      c.Token,
			"username":   c.Username,
			"name":       c.Name,
			"owner_id":   c.OwnerID,
			"created_at": c.CreatedAt,
		}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

// GetClone returns a clone by bot ID. The bool is false if it does not exist.
func (db *Database) GetClone(botID int64) (*Clone, bool, error) {
	ctx, cancel := db.ctx()
	defer cancel()

	var c Clone
	err := db.cloneDB.FindOne(ctx, bson.M{"_id": botID}).Decode(&c)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &c, true, nil
}

func (db *Database) findClones(filter bson.M) ([]Clone, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := db.cloneDB.Find(ctx, filter, options.Find().SetSort(bson.M{"created_at": 1}))
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var out []Clone
	for cursor.Next(ctx) {
		var c Clone
		if err := cursor.Decode(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, cursor.Err()
}

// GetAllClones returns every cloned bot, oldest first.
func (db *Database) GetAllClones() ([]Clone, error) {
	return db.findClones(bson.M{})
}

// GetUserClones returns the clones owned by a user.
func (db *Database) GetUserClones(ownerID int64) ([]Clone, error) {
	return db.findClones(bson.M{"owner_id": ownerID})
}

// CountClones returns the total number of clones.
func (db *Database) CountClones() (int64, error) {
	ctx, cancel := db.ctx()
	defer cancel()
	return db.cloneDB.CountDocuments(ctx, bson.M{})
}

// DeleteClone removes one clone record.
func (db *Database) DeleteClone(botID int64) error {
	ctx, cancel := db.ctx()
	defer cancel()
	_, err := db.cloneDB.DeleteOne(ctx, bson.M{"_id": botID})
	return err
}

// DeleteAllClones removes every clone record and returns how many were removed.
func (db *Database) DeleteAllClones() (int64, error) {
	ctx, cancel := db.ctx()
	defer cancel()
	res, err := db.cloneDB.DeleteMany(ctx, bson.M{})
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}
