package application

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestRectificacionRevalidaVigenciaDelMotivoAlValidarRecibo(t *testing.T) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(t, ports.OperacionRectificarAnalisis, "-vigencia-recibo-sintetica")
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	var hasta time.Time
	d.politicas.transformar = func(p *ports.PoliticaOperacionAnalisis) {
		hasta = p.EvaluadaEn.Add(time.Second)
		p.MotivoRectificacion.VigenteHasta = hasta
	}
	recibo, err := servicio.Rectificar(context.Background(), escenario.rectificar)
	if err != nil {
		t.Fatalf("rectificación vigente: %v", err)
	}
	for _, caso := range []struct {
		nombre   string
		instante time.Time
		valido   bool
	}{
		{"antes_del_limite", hasta.Add(-time.Microsecond), true},
		{"en_el_limite", hasta, false},
		{"despues_del_limite", hasta.Add(time.Microsecond), false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			candidato := recibo
			candidato.ConfirmadaEn = caso.instante
			err := candidato.ValidarParaOrdenDentroDeTransaccion(d.transaccion.orden)
			if (err == nil) != caso.valido {
				t.Fatalf("validación previa al commit: %v, válido esperado %v", err, caso.valido)
			}
		})
	}
}

func TestOperacionAnalisisRegistraDesdeArtefactoInterno(t *testing.T) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-registro-sintetico",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	recibo, err := servicio.Registrar(context.Background(), escenario.registrar)
	if err != nil {
		t.Fatalf("registrar análisis: %v", err)
	}
	if recibo.VersionAnterior != 1 || recibo.VersionResultante != 2 ||
		recibo.ArtefactoRef != escenario.registrar.ArtefactoRef ||
		d.artefactos.llamadas != 1 || d.politicas.llamadas != 1 ||
		d.autorizador.llamadas != 1 || d.transaccion.llamadas != 1 ||
		d.transaccion.consumosFuentes != 1 ||
		d.transaccion.consumosV3 != 1 ||
		d.transaccion.commits != 1 ||
		recibo.ValidarParaOrdenDentroDeTransaccion(
			d.transaccion.orden,
		) != nil {
		t.Fatalf("resultado o secuencia incorrectos: %#v", recibo)
	}
	evidencia, err := d.transaccion.orden.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if evidencia.ExpedienteSiguiente.Analisis == nil ||
		evidencia.ExpedienteSiguiente.Analisis.ValidacionRC.Resultado !=
			domain.RCValidada ||
		evidencia.ExpedienteSiguiente.Analisis.CostePrevisto == nil ||
		evidencia.ExpedienteSiguiente.Analisis.FuenteCosteRef !=
			"fuente_coste_sintetica_012345" {
		t.Fatalf("el análisis no fue derivado del artefacto: %#v",
			evidencia.ExpedienteSiguiente.Analisis)
	}
	pruebas, err := evidencia.Artefacto.PruebasParaO3(
		evidencia.SolicitudArtefacto,
	)
	if err != nil {
		t.Fatal(err)
	}
	ordenArtefacto, err := pruebas.OrdenConsumoConjunto.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ordenFinal, err := evidencia.OrdenConsumoFuentes.Datos()
	if err != nil || !reflect.DeepEqual(ordenArtefacto, ordenFinal) {
		t.Fatal("la orden final no transportó el consumo pendiente exacto")
	}
	datosV3, err := d.autorizador.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	huellaAnalisis, err := ports.HuellaAnalisisDerivadoO3(
		evidencia.SolicitudArtefacto,
		evidencia.Artefacto,
	)
	if err != nil {
		t.Fatal(err)
	}
	if datosV3.Recurso.Atributos[ports.AtributoArtefactoAnalisisRef] !=
		recibo.ArtefactoRef ||
		datosV3.Recurso.Atributos[ports.AtributoArtefactoAnalisisHuella] !=
			recibo.ArtefactoHuellaSHA256 ||
		datosV3.Recurso.Atributos[ports.AtributoAnalisisDerivadoHuella] !=
			huellaAnalisis ||
		len(datosV3.Recurso.Ambitos) != 4 ||
		len(datosV3.Recurso.Atributos) != 13 {
		t.Fatalf("recurso VEC V3 incompleto: %#v", datosV3.Recurso)
	}
}

