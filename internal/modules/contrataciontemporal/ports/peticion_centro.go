package ports

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	OperacionPresentarPeticionCentro = "presentar"
	OperacionRatificarPeticionCentro = "ratificar"
	AccionPresentarPeticionCentro    = "contratacion_temporal.peticion_centro.presentar"
	AccionRatificarPeticionCentro    = "contratacion_temporal.peticion_centro.ratificar"
	AccionConsultarPeticionCentro    = "contratacion_temporal.peticion_centro.consultar"
	TipoRecursoPeticionCentro        = "peticion_centro"
)

var (
	ErrPeticionCentroNoDisponible      = errors.New("contratacion temporal: peticion de centro no disponible")
	ErrClavePeticionCentroUsada        = errors.New("contratacion temporal: clave de peticion de centro usada")
	ErrReciboPeticionCentroNoConfiable = errors.New("contratacion temporal: recibo de peticion de centro no confiable")
)

// ComandoPeticionCentro contiene exclusivamente datos del formulario. Centro y
// contacto de la solicitud no son la identidad ni el cargo de quien actúa.
type ComandoPeticionCentro struct {
	Operacion         string                     `json:"operacion"`
	ClaveIdempotencia string                     `json:"clave_idempotencia"`
	EsquemaAlta       string                     `json:"esquema,omitempty"`
	NecesidadEntrada  *domain.DatosNecesidadAlta `json:"necesidad,omitempty"`
	PeticionRef       string                     `json:"peticion_ref,omitempty"`
	VersionEsperada   uint64                     `json:"version_esperada,omitempty"`
	Solicitud         *domain.SolicitudCentro    `json:"solicitud,omitempty"`
	Motivo            string                     `json:"motivo,omitempty"`
}

func (s ComandoPeticionCentro) Validar() error {
	if !claveUUIDPeticionCentro(s.ClaveIdempotencia) {
		return domain.ErrPeticionCentroInvalida
	}
	switch s.Operacion {
	case OperacionPresentarPeticionCentro:
		if s.Solicitud == nil || s.Solicitud.Validar() != nil || s.Solicitud.Necesidad != nil ||
			s.PeticionRef != "" || s.VersionEsperada != 0 || s.Motivo != "" ||
			!entradaNecesidadPeticionCentroValida(s) {
			return domain.ErrPeticionCentroInvalida
		}
	case OperacionRatificarPeticionCentro:
		if !domain.ReferenciaOpacaValida(s.PeticionRef) || s.VersionEsperada != 1 || s.Solicitud != nil ||
			s.EsquemaAlta != "" || s.NecesidadEntrada != nil ||
			s.Motivo == "" || strings.TrimSpace(s.Motivo) != s.Motivo || len(s.Motivo) > 1000 || strings.IndexFunc(s.Motivo, unicode.IsControl) >= 0 {
			return domain.ErrPeticionCentroInvalida
		}
	default:
		return domain.ErrPeticionCentroInvalida
	}
	return nil
}

func entradaNecesidadPeticionCentroValida(s ComandoPeticionCentro) bool {
	if s.EsquemaAlta == "" {
		return s.NecesidadEntrada == nil
	}
	if s.EsquemaAlta != EsquemaAltaNecesidadV1 || s.NecesidadEntrada == nil || s.Solicitud == nil {
		return false
	}
	n := s.NecesidadEntrada
	if personas := n.Campos["numero_personas"]; personas != "" &&
		personas != strconv.FormatUint(uint64(s.Solicitud.NumeroPersonas), 10) {
		return false
	}
	return n.Esquema == "vec.ct.necesidad_alta.v1" &&
		domain.ReferenciaOpacaValida(n.CatalogoRef) && n.CatalogoVersion > 0 &&
		patronHuellaPuestoRPTAlta.MatchString(n.CatalogoHuellaSHA256) &&
		n.CausaClave == s.Solicitud.MotivoClave && n.Periodo == s.Solicitud.Periodo &&
		n.JornadaMinutos == s.Solicitud.JornadaMinutos && n.JornadaMinutos > 0 &&
		len(n.Campos) > 0 && len(n.Campos) <= 32 && len(n.CatalogoInstantanea) == 0
}

func (s ComandoPeticionCentro) Clonar() (ComandoPeticionCentro, error) {
	if err := s.Validar(); err != nil {
		return ComandoPeticionCentro{}, err
	}
	if s.Solicitud != nil {
		c, err := s.Solicitud.Clonar()
		if err != nil {
			return ComandoPeticionCentro{}, err
		}
		s.Solicitud = &c
	}
	if s.NecesidadEntrada != nil {
		copia := *s.NecesidadEntrada
		copia.Campos = make(map[string]string, len(s.NecesidadEntrada.Campos))
		for clave, valor := range s.NecesidadEntrada.Campos {
			copia.Campos[clave] = valor
		}
		s.NecesidadEntrada = &copia
	}
	return s, nil
}

func claveUUIDPeticionCentro(s string) bool {
	if !ClaveIdempotenciaValida(s) || len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' || s[14] != '4' || !strings.ContainsRune("89ab", rune(s[19])) {
		return false
	}
	for i, r := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

// MaterialPeticionCentro solo lo prepara el caso de uso tras resolver la
// adscripción desde una autoridad confiable. No se deserializa desde HTTP.
type MaterialPeticionCentro struct {
	Comando  ComandoPeticionCentro      `json:"comando"`
	Actor    domain.ActorPeticionCentro `json:"actor"`
	Peticion domain.DatosPeticionCentro `json:"peticion"`
}

