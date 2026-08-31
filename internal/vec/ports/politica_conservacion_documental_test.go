package ports

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

type vinculosPoliticaConservacionDocumentalPrueba struct {
	procedimientoRef   string
	serieDocumentalRef string
	tipoDocumentalRef  string
	expedienteRef      string
	politicaRef        string
	versionPolitica    uint64
	huellaPolitica     []byte
	baseJuridicaRef    string
	vigenteDesde       time.Time
	vigenteHasta       time.Time
}

func nuevosVinculosPoliticaConservacionDocumentalPrueba() vinculosPoliticaConservacionDocumentalPrueba {
	return vinculosPoliticaConservacionDocumentalPrueba{
		procedimientoRef:   referenciaPoliticaConservacionDocumentalPrueba('a'),
		serieDocumentalRef: referenciaPoliticaConservacionDocumentalPrueba('b'),
		tipoDocumentalRef:  referenciaPoliticaConservacionDocumentalPrueba('c'),
		expedienteRef:      referenciaPoliticaConservacionDocumentalPrueba('d'),
		politicaRef:        referenciaPoliticaConservacionDocumentalPrueba('e'),
		versionPolitica:    7,
		huellaPolitica:     bytes.Repeat([]byte{0x5a}, 32),
		baseJuridicaRef:    referenciaPoliticaConservacionDocumentalPrueba('f'),
		vigenteDesde:       time.Date(2026, 9, 1, 0, 0, 0, 123_456_000, time.UTC),
		vigenteHasta:       time.Date(2030, 9, 1, 0, 0, 0, 123_456_000, time.UTC),
	}
}

func (v vinculosPoliticaConservacionDocumentalPrueba) construir(
	t *testing.T,
) SolicitudPoliticaConservacionDocumental {
	t.Helper()
	solicitud, err := NuevaSolicitudPoliticaConservacionDocumental(
		v.procedimientoRef, v.serieDocumentalRef, v.tipoDocumentalRef, v.expedienteRef,
		v.politicaRef, v.versionPolitica, v.huellaPolitica, v.baseJuridicaRef,
		v.vigenteDesde, v.vigenteHasta,
	)
	if err != nil {
		t.Fatalf("crear solicitud sintetica: %v", err)
	}
	return solicitud
}

func nuevaPoliticaConservacionDocumentalPrueba(
	t *testing.T,
	solicitud SolicitudPoliticaConservacionDocumental,
	proteccion ProteccionPoliticaConservacionDocumental,
	estado EstadoPoliticaConservacionDocumental,
) PoliticaConservacionDocumental {
	t.Helper()
	bloqueoRef := ""
	if proteccion == ProteccionPoliticaConservacionDocumentalBloqueada {
		bloqueoRef = referenciaPoliticaConservacionDocumentalPrueba('1')
	}
	retiradaEn := time.Time{}
	if estado == EstadoPoliticaConservacionDocumentalRetirada {
		retiradaEn = time.Date(2028, 3, 1, 0, 0, 0, 0, time.UTC)
	}
	politica, err := NuevaPoliticaConservacionDocumental(
		solicitud,
		time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC),
		proteccion,
		bloqueoRef,
		estado,
		retiradaEn,
	)
	if err != nil {
		t.Fatalf("crear politica sintetica: %v", err)
	}
	return politica
}

func referenciaPoliticaConservacionDocumentalPrueba(caracter byte) string {
	return "ref:" + strings.Repeat(string(caracter), 64)
}

