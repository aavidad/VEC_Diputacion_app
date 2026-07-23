package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

const (
	dominioDefinicionSeguimientoV1 = "vec.dipgra.contratacion-temporal.seguimiento.definicion"
	dominioRaizSeguimientoV1       = "vec.dipgra.contratacion-temporal.seguimiento.raiz"
	dominioPeticionSeguimientoV1   = "vec.dipgra.contratacion-temporal.seguimiento.peticion"
	dominioActuacionSeguimientoV1  = "vec.dipgra.contratacion-temporal.seguimiento.actuacion"
	dominioEstadoSeguimientoV1     = "vec.dipgra.contratacion-temporal.seguimiento.estado"
	versionCanonSeguimientoV1      = uint16(1)
	algoritmoCanonSeguimientoV1    = "sha-256"
)

type CanonSeguimiento struct {
	Dominio        string `json:"dominio"`
	VersionEsquema uint16 `json:"version_esquema"`
	Algoritmo      string `json:"algoritmo"`
}

func CanonDefinicionSeguimientoV1() CanonSeguimiento {
	return nuevoCanonSeguimiento(dominioDefinicionSeguimientoV1)
}

// ValidarReintentoPublicacionDefinicionSeguimiento acepta únicamente una
// repetición exacta sobre la misma referencia y versión durable.
func ValidarReintentoPublicacionDefinicionSeguimiento(
	registrada ReferenciaDefinicionSeguimiento,
	propuesta ReferenciaDefinicionSeguimiento,
) error {
	if registrada.Validar() != nil || propuesta.Validar() != nil ||
		registrada.Referencia != propuesta.Referencia ||
		registrada.Version != propuesta.Version {
		return ErrDefinicionSeguimientoInvalida
	}
	if registrada.HuellaSHA256 != propuesta.HuellaSHA256 {
		return ErrPublicacionDefinicionSeguimientoEnConflicto
	}
	return nil
}

func (c CanonSeguimiento) EsDefinicionV1() bool { return c == CanonDefinicionSeguimientoV1() }

func nuevoCanonSeguimiento(dominio string) CanonSeguimiento {
	return CanonSeguimiento{
		Dominio: dominio, VersionEsquema: versionCanonSeguimientoV1,
		Algoritmo: algoritmoCanonSeguimientoV1,
	}
}

func calcularHuellaDefinicionSeguimiento(
	publicacion PublicacionDefinicionSeguimiento,
) (string, error) {
	material, err := materialCanonicoDefinicionSeguimiento(publicacion)
	if err != nil {
		return "", err
	}
	return resumenSeguimiento(material), nil
}

func calcularHuellaRaizSeguimiento(
	estado EstadoPersistidoSeguimiento,
) (string, error) {
	var material bytes.Buffer
	e := nuevoEscritorCanonSeguimiento(&material, dominioRaizSeguimientoV1)
	e.cadena(estado.Referencia)
	e.cadena(estado.OrganizacionRef)
	e.cadena(estado.ExpedienteRef)
	e.cadena(estado.RelacionRef)
	e.referenciaDefinicion(estado.Definicion)
	e.clave(estado.EstadoActual)
	e.intervalo(estado.PeriodoPrevisto)
	e.instante(estado.CreadoEn)
	if e.err != nil || !referenciaOpacaSeguimientoValida(estado.Referencia) ||
		!referenciaOpacaSeguimientoValida(estado.OrganizacionRef) ||
		!referenciaOpacaSeguimientoValida(estado.ExpedienteRef) ||
		!referenciaOpacaSeguimientoValida(estado.RelacionRef) ||
		estado.Definicion.Validar() != nil ||
		!estado.EstadoActual.Valida() ||
		estado.PeriodoPrevisto.Validar() != nil ||
		!instanteSeguimientoValido(estado.CreadoEn) {
		return "", ErrSeguimientoInvalido
	}
	return resumenSeguimiento(material.Bytes()), nil
}

func calcularHuellaPeticionSeguimiento(
	datos DatosTransicionSeguimiento,
) (string, error) {
	material, err := materialCanonicoPeticionSeguimiento(datos)
	if err != nil {
		return "", err
	}
	return resumenSeguimiento(material), nil
}

func calcularHuellaActuacionSeguimiento(
	actuacion ActuacionSeguimiento,
) (string, error) {
	material, err := materialCanonicoActuacionSeguimiento(actuacion)
	if err != nil {
		return "", err
	}
	return resumenSeguimiento(material), nil
}

