package ports

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type revalidadorContextoConsultaRRHHPrueba struct {
	resultado dominiovec.AutenticacionRevalidadaV1
}

func (d revalidadorContextoConsultaRRHHPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	return d.resultado, nil
}

type resolutorContextoConsultaRRHHPrueba struct {
	resultado dominiovec.ResultadoContextoActorRegistradoV2
}

func (d resolutorContextoConsultaRRHHPrueba) ResolverContextoActorRegistradoV2(
	context.Context,
	dominiovec.SolicitudContextoActor,
) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	return d.resultado, nil
}

type relojContextoConsultaRRHHPrueba struct{ instante time.Time }

func (d relojContextoConsultaRRHHPrueba) Ahora() time.Time {
	return d.instante
}

func TestContextoConsultaRRHHValidaEvidenciaNuevaDeSesionYPerfil(t *testing.T) {
	t.Parallel()
	ahora := time.Date(2026, 7, 26, 8, 0, 0, 0, time.UTC)
	autoridad := autoridadContextoConsultaRRHHPrueba(t, ahora, "a")
	base, err := NuevoContextoConsultaRRHH(
		autoridad,
		"organizacion:diputacion-granada",
		ahora,
	)
	if err != nil {
		t.Fatalf("crear contexto base: %v", err)
	}
	if err := base.validarEn(ahora); err != nil {
		t.Fatalf("contexto base inválido: %v", err)
	}
	casos := []struct {
		nombre string
		mutar  func(*ContextoConsultaRRHH)
	}{
		{"autoridad retenida ausente", func(c *ContextoConsultaRRHH) {
			c.autoridad = nil
		}},
		{"autenticación cruzada", func(c *ContextoConsultaRRHH) {
			c.autenticacionRef = "aut_otra0123456789abcdefghijkl"
		}},
		{"huella de autenticación cruzada", func(c *ContextoConsultaRRHH) {
			c.autenticacionHuella = strings.Repeat("4", 64)
		}},
		{"sesión cruzada", func(c *ContextoConsultaRRHH) {
			c.sesionRef = "ses_otra0123456789abcdefghijkl"
		}},
		{"control sin referencia", func(c *ContextoConsultaRRHH) {
			c.controlSesionRef = ""
		}},
		{"control con referencia cruzada", func(c *ContextoConsultaRRHH) {
			c.controlSesionRef = "cse_otra0123456789abcdefghijkl"
		}},
		{"control sin revisión", func(c *ContextoConsultaRRHH) {
			c.controlSesionRevision = 0
		}},
		{"control con revisión cruzada", func(c *ContextoConsultaRRHH) {
			c.controlSesionRevision++
		}},
		{"control con huella nula", func(c *ContextoConsultaRRHH) {
			c.controlSesionHuellaSHA256 = strings.Repeat("0", 64)
		}},
		{"control con huella no canónica", func(c *ContextoConsultaRRHH) {
			c.controlSesionHuellaSHA256 = strings.Repeat("A", 64)
		}},
		{"control con huella cruzada", func(c *ContextoConsultaRRHH) {
			c.controlSesionHuellaSHA256 = strings.Repeat("5", 64)
		}},
		{"actor cruzado", func(c *ContextoConsultaRRHH) {
			c.actorRef = "per_otra0123456789abcdefghijkl"
		}},
		{"perfil cruzado", func(c *ContextoConsultaRRHH) {
			c.perfilRef = "prf_otra0123456789abcdefghijkl"
		}},
		{"perfil sin versión", func(c *ContextoConsultaRRHH) {
			c.perfilVersion = 0
		}},
		{"perfil con versión cruzada", func(c *ContextoConsultaRRHH) {
			c.perfilVersion++
		}},
		{"perfil con versión fuera del rango interoperable", func(c *ContextoConsultaRRHH) {
			c.perfilVersion = versionMaximaJSONSegura + 1
		}},
		{"registro de contexto cruzado", func(c *ContextoConsultaRRHH) {
			c.registroContextoRef = "rca_otra0123456789abcdefghijkl"
		}},
		{"huella de contexto cruzada", func(c *ContextoConsultaRRHH) {
			c.contextoActorHuella = strings.Repeat("6", 64)
		}},
		{"inicio de vigencia alterado", func(c *ContextoConsultaRRHH) {
			c.resueltoEn = c.resueltoEn.Add(-time.Microsecond)
		}},
		{"fin de vigencia alterado", func(c *ContextoConsultaRRHH) {
			c.validoHasta = c.validoHasta.Add(time.Microsecond)
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			mutada := base
			caso.mutar(&mutada)
			if err := mutada.validarEn(ahora); !errors.Is(
				err, ErrContextoConsultaRRHHInvalido,
			) {
				t.Fatalf("mutación aceptada: %v", err)
			}
		})
	}
}

