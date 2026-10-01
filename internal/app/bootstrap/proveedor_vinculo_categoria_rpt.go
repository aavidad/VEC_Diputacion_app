package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	ctpostgres "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridad "vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	finalidadVinculoCategoriaRPTCT = "gestionar_vinculo_categoria_rpt_ct"
	finalidadLecturaCategoriaRPT   = "consultar_categorias_rpt"
)

// FuenteSesionVinculoCategoriaRPT obtiene las referencias exclusivamente de la
// frontera de identidad acreditada. El cuerpo HTTP no aporta perfil ni sesión.
type FuenteSesionVinculoCategoriaRPT interface {
	ResolverSesionVinculoCategoriaRPT(context.Context) (SesionVinculoCategoriaRPT, error)
}

type SesionVinculoCategoriaRPT struct {
	CT  ReferenciasSesionVinculoCategoriaRPT
	RPT ReferenciasSesionVinculoCategoriaRPT
}

type ReferenciasSesionVinculoCategoriaRPT struct {
	AutenticacionRef string
	SesionRef        string
}

// ConfiguracionProveedorVinculoCategoriaRPT es una publicación privada de la
// composición. Los tres motivos y las dos finalidades son entradas publicadas
// por sus autoridades, nunca valores elegidos por quien invoca la operación.
type ConfiguracionProveedorVinculoCategoriaRPT struct {
	OrganizacionRef, PerfilCT, PerfilRPT string
	CatalogoID, ModuloID                 string
	// ConsumidorRPT es la audiencia V3 exacta de AD3-117, no una reserva.
	ConsumidorRPT                                        string
	FinalidadCT, FinalidadRPT                            string
	MotivoConsultaCT, MotivoRegistroCT, MotivoLecturaRPT vecdomain.ReferenciaEntradaCatalogo
}

type ProveedorVinculoCategoriaRPT struct {
	config                             ConfiguracionProveedorVinculoCategoriaRPT
	sesiones                           FuenteSesionVinculoCategoriaRPT
	contextos                          ctports.ResolutorContextoAutorizacionAltaV3
	consultaCT, registroCT, lecturaRPT *confianza.EmisorMaterialAutorizacionAtestadaV3
	pool                               *pgxpool.Pool
	reloj                              ctports.Reloj
	// La producción usa siempre MaterialPublicacionCategoriaRPTPostgreSQL.
	materialRPT func(context.Context, *pgxpool.Pool, ctdomain.PublicacionCategoriaRPT) ([]byte, error)
}

var _ ctports.AutoridadVinculoCategoriaRPT = (*ProveedorVinculoCategoriaRPT)(nil)

func NuevoProveedorVinculoCategoriaRPT(
	config ConfiguracionProveedorVinculoCategoriaRPT,
	sesiones FuenteSesionVinculoCategoriaRPT,
	contextos ctports.ResolutorContextoAutorizacionAltaV3,
	consultaCT, registroCT, lecturaRPT *confianza.EmisorMaterialAutorizacionAtestadaV3,
	pool *pgxpool.Pool,
	reloj ctports.Reloj,
) (*ProveedorVinculoCategoriaRPT, error) {
	if !config.valida() || dependenciaBootstrapNula(sesiones) || dependenciaBootstrapNula(contextos) ||
		consultaCT == nil || registroCT == nil || lecturaRPT == nil || pool == nil || dependenciaBootstrapNula(reloj) {
		return nil, ctports.ErrVinculoCategoriaRPTNoDisponible
	}
	return &ProveedorVinculoCategoriaRPT{config: config, sesiones: sesiones, contextos: contextos,
		consultaCT: consultaCT, registroCT: registroCT, lecturaRPT: lecturaRPT,
		pool: pool, reloj: reloj, materialRPT: ctpostgres.MaterialPublicacionCategoriaRPTPostgreSQL}, nil
}

