package adapter

import "database/sql"

func (m *MySQLPetAdapter) insertNotificationTx(
	tx *sql.Tx,
	recipientID string,
	notificationType string,
	title string,
	message string,
	petID *int,
	rehomeID *int,
) error {
	_, err := tx.Exec(`
		INSERT INTO Notifications (
			notification_userId, notification_type, notification_title,
			notification_message, notification_petId, notification_rehomeId
		) VALUES (?, ?, ?, ?, ?, ?)
	`, recipientID, notificationType, title, message, petID, rehomeID)
	return err
}
