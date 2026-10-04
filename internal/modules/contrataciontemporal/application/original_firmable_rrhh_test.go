package application

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type consultorOriginalRRHHPrueba struct {
	detalle   ports.DetalleExpedienteRRHH
	err       error
	llamadas  int
	solicitud ports.SolicitudDetalleRRHH
}

func (c *consultorOriginalRRHHPrueba) Consultar(_ context.Context, s ports.SolicitudDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	c.llamadas++
	c.solicitud = s
	return c.detalle, c.err
}

type tiposOriginalRRHHPrueba struct {
	ref      string
	err      error
	llamadas int
	tipo     ports.TipoBorradorRRHH
}

func (c *tiposOriginalRRHHPrueba) ResolverTipoOriginalRRHH(_ context.Context, tipo ports.TipoBorradorRRHH) (string, error) {
	c.llamadas++
	c.tipo = tipo
	return c.ref, c.err
}

type renderOriginalRRHHPrueba struct {
	contenido []byte
	err       error
	llamadas  int
	tipo      ports.TipoBorradorRRHH
	version   uint64
}

func (r *renderOriginalRRHHPrueba) RenderizarBorrador(_ context.Context, tipo ports.TipoBorradorRRHH, d ports.DetalleExpedienteRRHH) ([]byte, error) {
	r.llamadas++
	r.tipo, r.version = tipo, d.Resumen.Version
	return r.contenido, r.err
}

func prepararFuenteOriginalRRHHPrueba(t *testing.T) (*FuenteOriginalFirmableRRHH, vecports.SolicitudOriginalFirmableCT, *consultorOriginalRRHHPrueba, *consultorOriginalRRHHPrueba, *tiposOriginalRRHHPrueba, *renderOriginalRRHHPrueba) {
	t.Helper()
	contenido, err := os.ReadFile("../domain/testdata/expediente_nombramiento_v7.json")
	if err != nil {
		t.Fatal(err)
	}
	var expediente domain.Expediente
	if err := json.Unmarshal(contenido, &expediente); err != nil || expediente.Validar() != nil {
		t.Fatalf("fixture CT v7 inválido: %v", err)
	}
	ahora := expediente.ActualizadoEn.Add(time.Minute)
	contexto := contextoConsultaRRHHV3Prueba(t, ahora)
	solicitud, err := ports.NuevaSolicitudDetalleRRHH(expediente.Referencia, expediente.Version)
	if err != nil {
		t.Fatal(err)
	}
	capacidad := capacidadConsultaDetalleRRHHV3Prueba(t, contexto, solicitud, ahora)
	recibo := reciboConsultaRRHHPrueba(t, contexto, capacidad, ahora, expediente.Referencia, expediente.Version, 1)
	detalle, err := ports.NuevoDetalleExpedienteRRHH(expediente, recibo)
	if err != nil {
		t.Fatalf("proyección CT v7: %v", err)
	}
	actual := &consultorOriginalRRHHPrueba{detalle: detalle}
	historico := &consultorOriginalRRHHPrueba{detalle: detalle}
	tipos := &tiposOriginalRRHHPrueba{ref: "ref:" + strings.Repeat("a", 64)}
	render := &renderOriginalRRHHPrueba{contenido: []byte("%PDF-1.7\noriginal de prueba")}
	f, err := NuevaFuenteOriginalFirmableRRHH(actual, historico, render, tipos)
	if err != nil {
		t.Fatal(err)
	}
	in := vecports.SolicitudOriginalFirmableCT{
		OrganizacionRef: expediente.OrganizacionRef,
		ExpedienteRef:   expediente.Referencia,
		Documento:       string(ports.BorradorResolucion),
		OriginalVersion: expediente.Version,
	}
	in.OriginalRef = identidadOriginalFirmableRRHH(in).Referencia()
	return f, in, actual, historico, tipos, render
}

