package interna

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	"vec-diputacion-granada/internal/app/server"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestNuevaAplicacionSeguimientoNoAbreSinProveedorInstitucional(t *testing.T) {
	reserva, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	direccion := reserva.Addr().String()
	if err := reserva.Close(); err != nil {
		t.Fatal(err)
	}
	cfg := configuracionInternaValidaPrueba()
	cfg.DireccionEscucha = direccion
	cfg.RedesPermitidas = []string{"127.0.0.0/8"}
	aplicacion, err := NuevaAplicacion(context.Background(), cfg)
	if aplicacion != nil || !errors.Is(err, ErrDependenciasProductivasNoDisponibles) {
		t.Fatalf("aplicacion = (%v, %v)", aplicacion, err)
	}
	var faltantes *ErrorDependenciasFaltantes
	if !errors.As(err, &faltantes) || !reflect.DeepEqual(faltantes.Faltantes(), dependenciasConsultaSeguimiento[:]) {
		t.Fatalf("inventario = %v", err)
	}
	comprobacion, err := net.Listen("tcp", direccion)
	if err != nil {
		t.Fatalf("la composición reservó el puerto sin proveedores: %v", err)
	}
	_ = comprobacion.Close()
}

type recursoSeguimientoPrueba struct {
	propiedad atomic.Bool
	cierres   atomic.Int32
}

func (r *recursoSeguimientoPrueba) reclamarPropiedad() bool {
	return r.propiedad.CompareAndSwap(false, true)
}
func (r *recursoSeguimientoPrueba) cerrar() error {
	r.cierres.Add(1)
	return nil
}

func TestComposicionSeguimientoIncompletaLiberaRecursos(t *testing.T) {
	recurso := &recursoSeguimientoPrueba{}
	cfg := configuracionInternaValidaPrueba()
	cargado := false
	aplicacion, err := nuevaAplicacionConsultaSeguimiento(context.Background(), cfg,
		func(context.Context, Configuracion) (proveedoresConsultaSeguimiento, error) {
			cargado = true
			return proveedoresConsultaSeguimiento{recursos: []recursoCerrableAplicacionInterna{recurso}}, nil
		})
	if !cargado || aplicacion != nil || !errors.Is(err, ErrDependenciasProductivasNoDisponibles) || recurso.cierres.Load() != 1 {
		t.Fatalf("composición parcial = (%v, %v), cargado=%t, cierres=%d", aplicacion, err, cargado, recurso.cierres.Load())
	}
}

func TestPuenteSeguimientoNoDelegaSinIdentidadSellada(t *testing.T) {
	llamadas := 0
	puente := &puenteConsultaSeguimiento{api: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		llamadas++
	})}
	for _, ruta := range []string{httpct.RutaConsultaSeguimientoV2, "/api/vec/admin", "/api/vec/publica"} {
		r := httptest.NewRequest(http.MethodGet, ruta, nil)
		respuesta := httptest.NewRecorder()
		puente.ServeHTTP(respuesta, r)
		if respuesta.Code != http.StatusServiceUnavailable || respuesta.Body.Len() != 0 ||
			respuesta.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("ruta %s sin identidad: %d, cuerpo=%q", ruta, respuesta.Code, respuesta.Body.String())
		}
	}
	if llamadas != 0 {
		t.Fatalf("la API recibió %d llamadas sin identidad", llamadas)
	}
}

type extractorAsercionSeguimientoPrueba struct {
	asercion []byte
	err      error
	llamadas int
}

func (e *extractorAsercionSeguimientoPrueba) ExtraerAsercionProtegida(*http.Request) ([]byte, error) {
	e.llamadas++
	return append([]byte(nil), e.asercion...), e.err
}

type auditoriaDenegacionSeguimientoPrueba struct {
	ordenes      []vecports.OrdenAuditoriaFronteraRutaExacta
	contextoVivo bool
	plazoAcotado bool
}

func (a *auditoriaDenegacionSeguimientoPrueba) RegistrarAuditoriaFronteraRutaExacta(ctx context.Context, orden vecports.OrdenAuditoriaFronteraRutaExacta) error {
	a.ordenes = append(a.ordenes, orden)
	a.contextoVivo = ctx.Err() == nil
	limite, existe := ctx.Deadline()
	a.plazoAcotado = existe && time.Until(limite) > 0 && time.Until(limite) <= plazoAuditoriaDenegacionSeguimiento
	return nil
}

