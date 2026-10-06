package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	aspirantescertificado "vec-diputacion-granada/internal/modules/aspirantes/adapters/certificado"
	aspirantespg "vec-diputacion-granada/internal/modules/aspirantes/adapters/postgres"
	aspirantesports "vec-diputacion-granada/internal/modules/aspirantes/ports"
	"vec-diputacion-granada/internal/vec/datospersonales"

	"vec-diputacion-granada/internal/vec/adapters/observabilidad/telemetria/medidorpg"
)

var errComposicionAspirantes = errors.New("bootstrap: Aspirantes no disponible")

// VEC_ASPIRANTES_ENABLED añade la ficha propia de Aspirantes al área
// personal. Exige preferencias externas activas (misma frontera), AD3-111 y
// Aspirantes 000001/000002 instalados, y su configuración privada.
const envAspirantesDesarrollo = "VEC_ASPIRANTES_ENABLED"

// Ruta del catálogo de datos personales (F1.2). Por defecto, el paquete de
// ejemplo pendiente de RRHH y del DPD.
const (
	envCatalogoDatosPersonalesDesarrollo = "VEC_ASPIRANTES_CATALOGO_DATOS_PERSONALES"
	rutaCatalogoDatosPersonalesEjemplo   = "data/demo/reglas/aspirantes_datos_personales.ejemplo.demo.json"
	nombreConfiguracionAspirantesExterna = "aspirantes-externa.json"
	maximoConfiguracionAspirantes        = 16 << 10
	maximoPoliticasCertificadoDesarrollo = 2
	aplicacionPostgreSQLAspirantes       = "vec-aspirantes-desarrollo"
)

// accionesAspirantes fija el orden de los tres proveedores V3.
var accionesAspirantes = [3]struct{ accion, segmento string }{
	{aspirantesports.AccionConsultar, "consultar"},
	{aspirantesports.AccionAlta, "alta"},
	{aspirantesports.AccionRectificar, "rectificar"},
}

func descriptoresMaterialAspirantesDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	d := make([]descriptorMaterialConsumidorV3Desarrollo, 0, len(accionesAspirantes))
	for _, a := range accionesAspirantes {
		audiencia, _ := aspirantesports.Audiencia(a.accion)
		d = append(d, descriptorMaterialConsumidorV3Desarrollo{
			Audiencia:        audiencia,
			Dominio:          "vec.aspirantes.ficha." + a.segmento + ".externa.desarrollo.capacidad-v3",
			Prefijo:          "clave:capacidad:aspirantes-ficha-" + a.segmento + "-externa:",
			ProveedorNominal: "proveedor-material-aspirantes-ficha-" + a.segmento + "-externa",
		})
	}
	return d
}

// configuracionAspirantesDesarrollo es privada y vive fuera de Git junto a la
// de preferencias externas. Las políticas de certificado extra solo sirven
// para los certificados sintéticos de desarrollo; producción usa
// exactamente certificado.PerfilesOficiales().
type configuracionAspirantesDesarrollo struct {
	Version                        int                                     `json:"version"`
	Autoridad                      string                                  `json:"autoridad"`
	DSNAspirantes                  string                                  `json:"dsn_aspirantes"`
	TiposConvocatoria              []datospersonales.TipoConvocatoria      `json:"tipos_convocatoria"`
	PoliticasCertificadoDesarrollo map[string]aspirantescertificado.Perfil `json:"politicas_certificado_desarrollo"`
}

func (configuracionAspirantesDesarrollo) String() string { return "[CONFIGURACION PRIVADA ASPIRANTES]" }
func (configuracionAspirantesDesarrollo) GoString() string {
	return "[CONFIGURACION PRIVADA ASPIRANTES]"
}

func leerConfiguracionAspirantesDesarrollo(cfg config.Config) (configuracionAspirantesDesarrollo, error) {
	vacia := configuracionAspirantesDesarrollo{}
	b, err := leerFicheroMaterialSeguro(filepath.Join(cfg.DevelopmentMaterialDir, "identidad", nombreConfiguracionAspirantesExterna), maximoConfiguracionAspirantes)
	if err != nil || validarClavesJSONUnicas(b) != nil {
		return vacia, errComposicionAspirantes
	}
	defer borrarBytes(b)
	var c configuracionAspirantesDesarrollo
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var extra any
	if d.Decode(&c) != nil || !errors.Is(d.Decode(&extra), io.EOF) || c.Version != 1 || c.Autoridad != AutoridadNoAutoritativa ||
		c.DSNAspirantes == "" || len(c.TiposConvocatoria) == 0 || len(c.PoliticasCertificadoDesarrollo) > maximoPoliticasCertificadoDesarrollo {
		return vacia, errComposicionAspirantes
	}
	return c, nil
}

