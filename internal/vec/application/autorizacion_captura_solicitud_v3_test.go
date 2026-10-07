package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestCapturaSolicitudLigadaV3UsaUnaFuenteYUnRegistro(t *testing.T) {
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	decision, confirmacion, captura, err := e.servicio.ExigirSolicitudLigadaV3ConCaptura(
		context.Background(), e.solicitud, e.resultado,
	)
	concreta, ok := captura.(CapturaEvaluacionSolicitudLigadaV3)
	if err != nil || decision.ValidarPara(e.solicitud) != nil || confirmacion.Validar() != nil ||
		!ok || concreta.datos == nil || e.fuente.invocaciones != 1 || e.concesiones.invocaciones != 1 {
		t.Fatalf("captura sin fuente/registro unico: %v", err)
	}
	if _, err := captura.InstantaneaPara(
		e.solicitud, e.resultado, decision, confirmacion,
		ports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, "audiencia", e.ahora,
	); !errors.Is(err, ErrCapturaEvaluacionSolicitudLigadaV3Invalida) {
		t.Fatalf("captura sin material accesible: %v", err)
	}
	alterado := resultadoContextoAutorizacionV3AlternativoPrueba(t, e.ahora)
	if _, err := captura.LigarMaterial(e.solicitud, alterado, decision, confirmacion,
		ports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, "audiencia"); !errors.Is(err, ErrCapturaEvaluacionSolicitudLigadaV3Invalida) {
		t.Fatalf("captura acepto actor ajeno: %v", err)
	}
	if _, err := captura.LigarMaterial(e.solicitud, e.resultado, decision, confirmacion,
		ports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, "audiencia"); !errors.Is(err, ErrCapturaEvaluacionSolicitudLigadaV3Invalida) {
		t.Fatalf("captura acepto material sin preimagen: %v", err)
	}
	if _, err := json.Marshal(captura); err == nil ||
		strings.Contains(fmt.Sprintf("%+v", captura), e.resultado.RegistroContextoRef) {
		t.Fatal("captura serializable o expuesta en log")
	}
	// La fuente devuelve slices. La captura debe conservar la misma evaluacion
	// aunque el adaptador reutilice y altere después su memoria.
	huellaOriginal, err := concreta.datos.instantanea.AsignacionPerfil.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	e.fuente.instantanea.AsignacionPerfil.Ambitos[0].Valores[0] = "otro"
	huellaCaptura, err := concreta.datos.instantanea.AsignacionPerfil.HuellaSHA256()
	if err != nil || huellaOriginal != huellaCaptura {
		t.Fatalf("la captura compartio memoria con fuente: %v", err)
	}
}

func TestCapturaSolicitudLigadaV3NuncaSaleTrasFalloDeConfirmacion(t *testing.T) {
	for nombre, fallo := range map[string]error{
		"registro": errors.New("fallo sintetico"),
		"obsoleta": ports.ErrInstantaneaAutorizacionObsoleta,
	} {
		t.Run(nombre, func(t *testing.T) {
			e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
			e.concesiones.err = fallo
			_, confirmacion, captura, err := e.servicio.ExigirSolicitudLigadaV3ConCaptura(
				context.Background(), e.solicitud, e.resultado,
			)
			if err == nil || captura != nil || confirmacion.Validar() == nil ||
				e.fuente.invocaciones != 1 || e.concesiones.invocaciones != 1 {
				t.Fatalf("captura tras fallo durable: %v", err)
			}
		})
	}
	t.Run("confirmacion malformada", func(t *testing.T) {
		e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
		e.concesiones.devolverCero = true
		_, confirmacion, captura, err := e.servicio.ExigirSolicitudLigadaV3ConCaptura(
			context.Background(), e.solicitud, e.resultado,
		)
		if err == nil || captura != nil || confirmacion.Validar() == nil ||
			e.fuente.invocaciones != 1 || e.concesiones.invocaciones != 1 {
			t.Fatalf("captura tras confirmacion invalida: %v", err)
		}
	})
}
