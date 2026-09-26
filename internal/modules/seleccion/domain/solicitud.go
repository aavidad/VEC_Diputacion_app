package domain

import (
	"encoding/json"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode"

	"vec-diputacion-granada/internal/shared/baremacion"
)

// EstadoRequisito es lo que la persona declara de cada requisito de acceso.
// «pendiente» (sin acreditar o sin dato) nunca excluye por sí solo.
type EstadoRequisito string

const (
	RequisitoCumple    EstadoRequisito = "cumple"
	RequisitoNoCumple  EstadoRequisito = "no_cumple"
	RequisitoPendiente EstadoRequisito = "pendiente"
)

// ProcedenciaDeclaradaPersona: en la fase 1 todo estado de requisito lo
// declara la persona; ninguna fuente lo ha comprobado.
const ProcedenciaDeclaradaPersona = "declarado_persona"

// EstadoSolicitud de la solicitud de participación.
type EstadoSolicitud string

const (
	EstadoBorrador   EstadoSolicitud = "borrador"
	EstadoPresentada EstadoSolicitud = "presentada"
)

func (e EstadoRequisito) valido() bool {
	return e == RequisitoCumple || e == RequisitoNoCumple || e == RequisitoPendiente
}

// Direccion postal de la persona.
type Direccion struct {
	Via          string `json:"via"`
	CodigoPostal string `json:"codigo_postal"`
	Municipio    string `json:"municipio"`
	Provincia    string `json:"provincia"`
}

// DatosPersonales de la solicitud. Solo viajan cifrados a la base; en
// listados se usan el nombre visible y el documento parcial.
type DatosPersonales struct {
	Nombre             string    `json:"nombre"`
	Apellidos          string    `json:"apellidos"`
	DocumentoIdentidad string    `json:"documento_identidad"`
	FechaNacimiento    string    `json:"fecha_nacimiento"`
	Nacionalidad       string    `json:"nacionalidad"`
	Correo             string    `json:"correo"`
	Telefono           string    `json:"telefono"`
	Direccion          Direccion `json:"direccion"`
}

var (
	documentoValido    = regexp.MustCompile(`^[A-Z0-9]{5,20}$`)
	telefonoValido     = regexp.MustCompile(`^\+?[0-9]{9,15}$`)
	codigoPostalValido = regexp.MustCompile(`^[A-Za-z0-9]{3,10}$`)
)

func normalizarTexto(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// Normalizar recorta espacios y deja el documento en mayúsculas sin
// separadores y el teléfono sin espacios.
func (d DatosPersonales) Normalizar() DatosPersonales {
	documento := strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '.' {
			return -1
		}
		return unicode.ToUpper(r)
	}, strings.TrimSpace(d.DocumentoIdentidad))
	telefono := strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' {
			return -1
		}
		return r
	}, strings.TrimSpace(d.Telefono))
	return DatosPersonales{
		Nombre: normalizarTexto(d.Nombre), Apellidos: normalizarTexto(d.Apellidos), DocumentoIdentidad: documento,
		FechaNacimiento: strings.TrimSpace(d.FechaNacimiento), Nacionalidad: normalizarTexto(d.Nacionalidad),
		Correo: strings.TrimSpace(d.Correo), Telefono: telefono,
		Direccion: Direccion{Via: normalizarTexto(d.Direccion.Via), CodigoPostal: strings.TrimSpace(d.Direccion.CodigoPostal),
			Municipio: normalizarTexto(d.Direccion.Municipio), Provincia: normalizarTexto(d.Direccion.Provincia)},
	}
}

// ValidarParcial comprueba cada dato ya aportado (normalizado): un borrador
// puede guardarse incompleto, pero lo aportado debe ser plausible. La fecha
// de nacimiento, si se da, debe ser anterior a hoy.
func (d DatosPersonales) ValidarParcial(hoy time.Time) error {
	opcional := func(v string, valido bool) bool { return v == "" || valido }
	nacimientoValido := func() bool {
		nacimiento, err := time.Parse(time.DateOnly, d.FechaNacimiento)
		return err == nil && nacimiento.Before(hoy) && nacimiento.Year() >= 1900
	}
	correoValido := func() bool {
		correo, err := mail.ParseAddress(d.Correo)
		return err == nil && correo.Address == d.Correo && len(d.Correo) <= 254
	}
	if !opcional(d.Nombre, textoValido(d.Nombre, 120)) || !opcional(d.Apellidos, textoValido(d.Apellidos, 200)) ||
		!opcional(d.DocumentoIdentidad, documentoValido.MatchString(d.DocumentoIdentidad)) ||
		!opcional(d.FechaNacimiento, d.FechaNacimiento != "" && nacimientoValido()) ||
		!opcional(d.Nacionalidad, textoValido(d.Nacionalidad, 80)) || !opcional(d.Correo, d.Correo != "" && correoValido()) ||
		!opcional(d.Telefono, telefonoValido.MatchString(d.Telefono)) || !opcional(d.Direccion.Via, textoValido(d.Direccion.Via, 300)) ||
		!opcional(d.Direccion.CodigoPostal, codigoPostalValido.MatchString(d.Direccion.CodigoPostal)) ||
		!opcional(d.Direccion.Municipio, textoValido(d.Direccion.Municipio, 120)) ||
		!opcional(d.Direccion.Provincia, textoValido(d.Direccion.Provincia, 120)) {
		return ErrSolicitudInvalida
	}
	return nil
}

