package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// solicitudBandejaEntregaPeticionPrueba es la solicitud V3 que construye la
// lista de «Peticiones de los centros» de RRHH (modo bandeja).
func solicitudBandejaEntregaPeticionPrueba(t *testing.T) vecdomain.SolicitudAutorizacionLigadaV3 {
	t.Helper()
	p := vecdomain.Principal{ID: "desarrollo:rrhh", DisplayName: "Antonio Reyes Álvarez", Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo}, AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": strings.Repeat("a", 64)}}
	c, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(p, relojContratacionTemporalDesarrollo{}.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	v, err := c.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	m := ports.MaterialEntregaPeticionCentro{Modo: "bandeja", ActorRef: v.PrincipalID, PerfilRef: v.PerfilActivoRef}
	r, err := postgresct.RecursoEntregaPeticionCentro(m)
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		t.Fatal(err)
	}
	s, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: c.Vinculo, ReferenciaMotivo: motivoEntregaPeticionDesarrollo(),
		Accion: postgresct.AccionEntregaPeticionCentro(m), Recurso: r, Finalidad: ports.FinalidadEntregaPeticionCentro, Correlacion: correlacion})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// Reproduce el 503 permanente de la lista de peticiones de los centros en la
// principal: con Bolsa compuesta, el autorizador CT delega en el PDP común,
// que solo decide si la petición llega por una frontera declarada con
// política para su acción. La GET de la lista no tenía ninguna y el PDP la
// rechazaba antes de decidir (sin registro), lo que la ruta convertía en 503.
func TestPDPComunAutorizaLaListaDePeticionesDeCentrosDeRRHH(t *testing.T) {
	const perfil = "prf_ct_prueba"
	catalogoFronteras, err := nuevoCatalogoFronterasComunDesarrollo(descriptoresFronterasContratacionTemporalDesarrollo(perfil, []string{perfil}))
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoAutorizacionComunDesarrollo(catalogoFronteras, descriptoresAutorizacionContratacionTemporalDesarrollo(politicaDescriptoresCTPrueba(t)))
	if err != nil {
		t.Fatal(err)
	}
	pdp, err := nuevoAutorizadorComunDesarrollo(catalogo, relojContratacionTemporalDesarrollo{}, seguridadvec.GeneradorReferenciasCriptograficas{}, aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	// Lo que hace el revalidador HTTP con una GET a la ruta.
	descriptor, ok := catalogoFronteras.resolver(http.MethodGet, rutaEntregaPeticionCentro)
	if !ok || descriptor.Clave != "ct-peticiones-centro-rrhh-consultar" || !descriptor.admitePerfil(perfil) || descriptor.admitePerfil("prf_ct_ajeno") {
		t.Fatalf("la GET de la lista no tiene frontera propia del perfil CT: %#v", descriptor)
	}
	ctx := context.WithValue(context.Background(), claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: http.MethodGet, ruta: rutaEntregaPeticionCentro, superficie: descriptor.Superficie, catalogo: catalogoFronteras, descriptor: descriptor})
	solicitud := solicitudBandejaEntregaPeticionPrueba(t)
	if _, err := pdp.contextoSolicitud(ctx, solicitud); err != nil {
		t.Fatalf("el PDP común rechaza la lista de RRHH antes de decidir: %v", err)
	}
	// La frontera de la lista no sirve para entregar ni para crear expedientes.
	for _, accion := range []string{ports.AccionEntregarPeticionRRHH, ports.AccionCrearSolicitud} {
		if _, ok := catalogo.politicaPara(accion, descriptor.Clave, descriptor.ClavePolitica, descriptor.ClaveCapacidad); ok {
			t.Fatalf("la frontera de la lista admite %s", accion)
		}
	}
	post, ok := catalogoFronteras.resolver(http.MethodPost, rutaEntregaPeticionCentro)
	if !ok || post.Clave != "ct-peticiones-centro-rrhh-entregar" || !post.admitePerfil(perfil) || post.admitePerfil("prf_ct_ajeno") {
		t.Fatal("la entrega no conserva una frontera POST del perfil CT")
	}
	for _, accion := range []string{ports.AccionEntregarPeticionRRHH, ports.AccionCrearSolicitud} {
		if _, ok := catalogo.politicaPara(accion, post.Clave, post.ClavePolitica, post.ClaveCapacidad); !ok {
			t.Fatalf("el POST no enlaza la acción %s", accion)
		}
	}
	ctxPost := context.WithValue(context.Background(), claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: http.MethodPost, ruta: rutaEntregaPeticionCentro, superficie: post.Superficie, catalogo: catalogoFronteras, descriptor: post})
	if _, err := pdp.contextoSolicitud(ctxPost, solicitudBandejaEntregaPeticionPrueba(t)); !errors.Is(err, errAutorizacionComunDesarrolloNoDisponible) {
		t.Fatalf("el POST admitió la acción de consulta: %v", err)
	}
}

