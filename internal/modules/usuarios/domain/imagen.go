package domain

import (
	"errors"
	"math"
)

var ErrImagenInvalida = errors.New("usuarios imagen invalida")

type ModoImagen string

const (
	ModoIniciales ModoImagen = "iniciales"
	ModoIcono     ModoImagen = "icono"
	ModoFoto      ModoImagen = "foto"
)

// Los códigos son cerrados: el cliente no puede aportar colores, CSS ni rutas.
type EleccionImagen struct {
	Modo         ModoImagen `json:"modo"`
	Paleta       string     `json:"paleta"`
	Icono        string     `json:"icono,omitempty"`
	DocumentoRef string     `json:"documento_ref,omitempty"`
}

type PaletaImagen struct {
	Codigo    string `json:"codigo"`
	Fondo     string `json:"fondo"`
	Texto     string `json:"texto"`
	NombreKey string `json:"nombre_key"`
}

type IconoImagen struct {
	Codigo    string `json:"codigo"`
	NombreKey string `json:"nombre_key"`
}

type CatalogoImagen struct {
	VersionRef string         `json:"version_ref"`
	Paletas    []PaletaImagen `json:"paletas"`
	Iconos     []IconoImagen  `json:"iconos"`
}

func CatalogoBaseImagen() CatalogoImagen {
	return CatalogoImagen{
		VersionRef: "usuarios-imagen-v1",
		Paletas: []PaletaImagen{
			{"azul", "#173b65", "#ffffff", "ui.usuarios.imagen.paleta.azul"},
			{"verde", "#24513e", "#ffffff", "ui.usuarios.imagen.paleta.verde"},
			{"granate", "#692d40", "#ffffff", "ui.usuarios.imagen.paleta.granate"},
			{"gris", "#384452", "#ffffff", "ui.usuarios.imagen.paleta.gris"},
		},
		Iconos: []IconoImagen{
			{"persona", "ui.usuarios.imagen.icono.persona"},
			{"estrella", "ui.usuarios.imagen.icono.estrella"},
			{"hoja", "ui.usuarios.imagen.icono.hoja"},
		},
	}
}

func (c CatalogoImagen) Clonar() CatalogoImagen {
	c.Paletas = append([]PaletaImagen(nil), c.Paletas...)
	c.Iconos = append([]IconoImagen(nil), c.Iconos...)
	return c
}

func (c CatalogoImagen) Validar() error {
	base := CatalogoBaseImagen()
	if !versionValida(c.VersionRef) || len(c.Paletas) == 0 || len(c.Iconos) == 0 {
		return ErrImagenInvalida
	}
	paletas := map[string]bool{}
	for _, p := range c.Paletas {
		valida := false
		for _, b := range base.Paletas {
			if p == b && ContrasteAA(p.Fondo, p.Texto) {
				valida = true
				break
			}
		}
		if !valida || paletas[p.Codigo] {
			return ErrImagenInvalida
		}
		paletas[p.Codigo] = true
	}
	iconos := map[string]bool{}
	for _, i := range c.Iconos {
		valido := false
		for _, b := range base.Iconos {
			if i == b {
				valido = true
				break
			}
		}
		if !valido || iconos[i.Codigo] {
			return ErrImagenInvalida
		}
		iconos[i.Codigo] = true
	}
	return nil
}

func (c CatalogoImagen) ValidarEleccion(e EleccionImagen) error {
	if c.Validar() != nil {
		return ErrImagenInvalida
	}
	paleta := false
	for _, p := range c.Paletas {
		if p.Codigo == e.Paleta {
			paleta = true
		}
	}
	if !paleta {
		return ErrImagenInvalida
	}
	switch e.Modo {
	case ModoIniciales:
		if e.Icono != "" || e.DocumentoRef != "" {
			return ErrImagenInvalida
		}
	case ModoIcono:
		if e.DocumentoRef != "" {
			return ErrImagenInvalida
		}
		for _, i := range c.Iconos {
			if i.Codigo == e.Icono {
				return nil
			}
		}
		return ErrImagenInvalida
	case ModoFoto:
		if e.Icono != "" || !ReferenciaDocumentoValida(e.DocumentoRef) {
			return ErrImagenInvalida
		}
	default:
		return ErrImagenInvalida
	}
	return nil
}

func ReferenciaDocumentoValida(ref string) bool {
	if len(ref) < 16 || len(ref) > 128 {
		return false
	}
	for _, r := range ref {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != ':' {
			return false
		}
	}
	return true
}

func ContrasteAA(fondo, texto string) bool {
	a, okA := luminancia(fondo)
	b, okB := luminancia(texto)
	if !okA || !okB {
		return false
	}
	return (math.Max(a, b)+0.05)/(math.Min(a, b)+0.05) >= 4.5
}

func luminancia(hex string) (float64, bool) {
	if len(hex) != 7 || hex[0] != '#' {
		return 0, false
	}
	var v [3]float64
	for i := 0; i < 3; i++ {
		hi, a := digitoHex(hex[1+i*2])
		lo, b := digitoHex(hex[2+i*2])
		if !a || !b {
			return 0, false
		}
		x := float64(hi*16+lo) / 255
		if x <= 0.04045 {
			v[i] = x / 12.92
		} else {
			v[i] = math.Pow((x+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*v[0] + 0.7152*v[1] + 0.0722*v[2], true
}
func digitoHex(c byte) (int, bool) {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0'), true
	case c >= 'a' && c <= 'f':
		return int(c - 'a' + 10), true
	case c >= 'A' && c <= 'F':
		return int(c - 'A' + 10), true
	}
	return 0, false
}
