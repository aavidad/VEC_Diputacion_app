package domain

import (
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
	if err := validacionRCParaPrueba().Validar(); err != nil {
		t.Fatalf("RC validada: %v", err)
	}

	for _, resultado := range []ResultadoValidacionRC{RCNoValidada, RCNoRequerida} {
		validacion := validacionRCParaPrueba()
		validacion.Resultado = resultado
		validacion.Numero = ""
		validacion.Importe = Importe{}
		validacion.DocumentoRef = ""
		validacion.Motivo = "Decisión motivada conforme a la fuente presupuestaria."
		if err := validacion.Validar(); err != nil {
			t.Fatalf("%s con evidencia coherente: %v", resultado, err)
		}
	}
}

func TestValidacionRCRechazaEvidenciaIncompletaOResidual(t *testing.T) {
	pruebas := []struct {
		nombre    string
		modificar func(*ValidacionRC)
	}{
		{"resultado desconocido", func(v *ValidacionRC) { v.Resultado = "desconocida" }},
		{"fuente ausente", func(v *ValidacionRC) { v.FuenteRef = "" }},
		{"recibo ausente", func(v *ValidacionRC) { v.ReciboRef = "" }},
		{"instante ausente", func(v *ValidacionRC) { v.ValidadaEn = time.Time{} }},
		{"instante no canónico", func(v *ValidacionRC) {
			v.ValidadaEn = v.ValidadaEn.Add(time.Nanosecond)
		}},
		{"validada sin número", func(v *ValidacionRC) { v.Numero = "" }},
		{"validada sin importe", func(v *ValidacionRC) { v.Importe = Importe{} }},
		{"validada sin documento", func(v *ValidacionRC) { v.DocumentoRef = "" }},
		{"importe no calculable", func(v *ValidacionRC) {
			v.Importe.Centimos = maximoCentimosCalculablesAnalisis + 1
		}},
		{"no validada sin motivo", func(v *ValidacionRC) {
			prepararRCNegativa(v, RCNoValidada)
			v.Motivo = ""
		}},
		{"no requerida con número residual", func(v *ValidacionRC) {
			prepararRCNegativa(v, RCNoRequerida)
			v.Numero = "rc_residual_01"
		}},
		{"no requerida con importe residual", func(v *ValidacionRC) {
			prepararRCNegativa(v, RCNoRequerida)
			v.Importe = Importe{Centimos: 1, Moneda: "EUR"}
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

	invalido := original
	invalido.PorcentajeJornada = 0
	if _, err := invalido.Clonar(); err == nil {
		t.Fatal("se clonó un análisis inválido")
	}
}

func analisisCompletoParaPrueba() AnalisisRRHH {
	return AnalisisRRHH{
		ModalidadClave:    "sustitucion",
		CategoriaRef:      "categoria_trabajador_social",
		GrupoSubgrupo:     "A2",
		CausaClave:        "incapacidad_temporal",
		Periodo:           PeriodoPrevisto{Inicio: fechaAnalisis(2026, 8, 1), Fin: fechaAnalisis(2027, 3, 31)},
		PorcentajeJornada: JornadaCompletaDiezmilesimas,
		ValidacionRC:      validacionRCParaPrueba(),
		CostePrevisto:     importeParaPrueba(3_148_025),
		FuenteCosteRef:    "tabla_retributiva_2026_v3",
		Observaciones:     "Estimación sujeta a los conceptos de la fuente autorizada.",
	}
}

func validacionRCParaPrueba() ValidacionRC {
	return ValidacionRC{
		Resultado:    RCValidada,
		FuenteRef:    "fuente_presupuestaria_01",
		ReciboRef:    "recibo_validacion_rc_01",
		ValidadaEn:   time.Date(2026, 7, 23, 8, 30, 0, 0, time.UTC),
		Numero:       "rc_2026_0001",
		Importe:      Importe{Centimos: 3_245_000, Moneda: "EUR"},
		DocumentoRef: "documento_rc_01",
	}
}

func prepararRCNegativa(validacion *ValidacionRC, resultado ResultadoValidacionRC) {
	validacion.Resultado = resultado
	validacion.Numero = ""
	validacion.Importe = Importe{}
	validacion.DocumentoRef = ""
	validacion.Motivo = "La fuente autoritativa no acredita una RC utilizable."
}

func importeParaPrueba(centimos int64) *Importe {
	return &Importe{Centimos: centimos, Moneda: "EUR"}
}

func fechaAnalisis(anyo int, mes time.Month, dia int) time.Time {
	return time.Date(anyo, mes, dia, 0, 0, 0, 0, time.UTC)
}
