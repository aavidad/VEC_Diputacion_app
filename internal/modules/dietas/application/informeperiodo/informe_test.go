package informeperiodo

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

const raiz = "../../../../../"

func cargar(t *testing.T) (Configuracion, Datos) {
	t.Helper()
	var c Configuracion
	var d Datos
	for ruta, destino := range map[string]any{
		raiz + "data/catalogos/dietas/informes-ejemplo-v1.json": &c,
		raiz + "data/demo/dietas/informes.json":                 &d,
	} {
		raw, err := os.ReadFile(ruta)
		if err != nil {
			t.Fatal(err)
		}
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(destino); err != nil {
			t.Fatal(err)
		}
	}
	return c, d
}

func referencias(inf Informe) []string {
	var r []string
	for _, f := range inf.Filas {
		r = append(r, f.Referencia)
	}
	return r
}

func TestEjemploCompletoConservaOchoRegistrosY414Euros(t *testing.T) {
	c, d := cargar(t)
	inf, err := Preparar(c, d, Filtros{})
	if err != nil {
		t.Fatal(err)
	}
	if len(inf.Filas) != 8 || inf.TotalCentimos != 41420 || inf.Moneda != "EUR" {
		t.Fatalf("filas=%d total=%d", len(inf.Filas), inf.TotalCentimos)
	}
	if strings.Join(inf.Conceptos, ",") != "manutencion,kilometraje,otros_gastos" ||
		inf.ConceptosCentimos[0] != 25500 || inf.ConceptosCentimos[1] != 11720 || inf.ConceptosCentimos[2] != 4200 {
		t.Fatalf("conceptos=%v %v", inf.Conceptos, inf.ConceptosCentimos)
	}
	if f := inf.Filas[0]; f.Persona != "Ana Molina" || f.Fecha.Format("2006-01-02") != "2026-09-02" || f.ImporteIncluidoCentimos != 6110 {
		t.Fatalf("primera fila %+v", f)
	}
}

// Mismo caso que la vista: fecha de liquidación, solo liquidado y manutención.
func TestVarianteDeCatalogoCoincideConLaVista(t *testing.T) {
	c, d := cargar(t)
	c.Criterio = Criterio{CampoFecha: "fecha_liquidacion", EstadosIncluidos: []string{"liquidado"}, ConceptosIncluidos: []string{"manutencion"}}
	inf, err := Preparar(c, d, Filtros{Desde: "2026-09-05", Hasta: "2026-09-20"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(referencias(inf), ",") != "DI-002,DI-006" || inf.TotalCentimos != 4250 ||
		inf.Filas[0].ImporteIncluidoCentimos != 4250 || inf.Filas[1].ImporteIncluidoCentimos != 0 ||
		inf.Filas[0].Fecha.Format("2006-01-02") != "2026-09-06" {
		t.Fatalf("informe %+v", inf)
	}
}

func TestFiltrosPersonaUnidadYSituacion(t *testing.T) {
	c, d := cargar(t)
	casos := map[string]struct {
		f    Filtros
		refs string
	}{
		"persona":            {Filtros{Persona: "persona-demo-01"}, "DI-001,DI-004"},
		"unidad":             {Filtros{Unidad: "unidad-demo-02"}, "DI-002,DI-006"},
		"situacion":          {Filtros{Situacion: "fiscalizado"}, "DI-004,DI-008"},
		"periodo inclusivo":  {Filtros{Desde: "2026-09-08", Hasta: "2026-09-11"}, "DI-003,DI-004"},
		"situacion excluida": {Filtros{Situacion: "pagado"}, ""},
	}
	for nombre, caso := range casos {
		inf, err := Preparar(c, d, caso.f)
		if err != nil || strings.Join(referencias(inf), ",") != caso.refs {
			t.Fatalf("%s: %v %v", nombre, referencias(inf), err)
		}
	}
}

func TestRechazaFiltrosInvalidos(t *testing.T) {
	c, d := cargar(t)
	for _, f := range []Filtros{{Desde: "2026-09-20", Hasta: "2026-09-05"}, {Desde: "2026-02-30"}, {Hasta: "20/09/2026"}} {
		if _, err := Preparar(c, d, f); !errors.Is(err, ErrFiltrosInvalidos) {
			t.Fatalf("%+v: %v", f, err)
		}
	}
}

func TestRechazaDatosIncoherentes(t *testing.T) {
	casos := map[string]func(*Configuracion, *Datos){
		"referencia distinta": func(c *Configuracion, _ *Datos) { c.Referencia = "otra" },
		"version distinta":    func(_ *Configuracion, d *Datos) { d.ConfiguracionVersion = 2 },
		"no sintetico":        func(_ *Configuracion, d *Datos) { d.Naturaleza = "real" },
		"total alterado":      func(_ *Configuracion, d *Datos) { d.Registros[0].TotalCentimos++ },
		"referencia repetida": func(_ *Configuracion, d *Datos) { d.Registros[1].Referencia = d.Registros[0].Referencia },
		"fecha ausente":       func(_ *Configuracion, d *Datos) { d.Registros[0].FechaLiquidacion = TextoOpcional{} },
		"concepto ausente":    func(_ *Configuracion, d *Datos) { delete(d.Registros[0].ConceptosCentimos, "kilometraje") },
		"importe negativo": func(_ *Configuracion, d *Datos) {
			d.Registros[0].ConceptosCentimos["otros_gastos"] = -1
			d.Registros[0].TotalCentimos--
		},
		"fecha del periodo nula": func(c *Configuracion, _ *Datos) { c.Criterio.CampoFecha = "fecha_liquidacion" },
		"actor no de ejemplo":    func(c *Configuracion, _ *Datos) { c.Historia[0].ActorRef = "actor:real:1" },
		"moneda nula":            func(_ *Configuracion, d *Datos) { d.Registros[0].Moneda = TextoOpcional{Presente: true} },
		"moneda sin centimos":    func(_ *Configuracion, d *Datos) { d.Moneda = "JPY" },
		"campo de fecha libre":   func(c *Configuracion, _ *Datos) { c.Criterio.CampoFecha = "fecha_pago" },
	}
	for nombre, alterar := range casos {
		c, d := cargar(t)
		alterar(&c, &d)
		if _, err := Preparar(c, d, Filtros{}); !errors.Is(err, ErrDatosInvalidos) {
			t.Fatalf("%s: %v", nombre, err)
		}
	}
}
