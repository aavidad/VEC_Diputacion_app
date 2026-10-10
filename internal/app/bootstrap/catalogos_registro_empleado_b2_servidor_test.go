package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	apppersonal "vec-diputacion-granada/internal/modules/personal/application"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Publicar y retirar tienen un perfil nominal propio, en el ámbito del
// organismo, con la concesión exacta que el núcleo exige a esos actos, y el
// perfil de Personal que consulta el catálogo no gana ninguna de las dos.
func TestCatalogoEmpleadoB2GobiernoConPerfilPropioYConcesionExacta(t *testing.T) {
	config := configuracionB2PuraPrueba().PersonalB2
	p := perfilesB2VinculoPrueba(t, config)
	publicar := p.b2[personal.AccionPublicarCatalogoEmpleadoB2]
	retirar := p.b2[personal.AccionRetirarCatalogoEmpleadoB2]
	consulta := p.b2[personal.AccionConsultarCatalogoEmpleadoB2]
	if publicar == nil || publicar != retirar || publicar == consulta || publicar.clave != "incorporacion_b2_"+grupoCatalogoEmpleadoB2 {
		t.Fatal("publicar y retirar no comparten un perfil nominal propio")
	}
	c := publicar.plantilla.VersionRol.Concesiones
	if len(c) != 2 {
		t.Fatalf("concesiones del gobierno del catálogo: %+v", c)
	}
	for i, accion := range []string{personal.AccionPublicarCatalogoEmpleadoB2, personal.AccionRetirarCatalogoEmpleadoB2} {
		if c[i].Accion != accion || c[i].ModuloID != "personal" || c[i].TipoRecurso != "entrada_catalogo_empleado_rrhh" ||
			!slices.Equal(c[i].Finalidades, []string{"gobernar_catalogo_empleado"}) || !slices.Equal(c[i].CamposPermitidos, []string{"entrada", "recibo"}) ||
			c[i].GarantiaMinima != core.AuthAssuranceHigh {
			t.Fatalf("concesión distinta del núcleo para %s: %+v", accion, c[i])
		}
	}
	a := publicar.plantilla.AsignacionPerfil.Ambitos
	if len(a) != 1 || a[0].Clave != "organismo_ref" || !slices.Equal(a[0].Valores, []string{config.OrganismoRef}) {
		t.Fatalf("ámbitos del gobierno del catálogo: %+v", a)
	}
	for _, concesion := range consulta.plantilla.VersionRol.Concesiones {
		if accionGobiernoCatalogoEmpleadoB2(concesion.Accion) {
			t.Fatal("el perfil de consulta de Personal ganó publicar o retirar")
		}
	}
}

// Sin las dos operaciones (servidor ya desplegado) se arranca igual, no hay
// perfil y publicar se deniega; una sola de las dos no es una configuración.
func TestCatalogoEmpleadoB2GobiernoOpcionalYSiempreJunto(t *testing.T) {
	sin := configuracionB2PuraPrueba().PersonalB2
	delete(sin.Operaciones, claveCatalogoPublicarB2)
	delete(sin.Operaciones, claveCatalogoRetirarB2)
	if err := validarConfiguracionIncorporacionB2(sin); err != nil {
		t.Fatalf("configuración previa sin gobierno del catálogo rechazada: %v", err)
	}
	p := perfilesB2VinculoPrueba(t, sin)
	if p.b2[personal.AccionPublicarCatalogoEmpleadoB2] != nil || len(p.b2) != len(operacionesIncorporacionB2())-2 {
		t.Fatal("perfil de gobierno creado sin configuración")
	}
	for _, perfil := range p.todos() {
		if strings.HasSuffix(perfil.clave, grupoCatalogoEmpleadoB2) {
			t.Fatal("perfil de gobierno provisionable sin configuración")
		}
	}
	a := &autoridadIncorporacionPersonalB2{perfiles: p, organismoRef: sin.OrganismoRef}
	m := materialCatalogoPublicarPrueba(t, sin.OrganismoRef)
	ctx := context.WithValue(context.Background(), claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: "POST", ruta: rutaCatalogosRegistroEmpleadoB2})
	if _, err := a.AutorizarCatalogoRegistroEmpleadoB2(ctx, m); !errors.Is(err, ct.ErrAutorizacionDenegada) {
		t.Fatalf("publicación sin perfil configurado: %v", err)
	}
	for _, clave := range []string{claveCatalogoPublicarB2, claveCatalogoRetirarB2} {
		media := configuracionB2PuraPrueba().PersonalB2
		delete(media.Operaciones, clave)
		if validarConfiguracionIncorporacionB2(media) == nil {
			t.Fatalf("configuración sólo con una de las dos operaciones admitida (falta %s)", clave)
		}
	}
}