func TestNuevoContextoConsultaRRHHRetieneParNominalPrivado(t *testing.T) {
	t.Parallel()
	ahora := time.Date(2026, 7, 26, 8, 0, 0, 0, time.UTC)
	autoridad := autoridadContextoConsultaRRHHPrueba(t, ahora, "a")
	clonAjeno, err := autoridad.Resultado.Clonar()
	if err != nil {
		t.Fatalf("clonar resultado de origen: %v", err)
	}
	contextoRRHH, err := NuevoContextoConsultaRRHH(
		autoridad,
		"organizacion:diputacion-granada",
		ahora,
	)
	if err != nil {
		t.Fatalf("crear contexto RRHH: %v", err)
	}
	if contextoRRHH.autoridad == nil ||
		!contextoRRHH.autoridad.Vinculo.CoincideExactamenteCon(
			autoridad.Vinculo,
		) ||
		contextoRRHH.autoridad.Vinculo.ValidarPara(
			contextoRRHH.autoridad.Resultado,
		) != nil {
		t.Fatal("el contexto no retuvo el par nominal exacto y ligado")
	}

	mutarResultadoContextoConsultaRRHHPrueba(&autoridad.Resultado)
	mutarResultadoContextoConsultaRRHHPrueba(&clonAjeno)
	if err := contextoRRHH.validarEn(ahora); err != nil {
		t.Fatalf("una mutación externa alcanzó la copia privada: %v", err)
	}
}

func TestContextoConsultaRRHHCierraCeroCruceYVencimiento(t *testing.T) {
	t.Parallel()
	ahora := time.Date(2026, 7, 26, 8, 0, 0, 0, time.UTC)
	if err := (ContextoConsultaRRHH{}).validarEn(ahora); !errors.Is(
		err, ErrContextoConsultaRRHHInvalido,
	) {
		t.Fatalf("el contexto cero quedó abierto: %v", err)
	}
	autoridadA := autoridadContextoConsultaRRHHPrueba(t, ahora, "a")
	autoridadB := autoridadContextoConsultaRRHHPrueba(t, ahora, "b")
	base, err := NuevoContextoConsultaRRHH(
		autoridadA,
		"organizacion:diputacion-granada",
		ahora,
	)
	if err != nil {
		t.Fatalf("crear contexto base: %v", err)
	}
	autoridadCruzada := *base.autoridad
	autoridadCruzada.Resultado = autoridadB.Resultado
	cruzado := base
	cruzado.autoridad = &autoridadCruzada
	if err := cruzado.validarEn(ahora); !errors.Is(
		err, ErrContextoConsultaRRHHInvalido,
	) {
		t.Fatalf("el par cruzado quedó abierto: %v", err)
	}
	if err := base.validarEn(base.ValidoHasta()); !errors.Is(
		err, ErrContextoConsultaRRHHInvalido,
	) {
		t.Fatalf("el contexto vencido quedó abierto: %v", err)
	}
}

