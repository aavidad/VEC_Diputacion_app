// Publicación operativa de datos ficticios autorizada por dirección el 12/10.
// No sustituye al gobierno RRHH ni emite decisiones V3 o firmas.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const claveOperacion = "51825f1d-6a30-4ed2-9bbf-03058e128862"
const aprobacionOperacion = "operacion:datos-ficticios:plantillas-v3:20261012"
const huellaV3 = "ac1c917bfade9610b5141e7a1f15e01e7911bcb520e2a586fccce6564d73ecef"
const huellaBase = "5559604b7ac9aa1810f52859f38527ec42d3ded8b91b91c8f58b75772b06e263"

type cabeza struct {
	Secuencia                                                              int64
	Version, Revision                                                      int
	Estado, Origen, Huella, JSONHuella, Recibo, Auditoria, Actor, Decision string
	Catalogo                                                               json.RawMessage
	Clave, Solicitud, Consumo                                              string
}
type salida struct {
	Resultado  string `json:"resultado"`
	Version    int    `json:"version"`
	Revision   int    `json:"revision"`
	Huella     string `json:"catalogo_huella_sha256"`
	Recibo     string `json:"recibo_ref,omitempty"`
	Auditoria  string `json:"auditoria_ref,omitempty"`
	Canal      string `json:"canal"`
	DuracionMS int64  `json:"duracion_ms"`
}

func main() {
	ruta, comprobar, err := argumentos(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	inicio := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	r, err := ejecutar(ctx, ruta, comprobar)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	r.DuracionMS = time.Since(inicio).Milliseconds()
	if err = json.NewEncoder(os.Stdout).Encode(r); err != nil {
		fmt.Fprintln(os.Stderr, "Salida no entregada; repita la misma operación para recuperar el recibo")
		os.Exit(1)
	}
}

func argumentos(args []string) (string, bool, error) {
	if len(args) == 3 && args[0] == "--catalogo" && args[1] != "" && args[2] == "--comprobar" {
		return args[1], true, nil
	}
	if len(args) == 4 && args[0] == "--catalogo" && args[1] != "" && args[2] == "--aplicar" && args[3] == "PUBLICA-PLANTILLAS-V3" {
		return args[1], false, nil
	}
	return "", false, errors.New("Uso: --catalogo ruta --comprobar | --aplicar PUBLICA-PLANTILLAS-V3 (sin argumentos adicionales)")
}

func cargar(ctx context.Context, ruta string) (vecdomain.CatalogoConfigurable, string, []byte, error) {
	fuente, err := fichero.NuevaConsultaCatalogoProvisionable(ruta)
	if err != nil {
		return vecdomain.CatalogoConfigurable{}, "", nil, fmt.Errorf("catálogo: %w", err)
	}
	versiones, err := fuente.ListarVersionesCatalogo(ctx, informejuridico.CatalogoPlantillasBorradorID)
	if err != nil || len(versiones) != 1 {
		return vecdomain.CatalogoConfigurable{}, "", nil, errors.New("catálogo: se requiere únicamente v3 de #1065")
	}
	c, err := versiones[0].ClonarCanonico()
	if err != nil || c.Version != 3 || c.Revision != 1 || c.Estado != vecdomain.EstadoCatalogoPublicado || informejuridico.ValidarCatalogoPlantillasBorrador(c) != nil || len(c.Entradas) != 11 {
		return c, "", nil, errors.New("catálogo: versión/revisión/entradas incompatibles; esperado 3/1/11")
	}
	b, err := json.Marshal(c)
	if err != nil {
		return c, "", nil, err
	}
	for _, marca := range []string{"borrador de desarrollo", "datos sintéticos", "modelo de ejemplo"} {
		if strings.Contains(strings.ToLower(string(b)), marca) {
			return c, "", nil, fmt.Errorf("catálogo: conserva la marca %q", marca)
		}
	}
	h, err := c.HuellaSHA256()
	if err == nil && h != huellaV3 {
		return c, h, nil, fmt.Errorf("catálogo.huella: esperado %s (#1065); actual %s", huellaV3, h)
	}
	return c, h, b, err
}

func ejecutar(ctx context.Context, ruta string, comprobar bool) (salida, error) {
	_, h, b, err := cargar(ctx, ruta)
	if err != nil {
		return salida{}, err
	}
	// Se acepta PGHOST/PGPORT/PGUSER/PGDATABASE o un DSN privado, nunca argumentos.
	cfg, err := pgx.ParseConfig(os.Getenv("VEC_PLANTILLAS_V3_DATABASE_URL"))
	if err != nil {
		return salida{}, errors.New("conexión: configuración inválida")
	}
	cfg.ConnectTimeout = 5 * time.Second
	cfg.RuntimeParams["application_name"] = "vec-operacion-plantillas-v3"
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return salida{}, errorBD("conexión", err)
	}
	defer conn.Close(context.Background())
	modo := pgx.ReadWrite
	if comprobar {
		modo = pgx.ReadOnly
	}
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: modo})
	if err != nil {
		return salida{}, errorBD("transacción", err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL search_path=pg_catalog,pg_temp; SET LOCAL timezone='UTC'; SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='15s'`); err != nil {
		return salida{}, errorBD("límites", err)
	}
	var autorizado bool
	if err = tx.QueryRow(ctx, `SELECT rolsuper OR pg_catalog.pg_has_role(current_user,'vec_contratacion_temporal_propietario','USAGE') FROM pg_catalog.pg_roles WHERE rolname=current_user`).Scan(&autorizado); err != nil || !autorizado {
		return salida{}, errors.New("credencial: se requiere instalador con acceso al propietario CT; no use el LOGIN de la app")
	}
	if !comprobar {
		if _, err = tx.Exec(ctx, `SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contratacion_temporal:catalogo_plantillas_documentos',0)); LOCK TABLE vec_contratacion_temporal.catalogo_plantillas_historia_v1 IN SHARE ROW EXCLUSIVE MODE`); err != nil {
			return salida{}, errorBD("bloqueo", err)
		}
	}
	base, err := leerCabeza(ctx, tx, "ORDER BY secuencia LIMIT 1")
	if err != nil {
		return salida{}, err
	}
	if base.Version != 1 || base.Revision != 1 || base.Estado != "publicado" || base.Origen != "bootstrap" || base.Huella != huellaBase {
		return salida{}, fmt.Errorf("preimagen.version/huella: esperado 1/%s; actual %d/%s", huellaBase, base.Version, base.Huella)
	}
	var baseAuditoria bool
	if err = tx.QueryRow(ctx, sqlBaseAuditoria, base.Secuencia).Scan(&baseAuditoria); err != nil || !baseAuditoria {
		return salida{}, errors.New("preimagen.auditoría: esperado ligada e íntegra; actual ausente o divergente")
	}
	actual, err := leerCabeza(ctx, tx, "ORDER BY secuencia DESC LIMIT 1")
	if err != nil {
		return salida{}, err
	}
	r := salida{Version: 3, Revision: 1, Huella: h, Canal: "operacion_sql_datos_ficticios"}
	if actual.Version == 3 {
		if actual.Huella != h || actual.Clave != claveOperacion || actual.Decision != aprobacionOperacion {
			return r, fmt.Errorf("publicación.v3: esperado %s/%s; actual %s/%s", h, claveOperacion, actual.Huella, actual.Clave)
		}
		var integra bool
		if err = tx.QueryRow(ctx, sqlReplayIntegro, actual.Secuencia).Scan(&integra); err != nil || !integra {
			return r, errors.New("publicación.recibo: esperado recibo/evento/auditoría íntegros; actual divergente")
		}
		r.Resultado = "replay"
		r.Recibo = actual.Recibo
		r.Auditoria = actual.Auditoria
	} else {
		if actual.Secuencia != base.Secuencia {
			return r, fmt.Errorf("cabeza: esperado bootstrap secuencia %d; actual %d (v%d/%s); revise antes de publicar", base.Secuencia, actual.Secuencia, actual.Version, actual.Estado)
		}
		r.Resultado = "preparado"
		if !comprobar {
			// No se cambia la historia. Los tres INSERT constituyen una única operación.
			if err = tx.QueryRow(ctx, sqlPublicar, string(b), h, claveOperacion, aprobacionOperacion).Scan(&r.Recibo, &r.Auditoria); err != nil {
				return r, errorBD("publicación", err)
			}
			r.Resultado = "registrado"
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return r, errorBD("confirmación; repita la misma operación", err)
	}
	return r, nil
}

