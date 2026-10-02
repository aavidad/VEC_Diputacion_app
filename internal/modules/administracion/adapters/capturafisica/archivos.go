package capturafisica

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

type presupuesto struct {
	bytes    int64
	entradas int
}
type registro struct {
	ruta string
	info os.FileInfo
	sha  string
}
type lectorContexto struct {
	ctx context.Context
	r   io.Reader
}

func (l lectorContexto) Read(p []byte) (int, error) {
	if e := l.ctx.Err(); e != nil {
		return 0, e
	}
	return l.r.Read(p)
}

func capturarFuente(ctx context.Context, fuente Fuente, salida *os.Root, nombre string, limite *presupuesto, expected *copias.Artefacto) (copias.Artefacto, error) {
	var cero copias.Artefacto
	info, e := os.Lstat(fuente.Ruta)
	if e != nil || (!info.IsDir() && !info.Mode().IsRegular()) {
		return cero, ErrOrigen
	}
	base, entrada := fuente.Ruta, "."
	if !info.IsDir() {
		base = filepath.Dir(fuente.Ruta)
		entrada = filepath.Base(fuente.Ruta)
	}
	origen, e := os.OpenRoot(base)
	if e != nil {
		return cero, ErrOrigen
	}
	defer origen.Close()
	f, e := salida.OpenFile(nombre, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return cero, ErrOrigen
	}
	h := sha256.New()
	contador := &contadorEscritura{w: io.MultiWriter(f, h)}
	tw := tar.NewWriter(contador)
	var registros []registro
	e = agregar(ctx, origen, entrada, "contenido", tw, limite, &registros)
	if e == nil && expected != nil {
		if len(registros) != 1 || registros[0].info.IsDir() || registros[0].sha != expected.SHA256 || registros[0].info.Size() != expected.TamanoBytes {
			e = ErrCambio
		}
	}
	if e == nil {
		e = verificar(ctx, origen, registros)
	}
	if x := tw.Close(); e == nil {
		e = x
	}
	if x := f.Sync(); e == nil {
		e = x
	}
	if x := f.Close(); e == nil {
		e = x
	}
	if e != nil {
		return cero, e
	}
	return copias.Artefacto{ID: "fisica:" + fuente.ID, Tipo: fuente.Tipo, SHA256: hex.EncodeToString(h.Sum(nil)), TamanoBytes: contador.n}, nil
}

type contadorEscritura struct {
	w io.Writer
	n int64
}

func (c *contadorEscritura) Write(p []byte) (int, error) {
	n, e := c.w.Write(p)
	c.n += int64(n)
	return n, e
}

func agregar(ctx context.Context, root *os.Root, ruta, nombre string, tw *tar.Writer, limite *presupuesto, registros *[]registro) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if limite.entradas == 0 {
		return ErrLimite
	}
	limite.entradas--
	info, e := root.Lstat(ruta)
	if e != nil || (!info.IsDir() && !info.Mode().IsRegular()) {
		return ErrOrigen
	}
	// NOFOLLOW rejects final symlinks; Root confines all parent components.
	f, e := root.OpenFile(ruta, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if e != nil {
		return ErrOrigen
	}
	defer f.Close()
	actual, e := f.Stat()
	if e != nil || actual.Mode() != info.Mode() || actual.Size() != info.Size() {
		return ErrCambio
	}
	hdr, e := tar.FileInfoHeader(actual, "")
	if e != nil {
		return ErrOrigen
	}
	hdr.Name = nombre
	hdr.Uname = ""
	hdr.Gname = ""
	hdr.Format = tar.FormatPAX
	if e = tw.WriteHeader(hdr); e != nil {
		return ErrOrigen
	}
	reg := registro{ruta: ruta, info: actual}
	if actual.IsDir() {
		nombres, e := f.ReadDir(-1)
		if e != nil {
			return ErrOrigen
		}
		sort.Slice(nombres, func(i, j int) bool { return nombres[i].Name() < nombres[j].Name() })
		*registros = append(*registros, reg)
		for _, n := range nombres {
			if e = agregar(ctx, root, filepath.Join(ruta, n.Name()), nombre+"/"+n.Name(), tw, limite, registros); e != nil {
				return e
			}
		}
	} else {
		if actual.Size() > limite.bytes {
			return ErrLimite
		}
		limite.bytes -= actual.Size()
		hash := sha256.New()
		n, e := io.Copy(io.MultiWriter(tw, hash), io.LimitReader(lectorContexto{ctx, f}, actual.Size()+1))
		if e != nil {
			return ErrOrigen
		}
		if n != actual.Size() {
			return ErrCambio
		}
		reg.sha = hex.EncodeToString(hash.Sum(nil))
		*registros = append(*registros, reg)
	}
	return nil
}

