package main

import (
	"alert-center/internal/handlers"
	"alert-center/internal/middleware"
	"alert-center/internal/repository"
	"alert-center/internal/services"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/crypto/bcrypt"
)

// @title Alert Center API
// @version 1.0
// @description Alert Center - Enterprise Alert Management Platform
// @host localhost:8080
// @BasePath /api/v1
// @schemes http https

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
	seedDefaultUser(db)
	seedDefaultBusinessGroups(db)
	seedDefaultAlertTemplates(db)

	userRepo := repository.NewUserRepository(db)
	businessGroupRepo := repository.NewBusinessGroupRepository(db)
	alertRuleRepo := repository.NewAlertRuleRepository(db)
	alertChannelRepo := repository.NewAlertChannelRepository(db)
	alertHistoryRepo := repository.NewAlertHistoryRepository(db)

	userService := services.NewUserService(userRepo)
	alertRuleService := services.NewAlertRuleService(alertRuleRepo, alertChannelRepo, alertHistoryRepo)
	alertChannelService := services.NewAlertChannelService(alertChannelRepo)
	templateService := services.NewAlertTemplateService(db.Pool)
	bindingService := services.NewAlertChannelBindingService(db.Pool)
	userMgmtService := services.NewUserManagementService(db.Pool)
	auditLogService := services.NewAuditLogService(db.Pool)
	dataSourceService := services.NewDataSourceService(db.Pool)
	statisticsService := services.NewAlertStatisticsService(db.Pool)
	silenceService := services.NewAlertSilenceService(db.Pool)
	slaConfigRepo := repository.NewSLAConfigRepository(db)
	slaRepo := repository.NewAlertSLARepository(db)
	oncallScheduleRepo := repository.NewOnCallScheduleRepository(db)
	oncallMemberRepo := repository.NewOnCallMemberRepository(db)
	oncallAssignmentRepo := repository.NewOnCallAssignmentRepository(db)
	correlationService := services.NewAlertCorrelationService(db.Pool)
	escalationService := services.NewAlertEscalationMgmtService(db.Pool)
	schedulingService := services.NewSchedulingService(db.Pool)
	sender := services.NewNotificationSender(db.Pool)
	wsHandler := handlers.NewWebSocketHandler()
	slaBreachService := services.NewSLABreachService(db.Pool, sender, wsHandler)

	userHandler := handlers.NewUserHandler(userService)
	alertRuleHandler := handlers.NewAlertRuleHandler(alertRuleService, bindingService)
	alertChannelHandler := handlers.NewAlertChannelHandler(alertChannelService)
	businessGroupHandler := handlers.NewBusinessGroupHandler(businessGroupRepo)
	alertHistoryHandler := handlers.NewAlertHistoryHandler(alertHistoryRepo)
	templateHandler := handlers.NewAlertTemplateHandler(templateService)
	bindingHandler := handlers.NewAlertChannelBindingHandler(bindingService)
	userMgmtHandler := handlers.NewUserManagementHandler(userMgmtService)
	auditLogHandler := handlers.NewAuditLogHandler(auditLogService)
	dataSourceHandler := handlers.NewDataSourceHandler(dataSourceService)
	statisticsHandler := handlers.NewAlertStatisticsHandler(statisticsService)
	silenceHandler := handlers.NewAlertSilenceHandler(silenceService)
	batchHandler := handlers.NewBatchImportHandler(alertRuleService, silenceService)
	slaHandler := handlers.NewSLAHandler(slaConfigRepo).WithAlertSLARepository(slaRepo)
	oncallHandler := handlers.NewOnCallHandler(oncallScheduleRepo).WithRepositories(oncallMemberRepo, oncallAssignmentRepo)
	correlationHandler := handlers.NewCorrelationHandler(correlationService)
	escalationHandler := handlers.NewEscalationHandler(escalationService)
	schedulingHandler := handlers.NewSchedulingHandler(schedulingService)
	slaBreachHandler := handlers.NewSLABreachHandler(slaBreachService)
	escalationHistoryHandler := handlers.NewEscalationHistoryHandler(db)
	ticketHandler := handlers.NewTicketHandler(db, wsHandler)

	router := initRouter(
		wsHandler,
		userHandler,
		alertRuleHandler,
		alertChannelHandler,
		businessGroupHandler,
		alertHistoryHandler,
		templateHandler,
		bindingHandler,
		userMgmtHandler,
		auditLogHandler,
		dataSourceHandler,
		statisticsHandler,
		silenceHandler,
		batchHandler,
		slaHandler,
		oncallHandler,
		correlationHandler,
		escalationHandler,
		schedulingHandler,
		slaBreachHandler,
		escalationHistoryHandler,
		ticketHandler,
	)

	addr := fmt.Sprintf("%s:%d", viper.GetString("app.host"), viper.GetInt("app.port"))

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("Starting API server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	go startWorker(ctx, db, wsHandler)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func startWorker(ctx context.Context, db *repository.Database, broadcaster services.Broadcaster) {
	ruleRepo := repository.NewAlertRuleRepository(db)
	historyRepo := repository.NewAlertHistoryRepository(db)
	evaluator := services.NewAlertEvaluator(1 * time.Minute)
	sender := services.NewNotificationSender(db.Pool)
	templateSvc := services.NewAlertTemplateService(db.Pool)
	silenceSvc := services.NewAlertSilenceService(db.Pool)
	slaSvc := services.NewSLAService(db.Pool)
	slaBreachService := services.NewSLABreachService(db.Pool, sender, broadcaster)

	if err := slaSvc.SeedDefaultSLAConfigs(ctx); err != nil {
		log.Printf("Failed to seed SLA configs: %v", err)
	}

	worker := services.NewAlertNotificationWorker(db.Pool, ruleRepo, historyRepo, evaluator, sender, templateSvc, silenceSvc, slaSvc, slaBreachService, broadcaster, 1*time.Minute)

	if err := worker.Start(ctx); err != nil {
		log.Printf("Failed to start worker: %v", err)
	}
}

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/alert-center")
	viper.AutomaticEnv()
	// So env vars like DATABASE_HOST (not DATABASE.HOST) override config keys like database.host
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
		`CREATE TABLE IF NOT EXISTS alert_escalations (
			id UUID PRIMARY KEY,
			name VARCHAR(128) NOT NULL,
			description VARCHAR(512),
			rule_id UUID NOT NULL,
			severity VARCHAR(32) NOT NULL,
			escalate_to VARCHAR(32) NOT NULL,
			wait_minutes INT DEFAULT 5,
			channel_id UUID,
			repeat_count INT DEFAULT 0,
			repeat_minutes INT DEFAULT 30,
			status INT DEFAULT 1,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS alert_escalation_logs (
			id UUID PRIMARY KEY,
			escalation_id UUID NOT NULL,
			alert_id UUID NOT NULL,
			from_severity VARCHAR(32),
			to_severity VARCHAR(32),
			channel_id UUID,
			notified_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS notification_templates (
			id UUID PRIMARY KEY,
			name VARCHAR(128) NOT NULL,
			description VARCHAR(512),
			type VARCHAR(32) DEFAULT 'markdown',
			channel_type VARCHAR(32) NOT NULL,
			subject VARCHAR(256),
			content TEXT,
			variables JSONB,
			status INT DEFAULT 1,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sla_configs (
			id UUID PRIMARY KEY,
			name VARCHAR(128) NOT NULL,
			severity VARCHAR(32) NOT NULL,
			response_time_mins INT NOT NULL,
			resolution_time_mins INT NOT NULL,
			priority INT DEFAULT 0,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS alert_slas (
			id UUID PRIMARY KEY,
			alert_id UUID NOT NULL,
			rule_id UUID NOT NULL,
			severity VARCHAR(32) NOT NULL,
			sla_config_id UUID,
			response_deadline TIMESTAMP,
			resolution_deadline TIMESTAMP,
			first_acked_at TIMESTAMP,
			resolved_at TIMESTAMP,
			status VARCHAR(32) DEFAULT 'pending',
			response_breached BOOLEAN DEFAULT FALSE,
			resolution_breached BOOLEAN DEFAULT FALSE,
			response_time_secs FLOAT,
			resolution_time_secs FLOAT,
			created_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS oncall_schedules (
			id UUID PRIMARY KEY,
			name VARCHAR(128) NOT NULL,
			description VARCHAR(512),
			timezone VARCHAR(64) DEFAULT 'UTC',
			rotation_type VARCHAR(32) DEFAULT 'weekly',
			rotation_start TIMESTAMP,
			enabled BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS oncall_members (
			id UUID PRIMARY KEY,
			schedule_id UUID NOT NULL,
			user_id UUID NOT NULL,
			username VARCHAR(64) NOT NULL,
			email VARCHAR(128),
			phone VARCHAR(32),
			priority INT DEFAULT 0,
			start_time TIMESTAMP,
			end_time TIMESTAMP,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS oncall_assignments (
			id UUID PRIMARY KEY,
			schedule_id UUID NOT NULL,
			user_id UUID NOT NULL,
			username VARCHAR(64) NOT NULL,
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS oncall_escalations (
			id UUID PRIMARY KEY,
			schedule_id UUID NOT NULL,
			from_user_id UUID NOT NULL,
			to_user_id UUID NOT NULL,
			escalated_at TIMESTAMP NOT NULL,
			reason TEXT,
			created_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sla_breaches (
			id UUID PRIMARY KEY,
			alert_id UUID NOT NULL,
			rule_id UUID NOT NULL,
			severity VARCHAR(32) NOT NULL,
			breach_type VARCHAR(32) NOT NULL,
			breach_time TIMESTAMP NOT NULL,
			response_time FLOAT,
			assigned_to UUID,
			assigned_name VARCHAR(64),
			notified BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP NOT NULL
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
		`CREATE TABLE IF NOT EXISTS user_escalations (
			id UUID PRIMARY KEY,
			alert_id UUID NOT NULL,
			from_user_id UUID NOT NULL,
			from_username VARCHAR(64) NOT NULL,
			to_user_id UUID NOT NULL,
			to_username VARCHAR(64) NOT NULL,
			reason TEXT,
			status VARCHAR(32) NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP NOT NULL,
			resolved_at TIMESTAMP
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

// seedDefaultUser creates default admin if no user exists.
func seedDefaultUser(db *repository.Database) {
	ctx := context.Background()
	var n int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil || n > 0 {
		return
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash default password: %v", err)
		return
	}
	id := uuid.New()
	now := time.Now()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO users (id, username, password, email, phone, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, id, "admin", string(hashed), "", "", "admin", 1, now, now)
	if err != nil {
		log.Printf("Failed to seed default user: %v", err)
		return
	}
	log.Printf("Default user created: admin / admin123 (change password after first login)")
}

// seedDefaultBusinessGroups inserts default business groups if the table is empty.
func seedDefaultBusinessGroups(db *repository.Database) {
	ctx := context.Background()
	var n int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM business_groups`).Scan(&n); err != nil || n > 0 {
		return
	}
	defaults := []struct {
		name        string
		description string
	}{
		{"基础设施组", "负责基础设施运维"},
		{"应用服务组", "负责应用服务运维"},
		{"数据库组", "负责数据库运维"},
	}
	for _, d := range defaults {
		id := uuid.New()
		now := time.Now()
		_, err := db.Pool.Exec(ctx, `
			INSERT INTO business_groups (id, name, description, parent_id, manager_id, status, created_at, updated_at)
			VALUES ($1, $2, $3, NULL, NULL, 1, $4, $5)
		`, id, d.name, d.description, now, now)
		if err != nil {
			log.Printf("Failed to seed business group %q: %v", d.name, err)
			return
		}
	}
	log.Printf("Default business groups seeded: %d", len(defaults))
}

// seedDefaultAlertTemplates inserts default K8s Prometheus alert template if none exist, or updates existing one to dynamic format.
func seedDefaultAlertTemplates(db *repository.Database) {
	ctx := context.Background()
	var n int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM alert_templates WHERE status = 1`).Scan(&n); err != nil {
		return
	}
	content := "## 告警\n\n" +
		"**规则名称**: {{ruleName}}\n" +
		"**严重级别**: {{severity}}\n" +
		"**状态**: {{status}}\n" +
		"**触发时间**: {{startTime}}\n" +
		"**持续时间**: {{duration}}\n\n" +
		"### 标签 (Labels)\n" +
		"{{labelsFormatted}}\n\n" +
		"### 注释 (Annotations)\n" +
		"{{annotationsFormatted}}\n\n" +
		"### 处理建议\n" +
		"根据上述标签定位资源（如 namespace/pod/node/job 等），检查事件与日志：`kubectl describe` / `kubectl logs`。"
	variables := `{"ruleName":"规则名称","severity":"严重级别","status":"状态","startTime":"触发时间","duration":"持续时间","labelsFormatted":"标签键值（自动适配）","annotationsFormatted":"注释键值（自动适配）","labels":"原始 labels JSON","annotations":"原始 annotations JSON"}`
	desc := "动态适配任意 Prometheus 告警：标签与注释按实际键值自动展示，无需固定格式"
	if n > 0 {
		_, _ = db.Pool.Exec(ctx, `
			UPDATE alert_templates SET content = $1, variables = $2, description = $3, updated_at = $4
			WHERE name = 'K8s Prometheus 默认告警模板' AND status = 1
		`, content, variables, desc, time.Now())
		return
	}
	id := uuid.New()
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO alert_templates (id, name, description, content, variables, type, group_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NULL, 1, $7, $8)
	`, id, "K8s Prometheus 默认告警模板", desc, content, variables, "markdown", now, now)
	if err != nil {
		log.Printf("Failed to seed default alert template: %v", err)
		return
	}
	log.Printf("Default alert template seeded: K8s Prometheus 默认告警模板")
}

func initRouter(
	wsHandler *handlers.WebSocketHandler,
	userHandler *handlers.UserHandler,
	alertRuleHandler *handlers.AlertRuleHandler,
	alertChannelHandler *handlers.AlertChannelHandler,
	businessGroupHandler *handlers.BusinessGroupHandler,
	alertHistoryHandler *handlers.AlertHistoryHandler,
	templateHandler *handlers.AlertTemplateHandler,
	bindingHandler *handlers.AlertChannelBindingHandler,
	userMgmtHandler *handlers.UserManagementHandler,
	auditLogHandler *handlers.AuditLogHandler,
	dataSourceHandler *handlers.DataSourceHandler,
	statisticsHandler *handlers.AlertStatisticsHandler,
	silenceHandler *handlers.AlertSilenceHandler,
	batchHandler *handlers.BatchImportHandler,
	slaHandler *handlers.SLAHandler,
	oncallHandler *handlers.OnCallHandler,
	correlationHandler *handlers.CorrelationHandler,
	escalationHandler *handlers.EscalationHandler,
	schedulingHandler *handlers.SchedulingHandler,
	slaBreachHandler *handlers.SLABreachHandler,
	escalationHistoryHandler *handlers.EscalationHistoryHandler,
	ticketHandler *handlers.TicketHandler) *gin.Engine {

	router := gin.New()
	router.Use(middleware.RecoveryMiddleware())
	router.Use(middleware.LoggerMiddleware())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.RequestIDMiddleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	go wsHandler.HandleBroadcast()
	router.GET("/api/v1/ws", wsHandler.HandleConnection)

	public := router.Group("/api/v1")
	{
		public.POST("/auth/login", userHandler.Login)
	}

	api := router.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(viper.GetString("jwt.secret")))
	{
		api.GET("/profile", userHandler.GetProfile)

		api.GET("/business-groups", businessGroupHandler.List)

		api.POST("/users", userMgmtHandler.Create)
		api.GET("/users", userMgmtHandler.List)
		api.GET("/users/:id", userMgmtHandler.GetByID)
		api.PUT("/users/:id", userMgmtHandler.Update)
		api.DELETE("/users/:id", userMgmtHandler.Delete)
		api.POST("/users/:id/password", userMgmtHandler.ChangePassword)

		api.POST("/alert-rules", alertRuleHandler.Create)
		api.POST("/alert-rules/test-expression", alertRuleHandler.TestExpression)
		api.GET("/alert-rules", alertRuleHandler.List)
		api.GET("/alert-rules/:id", alertRuleHandler.GetByID)
		api.PUT("/alert-rules/:id", alertRuleHandler.Update)
		api.DELETE("/alert-rules/:id", alertRuleHandler.Delete)
		api.GET("/alert-rules/export", alertRuleHandler.Export)
		api.GET("/alert-rules/:id/bindings", alertRuleHandler.GetBindings)
		api.POST("/alert-rules/:id/bindings", bindingHandler.BindChannels)

		api.POST("/channels", alertChannelHandler.Create)
		api.GET("/channels", alertChannelHandler.List)
		api.GET("/channels/:id", alertChannelHandler.GetByID)
		api.PUT("/channels/:id", alertChannelHandler.Update)
		api.DELETE("/channels/:id", alertChannelHandler.Delete)
		api.POST("/channels/:id/test", alertChannelHandler.Test)
		api.POST("/channels/test-config", alertChannelHandler.TestWithConfig)

		api.GET("/templates", templateHandler.List)
		api.POST("/templates", templateHandler.Create)
		api.GET("/templates/:id", templateHandler.GetByID)
		api.PUT("/templates/:id", templateHandler.Update)
		api.DELETE("/templates/:id", templateHandler.Delete)

		api.GET("/alert-history", alertHistoryHandler.List)

		api.GET("/audit-logs", auditLogHandler.List)
		api.GET("/audit-logs/export", auditLogHandler.Export)

		api.GET("/data-sources", dataSourceHandler.List)
		api.POST("/data-sources", dataSourceHandler.Create)
		api.GET("/data-sources/:id", dataSourceHandler.GetByID)
		api.PUT("/data-sources/:id", dataSourceHandler.Update)
		api.DELETE("/data-sources/:id", dataSourceHandler.Delete)
		api.POST("/data-sources/:id/health-check", dataSourceHandler.HealthCheck)

		api.GET("/statistics", statisticsHandler.Statistics)
		api.GET("/dashboard", statisticsHandler.Dashboard)

		api.GET("/silences", silenceHandler.List)
		api.POST("/silences", silenceHandler.Create)
		api.PUT("/silences/:id", silenceHandler.Update)
		api.DELETE("/silences/:id", silenceHandler.Delete)
		api.POST("/silences/check", silenceHandler.Check)

		api.POST("/batch/import/rules", batchHandler.ImportRules)
		api.GET("/batch/export/rules", batchHandler.ExportRules)
		api.GET("/batch/export/channels", batchHandler.ExportChannels)
		api.POST("/batch/import/silences", batchHandler.ImportSilences)
		api.GET("/batch/export/silences", batchHandler.ExportSilences)

		api.GET("/sla/configs", slaHandler.ListSLAConfigs)
		api.POST("/sla/configs", slaHandler.CreateSLAConfig)
		api.GET("/sla/configs/:id", slaHandler.GetSLAConfig)
		api.PUT("/sla/configs/:id", slaHandler.UpdateSLAConfig)
		api.DELETE("/sla/configs/:id", slaHandler.DeleteSLAConfig)
		api.GET("/sla/configs/seed", slaHandler.SeedDefaultSLAConfigs)
		api.GET("/sla/alerts/:alert_id", slaHandler.GetAlertSLA)
		api.GET("/sla/report", slaHandler.GetSLAReport)

		api.GET("/oncall/schedules", oncallHandler.GetSchedules)
		api.POST("/oncall/schedules", oncallHandler.CreateSchedule)
		api.GET("/oncall/schedules/:id", oncallHandler.GetSchedule)
		api.PUT("/oncall/schedules/:id", oncallHandler.UpdateSchedule)
		api.DELETE("/oncall/schedules/:id", oncallHandler.DeleteSchedule)
		api.POST("/oncall/schedules/:id/members", oncallHandler.AddMember)
		api.GET("/oncall/schedules/:id/members", oncallHandler.GetMembers)
		api.DELETE("/oncall/schedules/:id/members/:member_id", oncallHandler.DeleteMember)
		api.GET("/oncall/schedules/:id/assignments", oncallHandler.GetScheduleAssignments)
		api.POST("/oncall/schedules/:id/generate-rotations", oncallHandler.GenerateRotations)
		api.POST("/oncall/schedules/:id/escalate", oncallHandler.Escalate)
		api.GET("/oncall/current", oncallHandler.GetCurrentOnCall)
		api.GET("/oncall/who", oncallHandler.WhoIsOnCall)
		api.GET("/oncall/report", oncallHandler.GetOnCallReport)
		api.GET("/oncall/seed", oncallHandler.SeedDefaultSchedules)

		api.GET("/correlation/analyze/:id", correlationHandler.AnalyzeCorrelations)
		api.GET("/correlation/patterns", correlationHandler.FindPatterns)
		api.GET("/correlation/groups", correlationHandler.GroupSimilarAlerts)
		api.GET("/correlation/timeline/:fingerprint", correlationHandler.GenerateTimeline)
		api.GET("/correlation/flapping", correlationHandler.DetectFlapping)
		api.GET("/correlation/predict/:rule_id", correlationHandler.PredictAlerts)

		api.GET("/escalations", escalationHistoryHandler.GetHistory)
		api.GET("/escalations/stats", escalationHistoryHandler.GetStats)
		api.GET("/escalations/alert/:alert_id", escalationHandler.GetAlertEscalations)

		api.POST("/escalations", escalationHandler.CreateEscalation)
		api.GET("/escalations/pending", escalationHandler.GetMyPendingEscalations)
		api.POST("/escalations/:id/accept", escalationHandler.AcceptEscalation)
		api.POST("/escalations/:id/reject", escalationHandler.RejectEscalation)
		api.POST("/escalations/:id/resolve", escalationHandler.ResolveEscalation)

		api.POST("/oncall/schedules/:id/generate", schedulingHandler.GenerateSchedule)
		api.GET("/oncall/schedules/:id/coverage", schedulingHandler.GetScheduleCoverage)
		api.GET("/oncall/schedules/:id/suggest", schedulingHandler.SuggestRotation)
		api.GET("/oncall/schedules/:id/validate", schedulingHandler.ValidateSchedule)

		api.GET("/sla/breaches", slaBreachHandler.GetBreaches)
		api.GET("/sla/breaches/stats", slaBreachHandler.GetBreachStats)
		api.POST("/sla/breaches/check", slaBreachHandler.TriggerCheck)
		api.POST("/sla/breaches/notify", slaBreachHandler.TriggerNotifications)

		api.GET("/tickets", ticketHandler.List)
		api.POST("/tickets", ticketHandler.Create)
		api.GET("/tickets/:id", ticketHandler.GetByID)
		api.PUT("/tickets/:id", ticketHandler.Update)
		api.POST("/tickets/:id/resolve", ticketHandler.Resolve)
		api.POST("/tickets/:id/close", ticketHandler.Close)
		api.DELETE("/tickets/:id", ticketHandler.Delete)
		api.GET("/tickets/stats", ticketHandler.Stats)
	}

	return router
}
