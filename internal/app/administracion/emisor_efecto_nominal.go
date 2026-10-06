package administracion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	efecto "vec-diputacion-granada/internal/vec/adapters/postgres/efectonominaladmin"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// EmisorEfectoNominalADMIN emite, con la sesión ADMIN de la misma petición, la
// decisión V3 de un contrato de efecto nominal y exporta su material para la
// fachada SQL del módulo. Es el mismo control que el gobierno del plan y el
// lote: vínculo privilegiado, concesión exacta en la asignación vigente y
// correlación del acceso. El motivo procede del catálogo privado.
type EmisorEfectoNominalADMIN struct {
	emisor   *confianza.EmisorMaterialAutorizacionAtestadaV3
	motivo   domain.ReferenciaEntradaCatalogo
	reloj    ports.Reloj
	contrato efecto.Contrato
}

var _ efecto.Emisor = (*EmisorEfectoNominalADMIN)(nil)

func NuevoEmisorEfectoNominalADMIN(emisor *confianza.EmisorMaterialAutorizacionAtestadaV3, motivo domain.ReferenciaEntradaCatalogo,
	reloj ports.Reloj, contrato efecto.Contrato) (*EmisorEfectoNominalADMIN, error) {
	if emisor == nil || !domain.ReferenciaMotivoAutorizacionV2Valida(motivo) || dependenciaConfianzaPerfilesNula(reloj) || !contrato.Valido() {
		return nil, ErrConfiguracion
	}
	return &EmisorEfectoNominalADMIN{emisor: emisor, motivo: motivo, reloj: reloj, contrato: contrato}, nil
}

// concesionExacta: la primera concesión de esa acción, módulo y tipo decide,
// como en el lote.
func (e *EmisorEfectoNominalADMIN) concesionExacta(s domain.InstantaneaAutorizacion, accion string) bool {
	c := e.contrato
	for _, x := range s.VersionRol.Concesiones {
		if x.Accion == accion && x.ModuloID == c.Modulo && x.TipoRecurso == c.Tipo {
			return slices.Equal(x.Finalidades, []string{c.Finalidad}) && x.GarantiaMinima == domain.AuthAssuranceHigh &&
				slices.Equal(x.CamposPermitidos, c.Campos) && len(x.Obligaciones) == 0
		}
	}
	return false
}

func (e *EmisorEfectoNominalADMIN) snapshotValido(s domain.InstantaneaAutorizacion, actor domain.ContextoActor, accion string,
	recurso domain.RecursoAutorizable, ahora time.Time) bool {
	return s.Validar() == nil && domain.VersionRolAplicacionAdmitida(s.VersionRol.Referencia()) &&
		s.VersionRol.Estado == domain.EstadoVersionRolPublicada &&
		s.ControlVigenciaVersionRol.Estado == domain.EstadoControlVigenciaVersionRolHabilitada &&
		!ahora.Before(s.VersionRol.PublicadaEn) && !ahora.Before(s.ControlVigenciaVersionRol.ActualizadoEn) &&
		s.AsignacionPerfil.PrincipalID == actor.PersonaRef && s.AsignacionPerfil.PerfilActivoRef == actor.PerfilActivoRef &&
		s.AsignacionPerfil.VigenteEn(ahora) && s.AsignacionPerfil.Cubre(recurso) && e.concesionExacta(s, accion)
}

