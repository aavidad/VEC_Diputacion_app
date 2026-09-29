package bootstrap

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	docautorizacion "vec-diputacion-granada/internal/vec/documentos/adapters/autorizacion"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Autoridad de lectura del almacén de Documentos en la composición de
// desarrollo: seudónimos con una clave propia derivada del KMS de desarrollo
// y concesiones V3 registradas del PDP de Documentos para el vínculo de la
// MISMA petición. No es un HSM ni una autoridad corporativa: solo se compone
// con el perfil de desarrollo y su doble llave.

// dominioSeudonimosAlmacenDesarrollo separa esta clave de las demás
// derivadas de la clave maestra del KMS de desarrollo (envoltura, etc.).
const dominioSeudonimosAlmacenDesarrollo = "vec.almacen.seudonimizacion.desarrollo.v1"

const (
	prefijoHMACSujetoAlmacenDesarrollo    = "hmac-sha256:almacen_sujeto_desarrollo_v1:"
	prefijoHMACSolicitudAlmacenDesarrollo = "hmac-sha256:almacen_solicitud_desarrollo_v1:"
)

var errLecturaAlmacenDocumentosDenegada = errors.New("bootstrap: lectura del almacén de Documentos denegada")

// seudonimizadorAlmacenDesarrollo guarda solo la clave derivada; la maestra
// no sale de la composición raíz.
type seudonimizadorAlmacenDesarrollo struct {
	clave [sha256.Size]byte
}

func nuevoSeudonimizadorAlmacenDesarrollo(claveMaestra [sha256.Size]byte) *seudonimizadorAlmacenDesarrollo {
	return &seudonimizadorAlmacenDesarrollo{clave: derivarClaveDesarrollo(claveMaestra, dominioSeudonimosAlmacenDesarrollo)}
}

// borrar pone a cero la clave derivada al cerrar la composición.
func (s *seudonimizadorAlmacenDesarrollo) borrar() {
	if s != nil {
		s.clave = [sha256.Size]byte{}
	}
}

func (s *seudonimizadorAlmacenDesarrollo) valido() bool {
	return s != nil && s.clave != [sha256.Size]byte{}
}

func (s *seudonimizadorAlmacenDesarrollo) hmac(etiqueta string, partes ...string) string {
	mac := hmac.New(sha256.New, s.clave[:])
	_, _ = mac.Write([]byte("vec.almacen.seudonimo.desarrollo.v1\x00" + etiqueta))
	for _, parte := range partes {
		_, _ = mac.Write([]byte{0})
		_, _ = mac.Write([]byte(parte))
	}
	return hex.EncodeToString(mac.Sum(nil))
}

// emisorConcesionAlmacenDocumentosDesarrollo implementa
// docautorizacion.EmisorConcesionAlmacenV3.
type emisorConcesionAlmacenDocumentosDesarrollo struct {
	autoridad      *autoridadDocumentosDesarrollo
	autorizador    autorizadorConcesionAlmacenV3
	motivo         core.ReferenciaEntradaCatalogo
	seudonimizador *seudonimizadorAlmacenDesarrollo
}

type autorizadorConcesionAlmacenV3 interface {
	ExigirSolicitudLigadaV3(context.Context, core.SolicitudAutorizacionLigadaV3, core.ResultadoContextoActorRegistradoV2) (
		core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error)
}

var _ docautorizacion.EmisorConcesionAlmacenV3 = (*emisorConcesionAlmacenDocumentosDesarrollo)(nil)

func (e *emisorConcesionAlmacenDocumentosDesarrollo) SeudonimosLecturaOriginal(
	ctx context.Context, a docports.AutorizacionV3,
) (docautorizacion.DatosSeudonimosLectura, error) {
	if e == nil || !e.seudonimizador.valido() || ctx == nil || ctx.Err() != nil ||
		a.PrincipalID == "" || a.CorrelacionRef == "" || a.RecursoRef == "" {
		return docautorizacion.DatosSeudonimosLectura{}, errLecturaAlmacenDocumentosDenegada
	}
	return docautorizacion.DatosSeudonimosLectura{
		SujetoHMAC:    prefijoHMACSujetoAlmacenDesarrollo + e.seudonimizador.hmac("sujeto", a.PrincipalID),
		SolicitudHMAC: prefijoHMACSolicitudAlmacenDesarrollo + e.seudonimizador.hmac("solicitud", a.CorrelacionRef, a.RecursoRef),
	}, nil
}

