package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/back2nix/devils/internal/logger"
	"github.com/back2nix/devils/internal/models" // Corrected import path
	"github.com/back2nix/devils/internal/subscription"
	"github.com/back2nix/devils/internal/ws_centrifuge" // For payload types
)

// PremiumSessionRequest represents a user's request to initiate a premium session.
type PremiumSessionRequest struct {
	ChatID   string `json:"chatId"`   // Model's ID
	UserID   string `json:"userId"`   // User's ID
	Duration int    `json:"duration"` // Duration in seconds
}

// PremiumSessionInitiateResponse is the response when a premium session request is successfully made.
type PremiumSessionInitiateResponse struct {
	Session     *models.PremiumSession `json:"session"` // The pending session object
	Message     string                 `json:"message"`
	Remaining   int                    `json:"remaining"` // Remaining *free* sessions from subscription
	UsedFree    bool                   `json:"usedFree"`
	PaidSession bool                   `json:"paidSession"`
}

// Prices for premium sessions: duration_in_seconds -> price_in_cents
var premiumSessionPrices = map[int]int64{
	300: 1800, // 5 min = $18.00
	600: 3200, // 10 min = $32.00
	900: 4500, // 15 min = $45.00
}

const (
	fiveMinuteDurationInSeconds = 300
	tenMinuteDurationInSeconds  = 600 // Added for 10-minute free session logic
)