func (c ConfiguracionProveedorVinculoCategoriaRPT) valida() bool {
	return ctdomain.ReferenciaOpacaValida(c.OrganizacionRef) &&
		strings.HasPrefix(c.PerfilCT, "prf_") && strings.HasPrefix(c.PerfilRPT, "prf_") &&
		len(c.PerfilCT) > 4 && len(c.PerfilRPT) > 4 &&
		c.CatalogoID != "" && c.ModuloID != "" &&
		c.ConsumidorRPT == ctports.AudienciaConsultarPublicacionCategoriaRPT &&
		c.FinalidadCT == finalidadVinculoCategoriaRPTCT && c.FinalidadRPT == finalidadLecturaCategoriaRPT &&
		vecdomain.ReferenciaMotivoAutorizacionV2Valida(c.MotivoConsultaCT) &&
		vecdomain.ReferenciaMotivoAutorizacionV2Valida(c.MotivoRegistroCT) &&
		vecdomain.ReferenciaMotivoAutorizacionV2Valida(c.MotivoLecturaRPT) &&
		(vecdomain.RecursoAutorizable{Referencia: c.CatalogoID, ModuloID: c.ModuloID,
			Tipo: "catalogo_configurable", Ambitos: map[string]string{"catalogo_id": c.CatalogoID, "modulo_id": c.ModuloID}}).Validar() == nil
}

func (p *ProveedorVinculoCategoriaRPT) contextoCT(ctx context.Context) (ctports.ContextoAutorizacionAltaV3, error) {
	ct, _, err := p.contextosVigentes(ctx, false)
	return ct, err
}

func (p *ProveedorVinculoCategoriaRPT) contextosVigentes(ctx context.Context, exigirRPT bool) (ctports.ContextoAutorizacionAltaV3, ctports.ContextoAutorizacionAltaV3, error) {
	var cero ctports.ContextoAutorizacionAltaV3
	if p == nil || ctx == nil || ctx.Err() != nil || p.sesiones == nil || p.contextos == nil || p.reloj == nil {
		return cero, cero, ctports.ErrVinculoCategoriaRPTNoDisponible
	}
	sesion, err := p.sesiones.ResolverSesionVinculoCategoriaRPT(ctx)
	if err != nil {
		return cero, cero, errorProveedorVinculoRPT(err)
	}
	resolver := func(perfil string, referencias ReferenciasSesionVinculoCategoriaRPT) (ctports.ContextoAutorizacionAltaV3, error) {
		solicitud := ctports.SolicitudResolverContextoAutorizacionAltaV3{
			AutenticacionRef: referencias.AutenticacionRef, SesionRef: referencias.SesionRef, PerfilRef: perfil,
		}
		if solicitud.Validar() != nil {
			return cero, ctports.ErrVinculoCategoriaRPTDenegado
		}
		actual, e := p.contextos.ResolverContextoAutorizacionAltaV3(ctx, solicitud)
		if e != nil {
			return cero, errorProveedorVinculoRPT(e)
		}
		if actual.ValidarPara(solicitud, p.reloj.Ahora()) != nil {
			return cero, ctports.ErrVinculoCategoriaRPTDenegado
		}
		return actual, nil
	}
	ct, err := resolver(p.config.PerfilCT, sesion.CT)
	if err != nil {
		return cero, cero, err
	}
	if !exigirRPT {
		return ct, cero, nil
	}
	rpt, err := resolver(p.config.PerfilRPT, sesion.RPT)
	if err != nil {
		return cero, cero, err
	}
	ctV, e1 := ct.Vinculo.Datos()
	rptV, e2 := rpt.Vinculo.Datos()
	if e1 != nil || e2 != nil || ctV.PrincipalID != rptV.PrincipalID ||
		ctV.CuentaRef != rptV.CuentaRef || ctV.CuentaOrdinariaRef != rptV.CuentaOrdinariaRef ||
		ct.Resultado.Contexto.PersonaRef != rpt.Resultado.Contexto.PersonaRef ||
		ctV.PrincipalID != ct.Resultado.Contexto.Principal.ID || rptV.PrincipalID != rpt.Resultado.Contexto.Principal.ID ||
		ctV.CuentaRef != ct.Resultado.Contexto.Instantanea.CuentaRef || rptV.CuentaRef != rpt.Resultado.Contexto.Instantanea.CuentaRef ||
		ctV.PerfilActivoRef != p.config.PerfilCT || rptV.PerfilActivoRef != p.config.PerfilRPT {
		return cero, cero, ctports.ErrVinculoCategoriaRPTDenegado
	}
	return ct, rpt, nil
}

