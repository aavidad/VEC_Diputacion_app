// vec-provision-usuarios-externo publica únicamente permisos AUT17 aprobados.
// La provisión CTX15 se hace antes con la herramienta de ContextoActor.
package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/app/bootstrap"
)

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Getenv)) }

func ejecutar(args []string, salida io.Writer, entorno func(string) string) (codigo int) {
	f := flag.NewFlagSet("vec-provision-usuarios-externo", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	ruta := f.String("plan", "", "")
	fase := f.String("fase", "autorizacion", "")
	dsnArchivo := f.String("dsn-archivo", "", "")
	huella := f.String("huella-aprobada", "", "")
	aprobacion := f.String("aprobacion-ref", "", "")
	publicar := f.Bool("publicar", false, "")
	if f.Parse(args) != nil || f.NArg() != 0 || (*fase != "autorizacion" && *fase != "contexto" && *fase != "motivos") {
		return fallo(salida, "entrada_invalida", 2)
	}
	b, err := bootstrap.LeerMaterialProvisionExterna(*ruta, 262144)
	if err != nil {
		return fallo(salida, "entrada_invalida", 2)
	}
	defer clear(b)
	if *fase == "contexto" {
		return ejecutarContexto(b, *dsnArchivo, *publicar, *huella, *aprobacion, salida)
	}
	if *fase == "motivos" {
		return ejecutarMotivos(b, *dsnArchivo, *publicar, *huella, *aprobacion, salida)
	}
	s, err := bootstrap.LeerSolicitudProvisionUsuariosExterno(bytes.NewReader(b))
	if err != nil {
		return fallo(salida, "entrada_invalida", 2)
	}
	h, err := s.HuellaSHA256()
	if err != nil {
		return fallo(salida, "entrada_invalida", 2)
	}
	if !*publicar {
		if *huella != "" || *aprobacion != "" {
			return fallo(salida, "entrada_invalida", 2)
		}
		if json.NewEncoder(salida).Encode(map[string]string{"estado": "preparado", "plan_huella_sha256": h}) != nil {
			return 1
		}
		return 0
	}
	if *huella != h || *aprobacion != s.AprobacionRef {
		return fallo(salida, "aprobacion_divergente", 2)
	}
	cfg, err := pgx.ParseConfig(entorno("VEC_USUARIOS_EXTERNO_PROVISION_DATABASE_URL"))
	if err != nil || entorno("VEC_USUARIOS_EXTERNO_PROVISION_DATABASE_URL") == "" || !tlsVerificado(cfg) {
		return fallo(salida, "conexion_no_disponible", 2)
	}
	configurarConexion(cfg, "vec-provision-usuarios-externo")
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	con, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return fallo(salida, "conexion_no_disponible", 1)
	}
	defer func() {
		if cerrarConexion(con) != nil {
			codigo = 1
		}
	}()
	r, err := bootstrap.PublicarPerfilUsuariosExterno(ctx, con, s, *huella)
	if err != nil {
		return fallo(salida, "publicacion_no_confirmada", 1)
	}
	if json.NewEncoder(salida).Encode(r) != nil {
		return 1
	}
	return 0
}

func tlsVerificado(c *pgx.ConnConfig) bool {
	if c == nil || c.TLSConfig == nil || c.TLSConfig.InsecureSkipVerify || c.TLSConfig.ServerName == "" {
		return false
	}
	c.TLSConfig.MinVersion = tls.VersionTLS12
	for _, f := range c.Fallbacks {
		if f.TLSConfig == nil || f.TLSConfig.InsecureSkipVerify || f.TLSConfig.ServerName == "" {
			return false
		}
		f.TLSConfig.MinVersion = tls.VersionTLS12
	}
	return true
}

