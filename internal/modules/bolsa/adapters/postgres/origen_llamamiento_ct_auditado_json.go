package postgres

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// B7 conserva estos máximos en la tabla de constituciones. No se impone
// otra cota al sobre, que también contiene el orden completo.
const maximoBolsaCanonicaOrigenCT = 8388608
const maximoInstantaneaCanonicaOrigenCT = 33554432

var errCanonicoOrigenCTAuditado = errors.New("bolsa: canonico de origen CT auditado invalido")

// Los bytea que PostgreSQL convierte con to_jsonb salen como "\\x" + hex.
// Las reservas antiguas fueron serializadas por Go como base64. Ambos casos
// deben recuperar los bytes originales antes del cotejo de huellas.
type bolsaCanonicaOrigenCTAuditado []byte
type instantaneaCanonicaOrigenCTAuditado []byte

func (c *bolsaCanonicaOrigenCTAuditado) UnmarshalJSON(raw []byte) error {
	contenido, err := decodificarCanonicoOrigenCTAuditado(raw, maximoBolsaCanonicaOrigenCT)
	if err != nil {
		return err
	}
	*c = contenido
	return nil
}

func (c *instantaneaCanonicaOrigenCTAuditado) UnmarshalJSON(raw []byte) error {
	contenido, err := decodificarCanonicoOrigenCTAuditado(raw, maximoInstantaneaCanonicaOrigenCT)
	if err != nil {
		return err
	}
	*c = contenido
	return nil
}

func longitudCodificadaCanonicoOrigenCTValida(longitud, maximo int, hexPostgreSQL bool) bool {
	if hexPostgreSQL {
		return longitud >= 4 && longitud%2 == 0 && longitud <= 2*maximo
	}
	return longitud >= 4 && longitud%4 == 0 && longitud <= base64.StdEncoding.EncodedLen(maximo)
}

func decodificarCanonicoOrigenCTAuditado(raw []byte, maximo int) ([]byte, error) {
	if len(raw) < 6 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return nil, errCanonicoOrigenCTAuditado
	}
	var contenido []byte
	if bytes.HasPrefix(raw, []byte{'"', '\\', '\\', 'x'}) {
		codificado := raw[4 : len(raw)-1]
		if !longitudCodificadaCanonicoOrigenCTValida(len(codificado), maximo, true) {
			return nil, errCanonicoOrigenCTAuditado
		}
		for _, b := range codificado {
			if !(b >= '0' && b <= '9' || b >= 'a' && b <= 'f') {
				return nil, errCanonicoOrigenCTAuditado
			}
		}
		contenido = make([]byte, len(codificado)/2)
		if _, err := hex.Decode(contenido, codificado); err != nil {
			return nil, errors.Join(errCanonicoOrigenCTAuditado, err)
		}
	} else {
		codificado := raw[1 : len(raw)-1]
		if !longitudCodificadaCanonicoOrigenCTValida(len(codificado), maximo, false) {
			return nil, errCanonicoOrigenCTAuditado
		}
		for _, b := range codificado {
			if !(b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' ||
				b >= '0' && b <= '9' || b == '+' || b == '/' || b == '=') {
				return nil, errCanonicoOrigenCTAuditado
			}
		}
		contenido = make([]byte, base64.StdEncoding.DecodedLen(len(codificado)))
		n, err := base64.StdEncoding.Strict().Decode(contenido, codificado)
		if err != nil {
			return nil, errors.Join(errCanonicoOrigenCTAuditado, err)
		}
		contenido = contenido[:n]
	}
	if len(contenido) < 2 || len(contenido) > maximo || !json.Valid(contenido) {
		return nil, errCanonicoOrigenCTAuditado
	}
	return contenido, nil
}

type origenCTAuditadoCanonicosJSON struct {
	ports.OrigenLlamamientoCT
	BolsaCanonica       bolsaCanonicaOrigenCTAuditado       `json:"bolsa_canonica"`
	InstantaneaCanonica instantaneaCanonicaOrigenCTAuditado `json:"instantanea_canonica"`
}

type respuestaOrigenCTAuditado struct {
	Origen  ports.OrigenLlamamientoCT
	Reserva *ports.ReservaOrigenCT
	Acuse   acuseOrigenCTAuditado
}

type acuseOrigenCTAuditado struct {
	AuditoriaRef   string    `json:"auditoria_ref"`
	Secuencia      int64     `json:"secuencia"`
	HuellaSHA256   string    `json:"huella_sha256"`
	CorrelacionRef string    `json:"correlacion_ref"`
	RegistradaEn   time.Time `json:"registrada_en"`
}

func decodificarRespuestaOrigenCTAuditado(raw []byte) (respuestaOrigenCTAuditado, error) {
	vacia := respuestaOrigenCTAuditado{}
	if len(raw) == 0 {
		return vacia, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	var contenido struct {
		Origen  origenCTAuditadoCanonicosJSON `json:"origen"`
		Reserva *ports.ReservaOrigenCT        `json:"reserva"`
		Acuse   acuseOrigenCTAuditado         `json:"acuse"`
	}
	if err := json.Unmarshal(raw, &contenido); err != nil {
		return vacia, errors.Join(ports.ErrOrigenLlamamientoCTNoDisponible, err)
	}
	if len(contenido.Origen.BolsaCanonica) == 0 || len(contenido.Origen.InstantaneaCanonica) == 0 {
		return vacia, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	origen := contenido.Origen.OrigenLlamamientoCT
	origen.BolsaCanonica = []byte(contenido.Origen.BolsaCanonica)
	origen.InstantaneaCanonica = []byte(contenido.Origen.InstantaneaCanonica)
	return respuestaOrigenCTAuditado{Origen: origen, Reserva: contenido.Reserva, Acuse: contenido.Acuse}, nil
}
