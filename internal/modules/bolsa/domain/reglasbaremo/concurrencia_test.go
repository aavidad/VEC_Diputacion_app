package reglasbaremo

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestGrupoDistingueCoincidenciaDeReglasYSolapeTemporal(t *testing.T) {
	for _, modo := range []ModoCoincidenciaReglas{
		CoincidenciaReglasRechazar,
		CoincidenciaReglasElegirPrioridad,
		CoincidenciaReglasElegirMayorPuntuacion,
		CoincidenciaReglasAcumular,
	} {
		if _, err := NuevaPoliticaCoincidenciaReglas(modo); err != nil {
			t.Fatalf("coincidencia %q rechazada: %v", modo, err)
		}
	}
	if _, err := NuevaPoliticaCoincidenciaReglas(ModoCoincidenciaReglas("inventado")); !errors.Is(err, ErrPoliticaIncompleta) {
		t.Fatalf("coincidencia abierta aceptada: %v", err)
	}

	coincidencia, _ := NuevaPoliticaCoincidenciaReglas(CoincidenciaReglasElegirPrioridad)
	solape, _ := NuevaPoliticaSolape(SolapeRechazar)
	grupo, err := NuevoGrupoConcurrenciaExperiencia(
		"grupo_prioridad",
		referenciaPrueba(t, "grupo-concurrencia:prioridad", 1, 'e'),
		1,
		coincidencia,
		solape,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if grupo.CoincidenciaReglas().Modo() != CoincidenciaReglasElegirPrioridad ||
		grupo.Solape().Modo() != SolapeRechazar {
		t.Fatal("el grupo confundio las dos politicas")
	}
}

func TestGrupoAcumulableExigeRepartoExcesoExplicito(t *testing.T) {
	coincidencia, _ := NuevaPoliticaCoincidenciaReglas(CoincidenciaReglasAcumular)
	limite := baremacion.JornadaCompleta()
	solape, err := NuevaPoliticaSolapeAcumulable(limite)
	if err != nil {
		t.Fatal(err)
	}
	definicion := referenciaPrueba(t, "grupo-concurrencia:j05", 1, 'f')
	if _, err := NuevoGrupoConcurrenciaExperiencia(
		"j05", definicion, 1, coincidencia, solape, nil,
	); !errors.Is(err, ErrPoliticaIncompleta) {
		t.Fatalf("acumulacion sin reparto de exceso aceptada: %v", err)
	}

	reparto, err := NuevaPoliticaRepartoExceso(RepartoExcesoRecortarPorPrioridad)
	if err != nil {
		t.Fatal(err)
	}
	grupo, err := NuevoGrupoConcurrenciaExperiencia(
		"j05", definicion, 1, coincidencia, solape, &reparto,
	)
	if err != nil {
		t.Fatalf("J-05 completo rechazado: %v", err)
	}
	recuperado, existe := grupo.RepartoExceso()
	if !existe || recuperado.Modo() != RepartoExcesoRecortarPorPrioridad ||
		recuperado.DesempateEntreReglas() != DesempateExcesoPrioridadConcurrencia ||
		recuperado.RepartoDentroMismaRegla() != RepartoDentroReglaProporcionalExacto {
		t.Fatalf("semantica de reparto incompleta: %#v", recuperado)
	}

	// El constructor copia el puntero: una modificacion posterior no cambia el grupo.
	reparto.modo = RepartoExcesoRechazar
	recuperado, _ = grupo.RepartoExceso()
	if recuperado.Modo() != RepartoExcesoRecortarPorPrioridad {
		t.Fatal("el grupo retuvo el puntero de entrada")
	}

	noAcumulable, _ := NuevaPoliticaSolape(SolapeElegirMayorPuntuacion)
	repartoValido, _ := NuevaPoliticaRepartoExceso(RepartoExcesoRechazar)
	if _, err := NuevoGrupoConcurrenciaExperiencia(
		"incoherente", definicion, 1, coincidencia, noAcumulable, &repartoValido,
	); !errors.Is(err, ErrPoliticaIncompleta) {
		t.Fatalf("reparto de exceso fuera de acumulacion aceptado: %v", err)
	}
}

func TestCanonJ05ExplicitaRepartoSinOrdenDeEntrada(t *testing.T) {
	conjunto := conjuntoAcumulablePrueba(t)
	canonico, err := conjunto.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	fragmentos := []string{
		`"coincidencia_reglas":{"modo":"acumular"}`,
		`"solape":{"modo":"acumular_hasta_limite","limite":"1/1"}`,
		`"reparto_exceso":{"modo":"recortar_por_prioridad",`,
		`"desempate_entre_reglas":"prioridad_concurrencia"`,
		`"reparto_dentro_misma_regla":"proporcional_exacto"`,
	}
	for _, fragmento := range fragmentos {
		if !bytes.Contains(canonico, []byte(fragmento)) {
			t.Fatalf("el canon J-05 no contiene %s: %s", fragmento, canonico)
		}
	}
	restaurado, err := RestaurarConjuntoReglasBaremo(canonico)
	if err != nil {
		t.Fatal(err)
	}
	reproducido, _ := restaurado.RepresentacionCanonica()
	if !bytes.Equal(canonico, reproducido) {
		t.Fatal("J-05 no se restaura byte a byte")
	}
}

func TestConjuntoGobiernaGruposPrioridadesYUso(t *testing.T) {
	identidad, bases, fecha, secciones, grupos, reglas := componentesPrueba(t)
	if reglas[0].seccionClave == reglas[1].seccionClave ||
		reglas[0].grupoConcurrenciaClave != reglas[1].grupoConcurrenciaClave {
		t.Fatal("la muestra no acredita un grupo entre secciones")
	}
	if _, err := NuevoConjuntoReglasBaremo(
		identidad, bases, fecha, secciones, grupos, reglas,
	); err != nil {
		t.Fatalf("grupo entre secciones rechazado: %v", err)
	}

	desconocida := reglas[0].clonar()
	desconocida.grupoConcurrenciaClave = "grupo_inexistente"
	if _, err := NuevoConjuntoReglasBaremo(
		identidad, bases, fecha, secciones, grupos,
		[]ReglaExperiencia{desconocida, reglas[1]},
	); !errors.Is(err, ErrGrupoDesconocido) {
		t.Fatalf("grupo desconocido aceptado: %v", err)
	}

	empatada := reglas[1].clonar()
	empatada.prioridadConcurrencia = reglas[0].prioridadConcurrencia
	if _, err := NuevoConjuntoReglasBaremo(
		identidad, bases, fecha, secciones, grupos,
		[]ReglaExperiencia{reglas[0], empatada},
	); !errors.Is(err, ErrValorDuplicado) {
		t.Fatalf("prioridad empatada dentro del grupo aceptada: %v", err)
	}

	grupoSinUso := grupoNoAcumulablePrueba(t, "sin_uso", 20, 'a')
	if _, err := NuevoConjuntoReglasBaremo(
		identidad, bases, fecha, secciones,
		append(grupos, grupoSinUso), reglas,
	); !errors.Is(err, ErrPoliticaIncompleta) {
		t.Fatalf("grupo sin reglas aceptado: %v", err)
	}

	if _, err := NuevoConjuntoReglasBaremo(
		identidad, bases, fecha, secciones, nil, reglas,
	); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("conjunto sin grupos aceptado: %v", err)
	}

	grupoDuplicado := grupoSinUso
	grupoDuplicado.clave = grupos[0].clave
	if _, err := NuevoConjuntoReglasBaremo(
		identidad, bases, fecha, secciones,
		[]GrupoConcurrenciaExperiencia{grupos[0], grupoDuplicado}, reglas,
	); !errors.Is(err, ErrValorDuplicado) {
		t.Fatalf("clave de grupo duplicada aceptada: %v", err)
	}

	demasiados := make([]GrupoConcurrenciaExperiencia, maximoGruposConcurrencia+1)
	if _, err := NuevoConjuntoReglasBaremo(
		identidad, bases, fecha, secciones, demasiados, reglas,
	); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("volumen de grupos no acotado: %v", err)
	}
}

