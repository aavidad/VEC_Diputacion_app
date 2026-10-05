package administracion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"time"

	lote "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Acción y audiencia del lote ordinario de perfiles (AUT45 v6 y AD190).
const (
	AccionLoteOrdinarioV3    = "administracion.perfiles.aplicar_lote_ordinario"
	AudienciaLoteOrdinarioV3 = "vec_autorizacion.administracion_perfiles.lote_ordinario.v1"
	finalidadLoteOrdinario   = "gestion_perfiles"
	// AudienciaGobiernoPlanFirmaV3 es la del gobierno del plan nominal de firma
	// de Contratación temporal (AD177/AD178, conjunto 2 de AD202). Tiene que
	// coincidir con plannominal.AudienciaGobiernoPlanFirma.
	AudienciaGobiernoPlanFirmaV3 = "vec_catalogos_configurables.plan_nominal_firma.gobierno.v1"
)

// EmisorLote adapta una orden de lote ya validada a la cadena común V3. El
// motivo procede del catálogo privado; ni el DTO ni el recurso lo aportan.
type EmisorLote struct {
	emisor *confianza.EmisorMaterialAutorizacionAtestadaV3
	motivo domain.ReferenciaEntradaCatalogo
	reloj  ports.Reloj
}

var _ lote.EmisorLoteOrdinario = (*EmisorLote)(nil)

func NuevoEmisorLote(emisor *confianza.EmisorMaterialAutorizacionAtestadaV3, motivo domain.ReferenciaEntradaCatalogo, reloj ports.Reloj) (*EmisorLote, error) {
	if emisor == nil || !domain.ReferenciaMotivoAutorizacionV2Valida(motivo) || dependenciaConfianzaPerfilesNula(reloj) {
		return nil, ErrConfiguracion
	}
	return &EmisorLote{emisor: emisor, motivo: motivo, reloj: reloj}, nil
}

// La instantánea es una precondición: la concesión exacta del lote tiene que
// estar en la versión del rol que señala la asignación del actor, y esa
// asignación tiene que cubrir los ámbitos del recurso. El PDP y AD190 lo
// vuelven a comprobar contra sus fuentes durables.
func snapshotLoteValido(s domain.InstantaneaAutorizacion, actor domain.ContextoActor, recurso domain.RecursoAutorizable, ahora time.Time) bool {
	if s.Validar() != nil || !domain.VersionRolAplicacionAdmitida(s.VersionRol.Referencia()) ||
		s.VersionRol.Estado != domain.EstadoVersionRolPublicada ||
		s.ControlVigenciaVersionRol.Estado != domain.EstadoControlVigenciaVersionRolHabilitada ||
		ahora.Before(s.VersionRol.PublicadaEn) || ahora.Before(s.ControlVigenciaVersionRol.ActualizadoEn) ||
		s.AsignacionPerfil.PrincipalID != actor.PersonaRef || s.AsignacionPerfil.PerfilActivoRef != actor.PerfilActivoRef ||
		!s.AsignacionPerfil.VigenteEn(ahora) || !s.AsignacionPerfil.Cubre(recurso) {
		return false
	}
	for _, c := range s.VersionRol.Concesiones {
		if c.Accion == AccionLoteOrdinarioV3 && c.ModuloID == "administracion" && c.TipoRecurso == "persona" {
			return slices.Equal(c.Finalidades, []string{finalidadLoteOrdinario}) && c.GarantiaMinima == domain.AuthAssuranceHigh &&
				len(c.CamposPermitidos) == 0 && slices.Equal(c.Obligaciones, []string{"auditar"})
		}
	}
	return false
}