func TestFuenteOriginalRRHHConsultaVersionV3YRenderizaUnaVez(t *testing.T) {
	t.Parallel()
	f, in, actual, historico, tipos, render := prepararFuenteOriginalRRHHPrueba(t)
	resultado, err := f.ObtenerPDFOriginalCT(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if actual.llamadas != 1 || actual.solicitud.VersionObservada() != 0 ||
		historico.llamadas != 0 || tipos.llamadas != 1 || render.llamadas != 1 ||
		render.version != in.OriginalVersion || render.tipo != ports.BorradorResolucion ||
		resultado.TipoRef != tipos.ref || string(resultado.Contenido) != string(render.contenido) {
		t.Fatal("la fuente no usó la lectura autorizada y el PDF esperados")
	}
	render.contenido[0] = 'X'
	if resultado.Contenido[0] != '%' {
		t.Fatal("el resultado comparte memoria mutable con el renderizador")
	}
}

func TestFuenteOriginalRRHHRecuperaPropuestaV7ConFachadaHistorica(t *testing.T) {
	t.Parallel()
	f, in, actual, historico, tipos, render := prepararFuenteOriginalRRHHPrueba(t)
	actual.detalle = actual.detalle.Clonar()
	anterior := actual.detalle.Hitos[6]
	for i, accion := range []domain.ClaveCatalogo{"registrar_resolucion_formalizacion", domain.AccionRegistrarAnotacionAdministrativa} {
		version := uint64(8 + i)
		hito := ports.HitoExpedienteRRHH{
			Secuencia: version, VersionExpediente: version, AccionClave: accion,
			RealizadaEn: anterior.RealizadaEn.Add(time.Duration(i+1) * time.Minute),
			FaseOrigen:  "nombramiento", FaseDestino: "nombramiento",
			EstadoOrigen: domain.EstadoEnCurso, EstadoDestino: domain.EstadoEnCurso,
		}
		actual.detalle.Hitos = append(actual.detalle.Hitos, hito)
	}
	actual.detalle.Resumen.Version = 9
	actual.detalle.Resumen.ActualizadoEn = actual.detalle.Hitos[8].RealizadaEn
	if _, err := f.ObtenerPDFOriginalCT(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	if actual.llamadas != 1 || historico.llamadas != 1 ||
		historico.solicitud.VersionObservada() != 7 || tipos.llamadas != 1 ||
		render.llamadas != 1 || render.version != 7 {
		t.Fatal("la propuesta no usó su fachada histórica v7")
	}
	historico.err = ports.ErrConsultaRRHHNoDisponible
	if _, err := f.ObtenerPDFOriginalCT(context.Background(), in); !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) || render.llamadas != 1 {
		t.Fatal("caída V3 histórica permitió una representación nueva")
	}
	historico.err = nil
	historico.detalle.Hitos[6].AccionClave = "otra_propuesta"
	if _, err := f.ObtenerPDFOriginalCT(context.Background(), in); !errors.Is(err, vecports.ErrOriginalFirmableCTConflicto) || render.llamadas != 1 {
		t.Fatal("historia v7 divergente se representó como original")
	}
}

func TestFuenteOriginalRRHHRechazaIdentidadYVersionAjena(t *testing.T) {
	t.Parallel()
	for _, caso := range []struct {
		nombre  string
		cambiar func(*vecports.SolicitudOriginalFirmableCT)
	}{
		{"organizacion", func(s *vecports.SolicitudOriginalFirmableCT) { s.OrganizacionRef = "organizacion:ajena" }},
		{"expediente", func(s *vecports.SolicitudOriginalFirmableCT) { s.ExpedienteRef = "expediente:ajeno" }},
		{"version_futura", func(s *vecports.SolicitudOriginalFirmableCT) { s.OriginalVersion++ }},
		{"tipo_ajeno", func(s *vecports.SolicitudOriginalFirmableCT) { s.Documento = "contrato_laboral" }},
	} {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			f, in, _, _, tipos, render := prepararFuenteOriginalRRHHPrueba(t)
			caso.cambiar(&in)
			in.OriginalRef = identidadOriginalFirmableRRHH(in).Referencia()
			if _, err := f.ObtenerPDFOriginalCT(context.Background(), in); err == nil {
				t.Fatal("se aceptó identidad, versión o tipo ajeno")
			}
			if tipos.llamadas != 0 || render.llamadas != 0 {
				t.Fatal("se procesó un documento rechazado")
			}
		})
	}
}

func TestFuenteOriginalRRHHCierraFallosV3CatalogoYPDF(t *testing.T) {
	t.Parallel()
	f, in, actual, _, tipos, render := prepararFuenteOriginalRRHHPrueba(t)
	falloV3 := errors.New("V3 no disponible")
	actual.err = falloV3
	if _, err := f.ObtenerPDFOriginalCT(context.Background(), in); !errors.Is(err, falloV3) || tipos.llamadas != 0 || render.llamadas != 0 {
		t.Fatal("una caída V3 permitió resolver tipo o renderizar")
	}
	actual.err = nil
	falloCatalogo := errors.New("catálogo no disponible")
	tipos.err = falloCatalogo
	if _, err := f.ObtenerPDFOriginalCT(context.Background(), in); !errors.Is(err, falloCatalogo) || render.llamadas != 0 {
		t.Fatal("la caída del catálogo permitió renderizar")
	}
	tipos.err = nil
	falloPDF := errors.New("renderizador no disponible")
	render.err = falloPDF
	if _, err := f.ObtenerPDFOriginalCT(context.Background(), in); !errors.Is(err, falloPDF) {
		t.Fatal("la caída del PDF se trató como original custodial")
	}
	render.err = nil
	render.contenido = []byte("contenido no PDF")
	if _, err := f.ObtenerPDFOriginalCT(context.Background(), in); !errors.Is(err, vecports.ErrOriginalFirmableCTNoDisponible) {
		t.Fatal("se aceptaron bytes que no son PDF")
	}
}

func TestFuenteOriginalRRHHNoAceptaRefNiTipoInyectados(t *testing.T) {
	t.Parallel()
	f, in, actual, _, tipos, render := prepararFuenteOriginalRRHHPrueba(t)
	in.OriginalRef = "ref:" + strings.Repeat("f", 64)
	if _, err := f.ObtenerPDFOriginalCT(context.Background(), in); !errors.Is(err, vecports.ErrOriginalFirmableCTInvalido) || actual.llamadas != 0 {
		t.Fatal("se aceptó una referencia de original ajena")
	}
	in.OriginalRef = identidadOriginalFirmableRRHH(in).Referencia()
	tipos.ref = "tipo:inventado"
	if _, err := f.ObtenerPDFOriginalCT(context.Background(), in); !errors.Is(err, vecports.ErrOriginalFirmableCTNoDisponible) || render.llamadas != 0 {
		t.Fatal("se aceptó un tipo sin referencia gobernada")
	}
}
