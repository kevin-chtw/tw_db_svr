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

// Match 独立的匹配服务
type Match struct {
	component.Base
	db       *gorm.DB
	app      pitaya.Pitaya
	handlers map[string]func(context.Context, proto.Message) (proto.Message, error)
}

func NewMatch(db *gorm.DB, app pitaya.Pitaya) *Match {
	return &Match{
		db:       db,
		app:      app,
		handlers: make(map[string]func(context.Context, proto.Message) (proto.Message, error)),
	}
}

// Init 组件初始化
func (m *Match) Init() {
	m.handlers[utils.TypeUrl(&sproto.PayReq{})] = m.payReq
}

func (m *Match) Message(ctx context.Context, req *sproto.Match2AccountReq) (*sproto.Match2AccountAck, error) {
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
		return m.newMatch2AccountAck(rsp)
	}

	return &sproto.Match2AccountAck{}, nil
}

func (m *Match) newMatch2AccountAck(msg proto.Message) (*sproto.Match2AccountAck, error) {
	data, err := anypb.New(msg)
	if err != nil {
		return nil, err
	}
	return &sproto.Match2AccountAck{Ack: data}, nil
}

func (m *Match) payReq(ctx context.Context, msg proto.Message) (proto.Message, error) {
	req := msg.(*sproto.PayReq)
	player, err := logic.NewPlayerDB(m.db).GetPlayerByAccount(req.Uid)
	if err != nil {
		return nil, err
	}

	if player.Diamond < req.Diamond {
		return nil, errors.New("diamond is not enough")
	}

	player.Diamond -= req.Diamond
	if err := logic.NewPlayerDB(m.db).Update(player); err != nil {
		return nil, err
	}

	ack := player.ToPlayerInfoAck()
	if _, err := m.app.SendPushToUsers("account", ack, []string{req.Uid}, "proxy"); err != nil {
		return nil, err
	}

	return &sproto.PayAck{Uid: req.Uid, Diamond: player.Diamond}, nil
}