func (e *EmisorEfectoNominalADMIN) Emitir(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles,
	snapshot domain.InstantaneaAutorizacion, material []byte, correlacionAcceso string) (efecto.Emision, error) {
	var vacia efecto.Emision
	fallo := ports.ErrAutoridadAdministracionPerfilesNoDisponible
	if e == nil || e.emisor == nil || ctx == nil || ctx.Err() != nil || dependenciaConfianzaPerfilesNula(e.reloj) || !e.contrato.Valido() {
		return vacia, fallo
	}
	ahora := e.reloj.Ahora()
	if actor.Validar() != nil || evidencia.ValidarEn(actor, ahora) != nil || !actor.Instantanea.VigenteEn(ahora) {
		return vacia, fallo
	}
	accion, recurso, err := e.contrato.Recurso(bytes.Clone(material), snapshot.AsignacionPerfil)
	if err != nil || !slices.Contains(e.contrato.Acciones, accion) || recurso.Validar() != nil ||
		recurso.ModuloID != e.contrato.Modulo || recurso.Tipo != e.contrato.Tipo {
		return vacia, domain.ErrActoAdministracionPerfilesInvalido
	}
	vinculo, err := evidencia.Vinculo.Datos()
	if err != nil {
		return vacia, errorEmisorLote(err)
	}
	if !vinculo.CuentaPrivilegiada || vinculo.Superficie != domain.SuperficieAutenticacionAdministracionPrivilegiadaV1 ||
		vinculo.GarantiaObservada != domain.AuthAssuranceHigh {
		return vacia, fallo
	}
	// Aquí los ámbitos vienen del material: uno que la asignación vigente no
	// cubre (otra organización u otra unidad) es una denegación, no una caída.
	if snapshot.Validar() == nil && snapshot.AsignacionPerfil.VigenteEn(ahora) && !snapshot.AsignacionPerfil.Cubre(recurso) {
		return vacia, domain.ErrAutorizacionDenegada
	}
	if !e.snapshotValido(snapshot, actor, accion, recurso, ahora) {
		return vacia, fallo
	}
	correlacion, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return vacia, errorEmisorLote(err)
	}
	valor, err := correlacion.ValorCanonico()
	if err != nil {
		return vacia, errorEmisorLote(err)
	}
	if valor != correlacionAcceso {
		return vacia, fallo
	}
	resultado, err := evidencia.ResultadoContexto.Clonar()
	if err != nil {
		return vacia, errorEmisorLote(err)
	}
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(domain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: evidencia.Vinculo, ReferenciaMotivo: e.motivo, Accion: accion,
		Recurso: recurso, Finalidad: e.contrato.Finalidad, Correlacion: correlacion})
	if err != nil {
		return vacia, errorEmisorLote(err)
	}
	decision, confirmacion, exportador, err := e.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
	if err != nil {
		if ctx.Err() == nil && errors.Is(err, ports.ErrDenegacionExplicitaAutorizacionLigadaV3) {
			return vacia, domain.ErrAutorizacionDenegada
		}
		return vacia, errorEmisorLote(err)
	}
	ahora = e.reloj.Ahora()
	if ctx.Err() != nil || dependenciaConfianzaPerfilesNula(exportador) || confirmacion.Validar() != nil ||
		e.validarDecision(decision, confirmacion, solicitud, resultado, ahora) != nil || !evidencia.Vinculo.VigenteEn(ahora, resultado) {
		return vacia, fallo
	}
	exportado, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return vacia, errorEmisorLote(err)
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return vacia, errorEmisorLote(err)
	}
	r := exportado.ResumenCapacidad()
	if exportado.ValidarEstructura() != nil || ctx.Err() != nil || r.Operacion() != accion ||
		r.AudienciaConsumo() != e.contrato.Audiencia || r.EfectoRef() != recurso.Referencia || r.EfectoHuellaSHA256() != huella ||
		r.ContextoRef() != resultado.RegistroContextoRef || r.ContextoHuellaSHA256() != resultado.HuellaSHA256 ||
		!bytes.Equal(exportado.ContextoActorCanonico(), resultado.RepresentacionCanonica) ||
		exportado.PersonaVersion() != resultado.Contexto.Instantanea.PersonaVersion ||
		exportado.PerfilVersion() != resultado.Contexto.Instantanea.PerfilVersion {
		return vacia, fallo
	}
	return efecto.Emision{Accion: accion, Recurso: recurso, Material: exportado}, nil
}

func (e *EmisorEfectoNominalADMIN) validarDecision(d domain.DecisionAutorizacionLigadaV3, confirmacion ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	solicitud domain.SolicitudAutorizacionLigadaV3, resultado domain.ResultadoContextoActorRegistradoV2, ahora time.Time) error {
	concedida, _, err := d.Resultado()
	if err != nil {
		return errorEmisorLote(err)
	}
	if !concedida || d.ValidarPara(solicitud) != nil {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, d, e.motivo, resultado)
	if err != nil {
		return errorEmisorLote(err)
	}
	if confirmacion.ValidarPara(orden) != nil || !confirmacion.DentroDeVentanaEn(ahora.UTC().Truncate(time.Microsecond)) {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	b, err := domain.RepresentacionCanonicaDecisionAutorizacionV3(d)
	if err != nil {
		return errorEmisorLote(err)
	}
	var datos struct {
		VersionRol   string               `json:"version_rol_ref"`
		Garantia     domain.AuthAssurance `json:"garantia_minima"`
		Campos       []string             `json:"campos_permitidos"`
		Obligaciones []string             `json:"obligaciones"`
	}
	if err := json.Unmarshal(b, &datos); err != nil {
		return errorEmisorLote(err)
	}
	if !domain.VersionRolAplicacionAdmitida(datos.VersionRol) || datos.Garantia != domain.AuthAssuranceHigh ||
		!slices.Equal(datos.Campos, e.contrato.Campos) || len(datos.Obligaciones) != 0 {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return nil
}
