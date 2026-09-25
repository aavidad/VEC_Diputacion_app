package bootstrap

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	"vec-diputacion-granada/internal/vec/reglas"
)

const (
	rutaRetribucionesCTEjemploPrueba = "../../../data/demo/reglas/ct_retribuciones.demo.json"
)

type relojRetribucionesPrueba time.Time

func (r relojRetribucionesPrueba) Ahora() time.Time { return time.Time(r) }

var diaRetribucionesPrueba = relojRetribucionesPrueba(time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC))

// tablaAnteriorCosteDesarrollo es la tabla que estaba fijada en el código
// antes del catálogo: coste empresa mensual por grupo a jornada completa.
var tablaAnteriorCosteDesarrollo = map[string]int64{
	"A1": 460_000, "A2": 390_000, "B": 330_000, "C1": 300_000, "C2": 260_000, "AP": 230_000,
}

// costeAnteriorDesarrollo reproduce literalmente el cálculo anterior.
func costeAnteriorDesarrollo(mensual, dias, jornada int64) int64 {
	numerador := mensual * dias * jornada
	return (numerador + diasMesDiezmilesimasCosteDesarrollo/2) / diasMesDiezmilesimasCosteDesarrollo
}

// escribirCatalogoRetribucionesPrueba parte del paquete de ejemplo y cambia sus
// filas por las indicadas (clave → atributos).
func escribirCatalogoRetribucionesPrueba(t *testing.T, filas []map[string]string) string {
	t.Helper()
	contenido, err := os.ReadFile(rutaRetribucionesCTEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	var paquete map[string]any
	if err := json.Unmarshal(contenido, &paquete); err != nil {
		t.Fatal(err)
	}
	catalogo := paquete["catalogo"].(map[string]any)
	entradas := make([]any, 0, len(filas))
	for indice, atributos := range filas {
		entradas = append(entradas, map[string]any{
			"clave": "fila." + strconv.Itoa(indice+1), "etiqueta": "Fila " + strconv.Itoa(indice+1),
			"descripcion": "Fila de prueba.", "vigente_desde": "2026-01-01T00:00:00Z",
			"atributos": atributos, "orden": indice + 1,
		})
	}
	catalogo["entradas"] = entradas
	return escribirJSONPrueba(t, paquete)
}

func escribirJSONPrueba(t *testing.T, paquete map[string]any) string {
	t.Helper()
	salida, err := json.Marshal(paquete)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "catalogo.demo.json")
	if err := os.WriteFile(ruta, salida, 0o600); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func filaRetribucionPrueba(grupo string, mensual int64, pagas, cuota string) map[string]string {
	return map[string]string{
		"origen": "ejemplo", "norma": "Prueba.", "duda": "Duda 8.", "grupo": grupo,
		"sueldo_mensual_centimos": strconv.FormatInt(mensual, 10), "complementos_mensual_centimos": "0",
		"pagas_anuales": pagas, "seguridad_social_centesimas": cuota, "proporcional_jornada": "si",
	}
}

// fuenteRetribucionesTablaAnteriorPrueba compone un catálogo que reproduce la
// tabla anterior: 12 pagas, sin cuota y el coste mensual como sueldo.
func fuenteRetribucionesTablaAnteriorPrueba(t *testing.T) *fuenteRetribucionesDesarrollo {
	t.Helper()
	filas := make([]map[string]string, 0, len(tablaAnteriorCosteDesarrollo))
	for _, grupo := range []string{"A1", "A2", "B", "C1", "C2", "AP"} {
		filas = append(filas, filaRetribucionPrueba(grupo, tablaAnteriorCosteDesarrollo[grupo], "12", "0"))
	}
	fuente, err := nuevaFuenteRetribucionesDesarrollo(escribirCatalogoRetribucionesPrueba(t, filas), diaRetribucionesPrueba)
	if err != nil || fuente == nil {
		t.Fatalf("catálogo equivalente a la tabla anterior: %v", err)
	}
	return fuente
}

func TestRetribucionesQueReproducenLaTablaAnteriorDanElMismoCoste(t *testing.T) {
	t.Parallel()
	fuente := fuenteRetribucionesTablaAnteriorPrueba(t)
	inicio := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	for grupo, mensual := range tablaAnteriorCosteDesarrollo {
		fila, ok, _ := fuente.retribucion(t.Context(), "categoria:cualquiera", grupo)
		if !ok {
			t.Fatalf("%s sin fila", grupo)
		}
		for dias := int64(1); dias <= 800; dias += 13 {
			periodo := domain.PeriodoPrevisto{Inicio: inicio, Fin: inicio.AddDate(0, 0, int(dias-1))}
			for jornada := int64(1); jornada <= 10_000; jornada += 97 {
				importe, ok := costeEstimadoAnalisisDesarrollo(fila, periodo, domain.JornadaDiezmilesimas(jornada))
				if esperado := costeAnteriorDesarrollo(mensual, dias, jornada); esperado > 0 && (!ok || importe.Centimos != esperado) {
					t.Fatalf("%s %d días jornada %d: %d, antes %d", grupo, dias, jornada, importe.Centimos, esperado)
				}
			}
		}
	}
	fin := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	c2, _, _ := fuente.retribucion(t.Context(), "", "C2")
	a1, _, _ := fuente.retribucion(t.Context(), "", "A1")
	for nombre, caso := range map[string]struct {
		fila     retribucionReferenciaDesarrollo
		periodo  domain.PeriodoPrevisto
		jornada  domain.JornadaDiezmilesimas
		centimos int64
	}{
		"C2 118 días jornada completa": {c2, domain.PeriodoPrevisto{Inicio: inicio, Fin: fin}, domain.JornadaCompletaDiezmilesimas, 1_007_967},
		"C2 media jornada":             {c2, domain.PeriodoPrevisto{Inicio: inicio, Fin: fin}, 5_000, 503_984},
		"A1 un día":                    {a1, domain.PeriodoPrevisto{Inicio: inicio, Fin: inicio}, domain.JornadaCompletaDiezmilesimas, 15_113},
	} {
		importe, ok := costeEstimadoAnalisisDesarrollo(caso.fila, caso.periodo, caso.jornada)
		if !ok || importe.Centimos != caso.centimos || importe.Moneda != "EUR" {
			t.Fatalf("%s: %+v %v", nombre, importe, ok)
		}
	}
	for nombre, caso := range map[string]struct {
		periodo domain.PeriodoPrevisto
		jornada domain.JornadaDiezmilesimas
	}{
		"periodo invertido":   {domain.PeriodoPrevisto{Inicio: fin, Fin: inicio}, domain.JornadaCompletaDiezmilesimas},
		"jornada cero":        {domain.PeriodoPrevisto{Inicio: inicio, Fin: fin}, 0},
		"periodo desmesurado": {domain.PeriodoPrevisto{Inicio: inicio, Fin: inicio.AddDate(11, 0, 0)}, domain.JornadaCompletaDiezmilesimas},
	} {
		if _, ok := costeEstimadoAnalisisDesarrollo(c2, caso.periodo, caso.jornada); ok {
			t.Fatalf("%s: debe quedar sin calcular", nombre)
		}
	}
	if _, ok, _ := fuente.retribucion(t.Context(), "", "Z9"); ok {
		t.Fatal("un grupo sin fila no tiene coste")
	}
}

func TestRetribucionesDeEjemploAplicanPagasCuotaYJornada(t *testing.T) {
	t.Parallel()
	fuente, err := nuevaFuenteRetribucionesDesarrollo(rutaRetribucionesCTEjemploPrueba, diaRetribucionesPrueba)
	if err != nil || fuente == nil {
		t.Fatalf("el paquete de ejemplo debe cargarse: %v", err)
	}
	filas, ejemplo, err := fuente.tabla(t.Context())
	if err != nil || !ejemplo || len(filas) != 6 {
		t.Fatalf("tabla de ejemplo: %d %v %v", len(filas), ejemplo, err)
	}
	c2, ok, _ := fuente.retribucion(t.Context(), "categoria:desarrollo:auxiliar", "C2")
	if !ok || c2.pagas != 14 || c2.seguridadSocialCentesimas != 3200 || !c2.proporcionalJornada {
		t.Fatalf("fila C2: %+v %v", c2, ok)
	}
	inicio := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	periodo := domain.PeriodoPrevisto{Inicio: inicio, Fin: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)}
	// (725 + 900) € × 14 ÷ 12 × 1,32 = 2.502,50 € al mes; 118 días.
	completa, ok := costeEstimadoAnalisisDesarrollo(c2, periodo, domain.JornadaCompletaDiezmilesimas)
	media, okMedia := costeEstimadoAnalisisDesarrollo(c2, periodo, 5_000)
	if !ok || !okMedia || completa.Centimos != 970_168 || media.Centimos != 485_084 {
		t.Fatalf("coste C2: %+v %+v", completa, media)
	}
	sinProrrateo := c2
	sinProrrateo.proporcionalJornada = false
	if entera, _ := costeEstimadoAnalisisDesarrollo(sinProrrateo, periodo, 5_000); entera.Centimos != completa.Centimos {
		t.Fatalf("sin prorrateo por jornada se cobra la completa: %+v", entera)
	}
}

