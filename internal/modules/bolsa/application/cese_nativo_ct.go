package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	material, actor, canon, err := s.autorizarCeseNativoCT(ctx, q)
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

func (s *ServicioCeseNativoCT) autorizarCeseNativoCT(ctx context.Context, q ports.SolicitudCeseNativoCT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, string, []byte, error) {
	var cero vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if nuloPersonaAceptacionCT(s.situacion.contexto) || nuloPersonaAceptacionCT(s.situacion.autorizador) ||
		nuloPersonaAceptacionCT(s.situacion.repositorio) || s.situacion.reloj == nil {
		return cero, "", nil, ports.ErrCeseNativoCTNoDisponible
	}
	v, err := q.Vinculo.Datos()
	if err != nil || v.PrincipalID == "" || v.PrincipalID != q.Personal.ActorRef ||
		q.ResultadoContexto.Contexto.PersonaRef != v.PrincipalID {
		return cero, "", nil, ports.ErrCeseNativoCTDenegado
	}
	resuelto, err := s.situacion.contexto.ResolverContextoSituacionParticipacion(ctx,
		q.ResultadoContexto.Contexto, q.Selector.BolsaRef, q.ParticipacionRef)
	if err != nil || resuelto.Validar() != nil {
		return cero, "", nil, ports.ErrCeseNativoCTNoDisponible
	}
	pertenece, err := s.situacion.repositorio.ParticipacionPerteneceABolsa(ctx,
		q.Selector.BolsaRef, q.ParticipacionRef)
	if err != nil {
		return cero, "", nil, ports.ErrCeseNativoCTNoDisponible
	}
	if !pertenece {
		return cero, "", nil, ports.ErrCeseNativoCTDenegado
	}
	intencion := ports.MaterialActoCeseNativoCT{Esquema: ports.EsquemaActoCeseNativoCT,
		Selector: q.Selector, ParticipacionRef: q.ParticipacionRef,
		UnidadAutorizacionRef: resuelto.UnidadRef, AmbitoRef: resuelto.AmbitoRef, Personal: q.Personal}
	canon, err := json.Marshal(intencion)
	if err != nil || len(canon) == 0 || len(canon) > 16<<10 {
		return cero, "", nil, ports.ErrCeseNativoCTInvalido
	}
	h := sha256.Sum256(canon)
	recurso := core.RecursoAutorizable{Referencia: q.ParticipacionRef,
		ModuloID: ports.ModuloSituacionParticipacion, Tipo: ports.TipoRecursoSituacionParticipacion,
		Ambitos:   map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	peticion, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: q.Vinculo, ReferenciaMotivo: q.MotivoAutorizacion,
		Accion: ports.AccionCambiarSituacionParticipacion, Recurso: recurso,
		Finalidad: ports.FinalidadCambiarSituacionParticipacion, Correlacion: q.Correlacion})
	if err != nil {
		return cero, "", nil, ports.ErrCeseNativoCTDenegado
	}
	decision, confirmacion, exportador, err := s.situacion.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx,
		peticion, q.ResultadoContexto)
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
		peticion, decision, q.MotivoAutorizacion, q.ResultadoContexto)
	if err != nil || confirmacion.ValidarPara(orden) != nil {
		return cero, "", nil, ports.ErrCeseNativoCTNoDisponible
	}
	restricciones, err := decision.RestriccionesProyeccionPara(peticion)
	if err != nil || len(restricciones.CamposPermitidos) != 0 || len(restricciones.Obligaciones) != 0 ||
		!confirmacion.DentroDeVentanaEn(s.situacion.reloj().UTC().Truncate(time.Microsecond)) {
		return cero, "", nil, ports.ErrCeseNativoCTDenegado
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(peticion, decision, confirmacion,
		q.ResultadoContexto, q.MotivoAutorizacion, material, ports.AudienciaCambiarSituacionParticipacion) {
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
