package ports

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	// ErrReciboEscrituraObjetoMaterialV2NoValido es deliberadamente opaco:
	// un dato ausente, un cruce de contexto y una prueba criptografica falsa
	// producen la misma denegacion cerrada.
	ErrReciboEscrituraObjetoMaterialV2NoValido = errors.New(
		"vec: recibo material v2 de escritura no valido",
	)
	ErrAtestacionMaterialAlmacenV2NoValida = errors.New(
		"vec: atestacion material v2 de almacen no valida",
	)
	ErrSerializacionMaterialAlmacenV2Prohibida = errors.New(
		"vec: serializacion generica de material v2 de almacen prohibida",
	)
)

const (
	EsquemaPerfilCapacidadesAlmacenMaterialV2 = "vec.almacen.perfil-capacidades-material.v2"
	EsquemaInstantaneaObjetoMaterialV2        = "vec.almacen.instantanea-objeto-material.v2"
	EsquemaReciboEscrituraObjetoMaterialV2    = "vec.almacen.recibo-escritura-material.v2"

	VersionEsquemaMaterialAlmacenV2 uint16 = 2

	dominioAtestacionPerfilCapacidadesMaterialV2 = "perfil-capacidades-almacen-material-v2"
	dominioAtestacionReciboEscrituraMaterialV2   = "recibo-escritura-objeto-material-v2"

	tamanoMaximoAtestacionMaterialAlmacenV2 = 16 * 1024
)

// EstadoInmovilizacionObjetoMaterialV2 evita representar el bloqueo legal con
// un booleano ambiguo. La ausencia y cualquier estado desconocido son
// invalidos.
type EstadoInmovilizacionObjetoMaterialV2 string

const (
	EstadoInmovilizacionMaterialNoAplicada EstadoInmovilizacionObjetoMaterialV2 = "no_inmovilizado"
	EstadoInmovilizacionMaterialAplicada   EstadoInmovilizacionObjetoMaterialV2 = "inmovilizado"
)

func (e EstadoInmovilizacionObjetoMaterialV2) valida() bool {
	return e == EstadoInmovilizacionMaterialNoAplicada ||
		e == EstadoInmovilizacionMaterialAplicada
}

// EstadoObjetoMaterialV2 es una lista positiva cerrada. Este primer corte solo
// emite recibos para objetos activos y no eliminados.
type EstadoObjetoMaterialV2 string

const EstadoObjetoMaterialActivo EstadoObjetoMaterialV2 = "activo"

func (e EstadoObjetoMaterialV2) valida() bool { return e == EstadoObjetoMaterialActivo }

// AlgoritmoAtestacionMaterialAlmacenV2 identifica el formato que el
// verificador debe comprobar. La forma valida nunca sustituye la verificacion.
type AlgoritmoAtestacionMaterialAlmacenV2 string

const (
	AlgoritmoAtestacionMaterialHMACSHA256 AlgoritmoAtestacionMaterialAlmacenV2 = "hmac-sha-256"
	AlgoritmoAtestacionMaterialCOSESign1  AlgoritmoAtestacionMaterialAlmacenV2 = "cose-sign1"
)

func (a AlgoritmoAtestacionMaterialAlmacenV2) valida() bool {
	return a == AlgoritmoAtestacionMaterialHMACSHA256 ||
		a == AlgoritmoAtestacionMaterialCOSESign1
}

// SeleccionPlanMaterialAlmacenV2 identifica un plan publicado. No contiene su
// huella: esta solo puede proceder del verificador autoritativo del registro.
type SeleccionPlanMaterialAlmacenV2 struct {
	referencia string
	version    uint32
}

func NuevaSeleccionPlanMaterialAlmacenV2(
	referencia string,
	version uint32,
) (SeleccionPlanMaterialAlmacenV2, error) {
	seleccion := SeleccionPlanMaterialAlmacenV2{referencia: referencia, version: version}
	if !seleccion.valida() {
		return SeleccionPlanMaterialAlmacenV2{}, errorReciboMaterialV2()
	}
	return seleccion, nil
}

func (s SeleccionPlanMaterialAlmacenV2) valida() bool {
	return aliasLogicoMaterialV2Valido(s.referencia, 512) && s.version > 0
}

// HuellaPlanMaterialAlmacenV2 es una capacidad opaca creada exclusivamente
// tras cotejar el plan publicado con todos los hechos estables del recibo.
type HuellaPlanMaterialAlmacenV2 struct {
	referencia    string
	version       uint32
	suma          [sha256.Size]byte
	huellaVinculo [sha256.Size]byte
}

func (h HuellaPlanMaterialAlmacenV2) valida() bool {
	return aliasLogicoMaterialV2Valido(h.referencia, 512) && h.version > 0 &&
		h.suma != ([sha256.Size]byte{}) && h.huellaVinculo != ([sha256.Size]byte{})
}

func (h HuellaPlanMaterialAlmacenV2) Bytes() ([]byte, error) {
	if !h.valida() {
		return nil, errorReciboMaterialV2()
	}
	return append([]byte(nil), h.suma[:]...), nil
}

// SolicitudVerificarPlanMaterialAlmacenV2 liga el plan a los hechos que el
// registro debe reconocer como estables. Operacion, carga y efecto nunca se
// aceptan solo porque aparezcan en el contexto V1.
type SolicitudVerificarPlanMaterialAlmacenV2 struct {
	seleccion        SeleccionPlanMaterialAlmacenV2
	conectorLogicoID string
	moduloID         string
	accionNegocio    string
	accionTecnica    string
	recursoRef       string
	operacionRef     string
	cargaRef         string
	efectoRef        string
	clasificacion    string
	huellaVinculo    [sha256.Size]byte
}

func nuevaSolicitudVerificarPlanMaterialAlmacenV2(
	seleccion SeleccionPlanMaterialAlmacenV2,
	conectorLogicoID string,
	proyeccion ProyeccionContextoOperacionAlmacen,
) (SolicitudVerificarPlanMaterialAlmacenV2, error) {
	solicitud := SolicitudVerificarPlanMaterialAlmacenV2{
		seleccion: seleccion, conectorLogicoID: conectorLogicoID,
		moduloID: proyeccion.ModuloID, accionNegocio: proyeccion.AccionNegocio,
		accionTecnica: proyeccion.AccionTecnica, recursoRef: proyeccion.RecursoRef,
		operacionRef: proyeccion.OperacionRef, cargaRef: proyeccion.CargaRef,
		efectoRef: proyeccion.EfectoRef, clasificacion: proyeccion.Clasificacion,
	}
	canonico, err := solicitud.canonicoSinHuella()
	if err != nil {
		return SolicitudVerificarPlanMaterialAlmacenV2{}, errorReciboMaterialV2()
	}
	solicitud.huellaVinculo = sha256.Sum256(canonico)
	return solicitud, nil
}

func (s SolicitudVerificarPlanMaterialAlmacenV2) canonicoSinHuella() ([]byte, error) {
	if !s.seleccion.valida() || !aliasLogicoMaterialV2Valido(s.conectorLogicoID, 128) ||
		!aliasLogicoMaterialV2Valido(s.moduloID, 128) ||
		!aliasLogicoMaterialV2Valido(s.accionNegocio, 256) ||
		s.accionTecnica != AccionAlmacenEscribir ||
		!aliasLogicoMaterialV2Valido(s.recursoRef, 512) ||
		!aliasLogicoMaterialV2Valido(s.operacionRef, 512) ||
		!aliasLogicoMaterialV2Valido(s.cargaRef, 512) ||
		!aliasLogicoMaterialV2Valido(s.efectoRef, 512) ||
		!aliasLogicoMaterialV2Valido(s.clasificacion, 256) {
		return nil, errorReciboMaterialV2()
	}
	var canonico []byte
	canonico = anexarTLVMaterialV2(canonico, 0, []byte("vec.almacen.vinculo-plan-material.v2"))
	canonico = anexarTLVMaterialV2(canonico, 1, uint16MaterialV2(VersionEsquemaMaterialAlmacenV2))
	canonico = anexarTLVMaterialV2(canonico, 2, []byte(s.seleccion.referencia))
	canonico = anexarTLVMaterialV2(canonico, 3, uint32MaterialV2(s.seleccion.version))
	canonico = anexarTLVMaterialV2(canonico, 4, []byte(s.conectorLogicoID))
	canonico = anexarTLVMaterialV2(canonico, 5, []byte(s.moduloID))
	canonico = anexarTLVMaterialV2(canonico, 6, []byte(s.accionNegocio))
	canonico = anexarTLVMaterialV2(canonico, 7, []byte(s.accionTecnica))
	canonico = anexarTLVMaterialV2(canonico, 8, []byte(s.recursoRef))
	canonico = anexarTLVMaterialV2(canonico, 9, []byte(s.operacionRef))
	canonico = anexarTLVMaterialV2(canonico, 10, []byte(s.cargaRef))
	canonico = anexarTLVMaterialV2(canonico, 11, []byte(s.efectoRef))
	canonico = anexarTLVMaterialV2(canonico, 12, []byte(s.clasificacion))
	return canonico, nil
}

