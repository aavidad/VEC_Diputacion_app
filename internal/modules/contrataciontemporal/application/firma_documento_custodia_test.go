package application

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

// custodioPrueba hace de Documentos: devuelve el documento con la huella de
// los bytes recibidos, salvo que se le pida fallar o mentir.
type custodioPrueba struct {
	ordenes []ports.OrdenCustodiaFirmado
	err     error
	alterar func(*ports.DocumentoCustodiado)
}

func (c *custodioPrueba) CustodiarFirmado(_ context.Context, o ports.OrdenCustodiaFirmado) (ports.DocumentoCustodiado, error) {
	c.ordenes = append(c.ordenes, o)
	if c.err != nil {
		return ports.DocumentoCustodiado{}, c.err
	}
	d := ports.DocumentoCustodiado{Ref: o.DocumentoRef, Version: o.Version, HuellaSHA256: huella(o.Contenido)}
	if c.alterar != nil {
		c.alterar(&d)
	}
	return d, nil
}

const tipoCustodiaPrueba = "contratacion_temporal.resolucion_firmada.v1"

func servicioConCustodia(t *testing.T) (*ServicioFirmaDocumento, *registroFirmaPrueba, *autorizadorFirmaPrueba, *custodioPrueba) {
	t.Helper()
	registro, autorizador, custodio := &registroFirmaPrueba{}, &autorizadorFirmaPrueba{}, &custodioPrueba{}
	s, err := NuevoServicioFirmaDocumento(circuitoFirmaPrueba{}, registro, autorizador, verificadorPrueba{motivo: docports.MotivoFirmaVerificada})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ComponerCustodia(custodio, map[string]string{"informe_definitivo": tipoCustodiaPrueba}); err != nil {
		t.Fatal(err)
	}
	return s, registro, autorizador, custodio
}

func TestFirmaDocumentoCustodiaElPDFFirmadoYLoEnlaza(t *testing.T) {
	s, registro, autorizador, custodio := servicioConCustodia(t)
	borrador := []byte("%PDF-1.7 borrador")
	firmado := append(append([]byte{}, borrador...), []byte(" firma1")...)
	sol := solicitudFirmaPrueba(1, borrador, firmado, "clave-firma-000000001")
	r, err := s.Firmar(context.Background(), sol)
	if err != nil {
		t.Fatal(err)
	}
	ref := ports.DocumentoCustodiaRef(sol.OrganizacionRef, sol.ExpedienteRef, sol.ClaveIdempotencia)
	if len(custodio.ordenes) != 1 || len(registro.registrado) != 1 {
		t.Fatalf("custodias %d, registros %d", len(custodio.ordenes), len(registro.registrado))
	}
	o := custodio.ordenes[0]
	if o.DocumentoRef != ref || o.Version != ports.VersionDocumentoCustodiado || o.TipoDocumental != tipoCustodiaPrueba ||
		!bytes.Equal(o.Contenido, firmado) || o.HuellaOriginalSHA256 != huella(borrador) ||
		o.ExpedienteRef != ports.ExpedienteDocumentalRef(sol.OrganizacionRef, sol.ExpedienteRef) ||
		o.ClaveIdempotencia == ref || o.FirmaOperacionRef == ref {
		t.Fatalf("orden de custodia inesperada: %+v", o)
	}
	// El enlace va en el material que autoriza la V3 y vuelve en el recibo.
	m := registro.registrado[0]
	if m.DocumentoCustodiaRef != ref || m.DocumentoCustodiaVersion != ports.VersionDocumentoCustodiado ||
		autorizador.visto[0].DocumentoCustodiaRef != ref || r.Recibo.DocumentoCustodiaRef != ref ||
		r.Custodiado.Ref != ref || r.Custodiado.HuellaSHA256 != huella(firmado) {
		t.Fatalf("enlace inesperado: %+v", r)
	}
}

