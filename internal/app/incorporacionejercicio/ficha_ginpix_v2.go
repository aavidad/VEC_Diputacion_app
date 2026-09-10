package incorporacionejercicio

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/ginpixfichero"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var (
	ErrFichaGINPIXV2Denegada     = errors.New("incorporacion: ficha ginpix v2 denegada")
	ErrFichaGINPIXV2Conflicto    = errors.New("incorporacion: ficha ginpix v2 sin recibo confirmado")
	ErrFichaGINPIXV2NoDisponible = errors.New("incorporacion: ficha ginpix v2 no disponible")
)

// MaterialFichaGINPIXV2 conserva el mapeo y las coordenadas originales
// restauradas. Nunca deriva correlación o idempotencia de auditoría/outbox.
type MaterialFichaGINPIXV2 struct {
	Mapeo                domain.MapeoVersionadoGINPIX
	ProcedenciaModeloRef string
	CorrelacionRef       string
	IdempotenciaRef      string
}

type RecuperacionFichaGINPIXV2 struct {
	Recibo               ports.ReciboIncorporacionAplicacionV2
	ProcedenciaModeloRef string
	CorrelacionRef       string
	IdempotenciaRef      string
}

type FuenteFichaGINPIXV2 interface {
	RecuperarFichaGINPIXV2(context.Context, string) (RecuperacionFichaGINPIXV2, error)
}

func (m MaterialFichaGINPIXV2) valido() bool {
	return m.Mapeo.Validar() == nil && domain.ReferenciaOpacaValida(m.ProcedenciaModeloRef) &&
		domain.ReferenciaOpacaValida(m.CorrelacionRef) && domain.ReferenciaOpacaValida(m.IdempotenciaRef)
}

// ProveedorMapeoFichaGINPIXV2 aporta únicamente el mapeo sintético vigente.
// Las coordenadas pertenecen a la recuperación durable, no al proveedor.
type ProveedorMapeoFichaGINPIXV2 interface {
	ResolverMapeoFichaGINPIXV2(context.Context, ports.ReciboIncorporacionAplicacionV2) (domain.MapeoVersionadoGINPIX, error)
}

type FichaGINPIXV2 struct {
	consulta FuenteFichaGINPIXV2
	mapeos   ProveedorMapeoFichaGINPIXV2
}

func NuevaFichaGINPIXV2(consulta FuenteFichaGINPIXV2, mapeos ProveedorMapeoFichaGINPIXV2) (*FichaGINPIXV2, error) {
	if nulo(consulta) || nulo(mapeos) {
		return nil, ErrFichaGINPIXV2NoDisponible
	}
	return &FichaGINPIXV2{consulta: consulta, mapeos: mapeos}, nil
}

// Preparar recupera exclusivamente la proyección V2 ya autorizada. No crea
// Orden/Recibo V1, no envía y falla cerrado si no existe recibo confirmado.
func (f *FichaGINPIXV2) Preparar(ctx context.Context, expediente string) (ginpixfichero.PreparacionExportacion, error) {
	if f == nil || ctx == nil || nulo(f.consulta) || nulo(f.mapeos) || !domain.ReferenciaOpacaValida(expediente) {
		return ginpixfichero.PreparacionExportacion{}, ErrFichaGINPIXV2NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ginpixfichero.PreparacionExportacion{}, ErrFichaGINPIXV2NoDisponible
	}
	recuperada, err := f.consulta.RecuperarFichaGINPIXV2(ctx, expediente)
	if ctx.Err() != nil {
		return ginpixfichero.PreparacionExportacion{}, ErrFichaGINPIXV2NoDisponible
	}
	if err != nil {
		return ginpixfichero.PreparacionExportacion{}, clasificarFichaGINPIXV2(ctx, err)
	}
	r := recuperada.Recibo
	if r == (ports.ReciboIncorporacionAplicacionV2{}) {
		return ginpixfichero.PreparacionExportacion{}, ErrFichaGINPIXV2Conflicto
	}
	if !reciboFichaGINPIXV2Valido(r, expediente) || !domain.ReferenciaOpacaValida(recuperada.ProcedenciaModeloRef) || !domain.ReferenciaOpacaValida(recuperada.CorrelacionRef) || !domain.ReferenciaOpacaValida(recuperada.IdempotenciaRef) {
		return ginpixfichero.PreparacionExportacion{}, ErrFichaGINPIXV2NoDisponible
	}
	mapeo, err := f.mapeos.ResolverMapeoFichaGINPIXV2(ctx, r)
	if ctx.Err() != nil {
		return ginpixfichero.PreparacionExportacion{}, ErrFichaGINPIXV2NoDisponible
	}
	material := MaterialFichaGINPIXV2{Mapeo: mapeo, ProcedenciaModeloRef: recuperada.ProcedenciaModeloRef, CorrelacionRef: recuperada.CorrelacionRef, IdempotenciaRef: recuperada.IdempotenciaRef}
	if err != nil || !material.valido() {
		return ginpixfichero.PreparacionExportacion{}, clasificarFichaGINPIXV2(ctx, err)
	}
	modelo, err := modeloFichaGINPIXV2(r, material)
	if err != nil {
		return ginpixfichero.PreparacionExportacion{}, ErrFichaGINPIXV2NoDisponible
	}
	carga, err := domain.AplicarMapeoGINPIX(modelo, material.Mapeo)
	if err != nil {
		return ginpixfichero.PreparacionExportacion{}, ErrFichaGINPIXV2NoDisponible
	}
	out, err := ginpixfichero.PrepararExportacion(carga)
	if err != nil {
		return ginpixfichero.PreparacionExportacion{}, ErrFichaGINPIXV2NoDisponible
	}
	metadatos, err := out.Metadatos()
	if ctx.Err() != nil || err != nil || metadatos.VersionExpediente != r.VersionActualExpediente || metadatos.ExpedienteRef != r.ExpedienteRef || metadatos.IncorporacionRef != r.ActuacionRef || metadatos.ProcedenciaModeloRef != material.ProcedenciaModeloRef || metadatos.CorrelacionRef != material.CorrelacionRef || metadatos.IdempotenciaRef != material.IdempotenciaRef {
		return ginpixfichero.PreparacionExportacion{}, ErrFichaGINPIXV2NoDisponible
	}
	return out, nil
}

