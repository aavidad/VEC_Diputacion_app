package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var _ ports.ComprobadorFirmaRemisionFiscalizacion = (*ServicioFirmaDocumento)(nil)

// ComprobarFirmaRemision aplica la habilitación «remision_intervencion» del
// circuito de firma: si algún paso la tiene, debe constar firmado en la
// historia registrada del expediente. Qué paso la habilita lo decide el
// catálogo; sin ningún paso marcado no se exige nada.
func (s *ServicioFirmaDocumento) ComprobarFirmaRemision(ctx context.Context, organizacionRef, expedienteRef string) error {
	estado, err := s.Consultar(ctx, organizacionRef, expedienteRef)
	if err != nil {
		return err
	}
	if len(domain.PasosRemisionSinFirmar(estado.Circuito, estado.Documentos)) > 0 {
		return ports.ErrFirmaRemisionPendiente
	}
	return nil
}
