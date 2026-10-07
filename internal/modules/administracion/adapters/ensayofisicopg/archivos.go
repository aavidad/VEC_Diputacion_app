package ensayofisicopg

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"syscall"
)

var errEntrada = errors.New("ensayo_fisico_entrada_no_admitida")

// El archivo abierto se comprueba por resultado y se copia a un directorio
// exclusivo. La herramienta de restauración sólo monta esa copia de lectura.
func copiarArchivo(ctx context.Context, a Archivo, destino string, limite int64) error {
	if ctx == nil || ctx.Err() != nil {
		return errEntrada
	}
	f, err := os.OpenFile(a.Ruta, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0) // #nosec G304 G703 -- ruta offline explícita; modo no bloqueante permite rechazar FIFO/dispositivos tras fstat.
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

func huellaContenido(ctx context.Context, ruta string) (string, int64, error) {
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0) // #nosec G304 -- raíz contenido extraída a scratch propio; rechaza enlaces y archivos especiales.
	if err != nil {
		return "", 0, errEntrada
	}
	defer f.Close()
	i, err := f.Stat()
	if err != nil {
		return "", 0, errEntrada
	}
	if i.IsDir() {
		return "", 0, nil
	}
	if !i.Mode().IsRegular() || i.Size() > 1<<30 {
		return "", 0, errEntrada
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(lectorContexto{ctx, f}, i.Size()+1))
	if err != nil || n != i.Size() {
		return "", 0, errEntrada
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}
