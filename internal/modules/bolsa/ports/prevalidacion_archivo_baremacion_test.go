package ports

import (
	"encoding/base64"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestPrevalidacionArchivoProbatorioTieneAccionYCamposMinimosCerrados(t *testing.T) {
	if AccionPrevalidarArchivoProbatorioBaremacion != "bolsa.baremacion.archivo.prevalidar" {
		t.Fatalf("accion contractual inesperada: %q", AccionPrevalidarArchivoProbatorioBaremacion)
	}
	clase, existe := ClaseRecursoRequeridaOperacionBaremacion(
		AccionPrevalidarArchivoProbatorioBaremacion,
	)
	if !existe || clase != ClaseRecursoBaremacion {
		t.Fatalf("clase de recurso inesperada: %q existe=%v", clase, existe)
	}
	campos, existe := CamposRequeridosOperacionBaremacion(
		AccionPrevalidarArchivoProbatorioBaremacion,
	)
	if !existe || !reflect.DeepEqual(campos, []string{"archivo_probatorio"}) {
		t.Fatalf("campos de minimo privilegio inesperados: %v", campos)
	}

	contexto := contextoOperacionValido(
		AccionPrevalidarArchivoProbatorioBaremacion, "baremacion-1",
	)
	if contexto.ValidarPara(
		AccionPrevalidarArchivoProbatorioBaremacion, ClaseRecursoBaremacion, "baremacion-1",
	) != nil {
		t.Fatal("la capacidad dedicada valida fue rechazada")
	}
	if contexto.ValidarPara(
		AccionConfirmarDecisionBaremacion, ClaseRecursoBaremacion, "baremacion-1",
	) == nil {
		t.Fatal("la capacidad de prevalidacion habilito confirmar")
	}
}

func TestPrevalidacionArchivoProbatorioRechazaCamposAmpliados(t *testing.T) {
	decision := decisionAutorizacionValida(
		AccionPrevalidarArchivoProbatorioBaremacion, "baremacion-1",
	)
	decision.CamposPermitidos = append(decision.CamposPermitidos, "decision")
	if _, err := nuevaAutorizacionPrueba(decision); !errors.Is(err, ErrAutorizacionBaremacionInvalida) {
		t.Fatalf("campos ampliados no fallaron en cerrado: %v", err)
	}
}

func TestConfirmacionDeclaraContextoPrevalidacionArchivoTipado(t *testing.T) {
	campo, existe := reflect.TypeOf(SolicitudConfirmarCambioBaremacion{}).
		FieldByName("ContextoPrevalidacionArchivo")
	if !existe || campo.Type != reflect.TypeOf(ContextoOperacionBaremacion{}) {
		t.Fatalf("contexto dedicado ausente o no tipado: %+v", campo)
	}
}

func TestAltaExigeAusenciaExactaDePrevalidacionAunqueLaCapacidadYaNoEsteVigente(t *testing.T) {
	baremacion := baremacionValidaPrueba(t)
	token, err := NuevoTokenReservaBaremacion(
		base64.RawURLEncoding.EncodeToString([]byte("abcdefghijklmnopqrstuvwx")),
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := SolicitudConfirmarCambioBaremacion{
		Contexto: contextoOperacionValido(AccionConfirmarAltaBaremacion, baremacion.ID),
		ContextoPrevalidacionArchivo: contextoOperacionValido(
			AccionPrevalidarArchivoProbatorioBaremacion, baremacion.ID,
		),
		Token: token, Clase: ClaseCambioAltaBaremacion,
		HuellaSolicitudHMAC: "hmac-sha256:confirmacion_v2:" + huellaPruebaPuertos("a"),
		Agregado:            baremacion,
		Trazabilidad: TrazabilidadCambioBaremacion{
			MotivoClave: "alta_merito", Motivo: "Alta del merito calculado oficialmente.",
		},
		// La ventana de uso de la capacidad termina mucho antes. Su presencia
		// sigue sin equivaler a ausencia y debe rechazarse.
		ConfirmadaEn: instantePuertosPrueba.Add(5 * time.Minute),
	}
	if solicitud.ContextoPrevalidacionArchivo.EsNulo() || solicitud.Validar() == nil {
		t.Fatal("alta admitio una capacidad de prevalidacion ignorada por el canonico")
	}
}
