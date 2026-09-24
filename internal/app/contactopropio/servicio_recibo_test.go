package contactopropio

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/usuarios"
	"vec-diputacion-granada/internal/vec/application"
)

func TestServicioRecibosSinConfiguracionNoMontaNiConsulta(t *testing.T) {
	if s, err := NuevoServicioConRecibos(nil, DependenciasConsultaRecibo{}); s != nil || !errors.Is(err, ErrContactoPropioNoDisponible) {
		t.Fatal("consulta sin autoridades configuradas")
	}
	if rutas, err := NuevasRutasConRecibos(nil, nil); len(rutas) != 0 || err == nil {
		t.Fatal("montó ruta sin autoridad")
	}
	var s *ServicioConRecibos
	if _, err := s.ConsultarRecibo(context.Background(), 1); !errors.Is(err, ErrContactoPropioNoDisponible) {
		t.Fatal("consultó sin identidad")
	}
	for _, version := range []uint64{0, 1 << 53} {
		if _, err := s.ConsultarRecibo(context.Background(), version); !errors.Is(err, ErrContactoPropioInvalido) {
			t.Fatal("versión no exacta admitida")
		}
	}
}

func TestAuditoriaReciboPropioUsaFuenteHMACYFinalidadPropia(t *testing.T) {
	p, actor, _ := entornoAuditoriaPropia(t)
	a, err := p.PrepararAuditoriaContactoUsuario(context.Background(), actor, application.AccionConsultarContactoUsuario, usuarios.ModuleID, actor.PersonaRef, 2)
	if err != nil {
		t.Fatal(err)
	}
	if a.Purpose != application.FinalidadConsultaReciboContactoUsuario || a.Action != application.AccionConsultarContactoUsuario || a.ObjectVersion != 2 || a.AuthorizationRef != "" || a.Signature != "" {
		t.Fatal("auditó envío/guardado o anticipó autorización")
	}
}
