package s3

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"vec-diputacion-granada/internal/vec/ports"
)

const (
	prefijoClaveCargaDirecta  = "vec/cargas/v1/"
	prefijoSesionCargaDirecta = "sesion-s3-"
	esquemaCargaDirecta       = "vec-s3-carga-directa-v1"
	metaFinalReferencia       = "vec-final-referencia"
	metaMIME                  = "vec-mime"
	metaExpiraEn              = "vec-expira-en"
	metaPreparacionSHA256     = "vec-preparacion-sha256"
)

func (a *Almacen) PrepararCargaDirecta(
	ctx context.Context,
	solicitud ports.SolicitudPrepararCargaDirecta,
) (ports.InstruccionesCargaDirecta, error) {
	if err := contextoValido(ctx); err != nil {
		return ports.InstruccionesCargaDirecta{}, err
	}
	if solicitud.Validar() != nil {
		return ports.InstruccionesCargaDirecta{}, ports.ErrSolicitudAlmacenInvalida
	}
	if !a.capacidades.CargaDirectaTemporal || solicitud.Tamano > a.configuracion.TamanoMaximo {
		return ports.InstruccionesCargaDirecta{}, ports.ErrCapacidadAlmacenNoDisponible
	}
	ahora := a.reloj.Ahora().UTC()
	if solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenPrepararCargaDirecta, ahora) != nil ||
		!solicitud.ExpiraEn.After(ahora) || solicitud.ExpiraEn.Sub(ahora) > a.configuracion.DuracionCargaDirecta {
		return ports.InstruccionesCargaDirecta{}, ports.ErrSolicitudAlmacenInvalida
	}
	finalRef := a.referenciaIdempotente("carga-directa", solicitud.ClaveIdempotencia)
	sesionRef, err := a.nuevaSesionRef(finalRef)
	if err != nil {
		return ports.InstruccionesCargaDirecta{}, ErrOperacionS3
	}
	evidenciaRef, err := a.referenciaAleatoria("evidencia-carga-directa")
	if err != nil {
		return ports.InstruccionesCargaDirecta{}, ErrOperacionS3
	}
	huellaPreparacion := huellaPreparacionCargaDirecta(solicitud)
	huellaOperacion := huellaOperacionCargaDirecta(solicitud.Contexto)
	if err := a.reservarIdempotencia(ctx, finalRef, huellaPreparacion); err != nil {
		return ports.InstruccionesCargaDirecta{}, err
	}
	metadatos := map[string]string{
		metaEsquema: esquemaCargaDirecta, metaConector: a.configuracion.ConectorID,
		metaZona: string(ports.ZonaAlmacenCuarentena), metaTamano: strconv.FormatInt(solicitud.Tamano, 10),
		metaSHA256: solicitud.HuellaSHA256, metaEvidencia: evidenciaRef,
		metaAlmacenadoEn: ahora.Format(time.RFC3339Nano), metaIdempotencia: huellaPreparacion,
		metaVinculoSesion: huellaOperacion, metaFinalReferencia: finalRef, metaMIME: solicitud.MIME,
		metaExpiraEn: solicitud.ExpiraEn.Format(time.RFC3339Nano), metaPreparacionSHA256: huellaPreparacion,
	}
	entrada := &awss3.PutObjectInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(claveCargaDirecta(sesionRef)),
		ContentLength: awsv2.Int64(solicitud.Tamano), ContentType: awsv2.String(solicitud.MIME),
		ChecksumAlgorithm: types.ChecksumAlgorithmSha256, ChecksumSHA256: awsv2.String(sha256Base64(solicitud.HuellaSHA256)),
		IfNoneMatch: awsv2.String("*"), Metadata: metadatos, ServerSideEncryption: a.configuracion.Cifrado,
	}
	if a.configuracion.Cifrado == types.ServerSideEncryptionAwsKms {
		entrada.SSEKMSKeyId = awsv2.String(a.configuracion.ClaveKMS)
		entrada.BucketKeyEnabled = awsv2.Bool(a.configuracion.UsarBucketKeyKMS)
	}
	firmada, err := a.presignador.PresignPutObject(ctx, entrada, func(opciones *awss3.PresignOptions) {
		opciones.Expires = solicitud.ExpiraEn.Sub(ahora)
	})
	if err != nil || firmada == nil || firmada.Method != http.MethodPut {
		return ports.InstruccionesCargaDirecta{}, ErrOperacionS3
	}
	if err := a.validarDestinoFirmadoCargaDirecta(firmada, entrada, solicitud); err != nil {
		return ports.InstruccionesCargaDirecta{}, err
	}
	cabeceras, err := cabecerasCargaDirecta(firmada.SignedHeader)
	if err != nil {
		return ports.InstruccionesCargaDirecta{}, err
	}
	instrucciones, err := ports.NuevasInstruccionesCargaDirectaParaSolicitud(
		solicitud, a.configuracion.ConectorID, sesionRef, ports.MetodoCargaDirectaPUT,
		firmada.URL, cabeceras, ahora,
	)
	if err != nil || instrucciones.ValidarContra(a.capacidades) != nil {
		return ports.InstruccionesCargaDirecta{}, ports.ErrInstruccionesCargaDirectaNoValidas
	}
	return instrucciones, nil
}

