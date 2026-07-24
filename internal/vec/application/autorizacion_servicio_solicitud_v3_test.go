package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type resolutorContextoAutorizacionV3Prueba struct {
	resultado domain.ResultadoContextoActorRegistradoV2
}

func (r resolutorContextoAutorizacionV3Prueba) ResolverContextoActorRegistradoV2(
	context.Context,
	domain.SolicitudContextoActor,
) (domain.ResultadoContextoActorRegistradoV2, error) {
	return r.resultado, nil
}

type registroConcesionesAutorizacionV3Prueba struct {
	err            error
	invocaciones   int
	orden          ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3
	cancelar       context.CancelFunc
	decisionNoVive bool
	registradaEn   *time.Time
	devolverCero   bool
}

func (r *registroConcesionesAutorizacionV3Prueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	_ context.Context,
	orden ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	r.invocaciones++
	r.orden = orden
	datos, err := orden.Datos()
	if err != nil {
		return time.Time{}, err
	}
	desde, _, err := datos.Decision.VentanaValidez()
	if err != nil {
		return time.Time{}, err
	}
	r.decisionNoVive = !datos.Decision.VigenteEn(desde)
	if r.err != nil {
		return time.Time{}, r.err
	}
	if r.devolverCero {
		return time.Time{}, nil
	}
	if r.registradaEn != nil {
		return *r.registradaEn, nil
	}
	if r.cancelar != nil {
		r.cancelar()
	}
	return desde, nil
}

type registroDenegacionesAutorizacionV3Prueba struct {
	err          error
	invocaciones int
	orden        ports.OrdenRegistroDenegacionAutorizacionLigadaV3
}

func (r *registroDenegacionesAutorizacionV3Prueba) RegistrarDenegacionAutorizacionLigadaV3(
	_ context.Context,
	orden ports.OrdenRegistroDenegacionAutorizacionLigadaV3,
) error {
	r.invocaciones++
	r.orden = orden
	if _, err := orden.Datos(); err != nil {
		return err
	}
	return r.err
}

type entornoAutorizacionSolicitudV3Prueba struct {
	ahora        time.Time
	solicitud    domain.SolicitudAutorizacionLigadaV3
	resultado    domain.ResultadoContextoActorRegistradoV2
	instantanea  domain.InstantaneaAutorizacion
	fuente       *fuenteAutorizacionServicioPrueba
	concesiones  *registroConcesionesAutorizacionV3Prueba
	denegaciones *registroDenegacionesAutorizacionV3Prueba
	motivos      *validadorMotivoAutorizacionV2Prueba
	servicio     *ServicioAutorizacionSolicitudLigadaV3
}

type relojAutorizacionV3ContadorPrueba struct {
	ahora        time.Time
	invocaciones int
}

func (r *relojAutorizacionV3ContadorPrueba) Ahora() time.Time {
	r.invocaciones++
	if r.invocaciones == 1 {
		return r.ahora
	}
	return r.ahora.Add(24 * time.Hour)
}

