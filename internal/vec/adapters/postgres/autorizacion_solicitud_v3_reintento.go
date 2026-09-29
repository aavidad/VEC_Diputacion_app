package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// carreraSerializacionRegistroV3 marca un intento abortado por 40001 o 40P01
// para la política común de reintento (internal/shared/postgresql). Solo
// circula dentro del bucle; hacia fuera se devuelve siempre la clasificación
// traducida de siempre.
type carreraSerializacionRegistroV3 struct{ traducido error }

func (c carreraSerializacionRegistroV3) Error() string           { return c.traducido.Error() }
func (c carreraSerializacionRegistroV3) Unwrap() error           { return c.traducido }
func (carreraSerializacionRegistroV3) CarreraSerializable() bool { return true }

func marcarCarreraSerializacionRegistroV3(causa, traducido error) error {
	var errorPG *pgconn.PgError
	if traducido != nil && errors.As(causa, &errorPG) && (errorPG.Code == "40001" || errorPG.Code == "40P01") {
		return carreraSerializacionRegistroV3{traducido: traducido}
	}
	return traducido
}
