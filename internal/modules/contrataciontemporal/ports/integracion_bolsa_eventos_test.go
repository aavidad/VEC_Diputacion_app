package ports

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEventoBolsaExigeDigestExactoYComprobanteTCB(t *testing.T) {
	base := instanteBolsaPrueba()
	ahora := base.Add(3 * time.Minute)
	sellador := &selladorHMACBolsaPrueba{clave: []byte("clave-eventos-bolsa")}
	evento := eventoBolsaPrueba(t, base, sellador)

	if err := evento.ValidarEn(ahora); err != nil {
		t.Fatalf("evento válido rechazado: %v", err)
	}
	verificador, _ := NuevoVerificadorEvidenciaIntegracionBolsa(sellador)
	comprobante, err := verificador.VerificarEvento(context.Background(), evento, ahora)
	if err != nil || comprobante.datos == nil {
		t.Fatalf("evento auténtico no promovido: %v", err)
	}
	comando, err := NuevoComandoRegistrarEventoBolsa(evento, comprobante)
	if err != nil {
		t.Fatalf("comando verificado rechazado: %v", err)
	}
	recuperado, prueba, err := comando.Datos()
	if err != nil || recuperado != evento || prueba.datos == nil {
		t.Fatalf("comando no conserva evento y prueba: %v", err)
	}

	if _, err := NuevoComandoRegistrarEventoBolsa(
		evento, ComprobanteEvidenciaIntegracionBolsa{},
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("evento sin TCB obtuvo comando: %v", err)
	}

	huellaFalsa := evento
	huellaFalsa.HuellaCargaSHA256 = referenciaBolsaPrueba("ignorada:huella", "4").HuellaSHA256
	if err := huellaFalsa.ValidarEn(ahora); !errors.Is(err, ErrEventoBolsaInvalido) {
		t.Fatalf("digest declarado ajeno aceptado: %v", err)
	}

	mutada := evento
	mutada.Estado = referenciaBolsaPrueba("estado:otro", "4")
	mutada.HuellaCargaSHA256 = huellaBytesBolsa(materialEventoBolsa(mutada))
	if err := mutada.ValidarEn(ahora); err != nil {
		t.Fatalf("evento mutado y autoconsistente debía seguir nominal: %v", err)
	}
	if _, err := verificador.VerificarEvento(
		context.Background(), mutada, ahora,
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("evento mutado conservó autenticidad: %v", err)
	}
}

func TestVectorCanonicoEventoBolsaEsEstable(t *testing.T) {
	base := instanteBolsaPrueba()
	sellador := &selladorHMACBolsaPrueba{clave: []byte("clave-vector-bolsa")}
	evento := eventoBolsaPrueba(t, base, sellador)
	const esperado = "877e5f4625a095a4b8b1033c9f03f5797af9739529f2e9d70679d4e05f7e4aaa"
	if evento.HuellaCargaSHA256 != esperado {
		t.Fatalf("vector canónico evento=%s", evento.HuellaCargaSHA256)
	}
}

func TestInboxIdentificaAutoridadEventoYRechazaColision(t *testing.T) {
	base := instanteBolsaPrueba()
	sellador := &selladorHMACBolsaPrueba{clave: []byte("clave-colision-bolsa")}
	evento := eventoBolsaPrueba(t, base, sellador)
	identico := evento
	if err := ValidarIdentidadEventoBolsa(evento, identico); err != nil {
		t.Fatalf("replay idéntico rechazado: %v", err)
	}

	mutado := evento
	mutado.Estado = referenciaBolsaPrueba("estado:rectificado", "4")
	mutado.HuellaCargaSHA256 = huellaBytesBolsa(materialEventoBolsa(mutado))
	if err := ValidarIdentidadEventoBolsa(evento, mutado); !errors.Is(err, ErrColisionEventoBolsa) {
		t.Fatalf("misma identidad con otra carga no fue colisión: %v", err)
	}

	mutado = evento
	mutado.Estado = referenciaBolsaPrueba("estado:rectificado", "4")
	// Incluso una colisión criptográfica declarada no oculta bytes distintos.
	mutado.HuellaCargaSHA256 = evento.HuellaCargaSHA256
	if err := ValidarIdentidadEventoBolsa(evento, mutado); !errors.Is(err, ErrColisionEventoBolsa) {
		t.Fatalf("bytes distintos con digest declarado igual no fueron colisión: %v", err)
	}

	otraAutoridad := evento
	otraAutoridad.Procedencia.AutoridadRef = "autoridad:otra"
	if err := ValidarIdentidadEventoBolsa(evento, otraAutoridad); !errors.Is(err, ErrEventoBolsaInvalido) {
		t.Fatalf("otra autoridad compartió identidad de inbox: %v", err)
	}
}

