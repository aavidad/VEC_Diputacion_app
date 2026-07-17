package reglasbaremo

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/shared/baremacion"
)

const (
	referenciaReglasConjuntoPrueba       = "rgl_11111111111111111111111111111111"
	referenciaConvocatoriaConjuntoPrueba = "con_22222222222222222222222222222222"
	referenciaExpedienteConjuntoPrueba   = "exp_33333333333333333333333333333333"
)

func TestConjuntoOrdenaYGeneraHuellaCanonicaDeterminista(t *testing.T) {
	primero := conjuntoPrueba(t, true)
	segundo := conjuntoPrueba(t, false)

	secciones := primero.Secciones()
	if len(secciones) != 2 || secciones[0].Clave() != "experiencia_publica" ||
		secciones[1].Clave() != "experiencia_privada" {
		t.Fatalf("orden de secciones no canonico: %#v", secciones)
	}
	reglas := primero.ReglasExperiencia()
	if len(reglas) != 2 || reglas[0].Clave() != "servicios_diputacion" ||
		reglas[1].Clave() != "servicios_privados" {
		t.Fatalf("orden de reglas no canonico: %#v", reglas)
	}

	bytesPrimero, errPrimero := primero.RepresentacionCanonica()
	bytesSegundo, errSegundo := segundo.RepresentacionCanonica()
	if errPrimero != nil || errSegundo != nil {
		t.Fatalf("representacion canonica: %v / %v", errPrimero, errSegundo)
	}
	if !bytes.Equal(bytesPrimero, bytesSegundo) {
		t.Fatalf("el orden de entrada cambio los bytes:\n%s\n%s", bytesPrimero, bytesSegundo)
	}
	huella, err := primero.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(bytesPrimero)
	if huella != hex.EncodeToString(suma[:]) {
		t.Fatalf("huella no corresponde a los bytes: %s", huella)
	}
	if huella != "e4a88c2a8ec116af75de99e2453881f370f677e6d3bb12aed6bd545f620c4b98" {
		t.Fatalf("actualizar vector golden: %s\n%s", huella, bytesPrimero)
	}

	serializado, err := json.Marshal(primero)
	if err != nil || !bytes.Equal(serializado, bytesPrimero) {
		t.Fatalf("MarshalJSON creo otro contrato: %v", err)
	}
	referencia, err := primero.ReferenciaVersionada()
	if err != nil || referencia.Referencia() != primero.Identidad().Referencia() ||
		referencia.Version() != primero.Identidad().Version() || referencia.HuellaSHA256() != huella {
		t.Fatalf("referencia versionada incoherente: %#v, %v", referencia, err)
	}
}

func TestConjuntoYCriteriosHacenCopiasDefensivas(t *testing.T) {
	valoresEntrada := []string{"diputacion_granada", "administracion_local"}
	criterioAislado, err := NuevoCriterioExperiencia(
		"empleador", referenciaPrueba(t, "catalogo:copia-defensiva", 1, 'd'), valoresEntrada,
	)
	if err != nil {
		t.Fatal(err)
	}
	valoresEntrada[0] = "manipulado"
	if criterioAislado.Valores()[0] != "administracion_local" {
		t.Fatal("el criterio conserva el slice recibido")
	}

	identidad, bases, fecha, secciones, grupos, reglas := componentesPrueba(t)
	conjunto, err := NuevoConjuntoReglasBaremo(identidad, bases, fecha, secciones, grupos, reglas)
	if err != nil {
		t.Fatal(err)
	}
	antes, _ := conjunto.RepresentacionCanonica()

	secciones[0] = SeccionBaremo{}
	grupos[0] = GrupoConcurrenciaExperiencia{}
	reglas[0] = ReglaExperiencia{}
	devueltas := conjunto.ReglasExperiencia()
	devueltas[0].criterios[0].valores[0] = "manipulado"
	devueltas[0] = ReglaExperiencia{}
	seccionesDevueltas := conjunto.Secciones()
	seccionesDevueltas[0] = SeccionBaremo{}
	reglaBuscada, existe := conjunto.ReglaExperienciaPorClave("servicios_diputacion")
	if !existe {
		t.Fatal("no se encontro la regla")
	}
	reglaBuscada.criterios[0].valores[0] = "manipulado"
	valores := reglaBuscada.Criterios()[0].Valores()
	valores[0] = "manipulado_de_nuevo"

	despues, err := conjunto.RepresentacionCanonica()
	if err != nil || !bytes.Equal(antes, despues) {
		t.Fatalf("una copia externa modifico el agregado: %v", err)
	}
}

