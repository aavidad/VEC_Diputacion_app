package catalogoefectos

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"unicode/utf8"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

var idiomaCatalogoC3 = regexp.MustCompile(`^[a-z]{2,3}(-[A-Z]{2})?$`)

type propuestaEfectosJSON struct {
	VersionEsquema int `json:"version_esquema"`
	Politica       struct {
		Referencia string                          `json:"referencia"`
		Version    int64                           `json:"version"`
		Reglas     []ports.ReglaEfectoPermisoSaldo `json:"reglas"`
	} `json:"politica"`
	Textos map[string]struct {
		Aviso string `json:"aviso"`
	} `json:"textos"`
}

// ValidarPropuestaC3 valida contenido configurable; no lo publica ni aprueba.
// El SHA corresponde a los bytes exactos, incluidos espacios y orden JSON.
func ValidarPropuestaC3(contenido []byte) (domain.PoliticaEfectosPermisoSaldo, error) {
	if !utf8.Valid(contenido) {
		return domain.PoliticaEfectosPermisoSaldo{}, domain.ErrEfectoPermisoSaldoInvalido
	}
	huella := sha256.Sum256(contenido)
	sha := hex.EncodeToString(huella[:])
	var documento propuestaEfectosJSON
	if DecodificarEstricto(contenido, sha, &documento) != nil ||
		documento.VersionEsquema != 1 || len(documento.Textos) < 1 || len(documento.Textos) > 10 {
		return domain.PoliticaEfectosPermisoSaldo{}, domain.ErrEfectoPermisoSaldoInvalido
	}
	for idioma, textos := range documento.Textos {
		if !idiomaCatalogoC3.MatchString(idioma) || len(textos.Aviso) == 0 || len(textos.Aviso) > 1024 {
			return domain.PoliticaEfectosPermisoSaldo{}, domain.ErrEfectoPermisoSaldoInvalido
		}
	}
	p := domain.PoliticaEfectosPermisoSaldo{
		Referencia: documento.Politica.Referencia, Version: documento.Politica.Version,
		SHA256: sha, Reglas: make([]domain.ReglaEfectoPermisoSaldo, len(documento.Politica.Reglas)),
	}
	if documento.Politica.Reglas == nil {
		return domain.PoliticaEfectosPermisoSaldo{}, domain.ErrEfectoPermisoSaldoInvalido
	}
	for i, regla := range documento.Politica.Reglas {
		p.Reglas[i] = domain.ReglaEfectoPermisoSaldo(regla)
	}
	if p.Validar() != nil {
		return domain.PoliticaEfectosPermisoSaldo{}, domain.ErrEfectoPermisoSaldoInvalido
	}
	return p, nil
}

type ValidadorPropuesta struct{}

func (ValidadorPropuesta) ValidarPropuestaC3(b []byte) (domain.PoliticaEfectosPermisoSaldo, error) {
	return ValidarPropuestaC3(b)
}
