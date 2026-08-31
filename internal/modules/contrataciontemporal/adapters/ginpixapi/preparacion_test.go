package ginpixapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/ginpixfichero"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const datoPersonalSinteticoPrueba = "DATO-PERSONAL-SINTETICO-NO-REGISTRAR"

func TestPrepararReutilizaO701O702O703YO705SinActivarEnvio(t *testing.T) {
	preparacion, solicitud := preparacionAPIGINPIXPrueba(t)
	cuerpo, err := preparacion.Cuerpo()
	if err != nil {
		t.Fatalf("obtener cuerpo preparado: %v", err)
	}
	modelo, _ := solicitud.Modelo()
	mapeo, _ := solicitud.Mapeo()
	carga, err := domain.AplicarMapeoGINPIX(modelo, mapeo)
	if err != nil {
		t.Fatalf("aplicar mapeo de referencia: %v", err)
	}
	esperado, err := ginpixfichero.Codificar(carga)
	if err != nil || !bytes.Equal(cuerpo, esperado) {
		t.Fatalf("la API no conserva los bytes O7-05: %v", err)
	}
	metadatos, err := preparacion.Metadatos()
	if err != nil || metadatos.ExpedienteRef != "expediente_sintetico_api_0001" ||
		metadatos.IncorporacionRef != "actuacion_incorporacion_api_0001" ||
		metadatos.VersionExpediente != 7 ||
		metadatos.CorrelacionRef != "correlacion_ginpix_api_0001" ||
		metadatos.IdempotenciaRef != "idempotencia_ginpix_api_0001" ||
		metadatos.ReciboIncorporacionRef != "recibo_incorporacion_api_0001" ||
		metadatos.ResultadoPersonalRef != "resultado_personal_api_0001" ||
		metadatos.ReciboPersonalRef != "recibo_personal_api_0001" {
		t.Fatalf("ligaduras incompletas: %+v / %v", metadatos, err)
	}
	if strings.Contains(fmt.Sprint(metadatos), datoPersonalSinteticoPrueba) {
		t.Fatal("los metadatos expusieron la carga funcional")
	}
}

func TestPreparacionClonaCuerpoYEntradasMutables(t *testing.T) {
	preparacion, solicitud := preparacionAPIGINPIXPrueba(t)
	primero, _ := preparacion.Cuerpo()
	primero[0] ^= 0xff
	segundo, err := preparacion.Cuerpo()
	if err != nil || bytes.Equal(primero, segundo) {
		t.Fatalf("el cuerpo conserva alias mutable: %v", err)
	}

	modelo, _ := solicitud.Modelo()
	publicacion := modelo.Publicacion()
	publicacion.Datos[0].Campo.Valor = "MUTADO"
	tercero, err := preparacion.Cuerpo()
	if err != nil || !bytes.Equal(segundo, tercero) {
		t.Fatalf("la preparación retuvo alias del modelo: %v", err)
	}

	solicitudEntrada, orden, recibo := insumosPreparacionAPIGINPIXPrueba(t)
	preparacionEntrada, err := Preparar(solicitudEntrada, orden, recibo)
	if err != nil {
		t.Fatal(err)
	}
	recibo.Documentos[0].Referencia = "documento_mutado_fuera_preparacion"
	if preparacionEntrada.Validar() != nil {
		t.Fatal("la preparación retuvo alias del recibo O7-02")
	}
}