// Relee contenido y metadatos semánticos, incluidos los directorios vacíos.
// No usa ctime, inode ni xattrs como criterio de equivalencia.
func verificar(ctx context.Context, root *os.Root, registros []registro) error {
	for _, reg := range registros {
		info, e := root.Lstat(reg.ruta)
		if e != nil || info.Mode() != reg.info.Mode() || !info.ModTime().Equal(reg.info.ModTime()) || (!info.IsDir() && info.Size() != reg.info.Size()) {
			return ErrCambio
		}
		antes, ok1 := reg.info.Sys().(*syscall.Stat_t)
		despues, ok2 := info.Sys().(*syscall.Stat_t)
		if !ok1 || !ok2 || antes.Uid != despues.Uid || antes.Gid != despues.Gid {
			return ErrCambio
		}
		if info.IsDir() {
			continue
		}
		f, e := root.OpenFile(reg.ruta, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
		if e != nil {
			return ErrOrigen
		}
		s, e := f.Stat()
		if e != nil || !s.Mode().IsRegular() {
			_ = f.Close()
			return ErrOrigen
		}
		h := sha256.New()
		n, e := io.Copy(h, io.LimitReader(lectorContexto{ctx, f}, reg.info.Size()+1))
		cl := f.Close()
		if e != nil || cl != nil || n != reg.info.Size() || hex.EncodeToString(h.Sum(nil)) != reg.sha {
			return ErrCambio
		}
	}
	return nil
}

// copiarRaw conserva también el artefacto release con su identidad original;
// lo obtiene del TAR ya medido, sin volver a abrir su ruta de origen.
func copiarRaw(ctx context.Context, root *os.Root, archivo, nombre string, expected copias.Artefacto, limite *presupuesto) (copias.Artefacto, error) {
	var cero copias.Artefacto
	if expected.TamanoBytes > limite.bytes {
		return cero, ErrLimite
	}
	limite.bytes -= expected.TamanoBytes
	entrada, e := root.Open(archivo)
	if e != nil {
		return cero, ErrOrigen
	}
	defer entrada.Close()
	tr := tar.NewReader(entrada)
	hdr, e := tr.Next()
	if e != nil || hdr.Name != "contenido" || hdr.Typeflag != tar.TypeReg || hdr.Size != expected.TamanoBytes {
		return cero, ErrCambio
	}
	salida, e := root.OpenFile(nombre, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return cero, ErrOrigen
	}
	h := sha256.New()
	n, e := io.Copy(io.MultiWriter(salida, h), lectorContexto{ctx, tr})
	if e == nil {
		if _, x := tr.Next(); x != io.EOF {
			e = ErrCambio
		}
	}
	if x := salida.Sync(); e == nil {
		e = x
	}
	if x := salida.Close(); e == nil {
		e = x
	}
	if e != nil || n != expected.TamanoBytes || hex.EncodeToString(h.Sum(nil)) != expected.SHA256 {
		return cero, ErrCambio
	}
	return expected, nil
}
