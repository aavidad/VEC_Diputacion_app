package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrMensajeAtestacionAutorizacionInvalido = errors.New("vec: mensaje de atestacion de autorizacion invalido")

const (
	// VersionFormatoAtestacionAutorizacionV1 identifica la representacion
	// binaria VEC-AD-1. No identifica ni aprueba un algoritmo de firma.
	VersionFormatoAtestacionAutorizacionV1 uint16 = 1

	// EsquemaMensajeAtestacionAutorizacionV1 separa este mensaje de cualquier
	// otro uso criptografico. El byte cero posterior forma parte del esquema.
	EsquemaMensajeAtestacionAutorizacionV1 = "VEC-AUTORIZACION-ATESTACION-V1-AUTENTICACION-ACTOR"

	// TamanoMaximoMensajeAtestacionAutorizacionV1 mantiene el mensaje como una
	// capacidad breve y coincide con el techo del documento de decision durable.
	TamanoMaximoMensajeAtestacionAutorizacionV1 = 512 * 1024
)

// CabeceraAtestacionAutorizacionV1 contiene toda la configuracion que debe
// estar seleccionada de forma exacta antes de solicitar una firma. Suite y
// ClaveID son identificadores; este tipo no implementa ni aprueba algoritmos,
// proveedores o material criptografico.
type CabeceraAtestacionAutorizacionV1 struct {
	FormatoVersion uint16
	Suite          string
	ClaveID        string
	Audiencia      string
}

func (c CabeceraAtestacionAutorizacionV1) Validar() error {
	if c.FormatoVersion != VersionFormatoAtestacionAutorizacionV1 ||
		!identificadorCabeceraAtestacionValido(c.Suite, 128) ||
		!identificadorCabeceraAtestacionValido(c.ClaveID, 512) ||
		!identificadorCabeceraAtestacionValido(c.Audiencia, 512) {
		return errors.Join(ErrConfiguracionAccesoInvalida, ErrMensajeAtestacionAutorizacionInvalido)
	}
	return nil
}

// SerializarMensajeAtestacionAutorizacionV1 produce la unica representacion
// binaria VEC-AD-1 de una concesion reforzada. No ordena ni corrige las listas
// recibidas: una lista que no llegue ya en orden UTF-8 estricto se rechaza. Los
// mapas, cuyo orden de iteracion no forma parte del valor Go, se emiten por
// clave UTF-8 ascendente.
func SerializarMensajeAtestacionAutorizacionV1(
	cabecera CabeceraAtestacionAutorizacionV1,
	decision DecisionAutorizacion,
) ([]byte, error) {
	if err := cabecera.Validar(); err != nil {
		return nil, err
	}
	if err := validarDecisionParaAtestacionAutorizacionV1(decision); err != nil {
		return nil, err
	}

	escritor := nuevoEscritorAtestacionAutorizacionV1()
	escritor.escribirBytes([]byte(EsquemaMensajeAtestacionAutorizacionV1))
	escritor.escribirByte(0)
	escritor.escribirUint16(cabecera.FormatoVersion)
	escritor.escribirTexto(cabecera.Suite)
	escritor.escribirTexto(cabecera.ClaveID)
	escritor.escribirTexto(cabecera.Audiencia)

	// Orden contractual cerrado de los 30 campos de DecisionAutorizacion.
	escritor.escribirTexto(decision.DecisionRef)
	escritor.escribirBooleano(decision.Concedida)
	escritor.escribirTexto(decision.Codigo)
	escritor.escribirTexto(decision.PrincipalID)
	escritor.escribirTexto(decision.PerfilActivoRef)
	escritor.escribirTexto(decision.Accion)
	escritor.escribirTexto(decision.RecursoRef)
	escritor.escribirTexto(decision.ModuloID)
	escritor.escribirTexto(decision.TipoRecurso)
	escritor.escribirTexto(decision.ContextoRecursoHuellaSHA256)
	escritor.escribirTexto(decision.Finalidad)
	escritor.escribirTexto(decision.CorrelacionRef)
	escribirVinculoAutenticacionActorV1(escritor, decision.VinculoAutenticacionActor)
	escritor.escribirTexto(decision.AsignacionRef)
	escritor.escribirTexto(decision.AsignacionHuellaSHA256)
	escritor.escribirTexto(decision.VersionRolRef)
	escritor.escribirTexto(decision.VersionRolHuellaSHA256)
	escritor.escribirTexto(decision.ControlVigenciaVersionRolRef)
	escritor.escribirUint64(decision.ControlVigenciaVersionRolRevision)
	escritor.escribirTexto(decision.ControlVigenciaVersionRolHuellaSHA256)
	escritor.escribirUint64(decision.RevisionCatalogoPoliticas)
	escritor.escribirTexto(decision.CatalogoPoliticasHuellaSHA256)
	escritor.escribirLista(decision.PoliticasEvaluadasRefs)
	escritor.escribirMapa(decision.PoliticasEvaluadasHuellasSHA256)
	escritor.escribirLista(decision.PoliticasRefs)
	escritor.escribirMapa(decision.PoliticasHuellasSHA256)
	escritor.escribirTexto(string(decision.GarantiaMinima))
	escritor.escribirLista(decision.CamposPermitidos)
	escritor.escribirLista(decision.Obligaciones)
	escritor.escribirInstante(decision.EmitidaEn)
	escritor.escribirInstante(decision.ValidaHasta)
	if escritor.err != nil {
		return nil, escritor.err
	}

	// La longitud final incluye todo el mensaje, incluidos sus propios 8 bytes.
	if escritor.buffer.Len() > TamanoMaximoMensajeAtestacionAutorizacionV1-8 {
		return nil, errorMensajeAtestacionAutorizacionInvalido()
	}
	longitudTotal := uint64(escritor.buffer.Len() + 8)
	escritor.escribirUint64(longitudTotal)
	if escritor.err != nil || escritor.buffer.Len() != int(longitudTotal) ||
		escritor.buffer.Len() > TamanoMaximoMensajeAtestacionAutorizacionV1 {
		return nil, errorMensajeAtestacionAutorizacionInvalido()
	}
	return append([]byte(nil), escritor.buffer.Bytes()...), nil
}

