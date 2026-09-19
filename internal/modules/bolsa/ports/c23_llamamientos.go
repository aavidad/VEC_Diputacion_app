package ports

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrLlamamientoOperativoInvalido     = errors.New("bolsa: llamamiento operativo invalido")
	ErrLlamamientoOperativoNoDisponible = errors.New("bolsa: llamamiento operativo no disponible")
	ErrLlamamientoOperativoConflicto    = errors.New("bolsa: llamamiento operativo en conflicto")
)

const (
	CanalCorreoLlamamiento           = "correo"
	ResultadoAceptadoLlamamiento     = "aceptado"
	ResultadoRenunciaLlamamiento     = "renuncia"
	ResultadoSinRespuestaLlamamiento = "sin_respuesta"
)

type AperturaLlamamientoOperativo struct {
	OperacionRef        string
	LlamamientoRef      string
	ParticipacionRef    string
	ActorRef            string
	Canal               string
	ComunicadoEn        time.Time
	PlazoRespuestaHasta time.Time
	Anotacion           string
}

func (a AperturaLlamamientoOperativo) Validar() error {
	if !ReferenciaOpacaLlamamientoValida(a.OperacionRef) || !ReferenciaOpacaLlamamientoValida(a.LlamamientoRef) ||
		!ReferenciaOpacaLlamamientoValida(a.ParticipacionRef) || !ReferenciaOpacaLlamamientoValida(a.ActorRef) ||
		a.OperacionRef == a.LlamamientoRef || a.Canal != CanalCorreoLlamamiento ||
		!instanteCanonicoLlamamiento(a.ComunicadoEn) || !instanteCanonicoLlamamiento(a.PlazoRespuestaHasta) ||
		!a.PlazoRespuestaHasta.After(a.ComunicadoEn) || len(a.Anotacion) > 512 || a.Anotacion != strings.TrimSpace(a.Anotacion) {
		return ErrLlamamientoOperativoInvalido
	}
	return nil
}

type ResultadoLlamamientoOperativo struct {
	OperacionRef   string
	LlamamientoRef string
	ActorRef       string
	Resultado      string
	RegistradoEn   time.Time
}

func (r ResultadoLlamamientoOperativo) Validar() error {
	if !ReferenciaOpacaLlamamientoValida(r.OperacionRef) || !ReferenciaOpacaLlamamientoValida(r.LlamamientoRef) ||
		!ReferenciaOpacaLlamamientoValida(r.ActorRef) || r.OperacionRef == r.LlamamientoRef ||
		(r.Resultado != ResultadoAceptadoLlamamiento && r.Resultado != ResultadoRenunciaLlamamiento && r.Resultado != ResultadoSinRespuestaLlamamiento) ||
		!instanteCanonicoLlamamiento(r.RegistradoEn) {
		return ErrLlamamientoOperativoInvalido
	}
	return nil
}

type ReciboLlamamientoOperativo struct {
	Reutilizado      bool
	LlamamientoRef   string
	ParticipacionRef string
	Estado           string
	ReciboRef        string
	ConfirmadoEn     time.Time
}

type ContactoLlamamientoOperativo struct {
	CandidatoRef string
	Canal        string
	Disponible   bool
}

// RepositorioLlamamientoOperativo confirma llamamiento, situación, recibo,
// auditoría y outbox en una sola transacción. Contactos devuelve solo el índice
// candidato/canal: la dirección pertenece a la autoridad de alta VEC.
type RepositorioLlamamientoOperativo interface {
	ContactosParticipacion(context.Context, string) ([]ContactoLlamamientoOperativo, error)
	AbrirLlamamiento(context.Context, AperturaLlamamientoOperativo) (ReciboLlamamientoOperativo, error)
	RegistrarResultadoLlamamiento(context.Context, ResultadoLlamamientoOperativo) (ReciboLlamamientoOperativo, error)
}
