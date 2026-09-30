package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

const tipoAvisoExternoLlamamiento = "vec.bolsa.aviso-llamamiento.v1"

var patronProductorAvisosExternos = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:._-]{7,191}$`)

type configuracionAvisosExternos struct {
	fuente    ports.FuenteDestinatarioExternoParticipacion
	productor string
}

// EstablecerAvisosExternos se invoca durante el montaje del proceso separado.
// El productor procede de configuración, y el vínculo de la autoridad externa.
func (s *ServicioEmisionLlamamiento) EstablecerAvisosExternos(f ports.FuenteDestinatarioExternoParticipacion, productor string) error {
	if s == nil || nulo(f) || !patronProductorAvisosExternos.MatchString(productor) || s.avisador == nil || s.avisador.conMisCorreos() {
		return ports.ErrEmisionLlamamientoNoDisponible
	}
	s.avisosExternos = &configuracionAvisosExternos{fuente: f, productor: productor}
	return nil
}

func (s *ServicioEmisionLlamamiento) prepararAvisosExternos(ctx context.Context, q ports.SolicitudEmitirLlamamiento, llamamiento string, ahora time.Time) ([]ports.EventoAvisoExterno, map[string]bool, error) {
	externos := map[string]bool{}
	if s.avisosExternos == nil {
		return nil, externos, nil
	}
	eventos := []ports.EventoAvisoExterno{}
	correlacion, err := q.Correlacion.ValorCanonico()
	if err != nil {
		return nil, nil, ports.ErrEmisionLlamamientoInvalida
	}
	// El replay conserva material y fecha originales, nunca acuña otro aviso.
	previa, recErr := s.repositorio.Recuperar(ctx, q.BolsaRef, q.ClaveIdempotencia)
	replayExterno := recErr == nil && mismaSolicitudEmision(previa, q) && len(previa.AvisosExternos) > 0
	if replayExterno {
		for _, pendiente := range previa.AvisosExternos {
			eventos = append(eventos, pendiente.Evento)
		}
	}
	for _, participacion := range q.Participaciones {
		candidato, err := s.avisosExternos.fuente.DestinatarioExterno(ctx, q.BolsaRef, participacion)
		if err != nil {
			return nil, nil, ports.ErrEmisionLlamamientoNoDisponible
		}
		if candidato == "" {
			// La ausencia no acredita población interna: nunca habilita fallback.
			return nil, nil, ports.ErrEmisionLlamamientoNoDisponible
		}
		externos[participacion] = true
		if replayExterno {
			continue
		}
		huella := sha256.Sum256([]byte(llamamiento + "\x1f" + participacion))
		eventos = append(eventos, ports.EventoAvisoExterno{
			EventoRef: "evento_aviso:" + hex.EncodeToString(huella[:]), ProductorRef: s.avisosExternos.productor,
			TipoVersionado: tipoAvisoExternoLlamamiento, OcurridoEn: ahora.UTC().Format("2006-01-02T15:04:05.000000Z"),
			CorrelacionRef: correlacion, DestinatarioExternoRef: candidato, ComunicacionRef: llamamiento,
			PlantillaRef: q.Configuracion.PlantillaVersion, PlantillaVersion: q.Configuracion.PlantillaVersion,
		})
	}
	return eventos, externos, nil
}
