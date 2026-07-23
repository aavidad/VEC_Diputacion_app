package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAnalisisRRHHAceptaCamposGobernadosYLimitesValidos(t *testing.T) {
	analisis := analisisCompletoParaPrueba()
	analisis.ModalidadClave = "modalidad_publicada_2030"
	analisis.CausaClave = "causa_programa_europeo"
	analisis.PorcentajeJornada = 1
	if err := analisis.Validar(); err != nil {
		t.Fatalf("jornada mínima y claves gobernadas válidas: %v", err)
	}

	analisis.PorcentajeJornada = JornadaCompletaDiezmilesimas
	if err := analisis.Validar(); err != nil {
		t.Fatalf("jornada completa válida: %v", err)
	}
}

func TestAnalisisRRHHRechazaCamposObligatoriosInvalidos(t *testing.T) {
	pruebas := []struct {
		nombre    string
		modificar func(*AnalisisRRHH)
	}{
		{"modalidad vacía", func(a *AnalisisRRHH) { a.ModalidadClave = "" }},
		{"categoría inválida", func(a *AnalisisRRHH) { a.CategoriaRef = "x" }},
		{"grupo inválido", func(a *AnalisisRRHH) { a.GrupoSubgrupo = "a2" }},
		{"causa vacía", func(a *AnalisisRRHH) { a.CausaClave = "" }},
		{"jornada nula", func(a *AnalisisRRHH) { a.PorcentajeJornada = 0 }},
		{"jornada superior al total", func(a *AnalisisRRHH) {
			a.PorcentajeJornada = JornadaCompletaDiezmilesimas + 1
		}},
		{"observaciones demasiado extensas", func(a *AnalisisRRHH) {
			a.Observaciones = strings.Repeat("á", 4001)
		}},
		{"observaciones no normalizadas", func(a *AnalisisRRHH) {
			a.Observaciones = "revisio\u0301n"
		}},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			analisis := analisisCompletoParaPrueba()
			prueba.modificar(&analisis)
			if analisis.Validar() == nil {
				t.Fatal("se aceptó un análisis inválido")
			}
		})
	}
}

func TestAnalisisRRHHAcotaPeriodoSinCodificarDuracionesLaborales(t *testing.T) {
	analisis := analisisCompletoParaPrueba()
	analisis.Periodo.Fin = analisis.Periodo.Inicio.AddDate(maximoAniosPeriodoAnalisis, 0, 0)
	if err := analisis.Validar(); err != nil {
		t.Fatalf("límite técnico válido: %v", err)
	}

	pruebas := []struct {
		nombre    string
		modificar func(*PeriodoPrevisto)
	}{
		{"orden inverso", func(p *PeriodoPrevisto) { p.Fin = p.Inicio.AddDate(0, 0, -1) }},
		{"más de cien años", func(p *PeriodoPrevisto) {
			p.Fin = p.Inicio.AddDate(maximoAniosPeriodoAnalisis, 0, 1)
		}},
		{"inicio con hora", func(p *PeriodoPrevisto) {
			p.Inicio = p.Inicio.Add(time.Hour)
		}},
		{"fin fuera de UTC", func(p *PeriodoPrevisto) {
			p.Fin = time.Date(2027, 3, 31, 0, 0, 0, 0, time.FixedZone("local", 3600))
		}},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			caso := analisisCompletoParaPrueba()
			prueba.modificar(&caso.Periodo)
			if caso.Validar() == nil {
				t.Fatal("se aceptó un periodo inválido")
			}
		})
	}
}

func TestValidacionRCAceptaDecisionesConEvidenciaCoherente(t *testing.T) {
	validada := validacionRCParaPrueba()
	if err := validada.Validar(); err != nil {
		t.Fatalf("RC validada: %v", err)
	}
	if !validada.HabilitaAvance() {
		t.Fatal("una RC validada no habilitó el avance")
	}

	for _, resultado := range []ResultadoValidacionRC{RCRechazada, RCNoRequerida} {
		validacion := validacionRCParaPrueba()
		validacion.Resultado = resultado
		validacion.FechaRC = nil
		validacion.Numero = ""
		validacion.Importe = nil
		validacion.DocumentoRef = ""
		validacion.Motivo = "Decisión motivada conforme a la fuente presupuestaria."
		if err := validacion.Validar(); err != nil {
			t.Fatalf("%s con evidencia coherente: %v", resultado, err)
		}
		esperaAvance := resultado == RCNoRequerida
		if validacion.HabilitaAvance() != esperaAvance {
			t.Fatalf("avance inesperado para %s: %t", resultado, validacion.HabilitaAvance())
		}
	}

	if (ValidacionRC{}).HabilitaAvance() {
		t.Fatal("la ausencia de validación, que representa pendiente, habilitó el avance")
	}
}

