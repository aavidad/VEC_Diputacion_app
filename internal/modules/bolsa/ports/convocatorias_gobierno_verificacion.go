package ports

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

const (
	VigenciaMaximaComprobacionDependenciasConvocatoria = 15 * time.Minute
	VigenciaMaximaAtestacionVerificacionConvocatoria   = 5 * time.Minute
)

var (
	ErrComprobacionDependenciasConvocatoriaInvalida = errors.New("bolsa: comprobacion de dependencias de convocatoria invalida")
	ErrAprobacionConvocatoriaInvalida               = errors.New("bolsa: aprobacion de convocatoria invalida")
	ErrDependenciaConvocatoriaNoDisponible          = errors.New("bolsa: dependencia exacta de convocatoria no disponible")
	ErrAprobacionConvocatoriaNoDisponible           = errors.New("bolsa: aprobacion de convocatoria no disponible")
	ErrSerializacionVerificacionConvocatoria        = errors.New("bolsa: serializacion de verificacion de convocatoria prohibida")
)

type SolicitudVerificarDependenciasConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	Version     dominiobolsa.VersionConvocatoriaGobernada
	VerificarEn time.Time
}

func (s SolicitudVerificarDependenciasConvocatoria) Validar() error {
	if s.Version.Validar() != nil ||
		s.Version.EstadoGobierno != dominiobolsa.EstadoGobiernoConvocatoriaBorrador ||
		!instanteGobiernoConvocatoriaCanonico(s.VerificarEn) ||
		s.VerificarEn.Before(ultimaEdicionConvocatoria(s.Version)) {
		return ErrComprobacionDependenciasConvocatoriaInvalida
	}
	return nil
}

func (s SolicitudVerificarDependenciasConvocatoria) Clonar() (
	SolicitudVerificarDependenciasConvocatoria,
	error,
) {
	version, err := s.Version.ClonarCanonico()
	if err != nil || s.Validar() != nil {
		return SolicitudVerificarDependenciasConvocatoria{}, ErrComprobacionDependenciasConvocatoriaInvalida
	}
	s.Version = version
	return s, nil
}

func (s SolicitudVerificarDependenciasConvocatoria) validarEvidencia(
	evidencia dominiobolsa.EvidenciaDependenciasConvocatoria,
) error {
	huella, err := s.Version.HuellaContenidoSHA256()
	huellaEstado, errEstado := s.Version.HuellaSHA256()
	if s.Validar() != nil || err != nil || errEstado != nil ||
		!referenciaGobiernoConvocatoriaValida(evidencia.Referencia) ||
		!huellaGobiernoConvocatoriaValida(evidencia.HuellaEvidenciaSHA256) ||
		evidencia.ConvocatoriaRef != s.Version.Referencia() ||
		evidencia.Revision != s.Version.Revision || evidencia.HuellaContenidoSHA256 != huella ||
		evidencia.HuellaEstadoSHA256 != huellaEstado ||
		!instanteGobiernoConvocatoriaCanonico(evidencia.VerificadaEn) ||
		evidencia.VerificadaEn.Before(ultimaEdicionConvocatoria(s.Version)) ||
		evidencia.VerificadaEn.After(s.VerificarEn) ||
		s.VerificarEn.Sub(evidencia.VerificadaEn) > VigenciaMaximaComprobacionDependenciasConvocatoria {
		return ErrComprobacionDependenciasConvocatoriaInvalida
	}
	return nil
}

type DatosAtestacionDependenciasConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	Evidencia                 dominiobolsa.EvidenciaDependenciasConvocatoria
	RevisionVersion           int
	HuellaEstadoVersionSHA256 string
	VerificadorRef            string
	AtestacionRef             string
	HuellaAtestacionSHA256    string
	TokenConsumoRef           string
	AtestacionEmitidaEn       time.Time
	AtestacionValidaHasta     time.Time
}

// AtestacionDependenciasConvocatoria es un testimonio reconstruible para que
// la barrera durable relea y verifique su procedencia. No es una capacidad ni
// concede autoridad por si sola.
type AtestacionDependenciasConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	datos *DatosAtestacionDependenciasConvocatoria
}

