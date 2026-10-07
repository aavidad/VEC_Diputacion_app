package domain

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func datosPropuestaConPoliticaFin(t *testing.T, fechaFin string) DatosCrearPropuestaDecisionCobertura {
	t.Helper()
	datos := datosPropuestaDecisionCoberturaPrueba(t)
	datos.Periodo.PoliticaFin = PoliticaFin{
		ReglaRef: "regla:ct:fin:sintetica", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("e", 64), FechaFin: fechaFin,
	}
	if fechaFin != "obligatoria" {
		datos.Periodo.PoliticaFin.CausaFin = "retorno_titular"
		datos.Periodo.CausaFin = "retorno_titular"
		datos.Periodo.Fin = time.Time{}
	}
	return datos
}

func TestPropuestaCoberturaFinPorFechaOCausaConservaSnapshotV2(t *testing.T) {
	for _, fechaFin := range []string{"obligatoria", "opcional", "no_aplica"} {
		t.Run(fechaFin, func(t *testing.T) {
			datos := datosPropuestaConPoliticaFin(t, fechaFin)
			propuesta, err := CrearPropuestaDecisionCobertura(datos)
			if err != nil {
				t.Fatalf("crear propuesta con política %s: %v", fechaFin, err)
			}
			publicacion := propuesta.Publicacion()
			if publicacion.Canon != CanonHuellaPropuestaDecisionCoberturaV2() || publicacion.Periodo != datos.Periodo {
				t.Fatal("la publicación perdió la política o no declaró V2")
			}
			restaurada, err := RestaurarPropuestaDecisionCobertura(publicacion, datos.Catalogo, datos.Politica)
			if err != nil || restaurada.HuellaSHA256() != propuesta.HuellaSHA256() {
				t.Fatalf("recuperar snapshot y huella originales: %v", err)
			}
			semantica, err := propuesta.IdentidadSemantica()
			if err != nil || semantica.Canon != CanonHuellaSemanticaPropuestaDecisionCoberturaV2() {
				t.Fatalf("reconfirmación sin canon V2: %v", err)
			}
			adulterada := semantica
			adulterada.Canon = CanonHuellaSemanticaPropuestaDecisionCoberturaV1()
			if adulterada.Validar() != ErrDatoInvalido {
				t.Fatal("cambiar el canon conservó una referencia de otra versión")
			}
			contenido, err := json.Marshal(publicacion)
			if err != nil || bytes.Contains(contenido, []byte("0001-")) {
				t.Fatalf("serialización contiene fecha inventada: %s, %v", contenido, err)
			}
			if fechaFin != "obligatoria" && bytes.Contains(contenido, []byte(`"fin":`)) {
				t.Fatal("la causa de fin publicó una fecha")
			}
			if _, err := materialCanonicoPropuestaDecisionCoberturaV1(publicacion); err != ErrDatoInvalido {
				t.Fatal("el canon antiguo reinterpretó el periodo nuevo")
			}
			if _, err := materialCanonicoSemanticoPropuestaDecisionCoberturaV1(CanonHuellaSemanticaPropuestaDecisionCoberturaV1(), publicacion); err != ErrDatoInvalido {
				t.Fatal("el canon semántico antiguo ignoró el snapshot")
			}
		})
	}
}

func TestPropuestaCoberturaFinSellaCadaCampoDePolitica(t *testing.T) {
	datos := datosPropuestaConPoliticaFin(t, "opcional")
	original, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil {
		t.Fatal(err)
	}
	semanticaOriginal, err := original.IdentidadSemantica()
	if err != nil {
		t.Fatal(err)
	}
	casos := map[string]func(*PeriodoPrevisto){
		"regla":     func(p *PeriodoPrevisto) { p.PoliticaFin.ReglaRef += ":otra" },
		"version":   func(p *PeriodoPrevisto) { p.PoliticaFin.CatalogoVersion++ },
		"huella":    func(p *PeriodoPrevisto) { p.PoliticaFin.CatalogoHuellaSHA256 = strings.Repeat("f", 64) },
		"fecha_fin": func(p *PeriodoPrevisto) { p.PoliticaFin.FechaFin = "no_aplica" },
		"causa": func(p *PeriodoPrevisto) {
			p.CausaFin = "fin_necesidad"
			p.PoliticaFin.CausaFin = p.CausaFin
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			otros := datos
			cambiar(&otros.Periodo)
			nueva, err := CrearPropuestaDecisionCobertura(otros)
			if err != nil {
				t.Fatal(err)
			}
			semanticaNueva, err := nueva.IdentidadSemantica()
			if err != nil || nueva.HuellaSHA256() == original.HuellaSHA256() || semanticaNueva.CoincideExactamente(semanticaOriginal) {
				t.Fatalf("la política cambiada dejó pasar la huella anterior: %v", err)
			}
			manipulada := original.Publicacion()
			manipulada.Periodo = otros.Periodo
			if _, err := RestaurarPropuestaDecisionCobertura(manipulada, datos.Catalogo, datos.Politica); err != ErrDatoInvalido {
				t.Fatal("se restauró el contenido manipulado con el sello original")
			}
		})
	}
}

func TestPropuestaCoberturaFinRechazaCausaSinPoliticaYMezclaConFecha(t *testing.T) {
	casos := map[string]func(*PeriodoPrevisto){
		"sin_snapshot":   func(p *PeriodoPrevisto) { p.PoliticaFin = PoliticaFin{} },
		"fecha_y_causa":  func(p *PeriodoPrevisto) { p.Fin = p.Inicio.AddDate(0, 1, 0) },
		"causa_distinta": func(p *PeriodoPrevisto) { p.CausaFin = "fin_necesidad" },
		"fecha_obligatoria": func(p *PeriodoPrevisto) {
			p.PoliticaFin.FechaFin = "obligatoria"
			p.PoliticaFin.CausaFin = ""
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			datos := datosPropuestaConPoliticaFin(t, "opcional")
			cambiar(&datos.Periodo)
			if _, err := CrearPropuestaDecisionCobertura(datos); err != ErrDatoInvalido {
				t.Fatalf("se aceptó periodo incompatible: %v", err)
			}
		})
	}
}

func TestPropuestaCoberturaFechaOpcionalSellaLaCausaDePolitica(t *testing.T) {
	datos := datosPropuestaConPoliticaFin(t, "opcional")
	datos.Periodo.Fin = datos.Periodo.Inicio.AddDate(0, 1, 0)
	datos.Periodo.CausaFin = ""
	original, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil {
		t.Fatalf("fecha conocida admitida por política opcional: %v", err)
	}
	originalSemantica, err := original.IdentidadSemantica()
	if err != nil {
		t.Fatal(err)
	}
	datos.Periodo.PoliticaFin.CausaFin = "fin_necesidad"
	nueva, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil {
		t.Fatal(err)
	}
	nuevaSemantica, err := nueva.IdentidadSemantica()
	if err != nil || nueva.HuellaSHA256() == original.HuellaSHA256() || nuevaSemantica.CoincideExactamente(originalSemantica) {
		t.Fatalf("la causa de política se ignoró cuando había fecha conocida: %v", err)
	}
}
