package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ServicioCeseNativoCT reutiliza la autoridad ACT V3 de la participación.
// El repositorio hace el cotejo definitivo CT/Personal/Bolsa y el efecto B45.
type ServicioCeseNativoCT struct {
	situacion   *ServicioSituacionParticipacion
	repositorio ports.RepositorioCeseNativoCT
}

func NuevoServicioCeseNativoCT(s *ServicioSituacionParticipacion, r ports.RepositorioCeseNativoCT) (*ServicioCeseNativoCT, error) {
	if s == nil || s.reloj == nil || nuloPersonaAceptacionCT(r) {
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
		!pruebaCierrePersonalCeseCTValida(q.Personal) ||
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
	material, actor, canon, err := autorizarActoNativoCT(ctx, s.situacion, q.Selector,
		q.ParticipacionRef, q.Personal.ActorRef, q.Vinculo, q.ResultadoContexto,
		q.Correlacion, q.MotivoAutorizacion, func(unidad, ambito string) any {
			return ports.MaterialActoCeseNativoCT{Esquema: ports.EsquemaActoCeseNativoCT,
				Selector: q.Selector, ParticipacionRef: q.ParticipacionRef,
				UnidadAutorizacionRef: unidad, AmbitoRef: ambito, Personal: q.Personal}
		})
	if err != nil {
		return cero, err
	}
	o := ports.OrdenCeseNativoCT{Solicitud: q, ActorRef: actor, MaterialCanonico: canon, Material: material}
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

func autorizarActoNativoCT(ctx context.Context, situacion *ServicioSituacionParticipacion,
	selector ports.SelectorAceptacionNativaCT, participacion, actorFuente string,
	vinculo core.VinculoAutenticacionActorV2, resultado core.ResultadoContextoActorRegistradoV2,
	correlacion core.ReferenciaCorrelacionAutorizacionV2, motivo core.ReferenciaEntradaCatalogo,
	construir func(string, string) any) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, string, []byte, error) {
	var cero vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if situacion == nil || construir == nil || nuloPersonaAceptacionCT(situacion.contexto) || nuloPersonaAceptacionCT(situacion.autorizador) ||
		nuloPersonaAceptacionCT(situacion.repositorio) || situacion.reloj == nil {
		return cero, "", nil, ports.ErrCeseNativoCTNoDisponible
	}
	v, err := vinculo.Datos()
	if err != nil || v.PrincipalID == "" || v.PrincipalID != actorFuente ||
		vinculo.ValidarPara(resultado) != nil {
		return cero, "", nil, ports.ErrCeseNativoCTDenegado
	}
	resuelto, err := situacion.contexto.ResolverContextoSituacionParticipacion(ctx,
		resultado.Contexto, selector.BolsaRef, participacion)
	if err != nil || resuelto.Validar() != nil {
		return cero, "", nil, ports.ErrCeseNativoCTNoDisponible
	}
	pertenece, err := situacion.repositorio.ParticipacionPerteneceABolsa(ctx,
		selector.BolsaRef, participacion)
	if err != nil {
		return cero, "", nil, ports.ErrCeseNativoCTNoDisponible
	}
	if !pertenece {
		return cero, "", nil, ports.ErrCeseNativoCTDenegado
	}
	intencion := construir(resuelto.UnidadRef, resuelto.AmbitoRef)
	canon, err := json.Marshal(intencion)
	if err != nil || len(canon) == 0 || len(canon) > 16<<10 {
		return cero, "", nil, ports.ErrCeseNativoCTInvalido
	}
	h := sha256.Sum256(canon)
	recurso := core.RecursoAutorizable{Referencia: participacion,
		ModuloID: ports.ModuloSituacionParticipacion, Tipo: ports.TipoRecursoSituacionParticipacion,
		Ambitos:   map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	peticion, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: vinculo, ReferenciaMotivo: motivo,
		Accion: ports.AccionCambiarSituacionParticipacion, Recurso: recurso,
		Finalidad: ports.FinalidadCambiarSituacionParticipacion, Correlacion: correlacion})
	if err != nil {
		return cero, "", nil, ports.ErrCeseNativoCTDenegado
	}
	decision, confirmacion, exportador, err := situacion.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx,
		peticion, resultado)
	if err != nil || nuloPersonaAceptacionCT(exportador) || decision.ValidarPara(peticion) != nil {
		return cero, "", nil, ports.ErrCeseNativoCTNoDisponible
	}
	concedida, _, err := decision.Resultado()
	if err != nil {
		return cero, "", nil, ports.ErrCeseNativoCTNoDisponible
	}
	if !concedida {
		return cero, "", nil, ports.ErrCeseNativoCTDenegado
	}
	orden, err := vecports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		peticion, decision, motivo, resultado)
	if err != nil || confirmacion.ValidarPara(orden) != nil {
		return cero, "", nil, ports.ErrCeseNativoCTNoDisponible
	}
	restricciones, err := decision.RestriccionesProyeccionPara(peticion)
	if err != nil || len(restricciones.CamposPermitidos) != 0 || len(restricciones.Obligaciones) != 0 ||
		!confirmacion.DentroDeVentanaEn(situacion.reloj().UTC().Truncate(time.Microsecond)) {
		return cero, "", nil, ports.ErrCeseNativoCTDenegado
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(peticion, decision, confirmacion,
		resultado, motivo, material, ports.AudienciaCambiarSituacionParticipacion) {
		return cero, "", nil, ports.ErrCeseNativoCTNoDisponible
	}
	return material, v.PrincipalID, canon, nil
}

