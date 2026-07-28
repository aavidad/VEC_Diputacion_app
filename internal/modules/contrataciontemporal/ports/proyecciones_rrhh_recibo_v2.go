package ports

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var patronAccesoRRHHV2 = regexp.MustCompile(`^acceso:rrhh:[0-9a-f]{32}$`)

const (
	VersionCanonReciboLecturaRRHHV2 uint16 = 2
	DominioCanonReciboLecturaRRHHV2        = "vec.contratacion_temporal.recibo_lectura_rrhh.v2"

	cabeceraCanonReciboLecturaRRHHV2 = "VEC-CT-RECIBO-LECTURA-RRHH-V2\n"
)

// ResultadoRegistradorAccesoRRHHV2 representa, sin ampliar su autoridad, los
// siete valores probatorios devueltos por registrar_acceso_rrhh_v2. El NULL
// SQL de alcance_huella_sha256 se adapta como cadena vacía.
//
// No contiene auditoría VEC, consumo VEC ni resultado de consulta: esos datos
// no los devuelve el registrador y pertenecen a EvidenciaConsumoResultadoRRHHV2.
type ResultadoRegistradorAccesoRRHHV2 struct {
	bloqueoSerializacionConsultaRRHH
	accesoRef                    string
	secuencia                    uint64
	anteriorSHA256               string
	huellaSHA256                 string
	vinculoIdentidadHuellaSHA256 string
	alcanceHuellaSHA256          string
	registradaEn                 time.Time
}

