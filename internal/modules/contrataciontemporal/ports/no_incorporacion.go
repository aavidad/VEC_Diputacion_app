package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// No incorporación (CT124, AD3-88): RRHH registra que la persona aceptada
// no se incorpora. Recurso, finalidad y audiencia propios; la baja la aplica
// Bolsa (000042) y el llamamiento continúa con el siguiente candidato.
const (
	OperacionRegistrarNoIncorporacion        = "registrar_no_incorporacion"
	TipoRecursoNoIncorporacion               = "no_incorporacion_contratacion_temporal"
	FinalidadRegistrarNoIncorporacion        = "registrar_no_incorporacion_contratacion_temporal"
	AudienciaConsumoNoIncorporacionV1        = "vec_contratacion_temporal.no_incorporacion.v1"
	DominioAmbitoIdempotenciaNoIncorporacion = "vec.contratacion-temporal.no-incorporacion.ambito"
	DominioHuellaPeticionNoIncorporacion     = "vec.contratacion-temporal.no-incorporacion.peticion"
	maximoMotivosNoIncorporacion             = 32
)

var (
	// ErrSinAceptacion: el expediente no tiene una aceptación confirmada con
	// su propuesta de nombramiento pendiente de continuar.
	ErrSinAceptacion = errors.New("contratacion temporal: expediente sin aceptacion vigente")
	// ErrIncorporacionExistente: ya consta la incorporación.
	ErrIncorporacionExistente = errors.New("contratacion temporal: la incorporacion ya consta")
	// ErrNoIncorporacionExistente: la aceptación ya tiene no incorporación.
	ErrNoIncorporacionExistente = errors.New("contratacion temporal: no incorporacion ya registrada")
	// ErrFechaNoIncorporacionNoAdmitida: notificación futura o anterior a la
	// aceptación.
	ErrFechaNoIncorporacionNoAdmitida = errors.New("contratacion temporal: fecha de no incorporacion no admitida")
)

// MaterialNoIncorporacion es la intención exacta que se sella y persiste.
type MaterialNoIncorporacion struct {
	OrganizacionRef, ExpedienteRef, ActorRef, PerfilRef string
	VersionEsperada                                     uint64
	ClaveIdempotencia                                   string
	Datos                                               domain.DatosNoIncorporacion
}

func (m MaterialNoIncorporacion) Valido() bool {
	return identidadOperacionValida(m.OrganizacionRef, m.ExpedienteRef, m.ActorRef, m.PerfilRef, m.VersionEsperada, m.ClaveIdempotencia) &&
		m.Datos.ValidarPara(m.ActorRef) == nil
}

// MotivoNoIncorporacion es un motivo del catálogo (c13) con la consecuencia
// que aplica Bolsa.
type MotivoNoIncorporacion struct {
	Clave             string
	Etiqueta          string
	ConsecuenciaClave string
}

// ReglaNoIncorporacion es la regla c13 vigente: motivos y segregación.
type ReglaNoIncorporacion struct {
	Motivos        []MotivoNoIncorporacion
	SegundaPersona bool
}

// Valida acota los motivos antes de publicarlos o usarlos.
func (r ReglaNoIncorporacion) Valida() bool {
	if len(r.Motivos) == 0 || len(r.Motivos) > maximoMotivosNoIncorporacion {
		return false
	}
	vistos := map[string]bool{}
	for _, m := range r.Motivos {
		if !domain.MotivoNoIncorporacionValido(m.Clave) || !domain.ConsecuenciaNoIncorporacionValida(m.ConsecuenciaClave) ||
			m.Etiqueta == "" || vistos[m.Clave] {
			return false
		}
		vistos[m.Clave] = true
	}
	return true
}

// Motivo devuelve el motivo del catálogo con esa clave.
func (r ReglaNoIncorporacion) Motivo(clave string) (MotivoNoIncorporacion, bool) {
	for _, m := range r.Motivos {
		if m.Clave == clave {
			return m, true
		}
	}
	return MotivoNoIncorporacion{}, false
}

// FuenteReglaNoIncorporacion resuelve la regla c13 y la política que ampara
// el registro, con el motivo de su ruta.
type FuenteReglaNoIncorporacion interface {
	ReglaNoIncorporacion(ctx context.Context, instante time.Time) (ReglaNoIncorporacion, PoliticaOperacionSeguimiento, error)
}

// EstadoNoIncorporacion es la no incorporación registrada, para el detalle.
type EstadoNoIncorporacion struct {
	MotivoClave       string
	ConsecuenciaClave string
	ResolucionRef     string
	ResolucionSHA256  string
	ResueltaPor       string
	FechaNotificacion string
	ReciboRef         string
	RegistradaEn      time.Time
}

func (e EstadoNoIncorporacion) Valido() bool {
	return domain.MotivoNoIncorporacionValido(e.MotivoClave) && domain.ConsecuenciaNoIncorporacionValida(e.ConsecuenciaClave) &&
		domain.ReferenciaOpacaValida(e.ResolucionRef) && huellaSHA256OperacionAnalisisValida(e.ResolucionSHA256) &&
		domain.ReferenciaOpacaValida(e.ResueltaPor) && fechaCivilTextoValida(e.FechaNotificacion) &&
		domain.ReferenciaOpacaValida(e.ReciboRef) && domain.InstanteUTCCanonico(e.RegistradaEn)
}

// AntecedenteNoIncorporacion acompaña a la continuación cuyo antecedente es
// una aceptación seguida de no incorporación: la intención y el comando de
// siguiente candidato son los de la no incorporación.
type AntecedenteNoIncorporacion struct {
	ReciboRef         string
	IntencionRef      string
	ComandoRef        string
	VersionResultante uint64
	RegistradaEn      time.Time
}

func (a AntecedenteNoIncorporacion) Valido() bool {
	return domain.ReferenciaOpacaValida(a.ReciboRef) && domain.ReferenciaOpacaValida(a.IntencionRef) &&
		domain.ReferenciaOpacaValida(a.ComandoRef) && a.VersionResultante > 7 &&
		a.VersionResultante <= MaximoEnteroSeguroIntegracionBolsa && domain.InstanteUTCCanonico(a.RegistradaEn)
}

// LectorPublicacionNoIncorporacionesBolsa lee las no incorporaciones que CT
// publica para Bolsa, posteriores al cursor y en orden, con la misma forma y
// marca de agua que la publicación de contratos.
type LectorPublicacionNoIncorporacionesBolsa interface {
	LeerNoIncorporacionesBolsa(ctx context.Context, desde CursorPublicacionContratosBolsa, limite int) ([]EventoContratoBolsaPublicado, error)
}