func (p *ProveedorVinculoCategoriaRPT) ConsultaCT(ctx context.Context, c ctports.ConsultaVinculoCategoriaRPT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var cero vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if p == nil || !p.config.valida() || c.OrganizacionRef != p.config.OrganizacionRef {
		return cero, ctports.ErrVinculoCategoriaRPTDenegado
	}
	b, err := c.Canonico()
	if err != nil {
		return cero, err
	}
	ct, err := p.contextoCT(ctx)
	if err != nil {
		return cero, err
	}
	return p.emitir(ctx, ct, p.consultaCT, p.config.MotivoConsultaCT, p.config.FinalidadCT,
		ctports.AccionConsultarVinculoCategoriaRPT, ctports.AudienciaConsultarVinculoCategoriaRPT,
		recursoVinculoCategoriaRPT(c.ExpedienteRef, "contratacion_temporal", "vinculo_categoria_rpt_ct",
			map[string]string{"organizacion_ref": c.OrganizacionRef}, b))
}

func (p *ProveedorVinculoCategoriaRPT) RegistroCT(ctx context.Context, m ctports.RegistroVinculoCategoriaRPT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var cero vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if p == nil || !p.config.valida() || m.OrganizacionRef != p.config.OrganizacionRef ||
		m.CatalogoID != p.config.CatalogoID || m.ModuloID != p.config.ModuloID {
		return cero, ctports.ErrVinculoCategoriaRPTDenegado
	}
	b, err := m.Canonico()
	if err != nil {
		return cero, err
	}
	ct, _, err := p.contextosVigentes(ctx, true)
	if err != nil {
		return cero, err
	}
	return p.emitir(ctx, ct, p.registroCT, p.config.MotivoRegistroCT, p.config.FinalidadCT,
		ctports.AccionRegistrarVinculoCategoriaRPT, ctports.AudienciaRegistrarVinculoCategoriaRPT,
		recursoVinculoCategoriaRPT(m.ExpedienteRef, "contratacion_temporal", "vinculo_categoria_rpt_ct",
			map[string]string{"organizacion_ref": m.OrganizacionRef}, b))
}

func (p *ProveedorVinculoCategoriaRPT) LecturaRPT(ctx context.Context, publicacion ctdomain.PublicacionCategoriaRPT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var cero vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if p == nil || !p.config.valida() || publicacion.Validar() != nil ||
		publicacion.CatalogoID != p.config.CatalogoID || publicacion.ModuloID != p.config.ModuloID ||
		publicacion.CategoriaID != publicacion.CategoriaClave {
		return cero, ctports.ErrVinculoCategoriaRPTDenegado
	}
	_, rpt, err := p.contextosVigentes(ctx, true)
	if err != nil {
		return cero, err
	}
	if p.materialRPT == nil {
		return cero, ctports.ErrVinculoCategoriaRPTNoDisponible
	}
	b, err := p.materialRPT(ctx, p.pool, publicacion)
	if err != nil {
		return cero, errorProveedorVinculoRPT(err)
	}
	if !materialPublicacionRPTValido(b, publicacion) {
		return cero, ctports.ErrVinculoCategoriaRPTDenegado
	}
	return p.emitir(ctx, rpt, p.lecturaRPT, p.config.MotivoLecturaRPT, p.config.FinalidadRPT,
		ctports.AccionConsultarPublicacionCategoriaRPT, p.config.ConsumidorRPT,
		recursoVinculoCategoriaRPT(publicacion.CatalogoID, publicacion.ModuloID, "catalogo_configurable",
			map[string]string{"catalogo_id": publicacion.CatalogoID, "modulo_id": publicacion.ModuloID}, b))
}

