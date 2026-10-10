package administracion

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad/adminperfiles"
	identidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	"vec-diputacion-granada/internal/vec/application"
)

func comprobarClaseFallo(t *testing.T, err error, clase string) {
	t.Helper()
	if !errors.Is(err, ErrConfiguracion) || ClaseFallo(err) != clase {
		t.Fatalf("clase %q esperada; obtenido %q (%v)", clase, ClaseFallo(err), err)
	}
}

func TestClaseFalloNuevoServidor(t *testing.T) {
	valida, _, _, _ := materialServidorFronteraPrueba(t)
	casos := map[string]func(*Configuracion){
		"entorno":    func(c *Configuracion) { c.Entorno = "produccion" },
		"retirada":   func(c *Configuracion) { c.RetiradaEn = time.Time{} },
		"rutas_tls":  func(c *Configuracion) { c.CRLAdministracion = "" },
		"host":       func(c *Configuracion) { c.Host = "" },
		"superficie": func(c *Configuracion) { c.Audiencia = "" },
		"tls":        func(c *Configuracion) { c.CertificadoServidor = c.CertificadoServidor + ".ausente" },
		"ca":         func(c *Configuracion) { c.CAAdministracion = c.CertificadoServidor },
	}
	for clase, cambiar := range casos {
		t.Run(clase, func(t *testing.T) {
			cfg := valida
			cambiar(&cfg)
			servidor, err := NuevoServidor(cfg)
			if servidor != nil {
				t.Fatal("servidor montado con configuración inválida")
			}
			comprobarClaseFallo(t, err, clase)
		})
	}
}

func TestClaseFalloResolverSesion(t *testing.T) {
	valida, _, _, _ := materialServidorFronteraPrueba(t)
	reloj := new(relojComposicionPrueba)
	casos := map[string]func(*Configuracion, *adminperfiles.Dependencias){
		"entorno":         func(c *Configuracion, _ *adminperfiles.Dependencias) { c.Entorno = "" },
		"dependencias":    func(_ *Configuracion, d *adminperfiles.Dependencias) { d.Reloj = nil },
		"host":            func(c *Configuracion, _ *adminperfiles.Dependencias) { c.Host = "admin" },
		"ca":              func(c *Configuracion, _ *adminperfiles.Dependencias) { c.CAAdministracion = c.ClaveServidor },
		"red":             func(c *Configuracion, _ *adminperfiles.Dependencias) { c.RedesPermitidas = []string{"no-es-red"} },
		"resolver_sesion": func(*Configuracion, *adminperfiles.Dependencias) {},
	}
	for clase, cambiar := range casos {
		t.Run(clase, func(t *testing.T) {
			cfg, deps := valida, adminperfiles.Dependencias{Reloj: reloj}
			cambiar(&cfg, &deps)
			resolver, err := NuevoResolverSesionPerfiles(cfg, deps)
			if resolver != nil {
				t.Fatal("resolver creado con configuración inválida")
			}
			comprobarClaseFallo(t, err, clase)
		})
	}
}

// constructoresPruebaClase falla en el paso indicado; los anteriores devuelven
// valores vacíos que nunca se usan contra una base.
func constructoresPruebaClase(falla string, causa error) constructoresPerfiles {
	errorEn := func(paso string) error {
		if paso == falla {
			return causa
		}
		return nil
	}
	return constructoresPerfiles{
		registro: func(context.Context, *pgxpool.Pool, *pgxpool.Pool, identidad.SeudonimizadorAlta, string, string) (*identidad.RegistroSesionesPostgreSQL, error) {
			return new(identidad.RegistroSesionesPostgreSQL), errorEn("registro_sesiones")
		},
		revalidador: func(context.Context, *pgxpool.Pool) (*identidad.RevalidadorAutenticacionActorPostgreSQL, error) {
			return new(identidad.RevalidadorAutenticacionActorPostgreSQL), errorEn("revalidador")
		},
		cuentas: func(context.Context, *pgxpool.Pool, httpseguridad.Reloj, adminperfiles.FuenteIdentificadoresADMIN, adminperfiles.SeudonimizadorFuenteADMIN) (*adminperfiles.PostgreSQL, error) {
			return new(adminperfiles.PostgreSQL), errorEn("cuentas_is16")
		},
		contextos: func(context.Context, *pgxpool.Pool, httpseguridad.Reloj, adminperfiles.ConfiguracionContextoADMIN) (*application.AutoridadContextoActorRegistradoV2, error) {
			if falla == "resolver_sesion" {
				return nil, nil // el resolver rechaza el contexto nulo
			}
			return new(application.AutoridadContextoActorRegistradoV2), errorEn("contexto_ca36")
		},
	}
}

