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

// Original conserva la referencia y SHA256 de la revisión exacta leída.
// El adaptador de composición obtiene estos comprobantes de las dependencias
// reales para la petición exacta. La existencia de un servicio, una opción de
// configuración o una concesión pasada no constituye una comprobación.
type DisponibilidadFirmaR5Verificada struct {
	Solicitud    SolicitudDisponibilidadFirmaR5
	Via          string
	Original     ComprobanteComponenteFirmaR5
	Verificador  ComprobanteComponenteFirmaR5
	Competencia  ComprobanteComponenteFirmaR5
	Perfil       ComprobanteComponenteFirmaR5
	Custodia     ComprobanteComponenteFirmaR5
	Registro     ComprobanteComponenteFirmaR5
	ComprobadaEn time.Time
	ValidaHasta  time.Time
}

type VerificadorDisponibilidadFirmaR5 interface {
	VerificarDisponibilidadFirmaR5(context.Context, SolicitudDisponibilidadFirmaR5) ([]DisponibilidadFirmaR5Verificada, error)
}
