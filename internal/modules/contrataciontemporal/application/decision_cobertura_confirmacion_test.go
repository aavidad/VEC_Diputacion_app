package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestDecidirCoberturaLigaDenegacionVECSinEfectoC1C2(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, false)
	ctx, cancelar := context.WithCancel(context.Background())
	escenario.transaccion.cancelar = cancelar
	recibo, err := escenario.servicio.Decidir(ctx, escenario.solicitud)
	if err != nil {
		t.Fatalf("el recibo durable no dominó la cancelación tardía: %v", err)
	}
	if _, denegada := recibo.ResultadoDenegadoVEC(); !denegada ||
		recibo.ConcedidaVEC ||
		escenario.vec.total() != 1 ||
		escenario.transaccion.total() != 1 ||
		escenario.reconciliador.total() != 0 {
		t.Fatalf("terminal VEC negativo inválido: %+v", recibo)
	}
	exigirCeroConsumoPreparacionGlobal(t, escenario.base.global)
}

func TestDecidirCoberturaErrorTrasCallbackExigeReconciliacion(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, false)
	escenario.transaccion.errorRetorno = errors.New(
		"respuesta HTTP perdida después del COMMIT",
	)
	_, err := escenario.servicio.Decidir(
		context.Background(),
		escenario.solicitud,
	)
	if !errors.Is(err, ErrConfirmacionDecisionCoberturaNoDisponible) ||
		escenario.transaccion.total() != 1 ||
		escenario.reconciliador.total() != 1 {
		t.Fatalf("el error tardío no condujo al primario: %v", err)
	}
}

func TestDecidirCoberturaConcesionAmbiguaReconciliaUnaVezSinRetry(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, true)
	escenario.transaccion.ambigua = true
	_, err := escenario.servicio.Decidir(
		context.Background(),
		escenario.solicitud,
	)
	if !errors.Is(err, ErrConfirmacionDecisionCoberturaNoDisponible) {
		t.Fatalf("la ambigüedad se convirtió en éxito o rollback: %v", err)
	}
	if escenario.vec.total() != 1 ||
		escenario.transaccion.total() != 1 ||
		escenario.reconciliador.total() != 1 {
		t.Fatal("se repitió Tx/VEC o no se consultó el primario una vez")
	}
	exigirCeroConsumoPreparacionGlobal(t, escenario.base.global)
}

func TestDecidirCoberturaConcesionLigaReciboAplicadoExactoSinPersistencia(
	t *testing.T,
) {
	// Esta frontera es nominal: acredita Recibo↔Orden, no COMMIT O4-04.
	escenario := nuevoEscenarioConfirmacionCobertura(t, true)
	recibo, err := escenario.servicio.Decidir(
		context.Background(),
		escenario.solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	aplicada, existe := recibo.ResultadoAplicado()
	if !existe ||
		aplicada.VersionResultante != escenario.base.expediente.Version+1 ||
		aplicada.EventoRef != "evento_confirmacion_cobertura_012345" ||
		aplicada.ActuacionRef !=
			"actuacion_confirmacion_cobertura_012345" ||
		recibo.CorrelacionVECRef !=
			"correlacion_11111111111111111111111111111111" ||
		recibo.DecisionVECRef !=
			"dec_11111111111111111111111111111111" ||
		escenario.vec.total() != 1 ||
		escenario.transaccion.total() != 1 ||
		escenario.reconciliador.total() != 0 {
		t.Fatalf("confirmación concedida no ligada: %+v", recibo)
	}
	exigirCeroConsumoPreparacionGlobal(t, escenario.base.global)
}

func TestDecidirCoberturaReconciliacionConfirmadaDominaCancelacionTardia(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, true)
	ctx, cancelar := context.WithCancel(context.Background())
	escenario.transaccion.ambigua = true
	escenario.reconciliador.confirmar = true
	escenario.reconciliador.cancelar = cancelar
	recibo, err := escenario.servicio.Decidir(ctx, escenario.solicitud)
	if err != nil {
		t.Fatalf("el recibo primario no dominó la cancelación tardía: %v", err)
	}
	if _, aplicada := recibo.ResultadoAplicado(); !aplicada ||
		escenario.transaccion.total() != 1 ||
		escenario.reconciliador.total() != 1 ||
		escenario.vec.total() != 1 {
		t.Fatalf("reconciliación confirmada inválida: %+v", recibo)
	}
}

