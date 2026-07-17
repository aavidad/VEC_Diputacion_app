package calculoexperienciaoficial

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

type datosReciboConsumoFuentePrueba struct {
	consumo, fuente, prueba, auditoria        ReferenciaExactaV1
	decision, esquemaDecision, huellaDecision string
	recurso, huellaContexto, correlacion      string
	huellaSelector                            string
	consumidaEn                               time.Time
}

func TestReciboConsumoFuenteCanonicoRestaurableYConHuella(t *testing.T) {
	recibo := reciboConsumoFuentePrueba(t, datosReciboConsumoFuentePrueba{})
	primera, err := recibo.RepresentacionCanonicaV1()
	segunda, errSegunda := recibo.RepresentacionCanonicaV1()
	if err != nil || errSegunda != nil || !bytes.Equal(primera, segunda) {
		t.Fatalf("representacion no determinista: %v / %v", err, errSegunda)
	}
	huella, err := recibo.HuellaSHA256V1()
	if err != nil || !huellaSHA256Valida(huella) {
		t.Fatalf("huella no valida: %q, %v", huella, err)
	}
	restaurado, err := RestaurarReciboConsumoAutorizacionFuenteV1ConHuellaSHA256(primera, huella)
	if err != nil {
		t.Fatal(err)
	}
	huellaRestaurada, _ := restaurado.HuellaSHA256V1()
	if huellaRestaurada != huella {
		t.Fatal("la restauracion cambio la identidad")
	}
	primera[0] ^= 0xff
	tercera, _ := recibo.RepresentacionCanonicaV1()
	if !bytes.Equal(segunda, tercera) {
		t.Fatal("la representacion comparte memoria")
	}
	const vector = "6913274b17f6cc449eeae1d9d4f55041db0b2eba7dabfe1aa151578bbaa27278"
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
			d.correlacion, d.huellaSelector, d.fuente, d.prueba, d.auditoria, desde, hasta,
		)
	}
	if err := cotejar(base, base.consumidaEn.Add(-time.Microsecond), base.consumidaEn); err != nil {
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
		{"fuente_referencia", func(d *datosReciboConsumoFuentePrueba) { d.fuente.Referencia += ":otra" }},
		{"fuente_version", func(d *datosReciboConsumoFuentePrueba) { d.fuente.Version++ }},
		{"fuente_huella", func(d *datosReciboConsumoFuentePrueba) { d.fuente.HuellaSHA256 = hashPrueba("4") }},
		{"prueba_referencia", func(d *datosReciboConsumoFuentePrueba) { d.prueba.Referencia += ":otra" }},
		{"prueba_version", func(d *datosReciboConsumoFuentePrueba) { d.prueba.Version++ }},
		{"prueba_huella", func(d *datosReciboConsumoFuentePrueba) { d.prueba.HuellaSHA256 = hashPrueba("5") }},
		{"auditoria_referencia", func(d *datosReciboConsumoFuentePrueba) { d.auditoria.Referencia += ":otra" }},
		{"auditoria_version", func(d *datosReciboConsumoFuentePrueba) { d.auditoria.Version++ }},
		{"auditoria_huella", func(d *datosReciboConsumoFuentePrueba) { d.auditoria.HuellaSHA256 = hashPrueba("6") }},
	}
	for _, caso := range mutaciones {
		t.Run(caso.nombre, func(t *testing.T) {
			alterado := base
			caso.mutar(&alterado)
			if err := cotejar(alterado, base.consumidaEn, base.consumidaEn); !errors.Is(err, ErrHuellaNoCoincide) {
				t.Fatalf("campo alterado aceptado: %v", err)
			}
		})
	}
	for nombre, limites := range map[string][2]time.Time{
		"antes":   {base.consumidaEn.Add(time.Microsecond), base.consumidaEn.Add(2 * time.Microsecond)},
		"despues": {base.consumidaEn.Add(-2 * time.Microsecond), base.consumidaEn.Add(-time.Microsecond)},
	} {
		t.Run(nombre, func(t *testing.T) {
			if err := cotejar(base, limites[0], limites[1]); !errors.Is(err, ErrHuellaNoCoincide) {
				t.Fatalf("instante fuera de ventana aceptado: %v", err)
			}
		})
	}
}

