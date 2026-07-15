// Package almacen contiene adaptadores que componen el puerto documental con
// conectores de objetos. El nucleo no conoce el proveedor seleccionado.
package almacen

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

const prefijoReferenciaContenido = "almacen:v1:"

// ContenidoDocumental adapta el contrato historico de generacion al puerto
// transversal de objetos. Es una fachada temporal: las nuevas cargas deben
// utilizar AlmacenObjetos y su ciclo de cuarentena directamente.
type ContenidoDocumental struct {
	objetos     ports.AlmacenObjetos
	capacidades ports.CapacidadesAlmacenObjetos
}

func NuevoContenidoDocumental(
	ctx context.Context,
	objetos ports.AlmacenObjetos,
	requisitos ports.RequisitosAlmacenObjetos,
) (*ContenidoDocumental, error) {
	if ctx == nil || dependenciaAlmacenNula(objetos) {
		return nil, ports.ErrSolicitudAlmacenInvalida
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	capacidades, err := objetos.Capacidades(ctx)
	if err != nil {
		return nil, fmt.Errorf("consultar capacidades del almacen: %w", err)
	}
	if err := ports.VerificarCapacidadesAlmacen(capacidades, requisitos); err != nil {
		return nil, fmt.Errorf("verificar capacidades del almacen: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &ContenidoDocumental{objetos: objetos, capacidades: capacidades}, nil
}

func (a *ContenidoDocumental) GuardarContenido(
	ctx context.Context,
	solicitud ports.SolicitudGuardarContenido,
) (ports.ContenidoDocumentoGuardado, error) {
	if a == nil || dependenciaAlmacenNula(a.objetos) || ctx == nil {
		return ports.ContenidoDocumentoGuardado{}, ports.ErrSolicitudAlmacenInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ContenidoDocumentoGuardado{}, err
	}
	if err := solicitud.Validar(); err != nil {
		return ports.ContenidoDocumentoGuardado{}, err
	}
	escritura := ports.SolicitudEscribirObjeto{
		Contexto:          solicitud.Contexto,
		ClaveIdempotencia: solicitud.ClaveIdempotencia,
		Zona:              solicitud.Zona,
		MIME:              solicitud.MIME,
		Tamano:            solicitud.Tamano,
		HuellaSHA256:      solicitud.HuellaSHA256,
		Contenido:         bytes.NewReader(solicitud.Contenido),
	}
	// La capacidad se revalida justo antes de delegar el efecto. Una
	// validacion estructural anterior no concede permiso si la decision ha
	// caducado mientras se preparaba la solicitud.
	if err := solicitud.ValidarEn(time.Now().UTC()); err != nil {
		return ports.ContenidoDocumentoGuardado{}, err
	}
	resultado, err := a.objetos.Escribir(ctx, escritura)
	if err != nil {
		return ports.ContenidoDocumentoGuardado{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ContenidoDocumentoGuardado{}, err
	}
	if err := resultado.ValidarEscritura(escritura, a.capacidades); err != nil {
		return ports.ContenidoDocumentoGuardado{}, ports.ErrIntegridadObjetoAlmacen
	}
	if resultado.Objeto.ConectorID != a.capacidades.ConectorID ||
		resultado.Objeto.Zona != solicitud.Zona || resultado.Objeto.MIME != strings.TrimSpace(solicitud.MIME) ||
		resultado.Objeto.Tamano != solicitud.Tamano ||
		!strings.EqualFold(resultado.Objeto.HuellaSHA256, strings.TrimSpace(solicitud.HuellaSHA256)) ||
		!evidenciaCorresponde(resultado.Evidencia, solicitud.Contexto) {
		return ports.ContenidoDocumentoGuardado{}, ports.ErrIntegridadObjetoAlmacen
	}
	// La referencia opaca canonica de la fachada es la identidad que entrega
	// el conector, acompañada de su version. Codificar ambas dentro de otra
	// cadena hacia imposible que Referencia y Evidencia.Objeto describiesen el
	// mismo objeto y obligaba al consumidor a inferir una version al leer.
	guardado := ports.ContenidoDocumentoGuardado{
		ReferenciaLogica:   solicitud.DocumentoID,
		Referencia:         resultado.Objeto.Objeto.Referencia,
		Version:            resultado.Objeto.Objeto.Version,
		ConectorID:         resultado.Objeto.ConectorID,
		Zona:               resultado.Objeto.Zona,
		MIME:               resultado.Objeto.MIME,
		HuellaSHA256:       resultado.Objeto.HuellaSHA256,
		Tamano:             resultado.Objeto.Tamano,
		EvidenciaOperacion: resultado.Evidencia,
	}
	if err := guardado.ValidarContra(solicitud); err != nil {
		return ports.ContenidoDocumentoGuardado{}, ports.ErrIntegridadObjetoAlmacen
	}
	return guardado, nil
}

func (a *ContenidoDocumental) LeerContenido(
	ctx context.Context,
	solicitud ports.SolicitudLeerContenido,
) (ports.ContenidoDocumentoLeido, error) {
	if a == nil || dependenciaAlmacenNula(a.objetos) || ctx == nil {
		return ports.ContenidoDocumentoLeido{}, ports.ErrSolicitudAlmacenInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ContenidoDocumentoLeido{}, err
	}
	if !solicitud.Zona.Valida() || solicitud.Limite < 1 || strings.TrimSpace(solicitud.Referencia) == "" {
		return ports.ContenidoDocumentoLeido{}, ports.ErrSolicitudAlmacenInvalida
	}
	proyeccion, err := solicitud.Contexto.Proyeccion()
	if err != nil || proyeccion.ObjetoVinculado.Validar() != nil {
		return ports.ContenidoDocumentoLeido{}, ports.ErrSolicitudAlmacenInvalida
	}
	referencia := strings.TrimSpace(solicitud.Referencia)
	objeto := proyeccion.ObjetoVinculado
	if strings.HasPrefix(referencia, prefijoReferenciaContenido) {
		// Compatibilidad de lectura exclusivamente para referencias emitidas por
		// la fachada anterior. La version nunca se adivina: el valor decodificado
		// debe coincidir con el objeto exacto autorizado por el PDP.
		objetoLegado, err := decodificarReferencia(referencia)
		if err != nil || objetoLegado != objeto {
			return ports.ContenidoDocumentoLeido{}, ports.ErrSolicitudAlmacenInvalida
		}
	} else if objeto.Referencia != referencia {
		return ports.ContenidoDocumentoLeido{}, ports.ErrSolicitudAlmacenInvalida
	}
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenLeer, time.Now().UTC()); err != nil {
		return ports.ContenidoDocumentoLeido{}, err
	}
	lectura, err := a.objetos.Abrir(ctx, ports.SolicitudAbrirObjeto{
		Contexto: solicitud.Contexto,
		Objeto:   objeto,
		Zona:     solicitud.Zona,
		Limite:   solicitud.Limite,
	})
	if err != nil {
		if errors.Is(err, ports.ErrObjetoAlmacenNoEncontrado) {
			return ports.ContenidoDocumentoLeido{}, errors.Join(ports.ErrContenidoDocumentoNoEncontrado, err)
		}
		if errors.Is(err, ports.ErrLimiteObjetoAlmacenExcedido) {
			return ports.ContenidoDocumentoLeido{}, errors.Join(ports.ErrLimiteLecturaExcedido, err)
		}
		return ports.ContenidoDocumentoLeido{}, err
	}
	if lectura.Contenido == nil {
		return ports.ContenidoDocumentoLeido{}, ports.ErrIntegridadObjetoAlmacen
	}
	defer lectura.Contenido.Close()
	if err := ctx.Err(); err != nil {
		return ports.ContenidoDocumentoLeido{}, err
	}
	if err := lectura.ValidarContra(ports.SolicitudAbrirObjeto{
		Contexto: solicitud.Contexto,
		Objeto:   objeto,
		Zona:     solicitud.Zona,
		Limite:   solicitud.Limite,
	}); err != nil ||
		lectura.Objeto.Objeto != objeto || lectura.Objeto.ConectorID != a.capacidades.ConectorID ||
		lectura.Objeto.Zona != solicitud.Zona || !evidenciaCorresponde(lectura.Evidencia, solicitud.Contexto) {
		return ports.ContenidoDocumentoLeido{}, ports.ErrIntegridadObjetoAlmacen
	}
	limiteEfectivo := solicitud.Limite
	if limiteEfectivo > a.capacidades.TamanoMaximoObjeto {
		limiteEfectivo = a.capacidades.TamanoMaximoObjeto
	}
	contenido, err := io.ReadAll(io.LimitReader(lectura.Contenido, limiteEfectivo+1))
	if err != nil {
		return ports.ContenidoDocumentoLeido{}, fmt.Errorf("leer contenido documental: %w", err)
	}
	if int64(len(contenido)) != lectura.Objeto.Tamano || int64(len(contenido)) > solicitud.Limite {
		return ports.ContenidoDocumentoLeido{}, ports.ErrIntegridadObjetoAlmacen
	}
	suma := sha256.Sum256(contenido)
	huella := hex.EncodeToString(suma[:])
	if !strings.EqualFold(huella, lectura.Objeto.HuellaSHA256) {
		return ports.ContenidoDocumentoLeido{}, ports.ErrIntegridadObjetoAlmacen
	}
	if err := ctx.Err(); err != nil {
		return ports.ContenidoDocumentoLeido{}, err
	}
	return ports.ContenidoDocumentoLeido{
		Contenido:          contenido,
		ConectorID:         lectura.Objeto.ConectorID,
		Zona:               lectura.Objeto.Zona,
		HuellaSHA256:       huella,
		Tamano:             lectura.Objeto.Tamano,
		EvidenciaOperacion: lectura.Evidencia,
	}, nil
}

type referenciaContenidoCodificada struct {
	Referencia string `json:"r"`
	Version    string `json:"v"`
}

func decodificarReferencia(valor string) (ports.ReferenciaObjetoAlmacen, error) {
	valor = strings.TrimSpace(valor)
	if !strings.HasPrefix(valor, prefijoReferenciaContenido) || len(valor) > 1024 {
		return ports.ReferenciaObjetoAlmacen{}, ports.ErrSolicitudAlmacenInvalida
	}
	serializada, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(valor, prefijoReferenciaContenido))
	if err != nil {
		return ports.ReferenciaObjetoAlmacen{}, ports.ErrSolicitudAlmacenInvalida
	}
	var codificada referenciaContenidoCodificada
	decodificador := json.NewDecoder(bytes.NewReader(serializada))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&codificada); err != nil {
		return ports.ReferenciaObjetoAlmacen{}, ports.ErrSolicitudAlmacenInvalida
	}
	var sobrante any
	if err := decodificador.Decode(&sobrante); !errors.Is(err, io.EOF) {
		return ports.ReferenciaObjetoAlmacen{}, ports.ErrSolicitudAlmacenInvalida
	}
	referencia := ports.ReferenciaObjetoAlmacen{Referencia: codificada.Referencia, Version: codificada.Version}
	if err := referencia.Validar(); err != nil {
		return ports.ReferenciaObjetoAlmacen{}, err
	}
	return referencia, nil
}

func evidenciaCorresponde(evidencia ports.EvidenciaOperacionAlmacen, contexto ports.ContextoOperacionAlmacen) bool {
	proyeccion, err := contexto.Proyeccion()
	if err != nil {
		return false
	}
	return evidencia.EsquemaContexto == proyeccion.Esquema &&
		evidencia.OperacionRef == proyeccion.OperacionRef &&
		evidencia.CorrelacionRef == proyeccion.CorrelacionRef &&
		evidencia.AutorizacionRef == proyeccion.AutorizacionRef &&
		evidencia.Finalidad == proyeccion.Finalidad &&
		evidencia.Clasificacion == proyeccion.Clasificacion &&
		evidencia.AccionNegocio == proyeccion.AccionNegocio &&
		evidencia.Accion == proyeccion.AccionTecnica &&
		evidencia.EfectoRef == proyeccion.EfectoRef &&
		evidencia.HuellaPlanEfectoSHA256 == proyeccion.HuellaPlanEfectoSHA256 &&
		evidencia.HuellaManifiestoSHA256 == proyeccion.HuellaManifiestoSHA256 &&
		evidencia.HuellaPasoSHA256 == proyeccion.HuellaPasoSHA256 &&
		evidencia.PasoRef == proyeccion.PasoRef &&
		evidencia.HuellaDecisionSHA256 == proyeccion.HuellaDecisionSHA256 &&
		evidencia.CargaRef == proyeccion.CargaRef &&
		evidencia.SujetoSeudonimoHMAC == proyeccion.SujetoSeudonimoHMAC &&
		evidencia.RecursoRef == proyeccion.RecursoRef &&
		evidencia.ModuloID == proyeccion.ModuloID &&
		evidencia.HuellaSolicitudHMAC == proyeccion.HuellaSolicitudHMAC
}

var _ ports.AlmacenContenidoDocumento = (*ContenidoDocumental)(nil)
