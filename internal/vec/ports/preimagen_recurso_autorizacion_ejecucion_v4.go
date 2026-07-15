package ports

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida   = errors.New("vec: preimagen de recurso de autorizacion de ejecucion documental v4 invalida")
	ErrSerializacionGeneralPreimagenRecursoAutorizacionV4Prohibida = errors.New("vec: serializacion general de preimagen de recurso de autorizacion v4 prohibida")
)

const (
	esquemaPreimagenRecursoAutorizacionEjecucionDocumentalV4            = "vec.documentos.autorizacion-ejecucion.preimagen-recurso.v4"
	versionPreimagenRecursoAutorizacionEjecucionDocumentalV4     uint16 = 1
	marcaPreimagenRecursoAutorizacionEjecucionDocumentalV4              = "vec.preimagen-recurso-autorizacion-ejecucion.v4"
	maximoBytesPreimagenRecursoAutorizacionEjecucionDocumentalV4        = 2 * 1024 * 1024
	maximoElementosMapaPreimagenRecursoAutorizacionV4                   = 512
	maximoBytesClaveMapaPreimagenRecursoAutorizacionV4                  = 128
	maximoBytesValorMapaPreimagenRecursoAutorizacionV4                  = 512
)

// PreimagenRecursoAutorizacionEjecucionDocumentalV4 conserva la representacion
// completa que permite recomputar el contexto de recurso firmado por VEC-AD-1.
// Es opaca, no es una decision y nunca concede autoridad. Solo nace de una
// SolicitudAplicacion ya ligada o de la interpretacion estricta de sus bytes
// persistidos, que continua siendo material no confiable hasta cotejarlo con
// una firma PDP y una raiz historica autentica.
type PreimagenRecursoAutorizacionEjecucionDocumentalV4 struct {
	marca                 string
	recurso               domain.RecursoAutorizable
	huellaContextoSHA256  string
	huellaAmbitosSHA256   string
	serializacionCanonica []byte
	huellaSHA256          string
}

// PreimagenRecursoParaEvidenciaDurable extrae una copia defensiva desde la
// solicitud opaca viva. El llamador no puede aportar ni reemplazar el recurso.
func (s SolicitudAplicacionAutorizacionEjecucionDocumentalV4) PreimagenRecursoParaEvidenciaDurable() (
	PreimagenRecursoAutorizacionEjecucionDocumentalV4,
	error,
) {
	if s.validarEstructura() != nil || s.datos == nil || s.datos.vinculo.datos == nil {
		return PreimagenRecursoAutorizacionEjecucionDocumentalV4{},
			ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
	}
	datos := s.datos.vinculo.datos
	preimagen, err := nuevaPreimagenRecursoAutorizacionEjecucionDocumentalV4(datos.recurso)
	if err != nil || preimagen.huellaContextoSHA256 != datos.huellaRecursoSHA256 ||
		preimagen.huellaAmbitosSHA256 != datos.huellaAmbitosSHA256 ||
		preimagen.recurso.Atributos[AtributoAutorizacionDocumentalHuellaPlanSHA256] !=
			datos.huellaPlanSHA256 ||
		preimagen.recurso.Atributos[AtributoAutorizacionDocumentalEfectoRef] != datos.efectoRef {
		return PreimagenRecursoAutorizacionEjecucionDocumentalV4{},
			ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
	}
	return preimagen, nil
}

