package ports

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestReciboConsultaPrevalidaLimitesAntesDeConstruirCompromisos(t *testing.T) {
	solicitud, datos := escenarioReciboConsultaConLimitesPrueba(t)

	casos := []struct {
		nombre        string
		mutar         func(*DatosReciboConsultaInternaFuenteAutoridad)
		auditoriaMala bool
	}{
		{
			nombre: "roles no permitidos",
			mutar: func(d *DatosReciboConsultaInternaFuenteAutoridad) {
				d.AuditoriaConfirmada.ActorRoles = []string{"rol_no_permitido"}
			},
			auditoriaMala: true,
		},
		{
			nombre: "faltan metadatos",
			mutar: func(d *DatosReciboConsultaInternaFuenteAutoridad) {
				delete(d.AuditoriaConfirmada.Metadata, "fuente_id")
			},
			auditoriaMala: true,
		},
		{
			nombre: "sobran metadatos",
			mutar: func(d *DatosReciboConsultaInternaFuenteAutoridad) {
				d.AuditoriaConfirmada.Metadata["metadato_no_permitido"] = "valor"
			},
			auditoriaMala: true,
		},
		{
			nombre: "valor de metadato demasiado largo",
			mutar: func(d *DatosReciboConsultaInternaFuenteAutoridad) {
				d.AuditoriaConfirmada.Metadata["fuente_id"] = strings.Repeat(
					"x", maximoTextoReciboConsultaAutoridadV1+1,
				)
			},
			auditoriaMala: true,
		},
		{
			nombre: "texto de auditoria demasiado largo",
			mutar: func(d *DatosReciboConsultaInternaFuenteAutoridad) {
				d.AuditoriaConfirmada.ActorID = strings.Repeat(
					"x", maximoTextoReciboConsultaAutoridadV1+1,
				)
			},
			auditoriaMala: true,
		},
		{
			nombre: "texto superior demasiado largo",
			mutar: func(d *DatosReciboConsultaInternaFuenteAutoridad) {
				d.TransaccionRef = strings.Repeat(
					"x", maximoTextoReciboConsultaAutoridadV1+1,
				)
			},
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			alterados := clonarDatosReciboConsultaAutoridad(datos)
			caso.mutar(&alterados)

			if err := prevalidarLimitesDatosReciboConsultaAutoridad(alterados); !errors.Is(
				err, ErrReciboConsultaFuenteAutoridadInvalido,
			) {
				t.Fatalf("la prevalidacion no rechazo la entrada: %v", err)
			}
			if caso.auditoriaMala {
				huella, err := huellaEntradaAuditoriaConsultaV1(alterados.AuditoriaConfirmada)
				if !errors.Is(err, ErrReciboConsultaFuenteAutoridadInvalido) || huella != "" {
					t.Fatalf("la entrada limitada produjo una huella: %q, %v", huella, err)
				}
			}
			if _, err := NuevoReciboConsultaInternaFuenteAutoridad(
				solicitud, alterados,
			); !errors.Is(err, ErrReciboConsultaFuenteAutoridadInvalido) {
				t.Fatalf("el constructor acepto la entrada fuera de limite: %v", err)
			}
		})
	}
}

func TestReciboConsultaPrevalidacionNoRechazaElLimiteExacto(t *testing.T) {
	_, datos := escenarioReciboConsultaConLimitesPrueba(t)
	datos.TransaccionRef = strings.Repeat("t", maximoTextoReciboConsultaAutoridadV1)
	datos.AuditoriaConfirmada.DocumentRef = strings.Repeat(
		"d", maximoTextoReciboConsultaAutoridadV1,
	)
	datos.AuditoriaConfirmada.Metadata["fuente_id"] = strings.Repeat(
		"f", maximoTextoReciboConsultaAutoridadV1,
	)

	if err := prevalidarLimitesDatosReciboConsultaAutoridad(datos); err != nil {
		t.Fatalf("la prevalidacion rechazo longitudes exactamente en el limite: %v", err)
	}
}

func TestReciboConsultaDetectaCorrupcionInternaPosteriorAntesDeCopiar(t *testing.T) {
	solicitud, datos := escenarioReciboConsultaConLimitesPrueba(t)
	recibo, err := NuevoReciboConsultaInternaFuenteAutoridad(solicitud, datos)
	if err != nil {
		t.Fatalf("precondicion de recibo valido: %v", err)
	}

	t.Run("rol inyectado", func(t *testing.T) {
		alterado := recibo
		copia := clonarDatosReciboConsultaAutoridad(*recibo.datos)
		copia.AuditoriaConfirmada.ActorRoles = []string{"rol_inyectado"}
		alterado.datos = &copia

		if obtenidos, err := alterado.Datos(); !errors.Is(
			err, ErrReciboConsultaFuenteAutoridadInvalido,
		) || !reflect.DeepEqual(obtenidos, DatosReciboConsultaInternaFuenteAutoridad{}) {
			t.Fatalf("la corrupcion interna produjo datos: %+v, %v", obtenidos, err)
		}
	})

	t.Run("metadato sobredimensionado", func(t *testing.T) {
		alterado := recibo
		copia := clonarDatosReciboConsultaAutoridad(*recibo.datos)
		copia.AuditoriaConfirmada.Metadata["fuente_id"] = strings.Repeat(
			"x", maximoTextoReciboConsultaAutoridadV1+1,
		)
		alterado.datos = &copia

		if obtenidos, err := alterado.Datos(); !errors.Is(
			err, ErrReciboConsultaFuenteAutoridadInvalido,
		) || !reflect.DeepEqual(obtenidos, DatosReciboConsultaInternaFuenteAutoridad{}) {
			t.Fatalf("la corrupcion interna produjo datos: %+v, %v", obtenidos, err)
		}
	})

	t.Run("campo superior sobredimensionado", func(t *testing.T) {
		alterado := recibo
		copia := clonarDatosReciboConsultaAutoridad(*recibo.datos)
		copia.TransaccionRef = strings.Repeat(
			"x", maximoTextoReciboConsultaAutoridadV1+1,
		)
		alterado.datos = &copia

		if obtenidos, err := alterado.Datos(); !errors.Is(
			err, ErrReciboConsultaFuenteAutoridadInvalido,
		) || !reflect.DeepEqual(obtenidos, DatosReciboConsultaInternaFuenteAutoridad{}) {
			t.Fatalf("la corrupcion interna produjo datos: %+v, %v", obtenidos, err)
		}
	})
}

func escenarioReciboConsultaConLimitesPrueba(
	t *testing.T,
) (SolicitudConsultaInternaGobernadaFuenteAutoridad, DatosReciboConsultaInternaFuenteAutoridad) {
	t.Helper()
	escenario := nuevoEscenarioPuertoAutoridad(t)
	selector := SelectorVersionFuenteAutoridad{FuenteID: escenario.Fuente.ID, Version: 1}
	solicitud, _, _ := solicitudConsultaGobernadaAutoridadPrueba(t, selector)
	estado, err := EstadoExactoFuenteAutoridad(escenario.Fuente)
	if err != nil {
		t.Fatalf("preparar estado de autoridad: %v", err)
	}
	return solicitud, datosReciboConsultaAutoridadPrueba(
		t, solicitud, ResultadoConsultaFuenteEncontrada, estado,
	)
}