func nuevoEntornoAutorizacionSolicitudV3Prueba(t *testing.T) *entornoAutorizacionSolicitudV3Prueba {
	t.Helper()
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	solicitudContexto := solicitudServicioContextoActorPrueba()
	actor := contextoActorServicioPrueba(t, ahora.Add(-2*time.Minute), solicitudContexto)
	recibo := confirmacionRegistroContextoActorV2Prueba(
		t, actor, referenciaServicioContextoActorPrueba("oca_", "o"),
	)
	resultado := domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: recibo.RegistroContextoRef, Contexto: recibo.Contexto,
		RepresentacionCanonica: append([]byte(nil), recibo.RepresentacionCanonica...),
		HuellaSHA256:           recibo.HuellaSHA256,
		ManifiestoProcedenciaCanonico: append(
			[]byte(nil), recibo.ManifiestoProcedenciaCanonico...,
		),
		ManifiestoProcedenciaHuellaSHA256: recibo.ManifiestoProcedenciaHuellaSHA256,
		AutoridadEfectiva:                 recibo.AutoridadEfectiva,
		ResueltoEnAutoritativo:            recibo.ResueltoEnAutoritativo,
	}
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef:             referenciaServicioContextoActorPrueba("aut_", "a"),
		AutenticacionHuellaSHA256:    strings.Repeat("1", 64),
		AsercionRef:                  referenciaServicioContextoActorPrueba("ase_", "s"),
		SesionRef:                    referenciaServicioContextoActorPrueba("ses_", "e"),
		ControlSesionRef:             referenciaServicioContextoActorPrueba("cse_", "c"),
		ControlSesionRevision:        2,
		ControlSesionHuellaSHA256:    strings.Repeat("2", 64),
		CuentaRef:                    solicitudContexto.Cuenta.CuentaRef,
		CuentaOrdinariaRef:           solicitudContexto.Cuenta.CuentaRef,
		Superficie:                   domain.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado:              solicitudContexto.Cuenta.Metodo,
		GarantiaObservada:            solicitudContexto.Cuenta.Garantia,
		PoliticaGarantiaRef:          referenciaServicioContextoActorPrueba("pga_", "g"),
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    ahora.Add(-10 * time.Minute),
		SesionEmitidaEn:              ahora.Add(-9 * time.Minute),
		SesionRevalidadaEn:           ahora.Add(-3 * time.Minute),
		SesionValidaHasta:            ahora.Add(20 * time.Minute),
	}
	revalidador := &revalidadorVinculoAplicacionAdversarial{resultado: autenticacion}
	vinculo, err := domain.CrearVinculoAutenticacionActorV2(
		context.Background(), revalidador,
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef,
		},
		resolutorContextoAutorizacionV3Prueba{resultado: resultado}, solicitudContexto,
		&relojAutorizacionServicioPrueba{ahora: ahora},
	)
	if err != nil {
		t.Fatalf("crear vinculo V2: %v", err)
	}
	referenciaMotivo := referenciaMotivoAutorizacionV2Prueba(claveMotivoAutorizacionV2Prueba)
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(
		domain.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: vinculo, ReferenciaMotivo: referenciaMotivo,
			Accion: "bolsa.expediente.leer",
			Recurso: domain.RecursoAutorizable{
				Referencia: "expediente:1", ModuloID: "bolsa", Tipo: "expediente",
				Ambitos: map[string]string{"unidad": "seleccion"},
			},
			Finalidad: "gestion_bolsa",
			Correlacion: referenciaCorrelacionAplicacionPrueba(
				referenciaCorrelacionAutorizacionV2Prueba,
			),
		},
	)
	if err != nil {
		t.Fatalf("crear solicitud V3: %v", err)
	}
	instantanea := instantaneaAutorizacionServicioPrueba(t)
	instantanea.AsignacionPerfil.PrincipalID = actor.Principal.ID
	instantanea.AsignacionPerfil.PerfilActivoRef = actor.PerfilActivoRef
	fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantanea}
	concesiones := &registroConcesionesAutorizacionV3Prueba{}
	denegaciones := &registroDenegacionesAutorizacionV3Prueba{}
	motivos := &validadorMotivoAutorizacionV2Prueba{referencia: referenciaMotivo}
	servicio, err := NuevoServicioAutorizacionSolicitudLigadaV3(
		fuente, concesiones, denegaciones, motivos,
		&relojAutorizacionServicioPrueba{ahora: ahora},
		&generadorAutorizacionServicioPrueba{referencia: "dec_0123456789abcdef0123456789abcdef"},
		ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		t.Fatalf("crear servicio V3: %v", err)
	}
	return &entornoAutorizacionSolicitudV3Prueba{
		ahora: ahora, solicitud: solicitud, resultado: resultado, instantanea: instantanea,
		fuente: fuente, concesiones: concesiones, denegaciones: denegaciones,
		motivos: motivos, servicio: servicio,
	}
}

