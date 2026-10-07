package reglasbaremo

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/shared/baremacion"
)

func parejaDiferenciaExperiencia(t *testing.T) (ConjuntoReglasBaremo, ConjuntoReglasBaremo) {
	t.Helper()
	var c [2]ConjuntoReglasBaremo
	for i, nombre := range []string{"comparacion_experiencia_v1", "comparacion_experiencia_v2"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "cmd", "vec-baremador", "testdata", nombre+".json"))
		if err != nil {
			t.Fatal(err)
		}
		c[i], err = RestaurarConjuntoReglasBaremo(b)
		if err != nil {
			t.Fatal(err)
		}
	}
	return c[0], c[1]
}

func versionComparacionExperiencia(t *testing.T, a ConjuntoReglasBaremo) ConjuntoReglasBaremo {
	t.Helper()
	b, err := a.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	n, err := RestaurarConjuntoReglasBaremo(b)
	if err != nil {
		t.Fatal(err)
	}
	n.identidad.version++
	return n
}

func TestDiferenciaExperienciaDeterministaConTodosLosValoresExactos(t *testing.T) {
	a, n := parejaDiferenciaExperiencia(t)
	ab, _ := a.RepresentacionCanonica()
	nb, _ := n.RepresentacionCanonica()
	var anterior []byte
	for i := 0; i < 10; i++ {
		d, err := CompararConjuntos(a, n)
		if err != nil {
			t.Fatal(err)
		}
		if len(d.Cambios) != 10 || d.Anterior.Version != 1 || d.Nuevo.Version != 2 || d.Anterior.HuellaSHA256 != "8fdcca0bce51606a680364997d1d232de22d7ba5957cfc5bcf8a4538e5d9bc1d" || d.Nuevo.HuellaSHA256 != "5ef04e7a8beac1563e85905da2ccebc9fe42e8a314bbf45fbbd246288ea99f09" {
			t.Fatalf("cabecera o cantidad: %+v", d)
		}
		for _, c := range d.Cambios {
			switch c.Campo {
			case "puntos_por_unidad":
				if c.Anterior.Puntos.Micropuntos() != 100000 || c.Nuevo.Puntos.Micropuntos() != 200000 {
					t.Fatal("coeficiente inexacto")
				}
			case "unidad_temporal":
				if c.Nuevo.UnidadTemporal.UnidadesBasePorUnidad.String() != "365/12" {
					t.Fatal("conversion inexacta")
				}
			case "jornada":
				if c.Nuevo.Jornada.Umbral.String() != "1/2" {
					t.Fatal("umbral inexacto")
				}
			}
		}
		b, err := json.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		if i > 0 && !bytes.Equal(b, anterior) {
			t.Fatal("orden de diferencias variable")
		}
		anterior = b
	}
	ax, _ := a.RepresentacionCanonica()
	nx, _ := n.RepresentacionCanonica()
	if !bytes.Equal(ab, ax) || !bytes.Equal(nb, nx) {
		t.Fatal("comparacion modifica instantaneas")
	}
}