func TestValidacionRCRechazaEvidenciaIncompletaOResidual(t *testing.T) {
	pruebas := []struct {
		nombre    string
		modificar func(*ValidacionRC)
	}{
		{"resultado desconocido", func(v *ValidacionRC) { v.Resultado = "desconocida" }},
		{"vocabulario antiguo", func(v *ValidacionRC) { v.Resultado = "no_validada" }},
		{"pendiente codificado", func(v *ValidacionRC) { v.Resultado = "pendiente" }},
		{"entrada ausente", func(v *ValidacionRC) { v.EntradaRef = "" }},
		{"huella ausente", func(v *ValidacionRC) { v.HuellaEntradaSHA256 = "" }},
		{"huella centinela", func(v *ValidacionRC) {
			v.HuellaEntradaSHA256 = strings.Repeat("0", 64)
		}},
		{"fuente ausente", func(v *ValidacionRC) { v.FuenteRef = "" }},
		{"recibo ausente", func(v *ValidacionRC) { v.ReciboRef = "" }},
		{"instante ausente", func(v *ValidacionRC) { v.ValidadaEn = time.Time{} }},
		{"instante no canónico", func(v *ValidacionRC) {
			v.ValidadaEn = v.ValidadaEn.Add(time.Nanosecond)
		}},
		{"validada sin fecha autoritativa", func(v *ValidacionRC) { v.FechaRC = nil }},
		{"fecha autoritativa con hora", func(v *ValidacionRC) {
			fecha := v.FechaRC.Add(time.Hour)
			v.FechaRC = &fecha
		}},
		{"fecha autoritativa fuera de UTC", func(v *ValidacionRC) {
			fecha := time.Date(2026, 7, 20, 0, 0, 0, 0, time.FixedZone("local", 3600))
			v.FechaRC = &fecha
		}},
		{"validada sin número", func(v *ValidacionRC) { v.Numero = "" }},
		{"validada sin importe", func(v *ValidacionRC) { v.Importe = nil }},
		{"validada sin documento", func(v *ValidacionRC) { v.DocumentoRef = "" }},
		{"importe no calculable", func(v *ValidacionRC) {
			v.Importe.Centimos = maximoCentimosCalculablesAnalisis + 1
		}},
		{"rechazada sin motivo", func(v *ValidacionRC) {
			prepararRCNegativa(v, RCRechazada)
			v.Motivo = ""
		}},
		{"rechazada con fecha residual", func(v *ValidacionRC) {
			prepararRCNegativa(v, RCRechazada)
			v.FechaRC = fechaParaPrueba(2026, 7, 20)
		}},
		{"no requerida con número residual", func(v *ValidacionRC) {
			prepararRCNegativa(v, RCNoRequerida)
			v.Numero = "rc_residual_01"
		}},
		{"no requerida con importe residual", func(v *ValidacionRC) {
			prepararRCNegativa(v, RCNoRequerida)
			v.Importe = importeParaPrueba(1)
		}},
		{"no requerida con documento residual", func(v *ValidacionRC) {
			prepararRCNegativa(v, RCNoRequerida)
			v.DocumentoRef = "documento_residual_01"
		}},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			validacion := validacionRCParaPrueba()
			prueba.modificar(&validacion)
			if validacion.Validar() == nil {
				t.Fatal("se aceptó una validación de RC incoherente")
			}
		})
	}
}

func TestAnalisisRRHHCotejaLaEntradaExactaEsperada(t *testing.T) {
	pruebas := []struct {
		nombre    string
		modificar func(*AnalisisRRHH)
	}{
		{"referencia esperada distinta", func(a *AnalisisRRHH) {
			a.EntradaRCEsperada.Referencia = "declaracion_rc_distinta_01"
		}},
		{"huella esperada distinta", func(a *AnalisisRRHH) {
			a.EntradaRCEsperada.HuellaSHA256 = strings.Repeat("b", 64)
		}},
		{"referencia recibida distinta", func(a *AnalisisRRHH) {
			a.ValidacionRC.EntradaRef = "declaracion_rc_distinta_01"
		}},
		{"huella recibida distinta", func(a *AnalisisRRHH) {
			a.ValidacionRC.HuellaEntradaSHA256 = strings.Repeat("b", 64)
		}},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			analisis := analisisCompletoParaPrueba()
			prueba.modificar(&analisis)
			if analisis.Validar() == nil || analisis.HabilitaAvance() {
				t.Fatal("una validación de otra entrada superó el cotejo semántico")
			}
		})
	}
}

