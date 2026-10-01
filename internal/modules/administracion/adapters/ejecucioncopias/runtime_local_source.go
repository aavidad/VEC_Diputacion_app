package ejecucioncopias

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

var ErrOrigenLocal = errors.New("copias_ejecucion_origen_no_comprobable")
var nombreDockerCS11 = regexp.MustCompile(`^vec-cs11-src-[a-z0-9]{8,32}$`)
var nombrePGCS11 = regexp.MustCompile(`^[a-z][a-z0-9_]{1,62}$`)

// FuenteLocal es un componente del conjunto instalado, dentro del árbol
// sintético propio. Los IDs y tipos deben coincidir con CS01 y CS05.
type FuenteLocal struct {
	ID, Tipo, Ruta string
}

type ConfigOrigenLocal struct {
	Contenedor, ImagenSHA256, PGDATA, RaizLogica, Base, Usuario string
	RaizPropia, ActorRef, OrigenRef, DestinoRef                 string
	InventarioEsperado                                          copias.Inventario
	Politica                                                    copias.Politica
	Fuentes                                                     []FuenteLocal
	Herramientas                                                map[string]string
	TiempoMaximo                                                time.Duration
}

// OrigenLocal sólo conoce un cluster efímero propio con PostgreSQL por socket
// Unix y sin red. No es una autoridad de producción ni lee el clon principal.
type OrigenLocal struct {
	c             ConfigOrigenLocal
	mu            sync.Mutex
	lease         *os.File
	identificador string
}

type HechosFuenteLocal struct {
	Version       string
	EsquemaSHA256 string
	Herramientas  []copias.Herramienta
	Extensiones   []copias.Extension
	Fuentes       []copias.Artefacto
}

// MedirHechosFuenteLocal permite preparar el descriptor CS01 del ejercicio
// a partir de bytes y PG reales antes de pedir una copia. No lo autentica;
// CS01/CS03 aún deben validar el descriptor y el paquete que representan.
func MedirHechosFuenteLocal(ctx context.Context, c ConfigOrigenLocal) (HechosFuenteLocal, error) {
	var h HechosFuenteLocal
	if ctx == nil || ctx.Err() != nil || !nombreDockerCS11.MatchString(c.Contenedor) || !huellaRuntime(c.ImagenSHA256) || !nombrePGCS11.MatchString(c.Base) || !nombrePGCS11.MatchString(c.Usuario) || len(c.Herramientas) != 5 || !directorioPrivadoCS11(c.RaizPropia) {
		return h, ErrOrigenLocal
	}
	for _, n := range []string{"postgres", "psql", "pg_dump", "pg_dumpall", "pg_restore"} {
		if !rutaHerramientaCS11(c.Herramientas[n]) {
			return h, ErrOrigenLocal
		}
	}
	o := &OrigenLocal{c: c}
	if _, err := o.observarContenedor(ctx, true); err != nil {
		return h, err
	}
	version, err := o.psqlDirecto(ctx, "postgres", "SHOW server_version", 1024)
	version = versionCortaCS11(version)
	if err != nil || version == "" {
		return h, ErrOrigenLocal
	}
	h.Version = version
	for _, n := range []string{"postgres", "psql", "pg_dump", "pg_dumpall", "pg_restore"} {
		ruta := c.Herramientas[n]
		b, e := dockerCS11(ctx, nil, 4096, "exec", c.Contenedor, "sha256sum", ruta)
		fields := strings.Fields(string(b))
		if e != nil || len(fields) != 2 || !huellaRuntime(fields[0]) {
			return HechosFuenteLocal{}, ErrOrigenLocal
		}
		v, e := dockerCS11(ctx, nil, 4096, "exec", c.Contenedor, ruta, "--version")
		if e != nil || !strings.Contains(string(v), "(PostgreSQL) "+version) {
			return HechosFuenteLocal{}, ErrOrigenLocal
		}
		h.Herramientas = append(h.Herramientas, copias.Herramienta{ID: n, Version: version, SHA256: fields[0]})
	}
	extensiones, err := o.psqlDirecto(ctx, c.Base, "SELECT coalesce(string_agg(extname||':'||extversion,',' ORDER BY extname),'') FROM pg_extension", 4096)
	if err != nil {
		return HechosFuenteLocal{}, ErrOrigenLocal
	}
	if extensiones != "" {
		for _, x := range strings.Split(extensiones, ",") {
			id, v, ok := strings.Cut(x, ":")
			if !ok || id == "" || v == "" {
				return HechosFuenteLocal{}, ErrOrigenLocal
			}
			h.Extensiones = append(h.Extensiones, copias.Extension{ID: id, Version: v})
		}
	}
	schema, err := dockerCS11(ctx, nil, 16<<20, "exec", c.Contenedor, "env", "-i", "PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C", "PGHOST=/var/run/postgresql", "PGUSER="+c.Usuario, c.Herramientas["pg_dump"], "--schema-only", "--no-owner", "--no-acl", "--restrict-key=0123456789abcdef0123456789abcdef", "--dbname="+c.Base)
	if err != nil {
		return HechosFuenteLocal{}, ErrOrigenLocal
	}
	sum := sha256.Sum256(schema)
	clear(schema)
	h.EsquemaSHA256 = hex.EncodeToString(sum[:])
	for _, f := range c.Fuentes {
		if !dentroCS11(c.RaizPropia, f.Ruta) {
			return HechosFuenteLocal{}, ErrOrigenLocal
		}
		a, e := medirFuenteCS11(ctx, f)
		if e != nil {
			return HechosFuenteLocal{}, e
		}
		h.Fuentes = append(h.Fuentes, a)
	}
	return h, nil
}

