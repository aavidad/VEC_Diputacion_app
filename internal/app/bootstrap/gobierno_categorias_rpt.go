package bootstrap

import (
	"net/http"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/ports"
)

// DependenciasGobiernoCategoriaRPTInterno son capacidades ya verificadas por la
// raíz. Pool pertenece exclusivamente al rol técnico nominal de este gobierno;
// Fuente procede de material privado y no se obtiene de una petición o de Git.
type DependenciasGobiernoCategoriaRPTInterno struct {
	Pool            *pgxpool.Pool
	Descriptor      ports.DescriptorCatalogoRPT
	OrganizacionRef string
	Fuente          postgres.FuenteGobiernoCategoriaRPT
	FuenteAdmitida  FuenteAdmitidaGobiernoCategoriaRPT
	Autorizador     ports.AutorizadorGobiernoCategoriaRPT
	AutoridadSesion httpapi.AutoridadGobiernoCategoriaRPTInterno
	AuditorRechazos httpapi.AuditorRechazoGobiernoCategoriaRPTInterno
	Reloj           ports.Reloj
	VersionesRol    application.VersionesRolGobiernoCategoriaRPT
}

// FuenteAdmitidaGobiernoCategoriaRPT es el descriptor cotejado por la raíz
// contra su configuración privada antes de construir la composición.
type FuenteAdmitidaGobiernoCategoriaRPT struct {
	CatalogoID      string
	ModuloID        string
	SHA256          string
	FuenteRef       string
	Clase           string
	ProcedenciaRef  string
	CustodiaRef     string
	OrganizacionRef string
	VigenteDesde    time.Time
	VigenteHasta    time.Time
}

// ComponerGobiernoCategoriaRPTInterno entrega un handler aislado. El dueño de
// la raíz decide si puede montarlo tras verificar el canal interno y el PDP.
func ComponerGobiernoCategoriaRPTInterno(d DependenciasGobiernoCategoriaRPTInterno) (http.Handler, error) {
	if d.Pool == nil || nuloGobiernoCategoriaRPTBootstrap(d.Autorizador) ||
		nuloGobiernoCategoriaRPTBootstrap(d.AutoridadSesion) ||
		nuloGobiernoCategoriaRPTBootstrap(d.AuditorRechazos) ||
		nuloGobiernoCategoriaRPTBootstrap(d.Reloj) {
		return nil, ports.ErrGobiernoCategoriaRPTNoDisponible
	}
	if d.Descriptor.CatalogoID == "" || d.Descriptor.ModuloID == "" ||
		d.FuenteAdmitida.CatalogoID != d.Descriptor.CatalogoID ||
		d.FuenteAdmitida.ModuloID != d.Descriptor.ModuloID ||
		d.OrganizacionRef == "" || d.Fuente.OrganizacionRef != d.OrganizacionRef ||
		d.FuenteAdmitida.OrganizacionRef != d.OrganizacionRef ||
		d.Fuente.SHA256 != d.FuenteAdmitida.SHA256 ||
		d.Fuente.FuenteRef != d.FuenteAdmitida.FuenteRef ||
		d.Fuente.Clase != d.FuenteAdmitida.Clase ||
		d.Fuente.ProcedenciaRef != d.FuenteAdmitida.ProcedenciaRef ||
		d.Fuente.CustodiaRef != d.FuenteAdmitida.CustodiaRef ||
		!d.Fuente.VigenteDesde.Equal(d.FuenteAdmitida.VigenteDesde) ||
		!d.Fuente.VigenteHasta.Equal(d.FuenteAdmitida.VigenteHasta) {
		return nil, ports.ErrGobiernoCategoriaRPTInvalido
	}
	gestor, err := postgres.NuevoGestorGobiernoCategoriaRPTPostgreSQL(d.Pool, d.Descriptor, d.Fuente)
	if err != nil {
		return nil, err
	}
	servicio, err := application.NuevoServicioGobiernoCategoriaRPTConRoles(gestor, d.Autorizador, gestor, d.Reloj, d.VersionesRol)
	if err != nil {
		return nil, err
	}
	return httpapi.NuevoHandlerGobiernoCategoriaRPTInterno(d.AutoridadSesion, servicio, d.AuditorRechazos)
}

func nuloGobiernoCategoriaRPTBootstrap(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}
