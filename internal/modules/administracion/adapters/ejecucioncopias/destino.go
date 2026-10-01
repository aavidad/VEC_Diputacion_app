package ejecucioncopias

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"

	cs03 "vec-diputacion-granada/internal/modules/administracion/application/destinocopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	puerto "vec-diputacion-granada/internal/modules/administracion/ports/destinocopias"
	ej "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
)

var ErrConjunto = errors.New("copias_conjunto_no_autenticado")

// FuenteComponentes es el material de la ventana fría, entregado por un proveedor
// configurado por el ejecutor. No acepta rutas procedentes del manifiesto.
type FuenteComponentes interface {
	// Leer transfiere al adaptador un buffer propio y borrable.
	Leer(context.Context, string, copias.Artefacto) ([]byte, error)
}

// CatalogoIndice conserva fuera del conjunto restaurable la referencia opaca
// necesaria para abrir el índice cifrado. Su CAS impide reemplazar la versión
// verificada sin observar la versión previa.
type CatalogoIndice interface {
	Crear(context.Context, string, puerto.Referencia) error
	CAS(context.Context, string, puerto.Referencia, puerto.Referencia) error
	Leer(context.Context, string) (puerto.Referencia, error)
}

type DestinoCS03 struct {
	Destino  puerto.Destino
	Fuente   FuenteComponentes
	Catalogo CatalogoIndice
	// ProteccionEsperada procede de la misma configuración privada que construye
	// el protector CS03. CifradoSHA256 se sustituye al publicar los bytes reales.
	ProteccionEsperada copias.Proteccion
}

type indiceConjunto struct {
	Version                  int                 `json:"version"`
	Original                 copias.Manifiesto   `json:"original"`
	Final                    copias.Manifiesto   `json:"final"`
	Origen                   copias.Evidencia    `json:"origen"`
	Componentes              []puerto.Referencia `json:"componentes"`
	ManifiestoBaseSHA256     string              `json:"manifiesto_base_sha256,omitempty"`
	EjecucionVerificacionRef string              `json:"ejecucion_verificacion_ref,omitempty"`
	IndiceInicial            *puerto.Referencia  `json:"indice_inicial,omitempty"`
}

func (d DestinoCS03) disponible() bool {
	return presente(d.Destino) && presente(d.Fuente) && presente(d.Catalogo)
}

func presente(v any) bool {
	if v == nil {
		return false
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return !r.IsNil()
	default:
		return true
	}
}

func (d DestinoCS03) Publicar(ctx context.Context, c ej.Captura) (ej.Conjunto, error) {
	if !d.disponible() || ctx == nil || ctx.Err() != nil {
		return ej.Conjunto{}, ErrConjunto
	}
	m := c.Manifiesto
	if m.Verificacion.Estado != "pendiente_verificacion" || m.Proteccion.CifradoSHA256 != strings.Repeat("0", 64) || !mismaProteccion(m.Proteccion, d.ProteccionEsperada) || !evidenciaOrigenValida(c.Origen) || len(m.Componentes) == 0 || len(m.Componentes) > 4096 || len(copias.ValidarManifiesto(m)) != 0 || !cs03.ReferenciaOpaca(m.ConjuntoRef) {
		return ej.Conjunto{}, ErrConjunto
	}
	mb, err := json.Marshal(m)
	if err != nil {
		return ej.Conjunto{}, ErrConjunto
	}
	defer clear(mb)
	mh := cs03.Huella(mb)
	refs := make([]puerto.Referencia, 0, len(m.Componentes))
	for n, a := range m.Componentes {
		if a.TamanoBytes < 0 || a.TamanoBytes > 1<<30 {
			return ej.Conjunto{}, ErrConjunto
		}
		claro, err := d.Fuente.Leer(ctx, m.ConjuntoRef, a)
		if err != nil {
			return ej.Conjunto{}, ErrConjunto
		}
		if int64(len(claro)) != a.TamanoBytes || cs03.Huella(claro) != a.SHA256 {
			clear(claro)
			return ej.Conjunto{}, ErrConjunto
		}
		almacenado := claro
		if a.TamanoBytes == 0 {
			almacenado = []byte{0}
		}
		r, err := d.Destino.Publicar(ctx, puerto.Solicitud{Vinculo: puerto.Vinculo{ConjuntoRef: m.ConjuntoRef, ComponenteRef: componenteRef(a.ID), Posicion: uint64(n + 1), ManifiestoSHA256: mh}, Manifiesto: mb, Contenido: almacenado})
		if a.TamanoBytes == 0 {
			clear(almacenado)
		}
		clear(claro)
		if err != nil {
			return ej.Conjunto{}, err
		}
		refs = append(refs, r)
	}
	final := m
	final.Proteccion.CifradoSHA256 = huellaReferencias(refs)
	idx := indiceConjunto{Version: 1, Original: m, Final: final, Origen: c.Origen, Componentes: refs}
	ir, err := d.publicarIndice(ctx, idx)
	if err != nil {
		return ej.Conjunto{}, err
	}
	if err = d.Catalogo.Crear(ctx, m.ConjuntoRef, ir); err != nil {
		return ej.Conjunto{}, err
	}
	return ej.Conjunto{Ref: m.ConjuntoRef, IndiceAutenticadoRef: ir.ObjetoRef, ManifiestoSHA256: cs03.Huella(mustJSON(final)), ManifiestoBaseSHA256: cs03.Huella(mustJSON(final)), EjecucionVerificacionRef: ir.ObjetoRef, Manifiesto: final, Origen: c.Origen}, nil
}

