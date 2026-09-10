package ports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	core "vec-diputacion-granada/internal/vec/domain"
)

const EsquemaReciboRegistroIncorporacionV2 = "vec.contratacion-temporal.incorporacion.ejercicio.recibo.v2"

var ErrRegistroIncorporacionV2 = errors.New("contratacion temporal: registro de incorporacion V2 no confiable")

// ValidarMaterialRegistroIncorporacionV2 no renueva la preparación ni el contexto.
func ValidarMaterialRegistroIncorporacionV2(m MaterialConfirmacionIncorporacionV2, ahora time.Time) error {
	return m.validarEn(ahora)
}

// IntencionRegistroIncorporacionV2 compromete TODO el negocio normalizado.
// Solo excluye trazas de autorización renovables: autenticación/sesión/contexto,
// versiones de su instantánea, decisión/capacidad y correlación V3. Preserva la
// correlación LOCAL de preparación, identidad/cuenta/perfil/superficie/autoridad,
// motivo V3 completo, versiones independientes y bytes originales de Personal.
// No sustituye el recurso V2 completo: cada intento se autoriza con SU contexto.
// La clave de idempotencia de SolicitudPersonal identifica el intento de negocio;
// una colisión con intención distinta debe ser conflicto, nunca otro efecto.
func IntencionRegistroIncorporacionV2(m MaterialConfirmacionIncorporacionV2) ([]byte, error) {
	d, err := m.Datos()
	if err != nil {
		return nil, ErrRegistroIncorporacionV2
	}
	v, err := d.Contexto.Vinculo.Datos()
	if err != nil {
		return nil, ErrRegistroIncorporacionV2
	}
	identidad := struct {
		Principal, Perfil, Cuenta, CuentaOrdinaria string
		Privilegiada                               bool
		Superficie                                 core.SuperficieAutenticacionActorV1
		Autoridad                                  core.AutoridadProcedenciaContextoActorV1
	}{v.PrincipalID, v.PerfilActivoRef, v.CuentaRef, v.CuentaOrdinariaRef, v.CuentaPrivilegiada, v.Superficie, v.AutoridadEfectiva}
	return json.Marshal(struct {
		Esquema                                                  string
		Confirmacion                                             DatosConfirmacionIncorporacion
		VersionActualExpediente                                  uint64
		Preparacion                                              PreparacionSeguimientoConfirmacionIncorporacion
		Identidad                                                any
		MotivoV3                                                 core.ReferenciaEntradaCatalogo
		Personal                                                 RegistroPersonalEjercicio
		EjercicioSintetico, FirmaOficial, EficaciaAdministrativa bool
	}{"vec.contratacion-temporal.incorporacion.ejercicio.intencion.v2", d.Confirmacion, d.VersionActualExpediente, d.Preparacion, identidad, d.MotivoV3, d.Personal, d.EjercicioSintetico, d.FirmaOficial, d.EficaciaAdministrativa})
}

// ConsumosRegistroIncorporacionV2 es evidencia devuelta por la TX, NO permisos.
// HuellaExportacionCT liga exactamente el material AD3 enviado; ConsumoCTSHA256
// identifica el consumo SQL (no son la misma huella). Los enlaces lectores se
// cotejan con el resultado de AD3-28/Personal DENTRO de la TX, nunca con JSON del
// canal. Este DTO no acredita por sí mismo esa verificación durable.
type ConsumosRegistroIncorporacionV2 struct {
	DecisionCTRef, HuellaExportacionCT, ConsumoCTSHA256, AuditoriaAD3CTRef                        string
	ConsumidaCTEn                                                                                 time.Time
	DecisionLecturaRef, ConsumoLecturaSHA256, AuditoriaAD3LecturaRef, AuditoriaPersonalLecturaRef string
	LeidaPersonalEn                                                                               time.Time
}

