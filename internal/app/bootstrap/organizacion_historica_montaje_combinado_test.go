package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/composicion/internactproveedores"
	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
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

func TestOrganizacionHistoricaPreflightCorporativoAusenteNoImpideSesionCT(t *testing.T) {
	e := nuevaSesionConsultaPrueba(t)
	e.p.fijarVinculoCorporativo(nil, errors.New("preflight corporativo ausente"))
	if !errors.Is(e.p.falloVinculoCorporativo, vecports.ErrVinculoCorporativoRRHHNoDisponible) || e.p.vinculoCorporativo != nil {
		t.Fatal("el fallo corporativo no quedó acotado a Organización histórica")
	}
	if c, err := e.p.ResolverContexto(e.contexto()); err != nil || c.Resultado.Validar() != nil {
		t.Fatal("la sesión registrada de CT dejó de funcionar", err)
	}
	cfg := configuracionOrganizacionHistoricaPrueba()
	cfg.OrganizacionHistoricaConsultaEnabled = "true"
	cfg.OrganizacionHistoricaMaterialDir = "/material/privado"
	autoridad := &autoridadConsultasContratacionTemporalDesarrollo{
		sesionOrganizacionHistorica:            e.p,
		proveedorMaterialOrganizacionHistorica: &proveedorMaterialAltaContratacionTemporalDesarrollo{},
		materialOrganizacionHistorica:          CapacidadPublicadaOrganizacionHistoricaV3{ClaveID: "clave:oh"},
	}
	montaje, err := nuevoMontajeOrganizacionHistoricaCombinada(context.Background(), cfg, autoridad, &registradorFronteraSuperficiePrueba{})
	if montaje != nil || !errors.Is(err, vecports.ErrVinculoCorporativoRRHHNoDisponible) {
		t.Fatal("Organización histórica se montó sin revalidación corporativa", err)
	}
}

type revalidadorVinculoOrganizacionHistoricaPrueba struct {
	llamadas     int
	falla        bool
	indisponible bool
}

func (r *revalidadorVinculoOrganizacionHistoricaPrueba) RevalidarVinculoCorporativoRRHHV1(_ context.Context, s vecports.SolicitudRevalidacionVinculoCorporativoRRHHV1) error {
	r.llamadas++
	if r.indisponible {
		return vecports.ErrVinculoCorporativoRRHHNoDisponible
	}
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
	corporativo := &revalidadorVinculoOrganizacionHistoricaPrueba{}
	e.p.fijarVinculoCorporativo(corporativo, nil)
	ctCatalogo, ctPerfil := e.p.fronteras, e.p.base.Contexto.PerfilActivoRef
	sesion, err := nuevaSesionOrganizacionHistoricaCombinada(e.p, cat,
		map[string]internagobierno.AmbitoOrganizacionHistorica{a.Instantanea.CuentaRef: ambito})
	if err != nil || sesion == e.p || sesion.clave == e.p.clave || e.p.organizacionHistoricaPorCuenta != nil ||
		!e.p.fronteras.mismaInstancia(ctCatalogo) || e.p.base.Contexto.PerfilActivoRef != ctPerfil || sesion.fronteras.mismaInstancia(e.p.fronteras) {
		t.Fatal("la sesión OH no quedó separada de CT", err)
	}
	seguridad, err := nuevaSeguridadComunDesarrollo(sesion, e.reloj)
	if err != nil {
		t.Fatal(err)
	}
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
	e.resolutor.base = e.p.base
	ctOriginal, err := e.p.ResolverContexto(e.contexto())
	if err != nil || ctOriginal.Resultado.Contexto.PerfilActivoRef != ctPerfil ||
		sesion.fronteras.mismaInstancia(e.p.fronteras) {
		t.Fatal("la selección OH alteró la sesión y el perfil CT", err)
	}
}

type consultaOrganizacionHistoricaCombinadaEspia struct{ llamadas int }

func (c *consultaOrganizacionHistoricaCombinadaEspia) Consultar(context.Context, personal.SolicitudConsultaOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error) {
	c.llamadas++
	return personalports.ResultadoConsultaOrganizacionHistorica{}, nil
}

type auditorOrganizacionHistoricaCombinadaEspia struct{ llamadas int }

func (a *auditorOrganizacionHistoricaCombinadaEspia) RegistrarDenegacionOrganizacionHistorica(context.Context, httpapi.DenegacionOrganizacionHistorica) error {
	a.llamadas++
	return nil
}

type sesionOrganizacionHistoricaNoDisponiblePrueba struct{}

