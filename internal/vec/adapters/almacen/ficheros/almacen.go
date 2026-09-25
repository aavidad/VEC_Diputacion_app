// Package ficheros implementa ports.AlmacenObjetos sobre un directorio local
// privado. Es el almacén de originales predeterminado porque no depende de
// ninguna otra aplicación; el conector S3 sigue disponible por configuración.
//
// Garantías y límites:
//   - El directorio lo fija la composición: absoluto, canónico, sin enlaces,
//     propiedad del proceso y modo 0700. Ninguna ruta procede del llamante:
//     cada objeto se nombra con un identificador opaco aleatorio.
//   - Ficheros 0600, escritura atómica (temporal O_EXCL, fsync, rename y
//     fsync del directorio) y huella SHA-256 verificada al escribir y al leer.
//   - Idempotencia por clave con huella de la solicitud completa.
//   - Retención, inmovilización y eliminación las hace cumplir este adaptador;
//     el sistema de ficheros no las impone frente a un administrador del
//     equipo. No cifra: el cifrado en reposo corresponde al volumen.
//   - Un cerrojo flock serializa procesos que compartan el directorio.
//   - Las rutas se forman con Join; no se usa openat/RESOLVE_BENEATH. Un
//     proceso con el mismo uid podría sustituir un subdirectorio.
package ficheros

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

// IdentificadorPredeterminado es el conector_id declarado en capacidades.
const IdentificadorPredeterminado = "ficheros-local"

const (
	tamanoMaximoAbsoluto = 64 << 20
	maximoMetadatos      = 64 << 10
	dirObjetos           = "objetos"
	dirIdempotencia      = "idempotencia"
	dirTemporal          = "tmp"
	ficheroCerrojo       = ".cerrojo"
	ficheroVolcados      = ".volcados"
	maximoVolcados       = 16
	volcadosPorDefecto   = 2
)

var (
	ErrConfiguracionInvalida = errors.New("vec: configuración del almacén de ficheros inválida")
	ErrDirectorioInseguro    = errors.New("vec: directorio del almacén de ficheros inseguro")
)

// Configuracion la construye la raíz de composición desde el material
// privado; nunca llega del cliente ni de un módulo.
type Configuracion struct {
	ConectorID   string
	Directorio   string
	TamanoMaximo int64
	// RetencionMinimaAdmitida fija la retención de cada objeto admitido al
	// escribirlo o promoverlo. Cero significa no fijarla: el objeto queda sin
	// retención hasta que una operación posterior la aplique (política de
	// conservación provisional). Nunca es negativa.
	RetencionMinimaAdmitida time.Duration
	// MaximoVolcadosConcurrentes limita temporales por instancia; cero usa 2.
	MaximoVolcadosConcurrentes int
}

type Almacen struct {
	mu           sync.Mutex
	conectorID   string
	raiz         string
	tamanoMaximo int64
	retencionMin time.Duration
	reloj        ports.Reloj
	cerrojo      *os.File
	volcados     chan struct{}
}

var _ ports.AlmacenObjetos = (*Almacen)(nil)

// Nuevo valida el directorio y prepara la estructura interna. Falla cerrado
// ante cualquier permiso, propietario o enlace inesperado.
func Nuevo(cfg Configuracion, reloj ports.Reloj) (*Almacen, error) {
	if reloj == nil || reloj.Ahora().IsZero() || cfg.TamanoMaximo < 1 || cfg.TamanoMaximo > tamanoMaximoAbsoluto ||
		cfg.RetencionMinimaAdmitida < 0 || cfg.RetencionMinimaAdmitida%time.Microsecond != 0 ||
		cfg.MaximoVolcadosConcurrentes < 0 || cfg.MaximoVolcadosConcurrentes > maximoVolcados {
		return nil, ErrConfiguracionInvalida
	}
	if cfg.ConectorID == "" {
		cfg.ConectorID = IdentificadorPredeterminado
	}
	if ports.VerificarCapacidadesAlmacen(capacidades(cfg.ConectorID, cfg.TamanoMaximo, cfg.RetencionMinimaAdmitida > 0), ports.RequisitosAlmacenObjetos{}) != nil {
		return nil, ErrConfiguracionInvalida
	}
	if err := directorioPrivado(cfg.Directorio); err != nil {
		return nil, err
	}
	for _, sub := range []string{dirObjetos, dirIdempotencia, dirTemporal} {
		ruta := filepath.Join(cfg.Directorio, sub)
		if err := os.Mkdir(ruta, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
			return nil, ErrDirectorioInseguro
		}
		if err := directorioPrivado(ruta); err != nil {
			return nil, err
		}
	}
	cerrojo, err := prepararCerrojoYLimpiar(cfg.Directorio)
	if err != nil {
		return nil, err
	}
	limite := cfg.MaximoVolcadosConcurrentes
	if limite == 0 {
		limite = volcadosPorDefecto
	}
	return &Almacen{conectorID: cfg.ConectorID, raiz: cfg.Directorio, tamanoMaximo: cfg.TamanoMaximo,
		retencionMin: cfg.RetencionMinimaAdmitida, reloj: reloj, cerrojo: cerrojo, volcados: make(chan struct{}, limite)}, nil
}