func (m MaterialPeticionCentro) Validar() error {
	if m.Comando.Validar() != nil {
		return domain.ErrPeticionCentroInvalida
	}
	if _, err := domain.RehidratarPeticionCentro(m.Peticion); err != nil {
		return err
	}
	d := m.Peticion
	if m.Comando.Operacion == OperacionPresentarPeticionCentro {
		solicitud := *m.Comando.Solicitud
		if m.Comando.EsquemaAlta == EsquemaAltaNecesidadV1 {
			necesidad := d.Solicitud.Necesidad
			if necesidad == nil || necesidad.ValidarInstantanea() != nil {
				return domain.ErrPeticionCentroInvalida
			}
			sinInstantanea := *necesidad
			sinInstantanea.CatalogoInstantanea = nil
			if !reflect.DeepEqual(*m.Comando.NecesidadEntrada, sinInstantanea) {
				return domain.ErrPeticionCentroInvalida
			}
			solicitud.Necesidad = necesidad
		} else if d.Solicitud.Necesidad != nil {
			return domain.ErrPeticionCentroInvalida
		}
		if d.Referencia != "peticion:centro:"+m.Comando.ClaveIdempotencia || d.Version != 1 ||
			m.Actor != d.Configuracion.Solicitante || !reflect.DeepEqual(solicitud, d.Solicitud) {
			return domain.ErrPeticionCentroInvalida
		}
	} else if d.Referencia != m.Comando.PeticionRef || d.Version != m.Comando.VersionEsperada+1 ||
		m.Actor != d.Configuracion.Ratificador || d.MotivoRatificacion != m.Comando.Motivo {
		return domain.ErrPeticionCentroInvalida
	}
	return nil
}

func (m MaterialPeticionCentro) MismaPeticion(actor domain.ActorPeticionCentro, s ComandoPeticionCentro) bool {
	if m.Validar() != nil || s.Validar() != nil || m.Actor != actor {
		return false
	}
	historico := m.Comando
	if historico.Solicitud != nil {
		copia := *historico.Solicitud
		copia.Periodo.PoliticaFin = domain.PoliticaFin{}
		historico.Solicitud = &copia
	}
	if historico.NecesidadEntrada != nil {
		copia := *historico.NecesidadEntrada
		copia.Periodo.PoliticaFin = domain.PoliticaFin{}
		historico.NecesidadEntrada = &copia
	}
	return reflect.DeepEqual(historico, s)
}

type ReciboPeticionCentro struct {
	ReciboRef    string    `json:"recibo_ref"`
	PeticionRef  string    `json:"peticion_ref"`
	Version      uint64    `json:"version"`
	Estado       string    `json:"estado"`
	ActorRef     string    `json:"actor_ref"`
	RegistradoEn time.Time `json:"registrado_en"`
	EstadoLocal  string    `json:"estado_local"`
}

func (r ReciboPeticionCentro) ValidarPara(m MaterialPeticionCentro) error {
	if m.Validar() != nil || !domain.ReferenciaOpacaValida(r.ReciboRef) || r.PeticionRef != m.Peticion.Referencia ||
		r.Version != m.Peticion.Version || r.Estado != m.Peticion.Estado || r.ActorRef != m.Actor.ActorRef ||
		!domain.InstanteUTCCanonico(r.RegistradoEn) || r.RegistradoEn != r.RegistradoEn.Truncate(time.Microsecond) ||
		r.RegistradoEn.Before(m.Peticion.CreadaEn) || r.RegistradoEn.Before(m.Peticion.RatificadaEn) ||
		(r.EstadoLocal != "registrado" && r.EstadoLocal != "replay_confirmado") {
		return ErrReciboPeticionCentroNoConfiable
	}
	return nil
}

// La autoridad resuelve identidad y adscripción, no las toma del comando. La
// configuración puede señalar cualquier cargo realmente habilitado: no existe
// una cadena de niveles de mando compilada en Contratación.
type AutoridadPeticionCentro interface {
	ActorPeticionCentro(context.Context) (domain.ActorPeticionCentro, error)
	ConfiguracionPeticionCentro(context.Context, domain.ActorPeticionCentro) (domain.ConfiguracionPeticionCentro, error)
}

// Toda lectura está autorizada para actor/centro; ausencia se representa por nil.
// Confirmar consume una autorización NUEVA aun al recuperar un recibo anterior,
// y confirma conjuntamente estado, recibo e historial. No hay éxito en memoria.
type RepositorioPeticionesCentro interface {
	ConsultarOperacion(context.Context, domain.ActorPeticionCentro, string) (*MaterialPeticionCentro, error)
	ObtenerPeticion(context.Context, domain.ActorPeticionCentro, string) (domain.DatosPeticionCentro, error)
	ConfirmarPeticion(context.Context, MaterialPeticionCentro) (ReciboPeticionCentro, error)
}

type ConsultaPeticionCentro struct {
	Modo       string                     `json:"modo"`
	Actor      domain.ActorPeticionCentro `json:"actor"`
	Referencia string                     `json:"referencia"`
}

func (s ConsultaPeticionCentro) Validar() error {
	if s.Actor.Validar() != nil {
		return domain.ErrRatificacionCentroDenegada
	}
	switch s.Modo {
	case "operacion":
		if claveUUIDPeticionCentro(s.Referencia) {
			return nil
		}
	case "peticion":
		if domain.ReferenciaOpacaValida(s.Referencia) {
			return nil
		}
	case "bandeja":
		if s.Referencia == s.Actor.CentroRef {
			return nil
		}
	}
	return domain.ErrPeticionCentroInvalida
}

type ConsultaBandejaPeticionesCentro interface {
	ListarPeticiones(context.Context, domain.ActorPeticionCentro) ([]domain.DatosPeticionCentro, error)
}