// Un organismo distinto del configurado en el servidor se deniega antes de
// pedir decisión alguna.
func TestCatalogoEmpleadoB2GobiernoRechazaOrganismoAjeno(t *testing.T) {
	config := configuracionB2PuraPrueba().PersonalB2
	a := &autoridadIncorporacionPersonalB2{perfiles: perfilesB2VinculoPrueba(t, config), organismoRef: config.OrganismoRef}
	if _, err := a.AutorizarCatalogoRegistroEmpleadoB2(context.Background(), materialCatalogoPublicarPrueba(t, "organismo:otro")); !errors.Is(err, ct.ErrAutorizacionDenegada) {
		t.Fatalf("organismo ajeno admitido: %v", err)
	}
}

func materialCatalogoPublicarPrueba(t *testing.T, organismo string) personal.MaterialCatalogoEmpleadoB2 {
	t.Helper()
	alta, _, _ := escenarioConsultasRRHHDesarrolloPrueba(t)
	s := personal.SolicitudCambioCatalogoEmpleadoB2{Operacion: "publicar", OrganismoRef: organismo, Tipo: "regimen",
		Ref: "regimen:funcionario-interino", Version: 1, Revision: 1, Denominacion: "Funcionario interino",
		VigenteDesde: personal.FechaCivil("2026-01-01"), IdempotenciaRef: "11111111-2222-4333-8444-555555555555",
		Actor: alta.soporte.contexto.Resultado.Contexto}
	s.HuellaSHA256 = personal.HuellaPublicacionCatalogoEmpleadoB2(s)
	m, err := personal.NuevoMaterialCambioCatalogoEmpleadoB2(s)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// La ruta nueva sólo admite publicar y retirar por POST; ninguna otra ruta B2
// admite esos actos.
func TestCatalogoEmpleadoB2AccionesPermitidasPorRuta(t *testing.T) {
	en := func(metodo, ruta string) context.Context {
		return context.WithValue(context.Background(), claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: metodo, ruta: ruta})
	}
	publicar, retirar := personal.AccionPublicarCatalogoEmpleadoB2, personal.AccionRetirarCatalogoEmpleadoB2
	casos := []struct {
		ctx    context.Context
		accion string
		ok     bool
	}{
		{en("POST", rutaCatalogosRegistroEmpleadoB2), publicar, true},
		{en("POST", rutaCatalogosRegistroEmpleadoB2), retirar, true},
		{en("GET", rutaCatalogosRegistroEmpleadoB2), publicar, false},
		{en("POST", rutaCatalogosRegistroEmpleadoB2), personal.AccionConsultarCatalogoEmpleadoB2, false},
		{en("POST", rutaCatalogosRegistroEmpleadoB2), personal.AccionAltaEmpleadoB2, false},
		{en("POST", rutaCatalogosRegistroEmpleadoB2), ct.AccionRegistrarPlanNominalB2, false},
		{en("POST", httpct.RutaConfirmacionB2), publicar, false},
		{en("POST", httpct.RutaConfirmacionB2), retirar, false},
		{en("POST", httpct.RutaPlanB2), publicar, false},
		{en("GET", httpct.RutaPlanB2), retirar, false},
		{en("POST", httpct.RutaVinculoCategoriaRPTB2), publicar, false},
	}
	for _, c := range casos {
		if operacionPermitidaEnRutaIncorporacionB2(c.ctx, c.accion) != c.ok {
			t.Errorf("%v %s: se esperaba %v", c.ctx.Value(claveRutaPeticionIncorporacionB2{}), c.accion, c.ok)
		}
	}
}

type registradorFronteraCatalogoPrueba struct {
	mu      sync.Mutex
	ordenes []vp.OrdenAuditoriaFronteraRutaExacta
}

