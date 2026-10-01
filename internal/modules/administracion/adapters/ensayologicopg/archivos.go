package ensayologicopg

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"syscall"
)

var errEntrada = errors.New("ensayo_logico_entrada_no_admitida")

// El archivo abierto se comprueba por resultado y se copia a un directorio
// exclusivo. La herramienta de restauración sólo monta esa copia de lectura.
func copiarArchivo(ctx context.Context, a Archivo, destino string, limite int64) error {
	if ctx == nil || ctx.Err() != nil {
		return errEntrada
	}
	f, err := os.OpenFile(a.Ruta, os.O_RDONLY|syscall.O_NONBLOCK, 0) // #nosec G304 G703 -- ruta offline explícita; modo no bloqueante permite rechazar FIFO/dispositivos tras fstat.
	if err != nil {
		return errEntrada
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > limite {
		return errEntrada
	}
	g, err := os.OpenFile(destino, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600) // #nosec G304 G703 -- nombre fijo dentro del directorio creado exclusivamente por este ensayo.
	if err != nil {
		return errEntrada
	}
	h := sha256.New()
	n, escribir := io.Copy(io.MultiWriter(g, h), io.LimitReader(lectorContexto{ctx, f}, limite+1))
	cerrar := g.Close()
	if escribir != nil || cerrar != nil || n != info.Size() || n > limite || hex.EncodeToString(h.Sum(nil)) != a.SHA256 {
		return errEntrada
	}
	return nil
}

var metaRestriccion = regexp.MustCompile(`^\\(un)?restrict [A-Za-z0-9]+$`)

// Los globals de pg_dumpall pueden llevar restrict/unrestrict. No se admite
// ningún otro metacomando psql, incluidos shell, includes o reconexiones. Los
// roles/ACL se restauran completos; no se filtran sentencias de negocio.
func validarGlobals(ctx context.Context, ruta string, limite int64) error {
	if ctx == nil || ctx.Err() != nil {
		return errEntrada
	}
	f, err := os.Open(ruta) // #nosec G304 G703 -- copia privada creada por este adaptador, nunca un destino externo.
	if err != nil {
		// Propagar un error cerrado impide divulgar rutas o contenido SQL.
		return errEntrada
	}
	defer f.Close()
	s := bufio.NewScanner(io.LimitReader(lectorContexto{ctx, f}, limite+1))
	s.Buffer(make([]byte, 4096), 1<<20)
	for s.Scan() {
		if ctx.Err() != nil {
			return errEntrada
		}
		linea := s.Text()
		if strings.ContainsRune(linea, '\\') && !metaRestriccion.MatchString(linea) {
			return errEntrada
		}
	}
	if s.Err() != nil || ctx.Err() != nil {
		return errEntrada
	}
	return nil
}

// La lectura de entradas también pertenece al plazo del ensayo. Un contexto
// cancelado no inicia otro bloque de lectura ni sigue calculando la huella.
type lectorContexto struct {
	ctx    context.Context
	lector io.Reader
}

func (l lectorContexto) Read(p []byte) (int, error) {
	if err := l.ctx.Err(); err != nil {
		return 0, err
	}
	return l.lector.Read(p)
}

func validarDump(ruta string) error {
	f, err := os.Open(ruta) // #nosec G304 G703 -- copia privada creada por este adaptador.
	if err != nil {
		return errEntrada
	}
	defer f.Close()
	var cabecera [5]byte
	_, err = io.ReadFull(f, cabecera[:])
	if err != nil || !bytes.Equal(cabecera[:], []byte("PGDMP")) {
		return errEntrada
	}
	return nil
}