// Completos dice si están todos los datos necesarios para presentar.
func (d DatosPersonales) Completos() bool {
	for _, v := range []string{d.Nombre, d.Apellidos, d.DocumentoIdentidad, d.FechaNacimiento, d.Nacionalidad, d.Correo, d.Telefono,
		d.Direccion.Via, d.Direccion.CodigoPostal, d.Direccion.Municipio, d.Direccion.Provincia} {
		if v == "" {
			return false
		}
	}
	return true
}

// DocumentoParcial oculta el documento salvo cuatro caracteres centrales
// («12345678Z» → «***5678*»), para listados.
func (d DatosPersonales) DocumentoParcial() string {
	doc := d.DocumentoIdentidad
	if doc == "" {
		return ""
	}
	if len(doc) < 6 {
		return "****"
	}
	return "***" + doc[len(doc)-5:len(doc)-1] + "*"
}

// NombreVisible es «Apellidos, Nombre».
func (d DatosPersonales) NombreVisible() string {
	return d.Apellidos + ", " + d.Nombre
}

// Canonico serializa los datos en orden estable (para cifrar y huellas).
func (d DatosPersonales) Canonico() ([]byte, error) {
	return json.Marshal(d)
}

// DatosPersonalesDesdeCanonico recupera los datos descifrados.
func DatosPersonalesDesdeCanonico(b []byte) (DatosPersonales, error) {
	var d DatosPersonales
	decodificador := json.NewDecoder(strings.NewReader(string(b)))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&d); err != nil {
		return DatosPersonales{}, ErrSolicitudInvalida
	}
	return d, nil
}

// RequisitoDeclarado por la persona.
type RequisitoDeclarado struct {
	Clave  string          `json:"clave"`
	Estado EstadoRequisito `json:"estado"`
}

// MeritoDeclarado por la persona. Cantidad es decimal en la unidad del
// mérito; la persona no fija puntos.
type MeritoDeclarado struct {
	ClaveGrupo  string `json:"clave_grupo"`
	ClaveMerito string `json:"clave_merito"`
	Descripcion string `json:"descripcion"`
	Cantidad    string `json:"cantidad"`
}

// Borrador es el contenido que la persona guarda.
type Borrador struct {
	Turno      string
	Datos      DatosPersonales
	Requisitos []RequisitoDeclarado
	Meritos    []MeritoDeclarado
}

// BorradorPreparado es el borrador normalizado con su autobaremo y si está
// completo para presentarse.
type BorradorPreparado struct {
	Borrador   Borrador
	Autobaremo Autobaremo
	Completo   bool
}

// Preparar normaliza y valida el borrador contra la convocatoria. Admite un
// borrador incompleto (se guarda desde el primer paso), pero lo aportado debe
// ser válido: turno admitido, datos plausibles, requisitos conocidos (los no
// declarados quedan «pendiente», en el orden de la convocatoria) y méritos
// del baremo. Completo exige turno y todos los datos personales.
func (b Borrador) Preparar(c Convocatoria, hoy time.Time) (BorradorPreparado, error) {
	if len(b.Requisitos) > len(c.Requisitos) || len(b.Meritos) > 200 {
		return BorradorPreparado{}, ErrSolicitudInvalida
	}
	if _, ok := c.Turno(b.Turno); b.Turno != "" && !ok {
		return BorradorPreparado{}, ErrSolicitudInvalida
	}
	datos := b.Datos.Normalizar()
	if err := datos.ValidarParcial(hoy); err != nil {
		return BorradorPreparado{}, err
	}
	declarados := make(map[string]EstadoRequisito, len(b.Requisitos))
	for _, r := range b.Requisitos {
		if _, ok := c.Requisito(r.Clave); !ok || !r.Estado.valido() {
			return BorradorPreparado{}, ErrSolicitudInvalida
		}
		if _, repetido := declarados[r.Clave]; repetido {
			return BorradorPreparado{}, ErrSolicitudInvalida
		}
		declarados[r.Clave] = r.Estado
	}
	requisitos := make([]RequisitoDeclarado, 0, len(c.Requisitos))
	for _, r := range c.Requisitos {
		estado, ok := declarados[r.Clave]
		if !ok {
			estado = RequisitoPendiente
		}
		requisitos = append(requisitos, RequisitoDeclarado{Clave: r.Clave, Estado: estado})
	}
	meritos := make([]MeritoDeclarado, 0, len(b.Meritos))
	vistos := map[string]bool{}
	for _, m := range b.Meritos {
		m.Descripcion = normalizarTexto(m.Descripcion)
		if len(m.Descripcion) > 500 || vistos[m.ClaveGrupo+"/"+m.ClaveMerito] {
			return BorradorPreparado{}, ErrSolicitudInvalida
		}
		vistos[m.ClaveGrupo+"/"+m.ClaveMerito] = true
		meritos = append(meritos, m)
	}
	autobaremo, err := c.Baremo.Calcular(meritos)
	if err != nil {
		return BorradorPreparado{}, err
	}
	return BorradorPreparado{
		Borrador:   Borrador{Turno: b.Turno, Datos: datos, Requisitos: requisitos, Meritos: meritos},
		Autobaremo: autobaremo,
		Completo:   b.Turno != "" && datos.Completos(),
	}, nil
}

