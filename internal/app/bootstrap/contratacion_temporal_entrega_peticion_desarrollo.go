package bootstrap

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"reflect"
	"strings"
	"time"

	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const rutaEntregaPeticionCentro = "/api/vec/contratacion-temporal/peticiones-centro/rrhh"

type claveMaterialEntregaPeticionDesarrollo struct{}
type claveAltaDePeticionDesarrollo struct{}

type proveedorEntregaPeticionDesarrollo struct {
	alta    *dependenciasAltaContratacionTemporalDesarrollo
	reloj   relojContratacionTemporalDesarrollo
	auditor vecports.RegistradorAuditoriaFronteraRutaExacta
}

func nuevaRutaEntregaPeticionDesarrollo(alta *dependenciasAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo) (vechttp.RutaExacta, error) {
	vacia := vechttp.RutaExacta{}
	p := &proveedorEntregaPeticionDesarrollo{alta: alta, reloj: reloj}
	if alta == nil || alta.soporte == nil || alta.postgresql.gobierno == nil ||
		alta.postgresql.ejecucion == nil || alta.postgresql.registradorAuditoriaFrontera == nil {
		return vacia, ports.ErrPeticionCentroNoDisponible
	}
	p.auditor = alta.postgresql.registradorAuditoriaFrontera
	if alta.soporte.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, http.MethodGet) == nil ||
		alta.soporte.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, http.MethodPost) == nil {
		return vacia, ports.ErrPeticionCentroNoDisponible
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(10*time.Second))
	defer cancelar()
	desde, _, _ := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(reloj.Ahora())
	if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []vecdomain.ReferenciaEntradaCatalogo{motivoEntregaPeticionDesarrollo()}, desde); err != nil {
		return vacia, err
	}
	repo, err := postgresct.NuevoRepositorioEntregasPeticionCentroPostgreSQL(alta.postgresql.ejecucion, p)
	if err != nil {
		return vacia, err
	}
	s, err := application.NuevoServicioEntregaPeticionCentro(repo, p)
	if err != nil {
		return vacia, err
	}
	return vechttp.RutaExacta{Ruta: rutaEntregaPeticionCentro, Manejador: &manejadorEntregaPeticionDesarrollo{p, repo, s}}, nil
}

func nuevaInstantaneaAutorizacionLectorEntregaPeticionDesarrollo(
	principalID, perfilRef string, ahora time.Time,
) (vecdomain.InstantaneaAutorizacion, error) {
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(principalID, perfilRef, ahora,
		"entrega-peticion-rrhh-lector", "Consulta de peticiones por RRHH",
		"entrega-peticion-rrhh-lector-desarrollo",
		[]vecdomain.ConcesionRol{{Accion: ports.AccionConsultarPeticionesRRHH, ModuloID: ports.ModuloContratacion,
			TipoRecurso: ports.TipoRecursoEntregaPeticionCentro,
			Finalidades: []string{ports.FinalidadEntregaPeticionCentro}, GarantiaMinima: vecdomain.AuthAssuranceHigh}},
		[]vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
}

// El POST y su alta anidada usan el mismo perfil y las mismas tres
// dimensiones, obtenidas del catálogo de centros/categorías y organización.
func nuevaInstantaneaAutorizacionEntregaPeticionDesarrollo(
	principalID, perfilRef string, ahora time.Time, origen *origenConsultasContratacionTemporalDesarrollo,
) (vecdomain.InstantaneaAutorizacion, error) {
	if origen == nil {
		return vecdomain.InstantaneaAutorizacion{}, ports.ErrPeticionCentroNoDisponible
	}
	catalogos, err := origen.catalogosAlta()
	if err != nil || len(catalogos.centrosOrganizacion) == 0 || len(catalogos.Categorias) == 0 {
		return vecdomain.InstantaneaAutorizacion{}, ports.ErrPeticionCentroNoDisponible
	}
	centros := append([]string(nil), catalogos.centrosOrganizacion...)
	categorias := make([]string, 0, len(catalogos.Categorias))
	for _, categoria := range catalogos.Categorias {
		categorias = append(categorias, categoria.Referencia)
	}
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
		principalID, perfilRef, ahora, "entrega-peticion-rrhh-fijo", "Entrega de peticiones por RRHH",
		"entrega-peticion-rrhh-fijo-desarrollo",
		[]vecdomain.ConcesionRol{
			{Accion: ports.AccionEntregarPeticionRRHH, ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoEntregaPeticionCentro, Finalidades: []string{ports.FinalidadEntregaPeticionCentro}, GarantiaMinima: vecdomain.AuthAssuranceHigh},
			{Accion: ports.AccionCrearSolicitud, ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoExpediente, Finalidades: []string{ports.FinalidadCrearSolicitud}, GarantiaMinima: vecdomain.AuthAssuranceHigh},
		},
		[]vecdomain.AmbitoPerfil{
			{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
			{Clave: "centro_ref", Valores: centros},
			{Clave: "categoria_ref", Valores: categorias},
		},
	)
}

func (p *proveedorEntregaPeticionDesarrollo) ActorEntregaPeticionCentro(ctx context.Context) (string, string, error) {
	if p == nil || p.alta == nil || p.alta.soporte == nil || ctx == nil {
		return "", "", ports.ErrAutorizacionDenegada
	}
	if ctx.Err() != nil {
		return "", "", ctx.Err()
	}
	c, ok := p.alta.soporte.capacidadValida(ctx)
	if !ok || c.ruta != rutaEntregaPeticionCentro || (c.metodo != http.MethodGet && c.metodo != http.MethodPost) ||
		c.certificadoVerificadoEn.IsZero() || !p.reloj.Ahora().Before(c.certificadoValidoHasta) {
		return "", "", ports.ErrAutorizacionDenegada
	}
	operativo, err := p.alta.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil {
		if errors.Is(err, ports.ErrConsultaRRHHNoDisponible) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "", "", err
		}
		return "", "", ports.ErrAutorizacionDenegada
	}
	v, err := operativo.Vinculo.Datos()
	if err != nil {
		return "", "", ports.ErrAutorizacionDenegada
	}
	return v.PrincipalID, v.PerfilActivoRef, nil
}