func resultadoContextoAutorizacionV3AlternativoPrueba(
	t *testing.T,
	ahora time.Time,
) domain.ResultadoContextoActorRegistradoV2 {
	t.Helper()
	solicitud := solicitudServicioContextoActorPrueba()
	solicitud.PerfilActivoRef = referenciaServicioContextoActorPrueba("prf_", "x")
	actor := contextoActorServicioPrueba(t, ahora.Add(-2*time.Minute), solicitud)
	recibo := confirmacionRegistroContextoActorV2Prueba(
		t, actor, referenciaServicioContextoActorPrueba("oca_", "x"),
	)
	return domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: recibo.RegistroContextoRef, Contexto: recibo.Contexto,
		RepresentacionCanonica: append([]byte(nil), recibo.RepresentacionCanonica...),
		HuellaSHA256:           recibo.HuellaSHA256,
		ManifiestoProcedenciaCanonico: append(
			[]byte(nil), recibo.ManifiestoProcedenciaCanonico...,
		),
		ManifiestoProcedenciaHuellaSHA256: recibo.ManifiestoProcedenciaHuellaSHA256,
		AutoridadEfectiva:                 recibo.AutoridadEfectiva,
		ResueltoEnAutoritativo:            recibo.ResueltoEnAutoritativo,
	}
}

func TestServicioAutorizacionSolicitudLigadaV3SoloConcedeTrasConfirmacionDurable(t *testing.T) {
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	decision, confirmacion, err := e.servicio.ExigirSolicitudLigadaV3(
		context.Background(), e.solicitud, e.resultado,
	)
	if err != nil {
		t.Fatalf("exigir V3: %v", err)
	}
	concedida, codigo, err := decision.Resultado()
	datosConfirmacion, errConfirmacion := confirmacion.Datos()
	if err != nil || errConfirmacion != nil || !concedida || codigo != "concedida" ||
		e.concesiones.invocaciones != 1 || e.denegaciones.invocaciones != 0 ||
		!e.concesiones.decisionNoVive || decision.VigenteEn(e.ahora) ||
		!confirmacion.DentroDeVentanaEn(datosConfirmacion.RegistradaEn) {
		t.Fatalf("resultado V3 incoherente: concedida=%t codigo=%q decisionErr=%v confirmacionErr=%v",
			concedida, codigo, err, errConfirmacion)
	}
	datosOrden, err := e.concesiones.orden.Datos()
	if err != nil || datosOrden.ReferenciaMotivo != e.motivos.referencia ||
		datosOrden.ResultadoContexto.Validar() != nil ||
		datosOrden.Decision.ValidarPara(e.solicitud) != nil {
		t.Fatalf("orden durable incompleta: %#v, %v", datosOrden, err)
	}
	if _, v1 := any(e.servicio).(ports.Autorizador); v1 {
		t.Fatal("el servicio V3 implemento por accidente el autorizador V1")
	}
	if _, v2 := any(e.servicio).(ports.AutorizadorSolicitudLigadaV2); v2 {
		t.Fatal("el servicio V3 implemento por accidente el autorizador V2")
	}
}

func TestServicioAutorizacionSolicitudLigadaV3RechazaAdulteracionYContextoCruzado(t *testing.T) {
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	adulterado, err := e.resultado.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	adulterado.RepresentacionCanonica[0] ^= 1
	for nombre, resultado := range map[string]domain.ResultadoContextoActorRegistradoV2{
		"adulterado": adulterado,
		"cruzado":    resultadoContextoAutorizacionV3AlternativoPrueba(t, e.ahora),
	} {
		t.Run(nombre, func(t *testing.T) {
			decision, confirmacion, err := e.servicio.ExigirSolicitudLigadaV3(
				context.Background(), e.solicitud, resultado,
			)
			if !errors.Is(err, domain.ErrAutorizacionDenegada) || decision.Validar() == nil ||
				confirmacion.Validar() == nil || e.fuente.invocaciones != 0 ||
				e.concesiones.invocaciones != 0 || e.denegaciones.invocaciones != 0 {
				t.Fatalf("contexto inseguro aceptado: decision=%v confirmacion=%v err=%v",
					decision, confirmacion, err)
			}
		})
	}
}

