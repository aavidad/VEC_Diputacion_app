package auditoria

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// MarcadorSinAnteriorV5 ocupa anterior_sha256 en los asientos escritos tras
// el corte de AD207. El escritor ya no lee la cabeza: su huella liga contenido
// y número de orden, y el enlace con el asiento anterior lo da el eslabón.
var MarcadorSinAnteriorV5 = strings.Repeat("f", 64)

// EslabonCadenaV5 es la fila de eslabon_auditoria_v5 de un asiento sellado.
// Posicion es su lugar en la cadena; Secuencia, su número de orden en la cola.
// Las dos fechas van en UTC con microsegundos y entran en la huella.
type EslabonCadenaV5 struct {
	Posicion       uint64 `json:"posicion"`
	Secuencia      uint64 `json:"secuencia"`
	AnteriorSHA256 string `json:"anterior_sha256"`
	EslabonSHA256  string `json:"eslabon_sha256"`
	RegistradaEn   string `json:"registrada_en"`
	SelladoEn      string `json:"sellado_en"`
}

// HuellaEslabonV5 reproduce vec_autorizacion_atestada_v3.eslabon_auditoria_v5.
func HuellaEslabonV5(cadena string, posicion uint64, anterior string, secuencia uint64, auditoriaRef, tipo, huella, registradaEn, selladoEn string) string {
	h := sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.eslabon.v5", cadena, strconv.FormatUint(posicion, 10), anterior,
		strconv.FormatUint(secuencia, 10), auditoriaRef, tipo, huella, registradaEn, selladoEn))
	return hex.EncodeToString(h[:])
}

func instanteEslabonValido(v string) bool {
	t, err := time.Parse("2006-01-02T15:04:05.000000Z", v)
	return err == nil && t.UTC().Format("2006-01-02T15:04:05.000000Z") == v
}

// registradaEnDelAsiento devuelve la fecha del asiento cuando su proyección
// la incluye (casi todos los tipos), para cotejarla con la del eslabón.
func registradaEnDelAsiento(r RegistroMixtoV2) (string, bool) {
	copia := r
	copia.Eslabon = nil
	b, err := json.Marshal(copia)
	if err != nil {
		return "", false
	}
	var objeto map[string]json.RawMessage
	if json.Unmarshal(b, &objeto) != nil {
		return "", false
	}
	for clave, crudo := range objeto {
		if clave == "tipo_registro" {
			continue
		}
		var campos map[string]json.RawMessage
		if json.Unmarshal(crudo, &campos) != nil {
			continue
		}
		var fecha string
		if json.Unmarshal(campos["registrada_en"], &fecha) == nil && fecha != "" {
			return fecha, true
		}
	}
	return "", false
}

// previaCapturaPeriodicaV5 devuelve la cabeza sellada que declaró una captura
// del sello periódico escrita tras el corte.
func previaCapturaPeriodicaV5(r RegistroMixtoV2) (uint64, string, bool) {
	if r.Periodica == nil || r.Periodica.AnteriorSHA256 != MarcadorSinAnteriorV5 || r.Periodica.Accion != "capturar_sello_periodico_v1" ||
		r.Periodica.MotivoRef != "captura_registrada" {
		return 0, "", false
	}
	detalle, err := decodificarDetallePeriodica(r.Periodica.DetalleCanonicoBase64)
	if err != nil {
		return 0, "", false
	}
	var d struct {
		PreviaSecuencia json.Number `json:"previa_secuencia"`
		PreviaCabeza    string      `json:"previa_cabeza_sha256"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(detalle)))
	decoder.UseNumber()
	if decoder.Decode(&d) != nil {
		return 0, "", false
	}
	n, err := strconv.ParseUint(d.PreviaSecuencia.String(), 10, 64)
	return n, d.PreviaCabeza, err == nil
}

func decodificarDetallePeriodica(transporte string) ([]byte, error) {
	normalizado := strings.ReplaceAll(strings.ReplaceAll(transporte, "\r", ""), "\n", "")
	return base64.StdEncoding.Strict().DecodeString(normalizado)
}
