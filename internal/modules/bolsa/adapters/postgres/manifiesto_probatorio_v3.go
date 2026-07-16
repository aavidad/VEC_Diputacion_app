package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const esquemaArchivoProbatorioPostgreSQLV3 = "vec.bolsa.archivo_probatorio-postgresql.v3"

type autorizacionManifiestoPostgreSQLV3 struct {
	Secuencia       uint32 `json:"secuencia"`
	Accion          string `json:"accion"`
	ClaseRecurso    string `json:"clase_recurso"`
	RecursoRef      string `json:"recurso_ref"`
	AutorizacionRef string `json:"autorizacion_ref"`
}

type evidenciaManifiestoPostgreSQLV3 struct {
	Secuencia             uint32 `json:"secuencia"`
	Tipo                  string `json:"tipo"`
	Referencia            string `json:"referencia"`
	HuellaEvidenciaSHA256 string `json:"huella_evidencia_sha256"`
}

// manifiestoProbatorioPostgreSQLV3 es el unico documento JSON admitido por el
// corte 000005. No se serializa el dominio directamente: los nombres y tipos
// snake_case quedan congelados y el decodificador rechaza campos desconocidos.
type manifiestoProbatorioPostgreSQLV3 struct {
	Esquema                   string                               `json:"esquema"`
	Finalidad                 string                               `json:"finalidad"`
	VersionEsquema            int                                  `json:"version_esquema"`
	Referencia                string                               `json:"referencia"`
	ProcesoRef                string                               `json:"proceso_ref"`
	SolicitudRef              string                               `json:"solicitud_ref"`
	SujetoRef                 string                               `json:"sujeto_ref"`
	BaremacionMeritoRef       string                               `json:"baremacion_merito_ref"`
	DecisionRef               string                               `json:"decision_ref"`
	VersionBase               uint64                               `json:"version_base"`
	HuellaVersionBaseSHA256   string                               `json:"huella_version_base_sha256"`
	Autorizaciones            []autorizacionManifiestoPostgreSQLV3 `json:"autorizaciones"`
	Evidencias                []evidenciaManifiestoPostgreSQLV3    `json:"evidencias"`
	CreadoEn                  string                               `json:"creado_en"`
	HuellaManifiestoSHA256    string                               `json:"huella_manifiesto_sha256"`
	SelloManifiestoHMACSHA256 string                               `json:"sello_manifiesto_hmac_sha256"`
}

type archivoProbatorioPostgreSQLV3 struct {
	Esquema             string                            `json:"esquema"`
	BaremacionMeritoRef string                            `json:"baremacion_merito_ref"`
	NumeroVersion       string                            `json:"numero_version"`
	HuellaArchivoSHA256 string                            `json:"huella_archivo_sha256"`
	Manifiestos         []manifiestoArchivadoPostgreSQLV3 `json:"manifiestos"`
}

type manifiestoArchivadoPostgreSQLV3 struct {
	Manifiesto                          manifiestoProbatorioPostgreSQLV3 `json:"manifiesto"`
	ContenidoManifiestoCanonicoHex      string                           `json:"contenido_manifiesto_canonico_hex"`
	RepresentacionManifiestoCanonicaHex string                           `json:"representacion_manifiesto_canonica_hex"`
	PreimagenHMACManifiestoHex          string                           `json:"preimagen_hmac_manifiesto_hex"`
}

type manifiestoArchivadoDecodificadoV3 struct {
	Manifiesto     puertosbolsa.ManifiestoProbatorioBaremacion
	Contenido      []byte
	Representacion []byte
	PreimagenHMAC  []byte
}

type archivoProbatorioDecodificadoV3 struct {
	BaremacionMeritoRef string
	NumeroVersion       uint64
	Manifiestos         []manifiestoArchivadoDecodificadoV3
}

