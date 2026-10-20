package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

type consultaPersonalEnProcesoPrueba struct {
	llamadas int
	actor    string
	perfil   string
	contexto context.Context
}

func (c *consultaPersonalEnProcesoPrueba) Consultar(ctx context.Context, s personal.SolicitudConsultaOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error) {
	c.llamadas++
	c.actor = s.Actor.Instantanea.CuentaRef
	c.perfil = s.Actor.PerfilActivoRef
	c.contexto = ctx
	return personalports.ResultadoConsultaOrganizacionHistorica{}, nil
}

func escenarioPersonalEnProcesoPrueba(t *testing.T) (*montajeOrganizacionHistoricaCombinada, *puenteBolsaLlamamientoDesarrollo, context.Context, preparacionLlamamientoDesarrollo, *consultaPersonalEnProcesoPrueba) {
	t.Helper()
	e := nuevaSesionConsultaPrueba(t)
	if err := componerPerfilesFijosAltaCoberturaCTDesarrollo(e.soporte, e.principal, e.reloj.Ahora(), origenEntregaPerfilFijoPrueba(t)); err != nil {
		t.Fatal(err)
	}
	fijo := e.soporte.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, http.MethodGet)
	if fijo == nil || fijo.contexto.Resultado.Contexto.PerfilActivoRef == e.p.base.Contexto.PerfilActivoRef {
		t.Fatal("el perfil Personal no quedó separado de CT")
	}
	e.resolutor.base = fijo.contexto.Resultado
	e.soporte.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: e.soporte.contexto}
	a := fijo.contexto.Resultado.Contexto
	ambito := internagobierno.AmbitoOrganizacionHistorica{PerfilActivoRef: a.PerfilActivoRef, PerfilVersion: a.Instantanea.PerfilVersion,
		OrganismoRef: "organizacion:prueba", UnidadClave: "unidad:prueba"}
	cat, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{Clave: fronteraPersonalDesdeCT,
		Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: httpct.RutaSeleccionLlamamiento,
		PerfilesActivosRef: []string{a.PerfilActivoRef}, ClavePolitica: "oh", ClaveCapacidad: "oh"}})
	if err != nil {
		t.Fatal(err)
	}
	publico, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{Clave: "seleccion-ct-prueba",
		Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: httpct.RutaSeleccionLlamamiento,
		PerfilesActivosRef: []string{e.p.base.Contexto.PerfilActivoRef}, ClavePolitica: "ct", ClaveCapacidad: "ct"}})
	if err != nil {
		t.Fatal(err)
	}
	contextoBolsa, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(
		e.principal, e.reloj.Ahora(), discriminadorContextoSinteticoBorradorBolsaDesarrollo())
	if err != nil {
		t.Fatal(err)
	}
	bolsa := &soporteAltaContratacionTemporalDesarrollo{sello: e.soporte.sello, principalID: e.soporte.principalID,
		certificadoSHA256: e.soporte.certificadoSHA256, contexto: contextoBolsa, reloj: e.soporte.reloj}
	if contextoBolsa.Resultado.Contexto.Instantanea.CuentaRef != e.soporte.contexto.Resultado.Contexto.Instantanea.CuentaRef ||
		contextoBolsa.Resultado.Contexto.PerfilActivoRef == e.soporte.contexto.Resultado.Contexto.PerfilActivoRef {
		t.Fatal("el actor Bolsa de prueba no comparte cuenta con CT o no tiene perfil propio")
	}
	privado, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{Clave: fronteraSeleccionBolsaNativaCT,
		Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: httpct.RutaSeleccionLlamamiento,
		PerfilesActivosRef: []string{contextoBolsa.Resultado.Contexto.PerfilActivoRef}, ClavePolitica: "bolsa", ClaveCapacidad: "bolsa"}})
	if err != nil {
		t.Fatal(err)
	}
	corporativo := &revalidadorVinculoOrganizacionHistoricaPrueba{}
	e.p.fijarVinculoCorporativo(corporativo, nil)
	sesion, err := nuevaSesionOrganizacionHistoricaCombinada(e.p, cat, map[string]internagobierno.AmbitoOrganizacionHistorica{a.Instantanea.CuentaRef: ambito})
	if err != nil {
		t.Fatal(err)
	}
	seguridad, err := nuevaSeguridadComunDesarrollo(sesion, e.reloj)
	if err != nil {
		t.Fatal(err)
	}
	fuente := &fuenteOrganizacionHistoricaCombinada{sesion: sesionesOrganizacionHistoricaPorHoja{e.soporte.certificadoSHA256: seguridad}, corporativo: corporativo, reloj: e.reloj,
		contextos: map[string]internagobierno.AmbitoOrganizacionHistorica{a.Instantanea.CuentaRef: ambito}, fronteras: cat}
	consulta := &consultaPersonalEnProcesoPrueba{}
	m := &montajeOrganizacionHistoricaCombinada{fuente: fuente, consulta: consulta, fronteras: cat, barrera: &barreraPeticionesOrganizacionHistorica{}}
	puente, _, d, _ := puenteBolsaPrueba(t)
	puente.alta.soporte = e.soporte
	puente.reloj = e.reloj
	puente.operacionNativa = &operacionBolsaNativaCT{ct: e.soporte,
		bolsa: &soporteSesionBorradorBolsaDesarrollo{soporteCanal: bolsa},
		sesion: &seguridadComunDesarrollo{sesion: &sesionOperacionBolsaCTPrueba{actual: contextoSeguridadComunDesarrollo{
			Vinculo: contextoBolsa.Vinculo, Resultado: contextoBolsa.Resultado}}, reloj: e.reloj},
		publico: publico, privado: privado}
	ctx := e.contexto()
	capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	capacidad.ruta = httpct.RutaSeleccionLlamamiento
	capacidad.metodo = http.MethodPost
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	frontera := fronteraSeguridadComunDesarrollo{metodo: http.MethodPost, ruta: httpct.RutaSeleccionLlamamiento,
		superficie: superficieInternaSeguridadComunDesarrollo, catalogo: publico}
	ctx = context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, frontera)
	estado := &estadoOrigenConstituidoCT{organizacion: d.expediente.Fiscalizado.OrganizacionRef,
		expediente: d.expediente.Fiscalizado.Referencia, clave: d.clave, version: d.expediente.Fiscalizado.Version}
	ctx = context.WithValue(ctx, claveEstadoOrigenConstituidoCT{}, estado)
	captura, err := puente.capturaOperacionNativa(ctx)
	if err != nil {
		t.Fatalf("captura CT/Bolsa en la selección original: %v", err)
	}
	if _, err := puente.capturaOperacionNativaDesdeHijo(captura.ctx, d); err != nil {
		t.Fatalf("captura Bolsa hija: %v", err)
	}
	return m, puente, ctx, d, consulta
}

