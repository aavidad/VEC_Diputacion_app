package domain

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

type revalidadorVinculoAutenticacionActorPrueba struct {
	resultado AutenticacionRevalidadaV1
	err       error
}

func (r revalidadorVinculoAutenticacionActorPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	SolicitudRevalidacionAutenticacionActorV1,
) (AutenticacionRevalidadaV1, error) {
	return r.resultado, r.err
}

func TestDatosVinculoAutenticacionActorV1EnumeraExactamenteLosVeinticincoCampos(t *testing.T) {
	esperados := []string{
		"bloque_version", "autenticacion_ref", "autenticacion_huella_sha256", "asercion_ref",
		"sesion_ref", "control_sesion_ref", "control_sesion_revision", "control_sesion_huella_sha256",
		"cuenta_ref", "cuenta_ordinaria_ref", "principal_id", "perfil_activo_ref",
		"cuenta_privilegiada", "superficie", "metodo_observado",
		"garantia_observada", "politica_garantia_ref", "politica_garantia_huella_sha256",
		"autenticacion_verificada_en", "sesion_emitida_en", "sesion_valida_hasta",
		"sesion_revalidada_en", "contexto_actor_ref", "contexto_actor_version",
		"contexto_actor_huella_sha256",
	}
	tipo := reflect.TypeOf(DatosVinculoAutenticacionActorV1{})
	if tipo.NumField() != len(esperados) {
		t.Fatalf("campos=%d; esperados=%d", tipo.NumField(), len(esperados))
	}
	for indice, esperado := range esperados {
		campo := tipo.Field(indice)
		if etiqueta := strings.Split(campo.Tag.Get("json"), ",")[0]; etiqueta != esperado || campo.PkgPath != "" {
			t.Fatalf("campo %d = %s/%q; esperado %q", indice, campo.Name, etiqueta, esperado)
		}
	}
}

func TestVinculoAutenticacionActorV1CreaCapacidadOpacaYCruzaActor(t *testing.T) {
	instante := time.Date(2026, 7, 15, 10, 0, 0, 123_456_000, time.UTC)
	actor := contextoActorVinculoPrueba(t, instante)
	autenticacion := autenticacionRevalidadaVinculoPrueba(instante)
	solicitud := solicitudRevalidacionVinculoPrueba(autenticacion)
	vinculo, err := CrearVinculoAutenticacionActorV1(
		context.Background(), revalidadorVinculoAutenticacionActorPrueba{resultado: autenticacion},
		solicitud, actor, instante,
	)
	if err != nil {
		t.Fatalf("crear vinculo: %v", err)
	}
	if vinculo.ValidarPara(actor) != nil || !vinculo.VigenteEn(instante, actor) {
		t.Fatal("el vinculo emitido no quedo ligado al actor")
	}
	datos, err := vinculo.Datos()
	if err != nil || datos.BloqueVersion != 1 || datos.ContextoActorRef != actor.Instantanea.VinculoRef ||
		datos.PrincipalID != actor.Principal.ID || datos.PerfilActivoRef != actor.PerfilActivoRef {
		t.Fatalf("datos inesperados: %+v, %v", datos, err)
	}

	contenido, err := json.Marshal(vinculo)
	if err != nil || !strings.Contains(string(contenido), `"cuenta_privilegiada":false`) ||
		!strings.Contains(string(contenido), `"principal_id":"`+actor.Principal.ID+`"`) ||
		!strings.Contains(string(contenido), `"perfil_activo_ref":"`+actor.PerfilActivoRef+`"`) {
		t.Fatalf("serializacion de evidencia: %s, %v", contenido, err)
	}
	var reconstruido VinculoAutenticacionActorV1
	if err := json.Unmarshal(contenido, &reconstruido); !errors.Is(err, ErrReconstruccionVinculoAutenticacionActorProhibida) ||
		reconstruido.Validar() == nil {
		t.Fatalf("la capacidad se reconstruyo desde JSON: %v", err)
	}
	if (VinculoAutenticacionActorV1{}).Validar() == nil {
		t.Fatal("el valor cero fue valido")
	}
}

