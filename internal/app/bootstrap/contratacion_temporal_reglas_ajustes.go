package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	ajusteshttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpapi/ajustesreglas"
	ajustespg "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ajustesapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/ajustesreglas"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const envCTMotivoAutorizacionAjustes = "VEC_CT_MOTIVO_AUTORIZACION_AJUSTES_FILE"

var errMotivoAutorizacionAjustes = errors.New("bootstrap: motivo de autorización de ajustes CT no disponible")

type catalogoMotivoAutorizacionAjustes struct {
	ID           string `json:"id"`
	Version      int    `json:"version"`
	EntradaClave string `json:"entrada_clave"`
	RolID        string `json:"rol_id"`
}

type motivoAutorizacionAjustesCT struct {
	Referencia vecdomain.ReferenciaEntradaCatalogo
	RolID      string
}

// El archivo se declara por configuración de desarrollo; su huella se liga a
// la publicación de motivos V3. Una ruta ausente conserva la capacidad apagada.
func cargarMotivoAutorizacionAjustesCT(cfg config.Config) (motivoAutorizacionAjustesCT, bool, error) {
	ruta := strings.TrimSpace(os.Getenv(envCTMotivoAutorizacionAjustes))
	if ruta == "" {
		return motivoAutorizacionAjustesCT{}, false, nil
	}
	if !cfg.DevelopmentEnabledByDoubleKey() {
		return motivoAutorizacionAjustesCT{}, false, ErrActivacionDesarrolloInvalida
	}
	archivo, err := os.Open(ruta)
	if err != nil {
		return motivoAutorizacionAjustesCT{}, false, errMotivoAutorizacionAjustes
	}
	defer archivo.Close()
	contenido, err := io.ReadAll(io.LimitReader(archivo, 4097))
	if err != nil || len(contenido) == 0 || len(contenido) > 4096 {
		return motivoAutorizacionAjustesCT{}, false, errMotivoAutorizacionAjustes
	}
	lector := json.NewDecoder(bytes.NewReader(contenido))
	lector.DisallowUnknownFields()
	var catalogo catalogoMotivoAutorizacionAjustes
	if lector.Decode(&catalogo) != nil || lector.Decode(new(any)) != io.EOF ||
		catalogo.Version < 1 || catalogo.ID == "" || catalogo.EntradaClave == "" || catalogo.RolID == "" {
		return motivoAutorizacionAjustesCT{}, false, errMotivoAutorizacionAjustes
	}
	huella := sha256.Sum256(contenido)
	resultado := motivoAutorizacionAjustesCT{
		Referencia: vecdomain.ReferenciaEntradaCatalogo{
			CatalogoID: catalogo.ID, CatalogoVersion: catalogo.Version,
			CatalogoHuellaSHA256: hex.EncodeToString(huella[:]),
			EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", catalogo.EntradaClave),
		},
		RolID: catalogo.RolID,
	}
	if !vecdomain.ReferenciaMotivoAutorizacionV2Valida(resultado.Referencia) ||
		!nombrePerfilSinteticoValido(catalogo.RolID) {
		return motivoAutorizacionAjustesCT{}, false, errMotivoAutorizacionAjustes
	}
	return resultado, true, nil
}

