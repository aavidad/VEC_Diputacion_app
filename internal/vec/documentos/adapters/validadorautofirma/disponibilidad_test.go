package validadorautofirma_test

import (
	"context"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/adapters/validadorautofirma/servidorprueba"
)

func TestDisponibilidadSoloPorRespuestaAcreditada(t *testing.T) {
	for _, caso := range []struct {
		nombre        string
		escenario     servidorprueba.Escenario
		observaciones []bool
	}{
		{"dictamen_valido", servidorprueba.ValidaSinSello, []bool{true}},
		{"firma_no_valida", servidorprueba.IntegridadRota, []bool{true}},
		{"servicio_5xx", servidorprueba.ErrorInterno, []bool{false}},
		{"json_no_interpretable", servidorprueba.SinDictamen, nil},
		{"contrato_distinto", servidorprueba.ContratoDesconocido, nil},
		{"tipo_incorrecto", servidorprueba.TipoIncorrecto, nil},
		{"eco_firmado_distinto", servidorprueba.HuellaEcoDistinta, nil},
		{"eco_original_distinto", servidorprueba.HuellaOriginalDistinta, nil},
		{"dictamen_incoherente", servidorprueba.ValidaIncoherente, nil},
		{"peticion_rechazada", servidorprueba.FormatoNoDetectado, nil},
		{"redireccion", servidorprueba.Redireccion, nil},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			s := arrancar(t, false)
			s.Escenario(caso.escenario)
			c := configuracion(s)
			peticion := solicitud([]byte{0x30})
			anterior, err := cliente(t, c).VerificarMotivado(context.Background(), peticion)
			if err != nil {
				t.Fatal(err)
			}
			var observadas []bool
			contexto := context.WithValue(context.Background(), claveDisponibilidadPrueba{}, "correlacion_interna")
			c.Disponibilidad = func(ctx context.Context, disponible bool) {
				if ctx != contexto {
					t.Error("el observador perdió el contexto original")
				}
				observadas = append(observadas, disponible)
			}
			actual, err := cliente(t, c).VerificarMotivado(contexto, peticion)
			if err != nil {
				t.Fatal(err)
			}
			if actual.Motivo != anterior.Motivo || actual.Resultado.Estado != anterior.Resultado.Estado ||
				actual.Resultado.HuellaOriginalSHA256 != anterior.Resultado.HuellaOriginalSHA256 ||
				actual.Resultado.HuellaFirmadoSHA256 != anterior.Resultado.HuellaFirmadoSHA256 {
				t.Fatal("el observador cambió la decisión o las huellas del documento")
			}
			if len(observadas) != len(caso.observaciones) {
				t.Fatalf("observaciones=%v, esperadas=%v", observadas, caso.observaciones)
			}
			for i := range observadas {
				if observadas[i] != caso.observaciones[i] {
					t.Fatalf("observaciones=%v", observadas)
				}
			}
		})
	}
}

type claveDisponibilidadPrueba struct{}

func TestDisponibilidadExcluyeCancelacionCredencialTLSYTimeout(t *testing.T) {
	for _, nombre := range []string{"cancelacion", "credencial", "tls", "timeout"} {
		t.Run(nombre, func(t *testing.T) {
			s := arrancar(t, nombre == "tls")
			c := configuracion(s)
			contexto := context.Background()
			if nombre == "cancelacion" {
				var cancelar context.CancelFunc
				contexto, cancelar = context.WithCancel(contexto)
				cancelar()
			}
			if nombre == "credencial" {
				c.Token = []byte("credencial_sintetica_distinta_de_la_admitida")
			}
			if nombre == "timeout" {
				s.Escenario(servidorprueba.Lento)
				s.Retardo(80 * time.Millisecond)
				c.Timeout = 5 * time.Millisecond
			}
			c.Disponibilidad = func(context.Context, bool) { t.Error("un fallo excluido produjo señal de disponibilidad") }
			_, _ = cliente(t, c).VerificarMotivado(contexto, solicitud([]byte{0x30}))
		})
	}
}
