package catalogoalta

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type recuperadorDiagnosticoPrueba struct {
	resultado ports.ResultadoInstantaneaNecesidadAlta
	err       error
}

func (r recuperadorDiagnosticoPrueba) RecuperarInstantaneaNecesidadConfirmada(
	context.Context, ports.ConsultaInstantaneaNecesidadConfirmada,
) (ports.ResultadoInstantaneaNecesidadAlta, error) {
	return r.resultado, r.err
}

type falloSQLFuentePrueba struct{ codigo string }

func (e *falloSQLFuentePrueba) Error() string    { return "causa privada de la lectura" }
func (e *falloSQLFuentePrueba) SQLState() string { return e.codigo }

func consultaInstantaneaDiagnosticoPrueba(t *testing.T) ports.ConsultaInstantaneaNecesidadConfirmada {
	t.Helper()
	ambitos, err := ports.NuevaColeccionSellosHMAC(
		"hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:"+strings.Repeat("a", 64), nil)
	if err != nil {
		t.Fatal(err)
	}
	consulta := ports.ConsultaInstantaneaNecesidadConfirmada{
		AmbitosHMAC: ambitos, OrganizacionRef: "organizacion:dipgra",
		ActorRef: "actor:prueba", PerfilRef: "perfil:rrhh",
	}
	if err := consulta.Validar(); err != nil {
		t.Fatal(err)
	}
	return consulta
}

func TestFuenteRecuperacionConservaCausaOriginalYSeparaResultadoInvalido(t *testing.T) {
	consulta := consultaInstantaneaDiagnosticoPrueba(t)
	original := &falloSQLFuentePrueba{codigo: "42501"}
	for _, caso := range []struct {
		nombre    string
		recuperar recuperadorDiagnosticoPrueba
		conCausa  bool
	}{
		{"error original", recuperadorDiagnosticoPrueba{err: original}, true},
		{"resultado invalido sin error", recuperadorDiagnosticoPrueba{
			resultado: ports.ResultadoInstantaneaNecesidadAlta{Estado: "invalido"}}, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			fuente, err := NuevaFuente("", caso.recuperar)
			if err != nil {
				t.Fatal(err)
			}
			resultado, err := fuente.RecuperarInstantaneaNecesidadConfirmada(context.Background(), consulta)
			if resultado.Estado != "" || !errors.Is(err, ports.ErrFuenteNecesidadesAltaNoDisponible) ||
				errors.Is(err, original) != caso.conCausa {
				t.Fatalf("recuperación distinguió mal error y resultado: %v", err)
			}
			var sql *falloSQLFuentePrueba
			if errors.As(err, &sql) != caso.conCausa {
				t.Fatal("la causa SQL original no quedó accesible solo en el caso de error")
			}
		})
	}
}
