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

type fuenteResultadosPrueba struct {
	politica domain.PoliticaResultadosFiscalizacion
	err      error
}

func (f fuenteResultadosPrueba) ResultadosFiscalizacion(context.Context) (domain.PoliticaResultadosFiscalizacion, error) {
	return f.politica, f.err
}

type comprobadorFirmaPrueba struct {
	err      error
	llamadas int
}

func (c *comprobadorFirmaPrueba) ComprobarFirmaRemision(context.Context, string, string) error {
	c.llamadas++
	return c.err
}

// Antes del efecto: el catálogo decide qué resultados se admiten y el
// circuito de firma si se puede remitir; sin ninguno de los dos, nada cambia.
func TestComprobarCatalogoYFirmaAntesDelEfecto(t *testing.T) {
	ctx := context.Background()
	expedientePrueba := domain.Expediente{OrganizacionRef: "organizacion:desarrollo:dipgra", Referencia: "expediente:ct:001"}
	if err := (&ServicioFiscalizaciones{}).comprobarCatalogoYFirma(ctx, expedientePrueba, domain.FiscalizacionFavorable); err != nil {
		t.Fatalf("sin catálogo ni firmas rige la conducta de siempre: %v", err)
	}
	sinObservaciones, err := domain.NuevaPoliticaResultadosFiscalizacion(
		[]domain.ResultadoFiscalizacion{domain.FiscalizacionFavorable, domain.FiscalizacionDesfavorable},
		map[domain.ResultadoFiscalizacion]domain.EfectoResultadoFiscalizacion{
			domain.FiscalizacionFavorable:    domain.EfectoFiscalizacionContinua,
			domain.FiscalizacionDesfavorable: domain.EfectoFiscalizacionVuelveUnidadGestora,
		})
	if err != nil {
		t.Fatal(err)
	}
	firma := &comprobadorFirmaPrueba{}
	s := &ServicioFiscalizaciones{}
	if s.GobernarResultados(fuenteResultadosPrueba{politica: sinObservaciones}) != nil || s.ExigirFirmaRemision(firma) != nil {
		t.Fatal("composición")
	}
	if err := s.comprobarCatalogoYFirma(ctx, expedientePrueba, domain.FiscalizacionFavorableConObservaciones); !errors.Is(err, ports.ErrResultadoFiscalizacionNoAdmitido) ||
		!errors.Is(err, domain.ErrDatoInvalido) || firma.llamadas != 0 {
		t.Fatalf("resultado retirado del catálogo admitido: %v", err)
	}
	if err := s.comprobarCatalogoYFirma(ctx, expedientePrueba, domain.FiscalizacionFavorable); err != nil || firma.llamadas != 1 {
		t.Fatalf("resultado admitido y firmado: %v", err)
	}
	firma.err = ports.ErrFirmaRemisionPendiente
	if err := s.comprobarCatalogoYFirma(ctx, expedientePrueba, domain.FiscalizacionDesfavorable); !errors.Is(err, ports.ErrFirmaRemisionPendiente) {
		t.Fatalf("sin firma no se remite: %v", err)
	}
	firma.err = ports.ErrRegistroFirmaDocumentoNoDisponible
	if err := s.comprobarCatalogoYFirma(ctx, expedientePrueba, domain.FiscalizacionFavorable); err == nil || errors.Is(err, ports.ErrFirmaRemisionPendiente) {
		t.Fatalf("registro caído: %v", err)
	}
	caido := &ServicioFiscalizaciones{}
	_ = caido.GobernarResultados(fuenteResultadosPrueba{err: errors.New("catálogo ilegible")})
	if err := caido.comprobarCatalogoYFirma(ctx, expedientePrueba, domain.FiscalizacionFavorable); err == nil {
		t.Fatal("un catálogo ilegible no admite nada")
	}
	if caido.GobernarResultados(fuenteResultadosPrueba{}) == nil {
		t.Fatal("la política se fija una sola vez")
	}
}
