package calculomeritos

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func parejaComparacion(t *testing.T) (Conjunto, Conjunto) {
	t.Helper()
	var conjuntos [2]Conjunto
	for i, nombre := range []string{"comparacion_meritos_v1", "comparacion_meritos_v2"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "cmd", "vec-baremador", "testdata", nombre+".json"))
		if err != nil {
			t.Fatal(err)
		}
		conjuntos[i], err = RestaurarConjunto(b, HuellaSHA256(b))
		if err != nil {
			t.Fatal(err)
		}
	}
	return conjuntos[0], conjuntos[1]
}

func TestDiferenciaSemanticaExactaYDeterminista(t *testing.T) {
	a, n := parejaComparacion(t)
	ab, _ := a.RepresentacionCanonica()
	nb, _ := n.RepresentacionCanonica()
	esperados := map[string][2]string{
		"conjunto//fecha_corte_inclusiva": {`"2026-09-30"`, `"2026-12-31"`},
		"conjunto//maximo_total":          {`"4000000"`, `"5000000"`},
		"regla/curso/maximo_puntos":       {`"1500000"`, `"2000000"`},
		"regla/curso/puntos_por_unidad":   {`"20000"`, `"30000"`},
		"seccion/formacion/maximo_puntos": {`"1200000"`, `"1800000"`},
	}
	var original []byte
	for intento := 0; intento < 20; intento++ {
		d, err := CompararConjuntos(a, n)
		if err != nil {
			t.Fatal(err)
		}
		if d.Anterior.Version != 1 || d.Nuevo.Version != 2 || d.Anterior.HuellaSHA256 != HuellaSHA256(ab) || d.Nuevo.HuellaSHA256 != HuellaSHA256(nb) || len(d.Cambios) != 7 {
			t.Fatalf("cabecera o cambios incorrectos: %+v", d)
		}
		for _, cambio := range d.Cambios {
			key := cambio.Ambito + "/" + cambio.Clave + "/" + cambio.Campo
			if par, ok := esperados[key]; ok {
				if string(cambio.Anterior) != par[0] || string(cambio.Nuevo) != par[1] || cambio.Tipo != "modificacion" {
					t.Fatalf("diferencia incorrecta: %+v", cambio)
				}
			} else if cambio.Campo != "definicion" {
				t.Fatalf("cambio inesperado: %+v", cambio)
			}
		}
		b, err := json.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		if intento != 0 && !bytes.Equal(b, original) {
			t.Fatal("orden no determinista")
		}
		original = b
	}
	aDespues, _ := a.RepresentacionCanonica()
	nDespues, _ := n.RepresentacionCanonica()
	if !bytes.Equal(ab, aDespues) || !bytes.Equal(nb, nDespues) {
		t.Fatal("comparo mutando instantaneas")
	}
}

func TestMismaVersionEsIdenticaOContradictoria(t *testing.T) {
	a, n := parejaComparacion(t)
	d, err := CompararConjuntos(a, a)
	if err != nil || d.Cambios == nil || len(d.Cambios) != 0 {
		t.Fatalf("identico: %+v %v", d, err)
	}
	n.Version = a.Version
	if d, err := CompararConjuntos(a, n); err == nil || d.Esquema != "" {
		t.Fatal("misma version distinta aceptada")
	}
	a, n = parejaComparacion(t)
	n.Version = 8
	if _, err := CompararConjuntos(a, n); err != nil {
		t.Fatalf("versiones no consecutivas: %v", err)
	}
}

func TestDiferenciaRechazaIdentidadOrdenYDependenciasContradictorias(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		mutar  func(*Conjunto)
	}{
		{"conjunto", func(c *Conjunto) { c.Referencia = "rgl_" + strings.Repeat("a", 32) }},
		{"convocatoria", func(c *Conjunto) { c.ConvocatoriaRef = "con_" + strings.Repeat("a", 32) }},
		{"expediente", func(c *Conjunto) { c.ExpedienteRef = "exp_" + strings.Repeat("a", 32) }},
		{"bases", func(c *Conjunto) { c.Bases.HuellaSHA256 = strings.Repeat("f", 64) }},
		{"seccion", func(c *Conjunto) { c.Secciones[1].Definicion.HuellaSHA256 = strings.Repeat("f", 64) }},
		{"regla", func(c *Conjunto) { c.Reglas[1].Definicion.HuellaSHA256 = strings.Repeat("f", 64) }},
		{"catalogo", func(c *Conjunto) {
			for i := range c.Reglas {
				c.Reglas[i].Catalogo.HuellaSHA256 = strings.Repeat("f", 64)
			}
		}},
		{"invalido", func(c *Conjunto) { c.MaximoTotal = nil }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			a, n := parejaComparacion(t)
			caso.mutar(&n)
			if d, err := CompararConjuntos(a, n); err == nil || d.Esquema != "" || len(d.Cambios) != 0 {
				t.Fatalf("comparacion invalida: %+v %v", d, err)
			}
		})
	}
	a, n := parejaComparacion(t)
	if _, err := CompararConjuntos(n, a); err == nil {
		t.Fatal("orden inverso aceptado")
	}
}

func TestClavesRenombradasSonBajaYAlta(t *testing.T) {
	a, n := parejaComparacion(t)
	n.Reglas[0].Clave = "curso_nuevo"
	n.Secciones[0].Clave = "formacion_nueva"
	n.Reglas[0].SeccionClave = "formacion_nueva"
	d, err := CompararConjuntos(a, n)
	if err != nil {
		t.Fatal(err)
	}
	altas, bajas := 0, 0
	for _, c := range d.Cambios {
		if c.Tipo == "alta" {
			altas++
			if string(c.Anterior) != "null" {
				t.Fatal("alta con valor anterior")
			}
		}
		if c.Tipo == "baja" {
			bajas++
			if string(c.Nuevo) != "null" {
				t.Fatal("baja con valor nuevo")
			}
		}
	}
	if altas != 2 || bajas != 2 {
		t.Fatalf("renombre inferido: %+v", d.Cambios)
	}
}

func TestDiferenciaNoPermiteSuplantarLasInstantaneas(t *testing.T) {
	for _, caso := range []string{"catalogo_suplanta_anterior", "catalogo_suplanta_nuevo", "anterior_suplanta_nuevo"} {
		t.Run(caso, func(t *testing.T) {
			a, n := parejaComparacion(t)
			dependencia := Dependencia{n.Referencia, n.Version, strings.Repeat("f", 64)}
			switch caso {
			case "catalogo_suplanta_anterior":
				dependencia.Version = a.Version
				n.Reglas[0].Catalogo = dependencia
			case "catalogo_suplanta_nuevo":
				n.Reglas[0].Catalogo = dependencia
			case "anterior_suplanta_nuevo":
				a.Reglas[0].Catalogo = dependencia
			}
			if err := a.Validar(); err != nil {
				t.Fatal(err)
			}
			if err := n.Validar(); err != nil {
				t.Fatal(err)
			}
			d, err := CompararConjuntos(a, n)
			fallo, ok := err.(*Error)
			if !ok || fallo.Codigo != "dependencia_contradictoria" || d.Esquema != "" || len(d.Cambios) != 0 {
				t.Fatalf("instantanea suplantada: esquema=%q cambios=%d error=%v", d.Esquema, len(d.Cambios), err)
			}
		})
	}
}
