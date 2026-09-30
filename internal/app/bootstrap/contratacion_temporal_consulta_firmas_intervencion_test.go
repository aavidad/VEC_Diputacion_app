package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	consultafirmas "vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmas"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestLectorFirmasIntervencionPerfilSoloLecturaOrganizacion(t *testing.T) {
	principal := dominiovec.Principal{ID: "desarrollo:intervencion-lector-firmas", Roles: []string{rolIntervencionContratacionTemporalDesarrollo},
		AuthMethod: dominiovec.AuthMethodCertificate, AuthAssurance: dominiovec.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa,
			"perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": strings.Repeat("a", 64)}}
	ahora := time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC)
	base, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	perfil, err := nuevoPerfilFijoCTDesarrollo(principal, base, ahora, clavePerfilFijoLectorFirmasIntervencionCT,
		[]string{httpinterno.RutaResultadosFiscalizacion},
		func(actor, ref string) (dominiovec.InstantaneaAutorizacion, error) {
			return nuevaInstantaneaLectorFirmasIntervencionCTDesarrollo(actor, ref, ahora)
		})
	if err != nil {
		t.Fatal(err)
	}
	baseVinculo, _ := base.Vinculo.Datos()
	lectorVinculo, _ := perfil.contexto.Vinculo.Datos()
	if perfil.perfilRef() == baseVinculo.PerfilActivoRef || lectorVinculo.SesionRef == baseVinculo.SesionRef ||
		perfil.contexto.Resultado.Contexto.PersonaRef != base.Resultado.Contexto.PersonaRef ||
		perfil.contexto.Resultado.Contexto.Instantanea.CuentaRef != base.Resultado.Contexto.Instantanea.CuentaRef {
		t.Fatal("el perfil lector no mantiene identidad y sesión separadas")
	}
	concesiones := perfil.plantilla.VersionRol.Concesiones
	if len(concesiones) != 1 || concesiones[0].Accion != ports.AccionConsultarFirmasDocumento ||
		concesiones[0].TipoRecurso != ports.TipoRecursoConsultaFirmasDocumento ||
		!slices.Equal(concesiones[0].CamposPermitidos, consultafirmas.CamposConsultaFirmasDocumento()) ||
		len(perfil.plantilla.AsignacionPerfil.Ambitos) != 1 ||
		perfil.plantilla.AsignacionPerfil.Ambitos[0].Clave != "organizacion_ref" ||
		!slices.Equal(perfil.plantilla.AsignacionPerfil.Ambitos[0].Valores, []string{organizacionAltaContratacionTemporalDesarrollo}) {
		t.Fatal("el perfil lector concede más que la consulta nominal org-only")
	}
}

func TestLectorFirmasIntervencionNoUsaOtroCanal(t *testing.T) {
	p := dominiovec.Principal{ID: "desarrollo:intervencion-lector-firmas", Roles: []string{rolIntervencionContratacionTemporalDesarrollo},
		AuthMethod: dominiovec.AuthMethodCertificate, AuthAssurance: dominiovec.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa,
			"perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": strings.Repeat("a", 64)}}
	sello := &selloConsultasContratacionTemporalDesarrollo{}
	canal := &soporteFiscalizacionContratacionTemporalDesarrollo{
		sello: sello, principalID: p.ID, certificadoSHA256: p.Attributes["certificate_sha256"],
	}
	lector := &lectorFirmasIntervencionCTDesarrollo{canal: canal}
	capacidad := capacidadConsultaContratacionTemporalDesarrollo{
		sello: sello, ruta: httpinterno.RutaResultadosFiscalizacion, metodo: http.MethodPost, principal: p,
	}
	ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	if !lector.capacidadValida(ctx) {
		t.Fatal("se rechazó el canal propio de Intervención")
	}
	for _, alterar := range []func(*capacidadConsultaContratacionTemporalDesarrollo){
		func(c *capacidadConsultaContratacionTemporalDesarrollo) { c.metodo = http.MethodGet },
		func(c *capacidadConsultaContratacionTemporalDesarrollo) {
			c.ruta = httpinterno.RutaConsultaFirmaDocumento
		},
		func(c *capacidadConsultaContratacionTemporalDesarrollo) {
			c.principal.Roles = []string{rolTecnicoRRHHContratacionTemporalDesarrollo}
		},
		func(c *capacidadConsultaContratacionTemporalDesarrollo) {
			c.sello = &selloConsultasContratacionTemporalDesarrollo{}
		},
	} {
		copia := capacidad
		alterar(&copia)
		if lector.capacidadValida(context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, copia)) {
			t.Fatal("se aceptó otro canal, método, rol o sello")
		}
	}
	if _, err := lector.ConsultarFirmas(ctx, organizacionAltaContratacionTemporalDesarrollo, "expediente:prueba"); !errors.Is(err, ports.ErrRegistroFirmaDocumentoNoDisponible) {
		t.Fatalf("dependencia ausente debe ser indisponibilidad, recibió %v", err)
	}
}