func TestServicioAutorizacionSolicitudLigadaV3NoFabricaConfirmacionSustitutiva(t *testing.T) {
	t.Run("valor cero", func(t *testing.T) {
		e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
		e.concesiones.devolverCero = true
		decision, confirmacion, err := e.servicio.ExigirSolicitudLigadaV3(
			context.Background(), e.solicitud, e.resultado,
		)
		concedida, _, errDecision := decision.Resultado()
		if errDecision != nil || !concedida || confirmacion.Validar() == nil ||
			!errors.Is(err, ports.ErrConfirmacionRegistroConcesionAutorizacionLigadaV3Invalida) ||
			e.concesiones.invocaciones != 1 {
			t.Fatalf("el servicio sustituyo confirmacion ausente: decision=%v confirmacion=%v err=%v",
				decision, confirmacion, err)
		}
	})
	t.Run("instante adulterado", func(t *testing.T) {
		e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
		fueraDeVentana := e.ahora.Add(90 * time.Second)
		e.concesiones.registradaEn = &fueraDeVentana
		decision, confirmacion, err := e.servicio.ExigirSolicitudLigadaV3(
			context.Background(), e.solicitud, e.resultado,
		)
		if decision.Validar() != nil || confirmacion.Validar() == nil ||
			!errors.Is(err, ports.ErrConfirmacionRegistroConcesionAutorizacionLigadaV3Invalida) {
			t.Fatalf("instante fuera de ventana aceptado: decision=%v confirmacion=%v err=%v",
				decision, confirmacion, err)
		}
	})
}

