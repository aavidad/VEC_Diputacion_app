package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/personal/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionAntecedentesCarreraV1    = "personal.antecedentes_carrera.consultar"
	AudienciaAntecedentesCarreraV1 = "vec_personal.antecedentes_carrera.v1"
)

// Las referencias de ConsultaAntecedentesCarreraV1 las fija el servidor
// competente para el Actor efectivo acreditado. No implica autoservicio ni
// competencia RRHH; ambos requieren concesión positiva para ese empleado.
type ConsultaAntecedentesCarreraV1 struct {
	Actor                     vecdomain.ContextoActor
	EmpleadoRef, OrganismoRef string
	Corte                     domain.CorteEmpleadoB2
}

type RelacionAntecedenteCarreraV1 struct {
	RelacionRef        string
	Version            int64
	Periodo            PeriodoPersonalNominalV1
	Estado, RegimenRef string
	RegimenVersion     int64
	Procedencia        ProcedenciaPersonalNominalV1
}

// CodigoRef/CodigoVersion identifican la entrada del catálogo de situaciones
// de Personal que conserva el hecho fuente. Si la fuente no acredita esa versión,
// CodigoVersion queda cero y Procedencia.Certeza pendiente; nunca se inventa.
// Estado conserva el estado del hecho fuente, incluida su rectificación.
type SituacionAntecedenteCarreraV1 struct {
	SituacionRef, RelacionRef string
	Version                   int64
	CodigoRef, Estado         string
	CodigoVersion             int64
	Periodo                   PeriodoPersonalNominalV1
	Procedencia               ProcedenciaPersonalNominalV1
}

// PuestoAntecedenteCarreraV1 conserva procedencia RPT recibida de M. Nivel
// ausente queda nil; no se infiere del puesto ni se convierte en grado personal.
type PuestoAntecedenteCarreraV1 struct {
	PuestoRef, PuestoVersion, RelacionRef string
	Periodo                               PeriodoPersonalNominalV1
	Nivel                                 *int
	Procedencia                           ProcedenciaPersonalNominalV1
}

type ResultadoAntecedentesCarreraV1 struct {
	EmpleadoRef, OrganismoRef string
	Version                   int64
	Corte                     domain.CorteEmpleadoB2
	Cobertura                 CoberturaPersonalNominalV1
	Relaciones                []RelacionAntecedenteCarreraV1
	Servicios                 []ServicioParaCertificadosV1
	Situaciones               []SituacionAntecedenteCarreraV1
	Puestos                   []PuestoAntecedenteCarreraV1
	Evidencia                 EvidenciaRegistroEmpleadoB2
}

// LectorAntecedentesCarreraV1 exige concesión central positiva, exacta y vigente
// para su acción/audiencia, actor, empleado, organismo, corte, finalidad, perfil,
// campos y obligaciones. La fuente revalida y consume esa autorización junto a
// lectura y auditoría propias en una transacción; registra también denegaciones.
// No acepta permisos aportados por el llamante ni concede acceso por módulo;
// conserva el perfil efectivo de ContextoActor, sin elevar un externo a RRHH.
// Fallo, ambigüedad o autoridad ausente producen error sin antecedentes. La fuente
// acota cardinalidad y rechaza desbordamientos, sin truncar cobertura completa.
// H decide Carrera y gobierna Méritos. Este contrato preparado no calcula grado,
// trienios ni puntos y no sustituye LectorAntecedentesSinteticos de ensayo.
type LectorAntecedentesCarreraV1 interface {
	ConsultarAntecedentesCarrera(context.Context, ConsultaAntecedentesCarreraV1) (ResultadoAntecedentesCarreraV1, error)
}
