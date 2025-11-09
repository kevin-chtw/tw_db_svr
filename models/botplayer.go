package models

import (
	"strconv"
	"time"

	"github.com/kevin-chtw/tw_proto/sproto"
)

type BotPlayer struct {
	Model
	Account    string     `gorm:"size:32;uniqueIndex" json:"account"` // 机器人账号
	Nickname   string     `gorm:"size:32" json:"nickname"`            // 昵称
	Avatar     string     `gorm:"size:256" json:"avatar"`             // 头像
	Diamond    int64      `gorm:"default:0" json:"diamond"`           // 钻石
	Vip        int32      `gorm:"default:0" json:"vip"`               // VIP等级
	LeaseTime  *time.Time `gorm:"index" json:"lease_time"`            // 租约时间，为空表示可用
	ExpireTime *time.Time `gorm:"index" json:"expire_time"`           // 租约过期时间
}

// IsAvailable 检查机器人是否可用（未分配或租约已过期）
func (b *BotPlayer) IsAvailable() bool {
	if b.LeaseTime == nil {
		return true
	}
	if b.ExpireTime != nil && time.Now().After(*b.ExpireTime) {
		return true
	}
	return false
}

// AcquireLease 获取租约
func (b *BotPlayer) AcquireLease(duration time.Duration) {
	now := time.Now()
	expireTime := now.Add(duration)
	b.LeaseTime = &now
	b.ExpireTime = &expireTime
}

// ToServerInfoAck 转换为服务器信息响应
func (b *BotPlayer) ToServerInfoAck() *sproto.PlayerInfoAck {
	if b == nil {
		return nil
	}

	return &sproto.PlayerInfoAck{
		Uid:      strconv.FormatUint(uint64(b.ID), 10),
		Nickname: b.Nickname,
		Avatar:   b.Avatar,
		Vip:      b.Vip,
		Diamond:  b.Diamond,
	}
}
