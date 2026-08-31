package ports

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"reflect"
	"strings"
	"time"
)

var (
	ErrSolicitudPoliticaConservacionDocumentalInvalida = errors.New(
		"vec: solicitud de politica de conservacion documental invalida",
	)
	ErrPoliticaConservacionDocumentalInvalida = errors.New(
		"vec: politica de conservacion documental invalida",
	)
	// ErrPoliticaConservacionDocumentalNoResuelta oculta si la causa fue
	// ausencia, ambiguedad, retirada, caducidad o fallo de la dependencia.
	ErrPoliticaConservacionDocumentalNoResuelta = errors.New(
		"vec: politica de conservacion documental no resuelta",
	)
)

// SolicitudPoliticaConservacionDocumental inmoviliza la publicacion exacta
// esperada. Sus referencias son opacas y no admiten texto humano ni selectores.
type SolicitudPoliticaConservacionDocumental struct {
	procedimientoRef   string
	serieDocumentalRef string
	tipoDocumentalRef  string
	expedienteRef      string
	politicaRef        string
	versionPolitica    uint64
	huellaPolitica     [sha256.Size]byte
	baseJuridicaRef    string
	vigenteDesde       time.Time
	vigenteHasta       time.Time
}

func NuevaSolicitudPoliticaConservacionDocumental(
	procedimientoRef, serieDocumentalRef, tipoDocumentalRef, expedienteRef string,
	politicaRef string,
	versionPolitica uint64,
	huellaPoliticaSHA256 []byte,
	baseJuridicaRef string,
	vigenteDesde, vigenteHasta time.Time,
) (SolicitudPoliticaConservacionDocumental, error) {
	solicitud := SolicitudPoliticaConservacionDocumental{
		procedimientoRef: procedimientoRef, serieDocumentalRef: serieDocumentalRef,
		tipoDocumentalRef: tipoDocumentalRef, expedienteRef: expedienteRef,
		politicaRef: politicaRef, versionPolitica: versionPolitica,
		baseJuridicaRef: baseJuridicaRef, vigenteDesde: vigenteDesde, vigenteHasta: vigenteHasta,
	}
	if len(huellaPoliticaSHA256) == sha256.Size {
		copy(solicitud.huellaPolitica[:], huellaPoliticaSHA256)
	}
	if solicitud.Validar() != nil {
		return SolicitudPoliticaConservacionDocumental{},
			ErrSolicitudPoliticaConservacionDocumentalInvalida
	}
	return solicitud, nil
}

func (s SolicitudPoliticaConservacionDocumental) Validar() error {
	referencias := []string{
		s.procedimientoRef, s.serieDocumentalRef, s.tipoDocumentalRef,
		s.expedienteRef, s.politicaRef, s.baseJuridicaRef,
	}
	if !referenciasPoliticaConservacionDocumentalValidas(referencias...) ||
		s.versionPolitica == 0 || huellaPoliticaConservacionDocumentalNula(s.huellaPolitica) ||
		!instantePoliticaConservacionDocumentalValido(s.vigenteDesde) ||
		!instantePoliticaConservacionDocumentalValido(s.vigenteHasta) ||
		!s.vigenteDesde.Before(s.vigenteHasta) {
		return ErrSolicitudPoliticaConservacionDocumentalInvalida
	}
	return nil
}

func (s SolicitudPoliticaConservacionDocumental) ProcedimientoRef() string {
	return s.procedimientoRef
}
func (s SolicitudPoliticaConservacionDocumental) SerieDocumentalRef() string {
	return s.serieDocumentalRef
}
func (s SolicitudPoliticaConservacionDocumental) TipoDocumentalRef() string {
	return s.tipoDocumentalRef
}
func (s SolicitudPoliticaConservacionDocumental) ExpedienteRef() string { return s.expedienteRef }
func (s SolicitudPoliticaConservacionDocumental) PoliticaRef() string   { return s.politicaRef }
func (s SolicitudPoliticaConservacionDocumental) VersionPolitica() uint64 {
	return s.versionPolitica
}
func (s SolicitudPoliticaConservacionDocumental) HuellaPoliticaSHA256() []byte {
	return append([]byte(nil), s.huellaPolitica[:]...)
}
func (s SolicitudPoliticaConservacionDocumental) BaseJuridicaRef() string {
	return s.baseJuridicaRef
}
func (s SolicitudPoliticaConservacionDocumental) VigenteDesde() time.Time {
	return s.vigenteDesde
}
func (s SolicitudPoliticaConservacionDocumental) VigenteHasta() time.Time {
	return s.vigenteHasta
}
func (SolicitudPoliticaConservacionDocumental) String() string {
	return "[SOLICITUD-POLITICA-CONSERVACION-DOCUMENTAL-OPACA]"
}
func (SolicitudPoliticaConservacionDocumental) GoString() string {
	return "ports.SolicitudPoliticaConservacionDocumental{[OPACA]}"
}

type EstadoPoliticaConservacionDocumental string

