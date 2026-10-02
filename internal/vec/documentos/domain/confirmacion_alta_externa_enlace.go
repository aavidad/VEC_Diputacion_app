package domain

import "time"

// RegistroAltaExternaEnlace contiene únicamente la constancia administrativa
// de una referencia externa. No afirma que el custodio conserve los bytes.
type RegistroAltaExternaEnlace struct {
	Documento struct {
		ID          string `json:"id"`
		Version     uint64 `json:"version"`
		SHA256      string `json:"sha256"`
		CustodioID  string `json:"custodio_id"`
		CustodiaRef string `json:"custodia_ref"`
	} `json:"documento"`
	ModuloID             string    `json:"modulo_id"`
	ExpedienteRef        string    `json:"expediente_ref"`
	TipoRef              string    `json:"tipo_ref"`
	NumeroVEC            string    `json:"numero_vec"`
	CreadoEnUTC          time.Time `json:"creado_en_utc"`
	PoliticaRef          string    `json:"politica_ref"`
	PoliticaVersion      uint64    `json:"politica_version"`
	PoliticaSHA256       string    `json:"politica_sha256"`
	ConservacionHastaUTC time.Time `json:"conservacion_hasta_utc"`
	Proteccion           string    `json:"proteccion"`
	EstadoPolitica       string    `json:"estado_politica"`
}

type ConstanciaAltaExternaEnlace struct {
	Registro            RegistroAltaExternaEnlace `json:"registro"`
	AltaDecisionRef     string                    `json:"alta_decision_ref"`
	AltaAuditoriaRef    string                    `json:"alta_auditoria_ref"`
	DecisionRef         string                    `json:"decision_ref"`
	AuditoriaRef        string                    `json:"auditoria_ref"`
	ConsumoHuellaSHA256 string                    `json:"consumo_huella_sha256"`
	ConfirmadaEn        time.Time                 `json:"confirmada_en"`
}

func (c ConstanciaAltaExternaEnlace) Validar() error {
	r := c.Registro
	if !ReferenciaOpacaValida(r.Documento.ID) || r.Documento.Version == 0 ||
		!HuellaValida(r.Documento.SHA256) || !IdentificadorTecnicoValido(r.Documento.CustodioID) ||
		!ReferenciaCustodioValida(r.Documento.CustodiaRef) || r.ModuloID != "cronos" ||
		!ReferenciaOpacaValida(r.ExpedienteRef) || !ReferenciaOpacaValida(r.TipoRef) ||
		!NumeroVECValido(r.NumeroVEC) || r.CreadoEnUTC.IsZero() ||
		!ReferenciaOpacaValida(r.PoliticaRef) || r.PoliticaVersion == 0 ||
		!HuellaValida(r.PoliticaSHA256) || r.ConservacionHastaUTC.IsZero() ||
		(r.Proteccion != "conservacion" && r.Proteccion != "bloqueo") ||
		(r.EstadoPolitica != EstadoPoliticaAprobada && r.EstadoPolitica != EstadoPoliticaProvisional) ||
		(r.EstadoPolitica == EstadoPoliticaProvisional && r.Proteccion != "conservacion") ||
		c.AltaDecisionRef == "" || c.AltaAuditoriaRef == "" ||
		c.DecisionRef == "" || c.AuditoriaRef == "" ||
		c.AltaDecisionRef == c.DecisionRef || c.ConfirmadaEn.IsZero() ||
		!HuellaValida(c.ConsumoHuellaSHA256) {
		return ErrDocumentoInvalido
	}
	return nil
}
