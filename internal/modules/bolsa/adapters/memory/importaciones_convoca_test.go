package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	aplicacion "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

func TestRepositorioImportacionesConvocaHaceCASYCopiasDefensivas(t *testing.T) {
	repositorio := NuevoRepositorioImportacionesConvoca()
	lote := loteConvocaMemoriaPrueba()
	actaGuardada, reutilizado, err := repositorio.GuardarSiAusente(context.Background(), lote)
	if err != nil || reutilizado {
		t.Fatalf("guardar: reutilizado=%v error=%v", reutilizado, err)
	}
	if actaGuardada.ActaRef != lote.Acta.ActaRef ||
		actaGuardada.HuellaFicheroSHA256 != lote.Acta.HuellaFicheroSHA256 {
		t.Fatalf("acta guardada inesperada: %#v", actaGuardada)
	}
	lote.Aceptadas[0].Resumen.Total = "999"
	releido, existe, err := repositorio.ObtenerPorHuella(
		context.Background(), lote.Acta.HuellaFicheroSHA256,
	)
	if err != nil || !existe || releido.Aceptadas[0].Resumen.Total != "2" {
		t.Fatalf("copia defensiva fallida: existe=%v error=%v lote=%#v", existe, err, releido)
	}
	releido.Aceptadas[0].Resumen.Total = "777"
	otraLectura, _, _ := repositorio.ObtenerPorHuella(context.Background(), lote.Acta.HuellaFicheroSHA256)
	if otraLectura.Aceptadas[0].Resumen.Total != "2" {
		t.Fatal("la lectura comparte estado durable")
	}
	_, reutilizado, err = repositorio.GuardarSiAusente(context.Background(), loteConvocaMemoriaPrueba())
	if err != nil || !reutilizado || repositorio.NumeroLotes() != 1 {
		t.Fatalf("CAS idempotente fallido: reutilizado=%v error=%v lotes=%d",
			reutilizado, err, repositorio.NumeroLotes())
	}
	conflictivo := loteConvocaMemoriaPrueba()
	conflictivo.Acta.ActorRef = "actor:rrhh:otro"
	if _, _, err := repositorio.GuardarSiAusente(
		context.Background(), conflictivo,
	); !errors.Is(err, aplicacion.ErrImportacionEnConflicto) {
		t.Fatalf("acta distinta con mismo SHA aceptada: %v", err)
	}
}

func TestRepositorioImportacionesConvocaRespetaContextoYValida(t *testing.T) {
	repositorio := NuevoRepositorioImportacionesConvoca()
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, _, err := repositorio.GuardarSiAusente(ctx, loteConvocaMemoriaPrueba()); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion ignorada: %v", err)
	}
	if _, _, err := repositorio.GuardarSiAusente(context.Background(), dominio.LoteValidado{}); !errors.Is(err, ErrLoteConvocaInvalido) {
		t.Fatalf("lote invalido aceptado: %v", err)
	}
}

func loteConvocaMemoriaPrueba() dominio.LoteValidado {
	huella := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	return dominio.LoteValidado{
		Acta: dominio.ActaImportacion{
			ActaRef:             "acta:importacion-convoca:" + huella,
			ImportacionRef:      "importacion:convoca:" + huella,
			HuellaFicheroSHA256: huella, NombreFichero: "sintetico.xls",
			FicheroCustodiadoRef: "almacen:objeto:convoca:" + huella,
			ActorRef:             "actor:rrhh:memoria", RegistradaEn: time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC),
			Esquema: dominio.EsquemaResumenPersona, FilasLeidas: 1, FilasAceptadas: 1,
			Procedencia: dominio.NuevaProcedenciaNoAutoritativa(),
		},
		Aceptadas: []dominio.FilaAceptada{{
			Numero: 2, Esquema: dominio.EsquemaResumenPersona,
			Identidad: dominio.IdentidadEnmascarada{Documento: "***0001**", PrimerApellido: "Sintetica", Nombre: "Ana"},
			Turno:     "Libre", Resumen: &dominio.ResumenPersona{Experiencia: "1", Formacion: "1", Total: "2"},
		}},
	}
}
