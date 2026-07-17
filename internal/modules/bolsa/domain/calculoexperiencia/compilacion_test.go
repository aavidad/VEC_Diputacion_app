package calculoexperiencia

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestCompilarFijaIdentidadCorteYOrdenCanonico(t *testing.T) {
	conjunto := conjuntoOrdenadoCompilacion(t)
	plan, err := Compilar(conjunto)
	if err != nil {
		t.Fatalf("compilar: %v", err)
	}
	esperada, err := conjunto.ReferenciaVersionada()
	if err != nil {
		t.Fatal(err)
	}
	if plan.Conjunto() != esperada {
		t.Fatalf("referencia fijada distinta: %#v != %#v", plan.Conjunto(), esperada)
	}
	if plan.FechaCorte().String() != conjunto.FechaCorte().String() {
		t.Fatalf("corte distinto: %s", plan.FechaCorte().String())
	}
	if clavesSecciones(plan.Secciones()) != "experiencia_publica,experiencia_privada" {
		t.Fatalf("secciones fuera de orden: %s", clavesSecciones(plan.Secciones()))
	}
	if clavesGrupos(plan.GruposConcurrencia()) != "grupo_publico,grupo_privado" {
		t.Fatalf("grupos fuera de orden: %s", clavesGrupos(plan.GruposConcurrencia()))
	}
	if clavesReglas(plan.Reglas()) != "regla_publica,regla_privada" {
		t.Fatalf("reglas fuera de orden: %s", clavesReglas(plan.Reglas()))
	}
	if err := plan.Validar(); err != nil {
		t.Fatalf("plan compilado invalido: %v", err)
	}
}

func TestCompilarEsDeterministaYDefiendeColecciones(t *testing.T) {
	conjunto := conjuntoOrdenadoCompilacion(t)
	primero, err := Compilar(conjunto)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := Compilar(conjunto)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(primero, segundo) {
		t.Fatal("el mismo conjunto produjo planes distintos")
	}

	secciones := primero.Secciones()
	grupos := primero.GruposConcurrencia()
	reglas := primero.Reglas()
	criterios := reglas[0].Criterios()
	secciones[0] = reglasbaremo.SeccionBaremo{}
	grupos[0] = reglasbaremo.GrupoConcurrenciaExperiencia{}
	reglas[0] = reglasbaremo.ReglaExperiencia{}
	criterios[0] = reglasbaremo.CriterioExperiencia{}

	if primero.Secciones()[0].Clave() != "experiencia_publica" ||
		primero.GruposConcurrencia()[0].Clave() != "grupo_publico" ||
		primero.Reglas()[0].Clave() != "regla_publica" ||
		primero.Reglas()[0].Criterios()[0].Clave() != "ambito" {
		t.Fatal("un acceso externo altero el plan")
	}
}

func TestCompilarExigeUnCatalogoPorClaveDeCriterio(t *testing.T) {
	referencia := "catalogo:ambito:compartido"
	compartido := referenciaCompilacion(t, referencia, 4)

	t.Run("mismo_catalogo_compartido", func(t *testing.T) {
		conjunto := conjuntoCatalogosCriterioCompilacion(t, compartido, compartido)
		if _, err := Compilar(conjunto); err != nil {
			t.Fatalf("catalogo compartido rechazado: %v", err)
		}
	})

	casos := []struct {
		nombre  string
		segundo reglasbaremo.ReferenciaVersionada
	}{
		{
			nombre:  "referencia_distinta",
			segundo: referenciaCompilacion(t, "catalogo:ambito:otro", 4),
		},
		{
			nombre:  "version_distinta",
			segundo: referenciaCompilacion(t, referencia, 5),
		},
		{
			nombre: "huella_distinta",
			segundo: referenciaCompilacionConSemillaHuella(
				t, referencia, 4, referencia+":contenido_distinto",
			),
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := Compilar(conjuntoCatalogosCriterioCompilacion(
				t, compartido, caso.segundo,
			))
			exigirErrorCompilacion(
				t, err,
				ErrCatalogoCriterioIncompatible,
				CodigoCatalogoCriterioIncompatible,
			)
		})
	}
}

