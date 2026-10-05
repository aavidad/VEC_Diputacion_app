package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/constitucion"
	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
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
)

// actorActaCargaConvoca es el actor que firma el acta del staging, que solo
// admite referencias opacas en minúsculas. Se deriva de la persona que carga
// (seudónimo estable: misma persona, mismo actor), así que el acta queda
// atribuida aunque la constitución posterior no llegue a consumir la decisión.
func actorActaCargaConvoca(personaRef string) string {
	suma := sha256.Sum256([]byte("vec.bolsa.carga_convoca.actor\x1f" + personaRef))
	return "actor:rrhh:" + hex.EncodeToString(suma[:16])
}

// ImportadorActaCargaConvoca guarda el lote en el staging protegido. Si el
// acta del mismo fichero y categoría ya existe no se vuelve a importar: la
// constitución la recupera tal cual (la carga es idempotente por acta).
type ImportadorActaCargaConvoca interface {
	ActaImportada(ctx context.Context, huellaSHA256, categoriaRef string) (bool, error)
	Importar(context.Context, importacionapp.SolicitudImportacion) (importacionapp.ResultadoImportacion, error)
}

// CustodioFicheroCargaConvoca conserva el fichero original y devuelve su
// referencia opaca (fichero:sha256:...).
type CustodioFicheroCargaConvoca interface {
	Custodiar(context.Context, []byte) (string, error)
}

// ConstituidorCargaConvoca constituye la bolsa del acta consumiendo el
// material de la decisión (constitucion.ServicioAutorizado).
type ConstituidorCargaConvoca interface {
	Constituir(context.Context, constitucion.Solicitud, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboCargaConvoca, error)
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
	custodio        CustodioFicheroCargaConvoca
	importador      ImportadorActaCargaConvoca
	constituidor    ConstituidorCargaConvoca
	reloj           func() time.Time
}

func NuevoServicioCargaConvoca(previsualizador *PrevisualizadorCargaConvoca, contexto ports.ResolutorContextoBorradorLlamamiento,
	autorizador ports.AutorizadorBorradorLlamamientoV3, custodio CustodioFicheroCargaConvoca, importador ImportadorActaCargaConvoca,
	constituidor ConstituidorCargaConvoca, reloj func() time.Time,
) (*ServicioCargaConvoca, error) {
	if previsualizador == nil || contexto == nil || autorizador == nil || custodio == nil || importador == nil || constituidor == nil || reloj == nil {
		return nil, ErrCargaConvocaDependencias
	}
	return &ServicioCargaConvoca{previsualizador: previsualizador, contexto: contexto, autorizador: autorizador,
		custodio: custodio, importador: importador, constituidor: constituidor, reloj: reloj}, nil
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
	vista, err := s.previsualizador.Previsualizar(ctx, solicitud.NombreFichero, solicitud.Contenido)
	if err != nil {
		return ResultadoCargaConvoca{}, err
	}
	if vista.Bloqueo != "" {
		return ResultadoCargaConvoca{}, ErrCargaConvocaBloqueada
	}
	if vista.Rechazadas > 0 && !excluirConErrores {
		return ResultadoCargaConvoca{}, ErrCargaConvocaConErrores
	}
	suma := sha256.Sum256(solicitud.Contenido)
	huella := hex.EncodeToString(suma[:])
	actaRef := importacionapp.ReferenciaActa(huella, solicitud.CategoriaRef)
	actor := solicitud.ResultadoContexto.Contexto
	resuelto, err := s.contexto.ResolverContextoBorradorLlamamiento(ctx, actor)
	if err != nil || resuelto.Validar() != nil {
		return ResultadoCargaConvoca{}, errorDependenciaCarga(err)
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: actaRef, ModuloID: ports.ModuloCargaConvoca, Tipo: ports.TipoRecursoCargaConvoca,
		Ambitos: map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef}}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: solicitud.Vinculo, ReferenciaMotivo: solicitud.MotivoAutorizacion,
		Accion: ports.AccionConfirmarCargaConvoca, Recurso: recurso, Finalidad: ports.FinalidadConfirmarCargaConvoca,
		Correlacion: solicitud.Correlacion})
	if err != nil {
		// La solicitud se arma con datos del servidor: es un fallo técnico.
		return ResultadoCargaConvoca{}, ports.ErrCargaConvocaNoDisponible
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, solicitud.ResultadoContexto)
	if err != nil || exportador == nil || decision.ValidarPara(auth) != nil {
		if errors.Is(err, puertosvec.ErrDenegacionExplicitaAutorizacionLigadaV3) {
			return ResultadoCargaConvoca{}, dominiovec.ErrAutorizacionDenegada
		}
		return ResultadoCargaConvoca{}, errorDependenciaCarga(err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, decision, confirmacion, solicitud.ResultadoContexto, solicitud.MotivoAutorizacion, material, ports.AudienciaConfirmarCargaConvoca) {
		return ResultadoCargaConvoca{}, errorDependenciaCarga(err)
	}
	reutilizada, err := s.importador.ActaImportada(ctx, huella, solicitud.CategoriaRef)
	if err != nil {
		return ResultadoCargaConvoca{}, errorDependenciaCarga(err)
	}
	if !reutilizada {
		custodia, err := s.custodio.Custodiar(ctx, solicitud.Contenido)
		if err != nil {
			return ResultadoCargaConvoca{}, errorDependenciaCarga(err)
		}
		importado, err := s.importador.Importar(ctx, importacionapp.SolicitudImportacion{
			CategoriaRef: solicitud.CategoriaRef, BolsaRef: solicitud.BolsaRef, NombreFichero: solicitud.NombreFichero,
			FicheroCustodiadoRef: custodia, ActorRef: actorActaCargaConvoca(actor.PersonaRef), Contenido: solicitud.Contenido})
		if err != nil {
			return ResultadoCargaConvoca{}, errorDependenciaCarga(err)
		}
		if importado.Acta.ActaRef != actaRef || importado.Acta.HuellaFicheroSHA256 != huella {
			return ResultadoCargaConvoca{}, ports.ErrCargaConvocaNoDisponible
		}
	}
	recibo, err := s.constituidor.Constituir(ctx, constitucion.Solicitud{HuellaFicheroSHA256: huella, CategoriaRef: solicitud.CategoriaRef, ActorRef: actor.PersonaRef}, material)
	if err != nil {
		if errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
			return ResultadoCargaConvoca{}, err
		}
		return ResultadoCargaConvoca{}, errorDependenciaCarga(err)
	}
	if recibo.ActaRef != actaRef {
		return ResultadoCargaConvoca{}, ports.ErrCargaConvocaNoDisponible
	}
	return ResultadoCargaConvoca{Recibo: recibo, HuellaSHA256: huella, FilasCargadas: vista.Aceptadas, FilasExcluidas: vista.Rechazadas, ActaReutilizada: reutilizada}, nil
}

// errorDependenciaCarga conserva la cancelación y convierte cualquier otro
// fallo de una dependencia en indisponibilidad (nunca en éxito ni en permiso).
func errorDependenciaCarga(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, ports.ErrConstitucionBolsaEnConflicto) {
		return ports.ErrConstitucionBolsaEnConflicto
	}
	return ports.ErrCargaConvocaNoDisponible
}
