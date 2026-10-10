package administracion

import (
	"context"
	"reflect"

	"github.com/jackc/pgx/v5/pgxpool"
	"io/fs"
	"net/http"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad/adminperfiles"
	identidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/ports"
)

// El único compositor recibe la fuente de lectura ya autorizada. No abre
// autoridad de actos ni requiere su pool. Fuentes ausentes conservan 503.
type DependenciasComposicionPerfiles struct {
	Confianza                                                                  ConfianzaPerfilesV3
	PoolCuentas, PoolRegistroSesion, PoolRevalidacionSesion, PoolContextoADMIN *pgxpool.Pool
	Seudonimizador                                                             identidad.SeudonimizadorAlta
	FuenteIdentificadoresADMIN                                                 adminperfiles.FuenteIdentificadoresADMIN
	ConfiguracionContextoADMIN                                                 adminperfiles.ConfiguracionContextoADMIN
	EspacioIdentidad, DominioHMACRef                                           string
	Lecturas                                                                   api.FuenteLecturas
	FuenteSeleccion                                                            FuenteSeleccionAuditadaADMIN
	Auditor                                                                    api.AuditorFrontera
	Reloj                                                                      ports.Reloj
	Activos                                                                    fs.FS
	SoloUsuariosMetadatos                                                      bool
	// Lote abre la preparación y el lote ordinario junto a las lecturas de
	// usuarios. Sólo se admite con SoloUsuariosMetadatos.
	Lote *LoteADMIN
	// GobiernoPlan abre el gobierno del plan nominal de firma de Contratación
	// temporal junto a las lecturas de usuarios. Sólo con SoloUsuariosMetadatos.
	GobiernoPlan api.ServicioGobiernoPlanFirmaADMIN
	// Efectos abre los efectos nominales (cargos competenciales,
	// certificados nominales) junto a las lecturas de usuarios. Sólo con
	// SoloUsuariosMetadatos.
	Efectos []EfectoNominalMontado
}

// LoteADMIN es la autoridad del lote ya compuesta: organización privada,
// catálogo de perfiles registrados y servicio de aplicación del lote.
type LoteADMIN struct {
	Organizacion string
	// Motivos admitidos para dar o quitar perfiles, de la configuración.
	Motivos  []api.MotivoLote
	Catalogo ports.CatalogoRolesAdministrables
	Servicio api.ServicioLotesADMIN
}

// constructoresPerfiles reúne los cuatro adaptadores PostgreSQL de la
// composición. ComponerServidorPerfiles pasa siempre los reales; sólo las
// pruebas pasan otros para alcanzar cada clase de fallo sin base de datos.
type constructoresPerfiles struct {
	registro    func(context.Context, *pgxpool.Pool, *pgxpool.Pool, identidad.SeudonimizadorAlta, string, string) (*identidad.RegistroSesionesPostgreSQL, error)
	revalidador func(context.Context, *pgxpool.Pool) (*identidad.RevalidadorAutenticacionActorPostgreSQL, error)
	cuentas     func(context.Context, *pgxpool.Pool, httpseguridad.Reloj, adminperfiles.FuenteIdentificadoresADMIN, adminperfiles.SeudonimizadorFuenteADMIN) (*adminperfiles.PostgreSQL, error)
	contextos   func(context.Context, *pgxpool.Pool, httpseguridad.Reloj, adminperfiles.ConfiguracionContextoADMIN) (*application.AutoridadContextoActorRegistradoV2, error)
}

func ComponerServidorPerfiles(ctx context.Context, cfg Configuracion, deps DependenciasComposicionPerfiles) (*http.Server, error) {
	return componerServidorPerfiles(ctx, cfg, deps, constructoresPerfiles{
		registro: identidad.NuevoRegistroSesionesPostgreSQL, revalidador: identidad.NuevoRevalidadorAutenticacionActorPostgreSQL,
		cuentas: adminperfiles.NuevoPostgreSQLConFuenteADMIN, contextos: adminperfiles.NuevoContextoRegistradoADMINPostgreSQL,
	})
}

