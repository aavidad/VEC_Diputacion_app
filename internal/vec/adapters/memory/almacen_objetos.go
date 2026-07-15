package memory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

// AlmacenObjetosMemoria ejercita el contrato del puerto sin fingir aptitud
// productiva. No cifra, no ofrece URL temporales y pierde todo al reiniciar.
type AlmacenObjetosMemoria struct {
	mu                  sync.RWMutex
	conectorID          string
	tamanoMaximo        int64
	reloj               ports.Reloj
	secuenciaObjetos    uint64
	secuenciaEvidencias uint64
	objetos             map[string]objetoAlmacenMemoria
	idempotencias       map[string]idempotenciaAlmacenMemoria
}

type objetoAlmacenMemoria struct {
	metadatos ports.ObjetoAlmacenado
	contenido []byte
}

type idempotenciaAlmacenMemoria struct {
	huellaSolicitud string
	objeto          ports.ReferenciaObjetoAlmacen
}

func NuevoAlmacenObjetosMemoria(conectorID string, tamanoMaximo int64, reloj ports.Reloj) (*AlmacenObjetosMemoria, error) {
	if reloj == nil || reloj.Ahora().IsZero() || ports.VerificarCapacidadesAlmacen(
		capacidadesAlmacenObjetosMemoria(conectorID, tamanoMaximo),
		ports.RequisitosAlmacenObjetos{},
	) != nil {
		return nil, ports.ErrSolicitudAlmacenInvalida
	}
	return &AlmacenObjetosMemoria{
		conectorID:    conectorID,
		tamanoMaximo:  tamanoMaximo,
		reloj:         reloj,
		objetos:       make(map[string]objetoAlmacenMemoria),
		idempotencias: make(map[string]idempotenciaAlmacenMemoria),
	}, nil
}

func (a *AlmacenObjetosMemoria) Capacidades(ctx context.Context) (ports.CapacidadesAlmacenObjetos, error) {
	if err := ctx.Err(); err != nil {
		return ports.CapacidadesAlmacenObjetos{}, err
	}
	return capacidadesAlmacenObjetosMemoria(a.conectorID, a.tamanoMaximo), nil
}

func capacidadesAlmacenObjetosMemoria(conectorID string, tamanoMaximo int64) ports.CapacidadesAlmacenObjetos {
	return ports.CapacidadesAlmacenObjetos{
		ConectorID:             conectorID,
		EscrituraEnFlujo:       true,
		LecturaEnFlujo:         true,
		ReferenciasOpacas:      true,
		IntegridadSHA256:       true,
		Versionado:             true,
		Retencion:              true,
		BloqueoLegal:           true,
		PromocionAtomica:       true,
		CargaDirectaTemporal:   false,
		CifradoEnTransito:      false,
		CifradoEnReposo:        false,
		CifradoPorObjeto:       false,
		TamanoMaximoObjeto:     tamanoMaximo,
		PreservaObjetoOriginal: true,
	}
}

