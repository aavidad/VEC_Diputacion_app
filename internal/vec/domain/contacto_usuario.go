package domain

import (
	"encoding/json"
	"errors"
	"net/mail"
	"strings"
)

var ErrContactoUsuarioInvalido = errors.New("vec: contacto de usuario invalido")

// ContactoUsuario liga una dirección declarada al sujeto nominal VEC. No es
// un directorio ni una identidad de Bolsa; la dirección sólo se abre mediante
// ConDireccion después de que un puerto autorizado haya resuelto el contacto.
type ContactoUsuario struct {
	sujetoRef string
	version   uint64
	direccion string
}

func NuevoContactoUsuario(sujetoRef, direccion string, version uint64) (ContactoUsuario, error) {
	if !ReferenciaSujetoContactoUsuarioValida(sujetoRef) || version == 0 || version > 1<<53-1 || !direccionCorreoUsuarioValida(direccion) {
		return ContactoUsuario{}, ErrContactoUsuarioInvalido
	}
	return ContactoUsuario{sujetoRef: sujetoRef, version: version, direccion: direccion}, nil
}

func (c ContactoUsuario) SujetoRef() string { return c.sujetoRef }
func (c ContactoUsuario) Version() uint64   { return c.version }
func (c ContactoUsuario) Validar() error {
	if !ReferenciaSujetoContactoUsuarioValida(c.sujetoRef) || c.version == 0 || c.version > 1<<53-1 || !direccionCorreoUsuarioValida(c.direccion) {
		return ErrContactoUsuarioInvalido
	}
	return nil
}

// ReferenciaSujetoContactoUsuarioValida acepta exclusivamente la persona
// canónica VEC. El contacto propio no toma identidades ni direcciones de
// Bolsa: el vínculo entre ambos espacios, si llega a existir, se resuelve
// antes en el contexto de identidad.
func ReferenciaSujetoContactoUsuarioValida(valor string) bool {
	return referenciaOpacaContextoActorValida(valor, "per_")
}

// ConDireccion no conserva, serializa ni registra la dirección; el receptor
// debe usarla durante el callback y no devolverla al exterior.
func (c ContactoUsuario) ConDireccion(ejecutar func(string) error) error {
	if c.Validar() != nil || ejecutar == nil {
		return ErrContactoUsuarioInvalido
	}
	return ejecutar(c.direccion)
}
func (ContactoUsuario) String() string   { return "vec.ContactoUsuario{redactado}" }
func (ContactoUsuario) GoString() string { return "vec.ContactoUsuario{redactado}" }
func (ContactoUsuario) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Redactado bool `json:"redactado"`
	}{true})
}

func direccionCorreoUsuarioValida(valor string) bool {
	if valor == "" || len(valor) > 254 || strings.ContainsAny(valor, "\r\n") || !esASCIIVisible(valor) {
		return false
	}
	d, err := mail.ParseAddress(valor)
	if err != nil || d.Address != valor {
		return false
	}
	if strings.Count(valor, "@") != 1 {
		return false
	}
	local, dominio, _ := strings.Cut(valor, "@")
	return len(local) <= 64 && atomCorreoValido(local, true) && dominioCorreoValido(dominio)
}
func esASCIIVisible(s string) bool {
	for _, r := range s {
		if r < 33 || r > 126 {
			return false
		}
	}
	return true
}
func atomCorreoValido(s string, local bool) bool {
	if s == "" || s[0] == '.' || s[len(s)-1] == '.' || strings.Contains(s, "..") {
		return false
	}
	for _, r := range s {
		ok := r == '.' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
		if local {
			ok = ok || strings.ContainsRune("!#$%&'*+-/=?^_`{|}~", r)
		}
		if !ok {
			return false
		}
	}
	return true
}
func dominioCorreoValido(s string) bool {
	if s == "" || len(s) > 253 || strings.HasPrefix(s, ".") || strings.HasSuffix(s, ".") || strings.Contains(s, "..") {
		return false
	}
	for _, etiqueta := range strings.Split(s, ".") {
		if len(etiqueta) == 0 || len(etiqueta) > 63 || etiqueta[0] == '-' || etiqueta[len(etiqueta)-1] == '-' {
			return false
		}
		for _, r := range etiqueta {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
				return false
			}
		}
	}
	return true
}
