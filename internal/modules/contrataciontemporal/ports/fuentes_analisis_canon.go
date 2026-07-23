package ports

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	esquemaPeticionValidacionRC        = "VEC-CT-FUENTE-ANALISIS-RC-V1"
	esquemaPeticionCalculoCoste        = "VEC-CT-FUENTE-ANALISIS-COSTE-V1"
	dominioSelloPeticionAnalisis       = "hmac-sha256:fuente-analisis-v1:"
	maximoAniosPeriodoFuente           = 100
	maximoCentimosFuente         int64 = 922_337_203_685_477
)

var patronPeticionFuenteAnalisis = regexp.MustCompile(
	`^pet_[A-Za-z0-9_-]{22,128}$`,
)

type PreimagenPeticionFuenteAnalisis struct {
	contenido []byte
}

func (p PreimagenPeticionFuenteAnalisis) Bytes() ([]byte, error) {
	if len(p.contenido) == 0 || len(p.contenido) > 64*1024 {
		return nil, ErrPeticionFuenteAnalisisInvalida
	}
	return append([]byte(nil), p.contenido...), nil
}

func (PreimagenPeticionFuenteAnalisis) String() string {
	return "[PREIMAGEN-FUENTE-ANALISIS-REDACTADA]"
}

func (p PreimagenPeticionFuenteAnalisis) GoString() string {
	return p.String()
}

func (p PreimagenPeticionFuenteAnalisis) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}

func (p PreimagenPeticionFuenteAnalisis) LogValue() slog.Value {
	return slog.StringValue(p.String())
}

type DatosSolicitudValidarRC struct {
	PeticionRef        string
	HuellaPeticionHMAC string
	OrganizacionRef    string
	ExpedienteRef      string
	VersionExpediente  uint64
	Entrada            domain.VinculoEntradaRC
	Declaracion        domain.DeclaracionRC
	SolicitadaEn       time.Time
}

func (d DatosSolicitudValidarRC) validar() error {
	if !referenciaPeticionFuenteAnalisisValida(d.PeticionRef) ||
		!selloPeticionFuenteAnalisisValido(d.HuellaPeticionHMAC) ||
		!domain.ReferenciaOpacaValida(d.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(d.ExpedienteRef) ||
		d.VersionExpediente == 0 || d.Entrada.Validar() != nil ||
		d.Declaracion.Validar() != nil ||
		!importeFuenteAnalisisValidoDeclaracion(d.Declaracion) ||
		!instanteFuenteAnalisisCanonico(d.SolicitadaEn) {
		return ErrPeticionFuenteAnalisisInvalida
	}
	return nil
}

type SolicitudValidarRC struct {
	datos     *DatosSolicitudValidarRC
	preimagen []byte
}

func (s SolicitudValidarRC) Validar() error {
	if s.datos == nil || s.datos.validar() != nil {
		return ErrPeticionFuenteAnalisisInvalida
	}
	canonica := s.datosCanonicos()
	if len(canonica) == 0 || !bytes.Equal(canonica, s.preimagen) {
		return ErrPeticionFuenteAnalisisInvalida
	}
	return nil
}

func (s SolicitudValidarRC) datosCanonicos() []byte {
	if s.datos == nil {
		return nil
	}
	copia := *s.datos
	copia.HuellaPeticionHMAC = ""
	contenido, _ := canonPeticionValidacionRC(copia)
	return contenido
}

func (s SolicitudValidarRC) Datos() (DatosSolicitudValidarRC, error) {
	if s.Validar() != nil {
		return DatosSolicitudValidarRC{}, ErrPeticionFuenteAnalisisInvalida
	}
	return *s.datos, nil
}

func (SolicitudValidarRC) String() string {
	return "[SOLICITUD-VALIDAR-RC-REDACTADA]"
}

func (s SolicitudValidarRC) GoString() string {
	return s.String()
}

func (s SolicitudValidarRC) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}

func (s SolicitudValidarRC) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