// perfilesCertificadoAspirantes parte siempre de los oficiales. Las políticas
// sintéticas se añaden solo con la doble llave de desarrollo y nunca pueden
// sustituir una oficial.
func perfilesCertificadoAspirantes(cfg config.Config, c configuracionAspirantesDesarrollo) (map[string]aspirantescertificado.Perfil, error) {
	perfiles := aspirantescertificado.PerfilesOficiales()
	if len(c.PoliticasCertificadoDesarrollo) == 0 {
		return perfiles, nil
	}
	if !cfg.DevelopmentEnabledByDoubleKey() {
		return nil, errComposicionAspirantes
	}
	for oid, p := range c.PoliticasCertificadoDesarrollo {
		if _, oficial := perfiles[oid]; oficial {
			return nil, errComposicionAspirantes
		}
		perfiles[oid] = p
	}
	return perfiles, nil
}

func rutaCatalogoDatosPersonalesAspirantes() string {
	if ruta := strings.TrimSpace(os.Getenv(envCatalogoDatosPersonalesDesarrollo)); ruta != "" {
		return ruta
	}
	return rutaCatalogoDatosPersonalesEjemplo
}

// abrirPoolAspirantes exige TLS salvo por socket local o loopback, como el
// resto de VEC; el LOGIN debe ser miembro exclusivo del ejecutor externo y el
// adaptador lo vuelve a acreditar en cada transacción.
func abrirPoolAspirantes(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, errComposicionAspirantes
	}
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c.ConnConfig.User == "" || validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, true) != nil {
		return nil, errComposicionAspirantes
	}
	c.MaxConns, c.MinConns = 4, 0
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = map[string]string{}
	}
	for k, v := range map[string]string{"application_name": aplicacionPostgreSQLAspirantes, "timezone": "UTC", "search_path": "pg_catalog",
		"statement_timeout": "10s", "lock_timeout": "2s", "idle_in_transaction_session_timeout": "15s"} {
		c.ConnConfig.RuntimeParams[k] = v
	}
	// Mide consultas y esperas de conexión por petición (registro técnico).
	medidorpg.Instrumentar(c)
	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, errComposicionAspirantes
	}
	return pool, nil
}

// La sonda comprueba, con el LOGIN del portal externo, las fachadas de
// 000001 y el registro de frontera de 000002 antes de publicar las tres
// audiencias V3.
const sondaFronteraAspirantesSQL = `SELECT pg_catalog.has_function_privilege(session_user,
 'vec_aspirantes.registrar_denegacion_frontera_v1(text,text)','EXECUTE')`

func preflightSQLAspirantesDesarrollo(cfg config.Config) error {
	c, err := leerConfiguracionAspirantesDesarrollo(cfg)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := abrirPoolAspirantes(ctx, c.DSNAspirantes)
	if err != nil {
		return err
	}
	defer pool.Close()
	if _, err := aspirantespg.NuevoRegistroFichasPostgreSQL(ctx, pool); err != nil {
		return errComposicionAspirantes
	}
	var frontera bool
	if pool.QueryRow(ctx, sondaFronteraAspirantesSQL).Scan(&frontera) != nil || !frontera {
		return errComposicionAspirantes
	}
	return nil
}

// seleccionAspirantesDesarrollo decide si se publican las tres audiencias.
func seleccionAspirantesDesarrollo(cfg config.Config, preferenciasActivas bool) (bool, []descriptorMaterialConsumidorV3Desarrollo, error) {
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envAspirantesDesarrollo)
	if err != nil || !activo {
		return false, nil, err
	}
	if !preferenciasActivas {
		return false, nil, errComposicionAspirantes
	}
	if err := preflightSQLAspirantesDesarrollo(cfg); err != nil {
		return false, nil, err
	}
	return true, descriptoresMaterialAspirantesDesarrollo(), nil
}

type proveedoresMaterialAspirantes [3]*proveedorMaterialAltaContratacionTemporalDesarrollo

// Las tres audiencias forman un único corte: si una falla, se revierten todas.
func publicarMaterialAspirantesEnLote(ctx context.Context, gobierno *pgxpool.Pool, base materialAtestacionContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo, catalogo catalogoMaterialAutorizacionComunDesarrollo,
) (proveedoresMaterialAspirantes, error) {
	vacios := proveedoresMaterialAspirantes{}
	if ctx == nil || gobierno == nil {
		return vacios, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	descriptores := descriptoresMaterialAspirantesDesarrollo()
	var preparados [3]materialAtestacionContratacionTemporalDesarrollo
	defer func() {
		for i := range preparados {
			borrarBytes(preparados[i].claveHMAC)
		}
	}()
	for i, d := range descriptores {
		declarado, ok := catalogo.descriptorPara(d.Audiencia)
		if !ok || declarado != d {
			return vacios, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
		}
		derivado, err := derivarMaterialConsumidorV3Desarrollo(base, d)
		if err != nil {
			return vacios, err
		}
		preparados[i] = derivado
	}
	err := ejecutarTransaccionGobiernoCTDesarrollo(ctx, gobierno, func(tx pgx.Tx) error {
		for i := range preparados {
			if err := publicarGobiernoAtestacionCTEnTxDesarrollo(ctx, tx, &preparados[i]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return vacios, err
	}
	var resultado proveedoresMaterialAspirantes
	for i := range preparados {
		p, err := nuevoProveedorMaterialAutorizacionBaseDesarrollo(preparados[i], reloj)
		if err != nil {
			return vacios, err
		}
		resultado[i] = p
	}
	return resultado, nil
}