func manifiestoPostgreSQLDesdeDominio(
	manifiesto puertosbolsa.ManifiestoProbatorioBaremacion,
) (manifiestoProbatorioPostgreSQLV3, error) {
	if manifiesto.Validar() != nil {
		return manifiestoProbatorioPostgreSQLV3{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	autorizaciones := make([]autorizacionManifiestoPostgreSQLV3, len(manifiesto.Autorizaciones))
	for indice, autorizacion := range manifiesto.Autorizaciones {
		autorizaciones[indice] = autorizacionManifiestoPostgreSQLV3{
			Secuencia: autorizacion.Secuencia, Accion: string(autorizacion.Accion),
			ClaseRecurso: string(autorizacion.ClaseRecurso), RecursoRef: autorizacion.RecursoRef,
			AutorizacionRef: autorizacion.AutorizacionRef,
		}
	}
	evidencias := make([]evidenciaManifiestoPostgreSQLV3, len(manifiesto.Evidencias))
	for indice, evidencia := range manifiesto.Evidencias {
		evidencias[indice] = evidenciaManifiestoPostgreSQLV3{
			Secuencia: evidencia.Secuencia, Tipo: string(evidencia.Tipo),
			Referencia: evidencia.Referencia, HuellaEvidenciaSHA256: evidencia.HuellaEvidenciaSHA256,
		}
	}
	return manifiestoProbatorioPostgreSQLV3{
		Esquema: manifiesto.Esquema, Finalidad: manifiesto.Finalidad,
		VersionEsquema: manifiesto.VersionEsquema, Referencia: manifiesto.Referencia,
		ProcesoRef: manifiesto.ProcesoRef, SolicitudRef: manifiesto.SolicitudRef,
		SujetoRef: manifiesto.SujetoRef, BaremacionMeritoRef: manifiesto.BaremacionMeritoRef,
		DecisionRef: manifiesto.DecisionRef, VersionBase: manifiesto.VersionBase,
		HuellaVersionBaseSHA256: manifiesto.HuellaVersionBaseSHA256,
		Autorizaciones:          autorizaciones, Evidencias: evidencias,
		CreadoEn:                  manifiesto.CreadoEn.UTC().Format(time.RFC3339Nano),
		HuellaManifiestoSHA256:    manifiesto.HuellaManifiestoSHA256,
		SelloManifiestoHMACSHA256: manifiesto.SelloManifiestoHMACSHA256,
	}, nil
}

func (m manifiestoProbatorioPostgreSQLV3) dominio() (puertosbolsa.ManifiestoProbatorioBaremacion, error) {
	creadoEn, err := time.Parse(time.RFC3339Nano, m.CreadoEn)
	if err != nil || creadoEn.Location() != time.UTC || creadoEn.Format(time.RFC3339Nano) != m.CreadoEn ||
		m.Autorizaciones == nil || m.Evidencias == nil {
		return puertosbolsa.ManifiestoProbatorioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	autorizaciones := make([]puertosbolsa.AutorizacionProbatoriaBaremacion, len(m.Autorizaciones))
	for indice, autorizacion := range m.Autorizaciones {
		autorizaciones[indice] = puertosbolsa.AutorizacionProbatoriaBaremacion{
			Secuencia:    autorizacion.Secuencia,
			Accion:       puertosbolsa.AccionOperacionBaremacion(autorizacion.Accion),
			ClaseRecurso: puertosbolsa.ClaseRecursoOperacionBaremacion(autorizacion.ClaseRecurso),
			RecursoRef:   autorizacion.RecursoRef, AutorizacionRef: autorizacion.AutorizacionRef,
		}
	}
	evidencias := make([]puertosbolsa.EvidenciaProbatoriaBaremacion, len(m.Evidencias))
	for indice, evidencia := range m.Evidencias {
		evidencias[indice] = puertosbolsa.EvidenciaProbatoriaBaremacion{
			Secuencia:  evidencia.Secuencia,
			Tipo:       puertosbolsa.TipoEvidenciaProbatoriaBaremacion(evidencia.Tipo),
			Referencia: evidencia.Referencia, HuellaEvidenciaSHA256: evidencia.HuellaEvidenciaSHA256,
		}
	}
	resultado := puertosbolsa.ManifiestoProbatorioBaremacion{
		Esquema: m.Esquema, Finalidad: m.Finalidad, VersionEsquema: m.VersionEsquema,
		Referencia: m.Referencia, ProcesoRef: m.ProcesoRef, SolicitudRef: m.SolicitudRef,
		SujetoRef: m.SujetoRef, BaremacionMeritoRef: m.BaremacionMeritoRef,
		DecisionRef: m.DecisionRef, VersionBase: m.VersionBase,
		HuellaVersionBaseSHA256: m.HuellaVersionBaseSHA256,
		Autorizaciones:          autorizaciones, Evidencias: evidencias, CreadoEn: creadoEn,
		HuellaManifiestoSHA256:    m.HuellaManifiestoSHA256,
		SelloManifiestoHMACSHA256: m.SelloManifiestoHMACSHA256,
	}
	if resultado.Validar() != nil {
		return puertosbolsa.ManifiestoProbatorioBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	return resultado, nil
}

func serializarManifiestoProbatorioV3(
	manifiesto *puertosbolsa.ManifiestoProbatorioBaremacion,
) (documento, contenido, representacion, preimagen []byte, err error) {
	if manifiesto == nil {
		return nil, nil, nil, nil, nil
	}
	fila, err := manifiestoPostgreSQLDesdeDominio(manifiesto.Clonar())
	if err != nil {
		return nil, nil, nil, nil, err
	}
	documento, err = json.Marshal(fila)
	if err != nil {
		return nil, nil, nil, nil, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	artefactos, err := puertosbolsa.ArtefactosCanonicosManifiestoProbatorioBaremacion(*manifiesto)
	if err != nil {
		borrarBytesPostgreSQL(documento)
		return nil, nil, nil, nil, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	return documento, artefactos.ContenidoSinHuella.Revelar(),
		artefactos.RepresentacionSellada.Revelar(), artefactos.PreimagenHMAC.Revelar(), nil
}

func (r *RepositorioBaremaciones) verificarSelloManifiestoProbatorioV3(
	ctx context.Context,
	manifiesto *puertosbolsa.ManifiestoProbatorioBaremacion,
) error {
	if manifiesto == nil {
		return nil
	}
	if r == nil || valorNulo(r.verificador) {
		return puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible
	}
	representacion, err := puertosbolsa.RepresentacionCanonicaManifiestoProbatorioBaremacion(
		manifiesto.Clonar(),
	)
	if err != nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	peticion := puertosbolsa.SolicitudVerificarSelloBaremacion{
		Finalidad:              puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3,
		RepresentacionCanonica: representacion,
		SelloHMAC:              manifiesto.SelloManifiestoHMACSHA256,
	}
	if peticion.Validar() != nil {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	if err = r.verificador.VerificarSelloBaremacion(ctx, peticion); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(err, puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible) {
			return puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible
		}
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	return validarContexto(ctx)
}

func decodificarArchivoProbatorioV3(contenido []byte) (archivoProbatorioDecodificadoV3, error) {
	var archivo archivoProbatorioPostgreSQLV3
	if len(contenido) == 0 || len(contenido) > puertosbolsa.MaximoTamanoCargaProtegida ||
		decodificarJSONEstricto(contenido, &archivo) != nil ||
		archivo.Esquema != esquemaArchivoProbatorioPostgreSQLV3 || archivo.Manifiestos == nil ||
		len(archivo.Manifiestos) > 4096 {
		return archivoProbatorioDecodificadoV3{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	numero, err := strconv.ParseUint(archivo.NumeroVersion, 10, 64)
	if err != nil || numero < 1 || strconv.FormatUint(numero, 10) != archivo.NumeroVersion ||
		archivo.BaremacionMeritoRef == "" ||
		uint64(len(archivo.Manifiestos))+1 != numero {
		return archivoProbatorioDecodificadoV3{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	manifiestos := make([]manifiestoArchivadoDecodificadoV3, len(archivo.Manifiestos))
	decodificacionCompleta := false
	defer func() {
		if !decodificacionCompleta {
			borrarArtefactosArchivoV3(manifiestos)
		}
	}()
	for indice, fila := range archivo.Manifiestos {
		manifiesto, err := fila.Manifiesto.dominio()
		if err != nil {
			return archivoProbatorioDecodificadoV3{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		contenidoCanonico, err := decodificarHexArchivoV3(fila.ContenidoManifiestoCanonicoHex)
		if err != nil {
			return archivoProbatorioDecodificadoV3{}, err
		}
		representacion, err := decodificarHexArchivoV3(fila.RepresentacionManifiestoCanonicaHex)
		if err != nil {
			borrarBytesPostgreSQL(contenidoCanonico)
			return archivoProbatorioDecodificadoV3{}, err
		}
		preimagen, err := decodificarHexArchivoV3(fila.PreimagenHMACManifiestoHex)
		if err != nil {
			borrarBytesPostgreSQL(contenidoCanonico, representacion)
			return archivoProbatorioDecodificadoV3{}, err
		}
		manifiestos[indice] = manifiestoArchivadoDecodificadoV3{
			Manifiesto: manifiesto, Contenido: contenidoCanonico,
			Representacion: representacion, PreimagenHMAC: preimagen,
		}
	}
	if huellaArchivoProbatorioV3(
		archivo.Esquema, archivo.BaremacionMeritoRef, archivo.NumeroVersion, manifiestos,
	) != archivo.HuellaArchivoSHA256 {
		return archivoProbatorioDecodificadoV3{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	decodificacionCompleta = true
	return archivoProbatorioDecodificadoV3{
		BaremacionMeritoRef: archivo.BaremacionMeritoRef,
		NumeroVersion:       numero, Manifiestos: manifiestos,
	}, nil
}

func huellaArchivoProbatorioV3(
	esquema, baremacionRef, numero string,
	manifiestos []manifiestoArchivadoDecodificadoV3,
) string {
	var material bytes.Buffer
	escribirTexto := func(valor string) {
		var longitud [8]byte
		binary.BigEndian.PutUint64(longitud[:], uint64(len(valor)))
		_, _ = material.Write(longitud[:])
		_, _ = material.WriteString(valor)
	}
	escribirBytes := func(valor []byte) {
		var longitud [8]byte
		binary.BigEndian.PutUint64(longitud[:], uint64(len(valor)))
		_, _ = material.Write(longitud[:])
		_, _ = material.Write(valor)
	}
	escribirTexto(esquema)
	escribirTexto(baremacionRef)
	escribirTexto(numero)
	escribirTexto(strconv.Itoa(len(manifiestos)))
	for _, archivado := range manifiestos {
		escribirBytes(archivado.Contenido)
		escribirBytes(archivado.Representacion)
		escribirBytes(archivado.PreimagenHMAC)
		escribirTexto(archivado.Manifiesto.SelloManifiestoHMACSHA256)
	}
	suma := sha256.Sum256(material.Bytes())
	borrarBytesPostgreSQL(material.Bytes())
	return hex.EncodeToString(suma[:])
}

func (r *RepositorioBaremaciones) verificarArchivoProbatorioV3(
	ctx context.Context,
	version puertosbolsa.VersionBaremacion,
	contenido []byte,
) ([]puertosbolsa.ManifiestoProbatorioBaremacion, []string, error) {
	if err := validarContexto(ctx); err != nil {
		return nil, nil, err
	}
	if r == nil || valorNulo(r.verificador) || version.Validar() != nil {
		return nil, nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	archivo, err := decodificarArchivoProbatorioV3(contenido)
	if err != nil {
		return nil, nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	defer borrarArtefactosArchivoV3(archivo.Manifiestos)
	if archivo.BaremacionMeritoRef != version.Referencia.BaremacionMeritoRef ||
		archivo.NumeroVersion != version.Referencia.Numero ||
		len(version.Agregado.Decisiones) != len(archivo.Manifiestos) {
		return nil, nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	manifiestos := make([]puertosbolsa.ManifiestoProbatorioBaremacion, len(archivo.Manifiestos))
	partesHuella := make([]string, 0, 1+6*len(archivo.Manifiestos))
	partesHuella = append(partesHuella, strconv.Itoa(len(archivo.Manifiestos)))
	for indice, archivado := range archivo.Manifiestos {
		if err = validarContexto(ctx); err != nil {
			return nil, nil, err
		}
		manifiesto := archivado.Manifiesto
		base, err := referenciaBaseManifiesto(version.Agregado, indice)
		if err != nil || manifiesto.BaremacionMeritoRef != version.Referencia.BaremacionMeritoRef ||
			manifiesto.VersionBase != base.Numero ||
			manifiesto.HuellaVersionBaseSHA256 != base.HuellaEstadoSHA256 ||
			manifiesto.ValidarCoberturaFirmaPara(
				base, version.Agregado.Decisiones[indice].Contenido,
				version.Agregado.Decisiones[indice].Firma,
			) != nil {
			return nil, nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		representacion, err := puertosbolsa.RepresentacionCanonicaManifiestoProbatorioBaremacion(manifiesto)
		if err != nil {
			return nil, nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		artefactos, err := puertosbolsa.ArtefactosCanonicosManifiestoProbatorioBaremacion(manifiesto)
		if err != nil {
			return nil, nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		contenidoReconstruido := artefactos.ContenidoSinHuella.Revelar()
		representacionReconstruida := artefactos.RepresentacionSellada.Revelar()
		preimagenReconstruida := artefactos.PreimagenHMAC.Revelar()
		coinciden := bytes.Equal(archivado.Contenido, contenidoReconstruido) &&
			bytes.Equal(archivado.Representacion, representacionReconstruida) &&
			bytes.Equal(archivado.PreimagenHMAC, preimagenReconstruida)
		borrarBytesPostgreSQL(contenidoReconstruido, representacionReconstruida, preimagenReconstruida)
		if !coinciden {
			return nil, nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		sumaContenido := sha256.Sum256(archivado.Contenido)
		sumaRepresentacion := sha256.Sum256(archivado.Representacion)
		sumaPreimagen := sha256.Sum256(archivado.PreimagenHMAC)
		partesHuella = append(partesHuella,
			manifiesto.Referencia, manifiesto.HuellaManifiestoSHA256,
			manifiesto.SelloManifiestoHMACSHA256, hex.EncodeToString(sumaContenido[:]),
			hex.EncodeToString(sumaRepresentacion[:]), hex.EncodeToString(sumaPreimagen[:]),
		)
		peticion := puertosbolsa.SolicitudVerificarSelloBaremacion{
			Finalidad:              puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3,
			RepresentacionCanonica: representacion,
			SelloHMAC:              manifiesto.SelloManifiestoHMACSHA256,
		}
		if peticion.Validar() != nil {
			return nil, nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		if err = r.verificarSelloHistoricoArchivoV3(ctx, peticion); err != nil {
			return nil, nil, err
		}
		manifiestos[indice] = manifiesto.Clonar()
	}
	return manifiestos, partesHuella, nil
}

func (r *RepositorioBaremaciones) verificarSelloHistoricoArchivoV3(
	ctx context.Context,
	peticion puertosbolsa.SolicitudVerificarSelloBaremacion,
) error {
	if err := r.verificador.VerificarSelloBaremacion(ctx, peticion); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(err, puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible) {
			return puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible
		}
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	return validarContexto(ctx)
}

func decodificarHexArchivoV3(valor string) ([]byte, error) {
	if valor == "" || len(valor) > 2*puertosbolsa.MaximoTamanoCargaProtegida || len(valor)%2 != 0 {
		return nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	resultado, err := hex.DecodeString(valor)
	if err != nil || hex.EncodeToString(resultado) != valor {
		borrarBytesPostgreSQL(resultado)
		return nil, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	return resultado, nil
}

func borrarArtefactosArchivoV3(manifiestos []manifiestoArchivadoDecodificadoV3) {
	for indice := range manifiestos {
		borrarBytesPostgreSQL(
			manifiestos[indice].Contenido,
			manifiestos[indice].Representacion,
			manifiestos[indice].PreimagenHMAC,
		)
	}
}

func referenciaBaseManifiesto(
	agregado dominiobolsa.BaremacionMerito, indice int,
) (puertosbolsa.ReferenciaVersionBaremacion, error) {
	if indice < 0 || indice >= len(agregado.Decisiones) {
		return puertosbolsa.ReferenciaVersionBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	prefijo, err := agregado.ClonarCanonica()
	if err != nil {
		return puertosbolsa.ReferenciaVersionBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	prefijo.Decisiones = append([]dominiobolsa.DecisionTecnica(nil), prefijo.Decisiones[:indice]...)
	huella, err := prefijo.HuellaEstadoSHA256()
	if err != nil {
		return puertosbolsa.ReferenciaVersionBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	referencia := puertosbolsa.ReferenciaVersionBaremacion{
		BaremacionMeritoRef: prefijo.ID, Numero: uint64(indice + 1), HuellaEstadoSHA256: huella,
	}
	if referencia.Validar() != nil {
		return puertosbolsa.ReferenciaVersionBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	return referencia, nil
}

func borrarBytesPostgreSQL(valores ...[]byte) {
	for _, valor := range valores {
		for indice := range valor {
			valor[indice] = 0
		}
	}
}

func cantidadManifiestosEsperada(numero string) (uint64, error) {
	valor, err := strconv.ParseUint(numero, 10, 64)
	if err != nil || valor < 1 || strconv.FormatUint(valor, 10) != numero {
		return 0, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	return valor - 1, nil
}