// NuevoResultadoRegistradorAccesoRRHHV2 adapta literalmente la salida SQL. No
// reconstruye campos ni acepta el objeto JSON abierto devuelto por PostgreSQL.
func NuevoResultadoRegistradorAccesoRRHHV2(
	accesoRef string,
	secuencia uint64,
	anteriorSHA256 string,
	huellaSHA256 string,
	vinculoIdentidadHuellaSHA256 string,
	alcanceHuellaSHA256 string,
	registradaEn time.Time,
) (ResultadoRegistradorAccesoRRHHV2, error) {
	resultado := ResultadoRegistradorAccesoRRHHV2{
		accesoRef:                    accesoRef,
		secuencia:                    secuencia,
		anteriorSHA256:               anteriorSHA256,
		huellaSHA256:                 huellaSHA256,
		vinculoIdentidadHuellaSHA256: vinculoIdentidadHuellaSHA256,
		alcanceHuellaSHA256:          alcanceHuellaSHA256,
		registradaEn:                 registradaEn,
	}
	if resultado.validar() != nil {
		return ResultadoRegistradorAccesoRRHHV2{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	return resultado, nil
}

func (r ResultadoRegistradorAccesoRRHHV2) validar() error {
	alcanceValido := r.alcanceHuellaSHA256 == "" ||
		huellaSHA256CanonicaRRHH(r.alcanceHuellaSHA256)
	anteriorValido := huellaCadenaAnteriorRRHHV2Valida(
		r.anteriorSHA256, r.secuencia,
	)
	if !patronAccesoRRHHV2.MatchString(r.accesoRef) ||
		r.secuencia < 1 || r.secuencia > versionMaximaJSONSegura ||
		!anteriorValido ||
		!huellaSHA256CanonicaRRHH(r.huellaSHA256) ||
		!huellaSHA256CanonicaRRHH(r.vinculoIdentidadHuellaSHA256) ||
		!alcanceValido ||
		!domain.InstanteUTCCanonico(r.registradaEn) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

func huellaCadenaAnteriorRRHHV2Valida(valor string, secuencia uint64) bool {
	if len(valor) != sha256.Size*2 || valor != strings.ToLower(valor) {
		return false
	}
	decodificada, err := hex.DecodeString(valor)
	defer clear(decodificada)
	if err != nil || len(decodificada) != sha256.Size {
		return false
	}
	esCero := valor == strings.Repeat("0", sha256.Size*2)
	return secuencia == 1 && esCero || secuencia > 1 && !esCero
}

func (ResultadoRegistradorAccesoRRHHV2) String() string {
	return "[resultado-registrador-acceso-rrhh-v2-opaco]"
}

func (ResultadoRegistradorAccesoRRHHV2) GoString() string {
	return "[resultado-registrador-acceso-rrhh-v2-opaco]"
}

// EvidenciaConsumoResultadoRRHHV2 reúne sólo la evidencia que procede del
// consumo VEC y del motor SQL de resultados. No duplica ni suplanta la salida
// del registrador de accesos.
type EvidenciaConsumoResultadoRRHHV2 struct {
	bloqueoSerializacionConsultaRRHH
	auditoriaRef          string
	auditoriaHuellaSHA256 string
	consumoHuellaSHA256   string
	contenidoHuellaSHA256 string
	resultadoHuellaSHA256 string
	cursorHuellaSHA256    string
	generadaEn            time.Time
	expedienteRef         string
	version               uint64
	total                 uint16
}

// NuevaEvidenciaConsumoResultadoRRHHV2 crea una prueba nominal cerrada. Para
// cuadro, expedienteRef y version son cero; para detalle se exige expediente,
// version y cardinalidad uno. Nunca recibe el cursor en claro.
func NuevaEvidenciaConsumoResultadoRRHHV2(
	auditoriaRef string,
	auditoriaHuellaSHA256 string,
	consumoHuellaSHA256 string,
	contenidoHuellaSHA256 string,
	resultadoHuellaSHA256 string,
	cursorHuellaSHA256 string,
	generadaEn time.Time,
	expedienteRef string,
	version uint64,
	total uint16,
) (EvidenciaConsumoResultadoRRHHV2, error) {
	evidencia := EvidenciaConsumoResultadoRRHHV2{
		auditoriaRef:          auditoriaRef,
		auditoriaHuellaSHA256: auditoriaHuellaSHA256,
		consumoHuellaSHA256:   consumoHuellaSHA256,
		contenidoHuellaSHA256: contenidoHuellaSHA256,
		resultadoHuellaSHA256: resultadoHuellaSHA256,
		cursorHuellaSHA256:    cursorHuellaSHA256,
		generadaEn:            generadaEn,
		expedienteRef:         expedienteRef,
		version:               version,
		total:                 total,
	}
	if evidencia.validar() != nil {
		return EvidenciaConsumoResultadoRRHHV2{},
			ErrResultadoConsultaRRHHNoConfiable
	}
	return evidencia, nil
}

func (e EvidenciaConsumoResultadoRRHHV2) validar() error {
	esCuadro := e.expedienteRef == "" && e.version == 0 &&
		e.total <= LimiteMaximoCuadroRRHH &&
		(e.cursorHuellaSHA256 == "" ||
			e.total > 0 && huellaSHA256CanonicaRRHH(e.cursorHuellaSHA256))
	esDetalle := domain.ReferenciaOpacaValida(e.expedienteRef) &&
		e.version >= 1 && e.version <= versionMaximaJSONSegura &&
		e.total == 1 && e.cursorHuellaSHA256 == ""
	if !domain.ReferenciaOpacaValida(e.auditoriaRef) ||
		!huellaSHA256CanonicaRRHH(e.auditoriaHuellaSHA256) ||
		!huellaSHA256CanonicaRRHH(e.consumoHuellaSHA256) ||
		!huellaSHA256CanonicaRRHH(e.contenidoHuellaSHA256) ||
		!huellaSHA256CanonicaRRHH(e.resultadoHuellaSHA256) ||
		!domain.InstanteUTCCanonico(e.generadaEn) ||
		(!esCuadro && !esDetalle) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

func (EvidenciaConsumoResultadoRRHHV2) String() string {
	return "[evidencia-consumo-resultado-rrhh-v2-opaca]"
}

func (EvidenciaConsumoResultadoRRHHV2) GoString() string {
	return "[evidencia-consumo-resultado-rrhh-v2-opaca]"
}

// NuevoReciboLecturaRRHHV2 enlaza las dos autoridades nominales con el
// Contexto y la Capacidad. Decisión, material, consulta, sesión, ámbito,
// acción y finalidad se derivan siempre; el adaptador no puede sustituirlos.
//
// selloCanonSHA256 es la huella del canon encuadrado descrito en
// canonReciboLecturaRRHHV2. La comparación se realiza en tiempo constante.
// La huella de cadena, el vínculo de identidad, el alcance, la auditoría y el
// consumo son pruebas autoritativas SQL: se validan nominalmente y quedan
// ligadas por el sello, pero no se finge que Contexto o Capacidad permitan
// recalcularlas. acceso_ref sí se cruza con consumo_huella_sha256 porque su
// derivación está publicada en el contrato SQL.
func NuevoReciboLecturaRRHHV2(
	contexto ContextoConsultaRRHH,
	capacidad CapacidadConsultaRRHH,
	registro ResultadoRegistradorAccesoRRHHV2,
	evidencia EvidenciaConsumoResultadoRRHHV2,
	selloCanonSHA256 string,
) (ReciboLecturaRRHH, error) {
	if registro.validar() != nil || evidencia.validar() != nil ||
		!huellaSHA256CanonicaRRHH(selloCanonSHA256) ||
		capacidad.validaPara(
			contexto, capacidad.consultaDominio, capacidad.consultaHuella,
			capacidad.accion, capacidad.finalidad, evidencia.expedienteRef,
			registro.registradaEn,
		) != nil ||
		evidencia.generadaEn.Before(capacidad.validaDesde) ||
		registro.registradaEn.Before(evidencia.generadaEn) {
		return ReciboLecturaRRHH{}, ErrResultadoConsultaRRHHNoConfiable
	}

	sello, err := decodificarHuellaReciboRRHHV2(selloCanonSHA256)
	if err != nil {
		return ReciboLecturaRRHH{}, ErrResultadoConsultaRRHHNoConfiable
	}
	recibo := ReciboLecturaRRHH{
		versionRecibo:           2,
		lecturaRef:              registro.accesoRef,
		auditoriaRef:            evidencia.auditoriaRef,
		decisionRef:             capacidad.decisionRef,
		decisionHuella:          capacidad.decisionHuella,
		capacidadHuella:         capacidad.capacidadHuella,
		materialHuella:          capacidad.materialHuella,
		consultaHuella:          capacidad.consultaHuella,
		correlacionRef:          capacidad.correlacionRef,
		sesionRef:               contexto.sesionRef,
		organizacionRef:         contexto.organizacionRef,
		claseAmbito:             capacidad.claseAmbito,
		ambitoRef:               capacidad.ambitoRef,
		accion:                  capacidad.accion,
		finalidad:               capacidad.finalidad,
		expedienteRef:           evidencia.expedienteRef,
		version:                 evidencia.version,
		totalPublicado:          evidencia.total,
		registradaEn:            registro.registradaEn,
		autenticacionRefV2:      contexto.autenticacionRef,
		autenticacionHuellaV2:   contexto.autenticacionHuella,
		controlSesionRefV2:      contexto.controlSesionRef,
		controlSesionRevisionV2: contexto.controlSesionRevision,
		controlSesionHuellaV2:   contexto.controlSesionHuellaSHA256,
		actorRefV2:              contexto.actorRef,
		perfilRefV2:             contexto.perfilRef,
		perfilVersionV2:         contexto.perfilVersion,
		registroV2:              registro,
		evidenciaV2:             evidencia,
		selloCanonV2:            sello,
	}
	if recibo.validarV2() != nil {
		return ReciboLecturaRRHH{}, ErrResultadoConsultaRRHHNoConfiable
	}
	return recibo, nil
}

func (r ReciboLecturaRRHH) validarV2() error {
	if r.versionRecibo != 2 ||
		r.validarCamposComunes() != nil ||
		r.registroV2.validar() != nil ||
		r.evidenciaV2.validar() != nil ||
		r.lecturaRef != r.registroV2.accesoRef ||
		r.auditoriaRef != r.evidenciaV2.auditoriaRef ||
		r.expedienteRef != r.evidenciaV2.expedienteRef ||
		r.version != r.evidenciaV2.version ||
		r.totalPublicado != r.evidenciaV2.total ||
		!r.registradaEn.Equal(r.registroV2.registradaEn) ||
		r.registradaEn.Before(r.evidenciaV2.generadaEn) ||
		!r.contextoV2Valido() ||
		!r.accesoDerivadoDelConsumoV2() ||
		!r.formaConsultaV2Valida() {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	canon, err := r.canonReciboLecturaRRHHV2()
	if err != nil {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	calculado := sha256.Sum256(canon)
	clear(canon)
	if subtle.ConstantTimeCompare(calculado[:], r.selloCanonV2[:]) != 1 {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

func (r ReciboLecturaRRHH) formaConsultaV2Valida() bool {
	esCuadro := r.accion == AccionConsultarCuadroRRHH &&
		r.registroV2.alcanceHuellaSHA256 != "" &&
		r.evidenciaV2.expedienteRef == "" &&
		r.evidenciaV2.version == 0
	esDetalle := r.accion == AccionConsultarDetalleRRHH &&
		r.registroV2.alcanceHuellaSHA256 == "" &&
		domain.ReferenciaOpacaValida(r.evidenciaV2.expedienteRef) &&
		r.evidenciaV2.version >= 1
	return esCuadro || esDetalle
}

func (r ReciboLecturaRRHH) contextoV2Valido() bool {
	return domain.ReferenciaOpacaValida(r.autenticacionRefV2) &&
		huellaSHA256CanonicaRRHH(r.autenticacionHuellaV2) &&
		domain.ReferenciaOpacaValida(r.controlSesionRefV2) &&
		r.controlSesionRevisionV2 >= 1 &&
		r.controlSesionRevisionV2 <= versionMaximaJSONSegura &&
		huellaSHA256CanonicaRRHH(r.controlSesionHuellaV2) &&
		domain.ReferenciaOpacaValida(r.actorRefV2) &&
		domain.ReferenciaOpacaValida(r.perfilRefV2) &&
		r.perfilVersionV2 >= 1 &&
		r.perfilVersionV2 <= versionMaximaJSONSegura
}

func (r ReciboLecturaRRHH) coincideConContextoV2(
	contexto ContextoConsultaRRHH,
) bool {
	return r.autenticacionRefV2 == contexto.autenticacionRef &&
		r.autenticacionHuellaV2 == contexto.autenticacionHuella &&
		r.sesionRef == contexto.sesionRef &&
		r.controlSesionRefV2 == contexto.controlSesionRef &&
		r.controlSesionRevisionV2 == contexto.controlSesionRevision &&
		r.controlSesionHuellaV2 == contexto.controlSesionHuellaSHA256 &&
		r.actorRefV2 == contexto.actorRef &&
		r.perfilRefV2 == contexto.perfilRef &&
		r.perfilVersionV2 == contexto.perfilVersion
}

func (r ReciboLecturaRRHH) accesoDerivadoDelConsumoV2() bool {
	suma := sha256.Sum256([]byte(
		"acceso:rrhh:" + r.evidenciaV2.consumoHuellaSHA256,
	))
	esperada := "acceso:rrhh:" + hex.EncodeToString(suma[:])[:32]
	return subtle.ConstantTimeCompare(
		[]byte(esperada), []byte(r.registroV2.accesoRef),
	) == 1
}

// canonReciboLecturaRRHHV2 no incluye el sello para evitar autorreferencia.
// Orden estable, todos los campos encuadrados:
//   - siete valores exactos de registrar_acceso_rrhh_v2;
//   - auditoría y consumo VEC;
//   - identidad derivada de Contexto y autorización derivada de Capacidad;
//   - consulta, sesión, ámbito, acción y finalidad derivadas;
//   - expediente, versión, cardinalidad y huellas del resultado.
//
// Nunca incluye JSON abierto, token de capacidad ni cursor en claro.
func (r ReciboLecturaRRHH) canonReciboLecturaRRHHV2() ([]byte, error) {
	c := nuevoConstructorCanonResultadoRRHH(
		cabeceraCanonReciboLecturaRRHHV2,
	)
	c.texto(r.registroV2.accesoRef)
	c.enteroSinSigno(r.registroV2.secuencia)
	c.texto(r.registroV2.anteriorSHA256)
	c.texto(r.registroV2.huellaSHA256)
	c.texto(r.registroV2.vinculoIdentidadHuellaSHA256)
	c.texto(r.registroV2.alcanceHuellaSHA256)
	c.instante(r.registroV2.registradaEn)
	c.texto(r.evidenciaV2.auditoriaRef)
	c.texto(r.evidenciaV2.auditoriaHuellaSHA256)
	c.texto(r.evidenciaV2.consumoHuellaSHA256)
	c.texto(r.decisionRef)
	c.texto(r.decisionHuella)
	c.texto(r.capacidadHuella)
	c.texto(r.materialHuella)
	c.texto(r.consultaHuella)
	c.texto(r.correlacionRef)
	c.texto(r.autenticacionRefV2)
	c.texto(r.autenticacionHuellaV2)
	c.texto(r.sesionRef)
	c.texto(r.controlSesionRefV2)
	c.enteroSinSigno(r.controlSesionRevisionV2)
	c.texto(r.controlSesionHuellaV2)
	c.texto(r.actorRefV2)
	c.texto(r.perfilRefV2)
	c.enteroSinSigno(r.perfilVersionV2)
	c.texto(r.organizacionRef)
	c.texto(string(r.claseAmbito))
	c.texto(r.ambitoRef)
	c.texto(r.accion)
	c.texto(r.finalidad)
	c.texto(r.evidenciaV2.expedienteRef)
	c.enteroSinSigno(r.evidenciaV2.version)
	c.enteroSinSigno(uint64(r.evidenciaV2.total))
	c.texto(r.evidenciaV2.contenidoHuellaSHA256)
	c.texto(r.evidenciaV2.resultadoHuellaSHA256)
	c.texto(r.evidenciaV2.cursorHuellaSHA256)
	c.instante(r.evidenciaV2.generadaEn)
	return c.finalizar()
}

func decodificarHuellaReciboRRHHV2(
	valor string,
) ([sha256.Size]byte, error) {
	var resultado [sha256.Size]byte
	decodificada, err := hex.DecodeString(valor)
	defer clear(decodificada)
	if err != nil || len(decodificada) != sha256.Size {
		return resultado, ErrResultadoConsultaRRHHNoConfiable
	}
	copy(resultado[:], decodificada)
	return resultado, nil
}
