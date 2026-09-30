package bootstrap

import core "vec-diputacion-granada/internal/vec/domain"

// La selección solo restringe los tres catálogos de la autoridad de Bolsa.
// Conserva su orden, entradas y huellas; no crea permisos ni motivos nuevos.
func seleccionarMotivosProvisionCandidatoExterno(autoridad []core.ReferenciaEntradaCatalogo, solicitados []string) ([]core.ReferenciaEntradaCatalogo, error) {
	if solicitados == nil {
		return append([]core.ReferenciaEntradaCatalogo(nil), autoridad...), nil
	}
	if len(solicitados) == 0 || len(solicitados) > len(autoridad) {
		return nil, ErrProvisionCandidatoExterno
	}
	seleccion := make(map[string]bool, len(solicitados))
	for _, catalogo := range solicitados {
		if seleccion[catalogo] {
			return nil, ErrProvisionCandidatoExterno
		}
		seleccion[catalogo] = true
	}
	var resultado []core.ReferenciaEntradaCatalogo
	for _, motivo := range autoridad {
		if seleccion[motivo.CatalogoID] {
			resultado = append(resultado, motivo)
		}
	}
	if len(resultado) != len(solicitados) {
		return nil, ErrProvisionCandidatoExterno
	}
	return resultado, nil
}