func TestPuenteSeguimientoAuditaDenegacionSinDelegarNiExponerDatos(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		asercion []byte
		err      error
	}{
		{nombre: "extraccion", err: errors.New("detalle privado del proveedor")},
		{nombre: "autenticacion", asercion: []byte("asercion-privada-sintetica")},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			extractor := &extractorAsercionSeguimientoPrueba{asercion: caso.asercion, err: caso.err}
			auditoria := &auditoriaDenegacionSeguimientoPrueba{}
			llamadasAPI := 0
			puente := &puenteConsultaSeguimiento{
				extractor: extractor,
				auditoria: auditoria,
				api: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					llamadasAPI++
					_, _ = w.Write([]byte("dato-protegido"))
				}),
			}
			puente.fachada.Store(&FachadaIdentidadOffline{})
			ctx, cancelar := context.WithCancel(context.Background())
			cancelar()
			ruta := httpct.RutaConsultaSeguimientoV2 + "?secreto=valor-privado"
			respuesta := httptest.NewRecorder()
			puente.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, ruta, strings.NewReader("cuerpo-privado")).WithContext(ctx))
			if respuesta.Code != http.StatusUnauthorized || respuesta.Body.Len() != 0 || llamadasAPI != 0 ||
				respuesta.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("denegación: estado=%d cuerpo=%q llamadasAPI=%d", respuesta.Code, respuesta.Body.String(), llamadasAPI)
			}
			if extractor.llamadas != 1 || len(auditoria.ordenes) != 1 || !auditoria.contextoVivo || !auditoria.plazoAcotado {
				t.Fatalf("auditoría: extracciones=%d órdenes=%d contextoVivo=%t plazoAcotado=%t", extractor.llamadas, len(auditoria.ordenes), auditoria.contextoVivo, auditoria.plazoAcotado)
			}
			orden := auditoria.ordenes[0]
			if orden.Validar() != nil || orden.Ruta != httpct.RutaConsultaSeguimientoV2 || orden.ActorRef != "" ||
				orden.Superficie != vecports.SuperficieAuditoriaFronteraRutaExactaContratacionTemporal ||
				orden.Motivo != vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida ||
				strings.Contains(orden.CorrelacionRef, "privado") {
				t.Fatalf("orden de auditoría inválida: %#v", orden)
			}
		})
	}
}

func TestPuenteSeguimientoNoAuditaOtraRuta(t *testing.T) {
	extractor := &extractorAsercionSeguimientoPrueba{asercion: []byte("asercion-privada-sintetica")}
	auditoria := &auditoriaDenegacionSeguimientoPrueba{}
	puente := &puenteConsultaSeguimiento{extractor: extractor, auditoria: auditoria, api: http.NotFoundHandler()}
	puente.fachada.Store(&FachadaIdentidadOffline{})
	respuesta := httptest.NewRecorder()
	puente.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, "/api/vec/admin?secreto=privado", nil))
	if respuesta.Code != http.StatusNotFound || respuesta.Body.Len() != 0 || extractor.llamadas != 0 || len(auditoria.ordenes) != 0 {
		t.Fatalf("ruta ajena: estado=%d cuerpo=%q extracciones=%d auditorías=%d", respuesta.Code, respuesta.Body.String(), extractor.llamadas, len(auditoria.ordenes))
	}
}