func TestPersonalEnProcesoUsaPerfilPropioYConservaSeleccionCT(t *testing.T) {
	m, puente, ctx, d, espia := escenarioPersonalEnProcesoPrueba(t)
	antes, _ := fronteraSeguridadComunDesdeContexto(ctx)
	cliente, organismo, err := m.consultaParaCT(ctx, puente, d)
	if err != nil || organismo != "organizacion:prueba" {
		t.Fatalf("consulta CT: organismo=%q, error=%v", organismo, err)
	}
	if _, err := cliente.Consultar(ctx, personal.SelectorOrganizacionHistorica{}); err != nil {
		t.Fatal(err)
	}
	despues, _ := fronteraSeguridadComunDesdeContexto(ctx)
	if espia.llamadas != 1 || espia.actor != puente.alta.soporte.contexto.Resultado.Contexto.Instantanea.CuentaRef ||
		espia.perfil == puente.alta.soporte.contexto.Resultado.Contexto.PerfilActivoRef || espia.contexto == ctx ||
		antes.ruta != httpct.RutaSeleccionLlamamiento || despues.ruta != antes.ruta || despues.metodo != http.MethodPost ||
		!despues.catalogo.mismaInstancia(antes.catalogo) || ctx.Value(claveContextoOrganizacionHistoricaCombinada{}) != nil {
		t.Fatal("Personal no conservó la identidad o modificó la petición CT original")
	}
	fronteraHija, ok := fronteraSeguridadComunDesdeContexto(espia.contexto)
	if !ok || fronteraHija.ruta != httpct.RutaSeleccionLlamamiento || fronteraHija.metodo != http.MethodPost || fronteraHija.descriptor.Clave != fronteraPersonalDesdeCT {
		t.Fatal("la consulta perdió la ruta POST original de selección")
	}
	if _, err := cliente.Consultar(context.WithValue(ctx, struct{}{}, true), personal.SelectorOrganizacionHistorica{}); !errors.Is(err, personal.ErrOrganizacionHistoricaNoDisponible) {
		t.Fatal("la consulta se usó desde otra petición", err)
	}
	m.barrera.detener()
	if _, err := cliente.Consultar(ctx, personal.SelectorOrganizacionHistorica{}); !errors.Is(err, personal.ErrOrganizacionHistoricaNoDisponible) || espia.llamadas != 1 {
		t.Fatal("la consulta sobrevivió al cierre", err)
	}
}