type DatosSolicitudCalcularCoste struct {
	PeticionRef        string
	HuellaPeticionHMAC string
	OrganizacionRef    string
	ExpedienteRef      string
	VersionExpediente  uint64
	CategoriaRef       string
	GrupoSubgrupo      string
	ModalidadClave     domain.ClaveCatalogo
	CausaClave         domain.ClaveCatalogo
	Periodo            domain.PeriodoPrevisto
	Jornada            domain.JornadaDiezmilesimas
	SolicitadaEn       time.Time
}

func (d DatosSolicitudCalcularCoste) validar() error {
	if !referenciaPeticionFuenteAnalisisValida(d.PeticionRef) ||
		!selloPeticionFuenteAnalisisValido(d.HuellaPeticionHMAC) ||
		!domain.ReferenciaOpacaValida(d.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(d.ExpedienteRef) ||
		d.VersionExpediente == 0 ||
		!domain.ReferenciaOpacaValida(d.CategoriaRef) ||
		!domain.GrupoSubgrupoValido(d.GrupoSubgrupo) ||
		!d.ModalidadClave.Valida() || !d.CausaClave.Valida() ||
		!periodoFuenteAnalisisValido(d.Periodo) ||
		d.Jornada.Validar() != nil ||
		!instanteFuenteAnalisisCanonico(d.SolicitadaEn) {
		return ErrPeticionFuenteAnalisisInvalida
	}
	return nil
}

type SolicitudCalcularCoste struct {
	datos     *DatosSolicitudCalcularCoste
	preimagen []byte
}

func (s SolicitudCalcularCoste) Validar() error {
	if s.datos == nil || s.datos.validar() != nil {
		return ErrPeticionFuenteAnalisisInvalida
	}
	canonica := s.datosCanonicos()
	if len(canonica) == 0 || !bytes.Equal(canonica, s.preimagen) {
		return ErrPeticionFuenteAnalisisInvalida
	}
	return nil
}

func (s SolicitudCalcularCoste) datosCanonicos() []byte {
	if s.datos == nil {
		return nil
	}
	copia := *s.datos
	copia.HuellaPeticionHMAC = ""
	contenido, _ := canonPeticionCalculoCoste(copia)
	return contenido
}

func (s SolicitudCalcularCoste) Datos() (DatosSolicitudCalcularCoste, error) {
	if s.Validar() != nil {
		return DatosSolicitudCalcularCoste{}, ErrPeticionFuenteAnalisisInvalida
	}
	return *s.datos, nil
}

func (SolicitudCalcularCoste) String() string {
	return "[SOLICITUD-CALCULAR-COSTE-REDACTADA]"
}

func (s SolicitudCalcularCoste) GoString() string {
	return s.String()
}

func (s SolicitudCalcularCoste) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}

func (s SolicitudCalcularCoste) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

func canonPeticionValidacionRC(datos DatosSolicitudValidarRC) ([]byte, error) {
	if datos.HuellaPeticionHMAC != "" {
		return nil, ErrPeticionFuenteAnalisisInvalida
	}
	escritor := nuevoEscritorCanonFuenteAnalisis()
	escritor.texto(esquemaPeticionValidacionRC)
	escritor.texto(datos.PeticionRef)
	escritor.texto(datos.OrganizacionRef)
	escritor.texto(datos.ExpedienteRef)
	escritor.entero64(datos.VersionExpediente)
	escritor.texto(datos.Entrada.Referencia)
	escritor.texto(datos.Entrada.HuellaSHA256)
	escritor.booleano(datos.Declaracion.Existe)
	escritor.texto(datos.Declaracion.Numero)
	escritor.instante(datos.Declaracion.Fecha)
	escritor.entero64(uint64(datos.Declaracion.Importe.Centimos))
	escritor.texto(datos.Declaracion.Importe.Moneda)
	escritor.texto(datos.Declaracion.DocumentoRef)
	escritor.instante(datos.SolicitadaEn)
	return escritor.resultado()
}