func TestOperacionAnalisisRectificaConSegregacionYMotivoGobernado(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRectificarAnalisis,
		"-rectificacion-sintetica",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	recibo, err := servicio.Rectificar(
		context.Background(),
		escenario.rectificar,
	)
	if err != nil {
		t.Fatalf("rectificar análisis: %v", err)
	}
	evidencia, err := d.transaccion.orden.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ultima := evidencia.ExpedienteSiguiente.Actuaciones[len(evidencia.ExpedienteSiguiente.Actuaciones)-1]
	if recibo.VersionAnterior != 2 || recibo.VersionResultante != 3 ||
		ultima.Observaciones != string(escenario.motivoRectificacion) ||
		!evidencia.Politica.ExigeActorDistinto ||
		evidencia.Politica.ActorRef ==
			evidencia.Politica.ActorAnalisisAnteriorRef ||
		d.transaccion.consumosFuentes != 1 ||
		d.transaccion.consumosV3 != 1 ||
		d.transaccion.commits != 1 {
		t.Fatalf("rectificación o segregación incorrecta: %#v", evidencia)
	}
}

func TestOperacionAnalisisDTOExternoNoAceptaCamposAutoritativos(
	t *testing.T,
) {
	tipos := []reflect.Type{
		reflect.TypeOf(SolicitudRegistrarAnalisis{}),
		reflect.TypeOf(SolicitudRectificarAnalisis{}),
	}
	prohibidos := []string{
		"analisis", "validacionrc", "costeprevisto", "fuentecoste",
		"recibocoste", "actorref", "accion", "unidadref",
		"raiz", "confianza", "credencial", "confirmacion", "recibo",
	}
	for _, tipo := range tipos {
		for indice := 0; indice < tipo.NumField(); indice++ {
			nombre := strings.ToLower(tipo.Field(indice).Name)
			for _, prohibido := range prohibidos {
				if strings.Contains(nombre, prohibido) {
					t.Fatalf("%s expone campo autoritativo %s",
						tipo.Name(), tipo.Field(indice).Name)
				}
			}
		}
	}
}

func TestOperacionAnalisisReintentoConfirmadoNoRepiteEfectos(t *testing.T) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-reintento-sintetico",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	primero, err := servicio.Registrar(context.Background(), escenario.registrar)
	if err != nil {
		t.Fatal(err)
	}
	d.preparaciones.consultaConfirmada = &primero
	d.artefactos.err = errors.New("fuentes-sinteticas-caidas")
	llamadasArtefacto := d.artefactos.llamadas
	consumosFuentes := d.transaccion.consumosFuentes
	consumosV3 := d.transaccion.consumosV3
	preparaciones := d.preparaciones.llamadas
	segundo, err := servicio.Registrar(context.Background(), escenario.registrar)
	if err != nil {
		t.Fatal(err)
	}
	if segundo != primero ||
		d.artefactos.llamadas != llamadasArtefacto ||
		d.transaccion.consumosFuentes != consumosFuentes ||
		d.transaccion.consumosV3 != consumosV3 ||
		d.preparaciones.llamadas != preparaciones ||
		d.preparaciones.consultas != 2 ||
		d.politicas.llamadas != 1 || d.autorizador.llamadas != 1 ||
		d.transaccion.llamadas != 1 {
		t.Fatalf("el reintento produjo efectos nuevos: %#v", segundo)
	}
}

func TestOperacionAnalisisMismaClaveConSemanticaDistintaEsConflicto(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-conflicto-sintetico",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	if _, err := servicio.Registrar(
		context.Background(),
		escenario.registrar,
	); err != nil {
		t.Fatal(err)
	}
	d.preparaciones.errConsulta =
		ports.ErrClaveIdempotenciaOperacionAnalisisUsada
	cambiada := escenario.registrar
	cambiada.DatosFuncionales.PorcentajeJornada = 5_000

	_, err := servicio.Registrar(context.Background(), cambiada)
	if !errors.Is(err, ErrOperacionAnalisisEnConflicto) ||
		len(d.sellador.preimagenes) != 3 ||
		d.artefactos.llamadas != 1 ||
		d.transaccion.consumosFuentes != 1 ||
		d.transaccion.consumosV3 != 1 ||
		d.transaccion.commits != 1 {
		t.Fatalf("se esperaba conflicto semántico, recibido: %v", err)
	}
}