func TestPrepararDeniegaLigadurasIncorporacionAlteradas(t *testing.T) {
	solicitud, orden, recibo := insumosPreparacionAPIGINPIXPrueba(t)
	cases := map[string]func(*ports.ReciboConfirmacionIncorporacion){
		"expediente": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.ExpedienteRef = "expediente_sintetico_api_otro"
		},
		"incorporacion": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.ActuacionRef = "actuacion_incorporacion_api_otra"
		},
		"version OCC":        func(r *ports.ReciboConfirmacionIncorporacion) { r.VersionExpediente++ },
		"recibo Personal":    func(r *ports.ReciboConfirmacionIncorporacion) { r.ReciboPersonalRef = "" },
		"resultado Personal": func(r *ports.ReciboConfirmacionIncorporacion) { r.ResultadoPersonalRef = "" },
		"version resultante": func(r *ports.ReciboConfirmacionIncorporacion) { r.VersionResultante++ },
		"decision V3": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.DecisionAutorizacionRef = "decision_v3_api_fabricada"
		},
		"actor": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.ActorRef = "actor_rrhh_api_fabricado"
		},
		"correlacion": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.CorrelacionRef = "correlacion_incorporacion_api_fabricada"
		},
		"fecha incorporacion": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.FechaIncorporacion = r.FechaIncorporacion.Add(time.Hour)
		},
		"documento sustituido": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.Documentos[0].Referencia = "documento_incorporacion_api_fabricado"
		},
		"documento duplicado": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.Documentos = append(r.Documentos, r.Documentos[0])
		},
	}
	for nombre, alterar := range cases {
		t.Run(nombre, func(t *testing.T) {
			adulterado := recibo
			adulterado.Documentos = append([]domain.DocumentoSeguimiento(nil), recibo.Documentos...)
			alterar(&adulterado)
			if _, err := Preparar(
				solicitud,
				orden,
				adulterado,
			); !errorsIs(err, ErrPreparacionAPIGINPIXInvalida) {
				t.Fatalf("ligadura alterada aceptada: %v", err)
			}
		})
	}
	if _, err := Preparar(
		ports.SolicitudMapeoGINPIX{},
		orden,
		recibo,
	); !errorsIs(err, ErrPreparacionAPIGINPIXInvalida) {
		t.Fatalf("solicitud O7-03 cero aceptada: %v", err)
	}
	if _, err := Preparar(
		solicitud,
		ports.OrdenConfirmarIncorporacion{},
		recibo,
	); !errorsIs(err, ErrPreparacionAPIGINPIXInvalida) {
		t.Fatalf("orden O7-02 fabricada aceptada: %v", err)
	}
}

func TestPrepararDeniegaReciboO702FabricadoQueSuperariaValidacionParcial(t *testing.T) {
	solicitud, orden, recibo := insumosPreparacionAPIGINPIXPrueba(t)
	recibo.DecisionAutorizacionRef = "decision_v3_api_fabricada"
	recibo.ActorRef = "actor_rrhh_api_fabricado"
	recibo.CorrelacionRef = "correlacion_incorporacion_api_fabricada"
	recibo.VersionAnterior++
	recibo.VersionResultante++
	recibo.FechaIncorporacion = recibo.FechaIncorporacion.Add(time.Hour)
	recibo.Documentos[0].Referencia = "documento_incorporacion_api_fabricado"
	if _, err := Preparar(solicitud, orden, recibo); !errors.Is(err, ErrPreparacionAPIGINPIXInvalida) {
		t.Fatalf("recibo fabricado aceptado sin ValidarPara: %v", err)
	}
}

func TestPreparacionRevalidaOrdenAutenticaEnCadaUso(t *testing.T) {
	preparacion, _ := preparacionAPIGINPIXPrueba(t)
	preparacion.datos.incorporacion.DecisionAutorizacionRef = "decision_v3_api_fabricada"
	if err := preparacion.Validar(); !errors.Is(err, ErrPreparacionAPIGINPIXInvalida) {
		t.Fatalf("la preparacion dejo de transportar la orden autentica: %v", err)
	}
}

func preparacionAPIGINPIXPrueba(t *testing.T) (Preparacion, ports.SolicitudMapeoGINPIX) {
	t.Helper()
	solicitud, orden, recibo := insumosPreparacionAPIGINPIXPrueba(t)
	preparacion, err := Preparar(solicitud, orden, recibo)
	if err != nil {
		t.Fatalf("preparar operación API sintética: %v", err)
	}
	return preparacion, solicitud
}

