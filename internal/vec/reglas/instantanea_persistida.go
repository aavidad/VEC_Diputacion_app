package reglas

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// InstantaneaPersistidaRegla conserva la definición exacta que gobernó el
// inicio de un tramo. La ausencia de una versión de ajustes también es un dato
// explícito: nunca se completa con la cabeza actual al recuperar el tramo.
// El almacén debe escribirla en la misma transacción que la entrada en fase.
type InstantaneaPersistidaRegla struct {
	CatalogoBaseID       string
	CatalogoBaseVersion  int
	CatalogoBaseHuella   string
	CatalogoBaseCanonico []byte
	CatalogoAjustesID    string
	AjustesEncontrados   bool
	VersionAjustes       int
	HuellaAjustes        string
	CanonicoAjustes      []byte
	AjustesVigenteDesde  time.Time
	PreparadaEn          time.Time
	ReglaClave           string
	Fase                 string
	FaseDesde            time.Time
}

// CanonicoCatalogoBaseReglas produce los mismos bytes que
// CatalogoConfigurable.HuellaSHA256: ClonarCanonico y json.Marshal. Una copia
// textual del archivo de origen no tiene necesariamente esa huella.
func CanonicoCatalogoBaseReglas(catalogo domain.CatalogoConfigurable) ([]byte, string, error) {
	canonico, err := catalogo.ClonarCanonico()
	if err != nil {
		return nil, "", ErrReglasNoDisponibles
	}
	contenido, err := json.Marshal(canonico)
	if err != nil {
		return nil, "", ErrReglasNoDisponibles
	}
	huella, err := canonico.HuellaSHA256()
	if err != nil {
		return nil, "", ErrReglasNoDisponibles
	}
	return contenido, huella, nil
}

// ValidarCatalogoBaseReglas comprueba todas las entradas con el mismo parser
// tipado que usa el resolutor, incluso las que empiezan a regir en el futuro o
// ya dejaron de regir. Sirve para revisar un artefacto antes de publicarlo;
// no cambia su canónico ni su huella.
func ValidarCatalogoBaseReglas(catalogo domain.CatalogoConfigurable) error {
	base, err := catalogo.ClonarCanonico()
	if err != nil {
		return ErrReglasNoDisponibles
	}
	huella, err := base.HuellaSHA256()
	if err != nil {
		return ErrReglasNoDisponibles
	}
	for _, entrada := range base.Entradas {
		if _, err := reglaDesdeEntrada(base, huella, base.FuenteRef == MarcaPaqueteEjemplo, entrada); err != nil {
			return ErrReglaInvalida
		}
	}
	return nil
}

