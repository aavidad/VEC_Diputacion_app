package constitucionconvoca

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/application/constitucion"
	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// Recuperador mantiene el contrato del staging cifrado de CONVOCA.
type Recuperador interface {
	RecuperarLote(context.Context, string, string) (importacion.LoteValidado, importacionapp.EstadoImportacion, bool, error)
}

type Fuente struct {
	recuperador Recuperador
	derivador   constitucion.DerivadorCandidato
}

var _ ports.RecuperadorListaDefinitiva = (*Fuente)(nil)

func NuevaFuente(recuperador Recuperador, derivador constitucion.DerivadorCandidato) (*Fuente, error) {
	if recuperador == nil || derivador == nil {
		return nil, constitucion.ErrDependenciasRequeridas
	}
	return &Fuente{recuperador: recuperador, derivador: derivador}, nil
}

// RecuperarListaDefinitiva resuelve aquí, y solo aquí, el desempate histórico
// de CONVOCA. Las posiciones pasan al núcleo ya fijadas.
func (f *Fuente) RecuperarListaDefinitiva(ctx context.Context, consulta ports.ConsultaListaDefinitiva) (ports.ListaDefinitivaAutorizada, bool, error) {
	if f == nil || ctx == nil || consulta.Fuente != ports.FuenteImportacionConvoca || consulta.Referencia == "" || consulta.CategoriaRef == "" {
		return ports.ListaDefinitivaAutorizada{}, false, ports.ErrConstitucionBolsaInvalida
	}
	lote, _, existe, err := f.recuperador.RecuperarLote(ctx, consulta.Referencia, consulta.CategoriaRef)
	if err != nil || !existe {
		return ports.ListaDefinitivaAutorizada{}, existe, err
	}
	if lote.Acta.HuellaFicheroSHA256 != consulta.Referencia || lote.Acta.CategoriaRef != consulta.CategoriaRef {
		return ports.ListaDefinitivaAutorizada{}, false, ports.ErrConstitucionBolsaInvalida
	}
	if lote.Acta.Esquema != importacion.EsquemaResumenPersona {
		return ports.ListaDefinitivaAutorizada{}, false, constitucion.ErrActaNoEsResumen
	}
	filas := make([]importacion.FilaAceptada, 0, len(lote.Aceptadas))
	for _, fila := range lote.Aceptadas {
		if fila.Resumen != nil {
			filas = append(filas, fila)
		}
	}
	if len(filas) == 0 {
		return ports.ListaDefinitivaAutorizada{}, false, constitucion.ErrActaSinFilasAceptadas
	}
	sort.SliceStable(filas, func(i, j int) bool {
		a, b := filas[i], filas[j]
		ta, tb := constitucion.PuntuacionConvoca(a.Resumen.Total), constitucion.PuntuacionConvoca(b.Resumen.Total)
		if ta != tb {
			return ta > tb
		}
		ka := strings.ToLower(a.Identidad.PrimerApellido + " " + a.Identidad.SegundoApellido + " " + a.Identidad.Nombre)
		kb := strings.ToLower(b.Identidad.PrimerApellido + " " + b.Identidad.SegundoApellido + " " + b.Identidad.Nombre)
		if ka != kb {
			return ka < kb
		}
		return a.Numero < b.Numero
	})
	acta := lote.Acta
	sufijoActa := constitucion.SufijoReferenciaOpaca(acta.ActaRef)
	bolsaRef := acta.BolsaRef
	if bolsaRef == "" {
		categoria := acta.CategoriaRef
		if indice := strings.LastIndex(categoria, ":"); indice >= 0 {
			categoria = categoria[indice+1:]
		}
		bolsaRef = "bolsa:" + categoria + ":" + sufijoActa
	}
	lista := ports.ListaDefinitivaAutorizada{
		Fuente: ports.FuenteImportacionConvoca, Referencia: acta.ActaRef, Version: 1,
		ConvocatoriaRef: "importacion:convoca:" + constitucion.SufijoReferenciaOpaca(acta.ImportacionRef),
		CategoriaRef:    acta.CategoriaRef, BolsaRef: bolsaRef,
		HuellaListadoSHA256:        acta.HuellaFicheroSHA256,
		AutorizacionPublicacionRef: acta.ActaRef, HuellaAutorizacionSHA256: acta.HuellaFicheroSHA256,
		Posiciones: make([]ports.PosicionListaDefinitiva, 0, len(filas)),
	}
	for indice, fila := range filas {
		orden := uint64(indice + 1)
		candidatoRef, err := f.derivador.CandidatoRef(fila.Identidad)
		if err != nil {
			return ports.ListaDefinitivaAutorizada{}, false, errors.Join(ports.ErrConstitucionBolsaInvalida, err)
		}
		identidad := fila.Identidad
		lista.Posiciones = append(lista.Posiciones, ports.PosicionListaDefinitiva{
			Posicion: orden, PersonaRef: candidatoRef, CandidatoRef: candidatoRef,
			SujetoRef:        "sujeto:convoca:" + constitucion.SufijoReferenciaOpaca(identidad.Documento+"|"+identidad.PrimerApellido+"|"+identidad.SegundoApellido+"|"+identidad.Nombre),
			ParticipacionRef: "participacion:" + constitucion.SufijoReferenciaOpaca(bolsaRef+"|"+identidad.Documento+"|"+strconv.FormatUint(orden, 10)),
			Puntuacion:       fila.Resumen.Total, FilaOrigenNumero: fila.Numero,
		})
	}
	return lista, true, nil
}