func escribirVinculoAutenticacionActorV1(
	escritor *escritorAtestacionAutorizacionV1,
	vinculo VinculoAutenticacionActorV1,
) {
	datos, err := vinculo.Datos()
	if err != nil {
		escritor.err = errorMensajeAtestacionAutorizacionInvalido()
		return
	}
	escritor.escribirUint16(datos.BloqueVersion)
	escritor.escribirTexto(datos.AutenticacionRef)
	escritor.escribirTexto(datos.AutenticacionHuellaSHA256)
	escritor.escribirTexto(datos.AsercionRef)
	escritor.escribirTexto(datos.SesionRef)
	escritor.escribirTexto(datos.ControlSesionRef)
	escritor.escribirUint64(datos.ControlSesionRevision)
	escritor.escribirTexto(datos.ControlSesionHuellaSHA256)
	escritor.escribirTexto(datos.CuentaRef)
	escritor.escribirTexto(datos.CuentaOrdinariaRef)
	escritor.escribirTexto(datos.PrincipalID)
	escritor.escribirTexto(datos.PerfilActivoRef)
	escritor.escribirBooleano(datos.CuentaPrivilegiada)
	escritor.escribirTexto(string(datos.Superficie))
	escritor.escribirTexto(string(datos.MetodoObservado))
	escritor.escribirTexto(string(datos.GarantiaObservada))
	escritor.escribirTexto(datos.PoliticaGarantiaRef)
	escritor.escribirTexto(datos.PoliticaGarantiaHuellaSHA256)
	escritor.escribirInstante(datos.AutenticacionVerificadaEn)
	escritor.escribirInstante(datos.SesionEmitidaEn)
	escritor.escribirInstante(datos.SesionValidaHasta)
	escritor.escribirInstante(datos.SesionRevalidadaEn)
	escritor.escribirTexto(datos.ContextoActorRef)
	escritor.escribirUint64(datos.ContextoActorVersion)
	escritor.escribirTexto(datos.ContextoActorHuellaSHA256)
}

