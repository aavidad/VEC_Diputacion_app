package calculoexperiencia

import (
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
)

func TestSeleccionAplicaANDEntreCriteriosORYAtributosAdicionales(t *testing.T) {
	catalogoAmbito := referenciaSeleccionPrueba(t, "catalogo:seleccion:ambito", 1)
	catalogoRelacion := referenciaSeleccionPrueba(t, "catalogo:seleccion:relacion", 1)
	catalogoExtra := referenciaSeleccionPrueba(t, "catalogo:seleccion:extra", 1)
	plan := planSeleccionPrueba(
		t,
		[]grupoSeleccionPrueba{{
			clave: "grupo_experiencia", orden: 1,
			modo: reglasbaremo.CoincidenciaReglasElegirPrioridad,
		}},
		[]reglaSeleccionPrueba{{
			clave: "regla_experiencia", orden: 1, grupo: "grupo_experiencia", prioridad: 1,
			criterios: []criterioSeleccionPrueba{
				{clave: "relacion", catalogo: catalogoRelacion, valores: []string{"funcionario"}},
				{
					clave: "ambito", catalogo: catalogoAmbito,
					valores: []string{"diputacion_granada", "administracion_local"},
				},
			},
		}},
	)

	pruebas := []struct {
		nombre       string
		atributos    []AtributoCatalogado
		aplicaciones int
		sinRegla     int
		evaluaciones numeroEvaluacionesSeleccion
	}{
		{
			nombre: "segundo valor OR y atributo extra",
			atributos: []AtributoCatalogado{
				atributoSeleccionPrueba(t, "extra", catalogoExtra, "ignorado"),
				atributoSeleccionPrueba(t, "relacion", catalogoRelacion, "funcionario"),
				atributoSeleccionPrueba(t, "ambito", catalogoAmbito, "administracion_local"),
			},
			aplicaciones: 1,
			evaluaciones: 2,
		},
		{
			nombre: "un criterio AND no coincide",
			atributos: []AtributoCatalogado{
				atributoSeleccionPrueba(t, "ambito", catalogoAmbito, "diputacion_granada"),
				atributoSeleccionPrueba(t, "relacion", catalogoRelacion, "laboral"),
			},
			sinRegla:     1,
			evaluaciones: 2,
		},
		{
			nombre: "criterio ausente",
			atributos: []AtributoCatalogado{
				atributoSeleccionPrueba(t, "ambito", catalogoAmbito, "diputacion_granada"),
			},
			sinRegla:     1,
			evaluaciones: 2,
		},
		{
			nombre: "ancla ausente evita cartesiano",
			atributos: []AtributoCatalogado{
				atributoSeleccionPrueba(t, "relacion", catalogoRelacion, "funcionario"),
			},
			sinRegla: 1,
		},
	}

	for indice, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			tramo := tramoSeleccionPrueba(t, prueba.nombre, prueba.atributos)
			entrada := entradaSeleccionPrueba(t, []TramoExperiencia{tramo})
			resultado, err := seleccionarAplicaciones(plan, entrada)
			if err != nil {
				t.Fatalf("seleccionar: %v", err)
			}
			if resultado.bloqueada() || len(resultado.aplicaciones) != prueba.aplicaciones ||
				len(resultado.noCoincidencias) != prueba.sinRegla ||
				resultado.evaluaciones != prueba.evaluaciones {
				t.Fatalf("caso %d: resultado inesperado: %#v", indice, resultado)
			}
			if prueba.aplicaciones == 1 {
				aplicacion := resultado.aplicaciones[0]
				if aplicacion.tramo != tramo.Referencia() ||
					aplicacion.reglaClave != "regla_experiencia" ||
					aplicacion.razon != motivoAplicacionUnica {
					t.Fatalf("aplicacion inesperada: %#v", aplicacion)
				}
			}
			if prueba.sinRegla == 1 &&
				resultado.noCoincidencias[0].razon != motivoNingunaReglaCoincidente {
				t.Fatalf("no coincidencia sin codigo: %#v", resultado.noCoincidencias[0])
			}
		})
	}
}

