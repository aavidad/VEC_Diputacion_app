package calculomeritos

import (
	"sort"
	"testing"

	"vec-diputacion-granada/internal/shared/baremacion"
)

func psicologiaV2(t *testing.T) (Conjunto, Entrada) {
	t.Helper()
	c, e := ejemplo(t, "meritos_reglas_a")
	c.Esquema, e.Esquema = EsquemaReglasV2, EsquemaEntradaV2
	c.CategoriaRef = "cat_00000000000000000000000000000021"
	c.PlazaRef = "pla_00000000000000000000000000000022"
	c.FechaCorteInclusiva = *fecha(t, 2024, 10, 8)
	c.MaximoTotal = puntos(t, 10_000_000)
	c.Secciones[0].MaximoPuntos = puntos(t, 10_000_000)
	c.Secciones[2].MaximoPuntos = puntos(t, 10_000_000)
	c.Reglas[0].MinimoUnidades = racional(t, 0, 1)
	c.Reglas[0].PuntosPorUnidad = puntos(t, 40_000)
	c.Reglas[0].MaximoPuntos = puntos(t, 10_000_000)
	c.Reglas[0].ConceptosRelacionados = []string{"con_00000000000000000000000000000031"}
	c.Reglas[0].Temporal = &PoliticaFormacion{
		DesdeBonificacion: *fecha(t, 2018, 10, 8), // RRHH lo fija desde bases, no se calcula aquí.
		EfectoAnterior:    "bonificar",
		PuntosRecientes:   *puntos(t, 80_000),
	}
	e.Meritos = []Merito{e.Meritos[0], e.Meritos[1], e.Meritos[3]}
	for i := range e.Meritos {
		e.Meritos[i].OrigenRef = []string{
			"ori_00000000000000000000000000000051",
			"ori_00000000000000000000000000000052",
			"ori_00000000000000000000000000000053",
		}[i]
		if i < 2 {
			e.Meritos[i].ConceptoRef = "con_00000000000000000000000000000031"
			e.Meritos[i].Unidades = racional(t, 10, 1)
		} else {
			e.Meritos[i].ConceptoRef = "con_00000000000000000000000000000032"
			e.Meritos[i].TipoHecho = "titulacion"
			e.Meritos[i].FechaObtencion = fecha(t, 2020, 1, 1)
		}
	}
	e.Meritos[0].TipoHecho = "curso_asistencia"
	e.Meritos[0].FechaObtencion = fecha(t, 2017, 1, 1)
	e.Meritos[1].TipoHecho = "curso_superacion"
	e.Meritos[1].FechaObtencion = fecha(t, 2022, 1, 1)
	return c, e
}

func TestFormacionPsicologiaTarifasYCategoriaExplicita(t *testing.T) {
	c, e := psicologiaV2(t)
	r, err := Calcular(c, e)
	if err != nil {
		t.Fatal(err)
	}
	if r.Estado != "completado" || r.CategoriaRef != c.CategoriaRef || r.PlazaRef != c.PlazaRef || r.Total == nil || r.Total.Micropuntos() != 3_200_000 {
		t.Fatalf("simulación inesperada: %+v", r)
	}
	if r.Reglas[0].BrutoRedondeado.Micropuntos() != 1_200_000 || r.Reglas[0].UnidadesRecientes.String() != "10/1" || r.Reglas[2].Puntos.Micropuntos() != 2_000_000 {
		t.Fatal("tarifa antigua/reciente o título mezclados")
	}
	if r.Meritos[0].Validez != "aplicable_segun_regla" || r.Meritos[0].PuntosProvisionales.Micropuntos() != 400_000 || r.Meritos[1].PuntosProvisionales.Micropuntos() != 800_000 || r.Meritos[0].Corte != "2024-10-08" {
		t.Fatal("desglose por hecho no explica regla y corte")
	}
	if r.Meritos[0].ReglaClave != "curso" || r.Meritos[2].ReglaClave != "titulo" {
		t.Fatal("regla de título confundida con curso")
	}
}