func ejecutarMotivos(b []byte, dsnArchivo string, publicar bool, huella, aprobacion string, salida io.Writer) (codigo int) {
	s, err := bootstrap.LeerSolicitudMotivosUsuariosExterno(b)
	if err != nil {
		return fallo(salida, "entrada_invalida", 2)
	}
	r, err := s.Resumen()
	if err != nil {
		return fallo(salida, "entrada_invalida", 2)
	}
	if !publicar {
		if huella != "" || aprobacion != "" {
			return fallo(salida, "entrada_invalida", 2)
		}
		if json.NewEncoder(salida).Encode(r) != nil {
			return 1
		}
		return 0
	}
	if huella != r.HuellaPlan || aprobacion != s.AprobacionRef {
		return fallo(salida, "aprobacion_divergente", 2)
	}
	secreto, err := bootstrap.LeerMaterialProvisionExterna(dsnArchivo, 16384)
	if err != nil {
		return fallo(salida, "conexion_no_disponible", 2)
	}
	defer clear(secreto)
	cfg, err := pgx.ParseConfig(strings.TrimSpace(string(secreto)))
	if err != nil || len(bytes.TrimSpace(secreto)) == 0 || !tlsVerificado(cfg) {
		return fallo(salida, "conexion_no_disponible", 2)
	}
	configurarConexion(cfg, "vec-provision-usuarios-externo-motivos")
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	con, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return fallo(salida, "conexion_no_disponible", 1)
	}
	defer func() {
		if cerrarConexion(con) != nil {
			codigo = 1
		}
	}()
	r, err = bootstrap.PublicarMotivosUsuariosExterno(ctx, con, s, huella, aprobacion)
	if err != nil {
		return fallo(salida, "publicacion_no_confirmada", 1)
	}
	if json.NewEncoder(salida).Encode(r) != nil {
		return 1
	}
	return 0
}

func fallo(w io.Writer, codigo string, salida int) int {
	if json.NewEncoder(w).Encode(map[string]string{"error": codigo}) != nil {
		return 1
	}
	return salida
}

func ejecutarContexto(b []byte, dsnArchivo string, publicar bool, huella, aprobacion string, salida io.Writer) (codigo int) {
	s, err := bootstrap.LeerSolicitudContextoUsuariosExterno(b)
	if err != nil {
		return fallo(salida, "entrada_invalida", 2)
	}
	if (!publicar && (huella != "" || aprobacion != "")) || (publicar && (huella == "" || aprobacion != s.AprobacionRef)) {
		return fallo(salida, "aprobacion_divergente", 2)
	}
	secreto, err := bootstrap.LeerMaterialProvisionExterna(dsnArchivo, 16384)
	if err != nil {
		return fallo(salida, "conexion_no_disponible", 2)
	}
	defer clear(secreto)
	cfg, err := pgx.ParseConfig(strings.TrimSpace(string(secreto)))
	if err != nil || len(bytes.TrimSpace(secreto)) == 0 || !tlsVerificado(cfg) {
		return fallo(salida, "conexion_no_disponible", 2)
	}
	configurarConexion(cfg, "vec-provision-usuarios-externo-contexto")
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	con, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return fallo(salida, "conexion_no_disponible", 1)
	}
	defer func() {
		if cerrarConexion(con) != nil {
			codigo = 1
		}
	}()
	r, err := bootstrap.OperarContextoUsuariosExterno(ctx, con, s, huella, aprobacion)
	if err != nil {
		return fallo(salida, "publicacion_no_confirmada", 1)
	}
	if json.NewEncoder(salida).Encode(r) != nil {
		return 1
	}
	return 0
}

func configurarConexion(c *pgx.ConnConfig, nombre string) {
	c.ConnectTimeout = 10 * time.Second
	if c.RuntimeParams == nil {
		c.RuntimeParams = map[string]string{}
	}
	for k, v := range map[string]string{"application_name": nombre, "timezone": "UTC", "search_path": "pg_catalog", "statement_timeout": "15s", "lock_timeout": "3s", "idle_in_transaction_session_timeout": "20s"} {
		c.RuntimeParams[k] = v
	}
}

func cerrarConexion(c *pgx.Conn) error {
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	return c.Close(ctx)
}