func (sesionOrganizacionHistoricaNoDisponiblePrueba) ResolverContexto(context.Context) (contextoSeguridadComunDesarrollo, error) {
	return contextoSeguridadComunDesarrollo{}, ct.ErrConsultaRRHHNoDisponible
}

func TestOrganizacionHistoricaSesionNoDisponibleConserva503(t *testing.T) {
	cat, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{Clave: "oh", Superficie: superficieInternaSeguridadComunDesarrollo,
		Metodo: http.MethodGet, Ruta: httpapi.RutaOrganizacionHistoricaPersonal, PerfilesActivosRef: []string{"prf_0123456789abcdefghijkl"}, ClavePolitica: "oh", ClaveCapacidad: "oh"}})
	if err != nil {
		t.Fatal(err)
	}
	reloj := relojContratacionTemporalDesarrollo{}
	seguridad, err := nuevaSeguridadComunDesarrollo(sesionOrganizacionHistoricaNoDisponiblePrueba{}, reloj)
	if err != nil {
		t.Fatal(err)
	}
	f := &fuenteOrganizacionHistoricaCombinada{sesion: seguridad, corporativo: &revalidadorVinculoOrganizacionHistoricaPrueba{}, reloj: reloj, fronteras: cat}
	descriptor, _ := cat.resolver(http.MethodGet, httpapi.RutaOrganizacionHistoricaPersonal)
	ctx := context.WithValue(context.Background(), claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{metodo: http.MethodGet,
		ruta: httpapi.RutaOrganizacionHistoricaPersonal, superficie: superficieInternaSeguridadComunDesarrollo, catalogo: cat, descriptor: descriptor})
	if _, err := f.vincular(ctx); !errors.Is(err, ct.ErrConsultaRRHHNoDisponible) {
		t.Fatal("sesión caída clasificada como denegación", err)
	}
}

func TestOrganizacionHistoricaCorporativoNoDisponibleDa503ConCapturaOriginal(t *testing.T) {
	e := nuevaSesionConsultaPrueba(t)
	actor := e.p.base.Contexto
	ambito := internagobierno.AmbitoOrganizacionHistorica{PerfilActivoRef: actor.PerfilActivoRef, PerfilVersion: actor.Instantanea.PerfilVersion,
		OrganismoRef: "organizacion:prueba", UnidadClave: "unidad:prueba"}
	cat, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{Clave: "oh", Superficie: superficieInternaSeguridadComunDesarrollo,
		Metodo: http.MethodGet, Ruta: httpapi.RutaOrganizacionHistoricaPersonal, PerfilesActivosRef: []string{actor.PerfilActivoRef}, ClavePolitica: "oh", ClaveCapacidad: "oh"}})
	if err != nil {
		t.Fatal(err)
	}
	corporativo := &revalidadorVinculoOrganizacionHistoricaPrueba{indisponible: true}
	e.p.fijarVinculoCorporativo(corporativo, nil)
	sesion, err := nuevaSesionOrganizacionHistoricaCombinada(e.p, cat,
		map[string]internagobierno.AmbitoOrganizacionHistorica{actor.Instantanea.CuentaRef: ambito})
	if err != nil {
		t.Fatal(err)
	}
	seguridad, err := nuevaSeguridadComunDesarrollo(sesion, e.reloj)
	if err != nil {
		t.Fatal(err)
	}
	f := &fuenteOrganizacionHistoricaCombinada{sesion: seguridad, corporativo: corporativo, reloj: e.reloj,
		contextos: map[string]internagobierno.AmbitoOrganizacionHistorica{actor.Instantanea.CuentaRef: ambito}, fronteras: cat}
	ctx := e.contexto()
	capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	capacidad.ruta, capacidad.metodo = httpapi.RutaOrganizacionHistoricaPersonal, http.MethodGet
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	descriptor, _ := cat.resolver(http.MethodGet, httpapi.RutaOrganizacionHistoricaPersonal)
	ctx = context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{metodo: http.MethodGet,
		ruta: httpapi.RutaOrganizacionHistoricaPersonal, superficie: superficieInternaSeguridadComunDesarrollo, catalogo: cat, descriptor: descriptor})
	consulta, auditor := &consultaOrganizacionHistoricaCombinadaEspia{}, &auditorOrganizacionHistoricaCombinadaEspia{}
	h, err := httpapi.NewHandlerOrganizacionHistoricaPersonal(internactproveedores.AutoridadContextoOrganizacionHistorica{Fuente: f}, consulta, auditor)
	if err != nil {
		t.Fatal(err)
	}
	var contextoCapturado context.Context
	envuelto := capturarOrganizacionHistoricaCombinada(f, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextoCapturado = r.Context()
		h.ServeHTTP(w, r)
	}))
	ruta := httpapi.RutaOrganizacionHistoricaPersonal + "?vigente_en=2026-09-25&conocido_en=2026-09-25T00:00:00.000000Z"
	w := httptest.NewRecorder()
	envuelto.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil).WithContext(ctx))
	if w.Code != http.StatusServiceUnavailable || consulta.llamadas != 0 || auditor.llamadas != 0 || !strings.Contains(w.Body.String(), `"codigo":"servicio_no_disponible"`) || strings.Contains(w.Body.String(), `"data"`) {
		t.Fatalf("fallo corporativo respondió %d, consulta=%d, auditoría=%d: %s", w.Code, consulta.llamadas, auditor.llamadas, w.Body.String())
	}
	if original, _, _, err := f.ContextoOriginalOrganizacionHistoricaParaAuditoria(contextoCapturado); err != nil || original.Resultado.Contexto.PersonaRef != actor.PersonaRef {
		t.Fatal("la auditoría perdió el actor original acreditado", err)
	}
	if _, _, _, err := f.ContextoVinculadoOrganizacionHistorica(contextoCapturado); !errors.Is(err, vecports.ErrVinculoCorporativoRRHHNoDisponible) {
		t.Fatal("el error de disponibilidad autorizó lectura o cambió a denegación", err)
	}
}

