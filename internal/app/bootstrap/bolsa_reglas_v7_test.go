package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	reglasbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/reglas"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	calendariosdomain "vec-diputacion-granada/internal/modules/calendarios/domain"
	calendariosports "vec-diputacion-granada/internal/modules/calendarios/ports"
	"vec-diputacion-granada/internal/vec/reglas"
	reglashttp "vec-diputacion-granada/internal/vec/reglas/httpinterno"
)

// Catálogo real que la copia local aporta junto al portal del candidato
// (deploy/principal/copia_local_caso40.py, F1-109). Las pruebas leen los
// valores esperados del propio fichero: si RRHH publica otra versión, estas
// pruebas siguen a esa versión en lugar de fijar cifras inventadas.
const rutaReglasBolsaV7 = "../../../data/catalogos/bolsa/reglas.v7.json"

// La entrada más reciente de v7 (b10) rige desde el 15/10/2026.
var relojReglasBolsaV7 = relojReglasEjemploPrueba{ahora: time.Date(2026, 10, 20, 8, 0, 0, 0, time.UTC)}

type catalogoBolsaV7Prueba struct {
	Catalogo struct {
		ID       string `json:"id"`
		Version  int    `json:"version"`
		Entradas []struct {
			Clave     string            `json:"clave"`
			Atributos map[string]string `json:"atributos"`
		} `json:"entradas"`
	} `json:"catalogo"`
}

func leerCatalogoBolsaV7(t *testing.T) catalogoBolsaV7Prueba {
	t.Helper()
	contenido, err := os.ReadFile(rutaReglasBolsaV7)
	if err != nil {
		t.Fatal(err)
	}
	var catalogo catalogoBolsaV7Prueba
	if err := json.Unmarshal(contenido, &catalogo); err != nil {
		t.Fatal(err)
	}
	return catalogo
}

func (c catalogoBolsaV7Prueba) atributos(t *testing.T, clave string) map[string]string {
	t.Helper()
	for _, entrada := range c.Catalogo.Entradas {
		if entrada.Clave == clave {
			return entrada.Atributos
		}
	}
	t.Fatalf("v7 no trae %s", clave)
	return nil
}

func enteroCatalogoV7(t *testing.T, atributos map[string]string, nombre string) int {
	t.Helper()
	valor, err := strconv.Atoi(atributos[nombre])
	if err != nil {
		t.Fatalf("atributo %s no entero: %q", nombre, atributos[nombre])
	}
	return valor
}

func referenciaV7(clave string) string { return reglas.CatalogoBolsa + ":7:" + clave }

// componerReglasBolsaV7 compone v7 por el mismo camino que el arranque
// (nuevasReglasEjemploDesarrollo) con un Calendarios de prueba.
func componerReglasBolsaV7(t *testing.T, calendarios calendariosports.ConsultaCalendarios) reglasEjemploDesarrollo {
	t.Helper()
	compuestas, err := nuevasReglasEjemploDesarrollo(configuracionDesarrolloReglasEjemplo(rutaReglasBolsaV7, ""), calendarios, relojReglasBolsaV7)
	if err != nil {
		t.Fatalf("v7 no se compone: %v", err)
	}
	if !compuestas.bolsa.Disponible() {
		t.Fatal("v7 compuesto sin resolutor de Bolsa")
	}
	return compuestas
}

func calendariosV7Prueba(t *testing.T, dia string) (*consultaCalendariosReglasPrueba, time.Time) {
	t.Helper()
	ultimo, err := calendariosdomain.ParsearFechaCivil(dia)
	if err != nil {
		t.Fatal(err)
	}
	fin, err := ultimo.FinEnMadrid()
	if err != nil {
		t.Fatal(err)
	}
	return &consultaCalendariosReglasPrueba{resultado: calendariosports.ResultadoCalculoPlazo{
		ResultadoPlazo: calendariosdomain.ResultadoPlazo{Vencimiento: ultimo, VenceAntesDe: fin},
	}}, fin
}