func TestReglaExigeCoeficienteYTodasLasPoliticas(t *testing.T) {
	componentes := componentesReglaPrueba(t, "regla_uno", "experiencia_publica", 1, 'a')
	cero, err := baremacion.PuntosDesdeMicropuntos(0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NuevaReglaExperiencia(
		componentes.clave, componentes.definicion, componentes.seccionClave, componentes.orden,
		componentes.criterios, componentes.grupoConcurrenciaClave, componentes.prioridadConcurrencia,
		componentes.temporal, componentes.jornada,
		componentes.restos, componentes.redondeo, cero, componentes.maximoUnidades,
		componentes.maximoPuntos,
	)
	if !errors.Is(err, ErrCoeficienteAusente) {
		t.Fatalf("coeficiente cero aceptado o error incorrecto: %v", err)
	}

	puntos := puntosPrueba(t, 100_000)
	_, err = NuevaReglaExperiencia(
		componentes.clave, componentes.definicion, componentes.seccionClave, componentes.orden,
		componentes.criterios, componentes.grupoConcurrenciaClave, componentes.prioridadConcurrencia,
		PoliticaUnidadTemporal{}, componentes.jornada,
		componentes.restos, componentes.redondeo, puntos, componentes.maximoUnidades,
		componentes.maximoPuntos,
	)
	if !errors.Is(err, ErrPoliticaIncompleta) {
		t.Fatalf("politica temporal vacia aceptada: %v", err)
	}

	_, err = NuevaReglaExperiencia(
		componentes.clave, componentes.definicion, componentes.seccionClave, componentes.orden,
		componentes.criterios, componentes.grupoConcurrenciaClave, componentes.prioridadConcurrencia,
		componentes.temporal, PoliticaJornada{},
		componentes.restos, componentes.redondeo, puntos, componentes.maximoUnidades,
		componentes.maximoPuntos,
	)
	if !errors.Is(err, ErrPoliticaIncompleta) {
		t.Fatalf("politica de jornada vacia aceptada: %v", err)
	}
}

func TestPoliticasRechazanCombinacionesAmbiguas(t *testing.T) {
	uno := racionalPrueba(t, 1, 1)
	dos := racionalPrueba(t, 2, 1)
	if _, err := NuevaPoliticaUnidadTemporal(
		UnidadTemporalDia, UnidadTemporalDia, dos, ExtremoFinalExclusivo,
	); !errors.Is(err, ErrValorInvalido) {
		t.Fatalf("conversion no identitaria para la misma unidad aceptada: %v", err)
	}
	if _, err := NuevaPoliticaUnidadTemporal(
		UnidadTemporalDia, UnidadTemporalDia, uno, TratamientoExtremoFinal("inventado"),
	); !errors.Is(err, ErrPoliticaIncompleta) {
		t.Fatalf("extremo desconocido aceptado: %v", err)
	}
	if _, err := NuevaPoliticaJornada(JornadaIntegraDesdeUmbral); !errors.Is(err, ErrPoliticaIncompleta) {
		t.Fatalf("umbral implicito aceptado: %v", err)
	}
	if _, err := NuevaPoliticaJornadaDesdeUmbral(baremacion.FraccionJornada{}); !errors.Is(err, ErrPoliticaIncompleta) {
		t.Fatalf("umbral vacio aceptado: %v", err)
	}
	if _, err := NuevaPoliticaSolape(SolapeAcumularHastaLimite); !errors.Is(err, ErrPoliticaIncompleta) {
		t.Fatalf("acumulacion sin limite aceptada: %v", err)
	}
	if _, err := NuevaPoliticaSolapeAcumulable(baremacion.FraccionJornada{}); !errors.Is(err, ErrPoliticaIncompleta) {
		t.Fatalf("limite vacio aceptado: %v", err)
	}
	if _, err := NuevaPoliticaRestos(ModoRestos("inventado")); !errors.Is(err, ErrPoliticaIncompleta) {
		t.Fatalf("restos desconocidos aceptados: %v", err)
	}
	if _, err := NuevaPoliticaRedondeo(RedondearPorRegla, baremacion.ModoRedondeo("inventado")); !errors.Is(err, ErrPoliticaIncompleta) {
		t.Fatalf("redondeo desconocido aceptado: %v", err)
	}
}

func TestJornadaPorHorasExigeHorasComoUnidadBase(t *testing.T) {
	componentes := componentesReglaPrueba(t, "regla_horas", "experiencia_publica", 1, 'a')
	jornada, err := NuevaPoliticaJornada(JornadaPorHoras)
	if err != nil {
		t.Fatal(err)
	}
	componentes.jornada = jornada
	if _, err := NuevaReglaExperiencia(
		componentes.clave, componentes.definicion, componentes.seccionClave, componentes.orden,
		componentes.criterios, componentes.grupoConcurrenciaClave, componentes.prioridadConcurrencia,
		componentes.temporal, componentes.jornada,
		componentes.restos, componentes.redondeo, componentes.puntos, componentes.maximoUnidades,
		componentes.maximoPuntos,
	); !errors.Is(err, ErrPoliticaIncompleta) {
		t.Fatalf("jornada por horas con unidad diaria aceptada: %v", err)
	}

	componentes.temporal, err = NuevaPoliticaUnidadTemporal(
		UnidadTemporalHora, UnidadTemporalDia, racionalPrueba(t, 8, 1), ExtremoFinalExclusivo,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NuevaReglaExperiencia(
		componentes.clave, componentes.definicion, componentes.seccionClave, componentes.orden,
		componentes.criterios, componentes.grupoConcurrenciaClave, componentes.prioridadConcurrencia,
		componentes.temporal, componentes.jornada,
		componentes.restos, componentes.redondeo, componentes.puntos, componentes.maximoUnidades,
		componentes.maximoPuntos,
	); err != nil {
		t.Fatalf("jornada por horas completa rechazada: %v", err)
	}
	if _, err := NuevaPoliticaJornada(ModoJornada("variable")); !errors.Is(err, ErrPoliticaIncompleta) {
		t.Fatalf("jornada variable sin fuente versionada aceptada: %v", err)
	}
}

func TestConjuntoRechazaDuplicadosReferenciasYSeccionesDesconocidas(t *testing.T) {
	identidad, bases, fecha, secciones, grupos, reglas := componentesPrueba(t)

	duplicada := secciones[1]
	duplicada.clave = secciones[0].clave
	if _, err := NuevoConjuntoReglasBaremo(
		identidad, bases, fecha, []SeccionBaremo{secciones[0], duplicada}, grupos, reglas,
	); !errors.Is(err, ErrValorDuplicado) {
		t.Fatalf("clave de seccion duplicada aceptada: %v", err)
	}

	duplicada = secciones[1]
	duplicada.definicion = secciones[0].definicion
	if _, err := NuevoConjuntoReglasBaremo(
		identidad, bases, fecha, []SeccionBaremo{secciones[0], duplicada}, grupos, reglas,
	); !errors.Is(err, ErrValorDuplicado) {
		t.Fatalf("referencia de definicion duplicada aceptada: %v", err)
	}

	desconocida := reglas[0].clonar()
	desconocida.seccionClave = "inexistente"
	if _, err := NuevoConjuntoReglasBaremo(
		identidad, bases, fecha, secciones, grupos, []ReglaExperiencia{desconocida, reglas[1]},
	); !errors.Is(err, ErrSeccionDesconocida) {
		t.Fatalf("seccion desconocida aceptada: %v", err)
	}

	repetida := reglas[1].clonar()
	repetida.seccionClave = reglas[0].seccionClave
	repetida.orden = reglas[0].orden
	if _, err := NuevoConjuntoReglasBaremo(
		identidad, bases, fecha, secciones, grupos, []ReglaExperiencia{reglas[0], repetida},
	); !errors.Is(err, ErrValorDuplicado) {
		t.Fatalf("orden de regla duplicado en seccion aceptado: %v", err)
	}
}

func TestConjuntoRechazaLimitesYSeccionesSinRegla(t *testing.T) {
	identidad, bases, fecha, secciones, grupos, reglas := componentesPrueba(t)
	regla := reglas[0].clonar()
	regla.maximoPuntos = limitePuntosPrueba(t, secciones[0].puntosMaximos.Micropuntos()+1)
	if _, err := NuevoConjuntoReglasBaremo(
		identidad, bases, fecha, secciones, grupos, []ReglaExperiencia{regla, reglas[1]},
	); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("tope de regla superior a la seccion aceptado: %v", err)
	}

	if _, err := NuevoConjuntoReglasBaremo(
		identidad, bases, fecha, secciones, grupos, []ReglaExperiencia{reglas[0]},
	); !errors.Is(err, ErrPoliticaIncompleta) {
		t.Fatalf("seccion sin regla aceptada: %v", err)
	}

	demasiadas := make([]SeccionBaremo, maximoSeccionesPorConjunto+1)
	copy(demasiadas, secciones)
	for indice := len(secciones); indice < len(demasiadas); indice++ {
		definicion := referenciaPrueba(t, "seccion:extra:"+numeroPrueba(indice), 1, byte('a'+indice%6))
		demasiadas[indice], _ = NuevaSeccionBaremo(
			"extra_"+numeroPrueba(indice), definicion, uint32(indice+1),
			puntosPrueba(t, 0), puntosPrueba(t, 1_000_000),
		)
	}
	if _, err := NuevoConjuntoReglasBaremo(
		identidad, bases, fecha, demasiadas, grupos, reglas,
	); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("volumen de secciones no acotado: %v", err)
	}
}

