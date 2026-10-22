package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/Hisaalu/rogele/internal/domain"
	"github.com/Hisaalu/rogele/internal/mailer"
	"github.com/Hisaalu/rogele/internal/payments"
	"github.com/Hisaalu/rogele/internal/repository"
)

type InitiatePaymentInput struct {
	PlanID      int64
	Method      string
	PhoneNumber string
}

type InitiatePaymentResult struct {
	Payment           *domain.Payment
	ProviderReference string
	Status            string
}

type PaymentService struct {
	payments      repository.PaymentRepository
	subscriptions *SubscriptionService
	plans         planLookup
	providers     *payments.Registry
	users         repository.UserRepository
	mailer        *mailer.Service
	notifications *NotificationService
	log           *slog.Logger
	currency      string
}

func NewPaymentService(
	payments repository.PaymentRepository,
	subscriptions *SubscriptionService,
	plans planLookup,
	providers *payments.Registry,
	users repository.UserRepository,
	mailerSvc *mailer.Service,
	notifications *NotificationService,
	log *slog.Logger,
	currency string,
) *PaymentService {
	return &PaymentService{
		payments:      payments,
		subscriptions: subscriptions,
		plans:         plans,
		providers:     providers,
		users:         users,
		mailer:        mailerSvc,
		notifications: notifications,
		log:           log,
		currency:      currency,
	}
}

func (s *PaymentService) Initiate(ctx context.Context, actor ActorContext, in InitiatePaymentInput) (*InitiatePaymentResult, error) {
	if in.PhoneNumber == "" {
		return nil, fmt.Errorf("%w: phone number is required", domain.ErrInvalidInput)
	}
	method := strings.ToLower(strings.TrimSpace(in.Method))
	provider, err := s.providers.Get(method)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidInput, err.Error())
	}
	if !provider.Enabled() {
		return nil, fmt.Errorf("%w: %s payment provider is not configured", domain.ErrInvalidInput, method)
	}

	plan, err := s.plans.GetPlan(ctx, in.PlanID)
	if err != nil {
		return nil, err
	}
	if !plan.IsActive {
		return nil, fmt.Errorf("%w: plan is not active", domain.ErrInvalidInput)
	}

	amountToPay := plan.Price

	if currentSub, err := s.subscriptions.GetMyActive(ctx, actor); err == nil && currentSub != nil {
		now := time.Now()
		if currentSub.EndDate.After(now) {
			currentPlanID, _ := strconv.ParseInt(string(currentSub.PlanType), 10, 64)
			currentPlan, err := s.plans.GetPlan(ctx, currentPlanID)

			if err == nil && currentPlan != nil {
				if plan.Price > currentPlan.Price {
					daysLeft := currentSub.EndDate.Sub(now).Hours() / 24
					if daysLeft > float64(currentPlan.DurationDays) {
						daysLeft = float64(currentPlan.DurationDays)
					}
					dailyRate := currentPlan.Price / float64(currentPlan.DurationDays)
					credit := dailyRate * daysLeft
					amountToPay = plan.Price - credit
					if amountToPay < 0 {
						amountToPay = 0
					}
				}
			}
		}
	}

	uid := actor.UserID
	planRef := strconv.FormatInt(plan.ID, 10)
	methodCopy := method

	payment := &domain.Payment{
		UserID:        &uid,
		Amount:        &amountToPay,
		PlanType:      &planRef,
		PaymentMethod: &methodCopy,
		PhoneNumber:   &in.PhoneNumber,
		Status:        domain.PaymentPending,
	}
	created, err := s.payments.Create(ctx, payment)
	if err != nil {
		return nil, err
	}

	externalID := fmt.Sprintf("rogeledb-%d", created.ID)
	reference := fmt.Sprintf("plan:%d", plan.ID)

	result, err := provider.Initiate(ctx, payments.InitiateRequest{
		Amount:     amountToPay,
		Currency:   s.currency,
		Phone:      in.PhoneNumber,
		ExternalID: externalID,
		Reference:  reference,
		PayerMsg:   "Rogele subscription",
		PayeeMsg:   "Rogele",
	})
	if err != nil {
		created.Status = domain.PaymentFailed
		_ = s.payments.Update(ctx, created)
		return nil, fmt.Errorf("initiate payment: %w", err)
	}

	txnID := result.ProviderReference
	created.TransactionID = &txnID
	if result.Raw != nil {
		created.PaymentGatewayResponse = result.Raw
	}
	if err := s.payments.Update(ctx, created); err != nil {
		return nil, err
	}

	return &InitiatePaymentResult{
		Payment:           created,
		ProviderReference: result.ProviderReference,
		Status:            result.Status,
	}, nil
}

