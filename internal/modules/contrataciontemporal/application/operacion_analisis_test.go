package application

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"gopkg.in/yaml.v3"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

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
		d.autorizador.llamadas != 1 || d.transaccion.llamadas != 1 {
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
	datosV3, err := d.autorizador.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if datosV3.Recurso.Atributos[ports.AtributoArtefactoAnalisisRef] !=
		recibo.ArtefactoRef ||
		datosV3.Recurso.Atributos[ports.AtributoArtefactoAnalisisHuella] !=
			recibo.ArtefactoHuellaSHA256 ||
		len(datosV3.Recurso.Ambitos) != 4 ||
		len(datosV3.Recurso.Atributos) != 9 {
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
			evidencia.Politica.ActorAnalisisAnteriorRef {
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
	d.preparaciones.confirmado = &primero

	segundo, err := servicio.Registrar(context.Background(), escenario.registrar)
	if err != nil {
		t.Fatal(err)
	}
	if segundo != primero || d.preparaciones.llamadas != 2 ||
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
	d.preparaciones.err =
		ports.ErrClaveIdempotenciaOperacionAnalisisUsada
	cambiada := escenario.registrar
	cambiada.DatosFuncionales.PorcentajeJornada = 5_000

	_, err := servicio.Registrar(context.Background(), cambiada)
	if !errors.Is(err, ErrOperacionAnalisisEnConflicto) ||
		len(d.sellador.preimagenes) != 2 {
		t.Fatalf("se esperaba conflicto semántico, recibido: %v", err)
	}
	primera, _ := d.sellador.preimagenes[0].BytesSemantica()
	segunda, _ := d.sellador.preimagenes[1].BytesSemantica()
	if bytes.Equal(primera, segunda) {
		t.Fatal("la preimagen semántica no liga los datos funcionales")
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
		d.transaccion.llamadas != 1 {
		t.Fatalf("conflicto CAS mal clasificado: %v", err)
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
			errors.Is(err, ErrDependenciaOperacionAnalisisNoDisponible) {
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
			d.autorizador.llamadas != 0 || d.transaccion.llamadas != 0 {
			t.Fatalf("política no confiable aceptada: %v", err)
		}
	})
}

func TestOperacionAnalisisRespetaCancelacionAntesYDespuesDelCommit(
	t *testing.T,
) {
	t.Run("antes", func(t *testing.T) {
		escenario := nuevoEscenarioOperacionAnalisisSaneado(
			t,
			ports.OperacionRegistrarAnalisis,
			"-cancelacion-previa-sintetica",
		)
		servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
		ctx, cancelar := context.WithCancel(context.Background())
		d.sellador.antes = cancelar
		_, err := servicio.Registrar(ctx, escenario.registrar)
		if !errors.Is(err, context.Canceled) ||
			d.preparaciones.llamadas != 0 || d.transaccion.llamadas != 0 {
			t.Fatalf("cancelación previa ambigua: %v", err)
		}
	})
	t.Run("despues", func(t *testing.T) {
		escenario := nuevoEscenarioOperacionAnalisisSaneado(
			t,
			ports.OperacionRegistrarAnalisis,
			"-cancelacion-posterior-sintetica",
		)
		servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
		ctx, cancelar := context.WithCancel(context.Background())
		d.transaccion.despues = cancelar
		recibo, err := servicio.Registrar(ctx, escenario.registrar)
		if err != nil || recibo.VersionResultante != 2 ||
			d.transaccion.llamadas != 1 {
			t.Fatalf("commit confirmado quedó ambiguo: %#v, %v", recibo, err)
		}
	})
}

func TestOperacionAnalisisRechazaVersionSinMargenYDependenciaTypedNil(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-limites-sinteticos",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	solicitud := escenario.registrar
	solicitud.VersionEsperada = ports.MaximoEnteroSeguroOperacionAnalisis
	_, err := servicio.Registrar(context.Background(), solicitud)
	if !errors.Is(err, ErrSolicitudOperacionAnalisisInvalida) ||
		d.contextos.llamadas != 0 {
		t.Fatalf("versión sin margen aceptada: %v", err)
	}
	var artefactos *preparadorArtefactoAnalisisDoble
	if _, err = NuevoServicioOperacionAnalisis(
		d.contextos,
		artefactos,
		d.sellador,
		d.preparaciones,
		d.politicas,
		d.correlaciones,
		d.autorizador,
		d.reloj,
		d.transaccion,
	); !errors.Is(err, ErrServicioOperacionAnalisisInvalido) {
		t.Fatalf("typed nil aceptado: %v", err)
	}
}

func TestOperacionAnalisisNoRectificaConCoberturaYaMaterializada(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRectificarAnalisis,
		"-cobertura-sintetica",
	)
	instanteCobertura := escenario.instante.Add(-5 * time.Minute)
	conCobertura, err := escenario.expediente.RegistrarViaCobertura(
		escenario.expediente.Version,
		domain.DecisionViaCobertura{
			ViaClave:         "via.cobertura_sintetica",
			ProcedimientoRef: "procedimiento:sintetico-001",
			BolsaRef:         "bolsa:sintetica-001",
			Comprobaciones: []domain.ComprobacionCobertura{{
				Clave:      "comprobacion.sintetica",
				Resultado:  domain.ComprobacionAfirmativa,
				FuenteRef:  "fuente:cobertura-sintetica-001",
				ReciboRef:  "recibo:cobertura-sintetica-001",
				EvaluadaEn: instanteCobertura,
			}},
			Motivacion: "Motivación enteramente sintética.",
		},
		domain.DatosActuacion{
			AccionClave:   "cobertura.sintetica_registrada",
			ActorRef:      "actor:cobertura-sintetico-001",
			UnidadRef:     "unidad:cobertura-sintetica-001",
			ReciboRef:     "recibo:cobertura-accion-sintetico-001",
			RealizadaEn:   instanteCobertura,
			FaseDestino:   escenario.expediente.FaseActual,
			EstadoDestino: escenario.expediente.EstadoActual,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	escenario.expediente = conCobertura
	escenario.rectificar.VersionEsperada = conCobertura.Version
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)

	_, err = servicio.Rectificar(context.Background(), escenario.rectificar)
	if !errors.Is(err, ErrSolicitudOperacionAnalisisInvalida) ||
		d.transaccion.llamadas != 0 {
		t.Fatalf("se rectificó una cobertura materializada: %v", err)
	}
}

func TestOrdenOperacionAnalisisRechazaCambioExtraYTodosLosCodecs(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-orden-sintetica",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	if _, err := servicio.Registrar(
		context.Background(),
		escenario.registrar,
	); err != nil {
		t.Fatal(err)
	}
	evidencia, err := d.transaccion.orden.Datos()
	if err != nil {
		t.Fatal(err)
	}
	alterado := evidencia.ExpedienteSiguiente.Clonar()
	alterado.NumeroVisible = "2026/SINT-9999"
	_, err = ports.NuevaOrdenConfirmarOperacionAnalisis(
		ports.DatosOrdenConfirmarOperacionAnalisis{
			SolicitudArtefacto:   evidencia.SolicitudArtefacto,
			Artefacto:            evidencia.Artefacto,
			SolicitudPreparacion: evidencia.SolicitudPreparacion,
			Preparacion:          evidencia.Preparacion,
			SolicitudPolitica:    evidencia.SolicitudPolitica,
			Politica:             evidencia.Politica,
			SolicitudV3:          evidencia.SolicitudV3,
			DecisionV3:           evidencia.DecisionV3,
			ConfirmacionV3:       evidencia.ConfirmacionV3,
			InstanteEfecto:       evidencia.InstanteEfecto,
			ExpedienteSiguiente:  alterado,
		},
	)
	if !errors.Is(err, ports.ErrOrdenOperacionAnalisisInvalida) {
		t.Fatal("la fábrica aceptó un cambio adicional del expediente")
	}
	comprobarError := func(nombre string, err error) {
		t.Helper()
		if err == nil {
			t.Fatalf("%s serializó una orden opaca", nombre)
		}
	}
	orden := d.transaccion.orden
	_, err = json.Marshal(orden)
	comprobarError("json", err)
	_, err = xml.Marshal(orden)
	comprobarError("xml", err)
	_, err = orden.MarshalText()
	comprobarError("texto", err)
	_, err = orden.MarshalBinary()
	comprobarError("binario", err)
	var destino bytes.Buffer
	comprobarError("gob", gob.NewEncoder(&destino).Encode(orden))
	_, err = cbor.Marshal(orden)
	comprobarError("cbor", err)
	_, err = yaml.Marshal(orden)
	comprobarError("yaml", err)
	if strings.Contains(fmt.Sprint(orden), escenario.registrar.ExpedienteRef) ||
		strings.Contains(fmt.Sprintf("%#v", orden), escenario.registrar.ArtefactoRef) {
		t.Fatal("la representación textual expone coordenadas internas")
	}
}