func NuevaAtestacionDependenciasConvocatoria(
	solicitud SolicitudVerificarDependenciasConvocatoria,
	datos DatosAtestacionDependenciasConvocatoria,
) (AtestacionDependenciasConvocatoria, error) {
	huellaEstado, errHuella := solicitud.Version.HuellaSHA256()
	if validarDatosAtestacionVerificacion(
		datos.VerificadorRef, datos.AtestacionRef, datos.HuellaAtestacionSHA256,
		datos.TokenConsumoRef, datos.AtestacionEmitidaEn, datos.AtestacionValidaHasta,
	) != nil || errHuella != nil || datos.RevisionVersion != solicitud.Version.Revision ||
		datos.HuellaEstadoVersionSHA256 != huellaEstado ||
		solicitud.validarEvidencia(datos.Evidencia) != nil ||
		datos.AtestacionEmitidaEn.Before(datos.Evidencia.VerificadaEn) ||
		datos.AtestacionEmitidaEn.After(solicitud.VerificarEn) {
		return AtestacionDependenciasConvocatoria{}, ErrComprobacionDependenciasConvocatoriaInvalida
	}
	copia := datos
	return AtestacionDependenciasConvocatoria{datos: &copia}, nil
}

func (DatosAtestacionDependenciasConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionVerificacionConvocatoria
}

func (DatosAtestacionDependenciasConvocatoria) String() string {
	return "[DATOS-ATESTACION-DEPENDENCIAS-CONVOCATORIA-INTERNOS]"
}

func (d DatosAtestacionDependenciasConvocatoria) GoString() string { return d.String() }
func (d DatosAtestacionDependenciasConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}
func (d DatosAtestacionDependenciasConvocatoria) LogValue() slog.Value {
	return slog.StringValue(d.String())
}

func (c AtestacionDependenciasConvocatoria) DatosParaConsumo() (
	DatosAtestacionDependenciasConvocatoria,
	error,
) {
	if c.datos == nil || validarDatosAtestacionVerificacion(
		c.datos.VerificadorRef, c.datos.AtestacionRef, c.datos.HuellaAtestacionSHA256,
		c.datos.TokenConsumoRef, c.datos.AtestacionEmitidaEn, c.datos.AtestacionValidaHasta,
	) != nil || c.datos.RevisionVersion < 1 ||
		!huellaGobiernoConvocatoriaValida(c.datos.HuellaEstadoVersionSHA256) {
		return DatosAtestacionDependenciasConvocatoria{}, ErrComprobacionDependenciasConvocatoriaInvalida
	}
	return *c.datos, nil
}

func (c AtestacionDependenciasConvocatoria) Evidencia() (
	dominiobolsa.EvidenciaDependenciasConvocatoria,
	error,
) {
	datos, err := c.DatosParaConsumo()
	return datos.Evidencia, err
}

func (c AtestacionDependenciasConvocatoria) ValidarPara(
	solicitud SolicitudVerificarDependenciasConvocatoria,
	instante time.Time,
) error {
	datos, err := c.DatosParaConsumo()
	huellaEstado, errHuella := solicitud.Version.HuellaSHA256()
	if err != nil || solicitud.validarEvidencia(datos.Evidencia) != nil ||
		errHuella != nil || datos.RevisionVersion != solicitud.Version.Revision ||
		datos.HuellaEstadoVersionSHA256 != huellaEstado ||
		!instanteGobiernoConvocatoriaCanonico(instante) || instante.Before(datos.AtestacionEmitidaEn) ||
		!instante.Before(datos.AtestacionValidaHasta) {
		return ErrComprobacionDependenciasConvocatoriaInvalida
	}
	return nil
}

func (AtestacionDependenciasConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionVerificacionConvocatoria
}

func (AtestacionDependenciasConvocatoria) String() string {
	return "[ATESTACION-DEPENDENCIAS-CONVOCATORIA-INTERNA]"
}

func (c AtestacionDependenciasConvocatoria) GoString() string { return c.String() }
func (c AtestacionDependenciasConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}
func (c AtestacionDependenciasConvocatoria) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

// El repositorio debe releer la atestacion y consumir TokenConsumoRef dentro
// de la misma transaccion que la publicacion.
type VerificadorDependenciasConvocatoria interface {
	VerificarDependencias(
		context.Context,
		SolicitudVerificarDependenciasConvocatoria,
	) (AtestacionDependenciasConvocatoria, error)
}

type AccionAprobacionConvocatoria string

const (
	AccionAprobacionPublicarConvocatoria AccionAprobacionConvocatoria = "publicar"
	AccionAprobacionRetirarConvocatoria  AccionAprobacionConvocatoria = "retirar"
)

func (a AccionAprobacionConvocatoria) Valida() bool {
	return a == AccionAprobacionPublicarConvocatoria || a == AccionAprobacionRetirarConvocatoria
}

type SolicitudComprobarAprobacionConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	Version       dominiobolsa.VersionConvocatoriaGobernada
	Accion        AccionAprobacionConvocatoria
	AprobacionRef string
	ComprobarEn   time.Time
}