// RevelarParaVerificacionPlanMaterial entrega una copia estable al registro
// de planes. El resultado debe quedar ligado a HuellaVinculo.
func (s SolicitudVerificarPlanMaterialAlmacenV2) RevelarParaVerificacionPlanMaterial() (
	referencia string,
	version uint32,
	conectorLogicoID, moduloID, accionNegocio, accionTecnica, recursoRef string,
	operacionRef, cargaRef, efectoRef, clasificacion string,
	huellaVinculo [sha256.Size]byte,
	err error,
) {
	canonico, err := s.canonicoSinHuella()
	esperada := sha256.Sum256(canonico)
	if err != nil || s.huellaVinculo == ([sha256.Size]byte{}) ||
		subtle.ConstantTimeCompare(s.huellaVinculo[:], esperada[:]) != 1 {
		return "", 0, "", "", "", "", "", "", "", "", "",
			[sha256.Size]byte{}, errorReciboMaterialV2()
	}
	return s.seleccion.referencia, s.seleccion.version, s.conectorLogicoID,
		s.moduloID, s.accionNegocio, s.accionTecnica, s.recursoRef,
		s.operacionRef, s.cargaRef, s.efectoRef, s.clasificacion, s.huellaVinculo, nil
}

type ResultadoVerificacionPlanMaterialAlmacenV2 struct {
	huellaPlan    [sha256.Size]byte
	huellaVinculo [sha256.Size]byte
}

func NuevoResultadoVerificacionPlanMaterialAlmacenV2(
	solicitud SolicitudVerificarPlanMaterialAlmacenV2,
	huellaPlanHexadecimal string,
) (ResultadoVerificacionPlanMaterialAlmacenV2, error) {
	huellaPlan, err := decodificarSHA256MaterialV2(huellaPlanHexadecimal)
	if _, _, _, _, _, _, _, _, _, _, _, vinculo, errSolicitud :=
		solicitud.RevelarParaVerificacionPlanMaterial(); err != nil || errSolicitud != nil || huellasMaterialV2Iguales(huellaPlan, vinculo) {
		return ResultadoVerificacionPlanMaterialAlmacenV2{}, errorReciboMaterialV2()
	} else {
		return ResultadoVerificacionPlanMaterialAlmacenV2{
			huellaPlan: huellaPlan, huellaVinculo: vinculo,
		}, nil
	}
}

func (r ResultadoVerificacionPlanMaterialAlmacenV2) validarPara(
	solicitud SolicitudVerificarPlanMaterialAlmacenV2,
) error {
	_, _, _, _, _, _, _, _, _, _, _, vinculo, err :=
		solicitud.RevelarParaVerificacionPlanMaterial()
	if err != nil || r.huellaPlan == ([sha256.Size]byte{}) ||
		r.huellaVinculo == ([sha256.Size]byte{}) ||
		subtle.ConstantTimeCompare(r.huellaVinculo[:], vinculo[:]) != 1 ||
		huellasMaterialV2Iguales(r.huellaPlan, vinculo) {
		return errorReciboMaterialV2()
	}
	return nil
}

type VerificadorPlanMaterialAlmacenV2 interface {
	VerificarPlanMaterialAlmacenV2(
		context.Context,
		SolicitudVerificarPlanMaterialAlmacenV2,
	) (ResultadoVerificacionPlanMaterialAlmacenV2, error)
}

func (h HuellaPlanMaterialAlmacenV2) Hexadecimal() (string, error) {
	if !h.valida() {
		return "", errorReciboMaterialV2()
	}
	return hex.EncodeToString(h.suma[:]), nil
}

// SolicitudAtestarMaterialAlmacenV2 es la unica apertura de los bytes hacia
// el adaptador criptografico. Entrega siempre una copia defensiva.
type SolicitudAtestarMaterialAlmacenV2 struct {
	dominio string
	mensaje []byte
	huella  [sha256.Size]byte
}

func nuevaSolicitudAtestarMaterialAlmacenV2(
	dominio string,
	mensaje []byte,
) (SolicitudAtestarMaterialAlmacenV2, error) {
	if !dominioAtestacionMaterialV2Valido(dominio) || len(mensaje) == 0 {
		return SolicitudAtestarMaterialAlmacenV2{}, errorAtestacionMaterialV2()
	}
	copia := append([]byte(nil), mensaje...)
	return SolicitudAtestarMaterialAlmacenV2{
		dominio: dominio, mensaje: copia, huella: sha256.Sum256(copia),
	}, nil
}

func (s SolicitudAtestarMaterialAlmacenV2) validar() error {
	esperada := sumaSHA256MaterialV2(s.mensaje)
	if !dominioAtestacionMaterialV2Valido(s.dominio) || len(s.mensaje) == 0 ||
		s.huella == ([sha256.Size]byte{}) ||
		subtle.ConstantTimeCompare(s.huella[:], esperada[:]) != 1 {
		return errorAtestacionMaterialV2()
	}
	return nil
}

// RevelarParaAtestacion devuelve el dominio separado, los bytes exactos y su
// huella. No revela credenciales ni acepta que el adaptador cambie el mensaje.
func (s SolicitudAtestarMaterialAlmacenV2) RevelarParaAtestacion() (
	dominio string,
	mensaje []byte,
	huellaSHA256 [sha256.Size]byte,
	err error,
) {
	if err = s.validar(); err != nil {
		return "", nil, [sha256.Size]byte{}, err
	}
	return s.dominio, append([]byte(nil), s.mensaje...), s.huella, nil
}

// AtestacionCriptograficaMaterialAlmacenV2 liga algoritmo, clave versionada,
// dominio y huella del mensaje. El codigo se copia al entrar y al salir.
type AtestacionCriptograficaMaterialAlmacenV2 struct {
	algoritmo    AlgoritmoAtestacionMaterialAlmacenV2
	claveRef     string
	claveVersion uint32
	dominio      string
	huella       [sha256.Size]byte
	codigo       []byte
}

func NuevaAtestacionCriptograficaMaterialAlmacenV2(
	solicitud SolicitudAtestarMaterialAlmacenV2,
	algoritmo AlgoritmoAtestacionMaterialAlmacenV2,
	claveRef string,
	claveVersion uint32,
	codigo []byte,
) (AtestacionCriptograficaMaterialAlmacenV2, error) {
	if solicitud.validar() != nil || !algoritmo.valida() ||
		!aliasLogicoMaterialV2Valido(claveRef, 256) || claveVersion == 0 ||
		!codigoAtestacionMaterialV2Valido(algoritmo, codigo) {
		return AtestacionCriptograficaMaterialAlmacenV2{}, errorAtestacionMaterialV2()
	}
	atestacion := AtestacionCriptograficaMaterialAlmacenV2{
		algoritmo: algoritmo, claveRef: claveRef, claveVersion: claveVersion,
		dominio: solicitud.dominio, huella: solicitud.huella,
		codigo: append([]byte(nil), codigo...),
	}
	if atestacion.validarPara(solicitud) != nil {
		return AtestacionCriptograficaMaterialAlmacenV2{}, errorAtestacionMaterialV2()
	}
	return atestacion, nil
}

func (a AtestacionCriptograficaMaterialAlmacenV2) validarPara(
	solicitud SolicitudAtestarMaterialAlmacenV2,
) error {
	if solicitud.validar() != nil || !a.algoritmo.valida() ||
		!aliasLogicoMaterialV2Valido(a.claveRef, 256) || a.claveVersion == 0 ||
		a.dominio != solicitud.dominio ||
		subtle.ConstantTimeCompare(a.huella[:], solicitud.huella[:]) != 1 ||
		!codigoAtestacionMaterialV2Valido(a.algoritmo, a.codigo) {
		return errorAtestacionMaterialV2()
	}
	return nil
}

// SolicitudVerificarAtestacionMaterialAlmacenV2 impide verificar una firma
// descontextualizada o sobre bytes recompuestos por otro componente.
type SolicitudVerificarAtestacionMaterialAlmacenV2 struct {
	solicitud  SolicitudAtestarMaterialAlmacenV2
	atestacion AtestacionCriptograficaMaterialAlmacenV2
}

func nuevaSolicitudVerificarAtestacionMaterialAlmacenV2(
	solicitud SolicitudAtestarMaterialAlmacenV2,
	atestacion AtestacionCriptograficaMaterialAlmacenV2,
) (SolicitudVerificarAtestacionMaterialAlmacenV2, error) {
	if atestacion.validarPara(solicitud) != nil {
		return SolicitudVerificarAtestacionMaterialAlmacenV2{}, errorAtestacionMaterialV2()
	}
	return SolicitudVerificarAtestacionMaterialAlmacenV2{
		solicitud: solicitud, atestacion: atestacion,
	}, nil
}

// RevelarParaVerificacion es la unica apertura de clave, version, mensaje y
// autenticador hacia un verificador homologado.
func (s SolicitudVerificarAtestacionMaterialAlmacenV2) RevelarParaVerificacion() (
	dominio string,
	mensaje []byte,
	algoritmo AlgoritmoAtestacionMaterialAlmacenV2,
	claveRef string,
	claveVersion uint32,
	codigo []byte,
	err error,
) {
	if s.atestacion.validarPara(s.solicitud) != nil {
		return "", nil, "", "", 0, nil, errorAtestacionMaterialV2()
	}
	return s.solicitud.dominio, append([]byte(nil), s.solicitud.mensaje...),
		s.atestacion.algoritmo, s.atestacion.claveRef, s.atestacion.claveVersion,
		append([]byte(nil), s.atestacion.codigo...), nil
}

type AtestadorMaterialAlmacenV2 interface {
	AtestarMaterialAlmacenV2(
		context.Context,
		SolicitudAtestarMaterialAlmacenV2,
	) (AtestacionCriptograficaMaterialAlmacenV2, error)
}

type VerificadorAtestacionMaterialAlmacenV2 interface {
	VerificarAtestacionMaterialAlmacenV2(
		context.Context,
		SolicitudVerificarAtestacionMaterialAlmacenV2,
	) error
}

