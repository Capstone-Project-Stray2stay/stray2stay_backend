package adapter

import (
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"os"
	"strconv"

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
		(user_id, user_email, user_firstname, user_lastname, user_authType)
		VALUES (?, ?, ?, ?, ?)
	`, userId, email, firstName, lastName, "PASS")
	if err != nil {
		return errors.New("failed to create user")
	}
	_, err = tx.Exec(`
		INSERT INTO User_Password (password_userId, password_pass)
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
) (string, error) {
	var userUID string
	err := m.db.QueryRow(`
		SELECT user_id
		FROM Users
		WHERE user_email = ? AND user_authType = ?
	`, email, provider).Scan(&userUID)
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
		(user_id, user_email, user_firstname, user_lastname, user_authType)
		VALUES (?, ?, ?, ?, ?)
	`, oAuthUID, email, firstName, lastName, "OAUTH")

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
		WHERE u.user_email = ? && u.authType = 'PASS'
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
	userId, err := uuid.Parse(uid)
	if err != nil {
		return errors.New("invalid user ID")
	}
	_, err = m.db.Exec("DELETE FROM Users WHERE user_id = ?", userId)
	if err != nil {
		return errors.New("failed to delete user")
	}
	return nil
}

func (m *MySQLUserAdapter) UpdateUserInfo(uid string, firstName string, lastName string, phoneNumber string, address string, addressLat float64, addressLong float64, dogBreed string, dogColor string, dogAgeGroup string, dogGender string, catBreed string, catColor string, catAgeGroup string, catGender string) (err error) {
	userId, err := uuid.Parse(uid)
	if err != nil {
		return errors.New("invalid user ID")
	}
	
	result, err := m.db.Exec(`UPDATE Users SET user_firstname = ?, user_lastname = ?, user_phoneNumber = ?, user_address = ?, user_addressLat = ?, user_addressLong = ? WHERE user_id = ?`, firstName, lastName, phoneNumber, address, addressLat, addressLong, userId)
	if err != nil {
		return errors.New("failed to update user")
	}

	rowsAffected, err := result.RowsAffected()
	if rowsAffected == 0 || err != nil {
		return errors.New("user not found")
	}

	_, err = m.db.Exec(`SELECT pref_id FROM User_Preferences WHERE pref_userId = ? AND pref_type = 'DOG'`, userId)
	if err != nil {
		return errors.New("failed to update user")
	}
	if err == sql.ErrNoRows {
		_, err = m.db.Exec(`INSERT INTO User_Preferences (pref_userId, pref_petType, pref_breed, pref_color, pref_ageGroup, pref_gender) VALUES (?, 'DOG', ?, ?, ?, ?)`, userId, dogBreed, dogColor, dogAgeGroup, dogGender)
		if err != nil {
			return errors.New("failed to update user")
		}
	}

	_, err = m.db.Exec(`SELECT pref_id FROM User_Preferences WHERE pref_userId = ? AND pref_type = 'CAT'`, userId)
	if err != nil {
		return errors.New("failed to update user")
	}
	if err == sql.ErrNoRows {
		_, err = m.db.Exec(`INSERT INTO User_Preferences (pref_userId, pref_petType, pref_breed, pref_color, pref_ageGroup, pref_gender) VALUES (?, 'CAT', ?, ?, ?, ?)`, userId, catBreed, catColor, catAgeGroup, catGender)
		if err != nil {
			return errors.New("failed to update user")
		}
	}

	return nil
}

func (m *MySQLUserAdapter) GetUserInfo(uid string) (userInfo *domain.UserInfo, err error) {
	var user domain.UserInfo

	userId, err := uuid.Parse(uid)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	err = m.db.QueryRow(
		"SELECT user_firstname, user_lastname, user_phoneNumber, user_address FROM Users WHERE user_id = ?",
		userId,
	).Scan(&user.Firstname, &user.Lastname, &user.Phone, &user.Address)
	if err != nil {
		return nil, errors.New("failed to get user info")
	}
	return &user, nil
}
