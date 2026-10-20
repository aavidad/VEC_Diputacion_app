package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type emisionB98PropuestaPortalPrueba struct {
	resultado bp.ResultadoEmisionCT
	llamadas  int
}

func (d *emisionB98PropuestaPortalPrueba) LeerResultadoEmisionCT(_ context.Context, bolsa, llamamiento, recibo string) (bp.ResultadoEmisionCT, error) {
	d.llamadas++
	if bolsa != d.resultado.BolsaRef || llamamiento != d.resultado.LlamamientoRef || recibo != d.resultado.ReciboEmisionRef {
		return bp.ResultadoEmisionCT{}, bp.ErrResultadoEmisionCTDivergente
	}
	return d.resultado, nil
}

type anclajeB113PropuestaPortalPrueba struct {
	resultado bp.ResultadoAnclajeAceptacionNativaCT
	err       error
	llamadas  int
}

func (d *anclajeB113PropuestaPortalPrueba) ConsultarAnclajeAceptacionNativaCT(_ context.Context, _ bp.SolicitudConsultaAceptacionNativaCT) (bp.ResultadoAnclajeAceptacionNativaCT, error) {
	d.llamadas++
	return d.resultado, d.err
}

func (*anclajeB113PropuestaPortalPrueba) ConsultarPersonaAceptacionNativaCT(context.Context, bp.SolicitudConsultaAceptacionNativaCT) (bp.ResultadoPersonaAceptacionNativaCT, error) {
	return bp.ResultadoPersonaAceptacionNativaCT{}, bp.ErrAceptacionNativaCTDenegada
}

func TestPreparadorPropuestaPortalConservaSoloB113YRechazaMezcla(t *testing.T) {
	f := nuevaPropuestaCoordinadorPrueba(t)
	s := f.solicitud
	legado := f.registro.antecedente
	p, err := prepararReferenciasLlamamientoDesarrollo(f.lector.expediente, legado.SeleccionClave)
	if err != nil {
		t.Fatal(err)
	}
	j := legado.Justificante.Seleccion
	portal := ports.AntecedentePortalPropuesta{
		FuenteRespuesta: "portal", OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		LlamamientoRef: s.LlamamientoRef, ResolucionRef: s.ResolucionLlamamientoAceptadaRef,
		ReciboResolucionRef: s.ReciboResolucionAceptadaRef, ResueltaEn: legado.Resolucion.ResueltaEn,
		VersionResultante: 1, RespuestaPortalRef: "respuesta:portal", ReciboRespuestaPortalRef: "recibo:portal",
		ParticipacionRef: "participacion:portal", RespondidaEn: legado.Justificante.Respuesta.RegistradaEn,
		EstadoPlazo: "revision_manual_portal", RevisionRespuestaRRHH: true, RevisionPlazoRRHH: true,
		CriterioValidacionRef: legado.Resolucion.Politica.Referencia,
		Politica: ports.PoliticaPortalPropuesta{Referencia: legado.Resolucion.Politica.Referencia,
			Version: legado.Resolucion.Politica.Version, HuellaSHA256: legado.Resolucion.Politica.HuellaSHA256},
		Seleccion: ports.SeleccionPortalPropuesta{ClaveCT: p.clave, UnidadRef: p.unidad,
			CategoriaRef: p.categoria, NecesidadRef: p.necesidad, BolsaRef: j.Bolsa.Referencia,
			LlamamientoRef: s.LlamamientoRef, ReciboEmisionRef: j.ReciboRef,
			ConfirmadaEn: j.ConfirmadaEn, VersionExpediente: s.VersionEsperada},
	}
	a := ports.AntecedentePropuestaFormalizacion{Portal: &portal}
	e := ports.EvidenciaAceptacionBolsaNativa{Selector: domain.SelectorBolsaPlanB2Nativo{
		OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, ClaveCT: p.clave,
		UnidadRef: p.unidad, CategoriaRef: p.categoria, NecesidadRef: p.necesidad,
		BolsaRef: portal.Seleccion.BolsaRef, LlamamientoRef: s.LlamamientoRef,
		ReciboEmisionRef: j.ReciboRef, ReciboRespuestaRef: portal.ReciboRespuestaPortalRef,
	}, RespuestaRef: portal.RespuestaPortalRef, RespondidaEn: portal.RespondidaEn}
	m := ports.MaterialPropuestaFormalizacion{Etapa: "confirmacion", Solicitud: s, AceptacionBolsaNativa: &e}
	if a.ValidarPara(s) != nil || !materialPropuestaDesarrolloAcreditado(m, a) {
		t.Fatal("material portal válido denegado")
	}
	ctx := context.WithValue(f.ctx, clavePropuestaFormalizacionDesarrollo{}, propuestaFormalizacionLigadaDesarrollo{antecedente: a, material: m})
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, p)
	ctx = context.WithValue(ctx, claveMaterialPropuestaFormalizacionDesarrollo{}, m)
	preparado, err := f.registro.proveedor.PrepararPropuestaFormalizacion(ctx, s)
	if err != nil || !materialPropuestaDesarrolloIgual(preparado, m) {
		t.Fatal("preparación perdió B113", err)
	}
	mezcla := m.Clonar()
	mezcla.AceptacionBolsa = &ports.EvidenciaAceptacionBolsaPropuesta{}
	if materialPropuestaDesarrolloAcreditado(mezcla, a) || materialPropuestaDesarrolloIgual(preparado, mezcla) {
		t.Fatal("material nativo y legado mezclados")
	}
	cruzado := m.Clonar()
	cruzado.AceptacionBolsaNativa.Selector.ReciboRespuestaRef = "recibo:ajeno"
	if materialPropuestaDesarrolloAcreditado(cruzado, a) || materialPropuestaDesarrolloIgual(preparado, cruzado) {
		t.Fatal("recibo de otra respuesta admitido")
	}
}

