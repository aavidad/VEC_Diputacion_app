package ports

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var (
	ErrConsultaComunicacionesExpedienteInvalida     = errors.New("contratacion temporal: consulta de comunicaciones invalida")
	ErrConsultaComunicacionesExpedienteDenegada     = errors.New("contratacion temporal: consulta de comunicaciones denegada")
	ErrConsultaComunicacionesExpedienteNoEncontrado = errors.New("contratacion temporal: expediente no encontrado")
	ErrConsultaComunicacionesExpedienteNoDisponible = errors.New("contratacion temporal: consulta de comunicaciones no disponible")
	ErrResultadoComunicacionesExpedienteNoConfiable = errors.New("contratacion temporal: resultado de comunicaciones no confiable")
)

const LimiteMaximoComunicacionesExpediente = 20
const EstadoSeleccionConfirmadaPendienteAviso = "seleccion_confirmada_pendiente_aviso"

var cursorComunicacionesExpediente = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{2,94}#[0-9a-f]{64}$`)
var cursorClaveComunicacionesExpediente = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{2,94}$`)

// El cliente identifica el expediente; organización e identidad proceden de
// la autoridad de canal, nunca de parámetros HTTP.
type ConsultaComunicacionesExpediente struct {
	ExpedienteRef string
	Limite        int
	Cursor        string
}

func (c ConsultaComunicacionesExpediente) Validar() error {
	if !domain.ReferenciaOpacaValida(c.ExpedienteRef) ||
		c.Limite < 1 || c.Limite > LimiteMaximoComunicacionesExpediente ||
		(c.Cursor != "" && !cursorComunicacionesExpediente.MatchString(c.Cursor)) {
		return ErrConsultaComunicacionesExpedienteInvalida
	}
	return nil
}

type ComunicacionExpediente struct {
	OrganizacionRef       string    `json:"organizacion_ref"`
	ExpedienteRef         string    `json:"expediente_ref"`
	LlamamientoRef        string    `json:"llamamiento_ref"`
	ComunicacionRef       string    `json:"comunicacion_ref,omitempty"`
	Version               uint64    `json:"version"`
	Estado                string    `json:"estado"`
	EstadoRespuesta       string    `json:"estado_respuesta,omitempty"`
	RegistradaEn          time.Time `json:"registrada_en"`
	ReciboComunicacionRef string    `json:"recibo_comunicacion_ref,omitempty"`
	AntecedenteTipo       string    `json:"antecedente_tipo"`
	ReciboAntecedenteRef  string    `json:"recibo_antecedente_ref"`
}

// CT140 serializa dos filas cerradas: la comunicación de siempre y la
// selección O6 confirmada que todavía espera el aviso CT54.
func (e *ComunicacionExpediente) UnmarshalJSON(datos []byte) error {
	if e == nil {
		return ErrResultadoComunicacionesExpedienteNoConfiable
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(datos, &campos) != nil || campos == nil {
		return ErrResultadoComunicacionesExpedienteNoConfiable
	}
	var estado string
	if json.Unmarshal(campos["estado"], &estado) != nil {
		return ErrResultadoComunicacionesExpedienteNoConfiable
	}
	esperados := []string{"organizacion_ref", "expediente_ref", "llamamiento_ref", "estado", "registrada_en", "version", "antecedente_tipo", "recibo_antecedente_ref"}
	if estado != EstadoSeleccionConfirmadaPendienteAviso {
		esperados = append(esperados, "comunicacion_ref", "estado_respuesta", "recibo_comunicacion_ref")
	}
	if len(campos) != len(esperados) {
		return ErrResultadoComunicacionesExpedienteNoConfiable
	}
	for _, campo := range esperados {
		if _, ok := campos[campo]; !ok {
			return ErrResultadoComunicacionesExpedienteNoConfiable
		}
	}
	type fila ComunicacionExpediente
	var valor fila
	if json.Unmarshal(datos, &valor) != nil {
		return ErrResultadoComunicacionesExpedienteNoConfiable
	}
	*e = ComunicacionExpediente(valor)
	return nil
}

func (e ComunicacionExpediente) claveOrden() string {
	if e.Estado == EstadoSeleccionConfirmadaPendienteAviso {
		return e.LlamamientoRef
	}
	return e.ComunicacionRef
}

type PaginaComunicacionesExpediente struct {
	ExpedienteRef   string                   `json:"expediente_ref"`
	Comunicaciones  []ComunicacionExpediente `json:"comunicaciones"`
	SiguienteCursor string                   `json:"siguiente_cursor,omitempty"`
}

func (p PaginaComunicacionesExpediente) ValidarPara(c ConsultaComunicacionesExpediente) error {
	if c.Validar() != nil || p.ExpedienteRef != c.ExpedienteRef ||
		p.Comunicaciones == nil || len(p.Comunicaciones) > c.Limite ||
		(p.SiguienteCursor != "" && (len(p.Comunicaciones) != c.Limite ||
			!cursorComunicacionesExpediente.MatchString(p.SiguienteCursor) ||
			!strings.HasPrefix(p.SiguienteCursor,
				p.Comunicaciones[len(p.Comunicaciones)-1].claveOrden()+"#"))) {
		return ErrResultadoComunicacionesExpedienteNoConfiable
	}
	var anterior time.Time
	var refAnterior, organizacion string
	pendientes := 0
	for i, e := range p.Comunicaciones {
		if e.Estado == EstadoSeleccionConfirmadaPendienteAviso {
			pendientes++
			if pendientes > 1 || !domain.ReferenciaOpacaValida(e.OrganizacionRef) ||
				e.ExpedienteRef != c.ExpedienteRef ||
				!domain.ReferenciaOpacaValida(e.LlamamientoRef) ||
				!cursorClaveComunicacionesExpediente.MatchString(e.LlamamientoRef) ||
				e.ComunicacionRef != "" || e.ReciboComunicacionRef != "" || e.EstadoRespuesta != "" ||
				e.Version != 1 || e.AntecedenteTipo != "seleccion_confirmada" ||
				!domain.ReferenciaOpacaValida(e.ReciboAntecedenteRef) ||
				!domain.InstanteUTCCanonico(e.RegistradaEn) ||
				(i > 0 && (e.RegistradaEn.Before(anterior) ||
					(e.RegistradaEn.Equal(anterior) && e.claveOrden() <= refAnterior))) ||
				(organizacion != "" && e.OrganizacionRef != organizacion) {
				return ErrResultadoComunicacionesExpedienteNoConfiable
			}
			organizacion, anterior, refAnterior = e.OrganizacionRef, e.RegistradaEn, e.claveOrden()
			continue
		}
		if !domain.ReferenciaOpacaValida(e.OrganizacionRef) ||
			e.ExpedienteRef != c.ExpedienteRef ||
			!domain.ReferenciaOpacaValida(e.LlamamientoRef) ||
			!domain.ReferenciaOpacaValida(e.ComunicacionRef) ||
			!domain.ReferenciaOpacaValida(e.ReciboComunicacionRef) ||
			!domain.ReferenciaOpacaValida(e.ReciboAntecedenteRef) ||
			(e.AntecedenteTipo != "seleccion_confirmada" && e.AntecedenteTipo != "continuacion_confirmada") ||
			e.Version != 2 || e.Estado != "registrada_localmente" ||
			(e.EstadoRespuesta != "sin_respuesta" && e.EstadoRespuesta != "registrada") ||
			!domain.InstanteUTCCanonico(e.RegistradaEn) ||
			(i > 0 && (e.RegistradaEn.Before(anterior) ||
				(e.RegistradaEn.Equal(anterior) && e.claveOrden() <= refAnterior))) ||
			(organizacion != "" && e.OrganizacionRef != organizacion) {
			return ErrResultadoComunicacionesExpedienteNoConfiable
		}
		organizacion, anterior, refAnterior = e.OrganizacionRef, e.RegistradaEn, e.claveOrden()
	}
	return nil
}

type LectorComunicacionesExpediente interface {
	ConsultarComunicacionesExpediente(context.Context, ConsultaComunicacionesExpediente) (PaginaComunicacionesExpediente, error)
}
