//go:build integracion_postgresql_o405

package cobertura

import "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"

// NuevaSolicitudRecuperacionResultadoOperacionDecisionCoberturaIntegracionPrueba
// rehidrata únicamente el fixture durable de integración. Al vivir en
// _test.go no abre un constructor nominal en el binario de producción.
func NuevaSolicitudRecuperacionResultadoOperacionDecisionCoberturaIntegracionPrueba(
	organizacionRef string,
	expedienteRef string,
	ambitos ports.ColeccionSellosHMAC,
) (SolicitudRecuperacionResultadoOperacionDecisionCobertura, error) {
	solicitud := SolicitudRecuperacionResultadoOperacionDecisionCobertura{
		datos: &DatosSolicitudRecuperacionResultadoOperacionDecisionCobertura{
			OrganizacionRef: organizacionRef,
			ExpedienteRef:   expedienteRef,
			AmbitosHMAC:     ambitos,
		},
	}
	if _, err := solicitud.DatosLectura(); err != nil {
		return SolicitudRecuperacionResultadoOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return solicitud, nil
}