func TestOperacionAnalisisClasificaConflictoDeConsumoConjunto(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-conflicto-consumo-conjunto-sintetico",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	d.transaccion.err =
		ports.ErrConjuntoFuentesAnalisisYaConsumido

	_, err := servicio.Registrar(context.Background(), escenario.registrar)
	if !errors.Is(err, ErrOperacionAnalisisEnConflicto) ||
		!errors.Is(
			err,
			ports.ErrConjuntoFuentesAnalisisYaConsumido,
		) ||
		d.transaccion.consumosFuentes != 0 ||
		d.transaccion.consumosV3 != 0 ||
		d.transaccion.commits != 0 ||
		d.transaccion.llamadas != 1 {
		t.Fatalf("conflicto conjunto mal clasificado: %v", err)
	}
}

func TestOperacionAnalisisPropagaConflictoCASDurable(t *testing.T) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-cas-sintetico",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	d.transaccion.err = domain.ErrVersionEnConflicto
	_, err := servicio.Registrar(context.Background(), escenario.registrar)
	if !errors.Is(err, ErrOperacionAnalisisEnConflicto) ||
		!errors.Is(err, domain.ErrVersionEnConflicto) ||
		d.transaccion.llamadas != 1 ||
		d.transaccion.consumosFuentes != 0 ||
		d.transaccion.consumosV3 != 0 ||
		d.transaccion.commits != 0 {
		t.Fatalf("conflicto CAS mal clasificado: %v", err)
	}
}

func TestOperacionAnalisisFalloDePersistenciaNoDejaConsumos(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-persistencia-caida-sintetica",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	d.transaccion.err =
		ports.ErrPersistenciaOperacionAnalisisNoDisponible

	_, err := servicio.Registrar(context.Background(), escenario.registrar)
	if !errors.Is(err, ErrDependenciaOperacionAnalisisNoDisponible) ||
		d.transaccion.llamadas != 1 ||
		d.transaccion.consumosFuentes != 0 ||
		d.transaccion.consumosV3 != 0 ||
		d.transaccion.commits != 0 {
		t.Fatalf("el rollback de persistencia dejó efectos: %v", err)
	}
}

func TestOperacionAnalisisReciboValidoPruebaCommitPeseAErrorCompetitivo(
	t *testing.T,
) {
	casos := []struct {
		nombre string
		err    error
	}{
		{"cancelacion", context.Canceled},
		{
			"transporte",
			ports.ErrPersistenciaOperacionAnalisisNoDisponible,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioOperacionAnalisisSaneado(
				t,
				ports.OperacionRegistrarAnalisis,
				"-recibo-valido-"+caso.nombre+"-sintetico",
			)
			servicio, d := construirServicioOperacionAnalisisSaneado(
				t,
				escenario,
			)
			ctx := context.Background()
			if errors.Is(caso.err, context.Canceled) {
				cancelable, cancelar := context.WithCancel(ctx)
				ctx = cancelable
				d.transaccion.despues = cancelar
			}
			d.transaccion.errTrasCommit = caso.err

			recibo, err := servicio.Registrar(ctx, escenario.registrar)
			if err != nil ||
				recibo.ValidarParaOrdenDentroDeTransaccion(
					d.transaccion.orden,
				) != nil ||
				d.transaccion.consumosFuentes != 1 ||
				d.transaccion.consumosV3 != 1 ||
				d.transaccion.commits != 1 {
				t.Fatalf("un recibo válido no probó el commit: %#v, %v",
					recibo, err)
			}
		})
	}
}

