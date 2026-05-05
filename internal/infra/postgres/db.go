package postgres

import (
	"log"

	"github.com/harshc9/llm-service/internal/domain/client"
	"github.com/harshc9/llm-service/internal/domain/keypool"
	"github.com/harshc9/llm-service/internal/domain/modelcatalog"
	"github.com/harshc9/llm-service/internal/domain/routing"
	"github.com/harshc9/llm-service/internal/domain/usage"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	log.Println("Migrating database models...")
	err = db.AutoMigrate(
		&client.Client{},
		&keypool.ApiProject{},
		&keypool.ApiKey{},
		&modelcatalog.ModelCatalog{},
		&routing.RoutingPolicy{},
		&usage.UsageEvent{},
		&usage.DailyUsageAggregate{},
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}
