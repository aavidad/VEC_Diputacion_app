package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
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

// La petición conserva solo la identidad revalidada y la correlación acuñada
// por el servidor. Un fallo posterior no debe provocar otra resolución de
// sesión, que podría perder al actor tras una revocación o cancelación.
type claveIntentoCargaConvocaBolsa struct{}

type intentoCargaConvocaBolsa struct {
	mu          sync.Mutex
	seguridad   contextoSeguridadComunDesarrollo
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2
	recurso     string
	verificada  bool
}

func capturarIntentoCargaConvocaBolsa(ctx context.Context, seguridad contextoSeguridadComunDesarrollo, correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2) {
	intento, ok := ctx.Value(claveIntentoCargaConvocaBolsa{}).(*intentoCargaConvocaBolsa)
	if !ok || intento == nil || seguridad.Resultado.Validar() != nil ||
		seguridad.Vinculo.ValidarPara(seguridad.Resultado) != nil {
		return
	}
	intento.mu.Lock()
	intento.seguridad, intento.correlacion, intento.verificada = seguridad, correlacion, true
	intento.mu.Unlock()
}

func intentoVerificadoCargaConvocaBolsa(ctx context.Context) (contextoSeguridadComunDesarrollo, dominiovec.ReferenciaCorrelacionAutorizacionV2, bool) {
	if ctx == nil {
		return contextoSeguridadComunDesarrollo{}, dominiovec.ReferenciaCorrelacionAutorizacionV2{}, false
	}
	intento, ok := ctx.Value(claveIntentoCargaConvocaBolsa{}).(*intentoCargaConvocaBolsa)
	if !ok || intento == nil {
		return contextoSeguridadComunDesarrollo{}, dominiovec.ReferenciaCorrelacionAutorizacionV2{}, false
	}
	intento.mu.Lock()
	defer intento.mu.Unlock()
	return intento.seguridad, intento.correlacion, intento.verificada
}

func capturarActaCargaConvocaBolsa(ctx context.Context, contenido []byte, categoriaRef string) {
	intento, ok := ctx.Value(claveIntentoCargaConvocaBolsa{}).(*intentoCargaConvocaBolsa)
	if !ok || intento == nil || len(contenido) == 0 || categoriaRef == "" {
		return
	}
	huella := sha256.Sum256(contenido)
	acta := importacionapp.ReferenciaActa(hex.EncodeToString(huella[:]), categoriaRef)
	intento.mu.Lock()
	if intento.verificada {
		intento.recurso = acta
	}
	intento.mu.Unlock()
}

func recursoActaCargaConvocaBolsa(ctx context.Context) (string, bool) {
	intento, ok := ctx.Value(claveIntentoCargaConvocaBolsa{}).(*intentoCargaConvocaBolsa)
	if !ok || intento == nil {
		return "", false
	}
	intento.mu.Lock()
	defer intento.mu.Unlock()
	return intento.recurso, intento.verificada && intento.recurso != ""
}

func correlacionIntentoCargaConvocaBolsa(ctx context.Context, generador interface {
	NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error)
}) (dominiovec.ReferenciaCorrelacionAutorizacionV2, error) {
	if _, correlacion, ok := intentoVerificadoCargaConvocaBolsa(ctx); ok && correlacion.Validar() == nil {
		return correlacion, nil
	}
	return dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, generador)
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
	correlacion, err := correlacionIntentoCargaConvocaBolsa(ctx, p.base.generar)
	if err != nil {
		return puertosbolsa.ErrCargaConvocaNoDisponible
	}
	capturarIntentoCargaConvocaBolsa(ctx, seguridad, correlacion)
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
	correlacion, err := correlacionIntentoCargaConvocaBolsa(ctx, p.base.generar)
	if err != nil {
		return puertosbolsa.SolicitudConfirmarCargaConvoca{}, puertosbolsa.ErrCargaConvocaNoDisponible
	}
	capturarIntentoCargaConvocaBolsa(ctx, seguridad, correlacion)
	categoriaRef, err := validarCategoriaImportacionConvoca(p.cfg, entrada.CategoriaClave)
	if err != nil {
		return puertosbolsa.SolicitudConfirmarCargaConvoca{}, bolsahttp.ErrCategoriaCargaConvocaNoValida
	}
	capturarActaCargaConvocaBolsa(ctx, entrada.Contenido, categoriaRef)
	return puertosbolsa.SolicitudConfirmarCargaConvoca{
		Vinculo: seguridad.Vinculo, ResultadoContexto: seguridad.Resultado, Correlacion: correlacion,
		MotivoAutorizacion: motivoConfirmarCargaConvocaBolsaDesarrollo(), CategoriaRef: categoriaRef,
		// Constitucion deriva la referencia estable del acta. Dos libros de la
		// misma categoria y fecha reciben bolsas distintas.
		BolsaRef:      "",
		NombreFichero: entrada.NombreFichero, Contenido: entrada.Contenido,
	}, nil
}

