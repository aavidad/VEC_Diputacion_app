package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

const AudienciaCierreAdministrativoSinCese = "vec_contratacion_temporal.cierre_administrativo_sin_cese.v1"

// ProveedorAutorizacionCierreAdministrativo pertenece a la frontera confiable.
// Resuelve identidad y concesion actuales tambien para recuperar un recibo.
// No debe construirse con campos de identidad recibidos por HTTP.
type ProveedorAutorizacionCierreAdministrativo interface {
	AutorizarCierreAdministrativo(context.Context, ports.SolicitudTransaccionCierreAdministrativo) (AutorizacionCierreAdministrativo, error)
}

type AutorizacionCierreAdministrativo struct {
	Contexto       ports.ContextoAutorizacionAltaV3
	Solicitud      vd.SolicitudAutorizacionLigadaV3
	Decision       vd.DecisionAutorizacionLigadaV3
	Confirmacion   vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Motivo         vd.ReferenciaEntradaCatalogo
	ActorRef       string
	PerfilRef      string
	UnidadRef      string
	CorrelacionRef string
	Exportacion    vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (a AutorizacionCierreAdministrativo) validarPara(s ports.SolicitudTransaccionCierreAdministrativo) error {
	if s.Validar() != nil || s.Operacion != ports.OperacionCerrarAdministrativamenteSinCese || a.Exportacion.ValidarEstructura() != nil || a.Decision.ValidarPara(a.Solicitud) != nil {
		return ports.ErrCierreAdministrativoDenegado
	}
	d, err := a.Solicitud.Datos()
	if err != nil {
		return ports.ErrCierreAdministrativoDenegado
	}
	v, err := d.VinculoAutenticacionActor.Datos()
	if err != nil {
		return ports.ErrCierreAdministrativoDenegado
	}
	correlacion, err := d.Correlacion.ValorCanonico()
	if err != nil {
		return ports.ErrCierreAdministrativoDenegado
	}
	r := vd.RecursoAutorizable{Referencia: s.SeguimientoRef, ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoCierreAdministrativo,
		Ambitos:   map[string]string{"organizacion_ref": s.OrganizacionRef, "expediente_ref": s.ExpedienteRef, "seguimiento_ref": s.SeguimientoRef},
		Atributos: map[string]string{"operacion": string(s.Operacion), "version_esperada": strconv.FormatUint(s.VersionEsperada, 10), "transicion_clave": string(s.TransicionClave), "motivo_clave": string(s.MotivoClave), "principal_v3_ref": v.PrincipalID, "actor_seguimiento_ref": a.ActorRef, "correlacion_v3_ref": correlacion, "correlacion_seguimiento_ref": a.CorrelacionRef},
	}
	esperado, err := r.HuellaContextoAutorizacionSHA256()
	actual, errActual := d.Recurso.HuellaContextoAutorizacionSHA256()
	decision, errDecision := vd.RepresentacionCanonicaDecisionAutorizacionV3(a.Decision)
	concedida, _, errConcedida := a.Decision.Resultado()
	confirmacion, errConfirmacion := a.Confirmacion.Datos()
	resumen := a.Exportacion.ResumenCapacidad()
	huella := sha256.Sum256(decision)
	if err != nil || errActual != nil || errDecision != nil || errConcedida != nil || errConfirmacion != nil || !concedida ||
		esperado != actual || d.Recurso.Referencia != r.Referencia || d.Recurso.ModuloID != r.ModuloID || d.Recurso.Tipo != r.Tipo ||
		d.Accion != ports.AccionAutorizacionCerrarAdministrativamente || d.Finalidad != ports.FinalidadAutorizacionCerrarAdministrativamente ||
		d.ReferenciaMotivo != a.Motivo || v.PerfilActivoRef != a.PerfilRef ||
		!d.VinculoAutenticacionActor.CoincideExactamenteCon(a.Contexto.Vinculo) || a.Contexto.Vinculo.ValidarPara(a.Contexto.Resultado) != nil ||
		!bytes.Equal(decision, a.Exportacion.DecisionCanonica()) || confirmacion.DecisionHuellaSHA256 != hex.EncodeToString(huella[:]) ||
		resumen.EfectoRef() != s.SeguimientoRef || resumen.EfectoHuellaSHA256() != esperado ||
		resumen.Operacion() != ports.AccionAutorizacionCerrarAdministrativamente || resumen.AudienciaConsumo() != AudienciaCierreAdministrativoSinCese {
		return ports.ErrCierreAdministrativoDenegado
	}
	return nil
}

// No exporta objetos opacos V3; solo coordenadas ya ligadas a su contexto.
func (a AutorizacionCierreAdministrativo) coordenadasSQL() ([]byte, error) {
	d, err := a.Solicitud.Datos()
	if err != nil {
		return nil, ports.ErrCierreAdministrativoDenegado
	}
	v, err := d.VinculoAutenticacionActor.Datos()
	if err != nil {
		return nil, ports.ErrCierreAdministrativoDenegado
	}
	c, err := d.Correlacion.ValorCanonico()
	if err != nil {
		return nil, ports.ErrCierreAdministrativoDenegado
	}
	return json.Marshal(struct {
		Actor         string `json:"actor_ref"`
		Perfil        string `json:"perfil_ref"`
		Unidad        string `json:"unidad_ref"`
		Correlacion   string `json:"correlacion_ref"`
		PrincipalV3   string `json:"principal_v3_ref"`
		CorrelacionV3 string `json:"correlacion_v3_ref"`
	}{a.ActorRef, a.PerfilRef, a.UnidadRef, a.CorrelacionRef, v.PrincipalID, c})
}
