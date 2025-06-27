package models

import (
	"fmt"

	"gorm.io/gorm"
)

type Player struct {
	Model
	Account  string `gorm:"size:32;index:,unique,composite:account_email_phone" json:"account"` // 账号
	Email    string `gorm:"size:64;index:,unique,composite:account_email_phone" json:"email"`   // 邮箱
	Phone    string `gorm:"size:16;index:,unique,composite:account_email_phone" json:"phone"`   // 手机号
	Pwd      string `gorm:"size:64" json:"-"`
	Nickname string `gorm:"size:32" json:"nickname"`
	Abstract string `gorm:"size:128" json:"abstract"` // 简介
	Avatar   string `gorm:"size:256" json:"avatar"`   // 头像
	IP       string `gorm:"size:32" json:"ip"`        // IP
	Addr     string `gorm:"size:64" json:"addr"`      // 地址
}

func (p *Player) BeforeCreate(tx *gorm.DB) error {
	if p.Account == "" && p.Email == "" && p.Phone == "" {
		return fmt.Errorf("at least one of account, email or phone must be provided")
	}
	return nil
}