func (a *AlmacenObjetosMemoria) Escribir(ctx context.Context, solicitud ports.SolicitudEscribirObjeto) (ports.ResultadoOperacionObjeto, error) {
	if err := ctx.Err(); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if err := solicitud.Validar(); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if solicitud.Tamano > a.tamanoMaximo {
		return ports.ResultadoOperacionObjeto{}, ports.ErrLimiteObjetoAlmacenExcedido
	}
	contenido, err := leerContenidoExacto(solicitud.Contenido, solicitud.Tamano)
	if err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	huella := calcularSHA256(contenido)
	if huella != solicitud.HuellaSHA256 {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	huellaSolicitud := huellaEscritura(solicitud)
	claveIdempotencia := solicitud.ClaveIdempotencia

	a.mu.Lock()
	defer a.mu.Unlock()
	ahora := a.reloj.Ahora().UTC()
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenEscribir, ahora); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if anterior, existe := a.idempotencias[claveIdempotencia]; existe {
		if anterior.huellaSolicitud != huellaSolicitud {
			return ports.ResultadoOperacionObjeto{}, ports.ErrIdempotenciaAlmacenReutilizada
		}
		objeto, existe := a.objetos[claveObjetoAlmacen(anterior.objeto)]
		if !existe {
			return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
		}
		if objeto.metadatos.Eliminado {
			return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenEliminado
		}
		if objeto.metadatos.Inmovilizado {
			return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenInmovilizado
		}
		if err := validarObjetoAlmacenMemoria(objeto); err != nil {
			return ports.ResultadoOperacionObjeto{}, err
		}
		resultado := ports.ResultadoOperacionObjeto{
			Objeto: objeto.metadatos,
			Evidencia: a.nuevaEvidenciaBloqueada(
				solicitud.Contexto,
				objeto.metadatos.Objeto,
				"",
				true,
				ahora,
			),
		}
		if resultado.ValidarEscritura(solicitud, capacidadesAlmacenObjetosMemoria(a.conectorID, a.tamanoMaximo)) != nil {
			return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
		}
		return resultado, nil
	}

	a.secuenciaObjetos++
	referencia := ports.ReferenciaObjetoAlmacen{
		Referencia: fmt.Sprintf("objeto-%016x", a.secuenciaObjetos),
		Version:    "1",
	}
	if ahora.IsZero() {
		return ports.ResultadoOperacionObjeto{}, ports.ErrSolicitudAlmacenInvalida
	}
	evidencia := a.nuevaEvidenciaBloqueada(solicitud.Contexto, referencia, "", false, ahora)
	metadatos := ports.ObjetoAlmacenado{
		Objeto:               referencia,
		ConectorID:           a.conectorID,
		Zona:                 solicitud.Zona,
		MIME:                 solicitud.MIME,
		Tamano:               solicitud.Tamano,
		HuellaSHA256:         huella,
		EvidenciaCreacionRef: evidencia.Referencia,
		AlmacenadoEn:         ahora,
	}
	resultado := ports.ResultadoOperacionObjeto{Objeto: metadatos, Evidencia: evidencia}
	if resultado.ValidarEscritura(solicitud, capacidadesAlmacenObjetosMemoria(a.conectorID, a.tamanoMaximo)) != nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	a.objetos[claveObjetoAlmacen(referencia)] = objetoAlmacenMemoria{
		metadatos: metadatos,
		contenido: append([]byte(nil), contenido...),
	}
	a.idempotencias[claveIdempotencia] = idempotenciaAlmacenMemoria{
		huellaSolicitud: huellaSolicitud,
		objeto:          referencia,
	}
	return resultado, nil
}

func (a *AlmacenObjetosMemoria) Abrir(ctx context.Context, solicitud ports.SolicitudAbrirObjeto) (ports.LecturaObjetoAlmacen, error) {
	if err := ctx.Err(); err != nil {
		return ports.LecturaObjetoAlmacen{}, err
	}
	if err := solicitud.Validar(); err != nil {
		return ports.LecturaObjetoAlmacen{}, err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	ahora := a.reloj.Ahora().UTC()
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenLeer, ahora); err != nil {
		return ports.LecturaObjetoAlmacen{}, err
	}
	objeto, existe := a.objetos[claveObjetoAlmacen(solicitud.Objeto)]
	if !existe {
		return ports.LecturaObjetoAlmacen{}, ports.ErrObjetoAlmacenNoEncontrado
	}
	if objeto.metadatos.Eliminado {
		return ports.LecturaObjetoAlmacen{}, ports.ErrObjetoAlmacenEliminado
	}
	if objeto.metadatos.Inmovilizado {
		return ports.LecturaObjetoAlmacen{}, ports.ErrObjetoAlmacenInmovilizado
	}
	if err := validarObjetoAlmacenMemoria(objeto); err != nil {
		return ports.LecturaObjetoAlmacen{}, err
	}
	if objeto.metadatos.Zona != solicitud.Zona {
		return ports.LecturaObjetoAlmacen{}, ports.ErrTransicionZonaAlmacenNoPermitida
	}
	if objeto.metadatos.Tamano > solicitud.Limite {
		return ports.LecturaObjetoAlmacen{}, ports.ErrLimiteObjetoAlmacenExcedido
	}
	contenido := append([]byte(nil), objeto.contenido...)
	lectura := ports.LecturaObjetoAlmacen{
		Objeto: objeto.metadatos,
		Evidencia: a.nuevaEvidenciaBloqueada(
			solicitud.Contexto,
			objeto.metadatos.Objeto,
			"",
			false,
			ahora,
		),
		Contenido: io.NopCloser(bytes.NewReader(contenido)),
	}
	if lectura.ValidarContra(solicitud) != nil {
		_ = lectura.Contenido.Close()
		return ports.LecturaObjetoAlmacen{}, ports.ErrIntegridadObjetoAlmacen
	}
	return lectura, nil
}