func (p *proveedorEntregaPeticionDesarrollo) ComprobarPerfilEntregaPeticionCentro(ctx context.Context) error {
	a, perfil, err := p.ActorEntregaPeticionCentro(ctx)
	if err != nil {
		if causaFalloEntregaPeticionDesarrollo(err) != "autorizacion_denegada" {
			return errors.Join(ports.ErrPeticionCentroNoDisponible, err)
		}
		return p.denegarEntregaPreV3(ctx, "")
	}
	capacidad, valida := p.alta.soporte.capacidadValida(ctx)
	fijo := p.alta.soporte.perfilFijoParaContexto(ctx, rutaEntregaPeticionCentro)
	if !valida || capacidad.metodo != http.MethodPost || fijo == nil ||
		fijo.perfilRef() != perfil || fijo.plantilla.AsignacionPerfil.PrincipalID != a {
		return p.denegarEntregaPreV3(ctx, a)
	}
	_, estado := p.alta.soporte.consumirPerfilFijoCTDesarrolloConEstado(ctx, fijo)
	switch estado {
	case perfilFijoConsumoVigente:
		return nil
	case perfilFijoConsumoDenegado:
		return p.denegarEntregaPreV3(ctx, a)
	default:
		return ports.ErrPeticionCentroNoDisponible
	}
}

func (p *proveedorEntregaPeticionDesarrollo) denegarEntregaPreV3(ctx context.Context, actor string) error {
	if p.registrarDenegacionPreV3(ctx, actor) != nil {
		return ports.ErrPeticionCentroNoDisponible
	}
	return errors.Join(ports.ErrAutorizacionDenegada, vecdomain.ErrAutorizacionDenegada)
}

// La consulta de ámbitos ocurre antes de solicitar una decisión V3. Su
// denegación se registra en la autoridad de auditoría de frontera ya montada,
// con causa fija, sin referencia de petición ni material HMAC.
func (p *proveedorEntregaPeticionDesarrollo) registrarDenegacionPreV3(ctx context.Context, actor string) error {
	if p == nil || p.auditor == nil {
		return ports.ErrPeticionCentroNoDisponible
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(),
		seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		slog.Warn("denegación previa a V3 sin correlación de auditoría", "ruta", rutaEntregaPeticionCentro)
		return ports.ErrPeticionCentroNoDisponible
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		slog.Warn("denegación previa a V3 sin correlación canónica", "ruta", rutaEntregaPeticionCentro)
		return ports.ErrPeticionCentroNoDisponible
	}
	if !strings.HasPrefix(correlacionRef, "correlacion_") ||
		len(correlacionRef) != len("correlacion_")+32 {
		slog.Warn("denegación previa a V3 sin correlación de frontera válida", "ruta", rutaEntregaPeticionCentro)
		return ports.ErrPeticionCentroNoDisponible
	}
	orden := vecports.OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: "corr_" + strings.TrimPrefix(correlacionRef, "correlacion_"),
		Motivo:         vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado,
		Superficie:     vecports.SuperficieAuditoriaFronteraRutaExactaContratacionTemporal,
		Ruta:           rutaEntregaPeticionCentro, ActorRef: actor,
	}
	if orden.Validar() != nil {
		slog.Warn("denegación previa a V3 sin orden válida de auditoría", "ruta", rutaEntregaPeticionCentro)
		return ports.ErrPeticionCentroNoDisponible
	}
	base := context.Background()
	if ctx != nil {
		base = context.WithoutCancel(ctx)
	}
	ctxAuditoria, cancelar := context.WithTimeout(base, plazoarranque.Ampliar(2*time.Second))
	defer cancelar()
	if err := p.auditor.RegistrarAuditoriaFronteraRutaExacta(ctxAuditoria, orden); err != nil {
		slog.Warn("denegación previa a V3 no registrada", "ruta", rutaEntregaPeticionCentro)
		return ports.ErrPeticionCentroNoDisponible
	}
	return nil
}

