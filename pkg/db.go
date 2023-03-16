package pkg

import (
	"github.com/breadchris/protoflow/pkg/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"path"
)

func functionDB(cacheDir string) error {
	db, err := gorm.Open(sqlite.Open(path.Join(cacheDir, "cache.db")), &gorm.Config{})
	if err != nil {
		return err
	}

	// Migrate the schema
	db.AutoMigrate(&model.Function{})
	db.AutoMigrate(&model.Run{})
	return nil
}