func clasificarFichaGINPIXV2(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ErrFichaGINPIXV2NoDisponible
	}
	if errors.Is(err, ports.ErrDenegadaIncorporacionAplicacion) || errors.Is(err, ports.ErrAutorizacionDenegada) {
		return ErrFichaGINPIXV2Denegada
	}
	if errors.Is(err, ports.ErrConflictoIncorporacionAplicacion) {
		return ErrFichaGINPIXV2Conflicto
	}
	return ErrFichaGINPIXV2NoDisponible
}

func reciboFichaGINPIXV2Valido(r ports.ReciboIncorporacionAplicacionV2, expediente string) bool {
	return reciboVisibleValido(r, expediente, r.RegistradaEn)
}

func modeloFichaGINPIXV2(r ports.ReciboIncorporacionAplicacionV2, material MaterialFichaGINPIXV2) (domain.ModeloCanonicoGINPIX, error) {
	datos := []struct {
		k domain.ClaveCatalogo
		v string
	}{
		{"actuacion_ref", r.ActuacionRef}, {"expediente_ref", r.ExpedienteRef}, {"recibo_ref", r.ReciboRef}, {"relacion_ref", r.RelacionRef}, {"seguimiento_ref", r.SeguimientoRef}, {"solicitud_personal_ref", r.SolicitudPersonalRef},
	}
	campos := make([]domain.DatoCanonicoGINPIX, 0, len(datos))
	for _, d := range datos {
		c, err := domain.CampoValorGINPIX(d.v)
		if err != nil {
			return domain.ModeloCanonicoGINPIX{}, err
		}
		campos = append(campos, domain.DatoCanonicoGINPIX{Clave: d.k, Campo: c})
	}
	return domain.NuevoModeloCanonicoGINPIX(domain.BorradorModeloCanonicoGINPIX{Esquema: domain.EsquemaModeloCanonicoGINPIXV1, VersionExpediente: r.VersionActualExpediente, ExpedienteRef: r.ExpedienteRef, IncorporacionRef: r.ActuacionRef, ProcedenciaRef: material.ProcedenciaModeloRef, CorrelacionRef: material.CorrelacionRef, IdempotenciaRef: material.IdempotenciaRef, Datos: campos})
}

// RecuperarFichaGINPIXV2 reutiliza la misma lectura durable y su revalidación
// final; no llama a Consultar ni abre un segundo recorrido SQL.
func (p *PeticionV2PostgreSQL) RecuperarFichaGINPIXV2(ctx context.Context, exp string) (RecuperacionFichaGINPIXV2, error) {
	var z RecuperacionFichaGINPIXV2
	if p == nil || p.preparador == nil || ctx == nil || !domain.ReferenciaOpacaValida(exp) {
		return z, ports.ErrComposicionIncorporacionAplicacion
	}
	if err := ctx.Err(); err != nil {
		return z, err
	}
	l, err := p.preparador.leer(ctx, exp, 0)
	if err != nil {
		return z, err
	}
	if l.recibo == nil {
		return z, ports.ErrConflictoIncorporacionAplicacion
	}
	t, err := p.preparador.finalizar(ctx, l.ultimo)
	if err != nil {
		return z, err
	}
	r := *l.recibo
	if !reciboVisibleValido(r, exp, t) || l.preparacion.SolicitudPersonal.ExpedienteRef != exp || !domain.ReferenciaOpacaValida(l.preparacion.SolicitudPersonal.CorrelacionRef) || !domain.ReferenciaOpacaValida(l.preparacion.SolicitudPersonal.IdempotenciaRef) {
		return z, ports.ErrComposicionIncorporacionAplicacion
	}
	return RecuperacionFichaGINPIXV2{Recibo: r, ProcedenciaModeloRef: r.ReciboRef, CorrelacionRef: l.preparacion.SolicitudPersonal.CorrelacionRef, IdempotenciaRef: l.preparacion.SolicitudPersonal.IdempotenciaRef}, nil
}
