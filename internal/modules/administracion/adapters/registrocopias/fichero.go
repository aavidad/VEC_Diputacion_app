// Package registrocopias implements a Linux offline append-only executor journal.
// Its hash chain detects accidental corruption, not malicious rewriting by its owner.
package registrocopias

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	unix "syscall"
	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	port "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

const MaxDiario = 64 << 20
const MaxTrama = 16 << 10

type Config struct {
	Directorio        string   `json:"directorio"`
	RaicesRestauradas []string `json:"raices_restauradas"`
	LimiteListado     int      `json:"limite_listado"`
}

// Fichero holds no cached state or open descriptors. Every call reopens under
// flock and verifies the complete committed journal before returning a receipt.
type Fichero struct {
	directorio         string
	limiteListado      int
	observadorAbandono port.ObservadorAbandono
}

func Abrir(c Config) (*Fichero, error) {
	if c.LimiteListado == 0 {
		c.LimiteListado = 25
	}
	if c.LimiteListado < 1 || c.LimiteListado > 100 {
		return nil, port.ErrConfiguracion
	}
	if !filepath.IsAbs(c.Directorio) || len(c.RaicesRestauradas) == 0 {
		return nil, port.ErrConfiguracion
	}
	dir, err := filepath.EvalSymlinks(c.Directorio)
	if err != nil {
		return nil, port.ErrConfiguracion
	}
	for _, raiz := range c.RaicesRestauradas {
		if !filepath.IsAbs(raiz) {
			return nil, port.ErrConfiguracion
		}
		r, err := filepath.EvalSymlinks(raiz)
		if err != nil {
			return nil, port.ErrConfiguracion
		}
		rel, err := filepath.Rel(r, dir)
		if err != nil || rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
			return nil, port.ErrConfiguracion
		}
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return nil, port.ErrConfiguracion
	}
	return &Fichero{directorio: dir, limiteListado: c.LimiteListado}, nil
}

func (f *Fichero) Reservar(ctx context.Context, d port.Declaracion, s operacionescopias.Solicitud) (port.Resultado, error) {
	return f.ejecutar(ctx, peticion{Accion: "reservar", Declaracion: d, Operacion: s.Operacion, Solicitud: &s})
}
func (f *Fichero) Aplicar(ctx context.Context, d port.Declaracion, op string, c operacionescopias.Comando) (port.Resultado, error) {
	return f.ejecutar(ctx, peticion{Accion: "aplicar", Declaracion: d, Operacion: op, Comando: &c})
}
func (f *Fichero) Consultar(ctx context.Context, d port.Declaracion, op string) (port.Resultado, error) {
	return f.ejecutar(ctx, peticion{Accion: "consultar", Declaracion: d, Operacion: op})
}

func (f *Fichero) Listar(ctx context.Context, d port.Declaracion, q port.Consulta) (port.Resultado, error) {
	if f == nil {
		return port.Resultado{}, port.ErrConfiguracion
	}
	if q.Limite == 0 {
		q.Limite = f.limiteListado
	}
	if q.Limite > f.limiteListado {
		return port.Resultado{}, port.ErrEntrada
	}
	return f.ejecutar(ctx, peticion{Accion: "listar", Declaracion: d, Consulta: &q})
}

