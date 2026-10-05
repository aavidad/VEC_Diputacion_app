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
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var errAuditoriaIntentosDesarrollo = errors.New("auditoria.intentos.configuracion_no_disponible")

type configuracionAuditoriaIntentosDesarrollo struct {
	Esquema        string `json:"esquema"`
	DSNFile        string `json:"dsn_file"`
	Proceso        string `json:"proceso"`
	Canal          string `json:"canal"`
	LimiteSegundos int    `json:"limite_segundos"`
}

func leerConfiguracionAuditoriaIntentosDesarrollo(cfg config.Config) (configuracionAuditoriaIntentosDesarrollo, error) {
	return leerConfiguracionAuditoriaIntentosParaCanalDesarrollo(cfg, "auditoria-intentos.json", string(core.SuperficieAutenticacionInternaCorporativaV1))
}

func leerConfiguracionAuditoriaIntentosParaCanalDesarrollo(cfg config.Config, nombre, canal string) (configuracionAuditoriaIntentosDesarrollo, error) {
	var c configuracionAuditoriaIntentosDesarrollo
	if !filepath.IsAbs(cfg.DevelopmentMaterialDir) || dentroDeRepositorioGit(cfg.DevelopmentMaterialDir) || validarArbolMaterialDesarrollo(cfg.DevelopmentMaterialDir) != nil {
		return c, errAuditoriaIntentosDesarrollo
	}
	b, err := leerFicheroMaterialSeguro(filepath.Join(cfg.DevelopmentMaterialDir, nombre), 16<<10)
	if err != nil {
		return c, errAuditoriaIntentosDesarrollo
	}
	defer borrarBytes(b)
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if validarClavesJSONUnicas(b) != nil || d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF ||
		c.Esquema != "vec.auditoria.intentos.servidor.v1" || !filepath.IsLocal(c.DSNFile) ||
		c.Canal != canal || c.LimiteSegundos < 1 || c.LimiteSegundos > 30 {
		return configuracionAuditoriaIntentosDesarrollo{}, errAuditoriaIntentosDesarrollo
	}
	if !procesoAuditoriaIntentosConfigurado(c.Proceso) {
		return configuracionAuditoriaIntentosDesarrollo{}, errAuditoriaIntentosDesarrollo
	}
	return c, nil
}

func procesoAuditoriaIntentosConfigurado(p string) bool {
	if len(p) < 2 || len(p) > 80 || p[0] < 'a' || p[0] > 'z' {
		return false
	}
	for _, r := range p[1:] {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != '.' {
			return false
		}
	}
	return true
}

// AbrirRegistradorIntentosAuditoriaDesarrollo compone el puerto nominal de L.
// La cuenta dedicada, el proceso y el canal proceden de material privado.
// No publica permisos, no comparte cuentas de negocio y no crea otra auditoría.
func AbrirRegistradorIntentosAuditoriaDesarrollo(ctx context.Context, cfg config.Config, referencia *pgxpool.Pool, reservados []string) (vecports.RegistradorIntentosAuditoria, string, func(), error) {
	if ctx == nil || ctx.Err() != nil || referencia == nil {
		return nil, "", nil, errAuditoriaIntentosDesarrollo
	}
	c, err := leerConfiguracionAuditoriaIntentosDesarrollo(cfg)
	if err != nil {
		return nil, "", nil, err
	}
	return abrirRegistradorIntentosAuditoriaConfiguradoDesarrollo(ctx, cfg, referencia, reservados, c)
}

// AbrirRegistradorIntentosAuditoriaExternaDesarrollo usa un archivo y LOGIN
// dedicados al canal personal externo. No reutiliza la cuenta ni configuración
// del canal corporativo; la base coteja su proceso/canal en el mismo preflight.
func AbrirRegistradorIntentosAuditoriaExternaDesarrollo(ctx context.Context, cfg config.Config, referencia *pgxpool.Pool, reservados []string) (vecports.RegistradorIntentosAuditoria, string, func(), error) {
	if ctx == nil || ctx.Err() != nil || referencia == nil {
		return nil, "", nil, errAuditoriaIntentosDesarrollo
	}
	c, err := leerConfiguracionAuditoriaIntentosParaCanalDesarrollo(cfg, "auditoria-intentos-externa.json", string(core.SuperficieAutenticacionExternaPersonalV1))
	if err != nil {
		return nil, "", nil, err
	}
	return abrirRegistradorIntentosAuditoriaConfiguradoDesarrollo(ctx, cfg, referencia, reservados, c)
}

func abrirRegistradorIntentosAuditoriaConfiguradoDesarrollo(ctx context.Context, cfg config.Config, referencia *pgxpool.Pool, reservados []string, c configuracionAuditoriaIntentosDesarrollo) (vecports.RegistradorIntentosAuditoria, string, func(), error) {
	raiz, err := os.OpenRoot(cfg.DevelopmentMaterialDir)
	if err != nil {
		return nil, "", nil, errAuditoriaIntentosDesarrollo
	}
	defer raiz.Close()
	b, err := leerArchivoIncorporacionV2(raiz, c.DSNFile, 16<<10)
	if err != nil {
		return nil, "", nil, errAuditoriaIntentosDesarrollo
	}
	defer borrarBytes(b)
	pc, err := pgxpool.ParseConfig(strings.TrimSpace(string(b)))
	if err != nil || pc.ConnConfig == nil || pc.ConnConfig.User == "" || len(pc.ConnConfig.Fallbacks) != 0 || validarTLSPostgreSQLBorradores(&pc.ConnConfig.Config, true) != nil {
		return nil, "", nil, errAuditoriaIntentosDesarrollo
	}
	for _, u := range append(append([]string{}, reservados...), referencia.Config().ConnConfig.User) {
		if u == pc.ConnConfig.User {
			return nil, "", nil, errAuditoriaIntentosDesarrollo
		}
	}
	pc.MaxConns, pc.MinConns = 2, 0
	pc.ConnConfig.ConnectTimeout = 5 * time.Second
	if pc.ConnConfig.RuntimeParams == nil {
		pc.ConnConfig.RuntimeParams = map[string]string{}
	}
	params := pc.ConnConfig.RuntimeParams
	params["application_name"], params["timezone"], params["search_path"] = c.Proceso, "UTC", "pg_catalog,pg_temp"
	params["statement_timeout"], params["lock_timeout"], params["idle_in_transaction_session_timeout"] = "15s", "3s", "20s"
	pc.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		var valido bool
		err := conn.QueryRow(ctx, `SELECT session_user=current_user AND vec_autorizacion_atestada_v3.preflight_registrador_intentos_v1($1,$2)`, c.Proceso, c.Canal).Scan(&valido)
		if err != nil || !valido {
			return errAuditoriaIntentosDesarrollo
		}
		return nil
	}
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, "", nil, errAuditoriaIntentosDesarrollo
	}
	cerrar := pool.Close
	r, err := pgvec.NuevoRegistradorIntentosAuditoriaPostgreSQL(pool, c.Proceso, c.Canal, time.Duration(c.LimiteSegundos)*time.Second)
	if err != nil {
		cerrar()
		return nil, "", nil, errAuditoriaIntentosDesarrollo
	}
	topologia, err := acreditarTopologiaPostgreSQLPreferenciasUsuarios(ctx, referencia)
	if err != nil || cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, pool, topologia) != nil || r.PreflightIntentoAuditoria(ctx) != nil {
		cerrar()
		return nil, "", nil, errAuditoriaIntentosDesarrollo
	}
	return r, c.Proceso, cerrar, nil
}
