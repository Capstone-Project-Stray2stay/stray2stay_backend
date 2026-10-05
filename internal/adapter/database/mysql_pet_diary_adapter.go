package adapter

// Pet diary persistence.
//
// Access follows the accepted rehoming: the adopter writes, the original owner
// (the "finder" in the UI) reads. Both sides are matched with the same
// SUBSTRING_INDEX(..., ':', 1) comparison the rest of this package uses, since
// a stored id may carry a provider suffix the JWT's uid does not.

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

// firstImage pulls the cover image out of a pet's JSON image array. Pets with
// no images yield "" rather than an error — a missing photo is not a failure.
func firstImage(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	var images []string
	if err := json.Unmarshal(raw, &images); err != nil {
		return "", errors.New("fail to parse pet image address")
	}
	if len(images) == 0 {
		return "", nil
	}
	return images[0], nil
}

func (m *MySQLPetAdapter) GetMyDiaryPets(uid string) (pets []domain.DiaryPet, err error) {
	// One row per pet the caller is attached to through an accepted rehoming,
	// labelled from whichever side matched. The counterpart columns are chosen
	// by the same CASE, so a single query serves both roles.
	const isAdopter = `SUBSTRING_INDEX(pr.rehome_adoptorId, ':', 1) = SUBSTRING_INDEX(?, ':', 1)`

	rows, err := m.mysql_db.Query(`
		SELECT p.pet_id, p.pet_name, p.pet_imageAddress, p.pet_ageGroup,
		       p.pet_gender, p.pet_breed, p.pet_color,
		       CASE WHEN `+isAdopter+` THEN 'ADOPTER' ELSE 'FINDER' END,
		       DATE_FORMAT(pr.rehome_createAt, '%Y-%m-%d'),
		       CASE WHEN `+isAdopter+` THEN owner.user_firstname ELSE adoptor.user_firstname END,
		       CASE WHEN `+isAdopter+` THEN owner.user_lastname ELSE adoptor.user_lastname END,
		       CASE WHEN `+isAdopter+` THEN owner.user_phoneNumber ELSE adoptor.user_phoneNumber END,
		       CASE WHEN `+isAdopter+` THEN owner.user_imageAddress ELSE adoptor.user_imageAddress END
		FROM Pets_Rehoming pr
		JOIN Pets p ON pr.rehome_petId = p.pet_id
		JOIN Users owner ON SUBSTRING_INDEX(p.pet_ownerId, ':', 1) = SUBSTRING_INDEX(owner.user_id, ':', 1)
		JOIN Users adoptor ON SUBSTRING_INDEX(pr.rehome_adoptorId, ':', 1) = SUBSTRING_INDEX(adoptor.user_id, ':', 1)
		WHERE pr.rehome_status = 'ACCEPT'
		  AND (`+isAdopter+`
		       OR SUBSTRING_INDEX(p.pet_ownerId, ':', 1) = SUBSTRING_INDEX(?, ':', 1))
		ORDER BY pr.rehome_createAt DESC
	`, uid, uid, uid, uid, uid, uid, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pets = make([]domain.DiaryPet, 0)

	for rows.Next() {
		var pet domain.DiaryPet
		var imageAddressRaw []byte
		var firstname, lastname, phone, image sql.NullString

		if err := rows.Scan(
			&pet.Pid, &pet.PetName, &imageAddressRaw, &pet.PetAgeGroup,
			&pet.PetGender, &pet.PetBreed, &pet.PetColor,
			&pet.Role, &pet.Since,
			&firstname, &lastname, &phone, &image,
		); err != nil {
			return nil, err
		}

		cover, imgErr := firstImage(imageAddressRaw)
		if imgErr != nil {
			return nil, imgErr
		}
		pet.PetImageAddress = []string{}
		if cover != "" {
			pet.PetImageAddress = []string{cover}
		}

		pet.CanWrite = pet.Role == domain.DiaryRoleAdopter
		pet.CounterpartName = strings.TrimSpace(firstname.String + " " + lastname.String)
		// The label under the counterpart's name is the role they played,
		// which is the opposite of the caller's.
		pet.CounterpartRole = "Adopter"
		if pet.Role == domain.DiaryRoleAdopter {
			pet.CounterpartRole = "Finder"
		}
		pet.CounterpartPhone = phone.String
		pet.CounterpartImage = image.String

		pets = append(pets, pet)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pets, nil
}

func (m *MySQLPetAdapter) GetDiaryAccess(pid int, uid string) (role string, since string, err error) {
	err = m.mysql_db.QueryRow(`
		SELECT CASE
		           WHEN SUBSTRING_INDEX(pr.rehome_adoptorId, ':', 1) = SUBSTRING_INDEX(?, ':', 1)
		               THEN 'ADOPTER' ELSE 'FINDER'
		       END,
		       DATE_FORMAT(pr.rehome_createAt, '%Y-%m-%d')
		FROM Pets_Rehoming pr
		JOIN Pets p ON pr.rehome_petId = p.pet_id
		WHERE pr.rehome_petId = ? AND pr.rehome_status = 'ACCEPT'
		  AND (SUBSTRING_INDEX(pr.rehome_adoptorId, ':', 1) = SUBSTRING_INDEX(?, ':', 1)
		       OR SUBSTRING_INDEX(p.pet_ownerId, ':', 1) = SUBSTRING_INDEX(?, ':', 1))
	`, uid, pid, uid, uid).Scan(&role, &since)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", errors.New("no diary access for this pet")
		}
		return "", "", err
	}
	return role, since, nil
}