func NuevoOrigenLocal(c ConfigOrigenLocal) (*OrigenLocal, error) {
	if !nombreDockerCS11.MatchString(c.Contenedor) || !huellaRuntime(c.ImagenSHA256) || !nombrePGCS11.MatchString(c.Base) || !nombrePGCS11.MatchString(c.Usuario) || c.TiempoMaximo < time.Second || c.TiempoMaximo > 30*time.Minute || c.ActorRef == "" || c.OrigenRef == "" || c.DestinoRef == "" || c.OrigenRef == c.DestinoRef || len(c.Fuentes) == 0 || len(c.Fuentes) > 32 || len(copias.ValidarInventario(c.InventarioEsperado)) != 0 || len(copias.ValidarPolitica(c.Politica)) != 0 {
		return nil, ErrOrigenLocal
	}
	if c.InventarioEsperado.PostgreSQL.RuntimeSHA256 != c.ImagenSHA256 || c.InventarioEsperado.PostgreSQL.ClusterRef != c.OrigenRef || len(c.InventarioEsperado.PostgreSQL.Bases) != 1 || c.InventarioEsperado.PostgreSQL.Bases[0] != c.Base {
		return nil, ErrOrigenLocal
	}
	for _, path := range []string{c.RaizPropia, c.PGDATA, c.RaizLogica} {
		if !rutaScratchCS11(path) || filepath.Clean(path) != path || !directorioPrivadoCS11(path) {
			return nil, ErrOrigenLocal
		}
	}
	if !dentroCS11(c.RaizPropia, c.PGDATA) || !dentroCS11(c.RaizPropia, filepath.Dir(c.PGDATA)) || !directorioPrivadoCS11(filepath.Dir(c.PGDATA)) || !dentroCS11(c.RaizPropia, c.RaizLogica) || c.PGDATA == c.RaizLogica {
		return nil, ErrOrigenLocal
	}
	if len(c.Herramientas) != 5 {
		return nil, ErrOrigenLocal
	}
	for _, n := range []string{"postgres", "psql", "pg_dump", "pg_dumpall", "pg_restore"} {
		if !rutaHerramientaCS11(c.Herramientas[n]) {
			return nil, ErrOrigenLocal
		}
	}
	requeridos := map[string]string{}
	for _, a := range append(append([]copias.Artefacto{}, c.InventarioEsperado.Release.Binarios...), c.InventarioEsperado.Release.Componentes...) {
		requeridos[a.ID] = a.Tipo
	}
	for _, a := range c.InventarioEsperado.PostgreSQL.Almacenes {
		requeridos[a.ID] = a.Tipo
	}
	if len(c.Fuentes) != len(requeridos) {
		return nil, ErrOrigenLocal
	}
	vistos := map[string]bool{}
	for _, f := range c.Fuentes {
		if f.ID == "" || f.Tipo == "" || vistos[f.ID] || requeridos[f.ID] != f.Tipo || !dentroCS11(c.RaizPropia, f.Ruta) || !rutaSinEnlacesCS11(f.Ruta) {
			return nil, ErrOrigenLocal
		}
		vistos[f.ID] = true
		info, err := os.Stat(f.Ruta)
		if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > 1<<30 {
			return nil, ErrOrigenLocal
		}
	}
	return &OrigenLocal{c: c}, nil
}

