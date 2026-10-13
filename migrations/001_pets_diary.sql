-- Pet Diary storage.
--
-- The repo has no migration tooling, so run this by hand against the same
-- MySQL database the API uses:
--   mysql -u <user> -p <database> < migrations/001_pets_diary.sql
--
-- One entry per pet per day is enforced by the DB, not just the UI, so a
-- double-submit or a second tab cannot produce two rows for the same date.
-- diary_authorId is recorded for attribution but deliberately not part of the
-- unique key: the day belongs to the pet, whoever wrote it.

CREATE TABLE IF NOT EXISTS Pets_Diary (
    diary_id           INT AUTO_INCREMENT PRIMARY KEY,
    diary_petId        INT          NOT NULL,
    diary_authorId     VARCHAR(255) NOT NULL,
    diary_date         DATE         NOT NULL,
    diary_imageAddress VARCHAR(512) NOT NULL,
    diary_caption      TEXT,
    diary_createAt     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    diary_updateAt     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE KEY uq_diary_pet_date (diary_petId, diary_date),
    -- The diary is meaningless once the pet record is gone.
    CONSTRAINT fk_diary_pet FOREIGN KEY (diary_petId)
        REFERENCES Pets (pet_id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
