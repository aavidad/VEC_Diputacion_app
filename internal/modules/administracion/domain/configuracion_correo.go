// Package domain contiene las reglas de la configuración administrativa.
package domain

import (
	"encoding/json"
	"errors"
	"net"
	"net/mail"
	"strings"
	"time"
	"unicode"
)

var ErrConfiguracionCorreoInvalida = errors.New("administracion: configuracion de correo invalida")

type ModoTLSCorreo string
type ModoAutenticacionCorreo string

const (
	ModoTLSCorreoImplicito         ModoTLSCorreo           = "tls_implicito"
	ModoTLSCorreoSTARTTLS          ModoTLSCorreo           = "starttls_obligatorio"
	ModoAutenticacionCorreoNinguna ModoAutenticacionCorreo = "ninguna"
	ModoAutenticacionCorreoPlain   ModoAutenticacionCorreo = "plain"
	ModoAutenticacionCorreoXOAUTH2 ModoAutenticacionCorreo = "xoauth2"
)

// VistaConfiguracionCorreo es la única representación que puede salir de la
// frontera administrativa. No contiene el secreto ni material de CA.
type VistaConfiguracionCorreo struct {
	Configurada        bool                    `json:"configurada"`
	Host               string                  `json:"host"`
	Puerto             uint16                  `json:"puerto"`
	NombreServidor     string                  `json:"server_name"`
	ReferenciaCA       string                  `json:"referencia_ca"`
	RemitenteFijo      string                  `json:"remitente_fijo"`
	Usuario            string                  `json:"usuario"`
	ModoTLS            ModoTLSCorreo           `json:"modo_tls"`
	ModoAutenticacion  ModoAutenticacionCorreo `json:"modo_autenticacion"`
	TiempoMaximo       time.Duration           `json:"-"`
	TiempoMaximoMillis int64                   `json:"tiempo_maximo_ms"`
	SecretoConfigurado bool                    `json:"secreto_configurado"`
	Version            uint64                  `json:"version"`
}

func (v VistaConfiguracionCorreo) Validar() error {
	if !v.Configurada {
		if v.Host != "" || v.Puerto != 0 || v.NombreServidor != "" || v.ReferenciaCA != "" ||
			v.RemitenteFijo != "" || v.Usuario != "" || v.ModoTLS != "" || v.ModoAutenticacion != "" ||
			v.TiempoMaximo != 0 || v.TiempoMaximoMillis != 0 || v.SecretoConfigurado || v.Version != 0 {
			return ErrConfiguracionCorreoInvalida
		}
		return nil
	}
	if !hostValido(v.Host) || v.Puerto == 0 || !nombreServidorValido(v.NombreServidor) ||
		!referenciaValida(v.ReferenciaCA) || !direccionValida(v.RemitenteFijo) ||
		(v.ModoTLS != ModoTLSCorreoImplicito && v.ModoTLS != ModoTLSCorreoSTARTTLS) ||
		v.TiempoMaximoMillis <= 0 || v.TiempoMaximoMillis > 3600000 || v.Version == 0 || contieneControl(v.Usuario) ||
		(v.ModoAutenticacion != ModoAutenticacionCorreoNinguna && v.ModoAutenticacion != ModoAutenticacionCorreoPlain && v.ModoAutenticacion != ModoAutenticacionCorreoXOAUTH2) {
		return ErrConfiguracionCorreoInvalida
	}
	if strings.TrimSpace(v.Usuario) != v.Usuario {
		return ErrConfiguracionCorreoInvalida
	}
	if v.ModoAutenticacion == ModoAutenticacionCorreoNinguna && (v.Usuario != "" || v.SecretoConfigurado) {
		return ErrConfiguracionCorreoInvalida
	}
	if v.ModoAutenticacion != ModoAutenticacionCorreoNinguna && (v.Usuario == "" || !v.SecretoConfigurado) {
		return ErrConfiguracionCorreoInvalida
	}
	return nil
}

// SecretoCorreo es write-only. Solamente el adaptador de persistencia puede
// consumirlo durante la misma operación atómica; nunca se serializa ni se
// representa en logs.
type SecretoCorreo struct{ bytes []byte }

