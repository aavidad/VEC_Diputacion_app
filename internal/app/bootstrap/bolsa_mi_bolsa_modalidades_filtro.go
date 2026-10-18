package bootstrap

import (
	"context"

	bolsapg "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	mibolsa "vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// Sin lectura nominal, la consulta propia conserva los datos ya autorizados
// pero no muestra ofertas por defecto.
type lectorModalidadesNoDisponible struct{}

var _ mibolsa.LectorEstadoPreferenciasModalidades = lectorModalidadesNoDisponible{}

func (lectorModalidadesNoDisponible) ConsultarPreferenciasModalidades(context.Context, mibolsa.Orden) (bolsa.EstadoPreferenciasModalidades, error) {
	return bolsa.EstadoPreferenciasModalidades{}, bolsa.ErrPreferenciasNoDisponibles
}

func nuevaLecturaModalidadesPortalExterno(d dependenciasMiBolsaPortalExterno) (*mibolsa.LecturaPreferenciasModalidades, error) {
	if d.bolsa == nil || d.catalogoModalidades == nil || !d.modalidadesSQL ||
		d.proveedorPreferencias == nil || d.fuentePreferencias == nil {
		return nil, bolsa.ErrPreferenciasNoDisponibles
	}
	lecturaPG, err := bolsapg.NuevaLecturaPreferenciasModalidadesPostgreSQL(d.bolsa)
	if err != nil {
		return nil, err
	}
	auditoriaPG, err := bolsapg.NuevaAuditoriaIntentosPreferenciasBolsasPostgreSQL(d.bolsa)
	if err != nil {
		return nil, err
	}
	return mibolsa.NuevaLecturaPreferenciasModalidades(lecturaPG, auditoriaPG,
		d.fuentePreferencias, d.catalogoModalidades, d.reloj)
}