func TestSecuenciaCASExactoYReplayDevuelvenMismoAcuse(t *testing.T) {
	base := instanteBolsaPrueba()
	sellador := &selladorHMACBolsaPrueba{clave: []byte("clave-acuse-bolsa")}
	evento := eventoBolsaPrueba(t, base, sellador)
	acuse := acuseEventoBolsaPrueba(evento, base.Add(4*time.Minute))
	if err := acuse.ValidarPara(evento); err != nil {
		t.Fatalf("acuse exacto rechazado: %v", err)
	}
	if err := ValidarReplayAcuseEventoBolsa(acuse, acuse, evento); err != nil {
		t.Fatalf("replay no devolvió mismo acuse: %v", err)
	}

	repetidoDistinto := acuse
	repetidoDistinto.InboxRef = "inbox:otro"
	if err := ValidarReplayAcuseEventoBolsa(
		acuse, repetidoDistinto, evento,
	); !errors.Is(err, ErrAcuseEventoBolsaNoConfiable) {
		t.Fatalf("replay fabricó otro acuse: %v", err)
	}

	for _, cambiar := range []func(*AcuseEventoLlamamientoBolsa){
		func(a *AcuseEventoLlamamientoBolsa) { a.AutoridadRef = "autoridad:otra" },
		func(a *AcuseEventoLlamamientoBolsa) { a.OrganizacionRef = "organizacion:otra" },
		func(a *AcuseEventoLlamamientoBolsa) { a.ExpedienteRef = "expediente:otro" },
		func(a *AcuseEventoLlamamientoBolsa) { a.CorrelacionRef = "correlacion:otra" },
		func(a *AcuseEventoLlamamientoBolsa) {
			a.HuellaEventoSHA256 = referenciaBolsaPrueba("huella:otra", "3").HuellaSHA256
		},
		func(a *AcuseEventoLlamamientoBolsa) { a.Secuencia++ },
		func(a *AcuseEventoLlamamientoBolsa) { a.VersionAnterior++ },
		func(a *AcuseEventoLlamamientoBolsa) { a.VersionResultante++ },
	} {
		alterado := acuse
		cambiar(&alterado)
		if err := alterado.ValidarPara(evento); !errors.Is(err, ErrAcuseEventoBolsaNoConfiable) {
			t.Fatalf("acuse no ligado aceptado: %+v err=%v", alterado, err)
		}
	}

	salto := evento
	salto.Secuencia++
	salto.HuellaCargaSHA256 = huellaBytesBolsa(materialEventoBolsa(salto))
	if err := salto.ValidarEn(base.Add(3 * time.Minute)); !errors.Is(err, ErrEventoBolsaInvalido) {
		t.Fatalf("salto de secuencia aceptado: %v", err)
	}
}

func TestTodosLosUint64DeIntegracionSonTransportables(t *testing.T) {
	base := instanteBolsaPrueba()
	ahora := base.Add(3 * time.Minute)

	referencia := referenciaBolsaPrueba("referencia:segura", "a")
	referencia.Version = MaximoEnteroSeguroIntegracionBolsa + 1
	if referencia.Validar() == nil {
		t.Fatal("versión no transportable aceptada")
	}

	contexto := contextoBolsaPrueba(base)
	contexto.VersionExpediente = MaximoEnteroSeguroIntegracionBolsa + 1
	if contexto.ValidarEn(ahora) == nil {
		t.Fatal("versión de expediente no transportable aceptada")
	}
	contexto = contextoBolsaPrueba(base)
	contexto.ContratoVersion = MaximoEnteroSeguroIntegracionBolsa + 1
	if contexto.ValidarEn(ahora) == nil {
		t.Fatal("versión de contrato no transportable aceptada")
	}

	solicitud := solicitudDisponibilidadPrueba(base)
	resultado := resultadoDisponibilidadPrueba(base)
	resultado.Procedencia.ContratoVersion = MaximoEnteroSeguroIntegracionBolsa + 1
	if resultado.ValidarParaEn(solicitud, ahora) == nil {
		t.Fatal("procedencia con versión no transportable aceptada")
	}

	sellador := &selladorHMACBolsaPrueba{clave: []byte("clave-enteros-bolsa")}
	evento := eventoBolsaPrueba(t, base, sellador)
	evento.VersionExpedienteEsperada = MaximoEnteroSeguroIntegracionBolsa
	evento.HuellaCargaSHA256 = huellaBytesBolsa(materialEventoBolsa(evento))
	if evento.ValidarEn(ahora) == nil {
		t.Fatal("versión sin sucesor transportable aceptada")
	}
	evento = eventoBolsaPrueba(t, base, sellador)
	evento.Secuencia = MaximoEnteroSeguroIntegracionBolsa + 1
	evento.SecuenciaAnterior = MaximoEnteroSeguroIntegracionBolsa
	evento.HuellaCargaSHA256 = huellaBytesBolsa(materialEventoBolsa(evento))
	if evento.ValidarEn(ahora) == nil {
		t.Fatal("secuencia no transportable aceptada")
	}

	acuse := acuseEventoBolsaPrueba(eventoBolsaPrueba(t, base, sellador), base.Add(4*time.Minute))
	acuse.VersionResultante = MaximoEnteroSeguroIntegracionBolsa + 1
	if acuse.ValidarPara(eventoBolsaPrueba(t, base, sellador)) == nil {
		t.Fatal("acuse con entero no transportable aceptado")
	}
}