func nuevaPreimagenRecursoAutorizacionEjecucionDocumentalV4(
	recurso domain.RecursoAutorizable,
) (PreimagenRecursoAutorizacionEjecucionDocumentalV4, error) {
	recurso = clonarRecursoAutorizacionEjecucionDocumentalV4(recurso)
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || !recursoAptoParaPreimagenAutorizacionEjecucionV4(recurso) {
		return PreimagenRecursoAutorizacionEjecucionDocumentalV4{},
			ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
	}
	preimagen := PreimagenRecursoAutorizacionEjecucionDocumentalV4{
		marca:                marcaPreimagenRecursoAutorizacionEjecucionDocumentalV4,
		recurso:              recurso,
		huellaContextoSHA256: huellaContexto,
		huellaAmbitosSHA256: huellaMapaAutorizacionEjecucionDocumentalV4(
			"vec.documentos.autorizacion-ejecucion.ambitos.v4",
			recurso.Ambitos,
		),
	}
	preimagen.serializacionCanonica = preimagen.calcularSerializacionCanonica()
	preimagen.huellaSHA256 = huellaBytesFormatoDocumental(preimagen.serializacionCanonica)
	if preimagen.Validar() != nil {
		return PreimagenRecursoAutorizacionEjecucionDocumentalV4{},
			ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
	}
	return preimagen, nil
}

func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) Validar() error {
	huellaContexto, err := p.recurso.HuellaContextoAutorizacionSHA256()
	huellaAmbitos := huellaMapaAutorizacionEjecucionDocumentalV4(
		"vec.documentos.autorizacion-ejecucion.ambitos.v4",
		p.recurso.Ambitos,
	)
	if p.marca != marcaPreimagenRecursoAutorizacionEjecucionDocumentalV4 ||
		err != nil || !recursoAptoParaPreimagenAutorizacionEjecucionV4(p.recurso) ||
		!esSHA256Hexadecimal(p.huellaContextoSHA256) ||
		p.huellaContextoSHA256 != huellaContexto ||
		!esSHA256Hexadecimal(p.huellaAmbitosSHA256) ||
		p.huellaAmbitosSHA256 != huellaAmbitos ||
		len(p.serializacionCanonica) == 0 ||
		len(p.serializacionCanonica) > maximoBytesPreimagenRecursoAutorizacionEjecucionDocumentalV4 ||
		!bytes.Equal(p.serializacionCanonica, p.calcularSerializacionCanonica()) ||
		!esSHA256Hexadecimal(p.huellaSHA256) ||
		p.huellaSHA256 != huellaBytesFormatoDocumental(p.serializacionCanonica) {
		return ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
	}
	return nil
}

func recursoAptoParaPreimagenAutorizacionEjecucionV4(
	recurso domain.RecursoAutorizable,
) bool {
	return recurso.Validar() == nil && !contieneComodinRecursoAlmacen(recurso) &&
		len(recurso.Ambitos) > 0 && len(recurso.Atributos) == 2 &&
		esSHA256Hexadecimal(
			recurso.Atributos[AtributoAutorizacionDocumentalHuellaPlanSHA256],
		) && referenciaEjecucionDocumentalV3Valida(
		recurso.Atributos[AtributoAutorizacionDocumentalEfectoRef],
	)
}

// RecursoCanonico devuelve la preimagen completa mediante copia profunda. Los
// valores sensibles deben haber sido tokenizados/HMAC antes de crear el
// RecursoAutorizable original.
func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) RecursoCanonico() (
	domain.RecursoAutorizable,
	error,
) {
	if p.Validar() != nil {
		return domain.RecursoAutorizable{},
			ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
	}
	return clonarRecursoAutorizacionEjecucionDocumentalV4(p.recurso), nil
}

func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) HuellaContextoRecursoSHA256() (
	string,
	error,
) {
	if p.Validar() != nil {
		return "", ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
	}
	return p.huellaContextoSHA256, nil
}

func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) HuellaAmbitosSHA256() (
	string,
	error,
) {
	if p.Validar() != nil {
		return "", ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
	}
	return p.huellaAmbitosSHA256, nil
}

func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) SerializacionCanonicaParaPersistencia() (
	[]byte,
	error,
) {
	if p.Validar() != nil {
		return nil, ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
	}
	return append([]byte(nil), p.serializacionCanonica...), nil
}

func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) HuellaSHA256() (string, error) {
	if p.Validar() != nil {
		return "", ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
	}
	return p.huellaSHA256, nil
}