func TestOperacionAnalisisNoExponeReciboAdulteradoYReplayRecuperaConfirmado(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-recibo-adulterado-sintetico",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	d.transaccion.adulterarSalida = true

	reciboAdulterado, err := servicio.Registrar(
		context.Background(),
		escenario.registrar,
	)
	if !errors.Is(err, ErrResultadoOperacionAnalisisNoConfiable) ||
		reciboAdulterado != (ports.ReciboOperacionAnalisis{}) ||
		d.transaccion.confirmado == nil ||
		d.transaccion.consumosFuentes != 1 ||
		d.transaccion.consumosV3 != 1 ||
		d.transaccion.commits != 1 {
		t.Fatalf("el adaptador defectuoso expuso un resultado: %v", err)
	}

	d.preparaciones.consultaConfirmada = d.transaccion.confirmado
	d.transaccion.adulterarSalida = false
	recuperado, err := servicio.Registrar(
		context.Background(),
		escenario.registrar,
	)
	if err != nil ||
		recuperado != *d.transaccion.confirmado ||
		d.transaccion.llamadas != 1 ||
		d.transaccion.consumosFuentes != 1 ||
		d.transaccion.consumosV3 != 1 ||
		d.transaccion.commits != 1 {
		t.Fatalf("el replay no recuperó el recibo durable: %#v, %v",
			recuperado, err)
	}
}

func TestOperacionAnalisisReciboAdulteradoNoPrevaleceSobreErrorCompetitivo(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-recibo-adulterado-error-sintetico",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	ctx, cancelar := context.WithCancel(context.Background())
	d.transaccion.despues = cancelar
	d.transaccion.adulterarSalida = true
	d.transaccion.errTrasCommit = context.Canceled

	recibo, err := servicio.Registrar(ctx, escenario.registrar)
	if recibo != (ports.ReciboOperacionAnalisis{}) ||
		!errors.Is(err, ErrDependenciaOperacionAnalisisNoDisponible) ||
		!errors.Is(err, context.Canceled) ||
		d.transaccion.confirmado == nil ||
		d.transaccion.commits != 1 {
		t.Fatalf("un recibo adulterado prevaleció: %#v, %v", recibo, err)
	}
}

func TestOperacionAnalisisNoConsumeFuentesSiNoPuedeReservarIntencion(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-reserva-caida-sintetica",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	d.preparaciones.err = errors.New("reserva-sintetica-no-disponible")

	_, err := servicio.Registrar(context.Background(), escenario.registrar)
	if !errors.Is(err, ErrDependenciaOperacionAnalisisNoDisponible) ||
		d.transaccion.consumosFuentes != 0 ||
		d.politicas.llamadas != 0 ||
		d.autorizador.llamadas != 0 ||
		d.transaccion.llamadas != 0 {
		t.Fatalf("se consumieron fuentes sin intención reservada: %v", err)
	}
}

func TestOperacionAnalisisDistingueDenegacionYDependencia(t *testing.T) {
	t.Run("denegacion", func(t *testing.T) {
		escenario := nuevoEscenarioOperacionAnalisisSaneado(
			t,
			ports.OperacionRegistrarAnalisis,
			"-denegacion-sintetica",
		)
		servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
		d.autorizador.decisionDenegada = true
		_, err := servicio.Registrar(context.Background(), escenario.registrar)
		if !errors.Is(err, ErrOperacionAnalisisDenegada) ||
			errors.Is(err, ErrDependenciaOperacionAnalisisNoDisponible) ||
			d.transaccion.consumosFuentes != 0 ||
			d.transaccion.consumosV3 != 0 {
			t.Fatalf("clasificación incorrecta: %v", err)
		}
	})
	t.Run("dependencia_sin_filtrar_causa", func(t *testing.T) {
		escenario := nuevoEscenarioOperacionAnalisisSaneado(
			t,
			ports.OperacionRegistrarAnalisis,
			"-dependencia-sintetica",
		)
		servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
		marcador := "causa-privada-sintetica-no-divulgar"
		d.autorizador.err = errors.New(marcador)
		_, err := servicio.Registrar(context.Background(), escenario.registrar)
		if !errors.Is(err, ErrDependenciaOperacionAnalisisNoDisponible) ||
			errors.Is(err, ErrOperacionAnalisisDenegada) ||
			strings.Contains(err.Error(), marcador) ||
			strings.Contains(slog.AnyValue(err).String(), marcador) {
			t.Fatalf("clasificación o redacción incorrecta: %v", err)
		}
	})
	t.Run("contexto_denegado", func(t *testing.T) {
		escenario := nuevoEscenarioOperacionAnalisisSaneado(
			t,
			ports.OperacionRegistrarAnalisis,
			"-contexto-denegado-sintetico",
		)
		servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
		d.contextos.err = ports.ErrAutorizacionDenegada
		_, err := servicio.Registrar(context.Background(), escenario.registrar)
		if !errors.Is(err, ErrOperacionAnalisisDenegada) ||
			d.artefactos.llamadas != 0 {
			t.Fatalf("denegación de contexto mal clasificada: %v", err)
		}
	})
	t.Run("contexto_no_disponible", func(t *testing.T) {
		escenario := nuevoEscenarioOperacionAnalisisSaneado(
			t,
			ports.OperacionRegistrarAnalisis,
			"-contexto-caido-sintetico",
		)
		servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
		d.contextos.err = errors.New("causa-interna-sintetica")
		_, err := servicio.Registrar(context.Background(), escenario.registrar)
		if !errors.Is(err, ErrDependenciaOperacionAnalisisNoDisponible) ||
			errors.Is(err, ErrOperacionAnalisisDenegada) {
			t.Fatalf("fallo de contexto mal clasificado: %v", err)
		}
	})
}

