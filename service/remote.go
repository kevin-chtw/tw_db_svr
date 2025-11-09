package service

import (
	"context"
	"errors"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/kevin-chtw/tw_common/utils"
	"github.com/kevin-chtw/tw_db_svr/logic"
	"github.com/kevin-chtw/tw_proto/sproto"
	pitaya "github.com/topfreegames/pitaya/v3/pkg"
	"github.com/topfreegames/pitaya/v3/pkg/component"
	"github.com/topfreegames/pitaya/v3/pkg/logger"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"gorm.io/gorm"
)

// Remote 独立的匹配服务
type Remote struct {
	component.Base
	db       *gorm.DB
	app      pitaya.Pitaya
	handlers map[string]func(context.Context, bool, proto.Message) (proto.Message, error)
}

func NewRemote(db *gorm.DB, app pitaya.Pitaya) *Remote {
	return &Remote{
		db:       db,
		app:      app,
		handlers: make(map[string]func(context.Context, bool, proto.Message) (proto.Message, error)),
	}
}

// Init 组件初始化
func (m *Remote) Init() {
	m.handlers[utils.TypeUrl(&sproto.ChangeDiamondReq{})] = m.changeDiamondReq
	m.handlers[utils.TypeUrl(&sproto.PlayerInfoReq{})] = m.playerInfoReq
	m.handlers[utils.TypeUrl(&sproto.GetBotReq{})] = m.getBotPlayerReq
}

func (m *Remote) Message(ctx context.Context, req *sproto.AccountReq) (*sproto.AccountAck, error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Errorf("panic recovered %s\n %s", r, string(debug.Stack()))
		}
	}()
	logger.Log.Infof("match: %v", req)

	msg, err := req.Req.UnmarshalNew()
	if err != nil {
		return nil, err
	}

	if handler, ok := m.handlers[req.Req.TypeUrl]; ok {
		rsp, err := handler(ctx, req.Bot, msg)
		if err != nil {
			return nil, err
		}
		return m.newAccountAck(req.Bot, rsp)
	}

	return &sproto.AccountAck{}, nil
}

func (m *Remote) newAccountAck(bot bool, msg proto.Message) (*sproto.AccountAck, error) {
	data, err := anypb.New(msg)
	if err != nil {
		return nil, err
	}
	return &sproto.AccountAck{Bot: bot, Ack: data}, nil
}

func (m *Remote) changeDiamondReq(ctx context.Context, bot bool, msg proto.Message) (proto.Message, error) {
	req := msg.(*sproto.ChangeDiamondReq)

	var playerDiamond int64

	if bot {
		// 处理bot玩家
		botPlayer, err := logic.NewBotPlayerDB(m.db).GetBotPlayerByID(req.Uid)
		if err != nil {
			return nil, err
		}

		if botPlayer.Diamond+req.Diamond < 0 {
			return nil, errors.New("diamond is not enough")
		}

		botPlayer.Diamond += req.Diamond
		if err := logic.NewBotPlayerDB(m.db).DB.Save(botPlayer).Error; err != nil {
			return nil, err
		}

		playerDiamond = botPlayer.Diamond
	} else {
		// 处理真实玩家
		player, err := logic.NewPlayerDB(m.db).GetPlayerByAccount(req.Uid)
		if err != nil {
			return nil, err
		}

		if player.Diamond+req.Diamond < 0 {
			return nil, errors.New("diamond is not enough")
		}

		player.Diamond += req.Diamond
		if err := logic.NewPlayerDB(m.db).Update(player); err != nil {
			return nil, err
		}

		ack := player.ToPlayerInfoAck()
		data, err := utils.Marshal(ctx, ack)
		if err != nil {
			return nil, err
		}
		if _, err := m.app.SendPushToUsers("account", data, []string{req.Uid}, "proxy"); err != nil {
			return nil, err
		}

		playerDiamond = player.Diamond
	}

	return &sproto.ChangeDiamondAck{Uid: req.Uid, Diamond: playerDiamond}, nil
}

func (m *Remote) playerInfoReq(ctx context.Context, bot bool, msg proto.Message) (proto.Message, error) {
	req := msg.(*sproto.PlayerInfoReq)

	// 根据bot标识判断是否为bot
	if bot {
		botPlayer, err := logic.NewBotPlayerDB(m.db).GetBotPlayerByID(req.Uid)
		if err != nil {
			return nil, err
		}
		return botPlayer.ToServerInfoAck(), nil
	}

	player, err := logic.NewPlayerDB(m.db).GetPlayerByUid(req.Uid)
	if err != nil {
		return nil, err
	}

	return player.ToServerInfoAck(), nil
}

func (m *Remote) getBotPlayerReq(ctx context.Context, bot bool, msg proto.Message) (proto.Message, error) {
	// 默认租约时间为1天
	leaseDuration := 24 * time.Hour

	botPlayer, err := logic.NewBotPlayerDB(m.db).GetBotPlayer(leaseDuration)
	if err != nil {
		return nil, err
	}

	return &sproto.GetBotAck{
		Uid:     strconv.FormatUint(uint64(botPlayer.ID), 10),
		Expired: botPlayer.ExpireTime.Unix(),
	}, nil
}