type operadorCargaConvocaBolsa struct {
	vista    *aplicacionbolsa.PrevisualizadorCargaConvoca
	servicio *aplicacionbolsa.ServicioCargaConvoca
}

func (o operadorCargaConvocaBolsa) Previsualizar(ctx context.Context, nombre string, contenido []byte) (aplicacionbolsa.VistaPreviaCargaConvoca, error) {
	if o.vista == nil {
		return aplicacionbolsa.VistaPreviaCargaConvoca{}, puertosbolsa.ErrCargaConvocaNoDisponible
	}
	formato, err := xls.FormatoContenido(contenido)
	if err != nil || !strings.EqualFold(filepath.Ext(nombre), "."+formato) {
		return aplicacionbolsa.VistaPreviaCargaConvoca{}, aplicacionbolsa.ErrFicheroCargaConvocaInvalido
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
	seguridad, correlacion, ok := intentoVerificadoCargaConvocaBolsa(ctx)
	if !ok {
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
	if acta, capturada := recursoActaCargaConvocaBolsa(ctx); capturada {
		recurso = acta
	} else if fichero, capturado := bolsahttp.RecursoIntentoCargaConvoca(ctx); capturado {
		recurso = fichero
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
	repoCarga, err := postgresbolsa.NuevoRepositorioCargaConvocaPostgreSQL(poolBolsa, protectorStaging)
	if err != nil {
		return nil, nil, err
	}
	lector := xls.NuevoLectorConLimiteFilas(aplicacionbolsa.MaximoFilasCargaConvoca)
	preparadorLote, err := importacionapp.NuevoPreparador(lector, reloj)
	if err != nil {
		return nil, nil, err
	}
	constituidor, err := constitucion.NuevoServicioAutorizado(derivador, reloj, repoCarga)
	if err != nil {
		return nil, nil, err
	}
	original, err := protector.NuevoProtectorOriginal(material.claveKMS)
	if err != nil {
		return nil, nil, err
	}
	vista, err := aplicacionbolsa.NuevoPrevisualizadorCargaConvoca(lector)
	if err != nil {
		return nil, nil, err
	}
	servicio, err := aplicacionbolsa.NuevoServicioCargaConvoca(vista, preparador, emisor,
		original, preparadorLote, constituidor, reloj)
	if err != nil {
		return nil, nil, err
	}
	handler, err := bolsahttp.NuevoHandlerCargaConvoca(
		&preparadorCargaConvocaBolsa{base: preparador, pdp: pdp, cfg: cfg}, operadorCargaConvocaBolsa{vista: vista, servicio: servicio},
		&auditorCargaConvocaBolsa{preparador: preparador, registrador: registrador, proceso: proceso})
	if err != nil {
		return nil, nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil {
			handler.ServeHTTP(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), claveIntentoCargaConvocaBolsa{}, &intentoCargaConvocaBolsa{})
		// La captura previa permite auditar también un JSON mal formado o una
		// revocación entre esta comprobación y la operación. Solo se conserva
		// cuando la frontera real entregó una sesión válida.
		if seguridad, err := preparador.contextoRevalidado(ctx); err == nil {
			if correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, preparador.generar); err == nil {
				capturarIntentoCargaConvocaBolsa(ctx, seguridad, correlacion)
			}
		}
		handler.ServeHTTP(w, r.WithContext(ctx))
	}), func() {}, nil
}