func TestPlanValidarDetectaCatalogoDeCriterioAlterado(t *testing.T) {
	compartido := referenciaCompilacion(t, "catalogo:ambito:compartido", 4)
	plan, err := Compilar(conjuntoCatalogosCriterioCompilacion(
		t, compartido, compartido,
	))
	if err != nil {
		t.Fatal(err)
	}

	alterado := plan
	alterado.reglas = append([]reglasbaremo.ReglaExperiencia(nil), plan.reglas...)
	alterado.reglas[1] = reglaConCatalogoCompilacion(
		t,
		alterado.reglas[1],
		referenciaCompilacion(t, "catalogo:ambito:alterado", 4),
	)
	err = alterado.Validar()
	exigirErrorCompilacion(
		t, err,
		ErrCatalogoCriterioIncompatible,
		CodigoCatalogoCriterioIncompatible,
	)
}

func TestCompilarRechazaPuertasNoGobernadasV1(t *testing.T) {
	pruebas := []struct {
		nombre   string
		ajustar  func(*configuracionCompilacionPrueba)
		esperado error
		codigo   CodigoError
	}{
		{
			nombre: "jornada por horas antes que unidad base",
			ajustar: func(c *configuracionCompilacionPrueba) {
				uno := racionalCompilacion(t, 1, 1)
				c.unidad = debeCompilacion(reglasbaremo.NuevaPoliticaUnidadTemporal(
					reglasbaremo.UnidadTemporalHora, reglasbaremo.UnidadTemporalHora,
					uno, reglasbaremo.ExtremoFinalInclusivo,
				))
				c.jornada = debeCompilacion(reglasbaremo.NuevaPoliticaJornada(
					reglasbaremo.JornadaPorHoras,
				))
			},
			esperado: ErrJornadaNoSoportada,
			codigo:   CodigoJornadaNoSoportada,
		},
		{
			nombre: "unidad base hora",
			ajustar: func(c *configuracionCompilacionPrueba) {
				c.unidad = debeCompilacion(reglasbaremo.NuevaPoliticaUnidadTemporal(
					reglasbaremo.UnidadTemporalHora, reglasbaremo.UnidadTemporalHora,
					racionalCompilacion(t, 1, 1), reglasbaremo.ExtremoFinalExclusivo,
				))
			},
			esperado: ErrUnidadBaseNoSoportada,
			codigo:   CodigoUnidadBaseNoSoportada,
		},
		{
			nombre: "redondeo por seccion",
			ajustar: func(c *configuracionCompilacionPrueba) {
				c.redondeo = debeCompilacion(reglasbaremo.NuevaPoliticaRedondeo(
					reglasbaremo.RedondearPorSeccion, baremacion.RedondeoTruncar,
				))
			},
			esperado: ErrRedondeoNoSoportado,
			codigo:   CodigoRedondeoNoSoportado,
		},
		{
			nombre: "redondeo total",
			ajustar: func(c *configuracionCompilacionPrueba) {
				c.redondeo = debeCompilacion(reglasbaremo.NuevaPoliticaRedondeo(
					reglasbaremo.RedondearEnTotal, baremacion.RedondeoMitadAlPar,
				))
			},
			esperado: ErrRedondeoNoSoportado,
			codigo:   CodigoRedondeoNoSoportado,
		},
		{
			nombre: "minimo positivo de seccion",
			ajustar: func(c *configuracionCompilacionPrueba) {
				c.minimoSeccion = puntosCompilacion(t, 1)
			},
			esperado: ErrMinimoSeccionNoSoportado,
			codigo:   CodigoMinimoSeccionNoSoportado,
		},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			configuracion := configuracionBaseCompilacion(t)
			prueba.ajustar(&configuracion)
			_, err := Compilar(conjuntoCompilacionPrueba(t, configuracion))
			exigirErrorCompilacion(t, err, prueba.esperado, prueba.codigo)
		})
	}
}

func TestCompilarRechazaConjuntoInvalidoYPlanCero(t *testing.T) {
	_, err := Compilar(reglasbaremo.ConjuntoReglasBaremo{})
	exigirErrorCompilacion(
		t, err, ErrCompilacionConjuntoInvalido, CodigoCompilacionConjuntoInvalido,
	)
	err = (PlanExperiencia{}).Validar()
	exigirErrorCompilacion(t, err, ErrCompilacionPlanInvalido, CodigoCompilacionPlanInvalido)
}

