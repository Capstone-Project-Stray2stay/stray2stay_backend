package service

// Pet diary rules. The repository decides who is attached to a pet; everything
// about *what they may do* — write vs read, and which days are in range —
// lives here.

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"mime/multipart"
	"time"

	"github.com/S-nudhana/stray2stay/internal/core/domain"
)

const diaryDateLayout = "2006-01-02"

// parseDiaryDay rejects anything that is not a plain calendar day, so a
// malformed value can never reach the DATE column or a BETWEEN clause.
func parseDiaryDay(value string) (time.Time, error) {
	day, err := time.ParseInLocation(diaryDateLayout, value, time.Local)
	if err != nil {
		return time.Time{}, errors.New("date must be formatted YYYY-MM-DD")
	}
	return day, nil
}

func (s *PetServiceImpl) MyDiaryPets(ctx context.Context, uid string) (pets []domain.DiaryPet, err error) {
	return s.mysqlRepo.GetMyDiaryPets(uid)
}

func (s *PetServiceImpl) DiaryEntries(ctx context.Context, uid string, pid int, from string, to string) (entries []domain.DiaryEntry, role string, since string, err error) {
	role, since, err = s.mysqlRepo.GetDiaryAccess(pid, uid)
	if err != nil {
		return nil, "", "", err
	}

	fromDay, err := parseDiaryDay(from)
	if err != nil {
		return nil, "", "", err
	}
	toDay, err := parseDiaryDay(to)
	if err != nil {
		return nil, "", "", err
	}
	if toDay.Before(fromDay) {
		return nil, "", "", errors.New("range ends before it starts")
	}

	entries, err = s.mysqlRepo.GetDiaryEntries(pid, from, to)
	if err != nil {
		return nil, "", "", err
	}

	return entries, role, since, nil
}

// checkWritableDay enforces the two rules the UI also applies: only the adopter
// writes, and only for days between the adoption and today. Re-checked here
// because the client's copy of either rule is not trustworthy.
func (s *PetServiceImpl) checkWritableDay(pid int, uid string, date string) error {
	role, since, err := s.mysqlRepo.GetDiaryAccess(pid, uid)
	if err != nil {
		return err
	}
	if role != domain.DiaryRoleAdopter {
		return errors.New("only the adopter can write this diary")
	}

	day, err := parseDiaryDay(date)
	if err != nil {
		return err
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	if day.After(today) {
		return errors.New("cannot write a diary entry for a future date")
	}

	sinceDay, err := parseDiaryDay(since)
	if err != nil {
		// A stored timestamp that will not parse is a data problem, not the
		// caller's; fall through rather than locking them out of the diary.
		log.Printf("[Diary] pet %d has an unparseable adoption date %q: %v", pid, since, err)
		return nil
	}
	if day.Before(sinceDay) {
		return errors.New("cannot write a diary entry from before the adoption")
	}

	return nil
}

func (s *PetServiceImpl) SaveDiaryEntry(ctx context.Context, uid string, pid int, date string, file *multipart.FileHeader, caption string) (entry domain.DiaryEntry, err error) {
	if err = s.checkWritableDay(pid, uid, date); err != nil {
		return domain.DiaryEntry{}, err
	}

	// Without a new file this is a caption-only edit, which is valid as long as
	// the day already has an entry — the repository enforces that.
	if file == nil {
		if _, lookupErr := s.mysqlRepo.GetDiaryEntry(pid, date); lookupErr != nil {
			if lookupErr == sql.ErrNoRows {
				return domain.DiaryEntry{}, errors.New("a photo is required for a new diary entry")
			}
			return domain.DiaryEntry{}, lookupErr
		}
	}

	imageAddress := ""
	if file != nil {
		urls, uploadErr := s.uploader.UploadImages([]*multipart.FileHeader{file}, "diary")
		if uploadErr != nil {
			return domain.DiaryEntry{}, uploadErr
		}
		imageAddress = urls[0]
	}

	entry, replacedImage, err := s.mysqlRepo.UpsertDiaryEntry(pid, uid, date, imageAddress, caption)
	if err != nil {
		// The upload already happened, so drop the orphan rather than leaving
		// an unreferenced file in storage forever.
		if imageAddress != "" {
			if deleteErr := s.uploader.DeleteImage(imageAddress); deleteErr != nil {
				log.Printf("[SaveDiaryEntry] failed to delete orphaned image %q: %v", imageAddress, deleteErr)
			}
		}
		return domain.DiaryEntry{}, err
	}

	// Best-effort: the row already points at the new photo, so a failure here
	// only leaves a stale file behind and must not fail the request.
	if replacedImage != "" {
		if deleteErr := s.uploader.DeleteImage(replacedImage); deleteErr != nil {
			log.Printf("[SaveDiaryEntry] failed to delete replaced image %q for pet %d: %v", replacedImage, pid, deleteErr)
		}
	}

	return entry, nil
}

func (s *PetServiceImpl) DeleteDiaryEntry(ctx context.Context, uid string, pid int, date string) (err error) {
	if err = s.checkWritableDay(pid, uid, date); err != nil {
		return err
	}

	removedImage, err := s.mysqlRepo.DeleteDiaryEntry(pid, date)
	if err != nil {
		return err
	}

	if removedImage != "" {
		if deleteErr := s.uploader.DeleteImage(removedImage); deleteErr != nil {
			log.Printf("[DeleteDiaryEntry] failed to delete image %q for pet %d: %v", removedImage, pid, deleteErr)
		}
	}

	return nil
}