func TestServidorInternoSirvePortalFueraDelPuenteYAcotaAPICT(t *testing.T) {
	// staticHandler localiza los activos desde la raíz del repositorio; este
	// paquete vive un nivel más profundo que internal/app/server.
	t.Chdir("../../../..")
	extractor := &extractorAsercionSeguimientoPrueba{err: errors.New("sin aserción institucional")}
	auditoria := &auditoriaDenegacionSeguimientoPrueba{}
	llamadasAPI := 0
	puente := &puenteConsultaSeguimiento{
		extractor: extractor,
		auditoria: auditoria,
		api: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			llamadasAPI++
			_, _ = w.Write([]byte("dato-protegido"))
		}),
	}
	puente.fachada.Store(&FachadaIdentidadOffline{})
	servidor, err := server.NewHTTPServerInterno(config.Config{
		Address: "127.0.0.1:0", HTTPAllowedCIDRs: []string{"127.0.0.0/8"},
	}, puente)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		ruta   string
		estado int
	}{
		{ruta: "/portal-empleado/", estado: http.StatusOK},
		{ruta: "/portal-empleado/portal.js", estado: http.StatusOK},
		{ruta: "/assets/logo-diputacion-granada.svg", estado: http.StatusOK},
		{ruta: "/api/vec/admin", estado: http.StatusNotFound},
		{ruta: httpct.RutaConsultaSeguimientoV2, estado: http.StatusUnauthorized},
	} {
		peticion := httptest.NewRequest(http.MethodGet, caso.ruta, nil)
		peticion.RemoteAddr = "127.0.0.1:41000"
		respuesta := httptest.NewRecorder()
		servidor.Handler.ServeHTTP(respuesta, peticion)
		if respuesta.Code != caso.estado || strings.Contains(respuesta.Body.String(), "dato-protegido") {
			t.Fatalf("%s: estado=%d, cuerpo protegido=%t", caso.ruta, respuesta.Code, strings.Contains(respuesta.Body.String(), "dato-protegido"))
		}
	}
	if extractor.llamadas != 1 || llamadasAPI != 0 || len(auditoria.ordenes) != 1 ||
		auditoria.ordenes[0].Ruta != httpct.RutaConsultaSeguimientoV2 {
		t.Fatalf("separación: extracciones=%d, API=%d, auditorías=%d", extractor.llamadas, llamadasAPI, len(auditoria.ordenes))
	}
}

type fuentePeticionSeguimientoPrueba struct {
	peticion inc.PeticionAutoridad
	llamadas atomic.Int32
}

func (f *fuentePeticionSeguimientoPrueba) PeticionVerificada(context.Context) (inc.PeticionAutoridad, error) {
	f.llamadas.Add(1)
	return f.peticion, nil
}

func TestFuenteF1ExigeSesionInstitucionalVigenteYExacta(t *testing.T) {
	e := nuevoEntornoIdentidadOfflinePrueba(t)
	fuente := &fuentePeticionSeguimientoPrueba{}
	guardada := fuenteAutoridadIdentidadVinculada{identidad: e.servicio, siguiente: fuente}
	if _, err := guardada.PeticionVerificada(context.Background()); !errors.Is(err, inc.ErrAutoridadAplicacion) || fuente.llamadas.Load() != 0 {
		t.Fatalf("fuente sin cápsula: error=%v, llamadas=%d", err, fuente.llamadas.Load())
	}
	e.ejecutarEnC4(t, func(ctx context.Context) {
		vinculado, err := e.fachada.AutenticarYVincular(ctx, []byte("asercion-institucional-protegida"))
		if err != nil {
			t.Fatalf("vincular identidad: %v", err)
		}
		cuenta, auditoria, err := e.servicio.ExtraerCapsulaIdentidadPeticion(vinculado)
		if err != nil {
			t.Fatalf("extraer identidad: %v", err)
		}
		fuente.peticion.Contexto.Cuenta = cuenta
		fuente.peticion.Contexto.PerfilActivoRef = "prf_0123456789abcdefghijkl"
		fuente.peticion.Autenticacion.AutenticacionRef = auditoria.AutenticacionRef()
		fuente.peticion.Autenticacion.SesionRef = auditoria.SesionRef()
		if _, err := guardada.PeticionVerificada(vinculado); err != nil {
			t.Fatalf("fuente ligada a la sesión: %v", err)
		}
		fuente.peticion.Autenticacion.SesionRef = "ses_abcdefghijkl0123456789"
		if _, err := guardada.PeticionVerificada(vinculado); !errors.Is(err, inc.ErrAutoridadAplicacion) {
			t.Fatalf("sesión sustituida: %v", err)
		}
		e.registro.errorRevalidacion = errors.New("revocación sintética")
		antes := fuente.llamadas.Load()
		if _, err := guardada.PeticionVerificada(vinculado); !errors.Is(err, inc.ErrAutoridadAplicacion) || fuente.llamadas.Load() != antes {
			t.Fatalf("sesión revocada: error=%v, llamadas=%d", err, fuente.llamadas.Load())
		}
	})
}
