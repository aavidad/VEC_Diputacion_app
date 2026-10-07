package domain

import (
	"strings"
	"testing"
	"time"
)

func TestCircuitoRRHHLigaDosHitosAlActoSinVersionArtificial(t *testing.T) {
	base := expedienteConAnalisisRehidratado(t, RCValidada)
	inicial := base.Actuaciones[0].FaseDestino
	definicion, err := NuevaDefinicionCircuitoRRHH(
		"flujo:ct:rrhh:prueba", 2, inicial,
		[]TransicionCircuitoRRHH{
			{Clave: "peticion_firmada", Tipo: HitoPeticionFirmada, Origen: inicial,
				Destino: "autorizacion_rrhh", RequiereDocumento: true, RequiereFirma: true,
				PerfilClave: "tecnico", FirmasRequeridas: []ClaveCatalogo{"tecnico", "delegacion"}},
			{Clave: "autorizacion_rrhh", Tipo: HitoAutorizacionRRHH, Origen: "autorizacion_rrhh",
				Destino: "credito", RequiereDocumento: true, PerfilClave: "direccion_rrhh",
				AutorizanteCargoClave: "direccion_rrhh"},
			{Clave: "credito_comprobado", Tipo: HitoCreditoComprobado, Origen: "credito",
				Destino: "oferta", RequiereDocumento: true, PerfilClave: "rrhh"},
			{Clave: "oferta_emitida", Tipo: HitoOfertaEmitida, Origen: "oferta",
				Destino: "adjudicacion", PerfilClave: "rrhh"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	base.Flujo = definicion.Flujo
	circuito, err := NuevoCircuitoAdministrativo(definicion)
	if err != nil {
		t.Fatal(err)
	}
	base.Circuito = &circuito
	if err := base.Validar(); err != nil {
		t.Fatalf("alta ligada al triple: %v", err)
	}
	act := base.Actuaciones[len(base.Actuaciones)-1]
	ahora := act.RealizadaEn
	peticion := HitoCircuitoRRHH{
		Clave: "peticion_firmada", ActuacionClave: act.AccionClave, ActorRef: act.ActorRef,
		PerfilClave: "tecnico", PerfilRef: "perfil:tecnico:sintetico",
		UnidadRef: act.UnidadRef, DocumentoRef: "documento:peticion:sintetica", HuellaDocumentoSHA256: strings.Repeat("a", 64),
		Firmas: []FirmaCircuitoRRHH{
			{CargoClave: "tecnico", FirmaRef: "firma:tecnico:sintetica", FirmanteRef: "actor:tecnico:sintetico", HuellaDocumentoSHA256: strings.Repeat("a", 64)},
			{CargoClave: "delegacion", FirmaRef: "firma:delegacion:sintetica", FirmanteRef: "actor:delegacion:sintetico", HuellaDocumentoSHA256: strings.Repeat("a", 64)},
		},
		ReciboRef: act.ReciboRef, RegistradoEn: ahora,
	}
	autorizacion := HitoCircuitoRRHH{
		Clave: "autorizacion_rrhh", ActuacionClave: act.AccionClave, ActorRef: act.ActorRef,
		PerfilClave: "direccion_rrhh", PerfilRef: "perfil:direccion_rrhh:sintetico",
		UnidadRef: act.UnidadRef, DocumentoRef: "documento:autorizacion:sintetica", HuellaDocumentoSHA256: strings.Repeat("b", 64),
		ActoAutorizacionRef: "acto:autorizacion:sintetico", AutorizanteRef: "actor:direccion:sintetico",
		CargoAutorizanteClave: "direccion_rrhh",
		ReciboRef:             act.ReciboRef, RegistradoEn: ahora,
	}
	conHitos, err := base.AdjuntarHitosCircuito(definicion, base.Version, []HitoCircuitoRRHH{peticion, autorizacion})
	if err != nil || conHitos.Validar() != nil {
		t.Fatalf("dos hitos en v2: %v", err)
	}
	if conHitos.Version != base.Version || len(conHitos.Circuito.Hitos) != 2 {
		t.Fatal("adjuntar hitos creó otra versión o perdió la historia")
	}
	datosCredito, err := conHitos.DatosCreditoCircuitoRRHH()
	if err != nil || datosCredito.CreditoRef != conHitos.Analisis.ValidacionRC.ReciboRef ||
		datosCredito.DocumentoRef != conHitos.Analisis.ValidacionRC.DocumentoRef {
		t.Fatalf("el crédito no procede del análisis validado: %v", err)
	}
	oferta := HitoCircuitoRRHH{
		Clave: "oferta_emitida", ActuacionClave: act.AccionClave, ActorRef: act.ActorRef,
		PerfilClave: "rrhh", PerfilRef: "perfil:rrhh:sintetico",
		UnidadRef: act.UnidadRef, OfertaRef: "oferta:sintetica", ReciboRef: act.ReciboRef,
		RegistradoEn: ahora,
	}
	if _, err := conHitos.AdjuntarHitosCircuito(definicion, conHitos.Version, []HitoCircuitoRRHH{oferta}); err == nil {
		t.Fatal("una oferta sin credito avanzó")
	}
	conVia, err := conHitos.RegistrarViaCobertura(conHitos.Version, decisionValida(),
		actuacion("cobertura.decidida", "asignacion_unidad", instanteBase.Add(2*time.Minute)))
	if err != nil {
		t.Fatalf("decisión de cobertura: %v", err)
	}
	if _, err := conVia.DatosCreditoCircuitoRRHH(); err == nil {
		t.Fatal("una decisión ya confirmada volvió a pedir hito de crédito")
	}
	actCredito := conVia.Actuaciones[len(conVia.Actuaciones)-1]
	credito := HitoCircuitoRRHH{
		Clave: "credito_comprobado", ActuacionClave: actCredito.AccionClave,
		ActorRef: actCredito.ActorRef, PerfilClave: "rrhh", PerfilRef: "perfil:rrhh:sintetico",
		UnidadRef:             actCredito.UnidadRef,
		DocumentoRef:          conVia.Analisis.ValidacionRC.DocumentoRef,
		DocumentoVersion:      1,
		HuellaDocumentoSHA256: strings.Repeat("c", 64),
		CreditoRef:            conVia.Analisis.ValidacionRC.ReciboRef,
		ReciboRef:             actCredito.ReciboRef, RegistradoEn: actCredito.RealizadaEn,
	}
	conCredito, err := conVia.AdjuntarHitosCircuito(definicion, conVia.Version, []HitoCircuitoRRHH{credito})
	if err != nil || conCredito.Validar() != nil || conCredito.Version != conVia.Version ||
		len(conCredito.Circuito.Hitos) != 3 || conCredito.Circuito.EstadoActual != "oferta" {
		t.Fatalf("crédito debía ligarse a la decisión v3: %v", err)
	}
	credito.DocumentoRef = "documento:otro:sintetico"
	if _, err := conVia.AdjuntarHitosCircuito(definicion, conVia.Version, []HitoCircuitoRRHH{credito}); err == nil {
		t.Fatal("un hito con documento ajeno a la RC avanzó")
	}
	credito.DocumentoRef = conVia.Analisis.ValidacionRC.DocumentoRef
	credito.DocumentoVersion = 0
	if _, err := conVia.AdjuntarHitosCircuito(definicion, conVia.Version, []HitoCircuitoRRHH{credito}); err == nil {
		t.Fatal("un hito sin versión documental avanzó")
	}
	adulterado := conHitos.Clonar()
	adulterado.Circuito.Definicion.HuellaSHA256 = strings.Repeat("b", 64)
	if adulterado.Validar() == nil {
		t.Fatal("el circuito aceptó otro triple")
	}
}

func TestInformeCircuitoRRHHExigeOfertaYAdjudicacion(t *testing.T) {
	expediente := expedienteConAsignacion(t)
	inicial := expediente.Actuaciones[0].FaseDestino
	definicion, err := NuevaDefinicionCircuitoRRHH(
		"flujo:ct:rrhh:sin-adjudicacion", 2, inicial,
		[]TransicionCircuitoRRHH{{Clave: "contratacion_temporal.circuito.peticion_firmada",
			Tipo: HitoPeticionFirmada, Origen: inicial, Destino: "autorizacion_rrhh",
			RequiereDocumento: true, RequiereFirma: true, PerfilClave: "tecnico_rrhh",
			FirmasRequeridas: []ClaveCatalogo{"tecnico_solicitante", "delegacion_solicitante"}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	circuito, err := NuevoCircuitoAdministrativo(definicion)
	if err != nil {
		t.Fatal(err)
	}
	expediente.Flujo, expediente.Circuito = definicion.Flujo, &circuito
	if expediente.Validar() != nil {
		t.Fatal("preimagen nueva inválida")
	}
	datosBorrador := datosInformeJuridicoPrueba()
	datosBorrador.ExpedienteRef = expediente.Referencia
	datosBorrador.VersionEsperadaExpediente = expediente.Version
	borrador, err := NuevoBorradorInformeJuridico(datosBorrador)
	if err != nil {
		t.Fatal(err)
	}
	instante := expediente.ActualizadoEn.Add(time.Minute)
	informe := InformeJuridicoEmitido{
		Borrador: borrador.Estado(), InformeRef: "informe:rrhh:sintetico",
		DocumentoRef:     "documento:informe:rrhh:sintetico",
		VersionDocumento: 1, HuellaDocumentoSHA256: strings.Repeat("a", 64),
		EmitidoEn: instante,
	}
	_, err = expediente.RegistrarInformeJuridico(expediente.Version, informe, DatosActuacion{
		AccionClave: AccionEmitirInformeJuridico, ActorRef: "actor:jefatura:sintetico",
		UnidadRef: expediente.Asignacion.UnidadRef, ReciboRef: "recibo:informe:rrhh:sintetico",
		RealizadaEn: instante, FaseDestino: FaseInformeJuridico,
		EstadoDestino: EstadoEnCurso, DocumentosRef: []string{informe.DocumentoRef},
	})
	if err == nil {
		t.Fatal("el informe avanzó sin oferta y adjudicación verificadas")
	}
}

func TestFiscalizacionCircuitoRRHHExigeFirmaJefaturaAcreditada(t *testing.T) {
	expediente := expedienteFiscalizablePrueba(t)
	inicial := expediente.Actuaciones[0].FaseDestino
	definicion, err := NuevaDefinicionCircuitoRRHH(
		"flujo:ct:rrhh:fiscalizacion", 2, inicial,
		[]TransicionCircuitoRRHH{{Clave: "contratacion_temporal.circuito.peticion_firmada",
			Tipo: HitoPeticionFirmada, Origen: inicial, Destino: "autorizacion_rrhh",
			RequiereDocumento: true, RequiereFirma: true, PerfilClave: "tecnico_rrhh",
			FirmasRequeridas: []ClaveCatalogo{"tecnico_solicitante", "delegacion_solicitante"}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	circuito, err := NuevoCircuitoAdministrativo(definicion)
	if err != nil {
		t.Fatal(err)
	}
	expediente.Flujo, expediente.Circuito = definicion.Flujo, &circuito
	if expediente.Validar() != nil {
		t.Fatal("preimagen nueva inválida")
	}
	instante := expediente.ActualizadoEn.Add(time.Minute)
	actuacion := DatosActuacion{
		AccionClave: AccionRegistrarFiscalizacion,
		ActorRef:    "actor:intervencion:sintetico", UnidadRef: "unidad:intervencion:sintetica",
		ReciboRef: "recibo:fiscalizacion:sintetica", RealizadaEn: instante,
		FaseDestino: FaseFiscalizacion, EstadoDestino: EstadoEnCurso,
		DocumentosRef: []string{expediente.InformeJuridico.DocumentoRef},
	}
	_, err = expediente.RegistrarFiscalizacion(expediente.Version, DatosRegistrarFiscalizacion{
		FiscalizacionRef: "fiscalizacion:sintetica", Resultado: FiscalizacionFavorable,
		UnidadFiscalizadoraRef: actuacion.UnidadRef, FiscalizadaEn: instante,
	}, actuacion)
	if err == nil {
		t.Fatal("fiscalización nueva avanzó sin firma y cargo de Jefatura acreditados")
	}
}

func TestFirmasCircuitoRRHHExigeAlternativaExactaYOrden(t *testing.T) {
	const huella = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	firma := func(cargo ClaveCatalogo) FirmaCircuitoRRHH {
		return FirmaCircuitoRRHH{CargoClave: cargo, FirmaRef: "firma:rrhh:sintetica",
			FirmanteRef: "actor:rrhh:sintetico", HuellaDocumentoSHA256: huella}
	}
	paraJefatura := []ClaveCatalogo{"jefatura_servicio_rrhh", "diputacion_delegada_rrhh"}
	paraDireccion := []ClaveCatalogo{"direccion_rrhh", "diputacion_delegada_rrhh"}
	if !firmasCircuitoCoinciden(paraJefatura, []FirmaCircuitoRRHH{firma(paraJefatura[0]), firma(paraJefatura[1])}) ||
		!firmasCircuitoCoinciden(paraDireccion, []FirmaCircuitoRRHH{firma(paraDireccion[0]), firma(paraDireccion[1])}) {
		t.Fatal("ambos vistos buenos alternativos deben preceder a la firma de la Diputada")
	}
	for nombre, firmas := range map[string][]FirmaCircuitoRRHH{
		"tres firmas":   {firma(paraJefatura[0]), firma(paraDireccion[0]), firma(paraJefatura[1])},
		"orden inverso": {firma(paraJefatura[1]), firma(paraJefatura[0])},
		"cargo ajeno":   {firma("intervencion"), firma(paraJefatura[1])},
	} {
		if firmasCircuitoCoinciden(paraJefatura, firmas) || firmasCircuitoCoinciden(paraDireccion, firmas) {
			t.Fatalf("%s no debe habilitar la resolución", nombre)
		}
	}
}
