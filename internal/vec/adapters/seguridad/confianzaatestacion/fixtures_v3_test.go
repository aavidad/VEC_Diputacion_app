package confianzaatestacion

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	gocose "github.com/veraison/go-cose"

	puertoscontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const audienciaConfianzaAtestacionV3Prueba = "vec-diputacion/pruebas/contratacion-temporal"

type relojConfianzaAtestacionV3Prueba struct {
	ahora        time.Time
	invocaciones int
	cancelar     context.CancelFunc
}

func (r *relojConfianzaAtestacionV3Prueba) Ahora() time.Time {
	r.invocaciones++
	if r.cancelar != nil {
		r.cancelar()
	}
	return r.ahora
}

type revalidadorConfianzaAtestacionV3Prueba struct {
	resultado domain.AutenticacionRevalidadaV1
}

func (r revalidadorConfianzaAtestacionV3Prueba) RevalidarAutenticacionActorV1(
	context.Context,
	domain.SolicitudRevalidacionAutenticacionActorV1,
) (domain.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

type resolutorConfianzaAtestacionV3Prueba struct {
	resultado domain.ResultadoContextoActorRegistradoV2
}

func (r resolutorConfianzaAtestacionV3Prueba) ResolverContextoActorRegistradoV2(
	context.Context,
	domain.SolicitudContextoActor,
) (domain.ResultadoContextoActorRegistradoV2, error) {
	return r.resultado, nil
}

type generadorCorrelacionConfianzaAtestacionV3Prueba struct{ valor string }

func (g generadorCorrelacionConfianzaAtestacionV3Prueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return g.valor, nil
}

type escenarioConfianzaAtestacionV3Prueba struct {
	ahora         time.Time
	reloj         *relojConfianzaAtestacionV3Prueba
	privada       ed25519.PrivateKey
	raiz          RaizPublicaAtestacionAutorizacionV3
	configuracion ConfiguracionConfianzaAtestacionAutorizacionV3
	servicio      *ServicioConfianzaAtestacionAutorizacionV3
	solicitud     domain.SolicitudAutorizacionLigadaV3
	decision      domain.DecisionAutorizacionLigadaV3
	motivo        domain.ReferenciaEntradaCatalogo
	resultado     domain.ResultadoContextoActorRegistradoV2
	atestacion    ports.AtestacionAutorizacionV3
}

func nuevoEscenarioConfianzaAtestacionV3Prueba(
	t *testing.T,
) escenarioConfianzaAtestacionV3Prueba {
	t.Helper()
	ahora := time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC)
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl",
		Metodo:    domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceHigh,
	}
	instantaneaActor := domain.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 3,
		CuentaRef: cuenta.CuentaRef, CuentaVersion: 4,
		PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 2,
		PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 5,
		Estado:       domain.EstadoVinculoContextoActorActivo,
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
	}
	actor, err := domain.NuevoContextoActor(
		cuenta,
		instantaneaActor,
		ahora.Add(-2*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	representacion, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := domain.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef:          "prc_0123456789abcdefghijkl",
		ProcedenciaVersion:      1,
		ProcedenciaHuellaSHA256: strings.Repeat("4", 64),
		ProcedenciaAutoridad:    domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := domain.ManifiestoProcedenciaContextoActorV1{
		Esquema:           domain.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: domain.ProcedenciaCuentaContextoActorV1{
			CuentaRef: cuenta.CuentaRef, Version: instantaneaActor.CuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: domain.ProcedenciaPersonaContextoActorV1{
			PersonaRef: instantaneaActor.PersonaRef, Version: instantaneaActor.PersonaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: domain.ProcedenciaPerfilContextoActorV1{
			PerfilRef: instantaneaActor.PerfilActivoRef, Version: instantaneaActor.PerfilVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: domain.ProcedenciaVinculoContextoActorV1{
			VinculoRef: instantaneaActor.VinculoRef, Version: instantaneaActor.VinculoVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: make([]domain.ProcedenciaVinculoReferenciaContextoActorV1, 0),
	}
	canonManifiesto, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	huellaManifiesto, err := domain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(
		canonManifiesto,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado := domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn",
		Contexto:            actor, RepresentacionCanonica: representacion,
		HuellaSHA256:                      huella,
		ManifiestoProcedenciaCanonico:     canonManifiesto,
		ManifiestoProcedenciaHuellaSHA256: huellaManifiesto,
		AutoridadEfectiva:                 domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo:            actor.ResueltoEn,
	}
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef:          "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef:               "ase_0123456789abcdefghijkl",
		SesionRef:                 "ses_0123456789abcdefghijkl",
		ControlSesionRef:          "cse_0123456789abcdefghijkl",
		ControlSesionRevision:     2,
		ControlSesionHuellaSHA256: strings.Repeat("2", 64),
		CuentaRef:                 cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef,
		Superficie:      domain.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    ahora.Add(-10 * time.Minute),
		SesionEmitidaEn:              ahora.Add(-9 * time.Minute),
		SesionRevalidadaEn:           ahora.Add(-3 * time.Minute),
		SesionValidaHasta:            ahora.Add(20 * time.Minute),
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV2(
		context.Background(),
		revalidadorConfianzaAtestacionV3Prueba{autenticacion},
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		resolutorConfianzaAtestacionV3Prueba{resultado},
		domain.SolicitudContextoActor{
			Cuenta: cuenta, PerfilActivoRef: instantaneaActor.PerfilActivoRef,
		},
		&relojConfianzaAtestacionV3Prueba{ahora: ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	motivo := domain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion", CatalogoVersion: 2,
		CatalogoHuellaSHA256: strings.Repeat("d", 64),
		EntradaClave:         "motivo_11111111111111111111111111111111",
	}
	correlacion, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		generadorCorrelacionConfianzaAtestacionV3Prueba{
			valor: "correlacion_11111111111111111111111111111111",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	efectoPrueba := sha256.Sum256(bytesEfectoAltaCompatibilidadO206())
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(
		domain.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: vinculo, ReferenciaMotivo: motivo,
			Accion: puertoscontratacion.AccionCrearSolicitud,
			Recurso: domain.RecursoAutorizable{
				Referencia: "hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				ModuloID:   puertoscontratacion.ModuloContratacion,
				Tipo:       puertoscontratacion.TipoRecursoExpediente,
				Ambitos: map[string]string{
					"organizacion_ref": "organizacion:dipgra",
					"centro_ref":       "centro:servicios-generales",
					"categoria_ref":    "categoria:auxiliar-administrativo",
				},
				Atributos: map[string]string{
					"flujo_ref":           "flujo:contratacion-temporal:alta",
					"flujo_version":       "1",
					"flujo_huella_sha256": strings.Repeat("e", 64),
					puertoscontratacion.AtributoHuellaEfectoAltaSHA256: hex.EncodeToString(
						efectoPrueba[:],
					),
					puertoscontratacion.AtributoHuellaPeticionHMACActiva: "hmac-sha256:vec.contratacion-temporal.huella-peticion/v1:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				},
			},
			Finalidad:   puertoscontratacion.FinalidadCrearSolicitud,
			Correlacion: correlacion,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	version := domain.VersionRol{
		RolID: "tecnico_rrhh", Version: 1, Nombre: "Tecnico RRHH",
		Estado: domain.EstadoVersionRolPublicada,
		Concesiones: []domain.ConcesionRol{{
			Accion:         puertoscontratacion.AccionCrearSolicitud,
			ModuloID:       puertoscontratacion.ModuloContratacion,
			TipoRecurso:    puertoscontratacion.TipoRecursoExpediente,
			Finalidades:    []string{puertoscontratacion.FinalidadCrearSolicitud},
			GarantiaMinima: domain.AuthAssuranceSubstantial,
		}},
		PublicadaPor: "responsable-seguridad",
		PublicadaEn:  ahora.Add(-24 * time.Hour),
	}
	huellaCatalogo, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := domain.InstantaneaAutorizacion{
		AsignacionPerfil: domain.AsignacionPerfil{
			AsignacionID: "asig-rrhh", Version: 1,
			PerfilActivoRef: instantaneaActor.PerfilActivoRef,
			PrincipalID:     instantaneaActor.PersonaRef,
			VersionRolRef:   version.Referencia(),
			Estado:          domain.EstadoAsignacionPerfilActiva,
			Ambitos: []domain.AmbitoPerfil{
				{
					Clave:   "organizacion_ref",
					Valores: []string{"organizacion:dipgra"},
				},
				{
					Clave:   "centro_ref",
					Valores: []string{"centro:servicios-generales"},
				},
				{
					Clave:   "categoria_ref",
					Valores: []string{"categoria:auxiliar-administrativo"},
				},
			},
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
			EmitidaPor: "administrador-identidades",
			EmitidaEn:  ahora.Add(-2 * time.Hour),
		},
		VersionRol: version,
		ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         domain.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn,
		},
		RevisionCatalogoPoliticas:     1,
		CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	evidencia, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud,
		instantanea,
		"dec_0123456789abcdef0123456789abcdef",
		ahora,
		ahora.Add(90*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := domain.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	if err := decision.Validar(); err != nil {
		t.Fatalf("decision V3 recien creada invalida: %v", err)
	}
	if concedida, codigo, err := decision.Resultado(); err != nil || !concedida {
		t.Fatalf("decision V3 no concedida (%t, %q): %v", concedida, codigo, err)
	}
	privada := ed25519.NewKeyFromSeed(
		[]byte(strings.Repeat("v", ed25519.SeedSize)),
	)
	claveID := "clave:atestacion:v3:activa:2026-07"
	raiz, err := NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(
		claveID,
		1,
		privada.Public().(ed25519.PublicKey),
		audienciaConfianzaAtestacionV3Prueba,
		EstadoClaveAtestacionAutorizacionV3Activa,
		ahora.Add(-time.Hour),
		ahora.Add(time.Hour),
		time.Time{},
	)
	if err != nil {
		t.Fatal(err)
	}
	configuracion, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
		"confianza:atestacion:v3:revision:activa",
		7,
		ahora.Add(-time.Minute),
		ahora.Add(30*time.Minute),
		raiz,
	)
	if err != nil {
		t.Fatal(err)
	}
	reloj := &relojConfianzaAtestacionV3Prueba{ahora: ahora}
	servicio, err := NuevoServicioConfianzaAtestacionAutorizacionV3(
		configuracion,
		reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	cabecera := domain.CabeceraAtestacionAutorizacionV3{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3,
		Suite:          SuiteAtestacionAutorizacionV3COSEEdDSA,
		ClaveID:        claveID, Audiencia: audienciaConfianzaAtestacionV3Prueba,
	}
	atestacion := atestacionConfianzaAtestacionV3Prueba(
		t, cabecera, decision, motivo, resultado, privada, ahora,
	)
	return escenarioConfianzaAtestacionV3Prueba{
		ahora: ahora, reloj: reloj, privada: privada, raiz: raiz,
		configuracion: configuracion, servicio: servicio,
		solicitud: solicitud, decision: decision, motivo: motivo,
		resultado: resultado, atestacion: atestacion,
	}
}

func bytesEfectoAltaCompatibilidadO206() []byte {
	return []byte(strings.Repeat("e", 256))
}

func atestacionConfianzaAtestacionV3Prueba(
	t *testing.T,
	cabecera domain.CabeceraAtestacionAutorizacionV3,
	decision domain.DecisionAutorizacionLigadaV3,
	motivo domain.ReferenciaEntradaCatalogo,
	resultado domain.ResultadoContextoActorRegistradoV2,
	privada ed25519.PrivateKey,
	firmadaEn time.Time,
) ports.AtestacionAutorizacionV3 {
	t.Helper()
	mensajeDirecto, err := domain.SerializarMensajeAtestacionAutorizacionV3(
		cabecera, decision, motivo, resultado,
	)
	if err != nil {
		t.Fatalf("serializar VEC-AD-3: %v", err)
	}
	if _, err := domain.ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(
		mensajeDirecto,
	); err != nil {
		t.Fatalf("parsear VEC-AD-3: %v", err)
	}
	solicitud, err := ports.NuevaSolicitudFirmaAtestacionAutorizacionV3(
		cabecera, decision, motivo, resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	mensaje, err := solicitud.Mensaje()
	if err != nil {
		t.Fatal(err)
	}
	aad, err := AADExternoAtestacionAutorizacionV3(cabecera.Audiencia)
	if err != nil {
		t.Fatal(err)
	}
	sobre := firmarSobreConfianzaAtestacionV3Prueba(
		t, privada, []byte(cabecera.ClaveID), mensaje, aad,
	)
	resultadoFirma, err := ports.NuevoResultadoFirmaAtestacionAutorizacionV3(
		solicitud,
		sobre,
		"evidencia:firma:confianza:atestacion:v3",
		firmadaEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	atestacion, err := ports.NuevaAtestacionAutorizacionV3(
		solicitud,
		resultadoFirma,
	)
	if err != nil {
		t.Fatal(err)
	}
	return atestacion
}

func firmarSobreConfianzaAtestacionV3Prueba(
	t *testing.T,
	privada ed25519.PrivateKey,
	claveID []byte,
	payload []byte,
	aad []byte,
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
	mensaje.Payload = nil
	mensaje.Headers.RawProtected = nil
	mensaje.Headers.RawUnprotected = nil
	contenido, err := mensaje.MarshalCBOR()
	if err != nil {
		t.Fatal(err)
	}
	return contenido
}