// RetencionAlEscribir indica si el almacén fija retención irreversible al
// escribir o promover un objeto admitido. La composición lo consulta para no
// fijarla mientras la política de conservación sea provisional.
func (a *Almacen) RetencionAlEscribir() bool { return a != nil && a.retencionMin > 0 }

// Cerrar libera el descriptor del cerrojo. El almacén no admite más uso.
func (a *Almacen) Cerrar() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cerrojo == nil {
		return nil
	}
	err := a.cerrojo.Close()
	a.cerrojo = nil
	return err
}

func directorioPrivado(ruta string) error {
	if ruta == "" || !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta {
		return ErrDirectorioInseguro
	}
	info, err := os.Lstat(ruta)
	if err != nil || !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return ErrDirectorioInseguro
	}
	if evaluada, err := filepath.EvalSymlinks(ruta); err != nil || evaluada != ruta {
		return ErrDirectorioInseguro
	}
	if st, ok := info.Sys().(*syscall.Stat_t); !ok || int(st.Uid) != os.Geteuid() {
		return ErrDirectorioInseguro
	}
	return nil
}

// capacidades declara retención (aplicable después con AplicarRetencion) y,
// solo si el almacén la fija al escribir, retención atómica en la promoción.
func capacidades(conectorID string, tamanoMaximo int64, retencionAlEscribir bool) ports.CapacidadesAlmacenObjetos {
	return ports.CapacidadesAlmacenObjetos{
		ConectorID: conectorID, EscrituraEnFlujo: true, LecturaEnFlujo: true, ReferenciasOpacas: true,
		IntegridadSHA256: true, Versionado: true, Retencion: true, BloqueoLegal: true,
		PromocionAtomica: true, RetencionAtomicaEnPromocion: retencionAlEscribir, PreservaObjetoOriginal: true,
		TamanoMaximoObjeto: tamanoMaximo,
	}
}

func (a *Almacen) Capacidades(ctx context.Context) (ports.CapacidadesAlmacenObjetos, error) {
	if ctx == nil {
		return ports.CapacidadesAlmacenObjetos{}, ports.ErrSolicitudAlmacenInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.CapacidadesAlmacenObjetos{}, err
	}
	if a == nil {
		return ports.CapacidadesAlmacenObjetos{}, ports.ErrCapacidadAlmacenNoDisponible
	}
	return capacidades(a.conectorID, a.tamanoMaximo, a.RetencionAlEscribir()), nil
}

// bloquear toma el mutex del proceso y el cerrojo exclusivo del directorio.
func (a *Almacen) bloquear() (func(), error) {
	if a == nil {
		return nil, ports.ErrCapacidadAlmacenNoDisponible
	}
	a.mu.Lock()
	if a.cerrojo == nil {
		a.mu.Unlock()
		return nil, ports.ErrCapacidadAlmacenNoDisponible
	}
	if err := syscall.Flock(int(a.cerrojo.Fd()), syscall.LOCK_EX); err != nil {
		a.mu.Unlock()
		return nil, ports.ErrCapacidadAlmacenNoDisponible
	}
	return func() {
		_ = syscall.Flock(int(a.cerrojo.Fd()), syscall.LOCK_UN)
		a.mu.Unlock()
	}, nil
}

type metadatosFichero struct {
	Referencia           string    `json:"referencia"`
	Version              string    `json:"version"`
	ConectorID           string    `json:"conector_id"`
	Zona                 string    `json:"zona"`
	MIME                 string    `json:"mime"`
	Tamano               int64     `json:"tamano"`
	HuellaSHA256         string    `json:"huella_sha256"`
	EvidenciaCreacionRef string    `json:"evidencia_creacion_ref"`
	AlmacenadoEn         time.Time `json:"almacenado_en"`
	RetenidoHasta        time.Time `json:"retenido_hasta"`
	Inmovilizado         bool      `json:"inmovilizado"`
	Eliminado            bool      `json:"eliminado"`
}

func (m metadatosFichero) objeto() ports.ObjetoAlmacenado {
	return ports.ObjetoAlmacenado{
		Objeto:     ports.ReferenciaObjetoAlmacen{Referencia: m.Referencia, Version: m.Version},
		ConectorID: m.ConectorID, Zona: ports.ZonaAlmacen(m.Zona), MIME: m.MIME, Tamano: m.Tamano,
		HuellaSHA256: m.HuellaSHA256, EvidenciaCreacionRef: m.EvidenciaCreacionRef,
		AlmacenadoEn: m.AlmacenadoEn.UTC(), RetenidoHasta: m.RetenidoHasta.UTC(),
		Inmovilizado: m.Inmovilizado, Eliminado: m.Eliminado,
	}
}

func desdeObjeto(o ports.ObjetoAlmacenado) metadatosFichero {
	return metadatosFichero{
		Referencia: o.Objeto.Referencia, Version: o.Objeto.Version, ConectorID: o.ConectorID,
		Zona: string(o.Zona), MIME: o.MIME, Tamano: o.Tamano, HuellaSHA256: o.HuellaSHA256,
		EvidenciaCreacionRef: o.EvidenciaCreacionRef, AlmacenadoEn: o.AlmacenadoEn.UTC(),
		RetenidoHasta: o.RetenidoHasta.UTC(), Inmovilizado: o.Inmovilizado, Eliminado: o.Eliminado,
	}
}

