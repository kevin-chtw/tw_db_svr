package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/kevin-chtw/tw_db_svr/models"
	"github.com/kevin-chtw/tw_db_svr/service"
	"github.com/sirupsen/logrus"
	pitaya "github.com/topfreegames/pitaya/v3/pkg"
	"github.com/topfreegames/pitaya/v3/pkg/component"
	"github.com/topfreegames/pitaya/v3/pkg/config"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var app pitaya.Pitaya

func main() {
	// 加载数据库配置
	configData, err := os.ReadFile("etc/db.yaml")
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

	builder := pitaya.NewBuilder(false, "db", pitaya.Cluster, map[string]string{}, *config.NewDefaultPitayaConfig())
	app = builder.Build()

	// 注册服务
	initServices(db)

	logrus.Infof("Pitaya database server started")
	app.Start()
}

func initServices(db *gorm.DB) {
	account := service.NewAccountService(db, app)
	app.Register(account, component.WithName("account"), component.WithNameFunc(strings.ToLower))
	app.RegisterRemote(account, component.WithName("account"), component.WithNameFunc(strings.ToLower))
}
