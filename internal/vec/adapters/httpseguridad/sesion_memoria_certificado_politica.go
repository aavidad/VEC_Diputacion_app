package httpseguridad

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
)

var ErrTokenSesionMemoriaCertificadoNoValido = errors.New("token de sesion de certificado no valido")

// Conserva la causa para diagnóstico interno sin exponer contenido del token
// ni del catálogo por Error, fmt o slog.
type falloTokenSesionMemoriaCertificado struct{ causa error }

func (falloTokenSesionMemoriaCertificado) Error() string {
	return ErrTokenSesionMemoriaCertificadoNoValido.Error()
}
func (f falloTokenSesionMemoriaCertificado) Unwrap() []error {
	return []error{ErrTokenSesionMemoriaCertificadoNoValido, f.causa}
}
func (f falloTokenSesionMemoriaCertificado) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, f.Error())
}
func (f falloTokenSesionMemoriaCertificado) LogValue() slog.Value {
	return slog.StringValue(f.Error())
}

const (
	esquemaSesionMemoriaCertificado = "vec.identidad.sesion-memoria-certificado.v1"
	prefijoPoliticaSesionMemoria    = "politica:identidad:sesion-memoria-certificado:"
	maximoDocumentoPoliticaSesion   = 4096
	maximoTecnicoVidaToken          = 10 * time.Minute
	maximoTecnicoTokens             = 65536
	bytesTokenSesion                = 32
)

// PoliticaSesionMemoriaCertificado solo procede de datos publicados que la
// composición haya seleccionado. El constructor no aporta valores por defecto.
type PoliticaSesionMemoriaCertificado struct {
	referencia string
	version    uint64
	vida       time.Duration
	capacidad  int
}

type documentoPoliticaSesionMemoriaCertificado struct {
	Esquema     string `json:"esquema"`
	Referencia  string `json:"referencia"`
	Version     uint64 `json:"version"`
	TTLSegundos uint64 `json:"ttl_segundos"`
	Capacidad   uint64 `json:"capacidad"`
}

// CargarPoliticaSesionMemoriaCertificado recibe el documento de la fuente de
// catálogo elegida por composición; nunca una petición HTTP ni un secreto.
func CargarPoliticaSesionMemoriaCertificado(lector io.Reader) (PoliticaSesionMemoriaCertificado, error) {
	var vacia PoliticaSesionMemoriaCertificado
	if interfazNulaPasarela(lector) {
		return vacia, ErrTokenSesionMemoriaCertificadoNoValido
	}
	contenido, err := io.ReadAll(io.LimitReader(lector, maximoDocumentoPoliticaSesion+1))
	if err != nil {
		return vacia, falloTokenSesionMemoriaCertificado{causa: err}
	}
	if len(contenido) == 0 || len(contenido) > maximoDocumentoPoliticaSesion {
		return vacia, ErrTokenSesionMemoriaCertificadoNoValido
	}
	if err := clavesUnicasPoliticaSesion(contenido); err != nil {
		return vacia, err
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var documento documentoPoliticaSesionMemoriaCertificado
	if err := decodificador.Decode(&documento); err != nil {
		return vacia, falloTokenSesionMemoriaCertificado{causa: err}
	}
	if err := decodificador.Decode(new(any)); err != io.EOF {
		if err != nil {
			return vacia, falloTokenSesionMemoriaCertificado{causa: err}
		}
		return vacia, ErrTokenSesionMemoriaCertificadoNoValido
	}
	if documento.Esquema != esquemaSesionMemoriaCertificado ||
		!referenciaPoliticaSesionValida(documento.Referencia) || documento.Version == 0 ||
		documento.TTLSegundos == 0 || documento.TTLSegundos > uint64(maximoTecnicoVidaToken/time.Second) ||
		documento.Capacidad == 0 || documento.Capacidad > maximoTecnicoTokens {
		return vacia, ErrTokenSesionMemoriaCertificadoNoValido
	}
	return PoliticaSesionMemoriaCertificado{
		referencia: documento.Referencia, version: documento.Version,
		vida:      time.Duration(documento.TTLSegundos) * time.Second,
		capacidad: int(documento.Capacidad),
	}, nil
}

func clavesUnicasPoliticaSesion(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	apertura, err := decodificador.Token()
	if err != nil {
		return falloTokenSesionMemoriaCertificado{causa: err}
	}
	if apertura != json.Delim('{') {
		return ErrTokenSesionMemoriaCertificadoNoValido
	}
	vistas := map[string]bool{}
	for decodificador.More() {
		clave, err := decodificador.Token()
		if err != nil {
			return falloTokenSesionMemoriaCertificado{causa: err}
		}
		nombre, ok := clave.(string)
		if !ok || vistas[nombre] {
			return ErrTokenSesionMemoriaCertificadoNoValido
		}
		vistas[nombre] = true
		var valor json.RawMessage
		if err := decodificador.Decode(&valor); err != nil {
			return falloTokenSesionMemoriaCertificado{causa: err}
		}
	}
	cierre, err := decodificador.Token()
	if err != nil {
		return falloTokenSesionMemoriaCertificado{causa: err}
	}
	if cierre != json.Delim('}') || len(vistas) != 5 {
		return ErrTokenSesionMemoriaCertificadoNoValido
	}
	_, err = decodificador.Token()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return falloTokenSesionMemoriaCertificado{causa: err}
	}
	return ErrTokenSesionMemoriaCertificadoNoValido
}

func referenciaPoliticaSesionValida(referencia string) bool {
	if !strings.HasPrefix(referencia, prefijoPoliticaSesionMemoria) ||
		len(referencia) <= len(prefijoPoliticaSesionMemoria) || len(referencia) > 160 {
		return false
	}
	for _, c := range referencia[len(prefijoPoliticaSesionMemoria):] {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