func TestFirmaDocumentoNoRegistraSiLaCustodiaFalla(t *testing.T) {
	casos := map[string]struct {
		preparar func(*custodioPrueba)
		esperado error
	}{
		"documentos caído":   {func(c *custodioPrueba) { c.err = ports.ErrCustodiaFirmadoNoDisponible }, ports.ErrCustodiaFirmadoNoDisponible},
		"documentos deniega": {func(c *custodioPrueba) { c.err = errors.New("403") }, ports.ErrCustodiaFirmadoDenegada},
		"otra huella": {func(c *custodioPrueba) {
			c.alterar = func(d *ports.DocumentoCustodiado) { d.HuellaSHA256 = huella([]byte("otro")) }
		}, ports.ErrResultadoFirmaDocumentoInvalido},
		"otro documento": {func(c *custodioPrueba) {
			c.alterar = func(d *ports.DocumentoCustodiado) { d.Ref = "ref:" + huella([]byte("otro")) }
		}, ports.ErrResultadoFirmaDocumentoInvalido},
		"otra versión": {func(c *custodioPrueba) { c.alterar = func(d *ports.DocumentoCustodiado) { d.Version = 2 } },
			ports.ErrResultadoFirmaDocumentoInvalido},
	}
	for nombre, caso := range casos {
		s, registro, autorizador, custodio := servicioConCustodia(t)
		caso.preparar(custodio)
		borrador := []byte("%PDF-1.7 borrador")
		_, err := s.Firmar(context.Background(), solicitudFirmaPrueba(1, borrador, append(append([]byte{}, borrador...), 'f'), "clave-firma-000000001"))
		if !errors.Is(err, caso.esperado) || len(registro.registrado) != 0 || len(autorizador.visto) != 0 {
			t.Errorf("%s: %v (registros %d, autorizaciones %d)", nombre, err, len(registro.registrado), len(autorizador.visto))
		}
	}
}

func TestFirmaDocumentoDevolucionNoSeCustodia(t *testing.T) {
	s, registro, _, custodio := servicioConCustodia(t)
	sol := SolicitudFirmaDocumento{OrganizacionRef: "organizacion:desarrollo:dipgra", ExpedienteRef: "expediente:ct:001", VersionExpediente: 7,
		Documento: "informe_definitivo", PasoOrden: 1, Resultado: domain.ResultadoFirmaDevuelto, MotivoDevolucion: "Falta la fecha",
		ClaveIdempotencia: "clave-devolucion-00001"}
	if _, err := s.Firmar(context.Background(), sol); err != nil {
		t.Fatal(err)
	}
	if len(custodio.ordenes) != 0 || registro.registrado[0].DocumentoCustodiaRef != "" {
		t.Fatal("una devolución no lleva documento custodiado")
	}
}

func TestFirmaDocumentoSinCustodiaNoLaPide(t *testing.T) {
	registro, autorizador, custodio := &registroFirmaPrueba{}, &autorizadorFirmaPrueba{}, &custodioPrueba{}
	s, err := NuevoServicioFirmaDocumento(circuitoFirmaPrueba{}, registro, autorizador, verificadorPrueba{motivo: docports.MotivoFirmaVerificada})
	if err != nil {
		t.Fatal(err)
	}
	// Custodia compuesta para otro documento: el informe no se custodia.
	if err := s.ComponerCustodia(custodio, map[string]string{"resolucion": tipoCustodiaPrueba}); err != nil {
		t.Fatal(err)
	}
	borrador := []byte("%PDF-1.7 borrador")
	if _, err := s.Firmar(context.Background(), solicitudFirmaPrueba(1, borrador, append(append([]byte{}, borrador...), 'f'), "clave-firma-000000001")); err != nil {
		t.Fatal(err)
	}
	if len(custodio.ordenes) != 0 || registro.registrado[0].DocumentoCustodiaRef != "" {
		t.Fatal("un documento sin custodia configurada no se custodia")
	}
}

