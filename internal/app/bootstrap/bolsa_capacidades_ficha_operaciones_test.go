package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	bolsapuertos "vec-diputacion-granada/internal/modules/bolsa/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type preparadorFichaOperacionesHTTPPrueba struct {
	q bolsapuertos.SolicitudCambiarSituacionParticipacion
}

func (p preparadorFichaOperacionesHTTPPrueba) PrepararSolicitudCambiarSituacion(_ context.Context, e bolsahttp.EntradaCambiarSituacionParticipacion) (bolsapuertos.SolicitudCambiarSituacionParticipacion, error) {
	q := p.q
	q.BolsaRef, q.ParticipacionRef = e.BolsaRef, e.ParticipacionRef
	return q, nil
}

type operadorFichaOperacionesHTTPPrueba struct{}

func (operadorFichaOperacionesHTTPPrueba) Operar(context.Context, bolsapuertos.SolicitudOperacionSituacion) (bolsapuertos.RegistroSituacionParticipacion, error) {
	return bolsapuertos.RegistroSituacionParticipacion{}, errors.New("no se ejecuta en GET")
}

func (operadorFichaOperacionesHTTPPrueba) ListarOperaciones(context.Context, bolsapuertos.SolicitudCambiarSituacionParticipacion) ([]bolsapuertos.RegistroOperacionSituacion, error) {
	return []bolsapuertos.RegistroOperacionSituacion{}, nil
}

type fuenteFichaOperacionesPrueba struct {
	instantanea vecdomain.InstantaneaAutorizacion
	err         error
	llamadas    int
}

func (f *fuenteFichaOperacionesPrueba) ObtenerInstantaneaAutorizacion(_ context.Context, _, _ string) (vecdomain.InstantaneaAutorizacion, error) {
	f.llamadas++
	return f.instantanea, f.err
}

func TestFichaOperacionesBolsaProyectaLoteF1DelGETLeido(t *testing.T) {
	directorio, soporteCT, principal, ahora := fixtureSoporteSesionBorradorBolsa(t)
	escribirManifiestoIdentidadBorradorBolsa(t, directorio, principal, ahora, nil)
	soporte, err := nuevoSoporteSesionBorradorBolsaDesarrollo(directorio, soporteCT, ahora)
	if err != nil {
		t.Fatal(err)
	}
	preparador := &preparadorBorradorLlamamientoDesarrollo{soporte: soporte}
	datos, err := soporte.soporteCanal.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		t.Fatal(err)
	}
	q := bolsapuertos.SolicitudCambiarSituacionParticipacion{Vinculo: soporte.soporteCanal.contexto.Vinculo,
		ResultadoContexto: soporte.soporteCanal.contexto.Resultado, BolsaRef: "bolsa:b2:desarrollo", ParticipacionRef: "participacion:b2",
		Destino: "disponible", Motivo: "consulta", ClaveIdempotencia: "consulta", Correlacion: correlacion,
		MotivoAutorizacion: motivoCambiarSituacionParticipacionBolsaDesarrollo()}
	if err := q.Validar(); err != nil {
		t.Fatal(err)
	}
	instante := ahora.Add(time.Minute)
	concesion, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
		datos.PrincipalID, datos.PerfilActivoRef, soporte.unidadRef, soporte.ambitoRef, ahora, 9)
	if err != nil {
		t.Fatal(err)
	}
	fuente := &fuenteFichaOperacionesPrueba{instantanea: concesion}
	reloj := relojFijoAltaContratacionTemporalDesarrollo{ahora: instante}
	documentales := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	p, err := nuevoProyectorDisponibilidadFichaOperacionesBolsa(preparador, fuente, reloj, documentales, nil)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := p.ProyectarDisponibilidadFichaOperaciones(context.Background(), q)
	if err != nil || fuente.llamadas != 1 || resultado.SolicitudesDocumentales.Estado != "disponible" ||
		resultado.ReincorporacionesTitular.Estado != "sin_montaje" ||
		resultado.SolicitudesDocumentales.BolsaRef != q.BolsaRef ||
		resultado.SolicitudesDocumentales.ParticipacionRef != q.ParticipacionRef {
		t.Fatalf("lote F1/montaje: %+v, llamadas=%d, err=%v", resultado, fuente.llamadas, err)
	}
	h, err := bolsahttp.NuevoHandlerOperacionesSituacion(preparadorFichaOperacionesHTTPPrueba{q: q}, operadorFichaOperacionesHTTPPrueba{}, p)
	if err != nil {
		t.Fatal(err)
	}
	peticion := httptest.NewRequest(http.MethodGet, bolsahttp.RutaBolsasGestion+"/"+q.BolsaRef+"/candidatos/"+q.ParticipacionRef+"/operaciones", nil)
	peticion.Header.Set("Accept", "application/json")
	respuesta := httptest.NewRecorder()
	h.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusOK || fuente.llamadas != 2 || !strings.Contains(respuesta.Body.String(), `"items":[]`) ||
		!strings.Contains(respuesta.Body.String(), `"solicitudes_documentales":{"estado":"disponible"`) {
		t.Fatalf("GET no conectó proyector F1: estado=%d, llamadas=%d, cuerpo=%s", respuesta.Code, fuente.llamadas, respuesta.Body.String())
	}
	// Una instantánea sin concesiones de lectura no hereda el permiso del GET B8.
	fuente.instantanea, err = nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
		datos.PrincipalID, datos.PerfilActivoRef, soporte.unidadRef, soporte.ambitoRef, ahora, 5)
	if err != nil {
		t.Fatal(err)
	}
	p, err = nuevoProyectorDisponibilidadFichaOperacionesBolsa(preparador, fuente, reloj, documentales, documentales)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err = p.ProyectarDisponibilidadFichaOperaciones(context.Background(), q)
	if err != nil || fuente.llamadas != 3 || resultado.SolicitudesDocumentales.Estado != "no_autorizado" || resultado.ReincorporacionesTitular.Estado != "no_autorizado" {
		t.Fatalf("denegación por recurso: %+v, llamadas=%d, err=%v", resultado, fuente.llamadas, err)
	}
	fuente.err = vecports.ErrFuenteAutorizacionNoDisponible
	resultado, err = p.ProyectarDisponibilidadFichaOperaciones(context.Background(), q)
	if err != nil || fuente.llamadas != 4 || resultado.SolicitudesDocumentales.Estado != "indisponible" || resultado.ReincorporacionesTitular.Estado != "indisponible" {
		t.Fatalf("fuente caída: %+v, llamadas=%d, err=%v", resultado, fuente.llamadas, err)
	}
	q.BolsaRef = "bolsa:ajena"
	if _, err := p.ProyectarDisponibilidadFichaOperaciones(context.Background(), q); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) || fuente.llamadas != 4 {
		t.Fatalf("bolsa ajena alcanzó la fuente: err=%v llamadas=%d", err, fuente.llamadas)
	}
	q.BolsaRef = "bolsa:b2:desarrollo"
	p, err = nuevoProyectorDisponibilidadFichaOperacionesBolsa(preparador, fuente, reloj, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err = p.ProyectarDisponibilidadFichaOperaciones(context.Background(), q)
	if err != nil || fuente.llamadas != 4 || resultado.SolicitudesDocumentales.Estado != "sin_montaje" || resultado.ReincorporacionesTitular.Estado != "sin_montaje" {
		t.Fatalf("dispatcher sin adjuntos: %+v, llamadas=%d, err=%v", resultado, fuente.llamadas, err)
	}
}
