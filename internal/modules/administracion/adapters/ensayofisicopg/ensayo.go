package ensayofisicopg

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ensayofisicopg"
)

var versionHerramienta = regexp.MustCompile(`\(PostgreSQL\) ([0-9]+\.[0-9]+)(?:[ \r\n]|$)`)

func (e Ensayador) ensayar(ctx context.Context, s Solicitud, r Resultado) (resultado Resultado) {
	resultado = r
	raiz, err := CrearRaizTemporal(e.Configuracion.RaizTemporal, "vec-cs06f-")
	if err != nil {
		fallo(&resultado, "entrada", "scratch")
		return
	}
	nombre := filepath.Base(raiz)
	nombres := []string{nombre, nombre + "-postgres", nombre + "-psql", nombre + "-control"}
	defer func() {
		limpio := true
		for _, n := range nombres {
			if !eliminarContenedor(n) {
				limpio = false
			}
		}
		if os.RemoveAll(raiz) != nil {
			limpio = false
		}
		resultado.LimpiezaCompletada = limpio
		if !limpio {
			fallo(&resultado, "limpieza", "recursos_propios")
		}
	}()
	cuenta := &cuentaTar{}
	componentes := make([]puertos.Componente, 0, len(s.Componentes))
	archivos := make([]string, len(s.Componentes))
	destinos := make([]string, len(s.Componentes))
	for i, c := range s.Componentes {
		archivos[i] = filepath.Join(raiz, fmt.Sprintf("componente-%04d.tar", i))
		destinos[i] = filepath.Join(raiz, fmt.Sprintf("componente-%04d", i))
		// Acotar también los TAR copiados acumulados, antes de consumir scratch.
		info, err := os.Stat(c.Tar.Ruta)
		if err != nil || info.Size() > e.Configuracion.LimiteArchivoBytes-cuentaTarCopiados(archivos[:i]) {
			fallo(&resultado, "entrada", "limite_archivos")
			return
		}
		if copiarArchivo(ctx, c.Tar, archivos[i], e.Configuracion.LimiteArchivoBytes) != nil || revisarTar(ctx, archivos[i], e.Configuracion, cuenta) != nil {
			fallo(&resultado, "entrada", "tar")
			return
		}
	}
	for i, c := range s.Componentes {
		if os.Mkdir(destinos[i], 0700) != nil || extraerTar(ctx, archivos[i], destinos[i]) != nil {
			fallo(&resultado, "entrada", "extraccion")
			return
		}
		hash, bytes, err := huellaContenido(ctx, filepath.Join(destinos[i], "contenido"))
		if err != nil {
			fallo(&resultado, "entrada", "contenido")
			return
		}
		componentes = append(componentes, puertos.Componente{ID: c.ID, Tipo: c.Tipo, RutaInterna: fmt.Sprintf("/componentes/%04d/contenido", i), SHA256: c.Tar.SHA256, ContenidoSHA256: hash, ContenidoBytes: bytes})
		if prepararMaterial(ctx, destinos[i], e.Configuracion, cuenta, &componentes[len(componentes)-1]) != nil {
			fallo(&resultado, "entrada", "material")
			return
		}
	}
	pgdata := filepath.Join(destinos[0], "contenido")
	versionPath := filepath.Join(pgdata, "PG_VERSION")
	info, err := os.Stat(versionPath)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 16 {
		fallo(&resultado, "versiones", "pgdata_version")
		return
	}
	b, err := os.ReadFile(versionPath) // #nosec G304 -- PG_VERSION regular de hasta 16 bytes en PGDATA propio extraído y sin enlaces.
	if err != nil || strings.TrimSpace(string(b)) != "18" {
		fallo(&resultado, "versiones", "pgdata_version")
		return
	}
	for _, n := range []string{"postmaster.pid", "standby.signal", "recovery.signal"} {
		if _, err := os.Stat(filepath.Join(pgdata, n)); !os.IsNotExist(err) {
			fallo(&resultado, "entrada", "cluster_no_frio")
			return
		}
	}
	if entradas, err := os.ReadDir(filepath.Join(pgdata, "pg_tblspc")); err != nil || len(entradas) != 0 {
		fallo(&resultado, "entrada", "tablespaces_no_admitidos")
		return
	}
	if !e.versiones(ctx, nombre) {
		fallo(&resultado, "versiones", "runtime_version")
		return
	}
	resultado.VersionPostgreSQL = e.Configuracion.VersionPostgreSQL
	control := append(e.opcionesAisladas(nombre+"-control"), "-v", pgdata+":/data:ro", "--entrypoint", "pg_controldata", "sha256:"+e.Configuracion.ImagenSHA256, "/data")
	b, err = docker(ctx, nil, 16384, control...)
	if err != nil || !regexp.MustCompile(`(?m)^Database cluster state:\s+shut down\s*$`).Match(b) {
		fallo(&resultado, "entrada", "parada_limpia")
		return
	}
	// Se conservan bytes de configuración en los TAR protegidos. Sólo la copia
	// desechable pierde auto.conf, evitando includes/replicación/archivado del origen.
	if os.WriteFile(filepath.Join(pgdata, "postgresql.auto.conf"), nil, 0600) != nil {
		fallo(&resultado, "inicio", "configuracion_clon")
		return
	}
	config := filepath.Join(raiz, "configuracion")
	if os.Mkdir(config, 0700) != nil || os.WriteFile(filepath.Join(config, "pg_ident.conf"), nil, 0600) != nil || os.WriteFile(filepath.Join(config, "postgresql.conf"), []byte("data_directory='/data'\nlisten_addresses=''\nunix_socket_directories='/var/run/postgresql'\narchive_mode=off\nshared_preload_libraries=''\nssl=off\nlogging_collector=off\ndefault_transaction_read_only=on\n"), 0600) != nil || os.WriteFile(filepath.Join(config, "pg_hba.conf"), []byte("local all all trust\n"), 0600) != nil {
		fallo(&resultado, "inicio", "configuracion_clon")
		return
	}
	if prepararAuxiliar(raiz) != nil {
		fallo(&resultado, "inicio", "auxiliar_propio")
		return
	}
	cmd := append(e.opcionesAisladas(nombre), "-d", "-v", filepath.Join(raiz, "verificador")+":/verificador:ro", "-v", filepath.Join(raiz, "control")+":/control:rw", "-v", pgdata+":/data:rw", "-v", config+":/configuracion:ro")
	cmd = append(cmd, montajesNSS(raiz)...)
	for i := 1; i < len(destinos); i++ {
		cmd = append(cmd, "-v", destinos[i]+fmt.Sprintf(":/componentes/%04d:ro", i))
	}
	cmd = append(cmd, "--entrypoint", "postgres", "sha256:"+e.Configuracion.ImagenSHA256, "-D", "/data", "-c", "config_file=/configuracion/postgresql.conf", "-c", "hba_file=/configuracion/pg_hba.conf", "-c", "ident_file=/configuracion/pg_ident.conf")
	if _, err = docker(ctx, nil, 4096, cmd...); err != nil || !e.esperar(ctx, nombre) {
		fallo(&resultado, "inicio", "postgresql_aislado")
		return
	}
	runtime := RuntimeObservacion{Nombre: nombre, Raiz: raiz, ImagenSHA256: e.Configuracion.ImagenSHA256, UsuarioBootstrap: e.Configuracion.UsuarioBootstrap, Componentes: componentes, ConfiguracionOrigenExcluida: true}
	if err := runtime.anclarFisico(ctx, s.Componentes[0].Tar.SHA256); err != nil {
		fallo(&resultado, "aislamiento", "procedencia_fisica")
		return
	}
	entorno, err := runtime.Entorno(ctx)
	if err != nil {
		fallo(&resultado, "aislamiento", "runtime")
		return
	}
	b, err = runtime.EjecutarPostgreSQL(ctx, "psql", []string{"-X", "-At", "-U", e.Configuracion.UsuarioBootstrap, "-d", "postgres", "-c", "SELECT current_setting('server_version_num'),current_setting('listen_addresses'),current_setting('archive_mode'),current_setting('default_transaction_read_only')"}, nil, 4096)
	partes := strings.Split(e.Configuracion.VersionPostgreSQL, ".")
	minor, _ := strconv.Atoi(partes[1])
	if err != nil || strings.TrimSpace(string(b)) != fmt.Sprintf("%d||off|on", 180000+minor) {
		fallo(&resultado, "versiones", "servidor_sin_tcp_archivado")
		return
	}
	if e.Observador != nil {
		o, err := e.Observador.Observar(ctx, entorno)
		if err != nil {
			fallo(&resultado, "observacion", "contraste_arranque")
			return
		}
		if !ObservacionValida(o) {
			fallo(&resultado, "observacion", "evidencia")
			return
		}
		resultado.Observacion = o
	}
	resultado.Estado, resultado.Etapa = "restauracion_fisica_completada", "completado"
	return
}
func cuentaTarCopiados(rutas []string) int64 {
	var n int64
	for _, p := range rutas {
		if i, e := os.Stat(p); e == nil {
			n += i.Size()
		}
	}
	return n
}