func leerCabeza(ctx context.Context, tx pgx.Tx, orden string) (cabeza, error) {
	// orden procede exclusivamente de los dos literales internos anteriores.
	var h cabeza
	err := tx.QueryRow(ctx, sqlCabeza+orden).Scan(&h.Secuencia, &h.Version, &h.Revision, &h.Estado, &h.Origen, &h.Huella, &h.JSONHuella, &h.Recibo, &h.Auditoria, &h.Actor, &h.Decision, &h.Catalogo, &h.Clave, &h.Solicitud, &h.Consumo)
	if err != nil {
		return h, errorBD("historia", err)
	}
	var c vecdomain.CatalogoConfigurable
	if json.Unmarshal(h.Catalogo, &c) != nil {
		return h, errors.New("historia: catálogo inválido")
	}
	canon, err := c.HuellaSHA256()
	if err != nil || canon != h.Huella {
		return h, fmt.Errorf("historia.huella: esperado %s; actual %s", h.Huella, canon)
	}
	return h, nil
}

func errorBD(paso string, err error) error {
	// PostgreSQL puede incluir el contenido o la conexión en el detalle: no se imprime.
	var e *pgconn.PgError
	if errors.As(err, &e) {
		return fmt.Errorf("%s: SQLSTATE %s; consulte el registro privado de PostgreSQL", paso, e.Code)
	}
	return fmt.Errorf("%s: conexión o plazo no disponible (%T)", paso, err)
}
