// Package catalogo traduce el catálogo de datos personales (F1.2,
// internal/vec/datospersonales) a lo que pide la ficha propia: qué datos de
// contacto se piden en la inscripción y si son obligatorios.
package catalogo

import (
	"context"
	"errors"
	"strconv"

	"vec-diputacion-granada/internal/modules/aspirantes/domain"
	"vec-diputacion-granada/internal/modules/aspirantes/ports"
	"vec-diputacion-granada/internal/vec/datospersonales"
)

var ErrConfiguracion = errors.New("aspirantes: catálogo de exigencias mal configurado")

// Resolutor es la parte del resolutor de datospersonales que se usa.
type Resolutor interface {
	Para(context.Context, datospersonales.TipoConvocatoria, datospersonales.Momento) ([]datospersonales.DatoRequerido, datospersonales.Catalogo, error)
}

// correspondencia de datos del catálogo con campos de la ficha. El correo
// sigue en Usuarios en este corte.
var correspondencia = map[string][]domain.CampoFicha{
	"telefono":               {domain.CampoTelefono},
	"telefono_secundario":    {domain.CampoMovil},
	"domicilio_notificacion": {domain.CampoDomicilio, domain.CampoCodigoPostal},
}

// Adaptador resuelve las exigencias para los tipos de convocatoria que cuentan
// para la persona. Mientras las inscripciones no lleven `asp_`, esos tipos
// son configuración (por defecto, bolsa).
type Adaptador struct {
	resolutor Resolutor
	tipos     []datospersonales.TipoConvocatoria
}

func Nuevo(r Resolutor, tipos []datospersonales.TipoConvocatoria) (*Adaptador, error) {
	if r == nil || len(tipos) == 0 || len(tipos) > 4 {
		return nil, ErrConfiguracion
	}
	vistos := map[datospersonales.TipoConvocatoria]bool{}
	for _, t := range tipos {
		// Promoción interna y provisión son trámites de empleado: no pasan
		// por Aspirantes.
		if !t.Valido() || t.EsTramiteEmpleado() || vistos[t] {
			return nil, ErrConfiguracion
		}
		vistos[t] = true
	}
	return &Adaptador{resolutor: r, tipos: append([]datospersonales.TipoConvocatoria{}, tipos...)}, nil
}

var _ ports.CatalogoExigenciasFicha = (*Adaptador)(nil)

func (a *Adaptador) ExigenciasContactoFichaPropia(ctx context.Context) (ports.ExigenciasContacto, error) {
	vacia := ports.ExigenciasContacto{}
	if a == nil || a.resolutor == nil || ctx == nil {
		return vacia, ports.ErrNoDisponible
	}
	type acumulado struct {
		obligatorio, siempre bool
		condicion            string
	}
	campos := map[domain.CampoFicha]*acumulado{}
	orden := []domain.CampoFicha{}
	referencia := ""
	ejemplo := false
	for _, tipo := range a.tipos {
		datos, catalogo, err := a.resolutor.Para(ctx, tipo, datospersonales.MomentoInscripcion)
		if err != nil {
			return vacia, ports.ErrNoDisponible
		}
		ref := catalogo.CatalogoID + ":" + strconv.Itoa(catalogo.Version) + ":" + catalogo.HuellaCatalogo
		if referencia != "" && ref != referencia {
			return vacia, ports.ErrNoDisponible
		}
		referencia = ref
		ejemplo = ejemplo || catalogo.PaqueteEjemplo
		for _, d := range datos {
			destino, ok := correspondencia[d.Dato]
			if !ok || d.Custodia != datospersonales.CustodiaAspirantes || d.Categoria != datospersonales.CategoriaOrdinaria {
				continue
			}
			ejemplo = ejemplo || d.EsEjemplo()
			for _, campo := range destino {
				e, existe := campos[campo]
				if !existe {
					e = &acumulado{condicion: d.Condicion}
					campos[campo] = e
					orden = append(orden, campo)
				}
				e.obligatorio = e.obligatorio || d.Obligatoriedad == datospersonales.Obligatorio
				// Si algún tipo lo pide sin condición, se pide siempre.
				if d.Obligatoriedad != datospersonales.Condicional {
					e.siempre = true
				}
			}
		}
	}
	resultado := ports.ExigenciasContacto{CatalogoRef: referencia, Ejemplo: ejemplo, Campos: []domain.ExigenciaCampo{}}
	for _, campo := range orden {
		e := campos[campo]
		x := domain.ExigenciaCampo{Campo: campo, Obligatorio: e.obligatorio}
		if !e.siempre {
			x.Condicion = e.condicion
		}
		resultado.Campos = append(resultado.Campos, x)
	}
	if domain.ValidarExigencias(resultado.Campos) != nil || len(referencia) == 0 || len(referencia) > 128 {
		return vacia, ports.ErrNoDisponible
	}
	return resultado, nil
}