func (f *Fichero) ejecutar(ctx context.Context, p peticion) (port.Resultado, error) {
	var vacio port.Resultado
	if f == nil {
		return vacio, port.ErrConfiguracion
	}
	if err := validar(p); err != nil {
		return vacio, err
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	dir, err := os.Open(f.directorio) // #nosec G304 -- Local configured canonical directory, never HTTP input.
	if err != nil {
		return vacio, port.ErrIO
	}
	defer func() { _ = dir.Close() }()
	descriptor := dir.Fd()
	if descriptor > math.MaxInt32 {
		return vacio, port.ErrIO
	}
	dirfd := int(descriptor)
	var ds unix.Stat_t
	if unix.Fstat(dirfd, &ds) != nil || ds.Mode&unix.S_IFMT != unix.S_IFDIR || ds.Mode&0077 != 0 || int64(ds.Uid) != int64(os.Geteuid()) {
		return vacio, port.ErrConfiguracion
	}
	fd, err := unix.Openat(dirfd, "operaciones.jsonl", unix.O_RDWR|unix.O_CREAT|unix.O_APPEND|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0600)
	if err != nil || fd < 0 {
		return vacio, port.ErrIO
	}
	file := os.NewFile(uintptr(fd), "operaciones.jsonl")
	defer func() { _ = file.Close() }()
	var st unix.Stat_t
	if unix.Fstat(fd, &st) != nil || st.Mode&unix.S_IFMT != unix.S_IFREG || st.Nlink != 1 || st.Mode&0077 != 0 || st.Uid != ds.Uid {
		return vacio, port.ErrConfiguracion
	}
	if err := bloquear(ctx, fd); err != nil {
		return vacio, err
	}
	defer func() { _ = unix.Flock(fd, unix.LOCK_UN) }()
	// A second append-only high-water journal detects truncation at a complete
	// frame boundary too. Any mismatch blocks; recovery never guesses a commit.
	confirmacion, err := abrirConfirmaciones(dirfd, ds.Uid)
	if err != nil {
		return vacio, err
	}
	defer func() { _ = confirmacion.Close() }()
	// Persist directory entry before any acknowledged append (also on reopen).
	if dir.Sync() != nil {
		return vacio, port.ErrIO
	}
	b, err := io.ReadAll(io.LimitReader(file, MaxDiario+1))
	if err != nil {
		return vacio, port.ErrIO
	}
	if len(b) > MaxDiario || (len(b) > 0 && b[len(b)-1] != '\n') {
		return vacio, port.ErrCorrupto
	}
	m := nuevoMotor()
	var esperadas []byte
	for _, linea := range bytes.Split(bytes.TrimSuffix(b, []byte{'\n'}), []byte{'\n'}) {
		if len(linea) == 0 {
			if len(b) == 0 {
				continue
			}
			return vacio, port.ErrCorrupto
		}
		if len(linea) > MaxTrama {
			return vacio, port.ErrCorrupto
		}
		var t trama
		d := json.NewDecoder(bytes.NewReader(linea))
		d.DisallowUnknownFields()
		if d.Decode(&t) != nil || d.Decode(new(any)) != io.EOF || m.recuperar(t) != nil {
			return vacio, port.ErrCorrupto
		}
		// Reject duplicate keys, aliases and alternative JSON encodings by requiring
		// exact canonical bytes emitted by the single writer.
		canonico, _ := json.Marshal(t)
		if !bytes.Equal(canonico, linea) {
			return vacio, port.ErrCorrupto
		}
		esperadas = append(esperadas, t.SHA256...)
		esperadas = append(esperadas, '\n')
	}
	confirmadas, err := io.ReadAll(io.LimitReader(confirmacion, MaxDiario+1))
	if err != nil {
		return vacio, port.ErrIO
	}
	if !bytes.Equal(confirmadas, esperadas) {
		return vacio, port.ErrCorrupto
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	instante := time.Now().UTC().Format(time.RFC3339Nano)
	if p.Accion == "abandonar_captura" {
		p, err = f.confirmarAbandono(ctx, p)
		if err != nil {
			p.AbandonoDenegado = errors.Is(err, port.ErrAbandonoNoAutorizado)
		}
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	res, evento, efectoErr := m.procesar(p, instante)
	r := registro{Secuencia: m.secuencia + 1, Anterior: m.anterior, Instante: instante, Peticion: p, Recibo: res.Recibo, Auditoria: res.Auditoria, Evento: evento}
	t := trama{Registro: r, SHA256: sello(r)}
	linea, _ := json.Marshal(t)
	linea = append(linea, '\n')
	if len(linea) > MaxTrama || len(b)+len(linea) > MaxDiario {
		return vacio, port.ErrIO
	}
	n, err := file.Write(linea)
	if err != nil || n != len(linea) || file.Sync() != nil {
		return vacio, port.ErrIO
	}
	marca := []byte(t.SHA256 + "\n")
	n, err = confirmacion.Write(marca)
	if err != nil || n != len(marca) || confirmacion.Sync() != nil {
		return vacio, port.ErrIO
	}
	return res, efectoErr
}

func abrirConfirmaciones(dirfd int, uid uint32) (*os.File, error) {
	fd, err := unix.Openat(dirfd, "confirmaciones.sha256", unix.O_RDWR|unix.O_CREAT|unix.O_APPEND|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0600)
	if err != nil || fd < 0 {
		return nil, port.ErrIO
	}
	file := os.NewFile(uintptr(fd), "confirmaciones.sha256")
	var st unix.Stat_t
	if unix.Fstat(fd, &st) != nil || st.Mode&unix.S_IFMT != unix.S_IFREG || st.Nlink != 1 || st.Mode&0077 != 0 || st.Uid != uid {
		_ = file.Close()
		return nil, port.ErrConfiguracion
	}
	return file, nil
}

func bloquear(ctx context.Context, fd int) error {
	for {
		err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return nil
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) {
			return port.ErrIO
		}
		t := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
	}
}
