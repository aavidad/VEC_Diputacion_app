package lecturaincorporacion

import (
	"context"
	"reflect"
	"strings"
	"time"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	TipoRecursoV2 = "registro_alta_ejercicio_incorporacion_v2"
	AudienciaV2   = "vec_personal.lectura_incorporacion.v2"
)

// MaterialV2 añade la unidad de Preparacion CT validada por el servidor.
// No convierte unidades/centros ni acredita origen por aceptar un string.
// Selector e identidad originales no cambian; V1 no incorpora este ámbito.
type MaterialV2 struct {
	base      Material
	unidadRef string
	politica  *PoliticaConsultaV2
}

// PoliticaConsultaV2 es una copia de la política gobernada que aporta la
// composición. Sólo habilita la lectura con garantía sustancial en desarrollo;
// la concesión y el consumo V3 siguen siendo obligatorios.
type PoliticaConsultaV2 struct {
	Tipo         httpseguridad.PoliticaInterna
	Referencia   string
	HuellaSHA256 string
	RetiradaEn   time.Time
}

func (p PoliticaConsultaV2) validaEn(ahora time.Time) bool {
	if p.Tipo != httpseguridad.PoliticaInternaDesarrolloCertificadoPersonal ||
		!ctdomain.InstanteUTCCanonico(ahora) ||
		len(p.Referencia) < 26 || len(p.Referencia) > 132 || !strings.HasPrefix(p.Referencia, "pga_") ||
		!huellaSHA256.MatchString(p.HuellaSHA256) || strings.Trim(p.HuellaSHA256, "0") == "" ||
		p.RetiradaEn.Location() != time.UTC || p.RetiradaEn.Nanosecond() != 0 ||
		!p.RetiradaEn.After(ahora) {
		return false
	}
	for _, c := range p.Referencia[4:] {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func (p PoliticaConsultaV2) admite(v core.DatosVinculoAutenticacionActorV2, ahora time.Time) bool {
	return p.validaEn(ahora) && v.Superficie == core.SuperficieAutenticacionInternaCorporativaV1 &&
		v.MetodoObservado == core.AuthMethodCertificate && v.GarantiaObservada == core.AuthAssuranceSubstantial &&
		!v.CuentaPrivilegiada && v.PoliticaGarantiaRef == p.Referencia &&
		v.PoliticaGarantiaHuellaSHA256 == p.HuellaSHA256
}

func NuevoMaterialV2(selector Selector, unidadRef string, contexto ct.ContextoAutorizacionAltaV3, ahora time.Time) (MaterialV2, error) {
	if !unidadLecturaV2Valida(unidadRef) {
		return MaterialV2{}, ErrDenegada
	}
	base, err := NuevoMaterial(selector, contexto, ahora)
	if err != nil {
		return MaterialV2{}, err
	}
	return MaterialV2{base: base, unidadRef: unidadRef}, nil
}

// NuevoMaterialV2ConPolitica recibe la copia de composición, nunca valores de
// una petición. La política debe seguir vigente al validar y consumir la orden.
func NuevoMaterialV2ConPolitica(selector Selector, unidadRef string, contexto ct.ContextoAutorizacionAltaV3, ahora time.Time, politica PoliticaConsultaV2) (MaterialV2, error) {
	if !unidadLecturaV2Valida(unidadRef) || !selector.validar() || !ctdomain.InstanteUTCCanonico(ahora) || !politica.validaEn(ahora) {
		return MaterialV2{}, ErrDenegada
	}
	v, err := contexto.Vinculo.Datos()
	if err != nil || !politica.admite(v, ahora) ||
		contexto.ValidarPara(ct.SolicitudResolverContextoAutorizacionAltaV3{
			AutenticacionRef: v.AutenticacionRef, SesionRef: v.SesionRef, PerfilRef: v.PerfilActivoRef,
		}, ahora) != nil {
		return MaterialV2{}, ErrDenegada
	}
	r, err := contexto.Resultado.Clonar()
	if err != nil {
		return MaterialV2{}, ErrDenegada
	}
	p := politica
	return MaterialV2{base: Material{selector: selector, contexto: ct.ContextoAutorizacionAltaV3{
		Vinculo: contexto.Vinculo, Resultado: r,
	}, preparadoEn: ahora}, unidadRef: unidadRef, politica: &p}, nil
}

func unidadLecturaV2Valida(ref string) bool {
	// Se conserva la gramática legacy del lector. La unidad del expediente
	// se transporta literalmente, sin cambiar selector, contexto ni audiencia.
	return ctdomain.UnidadSeguimientoValida(ref) ||
		strings.HasPrefix(ref, "ref:") && huellaSHA256.MatchString(strings.TrimPrefix(ref, "ref:"))
}
func (m MaterialV2) Selector() Selector                               { return m.base.Selector() }
func (m MaterialV2) UnidadRef() string                                { return m.unidadRef }
func (m MaterialV2) PreparadoEn() time.Time                           { return m.base.PreparadoEn() }
func (m MaterialV2) Contexto() (ct.ContextoAutorizacionAltaV3, error) { return m.base.Contexto() }
func (m MaterialV2) Recurso() (core.RecursoAutorizable, error) {
	if !unidadLecturaV2Valida(m.unidadRef) {
		return core.RecursoAutorizable{}, ErrDenegada
	}
	r, err := m.base.Recurso()
	if err != nil {
		return core.RecursoAutorizable{}, err
	}
	r.Tipo = TipoRecursoV2
	r.Ambitos["unidad_ref"] = m.unidadRef
	if _, err := r.HuellaContextoAutorizacionSHA256(); err != nil {
		return core.RecursoAutorizable{}, ErrDenegada
	}
	return r, nil
}
func (m MaterialV2) HuellaSHA256() (string, error) {
	r, err := m.Recurso()
	if err != nil {
		return "", err
	}
	return r.HuellaContextoAutorizacionSHA256()
}

// AutorizacionV2 transporta los mismos tipos nominales, sin hacerlos convertibles
// en permiso V1: la validación exige recurso y audiencia V2 exactos.
type AutorizacionV2 Autorizacion
type ProveedorV2 interface {
	AutorizarLecturaIncorporacionV2(context.Context, MaterialV2) (AutorizacionV2, error)
}
type OrdenV2 struct {
	material     MaterialV2
	autorizacion AutorizacionV2
	evaluadaEn   time.Time
}

func (o OrdenV2) ValidarEn(ahora time.Time) error {
	if !ctdomain.InstanteUTCCanonico(o.evaluadaEn) || ahora.Before(o.evaluadaEn) {
		return ErrDenegada
	}
	return o.autorizacion.validar(o.material, ahora)
}
func (o OrdenV2) Material() MaterialV2  { return o.material }
func (o OrdenV2) Selector() Selector    { return o.material.Selector() }
func (o OrdenV2) UnidadRef() string     { return o.material.UnidadRef() }
func (o OrdenV2) EvaluadaEn() time.Time { return o.evaluadaEn }
func (o OrdenV2) Exportacion() vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	return o.autorizacion.Exportacion
}

// TransaccionLecturaV2 debe consumir exclusivamente V3 lector V2 fresco, cotejar
// selector+unidad+contexto exactos, leer Personal y auditar en un único commit.
// Es RW por consumo/auditoría, nunca emite otra alta. No delegar en fachada V1.
// El DTO devuelto no demuestra commit: sólo salir tras confirmación durable.
type TransaccionLecturaV2 interface {
	LeerRegistroPersonalV2(context.Context, Selector, OrdenV2) (Resultado, error)
}
type ConsumidorV2 struct {
	proveedor   ProveedorV2
	transaccion TransaccionLecturaV2
	reloj       Reloj
	politica    *PoliticaConsultaV2
}

func dependenciaV2Nula(v any) bool {
	if nilv(v) {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return x.IsNil()
	}
	return false
}

func NuevoV2(proveedor ProveedorV2, transaccion TransaccionLecturaV2, reloj Reloj) (*ConsumidorV2, error) {
	if dependenciaV2Nula(proveedor) || dependenciaV2Nula(transaccion) || dependenciaV2Nula(reloj) {
		return nil, ErrNoDisponible
	}
	return &ConsumidorV2{proveedor: proveedor, transaccion: transaccion, reloj: reloj}, nil
}

func NuevoV2ConPolitica(proveedor ProveedorV2, transaccion TransaccionLecturaV2, reloj Reloj, politica PoliticaConsultaV2) (*ConsumidorV2, error) {
	c, err := NuevoV2(proveedor, transaccion, reloj)
	if err != nil || !politica.validaEn(reloj.Ahora()) {
		return nil, ErrNoDisponible
	}
	p := politica
	c.politica = &p
	return c, nil
}
func (c *ConsumidorV2) Leer(ctx context.Context, selector Selector, unidadRef string, contexto ct.ContextoAutorizacionAltaV3) (Resultado, error) {
	var cero Resultado
	if c == nil || ctx == nil {
		return cero, ErrNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if dependenciaV2Nula(c.proveedor) || dependenciaV2Nula(c.transaccion) || dependenciaV2Nula(c.reloj) {
		return cero, ErrNoDisponible
	}
	antes := c.reloj.Ahora()
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	var m MaterialV2
	var err error
	if c.politica == nil {
		m, err = NuevoMaterialV2(selector, unidadRef, contexto, antes)
	} else {
		m, err = NuevoMaterialV2ConPolitica(selector, unidadRef, contexto, antes, *c.politica)
	}
	if err != nil {
		return cero, ErrDenegada
	}
	a, err := c.proveedor.AutorizarLecturaIncorporacionV2(ctx, m)
	if err != nil {
		return cero, opaco(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	evaluada := c.reloj.Ahora()
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	o := OrdenV2{material: m, autorizacion: a, evaluadaEn: evaluada}
	if o.ValidarEn(evaluada) != nil {
		return cero, ErrDenegada
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	r, err := c.transaccion.LeerRegistroPersonalV2(ctx, selector, o)
	if err != nil {
		return cero, opaco(ctx, err)
	}
	despues := c.reloj.Ahora()
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if despues.Before(evaluada) || o.ValidarEn(despues) != nil ||
		!resultadoParaLectura(r, selector, evaluada, m.PreparadoEn(), o.Exportacion(), despues) {
		return cero, ErrDenegada
	}
	return copiar(r), nil
}
