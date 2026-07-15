package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"reflect"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrEntradaNeutralDocumentalInvalida         = errors.New("vec: entrada neutral documental invalida")
	ErrSumideroSalidaDocumentalInvalido         = errors.New("vec: sumidero de salida documental invalido")
	ErrLimiteSalidaDocumentalExcedido           = errors.New("vec: limite de salida documental excedido")
	ErrBloqueSalidaDocumentalExcedido           = errors.New("vec: bloque de salida documental excedido")
	ErrSumideroSalidaDocumentalCerrado          = errors.New("vec: sumidero de salida documental cerrado")
	ErrSalidaDocumentalVacia                    = errors.New("vec: salida documental vacia")
	ErrEscrituraSalidaDocumentalIncompleta      = errors.New("vec: escritura de salida documental incompleta")
	ErrPruebaEscrituraAlmacenInvalida           = errors.New("vec: prueba de escritura en almacen invalida")
	ErrSerializacionMaterialDocumentalProhibida = errors.New("vec: serializacion de material documental protegido prohibida")
)

const (
	EsquemaCanonizacionEntradaNeutralDocumentalV1 = "vec.documentos.entrada-neutral.contenido-longitud-prefijada.v1"
	EsquemaPruebaEscrituraAlmacenDocumentalV1     = "vec.documentos.prueba-escritura-almacen.v1"
	EsquemaPruebaEscrituraAlmacenDocumentalV2     = "vec.documentos.prueba-escritura-almacen.v2"

	maximoBytesEntradaNeutralDocumental = 16 * 1024 * 1024
	maximosParrafosEntradaNeutral       = 100_000
	maximoBytesSalidaDocumental         = uint64(256 * 1024 * 1024)
	maximoBytesOrigenCargaDirectaV4     = 512
	maximoBytesOrigenesCargaDirectaV4   = 8 * 1024
	// Debe permanecer igual o por debajo del limite de las audiencias COSE
	// documentales ordinarias del conector de confianza homologado.
	maximoBytesMensajeEscrituraAlmacenV4 = 64 * 1024
	// Cada escritura necesita dos copias defensivas: una para la observacion
	// privada y otra para el destino. Acotar el bloque evita que una unica
	// llamada multiplique el techo total en memoria. El productor debe emitir
	// el documento en flujo mediante bloques no superiores a este limite.
	maximoBytesBloqueSalidaDocumental = 1 * 1024 * 1024
)

type datosPreparacionEntradaNeutralDocumental struct {
	canonicalizacionRef string
	contenido           domain.ContenidoDocumento
	contenidoCanonico   []byte
	tamano              uint64
}

// PreparacionEntradaNeutralDocumental fija el contenido y su codec antes de
// solicitar la HMAC al servicio interno. Es opaca, no autoritativa y no puede
// serializarse por mecanismos genericos porque puede contener datos personales.
type PreparacionEntradaNeutralDocumental struct {
	datos *datosPreparacionEntradaNeutralDocumental
}

func PrepararEntradaNeutralDocumental(
	contenido domain.ContenidoDocumento,
) (PreparacionEntradaNeutralDocumental, error) {
	canonico, err := canonizarContenidoEntradaNeutralDocumental(contenido)
	if err != nil {
		return PreparacionEntradaNeutralDocumental{}, ErrEntradaNeutralDocumentalInvalida
	}
	preparacion := PreparacionEntradaNeutralDocumental{datos: &datosPreparacionEntradaNeutralDocumental{
		canonicalizacionRef: EsquemaCanonizacionEntradaNeutralDocumentalV1,
		contenido:           clonarContenidoEntradaNeutralDocumental(contenido),
		contenidoCanonico:   append([]byte(nil), canonico...),
		tamano:              uint64(len(canonico)),
	}}
	if preparacion.Validar() != nil {
		return PreparacionEntradaNeutralDocumental{}, ErrEntradaNeutralDocumentalInvalida
	}
	return preparacion, nil
}

func (p PreparacionEntradaNeutralDocumental) Validar() error {
	if p.datos == nil ||
		validarDatosCanonicosEntradaNeutralDocumental(
			p.datos.canonicalizacionRef, p.datos.contenido,
			p.datos.contenidoCanonico, p.datos.tamano,
		) != nil {
		return ErrEntradaNeutralDocumentalInvalida
	}
	return nil
}

func (p PreparacionEntradaNeutralDocumental) Contenido() (domain.ContenidoDocumento, error) {
	if p.Validar() != nil {
		return domain.ContenidoDocumento{}, ErrEntradaNeutralDocumentalInvalida
	}
	return clonarContenidoEntradaNeutralDocumental(p.datos.contenido), nil
}

func (p PreparacionEntradaNeutralDocumental) ContenidoCanonico() ([]byte, error) {
	if p.Validar() != nil {
		return nil, ErrEntradaNeutralDocumentalInvalida
	}
	return append([]byte(nil), p.datos.contenidoCanonico...), nil
}

func (PreparacionEntradaNeutralDocumental) String() string {
	return "[PREPARACION-ENTRADA-NEUTRAL-DOCUMENTAL-OPACA-NO-VERIFICADA]"
}

func (p PreparacionEntradaNeutralDocumental) GoString() string { return p.String() }

func (p PreparacionEntradaNeutralDocumental) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}

func (PreparacionEntradaNeutralDocumental) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialDocumentalProhibida
}

func (*PreparacionEntradaNeutralDocumental) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialDocumentalProhibida
}

func (PreparacionEntradaNeutralDocumental) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialDocumentalProhibida
}

func (*PreparacionEntradaNeutralDocumental) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialDocumentalProhibida
}

func (PreparacionEntradaNeutralDocumental) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialDocumentalProhibida
}

func (*PreparacionEntradaNeutralDocumental) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialDocumentalProhibida
}

type datosEntradaNeutralDocumental struct {
	canonicalizacionRef string
	contenido           domain.ContenidoDocumento
	contenidoCanonico   []byte
	huellaHMACDeclarada string
	tamano              uint64
}

// EntradaNeutralDocumental es opaca e inmutable. Puede contener datos
// personales, por lo que no ofrece serializacion generica ni representaciones
// de depuracion que revelen el contenido.
type EntradaNeutralDocumental struct {
	datos *datosEntradaNeutralDocumental
}

// NuevaEntradaNeutralDocumental asocia una HMAC declarada a la preparacion ya
// fijada, sin verificarla criptograficamente. Solo el conector homologado puede
// convertir en el futuro preparacion+HMAC comprobada en una capacidad de uso.
func NuevaEntradaNeutralDocumental(
	preparacion PreparacionEntradaNeutralDocumental,
	huellaHMACDeclarada string,
) (EntradaNeutralDocumental, error) {
	if preparacion.Validar() != nil || !hmacSHA256PuertoValido(huellaHMACDeclarada) {
		return EntradaNeutralDocumental{}, ErrEntradaNeutralDocumentalInvalida
	}
	entrada := EntradaNeutralDocumental{datos: &datosEntradaNeutralDocumental{
		canonicalizacionRef: preparacion.datos.canonicalizacionRef,
		contenido:           clonarContenidoEntradaNeutralDocumental(preparacion.datos.contenido),
		contenidoCanonico:   append([]byte(nil), preparacion.datos.contenidoCanonico...),
		huellaHMACDeclarada: huellaHMACDeclarada,
		tamano:              preparacion.datos.tamano,
	}}
	if entrada.Validar() != nil {
		return EntradaNeutralDocumental{}, ErrEntradaNeutralDocumentalInvalida
	}
	return entrada, nil
}

func (e EntradaNeutralDocumental) Validar() error {
	if e.datos == nil || validarDatosCanonicosEntradaNeutralDocumental(
		e.datos.canonicalizacionRef, e.datos.contenido,
		e.datos.contenidoCanonico, e.datos.tamano,
	) != nil ||
		!hmacSHA256PuertoValido(e.datos.huellaHMACDeclarada) {
		return ErrEntradaNeutralDocumentalInvalida
	}
	return nil
}

func (e EntradaNeutralDocumental) Contenido() (domain.ContenidoDocumento, error) {
	if e.Validar() != nil {
		return domain.ContenidoDocumento{}, ErrEntradaNeutralDocumentalInvalida
	}
	return clonarContenidoEntradaNeutralDocumental(e.datos.contenido), nil
}