type registroIdempotencia struct {
	HuellaSolicitud string `json:"huella_solicitud"`
	Referencia      string `json:"referencia"`
	Version         string `json:"version"`
}

// nombreObjeto solo admite referencias generadas por este adaptador: nunca
// se compone una ruta con texto ajeno.
func nombreObjeto(ref ports.ReferenciaObjetoAlmacen) (string, bool) {
	if ref.Version != "1" || len(ref.Referencia) != len("obj_")+32 || !strings.HasPrefix(ref.Referencia, "obj_") {
		return "", false
	}
	for _, c := range ref.Referencia[len("obj_"):] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return "", false
		}
	}
	return ref.Referencia, true
}

func aleatorio(prefijo string) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return prefijo + hex.EncodeToString(b[:]), nil
}

func sha256Hex(b []byte) string {
	suma := sha256.Sum256(b)
	return hex.EncodeToString(suma[:])
}

func (a *Almacen) rutaObjeto(nombre string) string { return filepath.Join(a.raiz, dirObjetos, nombre) }
func (a *Almacen) rutaMetadatos(nombre string) string {
	return filepath.Join(a.raiz, dirObjetos, nombre+".json")
}
func (a *Almacen) rutaIdempotencia(clave string) string {
	return filepath.Join(a.raiz, dirIdempotencia, sha256Hex([]byte("vec.almacen.ficheros.idempotencia.v1\x00"+clave))+".json")
}

// escribirAtomico crea un temporal 0600 exclusivo, lo sincroniza y lo
// renombra sobre el destino; después sincroniza el directorio.
func (a *Almacen) escribirAtomico(destino string, contenido []byte) error {
	tmp, err := a.temporal()
	if err != nil {
		return err
	}
	nombre := tmp.Name()
	_, errW := tmp.Write(contenido)
	errS := tmp.Sync()
	errC := tmp.Close()
	if errW != nil || errS != nil || errC != nil {
		_ = os.Remove(nombre)
		return ports.ErrCapacidadAlmacenNoDisponible
	}
	if err := os.Rename(nombre, destino); err != nil {
		_ = os.Remove(nombre)
		return ports.ErrCapacidadAlmacenNoDisponible
	}
	return sincronizarDirectorio(filepath.Dir(destino))
}

func (a *Almacen) temporal() (*os.File, error) {
	nombre, err := aleatorio("tmp_")
	if err != nil {
		return nil, ports.ErrCapacidadAlmacenNoDisponible
	}
	f, err := os.OpenFile(filepath.Join(a.raiz, dirTemporal, nombre), os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, ports.ErrCapacidadAlmacenNoDisponible
	}
	return f, nil
}

func sincronizarDirectorio(ruta string) error {
	d, err := os.Open(ruta)
	if err != nil {
		return ports.ErrCapacidadAlmacenNoDisponible
	}
	errS := d.Sync()
	errC := d.Close()
	if errS != nil || errC != nil {
		return ports.ErrCapacidadAlmacenNoDisponible
	}
	return nil
}

// leerAcotado abre sin seguir enlaces un fichero regular 0600 y lo lee hasta
// maximo+1 bytes.
func leerAcotado(ruta string, maximo int64) ([]byte, error) {
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ports.ErrObjetoAlmacenNoEncontrado
	}
	if err != nil {
		return nil, ports.ErrCapacidadAlmacenNoDisponible
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || info.Size() > maximo {
		return nil, ports.ErrIntegridadObjetoAlmacen
	}
	contenido, err := io.ReadAll(io.LimitReader(f, maximo+1))
	if err != nil {
		return nil, ports.ErrCapacidadAlmacenNoDisponible
	}
	if int64(len(contenido)) > maximo {
		return nil, ports.ErrIntegridadObjetoAlmacen
	}
	return contenido, nil
}

func (a *Almacen) cargarMetadatos(ref ports.ReferenciaObjetoAlmacen) (metadatosFichero, error) {
	nombre, ok := nombreObjeto(ref)
	if !ok {
		return metadatosFichero{}, ports.ErrObjetoAlmacenNoEncontrado
	}
	raw, err := leerAcotado(a.rutaMetadatos(nombre), maximoMetadatos)
	if err != nil {
		return metadatosFichero{}, err
	}
	var m metadatosFichero
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&m) != nil || m.Referencia != ref.Referencia || m.Version != ref.Version ||
		m.ConectorID != a.conectorID || m.objeto().Validar() != nil {
		return metadatosFichero{}, ports.ErrIntegridadObjetoAlmacen
	}
	return m, nil
}

func (a *Almacen) guardarMetadatos(m metadatosFichero) error {
	if m.objeto().Validar() != nil {
		return ports.ErrIntegridadObjetoAlmacen
	}
	nombre, ok := nombreObjeto(ports.ReferenciaObjetoAlmacen{Referencia: m.Referencia, Version: m.Version})
	if !ok {
		return ports.ErrIntegridadObjetoAlmacen
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return ports.ErrIntegridadObjetoAlmacen
	}
	return a.escribirAtomico(a.rutaMetadatos(nombre), raw)
}