func (a *AlmacenObjetosMemoria) Promover(ctx context.Context, solicitud ports.SolicitudPromoverObjeto) (ports.ResultadoOperacionObjeto, error) {
	if err := ctx.Err(); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if err := solicitud.Validar(); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	ahora := a.reloj.Ahora().UTC()
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenPromover, ahora); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	origen, existe := a.objetos[claveObjetoAlmacen(solicitud.Origen)]
	if !existe {
		return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenNoEncontrado
	}
	if origen.metadatos.Eliminado {
		return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenEliminado
	}
	if origen.metadatos.Inmovilizado {
		return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenInmovilizado
	}
	if err := validarObjetoAlmacenMemoria(origen); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if origen.metadatos.Zona != ports.ZonaAlmacenCuarentena {
		return ports.ResultadoOperacionObjeto{}, ports.ErrTransicionZonaAlmacenNoPermitida
	}
	claveIdempotencia := solicitud.ClaveIdempotencia
	huellaSolicitud := huellaPromocion(solicitud, origen.metadatos)
	if anterior, existe := a.idempotencias[claveIdempotencia]; existe {
		if anterior.huellaSolicitud != huellaSolicitud {
			return ports.ResultadoOperacionObjeto{}, ports.ErrIdempotenciaAlmacenReutilizada
		}
		promovido, existe := a.objetos[claveObjetoAlmacen(anterior.objeto)]
		if !existe {
			return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
		}
		if promovido.metadatos.Eliminado {
			return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenEliminado
		}
		if promovido.metadatos.Inmovilizado {
			return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenInmovilizado
		}
		if err := validarObjetoAlmacenMemoria(promovido); err != nil {
			return ports.ResultadoOperacionObjeto{}, err
		}
		resultado := ports.ResultadoOperacionObjeto{
			Objeto: promovido.metadatos,
			Evidencia: a.nuevaEvidenciaBloqueada(
				solicitud.Contexto,
				promovido.metadatos.Objeto,
				solicitud.EvidenciaAnalisisRef,
				true,
				ahora,
			),
		}
		if resultado.ValidarPromocion(
			solicitud,
			origen.metadatos,
			capacidadesAlmacenObjetosMemoria(a.conectorID, a.tamanoMaximo),
		) != nil {
			return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
		}
		return resultado, nil
	}

	a.secuenciaObjetos++
	referencia := ports.ReferenciaObjetoAlmacen{Referencia: fmt.Sprintf("objeto-%016x", a.secuenciaObjetos), Version: "1"}
	if ahora.IsZero() || ahora.Before(origen.metadatos.AlmacenadoEn) {
		return ports.ResultadoOperacionObjeto{}, ports.ErrSolicitudAlmacenInvalida
	}
	evidencia := a.nuevaEvidenciaBloqueada(
		solicitud.Contexto,
		referencia,
		solicitud.EvidenciaAnalisisRef,
		false,
		ahora,
	)
	metadatos := origen.metadatos
	metadatos.Objeto = referencia
	metadatos.Zona = ports.ZonaAlmacenAdmitida
	metadatos.EvidenciaCreacionRef = evidencia.Referencia
	metadatos.AlmacenadoEn = ahora
	metadatos.RetenidoHasta = time.Time{}
	metadatos.Inmovilizado = false
	resultado := ports.ResultadoOperacionObjeto{Objeto: metadatos, Evidencia: evidencia}
	if resultado.ValidarPromocion(
		solicitud,
		origen.metadatos,
		capacidadesAlmacenObjetosMemoria(a.conectorID, a.tamanoMaximo),
	) != nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	a.objetos[claveObjetoAlmacen(referencia)] = objetoAlmacenMemoria{
		metadatos: metadatos,
		contenido: append([]byte(nil), origen.contenido...),
	}
	a.idempotencias[claveIdempotencia] = idempotenciaAlmacenMemoria{
		huellaSolicitud: huellaSolicitud,
		objeto:          referencia,
	}
	return resultado, nil
}

