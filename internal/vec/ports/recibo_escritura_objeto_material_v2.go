package ports

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	recibomaterial "vec-diputacion-granada/internal/vec/canonico/recibomaterial"
)

var (
	// ErrReciboEscrituraObjetoMaterialV2NoValido es deliberadamente opaco:
	// un dato ausente, un cruce de contexto y una prueba criptografica falsa
	// producen la misma denegacion cerrada.
	ErrReciboEscrituraObjetoMaterialV2NoValido = recibomaterial.ErrReciboNoValido
	ErrAtestacionMaterialAlmacenV2NoValida     = recibomaterial.ErrAtestacionNoValida
	ErrSerializacionMaterialAlmacenV2Prohibida = recibomaterial.ErrSerializacionProhibida
)

const (
	EsquemaPerfilCapacidadesAlmacenMaterialV2 = recibomaterial.EsquemaPerfil
	EsquemaInstantaneaObjetoMaterialV2        = recibomaterial.EsquemaInstantanea
	EsquemaReciboEscrituraObjetoMaterialV2    = recibomaterial.EsquemaRecibo

	VersionEsquemaMaterialAlmacenV2 uint16 = recibomaterial.EsquemaVersion

	dominioAtestacionPerfilCapacidadesMaterialV2 = recibomaterial.DominioPerfil
	dominioAtestacionReciboEscrituraMaterialV2   = recibomaterial.DominioRecibo
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
	return recibomaterial.SeleccionPlanValida(datosSeleccionPlanMaterialV2(s))
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
	return recibomaterial.HuellaPlanValida(datosHuellaPlanMaterialV2(h))
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
	canonico, err := recibomaterial.CanonicoVinculoPlan(datosVinculoPlanMaterialV2(s))
	if err != nil {
		return nil, errorReciboMaterialV2()
	}
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
	if !recibomaterial.SolicitudPlanValida(datosVinculoPlanMaterialV2(s), s.huellaVinculo) {
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
	if !recibomaterial.ResultadoPlanValido(
		datosVinculoPlanMaterialV2(solicitud), solicitud.huellaVinculo, r.huellaPlan, r.huellaVinculo,
	) {
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
	preparada, err := recibomaterial.PrepararSolicitudAtestacion(dominio, mensaje)
	if err != nil {
		return SolicitudAtestarMaterialAlmacenV2{}, errorAtestacionMaterialV2()
	}
	return SolicitudAtestarMaterialAlmacenV2{
		dominio: preparada.Dominio, mensaje: preparada.Mensaje, huella: preparada.Huella,
	}, nil
}

func (s SolicitudAtestarMaterialAlmacenV2) validar() error {
	if _, err := recibomaterial.RevelarSolicitudAtestacion(datosSolicitudAtestacionMaterialV2(s)); err != nil {
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
	revelada, err := recibomaterial.RevelarSolicitudAtestacion(datosSolicitudAtestacionMaterialV2(s))
	if err != nil {
		return "", nil, [sha256.Size]byte{}, errorAtestacionMaterialV2()
	}
	return revelada.Dominio, revelada.Mensaje, revelada.Huella, nil
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
	datos, err := recibomaterial.NuevaAtestacionNominal(
		datosSolicitudAtestacionMaterialV2(solicitud),
		recibomaterial.DatosAtestacion{
			Algoritmo: string(algoritmo), ClaveRef: claveRef, ClaveVersion: claveVersion,
			Dominio: solicitud.dominio, Huella: solicitud.huella, Codigo: codigo,
		},
	)
	if err != nil {
		return AtestacionCriptograficaMaterialAlmacenV2{}, errorAtestacionMaterialV2()
	}
	return atestacionMaterialV2DesdeDatos(datos), nil
}

func (a AtestacionCriptograficaMaterialAlmacenV2) validarPara(
	solicitud SolicitudAtestarMaterialAlmacenV2,
) error {
	if _, _, err := recibomaterial.RevelarVerificacionAtestacion(
		datosSolicitudAtestacionMaterialV2(solicitud), datosAtestacionMaterialV2(a),
	); err != nil {
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
	solicitud, atestacion, err := recibomaterial.RevelarVerificacionAtestacion(
		datosSolicitudAtestacionMaterialV2(s.solicitud), datosAtestacionMaterialV2(s.atestacion),
	)
	if err != nil {
		return "", nil, "", "", 0, nil, errorAtestacionMaterialV2()
	}
	return solicitud.Dominio, solicitud.Mensaje, AlgoritmoAtestacionMaterialAlmacenV2(atestacion.Algoritmo),
		atestacion.ClaveRef, atestacion.ClaveVersion, atestacion.Codigo, nil
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
	datos, err := recibomaterial.RevelarPerfilPublicado(datosPerfilPublicadoMaterialV2(s))
	if err != nil {
		return "", 0, "", [sha256.Size]byte{}, nil, errorReciboMaterialV2()
	}
	return datos.Referencia, datos.Version, datos.ConectorLogicoID, datos.Huella, datos.Canonico, nil
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
	if !recibomaterial.PerfilValido(datosPerfilMaterialV2(p)) {
		return errorReciboMaterialV2()
	}
	return nil
}

func (p PerfilCapacidadesAlmacenMaterialV2) Validar() error {
	if !recibomaterial.PerfilSelladoNominalValido(
		datosPerfilMaterialV2(p), p.huella, datosAtestacionMaterialV2(p.atestacion),
	) {
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
	if p.Validar() != nil || !recibomaterial.PerfilCotejaCapacidades(datosPerfilMaterialV2(p), capacidades) {
		return errorReciboMaterialV2()
	}
	return nil
}

func (p PerfilCapacidadesAlmacenMaterialV2) canonicoSinAtestacion() ([]byte, error) {
	canonico, err := recibomaterial.CanonicoPerfil(datosPerfilMaterialV2(p))
	if err != nil {
		return nil, errorReciboMaterialV2()
	}
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
	if !recibomaterial.InstantaneaValida(datosInstantaneaMaterialV2(i)) {
		return errorReciboMaterialV2()
	}
	return nil
}

func (i InstantaneaObjetoMaterialV2) canonico() ([]byte, error) {
	canonico, err := recibomaterial.CanonicoInstantanea(datosInstantaneaMaterialV2(i))
	if err != nil {
		return nil, errorReciboMaterialV2()
	}
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
	return recibomaterial.SumaSHA256(canonico), nil
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
	huella, err := recibomaterial.NuevaHuellaIdentidad(canonicoIdentidad)
	if err != nil {
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
	if !recibomaterial.ResultadoReferenciaValido(
		solicitud.huellaIdentidad,
		recibomaterial.DatosResultadoReferencia{Referencia: r.referencia, HuellaIdentidad: r.huellaIdentidad},
	) {
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

func (r ReciboEscrituraObjetoMaterialV2) Validar() error {
	if !recibomaterial.ReciboSelladoNominalValido(
		datosReciboMaterialV2(r), r.huella, datosAtestacionMaterialV2(r.atestacion),
	) {
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
	canonico, err := recibomaterial.CanonicoRecibo(datosReciboMaterialV2(r))
	if err != nil {
		return nil, errorReciboMaterialV2()
	}
	return canonico, nil
}

func (r ReciboEscrituraObjetoMaterialV2) canonicoIdentidadDurable() ([]byte, error) {
	canonico, err := recibomaterial.CanonicoIdentidadDurable(datosReciboMaterialV2(r))
	if err != nil {
		return nil, errorReciboMaterialV2()
	}
	return canonico, nil
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

func datosSeleccionPlanMaterialV2(s SeleccionPlanMaterialAlmacenV2) recibomaterial.SeleccionPlan {
	return recibomaterial.SeleccionPlan{Referencia: s.referencia, Version: s.version}
}

func datosHuellaPlanMaterialV2(h HuellaPlanMaterialAlmacenV2) recibomaterial.HuellaPlan {
	return recibomaterial.HuellaPlan{
		Referencia: h.referencia, Version: h.version, Suma: h.suma, HuellaVinculo: h.huellaVinculo,
	}
}

func datosHechosContextoMaterialV2(p ProyeccionContextoOperacionAlmacen) recibomaterial.HechosContexto {
	return recibomaterial.HechosContexto{
		ModuloID: p.ModuloID, AccionNegocio: p.AccionNegocio, AccionTecnica: p.AccionTecnica,
		RecursoRef: p.RecursoRef, OperacionRef: p.OperacionRef, CargaRef: p.CargaRef,
		EfectoRef: p.EfectoRef, Clasificacion: p.Clasificacion,
	}
}

func datosVinculoPlanMaterialV2(s SolicitudVerificarPlanMaterialAlmacenV2) recibomaterial.VinculoPlan {
	return recibomaterial.VinculoPlan{
		Seleccion: datosSeleccionPlanMaterialV2(s.seleccion), ConectorLogicoID: s.conectorLogicoID,
		Hechos: recibomaterial.HechosContexto{
			ModuloID: s.moduloID, AccionNegocio: s.accionNegocio, AccionTecnica: s.accionTecnica,
			RecursoRef: s.recursoRef, OperacionRef: s.operacionRef, CargaRef: s.cargaRef,
			EfectoRef: s.efectoRef, Clasificacion: s.clasificacion,
		},
	}
}

func datosSolicitudAtestacionMaterialV2(s SolicitudAtestarMaterialAlmacenV2) recibomaterial.DatosSolicitudAtestacion {
	return recibomaterial.DatosSolicitudAtestacion{Dominio: s.dominio, Mensaje: s.mensaje, Huella: s.huella}
}

func datosAtestacionMaterialV2(a AtestacionCriptograficaMaterialAlmacenV2) recibomaterial.DatosAtestacion {
	return recibomaterial.DatosAtestacion{
		Algoritmo: string(a.algoritmo), ClaveRef: a.claveRef, ClaveVersion: a.claveVersion,
		Dominio: a.dominio, Huella: a.huella, Codigo: a.codigo,
	}
}

func atestacionMaterialV2DesdeDatos(a recibomaterial.DatosAtestacion) AtestacionCriptograficaMaterialAlmacenV2 {
	return AtestacionCriptograficaMaterialAlmacenV2{
		algoritmo: AlgoritmoAtestacionMaterialAlmacenV2(a.Algoritmo), claveRef: a.ClaveRef,
		claveVersion: a.ClaveVersion, dominio: a.Dominio, huella: a.Huella, codigo: a.Codigo,
	}
}

func datosPerfilMaterialV2(p PerfilCapacidadesAlmacenMaterialV2) recibomaterial.Perfil {
	return recibomaterial.Perfil{
		Esquema: p.esquema, VersionEsquema: p.versionEsquema, Referencia: p.referencia,
		Version: p.version, ConectorLogicoID: p.conectorLogicoID, EscrituraEnFlujo: p.escrituraEnFlujo,
		ReferenciasOpacas: p.referenciasOpacas, IntegridadSHA256: p.integridadSHA256,
		Versionado: p.versionado, Retencion: p.retencion, BloqueoLegal: p.bloqueoLegal,
		CifradoEnTransito: p.cifradoEnTransito, CifradoEnReposo: p.cifradoEnReposo,
		CifradoPorObjeto: p.cifradoPorObjeto, PreservaObjetoOriginal: p.preservaObjetoOriginal,
		TamanoMaximoObjeto: p.tamanoMaximoObjeto,
	}
}

func datosPerfilPublicadoMaterialV2(s SolicitudVerificarPerfilPublicadoMaterialV2) recibomaterial.DatosPerfilPublicado {
	return recibomaterial.DatosPerfilPublicado{
		Referencia: s.referencia, Version: s.version, ConectorLogicoID: s.conectorLogicoID,
		Huella: s.huella, Canonico: s.canonico,
	}
}

func datosInstantaneaMaterialV2(i InstantaneaObjetoMaterialV2) recibomaterial.Instantanea {
	return recibomaterial.Instantanea{
		Esquema: i.esquema, VersionEsquema: i.versionEsquema, ConectorLogicoID: i.conectorLogicoID,
		ObjetoRef: i.objetoRef, ObjetoVersion: i.objetoVersion, Zona: i.zona, MIME: i.mime,
		Tamano: i.tamano, HuellaContenido: i.huellaContenido, EvidenciaCreacionRef: i.evidenciaCreacionRef,
		AlmacenadoEn: i.almacenadoEn, TieneRetencion: i.tieneRetencion, RetenidoHasta: i.retenidoHasta,
		EstadoInmovilizacion: string(i.estadoInmovilizacion), EstadoObjeto: string(i.estadoObjeto),
	}
}

func datosReciboMaterialV2(r ReciboEscrituraObjetoMaterialV2) recibomaterial.Recibo {
	return recibomaterial.Recibo{
		Esquema: r.esquema, VersionEsquema: r.versionEsquema,
		ReferenciaDurableOriginal: r.referenciaDurableOriginal,
		PerfilReferencia:          r.perfilReferencia, PerfilVersion: r.perfilVersion, HuellaPerfil: r.huellaPerfil,
		Hechos: recibomaterial.HechosContexto{
			ModuloID: r.moduloID, AccionNegocio: r.accionNegocio, AccionTecnica: r.accionTecnica,
			RecursoRef: r.recursoRef, OperacionRef: r.operacionRef, CargaRef: r.cargaRef,
			EfectoRef: r.efectoRef, Clasificacion: r.clasificacion,
		},
		HuellaPlan:  datosHuellaPlanMaterialV2(r.huellaPlanMaterial),
		Instantanea: datosInstantaneaMaterialV2(r.instantanea),
	}
}

func hechosEstablesContextoMaterialV2Validos(p ProyeccionContextoOperacionAlmacen) bool {
	return recibomaterial.HechosContextoValidos(datosHechosContextoMaterialV2(p))
}

func aliasLogicoMaterialV2Valido(valor string, maximo int) bool {
	return recibomaterial.AliasLogicoValido(valor, maximo)
}

func decodificarSHA256MaterialV2(valor string) ([sha256.Size]byte, error) {
	resultado, err := recibomaterial.DecodificarSHA256(valor)
	if err != nil {
		return [sha256.Size]byte{}, errorReciboMaterialV2()
	}
	return resultado, nil
}

func huellasMaterialV2Iguales(
	primera [sha256.Size]byte,
	segunda [sha256.Size]byte,
) bool {
	return recibomaterial.HuellasIguales(primera, segunda)
}

func dependenciaMaterialV2Nula(dependencia any) bool {
	return recibomaterial.DependenciaNula(dependencia)
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

const textoRedactadoMaterialAlmacenV2 = recibomaterial.TextoRedactado

func formatoRedactadoMaterialV2(estado fmt.State) { recibomaterial.FormatoRedactado(estado) }
func serializacionMaterialV2Prohibida() ([]byte, error) {
	return recibomaterial.SerializacionProhibida()
}
func deserializacionMaterialV2Prohibida() error { return recibomaterial.DeserializacionProhibida() }
func valorLogRedactadoMaterialV2() slog.Value {
	return slog.StringValue(textoRedactadoMaterialAlmacenV2)
}

// Los metodos permanecen declarados sobre cada tipo para conservar su forma,
// tamaño y reflexión. Las funciones comunes solo centralizan el resultado.
func (SeleccionPlanMaterialAlmacenV2) String() string             { return textoRedactadoMaterialAlmacenV2 }
func (SeleccionPlanMaterialAlmacenV2) GoString() string           { return textoRedactadoMaterialAlmacenV2 }
func (SeleccionPlanMaterialAlmacenV2) Format(e fmt.State, _ rune) { formatoRedactadoMaterialV2(e) }
func (SeleccionPlanMaterialAlmacenV2) MarshalJSON() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*SeleccionPlanMaterialAlmacenV2) UnmarshalJSON([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (SeleccionPlanMaterialAlmacenV2) MarshalText() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*SeleccionPlanMaterialAlmacenV2) UnmarshalText([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (SeleccionPlanMaterialAlmacenV2) MarshalBinary() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*SeleccionPlanMaterialAlmacenV2) UnmarshalBinary([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (SeleccionPlanMaterialAlmacenV2) LogValue() slog.Value { return valorLogRedactadoMaterialV2() }

func (SolicitudVerificarPlanMaterialAlmacenV2) String() string {
	return textoRedactadoMaterialAlmacenV2
}
func (SolicitudVerificarPlanMaterialAlmacenV2) GoString() string {
	return textoRedactadoMaterialAlmacenV2
}
func (SolicitudVerificarPlanMaterialAlmacenV2) Format(e fmt.State, _ rune) {
	formatoRedactadoMaterialV2(e)
}
func (SolicitudVerificarPlanMaterialAlmacenV2) MarshalJSON() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*SolicitudVerificarPlanMaterialAlmacenV2) UnmarshalJSON([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (SolicitudVerificarPlanMaterialAlmacenV2) MarshalText() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*SolicitudVerificarPlanMaterialAlmacenV2) UnmarshalText([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (SolicitudVerificarPlanMaterialAlmacenV2) MarshalBinary() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*SolicitudVerificarPlanMaterialAlmacenV2) UnmarshalBinary([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (SolicitudVerificarPlanMaterialAlmacenV2) LogValue() slog.Value {
	return valorLogRedactadoMaterialV2()
}

func (ResultadoVerificacionPlanMaterialAlmacenV2) String() string {
	return textoRedactadoMaterialAlmacenV2
}
func (ResultadoVerificacionPlanMaterialAlmacenV2) GoString() string {
	return textoRedactadoMaterialAlmacenV2
}
func (ResultadoVerificacionPlanMaterialAlmacenV2) Format(e fmt.State, _ rune) {
	formatoRedactadoMaterialV2(e)
}
func (ResultadoVerificacionPlanMaterialAlmacenV2) MarshalJSON() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*ResultadoVerificacionPlanMaterialAlmacenV2) UnmarshalJSON([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (ResultadoVerificacionPlanMaterialAlmacenV2) MarshalText() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*ResultadoVerificacionPlanMaterialAlmacenV2) UnmarshalText([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (ResultadoVerificacionPlanMaterialAlmacenV2) MarshalBinary() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*ResultadoVerificacionPlanMaterialAlmacenV2) UnmarshalBinary([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (ResultadoVerificacionPlanMaterialAlmacenV2) LogValue() slog.Value {
	return valorLogRedactadoMaterialV2()
}

func (SolicitudAtestarMaterialAlmacenV2) String() string             { return textoRedactadoMaterialAlmacenV2 }
func (SolicitudAtestarMaterialAlmacenV2) GoString() string           { return textoRedactadoMaterialAlmacenV2 }
func (SolicitudAtestarMaterialAlmacenV2) Format(e fmt.State, _ rune) { formatoRedactadoMaterialV2(e) }
func (SolicitudAtestarMaterialAlmacenV2) MarshalJSON() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*SolicitudAtestarMaterialAlmacenV2) UnmarshalJSON([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (SolicitudAtestarMaterialAlmacenV2) MarshalText() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*SolicitudAtestarMaterialAlmacenV2) UnmarshalText([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (SolicitudAtestarMaterialAlmacenV2) MarshalBinary() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*SolicitudAtestarMaterialAlmacenV2) UnmarshalBinary([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (SolicitudAtestarMaterialAlmacenV2) LogValue() slog.Value { return valorLogRedactadoMaterialV2() }

func (AtestacionCriptograficaMaterialAlmacenV2) String() string {
	return textoRedactadoMaterialAlmacenV2
}
func (AtestacionCriptograficaMaterialAlmacenV2) GoString() string {
	return textoRedactadoMaterialAlmacenV2
}
func (AtestacionCriptograficaMaterialAlmacenV2) Format(e fmt.State, _ rune) {
	formatoRedactadoMaterialV2(e)
}
func (AtestacionCriptograficaMaterialAlmacenV2) MarshalJSON() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*AtestacionCriptograficaMaterialAlmacenV2) UnmarshalJSON([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (AtestacionCriptograficaMaterialAlmacenV2) MarshalText() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*AtestacionCriptograficaMaterialAlmacenV2) UnmarshalText([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (AtestacionCriptograficaMaterialAlmacenV2) MarshalBinary() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*AtestacionCriptograficaMaterialAlmacenV2) UnmarshalBinary([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (AtestacionCriptograficaMaterialAlmacenV2) LogValue() slog.Value {
	return valorLogRedactadoMaterialV2()
}

func (SolicitudVerificarAtestacionMaterialAlmacenV2) String() string {
	return textoRedactadoMaterialAlmacenV2
}
func (SolicitudVerificarAtestacionMaterialAlmacenV2) GoString() string {
	return textoRedactadoMaterialAlmacenV2
}
func (SolicitudVerificarAtestacionMaterialAlmacenV2) Format(e fmt.State, _ rune) {
	formatoRedactadoMaterialV2(e)
}
func (SolicitudVerificarAtestacionMaterialAlmacenV2) MarshalJSON() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*SolicitudVerificarAtestacionMaterialAlmacenV2) UnmarshalJSON([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (SolicitudVerificarAtestacionMaterialAlmacenV2) MarshalText() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*SolicitudVerificarAtestacionMaterialAlmacenV2) UnmarshalText([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (SolicitudVerificarAtestacionMaterialAlmacenV2) MarshalBinary() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*SolicitudVerificarAtestacionMaterialAlmacenV2) UnmarshalBinary([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (SolicitudVerificarAtestacionMaterialAlmacenV2) LogValue() slog.Value {
	return valorLogRedactadoMaterialV2()
}

func (SolicitudVerificarPerfilPublicadoMaterialV2) String() string {
	return textoRedactadoMaterialAlmacenV2
}
func (SolicitudVerificarPerfilPublicadoMaterialV2) GoString() string {
	return textoRedactadoMaterialAlmacenV2
}
func (SolicitudVerificarPerfilPublicadoMaterialV2) Format(e fmt.State, _ rune) {
	formatoRedactadoMaterialV2(e)
}
func (SolicitudVerificarPerfilPublicadoMaterialV2) MarshalJSON() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*SolicitudVerificarPerfilPublicadoMaterialV2) UnmarshalJSON([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (SolicitudVerificarPerfilPublicadoMaterialV2) MarshalText() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*SolicitudVerificarPerfilPublicadoMaterialV2) UnmarshalText([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (SolicitudVerificarPerfilPublicadoMaterialV2) MarshalBinary() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*SolicitudVerificarPerfilPublicadoMaterialV2) UnmarshalBinary([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (SolicitudVerificarPerfilPublicadoMaterialV2) LogValue() slog.Value {
	return valorLogRedactadoMaterialV2()
}

func (SolicitudReservarReferenciaReciboMaterialV2) String() string {
	return textoRedactadoMaterialAlmacenV2
}
func (SolicitudReservarReferenciaReciboMaterialV2) GoString() string {
	return textoRedactadoMaterialAlmacenV2
}
func (SolicitudReservarReferenciaReciboMaterialV2) Format(e fmt.State, _ rune) {
	formatoRedactadoMaterialV2(e)
}
func (SolicitudReservarReferenciaReciboMaterialV2) MarshalJSON() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*SolicitudReservarReferenciaReciboMaterialV2) UnmarshalJSON([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (SolicitudReservarReferenciaReciboMaterialV2) MarshalText() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*SolicitudReservarReferenciaReciboMaterialV2) UnmarshalText([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (SolicitudReservarReferenciaReciboMaterialV2) MarshalBinary() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*SolicitudReservarReferenciaReciboMaterialV2) UnmarshalBinary([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (SolicitudReservarReferenciaReciboMaterialV2) LogValue() slog.Value {
	return valorLogRedactadoMaterialV2()
}

func (ResultadoReferenciaReciboMaterialV2) String() string             { return textoRedactadoMaterialAlmacenV2 }
func (ResultadoReferenciaReciboMaterialV2) GoString() string           { return textoRedactadoMaterialAlmacenV2 }
func (ResultadoReferenciaReciboMaterialV2) Format(e fmt.State, _ rune) { formatoRedactadoMaterialV2(e) }
func (ResultadoReferenciaReciboMaterialV2) MarshalJSON() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*ResultadoReferenciaReciboMaterialV2) UnmarshalJSON([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (ResultadoReferenciaReciboMaterialV2) MarshalText() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*ResultadoReferenciaReciboMaterialV2) UnmarshalText([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (ResultadoReferenciaReciboMaterialV2) MarshalBinary() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*ResultadoReferenciaReciboMaterialV2) UnmarshalBinary([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (ResultadoReferenciaReciboMaterialV2) LogValue() slog.Value {
	return valorLogRedactadoMaterialV2()
}

func (HuellaPlanMaterialAlmacenV2) String() string             { return textoRedactadoMaterialAlmacenV2 }
func (HuellaPlanMaterialAlmacenV2) GoString() string           { return textoRedactadoMaterialAlmacenV2 }
func (HuellaPlanMaterialAlmacenV2) Format(e fmt.State, _ rune) { formatoRedactadoMaterialV2(e) }
func (HuellaPlanMaterialAlmacenV2) MarshalJSON() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*HuellaPlanMaterialAlmacenV2) UnmarshalJSON([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (HuellaPlanMaterialAlmacenV2) MarshalText() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*HuellaPlanMaterialAlmacenV2) UnmarshalText([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (HuellaPlanMaterialAlmacenV2) MarshalBinary() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*HuellaPlanMaterialAlmacenV2) UnmarshalBinary([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (HuellaPlanMaterialAlmacenV2) LogValue() slog.Value { return valorLogRedactadoMaterialV2() }

func (PerfilCapacidadesAlmacenMaterialV2) String() string             { return textoRedactadoMaterialAlmacenV2 }
func (PerfilCapacidadesAlmacenMaterialV2) GoString() string           { return textoRedactadoMaterialAlmacenV2 }
func (PerfilCapacidadesAlmacenMaterialV2) Format(e fmt.State, _ rune) { formatoRedactadoMaterialV2(e) }
func (PerfilCapacidadesAlmacenMaterialV2) MarshalJSON() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*PerfilCapacidadesAlmacenMaterialV2) UnmarshalJSON([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (PerfilCapacidadesAlmacenMaterialV2) MarshalText() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*PerfilCapacidadesAlmacenMaterialV2) UnmarshalText([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (PerfilCapacidadesAlmacenMaterialV2) MarshalBinary() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*PerfilCapacidadesAlmacenMaterialV2) UnmarshalBinary([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (PerfilCapacidadesAlmacenMaterialV2) LogValue() slog.Value { return valorLogRedactadoMaterialV2() }

func (InstantaneaObjetoMaterialV2) String() string             { return textoRedactadoMaterialAlmacenV2 }
func (InstantaneaObjetoMaterialV2) GoString() string           { return textoRedactadoMaterialAlmacenV2 }
func (InstantaneaObjetoMaterialV2) Format(e fmt.State, _ rune) { formatoRedactadoMaterialV2(e) }
func (InstantaneaObjetoMaterialV2) MarshalJSON() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*InstantaneaObjetoMaterialV2) UnmarshalJSON([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (InstantaneaObjetoMaterialV2) MarshalText() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*InstantaneaObjetoMaterialV2) UnmarshalText([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (InstantaneaObjetoMaterialV2) MarshalBinary() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*InstantaneaObjetoMaterialV2) UnmarshalBinary([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (InstantaneaObjetoMaterialV2) LogValue() slog.Value { return valorLogRedactadoMaterialV2() }

func (ReciboEscrituraObjetoMaterialV2) String() string             { return textoRedactadoMaterialAlmacenV2 }
func (ReciboEscrituraObjetoMaterialV2) GoString() string           { return textoRedactadoMaterialAlmacenV2 }
func (ReciboEscrituraObjetoMaterialV2) Format(e fmt.State, _ rune) { formatoRedactadoMaterialV2(e) }
func (ReciboEscrituraObjetoMaterialV2) MarshalJSON() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*ReciboEscrituraObjetoMaterialV2) UnmarshalJSON([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (ReciboEscrituraObjetoMaterialV2) MarshalText() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*ReciboEscrituraObjetoMaterialV2) UnmarshalText([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (ReciboEscrituraObjetoMaterialV2) MarshalBinary() ([]byte, error) {
	return serializacionMaterialV2Prohibida()
}
func (*ReciboEscrituraObjetoMaterialV2) UnmarshalBinary([]byte) error {
	return deserializacionMaterialV2Prohibida()
}
func (ReciboEscrituraObjetoMaterialV2) LogValue() slog.Value { return valorLogRedactadoMaterialV2() }
