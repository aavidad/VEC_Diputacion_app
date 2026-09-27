// vec-provision-plantillas instala una sola preimagen publicada de plantillas
// por el canal migrador, antes de arrancar la API RRHH. No forma parte del
// servidor ni acepta credenciales en argumentos.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const sqlProvision = `SELECT resultado,recibo_ref,version,revision,catalogo_huella_sha256,contenido_json_sha256,procedencia_ref,registrada_en
 FROM vec_contratacion_temporal.provisionar_catalogo_plantillas_base_v1($1::jsonb,$2,$3,$4)`

var (
	errEntrada   = errors.New("provisión CT: catálogo o aprobación inválidos")
	errConexion  = errors.New("provisión CT: conexión migradora no disponible")
	huellaValida = regexp.MustCompile(`^[0-9a-f]{64}$`)
	reciboValido = regexp.MustCompile(`^recibo:[0-9a-f-]{36}$`)
)

type reciboProvision struct {
	Resultado            string    `json:"resultado"`
	ReciboRef            string    `json:"recibo_ref"`
	Version              int64     `json:"version"`
	Revision             int64     `json:"revision"`
	CatalogoHuellaSHA256 string    `json:"catalogo_huella_sha256"`
	ContenidoJSONSHA256  string    `json:"contenido_json_sha256"`
	ProcedenciaRef       string    `json:"procedencia_ref"`
	RegistradaEn         time.Time `json:"registrada_en"`
}

func main() {
	var ruta, aprobacion string
	flag.StringVar(&ruta, "catalogo", "", "ruta absoluta del catálogo JSON publicado")
	flag.StringVar(&aprobacion, "aprobacion-ref", "", "referencia de la aprobación de la fuente")
	flag.Parse()
	if ruta == "" {
		ruta = os.Getenv("VEC_CT_PLANTILLAS_SOURCE_PATH")
	}
	if flag.NArg() != 0 || ruta == "" || aprobacion == "" || len(aprobacion) > 512 || aprobacion != strings.TrimSpace(aprobacion) {
		salir(errEntrada, 2)
	}
	dsn := os.Getenv("VEC_CT_PLANTILLAS_MIGRADOR_DATABASE_URL")
	if dsn == "" {
		salir(errConexion, 2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := cargarCatalogo(ctx, ruta, time.Now().UTC())
	if err != nil {
		salir(errEntrada, 2)
	}
	if aprobacion != c.AprobacionRef {
		salir(errEntrada, 2)
	}
	huella, err := c.HuellaSHA256()
	if err != nil {
		salir(errEntrada, 2)
	}
	b, err := json.Marshal(c)
	if err != nil {
		salir(errEntrada, 2)
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		salir(errConexion, 1)
	}
	if cfg.ConnectTimeout == 0 || cfg.ConnectTimeout > 10*time.Second {
		cfg.ConnectTimeout = 10 * time.Second
	}
	if cfg.RuntimeParams == nil {
		cfg.RuntimeParams = map[string]string{}
	}
	cfg.RuntimeParams["application_name"] = "vec-provision-plantillas"
	con, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		salir(errConexion, 1)
	}
	defer con.Close(context.Background())
	tx, err := con.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		salir(errConexion, 1)
	}
	defer tx.Rollback(context.Background())
	var recibo reciboProvision
	err = tx.QueryRow(ctx, sqlProvision, string(b), huella, c.FuenteRef, aprobacion).Scan(&recibo.Resultado, &recibo.ReciboRef, &recibo.Version, &recibo.Revision, &recibo.CatalogoHuellaSHA256, &recibo.ContenidoJSONSHA256, &recibo.ProcedenciaRef, &recibo.RegistradaEn)
	if err != nil || !reciboCompatible(recibo, c, huella) {
		salir(errConexion, 1)
	}
	if err = tx.Commit(ctx); err != nil {
		salir(errConexion, 1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(recibo)
}

func cargarCatalogo(ctx context.Context, ruta string, instante time.Time) (vecdomain.CatalogoConfigurable, error) {
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		return vecdomain.CatalogoConfigurable{}, err
	}
	versiones, err := consulta.ListarVersionesCatalogo(ctx, informejuridico.CatalogoPlantillasBorradorID)
	if err != nil {
		return vecdomain.CatalogoConfigurable{}, err
	}
	var elegido *vecdomain.CatalogoConfigurable
	for i := range versiones {
		if versiones[i].Estado == vecdomain.EstadoCatalogoPublicado && (elegido == nil || versiones[i].Version > elegido.Version) {
			elegido = &versiones[i]
		}
	}
	if elegido == nil {
		return vecdomain.CatalogoConfigurable{}, errEntrada
	}
	c, err := elegido.ClonarCanonico()
	if err != nil {
		return vecdomain.CatalogoConfigurable{}, err
	}
	if _, err = informejuridico.NuevasPlantillasBorrador(c, instante); err != nil {
		return vecdomain.CatalogoConfigurable{}, err
	}
	return c, nil
}

func reciboCompatible(r reciboProvision, c vecdomain.CatalogoConfigurable, huella string) bool {
	return (r.Resultado == "registrado" || r.Resultado == "replay") && reciboValido.MatchString(r.ReciboRef) &&
		r.Version == int64(c.Version) && r.Revision == int64(c.Revision) && r.CatalogoHuellaSHA256 == huella &&
		huellaValida.MatchString(r.ContenidoJSONSHA256) && r.ProcedenciaRef == c.FuenteRef && !r.RegistradaEn.IsZero()
}

func salir(err error, codigo int) { fmt.Fprintln(os.Stderr, err); os.Exit(codigo) }
