package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrRecuperacionCompetenciaFirmanteHistoricaV1 = domain.ErrRecuperacionCompetenciaFirmanteHistoricaV1

// La referencia selecciona una entrada previamente registrada por el
// propietario del efecto. Actor, resultado y vinculo proceden de identidad
// central; RecursoActual lo resuelve el servidor para el PDP del consultante.
// Accion y finalidad son de esta lectura, no las del firmante historico.
type SolicitudRecuperacionCompetenciaFirmanteHistoricaV1 = domain.SolicitudRecuperacionCompetenciaFirmanteHistoricaV1

// La implementacion obtiene el selector del efecto durable identificado por
// RegistroRef y comprueba la version/huella del DescriptorLectura publicado;
// no confia en los valores de la solicitud aislada. Consume PDP V3 vigente
// para RecursoActual, Accion y
// Finalidad del consultante y audita antes de devolver bytes o metadatos.
// Nunca sustituye ese recurso por el de competencia historica. No emite HTTP.
type LectorCompetenciaFirmanteHistoricaV1 interface {
	RecuperarCompetenciaFirmanteHistoricaV1(context.Context, SolicitudRecuperacionCompetenciaFirmanteHistoricaV1) (domain.CanonCompetenciaFirmanteHistoricaV1, error)
}
