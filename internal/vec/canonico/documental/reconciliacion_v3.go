package documental

import (
	"fmt"
	"io"
	"log/slog"
	"time"
)

const (
	minimoBytesSobreReconciliacionV3 = 16
	maximoBytesSobreReconciliacionV3 = 64 * 1024
)

// EstadoResultadoReconciliacionV3 expresa el resultado cerrado comunicado por
// el conector. Ninguno de sus valores constituye por si solo una comprobacion.
type EstadoResultadoReconciliacionV3 string

const (
	ResultadoReconciliacionV3AplicadoExacto EstadoResultadoReconciliacionV3 = "aplicado_exacto"
	ResultadoReconciliacionV3NoAplicado     EstadoResultadoReconciliacionV3 = "no_aplicado_atestado"
	ResultadoReconciliacionV3Desconocido    EstadoResultadoReconciliacionV3 = "desconocido"
	ResultadoReconciliacionV3Conflictivo    EstadoResultadoReconciliacionV3 = "conflictivo"
)

func (e EstadoResultadoReconciliacionV3) Valido() bool {
	return e == ResultadoReconciliacionV3AplicadoExacto ||
		e == ResultadoReconciliacionV3NoAplicado ||
		e == ResultadoReconciliacionV3Desconocido ||
		e == ResultadoReconciliacionV3Conflictivo
}

// SobreAtestacionReconciliacionV3 conserva el COSE_Sign1 y su compromiso sin
// exponer representacion mutable. Sigue siendo material nominal hasta que una
// dependencia criptografica privada comprueba la atestacion.
type SobreAtestacionReconciliacionV3 struct {
	coseSign1 []byte
	huella    string
}

func NuevoSobreAtestacionReconciliacionV3(
	coseSign1 []byte,
) (SobreAtestacionReconciliacionV3, error) {
	sobre := RestaurarSobreAtestacionReconciliacionV3(
		coseSign1,
		HuellaBytesSHA256(coseSign1),
	)
	if sobre.Validar() != nil {
		return SobreAtestacionReconciliacionV3{}, ErrReconciliacionDocumentalV3Invalida
	}
	return sobre, nil
}

// RestaurarSobreAtestacionReconciliacionV3 permite verificar una forma
// persistida, incluida su huella original. Copia siempre los octetos recibidos.
func RestaurarSobreAtestacionReconciliacionV3(
	coseSign1 []byte,
	huella string,
) SobreAtestacionReconciliacionV3 {
	return SobreAtestacionReconciliacionV3{
		coseSign1: append([]byte(nil), coseSign1...),
		huella:    huella,
	}
}

func (s SobreAtestacionReconciliacionV3) Validar() error {
	if len(s.coseSign1) < minimoBytesSobreReconciliacionV3 ||
		len(s.coseSign1) > maximoBytesSobreReconciliacionV3 ||
		!BytesNoNulos(s.coseSign1) || !SHA256HexadecimalValido(s.huella) ||
		HuellaBytesSHA256(s.coseSign1) != s.huella {
		return ErrReconciliacionDocumentalV3Invalida
	}
	return nil
}

func (s SobreAtestacionReconciliacionV3) COSESign1() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrReconciliacionDocumentalV3Invalida
	}
	return append([]byte(nil), s.coseSign1...), nil
}

func (s SobreAtestacionReconciliacionV3) HuellaSHA256() (string, error) {
	if s.Validar() != nil {
		return "", ErrReconciliacionDocumentalV3Invalida
	}
	return s.huella, nil
}

