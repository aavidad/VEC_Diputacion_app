package ports

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"
)

var (
	ErrInformeResultadoEjecucionInvalido       = errors.New("informe de resultado de ejecucion invalido")
	ErrRegistroResultadoEjecucionNoDisponible  = errors.New("registro de resultado de ejecucion no disponible")
	ErrRegistroResultadoEjecucionIndeterminado = errors.New("confirmacion del registro de resultado indeterminada")
	ErrInformeResultadoEjecucionEnConflicto    = errors.New("referencia de informe de resultado en conflicto")
)

type PerfilResultadoEjecucion string

const (
	ResultadoRutasDietas    PerfilResultadoEjecucion = "dietas_rutas"
	ResultadoBorradorDietas PerfilResultadoEjecucion = "dietas_borrador"
	ResultadoMarcajeCronos  PerfilResultadoEjecucion = "cronos_marcaje"
)

// DatosResultadoEjecucionAutorizada contiene sólo enlaces opacos a una concesión
// positiva ya durable. No acepta payload, errores libres ni instante del cliente.
// InformeRef debe conservarse al reintentar una confirmación ambigua del registro.
type DatosResultadoEjecucionAutorizada struct {
	InformeRef                              string
	PerfilConsumidor                        PerfilResultadoEjecucion
	DecisionRef, DecisionHuellaSHA256       string
	ContextoRef, ContextoHuellaSHA256       string
	ActorRef, PerfilRef, CorrelacionRef     string
	Accion, RecursoRef, RecursoHuellaSHA256 string
	Resultado, Etapa, Causa                 string
}

func (DatosResultadoEjecucionAutorizada) String() string         { return "[resultado ejecucion redactado]" }
func (d DatosResultadoEjecucionAutorizada) GoString() string     { return d.String() }
func (d DatosResultadoEjecucionAutorizada) LogValue() slog.Value { return slog.StringValue(d.String()) }

// InformeResultadoEjecucionAutorizada no es una capacidad ni una decisión PDP.
type InformeResultadoEjecucionAutorizada struct {
	datos DatosResultadoEjecucionAutorizada
}

func NuevoInformeResultadoEjecucionAutorizada(d DatosResultadoEjecucionAutorizada) (InformeResultadoEjecucionAutorizada, error) {
	if !datosResultadoEjecucionValidos(d) {
		return InformeResultadoEjecucionAutorizada{}, ErrInformeResultadoEjecucionInvalido
	}
	return InformeResultadoEjecucionAutorizada{datos: d}, nil
}
func (i InformeResultadoEjecucionAutorizada) Datos() (DatosResultadoEjecucionAutorizada, error) {
	if !datosResultadoEjecucionValidos(i.datos) {
		return DatosResultadoEjecucionAutorizada{}, ErrInformeResultadoEjecucionInvalido
	}
	return i.datos, nil
}
func (InformeResultadoEjecucionAutorizada) String() string     { return "[informe ejecucion redactado]" }
func (i InformeResultadoEjecucionAutorizada) GoString() string { return i.String() }
func (i InformeResultadoEjecucionAutorizada) LogValue() slog.Value {
	return slog.StringValue(i.String())
}

type ReciboResultadoEjecucionAutorizada struct {
	InformeRef, ReciboRef, HuellaSHA256 string
	ObservadoEn                         time.Time
	Repeticion                          bool
}

func (ReciboResultadoEjecucionAutorizada) String() string     { return "[recibo ejecucion redactado]" }
func (r ReciboResultadoEjecucionAutorizada) GoString() string { return r.String() }
func (r ReciboResultadoEjecucionAutorizada) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

// El adaptador usa un pool auditor segregado, una transacción nueva y confirma
// únicamente después de COMMIT. No llamar desde una transacción de negocio viva.
type RegistroResultadoEjecucionAutorizada interface {
	RegistrarResultadoEjecucionAutorizada(context.Context, InformeResultadoEjecucionAutorizada) (ReciboResultadoEjecucionAutorizada, error)
}

