package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrSolicitudCorreoLlamamientoInvalida    = errors.New("contratacion temporal: solicitud de correo de llamamiento invalida")
	ErrResultadoCorreoLlamamientoNoConfiable = errors.New("contratacion temporal: resultado de correo de llamamiento no confiable")
)

// SolicitudDespacharCorreoLlamamiento liga el despacho a la intención local.
// Sus cinco referencias forman parte de su huella estable.
type SolicitudDespacharCorreoLlamamiento struct {
	OrganizacionRef   string
	ExpedienteRef     string
	LlamamientoRef    string
	ComunicacionRef   string
	IntencionEnvioRef string
}

func (s SolicitudDespacharCorreoLlamamiento) Validar() error {
	if !ReferenciaOpacaValida(s.OrganizacionRef) || !ReferenciaOpacaValida(s.ExpedienteRef) ||
		!ReferenciaOpacaValida(s.LlamamientoRef) || !ReferenciaOpacaValida(s.ComunicacionRef) ||
		!ReferenciaOpacaValida(s.IntencionEnvioRef) {
		return ErrSolicitudCorreoLlamamientoInvalida
	}
	return nil
}

func (s SolicitudDespacharCorreoLlamamiento) SerializarCanonico() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrSolicitudCorreoLlamamientoInvalida
	}
	return json.Marshal(s)
}

func (s SolicitudDespacharCorreoLlamamiento) HuellaSHA256() (string, error) {
	b, err := s.SerializarCanonico()
	if err != nil {
		return "", err
	}
	huella := sha256.Sum256(b)
	return hex.EncodeToString(huella[:]), nil
}

type EstadoCorreoLlamamiento uint8

const (
	CorreoLlamamientoIniciado EstadoCorreoLlamamiento = iota + 1
	CorreoLlamamientoNoAceptadoTransitorio
	CorreoLlamamientoNoAceptadoPermanente
	CorreoLlamamientoIndeterminado
	CorreoLlamamientoAceptadoPorRelay
)

func (e EstadoCorreoLlamamiento) Valido() bool {
	return e >= CorreoLlamamientoIniciado && e <= CorreoLlamamientoAceptadoPorRelay
}
func (e EstadoCorreoLlamamiento) EsResultado() bool {
	return e >= CorreoLlamamientoNoAceptadoTransitorio && e <= CorreoLlamamientoAceptadoPorRelay
}

func (e EstadoCorreoLlamamiento) textoResultadoCanonico() (string, error) {
	switch e {
	case CorreoLlamamientoNoAceptadoTransitorio:
		return "no_aceptado_transitorio", nil
	case CorreoLlamamientoNoAceptadoPermanente:
		return "no_aceptado_permanente", nil
	case CorreoLlamamientoIndeterminado:
		return "indeterminado", nil
	case CorreoLlamamientoAceptadoPorRelay:
		return "aceptado_por_relay", nil
	default:
		return "", ErrResultadoCorreoLlamamientoNoConfiable
	}
}

type ReservaIntentoCorreoLlamamiento struct {
	IntentoRef      string
	MessageID       string
	FechaOrigen     time.Time
	YaReservado     bool
	SolicitudHuella string
	Estado          EstadoCorreoLlamamiento
}

const PlantillaCorreoLlamamientoV1 = "llamamiento_rrhh_v1"

// SolicitudRegistrarResultadoCorreoLlamamiento fija la preimagen que consume
// la autorización del resultado. El orden de campos es parte del contrato PG.
type SolicitudRegistrarResultadoCorreoLlamamiento struct {
	OrganizacionRef   string
	ExpedienteRef     string
	LlamamientoRef    string
	ComunicacionRef   string
	IntencionEnvioRef string
	IntentoRef        string
	SolicitudHuella   string
	Estado            EstadoCorreoLlamamiento
	PlantillaRef      string
	VersionEsperada   uint64
}

func NuevaSolicitudRegistrarResultadoCorreoLlamamiento(s SolicitudDespacharCorreoLlamamiento, r ReservaIntentoCorreoLlamamiento, estado EstadoCorreoLlamamiento, plantilla string) (SolicitudRegistrarResultadoCorreoLlamamiento, error) {
	if r.ValidarPara(s) != nil || !estado.EsResultado() || plantilla != PlantillaCorreoLlamamientoV1 || r.YaReservado || r.Estado != CorreoLlamamientoIniciado {
		return SolicitudRegistrarResultadoCorreoLlamamiento{}, ErrResultadoCorreoLlamamientoNoConfiable
	}
	return SolicitudRegistrarResultadoCorreoLlamamiento{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, LlamamientoRef: s.LlamamientoRef, ComunicacionRef: s.ComunicacionRef, IntencionEnvioRef: s.IntencionEnvioRef, IntentoRef: r.IntentoRef, SolicitudHuella: r.SolicitudHuella, Estado: estado, PlantillaRef: plantilla, VersionEsperada: 1}, nil
}

