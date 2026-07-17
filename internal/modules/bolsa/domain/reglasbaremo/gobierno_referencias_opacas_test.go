package reglasbaremo

import (
	"errors"
	"testing"
	"time"
)

func TestGobiernoRechazaDatosPersonalesComoReferenciasTecnicas(t *testing.T) {
	conjunto := conjuntoPrueba(t, true)
	motivo := motivoGobiernoPrueba(t, "creacion")
	for _, actor := range []string{
		"12345678Z",
		"Alberto Apellidos",
		"principal:rrhh:tecnico-1",
		"per_demasiado_corta",
	} {
		if _, err := NuevaVersionGobernadaReglasBaremo(
			conjunto, actor, motivo, instanteBaseGobiernoPrueba,
		); !errors.Is(err, ErrGobiernoValorInvalido) {
			t.Fatalf("actor personal/no opaco aceptado %q: %v", actor, err)
		}
	}
	if _, err := NuevaVersionGobernadaReglasBaremo(
		conjunto,
		"per_0123456789abcdefghijkl",
		motivo,
		instanteBaseGobiernoPrueba,
	); err != nil {
		t.Fatalf("referencia per_ canonica VEC rechazada: %v", err)
	}

	borrador := nuevaVersionGobiernoPrueba(t, instanteBaseGobiernoPrueba)
	aprobacion := aprobacionGobiernoPrueba(
		t, borrador, instanteBaseGobiernoPrueba.Add(time.Minute),
	)
	for _, firmante := range []string{"12345678Z", "Nombre Apellidos", "svc_0123456789abcdef0123456789abcdef"} {
		datos := DatosAtestacionAprobacionFirmadaReglasBaremo{
			Atestacion: aprobacion.Atestacion(), Vinculo: aprobacion.Vinculo(),
			Firma: aprobacion.Firma(), PoliticaFirma: aprobacion.PoliticaFirma(),
			Firmantes: []string{firmante}, FirmadaEn: aprobacion.FirmadaEn(),
			VerificadaEn: aprobacion.VerificadaEn(), ValidaHasta: aprobacion.ValidaHasta(),
		}
		if _, err := NuevaAtestacionAprobacionFirmadaReglasBaremo(datos); !errors.Is(err, ErrGobiernoEvidenciaInvalida) {
			t.Fatalf("firmante no per_ aceptado %q: %v", firmante, err)
		}
	}

	publicada := publicarGobiernoPrueba(
		t, borrador, instanteBaseGobiernoPrueba.Add(3*time.Minute),
	)
	dependencias := dependenciasGobiernoPrueba(
		t, publicada, instanteBaseGobiernoPrueba.Add(4*time.Minute),
	)
	for _, verificador := range []string{"12345678Z", "Nombre del servicio", actorGobiernoPrueba} {
		datos := datosDependenciasDesdeAtestacion(dependencias)
		datos.VerificadorRef = verificador
		if _, err := NuevaAtestacionDependenciasVigentesReglasBaremo(datos); !errors.Is(err, ErrGobiernoEvidenciaInvalida) {
			t.Fatalf("verificador no svc_ aceptado %q: %v", verificador, err)
		}
	}

	autoridad := autoridadGobiernoPrueba(
		t, borrador, AccionDescartarReglasBaremo, actorGobiernoPrueba, nil,
		instanteBaseGobiernoPrueba.Add(time.Minute),
	)
	for _, principal := range []string{"12345678Z", "Nombre Apellidos", "svc_0123456789abcdef0123456789abcdef"} {
		datos := DatosAtestacionAutoridadReglasBaremo{
			Atestacion: autoridad.Atestacion(), Vinculo: autoridad.Vinculo(),
			Accion: autoridad.Accion(), PrincipalRef: principal,
			EmitidaEn: autoridad.EmitidaEn(), ValidaHasta: autoridad.ValidaHasta(),
		}
		if _, err := NuevaAtestacionAutoridadReglasBaremo(datos); !errors.Is(err, ErrGobiernoEvidenciaInvalida) {
			t.Fatalf("principal no per_ aceptado %q: %v", principal, err)
		}
	}
}
