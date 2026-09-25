package ports

import (
	"context"
	"errors"
)

// TipoBorradorRRHH identifica las representaciones preparatorias disponibles;
// no implica firma, aprobación ni eficacia del documento.
type TipoBorradorRRHH string

const (
	BorradorInformeDefinitivo  TipoBorradorRRHH = "informe_definitivo"
	BorradorResolucion         TipoBorradorRRHH = "resolucion"
	BorradorDiligencia         TipoBorradorRRHH = "diligencia"
	BorradorTomaPosesion       TipoBorradorRRHH = "toma_posesion"
	BorradorNotificacion       TipoBorradorRRHH = "notificacion"
	BorradorComunicacionCentro TipoBorradorRRHH = "comunicacion_centro"
	// Documentos de formalización y de cambios posteriores. Su texto, sus
	// modalidades y la actuación que exigen los fija el catálogo de plantillas.
	BorradorContratoLaboral          TipoBorradorRRHH = "contrato_laboral"
	BorradorNombramiento             TipoBorradorRRHH = "nombramiento"
	BorradorCese                     TipoBorradorRRHH = "cese"
	BorradorModificacionNombramiento TipoBorradorRRHH = "modificacion_nombramiento"
)

// ErrBorradorRRHHNoDisponible indica que el detalle autorizado no
// cumple las precondiciones del borrador; no equivale a denegar la consulta.
var ErrBorradorRRHHNoDisponible = errors.New("contratacion temporal: borrador RRHH no disponible")

// RenderizadorBorradorRRHH transforma únicamente el detalle reducido
// ya autorizado. No consulta otra fuente, concede permisos ni acredita firma.
type RenderizadorBorradorRRHH interface {
	RenderizarBorrador(context.Context, TipoBorradorRRHH, DetalleExpedienteRRHH) ([]byte, error)
}
