package ports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
)

var (
	// ErrAutoridadObjetoEsperadoDocumentalV1NoValida no distingue entre una
	// prueba falsa, una referencia durable ausente y un cruce V4 incorrecto.
	ErrAutoridadObjetoEsperadoDocumentalV1NoValida = errors.New(
		"vec: autoridad de objeto esperado documental v1 no valida",
	)
	ErrSerializacionAutoridadObjetoEsperadoDocumentalV1Prohibida = errors.New(
		"vec: serializacion generica de autoridad de objeto esperado prohibida",
	)
)

const (
	EsquemaAutoridadObjetoEsperadoDocumentalV1        = "vec.documentos.autoridad-objeto-esperado.v1"
	VersionAutoridadObjetoEsperadoDocumentalV1 uint16 = 1
)

// AtestacionAutoridadObjetoEsperadoDocumentalV1 es material de verificacion,
// no una autoridad reconstruible. Codigo contiene una copia del autenticador;
// nunca una clave. El adaptador SQL debe verificarlo mediante la autoridad ya
// configurada o cotejarlo con un registro durable que lo haya verificado.
type AtestacionAutoridadObjetoEsperadoDocumentalV1 struct {
	Algoritmo    AlgoritmoAtestacionMaterialAlmacenV2
	ClaveRef     string
	ClaveVersion uint32
	Dominio      string
	Codigo       []byte
}

// ProyeccionAutoridadObjetoEsperadoDocumentalV1 contiene la entrada minima
// para el siguiente adaptador SQL. Es deliberadamente no autoritativa: solo
// PrepararRegistro la entrega, tras revalidar la referencia durable y la
// atestacion del recibo material.
type ProyeccionAutoridadObjetoEsperadoDocumentalV1 struct {
	Esquema                    string
	Version                    uint16
	ReciboMaterialRef          string
	HuellaReciboMaterialSHA256 string
	Objeto                     ReferenciaObjetoAlmacen
	ConectorID                 string
	EfectoRef                  string
	HuellaPlanEfectoSHA256     string
	HuellaManifiestoSHA256     string
	PasoRef                    PasoOperacionAlmacen
	HuellaPasoSHA256           string
	ReciboMaterialCanonico     []byte
	AtestacionReciboMaterial   AtestacionAutoridadObjetoEsperadoDocumentalV1
}

type datosAutoridadObjetoEsperadoDocumentalV1 struct {
	esquema                string
	version                uint16
	recibo                 ReciboEscrituraObjetoMaterialV2
	reciboRef              string
	huellaRecibo           [sha256.Size]byte
	objeto                 ReferenciaObjetoAlmacen
	conectorID             string
	efectoRef              string
	huellaPlanEfectoSHA256 string
	huellaManifiestoSHA256 string
	pasoRef                PasoOperacionAlmacen
	huellaPasoSHA256       string
}

// AutoridadObjetoEsperadoDocumentalV1 sella la pareja de objeto que la
// confirmacion durable debe esperar. No reserva ni inventa una referencia:
// deriva la pareja exclusivamente de un recibo material V2 verificable y la
// cruza con la declaracion V4 exacta que produjo el objeto.
type AutoridadObjetoEsperadoDocumentalV1 struct {
	datos *datosAutoridadObjetoEsperadoDocumentalV1
}