// SerializarEstadoSeguimientoCanonico genera el formato binario cerrado V1.
// No usa JSON, reflexión, orden de mapas ni etiquetas de struct.
func SerializarEstadoSeguimientoCanonico(
	estado EstadoPersistidoSeguimiento,
) ([]byte, error) {
	return materialCanonicoEstadoSeguimiento(estado)
}

func materialCanonicoDefinicionSeguimiento(
	p PublicacionDefinicionSeguimiento,
) ([]byte, error) {
	if !p.Canon.EsDefinicionV1() ||
		!referenciaOpacaSeguimientoValida(p.Referencia) ||
		p.Version == 0 || !instanteSeguimientoValido(p.PublicadoEn) ||
		p.Vigencia.Validar() != nil || !p.EstadoInicial.Valida() ||
		len(p.Estados) > maximoEstadosSeguimiento ||
		len(p.Motivos) > maximoMotivosSeguimiento ||
		len(p.Transiciones) > maximoTransicionesSeguimiento {
		return nil, ErrDefinicionSeguimientoInvalida
	}
	var material bytes.Buffer
	e := nuevoEscritorCanonSeguimiento(&material, dominioDefinicionSeguimientoV1)
	e.cadena(p.Referencia)
	e.entero64(p.Version)
	e.instante(p.PublicadoEn)
	e.vigencia(p.Vigencia)
	e.clave(p.EstadoInicial)
	e.booleano(p.ProhibeCiclosSilenciosos)
	e.entero32(uint32(len(p.Estados)))
	for _, estado := range p.Estados {
		e.clave(estado.Clave)
		e.booleano(estado.Final)
	}
	e.claves(p.Motivos)
	e.entero32(uint32(len(p.Transiciones)))
	for _, transicion := range p.Transiciones {
		e.transicionDefinida(transicion)
	}
	if e.err != nil {
		return nil, ErrDefinicionSeguimientoInvalida
	}
	return material.Bytes(), nil
}

func materialCanonicoPeticionSeguimiento(
	d DatosTransicionSeguimiento,
) ([]byte, error) {
	if _, err := normalizarDatosTransicionSeguimiento(d); err != nil {
		return nil, ErrSeguimientoInvalido
	}
	var material bytes.Buffer
	e := nuevoEscritorCanonSeguimiento(&material, dominioPeticionSeguimientoV1)
	e.datosTransicion(d)
	if e.err != nil {
		return nil, ErrSeguimientoInvalido
	}
	return material.Bytes(), nil
}

func materialCanonicoActuacionSeguimiento(
	a ActuacionSeguimiento,
) ([]byte, error) {
	if a.Secuencia == 0 || a.VersionSeguimiento != a.Secuencia ||
		a.Definicion.Validar() != nil || !a.Clase.valida() ||
		!a.EstadoOrigen.Valida() || !a.EstadoDestino.Valida() ||
		!huellaSeguimientoValida(a.HuellaPeticionSHA256) ||
		!huellaSeguimientoValida(a.HuellaAnteriorSHA256) {
		return nil, ErrSeguimientoInvalido
	}
	if _, err := normalizarDatosTransicionSeguimiento(a.datos()); err != nil {
		return nil, ErrSeguimientoInvalido
	}
	var material bytes.Buffer
	e := nuevoEscritorCanonSeguimiento(&material, dominioActuacionSeguimientoV1)
	e.entero64(a.Secuencia)
	e.entero64(a.VersionSeguimiento)
	e.referenciaDefinicion(a.Definicion)
	e.cadena(string(a.Clase))
	e.clave(a.EstadoOrigen)
	e.clave(a.EstadoDestino)
	e.datosTransicion(a.datos())
	e.huella(a.HuellaPeticionSHA256)
	e.huella(a.HuellaAnteriorSHA256)
	if e.err != nil {
		return nil, ErrSeguimientoInvalido
	}
	return material.Bytes(), nil
}

