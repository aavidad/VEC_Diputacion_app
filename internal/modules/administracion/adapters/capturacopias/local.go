package capturacopias

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"os"
	"syscall"

	"vec-diputacion-granada/internal/modules/administracion/adapters/inventariocopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

var ErrLocal = errors.New("copias_captura_local")

// ExclusorLocal coordina procesos del MISMO host y directorio privado. No
// pretende exclusión distribuida: otros hosts necesitan otro adaptador.
type ExclusorLocal struct{ Raiz *os.Root }

func (e ExclusorLocal) Adquirir(ctx context.Context, origen string) (func() error, error) {
	if ctx.Err() != nil || e.Raiz == nil {
		return nil, ErrLocal
	}
	privado, err := e.Raiz.Stat(".")
	if err != nil || !privado.IsDir() || privado.Mode().Perm()&0077 != 0 {
		return nil, ErrLocal
	}
	h := sha256.Sum256([]byte(origen))
	f, err := e.Raiz.OpenFile(hex.EncodeToString(h[:])+".lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return nil, ErrLocal
	}
	info, err := f.Stat()
	fd := f.Fd()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || fd > uintptr(math.MaxInt) || syscall.Flock(int(fd), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		_ = f.Close()
		return nil, ErrLocal
	}
	return f.Close, nil // close libera flock; no borrar el inode compartido.
}

type InventarioLocal struct {
	Raiz       string
	Descriptor inventariocopias.Descriptor
	Observado  copias.Inventario
}

func (i InventarioLocal) Observar(ctx context.Context) (copias.Inventario, error) {
	if ctx.Err() != nil {
		return copias.Inventario{}, ErrLocal
	}
	informe := inventariocopias.Inventariar(i.Raiz, i.Descriptor, i.Observado)
	if informe.Resultado.Estado != copias.Compatible {
		return copias.Inventario{}, ErrLocal
	}
	return i.Observado, nil
}
