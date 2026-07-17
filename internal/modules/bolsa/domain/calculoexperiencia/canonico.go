package calculoexperiencia

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

const (
	esquemaEntradaExperiencia        = "vec.bolsa.entrada_experiencia.v1"
	maximoBytesRepresentacionEntrada = 16 * 1024 * 1024
	maximaProfundidadJSONEntrada     = 16
	maximosCamposObjetoJSONEntrada   = 16
)

type materialEntradaExperiencia struct {
	Esquema     string             `json:"esquema"`
	Instantanea materialReferencia `json:"instantanea"`
	Tramos      materialesTramos   `json:"tramos"`
}

type materialReferencia struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type materialTramo struct {
	Referencia  materialReferencia         `json:"referencia"`
	ServicioRef string                     `json:"servicio_ref"`
	Periodo     materialPeriodo            `json:"periodo"`
	Jornada     baremacion.FraccionJornada `json:"jornada"`
	Atestacion  materialAtestacion         `json:"computo_integro_atestado"`
	Atributos   materialesAtributos        `json:"atributos"`
}

// materialesTramos y materialesAtributos limitan las colecciones durante la
// decodificacion. Comprobar len despues de json.Decode seria demasiado tarde:
// una entrada pequena con millones de objetos vacios podria amplificar memoria
// antes de que el dominio aplicase sus limites.
type materialesTramos []materialTramo

type materialesAtributos []materialAtributo

func (m *materialesTramos) UnmarshalJSON(datos []byte) error {
	if m == nil {
		return nuevoError("entrada.tramos", CodigoValorNoCanonico)
	}
	decodificador := json.NewDecoder(bytes.NewReader(datos))
	decodificador.DisallowUnknownFields()
	delimitador, err := decodificador.Token()
	if err != nil || delimitador != json.Delim('[') {
		return nuevoError("entrada.tramos", CodigoValorNoCanonico)
	}
	resultado := make(materialesTramos, 0)
	for decodificador.More() {
		if len(resultado) >= maximoTramosEntrada {
			return nuevoError("entrada.tramos", CodigoFueraDeLimites)
		}
		var tramo materialTramo
		if err := decodificador.Decode(&tramo); err != nil {
			return clasificarErrorDecodificacionEntrada(err, "entrada.tramos")
		}
		resultado = append(resultado, tramo)
	}
	if delimitador, err = decodificador.Token(); err != nil || delimitador != json.Delim(']') {
		return nuevoError("entrada.tramos", CodigoValorNoCanonico)
	}
	if err := exigirFinJSON(decodificador); err != nil {
		return err
	}
	*m = resultado
	return nil
}

func (m *materialesAtributos) UnmarshalJSON(datos []byte) error {
	if m == nil {
		return nuevoError("tramo.atributos", CodigoValorNoCanonico)
	}
	decodificador := json.NewDecoder(bytes.NewReader(datos))
	decodificador.DisallowUnknownFields()
	delimitador, err := decodificador.Token()
	if err != nil || delimitador != json.Delim('[') {
		return nuevoError("tramo.atributos", CodigoValorNoCanonico)
	}
	resultado := make(materialesAtributos, 0)
	for decodificador.More() {
		if len(resultado) >= maximoAtributosPorTramo {
			return nuevoError("tramo.atributos", CodigoFueraDeLimites)
		}
		var atributo materialAtributo
		if err := decodificador.Decode(&atributo); err != nil {
			return clasificarErrorDecodificacionEntrada(err, "tramo.atributos")
		}
		resultado = append(resultado, atributo)
	}
	if delimitador, err = decodificador.Token(); err != nil || delimitador != json.Delim(']') {
		return nuevoError("tramo.atributos", CodigoValorNoCanonico)
	}
	if err := exigirFinJSON(decodificador); err != nil {
		return err
	}
	*m = resultado
	return nil
}

func clasificarErrorDecodificacionEntrada(err error, campo string) error {
	if errors.Is(err, ErrFueraDeLimites) {
		return nuevoError(campo, CodigoFueraDeLimites)
	}
	return nuevoError(campo, CodigoValorNoCanonico)
}

