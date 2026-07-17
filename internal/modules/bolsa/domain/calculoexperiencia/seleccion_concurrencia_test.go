package calculoexperiencia

import (
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
)

func TestSeleccionResuelveLasTresPoliticasDeCoincidenciaV1(t *testing.T) {
	catalogo := referenciaSeleccionPrueba(t, "catalogo:seleccion:ambito", 1)
	pruebas := []struct {
		nombre       string
		modo         reglasbaremo.ModoCoincidenciaReglas
		aplicaciones int
		descartes    int
		bloqueo      codigoBloqueoSeleccion
	}{
		{
			nombre:  "rechazar",
			modo:    reglasbaremo.CoincidenciaReglasRechazar,
			bloqueo: bloqueoCoincidenciaRechazada,
		},
		{
			nombre:       "prioridad",
			modo:         reglasbaremo.CoincidenciaReglasElegirPrioridad,
			aplicaciones: 1,
			descartes:    1,
		},
		{
			nombre:       "acumular",
			modo:         reglasbaremo.CoincidenciaReglasAcumular,
			aplicaciones: 2,
		},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			plan := planDosReglasSeleccion(t, catalogo, prueba.modo)
			tramo := tramoSeleccionPrueba(t, "coincidencia-"+prueba.nombre, []AtributoCatalogado{
				atributoSeleccionPrueba(t, "ambito", catalogo, "local"),
			})
			resultado, err := seleccionarAplicaciones(
				plan,
				entradaSeleccionPrueba(t, []TramoExperiencia{tramo}),
			)
			if err != nil {
				t.Fatalf("seleccionar: %v", err)
			}
			if len(resultado.aplicaciones) != prueba.aplicaciones ||
				len(resultado.descartes) != prueba.descartes {
				t.Fatalf("resultado inesperado: %#v", resultado)
			}
			if prueba.bloqueo != "" {
				if !resultado.bloqueada() || len(resultado.bloqueos) != 1 ||
					resultado.bloqueos[0].codigo != prueba.bloqueo ||
					!cadenasSeleccionIguales(
						resultado.bloqueos[0].reglas,
						[]string{"regla_prioridad_alta", "regla_prioridad_baja"},
					) {
					t.Fatalf("bloqueo inesperado: %#v", resultado.bloqueos)
				}
				return
			}
			if resultado.bloqueada() {
				t.Fatalf("politica admitida bloqueada: %#v", resultado.bloqueos)
			}
			switch prueba.modo {
			case reglasbaremo.CoincidenciaReglasElegirPrioridad:
				aplicacion := resultado.aplicaciones[0]
				descarte := resultado.descartes[0]
				if aplicacion.reglaClave != "regla_prioridad_alta" || aplicacion.prioridad != 1 ||
					aplicacion.razon != motivoAplicacionPrioridad ||
					descarte.reglaClave != "regla_prioridad_baja" ||
					descarte.reglaSeleccionada != "regla_prioridad_alta" ||
					descarte.razon != motivoDescartePrioridadInferior {
					t.Fatalf("prioridad no respetada: %#v / %#v", aplicacion, descarte)
				}
			case reglasbaremo.CoincidenciaReglasAcumular:
				if resultado.aplicaciones[0].reglaClave != "regla_prioridad_alta" ||
					resultado.aplicaciones[1].reglaClave != "regla_prioridad_baja" ||
					resultado.aplicaciones[0].razon != motivoAplicacionAcumulada ||
					resultado.aplicaciones[1].razon != motivoAplicacionAcumulada {
					t.Fatalf("acumulacion no canonica: %#v", resultado.aplicaciones)
				}
			}
		})
	}
}

func TestSeleccionBloqueaReglasCoincidentesEnGruposDistintos(t *testing.T) {
	catalogo := referenciaSeleccionPrueba(t, "catalogo:seleccion:ambito", 1)
	plan := planSeleccionPrueba(
		t,
		[]grupoSeleccionPrueba{
			{clave: "grupo_b", orden: 2, modo: reglasbaremo.CoincidenciaReglasAcumular},
			{clave: "grupo_a", orden: 1, modo: reglasbaremo.CoincidenciaReglasElegirPrioridad},
		},
		[]reglaSeleccionPrueba{
			{
				clave: "regla_b", orden: 2, grupo: "grupo_b", prioridad: 1,
				criterios: []criterioSeleccionPrueba{{
					clave: "ambito", catalogo: catalogo, valores: []string{"local"},
				}},
			},
			{
				clave: "regla_a", orden: 1, grupo: "grupo_a", prioridad: 1,
				criterios: []criterioSeleccionPrueba{{
					clave: "ambito", catalogo: catalogo, valores: []string{"local"},
				}},
			},
		},
	)
	tramo := tramoSeleccionPrueba(t, "grupos", []AtributoCatalogado{
		atributoSeleccionPrueba(t, "ambito", catalogo, "local"),
	})
	resultado, err := seleccionarAplicaciones(
		plan,
		entradaSeleccionPrueba(t, []TramoExperiencia{tramo}),
	)
	if err != nil || !resultado.bloqueada() || len(resultado.bloqueos) != 1 ||
		resultado.bloqueos[0].codigo != bloqueoGruposDistintos ||
		!cadenasSeleccionIguales(resultado.bloqueos[0].reglas, []string{"regla_a", "regla_b"}) ||
		len(resultado.aplicaciones) != 0 {
		t.Fatalf("cruce de grupos no bloqueado: %#v, %v", resultado, err)
	}
}

func TestSeleccionUnaSolaReglaNoInventaResolucion(t *testing.T) {
	catalogo := referenciaSeleccionPrueba(t, "catalogo:seleccion:ambito", 1)
	plan := planSeleccionUnaRegla(t, catalogo)
	tramo := tramoSeleccionPrueba(t, "unica", []AtributoCatalogado{
		atributoSeleccionPrueba(t, "ambito", catalogo, "diputacion_granada"),
	})
	resultado, err := seleccionarAplicaciones(
		plan,
		entradaSeleccionPrueba(t, []TramoExperiencia{tramo}),
	)
	if err != nil || resultado.bloqueada() || len(resultado.aplicaciones) != 1 ||
		resultado.aplicaciones[0].razon != motivoAplicacionUnica ||
		len(resultado.descartes) != 0 {
		t.Fatalf("coincidencia unica inesperada: %#v, %v", resultado, err)
	}
}

func planDosReglasSeleccion(
	t *testing.T,
	catalogo reglasbaremo.ReferenciaVersionada,
	modo reglasbaremo.ModoCoincidenciaReglas,
) PlanExperiencia {
	t.Helper()
	return planSeleccionPrueba(
		t,
		[]grupoSeleccionPrueba{{clave: "grupo", orden: 1, modo: modo}},
		[]reglaSeleccionPrueba{
			{
				// El orden de regla favorece deliberadamente a la prioridad peor.
				clave: "regla_prioridad_baja", orden: 1, grupo: "grupo", prioridad: 9,
				criterios: []criterioSeleccionPrueba{{
					clave: "ambito", catalogo: catalogo, valores: []string{"local"},
				}},
			},
			{
				clave: "regla_prioridad_alta", orden: 2, grupo: "grupo", prioridad: 1,
				criterios: []criterioSeleccionPrueba{{
					clave: "ambito", catalogo: catalogo, valores: []string{"local"},
				}},
			},
		},
	)
}

func cadenasSeleccionIguales(izquierda, derecha []string) bool {
	if len(izquierda) != len(derecha) {
		return false
	}
	for indice := range izquierda {
		if izquierda[indice] != derecha[indice] {
			return false
		}
	}
	return true
}
