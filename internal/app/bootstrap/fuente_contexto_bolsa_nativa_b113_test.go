package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type consultaAceptacionNativaB113Prueba struct {
	resultado bp.ResultadoPersonaAceptacionNativaCT
	selector  bp.SelectorAceptacionNativaCT
	llamadas  int
}

func (p *consultaAceptacionNativaB113Prueba) ConsultarAnclajeAceptacionNativaCT(context.Context,
	bp.SolicitudConsultaAceptacionNativaCT) (bp.ResultadoAnclajeAceptacionNativaCT, error) {
	return bp.ResultadoAnclajeAceptacionNativaCT{}, nil
}
func (p *consultaAceptacionNativaB113Prueba) ConsultarPersonaAceptacionNativaCT(_ context.Context,
	s bp.SolicitudConsultaAceptacionNativaCT) (bp.ResultadoPersonaAceptacionNativaCT, error) {
	p.llamadas++
	p.selector = s.Selector
	return p.resultado, nil
}

func TestConsumidorB113ConservaRecibosNativosExactos(t *testing.T) {
	h := strings.Repeat("a", 64)
	x := dom.SelectorBolsaPlanB2Nativo{
		OrganizacionRef: "organizacion:prueba", ExpedienteRef: "expediente:prueba",
		ClaveCT:   "44444444-4444-4444-8444-444444444444",
		UnidadRef: "unidad:rrhh", CategoriaRef: "categoria:uno", NecesidadRef: "necesidad:uno",
		BolsaRef: "bolsa:uno", LlamamientoRef: "llamamiento:uno",
		ReciboEmisionRef: "recibo:emision:uno", ReciboRespuestaRef: "recibo:respuesta:uno",
	}
	selector := selectorAceptacionNativaB2(x)
	instante := time.Date(2026, time.October, 13, 0, 0, 0, 0, time.UTC)
	anclaje := &bp.AnclajeAceptacionNativaCT{
		SelectorAceptacionNativaCT: selector,
		VersionExpediente:          6, ActaRef: "acta:uno", VersionBolsa: 1, HuellaBolsaSHA256: h,
		InstantaneaRef: "instantanea:uno", VersionInstantanea: 1, HuellaInstantaneaSHA256: h,
		PoliticaRef: "politica:uno", VersionPolitica: 1, HuellaPoliticaSHA256: h,
		ParticipacionRef: "participacion:uno", CandidatoRef: "can_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		OrdenSeleccionado: 1, RespuestaRef: "respuesta:uno", RespondidaEn: instante,
	}
	doble := &consultaAceptacionNativaB113Prueba{resultado: bp.ResultadoPersonaAceptacionNativaCT{
		Estado: "acreditado", Aceptacion: anclaje,
		Persona: &bp.PersonaAceptadaCT{Ref: "per_aaaaaaaaaaaaaaaaaaaaaaaa", Version: 1},
		Vinculo: &bp.VinculoPersonaAceptadaCT{ProcedenciaRef: "fuente:persona", ProcedenciaVersion: 1, ProcedenciaSHA256: h},
		Evidencia: bp.EvidenciaConsultaPersonaAceptacionCT{DecisionRef: "decision:uno",
			AuditoriaRef: "auditoria:uno", ConsumoHuellaSHA256: h, ConsultadaEn: instante},
	}}
	f := &FuentesContextoIncorporacionB2{c: ConfiguracionFuentesContextoIncorporacionB2{
		BolsaNativa: doble,
		ActorBolsa: func(context.Context) (bp.ActorConfiablePersonaAceptacionCT, error) {
			return bp.ActorConfiablePersonaAceptacionCT{}, nil
		},
	}}
	c := inc.ContratoPlanNominal{Protocolo: inc.ProtocoloPersonalB2V1,
		OrganizacionRef: x.OrganizacionRef, ExpedienteRef: x.ExpedienteRef,
		CategoriaRef: x.CategoriaRef, LlamamientoRef: x.LlamamientoRef,
		SeleccionReciboRef: x.ReciboRespuestaRef, AceptacionRef: "aceptacion:ct",
		SelectorBolsaNativa: x}
	p, err := f.leerPersonaSeleccionadaNativaB2(context.Background(), c)
	if err != nil || doble.llamadas != 1 || doble.selector != selector ||
		p.ReciboRef != x.ReciboRespuestaRef || p.LlamamientoRef != x.LlamamientoRef ||
		p.PersonaRef != doble.resultado.Persona.Ref {
		t.Fatal("el consumidor no conservó B113", err)
	}
	autoridad, ctx, _ := escenarioCapturaAsignacionB2Prueba(t)
	doble.resultado.Vinculo.VigenteHasta = autoridad.reloj.Ahora().Add(time.Hour)
	f.c.ActorBolsa = func(ctx context.Context) (bp.ActorConfiablePersonaAceptacionCT, error) {
		actual, err := autoridad.contexto(ctx, bp.AccionConsultaPersonaAceptacionNativaCT)
		return bp.ActorConfiablePersonaAceptacionCT{Vinculo: actual.Vinculo, Resultado: actual.Resultado}, err
	}
	for range 2 {
		if _, err := f.leerPersonaSeleccionadaNativaB2(ctx, c); err != nil {
			t.Fatal("consulta nativa capturada", err)
		}
	}
	if doble.llamadas != 2 {
		t.Fatalf("la petición consumió %d consultas B113 en lugar de una", doble.llamadas-1)
	}
	doble.resultado.Aceptacion = &bp.AnclajeAceptacionNativaCT{}
	if _, err := f.leerPersonaSeleccionadaNativaB2(context.Background(), c); err == nil {
		t.Fatal("anclaje de otra emisión admitido")
	}
}
