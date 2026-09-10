package ports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var ErrAcreditacionPersonalIncorporacion = errors.New("contratacion temporal: origen personal no acreditado")

// Cota del transporte de recibo propio Personal (adapters/postgres/alta_ejercicio).
// No sustituye las cotas independientes de cada pieza de la exportación V3.
const maximoMaterialRegistroPersonalEjercicio = 64 << 10

// RegistroPersonalEjercicio es transporte, NO autoridad. MaterialCanonico son
// los bytes originales de vec.personal.alta-ejercicio.material.v1 (no jsonb::text
// ni material reconstruido por CT). La API propietaria debe cotejar su esquema,
// solicitud, fuente, persona sintética, centro, período, RPT y actor originales.
// CT verifica el hash exacto; no duplica el codec/validador propiedad de Personal.
type RegistroPersonalEjercicio struct {
	Solicitud              SolicitudAltaPersonalRPT
	Resultado              ResultadoAltaPersonalRPT
	MaterialCanonico       []byte
	MaterialSHA256         string
	RegistradoEn           time.Time
	DecisionOriginalRef    string
	AuditoriaRef           string
	OutboxRef              string
	EjercicioSintetico     bool
	FirmaOficial           bool
	EficaciaAdministrativa bool
}

func (r RegistroPersonalEjercicio) copiar() RegistroPersonalEjercicio {
	r.MaterialCanonico = bytes.Clone(r.MaterialCanonico)
	return r
}

// ValidarEstructuraPara no demuestra origen. Ningún constructor público, SHA o
// estado confirmada puede acreditar que estos datos existan en un commit.
func (r RegistroPersonalEjercicio) ValidarEstructuraPara(s SolicitudAltaPersonalRPT, ahora time.Time) error {
	if r.Solicitud != s || s.Validar() != nil || r.Resultado.ValidarPara(s) != nil || r.Resultado.Estado != AltaPersonalRPTConfirmada ||
		len(r.MaterialCanonico) == 0 || len(r.MaterialCanonico) > maximoMaterialRegistroPersonalEjercicio ||
		!domain.InstanteUTCCanonico(ahora) || !domain.InstanteUTCCanonico(r.RegistradoEn) || r.RegistradoEn.After(ahora) ||
		!domain.ReferenciaOpacaValida(r.DecisionOriginalRef) || !domain.ReferenciaOpacaValida(r.AuditoriaRef) || !domain.ReferenciaOpacaValida(r.OutboxRef) ||
		!r.EjercicioSintetico || r.FirmaOficial || r.EficaciaAdministrativa {
		return ErrAcreditacionPersonalIncorporacion
	}
	h := sha256.Sum256(r.MaterialCanonico)
	if r.MaterialSHA256 != hex.EncodeToString(h[:]) {
		return ErrAcreditacionPersonalIncorporacion
	}
	return nil
}

func mismoRegistroPersonalEjercicio(a, b RegistroPersonalEjercicio) bool {
	return a.Solicitud == b.Solicitud && a.Resultado == b.Resultado && bytes.Equal(a.MaterialCanonico, b.MaterialCanonico) &&
		a.MaterialSHA256 == b.MaterialSHA256 && a.RegistradoEn.Equal(b.RegistradoEn) && a.DecisionOriginalRef == b.DecisionOriginalRef &&
		a.AuditoriaRef == b.AuditoriaRef && a.OutboxRef == b.OutboxRef && a.EjercicioSintetico == b.EjercicioSintetico &&
		a.FirmaOficial == b.FirmaOficial && a.EficaciaAdministrativa == b.EficaciaAdministrativa
}

// AcreditadorPersonalIncorporacion es autoridad inyectada desde composición,
// nunca un callback/browser. Implementación durable PENDIENTE: debe autorizar
// lectura propia de Personal ANTES de acceder (alta Personal y permiso de efecto
// CT NO son permiso lector), revalidarla en su transacción y resolver solicitud,
// material, recibo, audit y outbox del MISMO commit. Solo lectura, sin alta/replay
// de alta ni reserva. Error/cancelación devuelven registro cero. No grants SELECT
// a CT. El DTO esperado de la orden sirve para cotejo, nunca de fuente de verdad.
type AcreditadorPersonalIncorporacion interface {
	AcreditarRegistroPersonal(context.Context, OrdenConfirmacionIncorporacionV2) (RegistroPersonalEjercicio, error)
}

type RelojAcreditacionPersonalIncorporacion interface{ Ahora() time.Time }

// AcreditacionPersonalIncorporacion solo nace al consultar el puerto propietario
// con orden nominal vigente. Acredita el cotejo en esa frontera, no el commit de
// CT ni una firma de Personal. Un doble no prueba origen durable ni gobierno SQL.
type AcreditacionPersonalIncorporacion struct {
	orden        OrdenConfirmacionIncorporacionV2
	registro     RegistroPersonalEjercicio
	consultadaEn time.Time
}

func AcreditarPersonalParaIncorporacion(ctx context.Context, o OrdenConfirmacionIncorporacionV2, p AcreditadorPersonalIncorporacion, reloj RelojAcreditacionPersonalIncorporacion) (AcreditacionPersonalIncorporacion, error) {
	cero := AcreditacionPersonalIncorporacion{}
	if ctx == nil || p == nil || reloj == nil {
		return cero, ErrAcreditacionPersonalIncorporacion
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	antes := reloj.Ahora()
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if o.ValidarEn(antes) != nil {
		return cero, ErrAcreditacionPersonalIncorporacion
	}
	r, err := p.AcreditarRegistroPersonal(ctx, o)
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	if err != nil {
		return cero, ErrAcreditacionPersonalIncorporacion
	}
	despues := reloj.Ahora()
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	d, ed := o.material.Datos()
	if despues.Before(antes) || o.ValidarEn(despues) != nil || ed != nil ||
		r.ValidarEstructuraPara(d.Confirmacion.SolicitudPersonal, despues) != nil || !mismoRegistroPersonalEjercicio(r, d.Personal) {
		return cero, ErrAcreditacionPersonalIncorporacion
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	return AcreditacionPersonalIncorporacion{orden: o, registro: r.copiar(), consultadaEn: despues}, nil
}

// RegistroPara exige la misma orden nominal y ventana del cotejo. Para un nuevo
// permiso/reintento se consulta otra vez: esta copia no es un token reutilizable.
func (a AcreditacionPersonalIncorporacion) RegistroPara(o OrdenConfirmacionIncorporacionV2, ahora time.Time) (RegistroPersonalEjercicio, error) {
	h, e := o.material.HuellaSHA256()
	ha, ea := a.orden.material.HuellaSHA256()
	he, ee := o.Exportacion().HuellaConjuntoSHA256()
	hea, eea := a.orden.Exportacion().HuellaConjuntoSHA256()
	if e != nil || ea != nil || h != ha || ahora.Before(a.consultadaEn) || o.ValidarEn(ahora) != nil || a.orden.ValidarEn(ahora) != nil ||
		ee != nil || eea != nil || he != hea {
		return RegistroPersonalEjercicio{}, ErrAcreditacionPersonalIncorporacion
	}
	return a.registro.copiar(), nil
}