func materialCanonicoEstadoSeguimiento(
	estado EstadoPersistidoSeguimiento,
) ([]byte, error) {
	if validarEstadoCanonicoSeguimiento(estado) != nil {
		return nil, ErrSeguimientoInvalido
	}
	var material bytes.Buffer
	e := nuevoEscritorCanonSeguimiento(&material, dominioEstadoSeguimientoV1)
	e.cadena(estado.Referencia)
	e.cadena(estado.OrganizacionRef)
	e.cadena(estado.ExpedienteRef)
	e.cadena(estado.RelacionRef)
	e.referenciaDefinicion(estado.Definicion)
	e.entero64(estado.Version)
	e.clave(estado.EstadoActual)
	e.intervalo(estado.PeriodoPrevisto)
	e.instante(estado.CreadoEn)
	e.instante(estado.ActualizadoEn)
	e.huella(estado.HuellaRaizSHA256)
	e.entero32(uint32(len(estado.PeriodosResultantes)))
	for _, periodo := range estado.PeriodosResultantes {
		e.intervalo(periodo.Intervalo)
		e.cadena(periodo.ActuacionRef)
	}
	e.booleano(estado.CeseEfectivo != nil)
	if estado.CeseEfectivo != nil {
		e.instante(estado.CeseEfectivo.EfectivoEn)
		e.cadena(estado.CeseEfectivo.ActuacionRef)
	}
	e.entero32(uint32(len(estado.Actuaciones)))
	for _, actuacion := range estado.Actuaciones {
		materialActuacion, err := materialCanonicoActuacionSeguimiento(actuacion)
		if err != nil {
			return nil, ErrSeguimientoInvalido
		}
		e.bytes(materialActuacion)
		e.huella(actuacion.HuellaActuacionSHA256)
	}
	if e.err != nil {
		return nil, ErrSeguimientoInvalido
	}
	return material.Bytes(), nil
}

func validarEstadoCanonicoSeguimiento(
	estado EstadoPersistidoSeguimiento,
) error {
	if estado.Version != uint64(len(estado.Actuaciones)) ||
		len(estado.Actuaciones) > maximoActuacionesSeguimiento ||
		len(estado.PeriodosResultantes) > maximoActuacionesSeguimiento ||
		len(estado.PeriodosResultantes) > len(estado.Actuaciones) ||
		!referenciaOpacaSeguimientoValida(estado.Referencia) ||
		!referenciaOpacaSeguimientoValida(estado.OrganizacionRef) ||
		!referenciaOpacaSeguimientoValida(estado.ExpedienteRef) ||
		!referenciaOpacaSeguimientoValida(estado.RelacionRef) ||
		estado.Definicion.Validar() != nil || !estado.EstadoActual.Valida() ||
		estado.PeriodoPrevisto.Validar() != nil ||
		!instanteSeguimientoValido(estado.CreadoEn) ||
		!instanteSeguimientoValido(estado.ActualizadoEn) ||
		estado.ActualizadoEn.Before(estado.CreadoEn) ||
		!huellaSeguimientoValida(estado.HuellaRaizSHA256) {
		return ErrSeguimientoInvalido
	}
	if len(estado.Actuaciones) == 0 {
		if !estado.ActualizadoEn.Equal(estado.CreadoEn) ||
			len(estado.PeriodosResultantes) != 0 ||
			estado.CeseEfectivo != nil {
			return ErrSeguimientoInvalido
		}
		return nil
	}
	ultima := estado.Actuaciones[len(estado.Actuaciones)-1]
	if estado.EstadoActual != ultima.EstadoDestino ||
		!estado.ActualizadoEn.Equal(ultima.RegistradaEn) {
		return ErrSeguimientoInvalido
	}
	referencias := make(map[string]struct{}, len(estado.Actuaciones))
	anterior := estado.HuellaRaizSHA256
	for indice, actuacion := range estado.Actuaciones {
		if actuacion.Secuencia != uint64(indice+1) ||
			actuacion.VersionSeguimiento != uint64(indice+1) ||
			!actuacion.Definicion.Coincide(estado.Definicion) ||
			actuacion.HuellaAnteriorSHA256 != anterior ||
			!huellaSeguimientoValida(actuacion.HuellaActuacionSHA256) {
			return ErrSeguimientoInvalido
		}
		if _, repetida := referencias[actuacion.ActuacionRef]; repetida {
			return ErrSeguimientoInvalido
		}
		referencias[actuacion.ActuacionRef] = struct{}{}
		normalizados, err := normalizarDatosTransicionSeguimiento(actuacion.datos())
		if err != nil {
			return ErrSeguimientoInvalido
		}
		huellaPeticion, errPeticion := calcularHuellaPeticionSeguimiento(normalizados)
		huellaActuacion, errActuacion := calcularHuellaActuacionSeguimiento(actuacion)
		if errPeticion != nil || errActuacion != nil ||
			huellaPeticion != actuacion.HuellaPeticionSHA256 ||
			huellaActuacion != actuacion.HuellaActuacionSHA256 {
			return ErrSeguimientoInvalido
		}
		anterior = actuacion.HuellaActuacionSHA256
	}
	for indice, periodo := range estado.PeriodosResultantes {
		if periodo.Intervalo.Validar() != nil ||
			!referenciaOpacaSeguimientoValida(periodo.ActuacionRef) {
			return ErrSeguimientoInvalido
		}
		if _, existe := referencias[periodo.ActuacionRef]; !existe {
			return ErrSeguimientoInvalido
		}
		if indice > 0 && periodo.Intervalo.Desde.Before(
			estado.PeriodosResultantes[indice-1].Intervalo.Hasta,
		) {
			return ErrSeguimientoInvalido
		}
	}
	if estado.CeseEfectivo != nil {
		if len(estado.PeriodosResultantes) == 0 ||
			!instanteSeguimientoValido(estado.CeseEfectivo.EfectivoEn) ||
			estado.CeseEfectivo.EfectivoEn.Before(
				estado.PeriodosResultantes[0].Intervalo.Desde,
			) ||
			!referenciaOpacaSeguimientoValida(estado.CeseEfectivo.ActuacionRef) {
			return ErrSeguimientoInvalido
		}
		if _, existe := referencias[estado.CeseEfectivo.ActuacionRef]; !existe {
			return ErrSeguimientoInvalido
		}
	}
	return nil
}

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

