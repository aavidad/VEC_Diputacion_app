package s3

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"strconv"
	"strings"
	"time"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/transport/http"

	"vec-diputacion-granada/internal/vec/ports"
)

const (
	prefijoClaveObjeto       = "vec/v1/"
	prefijoClaveIdempotencia = "vec/control/idempotencia/v1/"
	prefijoReferencia        = "obj-"
	esquemaMetadatos         = "vec-s3-objeto-v1"
	esquemaIdempotencia      = "vec-s3-idempotencia-v1"
)

const (
	metaEsquema       = "vec-esquema"
	metaConector      = "vec-conector"
	metaZona          = "vec-zona"
	metaTamano        = "vec-tamano"
	metaSHA256        = "vec-sha256"
	metaEvidencia     = "vec-evidencia"
	metaAlmacenadoEn  = "vec-almacenado-en"
	metaRetencionBase = "vec-retencion-base"
	metaIdempotencia  = "vec-idempotencia-sha256"
	metaVinculoSesion = "vec-vinculo-sesion-sha256"
)

// Almacen es un adaptador, no un gestor de permisos. Solo ejecuta una
// capacidad opaca ya emitida por el nucleo y la revalida inmediatamente antes
// de cada efecto remoto.
type Almacen struct {
	configuracion Configuracion
	cliente       clienteSDK
	presignador   presignadorSDK
	reloj         ports.Reloj
	capacidades   ports.CapacidadesAlmacenObjetos
}

type relojSistema struct{}

func (relojSistema) Ahora() time.Time { return time.Now().UTC() }

func Nuevo(ctx context.Context, configuracion Configuracion) (*Almacen, error) {
	cliente, presignador, err := nuevoClienteReal(ctx, configuracion)
	if err != nil {
		return nil, err
	}
	return NuevoConCliente(ctx, configuracion, cliente, presignador, relojSistema{})
}

// NuevoConCliente es el punto de inyeccion para pruebas contractuales y para
// cabinas que proporcionen un cliente S3 instrumentado por la organizacion.
func NuevoConCliente(
	ctx context.Context,
	configuracion Configuracion,
	cliente clienteSDK,
	presignador presignadorSDK,
	reloj ports.Reloj,
) (*Almacen, error) {
	if ctx == nil || ctx.Err() != nil || configuracion.Validar() != nil || cliente == nil ||
		presignador == nil || reloj == nil || reloj.Ahora().IsZero() {
		return nil, ErrConfiguracionInvalida
	}
	almacen := &Almacen{configuracion: configuracion, cliente: cliente, presignador: presignador, reloj: reloj}
	capacidades, err := almacen.detectarCapacidades(ctx)
	if err != nil {
		return nil, err
	}
	almacen.capacidades = capacidades
	if configuracion.PerfilFuerte {
		requisitos := ports.RequisitosAlmacenObjetos{
			EscrituraEnFlujo: true, LecturaEnFlujo: true, ReferenciasOpacas: true,
			IntegridadSHA256: true, Versionado: true, Retencion: true, BloqueoLegal: true,
			PromocionAtomica: true, RetencionAtomicaEnPromocion: true,
			CargaDirectaTemporal: true, CifradoEnTransito: true,
			CifradoEnReposo: true, CifradoPorObjeto: true, PreservaObjetoOriginal: true,
			TamanoMinimoObjeto: configuracion.TamanoMaximo,
		}
		if ports.VerificarCapacidadesAlmacen(capacidades, requisitos) != nil {
			return nil, ErrSondaS3NoSuperada
		}
	}
	return almacen, nil
}

func (a *Almacen) Capacidades(ctx context.Context) (ports.CapacidadesAlmacenObjetos, error) {
	if err := contextoValido(ctx); err != nil {
		return ports.CapacidadesAlmacenObjetos{}, err
	}
	resultado := a.capacidades
	resultado.OrigenesCargaDirecta = append([]string(nil), a.capacidades.OrigenesCargaDirecta...)
	return resultado, nil
}

func (a *Almacen) detectarCapacidades(ctx context.Context) (ports.CapacidadesAlmacenObjetos, error) {
	versionado := true
	bloqueoConfigurado := true
	cuarentenaSinRetencionPredeterminada := true
	for _, bucket := range []string{a.configuracion.BucketCuarentena, a.configuracion.BucketAdmitida} {
		if _, err := a.cliente.HeadBucket(ctx, &awss3.HeadBucketInput{Bucket: awsv2.String(bucket)}); err != nil {
			return ports.CapacidadesAlmacenObjetos{}, errorRemoto(ctx, err)
		}
		estado, err := a.cliente.GetBucketVersioning(ctx, &awss3.GetBucketVersioningInput{Bucket: awsv2.String(bucket)})
		if err != nil {
			return ports.CapacidadesAlmacenObjetos{}, errorRemoto(ctx, err)
		}
		versionado = versionado && estado != nil && estado.Status == types.BucketVersioningStatusEnabled
		bloqueo, err := a.cliente.GetObjectLockConfiguration(ctx, &awss3.GetObjectLockConfigurationInput{Bucket: awsv2.String(bucket)})
		if err != nil {
			bloqueoConfigurado = false
			continue
		}
		bloqueoConfigurado = bloqueoConfigurado && bloqueo != nil && bloqueo.ObjectLockConfiguration != nil &&
			bloqueo.ObjectLockConfiguration.ObjectLockEnabled == types.ObjectLockEnabledEnabled
		if bucket == a.configuracion.BucketCuarentena {
			cuarentenaSinRetencionPredeterminada = bloqueo != nil && bloqueo.ObjectLockConfiguration != nil &&
				bloqueo.ObjectLockConfiguration.Rule == nil
		}
	}
	retencionProbada, bloqueoProbado, promocionProbada, checksumProbado, retencionAtomicaProbada := false, false, false, false, false
	if a.configuracion.ProbarCapacidades && versionado && bloqueoConfigurado && cuarentenaSinRetencionPredeterminada {
		retencionProbada, bloqueoProbado, promocionProbada, checksumProbado, retencionAtomicaProbada = a.sondearCapacidades(ctx)
	}
	return ports.CapacidadesAlmacenObjetos{
		ConectorID:       a.configuracion.ConectorID,
		EscrituraEnFlujo: true, LecturaEnFlujo: true, ReferenciasOpacas: true,
		IntegridadSHA256: checksumProbado, Versionado: versionado,
		Retencion: retencionProbada, BloqueoLegal: bloqueoProbado,
		PromocionAtomica: promocionProbada, RetencionAtomicaEnPromocion: retencionAtomicaProbada,
		CargaDirectaTemporal: checksumProbado,
		CifradoEnTransito:    true, CifradoEnReposo: checksumProbado, CifradoPorObjeto: checksumProbado,
		TamanoMaximoObjeto:     a.configuracion.TamanoMaximo,
		PreservaObjetoOriginal: promocionProbada,
		OrigenesCargaDirecta:   []string{a.configuracion.origenCargaDirecta()},
	}, nil
}

