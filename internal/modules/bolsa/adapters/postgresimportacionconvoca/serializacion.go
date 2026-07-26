package postgresimportacionconvoca

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"

	aplicacion "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

type incidenciaPostgreSQL struct {
	Fila   int    `json:"fila"`
	Campo  string `json:"campo"`
	Codigo string `json:"codigo"`
}

type procedenciaPostgreSQL struct {
	Esquema                      string `json:"esquema"`
	Fuente                       string `json:"fuente"`
	Autoridad                    string `json:"autoridad"`
	HabilitaActosConEfectos      bool   `json:"habilita_actos_con_efectos"`
	RequiereConfirmacionRegistro bool   `json:"requiere_confirmacion_registro"`
	UsoPuntosAutobaremacion      string `json:"uso_puntos_autobaremacion"`
}

type actaPostgreSQL struct {
	ActaRef              string                 `json:"acta_ref"`
	ImportacionRef       string                 `json:"importacion_ref"`
	HuellaFicheroSHA256  string                 `json:"huella_fichero_sha256"`
	FicheroCustodiadoRef string                 `json:"fichero_custodiado_ref"`
	NombreFichero        string                 `json:"nombre_fichero"`
	ActorRef             string                 `json:"actor_ref"`
	RegistradaEn         string                 `json:"registrada_en"`
	Esquema              string                 `json:"esquema"`
	FilasLeidas          int                    `json:"filas_leidas"`
	FilasAceptadas       int                    `json:"filas_aceptadas"`
	FilasRechazadas      int                    `json:"filas_rechazadas"`
	Incidencias          []incidenciaPostgreSQL `json:"incidencias"`
	Procedencia          procedenciaPostgreSQL  `json:"procedencia"`
}

type filaProtegidaPostgreSQL struct {
	Numero                        int    `json:"numero"`
	EsquemaProteccion             string `json:"esquema_proteccion"`
	ClaveRef                      string `json:"clave_ref"`
	ClaveDerivacionRef            string `json:"clave_derivacion_ref"`
	ClaveAtestacionRef            string `json:"clave_atestacion_ref"`
	NonceHex                      string `json:"nonce_hex"`
	ContenidoCifradoHex           string `json:"contenido_cifrado_hex"`
	HuellaContenidoCifradoSHA256  string `json:"huella_contenido_cifrado_sha256"`
	DerivacionDocumentoHMACSHA256 string `json:"derivacion_documento_hmac_sha256"`
	AtestacionFilaHMACSHA256      string `json:"atestacion_fila_hmac_sha256"`
}

type estadoPostgreSQL struct {
	Acta                     actaPostgreSQL `json:"acta"`
	EstadoConciliacion       string         `json:"estado_conciliacion"`
	EstadoStaging            string         `json:"estado_staging"`
	PoliticaRetencionRef     string         `json:"politica_retencion_ref"`
	PoliticaRetencionVersion uint64         `json:"politica_retencion_version"`
	ConservarStagingHasta    string         `json:"conservar_staging_hasta"`
	BloqueoRetencion         bool           `json:"bloqueo_retencion"`
	Version                  uint64         `json:"version"`
}

type loteRecuperadoPostgreSQL struct {
	Estado          estadoPostgreSQL          `json:"estado"`
	Filas           []filaProtegidaPostgreSQL `json:"filas"`
	SiguienteNumero *int                      `json:"siguiente_numero"`
}