func (a *Almacen) ConfirmarCargaDirecta(
	ctx context.Context,
	solicitud ports.SolicitudConfirmarCargaDirecta,
) (ports.ResultadoOperacionObjeto, error) {
	if err := contextoValido(ctx); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	contexto, sesionRef, intencionRef, _, _, _, consumidoEn, expiraRecibo, validaHasta, err := solicitud.RevelarParaConector()
	if err != nil || !sesionRefValida(sesionRef) {
		return ports.ResultadoOperacionObjeto{}, ports.ErrSolicitudAlmacenInvalida
	}
	ahora := a.reloj.Ahora().UTC()
	if contexto.ValidarParaEn(ports.AccionAlmacenConfirmarCargaDirecta, ahora) != nil ||
		ahora.Before(consumidoEn) || !ahora.Before(expiraRecibo) || !ahora.Before(validaHasta) {
		return ports.ResultadoOperacionObjeto{}, ports.ErrSesionCargaDirectaNoValida
	}
	finalRef, err := referenciaFinalDeSesion(sesionRef)
	if err != nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrSesionCargaDirectaNoValida
	}
	evidenciaReintentoRef, err := a.referenciaAleatoria("evidencia-confirmacion-reintento")
	if err != nil {
		return ports.ResultadoOperacionObjeto{}, ErrOperacionS3
	}
	var existentePrevio ports.ObjetoAlmacenado
	if existente, errorExistente := a.cargarObjetoActual(ctx, finalRef, ports.ZonaAlmacenCuarentena); errorExistente == nil {
		metadatos, errorMetadatos := a.metadatosObjeto(ctx, existente)
		if errorMetadatos == nil && metadatos.VinculoSesionSHA256 == huellaSesionCargaDirecta(sesionRef) {
			if errorReserva := a.cotejarReservaIdempotencia(ctx, finalRef, metadatos.IdempotenciaSHA256); errorReserva != nil {
				return ports.ResultadoOperacionObjeto{}, errorReserva
			}
			if errorLimpieza := a.limpiarCargaTemporalRestante(
				ctx, sesionRef, finalRef, metadatos.IdempotenciaSHA256, contexto,
			); errorLimpieza != nil {
				return ports.ResultadoOperacionObjeto{}, errorLimpieza
			}
			return a.resultadoConfirmacionIdempotente(contexto, existente, intencionRef, evidenciaReintentoRef, ahora)
		}
		existentePrevio = existente
	} else if !errors.Is(errorExistente, ports.ErrObjetoAlmacenNoEncontrado) {
		return ports.ResultadoOperacionObjeto{}, errorExistente
	}
	carga, versionCarga, err := a.inspeccionarCargaDirecta(ctx, sesionRef, contexto, ahora)
	if err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if carga.FinalReferencia != finalRef {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	if err := a.cotejarReservaIdempotencia(ctx, finalRef, carga.PreparacionSHA256); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	if existente := existentePrevio; existente.Validar() == nil {
		metadatos, errorMetadatos := a.metadatosObjeto(ctx, existente)
		if errorMetadatos != nil || metadatos.IdempotenciaSHA256 != carga.PreparacionSHA256 ||
			existente.Tamano != carga.Tamano || existente.MIME != carga.MIME || existente.HuellaSHA256 != carga.SHA256 {
			return ports.ResultadoOperacionObjeto{}, ports.ErrIdempotenciaAlmacenReutilizada
		}
		if err := a.eliminarVersionCargaDirecta(ctx, sesionRef, versionCarga); err != nil {
			return ports.ResultadoOperacionObjeto{}, err
		}
		return a.resultadoConfirmacionIdempotente(contexto, existente, intencionRef, evidenciaReintentoRef, ahora)
	}
	respuesta, err := a.cliente.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(claveCargaDirecta(sesionRef)),
		VersionId: awsv2.String(versionCarga), ChecksumMode: types.ChecksumModeEnabled,
	})
	if err != nil {
		return ports.ResultadoOperacionObjeto{}, errorRemoto(ctx, err)
	}
	if respuesta == nil || respuesta.Body == nil || awsv2.ToString(respuesta.VersionId) != versionCarga ||
		respuesta.ContentLength == nil || awsv2.ToInt64(respuesta.ContentLength) != carga.Tamano ||
		awsv2.ToString(respuesta.ContentType) != carga.MIME ||
		awsv2.ToString(respuesta.ChecksumSHA256) != sha256Base64(carga.SHA256) ||
		!a.cifradoRespuestaValido(respuesta.ServerSideEncryption, awsv2.ToString(respuesta.SSEKMSKeyId), awsv2.ToBool(respuesta.BucketKeyEnabled)) {
		if respuesta != nil && respuesta.Body != nil {
			_ = respuesta.Body.Close()
		}
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	defer respuesta.Body.Close()
	metadatos := metadatosObjeto{
		Referencia: carga.FinalReferencia, Zona: ports.ZonaAlmacenCuarentena, MIME: carga.MIME,
		Tamano: carga.Tamano, SHA256: carga.SHA256, Evidencia: carga.Evidencia,
		AlmacenadoEn: ahora, IdempotenciaSHA256: carga.PreparacionSHA256,
		VinculoSesionSHA256: huellaSesionCargaDirecta(sesionRef),
	}
	objeto, err := a.poner(ctx, ports.ZonaAlmacenCuarentena, metadatos, respuesta.Body, true)
	if err != nil {
		if esPrecondicion(err) {
			existente, errorExistente := a.cargarObjetoActual(ctx, carga.FinalReferencia, ports.ZonaAlmacenCuarentena)
			if errorExistente == nil {
				metadatosExistentes, errorMetadatos := a.metadatosObjeto(ctx, existente)
				if errorMetadatos == nil && metadatosExistentes.IdempotenciaSHA256 == carga.PreparacionSHA256 &&
					existente.Tamano == carga.Tamano && existente.MIME == carga.MIME && existente.HuellaSHA256 == carga.SHA256 {
					if errorLimpieza := a.eliminarVersionCargaDirecta(ctx, sesionRef, versionCarga); errorLimpieza != nil {
						return ports.ResultadoOperacionObjeto{}, errorLimpieza
					}
					resultado := ports.ResultadoOperacionObjeto{Objeto: existente, Evidencia: a.nuevaEvidencia(
						contexto, existente.Objeto, evidenciaReintentoRef, intencionRef, true, ahora,
					)}
					if resultado.Validar() == nil {
						return resultado, nil
					}
				}
			}
			return ports.ResultadoOperacionObjeto{}, ports.ErrIdempotenciaAlmacenReutilizada
		}
		return ports.ResultadoOperacionObjeto{}, errorRemoto(ctx, err)
	}
	resultado := ports.ResultadoOperacionObjeto{Objeto: objeto, Evidencia: a.nuevaEvidencia(
		contexto, objeto.Objeto, carga.Evidencia, intencionRef, false, ahora,
	)}
	if resultado.Validar() != nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	// La carga temporal no tiene retencion ni hold. La confirmacion solo se
	// declara terminada tras borrar y comprobar la ausencia de su version exacta.
	if err := a.eliminarVersionCargaDirecta(ctx, sesionRef, versionCarga); err != nil {
		return ports.ResultadoOperacionObjeto{}, err
	}
	return resultado, nil
}

