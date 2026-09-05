package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var ErrIntegracionLlamamientoDesarrollo = errors.New("bolsa: integracion de llamamiento de desarrollo no disponible")

// Evita que una huella hexadecimal aleatoria coincida con la detección de
// documentos de identidad aplicada a referencias. Conserva los 256 bits.
func ReferenciaDesdeHuellaLlamamientoDesarrollo(prefijo, huella string) string {
	if !huellaSHA256LlamamientoValida(huella) {
		return ""
	}
	return prefijo + ":" + strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return 'g' + r - '0'
		}
		return r
	}, huella)
}

const (
	AudienciaIntegracionLlamamientoDesarrollo = "vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1"
	AccionPrepararOrdenDesarrollo             = "bolsa.orden.preparar"
	AccionAbrirLlamamientoDesarrollo          = "bolsa.llamamiento.abrir"
	AccionAceptarLlamamientoRRHHDesarrollo    = "bolsa.llamamiento.aceptacion_rrhh.registrar"
	AccionRenunciarLlamamientoRRHHDesarrollo  = "bolsa.llamamiento.renuncia_rrhh.registrar"
	FinalidadIntegracionLlamamientoDesarrollo = "gestionar_contratacion_temporal"
)

type DisponibilidadLlamamientosDesarrollo struct {
	Bolsa              domain.BolsaConstituida
	Politica           domain.ReferenciaPoliticaLlamamiento
	Necesidad          domain.NecesidadCobertura
	CantidadDisponible uint32
	CantidadExacta     bool
}

// FuenteFirmadaLlamamientosDesarrollo no decide permisos. Su autoridad se limita
// a los datos sintéticos, su orden y el criterio explícito de elegibilidad.
type FuenteFirmadaLlamamientosDesarrollo interface {
	FuenteDatosLlamamiento
	MotorElegibilidadLlamamiento
	ExportarFuenteFirmada(context.Context, string) (json.RawMessage, []byte, error)
}

// PeticionLlamamientoDesarrollo la construye el puente confiable, nunca un DTO
// del navegador. OperacionRef es estable en reintentos; OrdenOperacionRef liga
// una propuesta a la orden durable que ya existe en Bolsa.
type PeticionLlamamientoDesarrollo struct {
	OperacionRef      string
	NecesidadRef      string
	OrdenOperacionRef string
	MaximoPosiciones  uint32
}

// ResolucionLlamamientoDesarrollo liga la aceptación o renuncia a una evaluación y una
// política reacreditadas por la frontera confiable. Las referencias no prueban
// por sí mismas competencia, entrega, origen del correo ni vigencia del plazo.
// Este proveedor no fabrica una evaluación positiva ni calcula plazos legales.
type ResolucionLlamamientoDesarrollo struct {
	AperturaOperacionRef string    `json:"apertura_operacion_ref"`
	JustificanteRef      string    `json:"justificante_ref"`
	EvaluacionPlazoRef   string    `json:"evaluacion_plazo_ref"`
	PoliticaRef          string    `json:"politica_ref"`
	PoliticaVersion      uint64    `json:"politica_version"`
	PoliticaSHA256       string    `json:"politica_sha256"`
	VersionEsperada      uint64    `json:"version_esperada"`
	ResueltaEn           time.Time `json:"resuelta_en"`
}

func (r ResolucionLlamamientoDesarrollo) referenciasValidas() bool {
	return ReferenciaOpacaLlamamientoValida(r.AperturaOperacionRef) &&
		ReferenciaOpacaLlamamientoValida(r.JustificanteRef) && ReferenciaOpacaLlamamientoValida(r.EvaluacionPlazoRef) &&
		ReferenciaOpacaLlamamientoValida(r.PoliticaRef) && r.PoliticaVersion > 0 && r.PoliticaVersion <= 1<<53-1 &&
		huellaSHA256LlamamientoValida(r.PoliticaSHA256) && r.PoliticaSHA256 != strings.Repeat("0", 64) && r.VersionEsperada == 1
}

func (r ResolucionLlamamientoDesarrollo) Validar() error {
	if !r.referenciasValidas() || !instanteCanonicoLlamamiento(r.ResueltaEn) {
		return ErrIntegracionLlamamientoDesarrollo
	}
	return nil
}

