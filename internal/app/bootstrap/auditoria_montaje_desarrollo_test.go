package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/auditoria"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type autoridadRutaAuditoriaSintetica struct{}

func (autoridadRutaAuditoriaSintetica) AutorizarRutaExacta(context.Context, string) error { return nil }

type registradorRutaAuditoriaSintetica struct{}

func (registradorRutaAuditoriaSintetica) RegistrarAuditoriaFronteraRutaExacta(context.Context, vecports.OrdenAuditoriaFronteraRutaExacta) error {
	return nil
}

type registradorFronteraSuperficiePrueba struct {
	llamadas int
	ultima   vecports.OrdenAuditoriaFronteraRutaExacta
	fallo    error
}

func (r *registradorFronteraSuperficiePrueba) RegistrarAuditoriaFronteraRutaExacta(_ context.Context, orden vecports.OrdenAuditoriaFronteraRutaExacta) error {
	r.llamadas++
	r.ultima = orden
	return r.fallo
}

func TestManejadorAuditoriaNoEmiteDenegacionSinApunteConfirmado(t *testing.T) {
	registrador := &registradorFronteraSuperficiePrueba{fallo: errors.New("sin apunte")}
	for _, codigo := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		h := manejadorAuditoriaDenegacionesLocales{registrador: registrador,
			siguiente: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", "17")
				http.Error(w, "detalle reservado", codigo)
			})}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, auditoria.RutaConsulta, nil))
		if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "detalle reservado") ||
			w.Result().Header.Get("Content-Length") != "" ||
			registrador.ultima.Validar() != nil {
			t.Fatalf("denegacion sin apunte confirmado: HTTP %d cuerpo=%q orden=%+v", w.Code, w.Body.String(), registrador.ultima)
		}
	}
	if registrador.llamadas != 2 {
		t.Fatalf("se esperaban dos intentos de apunte, hay %d", registrador.llamadas)
	}
}

func TestManejadorAuditoriaAcotaCuerpoDenegado(t *testing.T) {
	registrador := &registradorFronteraSuperficiePrueba{}
	h := manejadorAuditoriaDenegacionesLocales{registrador: registrador,
		siguiente: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", "4097")
			w.WriteHeader(http.StatusForbidden)
			if _, err := w.Write([]byte(strings.Repeat("x", 4097))); err != nil {
				t.Fatalf("escribir cuerpo de prueba: %v", err)
			}
		})}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, auditoria.RutaConsulta, nil))
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "xxxx") ||
		w.Result().Header.Get("Content-Length") != "" || registrador.llamadas != 1 ||
		registrador.ultima.Validar() != nil {
		t.Fatalf("cuerpo denegado excedido: HTTP %d cabeceras=%v orden=%+v", w.Code, w.Result().Header, registrador.ultima)
	}
}

func TestManejadorAuditoriaRegistra403LocalConActorVerificado(t *testing.T) {
	registrador := &registradorFronteraSuperficiePrueba{}
	h := manejadorAuditoriaDenegacionesLocales{registrador: registrador, siguiente: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		holder, _ := r.Context().Value(claveActorAuditoriaLocal{}).(*actorAuditoriaLocal)
		holder.fijar("per_actor_verificado")
		w.WriteHeader(http.StatusForbidden)
	})}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, auditoria.RutaConsulta, nil))
	if w.Code != http.StatusForbidden || registrador.llamadas != 1 ||
		registrador.ultima.Validar() != nil || registrador.ultima.ActorRef != "per_actor_verificado" ||
		registrador.ultima.Ruta != auditoria.RutaConsulta ||
		registrador.ultima.Superficie != vecports.SuperficieAuditoriaFronteraRutaExactaAuditoria ||
		w.Header().Get("X-Correlation-Ref") != registrador.ultima.CorrelacionRef {
		t.Fatalf("403 local sin bitácora correlacionable: estado=%d orden=%+v", w.Code, registrador.ultima)
	}
	registrador.llamadas = 0
	h.siguiente = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, auditoria.RutaOpciones, nil))
	if registrador.llamadas != 0 {
		t.Fatal("GET permitido registró denegación")
	}
}

