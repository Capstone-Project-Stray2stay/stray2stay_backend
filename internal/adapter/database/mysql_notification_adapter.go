package adapter

import (
	"database/sql"
	"errors"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

type MySQLNotificationAdapter struct {
	db *sql.DB
}

func NewMySQLNotificationAdapter(db *sql.DB) *MySQLNotificationAdapter {
	return &MySQLNotificationAdapter{db: db}
}

func (m *MySQLNotificationAdapter) GetNotifications(uid string) (notifications []domain.Notification, unreadCount int, err error) {
	rows, err := m.db.Query(`
		SELECT notification_id, notification_type, notification_title,
		       notification_message, notification_petId, notification_rehomeId,
		       notification_isRead, notification_createAt
		FROM Notifications
		WHERE notification_userId = ?
		ORDER BY notification_createAt DESC, notification_id DESC
		LIMIT 50
	`, uid)
	if err != nil {
		return nil, 0, errors.New("failed to retrieve notifications")
	}
	defer rows.Close()

	notifications = make([]domain.Notification, 0)
	for rows.Next() {
		var notification domain.Notification
		var petID, rehomeID sql.NullInt64
		if err := rows.Scan(
			&notification.ID, &notification.Type, &notification.Title,
			&notification.Message, &petID, &rehomeID,
			&notification.IsRead, &notification.CreatedAt,
		); err != nil {
			return nil, 0, errors.New("failed to retrieve notifications")
		}
		if petID.Valid {
			id := int(petID.Int64)
			notification.PetID = &id
		}
		if rehomeID.Valid {
			id := int(rehomeID.Int64)
			notification.RehomeID = &id
		}
		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, errors.New("failed to retrieve notifications")
	}

	if err := m.db.QueryRow(`
		SELECT COUNT(*) FROM Notifications
		WHERE notification_userId = ? AND notification_isRead = FALSE
	`, uid).Scan(&unreadCount); err != nil {
		return nil, 0, errors.New("failed to count notifications")
	}

	return notifications, unreadCount, nil
}

func (m *MySQLNotificationAdapter) MarkNotificationRead(uid string, notificationID int) error {
	if _, err := m.db.Exec(`
		UPDATE Notifications
		SET notification_isRead = TRUE
		WHERE notification_id = ? AND notification_userId = ?
	`, notificationID, uid); err != nil {
		return errors.New("failed to mark notification as read")
	}
	return nil
}

func (m *MySQLNotificationAdapter) MarkAllNotificationsRead(uid string) error {
	if _, err := m.db.Exec(`
		UPDATE Notifications
		SET notification_isRead = TRUE
		WHERE notification_userId = ? AND notification_isRead = FALSE
	`, uid); err != nil {
		return errors.New("failed to mark notifications as read")
	}
	return nil
}