func (e EntradaNeutralDocumental) ContenidoCanonico() ([]byte, error) {
	if e.Validar() != nil {
		return nil, ErrEntradaNeutralDocumentalInvalida
	}
	return append([]byte(nil), e.datos.contenidoCanonico...), nil
}

// HuellaHMACDeclarada es una vinculacion nominal pendiente de comprobacion por
// de confianza. Su formato valido no demuestra que se haya
// calculado con una clave confiable ni habilita por si solo ningun efecto.
func (e EntradaNeutralDocumental) HuellaHMACDeclarada() (string, error) {
	if e.Validar() != nil {
		return "", ErrEntradaNeutralDocumentalInvalida
	}
	return e.datos.huellaHMACDeclarada, nil
}

func (e EntradaNeutralDocumental) Tamano() (uint64, error) {
	if e.Validar() != nil {
		return 0, ErrEntradaNeutralDocumentalInvalida
	}
	return e.datos.tamano, nil
}

func (e EntradaNeutralDocumental) CanonicalizacionRef() (string, error) {
	if e.Validar() != nil {
		return "", ErrEntradaNeutralDocumentalInvalida
	}
	return e.datos.canonicalizacionRef, nil
}

func (EntradaNeutralDocumental) String() string {
	return "[ENTRADA-NEUTRAL-DOCUMENTAL-OPACA]"
}

func (e EntradaNeutralDocumental) GoString() string { return e.String() }

func (e EntradaNeutralDocumental) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, e.String())
}

func (EntradaNeutralDocumental) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialDocumentalProhibida
}

func (*EntradaNeutralDocumental) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialDocumentalProhibida
}

func (EntradaNeutralDocumental) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialDocumentalProhibida
}

func (*EntradaNeutralDocumental) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialDocumentalProhibida
}

func (EntradaNeutralDocumental) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialDocumentalProhibida
}

func (*EntradaNeutralDocumental) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialDocumentalProhibida
}

// DatosSalidaObservadaDocumental es una proyeccion no autoritativa. La salida
// opaca que la origina solo puede fabricarla SumideroLimitadoSalidaDocumental.
type DatosSalidaObservadaDocumental struct {
	HuellaSHA256 string
	Tamano       uint64
	LimiteBytes  uint64
}

type SalidaObservadaDocumental struct {
	datos *DatosSalidaObservadaDocumental
}

func (s SalidaObservadaDocumental) Validar() error {
	if s.datos == nil || !esSHA256Hexadecimal(s.datos.HuellaSHA256) ||
		s.datos.Tamano == 0 || s.datos.LimiteBytes == 0 ||
		s.datos.LimiteBytes > maximoBytesSalidaDocumental ||
		s.datos.Tamano > s.datos.LimiteBytes {
		return ErrSumideroSalidaDocumentalInvalido
	}
	return nil
}

func (s SalidaObservadaDocumental) Datos() (DatosSalidaObservadaDocumental, error) {
	if s.Validar() != nil {
		return DatosSalidaObservadaDocumental{}, ErrSumideroSalidaDocumentalInvalido
	}
	return *s.datos, nil
}

// ValidarContraSolicitudEscribirObjeto enlaza la observacion local con el
// puerto de almacen ya existente. El sumidero no guarda objetos: aporta la
// pareja exacta Tamano/HuellaSHA256 con la que SolicitudEscribirObjeto ejecuta
// la escritura en flujo y aplica su contexto de autorizacion.
func (s SalidaObservadaDocumental) ValidarContraSolicitudEscribirObjeto(
	solicitud SolicitudEscribirObjeto,
) error {
	datos, err := s.Datos()
	if err != nil || solicitud.Validar() != nil || solicitud.Tamano < 1 ||
		uint64(solicitud.Tamano) != datos.Tamano || solicitud.HuellaSHA256 != datos.HuellaSHA256 {
		return ErrSumideroSalidaDocumentalInvalido
	}
	return nil
}

// SumideroLimitadoSalidaDocumental observa exactamente los bytes aceptados
// por el destino. Cada Write es atomico respecto de otros Write y de Cerrar;
// al superar el limite o fallar el destino queda cerrado sin posibilidad de
// reanudar una salida parcial.
type SumideroLimitadoSalidaDocumental struct {
	mu      sync.Mutex
	destino io.Writer
	limite  uint64
	tamano  uint64
	huella  hash.Hash
	cerrado bool
	fallo   error
	salida  SalidaObservadaDocumental
}

func NuevoSumideroLimitadoSalidaDocumental(
	destino io.Writer,
	limiteBytes uint64,
) (*SumideroLimitadoSalidaDocumental, error) {
	if interfazMaterialDocumentalNula(destino) || limiteBytes == 0 ||
		limiteBytes > maximoBytesSalidaDocumental {
		return nil, ErrSumideroSalidaDocumentalInvalido
	}
	return &SumideroLimitadoSalidaDocumental{
		destino: destino, limite: limiteBytes, huella: sha256.New(),
	}, nil
}