// sondearCapacidades realiza operaciones reales, con claves tecnicas sin PII.
// Cuarentena no tiene retencion predeterminada para permitir compensaciones;
// la version admitida nace con retencion explicita en el mismo PUT.
func (a *Almacen) sondearCapacidades(ctx context.Context) (bool, bool, bool, bool, bool) {
	contenido := []byte("sonda-vec-s3-capacidad-fuerte-v1")
	suma := sha256.Sum256(contenido)
	huella := hex.EncodeToString(suma[:])
	ahora := a.reloj.Ahora().UTC()
	referenciaOrigen, err := a.referenciaAleatoriaObjeto()
	if err != nil {
		return false, false, false, false, false
	}
	referenciaHold, err := a.referenciaAleatoriaObjeto()
	if err != nil {
		return false, false, false, false, false
	}
	referenciaDestino, err := a.referenciaAleatoriaObjeto()
	if err != nil {
		return false, false, false, false, false
	}
	metadatosOrigen := metadatosObjeto{
		Referencia: referenciaOrigen, Zona: ports.ZonaAlmacenCuarentena, MIME: "application/octet-stream",
		Tamano: int64(len(contenido)), SHA256: huella, Evidencia: "evidencia-sonda-" + referenciaOrigen,
		AlmacenadoEn: ahora, IdempotenciaSHA256: strings.Repeat("0", 64),
	}
	origen, err := a.poner(ctx, ports.ZonaAlmacenCuarentena, metadatosOrigen, strings.NewReader(string(contenido)), true)
	if err != nil {
		return false, false, false, false, false
	}
	cargado, err := a.cargarObjeto(ctx, origen.Objeto, ports.ZonaAlmacenCuarentena)
	checksumOK := err == nil && cargado.HuellaSHA256 == huella

	// Hold destructivo sobre un objeto independiente sin retencion.
	objetoHold, err := a.poner(ctx, ports.ZonaAlmacenCuarentena, metadatosObjeto{
		Referencia: referenciaHold, Zona: ports.ZonaAlmacenCuarentena, MIME: "application/octet-stream",
		Tamano: int64(len(contenido)), SHA256: huella, Evidencia: "evidencia-sonda-hold-" + referenciaHold,
		AlmacenadoEn: ahora, IdempotenciaSHA256: strings.Repeat("2", 64),
	}, strings.NewReader(string(contenido)), true)
	if err != nil {
		return false, false, false, checksumOK, false
	}
	_, errorHoldOn := a.cliente.PutObjectLegalHold(ctx, &awss3.PutObjectLegalHoldInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(claveObjeto(objetoHold.Objeto.Referencia)),
		VersionId: awsv2.String(objetoHold.Objeto.Version), LegalHold: &types.ObjectLockLegalHold{Status: types.ObjectLockLegalHoldStatusOn},
	})
	holdOn, errorConsultaHold := a.cliente.GetObjectLegalHold(ctx, &awss3.GetObjectLegalHoldInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(claveObjeto(objetoHold.Objeto.Referencia)),
		VersionId: awsv2.String(objetoHold.Objeto.Version),
	})
	_, errorDeleteHold := a.cliente.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(claveObjeto(objetoHold.Objeto.Referencia)),
		VersionId: awsv2.String(objetoHold.Objeto.Version),
	})
	_, errorHoldOff := a.cliente.PutObjectLegalHold(ctx, &awss3.PutObjectLegalHoldInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(claveObjeto(objetoHold.Objeto.Referencia)),
		VersionId: awsv2.String(objetoHold.Objeto.Version), LegalHold: &types.ObjectLockLegalHold{Status: types.ObjectLockLegalHoldStatusOff},
	})
	_, errorDeleteOff := a.cliente.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(claveObjeto(objetoHold.Objeto.Referencia)),
		VersionId: awsv2.String(objetoHold.Objeto.Version),
	})
	holdOK := errorHoldOn == nil && errorConsultaHold == nil && holdOn != nil && holdOn.LegalHold != nil &&
		holdOn.LegalHold.Status == types.ObjectLockLegalHoldStatusOn && errorDeleteHold != nil &&
		errorHoldOff == nil && errorDeleteOff == nil

	// La API de extension de retencion tambien se prueba sobre cuarentena.
	// Margen suficiente para no convertir latencia de una cabina cargada en un
	// falso negativo de WORM durante la propia sonda.
	hastaOrigen := ahora.Add(5 * time.Minute)
	_, errorRetencion := a.cliente.PutObjectRetention(ctx, &awss3.PutObjectRetentionInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(claveObjeto(origen.Objeto.Referencia)),
		VersionId: awsv2.String(origen.Objeto.Version),
		Retention: &types.ObjectLockRetention{Mode: a.configuracion.ModoRetencion, RetainUntilDate: &hastaOrigen},
	})
	retencionOrigen, errorConsultaRetencion := a.cliente.GetObjectRetention(ctx, &awss3.GetObjectRetentionInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(claveObjeto(origen.Objeto.Referencia)),
		VersionId: awsv2.String(origen.Objeto.Version),
	})
	_, errorDeleteRetenido := a.cliente.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(claveObjeto(origen.Objeto.Referencia)),
		VersionId: awsv2.String(origen.Objeto.Version),
	})
	retencionOK := errorRetencion == nil && errorConsultaRetencion == nil && retencionOrigen != nil &&
		retencionOrigen.Retention != nil && retencionOrigen.Retention.Mode == a.configuracion.ModoRetencion &&
		retencionOrigen.Retention.RetainUntilDate != nil && errorDeleteRetenido != nil

	respuesta, err := a.cliente.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(claveObjeto(origen.Objeto.Referencia)),
		VersionId: awsv2.String(origen.Objeto.Version), ChecksumMode: types.ChecksumModeEnabled,
	})
	if err != nil || respuesta == nil || respuesta.Body == nil {
		return retencionOK, holdOK, false, checksumOK, false
	}
	defer respuesta.Body.Close()
	metadatosDestino := metadatosObjeto{
		Referencia: referenciaDestino, Zona: ports.ZonaAlmacenAdmitida, MIME: "application/octet-stream",
		Tamano: int64(len(contenido)), SHA256: huella, Evidencia: "evidencia-sonda-" + referenciaDestino,
		AlmacenadoEn: ahora, RetenidoHasta: ahora.Add(a.configuracion.RetencionMinimaAdmitida),
		IdempotenciaSHA256: strings.Repeat("1", 64),
	}
	destino, err := a.poner(ctx, ports.ZonaAlmacenAdmitida, metadatosDestino, respuesta.Body, true)
	if err != nil {
		return retencionOK, holdOK, false, checksumOK, false
	}
	retencionDestino, errorRetencionDestino := a.cliente.GetObjectRetention(ctx, &awss3.GetObjectRetentionInput{
		Bucket: awsv2.String(a.configuracion.BucketAdmitida), Key: awsv2.String(claveObjeto(destino.Objeto.Referencia)),
		VersionId: awsv2.String(destino.Objeto.Version),
	})
	_, errorDeleteDestino := a.cliente.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: awsv2.String(a.configuracion.BucketAdmitida), Key: awsv2.String(claveObjeto(destino.Objeto.Referencia)),
		VersionId: awsv2.String(destino.Objeto.Version),
	})
	retencionAtomicaOK := destino.Validar() == nil && destino.RetenidoHasta.Equal(metadatosDestino.RetenidoHasta) &&
		errorRetencionDestino == nil && retencionDestino != nil && retencionDestino.Retention != nil &&
		retencionDestino.Retention.Mode == a.configuracion.ModoRetencion && retencionDestino.Retention.RetainUntilDate != nil &&
		retencionDestino.Retention.RetainUntilDate.Equal(metadatosDestino.RetenidoHasta) && errorDeleteDestino != nil
	promocionOK := retencionAtomicaOK && destino.HuellaSHA256 == origen.HuellaSHA256
	if promocionOK {
		origenReleido, errorOrigen := a.cargarObjeto(ctx, origen.Objeto, ports.ZonaAlmacenCuarentena)
		promocionOK = errorOrigen == nil && origenReleido.HuellaSHA256 == huella && origenReleido.Objeto == origen.Objeto
		_, errorHoldDestinoOn := a.cliente.PutObjectLegalHold(ctx, &awss3.PutObjectLegalHoldInput{
			Bucket: awsv2.String(a.configuracion.BucketAdmitida), Key: awsv2.String(claveObjeto(destino.Objeto.Referencia)),
			VersionId: awsv2.String(destino.Objeto.Version), LegalHold: &types.ObjectLockLegalHold{Status: types.ObjectLockLegalHoldStatusOn},
		})
		holdDestino, errorConsultaDestino := a.cliente.GetObjectLegalHold(ctx, &awss3.GetObjectLegalHoldInput{
			Bucket: awsv2.String(a.configuracion.BucketAdmitida), Key: awsv2.String(claveObjeto(destino.Objeto.Referencia)),
			VersionId: awsv2.String(destino.Objeto.Version),
		})
		_, errorHoldDestinoOff := a.cliente.PutObjectLegalHold(ctx, &awss3.PutObjectLegalHoldInput{
			Bucket: awsv2.String(a.configuracion.BucketAdmitida), Key: awsv2.String(claveObjeto(destino.Objeto.Referencia)),
			VersionId: awsv2.String(destino.Objeto.Version), LegalHold: &types.ObjectLockLegalHold{Status: types.ObjectLockLegalHoldStatusOff},
		})
		holdOK = holdOK && errorHoldDestinoOn == nil && errorConsultaDestino == nil && holdDestino != nil &&
			holdDestino.LegalHold != nil && holdDestino.LegalHold.Status == types.ObjectLockLegalHoldStatusOn &&
			errorHoldDestinoOff == nil
	}
	return retencionOK && retencionAtomicaOK, holdOK, promocionOK, checksumOK, retencionAtomicaOK
}

func (a *Almacen) Escribir(ctx context.Context, solicitud ports.SolicitudEscribirObjeto) (ports.ResultadoOperacionObjeto, error) {
	if err := contextoValido(ctx); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if err := solicitud.Validar(); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if !a.capacidades.EscrituraEnFlujo || !a.capacidades.IntegridadSHA256 || !a.capacidades.Versionado ||
		!a.capacidades.CifradoEnReposo || !a.capacidades.CifradoPorObjeto {
		return ports.ResultadoOperacionObjeto{}, ports.ErrCapacidadAlmacenNoDisponible
	}
	if solicitud.Tamano > a.configuracion.TamanoMaximo {
		return ports.ResultadoOperacionObjeto{}, ports.ErrLimiteObjetoAlmacenExcedido
	}
	ahora := a.reloj.Ahora().UTC()
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenEscribir, ahora); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	evidenciaRef, err := a.referenciaAleatoria("evidencia-escritura")
	if err != nil {
		return ports.ResultadoOperacionObjeto{}, ErrOperacionS3
	}
	referencia := a.referenciaIdempotente("escribir", solicitud.ClaveIdempotencia)
	huellaSolicitud := huellaEscritura(solicitud)
	if err := a.reservarIdempotencia(ctx, referencia, huellaSolicitud); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if existente, err := a.cargarObjetoActual(ctx, referencia, solicitud.Zona); err == nil {
		return a.resultadoIdempotente(ctx, solicitud.Contexto, existente, huellaSolicitud, "", evidenciaRef, solicitud)
	} else if !errors.Is(err, ports.ErrObjetoAlmacenNoEncontrado) {
		return ports.ResultadoOperacionObjeto{}, err
	}
	metadatos := metadatosObjeto{
		Referencia: referencia, Zona: solicitud.Zona, MIME: solicitud.MIME, Tamano: solicitud.Tamano,
		SHA256: solicitud.HuellaSHA256, Evidencia: evidenciaRef, AlmacenadoEn: ahora,
		IdempotenciaSHA256: huellaSolicitud,
	}
	if solicitud.Zona == ports.ZonaAlmacenAdmitida {
		if !a.capacidades.RetencionAtomicaEnPromocion {
			return ports.ResultadoOperacionObjeto{}, ports.ErrCapacidadAlmacenNoDisponible
		}
		metadatos.RetenidoHasta = ahora.Add(a.configuracion.RetencionMinimaAdmitida)
	}
	objeto, err := a.poner(ctx, solicitud.Zona, metadatos, solicitud.Contenido, true)
	if err != nil {
		if esPrecondicion(err) {
			existente, cargaErr := a.cargarObjetoActual(ctx, referencia, solicitud.Zona)
			if cargaErr == nil {
				return a.resultadoIdempotente(ctx, solicitud.Contexto, existente, huellaSolicitud, "", evidenciaRef, solicitud)
			}
		}
		return ports.ResultadoOperacionObjeto{}, errorRemoto(ctx, err)
	}
	evidencia := a.nuevaEvidencia(solicitud.Contexto, objeto.Objeto, evidenciaRef, "", false, ahora)
	resultado := ports.ResultadoOperacionObjeto{Objeto: objeto, Evidencia: evidencia}
	if resultado.ValidarEscritura(solicitud, a.capacidades) != nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	return resultado, nil
}

