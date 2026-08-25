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

func (m *MySQLPetAdapter) CreatePet(
	uid string,
	petName string,
	imageAddress json.RawMessage,
	ageGroup string,
	gender string,
	petType string,
	breed string,
	color string,
	personality json.RawMessage,
	specialCare json.RawMessage,
	sterilized bool,
	vaccination []string,
	address string,
	addressLat float64,
	addressLong float64,
	status string,
	note string,
) (pid int, err error) {
	vaccineTypesStr := strings.Join(vaccination, ",")
	result, err := m.mysql_db.Exec(`
		INSERT INTO Pets (
			pet_ownerId, pet_name, pet_imageAddress, pet_ageGroup, pet_gender,
			pet_type, pet_breed, pet_color, pet_personality, pet_specialCare,
			pet_sterilized, pet_vaccination, pet_address, pet_addressLat,
			pet_addressLong, pet_status, pet_note
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, uid, petName, imageAddress, ageGroup, gender, petType, breed, color,
		personality, specialCare, sterilized, vaccineTypesStr, address,
		addressLat, addressLong, status, note)
	if err != nil {
		return -1, errors.New("fail to create pet data")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}
	return int(id), nil
}

func (m *MySQLPetAdapter) GetPetsInfo(
	page int,
	pageSize int,
	petAgeGroup string,
	petGender string,
	petType string,
	petBreed string,
	petColor string,
	petLocation string,
	userLat float64,
	userLong float64,
) (petData []domain.PetsInfo, totalCount int, err error) {

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	hasLocation := userLat != 0 || userLong != 0

	conditions := []string{"pet_status = 'AVALIABLE'"}
	var filterArgs []any

	if petAgeGroup != "" {
		conditions = append(conditions, "pet_ageGroup = ?")
		filterArgs = append(filterArgs, petAgeGroup)
	}
	if petGender != "" {
		conditions = append(conditions, "pet_gender = ?")
		filterArgs = append(filterArgs, petGender)
	}
	if petType != "" {
		conditions = append(conditions, "pet_type = ?")
		filterArgs = append(filterArgs, petType)
	}
	if petBreed != "" {
		conditions = append(conditions, "pet_breed = ?")
		filterArgs = append(filterArgs, petBreed)
	}
	if petColor != "" {
		conditions = append(conditions, "pet_color = ?")
		filterArgs = append(filterArgs, petColor)
	}
	if petLocation != "" {
		// pet_address is freeform text built as "street, subDistrict, district,
		// province" (see the frontend's joinAddress/resolveLocation) rather than
		// a location enum, so this is a substring match against the province
		// name rather than an exact-match column.
		conditions = append(conditions, "pet_address LIKE ?")
		filterArgs = append(filterArgs, "%"+petLocation+"%")
	}

	whereClause := " WHERE " + strings.Join(conditions, " AND ")

	err = m.mysql_db.QueryRow("SELECT COUNT(*) FROM Pets"+whereClause, filterArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}
	
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
		args = append([]any{userLat, userLong, userLat}, filterArgs...)
	} else {
		query = `
		SELECT pet_id, pet_name, pet_imageAddress, pet_ageGroup,
		       pet_gender, pet_type, pet_breed, pet_color,
		       pet_address, pet_addressLat, pet_addressLong,
		       0 AS distance
		FROM Pets
		`
		args = append([]any{}, filterArgs...)
	}

	query += whereClause

	if hasLocation {
		query += " ORDER BY distance ASC, pet_createAt DESC"
	} else {
		query += " ORDER BY pet_createAt DESC"
	}

	query += " LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	rows, err := m.mysql_db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	pets := make([]domain.PetsInfo, 0)

	for rows.Next() {
		var pet domain.PetsInfo
		var imageAddressRaw []byte
		var distance float64

		err := rows.Scan(
			&pet.Pid,
			&pet.PetName,
			&imageAddressRaw,
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
			return nil, 0, err
		}

		if len(imageAddressRaw) > 0 {
			if err := json.Unmarshal(imageAddressRaw, &pet.PetImageAddress); err != nil {
				return nil, 0, errors.New("fail to parse pet image address")
			}
		}

		pets = append(pets, pet)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return pets, totalCount, nil
}

func (m *MySQLPetAdapter) GetPetsSuggestion() (petData []domain.PetsInfo, err error) {
	const q = `
		SELECT pet_id, pet_name, pet_imageAddress, pet_ageGroup, pet_gender,
		       pet_type, pet_breed, pet_color, pet_address, pet_addressLat, pet_addressLong
		FROM Pets
		WHERE pet_status = 'AVALIABLE' AND pet_type = ?
		ORDER BY pet_createAt DESC
		LIMIT 8
	`

	pets := make([]domain.PetsInfo, 0)

	for _, petType := range []string{"CAT", "DOG"} {
		rows, err := m.mysql_db.Query(q, petType)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var pet domain.PetsInfo
			var imageAddressRaw []byte

			err := rows.Scan(
				&pet.Pid,
				&pet.PetName,
				&imageAddressRaw,
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
				rows.Close()
				return nil, err
			}

			if len(imageAddressRaw) > 0 {
				if err := json.Unmarshal(imageAddressRaw, &pet.PetImageAddress); err != nil {
					rows.Close()
					return nil, errors.New("fail to parse pet image address")
				}
			}

			pets = append(pets, pet)
		}

		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}

	return pets, nil
}

func (m *MySQLPetAdapter) GetPetInfo(pid int) (domain.PetInfo, error) {
	var pet domain.PetInfo
	var imageAddressRaw []byte
	var personalityRaw []byte
	var vacinationRaw []byte
	var specialCareRaw []byte

	err := m.mysql_db.QueryRow(`
		SELECT pet_id, pet_name, pet_imageAddress, pet_ageGroup,
		       pet_gender, pet_type, pet_breed, pet_color,
		       pet_sterilized, pet_vaccination, pet_address, pet_addressLat,
		       pet_addressLong, pet_status, pet_note, pet_personality, pet_specialCare
		FROM Pets
		WHERE pet_id = ?
	`, pid).Scan(
		&pet.Pid,
		&pet.PetName,
		&imageAddressRaw,
		&pet.PetAgeGroup,
		&pet.PetGender,
		&pet.PetType,
		&pet.PetBreed,
		&pet.PetColor,
		&pet.PetSterilized,
		&vacinationRaw,
		&pet.PetAddress,
		&pet.PetAddressLat,
		&pet.PetAddressLong,
		&pet.Status,
		&pet.Note,
		&personalityRaw,
		&specialCareRaw,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.PetInfo{}, errors.New("pet not found")
		}
		return domain.PetInfo{}, errors.New("fail to get pet info")
	}
	
	if len(imageAddressRaw) > 0 {
		if err := json.Unmarshal(imageAddressRaw, &pet.PetImageAddress); err != nil {
			return domain.PetInfo{}, errors.New("fail to parse pet image address")
		}
	}
	if len(personalityRaw) > 0 {
		if err := json.Unmarshal(personalityRaw, &pet.PetPersonality); err != nil {
			return domain.PetInfo{}, errors.New("fail to parse pet personality")
		}
	}
	// pet_vaccination is a MySQL SET column, not JSON — it comes back as a
	// plain comma-separated string (e.g. "DHPPi,Rabies"), matching how
	// CreatePet writes it via strings.Join(vaccination, ",").
	if raw := strings.TrimSpace(string(vacinationRaw)); raw != "" {
		pet.PetVaccination = strings.Split(raw, ",")
	}
	if len(specialCareRaw) > 0 {
		if err := json.Unmarshal(specialCareRaw, &pet.PetSpecialCare); err != nil {
			return domain.PetInfo{}, errors.New("fail to parse pet special care")
		}
	}

	return pet, nil
}

func (m *MySQLPetAdapter) PostPetAdopt(
	uid string,
	pid int,
	q1_1 bool, q1_2 bool, q1_3 string,
	q2_1 string, q2_2 bool, q2_3 bool,
	q3_1 int8, q3_2 bool, q3_3 string,
	q4_1 int8, q5_1 int8, q6_1 int8, q6_2 int8,
	note string,
) (rid int, err error) {
	tx, err := m.mysql_db.Begin()
	if err != nil {
		return -1, errors.New("fail to adopt pet")
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var petId int
	err = tx.QueryRow(`
		SELECT pet_id FROM Pets
		WHERE pet_id = ? AND pet_status = 'AVALIABLE'
		FOR UPDATE
	`, pid).Scan(&petId)
	if err != nil {
		if err == sql.ErrNoRows {
			return -1, errors.New("pet not available for adoption")
		}
		return -1, errors.New("fail to adopt pet")
	}

	result, execErr := tx.Exec(`
		INSERT INTO Pets_Rehoming (
			rehome_petId, rehome_adoptorId, rehome_status, rehome_contact,
			rehome_Q1_1, rehome_Q1_2, rehome_Q1_3,
			rehome_Q2_1, rehome_Q2_2, rehome_Q2_3,
			rehome_Q3_1, rehome_Q3_2, rehome_Q3_3,
			rehome_Q4_1, rehome_Q5_1, rehome_Q6_1, rehome_Q6_2,
			rehome_note
		) VALUES (?, ?, 'PENDING', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, pid, uid, "", q1_1, q1_2, q1_3, q2_1, q2_2, q2_3, q3_1, q3_2, q3_3, q4_1, q5_1, q6_1, q6_2, note)
	if execErr != nil {
		err = execErr
		return -1, errors.New("fail to adopt pet")
	}

	id, idErr := result.LastInsertId()
	if idErr != nil {
		err = idErr
		return -1, idErr
	}

	if err = tx.Commit(); err != nil {
		return -1, errors.New("fail to adopt pet")
	}

	return int(id), nil
}

func (m *MySQLPetAdapter) UpdatePetAdopter(rid int) (err error) {
	tx, err := m.mysql_db.Begin()
	if err != nil {
		return errors.New("fail to start transaction")
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	result, execErr := tx.Exec(`
		UPDATE Pets_Rehoming pr
		JOIN Pets p ON pr.rehome_petId = p.pet_id
		SET pr.rehome_status = 'ACCEPT', p.pet_status = 'ADOPTED'
		WHERE pr.rehome_id = ? AND pr.rehome_status = 'PENDING' AND p.pet_status = 'AVALIABLE'
	`, rid)
	if execErr != nil {
		err = execErr
		return errors.New("fail to accept adopter")
	}

	rowsAffected, raErr := result.RowsAffected()
	if raErr != nil {
		err = raErr
		return raErr
	}
	if rowsAffected == 0 {
		err = errors.New("adoption request not found or already processed")
		return err
	}

	_, execErr = tx.Exec(`
		UPDATE Pets_Rehoming
		SET rehome_status = 'DENIED'
		WHERE rehome_petId = (
			SELECT rehome_petId FROM Pets_Rehoming WHERE rehome_id = ?
		)
		AND rehome_id != ?
		AND rehome_status = 'PENDING'
	`, rid, rid)
	if execErr != nil {
		err = execErr
		return errors.New("fail to deny other adopters")
	}

	if err = tx.Commit(); err != nil {
		return errors.New("fail to update pet adopter")
	}
	return nil
}

func (m *MySQLPetAdapter) GetScreeningAnswer(rehomeID int) (domain.ScreeningAnswer, error) {
	var a domain.ScreeningAnswer

	err := m.mysql_db.QueryRow(`
		SELECT rehome_Q1_1, rehome_Q1_2, rehome_Q1_3,
		       rehome_Q2_1, rehome_Q2_2, rehome_Q2_3,
		       rehome_Q3_1, rehome_Q3_2, rehome_Q3_3,
		       rehome_Q4_1, rehome_Q5_1, rehome_Q6_1, rehome_Q6_2,
		       rehome_note
		FROM Pets_Rehoming
		WHERE rehome_id = ?
	`, rehomeID).Scan(
		&a.Q1_1, &a.Q1_2, &a.Q1_3,
		&a.Q2_1, &a.Q2_2, &a.Q2_3,
		&a.Q3_1, &a.Q3_2, &a.Q3_3,
		&a.Q4_1, &a.Q5_1, &a.Q6_1, &a.Q6_2,
		&a.Note,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.ScreeningAnswer{}, errors.New("screening answer not found")
		}
		return domain.ScreeningAnswer{}, errors.New("fail to get screening answer")
	}

	return a, nil
}

func (m *MySQLPetAdapter) GetAllAdoptors(uid string) (adoptors []domain.PetAdoptorsInfo, err error) {
	adoptorsMap := make(map[int][]domain.AdoptorInfo)
	petInfoMap := make(map[int]struct {
		Name  string
		Image string
	})
	var petOrder []int

	rows, err := m.mysql_db.Query(`
		SELECT p.pet_id, p.pet_name, p.pet_imageAddress, u.user_id, u.user_firstname,
		       u.user_lastname, u.user_phoneNumber, u.user_address, pr.rehome_id, pr.rehome_status
		FROM Pets_Rehoming pr
		JOIN Pets p ON pr.rehome_petId = p.pet_id
		JOIN Users u ON pr.rehome_adoptorId = u.user_id
		WHERE p.pet_ownerId = ?
	`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var adoptor domain.AdoptorInfo
		var petID int
		var petName, petImageAddress string

		if err := rows.Scan(&petID, &petName, &petImageAddress, &adoptor.UserID,
			&adoptor.Firstname, &adoptor.Lastname, &adoptor.PhoneNumber, &adoptor.Address,
			&adoptor.Rid, &adoptor.RehomeStatus); err != nil {
			return nil, err
		}

		if _, ok := petInfoMap[petID]; !ok {
			petInfoMap[petID] = struct {
				Name  string
				Image string
			}{petName, petImageAddress}
			petOrder = append(petOrder, petID)
		}

		adoptorsMap[petID] = append(adoptorsMap[petID], adoptor)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, petID := range petOrder {
		info := petInfoMap[petID]
		adoptors = append(adoptors, domain.PetAdoptorsInfo{
			Pid:             petID,
			PetName:         info.Name,
			PetImageAddress: info.Image,
			AdoptorsInfo:    adoptorsMap[petID],
		})
	}

	return adoptors, nil
}
