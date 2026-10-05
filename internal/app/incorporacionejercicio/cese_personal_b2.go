package incorporacionejercicio

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"time"

	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	pp "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

// ReglaFechaCesePersonalB2 traduce la fecha de efectos del cese de CT al fin
// semiabierto [desde, hasta) de la relación en Personal. RRHH no ha dicho aún
// si la fecha de efectos es el último día trabajado o el primero ya sin
// relación (duda 140 de dudas.md), así que se elige en la configuración
// privada B2 y no tiene valor por defecto: sin ella el paso no se compone.
type ReglaFechaCesePersonalB2 string

const (
	// CeseUltimoDiaTrabajado: la fecha de efectos aún pertenece a la relación;
	// el fin semiabierto es el día siguiente.
	CeseUltimoDiaTrabajado ReglaFechaCesePersonalB2 = "ultimo_dia_trabajado"
	// CesePrimerDiaSinRelacion: la fecha de efectos ya queda fuera; es el fin.
	CesePrimerDiaSinRelacion ReglaFechaCesePersonalB2 = "primer_dia_sin_relacion"
)

func (r ReglaFechaCesePersonalB2) Valida() bool {
	return r == CeseUltimoDiaTrabajado || r == CesePrimerDiaSinRelacion
}

// Hasta devuelve el fin semiabierto de la relación para una fecha civil de
// efectos AAAA-MM-DD. No interpreta zonas horarias: opera sobre fechas civiles.
func (r ReglaFechaCesePersonalB2) Hasta(efecto string) (personal.FechaCivil, error) {
	if !r.Valida() {
		return "", ct.ErrComposicionIncorporacionAplicacion
	}
	f, err := time.Parse(time.DateOnly, efecto)
	if err != nil || f.Format(time.DateOnly) != efecto {
		return "", ct.ErrIntencionIncorporacionAplicacion
	}
	if r == CeseUltimoDiaTrabajado {
		f = f.AddDate(0, 0, 1)
	}
	hasta := personal.FechaCivil(f.Format(time.DateOnly))
	if hasta.Validar() != nil {
		return "", ct.ErrIntencionIncorporacionAplicacion
	}
	return hasta, nil
}

// LectorOrigenCesePersonalB2 lee de CT, con permiso actual, el origen B2 que
// liga el expediente con la relación exacta de Personal.
type LectorOrigenCesePersonalB2 interface {
	LeerOrigenIncorporacionB2(context.Context, string, string) (ct.OrigenIncorporacionPersonalB2, bool, error)
}

// ActosCesePersonalB2 es la autoridad única de actos B2 de Personal. El cese
// no abre otra vía de alta o baja: registra una revisión de la relación.
type ActosCesePersonalB2 interface {
	RegistrarHecho(context.Context, personal.SolicitudHechoEmpleadoB2) (pp.ResultadoHechoEmpleadoB2, error)
}

// Cada actor se resuelve en la frontera confiable para la operación concreta.
type ActoresCesePersonalB2 interface {
	ActorLecturaHechosB2(context.Context) (core.ContextoActor, error)
	ActorHechoB2(context.Context) (core.ContextoActor, error)
}

type ConfiguracionCesePersonalB2 struct {
	Contratos FuenteContratoPlanNominal
	Origen    LectorOrigenCesePersonalB2
	Ficha     pp.FuenteFichaIncorporacionCT
	Actos     ActosCesePersonalB2
	Actores   ActoresCesePersonalB2
	Reloj     ct.Reloj
	Fecha     ReglaFechaCesePersonalB2
}

// CesePersonalB2 es el paso simétrico a ConfirmarPersonalB2: tras un cese ya
// confirmado en CT, termina en Personal la relación que nació de ese
// expediente. Corre en una transacción distinta de la de CT y se recupera
// repitiendo la misma entrada; nunca da el fin por hecho sin recibo B2.
type CesePersonalB2 struct{ c ConfiguracionCesePersonalB2 }

func NuevoCesePersonalB2(c ConfiguracionCesePersonalB2) (*CesePersonalB2, error) {
	for _, d := range []any{c.Contratos, c.Origen, c.Ficha, c.Actos, c.Actores, c.Reloj} {
		if nuloPlanNominal(d) {
			return nil, ct.ErrComposicionIncorporacionAplicacion
		}
	}
	if !c.Fecha.Valida() {
		return nil, ct.ErrComposicionIncorporacionAplicacion
	}
	return &CesePersonalB2{c}, nil
}

