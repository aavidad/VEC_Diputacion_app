package bootstrap

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"

	ajusteshttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpapi/ajustesreglas"
	ajustespg "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ajustesapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/ajustesreglas"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	accionConsultarAjustesCT = "contratacion_temporal.reglas.consultar_ajustes"
	accionPublicarAjustesCT  = "contratacion_temporal.reglas.ajustar"
	finalidadAjustesCT       = "gobierno_reglas_contratacion_temporal"
)

func motivoAutorizacionAjustesCT() vecdomain.ReferenciaEntradaCatalogo {
	return vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion_ajustes_reglas_ct", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("motivos-autorizacion-ajustes-reglas-ct-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "gobernar-ajustes-reglas-ct")}
}

func ampliarInstantaneaAltaConAjustesCT(plantilla vecdomain.InstantaneaAutorizacion) (vecdomain.InstantaneaAutorizacion, error) {
	concesion := func(accion string, campos []string) vecdomain.ConcesionRol {
		return vecdomain.ConcesionRol{Accion: accion, ModuloID: "contratacion_temporal", TipoRecurso: "catalogo_reglas",
			Finalidades: []string{finalidadAjustesCT}, CamposPermitidos: campos, GarantiaMinima: vecdomain.AuthAssuranceHigh}
	}
	if plantilla.Validar() != nil || len(plantilla.VersionRol.Concesiones) != 1 ||
		plantilla.VersionRol.Concesiones[0].Accion != ctports.AccionCrearSolicitud {
		return vecdomain.InstantaneaAutorizacion{}, errMontajeAjustesReglasCT
	}
	plantilla.VersionRol.Concesiones = append(plantilla.VersionRol.Concesiones,
		concesion(accionConsultarAjustesCT, []string{"historial", "vigente"}),
		concesion(accionPublicarAjustesCT, []string{"ajustes", "recibo"}))
	if plantilla.Validar() != nil {
		return vecdomain.InstantaneaAutorizacion{}, errMontajeAjustesReglasCT
	}
	return plantilla, nil
}

func descriptoresFronteraAjustesReglasCT(perfil string) []descriptorFronteraComunDesarrollo {
	return []descriptorFronteraComunDesarrollo{
		{Clave: "ct-reglas-ajustes-consultar", Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo: http.MethodGet, Ruta: ajusteshttp.Ruta, PerfilesActivosRef: []string{perfil},
			ClavePolitica: clavePoliticaContratacionTemporalDesarrollo, ClaveCapacidad: accionConsultarAjustesCT},
		{Clave: "ct-reglas-ajustes-publicar", Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo: http.MethodPost, Ruta: ajusteshttp.Ruta, PerfilesActivosRef: []string{perfil},
			ClavePolitica: clavePoliticaContratacionTemporalDesarrollo, ClaveCapacidad: accionPublicarAjustesCT},
	}
}

func descriptorMaterialAjustesReglasCT() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{Audiencia: "vec_contratacion_temporal.ajustes_reglas.v1",
		Dominio: "vec.ct.ajustes-reglas.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:ct-reglas-ajustes:",
		ProveedorNominal: proveedorMaterialContratacionTemporal}
}

type proveedorAjustesReglasCT struct {
	soporte      *soporteAltaContratacionTemporalDesarrollo
	pdp          *aplicacionvec.ServicioAutorizacionSolicitudLigadaV3
	material     *proveedorMaterialAltaContratacionTemporalDesarrollo
	motivo       vecdomain.ReferenciaEntradaCatalogo
	reloj        relojContratacionTemporalDesarrollo
	poolPerfiles *pgxpool.Pool
	plantilla    vecdomain.InstantaneaAutorizacion
}

