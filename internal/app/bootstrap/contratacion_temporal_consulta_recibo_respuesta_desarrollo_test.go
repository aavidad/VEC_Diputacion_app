package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type auditorConsultaReciboPrueba struct {
	ordenes []puertosvec.OrdenAuditoriaFronteraRutaExacta
	err     error
}

type consultorReciboRespuestaDenegadoPrueba struct{}

func (consultorReciboRespuestaDenegadoPrueba) Consultar(context.Context, ports.SolicitudConsultaReciboRespuesta) (ports.ReciboRespuestaConsultado, error) {
	return ports.ReciboRespuestaConsultado{}, ports.ErrConsultaReciboRespuestaDenegada
}

type consultorReciboRespuestaSesionPrueba struct {
	proveedor *proveedorConsultaReciboRespuestaDesarrollo
}

func (c consultorReciboRespuestaSesionPrueba) Consultar(ctx context.Context, s ports.SolicitudConsultaReciboRespuesta) (ports.ReciboRespuestaConsultado, error) {
	_, err := c.proveedor.AutorizarConsultaReciboRespuesta(ctx, s)
	return ports.ReciboRespuestaConsultado{}, err
}

func (a *auditorConsultaReciboPrueba) RegistrarAuditoriaFronteraRutaExacta(_ context.Context, orden puertosvec.OrdenAuditoriaFronteraRutaExacta) error {
	a.ordenes = append(a.ordenes, orden)
	return a.err
}

func TestConsultaReciboRespuestaMontajeLigaPermisoPropioYCamposExactos(t *testing.T) {
	s := ports.SolicitudConsultaReciboRespuesta{OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, ExpedienteRef: "expediente:sintetico:001", ComunicacionRef: "comunicacion:sintetica:001"}
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaConsultaReciboRespuesta)
	ctx = context.WithValue(ctx, claveConsultaReciboRespuestaDesarrollo{}, s)
	recurso, err := postgresct.RecursoConsultaReciboRespuesta(s)
	if err != nil {
		t.Fatal(err)
	}
	d := dominiovec.DatosSolicitudAutorizacionLigadaV3{
		Finalidad: "gestionar_contratacion_temporal", ReferenciaMotivo: motivoConsultaReciboRespuestaDesarrollo(),
		Accion: postgresct.AccionConsultaReciboRespuesta, Recurso: recurso,
	}
	if !solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaConsultaReciboRespuesta, d) {
		t.Fatal("GET válido denegado")
	}
	for _, cambiar := range []func(*dominiovec.DatosSolicitudAutorizacionLigadaV3){
		func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Accion = postgresct.AccionRegistroRespuestaRecibida
		},
		func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.ReferenciaMotivo = motivoRespuestaRecibidaDesarrollo()
		},
		func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.Recurso.Referencia = "comunicacion:ajena" },
		func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Ambitos = map[string]string{"organizacion_ref": "organizacion:ajena", "expediente_ref": s.ExpedienteRef}
		},
		func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Ambitos = map[string]string{"organizacion_ref": s.OrganizacionRef, "expediente_ref": "expediente:ajeno"}
		},
		func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos = map[string]string{"material_sha256": strings.Repeat("f", 64)}
		},
	} {
		otro := d
		otro.Recurso.Ambitos = map[string]string{"organizacion_ref": s.OrganizacionRef, "expediente_ref": s.ExpedienteRef}
		otro.Recurso.Atributos = map[string]string{"material_sha256": d.Recurso.Atributos["material_sha256"]}
		cambiar(&otro)
		if solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaConsultaReciboRespuesta, otro) {
			t.Fatal("permiso o material ajeno admitido")
		}
	}
	if solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaRegistroRespuestaRecibida, d) ||
		!esRutaContratacionTemporalDesarrollo(httptest.NewRequest(http.MethodGet, httpinterno.RutaConsultaReciboRespuesta, nil)) {
		t.Fatal("la ruta GET reutiliza POST o queda fuera de mTLS")
	}
	concesion := concesionConsultaReciboRespuestaDesarrollo()
	instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
		soporte.principalID, soporte.contexto.Resultado.Contexto.PerfilActivoRef, time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC),
		"consulta_recibo_respuesta_rrhh_desarrollo", "Consulta de recibo de respuesta RRHH", "consulta-recibo-respuesta-rrhh-desarrollo",
		[]dominiovec.ConcesionRol{concesion},
		[]dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
	if err != nil || instantanea.Validar() != nil {
		t.Fatal(err)
	}
	concesion = instantanea.VersionRol.Concesiones[0]
	if !reflect.DeepEqual(concesion.CamposPermitidos, []string{"auditoria_ref", "comunicacion_ref", "estado", "expediente_ref", "justificante_ref", "organizacion_ref", "recibo_ref", "registrada_en", "respuesta"}) ||
		len(concesion.Obligaciones) != 0 || concesion.Accion != postgresct.AccionConsultaReciboRespuesta {
		t.Fatal("concesión GET divergente")
	}
}

