package postgres

import (
	"bytes"
	"testing"
)

func TestCanonAltaCopiaDatosPuestoPeticionAntesDeNecesidad(t *testing.T) {
	evidencia, _ := evidenciaConfirmacionPostgreSQLPrueba(t)
	evidencia.Expediente.Solicitud.JornadaMinutos = 2250
	evidencia.Expediente.Solicitud.NumeroPersonas = 2
	evidencia.Expediente.Solicitud.PuestoSolicitado = "Administrativo C2"
	v2, _, err := canonEfectoAlta(evidencia.Expediente, evidencia.Candidatura)
	if err != nil || !bytes.Contains(v2, []byte(`"observaciones":"","jornada_minutos":2250,"numero_personas":2,"puesto_solicitado":"Administrativo C2"}`)) {
		t.Fatalf("alta v2 pierde datos del centro: %v", err)
	}
	n := evidenciaAltaConNecesidad(t)
	evidencia.Expediente.Solicitud.Necesidad = &n
	v3, _, err := canonEfectoAlta(evidencia.Expediente, evidencia.Candidatura)
	if err != nil || !bytes.Contains(v3, []byte(`"observaciones":"","jornada_minutos":2250,"numero_personas":2,"puesto_solicitado":"Administrativo C2","necesidad":{`)) {
		t.Fatalf("alta v3 pierde datos del centro o altera su orden: %v", err)
	}
}