// contenidoVerificado lee el original y exige tamaño y huella registrados.
func (a *Almacen) contenidoVerificado(m metadatosFichero) ([]byte, error) {
	if m.Eliminado {
		return nil, ports.ErrObjetoAlmacenEliminado
	}
	contenido, err := leerAcotado(a.rutaObjeto(m.Referencia), m.Tamano)
	if errors.Is(err, ports.ErrObjetoAlmacenNoEncontrado) {
		return nil, ports.ErrIntegridadObjetoAlmacen
	}
	if err != nil {
		return nil, err
	}
	if int64(len(contenido)) != m.Tamano || sha256Hex(contenido) != m.HuellaSHA256 {
		return nil, ports.ErrIntegridadObjetoAlmacen
	}
	return contenido, nil
}

func (a *Almacen) cargarIdempotencia(clave string) (registroIdempotencia, bool, error) {
	raw, err := leerAcotado(a.rutaIdempotencia(clave), maximoMetadatos)
	if errors.Is(err, ports.ErrObjetoAlmacenNoEncontrado) {
		return registroIdempotencia{}, false, nil
	}
	if err != nil {
		return registroIdempotencia{}, false, err
	}
	var r registroIdempotencia
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&r) != nil || len(r.HuellaSolicitud) != 64 {
		return registroIdempotencia{}, false, ports.ErrIntegridadObjetoAlmacen
	}
	if _, ok := nombreObjeto(ports.ReferenciaObjetoAlmacen{Referencia: r.Referencia, Version: r.Version}); !ok {
		return registroIdempotencia{}, false, ports.ErrIntegridadObjetoAlmacen
	}
	return r, true, nil
}

func (a *Almacen) guardarIdempotencia(clave string, r registroIdempotencia) error {
	raw, err := json.Marshal(r)
	if err != nil {
		return ports.ErrIntegridadObjetoAlmacen
	}
	return a.escribirAtomico(a.rutaIdempotencia(clave), raw)
}

func contextoValido(ctx context.Context) error {
	if ctx == nil {
		return ports.ErrSolicitudAlmacenInvalida
	}
	return ctx.Err()
}

func (a *Almacen) Escribir(ctx context.Context, solicitud ports.SolicitudEscribirObjeto) (ports.ResultadoOperacionObjeto, error) {
	vacio := ports.ResultadoOperacionObjeto{}
	if err := contextoValido(ctx); err != nil {
		return vacio, err
	}
	if a == nil {
		return vacio, ports.ErrCapacidadAlmacenNoDisponible
	}
	if err := solicitud.Validar(); err != nil {
		return vacio, err
	}
	if solicitud.Tamano > a.tamanoMaximo {
		return vacio, ports.ErrLimiteObjetoAlmacenExcedido
	}
	// Denegar antes de leer el contenido y repetir después del volcado, ya
	// bajo el cerrojo exclusivo, porque la concesión puede vencer entretanto.
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenEscribir, a.reloj.Ahora().UTC()); err != nil {
		return vacio, err
	}
	select {
	case a.volcados <- struct{}{}:
	case <-ctx.Done():
		return vacio, ctx.Err()
	}
	if err := contextoValido(ctx); err != nil {
		<-a.volcados
		return vacio, err
	}
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenEscribir, a.reloj.Ahora().UTC()); err != nil {
		<-a.volcados
		return vacio, err
	}
	liberarVolcado, err := a.cerrojoVolcado()
	if err != nil {
		<-a.volcados
		return vacio, err
	}
	defer liberarVolcado()
	temporal, err := a.volcarContenido(ctx, solicitud.Contenido, solicitud.Tamano, solicitud.HuellaSHA256)
	<-a.volcados
	if err != nil {
		return vacio, err
	}
	defer func() { _ = os.Remove(temporal) }()
	desbloquear, err := a.bloquear()
	if err != nil {
		return vacio, err
	}
	defer desbloquear()
	ahora := a.reloj.Ahora().UTC()
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenEscribir, ahora); err != nil {
		return vacio, err
	}
	huellaSolicitud, err := huellaEscritura(solicitud)
	if err != nil {
		return vacio, err
	}
	caps := capacidades(a.conectorID, a.tamanoMaximo, a.RetencionAlEscribir())
	if previo, existe, err := a.cargarIdempotencia(solicitud.ClaveIdempotencia); err != nil {
		return vacio, err
	} else if existe {
		if previo.HuellaSolicitud != huellaSolicitud {
			return vacio, ports.ErrIdempotenciaAlmacenReutilizada
		}
		m, err := a.cargarMetadatos(ports.ReferenciaObjetoAlmacen{Referencia: previo.Referencia, Version: previo.Version})
		if err != nil {
			return vacio, ports.ErrIntegridadObjetoAlmacen
		}
		if m.Eliminado {
			return vacio, ports.ErrObjetoAlmacenEliminado
		}
		if m.Inmovilizado {
			return vacio, ports.ErrObjetoAlmacenInmovilizado
		}
		if _, err := a.contenidoVerificado(m); err != nil {
			return vacio, err
		}
		evidencia, err := a.evidencia(solicitud.Contexto, m.objeto().Objeto, "", true, ahora)
		if err != nil {
			return vacio, err
		}
		resultado := ports.ResultadoOperacionObjeto{Objeto: m.objeto(), Evidencia: evidencia}
		if resultado.ValidarEscritura(solicitud, caps) != nil {
			return vacio, ports.ErrIntegridadObjetoAlmacen
		}
		return resultado, nil
	}
	nombre, err := aleatorio("obj_")
	if err != nil {
		return vacio, ports.ErrCapacidadAlmacenNoDisponible
	}
	referencia := ports.ReferenciaObjetoAlmacen{Referencia: nombre, Version: "1"}
	evidencia, err := a.evidencia(solicitud.Contexto, referencia, "", false, ahora)
	if err != nil {
		return vacio, err
	}
	objeto := ports.ObjetoAlmacenado{
		Objeto: referencia, ConectorID: a.conectorID, Zona: solicitud.Zona, MIME: solicitud.MIME,
		Tamano: solicitud.Tamano, HuellaSHA256: solicitud.HuellaSHA256,
		EvidenciaCreacionRef: evidencia.Referencia, AlmacenadoEn: ahora,
	}
	if solicitud.Zona == ports.ZonaAlmacenAdmitida && a.RetencionAlEscribir() {
		objeto.RetenidoHasta = ahora.Add(a.retencionMin)
	}
	resultado := ports.ResultadoOperacionObjeto{Objeto: objeto, Evidencia: evidencia}
	if resultado.ValidarEscritura(solicitud, caps) != nil {
		return vacio, ports.ErrIntegridadObjetoAlmacen
	}
	if err := a.materializar(temporal, objeto); err != nil {
		return vacio, err
	}
	if err := a.guardarIdempotencia(solicitud.ClaveIdempotencia, registroIdempotencia{
		HuellaSolicitud: huellaSolicitud, Referencia: referencia.Referencia, Version: referencia.Version,
	}); err != nil {
		// El objeto queda huérfano, sin índice: un reintento crea otro.
		return vacio, err
	}
	return resultado, nil
}