func insumosPreparacionAPIGINPIXPrueba(
	t *testing.T,
) (
	ports.SolicitudMapeoGINPIX,
	ports.OrdenConfirmarIncorporacion,
	ports.ReciboConfirmacionIncorporacion,
) {
	t.Helper()
	solicitudPersonal := ports.SolicitudAltaPersonalRPT{
		Esquema: ports.EsquemaAltaPersonalRPT, ContratoVersion: ports.VersionContratoAltaPersonalRPT,
		SolicitudRef: "solicitud_personal_api_0001", ExpedienteRef: "expediente_sintetico_api_0001",
		VersionExpediente: 7, CapacidadRef: "capacidad_personal_api_0001",
		CorrelacionRef: "correlacion_personal_api_0001", IdempotenciaRef: "idempotencia_personal_api_0001",
		FuenteRPT: ports.ReferenciaVersionadaPersonalRPT{
			Referencia: "rpt_sintetica_api_0001", Version: 3, HuellaSHA256: strings.Repeat("a", 64),
		},
		PuestoRef: "puesto_sintetico_api_0001", PlazaRef: "plaza_sintetica_api_0001",
	}
	huellaSolicitud, err := solicitudPersonal.HuellaSHA256()
	if err != nil {
		t.Fatalf("validar contrato O7-01: %v", err)
	}
	resultadoPersonal := ports.ResultadoAltaPersonalRPT{
		Esquema: ports.EsquemaAltaPersonalRPT, ContratoVersion: ports.VersionContratoAltaPersonalRPT,
		ResultadoRef: "resultado_personal_api_0001", ReciboRef: "recibo_personal_api_0001",
		SolicitudRef: solicitudPersonal.SolicitudRef, CorrelacionRef: solicitudPersonal.CorrelacionRef,
		IdempotenciaRef: solicitudPersonal.IdempotenciaRef, HuellaSolicitudSHA256: huellaSolicitud,
		Estado:      ports.AltaPersonalRPTConfirmada,
		RelacionRef: "relacion_personal_api_0001", OcupacionRef: "ocupacion_personal_api_0001",
	}
	if err := resultadoPersonal.ValidarPara(solicitudPersonal); err != nil {
		t.Fatalf("resultado O7-01 inválido: %v", err)
	}
	instante := time.Date(2026, 8, 31, 9, 30, 0, 0, time.UTC)
	datosConfirmacion := ports.DatosConfirmacionIncorporacion{
		SolicitudPersonal: solicitudPersonal, ResultadoPersonal: resultadoPersonal,
		VersionSeguimientoEsperada: 3,
		PeriodoIncorporacion: domain.IntervaloSeguimiento{
			Desde: instante.Add(24 * time.Hour), Hasta: instante.AddDate(0, 1, 0),
		},
		MotivoClave: "incorporacion_confirmada",
		Documentos: []domain.DocumentoSeguimiento{{
			TipoClave: "resolucion_incorporacion", Referencia: "documento_incorporacion_api_0001",
		}},
	}
	contexto := contextoConfirmacionAPIGINPIXPrueba(t, datosConfirmacion, instante)
	orden, err := ports.NuevaOrdenConfirmarIncorporacion(contexto, datosConfirmacion, instante)
	if err != nil {
		t.Fatalf("crear orden O7-02 autentica: %v", err)
	}
	confirmacionV3, err := contexto.ConfirmacionRegistroV3.Datos()
	if err != nil {
		t.Fatalf("leer confirmacion V3: %v", err)
	}
	preparacionSeguimiento := contexto.PreparacionSeguimiento
	recibo := ports.ReciboConfirmacionIncorporacion{
		ReciboRef: "recibo_incorporacion_api_0001", ActuacionRef: "actuacion_incorporacion_api_0001",
		CorrelacionRef: preparacionSeguimiento.CorrelacionRef, ActorRef: preparacionSeguimiento.ActorRef,
		OrganizacionRef: preparacionSeguimiento.OrganizacionRef, UnidadRef: preparacionSeguimiento.UnidadRef,
		ExpedienteRef: solicitudPersonal.ExpedienteRef, SolicitudPersonalRef: solicitudPersonal.SolicitudRef,
		DecisionAutorizacionRef: confirmacionV3.DecisionRef,
		ResultadoPersonalRef:    resultadoPersonal.ResultadoRef, ReciboPersonalRef: resultadoPersonal.ReciboRef,
		RelacionRef: resultadoPersonal.RelacionRef, OcupacionRef: resultadoPersonal.OcupacionRef,
		TransicionClave: ports.TransicionConfirmarIncorporacion, MotivoClave: datosConfirmacion.MotivoClave,
		VersionExpediente: 7, VersionAnterior: 3, VersionResultante: 4,
		FechaIncorporacion: datosConfirmacion.PeriodoIncorporacion.Desde,
		FechaFinPrevista:   datosConfirmacion.PeriodoIncorporacion.Hasta,
		ConfirmadaEn:       instante,
		Documentos:         append([]domain.DocumentoSeguimiento(nil), datosConfirmacion.Documentos...),
	}
	if err := recibo.ValidarPara(orden); err != nil {
		t.Fatalf("recibo O7-02 de referencia invalido: %v", err)
	}
	campo, err := domain.CampoValorGINPIX(datoPersonalSinteticoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	modelo, err := domain.NuevoModeloCanonicoGINPIX(domain.BorradorModeloCanonicoGINPIX{
		Esquema: domain.EsquemaModeloCanonicoGINPIXV1, VersionExpediente: 7,
		ExpedienteRef: recibo.ExpedienteRef, IncorporacionRef: recibo.ActuacionRef,
		ProcedenciaRef: "procedencia_modelo_api_0001", CorrelacionRef: "correlacion_ginpix_api_0001",
		IdempotenciaRef: "idempotencia_ginpix_api_0001",
		Datos:           []domain.DatoCanonicoGINPIX{{Clave: "codigo_puesto", Campo: campo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	mapeo, err := domain.PublicarMapeoVersionadoGINPIX(domain.BorradorMapeoVersionadoGINPIX{
		Esquema: domain.EsquemaMapeoGINPIXV1, Referencia: "mapeo_api_0001", Version: 2,
		ProcedenciaRef: "procedencia_mapeo_api_0001",
		Reglas: []domain.ReglaMapeoGINPIX{{
			CampoCanonico: "codigo_puesto", CampoDestino: "puesto", Obligatorio: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := ports.NuevaSolicitudMapeoGINPIX(modelo, mapeo)
	if err != nil {
		t.Fatal(err)
	}
	return solicitud, orden, recibo
}

func contextoConfirmacionAPIGINPIXPrueba(
	t *testing.T,
	datos ports.DatosConfirmacionIncorporacion,
	ahora time.Time,
) ports.ContextoConfirmacionIncorporacion {
	t.Helper()
	autoridad := autoridadConfirmacionAPIGINPIXPrueba(t, ahora)
	vinculo, err := autoridad.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		generadorCorrelacionAPIGINPIXPrueba("correlacion_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
	)
	if err != nil {
		t.Fatal(err)
	}
	correlacionV3, err := correlacion.ValorCanonico()
	if err != nil {
		t.Fatal(err)
	}
	motivo := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("c", 64),
		EntradaClave:         "motivo_0123456789abcdef0123456789abcdef",
	}
	preparacion := ports.PreparacionSeguimientoConfirmacionIncorporacion{
		OrganizacionRef: "organizacion_api_0001", UnidadRef: "unidad_rrhh_api_0001",
		ActorRef: "actor_rrhh_api_0001", CorrelacionRef: "correlacion_incorporacion_api_0001",
	}
	solicitudDatos := dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: autoridad.Vinculo,
		ReferenciaMotivo:          motivo,
		Accion:                    ports.AccionConfirmarIncorporacion,
		Recurso: dominiovec.RecursoAutorizable{
			Referencia: datos.SolicitudPersonal.ExpedienteRef,
			ModuloID:   ports.ModuloContratacion,
			Tipo:       ports.TipoRecursoConfirmacionIncorporacion,
			Ambitos: map[string]string{
				"organizacion_ref": preparacion.OrganizacionRef,
				"unidad_ref":       preparacion.UnidadRef,
			},
			Atributos: map[string]string{
				"resultado_personal_ref":       datos.ResultadoPersonal.ResultadoRef,
				"relacion_ref":                 datos.ResultadoPersonal.RelacionRef,
				"ocupacion_ref":                datos.ResultadoPersonal.OcupacionRef,
				"version_expediente_esperada":  strconv.FormatUint(datos.SolicitudPersonal.VersionExpediente, 10),
				"version_seguimiento_esperada": strconv.FormatUint(datos.VersionSeguimientoEsperada, 10),
				"principal_v3_ref":             vinculo.PrincipalID,
				"actor_seguimiento_ref":        preparacion.ActorRef,
				"correlacion_v3_ref":           correlacionV3,
				"correlacion_seguimiento_ref":  preparacion.CorrelacionRef,
				"motivo_v3_ref":                motivo.EntradaClave,
			},
		},
		Finalidad:   ports.FinalidadConfirmarIncorporacion,
		Correlacion: correlacion,
	}
	solicitudV3, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(solicitudDatos)
	if err != nil {
		t.Fatal(err)
	}
	decision, confirmacion := concesionConfirmacionAPIGINPIXPrueba(
		t, solicitudV3, autoridad.Resultado, motivo, ahora,
	)
	return ports.ContextoConfirmacionIncorporacion{
		SolicitudContexto: ports.SolicitudResolverContextoAutorizacionAltaV3{
			AutenticacionRef: vinculo.AutenticacionRef,
			SesionRef:        vinculo.SesionRef,
			PerfilRef:        vinculo.PerfilActivoRef,
		},
		ContextoAutorizacion: autoridad, PreparacionSeguimiento: preparacion,
		SolicitudAutorizacionV3: solicitudV3, DecisionAutorizacionV3: decision,
		ConfirmacionRegistroV3: confirmacion,
	}
}

type revalidadorAutenticacionAPIGINPIXPrueba struct {
	resultado dominiovec.AutenticacionRevalidadaV1
}

func (r revalidadorAutenticacionAPIGINPIXPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

type resolutorContextoActorAPIGINPIXPrueba struct {
	resultado dominiovec.ResultadoContextoActorRegistradoV2
}

func (r resolutorContextoActorAPIGINPIXPrueba) ResolverContextoActorRegistradoV2(
	context.Context,
	dominiovec.SolicitudContextoActor,
) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	return r.resultado, nil
}

type relojAutenticacionAPIGINPIXPrueba struct{ instante time.Time }

func (r relojAutenticacionAPIGINPIXPrueba) Ahora() time.Time { return r.instante }

func autoridadConfirmacionAPIGINPIXPrueba(
	t *testing.T,
	ahora time.Time,
) ports.ContextoAutorizacionAltaV3 {
	t.Helper()
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl", Metodo: dominiovec.AuthMethodCertificate,
		Garantia: dominiovec.AuthAssuranceHigh,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 3,
		CuentaRef: cuenta.CuentaRef, CuentaVersion: 4,
		PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 2,
		PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 5,
		Estado:       dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
	}
	actor, err := dominiovec.NuevoContextoActor(cuenta, instantanea, ahora.Add(-2*time.Minute))
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
		ProcedenciaRef: "prc_0123456789abcdefghijkl", ProcedenciaVersion: 1,
		ProcedenciaHuellaSHA256: strings.Repeat("4", 64),
		ProcedenciaAutoridad:    dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := dominiovec.ManifiestoProcedenciaContextoActorV1{
		Esquema:           dominiovec.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
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
	manifiestoHuella, err := dominiovec.HuellaSHA256ManifiestoProcedenciaContextoActorV1(
		manifiestoCanon,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado := dominiovec.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn", Contexto: actor,
		RepresentacionCanonica: canon, HuellaSHA256: huella,
		ManifiestoProcedenciaCanonico:     manifiestoCanon,
		ManifiestoProcedenciaHuellaSHA256: manifiestoHuella,
		AutoridadEfectiva:                 dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo:            actor.ResueltoEn,
	}
	autenticacion := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef:          "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef:               "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl",
		ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 2,
		ControlSesionHuellaSHA256: strings.Repeat("2", 64),
		CuentaRef:                 cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef,
		Superficie:      dominiovec.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    ahora.Add(-10 * time.Minute),
		SesionEmitidaEn:              ahora.Add(-9 * time.Minute), SesionValidaHasta: ahora.Add(20 * time.Minute),
		SesionRevalidadaEn: ahora.Add(-3 * time.Minute),
	}
	vinculo, err := dominiovec.CrearVinculoAutenticacionActorV2(
		context.Background(), revalidadorAutenticacionAPIGINPIXPrueba{autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef,
		},
		resolutorContextoActorAPIGINPIXPrueba{resultado},
		dominiovec.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: instantanea.PerfilActivoRef},
		relojAutenticacionAPIGINPIXPrueba{ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	return ports.ContextoAutorizacionAltaV3{Vinculo: vinculo, Resultado: resultado}
}

type generadorCorrelacionAPIGINPIXPrueba string

func (g generadorCorrelacionAPIGINPIXPrueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return string(g), nil
}

func (generadorCorrelacionAPIGINPIXPrueba) NuevaClaveMotivoAutorizacionV2(
	context.Context,
) (string, error) {
	return "", errors.New("operacion no usada")
}

type registroConcesionAPIGINPIXPrueba struct{ instante time.Time }

func (r registroConcesionAPIGINPIXPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	context.Context,
	puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	return r.instante, nil
}

func concesionConfirmacionAPIGINPIXPrueba(
	t *testing.T,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	ahora time.Time,
) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3) {
	t.Helper()
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ambitos := make([]dominiovec.AmbitoPerfil, 0, len(datos.Recurso.Ambitos))
	for clave, valor := range datos.Recurso.Ambitos {
		ambitos = append(ambitos, dominiovec.AmbitoPerfil{Clave: clave, Valores: []string{valor}})
	}
	version := dominiovec.VersionRol{
		RolID: "tecnico_rrhh", Version: 1, Nombre: "Tecnico RRHH",
		Estado: dominiovec.EstadoVersionRolPublicada,
		Concesiones: []dominiovec.ConcesionRol{{
			Accion: datos.Accion, ModuloID: datos.Recurso.ModuloID, TipoRecurso: datos.Recurso.Tipo,
			Finalidades: []string{datos.Finalidad}, GarantiaMinima: dominiovec.AuthAssuranceSubstantial,
		}},
		PublicadaPor: "responsable-seguridad", PublicadaEn: ahora.Add(-24 * time.Hour),
	}
	huellaCatalogo, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := dominiovec.InstantaneaAutorizacion{
		AsignacionPerfil: dominiovec.AsignacionPerfil{
			AsignacionID: "asig-contratacion-temporal", Version: 1,
			PerfilActivoRef: vinculo.PerfilActivoRef, PrincipalID: vinculo.PrincipalID,
			VersionRolRef: version.Referencia(), Estado: dominiovec.EstadoAsignacionPerfilActiva,
			Ambitos: ambitos, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
			EmitidaPor: "administrador-identidades", EmitidaEn: ahora.Add(-2 * time.Hour),
		},
		VersionRol: version,
		ControlVigenciaVersionRol: dominiovec.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         dominiovec.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn,
		},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	evidencia, err := dominiovec.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea, "dec_0123456789abcdef0123456789abcdef", ahora, ahora.Add(90*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := dominiovec.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	orden, err := puertosvec.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		solicitud, decision, motivo, resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion, err := puertosvec.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
		context.Background(), registroConcesionAPIGINPIXPrueba{ahora}, orden,
	)
	if err != nil {
		t.Fatal(err)
	}
	return decision, confirmacion
}

func errorsIs(err, objetivo error) bool {
	return errors.Is(err, objetivo)
}
