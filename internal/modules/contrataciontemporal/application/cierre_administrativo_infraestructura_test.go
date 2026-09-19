package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var instanteCierreAdministrativoPrueba = time.Date(
	2026, time.August, 21, 10, 0, 0, 0, time.UTC,
)

type transaccionCierreAdministrativoPrueba struct {
	preparacion             ports.PreparacionTransaccionCierreAdministrativo
	err                     error
	resultadoInvalido       bool
	actorResultadoRef       string
	correlacionResultadoRef string
	llamadas                int
	aplicaciones            int
	solicitud               ports.SolicitudTransaccionCierreAdministrativo
	solicitudConfirmada     ports.SolicitudTransaccionCierreAdministrativo
	siguiente               domain.Seguimiento
	confirmada              bool
}

func (t *transaccionCierreAdministrativoPrueba) EjecutarCierreAdministrativo(
	ctx context.Context,
	solicitud ports.SolicitudTransaccionCierreAdministrativo,
	aplicar ports.AplicarCierreAdministrativo,
) (ports.ResultadoCierreAdministrativo, error) {
	t.llamadas++
	t.solicitud = solicitud
	if t.err != nil {
		return ports.ResultadoCierreAdministrativo{}, t.err
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoCierreAdministrativo{}, err
	}
	if t.confirmada &&
		solicitud.ClaveIdempotencia == t.solicitudConfirmada.ClaveIdempotencia {
		if solicitud != t.solicitudConfirmada {
			return ports.ResultadoCierreAdministrativo{},
				ports.ErrClaveIdempotenciaCierreAdministrativoUsada
		}
		return ports.NuevoResultadoCierreAdministrativo(
			ports.DatosResultadoCierreAdministrativo{
				Solicitud:         t.solicitudConfirmada,
				VersionResultante: t.solicitudConfirmada.VersionEsperada + 1,
				ActuacionRef:      t.preparacion.ActuacionRef,
				ReciboRef:         t.preparacion.ReciboRef,
				ActorRef:          t.preparacion.ActorRef,
				CorrelacionRef:    t.preparacion.CorrelacionRef,
				Estado:            ports.EstadoResultadoCierreAdministrativoReplayConfirmado,
			},
		)
	}
	if t.preparacion.ValidarPara(solicitud) != nil {
		return ports.ResultadoCierreAdministrativo{},
			ports.ErrCierreAdministrativoDenegado
	}
	t.aplicaciones++
	siguiente, err := aplicar(t.preparacion)
	if err != nil {
		return ports.ResultadoCierreAdministrativo{}, err
	}
	t.siguiente = siguiente
	if err := ctx.Err(); err != nil {
		return ports.ResultadoCierreAdministrativo{}, err
	}
	t.confirmada = true
	t.solicitudConfirmada = solicitud
	if t.resultadoInvalido {
		return ports.ResultadoCierreAdministrativo{}, nil
	}
	actorRef := t.preparacion.ActorRef
	if t.actorResultadoRef != "" {
		actorRef = t.actorResultadoRef
	}
	correlacionRef := t.preparacion.CorrelacionRef
	if t.correlacionResultadoRef != "" {
		correlacionRef = t.correlacionResultadoRef
	}
	return ports.NuevoResultadoCierreAdministrativo(
		ports.DatosResultadoCierreAdministrativo{
			Solicitud: solicitud, VersionResultante: siguiente.Version(),
			ActuacionRef:   t.preparacion.ActuacionRef,
			ReciboRef:      t.preparacion.ReciboRef,
			ActorRef:       actorRef,
			CorrelacionRef: correlacionRef,
			Estado:         ports.EstadoResultadoCierreAdministrativoConfirmado,
		},
	)
}

func nuevoServicioCierreAdministrativoPrueba(
	t *testing.T,
	transaccion ports.TransaccionCierreAdministrativo,
) *ServicioCierreAdministrativo {
	t.Helper()
	servicio, err := NuevoServicioCierreAdministrativo(transaccion)
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	return servicio
}

