package preparacionliquidacion

import (
	"reflect"

	"vec-diputacion-granada/internal/modules/dietas/domain"
)

// Recuperar reconstruye una preparación local mediante las reglas existentes.
// Las huellas comprueban coherencia; no acreditan autenticidad ni aprobación.
func Recuperar(s domain.InstantaneaLiquidacionPropuesta) (*domain.PreparacionLiquidacion, error) {
	if len(s.Lineas) < 1 || len(s.Lineas) > 102 || len(s.Lineas) != len(s.Documento.Lineas) {
		return nil, domain.ErrPreparacionLiquidacion
	}
	revisiones := make([]domain.RevisionLineaLiquidacion, len(s.Lineas))
	for i, linea := range s.Lineas {
		revisiones[i] = domain.RevisionLineaLiquidacion{
			Indice: linea.Indice, ReglaRef: linea.ReglaRef,
			ReconocidoPropuestoCentimos: linea.ReconocidoPropuestoCentimos,
			MotivoCodigo:                linea.MotivoCodigo,
		}
	}
	p, err := Preparar(Entrada{
		Esquema: s.Esquema, ComisionRef: s.ComisionRef, ComisionVersion: s.ComisionVersion,
		DocumentoSHA256: s.DocumentoSHA256, Documento: s.Documento,
		Catalogo: s.Catalogo, Revisiones: revisiones,
	})
	if err != nil {
		return nil, err
	}
	// Se cotejan también los datos derivados y los metadatos; una huella
	// recalculada sobre una importación incoherente no evita esta comprobación.
	if !reflect.DeepEqual(p.Instantanea(), s) {
		return nil, domain.ErrPreparacionLiquidacion
	}
	return p, nil
}