func TestConjuntoOrdenaGruposYPermitePrioridadUnoEnGruposDistintos(t *testing.T) {
	identidad, bases, fecha, secciones, grupos, reglas := componentesPrueba(t)
	segundo := grupoNoAcumulablePrueba(t, "experiencia_privada_independiente", 20, 'b')
	reglas[1].grupoConcurrenciaClave = segundo.clave
	reglas[1].prioridadConcurrencia = 1
	conjunto, err := NuevoConjuntoReglasBaremo(
		identidad, bases, fecha, secciones,
		[]GrupoConcurrenciaExperiencia{segundo, grupos[0]}, reglas,
	)
	if err != nil {
		t.Fatal(err)
	}
	ordenados := conjunto.GruposConcurrenciaExperiencia()
	if len(ordenados) != 2 || ordenados[0].Orden() != 10 || ordenados[1].Orden() != 20 {
		t.Fatalf("orden de grupos no canonico: %#v", ordenados)
	}
	ordenados[0] = GrupoConcurrenciaExperiencia{}
	if _, existe := conjunto.GrupoConcurrenciaPorClave("experiencia_mas_favorable"); !existe {
		t.Fatal("la copia devuelta modifico el agregado")
	}
}

func TestRestaurarRechazaConcurrenciaIncoherente(t *testing.T) {
	canonico, _ := conjuntoPrueba(t, true).RepresentacionCanonica()
	casos := []struct {
		nombre string
		buscar string
		poner  string
	}{
		{
			"coincidencia_desconocida",
			`"coincidencia_reglas":{"modo":"elegir_mayor_puntuacion"}`,
			`"coincidencia_reglas":{"modo":"inventado"}`,
		},
		{
			"acumulacion_sin_reparto",
			`"solape":{"modo":"elegir_mayor_puntuacion"}`,
			`"solape":{"modo":"acumular_hasta_limite","limite":"1/1"}`,
		},
		{
			"reparto_fuera_de_acumulacion",
			`"solape":{"modo":"elegir_mayor_puntuacion"}`,
			`"solape":{"modo":"elegir_mayor_puntuacion"},"reparto_exceso":{"modo":"rechazar","desempate_entre_reglas":"no_aplica","reparto_dentro_misma_regla":"no_aplica"}`,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			alterado := sustituirUnaVez(t, canonico, caso.buscar, caso.poner)
			_, err := RestaurarConjuntoReglasBaremo(alterado)
			if !errors.Is(err, ErrPoliticaIncompleta) {
				t.Fatalf("concurrencia incoherente aceptada: %v", err)
			}
		})
	}

	j05, _ := conjuntoAcumulablePrueba(t).RepresentacionCanonica()
	alterado := sustituirUnaVez(
		t,
		j05,
		`"desempate_entre_reglas":"prioridad_concurrencia"`,
		`"desempate_entre_reglas":"no_aplica"`,
	)
	if _, err := RestaurarConjuntoReglasBaremo(alterado); !errors.Is(err, ErrPoliticaIncompleta) {
		t.Fatalf("semantica de reparto manipulada aceptada: %v", err)
	}
}

