package administracionperfiles

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

type fuenteUsuariosPrueba struct {
	lecturasPrueba
	consulta  ConsultaPersonas
	actor     domain.ContextoActor
	evidencia domain.EvidenciaSesionAdministracionPerfiles
	err       error
}

func (f *fuenteUsuariosPrueba) BuscarPersonas(_ context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles, consulta ConsultaPersonas) (PaginaPersonas, error) {
	f.llamadas++
	f.consulta, f.actor, f.evidencia = consulta, actor, evidencia
	// Incluso un resultado parcial junto con un error debe quedar privado.
	return PaginaPersonas{Personas: []Persona{{PersonaRef: "per_" + strings.Repeat("f", 22), Nombre: "Persona sintética"}}, SiguienteCursor: "cursor_fuente"}, f.err
}

func handlerUsuariosPrueba(t *testing.T, f FuenteLecturas, aud *auditorPrueba) (*Handler, SesionConfiable) {
	t.Helper()
	s := sesionADMINPrueba(t)
	h, err := NuevoHandlerLecturas("https://admin.example.test", &sesionPrueba{resultado: s}, f, aud)
	if err != nil {
		t.Fatal(err)
	}
	return h, s
}

func TestConsultaUsuariosPropagaTodosFiltrosYContextoConfiable(t *testing.T) {
	f := &fuenteUsuariosPrueba{}
	h, s := handlerUsuariosPrueba(t, f, &auditorPrueba{})
	ruta := PrefijoV1 + "/personas?q=Persona&cursor=cursor_anterior&perfil_ref=rol%3Adietas_liquidacion_rrhh%3Av1&unidad_ref=unidad%3Aprueba&estado=caducado"
	r := peticionADMIN(http.MethodGet, ruta, "")
	r.Header.Set("X-Role", "sistemas")
	r.Header.Set("X-Persona-Ref", "per_no_confiable")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	esperado := ConsultaPersonas{Texto: "Persona", Cursor: "cursor_anterior", PerfilRef: "rol:dietas_liquidacion_rrhh:v1", UnidadRef: "unidad:prueba", Estado: "caducado"}
	if w.Code != http.StatusOK || f.llamadas != 1 || f.consulta != esperado || f.actor.PersonaRef != s.Actor.PersonaRef || f.actor.PerfilActivoRef != s.Actor.PerfilActivoRef || f.evidencia.ValidarPara(s.Actor) != nil {
		t.Fatalf("estado=%d llamadas=%d consulta=%+v", w.Code, f.llamadas, f.consulta)
	}
	var pagina PaginaPersonas
	if json.Unmarshal(w.Body.Bytes(), &pagina) != nil || pagina.SiguienteCursor != "cursor_fuente" || len(pagina.Personas) != 1 {
		t.Fatal("página no procede de la fuente")
	}
}

func TestListadoUsuariosVacioExigeFuenteAutorizada(t *testing.T) {
	for _, err := range []error{nil, ErrAccesoDenegado, domain.ErrAutorizacionDenegada} {
		f := &fuenteUsuariosPrueba{err: err}
		h, _ := handlerUsuariosPrueba(t, f, &auditorPrueba{})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionADMIN(http.MethodGet, PrefijoV1+"/personas", ""))
		esperado := http.StatusOK
		if err != nil {
			esperado = http.StatusForbidden
		}
		if w.Code != esperado || f.llamadas != 1 || f.consulta != (ConsultaPersonas{}) {
			t.Fatalf("estado=%d llamadas=%d", w.Code, f.llamadas)
		}
		if err != nil && strings.Contains(w.Body.String(), "Persona sintética") {
			t.Fatal("denegación filtró datos")
		}
	}
}

func TestFiltrosUsuariosInvalidosSeAuditanAntesDeFuente(t *testing.T) {
	for _, consulta := range []string{
		"q=Persona&desconocido=1", "q=Persona&actor=otro", "q=Persona&q=Otra", "q=", "q=x", "q=%20%20", "q=%00xx", "q=%FFxx", "q=%zz", "q=Persona;estado=vigente",
		"estado=activo", "estado=*", "estado=vigente&estado=caducado", "perfil_ref=*", "perfil_ref=prf_" + strings.Repeat("a", 22), "unidad_ref=*", "unidad_ref=unidad:", "unidad_ref=unidad:../otra", "cursor=", "cursor=" + strings.Repeat("x", 257), "q=" + strings.Repeat("x", 133),
	} {
		t.Run(consulta, func(t *testing.T) {
			f := &fuenteUsuariosPrueba{}
			aud := &auditorPrueba{}
			h, s := handlerUsuariosPrueba(t, f, aud)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionADMIN(http.MethodGet, PrefijoV1+"/personas?"+consulta, ""))
			if w.Code != http.StatusBadRequest || f.llamadas != 0 || aud.llamadas != 1 || aud.ultima.ActorPersonaRef != s.Actor.PersonaRef || aud.ultima.Accion != "buscar_personas" || aud.ultima.CorrelacionRef != s.CorrelacionRef {
				t.Fatalf("estado=%d llamadas=%d auditoria=%+v", w.Code, f.llamadas, aud.ultima)
			}
		})
	}
}

