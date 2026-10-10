package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/constitucion"
	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	bolsadominio "vec-diputacion-granada/internal/modules/bolsa/domain"
	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// Datos sintéticos: ninguna fila procede de una lista real.

func filaNombrePropioPrueba(numero int, doc, apellido1, apellido2, nombre string) importacion.FilaAceptada {
	return importacion.FilaAceptada{
		Numero: numero, Esquema: importacion.EsquemaResumenPersona,
		Identidad: importacion.IdentidadEnmascarada{Documento: doc, PrimerApellido: apellido1, SegundoApellido: apellido2, Nombre: nombre},
		Turno:     "Libre",
		Resumen:   &importacion.ResumenPersona{Experiencia: "1", Formacion: "2", Total: "3"},
	}
}

func loteNombrePropioPrueba() importacion.LoteValidado {
	filas := []importacion.FilaAceptada{
		filaNombrePropioPrueba(2, "***4821**", "Moreno", "Castillo", "Lucía"),
		filaNombrePropioPrueba(3, "***1190**", "Reyes", "Álvarez", "Antonio"),
		// Filas 4 y 5: misma identidad enmascarada; no se atribuyen a nadie.
		filaNombrePropioPrueba(4, "***7302**", "Peña", "Ibáñez", "José"),
		filaNombrePropioPrueba(5, "***7302**", "Peña", "Ibáñez", "José"),
	}
	huella, categoria := strings.Repeat("ab", 32), "categoria:rpt:auxiliar"
	contexto := importacion.ReferenciaContexto(huella, categoria)
	return importacion.LoteValidado{
		Acta: importacion.ActaImportacion{
			CategoriaRef: categoria, ActaRef: "acta:importacion-convoca:" + contexto,
			ImportacionRef: "importacion:convoca:" + contexto, HuellaFicheroSHA256: huella,
			FicheroCustodiadoRef: "custodia:convoca:sintetico", NombreFichero: "lista_sintetica.xls",
			ActorRef: "actor:rrhh:pruebas", RegistradaEn: time.Date(2026, 9, 29, 9, 0, 0, 0, time.UTC),
			Esquema: importacion.EsquemaResumenPersona, FilasLeidas: len(filas), FilasAceptadas: len(filas),
			Procedencia: importacion.NuevaProcedenciaNoAutoritativa(),
		},
		Aceptadas: filas,
	}
}

type recuperadorNombrePropioPrueba struct {
	lote    importacion.LoteValidado
	err     error
	llamado *int
	// fallaHuella hace fallar solo el acta de esa huella.
	fallaHuella string
}

func (r recuperadorNombrePropioPrueba) RecuperarLote(_ context.Context, huella, _ string) (importacion.LoteValidado, importacionapp.EstadoImportacion, bool, error) {
	*r.llamado++
	if huella == r.fallaHuella {
		return importacion.LoteValidado{}, importacionapp.EstadoImportacion{}, false, errors.New("acta ilegible")
	}
	return r.lote, importacionapp.EstadoImportacion{}, r.err == nil, r.err
}

func fuenteNombrePropioPrueba(t *testing.T, err error) (*fuenteConstituidaRRHHDesarrollo, *constitucion.DerivadorCandidatoHMAC, *int) {
	t.Helper()
	var clave [32]byte
	copy(clave[:], "clave-de-pruebas-para-candidatos")
	derivador, e := constitucion.NuevoDerivadorCandidatoHMAC(clave)
	if e != nil {
		t.Fatal(e)
	}
	llamadas := new(int)
	repo := repositorioVariasBolsasRRHHPrueba{vigentes: []ports.ConstitucionVigente{
		{CategoriaRef: "categoria:rpt:auxiliar", Bolsa: bolsadominio.BolsaConstituida{BolsaRef: "bolsa:auxiliar", HuellaListadoSHA256: strings.Repeat("ab", 32)}},
		{CategoriaRef: "categoria:rpt:otra", Bolsa: bolsadominio.BolsaConstituida{BolsaRef: "bolsa:otra", HuellaListadoSHA256: strings.Repeat("cd", 32)}},
	}}
	return &fuenteConstituidaRRHHDesarrollo{repositorio: repo, derivador: derivador,
		recuperador: recuperadorNombrePropioPrueba{lote: loteNombrePropioPrueba(), err: err, llamado: llamadas}}, derivador, llamadas
}