func TestFormacionRelacionPlazaYVentanaNoInventadas(t *testing.T) {
	c, e := psicologiaV2(t)
	e.Meritos[1].ConceptoRef = "con_00000000000000000000000000000039"
	r, err := Calcular(c, e)
	if err != nil {
		t.Fatal(err)
	}
	if r.Estado != "provisional" || r.Meritos[1].Validez != "no_aplicable_segun_regla" || r.Meritos[1].Codigo != "sin_relacion_publicada_plaza" || r.Meritos[1].PuntosProvisionales != nil || r.Total.Micropuntos() != 2_400_000 {
		t.Fatal("concepto ajeno puntuó o se ocultó la incertidumbre")
	}
	c, e = psicologiaV2(t)
	e.Meritos[0].VigenteHasta = fecha(t, 2019, 1, 1)
	r, err = Calcular(c, e)
	if err != nil || r.Total == nil || r.Total.Micropuntos() != 3_200_000 {
		t.Fatal("la vigencia factual no es una caducidad de puntuación publicada")
	}
	c, e = psicologiaV2(t)
	c.Reglas[0].Temporal.EfectoAnterior = "excluir"
	r, err = Calcular(c, e)
	if err != nil {
		t.Fatal(err)
	}
	if r.Meritos[0].Codigo != "fuera_ventana_publicada" || r.Meritos[0].Validez != "no_aplicable_segun_regla" || r.Total.Micropuntos() != 2_800_000 {
		t.Fatal("exclusión temporal no aplicada")
	}
	c, e = psicologiaV2(t)
	c.Reglas[0].Temporal.DesdeBonificacion = *fecha(t, 2025, 1, 1)
	if _, err := Calcular(c, e); err == nil {
		t.Fatal("ventana posterior al corte admitida")
	}
}

func TestMismaCarteraDosCategoriasSinTechoInventado(t *testing.T) {
	c, e := psicologiaV2(t)
	e.Meritos = e.Meritos[1:] // mismo curso reciente y título para ambas categorías
	c.MaximoTotal = nil
	for i := range c.Secciones {
		c.Secciones[i].MaximoPuntos = nil
	}
	for i := range c.Reglas {
		c.Reglas[i].MaximoPuntos = nil
		c.Reglas[i].MaximoUnidades = nil
		c.Reglas[i].MaximoElementos = nil
	}
	c.Reglas[0].Temporal.PuntosRecientes = *puntos(t, 450_000)
	r1, err := Calcular(c, e)
	if err != nil {
		t.Fatal(err)
	}
	if r1.Total == nil || r1.Total.Micropuntos() != 6_500_000 || r1.MaximoTotal != nil {
		t.Fatalf("primera categoría o ausencia de tope: %+v", r1)
	}
	c.Referencia = "rgl_00000000000000000000000000000041"
	c.CategoriaRef = "cat_00000000000000000000000000000042"
	c.PlazaRef = "pla_00000000000000000000000000000043"
	c.Reglas[0].Temporal.PuntosRecientes = *puntos(t, 390_000)
	r2, err := Calcular(c, e)
	if err != nil {
		t.Fatal(err)
	}
	if r2.Total == nil || r2.Total.Micropuntos() != 5_900_000 || r2.CategoriaRef == r1.CategoriaRef || r2.Meritos[0].Corte != r1.Meritos[0].Corte {
		t.Fatalf("segunda categoría reutilizó puntuación ajena: %+v", r2)
	}
}

func TestDosModalidadesMismoOrigenNoDuplicanCurso(t *testing.T) {
	c, e := psicologiaV2(t)
	e.Meritos[1].OrigenRef = e.Meritos[0].OrigenRef
	r, err := Calcular(c, e)
	if err != nil {
		t.Fatal(err)
	}
	if r.Estado != "bloqueado" || r.Total != nil || len(r.Incidencias) != 1 || r.Incidencias[0].Codigo != "merito_duplicado" {
		t.Fatalf("dos modalidades del mismo curso duplicaron puntos: %+v", r)
	}
}

func TestFormacionExactaRedondeaSoloTrasAgruparTarifa(t *testing.T) {
	c, e := psicologiaV2(t)
	e.Meritos = e.Meritos[:2]
	for i := range e.Meritos {
		e.Meritos[i].FechaObtencion = fecha(t, 2022, 1, 1)
		e.Meritos[i].Unidades = racional(t, 1, 2)
	}
	c.Reglas[0].PuntosPorUnidad = puntos(t, 1)
	c.Reglas[0].Temporal.PuntosRecientes = *puntos(t, 3)
	c.Reglas[0].Redondeo = baremacion.RedondeoExacto
	r, err := Calcular(c, e)
	if err != nil || r.Total == nil || r.Total.Micropuntos() != 3 {
		t.Fatalf("el desglose fraccionario forzó redondeo previo: %v %+v", err, r)
	}
	if r.Meritos[0].PuntosProvisionales != nil || r.Meritos[1].PuntosProvisionales != nil {
		t.Fatal("se publicó una fracción de micropunto por mérito")
	}
}

