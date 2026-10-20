package ports

import (
	"errors"
	"testing"
)

// CT243: el grupo y la entrada de retención del análisis viajan fuera del
// recibo. Solo se aceptan juntos, válidos y con análisis; nunca alteran el
// análisis original ni entran en su huella.
func TestAnalisisConGrupoYEntradaRC(t *testing.T) {
	texto := func(v string) *string { return &v }
	original := &AnalisisOperativoRRHH{CategoriaRef: "categoria:a1", ResultadoRC: "validada"}

	con, err := original.ConGrupoYEntradaRC(texto("A1"), texto("rc:desarrollo:001"))
	if err != nil || con.GrupoSubgrupo != "A1" || con.EntradaRCRef != "rc:desarrollo:001" {
		t.Fatalf("análisis con grupo = %+v, %v", con, err)
	}
	if original.GrupoSubgrupo != "" || original.EntradaRCRef != "" || con == original {
		t.Fatal("el análisis original no debe mutar")
	}

	var sinAnalisis *AnalisisOperativoRRHH
	if a, err := sinAnalisis.ConGrupoYEntradaRC(nil, nil); err != nil || a != nil {
		t.Fatalf("sin análisis = %v, %v", a, err)
	}
	rechazos := []struct {
		nombre    string
		analisis  *AnalisisOperativoRRHH
		grupo, rc *string
	}{
		{"datos sin análisis", nil, texto("A1"), nil},
		{"entrada sin análisis", nil, nil, texto("rc:desarrollo:001")},
		{"falta grupo", original, nil, texto("rc:desarrollo:001")},
		{"falta entrada", original, texto("A1"), nil},
		{"grupo inválido", original, texto("A1<script>"), texto("rc:desarrollo:001")},
		{"entrada inválida", original, texto("A1"), texto("rc declarado con espacios")},
	}
	for _, caso := range rechazos {
		if _, err := caso.analisis.ConGrupoYEntradaRC(caso.grupo, caso.rc); !errors.Is(
			err, ErrResultadoConsultaRRHHNoConfiable,
		) {
			t.Errorf("%s: err = %v", caso.nombre, err)
		}
	}
}
