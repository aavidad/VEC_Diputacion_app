package calculoexperienciaoficial

import "testing"

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
		{"huella_entrada", func(d *datosReciboConsumoFuentePrueba) { d.huellaEntrada = "X" }},
		{"fuente", func(d *datosReciboConsumoFuentePrueba) { d.fuente.Version = 0 }},
		{"verificador", func(d *datosReciboConsumoFuentePrueba) { d.verificador.Version = 0 }},
		{"prueba", func(d *datosReciboConsumoFuentePrueba) { d.prueba.HuellaSHA256 = "X" }},
		{"auditoria", func(d *datosReciboConsumoFuentePrueba) { d.auditoria.Referencia = "?" }},
		{"ventana_invertida", func(d *datosReciboConsumoFuentePrueba) { d.pruebaValidaHasta = d.pruebaEmitidaEn }},
		{"consumo_fuera_prueba", func(d *datosReciboConsumoFuentePrueba) { d.consumidaEn = d.pruebaValidaHasta }},
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

func TestReciboConsumoFuenteRechazaPIIYRecorridosEnCadaRolNominal(t *testing.T) {
	roles := []struct {
		nombre  string
		prefijo string
		mutar   func(*datosReciboConsumoFuentePrueba, string)
	}{
		{"consumo", "consumo:autorizacion:", func(d *datosReciboConsumoFuentePrueba, v string) { d.consumo.Referencia = v }},
		{"decision", "decision:", func(d *datosReciboConsumoFuentePrueba, v string) { d.decision = v }},
		{"evidencia", "evidencia:fuente:", func(d *datosReciboConsumoFuentePrueba, v string) { d.fuente.Referencia = v }},
		{"verificador", "verificador:fuente:", func(d *datosReciboConsumoFuentePrueba, v string) { d.verificador.Referencia = v }},
		{"prueba", "consumo:prueba:", func(d *datosReciboConsumoFuentePrueba, v string) { d.prueba.Referencia = v }},
		{"auditoria", "auditoria:fuente:", func(d *datosReciboConsumoFuentePrueba, v string) { d.auditoria.Referencia = v }},
	}
	hostiles := map[string]string{
		"dni": "12345678z", "nie": "x1234567l",
		"correo": "persona@example.test", "ruta": "../../expedientes",
	}
	for _, rol := range roles {
		for nombre, hostil := range hostiles {
			t.Run(rol.nombre+"_"+nombre, func(t *testing.T) {
				datos := completarDatosReciboConsumoFuentePrueba(datosReciboConsumoFuentePrueba{})
				rol.mutar(&datos, rol.prefijo+hostil)
				if _, err := nuevoReciboConsumoFuenteDesdeDatos(datos); err == nil {
					t.Fatal("rol nominal acepto PII o recorrido")
				}
			})
		}
	}
}

func TestReciboConsumoFuenteExigeRecursoYCorrelacionConContratoReal(t *testing.T) {
	base := completarDatosReciboConsumoFuentePrueba(datosReciboConsumoFuentePrueba{})
	casos := []func(*datosReciboConsumoFuentePrueba){
		func(d *datosReciboConsumoFuentePrueba) { d.recurso = "fuente:alias-vigente" },
		func(d *datosReciboConsumoFuentePrueba) { d.recurso = "12345678Z" },
		func(d *datosReciboConsumoFuentePrueba) { d.recurso = "persona@example.test" },
		func(d *datosReciboConsumoFuentePrueba) { d.recurso = "../../expedientes" },
		func(d *datosReciboConsumoFuentePrueba) { d.correlacion = "correlacion:legible" },
		func(d *datosReciboConsumoFuentePrueba) { d.correlacion = "12345678z" },
		func(d *datosReciboConsumoFuentePrueba) { d.correlacion = "x1234567l" },
		func(d *datosReciboConsumoFuentePrueba) { d.correlacion = "persona@example.test" },
		func(d *datosReciboConsumoFuentePrueba) { d.correlacion = "../../expedientes" },
		func(d *datosReciboConsumoFuentePrueba) {
			d.correlacion = "correlacion_0123456789ABCDEF0123456789ABCDEF"
		},
	}
	for indice, mutar := range casos {
		alterado := base
		mutar(&alterado)
		if _, err := nuevoReciboConsumoFuenteDesdeDatos(alterado); err == nil {
			t.Fatalf("gramatica nominal debil aceptada en caso %d", indice)
		}
	}
}

func TestReferenciasSelectorFuenteRechazanPIIYAlias(t *testing.T) {
	for _, hostil := range []string{
		"12345678z", "x1234567l", "persona@example.test", "../../expediente",
	} {
		if ReferenciaReglasFuenteExactaV1Valida("reglas:"+hostil) ||
			ReferenciaConvocatoriaFuenteExactaV1Valida("convocatoria:"+hostil) ||
			ReferenciaInstantaneaFuenteExactaV1Valida("iex_"+hostil) {
			t.Fatalf("selector acepto PII, ruta o alias: %q", hostil)
		}
	}
	if !ReferenciaReglasFuenteExactaV1Valida("reglas:oficial:v1") ||
		!ReferenciaConvocatoriaFuenteExactaV1Valida("convocatoria:oficial:v1") ||
		!ReferenciaInstantaneaFuenteExactaV1Valida("iex_"+hashPrueba("a")) {
		t.Fatal("perfil nominal rechazo referencias tecnicas reales")
	}
}
