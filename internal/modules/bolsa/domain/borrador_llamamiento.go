package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

var ErrBorradorLlamamientoInvalido = errors.New("bolsa: borrador de llamamiento invalido")

const (
	EstadoBorradorLlamamientoInterno EstadoBorradorLlamamiento = "borrador_interno"
	EsquemaCrearBorradorLlamamiento                            = "vec.bolsa.llamamiento.borrador-interno.crear.v1"
)

type EstadoBorradorLlamamiento string

// ContenidoBorradorLlamamiento es deliberadamente pequeño: no decide todavía
// necesidad, candidato, elegibilidad, contacto, plazo ni resultado.
type ContenidoBorradorLlamamiento struct {
	Resumen string `json:"resumen"`
}

type BorradorLlamamiento struct {
	referencia     string
	propietarioRef string
	unidadRef      string
	ambitoRef      string
	contenido      ContenidoBorradorLlamamiento
	estado         EstadoBorradorLlamamiento
	version        uint64
}

var patronReferenciaBorradorLlamamiento = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._:/-]{2,159}$`)
var patronPersonaBorradorLlamamiento = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,128}$`)
var patronClaveBorradorLlamamiento = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$`)
var patronDatoPersonalBorradorLlamamiento = regexp.MustCompile(`(?i)(?:^|[^\p{L}])(?:dni|nie|nif|pasaporte|email|tel[eé]fono)(?:$|[^\p{L}])|@`)
var patronDatoPersonalContacto = regexp.MustCompile(`(?i)(?:^|[^\p{L}])(?:dni|nie|nif|pasaporte|email)(?:$|[^\p{L}])|@`)
var patronTelefonoConCifrasContacto = regexp.MustCompile(`(?i)(?:^|[^\p{L}])tel[eé]fono[^\p{L}\p{N}]{0,8}[+]?[0-9]`)
var patronNumeroTelefonoContacto = regexp.MustCompile(`(?:^|[^\p{L}\p{N}])(?:\+34[\s.-]*)?[6789](?:[\s.-]*[0-9]){8}(?:$|[^0-9])`)

func NuevoBorradorLlamamiento(referencia, propietarioRef, unidadRef, ambitoRef string, contenido ContenidoBorradorLlamamiento) (BorradorLlamamiento, error) {
	b := BorradorLlamamiento{referencia: referencia, propietarioRef: propietarioRef, unidadRef: unidadRef, ambitoRef: ambitoRef, contenido: contenido, estado: EstadoBorradorLlamamientoInterno, version: 1}
	if b.Validar() != nil {
		return BorradorLlamamiento{}, ErrBorradorLlamamientoInvalido
	}
	return b, nil
}

func (b BorradorLlamamiento) Validar() error {
	if !referenciaBorradorLlamamientoValida(b.referencia) || !patronPersonaBorradorLlamamiento.MatchString(b.propietarioRef) ||
		!referenciaBorradorLlamamientoValida(b.unidadRef) || !referenciaBorradorLlamamientoValida(b.ambitoRef) ||
		b.estado != EstadoBorradorLlamamientoInterno || b.version != 1 || b.contenido.validar() != nil {
		return ErrBorradorLlamamientoInvalido
	}
	return nil
}

func (c ContenidoBorradorLlamamiento) validar() error {
	if c.Resumen != strings.TrimSpace(c.Resumen) || len(c.Resumen) < 3 || len(c.Resumen) > 2000 || strings.ContainsAny(c.Resumen, "\x00\u2028\u2029") || contieneDatoPersonalEvidente(c.Resumen, false) {
		return ErrBorradorLlamamientoInvalido
	}
	return nil
}

// La nota del contacto puede describir el canal sin conservar un número.
// El borrador mantiene su regla original, más estricta.
func contieneDatoPersonalEvidente(s string, anotacionContacto bool) bool {
	if !anotacionContacto {
		return patronDatoPersonalBorradorLlamamiento.MatchString(s)
	}
	return patronDatoPersonalContacto.MatchString(s) ||
		patronTelefonoConCifrasContacto.MatchString(s) ||
		patronNumeroTelefonoContacto.MatchString(s)
}

func referenciaBorradorLlamamientoValida(valor string) bool {
	return patronReferenciaBorradorLlamamiento.MatchString(valor) && valor == strings.TrimSpace(valor) && !strings.ContainsRune(valor, '*')
}

func (b BorradorLlamamiento) Referencia() string                      { return b.referencia }
func (b BorradorLlamamiento) PropietarioRef() string                  { return b.propietarioRef }
func (b BorradorLlamamiento) UnidadRef() string                       { return b.unidadRef }
func (b BorradorLlamamiento) AmbitoRef() string                       { return b.ambitoRef }
func (b BorradorLlamamiento) Contenido() ContenidoBorradorLlamamiento { return b.contenido }
func (b BorradorLlamamiento) Estado() EstadoBorradorLlamamiento       { return b.estado }
func (b BorradorLlamamiento) Version() uint64                         { return b.version }

// HuellaComandoCrear excluye datos de transporte, identidad de sesión,
// decisión, correlación, instante y la referencia generada.
func HuellaComandoCrearBorradorLlamamiento(propietarioRef, unidadRef, ambitoRef, clave string, contenido ContenidoBorradorLlamamiento) (string, error) {
	representacion, err := RepresentacionCanonicaComandoCrearBorradorLlamamiento(propietarioRef, unidadRef, ambitoRef, clave, contenido)
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(representacion)
	return hex.EncodeToString(suma[:]), nil
}

// RepresentacionCanonicaComandoCrearBorradorLlamamiento evita que dominio y
// persistencia calculen la misma huella con serializadores diferentes.
func RepresentacionCanonicaComandoCrearBorradorLlamamiento(propietarioRef, unidadRef, ambitoRef, clave string, contenido ContenidoBorradorLlamamiento) ([]byte, error) {
	if !patronPersonaBorradorLlamamiento.MatchString(propietarioRef) || !referenciaBorradorLlamamientoValida(unidadRef) || !referenciaBorradorLlamamientoValida(ambitoRef) || !claveBorradorLlamamientoValida(clave) || contenido.validar() != nil {
		return nil, ErrBorradorLlamamientoInvalido
	}
	material := struct {
		Esquema        string                       `json:"esquema"`
		PropietarioRef string                       `json:"propietario_ref"`
		UnidadRef      string                       `json:"unidad_ref"`
		AmbitoRef      string                       `json:"ambito_ref"`
		Clave          string                       `json:"clave_idempotencia"`
		Contenido      ContenidoBorradorLlamamiento `json:"contenido"`
	}{EsquemaCrearBorradorLlamamiento, propietarioRef, unidadRef, ambitoRef, clave, contenido}
	var canon bytes.Buffer
	encoder := json.NewEncoder(&canon)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(material); err != nil {
		return nil, ErrBorradorLlamamientoInvalido
	}
	return bytes.Clone(bytes.TrimSuffix(canon.Bytes(), []byte{'\n'})), nil
}

func claveBorradorLlamamientoValida(valor string) bool {
	return patronClaveBorradorLlamamiento.MatchString(valor) && valor == strings.TrimSpace(valor)
}