func TestConjuntoRechazaFechaCorteSinSiguienteCivil(t *testing.T) {
	identidad, bases, _, secciones, grupos, reglas := componentesPrueba(t)
	fechaMaxima, err := baremacion.NuevaFechaCivil(9999, 12, 31)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NuevoConjuntoReglasBaremo(
		identidad, bases, fechaMaxima, secciones, grupos, reglas,
	); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("fecha de corte sin extremo semiabierto aceptada: %v", err)
	}
	canonico, _ := conjuntoPrueba(t, true).RepresentacionCanonica()
	alterado := sustituirUnaVez(
		t, canonico,
		`"fecha_corte_inclusiva":"2026-07-01"`,
		`"fecha_corte_inclusiva":"9999-12-31"`,
	)
	if _, err := RestaurarConjuntoReglasBaremo(alterado); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("restauracion acepto fecha sin extremo semiabierto: %v", err)
	}
}

func TestValoresRechazanTextoNoCanonicoYHuellasInvalidas(t *testing.T) {
	huella := strings.Repeat("a", sha256.Size*2)
	casos := []struct {
		nombre     string
		referencia string
		version    uint64
		huella     string
	}{
		{"espacios", " documento:bases ", 1, huella},
		{"referencia_no_opaca", "documento bases", 1, huella},
		{"sin_version", "documento:bases", 0, huella},
		{"version_excesiva", "documento:bases", maximoVersion + 1, huella},
		{"huella_mayuscula", "documento:bases", 1, strings.ToUpper(huella)},
		{"huella_corta", "documento:bases", 1, huella[:63]},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := NuevaReferenciaVersionada(caso.referencia, caso.version, caso.huella); err == nil {
				t.Fatal("valor no canonico aceptado")
			}
		})
	}

	definicion := referenciaPrueba(t, "seccion:experiencia", 1, 'b')
	if _, err := NuevaSeccionBaremo(
		" Experiencia", definicion, 1, puntosPrueba(t, 0), puntosPrueba(t, 1_000_000),
	); !errors.Is(err, ErrValorNoCanonico) {
		t.Fatalf("clave no canonica aceptada: %v", err)
	}
}