func (s SolicitudComprobarAprobacionConvocatoria) Validar() error {
	estadoValido := s.Accion == AccionAprobacionPublicarConvocatoria &&
		s.Version.EstadoGobierno == dominiobolsa.EstadoGobiernoConvocatoriaBorrador
	estadoValido = estadoValido || s.Accion == AccionAprobacionRetirarConvocatoria &&
		s.Version.EstadoGobierno == dominiobolsa.EstadoGobiernoConvocatoriaPublicada
	if s.Version.Validar() != nil || !s.Accion.Valida() || !estadoValido ||
		!referenciaGobiernoConvocatoriaValida(s.AprobacionRef) ||
		!instanteGobiernoConvocatoriaCanonico(s.ComprobarEn) ||
		s.ComprobarEn.Before(noAntesAprobacionConvocatoria(s.Version, s.Accion)) {
		return ErrAprobacionConvocatoriaInvalida
	}
	return nil
}

func (s SolicitudComprobarAprobacionConvocatoria) validarEvidencia(
	evidencia dominiobolsa.EvidenciaAprobacionConvocatoria,
) error {
	huella, err := s.Version.HuellaContenidoSHA256()
	huellaEstado, errEstado := s.Version.HuellaSHA256()
	noAntes := noAntesAprobacionConvocatoria(s.Version, s.Accion)
	actorSeparado := evidencia.AprobadaPor != s.Version.CreadaPor &&
		evidencia.AprobadaPor != s.Version.UltimaModificacionPor
	if s.Accion == AccionAprobacionRetirarConvocatoria {
		actorSeparado = evidencia.AprobadaPor != s.Version.PublicadaPor
	}
	if s.Validar() != nil || err != nil || errEstado != nil || evidencia.Accion != string(s.Accion) ||
		evidencia.Referencia != s.AprobacionRef ||
		!huellaGobiernoConvocatoriaValida(evidencia.HuellaEvidenciaSHA256) ||
		evidencia.ConvocatoriaRef != s.Version.Referencia() ||
		evidencia.Revision != s.Version.Revision || evidencia.HuellaContenidoSHA256 != huella ||
		evidencia.HuellaEstadoSHA256 != huellaEstado ||
		!referenciaGobiernoConvocatoriaValida(evidencia.AprobadaPor) || !actorSeparado ||
		!instanteGobiernoConvocatoriaCanonico(evidencia.AprobadaEn) ||
		evidencia.AprobadaEn.Before(noAntes) || evidencia.AprobadaEn.After(s.ComprobarEn) {
		return ErrAprobacionConvocatoriaInvalida
	}
	return nil
}

type DatosAtestacionAprobacionConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	Evidencia                 dominiobolsa.EvidenciaAprobacionConvocatoria
	RevisionVersion           int
	HuellaEstadoVersionSHA256 string
	VerificadorRef            string
	AtestacionRef             string
	HuellaAtestacionSHA256    string
	TokenConsumoRef           string
	AtestacionEmitidaEn       time.Time
	AtestacionValidaHasta     time.Time
}

// AtestacionAprobacionConvocatoria es reconstruible y no autoritativa. El
// repositorio debe verificarla y consumir su token dentro de la transaccion.
type AtestacionAprobacionConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	datos *DatosAtestacionAprobacionConvocatoria
}

func NuevaAtestacionAprobacionConvocatoria(
	solicitud SolicitudComprobarAprobacionConvocatoria,
	datos DatosAtestacionAprobacionConvocatoria,
) (AtestacionAprobacionConvocatoria, error) {
	huellaEstado, errHuella := solicitud.Version.HuellaSHA256()
	if validarDatosAtestacionVerificacion(
		datos.VerificadorRef, datos.AtestacionRef, datos.HuellaAtestacionSHA256,
		datos.TokenConsumoRef, datos.AtestacionEmitidaEn, datos.AtestacionValidaHasta,
	) != nil || errHuella != nil || datos.RevisionVersion != solicitud.Version.Revision ||
		datos.HuellaEstadoVersionSHA256 != huellaEstado ||
		solicitud.validarEvidencia(datos.Evidencia) != nil ||
		datos.AtestacionEmitidaEn.Before(datos.Evidencia.AprobadaEn) ||
		datos.AtestacionEmitidaEn.After(solicitud.ComprobarEn) {
		return AtestacionAprobacionConvocatoria{}, ErrAprobacionConvocatoriaInvalida
	}
	copia := datos
	return AtestacionAprobacionConvocatoria{datos: &copia}, nil
}

func (DatosAtestacionAprobacionConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionVerificacionConvocatoria
}

