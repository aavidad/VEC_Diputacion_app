package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/personal/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionServiciosParaCertificadosV1    = "personal.servicios_certificados.consultar"
	AudienciaServiciosParaCertificadosV1 = "vec_personal.servicios_certificados.v1"
)

// CoberturaPersonalNominalV1 expresa la certeza sobre el conjunto solicitado.
// Vacía o desconocida nunca equivale a completa; parcial no acredita ausencia.
type CoberturaPersonalNominalV1 string

const (
	CoberturaPersonalCompletaV1     CoberturaPersonalNominalV1 = "completa"
	CoberturaPersonalParcialV1      CoberturaPersonalNominalV1 = "parcial"
	CoberturaPersonalNoAcreditadaV1 CoberturaPersonalNominalV1 = "no_acreditada"
)

type CertezaPersonalNominalV1 string

const (
	CertezaPersonalAcreditadaV1   CertezaPersonalNominalV1 = "acreditado"
	CertezaPersonalPendienteV1    CertezaPersonalNominalV1 = "pendiente"
	CertezaPersonalNoAcreditadaV1 CertezaPersonalNominalV1 = "no_acreditado"
)

// ProcedenciaPersonalNominalV1 identifica el acto y la versión de su fuente.
// Certeza vacía o desconocida no acredita el hecho. Los estados de relación y
// servicio siguen sus catálogos versionados, sin reglas de RRHH.
type ProcedenciaPersonalNominalV1 struct {
	ActoRef, FuenteRef, FuenteVersion string
	Certeza                           CertezaPersonalNominalV1
}

// PeriodoPersonalNominalV1 usa fechas civiles y el intervalo [Desde, Hasta).
// Hasta vacía significa abierto, nunca una fecha inferida.
type PeriodoPersonalNominalV1 struct {
	Desde, Hasta domain.FechaCivil
}

type ServicioParaCertificadosV1 struct {
	ServicioRef, RelacionRef string
	Version                  int64
	Periodo                  PeriodoPersonalNominalV1
	Estado, ClaseRef         string
	ClaseVersion             int64
	Procedencia              ProcedenciaPersonalNominalV1
}

// ConsultaServiciosParaCertificadosV1 recibe el actor efectivo acreditado por
// ContextoActor y el empleado/organismo fijados por el servidor competente.
// Para autoservicio, EmpleadoRef procede de la proyección canónica de ese actor;
// consultar a otra persona exige competencia positiva para ese recurso exacto.
type ConsultaServiciosParaCertificadosV1 struct {
	Actor                     vecdomain.ContextoActor
	EmpleadoRef, OrganismoRef string
	Corte                     domain.CorteEmpleadoB2
}

type ResultadoServiciosParaCertificadosV1 struct {
	EmpleadoRef, OrganismoRef string
	Version                   int64
	Corte                     domain.CorteEmpleadoB2
	Cobertura                 CoberturaPersonalNominalV1
	Servicios                 []ServicioParaCertificadosV1
	// Evidencia corresponde a esta consulta nominal, nunca a una lectura B2.
	Evidencia EvidenciaRegistroEmpleadoB2
}

// LectorServiciosParaCertificadosV1 lo implementa application.ServicioLectorServiciosCertificados
// sobre la fachada Personal36/AD195; su montaje en una ruta lo hace el consumidor.
// La fuente exige concesión central positiva, exacta y vigente para la acción
// y audiencia anteriores, actor, empleado, organismo, corte, finalidad, perfil,
// campos y obligaciones. Revalida y consume esa autorización junto a la lectura
// y auditoría en una transacción propia; registra también accesos denegados.
// El llamante no aporta permisos y ser otro módulo no concede acceso. Un actor
// externo no se transforma en RRHH. Fallos/ambigüedad deniegan sin datos parciales.
// La fuente acota cardinalidad; desbordamiento se informa con error, nunca se
// trunca como cobertura completa. Solo servicios reconocidos y acreditados
// pueden sustentar certificación; un declarado permanece como tal. Este puerto
// no emite certificado, firma ni entrega y no sustituye FuenteServicios de ensayo.
type LectorServiciosParaCertificadosV1 interface {
	ConsultarServiciosParaCertificados(context.Context, ConsultaServiciosParaCertificadosV1) (ResultadoServiciosParaCertificadosV1, error)
}

// ServicioParaCertificadosV2 añade a V1 los días reconocidos que constan en la
// fuente. Son el dato registrado por Personal, no un cómputo: el consumidor no
// los recalcula ni los suma, y un periodo sin días acreditados no se infiere.
type ServicioParaCertificadosV2 struct {
	ServicioParaCertificadosV1
	DiasReconocidos int64
}

type ResultadoServiciosParaCertificadosV2 struct {
	EmpleadoRef, OrganismoRef string
	Version                   int64
	Corte                     domain.CorteEmpleadoB2
	Cobertura                 CoberturaPersonalNominalV1
	Servicios                 []ServicioParaCertificadosV2
	Evidencia                 EvidenciaRegistroEmpleadoB2
}

// V1 proyecta la respuesta V2 sobre el contrato anterior, sin los días. Copia
// la lista para que el consumidor V1 no comparta memoria con la respuesta V2.
func (r ResultadoServiciosParaCertificadosV2) V1() ResultadoServiciosParaCertificadosV1 {
	v := ResultadoServiciosParaCertificadosV1{EmpleadoRef: r.EmpleadoRef, OrganismoRef: r.OrganismoRef, Version: r.Version,
		Corte: r.Corte, Cobertura: r.Cobertura, Evidencia: r.Evidencia, Servicios: make([]ServicioParaCertificadosV1, 0, len(r.Servicios))}
	for _, s := range r.Servicios {
		v.Servicios = append(v.Servicios, s.ServicioParaCertificadosV1)
	}
	return v
}

// LectorServiciosParaCertificadosV2 recibe la misma consulta que V1 y aplica
// sus mismas garantías de autorización, auditoría, cardinalidad y cobertura.
type LectorServiciosParaCertificadosV2 interface {
	ConsultarServiciosParaCertificadosV2(context.Context, ConsultaServiciosParaCertificadosV1) (ResultadoServiciosParaCertificadosV2, error)
}