func (a *Almacen) AbandonarCargaDirecta(
	ctx context.Context,
	contexto ports.ContextoOperacionAlmacen,
	sesionRef string,
) error {
	if err := contextoValido(ctx); err != nil {
		return err
	}
	ahora := a.reloj.Ahora().UTC()
	if !sesionRefValida(sesionRef) || contexto.ValidarParaEn(ports.AccionAlmacenAbandonarCargaDirecta, ahora) != nil {
		return ports.ErrSesionCargaDirectaNoValida
	}
	respuesta, err := a.cliente.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(claveCargaDirecta(sesionRef)),
		ChecksumMode: types.ChecksumModeEnabled,
	})
	if err != nil {
		if esNoEncontrado(err) {
			return nil
		}
		return errorRemoto(ctx, err)
	}
	if respuesta == nil || !versionPropiaValida(awsv2.ToString(respuesta.VersionId)) ||
		respuesta.Metadata[metaEsquema] != esquemaCargaDirecta ||
		respuesta.Metadata[metaConector] != a.configuracion.ConectorID ||
		respuesta.Metadata[metaVinculoSesion] != huellaOperacionCargaDirecta(contexto) ||
		respuesta.ObjectLockLegalHoldStatus == types.ObjectLockLegalHoldStatusOn || respuesta.ObjectLockRetainUntilDate != nil {
		return ports.ErrSesionCargaDirectaNoValida
	}
	return a.eliminarVersionCargaDirecta(ctx, sesionRef, awsv2.ToString(respuesta.VersionId))
}