func canonPeticionCalculoCoste(datos DatosSolicitudCalcularCoste) ([]byte, error) {
	if datos.HuellaPeticionHMAC != "" {
		return nil, ErrPeticionFuenteAnalisisInvalida
	}
	escritor := nuevoEscritorCanonFuenteAnalisis()
	escritor.texto(esquemaPeticionCalculoCoste)
	escritor.texto(datos.PeticionRef)
	escritor.texto(datos.OrganizacionRef)
	escritor.texto(datos.ExpedienteRef)
	escritor.entero64(datos.VersionExpediente)
	escritor.texto(datos.CategoriaRef)
	escritor.texto(datos.GrupoSubgrupo)
	escritor.texto(string(datos.ModalidadClave))
	escritor.texto(string(datos.CausaClave))
	escritor.instante(datos.Periodo.Inicio)
	escritor.instante(datos.Periodo.Fin)
	escritor.entero16(uint16(datos.Jornada))
	escritor.instante(datos.SolicitadaEn)
	return escritor.resultado()
}

func referenciaPeticionFuenteAnalisisValida(valor string) bool {
	return patronPeticionFuenteAnalisis.MatchString(valor)
}

func selloPeticionFuenteAnalisisValido(valor string) bool {
	return strings.HasPrefix(valor, dominioSelloPeticionAnalisis) &&
		SelloHMACSHA256Valido(valor)
}

func instanteFuenteAnalisisCanonico(valor time.Time) bool {
	return domain.InstanteUTCCanonico(valor) &&
		valor.Year() >= 1 && valor.Year() <= 9999
}

func periodoFuenteAnalisisValido(periodo domain.PeriodoPrevisto) bool {
	return periodo.Validar() == nil &&
		instanteFuenteAnalisisCanonico(periodo.Inicio) &&
		instanteFuenteAnalisisCanonico(periodo.Fin) &&
		!periodo.Fin.After(periodo.Inicio.AddDate(maximoAniosPeriodoFuente, 0, 0))
}

func importeFuenteAnalisisValido(importe domain.Importe) bool {
	return importe.Validar(false) == nil &&
		importe.Centimos <= maximoCentimosFuente
}

func importeFuenteAnalisisValidoOpcional(importe *domain.Importe) bool {
	return importe == nil || importeFuenteAnalisisValido(*importe)
}

func importeFuenteAnalisisValidoDeclaracion(declaracion domain.DeclaracionRC) bool {
	if !declaracion.Existe {
		return declaracion.Importe == (domain.Importe{})
	}
	return importeFuenteAnalisisValido(declaracion.Importe)
}

type escritorCanonFuenteAnalisis struct {
	buffer bytes.Buffer
	err    error
}

func nuevoEscritorCanonFuenteAnalisis() *escritorCanonFuenteAnalisis {
	return &escritorCanonFuenteAnalisis{}
}

func (e *escritorCanonFuenteAnalisis) texto(valor string) {
	if e.err != nil || len(valor) > 64*1024 {
		e.err = ErrPeticionFuenteAnalisisInvalida
		return
	}
	var longitud [4]byte
	binary.BigEndian.PutUint32(longitud[:], uint32(len(valor)))
	_, _ = e.buffer.Write(longitud[:])
	_, _ = e.buffer.WriteString(valor)
}

func (e *escritorCanonFuenteAnalisis) entero16(valor uint16) {
	var contenido [2]byte
	binary.BigEndian.PutUint16(contenido[:], valor)
	_, _ = e.buffer.Write(contenido[:])
}

func (e *escritorCanonFuenteAnalisis) entero64(valor uint64) {
	var contenido [8]byte
	binary.BigEndian.PutUint64(contenido[:], valor)
	_, _ = e.buffer.Write(contenido[:])
}

func (e *escritorCanonFuenteAnalisis) booleano(valor bool) {
	if valor {
		_ = e.buffer.WriteByte(1)
		return
	}
	_ = e.buffer.WriteByte(0)
}

func (e *escritorCanonFuenteAnalisis) instante(valor time.Time) {
	if valor.IsZero() {
		e.entero64(0)
		return
	}
	if !instanteFuenteAnalisisCanonico(valor) {
		e.err = ErrPeticionFuenteAnalisisInvalida
		return
	}
	e.entero64(uint64(valor.UnixMicro()))
}

func (e *escritorCanonFuenteAnalisis) resultado() ([]byte, error) {
	if e.err != nil || e.buffer.Len() == 0 || e.buffer.Len() > 64*1024 {
		return nil, ErrPeticionFuenteAnalisisInvalida
	}
	return append([]byte(nil), e.buffer.Bytes()...), nil
}