// componerServidorPerfiles devuelve cada fallo con una clase de la lista
// cerrada documentada en ClaseFallo; la causa queda dentro, nunca en el texto.
func componerServidorPerfiles(ctx context.Context, cfg Configuracion, deps DependenciasComposicionPerfiles, c constructoresPerfiles) (*http.Server, error) {
	if ctx == nil || ctx.Err() != nil || dependenciaComposicionNula(deps.Reloj) ||
		deps.PoolCuentas == nil || deps.PoolRegistroSesion == nil || deps.PoolRevalidacionSesion == nil ||
		dependenciaComposicionNula(deps.Auditor) || dependenciaComposicionNula(deps.Activos) ||
		dependenciaComposicionNula(deps.Confianza.Fuente) {
		return nil, falloConfiguracion{clase: "dependencias"}
	}
	if deps.PoolContextoADMIN == nil || deps.PoolContextoADMIN == deps.PoolCuentas ||
		deps.PoolContextoADMIN == deps.PoolRegistroSesion || deps.PoolContextoADMIN == deps.PoolRevalidacionSesion ||
		deps.PoolCuentas == deps.PoolRegistroSesion || deps.PoolCuentas == deps.PoolRevalidacionSesion ||
		deps.PoolRegistroSesion == deps.PoolRevalidacionSesion ||
		dependenciaComposicionNula(deps.FuenteIdentificadoresADMIN) || dependenciaComposicionNula(deps.Seudonimizador) ||
		deps.ConfiguracionContextoADMIN.Proceso == "" {
		return nil, falloConfiguracion{clase: "dependencias_runtime"}
	}
	registro, err := c.registro(ctx, deps.PoolRegistroSesion, deps.PoolRevalidacionSesion, deps.Seudonimizador, deps.EspacioIdentidad, deps.DominioHMACRef)
	if err != nil {
		return nil, falloConfiguracion{clase: "registro_sesiones", causa: err}
	}
	revalidador, err := c.revalidador(ctx, deps.PoolRevalidacionSesion)
	if err != nil {
		return nil, falloConfiguracion{clase: "revalidador", causa: err}
	}
	cuentas, err := c.cuentas(ctx, deps.PoolCuentas, deps.Reloj, deps.FuenteIdentificadoresADMIN, deps.Seudonimizador)
	if err != nil {
		return nil, falloConfiguracion{clase: "cuentas_is16", causa: err}
	}
	contextos, err := c.contextos(ctx, deps.PoolContextoADMIN, deps.Reloj, deps.ConfiguracionContextoADMIN)
	if err != nil {
		return nil, falloConfiguracion{clase: "contexto_ca36", causa: err}
	}
	sesiones, err := NuevoResolverSesionPerfiles(cfg, adminperfiles.Dependencias{Cuentas: cuentas, Registro: registro, Revalidador: revalidador, Contextos: contextos, Autorizacion: deps.Confianza.Fuente, Reloj: deps.Reloj})
	if err != nil {
		return nil, conClase("resolver_sesion", err)
	}
	contextoConexion, err := NuevoContextoConexionPerfiles(deps.Reloj)
	if err != nil {
		return nil, falloConfiguracion{clase: "contexto_conexion", causa: err}
	}
	servidor, err := NuevoServidorConLecturas(cfg, DependenciasPerfiles{ContextoConexion: contextoConexion, Sesiones: sesiones, Lecturas: deps.Lecturas, Auditor: deps.Auditor, Reloj: deps.Reloj, Activos: deps.Activos,
		ObservadorSelector: sesiones, FuenteSeleccion: deps.FuenteSeleccion, AudienciaSelector: cfg.Audiencia, SoloUsuariosMetadatos: deps.SoloUsuariosMetadatos,
		Lote: deps.Lote, GobiernoPlan: deps.GobiernoPlan, Efectos: deps.Efectos})
	if err != nil {
		return nil, conClase("montaje_lecturas", err)
	}
	return servidor, nil
}

func dependenciaComposicionNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}
