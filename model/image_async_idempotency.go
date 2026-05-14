package model

import (
	"time"

	"gorm.io/gorm"
)

type ImageAsyncIdempotency struct {
	ID          int64 `gorm:"primaryKey"`
	CreatedAt   int64 `gorm:"index"`
	UpdatedAt   int64
	UserId      int    `gorm:"uniqueIndex:idx_image_async_idem_user_key;index;not null"`
	Key         string `gorm:"type:varchar(191);uniqueIndex:idx_image_async_idem_user_key;not null"`
	RequestHash string `gorm:"type:varchar(64);not null"`
	TaskID      string `gorm:"type:varchar(191);not null;index"`
}

func (i *ImageAsyncIdempotency) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	i.CreatedAt = now
	i.UpdatedAt = now
	return nil
}

func (i *ImageAsyncIdempotency) BeforeUpdate(tx *gorm.DB) error {
	i.UpdatedAt = time.Now().Unix()
	return nil
}

func GetImageAsyncIdempotency(userID int, key string) (*ImageAsyncIdempotency, bool, error) {
	if key == "" {
		return nil, false, nil
	}
	var item ImageAsyncIdempotency
	err := DB.Where(&ImageAsyncIdempotency{UserId: userID, Key: key}).First(&item).Error
	exist, err := RecordExist(err)
	if err != nil {
		return nil, false, err
	}
	return &item, exist, nil
}