func solicitudCerrarAdministrativamentePrueba(
	version uint64,
) SolicitudCerrarAdministrativamente {
	return SolicitudCerrarAdministrativamente{
		OrganizacionRef:   referenciaCierreAdministrativoPrueba("organizacion_publica_01"),
		ExpedienteRef:     referenciaCierreAdministrativoPrueba("expediente_temporal_01"),
		SeguimientoRef:    referenciaCierreAdministrativoPrueba("seguimiento_laboral_01"),
		VersionEsperada:   version,
		ClaveIdempotencia: "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
		TransicionClave:   "cerrar_administrativamente",
		MotivoClave:       "fin_expediente",
	}
}

func solicitudTransaccionalCierrePrueba(
	solicitud SolicitudCerrarAdministrativamente,
) ports.SolicitudTransaccionCierreAdministrativo {
	return ports.SolicitudTransaccionCierreAdministrativo{
		Operacion:         ports.OperacionCerrarAdministrativamente,
		OrganizacionRef:   solicitud.OrganizacionRef,
		ExpedienteRef:     solicitud.ExpedienteRef,
		SeguimientoRef:    solicitud.SeguimientoRef,
		VersionEsperada:   solicitud.VersionEsperada,
		ClaveIdempotencia: solicitud.ClaveIdempotencia,
		TransicionClave:   solicitud.TransicionClave,
		MotivoClave:       solicitud.MotivoClave,
	}
}

func preparacionCierreAdministrativoPrueba(
	t *testing.T,
	solicitud ports.SolicitudTransaccionCierreAdministrativo,
	definicion domain.DefinicionSeguimiento,
	seguimiento domain.Seguimiento,
	instante time.Time,
) ports.PreparacionTransaccionCierreAdministrativo {
	t.Helper()
	preparacion := ports.PreparacionTransaccionCierreAdministrativo{
		Solicitud: solicitud, Definicion: definicion, Seguimiento: seguimiento,
		Inventario: ports.InventarioTareasCierreAdministrativo{
			Referencia:      referenciaCierreAdministrativoPrueba("inventario_tareas_01"),
			OrganizacionRef: solicitud.OrganizacionRef,
			ExpedienteRef:   solicitud.ExpedienteRef,
			SeguimientoRef:  solicitud.SeguimientoRef, Version: 4,
			Total: 6, Pendientes: 0, Completo: true,
		},
		UnidadRef:    referenciaCierreAdministrativoPrueba("unidad_rrhh_01"),
		ActorRef:     referenciaCierreAdministrativoPrueba("actor_rrhh_confiable_01"),
		ActuacionRef: referenciaCierreAdministrativoPrueba("actuacion_cierre_01"),
		ReciboRef:    referenciaCierreAdministrativoPrueba("recibo_cierre_01"),
		CorrelacionRef: referenciaCierreAdministrativoPrueba(
			"correlacion_cierre_01",
		),
		EfectivoEn: instante, RegistradaEn: instante,
	}
	prepararAutorizacionCierreAdministrativoPrueba(t, &preparacion, solicitud, instante)
	if solicitud.Operacion != ports.OperacionCerrarAdministrativamenteSinCese {
		if err := preparacion.ValidarPara(solicitud); err != nil {
			t.Fatalf("preparación base de cierre inválida: %v", err)
		}
	}
	return preparacion
}

type generadorCorrelacionCierreAdministrativoPrueba struct {
	valor string
}

func (g generadorCorrelacionCierreAdministrativoPrueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return g.valor, nil
}

