package application

import (
	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	AudienciaRecuperarReciboCorreccionPropia = "vec_cronos_v1.correccion_propia.recibo.consultar.v1"
	FinalidadRecuperarReciboCorreccionPropia = "consultar_recibo_correccion_propia"
)

func RecursoRecuperarReciboCorreccionPropia(m domain.MaterialRecuperacionCorreccionPropia) (vecdomain.RecursoAutorizable, error) {
	canonico, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return recursoEmpleado(m.SolicitudRef, "recibo_correccion_propia", m.EmpleadoRef, canonico)
}