// PrepararFuenteSintetica inicia exclusivamente el cluster propio vacío y le
// aplica un esquema fijo de prueba. La operación falla si ya existe el nombre
// Docker; nunca sustituye ni modifica un contenedor previo.
func PrepararFuenteSintetica(ctx context.Context, c ConfigOrigenLocal) error {
	if ctx == nil || ctx.Err() != nil || !nombreDockerCS11.MatchString(c.Contenedor) || !huellaRuntime(c.ImagenSHA256) || !nombrePGCS11.MatchString(c.Base) || !nombrePGCS11.MatchString(c.Usuario) || !dentroCS11(c.RaizPropia, c.PGDATA) || !dentroCS11(c.RaizPropia, filepath.Dir(c.PGDATA)) || !directorioPrivadoCS11(filepath.Dir(c.PGDATA)) || !directorioPrivadoCS11(c.PGDATA) {
		return ErrOrigenLocal
	}
	entradas, err := os.ReadDir(c.PGDATA)
	if err != nil || len(entradas) != 0 {
		return ErrOrigenLocal
	}
	if _, err := dockerCS11(ctx, nil, 1024, "inspect", c.Contenedor); err == nil {
		return ErrOrigenLocal
	}
	uid := strconv.Itoa(os.Geteuid())
	gid := strconv.Itoa(os.Getegid())
	args := []string{"run", "--detach", "--name", c.Contenedor, "--network", "none", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--user", uid + ":" + gid, "--pids-limit", "128", "--memory", "1073741824", "--cpus", "2", "--tmpfs", "/tmp:rw,nosuid,nodev,size=67108864", "--tmpfs", "/var/run/postgresql:rw,nosuid,nodev,size=16777216,mode=1777", "--mount", "type=bind,src=" + filepath.Dir(c.PGDATA) + ",dst=/var/lib/postgresql", "--env", "PGDATA=/var/lib/postgresql/data", "--env", "POSTGRES_USER=" + c.Usuario, "--env", "POSTGRES_DB=" + c.Base, "--env", "POSTGRES_HOST_AUTH_METHOD=trust", "sha256:" + c.ImagenSHA256, "postgres", "-c", "listen_addresses=", "-c", "archive_mode=off"}
	if _, err = dockerCS11(ctx, nil, 1024, args...); err != nil {
		return ErrOrigenLocal
	}
	o := &OrigenLocal{c: c}
	defer func() {
		if err != nil {
			_, _ = dockerCS11(context.WithoutCancel(ctx), nil, 1024, "rm", "--force", c.Contenedor)
		}
	}()
	ready := false
	for n := 0; n < 100; n++ {
		if _, e := o.observarContenedor(ctx, true); e == nil {
			if _, e = dockerCS11(ctx, nil, 1024, "exec", c.Contenedor, "pg_isready", "-h", "/var/run/postgresql", "-U", c.Usuario, "-d", c.Base); e == nil {
				ready = true
				break
			}
		}
		select {
		case <-ctx.Done():
			err = ErrOrigenLocal
			return err
		case <-time.After(100 * time.Millisecond):
		}
	}
	if !ready {
		err = ErrOrigenLocal
		return err
	}
	_, err = dockerCS11(ctx, strings.NewReader(esquemaSinteticoCS11), 4096, "exec", "--interactive", c.Contenedor, "env", "-i", "PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C", "PGHOST=/var/run/postgresql", "PGUSER="+c.Usuario, "psql", "-X", "-q", "-v", "ON_ERROR_STOP=1", "-d", c.Base)
	if err != nil {
		return ErrOrigenLocal
	}
	return nil
}

const esquemaSinteticoCS11 = `
CREATE ROLE cs06_ejecutor LOGIN SUPERUSER;
CREATE ROLE vec_cs11_lector NOLOGIN;
CREATE TABLE public.cs11_registros (
 id bigint PRIMARY KEY,
 codigo text UNIQUE NOT NULL CHECK (length(codigo) BETWEEN 1 AND 32),
 valor integer NOT NULL CHECK (valor >= 0)
);
INSERT INTO public.cs11_registros (id,codigo,valor) VALUES (1,'sintetico_a',10),(2,'sintetico_b',20);
GRANT USAGE ON SCHEMA public TO vec_cs11_lector;
GRANT SELECT ON TABLE public.cs11_registros TO vec_cs11_lector;
`

// RetirarFuenteSintetica elimina únicamente el contenedor propio, conservando
// su árbol privado para examinar o destruir por el operador del ensayo.
func RetirarFuenteSintetica(ctx context.Context, c ConfigOrigenLocal) error {
	if !nombreDockerCS11.MatchString(c.Contenedor) || !huellaRuntime(c.ImagenSHA256) {
		return ErrOrigenLocal
	}
	o := &OrigenLocal{c: c}
	if _, err := o.observarContenedor(ctx, true); err != nil {
		if _, err = o.observarContenedor(ctx, false); err != nil {
			return ErrOrigenLocal
		}
	}
	_, err := dockerCS11(ctx, nil, 1024, "rm", "--force", c.Contenedor)
	return err
}