// InterpretarPreimagenRecursoAutorizacionEjecucionDocumentalV4 acepta solo el
// formato cerrado, limitado y canonico. El resultado sigue sin ser autoridad.
func InterpretarPreimagenRecursoAutorizacionEjecucionDocumentalV4(
	serializacion []byte,
	huellaEsperadaSHA256 string,
) (PreimagenRecursoAutorizacionEjecucionDocumentalV4, error) {
	if len(serializacion) == 0 ||
		len(serializacion) > maximoBytesPreimagenRecursoAutorizacionEjecucionDocumentalV4 ||
		!esSHA256Hexadecimal(huellaEsperadaSHA256) ||
		huellaBytesFormatoDocumental(serializacion) != huellaEsperadaSHA256 {
		return PreimagenRecursoAutorizacionEjecucionDocumentalV4{},
			ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
	}
	prefijo := append([]byte(esquemaPreimagenRecursoAutorizacionEjecucionDocumentalV4), 0)
	minimo := len(prefijo) + 2 + 3*8 + 2*4 + 8
	if len(serializacion) < minimo || !bytes.Equal(serializacion[:len(prefijo)], prefijo) ||
		binary.BigEndian.Uint16(serializacion[len(prefijo):len(prefijo)+2]) !=
			versionPreimagenRecursoAutorizacionEjecucionDocumentalV4 ||
		binary.BigEndian.Uint64(serializacion[len(serializacion)-8:]) != uint64(len(serializacion)) {
		return PreimagenRecursoAutorizacionEjecucionDocumentalV4{},
			ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
	}
	lector := lectorPreimagenRecursoAutorizacionV4{
		contenido: serializacion,
		posicion:  len(prefijo) + 2,
		fin:       len(serializacion) - 8,
	}
	recurso := domain.RecursoAutorizable{
		Referencia: lector.leerTexto(512),
		ModuloID:   lector.leerTexto(128),
		Tipo:       lector.leerTexto(128),
		Ambitos:    lector.leerMapa(),
		Atributos:  lector.leerMapa(),
	}
	if lector.err != nil || lector.posicion != lector.fin {
		return PreimagenRecursoAutorizacionEjecucionDocumentalV4{},
			ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
	}
	preimagen, err := nuevaPreimagenRecursoAutorizacionEjecucionDocumentalV4(recurso)
	if err != nil || preimagen.huellaSHA256 != huellaEsperadaSHA256 ||
		!bytes.Equal(preimagen.serializacionCanonica, serializacion) {
		return PreimagenRecursoAutorizacionEjecucionDocumentalV4{},
			ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
	}
	return preimagen, nil
}

type lectorPreimagenRecursoAutorizacionV4 struct {
	contenido []byte
	posicion  int
	fin       int
	err       error
}

func (l *lectorPreimagenRecursoAutorizacionV4) leerTexto(maximo uint64) string {
	if l.err != nil || l.posicion > l.fin-8 {
		l.err = ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
		return ""
	}
	longitud := binary.BigEndian.Uint64(l.contenido[l.posicion : l.posicion+8])
	l.posicion += 8
	if longitud == 0 || longitud > maximo || longitud > uint64(l.fin-l.posicion) {
		l.err = ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
		return ""
	}
	final := l.posicion + int(longitud)
	valor := string(l.contenido[l.posicion:final])
	l.posicion = final
	return valor
}

func (l *lectorPreimagenRecursoAutorizacionV4) leerMapa() map[string]string {
	if l.err != nil || l.posicion > l.fin-4 {
		l.err = ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
		return nil
	}
	cantidad := binary.BigEndian.Uint32(l.contenido[l.posicion : l.posicion+4])
	l.posicion += 4
	if cantidad > maximoElementosMapaPreimagenRecursoAutorizacionV4 {
		l.err = ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
		return nil
	}
	resultado := make(map[string]string, int(cantidad))
	anterior := ""
	for indice := uint32(0); indice < cantidad; indice++ {
		clave := l.leerTexto(maximoBytesClaveMapaPreimagenRecursoAutorizacionV4)
		valor := l.leerTexto(maximoBytesValorMapaPreimagenRecursoAutorizacionV4)
		if l.err != nil || (indice > 0 && clave <= anterior) {
			l.err = ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
			return nil
		}
		if _, duplicada := resultado[clave]; duplicada {
			l.err = ErrPreimagenRecursoAutorizacionEjecucionDocumentalV4Invalida
			return nil
		}
		resultado[clave] = valor
		anterior = clave
	}
	return resultado
}

