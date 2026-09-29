package constitucion

import (
	"context"
	"strings"
	"testing"
	"time"

	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// Datos sintéticos: ninguna fila procede de una lista real.

func filaIdentidad(numero int, doc, apellido1, apellido2, nombre, total string) importacion.FilaAceptada {
	return importacion.FilaAceptada{
		Numero: numero, Esquema: importacion.EsquemaResumenPersona,
		Identidad: importacion.IdentidadEnmascarada{Documento: doc, PrimerApellido: apellido1, SegundoApellido: apellido2, Nombre: nombre},
		Turno:     "Libre",
		Resumen:   &importacion.ResumenPersona{Experiencia: "1", Formacion: "2", Total: total},
	}
}

func loteSintetico(filas ...importacion.FilaAceptada) importacion.LoteValidado {
	huella := strings.Repeat("ab", 32)
	categoria := "categoria:rpt:auxiliar"
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

// Filas 2 y 3: dos personas con el mismo DNI enmascarado y el mismo nombre,
// escrito igual. Filas 5 y 6: coinciden tras normalizar tildes y mayúsculas.
// Filas 4 y 7 no coinciden con nadie.
func loteConColisiones() importacion.LoteValidado {
	return loteSintetico(
		filaIdentidad(2, "***4821**", "Moreno", "Castillo", "Lucía", "3"),
		filaIdentidad(3, "***4821**", "Moreno", "Castillo", "Lucía", "3"),
		filaIdentidad(4, "***4821**", "Moreno", "Castillo", "Lucas", "3"),
		filaIdentidad(5, "***7302**", "Peña", "Ibáñez", "José Antonio", "3"),
		filaIdentidad(6, "***7302**", "PENA", "IBANEZ", "JOSE  ANTONIO", "3"),
		filaIdentidad(7, "***1190**", "Reyes", "Álvarez", "Antonio", "3"),
	)
}

func numerosPendientes(pendientes []ports.FilaPendienteRevision) []int {
	numeros := make([]int, len(pendientes))
	for i, p := range pendientes {
		if p.Motivo != ports.MotivoRevisionIdentidadAmbigua {
			return nil
		}
		numeros[i] = p.FilaNumero
	}
	return numeros
}

func igualesEnteros(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDerivarFilasVinculoNoFundePersonasConMismaIdentidadEnmascarada(t *testing.T) {
	lote := loteConColisiones()
	if err := lote.Validar(); err != nil {
		t.Fatalf("lote sintético inválido: %v", err)
	}
	d := derivadorPrueba(t)
	// Precondición: las filas 2 y 3 derivan hoy la misma referencia.
	refA, _ := d.CandidatoRef(lote.Aceptadas[0].Identidad)
	refB, _ := d.CandidatoRef(lote.Aceptadas[1].Identidad)
	if refA != refB {
		t.Fatal("la prueba necesita dos filas que deriven la misma referencia")
	}
	filas, pendientes, err := DerivarFilasVinculo(lote, d)
	if err != nil {
		t.Fatalf("derivar: %v", err)
	}
	if !igualesEnteros(numerosPendientes(pendientes), []int{2, 3, 5, 6}) {
		t.Fatalf("pendientes inesperadas: %+v", pendientes)
	}
	if len(filas) != 2 || filas[0].FilaNumero != 4 || filas[1].FilaNumero != 7 {
		t.Fatalf("solo deben vincularse las filas sin coincidencia: %+v", filas)
	}
	for _, f := range filas {
		if f.CandidatoRef == refA {
			t.Fatalf("una fila ambigua se ha vinculado: %+v", f)
		}
	}
}

func TestDerivarFilasVinculoSinColisionesVinculaTodas(t *testing.T) {
	lote := loteSintetico(
		filaIdentidad(2, "***4821**", "Moreno", "Castillo", "Lucía", "3"),
		filaIdentidad(3, "***4822**", "Moreno", "Castillo", "Lucía", "3"),
	)
	filas, pendientes, err := DerivarFilasVinculo(lote, derivadorPrueba(t))
	if err != nil || len(pendientes) != 0 || len(filas) != 2 {
		t.Fatalf("filas=%+v pendientes=%+v err=%v", filas, pendientes, err)
	}
}

type repositorioContador struct {
	repositorioPrueba
	llamadasVinculos int
}

func (r *repositorioContador) RegistrarVinculos(ctx context.Context, actaRef string, vinculos []ports.VinculoCandidato, ahora time.Time) (ports.ReciboVinculosCandidato, error) {
	r.llamadasVinculos++
	return r.repositorioPrueba.RegistrarVinculos(ctx, actaRef, vinculos, ahora)
}

func constituirSintetico(t *testing.T, lote importacion.LoteValidado) (ports.ReciboConstitucion, *repositorioContador) {
	t.Helper()
	repo := &repositorioContador{}
	servicio, err := NuevoServicio(recuperadorPrueba{lote}, repo, derivadorPrueba(t), func() time.Time { return time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	recibo, err := servicio.Constituir(context.Background(), Solicitud{HuellaFicheroSHA256: lote.Acta.HuellaFicheroSHA256, CategoriaRef: lote.Acta.CategoriaRef, ActorRef: "actor:rrhh:pruebas"})
	if err != nil {
		t.Fatalf("constituir: %v", err)
	}
	return recibo, repo
}

func TestConstituirConservaPuestosYDejaPendientesLasIdentidadesAmbiguas(t *testing.T) {
	lote := loteConColisiones()
	recibo, repo := constituirSintetico(t, lote)
	c := repo.guardada
	if err := c.Instantanea.Validar(); err != nil {
		t.Fatalf("instantánea inválida: %v", err)
	}
	if len(c.Instantanea.Entradas) != len(lote.Aceptadas) {
		t.Fatalf("todas las filas deben conservar su puesto: %d de %d", len(c.Instantanea.Entradas), len(lote.Aceptadas))
	}
	if !igualesEnteros(numerosPendientes(recibo.PendientesRevision), []int{2, 3, 5, 6}) {
		t.Fatalf("el recibo debe listar las pendientes: %+v", recibo.PendientesRevision)
	}
	if len(repo.vinculos) != 2 || recibo.Vinculos.Nuevos != 2 {
		t.Fatalf("solo deben vincularse las filas 4 y 7: %+v", repo.vinculos)
	}
	participacionDeFila := make(map[int]string, len(c.Entradas))
	for _, e := range c.Entradas {
		participacionDeFila[e.FilaNumero] = e.ParticipacionRef
	}
	candidatos := make(map[string]struct{}, len(repo.vinculos))
	for _, v := range repo.vinculos {
		if _, repetido := candidatos[v.CandidatoRef]; repetido {
			t.Fatalf("dos participaciones vinculadas a la misma persona: %+v", repo.vinculos)
		}
		candidatos[v.CandidatoRef] = struct{}{}
		for _, fila := range []int{2, 3, 5, 6} {
			if v.ParticipacionRef == participacionDeFila[fila] {
				t.Fatalf("la fila ambigua %d se ha vinculado: %+v", fila, v)
			}
		}
	}
	// La fila 7 no coincide con nadie: su referencia de sujeto no cambia.
	historico := "sujeto:convoca:" + sufijoOpaco("***1190**|Reyes|Álvarez|Antonio")
	encontrado := false
	for _, e := range c.Instantanea.Entradas {
		if e.Participacion.ParticipacionRef == participacionDeFila[7] {
			encontrado = e.Participacion.SujetoRef == historico
		}
	}
	if !encontrado {
		t.Fatal("la referencia de sujeto de una fila sin coincidencia ha cambiado")
	}
}

func TestConstituirSinFilasVinculablesNoRegistraVinculos(t *testing.T) {
	lote := loteSintetico(
		filaIdentidad(2, "***4821**", "Moreno", "Castillo", "Lucía", "3"),
		filaIdentidad(3, "***4821**", "Moreno", "Castillo", "Lucía", "3"),
	)
	recibo, repo := constituirSintetico(t, lote)
	if repo.llamadasVinculos != 0 || recibo.Vinculos != (ports.ReciboVinculosCandidato{}) {
		t.Fatalf("no debe registrarse ningún vínculo: llamadas=%d recibo=%+v", repo.llamadasVinculos, recibo.Vinculos)
	}
	if !igualesEnteros(numerosPendientes(recibo.PendientesRevision), []int{2, 3}) {
		t.Fatalf("pendientes inesperadas: %+v", recibo.PendientesRevision)
	}
	if len(repo.guardada.Instantanea.Entradas) != 2 {
		t.Fatal("la bolsa debe conservar los dos puestos")
	}
}