func (s *SumideroLimitadoSalidaDocumental) Write(p []byte) (int, error) {
	if s == nil {
		return 0, ErrSumideroSalidaDocumentalInvalido
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cerrado {
		return 0, ErrSumideroSalidaDocumentalCerrado
	}
	if len(p) > maximoBytesBloqueSalidaDocumental {
		s.cerrarConFallo(ErrBloqueSalidaDocumentalExcedido)
		return 0, ErrBloqueSalidaDocumentalExcedido
	}
	if uint64(len(p)) > s.limite-s.tamano {
		s.cerrarConFallo(ErrLimiteSalidaDocumentalExcedido)
		return 0, ErrLimiteSalidaDocumentalExcedido
	}
	if len(p) == 0 {
		return 0, nil
	}
	// La observacion y la entrega no comparten memoria. Un destino hostil puede
	// retener o mutar su slice sin cambiar los bytes privados que se hashean.
	observada := append([]byte(nil), p...)
	entrega := append([]byte(nil), observada...)
	n, err := s.destino.Write(entrega)
	if n < 0 || n > len(entrega) {
		n = 0
		err = ErrEscrituraSalidaDocumentalIncompleta
	}
	if err != nil {
		s.cerrarConFallo(err)
		return n, err
	}
	if n != len(entrega) {
		s.cerrarConFallo(ErrEscrituraSalidaDocumentalIncompleta)
		return n, ErrEscrituraSalidaDocumentalIncompleta
	}
	_, _ = s.huella.Write(observada)
	s.tamano += uint64(len(observada))
	return n, nil
}

// Cerrar es irreversible e idempotente cuando la primera clausura tuvo exito.
// Tras un fallo devuelve siempre ese fallo y nunca fabrica una salida parcial.
func (s *SumideroLimitadoSalidaDocumental) Cerrar() (SalidaObservadaDocumental, error) {
	if s == nil {
		return SalidaObservadaDocumental{}, ErrSumideroSalidaDocumentalInvalido
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fallo != nil {
		return SalidaObservadaDocumental{}, s.fallo
	}
	if s.cerrado {
		if s.salida.Validar() != nil {
			return SalidaObservadaDocumental{}, ErrSumideroSalidaDocumentalCerrado
		}
		return s.salida, nil
	}
	if s.tamano == 0 {
		s.cerrarConFallo(ErrSalidaDocumentalVacia)
		return SalidaObservadaDocumental{}, ErrSalidaDocumentalVacia
	}
	datos := &DatosSalidaObservadaDocumental{
		HuellaSHA256: hex.EncodeToString(s.huella.Sum(nil)),
		Tamano:       s.tamano,
		LimiteBytes:  s.limite,
	}
	s.salida = SalidaObservadaDocumental{datos: datos}
	s.cerrado = true
	s.destino = nil
	if s.salida.Validar() != nil {
		s.fallo = ErrSumideroSalidaDocumentalInvalido
		return SalidaObservadaDocumental{}, s.fallo
	}
	return s.salida, nil
}

func (s *SumideroLimitadoSalidaDocumental) cerrarConFallo(err error) {
	if s.fallo == nil {
		s.fallo = err
	}
	s.cerrado = true
	s.destino = nil
}

// VinculoPoliticaInmutabilidadDocumental identifica la version exacta de una
// politica gobernada y el perfil de capacidades que esta exige. El nombre se
// conserva por compatibilidad conceptual, pero Versionado no se interpreta
// como WORM ni como bloqueo legal: Retencion y BloqueoLegal son requisitos
// independientes y solo producen esos estados cuando la politica los exige.
//
// Este valor sigue siendo declarativo. Su autoridad y la atestacion de las
// capacidades de arranque pertenecen al conector de confianza instalado.
type VinculoPoliticaInmutabilidadDocumental struct {
	PoliticaRef                string
	Version                    uint64
	HuellaSHA256               string
	Requisitos                 RequisitosAlmacenObjetos
	HuellaRequisitosSHA256     string
	HuellaCapacidadesSHA256    string
	RetencionHasta             time.Time
	ExigeInmovilizacionInicial bool
}

func NuevoVinculoPoliticaInmutabilidadDocumental(
	politicaRef string,
	version uint64,
	huellaSHA256 string,
	requisitos RequisitosAlmacenObjetos,
	capacidades CapacidadesAlmacenObjetos,
	retencionHasta time.Time,
	exigeInmovilizacionInicial bool,
) (VinculoPoliticaInmutabilidadDocumental, error) {
	vinculo := VinculoPoliticaInmutabilidadDocumental{
		PoliticaRef: politicaRef, Version: version, HuellaSHA256: huellaSHA256,
		Requisitos:              requisitos,
		HuellaRequisitosSHA256:  huellaRequisitosAlmacenDocumentalV4(requisitos),
		HuellaCapacidadesSHA256: huellaCapacidadesAlmacenDocumentalV4(capacidades),
		RetencionHasta:          retencionHasta, ExigeInmovilizacionInicial: exigeInmovilizacionInicial,
	}
	if vinculo.Validar() != nil ||
		VerificarCapacidadesAlmacen(capacidades, requisitos) != nil {
		return VinculoPoliticaInmutabilidadDocumental{}, ErrPruebaEscrituraAlmacenInvalida
	}
	return vinculo, nil
}

func (v VinculoPoliticaInmutabilidadDocumental) Validar() error {
	if !referenciaMaterialDocumentalV4Valida(v.PoliticaRef) || v.Version == 0 ||
		!esSHA256Hexadecimal(v.HuellaSHA256) ||
		!requisitosBaseEscrituraDocumentalV4(v.Requisitos) ||
		v.HuellaRequisitosSHA256 != huellaRequisitosAlmacenDocumentalV4(v.Requisitos) ||
		!esSHA256Hexadecimal(v.HuellaCapacidadesSHA256) ||
		(!v.RetencionHasta.IsZero() &&
			(!v.Requisitos.Retencion || !instanteEjecucionDocumentalV3Valido(v.RetencionHasta))) ||
		(v.ExigeInmovilizacionInicial && !v.Requisitos.BloqueoLegal) ||
		!huellasMaterialDocumentalV4Distintas(
			v.HuellaSHA256, v.HuellaRequisitosSHA256, v.HuellaCapacidadesSHA256,
		) {
		return ErrPruebaEscrituraAlmacenInvalida
	}
	return nil
}

func (v VinculoPoliticaInmutabilidadDocumental) validarContra(
	solicitud SolicitudEscribirObjeto,
	resultado ResultadoOperacionObjeto,
	capacidades CapacidadesAlmacenObjetos,
) error {
	if v.Validar() != nil || VerificarCapacidadesAlmacen(capacidades, v.Requisitos) != nil ||
		v.HuellaCapacidadesSHA256 != huellaCapacidadesAlmacenDocumentalV4(capacidades) ||
		capacidades.ConectorID != resultado.Objeto.ConectorID ||
		capacidades.TamanoMaximoObjeto < solicitud.Tamano ||
		resultado.Objeto.Inmovilizado != v.ExigeInmovilizacionInicial {
		return ErrPruebaEscrituraAlmacenInvalida
	}
	if !v.RetencionHasta.IsZero() {
		if !resultado.Objeto.RetenidoHasta.Equal(v.RetencionHasta) ||
			!v.RetencionHasta.After(resultado.Objeto.AlmacenadoEn) {
			return ErrPruebaEscrituraAlmacenInvalida
		}
	} else if !resultado.Objeto.RetenidoHasta.IsZero() {
		return ErrPruebaEscrituraAlmacenInvalida
	}
	return nil
}

type datosVinculoEjecucionEscrituraAlmacenDocumental struct {
	ReservaRef                 string
	BorradorRef                string
	EfectoRef                  string
	HuellaPlanSHA256           string
	DecisionRef                string
	HuellaDecisionSHA256       string
	SecuenciaCercado           uint64
	HuellaVinculoCercadoSHA256 string
	VerificacionCercadoRef     string
	VerificacionCercadoEn      time.Time
}

// VinculoEjecucionEscrituraAlmacenDocumental proyecta, sin secretos, la
// reserva, el manifiesto, la decision consumida y el cercado V3 exactos. Es
// opaco para que un adaptador no pueda recomponer ni alterar la proyeccion.
// Sigue siendo dato nominal no autoritativo: al consumir un recibo debe
// cotejarse de nuevo contra los valores tipados originales.
type VinculoEjecucionEscrituraAlmacenDocumental struct {
	datos *datosVinculoEjecucionEscrituraAlmacenDocumental
}

func NuevoVinculoEjecucionEscrituraAlmacenDocumental(
	reservaRef string,
	manifiesto ManifiestoEjecucionDocumentalV3,
	consumo ConsumoDecisionEjecucionDocumentalV3,
	token TokenCercadoEjecucionDocumentalV3,
	verificacion ResultadoVerificacionTokenCercadoDocumentalV3,
) (VinculoEjecucionEscrituraAlmacenDocumental, error) {
	if !verificacionTokenCercadoDocumentalV3Valida(
		reservaRef, manifiesto, consumo, token, verificacion,
	) {
		return VinculoEjecucionEscrituraAlmacenDocumental{}, ErrPruebaEscrituraAlmacenInvalida
	}
	datos, err := manifiesto.Datos()
	if err != nil {
		return VinculoEjecucionEscrituraAlmacenDocumental{}, ErrPruebaEscrituraAlmacenInvalida
	}
	datosVinculo := &datosVinculoEjecucionEscrituraAlmacenDocumental{
		ReservaRef: reservaRef, BorradorRef: datos.BorradorRef, EfectoRef: datos.EfectoRef,
		HuellaPlanSHA256: datos.HuellaPlanSHA256,
		DecisionRef:      consumo.DecisionRef, HuellaDecisionSHA256: consumo.HuellaDecisionSHA256,
		SecuenciaCercado: token.Secuencia(), HuellaVinculoCercadoSHA256: token.HuellaVinculoSHA256(),
		VerificacionCercadoRef: verificacion.verificacionRef,
		VerificacionCercadoEn:  verificacion.verificadaEn,
	}
	vinculo := VinculoEjecucionEscrituraAlmacenDocumental{datos: datosVinculo}
	if vinculo.Validar() != nil {
		return VinculoEjecucionEscrituraAlmacenDocumental{}, ErrPruebaEscrituraAlmacenInvalida
	}
	return vinculo, nil
}

func (v VinculoEjecucionEscrituraAlmacenDocumental) Validar() error {
	if v.datos == nil {
		return ErrPruebaEscrituraAlmacenInvalida
	}
	datos := v.datos
	referencias := []string{
		datos.ReservaRef, datos.BorradorRef, datos.EfectoRef,
		datos.DecisionRef, datos.VerificacionCercadoRef,
	}
	for indice, referencia := range referencias {
		if !referenciaMaterialDocumentalV4Valida(referencia) {
			return ErrPruebaEscrituraAlmacenInvalida
		}
		for otra := indice + 1; otra < len(referencias); otra++ {
			if referencia == referencias[otra] {
				return ErrPruebaEscrituraAlmacenInvalida
			}
		}
	}
	if !huellasMaterialDocumentalV4Distintas(
		datos.HuellaPlanSHA256, datos.HuellaDecisionSHA256, datos.HuellaVinculoCercadoSHA256,
	) || datos.SecuenciaCercado == 0 ||
		!instanteEjecucionDocumentalV3Valido(datos.VerificacionCercadoEn) {
		return ErrPruebaEscrituraAlmacenInvalida
	}
	return nil
}

func (v VinculoEjecucionEscrituraAlmacenDocumental) ValidarContra(
	manifiesto ManifiestoEjecucionDocumentalV3,
	consumo ConsumoDecisionEjecucionDocumentalV3,
	token TokenCercadoEjecucionDocumentalV3,
	verificacion ResultadoVerificacionTokenCercadoDocumentalV3,
) error {
	if v.Validar() != nil {
		return ErrPruebaEscrituraAlmacenInvalida
	}
	esperado, err := NuevoVinculoEjecucionEscrituraAlmacenDocumental(
		v.datos.ReservaRef, manifiesto, consumo, token, verificacion,
	)
	if err != nil || esperado.Validar() != nil || *v.datos != *esperado.datos {
		return ErrPruebaEscrituraAlmacenInvalida
	}
	return nil
}

func (VinculoEjecucionEscrituraAlmacenDocumental) String() string {
	return "[VINCULO-EJECUCION-ESCRITURA-ALMACEN-V4-OPACO-NO-VERIFICADO]"
}

func (v VinculoEjecucionEscrituraAlmacenDocumental) GoString() string { return v.String() }

func (v VinculoEjecucionEscrituraAlmacenDocumental) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, v.String())
}

