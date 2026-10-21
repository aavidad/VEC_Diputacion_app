package ports

import (
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func antecedentePortalPropuestaPrueba(t *testing.T) (SolicitudPropuestaFormalizacion, AntecedentePropuestaFormalizacion, EvidenciaAceptacionBolsaNativa) {
	t.Helper()
	s, legado, _ := propuestaDesarrolloPrueba(t)
	j := legado.Justificante.Seleccion
	p := AntecedentePortalPropuesta{
		FuenteRespuesta: "portal", OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		LlamamientoRef: s.LlamamientoRef, ResolucionRef: s.ResolucionLlamamientoAceptadaRef,
		ReciboResolucionRef: s.ReciboResolucionAceptadaRef, ResueltaEn: legado.Resolucion.ResueltaEn,
		VersionResultante: 1, RespuestaPortalRef: "respuesta:portal", ReciboRespuestaPortalRef: "recibo:portal",
		ParticipacionRef: "participacion:portal", RespondidaEn: legado.Justificante.Respuesta.RegistradaEn,
		EstadoPlazo: "revision_manual_portal", RevisionRespuestaRRHH: true, RevisionPlazoRRHH: true,
		CriterioValidacionRef: legado.Resolucion.Politica.Referencia,
		Politica: PoliticaPortalPropuesta{Referencia: legado.Resolucion.Politica.Referencia,
			Version: legado.Resolucion.Politica.Version, HuellaSHA256: legado.Resolucion.Politica.HuellaSHA256},
		Seleccion: SeleccionPortalPropuesta{ClaveCT: legado.SeleccionClave, UnidadRef: "unidad:rrhh",
			CategoriaRef: "categoria:uno", NecesidadRef: j.Necesidad.Referencia,
			BolsaRef: j.Bolsa.Referencia, LlamamientoRef: s.LlamamientoRef,
			ReciboEmisionRef: j.ReciboRef, ConfirmadaEn: j.ConfirmadaEn,
			VersionExpediente: s.VersionEsperada},
	}
	a := AntecedentePropuestaFormalizacion{Portal: &p}
	e := EvidenciaAceptacionBolsaNativa{
		Selector: domain.SelectorBolsaPlanB2Nativo{
			OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, ClaveCT: p.Seleccion.ClaveCT,
			UnidadRef: p.Seleccion.UnidadRef, CategoriaRef: p.Seleccion.CategoriaRef,
			NecesidadRef: p.Seleccion.NecesidadRef, BolsaRef: p.Seleccion.BolsaRef,
			LlamamientoRef: s.LlamamientoRef, ReciboEmisionRef: p.Seleccion.ReciboEmisionRef,
			ReciboRespuestaRef: p.ReciboRespuestaPortalRef,
		}, RespuestaRef: p.RespuestaPortalRef, RespondidaEn: p.RespondidaEn,
	}
	if a.ValidarPara(s) != nil || e.ValidarPara(a) != nil {
		t.Fatal("fixture portal inválida")
	}
	return s, a, e
}

func TestAntecedentePortalPropuestaValido(t *testing.T) {
	s, a, e := antecedentePortalPropuestaPrueba(t)
	if a.Portal.Validar() != nil || a.ValidarPara(s) != nil || e.ValidarPara(a) != nil {
		t.Fatal("antecedente portal acreditado rechazado")
	}
}

func TestAntecedentePortalPropuestaRechazaMezclasYReferenciasCruzadas(t *testing.T) {
	s, a, e := antecedentePortalPropuestaPrueba(t)
	for nombre, alterar := range map[string]func(*AntecedentePropuestaFormalizacion){
		"fuente":        func(a *AntecedentePropuestaFormalizacion) { a.Portal.FuenteRespuesta = "correo" },
		"pareja":        func(a *AntecedentePropuestaFormalizacion) { a.Portal.ReciboResolucionRef = "recibo:ajeno" },
		"participacion": func(a *AntecedentePropuestaFormalizacion) { a.Portal.ParticipacionRef = "" },
		"revision":      func(a *AntecedentePropuestaFormalizacion) { a.Portal.RevisionPlazoRRHH = false },
		"seleccion":     func(a *AntecedentePropuestaFormalizacion) { a.Portal.Seleccion.VersionExpediente++ },
		"mezcla":        func(a *AntecedentePropuestaFormalizacion) { a.SeleccionClave = "21111111-1111-4111-8111-111111111111" },
	} {
		t.Run(nombre, func(t *testing.T) {
			copia, portal := a, *a.Portal
			copia.Portal = &portal
			alterar(&copia)
			if copia.ValidarPara(s) == nil {
				t.Fatal("antecedente no acreditado admitido")
			}
		})
	}
	e.Selector.ReciboRespuestaRef = "recibo:ajeno"
	if e.ValidarPara(a) == nil {
		t.Fatal("B113 cruzado con otra respuesta")
	}
}
