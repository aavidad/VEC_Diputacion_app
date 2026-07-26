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

const (
	dominioPreparacionConjuntosVias = "VEC-CT-PREPARACION-CONJUNTOS-VIAS-COBERTURA-V1"
	maximoViasPreparacion           = 64
)

type parHuellaConjuntoVia struct {
	via    domain.ClaveCatalogo
	huella string
}

type identidadFuenteClave struct {
	procedencia domain.ClaveCatalogo
	definicion  string
}

type claveResultadoFuncional struct {
	clave     domain.ClaveCatalogo
	resultado domain.ResultadoComprobacion
}

type escritorConjuntosVias struct {
	destino bytes.Buffer
	err     error
}

func calcularHuellaConjuntosVias(
	analisisRef string,
	analisisHuella string,
	pares []parHuellaConjuntoVia,
) (string, error) {
	if !domain.ReferenciaOpacaValida(analisisRef) ||
		!huellaValida(analisisHuella) ||
		len(pares) == 0 || len(pares) > maximoViasPreparacion {
		return "", ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	escritor := escritorConjuntosVias{}
	escritor.texto(dominioPreparacionConjuntosVias)
	escritor.texto(analisisRef)
	escritor.texto(analisisHuella)
	escritor.entero16(uint16(len(pares)))
	for _, par := range pares {
		if !par.via.Valida() || !huellaValida(par.huella) {
			return "", ports.ErrResultadoFuenteCoberturaNoConfiable
		}
		escritor.texto(string(par.via))
		escritor.texto(par.huella)
	}
	material, err := escritor.resultado()
	if err != nil {
		return "", err
	}
	huella := sha256.Sum256(material)
	return hex.EncodeToString(huella[:]), nil
}

func proyectarConjuntosVias(
	catalogo domain.CatalogoViasCobertura,
	politica domain.PoliticaDecisionCobertura,
	coordenadas CoordenadasConjuntoEvidencias,
	conjuntos []conjuntoViaOrdenado,
	instante time.Time,
) (estadoConjuntosVias, error) {
	estado := estadoConjuntosVias{
		catalogo: catalogo, politica: politica, coordenadas: coordenadas,
		conjuntos: append([]conjuntoViaOrdenado(nil), conjuntos...),
	}
	peticiones := make(map[string]domain.ClaveCatalogo)
	huellasPeticion := make(map[string]domain.ClaveCatalogo)
	respuestas := make(map[string]domain.ClaveCatalogo)
	pruebas := make(map[identificadorPrueba]domain.ClaveCatalogo)
	fuentes := make(map[domain.ClaveCatalogo]identidadFuenteClave)
	resultadosProyectados := make(map[claveResultadoFuncional]struct{})
	for _, conjunto := range conjuntos {
		evidencias, err := conjunto.conjunto.evidenciasNormalizadasEn(instante)
		if err != nil {
			return estadoConjuntosVias{},
				ports.ErrResultadoFuenteCoberturaNoConfiable
		}
		estado.materialCanon = append(estado.materialCanon,
			parHuellaConjuntoVia{via: conjunto.via, huella: conjunto.huella},
		)
		for _, evidencia := range evidencias {
			resumen, err := evidencia.Resumen()
			fuente := identidadFuenteClave{
				procedencia: resumen.ProcedenciaClave,
				definicion:  resumen.DefinicionFuenteRef,
			}
			fuenteAnterior, claveReutilizada :=
				fuentes[resumen.Comprobacion.Clave]
			if err != nil ||
				(claveReutilizada && fuenteAnterior != fuente) ||
				!usoExclusivoEntreVias(
					peticiones, resumen.PeticionRef, conjunto.via,
				) ||
				!usoExclusivoEntreVias(
					huellasPeticion,
					resumen.HuellaPeticionSHA256,
					conjunto.via,
				) ||
				!usoExclusivoEntreVias(
					respuestas,
					resumen.HuellaRespuestaSHA256,
					conjunto.via,
				) ||
				!usoExclusivoEntreVias(
					pruebas,
					identificadorPrueba{
						resumen.AutoridadRef,
						resumen.Generacion,
						resumen.ReciboRespuestaRef,
					},
					conjunto.via,
				) {
				return estadoConjuntosVias{},
					ports.ErrResultadoFuenteCoberturaNoConfiable
			}
			fuentes[resumen.Comprobacion.Clave] = fuente
			comprobacion, err := evidencia.Comprobacion()
			if err != nil {
				return estadoConjuntosVias{},
					ports.ErrResultadoFuenteCoberturaNoConfiable
			}
			orden, err := evidencia.OrdenPendienteEn(instante)
			if err != nil {
				return estadoConjuntosVias{},
					ports.ErrResultadoFuenteCoberturaNoConfiable
			}
			estado.ordenes = append(estado.ordenes, orden)
			claveResultado := claveResultadoFuncional{
				clave:     comprobacion.Clave,
				resultado: comprobacion.Resultado,
			}
			if _, proyectado := resultadosProyectados[claveResultado]; !proyectado {
				resultadosProyectados[claveResultado] = struct{}{}
				estado.resultados = append(
					estado.resultados,
					comprobacion,
				)
			}
			estado.validaHasta = minimoHasta(
				estado.validaHasta,
				resumen.ValidaHasta,
			)
		}
	}
	estado.validaHasta = minimoVigencias(
		estado.validaHasta,
		catalogo.Vigencia(),
		politica.Vigencia(),
	)
	if len(estado.ordenes) == 0 ||
		!domain.InstanteUTCCanonico(estado.validaHasta) ||
		!instante.Before(estado.validaHasta) {
		return estadoConjuntosVias{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return estado, nil
}

func usoExclusivoEntreVias[K comparable](
	usos map[K]domain.ClaveCatalogo,
	identidad K,
	via domain.ClaveCatalogo,
) bool {
	anterior, existe := usos[identidad]
	if existe {
		return anterior == via
	}
	usos[identidad] = via
	return true
}

func minimoVigencias(
	minimo time.Time,
	catalogo domain.VigenciaCatalogoCobertura,
	politica domain.VigenciaCatalogoCobertura,
) time.Time {
	if !catalogo.Hasta.IsZero() {
		minimo = minimoHasta(minimo, catalogo.Hasta)
	}
	if !politica.Hasta.IsZero() {
		minimo = minimoHasta(minimo, politica.Hasta)
	}
	return minimo
}

func minimoHasta(actual time.Time, candidata time.Time) time.Time {
	if actual.IsZero() || candidata.Before(actual) {
		return candidata
	}
	return actual
}

func (e *escritorConjuntosVias) texto(valor string) {
	if e.err != nil || len(valor) > 64*1024 {
		e.err = ports.ErrResultadoFuenteCoberturaNoConfiable
		return
	}
	e.entero32(uint32(len(valor)))
	if e.err == nil {
		_, e.err = e.destino.WriteString(valor)
	}
}

func (e *escritorConjuntosVias) entero16(valor uint16) {
	if e.err == nil {
		e.err = binary.Write(&e.destino, binary.BigEndian, valor)
	}
}

func (e *escritorConjuntosVias) entero32(valor uint32) {
	if e.err == nil {
		e.err = binary.Write(&e.destino, binary.BigEndian, valor)
	}
}

func (e *escritorConjuntosVias) resultado() ([]byte, error) {
	if e.err != nil || e.destino.Len() == 0 ||
		e.destino.Len() > 2*1024*1024 {
		return nil, ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return append([]byte(nil), e.destino.Bytes()...), nil
}
