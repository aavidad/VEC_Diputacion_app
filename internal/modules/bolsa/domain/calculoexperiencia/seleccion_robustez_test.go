package calculoexperiencia

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
)

func TestSeleccionEsDeterministaAntePermutaciones(t *testing.T) {
	catalogoAmbito := referenciaSeleccionPrueba(t, "catalogo:seleccion:ambito", 1)
	catalogoRelacion := referenciaSeleccionPrueba(t, "catalogo:seleccion:relacion", 1)
	grupo := []grupoSeleccionPrueba{{
		clave: "grupo", orden: 1, modo: reglasbaremo.CoincidenciaReglasAcumular,
	}}
	reglaA := reglaSeleccionPrueba{
		clave: "regla_a", orden: 1, grupo: "grupo", prioridad: 2,
		criterios: []criterioSeleccionPrueba{{
			clave: "ambito", catalogo: catalogoAmbito, valores: []string{"local"},
		}},
	}
	reglaB := reglaSeleccionPrueba{
		clave: "regla_b", orden: 2, grupo: "grupo", prioridad: 1,
		criterios: []criterioSeleccionPrueba{
			{clave: "relacion", catalogo: catalogoRelacion, valores: []string{"funcionario"}},
			{clave: "ambito", catalogo: catalogoAmbito, valores: []string{"local"}},
		},
	}
	planUno := planSeleccionPrueba(t, grupo, []reglaSeleccionPrueba{reglaB, reglaA})
	planDos := planSeleccionPrueba(t, grupo, []reglaSeleccionPrueba{reglaA, reglaB})

	atributoAmbito := atributoSeleccionPrueba(t, "ambito", catalogoAmbito, "local")
	atributoRelacion := atributoSeleccionPrueba(t, "relacion", catalogoRelacion, "funcionario")
	tramoA := tramoSeleccionPrueba(t, "determinismo-a", []AtributoCatalogado{
		atributoRelacion, atributoAmbito,
	})
	tramoB := tramoSeleccionPrueba(t, "determinismo-b", []AtributoCatalogado{
		atributoAmbito, atributoRelacion,
	})
	entradaUno := entradaSeleccionPrueba(t, []TramoExperiencia{tramoB, tramoA})
	entradaDos := entradaSeleccionPrueba(t, []TramoExperiencia{tramoA, tramoB})

	primero, err := seleccionarAplicaciones(planUno, entradaUno)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := seleccionarAplicaciones(planDos, entradaDos)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(primero, segundo) || primero.evaluaciones != 6 {
		t.Fatalf("las permutaciones cambiaron la salida:\n%#v\n%#v", primero, segundo)
	}
	for repeticion := 0; repeticion < 20; repeticion++ {
		siguiente, err := seleccionarAplicaciones(planUno, entradaUno)
		if err != nil || !reflect.DeepEqual(primero, siguiente) {
			t.Fatalf("ejecucion %d no determinista: %#v, %v", repeticion, siguiente, err)
		}
	}
	for indice := 1; indice < len(primero.aplicaciones); indice++ {
		anterior := primero.aplicaciones[indice-1]
		actual := primero.aplicaciones[indice]
		comparacion := compararReferenciasSeleccion(anterior.tramo, actual.tramo)
		if comparacion > 0 || (comparacion == 0 && anterior.reglaClave >= actual.reglaClave) {
			t.Fatalf("salida no canonica: %#v", primero.aplicaciones)
		}
	}
}

func TestSeleccionImponePresupuestoTipadoSinProductoCartesiano(t *testing.T) {
	catalogoAmbito := referenciaSeleccionPrueba(t, "catalogo:seleccion:ambito", 1)
	catalogoRelacion := referenciaSeleccionPrueba(t, "catalogo:seleccion:relacion", 1)
	planDosCriterios := planSeleccionPrueba(
		t,
		[]grupoSeleccionPrueba{{
			clave: "grupo", orden: 1, modo: reglasbaremo.CoincidenciaReglasAcumular,
		}},
		[]reglaSeleccionPrueba{{
			clave: "regla", orden: 1, grupo: "grupo", prioridad: 1,
			criterios: []criterioSeleccionPrueba{
				{clave: "ambito", catalogo: catalogoAmbito, valores: []string{"local"}},
				{clave: "relacion", catalogo: catalogoRelacion, valores: []string{"funcionario"}},
			},
		}},
	)
	tramo := tramoSeleccionPrueba(t, "presupuesto", []AtributoCatalogado{
		atributoSeleccionPrueba(t, "ambito", catalogoAmbito, "local"),
		atributoSeleccionPrueba(t, "relacion", catalogoRelacion, "funcionario"),
	})
	entrada := entradaSeleccionPrueba(t, []TramoExperiencia{tramo})
	resultado, err := seleccionarAplicacionesConLimite(planDosCriterios, entrada, 1)
	if !errors.Is(err, ErrLimiteOperaciones) || !reflect.DeepEqual(resultado, seleccionExperiencia{}) {
		t.Fatalf("presupuesto no cerro la seleccion: %#v, %v", resultado, err)
	}

	// La segunda regla tiene otra ancla y no debe evaluarse si el tramo no la
	// presenta. Con limite uno, un barrido cartesiano fallaria.
	planIndexado := planSeleccionPrueba(
		t,
		[]grupoSeleccionPrueba{{
			clave: "grupo", orden: 1, modo: reglasbaremo.CoincidenciaReglasAcumular,
		}},
		[]reglaSeleccionPrueba{
			{
				clave: "regla_ambito", orden: 1, grupo: "grupo", prioridad: 1,
				criterios: []criterioSeleccionPrueba{{
					clave: "ambito", catalogo: catalogoAmbito, valores: []string{"local"},
				}},
			},
			{
				clave: "regla_relacion", orden: 2, grupo: "grupo", prioridad: 2,
				criterios: []criterioSeleccionPrueba{{
					clave: "relacion", catalogo: catalogoRelacion, valores: []string{"funcionario"},
				}},
			},
		},
	)
	tramoSoloAmbito := tramoSeleccionPrueba(t, "indice", []AtributoCatalogado{
		atributoSeleccionPrueba(t, "ambito", catalogoAmbito, "local"),
	})
	indexado, err := seleccionarAplicacionesConLimite(
		planIndexado,
		entradaSeleccionPrueba(t, []TramoExperiencia{tramoSoloAmbito}),
		1,
	)
	if err != nil || indexado.evaluaciones != 1 || len(indexado.aplicaciones) != 1 ||
		indexado.aplicaciones[0].reglaClave != "regla_ambito" {
		t.Fatalf("el indice evaluo reglas ajenas: %#v, %v", indexado, err)
	}
}

