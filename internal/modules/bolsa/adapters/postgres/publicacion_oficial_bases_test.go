package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

func declaracionPublicacionAdaptadorPrueba() ports.DeclaracionPublicacionOficial {
	boletin := int64(201)
	return ports.DeclaracionPublicacionOficial{AnuncioRef: "ENSAYO-BOP-2026-000201", BoletinNumero: &boletin,
		ClaveIdempotencia: "f195-prueba-0001", ConvocatoriaRef: "cv1_" + strings.Repeat("a", 64) + "_v1",
		FechaPublicacion: "2026-10-15", HuellaVersionSHA256: strings.Repeat("c", 64), Medio: "bop_granada"}
}

func reciboPublicacionAdaptadorPrueba(d ports.DeclaracionPublicacionOficial) ports.ReciboPublicacionOficial {
	ref := "publicacion_bases:" + strings.Repeat("b", 64)
	boletin := *d.BoletinNumero
	return ports.ReciboPublicacionOficial{PublicacionRef: ref, ReciboRef: "recibo:" + ref, EventoRef: "evento:" + ref,
		ConvocatoriaRef: d.ConvocatoriaRef, Secuencia: 1, Medio: d.Medio, BoletinNumero: &boletin, AnuncioRef: d.AnuncioRef,
		FechaPublicacion: d.FechaPublicacion, AuditoriaRef: "auditoria:1", RegistradaEn: time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)}
}

func TestPublicacionOficialReciboLigadoALaDeclaracion(t *testing.T) {
	d := declaracionPublicacionAdaptadorPrueba()
	if !reciboPublicacionOficialValido(reciboPublicacionAdaptadorPrueba(d), d) {
		t.Fatal("recibo válido rechazado")
	}
	for nombre, alterar := range map[string]func(*ports.ReciboPublicacionOficial){
		"medio":     func(r *ports.ReciboPublicacionOficial) { r.Medio = "boe" },
		"boletin":   func(r *ports.ReciboPublicacionOficial) { otro := int64(7); r.BoletinNumero = &otro },
		"version":   func(r *ports.ReciboPublicacionOficial) { r.ConvocatoriaRef = "cv1_otra_v1" },
		"recibo":    func(r *ports.ReciboPublicacionOficial) { r.ReciboRef = "recibo:otro" },
		"auditoria": func(r *ports.ReciboPublicacionOficial) { r.AuditoriaRef = "" },
		"sustituye": func(r *ports.ReciboPublicacionOficial) { s := "publicacion_bases:x"; r.SustituyeRef = &s },
	} {
		r := reciboPublicacionAdaptadorPrueba(d)
		alterar(&r)
		if reciboPublicacionOficialValido(r, d) {
			t.Fatalf("%s: recibo ajeno admitido", nombre)
		}
	}
}

func TestPublicacionOficialErroresSQLCatalogados(t *testing.T) {
	for codigo, esperado := range map[string]error{
		"42501": vd.ErrAutorizacionDenegada, "22023": ports.ErrPublicacionOficialInvalida,
		"B9601": ports.ErrPublicacionOficialNoVigente, "23505": ports.ErrPublicacionOficialConflicto,
		"55000": ports.ErrPublicacionOficialNoDisponible,
	} {
		if err := errorPublicacionOficialSQL(&pgconn.PgError{Code: codigo}); !errors.Is(err, esperado) {
			t.Fatalf("%s -> %v", codigo, err)
		}
	}
}

func TestPublicacionOficialCommitIndeterminadoRecuperaElRecibo(t *testing.T) {
	d := declaracionPublicacionAdaptadorPrueba()
	i := ports.IntencionPublicacionOficial{Solicitud: ports.SolicitudDeclaracionPublicacionOficial{Declaracion: d}}
	llamadas := 0
	recibo, err := declararPublicacionOficialConRecuperacion(context.Background(), i,
		func(context.Context, ports.IntencionPublicacionOficial) (ports.ReciboPublicacionOficial, error) {
			llamadas++
			if llamadas == 1 {
				return ports.ReciboPublicacionOficial{}, errCommitPublicacionOficialIndeterminado
			}
			r := reciboPublicacionAdaptadorPrueba(d)
			r.Reutilizada = true
			return r, nil
		})
	if err != nil || llamadas != 2 || !recibo.Reutilizada {
		t.Fatalf("recuperación: %v, llamadas %d", err, llamadas)
	}
	llamadas = 0
	_, err = declararPublicacionOficialConRecuperacion(context.Background(), i,
		func(context.Context, ports.IntencionPublicacionOficial) (ports.ReciboPublicacionOficial, error) {
			llamadas++
			return ports.ReciboPublicacionOficial{}, errCommitPublicacionOficialIndeterminado
		})
	if !errors.Is(err, ports.ErrPublicacionOficialNoConfirmada) || llamadas != 2 {
		t.Fatalf("sin confirmación: %v, llamadas %d", err, llamadas)
	}
}

func TestPublicacionOficialSinPoolNoDisponible(t *testing.T) {
	if _, err := NuevoRepositorioPublicacionOficialPostgreSQL(nil); !errors.Is(err, ports.ErrPublicacionOficialNoDisponible) {
		t.Fatal(err)
	}
}
