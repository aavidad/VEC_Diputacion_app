package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	esquemaPeticionCobertura  = "VEC-CT-FUENTE-COBERTURA-PETICION-V1"
	esquemaRespuestaCobertura = "VEC-CT-FUENTE-COBERTURA-RESPUESTA-V1"
)

// DatosResultadoConsultaCobertura contiene únicamente datos funcionales y
// referencias opacas. La atestación se añade después de canonizar estos datos.
type DatosResultadoConsultaCobertura struct {
	PeticionRef          string
	HuellaPeticionSHA256 string
	OrganizacionRef      string
	ExpedienteRef        string
	VersionExpediente    uint64
	Catalogo             domain.IdentidadCatalogoViasCobertura
	ViaClave             domain.ClaveCatalogo
	ProcedenciaClave     domain.ClaveCatalogo
	CategoriaRef         string
	Periodo              domain.PeriodoPrevisto
	Comprobacion         domain.ComprobacionCobertura
	DefinicionFuenteRef  string
}

func (d DatosResultadoConsultaCobertura) validarPara(
	solicitud SolicitudConsultarCobertura,
) error {
	huellaPeticion, errHuella := huellaPeticionCobertura(solicitud)
	if solicitud.Validar() != nil ||
		errHuella != nil ||
		d.PeticionRef != solicitud.PeticionRef ||
		d.HuellaPeticionSHA256 != huellaPeticion ||
		d.OrganizacionRef != solicitud.OrganizacionRef ||
		d.ExpedienteRef != solicitud.ExpedienteRef ||
		d.VersionExpediente != solicitud.VersionExpediente ||
		!d.Catalogo.CoincideExactamente(solicitud.Catalogo) ||
		d.ViaClave != solicitud.ViaClave ||
		d.ProcedenciaClave != solicitud.Comprobacion.Procedencia.Clave ||
		d.CategoriaRef != solicitud.CategoriaRef ||
		!d.Periodo.Inicio.Equal(solicitud.Periodo.Inicio) ||
		!d.Periodo.Fin.Equal(solicitud.Periodo.Fin) ||
		d.Comprobacion.Validar() != nil ||
		d.Comprobacion.Detalle != "" ||
		d.Comprobacion.Clave != solicitud.Comprobacion.Clave ||
		d.DefinicionFuenteRef !=
			solicitud.Comprobacion.Procedencia.DefinicionFuenteRef ||
		d.Comprobacion.EvaluadaEn.Before(solicitud.SolicitadaEn) {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

// PreimagenRespuestaCobertura es la representación canónica completa que firma
// la fuente y verifica una autoridad institucional independiente.
type PreimagenRespuestaCobertura struct {
	contenido []byte
}

func (p PreimagenRespuestaCobertura) Bytes() ([]byte, error) {
	if len(p.contenido) == 0 || len(p.contenido) > 64*1024 {
		return nil, ErrResultadoFuenteCoberturaNoConfiable
	}
	return append([]byte(nil), p.contenido...), nil
}

func (p PreimagenRespuestaCobertura) huellaSHA256() (string, error) {
	contenido, err := p.Bytes()
	if err != nil {
		return "", err
	}
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:]), nil
}

func (PreimagenRespuestaCobertura) String() string {
	return "[PREIMAGEN-RESPUESTA-COBERTURA-REDACTADA]"
}

func (p PreimagenRespuestaCobertura) GoString() string { return p.String() }
func (p PreimagenRespuestaCobertura) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, p.String())
}
func (p PreimagenRespuestaCobertura) LogValue() slog.Value {
	return slog.StringValue(p.String())
}

