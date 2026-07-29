package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// salidaCierreConsultaRRHH conserva únicamente los escalares probatorios
// devueltos por CT-000045. No contiene la capacidad, el cursor de entrada ni
// identificadores de personas.
type salidaCierreConsultaRRHH struct {
	esquema                      string
	accesoRef                    string
	secuencia                    int64
	anteriorSHA256               string
	huellaSHA256                 string
	vinculoIdentidadHuellaSHA256 string
	alcanceHuellaSHA256          string
	registradaEn                 time.Time
	auditoriaRef                 string
	auditoriaHuellaSHA256        string
	consumoHuellaSHA256          string
	contenidoHuellaSHA256        string
	resultadoHuellaSHA256        string
	cursorHuellaSHA256           string
	generadaEn                   time.Time
	expedienteRef                string
	versionExpediente            int64
	total                        int16
	reciboSelloSHA256            string
}

type salidaCuadroConsultaRRHH struct {
	contenidoCanonico []byte
	cursorSiguiente   string
	cierre            salidaCierreConsultaRRHH
}

type salidaDetalleConsultaRRHH struct {
	contenidoCanonico []byte
	cierre            salidaCierreConsultaRRHH
}

// analizadorCanonConsultaRRHH es el límite privado con el analizador del
// canon binario CT-000042. El analizador no recibe ni fabrica autoridad.
type analizadorCanonConsultaRRHH interface {
	analizarCuadro(
		[]byte,
		string,
		time.Time,
		uint16,
	) (ports.PaginaCuadroRRHH, error)
	analizarDetalle(
		[]byte,
		time.Time,
		string,
		uint64,
	) (ports.EntradaDetalleExpedienteRRHHMinimizada, error)
}

type analizadorCanonConsultaRRHHPostgreSQL struct{}

func (analizadorCanonConsultaRRHHPostgreSQL) analizarCuadro(
	contenido []byte,
	cursor string,
	generadaEn time.Time,
	total uint16,
) (ports.PaginaCuadroRRHH, error) {
	decodificado, err := decodificarContenidoCuadroRRHHPostgreSQL(contenido)
	if err != nil ||
		!decodificado.paginaSinRecibo.GeneradaEn.Equal(generadaEn) ||
		len(decodificado.paginaSinRecibo.Expedientes) != int(total) {
		return ports.PaginaCuadroRRHH{},
			ports.ErrResultadoConsultaRRHHNoConfiable
	}
	pagina := decodificado.paginaSinRecibo
	if pagina.HayMas {
		materialCursor, err := base64.RawURLEncoding.Strict().DecodeString(cursor)
		if err != nil {
			return ports.PaginaCuadroRRHH{},
				ports.ErrResultadoConsultaRRHHNoConfiable
		}
		defer clear(materialCursor)
		huellaCursor := sha256.Sum256(materialCursor)
		if len(materialCursor) != sha256.Size ||
			base64.RawURLEncoding.EncodeToString(materialCursor) != cursor ||
			!bytes.Equal(huellaCursor[:], decodificado.cursorHuella[:]) {
			return ports.PaginaCuadroRRHH{},
				ports.ErrResultadoConsultaRRHHNoConfiable
		}
		pagina.CursorSiguiente = cursor
	} else if cursor != "" ||
		decodificado.cursorHuella != ([sha256.Size]byte{}) {
		return ports.PaginaCuadroRRHH{},
			ports.ErrResultadoConsultaRRHHNoConfiable
	}
	return pagina, nil
}

func (analizadorCanonConsultaRRHHPostgreSQL) analizarDetalle(
	contenido []byte,
	generadaEn time.Time,
	expedienteRef string,
	version uint64,
) (ports.EntradaDetalleExpedienteRRHHMinimizada, error) {
	decodificado, err := decodificarContenidoDetalleRRHHPostgreSQL(contenido)
	if err != nil || decodificado.expedienteRef != expedienteRef ||
		decodificado.version != version {
		return ports.EntradaDetalleExpedienteRRHHMinimizada{},
			ports.ErrResultadoConsultaRRHHNoConfiable
	}
	reconstruido, err :=
		decodificado.entrada.ExportarContenidoCanonicoParaSQL(generadaEn)
	if err != nil || !bytes.Equal(reconstruido.BytesCanonicos(), contenido) {
		return ports.EntradaDetalleExpedienteRRHHMinimizada{},
			ports.ErrResultadoConsultaRRHHNoConfiable
	}
	return decodificado.entrada, nil
}

func (s salidaCierreConsultaRRHH) construirRecibo(
	ordenContexto ports.ContextoConsultaRRHH,
	capacidad ports.CapacidadConsultaRRHH,
) (ports.ReciboLecturaRRHH, error) {
	secuencia, version, total, err := s.enterosSeguros()
	if err != nil {
		return ports.ReciboLecturaRRHH{},
			ports.ErrResultadoConsultaRRHHNoConfiable
	}
	registro, err := ports.NuevoResultadoRegistradorAccesoRRHHV2(
		s.esquema,
		s.accesoRef,
		secuencia,
		s.anteriorSHA256,
		s.huellaSHA256,
		s.vinculoIdentidadHuellaSHA256,
		s.alcanceHuellaSHA256,
		s.registradaEn,
	)
	if err != nil {
		return ports.ReciboLecturaRRHH{},
			ports.ErrResultadoConsultaRRHHNoConfiable
	}
	evidencia, err := ports.NuevaEvidenciaConsumoResultadoRRHHV2(
		s.auditoriaRef,
		s.auditoriaHuellaSHA256,
		s.consumoHuellaSHA256,
		s.contenidoHuellaSHA256,
		s.resultadoHuellaSHA256,
		s.cursorHuellaSHA256,
		s.generadaEn,
		s.expedienteRef,
		version,
		total,
	)
	if err != nil {
		return ports.ReciboLecturaRRHH{},
			ports.ErrResultadoConsultaRRHHNoConfiable
	}
	return ports.NuevoReciboLecturaRRHHV2(
		ordenContexto,
		capacidad,
		registro,
		evidencia,
		s.reciboSelloSHA256,
	)
}

func (s salidaCierreConsultaRRHH) enterosSeguros() (
	uint64,
	uint64,
	uint16,
	error,
) {
	const maximoEnteroJSONSeguro = int64(9_007_199_254_740_991)
	if s.secuencia < 1 || s.secuencia > maximoEnteroJSONSeguro ||
		s.versionExpediente < 0 ||
		s.versionExpediente > maximoEnteroJSONSeguro ||
		s.total < 0 || s.total > ports.LimiteMaximoCuadroRRHH {
		return 0, 0, 0, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	return uint64(s.secuencia), uint64(s.versionExpediente),
		uint16(s.total), nil
}
