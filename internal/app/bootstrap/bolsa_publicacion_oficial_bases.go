package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	appbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// F1-95. La declaración exige el consumidor del núcleo para su audiencia. Sin
// él, publicar la clave de capacidad falla, el proveedor queda nil y la ruta
// responde 503 sin aceptar nada; el resto de Bolsa sigue en marcha.
func proveedorPublicacionOficialBases(ctx context.Context, gobierno *pgxpool.Pool,
	material materialAtestacionContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo,
) *proveedorMaterialAltaContratacionTemporalDesarrollo {
	d := descriptorMaterialConsumidorV3Desarrollo{Audiencia: ports.AudienciaDeclararPublicacionOficialBases,
		Dominio:          "vec.bolsa.publicacion-oficial-bases.capacidad-v3",
		Prefijo:          "clave:capacidad:bolsa-publicacion-oficial-bases:",
		ProveedorNominal: "proveedor-material-publicacion-oficial-bases-bolsa"}
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo([]descriptorMaterialConsumidorV3Desarrollo{d})
	if err != nil {
		slog.Warn("publicacion_oficial_bases_no_habilitada", "etapa", "catalogo")
		return nil
	}
	p, err := nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogo, d.Audiencia)
	if err != nil {
		slog.Warn("publicacion_oficial_bases_no_habilitada", "etapa", "material")
		return nil
	}
	return p
}

func motivoPublicacionOficialBases() vd.ReferenciaEntradaCatalogo {
	return vd.ReferenciaEntradaCatalogo{CatalogoID: "motivos_publicacion_oficial_bases_bolsa", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-bolsa-publicacion-oficial-bases-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-publicacion-oficial-bases-declarar")}
}

// concesionPublicacionOficialBasesExacta es la forma exacta de la fuente
// gobernada bolsa_publicacion_oficial_bases_rrhh_v1: sin campos y auditada.
func concesionPublicacionOficialBasesExacta(c vd.ConcesionRol) bool {
	return c.Accion == ports.AccionDeclararPublicacionOficialBases && c.ModuloID == ports.ModuloPublicacionOficialBases &&
		c.TipoRecurso == ports.TipoRecursoVersionConvocatoria &&
		slices.Equal(c.Finalidades, []string{ports.FinalidadDeclararPublicacionOficialBases}) &&
		len(c.CamposPermitidos) == 0 && slices.Equal(c.Obligaciones, []string{"auditar"}) &&
		c.GarantiaMinima == vd.AuthAssuranceHigh && c.Validar() == nil
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararDeclaracionPublicacionOficial(ctx context.Context,
	d ports.DeclaracionPublicacionOficial,
) (ports.SolicitudDeclaracionPublicacionOficial, error) {
	var cero ports.SolicitudDeclaracionPublicacionOficial
	c, err := p.contextoRevalidado(ctx)
	if err != nil {
		return cero, vd.ErrAutorizacionDenegada
	}
	correlacion, err := vd.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.generar)
	if err != nil {
		return cero, ports.ErrPublicacionOficialNoDisponible
	}
	return ports.SolicitudDeclaracionPublicacionOficial{Vinculo: c.Vinculo, ResultadoContexto: c.Resultado,
		Correlacion: correlacion, MotivoAutorizacion: motivoPublicacionOficialBases(), Declaracion: d}, nil
}

// consultaPublicacionesOficialesBolsa lista solo las versiones cuyo ámbito
// cubre la concesión de declarar del perfil activo, con la misma evaluación
// pura que la decisión V3 (sin emitir decisión). Cada consulta queda auditada.
type consultaPublicacionesOficialesBolsa struct {
	preparador  *preparadorBorradorLlamamientoDesarrollo
	fuente      vp.FuenteAutorizacion
	repositorio ports.RepositorioPublicacionOficial
	registrador vp.RegistradorIntentosAuditoria
	proceso     string
	ahora       func() time.Time
}

func (c *consultaPublicacionesOficialesBolsa) ConsultarPublicacionesOficiales(ctx context.Context) (pagina ports.PaginaVersionesPublicacionOficial, fallo error) {
	if c == nil || c.preparador == nil || c.fuente == nil || c.repositorio == nil || c.registrador == nil ||
		c.ahora == nil || c.proceso == "" || ctx == nil || ctx.Err() != nil {
		return pagina, ports.ErrPublicacionOficialNoDisponible
	}
	actor, err := c.preparador.contextoRevalidado(ctx)
	if err != nil {
		return pagina, vd.ErrAutorizacionDenegada
	}
	correlacion, err := vd.GenerarReferenciaCorrelacionAutorizacionV2(ctx, c.preparador.generar)
	if err != nil {
		return pagina, ports.ErrPublicacionOficialNoDisponible
	}
	defer func() {
		resultado := vd.ResultadoIntentoAuditoriaExito
		if fallo != nil || ctx.Err() != nil {
			resultado = vd.ResultadoIntentoAuditoriaError
			if errors.Is(fallo, vd.ErrAutorizacionDenegada) {
				resultado = vd.ResultadoIntentoAuditoriaDenegado
			}
		}
		if err := c.auditar(ctx, actor, correlacion, resultado); err != nil {
			slog.Error("bolsa publicacion oficial: auditoria no confirmada", "causa", err)
			pagina, fallo = ports.PaginaVersionesPublicacionOficial{}, ports.ErrPublicacionOficialNoDisponible
		}
	}()
	ahora := c.ahora().UTC().Truncate(time.Microsecond)
	datos, err := actor.Vinculo.Datos()
	if err != nil || !actor.Vinculo.VigenteEn(ahora, actor.Resultado) || datos.Superficie != "interna_corporativa" || datos.CuentaPrivilegiada {
		return pagina, vd.ErrAutorizacionDenegada
	}
	instantanea, err := c.fuente.ObtenerInstantaneaAutorizacion(ctx, datos.PrincipalID, datos.PerfilActivoRef)
	if err != nil {
		return pagina, vd.ErrAutorizacionDenegada
	}
	if instantanea.Validar() != nil || !slices.ContainsFunc(instantanea.VersionRol.Concesiones, concesionPublicacionOficialBasesExacta) ||
		instantanea.AsignacionPerfil.EmitidaPor == "identidad:desarrollo:no-autoritativa" ||
		instantanea.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" ||
		instantanea.ControlVigenciaVersionRol.ActualizadoPor == "seguridad:desarrollo:no-autoritativa" {
		return pagina, vd.ErrAutorizacionDenegada
	}
	leidas, err := c.repositorio.ConsultarVersiones(ctx, "", ports.LimiteConsultaPublicacionesOficialesBases)
	if err != nil {
		return pagina, err
	}
	pagina = ports.PaginaVersionesPublicacionOficial{Items: []ports.VersionPublicacionOficial{}, Completa: leidas.Completa}
	if len(leidas.Items) == 0 {
		return pagina, nil
	}
	consultas := make([]vd.ConsultaCapacidadInformativaV3, 0, len(leidas.Items))
	for _, v := range leidas.Items {
		consultas = append(consultas, vd.ConsultaCapacidadInformativaV3{Accion: ports.AccionDeclararPublicacionOficialBases,
			Finalidad: ports.FinalidadDeclararPublicacionOficialBases,
			Recurso: vd.RecursoAutorizable{Referencia: v.ConvocatoriaRef, ModuloID: ports.ModuloPublicacionOficialBases,
				Tipo: ports.TipoRecursoVersionConvocatoria, Ambitos: map[string]string{"unidad_ref": v.UnidadRef, "ambito_ref": v.AmbitoRef}}})
	}
	ahora = c.ahora().UTC().Truncate(time.Microsecond)
	evaluadas, _, err := vd.EvaluarCapacidadesInformativasV3(actor.Vinculo, actor.Resultado, instantanea, consultas, ahora)
	if err != nil || len(evaluadas) != len(leidas.Items) {
		return ports.PaginaVersionesPublicacionOficial{}, vd.ErrAutorizacionDenegada
	}
	for i, e := range evaluadas {
		if e.Concedida && len(e.CamposPermitidos) == 0 && (len(e.Obligaciones) == 0 || slices.Equal(e.Obligaciones, []string{"auditar"})) {
			pagina.Items = append(pagina.Items, leidas.Items[i])
		}
	}
	return pagina, nil
}

func (c *consultaPublicacionesOficialesBolsa) auditar(ctx context.Context, actor contextoSeguridadComunDesarrollo,
	correlacion vd.ReferenciaCorrelacionAutorizacionV2, resultado vd.ResultadoIntentoAuditoria,
) error {
	protegido, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelar()
	ref, err := vp.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return err
	}
	valor, err := correlacion.ValorCanonico()
	if err != nil {
		return err
	}
	orden, err := vp.NuevaOrdenIntentoAuditoria(ref, actor.Resultado, actor.Vinculo, vd.DatosIntentoAuditoria{
		Accion: "lectura_bolsas_rrhh_consultar", ModuloID: "bolsa", RecursoRef: "bolsa:convocatorias:publicaciones-oficiales",
		FinalidadRef: "gestion_bolsa_rrhh", Resultado: resultado, Motivo: motivoPublicacionOficialBases(),
		Proceso: c.proceso, Canal: "interna_corporativa", CorrelacionRef: valor})
	if err != nil {
		return err
	}
	acuse, err := c.registrador.AppendIntentoAuditoria(protegido, orden)
	if err != nil {
		return err
	}
	return acuse.ValidarPara(orden)
}

