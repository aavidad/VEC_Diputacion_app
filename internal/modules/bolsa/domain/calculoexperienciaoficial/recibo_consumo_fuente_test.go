package calculoexperienciaoficial

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

type datosReciboConsumoFuentePrueba struct {
	consumo, fuente, verificador, prueba, auditoria ReferenciaExactaV1
	decision, esquemaDecision, huellaDecision       string
	recurso, huellaContexto, correlacion            string
	huellaSelector, huellaEntrada                   string
	pruebaEmitidaEn, pruebaValidaHasta              time.Time
	consumidaEn, obtenidaEn                         time.Time
}

func TestReciboConsumoFuenteCanonicoRestaurableYConHuella(t *testing.T) {
	recibo := reciboConsumoFuentePrueba(t, datosReciboConsumoFuentePrueba{})
	primera, err := recibo.RepresentacionCanonicaV2()
	segunda, errSegunda := recibo.RepresentacionCanonicaV2()
	if err != nil || errSegunda != nil || !bytes.Equal(primera, segunda) {
		t.Fatalf("representacion no determinista: %v / %v", err, errSegunda)
	}
	huella, err := recibo.HuellaSHA256V2()
	if err != nil || !huellaSHA256Valida(huella) {
		t.Fatalf("huella no valida: %q, %v", huella, err)
	}
	restaurado, err := RestaurarReciboConsumoAutorizacionFuenteV2ConHuellaSHA256(primera, huella)
	if err != nil {
		t.Fatal(err)
	}
	huellaRestaurada, _ := restaurado.HuellaSHA256V2()
	if huellaRestaurada != huella {
		t.Fatal("la restauracion cambio la identidad")
	}
	primera[0] ^= 0xff
	tercera, _ := recibo.RepresentacionCanonicaV2()
	if !bytes.Equal(segunda, tercera) {
		t.Fatal("la representacion comparte memoria")
	}
	const vector = "6641533ef41608efab600ae8e25e728e98034b0bc7aedab47fd8c73cf7ad7d86"
	if huella != vector {
		t.Fatalf("vector canonico modificado: %s", huella)
	}
}

func TestReciboConsumoFuenteCotejaDecisionSelectorYArtefactosCampoACampo(t *testing.T) {
	base := completarDatosReciboConsumoFuentePrueba(datosReciboConsumoFuentePrueba{})
	recibo := reciboConsumoFuentePrueba(t, base)
	cotejar := func(d datosReciboConsumoFuentePrueba, desde, hasta time.Time) error {
		return recibo.ValidarPara(
			d.decision, d.esquemaDecision, d.huellaDecision, d.recurso, d.huellaContexto,
			d.correlacion, d.huellaSelector, d.huellaEntrada, d.fuente, d.verificador,
			d.prueba, d.auditoria, d.pruebaEmitidaEn, d.pruebaValidaHasta,
			d.obtenidaEn, desde, hasta,
		)
	}
	if err := cotejar(
		base, base.consumidaEn.Add(-time.Microsecond), base.obtenidaEn,
	); err != nil {
		t.Fatalf("cotejo exacto rechazado: %v", err)
	}
	mutaciones := []struct {
		nombre string
		mutar  func(*datosReciboConsumoFuentePrueba)
	}{
		{"decision_ref", func(d *datosReciboConsumoFuentePrueba) { d.decision += ":otra" }},
		{"esquema_decision", func(d *datosReciboConsumoFuentePrueba) { d.esquemaDecision += ".otro" }},
		{"huella_decision", func(d *datosReciboConsumoFuentePrueba) { d.huellaDecision = hashPrueba("1") }},
		{"recurso_ref", func(d *datosReciboConsumoFuentePrueba) { d.recurso += ":otro" }},
		{"huella_contexto", func(d *datosReciboConsumoFuentePrueba) { d.huellaContexto = hashPrueba("2") }},
		{"correlacion_ref", func(d *datosReciboConsumoFuentePrueba) { d.correlacion += ":otra" }},
		{"huella_selector", func(d *datosReciboConsumoFuentePrueba) { d.huellaSelector = hashPrueba("3") }},
		{"huella_entrada", func(d *datosReciboConsumoFuentePrueba) { d.huellaEntrada = hashPrueba("0") }},
		{"fuente_referencia", func(d *datosReciboConsumoFuentePrueba) { d.fuente.Referencia += ":otra" }},
		{"fuente_version", func(d *datosReciboConsumoFuentePrueba) { d.fuente.Version++ }},
		{"fuente_huella", func(d *datosReciboConsumoFuentePrueba) { d.fuente.HuellaSHA256 = hashPrueba("4") }},
		{"verificador_referencia", func(d *datosReciboConsumoFuentePrueba) { d.verificador.Referencia += ":otro" }},
		{"verificador_version", func(d *datosReciboConsumoFuentePrueba) { d.verificador.Version++ }},
		{"verificador_huella", func(d *datosReciboConsumoFuentePrueba) { d.verificador.HuellaSHA256 = hashPrueba("9") }},
		{"prueba_referencia", func(d *datosReciboConsumoFuentePrueba) { d.prueba.Referencia += ":otra" }},
		{"prueba_version", func(d *datosReciboConsumoFuentePrueba) { d.prueba.Version++ }},
		{"prueba_huella", func(d *datosReciboConsumoFuentePrueba) { d.prueba.HuellaSHA256 = hashPrueba("5") }},
		{"auditoria_referencia", func(d *datosReciboConsumoFuentePrueba) { d.auditoria.Referencia += ":otra" }},
		{"auditoria_version", func(d *datosReciboConsumoFuentePrueba) { d.auditoria.Version++ }},
		{"auditoria_huella", func(d *datosReciboConsumoFuentePrueba) { d.auditoria.HuellaSHA256 = hashPrueba("6") }},
		{"prueba_emitida", func(d *datosReciboConsumoFuentePrueba) { d.pruebaEmitidaEn = d.pruebaEmitidaEn.Add(-time.Microsecond) }},
		{"prueba_validez", func(d *datosReciboConsumoFuentePrueba) {
			d.pruebaValidaHasta = d.pruebaValidaHasta.Add(time.Microsecond)
		}},
		{"obtenida_en", func(d *datosReciboConsumoFuentePrueba) {
			d.obtenidaEn = d.obtenidaEn.Add(time.Microsecond)
		}},
	}
	for _, caso := range mutaciones {
		t.Run(caso.nombre, func(t *testing.T) {
			alterado := base
			caso.mutar(&alterado)
			if err := cotejar(alterado, base.consumidaEn, base.obtenidaEn); !errors.Is(err, ErrHuellaNoCoincide) {
				t.Fatalf("campo alterado aceptado: %v", err)
			}
		})
	}
	for nombre, limites := range map[string][2]time.Time{
		"antes":   {base.consumidaEn.Add(time.Microsecond), base.obtenidaEn.Add(time.Microsecond)},
		"despues": {base.consumidaEn.Add(-2 * time.Microsecond), base.obtenidaEn.Add(-time.Microsecond)},
	} {
		t.Run(nombre, func(t *testing.T) {
			if err := cotejar(base, limites[0], limites[1]); !errors.Is(err, ErrHuellaNoCoincide) {
				t.Fatalf("instante fuera de ventana aceptado: %v", err)
			}
		})
	}
	if err := cotejar(
		base, base.consumidaEn, base.pruebaValidaHasta,
	); !errors.Is(err, ErrValorInvalido) {
		t.Fatalf("la comprobacion en el instante de caducidad fue aceptada: %v", err)
	}
}

