package service

import (
	"context"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
	"github.com/S-nudhana/stray2stay/internal/core/port"
)

type NotificationService interface {
	GetNotifications(ctx context.Context, uid string) (notifications []domain.Notification, unreadCount int, err error)
	MarkNotificationRead(ctx context.Context, uid string, notificationID int) error
	MarkAllNotificationsRead(ctx context.Context, uid string) error
}

type NotificationServiceImpl struct {
	repo port.NotificationRepository
}

func NewNotificationService(repo port.NotificationRepository) NotificationService {
	return &NotificationServiceImpl{repo: repo}
}

func (s *NotificationServiceImpl) GetNotifications(ctx context.Context, uid string) (notifications []domain.Notification, unreadCount int, err error) {
	return s.repo.GetNotifications(uid)
}

func (s *NotificationServiceImpl) MarkNotificationRead(ctx context.Context, uid string, notificationID int) error {
	return s.repo.MarkNotificationRead(uid, notificationID)
}

func (s *NotificationServiceImpl) MarkAllNotificationsRead(ctx context.Context, uid string) error {
	return s.repo.MarkAllNotificationsRead(uid)
}
