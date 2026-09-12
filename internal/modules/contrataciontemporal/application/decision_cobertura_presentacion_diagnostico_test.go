package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestProponerCoberturaDiagnosticaEtapaSinConservarCausa(t *testing.T) {
	cases := []struct {
		nombre   string
		preparar func(*escenarioPresentacionCobertura, error)
		etapa    EtapaDiagnosticoPresentacionPropuestaCobertura
	}{
		{
			nombre: "contexto", etapa: EtapaDiagnosticoPresentacionContexto,
			preparar: func(e *escenarioPresentacionCobertura, causa error) {
				e.contextos.err = causa
			},
		},
		{
			nombre: "autorizacion", etapa: EtapaDiagnosticoPresentacionAutorizacion,
			preparar: func(e *escenarioPresentacionCobertura, causa error) {
				e.accesos.err = causa
			},
		},
		{
			nombre: "lector O3", etapa: EtapaDiagnosticoPresentacionLectorO3,
			preparar: func(e *escenarioPresentacionCobertura, causa error) {
				e.analisis.err = causa
			},
		},
		{
			nombre: "gobierno", etapa: EtapaDiagnosticoPresentacionGobierno,
			preparar: func(e *escenarioPresentacionCobertura, causa error) {
				e.gobierno.err = causa
			},
		},
		{
			nombre: "reloj", etapa: EtapaDiagnosticoPresentacionReloj,
			preparar: func(e *escenarioPresentacionCobertura, causa error) {
				e.reloj.err = causa
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioPresentacionCobertura(t, viasPresentacionCoberturaPrueba(1))
			causa := errors.New("postgres://persona:secreto@privado/rrhh")
			tc.preparar(escenario, causa)

			_, err := escenario.servicio.Proponer(context.Background(), escenario.solicitud)
			if !errors.Is(err, ErrPresentacionPropuestaCoberturaNoDisponible) || errors.Is(err, causa) {
				t.Fatalf("la indisponibilidad perdió su contrato saneado: %v", err)
			}
			if err.Error() != ErrPresentacionPropuestaCoberturaNoDisponible.Error() {
				t.Fatalf("el error público cambió o expuso una etapa: %q", err.Error())
			}
			etapa, ok := EtapaDiagnosticoDePresentacionPropuestaCobertura(err)
			if !ok || etapa != tc.etapa {
				t.Fatalf("etapa=%q, ok=%t; se esperaba %q", etapa, ok, tc.etapa)
			}
			texto := fmt.Sprintf("%v|%+v|%#v", err, err, err)
			if strings.Contains(texto, "persona") || strings.Contains(texto, "secreto") || strings.Contains(texto, "postgres") {
				t.Fatalf("la causa privada salió por la superficie pública: %q", texto)
			}
			valor := slog.AnyValue(err).Resolve()
			if valor.Kind() != slog.KindString || valor.String() != string(tc.etapa) {
				t.Fatalf("el registro interno no recibió el código estable: %#v", valor)
			}
		})
	}
}

func TestEtapaDiagnosticoPresentacionNoClasificaErroresAjeno(t *testing.T) {
	if etapa, ok := EtapaDiagnosticoDePresentacionPropuestaCobertura(errors.New("causa privada")); ok || etapa != "" {
		t.Fatalf("un error ajeno se presentó como diagnóstico interno: etapa=%q ok=%t", etapa, ok)
	}
}

func TestClasificacionDiagnosticoPresentacionReconstruyeCadenaReconocida(t *testing.T) {
	servicio := &ServicioPresentacionPropuestaCobertura{}
	diagnostico := nuevoErrorEtapaDiagnosticoPresentacionPropuestaCobertura(
		EtapaDiagnosticoPresentacionLectorO3,
	)
	for nombre, causa := range map[string]error{
		"envuelta": fmt.Errorf("privado: %w", diagnostico),
		"unida":    errors.Join(errors.New("postgres://persona:secreto@privado/rrhh"), diagnostico),
	} {
		t.Run(nombre, func(t *testing.T) {
			for clasificacion, clasificar := range map[string]func(context.Context, error) error{
				"contexto": func(ctx context.Context, err error) error {
					return servicio.clasificarFalloContexto(ctx, err, EtapaDiagnosticoPresentacionContexto)
				},
				"dependencia": func(ctx context.Context, err error) error {
					return servicio.clasificarFalloDependencia(ctx, err, EtapaDiagnosticoPresentacionPreparador)
				},
			} {
				t.Run(clasificacion, func(t *testing.T) {
					err := clasificar(context.Background(), causa)
					if !errors.Is(err, ErrPresentacionPropuestaCoberturaNoDisponible) ||
						errors.Is(err, causa) || err.Error() != ErrPresentacionPropuestaCoberturaNoDisponible.Error() {
						t.Fatalf("la clasificación retuvo la cadena privada: %v", err)
					}
					if siguiente := errors.Unwrap(err); siguiente != ErrPresentacionPropuestaCoberturaNoDisponible || errors.Unwrap(siguiente) != nil {
						t.Fatalf("unwrap no quedó limitado al centinela: %#v", err)
					}
					etapa, ok := EtapaDiagnosticoDePresentacionPropuestaCobertura(err)
					if !ok || etapa != EtapaDiagnosticoPresentacionLectorO3 {
						t.Fatalf("etapa saneada=%q ok=%t", etapa, ok)
					}
					if registro := slog.AnyValue(err).Resolve(); registro.Kind() != slog.KindString || registro.String() != string(etapa) {
						t.Fatalf("registro no redactado: %#v", registro)
					}
					texto := fmt.Sprintf("%v|%+v|%#v", err, err, err)
					if strings.Contains(texto, "privado") || strings.Contains(texto, "secreto") || strings.Contains(texto, "postgres") {
						t.Fatalf("la causa cruzó la frontera: %q", texto)
					}
				})
			}
		})
	}
}