// ObservacionValida valida la forma de la evidencia; no otorga validez al conjunto.
func ObservacionValida(o puertos.Observacion) bool {
	return (o.ContrasteEstado == "igual" || o.ContrasteEstado == "diferente" || o.ContrasteEstado == "no_comprobable") && (o.ArranqueEstado == "comprobado" || o.ArranqueEstado == "fallido" || o.ArranqueEstado == "no_comprobable") && (o.EvidenciaSHA256 == "" || huellaValida.MatchString(o.EvidenciaSHA256)) && (o.ArranqueEstado != "comprobado" || (o.SaludComprobada && o.ConsultaAutorizadaComprobada && o.DespachosBloqueados && huellaValida.MatchString(o.EvidenciaSHA256)))
}
func (e Ensayador) versiones(ctx context.Context, nombre string) bool {
	image := "sha256:" + e.Configuracion.ImagenSHA256
	b, err := docker(ctx, nil, 4096, "image", "inspect", "--format", "{{.Id}}", image)
	if err != nil || strings.TrimSpace(string(b)) != image {
		slog.Error("cs06_imagen_runtime_no_verificada")
		return false
	}
	for _, h := range []string{"postgres", "psql"} {
		b, err = e.herramienta(ctx, nombre+"-"+h, h, "--version")
		m := versionHerramienta.FindSubmatch(b)
		if err != nil || len(m) != 2 || string(m[1]) != e.Configuracion.VersionPostgreSQL {
			slog.Error("cs06_version_herramienta_no_verificada")
			return false
		}
	}
	return true
}
func (e Ensayador) esperar(ctx context.Context, n string) bool {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := docker(ctx, nil, 4096, "exec", n, "pg_isready", "-h", "/var/run/postgresql", "-U", e.Configuracion.UsuarioBootstrap, "-d", "postgres"); err == nil {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
		}
	}
}
