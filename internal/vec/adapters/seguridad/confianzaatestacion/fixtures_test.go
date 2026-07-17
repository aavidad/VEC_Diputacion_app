package confianzaatestacion

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"
	"time"

	gocose "github.com/veraison/go-cose"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	principalConfianzaAtestacionV2Prueba = "per_0123456789abcdefghijkl"
	perfilConfianzaAtestacionV2Prueba    = "prf_0123456789abcdefghijkl"
	claveMotivoConfianzaAtestacionV2     = "motivo_33333333333333333333333333333333"
	audienciaConfianzaAtestacionV2Prueba = "vec-diputacion/pruebas/vec/autorizacion-v2"
)

type relojConfianzaAtestacionV2Prueba struct{ ahora time.Time }

func (r *relojConfianzaAtestacionV2Prueba) Ahora() time.Time { return r.ahora }

type revalidadorConfianzaAtestacionV2Prueba struct {
	resultado domain.AutenticacionRevalidadaV1
}

func (r revalidadorConfianzaAtestacionV2Prueba) RevalidarAutenticacionActorV1(
	context.Context,
	domain.SolicitudRevalidacionAutenticacionActorV1,
) (domain.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

type escenarioConfianzaAtestacionV2Prueba struct {
	ahora         time.Time
	reloj         *relojConfianzaAtestacionV2Prueba
	privada       ed25519.PrivateKey
	raiz          RaizPublicaAtestacionAutorizacionV2
	configuracion ConfiguracionConfianzaAtestacionAutorizacionV2
	servicio      *ServicioConfianzaAtestacionAutorizacionV2
	decision      domain.DecisionAutorizacion
	motivo        domain.ReferenciaEntradaCatalogo
	atestacion    ports.AtestacionAutorizacionV2
}

func nuevoEscenarioConfianzaAtestacionV2Prueba(
	t *testing.T,
) escenarioConfianzaAtestacionV2Prueba {
	t.Helper()
	ahora := instanteConfianzaAtestacionV2Prueba()
	privada := clavePrivadaConfianzaAtestacionV2Prueba(80)
	claveID := "clave:atestacion:v2:activa:2026-07"
	raiz := nuevaRaizConfianzaAtestacionV2Prueba(
		t,
		claveID,
		privada.Public().(ed25519.PublicKey),
		audienciaConfianzaAtestacionV2Prueba,
		EstadoClaveAtestacionAutorizacionV2Activa,
		ahora.Add(-time.Hour),
		ahora.Add(time.Hour),
		time.Time{},
	)
	configuracion, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV2(
		"confianza:atestacion:v2:revision:activa",
		ahora.Add(-time.Minute),
		ahora.Add(30*time.Minute),
		raiz,
	)
	if err != nil {
		t.Fatal(err)
	}
	reloj := &relojConfianzaAtestacionV2Prueba{ahora: ahora}
	servicio, err := NuevoServicioConfianzaAtestacionAutorizacionV2(configuracion, reloj)
	if err != nil {
		t.Fatal(err)
	}
	motivo := referenciaMotivoConfianzaAtestacionV2Prueba()
	decision := decisionConfianzaAtestacionV2Prueba(t, ahora, motivo)
	cabecera := domain.CabeceraAtestacionAutorizacionV2{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV2,
		Suite:          SuiteAtestacionAutorizacionV2COSEEdDSA,
		ClaveID:        claveID,
		Audiencia:      audienciaConfianzaAtestacionV2Prueba,
	}
	atestacion := atestacionConfianzaAtestacionV2Prueba(
		t,
		cabecera,
		decision,
		motivo,
		privada,
		[]byte(claveID),
		nil,
		nil,
		true,
		ahora,
	)
	return escenarioConfianzaAtestacionV2Prueba{
		ahora: ahora, reloj: reloj, privada: privada, raiz: raiz,
		configuracion: configuracion, servicio: servicio, decision: decision,
		motivo: motivo, atestacion: atestacion,
	}
}

func atestacionConfianzaAtestacionV2Prueba(
	t *testing.T,
	cabecera domain.CabeceraAtestacionAutorizacionV2,
	decision domain.DecisionAutorizacion,
	motivo domain.ReferenciaEntradaCatalogo,
	privada ed25519.PrivateKey,
	claveIDSobre []byte,
	payloadAlternativo []byte,
	aadAlternativo []byte,
	separado bool,
	firmadaEn time.Time,
) ports.AtestacionAutorizacionV2 {
	t.Helper()
	solicitud, err := ports.NuevaSolicitudFirmaAtestacionAutorizacionV2(
		cabecera,
		decision,
		motivo,
	)
	if err != nil {
		t.Fatalf("crear solicitud atestada V2: %v", err)
	}
	mensaje, err := solicitud.Mensaje()
	if err != nil {
		t.Fatal(err)
	}
	payload := mensaje
	if payloadAlternativo != nil {
		payload = payloadAlternativo
	}
	aad, err := AADExternoAtestacionAutorizacionV2(cabecera.Audiencia)
	if err != nil {
		t.Fatal(err)
	}
	if aadAlternativo != nil {
		aad = aadAlternativo
	}
	sobre := firmarSobreConfianzaAtestacionV2Prueba(
		t,
		privada,
		claveIDSobre,
		payload,
		aad,
		separado,
	)
	resultado, err := ports.NuevoResultadoFirmaAtestacionAutorizacionV2(
		solicitud,
		sobre,
		"evidencia:firma:confianza:atestacion:v2",
		firmadaEn,
	)
	if err != nil {
		t.Fatalf("crear resultado de firma V2: %v", err)
	}
	atestacion, err := ports.NuevaAtestacionAutorizacionV2(solicitud, resultado)
	if err != nil {
		t.Fatalf("crear atestacion V2: %v", err)
	}
	return atestacion
}

func firmarSobreConfianzaAtestacionV2Prueba(
	t *testing.T,
	privada ed25519.PrivateKey,
	claveID []byte,
	payload []byte,
	aad []byte,
	separado bool,
) []byte {
	t.Helper()
	mensaje := gocose.NewSign1Message()
	mensaje.Headers.Protected.SetAlgorithm(gocose.AlgorithmEdDSA)
	mensaje.Headers.Protected[gocose.HeaderLabelKeyID] = append([]byte(nil), claveID...)
	mensaje.Payload = append([]byte(nil), payload...)
	firmante, err := gocose.NewSigner(gocose.AlgorithmEdDSA, privada)
	if err != nil {
		t.Fatal(err)
	}
	if err := mensaje.Sign(rand.Reader, aad, firmante); err != nil {
		t.Fatal(err)
	}
	if separado {
		mensaje.Payload = nil
	}
	mensaje.Headers.RawProtected = nil
	mensaje.Headers.RawUnprotected = nil
	contenido, err := mensaje.MarshalCBOR()
	if err != nil {
		t.Fatal(err)
	}
	return contenido
}

func reemplazarSobreConfianzaAtestacionV2Prueba(
	t *testing.T,
	atestacion ports.AtestacionAutorizacionV2,
	sobre []byte,
) ports.AtestacionAutorizacionV2 {
	t.Helper()
	solicitud, err := atestacion.Solicitud()
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := ports.NuevoResultadoFirmaAtestacionAutorizacionV2(
		solicitud,
		sobre,
		"evidencia:firma:confianza:atestacion:v2:reemplazada",
		instanteConfianzaAtestacionV2Prueba(),
	)
	if err != nil {
		t.Fatal(err)
	}
	nueva, err := ports.NuevaAtestacionAutorizacionV2(solicitud, resultado)
	if err != nil {
		t.Fatal(err)
	}
	return nueva
}

func decisionConfianzaAtestacionV2Prueba(
	t *testing.T,
	ahora time.Time,
	motivo domain.ReferenciaEntradaCatalogo,
) domain.DecisionAutorizacion {
	t.Helper()
	vinculo := vinculoConfianzaAtestacionV2Prueba(t, ahora)
	evaluadas := []string{"politica:ambito:v1", "politica:seguridad:v1"}
	huellas := map[string]string{
		evaluadas[0]: strings.Repeat("a", 64),
		evaluadas[1]: strings.Repeat("b", 64),
	}
	huellaCatalogo, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(evaluadas, huellas)
	if err != nil {
		t.Fatal(err)
	}
	huellaMotivo, err := domain.HuellaSHA256MotivoAutorizacionV2(motivo)
	if err != nil {
		t.Fatal(err)
	}
	decision := domain.DecisionAutorizacion{
		DecisionRef: "decision:confianza:atestacion:v2:1", Concedida: true, Codigo: "concedida",
		PrincipalID:     principalConfianzaAtestacionV2Prueba,
		PerfilActivoRef: perfilConfianzaAtestacionV2Prueba,
		Accion:          "bolsa.baremacion.reservar", RecursoRef: "baremacion:merito:1",
		ModuloID: "bolsa", TipoRecurso: "baremacion_merito",
		ContextoRecursoHuellaSHA256: strings.Repeat("c", 64),
		Finalidad:                   "gestion_bolsa", CorrelacionRef: "correlacion_44444444444444444444444444444444",
		EsquemaHuellaSolicitud: domain.EsquemaHuellaSolicitudAutorizacionV2,
		SolicitudHuellaSHA256:  strings.Repeat("d", 64),
		EsquemaHuellaMotivo:    domain.EsquemaHuellaMotivoAutorizacionV2,
		MotivoHuellaSHA256:     huellaMotivo, VinculoAutenticacionActor: vinculo,
		AsignacionRef: "asignacion:rrhh:1", AsignacionHuellaSHA256: strings.Repeat("e", 64),
		VersionRolRef: "rol:tecnico_rrhh:v1", VersionRolHuellaSHA256: strings.Repeat("f", 64),
		ControlVigenciaVersionRolRef:          "rol:tecnico_rrhh:v1",
		ControlVigenciaVersionRolRevision:     1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("1", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasRefs:          evaluadas,
		PoliticasEvaluadasHuellasSHA256: huellas,
		PoliticasRefs:                   []string{evaluadas[0]},
		PoliticasHuellasSHA256:          map[string]string{evaluadas[0]: huellas[evaluadas[0]]},
		GarantiaMinima:                  domain.AuthAssuranceHigh,
		CamposPermitidos:                []string{"estado"}, Obligaciones: []string{"registrar_acceso"},
		EmitidaEn: ahora.Add(-time.Minute), ValidaHasta: ahora.Add(3 * time.Minute),
	}
	if err := decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2(); err != nil {
		t.Fatalf("decision V2 de confianza invalida: %v", err)
	}
	return decision
}

func referenciaMotivoConfianzaAtestacionV2Prueba() domain.ReferenciaEntradaCatalogo {
	return domain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion", CatalogoVersion: 7,
		CatalogoHuellaSHA256: strings.Repeat("6", 64),
		EntradaClave:         claveMotivoConfianzaAtestacionV2,
	}
}

func vinculoConfianzaAtestacionV2Prueba(
	t *testing.T,
	ahora time.Time,
) domain.VinculoAutenticacionActorV1 {
	t.Helper()
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl", Metodo: domain.AuthMethodCertificate,
		Garantia: domain.AuthAssuranceHigh,
	}
	instantanea := domain.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 5,
		CuentaRef: cuenta.CuentaRef, PersonaRef: principalConfianzaAtestacionV2Prueba,
		PersonaVersion: 3, PerfilActivoRef: perfilConfianzaAtestacionV2Prueba, PerfilVersion: 4,
		Estado:       domain.EstadoVinculoContextoActorActivo,
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
	}
	actor, err := domain.NuevoContextoActor(cuenta, instantanea, ahora.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef:          "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256: strings.Repeat("2", 64),
		AsercionRef:               "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl",
		ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 3,
		ControlSesionHuellaSHA256: strings.Repeat("3", 64),
		CuentaRef:                 cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef,
		Superficie:      domain.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("4", 64),
		AutenticacionVerificadaEn:    ahora.Add(-5 * time.Minute),
		SesionEmitidaEn:              ahora.Add(-4 * time.Minute),
		SesionRevalidadaEn:           ahora.Add(-3 * time.Minute),
		SesionValidaHasta:            ahora.Add(30 * time.Minute),
	}
	solicitud := domain.SolicitudRevalidacionAutenticacionActorV1{
		AutenticacionRef: autenticacion.AutenticacionRef,
		SesionRef:        autenticacion.SesionRef,
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV1(
		context.Background(),
		revalidadorConfianzaAtestacionV2Prueba{resultado: autenticacion},
		solicitud,
		actor,
		ahora,
	)
	if err != nil {
		t.Fatalf("crear vinculo de prueba: %v", err)
	}
	return vinculo
}

func clavePrivadaConfianzaAtestacionV2Prueba(semilla byte) ed25519.PrivateKey {
	return ed25519.NewKeyFromSeed(
		[]byte(strings.Repeat(string([]byte{semilla}), ed25519.SeedSize)),
	)
}
