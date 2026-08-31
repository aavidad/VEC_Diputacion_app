package ports

import (
	"context"
	"errors"
	"sort"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var (
	ErrSolicitudPropuestaFormalizacionInvalida = errors.New(
		"contratacion temporal: solicitud de propuesta de formalizacion invalida",
	)
	ErrResultadoPropuestaFormalizacionNoConfiable = errors.New(
		"contratacion temporal: resultado de propuesta de formalizacion no confiable",
	)
	ErrOperacionPropuestaFormalizacionDenegada = errors.New(
		"contratacion temporal: operacion de propuesta de formalizacion denegada",
	)
	ErrVersionPropuestaFormalizacionEnConflicto = errors.New(
		"contratacion temporal: version de propuesta de formalizacion en conflicto",
	)
	ErrClavePropuestaFormalizacionUsada = errors.New(
		"contratacion temporal: clave de propuesta de formalizacion usada con otros datos",
	)
	ErrResolucionLlamamientoNoAceptada = errors.New(
		"contratacion temporal: resolucion de llamamiento no aceptada",
	)
)

const (
	// MaximoAnexosPropuestaFormalizacion es un limite tecnico previo a cualquier
	// copia u ordenacion. La politica gobernada puede imponer uno menor.
	MaximoAnexosPropuestaFormalizacion = 256
	// MaximoBytesAnexosPropuestaFormalizacion limita la suma declarada. No
	// autoriza a leer, reservar ni materializar esos bytes.
	MaximoBytesAnexosPropuestaFormalizacion uint64 = 256 * 1024 * 1024
)

// SnapshotGobernadoFormalizacion identifica una publicacion inmutable. La
// referencia no transporta modalidades, plazos, firmantes ni reglas legales.
type SnapshotGobernadoFormalizacion struct {
	Referencia   string
	Version      uint64
	HuellaSHA256 string
}

func (s SnapshotGobernadoFormalizacion) valido() bool {
	return domain.ReferenciaOpacaValida(s.Referencia) &&
		enteroSeguroBolsa(s.Version) && huellaSHA256BolsaValida(s.HuellaSHA256)
}

// AnexoPropuestaFormalizacion vincula una version ya identificada por su
// autoridad documental. No contiene bytes, rutas, URLs ni datos personales.
type AnexoPropuestaFormalizacion struct {
	DocumentoRef string
	Version      uint64
	HuellaSHA256 string
	TamanoBytes  uint64
}

func (a AnexoPropuestaFormalizacion) valido() bool {
	return domain.ReferenciaOpacaValida(a.DocumentoRef) &&
		enteroSeguroBolsa(a.Version) && huellaSHA256BolsaValida(a.HuellaSHA256) &&
		a.TamanoBytes > 0 && a.TamanoBytes <= MaximoBytesAnexosPropuestaFormalizacion
}

// SolicitudPropuestaFormalizacion solo transporta referencias y snapshots. La
// resolucion y su recibo se tratan como afirmaciones nominales hasta que la
// transaccion local los relee y acredita como una aceptacion exacta.
type SolicitudPropuestaFormalizacion struct {
	ClaveIdempotencia                string
	OrganizacionRef                  string
	ExpedienteRef                    string
	LlamamientoRef                   string
	ResolucionLlamamientoAceptadaRef string
	ReciboResolucionAceptadaRef      string
	VersionEsperada                  uint64
	TipoFormalizacion                SnapshotGobernadoFormalizacion
	Plantilla                        SnapshotGobernadoFormalizacion
	Anexos                           []AnexoPropuestaFormalizacion
	PoliticaFirma                    SnapshotGobernadoFormalizacion
	PlanFirma                        SnapshotGobernadoFormalizacion
}

// Normalizar valida los limites antes de copiar, ordena los anexos y elimina
// repeticiones exactas. Una misma referencia/version con otro compromiso es
// conflicto y nunca se resuelve por precedencia.
func (s SolicitudPropuestaFormalizacion) Normalizar() (
	SolicitudPropuestaFormalizacion,
	error,
) {
	if !ClaveIdempotenciaValida(s.ClaveIdempotencia) ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(s.LlamamientoRef) ||
		!domain.ReferenciaOpacaValida(s.ResolucionLlamamientoAceptadaRef) ||
		!domain.ReferenciaOpacaValida(s.ReciboResolucionAceptadaRef) ||
		!enteroSeguroBolsa(s.VersionEsperada) ||
		s.VersionEsperada == MaximoEnteroSeguroIntegracionBolsa ||
		!s.TipoFormalizacion.valido() || !s.Plantilla.valido() ||
		!s.PoliticaFirma.valido() || !s.PlanFirma.valido() ||
		len(s.Anexos) > MaximoAnexosPropuestaFormalizacion {
		return SolicitudPropuestaFormalizacion{}, ErrSolicitudPropuestaFormalizacionInvalida
	}

	normalizada := s
	normalizada.Anexos = append([]AnexoPropuestaFormalizacion(nil), s.Anexos...)
	sort.Slice(normalizada.Anexos, func(i, j int) bool {
		if normalizada.Anexos[i].DocumentoRef != normalizada.Anexos[j].DocumentoRef {
			return normalizada.Anexos[i].DocumentoRef < normalizada.Anexos[j].DocumentoRef
		}
		return normalizada.Anexos[i].Version < normalizada.Anexos[j].Version
	})

	deduplicados := normalizada.Anexos[:0]
	var total uint64
	for _, anexo := range normalizada.Anexos {
		if !anexo.valido() {
			return SolicitudPropuestaFormalizacion{}, ErrSolicitudPropuestaFormalizacionInvalida
		}
		if len(deduplicados) > 0 {
			anterior := deduplicados[len(deduplicados)-1]
			if anterior.DocumentoRef == anexo.DocumentoRef && anterior.Version == anexo.Version {
				if anterior != anexo {
					return SolicitudPropuestaFormalizacion{}, ErrSolicitudPropuestaFormalizacionInvalida
				}
				continue
			}
		}
		if anexo.TamanoBytes > MaximoBytesAnexosPropuestaFormalizacion-total {
			return SolicitudPropuestaFormalizacion{}, ErrSolicitudPropuestaFormalizacionInvalida
		}
		total += anexo.TamanoBytes
		deduplicados = append(deduplicados, anexo)
	}
	normalizada.Anexos = deduplicados
	return normalizada, nil
}

// Validar exige la forma canonica que debe cruzar el puerto transaccional.
func (s SolicitudPropuestaFormalizacion) Validar() error {
	normalizada, err := s.Normalizar()
	if err != nil || !solicitudesPropuestaFormalizacionIguales(s, normalizada) {
		return ErrSolicitudPropuestaFormalizacionInvalida
	}
	return nil
}

func (s SolicitudPropuestaFormalizacion) Clonar() SolicitudPropuestaFormalizacion {
	clon := s
	clon.Anexos = append([]AnexoPropuestaFormalizacion(nil), s.Anexos...)
	return clon
}

type EstadoResultadoPropuestaFormalizacion string

const (
	ResultadoPropuestaFormalizacionConfirmado EstadoResultadoPropuestaFormalizacion = "confirmado"
	ResultadoPropuestaFormalizacionReplay     EstadoResultadoPropuestaFormalizacion = "replay_confirmado"
)

func (e EstadoResultadoPropuestaFormalizacion) valido() bool {
	return e == ResultadoPropuestaFormalizacionConfirmado ||
		e == ResultadoPropuestaFormalizacionReplay
}

// ResultadoPropuestaFormalizacion acredita exclusivamente el commit local de
// propuesta, recibo y auditoria. No acredita renderizado, firma, custodia,
// registro, descarga ni la creacion de una orden documental externa.
type ResultadoPropuestaFormalizacion struct {
	Solicitud         SolicitudPropuestaFormalizacion
	PropuestaRef      string
	ReciboLocalRef    string
	AuditoriaRef      string
	VersionResultante uint64
	ConfirmadaEn      time.Time
	Estado            EstadoResultadoPropuestaFormalizacion
}

func (r ResultadoPropuestaFormalizacion) ValidarPara(
	solicitud SolicitudPropuestaFormalizacion,
) error {
	if solicitud.Validar() != nil || r.Solicitud.Validar() != nil ||
		!solicitudesPropuestaFormalizacionIguales(r.Solicitud, solicitud) ||
		!domain.ReferenciaOpacaValida(r.PropuestaRef) ||
		!domain.ReferenciaOpacaValida(r.ReciboLocalRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		r.VersionResultante != solicitud.VersionEsperada+1 ||
		!domain.InstanteUTCCanonico(r.ConfirmadaEn) || !r.Estado.valido() {
		return ErrResultadoPropuestaFormalizacionNoConfiable
	}
	return nil
}

func (r ResultadoPropuestaFormalizacion) EsReplayConfirmado() bool {
	return r.Estado == ResultadoPropuestaFormalizacionReplay
}

func (r ResultadoPropuestaFormalizacion) Clonar() ResultadoPropuestaFormalizacion {
	clon := r
	clon.Solicitud = r.Solicitud.Clonar()
	return clon
}

func (r ResultadoPropuestaFormalizacion) EsCero() bool {
	return solicitudPropuestaFormalizacionEsCero(r.Solicitud) &&
		r.PropuestaRef == "" && r.ReciboLocalRef == "" && r.AuditoriaRef == "" &&
		r.VersionResultante == 0 && r.ConfirmadaEn.IsZero() && r.Estado == ""
}

// TransaccionPropuestaFormalizacion representa un unico commit local. La
// implementacion debe releer con OCC el expediente y la resolucion aceptada,
// cotejar su recibo, revalidar las publicaciones exactas de tipo, plantilla,
// anexos, politica y plan, y confirmar propuesta, recibo y auditoria juntos.
// Este corte no permite crear ni despachar una intencion documental.
type TransaccionPropuestaFormalizacion interface {
	ConfirmarPropuesta(
		context.Context,
		SolicitudPropuestaFormalizacion,
	) (ResultadoPropuestaFormalizacion, error)
}

func solicitudesPropuestaFormalizacionIguales(
	primera SolicitudPropuestaFormalizacion,
	segunda SolicitudPropuestaFormalizacion,
) bool {
	if primera.ClaveIdempotencia != segunda.ClaveIdempotencia ||
		primera.OrganizacionRef != segunda.OrganizacionRef ||
		primera.ExpedienteRef != segunda.ExpedienteRef ||
		primera.LlamamientoRef != segunda.LlamamientoRef ||
		primera.ResolucionLlamamientoAceptadaRef != segunda.ResolucionLlamamientoAceptadaRef ||
		primera.ReciboResolucionAceptadaRef != segunda.ReciboResolucionAceptadaRef ||
		primera.VersionEsperada != segunda.VersionEsperada ||
		primera.TipoFormalizacion != segunda.TipoFormalizacion ||
		primera.Plantilla != segunda.Plantilla ||
		primera.PoliticaFirma != segunda.PoliticaFirma ||
		primera.PlanFirma != segunda.PlanFirma || len(primera.Anexos) != len(segunda.Anexos) {
		return false
	}
	for indice := range primera.Anexos {
		if primera.Anexos[indice] != segunda.Anexos[indice] {
			return false
		}
	}
	return true
}

func solicitudPropuestaFormalizacionEsCero(s SolicitudPropuestaFormalizacion) bool {
	return s.ClaveIdempotencia == "" && s.OrganizacionRef == "" &&
		s.ExpedienteRef == "" && s.LlamamientoRef == "" &&
		s.ResolucionLlamamientoAceptadaRef == "" &&
		s.ReciboResolucionAceptadaRef == "" && s.VersionEsperada == 0 &&
		s.TipoFormalizacion == (SnapshotGobernadoFormalizacion{}) &&
		s.Plantilla == (SnapshotGobernadoFormalizacion{}) && len(s.Anexos) == 0 &&
		s.PoliticaFirma == (SnapshotGobernadoFormalizacion{}) &&
		s.PlanFirma == (SnapshotGobernadoFormalizacion{})
}