func (v VinculoEjecucionEscrituraAlmacenDocumental) LogValue() slog.Value {
	return slog.StringValue(v.String())
}

func (VinculoEjecucionEscrituraAlmacenDocumental) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialDocumentalProhibida
}

func (*VinculoEjecucionEscrituraAlmacenDocumental) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialDocumentalProhibida
}

func (VinculoEjecucionEscrituraAlmacenDocumental) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialDocumentalProhibida
}

func (*VinculoEjecucionEscrituraAlmacenDocumental) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialDocumentalProhibida
}

func (VinculoEjecucionEscrituraAlmacenDocumental) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialDocumentalProhibida
}

func (*VinculoEjecucionEscrituraAlmacenDocumental) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialDocumentalProhibida
}

type instantaneaSolicitudEscrituraAlmacenDocumentalV4 struct {
	Contexto          ProyeccionContextoOperacionAlmacen
	ClaveIdempotencia string
	Zona              ZonaAlmacen
	MIME              string
	Tamano            int64
	HuellaSHA256      string
}

type datosPreparacionEscrituraAlmacenDocumentalV4 struct {
	contexto                       ContextoOperacionAlmacen
	solicitud                      instantaneaSolicitudEscrituraAlmacenDocumentalV4
	huellaSolicitudSHA256          string
	resultado                      ResultadoOperacionObjeto
	capacidades                    CapacidadesAlmacenObjetos
	salida                         DatosSalidaObservadaDocumental
	vinculoEjecucion               VinculoEjecucionEscrituraAlmacenDocumental
	politica                       VinculoPoliticaInmutabilidadDocumental
	huellaEvidenciaOperacionSHA256 string
}

// PreparacionEscrituraAlmacenDocumentalV4 es una instantanea opaca y no
// autoritativa. Solo nace tras cotejar la solicitud semantica completa, el
// resultado del conector, sus capacidades exactas, los bytes observados, la
// ejecucion cercada y la politica. No es un recibo ni confirma el efecto.
type PreparacionEscrituraAlmacenDocumentalV4 struct {
	datos *datosPreparacionEscrituraAlmacenDocumentalV4
}

func PrepararEscrituraAlmacenDocumentalV4(
	solicitud SolicitudEscribirObjeto,
	resultado ResultadoOperacionObjeto,
	capacidades CapacidadesAlmacenObjetos,
	salida SalidaObservadaDocumental,
	vinculo VinculoEjecucionEscrituraAlmacenDocumental,
	politica VinculoPoliticaInmutabilidadDocumental,
) (PreparacionEscrituraAlmacenDocumentalV4, error) {
	// Preflight previo a cualquier copia o canonizacion: el contrato heredado
	// limita el numero de origenes, pero no sus bytes agregados.
	if !origenesCapacidadesAlmacenDocumentalV4Validos(capacidades) {
		return PreparacionEscrituraAlmacenDocumentalV4{}, ErrPruebaEscrituraAlmacenInvalida
	}
	proyeccion, err := solicitud.Contexto.Proyeccion()
	datosSalida, errSalida := salida.Datos()
	instantanea := instantaneaSolicitudEscrituraAlmacenDocumentalV4{
		Contexto: proyeccion, ClaveIdempotencia: solicitud.ClaveIdempotencia,
		Zona: solicitud.Zona, MIME: solicitud.MIME, Tamano: solicitud.Tamano,
		HuellaSHA256: solicitud.HuellaSHA256,
	}
	preparacion := PreparacionEscrituraAlmacenDocumentalV4{
		datos: &datosPreparacionEscrituraAlmacenDocumentalV4{
			contexto: solicitud.Contexto, solicitud: instantanea,
			huellaSolicitudSHA256: huellaSolicitudEscrituraAlmacenDocumentalV4(instantanea),
			resultado:             resultado, capacidades: clonarCapacidadesAlmacenDocumentalV4(capacidades),
			salida:                         datosSalida,
			vinculoEjecucion:               clonarVinculoEjecucionEscrituraAlmacenDocumentalV4(vinculo),
			politica:                       politica,
			huellaEvidenciaOperacionSHA256: huellaEvidenciaOperacionAlmacenDocumental(resultado.Evidencia),
		},
	}
	if err != nil || errSalida != nil || preparacion.Validar() != nil {
		return PreparacionEscrituraAlmacenDocumentalV4{}, ErrPruebaEscrituraAlmacenInvalida
	}
	return preparacion, nil
}

func (p PreparacionEscrituraAlmacenDocumentalV4) Validar() error {
	if p.datos == nil || validarDatosPreparacionEscrituraAlmacenDocumentalV4(p.datos) != nil {
		return ErrPruebaEscrituraAlmacenInvalida
	}
	return nil
}

func (PreparacionEscrituraAlmacenDocumentalV4) String() string {
	return "[PREPARACION-ESCRITURA-ALMACEN-V4-OPACA-NO-VERIFICADA]"
}

func (p PreparacionEscrituraAlmacenDocumentalV4) GoString() string { return p.String() }

func (p PreparacionEscrituraAlmacenDocumentalV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}

func (p PreparacionEscrituraAlmacenDocumentalV4) LogValue() slog.Value {
	return slog.StringValue(p.String())
}

func (PreparacionEscrituraAlmacenDocumentalV4) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialDocumentalProhibida
}

func (*PreparacionEscrituraAlmacenDocumentalV4) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialDocumentalProhibida
}

func (PreparacionEscrituraAlmacenDocumentalV4) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialDocumentalProhibida
}

func (*PreparacionEscrituraAlmacenDocumentalV4) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialDocumentalProhibida
}

func (PreparacionEscrituraAlmacenDocumentalV4) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialDocumentalProhibida
}

func (*PreparacionEscrituraAlmacenDocumentalV4) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialDocumentalProhibida
}

// DeclaracionEscrituraAlmacenDocumental es una copia opaca de la preparacion
// exacta que debe firmar el conector. Sus campos no son reconstruibles desde
// un DTO y su validez sintactica no concede autoridad.
type DeclaracionEscrituraAlmacenDocumental struct {
	datos *datosPreparacionEscrituraAlmacenDocumentalV4
}

func NuevaDeclaracionEscrituraAlmacenDocumental(
	preparacion PreparacionEscrituraAlmacenDocumentalV4,
) (DeclaracionEscrituraAlmacenDocumental, error) {
	if preparacion.Validar() != nil {
		return DeclaracionEscrituraAlmacenDocumental{}, ErrPruebaEscrituraAlmacenInvalida
	}
	declaracion := DeclaracionEscrituraAlmacenDocumental{
		datos: clonarDatosPreparacionEscrituraAlmacenDocumentalV4(preparacion.datos),
	}
	if declaracion.Validar() != nil {
		return DeclaracionEscrituraAlmacenDocumental{}, ErrPruebaEscrituraAlmacenInvalida
	}
	return declaracion, nil
}

