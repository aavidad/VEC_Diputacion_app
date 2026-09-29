package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// Misma política que la transacción de consulta de contratación temporal:
// hasta seis intentos con espera creciente y una parte aleatoria para que las
// peticiones no vuelvan a coincidir.
const (
	intentosRegistroContextoActorV3 = 6
	esperaBaseReintentoRegistroV3   = 25 * time.Millisecond
)

// carreraSerializacionRegistroV3 marca un intento abortado por 40001 o 40P01.
// Solo circula dentro del bucle de reintento; hacia fuera se devuelve siempre
// la clasificación traducida de siempre.
type carreraSerializacionRegistroV3 struct{ traducido error }

func (c carreraSerializacionRegistroV3) Error() string { return c.traducido.Error() }
func (c carreraSerializacionRegistroV3) Unwrap() error { return c.traducido }

func marcarCarreraSerializacionRegistroV3(causa, traducido error) error {
	var errorPG *pgconn.PgError
	if traducido != nil && errors.As(causa, &errorPG) && (errorPG.Code == "40001" || errorPG.Code == "40P01") {
		return carreraSerializacionRegistroV3{traducido: traducido}
	}
	return traducido
}

// esperarReintentoRegistroV3 espera antes del intento siguiente y devuelve
// false si el contexto ya venció o se cancela durante la espera.
func esperarReintentoRegistroV3(ctx context.Context, intento int) bool {
	if ctx == nil || ctx.Err() != nil {
		return false
	}
	espera := time.Duration(intento)*esperaBaseReintentoRegistroV3 +
		time.Duration(time.Now().UnixNano())%esperaBaseReintentoRegistroV3
	t := time.NewTimer(espera)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