func TestPlanValidarRederivaHuellaYRelacionesDelConjunto(t *testing.T) {
	plan, err := Compilar(conjuntoOrdenadoCompilacion(t))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("contenido_distinto_misma_referencia_de_plan", func(t *testing.T) {
		alterado := plan
		alterado.secciones = append([]reglasbaremo.SeccionBaremo(nil), plan.secciones...)
		original := alterado.secciones[0]
		alterado.secciones[0] = debeCompilacion(reglasbaremo.NuevaSeccionBaremo(
			original.Clave(), original.Definicion(), original.Orden(),
			original.PuntosMinimos(), puntosCompilacion(t, 90_000_000),
		))
		exigirPlanInvalido(t, alterado.Validar())
	})

	t.Run("referencia_de_definicion_reutilizada", func(t *testing.T) {
		alterado := plan
		alterado.secciones = append([]reglasbaremo.SeccionBaremo(nil), plan.secciones...)
		primera, segunda := alterado.secciones[0], alterado.secciones[1]
		alterado.secciones[1] = debeCompilacion(reglasbaremo.NuevaSeccionBaremo(
			segunda.Clave(), primera.Definicion(), segunda.Orden(),
			segunda.PuntosMinimos(), segunda.PuntosMaximos(),
		))
		exigirPlanInvalido(t, alterado.Validar())
	})

	t.Run("maximo_regla_supera_seccion", func(t *testing.T) {
		alterado := plan
		alterado.reglas = append([]reglasbaremo.ReglaExperiencia(nil), plan.reglas...)
		original := alterado.reglas[0]
		maximo := debeCompilacion(reglasbaremo.NuevoLimitePuntos(
			puntosCompilacion(t, 101_000_000),
		))
		alterado.reglas[0] = debeCompilacion(reglasbaremo.NuevaReglaExperiencia(
			original.Clave(), original.Definicion(), original.SeccionClave(), original.Orden(),
			original.Criterios(), original.GrupoConcurrenciaClave(), original.PrioridadConcurrencia(),
			original.UnidadTemporal(), original.Jornada(), original.Restos(), original.Redondeo(),
			original.PuntosPorUnidad(), original.MaximoUnidades(), maximo,
		))
		exigirPlanInvalido(t, alterado.Validar())
	})

	t.Run("volumen_no_acotado", func(t *testing.T) {
		alterado := plan
		alterado.reglas = make([]reglasbaremo.ReglaExperiencia, 1_025)
		for indice := range alterado.reglas {
			alterado.reglas[indice] = plan.reglas[0]
		}
		exigirPlanInvalido(t, alterado.Validar())
	})
}

func TestCompilarAdmiteJornadasExtremosRestosYRedondeosV1(t *testing.T) {
	umbral := debeCompilacion(baremacion.NuevaFraccionJornada(1, 2))
	jornadas := []reglasbaremo.PoliticaJornada{
		debeCompilacion(reglasbaremo.NuevaPoliticaJornada(reglasbaremo.JornadaProporcional)),
		debeCompilacion(reglasbaremo.NuevaPoliticaJornada(reglasbaremo.JornadaIntegra)),
		debeCompilacion(reglasbaremo.NuevaPoliticaJornadaDesdeUmbral(umbral)),
		debeCompilacion(reglasbaremo.NuevaPoliticaJornada(reglasbaremo.JornadaProtegidaIntegra)),
	}
	for _, jornada := range jornadas {
		configuracion := configuracionBaseCompilacion(t)
		configuracion.jornada = jornada
		if _, err := Compilar(conjuntoCompilacionPrueba(t, configuracion)); err != nil {
			t.Fatalf("jornada %s rechazada: %v", jornada.Modo(), err)
		}
	}

	for _, extremo := range []reglasbaremo.TratamientoExtremoFinal{
		reglasbaremo.ExtremoFinalExclusivo, reglasbaremo.ExtremoFinalInclusivo,
	} {
		configuracion := configuracionBaseCompilacion(t)
		configuracion.unidad = debeCompilacion(reglasbaremo.NuevaPoliticaUnidadTemporal(
			reglasbaremo.UnidadTemporalDia, reglasbaremo.UnidadTemporalMes,
			racionalCompilacion(t, 30, 1), extremo,
		))
		plan, err := Compilar(conjuntoCompilacionPrueba(t, configuracion))
		if err != nil || plan.Reglas()[0].UnidadTemporal().ExtremoFinal() != extremo {
			t.Fatalf("extremo %s no preservado: %v", extremo, err)
		}
	}

	for _, modo := range []reglasbaremo.ModoRestos{
		reglasbaremo.RestosConservarExactos,
		reglasbaremo.RestosAcumularPorRegla,
		reglasbaremo.RestosDescartarPorPeriodo,
		reglasbaremo.RestosDescartarPorRegla,
	} {
		configuracion := configuracionBaseCompilacion(t)
		configuracion.restos = debeCompilacion(reglasbaremo.NuevaPoliticaRestos(modo))
		plan, err := Compilar(conjuntoCompilacionPrueba(t, configuracion))
		if err != nil || plan.Reglas()[0].Restos().Modo() != modo {
			t.Fatalf("restos %s no preservados: %v", modo, err)
		}
	}

	for _, momento := range []reglasbaremo.MomentoRedondeo{
		reglasbaremo.RedondearPorPeriodo, reglasbaremo.RedondearPorRegla,
	} {
		configuracion := configuracionBaseCompilacion(t)
		if momento == reglasbaremo.RedondearPorPeriodo {
			configuracion.maximoUnidades = reglasbaremo.SinLimiteUnidades()
		}
		configuracion.redondeo = debeCompilacion(reglasbaremo.NuevaPoliticaRedondeo(
			momento, baremacion.RedondeoHaciaArriba,
		))
		if _, err := Compilar(conjuntoCompilacionPrueba(t, configuracion)); err != nil {
			t.Fatalf("redondeo %s rechazado: %v", momento, err)
		}
	}
}