func TestNombrePropioMiBolsaSoloLaFilaDeLaTitular(t *testing.T) {
	fuente, derivador, llamadas := fuenteNombrePropioPrueba(t, nil)
	lote := loteNombrePropioPrueba()
	propia, err := derivador.CandidatoRef(lote.Aceptadas[1].Identidad)
	if err != nil {
		t.Fatal(err)
	}
	nombre, apellidos, encontrado, err := fuente.nombrePropio(t.Context(), propia, []string{"bolsa:auxiliar"})
	if err != nil || !encontrado || nombre != "Antonio" || apellidos != "Reyes Álvarez" || *llamadas != 1 {
		t.Fatalf("nombre propio inexacto: %q %q %v %v", nombre, apellidos, encontrado, err)
	}
	ambigua, err := derivador.CandidatoRef(lote.Aceptadas[2].Identidad)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre, candidato string
		bolsas            []string
		recuperaciones    int
	}{
		{"fila ambigua", ambigua, []string{"bolsa:auxiliar"}, 1},
		{"candidato ajeno al acta", "can_ajenoSintetico", []string{"bolsa:auxiliar"}, 1},
		{"bolsa que no es suya", propia, []string{"bolsa:inexistente"}, 0},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			*llamadas = 0
			nombre, _, encontrado, err := fuente.nombrePropio(t.Context(), caso.candidato, caso.bolsas)
			if err != nil || encontrado || nombre != "" || *llamadas != caso.recuperaciones {
				t.Fatalf("atribuyó un nombre: %q %v %v %d", nombre, encontrado, err, *llamadas)
			}
		})
	}
}

func TestNombrePropioMiBolsaFalloSinDatosPersonales(t *testing.T) {
	fuente, _, _ := fuenteNombrePropioPrueba(t, errors.New("descifrado Antonio Reyes"))
	_, _, encontrado, err := fuente.nombrePropio(t.Context(), "can_sintetico", []string{"bolsa:auxiliar"})
	if encontrado || !errors.Is(err, errNombrePropioMiBolsaNoDisponible) || strings.Contains(err.Error(), "Antonio") {
		t.Fatalf("fallo inseguro: %v", err)
	}
	otra, derivador, _ := fuenteNombrePropioPrueba(t, nil)
	recuperador := otra.recuperador.(recuperadorNombrePropioPrueba)
	recuperador.fallaHuella = strings.Repeat("cd", 32)
	otra.recuperador = recuperador
	propia, err := derivador.CandidatoRef(loteNombrePropioPrueba().Aceptadas[0].Identidad)
	if err != nil {
		t.Fatal(err)
	}
	if nombre, _, encontrado, err := otra.nombrePropio(t.Context(), propia, []string{"bolsa:otra", "bolsa:auxiliar"}); err != nil || !encontrado || nombre != "Lucía" {
		t.Fatalf("un acta ilegible cortó la búsqueda: %q %v %v", nombre, encontrado, err)
	}
	var sinEnlace fuentePersonalizacionB7
	if _, _, _, err := sinEnlace.NombrePropio(t.Context(), "can_sintetico", []string{"bolsa:auxiliar"}); !errors.Is(err, errNombrePropioMiBolsaNoDisponible) {
		t.Fatalf("sin fuente enlazada: %v", err)
	}
	sinDerivador := &fuenteConstituidaRRHHDesarrollo{repositorio: fuente.repositorio, recuperador: fuente.recuperador}
	if _, _, _, err := sinDerivador.nombrePropio(t.Context(), "can_sintetico", []string{"bolsa:auxiliar"}); !errors.Is(err, errNombrePropioMiBolsaNoDisponible) {
		t.Fatalf("sin derivador: %v", err)
	}
}
