package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

// circuitoRemisionPrueba marca el paso 2 del informe como el que habilita la
// remisión a Intervención, igual que el circuito de ejemplo.
type circuitoRemisionPrueba struct{ sinMarca bool }

func (c circuitoRemisionPrueba) CircuitoFirma(ctx context.Context) (domain.CircuitoFirma, error) {
	circuito, err := circuitoFirmaPrueba{}.CircuitoFirma(ctx)
	if !c.sinMarca {
		circuito.Documentos[0].Pasos[0].Habilita = "siguiente_paso"
		circuito.Documentos[0].Pasos[1].Habilita = domain.HabilitaRemisionIntervencion
	}
	return circuito, err
}

type registroFirmasFallidoPrueba struct{ registroFirmaPrueba }

func (registroFirmasFallidoPrueba) ConsultarFirmas(context.Context, string, string) ([]ports.FirmaRegistrada, error) {
	return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
}

func TestComprobarFirmaRemisionSigueElCircuito(t *testing.T) {
	ctx := context.Background()
	registro := &registroFirmaPrueba{}
	s, err := NuevoServicioFirmaDocumento(circuitoRemisionPrueba{}, registro, &autorizadorFirmaPrueba{}, verificadorPrueba{motivo: docports.MotivoFirmaVerificada})
	if err != nil {
		t.Fatal(err)
	}
	const org, exp = "organizacion:desarrollo:dipgra", "expediente:ct:001"
	if err := s.ComprobarFirmaRemision(ctx, org, exp); !errors.Is(err, ports.ErrFirmaRemisionPendiente) {
		t.Fatalf("sin firmas la remisión debe esperar: %v", err)
	}
	borrador := []byte("%PDF-1.7 borrador")
	if _, err := s.Firmar(ctx, solicitudFirmaPrueba(1, borrador, append(append([]byte{}, borrador...), '1'), "clave-firma-000000001")); err != nil {
		t.Fatal(err)
	}
	if err := s.ComprobarFirmaRemision(ctx, org, exp); !errors.Is(err, ports.ErrFirmaRemisionPendiente) {
		t.Fatalf("la firma del paso 1 no habilita la remisión: %v", err)
	}
	if _, err := s.Firmar(ctx, solicitudFirmaPrueba(2, borrador, append(append([]byte{}, borrador...), '2'), "clave-firma-000000002")); err != nil {
		t.Fatal(err)
	}
	if err := s.ComprobarFirmaRemision(ctx, org, exp); err != nil {
		t.Fatalf("con el paso habilitante firmado se puede remitir: %v", err)
	}

	sinMarca, err := NuevoServicioFirmaDocumento(circuitoRemisionPrueba{sinMarca: true}, &registroFirmaPrueba{}, &autorizadorFirmaPrueba{}, nil)
	if err != nil || sinMarca.ComprobarFirmaRemision(ctx, org, exp) != nil {
		t.Fatalf("un circuito sin paso habilitante no exige firma: %v", err)
	}
	fallido, err := NuevoServicioFirmaDocumento(circuitoRemisionPrueba{}, &registroFirmasFallidoPrueba{}, &autorizadorFirmaPrueba{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := fallido.ComprobarFirmaRemision(ctx, org, exp); err == nil || errors.Is(err, ports.ErrFirmaRemisionPendiente) {
		t.Fatalf("un registro caído no equivale a firmado ni a pendiente: %v", err)
	}
	sinCircuito, err := NuevoServicioFirmaDocumento(circuitoFirmaPrueba{err: errors.New("caído")}, &registroFirmaPrueba{}, &autorizadorFirmaPrueba{}, nil)
	if err != nil || !errors.Is(sinCircuito.ComprobarFirmaRemision(ctx, org, exp), ErrCircuitoFirmaNoDisponible) {
		t.Fatalf("sin circuito no se remite: %v", err)
	}
}

func TestExigirFirmaRemisionSeFijaUnaVez(t *testing.T) {
	var nulo *ServicioFiscalizaciones
	firmas, _ := NuevoServicioFirmaDocumento(circuitoRemisionPrueba{}, &registroFirmaPrueba{}, &autorizadorFirmaPrueba{}, nil)
	if !errors.Is(nulo.ExigirFirmaRemision(firmas), ErrServicioFiscalizacionesInvalido) {
		t.Fatal("servicio nulo admitido")
	}
	s := &ServicioFiscalizaciones{}
	var firmasNulas *ServicioFirmaDocumento
	if !errors.Is(s.ExigirFirmaRemision(firmasNulas), ErrServicioFiscalizacionesInvalido) {
		t.Fatal("comprobador nulo admitido")
	}
	if s.ExigirFirmaRemision(firmas) != nil || !errors.Is(s.ExigirFirmaRemision(firmas), ErrServicioFiscalizacionesInvalido) {
		t.Fatal("el comprobador se fija una sola vez")
	}
}