func TestPoliticaConservacionDocumentalValoresNeutralesValidosYBloqueoPrevalece(t *testing.T) {
	vinculos := nuevosVinculosPoliticaConservacionDocumentalPrueba()
	solicitud := vinculos.construir(t)
	politica := nuevaPoliticaConservacionDocumentalPrueba(
		t, solicitud, ProteccionPoliticaConservacionDocumentalBloqueada,
		EstadoPoliticaConservacionDocumentalAprobada,
	)
	ahora := time.Date(2028, 2, 1, 12, 0, 0, 0, time.UTC)
	resultado, err := NuevoResultadoPoliticaConservacionDocumental(politica, solicitud, ahora)
	if err != nil || resultado.Validar() != nil {
		t.Fatalf("resultado neutral exacto denegado: resultado=%v error=%v", resultado, err)
	}
	obtenida := resultado.Politica()
	if !resultado.ResueltaEn().Equal(ahora) ||
		obtenida.Proteccion() != ProteccionPoliticaConservacionDocumentalBloqueada ||
		obtenida.BloqueoRef() != referenciaPoliticaConservacionDocumentalPrueba('1') ||
		obtenida.Estado() != EstadoPoliticaConservacionDocumentalAprobada ||
		!obtenida.ConservacionHasta().Equal(time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("resultado no conservo el bloqueo prevalente: %#v", resultado)
	}
	if obtenida.Solicitud().PoliticaRef() != vinculos.politicaRef ||
		obtenida.Solicitud().VersionPolitica() != vinculos.versionPolitica ||
		obtenida.Solicitud().BaseJuridicaRef() != vinculos.baseJuridicaRef {
		t.Fatal("resultado desligado de la publicacion aprobada")
	}
	if strings.Contains(resultado.String(), vinculos.expedienteRef) ||
		strings.Contains(solicitud.GoString(), vinculos.politicaRef) {
		t.Fatal("representacion textual filtro referencias opacas")
	}
}

func TestPoliticaConservacionDocumentalRechazaCeroYValoresNoOpacos(t *testing.T) {
	base := nuevosVinculosPoliticaConservacionDocumentalPrueba()
	casos := []struct {
		nombre  string
		alterar func(*vinculosPoliticaConservacionDocumentalPrueba)
	}{
		{"procedimiento cero", func(v *vinculosPoliticaConservacionDocumentalPrueba) { v.procedimientoRef = "" }},
		{"serie cero", func(v *vinculosPoliticaConservacionDocumentalPrueba) { v.serieDocumentalRef = "" }},
		{"tipo cero", func(v *vinculosPoliticaConservacionDocumentalPrueba) { v.tipoDocumentalRef = "" }},
		{"expediente cero", func(v *vinculosPoliticaConservacionDocumentalPrueba) { v.expedienteRef = "" }},
		{"politica cero", func(v *vinculosPoliticaConservacionDocumentalPrueba) { v.politicaRef = "" }},
		{"version cero", func(v *vinculosPoliticaConservacionDocumentalPrueba) { v.versionPolitica = 0 }},
		{"huella cero", func(v *vinculosPoliticaConservacionDocumentalPrueba) { v.huellaPolitica = make([]byte, 32) }},
		{"base cero", func(v *vinculosPoliticaConservacionDocumentalPrueba) { v.baseJuridicaRef = "" }},
		{"desde cero", func(v *vinculosPoliticaConservacionDocumentalPrueba) { v.vigenteDesde = time.Time{} }},
		{"hasta cero", func(v *vinculosPoliticaConservacionDocumentalPrueba) { v.vigenteHasta = time.Time{} }},
		{"vigencia vacia", func(v *vinculosPoliticaConservacionDocumentalPrueba) { v.vigenteHasta = v.vigenteDesde }},
		{"referencia repetida", func(v *vinculosPoliticaConservacionDocumentalPrueba) { v.baseJuridicaRef = v.politicaRef }},
		{"dato personal", func(v *vinculosPoliticaConservacionDocumentalPrueba) { v.expedienteRef = "12345678Z" }},
		{"credencial", func(v *vinculosPoliticaConservacionDocumentalPrueba) { v.politicaRef = "Bearer secreto" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			vinculos := base
			vinculos.huellaPolitica = append([]byte(nil), base.huellaPolitica...)
			caso.alterar(&vinculos)
			if _, err := NuevaSolicitudPoliticaConservacionDocumental(
				vinculos.procedimientoRef, vinculos.serieDocumentalRef,
				vinculos.tipoDocumentalRef, vinculos.expedienteRef,
				vinculos.politicaRef, vinculos.versionPolitica, vinculos.huellaPolitica,
				vinculos.baseJuridicaRef, vinculos.vigenteDesde, vinculos.vigenteHasta,
			); !errors.Is(err, ErrSolicitudPoliticaConservacionDocumentalInvalida) {
				t.Fatalf("valor invalido aceptado: %v", err)
			}
		})
	}
	if (SolicitudPoliticaConservacionDocumental{}).Validar() == nil ||
		(PoliticaConservacionDocumental{}).Validar() == nil ||
		(ResultadoPoliticaConservacionDocumental{}).Validar() == nil {
		t.Fatal("un valor cero conservo validez")
	}
}

