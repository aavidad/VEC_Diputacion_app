// Package fuentesintetica autentica exclusivamente fuentes de desarrollo.
// No altera el orden recibido ni interpreta un reglamento no modelado.
package fuentesintetica

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

const EsquemaFuenteLlamamientos = "vec.bolsa.fuente-llamamientos-sintetica.v1"
const MaximoPosicionesFuente = 128

// ReglaEstado declara el resultado del caso sintético. La firma de la fuente
// gobierna este dato; no constituye una implementación del Reglamento de Bolsas.
type ReglaEstado struct {
	EstadoClave        string                                  `json:"estado_clave"`
	EstadoVersion      uint64                                  `json:"estado_version"`
	HuellaEstadoSHA256 string                                  `json:"huella_estado_sha256"`
	Resultado          domain.ResultadoElegibilidadLlamamiento `json:"resultado"`
	Motivo             domain.MotivoEvaluacionLlamamiento      `json:"motivo"`
}

type DocumentoFuenteLlamamientos struct {
	Esquema      string                              `json:"esquema"`
	OrigenRef    string                              `json:"origen_ref"`
	Version      uint64                              `json:"version"`
	VigenteDesde time.Time                           `json:"vigente_desde"`
	VigenteHasta time.Time                           `json:"vigente_hasta"`
	Datos        ports.DatosAutoritativosLlamamiento `json:"datos"`
	Reglas       []ReglaEstado                       `json:"reglas"`
}

// FuenteLlamamientos conserva copia privada del documento autenticado y de su
// firma. Su clave de confianza procede de composición, no del documento.
type FuenteLlamamientos struct {
	documento DocumentoFuenteLlamamientos
	canonico  []byte
	firma     []byte
	reloj     ports.RelojLlamamientos
}

var _ ports.FuenteFirmadaLlamamientosDesarrollo = (*FuenteLlamamientos)(nil)

// MaterialFirmaFuenteLlamamientos separa criptográficamente esta fuente de
// autorizaciones, recibos y fuentes de otros módulos.
func MaterialFirmaFuenteLlamamientos(canonico []byte) []byte {
	return append([]byte(EsquemaFuenteLlamamientos+"\n"), canonico...)
}

func NuevaFuenteLlamamientos(canonico, firma []byte, publica ed25519.PublicKey, reloj ports.RelojLlamamientos) (*FuenteLlamamientos, error) {
	if len(canonico) == 0 || len(canonico) > 1024*1024 || len(firma) != ed25519.SignatureSize ||
		len(publica) != ed25519.PublicKeySize || reloj == nil ||
		!ed25519.Verify(publica, MaterialFirmaFuenteLlamamientos(canonico), firma) {
		return nil, ports.ErrDatosLlamamientoNoConfiables
	}
	var d DocumentoFuenteLlamamientos
	decoder := json.NewDecoder(bytes.NewReader(canonico))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&d) != nil {
		return nil, ports.ErrDatosLlamamientoNoConfiables
	}
	var resto any
	if decoder.Decode(&resto) != io.EOF {
		return nil, ports.ErrDatosLlamamientoNoConfiables
	}
	normalizado, err := json.Marshal(d)
	if err != nil || !bytes.Equal(normalizado, canonico) || validarDocumento(d) != nil {
		return nil, ports.ErrDatosLlamamientoNoConfiables
	}
	f := &FuenteLlamamientos{documento: d, canonico: bytes.Clone(canonico), firma: bytes.Clone(firma), reloj: reloj}
	if !f.vigente() {
		return nil, ports.ErrDatosLlamamientoNoConfiables
	}
	return f, nil
}