func NuevaAutoridadObjetoEsperadoDocumentalV1(
	ctx context.Context,
	declaracion DeclaracionEscrituraAlmacenDocumental,
	recibo ReciboEscrituraObjetoMaterialV2,
	verificadorReferencia VerificadorReferenciaReciboMaterialV2,
	verificadorAtestacion VerificadorAtestacionMaterialAlmacenV2,
) (AutoridadObjetoEsperadoDocumentalV1, error) {
	if ctx == nil || ctx.Err() != nil || declaracion.Validar() != nil || recibo.Validar() != nil ||
		dependenciaMaterialV2Nula(verificadorReferencia) ||
		dependenciaMaterialV2Nula(verificadorAtestacion) ||
		!reciboMaterialV2CotejaDeclaracionV4(recibo, declaracion) {
		return AutoridadObjetoEsperadoDocumentalV1{}, errorAutoridadObjetoEsperadoDocumentalV1()
	}
	if verificarReciboAutoridadObjetoEsperadoDocumentalV1(
		ctx, recibo, verificadorReferencia, verificadorAtestacion,
	) != nil {
		return AutoridadObjetoEsperadoDocumentalV1{}, errorAutoridadObjetoEsperadoDocumentalV1()
	}

	contexto := declaracion.datos.solicitud.Contexto
	huellaRecibo, err := recibo.HuellaSHA256()
	if err != nil {
		return AutoridadObjetoEsperadoDocumentalV1{}, errorAutoridadObjetoEsperadoDocumentalV1()
	}
	autoridad := AutoridadObjetoEsperadoDocumentalV1{
		datos: &datosAutoridadObjetoEsperadoDocumentalV1{
			esquema:   EsquemaAutoridadObjetoEsperadoDocumentalV1,
			version:   VersionAutoridadObjetoEsperadoDocumentalV1,
			recibo:    clonarReciboAutoridadObjetoEsperadoDocumentalV1(recibo),
			reciboRef: recibo.referenciaDurableOriginal, huellaRecibo: huellaRecibo,
			objeto:                 declaracion.datos.resultado.Objeto.Objeto,
			conectorID:             declaracion.datos.resultado.Objeto.ConectorID,
			efectoRef:              contexto.EfectoRef,
			huellaPlanEfectoSHA256: contexto.HuellaPlanEfectoSHA256,
			huellaManifiestoSHA256: contexto.HuellaManifiestoSHA256,
			pasoRef:                contexto.PasoRef, huellaPasoSHA256: contexto.HuellaPasoSHA256,
		},
	}
	if autoridad.Validar() != nil {
		return AutoridadObjetoEsperadoDocumentalV1{}, errorAutoridadObjetoEsperadoDocumentalV1()
	}
	return autoridad, nil
}

// Validar comprueba la clausura nominal del valor opaco. No reemplaza la
// comprobacion viva de la atestacion ni de la referencia durable.
func (a AutoridadObjetoEsperadoDocumentalV1) Validar() error {
	if a.datos == nil || validarDatosAutoridadObjetoEsperadoDocumentalV1(a.datos) != nil {
		return errorAutoridadObjetoEsperadoDocumentalV1()
	}
	return nil
}

// PrepararRegistro es la unica apertura hacia el futuro adaptador SQL. Antes
// de devolver copias vuelve a consultar el registro original y el verificador
// criptografico; una revocacion, sustitucion o cancelacion falla cerrada.
func (a AutoridadObjetoEsperadoDocumentalV1) PrepararRegistro(
	ctx context.Context,
	verificadorReferencia VerificadorReferenciaReciboMaterialV2,
	verificadorAtestacion VerificadorAtestacionMaterialAlmacenV2,
) (ProyeccionAutoridadObjetoEsperadoDocumentalV1, error) {
	if ctx == nil || ctx.Err() != nil || a.Validar() != nil ||
		dependenciaMaterialV2Nula(verificadorReferencia) ||
		dependenciaMaterialV2Nula(verificadorAtestacion) ||
		verificarReciboAutoridadObjetoEsperadoDocumentalV1(
			ctx, a.datos.recibo, verificadorReferencia, verificadorAtestacion,
		) != nil {
		return ProyeccionAutoridadObjetoEsperadoDocumentalV1{},
			errorAutoridadObjetoEsperadoDocumentalV1()
	}
	canonico, atestacion, err := materialVerificacionAutoridadObjetoEsperadoDocumentalV1(
		a.datos.recibo,
	)
	if err != nil || ctx.Err() != nil {
		return ProyeccionAutoridadObjetoEsperadoDocumentalV1{},
			errorAutoridadObjetoEsperadoDocumentalV1()
	}
	return ProyeccionAutoridadObjetoEsperadoDocumentalV1{
		Esquema: a.datos.esquema, Version: a.datos.version,
		ReciboMaterialRef:          a.datos.reciboRef,
		HuellaReciboMaterialSHA256: hex.EncodeToString(a.datos.huellaRecibo[:]),
		Objeto:                     a.datos.objeto, ConectorID: a.datos.conectorID,
		EfectoRef:              a.datos.efectoRef,
		HuellaPlanEfectoSHA256: a.datos.huellaPlanEfectoSHA256,
		HuellaManifiestoSHA256: a.datos.huellaManifiestoSHA256,
		PasoRef:                a.datos.pasoRef, HuellaPasoSHA256: a.datos.huellaPasoSHA256,
		ReciboMaterialCanonico:   append([]byte(nil), canonico...),
		AtestacionReciboMaterial: clonarAtestacionAutoridadObjetoEsperadoDocumentalV1(atestacion),
	}, nil
}