const (
	EstadoPoliticaConservacionDocumentalAprobada EstadoPoliticaConservacionDocumental = "aprobada"
	EstadoPoliticaConservacionDocumentalRetirada EstadoPoliticaConservacionDocumental = "retirada"
)

type ProteccionPoliticaConservacionDocumental string

const (
	ProteccionPoliticaConservacionDocumentalOrdinaria ProteccionPoliticaConservacionDocumental = "conservacion"
	ProteccionPoliticaConservacionDocumentalBloqueada ProteccionPoliticaConservacionDocumental = "bloqueo"
)

// PoliticaConservacionDocumental es una declaracion de la autoridad
// documental. ConservacionHasta nunca equivale a permiso de borrado o expurgo.
type PoliticaConservacionDocumental struct {
	solicitud         SolicitudPoliticaConservacionDocumental
	conservacionHasta time.Time
	proteccion        ProteccionPoliticaConservacionDocumental
	bloqueoRef        string
	estado            EstadoPoliticaConservacionDocumental
	retiradaEn        time.Time
}

func NuevaPoliticaConservacionDocumental(
	solicitud SolicitudPoliticaConservacionDocumental,
	conservacionHasta time.Time,
	proteccion ProteccionPoliticaConservacionDocumental,
	bloqueoRef string,
	estado EstadoPoliticaConservacionDocumental,
	retiradaEn time.Time,
) (PoliticaConservacionDocumental, error) {
	politica := PoliticaConservacionDocumental{
		solicitud: solicitud, conservacionHasta: conservacionHasta,
		proteccion: proteccion, bloqueoRef: bloqueoRef, estado: estado, retiradaEn: retiradaEn,
	}
	if politica.Validar() != nil {
		return PoliticaConservacionDocumental{}, ErrPoliticaConservacionDocumentalInvalida
	}
	return politica, nil
}

func (p PoliticaConservacionDocumental) Validar() error {
	if p.solicitud.Validar() != nil ||
		!instantePoliticaConservacionDocumentalValido(p.conservacionHasta) {
		return ErrPoliticaConservacionDocumentalInvalida
	}
	switch p.proteccion {
	case ProteccionPoliticaConservacionDocumentalOrdinaria:
		if p.bloqueoRef != "" {
			return ErrPoliticaConservacionDocumentalInvalida
		}
	case ProteccionPoliticaConservacionDocumentalBloqueada:
		if !referenciaPoliticaConservacionDocumentalValida(p.bloqueoRef) ||
			p.solicitud.contieneReferencia(p.bloqueoRef) {
			return ErrPoliticaConservacionDocumentalInvalida
		}
	default:
		return ErrPoliticaConservacionDocumentalInvalida
	}
	switch p.estado {
	case EstadoPoliticaConservacionDocumentalAprobada:
		if !p.retiradaEn.IsZero() {
			return ErrPoliticaConservacionDocumentalInvalida
		}
	case EstadoPoliticaConservacionDocumentalRetirada:
		if !instantePoliticaConservacionDocumentalValido(p.retiradaEn) ||
			p.retiradaEn.Before(p.solicitud.vigenteDesde) ||
			!p.retiradaEn.Before(p.solicitud.vigenteHasta) {
			return ErrPoliticaConservacionDocumentalInvalida
		}
	default:
		return ErrPoliticaConservacionDocumentalInvalida
	}
	return nil
}

func (p PoliticaConservacionDocumental) Solicitud() SolicitudPoliticaConservacionDocumental {
	return p.solicitud
}
func (p PoliticaConservacionDocumental) ConservacionHasta() time.Time {
	return p.conservacionHasta
}
func (p PoliticaConservacionDocumental) Proteccion() ProteccionPoliticaConservacionDocumental {
	return p.proteccion
}
func (p PoliticaConservacionDocumental) BloqueoRef() string {
	return p.bloqueoRef
}
func (p PoliticaConservacionDocumental) Estado() EstadoPoliticaConservacionDocumental {
	return p.estado
}
func (p PoliticaConservacionDocumental) RetiradaEn() time.Time { return p.retiradaEn }
func (PoliticaConservacionDocumental) String() string {
	return "[POLITICA-CONSERVACION-DOCUMENTAL-OPACA]"
}
func (PoliticaConservacionDocumental) GoString() string {
	return "ports.PoliticaConservacionDocumental{[OPACA]}"
}

type ResultadoPoliticaConservacionDocumental struct {
	politica   PoliticaConservacionDocumental
	resueltaEn time.Time
}

func (r ResultadoPoliticaConservacionDocumental) Validar() error {
	if !r.politica.aplicableEn(r.politica.solicitud, r.resueltaEn) {
		return ErrPoliticaConservacionDocumentalNoResuelta
	}
	return nil
}

