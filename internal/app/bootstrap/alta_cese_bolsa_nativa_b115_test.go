package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	pp "vec-diputacion-granada/internal/modules/personal/ports"
)

type anclajeB115Prueba struct {
	resultado bp.ResultadoAnclajeAceptacionNativaCT
}

func (a anclajeB115Prueba) ConsultarAnclajeAceptacionNativaCT(context.Context, bp.SolicitudConsultaAceptacionNativaCT) (bp.ResultadoAnclajeAceptacionNativaCT, error) {
	return a.resultado, nil
}
func (anclajeB115Prueba) ConsultarPersonaAceptacionNativaCT(context.Context, bp.SolicitudConsultaAceptacionNativaCT) (bp.ResultadoPersonaAceptacionNativaCT, error) {
	return bp.ResultadoPersonaAceptacionNativaCT{}, nil
}

type efectosB115Prueba struct {
	altas []bp.SolicitudAltaNativaCT
	ceses []bp.SolicitudCeseNativoCT
}

func (e *efectosB115Prueba) AplicarAltaNativaCT(_ context.Context, q bp.SolicitudAltaNativaCT) (bp.ReciboAltaNativaCT, error) {
	e.altas = append(e.altas, q)
	if len(e.altas) == 1 {
		return bp.ReciboAltaNativaCT{}, bp.ErrAltaNativaCTNoDisponible
	}
	return bp.ReciboAltaNativaCT{BolsaRef: q.Selector.BolsaRef, ParticipacionRef: q.ParticipacionRef,
		ReciboSituacion: "recibo:situacion:uno", EventoRef: "evento:ct:uno"}, nil
}
func (e *efectosB115Prueba) AplicarCeseNativoCT(_ context.Context, q bp.SolicitudCeseNativoCT) (bp.ReciboCeseNativoCT, error) {
	e.ceses = append(e.ceses, q)
	if len(e.ceses) == 1 {
		return bp.ReciboCeseNativoCT{}, bp.ErrCeseNativoCTNoDisponible
	}
	return bp.ReciboCeseNativoCT{BolsaRef: q.Selector.BolsaRef,
		ParticipacionRef: q.ParticipacionRef, ReciboRef: "recibo:bolsa:cese:uno",
		FechaEfecto: "2027-02-15"}, nil
}