var (
	refInformeEjecucion      = regexp.MustCompile(`^inf_ejec_[0-9a-f]{32}$`)
	refDecisionEjecucion     = regexp.MustCompile(`^(decision:[0-9a-f]{32}|dec_[A-Za-z0-9_-]{16,128})$`)
	refContextoEjecucion     = regexp.MustCompile(`^rca_[A-Za-z0-9_-]{22,128}$`)
	refActorEjecucion        = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,128}$`)
	refPerfilEjecucion       = regexp.MustCompile(`^prf_[A-Za-z0-9_-]{22,128}$`)
	refCorrelacionEjecucion  = regexp.MustCompile(`^correlacion_[0-9a-f]{32}$`)
	huellaResultadoEjecucion = regexp.MustCompile(`^[0-9a-f]{64}$`)
	recursoBorradorEjecucion = regexp.MustCompile(`^dietas:borrador:[A-Za-z0-9_-]{16,128}$`)
	recursoMarcajeEjecucion  = regexp.MustCompile(`^marcaje:cronos:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$`)
)

func datosResultadoEjecucionValidos(d DatosResultadoEjecucionAutorizada) bool {
	if !refInformeEjecucion.MatchString(d.InformeRef) || !refDecisionEjecucion.MatchString(d.DecisionRef) || !refContextoEjecucion.MatchString(d.ContextoRef) || !refActorEjecucion.MatchString(d.ActorRef) || !refPerfilEjecucion.MatchString(d.PerfilRef) || !refCorrelacionEjecucion.MatchString(d.CorrelacionRef) {
		return false
	}
	for _, h := range []string{d.DecisionHuellaSHA256, d.ContextoHuellaSHA256, d.RecursoHuellaSHA256} {
		if !huellaResultadoEjecucion.MatchString(h) || h == strings.Repeat("0", 64) {
			return false
		}
	}
	if d.Resultado != "fallo_confirmado" && d.Resultado != "resultado_indeterminado" {
		return false
	}
	// Etapa y causa se catalogan conjuntamente; no se aceptan textos del proveedor.
	valida := false
	switch d.Etapa {
	case "preparacion":
		valida = d.Resultado == "fallo_confirmado" && (d.Causa == "material_no_disponible" || d.Causa == "material_invalido" || d.Causa == "contexto_cancelado")
	case "transaccion":
		valida = d.Resultado == "fallo_confirmado" && (d.Causa == "persistencia" || d.Causa == "conflicto" || d.Causa == "recibo_invalido" || d.Causa == "contexto_cancelado" || d.Causa == "consumo_rechazado")
	case "rollback":
		valida = d.Resultado == "resultado_indeterminado" && d.Causa == "rollback_no_confirmado"
	case "commit":
		valida = (d.Resultado == "resultado_indeterminado" && d.Causa == "commit_no_confirmado") || (d.Resultado == "fallo_confirmado" && d.Causa == "commit_revertido")
	case "entrega":
		valida = d.Resultado == "resultado_indeterminado" && (d.Causa == "respuesta_no_confirmada" || d.Causa == "recibo_invalido")
	}
	if !valida {
		return false
	}
	switch d.PerfilConsumidor {
	case ResultadoRutasDietas:
		return (d.Accion == "dietas.ruta.catalogo.consultar" && d.RecursoRef == "dietas:rutas:catalogo_rutas_dietas") || (d.Accion == "dietas.ruta.calculo.solicitar" && d.RecursoRef == "dietas:rutas:calculo_rutas_dietas")
	case ResultadoBorradorDietas:
		return ((d.Accion == "dietas.borrador.crear_propio" || d.Accion == "dietas.borrador.recuperar_propio") && recursoBorradorEjecucion.MatchString(d.RecursoRef)) || (d.Accion == "dietas.borrador.listar_propios" && d.RecursoRef == "dietas:borradores:propios")
	case ResultadoMarcajeCronos:
		return d.Accion == "cronos.marcaje.propio.registrar" && recursoMarcajeEjecucion.MatchString(d.RecursoRef)
	}
	return false
}