func serializarActa(acta dominio.ActaImportacion) ([]byte, error) {
	if acta.Validar() != nil {
		return nil, ErrLoteNoConfiable
	}
	incidencias := make([]incidenciaPostgreSQL, len(acta.Incidencias))
	for i := range acta.Incidencias {
		incidencias[i] = incidenciaPostgreSQL(acta.Incidencias[i])
	}
	p := acta.Procedencia
	contenido, err := json.Marshal(actaPostgreSQL{
		ActaRef: acta.ActaRef, ImportacionRef: acta.ImportacionRef,
		HuellaFicheroSHA256:  acta.HuellaFicheroSHA256,
		FicheroCustodiadoRef: acta.FicheroCustodiadoRef,
		NombreFichero:        acta.NombreFichero, ActorRef: acta.ActorRef,
		RegistradaEn: formatearInstante(acta.RegistradaEn), Esquema: string(acta.Esquema),
		FilasLeidas: acta.FilasLeidas, FilasAceptadas: acta.FilasAceptadas,
		FilasRechazadas: acta.FilasRechazadas, Incidencias: incidencias,
		Procedencia: procedenciaPostgreSQL{
			Esquema: p.Esquema, Fuente: p.Fuente, Autoridad: p.Autoridad,
			HabilitaActosConEfectos:      p.HabilitaActosConEfectos,
			RequiereConfirmacionRegistro: p.RequiereConfirmacionRegistro,
			UsoPuntosAutobaremacion:      p.UsoPuntosAutobaremacion,
		},
	})
	if err != nil || len(contenido) > maximoBytesJSONActa {
		return nil, ErrLoteNoConfiable
	}
	return contenido, nil
}

func restaurarActa(datos actaPostgreSQL) (dominio.ActaImportacion, error) {
	instante, err := time.Parse("2006-01-02T15:04:05.000000Z", datos.RegistradaEn)
	if err != nil {
		return dominio.ActaImportacion{}, ErrResultadoNoConfiable
	}
	incidencias := make([]dominio.Incidencia, len(datos.Incidencias))
	for i := range datos.Incidencias {
		incidencias[i] = dominio.Incidencia(datos.Incidencias[i])
	}
	acta := dominio.ActaImportacion{
		ActaRef: datos.ActaRef, ImportacionRef: datos.ImportacionRef,
		HuellaFicheroSHA256:  datos.HuellaFicheroSHA256,
		FicheroCustodiadoRef: datos.FicheroCustodiadoRef,
		NombreFichero:        datos.NombreFichero, ActorRef: datos.ActorRef,
		RegistradaEn: instante.UTC(), Esquema: dominio.EsquemaExportacion(datos.Esquema),
		FilasLeidas: datos.FilasLeidas, FilasAceptadas: datos.FilasAceptadas,
		FilasRechazadas: datos.FilasRechazadas, Incidencias: incidencias,
		Procedencia: dominio.Procedencia{
			Esquema: datos.Procedencia.Esquema, Fuente: datos.Procedencia.Fuente,
			Autoridad:                    datos.Procedencia.Autoridad,
			HabilitaActosConEfectos:      datos.Procedencia.HabilitaActosConEfectos,
			RequiereConfirmacionRegistro: datos.Procedencia.RequiereConfirmacionRegistro,
			UsoPuntosAutobaremacion:      datos.Procedencia.UsoPuntosAutobaremacion,
		},
	}
	if acta.Validar() != nil {
		return dominio.ActaImportacion{}, ErrResultadoNoConfiable
	}
	return acta, nil
}

func deserializarActa(contenido []byte) (dominio.ActaImportacion, error) {
	var datos actaPostgreSQL
	if err := decodificarJSONExacto(contenido, &datos); err != nil {
		return dominio.ActaImportacion{}, ErrResultadoNoConfiable
	}
	return restaurarActa(datos)
}

func serializarFilasProtegidas(filas []FilaStagingProtegida) ([]byte, error) {
	var contenido bytes.Buffer
	contenido.WriteByte('[')
	for i := range filas {
		huella := sha256.Sum256(filas[i].ContenidoCifrado)
		dato := filaProtegidaPostgreSQL{
			Numero: filas[i].Numero, EsquemaProteccion: filas[i].EsquemaProteccion,
			ClaveRef:                     filas[i].ClaveRef,
			ClaveDerivacionRef:           filas[i].ClaveDerivacionRef,
			ClaveAtestacionRef:           filas[i].ClaveAtestacionRef,
			NonceHex:                     hex.EncodeToString(filas[i].Nonce),
			ContenidoCifradoHex:          hex.EncodeToString(filas[i].ContenidoCifrado),
			HuellaContenidoCifradoSHA256: hex.EncodeToString(huella[:]),
			DerivacionDocumentoHMACSHA256: hex.EncodeToString(
				filas[i].DerivacionDocumentoHMACSHA256,
			),
			AtestacionFilaHMACSHA256: hex.EncodeToString(
				filas[i].AtestacionFilaHMACSHA256,
			),
		}
		filaJSON, err := json.Marshal(dato)
		if err != nil || len(filaJSON)+contenido.Len()+2 > maximoBytesJSONFilas {
			borrarBytes(filaJSON, contenido.Bytes())
			return nil, ErrMaterialNoConfiable
		}
		if i > 0 {
			contenido.WriteByte(',')
		}
		contenido.Write(filaJSON)
		borrarBytes(filaJSON)
	}
	contenido.WriteByte(']')
	return contenido.Bytes(), nil
}

