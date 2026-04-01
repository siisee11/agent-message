package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"agent-message/server/models"
	"agent-message/server/store"

	webpush "github.com/SherClockHolmes/webpush-go"
)

const defaultNotificationTTL = 60 * time.Second

type Config struct {
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	Subject         string
}

type Service struct {
	store           store.Store
	vapidPublicKey  string
	vapidPrivateKey string
	subject         string
}

func NewService(dataStore store.Store, cfg Config) (*Service, error) {
	service := &Service{
		store:           dataStore,
		vapidPublicKey:  strings.TrimSpace(cfg.VAPIDPublicKey),
		vapidPrivateKey: strings.TrimSpace(cfg.VAPIDPrivateKey),
		subject:         strings.TrimSpace(cfg.Subject),
	}

	if service.vapidPublicKey == "" && service.vapidPrivateKey == "" && service.subject == "" {
		return service, nil
	}
	if service.vapidPublicKey == "" || service.vapidPrivateKey == "" || service.subject == "" {
		return nil, errors.New("web push requires WEB_PUSH_VAPID_PUBLIC_KEY, WEB_PUSH_VAPID_PRIVATE_KEY, and WEB_PUSH_SUBJECT")
	}
	return service, nil
}

func (s *Service) Enabled() bool {
	return s != nil && s.vapidPublicKey != "" && s.vapidPrivateKey != "" && s.subject != ""
}

func (s *Service) PublicConfig() models.PushConfigResponse {
	if !s.Enabled() {
		return models.PushConfigResponse{Enabled: false}
	}
	return models.PushConfigResponse{
		Enabled:        true,
		VAPIDPublicKey: s.vapidPublicKey,
	}
}

func (s *Service) NotifyMessage(ctx context.Context, recipientUserID string, notification MessageNotification) error {
	if !s.Enabled() {
		return nil
	}

	subscriptions, err := s.store.ListPushSubscriptionsByUser(ctx, recipientUserID)
	if err != nil {
		return fmt.Errorf("list push subscriptions: %w", err)
	}
	if len(subscriptions) == 0 {
		return nil
	}

	payload, err := json.Marshal(notificationPayload{
		Title: notification.SenderName,
		Body:  notification.Preview,
		Tag:   "chat:" + notification.ConversationID,
		Data: notificationPayloadData{
			URL:            notification.URL,
			ConversationID: notification.ConversationID,
			MessageID:      notification.MessageID,
		},
	})
	if err != nil {
		return fmt.Errorf("marshal push payload: %w", err)
	}

	options := &webpush.Options{
		Subscriber:      s.subject,
		VAPIDPublicKey:  s.vapidPublicKey,
		VAPIDPrivateKey: s.vapidPrivateKey,
		TTL:             int(defaultNotificationTTL.Seconds()),
	}

	var sendErr error
	for _, subscription := range subscriptions {
		if err := s.sendSubscription(ctx, subscription, payload, options); err != nil {
			log.Printf("push send failed user=%s endpoint=%s: %v", recipientUserID, subscription.Endpoint, err)
			sendErr = err
		}
	}

	return sendErr
}

func (s *Service) sendSubscription(
	ctx context.Context,
	subscription models.PushSubscription,
	payload []byte,
	options *webpush.Options,
) error {
	resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
		Endpoint: subscription.Endpoint,
		Keys: webpush.Keys{
			Auth:   subscription.Auth,
			P256dh: subscription.P256DH,
		},
	}, options)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return fmt.Errorf("send notification: %w", err)
	}
	if resp == nil {
		return errors.New("push response missing")
	}

	switch resp.StatusCode {
	case http.StatusCreated, http.StatusAccepted:
		return nil
	case http.StatusGone, http.StatusNotFound:
		if err := s.store.DeletePushSubscriptionByEndpoint(ctx, subscription.Endpoint); err != nil && !errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("delete expired push subscription: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unexpected push status %d", resp.StatusCode)
	}
}

type notificationPayload struct {
	Title string                  `json:"title"`
	Body  string                  `json:"body"`
	Tag   string                  `json:"tag"`
	Data  notificationPayloadData `json:"data"`
}

type notificationPayloadData struct {
	URL            string `json:"url"`
	ConversationID string `json:"conversationId"`
	MessageID      string `json:"messageId"`
}
