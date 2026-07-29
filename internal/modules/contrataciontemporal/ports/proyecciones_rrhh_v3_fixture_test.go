package ports_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
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

type emisorMaterialPuertosRRHHPrueba struct {
	t         *testing.T
	audiencia string
	instante  time.Time
}

func (e *emisorMaterialPuertosRRHHPrueba) EmitirMaterialAutorizacionAtestadaV3(
	_ context.Context,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
) (
	dominiovec.DecisionAutorizacionLigadaV3,
	puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3,
	error,
) {
	e.t.Helper()
	datos, err := solicitud.Datos()
	if err != nil {
		e.t.Fatal(err)
	}
	decision, confirmacion := concesionConsultaRRHHPrueba(
		e.t, solicitud, resultado, datos.ReferenciaMotivo,
		e.instante.Add(-2*time.Millisecond),
	)
	decisionHuella, err :=
		dominiovec.HuellaSHA256DecisionAutorizacionV3(decision)
	if err != nil {
		e.t.Fatal(err)
	}
	motivoHuella, err := dominiovec.HuellaSHA256MotivoAutorizacionV2(
		datos.ReferenciaMotivo,
	)
	if err != nil {
		e.t.Fatal(err)
	}
	recursoHuella, err := datos.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		e.t.Fatal(err)
	}
	datosConfirmacion, err := confirmacion.Datos()
	if err != nil {
		e.t.Fatal(err)
	}
	emitidaEn := e.instante.Add(-time.Millisecond)
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3(
		datosConfirmacion.DecisionRef, decisionHuella, motivoHuella,
		resultado.RegistroContextoRef, resultado.HuellaSHA256,
		datos.Accion, datos.Recurso.Referencia, recursoHuella,
		e.audiencia, emitidaEn, emitidaEn.Add(5*time.Second),
	)
	if err != nil {
		e.t.Fatal(err)
	}
	decisionCanonica, err :=
		dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		e.t.Fatal(err)
	}
	motivoCanonico, err :=
		dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(
			datos.ReferenciaMotivo,
		)
	if err != nil {
		e.t.Fatal(err)
	}
	semilla := [ed25519.SeedSize]byte{1}
	privada := ed25519.NewKeyFromSeed(semilla[:])
	raizSPKI, err := x509.MarshalPKIXPublicKey(privada.Public())
	if err != nil {
		e.t.Fatal(err)
	}
	exportacion, err :=
		puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
			bytes.Repeat([]byte{0xa5}, puertosvec.TamanoMinimoCapacidadCanonicaV3),
			resumen, decisionCanonica, motivoCanonico,
			resultado.RepresentacionCanonica,
			resultado.Contexto.Instantanea.PersonaVersion,
			resultado.Contexto.Instantanea.PerfilVersion,
			[]byte("payload-vec-ad-3-prueba"),
			[]byte("sobre-cose-sign1-prueba"),
			[]byte("evidencia-verificacion-prueba"), raizSPKI,
		)
	if err != nil {
		e.t.Fatal(err)
	}
	return decision, confirmacion,
		exportadorMaterialRRHHPrueba{exportacion: exportacion}, nil
}

func autoridadContextoPuertosRRHH(
	t *testing.T,
	ahora time.Time,
	marcaActor, marcaPerfil string,
) ports.ContextoAutorizacionAltaV3 {
	t.Helper()
	return autoridadContextoPuertosRRHHPersonalizada(
		t, ahora, marcaActor, marcaPerfil, nil, nil,
	)
}

func autoridadContextoPuertosRRHHPersonalizada(
	t *testing.T,
	ahora time.Time,
	marcaActor, marcaPerfil string,
	mutarInstantanea func(*dominiovec.InstantaneaContextoActor),
	mutarAutenticacion func(*dominiovec.AutenticacionRevalidadaV1),
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
	if mutarInstantanea != nil {
		mutarInstantanea(&instantanea)
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
	if mutarAutenticacion != nil {
		mutarAutenticacion(&autenticacion)
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
	contexto ports.ContextoConsultaRRHH,
	solicitud ports.SolicitudCuadroRRHH,
	instante time.Time,
) ports.CapacidadConsultaRRHH {
	t.Helper()
	material, err := materialCuadroPuertosRRHH(
		t, contexto, solicitud, instante,
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
	contexto ports.ContextoConsultaRRHH,
	solicitud ports.SolicitudDetalleRRHH,
	instante time.Time,
) ports.CapacidadConsultaRRHH {
	t.Helper()
	material, err := materialDetallePuertosRRHH(
		t, contexto, solicitud, instante,
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

func materialCuadroPuertosRRHH(
	t *testing.T,
	contexto ports.ContextoConsultaRRHH,
	cuadro ports.SolicitudCuadroRRHH,
	instante time.Time,
) (ports.MaterialAutorizacionConsultaRRHH, error) {
	t.Helper()
	guardian := nuevoEmisorMaterialPuertosRRHHPrueba(t, instante)
	return guardian.EmitirMaterialCuadroRRHH(
		context.Background(), contexto, cuadro,
	)
}

func materialDetallePuertosRRHH(
	t *testing.T,
	contexto ports.ContextoConsultaRRHH,
	detalle ports.SolicitudDetalleRRHH,
	instante time.Time,
) (ports.MaterialAutorizacionConsultaRRHH, error) {
	t.Helper()
	guardian := nuevoEmisorMaterialPuertosRRHHPrueba(t, instante)
	return guardian.EmitirMaterialDetalleRRHH(
		context.Background(), contexto, detalle,
	)
}

func nuevoEmisorMaterialPuertosRRHHPrueba(
	t *testing.T,
	instante time.Time,
) *ports.EmisorMaterialConsultaRRHH {
	t.Helper()
	motivo := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion",
		CatalogoVersion:      2,
		CatalogoHuellaSHA256: strings.Repeat("d", 64),
		EntradaClave:         "motivo_11111111111111111111111111111111",
	}
	motivos := &resolutorMotivoGuardianConsultaRRHHPrueba{
		motivoCuadro: motivo, motivoDetalle: motivo,
	}
	correlaciones := &generadorCorrelacionGuardianConsultaRRHHPrueba{
		referencia: "correlacion_11111111111111111111111111111111",
	}
	reloj := &relojGuardianConsultaRRHHPrueba{
		instantes: []time.Time{instante, instante},
	}
	cuadro := &emisorMaterialPuertosRRHHPrueba{
		t: t, audiencia: ports.AudienciaConsumoConsultaCuadroRRHHV3,
		instante: instante,
	}
	detalle := &emisorMaterialPuertosRRHHPrueba{
		t: t, audiencia: ports.AudienciaConsumoConsultaDetalleRRHHV3,
		instante: instante,
	}
	emisor, err := ports.NuevoEmisorMaterialConsultaRRHH(
		motivos, correlaciones, reloj, cuadro, detalle,
	)
	if err != nil {
		t.Fatalf("crear emisor A4.3 del fixture de puertos: %v", err)
	}
	return emisor
}