func TestSeleccionBloqueaCatalogoIncompatibleAntesDeCoincidencia(t *testing.T) {
	catalogo := referenciaSeleccionPrueba(t, "catalogo:seleccion:ambito", 1)
	plan := planSeleccionUnaRegla(t, catalogo)
	pruebas := []struct {
		nombre   string
		catalogo reglasbaremo.ReferenciaVersionada
	}{
		{
			nombre:   "referencia",
			catalogo: referenciaSeleccionPrueba(t, "catalogo:seleccion:ambito_otro", 1),
		},
		{
			nombre:   "version",
			catalogo: referenciaSeleccionPrueba(t, "catalogo:seleccion:ambito", 2),
		},
		{
			nombre: "huella",
			catalogo: referenciaSeleccionConHuellaPrueba(
				t, "catalogo:seleccion:ambito", 1, 'f',
			),
		},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			tramo := tramoSeleccionPrueba(t, "catalogo-"+prueba.nombre, []AtributoCatalogado{
				atributoSeleccionPrueba(t, "ambito", prueba.catalogo, "valor_no_admitido"),
			})
			resultado, err := seleccionarAplicaciones(
				plan,
				entradaSeleccionPrueba(t, []TramoExperiencia{tramo}),
			)
			if err != nil {
				t.Fatalf("un bloqueo de negocio no es error tecnico: %v", err)
			}
			if !resultado.bloqueada() || len(resultado.bloqueos) != 1 ||
				resultado.bloqueos[0].codigo != bloqueoCatalogoIncompatible ||
				resultado.bloqueos[0].claveGobernada != "ambito" ||
				resultado.bloqueos[0].tramo != tramo.Referencia() ||
				len(resultado.aplicaciones) != 0 || len(resultado.noCoincidencias) != 0 {
				t.Fatalf("bloqueo inesperado: %#v", resultado)
			}
		})
	}
}

func TestSeleccionBloqueaCualquierClaveGobernadaAunqueFalteElAncla(t *testing.T) {
	catalogoAmbito := referenciaSeleccionPrueba(t, "catalogo:seleccion:ambito", 1)
	catalogoRelacion := referenciaSeleccionPrueba(t, "catalogo:seleccion:relacion", 1)
	plan := planSeleccionPrueba(
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
	incompatible := referenciaSeleccionPrueba(t, "catalogo:seleccion:relacion", 2)
	tramo := tramoSeleccionPrueba(t, "sin-ancla", []AtributoCatalogado{
		atributoSeleccionPrueba(t, "relacion", incompatible, "funcionario"),
	})
	resultado, err := seleccionarAplicaciones(
		plan,
		entradaSeleccionPrueba(t, []TramoExperiencia{tramo}),
	)
	if err != nil || !resultado.bloqueada() || len(resultado.bloqueos) != 1 ||
		resultado.bloqueos[0].claveGobernada != "relacion" || resultado.evaluaciones != 0 {
		t.Fatalf("la clave gobernada no bloqueo antes del indice: %#v, %v", resultado, err)
	}
}

func TestSeleccionRechazaComoErrorTecnicoCatalogosDistintosEnElPlan(t *testing.T) {
	compartido := referenciaSeleccionPrueba(t, "catalogo:seleccion:ambito", 1)
	plan := planDosReglasSeleccion(
		t,
		compartido,
		reglasbaremo.CoincidenciaReglasElegirPrioridad,
	)
	alterado := plan
	alterado.reglas = append([]reglasbaremo.ReglaExperiencia(nil), plan.reglas...)
	original := alterado.reglas[1]
	criterio := original.Criterios()[0]
	criterioAlterado := debeSeleccion(reglasbaremo.NuevoCriterioExperiencia(
		criterio.Clave(),
		referenciaSeleccionPrueba(t, "catalogo:seleccion:ambito", 2),
		criterio.Valores(),
	))
	alterado.reglas[1] = debeSeleccion(reglasbaremo.NuevaReglaExperiencia(
		original.Clave(), original.Definicion(), original.SeccionClave(), original.Orden(),
		[]reglasbaremo.CriterioExperiencia{criterioAlterado},
		original.GrupoConcurrenciaClave(), original.PrioridadConcurrencia(),
		original.UnidadTemporal(), original.Jornada(), original.Restos(), original.Redondeo(),
		original.PuntosPorUnidad(), original.MaximoUnidades(), original.MaximoPuntos(),
	))

	_, err := seleccionarAplicaciones(alterado, entradaSeleccionPrueba(t, nil))
	if !errors.Is(err, ErrCatalogoCriterioIncompatible) {
		t.Fatalf("error=%v; se esperaba catalogos incompatibles", err)
	}
}

func planSeleccionUnaRegla(
	t *testing.T,
	catalogo reglasbaremo.ReferenciaVersionada,
) PlanExperiencia {
	t.Helper()
	return planSeleccionPrueba(
		t,
		[]grupoSeleccionPrueba{{
			clave: "grupo", orden: 1, modo: reglasbaremo.CoincidenciaReglasAcumular,
		}},
		[]reglaSeleccionPrueba{{
			clave: "regla", orden: 1, grupo: "grupo", prioridad: 1,
			criterios: []criterioSeleccionPrueba{{
				clave: "ambito", catalogo: catalogo,
				valores: []string{"administracion_local", "diputacion_granada"},
			}},
		}},
	)
}
