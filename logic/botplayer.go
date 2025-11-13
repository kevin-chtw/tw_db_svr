package logic

import (
	"time"

	"github.com/kevin-chtw/tw_db_svr/models"
	"gorm.io/gorm"
)

type BotPlayerDB struct {
	DB *gorm.DB
}

func NewBotPlayerDB(db *gorm.DB) *BotPlayerDB {
	return &BotPlayerDB{DB: db}
}

// GetBotPlayer 获取botplayer（如果没有就自动创建）
func (b *BotPlayerDB) GetBotPlayer(leaseDuration time.Duration) (*models.BotPlayer, error) {
	var botPlayer models.BotPlayer

	// 查询可用的botplayer（租约为空或已过期）
	err := b.DB.Where("(lease_time IS NULL OR expire_time < ?) AND id > ?", time.Now(), 20).First(&botPlayer).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 如果找到可用的botplayer，设置新租约
	if err == nil {
		botPlayer.AcquireLease(leaseDuration)
		if err := b.DB.Save(&botPlayer).Error; err != nil {
			return nil, err
		}
		return &botPlayer, nil
	}

	// 如果没有可用的botplayer，创建新的
	newBot := &models.BotPlayer{
		Account:  "bot_" + time.Now().Format("20060102150405"),
		Nickname: "机器人_" + time.Now().Format("1504"),
		Avatar:   "/avatars/bot_default.png",
		Diamond:  1000,
		Vip:      0,
	}
	newBot.AcquireLease(leaseDuration)

	if err := b.DB.Create(newBot).Error; err != nil {
		return nil, err
	}

	return newBot, nil
}

// GetBotPlayerByID 根据ID获取botplayer
func (b *BotPlayerDB) GetBotPlayerByID(uid string) (*models.BotPlayer, error) {
	var botPlayer models.BotPlayer
	err := b.DB.Where("id = ?", uid).First(&botPlayer).Error
	if err != nil {
		return nil, err
	}
	return &botPlayer, nil
}