func (a *AlmacenObjetosMemoria) AplicarRetencion(ctx context.Context, solicitud ports.SolicitudRetenerObjeto) (ports.ResultadoOperacionObjeto, error) {
	if err := ctx.Err(); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if err := solicitud.Validar(); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	ahora := a.reloj.Ahora().UTC()
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenAplicarRetencion, ahora); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if err := solicitud.ValidarEn(ahora); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	objeto, existe := a.objetos[claveObjetoAlmacen(solicitud.Objeto)]
	if !existe {
		return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenNoEncontrado
	}
	if objeto.metadatos.Eliminado {
		return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenEliminado
	}
	if objeto.metadatos.Inmovilizado {
		return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenInmovilizado
	}
	if err := validarObjetoAlmacenMemoria(objeto); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if !objeto.metadatos.RetenidoHasta.IsZero() && solicitud.Hasta.Before(objeto.metadatos.RetenidoHasta) {
		return ports.ResultadoOperacionObjeto{}, ports.ErrRetencionObjetoAlmacenVigente
	}
	anterior := objeto.metadatos
	objeto.metadatos.RetenidoHasta = solicitud.Hasta.UTC()
	resultado := ports.ResultadoOperacionObjeto{
		Objeto: objeto.metadatos,
		Evidencia: a.nuevaEvidenciaBloqueada(
			solicitud.Contexto,
			objeto.metadatos.Objeto,
			solicitud.PoliticaRef,
			false,
			ahora,
		),
	}
	if resultado.ValidarRetencion(solicitud, anterior) != nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	a.objetos[claveObjetoAlmacen(solicitud.Objeto)] = objeto
	return resultado, nil
}

func (a *AlmacenObjetosMemoria) Inmovilizar(ctx context.Context, solicitud ports.SolicitudInmovilizarObjeto) (ports.ResultadoOperacionObjeto, error) {
	if err := ctx.Err(); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if err := solicitud.Validar(); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	ahora := a.reloj.Ahora().UTC()
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenInmovilizar, ahora); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	objeto, existe := a.objetos[claveObjetoAlmacen(solicitud.Objeto)]
	if !existe {
		return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenNoEncontrado
	}
	if objeto.metadatos.Eliminado {
		return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenEliminado
	}
	if objeto.metadatos.Inmovilizado {
		return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenInmovilizado
	}
	if err := validarObjetoAlmacenMemoria(objeto); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	anterior := objeto.metadatos
	objeto.metadatos.Inmovilizado = true
	resultado := ports.ResultadoOperacionObjeto{
		Objeto: objeto.metadatos,
		Evidencia: a.nuevaEvidenciaBloqueada(
			solicitud.Contexto,
			objeto.metadatos.Objeto,
			solicitud.AprobacionRef,
			false,
			ahora,
		),
	}
	if resultado.ValidarInmovilizacion(solicitud, anterior) != nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	a.objetos[claveObjetoAlmacen(solicitud.Objeto)] = objeto
	return resultado, nil
}

func (a *AlmacenObjetosMemoria) LevantarInmovilizacion(
	ctx context.Context,
	solicitud ports.SolicitudLevantarInmovilizacionObjeto,
) (ports.ResultadoOperacionObjeto, error) {
	if err := ctx.Err(); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if err := solicitud.Validar(); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	ahora := a.reloj.Ahora().UTC()
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenLevantarInmovilizacion, ahora); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	objeto, existe := a.objetos[claveObjetoAlmacen(solicitud.Objeto)]
	if !existe {
		return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenNoEncontrado
	}
	if objeto.metadatos.Eliminado {
		return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenEliminado
	}
	if !objeto.metadatos.Inmovilizado {
		return ports.ResultadoOperacionObjeto{}, ports.ErrSolicitudAlmacenInvalida
	}
	if err := validarObjetoAlmacenMemoria(objeto); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	anterior := objeto.metadatos
	objeto.metadatos.Inmovilizado = false
	resultado := ports.ResultadoOperacionObjeto{
		Objeto: objeto.metadatos,
		Evidencia: a.nuevaEvidenciaBloqueada(
			solicitud.Contexto,
			objeto.metadatos.Objeto,
			solicitud.AprobacionRef,
			false,
			ahora,
		),
	}
	if resultado.ValidarLevantamientoInmovilizacion(solicitud, anterior) != nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	a.objetos[claveObjetoAlmacen(solicitud.Objeto)] = objeto
	return resultado, nil
}

