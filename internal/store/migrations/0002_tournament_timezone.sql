-- Single IANA timezone anchor per tournament (e.g. Asia/Seoul). Nullable.
-- The backend stays timezone-naive; this is the frontend's display anchor.
ALTER TABLE tournament ADD COLUMN time_zone TEXT;
