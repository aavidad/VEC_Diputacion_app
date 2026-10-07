// vec-copias-restauracion conserva propuestas declaradas; nunca sustituye datos.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"
	a "vec-diputacion-granada/internal/modules/administracion/adapters/restauracioncopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	d "vec-diputacion-granada/internal/modules/administracion/domain/restauracioncopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/restauracioncopias"
)

type entrada struct {
	Accion              string            `json:"accion"`
	Ahora               time.Time         `json:"ahora"`
	PersonaDeclaradaRef string            `json:"persona_declarada_ref"`
	Configuracion       d.Configuracion   `json:"configuracion"`
	Propuesta           d.Propuesta       `json:"propuesta"`
	SHA256              string            `json:"sha256"`
	Version             uint64            `json:"version"`
	Manifiesto          copias.Manifiesto `json:"manifiesto"`
	Destino             copias.Inventario `json:"destino"`
	Politica            copias.Politica   `json:"politica"`
}
type salida struct {
	Alcance              string      `json:"alcance"`
	HabilitaRestauracion bool        `json:"habilita_restauracion"`
	Autorizacion         string      `json:"autorizacion"`
	EstadoClave          string      `json:"estado_clave"`
	Registro             *p.Registro `json:"registro,omitempty"`
}

func run(args []string, in io.Reader, out io.Writer) int {
	emitir := func(key string, r *p.Registro, code int) int {
		if json.NewEncoder(out).Encode(salida{Alcance: "declaracion_offline", HabilitaRestauracion: false, Autorizacion: "no_comprobada", EstadoClave: key, Registro: r}) != nil {
			return 4
		}
		return code
	}
	if len(args) != 2 || args[0] != "-registro" {
		return emitir("copias_restauracion_error_argumentos", nil, 3)
	}
	var e entrada
	if leerEstricto(in, &e) != nil || !d.UTC(e.Ahora) || e.Configuracion.Entorno != "sintetico_offline" {
		return emitir("copias_restauracion_error_entrada", nil, 3)
	}
	doble, err := e.Configuracion.ExigirDobleControl()
	if err != nil || doble != e.Propuesta.DobleControl || e.Propuesta.Entorno != e.Configuracion.Entorno {
		return emitir("copias_restauracion_configuracion_invalida", nil, 2)
	}
	s := d.Sellada{Propuesta: e.Propuesta, SHA256: e.SHA256}
	accion := p.Revisar
	switch e.Accion {
	case "proponer":
		accion = p.Proponer
		s, err = d.Sellar(e.Propuesta)
	case "revisar", "consultar":
		err = s.Comprobar(e.Ahora)
	default:
		return emitir("copias_restauracion_error_argumentos", nil, 3)
	}
	if err != nil {
		return emitir(err.Error(), nil, 2)
	}
	if err = s.Comprobar(e.Ahora); err != nil {
		return emitir(err.Error(), nil, 2)
	}
	if e.Manifiesto.ConjuntoRef != e.Propuesta.ConjuntoRef || e.Destino.Ref != e.Propuesta.DestinoRef || d.Huella("vec-restauracion-conjunto-v1", e.Manifiesto) != e.Propuesta.ConjuntoSHA256 || d.Huella("vec-restauracion-politica-v1", e.Politica) != e.Propuesta.PoliticaSHA256 || e.Politica.Ref != e.Propuesta.PoliticaRef || copias.CompararVersiones(e.Manifiesto, e.Destino, e.Politica, copias.ConjuntoCompleto).Estado != copias.Compatible {
		return emitir("copias_restauracion_compatibilidad_bloqueada", nil, 2)
	}
	c := p.Concesion{PersonaRef: e.PersonaDeclaradaRef, Ref: "declaracion:offline", Caduca: e.Propuesta.Caduca, Acceso: p.Acceso{SesionRef: "declaracion:offline", Accion: accion, DestinoRef: e.Propuesta.DestinoRef, PropuestaSHA256: s.SHA256}}
	registro, err := a.Abrir(args[1], func(_ context.Context, actual p.Concesion) error {
		if actual != c || !e.Ahora.Before(actual.Caduca) {
			return errors.New("copias_restauracion_autoridad_no_vigente")
		}
		return nil
	})
	if err != nil {
		return emitir("copias_restauracion_registro_no_valido", nil, 2)
	}
	var r p.Registro
	switch e.Accion {
	case "proponer":
		r, err = registro.Crear(context.Background(), s, c)
	case "consultar":
		r, err = registro.Leer(context.Background(), s.Propuesta.Ref, c)
	case "revisar":
		var rev d.Revision
		rev, err = s.Revisar(c.PersonaRef, e.Ahora)
		if err == nil {
			r, err = registro.RevisarCAS(context.Background(), s.Propuesta.Ref, e.Version, s.SHA256, rev, c)
		}
	}
	err = errors.Join(err, registro.Close())
	if err != nil {
		key := "copias_restauracion_registro_no_valido"
		for _, known := range []error{d.ErrMismaPersona, d.ErrAlterada, d.ErrCaducada, a.ErrConflicto} {
			if errors.Is(err, known) {
				key = known.Error()
			}
		}
		return emitir(key, nil, 2)
	}
	return emitir("copias_restauracion_declaracion_guardada", &r, 0)
}
func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout)) }
