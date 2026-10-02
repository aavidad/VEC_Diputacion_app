package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	reglasapp "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoreglasbaremo"
	bolsapuertos "vec-diputacion-granada/internal/modules/bolsa/ports"
	seguridad "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// RutasGobiernoReglasBaremoV3 son rutas exactas registradas por la raíz, una
// por operación. El broker las coteja con la capacidad mTLS y la frontera
// sellada, nunca con un nombre de ruta suministrado por el cliente.
type RutasGobiernoReglasBaremoV3 struct {
	Alta, Consulta, Recuperar string
}

func (r RutasGobiernoReglasBaremoV3) valida() bool {
	return rutaExactaGobiernoV3(r.Alta) && rutaExactaGobiernoV3(r.Consulta) && rutaExactaGobiernoV3(r.Recuperar) &&
		r.Alta != r.Consulta && r.Alta != r.Recuperar && r.Consulta != r.Recuperar
}
func rutaExactaGobiernoV3(r string) bool {
	return len(r) > 1 && len(r) <= 180 && strings.HasPrefix(r, "/") && !strings.ContainsAny(r, "*?#\\\r\n\t ")
}

type ProveedorGobiernoReglasBaremoV3 struct {
	perfil   *PerfilGobiernoReglasBaremoV3
	sesion   *proveedorSesionConsultaRRHHDesarrollo
	pdp      vecports.AutorizadorSolicitudLigadaV3
	material *proveedorMaterialAltaContratacionTemporalDesarrollo
	pool     *pgxpool.Pool
	rutas    RutasGobiernoReglasBaremoV3
	reloj    relojContratacionTemporalDesarrollo
}

// NuevoProveedorGobiernoReglasBaremoV3 no recibe un publicador. El perfil ya
// debe estar provisionado en el arranque/CLI y se lee de PostgreSQL en cada
// petición; las revocaciones o restricciones deniegan el consumo inmediato.
func NuevoProveedorGobiernoReglasBaremoV3(p *PerfilGobiernoReglasBaremoV3,
	sesion *proveedorSesionConsultaRRHHDesarrollo, pdp vecports.AutorizadorSolicitudLigadaV3,
	material *proveedorMaterialAltaContratacionTemporalDesarrollo, pool *pgxpool.Pool,
	rutas RutasGobiernoReglasBaremoV3, reloj relojContratacionTemporalDesarrollo,
) (*ProveedorGobiernoReglasBaremoV3, error) {
	if p == nil || p.soporte == nil || sesion == nil || sesion.soporte != p.soporte ||
		dependenciaEsNulaContratacionTemporalDesarrollo(pdp) || material == nil ||
		material.atestador == nil || material.confianza == nil || material.emisor == nil ||
		pool == nil || !rutas.valida() || dependenciaEsNulaContratacionTemporalDesarrollo(reloj) {
		return nil, errPerfilGobiernoReglasBaremoV3
	}
	return &ProveedorGobiernoReglasBaremoV3{p, sesion, pdp, material, pool, rutas, reloj}, nil
}

// Credenciales obtiene actor y vínculo únicamente de la sesión mTLS de esta
// petición. El holder común guarda exactamente ese recibo para la llamada
// posterior al proveedor de material; nunca se aceptan actor/rol por JSON.
func (p *ProveedorGobiernoReglasBaremoV3) Credenciales(ctx context.Context) (reglasapp.CredencialesGobiernoV3, error) {
	actual, err := p.contexto(ctx, "")
	if err != nil {
		return reglasapp.CredencialesGobiernoV3{}, err
	}
	return reglasapp.NuevasCredencialesGobiernoV3(actual.Resultado.Contexto, actual.Vinculo)
}

