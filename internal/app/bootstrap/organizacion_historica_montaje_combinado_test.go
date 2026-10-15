package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/composicion/internactproveedores"
	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestOrganizacionHistoricaCombinadaSeleccionYFalloAislado(t *testing.T) {
	cfg := configuracionOrganizacionHistoricaPrueba()
	cfg.OrganizacionHistoricaMaterialDir = "/material/privado"
	for _, caso := range []struct {
		valor         string
		activo, falla bool
	}{
		{"", false, false}, {"false", false, false}, {"true", true, false}, {"TRUE", false, true},
	} {
		cfg.OrganizacionHistoricaConsultaEnabled = caso.valor
		activo, err := cfg.OrganizacionHistoricaConsultaDesarrolloActivo()
		if activo != caso.activo || (err != nil) != caso.falla {
			t.Fatalf("selector %q: activo=%v, fallo=%v", caso.valor, activo, err)
		}
	}
	cfg.OrganizacionHistoricaConsultaEnabled = "true"
	cfg.DevelopmentGuard = ""
	if activo, err := cfg.OrganizacionHistoricaConsultaDesarrolloActivo(); activo || err == nil {
		t.Fatal("consulta activa fuera de la doble llave")
	}
	base := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := protegerOrganizacionHistoricaNoDisponible(base)
	for _, caso := range []struct {
		metodo, ruta string
		estado       int
	}{
		{http.MethodGet, httpapi.RutaOrganizacionHistoricaPersonal, http.StatusServiceUnavailable},
		{http.MethodPost, httpapi.RutaOrganizacionHistoricaPersonal, http.StatusMethodNotAllowed},
		{http.MethodGet, "/api/vec/contratacion-temporal/seguimiento", http.StatusNoContent},
		{http.MethodGet, httpapi.RutaOrganizacionHistoricaPersonal + "/importar", http.StatusNoContent},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(caso.metodo, caso.ruta, nil))
		if w.Code != caso.estado {
			t.Fatalf("%s %s: %d", caso.metodo, caso.ruta, w.Code)
		}
	}
}

func TestOrganizacionHistoricaCombinadaRechazaOtraAudienciaYCapacidadDivergente(t *testing.T) {
	publicada := CapacidadPublicadaOrganizacionHistoricaV3{DescriptorCapacidadOrganizacionHistoricaV3: DescriptorCapacidadOrganizacionHistoricaV3Desarrollo(), ClaveID: "clave:oh"}
	m := internactproveedores.MaterialOrganizacionHistorica{}
	if err := cotejarCapacidadOrganizacionHistoricaCombinada(m, publicada, "", "", time.Now()); err == nil {
		t.Fatal("el material privado sin capacidad se aceptó")
	}
	publicada.Audiencia = "vec_personal.ficha_empleado.consultar.v1"
	if err := cotejarCapacidadOrganizacionHistoricaCombinada(m, publicada, "", "", time.Now()); err == nil {
		t.Fatal("audiencia ajena admitida")
	}
}

type revalidadorVinculoOrganizacionHistoricaPrueba struct {
	llamadas int
	falla    bool
}

func (r *revalidadorVinculoOrganizacionHistoricaPrueba) RevalidarVinculoCorporativoRRHHV1(_ context.Context, s vecports.SolicitudRevalidacionVinculoCorporativoRRHHV1) error {
	r.llamadas++
	if r.falla || s.Validar() != nil {
		return vecports.ErrVinculoCorporativoRRHHNoVigente
	}
	return nil
}

