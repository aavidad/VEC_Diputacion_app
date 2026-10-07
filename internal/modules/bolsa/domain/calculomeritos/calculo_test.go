package calculomeritos

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/shared/baremacion"
)

func fixture(t *testing.T, nombre string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "application", "simulacionbaremo", "testdata", nombre+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func ejemplo(t *testing.T, nombre string) (Conjunto, Entrada) {
	t.Helper()
	cb, eb := fixture(t, nombre), fixture(t, "meritos_entrada")
	c, err := RestaurarConjunto(cb, HuellaSHA256(cb))
	if err != nil {
		t.Fatal(err)
	}
	e, err := RestaurarEntrada(eb, HuellaSHA256(eb))
	if err != nil {
		t.Fatal(err)
	}
	return c, e
}
func racional(t *testing.T, n, d int64) *baremacion.Racional {
	t.Helper()
	r, err := baremacion.NuevoRacional(n, d)
	if err != nil {
		t.Fatal(err)
	}
	return &r
}
func puntos(t *testing.T, n int64) *baremacion.Puntos {
	t.Helper()
	p, err := baremacion.PuntosDesdeMicropuntos(n)
	if err != nil {
		t.Fatal(err)
	}
	return &p
}
func fecha(t *testing.T, a, m, d int) *baremacion.FechaCivil {
	t.Helper()
	f, err := baremacion.NuevaFechaCivil(a, m, d)
	if err != nil {
		t.Fatal(err)
	}
	return &f
}

func TestDosConfiguracionesTopesYCorte(t *testing.T) {
	for _, caso := range []struct {
		reglas                     string
		total, suma, curso, titulo int64
	}{{"meritos_reglas_a", 4000000, 4100000, 1500000, 2000000}, {"meritos_reglas_b", 6000000, 6600000, 2000000, 5000000}} {
		t.Run(caso.reglas, func(t *testing.T) {
			c, e := ejemplo(t, caso.reglas)
			r, err := Calcular(c, e)
			if err != nil {
				t.Fatal(err)
			}
			if r.Estado != "completado" || r.Total == nil || r.Total.Micropuntos() != caso.total || r.SumaSecciones.Micropuntos() != caso.suma || r.Reglas[0].Puntos.Micropuntos() != caso.curso || r.Reglas[2].Puntos.Micropuntos() != caso.titulo {
				t.Fatalf("desglose inesperado: %+v", r)
			}
			for idx, codigo := range map[int]string{2: "inferior_al_minimo", 5: "requisito_no_puntua", 7: "posterior_al_corte", 8: "merito_caducado"} {
				if r.Meritos[idx].Codigo != codigo {
					t.Fatalf("merito %d: %s", idx, r.Meritos[idx].Codigo)
				}
			}
			if r.Reglas[1].UnidadesComputadas.String() != "3/2" || r.Reglas[1].Puntos.Micropuntos() != 900000 {
				t.Fatal("fracción perdida")
			}
			b, err := r.RepresentacionCanonica()
			if err != nil {
				t.Fatal(err)
			}
			r2, err := Calcular(c, e)
			if err != nil {
				t.Fatal(err)
			}
			b2, _ := r2.RepresentacionCanonica()
			if !bytes.Equal(b, b2) {
				t.Fatal("no determinista")
			}
			h, _ := r.HuellaSHA256()
			if h != HuellaSHA256(b) {
				t.Fatal("huella incoherente")
			}
		})
	}
}
func TestImpedimentosNuncaPublicanTotalNiPuntosParciales(t *testing.T) {
	casos := []struct {
		nombre, codigo string
		cambiar        func(*Entrada)
	}{
		{"sin_regla", "regla_ausente", func(e *Entrada) { e.Meritos[0].Clase = "sin_regla" }},
		{"catalogo", "catalogo_incompatible", func(e *Entrada) { e.Meritos[0].Catalogo.Version++ }},
		{"declarado", "acreditacion_pendiente", func(e *Entrada) { e.Meritos[0].Estado = "declarado" }},
		{"pendiente", "acreditacion_pendiente", func(e *Entrada) { e.Meritos[0].Estado = "pendiente" }},
		{"aplicable", "aplicabilidad_pendiente", func(e *Entrada) { e.Meritos[0].Aplicabilidad = "pendiente" }},
		{"fecha", "dato_pendiente", func(e *Entrada) { e.Meritos[0].FechaObtencion = nil }},
		{"cantidad", "dato_pendiente", func(e *Entrada) { e.Meritos[0].Unidades = nil }},
		{"hecho_duplicado", "merito_duplicado", func(e *Entrada) { e.Meritos[1].Hecho = e.Meritos[0].Hecho; e.Meritos[1].Hecho.Version++ }},
		{"evidencia_duplicada", "merito_duplicado", func(e *Entrada) { e.Meritos[1].Evidencia = e.Meritos[0].Evidencia }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			c, e := ejemplo(t, "meritos_reglas_a")
			caso.cambiar(&e)
			r, err := Calcular(c, e)
			if err != nil {
				t.Fatal(err)
			}
			if r.Estado != "bloqueado" || r.Total != nil || r.SumaSecciones != nil || len(r.Reglas) != 0 || len(r.Secciones) != 0 {
				t.Fatal("resultado parcial publicable")
			}
			found := false
			for _, i := range r.Incidencias {
				found = found || i.Codigo == caso.codigo
			}
			if !found {
				t.Fatalf("falta %s: %+v", caso.codigo, r.Incidencias)
			}
			b, _ := r.RepresentacionCanonica()
			if bytes.Contains(b, []byte(`"total"`)) {
				t.Fatal("bloqueado serializa total")
			}
		})
	}
}
func TestUmbralInclusivoYVigenciaAlCorte(t *testing.T) {
	c, e := ejemplo(t, "meritos_reglas_a")
	e.Meritos[0].Unidades = racional(t, 20, 1)
	e.Meritos[0].VigenteHasta = &c.FechaCorteInclusiva
	r, err := Calcular(c, e)
	if err != nil {
		t.Fatal(err)
	}
	if r.Meritos[0].Codigo != "admitido" || r.Reglas[0].UnidadesAdmitidas.String() != "50/1" {
		t.Fatal("extremo inclusivo descartado")
	}
}
func TestSeleccionMayorUnidadYTopeUnidades(t *testing.T) {
	c, e := ejemplo(t, "meritos_reglas_a")
	*c.Reglas[0].MaximoElementos = 1
	c.Reglas[0].MaximoUnidades = racional(t, 40, 1)
	e.Meritos[0].Unidades = racional(t, 30, 1)
	e.Meritos[1].Unidades = racional(t, 60, 1)
	r, err := Calcular(c, e)
	if err != nil {
		t.Fatal(err)
	}
	if r.Meritos[0].Codigo != "maximo_elementos" || r.Meritos[1].Codigo != "admitido" || r.Reglas[0].UnidadesAdmitidas.String() != "60/1" || r.Reglas[0].UnidadesComputadas.String() != "40/1" || r.Reglas[0].Puntos.Micropuntos() != 800000 {
		t.Fatal("selección/topado incorrectos")
	}
}
func TestRedondeaUnaVezPorReglaYRechazaPerdidaExacta(t *testing.T) {
	c, e := ejemplo(t, "meritos_reglas_a")
	c.Reglas[0].MinimoUnidades = racional(t, 0, 1)
	c.Reglas[0].PuntosPorUnidad = puntos(t, 1)
	e.Meritos = e.Meritos[:2]
	e.Meritos[0].Unidades = racional(t, 1, 2)
	e.Meritos[1].Unidades = racional(t, 1, 2)
	r, err := Calcular(c, e)
	if err != nil {
		t.Fatal(err)
	}
	if r.Total.Micropuntos() != 1 {
		t.Fatal("redondeó cada curso")
	}
	e.Meritos = e.Meritos[:1]
	e.Meritos[0].Unidades = racional(t, 1, 3)
	c.Reglas[0].Redondeo = baremacion.RedondeoExacto
	if r, err := Calcular(c, e); err == nil || r.Total != nil {
		t.Fatal("exacto inventó redondeo")
	}
}
func TestAusenciaYAmbiguedadDePoliticasSeRechazan(t *testing.T) {
	casos := []struct {
		nombre  string
		cambiar func(*Conjunto)
	}{
		{"tope", func(c *Conjunto) { c.MaximoTotal = nil }}, {"unidad", func(c *Conjunto) { c.Reglas[0].Unidad = "titulo" }}, {"redondeo", func(c *Conjunto) { c.Reglas[0].Redondeo = "" }}, {"duplicados", func(c *Conjunto) { c.Duplicados = "ignorar" }}, {"momento", func(c *Conjunto) { c.MomentoRedondeo = "por_merito" }}, {"seleccion", func(c *Conjunto) { c.SeleccionElementos = "" }}, {"catalogo", func(c *Conjunto) { c.Reglas[0].Catalogo.HuellaSHA256 = "" }}, {"ambigua", func(c *Conjunto) {
			c.Reglas[1].Familia = c.Reglas[0].Familia
			c.Reglas[1].Unidad = c.Reglas[0].Unidad
			c.Reglas[1].Clase = c.Reglas[0].Clase
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			c, e := ejemplo(t, "meritos_reglas_a")
			caso.cambiar(&c)
			if r, err := Calcular(c, e); err == nil || r.Total != nil {
				t.Fatal("aceptó regla incompleta")
			}
		})
	}
}
func TestContratosNoCanonicosHuellaYPrivacidad(t *testing.T) {
	b := fixture(t, "meritos_reglas_a")
	if _, err := RestaurarConjunto(b, strings.Repeat("0", 64)); err == nil {
		t.Fatal("huella incorrecta admitida")
	}
	for _, datos := range [][]byte{append(append([]byte(nil), b...), '\n'), bytes.Replace(b, []byte(`"version":1`), []byte(`"version":1,"version":1`), 1), bytes.Replace(b, []byte(`"esquema":`), []byte(`"intruso":true,"esquema":`), 1)} {
		if _, err := RestaurarConjunto(datos, HuellaSHA256(datos)); err == nil {
			t.Fatal("aceptó JSON no canónico")
		}
	}
	c, e := ejemplo(t, "meritos_reglas_a")
	e.Meritos[0].Evidencia.Referencia = "documento:nombre_personal"
	if _, err := Calcular(c, e); err == nil {
		t.Fatal("referencia no opaca admitida")
	}
	c, e = ejemplo(t, "meritos_reglas_a")
	e.Meritos[3].Unidades = racional(t, 2, 1)
	if _, err := Calcular(c, e); err == nil {
		t.Fatal("dos títulos en un hecho")
	}
	e.Meritos[3].Unidades = racional(t, 1, 1)
	e.Meritos[0].VigenteHasta = fecha(t, 2026, 1, 1)
	if _, err := Calcular(c, e); err == nil {
		t.Fatal("vigencia invertida")
	}
	e.Meritos[0].VigenteHasta = nil
	e.Meritos = []Merito{}
	r, err := Calcular(c, e)
	if err != nil || r.Total == nil || r.Total.Micropuntos() != 0 {
		t.Fatal("entrada vacía explícita no produce cero válido")
	}
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RestaurarEntrada(raw, HuellaSHA256(raw)); err != nil {
		t.Fatal(err)
	}
}