func restaurarFilasProtegidas(datos []filaProtegidaPostgreSQL) ([]FilaStagingProtegida, error) {
	filas := make([]FilaStagingProtegida, len(datos))
	for i := range datos {
		nonce, errNonce := hex.DecodeString(datos[i].NonceHex)
		cifrado, errCifrado := hex.DecodeString(datos[i].ContenidoCifradoHex)
		derivacion, errDerivacion := hex.DecodeString(datos[i].DerivacionDocumentoHMACSHA256)
		atestacion, errAtestacion := hex.DecodeString(datos[i].AtestacionFilaHMACSHA256)
		huellaEsperada, errHuella := hex.DecodeString(datos[i].HuellaContenidoCifradoSHA256)
		huella := sha256.Sum256(cifrado)
		if errNonce != nil || errCifrado != nil || errDerivacion != nil ||
			errAtestacion != nil || errHuella != nil ||
			len(huellaEsperada) != sha256.Size ||
			!bytesIgualesConstantes(huella[:], huellaEsperada) {
			borrarBytes(nonce, cifrado, derivacion, atestacion, huellaEsperada)
			borrarFilasProtegidas(filas[:i])
			return nil, ErrResultadoNoConfiable
		}
		borrarBytes(huellaEsperada)
		filas[i] = FilaStagingProtegida{
			Numero: datos[i].Numero, EsquemaProteccion: datos[i].EsquemaProteccion,
			ClaveRef:           datos[i].ClaveRef,
			ClaveDerivacionRef: datos[i].ClaveDerivacionRef,
			ClaveAtestacionRef: datos[i].ClaveAtestacionRef,
			Nonce:              nonce, ContenidoCifrado: cifrado,
			DerivacionDocumentoHMACSHA256: derivacion,
			AtestacionFilaHMACSHA256:      atestacion,
		}
	}
	return filas, nil
}

func restaurarEstado(datos estadoPostgreSQL) (aplicacion.EstadoImportacion, error) {
	acta, err := restaurarActa(datos.Acta)
	if err != nil {
		return aplicacion.EstadoImportacion{}, err
	}
	conservarHasta, err := time.Parse("2006-01-02T15:04:05.000000Z", datos.ConservarStagingHasta)
	if err != nil {
		return aplicacion.EstadoImportacion{}, ErrResultadoNoConfiable
	}
	estado := aplicacion.EstadoImportacion{
		Acta: acta, EstadoConciliacion: aplicacion.EstadoConciliacion(datos.EstadoConciliacion),
		EstadoStaging:            aplicacion.EstadoStaging(datos.EstadoStaging),
		PoliticaRetencionRef:     datos.PoliticaRetencionRef,
		PoliticaRetencionVersion: datos.PoliticaRetencionVersion,
		ConservarStagingHasta:    conservarHasta.UTC(),
		BloqueoRetencion:         datos.BloqueoRetencion, Version: datos.Version,
	}
	if estado.Validar() != nil {
		return aplicacion.EstadoImportacion{}, ErrResultadoNoConfiable
	}
	return estado, nil
}

func decodificarJSONExacto(contenido []byte, destino any) error {
	if len(contenido) < 2 {
		return errors.New("JSON vacío")
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return err
	}
	var resto any
	if err := decodificador.Decode(&resto); !errors.Is(err, io.EOF) {
		return errors.New("JSON con contenido adicional")
	}
	return nil
}
