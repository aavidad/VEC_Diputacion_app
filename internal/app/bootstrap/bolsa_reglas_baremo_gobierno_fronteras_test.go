package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"vec-diputacion-granada/config"

	app "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoreglasbaremo"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func contextoSesionGobiernoBaremoHTTPPrueba(t *testing.T, ruta string) (*proveedorSesionConsultaRRHHDesarrollo, context.Context) {
	t.Helper()
	p := perfilGobiernoReglasBaremoPrueba(t)
	fronteras, err := fronterasGobiernoReglasBaremoHTTPV3(p.PerfilRef())
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	f, ok := catalogo.resolver(http.MethodPost, ruta)
	if !ok {
		t.Fatal("ruta no declarada")
	}
	principal := vecdomain.Principal{ID: p.soporte.principalID, Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": p.soporte.certificadoSHA256}}
	capacidad := capacidadConsultaContratacionTemporalDesarrollo{sello: p.soporte.sello, ruta: ruta, metodo: http.MethodPost,
		principal: principal, contextoOperacion: &contextoOperacionCTDesarrollo{}}
	// La raíz real emite esta capacidad después de mTLS; aquí se prueba
	// exclusivamente el transporte nominal del contexto y su clasificación.
	ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	ctx = context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{metodo: http.MethodPost, ruta: ruta, superficie: superficieInternaSeguridadComunDesarrollo, catalogo: catalogo, descriptor: f})
	sesion := &proveedorSesionConsultaRRHHDesarrollo{soporte: p.soporte, fronteras: catalogo, base: p.soporte.contexto.Resultado}
	ctx, err = contextoIntentoGobiernoBaremoHTTPV3(ctx, "expediente:prueba")
	if err != nil {
		t.Fatal(err)
	}
	return sesion, ctx
}

func TestGobiernoBaremoHTTPFronterasRegistradasYCerradas(t *testing.T) {
	declaradas, err := fronterasGobiernoReglasBaremoHTTPV3("prf_baremo_prueba")
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(declaradas)
	if err != nil {
		t.Fatal(err)
	}
	politica := politicaDescriptoresCTPrueba(t)
	autorizaciones, err := autorizacionesGobiernoReglasBaremoHTTPV3(politica)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nuevoCatalogoAutorizacionComunDesarrollo(catalogo, autorizaciones); err != nil {
		t.Fatal(err)
	}
	rutas := (&montajeGobiernoReglasBaremoHTTPV3{}).indisponibles()
	if len(rutas) != 3 || len(declaradas) != 3 {
		t.Fatal("registro y declaración difieren")
	}
	for _, ruta := range rutas {
		f, ok := catalogo.resolver(http.MethodPost, ruta.Ruta)
		if !ok || len(f.PerfilesActivosRef) != 1 || f.PerfilesActivosRef[0] != "prf_baremo_prueba" || f.DetalleColeccion || len(f.PlantillaDetalle) != 0 {
			t.Fatal("familia no conserva frontera exacta")
		}
		for _, metodo := range []string{http.MethodGet, http.MethodHead, http.MethodPut} {
			if _, ok := catalogo.resolver(metodo, ruta.Ruta); ok {
				t.Fatal("otro método adquirió frontera")
			}
		}
		w := httptest.NewRecorder()
		ruta.Manejador.ServeHTTP(w, httptest.NewRequest(http.MethodPost, ruta.Ruta, nil))
		if w.Code != 503 {
			t.Fatalf("familia no compuesta: %d", w.Code)
		}
	}
	autorizaciones[1].Fronteras = []string{declaradas[0].Clave}
	if _, err := nuevoCatalogoAutorizacionComunDesarrollo(catalogo, autorizaciones); err == nil {
		t.Fatal("acción de consulta ligada al alta")
	}
}

