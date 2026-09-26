package application

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

type fuenteInformeTrasSubsanacionPrueba struct {
	politica domain.PoliticaInformeTrasSubsanacion
	err      error
}

func (f fuenteInformeTrasSubsanacionPrueba) InformeTrasSubsanacion(context.Context) (domain.PoliticaInformeTrasSubsanacion, error) {
	return f.politica, f.err
}

type rondaInformePrueba struct{ inicio uint64 }

func (r *rondaInformePrueba) InicioRondaInformeNuevo(context.Context, string, string) (uint64, error) {
	return r.inicio, nil
}

// Agregados escritos por PostgreSQL 18 en el ensayo de CT123 (v7 subsanado,
// v8 con informe nuevo, v9 fiscalizado favorable).
func agregadoCT123Prueba(t *testing.T, version string) domain.Expediente {
	t.Helper()
	contenido, err := os.ReadFile("../domain/testdata/expediente_informe_nuevo_v" + version + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var e domain.Expediente
	if err := json.Unmarshal(contenido, &e); err != nil || e.Validar() != nil {
		t.Fatalf("agregado v%s: %v", version, err)
	}
	return e
}

func firmaRondaAplicacionPrueba(orden int, version uint64, original []byte, clave string) SolicitudFirmaDocumento {
	s := solicitudFirmaPrueba(orden, original, append(append([]byte{}, original...), byte('0'+orden)), clave)
	s.VersionExpediente = version
	return s
}

// Recorrido de la duda 5 con el catálogo que exige informe nuevo:
// desfavorable → subsanación → la nueva fiscalización espera el informe
// nuevo → informe nuevo → su firma (segunda ronda) → remisión → favorable.
func TestInformeNuevoTrasSubsanacionRecorrido(t *testing.T) {
	ctx := context.Background()
	exige := fuenteInformeTrasSubsanacionPrueba{politica: domain.PoliticaInformeTrasSubsanacion{ExigeInformeNuevo: true, DocumentoFirma: "informe_definitivo"}}
	subsanado, conInforme, fiscalizado := agregadoCT123Prueba(t, "7"), agregadoCT123Prueba(t, "8"), agregadoCT123Prueba(t, "9")

	ronda := &rondaInformePrueba{}
	registro := &registroFirmaPrueba{}
	firmas, err := NuevoServicioFirmaDocumento(circuitoRemisionPrueba{}, registro, &autorizadorFirmaPrueba{}, verificadorPrueba{motivo: docports.MotivoFirmaVerificada})
	if err != nil || firmas.AbrirRondaInformeNuevo(exige, ronda) != nil {
		t.Fatalf("composición de firmas: %v", err)
	}
	// Primera ronda: el informe inicial quedó firmado antes de la remisión.
	borrador := []byte("%PDF-1.7 informe inicial")
	for orden := 1; orden <= 2; orden++ {
		if _, err := firmas.Firmar(ctx, firmaRondaAplicacionPrueba(orden, 5, borrador, "clave-ronda-uno-00"+string(rune('0'+orden)))); err != nil {
			t.Fatalf("primera ronda paso %d: %v", orden, err)
		}
	}

	fiscalizaciones := &ServicioFiscalizaciones{}
	if fiscalizaciones.GobernarInformeTrasSubsanacion(exige) != nil || fiscalizaciones.ExigirFirmaRemision(firmas) != nil {
		t.Fatal("composición de fiscalización")
	}
	// Tras subsanar, la nueva fiscalización sin informe nuevo se rechaza.
	if err := fiscalizaciones.comprobarCatalogoYFirma(ctx, subsanado, domain.FiscalizacionFavorable); !errors.Is(err, ports.ErrInformeNuevoPendiente) {
		t.Fatalf("nueva fiscalización sin informe nuevo: %v", err)
	}
	informes := &ServicioInformesJuridicos{}
	if err := informes.comprobarInformeNuevoPrevisto(ctx, subsanado); !errors.Is(err, ports.ErrInformeNuevoNoPrevisto) {
		t.Fatalf("sin catálogo compuesto no cabe informe nuevo: %v", err)
	}
	if informes.GobernarInformeTrasSubsanacion(exige) != nil || informes.comprobarInformeNuevoPrevisto(ctx, subsanado) != nil {
		t.Fatal("con el catálogo se admite el informe nuevo tras subsanar")
	}
	// Emitido el informe nuevo (v8), su firma abre la segunda ronda.
	ronda.inicio = conInforme.InformeJuridico.ActuacionRegistro.VersionExpediente
	if err := informes.comprobarInformeNuevoPrevisto(ctx, conInforme); !errors.Is(err, ports.ErrInformeNuevoNoPrevisto) {
		t.Fatalf("un segundo informe nuevo para el mismo reparo: %v", err)
	}
	if err := fiscalizaciones.comprobarCatalogoYFirma(ctx, conInforme, domain.FiscalizacionFavorable); !errors.Is(err, ports.ErrFirmaRemisionPendiente) {
		t.Fatalf("sin firmar el informe nuevo no se remite: %v", err)
	}
	nuevo := []byte("%PDF-1.7 informe nuevo")
	if _, err := firmas.Firmar(ctx, firmaRondaAplicacionPrueba(2, 8, nuevo, "clave-ronda-dos-002")); !errors.Is(err, ErrPasoFirmaNoPendiente) {
		t.Fatalf("la segunda ronda empieza por el paso 1: %v", err)
	}
	for orden := 1; orden <= 2; orden++ {
		if _, err := firmas.Firmar(ctx, firmaRondaAplicacionPrueba(orden, 8, nuevo, "clave-ronda-dos-00"+string(rune('0'+orden)))); err != nil {
			t.Fatalf("segunda ronda paso %d: %v", orden, err)
		}
	}
	estado, err := firmas.Consultar(ctx, "organizacion:desarrollo:dipgra", "expediente:ct:001")
	if err != nil || !estado.Documentos[0].Completo || estado.Documentos[0].EventosRondaAnterior != 2 {
		t.Fatalf("segunda ronda completa: %+v %v", estado.Documentos, err)
	}
	// Firmado y remitido: la nueva fiscalización favorable sigue adelante.
	if err := fiscalizaciones.comprobarCatalogoYFirma(ctx, conInforme, domain.FiscalizacionFavorable); err != nil {
		t.Fatalf("con informe nuevo firmado se fiscaliza: %v", err)
	}
	if fiscalizado.FaseActual != domain.FaseFiscalizacion || fiscalizado.Fiscalizacion.InformeJuridicoRef != conInforme.InformeJuridico.InformeRef ||
		informes.comprobarInformeNuevoPrevisto(ctx, fiscalizado) == nil {
		t.Fatal("la fiscalización favorable continúa con el informe nuevo y ya no admite otro")
	}
}

// Sin catálogo, o con uno que no exige informe nuevo, la nueva fiscalización
// tras subsanar se hace con el mismo informe, como siempre.
func TestInformeNuevoTrasSubsanacionSegunCatalogo(t *testing.T) {
	ctx := context.Background()
	subsanado := agregadoCT123Prueba(t, "7")
	if err := (&ServicioFiscalizaciones{}).comprobarCatalogoYFirma(ctx, subsanado, domain.FiscalizacionFavorable); err != nil {
		t.Fatalf("sin catálogo: %v", err)
	}
	noExige := &ServicioFiscalizaciones{}
	_ = noExige.GobernarInformeTrasSubsanacion(fuenteInformeTrasSubsanacionPrueba{})
	if err := noExige.comprobarCatalogoYFirma(ctx, subsanado, domain.FiscalizacionFavorable); err != nil {
		t.Fatalf("catálogo sin informe nuevo: %v", err)
	}
	informes := &ServicioInformesJuridicos{}
	_ = informes.GobernarInformeTrasSubsanacion(fuenteInformeTrasSubsanacionPrueba{})
	if !errors.Is(informes.comprobarInformeNuevoPrevisto(ctx, subsanado), ports.ErrInformeNuevoNoPrevisto) {
		t.Fatal("un catálogo que no exige informe nuevo no lo admite")
	}
	caido := &ServicioFiscalizaciones{}
	_ = caido.GobernarInformeTrasSubsanacion(fuenteInformeTrasSubsanacionPrueba{err: errors.New("catálogo ilegible")})
	if err := caido.comprobarCatalogoYFirma(ctx, subsanado, domain.FiscalizacionFavorable); err == nil || errors.Is(err, ports.ErrInformeNuevoPendiente) {
		t.Fatalf("un catálogo ilegible no deja fiscalizar: %v", err)
	}
	if caido.GobernarInformeTrasSubsanacion(fuenteInformeTrasSubsanacionPrueba{}) == nil ||
		informes.GobernarInformeTrasSubsanacion(fuenteInformeTrasSubsanacionPrueba{}) == nil {
		t.Fatal("la política se fija una sola vez")
	}
	invalida := fuenteInformeTrasSubsanacionPrueba{politica: domain.PoliticaInformeTrasSubsanacion{DocumentoFirma: "informe_definitivo"}}
	if invalida.politica.Validar() == nil {
		t.Fatal("un documento de firma sin exigir informe nuevo es incoherente")
	}
}