func conjuntoAcumulablePrueba(t *testing.T) ConjuntoReglasBaremo {
	t.Helper()
	identidad, bases, fecha, secciones, _, reglas := componentesPrueba(t)
	for indice := range reglas {
		reglas[indice].grupoConcurrenciaClave = "j05"
	}
	coincidencia, _ := NuevaPoliticaCoincidenciaReglas(CoincidenciaReglasAcumular)
	solape, _ := NuevaPoliticaSolapeAcumulable(baremacion.JornadaCompleta())
	reparto, _ := NuevaPoliticaRepartoExceso(RepartoExcesoRecortarPorPrioridad)
	grupo, err := NuevoGrupoConcurrenciaExperiencia(
		"j05", referenciaPrueba(t, "grupo-concurrencia:j05-completo", 1, '9'), 1,
		coincidencia, solape, &reparto,
	)
	if err != nil {
		t.Fatal(err)
	}
	conjunto, err := NuevoConjuntoReglasBaremo(
		identidad, bases, fecha, secciones, []GrupoConcurrenciaExperiencia{grupo}, reglas,
	)
	if err != nil {
		t.Fatal(err)
	}
	return conjunto
}

func grupoNoAcumulablePrueba(
	t *testing.T,
	clave string,
	orden uint32,
	marca byte,
) GrupoConcurrenciaExperiencia {
	t.Helper()
	coincidencia, _ := NuevaPoliticaCoincidenciaReglas(CoincidenciaReglasElegirMayorPuntuacion)
	solape, _ := NuevaPoliticaSolape(SolapeElegirMayorPuntuacion)
	grupo, err := NuevoGrupoConcurrenciaExperiencia(
		clave, referenciaPrueba(t, "grupo-concurrencia:"+clave, 1, marca), orden,
		coincidencia, solape, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return grupo
}

func TestPrioridadConcurrenciaEsPositivaYAcotada(t *testing.T) {
	componentes := componentesReglaPrueba(t, "prioridad", "experiencia_publica", 1, '7')
	for _, prioridad := range []uint32{0, maximoOrden + 1} {
		componentes.prioridadConcurrencia = prioridad
		_, err := NuevaReglaExperiencia(
			componentes.clave, componentes.definicion, componentes.seccionClave, componentes.orden,
			componentes.criterios, componentes.grupoConcurrenciaClave, componentes.prioridadConcurrencia,
			componentes.temporal, componentes.jornada, componentes.restos, componentes.redondeo,
			componentes.puntos, componentes.maximoUnidades, componentes.maximoPuntos,
		)
		if !errors.Is(err, ErrFueraDeLimites) {
			t.Fatalf("prioridad %d aceptada: %v", prioridad, err)
		}
	}
}

func TestTodosLosModosRepartoTienenSemanticaCanonizable(t *testing.T) {
	casos := []struct {
		modo      ModoRepartoExceso
		desempate CriterioDesempateExceso
		dentro    ModoRepartoDentroRegla
	}{
		{RepartoExcesoRechazar, DesempateExcesoNoAplica, RepartoDentroReglaNoAplica},
		{RepartoExcesoRecortarPorPrioridad, DesempateExcesoPrioridadConcurrencia, RepartoDentroReglaProporcionalExacto},
		{RepartoExcesoProporcionalExacto, DesempateExcesoNoAplica, RepartoDentroReglaProporcionalExacto},
		{RepartoExcesoElegirMayorPuntuacionMarginal, DesempateExcesoPrioridadConcurrencia, RepartoDentroReglaProporcionalExacto},
	}
	for _, caso := range casos {
		politica, err := NuevaPoliticaRepartoExceso(caso.modo)
		if err != nil || politica.DesempateEntreReglas() != caso.desempate ||
			politica.RepartoDentroMismaRegla() != caso.dentro {
			t.Errorf("modo %q incompleto: %#v, %v", caso.modo, politica, err)
		}
	}
	if _, err := NuevaPoliticaRepartoExceso(ModoRepartoExceso("inventado")); !errors.Is(err, ErrPoliticaIncompleta) {
		t.Fatalf("reparto abierto aceptado: %v", err)
	}
}

func TestConcurrenciaNoContieneConvencionesTextualesDinamicas(t *testing.T) {
	canonico, _ := conjuntoAcumulablePrueba(t).RepresentacionCanonica()
	for _, prohibido := range []string{"formula", "script", "consulta_sql", "orden_entrada"} {
		if strings.Contains(string(canonico), prohibido) {
			t.Fatalf("canon contiene una convencion dinamica: %s", prohibido)
		}
	}
}
