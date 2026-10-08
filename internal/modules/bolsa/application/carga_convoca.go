package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/constitucion"
	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Motivos por los que la confirmación no puede seguir con este fichero. Son
// códigos estables para el transporte; nunca texto visible.
var (
	ErrCargaConvocaBloqueada    = errors.New("bolsa: el fichero de CONVOCA no se puede cargar")
	ErrCargaConvocaConErrores   = errors.New("bolsa: el fichero de CONVOCA tiene filas con errores")
	ErrCargaConvocaDependencias = errors.New("bolsa: dependencias de la carga CONVOCA requeridas")
	ErrDependenciaCargaConvoca  = errors.New("bolsa: dependencia de carga CONVOCA fallida")
)

// CausaInternaCargaConvoca conserva solo etapa y clase estable. No incorpora
// mensajes de XLS, SQL, identidad ni datos personales a errores o registros.
type CausaInternaCargaConvoca struct {
	Etapa  string
	Codigo string
}

func (c CausaInternaCargaConvoca) Error() string {
	return "bolsa.carga_convoca." + c.Etapa + "." + c.Codigo
}

func errorCargaConCausa(publico error, etapa, codigo string) error {
	return errors.Join(publico, CausaInternaCargaConvoca{Etapa: etapa, Codigo: codigo})
}

// actorActaCargaConvoca es el actor que firma el acta del staging, que solo
// admite referencias opacas en minúsculas. Se deriva de la persona que carga
// (seudónimo estable: misma persona, mismo actor), así que el acta queda
// atribuida aunque la constitución posterior no llegue a consumir la decisión.
func actorActaCargaConvoca(personaRef string) string {
	suma := sha256.Sum256([]byte("vec.bolsa.carga_convoca.actor\x1f" + personaRef))
	return "actor:rrhh:" + hex.EncodeToString(suma[:16])
}

// PreparadorLoteCargaConvoca decodifica y valida una sola vez, sin escribir.
type PreparadorLoteCargaConvoca interface {
	PrepararLote(context.Context, importacionapp.SolicitudImportacion) (importacion.LoteValidado, error)
}

// PreparadorOriginalCargaConvoca cifra el original en memoria; el repositorio
// lo guarda en la misma transacción que el acta y la constitución.
type PreparadorOriginalCargaConvoca interface {
	Preparar(context.Context, string, string, string, []byte) (ports.OriginalProtegidoCargaConvoca, error)
}

// ConstituidorCargaConvoca constituye la bolsa del acta consumiendo el
// material de la decisión (constitucion.ServicioAutorizado).
type ConstituidorCargaConvoca interface {
	Constituir(context.Context, importacion.LoteValidado, constitucion.Solicitud, ports.OriginalProtegidoCargaConvoca, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboCargaConvoca, error)
}

// ResultadoCargaConvoca es lo que la pantalla enseña tras confirmar.
type ResultadoCargaConvoca struct {
	Recibo          ports.ReciboCargaConvoca
	HuellaSHA256    string
	FilasCargadas   int
	FilasExcluidas  int
	ActaReutilizada bool
}

// ServicioCargaConvoca confirma la carga: revalida el fichero, obtiene la
// decisión V3 propia ligada al acta, importa (si hace falta) y constituye la
// bolsa consumiendo la decisión.
type ServicioCargaConvoca struct {
	previsualizador *PrevisualizadorCargaConvoca
	contexto        ports.ResolutorContextoBorradorLlamamiento
	autorizador     ports.AutorizadorBorradorLlamamientoV3
	original        PreparadorOriginalCargaConvoca
	preparador      PreparadorLoteCargaConvoca
	constituidor    ConstituidorCargaConvoca
	reloj           func() time.Time
}

