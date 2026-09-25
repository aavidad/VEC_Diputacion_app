package application

import (
	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Contrato V3 del segundo corte de la persona empleada. Cada acción tiene su
// audiencia nominal (AD3-70) y la función durable de cronos_v1 000008
// comprueba exactamente estos valores antes de consumir.
const (
	AudienciaConsultaMovimientosPropios  = "vec_cronos_v1.movimientos_propio.consultar.v1"
	AccionConsultarMovimientosPropios    = "cronos.movimientos.propio.consultar"
	FinalidadConsultarMovimientosPropios = "consultar_movimientos_propio"

	AudienciaSolicitudCorreccionPropia = "vec_cronos_v1.correccion_propia.solicitar.v1"
	FinalidadSolicitarCorreccion       = "solicitar_correccion_marcaje"

	AudienciaConsultaPermisosPropios  = "vec_cronos_v1.permisos_propio.consultar.v1"
	AccionConsultarPermisosPropios    = "cronos.permisos.propio.consultar"
	FinalidadConsultarPermisosPropios = "consultar_permisos_propio"

	AudienciaSolicitudPermisoPropio = "vec_cronos_v1.permiso_propio.solicitar.v1"
	AccionSolicitarPermisoPropio    = "cronos.permiso.solicitar"
	FinalidadSolicitarPermisoPropio = "solicitar_permiso_propio"
)

func RecursoConsultaMovimientosPropios(m domain.MaterialConsultaMovimientosPropios) (vecdomain.RecursoAutorizable, error) {
	canonico, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return recursoEmpleado("movimientos:cronos:"+m.EmpleadoRef, "movimientos_propio", m.EmpleadoRef, canonico)
}

func RecursoConsultaPermisosPropios(m domain.MaterialConsultaPermisosPropios) (vecdomain.RecursoAutorizable, error) {
	canonico, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return recursoEmpleado("permisos:cronos:"+m.EmpleadoRef, "permisos_propio", m.EmpleadoRef, canonico)
}

func RecursoSolicitudPermisoPropio(m domain.MaterialSolicitudPermisoPropio) (vecdomain.RecursoAutorizable, error) {
	canonico, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return recursoEmpleado(domain.SolicitudPermisoPropioRef(m.ClaveOperacion), "solicitud_permiso", m.EmpleadoRef, canonico)
}

// RecursoSolicitudCorreccionPropia liga la solicitud a la huella del
// material exacto (ComandoSHA256) que resume la función durable.
func RecursoSolicitudCorreccionPropia(m domain.MaterialAutorizacionCorreccion) (vecdomain.RecursoAutorizable, error) {
	if m.Validar() != nil || m.Paso != domain.PasoSolicitudCorreccion || m.SolicitudRef != "correccion:cronos:"+m.ClaveOperacion {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return recursoEmpleadoHuella(m.SolicitudRef, "correccion_marcaje", m.EmpleadoRef, m.ComandoSHA256)
}
