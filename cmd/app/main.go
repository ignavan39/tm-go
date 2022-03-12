package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi"
	chim "github.com/go-chi/chi/middleware"
	"github.com/ignavan39/ucrm-go/app/api"
	"github.com/ignavan39/ucrm-go/app/api/cards"
	"github.com/ignavan39/ucrm-go/app/api/dashboards"
	"github.com/ignavan39/ucrm-go/app/api/pipelines"
	"github.com/ignavan39/ucrm-go/app/api/users"
	"github.com/ignavan39/ucrm-go/app/auth"
	"github.com/ignavan39/ucrm-go/app/config"
	"github.com/ignavan39/ucrm-go/app/repository/database"
	"github.com/ignavan39/ucrm-go/pkg/pg"
	blogger "github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()
	config, err := config.GetConfig()
	blogger.SetOutput(os.Stdout)
	blogger.SetFormatter(&blogger.TextFormatter{})

	if err != nil {
		blogger.Fatal(err.Error())
	}

	rwConn, err := pg.NewReadAndWriteConnection(ctx, config.Database, config.Database)

	if err != nil {
		blogger.Fatal(err.Error())
	}

	web := api.NewAPIServer(":8080").
		WithCors(config.Cors)
	dbService := database.NewDbService(rwConn)
	authorizer := auth.NewAuthorizer(config.JWT.HashSalt, []byte(config.JWT.SigningKey), config.JWT.ExpireDuration)
	userController := users.NewController(authorizer, dbService)
	dashboardController := dashboards.NewController(dbService, dbService)
	pipelineController := pipelines.NewController(dbService)
	cardController := cards.NewController(dbService, dbService)

	web.Router().Route("/api/v1", func(v1 chi.Router) {
		v1.Use(
			chim.Logger,
			chim.Recoverer,
			chim.RequestID,
		)
		users.RegisterRouter(v1, userController)
		dashboards.RegisterRouter(v1, dashboardController, dbService, config.JWT)
		pipelines.RegisterRouter(v1, pipelineController, dbService, dbService, config.JWT)
		cards.RegisterRouter(v1, cardController, dbService, config.JWT)
	})
	web.Start()

	appCloser := make(chan os.Signal)
	signal.Notify(appCloser, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-appCloser
		blogger.Info("[os.SIGNAL] close request")
		go web.Stop()
		blogger.Info("[os.SIGNAL] done")
	}()

}
