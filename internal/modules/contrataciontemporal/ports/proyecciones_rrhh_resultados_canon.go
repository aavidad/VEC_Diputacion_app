package ports

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	VersionCanonContenidoResultadoRRHH uint16 = 1
	VersionCanonResultadoConsultaRRHH  uint16 = 1

	DominioCanonContenidoCuadroRRHH   = "vec.contratacion_temporal.resultado_rrhh.contenido_cuadro.v1"
	DominioCanonResultadoConsultaRRHH = "vec.contratacion_temporal.resultado_rrhh.v1"

	tipoResultadoCuadroRRHH = "cuadro"

	cabeceraCanonContenidoCuadroRRHH   = "VEC-CT-CONTENIDO-CUADRO-RRHH-V1\n"
	cabeceraCanonResultadoConsultaRRHH = "VEC-CT-RESULTADO-CONSULTA-RRHH-V1\n"

	// LimiteMaximoCanonResultadoRRHH coincide con el contrato estricto de
	// PostgreSQL. El constructor comprueba cada encuadre antes de anexarlo:
	// nunca reserva una representación canónica por encima de 256 KiB.
	LimiteMaximoCanonResultadoRRHH = 256 * 1024

	formatoInstanteCanonicoRRHH = "2006-01-02T15:04:05.000000Z"
)

// ExportacionCanonicaContenidoCuadroRRHH conserva el contenido ya reducido de
// una página antes de que exista su recibo de lectura. Es material, no
// autoridad. El cursor en claro nunca forma parte de esta exportación.
type ExportacionCanonicaContenidoCuadroRRHH struct {
	exportacionCanonicaRRHH
	generadaEn   time.Time
	total        uint16
	cursorHuella [sha256.Size]byte
	tieneCursor  bool
}

// ExportacionCanonicaResultadoConsultaRRHH reproduce el canon que PostgreSQL
// usa para ligar tipo, instante, cardinalidad y huellas del resultado. Sus
// accesores exponen únicamente argumentos no secretos del contrato tipado.
type ExportacionCanonicaResultadoConsultaRRHH struct {
	exportacionCanonicaRRHH
	tipoConsulta          string
	generadaEn            time.Time
	total                 uint16
	contenidoHuellaSHA256 string
	cursorHuellaSHA256    string
}