func validarDatosAutoridadObjetoEsperadoDocumentalV1(
	d *datosAutoridadObjetoEsperadoDocumentalV1,
) error {
	if d == nil || d.esquema != EsquemaAutoridadObjetoEsperadoDocumentalV1 ||
		d.version != VersionAutoridadObjetoEsperadoDocumentalV1 || d.recibo.Validar() != nil ||
		!referenciaOpacaAlmacenValida(d.reciboRef, 512) || d.reciboRef != d.recibo.referenciaDurableOriginal ||
		d.objeto.Validar() != nil || d.objeto.Referencia != d.recibo.instantanea.objetoRef ||
		d.objeto.Version != d.recibo.instantanea.objetoVersion ||
		!referenciaOpacaAlmacenValida(d.conectorID, 128) ||
		d.conectorID != d.recibo.instantanea.conectorLogicoID ||
		!referenciaOpacaAlmacenValida(d.efectoRef, 512) || d.efectoRef != d.recibo.efectoRef ||
		!esSHA256Hexadecimal(d.huellaPlanEfectoSHA256) ||
		!esSHA256Hexadecimal(d.huellaManifiestoSHA256) ||
		!esSHA256Hexadecimal(d.huellaPasoSHA256) || d.pasoRef == "" ||
		contieneComodinContextoAlmacen(d.efectoRef, string(d.pasoRef)) {
		return errorAutoridadObjetoEsperadoDocumentalV1()
	}
	huella, err := d.recibo.HuellaSHA256()
	if err != nil || !huellasMaterialV2Iguales(huella, d.huellaRecibo) {
		return errorAutoridadObjetoEsperadoDocumentalV1()
	}
	return nil
}

func reciboMaterialV2CotejaDeclaracionV4(
	recibo ReciboEscrituraObjetoMaterialV2,
	declaracion DeclaracionEscrituraAlmacenDocumental,
) bool {
	if recibo.Validar() != nil || declaracion.Validar() != nil || declaracion.datos == nil {
		return false
	}
	datos := declaracion.datos
	contexto := datos.solicitud.Contexto
	vinculo := datos.vinculoEjecucion.datos
	if vinculo == nil || !instantaneaMaterialV2CotejaObjeto(recibo.instantanea, datos.resultado.Objeto) {
		return false
	}
	return recibo.moduloID == contexto.ModuloID &&
		recibo.accionNegocio == contexto.AccionNegocio &&
		recibo.accionTecnica == contexto.AccionTecnica &&
		recibo.accionTecnica == AccionAlmacenEscribir &&
		recibo.recursoRef == contexto.RecursoRef &&
		recibo.operacionRef == contexto.OperacionRef &&
		recibo.cargaRef == contexto.CargaRef &&
		recibo.efectoRef == contexto.EfectoRef &&
		recibo.efectoRef == vinculo.EfectoRef &&
		recibo.clasificacion == contexto.Clasificacion &&
		recibo.instantanea.conectorLogicoID == datos.capacidades.ConectorID
}

