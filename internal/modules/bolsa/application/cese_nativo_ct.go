package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

// ServicioCeseNativoCT reutiliza la autoridad ACT V3 de la participación.
// El repositorio hace el cotejo definitivo CT/Personal/Bolsa y el efecto B45.
type ServicioCeseNativoCT struct {
	situacion   *ServicioSituacionParticipacion
	repositorio ports.RepositorioCeseNativoCT
}

func NuevoServicioCeseNativoCT(s *ServicioSituacionParticipacion, r ports.RepositorioCeseNativoCT) (*ServicioCeseNativoCT, error) {
	if s == nil || nuloPersonaAceptacionCT(r) {
		return nil, ports.ErrCeseNativoCTNoDisponible
	}
	return &ServicioCeseNativoCT{situacion: s, repositorio: r}, nil
}

func (s *ServicioCeseNativoCT) AplicarCeseNativoCT(ctx context.Context, q ports.SolicitudCeseNativoCT) (ports.ReciboCeseNativoCT, error) {
	var cero ports.ReciboCeseNativoCT
	if ctx == nil || s == nil || s.situacion == nil || nuloPersonaAceptacionCT(s.repositorio) {
		return cero, ports.ErrCeseNativoCTNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if !referenciaAceptacionNativaCT.MatchString(q.ParticipacionRef) ||
		!referenciaAceptacionNativaCT.MatchString(q.CeseCTReciboRef) ||
		!referenciaAceptacionNativaCT.MatchString(q.RelacionReciboRef) ||
		!referenciaAceptacionNativaCT.MatchString(q.OcupacionReciboRef) ||
		q.RelacionReciboRef == q.OcupacionReciboRef ||
		q.ResultadoContexto.Validar() != nil || q.Vinculo.ValidarPara(q.ResultadoContexto) != nil ||
		q.Correlacion.Validar() != nil || !core.ReferenciaMotivoAutorizacionV2Valida(q.MotivoAutorizacion) {
		return cero, ports.ErrCeseNativoCTInvalido
	}
	if _, err := PrepararConsultaAceptacionNativaCT(ports.SolicitudConsultaAceptacionNativaCT{
		Selector: q.Selector, ActorConfiable: ports.ActorConfiablePersonaAceptacionCT{
			Vinculo: q.Vinculo, Resultado: q.ResultadoContexto,
		},
	}, false); err != nil {
		return cero, ports.ErrCeseNativoCTInvalido
	}
	// La clave de situación sólo identifica el acto CT; el recibo B45 original
	// lo deriva SQL del evento publicado por CT. No se crea una situación manual.
	h := sha256.Sum256([]byte("vec.bolsa.cese-nativo-ct.idempotencia.v1\x00" + q.Selector.ExpedienteRef + "\x00" + q.CeseCTReciboRef))
	clave := "cese-ct-b2:" + hex.EncodeToString(h[:])
	autorizacion := ports.SolicitudCambiarSituacionParticipacion{
		Vinculo: q.Vinculo, ResultadoContexto: q.ResultadoContexto,
		ParticipacionRef: q.ParticipacionRef, BolsaRef: q.Selector.BolsaRef,
		Destino: "disponible_desde", Motivo: "cese_ct_b2", ClaveIdempotencia: clave,
		Correlacion: q.Correlacion, MotivoAutorizacion: q.MotivoAutorizacion,
	}
	if autorizacion.Validar() != nil {
		return cero, ports.ErrCeseNativoCTInvalido
	}
	_, _, _, material, err := s.situacion.autorizarOperacion(ctx, autorizacion)
	if err != nil {
		if errors.Is(err, core.ErrAutorizacionDenegada) || errors.Is(err, core.ErrPermissionDenied) {
			return cero, ports.ErrCeseNativoCTDenegado
		}
		return cero, ports.ErrCeseNativoCTNoDisponible
	}
	o := ports.OrdenCeseNativoCT{Solicitud: q, ActorRef: q.ResultadoContexto.Contexto.PersonaRef, Material: material}
	r, err := s.repositorio.AplicarCeseNativoCT(ctx, o)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, err
	}
	if r.ParticipacionRef != q.ParticipacionRef || r.BolsaRef != q.Selector.BolsaRef ||
		!strings.HasPrefix(r.ReciboRef, "recibo:bolsa:cese:") ||
		!strings.HasPrefix(r.AuditoriaRef, "auditoria:bolsa:cese:") ||
		!strings.HasPrefix(r.EventoRef, "evento:ct:contrato-bolsa:") ||
		r.PoliticaVersion == 0 || !fechaCivilCeseNativoValida(r.FechaEfecto) ||
		!fechaCivilCeseNativoValida(r.DisponibleDesde) || r.DisponibleDesde < r.FechaEfecto {
		return cero, ports.ErrCeseNativoCTNoDisponible
	}
	return r, nil
}

func fechaCivilCeseNativoValida(s string) bool {
	t, err := time.Parse(time.DateOnly, s)
	return err == nil && t.Format(time.DateOnly) == s
}

var _ ports.AplicadorCeseNativoCT = (*ServicioCeseNativoCT)(nil)