func TestOperacionAnalisisRechazaResultadosNoConfiables(t *testing.T) {
	t.Run("artefacto_opaco_cero", func(t *testing.T) {
		escenario := nuevoEscenarioOperacionAnalisisSaneado(
			t,
			ports.OperacionRegistrarAnalisis,
			"-artefacto-cero-sintetico",
		)
		servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
		cero := ports.ArtefactoAnalisisPreparado{}
		d.artefactos.forzado = &cero
		_, err := servicio.Registrar(context.Background(), escenario.registrar)
		if !errors.Is(err, ErrResultadoOperacionAnalisisNoConfiable) ||
			d.preparaciones.llamadas != 0 {
			t.Fatalf("artefacto no confiable aceptado: %v", err)
		}
	})
	t.Run("politica_alterada", func(t *testing.T) {
		escenario := nuevoEscenarioOperacionAnalisisSaneado(
			t,
			ports.OperacionRegistrarAnalisis,
			"-politica-alterada-sintetica",
		)
		servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
		d.politicas.transformar = func(
			p *ports.PoliticaOperacionAnalisis,
		) {
			p.ArtefactoHuellaSHA256 = strings.Repeat("9", 64)
		}
		_, err := servicio.Registrar(context.Background(), escenario.registrar)
		if !errors.Is(err, ErrResultadoOperacionAnalisisNoConfiable) ||
			d.transaccion.consumosFuentes != 0 ||
			d.autorizador.llamadas != 0 || d.transaccion.llamadas != 0 {
			t.Fatalf("política no confiable aceptada: %v", err)
		}
	})
	t.Run("rectificacion_sin_segregacion", func(t *testing.T) {
		escenario := nuevoEscenarioOperacionAnalisisSaneado(
			t,
			ports.OperacionRectificarAnalisis,
			"-segregacion-obligatoria-sintetica",
		)
		servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
		d.politicas.transformar = func(
			p *ports.PoliticaOperacionAnalisis,
		) {
			p.ExigeActorDistinto = false
		}
		_, err := servicio.Rectificar(
			context.Background(),
			escenario.rectificar,
		)
		if !errors.Is(err, ErrResultadoOperacionAnalisisNoConfiable) ||
			d.transaccion.consumosFuentes != 0 ||
			d.autorizador.llamadas != 0 ||
			d.transaccion.llamadas != 0 {
			t.Fatalf("rectificación sin segregación aceptada: %v", err)
		}
	})
}

func TestOperacionAnalisisRechazaReciboFueraDeContextoYConcesion(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-recibo-tardio-sintetico",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	d.transaccion.desfaseConfirmacion = 24 * time.Hour
	_, err := servicio.Registrar(context.Background(), escenario.registrar)
	if !errors.Is(err, ErrDependenciaOperacionAnalisisNoDisponible) ||
		d.transaccion.consumosFuentes != 0 ||
		d.transaccion.consumosV3 != 0 ||
		d.transaccion.commits != 0 {
		t.Fatalf("recibo 24h posterior aceptado: %v", err)
	}
}
