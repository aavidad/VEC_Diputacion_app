package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/composicion/identidadordinaria"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestSesionFirmanteCertificadoExigeFuenteComunYCotejosAUT56(t *testing.T) {
	e := nuevoEntornoFirmanteV2Prueba(t)
	if _, err := nuevaAutoridadSesionFirmanteV2Certificado(nil, e.reloj); !errors.Is(err, errSesionFirmanteV2NoDisponible) {
		t.Fatalf("fuente ausente: %v", err)
	}
	a, err := nuevaAutoridadSesionFirmanteV2Certificado(&identidadordinaria.FuenteCertificadoTemporal{}, e.reloj)
	if err != nil || a.garantia != core.AuthAssuranceSubstantial {
		t.Fatalf("garantía temporal: %v", err)
	}
	f := a.fuente.(*fuenteCertificadoFirmaVecV2)
	ahora := e.reloj.Ahora()
	q := ports.SolicitudSesionFirmanteV2{
		CertificadoCanalSHA256: e.huella, PersonaEsperadaRef: e.persona,
		CuentaEsperadaRef: e.seleccion.CuentaRef, PerfilEsperadoRef: e.seleccion.PerfilActivoRef,
		RolEsperadoID: e.seleccion.RolID, CertificadoVerificadoEn: ahora,
		CertificadoTLSValidoHasta: ahora.Add(time.Minute),
	}
	if !solicitudCertificadoFirmaVecV2Valida(q, ahora) {
		t.Fatal("cotejos completos rechazados")
	}
	for nombre, cambiar := range map[string]func(*ports.SolicitudSesionFirmanteV2){
		"sin_certificado": func(q *ports.SolicitudSesionFirmanteV2) { q.CertificadoCanalSHA256 = "" },
		"sin_persona":     func(q *ports.SolicitudSesionFirmanteV2) { q.PersonaEsperadaRef = "" },
		"sin_cuenta":      func(q *ports.SolicitudSesionFirmanteV2) { q.CuentaEsperadaRef = "" },
		"sin_perfil":      func(q *ports.SolicitudSesionFirmanteV2) { q.PerfilEsperadoRef = "" },
		"sin_rol":         func(q *ports.SolicitudSesionFirmanteV2) { q.RolEsperadoID = "" },
		"sin_vigencia":    func(q *ports.SolicitudSesionFirmanteV2) { q.CertificadoTLSValidoHasta = ahora },
		"fecha_futura":    func(q *ports.SolicitudSesionFirmanteV2) { q.CertificadoVerificadoEn = ahora.Add(time.Second) },
	} {
		t.Run(nombre, func(t *testing.T) {
			mala := q
			cambiar(&mala)
			if _, err := f.AbrirSesionFirmanteV2(context.Background(), mala); !errors.Is(err, errSesionFirmanteV2Denegada) {
				t.Fatalf("cotejo ausente admitido: %v", err)
			}
		})
	}
	if _, err := f.AbrirSesionFirmanteV2(context.Background(), q); !errors.Is(err, errSesionFirmanteV2Denegada) {
		t.Fatalf("sin cápsula vinculada abrió sesión: %v", err)
	}
}

func TestSesionFirmanteCertificadoConservaCancelacionSinFiltrarCausa(t *testing.T) {
	e := nuevoEntornoFirmanteV2Prueba(t)
	a, err := nuevaAutoridadSesionFirmanteV2Certificado(&identidadordinaria.FuenteCertificadoTemporal{}, e.reloj)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	q := ports.SolicitudSesionFirmanteV2{CertificadoCanalSHA256: e.huella,
		PersonaEsperadaRef: e.persona, CuentaEsperadaRef: e.seleccion.CuentaRef,
		PerfilEsperadoRef: e.seleccion.PerfilActivoRef, RolEsperadoID: e.seleccion.RolID,
		CertificadoVerificadoEn: e.reloj.Ahora(), CertificadoTLSValidoHasta: e.reloj.Ahora().Add(time.Minute)}
	_, err = a.fuente.AbrirSesionFirmanteV2(ctx, q)
	if !errors.Is(err, errSesionFirmanteV2Denegada) || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación previa inesperada: %v", err)
	}
	opaco := falloSesionFirmanteV2(errors.New("material privado de la autoridad"))
	if !errors.Is(opaco, errSesionFirmanteV2Denegada) ||
		strings.Contains(fmt.Sprintf("%+v", opaco), "material privado") {
		t.Fatal("error externo filtró la causa")
	}
	if !errors.Is(falloSesionFirmanteV2(context.Canceled), context.Canceled) {
		t.Fatal("cancelación interna sin causa")
	}
}
