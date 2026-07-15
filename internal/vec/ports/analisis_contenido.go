package ports

import (
	"context"
	"errors"
	"io"
	"mime"
	"reflect"
	"strings"
	"time"
)

var (
	ErrCapacidadAnalisisContenidoNoDisponible = errors.New("vec: capacidad de analisis de contenido no disponible")
	ErrSolicitudAnalisisContenidoInvalida     = errors.New("vec: solicitud de analisis de contenido invalida")
	ErrResultadoAnalisisContenidoInvalido     = errors.New("vec: resultado de analisis de contenido invalido")
	ErrAnalizadorContenidoNoDisponible        = errors.New("vec: analizador de contenido no disponible")
)

const (
	maximoDeteccionesAnalisisContenido = 256
	duracionMaximaAnalisisContenido    = time.Hour
)

// CapacidadesAnalizadorContenido permite conectar ICAP, una API corporativa,
// un proceso aislado o cualquier motor futuro sin introducir el producto en
// el nucleo. MCP no es el transporte de seguridad: el analisis debe ser una
// integracion autenticada, determinista y observable entre servicios.
type CapacidadesAnalizadorContenido struct {
	ConectorID             string
	VersionConector        int
	AnalisisEnFlujo        bool
	CanalAutenticado       bool
	CifradoEnTransito      bool
	IdentidadMutua         bool
	ActualizacionFirmas    bool
	DetectaMalware         bool
	DetectaContenidoActivo bool
	TamanoMaximo           int64
}

type RequisitosAnalizadorContenido struct {
	AnalisisEnFlujo        bool
	CanalAutenticado       bool
	CifradoEnTransito      bool
	IdentidadMutua         bool
	ActualizacionFirmas    bool
	DetectaMalware         bool
	DetectaContenidoActivo bool
	TamanoMinimo           int64
}

func VerificarCapacidadesAnalizadorContenido(
	capacidades CapacidadesAnalizadorContenido,
	requisitos RequisitosAnalizadorContenido,
) error {
	if !referenciaOpacaAlmacenValida(capacidades.ConectorID, 128) || capacidades.VersionConector < 1 ||
		capacidades.TamanoMaximo < 1 || requisitos.TamanoMinimo < 1 ||
		capacidades.TamanoMaximo < requisitos.TamanoMinimo ||
		(requisitos.AnalisisEnFlujo && !capacidades.AnalisisEnFlujo) ||
		(requisitos.CanalAutenticado && !capacidades.CanalAutenticado) ||
		(requisitos.CifradoEnTransito && !capacidades.CifradoEnTransito) ||
		(requisitos.IdentidadMutua && !capacidades.IdentidadMutua) ||
		(requisitos.ActualizacionFirmas && !capacidades.ActualizacionFirmas) ||
		(requisitos.DetectaMalware && !capacidades.DetectaMalware) ||
		(requisitos.DetectaContenidoActivo && !capacidades.DetectaContenidoActivo) {
		return ErrCapacidadAnalisisContenidoNoDisponible
	}
	return nil
}

// SolicitudAnalizarContenido siempre apunta a la version exacta de un objeto
// en cuarentena. El caso de uso abre el objeto desde AlmacenObjetos y entrega
// un lector limitado; el navegador nunca invoca directamente al motor.
type SolicitudAnalizarContenido struct {
	Contexto          ContextoOperacionAlmacen
	Objeto            ReferenciaObjetoAlmacen
	ConectorAlmacenID string
	Zona              ZonaAlmacen
	MIME              string
	Tamano            int64
	HuellaSHA256      string
	Contenido         io.Reader
}

func (s SolicitudAnalizarContenido) Validar() error {
	if err := s.Contexto.validarParaPaso(AccionAlmacenAnalizarContenido); err != nil {
		return err
	}
	if err := s.Objeto.Validar(); err != nil {
		return ErrSolicitudAnalisisContenidoInvalida
	}
	if !referenciaOpacaAlmacenValida(s.ConectorAlmacenID, 128) || s.Zona != ZonaAlmacenCuarentena ||
		!textoSeguroAlmacen(s.MIME, 255) || s.Tamano < 1 || !esSHA256Hexadecimal(s.HuellaSHA256) ||
		lectorContenidoNulo(s.Contenido) {
		return ErrSolicitudAnalisisContenidoInvalida
	}
	return nil
}