func (a *Almacen) Abrir(ctx context.Context, solicitud ports.SolicitudAbrirObjeto) (ports.LecturaObjetoAlmacen, error) {
	if err := contextoValido(ctx); err != nil {
		return ports.LecturaObjetoAlmacen{}, err
	}
	if err := solicitud.Validar(); err != nil {
		return ports.LecturaObjetoAlmacen{}, err
	}
	if !a.capacidades.LecturaEnFlujo || !a.capacidades.IntegridadSHA256 || !a.capacidades.Versionado {
		return ports.LecturaObjetoAlmacen{}, ports.ErrCapacidadAlmacenNoDisponible
	}
	ahora := a.reloj.Ahora().UTC()
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenLeer, ahora); err != nil {
		return ports.LecturaObjetoAlmacen{}, err
	}
	objeto, err := a.cargarObjeto(ctx, solicitud.Objeto, solicitud.Zona)
	if err != nil {
		return ports.LecturaObjetoAlmacen{}, err
	}
	if objeto.Inmovilizado {
		return ports.LecturaObjetoAlmacen{}, ports.ErrObjetoAlmacenInmovilizado
	}
	if objeto.Tamano > solicitud.Limite {
		return ports.LecturaObjetoAlmacen{}, ports.ErrLimiteObjetoAlmacenExcedido
	}
	evidenciaRef, err := a.referenciaAleatoria("evidencia-lectura")
	if err != nil {
		return ports.LecturaObjetoAlmacen{}, ErrOperacionS3
	}
	respuesta, err := a.cliente.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: awsv2.String(a.bucket(solicitud.Zona)), Key: awsv2.String(claveObjeto(solicitud.Objeto.Referencia)),
		VersionId: awsv2.String(solicitud.Objeto.Version), ChecksumMode: types.ChecksumModeEnabled,
	})
	if err != nil {
		return ports.LecturaObjetoAlmacen{}, errorRemoto(ctx, err)
	}
	if respuesta == nil || respuesta.Body == nil || awsv2.ToString(respuesta.VersionId) != solicitud.Objeto.Version {
		if respuesta != nil && respuesta.Body != nil {
			_ = respuesta.Body.Close()
		}
		return ports.LecturaObjetoAlmacen{}, ports.ErrIntegridadObjetoAlmacen
	}
	if respuesta.ContentLength == nil || awsv2.ToInt64(respuesta.ContentLength) != objeto.Tamano ||
		awsv2.ToString(respuesta.ContentType) != objeto.MIME ||
		awsv2.ToString(respuesta.ChecksumSHA256) != sha256Base64(objeto.HuellaSHA256) ||
		!a.cifradoRespuestaValido(respuesta.ServerSideEncryption, awsv2.ToString(respuesta.SSEKMSKeyId), awsv2.ToBool(respuesta.BucketKeyEnabled)) {
		_ = respuesta.Body.Close()
		return ports.LecturaObjetoAlmacen{}, ports.ErrIntegridadObjetoAlmacen
	}
	contenido := nuevoLectorVerificado(respuesta.Body, objeto.Tamano, objeto.HuellaSHA256)
	lectura := ports.LecturaObjetoAlmacen{
		Objeto:    objeto,
		Evidencia: a.nuevaEvidencia(solicitud.Contexto, objeto.Objeto, evidenciaRef, "", false, ahora),
		Contenido: contenido,
	}
	if lectura.ValidarContra(solicitud) != nil {
		_ = contenido.Close()
		return ports.LecturaObjetoAlmacen{}, ports.ErrIntegridadObjetoAlmacen
	}
	return lectura, nil
}

func (a *Almacen) Promover(ctx context.Context, solicitud ports.SolicitudPromoverObjeto) (ports.ResultadoOperacionObjeto, error) {
	if err := contextoValido(ctx); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if err := solicitud.Validar(); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if !a.capacidades.PromocionAtomica || !a.capacidades.RetencionAtomicaEnPromocion ||
		!a.capacidades.PreservaObjetoOriginal {
		return ports.ResultadoOperacionObjeto{}, ports.ErrCapacidadAlmacenNoDisponible
	}
	ahora := a.reloj.Ahora().UTC()
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenPromover, ahora); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	evidenciaRef, err := a.referenciaAleatoria("evidencia-promocion")
	if err != nil {
		return ports.ResultadoOperacionObjeto{}, ErrOperacionS3
	}
	origen, err := a.cargarObjeto(ctx, solicitud.Origen, ports.ZonaAlmacenCuarentena)
	if err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if origen.Inmovilizado {
		return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenInmovilizado
	}
	referencia := a.referenciaIdempotente("promover", solicitud.ClaveIdempotencia)
	huellaSolicitud := huellaPromocion(solicitud, origen)
	if err := a.reservarIdempotencia(ctx, referencia, huellaSolicitud); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if existente, err := a.cargarObjetoActual(ctx, referencia, ports.ZonaAlmacenAdmitida); err == nil {
		return a.resultadoPromocionIdempotente(ctx, solicitud, origen, existente, huellaSolicitud, evidenciaRef, ahora)
	} else if !errors.Is(err, ports.ErrObjetoAlmacenNoEncontrado) {
		return ports.ResultadoOperacionObjeto{}, err
	}
	respuesta, err := a.cliente.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(claveObjeto(origen.Objeto.Referencia)),
		VersionId: awsv2.String(origen.Objeto.Version), ChecksumMode: types.ChecksumModeEnabled,
	})
	if err != nil {
		return ports.ResultadoOperacionObjeto{}, errorRemoto(ctx, err)
	}
	if respuesta == nil || respuesta.Body == nil || awsv2.ToString(respuesta.VersionId) != origen.Objeto.Version ||
		respuesta.ContentLength == nil || awsv2.ToInt64(respuesta.ContentLength) != origen.Tamano ||
		awsv2.ToString(respuesta.ContentType) != origen.MIME ||
		awsv2.ToString(respuesta.ChecksumSHA256) != sha256Base64(origen.HuellaSHA256) ||
		!a.cifradoRespuestaValido(respuesta.ServerSideEncryption, awsv2.ToString(respuesta.SSEKMSKeyId), awsv2.ToBool(respuesta.BucketKeyEnabled)) {
		if respuesta != nil && respuesta.Body != nil {
			_ = respuesta.Body.Close()
		}
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	defer respuesta.Body.Close()
	metadatos := metadatosObjeto{
		Referencia: referencia, Zona: ports.ZonaAlmacenAdmitida, MIME: origen.MIME, Tamano: origen.Tamano,
		SHA256: origen.HuellaSHA256, Evidencia: evidenciaRef, AlmacenadoEn: ahora,
		RetenidoHasta: ahora.Add(a.configuracion.RetencionMinimaAdmitida), IdempotenciaSHA256: huellaSolicitud,
	}
	promovido, err := a.poner(ctx, ports.ZonaAlmacenAdmitida, metadatos, respuesta.Body, true)
	if err != nil {
		if esPrecondicion(err) {
			existente, errorExistente := a.cargarObjetoActual(ctx, referencia, ports.ZonaAlmacenAdmitida)
			if errorExistente == nil {
				return a.resultadoPromocionIdempotente(ctx, solicitud, origen, existente, huellaSolicitud, evidenciaRef, ahora)
			}
		}
		return ports.ResultadoOperacionObjeto{}, errorRemoto(ctx, err)
	}
	resultado := ports.ResultadoOperacionObjeto{Objeto: promovido, Evidencia: a.nuevaEvidencia(
		solicitud.Contexto, promovido.Objeto, evidenciaRef, solicitud.EvidenciaAnalisisRef, false, ahora,
	)}
	if resultado.ValidarPromocion(solicitud, origen, a.capacidades) != nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	return resultado, nil
}