// PeticionResolverLlamamientoDesarrollo es interna, no un DTO HTTP. Su emisor
// debe reacreditar justificante, evaluación y política antes de cada llamada,
// también en replay. ResueltaEn debe venir vacía: la fija el servicio solo al
// primer intento y la conserva al recuperar. El consumidor durable debe ligar
// esos antecedentes y consumir autorización fresca junto al único terminal.
type PeticionResolverLlamamientoDesarrollo struct {
	OperacionRef string
	Resolucion   ResolucionLlamamientoDesarrollo
}

func (p PeticionResolverLlamamientoDesarrollo) Validar() error {
	if !ReferenciaOpacaLlamamientoValida(p.OperacionRef) || !p.Resolucion.referenciasValidas() ||
		p.OperacionRef == p.Resolucion.AperturaOperacionRef || !p.Resolucion.ResueltaEn.IsZero() {
		return ErrIntegracionLlamamientoDesarrollo
	}
	return nil
}

// RegistroLlamamientoDesarrollo conserva la instantánea completa y la propuesta
// del dominio, incluido su prefijo evaluado. Los agregados no salen hacia CT;
// el puente traduce exclusivamente sus referencias y seudonimiza la selección.
type RegistroLlamamientoDesarrollo struct {
	Esquema           string                           `json:"esquema"`
	OperacionRef      string                           `json:"operacion_ref"`
	Tipo              string                           `json:"tipo"`
	OrdenOperacionRef string                           `json:"orden_operacion_ref"`
	NecesidadRef      string                           `json:"necesidad_ref"`
	VersionNecesidad  uint64                           `json:"version_necesidad"`
	CategoriaRef      string                           `json:"categoria_ref"`
	UnidadRef         string                           `json:"unidad_ref"`
	Fuente            json.RawMessage                  `json:"fuente"`
	FirmaFuente       []byte                           `json:"firma_fuente"`
	Instantanea       domain.InstantaneaOrdenBolsa     `json:"instantanea"`
	Propuesta         *domain.PropuestaLlamamiento     `json:"propuesta,omitempty"`
	Llamamiento       *domain.DatosLlamamientoAbierto  `json:"llamamiento,omitempty"`
	EstadoLlamamiento domain.EstadoLlamamiento         `json:"estado_llamamiento,omitempty"`
	Resolucion        *ResolucionLlamamientoDesarrollo `json:"resolucion,omitempty"`
}

type ReciboLlamamientoDesarrollo struct {
	Registro     RegistroLlamamientoDesarrollo
	ReciboRef    string
	AuditoriaRef string
	EventoRef    string
	ConfirmadaEn time.Time
}