func (d DestinoCS03) CerrarVerificacion(ctx context.Context, c ej.Conjunto, v copias.Verificacion) (ej.Conjunto, error) {
	if !d.disponible() || ctx == nil || ctx.Err() != nil || v.Estado != "valida" && v.Estado != "no_valida" {
		return ej.Conjunto{}, ErrConjunto
	}
	// Los componentes se vuelven a autenticar antes de sellar una verificación.
	if _, err := d.Recuperar(ctx, c.Ref); err != nil {
		return ej.Conjunto{}, ErrConjunto
	}
	actual, old, err := d.leerIndice(ctx, c.Ref)
	if err != nil || old.ObjetoRef != c.IndiceAutenticadoRef || actual.Final.Verificacion.Estado != "pendiente_verificacion" || c.ManifiestoSHA256 != cs03.Huella(mustJSON(actual.Final)) || c.ManifiestoBaseSHA256 != c.ManifiestoSHA256 || c.EjecucionVerificacionRef != old.ObjetoRef || !reflect.DeepEqual(actual.Final, c.Manifiesto) || !reflect.DeepEqual(actual.Origen, c.Origen) {
		return ej.Conjunto{}, ErrConjunto
	}
	final := actual.Final
	final.Verificacion = v
	if len(copias.ValidarManifiesto(final)) != 0 {
		return ej.Conjunto{}, ErrConjunto
	}
	actual.Final = final
	actual.ManifiestoBaseSHA256 = c.ManifiestoBaseSHA256
	actual.EjecucionVerificacionRef = c.EjecucionVerificacionRef
	actual.IndiceInicial = &old
	nuevo, err := d.publicarIndice(ctx, actual)
	if err != nil {
		return ej.Conjunto{}, err
	}
	if err = d.Catalogo.CAS(ctx, c.Ref, old, nuevo); err != nil {
		return ej.Conjunto{}, err
	}
	return ej.Conjunto{Ref: c.Ref, IndiceAutenticadoRef: nuevo.ObjetoRef, ManifiestoSHA256: cs03.Huella(mustJSON(final)), ManifiestoBaseSHA256: actual.ManifiestoBaseSHA256, EjecucionVerificacionRef: actual.EjecucionVerificacionRef, Manifiesto: final, Origen: actual.Origen}, nil
}

func (d DestinoCS03) Recuperar(ctx context.Context, ref string) (ej.Conjunto, error) {
	i, ir, err := d.leerIndice(ctx, ref)
	if err != nil {
		return ej.Conjunto{}, err
	}
	if len(i.Componentes) != len(i.Original.Componentes) || len(i.Componentes) == 0 || len(i.Componentes) > 4096 {
		return ej.Conjunto{}, ErrConjunto
	}
	mb, _ := json.Marshal(i.Original)
	mh := cs03.Huella(mb)
	seen := make(map[string]bool, len(i.Componentes))
	for n, r := range i.Componentes {
		a := i.Original.Componentes[n]
		if seen[a.ID] || r.Vinculo.ConjuntoRef != ref || r.Vinculo.ComponenteRef != componenteRef(a.ID) || r.Vinculo.Posicion != uint64(n+1) || r.Vinculo.ManifiestoSHA256 != mh {
			return ej.Conjunto{}, ErrConjunto
		}
		seen[a.ID] = true
		p, e := d.Destino.Recuperar(ctx, r)
		if e != nil {
			return ej.Conjunto{}, ErrConjunto
		}
		ok := bytes.Equal(p.Manifiesto, mb) && contenidoCoincide(p.Contenido, a)
		clear(p.Manifiesto)
		clear(p.Contenido)
		if !ok {
			return ej.Conjunto{}, ErrConjunto
		}
	}
	baseSHA, ejecucionRef := i.ManifiestoBaseSHA256, i.EjecucionVerificacionRef
	if i.Final.Verificacion.Estado == "pendiente_verificacion" {
		baseSHA, ejecucionRef = cs03.Huella(mustJSON(i.Final)), ir.ObjetoRef
	}
	return ej.Conjunto{Ref: ref, IndiceAutenticadoRef: ir.ObjetoRef, ManifiestoSHA256: cs03.Huella(mustJSON(i.Final)), ManifiestoBaseSHA256: baseSHA, EjecucionVerificacionRef: ejecucionRef, Manifiesto: i.Final, Origen: i.Origen}, nil
}

