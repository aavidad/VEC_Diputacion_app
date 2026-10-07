package postgres

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/diagnostico"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	reglasdomain "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

// El límite incluye los tres contextos de la página. SQL limita conjuntamente
// los dos diccionarios; aquí se vuelve a imponer antes de decodificarlos.
const maximoBytesContextoPlazosRRHH = 8 << 20

type vinculoInstantaneaRRHH struct {
	Estado              string     `json:"estado"`
	ExpedienteRef       string     `json:"expediente_ref"`
	VersionExpediente   uint64     `json:"version_expediente"`
	Fase                string     `json:"fase"`
	FaseDesde           time.Time  `json:"fase_desde"`
	CatalogoBaseID      string     `json:"catalogo_base_id"`
	BaseVersion         int        `json:"base_version"`
	BaseHuellaSHA256    string     `json:"base_huella_sha256"`
	CatalogoAjustesID   string     `json:"catalogo_ajustes_id"`
	AjustesEncontrados  *bool      `json:"ajustes_encontrados"`
	AjustesVersion      int        `json:"ajustes_version"`
	AjustesHuellaSHA256 string     `json:"ajustes_huella_sha256"`
	AjustesVigenteDesde *time.Time `json:"ajustes_vigente_desde"`
	CapturadaEn         time.Time  `json:"capturada_en"`
}

type definicionBaseRRHH struct {
	CatalogoBaseID   string `json:"catalogo_base_id"`
	BaseVersion      int    `json:"base_version"`
	BaseHuellaSHA256 string `json:"base_huella_sha256"`
	BaseCanonico     string `json:"base_canonico"`
}

type definicionAjustesRRHH struct {
	CatalogoAjustesID   string `json:"catalogo_ajustes_id"`
	AjustesVersion      int    `json:"ajustes_version"`
	AjustesHuellaSHA256 string `json:"ajustes_huella_sha256"`
	AjustesCanonico     string `json:"ajustes_canonico"`
}

type claveContextoPlazosRRHH struct {
	id      string
	version int
	huella  string
}

// instantaneasAlineadas rehidrata únicamente los contextos citados por esta
// página. Los tres bloques provienen de la misma lectura SQL v6 y nunca se
// sustituyen por las definiciones actualmente vigentes.
type esperadoPlazoRRHH struct {
	expedienteRef string
	version       uint64
	fase          string
	desde         time.Time
}

