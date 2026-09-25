package httpseguridad

import (
	"context"
	"errors"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func configuracionCertificadoDesarrollo() ConfiguracionSuperficie {
	c := configuracionInternaValida()
	c.PoliticaInterna = PoliticaInternaDesarrolloCertificadoPersonal
	c.RetiradaPoliticaInternaEn = time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	c.MetodosAdmitidos = []MetodoAutenticacion{MetodoCertificado}
	c.FactoresRequeridos = []MetodoAutenticacion{MetodoCertificado}
	c.MinimoFactoresVerificados = 1
	c.MinimoGruposCriptograficosDistintos = 1
	c.GarantiaMinima = dominiovec.AuthAssuranceSubstantial
	return c
}

func TestCertificadoDesarrolloSoloEnInternaYConRetirada(t *testing.T) {
	base := configuracionCertificadoDesarrollo()
	if err := base.Validar(); err != nil {
		t.Fatalf("politica temporal valida: %v", err)
	}
	pruebas := []struct {
		nombre  string
		cambiar func(*ConfiguracionSuperficie)
	}{
		{"sin fecha de retirada", func(c *ConfiguracionSuperficie) { c.RetiradaPoliticaInternaEn = time.Time{} }},
		{"fecha local ambigua", func(c *ConfiguracionSuperficie) {
			c.RetiradaPoliticaInternaEn = c.RetiradaPoliticaInternaEn.In(time.FixedZone("local", 3600))
		}},
		{"Kerberos adicional", func(c *ConfiguracionSuperficie) { c.MetodosAdmitidos = append(c.MetodosAdmitidos, MetodoKerberos) }},
		{"factor requerido ausente", func(c *ConfiguracionSuperficie) { c.FactoresRequeridos = nil }},
		{"alta por equivalencia", func(c *ConfiguracionSuperficie) { c.GarantiaMinima = dominiovec.AuthAssuranceHigh }},
		{"administracion", func(c *ConfiguracionSuperficie) {
			c.Superficie = SuperficieAdministracionPrivilegiada
			c.ZonaRed = ZonaRedAdministracion
			c.RequiereCuentaPrivilegiada = true
		}},
		{"sin modo pero con fecha", func(c *ConfiguracionSuperficie) { c.PoliticaInterna = "" }},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			c := base
			prueba.cambiar(&c)
			if err := c.Validar(); !errors.Is(err, ErrConfiguracionSuperficie) {
				t.Fatalf("la variante debe denegarse: %v", err)
			}
		})
	}
}

func TestCertificadoDesarrolloGarantiaRealYCaducidad(t *testing.T) {
	ahora := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	c := configuracionCertificadoDesarrollo()
	verificador := &verificadorFalso{}
	evaluador := evaluadorValido(dominiovec.AuthAssuranceSubstantial)
	reloj := &relojFijo{ahora: ahora}
	servicio := debeServicio(t, c, verificador, evaluador, nuevoRegistroMemoria(), reloj)
	canal := debeCanalTLS(t, servicio, c)
	asercion := asercionInternaValida(ahora, c, canal)
	asercion.ACRVerificado = ACRCertificadoPersonalDesarrolloProtegido
	asercion.Factores = asercion.Factores[1:]
	verificador.fijarAsercion(asercion)
	credencial := debeCredencial(t, []byte("asercion-protegida"), canal)

	sinProteccion := asercion
	sinProteccion.ACRVerificado = "urn:vec:acr:certificado-desarrollo"
	verificador.fijarAsercion(sinProteccion)
	if _, err := servicio.Resolver(context.Background(), credencial); !errors.Is(err, ErrAsercionNoValida) {
		t.Fatalf("la posesion sin clave protegida no acredita el factor: %v", err)
	}
	verificador.fijarAsercion(asercion)

	evaluador.fijarResultado(resultadoGarantia(dominiovec.AuthAssuranceHigh))
	if _, err := servicio.Resolver(context.Background(), credencial); !errors.Is(err, ErrAsercionNoValida) {
		t.Fatalf("una declaracion alta debe denegarse: %v", err)
	}
	evaluador.fijarResultado(resultadoGarantia(dominiovec.AuthAssuranceSubstantial))
	identidad, err := servicio.Resolver(context.Background(), credencial)
	if err != nil {
		t.Fatalf("certificado personal protegido: %v", err)
	}
	cuenta, _, err := servicio.ProyectarCuentaAutenticada(context.Background(), identidad)
	if err != nil || cuenta.Metodo != dominiovec.AuthMethodCertificate || cuenta.Garantia != dominiovec.AuthAssuranceSubstantial {
		t.Fatalf("cuenta o garantia real incorrecta: %#v, %v", cuenta, err)
	}

	reloj.fijar(c.RetiradaPoliticaInternaEn)
	if _, _, err := servicio.ProyectarCuentaAutenticada(context.Background(), identidad); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("la sesion previa debe perder vigencia al retirar la politica: %v", err)
	}
	if _, err := servicio.Resolver(context.Background(), credencial); !errors.Is(err, ErrAsercionNoValida) {
		t.Fatalf("la politica retirada no admite nuevas aserciones: %v", err)
	}
	if _, err := NuevoServicioIdentidad(c, verificador, evaluador, nuevoRegistroMemoria(), reloj); !errors.Is(err, ErrConfiguracionSuperficie) {
		t.Fatalf("el arranque tras retirada debe fallar: %v", err)
	}
}
