package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	importacionpg "vec-diputacion-granada/internal/modules/bolsa/adapters/postgresimportacionconvoca"
	protector "vec-diputacion-granada/internal/modules/bolsa/adapters/protectorstagingdesarrollo"
	xls "vec-diputacion-granada/internal/modules/bolsa/adapters/xlsconvoca"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/modules/bolsa/application/constitucion"
	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type preparadorCargaConvocaBolsa struct {
	base *preparadorBorradorLlamamientoDesarrollo
	pdp  *autorizadorComunDesarrollo
	cfg  config.Config
}

// La vista previa verifica la concesión propia antes de leer el fichero. La
// referencia opaca de esta comprobación no representa ningún acta importada.
func (p *preparadorCargaConvocaBolsa) PrepararVistaPreviaCargaConvoca(ctx context.Context) error {
	if p == nil || p.base == nil || p.pdp == nil || ctx == nil {
		return puertosbolsa.ErrCargaConvocaNoDisponible
	}
	seguridad, err := p.base.contextoRevalidado(ctx)
	if err != nil {
		return err
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.base.generar)
	if err != nil {
		return puertosbolsa.ErrCargaConvocaNoDisponible
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: "acta:importacion-convoca:" + "0000000000000000000000000000000000000000000000000000000000000000",
		ModuloID:   puertosbolsa.ModuloCargaConvoca, Tipo: puertosbolsa.TipoRecursoCargaConvoca,
		Ambitos: map[string]string{"unidad_ref": p.base.soporte.unidadRef, "ambito_ref": p.base.soporte.ambitoRef},
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: seguridad.Vinculo, ReferenciaMotivo: motivoConfirmarCargaConvocaBolsaDesarrollo(),
		Accion: puertosbolsa.AccionConfirmarCargaConvoca, Recurso: recurso,
		Finalidad: puertosbolsa.FinalidadConfirmarCargaConvoca, Correlacion: correlacion,
	})
	if err != nil {
		return puertosbolsa.ErrCargaConvocaNoDisponible
	}
	decision, _, err := p.pdp.ExigirSolicitudLigadaV3(ctx, solicitud, seguridad.Resultado)
	if errors.Is(err, puertosvec.ErrDenegacionExplicitaAutorizacionLigadaV3) || errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		return dominiovec.ErrAutorizacionDenegada
	}
	if err != nil || decision.ValidarPara(solicitud) != nil {
		return puertosbolsa.ErrCargaConvocaNoDisponible
	}
	return nil
}

func (p *preparadorCargaConvocaBolsa) PrepararConfirmacionCargaConvoca(ctx context.Context, entrada bolsahttp.EntradaConfirmarCargaConvoca) (puertosbolsa.SolicitudConfirmarCargaConvoca, error) {
	if p == nil || p.base == nil || ctx == nil {
		return puertosbolsa.SolicitudConfirmarCargaConvoca{}, puertosbolsa.ErrCargaConvocaNoDisponible
	}
	seguridad, err := p.base.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.SolicitudConfirmarCargaConvoca{}, err
	}
	categoriaRef, err := validarCategoriaImportacionConvoca(p.cfg, entrada.CategoriaClave)
	if err != nil {
		return puertosbolsa.SolicitudConfirmarCargaConvoca{}, bolsahttp.ErrCategoriaCargaConvocaNoValida
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.base.generar)
	if err != nil {
		return puertosbolsa.SolicitudConfirmarCargaConvoca{}, puertosbolsa.ErrCargaConvocaNoDisponible
	}
	return puertosbolsa.SolicitudConfirmarCargaConvoca{
		Vinculo: seguridad.Vinculo, ResultadoContexto: seguridad.Resultado, Correlacion: correlacion,
		MotivoAutorizacion: motivoConfirmarCargaConvocaBolsaDesarrollo(), CategoriaRef: categoriaRef,
		// Constitucion deriva la referencia estable del acta. Dos libros de la
		// misma categoria y fecha reciben bolsas distintas.
		BolsaRef:      "",
		NombreFichero: entrada.NombreFichero, Contenido: entrada.Contenido,
	}, nil
}

type importadorCargaConvocaBolsa struct {
	servicio    *importacionapp.Servicio
	recuperador *importacionpg.RepositorioRecuperacionPostgreSQL
}

