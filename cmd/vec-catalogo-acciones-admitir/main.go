// Command vec-catalogo-acciones-admitir consume una aprobación que el DBA ya
// fijó en PostgreSQL. No crea el LOGIN, la aprobación ni el paquete de fuentes.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	adminpg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/domain"
)

//go:embed textos/*.json
var catalogos embed.FS

const maximoConfig = 8192
const maximoPlan = 32 << 20

type configuracion struct {
	DSN            string `json:"dsn"`
	TiempoSegundos int    `json:"tiempo_segundos"`
}

type planAdmision struct {
	Esquema          string `json:"esquema"`
	OperacionRef     string `json:"operacion_ref"`
	AprobacionRef    string `json:"aprobacion_ref"`
	AprobacionSHA256 string `json:"aprobacion_sha256"`
	PaqueteCanon     string `json:"paquete_canon"`
	PaqueteRef       string `json:"paquete_ref"`
	PaqueteVersion   string `json:"paquete_version"`
	PaqueteSHA256    string `json:"paquete_sha256"`
	CatalogoCanon    string `json:"catalogo_canon"`
	CatalogoRef      string `json:"catalogo_ref"`
	CatalogoVersion  string `json:"catalogo_version"`
	CatalogoSHA256   string `json:"catalogo_sha256"`
	EsperadoVersion  string `json:"esperado_version"`
	EsperadoSHA256   string `json:"esperado_sha256"`
	PreparadoEn      string `json:"preparado_en"`
	CaducaEn         string `json:"caduca_en"`
	Entorno          string `json:"entorno"`
}

type salida struct {
	Codigo   string                                   `json:"codigo"`
	Mensaje  string                                   `json:"mensaje"`
	Estado   string                                   `json:"estado,omitempty"`
	Replay   bool                                     `json:"replay,omitempty"`
	Recibo   json.RawMessage                          `json:"recibo,omitempty"`
	Catalogo *domain.CatalogoAccionesAdministracionV1 `json:"catalogo,omitempty"`
}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr)) }

func ejecutar(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("vec-catalogo-acciones-admitir", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var fase, rutaConfig, rutaPlan, shaPlan, ref, shaCatalogo, idioma string
	var version int
	f.StringVar(&fase, "fase", "", "")
	f.StringVar(&rutaConfig, "config", "", "")
	f.StringVar(&rutaPlan, "plan", "", "")
	f.StringVar(&shaPlan, "sha", "", "")
	f.StringVar(&ref, "ref", "", "")
	f.IntVar(&version, "version", 0, "")
	f.StringVar(&shaCatalogo, "catalogo-sha", "", "")
	f.StringVar(&idioma, "idioma", "es", "")
	parseErr := f.Parse(args)
	idiomaValido := idioma == "es" || idioma == "en"
	if !idiomaValido {
		idioma = "es"
	}
	mensajes, err := leerMensajes(idioma)
	if err != nil {
		return 2
	}
	emitir := func(w io.Writer, codigo string, extra salida, estado int) int {
		extra.Codigo, extra.Mensaje = codigo, mensajes[codigo]
		if json.NewEncoder(w).Encode(extra) != nil {
			return 2
		}
		return estado
	}
	fallo := func(codigo string) int { return emitir(stderr, codigo, salida{}, 1) }
	if parseErr != nil || !idiomaValido || f.NArg() != 0 || rutaConfig == "" ||
		!((fase == "aplicar" && rutaPlan != "" && huellaValida(shaPlan) && ref == "" && version == 0 && shaCatalogo == "") ||
			(fase == "consultar" && rutaPlan == "" && shaPlan == "" && ref != "" && version > 0 && huellaValida(shaCatalogo))) {
		return fallo("uso_invalido")
	}
	b, err := leerPrivado(rutaConfig, maximoConfig)
	if err != nil {
		return fallo("configuracion_insegura")
	}
	var cfg configuracion
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(&cfg) != nil || dec.Decode(new(any)) != io.EOF || cfg.DSN == "" ||
		len(cfg.DSN) > 4096 || cfg.TiempoSegundos < 1 || cfg.TiempoSegundos > 60 {
		clear(b)
		return fallo("configuracion_invalida")
	}
	clear(b)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TiempoSegundos)*time.Second)
	defer cancel()
	pgcfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return fallo("configuracion_invalida")
	}
	pgcfg.MaxConns = 1
	pgcfg.MinConns = 0
	pool, err := pgxpool.NewWithConfig(ctx, pgcfg)
	if err != nil {
		return fallo("conexion_no_disponible")
	}
	defer pool.Close()
	if fase == "consultar" {
		fuente, err := adminpg.NuevaFuenteCatalogoAcciones(ctx, pool)
		if err != nil {
			return fallo("catalogo_no_disponible")
		}
		catalogo, err := fuente.ObtenerCatalogoAccionesAdministracionV1(ctx, ref, version, shaCatalogo)
		if err != nil {
			return fallo("catalogo_no_disponible")
		}
		return emitir(stdout, "catalogo_consultado", salida{Catalogo: &catalogo}, 0)
	}
	plan, err := leerPrivado(rutaPlan, maximoPlan)
	if err != nil {
		return fallo("plan_inseguro")
	}
	defer clear(plan)
	aprobado, err := validarPlanAntesDeEnviar(plan, shaPlan)
	if err != nil {
		return fallo("plan_invalido")
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return fallo("conexion_no_disponible")
	}
	r, err := aplicarPlanEnTransaccion(ctx, tx, plan, shaPlan, aprobado)
	if errors.Is(err, errCommitAdmision) {
		return fallo("commit_indeterminado")
	}
	if err != nil {
		return fallo("admision_no_confirmada")
	}
	if r.Estado == "error" {
		return emitir(stderr, "admision_no_disponible", salida{Estado: r.Estado}, 1)
	}
	if r.Estado != "permitido" {
		return emitir(stderr, "admision_rechazada", salida{Estado: r.Estado}, 1)
	}
	return emitir(stdout, "catalogo_admitido", salida{Estado: r.Estado, Replay: r.Replay, Recibo: r.Recibo}, 0)
}

