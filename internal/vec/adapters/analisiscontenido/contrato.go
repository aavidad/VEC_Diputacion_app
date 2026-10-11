package analisiscontenido

import (
	"encoding/hex"
	"errors"
	"strings"
	"unicode"

	"vec-diputacion-granada/internal/vec/ports"
)

type solicitudRemota struct {
	Objeto            ports.ReferenciaObjetoAlmacen `json:"objeto"`
	ConectorAlmacenID string                        `json:"conector_almacen_id"`
	Zona              ports.ZonaAlmacen             `json:"zona"`
	MIME              string                        `json:"mime"`
	Tamano            int64                         `json:"tamano"`
	HuellaSHA256      string                        `json:"huella_sha256"`
	CorrelacionRef    string                        `json:"correlacion_ref"`
	AutorizacionRef   string                        `json:"autorizacion_ref"`
	Finalidad         string                        `json:"finalidad"`
	Clasificacion     string                        `json:"clasificacion"`
}

func (s solicitudRemota) validar() error {
	if s.Objeto.Validar() != nil || !referenciaSegura(s.ConectorAlmacenID, 128) || s.Zona != ports.ZonaAlmacenCuarentena || s.MIME != "application/pdf" ||
		s.Tamano < 1 || s.Tamano > tamanoMaximoPDF || !esSHA256(s.HuellaSHA256) ||
		!referenciaSegura(s.CorrelacionRef, 512) || !referenciaSegura(s.AutorizacionRef, 512) ||
		!textoSeguro(s.Finalidad, 1024) || !textoSeguro(s.Clasificacion, 256) {
		return errors.New("metadatos_invalidos")
	}
	return nil
}

func esSHA256(s string) bool {
	if len(s) != 64 || s != strings.ToLower(s) {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func referenciaSegura(s string, max int) bool {
	if len(s) < 1 || len(s) > max {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' || r == ':') {
			return false
		}
	}
	return true
}

func textoSeguro(s string, max int) bool {
	if len(s) < 1 || len(s) > max {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