func validarDocumento(d DocumentoFuenteLlamamientos) error {
	if d.Esquema != EsquemaFuenteLlamamientos || !ports.ReferenciaOpacaLlamamientoValida(d.OrigenRef) ||
		d.Version == 0 || d.Version > 1<<53-1 || !instanteValido(d.VigenteDesde) || !instanteValido(d.VigenteHasta) ||
		!d.VigenteDesde.Before(d.VigenteHasta) || len(d.Datos.Entradas) == 0 ||
		len(d.Datos.Entradas) > MaximoPosicionesFuente || len(d.Reglas) != 5 {
		return ports.ErrDatosLlamamientoNoConfiables
	}
	if _, err := d.Datos.Clonar(); err != nil {
		return err
	}
	estados := map[string]bool{"disponible": false, "ocupado": false, "no_disponible": false, "excluido": false, "renuncia_pendiente": false}
	for _, r := range d.Reglas {
		visto, existe := estados[r.EstadoClave]
		if !existe || visto || r.EstadoVersion == 0 || !huellaValida(r.HuellaEstadoSHA256) ||
			(r.Resultado != domain.ResultadoElegible && r.Resultado != domain.ResultadoNoElegible) || r.Motivo.Validar() != nil {
			return ports.ErrDatosLlamamientoNoConfiables
		}
		estados[r.EstadoClave] = true
	}
	for i, entrada := range d.Datos.Entradas {
		if entrada.Orden != uint64(i+1) || entrada.Participacion.BolsaRef != d.Datos.Bolsa.BolsaRef {
			return ports.ErrDatosLlamamientoNoConfiables
		}
		for _, situacion := range entrada.Participacion.Situaciones {
			encontrada := false
			for _, r := range d.Reglas {
				if r.EstadoClave == situacion.EstadoClave && r.EstadoVersion == situacion.EstadoVersion &&
					r.HuellaEstadoSHA256 == situacion.HuellaEstadoSHA256 {
					encontrada = true
				}
			}
			if !encontrada {
				return ports.ErrDatosLlamamientoNoConfiables
			}
		}
	}
	return nil
}

