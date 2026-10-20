package bootstrap

import (
	"context"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

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
