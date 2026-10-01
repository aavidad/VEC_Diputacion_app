package ejecucioncopias

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

// Capturar produce un volcado custom con CREATE y globals del mismo cluster
// propio, bajo la ventana CS04. Docker ejecuta las herramientas de la imagen
// fijada; ninguna herramienta del host ni comando de una petición se interpreta.
func (o *OrigenLocal) Capturar(ctx context.Context, inv copias.Inventario) ([]copias.Artefacto, error) {
	if o == nil || ctx == nil || ctx.Err() != nil || copias.CompararInventarios(o.c.InventarioEsperado, inv).Estado != copias.Compatible {
		return nil, ErrOrigenLocal
	}
	antes, err := o.ComprobarExclusion(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := o.Observar(ctx); err != nil {
		return nil, ErrOrigenLocal
	}
	base := []string{"exec", o.c.Contenedor, "env", "-i", "PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C", "PGHOST=/var/run/postgresql", "PGUSER=" + o.c.Usuario}
	planes := []struct {
		archivo, id, tipo, herramienta string
		args                           []string
	}{
		{"base-0.dump", o.c.Base, "postgresql_logico", "pg_dump", []string{"--format=custom", "--create", "--no-password", "--dbname=" + o.c.Base}},
		{"globals.sql", "postgresql:globals", "postgresql_globals", "pg_dumpall", []string{"--globals-only", "--no-password"}},
	}
	resultado := make([]copias.Artefacto, 0, 2)
	for _, p := range planes {
		cmd := append(append([]string{}, base...), o.c.Herramientas[p.herramienta])
		cmd = append(cmd, p.args...)
		b, err := dockerCS11(ctx, nil, 64<<20, cmd...)
		if err != nil || len(b) == 0 {
			clear(b)
			return nil, ErrOrigenLocal
		}
		if p.tipo == "postgresql_globals" && !strings.Contains(string(b), "PostgreSQL database cluster dump") {
			clear(b)
			return nil, ErrOrigenLocal
		}
		h := sha256.Sum256(b)
		ruta := filepath.Join(o.c.RaizLogica, p.archivo)
		f, err := os.OpenFile(ruta, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600) // #nosec G304 G703 -- nombre fijo y raíz propia validada.
		if err != nil {
			clear(b)
			return nil, ErrOrigenLocal
		}
		n, err := f.Write(b)
		if err == nil {
			err = f.Sync()
		}
		cerrar := f.Close()
		clear(b)
		if err != nil || cerrar != nil || n == 0 {
			_ = os.Remove(ruta)
			return nil, ErrOrigenLocal
		}
		resultado = append(resultado, copias.Artefacto{ID: p.id, Tipo: p.tipo, SHA256: hex.EncodeToString(h[:]), TamanoBytes: int64(n)})
	}
	despues, err := o.ComprobarExclusion(ctx)
	if err != nil || antes != despues {
		return nil, ErrOrigenLocal
	}
	return resultado, nil
}
