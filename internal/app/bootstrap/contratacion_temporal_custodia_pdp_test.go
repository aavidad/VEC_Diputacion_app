package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type escenarioCustodiaPDP struct {
	firma   *firmaDocumentoCTDesarrollo
	soporte *soporteAltaContratacionTemporalDesarrollo
	perfil  *perfilFijoCTDesarrollo
	ctx     context.Context
	recurso core.RecursoAutorizable
}

func nuevoEscenarioCustodiaPDP(t *testing.T, conCustodia bool,
	alterar func(*politicaAutorizacionSolicitudLigadaV3Desarrollo)) escenarioCustodiaPDP {
	t.Helper()
	s, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := s.reloj.Ahora()
	p, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoFirmaCTDesarrollo,
		[]string{httpinterno.RutaFirmaDocumento, httpinterno.RutaConsultaFirmaDocumento},
		func(actor, ref string) (core.InstantaneaAutorizacion, error) {
			return instantaneaPerfilFijoFirmaDocumentoCTDesarrollo(actor, ref, ahora, conCustodia)
		})
	if err != nil || s.registrarPerfilFijoCTDesarrollo(p) != nil {
		t.Fatal(err)
	}
	p.contextoEsperadoRegistrado = p.contexto.Resultado
	p.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: p.contexto}
	s.motivoFirmaDocumento = motivoFirmaDocumentoCTDesarrollo()
	asignaciones := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	asignaciones.asignaciones = map[string]instantaneaPublicadaDesarrollo{p.perfilRef(): {
		instantanea: clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(p.plantilla), actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}}
	politica, err := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(s, s, s, s)
	if err != nil {
		t.Fatal(err)
	}
	if alterar != nil {
		alterar(&politica)
	}
	bolsa, err := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo("prf_bolsa_custodia_prueba")
	if err != nil {
		t.Fatal(err)
	}
	fronteras, err := nuevoCatalogoFronterasComunDesarrollo(append(bolsa, descriptoresFronterasContratacionTemporalDesarrollo(p.perfilRef(), []string{p.perfilRef()}, true)...))
	if err != nil {
		t.Fatal(err)
	}
	autBolsa, err := descriptoresAutorizacionBorradorLlamamientoBolsaDesarrollo(politica)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoAutorizacionComunDesarrollo(fronteras, append(autBolsa, descriptoresAutorizacionContratacionTemporalDesarrollo(politica, false, true)...))
	if err != nil {
		t.Fatal(err)
	}
	// La custodia no se liga a consultar firmas ni a otra frontera del catálogo.
	if _, ok := catalogo.politicaPara(docports.AccionCustodiarFirmado, "ct-firmas-documento-consultar", clavePoliticaContratacionTemporalDesarrollo, ctports.AccionConsultarFirmasDocumento); ok {
		t.Fatal("custodia ligada a lectura")
	}
	comun, err := nuevoAutorizadorComunDesarrollo(catalogo, s.reloj, seguridad.GeneradorReferenciasCriptograficas{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaFirmaDocumento)
	frontera, ok := fronteras.resolver(http.MethodPost, httpinterno.RutaFirmaDocumento)
	if !ok {
		t.Fatal("sin frontera firma")
	}
	ctx = context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: http.MethodPost, ruta: httpinterno.RutaFirmaDocumento, superficie: superficieInternaSeguridadComunDesarrollo, catalogo: fronteras, descriptor: frontera})
	esperado := esperadoCustodiaPrueba()
	preimagen := preimagenCustodiaPrueba(esperado, nil)
	suma := sha256.Sum256(preimagen)
	esperado.fijar(hex.EncodeToString(suma[:]), "")
	ctx = context.WithValue(ctx, claveCustodiaFirmadoCTDesarrollo{}, esperado)
	recurso, err := docports.RecursoV3(docports.AccionCustodiarFirmado, esperado.documentoRef, preimagen)
	if err != nil {
		t.Fatal(err)
	}
	return escenarioCustodiaPDP{firma: &firmaDocumentoCTDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: s, autorizador: comun}}, soporte: s, perfil: p, ctx: ctx, recurso: recurso}
}

