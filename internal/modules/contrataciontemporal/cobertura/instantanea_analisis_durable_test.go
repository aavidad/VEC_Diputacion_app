package cobertura

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

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type lectorInstantaneaAnalisisDurablePrueba struct {
	expediente domain.Expediente
	err        error
	llamadas   int
	cancelar   context.CancelFunc
}

func (l *lectorInstantaneaAnalisisDurablePrueba) LeerExpedienteAnalisisDurableO3(
	_ context.Context,
	_ SolicitudInstantaneaAnalisisDurableO3,
) (domain.Expediente, error) {
	l.llamadas++
	if l.cancelar != nil {
		l.cancelar()
	}
	return l.expediente, l.err
}

func expedienteConAnalisisDurableO3Prueba(
	t *testing.T,
	resultado domain.ResultadoValidacionRC,
) domain.Expediente {
	t.Helper()
	expediente := expedienteOperacionDecisionCoberturaPrueba(t)
	instante := instanteOperacionDecisionCoberturaPrueba.Add(time.Minute)
	entrada := domain.VinculoEntradaRC{
		Referencia:   "entrada_rc_analisis_durable_o3_01",
		HuellaSHA256: strings.Repeat("6", 64),
	}
	analisis := domain.AnalisisRRHH{
		ModalidadClave:    "modalidad_interinidad_publicada",
		CategoriaRef:      expediente.Solicitud.CategoriaRef,
		GrupoSubgrupo:     expediente.Solicitud.GrupoSubgrupo,
		CausaClave:        "causa_sustitucion_publicada",
		Periodo:           expediente.Solicitud.Periodo,
		PorcentajeJornada: domain.JornadaCompletaDiezmilesimas,
		EntradaRCEsperada: entrada,
		ValidacionRC: domain.ValidacionRC{
			Resultado:           resultado,
			EntradaRef:          entrada.Referencia,
			HuellaEntradaSHA256: entrada.HuellaSHA256,
			FuenteRef:           "fuente_presupuestaria_durable_o3_01",
			ReciboRef:           "recibo_validacion_rc_durable_o3_01",
			ValidadaEn:          instante.Add(-time.Second),
			Motivo:              "Resultado sintético gobernado para la prueba.",
		},
	}
	siguiente, err := expediente.RegistrarAnalisis(
		expediente.Version,
		analisis,
		domain.DatosActuacion{
			AccionClave: domain.ClaveCatalogo(
				ports.AccionRegistrarAnalisis,
			),
			ActorRef:      "actor_rrhh_analisis_durable_o3_01",
			UnidadRef:     "unidad_rrhh_analisis_durable_o3_01",
			ReciboRef:     "recibo_confirmacion_analisis_o3_01",
			RealizadaEn:   instante,
			FaseDestino:   "analisis_rrhh",
			EstadoDestino: domain.EstadoEnCurso,
		},
	)
	if err != nil {
		t.Fatalf("registrar análisis durable O3 de prueba: %v", err)
	}
	return siguiente
}

func solicitudInstantaneaAnalisisDurableO3Prueba(
	t *testing.T,
	expediente domain.Expediente,
) SolicitudInstantaneaAnalisisDurableO3 {
	t.Helper()
	solicitud, err := NuevaSolicitudInstantaneaAnalisisDurableO3(
		expediente.OrganizacionRef,
		expediente.Referencia,
		expediente.Version,
	)
	if err != nil {
		t.Fatal(err)
	}
	return solicitud
}

