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
	bolsas := &bolsasVigentesFalsas{vigentes: map[string]bool{}}
	bolsas.declarar("bolsa:b2:desarrollo", bolsapuertos.AmbitoBolsa{AmbitoRef: soporte.ambitoRef, UnidadRef: soporte.unidadRef})
	preparador := &preparadorBorradorLlamamientoDesarrollo{soporte: soporte, vigentes: bolsas}
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
	if !errors.Is(err, vecports.ErrFuenteAutorizacionNoDisponible) || fuente.llamadas != 4 || resultado.SolicitudesDocumentales.Estado != "" || resultado.ReincorporacionesTitular.Estado != "" {
		t.Fatalf("fuente caída: %+v, llamadas=%d, err=%v", resultado, fuente.llamadas, err)
	}
	respuesta = httptest.NewRecorder()
	h, err = bolsahttp.NuevoHandlerOperacionesSituacion(preparadorFichaOperacionesHTTPPrueba{q: q}, operadorFichaOperacionesHTTPPrueba{}, p)
	if err != nil {
		t.Fatal(err)
	}
	h.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusOK || fuente.llamadas != 5 || !strings.Contains(respuesta.Body.String(), `"items":[]`) ||
		!strings.Contains(respuesta.Body.String(), `"solicitudes_documentales":{"estado":"indisponible"`) {
		t.Fatalf("fallo de fuente destruyó historial o metadata segura: estado=%d, llamadas=%d, cuerpo=%s", respuesta.Code, fuente.llamadas, respuesta.Body.String())
	}
	q.BolsaRef = "bolsa:ajena"
	if _, err := p.ProyectarDisponibilidadFichaOperaciones(context.Background(), q); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) || fuente.llamadas != 5 {
		t.Fatalf("bolsa ajena alcanzó la fuente: err=%v llamadas=%d", err, fuente.llamadas)
	}
	q.BolsaRef = "bolsa:b2:desarrollo"
	p, err = nuevoProyectorDisponibilidadFichaOperacionesBolsa(preparador, fuente, reloj, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err = p.ProyectarDisponibilidadFichaOperaciones(context.Background(), q)
	if err != nil || fuente.llamadas != 5 || resultado.SolicitudesDocumentales.Estado != "sin_montaje" || resultado.ReincorporacionesTitular.Estado != "sin_montaje" {
		t.Fatalf("dispatcher sin adjuntos: %+v, llamadas=%d, err=%v", resultado, fuente.llamadas, err)
	}
	fuente.err = nil
	fuente.instantanea, err = nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
		datos.PrincipalID, datos.PerfilActivoRef, soporte.unidadRef, soporte.ambitoRef, ahora, 11)
	if err != nil {
		t.Fatal(err)
	}
	p, err = nuevoProyectorDisponibilidadFichaOperacionesBolsa(preparador, fuente, reloj, documentales, documentales)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err = p.ProyectarDisponibilidadFichaOperaciones(context.Background(), q)
	if err != nil || resultado.SolicitudesDocumentales.Estado != "disponible" || resultado.ReincorporacionesTitular.Estado != "disponible" {
		t.Fatalf("concesiones completas: %+v err=%v", resultado, err)
	}
	for i := range fuente.instantanea.VersionRol.Concesiones {
		if fuente.instantanea.VersionRol.Concesiones[i].Accion == bolsapuertos.AccionConsultarSolicitudesDocumentalesRRHH {
			fuente.instantanea.VersionRol.Concesiones[i].CamposPermitidos = []string{"campo_parcial"}
		}
	}
	resultado, err = p.ProyectarDisponibilidadFichaOperaciones(context.Background(), q)
	if err != nil || resultado.SolicitudesDocumentales.Estado != "no_autorizado" || resultado.ReincorporacionesTitular.Estado != "disponible" {
		t.Fatalf("campo documental parcial anunciado: %+v err=%v", resultado, err)
	}
	fuente.instantanea, err = nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
		datos.PrincipalID, datos.PerfilActivoRef, soporte.unidadRef, soporte.ambitoRef, ahora, 11)
	if err != nil {
		t.Fatal(err)
	}
	for i := range fuente.instantanea.VersionRol.Concesiones {
		if fuente.instantanea.VersionRol.Concesiones[i].Accion == bolsapuertos.AccionConsultarReincorporacionTitular {
			fuente.instantanea.VersionRol.Concesiones[i].CamposPermitidos = nil
		}
	}
	resultado, err = p.ProyectarDisponibilidadFichaOperaciones(context.Background(), q)
	if err != nil || resultado.SolicitudesDocumentales.Estado != "disponible" || resultado.ReincorporacionesTitular.Estado != "no_autorizado" {
		t.Fatalf("campo de reincorporación ausente anunciado: %+v err=%v", resultado, err)
	}
	fuente.instantanea, err = nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
		datos.PrincipalID, datos.PerfilActivoRef, soporte.unidadRef, soporte.ambitoRef, ahora, 11)
	if err != nil {
		t.Fatal(err)
	}
	for i := range fuente.instantanea.VersionRol.Concesiones {
		if fuente.instantanea.VersionRol.Concesiones[i].Accion == bolsapuertos.AccionConsultarReincorporacionTitular {
			fuente.instantanea.VersionRol.Concesiones[i].Obligaciones = []string{"control_pendiente"}
		}
	}
	resultado, err = p.ProyectarDisponibilidadFichaOperaciones(context.Background(), q)
	if err != nil || resultado.SolicitudesDocumentales.Estado != "disponible" || resultado.ReincorporacionesTitular.Estado != "no_autorizado" {
		t.Fatalf("obligación no ejecutada anunciada: %+v err=%v", resultado, err)
	}
}