func (a *Almacen) resultadoPromocionIdempotente(
	ctx context.Context,
	solicitud ports.SolicitudPromoverObjeto,
	origen, existente ports.ObjetoAlmacenado,
	huellaSolicitud, evidenciaRef string,
	ahora time.Time,
) (ports.ResultadoOperacionObjeto, error) {
	metadatosExistentes, err := a.metadatosObjeto(ctx, existente)
	if err != nil || existente.HuellaSHA256 != origen.HuellaSHA256 || existente.Tamano != origen.Tamano ||
		existente.MIME != origen.MIME || metadatosExistentes.IdempotenciaSHA256 != huellaSolicitud {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIdempotenciaAlmacenReutilizada
	}
	resultado := ports.ResultadoOperacionObjeto{Objeto: existente, Evidencia: a.nuevaEvidencia(
		solicitud.Contexto, existente.Objeto, evidenciaRef, solicitud.EvidenciaAnalisisRef, true, ahora,
	)}
	if resultado.ValidarPromocion(solicitud, origen, a.capacidades) != nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	return resultado, nil
}

func (a *Almacen) AplicarRetencion(ctx context.Context, solicitud ports.SolicitudRetenerObjeto) (ports.ResultadoOperacionObjeto, error) {
	if err := contextoValido(ctx); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if !a.capacidades.Retencion {
		return ports.ResultadoOperacionObjeto{}, ports.ErrCapacidadAlmacenNoDisponible
	}
	ahora := a.reloj.Ahora().UTC()
	if solicitud.ValidarEn(ahora) != nil || solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenAplicarRetencion, ahora) != nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrSolicitudAlmacenInvalida
	}
	anterior, zona, err := a.localizarObjeto(ctx, solicitud.Objeto)
	if err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if anterior.Inmovilizado {
		return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenInmovilizado
	}
	if !anterior.RetenidoHasta.IsZero() && solicitud.Hasta.Before(anterior.RetenidoHasta) {
		return ports.ResultadoOperacionObjeto{}, ports.ErrRetencionObjetoAlmacenVigente
	}
	evidenciaRef, err := a.referenciaAleatoria("evidencia-retencion")
	if err != nil {
		return ports.ResultadoOperacionObjeto{}, ErrOperacionS3
	}
	_, err = a.cliente.PutObjectRetention(ctx, &awss3.PutObjectRetentionInput{
		Bucket: awsv2.String(a.bucket(zona)), Key: awsv2.String(claveObjeto(solicitud.Objeto.Referencia)),
		VersionId: awsv2.String(solicitud.Objeto.Version),
		Retention: &types.ObjectLockRetention{Mode: a.configuracion.ModoRetencion, RetainUntilDate: awsv2.Time(solicitud.Hasta.UTC())},
	})
	if err != nil {
		return ports.ResultadoOperacionObjeto{}, errorRemoto(ctx, err)
	}
	confirmacion, err := a.cliente.GetObjectRetention(ctx, &awss3.GetObjectRetentionInput{
		Bucket: awsv2.String(a.bucket(zona)), Key: awsv2.String(claveObjeto(solicitud.Objeto.Referencia)),
		VersionId: awsv2.String(solicitud.Objeto.Version),
	})
	if err != nil || confirmacion == nil || confirmacion.Retention == nil ||
		confirmacion.Retention.Mode != a.configuracion.ModoRetencion || confirmacion.Retention.RetainUntilDate == nil ||
		!confirmacion.Retention.RetainUntilDate.Equal(solicitud.Hasta) {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	actual := anterior
	actual.RetenidoHasta = solicitud.Hasta.UTC()
	resultado := ports.ResultadoOperacionObjeto{Objeto: actual, Evidencia: a.nuevaEvidencia(
		solicitud.Contexto, actual.Objeto, evidenciaRef, solicitud.PoliticaRef, false, ahora,
	)}
	if resultado.ValidarRetencion(solicitud, anterior) != nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	return resultado, nil
}

func (a *Almacen) Inmovilizar(ctx context.Context, solicitud ports.SolicitudInmovilizarObjeto) (ports.ResultadoOperacionObjeto, error) {
	if !a.capacidades.BloqueoLegal {
		return ports.ResultadoOperacionObjeto{}, ports.ErrCapacidadAlmacenNoDisponible
	}
	if solicitud.Validar() != nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrSolicitudAlmacenInvalida
	}
	return a.cambiarBloqueoLegal(ctx, solicitud.Contexto, solicitud.Objeto, solicitud.AprobacionRef, true)
}

func (a *Almacen) LevantarInmovilizacion(ctx context.Context, solicitud ports.SolicitudLevantarInmovilizacionObjeto) (ports.ResultadoOperacionObjeto, error) {
	if !a.capacidades.BloqueoLegal {
		return ports.ResultadoOperacionObjeto{}, ports.ErrCapacidadAlmacenNoDisponible
	}
	if solicitud.Validar() != nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrSolicitudAlmacenInvalida
	}
	return a.cambiarBloqueoLegal(ctx, solicitud.Contexto, solicitud.Objeto, solicitud.AprobacionRef, false)
}

func (a *Almacen) cambiarBloqueoLegal(
	ctx context.Context,
	contexto ports.ContextoOperacionAlmacen,
	objeto ports.ReferenciaObjetoAlmacen,
	fundamento string,
	activar bool,
) (ports.ResultadoOperacionObjeto, error) {
	if err := contextoValido(ctx); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	accion := ports.AccionAlmacenInmovilizar
	if !activar {
		accion = ports.AccionAlmacenLevantarInmovilizacion
	}
	ahora := a.reloj.Ahora().UTC()
	if contexto.ValidarParaEn(accion, ahora) != nil || objeto.Validar() != nil || fundamento == "" {
		return ports.ResultadoOperacionObjeto{}, ports.ErrSolicitudAlmacenInvalida
	}
	anterior, zona, err := a.localizarObjeto(ctx, objeto)
	if err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if anterior.Inmovilizado == activar {
		if activar {
			return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenInmovilizado
		}
		return ports.ResultadoOperacionObjeto{}, ports.ErrSolicitudAlmacenInvalida
	}
	evidenciaRef, err := a.referenciaAleatoria("evidencia-bloqueo")
	if err != nil {
		return ports.ResultadoOperacionObjeto{}, ErrOperacionS3
	}
	estado := types.ObjectLockLegalHoldStatusOff
	if activar {
		estado = types.ObjectLockLegalHoldStatusOn
	}
	_, err = a.cliente.PutObjectLegalHold(ctx, &awss3.PutObjectLegalHoldInput{
		Bucket: awsv2.String(a.bucket(zona)), Key: awsv2.String(claveObjeto(objeto.Referencia)), VersionId: awsv2.String(objeto.Version),
		LegalHold: &types.ObjectLockLegalHold{Status: estado},
	})
	if err != nil {
		return ports.ResultadoOperacionObjeto{}, errorRemoto(ctx, err)
	}
	confirmacion, err := a.cliente.GetObjectLegalHold(ctx, &awss3.GetObjectLegalHoldInput{
		Bucket: awsv2.String(a.bucket(zona)), Key: awsv2.String(claveObjeto(objeto.Referencia)), VersionId: awsv2.String(objeto.Version),
	})
	if err != nil || confirmacion == nil || confirmacion.LegalHold == nil || confirmacion.LegalHold.Status != estado {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	actual := anterior
	actual.Inmovilizado = activar
	resultado := ports.ResultadoOperacionObjeto{Objeto: actual, Evidencia: a.nuevaEvidencia(
		contexto, objeto, evidenciaRef, fundamento, false, ahora,
	)}
	if activar {
		if resultado.ValidarInmovilizacion(ports.SolicitudInmovilizarObjeto{
			Contexto: contexto, Objeto: objeto, AprobacionRef: fundamento, Motivo: "bloqueo_legal_aprobado",
		}, anterior) != nil {
			return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
		}
	} else if resultado.ValidarLevantamientoInmovilizacion(ports.SolicitudLevantarInmovilizacionObjeto{
		Contexto: contexto, Objeto: objeto, AprobacionRef: fundamento, Motivo: "levantamiento_bloqueo_aprobado",
	}, anterior) != nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	return resultado, nil
}

func (a *Almacen) Eliminar(ctx context.Context, solicitud ports.SolicitudEliminarObjeto) (ports.EvidenciaOperacionAlmacen, error) {
	if err := contextoValido(ctx); err != nil {
		return ports.EvidenciaOperacionAlmacen{}, err
	}
	if !a.configuracion.PermitirEliminacion || !a.capacidades.Retencion || !a.capacidades.BloqueoLegal {
		return ports.EvidenciaOperacionAlmacen{}, ports.ErrCapacidadAlmacenNoDisponible
	}
	ahora := a.reloj.Ahora().UTC()
	if solicitud.Validar() != nil || solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenEliminar, ahora) != nil {
		return ports.EvidenciaOperacionAlmacen{}, ports.ErrSolicitudAlmacenInvalida
	}
	anterior, zona, err := a.localizarObjeto(ctx, solicitud.Objeto)
	if err != nil {
		return ports.EvidenciaOperacionAlmacen{}, err
	}
	hold, err := a.cliente.GetObjectLegalHold(ctx, &awss3.GetObjectLegalHoldInput{
		Bucket: awsv2.String(a.bucket(zona)), Key: awsv2.String(claveObjeto(solicitud.Objeto.Referencia)),
		VersionId: awsv2.String(solicitud.Objeto.Version),
	})
	if err != nil || hold == nil || hold.LegalHold == nil {
		return ports.EvidenciaOperacionAlmacen{}, ErrOperacionS3
	}
	retencion, err := a.cliente.GetObjectRetention(ctx, &awss3.GetObjectRetentionInput{
		Bucket: awsv2.String(a.bucket(zona)), Key: awsv2.String(claveObjeto(solicitud.Objeto.Referencia)),
		VersionId: awsv2.String(solicitud.Objeto.Version),
	})
	if err != nil || retencion == nil {
		return ports.EvidenciaOperacionAlmacen{}, ErrOperacionS3
	}
	anterior.Inmovilizado = hold.LegalHold.Status == types.ObjectLockLegalHoldStatusOn
	if retencion.Retention != nil && retencion.Retention.RetainUntilDate != nil {
		anterior.RetenidoHasta = retencion.Retention.RetainUntilDate.UTC()
	}
	if anterior.Inmovilizado {
		return ports.EvidenciaOperacionAlmacen{}, ports.ErrObjetoAlmacenInmovilizado
	}
	if !anterior.RetenidoHasta.IsZero() && ahora.Before(anterior.RetenidoHasta) {
		return ports.EvidenciaOperacionAlmacen{}, ports.ErrRetencionObjetoAlmacenVigente
	}
	evidenciaRef, err := a.referenciaAleatoria("evidencia-eliminacion")
	if err != nil {
		return ports.EvidenciaOperacionAlmacen{}, ErrOperacionS3
	}
	evidencia := a.nuevaEvidencia(solicitud.Contexto, anterior.Objeto, evidenciaRef, solicitud.AprobacionRef, false, ahora)
	if evidencia.ValidarEliminacion(solicitud, anterior) != nil {
		return ports.EvidenciaOperacionAlmacen{}, ports.ErrIntegridadObjetoAlmacen
	}
	_, err = a.cliente.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: awsv2.String(a.bucket(zona)), Key: awsv2.String(claveObjeto(solicitud.Objeto.Referencia)),
		VersionId: awsv2.String(solicitud.Objeto.Version),
	})
	if err != nil {
		return ports.EvidenciaOperacionAlmacen{}, errorRemoto(ctx, err)
	}
	_, err = a.cliente.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: awsv2.String(a.bucket(zona)), Key: awsv2.String(claveObjeto(solicitud.Objeto.Referencia)),
		VersionId: awsv2.String(solicitud.Objeto.Version), ChecksumMode: types.ChecksumModeEnabled,
	})
	if err == nil || !esNoEncontrado(err) {
		return ports.EvidenciaOperacionAlmacen{}, ports.ErrIntegridadObjetoAlmacen
	}
	return evidencia, nil
}

