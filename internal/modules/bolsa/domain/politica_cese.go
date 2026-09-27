package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var (
	ErrPoliticaCeseInvalida = errors.New("bolsa: politica de cese invalida")
	patronSHA256Cese        = regexp.MustCompile(`^[a-f0-9]{64}$`)
	patronClaveMapeoCese    = regexp.MustCompile(`^[a-z0-9][a-z0-9._:-]{0,79}(\|[a-z0-9][a-z0-9._:-]{0,79})?$`)
)

// PoliticaCese describe únicamente la versión efectiva que aplica Bolsa B45.
// No contiene personas, participaciones ni recibos individuales.
type PoliticaCese struct {
	Version          uint64
	CatalogoRef      string
	CatalogoSHA256   string
	Mapeo            map[string]string
	MesesGeneral     int
	MesesAcumulacion int
	Computo          string
	Estado           string
	PublicadaEn      time.Time
}

func (p PoliticaCese) Validar() error {
	if p.Version == 0 || p.CatalogoRef == "" || len(p.CatalogoRef) > 512 ||
		strings.TrimSpace(p.CatalogoRef) != p.CatalogoRef || !patronSHA256Cese.MatchString(p.CatalogoSHA256) ||
		p.MesesGeneral < 1 || p.MesesGeneral > 120 || p.MesesAcumulacion < 1 || p.MesesAcumulacion > 120 ||
		p.Computo != "fecha_cese_meses_calendario_ajuste_fin_mes" || p.Estado != "ejemplo_sintetico" ||
		p.PublicadaEn.IsZero() || len(p.Mapeo) == 0 || len(p.Mapeo) > 64 {
		return ErrPoliticaCeseInvalida
	}
	for clave, clase := range p.Mapeo {
		if !patronClaveMapeoCese.MatchString(clave) || clase != "general" && clase != "acumulacion_tareas" {
			return ErrPoliticaCeseInvalida
		}
	}
	return nil
}

func (p PoliticaCese) Clonar() PoliticaCese {
	copia := p
	copia.Mapeo = make(map[string]string, len(p.Mapeo))
	for clave, clase := range p.Mapeo {
		copia.Mapeo[clave] = clase
	}
	return copia
}
