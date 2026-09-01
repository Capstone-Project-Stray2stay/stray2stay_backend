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

// UpdatePet overwrites a pet's editable fields, but only when uid is its
// owner. imageAddress is the *final* image list (existing URLs the caller
// kept, plus any newly uploaded ones already merged in by the service) —
// whatever was on the pet before that's no longer in it comes back as
// removedImages so the caller can clean those up in Cloudinary.
func (m *MySQLPetAdapter) UpdatePet(
	uid string,
	pid int,
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
	note string,
) (removedImages []string, err error) {
	vaccineTypesStr := strings.Join(vaccination, ",")

	tx, err := m.mysql_db.Begin()
	if err != nil {
		return nil, errors.New("fail to update pet")
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var oldImageRaw []byte
	err = tx.QueryRow(`
		SELECT pet_imageAddress FROM Pets
		WHERE pet_id = ? AND SUBSTRING_INDEX(pet_ownerId, ':', 1) = SUBSTRING_INDEX(?, ':', 1)
		FOR UPDATE
	`, pid, uid).Scan(&oldImageRaw)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("pet not found or not owned by user")
		}
		return nil, errors.New("fail to update pet")
	}

	var oldImages []string
	if len(oldImageRaw) > 0 {
		if unmarshalErr := json.Unmarshal(oldImageRaw, &oldImages); unmarshalErr != nil {
			err = unmarshalErr
			return nil, errors.New("fail to parse pet image address")
		}
	}

	var newImages []string
	if unmarshalErr := json.Unmarshal(imageAddress, &newImages); unmarshalErr != nil {
		err = unmarshalErr
		return nil, errors.New("fail to parse pet image address")
	}
	kept := make(map[string]bool, len(newImages))
	for _, url := range newImages {
		kept[url] = true
	}
	for _, url := range oldImages {
		if !kept[url] {
			removedImages = append(removedImages, url)
		}
	}

	_, execErr := tx.Exec(`
		UPDATE Pets SET
			pet_name = ?, pet_imageAddress = ?, pet_ageGroup = ?, pet_gender = ?,
			pet_type = ?, pet_breed = ?, pet_color = ?, pet_personality = ?,
			pet_specialCare = ?, pet_sterilized = ?, pet_vaccination = ?,
			pet_address = ?, pet_addressLat = ?, pet_addressLong = ?, pet_note = ?
		WHERE pet_id = ? AND SUBSTRING_INDEX(pet_ownerId, ':', 1) = SUBSTRING_INDEX(?, ':', 1)
	`, petName, imageAddress, ageGroup, gender, petType, breed, color, personality,
		specialCare, sterilized, vaccineTypesStr, address, addressLat, addressLong, note,
		pid, uid)
	if execErr != nil {
		err = execErr
		return nil, errors.New("fail to update pet")
	}

	if err = tx.Commit(); err != nil {
		return nil, errors.New("fail to update pet")
	}

	return removedImages, nil
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
		SELECT pet_id, pet_ownerId, pet_name, pet_imageAddress, pet_ageGroup,
		       pet_gender, pet_type, pet_breed, pet_color,
		       pet_sterilized, pet_vaccination, pet_address, pet_addressLat,
		       pet_addressLong, pet_status, pet_note, pet_personality, pet_specialCare
		FROM Pets
		WHERE pet_id = ?
	`, pid).Scan(
		&pet.Pid,
		&pet.PetOwnerID,
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

	var existingRid int
	err = tx.QueryRow(`
		SELECT rehome_id FROM Pets_Rehoming
		WHERE rehome_petId = ? AND SUBSTRING_INDEX(rehome_adoptorId, ':', 1) = SUBSTRING_INDEX(?, ':', 1)
		      AND rehome_status = 'PENDING'
		FOR UPDATE
	`, pid, uid).Scan(&existingRid)
	if err == nil {
		err = errors.New("you already have a pending request for this pet")
		return -1, err
	}
	if err != sql.ErrNoRows {
		return -1, errors.New("fail to adopt pet")
	}
	err = nil

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

func (m *MySQLPetAdapter) UpdatePetAdopter(rid int, uid string) (err error) {
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
		      AND SUBSTRING_INDEX(p.pet_ownerId, ':', 1) = SUBSTRING_INDEX(?, ':', 1)
	`, rid, uid)
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

