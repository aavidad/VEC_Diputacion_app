package constitucion

import (
	"context"
	"strings"
	"testing"
	"time"

	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type recuperadorPrueba struct{ lote importacion.LoteValidado }

func (r recuperadorPrueba) RecuperarLote(context.Context, string, string) (importacion.LoteValidado, importacionapp.EstadoImportacion, bool, error) {
	return r.lote, importacionapp.EstadoImportacion{}, true, nil
}

type repositorioPrueba struct{ guardada ports.Constitucion }

func (r *repositorioPrueba) Constituir(_ context.Context, c ports.Constitucion) (ports.ReciboConstitucion, error) {
	r.guardada = c
	return ports.ReciboConstitucion{ActaRef: c.ActaRef, BolsaRef: c.Bolsa.BolsaRef}, nil
}
func (r *repositorioPrueba) ListarVigentes(context.Context) ([]ports.ConstitucionVigente, error) {
	return nil, nil
}
func (r *repositorioPrueba) Entradas(context.Context, string, uint64) ([]ports.EntradaConstitucion, error) {
	return nil, nil
}

func fila(numero int, doc, apellido, nombre, total string) importacion.FilaAceptada {
	return importacion.FilaAceptada{
		Numero: numero, Esquema: importacion.EsquemaResumenPersona,
		Identidad: importacion.IdentidadEnmascarada{Documento: doc, PrimerApellido: apellido, SegundoApellido: "Sintético", Nombre: nombre},
		Turno:     "Libre",
		Resumen:   &importacion.ResumenPersona{Experiencia: "1", Formacion: "1", Total: total},
	}
}

func TestConstituirOrdenaPorTotalYPersisteCanonicos(t *testing.T) {
	lote := importacion.LoteValidado{
		Acta: importacion.ActaImportacion{
			CategoriaRef: "categoria:rpt:administrativo", BolsaRef: "bolsa:administrativo:2026-09-18",
			ActaRef: "acta:importacion-convoca:" + strings.Repeat("ab", 32), ImportacionRef: "importacion:convoca:" + strings.Repeat("cd", 32),
			HuellaFicheroSHA256: strings.Repeat("ef", 32), Esquema: importacion.EsquemaResumenPersona,
		},
		Aceptadas: []importacion.FilaAceptada{
			fila(1, "***0001**", "Zamora", "Ana", "10.5"),
			fila(2, "***0002**", "Barranco", "Bruno", "22.25"),
			fila(3, "***0003**", "Alcalde", "Carla", "22.25"),
		},
	}
	repo := &repositorioPrueba{}
	servicio, err := NuevoServicio(recuperadorPrueba{lote}, repo, func() time.Time { return time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	recibo, err := servicio.Constituir(context.Background(), Solicitud{HuellaFicheroSHA256: lote.Acta.HuellaFicheroSHA256, CategoriaRef: lote.Acta.CategoriaRef, ActorRef: "actor:rrhh:pruebas"})
	if err != nil {
		t.Fatalf("constituir: %v", err)
	}
	if recibo.BolsaRef != "bolsa:administrativo:2026-09-18" {
		t.Fatalf("bolsa_ref inesperada: %s", recibo.BolsaRef)
	}
	c := repo.guardada
	if err := c.Bolsa.Validar(); err != nil {
		t.Fatalf("bolsa inválida: %v", err)
	}
	if err := c.Instantanea.Validar(); err != nil {
		t.Fatalf("instantánea inválida: %v", err)
	}
	if len(c.Entradas) != 3 || c.Entradas[0].FilaNumero != 3 || c.Entradas[1].FilaNumero != 2 || c.Entradas[2].FilaNumero != 1 {
		t.Fatalf("orden inesperado (22.25 Alcalde, 22.25 Barranco, 10.5 Zamora): %+v", c.Entradas)
	}
	for _, entrada := range c.Instantanea.Entradas {
		if strings.ContainsAny(entrada.Participacion.ParticipacionRef, "0123456789") || strings.Contains(entrada.Participacion.SujetoRef, "*") {
			t.Fatalf("referencia con dígitos o máscara: %+v", entrada.Participacion)
		}
		if entrada.Participacion.Situaciones[0].EstadoClave != EstadoInicial {
			t.Fatalf("situación inicial inesperada: %+v", entrada.Participacion.Situaciones[0])
		}
	}
}
