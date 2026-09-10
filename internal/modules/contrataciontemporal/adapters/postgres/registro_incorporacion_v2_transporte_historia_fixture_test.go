package postgres

import (
	"encoding/json"
	"os"
	"testing"
	"time"
	domain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func registroV2Consumos(t *testing.T, o ct.OrdenConfirmacionIncorporacionV2, ahora time.Time) ct.ConsumosRegistroIncorporacionV2 {
	h, e := o.Exportacion().HuellaConjuntoSHA256()
	registroV2Exigir(t, e)
	dec := o.Exportacion().ResumenCapacidad().DecisionRef()
	return ct.ConsumosRegistroIncorporacionV2{
		DecisionCTRef: dec, HuellaExportacionCT: h, ConsumoCTSHA256: registroV2Hash([]byte("ct" + dec)), AuditoriaAD3CTRef: "aud:ad3:ct:" + dec, ConsumidaCTEn: ahora,
		DecisionLecturaRef: "decision:lectura:" + dec, ConsumoLecturaSHA256: registroV2Hash([]byte("lectura" + dec)), AuditoriaAD3LecturaRef: "aud:ad3:lectura:" + dec,
		AuditoriaPersonalLecturaRef: "aud:personal:lectura:" + dec, LeidaPersonalEn: ahora,
	}
}

func registroV2ResultadoOriginal(t *testing.T, o ct.OrdenConfirmacionIncorporacionV2, ahora time.Time) ct.ResultadoRegistroIncorporacionV2 {
	t.Helper()
	d, e := o.Material().Datos()
	registroV2Exigir(t, e)
	b, e := os.ReadFile("../seguimientoejercicio/testdata/definicion-ejercicio.json")
	registroV2Exigir(t, e)
	var fuente struct {
		Publicacion domain.PublicacionDefinicionSeguimiento `json:"publicacion"`
	}
	registroV2Exigir(t, json.Unmarshal(b, &fuente))
	def, e := domain.RestaurarDefinicionSeguimiento(fuente.Publicacion)
	registroV2Exigir(t, e)
	// Raíz de ejercicio suministrada por el doble de almacén, NO por el servicio.
	antes, e := domain.NuevoSeguimiento(def, domain.AltaSeguimiento{Referencia: registroV2Ref("seguimiento:ejercicio:001"), OrganizacionRef: d.Preparacion.OrganizacionRef,
		ExpedienteRef: d.Confirmacion.SolicitudPersonal.ExpedienteRef, RelacionRef: d.Confirmacion.ResultadoPersonal.RelacionRef,
		PeriodoPrevisto: d.Confirmacion.PeriodoIncorporacion, CreadoEn: ahora.Add(-time.Minute)})
	registroV2Exigir(t, e)
	periodo := d.Confirmacion.PeriodoIncorporacion
	transicion := domain.DatosTransicionSeguimiento{ActuacionRef: registroV2Ref("actuacion:ejercicio:ct:001"), TransicionClave: ct.TransicionConfirmarIncorporacion,
		MotivoClave: d.Confirmacion.MotivoClave, ActorRef: d.Preparacion.ActorRef, UnidadRef: d.Preparacion.UnidadRef, EfectivoEn: periodo.Desde, RegistradaEn: ahora,
		Documentos: d.Confirmacion.Documentos, Periodo: &periodo, ReciboRef: registroV2Ref("recibo:ejercicio:ct:001"), CorrelacionRef: d.Preparacion.CorrelacionRef}
	despues, e := antes.Aplicar(def, d.Confirmacion.VersionSeguimientoEsperada, transicion)
	registroV2Exigir(t, e)
	ca, e := domain.SerializarEstadoSeguimientoCanonico(def, antes.Estado())
	registroV2Exigir(t, e)
	cp, e := domain.SerializarEstadoSeguimientoCanonico(def, despues.Estado())
	registroV2Exigir(t, e)
	canon, e := o.Material().MaterialCanonico()
	registroV2Exigir(t, e)
	intencion, e := ct.IntencionRegistroIncorporacionV2(o.Material())
	registroV2Exigir(t, e)
	r := ct.ReciboRegistroIncorporacionV2{Esquema: ct.EsquemaReciboRegistroIncorporacionV2, MaterialOriginalCanonico: canon, MaterialOriginalSHA256: registroV2Hash(canon),
		IntencionSHA256: registroV2Hash(intencion), ContextoOriginalRef: d.Contexto.Resultado.RegistroContextoRef,
		VersionSolicitudPersonal: d.Confirmacion.SolicitudPersonal.VersionExpediente, VersionActualExpediente: d.VersionActualExpediente,
		SeguimientoRef: antes.Estado().Referencia, DefinicionSeguimiento: def.Referencia(), HuellaRaizSeguimiento: antes.Estado().HuellaRaizSHA256,
		HuellaEstadoAnterior: registroV2Hash(ca), HuellaEstadoResultante: registroV2Hash(cp), VersionSeguimientoAnterior: antes.Version(), VersionSeguimientoResultante: despues.Version(),
		Transicion: transicion, AuditoriaCTRef: "auditoria:ejercicio:ct:001", OutboxCTRef: "outbox:ejercicio:ct:001", ConsumosOriginales: registroV2Consumos(t, o, ahora), EjercicioSintetico: true}
	registroV2Exigir(t, r.ValidarPara(o))
	h, e := ct.NuevaHistoriaRegistroIncorporacionV2(o, r, def.Publicacion(), antes.Estado(), despues.Estado())
	registroV2Exigir(t, e)
	return ct.ResultadoRegistroIncorporacionV2{Recibo: r, Historia: h, ConsumosActuales: r.ConsumosOriginales}
}
