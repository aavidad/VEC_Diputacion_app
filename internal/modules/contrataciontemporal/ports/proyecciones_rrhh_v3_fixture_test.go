package ports_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type revalidadorContextoRRHHPrueba struct {
	resultado dominiovec.AutenticacionRevalidadaV1
}

func (d revalidadorContextoRRHHPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	return d.resultado, nil
}

type resolutorContextoRRHHPrueba struct {
	resultado dominiovec.ResultadoContextoActorRegistradoV2
}

func (d resolutorContextoRRHHPrueba) ResolverContextoActorRegistradoV2(
	context.Context,
	dominiovec.SolicitudContextoActor,
) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	return d.resultado, nil
}

type relojContextoRRHHPrueba struct{ instante time.Time }

func (d relojContextoRRHHPrueba) Ahora() time.Time { return d.instante }

type generadorCorrelacionRRHHPrueba struct{ referencia string }

func (d generadorCorrelacionRRHHPrueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return d.referencia, nil
}

func (generadorCorrelacionRRHHPrueba) NuevaClaveMotivoAutorizacionV2(
	context.Context,
) (string, error) {
	return "", errors.New("operación no utilizada por la prueba")
}

type registroConcesionRRHHPrueba struct{ instante time.Time }

func (d registroConcesionRRHHPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	context.Context,
	puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	return d.instante, nil
}

