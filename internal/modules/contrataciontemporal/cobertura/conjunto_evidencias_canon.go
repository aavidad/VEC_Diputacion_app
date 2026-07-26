package cobertura

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func calcularHuellaConjunto(
	coordenadas CoordenadasConjuntoEvidencias,
	evidencias []EvidenciaConsultaCobertura,
) (string, error) {
	escritor := nuevoEscritorConjunto()
	escritor.texto(dominioConjuntoEvidencias)
	escritor.texto(coordenadas.OrganizacionRef)
	escritor.texto(coordenadas.ExpedienteRef)
	escritor.entero64(coordenadas.VersionExpediente)
	escritor.identidadCatalogo(coordenadas.Catalogo)
	escritor.identidadPolitica(coordenadas.Politica)
	escritor.texto(string(coordenadas.FinalidadClave))
	escritor.texto(coordenadas.FinalidadRef)
	escritor.texto(string(coordenadas.ViaClave))
	escritor.texto(coordenadas.CategoriaRef)
	escritor.instante(coordenadas.Periodo.Inicio)
	escritor.instante(coordenadas.Periodo.Fin)
	escritor.entero16(uint16(len(evidencias)))
	for _, evidencia := range evidencias {
		resumen, err := evidencia.Resumen()
		if err != nil {
			return "", ports.ErrResultadoFuenteCoberturaNoConfiable
		}
		escritor.instante(evidencia.sueloTemporal())
		escritor.resumen(resumen)
	}
	material, err := escritor.resultado()
	if err != nil {
		return "", ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	huella := sha256.Sum256(material)
	return hex.EncodeToString(huella[:]), nil
}

type escritorConjunto struct {
	destino bytes.Buffer
	err     error
}

func nuevoEscritorConjunto() *escritorConjunto { return &escritorConjunto{} }
func (e *escritorConjunto) texto(valor string) {
	if e.err != nil || len(valor) > 64*1024 {
		e.err = ports.ErrResultadoFuenteCoberturaNoConfiable
		return
	}
	e.entero32(uint32(len(valor)))
	if e.err == nil {
		_, e.err = e.destino.WriteString(valor)
	}
}
func (e *escritorConjunto) entero16(valor uint16) {
	if e.err == nil {
		e.err = binary.Write(&e.destino, binary.BigEndian, valor)
	}
}
func (e *escritorConjunto) entero32(valor uint32) {
	if e.err == nil {
		e.err = binary.Write(&e.destino, binary.BigEndian, valor)
	}
}
func (e *escritorConjunto) entero64(valor uint64) {
	if e.err == nil {
		e.err = binary.Write(&e.destino, binary.BigEndian, valor)
	}
}
func (e *escritorConjunto) booleano(valor bool) {
	if valor {
		e.texto("1")
		return
	}
	e.texto("0")
}
func (e *escritorConjunto) instante(valor time.Time) {
	e.texto(valor.UTC().Format(time.RFC3339Nano))
}
func (e *escritorConjunto) identidadCatalogo(
	i domain.IdentidadCatalogoViasCobertura,
) {
	e.texto(i.Referencia)
	e.entero64(i.Version)
	e.texto(i.HuellaSHA256)
}
func (e *escritorConjunto) identidadPolitica(
	i domain.IdentidadPoliticaDecisionCobertura,
) {
	e.texto(i.Referencia)
	e.entero64(i.Version)
	e.texto(i.HuellaSHA256)
}
func (e *escritorConjunto) resumen(r ports.ResumenOrdenConsumoCobertura) {
	e.texto(r.PeticionRef)
	e.entero16(r.OrdenComprobacion)
	e.booleano(r.ComprobacionObligatoria)
	e.texto(string(r.Comprobacion.Clave))
	e.texto(string(r.Comprobacion.Resultado))
	e.texto(r.Comprobacion.FuenteRef)
	e.texto(r.Comprobacion.ReciboRef)
	e.instante(r.Comprobacion.EvaluadaEn)
	e.texto(string(r.ProcedenciaClave))
	e.texto(r.DefinicionFuenteRef)
	e.instante(r.SolicitadaEn)
	e.instante(r.EmitidaEn)
	e.instante(r.ValidaHasta)
	e.texto(r.HuellaPeticionSHA256)
	e.texto(r.HuellaResultadoSHA256)
	e.texto(r.HuellaRespuestaSHA256)
	e.texto(r.AutoridadRef)
	e.entero32(r.Generacion)
	e.texto(r.ReciboRespuestaRef)
	e.texto(r.VerificadorRef)
	e.texto(r.PublicadorCatalogoRef)
}
func (e *escritorConjunto) resultado() ([]byte, error) {
	if e.err != nil || e.destino.Len() == 0 ||
		e.destino.Len() > 2*1024*1024 {
		return nil, ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return append([]byte(nil), e.destino.Bytes()...), nil
}

func huellaValida(valor string) bool {
	if len(valor) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(valor)
	return err == nil
}