func resumenSeguimiento(material []byte) string {
	suma := sha256.Sum256(material)
	return hex.EncodeToString(suma[:])
}

func huellaSeguimientoValida(valor string) bool {
	return patronHuella.MatchString(valor) &&
		valor != strings.Repeat("0", sha256.Size*2)
}

func referenciaOpacaSeguimientoValida(valor string) bool {
	return len(valor) == len("ref:")+sha256.Size*2 &&
		strings.HasPrefix(valor, "ref:") &&
		huellaSeguimientoValida(valor[len("ref:"):])
}

func huellaAnteriorSeguimiento(estado EstadoPersistidoSeguimiento) string {
	if len(estado.Actuaciones) == 0 {
		return estado.HuellaRaizSHA256
	}
	return estado.Actuaciones[len(estado.Actuaciones)-1].HuellaActuacionSHA256
}

func normalizarDefinicionSeguimiento(
	b BorradorDefinicionSeguimiento,
) (BorradorDefinicionSeguimiento, error) {
	if !referenciaOpacaSeguimientoValida(b.Referencia) || b.Version == 0 ||
		!instanteSeguimientoValido(b.PublicadoEn) || b.Vigencia.Validar() != nil ||
		b.PublicadoEn.After(b.Vigencia.Desde) || !b.EstadoInicial.Valida() ||
		len(b.Estados) < 2 || len(b.Estados) > maximoEstadosSeguimiento ||
		len(b.Motivos) > maximoMotivosSeguimiento ||
		len(b.Transiciones) == 0 ||
		len(b.Transiciones) > maximoTransicionesSeguimiento {
		return BorradorDefinicionSeguimiento{}, ErrDefinicionSeguimientoInvalida
	}
	n := b
	n.Estados = append([]EstadoDefinidoSeguimiento(nil), b.Estados...)
	n.Motivos = append([]ClaveCatalogo(nil), b.Motivos...)
	n.Transiciones = clonarTransicionesSeguimiento(b.Transiciones)
	sort.Slice(n.Estados, func(i, j int) bool { return n.Estados[i].Clave < n.Estados[j].Clave })
	sort.Slice(n.Motivos, func(i, j int) bool { return n.Motivos[i] < n.Motivos[j] })
	sort.Slice(n.Transiciones, func(i, j int) bool {
		return n.Transiciones[i].Clave < n.Transiciones[j].Clave
	})
	estados, finales := make(map[ClaveCatalogo]bool, len(n.Estados)), 0
	for _, estado := range n.Estados {
		if !estado.Clave.Valida() {
			return BorradorDefinicionSeguimiento{}, ErrDefinicionSeguimientoInvalida
		}
		if _, repetido := estados[estado.Clave]; repetido {
			return BorradorDefinicionSeguimiento{}, ErrDefinicionSeguimientoInvalida
		}
		estados[estado.Clave] = estado.Final
		if estado.Final {
			finales++
		}
	}
	if _, existe := estados[n.EstadoInicial]; !existe || estados[n.EstadoInicial] ||
		finales == 0 || !clavesSeguimientoUnicasValidas(n.Motivos, maximoMotivosSeguimiento) {
		return BorradorDefinicionSeguimiento{}, ErrDefinicionSeguimientoInvalida
	}
	motivos := conjuntoClavesSeguimiento(n.Motivos)
	clavesTransicion := make(map[ClaveCatalogo]struct{}, len(n.Transiciones))
	for indice := range n.Transiciones {
		t := &n.Transiciones[indice]
		if normalizarTransicionSeguimiento(t, estados, motivos) != nil {
			return BorradorDefinicionSeguimiento{}, ErrDefinicionSeguimientoInvalida
		}
		if _, repetida := clavesTransicion[t.Clave]; repetida {
			return BorradorDefinicionSeguimiento{}, ErrDefinicionSeguimientoInvalida
		}
		clavesTransicion[t.Clave] = struct{}{}
	}
	if n.ProhibeCiclosSilenciosos && tieneCicloSilencioso(n.Transiciones) {
		return BorradorDefinicionSeguimiento{}, ErrDefinicionSeguimientoInvalida
	}
	return n, nil
}