func TestGobiernoBaremoHTTPSesionConservaCaidaYDenegacionSoloEnFronteraSellada(t *testing.T) {
	for _, par := range paresGobiernoReglasBaremoHTTPV3() {
		sesion, ctx := contextoSesionGobiernoBaremoHTTPPrueba(t, par.ruta)
		if !sesion.rutaSesionConIndisponibilidad(par.ruta, ctx) || sesion.rutaSesionConIndisponibilidad(par.ruta) {
			t.Fatal("excepción depende sólo del nombre de ruta")
		}
		for _, caso := range []struct{ err, espera error }{{errors.New("dependencia privada caída"), ctports.ErrConsultaRRHHNoDisponible}, {vecdomain.ErrAutorizacionDenegada, ErrSeguridadComunDesarrolloDenegada}, {ErrSeguridadComunDesarrolloDenegada, ErrSeguridadComunDesarrolloDenegada}} {
			err := sesion.errorSesionConsultaComunicacionesExpediente(ctx, caso.err)
			if !errors.Is(err, caso.espera) {
				t.Fatalf("categoría perdida: %v", err)
			}
		}
		frontera, _ := fronteraSeguridadComunDesdeContexto(ctx)
		otra, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{frontera.descriptor})
		if err != nil {
			t.Fatal(err)
		}
		for indice, mutar := range []func(*fronteraSeguridadComunDesarrollo){
			func(f *fronteraSeguridadComunDesarrollo) { f.catalogo = otra }, func(f *fronteraSeguridadComunDesarrollo) { f.metodo = http.MethodGet },
			func(f *fronteraSeguridadComunDesarrollo) { f.ruta = par.ruta + "/ajena" },
		} {
			f := frontera
			mutar(&f)
			ajeno := context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, f)
			if sesion.rutaSesionConIndisponibilidad(par.ruta, ajeno) || !errors.Is(sesion.errorSesionConsultaComunicacionesExpediente(ajeno, errors.New("caída")), ErrSeguridadComunDesarrolloDenegada) {
				t.Fatalf("frontera ajena adquirió clasificación nominal: caso=%d clasificador=%v error=%v", indice, sesion.rutaSesionConIndisponibilidad(par.ruta, ajeno), sesion.errorSesionConsultaComunicacionesExpediente(ajeno, errors.New("caída")))
			}
		}
		// El descriptor entregado se vuelve a leer del catálogo autoritativo;
		// cambiar una copia no añade una capacidad de publicación.
		copia := frontera
		copia.descriptor.ClaveCapacidad = "bolsa.reglas_baremo.publicar"
		normalizada, ok := fronteraSeguridadComunDesdeContexto(context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, copia))
		if !ok || normalizada.descriptor.ClaveCapacidad != par.accion {
			t.Fatal("copia de descriptor sustituyó la declaración gobernada")
		}
	}
}

func TestGobiernoBaremoHTTPBrokerConservaErrorCacheadoSinRecrearSesion(t *testing.T) {
	p := perfilGobiernoReglasBaremoPrueba(t)
	sesion, ctx := contextoSesionGobiernoBaremoHTTPPrueba(t, paresGobiernoReglasBaremoHTTPV3()[0].ruta)
	// El escenario necesita compartir exactamente el soporte y catálogo.
	p.soporte = sesion.soporte
	p.plantilla.AsignacionPerfil.PerfilActivoRef = sesion.base.Contexto.PerfilActivoRef
	proveedor := &ProveedorGobiernoReglasBaremoV3{perfil: p, sesion: sesion, pdp: &autorizadorAnalisisContratacionTemporalDesarrollo{},
		rutas: RutasGobiernoReglasBaremoV3{paresGobiernoReglasBaremoHTTPV3()[0].ruta, paresGobiernoReglasBaremoHTTPV3()[1].ruta, paresGobiernoReglasBaremoHTTPV3()[2].ruta}}
	capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	for _, err := range []error{app.ErrGobiernoV3NoDisponible, app.ErrGobiernoV3Prohibido} {
		capacidad.contextoOperacion = &contextoOperacionCTDesarrollo{soporte: p.soporte, err: err}
		ligado := context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
		if _, recibido := proveedor.Credenciales(ligado); !errors.Is(recibido, err) {
			t.Fatalf("broker mezcló503/403: %v", recibido)
		}
	}
}

func TestGobiernoBaremoHTTPBrokerCancelacionAutenticada503YAnonima401(t *testing.T) {
	for _, par := range paresGobiernoReglasBaremoHTTPV3() {
		sesion, autenticado := contextoSesionGobiernoBaremoHTTPPrueba(t, par.ruta)
		proveedor := &ProveedorGobiernoReglasBaremoV3{sesion: sesion,
			rutas: RutasGobiernoReglasBaremoV3{paresGobiernoReglasBaremoHTTPV3()[0].ruta, paresGobiernoReglasBaremoHTTPV3()[1].ruta, paresGobiernoReglasBaremoHTTPV3()[2].ruta}}
		for _, base := range []context.Context{autenticado, context.Background()} {
			esperado := app.ErrGobiernoV3NoAutenticado
			if base == autenticado {
				esperado = app.ErrGobiernoV3NoDisponible
			}
			cancelado, cancelar := context.WithCancel(base)
			cancelar()
			if _, err := proveedor.Credenciales(cancelado); !errors.Is(err, esperado) {
				t.Fatalf("cancelación perdió categoría: %v", err)
			}
			vencido, detener := context.WithDeadline(base, time.Now().Add(-time.Second))
			defer detener()
			if _, err := proveedor.Credenciales(vencido); !errors.Is(err, esperado) {
				t.Fatalf("deadline perdió categoría: %v", err)
			}
		}
	}
}