func TestClaseFalloComposicionPerfiles(t *testing.T) {
	valida, _, _, _ := materialServidorFronteraPrueba(t)
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	casos := []struct {
		clase   string
		ctx     context.Context
		cambiar func(*Configuracion, *DependenciasComposicionPerfiles)
	}{
		{"dependencias", cancelado, nil},
		{"dependencias_runtime", context.Background(), func(_ *Configuracion, d *DependenciasComposicionPerfiles) { d.PoolContextoADMIN = d.PoolCuentas }},
		{"registro_sesiones", context.Background(), nil},
		{"revalidador", context.Background(), nil},
		{"cuentas_is16", context.Background(), nil},
		{"contexto_ca36", context.Background(), nil},
		{"resolver_sesion", context.Background(), nil},
		// Un fallo con clase propia del resolver o del servidor se conserva.
		{"host", context.Background(), func(c *Configuracion, _ *DependenciasComposicionPerfiles) { c.Host = "admin" }},
		{"rutas_tls", context.Background(), func(c *Configuracion, _ *DependenciasComposicionPerfiles) { c.CRLAdministracion = "" }},
		{"montaje_lecturas", context.Background(), func(_ *Configuracion, d *DependenciasComposicionPerfiles) { d.Lote = &LoteADMIN{} }},
	}
	for _, caso := range casos {
		t.Run(caso.clase, func(t *testing.T) {
			cfg, deps := valida, dependenciasRuntimeComposicionPrueba()
			deps.Activos = activosMapaPrueba()
			if caso.cambiar != nil {
				caso.cambiar(&cfg, &deps)
			}
			causa := errors.New("causa de prueba")
			servidor, err := componerServidorPerfiles(caso.ctx, cfg, deps, constructoresPruebaClase(caso.clase, causa))
			if servidor != nil {
				t.Fatal("servidor compuesto con un paso fallido")
			}
			comprobarClaseFallo(t, err, caso.clase)
		})
	}
}

// La clase es lo único que sale: ni ClaseFallo ni el texto del error llevan la
// causa, que sigue disponible sólo para errors.Is/As dentro del proceso.
func TestClaseFalloNoExponeCausa(t *testing.T) {
	secreto := errors.New("/srv/privado/clave.pem postgres://usuario:secreto@cidonia/vec")
	err := falloConfiguracion{clase: "registro_sesiones", causa: secreto}
	if ClaseFallo(err) != "registro_sesiones" || strings.Contains(err.Error(), "secreto") ||
		strings.Contains(err.Error(), "/srv/privado") || !errors.Is(err, secreto) || !errors.Is(err, ErrConfiguracion) {
		t.Fatalf("la clase expone la causa o la pierde: %q", err.Error())
	}
	for _, ajeno := range []error{secreto, nil, ErrConfiguracion, errors.New("administracion: configuracion no valida: host"),
		falloConfiguracion{clase: secreto.Error()}} {
		if clase := ClaseFallo(ajeno); clase != "" {
			t.Fatalf("clase fuera de la lista cerrada: %q", clase)
		}
	}
	for _, clase := range []string{"dependencias", "dependencias_runtime", "registro_sesiones", "revalidador", "cuentas_is16",
		"contexto_ca36", "resolver_sesion", "contexto_conexion", "montaje_lecturas", "entorno", "retirada", "rutas_tls",
		"host", "superficie", "red", "tls", "ca"} {
		if ClaseFallo(falloConfiguracion{clase: clase}) != clase {
			t.Fatalf("clase %q fuera de la lista cerrada", clase)
		}
	}
}