func nombrePerfilSinteticoValido(v string) bool {
	if len(v) < 3 || len(v) > 64 {
		return false
	}
	for _, c := range v {
		if !('a' <= c && c <= 'z' || '0' <= c && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}

const (
	rutaAjustesReglasCT      = ajusteshttp.Ruta
	accionConsultarAjustesCT = "contratacion_temporal.reglas.consultar_ajustes"
	accionAjustarReglasCT    = "contratacion_temporal.reglas.ajustar"
	audienciaAjustesCT       = "vec_contratacion_temporal.ajustes_reglas.v1"
	recursoReglasCT          = "vec.contratacion_temporal.reglas"
	finalidadAjustesCT       = "gobierno_reglas_contratacion_temporal"
)

func rutaConsultaAjustesReglasCT(ruta string) bool { return ruta == rutaAjustesReglasCT }

// El perfil de este corte concede exclusivamente consulta. CT110 aún no
// conserva la instantánea al abrir cada tramo: no hay concesión de escritura.
func componerPerfilConsultaAjustesCT(ctx context.Context, pool *pgxpool.Pool, s *soporteAltaContratacionTemporalDesarrollo,
	m motivoAutorizacionAjustesCT, aprobacion aprobacionProvisionPerfilesRRHHDesarrollo,
) (*perfilFijoCTDesarrollo, error) {
	if s == nil || pool == nil || ctx == nil || ctx.Err() != nil || m.RolID == "" ||
		!vecdomain.ReferenciaMotivoAutorizacionV2Valida(m.Referencia) {
		return nil, errMotivoAutorizacionAjustes
	}
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(s.reloj.Ahora())
	if !vigente || publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, pool,
		[]vecdomain.ReferenciaEntradaCatalogo{m.Referencia}, desde) != nil {
		return nil, errMotivoAutorizacionAjustes
	}
	principal := vecdomain.Principal{ID: s.principalID, Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": s.certificadoSHA256}}
	p, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, s.reloj.Ahora(), m.RolID,
		[]string{ajusteshttp.Ruta}, func(principalID, perfilRef string) (vecdomain.InstantaneaAutorizacion, error) {
			return plantillaConsultaAjustesCT(principalID, perfilRef, s.reloj.Ahora(), m.RolID)
		})
	if p != nil {
		p.metodo = http.MethodGet
	}
	if err != nil || s.registrarPerfilFijoCTDesarrollo(p) != nil {
		return nil, errMotivoAutorizacionAjustes
	}
	if err := asegurarPerfilesFijosCTDesarrollo(ctx, pool, s, aprobacion, p); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.motivoConsultaAjustesReglas = m.Referencia
	s.mu.Unlock()
	return p, nil
}

func plantillaConsultaAjustesCT(principalID, perfilRef string, ahora time.Time, rolID string) (vecdomain.InstantaneaAutorizacion, error) {
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(principalID, perfilRef, ahora,
		rolID, rolID, rolID,
		[]vecdomain.ConcesionRol{{Accion: accionConsultarAjustesCT, ModuloID: "contratacion_temporal",
			TipoRecurso: "catalogo_reglas", Finalidades: []string{finalidadAjustesCT},
			CamposPermitidos: []string{"historial", "vigente"}, GarantiaMinima: vecdomain.AuthAssuranceHigh}},
		[]vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
}

type proveedorConsultaAjustesCT struct {
	soporte  *soporteAltaContratacionTemporalDesarrollo
	pdp      autorizadorLigadoContratacionTemporalDesarrollo
	material *proveedorMaterialAltaContratacionTemporalDesarrollo
	motivo   vecdomain.ReferenciaEntradaCatalogo
	reloj    relojContratacionTemporalDesarrollo
}

func (p *proveedorConsultaAjustesCT) contexto(ctx context.Context) (contextoSeguridadComunDesarrollo, error) {
	vacio := contextoSeguridadComunDesarrollo{}
	if p == nil || p.soporte == nil || p.pdp == nil || p.material == nil || ctx == nil || ctx.Err() != nil {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	capacidad, valida := p.soporte.capacidadValida(ctx)
	frontera, tieneFrontera := fronteraSeguridadComunDesdeContexto(ctx)
	perfil := p.soporte.perfilFijoParaRutaYMetodo(ajusteshttp.Ruta, http.MethodGet)
	if !valida || !tieneFrontera || perfil == nil || capacidad.ruta != ajusteshttp.Ruta || capacidad.metodo != http.MethodGet ||
		frontera.ruta != ajusteshttp.Ruta || frontera.metodo != http.MethodGet ||
		frontera.descriptor.ClaveCapacidad != accionConsultarAjustesCT || !frontera.descriptor.admitePerfil(perfil.perfilRef()) {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	operativo, err := p.soporte.contextoOperativoDesarrollo(ctx)
	if errors.Is(err, ctports.ErrConsultaRRHHNoDisponible) {
		return vacio, ajustesapp.ErrNoDisponible
	}
	if err != nil || operativo.Resultado.Validar() != nil || operativo.Vinculo.ValidarPara(operativo.Resultado) != nil ||
		!operativo.Vinculo.VigenteEn(p.reloj.Ahora(), operativo.Resultado) ||
		operativo.Resultado.Contexto.PerfilActivoRef != perfil.perfilRef() {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	if err := comprobarPerfilConsultaAjustesCT(ctx, p.soporte, perfil); err != nil {
		return vacio, err
	}
	return contextoSeguridadComunDesarrollo{Vinculo: operativo.Vinculo, Resultado: operativo.Resultado}, nil
}

// La fuente de asignaciones distingue caída de una revocación efectiva.
// Se comprueba antes del PDP, que vuelve a contrastar la instantánea viva.
func comprobarPerfilConsultaAjustesCT(ctx context.Context, soporte *soporteAltaContratacionTemporalDesarrollo,
	perfil *perfilFijoCTDesarrollo) error {
	if soporte == nil || perfil == nil || ctx == nil || ctx.Err() != nil {
		return vecdomain.ErrAutorizacionDenegada
	}
	_, estado := soporte.consumirPerfilFijoCTDesarrolloConEstado(ctx, perfil)
	switch estado {
	case perfilFijoConsumoVigente:
		return nil
	case perfilFijoConsumoFuenteNoDisponible:
		return ajustesapp.ErrNoDisponible
	default:
		return vecdomain.ErrAutorizacionDenegada
	}
}

func (p *proveedorConsultaAjustesCT) ResolverContextoActor(ctx context.Context) (vecdomain.ContextoActor, error) {
	c, err := p.contexto(ctx)
	if err != nil {
		return vecdomain.ContextoActor{}, err
	}
	return c.Resultado.Contexto.Clonar()
}

func (p *proveedorConsultaAjustesCT) AutorizarAjustesReglasCT(ctx context.Context, actor vecdomain.ContextoActor,
	accion string, recurso vecdomain.RecursoAutorizable,
) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if accion != accionConsultarAjustesCT || !recursoConsultaAjustesCTValido(recurso) || actor.Validar() != nil {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	operativo, err := p.contexto(ctx)
	if err != nil {
		return vacio, err
	}
	huellaActor, err := actor.HuellaSHA256VinculadaV2()
	huellaActual, errActual := operativo.Resultado.Contexto.HuellaSHA256VinculadaV2()
	if err != nil || errActual != nil || huellaActor != huellaActual {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, err
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: operativo.Vinculo, ReferenciaMotivo: p.motivo,
		Accion: accion, Recurso: recurso, Finalidad: finalidadAjustesCT, Correlacion: correlacion,
	})
	if err != nil {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	datos, err := solicitud.Datos()
	if err != nil || !solicitudConsultaAjustesCTValida(datos) {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	decision, confirmacion, err := p.pdp.ExigirSolicitudLigadaV3(ctx, solicitud, operativo.Resultado)
	if err != nil {
		if errors.Is(err, vecports.ErrFuenteAutorizacionNoDisponible) {
			perfil := p.soporte.perfilFijoParaRutaYMetodo(rutaAjustesReglasCT, http.MethodGet)
			if estado := comprobarPerfilConsultaAjustesCT(ctx, p.soporte, perfil); estado != nil {
				return vacio, estado
			}
			return vacio, ajustesapp.ErrNoDisponible
		}
		return vacio, err
	}
	exportacion, err := p.material.proveerMaterialConfirmacion(ctx, solicitud, decision, confirmacion, p.motivo, operativo.Resultado)
	if err != nil || exportacion.ValidarEstructura() != nil {
		return vacio, errMotivoAutorizacionAjustes
	}
	resumen := exportacion.ResumenCapacidad()
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || resumen.Operacion() != accion || resumen.AudienciaConsumo() != audienciaAjustesCT ||
		resumen.EfectoRef() != recurso.Referencia || resumen.EfectoHuellaSHA256() != huella ||
		exportacion.PersonaVersion() != actor.Instantanea.PersonaVersion || exportacion.PerfilVersion() != actor.Instantanea.PerfilVersion {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	return exportacion, nil
}

func (p *proveedorConsultaAjustesCT) ComprobarCapacidadAjustesReglasCT(context.Context, vecdomain.ContextoActor,
	string, vecdomain.RecursoAutorizable) (bool, error) {
	return false, nil // CT110 no guarda aún la instantánea al iniciar el plazo.
}

func recursoConsultaAjustesCTValido(r vecdomain.RecursoAutorizable) bool {
	return r.Validar() == nil && r.Referencia == recursoReglasCT && r.ModuloID == "contratacion_temporal" &&
		r.Tipo == "catalogo_reglas" && len(r.Ambitos) == 1 &&
		r.Ambitos["organizacion_ref"] == organizacionAltaContratacionTemporalDesarrollo &&
		len(r.Atributos) == 1 && huellaMaterialPlantillasCTValida(r.Atributos["material_sha256"])
}

func solicitudConsultaAjustesCTValida(d vecdomain.DatosSolicitudAutorizacionLigadaV3) bool {
	return d.Accion == accionConsultarAjustesCT && d.Finalidad == finalidadAjustesCT && recursoConsultaAjustesCTValido(d.Recurso)
}

var _ ajustespg.ProveedorAutorizacionAjustesReglasCT = (*proveedorConsultaAjustesCT)(nil)
var _ ajusteshttp.ResolverActor = (*proveedorConsultaAjustesCT)(nil)

func descriptorFronteraConsultaAjustesCT(perfil string) descriptorFronteraComunDesarrollo {
	return descriptorFronteraComunDesarrollo{Clave: "ct-reglas-ajustes-consultar",
		Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: ajusteshttp.Ruta,
		PerfilesActivosRef: []string{perfil}, ClavePolitica: clavePoliticaContratacionTemporalDesarrollo,
		ClaveCapacidad: accionConsultarAjustesCT}
}

func descriptorMaterialConsultaAjustesCT() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{Audiencia: audienciaAjustesCT,
		Dominio: "vec.ct.ajustes-reglas.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:ct-ajustes-reglas:",
		ProveedorNominal: proveedorMaterialContratacionTemporal}
}

func preflightConsultaAjustesCT(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil || ctx == nil || ctx.Err() != nil {
		return errMotivoAutorizacionAjustes
	}
	var permitida bool
	err := pool.QueryRow(ctx, `SELECT has_function_privilege(current_user,
		to_regprocedure('vec_contratacion_temporal.operar_ajustes_reglas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'), 'EXECUTE')`).Scan(&permitida)
	if err != nil || !permitida {
		return errMotivoAutorizacionAjustes
	}
	return nil
}

func plazoConsultaAjustesContexto() time.Duration { return 15 * time.Second }
