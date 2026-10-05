package main

import (
	"vec-diputacion-granada/internal/vec/auditoria"
	"vec-diputacion-granada/internal/vec/domain"
)

// La firma del checkpoint se comprueba antes de esta lectura. El parser común
// coteja el documento con la cobertura firmada, sin acreditar su extracción.
func verificarCadenaCheckpoint(b []byte, cobertura domain.CoberturaCheckpoint, maxBytes int64, maxRegistros uint64) (auditoria.InformeVerificacion, error) {
	_, informe, err := auditoria.VerificarDocumentoExportacionAuditoria(b, cobertura, maxBytes, maxRegistros)
	if err != nil {
		return informe, err
	}
	return informe, nil
}