// EmitirConcesionAlmacenV3 toma vínculo y resultado del contexto de la misma
// petición (el que resolvió la frontera de Documentos) y exige que sean de la
// persona y el perfil pedidos. La decisión y su registro los da el PDP; no se
// exporta material consumible.
func (e *emisorConcesionAlmacenDocumentosDesarrollo) EmitirConcesionAlmacenV3(
	ctx context.Context, s docautorizacion.SolicitudConcesionAlmacenV3,
) (docautorizacion.ConcesionAlmacenV3, error) {
	var vacia docautorizacion.ConcesionAlmacenV3
	if e == nil || e.autoridad == nil || dependenciaEsNulaContratacionTemporalDesarrollo(e.autorizador) || ctx == nil || ctx.Err() != nil {
		return vacia, errLecturaAlmacenDocumentosDenegada
	}
	c, ok := ctx.Value(claveContextoDocumentos{}).(contextoDocumentos)
	if !ok || c.autoridad != e.autoridad || c.seguridad.Resultado.Validar() != nil ||
		!c.seguridad.Vinculo.VigenteEn(e.autoridad.reloj.Ahora(), c.seguridad.Resultado) {
		return vacia, errLecturaAlmacenDocumentosDenegada
	}
	datos, err := c.seguridad.Vinculo.Datos()
	if err != nil || datos.PrincipalID != s.PrincipalID || datos.PerfilActivoRef != s.PerfilActivoRef {
		return vacia, errLecturaAlmacenDocumentosDenegada
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, errLecturaAlmacenDocumentosDenegada
	}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: c.seguridad.Vinculo, ReferenciaMotivo: e.motivo,
		Accion: s.Accion, Recurso: s.Recurso, Finalidad: s.Finalidad, Correlacion: correlacion,
	})
	if err != nil {
		return vacia, errLecturaAlmacenDocumentosDenegada
	}
	decision, confirmacion, err := e.autorizador.ExigirSolicitudLigadaV3(ctx, solicitud, c.seguridad.Resultado)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return vacia, err
		}
		// Una denegación es una respuesta del PDP; cualquier otro fallo es
		// indisponibilidad del gobierno V3 y queda declarada como incidencia.
		if !errors.Is(err, core.ErrAutorizacionDenegada) {
			e.autoridad.incidencia(core.IncidenciaGobiernoV3NoDisponible, core.ComponenteIncidenciaGobiernoV3, core.EtapaIncidenciaConsulta)
		}
		return vacia, errors.Join(errLecturaAlmacenDocumentosDenegada, err)
	}
	return docautorizacion.ConcesionAlmacenV3{Solicitud: solicitud, Decision: decision, Confirmacion: confirmacion}, nil
}

// nuevaFabricaLecturaDocumentosDesarrollo devuelve nil sin seudonimizador:
// entonces la descarga no se publica.
func nuevaFabricaLecturaDocumentosDesarrollo(
	autoridad *autoridadDocumentosDesarrollo, autorizador autorizadorConcesionAlmacenV3,
	motivo core.ReferenciaEntradaCatalogo, seudonimizador *seudonimizadorAlmacenDesarrollo,
) (docports.FabricaContextoLectura, error) {
	if !seudonimizador.valido() {
		return nil, nil
	}
	if autoridad == nil || dependenciaEsNulaContratacionTemporalDesarrollo(autorizador) || motivo.Validar() != nil {
		return nil, errLecturaAlmacenDocumentosDenegada
	}
	return docautorizacion.NuevaFabricaContextoLecturaOriginalV3(&emisorConcesionAlmacenDocumentosDesarrollo{
		autoridad: autoridad, autorizador: autorizador, motivo: motivo, seudonimizador: seudonimizador,
	}, autoridad.reloj)
}