func normalizarTransicionSeguimiento(
	t *TransicionDefinidaSeguimiento,
	estados map[ClaveCatalogo]bool,
	motivos map[ClaveCatalogo]struct{},
) error {
	origenFinal, existeOrigen := estados[t.Origen]
	destinoFinal, existeDestino := estados[t.Destino]
	if !t.Clave.Valida() || !existeOrigen || !existeDestino || !t.Clase.valida() ||
		!t.EfectoPeriodo.valido() ||
		len(t.Documentos) > maximoDocumentosPorTransicion ||
		len(t.MotivosPermitidos) > maximoMotivosSeguimiento {
		return ErrDefinicionSeguimientoInvalida
	}
	if origenFinal && t.Clase == TransicionOrdinaria ||
		origenFinal && t.Clase == TransicionRectificacion && !destinoFinal ||
		!efectoCompatibleConClaseSeguimiento(*t, origenFinal, destinoFinal) ||
		t.Clase != TransicionRectificacion && t.ExigeActorDistinto ||
		(t.EfectoPeriodo == EfectoPeriodoAbrir ||
			t.EfectoPeriodo == EfectoPeriodoAmpliar ||
			t.EfectoPeriodo == EfectoPeriodoRectificarTramo) !=
			t.RequierePeriodo {
		return ErrDefinicionSeguimientoInvalida
	}
	sort.Slice(t.MotivosPermitidos, func(i, j int) bool {
		return t.MotivosPermitidos[i] < t.MotivosPermitidos[j]
	})
	if !clavesSeguimientoUnicasValidas(t.MotivosPermitidos, maximoMotivosSeguimiento) ||
		t.MotivoObligatorio && len(t.MotivosPermitidos) == 0 {
		return ErrDefinicionSeguimientoInvalida
	}
	for _, motivo := range t.MotivosPermitidos {
		if _, existe := motivos[motivo]; !existe {
			return ErrDefinicionSeguimientoInvalida
		}
	}
	sort.Slice(t.Documentos, func(i, j int) bool {
		return t.Documentos[i].TipoClave < t.Documentos[j].TipoClave
	})
	for indice, requisito := range t.Documentos {
		if !requisito.TipoClave.Valida() ||
			indice > 0 && t.Documentos[indice-1].TipoClave == requisito.TipoClave {
			return ErrDefinicionSeguimientoInvalida
		}
	}
	if t.Calendario != nil && normalizarRequisitoCalendario(t.Calendario) != nil {
		return ErrDefinicionSeguimientoInvalida
	}
	return nil
}

func efectoCompatibleConClaseSeguimiento(
	t TransicionDefinidaSeguimiento,
	origenFinal bool,
	destinoFinal bool,
) bool {
	switch t.Clase {
	case TransicionOrdinaria:
		if t.EfectoPeriodo == EfectoPeriodoCerrar {
			return destinoFinal
		}
		return t.EfectoPeriodo == EfectoPeriodoNinguno ||
			t.EfectoPeriodo == EfectoPeriodoAbrir ||
			t.EfectoPeriodo == EfectoPeriodoAmpliar
	case TransicionRectificacion:
		if t.EfectoPeriodo == EfectoPeriodoRectificarCese {
			return origenFinal && destinoFinal
		}
		return t.EfectoPeriodo == EfectoPeriodoNinguno ||
			t.EfectoPeriodo == EfectoPeriodoRectificarTramo
	case TransicionReapertura:
		return origenFinal && !destinoFinal &&
			t.EfectoPeriodo == EfectoPeriodoReabrir
	default:
		return false
	}
}

