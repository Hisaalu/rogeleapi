package server

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/Hisaalu/rogele/internal/auth"
	"github.com/Hisaalu/rogele/internal/crud"
	"github.com/Hisaalu/rogele/internal/handler"
	"github.com/Hisaalu/rogele/internal/repository"
	"github.com/Hisaalu/rogele/internal/server/middleware"
	"github.com/Hisaalu/rogele/internal/service"
)

type RouterDeps struct {
	Log           *slog.Logger
	Version       string
	DB            *sql.DB
	Repos         *repository.Repositories
	Tokens        *auth.TokenManager
	AuthSvc       *service.AuthService
	UserSvc       *service.UserService
	LessonSvc     *service.LessonService
	HomeworkSvc   *service.HomeworkService
	QuizSvc       *service.QuizService
	SubSvc        *service.SubscriptionService
	PaySvc        *service.PaymentService
	CRUDConfig    *crud.Config
	CRUDRepo      *crud.Repository
	UploadHandler *handler.UploadHandler
	CampaignSvc   *service.CampaignService
}

func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(deps.Log))
	r.Use(middleware.Recover(deps.Log))
	r.Use(middleware.CORS)
	r.Use(chimw.Compress(5))

	health := handler.NewHealthHandler(deps.Version)
	r.Get("/health", health.Check)

	r.Route("/api/v1", func(r chi.Router) {
		authHandler := handler.NewAuthHandler(deps.AuthSvc, deps.UserSvc, deps.Tokens, deps.Log)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/forgot-password", authHandler.ForgotPassword)
			r.Post("/reset-password", authHandler.ResetPassword)

			r.Group(func(r chi.Router) {
				r.Use(middleware.Authenticate(deps.Tokens))
				r.Get("/me", authHandler.Me)
			})
		})

		lessonHandler := handler.NewLessonHandler(deps.LessonSvc, deps.Log)
		r.Route("/lessons", func(r chi.Router) {
			r.Use(middleware.Authenticate(deps.Tokens))
			r.Get("/", lessonHandler.List)
			r.Get("/bookmarks", lessonHandler.ListBookmarks)
			r.Post("/", lessonHandler.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", lessonHandler.Get)
				r.Patch("/", lessonHandler.Update)
				r.Delete("/", lessonHandler.Delete)
				r.Post("/publish", lessonHandler.Publish)
				r.Post("/approve", lessonHandler.Approve)
				r.Post("/bookmark", lessonHandler.ToggleBookmark)
				r.Get("/materials", lessonHandler.ListMaterials)
				r.Post("/materials", lessonHandler.AddMaterial)
				r.Delete("/materials/{materialId}", lessonHandler.DeleteMaterial)
			})
		})

		homeworkHandler := handler.NewHomeworkHandler(deps.HomeworkSvc, deps.Log)
		r.Route("/homework", func(r chi.Router) {
			r.Use(middleware.Authenticate(deps.Tokens))
			r.Get("/", homeworkHandler.List)
			r.Get("/my-submissions", homeworkHandler.MySubmissions)
			r.Post("/", homeworkHandler.Create)
			r.Post("/submissions/{submissionId}/grade", homeworkHandler.Grade)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", homeworkHandler.Get)
				r.Patch("/", homeworkHandler.Update)
				r.Delete("/", homeworkHandler.Delete)
				r.Post("/submit", homeworkHandler.Submit)
				r.Get("/submissions", homeworkHandler.ListSubmissions)
			})
		})

		quizHandler := handler.NewQuizHandler(deps.QuizSvc, deps.Log)
		r.Route("/quizzes", func(r chi.Router) {
			r.Use(middleware.Authenticate(deps.Tokens))
			r.Get("/", quizHandler.List)
			r.Get("/my-attempts", quizHandler.ListMyAttempts)
			r.Post("/", quizHandler.Create)
			r.Get("/attempts/{attemptId}", quizHandler.GetAttempt)
			r.Post("/attempts/{attemptId}/submit", quizHandler.SubmitAttempt)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", quizHandler.Get)
				r.Patch("/", quizHandler.Update)
				r.Delete("/", quizHandler.Delete)
				r.Post("/approve", quizHandler.Approve)
				r.Get("/questions", quizHandler.ListQuestions)
				r.Post("/questions", quizHandler.AddQuestion)
				r.Delete("/questions/{questionId}", quizHandler.DeleteQuestion)
				r.Post("/attempts", quizHandler.StartAttempt)
			})
		})

		subHandler := handler.NewSubscriptionHandler(deps.SubSvc, deps.Log)
		r.Route("/subscriptions", func(r chi.Router) {
			r.Use(middleware.Authenticate(deps.Tokens))
			r.Get("/me", subHandler.MyActive)
			r.Get("/", subHandler.MyList)
			r.Get("/all", subHandler.ListAll)
			r.Post("/{id}/cancel", subHandler.Cancel)
		})

		r.Route("/trials", func(r chi.Router) {
			r.Use(middleware.Authenticate(deps.Tokens))
			r.Get("/me", subHandler.MyTrial)
			r.Post("/", subHandler.StartTrial)
		})

		payHandler := handler.NewPaymentHandler(deps.PaySvc, deps.Log)
		r.Route("/payments", func(r chi.Router) {
			r.Use(middleware.Authenticate(deps.Tokens))
			r.Post("/initiate", payHandler.Initiate)
			r.Get("/me", payHandler.MyList)
			r.Get("/history", payHandler.MyHistory)
			r.Get("/all", payHandler.ListAll)
			r.Get("/{id}/status", payHandler.GetStatus)
		})

		campaignHandler := handler.NewCampaignHandler(deps.CampaignSvc, deps.Log)
		r.Route("/admin/email-campaigns", func(r chi.Router) {
			r.Use(middleware.Authenticate(deps.Tokens))
			r.Get("/", campaignHandler.List)
			r.Post("/", campaignHandler.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", campaignHandler.Get)
				r.Patch("/", campaignHandler.Update)
				r.Delete("/", campaignHandler.Delete)
				r.Post("/preview", campaignHandler.Preview)
				r.Post("/send", campaignHandler.Send)
				r.Get("/logs", campaignHandler.Logs)
			})
		})

		webhookHandler := handler.NewWebhookHandler(deps.PaySvc, deps.Log)
		r.Post("/webhooks/mtn", webhookHandler.MTN)
		r.Post("/webhooks/airtel", webhookHandler.Airtel)

		if deps.UploadHandler != nil {
			r.Route("/uploads", func(r chi.Router) {
				r.Use(middleware.Authenticate(deps.Tokens))
				r.Post("/presign", deps.UploadHandler.Presign)
			})
		}

		if deps.CRUDConfig != nil && deps.CRUDRepo != nil {
			crud.Mount(r, deps.CRUDConfig, deps.CRUDRepo, deps.Tokens, deps.Log)
		}
	})

	return r
}