type metadatosObjeto struct {
	Referencia, MIME, SHA256, Evidencia, IdempotenciaSHA256, VinculoSesionSHA256 string
	Zona                                                                         ports.ZonaAlmacen
	Tamano                                                                       int64
	AlmacenadoEn                                                                 time.Time
	RetenidoHasta                                                                time.Time
}

func (a *Almacen) poner(
	ctx context.Context,
	zona ports.ZonaAlmacen,
	metadatos metadatosObjeto,
	contenido io.Reader,
	condicional bool,
) (ports.ObjetoAlmacenado, error) {
	if !referenciaPropiaValida(metadatos.Referencia) || !zona.Valida() || contenido == nil ||
		metadatos.Tamano < 1 || metadatos.Tamano > a.configuracion.TamanoMaximo ||
		!sha256Valido(metadatos.SHA256) || metadatos.AlmacenadoEn.IsZero() {
		return ports.ObjetoAlmacenado{}, ports.ErrSolicitudAlmacenInvalida
	}
	hashCalculado := sha256.New()
	lector := &lectorContado{lector: io.TeeReader(io.LimitReader(contenido, metadatos.Tamano), hashCalculado)}
	entrada := &awss3.PutObjectInput{
		Bucket: awsv2.String(a.bucket(zona)), Key: awsv2.String(claveObjeto(metadatos.Referencia)),
		Body: lector, ContentLength: awsv2.Int64(metadatos.Tamano), ContentType: awsv2.String(metadatos.MIME),
		ChecksumAlgorithm: types.ChecksumAlgorithmSha256, ChecksumSHA256: awsv2.String(sha256Base64(metadatos.SHA256)),
		Metadata: a.codificarMetadatos(metadatos), ServerSideEncryption: a.configuracion.Cifrado,
	}
	if condicional {
		entrada.IfNoneMatch = awsv2.String("*")
	}
	if a.configuracion.Cifrado == types.ServerSideEncryptionAwsKms {
		entrada.SSEKMSKeyId = awsv2.String(a.configuracion.ClaveKMS)
		entrada.BucketKeyEnabled = awsv2.Bool(a.configuracion.UsarBucketKeyKMS)
	}
	if zona == ports.ZonaAlmacenAdmitida {
		if !metadatos.RetenidoHasta.After(metadatos.AlmacenadoEn) {
			return ports.ObjetoAlmacenado{}, ports.ErrSolicitudAlmacenInvalida
		}
		entrada.ObjectLockMode = types.ObjectLockMode(a.configuracion.ModoRetencion)
		entrada.ObjectLockRetainUntilDate = awsv2.Time(metadatos.RetenidoHasta.UTC())
	} else if !metadatos.RetenidoHasta.IsZero() {
		// Cuarentena debe nacer eliminable para poder retirar cargas fallidas.
		// Una retencion posterior se aplica siempre a una version ya verificada.
		return ports.ObjetoAlmacenado{}, ports.ErrSolicitudAlmacenInvalida
	}
	respuesta, err := a.cliente.PutObject(ctx, entrada)
	if err != nil {
		return ports.ObjetoAlmacenado{}, err
	}
	if lector.leidos != metadatos.Tamano || hex.EncodeToString(hashCalculado.Sum(nil)) != metadatos.SHA256 {
		return ports.ObjetoAlmacenado{}, a.falloPosteriorPut(ctx, zona, metadatos, respuesta)
	}
	// ContentLength impide enviar bytes adicionales; aun asi los detectamos en
	// el lector del llamador para que un productor defectuoso no parezca valido.
	var extra [1]byte
	if n, errorExtra := contenido.Read(extra[:]); n != 0 || (errorExtra != nil && !errors.Is(errorExtra, io.EOF)) {
		return ports.ObjetoAlmacenado{}, a.falloPosteriorPut(ctx, zona, metadatos, respuesta)
	}
	version := ""
	if respuesta != nil {
		version = awsv2.ToString(respuesta.VersionId)
	}
	if !versionPropiaValida(version) || respuesta == nil || awsv2.ToString(respuesta.ChecksumSHA256) != sha256Base64(metadatos.SHA256) ||
		!a.cifradoRespuestaValido(respuesta.ServerSideEncryption, awsv2.ToString(respuesta.SSEKMSKeyId), awsv2.ToBool(respuesta.BucketKeyEnabled)) {
		return ports.ObjetoAlmacenado{}, a.falloPosteriorPut(ctx, zona, metadatos, respuesta)
	}
	objeto, err := a.cargarObjeto(ctx, ports.ReferenciaObjetoAlmacen{Referencia: metadatos.Referencia, Version: version}, zona)
	if err != nil {
		return ports.ObjetoAlmacenado{}, a.falloPosteriorPut(ctx, zona, metadatos, respuesta)
	}
	if err := a.verificarProteccionInicial(ctx, zona, metadatos, objeto); err != nil {
		return ports.ObjetoAlmacenado{}, err
	}
	return objeto, nil
}

// verificarProteccionInicial cierra la ventana WORM: una version admitida no
// se considera creada hasta comprobar que la retencion COMPLIANCE declarada en
// el mismo PUT quedo aplicada a esa version exacta. No intenta compensar una
// discrepancia en zona admitida: un backend correcto ya la habra inmovilizado.
func (a *Almacen) verificarProteccionInicial(
	ctx context.Context,
	zona ports.ZonaAlmacen,
	metadatos metadatosObjeto,
	objeto ports.ObjetoAlmacenado,
) error {
	if objeto.Inmovilizado {
		return ports.ErrIntegridadObjetoAlmacen
	}
	if zona == ports.ZonaAlmacenCuarentena {
		if !objeto.RetenidoHasta.IsZero() {
			return ports.ErrIntegridadObjetoAlmacen
		}
		return nil
	}
	if zona != ports.ZonaAlmacenAdmitida || !objeto.RetenidoHasta.Equal(metadatos.RetenidoHasta) {
		return ports.ErrIntegridadObjetoAlmacen
	}
	retencion, err := a.cliente.GetObjectRetention(ctx, &awss3.GetObjectRetentionInput{
		Bucket: awsv2.String(a.bucket(zona)), Key: awsv2.String(claveObjeto(objeto.Objeto.Referencia)),
		VersionId: awsv2.String(objeto.Objeto.Version),
	})
	if err != nil {
		return errorRemoto(ctx, err)
	}
	if retencion == nil || retencion.Retention == nil || retencion.Retention.Mode != a.configuracion.ModoRetencion ||
		retencion.Retention.RetainUntilDate == nil || !retencion.Retention.RetainUntilDate.Equal(metadatos.RetenidoHasta) {
		return ports.ErrIntegridadObjetoAlmacen
	}
	return nil
}

