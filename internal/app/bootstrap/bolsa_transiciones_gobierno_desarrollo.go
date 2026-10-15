package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// nuevaRutaGobiernoTransicionesBolsaDesarrollo recibe un emisor nominal de
// la audiencia propia. La composición raíz lo crea solo si hay publicación
// de clave y perfil de catálogo configurado. Ausencia de dependencias cierra
// esta ruta con 503; el resto de Bolsa puede arrancar y seguir consultando.
func nuevaRutaGobiernoTransicionesBolsaDesarrollo(pool *pgxpool.Pool,
	base *preparadorBorradorLlamamientoDesarrollo,
	emisor *emisorMaterialRenovableCTDesarrollo,
	situacion *aplicacionbolsa.ServicioSituacionParticipacion,
	catalogoVersion int, catalogoSHA256 string,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	registrador puertosvec.RegistradorIntentosAuditoria, proceso string,
) vechttp.RutaExacta {
	ruta := vechttp.RutaExacta{Ruta: bolsahttp.RutaGobiernoTransicionesSituacion,
		Manejador: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"codigo":"servicio_no_disponible"}}`))
		})}
	if pool == nil || base == nil || base.soporte == nil || emisor == nil || situacion == nil ||
		catalogoVersion < 1 || len(catalogoSHA256) != 64 || motivo.Validar() != nil ||
		registrador == nil || proceso != "vec-rrhh" {
		return ruta
	}
	repo, err := postgresbolsa.NuevoRepositorioGobiernoTransicionesSituacionPostgreSQL(pool)
	if err != nil {
		return ruta
	}
	servicio, err := aplicacionbolsa.NuevoServicioGobiernoTransicionesSituacion(repo, situacion, catalogoVersion, catalogoSHA256)
	if err != nil {
		return ruta
	}
	preparador := &preparadorGobiernoTransicionesBolsaDesarrollo{base: base, emisor: emisor, motivo: motivo,
		catalogoVersion: catalogoVersion, catalogoSHA256: catalogoSHA256, registrador: registrador, proceso: proceso}
	handler, err := bolsahttp.NuevoHandlerGobiernoTransicionesSituacion(preparador, servicio)
	if err == nil {
		ruta.Manejador = handler
	}
	return ruta
}

type preparadorGobiernoTransicionesBolsaDesarrollo struct {
	base            *preparadorBorradorLlamamientoDesarrollo
	emisor          *emisorMaterialRenovableCTDesarrollo
	motivo          dominiovec.ReferenciaEntradaCatalogo
	catalogoVersion int
	catalogoSHA256  string
	registrador     puertosvec.RegistradorIntentosAuditoria
	proceso         string
}

func (p *preparadorGobiernoTransicionesBolsaDesarrollo) solicitud(ctx context.Context,
	entrada *bolsahttp.EntradaGobiernoTransicionesSituacion) (
	contextoSeguridadComunDesarrollo, dominiovec.SolicitudAutorizacionLigadaV3, error,
) {
	if p == nil || p.base == nil || p.base.soporte == nil || p.emisor == nil || p.motivo.Validar() != nil {
		return contextoSeguridadComunDesarrollo{}, dominiovec.SolicitudAutorizacionLigadaV3{}, puertosbolsa.ErrGobiernoTransicionesNoDisponible
	}
	actor, err := p.base.contextoRevalidado(ctx)
	if err != nil {
		return contextoSeguridadComunDesarrollo{}, dominiovec.SolicitudAutorizacionLigadaV3{}, dominiovec.ErrAutorizacionDenegada
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.base.generar)
	if err != nil {
		return contextoSeguridadComunDesarrollo{}, dominiovec.SolicitudAutorizacionLigadaV3{}, err
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: puertosbolsa.RecursoGobiernoTransicionesSituacion,
		ModuloID: "bolsa", Tipo: puertosbolsa.TipoRecursoGobiernoTransicionesSituacion,
		Ambitos: map[string]string{"unidad_ref": p.base.soporte.unidadRef, "ambito_ref": p.base.soporte.ambitoRef}}
	if entrada != nil {
		politica, err := dominiobolsa.PoliticaTransicionesDesdePares(entrada.Transiciones)
		if err != nil || !slices.Equal(politica.Pares(), entrada.Transiciones) || p.catalogoVersion < 1 || len(p.catalogoSHA256) != 64 {
			return contextoSeguridadComunDesarrollo{}, dominiovec.SolicitudAutorizacionLigadaV3{}, puertosbolsa.ErrGobiernoTransicionesInvalido
		}
		actorRef := actor.Resultado.Contexto.Principal.ID
		perfilRef := actor.Resultado.Contexto.PerfilActivoRef
		recurso.Atributos = atributosIntencionTransicionesGobierno(*entrada, actorRef, perfilRef,
			p.catalogoVersion, p.catalogoSHA256)
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: actor.Vinculo, ReferenciaMotivo: p.motivo,
		Accion: puertosbolsa.AccionPublicarTransicionesSituacion, Recurso: recurso,
		Finalidad: puertosbolsa.FinalidadGobiernoTransicionesSituacion, Correlacion: correlacion})
	if err != nil {
		return contextoSeguridadComunDesarrollo{}, dominiovec.SolicitudAutorizacionLigadaV3{}, dominiovec.ErrAutorizacionDenegada
	}
	return actor, solicitud, nil
}

func (p *preparadorGobiernoTransicionesBolsaDesarrollo) PrepararConsultaGobiernoTransiciones(ctx context.Context) (bool, error) {
	if p == nil || p.base == nil || p.base.soporte == nil || p.base.fuenteRespuestas == nil {
		return false, puertosbolsa.ErrGobiernoTransicionesNoDisponible
	}
	actor, err := p.base.contextoRevalidado(ctx)
	if err != nil {
		return false, dominiovec.ErrAutorizacionDenegada
	}
	datos, err := actor.Vinculo.Datos()
	if err != nil || datos.Superficie != "interna_corporativa" || datos.CuentaPrivilegiada {
		return false, dominiovec.ErrAutorizacionDenegada
	}
	instantanea, err := p.base.fuenteRespuestas.ObtenerInstantaneaAutorizacion(ctx, datos.PrincipalID, datos.PerfilActivoRef)
	if err != nil {
		return false, puertosbolsa.ErrGobiernoTransicionesNoDisponible
	}
	contrato := contratoLecturaBolsa{accion: puertosbolsa.AccionConsultarTransicionesSituacion,
		finalidad: puertosbolsa.FinalidadConsultaTransicionesSituacion,
		tipo:      puertosbolsa.TipoRecursoGobiernoTransicionesSituacion,
		campos:    []string{puertosbolsa.CampoConsultaTransicionesSituacion}}
	if instantanea.Validar() != nil || !concesionLecturaBolsaExacta(instantanea, contrato) ||
		instantanea.AsignacionPerfil.EmitidaPor == "identidad:desarrollo:no-autoritativa" ||
		instantanea.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" ||
		instantanea.ControlVigenciaVersionRol.ActualizadoPor == "seguridad:desarrollo:no-autoritativa" {
		return false, dominiovec.ErrAutorizacionDenegada
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: puertosbolsa.RecursoGobiernoTransicionesSituacion,
		ModuloID: "bolsa", Tipo: puertosbolsa.TipoRecursoGobiernoTransicionesSituacion,
		Ambitos: map[string]string{"unidad_ref": p.base.soporte.unidadRef, "ambito_ref": p.base.soporte.ambitoRef}}
	evaluacion, err := dominiovec.EvaluarLecturaActualAutorizacionV3(actor.Vinculo, actor.Resultado,
		instantanea, contrato.accion, recurso, contrato.finalidad, contrato.campos, time.Now().UTC().Truncate(time.Microsecond))
	if err != nil || !evaluacion.Permitido || !slices.Equal(evaluacion.CamposPermitidos, contrato.campos) ||
		(len(evaluacion.Obligaciones) != 0 && !slices.Equal(evaluacion.Obligaciones, []string{"auditar"})) {
		return false, dominiovec.ErrAutorizacionDenegada
	}
	puede, err := dominiovec.PuedeEjecutarAccionActualV3(actor.Vinculo, actor.Resultado, instantanea,
		puertosbolsa.AccionPublicarTransicionesSituacion, recurso,
		puertosbolsa.FinalidadGobiernoTransicionesSituacion, time.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		return false, puertosbolsa.ErrGobiernoTransicionesNoDisponible
	}
	return puede, nil
}

func (p *preparadorGobiernoTransicionesBolsaDesarrollo) AuditarConsultaGobiernoTransiciones(ctx context.Context, exito bool) error {
	if p == nil || p.registrador == nil || p.proceso != "vec-rrhh" {
		return puertosbolsa.ErrGobiernoTransicionesNoDisponible
	}
	protegido, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelar()
	if _, err := p.PrepararConsultaGobiernoTransiciones(protegido); err != nil {
		return err
	}
	actor, err := p.base.contextoRevalidado(protegido)
	if err != nil {
		return err
	}
	ref, err := puertosvec.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return err
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(protegido, p.base.generar)
	if err != nil {
		return err
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		return err
	}
	resultado := dominiovec.ResultadoIntentoAuditoriaExito
	if !exito {
		resultado = dominiovec.ResultadoIntentoAuditoriaError
	}
	orden, err := puertosvec.NuevaOrdenIntentoAuditoria(ref, actor.Resultado, actor.Vinculo, dominiovec.DatosIntentoAuditoria{
		Accion: "lectura_bolsas_rrhh_consultar", ModuloID: "bolsa",
		RecursoRef:   puertosbolsa.RecursoGobiernoTransicionesSituacion,
		FinalidadRef: "gestion_bolsa_rrhh",
		Resultado:    resultado, Motivo: p.motivo, Proceso: p.proceso,
		Canal: "interna_corporativa", CorrelacionRef: correlacionRef,
	})
	if err != nil {
		return err
	}
	acuse, err := p.registrador.AppendIntentoAuditoria(protegido, orden)
	if err != nil {
		return err
	}
	return acuse.ValidarPara(orden)
}

func (p *preparadorGobiernoTransicionesBolsaDesarrollo) PrepararPublicacionGobiernoTransiciones(ctx context.Context,
	e bolsahttp.EntradaGobiernoTransicionesSituacion) (puertosbolsa.ComandoGobiernoTransicionesSituacion, error) {
	actor, solicitud, err := p.solicitud(ctx, &e)
	if err != nil {
		return puertosbolsa.ComandoGobiernoTransicionesSituacion{}, err
	}
	_, _, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, actor.Resultado)
	if err != nil || exportador == nil {
		if errors.Is(err, dominiovec.ErrAutorizacionDenegada) || errors.Is(err, dominiovec.ErrPermissionDenied) {
			return puertosbolsa.ComandoGobiernoTransicionesSituacion{}, err
		}
		return puertosbolsa.ComandoGobiernoTransicionesSituacion{}, puertosbolsa.ErrGobiernoTransicionesNoDisponible
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || material.ValidarEstructura() != nil {
		return puertosbolsa.ComandoGobiernoTransicionesSituacion{}, puertosbolsa.ErrGobiernoTransicionesNoDisponible
	}
	actorRef := actor.Resultado.Contexto.Principal.ID
	huella := sha256.Sum256([]byte(actorRef + "\x1f" + e.ClaveIdempotencia))
	return puertosbolsa.ComandoGobiernoTransicionesSituacion{
		VersionEsperada: e.VersionEsperada, Transiciones: e.Transiciones, Motivo: e.Motivo,
		ClaveIdempotencia: e.ClaveIdempotencia, ActorRef: actorRef,
		ReciboRef: "recibo:politica-transiciones:" + hex.EncodeToString(huella[:]), Material: material,
	}, nil
}

func huellaTransicionesGobierno(texto string) string {
	suma := sha256.Sum256([]byte(texto))
	return hex.EncodeToString(suma[:])
}

func atributosIntencionTransicionesGobierno(e bolsahttp.EntradaGobiernoTransicionesSituacion,
	actorRef, perfilRef string, catalogoVersion int, catalogoSHA256 string) map[string]string {
	return map[string]string{
		"actor_ref":           actorRef,
		"catalogo_sha256":     catalogoSHA256,
		"catalogo_version":    strconv.Itoa(catalogoVersion),
		"clave_sha256":        huellaTransicionesGobierno(e.ClaveIdempotencia),
		"motivo_sha256":       huellaTransicionesGobierno(e.Motivo),
		"perfil_activo_ref":   perfilRef,
		"transiciones_sha256": huellaTransicionesGobierno(strings.Join(e.Transiciones, ",")),
		"version_esperada":    strconv.FormatInt(e.VersionEsperada, 10),
	}
}

var _ bolsahttp.PreparadorGobiernoTransicionesSituacion = (*preparadorGobiernoTransicionesBolsaDesarrollo)(nil)
