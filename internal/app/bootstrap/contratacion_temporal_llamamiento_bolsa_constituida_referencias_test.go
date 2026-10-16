package bootstrap

import (
	"strings"
	"testing"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestReferenciasConsumoCTAdmitenNonceHexadecimalSinAbrirOtrasFamilias(t *testing.T) {
	// La coincidencia con el patrón de DNI es accidental: todos los valores
	// son referencias sintéticas emitidas por el sistema, sin persona detrás.
	nonce := "12345678a" + strings.Repeat("0", 23)
	decision := "decision:" + nonce
	auditoria := "aud_v3_" + nonce
	evento := "evento:bolsa:ct:" + strings.Repeat("b", 64)
	if bolsa.ReferenciaOpacaLlamamientoValida(decision) || bolsa.ReferenciaOpacaLlamamientoValida(auditoria) {
		t.Fatal("el vector debe reproducir el falso positivo del filtro general")
	}
	if !referenciasConsumoLlamamientoCTValidas(decision, auditoria, evento) {
		t.Fatal("las referencias V3 y el evento propio fueron rechazados")
	}
	casos := []struct{ nombre, decision, auditoria, evento string }{
		{"decision_otra_familia", "decision_" + nonce, auditoria, evento},
		{"decision_corta", decision[:len(decision)-1], auditoria, evento},
		{"decision_mayuscula", "decision:" + "A" + nonce[1:], auditoria, evento},
		{"decision_no_hex", "decision:" + "g" + nonce[1:], auditoria, evento},
		{"auditoria_otra_familia", decision, "aud_v3_i_" + nonce, evento},
		{"auditoria_corta", decision, auditoria[:len(auditoria)-1], evento},
		{"auditoria_mayuscula", decision, "aud_v3_" + "A" + nonce[1:], evento},
		{"auditoria_no_hex", decision, "aud_v3_" + "g" + nonce[1:], evento},
		{"evento_ajeno", decision, auditoria, "dni:12345678A"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if referenciasConsumoLlamamientoCTValidas(caso.decision, caso.auditoria, caso.evento) {
				t.Fatal("referencia fuera del contrato aceptada")
			}
		})
	}
}