func (p *ProveedorGobiernoReglasBaremoV3) contexto(ctx context.Context, operacion string) (contextoSeguridadComunDesarrollo, error) {
	vacio := contextoSeguridadComunDesarrollo{}
	if p == nil || p.perfil == nil || p.sesion == nil || p.pdp == nil ||
		p.perfil.soporte == nil || ctx == nil || ctx.Err() != nil {
		return vacio, reglasapp.ErrGobiernoV3NoAutenticado
	}
	capacidad, ok := p.perfil.soporte.capacidadValida(ctx)
	frontera, valida := fronteraSeguridadComunDesdeContexto(ctx)
	ruta := capacidad.ruta
	accion := "bolsa.reglas_baremo.version.consultar"
	if ruta == p.rutas.Alta {
		accion = "bolsa.reglas_baremo.borrador.crear"
	} else if ruta == p.rutas.Recuperar {
		accion = "bolsa.reglas_baremo.recibo.consultar"
	}
	if !ok || !valida || frontera.ruta != ruta || frontera.metodo != http.MethodPost ||
		capacidad.metodo != http.MethodPost || frontera.descriptor.ClaveCapacidad != accion ||
		frontera.superficie != superficieInternaSeguridadComunDesarrollo ||
		!frontera.descriptor.admitePerfil(p.perfil.PerfilRef()) ||
		!p.rutaOperacion(ruta, operacion) || capacidad.principal.ID != p.perfil.soporte.principalID ||
		capacidad.principal.Attributes["certificate_sha256"] != p.perfil.soporte.certificadoSHA256 ||
		capacidad.contextoOperacion == nil {
		return vacio, reglasapp.ErrGobiernoV3NoAutenticado
	}
	holder := capacidad.contextoOperacion
	holder.mu.Lock()
	defer holder.mu.Unlock()
	if holder.soporte == nil {
		holder.soporte = p.perfil.soporte
		operativo, err := p.sesion.ResolverContexto(ctx)
		if err == nil && operativo.Resultado.Validar() == nil &&
			operativo.Vinculo.ValidarPara(operativo.Resultado) == nil &&
			mismoContextoEsperadoRegistradoDesarrollo(p.perfil.soporte.contextoEsperadoRegistrado, operativo.Resultado) {
			holder.contexto.Vinculo, holder.contexto.Resultado = operativo.Vinculo, operativo.Resultado
		} else {
			holder.err = reglasapp.ErrGobiernoV3NoAutenticado
		}
	}
	ahora := p.reloj.Ahora()
	if holder.soporte != p.perfil.soporte || holder.err != nil ||
		holder.contexto.Vinculo.ValidarPara(holder.contexto.Resultado) != nil ||
		!holder.contexto.Vinculo.VigenteEn(ahora, holder.contexto.Resultado) ||
		holder.contexto.Resultado.Contexto.Principal.AuthMethod != vecdomain.AuthMethodCertificate ||
		holder.contexto.Resultado.Contexto.Principal.AuthAssurance != vecdomain.AuthAssuranceHigh ||
		holder.contexto.Resultado.Contexto.PerfilActivoRef != p.perfil.PerfilRef() {
		return vacio, reglasapp.ErrGobiernoV3NoAutenticado
	}
	clon, err := holder.contexto.Resultado.Clonar()
	if err != nil {
		return vacio, reglasapp.ErrGobiernoV3NoAutenticado
	}
	return contextoSeguridadComunDesarrollo{Vinculo: holder.contexto.Vinculo, Resultado: clon}, nil
}

func (p *ProveedorGobiernoReglasBaremoV3) rutaOperacion(ruta, operacion string) bool {
	if p == nil || !p.rutas.valida() {
		return false
	}
	switch operacion {
	case "":
		return ruta == p.rutas.Alta || ruta == p.rutas.Consulta || ruta == p.rutas.Recuperar
	case "alta_borrador":
		return ruta == p.rutas.Alta
	case "consultar_exacta":
		return ruta == p.rutas.Consulta
	case "recuperar_recibo":
		return ruta == p.rutas.Recuperar
	default:
		return false
	}
}