// SolicitudVerificarPerfilPublicadoMaterialV2 separa autenticidad
// criptografica de homologacion. Una firma valida sobre capacidades elegidas
// por el llamador no convierte el perfil en un perfil publicado.
type SolicitudVerificarPerfilPublicadoMaterialV2 struct {
	referencia       string
	version          uint32
	conectorLogicoID string
	huella           [sha256.Size]byte
	canonico         []byte
}

func nuevaSolicitudVerificarPerfilPublicadoMaterialV2(
	perfil PerfilCapacidadesAlmacenMaterialV2,
) (SolicitudVerificarPerfilPublicadoMaterialV2, error) {
	if perfil.Validar() != nil {
		return SolicitudVerificarPerfilPublicadoMaterialV2{}, errorReciboMaterialV2()
	}
	canonico, err := perfil.canonicoSinAtestacion()
	if err != nil {
		return SolicitudVerificarPerfilPublicadoMaterialV2{}, errorReciboMaterialV2()
	}
	return SolicitudVerificarPerfilPublicadoMaterialV2{
		referencia: perfil.referencia, version: perfil.version,
		conectorLogicoID: perfil.conectorLogicoID, huella: perfil.huella,
		canonico: append([]byte(nil), canonico...),
	}, nil
}

func (s SolicitudVerificarPerfilPublicadoMaterialV2) RevelarParaHomologacion() (
	referencia string,
	version uint32,
	conectorLogicoID string,
	huella [sha256.Size]byte,
	canonico []byte,
	err error,
) {
	if !aliasLogicoMaterialV2Valido(s.referencia, 512) || s.version == 0 ||
		!aliasLogicoMaterialV2Valido(s.conectorLogicoID, 128) ||
		s.huella == ([sha256.Size]byte{}) || len(s.canonico) == 0 {
		return "", 0, "", [sha256.Size]byte{}, nil, errorReciboMaterialV2()
	}
	esperada := sha256.Sum256(s.canonico)
	if subtle.ConstantTimeCompare(s.huella[:], esperada[:]) != 1 {
		return "", 0, "", [sha256.Size]byte{}, nil, errorReciboMaterialV2()
	}
	return s.referencia, s.version, s.conectorLogicoID, s.huella,
		append([]byte(nil), s.canonico...), nil
}

// VerificadorPerfilPublicadoMaterialV2 debe consultar el catalogo
// autoritativo de perfiles homologados y cotejar referencia, version, huella
// y bytes exactos. No puede limitarse a volver a verificar la firma.
type VerificadorPerfilPublicadoMaterialV2 interface {
	VerificarPerfilPublicadoMaterialV2(
		context.Context,
		SolicitudVerificarPerfilPublicadoMaterialV2,
	) error
}

// PerfilCapacidadesAlmacenMaterialV2 es una instantanea atestada de las
// capacidades materiales relevantes para escritura. Omite de forma expresa
// origenes, endpoint, bucket, rutas y cualquier detalle fisico.
type PerfilCapacidadesAlmacenMaterialV2 struct {
	esquema                string
	versionEsquema         uint16
	referencia             string
	version                uint32
	conectorLogicoID       string
	escrituraEnFlujo       bool
	referenciasOpacas      bool
	integridadSHA256       bool
	versionado             bool
	retencion              bool
	bloqueoLegal           bool
	cifradoEnTransito      bool
	cifradoEnReposo        bool
	cifradoPorObjeto       bool
	preservaObjetoOriginal bool
	tamanoMaximoObjeto     int64
	huella                 [sha256.Size]byte
	atestacion             AtestacionCriptograficaMaterialAlmacenV2
}

func NuevoPerfilCapacidadesAlmacenMaterialV2(
	ctx context.Context,
	referencia string,
	version uint32,
	capacidades CapacidadesAlmacenObjetos,
	atestador AtestadorMaterialAlmacenV2,
	verificador VerificadorAtestacionMaterialAlmacenV2,
	verificadorPublicacion VerificadorPerfilPublicadoMaterialV2,
) (PerfilCapacidadesAlmacenMaterialV2, error) {
	perfil := PerfilCapacidadesAlmacenMaterialV2{
		esquema:        EsquemaPerfilCapacidadesAlmacenMaterialV2,
		versionEsquema: VersionEsquemaMaterialAlmacenV2,
		referencia:     referencia, version: version,
		conectorLogicoID:       capacidades.ConectorID,
		escrituraEnFlujo:       capacidades.EscrituraEnFlujo,
		referenciasOpacas:      capacidades.ReferenciasOpacas,
		integridadSHA256:       capacidades.IntegridadSHA256,
		versionado:             capacidades.Versionado,
		retencion:              capacidades.Retencion,
		bloqueoLegal:           capacidades.BloqueoLegal,
		cifradoEnTransito:      capacidades.CifradoEnTransito,
		cifradoEnReposo:        capacidades.CifradoEnReposo,
		cifradoPorObjeto:       capacidades.CifradoPorObjeto,
		preservaObjetoOriginal: capacidades.PreservaObjetoOriginal,
		tamanoMaximoObjeto:     capacidades.TamanoMaximoObjeto,
	}
	if ctx == nil || ctx.Err() != nil || perfil.validarHechos() != nil ||
		dependenciaMaterialV2Nula(atestador) || dependenciaMaterialV2Nula(verificador) ||
		dependenciaMaterialV2Nula(verificadorPublicacion) {
		return PerfilCapacidadesAlmacenMaterialV2{}, errorReciboMaterialV2()
	}
	canonico, err := perfil.canonicoSinAtestacion()
	if err != nil {
		return PerfilCapacidadesAlmacenMaterialV2{}, errorReciboMaterialV2()
	}
	perfil.huella = sha256.Sum256(canonico)
	solicitud, err := nuevaSolicitudAtestarMaterialAlmacenV2(
		dominioAtestacionPerfilCapacidadesMaterialV2, canonico,
	)
	if err != nil {
		return PerfilCapacidadesAlmacenMaterialV2{}, errorReciboMaterialV2()
	}
	if ctx.Err() != nil {
		return PerfilCapacidadesAlmacenMaterialV2{}, errorReciboMaterialV2()
	}
	atestacion, err := atestador.AtestarMaterialAlmacenV2(ctx, solicitud)
	if err != nil || ctx.Err() != nil || atestacion.validarPara(solicitud) != nil {
		return PerfilCapacidadesAlmacenMaterialV2{}, errorReciboMaterialV2()
	}
	peticion, err := nuevaSolicitudVerificarAtestacionMaterialAlmacenV2(solicitud, atestacion)
	if err != nil || ctx.Err() != nil {
		return PerfilCapacidadesAlmacenMaterialV2{}, errorReciboMaterialV2()
	}
	errVerificacion := verificador.VerificarAtestacionMaterialAlmacenV2(ctx, peticion)
	if errVerificacion != nil || ctx.Err() != nil {
		return PerfilCapacidadesAlmacenMaterialV2{}, errorReciboMaterialV2()
	}
	perfil.atestacion = atestacion
	if perfil.Validar() != nil {
		return PerfilCapacidadesAlmacenMaterialV2{}, errorReciboMaterialV2()
	}
	if perfil.VerificarPublicacion(ctx, verificadorPublicacion) != nil {
		return PerfilCapacidadesAlmacenMaterialV2{}, errorReciboMaterialV2()
	}
	return perfil, nil
}

func (p PerfilCapacidadesAlmacenMaterialV2) validarHechos() error {
	if p.esquema != EsquemaPerfilCapacidadesAlmacenMaterialV2 ||
		p.versionEsquema != VersionEsquemaMaterialAlmacenV2 ||
		!aliasLogicoMaterialV2Valido(p.referencia, 512) || p.version == 0 ||
		!aliasLogicoMaterialV2Valido(p.conectorLogicoID, 128) ||
		!p.escrituraEnFlujo || !p.referenciasOpacas || !p.integridadSHA256 ||
		!p.versionado || !p.cifradoEnTransito || !p.cifradoEnReposo ||
		p.tamanoMaximoObjeto < 1 {
		return errorReciboMaterialV2()
	}
	return nil
}

func (p PerfilCapacidadesAlmacenMaterialV2) Validar() error {
	if p.validarHechos() != nil || p.huella == ([sha256.Size]byte{}) {
		return errorReciboMaterialV2()
	}
	canonico, err := p.canonicoSinAtestacion()
	esperada := sumaSHA256MaterialV2(canonico)
	if err != nil || subtle.ConstantTimeCompare(p.huella[:], esperada[:]) != 1 {
		return errorReciboMaterialV2()
	}
	solicitud, err := nuevaSolicitudAtestarMaterialAlmacenV2(
		dominioAtestacionPerfilCapacidadesMaterialV2, canonico,
	)
	if err != nil || p.atestacion.validarPara(solicitud) != nil {
		return errorReciboMaterialV2()
	}
	return nil
}

func (p PerfilCapacidadesAlmacenMaterialV2) VerificarAtestacion(
	ctx context.Context,
	verificador VerificadorAtestacionMaterialAlmacenV2,
) error {
	if ctx == nil || ctx.Err() != nil || p.Validar() != nil ||
		dependenciaMaterialV2Nula(verificador) {
		return errorReciboMaterialV2()
	}
	canonico, _ := p.canonicoSinAtestacion()
	solicitud, _ := nuevaSolicitudAtestarMaterialAlmacenV2(
		dominioAtestacionPerfilCapacidadesMaterialV2, canonico,
	)
	peticion, err := nuevaSolicitudVerificarAtestacionMaterialAlmacenV2(solicitud, p.atestacion)
	if err != nil || ctx.Err() != nil {
		return errorReciboMaterialV2()
	}
	errVerificacion := verificador.VerificarAtestacionMaterialAlmacenV2(ctx, peticion)
	if errVerificacion != nil || ctx.Err() != nil {
		return errorReciboMaterialV2()
	}
	return nil
}

