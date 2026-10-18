package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestMotivoPoliticaOfertasUsaConcesionPublicadaExactaEnV11(t *testing.T) {
	consulta := dominiovec.ConcesionRol{
		Accion: puertosbolsa.AccionConsultarPoliticaOfertas, ModuloID: "bolsa",
		TipoRecurso: "bolsa_constituida", Finalidades: []string{puertosbolsa.FinalidadConsultarPoliticaOfertas},
		GarantiaMinima:   dominiovec.AuthAssuranceHigh,
		CamposPermitidos: []string{puertosbolsa.CampoConsultarPoliticaOfertas},
	}
	publicacion := dominiovec.ConcesionRol{
		Accion: puertosbolsa.AccionPublicarPoliticaOfertas, ModuloID: "bolsa",
		TipoRecurso: "bolsa_constituida", Finalidades: []string{puertosbolsa.FinalidadPoliticaOfertas},
		GarantiaMinima: dominiovec.AuthAssuranceHigh,
	}
	nueva := func(t *testing.T) *politicaBorradorLlamamientoBolsaDesarrollo {
		t.Helper()
		politica, autoridad, _ := politicaProvisionBolsaPrueba(t, 11)
		politica.publicada = true
		politica.instantanea = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(autoridad.leida.instantanea)
		return politica
	}
	validar := func(p *politicaBorradorLlamamientoBolsaDesarrollo, motivo dominiovec.ReferenciaEntradaCatalogo) error {
		return p.ValidarReferenciaMotivoAutorizacionV2(context.Background(), motivo,
			p.reloj.Ahora().UTC().Truncate(time.Microsecond))
	}
	t.Run("sin_concesion", func(t *testing.T) {
		p := nueva(t)
		if err := validar(p, motivoConsultarPoliticaOfertasBolsaDesarrollo()); !errors.Is(err, dominiovec.ErrSolicitudAutorizacionInvalida) {
			t.Fatalf("v11 sin concesión consultó: %v", err)
		}
	})
	t.Run("publicada_exacta", func(t *testing.T) {
		p := nueva(t)
		p.instantanea.VersionRol.Concesiones = append(p.instantanea.VersionRol.Concesiones, consulta, publicacion)
		if err := p.instantanea.Validar(); err != nil {
			t.Fatalf("instantánea publicada inválida: %v", err)
		}
		for _, motivo := range []dominiovec.ReferenciaEntradaCatalogo{
			motivoConsultarPoliticaOfertasBolsaDesarrollo(), motivoPublicarPoliticaOfertasBolsaDesarrollo(),
		} {
			if err := validar(p, motivo); err != nil {
				t.Fatalf("concesión exacta v11 rechazada: %v", err)
			}
		}
	})
	for nombre, mutar := range map[string]func(*dominiovec.ConcesionRol){
		"campo_extra":     func(c *dominiovec.ConcesionRol) { c.CamposPermitidos = append(c.CamposPermitidos, "persona_ref") },
		"finalidad_ajena": func(c *dominiovec.ConcesionRol) { c.Finalidades = []string{"finalidad:ajena"} },
		"recurso_ajeno":   func(c *dominiovec.ConcesionRol) { c.TipoRecurso = "participacion_bolsa" },
		"garantia_menor":  func(c *dominiovec.ConcesionRol) { c.GarantiaMinima = dominiovec.AuthAssuranceSubstantial },
	} {
		t.Run(nombre, func(t *testing.T) {
			p := nueva(t)
			c := consulta
			mutar(&c)
			p.instantanea.VersionRol.Concesiones = append(p.instantanea.VersionRol.Concesiones, c)
			if err := p.instantanea.Validar(); err != nil {
				t.Fatalf("prueba de concesión mal formada: %v", err)
			}
			if err := validar(p, motivoConsultarPoliticaOfertasBolsaDesarrollo()); !errors.Is(err, dominiovec.ErrSolicitudAutorizacionInvalida) {
				t.Fatalf("concesión divergente admitida: %v", err)
			}
		})
	}
	t.Run("publicacion_con_campo_de_lectura", func(t *testing.T) {
		p := nueva(t)
		c := publicacion
		c.CamposPermitidos = []string{puertosbolsa.CampoConsultarPoliticaOfertas}
		p.instantanea.VersionRol.Concesiones = append(p.instantanea.VersionRol.Concesiones, c)
		if err := validar(p, motivoPublicarPoliticaOfertasBolsaDesarrollo()); !errors.Is(err, dominiovec.ErrSolicitudAutorizacionInvalida) {
			t.Fatalf("publicación divergente admitida: %v", err)
		}
	})
}