func TestRetribucionesPorCategoriaPrevalecenSobreElGrupo(t *testing.T) {
	t.Parallel()
	especial := filaRetribucionPrueba("C2", 300_000, "14", "3200")
	especial["categoria_ref"] = "categoria:desarrollo:conductor"
	fuente, err := nuevaFuenteRetribucionesDesarrollo(escribirCatalogoRetribucionesPrueba(t, []map[string]string{
		filaRetribucionPrueba("C2", 200_000, "14", "3200"), especial,
	}), diaRetribucionesPrueba)
	if err != nil {
		t.Fatal(err)
	}
	if fila, ok, _ := fuente.retribucion(t.Context(), "categoria:desarrollo:conductor", "C2"); !ok || fila.sueldo != 300_000 {
		t.Fatalf("la categoría debe prevalecer: %+v %v", fila, ok)
	}
	if fila, ok, _ := fuente.retribucion(t.Context(), "categoria:desarrollo:otra", "C2"); !ok || fila.sueldo != 200_000 {
		t.Fatalf("otra categoría usa su grupo: %+v %v", fila, ok)
	}
	if _, ok, _ := fuente.retribucion(t.Context(), "categoria:desarrollo:conductor", "C1"); ok {
		t.Fatal("una fila de categoría no vale para otro grupo")
	}
}