func TestFiltrosInvalidosFallaCerradoSiAuditoriaNoDisponible(t *testing.T) {
	f := &fuenteUsuariosPrueba{}
	aud := &auditorPrueba{err: errors.New("fallo privado")}
	h, _ := handlerUsuariosPrueba(t, f, aud)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodGet, PrefijoV1+"/personas?estado=*", ""))
	if w.Code != http.StatusServiceUnavailable || f.llamadas != 0 || strings.Contains(w.Body.String(), "fallo privado") {
		t.Fatal("fallo abierto o datos de error expuestos")
	}
}

func TestParseQueryPropagaErrorYNoPublicaConsultaParcial(t *testing.T) {
	const consulta = "q=fragmento_privado_de_prueba&cursor=%zz"
	filtros, err := consultaPersonas(consulta)
	var escape url.EscapeError
	if !errors.As(err, &escape) || filtros != (ConsultaPersonas{}) {
		t.Fatal("error del parser reducido o consulta parcial devuelta")
	}
	fuente, auditor := &fuenteUsuariosPrueba{}, &auditorPrueba{}
	h, _ := handlerUsuariosPrueba(t, fuente, auditor)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodGet, PrefijoV1+"/personas?"+consulta, ""))
	if w.Code != http.StatusBadRequest || fuente.llamadas != 0 || auditor.llamadas != 1 ||
		auditor.ultima.Codigo != "solicitud_invalida" || auditor.ultima.Accion != "buscar_personas" ||
		auditor.ultima.RecursoRef != "" || strings.Contains(w.Body.String(), "fragmento_privado") || strings.Contains(w.Body.String(), "%zz") {
		t.Fatal("fallo del parser sin denegación auditada o con datos de consulta")
	}
}

func TestFuenteDeLecturasDeniegaSistemasYErroresSinDatos(t *testing.T) {
	for _, err := range []error{ErrAccesoDenegado, domain.ErrAutorizacionDenegada, domain.ErrActoAdministracionPerfilesInvalido, errors.New("datos privados de conexión")} {
		f := &fuenteUsuariosPrueba{err: err}
		h, _ := handlerUsuariosPrueba(t, f, &auditorPrueba{})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionADMIN(http.MethodGet, PrefijoV1+"/personas?q=Persona", ""))
		estado := http.StatusForbidden
		if err == domain.ErrActoAdministracionPerfilesInvalido {
			estado = http.StatusBadRequest
		} else if err != ErrAccesoDenegado && err != domain.ErrAutorizacionDenegada {
			estado = http.StatusServiceUnavailable
		}
		if w.Code != estado || strings.Contains(w.Body.String(), "Persona sintética") || strings.Contains(w.Body.String(), "conexión") {
			t.Fatalf("estado=%d cuerpo=%s", w.Code, w.Body.String())
		}
	}
}

func TestHandlerLecturasCierraEscriturasSinProveedor(t *testing.T) {
	f := &fuenteUsuariosPrueba{}
	aud := &auditorPrueba{}
	h, _ := handlerUsuariosPrueba(t, f, aud)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodPost, PrefijoV1+"/actos-ordinarios", `{"operacion":"otorgar"}`))
	if w.Code != http.StatusServiceUnavailable || f.llamadas != 0 || aud.llamadas != 1 || aud.ultima.Accion != "escribir" {
		t.Fatalf("estado=%d lecturas=%d auditoria=%+v", w.Code, f.llamadas, aud.ultima)
	}
}

func TestLecturasRechazanCuerpoDeLongitudDesconocida(t *testing.T) {
	f := &fuenteUsuariosPrueba{}
	h, _ := handlerUsuariosPrueba(t, f, &auditorPrueba{})
	r := peticionADMIN(http.MethodGet, PrefijoV1+"/personas", `{"actor":"otro"}`)
	r.ContentLength = -1
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || f.llamadas != 0 {
		t.Fatalf("estado=%d lecturas=%d", w.Code, f.llamadas)
	}
}

func TestConstructorLecturasExigeTodasFuentes(t *testing.T) {
	s := &sesionPrueba{}
	f := &fuenteUsuariosPrueba{}
	a := &auditorPrueba{}
	var fNula *fuenteUsuariosPrueba
	for _, caso := range []struct {
		sesion  ResolvedorSesion
		fuente  FuenteLecturas
		auditor AuditorFrontera
	}{
		{nil, f, a}, {s, nil, a}, {s, fNula, a}, {s, f, nil},
	} {
		if _, err := NuevoHandlerLecturas("https://admin.example.test", caso.sesion, caso.fuente, caso.auditor); err == nil {
			t.Fatal("dependencia ausente aceptada")
		}
	}
}

// Una referencia cruzada del proveedor tampoco debe salir por HTTP.
type fichaCruzadaPrueba struct{ lecturasPrueba }

func (f *fichaCruzadaPrueba) ConsultarPersona(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, string) (FichaPersona, error) {
	return FichaPersona{PersonaRef: "per_" + strings.Repeat("f", 22), Nombre: "Nombre reservado"}, nil
}
func TestFichaUsuariosRechazaPersonaCruzadaDeFuente(t *testing.T) {
	h, _ := handlerUsuariosPrueba(t, &fichaCruzadaPrueba{}, &auditorPrueba{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodGet, PrefijoV1+"/personas/per_"+strings.Repeat("g", 22), ""))
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "Nombre reservado") {
		t.Fatalf("estado=%d cuerpo=%s", w.Code, w.Body.String())
	}
}