func TestOrganizacionHistoricaCombinadaCapturaUnaSesionYConservaOriginal(t *testing.T) {
	e := nuevaSesionConsultaPrueba(t)
	origen := origenEntregaPerfilFijoPrueba(t)
	if err := componerPerfilesFijosAltaCoberturaCTDesarrollo(e.soporte, e.principal, e.reloj.Ahora(), origen); err != nil {
		t.Fatal(err)
	}
	fijo := e.soporte.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, http.MethodGet)
	if fijo == nil || fijo.contexto.Resultado.Contexto.PerfilActivoRef == e.p.base.Contexto.PerfilActivoRef {
		t.Fatal("el escenario no separa perfiles CT y Organización histórica")
	}
	e.resolutor.base = fijo.contexto.Resultado
	a := fijo.contexto.Resultado.Contexto
	ambito := internagobierno.AmbitoOrganizacionHistorica{PerfilActivoRef: a.PerfilActivoRef, PerfilVersion: a.Instantanea.PerfilVersion, OrganismoRef: "organizacion:prueba", UnidadClave: "unidad:prueba"}
	cat, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{Clave: "oh", Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet,
		Ruta: httpapi.RutaOrganizacionHistoricaPersonal, PerfilesActivosRef: []string{a.PerfilActivoRef}, ClavePolitica: "oh", ClaveCapacidad: "oh"}})
	if err != nil {
		t.Fatal(err)
	}
	sesion := *e.p
	sesion.fronteras = cat
	sesion.organizacionHistoricaPorCuenta = map[string]seleccionPerfilOrganizacionHistorica{a.Instantanea.CuentaRef: {perfil: a.PerfilActivoRef, version: a.Instantanea.PerfilVersion}}
	seguridad, err := nuevaSeguridadComunDesarrollo(&sesion, e.reloj)
	if err != nil {
		t.Fatal(err)
	}
	corporativo := &revalidadorVinculoOrganizacionHistoricaPrueba{}
	f := &fuenteOrganizacionHistoricaCombinada{sesion: seguridad, corporativo: corporativo, reloj: e.reloj,
		contextos: map[string]internagobierno.AmbitoOrganizacionHistorica{a.Instantanea.CuentaRef: ambito}, fronteras: cat}
	ctx := e.contexto()
	capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	capacidad.ruta, capacidad.metodo = httpapi.RutaOrganizacionHistoricaPersonal, http.MethodGet
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	descriptor, _ := cat.resolver(http.MethodGet, httpapi.RutaOrganizacionHistoricaPersonal)
	ctx = context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{metodo: http.MethodGet, ruta: httpapi.RutaOrganizacionHistoricaPersonal,
		superficie: superficieInternaSeguridadComunDesarrollo, catalogo: cat, descriptor: descriptor})
	capturado, err := f.vincular(ctx)
	if err != nil {
		t.Fatal(err)
	}
	primero, org, unidad, err := f.ContextoVinculadoOrganizacionHistorica(capturado)
	if err != nil || org != ambito.OrganismoRef || unidad != ambito.UnidadClave || primero.Resultado.Validar() != nil ||
		len(e.registro.altas) != 1 || e.resolutor.llamadas != 1 || corporativo.llamadas != 1 {
		t.Fatal("la petición no conservó una única sesión y vínculo corporativo", err)
	}
	if _, err := f.vincular(capturado); err == nil {
		t.Fatal("segunda captura admitida")
	}
	if e.p.base.Contexto.PerfilActivoRef == primero.Resultado.Contexto.PerfilActivoRef {
		t.Fatal("la sesión devolvió el perfil CT")
	}
	cancelado, cancelar := context.WithCancel(capturado)
	cancelar()
	if _, _, _, err := f.ContextoOriginalOrganizacionHistoricaParaAuditoria(context.WithoutCancel(cancelado)); err != nil {
		t.Fatal("la auditoría perdió la captura original", err)
	}
	ahora := e.reloj.Ahora()
	e.reloj.ahora = ahora.Add(3 * time.Minute)
	if _, _, _, err := f.ContextoVinculadoOrganizacionHistorica(capturado); err == nil {
		t.Fatal("sesión caducada admitida")
	}
	if _, _, _, err := f.ContextoOriginalOrganizacionHistoricaParaAuditoria(capturado); err != nil {
		t.Fatal("captura histórica perdida", err)
	}
	e.reloj.ahora = ahora
	sesion.organizacionHistoricaPorCuenta = map[string]seleccionPerfilOrganizacionHistorica{
		"cta_ajena000000000000000000000000": {perfil: ambito.PerfilActivoRef, version: ambito.PerfilVersion},
	}
	if _, err := f.vincular(ctx); err == nil || len(e.registro.altas) != 1 {
		t.Fatal("una cuenta ajena seleccionó el perfil histórico")
	}
	sesion.organizacionHistoricaPorCuenta = map[string]seleccionPerfilOrganizacionHistorica{
		a.Instantanea.CuentaRef: {perfil: "prf_ajeno000000000000000000000000", version: ambito.PerfilVersion},
	}
	if _, err := f.vincular(ctx); err == nil || len(e.registro.altas) != 1 {
		t.Fatal("un perfil ajeno atravesó el catálogo exacto")
	}
	sesion.organizacionHistoricaPorCuenta = map[string]seleccionPerfilOrganizacionHistorica{
		a.Instantanea.CuentaRef: {perfil: ambito.PerfilActivoRef, version: ambito.PerfilVersion},
	}
	corporativo.falla = true
	if _, err := f.vincular(ctx); err == nil || corporativo.llamadas != 2 {
		t.Fatal("el vínculo corporativo retirado no denegó la petición")
	}
}
