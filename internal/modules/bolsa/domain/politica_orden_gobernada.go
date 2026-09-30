package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

// ErrRegimenTurnoGobernadoInvalido es un codigo interno; el canal resuelve su texto.
var ErrRegimenTurnoGobernadoInvalido = errors.New("bolsa: regimen_turno_gobernado_invalido")

type OperacionOrdenTurno string

const (
	OperacionPrelacionInicial            OperacionOrdenTurno = "prelacion_inicial"
	OperacionSucesivoDesdeTurnoConsumido OperacionOrdenTurno = "sucesivo_desde_turno_consumido"
	VersionContratoRegimenTurnoPropuesto uint64              = 2
)

// DatosPropuestaRegimenTurno son la entrada de una propuesta inmutable.
// VersionContrato distingue esta propuesta de PoliticaOrdenBolsa B18 v1; no
// habilita convertir una politica antigua ni asignarle una operacion por defecto.
// TipoLista clasifica la lista B18, sin determinar la operacion seleccionada.
// ReposicionTrasCese solo rige el retorno posterior al cese.
type DatosPropuestaRegimenTurno struct {
	VersionContrato       uint64              `json:"version_contrato"`
	PoliticaRef           string              `json:"politica_ref"`
	Version               uint64              `json:"version"`
	BolsaRef              string              `json:"bolsa_ref"`
	FuenteEvidenciaRef    string              `json:"fuente_evidencia_ref"`
	FuenteEvidenciaSHA256 string              `json:"fuente_evidencia_sha256"`
	DefinidaEn            time.Time           `json:"definida_en"`
	ActorProponenteRef    string              `json:"actor_proponente_ref"`
	TipoLista             string              `json:"tipo_lista"`
	ReposicionTrasCese    string              `json:"reposicion_tras_cese"`
	OperacionTurno        OperacionOrdenTurno `json:"operacion_turno"`
	ReglaRef              string              `json:"regla_ref"`
	ReglaVersion          uint64              `json:"regla_version"`
	ReglaHuellaSHA256     string              `json:"regla_huella_sha256"`
}

// PropuestaRegimenTurnoGobernado conserva una version y su huella sin permitir
// que un consumidor cambie las reglas despues de validarlas. Una fuente de
// evidencia no acredita aprobacion, publicacion, autorizacion ni efectos sobre
// llamamientos. El recibo corresponde a una actuacion durable posterior.
type PropuestaRegimenTurnoGobernado struct {
	datos  DatosPropuestaRegimenTurno
	huella string
}

func NuevaPropuestaRegimenTurnoGobernado(datos DatosPropuestaRegimenTurno) (PropuestaRegimenTurnoGobernado, error) {
	if !datosRegimenTurnoValidos(datos) {
		return PropuestaRegimenTurnoGobernado{}, ErrRegimenTurnoGobernadoInvalido
	}
	representacion, err := json.Marshal(datos)
	if err != nil {
		return PropuestaRegimenTurnoGobernado{}, ErrRegimenTurnoGobernadoInvalido
	}
	suma := sha256.Sum256(representacion)
	return PropuestaRegimenTurnoGobernado{datos: datos, huella: hex.EncodeToString(suma[:])}, nil
}

func datosRegimenTurnoValidos(d DatosPropuestaRegimenTurno) bool {
	if d.VersionContrato != VersionContratoRegimenTurnoPropuesto ||
		!referenciaLlamamientoOpacaValida(d.PoliticaRef) || d.Version == 0 ||
		!referenciaLlamamientoOpacaValida(d.BolsaRef) ||
		!referenciaLlamamientoOpacaValida(d.FuenteEvidenciaRef) ||
		!huellaSHA256Valida(d.FuenteEvidenciaSHA256) ||
		!instanteUTCCanonico(d.DefinidaEn) ||
		!referenciaLlamamientoOpacaValida(d.ActorProponenteRef) ||
		(d.TipoLista != TipoListaCerrada && d.TipoLista != TipoListaRotatoria) ||
		(d.ReposicionTrasCese != ReposicionMismaPosicion &&
			d.ReposicionTrasCese != ReposicionFinLista &&
			d.ReposicionTrasCese != ReposicionNoDisponibleHasta) ||
		(d.OperacionTurno != OperacionPrelacionInicial &&
			d.OperacionTurno != OperacionSucesivoDesdeTurnoConsumido) ||
		!referenciaLlamamientoOpacaValida(d.ReglaRef) || d.ReglaVersion == 0 ||
		!huellaSHA256Valida(d.ReglaHuellaSHA256) {
		return false
	}
	return true
}

func (r PropuestaRegimenTurnoGobernado) Validar() error {
	if !datosRegimenTurnoValidos(r.datos) || !huellaSHA256Valida(r.huella) {
		return ErrRegimenTurnoGobernadoInvalido
	}
	actual, err := NuevaPropuestaRegimenTurnoGobernado(r.datos)
	if err != nil || actual.huella != r.huella {
		return ErrRegimenTurnoGobernadoInvalido
	}
	return nil
}

func (r PropuestaRegimenTurnoGobernado) Datos() DatosPropuestaRegimenTurno {
	return r.datos
}

func (r PropuestaRegimenTurnoGobernado) HuellaContenidoSHA256() string { return r.huella }
