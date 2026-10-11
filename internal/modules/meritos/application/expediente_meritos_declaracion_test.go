package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

type comprobanteMeritoPrueba struct {
	documento  vec.ReferenciaDocumento
	err        error
	llamadas   int
	persona    string
	referencia string
}

func (c *comprobanteMeritoPrueba) ConsultarVersionConfirmadaPropia(_ context.Context, persona, referencia string) (vec.ReferenciaDocumento, error) {
	c.llamadas++
	c.persona, c.referencia = persona, referencia
	return c.documento, c.err
}

func TestDeclaracionConComprobanteSoloUsaVersionDeDocumentos(t *testing.T) {
	servicio, solicitud, _, registro, _ := escenario(t)
	solicitud.Hecho.Evidencias = nil
	c := &comprobanteMeritoPrueba{documento: vec.ReferenciaDocumento{ID: "documento:original", Version: 2}}
	puente, err := NuevaDeclaracionConComprobante(c, servicio)
	if err != nil {
		t.Fatal(err)
	}
	recibo, err := puente.Declarar(context.Background(), solicitud, "comprobante:prueba")
	if err != nil || c.llamadas != 1 || c.persona != persona || c.referencia != "comprobante:prueba" ||
		registro.confirmaciones != 1 || len(recibo.Registro.Hecho.Evidencias) != 1 ||
		recibo.Registro.Hecho.Evidencias[0] != c.documento {
		t.Fatalf("no se enlazo la version exacta: recibo=%+v error=%v", recibo, err)
	}
}

func TestDeclaracionConComprobanteRechazaEntradaYConsultaNoConfirmada(t *testing.T) {
	for _, caso := range []string{"evidencia_cliente", "ajeno", "pendiente", "sustituido", "invalido"} {
		t.Run(caso, func(t *testing.T) {
			servicio, solicitud, auth, registro, _ := escenario(t)
			solicitud.Hecho.Evidencias = nil
			c := &comprobanteMeritoPrueba{documento: vec.ReferenciaDocumento{ID: "documento:original", Version: 2}}
			switch caso {
			case "evidencia_cliente":
				solicitud.Hecho.Evidencias = []vec.ReferenciaDocumento{{ID: "documento:inyectado", Version: 1}}
			case "ajeno", "pendiente", "sustituido":
				// Documentos deniega antes de proyectar datos de ese comprobante.
				c.err = ports.ErrComprobanteMeritoNoDisponible
			case "invalido":
				c.documento.Version = 0
			}
			puente, _ := NuevaDeclaracionConComprobante(c, servicio)
			if recibo, err := puente.Declarar(context.Background(), solicitud, "comprobante:prueba"); err == nil ||
				recibo.Referencia != "" || registro.confirmaciones != 0 || auth.llamadas != 0 {
				t.Fatalf("comprobante no confirmado produjo declaracion: recibo=%+v error=%v", recibo, err)
			}
			if caso == "evidencia_cliente" && c.llamadas != 0 {
				t.Fatal("se leyo Documentos pese a referencia inyectada")
			}
		})
	}
}

func TestDeclaracionConComprobanteReintentaRUMSinConsumirDocumento(t *testing.T) {
	servicio, solicitud, _, registro, _ := escenario(t)
	solicitud.Hecho.Evidencias = nil
	c := &comprobanteMeritoPrueba{documento: vec.ReferenciaDocumento{ID: "documento:original", Version: 2}}
	puente, _ := NuevaDeclaracionConComprobante(c, servicio)
	registro.err = errors.New("confirmacion RUM incierta")
	if recibo, err := puente.Declarar(context.Background(), solicitud, "comprobante:prueba"); err == nil || recibo.Referencia != "" {
		t.Fatalf("fallo RUM presentado como exito: %+v, %v", recibo, err)
	}
	registro.err = nil
	primero, err := puente.Declarar(context.Background(), solicitud, "comprobante:prueba")
	if err != nil || primero.Referencia == "" || c.llamadas != 2 {
		t.Fatalf("reintento bloqueado por consulta Documentos: %+v, %v", primero, err)
	}
	segundo, err := puente.Declarar(context.Background(), solicitud, "comprobante:prueba")
	if err != nil || segundo.Referencia != primero.Referencia || registro.confirmaciones != 2 || registro.recuperaciones != 1 {
		t.Fatalf("replay creo otro merito o recibo: %+v, %v", segundo, err)
	}
}