func TestInstantaneaAnalisisDurableDerivaIdentidadO3YClona(t *testing.T) {
	expediente := expedienteConAnalisisDurableO3Prueba(
		t,
		domain.RCNoRequerida,
	)
	solicitud := solicitudInstantaneaAnalisisDurableO3Prueba(t, expediente)
	lector := &lectorInstantaneaAnalisisDurablePrueba{expediente: expediente}
	instantanea, err := ObtenerInstantaneaAnalisisDurableO3(
		context.Background(),
		lector,
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	obtenido, analisisRef, huella, err := instantanea.DesplegarPara(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	esperada, err := ports.HuellaAnalisisRRHHRehidratadoO3(
		*expediente.Analisis,
	)
	if err != nil {
		t.Fatal(err)
	}
	if lector.llamadas != 1 ||
		analisisRef != expediente.Analisis.ActuacionRegistro.ReciboRef ||
		huella != esperada ||
		!reflect.DeepEqual(obtenido, expediente) {
		t.Fatal("la instantánea perdió identidad durable O3")
	}

	categoriaOriginal := expediente.Analisis.CategoriaRef
	obtenido.Analisis.CategoriaRef = "categoria_mutada_por_consumidor"
	obtenido.Actuaciones[0].DocumentosRef = append(
		obtenido.Actuaciones[0].DocumentosRef,
		"documento_mutado_por_consumidor",
	)
	lector.expediente.Analisis.CategoriaRef = "categoria_mutada_por_lector"
	segundo, segundoRef, segundaHuella, err :=
		instantanea.DesplegarPara(solicitud)
	if err != nil ||
		segundo.Analisis.CategoriaRef != categoriaOriginal ||
		segundoRef != analisisRef ||
		segundaHuella != huella {
		t.Fatalf("la instantánea comparte memoria mutable: %v", err)
	}
}

func TestInstantaneaAnalisisDurableRechazaAgregadoNoAutoritativo(t *testing.T) {
	base := expedienteConAnalisisDurableO3Prueba(t, domain.RCNoRequerida)
	casos := []struct {
		nombre     string
		expediente func() domain.Expediente
		solicitud  func(*testing.T, domain.Expediente) SolicitudInstantaneaAnalisisDurableO3
	}{
		{
			nombre: "organizacion distinta",
			expediente: func() domain.Expediente {
				return base.Clonar()
			},
			solicitud: func(t *testing.T, e domain.Expediente) SolicitudInstantaneaAnalisisDurableO3 {
				e.OrganizacionRef = "organizacion_durable_ajena_01"
				return solicitudInstantaneaAnalisisDurableO3Prueba(t, e)
			},
		},
		{
			nombre: "version distinta",
			expediente: func() domain.Expediente {
				return base.Clonar()
			},
			solicitud: func(t *testing.T, e domain.Expediente) SolicitudInstantaneaAnalisisDurableO3 {
				solicitud, err := NuevaSolicitudInstantaneaAnalisisDurableO3(
					e.OrganizacionRef,
					e.Referencia,
					e.Version+1,
				)
				if err != nil {
					t.Fatal(err)
				}
				return solicitud
			},
		},
		{
			nombre: "sin analisis",
			expediente: func() domain.Expediente {
				e := base.Clonar()
				e.Analisis = nil
				return e
			},
			solicitud: solicitudInstantaneaAnalisisDurableO3Prueba,
		},
		{
			nombre: "ligadura adulterada",
			expediente: func() domain.Expediente {
				e := base.Clonar()
				e.Analisis.ActuacionRegistro.ReciboRef =
					"recibo_confirmacion_adulterado_o3_01"
				return e
			},
			solicitud: solicitudInstantaneaAnalisisDurableO3Prueba,
		},
		{
			nombre: "accion ajena a O3",
			expediente: func() domain.Expediente {
				e := base.Clonar()
				indice := e.Analisis.ActuacionRegistro.Secuencia - 1
				e.Analisis.ActuacionRegistro.AccionClave =
					"analisis_accion_no_o3"
				e.Actuaciones[indice].AccionClave = "analisis_accion_no_o3"
				return e
			},
			solicitud: solicitudInstantaneaAnalisisDurableO3Prueba,
		},
		{
			nombre: "RC no habilitante",
			expediente: func() domain.Expediente {
				return expedienteConAnalisisDurableO3Prueba(
					t,
					domain.RCRechazada,
				)
			},
			solicitud: solicitudInstantaneaAnalisisDurableO3Prueba,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			expediente := caso.expediente()
			solicitud := caso.solicitud(t, expediente)
			lector := &lectorInstantaneaAnalisisDurablePrueba{
				expediente: expediente,
			}
			_, err := ObtenerInstantaneaAnalisisDurableO3(
				context.Background(),
				lector,
				solicitud,
			)
			if !errors.Is(err, ErrInstantaneaAnalisisDurableNoConfiable) {
				t.Fatalf("agregado no confiable aceptado: %v", err)
			}
		})
	}
}

func TestInstantaneaAnalisisDurableFallaCerradaEnFrontera(t *testing.T) {
	expediente := expedienteConAnalisisDurableO3Prueba(
		t,
		domain.RCNoRequerida,
	)
	solicitud := solicitudInstantaneaAnalisisDurableO3Prueba(t, expediente)
	t.Run("contexto nulo", func(t *testing.T) {
		lector := &lectorInstantaneaAnalisisDurablePrueba{expediente: expediente}
		_, err := ObtenerInstantaneaAnalisisDurableO3(nil, lector, solicitud)
		if !errors.Is(err, ErrSolicitudInstantaneaAnalisisDurableInvalida) ||
			lector.llamadas != 0 {
			t.Fatalf("contexto nulo alcanzó el lector: %v", err)
		}
	})
	t.Run("contexto cancelado", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		lector := &lectorInstantaneaAnalisisDurablePrueba{expediente: expediente}
		_, err := ObtenerInstantaneaAnalisisDurableO3(ctx, lector, solicitud)
		if !errors.Is(err, ErrInstantaneaAnalisisDurableNoDisponible) ||
			!errors.Is(err, context.Canceled) || lector.llamadas != 0 {
			t.Fatalf("cancelación no priorizada: %v", err)
		}
	})
	t.Run("cancelacion durante lectura", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		lector := &lectorInstantaneaAnalisisDurablePrueba{
			expediente: expediente,
			cancelar:   cancelar,
		}
		_, err := ObtenerInstantaneaAnalisisDurableO3(ctx, lector, solicitud)
		if !errors.Is(err, ErrInstantaneaAnalisisDurableNoDisponible) ||
			!errors.Is(err, context.Canceled) || lector.llamadas != 1 {
			t.Fatalf("resultado posterior a cancelación aceptado: %v", err)
		}
	})
	t.Run("error privado redactado", func(t *testing.T) {
		privado := errors.New("dsn=material_privado_no_exponible")
		lector := &lectorInstantaneaAnalisisDurablePrueba{err: privado}
		_, err := ObtenerInstantaneaAnalisisDurableO3(
			context.Background(),
			lector,
			solicitud,
		)
		if !errors.Is(err, ErrInstantaneaAnalisisDurableNoDisponible) ||
			errors.Is(err, privado) ||
			strings.Contains(err.Error(), "material_privado") {
			t.Fatalf("se filtró el error privado: %v", err)
		}
	})
	t.Run("lector tipado nulo", func(t *testing.T) {
		var lector *lectorInstantaneaAnalisisDurablePrueba
		_, err := ObtenerInstantaneaAnalisisDurableO3(
			context.Background(),
			lector,
			solicitud,
		)
		if !errors.Is(err, ErrSolicitudInstantaneaAnalisisDurableInvalida) {
			t.Fatalf("lector tipado nulo aceptado: %v", err)
		}
	})
}

