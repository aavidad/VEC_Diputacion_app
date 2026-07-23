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
	ambito.texto(esquemaAmbitoOperacionAnalisis)
	ambito.texto(datos.ClaveIdempotencia)
	ambito.texto(artefacto.OrganizacionRef)
	ambito.texto(artefacto.ExpedienteRef)
	ambito.texto(datos.ActorRef)
	ambito.texto(datos.PerfilRef)
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
