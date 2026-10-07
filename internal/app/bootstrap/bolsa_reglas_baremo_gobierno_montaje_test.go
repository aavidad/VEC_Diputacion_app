package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	app "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoreglasbaremo"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func configGobiernoBaremoHTTPPrueba(t *testing.T) (config.Config, configuracionGobiernoReglasBaremoHTTPV3) {
	t.Helper()
	raiz := directorioTemporalFueraDeGitPrueba(t)
	if err := os.Chmod(raiz, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(raiz, "bolsa"), 0700); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment,
		DevelopmentGuard: config.DevelopmentGuardAcknowledgement, DevelopmentMaterialDir: raiz}
	c := configuracionGobiernoReglasBaremoHTTPV3{Esquema: "vec.bolsa.gobierno-reglas-baremo.configuracion.v3",
		MotivoIntentoDenegado: motivoCatalogoPlantillasCTDesarrollo(), MotivoIntentoError: motivoCatalogoPlantillasCTDesarrollo(), ConvocatoriaRef: "convocatoria:prueba", ExpedienteRef: "expediente:prueba", CatalogoMotivosID: "motivos_prueba",
		DSNFiles: map[string]string{"runtime": "runtime.conf", "fuente_autorizacion": "fuente.conf", "motivos_autorizacion": "motivos.conf"}}
	return cfg, c
}

type autoridadSesionRechazadaBaremoHTTPPrueba struct{ err error }

func (a autoridadSesionRechazadaBaremoHTTPPrueba) Credenciales(context.Context) (app.CredencialesGobiernoV3, error) {
	return app.CredencialesGobiernoV3{}, a.err
}

type operadorNoInvocadoBaremoHTTPPrueba struct {
	llamadas int
	err      error
}

func (o *operadorNoInvocadoBaremoHTTPPrueba) GuardarAltaBorrador(context.Context, app.CredencialesGobiernoV3, app.PeticionAltaBorradorV3) (bolsaports.ResultadoAltaBorradorReglasV3, error) {
	o.llamadas++
	return bolsaports.ResultadoAltaBorradorReglasV3{}, o.err
}
func (o *operadorNoInvocadoBaremoHTTPPrueba) ConsultarExacta(context.Context, app.CredencialesGobiernoV3, app.PeticionConsultaExactaV3) (bolsaports.ResultadoConsultaGobiernoReglasV3, error) {
	o.llamadas++
	return bolsaports.ResultadoConsultaGobiernoReglasV3{}, o.err
}
func (o *operadorNoInvocadoBaremoHTTPPrueba) RecuperarRecibo(context.Context, app.CredencialesGobiernoV3, app.PeticionRecuperarReciboV3) (bolsaports.ResultadoRecuperacionGobiernoReglasV3, error) {
	o.llamadas++
	return bolsaports.ResultadoRecuperacionGobiernoReglasV3{}, o.err
}

type auditorContextoBaremoHTTPPrueba struct {
	base auditorConsultaReciboPrueba
	ctx  context.Context
}

func (a *auditorContextoBaremoHTTPPrueba) RegistrarAuditoriaFronteraRutaExacta(ctx context.Context, o vecports.OrdenAuditoriaFronteraRutaExacta) error {
	a.ctx = ctx
	return a.base.RegistrarAuditoriaFronteraRutaExacta(ctx, o)
}