// nuevaRutaPublicacionOficialBases monta siempre la ruta. La lectura necesita
// la fuente y la auditoría; la declaración, además, el emisor de su audiencia.
func nuevaRutaPublicacionOficialBases(pool *pgxpool.Pool, preparador *preparadorBorradorLlamamientoDesarrollo,
	emisor *emisorBorradorLlamamientoDesarrollo, fuente vp.FuenteAutorizacion,
	registrador vp.RegistradorIntentosAuditoria, proceso string, ahora func() time.Time,
) vechttp.RutaExacta {
	var consultor bolsahttp.ConsultorPublicacionesOficiales
	var servicio ports.ServicioPublicacionOficial
	repositorio, err := postgresbolsa.NuevoRepositorioPublicacionOficialPostgreSQL(pool)
	if err == nil && preparador != nil && fuente != nil && registrador != nil && proceso == "vec-rrhh" && ahora != nil {
		consultor = &consultaPublicacionesOficialesBolsa{preparador, fuente, repositorio, registrador, proceso, ahora}
	}
	if err == nil && emisor != nil && emisor.publicacionOficial != nil {
		if s, e := appbolsa.NuevoServicioPublicacionOficialBases(emisor, repositorio); e == nil {
			servicio = s
		}
	}
	if servicio == nil {
		slog.Warn("publicacion_oficial_bases_no_habilitada", "etapa", "composicion")
	}
	var preparadorHTTP bolsahttp.PreparadorPublicacionOficial
	if preparador != nil {
		preparadorHTTP = preparador
	}
	return vechttp.RutaExacta{Ruta: bolsahttp.RutaPublicacionesOficialesBases,
		Manejador: bolsahttp.NuevoHandlerPublicacionesOficialesBases(consultor, preparadorHTTP, servicio)}
}