func TestReciboConsumoFuenteLigaIdentidadConsumoEnMaterialCanonico(t *testing.T) {
	base := completarDatosReciboConsumoFuentePrueba(datosReciboConsumoFuentePrueba{})
	original := reciboConsumoFuentePrueba(t, base)
	huellaOriginal, _ := original.HuellaSHA256V1()
	mutaciones := []func(*datosReciboConsumoFuentePrueba){
		func(d *datosReciboConsumoFuentePrueba) { d.consumo.Referencia += ":otro" },
		func(d *datosReciboConsumoFuentePrueba) { d.consumo.Version++ },
		func(d *datosReciboConsumoFuentePrueba) { d.consumo.HuellaSHA256 = hashPrueba("7") },
	}
	for indice, mutar := range mutaciones {
		alterado := base
		mutar(&alterado)
		huella, err := reciboConsumoFuentePrueba(t, alterado).HuellaSHA256V1()
		if err != nil || huella == huellaOriginal {
			t.Fatalf("componente %d no quedo ligado: %v", indice, err)
		}
	}
	consumo, err := original.Consumo()
	instante, errInstante := original.ConsumidaEn()
	if err != nil || errInstante != nil || consumo != base.consumo || !instante.Equal(base.consumidaEn) {
		t.Fatal("las proyecciones tipadas no conservaron el consumo")
	}
}

func TestReciboConsumoFuenteRechazaInstanteQueExigiriaNormalizacion(t *testing.T) {
	base := completarDatosReciboConsumoFuentePrueba(datosReciboConsumoFuentePrueba{})
	casos := map[string]time.Time{
		"zona_no_utc":     base.consumidaEn.In(time.FixedZone("UTC+02", 2*60*60)),
		"submicrosegundo": base.consumidaEn.Add(time.Nanosecond),
		"cero":            {},
	}
	for nombre, instante := range casos {
		t.Run(nombre, func(t *testing.T) {
			_, err := NuevoReciboConsumoAutorizacionFuenteV1(
				base.consumo, base.decision, base.esquemaDecision, base.huellaDecision,
				base.recurso, base.huellaContexto, base.correlacion, base.huellaSelector,
				base.fuente, base.prueba, base.auditoria, instante,
			)
			if !errors.Is(err, ErrValorNoCanonico) {
				t.Fatalf("instante no canonico aceptado: %v", err)
			}
		})
	}
}

func TestReciboConsumoFuenteRechazaCadaCampoAusenteONoCanonico(t *testing.T) {
	base := completarDatosReciboConsumoFuentePrueba(datosReciboConsumoFuentePrueba{})
	casos := []struct {
		nombre string
		mutar  func(*datosReciboConsumoFuentePrueba)
	}{
		{"consumo_referencia", func(d *datosReciboConsumoFuentePrueba) { d.consumo.Referencia = "?" }},
		{"consumo_version", func(d *datosReciboConsumoFuentePrueba) { d.consumo.Version = 0 }},
		{"consumo_huella", func(d *datosReciboConsumoFuentePrueba) { d.consumo.HuellaSHA256 = "X" }},
		{"decision", func(d *datosReciboConsumoFuentePrueba) { d.decision = "?" }},
		{"esquema", func(d *datosReciboConsumoFuentePrueba) { d.esquemaDecision = "NO CANONICO" }},
		{"huella_decision", func(d *datosReciboConsumoFuentePrueba) { d.huellaDecision = "X" }},
		{"recurso", func(d *datosReciboConsumoFuentePrueba) { d.recurso = "?" }},
		{"huella_contexto", func(d *datosReciboConsumoFuentePrueba) { d.huellaContexto = "X" }},
		{"correlacion", func(d *datosReciboConsumoFuentePrueba) { d.correlacion = "?" }},
		{"huella_selector", func(d *datosReciboConsumoFuentePrueba) { d.huellaSelector = "X" }},
		{"fuente", func(d *datosReciboConsumoFuentePrueba) { d.fuente.Version = 0 }},
		{"prueba", func(d *datosReciboConsumoFuentePrueba) { d.prueba.HuellaSHA256 = "X" }},
		{"auditoria", func(d *datosReciboConsumoFuentePrueba) { d.auditoria.Referencia = "?" }},
		{"roles", func(d *datosReciboConsumoFuentePrueba) { d.decision = d.recurso }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			alterado := base
			caso.mutar(&alterado)
			if _, err := nuevoReciboConsumoFuenteDesdeDatos(alterado); err == nil {
				t.Fatal("campo no canonico aceptado")
			}
		})
	}
}

