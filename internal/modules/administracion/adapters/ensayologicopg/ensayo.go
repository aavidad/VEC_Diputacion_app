package ensayologicopg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var versionHerramienta = regexp.MustCompile(`\(PostgreSQL\) ([0-9]+\.[0-9]+)(?:[ \r\n]|$)`)
var versionOrigen = regexp.MustCompile(`(?m)^;[ \t]+Dumped from database version: ([0-9]+\.[0-9]+)(?:[ \r\n]|$)`)
var versionDump = regexp.MustCompile(`(?m)^;[ \t]+Dumped by pg_dump version: ([0-9]+\.[0-9]+)(?:[ \r\n]|$)`)

func version(texto []byte, patron *regexp.Regexp) string {
	v := patron.FindSubmatch(texto)
	if len(v) != 2 {
		return "no_comprobable"
	}
	return string(v[1])
}

func (e Ensayador) ensayar(ctx context.Context, s Solicitud, r Resultado) (resultado Resultado) {
	resultado = r
	// Raíz y nombres se generan aquí. El operador no puede elegir un volumen,
	// servicio o contenedor existente. Todos los archivos permanecen privados.
	raiz, err := os.MkdirTemp("/dev/shm", "vec-cs06l-")
	if err != nil {
		fallo(&resultado, "entrada", "scratch", "disponible", "no_disponible")
		return
	}
	nombre := filepath.Base(raiz)
	nombres := []string{nombre, nombre + "-postgres", nombre + "-psql", nombre + "-restore", nombre + "-toc"}
	defer func() {
		limpio := true
		for _, n := range nombres {
			if !eliminarContenedor(n) {
				limpio = false
			}
		}
		if os.RemoveAll(raiz) != nil { // #nosec G703 -- raíz generada por MkdirTemp, nunca aportada por entrada o configuración.
			limpio = false
		}
		resultado.LimpiezaCompletada = limpio
		if !limpio {
			fallo(&resultado, "limpieza", "recursos_propios", "retirados", "pendiente_limpieza")
		}
	}()
	entrada := filepath.Join(raiz, "entrada")
	if os.Mkdir(entrada, 0700) != nil || os.Mkdir(filepath.Join(raiz, "pgdata"), 0700) != nil {
		fallo(&resultado, "entrada", "scratch", "disponible", "no_disponible")
		return
	}
	dump, globals := filepath.Join(entrada, "copia.dump"), filepath.Join(entrada, "globals.sql")
	if copiarArchivo(s.Dump, dump, e.Configuracion.LimiteArchivoBytes) != nil ||
		copiarArchivo(s.Globals, globals, e.Configuracion.LimiteArchivoBytes) != nil ||
		!dumpAdmitido(dump) || !globalsAdmitidos(globals, e.Configuracion.LimiteArchivoBytes) {
		fallo(&resultado, "entrada", "archivos", "huellas_y_formatos_admitidos", "no_admitidos")
		return
	}
	if !e.comprobarVersiones(ctx, nombre, entrada, &resultado) {
		return
	}
	cmd := append(e.opcionesAisladas(nombre), "-d", "-v", filepath.Join(raiz, "pgdata")+":/data:rw",
		"-v", entrada+":/entrada:ro", "-e", "PGDATA=/data", "-e", "POSTGRES_USER="+e.Configuracion.UsuarioBootstrap,
		"-e", "POSTGRES_DB=postgres", "-e", "POSTGRES_HOST_AUTH_METHOD=trust",
		"sha256:"+e.Configuracion.ImagenSHA256, "postgres", "-c", "listen_addresses=", "-c", "unix_socket_directories=/var/run/postgresql")
	if _, err = docker(ctx, nil, 4096, cmd...); err != nil || !e.esperar(ctx, nombre) {
		fallo(&resultado, "inicio", "postgresql_aislado", "disponible", "no_disponible")
		return
	}
	// La sonda de servidor sólo lee parámetros: antes de enviar globals/dump
	// verifica versión efectiva y ausencia de escucha TCP en el cluster nuevo.
	sonda := append(e.psql(nombre), "-At", "-c", "SELECT current_setting('server_version_num'), current_setting('listen_addresses')")
	b, err := docker(ctx, nil, 4096, sonda...)
	esperado := numeroVersion(e.Configuracion.VersionPostgreSQL) + "|"
	if err != nil || strings.TrimSpace(string(b)) != esperado {
		fallo(&resultado, "versiones", "postgresql_servidor_sin_tcp", esperado, "no_comprobable")
		return
	}
	if _, err = docker(ctx, nil, 4096, append(e.psql(nombre), "-f", "/entrada/globals.sql")...); err != nil {
		fallo(&resultado, "globals", "restauracion_globals", "completada", "fallida")
		return
	}
	cmd = []string{"exec", nombre, "pg_restore", "--host", "/var/run/postgresql", "--username", e.Configuracion.UsuarioBootstrap,
		"--dbname", "postgres", "--create", "--exit-on-error", "/entrada/copia.dump"}
	if _, err = docker(ctx, nil, 4096, cmd...); err != nil {
		fallo(&resultado, "restauracion", "restauracion_logica", "completada", "fallida")
		return
	}
	resultado.Estado, resultado.Etapa = "restauracion_logica_completada", "completado"
	return
}

