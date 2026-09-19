//go:build development

package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

type filaIdentidadAuditoriaFronteraPrueba struct {
	usuario string
	valida  bool
	err     error
}

func (f filaIdentidadAuditoriaFronteraPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*(destinos[0].(*string)) = f.usuario
	*(destinos[1].(*bool)) = f.valida
	return nil
}

type consultadorIdentidadAuditoriaFronteraPrueba struct {
	fila     filaIdentidadAuditoriaFronteraPrueba
	consulta string
	rol      string
}

func (c *consultadorIdentidadAuditoriaFronteraPrueba) QueryRow(
	_ context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	c.consulta = consulta
	if len(argumentos) == 1 {
		c.rol, _ = argumentos[0].(string)
	}
	return c.fila
}

func TestComprobarIdentidadAuditoriaFronteraRechazaMembresiasIncompatibles(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nombre string
	}{
		{nombre: "confirmador y registrador"},
		{nombre: "lector y registrador"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			consultador := &consultadorIdentidadAuditoriaFronteraPrueba{
				fila: filaIdentidadAuditoriaFronteraPrueba{
					usuario: "login_registrador_prueba",
					valida:  false,
				},
			}
			_, err := comprobarIdentidadPostgreSQLContratacionTemporalDesarrollo(
				context.Background(),
				consultador,
				rolAuditoriaFronteraPostgreSQLContratacionTemporalDesarrollo,
			)
			if !errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible) {
				t.Fatalf("error=%v; se esperaba denegación cerrada", err)
			}
			if consultador.rol != rolAuditoriaFronteraPostgreSQLContratacionTemporalDesarrollo {
				t.Fatalf("rol consultado=%q", consultador.rol)
			}
			for _, fragmento := range []string{
				"WITH RECURSIVE membresias_efectivas",
				"pg_catalog.pg_auth_members",
				"rol_id <> $1::regrole",
			} {
				if !strings.Contains(consultador.consulta, fragmento) {
					t.Fatalf("consulta no recorre membresías efectivas: falta %q", fragmento)
				}
			}
		})
	}
}

func TestComprobarIdentidadAuditoriaFronteraAceptaSoloRegistrador(t *testing.T) {
	t.Parallel()

	consultador := &consultadorIdentidadAuditoriaFronteraPrueba{
		fila: filaIdentidadAuditoriaFronteraPrueba{
			usuario: "login_registrador_prueba",
			valida:  true,
		},
	}
	usuario, err := comprobarIdentidadPostgreSQLContratacionTemporalDesarrollo(
		context.Background(),
		consultador,
		rolAuditoriaFronteraPostgreSQLContratacionTemporalDesarrollo,
	)
	if err != nil {
		t.Fatalf("comprobar identidad: %v", err)
	}
	if usuario != "login_registrador_prueba" {
		t.Fatalf("usuario=%q", usuario)
	}
}
