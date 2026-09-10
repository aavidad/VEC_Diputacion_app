// Package lecturaincorporacion define la frontera propietaria de lectura de
// Personal para preparar una incorporación CT. No concede ni ejecuta un alta.
package lecturaincorporacion

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"time"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	Accion      = "personal.alta_ejercicio.registro.consultar_incorporacion"
	Finalidad   = "preparar_confirmacion_incorporacion_ct"
	TipoRecurso = "registro_alta_ejercicio_incorporacion"
	Audiencia   = "vec_personal.lectura_incorporacion.v1"
)

var ErrNoDisponible = errors.New("personal: lectura de incorporación no disponible")
var ErrDenegada = errors.New("personal: lectura de incorporación denegada")
var huellaSHA256 = regexp.MustCompile(`^[a-f0-9]{64}$`)

// Selector es input tipado de servidor: no contiene actor, permiso ni datos web.
type Selector struct {
	OrganizacionRef, SolicitudRef, ExpedienteRef                       string
	VersionExpediente                                                  uint64
	ResultadoRef, ReciboRef, RelacionRef, OcupacionRef, MaterialSHA256 string
}

func (s Selector) validar() bool {
	return ctdomain.ReferenciaOpacaValida(s.OrganizacionRef) && ctdomain.ReferenciaOpacaValida(s.SolicitudRef) && ctdomain.ReferenciaOpacaValida(s.ExpedienteRef) && s.VersionExpediente > 0 && s.VersionExpediente <= 9_007_199_254_740_991 && ctdomain.ReferenciaOpacaValida(s.ResultadoRef) && ctdomain.ReferenciaOpacaValida(s.ReciboRef) && ctdomain.ReferenciaOpacaValida(s.RelacionRef) && ctdomain.ReferenciaOpacaValida(s.OcupacionRef) && huellaSHA256.MatchString(s.MaterialSHA256)
}

// Material es inmutable por copia y liga exactamente el selector al contexto V3.
type Material struct {
	selector    Selector
	contexto    ct.ContextoAutorizacionAltaV3
	preparadoEn time.Time
}

func NuevoMaterial(selector Selector, contexto ct.ContextoAutorizacionAltaV3, ahora time.Time) (Material, error) {
	if !selector.validar() || !ctdomain.InstanteUTCCanonico(ahora) {
		return Material{}, ErrDenegada
	}
	v, e := contexto.Vinculo.Datos()
	if e != nil || v.GarantiaObservada != core.AuthAssuranceHigh ||
		(v.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 && v.Superficie != core.SuperficieAutenticacionAdministracionPrivilegiadaV1) ||
		contexto.ValidarPara(ct.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: v.AutenticacionRef, SesionRef: v.SesionRef, PerfilRef: v.PerfilActivoRef}, ahora) != nil {
		return Material{}, ErrDenegada
	}
	r, err := contexto.Resultado.Clonar()
	if err != nil {
		return Material{}, ErrDenegada
	}
	return Material{selector: selector, contexto: ct.ContextoAutorizacionAltaV3{Vinculo: contexto.Vinculo, Resultado: r}, preparadoEn: ahora}, nil
}
func (m Material) Selector() Selector     { return m.selector }
func (m Material) PreparadoEn() time.Time { return m.preparadoEn }

// Contexto entrega una copia; no incorpora autoridad del canal ni permite mutar
// la orden retenida por el consumidor.
func (m Material) Contexto() (ct.ContextoAutorizacionAltaV3, error) {
	r, err := m.contexto.Resultado.Clonar()
	if err != nil {
		return ct.ContextoAutorizacionAltaV3{}, ErrDenegada
	}
	return ct.ContextoAutorizacionAltaV3{Vinculo: m.contexto.Vinculo, Resultado: r}, nil
}
func (m Material) Recurso() (core.RecursoAutorizable, error) {
	if !m.selector.validar() {
		return core.RecursoAutorizable{}, ErrDenegada
	}
	r := core.RecursoAutorizable{Referencia: m.selector.ResultadoRef, ModuloID: "personal", Tipo: TipoRecurso, Ambitos: map[string]string{"organizacion_ref": m.selector.OrganizacionRef}, Atributos: map[string]string{"solicitud_ref": m.selector.SolicitudRef, "expediente_ref": m.selector.ExpedienteRef, "version_expediente": strconv.FormatUint(m.selector.VersionExpediente, 10), "resultado_ref": m.selector.ResultadoRef, "recibo_ref": m.selector.ReciboRef, "relacion_ref": m.selector.RelacionRef, "ocupacion_ref": m.selector.OcupacionRef, "material_sha256": m.selector.MaterialSHA256, "tipo_validacion": "ejercicio_sintetico"}}
	if _, e := r.HuellaContextoAutorizacionSHA256(); e != nil {
		return core.RecursoAutorizable{}, ErrDenegada
	}
	return r, nil
}
func (m Material) HuellaSHA256() (string, error) {
	r, e := m.Recurso()
	if e != nil {
		return "", e
	}
	h, e := r.HuellaContextoAutorizacionSHA256()
	return h, e
}

