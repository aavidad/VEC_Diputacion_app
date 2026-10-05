package administracionperfiles

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestMetadatosRechazaRutasAuxiliaresYBusquedaNombreAntesDeFuente(t *testing.T) {
	for _, ruta := range []string{PrefijoV1 + "/capacidades", PrefijoV1 + "/roles", PrefijoV1 + "/propuestas", PrefijoV1 + "/personas?q=Elena"} {
		t.Run(ruta, func(t *testing.T) {
			s := sesionADMINPrueba(t)
			f := &fuenteUsuariosPrueba{}
			a := &auditorPrueba{}
			h, err := NuevoHandlerUsuariosMetadatos("https://admin.example.test", &sesionPrueba{resultado: s}, f, a)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionADMIN(http.MethodGet, ruta, ""))
			if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest || f.llamadas != 0 {
				t.Fatal("ruta_fuera_alcance")
			}
		})
	}
}

type fichaCompletaEnMetadatosPrueba struct{ lecturasPrueba }

func (f *fichaCompletaEnMetadatosPrueba) ConsultarPersona(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, string) (FichaPersona, error) {
	return FichaPersona{PersonaRef: "per_" + strings.Repeat("g", 22), Nombre: "Nombre reservado"}, nil
}

func TestModoMetadatosNuncaSerializaFichaCompleta(t *testing.T) {
	s := sesionADMINPrueba(t)
	a := &auditorPrueba{}
	h, err := NuevoHandlerUsuariosMetadatos("https://admin.example.test", &sesionPrueba{resultado: s}, &fichaCompletaEnMetadatosPrueba{}, a)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodGet, PrefijoV1+"/personas/per_"+strings.Repeat("g", 22), ""))
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "Nombre reservado") {
		t.Fatal("ficha_completa_filtrada")
	}
}

func TestModoMetadatosAuditaFichaCompletaConReferenciaAjena(t *testing.T) {
	s := sesionADMINPrueba(t)
	a := &auditorPrueba{}
	h, err := NuevoHandlerUsuariosMetadatos("https://admin.example.test", &sesionPrueba{resultado: s}, &fichaCompletaEnMetadatosPrueba{}, a)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodGet, PrefijoV1+"/personas/per_"+strings.Repeat("h", 22), ""))
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "Nombre reservado") || a.llamadas != 1 || a.ultima.Actor.PersonaRef != s.Actor.PersonaRef || a.ultima.Codigo != "respuesta_incompatible" {
		t.Fatal("ficha_incompatible_sin_auditoria")
	}
}

func TestModoMetadatosNuncaSerializaPaginaCompleta(t *testing.T) {
	s := sesionADMINPrueba(t)
	f := &fuenteUsuariosPrueba{}
	a := &auditorPrueba{}
	h, err := NuevoHandlerUsuariosMetadatos("https://admin.example.test", &sesionPrueba{resultado: s}, f, a)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodGet, PrefijoV1+"/personas", ""))
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "Persona sintética") {
		t.Fatal("pagina_completa_filtrada")
	}
}
