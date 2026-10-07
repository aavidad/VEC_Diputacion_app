package ports

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrPaginaCandidatosRRHHInvalida = errors.New("bolsa: pagina RRHH invalida")

// FiltroPaginaCandidatosRRHH conserva los argumentos de la ruta existente.
// El alcance del actor se aplica antes de consultar: este filtro no lo concede.
type FiltroPaginaCandidatosRRHH struct {
	BolsaRef string
	Estado   string
	Texto    string
	Cursor   string
	Limite   int
	Corte    time.Time
}

// Validar no interpreta el cursor ni acepta una referencia de actor del HTTP.
// La autoridad propietaria debe ligar cursor, actor, filtro y corte al leer.
func (f FiltroPaginaCandidatosRRHH) Validar() error {
	if f.BolsaRef == "" || len(f.BolsaRef) > 256 || strings.TrimSpace(f.BolsaRef) != f.BolsaRef || strings.Contains(f.BolsaRef, "/") ||
		!EstadoPaginaRRHHValido(f.Estado) || len(f.Texto) > 100 || !utf8.ValidString(f.Texto) ||
		len(f.Cursor) > 256 || !utf8.ValidString(f.Cursor) || strings.TrimSpace(f.Cursor) != f.Cursor ||
		f.Limite < 1 || f.Limite > 100 || f.Corte.IsZero() || f.Corte.Location() != time.UTC {
		return ErrPaginaCandidatosRRHHInvalida
	}
	for _, r := range f.Cursor {
		if r < '!' || r == 0x7f {
			return ErrPaginaCandidatosRRHHInvalida
		}
	}
	return nil
}

// EstadoPaginaRRHHValido usa el catálogo de situaciones que el cuadro RRHH
// publica hoy; renuncia es situación B2, no el terminal de llamamiento.
func EstadoPaginaRRHHValido(estado string) bool {
	switch estado {
	case "", "disponible", "no_disponible", "trabajando", "pendiente_incorporacion", "renuncia", "excluido", "disponible_desde", "en_revision":
		return true
	default:
		return false
	}
}
