package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"time"
)

const (
	dominioCanonCatalogoCoberturaV1   = "vec.dipgra.contratacion-temporal.catalogo-vias-cobertura"
	versionCanonCatalogoCoberturaV1   = uint16(1)
	algoritmoCanonCatalogoCoberturaV1 = "sha-256"
)

// CanonHuellaCatalogoCobertura identifica el dominio de separación, la versión
// de esquema y el algoritmo del resumen. Solo V1 está admitido en este corte.
type CanonHuellaCatalogoCobertura struct {
	Dominio        string `json:"dominio"`
	VersionEsquema uint16 `json:"version_esquema"`
	Algoritmo      string `json:"algoritmo"`
}

func CanonHuellaCatalogoCoberturaV1() CanonHuellaCatalogoCobertura {
	return CanonHuellaCatalogoCobertura{
		Dominio:        dominioCanonCatalogoCoberturaV1,
		VersionEsquema: versionCanonCatalogoCoberturaV1,
		Algoritmo:      algoritmoCanonCatalogoCoberturaV1,
	}
}

func (c CanonHuellaCatalogoCobertura) Valido() bool {
	return c == CanonHuellaCatalogoCoberturaV1()
}

func calcularHuellaCatalogo(
	publicacion PublicacionCatalogoViasCobertura,
) (string, error) {
	if !publicacion.Canon.Valido() {
		return "", ErrDatoInvalido
	}
	material, err := materialCanonicoCatalogoCoberturaV1(publicacion)
	if err != nil {
		return "", ErrDatoInvalido
	}
	suma := sha256.Sum256(material)
	return hex.EncodeToString(suma[:]), nil
}

func huellaCatalogoValida(valor string) bool {
	return patronHuella.MatchString(valor) &&
		valor != strings.Repeat("0", sha256.Size*2)
}

// materialCanonicoCatalogoCoberturaV1 usa cadenas UTF-8 precedidas por uint32
// y enteros big-endian. La secuencia es dominio, versión del esquema,
// algoritmo, referencia, versión funcional, publicación, vigencia,
// procedencia y vías ordenadas con sus comprobaciones ordenadas. No depende de
// etiquetas JSON, orden de campos de structs ni reglas omitempty. Hasta se
// codifica con un octeto de presencia y, si existe, Unix microsegundo. Ningún
// otro instante admite el valor cero.
func materialCanonicoCatalogoCoberturaV1(
	publicacion PublicacionCatalogoViasCobertura,
) ([]byte, error) {
	if !publicacion.Canon.Valido() ||
		!instanteCatalogoCoberturaValido(publicacion.PublicadoEn) ||
		publicacion.Vigencia.Validar() != nil {
		return nil, ErrDatoInvalido
	}
	var material bytes.Buffer
	escritor := escritorCanonCatalogo{destino: &material}
	escritor.cadena(publicacion.Canon.Dominio)
	escritor.entero16(publicacion.Canon.VersionEsquema)
	escritor.cadena(publicacion.Canon.Algoritmo)
	escritor.cadena(publicacion.Referencia)
	escritor.entero64(publicacion.Version)
	escritor.instante(publicacion.PublicadoEn)
	escritor.instante(publicacion.Vigencia.Desde)
	escritor.instanteOpcional(publicacion.Vigencia.Hasta)
	escritor.cadena(publicacion.ProcedenciaRef)
	escritor.entero32(uint32(len(publicacion.Vias)))
	for _, via := range publicacion.Vias {
		escritor.cadena(string(via.Clave))
		escritor.entero16(via.Orden)
		escritor.entero32(uint32(len(via.Comprobaciones)))
		for _, comprobacion := range via.Comprobaciones {
			escritor.cadena(string(comprobacion.Clave))
			escritor.entero16(comprobacion.Orden)
			escritor.booleano(comprobacion.Obligatoria)
			escritor.cadena(string(comprobacion.Procedencia.Clave))
			escritor.cadena(comprobacion.Procedencia.DefinicionFuenteRef)
		}
	}
	if escritor.err != nil {
		return nil, ErrDatoInvalido
	}
	return material.Bytes(), nil
}

type escritorCanonCatalogo struct {
	destino *bytes.Buffer
	err     error
}

func (e *escritorCanonCatalogo) cadena(valor string) {
	if e.err != nil || uint64(len(valor)) > uint64(^uint32(0)) {
		e.err = ErrDatoInvalido
		return
	}
	e.entero32(uint32(len(valor)))
	if e.err == nil {
		_, e.err = e.destino.WriteString(valor)
	}
}

func (e *escritorCanonCatalogo) instante(valor time.Time) {
	e.entero64(uint64(valor.UnixMicro()))
}

func (e *escritorCanonCatalogo) instanteOpcional(valor time.Time) {
	if valor.IsZero() {
		e.booleano(false)
		return
	}
	e.booleano(true)
	e.instante(valor)
}

func (e *escritorCanonCatalogo) booleano(valor bool) {
	if valor {
		e.entero8(1)
		return
	}
	e.entero8(0)
}

func (e *escritorCanonCatalogo) entero8(valor byte) {
	if e.err == nil {
		e.err = e.destino.WriteByte(valor)
	}
}

func (e *escritorCanonCatalogo) entero16(valor uint16) {
	var datos [2]byte
	binary.BigEndian.PutUint16(datos[:], valor)
	e.escribir(datos[:])
}

func (e *escritorCanonCatalogo) entero32(valor uint32) {
	var datos [4]byte
	binary.BigEndian.PutUint32(datos[:], valor)
	e.escribir(datos[:])
}

func (e *escritorCanonCatalogo) entero64(valor uint64) {
	var datos [8]byte
	binary.BigEndian.PutUint64(datos[:], valor)
	e.escribir(datos[:])
}

func (e *escritorCanonCatalogo) escribir(datos []byte) {
	if e.err == nil {
		_, e.err = e.destino.Write(datos)
	}
}