func (s *PaymentService) GetStatus(ctx context.Context, id int64, actor ActorContext) (*domain.Payment, error) {
	p, err := s.payments.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.UserID != nil && *p.UserID != actor.UserID && actor.Role != domain.RoleAdmin {
		return nil, domain.ErrForbidden
	}
	if p.Status == domain.PaymentCompleted || p.Status == domain.PaymentFailed || p.Status == domain.PaymentRefunded {
		return p, nil
	}
	if p.PaymentMethod == nil || p.TransactionID == nil {
		return p, nil
	}

	provider, err := s.providers.Get(*p.PaymentMethod)
	if err != nil || !provider.Enabled() {
		return p, nil
	}

	result, err := provider.Status(ctx, *p.TransactionID)
	if err != nil {
		return p, nil
	}

	if err := s.applyProviderStatus(ctx, p, result.Status, result.Reason, result.Raw); err != nil {
		return nil, err
	}
	return s.payments.GetByID(ctx, id)
}

func (s *PaymentService) HandleWebhook(ctx context.Context, method string, payload map[string]any) error {
	provider, err := s.providers.Get(method)
	if err != nil {
		return fmt.Errorf("%w: unknown provider %q", domain.ErrInvalidInput, method)
	}
	if !provider.Enabled() {
		return fmt.Errorf("%w: provider %s not configured", domain.ErrInvalidInput, method)
	}

	ref := extractReference(payload)
	if ref == "" {
		return fmt.Errorf("%w: missing reference in webhook payload", domain.ErrInvalidInput)
	}

	p, err := s.payments.GetByTransactionID(ctx, ref)
	if err != nil {
		return err
	}

	status := extractStatus(payload)
	return s.applyProviderStatus(ctx, p, status, "", payload)
}

func (s *PaymentService) ListForUser(ctx context.Context, actor ActorContext) ([]*domain.Payment, error) {
	uid := actor.UserID
	return s.payments.List(ctx, repository.PaymentFilter{UserID: &uid})
}

func (s *PaymentService) ListAll(ctx context.Context, actor ActorContext, f repository.PaymentFilter) ([]*domain.Payment, error) {
	if actor.Role != domain.RoleAdmin {
		return nil, fmt.Errorf("%w: admin only", domain.ErrForbidden)
	}
	return s.payments.ListWithUsers(ctx, f)
}

func (s *PaymentService) ListHistory(ctx context.Context, actor ActorContext) ([]*domain.PaymentHistory, error) {
	return s.payments.ListHistoryByUser(ctx, actor.UserID)
}

func (s *PaymentService) applyProviderStatus(
	ctx context.Context,
	p *domain.Payment,
	providerStatus string,
	reason string,
	raw map[string]any,
) error {
	normalized := strings.ToLower(providerStatus)
	var newStatus domain.PaymentStatus
	switch normalized {
	case "successful", "success", "completed", "paid":
		newStatus = domain.PaymentCompleted
	case "failed", "rejected", "cancelled":
		newStatus = domain.PaymentFailed
	default:
		return nil
	}

	if p.Status == newStatus {
		return nil
	}

	now := time.Now()
	p.Status = newStatus
	if raw != nil {
		p.PaymentGatewayResponse = raw
	}
	if newStatus == domain.PaymentCompleted {
		p.PaymentDate = &now
	}
	if err := s.payments.Update(ctx, p); err != nil {
		return err
	}

	if newStatus == domain.PaymentCompleted && p.UserID != nil && p.Amount != nil && p.PlanType != nil {
		planID, parseErr := strconv.ParseInt(*p.PlanType, 10, 64)
		if parseErr == nil {
			plan, err := s.plans.GetPlan(ctx, planID)
			if err == nil {
				sub, err := s.subscriptions.ActivateFromPayment(
					ctx, *p.UserID, plan, p.ID,
					derefString(p.PaymentMethod),
					derefString(p.TransactionID),
				)
				if err == nil && sub != nil {
					p.SubscriptionID = &sub.ID
					_ = s.payments.Update(ctx, p)
					_ = s.subscriptions.MarkConverted(ctx, *p.UserID)
				}
			}
		}
	}

	history := &domain.PaymentHistory{
		UserID:         derefInt64(p.UserID),
		SubscriptionID: p.SubscriptionID,
		Amount:         derefFloat(p.Amount),
		PaymentType:    "subscription",
		Status:         string(newStatus),
		PaymentMethod:  p.PaymentMethod,
		TransactionID:  p.TransactionID,
	}
	_, _ = s.payments.CreateHistory(ctx, history)

	if newStatus == domain.PaymentCompleted {
		s.sendPaymentConfirmation(ctx, p)
	}
	s.notifyPaymentStatus(ctx, p, newStatus)

	return nil
}

