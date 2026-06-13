package adapter

import (
	"database/sql"
	"encoding/json"

	"errors"
	"strings"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

type MySQLPetAdapter struct {
	mysql_db *sql.DB
}

func NewMySQLPetAdapter(mysql_db *sql.DB) *MySQLPetAdapter {
	return &MySQLPetAdapter{mysql_db: mysql_db}
}

func (m *MySQLPetAdapter) CreatePet(uid string, petName string, imageAddress json.RawMessage, ageGroup string, gender string, petType string, breed string, color string, personality json.RawMessage, specialCare string, sterilized bool, vaccination bool, address string, addressLat float64, addressLong float64, status bool, note string) (pid int, err error) {
	result, err := m.mysql_db.Exec(`INSERT INTO Pets (pet_ownerId, pet_name, pet_imageAddress, pet_ageGroup, pet_gender, pet_type, pet_breed, pet_color, pet_personality, pet_specialCare, pet_sterilized, pet_vaccination, pet_address, pet_addressLat, pet_addressLong, pet_status, pet_note) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, uid, petName, imageAddress, ageGroup, gender, petType, breed, color, personality, specialCare, sterilized, vaccination, address, addressLat, addressLong, status, note)
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

	hasLocation := userLat != 0 || userLong != 0

	var query string
	var args []any

	if hasLocation {
		query = `
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
		args = []any{userLat, userLong, userLat}
	} else {
		query = `
		SELECT pet_id, pet_name, pet_imageAddress, pet_ageGroup,
		       pet_gender, pet_type, pet_breed, pet_color,
		       pet_address, pet_addressLat, pet_addressLong,
		       0 AS distance
		FROM Pets
		`
		args = []any{}
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

	if hasLocation {
		query += " ORDER BY distance ASC, pet_createdAt DESC"
	} else {
		query += " ORDER BY pet_createdAt DESC"
	}

	query += " LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	rows, err := m.mysql_db.Query(query, args...)
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

	err := m.mysql_db.QueryRow(`
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

func (m *MySQLPetAdapter) PostPetAdopt(uid string, pid int, q1_1 bool, q1_2 bool, q1_3 string, q2_1 string, q2_2 bool, q2_3 bool, q3_1 int8, q3_2 bool, q3_3 string, q4_1 int8, q5_1 int8, q6_1 int8, q6_2 int8, note string) (rid int, err error) {
	var petId int
	err = m.mysql_db.QueryRow(`SELECT pet_id FROM Pets WHERE pet_id = ? AND pet_status = "AVAILABLE"`, pid).Scan(&petId)
	if err != nil {
		return -1, errors.New("fail to adopt pet")
	}
	result, err := m.mysql_db.Exec(`
		INSERT INTO Pets_Rehoming (rehome_petId, rehome_adoptorId, rehome_status, rehome_contact, rehome_Q1_1, rehome_Q1_2, rehome_Q1_3, rehome_Q2_1, rehome_Q2_2, rehome_Q2_3, rehome_Q3_1, rehome_Q3_2, rehome_Q3_3, rehome_Q4_1, rehome_Q5_1, rehome_Q6_1, rehome_Q6_2, rehome_note)
		VALUES (?, ?, 'PENDING', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		WHERE pet_id = ? AND pet_status = 'AVAILABLE'
	`, pid, uid, "", q1_1, q1_2, q1_3, q2_1, q2_2, q2_3, q3_1, q3_2, q3_3, q4_1, q5_1, q6_1, q6_2, note)

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
	tx, err := m.mysql_db.Begin()
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

func (m *MySQLPetAdapter) GetPetsSuggestion() (petData []domain.PetsInfo, err error) {
	resultsCat, err := m.mysql_db.Query(`SELECT pet_id, pet_name, pet_imageAddress, pet_ageGroup, pet_gender, pet_type, pet_breed, pet_color, pet_address, pet_addressLat, pet_addressLong FROM Pets WHERE pet_status = 'AVAILABLE' AND pet_type = 'CAT' ORDER BY pet_createdAt DESC LIMIT 4`)
	if err != nil {
		return nil, err
	}
	resultsDog, err := m.mysql_db.Query(`SELECT pet_id, pet_name, pet_imageAddress, pet_ageGroup, pet_gender, pet_type, pet_breed, pet_color, pet_address, pet_addressLat, pet_addressLong FROM Pets WHERE pet_status = 'AVAILABLE' AND pet_type = 'DOG' ORDER BY pet_createdAt DESC LIMIT 4`)
	if err != nil {
		return nil, err
	}
	
	defer resultsCat.Close()
	defer resultsDog.Close()

	pets := make([]domain.PetsInfo, 0)
	for resultsCat.Next() {
		var pet domain.PetsInfo
		err := resultsCat.Scan(
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
		)
		if err != nil {
			return nil, err
		}
		pets = append(pets, pet)
	}

	for resultsDog.Next() {
		var pet domain.PetsInfo
		err := resultsDog.Scan(
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
		)
		if err != nil {
			return nil, err
		}
		pets = append(pets, pet)
	}

	if err := resultsCat.Err(); err != nil {
		return nil, err
	}

	if err := resultsDog.Err(); err != nil {
		return nil, err
	}

	return pets, nil
}