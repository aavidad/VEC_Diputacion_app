package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func solicitudDescargaPrueba() SolicitudDescargaBorradorRRHH {
	return SolicitudDescargaBorradorRRHH{
		ExpedienteRef: "expediente:ct:" + strings.Repeat("a", 64), VersionExpediente: 9,
		Tipo: BorradorResolucion, Formato: FormatoBorradorRRHHPDF, DocumentoSHA256: strings.Repeat("b", 64),
		TamanoBytes: 29770, ConsultaHuellaSHA256: strings.Repeat("c", 64),
	}
}

// La huella del contexto del recurso debe coincidir byte a byte con la que
// recalcula CT177: mismo canon y mismo JSON de ámbitos y atributos.
func TestRecursoDescargaBorradorCoincideConCT177(t *testing.T) {
	s := solicitudDescargaPrueba()
	a := AlcanceDescargaBorradorRRHH{OrganizacionRef: "organizacion:desarrollo:dipgra", ClaseAmbito: AmbitoOrganizacionRRHH, AmbitoRef: "organizacion:desarrollo:dipgra"}
	r, err := RecursoDescargaBorradorRRHH(s, a)
	if err != nil {
		t.Fatal(err)
	}
	canon := DominioHuellaDescargaBorradorRRHH + "\n" + s.ExpedienteRef + "\n9\nresolucion\npdf\n" + s.DocumentoSHA256 + "\n29770\n" + s.ConsultaHuellaSHA256
	hc := sha256.Sum256([]byte(canon))
	contexto := `{"ambitos":{"ambito_ref":"` + a.AmbitoRef + `","clase_ambito":"` + string(a.ClaseAmbito) +
		`","organizacion_ref":"` + a.OrganizacionRef + `"},"atributos":{"consulta_dominio":"` + DominioHuellaDescargaBorradorRRHH +
		`","consulta_huella_sha256":"` + hex.EncodeToString(hc[:]) + `"}}`
	esperada := sha256.Sum256([]byte(contexto))
	huella, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil || huella != hex.EncodeToString(esperada[:]) {
		t.Fatalf("huella de contexto distinta de CT177: %s", huella)
	}
	if r.Referencia != s.ExpedienteRef || r.Tipo != TipoRecursoExpediente || r.ModuloID != ModuloContratacion {
		t.Fatalf("recurso inesperado: %+v", r)
	}
	// Otro archivo, otro formato u otro tipo cambian la huella.
	for _, cambiar := range []func(*SolicitudDescargaBorradorRRHH){
		func(x *SolicitudDescargaBorradorRRHH) { x.DocumentoSHA256 = strings.Repeat("d", 64) },
		func(x *SolicitudDescargaBorradorRRHH) { x.Formato = FormatoBorradorRRHHDOCX },
		func(x *SolicitudDescargaBorradorRRHH) { x.Tipo = BorradorDiligencia },
		func(x *SolicitudDescargaBorradorRRHH) { x.TamanoBytes++ },
	} {
		otra := s
		cambiar(&otra)
		r2, err := RecursoDescargaBorradorRRHH(otra, a)
		h2, _ := r2.HuellaContextoAutorizacionSHA256()
		if err != nil || h2 == huella {
			t.Fatalf("la huella no liga el documento: err=%v", err)
		}
	}
}

func TestSolicitudDescargaBorradorRechazaFormasAbiertas(t *testing.T) {
	for nombre, cambiar := range map[string]func(*SolicitudDescargaBorradorRRHH){
		"expediente sin prefijo": func(x *SolicitudDescargaBorradorRRHH) { x.ExpedienteRef = "otro:ct:1" },
		"expediente con salto":   func(x *SolicitudDescargaBorradorRRHH) { x.ExpedienteRef = "expediente:a\nb" },
		"versión cero":           func(x *SolicitudDescargaBorradorRRHH) { x.VersionExpediente = 0 },
		"tipo con mayúsculas":    func(x *SolicitudDescargaBorradorRRHH) { x.Tipo = "Resolucion" },
		"formato ajeno":          func(x *SolicitudDescargaBorradorRRHH) { x.Formato = "odt" },
		"huella corta":           func(x *SolicitudDescargaBorradorRRHH) { x.DocumentoSHA256 = "abc" },
		"tamaño cero":            func(x *SolicitudDescargaBorradorRRHH) { x.TamanoBytes = 0 },
		"consulta sin huella":    func(x *SolicitudDescargaBorradorRRHH) { x.ConsultaHuellaSHA256 = "" },
	} {
		s := solicitudDescargaPrueba()
		cambiar(&s)
		if s.Validar() == nil {
			t.Fatalf("%s admitido", nombre)
		}
		if _, err := CanonDescargaBorradorRRHH(s); err == nil {
			t.Fatalf("%s produjo canon", nombre)
		}
	}
	if _, err := RecursoDescargaBorradorRRHH(solicitudDescargaPrueba(), AlcanceDescargaBorradorRRHH{}); err == nil {
		t.Fatal("alcance vacío admitido")
	}
}