func TestDependenciasVersionadasNoAdmitenContenidosContradictorios(t *testing.T) {
	c, e := ejemplo(t, "meritos_reglas_a")
	c.Reglas[1].Catalogo = c.Reglas[0].Catalogo
	c.Reglas[1].Catalogo.HuellaSHA256 = strings.Repeat("f", 64)
	if r, err := Calcular(c, e); err == nil || r.Total != nil {
		t.Fatal("reglas aceptan dos huellas de una versión")
	}
	c, e = ejemplo(t, "meritos_reglas_a")
	e.Meritos[1].Catalogo.HuellaSHA256 = strings.Repeat("f", 64)
	if r, err := Calcular(c, e); err == nil || r.Total != nil {
		t.Fatal("entrada acepta dos huellas de una versión")
	}
}

func TestDependenciasContradictoriasEntreReglasYEntradaSinPuntos(t *testing.T) {
	for _, uso := range []string{"merito", "requisito"} {
		c, e := ejemplo(t, "meritos_reglas_a")
		e.Meritos = []Merito{e.Meritos[0]}
		e.Meritos[0].Uso = uso
		e.Meritos[0].Aplicabilidad = "no"
		e.Meritos[0].Catalogo.HuellaSHA256 = strings.Repeat("f", 64)
		if r, err := Calcular(c, e); err == nil || r.Total != nil {
			t.Fatal("entrada contradictoria ignorada por clasificación")
		}
	}
}

func TestDependenciasNoSuplantanLasInstantaneasDelCalculo(t *testing.T) {
	for _, instantanea := range []string{"conjunto", "entrada"} {
		c, e := ejemplo(t, "meritos_reglas_a")
		e.Meritos = []Merito{e.Meritos[0]}
		if instantanea == "conjunto" {
			e.Meritos[0].Hecho = Dependencia{c.Referencia, c.Version, strings.Repeat("f", 64)}
		} else {
			e.Meritos[0].Hecho = Dependencia{e.Referencia, e.Version, strings.Repeat("f", 64)}
		}
		if r, err := Calcular(c, e); err == nil || r.Total != nil {
			t.Fatal("dependencia suplanta la instantánea")
		}
	}
}