func TestCrearVinculoAutenticacionActorV1RechazaRevalidadorNuloTipado(t *testing.T) {
	instante := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	autenticacion := autenticacionRevalidadaVinculoPrueba(instante)
	var revalidador *revalidadorVinculoAutenticacionActorPrueba
	vinculo, err := CrearVinculoAutenticacionActorV1(
		context.Background(), revalidador, solicitudRevalidacionVinculoPrueba(autenticacion),
		contextoActorVinculoPrueba(t, instante), instante,
	)
	if !errors.Is(err, ErrVinculoAutenticacionActorInvalido) || vinculo.Validar() == nil {
		t.Fatalf("revalidador nulo tipado aceptado: %v", err)
	}
}

func TestDatosVinculoAutenticacionActorV1RechazaCadaAusencia(t *testing.T) {
	instante := time.Date(2026, 7, 15, 10, 0, 0, 123_456_000, time.UTC)
	vinculo := vinculoAutenticacionActorV1Prueba(t, instante)
	base, _ := vinculo.Datos()
	casos := []struct {
		nombre string
		mutar  func(*DatosVinculoAutenticacionActorV1)
	}{
		{"version", func(d *DatosVinculoAutenticacionActorV1) { d.BloqueVersion = 0 }},
		{"autenticacion", func(d *DatosVinculoAutenticacionActorV1) { d.AutenticacionRef = "" }},
		{"huella autenticacion", func(d *DatosVinculoAutenticacionActorV1) { d.AutenticacionHuellaSHA256 = "" }},
		{"asercion", func(d *DatosVinculoAutenticacionActorV1) { d.AsercionRef = "" }},
		{"sesion", func(d *DatosVinculoAutenticacionActorV1) { d.SesionRef = "" }},
		{"control sesion", func(d *DatosVinculoAutenticacionActorV1) { d.ControlSesionRef = "" }},
		{"revision control", func(d *DatosVinculoAutenticacionActorV1) { d.ControlSesionRevision = 0 }},
		{"huella control", func(d *DatosVinculoAutenticacionActorV1) { d.ControlSesionHuellaSHA256 = "" }},
		{"cuenta", func(d *DatosVinculoAutenticacionActorV1) { d.CuentaRef = "" }},
		{"cuenta ordinaria", func(d *DatosVinculoAutenticacionActorV1) { d.CuentaOrdinariaRef = "" }},
		{"principal", func(d *DatosVinculoAutenticacionActorV1) { d.PrincipalID = "" }},
		{"perfil activo", func(d *DatosVinculoAutenticacionActorV1) { d.PerfilActivoRef = "" }},
		{"superficie", func(d *DatosVinculoAutenticacionActorV1) { d.Superficie = "" }},
		{"metodo", func(d *DatosVinculoAutenticacionActorV1) { d.MetodoObservado = "" }},
		{"garantia", func(d *DatosVinculoAutenticacionActorV1) { d.GarantiaObservada = "" }},
		{"politica garantia", func(d *DatosVinculoAutenticacionActorV1) { d.PoliticaGarantiaRef = "" }},
		{"huella politica", func(d *DatosVinculoAutenticacionActorV1) { d.PoliticaGarantiaHuellaSHA256 = "" }},
		{"autenticacion verificada", func(d *DatosVinculoAutenticacionActorV1) { d.AutenticacionVerificadaEn = time.Time{} }},
		{"sesion emitida", func(d *DatosVinculoAutenticacionActorV1) { d.SesionEmitidaEn = time.Time{} }},
		{"sesion valida hasta", func(d *DatosVinculoAutenticacionActorV1) { d.SesionValidaHasta = time.Time{} }},
		{"sesion revalidada", func(d *DatosVinculoAutenticacionActorV1) { d.SesionRevalidadaEn = time.Time{} }},
		{"contexto actor", func(d *DatosVinculoAutenticacionActorV1) { d.ContextoActorRef = "" }},
		{"version contexto", func(d *DatosVinculoAutenticacionActorV1) { d.ContextoActorVersion = 0 }},
		{"huella contexto", func(d *DatosVinculoAutenticacionActorV1) { d.ContextoActorHuellaSHA256 = "" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			candidata := base
			caso.mutar(&candidata)
			if candidata.Validar() == nil {
				t.Fatal("ausencia aceptada")
			}
		})
	}
}