func (a *AlmacenObjetosMemoria) Eliminar(ctx context.Context, solicitud ports.SolicitudEliminarObjeto) (ports.EvidenciaOperacionAlmacen, error) {
	if err := ctx.Err(); err != nil {
		return ports.EvidenciaOperacionAlmacen{}, err
	}
	if err := solicitud.Validar(); err != nil {
		return ports.EvidenciaOperacionAlmacen{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	ahora := a.reloj.Ahora().UTC()
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenEliminar, ahora); err != nil {
		return ports.EvidenciaOperacionAlmacen{}, err
	}
	clave := claveObjetoAlmacen(solicitud.Objeto)
	objeto, existe := a.objetos[clave]
	if !existe {
		return ports.EvidenciaOperacionAlmacen{}, ports.ErrObjetoAlmacenNoEncontrado
	}
	if objeto.metadatos.Eliminado {
		return ports.EvidenciaOperacionAlmacen{}, ports.ErrObjetoAlmacenEliminado
	}
	if objeto.metadatos.Inmovilizado {
		return ports.EvidenciaOperacionAlmacen{}, ports.ErrObjetoAlmacenInmovilizado
	}
	if err := validarObjetoAlmacenMemoria(objeto); err != nil {
		return ports.EvidenciaOperacionAlmacen{}, err
	}
	if !objeto.metadatos.RetenidoHasta.IsZero() && ahora.Before(objeto.metadatos.RetenidoHasta) {
		return ports.EvidenciaOperacionAlmacen{}, ports.ErrRetencionObjetoAlmacenVigente
	}
	evidencia := a.nuevaEvidenciaBloqueada(
		solicitud.Contexto,
		objeto.metadatos.Objeto,
		solicitud.AprobacionRef,
		false,
		ahora,
	)
	if evidencia.ValidarEliminacion(solicitud, objeto.metadatos) != nil {
		return ports.EvidenciaOperacionAlmacen{}, ports.ErrIntegridadObjetoAlmacen
	}
	objeto.metadatos.Eliminado = true
	objeto.contenido = nil
	a.objetos[clave] = objeto
	return evidencia, nil
}

func leerContenidoExacto(origen io.Reader, tamano int64) ([]byte, error) {
	contenido, err := io.ReadAll(io.LimitReader(origen, tamano+1))
	if err != nil {
		return nil, fmt.Errorf("leer objeto: %w", err)
	}
	if int64(len(contenido)) != tamano {
		return nil, ports.ErrIntegridadObjetoAlmacen
	}
	return contenido, nil
}

func calcularSHA256(contenido []byte) string {
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:])
}

func huellaEscritura(solicitud ports.SolicitudEscribirObjeto) string {
	componentes := []string{
		"escritura-v1",
		string(solicitud.Zona),
		solicitud.MIME,
		strconv.FormatInt(solicitud.Tamano, 10),
		solicitud.HuellaSHA256,
	}
	componentes = append(componentes, componentesContextoAlmacen(solicitud.Contexto)...)
	return calcularSHA256([]byte(strings.Join(componentes, "\x00")))
}

func huellaPromocion(solicitud ports.SolicitudPromoverObjeto, origen ports.ObjetoAlmacenado) string {
	componentes := []string{
		"promocion-v1",
		solicitud.Origen.Referencia,
		solicitud.Origen.Version,
		solicitud.EvidenciaAnalisisRef,
		origen.HuellaSHA256,
	}
	componentes = append(componentes, componentesContextoAlmacen(solicitud.Contexto)...)
	return calcularSHA256([]byte(strings.Join(componentes, "\x00")))
}