func TestConfirmacionRegistroConcesionAutorizacionLigadaV3EsOpacaYNoContienePII(t *testing.T) {
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	_, confirmacion, err := e.servicio.ExigirSolicitudLigadaV3(
		context.Background(), e.solicitud, e.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := confirmacion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	tipo := reflect.TypeOf(datos)
	for indice := 0; indice < tipo.NumField(); indice++ {
		nombre := strings.ToLower(tipo.Field(indice).Name)
		for _, pii := range []string{"principal", "persona", "perfil", "cuenta", "dni", "nombre", "email", "contexto"} {
			if strings.Contains(nombre, pii) {
				t.Fatalf("PII o identidad expuesta en confirmacion: %s", tipo.Field(indice).Name)
			}
		}
	}
	if _, err := json.Marshal(confirmacion); !errors.Is(
		err, ports.ErrSerializacionRegistroAutorizacionLigadaV3Prohibida,
	) {
		t.Fatalf("JSON aceptado: %v", err)
	}
	texto := fmt.Sprintf("%v %#v", confirmacion, confirmacion)
	if texto != "[REGISTRO-AUTORIZACION-LIGADA-V3-OPACO] [REGISTRO-AUTORIZACION-LIGADA-V3-OPACO]" ||
		strings.Contains(texto, e.resultado.Contexto.Principal.ID) {
		t.Fatalf("formateo filtro datos: %q", texto)
	}
}

func TestServicioAutorizacionSolicitudLigadaV3DeniegaYPropagaFalloDeTraza(t *testing.T) {
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	e.instantanea.VersionRol.Concesiones[0].Accion = "bolsa.expediente.modificar"
	e.fuente.instantanea = e.instantanea
	fallo := errors.New("almacen de denegaciones caido")
	e.denegaciones.err = fallo
	decision, confirmacion, err := e.servicio.ExigirSolicitudLigadaV3(
		context.Background(), e.solicitud, e.resultado,
	)
	concedida, codigo, errDecision := decision.Resultado()
	if errDecision != nil || concedida || codigo != "accion_no_concedida" ||
		!errors.Is(err, domain.ErrAutorizacionDenegada) || !errors.Is(err, fallo) ||
		!errors.Is(err, ports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) ||
		confirmacion.Validar() == nil || e.denegaciones.invocaciones != 1 ||
		e.concesiones.invocaciones != 0 {
		t.Fatalf("denegacion/registro incoherentes: concedida=%t codigo=%q error=%v", concedida, codigo, err)
	}
}

func TestServicioAutorizacionSolicitudLigadaV3FallaCerradoEnRegistroYSnapshotObsoleto(t *testing.T) {
	for nombre, fallo := range map[string]error{
		"registro": errors.New("registro caido"),
		"snapshot": ports.ErrInstantaneaAutorizacionObsoleta,
	} {
		t.Run(nombre, func(t *testing.T) {
			e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
			e.concesiones.err = fallo
			decision, confirmacion, err := e.servicio.ExigirSolicitudLigadaV3(
				context.Background(), e.solicitud, e.resultado,
			)
			concedida, _, errDecision := decision.Resultado()
			if errDecision != nil || !concedida || !errors.Is(err, domain.ErrAutorizacionDenegada) ||
				!errors.Is(err, fallo) || confirmacion.Validar() == nil ||
				decision.VigenteEn(e.ahora) || !e.concesiones.decisionNoVive {
				t.Fatalf("fallo abierto: decision=%v confirmacion=%v error=%v", decision, confirmacion, err)
			}
			if nombre == "registro" &&
				!errors.Is(err, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) {
				t.Fatalf("falta error nominal de registro: %v", err)
			}
		})
	}
}

func TestServicioAutorizacionSolicitudLigadaV3RespetaCancelacionEnFronteras(t *testing.T) {
	t.Run("antes", func(t *testing.T) {
		e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		_, confirmacion, err := e.servicio.ExigirSolicitudLigadaV3(ctx, e.solicitud, e.resultado)
		if !errors.Is(err, context.Canceled) || confirmacion.Validar() == nil ||
			e.fuente.invocaciones != 0 || e.concesiones.invocaciones != 0 {
			t.Fatalf("cancelacion previa ignorada: %v", err)
		}
	})
	t.Run("tras fuente", func(t *testing.T) {
		e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
		ctx, cancelar := context.WithCancel(context.Background())
		e.fuente.despues = cancelar
		_, confirmacion, err := e.servicio.ExigirSolicitudLigadaV3(ctx, e.solicitud, e.resultado)
		if !errors.Is(err, context.Canceled) || confirmacion.Validar() == nil ||
			e.concesiones.invocaciones != 0 {
			t.Fatalf("cancelacion tras fuente ignorada: %v", err)
		}
	})
	t.Run("tras commit", func(t *testing.T) {
		e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
		ctx, cancelar := context.WithCancel(context.Background())
		e.concesiones.cancelar = cancelar
		_, confirmacion, err := e.servicio.ExigirSolicitudLigadaV3(ctx, e.solicitud, e.resultado)
		if err != nil || confirmacion.Validar() != nil ||
			e.concesiones.invocaciones != 1 {
			t.Fatalf("cancelacion tardia oculto commit durable: confirmacion=%v err=%v", confirmacion, err)
		}
	})
}

func TestServicioAutorizacionSolicitudLigadaV3NoIntroduceFalloTrasCommit(t *testing.T) {
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	reloj := &relojAutorizacionV3ContadorPrueba{ahora: e.ahora}
	e.servicio.reloj = reloj
	_, confirmacion, err := e.servicio.ExigirSolicitudLigadaV3(
		context.Background(), e.solicitud, e.resultado,
	)
	if err != nil || confirmacion.Validar() != nil || e.concesiones.invocaciones != 1 ||
		reloj.invocaciones != 1 {
		t.Fatalf(
			"se consulto/fallo una dependencia tras commit: reloj=%d confirmacion=%v err=%v",
			reloj.invocaciones, confirmacion, err,
		)
	}
}

func TestServicioAutorizacionSolicitudLigadaV3SaneaErrorDeAdaptador(t *testing.T) {
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	secreto := "postgres://usuario:clave@servidor/vec persona=12345678Z"
	fallo := errors.New(secreto)
	e.fuente.err = fallo
	_, confirmacion, err := e.servicio.ExigirSolicitudLigadaV3(
		context.Background(), e.solicitud, e.resultado,
	)
	if !errors.Is(err, fallo) || !errors.Is(err, ports.ErrFuenteAutorizacionNoDisponible) ||
		strings.Contains(err.Error(), secreto) || confirmacion.Validar() == nil {
		t.Fatalf("error de adaptador no saneado o no trazable: %v", err)
	}
	formateado := fmt.Sprintf("%v %+v %#v", err, err, err)
	var bitacora bytes.Buffer
	slog.New(slog.NewTextHandler(&bitacora, nil)).Error("fallo", "error", err)
	if strings.Contains(formateado, secreto) || strings.Contains(bitacora.String(), secreto) {
		t.Fatalf("fmt/slog filtro error de adaptador: fmt=%q slog=%q", formateado, bitacora.String())
	}
}

func TestNuevoServicioAutorizacionSolicitudLigadaV3RechazaNulosTipados(t *testing.T) {
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	var fuente *fuenteAutorizacionServicioPrueba
	var concesiones *registroConcesionesAutorizacionV3Prueba
	var denegaciones *registroDenegacionesAutorizacionV3Prueba
	var motivos *validadorMotivoAutorizacionV2Prueba
	var reloj *relojAutorizacionServicioPrueba
	var generador *generadorAutorizacionServicioPrueba
	casos := []struct {
		fuente       ports.FuenteAutorizacion
		concesiones  ports.RegistroConcesionesCandidatasAutorizacionLigadaV3
		denegaciones ports.RegistroDenegacionesAutorizacionLigadaV3
		motivos      ports.ValidadorReferenciaMotivoAutorizacionV2
		reloj        ports.Reloj
		generador    ports.GeneradorReferenciaDecisionAutorizacion
	}{
		{fuente, e.concesiones, e.denegaciones, e.motivos, &relojAutorizacionServicioPrueba{ahora: e.ahora}, &generadorAutorizacionServicioPrueba{referencia: "dec_ok"}},
		{e.fuente, concesiones, e.denegaciones, e.motivos, &relojAutorizacionServicioPrueba{ahora: e.ahora}, &generadorAutorizacionServicioPrueba{referencia: "dec_ok"}},
		{e.fuente, e.concesiones, denegaciones, e.motivos, &relojAutorizacionServicioPrueba{ahora: e.ahora}, &generadorAutorizacionServicioPrueba{referencia: "dec_ok"}},
		{e.fuente, e.concesiones, e.denegaciones, motivos, &relojAutorizacionServicioPrueba{ahora: e.ahora}, &generadorAutorizacionServicioPrueba{referencia: "dec_ok"}},
		{e.fuente, e.concesiones, e.denegaciones, e.motivos, reloj, &generadorAutorizacionServicioPrueba{referencia: "dec_ok"}},
		{e.fuente, e.concesiones, e.denegaciones, e.motivos, &relojAutorizacionServicioPrueba{ahora: e.ahora}, generador},
	}
	for indice, caso := range casos {
		servicio, err := NuevoServicioAutorizacionSolicitudLigadaV3(
			caso.fuente, caso.concesiones, caso.denegaciones, caso.motivos,
			caso.reloj, caso.generador, ConfiguracionServicioAutorizacion{},
		)
		if servicio != nil || !errors.Is(err, domain.ErrConfiguracionAccesoInvalida) {
			t.Fatalf("nulo tipado %d aceptado: servicio=%v error=%v", indice, servicio, err)
		}
	}
}