func (f *FuenteLlamamientos) vigente() bool {
	if f == nil || f.reloj == nil {
		return false
	}
	ahora := f.reloj.Ahora()
	return instanteValido(ahora) && !ahora.Before(f.documento.VigenteDesde) && ahora.Before(f.documento.VigenteHasta)
}
func contextoValido(ctx context.Context) error {
	if ctx == nil {
		return ports.ErrDatosLlamamientoNoConfiables
	}
	return ctx.Err()
}
func instanteValido(t time.Time) bool {
	return !t.IsZero() && t.Year() >= 1 && t.Year() <= 9999 && t.Location() == time.UTC && t.Nanosecond()%1000 == 0
}
func huellaValida(h string) bool {
	b, err := hex.DecodeString(h)
	return err == nil && len(b) == sha256.Size && hex.EncodeToString(b) == h
}
func huella(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func (f *FuenteLlamamientos) CargarDatosAutoritativosLlamamiento(ctx context.Context, necesidadRef string) ([]ports.DatosAutoritativosLlamamiento, error) {
	if err := contextoValido(ctx); err != nil {
		return nil, err
	}
	if !f.vigente() || necesidadRef != f.documento.Datos.Necesidad.NecesidadRef {
		return nil, ports.ErrDatosLlamamientoNoConfiables
	}
	d, err := f.documento.Datos.Clonar()
	if err != nil {
		return nil, err
	}
	return []ports.DatosAutoritativosLlamamiento{d}, nil
}
func (f *FuenteLlamamientos) ExportarFuenteFirmada(ctx context.Context, necesidadRef string) (json.RawMessage, []byte, error) {
	if err := contextoValido(ctx); err != nil {
		return nil, nil, err
	}
	if !f.vigente() || necesidadRef != f.documento.Datos.Necesidad.NecesidadRef {
		return nil, nil, ports.ErrDatosLlamamientoNoConfiables
	}
	return bytes.Clone(f.canonico), bytes.Clone(f.firma), nil
}

func (f *FuenteLlamamientos) EvaluarParticipacion(ctx context.Context, s ports.SolicitudEvaluarParticipacionLlamamiento) (domain.EvaluacionParticipacionLlamamiento, error) {
	if err := contextoValido(ctx); err != nil {
		return domain.EvaluacionParticipacionLlamamiento{}, err
	}
	if !f.vigente() || s.Validar() != nil || s.Entrada.Orden > uint64(len(f.documento.Datos.Entradas)) ||
		s.EvaluadaEn.After(f.reloj.Ahora()) || s.InstanteReferencia.Before(f.documento.VigenteDesde) {
		return domain.EvaluacionParticipacionLlamamiento{}, ports.ErrEvaluacionMotorNoConfiable
	}
	d := f.documento.Datos
	necesidadHash, err := s.Necesidad.HuellaCanonicaSHA256()
	esperadaHash, _ := d.Necesidad.HuellaCanonicaSHA256()
	entrada := d.Entradas[s.Entrada.Orden-1]
	recibida, _ := json.Marshal(s.Entrada)
	esperada, _ := json.Marshal(entrada)
	politicaRecibida, _ := json.Marshal(s.Politica)
	politicaEsperada, _ := json.Marshal(d.Politica)
	if err != nil || necesidadHash != esperadaHash || !bytes.Equal(politicaRecibida, politicaEsperada) || !bytes.Equal(recibida, esperada) {
		return domain.EvaluacionParticipacionLlamamiento{}, ports.ErrEvaluacionMotorNoConfiable
	}
	situacion, vigente := entrada.Participacion.SituacionVigenteEn(s.InstanteReferencia)
	if !vigente {
		return domain.EvaluacionParticipacionLlamamiento{}, ports.ErrEvaluacionMotorNoConfiable
	}
	var regla ReglaEstado
	encontrada := false
	for _, r := range f.documento.Reglas {
		if r.EstadoClave == situacion.EstadoClave && r.EstadoVersion == situacion.EstadoVersion && r.HuellaEstadoSHA256 == situacion.HuellaEstadoSHA256 {
			regla = r
			encontrada = true
		}
	}
	if !encontrada {
		return domain.EvaluacionParticipacionLlamamiento{}, ports.ErrEvaluacionMotorNoConfiable
	}
	material, err := json.Marshal(s)
	if err != nil {
		return domain.EvaluacionParticipacionLlamamiento{}, ports.ErrEvaluacionMotorNoConfiable
	}
	entradaHash := huella(material)
	resultadoMaterial, _ := json.Marshal(struct {
		Entrada string
		Regla   ReglaEstado
	}{entradaHash, regla})
	resultadoHash := huella(resultadoMaterial)
	e := domain.EvaluacionParticipacionLlamamiento{
		ParticipacionRef: entrada.Participacion.ParticipacionRef, SujetoRef: entrada.Participacion.SujetoRef, Orden: entrada.Orden,
		SituacionSecuencia: situacion.Secuencia, EstadoClave: situacion.EstadoClave, EstadoVersion: situacion.EstadoVersion, HuellaEstadoSHA256: situacion.HuellaEstadoSHA256,
		NecesidadRef: s.Necesidad.NecesidadRef, VersionNecesidad: s.Necesidad.Version, HuellaNecesidadSHA256: necesidadHash,
		InstantaneaRef: s.InstantaneaRef, VersionInstantanea: s.VersionInstantanea, HuellaInstantaneaSHA256: s.HuellaInstantaneaSHA256,
		PoliticaRef: s.Politica.PoliticaRef, VersionPolitica: s.Politica.Version, HuellaPoliticaSHA256: s.Politica.HuellaSHA256,
		Resultado: regla.Resultado, Motivos: []domain.MotivoEvaluacionLlamamiento{regla.Motivo},
		EntradaEvaluacionRef: ports.ReferenciaDesdeHuellaLlamamientoDesarrollo("entrada", entradaHash), HuellaEntradaSHA256: entradaHash,
		ResultadoEvaluacionRef: ports.ReferenciaDesdeHuellaLlamamientoDesarrollo("resultado", resultadoHash), HuellaResultadoSHA256: resultadoHash, EvaluadaEn: s.EvaluadaEn,
	}
	if e.Validar() != nil {
		return domain.EvaluacionParticipacionLlamamiento{}, ports.ErrEvaluacionMotorNoConfiable
	}
	return e, nil
}