func autoridadContextoConsultaRRHHPrueba(
	t *testing.T,
	ahora time.Time,
	marca string,
) ContextoAutorizacionAltaV3 {
	t.Helper()
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl",
		Metodo:    dominiovec.AuthMethodCertificate,
		Garantia:  dominiovec.AuthAssuranceHigh,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef:      "vca_0123456789abcdefghijkl" + marca,
		VinculoVersion:  3,
		CuentaRef:       cuenta.CuentaRef,
		CuentaVersion:   4,
		PersonaRef:      "per_0123456789abcdefghijkl" + marca,
		PersonaVersion:  2,
		PerfilActivoRef: "prf_0123456789abcdefghijkl" + marca,
		PerfilVersion:   5,
		Estado:          dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde:    ahora.Add(-time.Hour),
		VigenteHasta:    ahora.Add(time.Hour),
		Vinculos:        []dominiovec.VinculoReferenciaContextoActor{},
	}
	actor, err := dominiovec.NuevoContextoActor(
		cuenta,
		instantanea,
		ahora.Add(-2*time.Minute),
	)
	if err != nil {
		t.Fatalf("crear actor de prueba: %v", err)
	}
	actor.Principal.Roles = []string{}
	actor.Principal.Permissions = []string{}
	actor.Principal.Attributes = map[string]string{}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatalf("crear canon de actor: %v", err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatalf("crear huella de actor: %v", err)
	}
	acreditacion := dominiovec.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef:          "prc_0123456789abcdefghijkl",
		ProcedenciaVersion:      1,
		ProcedenciaHuellaSHA256: strings.Repeat("4", 64),
		ProcedenciaAutoridad: dominiovec.
			AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := dominiovec.ManifiestoProcedenciaContextoActorV1{
		Esquema: dominiovec.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: dominiovec.
			AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: dominiovec.ProcedenciaCuentaContextoActorV1{
			CuentaRef: instantanea.CuentaRef,
			Version:   instantanea.CuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: dominiovec.ProcedenciaPersonaContextoActorV1{
			PersonaRef: instantanea.PersonaRef,
			Version:    instantanea.PersonaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: dominiovec.ProcedenciaPerfilContextoActorV1{
			PerfilRef: instantanea.PerfilActivoRef,
			Version:   instantanea.PerfilVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: dominiovec.ProcedenciaVinculoContextoActorV1{
			VinculoRef: instantanea.VinculoRef,
			Version:    instantanea.VinculoVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: []dominiovec.ProcedenciaVinculoReferenciaContextoActorV1{},
	}
	manifiestoCanon, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatalf("crear manifiesto de actor: %v", err)
	}
	manifiestoHuella, err :=
		dominiovec.HuellaSHA256ManifiestoProcedenciaContextoActorV1(
			manifiestoCanon,
		)
	if err != nil {
		t.Fatalf("crear huella de manifiesto: %v", err)
	}
	resultado := dominiovec.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn" + marca,
		Contexto:            actor,
		RepresentacionCanonica: append(
			[]byte(nil),
			canon...,
		),
		HuellaSHA256: huella,
		ManifiestoProcedenciaCanonico: append(
			[]byte(nil),
			manifiestoCanon...,
		),
		ManifiestoProcedenciaHuellaSHA256: manifiestoHuella,
		AutoridadEfectiva: dominiovec.
			AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo: actor.ResueltoEn,
	}
	autenticacion := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef:             "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256:    strings.Repeat("1", 64),
		AsercionRef:                  "ase_0123456789abcdefghijkl",
		SesionRef:                    "ses_0123456789abcdefghijkl",
		ControlSesionRef:             "cse_0123456789abcdefghijkl",
		ControlSesionRevision:        2,
		ControlSesionHuellaSHA256:    strings.Repeat("2", 64),
		CuentaRef:                    cuenta.CuentaRef,
		CuentaOrdinariaRef:           cuenta.CuentaRef,
		Superficie:                   dominiovec.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado:              cuenta.Metodo,
		GarantiaObservada:            cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    ahora.Add(-10 * time.Minute),
		SesionEmitidaEn:              ahora.Add(-9 * time.Minute),
		SesionValidaHasta:            ahora.Add(20 * time.Minute),
		SesionRevalidadaEn:           ahora.Add(-3 * time.Minute),
	}
	vinculo, resultadoLigado, err :=
		dominiovec.CrearVinculoAutenticacionActorV2ConResultado(
			context.Background(),
			revalidadorContextoConsultaRRHHPrueba{resultado: autenticacion},
			dominiovec.SolicitudRevalidacionAutenticacionActorV1{
				AutenticacionRef: autenticacion.AutenticacionRef,
				SesionRef:        autenticacion.SesionRef,
			},
			resolutorContextoConsultaRRHHPrueba{resultado: resultado},
			dominiovec.SolicitudContextoActor{
				Cuenta:          cuenta,
				PerfilActivoRef: instantanea.PerfilActivoRef,
			},
			relojContextoConsultaRRHHPrueba{instante: ahora},
		)
	if err != nil {
		t.Fatalf("crear par nominal de prueba: %v", err)
	}
	return ContextoAutorizacionAltaV3{
		Vinculo:   vinculo,
		Resultado: resultadoLigado,
	}
}

func mutarResultadoContextoConsultaRRHHPrueba(
	resultado *dominiovec.ResultadoContextoActorRegistradoV2,
) {
	resultado.RepresentacionCanonica[0] ^= 1
	resultado.ManifiestoProcedenciaCanonico[0] ^= 1
	resultado.Contexto.Principal.Roles = append(
		resultado.Contexto.Principal.Roles,
		"rol_inyectado",
	)
	resultado.Contexto.Principal.Permissions = append(
		resultado.Contexto.Principal.Permissions,
		"permiso_inyectado",
	)
	resultado.Contexto.Principal.Attributes["atributo_inyectado"] = "valor"
	resultado.Contexto.Instantanea.Vinculos = append(
		resultado.Contexto.Instantanea.Vinculos,
		dominiovec.VinculoReferenciaContextoActor{},
	)
	resultado.Contexto.Instantanea.PerfilVersion++
}