// HuellaSHA256MensajeAtestacionAutorizacionV1 permite publicar vectores de
// interoperabilidad sin convertir la huella en firma o autorizacion.
func HuellaSHA256MensajeAtestacionAutorizacionV1(
	cabecera CabeceraAtestacionAutorizacionV1,
	decision DecisionAutorizacion,
) (string, error) {
	mensaje, err := SerializarMensajeAtestacionAutorizacionV1(cabecera, decision)
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(mensaje)
	return hex.EncodeToString(suma[:]), nil
}

func validarDecisionParaAtestacionAutorizacionV1(decision DecisionAutorizacion) error {
	if decision.ValidarEvidenciaInstantanea() != nil || !decision.Concedida || decision.Codigo != "concedida" ||
		contieneComodinAtestacionAutorizacion(decision) ||
		!listaAtestacionAutorizacionCanonica(decision.PoliticasEvaluadasRefs) ||
		!listaAtestacionAutorizacionCanonica(decision.PoliticasRefs) ||
		!listaAtestacionAutorizacionCanonica(decision.CamposPermitidos) ||
		!listaAtestacionAutorizacionCanonica(decision.Obligaciones) {
		return errors.Join(ErrDecisionAutorizacionInvalida, ErrMensajeAtestacionAutorizacionInvalido)
	}
	return nil
}

func identificadorCabeceraAtestacionValido(valor string, maximo int) bool {
	if valor == "" || len(valor) > maximo || valor != strings.TrimSpace(valor) ||
		!utf8.ValidString(valor) || strings.ContainsRune(valor, '*') {
		return false
	}
	// La cabecera es configuracion de seguridad, no texto humano. ASCII visible
	// evita normalizaciones Unicode o identificadores visualmente ambiguos.
	for _, caracter := range []byte(valor) {
		if caracter < 0x21 || caracter > 0x7e {
			return false
		}
	}
	return true
}

func listaAtestacionAutorizacionCanonica(valores []string) bool {
	for indice := 1; indice < len(valores); indice++ {
		if bytes.Compare([]byte(valores[indice-1]), []byte(valores[indice])) >= 0 {
			return false
		}
	}
	return true
}

func contieneComodinAtestacionAutorizacion(decision DecisionAutorizacion) bool {
	valores := []string{
		decision.DecisionRef, decision.Codigo, decision.PrincipalID, decision.PerfilActivoRef,
		decision.Accion, decision.RecursoRef, decision.ModuloID, decision.TipoRecurso,
		decision.ContextoRecursoHuellaSHA256, decision.Finalidad, decision.CorrelacionRef,
		decision.AsignacionRef, decision.AsignacionHuellaSHA256, decision.VersionRolRef,
		decision.VersionRolHuellaSHA256, decision.ControlVigenciaVersionRolRef,
		decision.ControlVigenciaVersionRolHuellaSHA256, decision.CatalogoPoliticasHuellaSHA256,
		string(decision.GarantiaMinima),
	}
	if vinculo, err := decision.VinculoAutenticacionActor.Datos(); err == nil {
		valores = append(valores,
			vinculo.AutenticacionRef, vinculo.AutenticacionHuellaSHA256,
			vinculo.AsercionRef, vinculo.SesionRef, vinculo.ControlSesionRef,
			vinculo.ControlSesionHuellaSHA256, vinculo.CuentaRef, vinculo.CuentaOrdinariaRef,
			vinculo.PrincipalID, vinculo.PerfilActivoRef,
			string(vinculo.Superficie), string(vinculo.MetodoObservado), string(vinculo.GarantiaObservada),
			vinculo.PoliticaGarantiaRef, vinculo.PoliticaGarantiaHuellaSHA256,
			vinculo.ContextoActorRef, vinculo.ContextoActorHuellaSHA256,
		)
	} else {
		return true
	}
	valores = append(valores, decision.PoliticasEvaluadasRefs...)
	valores = append(valores, decision.PoliticasRefs...)
	valores = append(valores, decision.CamposPermitidos...)
	valores = append(valores, decision.Obligaciones...)
	for referencia, huella := range decision.PoliticasEvaluadasHuellasSHA256 {
		valores = append(valores, referencia, huella)
	}
	for referencia, huella := range decision.PoliticasHuellasSHA256 {
		valores = append(valores, referencia, huella)
	}
	for _, valor := range valores {
		if strings.ContainsRune(valor, '*') {
			return true
		}
	}
	return false
}