func (m *MySQLPetAdapter) GetDiaryEntries(pid int, from string, to string) (entries []domain.DiaryEntry, err error) {
	rows, err := m.mysql_db.Query(`
		SELECT diary_id, diary_petId, DATE_FORMAT(diary_date, '%Y-%m-%d'),
		       diary_imageAddress, COALESCE(diary_caption, '')
		FROM Pets_Diary
		WHERE diary_petId = ? AND diary_date BETWEEN ? AND ?
		ORDER BY diary_date ASC
	`, pid, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries = make([]domain.DiaryEntry, 0)

	for rows.Next() {
		var entry domain.DiaryEntry
		if err := rows.Scan(&entry.Did, &entry.Pid, &entry.Date, &entry.ImageAddress, &entry.Caption); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func (m *MySQLPetAdapter) GetDiaryEntry(pid int, date string) (entry domain.DiaryEntry, err error) {
	err = m.mysql_db.QueryRow(`
		SELECT diary_id, diary_petId, DATE_FORMAT(diary_date, '%Y-%m-%d'),
		       diary_imageAddress, COALESCE(diary_caption, '')
		FROM Pets_Diary
		WHERE diary_petId = ? AND diary_date = ?
	`, pid, date).Scan(&entry.Did, &entry.Pid, &entry.Date, &entry.ImageAddress, &entry.Caption)
	if err != nil {
		return domain.DiaryEntry{}, err
	}
	return entry, nil
}

func (m *MySQLPetAdapter) UpsertDiaryEntry(pid int, uid string, date string, imageAddress string, caption string) (entry domain.DiaryEntry, replacedImage string, err error) {
	tx, err := m.mysql_db.Begin()
	if err != nil {
		return domain.DiaryEntry{}, "", errors.New("fail to save diary entry")
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Lock the day's row first. The unique key would reject a concurrent
	// insert anyway, but reading the current image under that lock is what
	// makes "keep the existing photo" and the cleanup of a replaced one safe.
	var diaryID int
	var existingImage string
	scanErr := tx.QueryRow(`
		SELECT diary_id, diary_imageAddress FROM Pets_Diary
		WHERE diary_petId = ? AND diary_date = ?
		FOR UPDATE
	`, pid, date).Scan(&diaryID, &existingImage)
	if scanErr != nil && scanErr != sql.ErrNoRows {
		err = scanErr
		return domain.DiaryEntry{}, "", errors.New("fail to save diary entry")
	}
	hasExisting := scanErr == nil

	if !hasExisting && imageAddress == "" {
		err = errors.New("a photo is required for a new diary entry")
		return domain.DiaryEntry{}, "", err
	}

	finalImage := imageAddress
	if finalImage == "" {
		finalImage = existingImage
	} else if hasExisting && existingImage != "" && existingImage != finalImage {
		replacedImage = existingImage
	}

	if hasExisting {
		if _, execErr := tx.Exec(`
			UPDATE Pets_Diary
			SET diary_imageAddress = ?, diary_caption = ?, diary_authorId = ?
			WHERE diary_id = ?
		`, finalImage, caption, uid, diaryID); execErr != nil {
			err = execErr
			return domain.DiaryEntry{}, "", errors.New("fail to save diary entry")
		}
	} else {
		result, execErr := tx.Exec(`
			INSERT INTO Pets_Diary
			(diary_petId, diary_authorId, diary_date, diary_imageAddress, diary_caption)
			VALUES (?, ?, ?, ?, ?)
		`, pid, uid, date, finalImage, caption)
		if execErr != nil {
			err = execErr
			return domain.DiaryEntry{}, "", errors.New("fail to save diary entry")
		}
		id, idErr := result.LastInsertId()
		if idErr != nil {
			err = idErr
			return domain.DiaryEntry{}, "", errors.New("fail to save diary entry")
		}
		diaryID = int(id)
	}

	if err = tx.Commit(); err != nil {
		return domain.DiaryEntry{}, "", errors.New("fail to save diary entry")
	}

	return domain.DiaryEntry{
		Did:          diaryID,
		Pid:          pid,
		Date:         date,
		ImageAddress: finalImage,
		Caption:      caption,
	}, replacedImage, nil
}

func (m *MySQLPetAdapter) DeleteDiaryEntry(pid int, date string) (removedImage string, err error) {
	tx, err := m.mysql_db.Begin()
	if err != nil {
		return "", errors.New("fail to delete diary entry")
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	scanErr := tx.QueryRow(`
		SELECT diary_imageAddress FROM Pets_Diary
		WHERE diary_petId = ? AND diary_date = ?
		FOR UPDATE
	`, pid, date).Scan(&removedImage)
	if scanErr != nil {
		err = scanErr
		if scanErr == sql.ErrNoRows {
			return "", errors.New("diary entry not found")
		}
		return "", errors.New("fail to delete diary entry")
	}

	if _, execErr := tx.Exec(`
		DELETE FROM Pets_Diary WHERE diary_petId = ? AND diary_date = ?
	`, pid, date); execErr != nil {
		err = execErr
		return "", errors.New("fail to delete diary entry")
	}

	if err = tx.Commit(); err != nil {
		return "", errors.New("fail to delete diary entry")
	}

	return removedImage, nil
}
