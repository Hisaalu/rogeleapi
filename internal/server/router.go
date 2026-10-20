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
	Log                 *slog.Logger
	Version             string
	DB                  *sql.DB
	Repos               *repository.Repositories
	Tokens              *auth.TokenManager
	AuthSvc             *service.AuthService
	UserSvc             *service.UserService
	LessonSvc           *service.LessonService
	HomeworkSvc         *service.HomeworkService
	QuizSvc             *service.QuizService
	SubSvc              *service.SubscriptionService
	PaySvc              *service.PaymentService
	CampaignSvc         *service.CampaignService
	NotificationSvc     *service.NotificationService
	ConfigHandler       *handler.ConfigHandler
	AdminHandler        *handler.AdminHandler
	AdminUserHandler    *handler.AdminUserHandler
	NotificationHandler *handler.NotificationHandler
	CRUDConfig          *crud.Config
	CRUDRepo            *crud.Repository
	UploadHandler       *handler.UploadHandler
	AnalyticsHandler    *handler.AnalyticsHandler
	ServiceTokenHandler *handler.ServiceTokenHandler
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
		if deps.ConfigHandler != nil {
			r.Get("/config/public", deps.ConfigHandler.PublicConfig)
		}

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

		lessonHandler := handler.NewLessonHandler(deps.LessonSvc, deps.DB, deps.Log)
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
			r.Get("/submissions/{submissionId}/files", homeworkHandler.ListSubmissionFiles)
			r.Post("/submissions/{submissionId}/files", homeworkHandler.AddSubmissionFile)
			r.Delete("/submissions/{submissionId}/files/{fileId}", homeworkHandler.DeleteSubmissionFile)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", homeworkHandler.Get)
				r.Patch("/", homeworkHandler.Update)
				r.Delete("/", homeworkHandler.Delete)
				r.Post("/submit", homeworkHandler.Submit)
				r.Get("/submissions", homeworkHandler.ListSubmissions)
				r.Get("/attachments", homeworkHandler.ListAttachments)
				r.Post("/attachments", homeworkHandler.AddAttachment)
				r.Delete("/attachments/{attachmentId}", homeworkHandler.DeleteAttachment)
			})
		})

		quizHandler := handler.NewQuizHandler(deps.QuizSvc, deps.DB, deps.Log)
		r.Route("/quizzes", func(r chi.Router) {
			r.Use(middleware.Authenticate(deps.Tokens))
			r.Get("/", quizHandler.List)
			r.Get("/my-attempts", quizHandler.ListMyAttempts)
			r.Post("/", quizHandler.Create)
			r.Get("/attempts/{attemptId}", quizHandler.GetAttempt)
			r.Post("/attempts/{attemptId}/submit", quizHandler.SubmitAttempt)
			r.Post("/quizzes/{id}/questions/bulk", quizHandler.AddBulkQuestions)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", quizHandler.Get)
				r.Patch("/", quizHandler.Update)
				r.Delete("/", quizHandler.Delete)
				r.Post("/approve", quizHandler.Approve)
				r.Get("/questions", quizHandler.ListQuestions)
				r.Post("/questions", quizHandler.AddQuestion)
				r.Delete("/questions/{questionId}", quizHandler.DeleteQuestion)
				r.Post("/attempts", quizHandler.StartAttempt)
				r.Get("/attempts", quizHandler.ListAttempts)
				r.Delete("/attempts", quizHandler.DeleteAllAttempts)
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

		webhookHandler := handler.NewWebhookHandler(deps.PaySvc, deps.Log)
		r.Post("/webhooks/mtn", webhookHandler.MTN)
		r.Post("/webhooks/airtel", webhookHandler.Airtel)

		if deps.UploadHandler != nil {
			r.Route("/uploads", func(r chi.Router) {
				r.Use(middleware.Authenticate(deps.Tokens))
				r.Post("/presign", deps.UploadHandler.Presign)
			})
		}

		if deps.CampaignSvc != nil {
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
		}

		if deps.NotificationHandler != nil {
			r.Route("/notifications", func(r chi.Router) {
				r.Use(middleware.Authenticate(deps.Tokens))
				r.Get("/me", deps.NotificationHandler.ListMine)
				r.Get("/me/unread-count", deps.NotificationHandler.UnreadCount)
				r.Post("/{id}/read", deps.NotificationHandler.MarkRead)
				r.Post("/read-all", deps.NotificationHandler.MarkAllRead)
			})
		}

		if deps.AdminHandler != nil || deps.AdminUserHandler != nil || deps.NotificationHandler != nil {
			r.Route("/admin", func(r chi.Router) {
				r.Use(middleware.Authenticate(deps.Tokens))
				r.Get("/learners-by-class", deps.AdminHandler.LearnersByClass)

				if deps.AdminHandler != nil {
					r.Get("/stats", deps.AdminHandler.Overview)
					r.Get("/activity", deps.AdminHandler.RecentActivity)
					r.Get("/recent-signups", deps.AdminHandler.RecentSignups)
					r.Get("/recent-payments", deps.AdminHandler.RecentPayments)
				}

				if deps.ServiceTokenHandler != nil {
					r.Post("/service-tokens", deps.ServiceTokenHandler.Issue)
				}

				if deps.NotificationHandler != nil {
					r.Post("/notifications/broadcast", deps.NotificationHandler.Broadcast)
					r.Get("/notifications", deps.NotificationHandler.ListBroadcasts)
				}

				if deps.AdminUserHandler != nil {
					r.Get("/users/{id}/attempts", deps.AdminUserHandler.QuizAttempts)
					r.Get("/users/{id}/submissions", deps.AdminUserHandler.HomeworkSubmissions)
					r.Post("/users/{id}/force-reset", deps.AdminUserHandler.ForcePasswordReset)
					r.Post("/users/{id}/set-password", deps.AdminUserHandler.SetPassword)
				}

				if deps.NotificationHandler != nil {
					r.Post("/notifications/broadcast", deps.NotificationHandler.Broadcast)
				}
			})
		}

		if deps.AnalyticsHandler != nil {
			r.Route("/analytics", func(r chi.Router) {
				r.Use(middleware.Authenticate(deps.Tokens))
				r.Get("/enrollment-trend", deps.AnalyticsHandler.EnrollmentTrend)
				r.Get("/revenue-over-time", deps.AnalyticsHandler.RevenueSeries)
				r.Get("/score-distribution", deps.AnalyticsHandler.ScoreDistribution)
				r.Get("/user-events", deps.AnalyticsHandler.UserEvents)
				r.Get("/signup-dates", deps.AnalyticsHandler.SignupDates)
				r.Get("/learner-signals", deps.AnalyticsHandler.LearnerSignals)
			})
		}

		if deps.CRUDConfig != nil && deps.CRUDRepo != nil {
			crud.Mount(r, deps.CRUDConfig, deps.CRUDRepo, deps.Tokens, deps.Log)
		}
	})

	return r
}