func (a *Almacen) limpiarCargaTemporalRestante(
	ctx context.Context,
	sesionRef, finalRef, huellaPreparacion string,
	contexto ports.ContextoOperacionAlmacen,
) error {
	respuesta, err := a.cliente.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(claveCargaDirecta(sesionRef)),
		ChecksumMode: types.ChecksumModeEnabled,
	})
	if err != nil {
		if esNoEncontrado(err) {
			return nil
		}
		return errorRemoto(ctx, err)
	}
	if respuesta == nil || !versionPropiaValida(awsv2.ToString(respuesta.VersionId)) ||
		respuesta.Metadata[metaEsquema] != esquemaCargaDirecta ||
		respuesta.Metadata[metaConector] != a.configuracion.ConectorID ||
		respuesta.Metadata[metaFinalReferencia] != finalRef ||
		respuesta.Metadata[metaPreparacionSHA256] != huellaPreparacion ||
		respuesta.Metadata[metaVinculoSesion] != huellaOperacionCargaDirecta(contexto) ||
		respuesta.ObjectLockLegalHoldStatus == types.ObjectLockLegalHoldStatusOn || respuesta.ObjectLockRetainUntilDate != nil {
		return ports.ErrIntegridadObjetoAlmacen
	}
	return a.eliminarVersionCargaDirecta(ctx, sesionRef, awsv2.ToString(respuesta.VersionId))
}

func (a *Almacen) eliminarVersionCargaDirecta(ctx context.Context, sesionRef, version string) error {
	if !sesionRefValida(sesionRef) || !versionPropiaValida(version) {
		return ports.ErrSesionCargaDirectaNoValida
	}
	base := context.WithoutCancel(ctx)
	limpieza, cancelar := context.WithTimeout(base, 10*time.Second)
	defer cancelar()
	_, err := a.cliente.DeleteObject(limpieza, &awss3.DeleteObjectInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(claveCargaDirecta(sesionRef)),
		VersionId: awsv2.String(version),
	})
	if err != nil && !esNoEncontrado(err) {
		return errorRemoto(limpieza, err)
	}
	_, err = a.cliente.HeadObject(limpieza, &awss3.HeadObjectInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(claveCargaDirecta(sesionRef)),
		VersionId: awsv2.String(version), ChecksumMode: types.ChecksumModeEnabled,
	})
	if err == nil {
		return ports.ErrIntegridadObjetoAlmacen
	}
	if !esNoEncontrado(err) {
		return errorRemoto(limpieza, err)
	}
	return nil
}

type cargaDirectaInspeccionada struct {
	FinalReferencia, MIME, SHA256, Evidencia, PreparacionSHA256, VinculoOperacionSHA256 string
	Tamano                                                                              int64
	ExpiraEn                                                                            time.Time
}

