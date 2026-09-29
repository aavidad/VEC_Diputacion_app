package domain

import (
	"errors"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// «Mi imagen» (5.08c): cómo se identifica a la persona en la cabecera de los
// dos portales. Solo admite códigos cerrados: ni colores, ni CSS, ni rutas,
// ni SVG llegan desde el cliente. La foto no vive aquí: Usuarios guarda solo
// la referencia opaca que le devuelve Documentos.

var ErrImagenInvalida = errors.New("usuarios imagen invalida")
var ErrCatalogoImagenInvalido = errors.New("usuarios catalogo de imagen invalido")

type ModoImagen string

const (
	ModoImagenIniciales ModoImagen = "iniciales"
	ModoImagenIcono     ModoImagen = "icono"
	ModoImagenFoto      ModoImagen = "foto"
)

// Vocabulario cerrado. El catálogo publicado puede ofrecer un subconjunto,
// nunca ampliarlo: cada paleta tiene su token en el tema común y cada icono
// su trazo en la interfaz.
var modosImagen = map[ModoImagen]bool{ModoImagenIniciales: true, ModoImagenIcono: true, ModoImagenFoto: true}
var paletasImagen = map[string]bool{"azul": true, "turquesa": true, "verde": true, "naranja": true, "morado": true, "gris": true}
var iconosImagen = map[string]bool{"persona": true, "estrella": true, "hoja": true, "sol": true, "corazon": true, "libro": true, "cafe": true, "montana": true}

// EleccionImagen es lo que la persona elige. Icono solo acompaña al modo
// icono; la paleta da el fondo de iniciales e icono y el de respaldo de la foto.
type EleccionImagen struct {
	Modo   ModoImagen `json:"modo"`
	Paleta string     `json:"paleta"`
	Icono  string     `json:"icono"`
}

func (e EleccionImagen) ValidarCodigos() error {
	if !modosImagen[e.Modo] || !paletasImagen[e.Paleta] {
		return ErrImagenInvalida
	}
	if e.Modo == ModoImagenIcono {
		if !iconosImagen[e.Icono] {
			return ErrImagenInvalida
		}
	} else if e.Icono != "" {
		return ErrImagenInvalida
	}
	return nil
}

type CatalogoImagen struct {
	VersionRef     string              `json:"version_ref"`
	Paletas        []OpcionPreferencia `json:"paletas"`
	Iconos         []OpcionPreferencia `json:"iconos"`
	Predeterminada EleccionImagen      `json:"predeterminada"`
}

func (c CatalogoImagen) Validar() error {
	if !versionValida(c.VersionRef) || !opcionesValidas(c.Paletas, paletasImagen) || !opcionesValidas(c.Iconos, iconosImagen) ||
		c.Predeterminada.ValidarCodigos() != nil || c.Predeterminada.Modo == ModoImagenFoto || !contiene(c.Paletas, c.Predeterminada.Paleta) ||
		(c.Predeterminada.Modo == ModoImagenIcono && !contiene(c.Iconos, c.Predeterminada.Icono)) {
		return ErrCatalogoImagenInvalido
	}
	return nil
}

// ValidarEleccion comprueba que la elección está en el catálogo publicado.
func (c CatalogoImagen) ValidarEleccion(e EleccionImagen) error {
	if c.Validar() != nil || e.ValidarCodigos() != nil || !contiene(c.Paletas, e.Paleta) ||
		(e.Modo == ModoImagenIcono && !contiene(c.Iconos, e.Icono)) {
		return ErrImagenInvalida
	}
	return nil
}

func (c CatalogoImagen) Clonar() CatalogoImagen {
	c.Paletas = append([]OpcionPreferencia(nil), c.Paletas...)
	c.Iconos = append([]OpcionPreferencia(nil), c.Iconos...)
	return c
}

// CatalogoBaseImagen es el catálogo que publica la migración Usuarios 000006.
// Las pruebas lo cotejan con el SQL para que no diverjan.
func CatalogoBaseImagen() CatalogoImagen {
	paletas := []string{"azul", "turquesa", "verde", "naranja", "morado", "gris"}
	iconos := []string{"persona", "estrella", "hoja", "sol", "corazon", "libro", "cafe", "montana"}
	c := CatalogoImagen{VersionRef: "usuarios-imagen-v1", Predeterminada: EleccionImagen{Modo: ModoImagenIniciales, Paleta: "azul"}}
	for _, p := range paletas {
		c.Paletas = append(c.Paletas, OpcionPreferencia{Codigo: p, NombreKey: "ui.usuarios.imagen.paleta." + p})
	}
	for _, i := range iconos {
		c.Iconos = append(c.Iconos, OpcionPreferencia{Codigo: i, NombreKey: "ui.usuarios.imagen.icono." + i})
	}
	return c
}

// IdentidadImagen reutiliza el vínculo persona–certificado–superficie de
// «Mis correos»: la misma frontera acredita a la persona para su imagen.
type IdentidadImagen = IdentidadCorreos

func NuevaIdentidadImagen(actor vecdomain.ContextoActor, vinculo vecdomain.VinculoAutenticacionActorV2, superficieRuta vecdomain.SuperficieAutenticacionActorV1) (IdentidadImagen, error) {
	return NuevaIdentidadCorreos(actor, vinculo, superficieRuta)
}
