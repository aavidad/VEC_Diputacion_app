package administracion

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type fuenteUsuariosMetadatosPrueba struct {
	persona *ports.UsuarioAdministrable
	filtros ports.FiltrosUsuariosAdministrables
	err     error
}

func (f *fuenteUsuariosMetadatosPrueba) ListarUsuarios(_ context.Context, _ domain.ContextoActor, _ domain.EvidenciaSesionAdministracionPerfiles, filtros ports.FiltrosUsuariosAdministrables) (ports.PaginaUsuariosAdministrables, error) {
	f.filtros = filtros
	return ports.PaginaUsuariosAdministrables{Personas: []ports.UsuarioAdministrable{*f.persona}, SiguienteCursor: "cursor:ensayo"}, f.err
}
func (f *fuenteUsuariosMetadatosPrueba) ConsultarUsuario(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, string) (*ports.UsuarioAdministrable, error) {
	return f.persona, f.err
}

func TestFuenteMetadatosConservaFiltrosVersionesYAusenciaComprobada(t *testing.T) {
	p := ports.UsuarioAdministrable{PersonaRef: "per_" + strings.Repeat("a", 24), UnidadRef: "unidad:ensayo", Perfiles: []ports.PerfilUsuarioAdministrable{{PerfilRef: "prf_" + strings.Repeat("b", 24), RolVersionRef: "rol:administracion_perfiles:v5", Version: 1, Estado: "vigente"}}}
	port := &fuenteUsuariosMetadatosPrueba{persona: &p}
	f, err := NuevaFuenteLecturasUsuariosMetadatos(port)
	if err != nil {
		t.Fatal(err)
	}
	consulta := api.ConsultaPersonas{PerfilRef: p.Perfiles[0].RolVersionRef, UnidadRef: p.UnidadRef, Estado: "vigente", Cursor: "cursor:anterior"}
	pagina, err := f.BuscarPersonas(context.Background(), domain.ContextoActor{}, domain.EvidenciaSesionAdministracionPerfiles{}, consulta)
	if err != nil || port.filtros.PerfilRef != consulta.PerfilRef || port.filtros.UnidadRef != consulta.UnidadRef || port.filtros.Estado != consulta.Estado || port.filtros.Cursor != consulta.Cursor {
		t.Fatal("filtros")
	}
	m := pagina.Metadatos.Personas[0]
	if m.NombreEstado != "no_registrado" || m.DenominacionVersion != nil || m.Nombre != "" || m.Perfiles[0].Version != 1 {
		t.Fatal("metadata")
	}
	v := uint64(2)
	p.DenominacionVersion = &v
	ficha, err := f.ConsultarPersona(context.Background(), domain.ContextoActor{}, domain.EvidenciaSesionAdministracionPerfiles{}, p.PersonaRef)
	if err != nil || ficha.Metadatos.NombreEstado != "no_consultado" || ficha.Metadatos.HistoriaEstado != "no_consultada" {
		t.Fatal("nombre_pendiente")
	}
	v = 9
	if *ficha.Metadatos.DenominacionVersion != 2 {
		t.Fatal("copia")
	}
	b, err := json.Marshal(ficha)
	if err != nil || strings.Contains(string(b), `"historia":`) || strings.Contains(string(b), `"actos_disponibles":`) {
		t.Fatal("campos_fuera_alcance")
	}
}

func TestFuenteMetadatosNoPublicaParcialesYNoDistingueAusenteAjena(t *testing.T) {
	p := ports.UsuarioAdministrable{PersonaRef: "per_" + strings.Repeat("a", 24), UnidadRef: "unidad:ensayo"}
	port := &fuenteUsuariosMetadatosPrueba{persona: &p, err: errors.New("detalle_privado_SECRET")}
	f, _ := NuevaFuenteLecturasUsuariosMetadatos(port)
	salida, err := f.BuscarPersonas(context.Background(), domain.ContextoActor{}, domain.EvidenciaSesionAdministracionPerfiles{}, api.ConsultaPersonas{})
	if err != ports.ErrAutoridadAdministracionPerfilesNoDisponible || salida.Metadatos != nil || strings.Contains(err.Error(), "SECRET") {
		t.Fatal("error_privado")
	}
	port.err = domain.ErrAutorizacionDenegada
	if _, err = f.ConsultarPersona(context.Background(), domain.ContextoActor{}, domain.EvidenciaSesionAdministracionPerfiles{}, p.PersonaRef); err != domain.ErrAutorizacionDenegada {
		t.Fatal("denegacion")
	}
	port.err = nil
	port.persona = nil
	if _, err = f.ConsultarPersona(context.Background(), domain.ContextoActor{}, domain.EvidenciaSesionAdministracionPerfiles{}, p.PersonaRef); err != api.ErrRecursoNoEncontrado {
		t.Fatal("ausencia")
	}
}
