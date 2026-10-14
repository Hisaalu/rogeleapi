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
		User:         postgres.NewUserRepo(database),
		Lesson:       postgres.NewLessonRepo(database),
		Homework:     postgres.NewHomeworkRepo(database),
		Quiz:         postgres.NewQuizRepo(database),
		Subscription: postgres.NewSubscriptionRepo(database),
		FreeTrial:    postgres.NewFreeTrialRepo(database),
		Payment:      postgres.NewPaymentRepo(database),
	}

	planRepo := postgres.NewPlanRepo(database)
	emailRepo := postgres.NewEmailRepo(database)

	tokenManager := auth.NewTokenManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
	)

	var sender mailer.Sender
	if cfg.Email.Enabled() {
		sender = mailer.NewResendSender(cfg.Email.ResendAPIKey, cfg.Email.From, cfg.Email.ReplyTo)
		log.Info("mailer enabled", "provider", "resend", "from", cfg.Email.From)
	} else {
		sender = mailer.NewNoopSender(log)
		log.Warn("mailer using noop sender — emails will be logged, not sent")
	}

	mailerSvc := mailer.NewService(sender, emailRepo, log, cfg.App.Name, cfg.Email.WebBaseURL)

	if err := seedEmailTemplates(ctx, emailRepo, log); err != nil {
		log.Warn("email template seed failed", "error", err)
	}

	go mailerSvc.Run(ctx)

	authSvc := service.NewAuthService(repos.User, tokenManager)
	userSvc := service.NewUserService(repos.User)
	lessonSvc := service.NewLessonService(repos.Lesson, repos.User)
	homeworkSvc := service.NewHomeworkService(repos.Homework, repos.User)
	quizSvc := service.NewQuizService(repos.Quiz, repos.User)
	subSvc := service.NewSubscriptionService(repos.Subscription, planRepo, repos.FreeTrial, repos.User, repos.Payment)

	providers := payments.NewRegistry(cfg.Momo, cfg.Airtel)
	log.Info("payment providers", "enabled", providers.Enabled())

	paySvc := service.NewPaymentService(repos.Payment, subSvc, planRepo, providers, cfg.Momo.Currency)

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
		log.Warn("r2 storage is not configured — upload endpoints disabled")
	}

	router := server.NewRouter(server.RouterDeps{
		Log:           log,
		Version:       version,
		DB:            database,
		Repos:         repos,
		Tokens:        tokenManager,
		AuthSvc:       authSvc,
		UserSvc:       userSvc,
		LessonSvc:     lessonSvc,
		HomeworkSvc:   homeworkSvc,
		QuizSvc:       quizSvc,
		SubSvc:        subSvc,
		PaySvc:        paySvc,
		CRUDConfig:    crudConfig,
		CRUDRepo:      crudRepo,
		UploadHandler: uploadHandler,
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