func (r ResultadoPoliticaConservacionDocumental) Politica() PoliticaConservacionDocumental {
	return r.politica
}
func (r ResultadoPoliticaConservacionDocumental) ResueltaEn() time.Time { return r.resueltaEn }
func (ResultadoPoliticaConservacionDocumental) String() string {
	return "[RESULTADO-POLITICA-CONSERVACION-DOCUMENTAL-OPACO]"
}
func (ResultadoPoliticaConservacionDocumental) GoString() string {
	return "ports.ResultadoPoliticaConservacionDocumental{[OPACO]}"
}

// ResolutorPoliticaConservacionDocumental devuelve todas las coincidencias
// exactas. El coordinador exige cardinalidad uno y nunca elige la primera.
type ResolutorPoliticaConservacionDocumental interface {
	BuscarPoliticasConservacionDocumental(
		context.Context,
		SolicitudPoliticaConservacionDocumental,
	) ([]PoliticaConservacionDocumental, error)
}

func ResolverPoliticaConservacionDocumental(
	ctx context.Context,
	resolutor ResolutorPoliticaConservacionDocumental,
	reloj Reloj,
	solicitud SolicitudPoliticaConservacionDocumental,
) (ResultadoPoliticaConservacionDocumental, error) {
	denegar := func() (ResultadoPoliticaConservacionDocumental, error) {
		return ResultadoPoliticaConservacionDocumental{},
			ErrPoliticaConservacionDocumentalNoResuelta
	}
	if ctx == nil || ctx.Err() != nil || solicitud.Validar() != nil ||
		dependenciaPoliticaConservacionDocumentalNula(resolutor) ||
		dependenciaPoliticaConservacionDocumentalNula(reloj) {
		return denegar()
	}
	politicas, err := resolutor.BuscarPoliticasConservacionDocumental(ctx, solicitud)
	if err != nil || ctx.Err() != nil || len(politicas) != 1 {
		return denegar()
	}
	resueltaEn := reloj.Ahora()
	politica := politicas[0]
	if ctx.Err() != nil || !politica.aplicableEn(solicitud, resueltaEn) {
		return denegar()
	}
	resultado := ResultadoPoliticaConservacionDocumental{
		politica: politica, resueltaEn: resueltaEn,
	}
	if resultado.Validar() != nil {
		return denegar()
	}
	return resultado, nil
}

func (p PoliticaConservacionDocumental) aplicableEn(
	solicitud SolicitudPoliticaConservacionDocumental,
	instante time.Time,
) bool {
	return p.Validar() == nil && solicitud.Validar() == nil && p.solicitud.coincide(solicitud) &&
		p.estado == EstadoPoliticaConservacionDocumentalAprobada &&
		instantePoliticaConservacionDocumentalValido(instante) &&
		!instante.Before(p.solicitud.vigenteDesde) && instante.Before(p.solicitud.vigenteHasta)
}

func (s SolicitudPoliticaConservacionDocumental) coincide(
	otra SolicitudPoliticaConservacionDocumental,
) bool {
	return s.Validar() == nil && otra.Validar() == nil &&
		s.procedimientoRef == otra.procedimientoRef &&
		s.serieDocumentalRef == otra.serieDocumentalRef &&
		s.tipoDocumentalRef == otra.tipoDocumentalRef && s.expedienteRef == otra.expedienteRef &&
		s.politicaRef == otra.politicaRef && s.versionPolitica == otra.versionPolitica &&
		subtle.ConstantTimeCompare(s.huellaPolitica[:], otra.huellaPolitica[:]) == 1 &&
		s.baseJuridicaRef == otra.baseJuridicaRef &&
		s.vigenteDesde.Equal(otra.vigenteDesde) && s.vigenteHasta.Equal(otra.vigenteHasta)
}

func (s SolicitudPoliticaConservacionDocumental) contieneReferencia(referencia string) bool {
	return referencia == s.procedimientoRef || referencia == s.serieDocumentalRef ||
		referencia == s.tipoDocumentalRef || referencia == s.expedienteRef ||
		referencia == s.politicaRef || referencia == s.baseJuridicaRef
}

func dependenciaPoliticaConservacionDocumentalNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

func referenciasPoliticaConservacionDocumentalValidas(referencias ...string) bool {
	vistas := make(map[string]struct{}, len(referencias))
	for _, referencia := range referencias {
		if !referenciaPoliticaConservacionDocumentalValida(referencia) {
			return false
		}
		if _, repetida := vistas[referencia]; repetida {
			return false
		}
		vistas[referencia] = struct{}{}
	}
	return true
}

func referenciaPoliticaConservacionDocumentalValida(referencia string) bool {
	if len(referencia) != len("ref:")+64 || !strings.HasPrefix(referencia, "ref:") {
		return false
	}
	noNula := false
	for _, caracter := range referencia[len("ref:"):] {
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
		noNula = noNula || caracter != '0'
	}
	return noNula
}

func huellaPoliticaConservacionDocumentalNula(huella [sha256.Size]byte) bool {
	var cero [sha256.Size]byte
	return subtle.ConstantTimeCompare(huella[:], cero[:]) == 1
}

func instantePoliticaConservacionDocumentalValido(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 && instante.Nanosecond()%1_000 == 0
}