// falloPosteriorPut intenta retirar una version que no pudo verificarse. Usa
// un contexto de compensacion corto aunque el solicitante haya cancelado: el
// efecto remoto ya ocurrio y dejar una clave idempotente envenenada seria peor.
// Nunca borra una version cuya metadata no pruebe que pertenece a este PUT.
func (a *Almacen) falloPosteriorPut(
	ctx context.Context,
	zona ports.ZonaAlmacen,
	metadatos metadatosObjeto,
	respuesta *awss3.PutObjectOutput,
) error {
	var base context.Context = context.Background()
	if ctx != nil {
		base = context.WithoutCancel(ctx)
	}
	compensacion, cancelar := context.WithTimeout(base, 10*time.Second)
	defer cancelar()
	version := ""
	if respuesta != nil {
		version = awsv2.ToString(respuesta.VersionId)
	}
	entradaHead := &awss3.HeadObjectInput{
		Bucket: awsv2.String(a.bucket(zona)), Key: awsv2.String(claveObjeto(metadatos.Referencia)),
		ChecksumMode: types.ChecksumModeEnabled,
	}
	if versionPropiaValida(version) {
		entradaHead.VersionId = awsv2.String(version)
	}
	head, err := a.cliente.HeadObject(compensacion, entradaHead)
	if err != nil || head == nil || head.Metadata[metaEsquema] != esquemaMetadatos ||
		head.Metadata[metaConector] != a.configuracion.ConectorID ||
		head.Metadata[metaIdempotencia] != metadatos.IdempotenciaSHA256 || !versionPropiaValida(awsv2.ToString(head.VersionId)) {
		return ports.ErrIntegridadObjetoAlmacen
	}
	version = awsv2.ToString(head.VersionId)
	_, err = a.cliente.DeleteObject(compensacion, &awss3.DeleteObjectInput{
		Bucket: awsv2.String(a.bucket(zona)), Key: awsv2.String(claveObjeto(metadatos.Referencia)), VersionId: awsv2.String(version),
	})
	if err != nil {
		return ports.ErrIntegridadObjetoAlmacen
	}
	_, err = a.cliente.HeadObject(compensacion, &awss3.HeadObjectInput{
		Bucket: awsv2.String(a.bucket(zona)), Key: awsv2.String(claveObjeto(metadatos.Referencia)), VersionId: awsv2.String(version),
		ChecksumMode: types.ChecksumModeEnabled,
	})
	if err == nil || !esNoEncontrado(err) {
		return ports.ErrIntegridadObjetoAlmacen
	}
	return ports.ErrIntegridadObjetoAlmacen
}

func (a *Almacen) cargarObjetoActual(ctx context.Context, referencia string, zona ports.ZonaAlmacen) (ports.ObjetoAlmacenado, error) {
	if !referenciaPropiaValida(referencia) {
		return ports.ObjetoAlmacenado{}, ports.ErrSolicitudAlmacenInvalida
	}
	respuesta, err := a.cliente.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: awsv2.String(a.bucket(zona)), Key: awsv2.String(claveObjeto(referencia)), ChecksumMode: types.ChecksumModeEnabled,
	})
	if err != nil {
		return ports.ObjetoAlmacenado{}, errorRemoto(ctx, err)
	}
	return a.objetoDesdeHead(ctx, referencia, zona, respuesta)
}

func (a *Almacen) cargarObjeto(ctx context.Context, objeto ports.ReferenciaObjetoAlmacen, zona ports.ZonaAlmacen) (ports.ObjetoAlmacenado, error) {
	if objeto.Validar() != nil || !referenciaPropiaValida(objeto.Referencia) || !versionPropiaValida(objeto.Version) || !zona.Valida() {
		return ports.ObjetoAlmacenado{}, ports.ErrSolicitudAlmacenInvalida
	}
	respuesta, err := a.cliente.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: awsv2.String(a.bucket(zona)), Key: awsv2.String(claveObjeto(objeto.Referencia)),
		VersionId: awsv2.String(objeto.Version), ChecksumMode: types.ChecksumModeEnabled,
	})
	if err != nil {
		return ports.ObjetoAlmacenado{}, errorRemoto(ctx, err)
	}
	resultado, err := a.objetoDesdeHead(ctx, objeto.Referencia, zona, respuesta)
	if err != nil || resultado.Objeto != objeto {
		return ports.ObjetoAlmacenado{}, ports.ErrIntegridadObjetoAlmacen
	}
	return resultado, nil
}

func (a *Almacen) objetoDesdeHead(
	ctx context.Context,
	referencia string,
	zona ports.ZonaAlmacen,
	respuesta *awss3.HeadObjectOutput,
) (ports.ObjetoAlmacenado, error) {
	if respuesta == nil || !versionPropiaValida(awsv2.ToString(respuesta.VersionId)) ||
		respuesta.ContentLength == nil || respuesta.ContentType == nil || respuesta.LastModified == nil ||
		!a.cifradoRespuestaValido(respuesta.ServerSideEncryption, awsv2.ToString(respuesta.SSEKMSKeyId), awsv2.ToBool(respuesta.BucketKeyEnabled)) {
		return ports.ObjetoAlmacenado{}, ports.ErrIntegridadObjetoAlmacen
	}
	metadatos, err := a.decodificarMetadatos(referencia, zona, respuesta.Metadata)
	metadatos.MIME = awsv2.ToString(respuesta.ContentType)
	if err != nil || metadatos.Tamano != awsv2.ToInt64(respuesta.ContentLength) ||
		sha256Base64(metadatos.SHA256) != awsv2.ToString(respuesta.ChecksumSHA256) {
		return ports.ObjetoAlmacenado{}, ports.ErrIntegridadObjetoAlmacen
	}
	objeto := ports.ObjetoAlmacenado{
		Objeto:     ports.ReferenciaObjetoAlmacen{Referencia: referencia, Version: awsv2.ToString(respuesta.VersionId)},
		ConectorID: a.configuracion.ConectorID, Zona: zona, MIME: metadatos.MIME, Tamano: metadatos.Tamano,
		HuellaSHA256: metadatos.SHA256, EvidenciaCreacionRef: metadatos.Evidencia, AlmacenadoEn: metadatos.AlmacenadoEn,
	}
	if respuesta.ObjectLockRetainUntilDate != nil {
		objeto.RetenidoHasta = respuesta.ObjectLockRetainUntilDate.UTC()
	}
	objeto.Inmovilizado = respuesta.ObjectLockLegalHoldStatus == types.ObjectLockLegalHoldStatusOn
	if zona == ports.ZonaAlmacenAdmitida && (metadatos.RetenidoHasta.IsZero() || objeto.RetenidoHasta.Before(metadatos.RetenidoHasta) ||
		respuesta.ObjectLockMode != types.ObjectLockMode(a.configuracion.ModoRetencion)) {
		return ports.ObjetoAlmacenado{}, ports.ErrIntegridadObjetoAlmacen
	}
	if objeto.Validar() != nil {
		return ports.ObjetoAlmacenado{}, ports.ErrIntegridadObjetoAlmacen
	}
	return objeto, nil
}

func (a *Almacen) localizarObjeto(ctx context.Context, objeto ports.ReferenciaObjetoAlmacen) (ports.ObjetoAlmacenado, ports.ZonaAlmacen, error) {
	for _, zona := range []ports.ZonaAlmacen{ports.ZonaAlmacenCuarentena, ports.ZonaAlmacenAdmitida} {
		resultado, err := a.cargarObjeto(ctx, objeto, zona)
		if err == nil {
			return resultado, zona, nil
		}
		if !errors.Is(err, ports.ErrObjetoAlmacenNoEncontrado) {
			return ports.ObjetoAlmacenado{}, "", err
		}
	}
	return ports.ObjetoAlmacenado{}, "", ports.ErrObjetoAlmacenNoEncontrado
}

func (a *Almacen) codificarMetadatos(m metadatosObjeto) map[string]string {
	return map[string]string{
		metaEsquema: esquemaMetadatos, metaConector: a.configuracion.ConectorID, metaZona: string(m.Zona),
		metaTamano: strconv.FormatInt(m.Tamano, 10), metaSHA256: m.SHA256, metaEvidencia: m.Evidencia,
		metaAlmacenadoEn: m.AlmacenadoEn.UTC().Format(time.RFC3339Nano), metaRetencionBase: formatearTiempoOpcional(m.RetenidoHasta),
		metaIdempotencia:  m.IdempotenciaSHA256,
		metaVinculoSesion: m.VinculoSesionSHA256,
	}
}

