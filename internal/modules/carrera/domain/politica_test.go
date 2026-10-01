package domain

import (
	"errors"
	"slices"
	"testing"
)

func politicaGradoSintetica() CatalogoPoliticaGrado {
	f := Fuente{Referencia: "fuente-sintetica", Version: "fuente-2"}
	ev := Evidencia{Referencia: "politica-sintetica", Fuente: f.Referencia, Version: f.Version}
	return CatalogoPoliticaGrado{
		Alcance: AlcanceSintetico, Politica: Politica{Referencia: ev.Referencia, Version: "catalogo-1", Fuente: f.Referencia, Regimenes: []string{"funcionario_carrera"}, AprobacionReferencia: "referencia-no-autenticada"},
		Fuentes: []Fuente{f}, Procedencia: ev, Vigencia: Periodo{Inicio: "2026-01-01", Evidencia: ev},
		Vias: []Evidencia{ev}, Periodos: []Evidencia{ev}, Limites: []Evidencia{ev}, Evidencias: []Evidencia{ev},
	}
}

func TestPoliticaConservaVersionesYSiguePendienteConReferenciaAprobacion(t *testing.T) {
	c := politicaGradoSintetica()
	r, err := RevisarPoliticaGrado(c)
	if err != nil {
		t.Fatal(err)
	}
	if r.Datos.Politica.Version != "catalogo-1" || r.Datos.Procedencia.Version != "fuente-2" || len(r.Pendientes) != 3 {
		t.Fatal("confunde versiones o aprobación")
	}
	for _, key := range []string{"fuente_admitida", "aprobacion_competente", "circuito_nominal"} {
		if !slices.Contains(r.Pendientes, "carrera.politica.pendiente."+key) {
			t.Fatalf("falta %s", key)
		}
	}
	c.Politica.Regimenes[0] = "otra"
	c.Fuentes[0].Version = "otra"
	c.Periodos[0].Referencia = "otra"
	if r.Datos.Politica.Regimenes[0] != "funcionario_carrera" || r.Datos.Fuentes[0].Version != "fuente-2" || r.Datos.Periodos[0].Referencia != "politica-sintetica" {
		t.Fatal("la revisión comparte colecciones")
	}
}

func TestPoliticaIncompletaYFechasFuentesDuplicadosQuedanPendientes(t *testing.T) {
	for _, campo := range []string{"incompleta", "fecha_imposible", "fecha_invertida", "fuente_version", "fuente_repetida", "elemento_repetido"} {
		t.Run(campo, func(t *testing.T) {
			c := politicaGradoSintetica()
			key := "vigencia"
			switch campo {
			case "incompleta":
				c = CatalogoPoliticaGrado{Alcance: AlcanceSintetico}
				key = "identificacion"
			case "fecha_imposible":
				c.Vigencia.Inicio = "2026-02-30"
			case "fecha_invertida":
				c.Vigencia.Fin = "2025-12-31"
			case "fuente_version":
				c.Procedencia.Version = "otra"
				key = "procedencia"
			case "fuente_repetida":
				c.Fuentes = append(c.Fuentes, c.Fuentes[0])
				key = "fuentes"
			case "elemento_repetido":
				c.Vias = append(c.Vias, c.Vias[0])
				key = "vias"
			}
			r, err := RevisarPoliticaGrado(c)
			if err != nil || !slices.Contains(r.Pendientes, "carrera.politica.pendiente."+key) {
				t.Fatalf("incompleto no queda pendiente: %v", err)
			}
		})
	}
}

func TestPoliticaRechazaAlcanceProductivoYCardinalidad(t *testing.T) {
	c := politicaGradoSintetica()
	c.Alcance = "produccion"
	if _, err := RevisarPoliticaGrado(c); !errors.Is(err, ErrPoliticaGrado) {
		t.Fatal("permite política productiva")
	}
	c = politicaGradoSintetica()
	c.Limites = make([]Evidencia, 33)
	if _, err := RevisarPoliticaGrado(c); !errors.Is(err, ErrPoliticaGrado) {
		t.Fatal("permite colección excesiva")
	}
}
