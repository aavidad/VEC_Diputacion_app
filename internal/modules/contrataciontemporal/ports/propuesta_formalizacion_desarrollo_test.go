package ports

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestMaterialPropuestaFormalizacionNativaSinAceptacionLegada(t *testing.T) {
	s, a, vieja := propuestaDesarrolloPrueba(t)
	a.Justificante.Seleccion.Procedencia.SinRetencionDeclarada = true
	a.Justificante.Seleccion.Procedencia.Evidencia.RetenerHasta = time.Time{}
	a.Justificante.Seleccion.RetencionSeleccion = ReferenciaVersionadaIntegracionBolsa{}
	if a.ValidarPara(s) != nil {
		t.Fatal("antecedente CT nativo inválido")
	}
	nativa := EvidenciaAceptacionBolsaNativa{
		Selector: domain.SelectorBolsaPlanB2Nativo{
			OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
			ClaveCT: a.SeleccionClave, UnidadRef: "unidad:rrhh", CategoriaRef: "categoria:uno",
			NecesidadRef:   a.Justificante.Seleccion.Necesidad.Referencia,
			BolsaRef:       a.Justificante.Seleccion.Bolsa.Referencia,
			LlamamientoRef: s.LlamamientoRef, ReciboEmisionRef: a.Justificante.Seleccion.ReciboRef,
			ReciboRespuestaRef: "recibo:respuesta:nativa",
		},
		RespuestaRef: "respuesta:nativa", RespondidaEn: a.Resolucion.ResueltaEn.Add(-time.Second),
	}
	m := MaterialPropuestaFormalizacion{Etapa: "confirmacion", Solicitud: s, AceptacionBolsaNativa: &nativa}
	if m.Validar() != nil || nativa.ValidarPara(a) != nil {
		t.Fatal("evidencia B113 real rechazada")
	}
	b, err := json.Marshal(m)
	if err != nil || bytes.Contains(b, []byte(`"AceptacionBolsa":`)) ||
		!bytes.Contains(b, []byte(`"AceptacionBolsaNativa":`)) ||
		bytes.Contains(b, []byte(`"AperturaOperacionRef"`)) {
		t.Fatal("el material nativo arrastró ledger antiguo", err)
	}
	mezcla := m
	mezcla.AceptacionBolsa = &vieja
	if mezcla.Validar() == nil {
		t.Fatal("dos autoridades de aceptación simultáneas")
	}
	cruzada := nativa
	cruzada.Selector.ReciboEmisionRef = "recibo:otro"
	if cruzada.ValidarPara(a) == nil {
		t.Fatal("emisión B7 ajena aceptada")
	}
}

