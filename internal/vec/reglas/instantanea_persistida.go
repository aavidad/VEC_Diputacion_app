package reglas

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
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

// CacheInstantaneasPersistidas comparte las definiciones inmutables durante una
// consulta. Las claves son las dos huellas; los bytes se comparan también en
// cada acierto para que una captura alterada no herede una validación ajena.
type CacheInstantaneasPersistidas struct {
	mu       sync.Mutex
	entradas map[parHuellasInstantanea]catalogosInstantanea
}

type parHuellasInstantanea struct{ base, ajustes string }
type catalogosInstantanea struct {
	base                    domain.CatalogoConfigurable
	ajustes                 map[string]map[string]string
	reglasBase              []Regla
	bytesBase, bytesAjustes []byte
}

const maximoParesInstantanea = 16

func NuevaCacheInstantaneasPersistidas() *CacheInstantaneasPersistidas {
	return &CacheInstantaneasPersistidas{entradas: make(map[parHuellasInstantanea]catalogosInstantanea)}
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
	return NuevaCacheInstantaneasPersistidas().Rehidratar(p)
}

// Rehidratar valida los metadatos de cada tramo, aunque su par de catálogos
// ya esté preparado. No conserva una regla por fase: la fase y la fecha de
// captura se resuelven de nuevo para cada tramo.
func (c *CacheInstantaneasPersistidas) Rehidratar(p InstantaneaPersistidaRegla) (InstantaneaRegla, error) {
	var vacia InstantaneaRegla
	if c == nil || !claveCanonica(p.CatalogoBaseID) ||
		(p.ReglaClave != "" && !claveCanonica(p.ReglaClave)) ||
		!claveCanonica(p.Fase) || p.FaseDesde.IsZero() || p.PreparadaEn.IsZero() ||
		p.CatalogoAjustesID != CatalogoAjustesDe(p.CatalogoBaseID) || !claveCanonica(p.CatalogoAjustesID) ||
		len(p.CatalogoBaseCanonico) == 0 || len(p.CatalogoBaseCanonico) > 1<<20 ||
		len(p.CanonicoAjustes) == 0 || len(p.CanonicoAjustes) > maximoBytesAjustes {
		return vacia, ErrReglasNoDisponibles
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	clave := parHuellasInstantanea{p.CatalogoBaseHuella, p.HuellaAjustes}
	contenidos, existe := c.entradas[clave]
	if existe {
		if !bytes.Equal(contenidos.bytesBase, p.CatalogoBaseCanonico) {
			return vacia, ErrReglasNoDisponibles
		}
		if !bytes.Equal(contenidos.bytesAjustes, p.CanonicoAjustes) {
			return vacia, ErrAjustesNoDisponibles
		}
	} else {
		var err error
		contenidos, err = decodificarCatalogosInstantanea(p)
		if err != nil {
			return vacia, err
		}
		// El par de definiciones ya está íntegro. Una fase sin regla o un
		// metadato de tramo inválido no obligan a decodificarlo de nuevo.
		if len(c.entradas) < maximoParesInstantanea {
			if c.entradas == nil {
				c.entradas = make(map[parHuellasInstantanea]catalogosInstantanea)
			}
			c.entradas[clave] = contenidos
		}
	}
	return rehidratarConCatalogos(p, contenidos)
}

func decodificarCatalogosInstantanea(p InstantaneaPersistidaRegla) (catalogosInstantanea, error) {
	var contenidos catalogosInstantanea
	var base domain.CatalogoConfigurable
	if err := json.Unmarshal(p.CatalogoBaseCanonico, &base); err != nil {
		return contenidos, ErrReglasNoDisponibles
	}
	canonicoBase, huellaBase, err := CanonicoCatalogoBaseReglas(base)
	if err != nil || !bytes.Equal(canonicoBase, p.CatalogoBaseCanonico) || huellaBase != p.CatalogoBaseHuella {
		return contenidos, ErrReglasNoDisponibles
	}
	var ajustes map[string]map[string]string
	if err := json.Unmarshal(p.CanonicoAjustes, &ajustes); err != nil || ajustes == nil {
		return contenidos, ErrAjustesNoDisponibles
	}
	canonicoAjustes, err := CanonicoAjustes(ajustes)
	if err != nil || !bytes.Equal(canonicoAjustes, p.CanonicoAjustes) {
		return contenidos, ErrAjustesNoDisponibles
	}
	huellaAjustes, err := HuellaAjustes(ajustes)
	if err != nil || huellaAjustes != p.HuellaAjustes {
		return contenidos, ErrAjustesNoDisponibles
	}
	reglasBase := make([]Regla, len(base.Entradas))
	for i, entrada := range base.Entradas {
		reglasBase[i], err = reglaDesdeEntrada(base, huellaBase, base.FuenteRef == MarcaPaqueteEjemplo, entrada)
		if err != nil {
			return contenidos, ErrReglaInvalida
		}
	}
	return catalogosInstantanea{base: base, ajustes: ajustes, reglasBase: reglasBase,
		bytesBase: bytes.Clone(p.CatalogoBaseCanonico), bytesAjustes: bytes.Clone(p.CanonicoAjustes)}, nil
}

func rehidratarConCatalogos(p InstantaneaPersistidaRegla, contenidos catalogosInstantanea) (InstantaneaRegla, error) {
	var vacia InstantaneaRegla
	base, ajustes := contenidos.base, contenidos.ajustes
	if base.ID != p.CatalogoBaseID || base.Version != p.CatalogoBaseVersion ||
		!catalogoVigenteEn(base, p.PreparadaEn.UTC()) {
		return vacia, ErrReglasNoDisponibles
	}
	version := VersionAjustes{CatalogoID: p.CatalogoAjustesID, Version: p.VersionAjustes,
		HuellaSHA256: p.HuellaAjustes, VigenteDesde: p.AjustesVigenteDesde, Ajustes: ajustes}
	if p.AjustesEncontrados {
		if p.VersionAjustes < 1 || p.AjustesVigenteDesde.IsZero() || p.AjustesVigenteDesde.After(p.PreparadaEn.UTC()) {
			return vacia, ErrAjustesNoDisponibles
		}
	} else if p.VersionAjustes != 0 || !p.AjustesVigenteDesde.IsZero() || string(p.CanonicoAjustes) != "{}" {
		return vacia, ErrAjustesNoDisponibles
	}
	var encontrada *domain.EntradaCatalogoConfigurable
	indiceEncontrada := -1
	for indice := range base.Entradas {
		entrada := &base.Entradas[indice]
		if !entrada.VigenteEn(p.PreparadaEn.UTC()) || !contieneFaseRegla(entrada.Atributos["fases"], p.Fase) {
			continue
		}
		if encontrada != nil {
			return vacia, ErrReglaInvalida
		}
		encontrada = entrada
		indiceEncontrada = indice
	}
	if encontrada == nil {
		return vacia, ErrReglaNoEncontrada
	}
	entrada := *encontrada
	if p.ReglaClave != "" && p.ReglaClave != entrada.Clave {
		return vacia, ErrReglaInvalida
	}
	// La regla tipada del par es inmutable. aplicarAjuste copia los atributos
	// antes de modificarlos; Datos y CalcularConInstantanea copian al exponer.
	reglaBase := contenidos.reglasBase[indiceEncontrada]
	efectiva := reglaBase
	if campos, existe := ajustes[entrada.Clave]; p.AjustesEncontrados && existe {
		var err error
		efectiva, err = aplicarAjuste(base, p.CatalogoBaseHuella, base.FuenteRef == MarcaPaqueteEjemplo,
			entrada, reglaBase, version, campos)
		if err != nil {
			return vacia, ErrAjusteInvalido
		}
	}
	instantanea := InstantaneaRegla{datos: DatosInstantaneaRegla{
		Base: reglaBase, Efectiva: efectiva,
		CatalogoAjustesID: p.CatalogoAjustesID, AjustesEncontrados: p.AjustesEncontrados,
		VersionAjustes: p.VersionAjustes, HuellaAjustes: p.HuellaAjustes,
		CanonicoAjustes:     append([]byte(nil), p.CanonicoAjustes...),
		AjustesVigenteDesde: p.AjustesVigenteDesde, PreparadaEn: p.PreparadaEn,
	}}
	// Las postcondiciones de valida ya se establecieron al decodificar el par
	// íntegro y al comprobar los metadatos de este tramo. Datos y
	// CalcularConInstantanea mantienen su validación al exponerlo.
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