func (p *proveedorAjustesReglasCT) contextoVigente(ctx context.Context, accion string) (contextoSeguridadComunDesarrollo, error) {
	vacio := contextoSeguridadComunDesarrollo{}
	if p == nil || p.soporte == nil || p.pdp == nil || p.material == nil || contextoInterfazNulo(ctx) || ctx.Err() != nil {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	capacidad, valida := p.soporte.capacidadValida(ctx)
	frontera, fronteraValida := fronteraSeguridadComunDesdeContexto(ctx)
	metodo := capacidad.metodo
	esperada := accionConsultarAjustesCT
	if metodo == http.MethodPost {
		esperada = accionPublicarAjustesCT
	}
	perfil := p.plantilla.AsignacionPerfil.PerfilActivoRef
	if !valida || !fronteraValida || capacidad.ruta != ajusteshttp.Ruta ||
		(metodo != http.MethodGet && metodo != http.MethodPost) ||
		(accion == accionPublicarAjustesCT && metodo != http.MethodPost) ||
		frontera.ruta != ajusteshttp.Ruta || frontera.descriptor.Metodo != metodo ||
		frontera.descriptor.ClaveCapacidad != esperada ||
		!frontera.descriptor.admitePerfil(perfil) ||
		capacidad.principal.ID != p.soporte.principalID ||
		capacidad.principal.Attributes["certificate_sha256"] != p.soporte.certificadoSHA256 {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	publicada, encontrada, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, p.poolPerfiles, perfil)
	if err != nil {
		return vacio, ajustesapp.ErrNoDisponible
	}
	if !encontrada {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	if _, exacta := instantaneaConsumible(publicada, p.plantilla, p.reloj.Ahora()); !exacta {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	operativo, err := p.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil || operativo.Resultado.Validar() != nil ||
		operativo.Vinculo.ValidarPara(operativo.Resultado) != nil ||
		!operativo.Vinculo.VigenteEn(p.reloj.Ahora(), operativo.Resultado) ||
		operativo.Resultado.Contexto.PerfilActivoRef != perfil {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	return contextoSeguridadComunDesarrollo{Vinculo: operativo.Vinculo, Resultado: operativo.Resultado}, nil
}

func (p *proveedorAjustesReglasCT) ResolverContextoActor(ctx context.Context) (vecdomain.ContextoActor, error) {
	if p == nil || ctx == nil {
		return vecdomain.ContextoActor{}, vecdomain.ErrAutorizacionDenegada
	}
	capacidad, valida := p.soporte.capacidadValida(ctx)
	if !valida || capacidad.ruta != ajusteshttp.Ruta {
		return vecdomain.ContextoActor{}, vecdomain.ErrAutorizacionDenegada
	}
	accion := accionConsultarAjustesCT
	if capacidad.metodo == http.MethodPost {
		accion = accionPublicarAjustesCT
	}
	operativo, err := p.contextoVigente(ctx, accion)
	if err != nil {
		return vecdomain.ContextoActor{}, err
	}
	return operativo.Resultado.Contexto.Clonar()
}

func (p *proveedorAjustesReglasCT) AutorizarAjustesReglasCT(ctx context.Context, actor vecdomain.ContextoActor,
	accion string, recurso vecdomain.RecursoAutorizable) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || (accion != accionConsultarAjustesCT && accion != accionPublicarAjustesCT) ||
		recurso.Validar() != nil || recurso.Referencia != "vec.contratacion_temporal.reglas" ||
		recurso.ModuloID != "contratacion_temporal" || recurso.Tipo != "catalogo_reglas" ||
		len(recurso.Ambitos) != 1 || recurso.Ambitos["organizacion_ref"] != organizacionAltaContratacionTemporalDesarrollo ||
		len(recurso.Atributos) != 1 || !huellaMaterialPlantillasCTValida(recurso.Atributos["material_sha256"]) ||
		actor.Validar() != nil {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	operativo, err := p.contextoVigente(ctx, accion)
	if err != nil {
		return vacio, err
	}
	huellaActor, errActor := actor.HuellaSHA256VinculadaV2()
	huellaActual, errActual := operativo.Resultado.Contexto.HuellaSHA256VinculadaV2()
	if errActor != nil || errActual != nil || huellaActor != huellaActual || actor.Principal.ID != operativo.Resultado.Contexto.Principal.ID {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, ajustesapp.ErrNoDisponible
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: operativo.Vinculo, ReferenciaMotivo: p.motivo,
		Accion: accion, Recurso: recurso, Finalidad: finalidadAjustesCT, Correlacion: correlacion})
	if err != nil {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	decision, confirmacion, err := p.pdp.ExigirSolicitudLigadaV3(ctx, solicitud, operativo.Resultado)
	if err != nil {
		return vacio, err
	}
	exportacion, err := p.material.proveerMaterialConfirmacion(ctx, solicitud, decision, confirmacion, p.motivo, operativo.Resultado)
	if err != nil || exportacion.ValidarEstructura() != nil {
		return vacio, ajustesapp.ErrNoDisponible
	}
	resumen := exportacion.ResumenCapacidad()
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || resumen.Operacion() != accion || resumen.AudienciaConsumo() != "vec_contratacion_temporal.ajustes_reglas.v1" ||
		resumen.EfectoRef() != recurso.Referencia || resumen.EfectoHuellaSHA256() != huella ||
		exportacion.PersonaVersion() != actor.Instantanea.PersonaVersion || exportacion.PerfilVersion() != actor.Instantanea.PerfilVersion {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	return exportacion, nil
}

var _ ajustespg.ProveedorAutorizacionAjustesReglasCT = (*proveedorAjustesReglasCT)(nil)
var _ ajusteshttp.ResolverActor = (*proveedorAjustesReglasCT)(nil)