func (a *Almacen) decodificarMetadatos(referencia string, zona ports.ZonaAlmacen, valores map[string]string) (metadatosObjeto, error) {
	tamano, err := strconv.ParseInt(valores[metaTamano], 10, 64)
	if err != nil {
		return metadatosObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	almacenadoEn, err := time.Parse(time.RFC3339Nano, valores[metaAlmacenadoEn])
	retencionBase, errorRetencion := parsearTiempoOpcional(valores[metaRetencionBase])
	resultado := metadatosObjeto{
		Referencia: referencia, Zona: ports.ZonaAlmacen(valores[metaZona]), MIME: "", Tamano: tamano,
		SHA256: valores[metaSHA256], Evidencia: valores[metaEvidencia], AlmacenadoEn: almacenadoEn,
		RetenidoHasta: retencionBase, IdempotenciaSHA256: valores[metaIdempotencia], VinculoSesionSHA256: valores[metaVinculoSesion],
	}
	if err != nil || errorRetencion != nil || valores[metaEsquema] != esquemaMetadatos || valores[metaConector] != a.configuracion.ConectorID ||
		resultado.Zona != zona || !sha256Valido(resultado.SHA256) || !sha256Valido(resultado.IdempotenciaSHA256) ||
		(resultado.VinculoSesionSHA256 != "" && !sha256Valido(resultado.VinculoSesionSHA256)) ||
		!textoTecnicoValido(resultado.Evidencia, 512) || !resultado.AlmacenadoEn.Equal(resultado.AlmacenadoEn.UTC()) ||
		!resultado.RetenidoHasta.Equal(resultado.RetenidoHasta.UTC()) ||
		(zona == ports.ZonaAlmacenAdmitida && !resultado.RetenidoHasta.After(resultado.AlmacenadoEn)) {
		return metadatosObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	return resultado, nil
}

func formatearTiempoOpcional(valor time.Time) string {
	if valor.IsZero() {
		return ""
	}
	return valor.UTC().Format(time.RFC3339Nano)
}

func parsearTiempoOpcional(valor string) (time.Time, error) {
	if valor == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, valor)
}

func (a *Almacen) cifradoRespuestaValido(cifrado types.ServerSideEncryption, claveKMS string, bucketKey bool) bool {
	if cifrado != a.configuracion.Cifrado {
		return false
	}
	if a.configuracion.Cifrado != types.ServerSideEncryptionAwsKms {
		return true
	}
	return claveKMS == a.configuracion.ClaveKMS && (!a.configuracion.UsarBucketKeyKMS || bucketKey)
}

func (a *Almacen) nuevaEvidencia(
	contexto ports.ContextoOperacionAlmacen,
	objeto ports.ReferenciaObjetoAlmacen,
	referencia, fundamento string,
	reintento bool,
	realizadaEn time.Time,
) ports.EvidenciaOperacionAlmacen {
	proyeccion, err := contexto.Proyeccion()
	if err != nil {
		return ports.EvidenciaOperacionAlmacen{}
	}
	return ports.EvidenciaOperacionAlmacen{
		Referencia: referencia, ConectorID: a.configuracion.ConectorID, EsquemaContexto: proyeccion.Esquema,
		AccionNegocio: proyeccion.AccionNegocio, Accion: proyeccion.AccionTecnica,
		EfectoRef: proyeccion.EfectoRef, HuellaPlanEfectoSHA256: proyeccion.HuellaPlanEfectoSHA256,
		HuellaManifiestoSHA256: proyeccion.HuellaManifiestoSHA256, HuellaPasoSHA256: proyeccion.HuellaPasoSHA256,
		PasoRef: proyeccion.PasoRef, HuellaDecisionSHA256: proyeccion.HuellaDecisionSHA256,
		Objeto: objeto, OperacionRef: proyeccion.OperacionRef, CorrelacionRef: proyeccion.CorrelacionRef,
		AutorizacionRef: proyeccion.AutorizacionRef, Finalidad: proyeccion.Finalidad,
		Clasificacion: proyeccion.Clasificacion, RealizadaEn: realizadaEn.UTC(), CargaRef: proyeccion.CargaRef,
		SujetoSeudonimoHMAC: proyeccion.SujetoSeudonimoHMAC, RecursoRef: proyeccion.RecursoRef,
		ModuloID: proyeccion.ModuloID, HuellaSolicitudHMAC: proyeccion.HuellaSolicitudHMAC,
		FundamentoRef: fundamento, ReintentoIdempotente: reintento,
	}
}

func (a *Almacen) resultadoIdempotente(
	ctx context.Context,
	contexto ports.ContextoOperacionAlmacen,
	objeto ports.ObjetoAlmacenado,
	huellaSolicitud, fundamento, evidenciaRef string,
	solicitud ports.SolicitudEscribirObjeto,
) (ports.ResultadoOperacionObjeto, error) {
	metadatos, err := a.metadatosObjeto(ctx, objeto)
	if err != nil || metadatos.IdempotenciaSHA256 != huellaSolicitud {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIdempotenciaAlmacenReutilizada
	}
	if objeto.Inmovilizado {
		return ports.ResultadoOperacionObjeto{}, ports.ErrObjetoAlmacenInmovilizado
	}
	ahora := a.reloj.Ahora().UTC()
	resultado := ports.ResultadoOperacionObjeto{Objeto: objeto, Evidencia: a.nuevaEvidencia(
		contexto, objeto.Objeto, evidenciaRef, fundamento, true, ahora,
	)}
	if resultado.ValidarEscritura(solicitud, a.capacidades) != nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	return resultado, nil
}

// metadatosObjeto vuelve a consultar HEAD para obtener la huella de
// idempotencia, que no forma parte deliberadamente del puerto publico.
func (a *Almacen) metadatosObjeto(ctx context.Context, objeto ports.ObjetoAlmacenado) (metadatosObjeto, error) {
	respuesta, err := a.cliente.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: awsv2.String(a.bucket(objeto.Zona)), Key: awsv2.String(claveObjeto(objeto.Objeto.Referencia)),
		VersionId: awsv2.String(objeto.Objeto.Version), ChecksumMode: types.ChecksumModeEnabled,
	})
	if err != nil {
		return metadatosObjeto{}, errorRemoto(ctx, err)
	}
	metadatos, err := a.decodificarMetadatos(objeto.Objeto.Referencia, objeto.Zona, respuesta.Metadata)
	if err == nil {
		metadatos.MIME = awsv2.ToString(respuesta.ContentType)
	}
	return metadatos, err
}

// reservarIdempotencia usa una unica clave tecnica en cuarentena para
// serializar operaciones de cualquier zona y modulo. El marcador solo contiene
// hashes y nace con legal hold; no puede borrarse por una carrera ni reutilizarse
// para otra peticion. Un proceso que cae tras reservar puede continuar la misma
// huella, pero nunca sustituirla.
func (a *Almacen) reservarIdempotencia(ctx context.Context, referencia, huellaSolicitud string) error {
	if !referenciaPropiaValida(referencia) || !sha256Valido(huellaSolicitud) || !a.capacidades.BloqueoLegal {
		return ports.ErrCapacidadAlmacenNoDisponible
	}
	contenido := []byte(huellaSolicitud)
	suma := sha256.Sum256(contenido)
	checksum := base64.StdEncoding.EncodeToString(suma[:])
	entrada := &awss3.PutObjectInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(prefijoClaveIdempotencia + referencia),
		Body: bytes.NewReader(contenido), ContentLength: awsv2.Int64(int64(len(contenido))),
		ContentType: awsv2.String("application/octet-stream"), IfNoneMatch: awsv2.String("*"),
		ChecksumAlgorithm: types.ChecksumAlgorithmSha256, ChecksumSHA256: awsv2.String(checksum),
		Metadata: map[string]string{
			metaEsquema: esquemaIdempotencia, metaConector: a.configuracion.ConectorID,
			metaIdempotencia: huellaSolicitud, "vec-referencia": referencia,
		},
		ServerSideEncryption:      a.configuracion.Cifrado,
		ObjectLockLegalHoldStatus: types.ObjectLockLegalHoldStatusOn,
	}
	if a.configuracion.Cifrado == types.ServerSideEncryptionAwsKms {
		entrada.SSEKMSKeyId = awsv2.String(a.configuracion.ClaveKMS)
		entrada.BucketKeyEnabled = awsv2.Bool(a.configuracion.UsarBucketKeyKMS)
	}
	respuesta, err := a.cliente.PutObject(ctx, entrada)
	if err != nil {
		if esPrecondicion(err) {
			return a.cotejarReservaIdempotencia(ctx, referencia, huellaSolicitud)
		}
		return errorRemoto(ctx, err)
	}
	if respuesta == nil || !versionPropiaValida(awsv2.ToString(respuesta.VersionId)) ||
		awsv2.ToString(respuesta.ChecksumSHA256) != checksum ||
		!a.cifradoRespuestaValido(respuesta.ServerSideEncryption, awsv2.ToString(respuesta.SSEKMSKeyId), awsv2.ToBool(respuesta.BucketKeyEnabled)) {
		// El PUT pudo haberse confirmado pese a una respuesta incompleta. HEAD
		// decide sin repetir ni exponer el error remoto.
		return a.cotejarReservaIdempotencia(ctx, referencia, huellaSolicitud)
	}
	return a.cotejarReservaIdempotencia(ctx, referencia, huellaSolicitud)
}