func (d DeclaracionEscrituraAlmacenDocumental) Validar() error {
	if d.datos == nil || validarDatosPreparacionEscrituraAlmacenDocumentalV4(d.datos) != nil {
		return ErrPruebaEscrituraAlmacenInvalida
	}
	return nil
}

func (d DeclaracionEscrituraAlmacenDocumental) Objeto() (ObjetoAlmacenado, error) {
	if d.Validar() != nil {
		return ObjetoAlmacenado{}, ErrPruebaEscrituraAlmacenInvalida
	}
	return d.datos.resultado.Objeto, nil
}

func (d DeclaracionEscrituraAlmacenDocumental) EvidenciaOperacion() (
	EvidenciaOperacionAlmacen,
	error,
) {
	if d.Validar() != nil {
		return EvidenciaOperacionAlmacen{}, ErrPruebaEscrituraAlmacenInvalida
	}
	return d.datos.resultado.Evidencia, nil
}

func (d DeclaracionEscrituraAlmacenDocumental) Capacidades() (
	CapacidadesAlmacenObjetos,
	error,
) {
	if d.Validar() != nil {
		return CapacidadesAlmacenObjetos{}, ErrPruebaEscrituraAlmacenInvalida
	}
	return clonarCapacidadesAlmacenDocumentalV4(d.datos.capacidades), nil
}

func (d DeclaracionEscrituraAlmacenDocumental) Politica() (
	VinculoPoliticaInmutabilidadDocumental,
	error,
) {
	if d.Validar() != nil {
		return VinculoPoliticaInmutabilidadDocumental{}, ErrPruebaEscrituraAlmacenInvalida
	}
	return d.datos.politica, nil
}

func (d DeclaracionEscrituraAlmacenDocumental) VinculoEjecucion() (
	VinculoEjecucionEscrituraAlmacenDocumental,
	error,
) {
	if d.Validar() != nil {
		return VinculoEjecucionEscrituraAlmacenDocumental{}, ErrPruebaEscrituraAlmacenInvalida
	}
	return clonarVinculoEjecucionEscrituraAlmacenDocumentalV4(d.datos.vinculoEjecucion), nil
}

func (d DeclaracionEscrituraAlmacenDocumental) ValidarContraSalida(
	salida SalidaObservadaDocumental,
) error {
	datos, err := salida.Datos()
	if d.Validar() != nil || err != nil || d.datos.salida != datos {
		return ErrPruebaEscrituraAlmacenInvalida
	}
	return nil
}

func (d DeclaracionEscrituraAlmacenDocumental) ValidarContraEjecucion(
	manifiesto ManifiestoEjecucionDocumentalV3,
	consumo ConsumoDecisionEjecucionDocumentalV3,
	token TokenCercadoEjecucionDocumentalV3,
	verificacion ResultadoVerificacionTokenCercadoDocumentalV3,
	salida SalidaObservadaDocumental,
) error {
	if d.ValidarContraSalida(salida) != nil ||
		d.datos.vinculoEjecucion.ValidarContra(manifiesto, consumo, token, verificacion) != nil {
		return ErrPruebaEscrituraAlmacenInvalida
	}
	return nil
}

func (DeclaracionEscrituraAlmacenDocumental) String() string {
	return "[DECLARACION-ESCRITURA-ALMACEN-V4-OPACA-NO-VERIFICADA]"
}

func (d DeclaracionEscrituraAlmacenDocumental) GoString() string { return d.String() }

func (d DeclaracionEscrituraAlmacenDocumental) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}

func (d DeclaracionEscrituraAlmacenDocumental) LogValue() slog.Value {
	return slog.StringValue(d.String())
}

func (DeclaracionEscrituraAlmacenDocumental) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialDocumentalProhibida
}

func (*DeclaracionEscrituraAlmacenDocumental) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialDocumentalProhibida
}

func (DeclaracionEscrituraAlmacenDocumental) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialDocumentalProhibida
}

func (*DeclaracionEscrituraAlmacenDocumental) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialDocumentalProhibida
}

func (DeclaracionEscrituraAlmacenDocumental) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialDocumentalProhibida
}

func (*DeclaracionEscrituraAlmacenDocumental) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialDocumentalProhibida
}

// PruebaCrudaEscrituraAlmacen es solo un sobre opaco pendiente de
// verificacion. Construirlo, persistirlo o calcular su SHA-256 no concede
// autoridad y nunca permite confirmar una escritura.
type PruebaCrudaEscrituraAlmacen struct {
	pruebaRef           string
	declaracion         DeclaracionEscrituraAlmacenDocumental
	mensaje             []byte
	huellaMensajeSHA256 string
	sobre               SobreCriptograficoDocumentalCrudoV4
	huellaSobreSHA256   string
}

func NuevaPruebaCrudaEscrituraAlmacen(
	pruebaRef string,
	declaracion DeclaracionEscrituraAlmacenDocumental,
	sobre SobreCriptograficoDocumentalCrudoV4,
) (PruebaCrudaEscrituraAlmacen, error) {
	huellaSobre, err := sobre.HuellaSHA256()
	if !referenciaMaterialDocumentalV4Valida(pruebaRef) || declaracion.Validar() != nil || err != nil {
		return PruebaCrudaEscrituraAlmacen{}, ErrPruebaEscrituraAlmacenInvalida
	}
	mensaje := serializarDeclaracionEscrituraAlmacenDocumental(declaracion)
	if len(mensaje) == 0 || len(mensaje) > maximoBytesMensajeEscrituraAlmacenV4 {
		return PruebaCrudaEscrituraAlmacen{}, ErrPruebaEscrituraAlmacenInvalida
	}
	prueba := PruebaCrudaEscrituraAlmacen{
		pruebaRef: pruebaRef,
		declaracion: DeclaracionEscrituraAlmacenDocumental{
			datos: clonarDatosPreparacionEscrituraAlmacenDocumentalV4(declaracion.datos),
		},
		mensaje:             append([]byte(nil), mensaje...),
		huellaMensajeSHA256: huellaSHA256MaterialDocumental(mensaje),
		sobre:               sobre,
		huellaSobreSHA256:   huellaSobre,
	}
	if prueba.ValidarSintaxis() != nil {
		return PruebaCrudaEscrituraAlmacen{}, ErrPruebaEscrituraAlmacenInvalida
	}
	return prueba, nil
}

// ValidarSintaxis solo coteja el mensaje nominal y el contenedor crudo. No
// interpreta COSE ni concede autoridad para confirmar el efecto remoto.
func (p PruebaCrudaEscrituraAlmacen) ValidarSintaxis() error {
	mensaje := serializarDeclaracionEscrituraAlmacenDocumental(p.declaracion)
	huellaSobre, err := p.sobre.HuellaSHA256()
	if !referenciaMaterialDocumentalV4Valida(p.pruebaRef) || p.declaracion.Validar() != nil ||
		len(p.mensaje) == 0 || len(p.mensaje) > maximoBytesMensajeEscrituraAlmacenV4 ||
		!bytes.Equal(p.mensaje, mensaje) ||
		p.huellaMensajeSHA256 != huellaSHA256MaterialDocumental(p.mensaje) ||
		err != nil || p.huellaSobreSHA256 != huellaSobre {
		return ErrPruebaEscrituraAlmacenInvalida
	}
	return nil
}

func (p PruebaCrudaEscrituraAlmacen) Mensaje() ([]byte, error) {
	if p.ValidarSintaxis() != nil {
		return nil, ErrPruebaEscrituraAlmacenInvalida
	}
	return append([]byte(nil), p.mensaje...), nil
}

func (p PruebaCrudaEscrituraAlmacen) SobreCrudo() (
	SobreCriptograficoDocumentalCrudoV4,
	error,
) {
	if p.ValidarSintaxis() != nil {
		return SobreCriptograficoDocumentalCrudoV4{}, ErrPruebaEscrituraAlmacenInvalida
	}
	return p.sobre, nil
}

func (p PruebaCrudaEscrituraAlmacen) Declaracion() (
	DeclaracionEscrituraAlmacenDocumental,
	error,
) {
	if p.ValidarSintaxis() != nil {
		return DeclaracionEscrituraAlmacenDocumental{}, ErrPruebaEscrituraAlmacenInvalida
	}
	return DeclaracionEscrituraAlmacenDocumental{
		datos: clonarDatosPreparacionEscrituraAlmacenDocumentalV4(p.declaracion.datos),
	}, nil
}