// ExportarContenidoCanonicoParaSQL valida la estructura independiente de
// autorización de una página aún sin recibo y crea su canon binario.
//
// Formato V1, siempre en este orden:
//   - cabecera ASCII terminada en LF;
//   - instante UTC con seis decimales y número de expedientes;
//   - por cada resumen, sus quince campos completos en el orden del DTO;
//   - indicador hay_mas (0/1);
//   - SHA-256 binario del cursor o un encuadre vacío.
//
// Cada valor usa «longitud UTF-8 decimal:valor LF». La longitud no tiene ceros
// iniciales. Los números se representan en decimal canónico. Los digestos
// binarios usan el mismo encuadre, pero conservan sus 32 bytes sin pasarlos a
// texto. No se usa encoding/json ni depende del orden de campos de un codec.
func (p PaginaCuadroRRHH) ExportarContenidoCanonicoParaSQL() (
	ExportacionCanonicaContenidoCuadroRRHH,
	error,
) {
	if validarContenidoCuadroRRHHSinRecibo(p) != nil {
		return ExportacionCanonicaContenidoCuadroRRHH{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	constructor := nuevoConstructorCanonResultadoRRHH(
		cabeceraCanonContenidoCuadroRRHH,
	)
	constructor.instante(p.GeneradaEn)
	constructor.enteroSinSigno(uint64(len(p.Expedientes)))
	for _, resumen := range p.Expedientes {
		constructor.resumen(resumen)
	}
	constructor.booleano(p.HayMas)

	var cursorHuella [sha256.Size]byte
	if p.HayMas {
		cursorHuella = sha256.Sum256([]byte(p.CursorSiguiente))
		if huellaBinariaNulaRRHH(cursorHuella) {
			return ExportacionCanonicaContenidoCuadroRRHH{},
				ErrResultadoConsultaRRHHNoConfiable
		}
		constructor.bytes(cursorHuella[:])
	} else {
		constructor.bytes(nil)
	}
	canon, err := constructor.finalizar()
	if err != nil {
		return ExportacionCanonicaContenidoCuadroRRHH{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	exportacion, err := nuevaExportacionCanonicaResultadoRRHH(
		DominioCanonContenidoCuadroRRHH,
		VersionCanonContenidoResultadoRRHH,
		canon,
	)
	if err != nil {
		return ExportacionCanonicaContenidoCuadroRRHH{}, err
	}
	return ExportacionCanonicaContenidoCuadroRRHH{
		exportacionCanonicaRRHH: exportacion,
		generadaEn:              p.GeneradaEn,
		total:                   uint16(len(p.Expedientes)),
		cursorHuella:            cursorHuella,
		tieneCursor:             p.HayMas,
	}, nil
}

// ExportarResultadoCanonicoParaSQL crea la evidencia encuadrada que debe
// coincidir byte a byte con canon_resultado_consulta_rrhh_v1. Deriva todos sus
// campos del contenido nominal y no permite al adaptador sustituirlos.
func (e ExportacionCanonicaContenidoCuadroRRHH) ExportarResultadoCanonicoParaSQL() (
	ExportacionCanonicaResultadoConsultaRRHH,
	error,
) {
	if !e.valida() {
		return ExportacionCanonicaResultadoConsultaRRHH{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	cursorHuella := ""
	if e.tieneCursor {
		cursorHuella = hex.EncodeToString(e.cursorHuella[:])
	}
	constructor := nuevoConstructorCanonResultadoRRHH(
		cabeceraCanonResultadoConsultaRRHH,
	)
	constructor.texto(tipoResultadoCuadroRRHH)
	constructor.instante(e.generadaEn)
	constructor.enteroSinSigno(uint64(e.total))
	constructor.texto(e.HuellaSHA256())
	constructor.texto(cursorHuella)
	canon, err := constructor.finalizar()
	if err != nil {
		return ExportacionCanonicaResultadoConsultaRRHH{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	exportacion, err := nuevaExportacionCanonicaResultadoRRHH(
		DominioCanonResultadoConsultaRRHH,
		VersionCanonResultadoConsultaRRHH,
		canon,
	)
	if err != nil {
		return ExportacionCanonicaResultadoConsultaRRHH{}, err
	}
	return ExportacionCanonicaResultadoConsultaRRHH{
		exportacionCanonicaRRHH: exportacion,
		tipoConsulta:            tipoResultadoCuadroRRHH,
		generadaEn:              e.generadaEn,
		total:                   e.total,
		contenidoHuellaSHA256:   e.HuellaSHA256(),
		cursorHuellaSHA256:      cursorHuella,
	}, nil
}

func (e ExportacionCanonicaContenidoCuadroRRHH) valida() bool {
	if !exportacionCanonicaResultadoRRHHValida(
		e.exportacionCanonicaRRHH,
		DominioCanonContenidoCuadroRRHH,
		VersionCanonContenidoResultadoRRHH,
	) || !domain.InstanteUTCCanonico(e.generadaEn) ||
		e.total > LimiteMaximoCuadroRRHH ||
		(e.tieneCursor && (e.total == 0 || huellaBinariaNulaRRHH(e.cursorHuella))) ||
		(!e.tieneCursor && !huellaBinariaNulaRRHH(e.cursorHuella)) {
		return false
	}
	return true
}

func (e ExportacionCanonicaResultadoConsultaRRHH) valida() bool {
	esCuadro := e.tipoConsulta == tipoResultadoCuadroRRHH &&
		e.total <= LimiteMaximoCuadroRRHH
	cursorValido := e.cursorHuellaSHA256 == "" ||
		e.total > 0 && huellaSHA256CanonicaRRHH(e.cursorHuellaSHA256)
	return esCuadro &&
		exportacionCanonicaResultadoRRHHValida(
			e.exportacionCanonicaRRHH,
			DominioCanonResultadoConsultaRRHH,
			VersionCanonResultadoConsultaRRHH,
		) &&
		domain.InstanteUTCCanonico(e.generadaEn) &&
		huellaSHA256CanonicaRRHH(e.contenidoHuellaSHA256) &&
		cursorValido
}

func (e ExportacionCanonicaResultadoConsultaRRHH) TipoConsulta() string {
	return e.tipoConsulta
}

func (e ExportacionCanonicaResultadoConsultaRRHH) GeneradaEn() time.Time {
	return e.generadaEn
}

func (e ExportacionCanonicaResultadoConsultaRRHH) Total() uint16 {
	return e.total
}

func (e ExportacionCanonicaResultadoConsultaRRHH) ContenidoHuellaSHA256() string {
	return e.contenidoHuellaSHA256
}

func (e ExportacionCanonicaResultadoConsultaRRHH) CursorHuellaSHA256() string {
	return e.cursorHuellaSHA256
}

func validarContenidoCuadroRRHHSinRecibo(p PaginaCuadroRRHH) error {
	if p.Lectura != (ReciboLecturaRRHH{}) ||
		!domain.InstanteUTCCanonico(p.GeneradaEn) ||
		len(p.Expedientes) > int(LimiteMaximoCuadroRRHH) ||
		(p.HayMas && (len(p.Expedientes) == 0 ||
			!cursorRRHHValido(p.CursorSiguiente))) ||
		(!p.HayMas && p.CursorSiguiente != "") {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	vistos := make(map[string]struct{}, len(p.Expedientes))
	for indice, resumen := range p.Expedientes {
		if resumen.Validar() != nil ||
			resumen.ActualizadoEn.After(p.GeneradaEn) {
			return ErrResultadoConsultaRRHHNoConfiable
		}
		if _, repetido := vistos[resumen.ExpedienteRef]; repetido {
			return ErrResultadoConsultaRRHHNoConfiable
		}
		vistos[resumen.ExpedienteRef] = struct{}{}
		if indice == 0 {
			continue
		}
		anterior := p.Expedientes[indice-1]
		if anterior.ActualizadoEn.Before(resumen.ActualizadoEn) ||
			(anterior.ActualizadoEn.Equal(resumen.ActualizadoEn) &&
				anterior.ExpedienteRef <= resumen.ExpedienteRef) {
			return ErrResultadoConsultaRRHHNoConfiable
		}
	}
	return nil
}

func nuevaExportacionCanonicaResultadoRRHH(
	dominio string,
	version uint16,
	canon []byte,
) (exportacionCanonicaRRHH, error) {
	if len(canon) == 0 || len(canon) > LimiteMaximoCanonResultadoRRHH {
		return exportacionCanonicaRRHH{}, ErrResultadoConsultaRRHHNoConfiable
	}
	suma := sha256.Sum256(canon)
	if huellaBinariaNulaRRHH(suma) {
		return exportacionCanonicaRRHH{}, ErrResultadoConsultaRRHHNoConfiable
	}
	return exportacionCanonicaRRHH{
		dominio: dominio, version: version,
		bytesCanonicos: append([]byte(nil), canon...),
		huellaSHA256:   hex.EncodeToString(suma[:]),
	}, nil
}

func exportacionCanonicaResultadoRRHHValida(
	e exportacionCanonicaRRHH,
	dominio string,
	version uint16,
) bool {
	if e.dominio != dominio || e.version != version ||
		len(e.bytesCanonicos) == 0 ||
		len(e.bytesCanonicos) > LimiteMaximoCanonResultadoRRHH ||
		!huellaSHA256CanonicaRRHH(e.huellaSHA256) {
		return false
	}
	suma := sha256.Sum256(e.bytesCanonicos)
	esperada, err := hex.DecodeString(e.huellaSHA256)
	return err == nil &&
		subtle.ConstantTimeCompare(suma[:], esperada) == 1
}

func huellaSHA256CanonicaRRHH(valor string) bool {
	return len(valor) == sha256.Size*2 &&
		valor == strings.ToLower(valor) &&
		valor != strings.Repeat("0", sha256.Size*2) &&
		func() bool {
			decodificada, err := hex.DecodeString(valor)
			return err == nil && len(decodificada) == sha256.Size
		}()
}

func huellaBinariaNulaRRHH(valor [sha256.Size]byte) bool {
	var cero [sha256.Size]byte
	return subtle.ConstantTimeCompare(valor[:], cero[:]) == 1
}

type constructorCanonResultadoRRHH struct {
	bytesCanonicos []byte
	err            error
}

func nuevoConstructorCanonResultadoRRHH(
	cabecera string,
) *constructorCanonResultadoRRHH {
	constructor := &constructorCanonResultadoRRHH{}
	constructor.crudo([]byte(cabecera))
	return constructor
}

func (c *constructorCanonResultadoRRHH) resumen(r ResumenExpedienteRRHH) {
	c.texto(r.ExpedienteRef)
	c.texto(r.OrganizacionRef)
	c.texto(r.NumeroVisible)
	c.enteroSinSigno(r.Version)
	c.texto(r.FlujoRef)
	c.enteroSinSigno(r.FlujoVersion)
	c.texto(r.FlujoHuella)
	c.texto(string(r.FaseClave))
	c.texto(string(r.EstadoClave))
	c.texto(r.CentroRef)
	c.texto(r.CategoriaRef)
	c.texto(string(r.ModalidadClave))
	c.texto(r.UnidadRef)
	c.instante(r.CreadoEn)
	c.instante(r.ActualizadoEn)
}

func (c *constructorCanonResultadoRRHH) instante(valor time.Time) {
	c.texto(valor.Format(formatoInstanteCanonicoRRHH))
}

func (c *constructorCanonResultadoRRHH) enteroSinSigno(valor uint64) {
	c.texto(strconv.FormatUint(valor, 10))
}

func (c *constructorCanonResultadoRRHH) booleano(valor bool) {
	if valor {
		c.texto("1")
		return
	}
	c.texto("0")
}

func (c *constructorCanonResultadoRRHH) texto(valor string) {
	if c == nil || c.err != nil {
		return
	}
	longitud := strconv.FormatInt(int64(len(valor)), 10)
	adicional := len(longitud) + 1 + len(valor) + 1
	if adicional > LimiteMaximoCanonResultadoRRHH-len(c.bytesCanonicos) {
		c.err = ErrResultadoConsultaRRHHNoConfiable
		return
	}
	c.bytesCanonicos = append(c.bytesCanonicos, longitud...)
	c.bytesCanonicos = append(c.bytesCanonicos, ':')
	c.bytesCanonicos = append(c.bytesCanonicos, valor...)
	c.bytesCanonicos = append(c.bytesCanonicos, '\n')
}

func (c *constructorCanonResultadoRRHH) bytes(valor []byte) {
	if c == nil || c.err != nil {
		return
	}
	longitud := strconv.FormatInt(int64(len(valor)), 10)
	adicional := len(longitud) + 1 + len(valor) + 1
	if adicional > LimiteMaximoCanonResultadoRRHH-len(c.bytesCanonicos) {
		c.err = ErrResultadoConsultaRRHHNoConfiable
		return
	}
	c.bytesCanonicos = append(c.bytesCanonicos, longitud...)
	c.bytesCanonicos = append(c.bytesCanonicos, ':')
	c.bytesCanonicos = append(c.bytesCanonicos, valor...)
	c.bytesCanonicos = append(c.bytesCanonicos, '\n')
}

func (c *constructorCanonResultadoRRHH) crudo(valor []byte) {
	if c == nil || c.err != nil {
		return
	}
	if len(valor) > LimiteMaximoCanonResultadoRRHH-len(c.bytesCanonicos) {
		c.err = ErrResultadoConsultaRRHHNoConfiable
		return
	}
	c.bytesCanonicos = append(c.bytesCanonicos, valor...)
}

func (c *constructorCanonResultadoRRHH) finalizar() ([]byte, error) {
	if c == nil || c.err != nil || len(c.bytesCanonicos) == 0 ||
		len(c.bytesCanonicos) > LimiteMaximoCanonResultadoRRHH {
		return nil, ErrResultadoConsultaRRHHNoConfiable
	}
	resultado := c.bytesCanonicos
	c.bytesCanonicos = nil
	return resultado, nil
}