func TestCompilarValidaMatrizRestosYRedondeoV1(t *testing.T) {
	pruebas := []struct {
		nombre   string
		restos   reglasbaremo.ModoRestos
		momento  reglasbaremo.MomentoRedondeo
		admitida bool
	}{
		{"exactos_periodo", reglasbaremo.RestosConservarExactos, reglasbaremo.RedondearPorPeriodo, true},
		{"exactos_regla", reglasbaremo.RestosConservarExactos, reglasbaremo.RedondearPorRegla, true},
		{"acumular_regla_periodo", reglasbaremo.RestosAcumularPorRegla, reglasbaremo.RedondearPorPeriodo, false},
		{"acumular_regla_regla", reglasbaremo.RestosAcumularPorRegla, reglasbaremo.RedondearPorRegla, true},
		{"descartar_periodo_periodo", reglasbaremo.RestosDescartarPorPeriodo, reglasbaremo.RedondearPorPeriodo, true},
		{"descartar_periodo_regla", reglasbaremo.RestosDescartarPorPeriodo, reglasbaremo.RedondearPorRegla, true},
		{"descartar_regla_periodo", reglasbaremo.RestosDescartarPorRegla, reglasbaremo.RedondearPorPeriodo, false},
		{"descartar_regla_regla", reglasbaremo.RestosDescartarPorRegla, reglasbaremo.RedondearPorRegla, true},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			configuracion := configuracionBaseCompilacion(t)
			configuracion.restos = debeCompilacion(reglasbaremo.NuevaPoliticaRestos(prueba.restos))
			configuracion.redondeo = debeCompilacion(reglasbaremo.NuevaPoliticaRedondeo(
				prueba.momento, baremacion.RedondeoMitadAlPar,
			))
			// El tope temporal tiene su propia incompatibilidad con el redondeo
			// por periodo y no debe ocultar el resultado de esta matriz.
			configuracion.maximoUnidades = reglasbaremo.SinLimiteUnidades()
			_, err := Compilar(conjuntoCompilacionPrueba(t, configuracion))
			if prueba.admitida {
				if err != nil {
					t.Fatalf("combinacion admitida rechazada: %v", err)
				}
				return
			}
			exigirErrorCompilacion(
				t, err, ErrRestosRedondeoNoSoportados, CodigoRestosRedondeoNoSoportados,
			)
		})
	}
}

func TestSemanticaRestosV1ExplicitaFronteraYTratamiento(t *testing.T) {
	pruebas := []struct {
		modo        reglasbaremo.ModoRestos
		frontera    fronteraRestosV1
		tratamiento tratamientoRestosV1
	}{
		{reglasbaremo.RestosConservarExactos, fronteraRestosExactaV1, tratamientoRestosConservarV1},
		{reglasbaremo.RestosAcumularPorRegla, fronteraRestosReglaV1, tratamientoRestosConservarV1},
		{reglasbaremo.RestosDescartarPorPeriodo, fronteraRestosPeriodoV1, tratamientoRestosDescartarV1},
		{reglasbaremo.RestosDescartarPorRegla, fronteraRestosReglaV1, tratamientoRestosDescartarV1},
	}
	for _, prueba := range pruebas {
		obtenida, err := semanticaRestosCompilableV1(prueba.modo)
		if err != nil || obtenida.frontera != prueba.frontera ||
			obtenida.tratamiento != prueba.tratamiento {
			t.Fatalf("semantica de %s incompleta: %#v, %v", prueba.modo, obtenida, err)
		}
	}
	_, err := semanticaRestosCompilableV1(reglasbaremo.ModoRestos("inventado"))
	exigirErrorCompilacion(
		t, err, ErrRestosRedondeoNoSoportados, CodigoRestosRedondeoNoSoportados,
	)
}