func TestTarifasMixtasUnSoloRedondeoPorRegla(t *testing.T) {
	c, e := psicologiaV2(t)
	e.Meritos = e.Meritos[:2]
	for i := range e.Meritos {
		e.Meritos[i].Unidades = racional(t, 1, 2)
	}
	c.Reglas[0].PuntosPorUnidad = puntos(t, 1)
	c.Reglas[0].Temporal.PuntosRecientes = *puntos(t, 3)
	for _, modo := range []baremacion.ModoRedondeo{baremacion.RedondeoExacto, baremacion.RedondeoMitadArriba} {
		c.Reglas[0].Redondeo = modo
		r, err := Calcular(c, e)
		if err != nil || r.Total == nil || r.Total.Micropuntos() != 2 || r.Reglas[0].BrutoRedondeado.Micropuntos() != 2 {
			t.Fatalf("dos tarifas redondeadas por separado, modo %s: %v %+v", modo, err, r)
		}
		if r.Meritos[0].PuntosProvisionales != nil || r.Meritos[1].PuntosProvisionales != nil || r.Meritos[0].CodigoPuntos != "acumulado_por_regla" {
			t.Fatal("el desglose publicó puntos individuales redondeados que no suman el total")
		}
	}
}

func TestEmpateBajoTopeNoDependeDelIdentificador(t *testing.T) {
	c, e := psicologiaV2(t)
	e.Meritos = e.Meritos[:2] // 10 horas antiguas y 10 recientes, tarifas distintas
	uno := uint32(1)
	c.Reglas[0].MaximoElementos = &uno
	for _, invertir := range []bool{false, true} {
		entrada := e
		entrada.Meritos = append([]Merito(nil), e.Meritos...)
		if invertir {
			entrada.Meritos[0].Referencia, entrada.Meritos[1].Referencia = entrada.Meritos[1].Referencia, entrada.Meritos[0].Referencia
			sort.Slice(entrada.Meritos, func(i, j int) bool { return entrada.Meritos[i].Referencia < entrada.Meritos[j].Referencia })
		}
		r, err := Calcular(c, entrada)
		if err != nil || r.Estado != "bloqueado" || r.Total != nil || len(r.Incidencias) != 1 || r.Incidencias[0].Codigo != "prioridad_bases_insuficiente" {
			t.Fatalf("identificador decidió bajo mayor_unidad: %v %+v", err, r)
		}
		c.SeleccionElementos = "mayor_puntuacion" // política expresa de las bases
		r, err = Calcular(c, entrada)
		if err != nil || r.Estado != "completado" || r.Total == nil || r.Total.Micropuntos() != 800_000 {
			t.Fatalf("prioridad publicada no seleccionó tarifa mayor: %v %+v", err, r)
		}
		for _, detalle := range r.Meritos {
			if detalle.Codigo == "maximo_elementos" && detalle.Validez != "no_aplicable_segun_regla" {
				t.Fatal("curso fuera del tope marcado aplicable")
			}
		}
		c.SeleccionElementos = "mayor_unidad"
	}
	c.Reglas[0].MaximoElementos = nil
	c.Reglas[0].MaximoUnidades = racional(t, 10, 1)
	r, err := Calcular(c, e)
	if err != nil || r.Estado != "bloqueado" || r.Total != nil {
		t.Fatal("tope de horas eligió un empate sin prioridad")
	}
}

func TestEmpatesEquivalentesConservanAplicabilidadSinAsignacionPorID(t *testing.T) {
	c, e := psicologiaV2(t)
	e.Meritos = e.Meritos[:2]
	e.Meritos[0].FechaObtencion = e.Meritos[1].FechaObtencion // ambas tarifas recientes
	uno := uint32(1)
	for _, caparElementos := range []bool{true, false} {
		if caparElementos {
			c.Reglas[0].MaximoElementos = &uno
			c.Reglas[0].MaximoUnidades = nil
		} else {
			c.Reglas[0].MaximoElementos = nil
			c.Reglas[0].MaximoUnidades = racional(t, 10, 1)
		}
		for _, invertir := range []bool{false, true} {
			entrada := e
			entrada.Meritos = append([]Merito(nil), e.Meritos...)
			if invertir {
				entrada.Meritos[0].Referencia, entrada.Meritos[1].Referencia = entrada.Meritos[1].Referencia, entrada.Meritos[0].Referencia
				sort.Slice(entrada.Meritos, func(i, j int) bool { return entrada.Meritos[i].Referencia < entrada.Meritos[j].Referencia })
			}
			r, err := Calcular(c, entrada)
			if err != nil || r.Estado != "completado" || r.Total == nil || r.Total.Micropuntos() != 800_000 {
				t.Fatalf("empate materialmente igual bloqueado o cambiado: %v %+v", err, r)
			}
			for _, detalle := range r.Meritos {
				if detalle.Validez != "aplicable_segun_regla" || detalle.Codigo != "tope_alcanzado_por_grupo_equivalente" || detalle.PuntosProvisionales != nil || detalle.CodigoPuntos != "asignacion_agrupada_por_tope" {
					t.Fatalf("desglose eligió hecho por ID: %+v", detalle)
				}
			}
		}
	}
}
