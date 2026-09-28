package plantillascatalogo

import (
	"strings"
	"testing"
)

func TestSolicitudDocumentalFijaOperacionYExpediente(t *testing.T) {
	base := SolicitudDocumental{Operacion: "listar", OrganizacionRef: "organizacion:desarrollo:dipgra", ExpedienteRef: "expediente:ct:0001", VersionObservada: 7, ConsultaHuellaSHA256: strings.Repeat("a", 64)}
	if err := base.Validar(); err != nil {
		t.Fatal(err)
	}
	c := base
	c.Operacion = "descargar"
	c.Tipo = "nuevo_tipo"
	c.Formato = "docx"
	if err := c.Validar(); err != nil {
		t.Fatal(err)
	}
	for nombre, mutar := range map[string]func(*SolicitudDocumental){
		"sin organización":     func(s *SolicitudDocumental) { s.OrganizacionRef = "" },
		"sin expediente":       func(s *SolicitudDocumental) { s.ExpedienteRef = "" },
		"sin version":          func(s *SolicitudDocumental) { s.VersionObservada = 0 },
		"sin recibo detalle":   func(s *SolicitudDocumental) { s.ConsultaHuellaSHA256 = "" },
		"descarga sin formato": func(s *SolicitudDocumental) { s.Operacion = "descargar"; s.Tipo = "nuevo_tipo"; s.Formato = "" },
		"lista con tipo":       func(s *SolicitudDocumental) { s.Tipo = "nuevo_tipo" },
	} {
		t.Run(nombre, func(t *testing.T) {
			s := base
			mutar(&s)
			if s.Validar() == nil {
				t.Fatal("material documental inválido admitido")
			}
		})
	}
}