func TestManejadorAuditoriaRegistra403TempranoConActorMTLSSellado(t *testing.T) {
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	registrador := &registradorFronteraSuperficiePrueba{}
	h := manejadorAuditoriaDenegacionesLocales{registrador: registrador, soporte: soporte,
		siguiente: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden) // RawQuery/Cookie/filtro: antes de ResolverContexto.
		})}
	for _, ruta := range []string{auditoria.RutaOpciones, auditoria.RutaConsulta} {
		ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, ruta)
		peticion := httptest.NewRequest(http.MethodPost, ruta+"?no-admitida=1", nil).WithContext(ctx)
		peticion.Header.Set("Cookie", "no-admitida=1")
		respuesta := httptest.NewRecorder()
		h.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusForbidden || registrador.ultima.Validar() != nil ||
			registrador.ultima.ActorRef != principal.ID || registrador.ultima.Ruta != ruta ||
			registrador.ultima.CorrelacionRef != respuesta.Header().Get("X-Correlation-Ref") {
			t.Fatalf("403 temprano sin actor/correlación mTLS: HTTP %d orden=%+v", respuesta.Code, registrador.ultima)
		}
	}
	if registrador.llamadas != 2 {
		t.Fatalf("una sola fila por petición denegada: %d", registrador.llamadas)
	}
}

type filaPreflightFronteraAuditoriaPrueba struct{ valida bool }

func (f filaPreflightFronteraAuditoriaPrueba) Scan(destino ...any) error {
	*destino[0].(*bool) = f.valida
	return nil
}

type consultadorPreflightFronteraAuditoriaPrueba struct {
	valida     bool
	consulta   string
	argumentos []any
}

func (q *consultadorPreflightFronteraAuditoriaPrueba) QueryRow(_ context.Context, consulta string, argumentos ...any) pgx.Row {
	q.consulta, q.argumentos = consulta, argumentos
	return filaPreflightFronteraAuditoriaPrueba{valida: q.valida}
}

func TestPreflightFronteraAuditoriaRechazaDerivaDeColumnasYCrear(t *testing.T) {
	q := &consultadorPreflightFronteraAuditoriaPrueba{valida: true}
	if err := preflightRegistradorFronteraAuditoriaConsultaDesarrollo(t.Context(), q); err != nil ||
		len(q.argumentos) != 1 || !strings.Contains(q.consulta, "has_any_column_privilege") ||
		!strings.Contains(q.consulta, "has_schema_privilege(session_user,'vec_contratacion_temporal','CREATE')") ||
		!strings.Contains(q.consulta, "has_schema_privilege('vec_contratacion_temporal_registrador_auditoria'::regrole,'vec_contratacion_temporal','USAGE')") ||
		!strings.Contains(q.consulta, "has_function_privilege('vec_contratacion_temporal_registrador_auditoria'::regrole,p.oid,'EXECUTE')") ||
		!strings.Contains(q.consulta, "x.oid<>p.oid") {
		t.Fatalf("preflight nominal incompleto: %v", err)
	}
	q.valida = false // PG18: GRANT SELECT(actor_ref), CREATE o REVOKE USAGE del grupo hace falsa la sonda.
	if err := preflightRegistradorFronteraAuditoriaConsultaDesarrollo(t.Context(), q); err == nil {
		t.Fatal("deriva de ACL admitida antes de publicar rutas")
	}
}