func (p *proveedorEntregaPeticionDesarrollo) RegistrarDenegacionEntregaPreV3(ctx context.Context) error {
	a, _, err := p.ActorEntregaPeticionCentro(ctx)
	if err != nil {
		a = ""
	}
	return p.registrarDenegacionPreV3(ctx, a)
}

func (p *proveedorEntregaPeticionDesarrollo) AutorizarEntregaPeticionCentro(ctx context.Context, m ports.MaterialEntregaPeticionCentro) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var vacio vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	a, perfil, err := p.ActorEntregaPeticionCentro(ctx)
	if err != nil {
		return vacio, err
	}
	if m.ActorRef != a || m.PerfilRef != perfil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	capacidad, _ := p.alta.soporte.capacidadValida(ctx)
	if (m.Modo == "bandeja") != (capacidad.metodo == http.MethodGet) {
		return vacio, ports.ErrAutorizacionDenegada
	}
	if m.Modo == "preparar" {
		sello, err := p.ambitoDeClaveAlta(ctx, m.ClaveAltaCandidata, a, perfil)
		if err != nil || !hmac.Equal([]byte(sello), []byte(m.AmbitoAltaHMAC)) {
			return vacio, p.denegarEntregaPreV3(ctx, a)
		}
	}
	fijo := p.alta.soporte.perfilFijoParaContexto(ctx, rutaEntregaPeticionCentro)
	if fijo == nil || fijo.perfilRef() != perfil {
		return vacio, p.denegarEntregaPreV3(ctx, a)
	}
	_, estado := p.alta.soporte.consumirPerfilFijoCTDesarrolloConEstado(ctx, fijo)
	switch estado {
	case perfilFijoConsumoDenegado:
		return vacio, p.denegarEntregaPreV3(ctx, a)
	case perfilFijoConsumoFuenteNoDisponible:
		return vacio, ports.ErrPeticionCentroNoDisponible
	}
	r, err := postgresct.RecursoEntregaPeticionCentro(m)
	if err != nil {
		return vacio, err
	}
	c, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, err
	}
	operativo, err := p.alta.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil {
		return vacio, err
	}
	d := vecdomain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: operativo.Vinculo, ReferenciaMotivo: motivoEntregaPeticionDesarrollo(), Accion: postgresct.AccionEntregaPeticionCentro(m), Recurso: r, Finalidad: ports.FinalidadEntregaPeticionCentro, Correlacion: c}
	ctx = context.WithValue(ctx, claveMaterialEntregaPeticionDesarrollo{}, m)
	if !solicitudAutorizacionEntregaPeticionValida(ctx, d) {
		return vacio, ports.ErrAutorizacionDenegada
	}
	s, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(d)
	if err != nil {
		return vacio, err
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, d)
	decision, confirmacion, err := p.alta.autorizador.ExigirSolicitudLigadaV3(ctx, s, operativo.Resultado)
	if err != nil {
		// La asignación puede cambiar entre la primera lectura y el PDP. La
		// segunda lectura conserva la distinción 403/503 sin conceder por carrera.
		_, estado := p.alta.soporte.consumirPerfilFijoCTDesarrolloConEstado(ctx, fijo)
		if estado == perfilFijoConsumoDenegado {
			return vacio, p.denegarEntregaPreV3(ctx, a)
		}
		if estado == perfilFijoConsumoFuenteNoDisponible {
			return vacio, ports.ErrPeticionCentroNoDisponible
		}
		return vacio, err
	}
	return p.alta.postgresql.proveedorMaterial.proveerMaterialConfirmacion(ctx, s, decision, confirmacion, motivoEntregaPeticionDesarrollo(), operativo.Resultado)
}