func (p PruebaCrudaEscrituraAlmacen) HuellaMensajeSHA256() (string, error) {
	if p.ValidarSintaxis() != nil {
		return "", ErrPruebaEscrituraAlmacenInvalida
	}
	return p.huellaMensajeSHA256, nil
}

func (p PruebaCrudaEscrituraAlmacen) PruebaRef() (string, error) {
	if p.ValidarSintaxis() != nil {
		return "", ErrPruebaEscrituraAlmacenInvalida
	}
	return p.pruebaRef, nil
}

func (PruebaCrudaEscrituraAlmacen) String() string {
	return "[PRUEBA-CRUDA-ESCRITURA-ALMACEN-OPACA-NO-VERIFICADA]"
}

func (p PruebaCrudaEscrituraAlmacen) GoString() string { return p.String() }

func (p PruebaCrudaEscrituraAlmacen) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}

func (p PruebaCrudaEscrituraAlmacen) LogValue() slog.Value {
	return slog.StringValue(p.String())
}

func (PruebaCrudaEscrituraAlmacen) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMaterialDocumentalProhibida
}

func (*PruebaCrudaEscrituraAlmacen) UnmarshalJSON([]byte) error {
	return ErrSerializacionMaterialDocumentalProhibida
}

func (PruebaCrudaEscrituraAlmacen) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMaterialDocumentalProhibida
}

func (*PruebaCrudaEscrituraAlmacen) UnmarshalText([]byte) error {
	return ErrSerializacionMaterialDocumentalProhibida
}

func (PruebaCrudaEscrituraAlmacen) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMaterialDocumentalProhibida
}

func (*PruebaCrudaEscrituraAlmacen) UnmarshalBinary([]byte) error {
	return ErrSerializacionMaterialDocumentalProhibida
}

// Deliberadamente no existe en ports una fabrica de
// ReciboEscrituraAlmacenVerificado. La conversion de esta prueba nominal en
// autoridad pertenece al conector de confianza homologado, donde el
// llamador no puede inyectar un verificador alternativo. Hasta integrar esa
// frontera, ninguna confirmacion debe aceptar esta prueba como recibo.

func validarDatosPreparacionEscrituraAlmacenDocumentalV4(
	d *datosPreparacionEscrituraAlmacenDocumentalV4,
) error {
	if d == nil || d.vinculoEjecucion.Validar() != nil || d.politica.Validar() != nil ||
		d.resultado.Validar() != nil || d.huellaSolicitudSHA256 == "" ||
		d.huellaSolicitudSHA256 != huellaSolicitudEscrituraAlmacenDocumentalV4(d.solicitud) ||
		d.huellaEvidenciaOperacionSHA256 == "" ||
		d.huellaEvidenciaOperacionSHA256 !=
			huellaEvidenciaOperacionAlmacenDocumental(d.resultado.Evidencia) {
		return ErrPruebaEscrituraAlmacenInvalida
	}
	proyeccion, err := d.contexto.Proyeccion()
	if err != nil || proyeccion != d.solicitud.Contexto {
		return ErrPruebaEscrituraAlmacenInvalida
	}
	solicitud := SolicitudEscribirObjeto{
		Contexto: d.contexto, ClaveIdempotencia: d.solicitud.ClaveIdempotencia,
		Zona: d.solicitud.Zona, MIME: d.solicitud.MIME, Tamano: d.solicitud.Tamano,
		HuellaSHA256: d.solicitud.HuellaSHA256, Contenido: bytes.NewReader(nil),
	}
	datosSalida := d.salida
	salida := SalidaObservadaDocumental{datos: &datosSalida}
	instanteResultado := d.resultado.Evidencia.RealizadaEn
	vinculo := d.vinculoEjecucion.datos
	if vinculo == nil {
		return ErrPruebaEscrituraAlmacenInvalida
	}
	instanteCercado := vinculo.VerificacionCercadoEn
	if d.solicitud.Zona != ZonaAlmacenCuarentena || solicitud.Validar() != nil ||
		d.resultado.ValidarEscritura(solicitud, d.capacidades) != nil ||
		salida.ValidarContraSolicitudEscribirObjeto(solicitud) != nil ||
		d.politica.validarContra(solicitud, d.resultado, d.capacidades) != nil ||
		d.contexto.ValidarParaEn(AccionAlmacenEscribir, instanteCercado) != nil ||
		d.contexto.ValidarParaEn(AccionAlmacenEscribir, instanteResultado) != nil ||
		proyeccion.EfectoRef != vinculo.EfectoRef ||
		proyeccion.AutorizacionRef != vinculo.DecisionRef ||
		proyeccion.HuellaDecisionSHA256 != vinculo.HuellaDecisionSHA256 ||
		proyeccion.HuellaManifiestoSHA256 != vinculo.HuellaPlanSHA256 ||
		proyeccion.VerificadaEn.After(instanteCercado) || instanteResultado.Before(instanteCercado) ||
		!instanteEjecucionDocumentalV3Valido(d.resultado.Objeto.AlmacenadoEn) ||
		!instanteEjecucionDocumentalV3Valido(instanteResultado) ||
		(!d.resultado.Objeto.RetenidoHasta.IsZero() &&
			!instanteEjecucionDocumentalV3Valido(d.resultado.Objeto.RetenidoHasta)) ||
		(!d.resultado.Evidencia.ReintentoIdempotente &&
			d.resultado.Objeto.AlmacenadoEn.Before(instanteCercado)) ||
		d.resultado.Objeto.Zona != ZonaAlmacenCuarentena || d.resultado.Objeto.Eliminado ||
		!huellasMaterialDocumentalV4Distintas(
			d.resultado.Objeto.HuellaSHA256, vinculo.HuellaPlanSHA256,
			vinculo.HuellaDecisionSHA256, vinculo.HuellaVinculoCercadoSHA256,
			d.huellaEvidenciaOperacionSHA256, d.huellaSolicitudSHA256,
			d.politica.HuellaSHA256, d.politica.HuellaRequisitosSHA256,
			d.politica.HuellaCapacidadesSHA256,
		) {
		return ErrPruebaEscrituraAlmacenInvalida
	}
	return nil
}

func requisitosBaseEscrituraDocumentalV4(requisitos RequisitosAlmacenObjetos) bool {
	return requisitos.EscrituraEnFlujo && requisitos.LecturaEnFlujo &&
		requisitos.ReferenciasOpacas && requisitos.IntegridadSHA256 && requisitos.Versionado &&
		requisitos.CifradoEnTransito && requisitos.CifradoEnReposo &&
		requisitos.TamanoMinimoObjeto >= 0
}

func huellaRequisitosAlmacenDocumentalV4(requisitos RequisitosAlmacenObjetos) string {
	valores := []string{
		"vec.documentos.requisitos-almacen.v4",
		strconv.FormatBool(requisitos.EscrituraEnFlujo),
		strconv.FormatBool(requisitos.LecturaEnFlujo),
		strconv.FormatBool(requisitos.ReferenciasOpacas),
		strconv.FormatBool(requisitos.IntegridadSHA256),
		strconv.FormatBool(requisitos.Versionado),
		strconv.FormatBool(requisitos.Retencion),
		strconv.FormatBool(requisitos.BloqueoLegal),
		strconv.FormatBool(requisitos.PromocionAtomica),
		strconv.FormatBool(requisitos.CargaDirectaTemporal),
		strconv.FormatBool(requisitos.CifradoEnTransito),
		strconv.FormatBool(requisitos.CifradoEnReposo),
		strconv.FormatBool(requisitos.CifradoPorObjeto),
		strconv.FormatInt(requisitos.TamanoMinimoObjeto, 10),
		strconv.FormatBool(requisitos.PreservaObjetoOriginal),
	}
	return huellaSHA256MaterialDocumental(
		serializarValoresLongitudPrefijadaMaterialDocumental(valores),
	)
}