func prepararAutorizacionCierreAdministrativoPrueba(
	t *testing.T,
	preparacion *ports.PreparacionTransaccionCierreAdministrativo,
	solicitud ports.SolicitudTransaccionCierreAdministrativo,
	instante time.Time,
) {
	t.Helper()
	contexto := contextoAutorizacionAltaV3Prueba(t, instante)
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		t.Fatalf("extraer vínculo V3 de cierre: %v", err)
	}
	resumenMotivo := sha256.Sum256([]byte("motivo_autorizacion_cierre"))
	motivo := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion_cierre", CatalogoVersion: 1,
		CatalogoHuellaSHA256: hex.EncodeToString(resumenMotivo[:]),
		EntradaClave:         "motivo_" + hex.EncodeToString(resumenMotivo[:16]),
	}
	resumenCorrelacion := sha256.Sum256([]byte(
		"correlacion_autorizacion_" + string(solicitud.Operacion) + solicitud.ClaveIdempotencia,
	))
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		generadorCorrelacionCierreAdministrativoPrueba{
			valor: "correlacion_" + hex.EncodeToString(resumenCorrelacion[:16]),
		},
	)
	if err != nil {
		t.Fatalf("generar correlación V3 de cierre: %v", err)
	}
	correlacionV3Ref, err := correlacion.ValorCanonico()
	if err != nil {
		t.Fatalf("extraer correlación V3 de cierre: %v", err)
	}
	accion := ports.AccionAutorizacionCerrarAdministrativamente
	finalidad := ports.FinalidadAutorizacionCerrarAdministrativamente
	if solicitud.Operacion == ports.OperacionReabrirExcepcionalmente {
		accion = ports.AccionAutorizacionReabrirExcepcionalmente
		finalidad = ports.FinalidadAutorizacionReabrirExcepcionalmente
	}
	solicitudV3, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(
		dominiovec.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: contexto.Vinculo,
			ReferenciaMotivo:          motivo,
			Accion:                    accion,
			Recurso: dominiovec.RecursoAutorizable{
				Referencia: solicitud.SeguimientoRef,
				ModuloID:   ports.ModuloContratacion,
				Tipo:       ports.TipoRecursoCierreAdministrativo,
				Ambitos: map[string]string{
					"organizacion_ref": solicitud.OrganizacionRef,
					"expediente_ref":   solicitud.ExpedienteRef,
					"seguimiento_ref":  solicitud.SeguimientoRef,
				},
				Atributos: map[string]string{
					"operacion":                   string(solicitud.Operacion),
					"version_esperada":            strconv.FormatUint(solicitud.VersionEsperada, 10),
					"transicion_clave":            string(solicitud.TransicionClave),
					"motivo_clave":                string(solicitud.MotivoClave),
					"principal_v3_ref":            vinculo.PrincipalID,
					"actor_seguimiento_ref":       preparacion.ActorRef,
					"correlacion_v3_ref":          correlacionV3Ref,
					"correlacion_seguimiento_ref": preparacion.CorrelacionRef,
				},
			},
			Finalidad:   finalidad,
			Correlacion: correlacion,
		},
	)
	if err != nil {
		t.Fatalf("crear solicitud V3 de cierre: %v", err)
	}
	decision, confirmacion, err := concesionAutorizacionV3Prueba(
		t,
		solicitudV3,
		contexto.Resultado,
		motivo,
		instante,
		"decision:cierre-administrativo:prueba",
		true,
	)
	if err != nil {
		t.Fatalf("conceder autorización V3 de cierre: %v", err)
	}
	preparacion.ContextoAutorizacionV3 = contexto
	preparacion.SolicitudAutorizacionV3 = solicitudV3
	preparacion.DecisionAutorizacionV3 = decision
	preparacion.ConfirmacionAutorizacionV3 = confirmacion
	preparacion.MotivoAutorizacionV3 = motivo
	preparacion.PerfilRef = vinculo.PerfilActivoRef
}

func escenarioSeguimientoVigente(
	t *testing.T,
) (domain.DefinicionSeguimiento, domain.Seguimiento) {
	return escenarioSeguimientoConEfectoCierre(t, domain.EfectoPeriodoCerrar)
}

