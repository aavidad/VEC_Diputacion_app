package domain

import (
	"strings"
	"testing"
)

func originalC8Prueba() (string, string, string, OriginalJustificacionC8V2) {
	solicitud := "permiso:cronos:solicitud:abcdefgh"
	persona := "per_" + strings.Repeat("A", 22)
	empleado := "emp_" + strings.Repeat("B", 22)
	return solicitud, persona, empleado, OriginalJustificacionC8V2{
		DocumentoID: "ref:" + strings.Repeat("a", 64), DocumentoVersion: 1,
		HuellaSHA256Bytes: strings.Repeat("b", 64), ReciboDocumentoRef: "recibo:documentos:abc",
		RecursoRef: solicitud, PersonaTitularRef: persona, PersonaAportanteRef: persona,
		CustodioID: "almacen.local", ModuloID: "cronos", TipoRecurso: "justificacion",
		Finalidad: "custodiar_justificacion_cronos", EstadoAdmision: "admitido",
	}
}

func TestOriginalC8V2RechazaDocumentoAjenoYOtraVersion(t *testing.T) {
	solicitud, persona, empleado, original := originalC8Prueba()
	for nombre, cambiar := range map[string]func(*OriginalJustificacionC8V2){
		"recurso":   func(o *OriginalJustificacionC8V2) { o.RecursoRef = "permiso:cronos:solicitud:otra1234" },
		"titular":   func(o *OriginalJustificacionC8V2) { o.PersonaTitularRef = "per_" + strings.Repeat("Z", 22) },
		"aportante": func(o *OriginalJustificacionC8V2) { o.PersonaAportanteRef = "per_" + strings.Repeat("Z", 22) },
		"huella":    func(o *OriginalJustificacionC8V2) { o.HuellaSHA256Bytes = "" },
		"recibo":    func(o *OriginalJustificacionC8V2) { o.ReciboDocumentoRef = "" },
		"estado":    func(o *OriginalJustificacionC8V2) { o.EstadoAdmision = "recibido" },
	} {
		t.Run(nombre, func(t *testing.T) {
			otro := original
			cambiar(&otro)
			if _, err := PrepararAnexoOriginalC8V2(solicitud, empleado, persona, otro, nil, 0); err == nil {
				t.Fatal("aceptó un original no acreditado para esta solicitud y persona")
			}
		})
	}
	primero, err := PrepararAnexoOriginalC8V2(solicitud, empleado, persona, original, nil, 0)
	if err != nil || primero.Version != 1 || primero.Estado != JustificacionPendiente {
		t.Fatalf("anexo admitido: %+v, %v", primero, err)
	}
	if _, err := PrepararAnexoOriginalC8V2(solicitud, empleado, persona, original, &primero, 1); err != ErrJustificacionConflicto {
		t.Fatalf("duplicó el mismo original y versión: %v", err)
	}
}

func TestRevisionOriginalC8V2ConservaIdentidadEHistoria(t *testing.T) {
	solicitud, persona, empleado, original := originalC8Prueba()
	pendiente, err := PrepararAnexoOriginalC8V2(solicitud, empleado, persona, original, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	revisada, err := PrepararRevisionOriginalC8V2(pendiente, 1, JustificacionAceptada, "motivo:documentacion:conforme")
	if err != nil || revisada.Version != 2 || revisada.Estado != JustificacionAceptada ||
		revisada.Original != original || pendiente.Estado != JustificacionPendiente {
		t.Fatalf("revisión alteró original o historia: %+v, %v", revisada, err)
	}
	if _, err := PrepararRevisionOriginalC8V2(revisada, 2, JustificacionRechazada, "motivo:documentacion:conforme"); err != ErrJustificacionConflicto {
		t.Fatalf("revisó otra vez un estado cerrado: %v", err)
	}
}