func TestReferenciasConsumoCTPasanFirmanteYVerificadorReales(t *testing.T) {
	p, ctx, d, reloj := puenteBolsaPrueba(t)
	ahora := reloj.Ahora()
	huella := strings.Repeat("a", 64)
	necesidad := referenciaVersionadaPuenteLlamamientoDesarrollo("necesidad:prueba", 1, huella)
	refBolsa := referenciaVersionadaPuenteLlamamientoDesarrollo("bolsa:prueba", 1, huella)
	politica := referenciaVersionadaPuenteLlamamientoDesarrollo("politica:prueba", 1, huella)
	refOrden := referenciaVersionadaPuenteLlamamientoDesarrollo("instantanea:prueba", 1, huella)
	contextoOrden, err := p.contexto(ctx, d, d.operacionOrden, bolsa.AccionConsultarOrigenCT, refBolsa)
	if err != nil {
		t.Fatal(err)
	}
	comandoOrden := ct.ComandoPrepararOrdenBolsa{
		Contexto: contextoOrden, Necesidad: necesidad, Bolsa: refBolsa,
		Politica: politica, MaximoPosiciones: 5,
	}
	datosOrden, err := contextoOrden.DatosEn(ahora)
	if err != nil {
		t.Fatal(err)
	}
	preparado := preparacionOrigenConstituidoCT{
		origen: origenConstituidoVerificadoCT{
			registro: bolsa.OrigenLlamamientoCT{
				InstantaneaRef: refOrden.Referencia, VersionInstantanea: 1,
				HuellaInstantaneaSHA256: huella,
			},
			auditoriaRef: "aud_v3_i_" + strings.Repeat("1", 32),
		},
		refNecesidad: necesidad, refBolsa: refBolsa, refPolitica: politica,
	}
	reciboOrden := ct.ReciboOrdenBolsa{
		OperacionRef: datosOrden.OperacionRef, OrganizacionRef: datosOrden.OrganizacionRef,
		ExpedienteRef: datosOrden.ExpedienteRef, VersionExpediente: datosOrden.VersionExpediente,
		CorrelacionRef: datosOrden.CorrelacionRef, Necesidad: necesidad, Bolsa: refBolsa,
		Politica:      politica,
		Resultado:     referenciaVersionadaPuenteLlamamientoDesarrollo("resultado:orden:prueba", 1, huella),
		OrdenGenerada: true, OrdenCompleta: true, OrdenAtestada: true,
		Orden: refOrden, AccionLlamamiento: referenciaCatalogoPuenteLlamamientoDesarrollo(bolsa.AccionEmitirLlamamiento),
		TotalPosiciones: 5, AuditoriaRef: preparado.origen.auditoriaRef,
		ConfirmadaEn: ahora,
		Procedencia:  p.procedenciaOrigenConstituidoCT(datosOrden, preparado.origen, "orden", ahora),
	}
	firmadoOrden, err := p.emisorRespuesta.FirmarOrden(ctx, comandoOrden, reciboOrden, ahora)
	if err != nil {
		t.Fatalf("firma real del orden: %v", err)
	}
	comprobante, _, err := p.verificador.VerificarReciboOrden(ctx, comandoOrden, firmadoOrden, ahora)
	if err != nil {
		t.Fatalf("verificación real del orden: %v", err)
	}
	contextoLlamamiento, err := p.contexto(ctx, d, d.operacionPropuesta, bolsa.AccionEmitirLlamamiento, refOrden)
	if err != nil {
		t.Fatal(err)
	}
	comando, err := ct.NuevoComandoSolicitarLlamamientoBolsa(ct.PreparacionComandoSolicitarLlamamientoBolsa{
		Contexto: contextoLlamamiento, ComandoOrden: comandoOrden,
		ReciboOrden: firmadoOrden, ComprobanteOrden: comprobante, MaximaPosicionEvaluable: 5,
	}, ahora)
	if err != nil {
		t.Fatalf("comando autenticado: %v", err)
	}
	datosLlamamiento, err := contextoLlamamiento.DatosEn(ahora)
	if err != nil {
		t.Fatal(err)
	}
	nonce := "12345678a" + strings.Repeat("0", 23)
	r := &bolsa.ReciboOrigenEmisionCT{
		OrigenEmisionCT: bolsa.OrigenEmisionCT{
			ParticipacionRef: "participacion:prueba", CandidatoRef: "candidato:prueba",
			OrdenSeleccionado: 5,
		},
		LlamamientoRef: "llamamiento:prueba", ReciboRef: "recibo:llamamiento:prueba",
		DecisionRef: "decision:" + nonce, AuditoriaRef: "aud_v3_" + nonce,
		EventoRef: "evento:bolsa:ct:" + strings.Repeat("b", 64), EmitidoEn: ahora,
	}
	firmado, err := p.firmarLlamamientoConstituidoCT(ctx, comando, datosLlamamiento, preparado,
		bolsa.EmisionLlamamiento{OrigenCT: r})
	if err != nil || firmado.Propuesta.Referencia != r.DecisionRef ||
		firmado.AuditoriaRef != r.AuditoriaRef || firmado.OrdenSeleccionado != 5 {
		t.Fatalf("firma real rechazó referencias V3: %v", err)
	}
	if _, _, err := p.verificador.VerificarReciboLlamamiento(ctx, comando, firmado, ahora); err != nil {
		t.Fatalf("verificación real del recibo: %v", err)
	}
	firmado.Propuesta.Referencia = "decision:alterada"
	if _, _, err := p.verificador.VerificarReciboLlamamiento(ctx, comando, firmado, ahora); err == nil {
		t.Fatal("el verificador aceptó una decisión distinta después de firmar")
	}
}