func exigirFinJSON(decodificador *json.Decoder) error {
	var sobrante struct{}
	if err := decodificador.Decode(&sobrante); !errors.Is(err, io.EOF) {
		return nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	return nil
}

type materialPeriodo struct {
	Modo         ModoPeriodoServicio    `json:"modo"`
	Desde        baremacion.FechaCivil  `json:"desde"`
	FinInformado *baremacion.FechaCivil `json:"fin_informado,omitempty"`
}

type materialAtestacion struct {
	Modo       modoComputoIntegroAtestado `json:"modo"`
	Referencia *materialReferencia        `json:"referencia,omitempty"`
}

type materialAtributo struct {
	Clave    string             `json:"clave"`
	Catalogo materialReferencia `json:"catalogo"`
	Valor    string             `json:"valor"`
}

// RepresentacionCanonica devuelve el unico JSON V1 admitido para la entrada.
func (e EntradaExperiencia) RepresentacionCanonica() ([]byte, error) {
	if err := e.validar(true); err != nil {
		return nil, err
	}
	material := materialEntradaExperiencia{
		Esquema:     esquemaEntradaExperiencia,
		Instantanea: materializarReferencia(e.instantanea),
		Tramos:      make([]materialTramo, len(e.tramos)),
	}
	for indice, tramo := range e.tramos {
		material.Tramos[indice] = materializarTramo(tramo)
	}
	contenido, err := json.Marshal(material)
	if err != nil {
		return nil, nuevoError("representacion_canonica", CodigoValorInvalido)
	}
	if len(contenido) == 0 || len(contenido) > maximoBytesRepresentacionEntrada {
		return nil, nuevoError("representacion_canonica", CodigoFueraDeLimites)
	}
	return contenido, nil
}

// MarshalJSON conserva exactamente la representacion canonica.
func (e EntradaExperiencia) MarshalJSON() ([]byte, error) {
	return e.RepresentacionCanonica()
}

func (e EntradaExperiencia) HuellaSHA256() (string, error) {
	contenido, err := e.RepresentacionCanonica()
	if err != nil {
		return "", err
	}
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:]), nil
}

