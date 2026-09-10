package domain

import (
	"errors"
	"testing"
)

func TestExpedienteRegistraPrimeraAnotacionAdministrativaSinTransicion(t *testing.T) {
	expediente := expedienteValido(t)
	actuacion := actuacion(
		string(AccionRegistrarAnotacionAdministrativa),
		string(expediente.FaseActual),
		expediente.ActualizadoEn.AddDate(0, 0, 1),
	)
	actuacion.EstadoDestino = expediente.EstadoActual
	actuacion.Observaciones = "Anotación administrativa sintética sobre la incorporación registrada."
	vinculo := VinculoSeguimientoOriginal{
		SeguimientoRef: "seguimiento_incorporacion_01", VersionSeguimiento: 1,
		HuellaRaizSeguimientoSHA256: cadena64("a"),
	}

	siguiente, err := expediente.RegistrarAnotacionAdministrativa(
		expediente.Version, vinculo, actuacion,
	)
	if err != nil {
		t.Fatalf("registrar anotación: %v", err)
	}
	if siguiente.Version != expediente.Version+1 ||
		siguiente.FaseActual != expediente.FaseActual ||
		siguiente.EstadoActual != expediente.EstadoActual ||
		len(siguiente.Actuaciones) != len(expediente.Actuaciones)+1 {
		t.Fatalf("transición no neutra: %#v", siguiente)
	}
	ultima := siguiente.Actuaciones[len(siguiente.Actuaciones)-1]
	if ultima.SeguimientoOriginal == nil || *ultima.SeguimientoOriginal != vinculo ||
		ultima.Observaciones != actuacion.Observaciones {
		t.Fatalf("vínculo o texto no preservados: %#v", ultima)
	}
	clon := siguiente.Clonar()
	clon.Actuaciones[len(clon.Actuaciones)-1].SeguimientoOriginal.SeguimientoRef = "mutado"
	if siguiente.Actuaciones[len(siguiente.Actuaciones)-1].SeguimientoOriginal.SeguimientoRef == "mutado" {
		t.Fatal("el clon comparte el vínculo de seguimiento")
	}
}

func TestExpedienteRechazaAnotacionAdministrativaNoNominal(t *testing.T) {
	base := expedienteValido(t)
	valida := func() DatosActuacion {
		a := actuacion(string(AccionRegistrarAnotacionAdministrativa), string(base.FaseActual), base.ActualizadoEn.AddDate(0, 0, 1))
		a.EstadoDestino = base.EstadoActual
		a.Observaciones = "Anotación administrativa sintética."
		return a
	}
	vinculo := VinculoSeguimientoOriginal{SeguimientoRef: "seguimiento_incorporacion_01", VersionSeguimiento: 1, HuellaRaizSeguimientoSHA256: cadena64("b")}
	casos := []struct {
		nombre string
		mutar  func(*DatosActuacion, *VinculoSeguimientoOriginal)
	}{
		{"sin observaciones", func(a *DatosActuacion, _ *VinculoSeguimientoOriginal) { a.Observaciones = "" }},
		{"cambia fase", func(a *DatosActuacion, _ *VinculoSeguimientoOriginal) { a.FaseDestino = "otra_fase" }},
		{"cambia estado", func(a *DatosActuacion, _ *VinculoSeguimientoOriginal) { a.EstadoDestino = EstadoIncidencia }},
		{"vínculo sin versión uno", func(_ *DatosActuacion, v *VinculoSeguimientoOriginal) { v.VersionSeguimiento = 2 }},
		{"vínculo sin huella", func(_ *DatosActuacion, v *VinculoSeguimientoOriginal) { v.HuellaRaizSeguimientoSHA256 = "" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			a, v := valida(), vinculo
			caso.mutar(&a, &v)
			if _, err := base.RegistrarAnotacionAdministrativa(base.Version, v, a); !errors.Is(err, ErrTransicionInvalida) {
				t.Fatalf("error = %v, se esperaba transición inválida", err)
			}
		})
	}

	primera, err := base.RegistrarAnotacionAdministrativa(base.Version, vinculo, valida())
	if err != nil {
		t.Fatalf("primera anotación: %v", err)
	}
	if _, err := primera.RegistrarAnotacionAdministrativa(primera.Version, vinculo, valida()); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("segunda anotación = %v, se esperaba transición inválida", err)
	}
}

func TestExpedienteRechazaSnapshotAnotacionTerminal(t *testing.T) {
	for _, terminal := range []EstadoOperativo{EstadoCompletado, EstadoCancelado} {
		t.Run(string(terminal), func(t *testing.T) {
			e := expedienteValido(t)
			e.Actuaciones[0].EstadoDestino = terminal
			e.EstadoActual = terminal
			a := actuacion(string(AccionRegistrarAnotacionAdministrativa), string(e.FaseActual), e.ActualizadoEn.AddDate(0, 0, 1))
			a.EstadoDestino = terminal
			a.Observaciones = "Nota administrativa."
			e.Version = 2
			e.Actuaciones = append(e.Actuaciones, e.nuevaActuacion(e.FaseActual, terminal, a, 2))
			e.ActualizadoEn = a.RealizadaEn
			e.Actuaciones[1].SeguimientoOriginal = &VinculoSeguimientoOriginal{SeguimientoRef: "seguimiento_incorporacion_01", VersionSeguimiento: 1, HuellaRaizSeguimientoSHA256: cadena64("a")}
			if e.Validar() == nil {
				t.Fatal("snapshot terminal aceptado")
			}
		})
	}
}