// La ruta ya no convierte todo fallo en 503: la denegación es 403 y la causa
// queda en el registro con un código fijo, sin el texto del error.
func TestFalloEntregaPeticionSeparaDenegacionDeIndisponibilidad(t *testing.T) {
	var registro bytes.Buffer
	previo := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&registro, nil)))
	t.Cleanup(func() { slog.SetDefault(previo) })
	// Referencia opaca sintética que no debe aparecer en el registro.
	secreto := "per_" + strings.Repeat("d", 32)
	casos := []struct {
		err    error
		estado int
		codigo string
		causa  string
	}{
		{errors.Join(vecdomain.ErrAutorizacionDenegada, errors.New(secreto)), 403, "operacion_denegada", "autorizacion_denegada"},
		{ports.ErrAutorizacionDenegada, 403, "operacion_denegada", "autorizacion_denegada"},
		{errors.Join(vecports.ErrFuenteAutorizacionNoDisponible, errors.New(secreto)), 503, "servicio_no_disponible", "fuente_autorizacion_no_disponible"},
		{vecports.ErrInstantaneaAutorizacionObsoleta, 503, "servicio_no_disponible", "permiso_vigente_no_operativo"},
		{ports.ErrReciboPeticionCentroNoConfiable, 503, "servicio_no_disponible", "respuesta_base_no_confiable"},
		{context.DeadlineExceeded, 503, "servicio_no_disponible", "peticion_cancelada_o_vencida"},
		{errors.New(secreto), 503, "servicio_no_disponible", "no_clasificada"},
		// Forma real: el servicio V3 y el PDP común envuelven la
		// indisponibilidad en ErrAutorizacionDenegada; sigue siendo 503.
		{errors.Join(vecdomain.ErrAutorizacionDenegada, errAutorizacionComunDesarrolloNoDisponible), 503, "servicio_no_disponible", "configuracion_autorizacion_no_disponible"},
		{errors.Join(vecdomain.ErrAutorizacionDenegada, vecdomain.ErrConfiguracionAccesoInvalida), 503, "servicio_no_disponible", "configuracion_autorizacion_no_disponible"},
		{errors.Join(vecdomain.ErrAutorizacionDenegada, vecports.ErrFuenteAutorizacionNoDisponible), 503, "servicio_no_disponible", "fuente_autorizacion_no_disponible"},
		{errors.Join(vecdomain.ErrAutorizacionDenegada, vecports.ErrInstantaneaAutorizacionObsoleta), 503, "servicio_no_disponible", "permiso_vigente_no_operativo"},
		{errors.Join(vecdomain.ErrAutorizacionDenegada, vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible), 503, "servicio_no_disponible", "registro_decision_no_disponible"},
		{errors.Join(vecdomain.ErrAutorizacionDenegada, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible), 503, "servicio_no_disponible", "registro_decision_no_disponible"},
		{errors.Join(vecdomain.ErrAutorizacionDenegada, context.Canceled), 503, "servicio_no_disponible", "peticion_cancelada_o_vencida"},
	}
	for _, c := range casos {
		registro.Reset()
		estado, codigo := falloEntregaPeticionDesarrollo(http.MethodGet, c.err)
		linea := registro.String()
		if estado != c.estado || codigo != c.codigo || !strings.Contains(linea, `"causa":"`+c.causa+`"`) || !strings.Contains(linea, rutaEntregaPeticionCentro) {
			t.Fatalf("%v: %d %s, registro %q", c.err, estado, codigo, linea)
		}
		if strings.Contains(linea, secreto) {
			t.Fatalf("el registro copia el texto del error: %q", linea)
		}
	}
}