// VerificarPublicacion reconsulta el catalogo autoritativo. Se ejecuta al
// crear el perfil y de nuevo antes de cada recibo para que una revocacion
// posterior falle cerrada.
func (p PerfilCapacidadesAlmacenMaterialV2) VerificarPublicacion(
	ctx context.Context,
	verificador VerificadorPerfilPublicadoMaterialV2,
) error {
	if ctx == nil || ctx.Err() != nil || p.Validar() != nil ||
		dependenciaMaterialV2Nula(verificador) {
		return errorReciboMaterialV2()
	}
	solicitud, err := nuevaSolicitudVerificarPerfilPublicadoMaterialV2(p)
	if err != nil || ctx.Err() != nil {
		return errorReciboMaterialV2()
	}
	errVerificacion := verificador.VerificarPerfilPublicadoMaterialV2(ctx, solicitud)
	if errVerificacion != nil || ctx.Err() != nil {
		return errorReciboMaterialV2()
	}
	return nil
}

func (p PerfilCapacidadesAlmacenMaterialV2) cotejar(
	capacidades CapacidadesAlmacenObjetos,
) error {
	if p.Validar() != nil || p.conectorLogicoID != capacidades.ConectorID ||
		p.escrituraEnFlujo != capacidades.EscrituraEnFlujo ||
		p.referenciasOpacas != capacidades.ReferenciasOpacas ||
		p.integridadSHA256 != capacidades.IntegridadSHA256 ||
		p.versionado != capacidades.Versionado || p.retencion != capacidades.Retencion ||
		p.bloqueoLegal != capacidades.BloqueoLegal ||
		p.cifradoEnTransito != capacidades.CifradoEnTransito ||
		p.cifradoEnReposo != capacidades.CifradoEnReposo ||
		p.cifradoPorObjeto != capacidades.CifradoPorObjeto ||
		p.preservaObjetoOriginal != capacidades.PreservaObjetoOriginal ||
		p.tamanoMaximoObjeto != capacidades.TamanoMaximoObjeto {
		return errorReciboMaterialV2()
	}
	return nil
}

func (p PerfilCapacidadesAlmacenMaterialV2) canonicoSinAtestacion() ([]byte, error) {
	if p.validarHechos() != nil {
		return nil, errorReciboMaterialV2()
	}
	var canonico []byte
	canonico = anexarTLVMaterialV2(canonico, 0, []byte(p.esquema))
	canonico = anexarTLVMaterialV2(canonico, 1, uint16MaterialV2(p.versionEsquema))
	canonico = anexarTLVMaterialV2(canonico, 2, []byte(p.referencia))
	canonico = anexarTLVMaterialV2(canonico, 3, uint32MaterialV2(p.version))
	canonico = anexarTLVMaterialV2(canonico, 4, []byte(p.conectorLogicoID))
	canonico = anexarTLVMaterialV2(canonico, 5, boolMaterialV2(p.escrituraEnFlujo))
	canonico = anexarTLVMaterialV2(canonico, 6, boolMaterialV2(p.referenciasOpacas))
	canonico = anexarTLVMaterialV2(canonico, 7, boolMaterialV2(p.integridadSHA256))
	canonico = anexarTLVMaterialV2(canonico, 8, boolMaterialV2(p.versionado))
	canonico = anexarTLVMaterialV2(canonico, 9, boolMaterialV2(p.retencion))
	canonico = anexarTLVMaterialV2(canonico, 10, boolMaterialV2(p.bloqueoLegal))
	canonico = anexarTLVMaterialV2(canonico, 11, boolMaterialV2(p.cifradoEnTransito))
	canonico = anexarTLVMaterialV2(canonico, 12, boolMaterialV2(p.cifradoEnReposo))
	canonico = anexarTLVMaterialV2(canonico, 13, boolMaterialV2(p.cifradoPorObjeto))
	canonico = anexarTLVMaterialV2(canonico, 14, boolMaterialV2(p.preservaObjetoOriginal))
	canonico = anexarTLVMaterialV2(canonico, 15, int64MaterialV2(p.tamanoMaximoObjeto))
	return canonico, nil
}

func (p PerfilCapacidadesAlmacenMaterialV2) BytesCanonicos() ([]byte, error) {
	if p.Validar() != nil {
		return nil, errorReciboMaterialV2()
	}
	return p.canonicoSinAtestacion()
}

func (p PerfilCapacidadesAlmacenMaterialV2) HuellaSHA256() ([sha256.Size]byte, error) {
	if p.Validar() != nil {
		return [sha256.Size]byte{}, errorReciboMaterialV2()
	}
	return p.huella, nil
}

// InstantaneaObjetoMaterialV2 contiene solo hechos originales del objeto. No
// incorpora la evidencia nueva de un reintento ni datos del intento.
type InstantaneaObjetoMaterialV2 struct {
	esquema              string
	versionEsquema       uint16
	conectorLogicoID     string
	objetoRef            string
	objetoVersion        string
	zona                 ZonaAlmacen
	mime                 string
	tamano               int64
	huellaContenido      [sha256.Size]byte
	evidenciaCreacionRef string
	almacenadoEn         time.Time
	tieneRetencion       bool
	retenidoHasta        time.Time
	estadoInmovilizacion EstadoInmovilizacionObjetoMaterialV2
	estadoObjeto         EstadoObjetoMaterialV2
}