func TestRestaurarReciboConsumoFuenteRechazaAlteracionYJSONNoCanonico(t *testing.T) {
	recibo := reciboConsumoFuentePrueba(t, datosReciboConsumoFuentePrueba{})
	canonico, _ := recibo.RepresentacionCanonicaV1()
	huella, _ := recibo.HuellaSHA256V1()
	casos := [][]byte{
		append([]byte(" "), canonico...),
		bytes.Replace(canonico, []byte(`"esquema":`), []byte(`"esquema_ajeno":`), 1),
		bytes.Replace(canonico, []byte(`"decision_ref":`), []byte(`"decision_ref":"duplicada","decision_ref":`), 1),
	}
	for indice, caso := range casos {
		if _, err := RestaurarReciboConsumoAutorizacionFuenteV1ConHuellaSHA256(
			caso, huella,
		); err == nil {
			t.Fatalf("caso no canonico %d aceptado", indice)
		}
	}
	alterado := bytes.Replace(canonico, []byte(hashPrueba("a")), []byte(hashPrueba("f")), 1)
	if _, err := RestaurarReciboConsumoAutorizacionFuenteV1ConHuellaSHA256(alterado, huella); !errors.Is(err, ErrHuellaNoCoincide) {
		t.Fatalf("alteracion no detectada: %v", err)
	}
}

func reciboConsumoFuentePrueba(
	t *testing.T, datos datosReciboConsumoFuentePrueba,
) ReciboConsumoAutorizacionFuenteV1 {
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
) (ReciboConsumoAutorizacionFuenteV1, error) {
	return NuevoReciboConsumoAutorizacionFuenteV1(
		d.consumo, d.decision, d.esquemaDecision, d.huellaDecision, d.recurso,
		d.huellaContexto, d.correlacion, d.huellaSelector, d.fuente, d.prueba,
		d.auditoria, d.consumidaEn,
	)
}

func completarDatosReciboConsumoFuentePrueba(
	d datosReciboConsumoFuentePrueba,
) datosReciboConsumoFuentePrueba {
	if d.consumo.Referencia == "" {
		d.consumo = referenciaExactaReciboPrueba("consumo:autorizacion:fuente:1", 1, "0")
		d.decision = "decision:fuente:1"
		d.esquemaDecision = "vec.autorizacion.decision.reforzada.v2.solicitud-ligada"
		d.huellaDecision = hashPrueba("a")
		d.recurso = "fuente:recurso:1"
		d.huellaContexto = hashPrueba("b")
		d.correlacion = "correlacion:lectura:1"
		d.huellaSelector = hashPrueba("c")
		d.fuente = referenciaExactaReciboPrueba("fuente:exacta:1", 2, "d")
		d.prueba = referenciaExactaReciboPrueba("consumo:prueba:1", 3, "e")
		d.auditoria = referenciaExactaReciboPrueba("auditoria:fuente:1", 4, "f")
		d.consumidaEn = time.Date(2026, 7, 17, 10, 11, 12, 345678000, time.UTC)
	}
	return d
}

func referenciaExactaReciboPrueba(ref string, version uint64, marca string) ReferenciaExactaV1 {
	return ReferenciaExactaV1{Referencia: ref, Version: version, HuellaSHA256: hashPrueba(marca)}
}

func hashPrueba(marca string) string { return strings.Repeat(marca, 64) }