func huellaCapacidadesAlmacenDocumentalV4(capacidades CapacidadesAlmacenObjetos) string {
	if !referenciaOpacaAlmacenValida(capacidades.ConectorID, 128) ||
		capacidades.TamanoMaximoObjeto < 1 ||
		!origenesCapacidadesAlmacenDocumentalV4Validos(capacidades) {
		return ""
	}
	valores := []string{
		"vec.documentos.capacidades-almacen.v4", capacidades.ConectorID,
		strconv.FormatBool(capacidades.EscrituraEnFlujo),
		strconv.FormatBool(capacidades.LecturaEnFlujo),
		strconv.FormatBool(capacidades.ReferenciasOpacas),
		strconv.FormatBool(capacidades.IntegridadSHA256),
		strconv.FormatBool(capacidades.Versionado),
		strconv.FormatBool(capacidades.Retencion),
		strconv.FormatBool(capacidades.BloqueoLegal),
		strconv.FormatBool(capacidades.PromocionAtomica),
		strconv.FormatBool(capacidades.CargaDirectaTemporal),
		strconv.FormatBool(capacidades.CifradoEnTransito),
		strconv.FormatBool(capacidades.CifradoEnReposo),
		strconv.FormatBool(capacidades.CifradoPorObjeto),
		strconv.FormatInt(capacidades.TamanoMaximoObjeto, 10),
		strconv.FormatBool(capacidades.PreservaObjetoOriginal),
		strconv.Itoa(len(capacidades.OrigenesCargaDirecta)),
	}
	valores = append(valores, capacidades.OrigenesCargaDirecta...)
	return huellaSHA256MaterialDocumental(
		serializarValoresLongitudPrefijadaMaterialDocumental(valores),
	)
}

func origenesCapacidadesAlmacenDocumentalV4Validos(
	capacidades CapacidadesAlmacenObjetos,
) bool {
	origenes := capacidades.OrigenesCargaDirecta
	if !capacidades.CargaDirectaTemporal {
		return len(origenes) == 0
	}
	if len(origenes) == 0 || len(origenes) > 32 {
		return false
	}
	total := 0
	for _, origen := range origenes {
		if len(origen) == 0 || len(origen) > maximoBytesOrigenCargaDirectaV4 ||
			total > maximoBytesOrigenesCargaDirectaV4-len(origen) {
			return false
		}
		total += len(origen)
	}
	return origenesCargaDirectaValidos(origenes)
}

func clonarCapacidadesAlmacenDocumentalV4(
	capacidades CapacidadesAlmacenObjetos,
) CapacidadesAlmacenObjetos {
	copia := capacidades
	copia.OrigenesCargaDirecta = append([]string(nil), capacidades.OrigenesCargaDirecta...)
	return copia
}

func clonarDatosPreparacionEscrituraAlmacenDocumentalV4(
	datos *datosPreparacionEscrituraAlmacenDocumentalV4,
) *datosPreparacionEscrituraAlmacenDocumentalV4 {
	if datos == nil {
		return nil
	}
	copia := *datos
	copia.capacidades = clonarCapacidadesAlmacenDocumentalV4(datos.capacidades)
	copia.vinculoEjecucion = clonarVinculoEjecucionEscrituraAlmacenDocumentalV4(datos.vinculoEjecucion)
	return &copia
}

func clonarVinculoEjecucionEscrituraAlmacenDocumentalV4(
	vinculo VinculoEjecucionEscrituraAlmacenDocumental,
) VinculoEjecucionEscrituraAlmacenDocumental {
	if vinculo.datos == nil {
		return VinculoEjecucionEscrituraAlmacenDocumental{}
	}
	copia := *vinculo.datos
	return VinculoEjecucionEscrituraAlmacenDocumental{datos: &copia}
}

func valoresSolicitudEscrituraAlmacenDocumentalV4(
	solicitud instantaneaSolicitudEscrituraAlmacenDocumentalV4,
) []string {
	contexto := solicitud.Contexto
	return []string{
		"vec.documentos.solicitud-escritura-almacen.v4",
		contexto.Esquema, contexto.OperacionRef, contexto.CorrelacionRef,
		contexto.AutorizacionRef, contexto.Finalidad, contexto.Clasificacion,
		contexto.AccionNegocio, contexto.AccionTecnica, contexto.CargaRef,
		contexto.SujetoSeudonimoHMAC, contexto.RecursoRef, contexto.ModuloID,
		contexto.TipoRecurso, contexto.HuellaRecursoSHA256,
		contexto.HuellaSolicitudHMAC, contexto.EfectoRef,
		contexto.HuellaPlanEfectoSHA256, contexto.HuellaManifiestoSHA256,
		contexto.HuellaPasoSHA256, string(contexto.PasoRef),
		contexto.ObjetoVinculado.Referencia, contexto.ObjetoVinculado.Version,
		contexto.HuellaDecisionSHA256,
		contexto.VerificadaEn.Format(time.RFC3339Nano),
		contexto.ValidaHasta.Format(time.RFC3339Nano),
		solicitud.ClaveIdempotencia, string(solicitud.Zona), solicitud.MIME,
		strconv.FormatInt(solicitud.Tamano, 10), solicitud.HuellaSHA256,
	}
}

func huellaSolicitudEscrituraAlmacenDocumentalV4(
	solicitud instantaneaSolicitudEscrituraAlmacenDocumentalV4,
) string {
	return huellaSHA256MaterialDocumental(serializarValoresLongitudPrefijadaMaterialDocumental(
		valoresSolicitudEscrituraAlmacenDocumentalV4(solicitud),
	))
}

// canonizarContenidoEntradaNeutralDocumental usa un codec binario cerrado del
// modelo neutral actual. Cada campo se codifica por longitud en bytes y se
// conserva el orden significativo de los parrafos. No interpreta JSON, no
// normaliza Unicode ni depende de reglas implicitas de una biblioteca.
func canonizarContenidoEntradaNeutralDocumental(
	contenido domain.ContenidoDocumento,
) ([]byte, error) {
	if len(contenido.Parrafos) > maximosParrafosEntradaNeutral ||
		!textoEntradaNeutralDocumentalValido(contenido.Titulo) {
		return nil, ErrEntradaNeutralDocumentalInvalida
	}
	if contenido.Titulo == "" && len(contenido.Parrafos) == 0 {
		return nil, ErrEntradaNeutralDocumentalInvalida
	}
	contador := strconv.Itoa(len(contenido.Parrafos))
	tamano := 0
	reservar := func(valor string) bool {
		for _, incremento := range []int{len(strconv.Itoa(len(valor))), 1, len(valor), 1} {
			if incremento < 0 || incremento > maximoBytesEntradaNeutralDocumental ||
				tamano > maximoBytesEntradaNeutralDocumental-incremento {
				return false
			}
			tamano += incremento
		}
		return true
	}
	if !reservar(EsquemaCanonizacionEntradaNeutralDocumentalV1) ||
		!reservar(contenido.Titulo) || !reservar(contador) {
		return nil, ErrEntradaNeutralDocumentalInvalida
	}
	for _, parrafo := range contenido.Parrafos {
		if !textoEntradaNeutralDocumentalValido(parrafo) || !reservar(parrafo) {
			return nil, ErrEntradaNeutralDocumentalInvalida
		}
	}
	canonico := make([]byte, 0, tamano)
	anadir := func(valor string) {
		canonico = strconv.AppendInt(canonico, int64(len(valor)), 10)
		canonico = append(canonico, ':')
		canonico = append(canonico, valor...)
		canonico = append(canonico, '\n')
	}
	anadir(EsquemaCanonizacionEntradaNeutralDocumentalV1)
	anadir(contenido.Titulo)
	anadir(contador)
	for _, parrafo := range contenido.Parrafos {
		anadir(parrafo)
	}
	if len(canonico) != tamano {
		return nil, ErrEntradaNeutralDocumentalInvalida
	}
	return canonico, nil
}

func validarDatosCanonicosEntradaNeutralDocumental(
	canonicalizacionRef string,
	contenido domain.ContenidoDocumento,
	contenidoCanonico []byte,
	tamano uint64,
) error {
	if canonicalizacionRef != EsquemaCanonizacionEntradaNeutralDocumentalV1 ||
		len(contenidoCanonico) == 0 || uint64(len(contenidoCanonico)) != tamano ||
		tamano > maximoBytesEntradaNeutralDocumental {
		return ErrEntradaNeutralDocumentalInvalida
	}
	canonico, err := canonizarContenidoEntradaNeutralDocumental(contenido)
	if err != nil || !bytes.Equal(canonico, contenidoCanonico) {
		return ErrEntradaNeutralDocumentalInvalida
	}
	return nil
}

