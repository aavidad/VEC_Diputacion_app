package ports

import "context"

type EscenarioSaldoPermisos struct {
	Referencia string             `json:"referencia"`
	Dias       []DiaSaldoPermisos `json:"dias"`
}

type SnapshotEnsayoSaldoPermisos struct {
	VersionEsquema  int                      `json:"version_esquema"`
	Demostracion    bool                     `json:"demostracion"`
	PoliticaRef     string                   `json:"politica_ref"`
	PoliticaVersion int64                    `json:"politica_version"`
	PoliticaSHA256  string                   `json:"politica_sha256"`
	SHA256          string                   `json:"-"`
	Escenarios      []EscenarioSaldoPermisos `json:"escenarios"`
}

// LectorSnapshotSaldoPermisos reúne programación, trabajo y concesiones de una
// sola lectura. Un futuro lector durable deberá autorizar y fijar esa instantánea.
type LectorSnapshotSaldoPermisos interface {
	LeerSnapshotSaldoPermisos(context.Context) (SnapshotEnsayoSaldoPermisos, error)
}

type ResultadoEscenarioSaldoPermisos struct {
	Referencia string               `json:"referencia"`
	Dias       []EfectoPermisoSaldo `json:"dias"`
}

type ResultadoEnsayoSaldoPermisos struct {
	Demostracion     bool                              `json:"demostracion"`
	PoliticaRef      string                            `json:"politica_ref"`
	PoliticaVersion  int64                             `json:"politica_version"`
	PoliticaSHA256   string                            `json:"politica_sha256"`
	EscenariosSHA256 string                            `json:"escenarios_sha256"`
	Escenarios       []ResultadoEscenarioSaldoPermisos `json:"escenarios"`
}

type ReglaEfectoPermisoSaldo struct {
	Referencia         string `json:"referencia"`
	PermisoRef         string `json:"permiso_ref"`
	CatalogoVersionRef string `json:"catalogo_version_ref"`
	ColectivoRef       string `json:"colectivo_ref"`
	Efecto             string `json:"efecto"`
	FuenteRef          string `json:"fuente_ref"`
	FuenteURL          string `json:"fuente_url"`
	VigenteDesde       string `json:"vigente_desde"`
	HastaExclusivo     string `json:"hasta_exclusivo"`
	FuenteVersionRef   string `json:"fuente_version_ref"`
	FuentePublicadaEn  string `json:"fuente_publicada_en"`
	FuenteConsultadaEn string `json:"fuente_consultada_en"`
}
type ProgramacionDiaPermisoSaldo struct {
	Fecha              string `json:"fecha"`
	Referencia         string `json:"referencia"`
	PoliticaVersionRef string `json:"politica_version_ref"`
	FuenteRef          string `json:"fuente_ref"`
	MinutosPrevistos   *int64 `json:"minutos_previstos"`
	Acreditada         bool   `json:"acreditada"`
}
type ConcesionPermisoSaldo struct {
	SolicitudRef       string `json:"solicitud_ref"`
	ResolucionRef      string `json:"resolucion_ref"`
	PermisoRef         string `json:"permiso_ref"`
	CatalogoVersionRef string `json:"catalogo_version_ref"`
	ColectivoRef       string `json:"colectivo_ref"`
	Desde              string `json:"desde"`
	Hasta              string `json:"hasta"`
	Concedido          bool   `json:"concedido"`
	JornadaCompleta    *bool  `json:"jornada_completa"`
}
type DiaSaldoPermisos struct {
	// SinTrabajo se deriva de duraciones exactas, nunca de minutos truncados.
	SinTrabajo        *bool                        `json:"sin_trabajo"`
	Fecha             string                       `json:"fecha"`
	Programacion      *ProgramacionDiaPermisoSaldo `json:"programacion"`
	TrabajadosMinutos *int64                       `json:"trabajados_minutos"`
	Completo          *bool                        `json:"completo"`
	SinAnomalias      *bool                        `json:"sin_anomalias"`
	Permisos          []ConcesionPermisoSaldo      `json:"permisos"`
}
type EfectoPermisoSaldo struct {
	VersionProgramacionRef  string `json:"version_programacion_ref,omitempty"`
	FuenteProgramacionRef   string `json:"fuente_programacion_ref,omitempty"`
	Fecha                   string `json:"fecha"`
	TrabajadosMinutos       *int64 `json:"trabajados_minutos"`
	SaldoBaseMinutos        *int64 `json:"saldo_base_minutos"`
	PermisoComputadoMinutos *int64 `json:"permiso_computado_minutos"`
	SaldoAjustadoMinutos    *int64 `json:"saldo_ajustado_minutos"`
	Causa                   string `json:"causa"`
	ProgramacionRef         string `json:"programacion_ref,omitempty"`
	SolicitudRef            string `json:"solicitud_ref,omitempty"`
	ResolucionRef           string `json:"resolucion_ref,omitempty"`
	ReglaRef                string `json:"regla_ref,omitempty"`
	FuenteRef               string `json:"fuente_ref,omitempty"`
}