func TestPersonalEnProcesoDeniegaCapturaAjenaSinEstadoYOtraRuta(t *testing.T) {
	m, puente, ctx, d, espia := escenarioPersonalEnProcesoPrueba(t)
	estado := ctx.Value(claveEstadoOrigenConstituidoCT{}).(*estadoOrigenConstituidoCT)
	ajeno := &estadoOrigenConstituidoCT{organizacion: estado.organizacion, expediente: estado.expediente, clave: estado.clave, version: estado.version,
		sesion: capturaOperacionNativaCT{ctx: estado.sesion.ctx, bolsa: estado.sesion.bolsa}}
	ajeno.unaSesion.Do(func() {})
	for _, caso := range []struct {
		name        string
		contexto    context.Context
		preparacion preparacionLlamamientoDesarrollo
	}{
		{"sin_estado", context.WithValue(ctx, claveEstadoOrigenConstituidoCT{}, (*estadoOrigenConstituidoCT)(nil)), d},
		{"captura_ajena", context.WithValue(ctx, claveEstadoOrigenConstituidoCT{}, ajeno), d},
		{"otro_expediente", ctx, func() preparacionLlamamientoDesarrollo { otro := d; otro.clave = d.clave + "-otra"; return otro }()},
		{"otra_ruta", context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{metodo: http.MethodGet, ruta: httpapi.RutaOrganizacionHistoricaPersonal, superficie: superficieInternaSeguridadComunDesarrollo, catalogo: puente.operacionNativa.privado}), d},
	} {
		t.Run(caso.name, func(t *testing.T) {
			if cliente, _, err := m.consultaParaCT(caso.contexto, puente, caso.preparacion); cliente != nil || err == nil {
				t.Fatalf("captura ajena admitida: %v", err)
			}
		})
	}
	propia := m.fuente.sesion
	m.fuente.sesion = sesionesOrganizacionHistoricaPorHoja{"certificado-de-otro-actor": propia.(sesionesOrganizacionHistoricaPorHoja)[puente.alta.soporte.certificadoSHA256]}
	if cliente, _, err := m.consultaParaCT(ctx, puente, d); cliente != nil || err == nil {
		t.Fatalf("la sesión de otra hoja se usó para la selección CT: %v", err)
	}
	m.fuente.sesion = propia
	if estado.sesion.ctx == ctx || estado.sesion.ctx == nil || espia.llamadas != 0 {
		t.Fatal("un rechazo alteró la captura original")
	}
}
