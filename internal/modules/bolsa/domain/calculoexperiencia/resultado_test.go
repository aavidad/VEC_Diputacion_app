package calculoexperiencia

import (
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestResultadoExperienciaV1CompletadoEsCerradoYExplicable(t *testing.T) {
	resultado := resultadoCompletadoPrueba(t)
	if resultado.Estado() != ResultadoExperienciaCompletado ||
		resultado.Fase() != FaseResultadoCompletado || resultado.Validar() != nil {
		t.Fatalf("resultado completado invalido: %v", resultado.Validar())
	}
	total, presente := resultado.Total()
	if !presente || total.Micropuntos() != 1_000_000 {
		t.Fatalf("total inesperado: %v presente=%v", total, presente)
	}
	if resultado.Vinculos().FechaCorte().String() != "2026-12-31" ||
		resultado.Vinculos().Motor().Contrato() != contratoMotorResultadoV1 ||
		len(resultado.Intervalos()) != 1 || len(resultado.Aplicaciones()) != 1 ||
		len(resultado.Reglas()) != 1 || len(resultado.Secciones()) != 1 {
		t.Fatal("el resultado no conserva todos los vinculos y desgloses")
	}
}

func TestResultadoExperienciaV1ContratoDeLectura(t *testing.T) {
	resultado := resultadoCompletadoPrueba(t)
	vinculos := resultado.Vinculos()
	if vinculos.Plan().Esquema() != esquemaPlanResultadoV1 ||
		vinculos.Plan().HuellaSHA256() == "" || vinculos.Conjunto().Referencia() == "" ||
		vinculos.Entrada().Instantanea().Referencia() == "" ||
		vinculos.Entrada().HuellaContenidoSHA256() == "" ||
		vinculos.Motor().Version() != 1 || vinculos.Motor().HuellaContratoSHA256() == "" {
		t.Fatal("vinculos incompletos")
	}
	seleccion := resultado.Seleccion()
	seleccionada := seleccion.Aplicaciones()[0]
	if seleccionada.Tramo().Referencia() == "" || seleccionada.ReglaClave() != reglaResultadoPrueba ||
		seleccionada.GrupoClave() != grupoResultadoPrueba ||
		seleccionada.SeccionClave() != seccionResultadoPrueba ||
		seleccionada.Prioridad() != 1 || seleccionada.Razon() != RazonCoincidenciaUnica ||
		seleccion.Evaluaciones() != 1 {
		t.Fatal("seleccion ilegible")
	}
	intervalo := resultado.Intervalos()[0]
	if intervalo.Tramo().Referencia() == "" || intervalo.ReglaClave() != reglaResultadoPrueba ||
		intervalo.Periodo().Modo() != PeriodoServicioCerrado ||
		intervalo.Extremo() != reglasbaremo.ExtremoFinalExclusivo || intervalo.Dias() != 30 ||
		intervalo.Razon() != "" {
		t.Fatal("intervalo ilegible")
	}
	if _, presente := intervalo.Efectivo(); !presente {
		t.Fatal("falta intervalo efectivo")
	}
	aplicacion := resultado.Aplicaciones()[0]
	if aplicacion.Tramo().Referencia() == "" || aplicacion.ReglaClave() != reglaResultadoPrueba ||
		aplicacion.Jornada().FactorExacto() != "1/2" ||
		aplicacion.Unidades().Aportadas() != "1/2" ||
		aplicacion.Puntuacion().BrutoExacto() != "1000000/1" {
		t.Fatal("aplicacion ilegible")
	}
	regla := resultado.Reglas()[0]
	if regla.SeccionClave() != seccionResultadoPrueba || regla.ReglaClave() != reglaResultadoPrueba ||
		regla.UnidadesAgregadas() != "1/2" || regla.BrutoExacto() != "1000000/1" ||
		regla.PuntosFinalesExactos() != "1000000/1" {
		t.Fatal("regla ilegible")
	}
	seccion := resultado.Secciones()[0]
	if seccion.SeccionClave() != seccionResultadoPrueba ||
		seccion.AntesTopeExacto() != "1000000/1" ||
		seccion.PuntosFinales().Micropuntos() != 1_000_000 {
		t.Fatal("seccion ilegible")
	}
}

func TestResultadoExperienciaV1BloqueadoNoAvanzaDeFase(t *testing.T) {
	resultado := resultadoBloqueadoSeleccionPrueba(t)
	if resultado.Estado() != ResultadoExperienciaBloqueado ||
		resultado.Fase() != FaseResultadoSeleccion || resultado.Validar() != nil {
		t.Fatalf("resultado bloqueado invalido: %v", resultado.Validar())
	}
	if _, presente := resultado.Total(); presente || len(resultado.Intervalos()) != 0 ||
		len(resultado.Aplicaciones()) != 0 || len(resultado.Reglas()) != 0 ||
		len(resultado.Secciones()) != 0 || len(resultado.Bloqueos()) != 1 {
		t.Fatal("un bloqueo de seleccion avanzo hacia puntuacion")
	}

	manipulado := resultado.clonar()
	manipulado.fase = FaseResultadoPuntuacion
	if manipulado.Validar() == nil {
		t.Fatal("se acepto un salto de fase incoherente")
	}
}

func TestResultadoExperienciaV1ErrorTecnicoNoProduceResultado(t *testing.T) {
	registrador := registradorResultadoPrueba(t, seleccionExperiencia{})
	total, _ := baremacion.PuntosDesdeMicropuntos(1)
	resultado, err := registrador.sellarCompletado(total)
	if err == nil || resultado.Estado() != "" || resultado.Validar() == nil {
		t.Fatalf("un fallo tecnico produjo resultado util: estado=%q error=%v", resultado.Estado(), err)
	}
	if _, err := resultado.RepresentacionCanonica(); err == nil {
		t.Fatal("el valor cero obtuvo bytes canonicos")
	}
}

func TestResultadoExperienciaV1ConservaEnteroGrandeHastaTopeSeccion(t *testing.T) {
	resultado := resultadoGrandeRescatadoPrueba(t)
	reglas := resultado.Reglas()
	if len(reglas) != 1 || reglas[0].PuntosFinalesExactos() != "18000000000000000/1" {
		t.Fatalf("se perdio el entero previo al tope de seccion: %+v", reglas)
	}
	secciones := resultado.Secciones()
	if len(secciones) != 1 || secciones[0].PuntosFinales().Micropuntos() != 1_000_000 {
		t.Fatal("el tope de seccion no rescato el intermedio grande")
	}
	if err := resultado.Validar(); err != nil {
		t.Fatal(err)
	}
}

func TestResultadoExperienciaV1CopiasDefensivas(t *testing.T) {
	completado := resultadoCompletadoPrueba(t)
	seleccion := completado.Seleccion()
	seleccion.aplicaciones[0].reglaClave = "alterada"
	intervalos := completado.Intervalos()
	intervalos[0].reglaClave = "alterada"
	if completado.Seleccion().aplicaciones[0].reglaClave != reglaResultadoPrueba ||
		completado.Intervalos()[0].reglaClave != reglaResultadoPrueba {
		t.Fatal("un accesor modifico el agregado")
	}

	bloqueado := resultadoBloqueadoSeleccionPrueba(t)
	bloqueos := bloqueado.Bloqueos()
	bloqueos[0].tramos[0] = reglasbaremo.ReferenciaVersionada{}
	if bloqueado.Bloqueos()[0].tramos[0].Referencia() == "" {
		t.Fatal("la copia anidada de bloqueos comparte memoria")
	}
}

func TestResultadoExperienciaV1RechazaJornadasSemanticasIncoherentes(t *testing.T) {
	media, err := baremacion.NuevaFraccionJornada(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	base := JornadaResultadoExperienciaV1{
		origen: media, modo: reglasbaremo.JornadaProporcional,
		factor: exactoResultadoPrueba(t, "1/2"), razon: RazonJornadaProporcional,
	}
	casos := []JornadaResultadoExperienciaV1{
		func() JornadaResultadoExperienciaV1 { v := base; v.factor = exactoResultadoPrueba(t, "1/1"); return v }(),
		func() JornadaResultadoExperienciaV1 {
			v := base
			v.modo = reglasbaremo.JornadaIntegra
			v.razon = RazonJornadaIntegra
			return v
		}(),
		func() JornadaResultadoExperienciaV1 {
			v := base
			v.modo = reglasbaremo.JornadaIntegraDesdeUmbral
			v.razon = RazonUmbralAlcanzado
			return v
		}(),
		func() JornadaResultadoExperienciaV1 {
			v := base
			v.modo = reglasbaremo.JornadaProtegidaIntegra
			v.razon = RazonProteccionAtestada
			v.atestacionPresente = true
			v.atestacionUsada = true
			return v
		}(),
	}
	for indice, caso := range casos {
		if validarJornadaResultadoV1(caso) == nil {
			t.Errorf("caso %d acepto factor que contradice la politica", indice)
		}
	}
}

func TestResultadoExperienciaV1RechazaAgregadoDeUnidadesManipulado(t *testing.T) {
	resultado := resultadoCompletadoPrueba(t).clonar()
	resultado.reglas[0].unidadesAgregadas = exactoResultadoPrueba(t, "3/2")
	resultado.reglas[0].unidadesTrasRestos = exactoResultadoPrueba(t, "3/2")
	resultado.reglas[0].topeUnidades = topeIlimitadoResultadoPrueba(t, "3/2")
	resultado.reglas[0].bruto = exactoResultadoPrueba(t, "3000000/1")
	resultado.reglas[0].redondeo.entrada = exactoResultadoPrueba(t, "3000000/1")
	resultado.reglas[0].redondeo.salida = exactoResultadoPrueba(t, "3000000/1")
	resultado.reglas[0].topePuntos = topeIlimitadoResultadoPrueba(t, "3000000/1")
	resultado.reglas[0].puntosFinales = exactoResultadoPrueba(t, "3000000/1")
	resultado.secciones[0].antesTope = exactoResultadoPrueba(t, "3000000/1")
	resultado.secciones[0].tope = topeLimitadoResultadoPrueba(
		t, "3000000/1", "10000000/1", "3000000/1", false,
	)
	puntos, _ := baremacion.PuntosDesdeMicropuntos(3_000_000)
	resultado.secciones[0].puntosFinales = puntos
	resultado.total = puntos
	if resultado.Validar() == nil {
		t.Fatal("se acepto una regla que no suma las aplicaciones")
	}
}

func TestResultadoExperienciaV1IntervaloExcluidoSoloAdmiteAplicacionCero(t *testing.T) {
	resultado := resultadoCompletadoPrueba(t).clonar()
	resultado.vinculos.fechaCorte = fechaResultadoPrueba(t, 2025, 12, 31)
	resultado.intervalos[0].efectivo = baremacion.IntervaloCivil{}
	resultado.intervalos[0].tieneEfectivo = false
	resultado.intervalos[0].dias = 0
	resultado.intervalos[0].razon = RazonPosteriorCorte
	ceroExacto := exactoResultadoPrueba(t, "0/1")
	resultado.aplicaciones[0].unidades = UnidadesAplicacionResultadoExperienciaV1{
		exactas: ceroExacto, aportadas: ceroExacto, resto: ceroExacto,
		frontera: FronteraRestosResultadoExacta,
	}
	resultado.aplicaciones[0].puntuacion.bruto = ceroExacto
	resultado.reglas[0].unidadesAgregadas = ceroExacto
	resultado.reglas[0].unidadesTrasRestos = ceroExacto
	resultado.reglas[0].restoRegla = ceroExacto
	resultado.reglas[0].topeUnidades = topeIlimitadoResultadoPrueba(t, "0/1")
	resultado.reglas[0].bruto = ceroExacto
	resultado.reglas[0].redondeo.entrada = ceroExacto
	resultado.reglas[0].redondeo.salida = ceroExacto
	resultado.reglas[0].topePuntos = topeIlimitadoResultadoPrueba(t, "0/1")
	resultado.reglas[0].puntosFinales = ceroExacto
	resultado.secciones[0].antesTope = ceroExacto
	resultado.secciones[0].tope = topeLimitadoResultadoPrueba(
		t, "0/1", "10000000/1", "0/1", false,
	)
	ceroPuntos, _ := baremacion.PuntosDesdeMicropuntos(0)
	resultado.secciones[0].puntosFinales = ceroPuntos
	resultado.total = ceroPuntos
	if err := resultado.Validar(); err != nil {
		t.Fatal(err)
	}
	resultado.aplicaciones[0].puntuacion.bruto = exactoResultadoPrueba(t, "1/1")
	if resultado.Validar() == nil {
		t.Fatal("un intervalo excluido produjo puntuacion")
	}
}

func TestExactoResultadoV1EsCanonicoYAcotado(t *testing.T) {
	for _, valor := range []string{"01/1", "1/01", "2/2", "0/2", "-1/2", "1/0", "1"} {
		if _, err := nuevoExactoResultadoV1(valor); err == nil {
			t.Errorf("se acepto %q", valor)
		}
	}
	if _, err := nuevoExactoResultadoV1("1/3"); err != nil {
		t.Fatal(err)
	}
	entrada := exactoResultadoPrueba(t, "5/2")
	redondeada, err := redondearExactoResultadoV1(entrada, baremacion.RedondeoMitadAlPar)
	if err != nil || redondeada.texto() != "2/1" {
		t.Fatalf("redondeo mitad al par incorrecto: %s %v", redondeada.texto(), err)
	}
	_, err = redondearExactoResultadoV1(entrada, baremacion.RedondeoExacto)
	if !errors.Is(err, ErrResultadoNoExacto) {
		t.Fatalf("redondeo exacto no bloqueo: %v", err)
	}
}
