package service

import (
	"context"
	"errors"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

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
	bindMutex   sync.Mutex
	bindTimes   map[string]time.Time
}

func NewPlayer(db *gorm.DB, app pitaya.Pitaya, sessionPool session.SessionPool) *Player {
	return &Player{
		db:          db,
		app:         app,
		sessionPool: sessionPool,
		handlers:    make(map[string]func(context.Context, proto.Message) (proto.Message, error)),
		bindTimes:   make(map[string]time.Time),
	}
}

func (l *Player) Init() {
	l.handlers[utils.TypeUrl(&cproto.LoginReq{})] = l.handleLogin
	l.handlers[utils.TypeUrl(&cproto.RegisterReq{})] = l.handleRegister
}

func (l *Player) Message(ctx context.Context, data []byte) ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Errorf("panic recovered %s\n %s", r, string(debug.Stack()))
		}
	}()
	req := &cproto.AccountReq{}
	if err := utils.Unmarshal(ctx, data, req); err != nil {
		return nil, err
	}

	logger.Log.Infof("PlayerMsg: %v", req)
	msg, err := req.Req.UnmarshalNew()
	if err != nil {
		return nil, err
	}

	if handler, ok := l.handlers[req.Req.TypeUrl]; ok {
		if rsp, err := handler(ctx, msg); err != nil {
			return nil, err
		} else {
			return l.newAccountAck(ctx, rsp)
		}
	}
	return nil, errors.ErrUnsupported
}

func (l *Player) newAccountAck(ctx context.Context, msg proto.Message) ([]byte, error) {
	data, err := anypb.New(msg)
	if err != nil {
		return nil, err
	}
	out := &cproto.AccountAck{Ack: data}
	return utils.Marshal(ctx, out)
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

	// 添加会话绑定频率限制
	l.bindMutex.Lock()
	now := time.Now()
	for uid, bindTime := range l.bindTimes {
		if now.Sub(bindTime) > 30*time.Second {
			delete(l.bindTimes, uid)
		}
	}

	lastBindTime, exists := l.bindTimes[info.Uid]
	if exists && now.Sub(lastBindTime) < 5*time.Second {
		l.bindMutex.Unlock()
		return nil, errors.New("session bind too frequent, please wait 5 seconds")
	}
	l.bindTimes[info.Uid] = now
	l.bindMutex.Unlock()

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
	go l.registerAward(ctx, player)
	return info, nil
}

func (l *Player) registerAward(ctx context.Context, player *models.Player) {
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Errorf("panic recovered %s\n %s", r, string(debug.Stack()))
		}
	}()
	player.Coin += 1000
	player.Diamond += 20
	if err := logic.NewPlayerDB(l.db).Update(player); err != nil {
		logger.Log.Errorf("update player error: %v", err)
		return
	}
	ack := &cproto.RegisterAwardAck{
		Uid:     strconv.FormatUint(uint64(player.ID), 10),
		Diamond: player.Diamond,
		Coin:    player.Coin,
	}

	if data, err := utils.Marshal(ctx, ack); err != nil {
		logger.Log.Errorf("marshal error: %v", err)

	} else if _, err := l.app.SendPushToUsers("account", data, []string{ack.Uid}, "proxy"); err != nil {
		logger.Log.Errorf("send push error: %v", err)
	}
}