func NuevoServicioCargaConvoca(previsualizador *PrevisualizadorCargaConvoca, contexto ports.ResolutorContextoBorradorLlamamiento,
	autorizador ports.AutorizadorBorradorLlamamientoV3, original PreparadorOriginalCargaConvoca, preparador PreparadorLoteCargaConvoca,
	constituidor ConstituidorCargaConvoca, reloj func() time.Time,
) (*ServicioCargaConvoca, error) {
	if previsualizador == nil || contexto == nil || autorizador == nil || original == nil || preparador == nil || constituidor == nil || reloj == nil {
		return nil, ErrCargaConvocaDependencias
	}
	return &ServicioCargaConvoca{previsualizador: previsualizador, contexto: contexto, autorizador: autorizador,
		original: original, preparador: preparador, constituidor: constituidor, reloj: reloj}, nil
}

// Confirmar carga la bolsa. Con filas rechazadas solo sigue si RRHH lo acepta
// expresamente (excluirConErrores): esas filas no entran en la bolsa.
func (s *ServicioCargaConvoca) Confirmar(ctx context.Context, solicitud ports.SolicitudConfirmarCargaConvoca, excluirConErrores bool) (ResultadoCargaConvoca, error) {
	if ctx == nil || s == nil || s.previsualizador == nil {
		return ResultadoCargaConvoca{}, ErrCargaConvocaDependencias
	}
	if solicitud.Validar() != nil {
		return ResultadoCargaConvoca{}, ports.ErrCargaConvocaInvalida
	}
	if len(solicitud.Contenido) > MaximoBytesCargaConvoca {
		return ResultadoCargaConvoca{}, ErrFicheroCargaConvocaExcesivo
	}
	if !NombreFicheroCargaConvocaValido(solicitud.NombreFichero) {
		return ResultadoCargaConvoca{}, ErrFicheroCargaConvocaInvalido
	}
	suma := sha256.Sum256(solicitud.Contenido)
	huella := hex.EncodeToString(suma[:])
	actaRef := importacionapp.ReferenciaActa(huella, solicitud.CategoriaRef)
	actor := solicitud.ResultadoContexto.Contexto
	resuelto, err := s.contexto.ResolverContextoBorradorLlamamiento(ctx, actor)
	if err != nil || resuelto.Validar() != nil {
		return ResultadoCargaConvoca{}, errorDependenciaCarga("contexto", err)
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: actaRef, ModuloID: ports.ModuloCargaConvoca, Tipo: ports.TipoRecursoCargaConvoca,
		Ambitos: map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef}}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: solicitud.Vinculo, ReferenciaMotivo: solicitud.MotivoAutorizacion,
		Accion: ports.AccionConfirmarCargaConvoca, Recurso: recurso, Finalidad: ports.FinalidadConfirmarCargaConvoca,
		Correlacion: solicitud.Correlacion})
	if err != nil {
		// La solicitud se arma con datos del servidor: es un fallo técnico.
		return ResultadoCargaConvoca{}, errorCargaConCausa(ports.ErrCargaConvocaNoDisponible, "solicitud_autorizacion", "incoherente")
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, solicitud.ResultadoContexto)
	if err != nil || exportador == nil || decision.ValidarPara(auth) != nil {
		if errors.Is(err, puertosvec.ErrDenegacionExplicitaAutorizacionLigadaV3) {
			return ResultadoCargaConvoca{}, dominiovec.ErrAutorizacionDenegada
		}
		return ResultadoCargaConvoca{}, errorDependenciaCarga("autorizacion", err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, decision, confirmacion, solicitud.ResultadoContexto, solicitud.MotivoAutorizacion, material, ports.AudienciaConfirmarCargaConvoca) {
		return ResultadoCargaConvoca{}, errorDependenciaCarga("material_autorizacion", err)
	}
	originalRef := "original:convoca:" + strings.TrimPrefix(actaRef, "acta:importacion-convoca:")
	lote, err := s.preparador.PrepararLote(ctx, importacionapp.SolicitudImportacion{
		CategoriaRef: solicitud.CategoriaRef, BolsaRef: solicitud.BolsaRef, NombreFichero: solicitud.NombreFichero,
		FicheroCustodiadoRef: originalRef, ActorRef: actorActaCargaConvoca(actor.PersonaRef), Contenido: solicitud.Contenido,
	})
	if err != nil {
		if ctx.Err() != nil {
			return ResultadoCargaConvoca{}, ctx.Err()
		}
		return ResultadoCargaConvoca{}, errorCargaConCausa(ErrFicheroCargaConvocaInvalido, "preparar_lote", codigoPreparacionCarga(err))
	}
	if lote.Acta.ActaRef != actaRef || lote.Acta.HuellaFicheroSHA256 != huella || lote.Acta.FicheroCustodiadoRef != originalRef {
		return ResultadoCargaConvoca{}, ports.ErrCargaConvocaNoDisponible
	}
	vista, err := VistaPreviaDesdeLote(lote)
	if err != nil {
		return ResultadoCargaConvoca{}, errorCargaConCausa(ErrFicheroCargaConvocaInvalido, "vista_lote", "incoherente")
	}
	if vista.Bloqueo != "" {
		return ResultadoCargaConvoca{}, ErrCargaConvocaBloqueada
	}
	if vista.Rechazadas > 0 && !excluirConErrores {
		return ResultadoCargaConvoca{}, ErrCargaConvocaConErrores
	}
	formato := strings.TrimPrefix(strings.ToLower(filepath.Ext(solicitud.NombreFichero)), ".")
	sobre, err := s.original.Preparar(ctx, actaRef, huella, formato, solicitud.Contenido)
	if err != nil {
		return ResultadoCargaConvoca{}, errorDependenciaCarga("cifrar_original", err)
	}
	defer clear(sobre.ContenidoCifrado)
	if sobre.Referencia != originalRef || sobre.Formato != formato || sobre.BytesOriginales != len(solicitud.Contenido) {
		return ResultadoCargaConvoca{}, ports.ErrCargaConvocaNoDisponible
	}
	recibo, err := s.constituidor.Constituir(ctx, lote, constitucion.Solicitud{HuellaFicheroSHA256: huella, CategoriaRef: solicitud.CategoriaRef, ActorRef: actor.PersonaRef}, sobre, material)
	if err != nil {
		if errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
			return ResultadoCargaConvoca{}, err
		}
		return ResultadoCargaConvoca{}, errorDependenciaCarga("confirmar_transaccion", err)
	}
	if recibo.ActaRef != actaRef {
		return ResultadoCargaConvoca{}, ports.ErrCargaConvocaNoDisponible
	}
	return ResultadoCargaConvoca{Recibo: recibo, HuellaSHA256: huella, FilasCargadas: vista.Aceptadas, FilasExcluidas: vista.Rechazadas, ActaReutilizada: recibo.ActaReutilizada}, nil
}

// errorDependenciaCarga conserva la cancelación y convierte cualquier otro
// fallo de una dependencia en indisponibilidad (nunca en éxito ni en permiso).
func errorDependenciaCarga(etapa string, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, ports.ErrConstitucionBolsaEnConflicto) {
		return errorCargaConCausa(ports.ErrConstitucionBolsaEnConflicto, etapa, "conflicto")
	}
	codigo := "dependencia_no_disponible"
	if err == nil {
		codigo = "respuesta_incoherente"
	} else if errors.Is(err, ports.ErrConstitucionBolsaNoDisponible) {
		codigo = "repositorio_no_disponible"
	}
	return errors.Join(ports.ErrCargaConvocaNoDisponible, ErrDependenciaCargaConvoca, CausaInternaCargaConvoca{Etapa: etapa, Codigo: codigo})
}

func codigoPreparacionCarga(err error) string {
	if errors.Is(err, importacionapp.ErrSolicitudInvalida) {
		return "solicitud_invalida"
	}
	if errors.Is(err, importacionapp.ErrResultadoInseguro) {
		return "lote_inseguro"
	}
	return "fichero_invalido"
}
