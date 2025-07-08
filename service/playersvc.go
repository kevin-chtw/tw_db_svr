package service

import (
	"context"
	"strconv"

	"github.com/kevin-chtw/tw_db_svr/logic"
	"github.com/kevin-chtw/tw_proto/cproto"
	"github.com/kevin-chtw/tw_proto/sproto"
	"github.com/sirupsen/logrus"
	pitaya "github.com/topfreegames/pitaya/v3/pkg"
	"github.com/topfreegames/pitaya/v3/pkg/component"
	"gorm.io/gorm"
)

type PlayerSvc struct {
	component.Base
	playerDB *logic.PlayerDB
	app      pitaya.Pitaya
}

func NewPlayerSvc(db *gorm.DB, app pitaya.Pitaya) *PlayerSvc {
	return &PlayerSvc{
		playerDB: logic.NewPlayerDB(db),
		app:      app,
	}
}

func (s *PlayerSvc) Get(ctx context.Context, req *sproto.GetPlayerReq) (*sproto.GetPlayerAck, error) {
	player, err := s.playerDB.VerifyAccount(req.Account, req.Password)
	if err != nil {
		return nil, err
	}

	return &sproto.GetPlayerAck{
		Userid:  strconv.FormatUint(uint64(player.ID), 10),
		Account: player.Account,
	}, nil
}

func (s *PlayerSvc) Create(ctx context.Context, req *cproto.LobbyReq) (*cproto.LobbyAck, error) {
	rsp := &cproto.LobbyAck{
		RegisterAck: &cproto.RegisterAck{
			Serverid: "",
			Userid:   "",
		},
	}
	// 检查账号是否已存在
	if exists, _ := s.playerDB.AccountExists(req.RegisterReq.Account); exists {
		logrus.Errorf("账号已存在: %s", req.RegisterReq.Account)
		return rsp, nil
	}

	// 创建新账号
	player, err := s.playerDB.CreateAccount(req.RegisterReq.Account, req.RegisterReq.Password)
	if err != nil {
		logrus.Errorf("创建账号失败: %v", err.Error())
		return rsp, err
	}

	rsp.RegisterAck.Userid = strconv.FormatUint(uint64(player.ID), 10)
	return rsp, nil
}