func TestReciboConsumoFuenteLigaIdentidadConsumoEnMaterialCanonico(t *testing.T) {
	base := completarDatosReciboConsumoFuentePrueba(datosReciboConsumoFuentePrueba{})
	original := reciboConsumoFuentePrueba(t, base)
	huellaOriginal, _ := original.HuellaSHA256V2()
	mutaciones := []func(*datosReciboConsumoFuentePrueba){
		func(d *datosReciboConsumoFuentePrueba) {
			d.consumo.Referencia = "consumo:autorizacion:" + hashPrueba("6")
		},
		func(d *datosReciboConsumoFuentePrueba) { d.consumo.Version++ },
		func(d *datosReciboConsumoFuentePrueba) { d.consumo.HuellaSHA256 = hashPrueba("7") },
	}
	for indice, mutar := range mutaciones {
		alterado := base
		mutar(&alterado)
		huella, err := reciboConsumoFuentePrueba(t, alterado).HuellaSHA256V2()
		if err != nil || huella == huellaOriginal {
			t.Fatalf("componente %d no quedo ligado: %v", indice, err)
		}
	}
	consumo, err := original.Consumo()
	instante, errInstante := original.ConsumidaEn()
	obtenida, errObtenida := original.ObtenidaEn()
	if err != nil || errInstante != nil || errObtenida != nil || consumo != base.consumo ||
		!instante.Equal(base.consumidaEn) || !obtenida.Equal(base.obtenidaEn) {
		t.Fatal("las proyecciones tipadas no conservaron el consumo")
	}
}

func TestReciboConsumoFuenteRechazaInstanteQueExigiriaNormalizacion(t *testing.T) {
	base := completarDatosReciboConsumoFuentePrueba(datosReciboConsumoFuentePrueba{})
	casos := []struct {
		nombre string
		mutar  func(*datosReciboConsumoFuentePrueba)
	}{
		{"emitida_zona", func(d *datosReciboConsumoFuentePrueba) {
			d.pruebaEmitidaEn = d.pruebaEmitidaEn.In(time.FixedZone("UTC+02", 7200))
		}},
		{"validez_submicro", func(d *datosReciboConsumoFuentePrueba) {
			d.pruebaValidaHasta = d.pruebaValidaHasta.Add(time.Nanosecond)
		}},
		{"consumo_cero", func(d *datosReciboConsumoFuentePrueba) { d.consumidaEn = time.Time{} }},
		{"obtencion_submicro", func(d *datosReciboConsumoFuentePrueba) {
			d.obtenidaEn = d.obtenidaEn.Add(time.Nanosecond)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			alterado := base
			caso.mutar(&alterado)
			_, err := nuevoReciboConsumoFuenteDesdeDatos(alterado)
			if !errors.Is(err, ErrValorNoCanonico) {
				t.Fatalf("instante no canonico aceptado: %v", err)
			}
		})
	}
}