func (s *PaymentService) notifyPaymentStatus(ctx context.Context, p *domain.Payment, status domain.PaymentStatus) {
	if s.notifications == nil || p.UserID == nil {
		return
	}

	link := "/learner/subscription"

	var title, message string
	switch status {
	case domain.PaymentCompleted:
		title = "Payment received"
		amount := 0.0
		if p.Amount != nil {
			amount = *p.Amount
		}
		message = fmt.Sprintf("Your payment of UGX %.2f was successful.", amount)
	case domain.PaymentFailed:
		title = "Payment failed"
		message = "Your recent payment did not go through. Please try again."
	default:
		return
	}

	var notifType domain.NotificationType
	if status == domain.PaymentCompleted {
		notifType = domain.NotificationTypePayment
	} else {
		notifType = domain.NotificationTypePayment
	}

	_ = s.notifications.NotifyUser(ctx, *p.UserID, notifType, title, message, &link)
}

func (s *PaymentService) sendPaymentConfirmation(ctx context.Context, p *domain.Payment) {
	if s.mailer == nil || p.UserID == nil {
		return
	}

	u, err := s.users.GetByID(ctx, *p.UserID)
	if err != nil {
		s.log.Warn("payment confirmation: user lookup failed", "user_id", *p.UserID, "error", err)
		return
	}

	planName := "your subscription"
	if p.PlanType != nil {
		if planID, err := strconv.ParseInt(*p.PlanType, 10, 64); err == nil {
			if plan, err := s.plans.GetPlan(ctx, planID); err == nil {
				planName = plan.Name
			}
		}
	}

	amount := "0.00"
	if p.Amount != nil {
		amount = fmt.Sprintf("%.2f", *p.Amount)
	}

	method := "unknown"
	if p.PaymentMethod != nil {
		method = *p.PaymentMethod
	}

	txnID := ""
	if p.TransactionID != nil {
		txnID = *p.TransactionID
	}

	endDate := time.Now().AddDate(0, 0, 30).Format("January 2, 2006")
	if sub, err := s.subscriptions.GetMyActive(ctx, ActorContext{UserID: *p.UserID, Role: u.Role}); err == nil {
		endDate = sub.EndDate.Format("January 2, 2006")
	}

	vars := map[string]any{
		"FirstName":     derefOr(u.FirstName, "there"),
		"PlanName":      planName,
		"Amount":        amount,
		"Currency":      s.currency,
		"PaymentMethod": method,
		"TransactionID": txnID,
		"EndDate":       endDate,
	}

	if _, err := s.mailer.Enqueue(ctx, "payment_confirmation", u.Email, u.FirstName, vars); err != nil {
		s.log.Warn("payment confirmation enqueue failed", "payment_id", p.ID, "error", err)
	}
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func derefFloat(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func extractReference(payload map[string]any) string {
	for _, k := range []string{"referenceId", "reference_id", "externalId", "external_id", "transaction_id", "transactionId", "id"} {
		if v, ok := payload[k].(string); ok && v != "" {
			return v
		}
	}
	if data, ok := payload["data"].(map[string]any); ok {
		return extractReference(data)
	}
	return ""
}

func extractStatus(payload map[string]any) string {
	for _, k := range []string{"status", "state", "transaction_status"} {
		if v, ok := payload[k].(string); ok {
			return v
		}
	}
	if data, ok := payload["data"].(map[string]any); ok {
		return extractStatus(data)
	}
	return ""
}

func (s *PaymentService) calculateProratedAmount(ctx context.Context, actor ActorContext, plan *domain.SubscriptionPlan) float64 {
	amountToPay := plan.Price

	if currentSub, err := s.subscriptions.GetMyActive(ctx, actor); err == nil && currentSub != nil {
		now := time.Now()
		if currentSub.EndDate.After(now) {
			currentPlanID, _ := strconv.ParseInt(string(currentSub.PlanType), 10, 64)
			currentPlan, err := s.plans.GetPlan(ctx, currentPlanID)

			if err == nil && currentPlan != nil {
				if plan.Price > currentPlan.Price {
					daysLeft := currentSub.EndDate.Sub(now).Hours() / 24
					if daysLeft > float64(currentPlan.DurationDays) {
						daysLeft = float64(currentPlan.DurationDays)
					}
					dailyRate := currentPlan.Price / float64(currentPlan.DurationDays)
					credit := dailyRate * daysLeft
					amountToPay = plan.Price - credit
					if amountToPay < 0 {
						amountToPay = 0
					}
				}
			}
		}
	}
	return amountToPay
}

func (s *PaymentService) PreviewPayment(ctx context.Context, actor ActorContext, planID int64) (float64, error) {
	plan, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return 0, err
	}
	if !plan.IsActive {
		return 0, fmt.Errorf("%w: plan is not active", domain.ErrInvalidInput)
	}
	return s.calculateProratedAmount(ctx, actor, plan), nil
}