func instantaneaMaterialV2CotejaObjeto(
	i InstantaneaObjetoMaterialV2,
	o ObjetoAlmacenado,
) bool {
	huella, err := decodificarSHA256MaterialV2(o.HuellaSHA256)
	estadoInmovilizacion := EstadoInmovilizacionMaterialNoAplicada
	if o.Inmovilizado {
		estadoInmovilizacion = EstadoInmovilizacionMaterialAplicada
	}
	return err == nil && i.Validar() == nil && o.Validar() == nil && !o.Eliminado &&
		i.conectorLogicoID == o.ConectorID && i.objetoRef == o.Objeto.Referencia &&
		i.objetoVersion == o.Objeto.Version && i.zona == o.Zona && i.mime == o.MIME &&
		i.tamano == o.Tamano && huellasMaterialV2Iguales(i.huellaContenido, huella) &&
		i.evidenciaCreacionRef == o.EvidenciaCreacionRef && i.almacenadoEn.Equal(o.AlmacenadoEn) &&
		i.tieneRetencion == !o.RetenidoHasta.IsZero() && i.retenidoHasta.Equal(o.RetenidoHasta) &&
		i.estadoInmovilizacion == estadoInmovilizacion && i.estadoObjeto == EstadoObjetoMaterialActivo
}

func verificarReciboAutoridadObjetoEsperadoDocumentalV1(
	ctx context.Context,
	recibo ReciboEscrituraObjetoMaterialV2,
	verificadorReferencia VerificadorReferenciaReciboMaterialV2,
	verificadorAtestacion VerificadorAtestacionMaterialAlmacenV2,
) error {
	if ctx == nil || ctx.Err() != nil || recibo.Validar() != nil ||
		dependenciaMaterialV2Nula(verificadorReferencia) ||
		dependenciaMaterialV2Nula(verificadorAtestacion) ||
		recibo.VerificarAtestacion(ctx, verificadorAtestacion) != nil || ctx.Err() != nil {
		return errorAutoridadObjetoEsperadoDocumentalV1()
	}
	identidad := recibo
	identidad.referenciaDurableOriginal = ""
	canonicoIdentidad, err := identidad.canonicoIdentidadDurable()
	if err != nil {
		return errorAutoridadObjetoEsperadoDocumentalV1()
	}
	solicitud, err := nuevaSolicitudReservarReferenciaReciboMaterialV2(canonicoIdentidad)
	if err != nil {
		return errorAutoridadObjetoEsperadoDocumentalV1()
	}
	resultado, err := NuevoResultadoReferenciaReciboMaterialV2(
		solicitud, recibo.referenciaDurableOriginal,
	)
	if err != nil || ctx.Err() != nil || verificadorReferencia.VerificarReferenciaReciboMaterialV2(
		ctx, solicitud, resultado,
	) != nil || ctx.Err() != nil {
		return errorAutoridadObjetoEsperadoDocumentalV1()
	}
	return nil
}

func materialVerificacionAutoridadObjetoEsperadoDocumentalV1(
	recibo ReciboEscrituraObjetoMaterialV2,
) ([]byte, AtestacionAutoridadObjetoEsperadoDocumentalV1, error) {
	canonico, err := recibo.BytesCanonicos()
	if err != nil {
		return nil, AtestacionAutoridadObjetoEsperadoDocumentalV1{},
			errorAutoridadObjetoEsperadoDocumentalV1()
	}
	solicitud, err := nuevaSolicitudAtestarMaterialAlmacenV2(
		dominioAtestacionReciboEscrituraMaterialV2, canonico,
	)
	peticion, errPeticion := nuevaSolicitudVerificarAtestacionMaterialAlmacenV2(
		solicitud, recibo.atestacion,
	)
	dominio, mensaje, algoritmo, claveRef, claveVersion, codigo, errRevelado :=
		peticion.RevelarParaVerificacion()
	if err != nil || errPeticion != nil || errRevelado != nil ||
		dominio != dominioAtestacionReciboEscrituraMaterialV2 || !bytes.Equal(mensaje, canonico) {
		return nil, AtestacionAutoridadObjetoEsperadoDocumentalV1{},
			errorAutoridadObjetoEsperadoDocumentalV1()
	}
	return append([]byte(nil), canonico...), AtestacionAutoridadObjetoEsperadoDocumentalV1{
		Algoritmo: algoritmo, ClaveRef: claveRef, ClaveVersion: claveVersion,
		Dominio: dominio, Codigo: append([]byte(nil), codigo...),
	}, nil
}