// ImpidePresentar devuelve el primer requisito obligatorio que impide
// presentar declarado «no_cumple». «pendiente» no excluye.
func ImpidePresentar(c Convocatoria, requisitos []RequisitoDeclarado) (string, bool) {
	for _, d := range requisitos {
		if r, ok := c.Requisito(d.Clave); ok && r.Obligatorio && r.ImpidePresentar && d.Estado == RequisitoNoCumple {
			return r.Clave, true
		}
	}
	return "", false
}

// PuntosMerito es la puntuación orientativa de una línea declarada, antes de
// aplicar los máximos del mérito, del grupo y del baremo.
type PuntosMerito struct {
	MeritoDeclarado
	Titulo string
	Unidad string
	Puntos baremacion.Puntos
}

// Autobaremo es el resultado orientativo, calculado siempre en el servidor.
type Autobaremo struct {
	Total  baremacion.Puntos
	Lineas []PuntosMerito
	// Maximos indica que se aplicó algún máximo del mérito, grupo o baremo.
	Maximos bool
}

// Calcular puntúa cada línea (cantidad × puntos por unidad, con el redondeo
// de la convocatoria) y aplica en orden los máximos del mérito, del grupo y
// del baremo. Un mérito o grupo desconocido invalida la solicitud.
func (b Baremo) Calcular(meritos []MeritoDeclarado) (Autobaremo, error) {
	cero, _ := baremacion.PuntosDesdeMicropuntos(0)
	porMerito := map[string]baremacion.Puntos{}
	resultado := Autobaremo{Total: cero, Lineas: make([]PuntosMerito, 0, len(meritos))}
	for _, m := range meritos {
		merito, ok := b.merito(m.ClaveGrupo, m.ClaveMerito)
		if !ok {
			return Autobaremo{}, ErrSolicitudInvalida
		}
		cantidad, err := ParsearCantidad(m.Cantidad)
		if err != nil {
			return Autobaremo{}, err
		}
		puntos, err := merito.PuntosPorUnidad.MultiplicarRedondeado(cantidad, b.Redondeo)
		if err != nil {
			return Autobaremo{}, ErrSolicitudInvalida
		}
		resultado.Lineas = append(resultado.Lineas, PuntosMerito{MeritoDeclarado: m, Titulo: merito.Titulo, Unidad: merito.Unidad, Puntos: puntos})
		clave := m.ClaveGrupo + "/" + m.ClaveMerito
		acumulado, ok := porMerito[clave]
		if !ok {
			acumulado = cero
		}
		if porMerito[clave], err = acumulado.Sumar(puntos); err != nil {
			return Autobaremo{}, ErrSolicitudInvalida
		}
	}
	total := cero
	for _, g := range b.Grupos {
		grupo := cero
		for _, m := range g.Meritos {
			puntos, ok := porMerito[g.Clave+"/"+m.Clave]
			if !ok {
				continue
			}
			puntos = minimo(puntos, m.Maximo, &resultado.Maximos)
			var err error
			if grupo, err = grupo.Sumar(puntos); err != nil {
				return Autobaremo{}, ErrSolicitudInvalida
			}
		}
		var err error
		if total, err = total.Sumar(minimo(grupo, g.Maximo, &resultado.Maximos)); err != nil {
			return Autobaremo{}, ErrSolicitudInvalida
		}
	}
	resultado.Total = minimo(total, b.Maximo, &resultado.Maximos)
	return resultado, nil
}

func (b Baremo) merito(grupo, merito string) (MeritoBaremo, bool) {
	for _, g := range b.Grupos {
		if g.Clave != grupo {
			continue
		}
		for _, m := range g.Meritos {
			if m.Clave == merito {
				return m, true
			}
		}
	}
	return MeritoBaremo{}, false
}

func minimo(a, b baremacion.Puntos, recortado *bool) baremacion.Puntos {
	if a.Micropuntos() > b.Micropuntos() {
		*recortado = true
		return b
	}
	return a
}
