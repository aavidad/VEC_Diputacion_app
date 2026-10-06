// Package plazoarranque amplía, solo mientras el proceso arranca, los plazos
// de las comprobaciones previas (preflight) que hace la composición contra
// PostgreSQL y otros servicios.
//
// Con una CPU muy escasa (por ejemplo, una máquina virtual con mucho robo del
// hipervisor) las mismas comprobaciones agotaban sus plazos de pocos segundos
// y el servidor no llegaba a arrancar. Las comprobaciones y su resultado no
// cambian: solo se espera más. En cuanto el servidor termina de componerse
// se llama a Terminar y todos los plazos vuelven a ser exactamente los
// declarados en el código; las peticiones nunca ven el plazo ampliado.
//
// Es estado de proceso escrito una sola vez (Fijar) y cerrado una sola vez
// (Terminar), sin vuelta atrás: no hay otra forma de alcanzar los plazos de
// los adaptadores sin cambiar cientos de firmas.
package plazoarranque

import (
	"errors"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// MaximoPlazo es el mayor plazo de arranque admitido.
const MaximoPlazo = 600 * time.Second

// ErrPlazoInvalido indica un valor no admitido: no es un número entero de
// segundos entre 1 y 600.
var ErrPlazoInvalido = errors.New("plazoarranque: plazo de arranque no válido")

var (
	minimo    atomic.Int64 // nanosegundos; 0 = sin ampliación
	terminado atomic.Bool
	fijado    atomic.Bool
)

// Analizar interpreta el valor de la variable de entorno: vacío significa sin
// ampliación; si no, un entero de segundos entre 1 y 600 (se admite el sufijo
// «s»).
func Analizar(valor string) (time.Duration, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return 0, nil
	}
	valor = strings.TrimSuffix(valor, "s")
	n, err := strconv.Atoi(valor)
	if err != nil || n < 1 || time.Duration(n)*time.Second > MaximoPlazo {
		return 0, ErrPlazoInvalido
	}
	return time.Duration(n) * time.Second, nil
}

// Fijar establece el plazo mínimo de las comprobaciones de arranque. Solo
// tiene efecto la primera vez y antes de Terminar.
func Fijar(plazo time.Duration) error {
	if plazo < 0 || plazo > MaximoPlazo {
		return ErrPlazoInvalido
	}
	if terminado.Load() || !fijado.CompareAndSwap(false, true) {
		return nil
	}
	minimo.Store(int64(plazo))
	return nil
}

// Terminar cierra el arranque: desde aquí Ampliar devuelve siempre el plazo
// recibido.
func Terminar() { terminado.Store(true) }

// Ampliar devuelve el plazo declarado o, mientras el proceso arranca, el
// mínimo fijado si es mayor.
func Ampliar(plazo time.Duration) time.Duration {
	if terminado.Load() {
		return plazo
	}
	if m := time.Duration(minimo.Load()); m > plazo {
		return m
	}
	return plazo
}
