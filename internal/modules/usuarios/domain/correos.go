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

// Límites del conjunto propio. SQL los vuelve a imponer.
const (
	MaxCorreosVivos         = 5
	MaxIntentosCodigoCorreo = 5
	DigitosCodigoCorreo     = 8
)

// CodigoPendiente describe el código vigente de una dirección sin confirmar.
// Nunca contiene el código, sólo su vencimiento e intentos restantes.
type CodigoPendiente struct {
	VenceUTC          time.Time `json:"vence_utc"`
	IntentosRestantes int       `json:"intentos_restantes"`
}

// CorreoPropio contiene sólo metadatos presentables a su titular. La dirección
// se entrega únicamente en la lectura autorizada; nunca contiene desafío,
// código ni material criptográfico.
type CorreoPropio struct {
	CorreoRef     string           `json:"correo_ref"`
	Direccion     string           `json:"direccion"`
	Estado        EstadoCorreo     `json:"estado"`
	Activo        bool             `json:"activo"`
	CreadoUTC     time.Time        `json:"creado_utc"`
	VerificadoUTC *time.Time       `json:"verificado_utc,omitempty"`
	Codigo        *CodigoPendiente `json:"codigo,omitempty"`
}

// DireccionCorreoValida admite sólo una dirección ASCII «dot-atom» sin nombre
// visible, comillas, espacios ni saltos de línea: la misma forma que acepta el
// transporte SMTP de VEC para el sobre y la cabecera To.
func DireccionCorreoValida(direccion string) bool {
	if len(direccion) < 6 || len(direccion) > 254 || strings.TrimSpace(direccion) != direccion || strings.ContainsAny(direccion, "\r\n\x00\" <>()[],;:\\") {
		return false
	}
	if analizarDireccion(direccion) != nil {
		return false
	}
	local, dominio, _ := strings.Cut(direccion, "@")
	return localValido(local) && dominioValido(dominio)
}

// analizarDireccion exige que net/mail lea la dirección exacta, sin nombre.
func analizarDireccion(direccion string) error {
	a, err := mail.ParseAddress(direccion)
	if err != nil {
		return err
	}
	if a.Address != direccion || a.Name != "" || strings.Count(direccion, "@") != 1 {
		return ErrCorreosInvalidos
	}
	return nil
}

func localValido(local string) bool {
	if local == "" || len(local) > 64 || local[0] == '.' || local[len(local)-1] == '.' || strings.Contains(local, "..") {
		return false
	}
	for _, r := range local {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune(".!#$%&'*+-/=?^_`{|}~", r)) {
			return false
		}
	}
	return true
}

func dominioValido(dominio string) bool {
	if len(dominio) < 4 || len(dominio) > 253 || !strings.Contains(dominio, ".") {
		return false
	}
	etiquetas := strings.Split(dominio, ".")
	for _, etiqueta := range etiquetas {
		if etiqueta == "" || len(etiqueta) > 63 || etiqueta[0] == '-' || etiqueta[len(etiqueta)-1] == '-' {
			return false
		}
		for _, r := range etiqueta {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
				return false
			}
		}
	}
	return len(etiquetas[len(etiquetas)-1]) >= 2
}

// NormalizarCodigoCorreo acepta el código tal como se copia del mensaje
// (con espacios o guiones de agrupación) y devuelve sólo sus 8 dígitos.
func NormalizarCodigoCorreo(codigo string) (string, bool) {
	if len(codigo) > 32 {
		return "", false
	}
	var b strings.Builder
	for _, r := range codigo {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '\t':
		default:
			return "", false
		}
	}
	if b.Len() != DigitosCodigoCorreo {
		return "", false
	}
	return b.String(), true
}

// ValidarConjuntoCorreos es una segunda guarda sobre las proyecciones del
// registro. La unicidad y la transición concurrente se imponen además en SQL.
func ValidarConjuntoCorreos(correos []CorreoPropio) error {
	if len(correos) > MaxCorreosVivos {
		return ErrCorreosInvalidos
	}
	refs := make(map[string]bool, len(correos))
	direcciones := make(map[string]bool, len(correos))
	activos := 0
	for _, correo := range correos {
		if correo.CorreoRef == "" || refs[correo.CorreoRef] || !DireccionCorreoValida(correo.Direccion) || correo.CreadoUTC.IsZero() {
			return ErrCorreosInvalidos
		}
		refs[correo.CorreoRef] = true
		direccion := strings.ToLower(correo.Direccion)
		if direcciones[direccion] {
			return ErrCorreosInvalidos
		}
		direcciones[direccion] = true
		switch correo.Estado {
		case CorreoPendiente:
			if correo.Activo || correo.VerificadoUTC != nil {
				return ErrCorreosInvalidos
			}
			if correo.Codigo != nil && (correo.Codigo.VenceUTC.IsZero() || correo.Codigo.IntentosRestantes < 1 || correo.Codigo.IntentosRestantes > MaxIntentosCodigoCorreo) {
				return ErrCorreosInvalidos
			}
		case CorreoVerificado:
			if correo.VerificadoUTC == nil || correo.VerificadoUTC.IsZero() || correo.Codigo != nil {
				return ErrCorreosInvalidos
			}
		default:
			// La vista del titular no incluye direcciones retiradas.
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