func normalizarRequisitoCalendario(c *RequisitoCalendarioSeguimiento) error {
	sort.Slice(c.AmbitosPermitidos, func(i, j int) bool {
		return c.AmbitosPermitidos[i] < c.AmbitosPermitidos[j]
	})
	sort.Slice(c.ResultadosPermitidos, func(i, j int) bool {
		return c.ResultadosPermitidos[i] < c.ResultadosPermitidos[j]
	})
	if len(c.AmbitosPermitidos) == 0 || len(c.ResultadosPermitidos) == 0 ||
		!clavesSeguimientoUnicasValidas(
			c.AmbitosPermitidos,
			maximoResultadosCalendarioPorTransicion,
		) ||
		!clavesSeguimientoUnicasValidas(
			c.ResultadosPermitidos,
			maximoResultadosCalendarioPorTransicion,
		) {
		return ErrDefinicionSeguimientoInvalida
	}
	return nil
}

func tieneCicloSilencioso(transiciones []TransicionDefinidaSeguimiento) bool {
	adyacencia := make(map[ClaveCatalogo][]ClaveCatalogo)
	for _, t := range transiciones {
		if t.Clase == TransicionOrdinaria && !t.MotivoObligatorio &&
			!tieneDocumentoObligatorioSeguimiento(t.Documentos) &&
			t.Calendario == nil &&
			!t.RequierePeriodo && t.EfectoPeriodo == EfectoPeriodoNinguno {
			adyacencia[t.Origen] = append(adyacencia[t.Origen], t.Destino)
		}
	}
	visita := make(map[ClaveCatalogo]uint8)
	var recorrer func(ClaveCatalogo) bool
	recorrer = func(estado ClaveCatalogo) bool {
		if visita[estado] == 1 {
			return true
		}
		if visita[estado] == 2 {
			return false
		}
		visita[estado] = 1
		for _, destino := range adyacencia[estado] {
			if recorrer(destino) {
				return true
			}
		}
		visita[estado] = 2
		return false
	}
	for origen := range adyacencia {
		if recorrer(origen) {
			return true
		}
	}
	return false
}

func tieneDocumentoObligatorioSeguimiento(
	documentos []RequisitoDocumentoSeguimiento,
) bool {
	for _, documento := range documentos {
		if documento.Obligatorio {
			return true
		}
	}
	return false
}

func clonarTransicionesSeguimiento(
	transiciones []TransicionDefinidaSeguimiento,
) []TransicionDefinidaSeguimiento {
	clon := make([]TransicionDefinidaSeguimiento, len(transiciones))
	for indice, transicion := range transiciones {
		clon[indice] = transicion.clonar()
	}
	return clon
}

func clonarActuacionesSeguimiento(
	actuaciones []ActuacionSeguimiento,
) []ActuacionSeguimiento {
	clon := make([]ActuacionSeguimiento, len(actuaciones))
	for indice, actuacion := range actuaciones {
		clon[indice] = actuacion.clonar()
	}
	return clon
}

func clavesSeguimientoUnicasValidas(
	claves []ClaveCatalogo,
	maximo int,
) bool {
	if len(claves) > maximo {
		return false
	}
	vistas := make(map[ClaveCatalogo]struct{}, len(claves))
	for _, clave := range claves {
		if !clave.Valida() {
			return false
		}
		if _, repetida := vistas[clave]; repetida {
			return false
		}
		vistas[clave] = struct{}{}
	}
	return true
}

func conjuntoClavesSeguimiento(claves []ClaveCatalogo) map[ClaveCatalogo]struct{} {
	conjunto := make(map[ClaveCatalogo]struct{}, len(claves))
	for _, clave := range claves {
		conjunto[clave] = struct{}{}
	}
	return conjunto
}

func contieneClaveSeguimiento(claves []ClaveCatalogo, buscada ClaveCatalogo) bool {
	indice := sort.Search(len(claves), func(i int) bool {
		return claves[i] >= buscada
	})
	return indice < len(claves) && claves[indice] == buscada
}

func instanteSeguimientoValido(instante time.Time) bool {
	return instanteCanonico(instante) && instante.Year() >= 1 && instante.Year() <= 9999
}
