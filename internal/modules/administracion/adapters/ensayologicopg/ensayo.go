package ensayologicopg

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"vec-diputacion-granada/internal/modules/administracion/adapters/ensayofisicopg"
	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ensayofisicopg"
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
	raiz, err := ensayofisicopg.CrearRaizTemporal(e.Configuracion.RaizTemporal, "vec-cs06l-")
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
	if copiarArchivo(ctx, s.Dump, dump, e.Configuracion.LimiteArchivoBytes) != nil ||
		copiarArchivo(ctx, s.Globals, globals, e.Configuracion.LimiteArchivoBytes) != nil ||
		validarDump(dump) != nil || validarGlobals(ctx, globals, e.Configuracion.LimiteArchivoBytes) != nil {
		fallo(&resultado, "entrada", "archivos", "huellas_y_formatos_admitidos", "no_admitidos")
		return
	}
	if !e.comprobarVersiones(ctx, nombre, entrada, &resultado) {
		return
	}
	var componentes []puertos.Componente
	var montajes []string
	if e.Observador != nil {
		componentes, montajes, err = ensayofisicopg.PrepararArchivados(ctx, raiz, e.ComponentesArchivados, e.Configuracion.LimiteArchivoBytes)
		if err != nil {
			fallo(&resultado, "entrada", "archivados", "verificados", "no_comprobable")
			return
		}
	}
	cmd := append(e.opcionesAisladas(nombre), "-d", "-v", filepath.Join(raiz, "pgdata")+":/data:rw",
		"-v", entrada+":/entrada:ro", "-e", "PGDATA=/data", "-e", "POSTGRES_USER="+e.Configuracion.UsuarioBootstrap,
		"-e", "POSTGRES_DB=postgres", "-e", "POSTGRES_HOST_AUTH_METHOD=trust",
		"sha256:"+e.Configuracion.ImagenSHA256, "postgres", "-c", "listen_addresses=", "-c", "unix_socket_directories=/var/run/postgresql")
	// Los montajes archivados van antes de la imagen y del comando cerrado.
	if len(montajes) > 0 {
		for i, a := range cmd {
			if a == "sha256:"+e.Configuracion.ImagenSHA256 {
				cmd = append(cmd[:i], append(montajes, cmd[i:]...)...)
				break
			}
		}
	}
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
	if e.Observador != nil {
		if _, err := docker(ctx, nil, 4096, append(e.psql(nombre), "-c", "ALTER SYSTEM SET default_transaction_read_only='on'")...); err != nil {
			fallo(&resultado, "observacion", "solo_lectura", "comprobado", "no_comprobable")
			return
		}
		if _, err := docker(ctx, nil, 4096, append(e.psql(nombre), "-c", "SELECT pg_reload_conf()")...); err != nil {
			fallo(&resultado, "observacion", "solo_lectura", "comprobado", "no_comprobable")
			return
		}
		runtime := ensayofisicopg.RuntimeObservacion{Nombre: nombre, Raiz: raiz, ImagenSHA256: e.Configuracion.ImagenSHA256, UsuarioBootstrap: e.Configuracion.UsuarioBootstrap, Componentes: componentes, ConfiguracionOrigenExcluida: true}
		entorno, err := runtime.Entorno(ctx)
		if err != nil {
			fallo(&resultado, "observacion", "aislamiento", "comprobado", "no_comprobable")
			return
		}
		o, err := e.Observador.Observar(ctx, entorno)
		if err != nil || !ensayofisicopg.ObservacionValida(o) {
			fallo(&resultado, "observacion", "contraste_arranque", "comprobado", "no_comprobable")
			return
		}
		resultado.Observacion = o
	}
	resultado.Estado, resultado.Etapa = "restauracion_logica_completada", "completado"
	return
}

func (e Ensayador) comprobarVersiones(ctx context.Context, nombre, entrada string, r *Resultado) bool {
	image := "sha256:" + e.Configuracion.ImagenSHA256
	b, err := docker(ctx, nil, 4096, "image", "inspect", "--format", "{{.Id}}", image)
	if err != nil || strings.TrimSpace(string(b)) != image {
		return fallo(r, "versiones", "runtime_sha256", e.Configuracion.ImagenSHA256, "no_disponible")
	}
	for _, herramienta := range []string{"postgres", "psql", "pg_restore"} {
		b, err = e.herramienta(ctx, nombre+"-"+strings.ReplaceAll(herramienta, "pg_", ""), herramienta, "--version")
		observado := version(b, versionHerramienta)
		if err != nil || observado != e.Configuracion.VersionPostgreSQL {
			return fallo(r, "versiones", herramienta+"_version", e.Configuracion.VersionPostgreSQL, observado)
		}
	}
	cmd := append(e.opcionesAisladas(nombre+"-toc"), "-v", entrada+":/entrada:ro", "--entrypoint", "pg_restore", image, "--list", "/entrada/copia.dump")
	b, err = docker(ctx, nil, 8<<20, cmd...)
	if err != nil {
		return fallo(r, "versiones", "dump_version", e.Configuracion.VersionPostgreSQL, "no_comprobable")
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

const MotivoVersionNoComprobable = "no_comprobable"

func numeroVersion(s string) string {
	partes := strings.Split(s, ".")
	if len(partes) != 2 {
		return MotivoVersionNoComprobable
	}
	mayor, err := strconv.ParseUint(partes[0], 10, 32)
	if err != nil {
		return MotivoVersionNoComprobable
	}
	menor, err := strconv.ParseUint(partes[1], 10, 32)
	if err != nil {
		return MotivoVersionNoComprobable
	}
	// Ambos operandos están acotados a 32 bits; su combinación cabe en uint64.
	return strconv.FormatUint(mayor*10000+menor, 10)
}