func leerMensajes(idioma string) (map[string]string, error) {
	b, err := catalogos.ReadFile("textos/" + idioma + ".json")
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if json.Unmarshal(b, &m) != nil || len(m) != 13 {
		return nil, errors.New("catalogo_invalido")
	}
	for _, clave := range []string{"uso_invalido", "configuracion_insegura", "configuracion_invalida",
		"conexion_no_disponible", "catalogo_no_disponible", "catalogo_consultado", "plan_inseguro",
		"plan_invalido", "admision_no_confirmada", "commit_indeterminado", "admision_rechazada",
		"admision_no_disponible", "catalogo_admitido"} {
		if m[clave] == "" {
			return nil, errors.New("catalogo_invalido")
		}
	}
	return m, nil
}

func huellaValida(s string) bool {
	if len(s) != 64 || s != strings.ToLower(s) {
		return false
	}
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == sha256.Size
}

func leerPrivado(ruta string, maximo int64) ([]byte, error) {
	if ruta == "" {
		return nil, os.ErrInvalid
	}
	// #nosec G304 -- Ruta elegida por el operador local; O_NOFOLLOW y fstat
	// exigen archivo regular 0600 del UID del proceso antes de leer sus bytes.
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 || st.Size() < 1 || st.Size() > maximo {
		return nil, os.ErrPermission
	}
	if stat, ok := st.Sys().(*syscall.Stat_t); !ok || int64(stat.Uid) != int64(os.Geteuid()) {
		return nil, os.ErrPermission
	}
	b, err := io.ReadAll(io.LimitReader(f, maximo+1))
	if err != nil || int64(len(b)) > maximo {
		return nil, os.ErrInvalid
	}
	return b, nil
}

func validarPlanAntesDeEnviar(plan []byte, aprobada string) (planAdmision, error) {
	var vacio planAdmision
	if len(plan) == 0 || len(plan) > maximoPlan {
		return vacio, os.ErrInvalid
	}
	suma := sha256.Sum256(plan)
	if hex.EncodeToString(suma[:]) != aprobada {
		return vacio, os.ErrPermission
	}
	campos, err := objetoExacto(plan, "esquema", "operacion_ref", "aprobacion_ref", "aprobacion_sha256",
		"paquete_canon", "paquete_ref", "paquete_version", "paquete_sha256", "catalogo_canon",
		"catalogo_ref", "catalogo_version", "catalogo_sha256", "esperado_version", "esperado_sha256",
		"preparado_en", "caduca_en", "entorno")
	if err != nil {
		return vacio, os.ErrInvalid
	}
	for _, valor := range campos {
		s, ok := cadenaJSON(valor)
		if !ok || s == "" {
			return vacio, os.ErrInvalid
		}
	}
	var p planAdmision
	if json.Unmarshal(plan, &p) != nil || p.Esquema != "vec.admin.catalogo-acciones.plan.v1" ||
		p.CatalogoCanon == "" || p.PaqueteCanon == "" || !huellaValida(p.CatalogoSHA256) ||
		!huellaValida(p.PaqueteSHA256) || !huellaValida(p.AprobacionSHA256) ||
		p.AprobacionRef == "" || p.PaqueteRef == "" || p.CatalogoRef == "" ||
		!strings.HasPrefix(p.OperacionRef, "caa_") || len(p.OperacionRef) < 26 {
		return vacio, os.ErrInvalid
	}
	if _, err := objetoExacto([]byte(p.CatalogoCanon), "referencia", "version", "fuente_ref",
		"fuente_version", "fuente_huella_sha256", "vigente_desde", "vigente_hasta",
		"entradas", "perfiles"); err != nil {
		return vacio, os.ErrInvalid
	}
	var c domain.CatalogoAccionesAdministracionV1
	dec := json.NewDecoder(strings.NewReader(p.CatalogoCanon))
	dec.DisallowUnknownFields()
	if dec.Decode(&c) != nil || dec.Decode(new(any)) != io.EOF || len(c.Perfiles) == 0 || c.Validar() != nil {
		return vacio, os.ErrInvalid
	}
	canon, err := json.Marshal(c)
	if err != nil || !bytes.Equal(canon, []byte(p.CatalogoCanon)) {
		return vacio, os.ErrInvalid
	}
	h, err := c.HuellaSHA256()
	if err != nil || h != p.CatalogoSHA256 || p.CatalogoRef != c.Referencia ||
		p.CatalogoVersion != strconv.Itoa(c.Version) || p.PaqueteRef != c.FuenteRef ||
		p.PaqueteVersion != strconv.Itoa(c.FuenteVersion) || p.PaqueteSHA256 != c.FuenteHuellaSHA256 {
		return vacio, os.ErrInvalid
	}
	return p, nil
}