func TestIdentidadConjuntoSoloAdmiteTokensOpacos128(t *testing.T) {
	if _, err := NuevaIdentidadConjuntoReglasBaremo(
		referenciaReglasConjuntoPrueba, 1,
		referenciaConvocatoriaConjuntoPrueba,
		referenciaExpedienteConjuntoPrueba,
	); err != nil {
		t.Fatalf("identidad opaca valida rechazada: %v", err)
	}
	casos := []struct {
		nombre       string
		referencia   string
		convocatoria string
		expediente   string
	}{
		{"DNI como reglas", "dni:12345678Z", referenciaConvocatoriaConjuntoPrueba, referenciaExpedienteConjuntoPrueba},
		{"codigo convocatoria", referenciaReglasConjuntoPrueba, "convocatoria:2026-001", referenciaExpedienteConjuntoPrueba},
		{"expediente legible", referenciaReglasConjuntoPrueba, referenciaConvocatoriaConjuntoPrueba, "EXP-2026-001"},
		{"prefijos cruzados", referenciaReglasConjuntoPrueba, referenciaExpedienteConjuntoPrueba, referenciaConvocatoriaConjuntoPrueba},
		{"mayusculas", "rgl_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", referenciaConvocatoriaConjuntoPrueba, referenciaExpedienteConjuntoPrueba},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := NuevaIdentidadConjuntoReglasBaremo(
				caso.referencia, 1, caso.convocatoria, caso.expediente,
			); err == nil {
				t.Fatal("identidad legible o no canonica aceptada")
			}
		})
	}
}

