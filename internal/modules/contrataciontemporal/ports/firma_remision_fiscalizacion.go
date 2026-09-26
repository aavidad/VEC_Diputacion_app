package ports

import (
	"context"
	"errors"
)

// ErrFirmaRemisionPendiente indica que el circuito de firma exige firmar un
// paso antes de remitir el expediente a Intervención y ese paso no consta
// firmado en el registro de firmas (duda 4 de RRHH). No es una denegación de
// permisos: el expediente no está listo para fiscalizarse.
var ErrFirmaRemisionPendiente = errors.New("contratacion temporal: falta la firma que habilita la remision a Intervencion")

// ComprobadorFirmaRemisionFiscalizacion responde si el expediente puede
// remitirse a Intervención según el circuito de firma vigente. Devuelve nil
// cuando ningún paso del circuito habilita la remisión o cuando todos los que
// la habilitan constan firmados; ErrFirmaRemisionPendiente si falta alguno; y
// cualquier otro error si no puede comprobarlo, que nunca equivale a firmado.
type ComprobadorFirmaRemisionFiscalizacion interface {
	ComprobarFirmaRemision(ctx context.Context, organizacionRef, expedienteRef string) error
}