func (i importadorCargaConvocaBolsa) ActaImportada(ctx context.Context, huella, categoria string) (bool, error) {
	if i.recuperador == nil {
		return false, puertosbolsa.ErrCargaConvocaNoDisponible
	}
	_, existe, err := i.recuperador.ConsultarEstado(ctx, huella, categoria)
	return existe, err
}

func (i importadorCargaConvocaBolsa) Importar(ctx context.Context, solicitud importacionapp.SolicitudImportacion) (importacionapp.ResultadoImportacion, error) {
	if i.servicio == nil {
		return importacionapp.ResultadoImportacion{}, puertosbolsa.ErrCargaConvocaNoDisponible
	}
	return i.servicio.Importar(ctx, solicitud)
}

type custodioCargaConvocaBolsa struct{ cfg config.Config }

func (c custodioCargaConvocaBolsa) Custodiar(ctx context.Context, contenido []byte) (string, error) {
	if ctx == nil || ctx.Err() != nil {
		return "", puertosbolsa.ErrCargaConvocaNoDisponible
	}
	return custodiarImportacionConvocaDesarrollo(c.cfg, contenido)
}

type operadorCargaConvocaBolsa struct {
	vista    *aplicacionbolsa.PrevisualizadorCargaConvoca
	servicio *aplicacionbolsa.ServicioCargaConvoca
}

func (o operadorCargaConvocaBolsa) Previsualizar(ctx context.Context, nombre string, contenido []byte) (aplicacionbolsa.VistaPreviaCargaConvoca, error) {
	if o.vista == nil {
		return aplicacionbolsa.VistaPreviaCargaConvoca{}, puertosbolsa.ErrCargaConvocaNoDisponible
	}
	return o.vista.Previsualizar(ctx, nombre, contenido)
}

func (o operadorCargaConvocaBolsa) Confirmar(ctx context.Context, solicitud puertosbolsa.SolicitudConfirmarCargaConvoca, excluirConErrores bool) (aplicacionbolsa.ResultadoCargaConvoca, error) {
	if o.servicio == nil {
		return aplicacionbolsa.ResultadoCargaConvoca{}, puertosbolsa.ErrCargaConvocaNoDisponible
	}
	return o.servicio.Confirmar(ctx, solicitud, excluirConErrores)
}

type auditorCargaConvocaBolsa struct {
	preparador  *preparadorBorradorLlamamientoDesarrollo
	registrador puertosvec.RegistradorIntentosAuditoria
	proceso     string
}

func (a *auditorCargaConvocaBolsa) RegistrarIntentoFallidoCargaConvoca(ctx context.Context, operacion string, fallo error) error {
	if a == nil || a.preparador == nil || a.registrador == nil || ctx == nil || fallo == nil {
		return puertosvec.ErrIntentoAuditoriaNoDisponible
	}
	seguridad, err := a.preparador.contextoRevalidado(ctx)
	if err != nil {
		return puertosvec.ErrIntentoAuditoriaNoDisponible
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, a.preparador.generar)
	if err != nil {
		return puertosvec.ErrIntentoAuditoriaNoDisponible
	}
	valorCorrelacion, err := correlacion.ValorCanonico()
	if err != nil {
		return puertosvec.ErrIntentoAuditoriaNoDisponible
	}
	ref, err := puertosvec.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return err
	}
	resultado := dominiovec.ResultadoIntentoAuditoriaError
	if errors.Is(fallo, dominiovec.ErrAutorizacionDenegada) || errors.Is(fallo, dominiovec.ErrPermissionDenied) {
		resultado = dominiovec.ResultadoIntentoAuditoriaDenegado
	}
	recurso := "carga:convoca:confirmacion"
	if operacion == bolsahttp.OperacionVistaPreviaCargaConvoca {
		recurso = "carga:convoca:vista_previa"
	} else if operacion != bolsahttp.OperacionConfirmarCargaConvoca {
		return puertosvec.ErrIntentoAuditoriaNoDisponible
	}
	orden, err := puertosvec.NuevaOrdenIntentoAuditoria(ref, seguridad.Resultado, seguridad.Vinculo, dominiovec.DatosIntentoAuditoria{
		Accion: puertosbolsa.AccionConfirmarCargaConvoca, ModuloID: puertosbolsa.ModuloCargaConvoca,
		RecursoRef: recurso, FinalidadRef: puertosbolsa.FinalidadConfirmarCargaConvoca,
		Resultado: resultado, Motivo: motivoConfirmarCargaConvocaBolsaDesarrollo(), Proceso: a.proceso,
		Canal: string(dominiovec.SuperficieAutenticacionInternaCorporativaV1), CorrelacionRef: valorCorrelacion,
	})
	if err != nil {
		return err
	}
	acuse, err := a.registrador.AppendIntentoAuditoria(context.WithoutCancel(ctx), orden)
	if err != nil || acuse.ValidarPara(orden) != nil {
		return puertosvec.ErrIntentoAuditoriaNoDisponible
	}
	return nil
}