func TestEfectoBolsaNativoTrasPersonalRecuperaMismoMaterial(t *testing.T) {
	autoridad, ctx, _ := escenarioCapturaAsignacionB2Prueba(t)
	actual, err := autoridad.contexto(ctx, bp.AccionConsultaAnclajeAceptacionNativaCT)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := actual.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	x := dom.SelectorBolsaPlanB2Nativo{OrganizacionRef: "organizacion:prueba",
		ExpedienteRef: "expediente:prueba", ClaveCT: "44444444-4444-4444-8444-444444444444",
		UnidadRef: "unidad:prueba", CategoriaRef: "categoria:uno", NecesidadRef: "necesidad:uno",
		BolsaRef: "bolsa:una", LlamamientoRef: "llamamiento:uno",
		ReciboEmisionRef: "recibo:emision:uno", ReciboRespuestaRef: "recibo:respuesta:uno"}
	h := strings.Repeat("a", 64)
	instante := time.Date(2026, time.October, 13, 0, 0, 0, 0, time.UTC)
	anclaje := &bp.AnclajeAceptacionNativaCT{SelectorAceptacionNativaCT: selectorAceptacionNativaB2(x),
		VersionExpediente: 8, ActaRef: "acta:una", VersionBolsa: 1, HuellaBolsaSHA256: h,
		InstantaneaRef: "instantanea:una", VersionInstantanea: 1, HuellaInstantaneaSHA256: h,
		PoliticaRef: "politica:una", VersionPolitica: 1, HuellaPoliticaSHA256: h,
		ParticipacionRef: "participacion:una", CandidatoRef: "can_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		OrdenSeleccionado: 1, RespuestaRef: "respuesta:una", RespondidaEn: instante}
	doble := &efectosB115Prueba{}
	e := &efectosBolsaNativosB115{alta: doble, cese: doble,
		anclaje: anclajeB115Prueba{resultado: bp.ResultadoAnclajeAceptacionNativaCT{
			Estado: "acreditado", Anclaje: anclaje,
			Evidencia: bp.EvidenciaConsultaPersonaAceptacionCT{DecisionRef: "decision:una",
				AuditoriaRef: "auditoria:una", ConsumoHuellaSHA256: h, ConsultadaEn: instante}}},
		actorAnclaje: func(context.Context) (bp.ActorConfiablePersonaAceptacionCT, error) {
			return bp.ActorConfiablePersonaAceptacionCT{Vinculo: actual.Vinculo, Resultado: actual.Resultado}, nil
		},
		actorSituacion: func(ctx context.Context, _ *ct.ContextoAutorizacionAltaV3) (context.Context, ct.ContextoAutorizacionAltaV3, error) {
			return ctx, actual, nil
		},
		motivo: motivoCambiarSituacionParticipacionBolsaDesarrollo()}
	contrato := ct.ContratoPlanNominalB2{PlanRef: "plan:uno", PlanSHA256: h,
		Material: dom.PlanIncorporacionPersonalB2{VersionExpediente: 8, BolsaNativa: x}}
	origen := ct.OrigenIncorporacionPersonalB2{Protocolo: ct.ProtocoloIncorporacionPersonalB2,
		ReciboRef: "recibo:incorporacion-personal-b2:uno",
		Confirmacion: ct.ConfirmacionOrigenIncorporacionB2{PlanRef: contrato.PlanRef,
			PlanSHA256: contrato.PlanSHA256, ExpedienteRef: x.ExpedienteRef,
			Hechos: ct.HechosPersonalIncorporacionB2{EmpleadoRef: "emp_aaaaaaaaaaaaaaaaaaaaaaaa",
				RelacionRef: "rel_aaaaaaaaaaaaaaaaaaaaaaaa", RelacionVersion: 2,
				RelacionReciboRef: "recibo:relacion:uno", OcupacionRef: "ocu_aaaaaaaaaaaaaaaaaaaaaaaa",
				OcupacionVersion: 2, OcupacionReciboRef: "recibo:ocupacion:uno"}}}
	for intento := 0; intento < 2; intento++ {
		err = e.confirmarAlta(ctx, ctx, contrato, origen, actual)
		if intento == 0 && !errors.Is(err, bp.ErrAltaNativaCTNoDisponible) || intento == 1 && err != nil {
			t.Fatalf("alta nativa intento %d: %v", intento, err)
		}
	}
	if len(doble.altas) != 2 || doble.altas[0].Personal != doble.altas[1].Personal ||
		doble.altas[0].Selector != doble.altas[1].Selector {
		t.Fatal("reintento alteró el recibo CT/Personal original")
	}
	recibo := ct.ReciboOperacionSeguimiento{OrganizacionRef: x.OrganizacionRef,
		ExpedienteRef: x.ExpedienteRef, ReciboRef: "recibo:cese:uno", ActorRef: vinculo.PrincipalID,
		VersionResultante: 9, FechaEfecto: "2027-02-15"}
	final := inc.ResultadoCesePersonalB2{Aplica: true, SelectorBolsaNativa: x,
		EmpleadoRef: origen.Confirmacion.Hechos.EmpleadoRef, OrganismoRef: "organismo:uno",
		UnidadRef: x.UnidadRef, RelacionRef: origen.Confirmacion.Hechos.RelacionRef,
		OcupacionRef: origen.Confirmacion.Hechos.OcupacionRef, VigenteHasta: personal.FechaCivil("2027-02-15"),
		FuenteHuellaSHA256: h,
		Recibo:             pp.ReciboActoRegistroEmpleadoB2{ReciboRef: "recibo:cierre:relacion", Version: 3},
		OcupacionRecibo:    pp.ReciboActoRegistroEmpleadoB2{ReciboRef: "recibo:cierre:ocupacion", Version: 3}}
	for intento := 0; intento < 2; intento++ {
		err = e.confirmarCese(ctx, ctx, recibo, final)
		if intento == 0 && !errors.Is(err, bp.ErrCeseNativoCTNoDisponible) || intento == 1 && err != nil {
			t.Fatalf("cese nativo intento %d: %v", intento, err)
		}
	}
	if len(doble.ceses) != 2 || !reflect.DeepEqual(doble.ceses[0].Personal, doble.ceses[1].Personal) ||
		doble.ceses[0].Personal.FuenteHuellaSHA256 != final.FuenteHuellaSHA256 {
		t.Fatal("reintento cambió prueba P48 del cese")
	}
}
