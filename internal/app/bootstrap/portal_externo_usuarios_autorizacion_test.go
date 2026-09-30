package bootstrap

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

type filaFachadaUsuariosExternoPrueba struct {
	nominal       bool
	configuracion []string
	err           error
}

func (f filaFachadaUsuariosExternoPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*(destinos[0].(*bool)) = f.nominal
	*(destinos[1].(*[]string)) = append([]string(nil), f.configuracion...)
	return nil
}

type consultaFachadaUsuariosExternoPrueba struct {
	fila filaFachadaUsuariosExternoPrueba
}

func (c consultaFachadaUsuariosExternoPrueba) QueryRow(context.Context, string, ...any) pgx.Row {
	return c.fila
}

func TestFachadaUsuariosExternaExigeClausuraFinalExacta(t *testing.T) {
	casos := []struct {
		nombre        string
		configuracion []string
		aceptar       bool
	}{
		{"final", []string{"search_path=pg_catalog, pg_temp"}, true},
		{"anterior", []string{"search_path=pg_catalog"}, false},
		{"temporal_primero", []string{"search_path=pg_temp, pg_catalog"}, false},
		{"configuracion_extra", []string{"search_path=pg_catalog, pg_temp", "row_security=off"}, false},
		{"ausente", nil, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			consulta := consultaFachadaUsuariosExternoPrueba{fila: filaFachadaUsuariosExternoPrueba{nominal: true, configuracion: caso.configuracion}}
			err := acreditarFachadaAutorizacionUsuariosExterno(t.Context(), consulta, "obtener_instantanea_usuarios_externo_v1", loginFuenteUsuariosExterno)
			if (err == nil) != caso.aceptar {
				t.Fatalf("aceptacion=%v, error=%v", caso.aceptar, err)
			}
		})
	}
}

func TestFachadaUsuariosExternaDeniegaIdentidadOConsultaIncierta(t *testing.T) {
	for _, fila := range []filaFachadaUsuariosExternoPrueba{
		{configuracion: []string{"search_path=pg_catalog, pg_temp"}},
		{nominal: true, configuracion: []string{"search_path=pg_catalog, pg_temp"}, err: errors.New("consulta fallida")},
	} {
		if acreditarFachadaAutorizacionUsuariosExterno(t.Context(), consultaFachadaUsuariosExternoPrueba{fila: fila}, "obtener_instantanea_usuarios_externo_v1", loginFuenteUsuariosExterno) == nil {
			t.Fatal("fachada no acreditada admitida")
		}
	}
	ctx, cancelar := context.WithCancel(t.Context())
	cancelar()
	if acreditarFachadaAutorizacionUsuariosExterno(ctx, consultaFachadaUsuariosExternoPrueba{fila: filaFachadaUsuariosExternoPrueba{nominal: true, configuracion: []string{"search_path=pg_catalog, pg_temp"}}}, "obtener_instantanea_usuarios_externo_v1", loginFuenteUsuariosExterno) == nil {
		t.Fatal("contexto cancelado admitido")
	}
}