func (a *Almacen) cotejarReservaIdempotencia(ctx context.Context, referencia, huellaSolicitud string) error {
	respuesta, err := a.cliente.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(prefijoClaveIdempotencia + referencia),
		ChecksumMode: types.ChecksumModeEnabled,
	})
	if err != nil {
		return errorRemoto(ctx, err)
	}
	if respuesta == nil {
		return ports.ErrIntegridadObjetoAlmacen
	}
	huellaReservada := respuesta.Metadata[metaIdempotencia]
	if !sha256Valido(huellaReservada) {
		return ports.ErrIntegridadObjetoAlmacen
	}
	suma := sha256.Sum256([]byte(huellaReservada))
	checksum := base64.StdEncoding.EncodeToString(suma[:])
	if !versionPropiaValida(awsv2.ToString(respuesta.VersionId)) ||
		respuesta.Metadata[metaEsquema] != esquemaIdempotencia ||
		respuesta.Metadata[metaConector] != a.configuracion.ConectorID ||
		respuesta.Metadata["vec-referencia"] != referencia ||
		respuesta.ObjectLockLegalHoldStatus != types.ObjectLockLegalHoldStatusOn ||
		awsv2.ToString(respuesta.ChecksumSHA256) != checksum ||
		!a.cifradoRespuestaValido(respuesta.ServerSideEncryption, awsv2.ToString(respuesta.SSEKMSKeyId), awsv2.ToBool(respuesta.BucketKeyEnabled)) {
		return ports.ErrIntegridadObjetoAlmacen
	}
	if huellaReservada != huellaSolicitud {
		return ports.ErrIdempotenciaAlmacenReutilizada
	}
	return nil
}

func (a *Almacen) referenciaIdempotente(accion, clave string) string {
	mac := hmac.New(sha256.New, a.configuracion.ClaveDerivacion)
	// El espacio es global a todas las operaciones. Reutilizar la misma clave
	// en escritura, promocion o carga directa debe colisionar, no crear dos
	// efectos en buckets distintos. accion queda en la firma por compatibilidad
	// de llamada, pero no participa en la identidad.
	_ = accion
	_, _ = io.WriteString(mac, "vec-s3-referencia-v2\x00"+clave)
	return prefijoReferencia + hex.EncodeToString(mac.Sum(nil))
}

func (a *Almacen) referenciaAleatoria(prefijo string) (string, error) {
	nonce := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, a.configuracion.ClaveDerivacion)
	_, _ = mac.Write(nonce)
	return prefijo + "-" + hex.EncodeToString(mac.Sum(nil)), nil
}

func (a *Almacen) referenciaAleatoriaObjeto() (string, error) {
	nonce := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, a.configuracion.ClaveDerivacion)
	_, _ = io.WriteString(mac, "vec-s3-objeto-aleatorio-v1\x00")
	_, _ = mac.Write(nonce)
	return prefijoReferencia + hex.EncodeToString(mac.Sum(nil)), nil
}

func (a *Almacen) bucket(zona ports.ZonaAlmacen) string {
	if zona == ports.ZonaAlmacenCuarentena {
		return a.configuracion.BucketCuarentena
	}
	return a.configuracion.BucketAdmitida
}

func claveObjeto(referencia string) string { return prefijoClaveObjeto + referencia }

func referenciaPropiaValida(valor string) bool {
	if !strings.HasPrefix(valor, prefijoReferencia) || len(valor) != len(prefijoReferencia)+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(valor, prefijoReferencia))
	return err == nil && valor == strings.ToLower(valor) && !strings.Contains(valor, "..") && !strings.ContainsAny(valor, "/\\")
}

func versionPropiaValida(valor string) bool {
	return textoTecnicoValido(valor, 256) && valor != "null" && valor != "0" && !strings.ContainsAny(valor, "/\\?#") && !strings.Contains(valor, "..")
}

func sha256Valido(valor string) bool {
	if len(valor) != 64 || valor != strings.ToLower(valor) {
		return false
	}
	bytes, err := hex.DecodeString(valor)
	return err == nil && len(bytes) == sha256.Size
}

func sha256Base64(valor string) string {
	bytes, err := hex.DecodeString(valor)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(bytes)
}

func huellaEscritura(s ports.SolicitudEscribirObjeto) string {
	componentes := []string{"escritura-s3-v1", string(s.Zona), s.MIME, strconv.FormatInt(s.Tamano, 10), s.HuellaSHA256}
	componentes = append(componentes, componentesContexto(s.Contexto)...)
	return sumaComponentes(componentes...)
}

func huellaPromocion(s ports.SolicitudPromoverObjeto, origen ports.ObjetoAlmacenado) string {
	componentes := []string{"promocion-s3-v1", s.Origen.Referencia, s.Origen.Version, s.EvidenciaAnalisisRef, origen.HuellaSHA256}
	componentes = append(componentes, componentesContexto(s.Contexto)...)
	return sumaComponentes(componentes...)
}

func componentesContexto(contexto ports.ContextoOperacionAlmacen) []string {
	p, err := contexto.Proyeccion()
	if err != nil {
		return nil
	}
	return []string{p.Esquema, p.OperacionRef, p.CorrelacionRef, p.AutorizacionRef, p.Finalidad,
		p.Clasificacion, p.AccionNegocio, p.AccionTecnica, p.CargaRef, p.SujetoSeudonimoHMAC,
		p.RecursoRef, p.ModuloID, p.HuellaSolicitudHMAC, p.EfectoRef, p.HuellaPlanEfectoSHA256,
		string(p.PasoRef), p.HuellaDecisionSHA256}
}

func sumaComponentes(componentes ...string) string {
	suma := sha256.Sum256([]byte(strings.Join(componentes, "\x00")))
	return hex.EncodeToString(suma[:])
}

func contextoValido(ctx context.Context) error {
	if ctx == nil {
		return context.Canceled
	}
	return ctx.Err()
}

func errorRemoto(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if esNoEncontrado(err) {
		return ports.ErrObjetoAlmacenNoEncontrado
	}
	if errors.Is(err, ports.ErrIntegridadObjetoAlmacen) || errors.Is(err, ports.ErrLimiteObjetoAlmacenExcedido) {
		return err
	}
	return ErrOperacionS3
}

func codigoAPI(err error) string {
	var api smithy.APIError
	if errors.As(err, &api) {
		return api.ErrorCode()
	}
	return ""
}

func esNoEncontrado(err error) bool {
	codigo := codigoAPI(err)
	if codigo == "NoSuchKey" || codigo == "NoSuchVersion" || codigo == "NotFound" {
		return true
	}
	var respuesta *http.ResponseError
	return errors.As(err, &respuesta) && respuesta.HTTPStatusCode() == 404
}

func esPrecondicion(err error) bool {
	codigo := codigoAPI(err)
	if codigo == "PreconditionFailed" || codigo == "ConditionalRequestConflict" {
		return true
	}
	var respuesta *http.ResponseError
	return errors.As(err, &respuesta) && (respuesta.HTTPStatusCode() == 409 || respuesta.HTTPStatusCode() == 412)
}

type lectorContado struct {
	lector io.Reader
	leidos int64
}

func (l *lectorContado) Read(p []byte) (int, error) {
	n, err := l.lector.Read(p)
	l.leidos += int64(n)
	return n, err
}

type lectorVerificado struct {
	origen   io.ReadCloser
	hash     hash.Hash
	esperado int64
	leidos   int64
	huella   string
	fallo    error
}

func nuevoLectorVerificado(origen io.ReadCloser, esperado int64, huella string) io.ReadCloser {
	return &lectorVerificado{origen: origen, hash: sha256.New(), esperado: esperado, huella: huella}
}

func (l *lectorVerificado) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if l.fallo != nil {
		return 0, l.fallo
	}
	if l.leidos >= l.esperado {
		var extra [1]byte
		n, err := l.origen.Read(extra[:])
		if n != 0 {
			l.fallo = ports.ErrIntegridadObjetoAlmacen
			return 0, l.fallo
		}
		if errors.Is(err, io.EOF) {
			if hex.EncodeToString(l.hash.Sum(nil)) != l.huella {
				l.fallo = ports.ErrIntegridadObjetoAlmacen
				return 0, l.fallo
			}
			return 0, io.EOF
		}
		return 0, err
	}
	maximo := int64(len(p))
	if restante := l.esperado - l.leidos; maximo > restante {
		maximo = restante
	}
	n, err := l.origen.Read(p[:maximo])
	if n > 0 {
		_, _ = l.hash.Write(p[:n])
		l.leidos += int64(n)
	}
	if l.leidos == l.esperado {
		if err != nil && !errors.Is(err, io.EOF) {
			return n, err
		}
		var extra [1]byte
		extraN, extraErr := l.origen.Read(extra[:])
		if extraN != 0 || (extraErr != nil && !errors.Is(extraErr, io.EOF)) ||
			hex.EncodeToString(l.hash.Sum(nil)) != l.huella {
			l.fallo = ports.ErrIntegridadObjetoAlmacen
			return n, l.fallo
		}
		return n, io.EOF
	}
	if errors.Is(err, io.EOF) {
		l.fallo = ports.ErrIntegridadObjetoAlmacen
		return n, l.fallo
	}
	return n, err
}

func (l *lectorVerificado) Close() error { return l.origen.Close() }

var _ ports.AlmacenObjetos = (*Almacen)(nil)

// Evita que una refactorizacion elimine accidentalmente el hash de flujo.
var _ hash.Hash = sha256.New()
