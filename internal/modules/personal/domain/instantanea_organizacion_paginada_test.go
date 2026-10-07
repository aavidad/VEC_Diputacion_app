package domain

import (
	"fmt"
	"reflect"
	"testing"
)

func paginasComparacionEjemplo() []PaginaInstantaneaOrganizacion {
	i := instantaneaComparacionEjemplo()
	s := i.Selector
	s.VersionRPTRef = ""
	s.VersionPlantillaRef = ""
	s.Limite = 1
	paginas := make([]PaginaInstantaneaOrganizacion, 6)
	for n := range paginas {
		p := PaginaInstantaneaOrganizacion{Instantanea: InstantaneaComparacionOrganizacion{Selector: s, Cobertura: i.Cobertura}, VersionRPTRef: i.Selector.VersionRPTRef, VersionPlantillaRef: i.Selector.VersionPlantillaRef}
		if n > 0 {
			p.Instantanea.Selector.Cursor = fmt.Sprintf("pagina_%d", n)
		}
		if n < 5 {
			p.CursorSiguiente = fmt.Sprintf("pagina_%d", n+1)
		}
		switch n {
		case 0:
			p.Instantanea.Unidades = i.Unidades
		case 1:
			p.Instantanea.PuestosTipo = i.PuestosTipo
		case 2:
			p.Instantanea.Dotaciones = i.Dotaciones
		case 3:
			p.Instantanea.Plazas = i.Plazas
		case 4:
			p.Instantanea.PuestosIndividuales = i.PuestosIndividuales
		case 5:
			p.Instantanea.Vinculos = i.Vinculos
		}
		paginas[n] = p
	}
	return paginas
}
func TestReunionPaginasConservaSeisColeccionesYCobertura(t *testing.T) {
	p := paginasComparacionEjemplo()
	for n := range p {
		p[n].Instantanea.Cobertura.Plazas = "parcial"
	}
	i, err := ReunirPaginasOrganizacionHistorica(p)
	if err != nil {
		t.Fatal(err)
	}
	esperado := instantaneaComparacionEjemplo()
	esperado.Selector.Limite = 1
	esperado.Cobertura.Plazas = "parcial"
	if !reflect.DeepEqual(i, esperado) {
		t.Fatalf("snapshot differs: %+v", i)
	}
	i.Unidades[0].Etiqueta = "mutada"
	if p[0].Instantanea.Unidades[0].Etiqueta == "mutada" {
		t.Fatal("shared mutable result")
	}
	r, _ := NuevaReunionPaginasOrganizacion(p[0].Instantanea.Selector)
	if err := r.Agregar(p[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Finalizar(); err == nil {
		t.Fatal("unfinished chain accepted")
	}
}
func TestReunionPaginasRechazaCadenasIncoherentes(t *testing.T) {
	casos := map[string]func([]PaginaInstantaneaOrganizacion) []PaginaInstantaneaOrganizacion{
		"sin paginas":       func(p []PaginaInstantaneaOrganizacion) []PaginaInstantaneaOrganizacion { return nil },
		"inicio intermedio": func(p []PaginaInstantaneaOrganizacion) []PaginaInstantaneaOrganizacion { return p[1:] },
		"pagina faltante": func(p []PaginaInstantaneaOrganizacion) []PaginaInstantaneaOrganizacion {
			return append(p[:1], p[2:]...)
		},
		"sin final":  func(p []PaginaInstantaneaOrganizacion) []PaginaInstantaneaOrganizacion { return p[:5] },
		"tras final": func(p []PaginaInstantaneaOrganizacion) []PaginaInstantaneaOrganizacion { return append(p, p[5]) },
		"ciclo": func(p []PaginaInstantaneaOrganizacion) []PaginaInstantaneaOrganizacion {
			p[3].CursorSiguiente = "pagina_1"
			return p
		},
		"cursor invalido": func(p []PaginaInstantaneaOrganizacion) []PaginaInstantaneaOrganizacion {
			p[0].CursorSiguiente = "../secret"
			return p
		},
		"selector": func(p []PaginaInstantaneaOrganizacion) []PaginaInstantaneaOrganizacion {
			p[1].Instantanea.Selector.VigenteEn = "2026-03-01"
			return p
		},
		"conocimiento": func(p []PaginaInstantaneaOrganizacion) []PaginaInstantaneaOrganizacion {
			p[1].Instantanea.Selector.ConocidoEn = p[1].Instantanea.Selector.ConocidoEn.Add(1)
			return p
		},
		"limite": func(p []PaginaInstantaneaOrganizacion) []PaginaInstantaneaOrganizacion {
			p[1].Instantanea.Selector.Limite = 2
			return p
		},
		"version rpt": func(p []PaginaInstantaneaOrganizacion) []PaginaInstantaneaOrganizacion {
			p[3].VersionRPTRef = "rpt:otra"
			return p
		},
		"version plantilla": func(p []PaginaInstantaneaOrganizacion) []PaginaInstantaneaOrganizacion {
			p[1].VersionPlantillaRef = "plantilla:otra"
			return p
		},
		"cobertura": func(p []PaginaInstantaneaOrganizacion) []PaginaInstantaneaOrganizacion {
			p[1].Instantanea.Cobertura.Plazas = "parcial"
			return p
		},
		"duplicado entre colecciones": func(p []PaginaInstantaneaOrganizacion) []PaginaInstantaneaOrganizacion {
			p[1].Instantanea.PuestosTipo[0].Traza.ID = p[0].Instantanea.Unidades[0].Traza.ID
			return p
		},
		"pagina vacia pendiente": func(p []PaginaInstantaneaOrganizacion) []PaginaInstantaneaOrganizacion {
			p[1].Instantanea.PuestosTipo = nil
			return p
		},
		"exceso pagina": func(p []PaginaInstantaneaOrganizacion) []PaginaInstantaneaOrganizacion {
			p[0].Instantanea.Unidades = append(p[0].Instantanea.Unidades, p[0].Instantanea.Unidades[0])
			return p
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			i, err := ReunirPaginasOrganizacionHistorica(mutar(paginasComparacionEjemplo()))
			if err == nil || !reflect.DeepEqual(i, InstantaneaComparacionOrganizacion{}) {
				t.Fatalf("accepted: %+v %v", i, err)
			}
		})
	}
}
func TestReunionPaginasAcotaHechosYConservaFallo(t *testing.T) {
	p := paginasComparacionEjemplo()[0]
	p.Instantanea.Selector.Limite = 100
	r, _ := NuevaReunionPaginasOrganizacion(p.Instantanea.Selector)
	unidad := p.Instantanea.Unidades[0]
	for n := 0; n < 100; n++ {
		p.Instantanea.Unidades = nil
		if n > 0 {
			p.Instantanea.Selector.Cursor = fmt.Sprintf("pagina_%d", n)
		}
		p.CursorSiguiente = fmt.Sprintf("pagina_%d", n+1)
		for j := 0; j < 100; j++ {
			u := unidad
			u.Traza.ID = fmt.Sprintf("unidad:%d", n*100+j)
			p.Instantanea.Unidades = append(p.Instantanea.Unidades, u)
		}
		if err := r.Agregar(p); err != nil {
			t.Fatal(err)
		}
	}
	p.Instantanea.Selector.Cursor = "pagina_100"
	p.CursorSiguiente = ""
	p.Instantanea.Unidades = []UnidadOrganizacionHistorica{unidad}
	p.Instantanea.Unidades[0].Traza.ID = "unidad:exceso"
	if r.Agregar(p) == nil {
		t.Fatal("aggregate limit ignored")
	}
	p.Instantanea.Unidades = nil
	if r.Agregar(p) == nil {
		t.Fatal("failed chain repaired silently")
	}
	if _, err := r.Finalizar(); err == nil {
		t.Fatal("partial result after failure")
	}
}
