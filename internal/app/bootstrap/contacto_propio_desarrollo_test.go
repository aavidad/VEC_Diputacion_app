package bootstrap

import (
	"context"
	"net/http"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/contactopropio"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/application"
)

func TestContactoPropioDeclaraTresFronterasExactasSinAbrirOtras(t *testing.T) {
	perfil := "prf_0123456789abcdefghijkl"
	c, err := nuevoCatalogoFronterasComunDesarrollo(descriptoresFronterasContactoPropioDesarrollo(perfil))
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct{ metodo, ruta, capacidad string }{
		{http.MethodPost, contactopropio.RutaContactoPropio, "vec.contacto_usuario.guardar"},
		{http.MethodGet, contactopropio.RutaContactoPropio, application.AccionVersionContactoPropia},
		{http.MethodPost, "/api/vec/usuarios/contacto-propio/recibo", application.AccionConsultarContactoUsuario},
	}
	for _, v := range casos {
		d, ok := c.resolver(v.metodo, v.ruta)
		if !ok || d.Superficie != superficieExternaPersonalSeguridadComunDesarrollo || !d.admitePerfil(perfil) || d.ClaveCapacidad != v.capacidad {
			t.Fatalf("frontera no exacta: %+v", v)
		}
	}
	for _, v := range []struct{ metodo, ruta string }{{http.MethodGet, "/api/vec/usuarios/contacto-propio/recibo"}, {http.MethodPost, "/api/vec/usuarios/contacto-propio/ajeno"}, {http.MethodDelete, contactopropio.RutaContactoPropio}} {
		if _, ok := c.resolver(v.metodo, v.ruta); ok {
			t.Fatalf("frontera abierta: %+v", v)
		}
	}
	if _, _, err := nuevasRutasContactoPropioDesarrollo(context.Background(), config.Config{}, nil, nil, nil, nil, c); err == nil {
		t.Fatal("dependencias ausentes admitidas")
	}
}

func TestContactoOpcionalCaidoCierraYConservaRaiz(t *testing.T) {
	cierres := 0
	rutas, cerrar := intentarRutasContactoPropioDesarrollo(func() ([]vechttp.RutaExacta, func(), error) {
		return []vechttp.RutaExacta{{Ruta: contactopropio.RutaContactoPropio}}, func() { cierres++ }, errContactoPropioDesarrolloNoDisponible
	})
	if len(rutas) != 0 || cerrar == nil || cierres != 1 {
		t.Fatal("fallo del contacto opcional bloqueó o filtró ruta")
	}
	cerrar()
	if cierres != 1 {
		t.Fatal("cierre doble")
	}
}
