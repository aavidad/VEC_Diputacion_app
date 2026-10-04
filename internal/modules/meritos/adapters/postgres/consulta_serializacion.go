package postgres

import (
	"bytes"
	"encoding/json"
	"io"

	"vec-diputacion-granada/internal/modules/meritos/application"
	"vec-diputacion-granada/internal/modules/meritos/ports"
)

func serializarConsultaPropia(orden ports.OrdenConsultaPropia) ([]byte, error) {
	if application.ValidarOrdenConsultaPropia(orden) != nil {
		return nil, ports.ErrConsultaNoDisponible
	}
	return bytes.Clone(orden.SelectorCanonico), nil
}

func leerResultadoConsultaPropia(raw []byte) (ports.ResultadoConsultaPropia, error) {
	if len(raw) == 0 || len(raw) > 65536 {
		return ports.ResultadoConsultaPropia{}, ports.ErrConsultaNoDisponible
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var resultado ports.ResultadoConsultaPropia
	if decoder.Decode(&resultado) != nil {
		return ports.ResultadoConsultaPropia{}, ports.ErrConsultaNoDisponible
	}
	if decoder.Decode(new(any)) != io.EOF {
		return ports.ResultadoConsultaPropia{}, ports.ErrConsultaNoDisponible
	}
	return resultado, nil
}