func TestRetribucionesNoValidasImpidenArrancar(t *testing.T) {
	t.Parallel()
	alterar := func(clave, valor string) map[string]string {
		fila := filaRetribucionPrueba("C2", 100_000, "14", "3200")
		if valor == "" {
			delete(fila, clave)
		} else {
			fila[clave] = valor
		}
		return fila
	}
	for nombre, filas := range map[string][]map[string]string{
		"sin pagas":           {alterar("pagas_anuales", "")},
		"pagas cero":          {alterar("pagas_anuales", "0")},
		"cuota excesiva":      {alterar("seguridad_social_centesimas", "10001")},
		"importe con signo":   {alterar("sueldo_mensual_centimos", "+100")},
		"proporcional dudoso": {alterar("proporcional_jornada", "quizá")},
		"grupo no válido":     {alterar("grupo", "c2")},
		"categoría vacía":     {alterar("categoria_ref", " ")},
		"sin importe":         {alterar("sueldo_mensual_centimos", "0")},
		"grupo repetido":      {filaRetribucionPrueba("C2", 1, "14", "0"), filaRetribucionPrueba("C2", 2, "14", "0")},
	} {
		if _, err := nuevaFuenteRetribucionesDesarrollo(escribirCatalogoRetribucionesPrueba(t, filas), diaRetribucionesPrueba); !errors.Is(err, errRetribucionesCTNoValidas) {
			t.Fatalf("%s: %v", nombre, err)
		}
	}
	if fuente, err := nuevaFuenteRetribucionesDesarrollo(" ", diaRetribucionesPrueba); fuente != nil || err != nil {
		t.Fatalf("sin ruta no hay catálogo: %v %v", fuente, err)
	}
	var sinCatalogo *fuenteRetribucionesDesarrollo
	if _, ok, err := sinCatalogo.retribucion(t.Context(), "", "C2"); ok || err != nil {
		t.Fatal("sin catálogo el coste queda sin calcular")
	}
}