func (p *ProveedorGobiernoReglasBaremoV3) ProveerMaterialGobiernoReglasV3(ctx context.Context,
	vinculo vecdomain.VinculoAutenticacionActorV2, pedido bolsapuertos.SolicitudMaterialGobiernoReglasV3,
) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	operativo, err := p.contexto(ctx, pedido.Operacion)
	if err != nil {
		return vacio, err
	}
	datos, err := vinculo.Datos()
	actuales, errActual := operativo.Vinculo.Datos()
	if err != nil || errActual != nil || datos != actuales || vinculo.ValidarPara(operativo.Resultado) != nil {
		return vacio, reglasapp.ErrGobiernoV3NoAutenticado
	}
	if err := p.validarPedido(pedido, operativo.Resultado.Contexto, p.reloj.Ahora()); err != nil {
		return vacio, err
	}
	publicada, existe, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, p.pool, p.perfil.PerfilRef())
	if err != nil {
		return vacio, reglasapp.ErrGobiernoV3NoDisponible
	}
	if !existe || publicada.actoAsignacion != actoAsignacionGobiernoReglasBaremoV3 ||
		publicada.actoControl != actoControlGobiernoReglasBaremoV3 {
		return vacio, reglasapp.ErrGobiernoV3Prohibido
	}
	if _, ok := instantaneaConsumible(publicada, p.perfil.plantilla, p.reloj.Ahora()); !ok {
		return vacio, reglasapp.ErrGobiernoV3Prohibido
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, reglasapp.ErrGobiernoV3NoDisponible
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: operativo.Vinculo, ReferenciaMotivo: pedido.Motivo,
		Accion: pedido.Accion, Recurso: pedido.Recurso, Finalidad: pedido.Finalidad, Correlacion: correlacion,
	})
	if err != nil {
		return vacio, reglasapp.ErrGobiernoV3Prohibido
	}
	decision, confirmacion, err := p.pdp.ExigirSolicitudLigadaV3(ctx, solicitud, operativo.Resultado)
	if errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		return vacio, reglasapp.ErrGobiernoV3Prohibido
	}
	if err != nil || decision.ValidarPara(solicitud) != nil {
		return vacio, reglasapp.ErrGobiernoV3NoDisponible
	}
	// La decisión del PDP ya debe contener el mínimo nominal; nunca se recorta
	// una decisión firmada para adaptarla a la operación SQL.
	if decision.ExigirProyeccionPara(solicitud, pedido.Campos, nil) != nil {
		return vacio, reglasapp.ErrGobiernoV3Prohibido
	}
	exportado, err := p.material.proveerMaterialConfirmacion(ctx, solicitud, decision, confirmacion, pedido.Motivo, operativo.Resultado)
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, operativo.Resultado,
		pedido.Motivo, exportado, reglasapp.AudienciaGobiernoBorradorReglasV3) {
		return vacio, reglasapp.ErrGobiernoV3NoDisponible
	}
	return exportado, nil
}