func (a *Almacen) inspeccionarCargaDirecta(
	ctx context.Context,
	sesionRef string,
	contexto ports.ContextoOperacionAlmacen,
	ahora time.Time,
) (cargaDirectaInspeccionada, string, error) {
	respuesta, err := a.cliente.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: awsv2.String(a.configuracion.BucketCuarentena), Key: awsv2.String(claveCargaDirecta(sesionRef)),
		ChecksumMode: types.ChecksumModeEnabled,
	})
	if err != nil {
		return cargaDirectaInspeccionada{}, "", errorRemoto(ctx, err)
	}
	if respuesta == nil || respuesta.ContentLength == nil || respuesta.ContentType == nil ||
		!versionPropiaValida(awsv2.ToString(respuesta.VersionId)) ||
		!a.cifradoRespuestaValido(respuesta.ServerSideEncryption, awsv2.ToString(respuesta.SSEKMSKeyId), awsv2.ToBool(respuesta.BucketKeyEnabled)) ||
		respuesta.ObjectLockLegalHoldStatus == types.ObjectLockLegalHoldStatusOn || respuesta.ObjectLockRetainUntilDate != nil {
		return cargaDirectaInspeccionada{}, "", ports.ErrIntegridadObjetoAlmacen
	}
	tamano, errorTamano := strconv.ParseInt(respuesta.Metadata[metaTamano], 10, 64)
	expira, errorExpira := time.Parse(time.RFC3339Nano, respuesta.Metadata[metaExpiraEn])
	carga := cargaDirectaInspeccionada{
		FinalReferencia: respuesta.Metadata[metaFinalReferencia], MIME: respuesta.Metadata[metaMIME],
		SHA256: respuesta.Metadata[metaSHA256], Evidencia: respuesta.Metadata[metaEvidencia],
		PreparacionSHA256:      respuesta.Metadata[metaPreparacionSHA256],
		VinculoOperacionSHA256: respuesta.Metadata[metaVinculoSesion], Tamano: tamano, ExpiraEn: expira,
	}
	if errorTamano != nil || errorExpira != nil || respuesta.Metadata[metaEsquema] != esquemaCargaDirecta ||
		respuesta.Metadata[metaConector] != a.configuracion.ConectorID ||
		respuesta.Metadata[metaZona] != string(ports.ZonaAlmacenCuarentena) ||
		!referenciaPropiaValida(carga.FinalReferencia) || !textoTecnicoValido(carga.MIME, 255) ||
		carga.MIME != awsv2.ToString(respuesta.ContentType) || carga.Tamano != awsv2.ToInt64(respuesta.ContentLength) ||
		carga.Tamano < 1 || carga.Tamano > a.configuracion.TamanoMaximo || !sha256Valido(carga.SHA256) ||
		sha256Base64(carga.SHA256) != awsv2.ToString(respuesta.ChecksumSHA256) ||
		!textoTecnicoValido(carga.Evidencia, 512) || !sha256Valido(carga.PreparacionSHA256) ||
		carga.PreparacionSHA256 != respuesta.Metadata[metaIdempotencia] ||
		carga.VinculoOperacionSHA256 != huellaOperacionCargaDirecta(contexto) ||
		!carga.ExpiraEn.After(ahora) || carga.ExpiraEn.Location() != time.UTC {
		return cargaDirectaInspeccionada{}, "", ports.ErrIntegridadObjetoAlmacen
	}
	return carga, awsv2.ToString(respuesta.VersionId), nil
}