func TestSeleccionDevuelveCopiasDefensivas(t *testing.T) {
	catalogo := referenciaSeleccionPrueba(t, "catalogo:seleccion:ambito", 1)
	tramo := tramoSeleccionPrueba(t, "copias", []AtributoCatalogado{
		atributoSeleccionPrueba(t, "ambito", catalogo, "local"),
	})
	entrada := entradaSeleccionPrueba(t, []TramoExperiencia{tramo})

	prioridad, err := seleccionarAplicaciones(
		planDosReglasSeleccion(t, catalogo, reglasbaremo.CoincidenciaReglasElegirPrioridad),
		entrada,
	)
	if err != nil {
		t.Fatal(err)
	}
	aplicaciones := prioridad.aplicacionesCopia()
	descartes := prioridad.descartesCopia()
	aplicaciones[0].reglaClave = "alterada"
	descartes[0].reglaClave = "alterada"
	if prioridad.aplicacionesCopia()[0].reglaClave == "alterada" ||
		prioridad.descartesCopia()[0].reglaClave == "alterada" {
		t.Fatal("los getters permitieron mutar aplicaciones o descartes")
	}

	rechazo, err := seleccionarAplicaciones(
		planDosReglasSeleccion(t, catalogo, reglasbaremo.CoincidenciaReglasRechazar),
		entrada,
	)
	if err != nil {
		t.Fatal(err)
	}
	bloqueos := rechazo.bloqueosCopia()
	bloqueos[0].reglas[0] = "alterada"
	bloqueos[0].codigo = "alterado"
	if rechazo.bloqueosCopia()[0].reglas[0] == "alterada" ||
		rechazo.bloqueosCopia()[0].codigo == "alterado" {
		t.Fatal("el getter permitio mutar un bloqueo o su coleccion")
	}

	sinReglaTramo := tramoSeleccionPrueba(t, "sin-regla", nil)
	sinRegla, err := seleccionarAplicaciones(
		planSeleccionUnaRegla(t, catalogo),
		entradaSeleccionPrueba(t, []TramoExperiencia{sinReglaTramo}),
	)
	if err != nil {
		t.Fatal(err)
	}
	noCoincidencias := sinRegla.noCoincidenciasCopia()
	noCoincidencias[0].razon = "alterada"
	if sinRegla.noCoincidenciasCopia()[0].razon == "alterada" {
		t.Fatal("el getter permitio mutar una no coincidencia")
	}
}

func TestSeleccionValidaPlanEntradaYNoConservaValoresCatalogados(t *testing.T) {
	catalogo := referenciaSeleccionPrueba(t, "catalogo:seleccion:ambito", 1)
	plan := planSeleccionUnaRegla(t, catalogo)
	entradaVacia := entradaSeleccionPrueba(t, nil)
	if _, err := seleccionarAplicaciones(PlanExperiencia{}, entradaVacia); !errors.Is(err, ErrCompilacionPlanInvalido) {
		t.Fatalf("plan cero no rechazado: %v", err)
	}
	if _, err := seleccionarAplicaciones(plan, EntradaExperiencia{}); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("entrada cero no rechazada: %v", err)
	}

	const valorGobernado = "valor_ultrasecreto_no_persistir"
	planValor := planSeleccionPrueba(
		t,
		[]grupoSeleccionPrueba{{
			clave: "grupo", orden: 1, modo: reglasbaremo.CoincidenciaReglasAcumular,
		}},
		[]reglaSeleccionPrueba{{
			clave: "regla", orden: 1, grupo: "grupo", prioridad: 1,
			criterios: []criterioSeleccionPrueba{{
				clave: "ambito", catalogo: catalogo, valores: []string{valorGobernado},
			}},
		}},
	)
	tramo := tramoSeleccionPrueba(t, "minimizado", []AtributoCatalogado{
		atributoSeleccionPrueba(t, "ambito", catalogo, valorGobernado),
	})
	resultado, err := seleccionarAplicaciones(
		planValor,
		entradaSeleccionPrueba(t, []TramoExperiencia{tramo}),
	)
	if err != nil || strings.Contains(fmt.Sprintf("%#v", resultado), valorGobernado) {
		t.Fatalf("la salida conservo un valor catalogado: %#v, %v", resultado, err)
	}
}