type escritorAtestacionAutorizacionV1 struct {
	buffer bytes.Buffer
	err    error
}

func nuevoEscritorAtestacionAutorizacionV1() *escritorAtestacionAutorizacionV1 {
	return &escritorAtestacionAutorizacionV1{}
}

func (e *escritorAtestacionAutorizacionV1) escribirByte(valor byte) {
	if e.err == nil {
		e.err = e.buffer.WriteByte(valor)
	}
}

func (e *escritorAtestacionAutorizacionV1) escribirBytes(valor []byte) {
	if e.err != nil {
		return
	}
	if uint64(len(valor)) > uint64(math.MaxUint32) ||
		len(valor) > TamanoMaximoMensajeAtestacionAutorizacionV1-e.buffer.Len() {
		e.err = errorMensajeAtestacionAutorizacionInvalido()
		return
	}
	_, e.err = e.buffer.Write(valor)
}

func (e *escritorAtestacionAutorizacionV1) escribirUint16(valor uint16) {
	if e.err != nil {
		return
	}
	var contenido [2]byte
	binary.BigEndian.PutUint16(contenido[:], valor)
	e.escribirBytes(contenido[:])
}

func (e *escritorAtestacionAutorizacionV1) escribirUint32(valor uint32) {
	if e.err != nil {
		return
	}
	var contenido [4]byte
	binary.BigEndian.PutUint32(contenido[:], valor)
	e.escribirBytes(contenido[:])
}

func (e *escritorAtestacionAutorizacionV1) escribirUint64(valor uint64) {
	if e.err != nil {
		return
	}
	var contenido [8]byte
	binary.BigEndian.PutUint64(contenido[:], valor)
	e.escribirBytes(contenido[:])
}

func (e *escritorAtestacionAutorizacionV1) escribirTexto(valor string) {
	if e.err != nil {
		return
	}
	if !utf8.ValidString(valor) || uint64(len(valor)) > uint64(math.MaxUint32) {
		e.err = errorMensajeAtestacionAutorizacionInvalido()
		return
	}
	e.escribirUint32(uint32(len(valor)))
	e.escribirBytes([]byte(valor))
}

func (e *escritorAtestacionAutorizacionV1) escribirBooleano(valor bool) {
	if valor {
		e.escribirByte(1)
		return
	}
	e.escribirByte(0)
}

func (e *escritorAtestacionAutorizacionV1) escribirLista(valores []string) {
	if e.err != nil {
		return
	}
	if uint64(len(valores)) > uint64(math.MaxUint32) {
		e.err = errorMensajeAtestacionAutorizacionInvalido()
		return
	}
	e.escribirUint32(uint32(len(valores)))
	for _, valor := range valores {
		e.escribirTexto(valor)
	}
}

func (e *escritorAtestacionAutorizacionV1) escribirMapa(valores map[string]string) {
	if e.err != nil {
		return
	}
	if uint64(len(valores)) > uint64(math.MaxUint32) {
		e.err = errorMensajeAtestacionAutorizacionInvalido()
		return
	}
	claves := make([]string, 0, len(valores))
	for clave := range valores {
		claves = append(claves, clave)
	}
	sort.Slice(claves, func(i, j int) bool {
		return bytes.Compare([]byte(claves[i]), []byte(claves[j])) < 0
	})
	e.escribirUint32(uint32(len(claves)))
	for _, clave := range claves {
		e.escribirTexto(clave)
		e.escribirTexto(valores[clave])
	}
}

func (e *escritorAtestacionAutorizacionV1) escribirInstante(instante time.Time) {
	// El dominio ya ha exigido UTC y precision de microsegundo. La conversion a
	// uint64 conserva en complemento a dos los instantes anteriores a Unix.
	e.escribirUint64(uint64(instante.UnixMicro()))
}

func errorMensajeAtestacionAutorizacionInvalido() error {
	return errors.Join(ErrDecisionAutorizacionInvalida, ErrMensajeAtestacionAutorizacionInvalido)
}
