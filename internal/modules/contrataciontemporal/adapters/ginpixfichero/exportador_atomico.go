package ginpixfichero

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"
	"syscall"
)

const (
	// AlcanceComprobanteExportacionLocal identifica persistencia local. No
	// representa firma, entrega, acuse ni confirmacion de GINPIX.
	AlcanceComprobanteExportacionLocal = "LOCAL"

	permisosFicheroExportado = fs.FileMode(0o600)
	maximoBytesReferencia    = 160
	maximosIntentosTemporal  = 16
	prefijoFicheroFinal      = "ginpix-"
	sufijoFicheroFinal       = ".json"
	prefijoFicheroTemporal   = ".ginpix-"
	sufijoFicheroTemporal    = ".tmp"
)

var ErrExportacionLocalGINPIX = errors.New(
	"contratacion temporal: exportacion local ginpix fallida",
)

// ExportadorAtomico conserva abierto el directorio configurado de confianza.
// Exportar no recibe rutas ni nombres y os.Root confina todas sus operaciones
// al descriptor del directorio original.
type ExportadorAtomico struct {
	mu   sync.RWMutex
	raiz *os.Root
}

type datosComprobanteExportacionLocal struct {
	rutaRelativa string
	huellaSHA256 string
	metadatos    MetadatosPreparacionExportacion
	tamanoBytes  int64
	permisos     fs.FileMode
	replay       bool
}

// ComprobanteExportacionLocal solo acredita que los bytes exactos existen en
// el directorio local configurado. No es firma, entrega, acuse ni confirmacion
// de GINPIX, y no concede autoridad sobre ningun sistema externo.
type ComprobanteExportacionLocal struct {
	datos *datosComprobanteExportacionLocal
}

// NuevoExportadorAtomico abre un directorio configurado ya existente. Nunca
// crea directorios y rechaza que la ruta configurada sea ella misma un enlace.
func NuevoExportadorAtomico(directorio string) (*ExportadorAtomico, error) {
	if directorio == "" {
		return nil, ErrExportacionLocalGINPIX
	}
	informacion, err := os.Lstat(directorio)
	if err != nil || !informacion.IsDir() || informacion.Mode()&os.ModeSymlink != 0 {
		return nil, ErrExportacionLocalGINPIX
	}
	raiz, err := os.OpenRoot(directorio)
	if err != nil {
		return nil, ErrExportacionLocalGINPIX
	}
	informacionRaiz, err := raiz.Stat(".")
	if err != nil || !informacionRaiz.IsDir() {
		_ = raiz.Close()
		return nil, ErrExportacionLocalGINPIX
	}
	return &ExportadorAtomico{raiz: raiz}, nil
}

