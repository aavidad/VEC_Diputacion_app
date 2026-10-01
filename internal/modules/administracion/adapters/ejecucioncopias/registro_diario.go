package ejecucioncopias

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const maxDiarioExterior = 64 << 20
const maxTramaExterior = 16 << 10

type estadoExterior struct {
	Ref, Actor, Correlacion, SolicitudSHA256, ConjuntoRef, DestinoRef, PoliticaRef                       string
	Estado                                                                                               string
	Version                                                                                              uint64
	CopiaPreviaRef, ConjuntoPreviaPlaneadaRef, PlanRef, PreimagenSHA256, HuellaPropuesta, IndiceFinalRef string
	ConjuntoObservadoRef                                                                                 string `json:",omitempty"`
	IndiceObservadoRef                                                                                   string `json:",omitempty"`
	FalloCapturaRef                                                                                      string `json:",omitempty"`
}
type tramaExterior struct {
	Secuencia uint64         `json:"secuencia"`
	Anterior  string         `json:"anterior"`
	Instante  string         `json:"instante"`
	Accion    string         `json:"accion"`
	Estado    estadoExterior `json:"estado"`
	SHA256    string         `json:"sha256"`
}
type diarioExterior struct {
	directorio string
	dir        *os.File
}

func (d *diarioExterior) Close() error {
	if d == nil || d.dir == nil {
		return nil
	}
	return d.dir.Close()
}

func abrirDiarioExterior(directorio string, raices []string) (*diarioExterior, error) {
	if !filepath.IsAbs(directorio) || len(raices) == 0 {
		return nil, errRegistroConfiguracion
	}
	d, err := filepath.EvalSymlinks(directorio)
	if err != nil {
		return nil, errRegistroConfiguracion
	}
	st, err := os.Stat(d)
	if err != nil || !st.IsDir() || st.Mode().Perm()&0077 != 0 {
		return nil, errRegistroConfiguracion
	}
	for _, raiz := range raices {
		if !filepath.IsAbs(raiz) {
			return nil, errRegistroConfiguracion
		}
		r, err := filepath.EvalSymlinks(raiz)
		if err != nil {
			return nil, errRegistroConfiguracion
		}
		rel, err := filepath.Rel(r, d)
		if err != nil || rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
			return nil, errRegistroConfiguracion
		}
	}
	fd, err := syscall.Open("/", syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, errRegistroConfiguracion
	}
	for _, parte := range strings.Split(strings.TrimPrefix(d, "/"), "/") {
		next, e := syscall.Openat(fd, parte, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		_ = syscall.Close(fd)
		if e != nil {
			return nil, errRegistroConfiguracion
		}
		fd = next
	}
	dir := os.NewFile(uintptr(fd), "restauraciones-exterior") // #nosec G115 -- Linux: descriptor válido obtenido de Open/Openat; el kernel usa int.
	if dir == nil || !directorioExteriorPrivado(fd) {
		if dir != nil {
			_ = dir.Close()
		} else {
			_ = syscall.Close(fd)
		}
		return nil, errRegistroConfiguracion
	}
	fijado, e := dir.Stat()
	if e != nil || !os.SameFile(st, fijado) {
		_ = dir.Close()
		return nil, errRegistroConfiguracion
	}
	return &diarioExterior{directorio: d, dir: dir}, nil
}

func directorioExteriorPrivado(fd int) bool {
	var st syscall.Stat_t
	return syscall.Fstat(fd, &st) == nil && st.Mode&syscall.S_IFMT == syscall.S_IFDIR && st.Mode&0777 == 0700 && st.Uid == uint32(os.Geteuid()) // #nosec G115 -- Linux: UID procede del kernel de 32 bits.
}

func (d *diarioExterior) leer(ctx context.Context, ref string) (estadoExterior, error) {
	var encontrado estadoExterior
	ok := false
	err := d.bloqueado(ctx, func(estados map[string]estadoExterior, _ uint64, _ string, _ anexarExterior) error {
		encontrado, ok = estados[ref]
		if !ok {
			return os.ErrNotExist
		}
		return nil
	})
	return encontrado, err
}
func (d *diarioExterior) guardar(ctx context.Context, accion string, e estadoExterior) error {
	return d.bloqueado(ctx, func(estados map[string]estadoExterior, secuencia uint64, anterior string, anexar anexarExterior) error {
		if previo, ok := estados[e.Ref]; ok && previo == e {
			return nil
		}
		if previo, ok := estados[e.Ref]; ok && e.Version != previo.Version+1 {
			return errRegistroTransicion
		}
		return anexar(ctx, accion, e, secuencia, anterior)
	})
}
func (d *diarioExterior) reservarRestauracion(ctx context.Context, e estadoExterior) error {
	return d.bloqueado(ctx, func(estados map[string]estadoExterior, secuencia uint64, anterior string, anexar anexarExterior) error {
		if previo, ok := estados[e.Ref]; ok {
			if previo == e {
				return nil
			}
			return errRegistroVinculo
		}
		return anexar(ctx, "reserva_restauracion", e, secuencia, anterior)
	})
}
func (d *diarioExterior) cas(ctx context.Context, ref, esperada, preimagen, etapa string) (estadoExterior, error) {
	var siguiente estadoExterior
	err := d.bloqueado(ctx, func(estados map[string]estadoExterior, secuencia uint64, anterior string, anexar anexarExterior) error {
		e, ok := estados[ref]
		if !ok {
			return os.ErrNotExist
		}
		if fmtVersion(e.Version) != esperada || e.PreimagenSHA256 != preimagen || e.Estado != "plan_preparado" {
			return errRegistroVinculo
		}
		e.Estado, e.Version = etapa, e.Version+1
		siguiente = e
		return anexar(ctx, "cas", e, secuencia, anterior)
	})
	return siguiente, err
}