func TestCatalogoBolsaV7SeCargaYValidaSinMarcaDeEjemplo(t *testing.T) {
	catalogo := leerCatalogoBolsaV7(t)
	if catalogo.Catalogo.ID != reglas.CatalogoBolsa || catalogo.Catalogo.Version != 7 {
		t.Fatalf("fichero inesperado: %s v%d", catalogo.Catalogo.ID, catalogo.Catalogo.Version)
	}
	compuestas := componerReglasBolsaV7(t, nil)
	vigentes, err := compuestas.bolsa.Reglas(t.Context())
	if err != nil || len(vigentes) != len(catalogo.Catalogo.Entradas) {
		t.Fatalf("reglas vigentes: %d/%d, %v", len(vigentes), len(catalogo.Catalogo.Entradas), err)
	}
	for _, regla := range vigentes {
		if regla.ReferenciaEntrada.CatalogoVersion != 7 || len(regla.HuellaCatalogo) != 64 {
			t.Fatalf("%s no sale de v7: %+v", regla.Clave, regla.ReferenciaEntrada)
		}
		if regla.PaqueteEjemplo || (regla.Origen == reglas.OrigenConfiguracion && regla.EsEjemplo()) {
			t.Fatalf("marca de demostración en %s", regla.Clave)
		}
	}
	// El selector del portal del candidato, con la ruta que pone la copia,
	// no detiene el arranque.
	cfg := configuracionDesarrolloReglasEjemplo(rutaReglasBolsaV7, "")
	cfg.BolsaPortalCandidatoEnabled = "true"
	if activo, err := cfg.BolsaPortalCandidatoDesarrolloActivo(); err != nil || !activo {
		t.Fatalf("portal con v7: activo=%v err=%v", activo, err)
	}
}

// v7 cuenta el plazo de la oferta desde su publicación (art. 8.1.b): la
// calculadora de ofertas pide a Calendarios el plazo con la fecha de
// publicación como inicio y con la cantidad, unidad y artículo del catálogo.
func TestOfertasV7CuentanElPlazoDesdeLaPublicacion(t *testing.T) {
	b10 := leerCatalogoBolsaV7(t).atributos(t, reglas.BolsaPlazoPublicacion)
	if b10["inicio"] != "publicacion" {
		t.Fatalf("v7 ya no cuenta desde la publicación: %q", b10["inicio"])
	}
	calendarios, fin := calendariosV7Prueba(t, "2026-10-22")
	compuestas := componerReglasBolsaV7(t, calendarios)
	regla, err := compuestas.bolsa.Regla(t.Context(), reglas.BolsaPlazoPublicacion)
	if err != nil || regla.Inicio != "publicacion" || regla.Referencia != referenciaV7(reglas.BolsaPlazoPublicacion) {
		t.Fatalf("b10 de v7: %+v %v", regla, err)
	}
	calculadora := &calculadoraPlazoOfertaDesarrollo{}
	calculadora.fijar(compuestas.bolsa)
	publicada := time.Date(2026, 10, 20, 9, 30, 0, 0, time.UTC)
	plazo, vence, err := calculadora.PlazoDisposicion(t.Context(), publicada)
	cantidad := enteroCatalogoV7(t, b10, "cantidad")
	if err != nil || !vence.Equal(fin) || plazo.UltimoDia != "2026-10-22" || plazo.Cantidad != cantidad ||
		plazo.Unidad != b10["unidad"] || plazo.Computo != b10["computo"] || plazo.Articulo != b10["articulo"] ||
		plazo.ReglaRef != referenciaV7(reglas.BolsaPlazoPublicacion) || plazo.Ejemplo || len(plazo.HuellaCatalogo) != 64 {
		t.Fatalf("plazo=%+v vence=%v err=%v", plazo, vence, err)
	}
	if !calendarios.recibida.NotificadoEn.Equal(publicada) || calendarios.recibida.Cantidad != cantidad ||
		string(calendarios.recibida.Unidad) != b10["unidad"] || calendarios.recibida.MunicipioSede != reglas.MunicipioSedeDiputacion {
		t.Fatalf("Calendarios no recibe la publicación como inicio: %+v", calendarios.recibida)
	}
}

