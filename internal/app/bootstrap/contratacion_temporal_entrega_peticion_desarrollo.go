package bootstrap

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"fmt"
	"maps"
	"time"

	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const rutaEntregaPeticionCentro = "/api/vec/contratacion-temporal/peticiones-centro/rrhh"

type claveMaterialEntregaPeticionDesarrollo struct{}
type claveAltaDePeticionDesarrollo struct{}

type proveedorEntregaPeticionDesarrollo struct {
	alta  *dependenciasAltaContratacionTemporalDesarrollo
	reloj relojContratacionTemporalDesarrollo
}

func nuevaRutaEntregaPeticionDesarrollo(alta *dependenciasAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo) (vechttp.RutaExacta, error) {
	vacia := vechttp.RutaExacta{}
	p := &proveedorEntregaPeticionDesarrollo{alta, reloj}
	if alta == nil || alta.soporte == nil || alta.postgresql.gobierno == nil || alta.postgresql.ejecucion == nil {
		return vacia, ports.ErrPeticionCentroNoDisponible
	}
	v, err := alta.soporte.contexto.Vinculo.Datos()
	if err != nil {
		return vacia, err
	}
	var concesiones []vecdomain.ConcesionRol
	for _, accion := range []string{ports.AccionConsultarPeticionesRRHH, ports.AccionEntregarPeticionRRHH} {
		concesiones = append(concesiones, vecdomain.ConcesionRol{Accion: accion, ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoEntregaPeticionCentro, Finalidades: []string{ports.FinalidadEntregaPeticionCentro}, GarantiaMinima: vecdomain.AuthAssuranceHigh})
	}
	instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, reloj.Ahora(), "entrega-peticion-rrhh", "Recepción de peticiones por RRHH", "entrega-peticion-rrhh-desarrollo", concesiones, []vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
	if err != nil {
		return vacia, err
	}
	alta.soporte.instantaneaEntregaPeticion = instantanea
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
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

func (p *proveedorEntregaPeticionDesarrollo) ActorEntregaPeticionCentro(ctx context.Context) (string, string, error) {
	if p == nil || p.alta == nil || p.alta.soporte == nil || ctx == nil || ctx.Err() != nil {
		return "", "", ports.ErrAutorizacionDenegada
	}
	c, ok := p.alta.soporte.capacidadValida(ctx)
	if !ok || c.ruta != rutaEntregaPeticionCentro || c.certificadoVerificadoEn.IsZero() || !p.reloj.Ahora().Before(c.certificadoValidoHasta) {
		return "", "", ports.ErrAutorizacionDenegada
	}
	v, err := p.alta.soporte.contexto.Vinculo.Datos()
	if err != nil {
		return "", "", ports.ErrAutorizacionDenegada
	}
	return v.PrincipalID, v.PerfilActivoRef, nil
}

func (p *proveedorEntregaPeticionDesarrollo) AutorizarEntregaPeticionCentro(ctx context.Context, m ports.MaterialEntregaPeticionCentro) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var vacio vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	a, perfil, err := p.ActorEntregaPeticionCentro(ctx)
	if err != nil || m.ActorRef != a || m.PerfilRef != perfil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	if m.Modo == "preparar" {
		sello, err := p.ambitoDeClaveAlta(ctx, m.ClaveAltaCandidata, a, perfil)
		if err != nil || !hmac.Equal([]byte(sello), []byte(m.AmbitoAltaHMAC)) {
			return vacio, ports.ErrAutorizacionDenegada
		}
	}
	r, err := postgresct.RecursoEntregaPeticionCentro(m)
	if err != nil {
		return vacio, err
	}
	c, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, err
	}
	d := vecdomain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: p.alta.soporte.contexto.Vinculo, ReferenciaMotivo: motivoEntregaPeticionDesarrollo(), Accion: postgresct.AccionEntregaPeticionCentro(m), Recurso: r, Finalidad: ports.FinalidadEntregaPeticionCentro, Correlacion: c}
	ctx = context.WithValue(ctx, claveMaterialEntregaPeticionDesarrollo{}, m)
	if !solicitudAutorizacionEntregaPeticionValida(ctx, d) {
		return vacio, ports.ErrAutorizacionDenegada
	}
	s, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(d)
	if err != nil {
		return vacio, err
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, d)
	decision, confirmacion, err := p.alta.autorizador.ExigirSolicitudLigadaV3(ctx, s, p.alta.soporte.contexto.Resultado)
	if err != nil {
		return vacio, err
	}
	return p.alta.postgresql.proveedorMaterial.proveerMaterialConfirmacion(ctx, s, decision, confirmacion, motivoEntregaPeticionDesarrollo(), p.alta.soporte.contexto.Resultado)
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

func (p *proveedorEntregaPeticionDesarrollo) RegistrarExpedientePeticion(ctx context.Context, e ports.EntregaPeticionCentro) (ports.AltaDePeticionCentro, error) {
	var vacia ports.AltaDePeticionCentro
	a, perfil, err := p.ActorEntregaPeticionCentro(ctx)
	if err != nil || e.ValidarReserva() != nil || e.EstadoEntrega != "preparada" || e.ActorRef != a || e.PerfilRef != perfil {
		return vacia, ports.ErrAutorizacionDenegada
	}
	ctx = context.WithValue(ctx, claveAltaDePeticionDesarrollo{}, e)
	comando, err := p.alta.soporte.ResolverContextoCanalAlta(ctx)
	if err != nil {
		return vacia, err
	}
	comando.ClaveIdempotencia = e.ClaveAlta
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