func TestValidacionRCOrdenaFechaEInstanteAutoritativos(t *testing.T) {
	validacion := validacionRCParaPrueba()
	validacion.ValidadaEn = fechaAnalisis(2026, 7, 20)
	validacion.FechaRC = fechaParaPrueba(2026, 7, 20)
	if err := validacion.Validar(); err != nil {
		t.Fatalf("fecha igual al instante de validación: %v", err)
	}

	validacion.FechaRC = fechaParaPrueba(2026, 7, 21)
	if validacion.Validar() == nil {
		t.Fatal("se aceptó una fecha de RC posterior a su validación")
	}
}

func TestValidacionRCOmiteResiduosEnJSONCuandoNoHayRCValidada(t *testing.T) {
	for _, resultado := range []ResultadoValidacionRC{RCRechazada, RCNoRequerida} {
		t.Run(string(resultado), func(t *testing.T) {
			validacion := validacionRCParaPrueba()
			prepararRCNegativa(&validacion, resultado)
			contenido, err := json.Marshal(validacion)
			if err != nil {
				t.Fatalf("serializar: %v", err)
			}
			var campos map[string]json.RawMessage
			if err := json.Unmarshal(contenido, &campos); err != nil {
				t.Fatalf("decodificar: %v", err)
			}
			for _, campo := range []string{"fecha_rc", "numero", "importe", "documento_ref"} {
				if _, existe := campos[campo]; existe {
					t.Fatalf("el JSON de %s conserva el residuo %q: %s", resultado, campo, contenido)
				}
			}
			for _, campo := range []string{
				"resultado", "entrada_ref", "huella_entrada_sha256",
				"fuente_ref", "recibo_ref", "validada_en", "motivo",
			} {
				if _, existe := campos[campo]; !existe {
					t.Fatalf("el JSON de %s perdió la evidencia %q: %s", resultado, campo, contenido)
				}
			}
		})
	}

	contenido, err := json.Marshal(validacionRCParaPrueba())
	if err != nil {
		t.Fatalf("serializar validada: %v", err)
	}
	var campos map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &campos); err != nil {
		t.Fatalf("decodificar validada: %v", err)
	}
	for _, campo := range []string{"fecha_rc", "numero", "importe", "documento_ref"} {
		if _, existe := campos[campo]; !existe {
			t.Fatalf("el JSON validado perdió %q: %s", campo, contenido)
		}
	}
}

func TestAnalisisRRHHExigeCosteTrazableYCreditoSuficiente(t *testing.T) {
	pruebas := []struct {
		nombre    string
		modificar func(*AnalisisRRHH)
	}{
		{"fuente sin coste", func(a *AnalisisRRHH) {
			a.CostePrevisto = nil
		}},
		{"coste sin fuente", func(a *AnalisisRRHH) {
			a.FuenteCosteRef = ""
		}},
		{"coste cero", func(a *AnalisisRRHH) {
			a.CostePrevisto = importeParaPrueba(0)
		}},
		{"coste negativo", func(a *AnalisisRRHH) {
			a.CostePrevisto = importeParaPrueba(-1)
		}},
		{"moneda no admitida", func(a *AnalisisRRHH) {
			a.CostePrevisto = &Importe{Centimos: 1, Moneda: "USD"}
		}},
		{"coste no calculable", func(a *AnalisisRRHH) {
			a.CostePrevisto = importeParaPrueba(maximoCentimosCalculablesAnalisis + 1)
		}},
		{"RC insuficiente", func(a *AnalisisRRHH) {
			a.CostePrevisto = importeParaPrueba(a.ValidacionRC.Importe.Centimos + 1)
		}},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			analisis := analisisCompletoParaPrueba()
			prueba.modificar(&analisis)
			if analisis.Validar() == nil {
				t.Fatal("se aceptó un coste incoherente")
			}
		})
	}

	analisis := analisisCompletoParaPrueba()
	analisis.CostePrevisto = importeParaPrueba(analisis.ValidacionRC.Importe.Centimos)
	if err := analisis.Validar(); err != nil {
		t.Fatalf("coste exactamente igual a la RC: %v", err)
	}

	analisis.ValidacionRC.Importe.Centimos = maximoCentimosCalculablesAnalisis
	analisis.CostePrevisto = importeParaPrueba(maximoCentimosCalculablesAnalisis)
	if err := analisis.Validar(); err != nil {
		t.Fatalf("importe técnico máximo exacto: %v", err)
	}

	analisis = analisisCompletoParaPrueba()
	analisis.CostePrevisto = nil
	analisis.FuenteCosteRef = ""
	if err := analisis.Validar(); err != nil {
		t.Fatalf("coste aún no disponible y sin fuente residual: %v", err)
	}
}

