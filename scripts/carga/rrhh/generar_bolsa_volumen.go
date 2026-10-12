// Genera cánones sintéticos de Bolsa mediante sus constructores de dominio.
// Su salida son solo dos variables psql; cargar.sh las pasa junto a 06.
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

const (
	totalParticipaciones  = 2000
	referenciaBolsa       = "bolsa:rendimiento:20261012"
	referenciaInstantanea = "instantanea:rendimiento:20261012"
	referenciaCategoria   = "categoria:rpt:analista-programador"
)

func situacion(i int) string {
	switch n := i % 100; {
	case n < 76:
		return "disponible"
	case n < 84:
		return "no_disponible"
	case n < 90:
		return "trabajando"
	case n < 94:
		return "pendiente_incorporacion"
	case n < 97:
		return "renuncia"
	case n < 99:
		return "disponible_desde"
	default:
		return "excluido"
	}
}

func ejecutar() error {
	alta := time.Date(2026, time.September, 20, 0, 0, 0, 0, time.UTC)
	corte := time.Date(2026, time.October, 12, 0, 0, 0, 0, time.UTC)
	b, err := bolsa.NuevaBolsaConstituida(bolsa.AltaBolsaConstituida{
		BolsaRef: referenciaBolsa, Version: 1,
		ProcesoRef:           "proceso:rendimiento:20261012",
		CategoriaRef:         referenciaCategoria,
		ListadoDefinitivoRef: "listado:rendimiento:20261012",
		VersionListado:       1, HuellaListadoSHA256: strings.Repeat("a", 64),
		ResolucionConstitucionRef: "resolucion:rendimiento:20261012",
		HuellaResolucionSHA256:    strings.Repeat("b", 64),
		ConstituidaEn:             alta, VigenteDesde: alta,
	})
	if err != nil {
		return fmt.Errorf("bolsa sintética inválida: %w", err)
	}
	entradas := make([]bolsa.EntradaOrdenBolsa, 0, totalParticipaciones)
	for n := 1; n <= totalParticipaciones; n++ {
		desde := alta.Add(time.Duration(n%20) * 24 * time.Hour)
		p, err := bolsa.NuevaParticipacionBolsa(bolsa.AltaParticipacionBolsa{
			ParticipacionRef: fmt.Sprintf("participacion:rendimiento:%05d", n),
			BolsaRef:         referenciaBolsa,
			SujetoRef:        fmt.Sprintf("sujeto:rendimiento:%05d", n),
			Version:          1, AltaEn: desde,
			Situaciones: []bolsa.SituacionParticipacionBolsa{{
				Secuencia: 1, EstadoClave: situacion(n),
				EstadoVersion: 1, HuellaEstadoSHA256: strings.Repeat("c", 64),
				CausaClave: "constitucion", CausaVersion: 1,
				HuellaCausaSHA256:    strings.Repeat("d", 64),
				DecisionRef:          fmt.Sprintf("decision:rendimiento:%05d", n),
				HuellaDecisionSHA256: strings.Repeat("e", 64),
				Desde:                desde,
			}},
		})
		if err != nil {
			return fmt.Errorf("participación %d inválida: %w", n, err)
		}
		entradas = append(entradas, bolsa.EntradaOrdenBolsa{
			Orden: uint64(n), Participacion: p,
		})
	}
	i, err := bolsa.NuevaInstantaneaOrdenBolsa(bolsa.AltaInstantaneaOrdenBolsa{
		InstantaneaRef: referenciaInstantanea, Version: 1, Bolsa: b,
		ReferidaEn: corte, GeneradaEn: corte, Entradas: entradas,
	})
	if err != nil {
		return fmt.Errorf("instantánea sintética inválida: %w", err)
	}
	bolsaJSON, err := json.Marshal(b)
	if err != nil {
		return err
	}
	instantaneaJSON, err := json.Marshal(i)
	if err != nil {
		return err
	}
	var vuelta bolsa.InstantaneaOrdenBolsa
	if err := json.Unmarshal(instantaneaJSON, &vuelta); err != nil {
		return err
	}
	if err := vuelta.Validar(); err != nil {
		return fmt.Errorf("canon de instantánea inválido: %w", err)
	}
	if _, err := fmt.Fprintf(os.Stdout, "\\set bolsa_canon_b64 '%s'\n",
		base64.StdEncoding.EncodeToString(bolsaJSON)); err != nil {
		return err
	}
	_, err = fmt.Fprintf(os.Stdout, "\\set instantanea_canon_b64 '%s'\n",
		base64.StdEncoding.EncodeToString(instantaneaJSON))
	return err
}

func main() {
	if err := ejecutar(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
