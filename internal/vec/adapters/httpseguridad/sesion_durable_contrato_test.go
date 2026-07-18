package httpseguridad

import (
	"context"
	"strings"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestContratosDurablesPublicosFallanCerrados(t *testing.T) {
	if !EstadoControlSesionActiva.Valido() || !EstadoControlSesionRevocada.Valido() || EstadoControlSesion("").Valido() {
		t.Fatal("catalogo de estados de control abierto o incompleto")
	}
	ahora := time.Date(2026, 7, 18, 9, 0, 0, 123_456_000, time.UTC)
	configuracion := configuracionInternaValida()
	verificador := &verificadorFalso{}
	registro := nuevoRegistroMemoria()
	servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceHigh), registro, &relojFijo{ahora: ahora})
	canal := debeCanalTLS(t, servicio, configuracion)
	verificador.fijarAsercion(asercionInternaValida(ahora, configuracion, canal))
	identidad, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("documento-protegido"), canal))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if err := identidad.confirmacion.AltaConfirmada.Validar(); err != nil {
		t.Fatalf("alta emitida invalida: %v", err)
	}
	if err := identidad.confirmacion.ValidarPara(identidad.confirmacion.AltaConfirmada); err != nil {
		t.Fatalf("confirmacion emitida invalida: %v", err)
	}
	otraAlta := identidad.confirmacion.AltaConfirmada
	otraAlta.SesionID = "otra-sesion"
	if identidad.confirmacion.ValidarPara(otraAlta) == nil {
		t.Fatal("un recibo de otra alta fue aceptado")
	}
	if _, _, err := servicio.ProyectarCuentaAutenticada(context.Background(), identidad); err != nil {
		t.Fatalf("proyectar: %v", err)
	}
	registro.mu.Lock()
	consulta := registro.ultimaConsulta
	registro.mu.Unlock()
	if err := consulta.Validar(); err != nil || !consulta.CoincideExactamente(consulta) {
		t.Fatalf("consulta emitida invalida: %#v %v", consulta, err)
	}

	mutacionesAlta := []func(*AltaSesionAtomica){
		func(a *AltaSesionAtomica) { a.AsercionID = " " + a.AsercionID },
		func(a *AltaSesionAtomica) { a.CuentaID = strings.ToUpper(a.CuentaID) },
		func(a *AltaSesionAtomica) { a.Superficie = SuperficiePublicaAnonima },
		func(a *AltaSesionAtomica) { a.EspacioIdentidad = "http://idp-inseguro.example.test" },
		func(a *AltaSesionAtomica) { a.MetodoObservado = dominiovec.AuthMethodDemo },
		func(a *AltaSesionAtomica) { a.AutenticacionHuellaSHA256 = strings.Repeat("A", 64) },
		func(a *AltaSesionAtomica) { a.AutenticacionHuellaSHA256 = strings.Repeat("0", 64) },
		func(a *AltaSesionAtomica) { a.AutenticacionVerificadaEn = a.SesionEmitidaEn.Add(time.Microsecond) },
		func(a *AltaSesionAtomica) { a.SesionEmitidaEn = a.SesionEmitidaEn.Add(time.Nanosecond) },
		func(a *AltaSesionAtomica) {
			a.AsercionExpiraEn = a.SesionEmitidaEn.Add(duracionLimiteAsercion + time.Microsecond)
		},
		func(a *AltaSesionAtomica) { a.PoliticaGarantiaRef = "pga_corta" },
		func(a *AltaSesionAtomica) { a.PoliticaGarantiaHuellaSHA256 = "sha256:" + strings.Repeat("a", 64) },
		func(a *AltaSesionAtomica) { a.PoliticaGarantiaHuellaSHA256 = strings.Repeat("0", 64) },
	}
	for indice, mutar := range mutacionesAlta {
		alterada := identidad.confirmacion.AltaConfirmada
		mutar(&alterada)
		if alterada.Validar() == nil {
			t.Fatalf("alta alterada %d aceptada: %#v", indice, alterada)
		}
	}

	mutacionesConsulta := []func(*ConsultaSesionActiva){
		func(c *ConsultaSesionActiva) { c.AutenticacionRef = "aut_corta" },
		func(c *ConsultaSesionActiva) { c.ControlSesionRevision = 0 },
		func(c *ConsultaSesionActiva) { c.ControlSesionEstado = EstadoControlSesionRevocada },
		func(c *ConsultaSesionActiva) { c.ControlSesionHuellaSHA256 = strings.Repeat("A", 64) },
		func(c *ConsultaSesionActiva) { c.ControlSesionHuellaSHA256 = strings.Repeat("0", 64) },
		func(c *ConsultaSesionActiva) { c.SesionRevalidadaEn = c.SesionRevalidadaEn.Add(time.Nanosecond) },
		func(c *ConsultaSesionActiva) { c.SesionValidaHasta = c.SesionRevalidadaEn },
		func(c *ConsultaSesionActiva) { c.CuentaOrdinariaRef = "cta_otra23456789abcdefghijkl" },
	}
	for indice, mutar := range mutacionesConsulta {
		alterada := consulta
		mutar(&alterada)
		if alterada.Validar() == nil || consulta.CoincideExactamente(alterada) {
			t.Fatalf("consulta alterada %d aceptada: %#v", indice, alterada)
		}
	}
}

func TestMetodoKerberosSePersisteComoKerberosAD(t *testing.T) {
	ahora := time.Date(2026, 7, 18, 9, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	verificador := &verificadorFalso{}
	servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceHigh), nuevoRegistroMemoria(), &relojFijo{ahora: ahora})
	canal := debeCanalTLS(t, servicio, configuracion)
	asercion := asercionInternaValida(ahora, configuracion, canal)
	asercion.MetodoPrimario = MetodoKerberos
	verificador.fijarAsercion(asercion)
	identidad, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("kerberos-protegido"), canal))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if identidad.confirmacion.AltaConfirmada.MetodoObservado != dominiovec.AuthMethodKerberos {
		t.Fatalf("metodo SQL incorrecto: %q", identidad.confirmacion.AltaConfirmada.MetodoObservado)
	}
}