func pruebaCierrePersonalCeseCTValida(p ports.PruebaCierrePersonalCeseCT) bool {
	if p.Esquema != ports.EsquemaPruebaCierrePersonalCeseCT || p.VersionCT == 0 ||
		p.RelacionVersion == 0 || p.OcupacionVersion == 0 ||
		p.RelacionReciboRef == p.OcupacionReciboRef ||
		!huellaPersonaAceptacionCT.MatchString(p.FuenteHuellaSHA256) ||
		!fechaCivilCeseNativoValida(p.VigenteHasta) {
		return false
	}
	for _, ref := range []string{p.ReciboCTRef, p.ActorRef, p.EmpleadoRef, p.RelacionRef,
		p.OcupacionRef, p.OrganismoRef, p.UnidadRef, p.RelacionReciboRef, p.OcupacionReciboRef} {
		if !referenciaAceptacionNativaCT.MatchString(ref) {
			return false
		}
	}
	return true
}

func fechaCivilCeseNativoValida(s string) bool {
	t, err := time.Parse(time.DateOnly, s)
	return err == nil && t.Format(time.DateOnly) == s
}

var _ ports.AplicadorCeseNativoCT = (*ServicioCeseNativoCT)(nil)

// ServicioAltaNativaCT registra el hecho de incorporación B2 en la historia
// B13 y sitúa la participación trabajando sólo después de CT/P23 confirmados.
type ServicioAltaNativaCT struct {
	situacion   *ServicioSituacionParticipacion
	repositorio ports.RepositorioAltaNativaCT
}

func NuevoServicioAltaNativaCT(s *ServicioSituacionParticipacion, r ports.RepositorioAltaNativaCT) (*ServicioAltaNativaCT, error) {
	if s == nil || s.reloj == nil || nuloPersonaAceptacionCT(r) {
		return nil, ports.ErrAltaNativaCTNoDisponible
	}
	return &ServicioAltaNativaCT{situacion: s, repositorio: r}, nil
}

