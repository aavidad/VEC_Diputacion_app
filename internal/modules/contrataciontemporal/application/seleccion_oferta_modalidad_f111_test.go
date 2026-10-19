package application

import (
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestSeleccionOfertaF111RecuperaReciboHistoricoConIntencionActual(t *testing.T) {
	servicio, preparador, registro, publicador, autorizador, solicitud := escenarioSeleccionOfertaPrueba(t)
	original, err := servicio.SeleccionarOferta(t.Context(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if registro.estado.Situacion != ports.SeleccionOfertaConfirmada ||
		registro.estado.Comando.Datos.ModalidadClave != "" || registro.estado.Comando.Datos.CausaClave != "" {
		t.Fatal("el fixture no conserva un comando histórico confirmado")
	}

	// La consulta actual del expediente ya conoce estas claves. El comando
	// confirmado conserva los bytes originales y su recibo anterior.
	preparador.intencion.ModalidadClave = "interinidad"
	preparador.intencion.CausaClave = "sustitucion"
	preparador.intencion.HuellaIntencion = ports.HuellaIntencionSeleccionOfertaCT(preparador.intencion)
	if err := preparador.intencion.Validar(); err != nil {
		t.Fatalf("intención actual inválida: %v", err)
	}
	preparaciones, publicaciones, reservas, confirmaciones := preparador.preparaciones, publicador.llamadas, registro.reservas, registro.confirmaciones
	recuperado, err := servicio.SeleccionarOferta(t.Context(), solicitud)
	if err != nil || recuperado != original || preparador.preparaciones != preparaciones ||
		publicador.llamadas != publicaciones || registro.reservas != reservas || registro.confirmaciones != confirmaciones ||
		autorizador.etapas[len(autorizador.etapas)-1] != ports.EtapaSeleccionOfertaConsultar {
		t.Fatalf("repetición cambió recibo o repitió efectos: recibo=%+v err=%v", recuperado, err)
	}
}
