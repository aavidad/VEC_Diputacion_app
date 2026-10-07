package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// Sin crédito no se ofrece: tras autorizar la lectura, la propuesta de
// cobertura no se construye y el motivo llega a la frontera sin gastar la
// preparación ni consultar el gobierno de la operación.
func TestProponerCoberturaSinCreditoExplicaElMotivoSinEfectos(t *testing.T) {
	escenario := nuevoEscenarioPresentacionCobertura(t, viasPresentacionCoberturaPrueba(2))
	expediente := escenario.analisis.expediente.Clonar()
	expediente.Analisis.ValidacionRC.Resultado = domain.RCRechazada
	if expediente.Validar() != nil {
		t.Fatal("el expediente de prueba con la retención rechazada debe ser íntegro")
	}
	escenario.analisis.expediente = expediente
	_, err := escenario.servicio.Proponer(context.Background(), escenario.solicitud)
	motivo, ok := domain.MotivoSinCreditoDe(err)
	if !errors.Is(err, ErrPresentacionPropuestaCoberturaEstadoNoAdmite) ||
		!ok || motivo != domain.SinCreditoRetencionRechazada {
		t.Fatalf("sin crédito debía explicar el motivo: %v", err)
	}
	if escenario.accesos.total() == 0 || escenario.gobierno.total() != 0 ||
		escenario.global.generador.llamadas() != 0 {
		t.Fatal("sin crédito no debe consultar el gobierno ni preparar la propuesta")
	}
	exigirCeroConsumoPreparacionGlobal(t, escenario.global)
}

// La decisión aún no ha autorizado el expediente cuando lee la instantánea:
// no revela el motivo de crédito y responde como antes, sin disponibilidad.
func TestConfirmacionSinCreditoNoRevelaElMotivoAntesDeAutorizar(t *testing.T) {
	servicio := &ServicioConfirmacionDecisionCobertura{}
	err := servicio.errorDependencia(context.Background(),
		errors.Join(errors.New("instantánea"), domain.NuevoErrorSinCredito(domain.SinCreditoRetencionRechazada)))
	if _, ok := domain.MotivoSinCreditoDe(err); ok || !errors.Is(err, ErrConfirmacionDecisionCoberturaNoDisponible) {
		t.Fatalf("la decisión reveló el motivo de crédito: %v", err)
	}
	// Una causa ajena no se convierte en motivo de crédito.
	if _, ok := errorSinCreditoCobertura(ErrConfirmacionDecisionCoberturaEnConflicto, errors.New("otra")); ok {
		t.Fatal("error ajeno tratado como falta de crédito")
	}
}

type politicaCreditoPrueba struct {
	politica domain.PoliticaCreditoOferta
	err      error
}

func (p politicaCreditoPrueba) PoliticaCreditoOferta(context.Context) (domain.PoliticaCreditoOferta, error) {
	return p.politica, p.err
}

func escenarioPartidasSinCoste(t *testing.T) *escenarioPresentacionCobertura {
	t.Helper()
	escenario := nuevoEscenarioPresentacionCobertura(t, viasPresentacionCoberturaPrueba(2))
	expediente := escenario.analisis.expediente.Clonar()
	expediente.Analisis.CostePrevisto, expediente.Analisis.FuenteCosteRef = nil, ""
	if expediente.Validar() != nil || expediente.Analisis.ValidacionRC.Resultado != domain.RCNoRequerida {
		t.Fatal("el expediente de prueba debe ser íntegro y sin retención")
	}
	escenario.analisis.expediente = expediente
	return escenario
}

