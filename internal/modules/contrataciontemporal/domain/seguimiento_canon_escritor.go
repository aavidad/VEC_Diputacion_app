package domain

import (
	"bytes"
	"encoding/binary"
	"time"
)

type escritorCanonSeguimiento struct {
	destino *bytes.Buffer
	err     error
}

func nuevoEscritorCanonSeguimiento(
	destino *bytes.Buffer,
	dominio string,
) *escritorCanonSeguimiento {
	e := &escritorCanonSeguimiento{destino: destino}
	e.cadena(dominio)
	e.entero16(versionCanonSeguimientoV1)
	e.cadena(algoritmoCanonSeguimientoV1)
	return e
}

func (e *escritorCanonSeguimiento) transicionDefinida(
	t TransicionDefinidaSeguimiento,
) {
	e.clave(t.Clave)
	e.clave(t.Origen)
	e.clave(t.Destino)
	e.cadena(string(t.Clase))
	e.claves(t.MotivosPermitidos)
	e.booleano(t.MotivoObligatorio)
	e.entero32(uint32(len(t.Documentos)))
	for _, documento := range t.Documentos {
		e.clave(documento.TipoClave)
		e.booleano(documento.Obligatorio)
	}
	e.booleano(t.Calendario != nil)
	if t.Calendario != nil {
		e.claves(t.Calendario.AmbitosPermitidos)
		e.claves(t.Calendario.ResultadosPermitidos)
	}
	e.booleano(t.RequierePeriodo)
	e.cadena(string(t.EfectoPeriodo))
	e.booleano(t.ExigeActorDistinto)
}

func (e *escritorCanonSeguimiento) datosTransicion(
	d DatosTransicionSeguimiento,
) {
	e.cadena(d.ActuacionRef)
	e.clave(d.TransicionClave)
	e.claveOpcional(d.MotivoClave)
	e.cadena(d.ActorRef)
	e.cadena(d.UnidadRef)
	e.instante(d.EfectivoEn)
	e.instante(d.RegistradaEn)
	e.entero32(uint32(len(d.Documentos)))
	for _, documento := range d.Documentos {
		e.clave(documento.TipoClave)
		e.cadena(documento.Referencia)
	}
	e.booleano(d.Periodo != nil)
	if d.Periodo != nil {
		e.intervalo(*d.Periodo)
	}
	e.booleano(d.Calendario != nil)
	if d.Calendario != nil {
		e.cadena(d.Calendario.Referencia)
		e.entero64(d.Calendario.Version)
		e.huella(d.Calendario.HuellaSHA256)
		e.clave(d.Calendario.AmbitoTerritorialClave)
		e.clave(d.Calendario.ResultadoClave)
		e.instante(d.Calendario.CalculadoEn)
	}
	e.cadena(d.ReciboRef)
	e.cadena(d.CorrelacionRef)
	e.cadenaOpcional(d.RectificaActuacionRef)
}

func (e *escritorCanonSeguimiento) referenciaDefinicion(
	r ReferenciaDefinicionSeguimiento,
) {
	e.cadena(r.Referencia)
	e.entero64(r.Version)
	e.huella(r.HuellaSHA256)
}

func (e *escritorCanonSeguimiento) vigencia(v VigenciaSeguimiento) {
	e.instante(v.Desde)
	e.booleano(!v.Hasta.IsZero())
	if !v.Hasta.IsZero() {
		e.instante(v.Hasta)
	}
}

func (e *escritorCanonSeguimiento) intervalo(p IntervaloSeguimiento) {
	e.instante(p.Desde)
	e.instante(p.Hasta)
}

func (e *escritorCanonSeguimiento) claves(claves []ClaveCatalogo) {
	e.entero32(uint32(len(claves)))
	for _, clave := range claves {
		e.clave(clave)
	}
}

func (e *escritorCanonSeguimiento) clave(clave ClaveCatalogo) {
	e.cadena(string(clave))
}

func (e *escritorCanonSeguimiento) claveOpcional(clave ClaveCatalogo) {
	e.booleano(clave != "")
	if clave != "" {
		e.clave(clave)
	}
}

func (e *escritorCanonSeguimiento) cadenaOpcional(valor string) {
	e.booleano(valor != "")
	if valor != "" {
		e.cadena(valor)
	}
}

func (e *escritorCanonSeguimiento) huella(valor string) {
	e.cadena(valor)
}

func (e *escritorCanonSeguimiento) instante(valor time.Time) {
	e.entero64(uint64(valor.UnixMicro()))
}

func (e *escritorCanonSeguimiento) booleano(valor bool) {
	if valor {
		e.entero8(1)
		return
	}
	e.entero8(0)
}

func (e *escritorCanonSeguimiento) cadena(valor string) {
	e.bytes([]byte(valor))
}

func (e *escritorCanonSeguimiento) bytes(valor []byte) {
	if e.err != nil || uint64(len(valor)) > uint64(^uint32(0)) {
		e.err = ErrSeguimientoInvalido
		return
	}
	e.entero32(uint32(len(valor)))
	if e.err == nil {
		_, e.err = e.destino.Write(valor)
	}
}

func (e *escritorCanonSeguimiento) entero8(valor byte) {
	if e.err == nil {
		e.err = e.destino.WriteByte(valor)
	}
}

func (e *escritorCanonSeguimiento) entero16(valor uint16) {
	var datos [2]byte
	binary.BigEndian.PutUint16(datos[:], valor)
	e.escribir(datos[:])
}

func (e *escritorCanonSeguimiento) entero32(valor uint32) {
	var datos [4]byte
	binary.BigEndian.PutUint32(datos[:], valor)
	e.escribir(datos[:])
}

func (e *escritorCanonSeguimiento) entero64(valor uint64) {
	var datos [8]byte
	binary.BigEndian.PutUint64(datos[:], valor)
	e.escribir(datos[:])
}

func (e *escritorCanonSeguimiento) escribir(valor []byte) {
	if e.err == nil {
		_, e.err = e.destino.Write(valor)
	}
}
