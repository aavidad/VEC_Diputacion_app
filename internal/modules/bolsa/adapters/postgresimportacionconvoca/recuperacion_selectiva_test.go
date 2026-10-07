package postgresimportacionconvoca

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	aplicacion "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

type protectorSelectivoPrueba struct {
	base     *protectorAEADIntegracion
	llamadas int
	numeros  []int
}

func (p *protectorSelectivoPrueba) ProtegerStaging(ctx context.Context, s SolicitudProteccionStaging) (ResultadoProteccionStaging, error) {
	return p.base.ProtegerStaging(ctx, s)
}

func (p *protectorSelectivoPrueba) RecuperarStaging(ctx context.Context, s SolicitudRecuperacionStaging) ([]dominio.FilaAceptada, error) {
	p.llamadas++
	for _, fila := range s.Filas {
		p.numeros = append(p.numeros, fila.Numero)
	}
	return p.base.RecuperarStaging(ctx, s)
}

func cargaSelectivaPrueba(t *testing.T, lote dominio.LoteValidado, filas []FilaStagingProtegida, staging string) []byte {
	t.Helper()
	actaJSON, err := serializarActa(lote.Acta)
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(actaJSON)
	var acta actaPostgreSQL
	if err := json.Unmarshal(actaJSON, &acta); err != nil {
		t.Fatal(err)
	}
	filasJSON, err := serializarFilasProtegidas(filas)
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(filasJSON)
	datos := struct {
		Estado estadoPostgreSQL `json:"estado"`
		Filas  json.RawMessage  `json:"filas"`
	}{Estado: estadoPostgreSQL{
		Acta: acta, EstadoConciliacion: string(aplicacion.EstadoConciliacionPendiente),
		EstadoStaging: staging, PoliticaRetencionRef: "politica:retencion:selectiva",
		PoliticaRetencionVersion: 1,
		ConservarStagingHasta:    formatearInstante(lote.Acta.RegistradaEn.Add(24 * time.Hour)),
		Version:                  1,
	}, Filas: filasJSON}
	crudo, err := json.Marshal(datos)
	if err != nil {
		t.Fatal(err)
	}
	return crudo
}

func alterarFilaSelectivaPrueba(t *testing.T, crudo []byte, campo string, valor any) []byte {
	t.Helper()
	var datos map[string]any
	if err := json.Unmarshal(crudo, &datos); err != nil {
		t.Fatal(err)
	}
	filas := datos["filas"].([]any)
	filas[0].(map[string]any)[campo] = valor
	mutado, err := json.Marshal(datos)
	if err != nil {
		t.Fatal(err)
	}
	return mutado
}

