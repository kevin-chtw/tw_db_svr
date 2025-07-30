package logic

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/kevin-chtw/tw_db_svr/models"
	"gorm.io/gorm"
)

type PlayerDB struct {
	DB *gorm.DB
}

func NewPlayerDB(db *gorm.DB) *PlayerDB {
	return &PlayerDB{DB: db}
}

// 密码加密
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// 检查账号是否存在
func (p *PlayerDB) AccountExists(account string) (bool, error) {
	var count int64
	err := p.DB.Model(&models.Player{}).
		Where("account = ? OR email = ? OR phone = ?", account, account, account).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// 创建新账号
func (p *PlayerDB) CreateAccount(account, password string) (*models.Player, error) {
	// 检查账号是否已存在
	if exists, err := p.AccountExists(account); err != nil {
		return nil, err
	} else if exists {
		return nil, fmt.Errorf("account already exists")
	}

	// 创建新玩家记录
	player := &models.Player{
		Account: account,
		Pwd:     hashPassword(password),
	}

	if err := player.BeforeCreate(p.DB); err != nil {
		return nil, err
	}

	if err := p.DB.Create(player).Error; err != nil {
		return nil, err
	}

	return player, nil
}

// 验证账号密码
func (p *PlayerDB) VerifyAccount(account, password string) (*models.Player, error) {
	var player models.Player

	// 优先通过账号查找
	result := p.DB.Where("account = ?", account).First(&player)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 尝试通过邮箱查找
			result = p.DB.Where("email = ?", account).First(&player)
			if result.Error != nil {
				if errors.Is(result.Error, gorm.ErrRecordNotFound) {
					// 尝试通过手机号查找
					result = p.DB.Where("phone = ?", account).First(&player)
					if result.Error != nil {
						return nil, fmt.Errorf("account not found")
					}
				} else {
					return nil, result.Error
				}
			}
		} else {
			return nil, result.Error
		}
	}

	// 验证密码
	if player.Pwd != hashPassword(password) {
		return nil, fmt.Errorf("invalid password")
	}

	return &player, nil
}

// 获取玩家房卡数量
func (p *PlayerDB) GetRoomCards(userid string) (int, error) {
	var player models.Player
	result := p.DB.Select("room_cards").Where("id = ?", userid).First(&player)
	if result.Error != nil {
		return 0, result.Error
	}
	return player.RoomCards, nil
}

// 扣除玩家房卡
func (p *PlayerDB) DeductRoomCards(userid string, count int) error {
	return p.DB.Model(&models.Player{}).
		Where("id = ? AND room_cards >= ?", userid, count).
		Update("room_cards", gorm.Expr("room_cards - ?", count)).
		Error
}
