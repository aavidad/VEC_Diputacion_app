package bootstrap

import (
	"errors"
	"net/http"
	"testing"
	"time"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	calendariosdomain "vec-diputacion-granada/internal/modules/calendarios/domain"
	calendariosports "vec-diputacion-granada/internal/modules/calendarios/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

func TestPlazoOfertaSinCatalogoNoSeInventa(t *testing.T) {
	calculadora := &calculadoraPlazoOfertaDesarrollo{}
	calculadora.fijar(nil)
	if _, _, err := calculadora.PlazoDisposicion(t.Context(), time.Now()); !errors.Is(err, puertosbolsa.ErrPlazoOfertaNoConfigurado) {
		t.Fatalf("sin catálogo debe negarse: %v", err)
	}
}

func TestPlazoOfertaUsaLaReglaB10DelCatalogo(t *testing.T) {
	ultimo, err := calendariosdomain.ParsearFechaCivil("2026-09-30")
	if err != nil {
		t.Fatal(err)
	}
	vence := time.Date(2026, 9, 30, 22, 0, 0, 0, time.UTC)
	calendarios := &consultaCalendariosReglasPrueba{resultado: calendariosports.ResultadoCalculoPlazo{ResultadoPlazo: calendariosdomain.ResultadoPlazo{Vencimiento: ultimo, VenceAntesDe: vence}}}
	compuestas, err := nuevasReglasEjemploDesarrollo(configuracionDesarrolloReglasEjemplo(rutaReglasBolsaEjemploPrueba, ""), calendarios, relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	calculadora := &calculadoraPlazoOfertaDesarrollo{}
	calculadora.fijar(compuestas.bolsa)
	calculadora.fijar(nil)
	publicada := time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)
	plazo, vencimiento, err := calculadora.PlazoDisposicion(t.Context(), publicada)
	if err != nil || !vencimiento.Equal(vence) || plazo.UltimoDia != "2026-09-30" || plazo.Cantidad != 2 ||
		plazo.Unidad != string(reglas.UnidadDiasHabiles) || plazo.Articulo != "art. 8.1.b" || len(plazo.HuellaCatalogo) != 64 ||
		plazo.ReglaRef == "" || calendarios.recibida.Cantidad != 2 || !calendarios.recibida.NotificadoEn.Equal(publicada) {
		t.Fatalf("plazo=%+v vence=%v err=%v recibida=%+v", plazo, vencimiento, err, calendarios.recibida)
	}
}

func TestFronterasOfertasUsanLaCapacidadDeEmision(t *testing.T) {
	fronteras, err := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo("prf_0123456789abcdefghijkl")
	if err != nil {
		t.Fatal(err)
	}
	rutas := map[string]bool{}
	for _, f := range fronteras {
		if f.Ruta == bolsahttp.RutaOfertasPublicadas || f.Ruta == bolsahttp.RutaResolucionesOferta {
			rutas[f.Metodo+" "+f.Ruta] = true
			if f.ClaveCapacidad != claveCapacidadEmisionLlamamientoBolsa || f.ClavePolitica != clavePoliticaBorradorLlamamientoBolsaDesarrollo {
				t.Errorf("frontera %s con capacidad %s", f.Clave, f.ClaveCapacidad)
			}
		}
	}
	if len(rutas) != 3 {
		t.Fatalf("fronteras de ofertas: %v", rutas)
	}
	catalogoFronteras, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct{ metodo, ruta string }{
		{http.MethodPost, bolsahttp.RutaOfertasPublicadas}, {http.MethodGet, bolsahttp.RutaOfertasPublicadas}, {http.MethodPost, bolsahttp.RutaResolucionesOferta},
	} {
		if _, ok := catalogoFronteras.resolver(caso.metodo, caso.ruta); !ok {
			t.Fatalf("%s %s sin frontera", caso.metodo, caso.ruta)
		}
	}
	if _, ok := catalogoFronteras.resolver(http.MethodGet, bolsahttp.RutaResolucionesOferta); ok {
		t.Fatal("GET de resoluciones declarado")
	}
	descriptores, err := descriptoresAutorizacionBorradorLlamamientoBolsaDesarrollo(politicaDescriptoresBolsaPrueba(t))
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoAutorizacionComunDesarrollo(catalogoFronteras, descriptores)
	if err != nil {
		t.Fatal(err)
	}
	for _, frontera := range []string{claveFronteraPublicarOfertaBolsa, claveFronteraConsultarOfertaBolsa, claveFronteraResolverOfertaBolsa} {
		if p, ok := catalogo.politicaPara(puertosbolsa.AccionEmitirLlamamiento, frontera, clavePoliticaBorradorLlamamientoBolsaDesarrollo, claveCapacidadEmisionLlamamientoBolsa); !ok || !p.valida() {
			t.Fatalf("la emisión no cubre %s", frontera)
		}
		if _, ok := catalogo.politicaPara(puertosbolsa.AccionCambiarSituacionParticipacion, frontera, clavePoliticaBorradorLlamamientoBolsaDesarrollo, claveCapacidadSituacionParticipacionBolsa); ok {
			t.Fatalf("otra acción admitida en %s", frontera)
		}
	}
}