// Canonico valida forma y ligadura de la instantánea/propuesta, pero no sustituye
// ni la autenticación de la fuente ni la autorización VEC del efecto.
func (r RegistroLlamamientoDesarrollo) Canonico() ([]byte, error) {
	if r.Esquema != "vec.bolsa.integracion-llamamientos-desarrollo.v1" ||
		!ReferenciaOpacaLlamamientoValida(r.OperacionRef) || !ReferenciaOpacaLlamamientoValida(r.NecesidadRef) ||
		!ReferenciaOpacaLlamamientoValida(r.CategoriaRef) || !ReferenciaOpacaLlamamientoValida(r.UnidadRef) ||
		r.VersionNecesidad == 0 || r.VersionNecesidad > 1<<53-1 || len(r.Fuente) == 0 || len(r.Fuente) > 1024*1024 ||
		len(r.FirmaFuente) != 64 || r.Instantanea.Validar() != nil {
		return nil, ErrIntegracionLlamamientoDesarrollo
	}
	terminal := r.Tipo == "aceptacion_rrhh" || r.Tipo == "renuncia_rrhh"
	if !terminal && r.Resolucion != nil {
		return nil, ErrIntegracionLlamamientoDesarrollo
	}
	if r.Tipo == "orden" {
		if r.Propuesta != nil || r.Llamamiento != nil || r.EstadoLlamamiento != "" || r.OrdenOperacionRef != "" {
			return nil, ErrIntegracionLlamamientoDesarrollo
		}
	} else if r.Tipo == "propuesta" || terminal {
		if !ReferenciaOpacaLlamamientoValida(r.OrdenOperacionRef) || r.Propuesta == nil ||
			r.Propuesta.Validar() != nil || r.Propuesta.NecesidadRef != r.NecesidadRef || r.Propuesta.VersionNecesidad != r.VersionNecesidad ||
			r.Propuesta.InstantaneaRef != r.Instantanea.InstantaneaRef ||
			r.Propuesta.HuellaInstantaneaSHA256 != r.Instantanea.HuellaContenidoSHA256 {
			return nil, ErrIntegracionLlamamientoDesarrollo
		}
		if r.Llamamiento == nil ||
			r.Llamamiento.NecesidadRef != r.NecesidadRef || r.Llamamiento.PropuestaRef != r.Propuesta.PropuestaRef ||
			r.Llamamiento.BolsaRef != r.Instantanea.BolsaRef {
			return nil, ErrIntegracionLlamamientoDesarrollo
		}
		if r.Tipo == "propuesta" {
			if r.EstadoLlamamiento != domain.EstadoLlamamientoAbierto || r.Llamamiento.Version != 1 {
				return nil, ErrIntegracionLlamamientoDesarrollo
			}
		} else if r.Resolucion == nil || r.Resolucion.Validar() != nil ||
			r.Resolucion.AperturaOperacionRef == r.OperacionRef ||
			(r.Tipo == "aceptacion_rrhh" && r.EstadoLlamamiento != domain.EstadoLlamamientoAceptado) ||
			(r.Tipo == "renuncia_rrhh" && r.EstadoLlamamiento != domain.EstadoLlamamientoRenunciado) || r.Llamamiento.Version != 2 ||
			r.Resolucion.ResueltaEn.Before(r.Propuesta.GeneradaEn) || r.Resolucion.ResueltaEn.Before(r.Instantanea.GeneradaEn) {
			return nil, ErrIntegracionLlamamientoDesarrollo
		}
		if _, err := domain.NuevoLlamamientoAbierto(*r.Llamamiento); err != nil {
			return nil, err
		}
	} else {
		return nil, ErrIntegracionLlamamientoDesarrollo
	}
	return json.Marshal(r)
}

func (r RegistroLlamamientoDesarrollo) RecursoAutorizable() (dominiovec.RecursoAutorizable, error) {
	b, err := r.Canonico()
	if err != nil {
		return dominiovec.RecursoAutorizable{}, err
	}
	h := sha256.Sum256(b)
	return dominiovec.RecursoAutorizable{
		Referencia: r.OperacionRef, ModuloID: "bolsa", Tipo: "integracion_llamamientos_bolsa",
		Ambitos:   map[string]string{"categoria_ref": r.CategoriaRef, "unidad_ref": r.UnidadRef},
		Atributos: map[string]string{"contenido_sha256": hex.EncodeToString(h[:]), "necesidad_ref": r.NecesidadRef},
	}, nil
}

func (r RegistroLlamamientoDesarrollo) Accion() string {
	if r.Tipo == "orden" {
		return AccionPrepararOrdenDesarrollo
	}
	if r.Tipo == "propuesta" {
		return AccionAbrirLlamamientoDesarrollo
	}
	if r.Tipo == "aceptacion_rrhh" && r.EstadoLlamamiento == domain.EstadoLlamamientoAceptado {
		return AccionAceptarLlamamientoRRHHDesarrollo
	}
	if r.Tipo == "renuncia_rrhh" && r.EstadoLlamamiento == domain.EstadoLlamamientoRenunciado {
		return AccionRenunciarLlamamientoRRHHDesarrollo
	}
	return ""
}

// AutorizarOperacion recibe el recurso completo ligado al contenido que se va
// a guardar. La autoridad VEC y sus diez piezas nominales siguen siendo únicas.
type AutorizadorLlamamientoDesarrollo interface {
	AutorizarOperacion(context.Context, string, dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// La lectura es interna del proveedor. No acredita permiso para devolver el
// resultado: cada salida se reautoriza y Guardar consume la autorización viva
// en la misma transacción, también en replay. No exponer este puerto por HTTP.
type RepositorioLlamamientoDesarrollo interface {
	BuscarOperacion(context.Context, string) (RegistroLlamamientoDesarrollo, bool, error)
	Guardar(context.Context, RegistroLlamamientoDesarrollo, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboLlamamientoDesarrollo, error)
}