func TestRestaurarReciboConsumoFuenteRechazaAlteracionYJSONNoCanonico(t *testing.T) {
	recibo := reciboConsumoFuentePrueba(t, datosReciboConsumoFuentePrueba{})
	canonico, _ := recibo.RepresentacionCanonicaV2()
	huella, _ := recibo.HuellaSHA256V2()
	casos := [][]byte{
		append([]byte(" "), canonico...),
		bytes.Replace(canonico, []byte("autorizacion-fuente.v2"), []byte("autorizacion-fuente.v1"), 1),
		bytes.Replace(canonico, []byte(`"esquema":`), []byte(`"esquema_ajeno":`), 1),
		bytes.Replace(canonico, []byte(`"decision_ref":`), []byte(`"decision_ref":"duplicada","decision_ref":`), 1),
	}
	for indice, caso := range casos {
		if _, err := RestaurarReciboConsumoAutorizacionFuenteV2ConHuellaSHA256(
			caso, huella,
		); err == nil {
			t.Fatalf("caso no canonico %d aceptado", indice)
		}
	}
	alterado := bytes.Replace(canonico, []byte(hashPrueba("a")), []byte(hashPrueba("f")), 1)
	if _, err := RestaurarReciboConsumoAutorizacionFuenteV2ConHuellaSHA256(alterado, huella); !errors.Is(err, ErrHuellaNoCoincide) {
		t.Fatalf("alteracion no detectada: %v", err)
	}
}

func reciboConsumoFuentePrueba(
	t *testing.T, datos datosReciboConsumoFuentePrueba,
) ReciboConsumoAutorizacionFuenteV2 {
	t.Helper()
	d := completarDatosReciboConsumoFuentePrueba(datos)
	recibo, err := nuevoReciboConsumoFuenteDesdeDatos(d)
	if err != nil {
		t.Fatal(err)
	}
	return recibo
}

func nuevoReciboConsumoFuenteDesdeDatos(
	d datosReciboConsumoFuentePrueba,
) (ReciboConsumoAutorizacionFuenteV2, error) {
	return NuevoReciboConsumoAutorizacionFuenteV2(
		d.consumo, d.decision, d.esquemaDecision, d.huellaDecision, d.recurso,
		d.huellaContexto, d.correlacion, d.huellaSelector, d.huellaEntrada,
		d.fuente, d.verificador, d.prueba, d.auditoria, d.pruebaEmitidaEn,
		d.pruebaValidaHasta, d.consumidaEn, d.obtenidaEn,
	)
}

func completarDatosReciboConsumoFuentePrueba(
	d datosReciboConsumoFuentePrueba,
) datosReciboConsumoFuentePrueba {
	if d.consumo.Referencia == "" {
		d.consumo = referenciaExactaReciboPrueba(
			"consumo:autorizacion:"+hashPrueba("0"), 1, "0",
		)
		d.decision = "decision:" + hashPrueba("1")
		d.esquemaDecision = "vec.autorizacion.decision.reforzada.v2.solicitud-ligada"
		d.huellaDecision = hashPrueba("a")
		d.recurso = "fuente:" + hashPrueba("9")
		d.huellaContexto = hashPrueba("b")
		d.correlacion = "correlacion_0123456789abcdef0123456789abcdef"
		d.huellaSelector = hashPrueba("c")
		d.huellaEntrada = hashPrueba("8")
		d.fuente = referenciaExactaReciboPrueba("evidencia:fuente:"+hashPrueba("2"), 2, "d")
		d.verificador = referenciaExactaReciboPrueba("verificador:fuente:"+hashPrueba("3"), 1, "7")
		d.prueba = referenciaExactaReciboPrueba("consumo:prueba:"+hashPrueba("4"), 3, "e")
		d.auditoria = referenciaExactaReciboPrueba("auditoria:fuente:"+hashPrueba("5"), 4, "f")
		d.consumidaEn = time.Date(2026, 7, 17, 10, 11, 12, 345678000, time.UTC)
		d.obtenidaEn = d.consumidaEn.Add(time.Microsecond)
		d.pruebaEmitidaEn = d.consumidaEn.Add(-time.Minute)
		d.pruebaValidaHasta = d.consumidaEn.Add(time.Minute)
	}
	return d
}

func referenciaExactaReciboPrueba(ref string, version uint64, marca string) ReferenciaExactaV1 {
	return ReferenciaExactaV1{Referencia: ref, Version: version, HuellaSHA256: hashPrueba(marca)}
}

func hashPrueba(marca string) string { return strings.Repeat(marca, 64) }