// Cerrar libera el descriptor del directorio. Espera a que terminen las
// exportaciones concurrentes ya iniciadas.
func (e *ExportadorAtomico) Cerrar() error {
	if e == nil {
		return ErrExportacionLocalGINPIX
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.raiz == nil {
		return nil
	}
	raiz := e.raiz
	e.raiz = nil
	if err := raiz.Close(); err != nil {
		return ErrExportacionLocalGINPIX
	}
	return nil
}

// Exportar escribe, sincroniza y publica sin sobrescribir. Un EEXIST solo se
// convierte en replay cuando el objeto abierto con O_NOFOLLOW es regular,
// 0600 y contiene la huella exacta de la preparacion.
func (e *ExportadorAtomico) Exportar(
	ctx context.Context,
	preparacion PreparacionExportacion,
) (ComprobanteExportacionLocal, error) {
	if e == nil || !contextoExportacionActivo(ctx) {
		return ComprobanteExportacionLocal{}, ErrExportacionLocalGINPIX
	}
	datos, err := prepararDatosExportacion(preparacion)
	if err != nil || !contextoExportacionActivo(ctx) {
		return ComprobanteExportacionLocal{}, ErrExportacionLocalGINPIX
	}

	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.raiz == nil {
		return ComprobanteExportacionLocal{}, ErrExportacionLocalGINPIX
	}
	return exportarEnRaiz(ctx, e.raiz, datos)
}

type datosExportacionLocal struct {
	contenido       []byte
	huella          [sha256.Size]byte
	huellaTexto     string
	metadatos       MetadatosPreparacionExportacion
	nombreFinal     string
	comprobanteBase ComprobanteExportacionLocal
}

func prepararDatosExportacion(
	preparacion PreparacionExportacion,
) (datosExportacionLocal, error) {
	if preparacion.Validar() != nil {
		return datosExportacionLocal{}, ErrExportacionLocalGINPIX
	}
	metadatos, err := preparacion.Metadatos()
	if err != nil || !metadatosExportacionAcotados(metadatos) {
		return datosExportacionLocal{}, ErrExportacionLocalGINPIX
	}
	contenido, err := preparacion.Contenido()
	if err != nil || len(contenido) == 0 || len(contenido) > MaximoBytesFicheroGINPIX {
		return datosExportacionLocal{}, ErrExportacionLocalGINPIX
	}
	huellaTexto, err := preparacion.HuellaSHA256()
	if err != nil || !huellaHexadecimalValida(huellaTexto) {
		return datosExportacionLocal{}, ErrExportacionLocalGINPIX
	}
	huella := sha256.Sum256(contenido)
	huellaEsperada, err := hex.DecodeString(huellaTexto)
	if err != nil || subtle.ConstantTimeCompare(huella[:], huellaEsperada) != 1 {
		return datosExportacionLocal{}, ErrExportacionLocalGINPIX
	}
	nombreFinal := nombreFinalExportacion(metadatos)
	comprobante := nuevoComprobanteExportacionLocal(
		nombreFinal,
		huellaTexto,
		metadatos,
		int64(len(contenido)),
		false,
	)
	if comprobante.Validar() != nil {
		return datosExportacionLocal{}, ErrExportacionLocalGINPIX
	}
	return datosExportacionLocal{
		contenido:       contenido,
		huella:          huella,
		huellaTexto:     huellaTexto,
		metadatos:       metadatos,
		nombreFinal:     nombreFinal,
		comprobanteBase: comprobante,
	}, nil
}

func exportarEnRaiz(
	ctx context.Context,
	raiz *os.Root,
	datos datosExportacionLocal,
) (ComprobanteExportacionLocal, error) {
	temporal, nombreTemporal, err := crearTemporalExclusivo(raiz)
	if err != nil {
		return ComprobanteExportacionLocal{}, ErrExportacionLocalGINPIX
	}
	temporalPendiente := true
	defer func() {
		if temporalPendiente {
			_ = raiz.Remove(nombreTemporal)
			_ = sincronizarDirectorio(raiz)
		}
	}()
	defer temporal.Close()

	if !contextoExportacionActivo(ctx) ||
		temporal.Chmod(permisosFicheroExportado) != nil ||
		escribirContenido(ctx, temporal, datos.contenido) != nil ||
		temporal.Sync() != nil || !ficheroTemporalCompatible(temporal, int64(len(datos.contenido))) ||
		temporal.Close() != nil || !contextoExportacionActivo(ctx) {
		return ComprobanteExportacionLocal{}, ErrExportacionLocalGINPIX
	}

	// Link publica un nombre completo de forma atomica y, a diferencia de
	// Rename, falla con EEXIST sin sustituir ningun objeto previo.
	err = raiz.Link(nombreTemporal, datos.nombreFinal)
	if err != nil {
		if raiz.Remove(nombreTemporal) != nil {
			return ComprobanteExportacionLocal{}, ErrExportacionLocalGINPIX
		}
		temporalPendiente = false
		if sincronizarDirectorio(raiz) != nil || !errors.Is(err, fs.ErrExist) ||
			!contextoExportacionActivo(ctx) ||
			comprobarFinalExacto(raiz, datos.nombreFinal, datos.huella, int64(len(datos.contenido))) != nil {
			return ComprobanteExportacionLocal{}, ErrExportacionLocalGINPIX
		}
		return nuevoComprobanteExportacionLocal(
			datos.nombreFinal,
			datos.huellaTexto,
			datos.metadatos,
			int64(len(datos.contenido)),
			true,
		), nil
	}

	if !contextoExportacionActivo(ctx) {
		_ = raiz.Remove(datos.nombreFinal)
		return ComprobanteExportacionLocal{}, ErrExportacionLocalGINPIX
	}
	if raiz.Remove(nombreTemporal) != nil {
		_ = raiz.Remove(datos.nombreFinal)
		return ComprobanteExportacionLocal{}, ErrExportacionLocalGINPIX
	}
	temporalPendiente = false
	if comprobarFinalExacto(
		raiz,
		datos.nombreFinal,
		datos.huella,
		int64(len(datos.contenido)),
	) != nil || sincronizarDirectorio(raiz) != nil {
		_ = raiz.Remove(datos.nombreFinal)
		_ = sincronizarDirectorio(raiz)
		return ComprobanteExportacionLocal{}, ErrExportacionLocalGINPIX
	}
	return datos.comprobanteBase, nil
}

func crearTemporalExclusivo(raiz *os.Root) (*os.File, string, error) {
	for intento := 0; intento < maximosIntentosTemporal; intento++ {
		var aleatorio [16]byte
		if _, err := rand.Read(aleatorio[:]); err != nil {
			return nil, "", err
		}
		nombre := prefijoFicheroTemporal + hex.EncodeToString(aleatorio[:]) + sufijoFicheroTemporal
		fichero, err := raiz.OpenFile(
			nombre,
			os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW|syscall.O_CLOEXEC,
			permisosFicheroExportado,
		)
		if err == nil {
			return fichero, nombre, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, "", err
		}
	}
	return nil, "", ErrExportacionLocalGINPIX
}

func escribirContenido(ctx context.Context, fichero *os.File, contenido []byte) error {
	const tamanoBloque = 64 * 1024
	for len(contenido) > 0 {
		if !contextoExportacionActivo(ctx) {
			return ErrExportacionLocalGINPIX
		}
		limite := min(len(contenido), tamanoBloque)
		escritos, err := fichero.Write(contenido[:limite])
		if err != nil {
			return err
		}
		if escritos != limite {
			return io.ErrShortWrite
		}
		contenido = contenido[escritos:]
	}
	return nil
}

func ficheroTemporalCompatible(fichero *os.File, tamano int64) bool {
	informacion, err := fichero.Stat()
	return err == nil && informacion.Mode() == permisosFicheroExportado &&
		informacion.Size() == tamano
}

func comprobarFinalExacto(
	raiz *os.Root,
	nombre string,
	huellaEsperada [sha256.Size]byte,
	tamanoEsperado int64,
) error {
	informacionPrevia, err := raiz.Lstat(nombre)
	if err != nil || !informacionPrevia.Mode().IsRegular() ||
		informacionPrevia.Mode() != permisosFicheroExportado ||
		informacionPrevia.Size() != tamanoEsperado {
		return ErrExportacionLocalGINPIX
	}
	fichero, err := raiz.OpenFile(
		nombre,
		os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC,
		0,
	)
	if err != nil {
		return ErrExportacionLocalGINPIX
	}
	defer fichero.Close()
	informacion, err := fichero.Stat()
	if err != nil || !informacion.Mode().IsRegular() ||
		informacion.Mode() != permisosFicheroExportado ||
		informacion.Size() != tamanoEsperado {
		return ErrExportacionLocalGINPIX
	}
	calculador := sha256.New()
	leidos, err := io.Copy(calculador, io.LimitReader(fichero, int64(MaximoBytesFicheroGINPIX)+1))
	if err != nil || leidos != tamanoEsperado ||
		subtle.ConstantTimeCompare(calculador.Sum(nil), huellaEsperada[:]) != 1 {
		return ErrExportacionLocalGINPIX
	}
	posterior, err := fichero.Stat()
	if err != nil || posterior.Mode() != informacion.Mode() ||
		posterior.Size() != informacion.Size() {
		return ErrExportacionLocalGINPIX
	}
	return nil
}

func sincronizarDirectorio(raiz *os.Root) error {
	directorio, err := raiz.Open(".")
	if err != nil {
		return err
	}
	defer directorio.Close()
	informacion, err := directorio.Stat()
	if err != nil || !informacion.IsDir() {
		return ErrExportacionLocalGINPIX
	}
	return directorio.Sync()
}

func nombreFinalExportacion(metadatos MetadatosPreparacionExportacion) string {
	// La idempotencia, y no la huella del contenido, fija el nombre. Asi una
	// reutilizacion hostil de la misma referencia colisiona y falla cerrada.
	material := "vec.dipgra.contratacion-temporal.ginpix.exportacion-local.v1\x00" +
		metadatos.IdempotenciaRef
	suma := sha256.Sum256([]byte(material))
	return prefijoFicheroFinal + hex.EncodeToString(suma[:]) + sufijoFicheroFinal
}

func metadatosExportacionAcotados(m MetadatosPreparacionExportacion) bool {
	if !m.validar() {
		return false
	}
	referencias := [...]string{
		m.ExpedienteRef,
		m.IncorporacionRef,
		m.ProcedenciaModeloRef,
		m.CorrelacionRef,
		m.IdempotenciaRef,
		m.MapeoRef,
		m.ProcedenciaMapeoRef,
	}
	for _, referencia := range referencias {
		if len(referencia) == 0 || len(referencia) > maximoBytesReferencia {
			return false
		}
	}
	return true
}

func nuevoComprobanteExportacionLocal(
	ruta, huella string,
	metadatos MetadatosPreparacionExportacion,
	tamano int64,
	replay bool,
) ComprobanteExportacionLocal {
	return ComprobanteExportacionLocal{datos: &datosComprobanteExportacionLocal{
		rutaRelativa: ruta,
		huellaSHA256: huella,
		metadatos:    metadatos,
		tamanoBytes:  tamano,
		permisos:     permisosFicheroExportado,
		replay:       replay,
	}}
}

func (c ComprobanteExportacionLocal) Validar() error {
	if c.datos == nil || !rutaRelativaExportacionSegura(c.datos.rutaRelativa) ||
		!huellaHexadecimalValida(c.datos.huellaSHA256) ||
		!metadatosExportacionAcotados(c.datos.metadatos) ||
		c.datos.rutaRelativa != nombreFinalExportacion(c.datos.metadatos) ||
		c.datos.tamanoBytes <= 0 || c.datos.tamanoBytes > MaximoBytesFicheroGINPIX ||
		c.datos.permisos != permisosFicheroExportado {
		return ErrExportacionLocalGINPIX
	}
	return nil
}

func (c ComprobanteExportacionLocal) Alcance() (string, error) {
	if c.Validar() != nil {
		return "", ErrExportacionLocalGINPIX
	}
	return AlcanceComprobanteExportacionLocal, nil
}

func (c ComprobanteExportacionLocal) RutaRelativa() (string, error) {
	if c.Validar() != nil {
		return "", ErrExportacionLocalGINPIX
	}
	return c.datos.rutaRelativa, nil
}

func (c ComprobanteExportacionLocal) HuellaSHA256() (string, error) {
	if c.Validar() != nil {
		return "", ErrExportacionLocalGINPIX
	}
	return c.datos.huellaSHA256, nil
}

func (c ComprobanteExportacionLocal) Metadatos() (MetadatosPreparacionExportacion, error) {
	if c.Validar() != nil {
		return MetadatosPreparacionExportacion{}, ErrExportacionLocalGINPIX
	}
	return c.datos.metadatos, nil
}

func (c ComprobanteExportacionLocal) TamanoBytes() (int64, error) {
	if c.Validar() != nil {
		return 0, ErrExportacionLocalGINPIX
	}
	return c.datos.tamanoBytes, nil
}

func (c ComprobanteExportacionLocal) Permisos() (fs.FileMode, error) {
	if c.Validar() != nil {
		return 0, ErrExportacionLocalGINPIX
	}
	return c.datos.permisos, nil
}

func (c ComprobanteExportacionLocal) EsReplayLocal() (bool, error) {
	if c.Validar() != nil {
		return false, ErrExportacionLocalGINPIX
	}
	return c.datos.replay, nil
}

func rutaRelativaExportacionSegura(ruta string) bool {
	if len(ruta) != len(prefijoFicheroFinal)+sha256.Size*2+len(sufijoFicheroFinal) ||
		!strings.HasPrefix(ruta, prefijoFicheroFinal) ||
		!strings.HasSuffix(ruta, sufijoFicheroFinal) {
		return false
	}
	huella := ruta[len(prefijoFicheroFinal) : len(ruta)-len(sufijoFicheroFinal)]
	return huellaHexadecimalValida(huella)
}

func huellaHexadecimalValida(huella string) bool {
	if len(huella) != sha256.Size*2 {
		return false
	}
	for _, caracter := range huella {
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

func contextoExportacionActivo(ctx context.Context) bool {
	return ctx != nil && ctx.Err() == nil
}
