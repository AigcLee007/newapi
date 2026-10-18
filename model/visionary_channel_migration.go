package model

import (
	"errors"

	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const legacyVisionaryChannelTypeMigrationKey = "migration.channel_type.visionary_58_to_64"

func migrateLegacyVisionaryChannelType(db *gorm.DB) error {
	if db == nil {
		return errors.New("migrate legacy Visionary channel type: database is nil")
	}
	if !db.Migrator().HasTable(&ImageAsyncIdempotency{}) || db.Migrator().HasTable(&TaskPlugin{}) {
		return nil
	}
	if !db.Migrator().HasTable(&Channel{}) {
		return nil
	}
	if !db.Migrator().HasTable(&Option{}) {
		if err := db.AutoMigrate(&Option{}); err != nil {
			return err
		}
	}

	return db.Transaction(func(tx *gorm.DB) error {
		var marker Option
		err := tx.Where(&Option{Key: legacyVisionaryChannelTypeMigrationKey}).First(&marker).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Model(&Channel{}).
			Where("type = ?", constant.ChannelTypeAdvancedCustom).
			Update("type", constant.ChannelTypeVisionary).Error; err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&Option{
			Key:   legacyVisionaryChannelTypeMigrationKey,
			Value: "1",
		}).Error
	})
}