type Autorizacion struct {
	Solicitud    core.SolicitudAutorizacionLigadaV3
	Decision     core.DecisionAutorizacionLigadaV3
	Confirmacion vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Exportacion  vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// Proveedor es de composición confiable: debe registrar concesión y verificar
// firma/confianza antes de exportar V3. Los DTO no acreditan esos efectos.
type Proveedor interface {
	AutorizarLecturaIncorporacion(context.Context, Material) (Autorizacion, error)
}
type Orden struct {
	material     Material
	autorizacion Autorizacion
	evaluadaEn   time.Time
}

func (o Orden) validar(ahora time.Time) error {
	if !ctdomain.InstanteUTCCanonico(o.evaluadaEn) || ahora.Before(o.evaluadaEn) {
		return ErrDenegada
	}
	return o.autorizacion.validar(o.material, ahora)
}
func (o Orden) ValidarEn(ahora time.Time) error { return o.validar(ahora) }
func (o Orden) Material() Material              { return o.material }
func (o Orden) EvaluadaEn() time.Time           { return o.evaluadaEn }
func (o Orden) Selector() Selector              { return o.material.Selector() }
func (o Orden) Exportacion() vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	return o.autorizacion.Exportacion
}

// Resultado añade evidencia de la lectura y conserva todos los campos originales.
type Resultado struct {
	Registro                                                     ct.RegistroPersonalEjercicio
	DecisionLecturaRef, ConsumoHuellaSHA256, AuditoriaLecturaRef string
	LeidaEn                                                      time.Time
}

// TransaccionLectura debe consumir V3 fresco y leer el registro propietario con
// auditoría en un único commit. No reemite altas; el DTO no prueba persistencia.
// No expone resultado antes de commit confirmado. No admite SELECT cruzado CT.
type TransaccionLectura interface {
	LeerRegistroPersonal(context.Context, Selector, Orden) (Resultado, error)
}
type Reloj interface{ Ahora() time.Time }
type Consumidor struct {
	proveedor   Proveedor
	transaccion TransaccionLectura
	reloj       Reloj
}

func Nuevo(proveedor Proveedor, transaccion TransaccionLectura, reloj Reloj) (*Consumidor, error) {
	if nilv(proveedor) || nilv(transaccion) || nilv(reloj) {
		return nil, ErrNoDisponible
	}
	return &Consumidor{proveedor, transaccion, reloj}, nil
}
func (c *Consumidor) Leer(ctx context.Context, selector Selector, contexto ct.ContextoAutorizacionAltaV3) (Resultado, error) {
	var cero Resultado
	if c == nil || ctx == nil {
		return cero, ErrNoDisponible
	}
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	if !selector.validar() || nilv(c.proveedor) || nilv(c.transaccion) || nilv(c.reloj) {
		return cero, ErrNoDisponible
	}
	antes := c.reloj.Ahora()
	m, e := NuevoMaterial(selector, contexto, antes)
	if e != nil {
		return cero, ErrDenegada
	}
	a, e := c.proveedor.AutorizarLecturaIncorporacion(ctx, m)
	if e != nil {
		return cero, opaco(ctx, e)
	}
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	evaluada := c.reloj.Ahora()
	o := Orden{material: m, autorizacion: a, evaluadaEn: evaluada}
	if o.validar(evaluada) != nil {
		return cero, ErrDenegada
	}
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	r, e := c.transaccion.LeerRegistroPersonal(ctx, selector, o)
	if e != nil {
		return cero, opaco(ctx, e)
	}
	despues := c.reloj.Ahora()
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if despues.Before(evaluada) || o.validar(despues) != nil || !resultadoParaOrden(r, o, despues) {
		return cero, ErrDenegada
	}
	return copiar(r), nil
}
func resultadoValido(r Resultado, s Selector, ahora time.Time) bool {
	return r.Registro.ValidarEstructuraPara(r.Registro.Solicitud, ahora) == nil && r.Registro.Solicitud.SolicitudRef == s.SolicitudRef && r.Registro.Solicitud.ExpedienteRef == s.ExpedienteRef && r.Registro.Solicitud.VersionExpediente == s.VersionExpediente && r.Registro.Resultado.ResultadoRef == s.ResultadoRef && r.Registro.Resultado.ReciboRef == s.ReciboRef && r.Registro.Resultado.RelacionRef == s.RelacionRef && r.Registro.Resultado.OcupacionRef == s.OcupacionRef && r.Registro.MaterialSHA256 == s.MaterialSHA256 && ctdomain.ReferenciaOpacaValida(r.DecisionLecturaRef) && huellaSHA256.MatchString(r.ConsumoHuellaSHA256) && ctdomain.ReferenciaOpacaValida(r.AuditoriaLecturaRef) && ctdomain.InstanteUTCCanonico(r.LeidaEn) && !r.LeidaEn.After(ahora)
}
func copiar(r Resultado) Resultado {
	r.Registro.MaterialCanonico = bytes.Clone(r.Registro.MaterialCanonico)
	return r
}
func nilv(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	return (x.Kind() == reflect.Pointer || x.Kind() == reflect.Interface) && x.IsNil()
}
func opaco(ctx context.Context, e error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(e, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(e, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return ErrDenegada
}
