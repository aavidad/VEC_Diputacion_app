package httpcopias

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

func TestJSONNombresExactosEnTodasLasEscrituras(t *testing.T) {
	now := time.Now().UTC()
	policy := p.SolicitudPolitica{OperacionRef: "operacion_aaaaaaaa", VersionEsperada: 5, Politica: p.Politica{Formato: 1, Referencia: "politica_aaaaaaaa", Destino: "destino_aaaaaaaa", ZonaHoraria: "Europe/Madrid", FechaInicial: "2026-10-01", CadaDias: 1, Ventana: p.VentanaPolitica{Inicio: "01:00", Fin: "02:00"}, Retencion: p.RetencionPolitica{ConservarMinimo: 2, EdadMaximaDias: 30}}}
	proposal := p.SolicitudPropuesta{OperacionRef: "propuesta_aaaaaaaa", ConjuntoRef: "conjunto_aaaaaaaa", DestinoRef: "destino_aaaaaaaa", MotivoRef: "motivo_aaaaaaaa", VentanaRef: "ventana_aaaaaaaa", VentanaInicio: now.Add(-time.Hour), VentanaFin: now.Add(2 * time.Hour), CaducaEn: now.Add(time.Hour)}
	control := p.SolicitudControl{OperacionRef: "operacion_aaaaaaaa", DestinoRef: "destino_aaaaaaaa", PropuestaHuellaSHA256: strings.Repeat("a", 64), VersionEsperada: 5}
	for _, tc := range []struct {
		ruta  string
		value any
		nuevo func() any
	}{
		{"/lanzamientos", p.SolicitudLanzamiento{OperacionRef: "operacion_aaaaaaaa", VersionEsperada: 5, Tipo: "completa"}, func() any { return &p.SolicitudLanzamiento{} }},
		{"/calendario", policy, func() any { return &p.SolicitudPolitica{} }},
		{"/retencion", policy, func() any { return &p.SolicitudPolitica{} }},
		{"/propuestas", proposal, func() any { return &p.SolicitudPropuesta{} }},
		{"/propuestas/propuesta_aaaaaaaa/revision", control, func() any { return &p.SolicitudControl{} }},
		{"/propuestas/propuesta_aaaaaaaa/ejecucion", control, func() any { return &p.SolicitudControl{} }},
	} {
		t.Run(tc.ruta, func(t *testing.T) {
			data, e := json.Marshal(tc.value)
			if e != nil {
				t.Fatal(e)
			}
			h, _, _ := preparar(t)
			if !h.decode(httptest.NewRecorder(), peticion("POST", PrefijoV1+tc.ruta, string(data)), sesionPrueba(t), tc.nuevo()) {
				t.Fatal("canonical contract rejected")
			}
			var fields map[string]json.RawMessage
			if e := json.Unmarshal(data, &fields); e != nil {
				t.Fatal(e)
			}
			for key, value := range fields {
				alias := strings.ToUpper(key[:1]) + key[1:]
				if alias == key {
					t.Fatal("test field must have lowercase name")
				}
				for _, last := range []bool{false, true} {
					extra := `"` + alias + `":` + string(value)
					body := `{` + extra + `,` + string(data[1:])
					if last {
						body = string(data[:len(data)-1]) + `,` + extra + `}`
					}
					h, a, b := preparar(t)
					w := httptest.NewRecorder()
					h.ServeHTTP(w, peticion("POST", PrefijoV1+tc.ruta, body))
					if w.Code != 400 || a.llamadas != 0 || b.llamadas != 0 {
						t.Fatalf("case alias %s last=%t status=%d authority=%d effects=%d", alias, last, w.Code, a.llamadas, b.llamadas)
					}
				}
			}
		})
	}
}

func TestJSONPoliticaNombresExactosTambienEnOpcionalesYAnidados(t *testing.T) {
	body := `{"operacion_ref":"operacion_aaaaaaaa","version_esperada":5,"politica":{"formato":1,"referencia":"politica_aaaaaaaa","destino":"destino_aaaaaaaa","zona_horaria":"Europe/Madrid","fecha_inicial":"2026-10-01","cada_dias":1,"ventana":{"inicio":"01:00","fin":"02:00"},"retencion":{"conservar_minimo":2,"edad_maxima_dias":30,"borrado_permitido":false}}}`
	for _, c := range []struct{ find, replacement string }{
		{`"cada_dias":1`, `"cada_dias":1,"Cada_dias":2`},
		{`"cada_dias":1`, `"cada_dias":1,"Dias_semana":[0]`},
		{`"inicio":"01:00"`, `"inicio":"01:00","Inicio":"03:00"`},
		{`"borrado_permitido":false`, `"borrado_permitido":false,"Borrado_permitido":true`},
		{`"borrado_permitido":false`, `"borrado_permitido":false,"Doble_control":false`},
		{`"borrado_permitido":false`, `"borrado_permitido":false,"desconocido":true`},
	} {
		for _, ruta := range []string{"/calendario", "/retencion"} {
			h, a, b := preparar(t)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticion("POST", PrefijoV1+ruta, strings.Replace(body, c.find, c.replacement, 1)))
			if w.Code != 400 || a.llamadas != 0 || b.llamadas != 0 {
				t.Fatalf("nested alias accepted: %s, status=%d", c.replacement, w.Code)
			}
		}
	}
}

func TestJSONAliasNoPuedeReescribirCAS(t *testing.T) {
	h, a, b := preparar(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion("POST", PrefijoV1+"/lanzamientos", `{"operacion_ref":"operacion_aaaaaaaa","version_esperada":5,"tipo":"completa","Version_esperada":4}`))
	if w.Code != 400 || a.llamadas != 0 || b.llamadas != 0 {
		t.Fatalf("CAS case alias reached operation: status=%d authority=%d effects=%d", w.Code, a.llamadas, b.llamadas)
	}
}