func TestCompilarCierraCoincidenciasV1(t *testing.T) {
	for _, modo := range []reglasbaremo.ModoCoincidenciaReglas{
		reglasbaremo.CoincidenciaReglasRechazar,
		reglasbaremo.CoincidenciaReglasElegirPrioridad,
		reglasbaremo.CoincidenciaReglasAcumular,
	} {
		configuracion := configuracionBaseCompilacion(t)
		configuracion.coincidencia = debeCompilacion(reglasbaremo.NuevaPoliticaCoincidenciaReglas(modo))
		plan, err := Compilar(conjuntoCompilacionPrueba(t, configuracion))
		if err != nil || plan.GruposConcurrencia()[0].CoincidenciaReglas().Modo() != modo {
			t.Fatalf("coincidencia %s no preservada: %v", modo, err)
		}
	}

	configuracion := configuracionBaseCompilacion(t)
	configuracion.coincidencia = debeCompilacion(reglasbaremo.NuevaPoliticaCoincidenciaReglas(
		reglasbaremo.CoincidenciaReglasElegirMayorPuntuacion,
	))
	_, err := Compilar(conjuntoCompilacionPrueba(t, configuracion))
	exigirErrorCompilacion(
		t, err, ErrCoincidenciaNoSoportada, CodigoCoincidenciaNoSoportada,
	)
}

func TestCompilarCierraSolapesV1(t *testing.T) {
	configuracion := configuracionBaseCompilacion(t)
	if _, err := Compilar(conjuntoCompilacionPrueba(t, configuracion)); err != nil {
		t.Fatalf("solape rechazar no admitido: %v", err)
	}

	for _, modo := range []reglasbaremo.ModoSolape{
		reglasbaremo.SolapeElegirMayorPuntuacion,
		reglasbaremo.SolapeElegirMayorDedicacion,
	} {
		configuracion := configuracionBaseCompilacion(t)
		configuracion.solape = debeCompilacion(reglasbaremo.NuevaPoliticaSolape(modo))
		_, err := Compilar(conjuntoCompilacionPrueba(t, configuracion))
		exigirErrorCompilacion(t, err, ErrSolapeNoSoportado, CodigoSolapeNoSoportado)
	}

	for _, modo := range []reglasbaremo.ModoRepartoExceso{
		reglasbaremo.RepartoExcesoRechazar,
		reglasbaremo.RepartoExcesoRecortarPorPrioridad,
		reglasbaremo.RepartoExcesoProporcionalExacto,
		reglasbaremo.RepartoExcesoElegirMayorPuntuacionMarginal,
	} {
		configuracion := configuracionBaseCompilacion(t)
		configuracion.solape = debeCompilacion(reglasbaremo.NuevaPoliticaSolapeAcumulable(baremacion.JornadaCompleta()))
		reparto := debeCompilacion(reglasbaremo.NuevaPoliticaRepartoExceso(modo))
		configuracion.reparto = &reparto
		_, err := Compilar(conjuntoCompilacionPrueba(t, configuracion))
		exigirErrorCompilacion(t, err, ErrSolapeNoSoportado, CodigoSolapeNoSoportado)
	}
}

func TestCompilarRechazaTopeUnidadesConRedondeoPorPeriodo(t *testing.T) {
	configuracion := configuracionBaseCompilacion(t)
	configuracion.redondeo = debeCompilacion(reglasbaremo.NuevaPoliticaRedondeo(
		reglasbaremo.RedondearPorPeriodo, baremacion.RedondeoMitadAlPar,
	))
	_, err := Compilar(conjuntoCompilacionPrueba(t, configuracion))
	exigirErrorCompilacion(
		t, err, ErrTopeUnidadesNoSoportado, CodigoTopeUnidadesNoSoportado,
	)

	configuracion.maximoUnidades = reglasbaremo.SinLimiteUnidades()
	if _, err := Compilar(conjuntoCompilacionPrueba(t, configuracion)); err != nil {
		t.Fatalf("redondeo por periodo sin tope temporal rechazado: %v", err)
	}
}