func nuevoHandlerCargaConvocaBolsaDesarrollo(ctx context.Context, cfg config.Config,
	poolBolsa *pgxpool.Pool, preparador *preparadorBorradorLlamamientoDesarrollo,
	emisor *emisorBorradorLlamamientoDesarrollo, pdp *autorizadorComunDesarrollo,
	registrador puertosvec.RegistradorIntentosAuditoria, proceso string, reloj func() time.Time,
) (http.Handler, func(), error) {
	if ctx == nil || poolBolsa == nil || preparador == nil || emisor == nil || emisor.cargaConvoca == nil || pdp == nil || registrador == nil || reloj == nil {
		return nil, nil, puertosbolsa.ErrCargaConvocaNoDisponible
	}
	material, err := cargarMaterialSeguridadDesarrollo(cfg)
	if err != nil {
		return nil, nil, err
	}
	defer borrarMaterialImportacionConvoca(material)
	protectorStaging, err := protector.Nuevo(material.claveKMS)
	if err != nil {
		return nil, nil, err
	}
	derivador, err := protector.NuevoDerivadorCandidato(material.claveKMS)
	if err != nil {
		return nil, nil, err
	}
	poolImportacion, err := abrirPoolImportacionConvoca(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}
	cerrar := func() { poolImportacion.Close() }
	fallar := func(err error) (http.Handler, func(), error) { cerrar(); return nil, nil, err }
	repoImportacion, err := importacionpg.NuevoRepositorioPostgreSQL(poolImportacion, protectorStaging)
	if err != nil {
		return fallar(err)
	}
	recuperador, err := importacionpg.NuevoRepositorioRecuperacionPostgreSQL(poolImportacion, protectorStaging)
	if err != nil {
		return fallar(err)
	}
	repoConstitucion, err := postgresbolsa.NuevoRepositorioConstitucionPostgreSQL(poolBolsa)
	if err != nil {
		return fallar(err)
	}
	lector := xls.NuevoLectorConLimiteFilas(aplicacionbolsa.MaximoFilasCargaConvoca)
	importador, err := importacionapp.NuevoServicio(lector, repoImportacion, reloj)
	if err != nil {
		return fallar(err)
	}
	constituidorBase, err := constitucion.NuevoServicio(recuperador, repoConstitucion, derivador, reloj)
	if err != nil {
		return fallar(err)
	}
	constituidor, err := constitucion.NuevoServicioAutorizado(constituidorBase, repoConstitucion)
	if err != nil {
		return fallar(err)
	}
	vista, err := aplicacionbolsa.NuevoPrevisualizadorCargaConvoca(lector)
	if err != nil {
		return fallar(err)
	}
	servicio, err := aplicacionbolsa.NuevoServicioCargaConvoca(vista, preparador, emisor,
		custodioCargaConvocaBolsa{cfg}, importadorCargaConvocaBolsa{importador, recuperador}, constituidor, reloj)
	if err != nil {
		return fallar(err)
	}
	handler, err := bolsahttp.NuevoHandlerCargaConvoca(
		&preparadorCargaConvocaBolsa{base: preparador, pdp: pdp, cfg: cfg}, operadorCargaConvocaBolsa{vista: vista, servicio: servicio},
		&auditorCargaConvocaBolsa{preparador: preparador, registrador: registrador, proceso: proceso})
	if err != nil {
		return fallar(err)
	}
	return handler, cerrar, nil
}
