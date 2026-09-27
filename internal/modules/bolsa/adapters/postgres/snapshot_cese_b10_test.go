package postgres

import (
	"bytes"
	"testing"
	"time"
)

func TestCotejarMaterialCeseB10RechazaEstadoYDocumentoNoMinimizado(t *testing.T) {
	corte := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	fila := filaSnapshotCeseB10{BolsaRef: "bolsa:revision", CategoriaRef: "categoria:revision",
		ActaRef: "acta:revision", VigenteDesde: corte.Add(-time.Hour), Total: 1,
		TipoLista: "rotatoria", ParticipacionRef: "participacion:revision", FilaNumero: 1,
		Orden: 1, EstadoEfectivo: "disponible_desde", EsOrigen: true}
	bolsa := BolsaFuenteCeseB10{Ref: fila.BolsaRef, Categoria: "Categoría sintética",
		CategoriaClave: "categoria", Grupos: []string{"C1"}, TipoLista: fila.TipoLista,
		VigenteDesde: fila.VigenteDesde, Total: 1,
		Participaciones: []ParticipacionFuenteCeseB10{{Ref: fila.ParticipacionRef,
			Orden: 1, Documento: "***1234**", EstadoEfectivo: "no_disponible"}}}
	material, err := serializarBolsasCeseB10(corte, []BolsaFuenteCeseB10{bolsa})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(material, []byte(`"generado_en":"2026-09-28T12:00:00.000000Z"`)) {
		t.Fatal("corte B10 sin precisión microsegundo contractual")
	}
	if _, err := cotejarMaterialBolsasCeseB10(corte, []filaSnapshotCeseB10{fila}, material); err != nil {
		t.Fatalf("estado B45 válido rechazado: %v", err)
	}
	bolsa.Participaciones[0].EstadoEfectivo = "disponible"
	material, _ = serializarBolsasCeseB10(corte, []BolsaFuenteCeseB10{bolsa})
	if _, err := cotejarMaterialBolsasCeseB10(corte, []filaSnapshotCeseB10{fila}, material); err == nil {
		t.Fatal("el proveedor sustituyó el estado B45")
	}
	bolsa.Participaciones[0].EstadoEfectivo = "no_disponible"
	bolsa.Participaciones[0].Documento = "12345678X"
	material, _ = serializarBolsasCeseB10(corte, []BolsaFuenteCeseB10{bolsa})
	if _, err := cotejarMaterialBolsasCeseB10(corte, []filaSnapshotCeseB10{fila}, material); err == nil {
		t.Fatal("el proveedor expuso un documento completo")
	}
}

func TestCapturadorCeseB10RequiereProveedores(t *testing.T) {
	if _, err := NuevoCapturadorCeseB10PostgreSQL(nil, nil, nil, nil); err == nil {
		t.Fatal("capturador sin proveedor aceptado")
	}
}
