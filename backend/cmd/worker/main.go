package main

import (
	"alert-center/internal/repository"
	"alert-center/internal/services"
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/viper"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initConfig()

	db, err := repository.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := runMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	checkInterval := viper.GetDuration("worker.check_interval")
	if checkInterval == 0 {
		checkInterval = 1 * time.Minute
	}

	log.Printf("Starting alert worker with check interval: %v", checkInterval)

	ruleRepo := repository.NewAlertRuleRepository(db)
	historyRepo := repository.NewAlertHistoryRepository(db)
	evaluator := services.NewAlertEvaluator(checkInterval)
	sender := services.NewNotificationSender(db.Pool)
	templateSvc := services.NewAlertTemplateService(db.Pool)
	silenceSvc := services.NewAlertSilenceService(db.Pool)
	slaSvc := services.NewSLAService(db.Pool)
	slaBreachSvc := services.NewSLABreachService(db.Pool, sender)
	worker := services.NewAlertNotificationWorker(db.Pool, ruleRepo, historyRepo, evaluator, sender, templateSvc, silenceSvc, slaSvc, slaBreachSvc, checkInterval)

	if err := worker.Start(ctx); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}

	log.Println("Alert worker started successfully")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down worker...")
	cancel()
	log.Println("Worker stopped")
}

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/alert-center")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.ReadInConfig()
}

func runMigrations(db *repository.Database) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			username VARCHAR(64) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			email VARCHAR(128) UNIQUE,
			phone VARCHAR(32),
			role VARCHAR(32) DEFAULT 'user',
			status INT DEFAULT 1,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			last_login_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS business_groups (
			id UUID PRIMARY KEY,
			name VARCHAR(128) NOT NULL,
			description VARCHAR(512),
			parent_id UUID,
			manager_id UUID,
			status INT DEFAULT 1,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS alert_channels (
			id UUID PRIMARY KEY,
			name VARCHAR(128) NOT NULL,
			type VARCHAR(32) NOT NULL,
			description VARCHAR(512),
			config JSONB,
			group_id UUID,
			status INT DEFAULT 1,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS alert_templates (
			id UUID PRIMARY KEY,
			name VARCHAR(128) NOT NULL,
			description VARCHAR(512),
			content TEXT NOT NULL,
			variables JSONB,
			type VARCHAR(32) DEFAULT 'markdown',
			group_id UUID,
			status INT DEFAULT 1,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS alert_rules (
			id UUID PRIMARY KEY,
			name VARCHAR(128) NOT NULL,
			description VARCHAR(512),
			expression TEXT NOT NULL,
			evaluation_interval_seconds INT DEFAULT 60,
			for_duration INT DEFAULT 60,
			severity VARCHAR(32) NOT NULL,
			labels JSONB,
			annotations JSONB,
			template_id UUID,
			group_id UUID NOT NULL,
			data_source_type VARCHAR(32) DEFAULT 'prometheus',
			data_source_url VARCHAR(512),
			status INT DEFAULT 1,
			effective_start_time VARCHAR(5) DEFAULT '00:00',
			effective_end_time VARCHAR(5) DEFAULT '23:59',
			exclusion_windows JSONB DEFAULT '[]',
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS effective_start_time VARCHAR(5) DEFAULT '00:00'`,
		`ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS effective_end_time VARCHAR(5) DEFAULT '23:59'`,
		`ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS exclusion_windows JSONB DEFAULT '[]'`,
		`ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS evaluation_interval_seconds INT DEFAULT 60`,
		`ALTER TABLE alert_history ADD COLUMN IF NOT EXISTS alert_no VARCHAR(32) UNIQUE`,
		`CREATE TABLE IF NOT EXISTS alert_channel_bindings (
			id UUID PRIMARY KEY,
			rule_id UUID NOT NULL,
			channel_id UUID NOT NULL,
			status INT DEFAULT 1,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			UNIQUE(rule_id, channel_id)
		)`,
		`CREATE TABLE IF NOT EXISTS alert_history (
			id UUID PRIMARY KEY,
			alert_no VARCHAR(32) UNIQUE,
			rule_id UUID NOT NULL,
			fingerprint VARCHAR(256),
			severity VARCHAR(32),
			status VARCHAR(32),
			started_at TIMESTAMP NOT NULL,
			ended_at TIMESTAMP,
			labels JSONB,
			annotations JSONB,
			payload TEXT,
			created_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS operation_logs (
			id UUID PRIMARY KEY,
			user_id UUID,
			action VARCHAR(64),
			resource VARCHAR(128),
			resource_id VARCHAR(128),
			detail TEXT,
			ip VARCHAR(64),
			created_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS data_sources (
			id UUID PRIMARY KEY,
			name VARCHAR(128) NOT NULL,
			type VARCHAR(32) NOT NULL,
			description VARCHAR(512),
			endpoint VARCHAR(512) NOT NULL,
			config JSONB,
			status INT DEFAULT 1,
			health_status VARCHAR(32) DEFAULT 'unknown',
			last_check_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS alert_silences (
			id UUID PRIMARY KEY,
			name VARCHAR(128) NOT NULL,
			description VARCHAR(512),
			matchers JSONB,
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP NOT NULL,
			created_by UUID,
			status INT DEFAULT 1,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS tickets (
			id UUID PRIMARY KEY,
			title VARCHAR(256) NOT NULL,
			description TEXT,
			alert_id UUID,
			rule_id UUID,
			priority VARCHAR(32) NOT NULL DEFAULT 'medium',
			status VARCHAR(32) NOT NULL DEFAULT 'open',
			assignee_id UUID,
			assignee_name VARCHAR(64),
			creator_id UUID NOT NULL,
			creator_name VARCHAR(64) NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			resolved_at TIMESTAMP,
			closed_at TIMESTAMP
		)`,
	}

	ctx := context.Background()
	for _, migration := range migrations {
		if _, err := db.Pool.Exec(ctx, migration); err != nil {
			return err
		}
	}

	return nil
}
