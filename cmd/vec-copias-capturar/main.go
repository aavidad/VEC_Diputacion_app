// vec-copias-capturar compone una captura real en un ensayo local sintético.
package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	adaptador "vec-diputacion-granada/internal/modules/administracion/adapters/capturacopias"
	"vec-diputacion-granada/internal/modules/administracion/adapters/inventariocopias"
	aplicacion "vec-diputacion-granada/internal/modules/administracion/application/capturacopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	puertos "vec-diputacion-granada/internal/modules/administracion/ports/capturacopias"
)

type configuracion struct {
	FormatoVersion           int                  `json:"formato_version"`
	EnsayoLocalSintetico     bool                 `json:"ensayo_local_sintetico"`
	OrigenRef                string               `json:"origen_ref"`
	OperacionRef             string               `json:"operacion_ref"`
	InicioVentana            time.Time            `json:"inicio_ventana"`
	FinVentana               time.Time            `json:"fin_ventana"`
	RaizInstalada            string               `json:"raiz_instalada"`
	Descriptor               string               `json:"descriptor"`
	Inventario               string               `json:"inventario"`
	DirectorioBloqueos       string               `json:"directorio_bloqueos"`
	DirectorioCaptura        string               `json:"directorio_captura"`
	PgDump                   string               `json:"pg_dump"`
	PgDumpall                string               `json:"pg_dumpall"`
	Psql                     string               `json:"psql"`
	Bases                    []adaptador.Base     `json:"bases"`
	Escritores               []adaptador.Escritor `json:"escritores"`
	EntornoPG                map[string]string    `json:"entorno_pg"`
	LimiteProcesoSegundos    int                  `json:"limite_proceso_segundos"`
	LimiteLiberacionSegundos int                  `json:"limite_liberacion_segundos"`
	MaxArchivoBytes          int64                `json:"max_archivo_bytes"`
	MaxTotalBytes            int64                `json:"max_total_bytes"`
}

func fallo(w io.Writer, clave string) int {
	_ = json.NewEncoder(w).Encode(map[string]string{"error_clave": clave})
	return 1
}

func documento(ruta string) (*os.File, error) {
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0) // #nosec G304 G703 -- operator-owned local configuration path.
	if err != nil {
		return nil, adaptador.ErrLocal
	}
	s, err := f.Stat()
	if err != nil || !s.Mode().IsRegular() || s.Size() > 4<<20 {
		_ = f.Close()
		return nil, adaptador.ErrLocal
	}
	return f, nil
}

func entorno(c configuracion) ([]string, bool) {
	permitido := map[string]bool{"PGHOST": true, "PGPORT": true, "PGUSER": true, "PGSERVICE": true, "PGSERVICEFILE": true, "PGPASSFILE": true, "PGSSLMODE": true, "PGSSLROOTCERT": true}
	var env []string
	for k, v := range c.EntornoPG {
		if !permitido[k] || strings.ContainsAny(v, "\x00\r\n") {
			return nil, false
		}
		env = append(env, k+"="+v)
	}
	// This CLI is deliberately unable to address deployed hosts. Explicit socket
	// paths are allowed for isolated Docker --network none fixtures.
	host := c.EntornoPG["PGHOST"]
	// libpq interprets commas as a host list, including socket-first lists.
	if strings.Contains(host, ",") || (host != "127.0.0.1" && host != "::1" && !strings.HasPrefix(host, "/")) {
		return nil, false
	}
	if c.EntornoPG["PGSERVICE"] != "" || c.EntornoPG["PGSERVICEFILE"] != "" {
		return nil, false
	}
	return env, true
}

func run(ctx context.Context, args []string, out, diag io.Writer) int {
	if len(args) != 2 || args[0] != "-config" {
		return fallo(diag, "copias_captura_argumentos")
	}
	f, err := documento(args[1])
	if err != nil {
		return fallo(diag, "copias_captura_configuracion")
	}
	c, err := leerConfiguracion(f)
	_ = f.Close()
	if err != nil || c.FormatoVersion != 1 || !c.EnsayoLocalSintetico || c.LimiteProcesoSegundos < 1 || c.LimiteProcesoSegundos > 3600 || c.LimiteLiberacionSegundos < 1 || c.LimiteLiberacionSegundos > 300 {
		return fallo(diag, "copias_captura_configuracion")
	}
	env, ok := entorno(c)
	if !ok {
		return fallo(diag, "copias_captura_configuracion")
	}
	f, err = documento(c.Descriptor)
	if err != nil {
		return fallo(diag, "copias_captura_inventario")
	}
	d, err := inventariocopias.LeerDescriptor(f)
	_ = f.Close()
	if err != nil {
		return fallo(diag, "copias_captura_inventario")
	}
	f, err = documento(c.Inventario)
	if err != nil {
		return fallo(diag, "copias_captura_inventario")
	}
	i, err := inventariocopias.LeerInventario(f)
	_ = f.Close()
	if err != nil {
		return fallo(diag, "copias_captura_inventario")
	}
	if copias.CompararInventarios(d.Inventario, i).Estado != copias.Compatible {
		return fallo(diag, "copias_captura_inventario")
	}
	locks, err := os.OpenRoot(c.DirectorioBloqueos)
	if err != nil {
		return fallo(diag, "copias_captura_destino")
	}
	defer locks.Close()
	// A new private directory prevents overwrite/reuse of an earlier operation.
	if os.Mkdir(c.DirectorioCaptura, 0700) != nil {
		return fallo(diag, "copias_captura_destino")
	}
	raiz, err := os.OpenRoot(c.DirectorioCaptura)
	if err != nil {
		return fallo(diag, "copias_captura_destino")
	}
	defer raiz.Close()
	e := adaptador.Ejecutor{Entorno: env, Limite: time.Duration(c.LimiteProcesoSegundos) * time.Second}
	s := aplicacion.Servicio{
		Exclusor:   adaptador.ExclusorLocal{Raiz: locks},
		Escritores: adaptador.Control{Ejecutor: e, Escritores: c.Escritores},
		Inventario: adaptador.InventarioLocal{Raiz: c.RaizInstalada, Descriptor: d, Observado: i},
		Logico:     adaptador.PostgreSQL{Ejecutor: e, Dump: c.PgDump, Globals: c.PgDumpall, Psql: c.Psql, Bases: c.Bases, Raiz: raiz, MaxArchivoBytes: c.MaxArchivoBytes, MaxTotalBytes: c.MaxTotalBytes},
		Ahora:      time.Now, TiempoLiberacion: time.Duration(c.LimiteLiberacionSegundos) * time.Second,
	}
	r, err := s.Capturar(ctx, puertos.Peticion{OrigenRef: c.OrigenRef, OperacionRef: c.OperacionRef, Esperado: d.Inventario, InicioVentana: c.InicioVentana, FinVentana: c.FinVentana})
	if err != nil {
		return fallo(diag, err.Error())
	}
	manifest, err := raiz.OpenFile("captura-parcial.json", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fallo(diag, "copias_captura_destino")
	}
	err = json.NewEncoder(manifest).Encode(r)
	if err == nil {
		err = manifest.Sync()
	}
	closeErr := manifest.Close()
	if err != nil || closeErr != nil {
		return fallo(diag, "copias_captura_destino")
	}
	// Do not expose globals/role names, database aliases or content digests.
	if json.NewEncoder(out).Encode(map[string]any{"estado_clave": "copias_captura_parcial", "completa": false, "valida": false, "publicable": false, "componentes": len(r.Componentes)}) != nil {
		return fallo(diag, "copias_captura_salida")
	}
	return 0
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}