// RehidratarInstantaneaRegla valida la definición y reconstruye la regla con
// el mismo motor del resolutor. No consulta el catálogo ni los ajustes actuales.
func RehidratarInstantaneaRegla(p InstantaneaPersistidaRegla) (InstantaneaRegla, error) {
	var vacia InstantaneaRegla
	if !claveCanonica(p.CatalogoBaseID) ||
		(p.ReglaClave != "" && !claveCanonica(p.ReglaClave)) ||
		!claveCanonica(p.Fase) || p.FaseDesde.IsZero() || p.PreparadaEn.IsZero() ||
		p.CatalogoAjustesID != CatalogoAjustesDe(p.CatalogoBaseID) ||
		len(p.CatalogoBaseCanonico) == 0 || len(p.CatalogoBaseCanonico) > 1<<20 ||
		len(p.CanonicoAjustes) == 0 || len(p.CanonicoAjustes) > maximoBytesAjustes {
		return vacia, ErrReglasNoDisponibles
	}
	var base domain.CatalogoConfigurable
	if err := json.Unmarshal(p.CatalogoBaseCanonico, &base); err != nil {
		return vacia, ErrReglasNoDisponibles
	}
	canonicoBase, huellaBase, err := CanonicoCatalogoBaseReglas(base)
	if err != nil || !bytes.Equal(canonicoBase, p.CatalogoBaseCanonico) ||
		base.ID != p.CatalogoBaseID || base.Version != p.CatalogoBaseVersion ||
		huellaBase != p.CatalogoBaseHuella || !catalogoVigenteEn(base, p.PreparadaEn.UTC()) {
		return vacia, ErrReglasNoDisponibles
	}
	var ajustes map[string]map[string]string
	if err := json.Unmarshal(p.CanonicoAjustes, &ajustes); err != nil || ajustes == nil {
		return vacia, ErrAjustesNoDisponibles
	}
	canonicoAjustes, err := CanonicoAjustes(ajustes)
	if err != nil || !bytes.Equal(canonicoAjustes, p.CanonicoAjustes) {
		return vacia, ErrAjustesNoDisponibles
	}
	huellaAjustes, err := HuellaAjustes(ajustes)
	if err != nil || huellaAjustes != p.HuellaAjustes {
		return vacia, ErrAjustesNoDisponibles
	}
	version := VersionAjustes{CatalogoID: p.CatalogoAjustesID, Version: p.VersionAjustes,
		HuellaSHA256: p.HuellaAjustes, VigenteDesde: p.AjustesVigenteDesde, Ajustes: ajustes}
	if p.AjustesEncontrados {
		if err := validarVersionAjustes(version, p.CatalogoAjustesID, p.PreparadaEn.UTC()); err != nil {
			return vacia, err
		}
	} else if p.VersionAjustes != 0 || !p.AjustesVigenteDesde.IsZero() || string(p.CanonicoAjustes) != "{}" {
		return vacia, ErrAjustesNoDisponibles
	}
	var encontrada *domain.EntradaCatalogoConfigurable
	for indice := range base.Entradas {
		entrada := &base.Entradas[indice]
		if !entrada.VigenteEn(p.PreparadaEn.UTC()) || !contieneFaseRegla(entrada.Atributos["fases"], p.Fase) {
			continue
		}
		if encontrada != nil {
			return vacia, ErrReglaInvalida
		}
		encontrada = entrada
	}
	if encontrada == nil {
		return vacia, ErrReglaNoEncontrada
	}
	entrada := *encontrada
	if p.ReglaClave != "" && p.ReglaClave != entrada.Clave {
		return vacia, ErrReglaInvalida
	}
	reglaBase, err := reglaDesdeEntrada(base, huellaBase, base.FuenteRef == MarcaPaqueteEjemplo, entrada)
	if err != nil {
		return vacia, err
	}
	efectiva := reglaBase
	if campos, existe := ajustes[entrada.Clave]; p.AjustesEncontrados && existe {
		efectiva, err = aplicarAjuste(base, huellaBase, base.FuenteRef == MarcaPaqueteEjemplo,
			entrada, reglaBase, version, campos)
		if err != nil {
			return vacia, ErrAjusteInvalido
		}
	}
	instantanea := InstantaneaRegla{datos: DatosInstantaneaRegla{
		Base: copiarRegla(reglaBase), Efectiva: copiarRegla(efectiva),
		CatalogoAjustesID: p.CatalogoAjustesID, AjustesEncontrados: p.AjustesEncontrados,
		VersionAjustes: p.VersionAjustes, HuellaAjustes: p.HuellaAjustes,
		CanonicoAjustes:     append([]byte(nil), p.CanonicoAjustes...),
		AjustesVigenteDesde: p.AjustesVigenteDesde, PreparadaEn: p.PreparadaEn,
	}}
	if !instantanea.valida() {
		return vacia, ErrReglasNoDisponibles
	}
	return instantanea, nil
}

func contieneFaseRegla(fases, fase string) bool {
	for _, candidata := range strings.Split(fases, ",") {
		if candidata == fase {
			return true
		}
	}
	return false
}

// CalcularConInstantaneaPersistida usa la definición guardada. Municipio y
// urgencia proceden del contexto operativo actual y se pasan explícitamente:
// esta instantánea no promete una fecha absoluta inmutable si cambian esos
// datos o el calendario. Nunca consulta versiones nuevas de reglas.
func (r *Resolutor) CalcularConInstantaneaPersistida(ctx context.Context, p InstantaneaPersistidaRegla, municipioSede string, urgente bool) (Regla, Vencimiento, error) {
	if municipioSede == "" || municipioSede != strings.TrimSpace(municipioSede) {
		return Regla{}, Vencimiento{}, ErrReglasNoDisponibles
	}
	instantanea, err := RehidratarInstantaneaRegla(p)
	if err != nil {
		return Regla{}, Vencimiento{}, err
	}
	return r.CalcularConInstantanea(ctx, instantanea, p.FaseDesde, municipioSede, urgente)
}
