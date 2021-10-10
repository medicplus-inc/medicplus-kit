package database

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const QUERY_KEY = "QUERY_KEY_DB_GORM"

func NewQueryContext(ctx context.Context, db *gorm.DB) context.Context {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", time.Now().UTC())))
	privateKey := fmt.Sprintf("%x", h.Sum(nil))
	ctx = context.WithValue(ctx, QUERY_KEY, privateKey)
	return context.WithValue(ctx, privateKey, db)
}

func QueryFromContext(ctx context.Context) (*gorm.DB, bool) {
	privateKey, ok := ctx.Value(QUERY_KEY).(string)
	if !ok {
		return nil, ok
	}

	db, ok := ctx.Value(privateKey).(*gorm.DB)
	return db, ok
}

type TransactionCallback func(ctx context.Context) error

func RunInTransaction(ctx context.Context, db *gorm.DB, fn TransactionCallback) error {
	tx := db.WithContext(ctx)
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	ctx = NewQueryContext(ctx, tx)
	err := fn(ctx)
	if nil != err {
		tx.Rollback()
		return err
	}

	tx.Commit()

	return nil
}