func TestSolicitudInstantaneaAnalisisDurableAcotaVersionQuePuedeAvanzar(
	t *testing.T,
) {
	if _, err := NuevaSolicitudInstantaneaAnalisisDurableO3(
		"organizacion_instantanea_durable_01",
		"expediente_instantanea_durable_01",
		MaximoEnteroSeguroOperacionDecisionCobertura-1,
	); err != nil {
		t.Fatalf("máxima versión que puede avanzar rechazada: %v", err)
	}
	if _, err := NuevaSolicitudInstantaneaAnalisisDurableO3(
		"organizacion_instantanea_durable_01",
		"expediente_instantanea_durable_01",
		MaximoEnteroSeguroOperacionDecisionCobertura,
	); !errors.Is(err, ErrSolicitudInstantaneaAnalisisDurableInvalida) {
		t.Fatalf("versión sin sucesora interoperable aceptada: %v", err)
	}
}

func TestInstantaneaAnalisisDurableEsOpacaYNoSerializable(t *testing.T) {
	expediente := expedienteConAnalisisDurableO3Prueba(
		t,
		domain.RCNoRequerida,
	)
	solicitud := solicitudInstantaneaAnalisisDurableO3Prueba(t, expediente)
	instantanea, err := ObtenerInstantaneaAnalisisDurableO3(
		context.Background(),
		&lectorInstantaneaAnalisisDurablePrueba{expediente: expediente},
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, valor := range []string{
		fmt.Sprintf("%v", instantanea),
		fmt.Sprintf("%+v", instantanea),
		fmt.Sprintf("%#v", instantanea),
		slog.AnyValue(instantanea).String(),
	} {
		if strings.Contains(valor, expediente.Referencia) ||
			strings.Contains(valor, instantanea.analisisRef) ||
			!strings.Contains(valor, "OPACA") {
			t.Fatalf("representación no redactada: %q", valor)
		}
	}
	if _, err := json.Marshal(instantanea); !errors.Is(
		err,
		ErrSerializacionOperacionDecisionCoberturaProhibida,
	) {
		t.Fatalf("JSON permitió serializar la capacidad: %v", err)
	}
	if _, err := xml.Marshal(instantanea); !errors.Is(
		err,
		ErrSerializacionOperacionDecisionCoberturaProhibida,
	) {
		t.Fatalf("XML permitió serializar la capacidad: %v", err)
	}
	if _, err := instantanea.MarshalText(); !errors.Is(
		err,
		ErrSerializacionOperacionDecisionCoberturaProhibida,
	) {
		t.Fatalf("texto permitió serializar la capacidad: %v", err)
	}
	if _, err := instantanea.MarshalBinary(); !errors.Is(
		err,
		ErrSerializacionOperacionDecisionCoberturaProhibida,
	) {
		t.Fatalf("binario permitió serializar la capacidad: %v", err)
	}
	var destino bytes.Buffer
	if err := gob.NewEncoder(&destino).Encode(instantanea); !errors.Is(
		err,
		ErrSerializacionOperacionDecisionCoberturaProhibida,
	) {
		t.Fatalf("gob permitió serializar la capacidad: %v", err)
	}
	tipo := reflect.TypeOf(instantanea)
	for indice := 0; indice < tipo.NumField(); indice++ {
		if tipo.Field(indice).IsExported() {
			t.Fatalf("la capacidad expone %s", tipo.Field(indice).Name)
		}
	}
	if _, _, _, err := (InstantaneaAnalisisDurableO3{}).DesplegarPara(
		solicitud,
	); !errors.Is(err, ErrInstantaneaAnalisisDurableNoConfiable) {
		t.Fatalf("el valor cero adquirió autoridad: %v", err)
	}
}

func TestInstantaneaAnalisisDurableDetectaAdulteracionInterna(t *testing.T) {
	expediente := expedienteConAnalisisDurableO3Prueba(
		t,
		domain.RCNoRequerida,
	)
	solicitud := solicitudInstantaneaAnalisisDurableO3Prueba(t, expediente)
	nueva := func() InstantaneaAnalisisDurableO3 {
		instantanea, err := ObtenerInstantaneaAnalisisDurableO3(
			context.Background(),
			&lectorInstantaneaAnalisisDurablePrueba{expediente: expediente},
			solicitud,
		)
		if err != nil {
			t.Fatal(err)
		}
		return instantanea
	}
	casos := []struct {
		nombre    string
		adulterar func(*InstantaneaAnalisisDurableO3)
	}{
		{"referencia", func(i *InstantaneaAnalisisDurableO3) {
			i.analisisRef = "recibo_analisis_adulterado_01"
		}},
		{"huella", func(i *InstantaneaAnalisisDurableO3) {
			i.analisisHuellaSHA256 = strings.Repeat("f", 64)
		}},
		{"expediente", func(i *InstantaneaAnalisisDurableO3) {
			i.expediente.Analisis.CategoriaRef = "categoria_adulterada"
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			instantanea := nueva()
			caso.adulterar(&instantanea)
			if _, _, _, err := instantanea.DesplegarPara(solicitud); !errors.Is(
				err,
				ErrInstantaneaAnalisisDurableNoConfiable,
			) {
				t.Fatalf("adulteración aceptada: %v", err)
			}
		})
	}
}