func TestMaterialPropuestaNativaCTRevalidaPortalConB98YB113(t *testing.T) {
	f := nuevaPropuestaCoordinadorPrueba(t)
	s := f.solicitud
	legado := f.registro.antecedente
	p, err := prepararReferenciasLlamamientoDesarrollo(f.lector.expediente, legado.SeleccionClave)
	if err != nil {
		t.Fatal(err)
	}
	j := legado.Justificante.Seleccion
	portal := ports.AntecedentePortalPropuesta{
		FuenteRespuesta: "portal", OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		LlamamientoRef: s.LlamamientoRef, ResolucionRef: s.ResolucionLlamamientoAceptadaRef,
		ReciboResolucionRef: s.ReciboResolucionAceptadaRef, ResueltaEn: legado.Resolucion.ResueltaEn,
		VersionResultante: 1, RespuestaPortalRef: "respuesta:portal", ReciboRespuestaPortalRef: "recibo:portal",
		ParticipacionRef: "participacion:portal", RespondidaEn: legado.Justificante.Respuesta.RegistradaEn,
		EstadoPlazo: "revision_manual_portal", RevisionRespuestaRRHH: true, RevisionPlazoRRHH: true,
		CriterioValidacionRef: legado.Resolucion.Politica.Referencia,
		Politica: ports.PoliticaPortalPropuesta{Referencia: legado.Resolucion.Politica.Referencia,
			Version: legado.Resolucion.Politica.Version, HuellaSHA256: legado.Resolucion.Politica.HuellaSHA256},
		Seleccion: ports.SeleccionPortalPropuesta{ClaveCT: p.clave, UnidadRef: p.unidad,
			CategoriaRef: p.categoria, NecesidadRef: p.necesidad, BolsaRef: j.Bolsa.Referencia,
			LlamamientoRef: s.LlamamientoRef, ReciboEmisionRef: j.ReciboRef,
			ConfirmadaEn: j.ConfirmadaEn, VersionExpediente: s.VersionEsperada},
	}
	a := ports.AntecedentePropuestaFormalizacion{Portal: &portal}
	respuesta := bp.ResultadoParticipacionEmisionCT{ParticipacionRef: portal.ParticipacionRef,
		Respuesta: bp.RespuestaPortalAcepta, Modo: bp.ModoRespuestaPortalFirme,
		ReciboRespuestaRef: portal.ReciboRespuestaPortalRef, RespondidaEn: &portal.RespondidaEn}
	b98 := &emisionB98PropuestaPortalPrueba{resultado: bp.ResultadoEmisionCT{
		BolsaRef: portal.Seleccion.BolsaRef, LlamamientoRef: s.LlamamientoRef,
		ReciboEmisionRef: portal.Seleccion.ReciboEmisionRef, EmitidoEn: portal.Seleccion.ConfirmadaEn,
		Participaciones: []bp.ResultadoParticipacionEmisionCT{respuesta}}}
	selector := bp.SelectorAceptacionNativaCT{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		ClaveCT: p.clave, UnidadRef: p.unidad, CategoriaRef: p.categoria, NecesidadRef: p.necesidad,
		BolsaRef: portal.Seleccion.BolsaRef, LlamamientoRef: s.LlamamientoRef,
		ReciboEmisionRef: portal.Seleccion.ReciboEmisionRef, ReciboRespuestaRef: portal.ReciboRespuestaPortalRef}
	h := strings.Repeat("a", 64)
	b113 := &anclajeB113PropuestaPortalPrueba{resultado: bp.ResultadoAnclajeAceptacionNativaCT{
		Estado: "acreditado", Anclaje: &bp.AnclajeAceptacionNativaCT{
			SelectorAceptacionNativaCT: selector, VersionExpediente: s.VersionEsperada,
			ParticipacionRef: portal.ParticipacionRef, RespuestaRef: portal.RespuestaPortalRef,
			RespondidaEn: portal.RespondidaEn},
		Evidencia: bp.EvidenciaConsultaPersonaAceptacionCT{DecisionRef: "decision:portal",
			AuditoriaRef: "auditoria:portal", ConsumoHuellaSHA256: h, ConsultadaEn: portal.ResueltaEn}}}
	f.ejecutor.resultadoEmisionNativo = b98
	f.ejecutor.aceptacionNativa = b113
	f.ejecutor.actorAceptacionNativa = func(context.Context) (bp.ActorConfiablePersonaAceptacionCT, error) {
		return bp.ActorConfiablePersonaAceptacionCT{}, nil
	}
	m, err := f.ejecutor.materialPropuestaNativaCT(f.ctx, s, a, p)
	if err != nil || m.AceptacionBolsaNativa == nil || m.AceptacionBolsaNativa.ValidarPara(a) != nil ||
		b98.llamadas != 1 || b113.llamadas != 1 {
		t.Fatal("B98/B113 portal válido rechazado", err)
	}
	for nombre, alterar := range map[string]func(*bp.ResultadoParticipacionEmisionCT){
		"respuesta":     func(x *bp.ResultadoParticipacionEmisionCT) { x.ReciboRespuestaRef = "recibo:ajeno" },
		"participacion": func(x *bp.ResultadoParticipacionEmisionCT) { x.ParticipacionRef = "participacion:ajena" },
		"fecha": func(x *bp.ResultadoParticipacionEmisionCT) {
			v := x.RespondidaEn.Add(time.Second)
			x.RespondidaEn = &v
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			x := respuesta
			alterar(&x)
			b98.resultado.Participaciones = []bp.ResultadoParticipacionEmisionCT{x}
			antes := b113.llamadas
			if _, err := f.ejecutor.materialPropuestaNativaCT(f.ctx, s, a, p); err == nil || b113.llamadas != antes {
				t.Fatal("respuesta B98 cruzada avanzó a B113")
			}
		})
	}
	b98.resultado.Participaciones = []bp.ResultadoParticipacionEmisionCT{respuesta}
	b113.err = bp.ErrAceptacionNativaCTNoDisponible
	if _, err := f.ejecutor.materialPropuestaNativaCT(f.ctx, s, a, p); !errors.Is(err, application.ErrPropuestaFormalizacionNoDisponible) {
		t.Fatal("error B113 presentado como éxito", err)
	}
	f.ejecutor.resultadoEmisionNativo = nil
	if _, err := f.ejecutor.materialPropuestaNativaCT(f.ctx, s, a, p); !errors.Is(err, application.ErrPropuestaFormalizacionNoDisponible) {
		t.Fatal("fuente portal ausente clasificada como aceptación inválida", err)
	}
}
