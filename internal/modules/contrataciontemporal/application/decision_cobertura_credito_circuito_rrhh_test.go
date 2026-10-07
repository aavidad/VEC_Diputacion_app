package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type fuenteCreditoCircuitoPrueba struct {
	definicion domain.DefinicionCircuitoRRHH
	llamadas   int
	alterar    func(*ports.EvidenciaCreditoCircuitoRRHH)
}

func (f *fuenteCreditoCircuitoPrueba) AcreditarCreditoCircuitoRRHH(
	_ context.Context, s ports.SolicitudEvidenciaCreditoCircuitoRRHH,
) (ports.EvidenciaCreditoCircuitoRRHH, error) {
	f.llamadas++
	e := ports.EvidenciaCreditoCircuitoRRHH{
		Solicitud: s, Definicion: f.definicion, DocumentoVersion: 2,
		HuellaDocumentoSHA256: strings.Repeat("c", 64),
	}
	if f.alterar != nil {
		f.alterar(&e)
	}
	return e, nil
}

func TestConfirmacionCircuitoExigeFuenteDocumentalLigada(t *testing.T) {
	base := expedientePresentacionCoberturaPrueba(t, time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC))
	fecha := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	importe := domain.Importe{Centimos: 4_000_000, Moneda: "EUR"}
	base.Analisis.ValidacionRC.Resultado = domain.RCValidada
	base.Analisis.ValidacionRC.Motivo = ""
	base.Analisis.ValidacionRC.FechaRC = &fecha
	base.Analisis.ValidacionRC.Numero = "rc:prueba:credito"
	base.Analisis.ValidacionRC.Importe = &importe
	base.Analisis.ValidacionRC.DocumentoRef = "documento:prueba:credito"
	definicion, err := domain.NuevaDefinicionCircuitoRRHH("flujo:ct:rrhh:credito-aplicacion", 2, "analisis_rrhh", []domain.TransicionCircuitoRRHH{{
		Clave: "credito_comprobado", Tipo: domain.HitoCreditoComprobado,
		Origen: "analisis_rrhh", Destino: "oferta", RequiereDocumento: true,
		PerfilClave: "tecnico_rrhh",
	}})
	if err != nil {
		t.Fatal(err)
	}
	base.Flujo = definicion.Flujo
	circuito, err := domain.NuevoCircuitoAdministrativo(definicion)
	if err != nil {
		t.Fatal(err)
	}
	base.Circuito = &circuito
	if err := base.Validar(); err != nil {
		t.Fatal(err)
	}
	servicio := &ServicioConfirmacionDecisionCobertura{}
	contexto := nuevoEscenarioPresentacionCobertura(t, viasPresentacionCoberturaPrueba(2)).contextos.contexto
	if _, err := servicio.acreditarCreditoCircuito(context.Background(), base, domain.DecisionCoberturaInicial, contexto); !errors.Is(err, ErrConfirmacionDecisionCoberturaNoDisponible) {
		t.Fatalf("sin fuente debía cerrarse: %v", err)
	}
	fuente := &fuenteCreditoCircuitoPrueba{definicion: definicion}
	if err := servicio.ConfigurarFuenteCreditoCircuitoRRHH(fuente); err != nil {
		t.Fatal(err)
	}
	if err := servicio.ConfigurarFuenteCreditoCircuitoRRHH(fuente); !errors.Is(err, ErrFuenteCreditoCircuitoRRHHInvalida) {
		t.Fatalf("la fuente se sustituyó: %v", err)
	}
	evidencias, err := servicio.acreditarCreditoCircuito(context.Background(), base, domain.DecisionCoberturaInicial, contexto)
	if err != nil || len(evidencias) != 1 || fuente.llamadas != 1 || evidencias[0].Solicitud.DocumentoRef != base.Analisis.ValidacionRC.DocumentoRef {
		t.Fatalf("evidencia documental no ligada: %v", err)
	}
	if _, err := servicio.acreditarCreditoCircuito(context.Background(), base, domain.DecisionCoberturaRectificacion, contexto); err != nil || fuente.llamadas != 1 {
		t.Fatalf("la rectificación volvió a consultar crédito: %v", err)
	}
	fuente.alterar = func(e *ports.EvidenciaCreditoCircuitoRRHH) { e.Solicitud.DocumentoRef = "documento:otro:credito" }
	if _, err := servicio.acreditarCreditoCircuito(context.Background(), base, domain.DecisionCoberturaInicial, contexto); !errors.Is(err, ErrConfirmacionDecisionCoberturaNoConfiable) {
		t.Fatalf("se aceptó fuente con otro documento: %v", err)
	}
}