func (DatosAtestacionAprobacionConvocatoria) String() string {
	return "[DATOS-ATESTACION-APROBACION-CONVOCATORIA-INTERNOS]"
}

func (d DatosAtestacionAprobacionConvocatoria) GoString() string { return d.String() }
func (d DatosAtestacionAprobacionConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}
func (d DatosAtestacionAprobacionConvocatoria) LogValue() slog.Value {
	return slog.StringValue(d.String())
}

func (c AtestacionAprobacionConvocatoria) DatosParaConsumo() (
	DatosAtestacionAprobacionConvocatoria,
	error,
) {
	if c.datos == nil || validarDatosAtestacionVerificacion(
		c.datos.VerificadorRef, c.datos.AtestacionRef, c.datos.HuellaAtestacionSHA256,
		c.datos.TokenConsumoRef, c.datos.AtestacionEmitidaEn, c.datos.AtestacionValidaHasta,
	) != nil || c.datos.RevisionVersion < 1 ||
		!huellaGobiernoConvocatoriaValida(c.datos.HuellaEstadoVersionSHA256) {
		return DatosAtestacionAprobacionConvocatoria{}, ErrAprobacionConvocatoriaInvalida
	}
	return *c.datos, nil
}

func (c AtestacionAprobacionConvocatoria) Evidencia() (
	dominiobolsa.EvidenciaAprobacionConvocatoria,
	error,
) {
	datos, err := c.DatosParaConsumo()
	return datos.Evidencia, err
}

func (c AtestacionAprobacionConvocatoria) ValidarPara(
	solicitud SolicitudComprobarAprobacionConvocatoria,
	instante time.Time,
) error {
	datos, err := c.DatosParaConsumo()
	huellaEstado, errHuella := solicitud.Version.HuellaSHA256()
	if err != nil || solicitud.validarEvidencia(datos.Evidencia) != nil ||
		errHuella != nil || datos.RevisionVersion != solicitud.Version.Revision ||
		datos.HuellaEstadoVersionSHA256 != huellaEstado ||
		!instanteGobiernoConvocatoriaCanonico(instante) || instante.Before(datos.AtestacionEmitidaEn) ||
		!instante.Before(datos.AtestacionValidaHasta) {
		return ErrAprobacionConvocatoriaInvalida
	}
	return nil
}

func (AtestacionAprobacionConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionVerificacionConvocatoria
}

func (AtestacionAprobacionConvocatoria) String() string {
	return "[ATESTACION-APROBACION-CONVOCATORIA-INTERNA]"
}

func (c AtestacionAprobacionConvocatoria) GoString() string { return c.String() }
func (c AtestacionAprobacionConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}
func (c AtestacionAprobacionConvocatoria) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

// El repositorio debe releer la atestacion y consumir TokenConsumoRef dentro
// de la misma transaccion que el cambio de gobierno.
type VerificadorAprobacionConvocatoria interface {
	ComprobarAprobacion(
		context.Context,
		SolicitudComprobarAprobacionConvocatoria,
	) (AtestacionAprobacionConvocatoria, error)
}

func validarDatosAtestacionVerificacion(
	verificadorRef, atestacionRef, huellaAtestacion, tokenConsumoRef string,
	emitidaEn, validaHasta time.Time,
) error {
	if !referenciaGobiernoConvocatoriaValida(verificadorRef) ||
		!referenciaGobiernoConvocatoriaValida(atestacionRef) ||
		!huellaGobiernoConvocatoriaValida(huellaAtestacion) ||
		!referenciaGobiernoConvocatoriaValida(tokenConsumoRef) ||
		!referenciasGobiernoConvocatoriaDistintas(verificadorRef, atestacionRef, tokenConsumoRef) ||
		!instanteGobiernoConvocatoriaCanonico(emitidaEn) ||
		!instanteGobiernoConvocatoriaCanonico(validaHasta) || !validaHasta.After(emitidaEn) ||
		validaHasta.Sub(emitidaEn) > VigenciaMaximaAtestacionVerificacionConvocatoria {
		return ErrComprobacionDependenciasConvocatoriaInvalida
	}
	return nil
}

func ultimaEdicionConvocatoria(v dominiobolsa.VersionConvocatoriaGobernada) time.Time {
	if !v.UltimaModificacionEn.IsZero() {
		return v.UltimaModificacionEn
	}
	return v.CreadaEn
}

func noAntesAprobacionConvocatoria(
	v dominiobolsa.VersionConvocatoriaGobernada,
	accion AccionAprobacionConvocatoria,
) time.Time {
	if accion == AccionAprobacionRetirarConvocatoria {
		return v.PublicadaEn
	}
	return ultimaEdicionConvocatoria(v)
}
