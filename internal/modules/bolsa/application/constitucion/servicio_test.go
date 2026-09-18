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

type repositorioPrueba struct {
	guardada ports.Constitucion
	vinculos []ports.VinculoCandidato
	actaRef  string
}

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
func (r *repositorioPrueba) RegistrarVinculos(_ context.Context, actaRef string, vinculos []ports.VinculoCandidato, _ time.Time) (ports.ReciboVinculosCandidato, error) {
	r.actaRef, r.vinculos = actaRef, vinculos
	return ports.ReciboVinculosCandidato{Nuevos: uint64(len(vinculos))}, nil
}
func (r *repositorioPrueba) ParticipacionesCandidato(context.Context, string) ([]ports.ParticipacionCandidato, error) {
	return nil, nil
}

func derivadorPrueba(t *testing.T) *DerivadorCandidatoHMAC {
	t.Helper()
	var clave [32]byte
	copy(clave[:], "clave-de-pruebas-para-candidatos")
	d, err := NuevoDerivadorCandidatoHMAC(clave)
	if err != nil {
		t.Fatal(err)
	}
	return d
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
			// huellas con ocho dígitos seguidos y una letra, como las reales
			ActaRef: "acta:importacion-convoca:12345678abcdef" + strings.Repeat("ab", 25), ImportacionRef: "importacion:convoca:87654321fedcba" + strings.Repeat("cd", 25),
			HuellaFicheroSHA256: strings.Repeat("ef", 32), Esquema: importacion.EsquemaResumenPersona,
		},
		Aceptadas: []importacion.FilaAceptada{
			fila(1, "***0001**", "Zamora", "Ana", "10.5"),
			fila(2, "***0002**", "Barranco", "Bruno", "22.25"),
			fila(3, "***0003**", "Alcalde", "Carla", "22.25"),
		},
	}
	repo := &repositorioPrueba{}
	servicio, err := NuevoServicio(recuperadorPrueba{lote}, repo, derivadorPrueba(t), func() time.Time { return time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC) })
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
	if repo.actaRef != lote.Acta.ActaRef || len(repo.vinculos) != 3 || recibo.Vinculos.Nuevos != 3 {
		t.Fatalf("vínculos no registrados: acta=%s vinculos=%d recibo=%+v", repo.actaRef, len(repo.vinculos), recibo.Vinculos)
	}
	esperado, _ := derivadorPrueba(t).CandidatoRef(importacion.IdentidadEnmascarada{Documento: "***0003**", PrimerApellido: "Alcalde", SegundoApellido: "Sintético", Nombre: "Carla"})
	if repo.vinculos[0].CandidatoRef != esperado || repo.vinculos[0].ParticipacionRef != c.Entradas[0].ParticipacionRef || !ReferenciaCandidatoValida(esperado) {
		t.Fatalf("vínculo del primer puesto inesperado: %+v (esperado %s)", repo.vinculos[0], esperado)
	}
}

func TestDerivadorCandidatoNormalizaIdentidad(t *testing.T) {
	d := derivadorPrueba(t)
	desdeLista, err := d.CandidatoRef(importacion.IdentidadEnmascarada{Documento: "***0071**", PrimerApellido: "LOZANO", SegundoApellido: "HIDALGO", Nombre: "YAGO"})
	if err != nil {
		t.Fatal(err)
	}
	desdeCertificado, err := d.CandidatoRef(importacion.IdentidadEnmascarada{Documento: "12300719-X", PrimerApellido: "Lozano", SegundoApellido: "Hidalgo", Nombre: " Yago "})
	if err != nil {
		t.Fatal(err)
	}
	if desdeLista != desdeCertificado {
		t.Fatalf("la lista y el certificado deben derivar la misma referencia: %s != %s", desdeLista, desdeCertificado)
	}
	conAcentos, _ := d.CandidatoRef(importacion.IdentidadEnmascarada{Documento: "***0071**", PrimerApellido: "Muñoz", SegundoApellido: "Peña", Nombre: "José María"})
	sinAcentos, _ := d.CandidatoRef(importacion.IdentidadEnmascarada{Documento: "***0071**", PrimerApellido: "MUNOZ", SegundoApellido: "PENA", Nombre: "JOSE  MARIA"})
	if conAcentos != sinAcentos {
		t.Fatal("los diacríticos y los espacios no deben cambiar la referencia")
	}
	otro, _ := d.CandidatoRef(importacion.IdentidadEnmascarada{Documento: "***0072**", PrimerApellido: "LOZANO", SegundoApellido: "HIDALGO", Nombre: "YAGO"})
	if otro == desdeLista {
		t.Fatal("documentos distintos deben derivar referencias distintas")
	}
	if _, err := d.CandidatoRef(importacion.IdentidadEnmascarada{Documento: "1234", PrimerApellido: "X", Nombre: "Y"}); err == nil {
		t.Fatal("documento inválido aceptado")
	}
	if _, err := d.CandidatoRef(importacion.IdentidadEnmascarada{Documento: "***0071**", PrimerApellido: "", Nombre: "Y"}); err == nil {
		t.Fatal("identidad sin primer apellido aceptada")
	}
	if enmascarado, err := EnmascararDocumento("X1234567L"); err != nil || enmascarado != "***4567**" {
		t.Fatalf("NIE enmascarado: %s %v", enmascarado, err)
	}
	if _, err := NuevoDerivadorCandidatoHMAC([32]byte{}); err == nil {
		t.Fatal("clave vacía aceptada")
	}
}