func TestComponerCustodiaUnaSolaVezYConDatosValidos(t *testing.T) {
	s, _, _, custodio := servicioConCustodia(t)
	if s.ComponerCustodia(custodio, map[string]string{"resolucion": tipoCustodiaPrueba}) == nil {
		t.Fatal("la custodia se fija una sola vez")
	}
	nuevo, _ := NuevoServicioFirmaDocumento(circuitoFirmaPrueba{}, &registroFirmaPrueba{}, &autorizadorFirmaPrueba{}, nil)
	for nombre, tipos := range map[string]map[string]string{
		"sin documentos": {}, "clave inválida": {"Resolución": tipoCustodiaPrueba}, "tipo vacío": {"resolucion": ""},
	} {
		if nuevo.ComponerCustodia(custodio, tipos) == nil {
			t.Errorf("%s: admitido", nombre)
		}
	}
	if nuevo.ComponerCustodia(nil, map[string]string{"resolucion": tipoCustodiaPrueba}) == nil {
		t.Error("custodio nulo admitido")
	}
}

// Referencia fija del JSON canónico con enlace: es el texto cuya huella ata la
// V3 y que CT145 recibe tal cual.
func TestMaterialFirmaDocumentoCanonicoConEnlace(t *testing.T) {
	h := func(c string) string { return strings.Repeat(c, 64) }
	m := ports.MaterialFirmaDocumento{OrganizacionRef: "organizacion:desarrollo:dipgra", ExpedienteRef: "expediente:ct:001", VersionExpediente: 7,
		Documento: "resolucion", CatalogoRef: "vec.contratacion_temporal.circuito_firma:1", CatalogoHuella: h("c"),
		PasoRef: "circuito:1:p1", PasoOrden: 1, Secuencia: 1, Resultado: domain.ResultadoFirmaFirmado,
		OriginalHuella: h("a"), FirmadoHuella: h("1"), CertificadoHuella: h("e"), FirmanteRef: "ref:" + h("f"),
		PoliticaVerificacion: ports.PoliticaVerificacionFirma, RevocacionEstado: "vigente", SelloTiempoEstado: "no_presente",
		ClaveIdempotencia: "clave-firma-000000001", DocumentoCustodiaRef: "ref:" + h("d"), DocumentoCustodiaVersion: 1}
	c, err := m.Canonico()
	want := `{"OrganizacionRef":"organizacion:desarrollo:dipgra","ExpedienteRef":"expediente:ct:001","VersionExpediente":7,"Documento":"resolucion","CatalogoRef":"vec.contratacion_temporal.circuito_firma:1","CatalogoHuella":"` + h("c") + `","PasoRef":"circuito:1:p1","PasoOrden":1,"Secuencia":1,"Resultado":"firmado","MotivoDevolucion":null,"OriginalHuella":"` + h("a") + `","FirmadoHuella":"` + h("1") + `","CertificadoHuella":"` + h("e") + `","FirmanteRef":"ref:` + h("f") + `","PoliticaVerificacion":"politica:vec:firma:verificacion-autonoma:v1","RevocacionEstado":"vigente","SelloTiempoEstado":"no_presente","ClaveIdempotencia":"clave-firma-000000001","DocumentoCustodiaRef":"ref:` + h("d") + `","DocumentoCustodiaVersion":1}`
	if err != nil || string(c) != want {
		t.Fatalf("canónico con enlace:\n%s\n%v", c, err)
	}
	for nombre, alterar := range map[string]func(*ports.MaterialFirmaDocumento){
		"referencia sin versión":       func(m *ports.MaterialFirmaDocumento) { m.DocumentoCustodiaVersion = 0 },
		"versión sin referencia":       func(m *ports.MaterialFirmaDocumento) { m.DocumentoCustodiaRef = "" },
		"versión por encima de 2^53-1": func(m *ports.MaterialFirmaDocumento) { m.DocumentoCustodiaVersion = 1 << 53 },
		"referencia con espacio":       func(m *ports.MaterialFirmaDocumento) { m.DocumentoCustodiaRef = "ref con espacio" },
	} {
		x := m
		alterar(&x)
		if x.Validar() == nil {
			t.Errorf("%s: admitido", nombre)
		}
	}
}
