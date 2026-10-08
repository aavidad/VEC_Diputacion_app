package postgres

import (
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type capturaSQLRRHH struct {
	Estado              string     `json:"estado"`
	Fase                string     `json:"fase"`
	FaseDesde           time.Time  `json:"fase_desde"`
	BaseID              string     `json:"base_id"`
	BaseVersion         int        `json:"base_version"`
	BaseHuella          string     `json:"base_huella"`
	AjustesID           string     `json:"ajustes_id"`
	AjustesEncontrados  bool       `json:"ajustes_encontrados"`
	AjustesVersion      int        `json:"ajustes_version"`
	AjustesHuella       string     `json:"ajustes_huella"`
	AjustesVigenteDesde *time.Time `json:"ajustes_vigente_desde"`
	CapturadaEn         *time.Time `json:"capturada_en"`
	Numero              uint64     `json:"numero"`
	Urgente             bool       `json:"urgente"`
}

type capturasSQLRRHH struct {
	Filas   []capturaSQLRRHH  `json:"filas"`
	Grupos  []capturaSQLRRHH  `json:"grupos"`
	Bases   map[string]string `json:"bases"`
	Ajustes map[string]string `json:"ajustes"`
	// Cada definición canónica se convierte una vez por payload. Las capturas
	// comparten estos bytes inmutables; el resolutor comprueba huella y bytes.
	basesCanonicas   map[string][]byte
	ajustesCanonicos map[string][]byte
}

func leerCapturasSQLRRHH(raw []byte) (capturasSQLRRHH, error) {
	var c capturasSQLRRHH
	if len(raw) == 0 || len(raw) > 4194304 || json.Unmarshal(raw, &c) != nil || c.Bases == nil || c.Ajustes == nil {
		return c, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	c.basesCanonicas = make(map[string][]byte, len(c.Bases))
	for huella, base := range c.Bases {
		c.basesCanonicas[huella] = []byte(base)
	}
	c.ajustesCanonicos = make(map[string][]byte, len(c.Ajustes))
	for huella, ajustes := range c.Ajustes {
		c.ajustesCanonicos[huella] = []byte(ajustes)
	}
	return c, nil
}

func (c capturasSQLRRHH) convertir(s capturaSQLRRHH) (ports.CapturaPlazoFaseRRHH, error) {
	p := ports.CapturaPlazoFaseRRHH{Estado: s.Estado, Fase: domain.ClaveFase(s.Fase), FaseDesde: s.FaseDesde.UTC()}
	if !p.Fase.Valida() || p.FaseDesde.IsZero() {
		return p, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	switch s.Estado {
	case "legado_sin_instantanea":
		return p, nil
	case "capturada", "legado_base_transicion":
		base, existeBase := c.basesCanonicas[s.BaseHuella]
		ajustes, existenAjustes := c.ajustesCanonicos[s.AjustesHuella]
		if !existeBase || !existenAjustes || s.CapturadaEn == nil || s.BaseID == "" || s.AjustesID == "" {
			return p, ports.ErrResultadoConsultaRRHHNoConfiable
		}
		p.BaseID, p.BaseVersion, p.BaseHuella, p.BaseCanonico = s.BaseID, s.BaseVersion, s.BaseHuella, base
		p.AjustesID, p.AjustesEncontrados, p.AjustesVersion, p.AjustesHuella, p.AjustesCanonico = s.AjustesID, s.AjustesEncontrados, s.AjustesVersion, s.AjustesHuella, ajustes
		p.CapturadaEn = s.CapturadaEn.UTC()
		if s.AjustesVigenteDesde != nil {
			p.AjustesVigenteDesde = s.AjustesVigenteDesde.UTC()
		}
		return p, nil
	default:
		return p, ports.ErrResultadoConsultaRRHHNoConfiable
	}
}

func (s salidaCuadroConsultaRRHH) capturasPagina(resumenes []ports.ResumenExpedienteRRHH, desde []time.Time) ([]ports.CapturaPlazoFaseRRHH, error) {
	c, err := leerCapturasSQLRRHH(s.capturasPlazo)
	if err != nil || len(c.Filas) != len(resumenes) || len(desde) != len(resumenes) {
		return nil, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	salida := make([]ports.CapturaPlazoFaseRRHH, len(c.Filas))
	for i, f := range c.Filas {
		p, err := c.convertir(f)
		if err != nil || p.Fase != resumenes[i].FaseClave || !p.FaseDesde.Equal(desde[i]) {
			return nil, ports.ErrResultadoConsultaRRHHNoConfiable
		}
		salida[i] = p
	}
	return salida, nil
}

func (s salidaCuadroConsultaRRHH) capturasResumen(agregados *ports.AgregadosCuadroRRHH) error {
	c, err := leerCapturasSQLRRHH(s.capturasGrupos)
	if err != nil || agregados == nil || len(c.Grupos) != len(agregados.GruposPlazo) {
		return ports.ErrResultadoConsultaRRHHNoConfiable
	}
	for i, f := range c.Grupos {
		grupo := &agregados.GruposPlazo[i]
		p, err := c.convertir(f)
		if err != nil || p.Fase != grupo.FaseClave || !p.FaseDesde.Equal(grupo.Desde) || f.Numero != grupo.Numero || f.Urgente != grupo.Urgente {
			return ports.ErrResultadoConsultaRRHHNoConfiable
		}
		grupo.Captura = &p
	}
	return nil
}