func TestSinRetribucionesElAnalisisContinuaSinCoste(t *testing.T) {
	derivador := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	capacidad, err := nuevoPreparadorFuentesAnalisisContratacionTemporalDesarrollo(
		derivador, relojContratacionTemporalDesarrollo{}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := solicitudPrepararArtefactoAnalisisDesarrolloPrueba("sustitucion", "expediente:ct:desarrollo:analisis:sin-coste")
	artefacto, err := capacidad.PrepararArtefactoAnalisis(t.Context(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := artefacto.DatosPara(solicitud)
	if err != nil || datos.CostePrevisto != nil || datos.ImporteRC == nil {
		t.Fatalf("sin catálogo el coste queda sin calcular: %+v %v", datos, err)
	}
}

func escribirReglasCTConJornadaPrueba(t *testing.T, atributos map[string]any) string {
	t.Helper()
	contenido, err := os.ReadFile(rutaReglasCTEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	var paquete map[string]any
	if err := json.Unmarshal(contenido, &paquete); err != nil {
		t.Fatal(err)
	}
	for _, entrada := range paquete["catalogo"].(map[string]any)["entradas"].([]any) {
		registro := entrada.(map[string]any)
		if registro["clave"] == reglas.CTJornadaCompleta {
			for clave, valor := range atributos {
				registro["atributos"].(map[string]any)[clave] = valor
			}
		}
	}
	return escribirJSONPrueba(t, paquete)
}

func fuenteJornadaPrueba(t *testing.T, ruta string) fuenteJornadaCompletaDesarrollo {
	t.Helper()
	resolutor, err := nuevoResolutorReglasEjemplo(ruta, reglas.CatalogoContratacionTemporal, reglas.ModuloContratacionTemporal, nil, diaRetribucionesPrueba)
	if err != nil {
		t.Fatal(err)
	}
	return fuenteJornadaCompletaDesarrollo{resolutor: resolutor}
}

func TestJornadaCompletaSaleDeLaReglaC07(t *testing.T) {
	t.Parallel()
	if minutos, err := (fuenteJornadaCompletaDesarrollo{}).minutos(t.Context()); err != nil || minutos != 37*60+30 {
		t.Fatalf("sin catálogo, 37 h 30 min: %d %v", minutos, err)
	}
	if minutos, err := fuenteJornadaPrueba(t, rutaReglasCTEjemploPrueba).minutos(t.Context()); err != nil || minutos != 2250 {
		t.Fatalf("paquete de ejemplo: %d %v", minutos, err)
	}
	treintaCinco := fuenteJornadaPrueba(t, escribirReglasCTConJornadaPrueba(t, map[string]any{"cantidad": "2100"}))
	if minutos, err := treintaCinco.minutos(t.Context()); err != nil || minutos != 2100 {
		t.Fatalf("la regla manda: %d %v", minutos, err)
	}
	semanaYMas := fuenteJornadaPrueba(t, escribirReglasCTConJornadaPrueba(t, map[string]any{"cantidad": "10081"}))
	if _, err := semanaYMas.minutos(t.Context()); !errors.Is(err, errJornadaCompletaNoDisponible) {
		t.Fatalf("una jornada mayor que la semana no vale: %v", err)
	}
	enHoras := fuenteJornadaPrueba(t, escribirReglasCTConJornadaPrueba(t, map[string]any{"unidad": "horas", "cantidad": "37"}))
	if _, err := enHoras.minutos(t.Context()); !errors.Is(err, errJornadaCompletaNoDisponible) {
		t.Fatalf("otra unidad no se reinterpreta: %v", err)
	}

	for nombre, caso := range map[string]struct {
		fuente fuenteJornadaCompletaDesarrollo
		estado int
		valor  int
	}{
		"sin catálogo":    {fuenteJornadaCompletaDesarrollo{}, http.StatusOK, 2250},
		"regla de 35 h":   {treintaCinco, http.StatusOK, 2100},
		"regla no válida": {enHoras, http.StatusServiceUnavailable, 0},
	} {
		ruta, err := nuevaRutaConfiguracionAnalisisConReglasDesarrollo(false, fuenteMotivosRectificacionAnalisisDesarrollo{}, caso.fuente)
		if err != nil {
			t.Fatal(err)
		}
		respuesta := httptest.NewRecorder()
		ruta.Manejador.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, ruta.Ruta, nil))
		if respuesta.Code != caso.estado {
			t.Fatalf("%s: estado %d", nombre, respuesta.Code)
		}
		if caso.estado != http.StatusOK {
			continue
		}
		var contenido struct {
			Data struct {
				Minutos *int `json:"jornada_completa_minutos_semanales"`
			} `json:"data"`
		}
		if err := json.Unmarshal(respuesta.Body.Bytes(), &contenido); err != nil || contenido.Data.Minutos == nil || *contenido.Data.Minutos != caso.valor {
			t.Fatalf("%s: %s %v", nombre, respuesta.Body.String(), err)
		}
	}
}

func TestFuentesReglasAnalisisSeComponenSoloConDobleLlave(t *testing.T) {
	t.Parallel()
	sinCatalogo, err := nuevasFuentesReglasAnalisisDesarrollo(configuracionDesarrolloReglasEjemplo("", ""), diaRetribucionesPrueba)
	if err != nil || sinCatalogo.jornada.resolutor != nil || sinCatalogo.retribuciones != nil {
		t.Fatalf("sin catálogos no se compone nada: %+v %v", sinCatalogo, err)
	}
	cfg := configuracionDesarrolloReglasEjemplo("", rutaReglasCTEjemploPrueba)
	cfg.ReglasEjemplo.CTRetribucionesSourcePath = rutaRetribucionesCTEjemploPrueba
	compuestas, err := nuevasFuentesReglasAnalisisDesarrollo(cfg, diaRetribucionesPrueba)
	if err != nil || compuestas.jornada.resolutor == nil || compuestas.retribuciones == nil {
		t.Fatalf("con catálogos se componen ambos: %+v %v", compuestas, err)
	}
	invalida := configuracionDesarrolloReglasEjemplo("", escribirReglasCTConJornadaPrueba(t, map[string]any{"unidad": "horas", "cantidad": "37"}))
	if _, err := nuevasFuentesReglasAnalisisDesarrollo(invalida, diaRetribucionesPrueba); !errors.Is(err, errReglasEjemploNoValidas) {
		t.Fatalf("una c07 no válida impide arrancar: %v", err)
	}
	produccion := cfg
	produccion.ExecutionProfile = "production"
	if _, err := nuevasFuentesReglasAnalisisDesarrollo(produccion, diaRetribucionesPrueba); err == nil {
		t.Fatal("fuera de desarrollo no se admiten catálogos de ejemplo")
	}
}

func TestRetribucionesNoDisponiblesNoSeConfundenConSinCatalogo(t *testing.T) {
	t.Parallel()
	antes := relojRetribucionesPrueba(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	if _, err := nuevaFuenteRetribucionesDesarrollo(rutaRetribucionesCTEjemploPrueba, antes); !errors.Is(err, errRetribucionesCTNoValidas) {
		t.Fatalf("sin versión vigente al arrancar no se compone: %v", err)
	}
	fuente, err := nuevaFuenteRetribucionesDesarrollo(rutaRetribucionesCTEjemploPrueba, diaRetribucionesPrueba)
	if err != nil {
		t.Fatal(err)
	}
	consulta, err := fichero.NuevaConsultaCatalogos(rutaRetribucionesCTEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{
		Consulta: consulta, CatalogoID: reglas.CatalogoRetribucionesCT,
		ModuloID: reglas.ModuloContratacionTemporal, Reloj: antes,
	})
	if err != nil {
		t.Fatal(err)
	}
	fuente.resolutor = resolutor
	if _, ok, err := fuente.retribucion(t.Context(), "", "C2"); ok || !errors.Is(err, reglas.ErrReglasNoDisponibles) {
		t.Fatalf("un catálogo declarado y no vigente es un error: %v %v", ok, err)
	}
}