func TestCustodiaPDPComunConBolsaEvaluaCamposDelPerfilFijo(t *testing.T) {
	for _, custodia := range []bool{false, true} {
		e := nuevoEscenarioCustodiaPDP(t, custodia, nil)
		pedida, err := e.firma.solicitarCustodiaV3(e.ctx, e.recurso)
		if !custodia {
			if !errors.Is(err, errCustodiaFirmadoCTDenegada) {
				t.Fatalf("perfil sin custodia: %v", err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		restricciones, err := pedida.decision.RestriccionesProyeccionPara(pedida.solicitud)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(restricciones.CamposPermitidos, []string{"documento_firmado.custodia", "evidencia_custodia"}) {
			t.Fatalf("campos derivados por PDP real: %v", restricciones.CamposPermitidos)
		}
		if pedida.actor.PerfilActivoRef != e.perfil.perfilRef() {
			t.Fatal("custodia usó otro perfil")
		}
		a := e.soporte.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
		if a.preparadas != 0 || a.publicadas != 0 {
			t.Fatal("PDP publicó permisos por petición")
		}
	}
}

type fuenteCustodiaPDPFallida struct {
	error
	antes func()
}

func (f fuenteCustodiaPDPFallida) ObtenerInstantaneaAutorizacion(context.Context, string, string) (core.InstantaneaAutorizacion, error) {
	if f.antes != nil {
		f.antes()
	}
	return core.InstantaneaAutorizacion{}, f.error
}

type registroCustodiaPDPFallido struct{ error }

func (r registroCustodiaPDPFallido) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Context, vecports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	return time.Time{}, r.error
}

func TestCustodiaPDPComunClasificaInfraestructuraYRevocacion(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		alterar func(*politicaAutorizacionSolicitudLigadaV3Desarrollo)
	}{
		{"fuente", func(p *politicaAutorizacionSolicitudLigadaV3Desarrollo) {
			p.fuente = fuenteCustodiaPDPFallida{error: vecports.ErrFuenteAutorizacionNoDisponible}
		}},
		{"registro", func(p *politicaAutorizacionSolicitudLigadaV3Desarrollo) {
			p.registroConcesiones = registroCustodiaPDPFallido{vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible}
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioCustodiaPDP(t, true, caso.alterar)
			_, err := e.firma.solicitarCustodiaV3(e.ctx, e.recurso)
			if !errors.Is(err, docports.ErrCapacidadNoDisponible) || errors.Is(err, errCustodiaFirmadoCTDenegada) {
				t.Fatalf("infraestructura clasificada como denegación: %v", err)
			}
		})
	}
	e := nuevoEscenarioCustodiaPDP(t, true, nil)
	a := e.soporte.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	publicada := a.asignaciones[e.perfil.perfilRef()]
	ahora := e.soporte.reloj.Ahora()
	publicada.instantanea.AsignacionPerfil.Estado = core.EstadoAsignacionPerfilRevocada
	publicada.instantanea.AsignacionPerfil.RevocadaEn, publicada.instantanea.AsignacionPerfil.RevocadaPor, publicada.instantanea.AsignacionPerfil.RevocacionRef = ahora, "revocador:prueba", "revocacion:prueba"
	a.asignaciones[e.perfil.perfilRef()] = publicada
	if _, err := e.firma.solicitarCustodiaV3(e.ctx, e.recurso); !errors.Is(err, errCustodiaFirmadoCTDenegada) {
		t.Fatalf("revocación: %v", err)
	}
	e.soporte.autoridadAsignaciones = fuentePerfilFirmaCaidaPrueba{a}
	if _, err := e.firma.solicitarCustodiaV3(e.ctx, e.recurso); !errors.Is(err, docports.ErrCapacidadNoDisponible) {
		t.Fatalf("fuente de asignaciones caída: %v", err)
	}
}

func TestCustodiaPDPComunFuenteCaidaConRevocacionConfirmadaDeniega(t *testing.T) {
	var e escenarioCustodiaPDP
	e = nuevoEscenarioCustodiaPDP(t, true, func(p *politicaAutorizacionSolicitudLigadaV3Desarrollo) {
		p.fuente = fuenteCustodiaPDPFallida{error: vecports.ErrFuenteAutorizacionNoDisponible, antes: func() {
			a := e.soporte.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
			publicada := a.asignaciones[e.perfil.perfilRef()]
			ahora := e.soporte.reloj.Ahora()
			publicada.instantanea.AsignacionPerfil.Estado = core.EstadoAsignacionPerfilRevocada
			publicada.instantanea.AsignacionPerfil.RevocadaEn, publicada.instantanea.AsignacionPerfil.RevocadaPor, publicada.instantanea.AsignacionPerfil.RevocacionRef = ahora, "revocador:prueba", "revocacion:prueba"
			a.asignaciones[e.perfil.perfilRef()] = publicada
		}}
	})
	if _, err := e.firma.solicitarCustodiaV3(e.ctx, e.recurso); !errors.Is(err, errCustodiaFirmadoCTDenegada) || errors.Is(err, docports.ErrCapacidadNoDisponible) {
		t.Fatalf("revocación concurrente confirmada: %v", err)
	}
}
