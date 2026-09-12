package httpinterno

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const maximoCuerpoAnotacionAdministrativaBytes = 16 * 1024

var errEntradaAnotacionAdministrativaInvalida = errors.New("contratacion temporal http: entrada de anotacion administrativa invalida")

type entradaAnotacionAdministrativaJSON struct {
	ExpedienteRef     string `json:"expediente_ref"`
	VersionEsperada   uint64 `json:"version_esperada"`
	ClaveIdempotencia string `json:"clave_idempotencia"`
	Observaciones     string `json:"observaciones"`
}

func entradaAnotacionValida(e entradaAnotacionAdministrativaJSON) bool {
	return domain.ReferenciaOpacaValida(e.ExpedienteRef) && e.VersionEsperada > 0 && e.VersionEsperada < ports.MaximoEnteroSeguroOperacionAnalisis && ports.ClaveIdempotenciaValida(e.ClaveIdempotencia) &&
		ports.ValidarResultadoFiscalizacion(domain.FiscalizacionFavorableConObservaciones, e.Observaciones) == nil
}

func decodificarEntradaAnotacionAdministrativa(contenido []byte) (entradaAnotacionAdministrativaJSON, error) {
	var cero entradaAnotacionAdministrativaJSON
	if len(contenido) == 0 || len(contenido) > maximoCuerpoAnotacionAdministrativaBytes || !utf8.Valid(contenido) {
		return cero, errEntradaAnotacionAdministrativaInvalida
	}
	d := json.NewDecoder(bytes.NewReader(contenido))
	d.UseNumber()
	inicio, err := d.Token()
	if err != nil || inicio != json.Delim('{') {
		return cero, errEntradaAnotacionAdministrativaInvalida
	}
	vistas := map[string]bool{}
	for d.More() {
		k, err := d.Token()
		clave, ok := k.(string)
		if err != nil || !ok || vistas[clave] || (clave != "expediente_ref" && clave != "version_esperada" && clave != "clave_idempotencia" && clave != "observaciones") {
			return cero, errEntradaAnotacionAdministrativaInvalida
		}
		vistas[clave] = true
		if _, err := d.Token(); err != nil {
			return cero, errEntradaAnotacionAdministrativaInvalida
		}
	}
	if fin, err := d.Token(); err != nil || fin != json.Delim('}') || len(vistas) != 4 {
		return cero, errEntradaAnotacionAdministrativaInvalida
	}
	if _, err := d.Token(); err != io.EOF {
		return cero, errEntradaAnotacionAdministrativaInvalida
	}
	d = json.NewDecoder(bytes.NewReader(contenido))
	d.DisallowUnknownFields()
	var entrada entradaAnotacionAdministrativaJSON
	if d.Decode(&entrada) != nil || d.Decode(&struct{}{}) != io.EOF || !entradaAnotacionValida(entrada) {
		return cero, errEntradaAnotacionAdministrativaInvalida
	}
	return entrada, nil
}