// RequestPremiumSession handles a user's request to start a premium session.
// It creates a pending session and notifies the model.
// Renamed from CreatePremiumSession.
func RequestPremiumSession(d Dependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		request, err := parseAndValidateRequest(c)
		if err != nil {
			return err // parseAndValidateRequest already sends response
		}

		chatObjID, userObjID, err := convertIDs(request.ChatID, request.UserID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		user, err := d.GetDB().GetUserByID(c.Context(), request.UserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).
				JSON(fiber.Map{"error": "Failed to get user details"})
		}

		remainingFreeSessions := subscription.GetRemainingPremiumSessions(&user)
		usedFreeSession := false
		paidForSession := false
		paidAmount := int64(0)
		sessionsToConsume := 0

		// Check if user wants to use a free session and is eligible
		if request.Duration == fiveMinuteDurationInSeconds && remainingFreeSessions >= 1 {
			sessionsToConsume = 1
		} else if request.Duration == tenMinuteDurationInSeconds && remainingFreeSessions >= 2 {
			sessionsToConsume = 2
		}

		if sessionsToConsume > 0 {
			for i := 0; i < sessionsToConsume; i++ {
				if errUsage := incrementPremiumUsage(c, d, userObjID); errUsage != nil {
					return errUsage // incrementPremiumUsage already sends response
				}
			}
			usedFreeSession = true
		} else {
			// No eligible free session path taken, or user selected a duration not covered by free allowance.
			// Proceed to payment logic.
			isEligibleToPay := user.SubscriptionPlan == subscription.PlanAdvanced || user.SubscriptionPlan == subscription.PlanUltimate
			if isEligibleToPay {
				price, ok := premiumSessionPrices[request.Duration]
				if !ok {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid session duration for purchase"})
				}
				paidAmount = price

				currentBalance, errWallet := d.GetWallet().GetBalance(c.Context(), userObjID)
				if errWallet != nil {
					logger.Errorf("Failed to get user balance for premium session purchase: %v", errWallet)
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to check balance"})
				}

				if currentBalance < price {
					return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
						"error":           "Insufficient funds",
						"requiresPayment": true,
						"paymentDetails":  fiber.Map{"amount": price, "currency": "USD"},
					})
				}

				errSpend := d.GetWallet().SpendMoney(c.Context(), userObjID, price, "Paid Premium Chat Session Request", chatObjID, primitive.NilObjectID, models.TransactionContentTypePremiumChat)
				if errSpend != nil {
					logger.Errorf("Failed to spend money for premium session: %v", errSpend)
					if errors.Is(errSpend, models.ErrInsufficientFunds) {
						return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
							"error":           "Insufficient funds after attempt",
							"requiresPayment": true,
							"paymentDetails":  fiber.Map{"amount": price, "currency": "USD"},
						})
					}
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Payment processing failed"})
				}
				paidForSession = true
			} else {
				_, durationIsPurchasable := premiumSessionPrices[request.Duration]
				if !durationIsPurchasable {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid session duration"})
				}
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error":                "Subscription required to purchase premium sessions or upgrade plan",
					"requiresSubscription": true,
				})
			}
		}

		// Create a PENDING premium session in the database
		pendingSession, err := d.GetDB().CreatePendingPremiumSession(
			c.Context(),
			chatObjID, // Model's ID
			userObjID, // User's ID
			request.Duration,
			paidAmount, // Store the amount paid (0 if free)
		)
		if err != nil {
			// TODO: If payment was made but session creation failed, a refund mechanism might be needed.
			// For Stage 1, we assume this is rare.
			return c.Status(fiber.StatusInternalServerError).
				JSON(fiber.Map{"error": "Failed to create pending premium session"})
		}

		// Create a WebSocket message to notify the model via the reliable publisher
		wsMessage := models.Message{
			ID:              primitive.NewObjectID(),         // New ID for this specific message
			RequestID:       pendingSession.ID.Hex(),         // Link to the PremiumSession object
			Type:            "premium_session_request",       // New message type
			ChatID:          pendingSession.ChatID,           // Model's ID
			SenderID:        pendingSession.UserID,           // User's ID
			Username:        user.Name,                       // Requesting user's name
			Timestamp:       time.Now(),
			DurationSeconds: pendingSession.DurationSeconds,
			PaidAmount:      pendingSession.PaidAmount,
			Content:         fmt.Sprintf("%s requested a %d min premium session.", user.Name, request.Duration/60),
		}

		err = d.GetWebsocket().PublishMessageWithTracking(&wsMessage)
		if err != nil {
			logger.Errorf(
				"RequestPremiumSession: Error publishing WebSocket message for premium session request %s (User: %s, Model: %s): %v",
				pendingSession.ID.Hex(),
				userObjID.Hex(),
				chatObjID.Hex(),
				err,
			)
			// Continue, but log error. The session is in DB. Model will get it on next subscribe or if WS eventually delivers.
		}

		// Notify the user that their request is pending (can remain direct for simplicity or also be converted)
		userNotificationPayload := ws_centrifuge.PremiumSessionRequestPendingUserPayload{
			Type:      "premium_session_request_pending_user",
			SessionID: pendingSession.ID.Hex(),
			ModelID:   chatObjID.Hex(),
		}
		userNotificationData, _ := json.Marshal(userNotificationPayload)
		userChannel := fmt.Sprintf("user:%s", userObjID.Hex()) // Requesting user's channel
		if err := d.GetWebsocket().PublishEventToChannel(userChannel, userNotificationData); err != nil {
			logger.Errorf(
				"Failed to publish premium session pending status to user %s: %v",
				userObjID.Hex(),
				err,
			)
		}

		// Update remaining free sessions for response
		updatedUserForResponse, _ := d.GetDB().GetUserByID(c.Context(), request.UserID)
		remainingFreeForResponse := 0
		if updatedUserForResponse.ID != primitive.NilObjectID {
			remainingFreeForResponse = subscription.GetRemainingPremiumSessions(
				&updatedUserForResponse,
			)
		}

		return c.Status(fiber.StatusOK).JSON(PremiumSessionInitiateResponse{
			Message:     "Premium session request sent. Waiting for model to accept.",
			Session:     pendingSession, // Return the pending session object
			Remaining:   remainingFreeForResponse,
			UsedFree:    usedFreeSession,
			PaidSession: paidForSession,
		})
	}
}