// La web de RRHH propone la política de ofertas de cada bolsa con la regla
// b10 que sirve la consulta de reglas vigentes (rrhh-plazos-ui.js). Con v7
// esa propuesta llega con inicio «publicacion» y sin marca de ejemplo.
func TestReglasVigentesV7ProponenElInicioPorPublicacion(t *testing.T) {
	b10 := leerCatalogoBolsaV7(t).atributos(t, reglas.BolsaPlazoPublicacion)
	ruta, err := nuevaRutaReglasVigentesDesarrollo(componerReglasBolsaV7(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta.Ruta, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("estado=%d %s", w.Code, w.Body.String())
	}
	var respuesta reglashttp.Respuesta
	if err := json.Unmarshal(w.Body.Bytes(), &respuesta); err != nil {
		t.Fatal(err)
	}
	var encontrada bool
	for _, catalogo := range respuesta.Data.Catalogos {
		if catalogo.Modulo != reglas.ModuloBolsa {
			continue
		}
		if catalogo.Estado != reglashttp.EstadoDisponible || catalogo.Version != 7 || catalogo.PaqueteEjemplo {
			t.Fatalf("catálogo de Bolsa servido: %+v", catalogo)
		}
		for _, regla := range catalogo.Reglas {
			if regla.Clave != reglas.BolsaPlazoPublicacion {
				continue
			}
			encontrada = true
			if regla.Inicio != "publicacion" || regla.Cantidad != enteroCatalogoV7(t, b10, "cantidad") ||
				regla.Unidad != b10["unidad"] || regla.Computo != b10["computo"] || regla.Origen != b10["origen"] {
				t.Fatalf("b10 servida: %+v", regla)
			}
		}
	}
	if !encontrada {
		t.Fatal("la consulta no sirve b10 de v7")
	}
}

// Los avisos y marcas que el arranque publica en Bolsa (b16, b17 y b19) y la
// política de intentos (b02, b03 y b04) son los de v7.
func TestAvisosEIntentosV7SonLosDelCatalogo(t *testing.T) {
	catalogo := leerCatalogoBolsaV7(t)
	compuestas := componerReglasBolsaV7(t, nil)
	publicacion, hay, err := reglasbolsa.PoliticaAvisos(t.Context(), compuestas.bolsa)
	if err != nil || !hay {
		t.Fatalf("avisos de v7: hay=%v err=%v", hay, err)
	}
	b16 := catalogo.atributos(t, reglas.BolsaPrestaServicios)
	b17 := catalogo.atributos(t, reglas.BolsaAvisoEncadenamiento)
	b19 := catalogo.atributos(t, reglas.BolsaVacanteDuracionMaxima)
	if b19["unidad"] != string(reglas.UnidadAnios) {
		t.Fatalf("b19 cambió de unidad: %q", b19["unidad"])
	}
	politica := publicacion.Politica
	if !politica.ContinuadoConfigurado || politica.ContinuadoMeses != 12*enteroCatalogoV7(t, b19, "cantidad") ||
		politica.ContinuadoAntelacionDias != enteroCatalogoV7(t, b19, "antelacion_dias") ||
		politica.EncadenamientoUmbralMeses != enteroCatalogoV7(t, b17, "cantidad") ||
		politica.EncadenamientoVentanaMeses != enteroCatalogoV7(t, b17, "ventana_meses") ||
		politica.PrestaServiciosModo != b16["modo"] ||
		!slices.Equal(politica.PrestaServiciosSituaciones, strings.Split(b16["situaciones"], ",")) {
		t.Fatalf("política de avisos: %+v", politica)
	}
	if publicacion.CatalogoRef != referenciaV7("avisos") || len(publicacion.CatalogoSHA256) != 64 {
		t.Fatalf("referencia de la publicación: %q", publicacion.CatalogoRef)
	}
	intentos, _, hay, err := reglasbolsa.NuevosIntentosContacto(compuestas.bolsa).PoliticaIntentosTelefonicos(t.Context())
	if err != nil || !hay {
		t.Fatalf("intentos de v7: hay=%v err=%v", hay, err)
	}
	b02 := catalogo.atributos(t, reglas.BolsaIntentosContacto)
	separacion := catalogo.atributos(t, reglas.BolsaSeparacionIntentos)
	b03 := catalogo.atributos(t, reglas.BolsaProcesosSinContacto)
	if separacion["unidad"] != "horas" {
		t.Fatalf("b02 separación cambió de unidad: %q", separacion["unidad"])
	}
	if intentos.IntentosPorProceso != enteroCatalogoV7(t, b02, "cantidad") || intentos.Procesos != enteroCatalogoV7(t, b03, "cantidad") ||
		intentos.SeparacionMinima != time.Duration(enteroCatalogoV7(t, separacion, "cantidad"))*time.Hour ||
		intentos.ControlSeparacion != separacion["control"] ||
		!slices.Equal(intentos.ResultadosSinContacto, strings.Split(b03["resultados_sin_contacto"], ",")) {
		t.Fatalf("política de intentos: %+v", intentos)
	}
}

// F1-99: Antonio responde al llamamiento desde Mi Bolsa. Con v7 el portal
// obtiene el modo firme, el contacto efectivo, las causas de renuncia
// justificada y el plazo de respuesta (b05) contado desde el contacto.
func TestPortalCandidatoV7DejaResponderAlLlamamiento(t *testing.T) {
	catalogo := leerCatalogoBolsaV7(t)
	b29 := catalogo.atributos(t, reglas.BolsaPortalCandidato)
	b05 := catalogo.atributos(t, reglas.BolsaPlazoRespuesta)
	b11 := catalogo.atributos(t, reglas.BolsaAcreditarRenunciaJustificada)
	calendarios, fin := calendariosV7Prueba(t, "2026-10-21")
	portal := reglasPortalCandidatoDesarrollo{resolutor: componerReglasBolsaV7(t, calendarios).bolsa}
	ctx := context.Background()
	modo, ref, err := portal.ModoRespuesta(ctx)
	if err != nil || modo != b29["modo_respuesta"] || modo != puertosbolsa.ModoRespuestaPortalFirme || ref != referenciaV7(reglas.BolsaPortalCandidato) {
		t.Fatalf("modo de respuesta: %q %q %v", modo, ref, err)
	}
	if efectivos, err := portal.ResultadosContactoEfectivo(ctx); err != nil || !slices.Equal(efectivos, strings.Split(b29["contacto_efectivo"], ",")) {
		t.Fatalf("contacto efectivo: %v %v", efectivos, err)
	}
	if causas, err := portal.CausasRenunciaJustificada(ctx); err != nil || !slices.Equal(causas, strings.Split(b11["causas"], ",")) {
		t.Fatalf("causas del art. 10: %v %v", causas, err)
	}
	contacto := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	vence, ref, err := portal.VencimientoRespuesta(ctx, contacto)
	if err != nil || !vence.Equal(fin.UTC()) || ref != referenciaV7(reglas.BolsaPlazoRespuesta) {
		t.Fatalf("plazo de respuesta: %v %q %v", vence, ref, err)
	}
	if !calendarios.recibida.NotificadoEn.Equal(contacto) || calendarios.recibida.Cantidad != enteroCatalogoV7(t, b05, "cantidad") ||
		string(calendarios.recibida.Unidad) != b05["unidad"] {
		t.Fatalf("Calendarios no recibe el contacto como inicio: %+v", calendarios.recibida)
	}
}
