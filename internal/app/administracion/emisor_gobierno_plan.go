package administracion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/plannominal"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Coordenadas del gobierno del plan nominal de firma (AD177/AD178/AD201). La
// acción concreta (vec.catalogos.{crear,actualizar,publicar,retirar}) sale de
// los bytes exactos del material, nunca de la petición.
const (
	moduloGobiernoPlanFirma    = "contratacion_temporal"
	tipoGobiernoPlanFirma      = "catalogo_configurable"
	finalidadGobiernoPlanFirma = "gestionar_contratacion_temporal"
)

var _ plannominal.EmisorGobiernoPlanFirma = (*EmisorGobiernoPlanFirma)(nil)

// EmisorGobiernoPlanFirma adapta el material exacto del kit del plan a la
// cadena común V3 de vec-admin. El motivo procede del catálogo privado.
type EmisorGobiernoPlanFirma struct {
	emisor *confianza.EmisorMaterialAutorizacionAtestadaV3
	motivo domain.ReferenciaEntradaCatalogo
	reloj  ports.Reloj
}

func NuevoEmisorGobiernoPlanFirma(emisor *confianza.EmisorMaterialAutorizacionAtestadaV3, motivo domain.ReferenciaEntradaCatalogo, reloj ports.Reloj) (*EmisorGobiernoPlanFirma, error) {
	if emisor == nil || !domain.ReferenciaMotivoAutorizacionV2Valida(motivo) || dependenciaConfianzaPerfilesNula(reloj) {
		return nil, ErrConfiguracion
	}
	return &EmisorGobiernoPlanFirma{emisor: emisor, motivo: motivo, reloj: reloj}, nil
}

// concesionGobiernoPlanFirma es la que publicó AUT51 en el rol de Aplicación.
func concesionGobiernoPlanFirma(c domain.ConcesionRol, accion string) bool {
	return c.Accion == accion && c.ModuloID == moduloGobiernoPlanFirma && c.TipoRecurso == tipoGobiernoPlanFirma &&
		slices.Equal(c.Finalidades, []string{finalidadGobiernoPlanFirma}) && c.GarantiaMinima == domain.AuthAssuranceHigh &&
		len(c.CamposPermitidos) == 0 && len(c.Obligaciones) == 0
}

// snapshotGobiernoPlanFirmaValido es una precondición: el PDP, AUT y AD201 lo
// vuelven a comprobar contra sus fuentes durables.
func snapshotGobiernoPlanFirmaValido(s domain.InstantaneaAutorizacion, actor domain.ContextoActor, accion string, recurso domain.RecursoAutorizable, ahora time.Time) bool {
	if s.Validar() != nil || !domain.VersionRolAplicacionAdmitida(s.VersionRol.Referencia()) ||
		s.VersionRol.Estado != domain.EstadoVersionRolPublicada ||
		s.ControlVigenciaVersionRol.Estado != domain.EstadoControlVigenciaVersionRolHabilitada ||
		ahora.Before(s.VersionRol.PublicadaEn) || ahora.Before(s.ControlVigenciaVersionRol.ActualizadoEn) ||
		s.AsignacionPerfil.PrincipalID != actor.PersonaRef || s.AsignacionPerfil.PerfilActivoRef != actor.PerfilActivoRef ||
		!s.AsignacionPerfil.VigenteEn(ahora) || !s.AsignacionPerfil.Cubre(recurso) {
		return false
	}
	// Como en el lote: decide la primera concesión de esa acción, módulo y tipo.
	for _, c := range s.VersionRol.Concesiones {
		if c.Accion == accion && c.ModuloID == moduloGobiernoPlanFirma && c.TipoRecurso == tipoGobiernoPlanFirma {
			return concesionGobiernoPlanFirma(c, accion)
		}
	}
	return false
}