func NuevoSecretoCorreo(valor []byte) (SecretoCorreo, error) {
	if len(valor) == 0 || len(valor) > 16384 {
		return SecretoCorreo{}, ErrConfiguracionCorreoInvalida
	}
	return SecretoCorreo{bytes: append([]byte(nil), valor...)}, nil
}
func (SecretoCorreo) String() string               { return "secreto_correo_redactado" }
func (SecretoCorreo) GoString() string             { return "secreto_correo_redactado" }
func (SecretoCorreo) MarshalJSON() ([]byte, error) { return []byte(`"secreto_correo_redactado"`), nil }
func (s SecretoCorreo) Consumir(fn func([]byte) error) error {
	if len(s.bytes) == 0 || fn == nil {
		return ErrConfiguracionCorreoInvalida
	}
	b := append([]byte(nil), s.bytes...)
	defer borrar(b)
	return fn(b)
}

// ActualizacionConfiguracionCorreo sólo acepta secreto si se desea sustituir.
// Un nil conserva el material protegido que ya exista.
type ActualizacionConfiguracionCorreo struct {
	VistaConfiguracionCorreo
	VersionEsperada uint64         `json:"version_esperada"`
	SecretoNuevo    *SecretoCorreo `json:"-"`
}

func (a ActualizacionConfiguracionCorreo) Validar() error {
	vista := a.VistaConfiguracionCorreo
	if !a.Configurada || vista.Version != 0 || a.VersionEsperada >= uint64(1<<63-1) ||
		(a.ModoAutenticacion == ModoAutenticacionCorreoNinguna && a.SecretoNuevo != nil) {
		return ErrConfiguracionCorreoInvalida
	}
	vista.Version = 1
	if vista.Validar() != nil {
		return ErrConfiguracionCorreoInvalida
	}
	if a.ModoAutenticacion != ModoAutenticacionCorreoNinguna && !a.SecretoConfigurado && a.SecretoNuevo == nil {
		return ErrConfiguracionCorreoInvalida
	}
	return nil
}
func (ActualizacionConfiguracionCorreo) String() string {
	return "actualizacion_configuracion_correo_redactada"
}
func (ActualizacionConfiguracionCorreo) GoString() string {
	return "actualizacion_configuracion_correo_redactada"
}
func (a ActualizacionConfiguracionCorreo) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Configurada        bool                    `json:"configurada"`
		Host               string                  `json:"host"`
		Puerto             uint16                  `json:"puerto"`
		NombreServidor     string                  `json:"server_name"`
		ReferenciaCA       string                  `json:"referencia_ca"`
		RemitenteFijo      string                  `json:"remitente_fijo"`
		Usuario            string                  `json:"usuario"`
		ModoTLS            ModoTLSCorreo           `json:"modo_tls"`
		ModoAutenticacion  ModoAutenticacionCorreo `json:"modo_autenticacion"`
		TiempoMaximoMillis int64                   `json:"tiempo_maximo_ms"`
		SecretoConfigurado bool                    `json:"secreto_configurado"`
		VersionEsperada    uint64                  `json:"version_esperada"`
	}{a.Configurada, a.Host, a.Puerto, a.NombreServidor, a.ReferenciaCA, a.RemitenteFijo, a.Usuario, a.ModoTLS, a.ModoAutenticacion, a.TiempoMaximoMillis, a.SecretoConfigurado, a.VersionEsperada})
}

func hostValido(v string) bool {
	return v != "" && strings.TrimSpace(v) == v && !contieneControl(v) && len(v) <= 253 &&
		(net.ParseIP(v) != nil || nombreServidorValido(v))
}
func nombreServidorValido(v string) bool {
	if net.ParseIP(v) != nil {
		return true
	}
	if v == "" || len(v) > 253 || contieneControl(v) {
		return false
	}
	for _, etiqueta := range strings.Split(v, ".") {
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
func referenciaValida(v string) bool {
	return v != "" && len(v) <= 512 && strings.TrimSpace(v) == v && !contieneControl(v)
}
func direccionValida(v string) bool {
	d, e := mail.ParseAddress(v)
	return e == nil && d.Address == v && !contieneControl(v)
}
func contieneControl(v string) bool { return strings.ContainsFunc(v, unicode.IsControl) }
func borrar(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