func eventoBolsaPrueba(
	t *testing.T,
	instante time.Time,
	sellador *selladorHMACBolsaPrueba,
) EventoLlamamientoBolsa {
	t.Helper()
	procedencia := procedenciaBolsaPrueba(instante)
	procedencia.RespuestaRef = "respuesta:evento:bolsa"
	evento := EventoLlamamientoBolsa{
		EventoRef: "evento:estado:llamamiento", OrganizacionRef: "organizacion:diputacion",
		ExpedienteRef: "expediente:temporal", VersionExpedienteEsperada: 5,
		CorrelacionRef: "correlacion:temporal",
		Necesidad:      referenciaBolsaPrueba("necesidad:temporal", "a"),
		Bolsa:          referenciaBolsaPrueba("bolsa:vigente", "b"),
		Orden:          referenciaBolsaPrueba("orden:instantanea", "e"),
		Politica:       referenciaBolsaPrueba("politica:llamamiento", "c"),
		Propuesta:      referenciaBolsaPrueba("propuesta:primera", "d"),
		LlamamientoRef: "llamamiento:primero", SeleccionRef: "seleccion:seudonimizada",
		RetencionSeleccion: referenciaBolsaPrueba("retencion:seleccion", "6"),
		Tipo:               referenciaBolsaPrueba("evento_tipo:estado_llamamiento", "5"),
		Estado:             referenciaBolsaPrueba("estado:llamamiento", "7"),
		SecuenciaAnterior:  1, Secuencia: 2,
		HuellaCargaSHA256: referenciaBolsaPrueba("huella:temporal", "4").HuellaSHA256,
		OcurridoEn:        instante.Add(time.Minute), PublicadoEn: instante.Add(2 * time.Minute),
		Procedencia: procedencia,
	}
	evento.HuellaCargaSHA256 = huellaBytesBolsa(materialEventoBolsa(evento))
	sello, err := sellador.SellarDatos(context.Background(), materialEventoBolsa(evento))
	if err != nil {
		t.Fatalf("sellar evento: %v", err)
	}
	evento.Procedencia.Evidencia.SelloHMAC = sello
	return evento
}

func acuseEventoBolsaPrueba(
	evento EventoLlamamientoBolsa,
	instante time.Time,
) AcuseEventoLlamamientoBolsa {
	return AcuseEventoLlamamientoBolsa{
		AutoridadRef: evento.Procedencia.AutoridadRef, EventoRef: evento.EventoRef,
		OrganizacionRef: evento.OrganizacionRef, ExpedienteRef: evento.ExpedienteRef,
		CorrelacionRef: evento.CorrelacionRef, HuellaEventoSHA256: evento.HuellaCargaSHA256,
		SecuenciaAnterior: evento.SecuenciaAnterior, Secuencia: evento.Secuencia,
		VersionAnterior:   evento.VersionExpedienteEsperada,
		VersionResultante: evento.VersionExpedienteEsperada + 1,
		ActuacionRef:      "actuacion:llamamiento", AuditoriaRef: "auditoria:llamamiento",
		InboxRef: "inbox:evento:llamamiento", RegistradoEn: instante,
	}
}
