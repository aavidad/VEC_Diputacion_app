package auditoria

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
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

// errRegistroEslabonV5 indica que un registro no se pudo leer para cotejar
// su eslabón; el verificador lo rechaza en lugar de omitir el cotejo.
var errRegistroEslabonV5 = errors.New("vec auditoria: registro no legible para el eslabon v5")

// registradaEnDelAsiento devuelve la fecha del asiento cuando su proyección
// la incluye (casi todos los tipos), para cotejarla con la del eslabón.
// presente=false sin error: el tipo no exporta la fecha.
func registradaEnDelAsiento(r RegistroMixtoV2) (fecha string, presente bool, err error) {
	copia := r
	copia.Eslabon = nil
	b, err := json.Marshal(copia)
	if err != nil {
		return "", false, errors.Join(errRegistroEslabonV5, err)
	}
	var objeto map[string]json.RawMessage
	if err = json.Unmarshal(b, &objeto); err != nil {
		return "", false, errors.Join(errRegistroEslabonV5, err)
	}
	for clave, crudo := range objeto {
		if clave == "tipo_registro" || len(crudo) == 0 || crudo[0] != '{' {
			continue
		}
		var campos map[string]json.RawMessage
		if err = json.Unmarshal(crudo, &campos); err != nil {
			return "", false, errors.Join(errRegistroEslabonV5, err)
		}
		crudoFecha, existe := campos["registrada_en"]
		if !existe {
			continue
		}
		if err = json.Unmarshal(crudoFecha, &fecha); err != nil {
			return "", false, errors.Join(errRegistroEslabonV5, err)
		}
		if fecha != "" {
			return fecha, true, nil
		}
	}
	return "", false, nil
}

// previaCapturaPeriodicaV5 devuelve la cabeza sellada que declaró una captura
// del sello periódico escrita tras el corte. es=false sin error: el registro
// no es una captura posterior al corte.
func previaCapturaPeriodicaV5(r RegistroMixtoV2) (previa uint64, cabeza string, es bool, err error) {
	if r.Periodica == nil || r.Periodica.AnteriorSHA256 != MarcadorSinAnteriorV5 || r.Periodica.Accion != "capturar_sello_periodico_v1" ||
		r.Periodica.MotivoRef != "captura_registrada" {
		return 0, "", false, nil
	}
	detalle, err := decodificarDetallePeriodica(r.Periodica.DetalleCanonicoBase64)
	if err != nil {
		return 0, "", false, errors.Join(errRegistroEslabonV5, err)
	}
	var d struct {
		PreviaSecuencia json.Number `json:"previa_secuencia"`
		PreviaCabeza    string      `json:"previa_cabeza_sha256"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(detalle)))
	decoder.UseNumber()
	if err = decoder.Decode(&d); err != nil {
		return 0, "", false, errors.Join(errRegistroEslabonV5, err)
	}
	if previa, err = strconv.ParseUint(d.PreviaSecuencia.String(), 10, 64); err != nil {
		return 0, "", false, errors.Join(errRegistroEslabonV5, err)
	}
	return previa, d.PreviaCabeza, true, nil
}

func decodificarDetallePeriodica(transporte string) ([]byte, error) {
	normalizado := strings.ReplaceAll(strings.ReplaceAll(transporte, "\r", ""), "\n", "")
	return base64.StdEncoding.Strict().DecodeString(normalizado)
}