// RestaurarEntradaExperiencia solo acepta los mismos bytes que produciria
// RepresentacionCanonica. Rechaza campos, orden, espacios y claves duplicadas
// alternativos aunque representen aparentemente los mismos datos. Restaurar y
// comprobar la huella no autentica la fuente, pertenencia, catalogos ni
// atestaciones: antes de calcular, la aplicacion debe obtener una atestacion
// externa exacta desde el puerto confiable de servicios y evidencias.
func RestaurarEntradaExperiencia(contenido []byte) (EntradaExperiencia, error) {
	if len(contenido) == 0 || len(contenido) > maximoBytesRepresentacionEntrada {
		return EntradaExperiencia{}, nuevoError("representacion_canonica", CodigoFueraDeLimites)
	}
	if !utf8.Valid(contenido) {
		return EntradaExperiencia{}, nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	if err := comprobarJSONEntradaSinClavesDuplicadas(contenido); err != nil {
		return EntradaExperiencia{}, err
	}
	var material materialEntradaExperiencia
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&material); err != nil {
		return EntradaExperiencia{}, clasificarErrorDecodificacionEntrada(
			err,
			"representacion_canonica",
		)
	}
	var sobrante struct{}
	if err := decodificador.Decode(&sobrante); !errors.Is(err, io.EOF) {
		return EntradaExperiencia{}, nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	if material.Esquema != esquemaEntradaExperiencia {
		return EntradaExperiencia{}, nuevoError("esquema", CodigoEsquemaIncompatible)
	}
	entrada, err := reconstruirEntrada(material)
	if err != nil {
		return EntradaExperiencia{}, err
	}
	canonico, err := entrada.RepresentacionCanonica()
	if err != nil || !bytes.Equal(canonico, contenido) {
		return EntradaExperiencia{}, nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	return entrada, nil
}

// comprobarJSONEntradaSinClavesDuplicadas recorre la representacion antes de
// materializarla. encoding/json conserva el ultimo valor de una clave repetida;
// sin esta puerta un atacante podria forzar la decodificacion reiterada de
// varias colecciones validas y amplificar CPU y asignaciones antes del rechazo
// canonico final.
func comprobarJSONEntradaSinClavesDuplicadas(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.UseNumber()
	if err := recorrerValorJSONEntrada(decodificador, 0); err != nil {
		return err
	}
	return exigirFinJSON(decodificador)
}

func recorrerValorJSONEntrada(decodificador *json.Decoder, profundidad int) error {
	if profundidad > maximaProfundidadJSONEntrada {
		return nuevoError("representacion_canonica", CodigoFueraDeLimites)
	}
	token, err := decodificador.Token()
	if err != nil {
		return nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	delimitador, esDelimitador := token.(json.Delim)
	if !esDelimitador {
		return nil
	}
	switch delimitador {
	case '{':
		var claves [maximosCamposObjetoJSONEntrada]string
		cantidad := 0
		for decodificador.More() {
			tokenClave, err := decodificador.Token()
			clave, valida := tokenClave.(string)
			if err != nil || !valida || cantidad >= len(claves) {
				return nuevoError("representacion_canonica", CodigoValorNoCanonico)
			}
			for indice := 0; indice < cantidad; indice++ {
				if claves[indice] == clave {
					return nuevoError("representacion_canonica", CodigoValorNoCanonico)
				}
			}
			claves[cantidad] = clave
			cantidad++
			if err := recorrerValorJSONEntrada(decodificador, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim('}') {
			return nuevoError("representacion_canonica", CodigoValorNoCanonico)
		}
	case '[':
		for decodificador.More() {
			if err := recorrerValorJSONEntrada(decodificador, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim(']') {
			return nuevoError("representacion_canonica", CodigoValorNoCanonico)
		}
	default:
		return nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	return nil
}

func RestaurarEntradaExperienciaConHuellaSHA256(
	contenido []byte,
	huellaEsperada string,
) (EntradaExperiencia, error) {
	if !huellaSHA256Canonica(huellaEsperada) {
		return EntradaExperiencia{}, nuevoError("huella_esperada_sha256", CodigoValorNoCanonico)
	}
	entrada, err := RestaurarEntradaExperiencia(contenido)
	if err != nil {
		return EntradaExperiencia{}, err
	}
	huellaReal, err := entrada.HuellaSHA256()
	if err != nil || subtle.ConstantTimeCompare([]byte(huellaReal), []byte(huellaEsperada)) != 1 {
		return EntradaExperiencia{}, nuevoError("huella_esperada_sha256", CodigoHuellaNoCoincide)
	}
	return entrada, nil
}

func materializarTramo(tramo TramoExperiencia) materialTramo {
	periodo := materialPeriodo{Modo: tramo.periodo.modo, Desde: tramo.periodo.desde}
	if tramo.periodo.modo == PeriodoServicioCerrado {
		fin := tramo.periodo.finInformado
		periodo.FinInformado = &fin
	}
	atestacion := materialAtestacion{Modo: tramo.atestacion.modo}
	if tramo.atestacion.modo == computoIntegroAtestado {
		referencia := materializarReferencia(tramo.atestacion.referencia)
		atestacion.Referencia = &referencia
	}
	atributos := make([]materialAtributo, len(tramo.atributos))
	for indice, atributo := range tramo.atributos {
		atributos[indice] = materialAtributo{
			Clave:    atributo.clave,
			Catalogo: materializarReferencia(atributo.catalogo),
			Valor:    atributo.valor,
		}
	}
	return materialTramo{
		Referencia:  materializarReferencia(tramo.referencia),
		ServicioRef: tramo.servicioRef,
		Periodo:     periodo,
		Jornada:     tramo.jornada,
		Atestacion:  atestacion,
		Atributos:   atributos,
	}
}

func materializarReferencia(referencia reglasbaremo.ReferenciaVersionada) materialReferencia {
	return materialReferencia{
		Referencia:   referencia.Referencia(),
		Version:      referencia.Version(),
		HuellaSHA256: referencia.HuellaSHA256(),
	}
}

func reconstruirEntrada(material materialEntradaExperiencia) (EntradaExperiencia, error) {
	if len(material.Tramos) > maximoTramosEntrada {
		return EntradaExperiencia{}, nuevoError("entrada.tramos", CodigoFueraDeLimites)
	}
	if err := comprobarPresupuestoMaterialEntrada(material); err != nil {
		return EntradaExperiencia{}, err
	}
	instantanea, err := reconstruirReferenciaEntrada(material.Instantanea, "entrada.instantanea")
	if err != nil {
		return EntradaExperiencia{}, err
	}
	tramos := make([]TramoExperiencia, len(material.Tramos))
	for indice, origen := range material.Tramos {
		tramos[indice], err = reconstruirTramo(origen)
		if err != nil {
			return EntradaExperiencia{}, err
		}
	}
	return NuevaEntradaExperiencia(instantanea, tramos)
}

func reconstruirTramo(material materialTramo) (TramoExperiencia, error) {
	if len(material.Atributos) > maximoAtributosPorTramo {
		return TramoExperiencia{}, nuevoError("tramo.atributos", CodigoFueraDeLimites)
	}
	referencia, err := reconstruirReferenciaEntrada(material.Referencia, "tramo.referencia")
	if err != nil {
		return TramoExperiencia{}, err
	}
	periodo, err := reconstruirPeriodo(material.Periodo)
	if err != nil {
		return TramoExperiencia{}, err
	}
	atestacion, err := reconstruirAtestacion(material.Atestacion)
	if err != nil {
		return TramoExperiencia{}, err
	}
	atributos := make([]AtributoCatalogado, len(material.Atributos))
	for indice, origen := range material.Atributos {
		catalogo, err := reconstruirReferenciaEntrada(origen.Catalogo, "atributo.catalogo")
		if err != nil {
			return TramoExperiencia{}, err
		}
		atributos[indice], err = NuevoAtributoCatalogado(origen.Clave, catalogo, origen.Valor)
		if err != nil {
			return TramoExperiencia{}, err
		}
	}
	return NuevoTramoExperiencia(
		referencia,
		material.ServicioRef,
		periodo,
		material.Jornada,
		atestacion,
		atributos,
	)
}

func reconstruirPeriodo(material materialPeriodo) (PeriodoServicio, error) {
	switch material.Modo {
	case PeriodoServicioCerrado:
		if material.FinInformado == nil {
			return PeriodoServicio{}, nuevoError("periodo.fin_informado", CodigoValorNoCanonico)
		}
		return NuevoPeriodoServicioCerrado(material.Desde, *material.FinInformado)
	case PeriodoServicioEnCurso:
		if material.FinInformado != nil {
			return PeriodoServicio{}, nuevoError("periodo.fin_informado", CodigoValorNoCanonico)
		}
		return NuevoPeriodoServicioEnCurso(material.Desde)
	default:
		return PeriodoServicio{}, nuevoError("periodo.modo", CodigoValorNoCanonico)
	}
}

func reconstruirAtestacion(material materialAtestacion) (ComputoIntegroAtestado, error) {
	switch material.Modo {
	case computoIntegroAusente:
		if material.Referencia != nil {
			return ComputoIntegroAtestado{}, nuevoError(
				"computo_integro_atestado.referencia", CodigoValorNoCanonico,
			)
		}
		return SinComputoIntegroAtestado(), nil
	case computoIntegroAtestado:
		if material.Referencia == nil {
			return ComputoIntegroAtestado{}, nuevoError(
				"computo_integro_atestado.referencia", CodigoValorNoCanonico,
			)
		}
		referencia, err := reconstruirReferenciaEntrada(
			*material.Referencia,
			"computo_integro_atestado.referencia",
		)
		if err != nil {
			return ComputoIntegroAtestado{}, err
		}
		return NuevoComputoIntegroAtestado(referencia)
	default:
		return ComputoIntegroAtestado{}, nuevoError(
			"computo_integro_atestado.modo", CodigoValorNoCanonico,
		)
	}
}

func reconstruirReferenciaEntrada(
	material materialReferencia,
	campo string,
) (reglasbaremo.ReferenciaVersionada, error) {
	referencia, err := reglasbaremo.NuevaReferenciaVersionada(
		material.Referencia,
		material.Version,
		material.HuellaSHA256,
	)
	if err != nil {
		return reglasbaremo.ReferenciaVersionada{}, nuevoError(campo, CodigoValorNoCanonico)
	}
	return referencia, nil
}

func huellaSHA256Canonica(valor string) bool {
	if len(valor) != sha256.Size*2 {
		return false
	}
	for indice := 0; indice < len(valor); indice++ {
		caracter := valor[indice]
		if (caracter >= '0' && caracter <= '9') || (caracter >= 'a' && caracter <= 'f') {
			continue
		}
		return false
	}
	return true
}

func comprobarPresupuestoMaterialEntrada(material materialEntradaExperiencia) error {
	total := 512 + tamanoMaterialReferenciaEstimado(material.Instantanea)
	for _, tramo := range material.Tramos {
		if len(tramo.Atributos) > maximoAtributosPorTramo {
			return nuevoError("tramo.atributos", CodigoFueraDeLimites)
		}
		total += 768 + tamanoMaterialReferenciaEstimado(tramo.Referencia) + len(tramo.ServicioRef)
		if tramo.Atestacion.Referencia != nil {
			total += tamanoMaterialReferenciaEstimado(*tramo.Atestacion.Referencia)
		}
		for _, atributo := range tramo.Atributos {
			total += 256 + len(atributo.Clave) + len(atributo.Valor) +
				tamanoMaterialReferenciaEstimado(atributo.Catalogo)
		}
		if total > maximoBytesRepresentacionEntrada {
			return nuevoError("representacion_canonica", CodigoFueraDeLimites)
		}
	}
	return nil
}

func tamanoMaterialReferenciaEstimado(referencia materialReferencia) int {
	return 160 + len(referencia.Referencia) + len(referencia.HuellaSHA256)
}
