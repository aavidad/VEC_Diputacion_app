package application

import (
	"context"
	"reflect"

	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

// LectorConvocatoriaV3 compone los puertos centrales y el repositorio nominal
// de Bolsa. Obtener un ContextoActor no sustituye AutorizarConsultaConvocatoria.
type LectorConvocatoriaV3 struct {
	autorizador ports.ProveedorAutorizacionConsultaConvocatoria
	repositorio ports.RepositorioConsultaConvocatoria
}

func NuevoLectorConvocatoriaV3(a ports.ProveedorAutorizacionConsultaConvocatoria, r ports.RepositorioConsultaConvocatoria) (*LectorConvocatoriaV3, error) {
	if dependenciaConsultaNula(a) || dependenciaConsultaNula(r) {
		return nil, ports.ErrConvocatoriaNoDisponible
	}
	return &LectorConvocatoriaV3{a, r}, nil
}

func (l *LectorConvocatoriaV3) ConsultarExacta(ctx context.Context, s ports.SolicitudConsultaConvocatoria) (ports.LecturaConvocatoria, error) {
	var cero ports.LecturaConvocatoria
	if ctx == nil || l == nil || dependenciaConsultaNula(l.autorizador) || dependenciaConsultaNula(l.repositorio) {
		return cero, ports.ErrConvocatoriaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if _, err := PrepararConsultaConvocatoria(s); err != nil {
		return cero, err
	}
	a, err := l.autorizador.AutorizarConsultaConvocatoria(ctx, s)
	if err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	o := ports.OrdenConsultaConvocatoria{Solicitud: s, Autorizacion: a}
	if err := ValidarMaterialConsultaConvocatoria(o); err != nil {
		return cero, err
	}
	r, err := l.repositorio.ObtenerVersionExactaV3(ctx, o)
	if err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if ValidarResultadoVersionConvocatoria(o, r) != nil {
		return cero, ports.ErrRespuestaConvocatoriaInvalida
	}
	if r.Estado == "no_encontrada" {
		return cero, ports.ErrConvocatoriaNoEncontrada
	}
	v := r.Version
	cero = ports.LecturaConvocatoria{Ficha: ports.FichaConvocatoria{
		ConvocatoriaID: v.ID, Secuencia: v.Secuencia, Revision: v.Revision, FuenteRef: v.Referencia(), HuellaVersionSHA256: r.HuellaVersionSHA256,
		Bases: v.Configuracion.Documentos, Requisitos: v.Contenido.Requisitos, FlujoProceso: v.Configuracion.FlujoProceso, ReglasBaremacion: v.Configuracion.ReglasBaremacion,
		FasesEstado: "pendiente_fuente", ReferenciasCoberturaEstado: "pendiente_fuente"}, Evidencia: r.Evidencia}
	if !lecturaConvocatoriaValida(s, cero) {
		return ports.LecturaConvocatoria{}, ports.ErrRespuestaConvocatoriaInvalida
	}
	cero.Ficha.Bases = append(cero.Ficha.Bases[:0:0], cero.Ficha.Bases...)
	cero.Ficha.Requisitos = append(cero.Ficha.Requisitos[:0:0], cero.Ficha.Requisitos...)
	return cero, nil
}

// ValidarResultadoVersionConvocatoria se ejecuta antes del COMMIT de lectura.
// Una ausencia autorizada conserva evidencia sin devolver una versión parcial.
func ValidarResultadoVersionConvocatoria(o ports.OrdenConsultaConvocatoria, r ports.ResultadoVersionConvocatoria) error {
	if ValidarMaterialConsultaConvocatoria(o) != nil {
		return ports.ErrRespuestaConvocatoriaInvalida
	}
	e, x := r.Evidencia, o.Autorizacion.ResumenCapacidad()
	if e.DecisionRef != x.DecisionRef() || !referenciaConsultaValida(e.ReciboRef) || !referenciaConsultaValida(e.AuditoriaRef) ||
		!huellaConsultaValida(e.ConsumoHuellaSHA256) || e.CorrelacionRef != correlacionConvocatoria(o.Solicitud) ||
		e.ConsultadaEn.IsZero() || e.ConsultadaEn.Nanosecond()%1000 != 0 || e.ConsultadaEn.Before(x.EmitidaEn()) || !e.ConsultadaEn.Before(x.ExpiraEn()) {
		return ports.ErrRespuestaConvocatoriaInvalida
	}
	if _, zona := e.ConsultadaEn.Zone(); zona != 0 {
		return ports.ErrRespuestaConvocatoriaInvalida
	}
	if r.Estado == "no_encontrada" {
		if r.Version.ID == "" && r.HuellaVersionSHA256 == "" {
			return nil
		}
		return ports.ErrRespuestaConvocatoriaInvalida
	}
	v := r.Version
	h, err := v.HuellaSHA256()
	if r.Estado != "obtenida" || err != nil || h != r.HuellaVersionSHA256 || v.ID != o.Solicitud.Selector.ID || v.Secuencia != o.Solicitud.Selector.Secuencia {
		return ports.ErrRespuestaConvocatoriaInvalida
	}
	return nil
}

func dependenciaConsultaNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return r.IsNil()
	}
	return false
}

var _ ports.LectorConvocatoriaExacta = (*LectorConvocatoriaV3)(nil)