func (m *MySQLPetAdapter) GetScreeningAnswer(rehomeID int, uid string) (domain.ScreeningAnswer, error) {
	var a domain.ScreeningAnswer

	err := m.mysql_db.QueryRow(`
		SELECT pr.rehome_Q1_1, pr.rehome_Q1_2, pr.rehome_Q1_3,
		       pr.rehome_Q2_1, pr.rehome_Q2_2, pr.rehome_Q2_3,
		       pr.rehome_Q3_1, pr.rehome_Q3_2, pr.rehome_Q3_3,
		       pr.rehome_Q4_1, pr.rehome_Q5_1, pr.rehome_Q6_1, pr.rehome_Q6_2,
		       pr.rehome_note
		FROM Pets_Rehoming pr
		JOIN Pets p ON pr.rehome_petId = p.pet_id
		WHERE pr.rehome_id = ? AND SUBSTRING_INDEX(p.pet_ownerId, ':', 1) = SUBSTRING_INDEX(?, ':', 1)
	`, rehomeID, uid).Scan(
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

// GetMyAdoptionStatus reports uid's own most recent adoption request on a
// pet — "" if they've never requested it — so the pet-info handler can tell
// the viewer "Adopt the Pet" from "Request Pending" without exposing anyone
// else's requests.
func (m *MySQLPetAdapter) GetMyAdoptionStatus(pid int, uid string) (status string, err error) {
	err = m.mysql_db.QueryRow(`
		SELECT rehome_status FROM Pets_Rehoming
		WHERE rehome_petId = ? AND SUBSTRING_INDEX(rehome_adoptorId, ':', 1) = SUBSTRING_INDEX(?, ':', 1)
		ORDER BY rehome_createAt DESC
		LIMIT 1
	`, pid, uid).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", errors.New("fail to get adoption status")
	}
	return status, nil
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
		       u.user_lastname, u.user_phoneNumber, u.user_address, u.user_imageAddress,
		       pr.rehome_id, pr.rehome_status
		FROM Pets_Rehoming pr
		JOIN Pets p ON pr.rehome_petId = p.pet_id
		JOIN Users u ON pr.rehome_adoptorId = u.user_id
		WHERE SUBSTRING_INDEX(p.pet_ownerId, ':', 1) = SUBSTRING_INDEX(?, ':', 1)
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
			&adoptor.ImageAddress, &adoptor.Rid, &adoptor.RehomeStatus); err != nil {
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

// GetMyAdoptionRequests lists every request uid has made as an adoptor,
// newest first — backs the Profile page's "My Adoptions" list.
func (m *MySQLPetAdapter) GetMyAdoptionRequests(uid string) (requests []domain.MyAdoptionRequest, err error) {
	rows, err := m.mysql_db.Query(`
		SELECT pr.rehome_id, p.pet_id, p.pet_name, p.pet_imageAddress,
		       pr.rehome_status, u.user_phoneNumber
		FROM Pets_Rehoming pr
		JOIN Pets p ON pr.rehome_petId = p.pet_id
		JOIN Users u ON p.pet_ownerId = u.user_id
		WHERE SUBSTRING_INDEX(pr.rehome_adoptorId, ':', 1) = SUBSTRING_INDEX(?, ':', 1)
		ORDER BY pr.rehome_createAt DESC
	`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests = make([]domain.MyAdoptionRequest, 0)

	for rows.Next() {
		var req domain.MyAdoptionRequest
		var imageAddressRaw []byte

		if err := rows.Scan(
			&req.Rid, &req.Pid, &req.PetName, &imageAddressRaw,
			&req.RehomeStatus, &req.OwnerPhone,
		); err != nil {
			return nil, err
		}

		if len(imageAddressRaw) > 0 {
			if err := json.Unmarshal(imageAddressRaw, &req.PetImageAddress); err != nil {
				return nil, errors.New("fail to parse pet image address")
			}
		}

		requests = append(requests, req)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return requests, nil
}

// CancelAdoptionRequest withdraws uid's own pending request, deleting the row
// outright. Already-decided requests (ACCEPT/DENIED) and requests belonging
// to someone else are left untouched — zero rows affected surfaces as an
// error rather than a silent no-op.
func (m *MySQLPetAdapter) CancelAdoptionRequest(uid string, rid int) (err error) {
	result, execErr := m.mysql_db.Exec(`
		DELETE FROM Pets_Rehoming
		WHERE rehome_id = ? AND rehome_status = 'PENDING'
		      AND SUBSTRING_INDEX(rehome_adoptorId, ':', 1) = SUBSTRING_INDEX(?, ':', 1)
	`, rid, uid)
	if execErr != nil {
		return errors.New("fail to cancel adoption request")
	}

	rowsAffected, raErr := result.RowsAffected()
	if raErr != nil {
		return raErr
	}
	if rowsAffected == 0 {
		return errors.New("request not found or not cancellable")
	}

	return nil
}

// DeletePet removes a pet row, but only when uid is its owner, and hands back
// the image URLs that were on it so the caller can clean those up in
// Cloudinary (this adapter has no uploader dependency to do that itself).
func (m *MySQLPetAdapter) DeletePet(uid string, pid int) (imageAddresses []string, err error) {
	tx, err := m.mysql_db.Begin()
	if err != nil {
		return nil, errors.New("fail to delete pet")
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// OAuth-registered accounts store user_id (and thus pet_ownerId) as
	// "<uuid>:OAUTH", but the JWT's uid claim carries the bare uuid — compare
	// on the part before the colon so OAuth owners can manage their own pets.
	var imageAddressRaw []byte
	err = tx.QueryRow(`
		SELECT pet_imageAddress FROM Pets
		WHERE pet_id = ? AND SUBSTRING_INDEX(pet_ownerId, ':', 1) = SUBSTRING_INDEX(?, ':', 1)
		FOR UPDATE
	`, pid, uid).Scan(&imageAddressRaw)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("pet not found or not owned by user")
		}
		return nil, errors.New("fail to delete pet")
	}

	result, execErr := tx.Exec(`
		DELETE FROM Pets
		WHERE pet_id = ? AND SUBSTRING_INDEX(pet_ownerId, ':', 1) = SUBSTRING_INDEX(?, ':', 1)
	`, pid, uid)
	if execErr != nil {
		err = execErr
		return nil, errors.New("fail to delete pet")
	}

	rowsAffected, raErr := result.RowsAffected()
	if raErr != nil {
		err = raErr
		return nil, errors.New("fail to delete pet")
	}
	if rowsAffected == 0 {
		err = errors.New("pet not found or not owned by user")
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, errors.New("fail to delete pet")
	}

	if len(imageAddressRaw) > 0 {
		if unmarshalErr := json.Unmarshal(imageAddressRaw, &imageAddresses); unmarshalErr != nil {
			return nil, errors.New("fail to parse pet image address")
		}
	}

	return imageAddresses, nil
}

// GetPetsByOwner lists every pet uid has registered, regardless of status —
// this backs the Profile page's "My Rehoming" list, which is a management
// view of the owner's own listings, not a public browse.
func (m *MySQLPetAdapter) GetPetsByOwner(uid string) (petData []domain.PetsInfo, err error) {
	rows, err := m.mysql_db.Query(`
		SELECT pet_id, pet_name, pet_imageAddress, pet_ageGroup, pet_gender,
		       pet_type, pet_breed, pet_color, pet_address, pet_addressLat, pet_addressLong
		FROM Pets
		WHERE SUBSTRING_INDEX(pet_ownerId, ':', 1) = SUBSTRING_INDEX(?, ':', 1)
		ORDER BY pet_createAt DESC
	`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pets := make([]domain.PetsInfo, 0)

	for rows.Next() {
		var pet domain.PetsInfo
		var imageAddressRaw []byte

		if err := rows.Scan(
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
		); err != nil {
			return nil, err
		}

		if len(imageAddressRaw) > 0 {
			if err := json.Unmarshal(imageAddressRaw, &pet.PetImageAddress); err != nil {
				return nil, errors.New("fail to parse pet image address")
			}
		}

		pets = append(pets, pet)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pets, nil
}