func (p *ProveedorGobiernoReglasBaremoV3) validarPedido(s bolsapuertos.SolicitudMaterialGobiernoReglasV3,
	actor vecdomain.ContextoActor, ahora time.Time) error {
	if p == nil || p.perfil == nil || s.Audiencia != reglasapp.AudienciaGobiernoBorradorReglasV3 ||
		actor.Validar() != nil || actor.Principal.AuthMethod != vecdomain.AuthMethodCertificate ||
		actor.Principal.AuthAssurance != vecdomain.AuthAssuranceHigh || s.Recurso.Validar() != nil ||
		!vecdomain.ReferenciaMotivoAutorizacionV2Valida(s.Motivo) ||
		len(s.Recurso.Ambitos) != 2 || len(s.Recurso.Atributos) != 1 ||
		s.Recurso.ModuloID != "bolsa" || s.Recurso.Ambitos["convocatoria_ref"] != p.perfil.convocatoriaRef ||
		s.Recurso.Ambitos["expediente_ref"] != p.perfil.expedienteRef ||
		len(s.MaterialCanonico) == 0 || len(s.MaterialCanonico) > 1<<20 {
		return reglasapp.ErrGobiernoV3Prohibido
	}
	var material reglasapp.MaterialGobiernoV3
	if json.Unmarshal(s.MaterialCanonico, &material) != nil {
		return reglasapp.ErrGobiernoV3Prohibido
	}
	canon, err := json.Marshal(material)
	motivo, errMotivo := vecdomain.RepresentacionCanonicaMotivoAutorizacionV2(s.Motivo)
	huella := sha256.Sum256(s.MaterialCanonico)
	if err != nil || errMotivo != nil || !bytes.Equal(canon, s.MaterialCanonico) ||
		!bytes.Equal(material.MotivoCanonico, motivo) ||
		s.Recurso.Atributos["material_sha256"] != hex.EncodeToString(huella[:]) ||
		material.Esquema != "vec.bolsa.gobierno-borrador.material.v3" || material.Operacion != s.Operacion ||
		material.Accion != s.Accion || material.ModuloID != s.Recurso.ModuloID ||
		material.TipoRecurso != s.Recurso.Tipo || material.Finalidad != s.Finalidad ||
		material.PersonaRef != actor.PersonaRef || material.PerfilRef != actor.PerfilActivoRef ||
		material.ConvocatoriaRef != p.perfil.convocatoriaRef || material.ExpedienteRef != p.perfil.expedienteRef {
		return reglasapp.ErrGobiernoV3Prohibido
	}
	solicitadaEn, err := time.Parse(time.RFC3339Nano, material.SolicitadaEn)
	if err != nil || solicitadaEn.Location() != time.UTC || solicitadaEn.After(ahora) ||
		ahora.Sub(solicitadaEn) > 90*time.Second {
		return reglasapp.ErrGobiernoV3Prohibido
	}
	switch s.Operacion {
	case "alta_borrador":
		if !(s.Accion == "bolsa.reglas_baremo.borrador.crear" && s.Finalidad == "gobierno_reglas_baremo" &&
			s.Recurso.Tipo == "intencion_gobierno_reglas_baremo" &&
			s.Recurso.Referencia == "intencion-reglas-baremo:"+material.HuellaSolicitudSHA256 &&
			shaHexGobiernoV3(material.HuellaSolicitudSHA256) &&
			slices.Equal(s.Campos, []string{"auditoria", "estado_reglas_baremo", "salida_eventos"}) &&
			len(material.VersionCanonica) > 0 && material.EstadoEsperado == nil &&
			claveOperacionGobiernoV3(material.ClaveOperacion)) {
			return reglasapp.ErrGobiernoV3Prohibido
		}
		return nil
	case "consultar_exacta", "recuperar_recibo":
		campos := []string{"estado_reglas_baremo"}
		accion := "bolsa.reglas_baremo.version.consultar"
		if s.Operacion == "recuperar_recibo" {
			campos = []string{"estado_reglas_baremo", "recibo"}
			accion = "bolsa.reglas_baremo.recibo.consultar"
		}
		claveYHuella := material.ClaveOperacion == "" && material.HuellaSolicitudSHA256 == ""
		if s.Operacion == "recuperar_recibo" {
			claveYHuella = claveOperacionGobiernoV3(material.ClaveOperacion) && shaHexGobiernoV3(material.HuellaSolicitudSHA256)
		}
		if !(s.Accion == accion && s.Finalidad == "consulta_gobierno_reglas_baremo" &&
			s.Recurso.Tipo == "version_reglas_baremo_gobernada" &&
			s.Recurso.Referencia == "reglas-baremo:"+material.Estado.HuellaEstadoSHA256 &&
			shaHexGobiernoV3(material.Estado.HuellaEstadoSHA256) &&
			slices.Equal(s.Campos, campos) && len(material.VersionCanonica) == 0 &&
			material.EstadoEsperado == nil && claveYHuella) {
			return reglasapp.ErrGobiernoV3Prohibido
		}
		return nil
	default:
		return reglasapp.ErrGobiernoV3Prohibido
	}
}

func claveOperacionGobiernoV3(v string) bool {
	if len(v) != 32 {
		return false
	}
	for _, c := range v {
		if c < '0' || c > '9' && c < 'a' || c > 'f' {
			return false
		}
	}
	return true
}

func shaHexGobiernoV3(v string) bool {
	if len(v) != sha256.Size*2 {
		return false
	}
	for _, c := range v {
		if c < '0' || c > '9' && c < 'a' || c > 'f' {
			return false
		}
	}
	return true
}

var _ bolsapuertos.ProveedorMaterialGobiernoReglasV3 = (*ProveedorGobiernoReglasBaremoV3)(nil)