// SolicitudCesePersonalB2 lleva el recibo durable del cese de CT, el
// justificante que CT ya validó y el recibo de la incorporación que CT asocia
// al expediente. Nada procede del cuerpo HTTP sin pasar por CT.
type SolicitudCesePersonalB2 struct {
	Recibo             ct.ReciboOperacionSeguimiento
	JustificanteRef    string
	JustificanteSHA256 string
	// IncorporacionReciboRef decide si aplica: solo un origen personal_b2_v1
	// (ct.ReciboOrigenIncorporacionPersonalB2) lleva a leer o escribir en B2.
	IncorporacionReciboRef string
}

type ResultadoCesePersonalB2 struct {
	// Aplica es falso cuando CT no incorporó el expediente por personal_b2_v1:
	// entonces no se ha leído nada con permisos B2 ni escrito en Personal.
	Aplica          bool
	IdempotenciaRef string
	RelacionRef     string
	VigenteHasta    personal.FechaCivil
	Recibo          pp.ReciboActoRegistroEmpleadoB2
}

func (c *CesePersonalB2) FinalizarRelacionPersonalB2(ctx context.Context, s SolicitudCesePersonalB2) (ResultadoCesePersonalB2, error) {
	var cero ResultadoCesePersonalB2
	if c == nil || ctx == nil {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	r := s.Recibo
	if !r.ValidoPara(ct.OperacionRegistrarCese, r.OrganizacionRef, r.ExpedienteRef, r.VersionAnterior) ||
		r.VersionResultante > math.MaxInt64 || !dom.ReferenciaOpacaValida(s.JustificanteRef) || !huellaPlanNominalValida(s.JustificanteSHA256) {
		return cero, ct.ErrIntencionIncorporacionAplicacion
	}
	if !ct.ReciboOrigenIncorporacionPersonalB2(s.IncorporacionReciboRef) {
		return cero, nil
	}
	contrato, err := c.c.Contratos.LeerContratoPlanNominal(ctx, r.OrganizacionRef, r.ExpedienteRef)
	if errors.Is(err, ct.ErrPlanNominalB2NoEncontrado) {
		// CT dice que la incorporación es B2: un plan ausente no es «no aplica».
		return cero, ct.ErrPreparacionIncorporacionPendiente
	}
	if err != nil {
		return cero, errorConsumidorPersonalB2(ctx, err)
	}
	if !contratoBasicoValido(contrato, r.OrganizacionRef, r.ExpedienteRef) || contrato.Protocolo != ProtocoloPersonalB2V1 || !contratoB2Valido(contrato) {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	origen, encontrado, err := c.c.Origen.LeerOrigenIncorporacionB2(ctx, r.OrganizacionRef, r.ExpedienteRef)
	if err != nil {
		return cero, errorConsumidorPersonalB2(ctx, err)
	}
	if !encontrado {
		return cero, ct.ErrPreparacionIncorporacionPendiente
	}
	h := origen.Confirmacion.Hechos
	if origen.ReciboRef != s.IncorporacionReciboRef || origen.Protocolo != ct.ProtocoloIncorporacionPersonalB2 || origen.Confirmacion.OrganizacionRef != r.OrganizacionRef ||
		origen.Confirmacion.ExpedienteRef != r.ExpedienteRef || !personal.ReferenciaEmpleadoValida(h.EmpleadoRef) ||
		!personal.ReferenciaRelacionValida(h.RelacionRef) || h.RelacionVersion < 1 {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	hasta, err := c.c.Fecha.Hasta(r.FechaEfecto)
	if err != nil {
		return cero, err
	}
	procedencia, err := procedenciaCesePersonalB2(s, h.RelacionRef)
	if err != nil {
		return cero, err
	}
	actual, previa, err := c.relacionActual(ctx, contrato, h)
	if err != nil {
		return cero, err
	}
	base := actual
	switch {
	case actual.Estado == "vigente":
	case actual.Estado == "finalizada" && previa != nil && previa.Estado == "vigente" && actual.Traza.Hasta == hasta &&
		actual.Traza.ActoRef == procedencia.ActoRef && actual.Traza.FuenteRef == procedencia.FuenteRef &&
		actual.Traza.FuenteVersion == procedencia.FuenteVersion:
		// Este mismo cese ya terminó la relación. Se repite la solicitud
		// original para obtener su recibo B2 por idempotencia, sin otro efecto.
		base = *previa
	default:
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	if !base.Traza.Desde.AntesDe(hasta) || (base.Traza.Hasta != "" && base.Traza.Hasta.AntesDe(hasta)) {
		// Un cese no adelanta el inicio ni alarga la relación prevista.
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	actor, err := c.c.Actores.ActorHechoB2(ctx)
	if err != nil {
		return cero, errorConsumidorPersonalB2(ctx, err)
	}
	if actor.Principal.ID != r.ActorRef {
		// El fin en Personal lo pide la misma persona de RRHH que registró el cese.
		return cero, ct.ErrDenegadaIncorporacionAplicacion
	}
	reg, mod := base.CatalogoSnapshot.Regimen, base.CatalogoSnapshot.Modalidad
	if reg == nil || mod == nil {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	solicitud := personal.SolicitudHechoEmpleadoB2{Tipo: "relacion", EmpleadoRef: h.EmpleadoRef, OrganismoRef: contrato.DatosPersonal.OrganismoRef,
		RelacionRef: h.RelacionRef, RelacionVersionEsperada: base.Traza.Version, RevisionEsperada: base.Traza.Version + 1,
		UnidadRef: base.UnidadRef, Regimen: personal.EntradaCatalogoEmpleadoB2{Ref: reg.Ref, Version: reg.Version},
		Modalidad: personal.EntradaCatalogoEmpleadoB2{Ref: mod.Ref, Version: mod.Version}, Estado: "finalizada",
		VigenteDesde: base.Traza.Desde, VigenteHasta: hasta, Procedencia: procedencia, Actor: actor}
	resultado, err := c.c.Actos.RegistrarHecho(ctx, solicitud)
	if err != nil {
		return cero, errorConsumidorPersonalB2(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	rec := resultado.Recibo
	if rec.Tipo != "relacion" || rec.EmpleadoRef != h.EmpleadoRef || rec.RelacionRef != h.RelacionRef ||
		rec.Version != solicitud.RevisionEsperada || rec.FirmaOficial || rec.EficaciaAdministrativa || !referenciaPlanB2(rec.ReciboRef) {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	return ResultadoCesePersonalB2{Aplica: true, IdempotenciaRef: procedencia.IdempotenciaRef, RelacionRef: h.RelacionRef,
		VigenteHasta: hasta, Recibo: rec}, nil
}

// relacionActual lee la ficha B2 vigente y devuelve la última revisión de la
// relación del origen y la anterior, si existe. Rechaza fichas de otra
// persona, empleado u organismo y relaciones que no aparecen en ella.
func (c *CesePersonalB2) relacionActual(ctx context.Context, contrato ContratoPlanNominal, h ct.HechosPersonalIncorporacionB2) (personal.RelacionRegistroEmpleadoB2, *personal.RelacionRegistroEmpleadoB2, error) {
	var cero personal.RelacionRegistroEmpleadoB2
	actor, err := c.c.Actores.ActorLecturaHechosB2(ctx)
	if err != nil {
		return cero, nil, errorConsumidorPersonalB2(ctx, err)
	}
	d := contrato.DatosPersonal
	corte := personal.CorteEmpleadoB2{VigenteEn: d.Desde, ConocidoEn: c.c.Reloj.Ahora().UTC().Truncate(time.Microsecond)}
	ficha, err := c.c.Ficha.ConsultarFicha(ctx, personal.SolicitudFichaEmpleadoB2{EmpleadoRef: h.EmpleadoRef, OrganismoRef: d.OrganismoRef, Corte: corte, Actor: actor})
	if err != nil {
		return cero, nil, errorConsumidorPersonalB2(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, nil, err
	}
	f := ficha.Ficha
	if f.EmpleadoRef != h.EmpleadoRef || f.PersonaRef != contrato.PersonaRef || f.OrganismoRef != d.OrganismoRef || f.Version < 1 {
		return cero, nil, ct.ErrConflictoIncorporacionAplicacion
	}
	var ultima, previa *personal.RelacionRegistroEmpleadoB2
	for i := range f.Relaciones {
		r := &f.Relaciones[i]
		if r.RelacionRef != h.RelacionRef {
			continue
		}
		if r.OrganismoRef != d.OrganismoRef {
			return cero, nil, ct.ErrConflictoIncorporacionAplicacion
		}
		if ultima == nil || r.Traza.Version > ultima.Traza.Version {
			ultima = r
		}
	}
	if ultima == nil || ultima.Traza.Version < int64(h.RelacionVersion) {
		return cero, nil, ct.ErrConflictoIncorporacionAplicacion
	}
	for i := range f.Relaciones {
		r := &f.Relaciones[i]
		if r.RelacionRef == h.RelacionRef && r.Traza.Version == ultima.Traza.Version-1 {
			previa = r
		}
	}
	return *ultima, previa, nil
}

// procedenciaCesePersonalB2 liga el acto de Personal al recibo durable del cese
// de CT. La clave de idempotencia es un UUID determinista: el mismo cese da
// siempre la misma clave (y por tanto el mismo recibo B2) y otro cese, otra.
func procedenciaCesePersonalB2(s SolicitudCesePersonalB2, relacionRef string) (personal.ProcedenciaActoEmpleadoB2, error) {
	r := s.Recibo
	huella, err := json.Marshal(struct {
		Esquema            string `json:"esquema"`
		Operacion          string `json:"operacion"`
		OrganizacionRef    string `json:"organizacion_ref"`
		ExpedienteRef      string `json:"expediente_ref"`
		VersionAnterior    uint64 `json:"version_anterior"`
		VersionResultante  uint64 `json:"version_resultante"`
		ReciboRef          string `json:"recibo_ref"`
		AuditoriaRef       string `json:"auditoria_ref"`
		EventoRef          string `json:"evento_ref"`
		ActorRef           string `json:"actor_ref"`
		RegistradaEn       string `json:"registrada_en"`
		CausaClave         string `json:"causa_clave"`
		FechaEfecto        string `json:"fecha_efecto"`
		JustificanteRef    string `json:"justificante_ref"`
		JustificanteSHA256 string `json:"justificante_sha256"`
	}{"vec.contratacion-temporal.cese.fuente-personal-b2.v1", r.Operacion, r.OrganizacionRef, r.ExpedienteRef, r.VersionAnterior,
		r.VersionResultante, r.ReciboRef, r.AuditoriaRef, r.EventoRef, r.ActorRef, r.RegistradaEn.UTC().Format(time.RFC3339Nano),
		string(r.CausaClave), r.FechaEfecto, s.JustificanteRef, s.JustificanteSHA256})
	if err != nil {
		return personal.ProcedenciaActoEmpleadoB2{}, ct.ErrIntencionIncorporacionAplicacion
	}
	suma := sha256.Sum256(huella)
	clave := sha256.Sum256([]byte("vec.personal-b2.cese-ct.idempotencia.v1\x00" + r.OrganizacionRef + "\x00" + r.ExpedienteRef + "\x00" + r.ReciboRef + "\x00" + relacionRef))
	p := personal.ProcedenciaActoEmpleadoB2{ActoRef: r.ReciboRef, FuenteRef: r.ReciboRef, FuenteVersion: int64(r.VersionResultante),
		FuenteHuellaSHA256: hex.EncodeToString(suma[:]), IdempotenciaRef: uuidDeterministaCeseB2(clave)}
	if p.Validar() != nil {
		return personal.ProcedenciaActoEmpleadoB2{}, ct.ErrIntencionIncorporacionAplicacion
	}
	return p, nil
}

// uuidDeterministaCeseB2 da forma de UUID v8 (RFC 9562) a los primeros 16
// bytes de la huella; no es aleatorio ni pretende serlo.
func uuidDeterministaCeseB2(h [32]byte) string {
	b := h[:16]
	b[6] = (b[6] & 0x0f) | 0x80
	b[8] = (b[8] & 0x3f) | 0x80
	x := hex.EncodeToString(b)
	return x[0:8] + "-" + x[8:12] + "-" + x[12:16] + "-" + x[16:20] + "-" + x[20:32]
}
