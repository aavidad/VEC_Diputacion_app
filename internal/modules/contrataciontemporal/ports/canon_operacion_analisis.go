package ports

import (
	"bytes"
	"encoding/binary"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	esquemaAmbitoOperacionAnalisis    = "VEC-CT-ANALISIS-AMBITO-IDEMPOTENCIA-V2"
	esquemaSemanticaOperacionAnalisis = "VEC-CT-ANALISIS-SEMANTICA-ARTEFACTO-O3-V2"
	esquemaConsultaOperacionAnalisis  = "VEC-CT-ANALISIS-CONSULTA-CONFIRMADA-O3-V1"
	maximoBytesCanonOperacionAnalisis = 64 * 1024
)

type DatosPreimagenesOperacionAnalisis struct {
	ClaveIdempotencia   string
	Operacion           TipoOperacionAnalisis
	ActorRef            string
	PerfilRef           string
	MotivoRectificacion domain.ClaveCatalogo
	SolicitudArtefacto  SolicitudPrepararArtefactoAnalisis
	Artefacto           ArtefactoAnalisisPreparado
}

type DatosPreimagenesConsultaOperacionAnalisis struct {
	ClaveIdempotencia   string
	Operacion           TipoOperacionAnalisis
	OrganizacionRef     string
	ExpedienteRef       string
	VersionExpediente   uint64
	ActorRef            string
	PerfilRef           string
	ArtefactoRef        string
	DatosFuncionales    DatosFuncionalesOperacionAnalisis
	MotivoRectificacion domain.ClaveCatalogo
}