func TestAnalisisRRHHClonaDefensivamenteElCoste(t *testing.T) {
	original := analisisCompletoParaPrueba()
	clon, err := original.Clonar()
	if err != nil {
		t.Fatalf("clonar: %v", err)
	}
	clon.CostePrevisto.Centimos++
	if clon.CostePrevisto == original.CostePrevisto ||
		clon.CostePrevisto.Centimos == original.CostePrevisto.Centimos {
		t.Fatal("el clon comparte el importe mutable con el original")
	}
	*clon.ValidacionRC.FechaRC = clon.ValidacionRC.FechaRC.AddDate(0, 0, 1)
	clon.ValidacionRC.Importe.Centimos++
	if clon.ValidacionRC.FechaRC == original.ValidacionRC.FechaRC ||
		clon.ValidacionRC.Importe == original.ValidacionRC.Importe ||
		clon.ValidacionRC.FechaRC.Equal(*original.ValidacionRC.FechaRC) ||
		clon.ValidacionRC.Importe.Centimos == original.ValidacionRC.Importe.Centimos {
		t.Fatal("el clon comparte punteros de la evidencia RC con el original")
	}

	invalido := original
	invalido.PorcentajeJornada = 0
	if _, err := invalido.Clonar(); err == nil {
		t.Fatal("se clonó un análisis inválido")
	}
}

func TestAnalisisRRHHRechazadoEsRegistrablePeroNoHabilitaAvance(t *testing.T) {
	analisis := analisisCompletoParaPrueba()
	prepararRCNegativa(&analisis.ValidacionRC, RCRechazada)
	if err := analisis.Validar(); err != nil {
		t.Fatalf("el rechazo motivado debe ser registrable: %v", err)
	}
	if analisis.HabilitaAvance() {
		t.Fatal("un análisis con RC rechazada habilitó el avance")
	}

	analisis.ValidacionRC.Resultado = RCNoRequerida
	if !analisis.HabilitaAvance() {
		t.Fatal("una RC no requerida y motivada no habilitó el avance")
	}

	analisis.ValidacionRC.Resultado = RCValidada
	if analisis.HabilitaAvance() {
		t.Fatal("datos residuales de rechazo convertidos en validada habilitaron el avance")
	}
}

func analisisCompletoParaPrueba() AnalisisRRHH {
	validacion := validacionRCParaPrueba()
	return AnalisisRRHH{
		ModalidadClave:    "sustitucion",
		CategoriaRef:      "categoria_trabajador_social",
		GrupoSubgrupo:     "A2",
		CausaClave:        "incapacidad_temporal",
		Periodo:           PeriodoPrevisto{Inicio: fechaAnalisis(2026, 8, 1), Fin: fechaAnalisis(2027, 3, 31)},
		PorcentajeJornada: JornadaCompletaDiezmilesimas,
		EntradaRCEsperada: VinculoEntradaRC{
			Referencia: validacion.EntradaRef, HuellaSHA256: validacion.HuellaEntradaSHA256,
		},
		ValidacionRC:   validacion,
		CostePrevisto:  importeParaPrueba(3_148_025),
		FuenteCosteRef: "tabla_retributiva_2026_v3",
		Observaciones:  "Estimación sujeta a los conceptos de la fuente autorizada.",
	}
}

func validacionRCParaPrueba() ValidacionRC {
	return ValidacionRC{
		Resultado:           RCValidada,
		EntradaRef:          "declaracion_rc_solicitud_01",
		HuellaEntradaSHA256: strings.Repeat("a", 64),
		FuenteRef:           "fuente_presupuestaria_01",
		ReciboRef:           "recibo_validacion_rc_01",
		ValidadaEn:          time.Date(2026, 7, 23, 8, 30, 0, 0, time.UTC),
		FechaRC:             fechaParaPrueba(2026, 7, 20),
		Numero:              "rc_2026_0001",
		Importe:             importeParaPrueba(3_245_000),
		DocumentoRef:        "documento_rc_01",
	}
}

func prepararRCNegativa(validacion *ValidacionRC, resultado ResultadoValidacionRC) {
	validacion.Resultado = resultado
	validacion.FechaRC = nil
	validacion.Numero = ""
	validacion.Importe = nil
	validacion.DocumentoRef = ""
	validacion.Motivo = "La fuente autoritativa no acredita una RC utilizable."
}

func importeParaPrueba(centimos int64) *Importe {
	return &Importe{Centimos: centimos, Moneda: "EUR"}
}

func fechaAnalisis(anyo int, mes time.Month, dia int) time.Time {
	return time.Date(anyo, mes, dia, 0, 0, 0, 0, time.UTC)
}

func fechaParaPrueba(anyo int, mes time.Month, dia int) *time.Time {
	fecha := fechaAnalisis(anyo, mes, dia)
	return &fecha
}