func lectorContenidoNulo(contenido io.Reader) bool {
	if contenido == nil {
		return true
	}
	valor := reflect.ValueOf(contenido)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

type EstadoAnalisisContenido string

const (
	EstadoAnalisisContenidoLimpio        EstadoAnalisisContenido = "limpio"
	EstadoAnalisisContenidoMalicioso     EstadoAnalisisContenido = "malicioso"
	EstadoAnalisisContenidoSospechoso    EstadoAnalisisContenido = "sospechoso"
	EstadoAnalisisContenidoNoConcluyente EstadoAnalisisContenido = "no_concluyente"
	EstadoAnalisisContenidoError         EstadoAnalisisContenido = "error"
)

func (e EstadoAnalisisContenido) Valido() bool {
	switch e {
	case EstadoAnalisisContenidoLimpio, EstadoAnalisisContenidoMalicioso,
		EstadoAnalisisContenidoSospechoso, EstadoAnalisisContenidoNoConcluyente,
		EstadoAnalisisContenidoError:
		return true
	default:
		return false
	}
}

type ClaseDeteccionContenido string

const (
	ClaseDeteccionMalware         ClaseDeteccionContenido = "malware"
	ClaseDeteccionContenidoActivo ClaseDeteccionContenido = "contenido_activo"
	ClaseDeteccionArchivoDanado   ClaseDeteccionContenido = "archivo_danado"
	ClaseDeteccionPolitica        ClaseDeteccionContenido = "politica_seguridad"
)

func (c ClaseDeteccionContenido) Valida() bool {
	return c == ClaseDeteccionMalware || c == ClaseDeteccionContenidoActivo ||
		c == ClaseDeteccionArchivoDanado || c == ClaseDeteccionPolitica
}

// DeteccionContenido conserva codigos normalizados, nunca la salida cruda del
// motor, contenido, nombres originales ni datos personales.
type DeteccionContenido struct {
	Clase    ClaseDeteccionContenido
	Codigo   string
	FirmaRef string
}

func (d DeteccionContenido) Validar() error {
	if !d.Clase.Valida() || !textoSeguroAlmacen(d.Codigo, 128) ||
		(d.FirmaRef != "" && !referenciaOpacaAlmacenValida(d.FirmaRef, 256)) {
		return ErrResultadoAnalisisContenidoInvalido
	}
	return nil
}

// ResultadoAnalisisContenido es evidencia tecnica normalizada y acotada. Un
// error o resultado no concluyente nunca equivale a limpio.
type ResultadoAnalisisContenido struct {
	Objeto                ReferenciaObjetoAlmacen
	ConectorAlmacenID     string
	HuellaObjetoSHA256    string
	TamanoObjeto          int64
	MIMEDeclarado         string
	MIMEDetectado         string
	ConectorAnalizadorID  string
	VersionConector       int
	MotorRef              string
	VersionMotor          string
	FirmasRef             string
	Estado                EstadoAnalisisContenido
	CodigoResultado       string
	Detecciones           []DeteccionContenido
	BytesAnalizados       int64
	EvidenciaRef          string
	HuellaEvidenciaSHA256 string
	AnalisisIniciadoEn    time.Time
	AnalisisCompletadoEn  time.Time
	CorrelacionRef        string
	AutorizacionRef       string
	Finalidad             string
	Clasificacion         string
}

func (r ResultadoAnalisisContenido) Validar() error {
	if err := r.Objeto.Validar(); err != nil {
		return ErrResultadoAnalisisContenidoInvalido
	}
	if !referenciaOpacaAlmacenValida(r.ConectorAlmacenID, 128) ||
		!esSHA256Hexadecimal(r.HuellaObjetoSHA256) || r.TamanoObjeto < 1 ||
		!mimeAnalisisCanonico(r.MIMEDeclarado) || !mimeAnalisisCanonico(r.MIMEDetectado) ||
		!referenciaOpacaAlmacenValida(r.ConectorAnalizadorID, 128) || r.VersionConector < 1 ||
		!r.Estado.Valido() || !textoSeguroAlmacen(r.CodigoResultado, 128) ||
		len(r.Detecciones) > maximoDeteccionesAnalisisContenido || r.BytesAnalizados < 0 ||
		!referenciaOpacaAlmacenValida(r.EvidenciaRef, 512) || !esSHA256Hexadecimal(r.HuellaEvidenciaSHA256) ||
		r.AnalisisIniciadoEn.IsZero() || r.AnalisisCompletadoEn.IsZero() ||
		r.AnalisisCompletadoEn.Before(r.AnalisisIniciadoEn) ||
		r.AnalisisCompletadoEn.Sub(r.AnalisisIniciadoEn) > duracionMaximaAnalisisContenido ||
		!textoSeguroAlmacen(r.CorrelacionRef, 512) || !textoSeguroAlmacen(r.AutorizacionRef, 512) ||
		!textoSeguroAlmacen(r.Finalidad, 1024) || !textoSeguroAlmacen(r.Clasificacion, 256) {
		return ErrResultadoAnalisisContenidoInvalido
	}
	concluyente := r.Estado == EstadoAnalisisContenidoLimpio ||
		r.Estado == EstadoAnalisisContenidoMalicioso || r.Estado == EstadoAnalisisContenidoSospechoso
	if concluyente {
		if !referenciaOpacaAlmacenValida(r.MotorRef, 256) || !textoSeguroAlmacen(r.VersionMotor, 128) ||
			!referenciaOpacaAlmacenValida(r.FirmasRef, 256) || r.BytesAnalizados != r.TamanoObjeto {
			return ErrResultadoAnalisisContenidoInvalido
		}
	}
	if r.Estado == EstadoAnalisisContenidoLimpio && len(r.Detecciones) != 0 {
		return ErrResultadoAnalisisContenidoInvalido
	}
	if (r.Estado == EstadoAnalisisContenidoMalicioso || r.Estado == EstadoAnalisisContenidoSospechoso) &&
		len(r.Detecciones) == 0 {
		return ErrResultadoAnalisisContenidoInvalido
	}
	vistas := make(map[string]struct{}, len(r.Detecciones))
	for _, deteccion := range r.Detecciones {
		if err := deteccion.Validar(); err != nil {
			return err
		}
		clave := string(deteccion.Clase) + "\x00" + strings.TrimSpace(deteccion.Codigo) + "\x00" +
			strings.TrimSpace(deteccion.FirmaRef)
		if _, repetida := vistas[clave]; repetida {
			return ErrResultadoAnalisisContenidoInvalido
		}
		vistas[clave] = struct{}{}
	}
	return nil
}

func (r ResultadoAnalisisContenido) ValidarContra(s SolicitudAnalizarContenido) error {
	proyeccion, err := s.Contexto.Proyeccion()
	if err != nil || s.Validar() != nil || r.Validar() != nil || r.Objeto != s.Objeto ||
		r.ConectorAlmacenID != s.ConectorAlmacenID || r.HuellaObjetoSHA256 != s.HuellaSHA256 ||
		r.TamanoObjeto != s.Tamano || r.MIMEDeclarado != s.MIME || r.CorrelacionRef != proyeccion.CorrelacionRef ||
		r.AutorizacionRef != proyeccion.AutorizacionRef || r.Finalidad != proyeccion.Finalidad {
		return ErrResultadoAnalisisContenidoInvalido
	}
	if r.Clasificacion != proyeccion.Clasificacion {
		return ErrResultadoAnalisisContenidoInvalido
	}
	// En ausencia de una politica versionada de equivalencias, solo una
	// deteccion exacta puede terminar como limpia y llegar a promocion.
	if r.Estado == EstadoAnalisisContenidoLimpio && r.MIMEDetectado != s.MIME {
		return ErrResultadoAnalisisContenidoInvalido
	}
	return nil
}

func mimeAnalisisCanonico(valor string) bool {
	if !textoSeguroAlmacen(valor, 255) || valor != strings.ToLower(valor) {
		return false
	}
	tipo, parametros, err := mime.ParseMediaType(valor)
	return err == nil && tipo == valor && len(parametros) == 0 && strings.Contains(tipo, "/")
}

type AnalizadorContenido interface {
	Capacidades(context.Context) (CapacidadesAnalizadorContenido, error)
	Analizar(context.Context, SolicitudAnalizarContenido) (ResultadoAnalisisContenido, error)
}