// NuevaPreimagenRespuestaCobertura permite al adaptador de fuente sellar
// exactamente la misma representación que consumirá el núcleo.
func NuevaPreimagenRespuestaCobertura(
	datos DatosResultadoConsultaCobertura,
	metadatos MetadatosAtestacionRespuestaCobertura,
) (PreimagenRespuestaCobertura, error) {
	contenido, err := canonRespuestaCobertura(datos, metadatos)
	if err != nil {
		return PreimagenRespuestaCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return PreimagenRespuestaCobertura{contenido: contenido}, nil
}

// ResultadoConsultaCobertura es inmutable: conserva copias del contenido
// funcional, la representación canónica y la atestación que la sella.
type ResultadoConsultaCobertura struct {
	datos      *DatosResultadoConsultaCobertura
	preimagen  PreimagenRespuestaCobertura
	atestacion AtestacionRespuestaCobertura
}

func NuevoResultadoConsultaCobertura(
	datos DatosResultadoConsultaCobertura,
	atestacion AtestacionRespuestaCobertura,
) (ResultadoConsultaCobertura, error) {
	preimagen, err := NuevaPreimagenRespuestaCobertura(
		datos,
		atestacion.Metadatos,
	)
	if err != nil || atestacion.Validar() != nil ||
		atestacion.Metadatos.AutoridadRef != datos.Comprobacion.FuenteRef ||
		atestacion.Metadatos.ReciboRef != datos.Comprobacion.ReciboRef {
		return ResultadoConsultaCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	copia := datos
	return ResultadoConsultaCobertura{
		datos: &copia, preimagen: preimagen, atestacion: atestacion,
	}, nil
}

func (r ResultadoConsultaCobertura) ValidarPara(
	solicitud SolicitudConsultarCobertura,
) error {
	if r.datos == nil || r.datos.validarPara(solicitud) != nil ||
		r.atestacion.Validar() != nil ||
		r.atestacion.Metadatos.AutoridadRef !=
			r.datos.Comprobacion.FuenteRef ||
		r.atestacion.Metadatos.ReciboRef != r.datos.Comprobacion.ReciboRef {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	recalculada, err := NuevaPreimagenRespuestaCobertura(
		*r.datos,
		r.atestacion.Metadatos,
	)
	if err != nil || !bytes.Equal(
		recalculada.contenido,
		r.preimagen.contenido,
	) {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

func (r ResultadoConsultaCobertura) Datos() (
	DatosResultadoConsultaCobertura,
	error,
) {
	if r.datos == nil {
		return DatosResultadoConsultaCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return *r.datos, nil
}

func (r ResultadoConsultaCobertura) Atestacion() (
	AtestacionRespuestaCobertura,
	error,
) {
	if r.datos == nil || r.atestacion.Validar() != nil {
		return AtestacionRespuestaCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return r.atestacion, nil
}

func (r ResultadoConsultaCobertura) SolicitudVerificacion() (
	SolicitudVerificarRespuestaCobertura,
	error,
) {
	datos, err := r.Datos()
	if err != nil {
		return SolicitudVerificarRespuestaCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return nuevaSolicitudVerificarRespuestaCobertura(
		datos.HuellaPeticionSHA256,
		r.preimagen,
		r.atestacion,
	)
}

func (ResultadoConsultaCobertura) String() string {
	return "[RESULTADO-CONSULTA-COBERTURA-REDACTADO]"
}

func (r ResultadoConsultaCobertura) GoString() string { return r.String() }
func (r ResultadoConsultaCobertura) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, r.String())
}
func (r ResultadoConsultaCobertura) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

func canonPeticionCobertura(
	solicitud SolicitudConsultarCobertura,
) ([]byte, error) {
	if solicitud.Validar() != nil {
		return nil, ErrPeticionFuenteCoberturaInvalida
	}
	escritor := nuevoEscritorCanonFuenteAnalisis()
	escribirSolicitudCobertura(escritor, solicitud)
	contenido, err := escritor.resultado()
	if err != nil {
		return nil, ErrPeticionFuenteCoberturaInvalida
	}
	return contenido, nil
}

func (s SolicitudConsultarCobertura) MaterialCanonico() ([]byte, error) {
	return canonPeticionCobertura(s)
}

func huellaPeticionCobertura(
	solicitud SolicitudConsultarCobertura,
) (string, error) {
	contenido, err := canonPeticionCobertura(solicitud)
	if err != nil {
		return "", err
	}
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:]), nil
}

func canonRespuestaCobertura(
	datos DatosResultadoConsultaCobertura,
	metadatos MetadatosAtestacionRespuestaCobertura,
) ([]byte, error) {
	if metadatos.Validar() != nil {
		return nil, ErrResultadoFuenteCoberturaNoConfiable
	}
	escritor := nuevoEscritorCanonFuenteAnalisis()
	escritor.texto(esquemaRespuestaCobertura)
	escribirDatosResultadoCobertura(escritor, datos)
	escritor.texto(metadatos.AutoridadRef)
	escritor.entero64(uint64(metadatos.Generacion))
	escritor.texto(metadatos.ReciboRef)
	escritor.instante(metadatos.EmitidaEn)
	escritor.instante(metadatos.ValidaHasta)
	contenido, err := escritor.resultado()
	if err != nil {
		return nil, ErrResultadoFuenteCoberturaNoConfiable
	}
	return contenido, nil
}

func escribirSolicitudCobertura(
	escritor *escritorCanonFuenteAnalisis,
	s SolicitudConsultarCobertura,
) {
	escritor.texto(esquemaPeticionCobertura)
	escritor.texto(s.PeticionRef)
	escritor.texto(s.OrganizacionRef)
	escritor.texto(s.ExpedienteRef)
	escritor.entero64(s.VersionExpediente)
	escritor.texto(s.Catalogo.Referencia)
	escritor.entero64(s.Catalogo.Version)
	escritor.texto(s.Catalogo.HuellaSHA256)
	escritor.texto(string(s.ViaClave))
	escritor.texto(string(s.Comprobacion.Clave))
	escritor.entero16(s.Comprobacion.Orden)
	escritor.booleano(s.Comprobacion.Obligatoria)
	escritor.texto(string(s.Comprobacion.Procedencia.Clave))
	escritor.texto(s.Comprobacion.Procedencia.DefinicionFuenteRef)
	escritor.texto(s.CategoriaRef)
	escritor.instante(s.Periodo.Inicio)
	escritor.instante(s.Periodo.Fin)
	escritor.instante(s.SolicitadaEn)
}

func escribirDatosResultadoCobertura(
	escritor *escritorCanonFuenteAnalisis,
	d DatosResultadoConsultaCobertura,
) {
	escritor.texto(d.PeticionRef)
	escritor.texto(d.HuellaPeticionSHA256)
	escritor.texto(d.OrganizacionRef)
	escritor.texto(d.ExpedienteRef)
	escritor.entero64(d.VersionExpediente)
	escritor.texto(d.Catalogo.Referencia)
	escritor.entero64(d.Catalogo.Version)
	escritor.texto(d.Catalogo.HuellaSHA256)
	escritor.texto(string(d.ViaClave))
	escritor.texto(string(d.ProcedenciaClave))
	escritor.texto(d.CategoriaRef)
	escritor.instante(d.Periodo.Inicio)
	escritor.instante(d.Periodo.Fin)
	escritor.texto(string(d.Comprobacion.Clave))
	escritor.texto(string(d.Comprobacion.Resultado))
	escritor.texto(d.Comprobacion.FuenteRef)
	escritor.texto(d.Comprobacion.ReciboRef)
	escritor.instante(d.Comprobacion.EvaluadaEn)
	escritor.texto(d.Comprobacion.Detalle)
	escritor.texto(d.DefinicionFuenteRef)
}
