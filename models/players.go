package models

import (
	"fmt"
	"strconv"

	"github.com/kevin-chtw/tw_proto/cproto"
	"github.com/kevin-chtw/tw_proto/sproto"
	"gorm.io/gorm"
)

type Player struct {
	Model
	Account  string `gorm:"size:32;index:,unique,composite:account_wechat_phone" json:"account"` // 账号
	Wechat   string `gorm:"size:64;index:,unique,composite:account_wechat_phone" json:"wechat"`  // 邮箱
	Phone    string `gorm:"size:16;index:,unique,composite:account_wechat_phone" json:"phone"`   // 手机号
	Pwd      string `gorm:"size:64" json:"-"`
	Nickname string `gorm:"size:32" json:"nickname"`
	Abstract string `gorm:"size:128" json:"abstract"` // 简介
	Avatar   string `gorm:"size:256" json:"avatar"`   // 头像
	IP       string `gorm:"size:32" json:"ip"`        // IP
	Addr     string `gorm:"size:64" json:"addr"`      // 地址
	Diamond  int64  `gorm:"default:0" json:"daimond"` // 钻石
	Coin     int64  `gorm:"default:0" json:"coin"`    // 金币
	Vip      int32  `gorm:"default:0" json:"vip"`     // VIP等级
}

func (p *Player) BeforeCreate(tx *gorm.DB) error {
	if p.Account == "" && p.Wechat == "" && p.Phone == "" {
		return fmt.Errorf("at least one of account, wechat or phone must be provided")
	}
	return nil
}

func (p *Player) ToPlayerInfoAck() *cproto.PlayerInfoAck {
	if p == nil {
		return nil
	}

	return &cproto.PlayerInfoAck{
		Uid:      strconv.FormatUint(uint64(p.ID), 10),
		Nickname: p.Nickname,
		Avatar:   p.Avatar,
		Vip:      int32(p.Vip),
		Diamond:  p.Diamond,
		Coin:     p.Coin,
	}
}

func (p *Player) ToServerInfoAck() *sproto.PlayerInfoAck {
	if p == nil {
		return nil
	}

	return &sproto.PlayerInfoAck{
		Uid:      strconv.FormatUint(uint64(p.ID), 10),
		Nickname: p.Nickname,
		Avatar:   p.Avatar,
		Vip:      int32(p.Vip),
		Diamond:  p.Diamond,
		Coin:     p.Coin,
	}
}