func claveObjetoAlmacen(referencia ports.ReferenciaObjetoAlmacen) string {
	return referencia.Referencia + "\x00" + referencia.Version
}

func componentesContextoAlmacen(contexto ports.ContextoOperacionAlmacen) []string {
	proyeccion, err := contexto.Proyeccion()
	if err != nil {
		return nil
	}
	return []string{
		proyeccion.Esquema,
		proyeccion.OperacionRef,
		proyeccion.CorrelacionRef,
		proyeccion.AutorizacionRef,
		proyeccion.Finalidad,
		proyeccion.Clasificacion,
		proyeccion.AccionNegocio,
		proyeccion.AccionTecnica,
		proyeccion.CargaRef,
		proyeccion.SujetoSeudonimoHMAC,
		proyeccion.RecursoRef,
		proyeccion.ModuloID,
		proyeccion.HuellaSolicitudHMAC,
		proyeccion.EfectoRef,
		proyeccion.HuellaPlanEfectoSHA256,
		string(proyeccion.PasoRef),
		proyeccion.HuellaDecisionSHA256,
	}
}

// nuevaEvidenciaBloqueada exige que el llamador mantenga a.mu. El recibo solo
// contiene referencias opacas suficientes para enlazar la auditoria externa.
func (a *AlmacenObjetosMemoria) nuevaEvidenciaBloqueada(
	contexto ports.ContextoOperacionAlmacen,
	objeto ports.ReferenciaObjetoAlmacen,
	fundamentoRef string,
	reintentoIdempotente bool,
	realizadaEn time.Time,
) ports.EvidenciaOperacionAlmacen {
	proyeccion, err := contexto.Proyeccion()
	if err != nil {
		return ports.EvidenciaOperacionAlmacen{}
	}
	a.secuenciaEvidencias++
	return ports.EvidenciaOperacionAlmacen{
		Referencia:             fmt.Sprintf("evidencia-operacion-almacen-%016x", a.secuenciaEvidencias),
		ConectorID:             a.conectorID,
		EsquemaContexto:        proyeccion.Esquema,
		AccionNegocio:          proyeccion.AccionNegocio,
		Accion:                 proyeccion.AccionTecnica,
		EfectoRef:              proyeccion.EfectoRef,
		HuellaPlanEfectoSHA256: proyeccion.HuellaPlanEfectoSHA256,
		HuellaManifiestoSHA256: proyeccion.HuellaManifiestoSHA256,
		HuellaPasoSHA256:       proyeccion.HuellaPasoSHA256,
		PasoRef:                proyeccion.PasoRef,
		HuellaDecisionSHA256:   proyeccion.HuellaDecisionSHA256,
		Objeto:                 objeto,
		OperacionRef:           proyeccion.OperacionRef,
		CorrelacionRef:         proyeccion.CorrelacionRef,
		AutorizacionRef:        proyeccion.AutorizacionRef,
		Finalidad:              proyeccion.Finalidad,
		Clasificacion:          proyeccion.Clasificacion,
		RealizadaEn:            realizadaEn.UTC(),
		CargaRef:               proyeccion.CargaRef,
		SujetoSeudonimoHMAC:    proyeccion.SujetoSeudonimoHMAC,
		RecursoRef:             proyeccion.RecursoRef,
		ModuloID:               proyeccion.ModuloID,
		HuellaSolicitudHMAC:    proyeccion.HuellaSolicitudHMAC,
		FundamentoRef:          fundamentoRef,
		ReintentoIdempotente:   reintentoIdempotente,
	}
}

func validarObjetoAlmacenMemoria(objeto objetoAlmacenMemoria) error {
	if objeto.metadatos.Validar() != nil || objeto.metadatos.Eliminado ||
		int64(len(objeto.contenido)) != objeto.metadatos.Tamano ||
		calcularSHA256(objeto.contenido) != objeto.metadatos.HuellaSHA256 {
		return ports.ErrIntegridadObjetoAlmacen
	}
	return nil
}

var _ ports.AlmacenObjetos = (*AlmacenObjetosMemoria)(nil)