func mustJSON(m copias.Manifiesto) []byte { b, _ := json.Marshal(m); return b }

func componenteRef(id string) string { return "componente:" + cs03.Huella([]byte(id)) }

func mismaProteccion(a, b copias.Proteccion) bool {
	return b.Formato != "" && b.Algoritmo != "" && b.ClaveRef != "" && b.ClaveVersion != "" && b.AutenticacionRef != "" && a.Formato == b.Formato && a.Algoritmo == b.Algoritmo && a.ClaveRef == b.ClaveRef && a.ClaveVersion == b.ClaveVersion && a.AutenticacionRef == b.AutenticacionRef
}

func evidenciaOrigenValida(e copias.Evidencia) bool {
	if e.Ref == "" || e.ArranqueRef == "" {
		return false
	}
	for _, v := range []string{e.RecuentosSHA256, e.ContenidoSHA256, e.EsquemaSHA256, e.RolesSHA256, e.ACLSHA256, e.SecuenciasSHA256, e.ObjetosGrandesSHA256, e.FicherosSHA256} {
		if len(v) != 64 {
			return false
		}
		b, err := hex.DecodeString(v)
		if err != nil || hex.EncodeToString(b) != v {
			return false
		}
	}
	return true
}

func huellaReferencias(refs []puerto.Referencia) string {
	b, _ := json.Marshal(struct {
		Version     int                 `json:"version"`
		Referencias []puerto.Referencia `json:"referencias"`
	}{1, refs})
	return cs03.Huella(b)
}

// LeerComponente permite al ensayador leer por referencia lógica después de
// autenticar el índice completo. El llamante debe borrar el contenido tras usarlo.
func (d DestinoCS03) LeerComponente(ctx context.Context, conjuntoRef, componenteRef string) ([]byte, error) {
	if _, err := d.Recuperar(ctx, conjuntoRef); err != nil {
		return nil, err
	}
	i, _, err := d.leerIndice(ctx, conjuntoRef)
	if err != nil {
		return nil, err
	}
	mb, _ := json.Marshal(i.Original)
	for n, a := range i.Original.Componentes {
		if a.ID != componenteRef {
			continue
		}
		p, e := d.Destino.Recuperar(ctx, i.Componentes[n])
		if e != nil || !bytes.Equal(p.Manifiesto, mb) || !contenidoCoincide(p.Contenido, a) {
			clear(p.Manifiesto)
			clear(p.Contenido)
			return nil, ErrConjunto
		}
		clear(p.Manifiesto)
		if a.TamanoBytes == 0 {
			clear(p.Contenido)
			return []byte{}, nil
		}
		return p.Contenido, nil
	}
	return nil, ErrConjunto
}

func contenidoCoincide(b []byte, a copias.Artefacto) bool {
	if a.TamanoBytes == 0 {
		return len(b) == 1 && b[0] == 0 && a.SHA256 == cs03.Huella(nil)
	}
	return int64(len(b)) == a.TamanoBytes && cs03.Huella(b) == a.SHA256
}

func (d DestinoCS03) publicarIndice(ctx context.Context, i indiceConjunto) (puerto.Referencia, error) {
	fb, err := json.Marshal(i.Final)
	if err != nil {
		return puerto.Referencia{}, ErrConjunto
	}
	defer clear(fb)
	ib, err := json.Marshal(i)
	if err != nil {
		return puerto.Referencia{}, ErrConjunto
	}
	defer clear(ib)
	return d.Destino.Publicar(ctx, puerto.Solicitud{Vinculo: puerto.Vinculo{ConjuntoRef: i.Original.ConjuntoRef, ComponenteRef: "indice", Posicion: uint64(len(i.Componentes) + 1), ManifiestoSHA256: cs03.Huella(fb)}, Manifiesto: fb, Contenido: ib})
}