func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) calcularSerializacionCanonica() []byte {
	var destino bytes.Buffer
	destino.WriteString(esquemaPreimagenRecursoAutorizacionEjecucionDocumentalV4)
	destino.WriteByte(0)
	var version [2]byte
	binary.BigEndian.PutUint16(version[:], versionPreimagenRecursoAutorizacionEjecucionDocumentalV4)
	destino.Write(version[:])
	escribirTextoPreimagenRecursoAutorizacionV4(&destino, p.recurso.Referencia)
	escribirTextoPreimagenRecursoAutorizacionV4(&destino, p.recurso.ModuloID)
	escribirTextoPreimagenRecursoAutorizacionV4(&destino, p.recurso.Tipo)
	escribirMapaPreimagenRecursoAutorizacionV4(&destino, p.recurso.Ambitos)
	escribirMapaPreimagenRecursoAutorizacionV4(&destino, p.recurso.Atributos)
	var longitud [8]byte
	binary.BigEndian.PutUint64(longitud[:], uint64(destino.Len()+len(longitud)))
	destino.Write(longitud[:])
	return destino.Bytes()
}

func escribirTextoPreimagenRecursoAutorizacionV4(destino *bytes.Buffer, valor string) {
	var longitud [8]byte
	binary.BigEndian.PutUint64(longitud[:], uint64(len(valor)))
	destino.Write(longitud[:])
	destino.WriteString(valor)
}

func escribirMapaPreimagenRecursoAutorizacionV4(
	destino *bytes.Buffer,
	valores map[string]string,
) {
	claves := make([]string, 0, len(valores))
	for clave := range valores {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	var cantidad [4]byte
	binary.BigEndian.PutUint32(cantidad[:], uint32(len(claves)))
	destino.Write(cantidad[:])
	for _, clave := range claves {
		escribirTextoPreimagenRecursoAutorizacionV4(destino, clave)
		escribirTextoPreimagenRecursoAutorizacionV4(destino, valores[clave])
	}
}

func (PreimagenRecursoAutorizacionEjecucionDocumentalV4) String() string {
	return "[PREIMAGEN-RECURSO-AUTORIZACION-EJECUCION-DOCUMENTAL-V4-REDACTADA]"
}
func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) GoString() string { return p.String() }
func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}
func (p PreimagenRecursoAutorizacionEjecucionDocumentalV4) LogValue() slog.Value {
	return slog.StringValue(p.String())
}
func (PreimagenRecursoAutorizacionEjecucionDocumentalV4) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionGeneralPreimagenRecursoAutorizacionV4Prohibida
}
func (*PreimagenRecursoAutorizacionEjecucionDocumentalV4) UnmarshalJSON([]byte) error {
	return ErrSerializacionGeneralPreimagenRecursoAutorizacionV4Prohibida
}
func (PreimagenRecursoAutorizacionEjecucionDocumentalV4) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionGeneralPreimagenRecursoAutorizacionV4Prohibida
}
func (*PreimagenRecursoAutorizacionEjecucionDocumentalV4) UnmarshalText([]byte) error {
	return ErrSerializacionGeneralPreimagenRecursoAutorizacionV4Prohibida
}
func (PreimagenRecursoAutorizacionEjecucionDocumentalV4) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionGeneralPreimagenRecursoAutorizacionV4Prohibida
}
func (*PreimagenRecursoAutorizacionEjecucionDocumentalV4) UnmarshalBinary([]byte) error {
	return ErrSerializacionGeneralPreimagenRecursoAutorizacionV4Prohibida
}