func TestPoliticaConservacionDocumentalRechazaEstadosIncoherentes(t *testing.T) {
	solicitud := nuevosVinculosPoliticaConservacionDocumentalPrueba().construir(t)
	conservacion := time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC)
	bloqueo := referenciaPoliticaConservacionDocumentalPrueba('1')
	retirada := time.Date(2028, 3, 1, 0, 0, 0, 0, time.UTC)
	casos := []struct {
		nombre     string
		hasta      time.Time
		proteccion ProteccionPoliticaConservacionDocumental
		bloqueoRef string
		estado     EstadoPoliticaConservacionDocumental
		retiradaEn time.Time
	}{
		{"conservacion cero", time.Time{}, ProteccionPoliticaConservacionDocumentalOrdinaria, "", EstadoPoliticaConservacionDocumentalAprobada, time.Time{}},
		{"proteccion cero", conservacion, "", "", EstadoPoliticaConservacionDocumentalAprobada, time.Time{}},
		{"bloqueo sin referencia", conservacion, ProteccionPoliticaConservacionDocumentalBloqueada, "", EstadoPoliticaConservacionDocumentalAprobada, time.Time{}},
		{"ordinaria con bloqueo", conservacion, ProteccionPoliticaConservacionDocumentalOrdinaria, bloqueo, EstadoPoliticaConservacionDocumentalAprobada, time.Time{}},
		{"estado cero", conservacion, ProteccionPoliticaConservacionDocumentalOrdinaria, "", "", time.Time{}},
		{"aprobada retirada", conservacion, ProteccionPoliticaConservacionDocumentalOrdinaria, "", EstadoPoliticaConservacionDocumentalAprobada, retirada},
		{"retirada sin instante", conservacion, ProteccionPoliticaConservacionDocumentalOrdinaria, "", EstadoPoliticaConservacionDocumentalRetirada, time.Time{}},
		{"retirada fuera de vigencia", conservacion, ProteccionPoliticaConservacionDocumentalOrdinaria, "", EstadoPoliticaConservacionDocumentalRetirada, solicitud.VigenteHasta()},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := NuevaPoliticaConservacionDocumental(
				solicitud, caso.hasta, caso.proteccion, caso.bloqueoRef,
				caso.estado, caso.retiradaEn,
			); !errors.Is(err, ErrPoliticaConservacionDocumentalInvalida) {
				t.Fatalf("estado incoherente aceptado: %v", err)
			}
		})
	}
}

func TestPoliticaConservacionDocumentalConservaCopiasDefensivas(t *testing.T) {
	vinculos := nuevosVinculosPoliticaConservacionDocumentalPrueba()
	huellaOriginal := append([]byte(nil), vinculos.huellaPolitica...)
	solicitud := vinculos.construir(t)
	vinculos.huellaPolitica[0] ^= 0xff
	if !bytes.Equal(solicitud.HuellaPoliticaSHA256(), huellaOriginal) {
		t.Fatal("el constructor retuvo la huella mutable")
	}
	huellaExpuesta := solicitud.HuellaPoliticaSHA256()
	huellaExpuesta[1] ^= 0xff
	if !bytes.Equal(solicitud.HuellaPoliticaSHA256(), huellaOriginal) {
		t.Fatal("el accesor expuso la huella interna")
	}
	politica := nuevaPoliticaConservacionDocumentalPrueba(
		t, solicitud, ProteccionPoliticaConservacionDocumentalBloqueada,
		EstadoPoliticaConservacionDocumentalAprobada,
	)
	resultado, err := NuevoResultadoPoliticaConservacionDocumental(
		politica, solicitud, time.Date(2028, 2, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("construir resultado neutral: %v", err)
	}
	primera := resultado.Politica().Solicitud().HuellaPoliticaSHA256()
	primera[2] ^= 0xff
	if !bytes.Equal(resultado.Politica().Solicitud().HuellaPoliticaSHA256(), huellaOriginal) {
		t.Fatal("el resultado compartio la huella mutable")
	}
}
