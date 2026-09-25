package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	calendariosdomain "vec-diputacion-granada/internal/modules/calendarios/domain"
	calendariosports "vec-diputacion-granada/internal/modules/calendarios/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

type datosPlazoRespuestaPrueba struct {
	Data struct {
		Configurada bool `json:"configurada"`
		Regla       struct {
			Referencia string `json:"referencia"`
			Origen     string `json:"origen"`
			Ejemplo    bool   `json:"ejemplo"`
		} `json:"regla"`
		Vencimiento struct {
			UltimoDia string    `json:"ultimo_dia"`
			VenceEn   time.Time `json:"vence_en"`
		} `json:"vencimiento"`
	} `json:"data"`
}

func consultarPlazoRespuestaPrueba(t *testing.T, ruta vechttp.RutaExacta) datosPlazoRespuestaPrueba {
	t.Helper()
	if ruta.Ruta != bolsahttp.RutaPlazoRespuestaLlamamiento {
		t.Fatalf("ruta inesperada %q", ruta.Ruta)
	}
	w := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta.Ruta, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("estado=%d cuerpo=%s", w.Code, w.Body.String())
	}
	var datos datosPlazoRespuestaPrueba
	if err := json.Unmarshal(w.Body.Bytes(), &datos); err != nil {
		t.Fatal(err)
	}
	return datos
}

func TestPlazoRespuestaBolsaSeComponeConYSinCatalogo(t *testing.T) {
	sin, err := nuevaRutaPlazoRespuestaBolsaDesarrollo(nil, relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	if datos := consultarPlazoRespuestaPrueba(t, sin); datos.Data.Configurada {
		t.Fatalf("sin catálogo el asistente sigue con texto libre: %+v", datos)
	}
	ultimo, err := calendariosdomain.ParsearFechaCivil("2026-09-29")
	if err != nil {
		t.Fatal(err)
	}
	fin, err := ultimo.FinEnMadrid()
	if err != nil {
		t.Fatal(err)
	}
	calendarios := &consultaCalendariosReglasPrueba{resultado: calendariosports.ResultadoCalculoPlazo{
		ResultadoPlazo: calendariosdomain.ResultadoPlazo{Vencimiento: ultimo, VenceAntesDe: fin},
	}}
	compuestas, err := nuevasReglasEjemploDesarrollo(configuracionDesarrolloReglasEjemplo(rutaReglasBolsaEjemploPrueba, ""), calendarios, relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	con, err := nuevaRutaPlazoRespuestaBolsaDesarrollo(compuestas.bolsa, relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	datos := consultarPlazoRespuestaPrueba(t, con)
	if !datos.Data.Configurada || datos.Data.Regla.Referencia != "vec.bolsa.reglas:1:b05.plazo_respuesta" ||
		datos.Data.Regla.Origen != string(reglas.OrigenEjemplo) || !datos.Data.Regla.Ejemplo ||
		datos.Data.Vencimiento.UltimoDia != "2026-09-29" || !datos.Data.Vencimiento.VenceEn.Equal(fin.Add(-time.Second)) {
		t.Fatalf("propuesta inesperada: %+v", datos)
	}
	if !calendarios.recibida.NotificadoEn.Equal(relojPresentacionReglasEjemplo.ahora) || calendarios.recibida.Cantidad != 1 ||
		calendarios.recibida.MunicipioSede != reglas.MunicipioSedeDiputacion {
		t.Fatalf("el cálculo no parte de ahora en la sede: %+v", calendarios.recibida)
	}
	if _, err := nuevaRutaPlazoRespuestaBolsaDesarrollo(nil, nil); err == nil {
		t.Fatal("sin reloj la composición debe fallar")
	}
}

// La consulta usa la misma autorización que las lecturas RRHH de Bolsa del
// asistente B7: frontera mTLS con perfil técnico RRHH y capacidad sellada
// para la ruta exacta.
func TestPlazoRespuestaBolsaUsaLaAutorizacionDeLecturasRRHH(t *testing.T) {
	ruta := bolsahttp.RutaPlazoRespuestaLlamamiento
	if !esRutaContratacionTemporalDesarrollo(httptest.NewRequest(http.MethodGet, ruta, nil)) {
		t.Fatal("la frontera mTLS no revalida la ruta del plazo")
	}
	principal := func(rol string) vecdomain.Principal {
		return vecdomain.Principal{ID: "per_sintetico_plazo_123456789012345678", Roles: []string{rol},
			AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
			Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment}}
	}
	if principalContratacionTemporalDesarrolloValidoParaRuta(principal(rolIntervencionContratacionTemporalDesarrollo), ruta) {
		t.Fatal("un perfil ajeno a RRHH no puede consultar el plazo")
	}
	if principalContratacionTemporalDesarrolloValidoParaRuta(principal(rolTecnicoRRHHContratacionTemporalDesarrollo), ruta) !=
		principalContratacionTemporalDesarrolloValidoParaRuta(principal(rolTecnicoRRHHContratacionTemporalDesarrollo), rutaBolsasRRHHDesarrollo) {
		t.Fatal("el plazo debe exigir lo mismo que la lectura de bolsas")
	}
	sello := &selloConsultasContratacionTemporalDesarrollo{}
	autoridad := &autoridadConsultasContratacionTemporalDesarrollo{sello: sello, resolvedor: &resolvedorIdentidadDesarrollo{}}
	if err := autoridad.AutorizarRutaExacta(context.Background(), ruta); err != vechttp.ErrAutenticacionRutaExactaRequerida {
		t.Fatalf("sin capacidad: %v", err)
	}
	conCapacidad := func(s *selloConsultasContratacionTemporalDesarrollo, r string) context.Context {
		return context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{},
			capacidadConsultaContratacionTemporalDesarrollo{sello: s, ruta: r})
	}
	if err := autoridad.AutorizarRutaExacta(conCapacidad(sello, rutaBolsasRRHHDesarrollo), ruta); err != vechttp.ErrAccesoRutaExactaDenegado {
		t.Fatalf("una capacidad de otra ruta debe denegarse: %v", err)
	}
	if err := autoridad.AutorizarRutaExacta(conCapacidad(&selloConsultasContratacionTemporalDesarrollo{}, ruta), ruta); err != vechttp.ErrAccesoRutaExactaDenegado {
		t.Fatalf("una capacidad no sellada por la frontera debe denegarse: %v", err)
	}
	if err := autoridad.AutorizarRutaExacta(conCapacidad(sello, ruta), ruta); err != nil {
		t.Fatalf("la capacidad sellada de la ruta debe admitirse: %v", err)
	}
}