func (r *registradorFronteraCatalogoPrueba) RegistrarAuditoriaFronteraRutaExacta(_ context.Context, o vp.OrdenAuditoriaFronteraRutaExacta) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ordenes = append(r.ordenes, o)
	return nil
}

type repositorioCatalogoNoUsadoPrueba struct{ llamadas int }

func (r *repositorioCatalogoNoUsadoPrueba) ConsultarRRHH(context.Context, personalports.OrdenCatalogoEmpleadoB2) (personalports.ResultadoConsultaCatalogoEmpleadoB2, error) {
	r.llamadas++
	return personalports.ResultadoConsultaCatalogoEmpleadoB2{}, personal.ErrRegistroEmpleadoB2NoDisponible
}
func (r *repositorioCatalogoNoUsadoPrueba) CambiarRRHH(context.Context, personalports.OrdenCatalogoEmpleadoB2) (personalports.ResultadoCambioCatalogoEmpleadoB2, error) {
	r.llamadas++
	return personalports.ResultadoCambioCatalogoEmpleadoB2{}, personal.ErrRegistroEmpleadoB2NoDisponible
}

// La petición sólo llega al manejador de Personal por la ruta CT exacta. Sin
// contexto nominal B2 (fuera del canal acreditado) la publicación se deniega
// con 403, se audita en la frontera CT con la ruta CT y no toca PostgreSQL.
func TestCatalogoEmpleadoB2RutaHTTPDeniegaYAuditaSinEscribir(t *testing.T) {
	config := configuracionB2PuraPrueba().PersonalB2
	a := &autoridadIncorporacionPersonalB2{perfiles: perfilesB2VinculoPrueba(t, config), organismoRef: config.OrganismoRef}
	repo := &repositorioCatalogoNoUsadoPrueba{}
	servicio, err := apppersonal.NuevoServicioCatalogosRegistroEmpleadoB2(a, repo)
	if err != nil {
		t.Fatal(err)
	}
	registrador := &registradorFronteraCatalogoPrueba{}
	ruta, err := rutaCatalogosEmpleadoB2(servicio, a, registrador, nil, catalogoFronterasComunDesarrollo{})
	if err != nil || ruta.Ruta != rutaCatalogosRegistroEmpleadoB2 {
		t.Fatalf("ruta del catálogo: %v", err)
	}
	cuerpo := `{"operacion":"publicar","tipo":"regimen","ref":"regimen:funcionario-interino","version":1,"revision":1,` +
		`"denominacion":"Funcionario interino","huella_sha256":"` + strings.Repeat("a", 64) + `","vigente_desde":"2026-01-01","vigente_hasta":""}`
	peticion := func(ruta string) *http.Request {
		r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "11111111-2222-4333-8444-555555555555")
		return r
	}
	w := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(w, peticion(rutaCatalogosRegistroEmpleadoB2))
	if w.Code != http.StatusForbidden {
		t.Fatalf("publicación sin canal nominal: HTTP %d %s", w.Code, w.Body.String())
	}
	if len(registrador.ordenes) != 1 || registrador.ordenes[0].Superficie != vp.SuperficieAuditoriaFronteraRutaExactaContratacionTemporal ||
		registrador.ordenes[0].Ruta != rutaCatalogosRegistroEmpleadoB2 || registrador.ordenes[0].Motivo != vp.MotivoAuditoriaFronteraRutaExactaAccesoDenegado {
		t.Fatalf("negativa sin auditoría de frontera CT: %+v", registrador.ordenes)
	}
	for _, otra := range []string{"/api/vec/personal/catalogos-registro-empleado", rutaCatalogosRegistroEmpleadoB2 + "/"} {
		w = httptest.NewRecorder()
		ruta.Manejador.ServeHTTP(w, peticion(otra))
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s: HTTP %d", otra, w.Code)
		}
	}
	if repo.llamadas != 0 {
		t.Fatal("se llegó a PostgreSQL sin decisión")
	}
	if _, err := rutaCatalogosEmpleadoB2(servicio, a, nil, nil, catalogoFronterasComunDesarrollo{}); err == nil {
		t.Fatal("ruta montada sin auditoría de frontera")
	}
}