func cabecerasCargaDirecta(cabeceras http.Header) ([]ports.CabeceraCargaDirecta, error) {
	nombres := make([]string, 0, len(cabeceras))
	for nombre := range cabeceras {
		nombres = append(nombres, strings.ToLower(strings.TrimSpace(nombre)))
	}
	sort.Strings(nombres)
	resultado := make([]ports.CabeceraCargaDirecta, 0, len(nombres))
	vistos := make(map[string]struct{}, len(nombres))
	for _, nombre := range nombres {
		if _, repetida := vistos[nombre]; repetida {
			return nil, ports.ErrInstruccionesCargaDirectaNoValidas
		}
		vistos[nombre] = struct{}{}
		if nombre == "host" || nombre == "content-length" {
			continue
		}
		valores := cabeceras.Values(nombre)
		if len(valores) != 1 {
			return nil, ports.ErrInstruccionesCargaDirectaNoValidas
		}
		resultado = append(resultado, ports.CabeceraCargaDirecta{Nombre: nombre, Valor: valores[0]})
	}
	requeridas := []string{
		"content-type", "if-none-match", "x-amz-meta-" + metaEsquema,
		"x-amz-meta-" + metaConector, "x-amz-meta-" + metaTamano, "x-amz-meta-" + metaSHA256,
		"x-amz-meta-" + metaEvidencia, "x-amz-meta-" + metaFinalReferencia,
		"x-amz-meta-" + metaExpiraEn, "x-amz-meta-" + metaPreparacionSHA256,
		"x-amz-server-side-encryption",
	}
	for _, requerida := range requeridas {
		encontrada := false
		for _, cabecera := range resultado {
			if cabecera.Nombre == requerida {
				encontrada = true
				break
			}
		}
		if !encontrada {
			return nil, ports.ErrInstruccionesCargaDirectaNoValidas
		}
	}
	return resultado, nil
}

func (a *Almacen) validarDestinoFirmadoCargaDirecta(
	firmada *v4.PresignedHTTPRequest,
	entrada *awss3.PutObjectInput,
	solicitud ports.SolicitudPrepararCargaDirecta,
) error {
	if firmada == nil || entrada == nil {
		return ports.ErrInstruccionesCargaDirectaNoValidas
	}
	destino, err := url.Parse(firmada.URL)
	if err != nil || destino.Scheme != "https" || destino.User != nil || destino.Fragment != "" ||
		strings.ToLower(destino.Scheme+"://"+destino.Host) != a.configuracion.origenCargaDirecta() {
		return ports.ErrInstruccionesCargaDirectaNoValidas
	}
	ruta, err := url.PathUnescape(destino.EscapedPath())
	sufijo := "/" + awsv2.ToString(entrada.Key)
	if a.configuracion.PathStyle {
		sufijo = "/" + a.configuracion.BucketCuarentena + sufijo
	}
	if err != nil || !strings.HasSuffix(ruta, sufijo) || strings.Contains(ruta, "..") || strings.Contains(ruta, "\\") {
		return ports.ErrInstruccionesCargaDirectaNoValidas
	}
	consulta := destino.Query()
	expiracionSegundos, err := strconv.ParseInt(valorConsultaUnico(consulta, "X-Amz-Expires"), 10, 64)
	if err != nil || expiracionSegundos < 1 || time.Duration(expiracionSegundos)*time.Second > solicitud.ExpiraEn.Sub(a.reloj.Ahora().UTC())+time.Second ||
		valorConsultaUnico(consulta, "X-Amz-Checksum-Sha256") != sha256Base64(solicitud.HuellaSHA256) ||
		valorConsultaUnico(consulta, "X-Amz-Sdk-Checksum-Algorithm") != "SHA256" ||
		valorConsultaUnico(consulta, "X-Amz-Algorithm") != "AWS4-HMAC-SHA256" ||
		valorConsultaUnico(consulta, "X-Amz-Signature") == "" {
		return ports.ErrInstruccionesCargaDirectaNoValidas
	}
	requeridas := map[string]string{
		"content-length": strconv.FormatInt(solicitud.Tamano, 10), "content-type": solicitud.MIME,
		"if-none-match": "*", "x-amz-server-side-encryption": string(a.configuracion.Cifrado),
	}
	for clave, valor := range entrada.Metadata {
		requeridas["x-amz-meta-"+strings.ToLower(clave)] = valor
	}
	if a.configuracion.Cifrado == types.ServerSideEncryptionAwsKms {
		requeridas["x-amz-server-side-encryption-aws-kms-key-id"] = a.configuracion.ClaveKMS
		if a.configuracion.UsarBucketKeyKMS {
			requeridas["x-amz-server-side-encryption-bucket-key-enabled"] = "true"
		}
	}
	firmadas := ";" + strings.ToLower(valorConsultaUnico(consulta, "X-Amz-SignedHeaders")) + ";"
	for nombre, valor := range requeridas {
		if firmada.SignedHeader.Get(nombre) != valor || !strings.Contains(firmadas, ";"+nombre+";") {
			return ports.ErrInstruccionesCargaDirectaNoValidas
		}
	}
	return nil
}

