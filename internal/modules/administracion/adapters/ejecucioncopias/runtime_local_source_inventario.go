package ejecucioncopias

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
)

func (o *OrigenLocal) LeerActual(ctx context.Context, ref string) (p.Lectura, error) {
	if ref != o.c.OrigenRef {
		return p.Lectura{}, ErrOrigenLocal
	}
	observado, err := o.Observar(ctx)
	if err != nil {
		return p.Lectura{}, err
	}
	return p.Lectura{Esperado: o.c.InventarioEsperado, Observado: observado, Politica: o.c.Politica, VersionRef: copias.HuellaInventario(observado)}, nil
}

func (o *OrigenLocal) psqlDirecto(ctx context.Context, base, sql string, limite int) (string, error) {
	b, err := dockerCS11(ctx, nil, limite, "exec", o.c.Contenedor, "env", "-i", "PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C", "PGHOST=/var/run/postgresql", "PGUSER="+o.c.Usuario, o.c.Herramientas["psql"], "-X", "-At", "-v", "ON_ERROR_STOP=1", "-d", base, "-c", sql)
	if err != nil {
		return "", ErrOrigenLocal
	}
	return strings.TrimSpace(string(b)), nil
}

func versionCortaCS11(s string) string {
	campos := strings.Fields(s)
	if len(campos) == 0 || !regexp.MustCompile(`^18\.[0-9]{1,2}$`).MatchString(campos[0]) {
		return ""
	}
	return campos[0]
}

// Observar sólo completa un inventario de ejercicio con un módulo sin
// migraciones. Comprueba imagen, herramientas, bases, extensión, esquema y
// bytes instalados. Si el conjunto real exige más clases, falla cerrado.
func (o *OrigenLocal) Observar(ctx context.Context) (copias.Inventario, error) {
	if o == nil || ctx == nil || ctx.Err() != nil {
		return copias.Inventario{}, ErrOrigenLocal
	}
	if _, err := o.observarContenedor(ctx, true); err != nil {
		return copias.Inventario{}, err
	}
	i := o.c.InventarioEsperado
	if len(i.Modulos) != 1 || len(i.Modulos[0].Migraciones) != 0 || !reflect.DeepEqual(i.Modulos, i.Release.EsquemaEsperado) {
		return copias.Inventario{}, ErrOrigenLocal
	}
	version, err := o.psqlDirecto(ctx, "postgres", "SHOW server_version", 1024)
	if err != nil || versionCortaCS11(version) != i.PostgreSQL.Version {
		return copias.Inventario{}, ErrOrigenLocal
	}
	bases, err := o.psqlDirecto(ctx, "postgres", "SELECT string_agg(datname,',' ORDER BY datname) FROM pg_database WHERE datallowconn AND datname NOT IN ('postgres','template0','template1')", 4096)
	if err != nil || bases != o.c.Base {
		return copias.Inventario{}, ErrOrigenLocal
	}
	extensiones, err := o.psqlDirecto(ctx, o.c.Base, "SELECT coalesce(string_agg(extname||':'||extversion,',' ORDER BY extname),'') FROM pg_extension", 4096)
	if err != nil {
		return copias.Inventario{}, ErrOrigenLocal
	}
	var esperadas []string
	for _, ext := range i.PostgreSQL.Extensiones {
		esperadas = append(esperadas, ext.ID+":"+ext.Version)
	}
	sort.Strings(esperadas)
	if extensiones != strings.Join(esperadas, ",") {
		return copias.Inventario{}, ErrOrigenLocal
	}
	for _, t := range i.PostgreSQL.Herramientas {
		ruta := o.c.Herramientas[t.ID]
		if !rutaHerramientaCS11(ruta) || t.Version != i.PostgreSQL.Version {
			return copias.Inventario{}, ErrOrigenLocal
		}
		b, err := dockerCS11(ctx, nil, 4096, "exec", o.c.Contenedor, "sha256sum", ruta)
		if err != nil || len(strings.Fields(string(b))) != 2 || strings.Fields(string(b))[0] != t.SHA256 {
			return copias.Inventario{}, ErrOrigenLocal
		}
	}
	schema, err := dockerCS11(ctx, nil, 16<<20, "exec", o.c.Contenedor, "env", "-i", "PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C", "PGHOST=/var/run/postgresql", "PGUSER="+o.c.Usuario, o.c.Herramientas["pg_dump"], "--schema-only", "--no-owner", "--no-acl", "--restrict-key=0123456789abcdef0123456789abcdef", "--dbname="+o.c.Base)
	if err != nil {
		return copias.Inventario{}, ErrOrigenLocal
	}
	h := sha256.Sum256(schema)
	clear(schema)
	if hex.EncodeToString(h[:]) != i.Modulos[0].EsquemaSHA256 {
		return copias.Inventario{}, ErrOrigenLocal
	}
	for _, f := range o.c.Fuentes {
		a, err := medirFuenteCS11(ctx, f)
		if err != nil {
			return copias.Inventario{}, err
		}
		match := false
		for _, expected := range append(append([]copias.Artefacto{}, i.Release.Binarios...), i.Release.Componentes...) {
			if expected == a {
				match = true
				break
			}
		}
		if !match {
			return copias.Inventario{}, ErrOrigenLocal
		}
	}
	return i, nil
}