func nuevaInstantaneaObjetoMaterialV2(
	objeto ObjetoAlmacenado,
) (InstantaneaObjetoMaterialV2, error) {
	huella, err := decodificarSHA256MaterialV2(objeto.HuellaSHA256)
	if err != nil {
		return InstantaneaObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	estadoInmovilizacion := EstadoInmovilizacionMaterialNoAplicada
	if objeto.Inmovilizado {
		estadoInmovilizacion = EstadoInmovilizacionMaterialAplicada
	}
	instantanea := InstantaneaObjetoMaterialV2{
		esquema:          EsquemaInstantaneaObjetoMaterialV2,
		versionEsquema:   VersionEsquemaMaterialAlmacenV2,
		conectorLogicoID: objeto.ConectorID,
		objetoRef:        objeto.Objeto.Referencia, objetoVersion: objeto.Objeto.Version,
		zona: objeto.Zona, mime: objeto.MIME, tamano: objeto.Tamano,
		huellaContenido:      huella,
		evidenciaCreacionRef: objeto.EvidenciaCreacionRef,
		almacenadoEn:         objeto.AlmacenadoEn,
		tieneRetencion:       !objeto.RetenidoHasta.IsZero(), retenidoHasta: objeto.RetenidoHasta,
		estadoInmovilizacion: estadoInmovilizacion, estadoObjeto: EstadoObjetoMaterialActivo,
	}
	if instantanea.Validar() != nil {
		return InstantaneaObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	return instantanea, nil
}

func (i InstantaneaObjetoMaterialV2) Validar() error {
	if i.esquema != EsquemaInstantaneaObjetoMaterialV2 ||
		i.versionEsquema != VersionEsquemaMaterialAlmacenV2 ||
		!aliasLogicoMaterialV2Valido(i.conectorLogicoID, 128) ||
		!aliasLogicoMaterialV2Valido(i.objetoRef, 512) ||
		!aliasLogicoMaterialV2Valido(i.objetoVersion, 256) || !i.zona.Valida() ||
		!mimeMaterialV2Valido(i.mime) || i.tamano < 1 ||
		i.huellaContenido == ([sha256.Size]byte{}) ||
		!aliasLogicoMaterialV2Valido(i.evidenciaCreacionRef, 512) ||
		!instanteMaterialV2Valido(i.almacenadoEn) ||
		!i.estadoInmovilizacion.valida() || !i.estadoObjeto.valida() {
		return errorReciboMaterialV2()
	}
	if i.tieneRetencion {
		if !instanteMaterialV2Valido(i.retenidoHasta) ||
			!i.retenidoHasta.After(i.almacenadoEn) {
			return errorReciboMaterialV2()
		}
	} else if !i.retenidoHasta.IsZero() {
		return errorReciboMaterialV2()
	}
	return nil
}

func (i InstantaneaObjetoMaterialV2) canonico() ([]byte, error) {
	if i.Validar() != nil {
		return nil, errorReciboMaterialV2()
	}
	var canonico []byte
	canonico = anexarTLVMaterialV2(canonico, 0, []byte(i.esquema))
	canonico = anexarTLVMaterialV2(canonico, 1, uint16MaterialV2(i.versionEsquema))
	canonico = anexarTLVMaterialV2(canonico, 2, []byte(i.conectorLogicoID))
	canonico = anexarTLVMaterialV2(canonico, 3, []byte(i.objetoRef))
	canonico = anexarTLVMaterialV2(canonico, 4, []byte(i.objetoVersion))
	canonico = anexarTLVMaterialV2(canonico, 5, []byte(i.zona))
	canonico = anexarTLVMaterialV2(canonico, 6, []byte(i.mime))
	canonico = anexarTLVMaterialV2(canonico, 7, int64MaterialV2(i.tamano))
	canonico = anexarTLVMaterialV2(canonico, 8, i.huellaContenido[:])
	canonico = anexarTLVMaterialV2(canonico, 9, []byte(i.evidenciaCreacionRef))
	canonico = anexarTLVMaterialV2(canonico, 10, int64MaterialV2(i.almacenadoEn.UnixMicro()))
	canonico = anexarTLVMaterialV2(canonico, 11, boolMaterialV2(i.tieneRetencion))
	if i.tieneRetencion {
		canonico = anexarTLVMaterialV2(canonico, 12, int64MaterialV2(i.retenidoHasta.UnixMicro()))
	}
	canonico = anexarTLVMaterialV2(canonico, 13, []byte(i.estadoInmovilizacion))
	canonico = anexarTLVMaterialV2(canonico, 14, []byte(i.estadoObjeto))
	return canonico, nil
}

func (i InstantaneaObjetoMaterialV2) BytesCanonicos() ([]byte, error) {
	return i.canonico()
}

func (i InstantaneaObjetoMaterialV2) HuellaSHA256() ([sha256.Size]byte, error) {
	canonico, err := i.canonico()
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(canonico), nil
}

// SolicitudReservarReferenciaReciboMaterialV2 identifica el recibo por todos
// sus hechos materiales, sin aceptar una referencia propuesta por el
// llamador. El registro debe reservarla o recuperar la original atomica y
// durablemente.
type SolicitudReservarReferenciaReciboMaterialV2 struct {
	huellaIdentidad [sha256.Size]byte
}

func nuevaSolicitudReservarReferenciaReciboMaterialV2(
	canonicoIdentidad []byte,
) (SolicitudReservarReferenciaReciboMaterialV2, error) {
	if len(canonicoIdentidad) == 0 {
		return SolicitudReservarReferenciaReciboMaterialV2{}, errorReciboMaterialV2()
	}
	huella := sha256.Sum256(canonicoIdentidad)
	if huella == ([sha256.Size]byte{}) {
		return SolicitudReservarReferenciaReciboMaterialV2{}, errorReciboMaterialV2()
	}
	return SolicitudReservarReferenciaReciboMaterialV2{huellaIdentidad: huella}, nil
}

func (s SolicitudReservarReferenciaReciboMaterialV2) HuellaIdentidad() (
	[sha256.Size]byte,
	error,
) {
	if s.huellaIdentidad == ([sha256.Size]byte{}) {
		return [sha256.Size]byte{}, errorReciboMaterialV2()
	}
	return s.huellaIdentidad, nil
}

// ResultadoReferenciaReciboMaterialV2 es una capacidad opaca ligada a la
// solicitud exacta. Su forma no acredita durabilidad: la fabrica exige ademas
// un verificador autoritativo del registro.
type ResultadoReferenciaReciboMaterialV2 struct {
	referencia      string
	huellaIdentidad [sha256.Size]byte
}

func NuevoResultadoReferenciaReciboMaterialV2(
	solicitud SolicitudReservarReferenciaReciboMaterialV2,
	referencia string,
) (ResultadoReferenciaReciboMaterialV2, error) {
	huella, err := solicitud.HuellaIdentidad()
	if err != nil || !aliasLogicoMaterialV2Valido(referencia, 512) {
		return ResultadoReferenciaReciboMaterialV2{}, errorReciboMaterialV2()
	}
	return ResultadoReferenciaReciboMaterialV2{
		referencia: referencia, huellaIdentidad: huella,
	}, nil
}

func (r ResultadoReferenciaReciboMaterialV2) validarPara(
	solicitud SolicitudReservarReferenciaReciboMaterialV2,
) error {
	huella, err := solicitud.HuellaIdentidad()
	if err != nil || !aliasLogicoMaterialV2Valido(r.referencia, 512) ||
		r.huellaIdentidad == ([sha256.Size]byte{}) ||
		subtle.ConstantTimeCompare(r.huellaIdentidad[:], huella[:]) != 1 {
		return errorReciboMaterialV2()
	}
	return nil
}

type RegistroReferenciasReciboMaterialV2 interface {
	ReservarORecuperarReferenciaReciboMaterialV2(
		context.Context,
		SolicitudReservarReferenciaReciboMaterialV2,
	) (ResultadoReferenciaReciboMaterialV2, error)
}

// VerificadorReferenciaReciboMaterialV2 debe consultar el mismo registro
// durable por la huella estable y confirmar que la referencia devuelta es la
// original. Un comprobador de forma no cumple este contrato.
type VerificadorReferenciaReciboMaterialV2 interface {
	VerificarReferenciaReciboMaterialV2(
		context.Context,
		SolicitudReservarReferenciaReciboMaterialV2,
		ResultadoReferenciaReciboMaterialV2,
	) error
}

// ReciboEscrituraObjetoMaterialV2 liga el objeto original con el perfil de
// capacidades y un plan material. No es aun un contrato productivo: sin
// diario durable, persistencia y recuperacion el flujo completo sigue NO-GO.
type ReciboEscrituraObjetoMaterialV2 struct {
	esquema                   string
	versionEsquema            uint16
	referenciaDurableOriginal string
	perfilReferencia          string
	perfilVersion             uint32
	huellaPerfil              [sha256.Size]byte
	moduloID                  string
	accionNegocio             string
	accionTecnica             string
	recursoRef                string
	operacionRef              string
	cargaRef                  string
	efectoRef                 string
	huellaPlanMaterial        HuellaPlanMaterialAlmacenV2
	clasificacion             string
	instantanea               InstantaneaObjetoMaterialV2
	huella                    [sha256.Size]byte
	atestacion                AtestacionCriptograficaMaterialAlmacenV2
}

func NuevoReciboEscrituraObjetoMaterialV2(
	ctx context.Context,
	solicitud SolicitudEscribirObjeto,
	resultado ResultadoOperacionObjeto,
	capacidades CapacidadesAlmacenObjetos,
	perfil PerfilCapacidadesAlmacenMaterialV2,
	verificadorPublicacion VerificadorPerfilPublicadoMaterialV2,
	seleccionPlan SeleccionPlanMaterialAlmacenV2,
	verificadorPlan VerificadorPlanMaterialAlmacenV2,
	registroReferencias RegistroReferenciasReciboMaterialV2,
	verificadorReferencia VerificadorReferenciaReciboMaterialV2,
	atestador AtestadorMaterialAlmacenV2,
	verificador VerificadorAtestacionMaterialAlmacenV2,
) (ReciboEscrituraObjetoMaterialV2, error) {
	if ctx == nil || ctx.Err() != nil ||
		solicitud.Validar() != nil ||
		resultado.ValidarEscritura(solicitud, capacidades) != nil ||
		perfil.cotejar(capacidades) != nil ||
		!seleccionPlan.valida() || dependenciaMaterialV2Nula(verificadorPlan) ||
		dependenciaMaterialV2Nula(verificadorPublicacion) ||
		dependenciaMaterialV2Nula(registroReferencias) ||
		dependenciaMaterialV2Nula(verificadorReferencia) ||
		dependenciaMaterialV2Nula(atestador) || dependenciaMaterialV2Nula(verificador) {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	if perfil.VerificarAtestacion(ctx, verificador) != nil || ctx.Err() != nil {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	if ctx.Err() != nil || perfil.VerificarPublicacion(ctx, verificadorPublicacion) != nil ||
		ctx.Err() != nil {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	proyeccion, err := solicitud.Contexto.Proyeccion()
	if err != nil || proyeccion.AccionTecnica != AccionAlmacenEscribir ||
		!hechosEstablesContextoMaterialV2Validos(proyeccion) {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	solicitudPlan, err := nuevaSolicitudVerificarPlanMaterialAlmacenV2(
		seleccionPlan, perfil.conectorLogicoID, proyeccion,
	)
	if err != nil {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	if ctx.Err() != nil {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	resultadoPlan, err := verificadorPlan.VerificarPlanMaterialAlmacenV2(ctx, solicitudPlan)
	if err != nil || ctx.Err() != nil || resultadoPlan.validarPara(solicitudPlan) != nil {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	huellaPlanMaterial := HuellaPlanMaterialAlmacenV2{
		referencia: seleccionPlan.referencia, version: seleccionPlan.version,
		suma: resultadoPlan.huellaPlan, huellaVinculo: resultadoPlan.huellaVinculo,
	}
	huellaPlanV1, err := decodificarSHA256MaterialV2(proyeccion.HuellaPlanEfectoSHA256)
	// El plan V1 compromete autorizacion e intento. El plan material V2 usa un
	// dominio probatorio independiente; una igualdad exacta prueba reutilizacion
	// del valor prohibido, no compatibilidad entre ambos planes.
	if err != nil || !huellaPlanMaterial.valida() ||
		subtle.ConstantTimeCompare(huellaPlanMaterial.suma[:], huellaPlanV1[:]) == 1 {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	instantanea, err := nuevaInstantaneaObjetoMaterialV2(resultado.Objeto)
	if err != nil || instantanea.conectorLogicoID != perfil.conectorLogicoID ||
		huellasMaterialV2Iguales(huellaPlanMaterial.suma, instantanea.huellaContenido) ||
		huellasMaterialV2Iguales(huellaPlanMaterial.suma, perfil.huella) ||
		(instantanea.tieneRetencion && !perfil.retencion) ||
		(instantanea.estadoInmovilizacion == EstadoInmovilizacionMaterialAplicada &&
			!perfil.bloqueoLegal) {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	recibo := ReciboEscrituraObjetoMaterialV2{
		esquema:          EsquemaReciboEscrituraObjetoMaterialV2,
		versionEsquema:   VersionEsquemaMaterialAlmacenV2,
		perfilReferencia: perfil.referencia, perfilVersion: perfil.version,
		huellaPerfil: perfil.huella,
		moduloID:     proyeccion.ModuloID, accionNegocio: proyeccion.AccionNegocio,
		accionTecnica: proyeccion.AccionTecnica, recursoRef: proyeccion.RecursoRef,
		operacionRef: proyeccion.OperacionRef, cargaRef: proyeccion.CargaRef,
		efectoRef: proyeccion.EfectoRef, huellaPlanMaterial: huellaPlanMaterial,
		clasificacion: proyeccion.Clasificacion, instantanea: instantanea,
	}
	canonicoIdentidad, err := recibo.canonicoIdentidadDurable()
	if err != nil {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	solicitudReferencia, err := nuevaSolicitudReservarReferenciaReciboMaterialV2(canonicoIdentidad)
	if err != nil || ctx.Err() != nil {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	resultadoReferencia, err := registroReferencias.ReservarORecuperarReferenciaReciboMaterialV2(
		ctx, solicitudReferencia,
	)
	if err != nil || ctx.Err() != nil || resultadoReferencia.validarPara(solicitudReferencia) != nil {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	if ctx.Err() != nil {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	errVerificacionReferencia := verificadorReferencia.VerificarReferenciaReciboMaterialV2(
		ctx, solicitudReferencia, resultadoReferencia,
	)
	if errVerificacionReferencia != nil || ctx.Err() != nil {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	recibo.referenciaDurableOriginal = resultadoReferencia.referencia
	canonico, err := recibo.canonicoSinAtestacion()
	if err != nil {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	recibo.huella = sha256.Sum256(canonico)
	if huellasMaterialV2Iguales(recibo.huella, instantanea.huellaContenido) ||
		huellasMaterialV2Iguales(recibo.huella, perfil.huella) ||
		huellasMaterialV2Iguales(recibo.huella, huellaPlanMaterial.suma) {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	solicitudAtestacion, err := nuevaSolicitudAtestarMaterialAlmacenV2(
		dominioAtestacionReciboEscrituraMaterialV2, canonico,
	)
	if err != nil || ctx.Err() != nil {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	atestacion, err := atestador.AtestarMaterialAlmacenV2(ctx, solicitudAtestacion)
	if err != nil || ctx.Err() != nil || atestacion.validarPara(solicitudAtestacion) != nil {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	peticion, err := nuevaSolicitudVerificarAtestacionMaterialAlmacenV2(
		solicitudAtestacion, atestacion,
	)
	if err != nil || ctx.Err() != nil {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	errVerificacionAtestacion := verificador.VerificarAtestacionMaterialAlmacenV2(ctx, peticion)
	if errVerificacionAtestacion != nil || ctx.Err() != nil {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	recibo.atestacion = atestacion
	if recibo.Validar() != nil {
		return ReciboEscrituraObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	return recibo, nil
}

func (r ReciboEscrituraObjetoMaterialV2) validarHechos() error {
	if r.validarHechosMateriales() != nil ||
		!aliasLogicoMaterialV2Valido(r.referenciaDurableOriginal, 512) {
		return errorReciboMaterialV2()
	}
	return nil
}

func (r ReciboEscrituraObjetoMaterialV2) validarHechosMateriales() error {
	if r.esquema != EsquemaReciboEscrituraObjetoMaterialV2 ||
		r.versionEsquema != VersionEsquemaMaterialAlmacenV2 ||
		!aliasLogicoMaterialV2Valido(r.perfilReferencia, 512) || r.perfilVersion == 0 ||
		r.huellaPerfil == ([sha256.Size]byte{}) ||
		!aliasLogicoMaterialV2Valido(r.moduloID, 128) ||
		!aliasLogicoMaterialV2Valido(r.accionNegocio, 256) ||
		r.accionTecnica != AccionAlmacenEscribir ||
		!aliasLogicoMaterialV2Valido(r.accionTecnica, 128) ||
		!aliasLogicoMaterialV2Valido(r.recursoRef, 512) ||
		!aliasLogicoMaterialV2Valido(r.operacionRef, 512) ||
		!aliasLogicoMaterialV2Valido(r.cargaRef, 512) ||
		!aliasLogicoMaterialV2Valido(r.efectoRef, 512) ||
		!r.huellaPlanMaterial.valida() ||
		!aliasLogicoMaterialV2Valido(r.clasificacion, 256) ||
		r.instantanea.Validar() != nil {
		return errorReciboMaterialV2()
	}
	return nil
}

func (r ReciboEscrituraObjetoMaterialV2) Validar() error {
	if r.validarHechos() != nil || r.huella == ([sha256.Size]byte{}) {
		return errorReciboMaterialV2()
	}
	canonico, err := r.canonicoSinAtestacion()
	esperada := sumaSHA256MaterialV2(canonico)
	if err != nil || subtle.ConstantTimeCompare(r.huella[:], esperada[:]) != 1 ||
		huellasMaterialV2Iguales(r.huella, r.instantanea.huellaContenido) ||
		huellasMaterialV2Iguales(r.huella, r.huellaPerfil) ||
		huellasMaterialV2Iguales(r.huella, r.huellaPlanMaterial.suma) {
		return errorReciboMaterialV2()
	}
	solicitud, err := nuevaSolicitudAtestarMaterialAlmacenV2(
		dominioAtestacionReciboEscrituraMaterialV2, canonico,
	)
	if err != nil || r.atestacion.validarPara(solicitud) != nil {
		return errorReciboMaterialV2()
	}
	return nil
}

func (r ReciboEscrituraObjetoMaterialV2) VerificarAtestacion(
	ctx context.Context,
	verificador VerificadorAtestacionMaterialAlmacenV2,
) error {
	if ctx == nil || ctx.Err() != nil || r.Validar() != nil ||
		dependenciaMaterialV2Nula(verificador) {
		return errorReciboMaterialV2()
	}
	canonico, _ := r.canonicoSinAtestacion()
	solicitud, _ := nuevaSolicitudAtestarMaterialAlmacenV2(
		dominioAtestacionReciboEscrituraMaterialV2, canonico,
	)
	peticion, err := nuevaSolicitudVerificarAtestacionMaterialAlmacenV2(solicitud, r.atestacion)
	if err != nil || verificador.VerificarAtestacionMaterialAlmacenV2(ctx, peticion) != nil ||
		ctx.Err() != nil {
		return errorReciboMaterialV2()
	}
	return nil
}

func (r ReciboEscrituraObjetoMaterialV2) canonicoSinAtestacion() ([]byte, error) {
	if r.validarHechos() != nil {
		return nil, errorReciboMaterialV2()
	}
	var canonico []byte
	canonico = anexarTLVMaterialV2(canonico, 0, []byte(r.esquema))
	canonico = anexarTLVMaterialV2(canonico, 1, uint16MaterialV2(r.versionEsquema))
	canonico = anexarTLVMaterialV2(canonico, 2, []byte(r.referenciaDurableOriginal))
	return r.anexarHechosCanonicos(canonico), nil
}

func (r ReciboEscrituraObjetoMaterialV2) canonicoIdentidadDurable() ([]byte, error) {
	if r.validarHechosMateriales() != nil || r.referenciaDurableOriginal != "" {
		return nil, errorReciboMaterialV2()
	}
	var canonico []byte
	canonico = anexarTLVMaterialV2(
		canonico, 0, []byte("vec.almacen.identidad-recibo-escritura-material.v2"),
	)
	canonico = anexarTLVMaterialV2(canonico, 1, uint16MaterialV2(r.versionEsquema))
	return r.anexarHechosCanonicos(canonico), nil
}

func (r ReciboEscrituraObjetoMaterialV2) anexarHechosCanonicos(canonico []byte) []byte {
	i := r.instantanea
	canonico = anexarTLVMaterialV2(canonico, 3, []byte(i.conectorLogicoID))
	canonico = anexarTLVMaterialV2(canonico, 4, []byte(r.perfilReferencia))
	canonico = anexarTLVMaterialV2(canonico, 5, uint32MaterialV2(r.perfilVersion))
	canonico = anexarTLVMaterialV2(canonico, 6, r.huellaPerfil[:])
	canonico = anexarTLVMaterialV2(canonico, 7, []byte(r.moduloID))
	canonico = anexarTLVMaterialV2(canonico, 8, []byte(r.accionNegocio))
	canonico = anexarTLVMaterialV2(canonico, 9, []byte(r.accionTecnica))
	canonico = anexarTLVMaterialV2(canonico, 10, []byte(r.recursoRef))
	canonico = anexarTLVMaterialV2(canonico, 11, []byte(r.operacionRef))
	canonico = anexarTLVMaterialV2(canonico, 12, []byte(r.cargaRef))
	canonico = anexarTLVMaterialV2(canonico, 13, []byte(r.efectoRef))
	canonico = anexarTLVMaterialV2(canonico, 14, []byte(r.huellaPlanMaterial.referencia))
	canonico = anexarTLVMaterialV2(canonico, 15, uint32MaterialV2(r.huellaPlanMaterial.version))
	canonico = anexarTLVMaterialV2(canonico, 16, r.huellaPlanMaterial.suma[:])
	canonico = anexarTLVMaterialV2(canonico, 17, r.huellaPlanMaterial.huellaVinculo[:])
	canonico = anexarTLVMaterialV2(canonico, 18, []byte(r.clasificacion))
	canonico = anexarTLVMaterialV2(canonico, 19, []byte(i.objetoRef))
	canonico = anexarTLVMaterialV2(canonico, 20, []byte(i.objetoVersion))
	canonico = anexarTLVMaterialV2(canonico, 21, []byte(i.zona))
	canonico = anexarTLVMaterialV2(canonico, 22, []byte(i.mime))
	canonico = anexarTLVMaterialV2(canonico, 23, int64MaterialV2(i.tamano))
	canonico = anexarTLVMaterialV2(canonico, 24, i.huellaContenido[:])
	canonico = anexarTLVMaterialV2(canonico, 25, []byte(i.evidenciaCreacionRef))
	canonico = anexarTLVMaterialV2(canonico, 26, int64MaterialV2(i.almacenadoEn.UnixMicro()))
	canonico = anexarTLVMaterialV2(canonico, 27, boolMaterialV2(i.tieneRetencion))
	if i.tieneRetencion {
		canonico = anexarTLVMaterialV2(canonico, 28, int64MaterialV2(i.retenidoHasta.UnixMicro()))
	}
	canonico = anexarTLVMaterialV2(canonico, 29, []byte(i.estadoInmovilizacion))
	canonico = anexarTLVMaterialV2(canonico, 30, []byte(i.estadoObjeto))
	return canonico
}

func (r ReciboEscrituraObjetoMaterialV2) BytesCanonicos() ([]byte, error) {
	if r.Validar() != nil {
		return nil, errorReciboMaterialV2()
	}
	return r.canonicoSinAtestacion()
}

func (r ReciboEscrituraObjetoMaterialV2) HuellaSHA256() ([sha256.Size]byte, error) {
	if r.Validar() != nil {
		return [sha256.Size]byte{}, errorReciboMaterialV2()
	}
	return r.huella, nil
}

func (r ReciboEscrituraObjetoMaterialV2) Instantanea() (
	InstantaneaObjetoMaterialV2,
	error,
) {
	if r.Validar() != nil {
		return InstantaneaObjetoMaterialV2{}, errorReciboMaterialV2()
	}
	return r.instantanea, nil
}

func hechosEstablesContextoMaterialV2Validos(p ProyeccionContextoOperacionAlmacen) bool {
	return aliasLogicoMaterialV2Valido(p.ModuloID, 128) &&
		aliasLogicoMaterialV2Valido(p.AccionNegocio, 256) &&
		aliasLogicoMaterialV2Valido(p.AccionTecnica, 128) &&
		aliasLogicoMaterialV2Valido(p.RecursoRef, 512) &&
		aliasLogicoMaterialV2Valido(p.OperacionRef, 512) &&
		aliasLogicoMaterialV2Valido(p.CargaRef, 512) &&
		aliasLogicoMaterialV2Valido(p.EfectoRef, 512) &&
		aliasLogicoMaterialV2Valido(p.Clasificacion, 256)
}

func aliasLogicoMaterialV2Valido(valor string, maximo int) bool {
	if valor == "" || len(valor) > maximo || valor != strings.TrimSpace(valor) ||
		!utf8.ValidString(valor) || !textoASCIICanonicoMaterialV2(valor) ||
		strings.Contains(valor, "/") ||
		strings.Contains(valor, "\\") || strings.Contains(valor, "..") ||
		strings.Contains(valor, "://") || strings.ContainsAny(valor, "?#@*") {
		return false
	}
	minusculas := strings.ToLower(valor)
	for _, marcaFisica := range []string{
		"arn:", "etag:", "kms:", "bucket:", "bucket_", "endpoint:",
		"ruta:", "path:", "file:", "s3:", "http:", "https:",
		"dni:", "nif:", "nie:", "nombre:", "apellido:", "correo:",
		"email:", "telefono:", "direccion:",
	} {
		if strings.Contains(minusculas, marcaFisica) {
			return false
		}
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || unicode.IsSpace(caracter) {
			return false
		}
	}
	return !pareceIdentificadorPersonalMaterialV2(valor)
}

func textoASCIICanonicoMaterialV2(valor string) bool {
	for indice := 0; indice < len(valor); indice++ {
		if valor[indice] < 0x21 || valor[indice] > 0x7e {
			return false
		}
	}
	return true
}

func pareceIdentificadorPersonalMaterialV2(valor string) bool {
	mayusculas := strings.ToUpper(valor)
	if len(mayusculas) == 9 {
		digitosDNI := true
		for _, caracter := range mayusculas[:8] {
			if caracter < '0' || caracter > '9' {
				digitosDNI = false
				break
			}
		}
		ultima := mayusculas[8]
		if digitosDNI && ultima >= 'A' && ultima <= 'Z' {
			return true
		}
		primera := mayusculas[0]
		digitosNIE := primera == 'X' || primera == 'Y' || primera == 'Z'
		for _, caracter := range mayusculas[1:8] {
			if caracter < '0' || caracter > '9' {
				digitosNIE = false
				break
			}
		}
		if digitosNIE && ultima >= 'A' && ultima <= 'Z' {
			return true
		}
	}
	return false
}

func mimeMaterialV2Valido(valor string) bool {
	if !textoSeguroAlmacen(valor, 255) || strings.Count(valor, "/") != 1 ||
		!textoASCIICanonicoMaterialV2(valor) ||
		strings.ContainsAny(valor, ";?#\\") || valor != strings.ToLower(valor) {
		return false
	}
	partes := strings.Split(valor, "/")
	return len(partes) == 2 && partes[0] != "" && partes[1] != ""
}

func instanteMaterialV2Valido(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Nanosecond()%1_000 == 0
}

func codigoAtestacionMaterialV2Valido(
	algoritmo AlgoritmoAtestacionMaterialAlmacenV2,
	codigo []byte,
) bool {
	if algoritmo == AlgoritmoAtestacionMaterialHMACSHA256 {
		return len(codigo) == sha256.Size
	}
	return algoritmo == AlgoritmoAtestacionMaterialCOSESign1 &&
		len(codigo) >= 16 && len(codigo) <= tamanoMaximoAtestacionMaterialAlmacenV2
}

func dominioAtestacionMaterialV2Valido(dominio string) bool {
	return dominio == dominioAtestacionPerfilCapacidadesMaterialV2 ||
		dominio == dominioAtestacionReciboEscrituraMaterialV2
}

func decodificarSHA256MaterialV2(valor string) ([sha256.Size]byte, error) {
	var resultado [sha256.Size]byte
	if len(valor) != sha256.Size*2 || valor != strings.ToLower(valor) {
		return resultado, errorReciboMaterialV2()
	}
	contenido, err := hex.DecodeString(valor)
	if err != nil || len(contenido) != sha256.Size {
		return resultado, errorReciboMaterialV2()
	}
	copy(resultado[:], contenido)
	if resultado == ([sha256.Size]byte{}) {
		return [sha256.Size]byte{}, errorReciboMaterialV2()
	}
	return resultado, nil
}

func sumaSHA256MaterialV2(contenido []byte) [sha256.Size]byte {
	return sha256.Sum256(contenido)
}

func huellasMaterialV2Iguales(
	primera [sha256.Size]byte,
	segunda [sha256.Size]byte,
) bool {
	return subtle.ConstantTimeCompare(primera[:], segunda[:]) == 1
}

func anexarTLVMaterialV2(destino []byte, etiqueta uint16, valor []byte) []byte {
	var cabecera [10]byte
	binary.BigEndian.PutUint16(cabecera[0:2], etiqueta)
	binary.BigEndian.PutUint64(cabecera[2:10], uint64(len(valor)))
	destino = append(destino, cabecera[:]...)
	return append(destino, valor...)
}

func uint16MaterialV2(valor uint16) []byte {
	resultado := make([]byte, 2)
	binary.BigEndian.PutUint16(resultado, valor)
	return resultado
}

func uint32MaterialV2(valor uint32) []byte {
	resultado := make([]byte, 4)
	binary.BigEndian.PutUint32(resultado, valor)
	return resultado
}

func int64MaterialV2(valor int64) []byte {
	resultado := make([]byte, 8)
	binary.BigEndian.PutUint64(resultado, uint64(valor))
	return resultado
}

func boolMaterialV2(valor bool) []byte {
	if valor {
		return []byte{1}
	}
	return []byte{0}
}

func dependenciaMaterialV2Nula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

func errorReciboMaterialV2() error {
	return errors.Join(
		ErrSolicitudAlmacenInvalida,
		ErrReciboEscrituraObjetoMaterialV2NoValido,
	)
}

func errorAtestacionMaterialV2() error {
	return errors.Join(errorReciboMaterialV2(), ErrAtestacionMaterialAlmacenV2NoValida)
}

const textoRedactadoMaterialAlmacenV2 = "[MATERIAL-ALMACEN-V2-REDACTADO]"

func formatoRedactadoMaterialV2(estado fmt.State) {
	_, _ = io.WriteString(estado, textoRedactadoMaterialAlmacenV2)
}

func (SeleccionPlanMaterialAlmacenV2) String() string               { return textoRedactadoMaterialAlmacenV2 }
func (SeleccionPlanMaterialAlmacenV2) GoString() string             { return textoRedactadoMaterialAlmacenV2 }
func (s SeleccionPlanMaterialAlmacenV2) Format(e fmt.State, _ rune) { formatoRedactadoMaterialV2(e) }
func (SeleccionPlanMaterialAlmacenV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*SeleccionPlanMaterialAlmacenV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (SeleccionPlanMaterialAlmacenV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*SeleccionPlanMaterialAlmacenV2) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (SeleccionPlanMaterialAlmacenV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*SeleccionPlanMaterialAlmacenV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (s SeleccionPlanMaterialAlmacenV2) LogValue() slog.Value { return slog.StringValue(s.String()) }

func (SolicitudVerificarPlanMaterialAlmacenV2) String() string {
	return textoRedactadoMaterialAlmacenV2
}
func (SolicitudVerificarPlanMaterialAlmacenV2) GoString() string {
	return textoRedactadoMaterialAlmacenV2
}
func (s SolicitudVerificarPlanMaterialAlmacenV2) Format(e fmt.State, _ rune) {
	formatoRedactadoMaterialV2(e)
}
func (SolicitudVerificarPlanMaterialAlmacenV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*SolicitudVerificarPlanMaterialAlmacenV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (SolicitudVerificarPlanMaterialAlmacenV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*SolicitudVerificarPlanMaterialAlmacenV2) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (SolicitudVerificarPlanMaterialAlmacenV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*SolicitudVerificarPlanMaterialAlmacenV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (s SolicitudVerificarPlanMaterialAlmacenV2) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

func (ResultadoVerificacionPlanMaterialAlmacenV2) String() string {
	return textoRedactadoMaterialAlmacenV2
}
func (ResultadoVerificacionPlanMaterialAlmacenV2) GoString() string {
	return textoRedactadoMaterialAlmacenV2
}
func (r ResultadoVerificacionPlanMaterialAlmacenV2) Format(e fmt.State, _ rune) {
	formatoRedactadoMaterialV2(e)
}
func (ResultadoVerificacionPlanMaterialAlmacenV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*ResultadoVerificacionPlanMaterialAlmacenV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (ResultadoVerificacionPlanMaterialAlmacenV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*ResultadoVerificacionPlanMaterialAlmacenV2) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (ResultadoVerificacionPlanMaterialAlmacenV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*ResultadoVerificacionPlanMaterialAlmacenV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (r ResultadoVerificacionPlanMaterialAlmacenV2) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

func (SolicitudVerificarPerfilPublicadoMaterialV2) String() string {
	return textoRedactadoMaterialAlmacenV2
}
func (SolicitudVerificarPerfilPublicadoMaterialV2) GoString() string {
	return textoRedactadoMaterialAlmacenV2
}
func (s SolicitudVerificarPerfilPublicadoMaterialV2) Format(e fmt.State, _ rune) {
	formatoRedactadoMaterialV2(e)
}
func (SolicitudVerificarPerfilPublicadoMaterialV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*SolicitudVerificarPerfilPublicadoMaterialV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (SolicitudVerificarPerfilPublicadoMaterialV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*SolicitudVerificarPerfilPublicadoMaterialV2) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (SolicitudVerificarPerfilPublicadoMaterialV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*SolicitudVerificarPerfilPublicadoMaterialV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (s SolicitudVerificarPerfilPublicadoMaterialV2) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

func (SolicitudReservarReferenciaReciboMaterialV2) String() string {
	return textoRedactadoMaterialAlmacenV2
}
func (SolicitudReservarReferenciaReciboMaterialV2) GoString() string {
	return textoRedactadoMaterialAlmacenV2
}
func (s SolicitudReservarReferenciaReciboMaterialV2) Format(e fmt.State, _ rune) {
	formatoRedactadoMaterialV2(e)
}
func (SolicitudReservarReferenciaReciboMaterialV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*SolicitudReservarReferenciaReciboMaterialV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (SolicitudReservarReferenciaReciboMaterialV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*SolicitudReservarReferenciaReciboMaterialV2) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (SolicitudReservarReferenciaReciboMaterialV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*SolicitudReservarReferenciaReciboMaterialV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (s SolicitudReservarReferenciaReciboMaterialV2) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

func (ResultadoReferenciaReciboMaterialV2) String() string   { return textoRedactadoMaterialAlmacenV2 }
func (ResultadoReferenciaReciboMaterialV2) GoString() string { return textoRedactadoMaterialAlmacenV2 }
func (r ResultadoReferenciaReciboMaterialV2) Format(e fmt.State, _ rune) {
	formatoRedactadoMaterialV2(e)
}
func (ResultadoReferenciaReciboMaterialV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*ResultadoReferenciaReciboMaterialV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (ResultadoReferenciaReciboMaterialV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*ResultadoReferenciaReciboMaterialV2) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (ResultadoReferenciaReciboMaterialV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*ResultadoReferenciaReciboMaterialV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (r ResultadoReferenciaReciboMaterialV2) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

func (HuellaPlanMaterialAlmacenV2) String() string               { return textoRedactadoMaterialAlmacenV2 }
func (HuellaPlanMaterialAlmacenV2) GoString() string             { return textoRedactadoMaterialAlmacenV2 }
func (h HuellaPlanMaterialAlmacenV2) Format(e fmt.State, _ rune) { formatoRedactadoMaterialV2(e) }
func (HuellaPlanMaterialAlmacenV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*HuellaPlanMaterialAlmacenV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (HuellaPlanMaterialAlmacenV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*HuellaPlanMaterialAlmacenV2) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (HuellaPlanMaterialAlmacenV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*HuellaPlanMaterialAlmacenV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (h HuellaPlanMaterialAlmacenV2) LogValue() slog.Value { return slog.StringValue(h.String()) }

func (SolicitudAtestarMaterialAlmacenV2) String() string               { return textoRedactadoMaterialAlmacenV2 }
func (SolicitudAtestarMaterialAlmacenV2) GoString() string             { return textoRedactadoMaterialAlmacenV2 }
func (s SolicitudAtestarMaterialAlmacenV2) Format(e fmt.State, _ rune) { formatoRedactadoMaterialV2(e) }
func (SolicitudAtestarMaterialAlmacenV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*SolicitudAtestarMaterialAlmacenV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (SolicitudAtestarMaterialAlmacenV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*SolicitudAtestarMaterialAlmacenV2) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (SolicitudAtestarMaterialAlmacenV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*SolicitudAtestarMaterialAlmacenV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (s SolicitudAtestarMaterialAlmacenV2) LogValue() slog.Value { return slog.StringValue(s.String()) }

func (AtestacionCriptograficaMaterialAlmacenV2) String() string {
	return textoRedactadoMaterialAlmacenV2
}
func (AtestacionCriptograficaMaterialAlmacenV2) GoString() string {
	return textoRedactadoMaterialAlmacenV2
}
func (a AtestacionCriptograficaMaterialAlmacenV2) Format(e fmt.State, _ rune) {
	formatoRedactadoMaterialV2(e)
}
func (AtestacionCriptograficaMaterialAlmacenV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*AtestacionCriptograficaMaterialAlmacenV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (AtestacionCriptograficaMaterialAlmacenV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*AtestacionCriptograficaMaterialAlmacenV2) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (AtestacionCriptograficaMaterialAlmacenV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*AtestacionCriptograficaMaterialAlmacenV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (a AtestacionCriptograficaMaterialAlmacenV2) LogValue() slog.Value {
	return slog.StringValue(a.String())
}

func (SolicitudVerificarAtestacionMaterialAlmacenV2) String() string {
	return textoRedactadoMaterialAlmacenV2
}
func (SolicitudVerificarAtestacionMaterialAlmacenV2) GoString() string {
	return textoRedactadoMaterialAlmacenV2
}
func (s SolicitudVerificarAtestacionMaterialAlmacenV2) Format(e fmt.State, _ rune) {
	formatoRedactadoMaterialV2(e)
}
func (SolicitudVerificarAtestacionMaterialAlmacenV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*SolicitudVerificarAtestacionMaterialAlmacenV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (SolicitudVerificarAtestacionMaterialAlmacenV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*SolicitudVerificarAtestacionMaterialAlmacenV2) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (SolicitudVerificarAtestacionMaterialAlmacenV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*SolicitudVerificarAtestacionMaterialAlmacenV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (s SolicitudVerificarAtestacionMaterialAlmacenV2) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

func (PerfilCapacidadesAlmacenMaterialV2) String() string   { return textoRedactadoMaterialAlmacenV2 }
func (PerfilCapacidadesAlmacenMaterialV2) GoString() string { return textoRedactadoMaterialAlmacenV2 }
func (p PerfilCapacidadesAlmacenMaterialV2) Format(e fmt.State, _ rune) {
	formatoRedactadoMaterialV2(e)
}
func (PerfilCapacidadesAlmacenMaterialV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*PerfilCapacidadesAlmacenMaterialV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (PerfilCapacidadesAlmacenMaterialV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*PerfilCapacidadesAlmacenMaterialV2) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (PerfilCapacidadesAlmacenMaterialV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*PerfilCapacidadesAlmacenMaterialV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (p PerfilCapacidadesAlmacenMaterialV2) LogValue() slog.Value {
	return slog.StringValue(p.String())
}

func (InstantaneaObjetoMaterialV2) String() string               { return textoRedactadoMaterialAlmacenV2 }
func (InstantaneaObjetoMaterialV2) GoString() string             { return textoRedactadoMaterialAlmacenV2 }
func (i InstantaneaObjetoMaterialV2) Format(e fmt.State, _ rune) { formatoRedactadoMaterialV2(e) }
func (InstantaneaObjetoMaterialV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*InstantaneaObjetoMaterialV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (InstantaneaObjetoMaterialV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*InstantaneaObjetoMaterialV2) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (InstantaneaObjetoMaterialV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*InstantaneaObjetoMaterialV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (i InstantaneaObjetoMaterialV2) LogValue() slog.Value { return slog.StringValue(i.String()) }

func (ReciboEscrituraObjetoMaterialV2) String() string               { return textoRedactadoMaterialAlmacenV2 }
func (ReciboEscrituraObjetoMaterialV2) GoString() string             { return textoRedactadoMaterialAlmacenV2 }
func (r ReciboEscrituraObjetoMaterialV2) Format(e fmt.State, _ rune) { formatoRedactadoMaterialV2(e) }
func (ReciboEscrituraObjetoMaterialV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*ReciboEscrituraObjetoMaterialV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (ReciboEscrituraObjetoMaterialV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*ReciboEscrituraObjetoMaterialV2) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (ReciboEscrituraObjetoMaterialV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialAlmacenV2Prohibida
}
func (*ReciboEscrituraObjetoMaterialV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialAlmacenV2Prohibida
}
func (r ReciboEscrituraObjetoMaterialV2) LogValue() slog.Value { return slog.StringValue(r.String()) }