// Decisión de RRHH del 02/10: sin retención, la constancia de las partidas
// va con el coste aproximado. Sin catálogo rige; el catálogo la puede quitar.
func TestProponerCoberturaPartidasSinCosteSegunLaPolitica(t *testing.T) {
	escenario := escenarioPartidasSinCoste(t)
	_, err := escenario.servicio.Proponer(context.Background(), escenario.solicitud)
	if motivo, ok := domain.MotivoSinCreditoDe(err); !ok || motivo != domain.SinCreditoPartidasSinCoste ||
		!errors.Is(err, ErrPresentacionPropuestaCoberturaEstadoNoAdmite) || escenario.gobierno.total() != 0 {
		t.Fatalf("sin catálogo debía exigir el coste: %v", err)
	}

	desactivada := escenarioPartidasSinCoste(t)
	if err := desactivada.servicio.ConfigurarPoliticaCredito(politicaCreditoPrueba{}); err != nil {
		t.Fatal(err)
	}
	if err := desactivada.servicio.ConfigurarPoliticaCredito(politicaCreditoPrueba{}); !errors.Is(err, ErrPoliticaCreditoCoberturaInvalida) {
		t.Fatalf("la política solo se fija una vez: %v", err)
	}
	if _, err := desactivada.servicio.Proponer(context.Background(), desactivada.solicitud); err != nil {
		t.Fatalf("con la exigencia desactivada basta la constancia: %v", err)
	}

	caida := escenarioPartidasSinCoste(t)
	if err := caida.servicio.ConfigurarPoliticaCredito(politicaCreditoPrueba{err: errors.New("catálogo caído")}); err != nil {
		t.Fatal(err)
	}
	_, err = caida.servicio.Proponer(context.Background(), caida.solicitud)
	if !errors.Is(err, ErrPresentacionPropuestaCoberturaNoDisponible) || caida.gobierno.total() != 0 {
		t.Fatalf("un catálogo no disponible no debe tomarse como «no exige»: %v", err)
	}
}

func TestDecisionPartidasSinCosteRechazaSinDecirElMotivo(t *testing.T) {
	escenario := escenarioPartidasSinCoste(t)
	servicio := &ServicioConfirmacionDecisionCobertura{}
	err := servicio.comprobarPoliticaCredito(context.Background(), escenario.analisis.expediente)
	if _, ok := domain.MotivoSinCreditoDe(err); ok || !errors.Is(err, ErrConfirmacionDecisionCoberturaEnConflicto) {
		t.Fatalf("la decisión debía rechazar sin motivo: %v", err)
	}
	if err := servicio.ConfigurarPoliticaCredito(politicaCreditoPrueba{err: errors.New("caído")}); err != nil {
		t.Fatal(err)
	}
	if err := servicio.comprobarPoliticaCredito(context.Background(), escenario.analisis.expediente); !errors.Is(err, ErrConfirmacionDecisionCoberturaNoDisponible) {
		t.Fatalf("catálogo caído: %v", err)
	}
}

// La exigencia del coste solo alcanza a la decisión inicial: una vía ya
// decidida se puede rectificar, y con la retención validada el catálogo no
// se consulta (si está caído no bloquea).
func TestPoliticaCreditoSoloEnLaDecisionInicialYSoloSiImporta(t *testing.T) {
	escenario := escenarioPartidasSinCoste(t)
	caida := politicaCreditoPrueba{err: errors.New("catálogo caído")}
	decidido := escenario.analisis.expediente.Clonar()
	decidido.ViaCobertura = &domain.DecisionViaCobertura{ViaClave: "bolsa_vigente"}
	if motivo, err := motivoSegunPoliticaCredito(context.Background(), caida, decidido); motivo != "" || err != nil {
		t.Fatalf("una vía ya decidida no debía volver a pedir el coste: %q %v", motivo, err)
	}
	validada := nuevoEscenarioPresentacionCobertura(t, viasPresentacionCoberturaPrueba(2)).analisis.expediente.Clonar()
	validada.Analisis.ValidacionRC.Resultado = domain.RCRechazada
	if motivo, err := motivoSegunPoliticaCredito(context.Background(), caida, validada); motivo != domain.SinCreditoRetencionRechazada || err != nil {
		t.Fatalf("la regla base no depende del catálogo: %q %v", motivo, err)
	}
}

func TestCircuitoNuevoNoPresentaOfertaConPartidasSinAcreditar(t *testing.T) {
	escenario := escenarioPartidasSinCoste(t)
	expediente := escenario.analisis.expediente.Clonar()
	expediente.Analisis.CostePrevisto = &domain.Importe{Centimos: 3_148_025, Moneda: "EUR"}
	expediente.Analisis.FuenteCosteRef = "tabla:retributiva-sintetica-2026"
	expediente.Circuito = &domain.CircuitoAdministrativo{}
	motivo, err := motivoSegunPoliticaCredito(context.Background(), nil, expediente)
	if err != nil || motivo != domain.SinCreditoPartidasNoAcreditadas {
		t.Fatalf("el motivo solo no acredita las partidas: %q %v", motivo, err)
	}
}