func TestCompilarPreservaConversionCoeficientesYTopes(t *testing.T) {
	configuracion := configuracionBaseCompilacion(t)
	configuracion.unidad = debeCompilacion(reglasbaremo.NuevaPoliticaUnidadTemporal(
		reglasbaremo.UnidadTemporalDia, reglasbaremo.UnidadTemporalAnio,
		racionalCompilacion(t, 365, 2), reglasbaremo.ExtremoFinalInclusivo,
	))
	configuracion.maximoUnidades = debeCompilacion(reglasbaremo.NuevoLimiteUnidades(racionalCompilacion(t, 25, 2)))
	configuracion.maximoPuntos = debeCompilacion(reglasbaremo.NuevoLimitePuntos(puntosCompilacion(t, 8_750_000)))
	configuracion.puntosPorUnidad = puntosCompilacion(t, 375_000)

	plan, err := Compilar(conjuntoCompilacionPrueba(t, configuracion))
	if err != nil {
		t.Fatal(err)
	}
	regla := plan.Reglas()[0]
	if regla.UnidadTemporal().UnidadesBasePorUnidad().String() != "365/2" ||
		regla.PuntosPorUnidad().Micropuntos() != 375_000 {
		t.Fatal("conversion o coeficiente alterados")
	}
	unidades, limitada := regla.MaximoUnidades().Valor()
	if !limitada || unidades.String() != "25/2" {
		t.Fatal("tope de unidades alterado")
	}
	puntos, limitado := regla.MaximoPuntos().Valor()
	if !limitado || puntos.Micropuntos() != 8_750_000 {
		t.Fatal("tope de puntos alterado")
	}

	configuracion.maximoUnidades = reglasbaremo.SinLimiteUnidades()
	configuracion.maximoPuntos = reglasbaremo.SinLimitePuntos()
	plan, err = Compilar(conjuntoCompilacionPrueba(t, configuracion))
	if err != nil {
		t.Fatalf("topes ausentes rechazados: %v", err)
	}
	regla = plan.Reglas()[0]
	if regla.MaximoUnidades().EstaLimitado() || regla.MaximoPuntos().EstaLimitado() {
		t.Fatal("la ausencia expresa de topes no se preservo")
	}
}

type configuracionCompilacionPrueba struct {
	minimoSeccion   baremacion.Puntos
	maximoSeccion   baremacion.Puntos
	unidad          reglasbaremo.PoliticaUnidadTemporal
	jornada         reglasbaremo.PoliticaJornada
	restos          reglasbaremo.PoliticaRestos
	redondeo        reglasbaremo.PoliticaRedondeo
	coincidencia    reglasbaremo.PoliticaCoincidenciaReglas
	solape          reglasbaremo.PoliticaSolape
	reparto         *reglasbaremo.PoliticaRepartoExceso
	puntosPorUnidad baremacion.Puntos
	maximoUnidades  reglasbaremo.LimiteUnidades
	maximoPuntos    reglasbaremo.LimitePuntos
}

func configuracionBaseCompilacion(t *testing.T) configuracionCompilacionPrueba {
	t.Helper()
	return configuracionCompilacionPrueba{
		minimoSeccion: puntosCompilacion(t, 0),
		maximoSeccion: puntosCompilacion(t, 100_000_000),
		unidad: debeCompilacion(reglasbaremo.NuevaPoliticaUnidadTemporal(
			reglasbaremo.UnidadTemporalDia, reglasbaremo.UnidadTemporalMes,
			racionalCompilacion(t, 30, 1), reglasbaremo.ExtremoFinalInclusivo,
		)),
		jornada: debeCompilacion(reglasbaremo.NuevaPoliticaJornada(reglasbaremo.JornadaProporcional)),
		restos:  debeCompilacion(reglasbaremo.NuevaPoliticaRestos(reglasbaremo.RestosConservarExactos)),
		redondeo: debeCompilacion(reglasbaremo.NuevaPoliticaRedondeo(
			reglasbaremo.RedondearPorRegla, baremacion.RedondeoMitadAlPar,
		)),
		coincidencia: debeCompilacion(reglasbaremo.NuevaPoliticaCoincidenciaReglas(
			reglasbaremo.CoincidenciaReglasElegirPrioridad,
		)),
		solape:          debeCompilacion(reglasbaremo.NuevaPoliticaSolape(reglasbaremo.SolapeRechazar)),
		puntosPorUnidad: puntosCompilacion(t, 500_000),
		maximoUnidades:  debeCompilacion(reglasbaremo.NuevoLimiteUnidades(racionalCompilacion(t, 120, 1))),
		maximoPuntos:    debeCompilacion(reglasbaremo.NuevoLimitePuntos(puntosCompilacion(t, 50_000_000))),
	}
}

