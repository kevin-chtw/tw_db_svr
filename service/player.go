package service

import (
	"context"
	"errors"
	"runtime/debug"

	"github.com/kevin-chtw/tw_common/utils"
	"github.com/kevin-chtw/tw_db_svr/logic"
	"github.com/kevin-chtw/tw_db_svr/models"
	"github.com/kevin-chtw/tw_proto/cproto"
	pitaya "github.com/topfreegames/pitaya/v3/pkg"
	"github.com/topfreegames/pitaya/v3/pkg/component"
	"github.com/topfreegames/pitaya/v3/pkg/logger"
	"github.com/topfreegames/pitaya/v3/pkg/session"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"gorm.io/gorm"
)

type Player struct {
	component.Base
	db          *gorm.DB
	app         pitaya.Pitaya
	sessionPool session.SessionPool
	handlers    map[string]func(context.Context, proto.Message) (proto.Message, error)
}

func NewPlayer(db *gorm.DB, app pitaya.Pitaya, sessionPool session.SessionPool) *Player {
	return &Player{
		db:          db,
		app:         app,
		sessionPool: sessionPool,
		handlers:    make(map[string]func(context.Context, proto.Message) (proto.Message, error)),
	}
}

func (l *Player) Init() {
	l.handlers[utils.TypeUrl(&cproto.LoginReq{})] = l.handleLogin
	l.handlers[utils.TypeUrl(&cproto.RegisterReq{})] = l.handleRegister
}

func (l *Player) Message(ctx context.Context, req *cproto.AccountReq) (*cproto.AccountAck, error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Errorf("panic recovered %s\n %s", r, string(debug.Stack()))
		}
	}()
	logger.Log.Infof("PlayerMsg: %v", req)

	msg, err := req.Req.UnmarshalNew()
	if err != nil {
		return nil, err
	}

	if handler, ok := l.handlers[req.Req.TypeUrl]; ok {
		rsp, err := handler(ctx, msg)
		if err != nil {
			return nil, err
		}
		return l.newLobbyAck(rsp)
	}

	return &cproto.AccountAck{}, nil
}

func (l *Player) newLobbyAck(msg proto.Message) (*cproto.AccountAck, error) {
	data, err := anypb.New(msg)
	if err != nil {
		return nil, err
	}
	return &cproto.AccountAck{Ack: data}, nil
}

func (l *Player) handleLogin(ctx context.Context, msg proto.Message) (proto.Message, error) {
	req := msg.(*cproto.LoginReq)
	player, err := logic.NewPlayerDB(l.db).GetPlayerByAccount(req.Account)
	if err != nil {
		return nil, err
	}
	if player.Pwd != logic.HashPassword(req.Password) {
		return nil, err
	}

	info := player.ToPlayerInfoAck()
	if old := l.sessionPool.GetSessionByUID(info.Uid); old != nil {
		old.Kick(context.Background())
	}

	s := l.app.GetSessionFromCtx(ctx)
	if err := s.Bind(ctx, info.Uid); err != nil {
		return nil, err
	}
	return info, nil
}

func (l *Player) handleRegister(ctx context.Context, msg proto.Message) (proto.Message, error) {
	req := msg.(*cproto.RegisterReq)
	_, err := logic.NewPlayerDB(l.db).GetPlayerByAccount(req.Account)
	if err == nil {
		return nil, errors.New("player is exist")
	}
	player := &models.Player{
		Nickname: req.Account,
		Account:  req.Account,
		Pwd:      logic.HashPassword(req.Password),
		Avatar:   req.Avatar,
	}

	player, err = logic.NewPlayerDB(l.db).CreatePlayer(player)
	if err != nil {
		return nil, err
	}
	info := player.ToPlayerInfoAck()
	s := l.app.GetSessionFromCtx(ctx)
	if err := s.Bind(ctx, info.Uid); err != nil {
		return nil, err
	}
	return info, nil
}