func (s salidaCuadroConsultaRRHH) instantaneasAlineadas(
	resumenes []ports.ResumenExpedienteRRHH, fasesDesde []time.Time,
) ([]*reglas.InstantaneaPersistidaRegla, error) {
	if len(resumenes) != len(fasesDesde) {
		return nil, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	esperados := make([]esperadoPlazoRRHH, len(resumenes))
	for i, r := range resumenes {
		esperados[i] = esperadoPlazoRRHH{r.ExpedienteRef, r.Version, string(r.FaseClave), fasesDesde[i]}
	}
	return decodificarInstantaneasPlazo(s.instantaneasRegla, s.basesRegla, s.ajustesRegla,
		esperados, s.cierre.generadaEn, true)
}

func (s salidaCuadroConsultaRRHH) instantaneasGrupos() ([]*reglas.InstantaneaPersistidaRegla, error) {
	esperados := make([]esperadoPlazoRRHH, len(s.plazoFases))
	if len(s.plazoFases) != len(s.plazoDesde) {
		return nil, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	for i := range esperados {
		esperados[i] = esperadoPlazoRRHH{fase: s.plazoFases[i], desde: s.plazoDesde[i]}
	}
	return decodificarInstantaneasPlazo(s.plazoContextos, s.plazoBases, s.plazoAjustes,
		esperados, s.cierre.generadaEn, false)
}

func decodificarInstantaneasPlazo(
	vinculosBytes, basesBytes, ajustesBytes []byte,
	esperados []esperadoPlazoRRHH, generadaEn time.Time, conExpediente bool,
) ([]*reglas.InstantaneaPersistidaRegla, error) {
	fallo := ports.ErrResultadoConsultaRRHHNoConfiable
	if len(vinculosBytes) == 0 || len(basesBytes) == 0 ||
		len(ajustesBytes) == 0 ||
		len(vinculosBytes) > maximoBytesContextoPlazosRRHH ||
		len(basesBytes)+len(ajustesBytes) > maximoBytesContextoPlazosRRHH {
		return nil, fallo
	}
	var vinculos []vinculoInstantaneaRRHH
	var bases []definicionBaseRRHH
	var ajustes []definicionAjustesRRHH
	if err := decodificarArrayPlazosRRHH(vinculosBytes, &vinculos); err != nil {
		return nil, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: fallo, Causa: err}
	}
	if err := decodificarArrayPlazosRRHH(basesBytes, &bases); err != nil {
		return nil, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: fallo, Causa: err}
	}
	if err := decodificarArrayPlazosRRHH(ajustesBytes, &ajustes); err != nil {
		return nil, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: fallo, Causa: err}
	}
	if len(vinculos) != len(esperados) ||
		len(bases) > len(esperados) || len(ajustes) > len(esperados) {
		return nil, fallo
	}
	basesPorClave := make(map[claveContextoPlazosRRHH][]byte, len(bases))
	usosBase := make(map[claveContextoPlazosRRHH]int, len(bases))
	for _, base := range bases {
		clave := claveContextoPlazosRRHH{base.CatalogoBaseID, base.BaseVersion, base.BaseHuellaSHA256}
		if clave.id == "" || clave.version < 1 || len(base.BaseCanonico) == 0 ||
			len(base.BaseCanonico) > 1<<20 || basesPorClave[clave] != nil {
			return nil, fallo
		}
		var catalogo reglasdomain.CatalogoConfigurable
		if err := json.Unmarshal([]byte(base.BaseCanonico), &catalogo); err != nil {
			return nil, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: fallo, Causa: err}
		}
		if err := reglas.ValidarCatalogoBaseReglas(catalogo); err != nil {
			return nil, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: fallo, Causa: err}
		}
		canonico, huella, err := reglas.CanonicoCatalogoBaseReglas(catalogo)
		if err != nil {
			return nil, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: fallo, Causa: err}
		}
		if !bytes.Equal(canonico, []byte(base.BaseCanonico)) ||
			catalogo.ID != clave.id || catalogo.Version != clave.version ||
			huella != clave.huella {
			return nil, fallo
		}
		basesPorClave[clave] = canonico
	}
	ajustesPorClave := make(map[claveContextoPlazosRRHH][]byte, len(ajustes))
	usosAjustes := make(map[claveContextoPlazosRRHH]int, len(ajustes))
	for _, ajuste := range ajustes {
		clave := claveContextoPlazosRRHH{ajuste.CatalogoAjustesID, ajuste.AjustesVersion, ajuste.AjustesHuellaSHA256}
		if clave.id == "" || clave.version < 0 || len(ajuste.AjustesCanonico) == 0 ||
			len(ajuste.AjustesCanonico) > 16<<10 || ajustesPorClave[clave] != nil {
			return nil, fallo
		}
		var datos map[string]map[string]string
		if err := json.Unmarshal([]byte(ajuste.AjustesCanonico), &datos); err != nil {
			return nil, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: fallo, Causa: err}
		}
		if datos == nil {
			return nil, fallo
		}
		canonico, err := reglas.CanonicoAjustes(datos)
		if err != nil {
			return nil, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: fallo, Causa: err}
		}
		huella, errHuella := reglas.HuellaAjustes(datos)
		if errHuella != nil {
			return nil, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: fallo, Causa: errHuella}
		}
		if !bytes.Equal(canonico, []byte(ajuste.AjustesCanonico)) || huella != clave.huella {
			return nil, fallo
		}
		ajustesPorClave[clave] = canonico
	}
	resultado := make([]*reglas.InstantaneaPersistidaRegla, len(esperados))
	for indice, vinculo := range vinculos {
		if vinculo.Fase != esperados[indice].fase ||
			!vinculo.FaseDesde.Equal(esperados[indice].desde) ||
			(conExpediente && (vinculo.ExpedienteRef != esperados[indice].expedienteRef ||
				vinculo.VersionExpediente != esperados[indice].version)) ||
			(!conExpediente && (vinculo.ExpedienteRef != "" || vinculo.VersionExpediente != 0)) {
			return nil, fallo
		}
		if vinculo.Estado == "legado_sin_instantanea" {
			if vinculo.CatalogoBaseID != "" || vinculo.CatalogoAjustesID != "" ||
				vinculo.BaseVersion != 0 || vinculo.AjustesVersion != 0 ||
				vinculo.BaseHuellaSHA256 != "" || vinculo.AjustesHuellaSHA256 != "" ||
				vinculo.AjustesEncontrados != nil || vinculo.AjustesVigenteDesde != nil ||
				!vinculo.CapturadaEn.IsZero() {
				return nil, fallo
			}
			continue
		}
		if vinculo.Estado != "capturada" || vinculo.AjustesEncontrados == nil ||
			vinculo.CapturadaEn.IsZero() || vinculo.CapturadaEn.After(generadaEn) ||
			vinculo.CatalogoAjustesID != reglas.CatalogoAjustesDe(vinculo.CatalogoBaseID) {
			return nil, fallo
		}
		claveBase := claveContextoPlazosRRHH{vinculo.CatalogoBaseID, vinculo.BaseVersion, vinculo.BaseHuellaSHA256}
		claveAjustes := claveContextoPlazosRRHH{vinculo.CatalogoAjustesID, vinculo.AjustesVersion, vinculo.AjustesHuellaSHA256}
		base, existeBase := basesPorClave[claveBase]
		ajuste, existeAjuste := ajustesPorClave[claveAjustes]
		if !existeBase || !existeAjuste {
			return nil, fallo
		}
		usosBase[claveBase]++
		usosAjustes[claveAjustes]++
		var desdeAjustes time.Time
		if vinculo.AjustesVigenteDesde != nil {
			desdeAjustes = vinculo.AjustesVigenteDesde.UTC()
		}
		instantanea := &reglas.InstantaneaPersistidaRegla{
			CatalogoBaseID: vinculo.CatalogoBaseID, CatalogoBaseVersion: vinculo.BaseVersion,
			CatalogoBaseHuella:   vinculo.BaseHuellaSHA256,
			CatalogoBaseCanonico: append([]byte(nil), base...),
			CatalogoAjustesID:    vinculo.CatalogoAjustesID,
			AjustesEncontrados:   *vinculo.AjustesEncontrados,
			VersionAjustes:       vinculo.AjustesVersion,
			HuellaAjustes:        vinculo.AjustesHuellaSHA256,
			CanonicoAjustes:      append([]byte(nil), ajuste...),
			AjustesVigenteDesde:  desdeAjustes,
			PreparadaEn:          vinculo.CapturadaEn.UTC(),
			Fase:                 vinculo.Fase, FaseDesde: vinculo.FaseDesde.UTC(),
		}
		if _, err := reglas.RehidratarInstantaneaRegla(*instantanea); err != nil &&
			!errors.Is(err, reglas.ErrReglaNoEncontrada) {
			return nil, fallo
		}
		resultado[indice] = instantanea
	}
	for clave := range basesPorClave {
		if usosBase[clave] == 0 {
			return nil, fallo
		}
	}
	for clave := range ajustesPorClave {
		if usosAjustes[clave] == 0 {
			return nil, fallo
		}
	}
	return resultado, nil
}

func decodificarArrayPlazosRRHH[T any](contenido []byte, destino *[]T) error {
	if len(contenido) < 2 || contenido[0] != '[' {
		return ports.ErrResultadoConsultaRRHHNoConfiable
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return err
	}
	if len(*destino) == 0 && !bytes.Equal(contenido, []byte("[]")) {
		return ports.ErrResultadoConsultaRRHHNoConfiable
	}
	var resto any
	if err := decodificador.Decode(&resto); err != io.EOF {
		return ports.ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}