// actualizar reads, validates and writes one restoration state while the same
// flock is held. Equal-version writes are never accepted outside this function.
func (d *diarioExterior) actualizar(ctx context.Context, ref, accion string, fn func(estadoExterior) (estadoExterior, error)) error {
	return d.bloqueado(ctx, func(estados map[string]estadoExterior, secuencia uint64, anterior string, anexar anexarExterior) error {
		previo, ok := estados[ref]
		if !ok {
			return os.ErrNotExist
		}
		siguiente, err := fn(previo)
		if err != nil {
			return err
		}
		if siguiente.Ref != previo.Ref || siguiente.Version != previo.Version {
			return errRegistroTransicion
		}
		if siguiente == previo {
			return nil
		}
		siguiente.Version++
		return anexar(ctx, accion, siguiente, secuencia, anterior)
	})
}

type anexarExterior func(context.Context, string, estadoExterior, uint64, string) error

func (d *diarioExterior) bloqueado(ctx context.Context, fn func(map[string]estadoExterior, uint64, string, anexarExterior) error) error {
	if d == nil || d.dir == nil || ctx == nil || ctx.Err() != nil {
		return errRegistroConfiguracion
	}
	dir := d.dir
	if !directorioExteriorPrivado(int(dir.Fd())) { // #nosec G115 -- Linux: descriptor válido obtenido de Open/Openat; el kernel usa int.
		return errRegistroConfiguracion
	}
	fd, err := syscall.Openat(int(dir.Fd()), "restauraciones.jsonl", syscall.O_RDWR|syscall.O_CREAT|syscall.O_APPEND|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600) // #nosec G115 -- Linux: descriptor válido obtenido de Open/Openat; el kernel usa int.
	if err != nil {
		return errRegistroConfiguracion
	}
	f := os.NewFile(uintptr(fd), "restauraciones.jsonl") // #nosec G115 -- Linux: descriptor válido obtenido de Open/Openat; el kernel usa int.
	defer f.Close()
	if !regularPrivado(fd) {
		return errRegistroConfiguracion
	}
	for {
		err = syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
			return errRegistroConfiguracion
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)
	estados, secuencia, anterior, huellas, err := leerExterior(f)
	if err != nil {
		return err
	}
	confirmacion, err := abrirTestigo(int(dir.Fd())) // #nosec G115 -- Linux: descriptor válido obtenido de Open/Openat; el kernel usa int.
	if err != nil {
		return err
	}
	defer confirmacion.Close()
	if !regularPrivado(int(confirmacion.Fd())) { // #nosec G115 -- Linux: descriptor válido obtenido de Open/Openat; el kernel usa int.
		return errRegistroConfiguracion
	}
	marcas, err := io.ReadAll(io.LimitReader(confirmacion, maxDiarioExterior+1))
	if err != nil || !bytes.Equal(marcas, huellas) {
		return errRegistroConfiguracion
	}
	anexar := func(c context.Context, accion string, e estadoExterior, s uint64, a string) error {
		return appendExterior(c, f, confirmacion, dir, accion, e, s, a)
	}
	if err := fn(estados, secuencia, anterior, anexar); err != nil {
		return err
	}
	return nil
}
func appendExterior(ctx context.Context, f, c, dir *os.File, accion string, e estadoExterior, secuencia uint64, anterior string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	t := tramaExterior{Secuencia: secuencia + 1, Anterior: anterior, Instante: time.Now().UTC().Format(time.RFC3339Nano), Accion: accion, Estado: e}
	t.SHA256 = selloExterior(t)
	b, _ := json.Marshal(t)
	b = append(b, '\n')
	if len(b) > maxTramaExterior {
		return errRegistroEntrada
	}
	actual, err := f.Stat()
	if err != nil || actual.Size() < 0 || actual.Size() > maxDiarioExterior-int64(len(b)) {
		return errRegistroConfiguracion
	}
	marca := []byte(t.SHA256 + "\n")
	marcaActual, err := c.Stat()
	if err != nil || marcaActual.Size() < 0 || marcaActual.Size() > maxDiarioExterior-int64(len(marca)) {
		return errRegistroConfiguracion
	}
	if n, err := f.Write(b); err != nil || n != len(b) || f.Sync() != nil {
		return errRegistroConfiguracion
	}
	if n, err := c.Write(marca); err != nil || n != len(marca) || c.Sync() != nil {
		return errRegistroConfiguracion
	}
	if err := dir.Sync(); err != nil {
		return errRegistroConfiguracion
	}
	return nil
}
func leerExterior(f *os.File) (map[string]estadoExterior, uint64, string, []byte, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, 0, "", nil, errRegistroConfiguracion
	}
	b, err := io.ReadAll(io.LimitReader(f, maxDiarioExterior+1))
	if err != nil || len(b) > maxDiarioExterior || (len(b) > 0 && b[len(b)-1] != '\n') {
		return nil, 0, "", nil, errRegistroConfiguracion
	}
	estados := map[string]estadoExterior{}
	var seq uint64
	anterior := ""
	var huellas []byte
	for _, l := range bytes.Split(b, []byte{'\n'}) {
		if len(l) == 0 {
			continue
		}
		if len(l) > maxTramaExterior {
			return nil, 0, "", nil, errRegistroConfiguracion
		}
		var t tramaExterior
		dec := json.NewDecoder(bytes.NewReader(l))
		dec.DisallowUnknownFields()
		if dec.Decode(&t) != nil || dec.Decode(new(any)) != io.EOF {
			return nil, 0, "", nil, errRegistroConfiguracion
		}
		canonico, _ := json.Marshal(t)
		if !bytes.Equal(canonico, l) || t.Secuencia != seq+1 || t.Anterior != anterior || selloExterior(t) != t.SHA256 || t.Estado.Ref == "" {
			return nil, 0, "", nil, errRegistroConfiguracion
		}
		if previo, ok := estados[t.Estado.Ref]; ok {
			if t.Estado.Version != previo.Version+1 || !transicionExteriorValida(previo.Estado, t.Estado.Estado, t.Accion) {
				return nil, 0, "", nil, errRegistroConfiguracion
			}
		} else if t.Accion != "reserva_copia" && t.Accion != "reserva_restauracion" {
			return nil, 0, "", nil, errRegistroConfiguracion
		}
		estados[t.Estado.Ref] = t.Estado
		seq, anterior = t.Secuencia, t.SHA256
		huellas = append(huellas, []byte(t.SHA256+"\n")...)
	}
	return estados, seq, anterior, huellas, nil
}
func transicionExteriorValida(anterior, siguiente, accion string) bool {
	if accion == "abandono_captura" {
		return (anterior == "capturando" || anterior == "captura_pendiente_conciliacion") && siguiente == "abandonada_declarada"
	}
	if accion == "observar_restauracion" {
		if siguiente != "revertida" && siguiente != "instalado_pendiente_conciliacion" {
			return false
		}
		switch anterior {
		case "sustitucion_iniciada", "reversion_iniciada", "pendiente_conciliacion", "instalado_pendiente_conciliacion", "revertida":
			return true
		default:
			return false
		}
	}
	if accion == "aplicar_copia" {
		return (anterior == "solicitada" && siguiente == "capturando") || (anterior == "capturando" && siguiente == "capturada") || (anterior == "capturada" && siguiente == "verificando") || (anterior == "verificando" && (siguiente == "verificando" || siguiente == "verificada_declarada" || siguiente == "no_valida_declarada"))
	}
	if accion == "anotar" || accion == "cas" {
		return anterior != siguiente
	}
	return false
}
func abrirTestigo(dirfd int) (*os.File, error) {
	fd, err := syscall.Openat(dirfd, "restauraciones.confirmadas", syscall.O_RDWR|syscall.O_CREAT|syscall.O_APPEND|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return nil, errRegistroConfiguracion
	}
	return os.NewFile(uintptr(fd), "restauraciones.confirmadas"), nil // #nosec G115 -- Linux: descriptor válido obtenido de Open/Openat; el kernel usa int.
}
func regularPrivado(fd int) bool {
	var st syscall.Stat_t
	return syscall.Fstat(fd, &st) == nil && st.Mode&syscall.S_IFMT == syscall.S_IFREG && st.Nlink == 1 && st.Uid == uint32(os.Geteuid()) && st.Mode&0077 == 0 && st.Size <= maxDiarioExterior // #nosec G115 -- Linux: UID procede del kernel de 32 bits.
}
func selloExterior(t tramaExterior) string {
	t.SHA256 = ""
	b, _ := json.Marshal(t)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func fmtVersion(v uint64) string { return strconv.FormatUint(v, 10) }