func TestDecidirCoberturaReplayCortaGobiernoC1VECYTransaccion(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, false)
	recibo, err := escenario.servicio.Decidir(
		context.Background(),
		escenario.solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	escenario.idempotencia.instalarReplay(t, recibo)
	analisisAntes := escenario.base.analisis.total()
	gobiernoAntes := escenario.base.gobierno.total()
	referenciasAntes := escenario.base.global.generador.llamadas()
	vecAntes := escenario.vec.total()
	txAntes := escenario.transaccion.total()
	_, reservasAntes := escenario.idempotencia.totales()

	repetido, err := escenario.servicio.Decidir(
		context.Background(),
		escenario.solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, reservasDespues := escenario.idempotencia.totales()
	if _, denegada := repetido.ResultadoDenegadoVEC(); !denegada ||
		escenario.base.analisis.total() != analisisAntes ||
		escenario.base.gobierno.total() != gobiernoAntes ||
		escenario.base.global.generador.llamadas() != referenciasAntes ||
		escenario.vec.total() != vecAntes ||
		escenario.transaccion.total() != txAntes ||
		reservasDespues != reservasAntes {
		t.Fatal("el replay volvió a ejecutar gobierno, C1, VEC, reserva o Tx")
	}
}

func TestDecidirCoberturaOcupadaNoAlcanzaAutoridadesPosteriores(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, false)
	escenario.idempotencia.modo = idempotenciaOcupada
	analisisAntes := escenario.base.analisis.total()
	gobiernoAntes := escenario.base.gobierno.total()
	_, err := escenario.servicio.Decidir(
		context.Background(),
		escenario.solicitud,
	)
	if !errors.Is(err, ErrConfirmacionDecisionCoberturaOcupada) ||
		escenario.base.analisis.total() != analisisAntes ||
		escenario.base.gobierno.total() != gobiernoAntes ||
		escenario.vec.total() != 0 ||
		escenario.transaccion.total() != 0 {
		t.Fatalf("la propiedad ajena no cerró el flujo: %v", err)
	}
}

func TestDecidirCoberturaDenegacionContextoEsTerminalAntesDeSellar(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, false)
	escenario.base.contextos.err = ports.ErrAutorizacionDenegada
	_, err := escenario.servicio.Decidir(
		context.Background(),
		escenario.solicitud,
	)
	consultas, reservas := escenario.idempotencia.totales()
	if !errors.Is(err, ErrConfirmacionDecisionCoberturaDenegada) ||
		escenario.sellador.total() != 0 ||
		consultas != 0 ||
		reservas != 0 ||
		escenario.vec.total() != 0 ||
		escenario.transaccion.total() != 0 {
		t.Fatalf("la denegación de contexto no cerró el flujo: %v", err)
	}
}

func TestDecidirCoberturaContextoExpiradoJustoAntesDeVEC(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, false)
	reloj := &relojRevocacionConfirmacionPrueba{
		delegado:         escenario.base.reloj,
		reloj:            escenario.base.global.entorno.reloj,
		revocarEnLlamada: 7,
		avanceRevocacion: 2 * time.Hour,
	}
	escenario.servicio.reloj = reloj
	_, err := escenario.servicio.Decidir(
		context.Background(),
		escenario.solicitud,
	)
	if !errors.Is(err, ErrConfirmacionDecisionCoberturaDenegada) ||
		reloj.total() != 7 ||
		escenario.vec.total() != 0 ||
		escenario.transaccion.total() != 0 {
		t.Fatalf("el contexto expirado alcanzó VEC/Tx: %v", err)
	}
}

func TestDecidirCoberturaCanceladaTrasReservaNoAlcanzaGobiernoNiVEC(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, false)
	ctx, cancelar := context.WithCancel(context.Background())
	escenario.idempotencia.cancelarReserva = cancelar
	gobiernoAntes := escenario.base.gobierno.total()
	_, err := escenario.servicio.Decidir(ctx, escenario.solicitud)
	if !errors.Is(err, context.Canceled) ||
		escenario.base.gobierno.total() != gobiernoAntes ||
		escenario.vec.total() != 0 ||
		escenario.transaccion.total() != 0 {
		t.Fatalf("la cancelación tras reserva alcanzó autoridades: %v", err)
	}
}

