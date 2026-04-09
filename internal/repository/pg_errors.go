package repository

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// IsUniqueViolation проверяет, является ли ошибка нарушением уникальности (SQLSTATE 23505)
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// IsForeignKeyViolation проверяет, является ли ошибка нарушением внешнего ключа (SQLSTATE 23503)
func IsForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23503"
	}
	return false
}

// IsNoRows проверяет, что запрос не вернул ни одной строки
func IsNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