// F1-60, prueba de aceptación: el actor tiene permiso en su unidad gestora y
// la decisión V3 se lo concede en una bolsa de esa unidad; la misma consulta
// sobre una bolsa declarada en otra unidad gestora se deniega, porque el
// recurso lleva la unidad de la bolsa (B144) y no la del perfil.
func TestFichaOperacionesBolsaDeOtraUnidadGestoraSeDeniega(t *testing.T) {
	directorio, soporteCT, principal, ahora := fixtureSoporteSesionBorradorBolsa(t)
	escribirManifiestoIdentidadBorradorBolsa(t, directorio, principal, ahora, nil)
	soporte, err := nuevoSoporteSesionBorradorBolsaDesarrollo(directorio, soporteCT, ahora)
	if err != nil {
		t.Fatal(err)
	}
	const propia, ajena = "bolsa:b2:desarrollo", "bolsa:otra-unidad:desarrollo"
	bolsas := &bolsasVigentesFalsas{vigentes: map[string]bool{}}
	bolsas.declarar(propia, bolsapuertos.AmbitoBolsa{AmbitoRef: soporte.ambitoRef, UnidadRef: soporte.unidadRef})
	bolsas.constituirEn(ajena, bolsapuertos.AmbitoBolsa{AmbitoRef: soporte.ambitoRef, UnidadRef: "unidad:desarrollo:otra-gestora"})
	const otroAmbito = "bolsa:otro-ambito:desarrollo"
	bolsas.constituirEn(otroAmbito, bolsapuertos.AmbitoBolsa{AmbitoRef: "ambito:desarrollo:otro", UnidadRef: soporte.unidadRef})
	preparador := &preparadorBorradorLlamamientoDesarrollo{soporte: soporte, vigentes: bolsas}
	datos, err := soporte.soporteCanal.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		t.Fatal(err)
	}
	q := bolsapuertos.SolicitudCambiarSituacionParticipacion{Vinculo: soporte.soporteCanal.contexto.Vinculo,
		ResultadoContexto: soporte.soporteCanal.contexto.Resultado, BolsaRef: propia, ParticipacionRef: "participacion:b2",
		Destino: "disponible", Motivo: "consulta", ClaveIdempotencia: "consulta", Correlacion: correlacion,
		MotivoAutorizacion: motivoCambiarSituacionParticipacionBolsaDesarrollo()}
	concesion, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
		datos.PrincipalID, datos.PerfilActivoRef, soporte.unidadRef, soporte.ambitoRef, ahora, 9)
	if err != nil {
		t.Fatal(err)
	}
	fuente := &fuenteFichaOperacionesPrueba{instantanea: concesion}
	documentales := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	p, err := nuevoProyectorDisponibilidadFichaOperacionesBolsa(preparador, fuente,
		relojFijoAltaContratacionTemporalDesarrollo{ahora: ahora.Add(time.Minute)}, documentales, nil)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := p.ProyectarDisponibilidadFichaOperaciones(context.Background(), q)
	if err != nil || resultado.SolicitudesDocumentales.Estado != "disponible" {
		t.Fatalf("bolsa de la propia unidad: %+v err=%v", resultado, err)
	}
	for _, bolsa := range []string{ajena, otroAmbito} {
		q.BolsaRef = bolsa
		resultado, err = p.ProyectarDisponibilidadFichaOperaciones(context.Background(), q)
		if err != nil || resultado.SolicitudesDocumentales.Estado != "no_autorizado" {
			t.Fatalf("bolsa %s de otra unidad o ámbito no denegada: %+v err=%v", bolsa, resultado, err)
		}
	}
}
