package domain

import (
	"errors"
	"testing"
	"time"
)

func TestSinCreditoNoSePasaALaOferta(t *testing.T) {
	pruebas := []struct {
		nombre    string
		resultado ResultadoValidacionRC
		sinCoste  bool
		motivo    MotivoSinCredito
	}{
		{"retención validada", RCValidada, false, ""},
		{"constancia del estado de las partidas", RCNoRequerida, false, ""},
		{"constancia de las partidas sin coste aproximado", RCNoRequerida, true, ""},
		{"retención rechazada", RCRechazada, false, SinCreditoRetencionRechazada},
		{"retención rechazada sin coste", RCRechazada, true, SinCreditoRetencionRechazada},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			analisis := analisisValido()
			if prueba.resultado != RCValidada {
				prepararRCNegativa(&analisis.ValidacionRC, prueba.resultado)
			}
			if prueba.sinCoste {
				analisis.CostePrevisto, analisis.FuenteCosteRef = nil, ""
			}
			conAnalisis, err := expedienteValido(t).RegistrarAnalisis(1, analisis,
				actuacion("analisis.validado", "gestion_bolsa", instanteBase.Add(time.Minute)))
			if err != nil {
				t.Fatalf("registrar análisis: %v", err)
			}
			if got := conAnalisis.Analisis.MotivoSinCreditoParaOferta(); got != prueba.motivo {
				t.Fatalf("motivo = %q; se esperaba %q", got, prueba.motivo)
			}
			_, err = conAnalisis.RegistrarViaCobertura(2, decisionValida(),
				actuacion("cobertura.decidida", "asignacion_unidad", instanteBase.Add(2*time.Minute)))
			if prueba.motivo == "" {
				if err != nil {
					t.Fatalf("con crédito debía poder ofrecerse: %v", err)
				}
				return
			}
			motivo, ok := MotivoSinCreditoDe(err)
			if !errors.Is(err, ErrTransicionInvalida) || !errors.Is(err, ErrSinCreditoParaOferta) ||
				!ok || motivo != prueba.motivo {
				t.Fatalf("sin crédito debía bloquear con motivo %q: %v", prueba.motivo, err)
			}
		})
	}
}

func TestSinAnalisisElMotivoEsAnalisisPendiente(t *testing.T) {
	expediente := expedienteValido(t)
	if got := expediente.Analisis.MotivoSinCreditoParaOferta(); got != SinCreditoAnalisisPendiente {
		t.Fatalf("motivo = %q", got)
	}
	if err := expediente.ErrorSinCreditoParaOferta(); !errors.Is(err, ErrSinCreditoParaOferta) {
		t.Fatalf("error = %v", err)
	}
	if _, ok := MotivoSinCreditoDe(errors.New("otro")); ok {
		t.Fatal("un error ajeno no lleva motivo de crédito")
	}
}

func analisisRegistradoPrueba(t *testing.T, preparar func(*AnalisisRRHH)) *AnalisisRRHH {
	t.Helper()
	analisis := analisisValido()
	preparar(&analisis)
	expediente, err := expedienteValido(t).RegistrarAnalisis(1, analisis,
		actuacion("analisis.validado", "gestion_bolsa", instanteBase.Add(time.Minute)))
	if err != nil {
		t.Fatalf("registrar análisis: %v", err)
	}
	return expediente.Analisis
}

func TestPoliticaDeCreditoExigeElCosteConLasPartidas(t *testing.T) {
	exige := PoliticaCreditoOferta{ExigeCosteConPartidas: true}
	partidas := func(a *AnalisisRRHH) { prepararRCNegativa(&a.ValidacionRC, RCNoRequerida) }
	sinCoste := func(a *AnalisisRRHH) { a.CostePrevisto, a.FuenteCosteRef = nil, "" }
	if got := analisisRegistradoPrueba(t, partidas).MotivoSinCreditoSegunPolitica(exige); got != "" {
		t.Fatalf("con coste aproximado no debía bloquear: %q", got)
	}
	sinCostePartidas := analisisRegistradoPrueba(t, func(a *AnalisisRRHH) { partidas(a); sinCoste(a) })
	if got := sinCostePartidas.MotivoSinCreditoSegunPolitica(exige); got != SinCreditoPartidasSinCoste {
		t.Fatalf("sin coste aproximado: %q", got)
	}
	if got := sinCostePartidas.MotivoSinCreditoSegunPolitica(PoliticaCreditoOferta{}); got != "" {
		t.Fatalf("con la exigencia desactivada basta la constancia: %q", got)
	}
	if got := analisisRegistradoPrueba(t, sinCoste).MotivoSinCreditoSegunPolitica(exige); got != "" {
		t.Fatalf("la retención validada no necesita el coste: %q", got)
	}
	rechazada := analisisRegistradoPrueba(t, func(a *AnalisisRRHH) { prepararRCNegativa(&a.ValidacionRC, RCRechazada) })
	if got := rechazada.MotivoSinCreditoSegunPolitica(PoliticaCreditoOferta{}); got != SinCreditoRetencionRechazada {
		t.Fatalf("la política no relaja la regla base: %q", got)
	}
}
