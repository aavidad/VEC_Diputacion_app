package ports

import (
	"context"
	"errors"
	"time"
)

var (
	ErrPreflightFirmaR5Invalido     = errors.New("contratacion temporal: preflight firma R5 invalido")
	ErrPreflightFirmaR5NoDisponible = errors.New("contratacion temporal: preflight firma R5 no disponible")
	ErrPreflightFirmaR5NoConfiable  = errors.New("contratacion temporal: preflight firma R5 no confiable")
)

// Canal procede de la frontera autenticada. El cliente solo identifica el
// expediente, documento y revisión del original que desea preparar.
type SolicitudPreflightFirmaR5 struct {
	Canal           SolicitudConsultaCircuitoRRHH
	Documento       string
	OriginalRef     string
	OriginalVersion uint64
}

// La respuesta nunca contiene personas, cargos, certificados ni capacidades.
// Una vía disponible permite preparar la operación; el registro posterior
// vuelve a verificar original, firma, competencia y autorización nominal.
type ResultadoPreflightFirmaR5 struct {
	VersionExpediente uint64
	Documento         string
	CatalogoRef       string
	CatalogoHuella    string
	PasoPendiente     int
	OriginalRef       string
	OriginalVersion   uint64
	ViasDisponibles   []string
}

// V2 identifica el PDF de entrada del paso pendiente. El original raíz
// permanece separado. No transporta bytes, firmantes, cargos ni certificados.
// Sin paso pendiente los tres campos de entrada están vacíos (versión cero).
type ResultadoPreflightFirmaR5V2 struct {
	ResultadoPreflightFirmaR5
	EntradaDocumentoRef     string
	EntradaDocumentoVersion uint64
	EntradaDocumentoHuella  string
	EntradaContenido        []byte
}

// Esta petición interna liga cada comprobación a la identidad actual resuelta
// por VEC, a la cabeza histórica y al paso del catálogo. Nunca se serializa.
type SolicitudDisponibilidadFirmaR5 struct {
	Preflight        SolicitudPreflightFirmaR5
	ActorRef         string
	PerfilRef        string
	ContextoHuella   string
	CatalogoRef      string
	CatalogoHuella   string
	PasoRef          string
	PasoOrden        int
	HistoriaRevision uint64
	HistoriaHuella   string
	OriginalHuella   string
}

type ComprobanteComponenteFirmaR5 struct {
	Referencia   string
	HuellaSHA256 string
}

// PreparacionViaFirmaR5 sólo informa cómo preparar el paso en esta petición.
// Conserva el original real leído, identidad y cabeza exactas. No acredita
// salud del verificador, custodia futura ni autorización del acto de firma.
type PreparacionViaFirmaR5 struct {
	Solicitud   SolicitudDisponibilidadFirmaR5
	Via         string
	Original    ComprobanteComponenteFirmaR5
	PreparadaEn time.Time
	ValidaHasta time.Time
}

type PreparadorViasFirmaR5 interface {
	PrepararViasFirmaR5(context.Context, SolicitudDisponibilidadFirmaR5) ([]PreparacionViaFirmaR5, error)
}

// ResolutorAmbitosConsultaFirmaVec aporta los ámbitos de la asignación vigente
// del contexto registrado. No admite selección de unidad por el navegador.
type ResolutorAmbitosConsultaFirmaVec interface {
	ResolverAmbitosConsultaFirmaVec(context.Context) (organizacion, unidad string, err error)
}
