package capturacopias

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

type Base struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
}

type PostgreSQL struct {
	Ejecutor        Ejecutor
	Dump            string
	Globals         string
	Psql            string
	Bases           []Base
	Raiz            *os.Root
	MaxArchivoBytes int64
	MaxTotalBytes   int64
}

var nombreBase = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,62}$`)

type salidaAcotada struct{ bytes.Buffer }

func (w *salidaAcotada) Write(p []byte) (int, error) {
	if w.Len()+len(p) > 4096 {
		return 0, ErrLocal
	}
	return w.Buffer.Write(p)
}

type acotado struct {
	f         *os.File
	h         hash.Hash
	restantes int64
	n         int64
}

func (w *acotado) Write(p []byte) (int, error) {
	if int64(len(p)) > w.restantes {
		return 0, ErrLocal
	}
	n, err := w.f.Write(p)
	_, _ = w.h.Write(p[:n])
	w.restantes -= int64(n)
	w.n += int64(n)
	return n, err
}

func (p PostgreSQL) archivo(ctx context.Context, nombre, id, tipo string, cmd Comando, limite int64) (a copias.Artefacto, err error) {
	f, err := p.Raiz.OpenFile(nombre, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return a, ErrLocal
	}
	w := &acotado{f: f, h: sha256.New(), restantes: limite}
	err = p.Ejecutor.Ejecutar(ctx, cmd, w)
	if err == nil {
		err = f.Sync()
	}
	if e := f.Close(); e != nil {
		err = ErrLocal
	}
	if err != nil || w.n == 0 {
		_ = p.Raiz.Remove(nombre)
		return a, ErrLocal
	}
	return copias.Artefacto{ID: id, Tipo: tipo, SHA256: hex.EncodeToString(w.h.Sum(nil)), TamanoBytes: w.n}, nil
}

// Capturar conserva propietarios, ACL, creación de bases, roles, membresías y
// tablespaces. Globals contiene secretos y permanece SOLO en la raíz privada.
// No omite constraints, triggers ni funciones. No restaura ni verifica contenido.
func (p PostgreSQL) Capturar(ctx context.Context, i copias.Inventario) ([]copias.Artefacto, error) {
	if p.Raiz == nil || p.MaxArchivoBytes <= 0 || p.MaxTotalBytes <= 0 || len(p.Bases) == 0 || len(p.Bases) != len(i.PostgreSQL.Bases) {
		return nil, ErrLocal
	}
	seenID, seenNombre := map[string]bool{}, map[string]bool{}
	for _, b := range p.Bases {
		if !nombreBase.MatchString(b.Nombre) || seenID[b.ID] || seenNombre[b.Nombre] {
			return nil, ErrLocal
		}
		found := false
		for _, id := range i.PostgreSQL.Bases {
			found = found || b.ID == id
		}
		if !found {
			return nil, ErrLocal
		}
		seenID[b.ID] = true
		seenNombre[b.Nombre] = true
	}
	// Pin de los dos ejecutables con las huellas CS01 del inventario esperado.
	for _, tool := range []struct{ id, ruta string }{{"pg_dump", p.Dump}, {"pg_dumpall", p.Globals}, {"psql", p.Psql}} {
		f, err := os.OpenFile(tool.ruta, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0) // #nosec G304 G703 -- private operator config supplies tool, never manifest or HTTP.
		if err != nil {
			return nil, ErrLocal
		}
		info, err := f.Stat()
		if err != nil || !info.Mode().IsRegular() || info.Size() > 128<<20 {
			_ = f.Close()
			return nil, ErrLocal
		}
		h := sha256.New()
		_, err = io.Copy(h, io.LimitReader(f, 128<<20))
		_ = f.Close()
		found := false
		for _, t := range i.PostgreSQL.Herramientas {
			found = found || t.ID == tool.id && t.SHA256 == hex.EncodeToString(h.Sum(nil)) && t.Version == i.PostgreSQL.Version
		}
		if err != nil || !found {
			return nil, ErrLocal
		}
		var version salidaAcotada
		if p.Ejecutor.Ejecutar(ctx, Comando{tool.ruta, []string{"--version"}}, &version) != nil {
			return nil, ErrLocal
		}
		fields := strings.Fields(version.String())
		if len(fields) < 3 || fields[0] != tool.id || fields[1] != "(PostgreSQL)" || fields[2] != i.PostgreSQL.Version {
			return nil, ErrLocal
		}
	}
	// Version read from the real source, not inferred from configuration/tool exit.
	var version salidaAcotada
	if p.Ejecutor.Ejecutar(ctx, Comando{p.Psql, []string{"--no-psqlrc", "--no-password", "--tuples-only", "--no-align", "--dbname=" + p.Bases[0].Nombre, "--command=SHOW server_version"}}, &version) != nil {
		return nil, ErrLocal
	}
	fields := strings.Fields(version.String())
	if len(fields) == 0 || fields[0] != i.PostgreSQL.Version {
		return nil, ErrLocal
	}
	var artefactos []copias.Artefacto
	restantes := p.MaxTotalBytes
	for n, b := range p.Bases {
		a, err := p.archivo(ctx, "base-"+strconv.Itoa(n)+".dump", b.ID, "postgresql_logico", Comando{p.Dump, []string{"--format=custom", "--create", "--no-password", "--dbname=" + b.Nombre}}, min(p.MaxArchivoBytes, restantes))
		if err != nil {
			return nil, ErrLocal
		}
		restantes -= a.TamanoBytes
		artefactos = append(artefactos, a)
	}
	a, err := p.archivo(ctx, "globals.sql", "postgresql:globals", "postgresql_globals", Comando{p.Globals, []string{"--globals-only", "--no-password"}}, min(p.MaxArchivoBytes, restantes))
	if err != nil {
		return nil, ErrLocal
	}
	return append(artefactos, a), nil
}