// materializar renombra el temporal ya verificado y registra sus metadatos.
// Si los metadatos no se guardan, retira el contenido.
func (a *Almacen) materializar(temporal string, objeto ports.ObjetoAlmacenado) error {
	destino := a.rutaObjeto(objeto.Objeto.Referencia)
	if _, err := os.Lstat(destino); !errors.Is(err, fs.ErrNotExist) {
		return ports.ErrIntegridadObjetoAlmacen
	}
	if err := os.Rename(temporal, destino); err != nil {
		return ports.ErrCapacidadAlmacenNoDisponible
	}
	if err := sincronizarDirectorio(filepath.Dir(destino)); err != nil {
		_ = os.Remove(destino)
		return err
	}
	if err := a.guardarMetadatos(desdeObjeto(objeto)); err != nil {
		_ = os.Remove(destino)
		return err
	}
	return nil
}

func (a *Almacen) Abrir(ctx context.Context, solicitud ports.SolicitudAbrirObjeto) (ports.LecturaObjetoAlmacen, error) {
	vacio := ports.LecturaObjetoAlmacen{}
	if err := contextoValido(ctx); err != nil {
		return vacio, err
	}
	if a == nil {
		return vacio, ports.ErrCapacidadAlmacenNoDisponible
	}
	if err := solicitud.Validar(); err != nil {
		return vacio, err
	}
	desbloquear, err := a.bloquear()
	if err != nil {
		return vacio, err
	}
	defer desbloquear()
	ahora := a.reloj.Ahora().UTC()
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenLeer, ahora); err != nil {
		return vacio, err
	}
	m, err := a.cargarMetadatos(solicitud.Objeto)
	if err != nil {
		return vacio, err
	}
	switch {
	case m.Eliminado:
		return vacio, ports.ErrObjetoAlmacenEliminado
	case m.Inmovilizado:
		return vacio, ports.ErrObjetoAlmacenInmovilizado
	case ports.ZonaAlmacen(m.Zona) != solicitud.Zona:
		return vacio, ports.ErrTransicionZonaAlmacenNoPermitida
	case m.Tamano > solicitud.Limite:
		return vacio, ports.ErrLimiteObjetoAlmacenExcedido
	}
	contenido, err := a.contenidoVerificado(m)
	if err != nil {
		return vacio, err
	}
	evidencia, err := a.evidencia(solicitud.Contexto, m.objeto().Objeto, "", false, ahora)
	if err != nil {
		return vacio, err
	}
	lectura := ports.LecturaObjetoAlmacen{Objeto: m.objeto(), Evidencia: evidencia, Contenido: io.NopCloser(bytes.NewReader(contenido))}
	if lectura.ValidarContra(solicitud) != nil {
		return vacio, ports.ErrIntegridadObjetoAlmacen
	}
	return lectura, nil
}

