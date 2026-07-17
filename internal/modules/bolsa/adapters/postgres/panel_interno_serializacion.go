package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	esquemaOperacionPanelInternoPostgreSQLV1 = "vec.bolsa.panel.interno.consulta-postgresql.v1"
	maximoBytesRespuestaPanelInterno         = 2 * 1024 * 1024
	maximaProfundidadJSONPanelInterno        = 24
)

type operacionPanelInternoPostgreSQL struct {
	Esquema          string `json:"esquema"`
	ClaseAmbito      string `json:"clase_ambito"`
	OrganizacionRef  string `json:"organizacion_ref"`
	UnidadGestionRef string `json:"unidad_gestion_ref,omitempty"`
	Accion           string `json:"accion"`
	RecursoRef       string `json:"recurso_ref"`
	ConsultadaEn     string `json:"consultada_en"`
}

type pruebaPanelInternoPostgreSQL struct {
	EsquemaHuella        string `json:"esquema_huella"`
	DecisionRef          string `json:"decision_ref"`
	HuellaDecisionSHA256 string `json:"huella_decision_sha256"`
	VerificadaEn         string `json:"verificada_en"`
}

func serializarConsultaPanelInternoPostgreSQL(
	solicitud puertosbolsa.SolicitudConsultaPanelInterno,
) ([]byte, []byte, []byte, []byte, string, error) {
	selector, errSelector := solicitud.Selector()
	autorizacion, errAutorizacion := solicitud.Autorizacion()
	datos, errDatos := autorizacion.Datos()
	motivo, errMotivo := solicitud.Motivo()
	correlacion, errCorrelacion := solicitud.Correlacion()
	correlacionRef, errCorrelacionRef := correlacion.ValorCanonico()
	consultadaEn, errInstante := solicitud.ConsultadaEn()
	if errSelector != nil || errAutorizacion != nil || errDatos != nil || errMotivo != nil ||
		errCorrelacion != nil || errCorrelacionRef != nil || errInstante != nil {
		return nil, nil, nil, nil, "", puertosbolsa.ErrConsultaPanelInternoInvalida
	}
	decisionCanonica, err := datos.RepresentacionCanonica()
	if err != nil {
		return nil, nil, nil, nil, "", puertosbolsa.ErrConsultaPanelInternoInvalida
	}
	motivoCanonico, err := dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(motivo)
	if err != nil {
		borrarBytesPostgreSQL(decisionCanonica)
		return nil, nil, nil, nil, "", puertosbolsa.ErrConsultaPanelInternoInvalida
	}
	sumaMotivo := sha256.Sum256(motivoCanonico)
	recurso, err := puertosbolsa.RecursoAutorizablePanelInterno(selector, motivo)
	if err != nil || hex.EncodeToString(sumaMotivo[:]) != datos.Decision.MotivoHuellaSHA256 ||
		datos.Decision.CorrelacionRef != correlacionRef {
		borrarBytesPostgreSQL(decisionCanonica, motivoCanonico)
		return nil, nil, nil, nil, "", puertosbolsa.ErrConsultaPanelInternoInvalida
	}
	operacion, err := json.Marshal(operacionPanelInternoPostgreSQL{
		Esquema:     esquemaOperacionPanelInternoPostgreSQLV1,
		ClaseAmbito: string(selector.Clase), OrganizacionRef: selector.OrganizacionRef,
		UnidadGestionRef: selector.UnidadGestionRef,
		Accion:           puertosbolsa.AccionConsultarPanelInterno, RecursoRef: recurso.Referencia,
		ConsultadaEn: consultadaEn.UTC().Format(formatoInstanteMicrosegundo),
	})
	if err != nil {
		borrarBytesPostgreSQL(decisionCanonica, motivoCanonico)
		return nil, nil, nil, nil, "", puertosbolsa.ErrConsultaPanelInternoInvalida
	}
	prueba, err := json.Marshal(pruebaPanelInternoPostgreSQL{
		EsquemaHuella: datos.EsquemaHuella, DecisionRef: datos.Decision.DecisionRef,
		HuellaDecisionSHA256: datos.HuellaDecisionSHA256,
		VerificadaEn:         datos.VerificadaEn.UTC().Format(formatoInstanteMicrosegundo),
	})
	if err != nil {
		borrarBytesPostgreSQL(operacion, decisionCanonica, motivoCanonico)
		return nil, nil, nil, nil, "", puertosbolsa.ErrConsultaPanelInternoInvalida
	}
	return operacion, prueba, decisionCanonica, motivoCanonico, correlacionRef, nil
}

func decodificarPanelInternoPostgreSQL(
	contenido []byte,
	solicitud puertosbolsa.SolicitudConsultaPanelInterno,
) (puertosbolsa.InstantaneaPanelInterno, error) {
	if len(contenido) == 0 || len(contenido) > maximoBytesRespuestaPanelInterno ||
		!utf8.Valid(contenido) || validarJSONPanelInternoNoAmbiguo(contenido) != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, puertosbolsa.ErrResultadoPanelInternoInvalido
	}
	var resultado puertosbolsa.InstantaneaPanelInterno
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&resultado); err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, puertosbolsa.ErrResultadoPanelInternoInvalido
	}
	var resto any
	if err := decodificador.Decode(&resto); !errors.Is(err, io.EOF) {
		return puertosbolsa.InstantaneaPanelInterno{}, puertosbolsa.ErrResultadoPanelInternoInvalido
	}
	resultado, err := resultado.ClonarValidadaPara(solicitud)
	if err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, puertosbolsa.ErrResultadoPanelInternoInvalido
	}
	return resultado, nil
}

func validarJSONPanelInternoNoAmbiguo(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.UseNumber()
	if err := consumirValorJSONPanel(decodificador, 0); err != nil {
		return err
	}
	if _, err := decodificador.Token(); !errors.Is(err, io.EOF) {
		return puertosbolsa.ErrResultadoPanelInternoInvalido
	}
	return nil
}

func consumirValorJSONPanel(decodificador *json.Decoder, profundidad int) error {
	if profundidad > maximaProfundidadJSONPanelInterno {
		return puertosbolsa.ErrResultadoPanelInternoInvalido
	}
	token, err := decodificador.Token()
	if err != nil {
		return puertosbolsa.ErrResultadoPanelInternoInvalido
	}
	delimitador, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	switch delimitador {
	case '{':
		claves := make(map[string]struct{})
		for decodificador.More() {
			tokenClave, err := decodificador.Token()
			clave, esCadena := tokenClave.(string)
			if err != nil || !esCadena {
				return puertosbolsa.ErrResultadoPanelInternoInvalido
			}
			if _, duplicada := claves[clave]; duplicada {
				return puertosbolsa.ErrResultadoPanelInternoInvalido
			}
			claves[clave] = struct{}{}
			if err := consumirValorJSONPanel(decodificador, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim('}') {
			return puertosbolsa.ErrResultadoPanelInternoInvalido
		}
	case '[':
		for decodificador.More() {
			if err := consumirValorJSONPanel(decodificador, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim(']') {
			return puertosbolsa.ErrResultadoPanelInternoInvalido
		}
	default:
		return puertosbolsa.ErrResultadoPanelInternoInvalido
	}
	return nil
}
