package ports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
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
	HuellaDeclaracionV4SHA256  string
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
	atestacionRecibo       AtestacionAutoridadObjetoEsperadoDocumentalV1
	selloAtestacionRecibo  [sha256.Size]byte
	declaracion            DeclaracionEscrituraAlmacenDocumental
	huellaDeclaracionV4    [sha256.Size]byte
	contextoV4             ProyeccionContextoOperacionAlmacen
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

	reciboSellado := clonarReciboAutoridadObjetoEsperadoDocumentalV1(recibo)
	declaracionSellada := clonarDeclaracionAutoridadObjetoEsperadoDocumentalV1(declaracion)
	if reciboSellado.Validar() != nil || declaracionSellada.Validar() != nil ||
		!reciboMaterialV2CotejaDeclaracionV4(reciboSellado, declaracionSellada) {
		return AutoridadObjetoEsperadoDocumentalV1{}, errorAutoridadObjetoEsperadoDocumentalV1()
	}

	contexto := declaracionSellada.datos.solicitud.Contexto
	huellaRecibo, err := reciboSellado.HuellaSHA256()
	canonicoRecibo, atestacionRecibo, errAtestacion :=
		materialVerificacionAutoridadObjetoEsperadoDocumentalV1(reciboSellado)
	selloAtestacion, errSello :=
		selloAtestacionAutoridadObjetoEsperadoDocumentalV1(canonicoRecibo, atestacionRecibo)
	huellaDeclaracion, errDeclaracion :=
		huellaDeclaracionAutoridadObjetoEsperadoDocumentalV1(declaracionSellada)
	if err != nil || errAtestacion != nil || errSello != nil || errDeclaracion != nil {
		return AutoridadObjetoEsperadoDocumentalV1{}, errorAutoridadObjetoEsperadoDocumentalV1()
	}
	instantanea := reciboSellado.instantanea
	autoridad := AutoridadObjetoEsperadoDocumentalV1{
		datos: &datosAutoridadObjetoEsperadoDocumentalV1{
			esquema:   EsquemaAutoridadObjetoEsperadoDocumentalV1,
			version:   VersionAutoridadObjetoEsperadoDocumentalV1,
			recibo:    reciboSellado,
			reciboRef: reciboSellado.referenciaDurableOriginal, huellaRecibo: huellaRecibo,
			atestacionRecibo:      clonarAtestacionAutoridadObjetoEsperadoDocumentalV1(atestacionRecibo),
			selloAtestacionRecibo: selloAtestacion,
			declaracion:           declaracionSellada,
			huellaDeclaracionV4:   huellaDeclaracion,
			contextoV4:            contexto,
			objeto: ReferenciaObjetoAlmacen{
				Referencia: instantanea.objetoRef, Version: instantanea.objetoVersion,
			},
			conectorID:             instantanea.conectorLogicoID,
			efectoRef:              contexto.EfectoRef,
			huellaPlanEfectoSHA256: contexto.HuellaPlanEfectoSHA256,
			huellaManifiestoSHA256: contexto.HuellaManifiestoSHA256,
			pasoRef:                contexto.PasoRef, huellaPasoSHA256: contexto.HuellaPasoSHA256,
		},
	}
	if autoridad.Validar() != nil || verificarReciboAutoridadObjetoEsperadoDocumentalV1(
		ctx, autoridad.datos.recibo, verificadorReferencia, verificadorAtestacion,
		autoridad.Validar,
	) != nil || ctx.Err() != nil || autoridad.Validar() != nil {
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
			a.Validar,
		) != nil || ctx.Err() != nil || a.Validar() != nil {
		return ProyeccionAutoridadObjetoEsperadoDocumentalV1{},
			errorAutoridadObjetoEsperadoDocumentalV1()
	}
	canonico, atestacion, err := materialVerificacionAutoridadObjetoEsperadoDocumentalV1(
		a.datos.recibo,
	)
	if err != nil || ctx.Err() != nil || a.Validar() != nil {
		return ProyeccionAutoridadObjetoEsperadoDocumentalV1{},
			errorAutoridadObjetoEsperadoDocumentalV1()
	}
	return ProyeccionAutoridadObjetoEsperadoDocumentalV1{
		Esquema: a.datos.esquema, Version: a.datos.version,
		ReciboMaterialRef:          a.datos.reciboRef,
		HuellaReciboMaterialSHA256: hex.EncodeToString(a.datos.huellaRecibo[:]),
		HuellaDeclaracionV4SHA256:  hex.EncodeToString(a.datos.huellaDeclaracionV4[:]),
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
	if d == nil {
		return errorAutoridadObjetoEsperadoDocumentalV1()
	}
	huellaDeclaracion, errDeclaracion :=
		huellaDeclaracionAutoridadObjetoEsperadoDocumentalV1(d.declaracion)
	contextoDeclaracion := ProyeccionContextoOperacionAlmacen{}
	if d.declaracion.datos != nil {
		contextoDeclaracion = d.declaracion.datos.solicitud.Contexto
	}
	if d.esquema != EsquemaAutoridadObjetoEsperadoDocumentalV1 ||
		d.version != VersionAutoridadObjetoEsperadoDocumentalV1 || d.recibo.Validar() != nil ||
		d.declaracion.Validar() != nil || errDeclaracion != nil ||
		!huellasMaterialV2Iguales(huellaDeclaracion, d.huellaDeclaracionV4) ||
		!reciboMaterialV2CotejaDeclaracionV4(d.recibo, d.declaracion) ||
		d.contextoV4 != contextoDeclaracion ||
		!referenciaOpacaAlmacenValida(d.reciboRef, 512) || d.reciboRef != d.recibo.referenciaDurableOriginal ||
		d.objeto.Validar() != nil || d.objeto.Referencia != d.recibo.instantanea.objetoRef ||
		d.objeto.Version != d.recibo.instantanea.objetoVersion ||
		!referenciaOpacaAlmacenValida(d.conectorID, 128) ||
		d.conectorID != d.recibo.instantanea.conectorLogicoID ||
		!referenciaOpacaAlmacenValida(d.efectoRef, 512) || d.efectoRef != d.recibo.efectoRef ||
		d.efectoRef != contextoDeclaracion.EfectoRef ||
		!esSHA256Hexadecimal(d.huellaPlanEfectoSHA256) ||
		d.huellaPlanEfectoSHA256 != contextoDeclaracion.HuellaPlanEfectoSHA256 ||
		!esSHA256Hexadecimal(d.huellaManifiestoSHA256) ||
		d.huellaManifiestoSHA256 != contextoDeclaracion.HuellaManifiestoSHA256 ||
		!esSHA256Hexadecimal(d.huellaPasoSHA256) ||
		d.huellaPasoSHA256 != contextoDeclaracion.HuellaPasoSHA256 ||
		d.pasoRef == "" || d.pasoRef != contextoDeclaracion.PasoRef ||
		contieneComodinContextoAlmacen(d.efectoRef, string(d.pasoRef)) {
		return errorAutoridadObjetoEsperadoDocumentalV1()
	}
	huella, err := d.recibo.HuellaSHA256()
	if err != nil || !huellasMaterialV2Iguales(huella, d.huellaRecibo) {
		return errorAutoridadObjetoEsperadoDocumentalV1()
	}
	canonico, atestacion, err := materialVerificacionAutoridadObjetoEsperadoDocumentalV1(d.recibo)
	sello, errSello := selloAtestacionAutoridadObjetoEsperadoDocumentalV1(canonico, atestacion)
	if err != nil || errSello != nil ||
		!atestacionesAutoridadObjetoEsperadoDocumentalV1Iguales(atestacion, d.atestacionRecibo) ||
		!huellasMaterialV2Iguales(sello, d.selloAtestacionRecibo) {
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
	revalidadores ...func() error,
) error {
	if ctx == nil || ctx.Err() != nil || recibo.Validar() != nil ||
		dependenciaMaterialV2Nula(verificadorReferencia) ||
		dependenciaMaterialV2Nula(verificadorAtestacion) ||
		revalidacionAutoridadObjetoEsperadoDocumentalV1Falla(revalidadores) ||
		recibo.VerificarAtestacion(ctx, verificadorAtestacion) != nil || ctx.Err() != nil ||
		revalidacionAutoridadObjetoEsperadoDocumentalV1Falla(revalidadores) {
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
	if err != nil || ctx.Err() != nil ||
		revalidacionAutoridadObjetoEsperadoDocumentalV1Falla(revalidadores) ||
		verificadorReferencia.VerificarReferenciaReciboMaterialV2(
			ctx, solicitud, resultado,
		) != nil || ctx.Err() != nil ||
		revalidacionAutoridadObjetoEsperadoDocumentalV1Falla(revalidadores) {
		return errorAutoridadObjetoEsperadoDocumentalV1()
	}
	return nil
}

func revalidacionAutoridadObjetoEsperadoDocumentalV1Falla(
	revalidadores []func() error,
) bool {
	for _, revalidar := range revalidadores {
		if revalidar == nil || revalidar() != nil {
			return true
		}
	}
	return false
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

func clonarDeclaracionAutoridadObjetoEsperadoDocumentalV1(
	declaracion DeclaracionEscrituraAlmacenDocumental,
) DeclaracionEscrituraAlmacenDocumental {
	datos := clonarDatosPreparacionEscrituraAlmacenDocumentalV4(declaracion.datos)
	if datos != nil {
		datos.contexto = clonarContextoAutoridadObjetoEsperadoDocumentalV1(datos.contexto)
		datos.vinculoEjecucion = clonarVinculoEjecucionAutoridadObjetoEsperadoDocumentalV1(
			declaracion.datos.vinculoEjecucion,
		)
	}
	return DeclaracionEscrituraAlmacenDocumental{datos: datos}
}

func clonarVinculoEjecucionAutoridadObjetoEsperadoDocumentalV1(
	vinculo VinculoEjecucionEscrituraAlmacenDocumental,
) VinculoEjecucionEscrituraAlmacenDocumental {
	if vinculo.datos == nil {
		return VinculoEjecucionEscrituraAlmacenDocumental{}
	}
	copia := *vinculo.datos
	copia.vinculoActivacion = clonarVinculoEstableActivacionDocumentalV3(
		vinculo.datos.vinculoActivacion,
	)
	copia.ordenDespachoConsumida = clonarOrdenDespachoDocumentalV3ConsumidaNominal(
		vinculo.datos.ordenDespachoConsumida,
	)
	return VinculoEjecucionEscrituraAlmacenDocumental{datos: &copia}
}

func clonarContextoAutoridadObjetoEsperadoDocumentalV1(
	contexto ContextoOperacionAlmacen,
) ContextoOperacionAlmacen {
	if contexto.datos == nil {
		return ContextoOperacionAlmacen{}
	}
	datosEvidencia, err := contexto.datos.evidencia.Datos()
	if err != nil {
		return ContextoOperacionAlmacen{}
	}
	evidencia, err := NuevaEvidenciaUsoDecisionAutorizacion(
		datosEvidencia.Decision, datosEvidencia.VerificadaEn,
	)
	if err != nil {
		return ContextoOperacionAlmacen{}
	}
	copia := *contexto.datos
	copia.evidencia = evidencia
	copia.pasos = clonarPasosOperacionAlmacen(contexto.datos.pasos)
	return ContextoOperacionAlmacen{datos: &copia}
}

func huellaDeclaracionAutoridadObjetoEsperadoDocumentalV1(
	declaracion DeclaracionEscrituraAlmacenDocumental,
) ([sha256.Size]byte, error) {
	canonico := serializarDeclaracionEscrituraAlmacenDocumental(declaracion)
	if len(canonico) == 0 {
		return [sha256.Size]byte{}, errorAutoridadObjetoEsperadoDocumentalV1()
	}
	return sha256.Sum256(canonico), nil
}

func selloAtestacionAutoridadObjetoEsperadoDocumentalV1(
	canonico []byte,
	atestacion AtestacionAutoridadObjetoEsperadoDocumentalV1,
) ([sha256.Size]byte, error) {
	if len(canonico) == 0 || len(atestacion.Codigo) == 0 {
		return [sha256.Size]byte{}, errorAutoridadObjetoEsperadoDocumentalV1()
	}
	sellador := sha256.New()
	escribir := func(campo []byte) {
		var longitud [8]byte
		binary.BigEndian.PutUint64(longitud[:], uint64(len(campo)))
		_, _ = sellador.Write(longitud[:])
		_, _ = sellador.Write(campo)
	}
	escribir([]byte("vec.documentos.sello-atestacion-autoridad-objeto-esperado.v1"))
	escribir(canonico)
	escribir([]byte(atestacion.Algoritmo))
	escribir([]byte(atestacion.ClaveRef))
	var version [4]byte
	binary.BigEndian.PutUint32(version[:], atestacion.ClaveVersion)
	escribir(version[:])
	escribir([]byte(atestacion.Dominio))
	escribir(atestacion.Codigo)
	var sello [sha256.Size]byte
	copy(sello[:], sellador.Sum(nil))
	return sello, nil
}

func atestacionesAutoridadObjetoEsperadoDocumentalV1Iguales(
	a, b AtestacionAutoridadObjetoEsperadoDocumentalV1,
) bool {
	return a.Algoritmo == b.Algoritmo && a.ClaveRef == b.ClaveRef &&
		a.ClaveVersion == b.ClaveVersion && a.Dominio == b.Dominio &&
		bytes.Equal(a.Codigo, b.Codigo)
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