func TestRecuperacionSelectivaDescifraSoloPermutacionSolicitada(t *testing.T) {
	ahora := time.Date(2026, 10, 7, 10, 0, 0, 0, time.UTC)
	lote := loteIntegracion(huellaIntegracion("a"), ahora, 3)
	lote.Aceptadas[1].Identidad.Documento = "***0002**"
	lote.Aceptadas[2].Identidad.Documento = "***0003**"
	if err := lote.Validar(); err != nil {
		t.Fatal(err)
	}
	protector := &protectorSelectivoPrueba{base: nuevoProtectorAEADIntegracion()}
	protegido, err := protector.ProtegerStaging(t.Context(), SolicitudProteccionStaging{
		ImportacionRef: lote.Acta.ImportacionRef, HuellaFicheroSHA256: lote.Acta.HuellaFicheroSHA256,
		Esquema: lote.Acta.Esquema, Filas: lote.Aceptadas,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer borrarFilasProtegidas(protegido.Filas)
	crudo := cargaSelectivaPrueba(t, lote, []FilaStagingProtegida{protegido.Filas[0], protegido.Filas[2]},
		string(aplicacion.EstadoStagingDisponible))
	filas, err := (&RepositorioRecuperacionPostgreSQL{protector: protector}).DescifrarFilasSeleccionadas(
		t.Context(), crudo, lote.Acta.HuellaFicheroSHA256, lote.Acta.CategoriaRef, []int{4, 2})
	if err != nil || len(filas) != 2 || filas[0].Numero != 4 || filas[1].Numero != 2 ||
		protector.llamadas != 1 || len(protector.numeros) != 2 ||
		protector.numeros[0] != 2 || protector.numeros[1] != 4 {
		t.Fatalf("selección alterada: filas=%v llamadas=%d numeros=%v err=%v",
			filas, protector.llamadas, protector.numeros, err)
	}
	if filas[0].Identidad.Documento != lote.Aceptadas[2].Identidad.Documento ||
		filas[1].Identidad.Documento != lote.Aceptadas[0].Identidad.Documento {
		t.Fatal("AAD recuperó datos de otra fila")
	}
}

func TestRecuperacionSelectivaRechazaHashMetadataAADYExpurgo(t *testing.T) {
	ahora := time.Date(2026, 10, 7, 10, 0, 0, 0, time.UTC)
	lote := loteIntegracion(huellaIntegracion("b"), ahora, 1)
	protector := &protectorSelectivoPrueba{base: nuevoProtectorAEADIntegracion()}
	protegido, err := protector.ProtegerStaging(t.Context(), SolicitudProteccionStaging{
		ImportacionRef: lote.Acta.ImportacionRef, HuellaFicheroSHA256: lote.Acta.HuellaFicheroSHA256,
		Esquema: lote.Acta.Esquema, Filas: lote.Aceptadas,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer borrarFilasProtegidas(protegido.Filas)
	crudo := cargaSelectivaPrueba(t, lote, protegido.Filas, string(aplicacion.EstadoStagingDisponible))
	repo := &RepositorioRecuperacionPostgreSQL{protector: protector}
	casos := []struct {
		nombre  string
		crudo   []byte
		numeros []int
		huella  string
		estado  error
	}{
		{"hash", alterarFilaSelectivaPrueba(t, crudo, "huella_contenido_cifrado_sha256", strings.Repeat("0", 64)), []int{2}, lote.Acta.HuellaFicheroSHA256, ErrResultadoNoConfiable},
		{"metadata", alterarFilaSelectivaPrueba(t, crudo, "clave_ref", "clave no válida"), []int{2}, lote.Acta.HuellaFicheroSHA256, ErrResultadoNoConfiable},
		{"AAD", alterarFilaSelectivaPrueba(t, crudo, "numero", 3), []int{3}, lote.Acta.HuellaFicheroSHA256, ErrMaterialNoConfiable},
		{"huella acta", crudo, []int{2}, strings.Repeat("c", 64), ErrResultadoNoConfiable},
		{"fila extra", crudo, []int{2, 3}, lote.Acta.HuellaFicheroSHA256, ErrResultadoNoConfiable},
		{"expurgo", cargaSelectivaPrueba(t, lote, protegido.Filas, string(aplicacion.EstadoStagingExpurgado)), []int{2}, lote.Acta.HuellaFicheroSHA256, aplicacion.ErrStagingExpurgado},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			previas := protector.llamadas
			filas, err := repo.DescifrarFilasSeleccionadas(t.Context(), caso.crudo, caso.huella, lote.Acta.CategoriaRef, caso.numeros)
			if filas != nil || !errors.Is(err, caso.estado) ||
				(caso.nombre != "AAD" && protector.llamadas != previas) ||
				(caso.nombre == "AAD" && protector.llamadas != previas+1) ||
				strings.Contains(err.Error(), lote.Aceptadas[0].Identidad.Documento) {
				t.Fatalf("rechazo selectivo: filas=%v llamadas=%d error=%v", filas, protector.llamadas, err)
			}
		})
	}
}

func TestRecuperacionSelectivaLimitesYCancelacion(t *testing.T) {
	for _, numeros := range [][]int{nil, {1}, {2, 2}, {0}, {2, int(1 << 31)}, make([]int, 102)} {
		if _, err := ordenarNumerosSeleccionados(numeros); !errors.Is(err, ErrLoteNoConfiable) {
			t.Fatalf("números inválidos admitidos: len=%d err=%v", len(numeros), err)
		}
	}
	ordenados := make([]int, 101)
	for i := range ordenados {
		ordenados[i] = i + 2
	}
	if salida, err := ordenarNumerosSeleccionados(ordenados); err != nil || len(salida) != 101 {
		t.Fatalf("límite de página rechazado: len=%d err=%v", len(salida), err)
	}
	ctx, cancelar := context.WithCancel(t.Context())
	cancelar()
	protector := &protectorSelectivoPrueba{base: nuevoProtectorAEADIntegracion()}
	filas, err := (&RepositorioRecuperacionPostgreSQL{protector: protector}).DescifrarFilasSeleccionadas(ctx, []byte(`{}`), "", "", []int{2})
	if filas != nil || !errors.Is(err, context.Canceled) || protector.llamadas != 0 {
		t.Fatalf("cancelación descifró filas: filas=%v llamadas=%d err=%v", filas, protector.llamadas, err)
	}
}
