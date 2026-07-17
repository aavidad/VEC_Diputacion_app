package calculoexperienciaoficial

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
)

const EsquemaSelectorFuenteExactaCalculoReglasBaremoV1 = "vec.bolsa.calculo-experiencia.selector-fuente-exacta.v1"

var ErrSelectorFuenteExactaCalculoReglasBaremoInvalido = errors.New("selector de fuente exacta para calculo de reglas de baremo invalido")

// SelectorFuenteExactaCalculoReglasBaremo liga el calculo a reglas, entrada,
// sujeto pseudonimizado y convocatoria exactos. No admite identificadores
// personales directos ni resolucion temporal de versiones.
type SelectorFuenteExactaCalculoReglasBaremo struct {
	EstadoReglas       reglas.VinculoEstadoReglasBaremo
	InstantaneaEntrada reglas.ReferenciaVersionada
	SujetoPseudonimo   reglas.ReferenciaVersionada
	Convocatoria       reglas.ReferenciaVersionada
}

type referenciaSelectorFuenteV1 struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type materialSelectorFuenteV1 struct {
	Esquema string `json:"esquema"`
	Reglas  struct {
		Contenido          referenciaSelectorFuenteV1 `json:"contenido"`
		Revision           uint64                     `json:"revision"`
		HuellaEstadoSHA256 string                     `json:"huella_estado_sha256"`
	} `json:"reglas"`
	InstantaneaEntrada referenciaSelectorFuenteV1 `json:"instantanea_entrada"`
	SujetoPseudonimo   referenciaSelectorFuenteV1 `json:"sujeto_pseudonimo"`
	Convocatoria       referenciaSelectorFuenteV1 `json:"convocatoria"`
}

// Validar comprueba que cada identidad versionada y el estado de reglas pueden
// reconstruirse exactamente. No resuelve alias ni completa valores ausentes.
func (s SelectorFuenteExactaCalculoReglasBaremo) Validar() error {
	if !referenciaSelectorValida(s.InstantaneaEntrada) ||
		!referenciaSelectorValida(s.SujetoPseudonimo) ||
		!referenciaSujetoPseudonimizadoHMACValida(s.SujetoPseudonimo.Referencia()) ||
		!referenciaSelectorValida(s.Convocatoria) ||
		!estadoReglasSelectorValido(s.EstadoReglas) || !rolesSelectorFuenteDistintos(s) {
		return ErrSelectorFuenteExactaCalculoReglasBaremoInvalido
	}
	return nil
}

func rolesSelectorFuenteDistintos(s SelectorFuenteExactaCalculoReglasBaremo) bool {
	referencias := []string{
		s.EstadoReglas.Contenido().Referencia(), s.InstantaneaEntrada.Referencia(),
		s.SujetoPseudonimo.Referencia(), s.Convocatoria.Referencia(),
	}
	vistas := make(map[string]struct{}, len(referencias))
	for _, referencia := range referencias {
		if _, existe := vistas[referencia]; existe {
			return false
		}
		vistas[referencia] = struct{}{}
	}
	return true
}

// RepresentacionCanonicaV1 es el unico material que deben compartir la capa de
// aplicacion y los adaptadores al ligar una autorizacion a este selector.
func (s SelectorFuenteExactaCalculoReglasBaremo) RepresentacionCanonicaV1() ([]byte, error) {
	if err := s.Validar(); err != nil {
		return nil, err
	}
	material := materialSelectorFuenteV1{
		Esquema:            EsquemaSelectorFuenteExactaCalculoReglasBaremoV1,
		InstantaneaEntrada: referenciaSelector(s.InstantaneaEntrada),
		SujetoPseudonimo:   referenciaSelector(s.SujetoPseudonimo),
		Convocatoria:       referenciaSelector(s.Convocatoria),
	}
	material.Reglas.Contenido = referenciaSelector(s.EstadoReglas.Contenido())
	material.Reglas.Revision = s.EstadoReglas.Revision()
	material.Reglas.HuellaEstadoSHA256 = s.EstadoReglas.HuellaEstadoSHA256()
	contenido, err := json.Marshal(material)
	if err != nil {
		return nil, errors.Join(ErrSelectorFuenteExactaCalculoReglasBaremoInvalido, err)
	}
	return contenido, nil
}

// HuellaSHA256V1 identifica el esquema y todos los campos exactos del selector.
func (s SelectorFuenteExactaCalculoReglasBaremo) HuellaSHA256V1() (string, error) {
	contenido, err := s.RepresentacionCanonicaV1()
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func referenciaSelector(referencia reglas.ReferenciaVersionada) referenciaSelectorFuenteV1 {
	return referenciaSelectorFuenteV1{
		Referencia: referencia.Referencia(), Version: referencia.Version(),
		HuellaSHA256: referencia.HuellaSHA256(),
	}
}

func referenciaSelectorValida(referencia reglas.ReferenciaVersionada) bool {
	reconstruida, err := reglas.NuevaReferenciaVersionada(
		referencia.Referencia(), referencia.Version(), referencia.HuellaSHA256(),
	)
	return err == nil && reconstruida.Referencia() == referencia.Referencia() &&
		reconstruida.Version() == referencia.Version() &&
		reconstruida.HuellaSHA256() == referencia.HuellaSHA256()
}

func estadoReglasSelectorValido(estado reglas.VinculoEstadoReglasBaremo) bool {
	reconstruido, err := reglas.NuevoVinculoEstadoReglasBaremo(
		estado.Contenido(), estado.Revision(), estado.HuellaEstadoSHA256(),
	)
	return err == nil &&
		reconstruido.Contenido().Referencia() == estado.Contenido().Referencia() &&
		reconstruido.Contenido().Version() == estado.Contenido().Version() &&
		reconstruido.Contenido().HuellaSHA256() == estado.Contenido().HuellaSHA256() &&
		reconstruido.Revision() == estado.Revision() &&
		reconstruido.HuellaEstadoSHA256() == estado.HuellaEstadoSHA256()
}