func TestRegistradorFronteraAuditoriaSeparaCTYAuditoria(t *testing.T) {
	ct, audit := &registradorFronteraSuperficiePrueba{}, &registradorFronteraSuperficiePrueba{}
	r := registradorFronterasPorSuperficieDesarrollo{ct: ct, auditoria: audit}
	orden := vecports.OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: "corr_no_disponible", Motivo: vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado,
		Superficie: vecports.SuperficieAuditoriaFronteraRutaExactaAuditoria,
		Ruta:       auditoria.RutaConsulta}
	if err := r.RegistrarAuditoriaFronteraRutaExacta(t.Context(), orden); err != nil || audit.llamadas != 1 || ct.llamadas != 0 {
		t.Fatalf("Auditoría cruzó registrador CT: %v ct=%d audit=%d", err, ct.llamadas, audit.llamadas)
	}
	orden.Superficie = vecports.SuperficieAuditoriaFronteraRutaExactaContratacionTemporal
	orden.Ruta = "/api/vec/contratacion-temporal/solicitudes"
	if err := r.RegistrarAuditoriaFronteraRutaExacta(t.Context(), orden); err != nil || audit.llamadas != 1 || ct.llamadas != 1 {
		t.Fatalf("CT cruzó registrador Audit: %v ct=%d audit=%d", err, ct.llamadas, audit.llamadas)
	}
	orden.Ruta = auditoria.RutaConsulta
	if err := r.RegistrarAuditoriaFronteraRutaExacta(t.Context(), orden); err == nil || audit.llamadas != 1 || ct.llamadas != 1 {
		t.Fatal("superficie CT aceptó ruta Audit")
	}
}

func TestAuditoriaNoReestablecePerfilRevocadoORestringido(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := soporte.reloj.Ahora()
	base := soporte.contexto.Resultado.Contexto
	semilla, err := instantaneaAuditoriaConsultaNominalDesarrollo(
		base.Principal.ID, base.PerfilActivoRef, "ct", "expediente:ct:sintetico:001",
		"revision_administrativa_auditoria_rrhh", ahora)
	if err != nil || semilla.Validar() != nil || !instantaneaAuditoriaConsultaVigenteExacta(semilla, semilla, ahora) {
		t.Fatalf("semilla nominal inválida: %v", err)
	}
	if !instantaneaInicialAuditoriaConsultaExacta(semilla, semilla) {
		t.Fatal("alta inicial exacta rechazada")
	}
	competidora := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
	competidora.AsignacionPerfil.Version = 2
	if competidora.Validar() != nil || instantaneaInicialAuditoriaConsultaExacta(competidora, semilla) {
		t.Fatal("una asignación aparecida durante la preparación se aceptó como alta inicial")
	}
	revocada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
	revocada.AsignacionPerfil.Estado = vecdomain.EstadoAsignacionPerfilRevocada
	revocada.AsignacionPerfil.RevocadaPor = "seguridad:desarrollo"
	revocada.AsignacionPerfil.RevocacionRef = "revocacion:auditoria:prueba"
	revocada.AsignacionPerfil.RevocadaEn = ahora
	if revocada.Validar() != nil || instantaneaAuditoriaConsultaVigenteExacta(revocada, semilla, ahora) {
		t.Fatal("una asignación revocada se consideró reiniciable")
	}
	restringida := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
	restringida.AsignacionPerfil.Ambitos[0].Valores[0] = "expediente:ct:otro:001"
	if restringida.Validar() != nil || instantaneaAuditoriaConsultaVigenteExacta(restringida, semilla, ahora) {
		t.Fatal("una asignación de ámbito ajeno se consideró exacta")
	}
	concesionAjena := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
	concesionAjena.VersionRol.Concesiones[0].Finalidades = []string{"otra_revision"}
	if concesionAjena.Validar() != nil || instantaneaAuditoriaConsultaVigenteExacta(concesionAjena, semilla, ahora) {
		t.Fatal("una concesión distinta se consideró exacta")
	}
}