func TestDiferenciaExperienciaIdentidadVersionEIntegridad(t *testing.T) {
	a, n := parejaDiferenciaExperiencia(t)
	d, err := CompararConjuntos(a, a)
	if err != nil || d.Cambios == nil || len(d.Cambios) != 0 {
		t.Fatal("identico no es vacio")
	}
	if _, err := CompararConjuntos(n, a); err == nil {
		t.Fatal("orden inverso admitido")
	}
	n.identidad.version = a.identidad.version
	if _, err := CompararConjuntos(a, n); !errors.Is(err, ErrHuellaNoCoincide) {
		t.Fatalf("misma version distinta: %v", err)
	}
	for _, campo := range []string{"referencia", "convocatoria", "expediente"} {
		a, n := parejaDiferenciaExperiencia(t)
		switch campo {
		case "referencia":
			n.identidad.referencia = "rgl_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		case "convocatoria":
			n.identidad.convocatoriaRef = "con_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		case "expediente":
			n.identidad.expedienteRef = "exp_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}
		if d, err := CompararConjuntos(a, n); err == nil || d.Esquema != "" {
			t.Fatal("identidad ajena admitida")
		}
	}
}

func TestDiferenciaExperienciaIncluyeTodasLasDependenciasYCabeceras(t *testing.T) {
	for _, campo := range []string{"bases", "seccion", "grupo", "regla", "catalogo", "anterior", "nuevo", "entre_instantaneas", "segundo_catalogo"} {
		t.Run(campo, func(t *testing.T) {
			a, n := parejaDiferenciaExperiencia(t)
			f := strings.Repeat("f", 64)
			switch campo {
			case "bases":
				n.bases.huellaSHA256 = f
			case "seccion":
				n.secciones[0].definicion.version = 1
				n.secciones[0].definicion.huellaSHA256 = f
			case "grupo":
				n.gruposConcurrencia[0].definicion.version = 1
				n.gruposConcurrencia[0].definicion.huellaSHA256 = f
			case "regla":
				n.reglasExperiencia[0].definicion.version = 1
				n.reglasExperiencia[0].definicion.huellaSHA256 = f
			case "catalogo":
				n.reglasExperiencia[0].criterios[0].catalogo.huellaSHA256 = f
			case "anterior":
				n.reglasExperiencia[0].criterios[0].catalogo = ReferenciaVersionada{a.identidad.referencia, a.identidad.version, f}
			case "nuevo":
				n.reglasExperiencia[0].criterios[0].catalogo = ReferenciaVersionada{n.identidad.referencia, n.identidad.version, f}
			case "entre_instantaneas":
				a.reglasExperiencia[0].criterios[0].catalogo = ReferenciaVersionada{n.identidad.referencia, n.identidad.version, f}
			case "segundo_catalogo":
				c := n.reglasExperiencia[0].criterios[0].clonar()
				c.clave = "otro"
				c.catalogo.huellaSHA256 = f
				n.reglasExperiencia[0].criterios = append(n.reglasExperiencia[0].criterios, c)
			}
			if err := a.Validar(); err != nil {
				t.Fatal(err)
			}
			if err := n.Validar(); err != nil {
				t.Fatal(err)
			}
			if d, err := CompararConjuntos(a, n); !errors.Is(err, ErrHuellaNoCoincide) || d.Esquema != "" {
				t.Fatalf("contradiccion no rechazada: %v", err)
			}
		})
	}
}

func TestDiferenciaExperienciaConservaCriteriosCompletosYCopias(t *testing.T) {
	a, _ := parejaDiferenciaExperiencia(t)
	m := materialDeRegla(a.reglasExperiencia[0])
	c := m.Criterios[0]
	c.Clave = "entidad"
	c.Catalogo.Referencia = "catalogo:sintetico:entidad"
	c.Valores = []string{"dato_a", "dato_b"}
	m.Criterios = append(m.Criterios, c)
	r, err := reconstruirReglaExperiencia(m)
	if err != nil {
		t.Fatal(err)
	}
	a.reglasExperiencia[0] = r
	simple, _ := parejaDiferenciaExperiencia(t)
	conAlta := versionComparacionExperiencia(t, a)
	alta, err := CompararConjuntos(simple, conAlta)
	if err != nil || len(alta.Cambios) != 1 || len(*alta.Cambios[0].Anterior.Criterios) != 1 || len(*alta.Cambios[0].Nuevo.Criterios) != 2 {
		t.Fatal("alta de criterio incompleta")
	}
	n := versionComparacionExperiencia(t, a)
	m = materialDeRegla(n.reglasExperiencia[0])
	m.Criterios[0], m.Criterios[1] = m.Criterios[1], m.Criterios[0]
	m.Criterios[0].Valores = []string{"dato_b", "dato_a"}
	r, err = reconstruirReglaExperiencia(m)
	if err != nil {
		t.Fatal(err)
	}
	n.reglasExperiencia[0] = r
	d, err := CompararConjuntos(a, n)
	if err != nil || len(d.Cambios) != 0 {
		t.Fatal("reordenacion equivalente cambia significado")
	}
	// Baja un criterio, cambia valores y version de catalogo del restante.
	m = materialDeRegla(n.reglasExperiencia[0])
	m.Criterios = m.Criterios[1:]
	m.Criterios[0].Valores = []string{"dato_c"}
	m.Criterios[0].Catalogo.Version = 2
	r, err = reconstruirReglaExperiencia(m)
	if err != nil {
		t.Fatal(err)
	}
	n.reglasExperiencia[0] = r
	d, err = CompararConjuntos(a, n)
	if err != nil || len(d.Cambios) != 1 {
		t.Fatalf("criterios: %v", err)
	}
	viejos, nuevos := *d.Cambios[0].Anterior.Criterios, *d.Cambios[0].Nuevo.Criterios
	if len(viejos) != 2 || len(nuevos) != 1 || nuevos[0].Catalogo.Version != 2 || nuevos[0].Valores[0] != "dato_c" {
		t.Fatal("detalle incompleto")
	}
	antes, _ := a.RepresentacionCanonica()
	nuevos[0].Valores[0] = "mutado"
	viejos[0].Valores[0] = "mutado"
	despues, _ := a.RepresentacionCanonica()
	if !bytes.Equal(antes, despues) || n.reglasExperiencia[0].criterios[0].valores[0] != "dato_c" {
		t.Fatal("salida conserva alias de entrada")
	}
}

func TestDiferenciaExperienciaOrdenPrioridadLimitesYAusencia(t *testing.T) {
	a, n := parejaDiferenciaExperiencia(t)
	n.secciones[0].orden = 2
	n.gruposConcurrencia[0].orden = 2
	n.reglasExperiencia[0].orden = 2
	n.reglasExperiencia[0].prioridadConcurrencia = 2
	n.reglasExperiencia[0].maximoUnidades = SinLimiteUnidades()
	n.reglasExperiencia[0].maximoPuntos = SinLimitePuntos()
	a.secciones[0].puntosMinimos, _ = baremacion.PuntosDesdeMicropuntos(1)
	n.gruposConcurrencia[0].solape, _ = NuevaPoliticaSolapeAcumulable(baremacion.JornadaCompleta())
	n.gruposConcurrencia[0].tieneRepartoExceso = true
	n.gruposConcurrencia[0].repartoExceso, _ = NuevaPoliticaRepartoExceso(RepartoExcesoRecortarPorPrioridad)
	d, err := CompararConjuntos(a, n)
	if err != nil {
		t.Fatal(err)
	}
	ordenes, prioridades, repartos := 0, 0, 0
	for _, c := range d.Cambios {
		switch c.Campo {
		case "orden":
			ordenes++
		case "prioridad_concurrencia":
			prioridades++
		case "maximo_unidades":
			if c.Nuevo.LimiteUnidades.Modo != modoSinLimite || c.Nuevo.LimiteUnidades.Valor != nil {
				t.Fatal("sin limite ambiguo")
			}
		case "maximo_puntos":
			if c.Nuevo.LimitePuntos.Modo != modoSinLimite || c.Nuevo.LimitePuntos.Valor != nil {
				t.Fatal("sin limite ambiguo")
			}
		case "puntos_minimos":
			if c.Nuevo.Puntos == nil || c.Nuevo.Puntos.Micropuntos() != 0 {
				t.Fatal("cero omitido")
			}
		case "reparto_exceso":
			repartos++
			if c.Anterior != nil || c.Nuevo.Reparto.DesempateEntreReglas != DesempateExcesoPrioridadConcurrencia || c.Tipo != "modificacion" {
				t.Fatal("ausencia o desempate perdido")
			}
		}
	}
	if ordenes != 3 || prioridades != 1 || repartos != 1 {
		t.Fatal("orden o reparto invisibles")
	}
	identificadorReglaNueva := "regla_nueva"
	n.reglasExperiencia[0].clave = identificadorReglaNueva
	d, err = CompararConjuntos(a, n)
	if err != nil {
		t.Fatal(err)
	}
	altas, bajas := 0, 0
	for _, c := range d.Cambios {
		if c.Tipo == "alta" {
			altas++
			if c.Anterior != nil {
				t.Fatal("alta con anterior")
			}
		}
		if c.Tipo == "baja" {
			bajas++
			if c.Nuevo != nil {
				t.Fatal("baja con nuevo")
			}
		}
	}
	if altas != 1 || bajas != 1 {
		t.Fatal("renombre inferido")
	}
}

func TestValoresDiferenciaExperienciaSonUnaUnionCerrada(t *testing.T) {
	c, _ := parejaDiferenciaExperiencia(t)
	for _, o := range proyectarComparacionExperiencia(c) {
		for _, campo := range o.campos {
			if campo.valor != nil {
				if _, err := json.Marshal(campo.valor); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	cero := baremacion.Puntos{}
	entero := uint32(0)
	for _, v := range []ValorCambioExperiencia{{}, {Tipo: "libre", Puntos: &cero}, {Tipo: "puntos", Entero: &entero}, {Tipo: "puntos", Puntos: &cero, Entero: &entero}} {
		if _, err := json.Marshal(v); err == nil {
			t.Fatal("union vacia, abierta o ambigua")
		}
	}
	b, err := json.Marshal(ValorCambioExperiencia{Tipo: "puntos", Puntos: &cero})
	if err != nil || !bytes.Contains(b, []byte(`"puntos":"0"`)) {
		t.Fatal("cero no explicito")
	}
}
