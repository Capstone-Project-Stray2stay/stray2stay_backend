package adapter

import (
	"database/sql"
	"encoding/json"

	"errors"
	"strings"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

type MySQLPetAdapter struct {
	db *sql.DB
}

func NewMySQLPetAdapter(db *sql.DB) *MySQLPetAdapter {
	return &MySQLPetAdapter{db: db}
}

func (m *MySQLPetAdapter) CreatePet(uid string, petName string, imageAddress json.RawMessage, ageGroup string, gender string, petType string, breed string, color string, healthCondition string, sterilized bool, vaccination bool, address string, addressLat float64, addressLong float64, status bool, note string) (pid int, err error) {
	result, err := m.db.Exec(`INSERT INTO Pets (pet_ownerId, pet_name, pet_imageAddress, pet_ageGroup, pet_gender, pet_type, pet_breed, pet_color, pet_healthCondition, pet_sterilized, pet_vaccination, pet_address, pet_addressLat, pet_addressLong, pet_status, pet_note) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, uid, petName, imageAddress, ageGroup, gender, petType, breed, color, healthCondition, sterilized, vaccination, address, addressLat, addressLong, status, note)
	if err != nil {
		return -1, errors.New("fail to create pet Data")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}
	return int(id), nil
}

func (m *MySQLPetAdapter) GetPetsInfo(page int, pageSize int, petAgeGroup string, petGender string, petType string, petBreed string, petColor string, userLat float64, userLong float64) ([]domain.PetsInfo, error) {

	if page <= 0 && pageSize <= 0 {
		page = 1
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	query := `
	SELECT pet_id, pet_name, pet_imageAddress, pet_ageGroup,
	       pet_gender, pet_type, pet_breed, pet_color,
	       pet_address, pet_addressLat, pet_addressLong,
	       (
	         6371 * acos(
	           cos(radians(?)) *
	           cos(radians(pet_addressLat)) *
	           cos(radians(pet_addressLong) - radians(?)) +
	           sin(radians(?)) *
	           sin(radians(pet_addressLat))
	         )
	       ) AS distance
	FROM Pets
	`

	args := []any{
		userLat,
		userLong,
		userLat,
	}

	conditions := []string{"pet_status = 'AVAILABLE'"}

	if petAgeGroup != "" {
		conditions = append(conditions, "pet_ageGroup = ?")
		args = append(args, petAgeGroup)
	}

	if petGender != "" {
		conditions = append(conditions, "pet_gender = ?")
		args = append(args, petGender)
	}

	if petType != "" {
		conditions = append(conditions, "pet_type = ?")
		args = append(args, petType)
	}

	if petBreed != "" {
		conditions = append(conditions, "pet_breed = ?")
		args = append(args, petBreed)
	}

	if petColor != "" {
		conditions = append(conditions, "pet_color = ?")
		args = append(args, petColor)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += `
	ORDER BY distance ASC, pet_createdAt DESC
	LIMIT ? OFFSET ?
	`

	args = append(args, pageSize, offset)

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pets := make([]domain.PetsInfo, 0)

	for rows.Next() {
		var pet domain.PetsInfo
		var distance float64

		err := rows.Scan(
			&pet.Pid,
			&pet.PetName,
			&pet.PetImageAddress,
			&pet.PetAgeGroup,
			&pet.PetGender,
			&pet.PetType,
			&pet.PetBreed,
			&pet.PetColor,
			&pet.PetAddress,
			&pet.PetAddressLat,
			&pet.PetAddressLong,
			&distance,
		)
		if err != nil {
			return nil, err
		}

		pets = append(pets, pet)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pets, nil
}

func (m *MySQLPetAdapter) GetPetInfo(pid int) (domain.PetInfo, error) {
	var pet domain.PetInfo

	err := m.db.QueryRow(`
		SELECT pet_id, pet_name, pet_detail, pet_imageAddress, pet_ageGroup,
		       pet_gender, pet_type, pet_breed, pet_color, pet_healthCondition,
		       pet_sterilized, pet_vaccination, pet_address, pet_addressLat,
		       pet_addressLong, pet_status, pet_note
		FROM Pets 
		WHERE pet_id = ?
	`, pid).Scan(
		&pet.Pid,
		&pet.PetName,
		&pet.PetDetail,
		&pet.PetImageAddress,
		&pet.PetAgeGroup,
		&pet.PetGender,
		&pet.PetType,
		&pet.PetBreed,
		&pet.PetColor,
		&pet.PetHealthCondition,
		&pet.PetSterilized,
		&pet.PetVaccination,
		&pet.PetAddress,
		&pet.PetAddressLat,
		&pet.PetAddressLong,
		&pet.Status,
		&pet.Note,
	)

	if err != nil {
		return domain.PetInfo{}, errors.New("fail to get pet info")
	}

	return pet, nil
}

func (m *MySQLPetAdapter) PostPetAdopt(uid string, pid int, contact string) (rid int, err error) {
	var petId int
	err = m.db.QueryRow(`SELECT pet_id FROM Pets WHERE pet_id = ? AND pet_status = ?`, pid, "AVAILABLE").Scan(&petId)
	if err != nil {
		return -1, errors.New("fail to adopt pet")
	}
	result, err := m.db.Exec(`
		INSERT INTO Pets_Rehoming (rehome_petId, rehome_adoptorId, rehome_status, rehome_contact)
		VALUES (?, ?, 'PENDING', ?)
		WHERE pet_id = ? AND pet_status = 'AVAILABLE'
	`, pid, uid, "PENDING", contact)

	if err != nil {
		return -1, errors.New("fail to adopt pet")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}
	return int(id), nil
}

func (m *MySQLPetAdapter) UpdatePetAdopter(rid int) (err error) {
	tx, err := m.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	result, err := tx.Exec(`
		UPDATE Pets_Rehoming pr
		JOIN Pets p ON pr.rehome_petId = p.pet_id
		SET pr.rehome_status = 'ACCEPT', p.pet_status = 'ADOPTED'
		WHERE pr.rehome_id = ? AND pr.rehome_status = 'PENDING' AND p.pet_status = 'AVAILABLE'
	`, rid)
	if err != nil {
		return errors.New("fail to select pet adopter")
	}
	rowsAffected, err := result.RowsAffected()
	if rowsAffected == 0 || err != nil {
		return errors.New("adoption request not found or already processed")
	}
	result, err = tx.Exec(`
		UPDATE Pets_Rehoming
		SET rehome_status = 'DENIED'
		WHERE rehome_id NOT IN (?)`, rid)

	if err != nil {
		return errors.New("fail to update pet adopter")
	}
	err = tx.Commit()
	if err != nil {
		return errors.New("fail to update pet adopter")
	}
	return nil
}