func (d DestinoCS03) leerIndice(ctx context.Context, ref string) (indiceConjunto, puerto.Referencia, error) {
	if !d.disponible() || ctx == nil || ctx.Err() != nil || !cs03.ReferenciaOpaca(ref) {
		return indiceConjunto{}, puerto.Referencia{}, ErrConjunto
	}
	r, err := d.Catalogo.Leer(ctx, ref)
	if err != nil || r.Vinculo.ConjuntoRef != ref || r.Vinculo.ComponenteRef != "indice" {
		return indiceConjunto{}, puerto.Referencia{}, ErrConjunto
	}
	p, err := d.Destino.Recuperar(ctx, r)
	if err != nil {
		return indiceConjunto{}, puerto.Referencia{}, ErrConjunto
	}
	defer clear(p.Manifiesto)
	defer clear(p.Contenido)
	var i indiceConjunto
	dec := json.NewDecoder(bytes.NewReader(p.Contenido))
	dec.DisallowUnknownFields()
	if dec.Decode(&i) != nil || dec.Decode(new(any)) != io.EOF || i.Version != 1 || i.Original.ConjuntoRef != ref || i.Final.ConjuntoRef != ref || !evidenciaOrigenValida(i.Origen) || !mismaProteccion(i.Original.Proteccion, d.ProteccionEsperada) || !mismaProteccion(i.Final.Proteccion, d.ProteccionEsperada) || len(copias.ValidarManifiesto(i.Original)) != 0 || len(copias.ValidarManifiesto(i.Final)) != 0 || i.Original.Verificacion.Estado != "pendiente_verificacion" {
		return indiceConjunto{}, puerto.Referencia{}, ErrConjunto
	}
	base := i.Final
	base.Verificacion = i.Original.Verificacion
	base.Proteccion.CifradoSHA256 = i.Original.Proteccion.CifradoSHA256
	if i.Original.Proteccion.CifradoSHA256 != strings.Repeat("0", 64) || i.Final.Proteccion.CifradoSHA256 != huellaReferencias(i.Componentes) || !reflect.DeepEqual(base, i.Original) || len(i.Componentes) != len(i.Original.Componentes) || r.Vinculo.Posicion != uint64(len(i.Componentes)+1) {
		return indiceConjunto{}, puerto.Referencia{}, ErrConjunto
	}
	fb, _ := json.Marshal(i.Final)
	if !bytes.Equal(p.Manifiesto, fb) || r.Vinculo.ManifiestoSHA256 != cs03.Huella(fb) {
		return indiceConjunto{}, puerto.Referencia{}, ErrConjunto
	}
	if i.Final.Verificacion.Estado == "pendiente_verificacion" {
		if i.IndiceInicial != nil || i.ManifiestoBaseSHA256 != "" || i.EjecucionVerificacionRef != "" {
			return indiceConjunto{}, puerto.Referencia{}, ErrConjunto
		}
	} else {
		pendiente := i.Final
		pendiente.Verificacion = i.Original.Verificacion
		pb := mustJSON(pendiente)
		if i.IndiceInicial == nil || i.ManifiestoBaseSHA256 != cs03.Huella(pb) || i.EjecucionVerificacionRef != i.IndiceInicial.ObjetoRef || i.IndiceInicial.Vinculo.ConjuntoRef != ref || i.IndiceInicial.Vinculo.ComponenteRef != "indice" || i.IndiceInicial.Vinculo.Posicion != uint64(len(i.Componentes)+1) || i.IndiceInicial.Vinculo.ManifiestoSHA256 != i.ManifiestoBaseSHA256 {
			return indiceConjunto{}, puerto.Referencia{}, ErrConjunto
		}
		anterior, er := d.Destino.Recuperar(ctx, *i.IndiceInicial)
		if er != nil {
			return indiceConjunto{}, puerto.Referencia{}, ErrConjunto
		}
		defer clear(anterior.Manifiesto)
		defer clear(anterior.Contenido)
		var original indiceConjunto
		decoder := json.NewDecoder(bytes.NewReader(anterior.Contenido))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&original) != nil || decoder.Decode(new(any)) != io.EOF || !bytes.Equal(anterior.Manifiesto, pb) || original.Version != 1 || !reflect.DeepEqual(original.Original, i.Original) || !reflect.DeepEqual(original.Final, pendiente) || !reflect.DeepEqual(original.Origen, i.Origen) || !reflect.DeepEqual(original.Componentes, i.Componentes) || original.IndiceInicial != nil || original.ManifiestoBaseSHA256 != "" || original.EjecucionVerificacionRef != "" {
			return indiceConjunto{}, puerto.Referencia{}, ErrConjunto
		}
	}
	return i, r, nil
}

var _ ej.Destino = DestinoCS03{}