// Usa el callback común real con el puerto de prueba. No acredita una
// instalación SQL ni sustituye la sesión o el registro central del PDP.
func TestGobiernoBaremoHTTPAuditaSesionAntesPDPConMismoRegistrador(t *testing.T) {
	for _, par := range paresGobiernoReglasBaremoHTTPV3() {
		sesion, ctx := contextoSesionGobiernoBaremoHTTPPrueba(t, par.ruta)
		for _, caso := range []struct {
			err    error
			estado int
			motivo vecports.MotivoAuditoriaFronteraRutaExacta
		}{
			{app.ErrGobiernoV3NoAutenticado, 401, vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida},
			{app.ErrGobiernoV3Prohibido, 403, vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado},
		} {
			registrador := &auditorContextoBaremoHTTPPrueba{}
			operador := &operadorNoInvocadoBaremoHTTPPrueba{}
			h, err := bolsahttp.NuevoHandlerGobiernoReglasBaremoV3(autoridadSesionRechazadaBaremoHTTPPrueba{caso.err}, operador,
				auditorRechazoSesionGobiernoReglasBaremoHTTPV3(sesion, registrador))
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, par.ruta, nil).WithContext(ctx))
			if w.Code != caso.estado || operador.llamadas != 0 || len(registrador.base.ordenes) != 1 {
				t.Fatalf("rechazo sin callback o con negocio: %d", w.Code)
			}
			o := registrador.base.ordenes[0]
			if o.Validar() != nil || o.Ruta != par.ruta || o.Superficie != vecports.SuperficieAuditoriaFronteraRutaExactaBolsaReglasBaremo || o.Motivo != caso.motivo || o.ActorRef != "" {
				t.Fatalf("orden distinta del contrato nominal: %#v", o)
			}
			frontera, ok := fronteraSeguridadComunDesdeContexto(registrador.ctx)
			original, _ := fronteraSeguridadComunDesdeContexto(ctx)
			if !ok || !frontera.catalogo.mismaInstancia(original.catalogo) || !reflect.DeepEqual(registrador.ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}), ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{})) {
				t.Fatal("callback reconstruyó contexto de identidad")
			}
		}
	}
}

func TestGobiernoBaremoHTTPAuditorAusenteOCaido503SinNegocio(t *testing.T) {
	for _, par := range paresGobiernoReglasBaremoHTTPV3() {
		sesion, ctx := contextoSesionGobiernoBaremoHTTPPrueba(t, par.ruta)
		for _, registrador := range []vecports.RegistradorAuditoriaFronteraRutaExacta{nil, &auditorConsultaReciboPrueba{err: errors.New("detalle privado de auditoría caída")}} {
			operador := &operadorNoInvocadoBaremoHTTPPrueba{}
			h, err := bolsahttp.NuevoHandlerGobiernoReglasBaremoV3(autoridadSesionRechazadaBaremoHTTPPrueba{app.ErrGobiernoV3Prohibido}, operador,
				auditorRechazoSesionGobiernoReglasBaremoHTTPV3(sesion, registrador))
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, par.ruta, nil).WithContext(ctx))
			if w.Code != 503 || operador.llamadas != 0 {
				t.Fatal("auditor ausente o caído permitió continuar")
			}
		}
	}
}

func TestGobiernoBaremoHTTPCallbackRechazaContextoAjenoSinRegistrar(t *testing.T) {
	par := paresGobiernoReglasBaremoHTTPV3()[0]
	sesion, ctx := contextoSesionGobiernoBaremoHTTPPrueba(t, par.ruta)
	frontera, _ := fronteraSeguridadComunDesdeContexto(ctx)
	ajeno, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{frontera.descriptor})
	if err != nil {
		t.Fatal(err)
	}
	frontera.catalogo = ajeno
	registrador := &auditorConsultaReciboPrueba{}
	callback := auditorRechazoSesionGobiernoReglasBaremoHTTPV3(sesion, registrador)
	for _, caso := range []struct {
		ctx  context.Context
		ruta string
	}{
		{context.Background(), par.ruta},
		{context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, frontera), par.ruta},
		{ctx, par.ruta + "/ajena"},
	} {
		if err := callback(caso.ctx, caso.ruta, app.ErrGobiernoV3Prohibido); !errors.Is(err, app.ErrGobiernoV3NoDisponible) || len(registrador.ordenes) != 0 {
			t.Fatal("callback aceptó identidad reconstruida o frontera ajena")
		}
	}
}