// errorEmisorLote no conserva mensaje SQL, material criptográfico ni texto del
// proveedor: sólo la indisponibilidad cerrada del puerto.
func errorEmisorLote(err error) error {
	if err != nil {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return nil
}

func (e *EmisorLote) EmitirLoteOrdinario(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles,
	snapshot domain.InstantaneaAutorizacion, recurso domain.RecursoAutorizable, efecto lote.Efecto,
) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacia := ports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	fallo := ports.ErrAutoridadAdministracionPerfilesNoDisponible
	if e == nil || e.emisor == nil || ctx == nil || ctx.Err() != nil || dependenciaConfianzaPerfilesNula(e.reloj) {
		return vacia, fallo
	}
	ahora := e.reloj.Ahora()
	if actor.Validar() != nil || evidencia.ValidarEn(actor, ahora) != nil || !actor.Instantanea.VigenteEn(ahora) ||
		efecto.Accion != AccionLoteOrdinarioV3 || efecto.Audiencia != AudienciaLoteOrdinarioV3 ||
		recurso.Validar() != nil || recurso.Referencia != efecto.Referencia || recurso.ModuloID != "administracion" ||
		recurso.Tipo != "persona" || recurso.Referencia == actor.PersonaRef || len(efecto.Material) == 0 ||
		len(recurso.Ambitos) != 2 || recurso.Ambitos["organizacion_ref"] == "" || recurso.Ambitos["unidad_ref"] == "" ||
		!atributoMaterialLote(recurso.Atributos, efecto.Material) {
		return vacia, fallo
	}
	vinculo, err := evidencia.Vinculo.Datos()
	if err != nil {
		return vacia, errorEmisorLote(err)
	}
	if !vinculo.CuentaPrivilegiada || vinculo.Superficie != domain.SuperficieAutenticacionAdministracionPrivilegiadaV1 ||
		vinculo.GarantiaObservada != domain.AuthAssuranceHigh || !snapshotLoteValido(snapshot, actor, recurso, ahora) {
		return vacia, fallo
	}
	// La correlación es la del acceso actual: sale de la frontera de la
	// petición y tiene que coincidir con la que el efecto conserva.
	correlacion, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return vacia, errorEmisorLote(err)
	}
	valor, err := correlacion.ValorCanonico()
	if err != nil {
		return vacia, errorEmisorLote(err)
	}
	if valor != efecto.CorrelacionAccesoRef {
		return vacia, fallo
	}
	resultado, err := evidencia.ResultadoContexto.Clonar()
	if err != nil {
		return vacia, errorEmisorLote(err)
	}
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(domain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: evidencia.Vinculo, ReferenciaMotivo: e.motivo, Accion: AccionLoteOrdinarioV3,
		Recurso: recurso, Finalidad: finalidadLoteOrdinario, Correlacion: correlacion})
	if err != nil {
		return vacia, errorEmisorLote(err)
	}
	decision, confirmacion, exportador, err := e.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
	if err != nil {
		// Sólo la denegación durable explícita se distingue; nunca se infiere
		// de un 42501 ni del texto de un error.
		if ctx.Err() == nil && errors.Is(err, ports.ErrDenegacionExplicitaAutorizacionLigadaV3) {
			return vacia, domain.ErrAutorizacionDenegada
		}
		return vacia, errorEmisorLote(err)
	}
	ahora = e.reloj.Ahora()
	if ctx.Err() != nil || dependenciaConfianzaPerfilesNula(exportador) || confirmacion.Validar() != nil ||
		validarDecisionLote(decision, confirmacion, solicitud, e.motivo, resultado, ahora) != nil || !evidencia.Vinculo.VigenteEn(ahora, resultado) {
		return vacia, fallo
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return vacia, errorEmisorLote(err)
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return vacia, errorEmisorLote(err)
	}
	r := material.ResumenCapacidad()
	if material.ValidarEstructura() != nil || ctx.Err() != nil || r.Operacion() != AccionLoteOrdinarioV3 ||
		r.AudienciaConsumo() != AudienciaLoteOrdinarioV3 || r.EfectoRef() != recurso.Referencia || r.EfectoHuellaSHA256() != huella ||
		r.ContextoRef() != resultado.RegistroContextoRef || r.ContextoHuellaSHA256() != resultado.HuellaSHA256 ||
		!bytes.Equal(material.ContextoActorCanonico(), resultado.RepresentacionCanonica) ||
		material.PersonaVersion() != resultado.Contexto.Instantanea.PersonaVersion ||
		material.PerfilVersion() != resultado.Contexto.Instantanea.PerfilVersion {
		return vacia, fallo
	}
	return material, nil
}

// La vigencia se comprueba sobre la confirmación durable del registro y la
// decisión debe llevar la concesión exacta del lote (sin campos, auditar).
func validarDecisionLote(d domain.DecisionAutorizacionLigadaV3, confirmacion ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
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
		len(datos.Campos) != 0 || !slices.Equal(datos.Obligaciones, []string{"auditar"}) {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return nil
}

// atributoMaterialLote exige un único atributo, el del lote o el de su
// preparación, con la huella exacta del material. AUT44 y AUT50 recalculan el
// contexto con su propio nombre de atributo: una decisión no sirve para el otro.
func atributoMaterialLote(atributos map[string]string, material []byte) bool {
	if len(atributos) != 1 {
		return false
	}
	h := huellaMaterialLote(material)
	return atributos[lote.AtributoSolicitudLote] == h || atributos[lote.AtributoPreparacionLote] == h
}

// huellaMaterialLote es la huella de la solicitud que el recurso declara y
// que AUT44 vuelve a calcular sobre el mismo material.
func huellaMaterialLote(material []byte) string {
	h := sha256.Sum256(material)
	return hex.EncodeToString(h[:])
}
