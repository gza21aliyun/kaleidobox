package service

import (
	"context"
	"database/sql"
	"lunabox/internal/appconf"
)

type TaskService struct {
	ctx    context.Context
	db     *sql.DB
	config *appconf.AppConfig
}