func escribirConfigGobiernoBaremoHTTPPrueba(t *testing.T, cfg config.Config, c any) string {
	t.Helper()
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(cfg.DevelopmentMaterialDir, archivoGobiernoReglasBaremoHTTPV3)
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func TestGobiernoBaremoHTTPConfiguracionPrivadaOpcionalSinProvisionImplicita(t *testing.T) {
	cfg, c := configGobiernoBaremoHTTPPrueba(t)
	if obtenida, err := leerConfiguracionGobiernoReglasBaremoHTTPV3(cfg); err != nil || obtenida != nil {
		t.Fatal("ausencia no conservó familia apagada")
	}
	escribirConfigGobiernoBaremoHTTPPrueba(t, cfg, c)
	obtenida, err := leerConfiguracionGobiernoReglasBaremoHTTPV3(cfg)
	if err != nil || obtenida == nil || obtenida.ProvisionarPerfil || obtenida.AprobacionRef != "" || obtenida.PreimagenPerfilSHA256 != "" {
		t.Fatalf("configuración no cargada: err=%v valor=%#v dobleLlave=%v arbol=%v", err, obtenida, cfg.DevelopmentEnabledByDoubleKey(), validarArbolMaterialDesarrollo(cfg.DevelopmentMaterialDir))
	}
	base, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	m, err := prepararMontajeGobiernoReglasBaremoHTTPV3(config.Config{}, base, relojContratacionTemporalDesarrollo{})
	if err != nil || m.perfil != nil || m.configuracion != nil || m.perfilRef == "" {
		t.Fatal("declaración sin configuración inventó permiso")
	}
	rutas, cerrar, err := m.rutas(context.Background(), config.Config{}, nil, nil, catalogoFronterasComunDesarrollo{}, nil, relojContratacionTemporalDesarrollo{})
	if err != nil || len(rutas) != 3 || cerrar == nil {
		t.Fatal("familia ausente intentó construir dependencias")
	}
	cerrar()
}

func TestGobiernoBaremoHTTPConfiguracionRechazaFronterasPrivadasInvalidas(t *testing.T) {
	for _, caso := range []string{"sin_doble_llave", "modo_publico", "enlace", "en_repositorio", "pool_extra", "dsn_escape", "dsn_alias", "aprobacion_sin_provision", "huella_sin_aprobacion", "campo_actor"} {
		t.Run(caso, func(t *testing.T) {
			cfg, c := configGobiernoBaremoHTTPPrueba(t)
			if caso == "pool_extra" {
				c.DSNFiles["gobierno"] = "gobierno.conf"
			}
			if caso == "dsn_escape" {
				c.DSNFiles["runtime"] = "../runtime.conf"
			}
			if caso == "dsn_alias" {
				c.DSNFiles["runtime"] = c.DSNFiles["fuente_autorizacion"]
			}
			if caso == "aprobacion_sin_provision" {
				c.AprobacionRef = "aprobacion:prueba"
			}
			if caso == "huella_sin_aprobacion" {
				c.ProvisionarPerfil = true
				c.PreimagenPerfilSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			}
			ruta := escribirConfigGobiernoBaremoHTTPPrueba(t, cfg, c)
			switch caso {
			case "sin_doble_llave":
				cfg.DevelopmentGuard = ""
			case "modo_publico":
				if err := os.Chmod(ruta, 0644); err != nil {
					t.Fatal(err)
				}
			case "en_repositorio":
				if err := os.Mkdir(filepath.Join(cfg.DevelopmentMaterialDir, ".git"), 0700); err != nil {
					t.Fatal(err)
				}
			case "enlace":
				target := filepath.Join(cfg.DevelopmentMaterialDir, "original.json")
				if err := os.Rename(ruta, target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, ruta); err != nil {
					t.Fatal(err)
				}
			case "campo_actor":
				b, err := json.Marshal(c)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(ruta, append([]byte(`{"actor_ref":"actor:cliente",`), b[1:]...), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := leerConfiguracionGobiernoReglasBaremoHTTPV3(cfg); err == nil {
				t.Fatal("aceptó configuración privada incompatible")
			}
		})
	}
}