func clonarContenidoEntradaNeutralDocumental(
	contenido domain.ContenidoDocumento,
) domain.ContenidoDocumento {
	return domain.ContenidoDocumento{
		Titulo: contenido.Titulo, Parrafos: append([]string(nil), contenido.Parrafos...),
	}
}

func textoEntradaNeutralDocumentalValido(valor string) bool {
	if !utf8.ValidString(valor) {
		return false
	}
	for _, caracter := range valor {
		if caracter == '\t' || caracter == '\n' || caracter == '\r' {
			continue
		}
		if caracter < 0x20 || caracter == 0x7f {
			return false
		}
	}
	return true
}

func serializarDeclaracionEscrituraAlmacenDocumental(
	d DeclaracionEscrituraAlmacenDocumental,
) []byte {
	if d.Validar() != nil {
		return nil
	}
	datos := d.datos
	vinculo := datos.vinculoEjecucion.datos
	if vinculo == nil {
		return nil
	}
	objeto := datos.resultado.Objeto
	evidencia := datos.resultado.Evidencia
	valores := []string{
		EsquemaPruebaEscrituraAlmacenDocumentalV2,
	}
	valores = append(valores, valoresSolicitudEscrituraAlmacenDocumentalV4(datos.solicitud)...)
	valores = append(valores,
		datos.huellaSolicitudSHA256,
		vinculo.ReservaRef, vinculo.BorradorRef, vinculo.EfectoRef,
		vinculo.HuellaPlanSHA256, vinculo.DecisionRef, vinculo.HuellaDecisionSHA256,
		strconv.FormatUint(vinculo.SecuenciaCercado, 10), vinculo.HuellaVinculoCercadoSHA256,
		vinculo.VerificacionCercadoRef, vinculo.VerificacionCercadoEn.Format(time.RFC3339Nano),
		objeto.Objeto.Referencia, objeto.Objeto.Version, objeto.ConectorID,
		string(objeto.Zona), objeto.MIME, objeto.HuellaSHA256,
		strconv.FormatInt(objeto.Tamano, 10), objeto.EvidenciaCreacionRef,
		objeto.AlmacenadoEn.Format(time.RFC3339Nano), objeto.RetenidoHasta.Format(time.RFC3339Nano),
		strconv.FormatBool(objeto.Inmovilizado), strconv.FormatBool(objeto.Eliminado),
	)
	valores = append(valores, valoresEvidenciaOperacionAlmacenDocumentalV4(evidencia)...)
	valores = append(valores,
		datos.huellaEvidenciaOperacionSHA256,
		datos.salida.HuellaSHA256, strconv.FormatUint(datos.salida.Tamano, 10),
		strconv.FormatUint(datos.salida.LimiteBytes, 10),
		datos.capacidades.ConectorID,
		strconv.FormatBool(datos.capacidades.EscrituraEnFlujo),
		strconv.FormatBool(datos.capacidades.LecturaEnFlujo),
		strconv.FormatBool(datos.capacidades.ReferenciasOpacas),
		strconv.FormatBool(datos.capacidades.IntegridadSHA256),
		strconv.FormatBool(datos.capacidades.Versionado),
		strconv.FormatBool(datos.capacidades.Retencion),
		strconv.FormatBool(datos.capacidades.BloqueoLegal),
		strconv.FormatBool(datos.capacidades.PromocionAtomica),
		strconv.FormatBool(datos.capacidades.CargaDirectaTemporal),
		strconv.FormatBool(datos.capacidades.CifradoEnTransito),
		strconv.FormatBool(datos.capacidades.CifradoEnReposo),
		strconv.FormatBool(datos.capacidades.CifradoPorObjeto),
		strconv.FormatInt(datos.capacidades.TamanoMaximoObjeto, 10),
		strconv.FormatBool(datos.capacidades.PreservaObjetoOriginal),
		strconv.Itoa(len(datos.capacidades.OrigenesCargaDirecta)),
	)
	valores = append(valores, datos.capacidades.OrigenesCargaDirecta...)
	politica := datos.politica
	requisitos := politica.Requisitos
	valores = append(valores,
		politica.PoliticaRef, strconv.FormatUint(politica.Version, 10), politica.HuellaSHA256,
		politica.HuellaRequisitosSHA256, politica.HuellaCapacidadesSHA256,
		politica.RetencionHasta.Format(time.RFC3339Nano),
		strconv.FormatBool(politica.ExigeInmovilizacionInicial),
		strconv.FormatBool(requisitos.EscrituraEnFlujo),
		strconv.FormatBool(requisitos.LecturaEnFlujo),
		strconv.FormatBool(requisitos.ReferenciasOpacas),
		strconv.FormatBool(requisitos.IntegridadSHA256),
		strconv.FormatBool(requisitos.Versionado),
		strconv.FormatBool(requisitos.Retencion),
		strconv.FormatBool(requisitos.BloqueoLegal),
		strconv.FormatBool(requisitos.PromocionAtomica),
		strconv.FormatBool(requisitos.CargaDirectaTemporal),
		strconv.FormatBool(requisitos.CifradoEnTransito),
		strconv.FormatBool(requisitos.CifradoEnReposo),
		strconv.FormatBool(requisitos.CifradoPorObjeto),
		strconv.FormatInt(requisitos.TamanoMinimoObjeto, 10),
		strconv.FormatBool(requisitos.PreservaObjetoOriginal),
	)
	return serializarValoresLongitudPrefijadaMaterialDocumental(valores)
}

func huellaSHA256MaterialDocumental(contenido []byte) string {
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:])
}

func serializarValoresLongitudPrefijadaMaterialDocumental(valores []string) []byte {
	var contenido []byte
	for _, valor := range valores {
		contenido = strconv.AppendInt(contenido, int64(len(valor)), 10)
		contenido = append(contenido, ':')
		contenido = append(contenido, valor...)
		contenido = append(contenido, '\n')
	}
	return contenido
}

func huellaEvidenciaOperacionAlmacenDocumental(evidencia EvidenciaOperacionAlmacen) string {
	if evidencia.Validar() != nil {
		return ""
	}
	contenido := serializarValoresLongitudPrefijadaMaterialDocumental(
		valoresEvidenciaOperacionAlmacenDocumentalV4(evidencia),
	)
	return huellaSHA256MaterialDocumental(contenido)
}

func valoresEvidenciaOperacionAlmacenDocumentalV4(
	evidencia EvidenciaOperacionAlmacen,
) []string {
	return []string{
		"vec.documentos.huella-evidencia-operacion-almacen.v1",
		evidencia.Referencia, evidencia.ConectorID, evidencia.EsquemaContexto,
		evidencia.AccionNegocio, evidencia.Accion, evidencia.EfectoRef,
		evidencia.HuellaPlanEfectoSHA256, evidencia.HuellaManifiestoSHA256,
		evidencia.HuellaPasoSHA256, string(evidencia.PasoRef),
		evidencia.HuellaDecisionSHA256, evidencia.Objeto.Referencia,
		evidencia.Objeto.Version, evidencia.OperacionRef, evidencia.CorrelacionRef,
		evidencia.AutorizacionRef, evidencia.Finalidad, evidencia.Clasificacion,
		evidencia.RealizadaEn.Format(time.RFC3339Nano), evidencia.CargaRef,
		evidencia.SujetoSeudonimoHMAC, evidencia.RecursoRef, evidencia.ModuloID,
		evidencia.HuellaSolicitudHMAC, evidencia.FundamentoRef,
		strconv.FormatBool(evidencia.ReintentoIdempotente),
	}
}

func huellasMaterialDocumentalV4Distintas(huellas ...string) bool {
	vistas := make(map[string]struct{}, len(huellas))
	for _, huella := range huellas {
		if !esSHA256Hexadecimal(huella) {
			return false
		}
		if _, repetida := vistas[huella]; repetida {
			return false
		}
		vistas[huella] = struct{}{}
	}
	return true
}

func referenciaMaterialDocumentalV4Valida(valor string) bool {
	return referenciaEjecucionDocumentalV3Valida(valor)
}

func interfazMaterialDocumentalNula(valor any) bool {
	if valor == nil {
		return true
	}
	reflejado := reflect.ValueOf(valor)
	switch reflejado.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejado.IsNil()
	default:
		return false
	}
}