type exportadorMaterialRRHHPrueba struct {
	exportacion puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (exportadorMaterialRRHHPrueba) String() string {
	return "[MATERIAL-RRHH-PRUEBA-OPACO]"
}

func (exportadorMaterialRRHHPrueba) LogValue() slog.Value {
	return slog.StringValue("[MATERIAL-RRHH-PRUEBA-OPACO]")
}

func (d exportadorMaterialRRHHPrueba) ExportarMaterialParaConsumidor() (
	puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	error,
) {
	return d.exportacion, nil
}

func autoridadContextoPuertosRRHH(
	t *testing.T,
	ahora time.Time,
	marcaActor, marcaPerfil string,
) ports.ContextoAutorizacionAltaV3 {
	t.Helper()
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl",
		Metodo:    dominiovec.AuthMethodCertificate,
		Garantia:  dominiovec.AuthAssuranceHigh,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef:      "vca_0123456789abcdefghijkl" + marcaActor + marcaPerfil,
		VinculoVersion:  3,
		CuentaRef:       cuenta.CuentaRef,
		CuentaVersion:   4,
		PersonaRef:      "per_0123456789abcdefghijkl" + marcaActor,
		PersonaVersion:  2,
		PerfilActivoRef: "prf_0123456789abcdefghijkl" + marcaPerfil,
		PerfilVersion:   5,
		Estado:          dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde:    ahora.Add(-time.Hour),
		VigenteHasta:    ahora.Add(time.Hour),
	}
	actor, err := dominiovec.NuevoContextoActor(
		cuenta, instantanea, ahora.Add(-2*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
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
			CuentaRef: instantanea.CuentaRef, Version: instantanea.CuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: dominiovec.ProcedenciaPersonaContextoActorV1{
			PersonaRef: instantanea.PersonaRef, Version: instantanea.PersonaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: dominiovec.ProcedenciaPerfilContextoActorV1{
			PerfilRef: instantanea.PerfilActivoRef, Version: instantanea.PerfilVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: dominiovec.ProcedenciaVinculoContextoActorV1{
			VinculoRef: instantanea.VinculoRef, Version: instantanea.VinculoVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: []dominiovec.ProcedenciaVinculoReferenciaContextoActorV1{},
	}
	manifiestoCanon, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	manifiestoHuella, err :=
		dominiovec.HuellaSHA256ManifiestoProcedenciaContextoActorV1(
			manifiestoCanon,
		)
	if err != nil {
		t.Fatal(err)
	}
	resultado := dominiovec.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn" +
			marcaActor + marcaPerfil,
		Contexto:                          actor,
		RepresentacionCanonica:            canon,
		HuellaSHA256:                      huella,
		ManifiestoProcedenciaCanonico:     manifiestoCanon,
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
	vinculo, err := dominiovec.CrearVinculoAutenticacionActorV2(
		context.Background(),
		revalidadorContextoRRHHPrueba{resultado: autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		resolutorContextoRRHHPrueba{resultado: resultado},
		dominiovec.SolicitudContextoActor{
			Cuenta: cuenta, PerfilActivoRef: instantanea.PerfilActivoRef,
		},
		relojContextoRRHHPrueba{instante: ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	return ports.ContextoAutorizacionAltaV3{
		Vinculo: vinculo, Resultado: resultado,
	}
}

func contextoPuertosRRHHConMarcas(
	t *testing.T,
	ahora time.Time,
	marcaActor, marcaPerfil, organizacionRef string,
) ports.ContextoConsultaRRHH {
	t.Helper()
	autoridad := autoridadContextoPuertosRRHH(
		t, ahora, marcaActor, marcaPerfil,
	)
	contexto, err := ports.NuevoContextoConsultaRRHH(
		autoridad, organizacionRef, ahora,
	)
	if err != nil {
		t.Fatalf("contexto RRHH V3: %v", err)
	}
	return contexto
}

func contextoPuertosRRHH(
	t *testing.T,
	ahora time.Time,
) ports.ContextoConsultaRRHH {
	t.Helper()
	return contextoPuertosRRHHConMarcas(
		t, ahora, "a", "a", "organizacion:diputacion-granada",
	)
}

func autoridadYContextoPuertosRRHH(
	t *testing.T,
	ahora time.Time,
) (ports.ContextoAutorizacionAltaV3, ports.ContextoConsultaRRHH) {
	t.Helper()
	autoridad := autoridadContextoPuertosRRHH(t, ahora, "a", "a")
	contexto, err := ports.NuevoContextoConsultaRRHH(
		autoridad, "organizacion:diputacion-granada", ahora,
	)
	if err != nil {
		t.Fatalf("contexto RRHH V3: %v", err)
	}
	return autoridad, contexto
}

func capacidadCuadroPuertosRRHH(
	t *testing.T,
	autoridad ports.ContextoAutorizacionAltaV3,
	contexto ports.ContextoConsultaRRHH,
	solicitud ports.SolicitudCuadroRRHH,
	instante time.Time,
) ports.CapacidadConsultaRRHH {
	t.Helper()
	material, err := materialConsultaRRHHPrueba(
		t, autoridad, contexto, solicitud, ports.SolicitudDetalleRRHH{},
		ports.AccionConsultarCuadroRRHH, ports.FinalidadConsultarCuadroRRHH,
		ports.AudienciaConsumoConsultaCuadroRRHHV3, instante,
	)
	if err != nil {
		t.Fatalf("material de cuadro RRHH V3: %v", err)
	}
	capacidad, err := ports.NuevaCapacidadConsultaCuadroRRHH(
		contexto, material, solicitud, instante,
	)
	if err != nil {
		t.Fatalf("capacidad de cuadro RRHH V3: %v", err)
	}
	return capacidad
}

func capacidadDetallePuertosRRHH(
	t *testing.T,
	autoridad ports.ContextoAutorizacionAltaV3,
	contexto ports.ContextoConsultaRRHH,
	solicitud ports.SolicitudDetalleRRHH,
	instante time.Time,
) ports.CapacidadConsultaRRHH {
	t.Helper()
	material, err := materialConsultaRRHHPrueba(
		t, autoridad, contexto, ports.SolicitudCuadroRRHH{}, solicitud,
		ports.AccionConsultarDetalleRRHH, ports.FinalidadConsultarDetalleRRHH,
		ports.AudienciaConsumoConsultaDetalleRRHHV3, instante,
	)
	if err != nil {
		t.Fatalf("material de detalle RRHH V3: %v", err)
	}
	capacidad, err := ports.NuevaCapacidadConsultaDetalleRRHH(
		contexto, material, solicitud, instante,
	)
	if err != nil {
		t.Fatalf("capacidad de detalle RRHH V3: %v", err)
	}
	return capacidad
}

func materialConsultaRRHHPrueba(
	t *testing.T,
	autoridad ports.ContextoAutorizacionAltaV3,
	contexto ports.ContextoConsultaRRHH,
	cuadro ports.SolicitudCuadroRRHH,
	detalle ports.SolicitudDetalleRRHH,
	accion, finalidad, audiencia string,
	instante time.Time,
) (ports.MaterialAutorizacionConsultaRRHH, error) {
	t.Helper()
	return materialConsultaRRHHPruebaAlterado(
		t, autoridad, contexto, cuadro, detalle, accion, finalidad,
		audiencia, instante, nil,
	)
}

func materialConsultaRRHHPruebaAlterado(
	t *testing.T,
	autoridad ports.ContextoAutorizacionAltaV3,
	contexto ports.ContextoConsultaRRHH,
	cuadro ports.SolicitudCuadroRRHH,
	detalle ports.SolicitudDetalleRRHH,
	accion, finalidad, audiencia string,
	instante time.Time,
	mutarRecurso func(*dominiovec.RecursoAutorizable),
) (ports.MaterialAutorizacionConsultaRRHH, error) {
	t.Helper()
	dominio, huella, recursoRef := "", "", contexto.OrganizacionRef()
	var err error
	switch accion {
	case ports.AccionConsultarCuadroRRHH:
		dominio = ports.DominioHuellaConsultaCuadroRRHH
		huella, err = cuadro.HuellaCanonicaSHA256()
	case ports.AccionConsultarDetalleRRHH:
		dominio = ports.DominioHuellaConsultaDetalleRRHH
		huella, err = detalle.HuellaCanonicaSHA256()
		recursoRef = detalle.ExpedienteRef()
	default:
		return ports.MaterialAutorizacionConsultaRRHH{},
			errors.New("acción de prueba no soportada")
	}
	if err != nil {
		t.Fatal(err)
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: recursoRef,
		ModuloID:   ports.ModuloContratacion,
		Tipo:       ports.TipoRecursoCuadroRRHH,
		Ambitos: map[string]string{
			"organizacion_ref": contexto.OrganizacionRef(),
			"clase_ambito":     string(ports.AmbitoOrganizacionRRHH),
			"ambito_ref":       contexto.OrganizacionRef(),
		},
		Atributos: map[string]string{
			"consulta_dominio":       dominio,
			"consulta_huella_sha256": huella,
		},
	}
	if accion == ports.AccionConsultarDetalleRRHH {
		recurso.Tipo = ports.TipoRecursoExpediente
	}
	if mutarRecurso != nil {
		mutarRecurso(&recurso)
	}
	motivo := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion",
		CatalogoVersion:      2,
		CatalogoHuellaSHA256: strings.Repeat("d", 64),
		EntradaClave:         "motivo_11111111111111111111111111111111",
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		generadorCorrelacionRRHHPrueba{
			referencia: "correlacion_11111111111111111111111111111111",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(
		dominiovec.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: autoridad.Vinculo,
			ReferenciaMotivo:          motivo, Accion: accion, Recurso: recurso,
			Finalidad: finalidad, Correlacion: correlacion,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	decisionEn := instante.Add(-2 * time.Millisecond)
	decision, confirmacion := concesionConsultaRRHHPrueba(
		t, solicitud, autoridad.Resultado, motivo, decisionEn,
	)
	decisionHuella, err :=
		dominiovec.HuellaSHA256DecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatal(err)
	}
	motivoHuella, err := dominiovec.HuellaSHA256MotivoAutorizacionV2(motivo)
	if err != nil {
		t.Fatal(err)
	}
	recursoHuella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	datosConfirmacion, err := confirmacion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	emitidaEn := instante.Add(-time.Millisecond)
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3(
		datosConfirmacion.DecisionRef, decisionHuella, motivoHuella,
		autoridad.Resultado.RegistroContextoRef,
		autoridad.Resultado.HuellaSHA256, accion, recurso.Referencia,
		recursoHuella, audiencia, emitidaEn, emitidaEn.Add(5*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	decisionCanonica, err :=
		dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatal(err)
	}
	motivoCanonico, err :=
		dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(motivo)
	if err != nil {
		t.Fatal(err)
	}
	semilla := [ed25519.SeedSize]byte{1}
	privada := ed25519.NewKeyFromSeed(semilla[:])
	raizSPKI, err := x509.MarshalPKIXPublicKey(privada.Public())
	if err != nil {
		t.Fatal(err)
	}
	exportacion, err :=
		puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
			bytes.Repeat([]byte{0xa5}, puertosvec.TamanoMinimoCapacidadCanonicaV3),
			resumen, decisionCanonica, motivoCanonico,
			autoridad.Resultado.RepresentacionCanonica,
			autoridad.Resultado.Contexto.Instantanea.PersonaVersion,
			autoridad.Resultado.Contexto.Instantanea.PerfilVersion,
			[]byte("payload-vec-ad-3-prueba"),
			[]byte("sobre-cose-sign1-prueba"),
			[]byte("evidencia-verificacion-prueba"), raizSPKI,
		)
	if err != nil {
		t.Fatal(err)
	}
	return ports.NuevoMaterialAutorizacionConsultaRRHH(
		contexto, solicitud, decision, confirmacion, autoridad.Resultado,
		exportadorMaterialRRHHPrueba{exportacion: exportacion}, instante,
	)
}