func escenarioSeguimientoConEfectoCierre(
	t *testing.T,
	efectoCierre domain.EfectoPeriodoSeguimiento,
) (domain.DefinicionSeguimiento, domain.Seguimiento) {
	t.Helper()
	definicion, err := domain.PublicarDefinicionSeguimiento(
		domain.BorradorDefinicionSeguimiento{
			Referencia:  referenciaCierreAdministrativoPrueba("definicion_cierre_administrativo_01"),
			Version:     1,
			PublicadoEn: instanteCierreAdministrativoPrueba.Add(-96 * time.Hour),
			Vigencia: domain.VigenciaSeguimiento{
				Desde: instanteCierreAdministrativoPrueba.Add(-72 * time.Hour),
				Hasta: instanteCierreAdministrativoPrueba.AddDate(1, 0, 0),
			},
			EstadoInicial: "pendiente_incorporacion", ProhibeCiclosSilenciosos: true,
			Estados: []domain.EstadoDefinidoSeguimiento{
				{Clave: "pendiente_incorporacion"},
				{Clave: "vigente"},
				{Clave: "cesada", Final: true},
			},
			Motivos: []domain.ClaveCatalogo{
				"incorporacion_confirmada", "fin_expediente",
				"subsanacion_excepcional", "incidencia_catalogada",
			},
			Transiciones: []domain.TransicionDefinidaSeguimiento{
				{
					Clave: "confirmar_incorporacion", Origen: "pendiente_incorporacion",
					Destino: "vigente", Clase: domain.TransicionOrdinaria,
					MotivosPermitidos: []domain.ClaveCatalogo{"incorporacion_confirmada"},
					MotivoObligatorio: true, RequierePeriodo: true,
					EfectoPeriodo: domain.EfectoPeriodoAbrir,
				},
				{
					Clave: "registrar_incidencia", Origen: "vigente", Destino: "vigente",
					Clase:             domain.TransicionOrdinaria,
					MotivosPermitidos: []domain.ClaveCatalogo{"incidencia_catalogada"},
					MotivoObligatorio: true, EfectoPeriodo: domain.EfectoPeriodoNinguno,
				},
				{
					Clave: "cerrar_administrativamente", Origen: "vigente", Destino: "cesada",
					Clase:             domain.TransicionOrdinaria,
					MotivosPermitidos: []domain.ClaveCatalogo{"fin_expediente"},
					MotivoObligatorio: true, EfectoPeriodo: efectoCierre,
				},
				{
					Clave: "reabrir_excepcionalmente", Origen: "cesada", Destino: "vigente",
					Clase:             domain.TransicionReapertura,
					MotivosPermitidos: []domain.ClaveCatalogo{"subsanacion_excepcional"},
					MotivoObligatorio: true, EfectoPeriodo: domain.EfectoPeriodoReabrir,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("publicar definición de cierre: %v", err)
	}
	periodo := domain.IntervaloSeguimiento{
		Desde: instanteCierreAdministrativoPrueba.Add(-24 * time.Hour),
		Hasta: instanteCierreAdministrativoPrueba.Add(30 * 24 * time.Hour),
	}
	seguimiento, err := domain.NuevoSeguimiento(
		definicion,
		domain.AltaSeguimiento{
			Referencia:      referenciaCierreAdministrativoPrueba("seguimiento_laboral_01"),
			OrganizacionRef: referenciaCierreAdministrativoPrueba("organizacion_publica_01"),
			ExpedienteRef:   referenciaCierreAdministrativoPrueba("expediente_temporal_01"),
			RelacionRef:     referenciaCierreAdministrativoPrueba("relacion_laboral_opaca_01"),
			PeriodoPrevisto: periodo,
			CreadoEn:        instanteCierreAdministrativoPrueba.Add(-48 * time.Hour),
		},
	)
	if err != nil {
		t.Fatalf("crear seguimiento: %v", err)
	}
	seguimiento, err = seguimiento.Aplicar(
		definicion,
		seguimiento.Version(),
		domain.DatosTransicionSeguimiento{
			ActuacionRef:    referenciaCierreAdministrativoPrueba("actuacion_incorporacion_01"),
			TransicionClave: "confirmar_incorporacion",
			MotivoClave:     "incorporacion_confirmada",
			ActorRef:        referenciaCierreAdministrativoPrueba("actor_personal_01"),
			UnidadRef:       referenciaCierreAdministrativoPrueba("unidad_personal_01"),
			EfectivoEn:      periodo.Desde, RegistradaEn: periodo.Desde,
			Periodo:        &periodo,
			ReciboRef:      referenciaCierreAdministrativoPrueba("recibo_incorporacion_01"),
			CorrelacionRef: referenciaCierreAdministrativoPrueba("correlacion_incorporacion_01"),
		},
	)
	if err != nil {
		t.Fatalf("confirmar incorporación: %v", err)
	}
	return definicion, seguimiento
}

func referenciaCierreAdministrativoPrueba(etiqueta string) string {
	resumen := sha256.Sum256([]byte(etiqueta))
	return "ref:" + hex.EncodeToString(resumen[:])
}