func TestConsultaReciboRespuestaMontajeExigeMTLSVigenteYAutorizacionNueva(t *testing.T) {
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	s := ports.SolicitudConsultaReciboRespuesta{OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, ExpedienteRef: "expediente:sintetico:001", ComunicacionRef: "comunicacion:sintetica:001"}
	ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaConsultaReciboRespuesta)
	a := &autorizacionComunicacionDesarrolloPrueba{}
	p := &proveedorConsultaReciboRespuestaDesarrollo{soporte: soporte, autorizador: a, reloj: relojFijoAltaContratacionTemporalDesarrollo{ahora: ahora}}
	if _, err := p.AutorizarConsultaReciboRespuesta(ctx, s); !errors.Is(err, ports.ErrConsultaReciboRespuestaDenegada) || a.llamadas != 0 {
		t.Fatal("certificado sin vigencia admitido")
	}
	capacidad, ok := soporte.capacidadValida(ctx)
	if !ok {
		t.Fatal("capacidad de prueba inválida")
	}
	capacidad.certificadoVerificadoEn = ahora.Add(-time.Minute)
	capacidad.certificadoValidoHasta = ahora.Add(time.Minute)
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	for n := 1; n <= 2; n++ {
		if _, err := p.AutorizarConsultaReciboRespuesta(ctx, s); !errors.Is(err, ports.ErrConsultaReciboRespuestaFallo) || a.llamadas != n || a.accion != postgresct.AccionConsultaReciboRespuesta {
			t.Fatal("GET omitió la autorización fresca o aceptó material vacío")
		}
	}
	s.ComunicacionRef = "comunicacion:ajena"
	if _, err := p.AutorizarConsultaReciboRespuesta(context.Background(), s); !errors.Is(err, ports.ErrConsultaReciboRespuestaDenegada) || a.llamadas != 2 {
		t.Fatal("referencia de query sustituyó identidad mTLS")
	}
}

func TestConsultaReciboRespuestaMontajeDistingueDenegacionDeDependenciaCaida(t *testing.T) {
	if !errors.Is(errorAutorizacionConsultaReciboRespuesta(dominiovec.ErrAutorizacionDenegada), ports.ErrConsultaReciboRespuestaDenegada) {
		t.Fatal("concesión ausente no denegada")
	}
	for _, causa := range []error{ports.ErrPersistenciaNoDisponible, puertosvec.ErrFuenteAutorizacionNoDisponible,
		puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible, puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible} {
		if !errors.Is(errorAutorizacionConsultaReciboRespuesta(errors.Join(dominiovec.ErrAutorizacionDenegada, causa)), ports.ErrConsultaReciboRespuestaFallo) {
			t.Fatalf("dependencia %v presentada como 403", causa)
		}
	}
}