func conjuntoPrueba(t *testing.T, invertido bool) ConjuntoReglasBaremo {
	t.Helper()
	identidad, bases, fecha, secciones, grupos, reglas := componentesPrueba(t)
	if !invertido {
		invertir(secciones)
		invertir(grupos)
		invertir(reglas)
	}
	conjunto, err := NuevoConjuntoReglasBaremo(identidad, bases, fecha, secciones, grupos, reglas)
	if err != nil {
		t.Fatal(err)
	}
	return conjunto
}

func componentesPrueba(t *testing.T) (
	IdentidadConjuntoReglasBaremo,
	ReferenciaVersionada,
	baremacion.FechaCivil,
	[]SeccionBaremo,
	[]GrupoConcurrenciaExperiencia,
	[]ReglaExperiencia,
) {
	t.Helper()
	identidad, err := NuevaIdentidadConjuntoReglasBaremo(
		referenciaReglasConjuntoPrueba, 1,
		referenciaConvocatoriaConjuntoPrueba,
		referenciaExpedienteConjuntoPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	bases := referenciaPrueba(t, "documento:bases:2026-001", 3, '1')
	fecha, err := baremacion.NuevaFechaCivil(2026, 7, 1)
	if err != nil {
		t.Fatal(err)
	}
	publica, err := NuevaSeccionBaremo(
		"experiencia_publica", referenciaPrueba(t, "seccion:experiencia-publica", 1, '2'), 10,
		puntosPrueba(t, 0), puntosPrueba(t, 20_000_000),
	)
	if err != nil {
		t.Fatal(err)
	}
	privada, err := NuevaSeccionBaremo(
		"experiencia_privada", referenciaPrueba(t, "seccion:experiencia-privada", 1, '3'), 20,
		puntosPrueba(t, 0), puntosPrueba(t, 10_000_000),
	)
	if err != nil {
		t.Fatal(err)
	}
	componentesPublica := componentesReglaPrueba(t, "servicios_diputacion", publica.Clave(), 10, '4')
	componentesPrivada := componentesReglaPrueba(t, "servicios_privados", privada.Clave(), 10, '8')
	componentesPublica.prioridadConcurrencia = 1
	componentesPrivada.prioridadConcurrencia = 2
	reglaPublica := crearReglaPrueba(t, componentesPublica)
	reglaPrivada := crearReglaPrueba(t, componentesPrivada)
	coincidencia, err := NuevaPoliticaCoincidenciaReglas(CoincidenciaReglasElegirMayorPuntuacion)
	if err != nil {
		t.Fatal(err)
	}
	solape, err := NuevaPoliticaSolape(SolapeElegirMayorPuntuacion)
	if err != nil {
		t.Fatal(err)
	}
	grupo, err := NuevoGrupoConcurrenciaExperiencia(
		"experiencia_mas_favorable",
		referenciaPrueba(t, "grupo-concurrencia:experiencia-mas-favorable", 1, 'c'),
		10,
		coincidencia,
		solape,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return identidad, bases, fecha,
		[]SeccionBaremo{privada, publica},
		[]GrupoConcurrenciaExperiencia{grupo},
		[]ReglaExperiencia{reglaPrivada, reglaPublica}
}

type componentesRegla struct {
	clave                  string
	definicion             ReferenciaVersionada
	seccionClave           string
	orden                  uint32
	criterios              []CriterioExperiencia
	grupoConcurrenciaClave string
	prioridadConcurrencia  uint32
	temporal               PoliticaUnidadTemporal
	jornada                PoliticaJornada
	restos                 PoliticaRestos
	redondeo               PoliticaRedondeo
	puntos                 baremacion.Puntos
	maximoUnidades         LimiteUnidades
	maximoPuntos           LimitePuntos
}

func componentesReglaPrueba(
	t *testing.T,
	clave string,
	seccion string,
	orden uint32,
	marca byte,
) componentesRegla {
	t.Helper()
	empleador, err := NuevoCriterioExperiencia(
		"empleador",
		referenciaPrueba(t, "catalogo:empleadores:"+clave, 2, marca),
		[]string{"administracion_local", "diputacion_granada"},
	)
	if err != nil {
		t.Fatal(err)
	}
	relacion, err := NuevoCriterioExperiencia(
		"relacion",
		referenciaPrueba(t, "catalogo:relaciones:"+clave, 1, marca+1),
		[]string{"laboral_temporal", "funcionario_interino"},
	)
	if err != nil {
		t.Fatal(err)
	}
	temporal, err := NuevaPoliticaUnidadTemporal(
		UnidadTemporalDia, UnidadTemporalMes, racionalPrueba(t, 30, 1), ExtremoFinalInclusivo,
	)
	if err != nil {
		t.Fatal(err)
	}
	jornada, err := NuevaPoliticaJornada(JornadaProporcional)
	if err != nil {
		t.Fatal(err)
	}
	restos, err := NuevaPoliticaRestos(RestosDescartarPorRegla)
	if err != nil {
		t.Fatal(err)
	}
	redondeo, err := NuevaPoliticaRedondeo(RedondearPorRegla, baremacion.RedondeoTruncar)
	if err != nil {
		t.Fatal(err)
	}
	return componentesRegla{
		clave:                  clave,
		definicion:             referenciaPrueba(t, "regla:experiencia:"+clave, 1, marca+2),
		seccionClave:           seccion,
		orden:                  orden,
		criterios:              []CriterioExperiencia{relacion, empleador},
		grupoConcurrenciaClave: "experiencia_mas_favorable",
		prioridadConcurrencia:  1,
		temporal:               temporal,
		jornada:                jornada,
		restos:                 restos,
		redondeo:               redondeo,
		puntos:                 puntosPrueba(t, 550_000),
		maximoUnidades:         limiteUnidadesPrueba(t, 120, 1),
		maximoPuntos:           limitePuntosPrueba(t, 10_000_000),
	}
}

func crearReglaPrueba(t *testing.T, c componentesRegla) ReglaExperiencia {
	t.Helper()
	regla, err := NuevaReglaExperiencia(
		c.clave, c.definicion, c.seccionClave, c.orden, c.criterios,
		c.grupoConcurrenciaClave, c.prioridadConcurrencia, c.temporal,
		c.jornada, c.restos, c.redondeo, c.puntos, c.maximoUnidades,
		c.maximoPuntos,
	)
	if err != nil {
		t.Fatal(err)
	}
	return regla
}

func referenciaPrueba(t *testing.T, referencia string, version uint64, marca byte) ReferenciaVersionada {
	t.Helper()
	suma := sha256.Sum256([]byte(referencia + ":" + string(marca)))
	valor, err := NuevaReferenciaVersionada(referencia, version, hex.EncodeToString(suma[:]))
	if err != nil {
		t.Fatal(err)
	}
	return valor
}

func racionalPrueba(t *testing.T, numerador, denominador int64) baremacion.Racional {
	t.Helper()
	valor, err := baremacion.NuevoRacional(numerador, denominador)
	if err != nil {
		t.Fatal(err)
	}
	return valor
}

func puntosPrueba(t *testing.T, micropuntos int64) baremacion.Puntos {
	t.Helper()
	valor, err := baremacion.PuntosDesdeMicropuntos(micropuntos)
	if err != nil {
		t.Fatal(err)
	}
	return valor
}

func limitePuntosPrueba(t *testing.T, micropuntos int64) LimitePuntos {
	t.Helper()
	limite, err := NuevoLimitePuntos(puntosPrueba(t, micropuntos))
	if err != nil {
		t.Fatal(err)
	}
	return limite
}

func limiteUnidadesPrueba(t *testing.T, numerador, denominador int64) LimiteUnidades {
	t.Helper()
	limite, err := NuevoLimiteUnidades(racionalPrueba(t, numerador, denominador))
	if err != nil {
		t.Fatal(err)
	}
	return limite
}

func numeroPrueba(valor int) string {
	const digitos = "0123456789"
	if valor < 10 {
		return string(digitos[valor])
	}
	return numeroPrueba(valor/10) + string(digitos[valor%10])
}

func invertir[T any](valores []T) {
	for izquierda, derecha := 0, len(valores)-1; izquierda < derecha; izquierda, derecha = izquierda+1, derecha-1 {
		valores[izquierda], valores[derecha] = valores[derecha], valores[izquierda]
	}
}
