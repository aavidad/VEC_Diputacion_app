package domain

import (
	"errors"
	"net/mail"
	"strings"
	"time"
)

var ErrCorreosInvalidos = errors.New("usuarios: conjunto de correos invalido")

type EstadoCorreo string

const (
	CorreoPendiente  EstadoCorreo = "pendiente"
	CorreoVerificado EstadoCorreo = "verificado"
	CorreoRetirado   EstadoCorreo = "retirado"
)

// CorreoPropio contiene sólo metadatos presentables a su titular. La dirección
// se entrega únicamente en la lectura autorizada; nunca contiene desafío,
// código ni material criptográfico.
type CorreoPropio struct {
	CorreoRef     string       `json:"correo_ref"`
	Direccion     string       `json:"direccion"`
	Estado        EstadoCorreo `json:"estado"`
	Activo        bool         `json:"activo"`
	CreadoUTC     time.Time    `json:"creado_utc"`
	VerificadoUTC *time.Time   `json:"verificado_utc,omitempty"`
}

func DireccionCorreoValida(direccion string) bool {
	if len(direccion) < 3 || len(direccion) > 254 || strings.TrimSpace(direccion) != direccion || strings.ContainsAny(direccion, "\r\n\x00") {
		return false
	}
	a, err := mail.ParseAddress(direccion)
	if err != nil || a.Address != direccion || a.Name != "" || strings.Count(direccion, "@") != 1 {
		return false
	}
	partes := strings.SplitN(direccion, "@", 2)
	return len(partes[0]) <= 64 && len(partes[1]) <= 253 && strings.Contains(partes[1], ".")
}

// ValidarConjuntoCorreos es una segunda guarda sobre las proyecciones del
// registro. La unicidad y la transición concurrente se imponen además en SQL.
func ValidarConjuntoCorreos(correos []CorreoPropio) error {
	refs := make(map[string]bool, len(correos))
	direcciones := make(map[string]bool, len(correos))
	activos := 0
	for _, correo := range correos {
		if correo.CorreoRef == "" || refs[correo.CorreoRef] || !DireccionCorreoValida(correo.Direccion) || correo.CreadoUTC.IsZero() {
			return ErrCorreosInvalidos
		}
		refs[correo.CorreoRef] = true
		direccion := strings.ToLower(correo.Direccion)
		if correo.Estado != CorreoRetirado && direcciones[direccion] {
			return ErrCorreosInvalidos
		}
		if correo.Estado != CorreoRetirado {
			direcciones[direccion] = true
		}
		switch correo.Estado {
		case CorreoPendiente:
			if correo.Activo || correo.VerificadoUTC != nil {
				return ErrCorreosInvalidos
			}
		case CorreoVerificado:
			if correo.VerificadoUTC == nil || correo.VerificadoUTC.IsZero() {
				return ErrCorreosInvalidos
			}
		case CorreoRetirado:
			if correo.Activo {
				return ErrCorreosInvalidos
			}
		default:
			return ErrCorreosInvalidos
		}
		if correo.Activo {
			activos++
		}
	}
	if activos > 1 {
		return ErrCorreosInvalidos
	}
	return nil
}
