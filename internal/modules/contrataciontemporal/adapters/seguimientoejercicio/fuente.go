// Package seguimientoejercicio expone una definición sintética, inmutable y
// solo de lectura para la composición local. No publica, autoriza ni registra.
package seguimientoejercicio

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	esquemaDocumento      = "vec.contratacion-temporal.seguimiento.ejercicio.v1"
	maximoBytesDocumento  = 128 << 10
	maximaProfundidadJSON = 16
	maximoEnteroJSON      = uint64(9_007_199_254_740_991)
)

var (
	ErrFuenteInvalida    = errors.New("contratacion temporal: fuente de seguimiento de ejercicio invalida")
	ErrNoDisponible      = errors.New("contratacion temporal: definicion de seguimiento de ejercicio no disponible")
	patronSHA256HexLower = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

// Configuracion procede de composición confiable. HuellaArchivoSHA256 es la
// del documento JSON; Definicion es la huella binaria propia del dominio.
type Configuracion struct {
	HuellaArchivoSHA256 string
	Definicion          domain.ReferenciaDefinicionSeguimiento
}

type documento struct {
	Esquema     string                                  `json:"esquema"`
	Publicacion domain.PublicacionDefinicionSeguimiento `json:"publicacion"`
}

// Fuente conserva una definición ya restaurada, sin entrada ni estado mutable.
type Fuente struct{ definicion domain.DefinicionSeguimiento }

// Nueva valida primero los bytes exactos del archivo y después restaura la
// publicación mediante el dominio, que verifica su huella binaria canónica.
func Nueva(contenido []byte, configuracion Configuracion) (*Fuente, error) {
	if len(contenido) == 0 || len(contenido) > maximoBytesDocumento || !patronSHA256HexLower.MatchString(configuracion.HuellaArchivoSHA256) || configuracion.Definicion.Validar() != nil {
		return nil, ErrFuenteInvalida
	}
	suma := sha256.Sum256(contenido)
	if subtle.ConstantTimeCompare([]byte(hex.EncodeToString(suma[:])), []byte(configuracion.HuellaArchivoSHA256)) != 1 {
		return nil, ErrFuenteInvalida
	}
	if err := validarJSONCerrado(contenido); err != nil {
		return nil, ErrFuenteInvalida
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var doc documento
	if err := decodificador.Decode(&doc); err != nil || exigirFin(decodificador) != nil || doc.Esquema != esquemaDocumento {
		return nil, ErrFuenteInvalida
	}
	definicion, err := domain.RestaurarDefinicionSeguimiento(doc.Publicacion)
	if err != nil || !definicion.Referencia().Coincide(configuracion.Definicion) {
		return nil, ErrFuenteInvalida
	}
	return &Fuente{definicion: definicion}, nil
}

// Consultar exige la terna completa y un instante UTC explícito; nunca elige
// versiones ni publica una definición. Devuelve una restauración defensiva.
func (f *Fuente) Consultar(ctx context.Context, esperada domain.ReferenciaDefinicionSeguimiento, ahora time.Time) (domain.DefinicionSeguimiento, error) {
	if f == nil || ctx == nil || esperada.Validar() != nil || ahora.IsZero() || ahora.Location() != time.UTC {
		return domain.DefinicionSeguimiento{}, ErrNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return domain.DefinicionSeguimiento{}, err
	}
	if !f.definicion.Referencia().Coincide(esperada) || !f.definicion.VigenteEn(ahora) {
		return domain.DefinicionSeguimiento{}, ErrNoDisponible
	}
	resultado, err := domain.RestaurarDefinicionSeguimiento(f.definicion.Publicacion())
	if err != nil {
		return domain.DefinicionSeguimiento{}, ErrNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return domain.DefinicionSeguimiento{}, err
	}
	return resultado, nil
}

func exigirFin(d *json.Decoder) error {
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("json adicional")
	}
	return nil
}
func validarJSONCerrado(contenido []byte) error {
	d := json.NewDecoder(bytes.NewReader(contenido))
	d.UseNumber()
	if err := validarValor(d, 0); err != nil || exigirFin(d) != nil {
		return errors.New("json inválido")
	}
	var raiz map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &raiz); err != nil || !clavesExactas(raiz, nil, "esquema", "publicacion") {
		return errors.New("raíz inválida")
	}
	return validarPublicacion(raiz["publicacion"])
}
func validarPublicacion(valor json.RawMessage) error {
	publicacion, err := objeto(valor)
	if err != nil || !clavesExactas(publicacion, nil, "referencia", "version", "huella_sha256", "canon", "publicado_en", "vigencia", "estado_inicial", "prohibe_ciclos_silenciosos", "estados", "motivos", "transiciones") {
		return errors.New("publicación inválida")
	}
	canon, err := objeto(publicacion["canon"])
	if err != nil || !clavesExactas(canon, nil, "dominio", "version_esquema", "algoritmo") {
		return errors.New("canon inválido")
	}
	vigencia, err := objeto(publicacion["vigencia"])
	if err != nil || !clavesExactas(vigencia, nil, "desde", "hasta") {
		return errors.New("vigencia inválida")
	}
	estados, err := arreglo(publicacion["estados"])
	if err != nil || len(estados) < 2 || len(estados) > 128 {
		return errors.New("estados inválidos")
	}
	for _, valor := range estados {
		estado, err := objeto(valor)
		if err != nil || !clavesExactas(estado, nil, "clave", "final") {
			return errors.New("estado inválido")
		}
	}
	transiciones, err := arreglo(publicacion["transiciones"])
	if err != nil || len(transiciones) == 0 || len(transiciones) > 512 {
		return errors.New("transiciones inválidas")
	}
	for _, valor := range transiciones {
		if err := validarTransicion(valor); err != nil {
			return err
		}
	}
	return nil
}
func validarTransicion(valor json.RawMessage) error {
	t, err := objeto(valor)
	if err != nil || !clavesExactas(t, []string{"calendario"}, "clave", "origen", "destino", "clase", "motivos_permitidos", "motivo_obligatorio", "documentos", "requiere_periodo", "efecto_periodo", "exige_actor_distinto") {
		return errors.New("transición inválida")
	}
	documentos, err := arreglo(t["documentos"])
	if err != nil || len(documentos) > 32 {
		return errors.New("documentos inválidos")
	}
	for _, valor := range documentos {
		documento, err := objeto(valor)
		if err != nil || !clavesExactas(documento, nil, "tipo_clave", "obligatorio") {
			return errors.New("documento inválido")
		}
	}
	if calendario, existe := t["calendario"]; existe {
		c, err := objeto(calendario)
		if err != nil || !clavesExactas(c, nil, "ambitos_permitidos", "resultados_permitidos") {
			return errors.New("calendario inválido")
		}
	}
	return nil
}
func objeto(valor json.RawMessage) (map[string]json.RawMessage, error) {
	var resultado map[string]json.RawMessage
	if len(valor) == 0 || bytes.Equal(bytes.TrimSpace(valor), []byte("null")) || json.Unmarshal(valor, &resultado) != nil || resultado == nil {
		return nil, errors.New("objeto")
	}
	return resultado, nil
}
func arreglo(valor json.RawMessage) ([]json.RawMessage, error) {
	var resultado []json.RawMessage
	if len(valor) == 0 || bytes.Equal(bytes.TrimSpace(valor), []byte("null")) || json.Unmarshal(valor, &resultado) != nil || resultado == nil {
		return nil, errors.New("arreglo")
	}
	return resultado, nil
}
func clavesExactas(campos map[string]json.RawMessage, opcionales []string, requeridas ...string) bool {
	permitidas := map[string]bool{}
	for _, clave := range requeridas {
		if campos[clave] == nil || bytes.Equal(bytes.TrimSpace(campos[clave]), []byte("null")) {
			return false
		}
		permitidas[clave] = true
	}
	for _, clave := range opcionales {
		permitidas[clave] = true
	}
	for clave, valor := range campos {
		if !permitidas[clave] || bytes.Equal(bytes.TrimSpace(valor), []byte("null")) {
			return false
		}
	}
	return true
}
func validarValor(d *json.Decoder, profundidad int) error {
	if profundidad > maximaProfundidadJSON {
		return errors.New("json profundo")
	}
	token, err := d.Token()
	if err != nil {
		return err
	}
	if numero, ok := token.(json.Number); ok {
		entero, err := strconv.ParseUint(string(numero), 10, 64)
		if err != nil || entero > maximoEnteroJSON {
			return errors.New("entero json")
		}
		return nil
	}
	delimitador, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	switch delimitador {
	case '{':
		claves := map[string]struct{}{}
		for d.More() {
			tokenClave, err := d.Token()
			clave, ok := tokenClave.(string)
			if err != nil || !ok {
				return errors.New("clave json")
			}
			if _, duplicada := claves[clave]; duplicada {
				return errors.New("clave duplicada")
			}
			claves[clave] = struct{}{}
			if err := validarValor(d, profundidad+1); err != nil {
				return err
			}
		}
		_, err = d.Token()
		return err
	case '[':
		for d.More() {
			if err := validarValor(d, profundidad+1); err != nil {
				return err
			}
		}
		_, err = d.Token()
		return err
	default:
		return errors.New("delimitador json")
	}
}
