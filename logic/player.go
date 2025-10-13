package logic

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/kevin-chtw/tw_db_svr/models"
	"gorm.io/gorm"
)

// 密码加密
func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

type PlayerDB struct {
	DB *gorm.DB
}

func NewPlayerDB(db *gorm.DB) *PlayerDB {
	return &PlayerDB{DB: db}
}

func (p *PlayerDB) GetPlayerByAccount(account string) (*models.Player, error) {
	var player models.Player
	if err := p.DB.Where("account = ?", account).First(&player).Error; err != nil {
		return nil, err
	}
	return &player, nil
}

// 创建新账号
func (p *PlayerDB) CreatePlayer(player *models.Player) (*models.Player, error) {
	if err := p.DB.Create(player).Error; err != nil {
		return nil, err
	}

	return player, nil
}