func materialPublicacionRPTValido(b []byte, p ctdomain.PublicacionCategoriaRPT) bool {
	if len(b) == 0 || len(b) > 4096 || !json.Valid(b) {
		return false
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(b, &campos) != nil || len(campos) != 5 {
		return false
	}
	esperados := map[string]any{"catalogo_id": p.CatalogoID, "modulo_id": p.ModuloID,
		"version": p.CatalogoVersion, "huella_sha256": p.CatalogoHuella, "categoria_id": p.CategoriaID}
	for clave, esperado := range esperados {
		raw, ok := campos[clave]
		if !ok {
			return false
		}
		var valor any
		if json.Unmarshal(raw, &valor) != nil {
			return false
		}
		if clave == "version" {
			if n, ok := valor.(float64); !ok || n != float64(p.CatalogoVersion) {
				return false
			}
		} else if valor != esperado {
			return false
		}
	}
	return true
}

func recursoVinculoCategoriaRPT(ref, modulo, tipo string, ambitos map[string]string, material []byte) vecdomain.RecursoAutorizable {
	h := sha256.Sum256(material)
	return vecdomain.RecursoAutorizable{Referencia: ref, ModuloID: modulo, Tipo: tipo,
		Ambitos: ambitos, Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
}

func (p *ProveedorVinculoCategoriaRPT) emitir(ctx context.Context, actor ctports.ContextoAutorizacionAltaV3,
	emisor *confianza.EmisorMaterialAutorizacionAtestadaV3, motivo vecdomain.ReferenciaEntradaCatalogo,
	finalidad, accion, audiencia string, recurso vecdomain.RecursoAutorizable,
) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var cero vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if emisor == nil || recurso.Validar() != nil || ctx == nil || ctx.Err() != nil {
		return cero, ctports.ErrVinculoCategoriaRPTNoDisponible
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return cero, ctports.ErrVinculoCategoriaRPTNoDisponible
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: actor.Vinculo, ReferenciaMotivo: motivo, Accion: accion,
		Recurso: recurso, Finalidad: finalidad, Correlacion: correlacion,
	})
	if err != nil {
		return cero, ctports.ErrVinculoCategoriaRPTDenegado
	}
	decision, confirmacion, exportador, err := emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, actor.Resultado)
	if err != nil {
		// El núcleo sólo distingue una denegación cuando también confirma
		// su registro explícito. Una caída del PDP no se presenta como 403.
		if errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) {
			return cero, ctports.ErrVinculoCategoriaRPTDenegado
		}
		return cero, ctports.ErrVinculoCategoriaRPTNoDisponible
	}
	if decision.ValidarPara(solicitud) != nil || exportador == nil {
		return cero, ctports.ErrVinculoCategoriaRPTDenegado
	}
	exportacion, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return cero, ctports.ErrVinculoCategoriaRPTNoDisponible
	}
	if !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, actor.Resultado, motivo, exportacion, audiencia) {
		return cero, ctports.ErrVinculoCategoriaRPTDenegado
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	resumen := exportacion.ResumenCapacidad()
	if err != nil || resumen.Operacion() != accion || resumen.AudienciaConsumo() != audiencia ||
		resumen.EfectoRef() != recurso.Referencia || resumen.EfectoHuellaSHA256() != huella ||
		exportacion.PersonaVersion() != actor.Resultado.Contexto.Instantanea.PersonaVersion ||
		exportacion.PerfilVersion() != actor.Resultado.Contexto.Instantanea.PerfilVersion {
		return cero, ctports.ErrVinculoCategoriaRPTDenegado
	}
	// La instantánea inmutable vuelve a contrastarse; un emisor no puede
	// sustituir el actor resuelto por el otro perfil.
	datos, e := solicitud.Datos()
	if e != nil || !reflect.DeepEqual(datos.Recurso, recurso) || !datos.VinculoAutenticacionActor.CoincideExactamenteCon(actor.Vinculo) {
		return cero, ctports.ErrVinculoCategoriaRPTDenegado
	}
	return exportacion, nil
}

func errorProveedorVinculoRPT(err error) error {
	if errors.Is(err, ctports.ErrVinculoCategoriaRPTDenegado) ||
		errors.Is(err, ctports.ErrAutorizacionDenegada) ||
		errors.Is(err, vecdomain.ErrAutorizacionDenegada) ||
		errors.Is(err, vecdomain.ErrContextoActorNoResuelto) {
		return ctports.ErrVinculoCategoriaRPTDenegado
	}
	return ctports.ErrVinculoCategoriaRPTNoDisponible
}
