package postgresql

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// Política única de reintento de transacciones SERIALIZABLE.
//
// Varias funciones de VEC ordenan sus escrituras con SELECT ... FOR UPDATE
// sobre una única fila de control (por ejemplo, la cadena de auditoría de
// consumos V3). En SERIALIZABLE la instantánea se toma antes de esperar ese
// cerrojo, así que cuando varias peticiones llegan a la vez solo la primera
// confirma y las demás reciben 40001 al obtenerlo. Es una pérdida de carrera,
// no un fallo: el aborto deja la transacción sin efectos y se repite entera.
//
// La espera crece exponencialmente con un tope y es aleatoria dentro de ese
// tramo («full jitter»), de modo que las peticiones que chocaron no vuelvan a
// coincidir. El número de intentos basta para absorber decenas de peticiones
// simultáneas sobre la misma fila; el contexto de la petición manda siempre.
const (
	IntentosMaximosCarreraSerializable = 30
	esperaInicialCarreraSerializable   = 4 * time.Millisecond
	esperaMaximaCarreraSerializable    = 200 * time.Millisecond
)

// MarcaCarreraSerializable la implementan los errores propios de un adaptador
// que ya tradujeron un 40001/40P01 y quieren que el intento se repita.
type MarcaCarreraSerializable interface {
	CarreraSerializable() bool
}

// EsCarreraSerializable indica si err es una pérdida de carrera repetible:
// 40001 (serialization_failure), 40P01 (deadlock_detected) o un error marcado.
// Ningún otro código se repite: 55P03, 23505 o una guarda rechazada siguen
// siendo fallos definitivos.
func EsCarreraSerializable(err error) bool {
	if err == nil {
		return false
	}
	var marca MarcaCarreraSerializable
	if errors.As(err, &marca) && marca.CarreraSerializable() {
		return true
	}
	var pg *pgconn.PgError
	return errors.As(err, &pg) && (pg.Code == "40001" || pg.Code == "40P01")
}

// RepetirTrasCarreraSerializable ejecuta intento y lo repite mientras pierda
// una carrera de serialización. Cada intento debe abrir y cerrar su propia
// transacción. Si el contexto vence o se cancela, o se agotan los intentos,
// devuelve el último error tal cual.
func RepetirTrasCarreraSerializable(ctx context.Context, intento func() error) error {
	var err error
	for n := 1; ; n++ {
		err = intento()
		if !EsCarreraSerializable(err) || n >= IntentosMaximosCarreraSerializable || ctx == nil || ctx.Err() != nil {
			return err
		}
		if !EsperarReintentoCarreraSerializable(ctx, n) {
			return err
		}
	}
}

// EsperarReintentoCarreraSerializable espera antes del intento n+1 y devuelve
// false si el contexto vence o se cancela mientras tanto.
func EsperarReintentoCarreraSerializable(ctx context.Context, n int) bool {
	if ctx == nil || ctx.Err() != nil {
		return false
	}
	t := time.NewTimer(esperaCarreraSerializable(n))
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return ctx.Err() == nil
	}
}

func esperaCarreraSerializable(n int) time.Duration {
	tope := esperaMaximaCarreraSerializable
	if n < 16 {
		if exponencial := esperaInicialCarreraSerializable << n; exponencial < tope {
			tope = exponencial
		}
	}
	// La aleatoriedad solo reparte reintentos; si falla, se espera el tope.
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Millisecond + tope
	}
	return time.Millisecond + time.Duration(binary.LittleEndian.Uint64(b[:])%uint64(tope))
}
