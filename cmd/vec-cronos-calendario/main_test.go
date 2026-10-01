package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	cal "vec-diputacion-granada/internal/modules/calendarios/domain"
	calports "vec-diputacion-granada/internal/modules/calendarios/ports"
	"vec-diputacion-granada/web"
)

func fixture(t *testing.T, nombre string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + nombre + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func resultado(t *testing.T, b []byte) (salida, []byte) {
	t.Helper()
	var out, diagnostico bytes.Buffer
	if code := ejecutar(bytes.NewReader(b), &out, &diagnostico, 0); code != 0 {
		t.Fatalf("code=%d diagnostico=%s", code, diagnostico.Bytes())
	}
	var s salida
	if err := json.Unmarshal(out.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	return s, out.Bytes()
}

func TestEscenariosHistoricos(t *testing.T) {
	for _, tt := range []struct {
		nombre, estado string
		tramos         int
		carencia       string
	}{
		{"cambio-centro", "determinado", 2, ""},
		{"hueco", "parcial", 3, "adscripcion_sin_cobertura"},
		{"solape", "parcial", 3, "adscripcion_multiple"},
		{"falta-calendario", "parcial", 2, "calendario_sin_cobertura"},
		{"cruce-anio", "determinado", 2, ""},
		{"conocimiento-corregido", "determinado", 2, ""},
	} {
		t.Run(tt.nombre, func(t *testing.T) {
			b := fixture(t, tt.nombre)
			s, primera := resultado(t, b)
			_, segunda := resultado(t, b)
			if !bytes.Equal(primera, segunda) {
				t.Fatal("replay")
			}
			if !s.Demostracion || s.Estado != tt.estado || len(s.Tramos) != tt.tramos {
				t.Fatalf("estado=%s tramos=%d", s.Estado, len(s.Tramos))
			}
			if s.PersonaProgramada != nil || s.MinutosTeoricos != nil {
				t.Fatal("programacion")
			}
			if len(s.HuellaEntrada) != 64 || len(s.HuellaSnapshot) != 64 {
				t.Fatal("huellas")
			}
			carencia := false
			fechas := map[string]bool{}
			for i, tramo := range s.Tramos {
				if i > 0 && !tramo.Desde.Igual(s.Tramos[i-1].Hasta) {
					t.Fatal("frontera")
				}
				for _, c := range tramo.Carencias {
					carencia = carencia || c == tt.carencia
				}
				if tramo.Estado != "determinado" {
					if tramo.Calendario != nil {
						t.Fatal("indeterminado_con_calendario")
					}
					continue
				}
				for _, d := range tramo.Calendario.Dias {
					if fechas[d.Fecha.String()] || d.Fecha.Antes(tramo.Desde) || !d.Fecha.Antes(tramo.Hasta) {
						t.Fatal("fecha")
					}
					fechas[d.Fecha.String()] = true
				}
			}
			if tt.carencia != "" && !carencia {
				t.Fatal(tt.carencia)
			}
			if tt.nombre == "falta-calendario" && (len(s.Tramos[1].AmbitosSinCalendario) != 1 || s.Tramos[1].AmbitosSinCalendario[0].Ref != "motril") {
				t.Fatal("ambito")
			}
			if tt.estado == "determinado" {
				for f := s.Desde; f.Antes(s.Hasta); f, _ = f.SumarDias(1) {
					if !fechas[f.String()] {
						t.Fatal(f.String())
					}
				}
			}
		})
	}
}

func TestFestivoLocalSigueCentroYConocimiento(t *testing.T) {
	anterior, _ := resultado(t, fixture(t, "cambio-centro"))
	corregido, _ := resultado(t, fixture(t, "conocimiento-corregido"))
	laborables := func(s salida) map[string]bool {
		dias := map[string]bool{}
		for _, tramo := range s.Tramos {
			for _, d := range tramo.Calendario.Dias {
				dias[d.Fecha.String()] = d.Laborable
			}
		}
		return dias
	}
	a, c := laborables(anterior), laborables(corregido)
	if a["2026-06-04"] || !a["2026-06-03"] || a["2026-06-08"] || !a["2026-06-05"] || !c["2026-06-04"] || c["2026-06-03"] {
		t.Fatalf("anterior=%v corregido=%v", a, c)
	}
	if anterior.HuellaEntrada == corregido.HuellaEntrada || anterior.HuellaSnapshot == corregido.HuellaSnapshot {
		t.Fatal("huella_corregida")
	}
}

func TestEntradaCerrada(t *testing.T) {
	b := fixture(t, "cambio-centro")
	var original map[string]any
	if err := json.Unmarshal(b, &original); err != nil {
		t.Fatal(err)
	}
	mutar := func(f func(map[string]any)) []byte {
		var x map[string]any
		_ = json.Unmarshal(b, &x)
		f(x)
		r, err := json.Marshal(x)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	for nombre, entrada := range map[string][]byte{
		"no_demo": mutar(func(x map[string]any) { x["demostracion"] = false }),
		"no_sintetico": mutar(func(x map[string]any) {
			x["solicitud"].(map[string]any)["snapshot"].(map[string]any)["sintetico"] = false
		}),
		"desconocido":       mutar(func(x map[string]any) { x["http"] = true }),
		"instante_distinto": mutar(func(x map[string]any) { x["calendarios"].(map[string]any)["conocido_en"] = "2026-07-01T12:00:01Z" }),
		"snapshot_distinto": mutar(func(x map[string]any) {
			x["solicitud"].(map[string]any)["snapshot"].(map[string]any)["conocido_en"] = "2026-07-01T12:00:01Z"
		}),
		"fuente_url": mutar(func(x map[string]any) {
			x["calendarios"].(map[string]any)["versiones"].([]any)[0].(map[string]any)["fuente"].(map[string]any)["referencia"] = "https://example.invalid"
		}),
		"sha_invalido": mutar(func(x map[string]any) {
			x["solicitud"].(map[string]any)["snapshot"].(map[string]any)["fuente"].(map[string]any)["sha256"] = "00"
		}),
		"versiones_duplicadas": mutar(func(x map[string]any) {
			c := x["calendarios"].(map[string]any)
			vs := c["versiones"].([]any)
			c["versiones"] = append(vs, vs[0])
		}),
		"claves_duplicadas": []byte(`{"demostracion":true,"demostracion":false}`),
		"varios_json":       append(append([]byte(nil), b...), b...),
		"exceso":            []byte(strings.Repeat(" ", maximoEntrada+1)),
		"profundidad":       []byte(strings.Repeat("[", 34) + "0" + strings.Repeat("]", 34)),
	} {
		t.Run(nombre, func(t *testing.T) {
			var out, diag bytes.Buffer
			if ejecutar(bytes.NewReader(entrada), &out, &diag, 0) == 0 || out.Len() != 0 || !json.Valid(diag.Bytes()) {
				t.Fatalf("out=%s diag=%s", out.Bytes(), diag.Bytes())
			}
		})
	}
}

func TestRepositorioExigeConocimientoExacto(t *testing.T) {
	e, _, err := leerEntrada(bytes.NewReader(fixture(t, "cambio-centro")))
	if err != nil {
		t.Fatal(err)
	}
	r := &repositorioEnsayo{conocido: e.Calendarios.ConocidoEn, versiones: []cal.VersionConDias{e.Calendarios.Versiones[0].Calendario}}
	q := calports.ConsultaVersiones{Anio: 2026, ConocidoEn: r.conocido.Add(time.Second), Ambitos: []cal.Ambito{{Tipo: cal.AmbitoNacional, Ref: "es"}}}
	if _, err = r.VersionesVigentes(context.Background(), q); err == nil {
		t.Fatal("conocimiento")
	}
	q.ConocidoEn = r.conocido
	vs, err := r.VersionesVigentes(context.Background(), q)
	if err != nil || len(vs) != 1 {
		t.Fatalf("vs=%v err=%v", vs, err)
	}
}

func TestEntradaRechazaAliasNoCanonicos(t *testing.T) {
	b := string(fixture(t, "cambio-centro"))
	for nombre, entrada := range map[string]string{
		"demo_alias_despues":      strings.Replace(b, `"demostracion": true`, `"demostracion": false, "Demostracion": true`, 1),
		"demo_alias_antes":        strings.Replace(b, `"demostracion": true`, `"Demostracion": true, "demostracion": false`, 1),
		"demo_solo_alias":         strings.Replace(b, `"demostracion": true`, `"Demostracion": true`, 1),
		"hasta_alias_despues":     strings.Replace(b, `"hasta": "2026-06-10"`, `"hasta": "2026-06-10", "HASTA": "2026-06-09"`, 1),
		"hasta_alias_antes":       strings.Replace(b, `"hasta": "2026-06-10"`, `"HASTA": "2026-06-09", "hasta": "2026-06-10"`, 1),
		"hasta_solo_alias":        strings.Replace(b, `"hasta": "2026-06-10"`, `"HASTA": "2026-06-10"`, 1),
		"solicitud_alias_unicode": strings.Replace(b, `"solicitud":`, `"ſolicitud":`, 1),
	} {
		t.Run(nombre, func(t *testing.T) {
			var out, diag bytes.Buffer
			if ejecutar(strings.NewReader(entrada), &out, &diag, 0) == 0 || out.Len() != 0 || !json.Valid(diag.Bytes()) {
				t.Fatalf("out=%s diag=%s", out.Bytes(), diag.Bytes())
			}
			var errorSalida struct {
				Error struct {
					Codigo string `json:"codigo"`
				} `json:"error"`
			}
			if json.Unmarshal(diag.Bytes(), &errorSalida) != nil || errorSalida.Error.Codigo != "entrada_invalida" {
				t.Fatal(diag.String())
			}
		})
	}
}

func TestCatalogoYReferenciasNoAbrenDocumentos(t *testing.T) {
	b := fixture(t, "cambio-centro")
	e, _, err := leerEntrada(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	// Una referencia opaca inexistente es metadato; no se busca en disco o red.
	e.Calendarios.Versiones[0].Fuente.Referencia = "fuente:inexistente"
	e.Idioma = "en"
	modificado, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := resultado(t, modificado)
	if s.Textos["titulo"] == "" || !strings.Contains(s.Textos["titulo"], "synthetic") {
		t.Fatal("catalogo")
	}
}

func TestCatalogoEnsayoUsaIndiceComunYDefault(t *testing.T) {
	catalogo, mensajes, err := web.CatalogoCronosCalendarioEnsayo()
	if err != nil {
		t.Fatal(err)
	}
	indiceJSON, err := os.ReadFile("../../web/static/textos/idiomas.json")
	if err != nil {
		t.Fatal(err)
	}
	var indice struct {
		PorDefecto string `json:"por_defecto"`
		Idiomas    []struct {
			Codigo string `json:"codigo"`
		} `json:"idiomas"`
	}
	if err := json.Unmarshal(indiceJSON, &indice); err != nil || catalogo.DefaultLocale() != indice.PorDefecto {
		t.Fatalf("default=%s err=%v", catalogo.DefaultLocale(), err)
	}
	if len(catalogo.Locales()) != len(indice.Idiomas) {
		t.Fatal(catalogo.Locales())
	}
	var entrada map[string]any
	if err := json.Unmarshal(fixture(t, "cambio-centro"), &entrada); err != nil {
		t.Fatal(err)
	}
	delete(entrada, "idioma")
	b, err := json.Marshal(entrada)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := resultado(t, b)
	if s.Idioma != indice.PorDefecto || s.Textos["titulo"] != catalogo.T(indice.PorDefecto, "titulo") {
		t.Fatal(s.Idioma)
	}
	for _, idioma := range indice.Idiomas {
		traducciones := mensajes[idioma.Codigo]
		if len(traducciones) != len(mensajes[indice.PorDefecto]) {
			t.Fatal(idioma.Codigo)
		}
		for clave := range mensajes[indice.PorDefecto] {
			if traducciones[clave] == "" || catalogo.T(idioma.Codigo, clave) != traducciones[clave] {
				t.Fatal(clave)
			}
		}
	}
	entrada["idioma"] = "xx"
	b, _ = json.Marshal(entrada)
	var out, diag bytes.Buffer
	if ejecutar(bytes.NewReader(b), &out, &diag, 0) == 0 || out.Len() != 0 {
		t.Fatal(diag.String())
	}
}

type escritorCLIFallido struct{}

func (escritorCLIFallido) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func TestCLIPropagaFalloDeDiagnostico(t *testing.T) {
	var out bytes.Buffer
	if codigo := ejecutar(strings.NewReader("{"), &out, escritorCLIFallido{}, 0); codigo != 2 || out.Len() != 0 {
		t.Fatalf("codigo=%d out=%s", codigo, out.Bytes())
	}
}

func TestCLIPropagaFalloDeSalida(t *testing.T) {
	var diagnostico bytes.Buffer
	codigo := ejecutar(bytes.NewReader(fixture(t, "cambio-centro")), escritorCLIFallido{}, &diagnostico, 0)
	var fallo struct {
		Error struct {
			Codigo string `json:"codigo"`
		} `json:"error"`
	}
	if codigo != 2 || json.Unmarshal(diagnostico.Bytes(), &fallo) != nil || fallo.Error.Codigo != "salida_no_disponible" {
		t.Fatalf("codigo=%d diagnostico=%s", codigo, diagnostico.Bytes())
	}
}

func TestCLIPropagaFalloDeAmbasSalidas(t *testing.T) {
	if codigo := ejecutar(bytes.NewReader(fixture(t, "cambio-centro")), escritorCLIFallido{}, escritorCLIFallido{}, 0); codigo != 2 {
		t.Fatalf("codigo=%d", codigo)
	}
}