func TestVinculoAutenticacionActorV1CoincideExactamenteSinNormalizar(t *testing.T) {
	instante := time.Date(2026, 7, 15, 10, 0, 0, 123_456_000, time.UTC)
	primero := vinculoAutenticacionActorV1Prueba(t, instante)
	mismaEvidencia := vinculoAutenticacionActorV1Prueba(t, instante)
	if !primero.CoincideExactamenteCon(mismaEvidencia) ||
		!mismaEvidencia.CoincideExactamenteCon(primero) {
		t.Fatal("dos capacidades validas con los mismos datos no coincidieron")
	}
	if primero.CoincideExactamenteCon(VinculoAutenticacionActorV1{}) ||
		(VinculoAutenticacionActorV1{}).CoincideExactamenteCon(VinculoAutenticacionActorV1{}) {
		t.Fatal("un valor invalido participo en una igualdad positiva")
	}

	autenticacion := autenticacionRevalidadaVinculoPrueba(instante)
	autenticacion.ControlSesionRevision++
	autenticacion.ControlSesionHuellaSHA256 = strings.Repeat("9", 64)
	distinto, err := CrearVinculoAutenticacionActorV1(
		context.Background(), revalidadorVinculoAutenticacionActorPrueba{resultado: autenticacion},
		solicitudRevalidacionVinculoPrueba(autenticacion), contextoActorVinculoPrueba(t, instante), instante,
	)
	if err != nil {
		t.Fatalf("crear capacidad distinta: %v", err)
	}
	if primero.CoincideExactamenteCon(distinto) || distinto.CoincideExactamenteCon(primero) {
		t.Fatal("se ignoraron datos distintos del control de sesion")
	}
}

func TestVinculoAutenticacionActorV1DeniegaMezclasYEstadosNoExplicitos(t *testing.T) {
	instante := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	actor := contextoActorVinculoPrueba(t, instante)
	base := autenticacionRevalidadaVinculoPrueba(instante)
	casos := []struct {
		nombre string
		mutar  func(*AutenticacionRevalidadaV1)
	}{
		{"demo", func(a *AutenticacionRevalidadaV1) { a.MetodoObservado = AuthMethodDemo }},
		{"anonima", func(a *AutenticacionRevalidadaV1) { a.Superficie = "publica_anonima" }},
		{"cuenta mezclada", func(a *AutenticacionRevalidadaV1) { a.CuentaRef = "cta_otra234567890abcdefghijkl" }},
		{"ordinaria distinta", func(a *AutenticacionRevalidadaV1) { a.CuentaOrdinariaRef = "cta_otra234567890abcdefghijkl" }},
		{"privilegiada fuera de administracion", func(a *AutenticacionRevalidadaV1) {
			a.CuentaPrivilegiada = true
			a.CuentaOrdinariaRef = "cta_otra234567890abcdefghijkl"
		}},
		{"administracion sin privilegiada", func(a *AutenticacionRevalidadaV1) {
			a.Superficie = SuperficieAutenticacionAdministracionPrivilegiadaV1
		}},
		{"cronologia invertida", func(a *AutenticacionRevalidadaV1) {
			a.SesionRevalidadaEn = a.AutenticacionVerificadaEn.Add(-time.Microsecond)
		}},
		{"autenticacion posterior a emision", func(a *AutenticacionRevalidadaV1) {
			a.AutenticacionVerificadaEn = a.SesionEmitidaEn.Add(time.Microsecond)
		}},
		{"revalidacion anterior a emision", func(a *AutenticacionRevalidadaV1) {
			a.SesionRevalidadaEn = a.SesionEmitidaEn.Add(-time.Microsecond)
		}},
		{"submicrosegundo", func(a *AutenticacionRevalidadaV1) { a.SesionEmitidaEn = a.SesionEmitidaEn.Add(time.Nanosecond) }},
		{"ano cero", func(a *AutenticacionRevalidadaV1) {
			a.SesionEmitidaEn = time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
		}},
		{"ano 10000", func(a *AutenticacionRevalidadaV1) {
			a.SesionValidaHasta = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			autenticacion := base
			caso.mutar(&autenticacion)
			solicitud := solicitudRevalidacionVinculoPrueba(base)
			vinculo, err := CrearVinculoAutenticacionActorV1(
				context.Background(), revalidadorVinculoAutenticacionActorPrueba{resultado: autenticacion},
				solicitud, actor, instante,
			)
			if err == nil || vinculo.Validar() == nil {
				t.Fatalf("mezcla aceptada: %+v", autenticacion)
			}
		})
	}

	otroPerfil := actor
	otroPerfil.PerfilActivoRef = "prf_otra234567890abcdefghijkl"
	otroPerfil.Instantanea.PerfilActivoRef = otroPerfil.PerfilActivoRef
	otroPerfil.Instantanea.PerfilVersion++
	vinculo := vinculoAutenticacionActorV1Prueba(t, instante)
	if vinculo.ValidarPara(otroPerfil) == nil {
		t.Fatal("un contexto de otro perfil conservo el vinculo")
	}
}

