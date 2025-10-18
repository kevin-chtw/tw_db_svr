package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/kevin-chtw/tw_common/utils"
	"github.com/kevin-chtw/tw_db_svr/models"
	"github.com/kevin-chtw/tw_db_svr/service"
	"github.com/sirupsen/logrus"
	pitaya "github.com/topfreegames/pitaya/v3/pkg"
	"github.com/topfreegames/pitaya/v3/pkg/component"
	"github.com/topfreegames/pitaya/v3/pkg/config"
	"github.com/topfreegames/pitaya/v3/pkg/logger"
	"github.com/topfreegames/pitaya/v3/pkg/serialize"
	"github.com/topfreegames/pitaya/v3/pkg/session"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var app pitaya.Pitaya

func main() {
	pitaya.SetLogger(utils.Logger(logrus.DebugLevel))
	// 加载数据库配置
	configData, err := os.ReadFile("etc/account/config.yaml")
	if err != nil {
		panic(fmt.Sprintf("failed to read db config: %v", err))
	}

	var dbConfig struct {
		MySQL string `yaml:"MySQL"`
	}
	if err := yaml.Unmarshal(configData, &dbConfig); err != nil {
		panic(fmt.Sprintf("failed to parse db config: %v", err))
	}

	// 初始化MySQL数据库
	db, err := gorm.Open(mysql.Open(dbConfig.MySQL), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %v", err))
	}

	// 自动迁移模型
	if err := db.AutoMigrate(&models.Player{}); err != nil {
		panic("failed to migrate database")
	}

	if err := adjustAutoIncrement(db); err != nil {
		panic(fmt.Sprintf("failed to connect database: %v", err))
	}
	config := config.NewDefaultPitayaConfig()
	config.SerializerType = uint16(serialize.PROTOBUF)
	config.Handler.Messages.Compression = false
	builder := pitaya.NewBuilder(false, "account", pitaya.Cluster, map[string]string{}, *config)
	app = builder.Build()
	defer app.Shutdown()

	// 注册服务
	initServices(db, builder.SessionPool)

	logger.Log.Infof("Pitaya database server started")
	app.Start()
}

func initServices(db *gorm.DB, sessionPool session.SessionPool) {
	player := service.NewPlayer(db, app, sessionPool)
	app.Register(player, component.WithName("player"), component.WithNameFunc(strings.ToLower)) //仅客户端访问

	server := service.NewServer(db, app)
	app.RegisterRemote(server, component.WithName("server"), component.WithNameFunc(strings.ToLower)) //仅服务器访问
}

// 核心函数：读当前值，< want 就调到 want
func adjustAutoIncrement(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var current sql.NullInt64
		result := tx.Raw(`
			SELECT AUTO_INCREMENT
			FROM information_schema.TABLES
			WHERE TABLE_SCHEMA = DATABASE()
			  AND TABLE_NAME   = 'players'`).Scan(&current)
		if result.Error != nil {
			return result.Error
		}
		if !current.Valid { // 表刚建、还没有任何数据时可能为 NULL
			current.Int64 = 1
		}
		if current.Int64 >= 10000 { // 已经≥目标值，无需调整
			return nil
		}
		// 调到 10000
		return tx.Exec("ALTER TABLE `players` AUTO_INCREMENT = 10000").Error
	})
}