// NuevasPreimagenesConsultaOperacionAnalisis sella toda la identidad de la
// petición temprana sin entregarla al adaptador de persistencia.
func NuevasPreimagenesConsultaOperacionAnalisis(
	datos DatosPreimagenesConsultaOperacionAnalisis,
) (PreimagenesOperacionAnalisis, error) {
	if !ClaveIdempotenciaValida(datos.ClaveIdempotencia) ||
		!datos.Operacion.Valida() ||
		!domain.ReferenciaOpacaValida(datos.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(datos.ExpedienteRef) ||
		!VersionOperacionAnalisisConIncrementoValida(
			datos.VersionExpediente,
		) ||
		!domain.ReferenciaOpacaValida(datos.ActorRef) ||
		!domain.ReferenciaOpacaValida(datos.PerfilRef) ||
		!domain.ReferenciaOpacaValida(datos.ArtefactoRef) ||
		datos.DatosFuncionales.Validar() != nil {
		return PreimagenesOperacionAnalisis{},
			ErrOperacionAnalisisInvalida
	}
	if datos.Operacion == OperacionRegistrarAnalisis {
		if datos.MotivoRectificacion != "" {
			return PreimagenesOperacionAnalisis{},
				ErrOperacionAnalisisInvalida
		}
	} else if !datos.MotivoRectificacion.Valida() {
		return PreimagenesOperacionAnalisis{},
			ErrOperacionAnalisisInvalida
	}

	ambito := nuevoCanonOperacionAnalisis()
	escribirAmbitoOperacionAnalisis(
		ambito,
		datos.ClaveIdempotencia,
		datos.OrganizacionRef,
		datos.ExpedienteRef,
		datos.ActorRef,
		datos.PerfilRef,
	)
	bytesAmbito, err := ambito.resultado()
	if err != nil {
		return PreimagenesOperacionAnalisis{}, err
	}

	semantica := nuevoCanonOperacionAnalisis()
	semantica.texto(esquemaConsultaOperacionAnalisis)
	semantica.texto(datos.ClaveIdempotencia)
	semantica.texto(string(datos.Operacion))
	semantica.texto(datos.OrganizacionRef)
	semantica.texto(datos.ExpedienteRef)
	semantica.enteroSinSigno(datos.VersionExpediente)
	semantica.texto(datos.ActorRef)
	semantica.texto(datos.PerfilRef)
	semantica.texto(datos.ArtefactoRef)
	semantica.texto(string(datos.MotivoRectificacion))
	escribirDatosFuncionalesCanonicos(semantica, datos.DatosFuncionales)
	bytesSemantica, err := semantica.resultado()
	if err != nil {
		return PreimagenesOperacionAnalisis{}, err
	}
	return PreimagenesOperacionAnalisis{
		ambito:    bytesAmbito,
		semantica: bytesSemantica,
	}, nil
}

func NuevasPreimagenesOperacionAnalisis(
	datos DatosPreimagenesOperacionAnalisis,
) (PreimagenesOperacionAnalisis, error) {
	artefacto, err := datos.Artefacto.DatosPara(datos.SolicitudArtefacto)
	if err != nil ||
		!ClaveIdempotenciaValida(datos.ClaveIdempotencia) ||
		!datos.Operacion.Valida() ||
		!domain.ReferenciaOpacaValida(datos.ActorRef) ||
		!domain.ReferenciaOpacaValida(datos.PerfilRef) {
		return PreimagenesOperacionAnalisis{}, ErrOperacionAnalisisInvalida
	}
	if datos.Operacion == OperacionRegistrarAnalisis {
		if datos.MotivoRectificacion != "" {
			return PreimagenesOperacionAnalisis{}, ErrOperacionAnalisisInvalida
		}
	} else if !datos.MotivoRectificacion.Valida() {
		return PreimagenesOperacionAnalisis{}, ErrOperacionAnalisisInvalida
	}

	ambito := nuevoCanonOperacionAnalisis()
	escribirAmbitoOperacionAnalisis(
		ambito,
		datos.ClaveIdempotencia,
		artefacto.OrganizacionRef,
		artefacto.ExpedienteRef,
		datos.ActorRef,
		datos.PerfilRef,
	)
	bytesAmbito, err := ambito.resultado()
	if err != nil {
		return PreimagenesOperacionAnalisis{}, err
	}

	semantica := nuevoCanonOperacionAnalisis()
	semantica.texto(esquemaSemanticaOperacionAnalisis)
	semantica.texto(string(datos.Operacion))
	semantica.texto(artefacto.OrganizacionRef)
	semantica.texto(artefacto.ExpedienteRef)
	semantica.enteroSinSigno(artefacto.VersionExpediente)
	semantica.texto(datos.ActorRef)
	semantica.texto(datos.PerfilRef)
	semantica.texto(string(datos.MotivoRectificacion))
	escribirDatosFuncionalesCanonicos(
		semantica,
		artefacto.DatosFuncionales,
	)
	escribirArtefactoCanonico(semantica, artefacto)
	bytesSemantica, err := semantica.resultado()
	if err != nil {
		return PreimagenesOperacionAnalisis{}, err
	}
	return PreimagenesOperacionAnalisis{
		ambito:    bytesAmbito,
		semantica: bytesSemantica,
	}, nil
}

func escribirAmbitoOperacionAnalisis(
	canon *canonOperacionAnalisis,
	claveIdempotencia string,
	organizacionRef string,
	expedienteRef string,
	actorRef string,
	perfilRef string,
) {
	canon.texto(esquemaAmbitoOperacionAnalisis)
	canon.texto(claveIdempotencia)
	canon.texto(organizacionRef)
	canon.texto(expedienteRef)
	canon.texto(actorRef)
	canon.texto(perfilRef)
}

func escribirDatosFuncionalesCanonicos(
	canon *canonOperacionAnalisis,
	datos DatosFuncionalesOperacionAnalisis,
) {
	canon.texto(string(datos.ModalidadClave))
	canon.texto(datos.CategoriaRef)
	canon.texto(datos.GrupoSubgrupo)
	canon.texto(string(datos.CausaClave))
	canon.instante(datos.Periodo.Inicio)
	canon.instante(datos.Periodo.Fin)
	canon.enteroSinSigno(uint64(datos.PorcentajeJornada))
	canon.texto(datos.EntradaRC.Referencia)
	canon.texto(datos.EntradaRC.HuellaSHA256)
}

func escribirArtefactoCanonico(
	canon *canonOperacionAnalisis,
	datos DatosArtefactoAnalisis,
) {
	escribirContenidoArtefactoAnalisisO3(canon, datos, true)
}

type canonOperacionAnalisis struct {
	buffer bytes.Buffer
	err    error
}

func nuevoCanonOperacionAnalisis() *canonOperacionAnalisis {
	return &canonOperacionAnalisis{}
}

func (c *canonOperacionAnalisis) texto(valor string) {
	if c.err != nil || len(valor) > maximoBytesCanonOperacionAnalisis {
		c.err = ErrOperacionAnalisisInvalida
		return
	}
	var longitud [4]byte
	binary.BigEndian.PutUint32(longitud[:], uint32(len(valor)))
	_, _ = c.buffer.Write(longitud[:])
	_, _ = c.buffer.WriteString(valor)
}

func (c *canonOperacionAnalisis) enteroSinSigno(valor uint64) {
	if c.err != nil ||
		valor > MaximoEnteroSeguroOperacionAnalisis {
		c.err = ErrOperacionAnalisisInvalida
		return
	}
	var contenido [8]byte
	binary.BigEndian.PutUint64(contenido[:], valor)
	_, _ = c.buffer.Write(contenido[:])
}

func (c *canonOperacionAnalisis) enteroConSigno(valor int64) {
	if c.err != nil || !enteroCanonicoOperacionAnalisisValido(valor) {
		c.err = ErrOperacionAnalisisInvalida
		return
	}
	var contenido [8]byte
	binary.BigEndian.PutUint64(contenido[:], uint64(valor))
	_, _ = c.buffer.Write(contenido[:])
}

func (c *canonOperacionAnalisis) booleano(valor bool) {
	if valor {
		_ = c.buffer.WriteByte(1)
		return
	}
	_ = c.buffer.WriteByte(0)
}

func (c *canonOperacionAnalisis) instante(valor time.Time) {
	if !instanteSeguroOperacionAnalisis(valor) {
		c.err = ErrOperacionAnalisisInvalida
		return
	}
	c.enteroConSigno(valor.UnixMicro())
}

func (c *canonOperacionAnalisis) resultado() ([]byte, error) {
	if c.err != nil || c.buffer.Len() == 0 ||
		c.buffer.Len() > maximoBytesCanonOperacionAnalisis {
		return nil, ErrOperacionAnalisisInvalida
	}
	return append([]byte(nil), c.buffer.Bytes()...), nil
}

func preimagenOperacionAnalisisValida(contenido []byte) bool {
	return len(contenido) > 0 &&
		len(contenido) <= maximoBytesCanonOperacionAnalisis
}
