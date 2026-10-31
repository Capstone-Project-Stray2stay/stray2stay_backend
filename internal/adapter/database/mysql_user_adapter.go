package adapter

import (
	"database/sql"
	"errors"
	"os"
	"strconv"

	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

type MySQLUserAdapter struct {
	db *sql.DB
}

func NewMySQLUserAdapter(db *sql.DB) *MySQLUserAdapter {
	return &MySQLUserAdapter{db: db}
}

func (m *MySQLUserAdapter) CreateUser(email string, password string, firstName string, lastName string) (err error) {
	var existingEmail string
	_ = m.db.QueryRow("SELECT user_email FROM Users WHERE user_email = ?", email).Scan(&existingEmail)
	if existingEmail != "" {
		return errors.New("email already exists")
	}
	userId := uuid.New()
	cost, err := strconv.Atoi(os.Getenv("BCRYPT_COST"))
	if err != nil {
		cost = bcrypt.DefaultCost
	}
	hashPassword, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return errors.New("failed to hash password")
	}

	tx, err := m.db.Begin()
	if err != nil {
		return errors.New("failed to begin transaction")
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	_, err = tx.Exec(`
		INSERT INTO Users
		(user_id, user_email, user_firstname, user_lastname, user_phoneNumber, user_address, user_addressLat, user_addressLong, user_authType)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, userId, email, firstName, lastName, "", "", 0.00, 0.00, "PASS")
	if err != nil {
		return errors.New("failed to create user")
	}
	_, err = tx.Exec(`
		INSERT INTO Users_Password (password_userId, password_pass)
		VALUES (?, ?)
	`, userId, hashPassword)
	if err != nil {
		return errors.New("failed to create user")
	}
	err = tx.Commit()
	if err != nil {
		return errors.New("failed to create user")
	}
	return nil
}

func (m *MySQLUserAdapter) OAuthAuthenticateUser(
	email string,
	provider string,
	firstName string,
	lastName string,
) (uid string, err error) {
	var userUID string
	err = m.db.QueryRow(`
		SELECT user_id
		FROM Users
		WHERE user_email = ? AND user_authType = ?
	`, email, "OAUTH").Scan(&userUID)

	if err == nil {
		return userUID, nil
	}

	if err != sql.ErrNoRows {
		return "", err
	}

	newUID := uuid.New().String()
	oAuthUID := newUID + ":" + provider

	_, err = m.db.Exec(`
		INSERT INTO Users
		(user_id, user_email, user_firstname, user_lastname, user_phoneNumber, user_address, user_addressLat, user_addressLong, user_authType)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, oAuthUID, email, firstName, lastName, "", "", 0.00, 0.00, provider)

	if err != nil {
		return "", err
	}

	var userStatus bool
	err = m.db.QueryRow(`
		SELECT user_newUser
		FROM Users
		WHERE user_id = ?
	`, oAuthUID).Scan(&userStatus)

	if err != nil {
		return "", err
	}
	return oAuthUID, nil
}

func (m *MySQLUserAdapter) AuthenticateUser(email string, password string) (uid string, err error) {
	var storedPassword string
	var userId uuid.UUID

	err = m.db.QueryRow(`
		SELECT u.user_id, up.password_pass
		FROM Users AS u 
		JOIN Users_Password AS up ON u.user_id = up.password_userId  
		WHERE u.user_email = ? && u.user_authType = 'PASS'
	`, email).Scan(&userId, &storedPassword)
	if err != nil {
		return "", errors.New("user not found")
	}
	
	err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
	if err != nil {
		return "", errors.New("password does not match")
	}

	return userId.String(), nil
}

func (m *MySQLUserAdapter) RemoveUser(uid string) (err error) {
	_, err = m.db.Exec("DELETE FROM Users WHERE user_id = ?", uid)
	if err != nil {
		return errors.New("failed to delete user")
	}
	return nil
}

// nullIfEmpty stores "" as NULL — the age-group and gender columns are ENUMs that
// don't accept an empty string.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (m *MySQLUserAdapter) UpdateUserInfo(uid string, firstName string, lastName string, phoneNumber string, address string, addressLat float64, addressLong float64, preferences []domain.UserPreference) (err error) {
	tx, err := m.db.Begin()
	if err != nil {
		return errors.New("failed to update user")
	}
	// Rolls back unless Commit below succeeds, so the profile and its preferences change together
	defer tx.Rollback()

	result, err := tx.Exec(`UPDATE Users SET user_firstname = ?, user_lastname = ?, user_phoneNumber = ?, user_address = ?, user_addressLat = ?, user_addressLong = ? WHERE user_id = ?`, firstName, lastName, phoneNumber, address, addressLat, addressLong, uid)
	if err != nil {
		return errors.New("failed to update user")
	}

	rowsAffected, err := result.RowsAffected()
	if rowsAffected == 0 || err != nil {
		return errors.New("user not found")
	}

	// The submitted list is the whole set (any number per species, possibly none),
	// so replace the user's rows rather than patching them one by one.
	if _, err = tx.Exec(`DELETE FROM Users_Preferences WHERE pref_userId = ?`, uid); err != nil {
		return errors.New("failed to update user")
	}
	for _, pref := range preferences {
		_, err = tx.Exec(
			`INSERT INTO Users_Preferences (pref_userId, pref_petType, pref_breed, pref_color, pref_ageGroup, pref_gender) VALUES (?, ?, ?, ?, ?, ?)`,
			uid, pref.PetType, nullIfEmpty(pref.Breed), nullIfEmpty(pref.Color), nullIfEmpty(pref.AgeGroup), nullIfEmpty(pref.Gender),
		)
		if err != nil {
			return errors.New("failed to update user")
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.New("failed to update user")
	}
	return nil
}

func (m *MySQLUserAdapter) GetUserImage(uid string) (imageAddress *string, err error) {
	var image sql.NullString
	err = m.db.QueryRow("SELECT user_imageAddress FROM Users WHERE user_id = ?", uid).Scan(&image)
	if err != nil {
		return nil, errors.New("failed to get user image")
	}

	if !image.Valid {
		return nil, nil
	}
	return &image.String, nil
}

func (m *MySQLUserAdapter) UpdateUserImage(uid string, imageAddress string) (err error) {
	result, err := m.db.Exec("UPDATE Users SET user_imageAddress = ? WHERE user_id = ?", imageAddress, uid)
	if err != nil {
		return errors.New("failed to update user image")
	}

	rowsAffected, err := result.RowsAffected()
	if rowsAffected == 0 || err != nil {
		return errors.New("user not found")
	}

	return nil
}

func (m *MySQLUserAdapter) GetUserInfo(uid string) (userInfo *domain.UserInfo, err error) {
	var user domain.UserInfo
	
	err = m.db.QueryRow(
		"SELECT user_firstname, user_lastname, user_phoneNumber, user_address, user_addressLat, user_addressLong, user_imageAddress FROM Users WHERE user_id = ?",
		uid,
	).Scan(&user.Firstname, &user.Lastname, &user.Phone, &user.Address, &user.AddressLat, &user.AddressLong, &user.CoverImage)
	if err != nil {
		return nil, errors.New("failed to get user info")
	}

	rows, err := m.db.Query(
		`SELECT pref_petType, pref_breed, pref_color, pref_ageGroup, pref_gender FROM Users_Preferences WHERE pref_userId = ? ORDER BY pref_id`,
		uid,
	)
	if err != nil {
		return nil, errors.New("failed to get user info")
	}
	defer rows.Close()

	// Never nil, so the JSON is [] rather than null for a user with no preferences
	user.Preferences = []domain.UserPreference{}
	for rows.Next() {
		var petType, breed, color, ageGroup, gender sql.NullString
		if err := rows.Scan(&petType, &breed, &color, &ageGroup, &gender); err != nil {
			return nil, errors.New("failed to get user info")
		}
		// The old one-dog-one-cat code always wrote a row, even when nothing was picked;
		// skip those so they don't show up as blank Dog #1 / Cat #1 cards
		if !petType.Valid || (breed.String == "" && color.String == "" && ageGroup.String == "" && gender.String == "") {
			continue
		}
		user.Preferences = append(user.Preferences, domain.UserPreference{
			PetType:  petType.String,
			Breed:    breed.String,
			Color:    color.String,
			AgeGroup: ageGroup.String,
			Gender:   gender.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, errors.New("failed to get user info")
	}

	return &user, nil
}

func (m *MySQLUserAdapter) GetNewUserStatus(uid string) (userStatus bool, err error) {
	var status bool
	err = m.db.QueryRow(
		"SELECT user_newUser FROM Users WHERE user_id = ?",
		uid,
	).Scan(&status)
	if err != nil {
		return false, errors.New("failed to get user status")
	}
	return status, nil
}

func (m *MySQLUserAdapter) UpdateNewUserStatus(uid string) (userStatus bool, err error) {
	result, err := m.db.Exec("UPDATE Users SET user_newUser = ? WHERE user_id = ?", false, uid)
	if err != nil {
		return false, errors.New("failed to update user status")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, errors.New("failed to determine rows affected")
	}

	return rowsAffected > 0, nil
}