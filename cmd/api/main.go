package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/Hisaalu/rogele/internal/auth"
	"github.com/Hisaalu/rogele/internal/config"
	"github.com/Hisaalu/rogele/internal/crud"
	"github.com/Hisaalu/rogele/internal/db"
	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/handler"
	"github.com/Hisaalu/rogele/internal/logger"
	"github.com/Hisaalu/rogele/internal/mailer"
	"github.com/Hisaalu/rogele/internal/payments"
	"github.com/Hisaalu/rogele/internal/repository"
	"github.com/Hisaalu/rogele/internal/repository/postgres"
	"github.com/Hisaalu/rogele/internal/scheduler"
	"github.com/Hisaalu/rogele/internal/server"
	"github.com/Hisaalu/rogele/internal/service"
	"github.com/Hisaalu/rogele/internal/storage"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(cfg.App.Env)
	log.Info("starting rogele api", "env", cfg.App.Env, "version", version)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	database, err := db.NewPostgres(ctx, cfg.Database, log)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Error("closing db", "error", err)
		}
	}()

	repos := &repository.Repositories{
		User:          postgres.NewUserRepo(database),
		Lesson:        postgres.NewLessonRepo(database),
		Homework:      postgres.NewHomeworkRepo(database),
		Quiz:          postgres.NewQuizRepo(database),
		Subscription:  postgres.NewSubscriptionRepo(database),
		FreeTrial:     postgres.NewFreeTrialRepo(database),
		Payment:       postgres.NewPaymentRepo(database),
		PasswordReset: postgres.NewPasswordResetRepo(database),
		Notification:  postgres.NewNotificationRepo(database),
	}

	planRepo := postgres.NewPlanRepo(database)
	emailRepo := postgres.NewEmailRepo(database)
	settingsRepo := postgres.NewSettingsRepo(database)
	adminStatsRepo := postgres.NewAdminStatsRepo(database)
	analyticsRepo := postgres.NewAnalyticsRepo(database)

	tokenManager := auth.NewTokenManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
	)
	serviceTokenHandler := handler.NewServiceTokenHandler(tokenManager, log)

	var sender mailer.Sender
	if cfg.Email.Enabled() {
		sender = mailer.NewResendSender(cfg.Email.ResendAPIKey, cfg.Email.From, cfg.Email.ReplyTo)
		log.Info("mailer enabled", "provider", "resend", "from", cfg.Email.From)
	} else {
		sender = mailer.NewNoopSender(log)
		log.Warn("mailer using noop sender")
	}

	brandingProvider := mailer.NewBrandingProvider(database)
	mailerSvc := mailer.NewService(sender, emailRepo, brandingProvider, log, cfg.App.Name, cfg.Email.WebBaseURL)
	notificationSvc := service.NewNotificationService(repos.Notification, log)
	activitySvc := service.NewActivityService(database, log)

	if err := seedEmailTemplates(ctx, emailRepo, log); err != nil {
		log.Warn("email template seed failed", "error", err)
	}

	go mailerSvc.Run(ctx)

	expiryScheduler := scheduler.NewExpiryScheduler(
		database,
		mailerSvc,
		notificationSvc,
		log,
		cfg.Email.WebBaseURL,
	)
	go expiryScheduler.Run(ctx)

	authSvc := service.NewAuthService(
		repos.User,
		repos.PasswordReset,
		mailerSvc,
		activitySvc,
		cfg.Email.WebBaseURL,
		tokenManager,
	)
	userSvc := service.NewUserService(repos.User)
	subSvc := service.NewSubscriptionService(repos.Subscription, planRepo, repos.FreeTrial, repos.User, repos.Payment)
	lessonSvc := service.NewLessonService(repos.Lesson, repos.User, notificationSvc, activitySvc)
	homeworkSvc := service.NewHomeworkService(repos.Homework, repos.User, notificationSvc, activitySvc)
	quizSvc := service.NewQuizService(repos.Quiz, repos.User, notificationSvc, activitySvc)

	providers := payments.NewRegistry(cfg.Momo, cfg.Airtel)
	log.Info("payment providers", "enabled", providers.Enabled())

	paySvc := service.NewPaymentService(
		repos.Payment,
		subSvc,
		planRepo,
		providers,
		repos.User,
		mailerSvc,
		notificationSvc,
		log,
		cfg.Momo.Currency,
	)

	campaignSvc := service.NewCampaignService(emailRepo, mailerSvc, log)
	analyticsSvc := service.NewAnalyticsService(analyticsRepo)

	crudConfig, err := crud.Load("./configs/resources.yaml")
	if err != nil {
		return fmt.Errorf("load crud config: %w", err)
	}
	log.Info("crud resources loaded", "count", len(crudConfig.Resources))

	crudRepo := crud.NewRepository(database)

	var uploadHandler *handler.UploadHandler
	if cfg.R2.Enabled() {
		r2Client, err := storage.NewR2Client(ctx, cfg.R2)
		if err != nil {
			return fmt.Errorf("init r2: %w", err)
		}
		uploadHandler = handler.NewUploadHandler(r2Client, log)
		log.Info("r2 storage enabled", "bucket", cfg.R2.Bucket)
	} else {
		log.Warn("r2 storage is not configured")
	}

	configHandler := handler.NewConfigHandler(settingsRepo, log)
	adminHandler := handler.NewAdminHandler(adminStatsRepo, log)
	adminUserHandler := handler.NewAdminUserHandler(
		repos.User,
		repos.Quiz,
		repos.Homework,
		repos.PasswordReset,
		mailerSvc,
		cfg.Email.WebBaseURL,
		log,
	)
	notificationHandler := handler.NewNotificationHandler(notificationSvc, log)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsSvc, log)

	router := server.NewRouter(server.RouterDeps{
		Log:                 log,
		Version:             version,
		DB:                  database,
		Repos:               repos,
		Tokens:              tokenManager,
		AuthSvc:             authSvc,
		UserSvc:             userSvc,
		LessonSvc:           lessonSvc,
		HomeworkSvc:         homeworkSvc,
		QuizSvc:             quizSvc,
		SubSvc:              subSvc,
		PaySvc:              paySvc,
		CampaignSvc:         campaignSvc,
		NotificationSvc:     notificationSvc,
		ConfigHandler:       configHandler,
		AdminHandler:        adminHandler,
		AdminUserHandler:    adminUserHandler,
		NotificationHandler: notificationHandler,
		AnalyticsHandler:    analyticsHandler,
		CRUDConfig:          crudConfig,
		CRUDRepo:            crudRepo,
		UploadHandler:       uploadHandler,
		ServiceTokenHandler: serviceTokenHandler,
	})

	addr := fmt.Sprintf(":%d", cfg.App.Port)
	srv := server.New(addr, router, log)

	return srv.Run(ctx)
}

func seedEmailTemplates(ctx context.Context, repo *postgres.EmailRepo, log *slog.Logger) error {
	for _, t := range mailer.DefaultTemplates {
		_, err := repo.UpsertTemplate(ctx, &domain.EmailTemplate{
			Name:    t.Name,
			Subject: t.Subject,
			Body:    t.Body,
		})
		if err != nil {
			return fmt.Errorf("seed %q: %w", t.Name, err)
		}
	}
	log.Info("email templates seeded", "count", len(mailer.DefaultTemplates))
	return nil
}
