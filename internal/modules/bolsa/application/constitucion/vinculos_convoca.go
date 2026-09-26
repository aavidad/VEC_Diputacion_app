package constitucion

import (
	"context"
	"errors"

	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// Recuperador se conserva para la lectura histórica de vínculos CONVOCA.
type Recuperador interface {
	RecuperarLote(context.Context, string, string) (importacion.LoteValidado, importacionapp.EstadoImportacion, bool, error)
}

// FilaVinculoCandidato transporta el número del staging y su referencia opaca.
type FilaVinculoCandidato struct {
	FilaNumero   int    `json:"fila_numero"`
	CandidatoRef string `json:"candidato_ref"`
}

// DerivarFilasVinculo conserva el contrato de recuperación histórica de
// CONVOCA sin participar en el orden de las listas nativas.
func DerivarFilasVinculo(lote importacion.LoteValidado, derivador DerivadorCandidato) ([]FilaVinculoCandidato, error) {
	if derivador == nil || lote.Validar() != nil || lote.Acta.Esquema != importacion.EsquemaResumenPersona {
		return nil, ports.ErrConstitucionBolsaInvalida
	}
	filas := make([]FilaVinculoCandidato, 0, len(lote.Aceptadas))
	for _, fila := range lote.Aceptadas {
		if fila.Resumen == nil || fila.Numero <= 0 {
			return nil, ports.ErrConstitucionBolsaInvalida
		}
		ref, err := derivador.CandidatoRef(fila.Identidad)
		if err != nil {
			return nil, errors.Join(ports.ErrConstitucionBolsaInvalida, err)
		}
		filas = append(filas, FilaVinculoCandidato{FilaNumero: fila.Numero, CandidatoRef: ref})
	}
	if len(filas) == 0 {
		return nil, ErrActaSinFilasAceptadas
	}
	return filas, nil
}