func TestDecidirCoberturaRechazaSellosInvalidosYClasificaDependencia(
	t *testing.T,
) {
	t.Run("salida inválida", func(t *testing.T) {
		escenario := nuevoEscenarioConfirmacionCobertura(t, false)
		escenario.sellador.invalido = true
		_, err := escenario.servicio.Decidir(
			context.Background(),
			escenario.solicitud,
		)
		consultas, reservas := escenario.idempotencia.totales()
		if !errors.Is(
			err,
			ErrConfirmacionDecisionCoberturaNoConfiable,
		) || consultas != 0 || reservas != 0 {
			t.Fatalf("sellos inválidos alcanzaron idempotencia: %v", err)
		}
	})
	t.Run("dependencia", func(t *testing.T) {
		escenario := nuevoEscenarioConfirmacionCobertura(t, false)
		escenario.sellador.err = errors.New("HSM no disponible")
		_, err := escenario.servicio.Decidir(
			context.Background(),
			escenario.solicitud,
		)
		if !errors.Is(
			err,
			ErrConfirmacionDecisionCoberturaNoDisponible,
		) {
			t.Fatalf("el fallo del HSM se clasificó mal: %v", err)
		}
	})
}

func TestDecidirCoberturaRechazaPropuestaCambiadaAntesDeVEC(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, false)
	otraHuella := strings.Repeat("9", 64)
	escenario.solicitud.IdentidadSemantica.HuellaSHA256 = otraHuella
	escenario.solicitud.IdentidadSemantica.Referencia =
		"propuesta-cobertura-semantica:sha256:" + otraHuella
	_, err := escenario.servicio.Decidir(
		context.Background(),
		escenario.solicitud,
	)
	if !errors.Is(err, ErrConfirmacionDecisionCoberturaEnConflicto) ||
		escenario.vec.total() != 0 ||
		escenario.transaccion.total() != 0 {
		t.Fatalf("la propuesta obsoleta alcanzó VEC/Tx: %v", err)
	}
	exigirCeroConsumoPreparacionGlobal(t, escenario.base.global)
}

func TestConfirmacionCoberturaValidaEntradasAntesDeAutoridades(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, false)
	casos := map[string]func(*SolicitudDecidirCobertura){
		"autenticación": func(s *SolicitudDecidirCobertura) {
			s.AutenticacionRef = ""
		},
		"sesión": func(s *SolicitudDecidirCobertura) { s.SesionRef = "" },
		"perfil": func(s *SolicitudDecidirCobertura) { s.PerfilRef = "" },
		"organización": func(s *SolicitudDecidirCobertura) {
			s.OrganizacionRef = ""
		},
		"expediente": func(s *SolicitudDecidirCobertura) {
			s.ExpedienteRef = ""
		},
		"idempotencia": func(s *SolicitudDecidirCobertura) {
			s.ClaveIdempotencia = ""
		},
		"semántica": func(s *SolicitudDecidirCobertura) {
			s.IdentidadSemantica = domain.IdentidadSemanticaPropuestaDecisionCobertura{}
		},
		"vía": func(s *SolicitudDecidirCobertura) { s.ViaElegida = "" },
		"versión 2^53-1": func(s *SolicitudDecidirCobertura) {
			s.VersionEsperada =
				cobertura.MaximoEnteroSeguroOperacionDecisionCobertura
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			antes := escenario.base.contextos.total()
			solicitud := escenario.solicitud
			alterar(&solicitud)
			_, err := escenario.servicio.Decidir(
				context.Background(),
				solicitud,
			)
			if !errors.Is(
				err,
				ErrSolicitudConfirmacionDecisionCoberturaInvalida,
			) || escenario.base.contextos.total() != antes {
				t.Fatalf("la entrada inválida alcanzó autoridades: %v", err)
			}
		})
	}
}