func (a *Almacen) Promover(ctx context.Context, solicitud ports.SolicitudPromoverObjeto) (ports.ResultadoOperacionObjeto, error) {
	vacio := ports.ResultadoOperacionObjeto{}
	if err := contextoValido(ctx); err != nil {
		return vacio, err
	}
	if a == nil {
		return vacio, ports.ErrCapacidadAlmacenNoDisponible
	}
	if err := solicitud.Validar(); err != nil {
		return vacio, err
	}
	desbloquear, err := a.bloquear()
	if err != nil {
		return vacio, err
	}
	defer desbloquear()
	ahora := a.reloj.Ahora().UTC()
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenPromover, ahora); err != nil {
		return vacio, err
	}
	origen, err := a.cargarMetadatos(solicitud.Origen)
	if err != nil {
		return vacio, err
	}
	switch {
	case origen.Eliminado:
		return vacio, ports.ErrObjetoAlmacenEliminado
	case origen.Inmovilizado:
		return vacio, ports.ErrObjetoAlmacenInmovilizado
	case ports.ZonaAlmacen(origen.Zona) != ports.ZonaAlmacenCuarentena:
		return vacio, ports.ErrTransicionZonaAlmacenNoPermitida
	}
	contenido, err := a.contenidoVerificado(origen)
	if err != nil {
		return vacio, err
	}
	caps := capacidades(a.conectorID, a.tamanoMaximo, a.RetencionAlEscribir())
	huellaSolicitud, err := huellaPromocion(solicitud, origen.objeto())
	if err != nil {
		return vacio, err
	}
	if previo, existe, err := a.cargarIdempotencia(solicitud.ClaveIdempotencia); err != nil {
		return vacio, err
	} else if existe {
		if previo.HuellaSolicitud != huellaSolicitud {
			return vacio, ports.ErrIdempotenciaAlmacenReutilizada
		}
		m, err := a.cargarMetadatos(ports.ReferenciaObjetoAlmacen{Referencia: previo.Referencia, Version: previo.Version})
		if err != nil {
			return vacio, ports.ErrIntegridadObjetoAlmacen
		}
		if m.Eliminado {
			return vacio, ports.ErrObjetoAlmacenEliminado
		}
		if m.Inmovilizado {
			return vacio, ports.ErrObjetoAlmacenInmovilizado
		}
		if _, err := a.contenidoVerificado(m); err != nil {
			return vacio, err
		}
		evidencia, err := a.evidencia(solicitud.Contexto, m.objeto().Objeto, solicitud.EvidenciaAnalisisRef, true, ahora)
		if err != nil {
			return vacio, err
		}
		resultado := ports.ResultadoOperacionObjeto{Objeto: m.objeto(), Evidencia: evidencia}
		if resultado.ValidarPromocion(solicitud, origen.objeto(), caps) != nil {
			return vacio, ports.ErrIntegridadObjetoAlmacen
		}
		return resultado, nil
	}
	if ahora.Before(origen.AlmacenadoEn) {
		return vacio, ports.ErrSolicitudAlmacenInvalida
	}
	nombre, err := aleatorio("obj_")
	if err != nil {
		return vacio, ports.ErrCapacidadAlmacenNoDisponible
	}
	referencia := ports.ReferenciaObjetoAlmacen{Referencia: nombre, Version: "1"}
	evidencia, err := a.evidencia(solicitud.Contexto, referencia, solicitud.EvidenciaAnalisisRef, false, ahora)
	if err != nil {
		return vacio, err
	}
	destino := origen.objeto()
	destino.Objeto = referencia
	destino.Zona = ports.ZonaAlmacenAdmitida
	destino.EvidenciaCreacionRef = evidencia.Referencia
	destino.AlmacenadoEn = ahora
	destino.RetenidoHasta = time.Time{}
	if a.RetencionAlEscribir() {
		destino.RetenidoHasta = ahora.Add(a.retencionMin)
	}
	destino.Inmovilizado = false
	resultado := ports.ResultadoOperacionObjeto{Objeto: destino, Evidencia: evidencia}
	if resultado.ValidarPromocion(solicitud, origen.objeto(), caps) != nil {
		return vacio, ports.ErrIntegridadObjetoAlmacen
	}
	tmp, err := a.temporal()
	if err != nil {
		return vacio, err
	}
	temporal := tmp.Name()
	defer func() { _ = os.Remove(temporal) }()
	_, errW := tmp.Write(contenido)
	errS := tmp.Sync()
	errC := tmp.Close()
	if errW != nil || errS != nil || errC != nil {
		return vacio, ports.ErrCapacidadAlmacenNoDisponible
	}
	if err := a.materializar(temporal, destino); err != nil {
		return vacio, err
	}
	if err := a.guardarIdempotencia(solicitud.ClaveIdempotencia, registroIdempotencia{
		HuellaSolicitud: huellaSolicitud, Referencia: referencia.Referencia, Version: referencia.Version,
	}); err != nil {
		return vacio, err
	}
	return resultado, nil
}

