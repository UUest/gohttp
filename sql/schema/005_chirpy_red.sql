-- +goose Up
ALTER TABLE users
ADD COLUMN chirpy_red BOOLEAN DEFAULT FALSE;
-- +goose Down