func TestRectificarCoberturaExigeMotivoYPredecesoraDesdeElCanalMinimo(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, false)
	base := SolicitudRectificarCobertura{
		AutenticacionRef:   escenario.solicitud.AutenticacionRef,
		SesionRef:          escenario.solicitud.SesionRef,
		PerfilRef:          escenario.solicitud.PerfilRef,
		OrganizacionRef:    escenario.solicitud.OrganizacionRef,
		ExpedienteRef:      escenario.solicitud.ExpedienteRef,
		VersionEsperada:    escenario.solicitud.VersionEsperada,
		ClaveIdempotencia:  escenario.solicitud.ClaveIdempotencia,
		IdentidadSemantica: escenario.solicitud.IdentidadSemantica,
		ViaElegida:         escenario.solicitud.ViaElegida,
	}
	for nombre, completar := range map[string]func(*SolicitudRectificarCobertura){
		"sin motivo": func(s *SolicitudRectificarCobertura) {
			s.PredecesoraRef = "decision_cobertura_predecesora_012345"
			s.PredecesoraHuella = strings.Repeat("a", 64)
		},
		"sin predecesora": func(s *SolicitudRectificarCobertura) {
			s.MotivoClave = "rectificacion_decision"
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			solicitud := base
			completar(&solicitud)
			_, err := escenario.servicio.Rectificar(
				context.Background(),
				solicitud,
			)
			if !errors.Is(
				err,
				ErrSolicitudConfirmacionDecisionCoberturaInvalida,
			) {
				t.Fatalf("rectificación incompleta aceptada: %v", err)
			}
		})
	}
}

func TestSolicitudesConfirmacionCoberturaSonMinimasYRedactadas(
	t *testing.T,
) {
	exigirCamposSolicitudConfirmacion(t, reflect.TypeOf(SolicitudDecidirCobertura{}), 10)
	exigirCamposSolicitudConfirmacion(t, reflect.TypeOf(SolicitudRectificarCobertura{}), 12)
	decidir := SolicitudDecidirCobertura{AutenticacionRef: "secreto_autenticacion"}
	rectificar := SolicitudRectificarCobertura{SesionRef: "secreto_sesion"}
	if texto := fmt.Sprintf("%v|%+v|%#v", decidir, decidir, decidir); texto != redaccionSolicitudConfirmacionCobertura+"|"+
		redaccionSolicitudConfirmacionCobertura+"|"+
		redaccionSolicitudConfirmacionCobertura {
		t.Fatalf("decidir no se redactó: %s", texto)
	}
	if texto := fmt.Sprintf("%v", rectificar); texto != redaccionSolicitudConfirmacionCobertura {
		t.Fatalf("rectificar no se redactó: %s", texto)
	}
	if valor := decidir.LogValue(); valor.Kind() != slog.KindString ||
		valor.String() != redaccionSolicitudConfirmacionCobertura {
		t.Fatal("LogValue expuso material de canal")
	}
}

func TestServicioConfirmacionCoberturaRechazaNilTipadoYCancelacion(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, false)
	var sellador *selladorConfirmacionPrueba
	if _, err := NuevoServicioConfirmacionDecisionCobertura(
		escenario.base.contextos, escenario.motivos, sellador,
		escenario.idempotencia, escenario.base.analisis,
		escenario.base.reloj, escenario.base.gobierno,
		escenario.base.global.preparador, escenario.vec,
		escenario.frontera, escenario.reconciliador,
	); !errors.Is(err, ErrServicioConfirmacionDecisionCoberturaInvalido) {
		t.Fatalf("se aceptó nil tipado: %v", err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := escenario.servicio.Decidir(
		ctx,
		escenario.solicitud,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("se ignoró contexto ya cancelado: %v", err)
	}
}

func exigirCamposSolicitudConfirmacion(
	t *testing.T,
	tipo reflect.Type,
	total int,
) {
	t.Helper()
	if tipo.NumField() != total {
		t.Fatalf("%s amplió superficie: %d", tipo.Name(), tipo.NumField())
	}
	for indice := 0; indice < tipo.NumField(); indice++ {
		nombre := strings.ToLower(tipo.Field(indice).Name)
		for _, prohibido := range []string{
			"actor", "rol", "accion", "reloj", "catalogo", "politica",
			"finalidad", "unidad", "vec", "evidencia", "cookie",
		} {
			if strings.Contains(nombre, prohibido) {
				t.Fatalf("%s admite autoridad libre: %s", tipo.Name(), nombre)
			}
		}
	}
}