func clonarReciboAutoridadObjetoEsperadoDocumentalV1(
	recibo ReciboEscrituraObjetoMaterialV2,
) ReciboEscrituraObjetoMaterialV2 {
	copia := recibo
	copia.atestacion.codigo = append([]byte(nil), recibo.atestacion.codigo...)
	return copia
}

func clonarAtestacionAutoridadObjetoEsperadoDocumentalV1(
	a AtestacionAutoridadObjetoEsperadoDocumentalV1,
) AtestacionAutoridadObjetoEsperadoDocumentalV1 {
	a.Codigo = append([]byte(nil), a.Codigo...)
	return a
}

func errorAutoridadObjetoEsperadoDocumentalV1() error {
	return errors.Join(
		ErrAutoridadObjetoEsperadoDocumentalV1NoValida,
		ErrReciboEscrituraObjetoMaterialV2NoValido,
	)
}

const textoAutoridadObjetoEsperadoDocumentalV1Redactado = "[AUTORIDAD-OBJETO-ESPERADO-DOCUMENTAL-V1-OPACA-REDACTADA]"

func (AutoridadObjetoEsperadoDocumentalV1) String() string {
	return textoAutoridadObjetoEsperadoDocumentalV1Redactado
}
func (a AutoridadObjetoEsperadoDocumentalV1) GoString() string { return a.String() }
func (a AutoridadObjetoEsperadoDocumentalV1) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, a.String())
}
func (a AutoridadObjetoEsperadoDocumentalV1) LogValue() slog.Value {
	return slog.StringValue(a.String())
}
func (AutoridadObjetoEsperadoDocumentalV1) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAutoridadObjetoEsperadoDocumentalV1Prohibida
}
func (*AutoridadObjetoEsperadoDocumentalV1) UnmarshalJSON([]byte) error {
	return ErrSerializacionAutoridadObjetoEsperadoDocumentalV1Prohibida
}
func (AutoridadObjetoEsperadoDocumentalV1) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAutoridadObjetoEsperadoDocumentalV1Prohibida
}
func (*AutoridadObjetoEsperadoDocumentalV1) UnmarshalText([]byte) error {
	return ErrSerializacionAutoridadObjetoEsperadoDocumentalV1Prohibida
}
func (AutoridadObjetoEsperadoDocumentalV1) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionAutoridadObjetoEsperadoDocumentalV1Prohibida
}
func (*AutoridadObjetoEsperadoDocumentalV1) UnmarshalBinary([]byte) error {
	return ErrSerializacionAutoridadObjetoEsperadoDocumentalV1Prohibida
}

func (ProyeccionAutoridadObjetoEsperadoDocumentalV1) String() string {
	return textoAutoridadObjetoEsperadoDocumentalV1Redactado
}
func (p ProyeccionAutoridadObjetoEsperadoDocumentalV1) GoString() string { return p.String() }
func (p ProyeccionAutoridadObjetoEsperadoDocumentalV1) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}
func (p ProyeccionAutoridadObjetoEsperadoDocumentalV1) LogValue() slog.Value {
	return slog.StringValue(p.String())
}
func (ProyeccionAutoridadObjetoEsperadoDocumentalV1) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAutoridadObjetoEsperadoDocumentalV1Prohibida
}
func (*ProyeccionAutoridadObjetoEsperadoDocumentalV1) UnmarshalJSON([]byte) error {
	return ErrSerializacionAutoridadObjetoEsperadoDocumentalV1Prohibida
}
func (ProyeccionAutoridadObjetoEsperadoDocumentalV1) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAutoridadObjetoEsperadoDocumentalV1Prohibida
}
func (*ProyeccionAutoridadObjetoEsperadoDocumentalV1) UnmarshalText([]byte) error {
	return ErrSerializacionAutoridadObjetoEsperadoDocumentalV1Prohibida
}
func (ProyeccionAutoridadObjetoEsperadoDocumentalV1) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionAutoridadObjetoEsperadoDocumentalV1Prohibida
}
func (*ProyeccionAutoridadObjetoEsperadoDocumentalV1) UnmarshalBinary([]byte) error {
	return ErrSerializacionAutoridadObjetoEsperadoDocumentalV1Prohibida
}