func (s *ServicioAltaNativaCT) AplicarAltaNativaCT(ctx context.Context, q ports.SolicitudAltaNativaCT) (ports.ReciboAltaNativaCT, error) {
	var cero ports.ReciboAltaNativaCT
	if ctx == nil || s == nil || s.situacion == nil || nuloPersonaAceptacionCT(s.repositorio) {
		return cero, ports.ErrAltaNativaCTNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if !referenciaAceptacionNativaCT.MatchString(q.ParticipacionRef) ||
		!pruebaAltaPersonalCTValida(q.Personal) ||
		q.ResultadoContexto.Validar() != nil || q.Vinculo.ValidarPara(q.ResultadoContexto) != nil ||
		q.Correlacion.Validar() != nil || !core.ReferenciaMotivoAutorizacionV2Valida(q.MotivoAutorizacion) {
		return cero, ports.ErrAltaNativaCTInvalida
	}
	if _, err := PrepararConsultaAceptacionNativaCT(ports.SolicitudConsultaAceptacionNativaCT{
		Selector: q.Selector, ActorConfiable: ports.ActorConfiablePersonaAceptacionCT{
			Vinculo: q.Vinculo, Resultado: q.ResultadoContexto,
		},
	}, false); err != nil {
		return cero, ports.ErrAltaNativaCTInvalida
	}
	material, actor, canon, err := autorizarActoNativoCT(ctx, s.situacion, q.Selector,
		q.ParticipacionRef, q.Personal.ActorRef, q.Vinculo, q.ResultadoContexto,
		q.Correlacion, q.MotivoAutorizacion, func(unidad, ambito string) any {
			return ports.MaterialActoAltaNativaCT{Esquema: ports.EsquemaAltaNativaCT,
				Selector: q.Selector, ParticipacionRef: q.ParticipacionRef,
				UnidadAutorizacionRef: unidad, AmbitoRef: ambito, Personal: q.Personal}
		})
	if err != nil {
		return cero, errorAltaNativaCT(err)
	}
	r, err := s.repositorio.AplicarAltaNativaCT(ctx, ports.OrdenAltaNativaCT{
		Solicitud: q, ActorRef: actor, MaterialCanonico: canon, Material: material})
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, err
	}
	instante, err := time.Parse("2006-01-02T15:04:05.000000Z", r.TrabajandoDesde)
	if err != nil || instante.Location() != time.UTC ||
		r.ParticipacionRef != q.ParticipacionRef || r.BolsaRef != q.Selector.BolsaRef ||
		!strings.HasPrefix(r.EventoRef, "evento:ct:contrato-bolsa:") ||
		!strings.HasPrefix(r.ReciboSituacion, "recibo:situacion:") ||
		!strings.HasPrefix(r.OutboxRef, "evento:bolsa:alta-nativa:") ||
		!referenciaAceptacionNativaCT.MatchString(r.AuditoriaRef) {
		return cero, ports.ErrAltaNativaCTNoDisponible
	}
	return r, nil
}

func pruebaAltaPersonalCTValida(p ports.PruebaAltaPersonalCT) bool {
	if p.VersionExpediente == 0 || p.RelacionVersion == 0 || p.OcupacionVersion == 0 ||
		p.RelacionReciboRef == p.OcupacionReciboRef ||
		!huellaPersonaAceptacionCT.MatchString(p.PlanSHA256) {
		return false
	}
	for _, ref := range []string{p.ReciboCTRef, p.ActorRef, p.EmpleadoRef, p.RelacionRef,
		p.RelacionReciboRef, p.OcupacionRef, p.OcupacionReciboRef, p.PlanRef} {
		if !referenciaAceptacionNativaCT.MatchString(ref) {
			return false
		}
	}
	return true
}

func errorAltaNativaCT(err error) error {
	switch {
	case errors.Is(err, ports.ErrCeseNativoCTInvalido):
		return ports.ErrAltaNativaCTInvalida
	case errors.Is(err, ports.ErrCeseNativoCTDenegado):
		return ports.ErrAltaNativaCTDenegada
	case errors.Is(err, ports.ErrCeseNativoCTConflicto):
		return ports.ErrAltaNativaCTConflicto
	case errors.Is(err, ports.ErrCeseNativoCTNoDisponible):
		return ports.ErrAltaNativaCTNoDisponible
	default:
		return err
	}
}

var _ ports.AplicadorAltaNativaCT = (*ServicioAltaNativaCT)(nil)