func (c ConsumosRegistroIncorporacionV2) validar(o OrdenConfirmacionIncorporacionV2, ahora time.Time) error {
	x := o.Exportacion().ResumenCapacidad()
	h, e := o.Exportacion().HuellaConjuntoSHA256()
	d, ed := o.Material().Datos()
	if e != nil || ed != nil || o.ValidarEn(c.ConsumidaCTEn) != nil || o.ValidarEn(c.LeidaPersonalEn) != nil ||
		!domain.InstanteUTCCanonico(ahora) || c.ConsumidaCTEn.Before(o.evaluadaEn) || c.LeidaPersonalEn.Before(c.ConsumidaCTEn) || c.LeidaPersonalEn.After(ahora) ||
		c.DecisionCTRef != x.DecisionRef() || c.HuellaExportacionCT != h ||
		!shaRegistroV2(c.ConsumoCTSHA256) || !shaRegistroV2(c.ConsumoLecturaSHA256) ||
		c.DecisionLecturaRef == c.DecisionCTRef || c.DecisionLecturaRef == d.Personal.DecisionOriginalRef ||
		c.DecisionCTRef == d.Personal.DecisionOriginalRef || c.ConsumoCTSHA256 == c.ConsumoLecturaSHA256 {
		return ErrRegistroIncorporacionV2
	}
	for _, ref := range []string{c.DecisionLecturaRef, c.AuditoriaAD3CTRef, c.AuditoriaAD3LecturaRef, c.AuditoriaPersonalLecturaRef} {
		if !domain.ReferenciaOpacaValida(ref) {
			return ErrRegistroIncorporacionV2
		}
	}
	if c.AuditoriaAD3CTRef == c.AuditoriaAD3LecturaRef || c.AuditoriaPersonalLecturaRef == d.Personal.AuditoriaRef {
		return ErrRegistroIncorporacionV2
	}
	return nil
}

// ReciboRegistroIncorporacionV2 conserva el material original (incluido contexto
// y Personal) y las versiones por separado. No hay versión expediente resultante:
// este contrato NO autoriza avanzar expediente ni efectos jurídicos.
type ReciboRegistroIncorporacionV2 struct {
	Esquema                                                             string
	MaterialOriginalCanonico                                            []byte
	MaterialOriginalSHA256, IntencionSHA256, ContextoOriginalRef        string
	VersionSolicitudPersonal, VersionActualExpediente                   uint64
	SeguimientoRef                                                      string
	DefinicionSeguimiento                                               domain.ReferenciaDefinicionSeguimiento
	HuellaRaizSeguimiento, HuellaEstadoAnterior, HuellaEstadoResultante string
	VersionSeguimientoAnterior, VersionSeguimientoResultante            uint64
	Transicion                                                          domain.DatosTransicionSeguimiento
	AuditoriaCTRef, OutboxCTRef                                         string
	ConsumosOriginales                                                  ConsumosRegistroIncorporacionV2
	EjercicioSintetico, FirmaOficial, EficaciaAdministrativa            bool
}

func (r ReciboRegistroIncorporacionV2) Copia() ReciboRegistroIncorporacionV2 {
	r.MaterialOriginalCanonico = bytes.Clone(r.MaterialOriginalCanonico)
	r.Transicion.Documentos = append([]domain.DocumentoSeguimiento(nil), r.Transicion.Documentos...)
	if r.Transicion.Periodo != nil {
		p := *r.Transicion.Periodo
		r.Transicion.Periodo = &p
	}
	if r.Transicion.Calendario != nil {
		p := *r.Transicion.Calendario
		r.Transicion.Calendario = &p
	}
	return r
}