func valorConsultaUnico(valores url.Values, clave string) string {
	obtenidos := valores[clave]
	if len(obtenidos) != 1 {
		return ""
	}
	return obtenidos[0]
}

func (a *Almacen) nuevaSesionRef(finalRef string) (string, error) {
	referencia, err := a.referenciaAleatoriaObjeto()
	if err != nil {
		return "", err
	}
	if !referenciaPropiaValida(finalRef) {
		return "", ports.ErrSolicitudAlmacenInvalida
	}
	return prefijoSesionCargaDirecta + strings.TrimPrefix(referencia, prefijoReferencia) + "-" +
		strings.TrimPrefix(finalRef, prefijoReferencia), nil
}

func referenciaFinalDeSesion(sesionRef string) (string, error) {
	if !sesionRefValida(sesionRef) {
		return "", ports.ErrSesionCargaDirectaNoValida
	}
	partes := strings.Split(strings.TrimPrefix(sesionRef, prefijoSesionCargaDirecta), "-")
	referencia := prefijoReferencia + partes[1]
	if !referenciaPropiaValida(referencia) {
		return "", ports.ErrSesionCargaDirectaNoValida
	}
	return referencia, nil
}

func huellaSesionCargaDirecta(sesionRef string) string {
	if !sesionRefValida(sesionRef) {
		return ""
	}
	return sumaComponentes("vec-s3-sesion-carga-directa-v1", sesionRef)
}

func (a *Almacen) resultadoConfirmacionIdempotente(
	contexto ports.ContextoOperacionAlmacen,
	objeto ports.ObjetoAlmacenado,
	intencionRef, evidenciaRef string,
	ahora time.Time,
) (ports.ResultadoOperacionObjeto, error) {
	resultado := ports.ResultadoOperacionObjeto{Objeto: objeto, Evidencia: a.nuevaEvidencia(
		contexto, objeto.Objeto, evidenciaRef, intencionRef, true, ahora,
	)}
	if resultado.Validar() != nil {
		return ports.ResultadoOperacionObjeto{}, ports.ErrIntegridadObjetoAlmacen
	}
	return resultado, nil
}

func sesionRefValida(valor string) bool {
	if !strings.HasPrefix(valor, prefijoSesionCargaDirecta) || len(valor) != len(prefijoSesionCargaDirecta)+64+1+64 ||
		valor != strings.ToLower(valor) || strings.ContainsAny(valor, "/\\") || strings.Contains(valor, "..") {
		return false
	}
	partes := strings.Split(strings.TrimPrefix(valor, prefijoSesionCargaDirecta), "-")
	if len(partes) != 2 {
		return false
	}
	for _, parte := range partes {
		decodificado, err := hex.DecodeString(parte)
		if err != nil || len(decodificado) != 32 {
			return false
		}
	}
	return true
}

func claveCargaDirecta(sesionRef string) string { return prefijoClaveCargaDirecta + sesionRef }

func huellaPreparacionCargaDirecta(s ports.SolicitudPrepararCargaDirecta) string {
	componentes := append(componentesContexto(s.Contexto), s.ClaveIdempotencia, s.MIME,
		strconv.FormatInt(s.Tamano, 10), s.HuellaSHA256, s.ExpiraEn.UTC().Format(time.RFC3339Nano))
	return sumaComponentes(componentes...)
}

func huellaOperacionCargaDirecta(contexto ports.ContextoOperacionAlmacen) string {
	p, err := contexto.Proyeccion()
	if err != nil {
		return ""
	}
	return sumaComponentes(p.OperacionRef, p.CorrelacionRef, p.Finalidad, p.Clasificacion,
		p.CargaRef, p.SujetoSeudonimoHMAC, p.RecursoRef, p.ModuloID, p.HuellaSolicitudHMAC)
}

var _ ports.GestorCargaDirecta = (*Almacen)(nil)
var _ io.Reader = strings.NewReader("")