// mutar carga, valida y reescribe los metadatos de un objeto vivo.
func (a *Almacen) mutar(ctx context.Context, accion string, contexto ports.ContextoOperacionAlmacen,
	ref ports.ReferenciaObjetoAlmacen, cambiar func(ahora time.Time, m *metadatosFichero) error,
	validar func(ports.ResultadoOperacionObjeto, ports.ObjetoAlmacenado) error, fundamento string,
) (ports.ResultadoOperacionObjeto, error) {
	vacio := ports.ResultadoOperacionObjeto{}
	desbloquear, err := a.bloquear()
	if err != nil {
		return vacio, err
	}
	defer desbloquear()
	ahora := a.reloj.Ahora().UTC()
	if err := contexto.ValidarParaEn(accion, ahora); err != nil {
		return vacio, err
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	m, err := a.cargarMetadatos(ref)
	if err != nil {
		return vacio, err
	}
	if m.Eliminado {
		return vacio, ports.ErrObjetoAlmacenEliminado
	}
	if _, err := a.contenidoVerificado(m); err != nil {
		return vacio, err
	}
	anterior := m.objeto()
	if err := cambiar(ahora, &m); err != nil {
		return vacio, err
	}
	evidencia, err := a.evidencia(contexto, m.objeto().Objeto, fundamento, false, ahora)
	if err != nil {
		return vacio, err
	}
	resultado := ports.ResultadoOperacionObjeto{Objeto: m.objeto(), Evidencia: evidencia}
	if validar(resultado, anterior) != nil {
		return vacio, ports.ErrIntegridadObjetoAlmacen
	}
	if err := a.guardarMetadatos(m); err != nil {
		return vacio, err
	}
	return resultado, nil
}

func (a *Almacen) AplicarRetencion(ctx context.Context, solicitud ports.SolicitudRetenerObjeto) (ports.ResultadoOperacionObjeto, error) {
	if err := contextoValido(ctx); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if a == nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrCapacidadAlmacenNoDisponible
	}
	if err := solicitud.Validar(); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	return a.mutar(ctx, ports.AccionAlmacenAplicarRetencion, solicitud.Contexto, solicitud.Objeto,
		func(ahora time.Time, m *metadatosFichero) error {
			if err := solicitud.ValidarEn(ahora); err != nil {
				return err
			}
			if m.Inmovilizado {
				return ports.ErrObjetoAlmacenInmovilizado
			}
			if !m.RetenidoHasta.IsZero() && solicitud.Hasta.Before(m.RetenidoHasta) {
				return ports.ErrRetencionObjetoAlmacenVigente
			}
			m.RetenidoHasta = solicitud.Hasta.UTC()
			return nil
		},
		func(r ports.ResultadoOperacionObjeto, anterior ports.ObjetoAlmacenado) error {
			return r.ValidarRetencion(solicitud, anterior)
		}, solicitud.PoliticaRef)
}

func (a *Almacen) Inmovilizar(ctx context.Context, solicitud ports.SolicitudInmovilizarObjeto) (ports.ResultadoOperacionObjeto, error) {
	if err := contextoValido(ctx); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if a == nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrCapacidadAlmacenNoDisponible
	}
	if err := solicitud.Validar(); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	return a.mutar(ctx, ports.AccionAlmacenInmovilizar, solicitud.Contexto, solicitud.Objeto,
		func(_ time.Time, m *metadatosFichero) error {
			if m.Inmovilizado {
				return ports.ErrObjetoAlmacenInmovilizado
			}
			m.Inmovilizado = true
			return nil
		},
		func(r ports.ResultadoOperacionObjeto, anterior ports.ObjetoAlmacenado) error {
			return r.ValidarInmovilizacion(solicitud, anterior)
		}, solicitud.AprobacionRef)
}

func (a *Almacen) LevantarInmovilizacion(ctx context.Context, solicitud ports.SolicitudLevantarInmovilizacionObjeto) (ports.ResultadoOperacionObjeto, error) {
	if err := contextoValido(ctx); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if a == nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrCapacidadAlmacenNoDisponible
	}
	if err := solicitud.Validar(); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	return a.mutar(ctx, ports.AccionAlmacenLevantarInmovilizacion, solicitud.Contexto, solicitud.Objeto,
		func(_ time.Time, m *metadatosFichero) error {
			if !m.Inmovilizado {
				return ports.ErrSolicitudAlmacenInvalida
			}
			m.Inmovilizado = false
			return nil
		},
		func(r ports.ResultadoOperacionObjeto, anterior ports.ObjetoAlmacenado) error {
			return r.ValidarLevantamientoInmovilizacion(solicitud, anterior)
		}, solicitud.AprobacionRef)
}

func (a *Almacen) Eliminar(ctx context.Context, solicitud ports.SolicitudEliminarObjeto) (ports.EvidenciaOperacionAlmacen, error) {
	vacio := ports.EvidenciaOperacionAlmacen{}
	if err := contextoValido(ctx); err != nil {
		return vacio, err
	}
	if a == nil {
		return vacio, ports.ErrCapacidadAlmacenNoDisponible
	}
	if err := solicitud.Validar(); err != nil {
		return vacio, err
	}
	desbloquear, err := a.bloquear()
	if err != nil {
		return vacio, err
	}
	defer desbloquear()
	ahora := a.reloj.Ahora().UTC()
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenEliminar, ahora); err != nil {
		return vacio, err
	}
	m, err := a.cargarMetadatos(solicitud.Objeto)
	if err != nil {
		return vacio, err
	}
	switch {
	case m.Eliminado:
		return vacio, ports.ErrObjetoAlmacenEliminado
	case m.Inmovilizado:
		return vacio, ports.ErrObjetoAlmacenInmovilizado
	case !m.RetenidoHasta.IsZero() && ahora.Before(m.RetenidoHasta):
		return vacio, ports.ErrRetencionObjetoAlmacenVigente
	}
	if _, err := a.contenidoVerificado(m); err != nil {
		return vacio, err
	}
	evidencia, err := a.evidencia(solicitud.Contexto, m.objeto().Objeto, solicitud.AprobacionRef, false, ahora)
	if err != nil {
		return vacio, err
	}
	if evidencia.ValidarEliminacion(solicitud, m.objeto()) != nil {
		return vacio, ports.ErrIntegridadObjetoAlmacen
	}
	// Primero se marca en metadatos (punto de confirmación) y después se
	// retira el contenido; un fallo intermedio deja el objeto ilegible.
	m.Eliminado = true
	if err := a.guardarMetadatos(m); err != nil {
		return vacio, err
	}
	if err := os.Remove(a.rutaObjeto(m.Referencia)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return vacio, ports.ErrCapacidadAlmacenNoDisponible
	}
	if err := sincronizarDirectorio(filepath.Join(a.raiz, dirObjetos)); err != nil {
		return vacio, err
	}
	return evidencia, nil
}