// EmitirGobiernoPlanFirma deriva acción y recurso de los bytes exactos del
// material y de los ámbitos de la asignación del administrador, y pide al PDP
// la decisión V3 de la audiencia del gobierno del plan.
func (e *EmisorGobiernoPlanFirma) EmitirGobiernoPlanFirma(ctx context.Context, actor domain.ContextoActor,
	evidencia domain.EvidenciaSesionAdministracionPerfiles, snapshot domain.InstantaneaAutorizacion,
	material []byte, correlacionAcceso string,
) (plannominal.EmisionGobiernoPlanFirma, error) {
	var vacia plannominal.EmisionGobiernoPlanFirma
	fallo := ports.ErrAutoridadAdministracionPerfilesNoDisponible
	if e == nil || e.emisor == nil || ctx == nil || ctx.Err() != nil || dependenciaConfianzaPerfilesNula(e.reloj) {
		return vacia, fallo
	}
	ahora := e.reloj.Ahora()
	if actor.Validar() != nil || evidencia.ValidarEn(actor, ahora) != nil || !actor.Instantanea.VigenteEn(ahora) {
		return vacia, fallo
	}
	ambito, err := plannominal.AmbitoGobiernoPlanFirmaDeAsignacion(snapshot.AsignacionPerfil)
	if err != nil {
		return vacia, domain.ErrActoAdministracionPerfilesInvalido
	}
	accion, recurso, err := plannominal.RecursoGobiernoPlanFirma(material, ambito)
	if err != nil {
		return vacia, domain.ErrActoAdministracionPerfilesInvalido
	}
	vinculo, err := evidencia.Vinculo.Datos()
	if err != nil {
		return vacia, errorEmisorLote(err)
	}
	if !vinculo.CuentaPrivilegiada || vinculo.Superficie != domain.SuperficieAutenticacionAdministracionPrivilegiadaV1 ||
		vinculo.GarantiaObservada != domain.AuthAssuranceHigh || !snapshotGobiernoPlanFirmaValido(snapshot, actor, accion, recurso, ahora) {
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
		Recurso: recurso, Finalidad: finalidadGobiernoPlanFirma, Correlacion: correlacion})
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
		validarDecisionGobiernoPlanFirma(decision, confirmacion, solicitud, e.motivo, resultado, ahora) != nil ||
		!evidencia.Vinculo.VigenteEn(ahora, resultado) {
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
		r.AudienciaConsumo() != AudienciaGobiernoPlanFirmaV3 || r.EfectoRef() != recurso.Referencia || r.EfectoHuellaSHA256() != huella ||
		r.ContextoRef() != resultado.RegistroContextoRef || r.ContextoHuellaSHA256() != resultado.HuellaSHA256 ||
		!bytes.Equal(exportado.ContextoActorCanonico(), resultado.RepresentacionCanonica) ||
		exportado.PersonaVersion() != resultado.Contexto.Instantanea.PersonaVersion ||
		exportado.PerfilVersion() != resultado.Contexto.Instantanea.PerfilVersion {
		return vacia, fallo
	}
	return plannominal.EmisionGobiernoPlanFirma{Accion: accion, Recurso: recurso, Ambito: ambito, Material: exportado}, nil
}

// La decisión debe llevar la concesión exacta del gobierno: sin campos ni
// obligaciones (AD178 lo exige también en el consumo).
func validarDecisionGobiernoPlanFirma(d domain.DecisionAutorizacionLigadaV3, confirmacion ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	solicitud domain.SolicitudAutorizacionLigadaV3, motivo domain.ReferenciaEntradaCatalogo, resultado domain.ResultadoContextoActorRegistradoV2, ahora time.Time,
) error {
	concedida, _, err := d.Resultado()
	if err != nil {
		return errorEmisorLote(err)
	}
	if !concedida || d.ValidarPara(solicitud) != nil {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, d, motivo, resultado)
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
		len(datos.Campos) != 0 || len(datos.Obligaciones) != 0 {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return nil
}
