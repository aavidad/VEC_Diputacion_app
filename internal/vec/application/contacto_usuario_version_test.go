package application

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

type consultorVersionContadorPrueba struct{ llamadas int }

func (c *consultorVersionContadorPrueba) ConsultarVersionContactoUsuario(context.Context, ports.OrdenVersionContactoUsuario) (ports.ResultadoVersionContactoUsuario, error) {
	c.llamadas++
	return ports.ResultadoVersionContactoUsuario{}, nil
}

func TestSelectorVersionSeparaPropioYLlamamientoAntesDeConsultar(t *testing.T) {
	propio, err := PayloadVersionContactoUsuario(ports.VersionContactoPropia, "per_"+strings.Repeat("a", 22))
	if err != nil {
		t.Fatal(err)
	}
	llamamiento, err := PayloadVersionContactoUsuario(ports.VersionContactoLlamamiento, "per_"+strings.Repeat("a", 22))
	if err != nil {
		t.Fatal(err)
	}
	var p, l map[string]any
	if json.Unmarshal(propio, &p) != nil || json.Unmarshal(llamamiento, &l) != nil || p["FinalidadRef"] != FinalidadVersionContactoPropia || l["FinalidadRef"] != FinalidadVersionContactoLlamamiento || p["Audiencia"] == l["Audiencia"] {
		t.Fatal("contratos de finalidad o audiencia cruzados")
	}
	_, peticion, _, e := entornoConsultaReciboPrueba(t)
	peticion.SolicitudBase.Accion = AccionVersionContactoPropia
	peticion.SolicitudBase.Finalidad = FinalidadVersionContactoPropia
	c := &consultorVersionContadorPrueba{}
	correlacion, _ := peticion.SolicitudBase.Correlacion.ValorCanonico()
	servicio, err := NuevoServicioVersionContactoUsuario(auditorContactoPrueba{e.ahora, FinalidadVersionContactoPropia, correlacion}, e.emisor, c)
	if err != nil {
		t.Fatal(err)
	}
	servicio.ahora = func() time.Time { return e.ahora }
	sujetoAjeno := "per_" + strings.Repeat("z", 22)
	_, err = servicio.Consultar(context.Background(), ports.SolicitudVersionContactoUsuario{Clase: ports.VersionContactoPropia, SujetoRef: sujetoAjeno, ContextoActor: peticion.ContextoActor, Recurso: peticion.Recurso, SolicitudBase: peticion.SolicitudBase, ResultadoContexto: peticion.ResultadoContexto})
	if err == nil || c.llamadas != 0 {
		t.Fatal("selector propio aceptó persona ajena")
	}
	peticion.SolicitudBase.Accion = AccionVersionContactoLlamamiento
	_, err = servicio.Consultar(context.Background(), ports.SolicitudVersionContactoUsuario{Clase: ports.VersionContactoLlamamiento, SujetoRef: peticion.ContextoActor.PersonaRef, ContextoActor: peticion.ContextoActor, Recurso: peticion.Recurso, SolicitudBase: peticion.SolicitudBase, ResultadoContexto: peticion.ResultadoContexto})
	if err == nil || c.llamadas != 0 {
		t.Fatal("selector de RRHH reutilizó finalidad del titular")
	}
}