// AutoridadLocal sólo emite una concesión sintética de corta duración para
// el actor y recurso propios; no afirma ser el PDP administrativo central.
func (o *OrigenLocal) Autorizar(ctx context.Context, pet p.Peticion, accion string) (p.Concesion, error) {
	if o == nil || ctx == nil || ctx.Err() != nil || accion != "copiar" || pet.ActorRef != o.c.ActorRef || pet.OrigenRef != o.c.OrigenRef || pet.DestinoRef != o.c.DestinoRef || pet.PoliticaRef != o.c.Politica.Ref || !directorioPrivadoCS11(o.c.RaizPropia) {
		return p.Concesion{}, ErrOrigenLocal
	}
	if _, err := o.observarContenedor(ctx, true); err != nil {
		return p.Concesion{}, err
	}
	return p.Concesion{ActorRef: pet.ActorRef, Accion: accion, RecursoRef: pet.OrigenRef, DecisionRef: "local-sintetico:" + pet.OperacionRef, Vence: time.Now().Add(30 * time.Second).UTC()}, nil
}
func (*OrigenLocal) Aprobar(context.Context, p.Propuesta) (p.Aprobacion, error) {
	return p.Aprobacion{}, ErrOrigenLocal
}

func (o *OrigenLocal) MedirOrigen(ctx context.Context, i copias.Inventario) ([]copias.Artefacto, error) {
	if o == nil || copias.CompararInventarios(o.c.InventarioEsperado, i).Estado != copias.Compatible {
		return nil, ErrOrigenLocal
	}
	if _, err := o.ComprobarExclusion(ctx); err != nil {
		return nil, err
	}
	resultado := make([]copias.Artefacto, 0, len(o.c.Fuentes))
	for _, f := range o.c.Fuentes {
		a, err := medirFuenteCS11(ctx, f)
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, a)
	}
	return resultado, nil
}

func medirFuenteCS11(ctx context.Context, f FuenteLocal) (copias.Artefacto, error) {
	if !rutaSinEnlacesCS11(f.Ruta) || ctx.Err() != nil {
		return copias.Artefacto{}, ErrOrigenLocal
	}
	info, err := os.Lstat(f.Ruta)
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > 1<<30 {
		return copias.Artefacto{}, ErrOrigenLocal
	}
	file, err := os.OpenFile(f.Ruta, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0) // #nosec G304 G703 -- origen regular propio; O_NOFOLLOW y fstat impiden sustituir el archivo final.
	if err != nil {
		return copias.Artefacto{}, ErrOrigenLocal
	}
	defer file.Close()
	actual, err := file.Stat()
	if err != nil || !actual.Mode().IsRegular() || !os.SameFile(info, actual) || actual.Size() != info.Size() {
		return copias.Artefacto{}, ErrOrigenLocal
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(file, actual.Size()+1))
	post, postErr := file.Stat()
	if err != nil || postErr != nil || n != actual.Size() || !os.SameFile(actual, post) || post.Size() != actual.Size() {
		return copias.Artefacto{}, ErrOrigenLocal
	}
	return copias.Artefacto{ID: f.ID, Tipo: f.Tipo, SHA256: hex.EncodeToString(h.Sum(nil)), TamanoBytes: n}, nil
}