func (s SolicitudRegistrarResultadoCorreoLlamamiento) Validar() error {
	peticion := SolicitudDespacharCorreoLlamamiento{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, LlamamientoRef: s.LlamamientoRef, ComunicacionRef: s.ComunicacionRef, IntencionEnvioRef: s.IntencionEnvioRef}
	huella, err := peticion.HuellaSHA256()
	if err != nil || !ReferenciaOpacaValida(s.IntentoRef) || s.SolicitudHuella != huella || !s.Estado.EsResultado() || s.PlantillaRef != PlantillaCorreoLlamamientoV1 || s.VersionEsperada != 1 {
		return ErrResultadoCorreoLlamamientoNoConfiable
	}
	return nil
}
func (s SolicitudRegistrarResultadoCorreoLlamamiento) SerializarCanonico() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrResultadoCorreoLlamamientoNoConfiable
	}
	estado, err := s.Estado.textoResultadoCanonico()
	if err != nil {
		return nil, ErrResultadoCorreoLlamamientoNoConfiable
	}
	// Es la preimagen JSON10 exacta de CT88/T13: el enum Go nunca cruza la
	// frontera como uint8. HuellaSHA256 y PostgreSQL consumen estos mismos bytes.
	return json.Marshal(struct {
		OrganizacionRef   string
		ExpedienteRef     string
		LlamamientoRef    string
		ComunicacionRef   string
		IntencionEnvioRef string
		IntentoRef        string
		SolicitudHuella   string
		Estado            string
		PlantillaRef      string
		VersionEsperada   uint64
	}{s.OrganizacionRef, s.ExpedienteRef, s.LlamamientoRef, s.ComunicacionRef,
		s.IntencionEnvioRef, s.IntentoRef, s.SolicitudHuella, estado,
		s.PlantillaRef, s.VersionEsperada})
}
func (s SolicitudRegistrarResultadoCorreoLlamamiento) HuellaSHA256() (string, error) {
	b, err := s.SerializarCanonico()
	if err != nil {
		return "", err
	}
	huella := sha256.Sum256(b)
	return hex.EncodeToString(huella[:]), nil
}

func (r ReservaIntentoCorreoLlamamiento) Validar() error {
	if !ReferenciaOpacaValida(r.IntentoRef) || r.MessageID == "" || !InstanteUTCCanonico(r.FechaOrigen) {
		return ErrResultadoCorreoLlamamientoNoConfiable
	}
	return nil
}

func (r ReservaIntentoCorreoLlamamiento) ValidarPara(s SolicitudDespacharCorreoLlamamiento) error {
	huella, err := s.HuellaSHA256()
	if err != nil || r.Validar() != nil || r.SolicitudHuella != huella || !r.Estado.Valido() || (!r.YaReservado && r.Estado != CorreoLlamamientoIniciado) {
		return ErrResultadoCorreoLlamamientoNoConfiable
	}
	return nil
}

// CapacidadFinalizacionIntentoCorreoLlamamiento sólo sirve para la reserva
// nueva que la creó. No expone su secreto en formato ni JSON.
type CapacidadFinalizacionIntentoCorreoLlamamiento struct {
	intentoRef, solicitudHuella string
	secreto                     [32]byte
}

func NuevaCapacidadFinalizacionIntentoCorreoLlamamiento(r ReservaIntentoCorreoLlamamiento, secreto []byte) (CapacidadFinalizacionIntentoCorreoLlamamiento, error) {
	var c CapacidadFinalizacionIntentoCorreoLlamamiento
	decodificada, errHuella := hex.DecodeString(r.SolicitudHuella)
	if r.Validar() != nil || r.YaReservado || r.Estado != CorreoLlamamientoIniciado || len(r.SolicitudHuella) != 64 || errHuella != nil || hex.EncodeToString(decodificada) != r.SolicitudHuella || r.SolicitudHuella == strings.Repeat("0", 64) || len(secreto) != len(c.secreto) {
		return c, ErrResultadoCorreoLlamamientoNoConfiable
	}
	copy(c.secreto[:], secreto)
	if secretoNulo(c.secreto) {
		return CapacidadFinalizacionIntentoCorreoLlamamiento{}, ErrResultadoCorreoLlamamientoNoConfiable
	}
	c.intentoRef, c.solicitudHuella = r.IntentoRef, r.SolicitudHuella
	return c, nil
}

func (c CapacidadFinalizacionIntentoCorreoLlamamiento) EsCero() bool {
	return c == (CapacidadFinalizacionIntentoCorreoLlamamiento{})
}
func (c CapacidadFinalizacionIntentoCorreoLlamamiento) ValidarPara(r ReservaIntentoCorreoLlamamiento) error {
	if c.EsCero() || r.Validar() != nil || r.YaReservado || r.Estado != CorreoLlamamientoIniciado || r.IntentoRef != c.intentoRef || r.SolicitudHuella != c.solicitudHuella || secretoNulo(c.secreto) {
		return ErrResultadoCorreoLlamamientoNoConfiable
	}
	return nil
}
func (c CapacidadFinalizacionIntentoCorreoLlamamiento) ValidarParaSolicitudResultado(s SolicitudRegistrarResultadoCorreoLlamamiento) error {
	if c.EsCero() || s.Validar() != nil || s.IntentoRef != c.intentoRef || s.SolicitudHuella != c.solicitudHuella || secretoNulo(c.secreto) {
		return ErrResultadoCorreoLlamamientoNoConfiable
	}
	return nil
}
func (c CapacidadFinalizacionIntentoCorreoLlamamiento) ExportarSecretoParaConsumidor() []byte {
	return append([]byte(nil), c.secreto[:]...)
}
func (CapacidadFinalizacionIntentoCorreoLlamamiento) String() string {
	return "CapacidadFinalizacionIntentoCorreoLlamamiento{redactada}"
}
func (CapacidadFinalizacionIntentoCorreoLlamamiento) GoString() string {
	return "CapacidadFinalizacionIntentoCorreoLlamamiento{redactada}"
}
func (CapacidadFinalizacionIntentoCorreoLlamamiento) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Redactado bool `json:"redactado"`
	}{true})
}

func secretoNulo(secreto [32]byte) bool {
	for _, valor := range secreto {
		if valor != 0 {
			return false
		}
	}
	return true
}