func TestPropuestaDerivadaConservaJSONLegadoYExcluyeOtrosOrigenes(t *testing.T) {
	legado, _, vieja := propuestaDesarrolloPrueba(t)
	b, err := json.Marshal(legado)
	if err != nil {
		t.Fatal(err)
	}
	esperado, err := json.Marshal(struct {
		ClaveIdempotencia                string
		OrganizacionRef                  string
		ExpedienteRef                    string
		LlamamientoRef                   string
		ResolucionLlamamientoAceptadaRef string
		ReciboResolucionAceptadaRef      string
		VersionEsperada                  uint64
		TipoFormalizacion                SnapshotGobernadoFormalizacion
		Plantilla                        SnapshotGobernadoFormalizacion
		Anexos                           []AnexoPropuestaFormalizacion
		PoliticaFirma                    SnapshotGobernadoFormalizacion
		PlanFirma                        SnapshotGobernadoFormalizacion
	}{legado.ClaveIdempotencia, legado.OrganizacionRef, legado.ExpedienteRef,
		legado.LlamamientoRef, legado.ResolucionLlamamientoAceptadaRef, legado.ReciboResolucionAceptadaRef,
		legado.VersionEsperada, legado.TipoFormalizacion, legado.Plantilla, legado.Anexos, legado.PoliticaFirma, legado.PlanFirma})
	if err != nil || !bytes.Equal(b, esperado) {
		t.Fatal("cambiaron los bytes del JSON legado", err)
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(b, &campos) != nil || len(campos) != 12 || bytes.Contains(b, []byte(`"DerivacionRef"`)) {
		t.Fatal("cambió el JSON legado de doce campos")
	}
	s := legado
	padreRef := s.ExpedienteRef
	s.DerivacionRef = "derivacion:ct234:01"
	s.ExpedienteRef = HijoRefDerivacionNombramiento(s.OrganizacionRef, padreRef, 1)
	s.LlamamientoRef, s.ResolucionLlamamientoAceptadaRef, s.ReciboResolucionAceptadaRef = "", "", ""
	e := EvidenciaAceptacionDerivada{
		DerivacionRef: s.DerivacionRef, PadreRef: padreRef, HijoRef: s.ExpedienteRef,
		OfertaRef: "oferta:" + strings.Repeat("a", 64), NumeroPlaza: 1,
		AceptacionRef: "aceptacion-plaza:sintetica:01", ReciboDerivacionRef: "recibo:derivacion:sintetica:01",
	}
	a := AntecedentePropuestaFormalizacion{Derivacion: &e}
	m := MaterialPropuestaFormalizacion{Etapa: "confirmacion", Solicitud: s, AceptacionDerivada: &e}
	if s.Validar() != nil || e.ValidarPara(s) != nil || a.ValidarPara(s) != nil || m.Validar() != nil {
		t.Fatal("origen derivado válido rechazado")
	}
	clon := m.Clonar()
	clon.AceptacionDerivada.ReciboDerivacionRef = "recibo:otro"
	if m.AceptacionDerivada.ReciboDerivacionRef == clon.AceptacionDerivada.ReciboDerivacionRef {
		t.Fatal("evidencia compartida por el clon")
	}
	b, err = json.Marshal(m)
	if err != nil || !bytes.Contains(b, []byte(`"AceptacionDerivada":`)) || bytes.Contains(b, []byte(`"AceptacionBolsa":`)) || bytes.Contains(b, []byte(`"AceptacionBolsaNativa":`)) {
		t.Fatal("JSON de material derivado mezclado", err)
	}
	cruce := m
	cruce.AceptacionBolsa = &vieja
	if cruce.Validar() == nil {
		t.Fatal("aceptación derivada y legado simultáneos")
	}
	cruce = m
	cruce.AceptacionBolsaNativa = &EvidenciaAceptacionBolsaNativa{}
	if cruce.Validar() == nil {
		t.Fatal("aceptación derivada y nativa simultáneas")
	}
	cruce = m
	cruce.AceptacionDerivada = &EvidenciaAceptacionDerivada{}
	if cruce.Validar() == nil {
		t.Fatal("aceptación sin ligadura")
	}
	cruce = m
	cruce.Solicitud.VersionEsperada = 7
	if cruce.Validar() == nil {
		t.Fatal("propuesta inicial derivada fuera de v6")
	}
	mezcla := a
	mezcla.SeleccionClave = "21111111-1111-4111-8111-111111111111"
	if mezcla.ValidarPara(s) == nil {
		t.Fatal("antecedente derivado con selección antigua")
	}
	mezcla = a
	mezcla.Resolucion.ResolucionRef = "resolucion:falsa"
	if mezcla.ValidarPara(s) == nil {
		t.Fatal("antecedente derivado con resolución ficticia")
	}
	mezcla = a
	mezcla.Derivacion = &EvidenciaAceptacionDerivada{}
	if mezcla.ValidarPara(s) == nil {
		t.Fatal("antecedente derivado desligado")
	}
}

func TestPropuestaFormalizacionDesarrolloSucesorConservaRaizYMaterial(t *testing.T) {
	s, a, e := propuestaDesarrolloPrueba(t)
	raiz := a.Justificante.Seleccion
	// El justificante sin continuación mantiene los dos campos históricos.
	antes, _ := json.Marshal(a.Justificante)
	esperado, _ := json.Marshal(struct {
		Respuesta RespuestaRecibidaRegistrada
		Seleccion ReciboSolicitudLlamamientoBolsa
	}{a.Justificante.Respuesta, raiz})
	if !bytes.Equal(antes, esperado) {
		t.Fatal("bytes históricos cambiados")
	}
	sc, _, b := continuacionPrueba(t)
	sc.OrganizacionRef, sc.ExpedienteRef = s.OrganizacionRef, s.ExpedienteRef
	b.ConfirmadaEn = raiz.ConfirmadaEn.Add(time.Second)
	c := ResultadoContinuacionLlamamiento{Solicitud: sc, LlamamientoAnteriorRef: raiz.LlamamientoRef, ReciboBolsa: b,
		ReciboRef: "recibo:continuacion", AuditoriaRef: "auditoria:continuacion", ConfirmadaEn: b.ConfirmadaEn.Add(time.Second), Estado: "confirmado"}
	a.Justificante.Continuacion = &c
	s.LlamamientoRef, a.Resolucion.Solicitud.LlamamientoRef = b.LlamamientoRef, b.LlamamientoRef
	a.Justificante.Respuesta.Solicitud.LlamamientoRef = b.LlamamientoRef
	e.LlamamientoRef, e.AperturaOperacionRef = b.LlamamientoRef, b.OperacionRef
	if a.ValidarPara(s) != nil || e.ValidarPara(a) != nil || a.Justificante.Seleccion != raiz {
		t.Fatal("sucesor ligado rechazado")
	}
	m := MaterialPropuestaFormalizacion{Etapa: "confirmacion", Solicitud: s, AceptacionBolsa: &e}
	bm, _ := json.Marshal(m)
	var campos map[string]json.RawMessage
	if m.Validar() != nil || json.Unmarshal(bm, &campos) != nil || len(campos) != 3 {
		t.Fatal("otro material de propuesta")
	}
	e.AperturaOperacionRef = raiz.OperacionRef
	if e.ValidarPara(a) == nil {
		t.Fatal("apertura raíz usada para el sucesor")
	}
	e.AperturaOperacionRef = b.OperacionRef
	for _, alterar := range []func(*ResultadoContinuacionLlamamiento){
		func(c *ResultadoContinuacionLlamamiento) { c.LlamamientoAnteriorRef += "otro" },
		func(c *ResultadoContinuacionLlamamiento) { c.ReciboBolsa.LlamamientoRef += "otro" },
		func(c *ResultadoContinuacionLlamamiento) { c.Estado = "replay_confirmado" },
	} {
		otra, copia := a, c
		alterar(&copia)
		otra.Justificante.Continuacion = &copia
		if e.ValidarPara(otra) == nil {
			t.Fatal("cadena cruzada")
		}
	}
}

func propuestaDesarrolloPrueba(t *testing.T) (SolicitudPropuestaFormalizacion, AntecedentePropuestaFormalizacion, EvidenciaAceptacionBolsaPropuesta) {
	t.Helper()
	s, j := justificanteRespuestaRecibidaPrueba(t)
	s.Respuesta, j.Respuesta.Solicitud.Respuesta = RespuestaLlamamientoAceptada, RespuestaLlamamientoAceptada
	s.RevisionRespuestaRRHH, s.RevisionPlazoRRHH = true, true
	s.CriterioValidacionRef = "politica:manual-sintetica"
	r := ResultadoResolucionLlamamiento{Solicitud: s,
		Politica:           ReferenciaGobernadaComunicacionLlamamiento{Referencia: s.CriterioValidacionRef, Version: 1, HuellaSHA256: strings.Repeat("a", 64)},
		EvaluacionPlazoRef: "evaluacion:manual", EstadoPlazo: PlazoLlamamientoVigente,
		ResolucionRef: "resolucion:aceptada", ReciboLocalRef: "recibo:aceptada", AuditoriaRef: "auditoria:aceptada",
		VersionResultante: 3, ResueltaEn: j.Respuesta.RegistradaEn.Add(time.Minute), Estado: ResultadoComunicacionLlamamientoConfirmado}
	p := solicitudPropuestaFormalizacionPrueba()
	p.VersionEsperada, p.Anexos = 6, nil
	p.OrganizacionRef, p.ExpedienteRef, p.LlamamientoRef = s.OrganizacionRef, s.ExpedienteRef, s.LlamamientoRef
	p.ResolucionLlamamientoAceptadaRef, p.ReciboResolucionAceptadaRef = r.ResolucionRef, r.ReciboLocalRef
	p.TipoFormalizacion.Version, p.Plantilla.Version, p.PoliticaFirma.Version, p.PlanFirma.Version = 1, 1, 1, 1
	a := AntecedentePropuestaFormalizacion{Resolucion: r, Justificante: j, SeleccionClave: "21111111-1111-4111-8111-111111111111"}
	e := EvidenciaAceptacionBolsaPropuesta{OperacionRef: "operacion:aceptada", AperturaOperacionRef: j.Seleccion.OperacionRef,
		LlamamientoRef: s.LlamamientoRef, JustificanteRef: s.PruebaRespuestaRef, EvaluacionPlazoRef: r.EvaluacionPlazoRef,
		Politica:       SnapshotGobernadoFormalizacion{Referencia: r.Politica.Referencia, Version: r.Politica.Version, HuellaSHA256: r.Politica.HuellaSHA256},
		RegistroSHA256: strings.Repeat("b", 64), ResueltaEn: r.ResueltaEn.Add(time.Second)}
	if a.ValidarPara(p) != nil || e.ValidarPara(a) != nil {
		t.Fatal("fixture estructural incoherente")
	}
	return p, a, e
}

func TestPropuestaFormalizacionDesarrolloMaterialYAntecedente(t *testing.T) {
	s, a, e := propuestaDesarrolloPrueba(t)
	consulta := MaterialPropuestaFormalizacion{Etapa: "consulta", Solicitud: s}
	confirmacion := MaterialPropuestaFormalizacion{Etapa: "confirmacion", Solicitud: s, AceptacionBolsa: &e}
	for _, m := range []MaterialPropuestaFormalizacion{consulta, confirmacion} {
		if err := m.Validar(); err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		var campos map[string]json.RawMessage
		if json.Unmarshal(b, &campos) != nil || len(campos) != map[string]int{"consulta": 2, "confirmacion": 3}[m.Etapa] {
			t.Fatal("envoltorio inesperado")
		}
	}
	clon := confirmacion.Clonar()
	clon.AceptacionBolsa.OperacionRef = "operacion:otro"
	if e.OperacionRef != "operacion:aceptada" {
		t.Fatal("alias de evidencia")
	}
	for nombre, mutar := range map[string]func(*AntecedentePropuestaFormalizacion){
		"resolucion": func(x *AntecedentePropuestaFormalizacion) { x.Resolucion.ResolucionRef = "resolucion:otra" },
		"renuncia": func(x *AntecedentePropuestaFormalizacion) {
			x.Resolucion.Solicitud.Respuesta = RespuestaLlamamientoRenunciada
		},
		"declaracion": func(x *AntecedentePropuestaFormalizacion) {
			x.Justificante.Respuesta.Solicitud.ExpedienteRef = "expediente:otro"
		},
		"seleccion": func(x *AntecedentePropuestaFormalizacion) { x.SeleccionClave = "no-uuid" },
		"politica":  func(x *AntecedentePropuestaFormalizacion) { x.Resolucion.Politica.Referencia = "politica:otra" },
	} {
		t.Run(nombre, func(t *testing.T) {
			x := a
			mutar(&x)
			if x.ValidarPara(s) == nil {
				t.Fatal("antecedente cruzado")
			}
		})
	}
	for _, mutar := range []func(*EvidenciaAceptacionBolsaPropuesta){
		func(x *EvidenciaAceptacionBolsaPropuesta) { x.AperturaOperacionRef = "operacion:otra" },
		func(x *EvidenciaAceptacionBolsaPropuesta) { x.JustificanteRef = "justificante:otro" },
		func(x *EvidenciaAceptacionBolsaPropuesta) { x.Politica.Version++ },
		func(x *EvidenciaAceptacionBolsaPropuesta) { x.ResueltaEn = a.Resolucion.ResueltaEn.Add(-time.Second) },
		func(x *EvidenciaAceptacionBolsaPropuesta) { x.RegistroSHA256 = strings.Repeat("0", 64) },
	} {
		x := e
		mutar(&x)
		if x.ValidarPara(a) == nil {
			t.Fatal("terminal cruzado")
		}
	}
}

func TestPropuestaFormalizacionDesarrolloRechazaEtapaYSuperficie(t *testing.T) {
	s, _, e := propuestaDesarrolloPrueba(t)
	for _, m := range []MaterialPropuestaFormalizacion{
		{Etapa: "consulta", Solicitud: s, AceptacionBolsa: &e},
		{Etapa: "confirmacion", Solicitud: s},
		{Etapa: "otra", Solicitud: s},
	} {
		if m.Validar() == nil {
			t.Fatal("etapa inválida")
		}
	}
	s.VersionEsperada = 7
	if (MaterialPropuestaFormalizacion{Etapa: "consulta", Solicitud: s}).Validar() != nil {
		t.Fatal("intención N rechazada")
	}
	s.VersionEsperada = 5
	if (MaterialPropuestaFormalizacion{Etapa: "consulta", Solicitud: s}).Validar() == nil {
		t.Fatal("versión previa al expediente fiscalizado admitida")
	}
	s.VersionEsperada = 6
	s.Anexos = []AnexoPropuestaFormalizacion{anexoPropuestaFormalizacionPrueba("x", "a", 1)}
	if (MaterialPropuestaFormalizacion{Etapa: "consulta", Solicitud: s}).Validar() == nil {
		t.Fatal("anexo fuera de este corte")
	}
}