func conjuntoCompilacionPrueba(
	t *testing.T,
	configuracion configuracionCompilacionPrueba,
) reglasbaremo.ConjuntoReglasBaremo {
	t.Helper()
	identidad := debeCompilacion(reglasbaremo.NuevaIdentidadConjuntoReglasBaremo(
		"rgl_11111111111111111111111111111111", 7,
		"con_22222222222222222222222222222222",
		"exp_33333333333333333333333333333333",
	))
	seccion := debeCompilacion(reglasbaremo.NuevaSeccionBaremo(
		"experiencia", referenciaCompilacion(t, "definicion:seccion:experiencia", 3), 1,
		configuracion.minimoSeccion, configuracion.maximoSeccion,
	))
	grupo := grupoCompilacion(t, "grupo_experiencia", 1, configuracion)
	regla := reglaCompilacion(t, "regla_experiencia", "experiencia", 1, "grupo_experiencia", configuracion)
	fecha := debeCompilacion(baremacion.NuevaFechaCivil(2026, 7, 17))
	return debeCompilacion(reglasbaremo.NuevoConjuntoReglasBaremo(
		identidad, referenciaCompilacion(t, "bases:compilacion:v1", 5), fecha,
		[]reglasbaremo.SeccionBaremo{seccion},
		[]reglasbaremo.GrupoConcurrenciaExperiencia{grupo},
		[]reglasbaremo.ReglaExperiencia{regla},
	))
}

func conjuntoOrdenadoCompilacion(t *testing.T) reglasbaremo.ConjuntoReglasBaremo {
	t.Helper()
	configuracion := configuracionBaseCompilacion(t)
	publica := debeCompilacion(reglasbaremo.NuevaSeccionBaremo(
		"experiencia_publica", referenciaCompilacion(t, "definicion:seccion:publica", 1),
		1, configuracion.minimoSeccion, configuracion.maximoSeccion,
	))
	privada := debeCompilacion(reglasbaremo.NuevaSeccionBaremo(
		"experiencia_privada", referenciaCompilacion(t, "definicion:seccion:privada", 1),
		2, configuracion.minimoSeccion, configuracion.maximoSeccion,
	))
	grupoPublico := grupoCompilacion(t, "grupo_publico", 1, configuracion)
	grupoPrivado := grupoCompilacion(t, "grupo_privado", 2, configuracion)
	reglaPublica := reglaCompilacion(
		t, "regla_publica", "experiencia_publica", 1, "grupo_publico", configuracion,
	)
	reglaPrivada := reglaCompilacion(
		t, "regla_privada", "experiencia_privada", 1, "grupo_privado", configuracion,
	)
	identidad := debeCompilacion(reglasbaremo.NuevaIdentidadConjuntoReglasBaremo(
		"rgl_44444444444444444444444444444444", 9,
		"con_55555555555555555555555555555555",
		"exp_66666666666666666666666666666666",
	))
	fecha := debeCompilacion(baremacion.NuevaFechaCivil(2026, 12, 31))
	return debeCompilacion(reglasbaremo.NuevoConjuntoReglasBaremo(
		identidad, referenciaCompilacion(t, "bases:compilacion:orden", 2), fecha,
		[]reglasbaremo.SeccionBaremo{privada, publica},
		[]reglasbaremo.GrupoConcurrenciaExperiencia{grupoPrivado, grupoPublico},
		[]reglasbaremo.ReglaExperiencia{reglaPrivada, reglaPublica},
	))
}

func grupoCompilacion(
	t *testing.T,
	clave string,
	orden uint32,
	configuracion configuracionCompilacionPrueba,
) reglasbaremo.GrupoConcurrenciaExperiencia {
	t.Helper()
	return debeCompilacion(reglasbaremo.NuevoGrupoConcurrenciaExperiencia(
		clave, referenciaCompilacion(t, "definicion:grupo:"+clave, 1), orden,
		configuracion.coincidencia, configuracion.solape, configuracion.reparto,
	))
}

func reglaCompilacion(
	t *testing.T,
	clave string,
	seccion string,
	orden uint32,
	grupo string,
	configuracion configuracionCompilacionPrueba,
) reglasbaremo.ReglaExperiencia {
	t.Helper()
	criterio := debeCompilacion(reglasbaremo.NuevoCriterioExperiencia(
		"ambito", referenciaCompilacion(t, "catalogo:ambito:compartido", 4),
		[]string{"administracion_local", "diputacion_granada"},
	))
	return debeCompilacion(reglasbaremo.NuevaReglaExperiencia(
		clave, referenciaCompilacion(t, "definicion:regla:"+clave, 2),
		seccion, orden, []reglasbaremo.CriterioExperiencia{criterio}, grupo, 1,
		configuracion.unidad, configuracion.jornada, configuracion.restos,
		configuracion.redondeo, configuracion.puntosPorUnidad,
		configuracion.maximoUnidades, configuracion.maximoPuntos,
	))
}