func (SobreAtestacionReconciliacionV3) String() string {
	return "[ATESTACION-RECONCILIACION-DOCUMENTAL-V3-CRUDA-NO-AUTORITATIVA-REDACTADA]"
}
func (s SobreAtestacionReconciliacionV3) GoString() string { return s.String() }
func (s SobreAtestacionReconciliacionV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SobreAtestacionReconciliacionV3) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
func (SobreAtestacionReconciliacionV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SobreAtestacionReconciliacionV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SobreAtestacionReconciliacionV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SobreAtestacionReconciliacionV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SobreAtestacionReconciliacionV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SobreAtestacionReconciliacionV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// DatosResultadoRenderizadoV3 es la proyeccion pura comprometida por la
// atestacion de reconciliacion. No contiene el documento ni una URL temporal.
type DatosResultadoRenderizadoV3 struct {
	BorradorRef           string
	EfectoRef             string
	ContenidoRef          string
	ContenidoVersion      string
	ConectorRef           string
	MIME                  string
	HuellaSalidaSHA256    string
	TamanoSalida          uint64
	EvidenciaOperacionRef string
}

func (d DatosResultadoRenderizadoV3) EsCero() bool {
	return d == (DatosResultadoRenderizadoV3{})
}

// ExpectativasResultadoReconciliacionV3 contiene exclusivamente los vinculos
// ya revalidados por el puerto. ResultadoAplicadoValido representa la
// validacion rica contra el manifiesto, que permanece fuera del canonico puro.
type ExpectativasResultadoReconciliacionV3 struct {
	ReservaRef              string
	EfectoRef               string
	SecuenciaCercado        uint64
	HuellaVinculoSHA256     string
	HuellaPlanSHA256        string
	ResultadoAplicadoValido bool
}

// DatosResultadoReconciliacionV3 concentra la forma estable que se valida y se
// serializa. El sobre permanece opaco y su huella debe coincidir con la
// atestacion declarada.
type DatosResultadoReconciliacionV3 struct {
	ReservaRef             string
	EfectoRef              string
	SecuenciaCercado       uint64
	HuellaVinculoSHA256    string
	HuellaPlanSHA256       string
	Estado                 EstadoResultadoReconciliacionV3
	Resultado              DatosResultadoRenderizadoV3
	AtestacionRef          string
	HuellaAtestacionSHA256 string
	SobreAtestacion        SobreAtestacionReconciliacionV3
	ConsultadaEn           time.Time
}

func (d DatosResultadoReconciliacionV3) ValidarContra(
	esperado ExpectativasResultadoReconciliacionV3,
) error {
	if d.ReservaRef != esperado.ReservaRef || d.EfectoRef != esperado.EfectoRef ||
		d.SecuenciaCercado != esperado.SecuenciaCercado ||
		d.HuellaVinculoSHA256 != esperado.HuellaVinculoSHA256 ||
		d.HuellaPlanSHA256 != esperado.HuellaPlanSHA256 || !d.Estado.Valido() ||
		!ReferenciaEjecucionV3Valida(d.AtestacionRef) ||
		!SHA256HexadecimalValido(d.HuellaAtestacionSHA256) ||
		d.SobreAtestacion.Validar() != nil || !InstanteV3Valido(d.ConsultadaEn) {
		return ErrReconciliacionDocumentalV3Invalida
	}
	if d.Estado == ResultadoReconciliacionV3AplicadoExacto {
		if !esperado.ResultadoAplicadoValido {
			return ErrReconciliacionDocumentalV3Invalida
		}
	} else if !d.Resultado.EsCero() {
		return ErrReconciliacionDocumentalV3Invalida
	}
	huellaSobre, _ := d.SobreAtestacion.HuellaSHA256()
	if huellaSobre != d.HuellaAtestacionSHA256 {
		return ErrReconciliacionDocumentalV3Invalida
	}
	return nil
}

// Bytes fija, sin alterar su orden historico, la preimagen firmada del
// resultado de reconciliacion.
func (d DatosResultadoReconciliacionV3) Bytes() []byte {
	return SerializarCamposV3([]string{
		"vec.documentos.resultado-reconciliacion.v3", d.ReservaRef, d.EfectoRef,
		Uint64Decimal(d.SecuenciaCercado), d.HuellaVinculoSHA256, d.HuellaPlanSHA256,
		string(d.Estado), d.Resultado.BorradorRef, d.Resultado.EfectoRef,
		d.Resultado.ContenidoRef, d.Resultado.ContenidoVersion, d.Resultado.ConectorRef,
		d.Resultado.MIME, d.Resultado.HuellaSalidaSHA256,
		Uint64Decimal(d.Resultado.TamanoSalida), d.Resultado.EvidenciaOperacionRef,
		d.AtestacionRef, d.ConsultadaEn.Format(time.RFC3339Nano),
	})
}

// HuellaSolicitudVerificacionReconciliacionV3 liga la preimagen exacta con el
// COSE declarado e impide reutilizar una comprobacion con otro sobre.
func HuellaSolicitudVerificacionReconciliacionV3(
	mensaje []byte,
	huellaAtestacionSHA256 string,
) string {
	return HuellaCamposSHA256V3([]string{
		"vec.documentos.solicitud-verificacion-reconciliacion.v3",
		HuellaBytesSHA256(mensaje), huellaAtestacionSHA256,
	})
}