func TestConsultaReciboRespuestaMontajeRevalidadorRealDistingueCaidaDeRevocacion(t *testing.T) {
	for _, caso := range []struct {
		nombre     string
		fallo      error
		estado     int
		auditorias int
	}{
		{"dependencia_caida", puertosvec.ErrRevalidacionAutenticacionActorNoDisponible, http.StatusServiceUnavailable, 0},
		{"sesion_revocada", dominiovec.ErrAutenticacionRevalidadaInvalida, http.StatusForbidden, 1},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevaSesionConsultaPrueba(t)
			e.revalidador.err = caso.fallo
			e.soporte.sesionOperativa = e.p
			ctx := contextoRutaCoberturaDesarrolloPrueba(e.soporte, e.principal, httpinterno.RutaConsultaReciboRespuesta)
			capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
			capacidad.certificadoVerificadoEn = e.reloj.Ahora().Add(-time.Second)
			capacidad.certificadoValidoHasta = e.reloj.Ahora().Add(time.Minute)
			ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
			ctx, err := puertosvec.ConCorrelacionIncidenciasPeticion(ctx)
			if err != nil {
				t.Fatal(err)
			}
			autorizador := &autorizadorLlamamientoDesarrollo{
				alta:     &dependenciasAltaContratacionTemporalDesarrollo{soporte: e.soporte},
				material: &proveedorMaterialAltaContratacionTemporalDesarrollo{}, consultaReciboRespuesta: true,
			}
			proveedor := &proveedorConsultaReciboRespuestaDesarrollo{soporte: e.soporte, autorizador: autorizador, reloj: e.reloj}
			manejador, err := httpinterno.NuevoManejadorConsultaReciboRespuesta(consultorReciboRespuestaSesionPrueba{proveedor})
			if err != nil {
				t.Fatal(err)
			}
			registro := &auditorConsultaReciboPrueba{}
			ruta := httpinterno.RutaConsultaReciboRespuesta
			h := auditorConsultaReciboRespuestaDenegada{registrador: registro, soporte: e.soporte, ruta: ruta, siguiente: manejador}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
				ruta+"?organizacion_ref="+organizacionAltaContratacionTemporalDesarrollo+
					"&expediente_ref=expediente:sintetico:001&comunicacion_ref=comunicacion:sintetica:001", nil).WithContext(ctx))
			if w.Code != caso.estado || len(registro.ordenes) != caso.auditorias {
				t.Fatalf("fallo=%v: estado=%d auditorias=%d", caso.fallo, w.Code, len(registro.ordenes))
			}
			if caso.auditorias == 1 && (registro.ordenes[0].Motivo != puertosvec.MotivoAuditoriaFronteraRutaExactaAccesoDenegado ||
				registro.ordenes[0].ActorRef != e.principal.ID || registro.ordenes[0].Validar() != nil) {
				t.Fatalf("revocacion sin auditoria de denegacion: %#v", registro.ordenes)
			}
		})
	}
}

