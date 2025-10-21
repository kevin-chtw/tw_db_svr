package service

import (
	"context"
	"errors"
	"runtime/debug"

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

// Server 独立的匹配服务
type Server struct {
	component.Base
	db       *gorm.DB
	app      pitaya.Pitaya
	handlers map[string]func(context.Context, proto.Message) (proto.Message, error)
}

func NewServer(db *gorm.DB, app pitaya.Pitaya) *Server {
	return &Server{
		db:       db,
		app:      app,
		handlers: make(map[string]func(context.Context, proto.Message) (proto.Message, error)),
	}
}

// Init 组件初始化
func (m *Server) Init() {
	m.handlers[utils.TypeUrl(&sproto.ChangeDiamondReq{})] = m.changeDiamondReq
	m.handlers[utils.TypeUrl(&sproto.PlayerInfoReq{})] = m.playerInfoReq
}

func (m *Server) Message(ctx context.Context, req *sproto.AccountReq) (*sproto.AccountAck, error) {
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
		rsp, err := handler(ctx, msg)
		if err != nil {
			return nil, err
		}
		return m.newAccountAck(rsp)
	}

	return &sproto.AccountAck{}, nil
}

func (m *Server) newAccountAck(msg proto.Message) (*sproto.AccountAck, error) {
	data, err := anypb.New(msg)
	if err != nil {
		return nil, err
	}
	return &sproto.AccountAck{Ack: data}, nil
}

func (m *Server) changeDiamondReq(ctx context.Context, msg proto.Message) (proto.Message, error) {
	req := msg.(*sproto.ChangeDiamondReq)
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

	return &sproto.ChangeDiamondAck{Uid: req.Uid, Diamond: player.Diamond}, nil
}

func (m *Server) playerInfoReq(ctx context.Context, msg proto.Message) (proto.Message, error) {
	req := msg.(*sproto.PlayerInfoReq)
	player, err := logic.NewPlayerDB(m.db).GetPlayerByUid(req.Uid)
	if err != nil {
		return nil, err
	}

	return player.ToServerInfoAck(), nil
}