// evidencia construye el recibo técnico ligado al contexto exacto.
func (a *Almacen) evidencia(contexto ports.ContextoOperacionAlmacen, objeto ports.ReferenciaObjetoAlmacen,
	fundamento string, reintento bool, realizadaEn time.Time,
) (ports.EvidenciaOperacionAlmacen, error) {
	p, err := contexto.Proyeccion()
	if err != nil {
		return ports.EvidenciaOperacionAlmacen{}, ports.ErrSolicitudAlmacenInvalida
	}
	ref, err := aleatorio("evidencia_ficheros_")
	if err != nil {
		return ports.EvidenciaOperacionAlmacen{}, ports.ErrCapacidadAlmacenNoDisponible
	}
	return ports.EvidenciaOperacionAlmacen{
		Referencia: ref, ConectorID: a.conectorID, EsquemaContexto: p.Esquema,
		AccionNegocio: p.AccionNegocio, Accion: p.AccionTecnica, EfectoRef: p.EfectoRef,
		HuellaPlanEfectoSHA256: p.HuellaPlanEfectoSHA256, HuellaManifiestoSHA256: p.HuellaManifiestoSHA256,
		HuellaPasoSHA256: p.HuellaPasoSHA256, PasoRef: p.PasoRef, HuellaDecisionSHA256: p.HuellaDecisionSHA256,
		Objeto: objeto, OperacionRef: p.OperacionRef, CorrelacionRef: p.CorrelacionRef,
		AutorizacionRef: p.AutorizacionRef, Finalidad: p.Finalidad, Clasificacion: p.Clasificacion,
		RealizadaEn: realizadaEn.UTC(), CargaRef: p.CargaRef, SujetoSeudonimoHMAC: p.SujetoSeudonimoHMAC,
		RecursoRef: p.RecursoRef, ModuloID: p.ModuloID, HuellaSolicitudHMAC: p.HuellaSolicitudHMAC,
		FundamentoRef: fundamento, ReintentoIdempotente: reintento,
	}, nil
}

// componentesContexto proyecta el contexto autorizado en los componentes de
// la huella de idempotencia. Si la proyección falla no hay huella posible:
// el error se propaga como ErrSolicitudAlmacenInvalida con la causa envuelta,
// en lugar de calcular una huella degenerada sin contexto.
func componentesContexto(contexto ports.ContextoOperacionAlmacen) ([]string, error) {
	p, err := contexto.Proyeccion()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ports.ErrSolicitudAlmacenInvalida, err)
	}
	return []string{p.Esquema, p.OperacionRef, p.CorrelacionRef, p.AutorizacionRef, p.Finalidad,
		p.Clasificacion, p.AccionNegocio, p.AccionTecnica, p.CargaRef, p.SujetoSeudonimoHMAC,
		p.RecursoRef, p.ModuloID, p.HuellaSolicitudHMAC, p.EfectoRef, p.HuellaPlanEfectoSHA256,
		string(p.PasoRef), p.HuellaDecisionSHA256}, nil
}

func huellaEscritura(s ports.SolicitudEscribirObjeto) (string, error) {
	contexto, err := componentesContexto(s.Contexto)
	if err != nil {
		return "", err
	}
	c := append([]string{"escritura-v1", string(s.Zona), s.MIME, strconv.FormatInt(s.Tamano, 10), s.HuellaSHA256},
		contexto...)
	return sha256Hex([]byte(strings.Join(c, "\x00"))), nil
}

func huellaPromocion(s ports.SolicitudPromoverObjeto, origen ports.ObjetoAlmacenado) (string, error) {
	contexto, err := componentesContexto(s.Contexto)
	if err != nil {
		return "", err
	}
	c := append([]string{"promocion-v1", s.Origen.Referencia, s.Origen.Version, s.EvidenciaAnalisisRef, origen.HuellaSHA256},
		contexto...)
	return sha256Hex([]byte(strings.Join(c, "\x00"))), nil
}

// String evita que la ruta privada llegue a un registro por accidente.
func (a *Almacen) String() string { return fmt.Sprintf("almacen-ficheros[%s]", a.conectorIDSeguro()) }

func (a *Almacen) conectorIDSeguro() string {
	if a == nil {
		return ""
	}
	return a.conectorID
}