func vinculoAutenticacionActorV1Prueba(t *testing.T, instante time.Time) VinculoAutenticacionActorV1 {
	t.Helper()
	vinculo, err := vinculoAutenticacionActorV1PruebaSinT(instante)
	if err != nil {
		t.Fatalf("crear vinculo de prueba: %v", err)
	}
	return vinculo
}

func TestVinculoAutenticacionActorV1RechazaContextoCanonicoV2(t *testing.T) {
	instante := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	actor := contextoActorVinculoPrueba(t, instante)
	actor.Instantanea.CuentaVersion = 1
	autenticacion := autenticacionRevalidadaVinculoPrueba(instante)
	vinculo, err := CrearVinculoAutenticacionActorV1(
		context.Background(), revalidadorVinculoAutenticacionActorPrueba{resultado: autenticacion},
		solicitudRevalidacionVinculoPrueba(autenticacion), actor, instante,
	)
	if !errors.Is(err, ErrVinculoAutenticacionActorInvalido) || vinculo.Validar() == nil {
		t.Fatalf("VinculoAutenticacionActorV1 acepto downgrade de contexto V2: %#v, %v", vinculo, err)
	}
}

func vinculoAutenticacionActorV1PruebaSinT(instante time.Time) (VinculoAutenticacionActorV1, error) {
	actor, err := contextoActorVinculoPruebaSinT(instante)
	if err != nil {
		return VinculoAutenticacionActorV1{}, err
	}
	autenticacion := autenticacionRevalidadaVinculoPrueba(instante)
	vinculo, err := CrearVinculoAutenticacionActorV1(
		context.Background(), revalidadorVinculoAutenticacionActorPrueba{resultado: autenticacion},
		solicitudRevalidacionVinculoPrueba(autenticacion), actor, instante,
	)
	return vinculo, err
}

func solicitudRevalidacionVinculoPrueba(a AutenticacionRevalidadaV1) SolicitudRevalidacionAutenticacionActorV1 {
	return SolicitudRevalidacionAutenticacionActorV1{
		AutenticacionRef: a.AutenticacionRef,
		SesionRef:        a.SesionRef,
	}
}

func autenticacionRevalidadaVinculoPrueba(instante time.Time) AutenticacionRevalidadaV1 {
	return AutenticacionRevalidadaV1{
		AutenticacionRef:          "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef:               "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl",
		ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 7,
		ControlSesionHuellaSHA256: strings.Repeat("2", 64),
		CuentaRef:                 "cta_0123456789abcdefghijkl", CuentaOrdinariaRef: "cta_0123456789abcdefghijkl",
		Superficie:      SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado: AuthMethodCertificate, GarantiaObservada: AuthAssuranceHigh,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    instante.Add(-5 * time.Minute),
		SesionEmitidaEn:              instante.Add(-4 * time.Minute),
		SesionRevalidadaEn:           instante.Add(-3 * time.Minute),
		SesionValidaHasta:            instante.Add(10 * time.Minute),
	}
}

func contextoActorVinculoPrueba(t *testing.T, instante time.Time) ContextoActor {
	t.Helper()
	actor, err := contextoActorVinculoPruebaSinT(instante)
	if err != nil {
		t.Fatalf("crear actor de prueba: %v", err)
	}
	return actor
}

func contextoActorVinculoPruebaSinT(instante time.Time) (ContextoActor, error) {
	cuenta := CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl", Metodo: AuthMethodCertificate,
		Garantia: AuthAssuranceHigh,
	}
	instantanea := InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 5,
		CuentaRef: cuenta.CuentaRef, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 3,
		PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 4,
		Estado:       EstadoVinculoContextoActorActivo,
		VigenteDesde: instante.Add(-time.Hour), VigenteHasta: instante.Add(30 * time.Minute),
	}
	return NuevoContextoActor(cuenta, instantanea, instante.Add(-2*time.Minute))
}
