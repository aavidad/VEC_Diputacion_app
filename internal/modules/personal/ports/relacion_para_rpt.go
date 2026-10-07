package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/personal/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionRelacionParaRPTV1    = "personal.relacion_rpt.consultar"
	AudienciaRelacionParaRPTV1 = "vec_personal.relacion_rpt.v1"
)

// ConsultaRelacionParaRPTV1 fija una relación opaca, no una lista de personas.
// Actor es el contexto efectivo acreditado; las referencias y organismo los
// resuelve el servidor competente. VersionEsperada debe ser positiva: enlaza
// la versión concreta que M necesita conciliar, sin aceptar su sustitución.
type ConsultaRelacionParaRPTV1 struct {
	Actor                                  vecdomain.ContextoActor
	EmpleadoRef, RelacionRef, OrganismoRef string
	VersionEsperada                        int64
	Corte                                  domain.CorteEmpleadoB2
}

type RelacionParaRPTV1 struct {
	EmpleadoRef, RelacionRef, OrganismoRef string
	Version                                int64
	Estado                                 string
	Periodo                                PeriodoPersonalNominalV1
	Procedencia                            ProcedenciaPersonalNominalV1
}

type ResultadoRelacionParaRPTV1 struct {
	Relacion  RelacionParaRPTV1
	Corte     domain.CorteEmpleadoB2
	Cobertura CoberturaPersonalNominalV1
	Evidencia EvidenciaRegistroEmpleadoB2
}

// LectorRelacionParaRPTV1 exige concesión central positiva y vigente para su
// acción/audiencia, actor, relación, empleado, organismo, versión, corte,
// finalidad, perfil, campos y obligaciones. La fuente revalida y consume esa
// autorización en la transacción de lectura y auditoría propia; registra también
// denegaciones. Una concesión CER/H05/B2 o identidad de módulo no sirve aquí.
// ContextoActor externo conserva su perfil; nunca se copia como RRHH. Error,
// ambigüedad, versión distinta o autoridad ausente devuelven error sin datos.
// Personal aporta el hecho laboral. M gobierna ocupación, reserva y vacantes;
// este contrato preparado no decide disponibilidad ni monta una consulta.
type LectorRelacionParaRPTV1 interface {
	ConsultarRelacionParaRPT(context.Context, ConsultaRelacionParaRPTV1) (ResultadoRelacionParaRPTV1, error)
}