func TestRaizExactaAuditoriaSirveOpcionesYDeniegaFuenteAjena(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	escenario := nuevoEscenarioMaterialRutasDietasPrueba(t, "dietas.ruta.catalogo.consultar", ahora)
	datos, err := escenario.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	identidadCT := &identidadAuditoriaConsultaPrueba{resuelta: auditoria.IdentidadResuelta{
		Vinculo: datos.VinculoAutenticacionActor, Resultado: escenario.resultado, Correlacion: datos.Correlacion}}
	identidadBolsa := &identidadAuditoriaConsultaPrueba{err: auditoria.ErrDenegada}
	opciones := auditoria.Opciones{FinalidadRef: "revision_administrativa_auditoria_rrhh",
		MotivoRef: escenario.motivo.Referencia(), PermisoRequerido: auditoria.AccionConsultar,
		Fuentes: []string{"ct", "bolsa"}, Motivo: escenario.motivo}
	rutas, err := nuevasRutasAuditoriaConsultaConIdentidadesRRHH(dependenciasIdentidadAuditoriaConsultaRRHH{
		PoolCT: &pgxpool.Pool{}, PoolBolsa: &pgxpool.Pool{},
		EmisorCT: &emisorAuditoriaConsultaPrueba{}, EmisorBolsa: &emisorAuditoriaConsultaPrueba{},
		IdentidadOpciones: identidadCT, IdentidadCT: identidadCT, IdentidadBolsa: identidadBolsa,
		Opciones: &opcionesAuditoriaConsultaPrueba{opciones: opciones},
		Intentos: configuracionIntentosConsultaPrueba(escenario.motivo),
	})
	if err != nil {
		t.Fatal(err)
	}
	registradorLocal := &registradorFronteraSuperficiePrueba{}
	for i := range rutas {
		rutas[i].Manejador = manejadorAuditoriaDenegacionesLocales{
			siguiente: rutas[i].Manejador, registrador: registradorLocal}
	}
	raiz, err := vechttp.NewHandlerSoloRutasExactas(rutas, autoridadRutaAuditoriaSintetica{}, registradorRutaAuditoriaSintetica{})
	if err != nil {
		t.Fatal(err)
	}
	get := httptest.NewRecorder()
	raiz.ServeHTTP(get, httptest.NewRequest(http.MethodGet, auditoria.RutaOpciones, nil))
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), opciones.FinalidadRef) {
		t.Fatalf("GET opciones en raíz = %d %s", get.Code, get.Body.String())
	}
	cuerpo, _ := json.Marshal(map[string]any{"fuente": "bolsa", "expediente_ref": "expediente:opaco:123",
		"desde": ahora.Add(-time.Hour).Format(time.RFC3339Nano), "hasta": ahora.Add(time.Hour).Format(time.RFC3339Nano),
		"limite": 1, "finalidad_ref": opciones.FinalidadRef, "motivo_ref": opciones.MotivoRef})
	peticion := httptest.NewRequest(http.MethodPost, auditoria.RutaConsulta, strings.NewReader(string(cuerpo)))
	peticion.Header.Set("Content-Type", "application/json")
	post := httptest.NewRecorder()
	raiz.ServeHTTP(post, peticion)
	if post.Code != http.StatusForbidden || registradorLocal.llamadas != 1 ||
		registradorLocal.ultima.Validar() != nil || registradorLocal.ultima.Ruta != auditoria.RutaConsulta {
		t.Fatalf("fuente Bolsa con identidad CT en raíz = HTTP %d orden=%+v", post.Code, registradorLocal.ultima)
	}
	registradorLocal.llamadas = 0
	var datosAjenos map[string]any
	if err := json.Unmarshal(cuerpo, &datosAjenos); err != nil {
		t.Fatal(err)
	}
	datosAjenos["fuente"], datosAjenos["motivo_ref"] = "ct", "motivo:ajeno"
	cuerpoAjeno, _ := json.Marshal(datosAjenos)
	peticionAjena := httptest.NewRequest(http.MethodPost, auditoria.RutaConsulta, strings.NewReader(string(cuerpoAjeno)))
	peticionAjena.Header.Set("Content-Type", "application/json")
	respuestaAjena := httptest.NewRecorder()
	raiz.ServeHTTP(respuestaAjena, peticionAjena)
	if respuestaAjena.Code != http.StatusForbidden || registradorLocal.llamadas != 1 ||
		registradorLocal.ultima.Validar() != nil {
		t.Fatalf("motivo ajeno sin bitácora local: HTTP %d orden=%+v", respuestaAjena.Code, registradorLocal.ultima)
	}
}