func motivoEntregaPeticionDesarrollo() vecdomain.ReferenciaEntradaCatalogo {
	return vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_entrega_peticion_centro", CatalogoVersion: 1, CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("entrega-peticion-centro-v1"), EntradaClave: referenciaAltaContratacionTemporalDesarrollo("motivo_", "entrega-peticion-centro")}
}

// UUIDv4 generado por el servidor: los 122 bits aleatorios proceden de CSPRNG.
// El mismo material autorizado liga la clave y su sello antes del alta.
func (p *proveedorEntregaPeticionDesarrollo) NuevaClaveAltaDePeticion(ctx context.Context) (string, string, error) {
	a, perfil, err := p.ActorEntregaPeticionCentro(ctx)
	if err != nil {
		return "", "", err
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", "", err
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	clave := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
	sello, err := p.ambitoDeClaveAlta(ctx, clave, a, perfil)
	return clave, sello, err
}

func (p *proveedorEntregaPeticionDesarrollo) ambitoDeClaveAlta(ctx context.Context, clave, actor, perfil string) (string, error) {
	sellos, err := p.sellosDeClaveAlta(ctx, clave, actor, perfil)
	if err != nil {
		return "", err
	}
	datos, err := sellos.Datos()
	if err != nil {
		return "", err
	}
	return datos.Activo.Valor, nil
}

func (p *proveedorEntregaPeticionDesarrollo) sellosDeClaveAlta(ctx context.Context, clave, actor, perfil string) (ports.ColeccionSellosHMAC, error) {
	return p.alta.soporte.ambitos.SellarAmbitoIdempotencia(ctx, ports.SolicitudSellarAmbitoIdempotencia{ClaveIdempotencia: clave, OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, ActorRef: actor, PerfilRef: perfil})
}

func solicitudAutorizacionEntregaPeticionValida(ctx context.Context, d vecdomain.DatosSolicitudAutorizacionLigadaV3) bool {
	if ctx == nil || d.Finalidad != ports.FinalidadEntregaPeticionCentro || d.ReferenciaMotivo != motivoEntregaPeticionDesarrollo() {
		return false
	}
	m, ok := ctx.Value(claveMaterialEntregaPeticionDesarrollo{}).(ports.MaterialEntregaPeticionCentro)
	if !ok || m.Validar() != nil || d.Accion != postgresct.AccionEntregaPeticionCentro(m) {
		return false
	}
	v, err := d.VinculoAutenticacionActor.Datos()
	if err != nil || m.ActorRef != v.PrincipalID || m.PerfilRef != v.PerfilActivoRef {
		return false
	}
	r, err := postgresct.RecursoEntregaPeticionCentro(m)
	return err == nil && r.Referencia == d.Recurso.Referencia && r.ModuloID == d.Recurso.ModuloID && r.Tipo == d.Recurso.Tipo && maps.Equal(r.Ambitos, d.Recurso.Ambitos) && maps.Equal(r.Atributos, d.Recurso.Atributos)
}

func (p *proveedorEntregaPeticionDesarrollo) RegistrarExpedientePeticion(ctx context.Context, e ports.EntregaPeticionCentro, numeroMOAD string) (ports.AltaDePeticionCentro, error) {
	var vacia ports.AltaDePeticionCentro
	a, perfil, err := p.ActorEntregaPeticionCentro(ctx)
	if err != nil {
		return vacia, err
	}
	if e.ValidarReserva() != nil || e.EstadoEntrega != "preparada" || e.ActorRef != a || e.PerfilRef != perfil {
		return vacia, ports.ErrAutorizacionDenegada
	}
	ctx = context.WithValue(ctx, claveAltaDePeticionDesarrollo{}, e)
	comando, err := p.alta.soporte.ResolverContextoCanalAlta(ctx)
	if err != nil {
		return vacia, err
	}
	comando.ClaveIdempotencia = e.ClaveAlta
	comando.NumeroExpedienteMOAD = numeroMOAD
	comando.Solicitud, err = e.Peticion.Solicitud.Clonar()
	if err != nil {
		return vacia, err
	}
	sellos, err := p.sellosDeClaveAlta(ctx, e.ClaveAlta, a, perfil)
	if err != nil || !sellos.Contiene(e.AmbitoAltaHMAC) {
		return vacia, ports.ErrAutorizacionDenegada
	}
	recibo, err := p.alta.servicio.Registrar(ctx, comando)
	if err != nil {
		return vacia, err
	}
	return ports.AltaDePeticionCentro{Recibo: recibo, AmbitoHMAC: e.AmbitoAltaHMAC}, nil
}

// VerificarOriginalAltaEntrega usa el perfil reservado sólo para recomponer
// la huella histórica. La autorización vigente ya se consume en CT150 dentro
// de la transacción de preparación; nunca se concede por este perfil antiguo.
func (p *proveedorEntregaPeticionDesarrollo) VerificarOriginalAltaEntrega(
	ctx context.Context, e ports.EntregaPeticionCentro, original ports.OriginalAltaEntrega,
) error {
	if p == nil || p.alta == nil || p.alta.soporte == nil || p.alta.huellas == nil ||
		ctx == nil || e.ValidarReserva() != nil || original.Validar() != nil {
		return ports.ErrReciboPeticionCentroNoConfiable
	}
	actorActual, _, err := p.ActorEntregaPeticionCentro(ctx)
	if err != nil {
		return err
	}
	if actorActual != e.ActorRef || original.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		original.ActorRef != e.ActorRef || original.PerfilRef != e.PerfilRef ||
		original.AmbitoHMAC != e.AmbitoAltaHMAC ||
		e.EstadoEntrega == "confirmada" && !reflect.DeepEqual(e.ReciboAlta, &original.ReciboAlta) {
		return ports.ErrReciboPeticionCentroNoConfiable
	}
	ambitos, err := p.sellosDeClaveAlta(ctx, e.ClaveAlta, e.ActorRef, e.PerfilRef)
	if err != nil {
		return ports.ErrPersistenciaNoDisponible
	}
	solicitud, err := e.Peticion.Solicitud.Clonar()
	if err != nil {
		return ports.ErrReciboPeticionCentroNoConfiable
	}
	if original.PoliticaFin == nil {
		if solicitud.Periodo.PoliticaFin != (domain.PoliticaFin{}) {
			return ports.ErrClaveIdempotenciaUsada
		}
	} else {
		if solicitud.Periodo.PoliticaFin != (domain.PoliticaFin{}) &&
			solicitud.Periodo.PoliticaFin != *original.PoliticaFin {
			return ports.ErrClaveIdempotenciaUsada
		}
		solicitud.Periodo.PoliticaFin = *original.PoliticaFin
	}
	huellas, err := p.alta.huellas.DerivarHuellaAlta(ctx, ports.MaterialHuellaAlta{
		OrganizacionRef: original.OrganizacionRef, ActorRef: e.ActorRef,
		PerfilRef: e.PerfilRef, Flujo: original.Flujo, Solicitud: solicitud,
	})
	if err != nil {
		return ports.ErrPersistenciaNoDisponible
	}
	if !ports.ColeccionesHMACAltaContienenPar(ambitos, huellas, original.AmbitoHMAC, original.HuellaPeticionHMAC) {
		return ports.ErrClaveIdempotenciaUsada
	}
	return nil
}

func altaDePeticionConfiable(ctx context.Context) (ports.EntregaPeticionCentro, bool) {
	if ctx == nil {
		return ports.EntregaPeticionCentro{}, false
	}
	e, ok := ctx.Value(claveAltaDePeticionDesarrollo{}).(ports.EntregaPeticionCentro)
	return e, ok && e.ValidarReserva() == nil && e.EstadoEntrega == "preparada"
}

func solicitudAutorizacionAltaDePeticionValida(ctx context.Context, d vecdomain.DatosSolicitudAutorizacionLigadaV3) bool {
	e, ok := altaDePeticionConfiable(ctx)
	v, err := d.VinculoAutenticacionActor.Datos()
	return ok && err == nil && v.PrincipalID == e.ActorRef && v.PerfilActivoRef == e.PerfilRef &&
		d.Accion == ports.AccionCrearSolicitud && d.Recurso.ModuloID == ports.ModuloContratacion && d.Recurso.Tipo == ports.TipoRecursoExpediente && d.Finalidad == ports.FinalidadCrearSolicitud &&
		len(d.Recurso.Ambitos) == 3 && d.Recurso.Ambitos["organizacion_ref"] == organizacionAltaContratacionTemporalDesarrollo &&
		d.Recurso.Ambitos["centro_ref"] == e.Peticion.Solicitud.CentroRef && d.Recurso.Ambitos["categoria_ref"] == e.Peticion.Solicitud.CategoriaRef
}
