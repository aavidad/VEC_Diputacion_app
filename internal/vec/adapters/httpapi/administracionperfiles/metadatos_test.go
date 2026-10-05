package administracionperfiles

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func fichaMetadatosPrueba() FichaPersonaMetadatos {
	return FichaPersonaMetadatos{Proyeccion: "metadatos_v1", PersonaMetadatos: PersonaMetadatos{PersonaRef: "per_" + strings.Repeat("g", 22), UnidadRef: "unidad:ensayo", NombreEstado: "no_registrado", Perfiles: []PerfilUsuarioMetadatos{}}, HistoriaEstado: "no_consultada", ActosEstado: "no_consultados"}
}

func TestMetadatosNoSerializanNombresHistoriaOActosDelModeloCompleto(t *testing.T) {
	m := fichaMetadatosPrueba()
	f := FichaPersona{PersonaRef: m.PersonaRef, Nombre: "Nombre fuera del alcance", Historia: []Historia{}, ActosDisponibles: []ActoDisponible{}, Metadatos: &m}
	b, err := json.Marshal(f)
	if err != nil || strings.Contains(string(b), "Nombre fuera") || strings.Contains(string(b), `"historia":`) || strings.Contains(string(b), `"actos_disponibles":`) || !strings.Contains(string(b), `"denominacion_version":null`) {
		t.Fatal("proyeccion_incompatible")
	}
	var fJSON map[string]any
	if json.Unmarshal(b, &fJSON) != nil || fJSON["historia_estado"] != "no_consultada" {
		t.Fatal("historia_no_consultada")
	}
}

type fuenteMetadatosPrueba struct {
	lecturasPrueba
	m FichaPersonaMetadatos
}

func (f *fuenteMetadatosPrueba) ConsultarPersona(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, string) (FichaPersona, error) {
	return FichaPersona{PersonaRef: "per_" + strings.Repeat("g", 22), Metadatos: &f.m}, nil
}

func TestHTTPMetadatosRechazaNombreOReferenciaFueraDeLaProyeccion(t *testing.T) {
	for _, caso := range []string{"nombre", "referencia", "ausencia", "valida"} {
		t.Run(caso, func(t *testing.T) {
			m := fichaMetadatosPrueba()
			switch caso {
			case "nombre":
				m.Nombre = "Nombre reservado"
			case "referencia":
				m.PersonaRef = "per_" + strings.Repeat("f", 22)
			case "ausencia":
				v := uint64(1)
				m.DenominacionVersion = &v
			}
			h, _ := handlerUsuariosPrueba(t, &fuenteMetadatosPrueba{m: m}, &auditorPrueba{})
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionADMIN(http.MethodGet, PrefijoV1+"/personas/per_"+strings.Repeat("g", 22), ""))
			esperado := http.StatusServiceUnavailable
			if caso == "valida" {
				esperado = http.StatusOK
			}
			if w.Code != esperado || strings.Contains(w.Body.String(), "Nombre reservado") {
				t.Fatal("metadatos_abiertos")
			}
		})
	}
}