func conjuntoCatalogosCriterioCompilacion(
	t *testing.T,
	primero reglasbaremo.ReferenciaVersionada,
	segundo reglasbaremo.ReferenciaVersionada,
) reglasbaremo.ConjuntoReglasBaremo {
	t.Helper()
	base := conjuntoOrdenadoCompilacion(t)
	reglas := base.ReglasExperiencia()
	reglas[0] = reglaConCatalogoCompilacion(t, reglas[0], primero)
	reglas[1] = reglaConCatalogoCompilacion(t, reglas[1], segundo)
	return debeCompilacion(reglasbaremo.NuevoConjuntoReglasBaremo(
		base.Identidad(), base.Bases(), base.FechaCorte(),
		base.Secciones(), base.GruposConcurrenciaExperiencia(), reglas,
	))
}

func reglaConCatalogoCompilacion(
	t *testing.T,
	regla reglasbaremo.ReglaExperiencia,
	catalogo reglasbaremo.ReferenciaVersionada,
) reglasbaremo.ReglaExperiencia {
	t.Helper()
	criterios := regla.Criterios()
	criterio := criterios[0]
	criterios[0] = debeCompilacion(reglasbaremo.NuevoCriterioExperiencia(
		criterio.Clave(), catalogo, criterio.Valores(),
	))
	return debeCompilacion(reglasbaremo.NuevaReglaExperiencia(
		regla.Clave(), regla.Definicion(), regla.SeccionClave(), regla.Orden(),
		criterios, regla.GrupoConcurrenciaClave(), regla.PrioridadConcurrencia(),
		regla.UnidadTemporal(), regla.Jornada(), regla.Restos(), regla.Redondeo(),
		regla.PuntosPorUnidad(), regla.MaximoUnidades(), regla.MaximoPuntos(),
	))
}

func referenciaCompilacion(
	t *testing.T,
	referencia string,
	version uint64,
) reglasbaremo.ReferenciaVersionada {
	t.Helper()
	return referenciaCompilacionConSemillaHuella(t, referencia, version, referencia)
}

func referenciaCompilacionConSemillaHuella(
	t *testing.T,
	referencia string,
	version uint64,
	semilla string,
) reglasbaremo.ReferenciaVersionada {
	t.Helper()
	huella := sha256.Sum256([]byte(semilla))
	return debeCompilacion(reglasbaremo.NuevaReferenciaVersionada(
		referencia, version, hex.EncodeToString(huella[:]),
	))
}

func racionalCompilacion(t *testing.T, numerador, denominador int64) baremacion.Racional {
	t.Helper()
	return debeCompilacion(baremacion.NuevoRacional(numerador, denominador))
}

func puntosCompilacion(t *testing.T, micropuntos int64) baremacion.Puntos {
	t.Helper()
	return debeCompilacion(baremacion.PuntosDesdeMicropuntos(micropuntos))
}

func debeCompilacion[T any](valor T, err error) T {
	if err != nil {
		panic(err)
	}
	return valor
}

func exigirErrorCompilacion(
	t *testing.T,
	err error,
	esperado error,
	codigo CodigoError,
) {
	t.Helper()
	if !errors.Is(err, esperado) {
		t.Fatalf("error=%v; se esperaba %v", err, esperado)
	}
	var tipado *ErrorCalculo
	if !errors.As(err, &tipado) || tipado.Codigo() != codigo {
		t.Fatalf("error sin codigo %s: %v", codigo, err)
	}
}

func exigirPlanInvalido(t *testing.T, err error) {
	t.Helper()
	exigirErrorCompilacion(
		t, err, ErrCompilacionPlanInvalido, CodigoCompilacionPlanInvalido,
	)
}

func clavesSecciones(secciones []reglasbaremo.SeccionBaremo) string {
	resultado := ""
	for indice, seccion := range secciones {
		if indice > 0 {
			resultado += ","
		}
		resultado += seccion.Clave()
	}
	return resultado
}

func clavesGrupos(grupos []reglasbaremo.GrupoConcurrenciaExperiencia) string {
	resultado := ""
	for indice, grupo := range grupos {
		if indice > 0 {
			resultado += ","
		}
		resultado += grupo.Clave()
	}
	return resultado
}

func clavesReglas(reglas []reglasbaremo.ReglaExperiencia) string {
	resultado := ""
	for indice, regla := range reglas {
		if indice > 0 {
			resultado += ","
		}
		resultado += regla.Clave()
	}
	return resultado
}