func TestConsultaReciboRespuestaMontaje401YAuditoriaFronteraSinQuery(t *testing.T) {
	ruta := httpinterno.RutaConsultaReciboRespuesta
	registro := &auditorConsultaReciboPrueba{}
	autoridad := &autoridadConsultasContratacionTemporalDesarrollo{sello: &selloConsultasContratacionTemporalDesarrollo{}, resolvedor: &resolvedorIdentidadDesarrollo{}}
	h, err := vechttp.NewHandlerSoloRutasExactas([]vechttp.RutaExacta{{Ruta: ruta, Manejador: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("401 llegó al GET")
	})}}, autoridad, registro)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, ruta+"?organizacion_ref=organizacion:sintetica&expediente_ref=expediente:sintetico&comunicacion_ref=comunicacion:sintetica", nil)
	w := httptest.NewRecorder()
	autoridad.proteger(h).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized || len(registro.ordenes) != 1 ||
		registro.ordenes[0].Motivo != puertosvec.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida ||
		registro.ordenes[0].Ruta != ruta || registro.ordenes[0].Validar() != nil {
		t.Fatalf("401/auditoría: %d %#v", w.Code, registro.ordenes)
	}
	// Una identidad presente sin la capacidad sellada de esta ruta recibe 403
	// antes del manejador y deja una orden de frontera separada de la query.
	capacidad := capacidadConsultaContratacionTemporalDesarrollo{
		sello: &selloConsultasContratacionTemporalDesarrollo{}, ruta: ruta,
	}
	r = r.WithContext(context.WithValue(r.Context(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || len(registro.ordenes) != 2 ||
		registro.ordenes[1].Motivo != puertosvec.MotivoAuditoriaFronteraRutaExactaAccesoDenegado ||
		registro.ordenes[1].Ruta != ruta || registro.ordenes[1].Validar() != nil {
		t.Fatalf("403/auditoría: %d %#v", w.Code, registro.ordenes)
	}
}

func TestConsultaReciboRespuestaMontaje403AuditoriaSeparadaY404Uniforme(t *testing.T) {
	ruta := httpinterno.RutaConsultaReciboRespuesta
	const correlacion = "corr_1234567890abcdef1234567890abcdef"
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, ruta)
	for _, estado := range []int{http.StatusForbidden, http.StatusNotFound} {
		registro := &auditorConsultaReciboPrueba{}
		h := auditorConsultaReciboRespuestaDenegada{registrador: registro, soporte: soporte, siguiente: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(estado)
			_, _ = w.Write([]byte(`{"error":{"codigo":"recurso_no_encontrado","correlacion_ref":"` + correlacion + `"}}`))
		})}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta+"?comunicacion_ref=comunicacion:sintetica", nil).WithContext(ctx))
		if w.Code != estado || !strings.Contains(w.Body.String(), correlacion) {
			t.Fatalf("estado %d: %d %q", estado, w.Code, w.Body.String())
		}
		if estado == http.StatusForbidden {
			if len(registro.ordenes) != 1 || registro.ordenes[0].Validar() != nil ||
				registro.ordenes[0].Ruta != ruta || registro.ordenes[0].Motivo != puertosvec.MotivoAuditoriaFronteraRutaExactaAccesoDenegado ||
				registro.ordenes[0].CorrelacionRef != correlacion || registro.ordenes[0].ActorRef != principal.ID {
				t.Fatalf("403 sin auditoría minimizada: %#v", registro.ordenes)
			}
		} else if len(registro.ordenes) != 0 {
			t.Fatal("404 reveló información mediante auditoría de denegación")
		}
	}
	registroSinIdentidad := &auditorConsultaReciboPrueba{}
	hSinIdentidad := auditorConsultaReciboRespuestaDenegada{registrador: registroSinIdentidad, soporte: soporte, siguiente: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"correlacion_ref":"` + correlacion + `"}}`))
	})}
	wSinIdentidad := httptest.NewRecorder()
	hSinIdentidad.ServeHTTP(wSinIdentidad, httptest.NewRequest(http.MethodGet, ruta, nil))
	if wSinIdentidad.Code != http.StatusForbidden || len(registroSinIdentidad.ordenes) != 1 ||
		registroSinIdentidad.ordenes[0].ActorRef != "" || registroSinIdentidad.ordenes[0].CorrelacionRef != correlacion {
		t.Fatal("identidad no atestada filtrada en auditoría")
	}
	registro := &auditorConsultaReciboPrueba{err: errors.New("auditoria caída")}
	h := auditorConsultaReciboRespuestaDenegada{registrador: registro, siguiente: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"correlacion_ref":"` + correlacion + `"}}`))
	})}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "clave_i18n") || strings.Contains(w.Body.String(), correlacion) {
		t.Fatal("403 sin registro durable no se cerró")
	}
	registro.ordenes = nil
	h = auditorConsultaReciboRespuestaDenegada{registrador: registro, siguiente: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"correlacion_ref":"no-confiable"}}`))
	})}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
	if w.Code != http.StatusServiceUnavailable || len(registro.ordenes) != 0 {
		t.Fatal("correlación no confiable enviada a auditoría")
	}
}

func TestConsultaReciboRespuestaMontajeCorrelacionRealDeHandlerYActorSellado(t *testing.T) {
	ruta := httpinterno.RutaConsultaReciboRespuesta
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, ruta)
	get, err := httpinterno.NuevoManejadorConsultaReciboRespuesta(consultorReciboRespuestaDenegadoPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	registro := &auditorConsultaReciboPrueba{}
	h := auditorConsultaReciboRespuestaDenegada{registrador: registro, soporte: soporte, siguiente: get}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta+"?organizacion_ref="+organizacionAltaContratacionTemporalDesarrollo+"&expediente_ref=expediente:sintetico:001&comunicacion_ref=comunicacion:sintetica:001", nil).WithContext(ctx))
	var cuerpo struct {
		Error struct {
			CorrelacionRef string `json:"correlacion_ref"`
		} `json:"error"`
	}
	if w.Code != http.StatusForbidden || json.Unmarshal(w.Body.Bytes(), &cuerpo) != nil ||
		len(registro.ordenes) != 1 || registro.ordenes[0].Validar() != nil ||
		registro.ordenes[0].CorrelacionRef != cuerpo.Error.CorrelacionRef ||
		registro.ordenes[0].ActorRef != principal.ID {
		t.Fatalf("403/recibo auditoría desalineados: estado=%d ordenes=%#v", w.Code, registro.ordenes)
	}
}