// ValidarPara se evalúa en la fecha ORIGINAL del recibo. Una autorización fresca
// no puede reetiquetar historia: la recuperación usa Resultado.ValidarPara.
func (r ReciboRegistroIncorporacionV2) ValidarPara(o OrdenConfirmacionIncorporacionV2) error {
	t := r.Transicion
	if o.ValidarEn(t.RegistradaEn) != nil {
		return ErrRegistroIncorporacionV2
	}
	d, e := o.Material().Datos()
	canon, ec := o.Material().MaterialCanonico()
	intencion, ei := IntencionRegistroIncorporacionV2(o.Material())
	if e != nil || ec != nil || ei != nil || r.Esquema != EsquemaReciboRegistroIncorporacionV2 || r.DefinicionSeguimiento.Validar() != nil ||
		!bytes.Equal(r.MaterialOriginalCanonico, canon) || r.MaterialOriginalSHA256 != hashRegistroV2(canon) || r.IntencionSHA256 != hashRegistroV2(intencion) ||
		r.ContextoOriginalRef != d.Contexto.Resultado.RegistroContextoRef || r.VersionSolicitudPersonal != d.Confirmacion.SolicitudPersonal.VersionExpediente ||
		r.VersionActualExpediente != d.VersionActualExpediente || r.VersionSeguimientoAnterior != d.Confirmacion.VersionSeguimientoEsperada ||
		!shaRegistroV2(r.HuellaRaizSeguimiento) || !shaRegistroV2(r.HuellaEstadoAnterior) || !shaRegistroV2(r.HuellaEstadoResultante) ||
		r.VersionSeguimientoAnterior >= MaximoEnteroSeguroOperacionAnalisis || r.VersionSeguimientoResultante != r.VersionSeguimientoAnterior+1 ||
		!r.EjercicioSintetico || r.FirmaOficial || r.EficaciaAdministrativa || t.Calendario != nil || t.RectificaActuacionRef != "" ||
		t.TransicionClave != TransicionConfirmarIncorporacion || t.MotivoClave != d.Confirmacion.MotivoClave ||
		t.ActorRef != d.Preparacion.ActorRef || t.UnidadRef != d.Preparacion.UnidadRef || t.CorrelacionRef != d.Preparacion.CorrelacionRef ||
		!t.EfectivoEn.Equal(d.Confirmacion.PeriodoIncorporacion.Desde) || t.Periodo == nil || *t.Periodo != d.Confirmacion.PeriodoIncorporacion ||
		!jsonIgualRegistroV2(t.Documentos, d.Confirmacion.Documentos) || r.ConsumosOriginales.validar(o, t.RegistradaEn) != nil {
		return ErrRegistroIncorporacionV2
	}
	for _, ref := range []string{r.SeguimientoRef, t.ActuacionRef, t.ReciboRef, r.AuditoriaCTRef, r.OutboxCTRef} {
		if !domain.ReferenciaOpacaValida(ref) {
			return ErrRegistroIncorporacionV2
		}
	}
	return nil
}

func shaRegistroV2(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 32 && hex.EncodeToString(b) == s
}
func hashRegistroV2(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func jsonIgualRegistroV2(a, b any) bool {
	x, e := json.Marshal(a)
	y, f := json.Marshal(b)
	return e == nil && f == nil && bytes.Equal(x, y)
}

// TransaccionRegistroIncorporacionV2 es la ÚNICA frontera de efecto V2, sin
// fallback ni conversión a V1. Implementación durable PENDIENTE. Obligaciones:
// SERIALIZABLE RW; reloj/orden vigentes y cancelación; AD3-27 consumo fresco →
// autorización lectora AD3-28 independiente/fresca → API Personal propietaria
// con sus enlaces originales → CAS de expediente observado y seguimiento →
// actuación/recibo/historia/auditoría/outbox CT, todo EN EL MISMO COMMIT.
// La acreditación previa en memoria NO sustituye esa lectura transaccional.
// Sin SELECT a tablas Personal ni permiso de alta reutilizado para leer.
// La historia se obtiene del MISMO registro durable verificado; los constructores
// públicos solo validan coherencia, no procedencia. Replay exige ambos consumos
// frescos y acceso auditado, pero conserva recibo/historia/efecto originales.
// Sin alta Personal, sin reintentos internos, sin avance expediente implícito.
// Retorno solo después de commit confirmado; fallo/cancelación/commit incierto:
// resultado cero. El reintento EXPLÍCITO puede recuperar el commit confirmado.
type TransaccionRegistroIncorporacionV2 interface {
	RegistrarORecuperarIncorporacion(context.Context, OrdenConfirmacionIncorporacionV2, AcreditacionPersonalIncorporacion) (ResultadoRegistroIncorporacionV2, error)
}