func TestOrganizacionHistoricaClavePropiaSeBorraEnFalloYCierreTrasDrenaje(t *testing.T) {
	e := nuevaSesionConsultaPrueba(t)
	actor := e.p.base.Contexto
	e.p.fijarVinculoCorporativo(&revalidadorVinculoOrganizacionHistoricaPrueba{}, nil)
	cat, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{Clave: "oh", Superficie: superficieInternaSeguridadComunDesarrollo,
		Metodo: http.MethodGet, Ruta: httpapi.RutaOrganizacionHistoricaPersonal, PerfilesActivosRef: []string{actor.PerfilActivoRef}, ClavePolitica: "oh", ClaveCapacidad: "oh"}})
	if err != nil {
		t.Fatal(err)
	}
	contextos := map[string]internagobierno.AmbitoOrganizacionHistorica{actor.Instantanea.CuentaRef: {
		PerfilActivoRef: actor.PerfilActivoRef, PerfilVersion: actor.Instantanea.PerfilVersion, OrganismoRef: "organizacion:prueba", UnidadClave: "unidad:prueba",
	}}
	claveCT := e.p.clave
	fallida, err := nuevaSesionOrganizacionHistoricaCombinada(e.p, cat, contextos)
	if err != nil {
		t.Fatal(err)
	}
	propiedadFallida := &propiedadClaveSesionOrganizacionHistorica{sesion: fallida}
	propiedadFallida.abandonar() // simula cualquier error posterior del montaje
	var cero [32]byte
	if fallida.clave != cero || e.p.clave != claveCT || claveCT == cero {
		t.Fatal("el fallo posterior retuvo la clave OH o borró la de CT")
	}
	sesion, err := nuevaSesionOrganizacionHistoricaCombinada(e.p, cat, contextos)
	if err != nil {
		t.Fatal(err)
	}
	propiedad := &propiedadClaveSesionOrganizacionHistorica{sesion: sesion, entregada: true}
	barrera := &barreraPeticionesOrganizacionHistorica{}
	entro, liberar, termino, cerrado := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	h := barrera.proteger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entro)
		<-liberar
		w.WriteHeader(http.StatusNoContent)
	}))
	go func() {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, httpapi.RutaOrganizacionHistoricaPersonal, nil))
		close(termino)
	}()
	<-entro
	cerrar := cierreOrganizacionHistoricaCombinada(barrera, func() { propiedad.borrar(); close(cerrado) })
	go cerrar()
	vence := time.After(time.Second)
	for {
		barrera.mu.Lock()
		cerrando := barrera.cerrando
		barrera.mu.Unlock()
		if cerrando {
			break
		}
		select {
		case <-vence:
			t.Fatal("el cierre OH no inició")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if sesion.clave == cero || e.p.clave != claveCT {
		t.Fatal("el cierre borró la clave antes de drenar el GET o alteró CT")
	}
	close(liberar)
	select {
	case <-termino:
	case <-time.After(time.Second):
		t.Fatal("GET OH no terminó")
	}
	select {
	case <-cerrado:
	case <-time.After(time.Second):
		t.Fatal("clave OH no se cerró tras el GET")
	}
	cerrar()
	if sesion.clave != cero || e.p.clave != claveCT {
		t.Fatal("cierre normal retuvo la clave OH o borró CT")
	}
}
