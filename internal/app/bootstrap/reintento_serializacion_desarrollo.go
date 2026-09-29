package bootstrap

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// La composición de desarrollo publica la instantánea de autorización en
// cada petición. Dos peticiones simultáneas del mismo perfil (por ejemplo, la
// bandeja y las incorporaciones que la página del centro pide a la vez) se
// ordenan en el bloqueo consultivo del perfil; la segunda tomó su instantánea
// SERIALIZABLE antes de esperar y, al leer con FOR UPDATE la asignación que la
// primera acaba de confirmar, PostgreSQL la aborta con 40001. Esa pérdida de
// carrera no es un fallo del servicio: se repite la publicación completa, que
// el aborto dejó sin efecto. Misma política que la transacción de consulta de
// contratación temporal (adapters/postgres/reintento_serializable.go).
const (
	intentosPublicacionSerializableCTDesarrollo   = 6
	esperaBasePublicacionSerializableCTDesarrollo = 25 * time.Millisecond
)

// falloSerializacionPostgreSQLCTDesarrollo es el centinela de siempre marcado
// como pérdida de carrera (40001 o 40P01). errors.Is con el centinela sigue
// funcionando; solo el reintento distingue la marca.
type falloSerializacionPostgreSQLCTDesarrollo struct{}

func (falloSerializacionPostgreSQLCTDesarrollo) Error() string {
	return errPostgreSQLContratacionTemporalDesarrolloNoDisponible.Error()
}

func (falloSerializacionPostgreSQLCTDesarrollo) Unwrap() error {
	return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
}

func causaSerializacionPostgreSQLCTDesarrollo(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && (pg.Code == "40001" || pg.Code == "40P01")
}

func falloReintentableSerializacionCTDesarrollo(err error) bool {
	var marca falloSerializacionPostgreSQLCTDesarrollo
	return errors.As(err, &marca) || causaSerializacionPostgreSQLCTDesarrollo(err)
}

// reintentarSerializacionCTDesarrollo repite intento mientras pierda una
// carrera de serialización o un interbloqueo. Cada intento abre y cierra su
// propia transacción. La espera crece con el intento y lleva una parte
// aleatoria; si el contexto vence o se cancela, deja de esperar y devuelve el
// último error. Cualquier otro error se devuelve sin repetir.
func reintentarSerializacionCTDesarrollo(ctx context.Context, intento func() error) error {
	var err error
	for n := 1; ; n++ {
		err = intento()
		if err == nil || !falloReintentableSerializacionCTDesarrollo(err) ||
			n == intentosPublicacionSerializableCTDesarrollo || ctx == nil || ctx.Err() != nil {
			return err
		}
		espera := time.Duration(n)*esperaBasePublicacionSerializableCTDesarrollo +
			time.Duration(time.Now().UnixNano())%esperaBasePublicacionSerializableCTDesarrollo
		t := time.NewTimer(espera)
		select {
		case <-ctx.Done():
			t.Stop()
			return err
		case <-t.C:
		}
	}
}