func (e Ensayador) comprobarVersiones(ctx context.Context, nombre, entrada string, r *Resultado) bool {
	image := "sha256:" + e.Configuracion.ImagenSHA256
	b, err := docker(ctx, nil, 4096, "image", "inspect", "--format", "{{.Id}}", image)
	if err != nil || strings.TrimSpace(string(b)) != image {
		fallo(r, "versiones", "runtime_sha256", e.Configuracion.ImagenSHA256, "no_disponible")
		return false
	}
	for _, herramienta := range []string{"postgres", "psql", "pg_restore"} {
		b, err = e.herramienta(ctx, nombre+"-"+strings.ReplaceAll(herramienta, "pg_", ""), herramienta, "--version")
		observado := version(b, versionHerramienta)
		if err != nil || observado != e.Configuracion.VersionPostgreSQL {
			fallo(r, "versiones", herramienta+"_version", e.Configuracion.VersionPostgreSQL, observado)
			return false
		}
	}
	cmd := append(e.opcionesAisladas(nombre+"-toc"), "-v", entrada+":/entrada:ro", "--entrypoint", "pg_restore", image, "--list", "/entrada/copia.dump")
	b, err = docker(ctx, nil, 8<<20, cmd...)
	if err != nil {
		fallo(r, "versiones", "dump_version", e.Configuracion.VersionPostgreSQL, "no_comprobable")
		return false
	}
	for _, dato := range []struct {
		clave  string
		patron *regexp.Regexp
	}{{"dump_origen_version", versionOrigen}, {"dump_herramienta_version", versionDump}} {
		observado := version(b, dato.patron)
		if observado != e.Configuracion.VersionPostgreSQL {
			fallo(r, "versiones", dato.clave, e.Configuracion.VersionPostgreSQL, observado)
			return false
		}
	}
	r.VersionPostgreSQL = e.Configuracion.VersionPostgreSQL
	return true
}

func (e Ensayador) psql(nombre string) []string {
	return []string{"exec", nombre, "psql", "-X", "--host", "/var/run/postgresql", "--username", e.Configuracion.UsuarioBootstrap,
		"--dbname", "postgres", "--set", "ON_ERROR_STOP=1"}
}

func (e Ensayador) esperar(ctx context.Context, nombre string) bool {
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()
	for {
		_, err := docker(ctx, nil, 4096, "exec", nombre, "pg_isready", "--host", "/var/run/postgresql", "--username", e.Configuracion.UsuarioBootstrap, "--dbname", "postgres")
		if err == nil {
			// La imagen inicia un servidor temporal durante initdb. Sólo aceptar
			// el proceso definitivo, antes de comenzar a restaurar los archivos.
			b, err := docker(ctx, nil, 4096, "exec", nombre, "cat", "/proc/1/comm")
			if err == nil && strings.TrimSpace(string(b)) == "postgres" {
				return true
			}
		}
		select {
		case <-ctx.Done():
			return false
		case <-t.C:
		}
	}
}

func numeroVersion(s string) string {
	var mayor, menor int
	if _, err := fmt.Sscanf(s, "%d.%d", &mayor, &menor); err != nil {
		return "no_comprobable"
	}
	return fmt.Sprint(mayor*10000 + menor)
}
