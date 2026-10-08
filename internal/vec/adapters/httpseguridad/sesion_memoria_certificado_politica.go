package httpseguridad

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
)

var ErrTokenSesionMemoriaCertificadoNoValido = errors.New("token de sesion de certificado no valido")

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
	if err != nil || len(contenido) == 0 || len(contenido) > maximoDocumentoPoliticaSesion ||
		!clavesUnicasPoliticaSesion(contenido) {
		return vacia, ErrTokenSesionMemoriaCertificadoNoValido
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var documento documentoPoliticaSesionMemoriaCertificado
	if decodificador.Decode(&documento) != nil || decodificador.Decode(new(any)) != io.EOF ||
		documento.Esquema != esquemaSesionMemoriaCertificado ||
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

func clavesUnicasPoliticaSesion(contenido []byte) bool {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	apertura, err := decodificador.Token()
	if err != nil || apertura != json.Delim('{') {
		return false
	}
	vistas := map[string]bool{}
	for decodificador.More() {
		clave, err := decodificador.Token()
		nombre, ok := clave.(string)
		if err != nil || !ok || vistas[nombre] {
			return false
		}
		vistas[nombre] = true
		var valor json.RawMessage
		if decodificador.Decode(&valor) != nil {
			return false
		}
	}
	cierre, err := decodificador.Token()
	if err != nil || cierre != json.Delim('}') || len(vistas) != 5 {
		return false
	}
	_, err = decodificador.Token()
	return err == io.EOF
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
