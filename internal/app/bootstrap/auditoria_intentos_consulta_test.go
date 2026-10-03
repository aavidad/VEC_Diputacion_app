package bootstrap

import (
	"context"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/auditoria"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type registradorIntentosConsultaPrueba struct{}

func (*registradorIntentosConsultaPrueba) AppendIntentoAuditoria(context.Context, vecports.OrdenIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
	return vecports.AcuseIntentoAuditoria{}, vecports.ErrIntentoAuditoriaNoDisponible
}

func configuracionIntentosConsultaPrueba(t *testing.T, motivo vecdomain.ReferenciaEntradaCatalogo) auditoria.ConfiguracionIntentosConsulta {
	t.Helper()
	if motivo.Validar() != nil {
		escenario := nuevoEscenarioMaterialRutasDietasPrueba(t, "dietas.ruta.catalogo.consultar")
		motivo = escenario.motivo
	}
	return auditoria.ConfiguracionIntentosConsulta{
		Proceso: "vec-rrhh", Canal: string(vecdomain.SuperficieAutenticacionInternaCorporativaV1),
		RecursoCTRef: "expediente:opaco:123", RecursoBolsaRef: "participacion:opaca:123",
		FinalidadRef: "revision_administrativa_auditoria_rrhh", Motivo: motivo, Plazo: time.Second,
	}
}
