package application

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
		"documentos caído":                {func(c *custodioPrueba) { c.err = ports.ErrCustodiaFirmadoNoDisponible }, ports.ErrCustodiaFirmadoNoDisponible},
		"documentos deniega":              {func(c *custodioPrueba) { c.err = errors.New("403") }, ports.ErrCustodiaFirmadoDenegada},
		"contenido no admitido":           {func(c *custodioPrueba) { c.err = fmt.Errorf("x: %w", ports.ErrCustodiaFirmadoInvalida) }, ports.ErrCustodiaFirmadoInvalida},
		"otro PDF con la misma operación": {func(c *custodioPrueba) { c.err = ports.ErrCustodiaFirmadoEnConflicto }, ports.ErrCustodiaFirmadoEnConflicto},
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
		if !errors.Is(err, caso.esperado) || len(registro.registrado) != 0 || len(autorizador.visto) != 1 {
			t.Errorf("%s: %v (registros %d, autorizaciones %d)", nombre, err, len(registro.registrado), len(autorizador.visto))
		}
	}
}

func TestFirmaDocumentoDenegadaNoCustodia(t *testing.T) {
	s, registro, autorizador, custodio := servicioConCustodia(t)
	autorizador.denegar = true
	original := []byte("%PDF-1.7 borrador")
	firmado := append(append([]byte(nil), original...), []byte(" firma")...)
	_, err := s.Firmar(context.Background(), solicitudFirmaPrueba(1, original, firmado, "clave-firma-denegada-001"))
	if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || len(autorizador.visto) != 1 ||
		len(custodio.ordenes) != 0 || len(registro.registrado) != 0 {
		t.Fatalf("denegación tras custodiar o registrar: %v, autorización=%d custodia=%d registro=%d",
			err, len(autorizador.visto), len(custodio.ordenes), len(registro.registrado))
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

// Reintento de una firma ya registrada (respuesta perdida): el paso ya no está
// pendiente, pero se recupera el recibo original sin registrar otra firma.
func TestFirmaDocumentoReintentoDeUnaFirmaYaRegistrada(t *testing.T) {
	s, registro, autorizador, custodio := servicioConCustodia(t)
	borrador := []byte("%PDF-1.7 borrador")
	firmado := append(append([]byte{}, borrador...), []byte(" firma1")...)
	sol := solicitudFirmaPrueba(1, borrador, firmado, "clave-firma-000000001")
	primero, err := s.Firmar(context.Background(), sol)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.Firmar(context.Background(), sol)
	if err != nil || !r.Recibo.YaRegistrada || r.Recibo.Secuencia != primero.Recibo.Secuencia ||
		r.Recibo.DocumentoCustodiaRef != primero.Recibo.DocumentoCustodiaRef || len(registro.registrado) != 1 || len(autorizador.visto) != 2 {
		t.Fatalf("reintento: %v %+v (registros %d)", err, r.Recibo, len(registro.registrado))
	}
	// La custodia se vuelve a pedir con la misma orden: Documentos devuelve el
	// original.
	if len(custodio.ordenes) != 2 || custodio.ordenes[0].DocumentoRef != custodio.ordenes[1].DocumentoRef ||
		custodio.ordenes[0].ClaveIdempotencia != custodio.ordenes[1].ClaveIdempotencia {
		t.Fatalf("órdenes de custodia: %+v", custodio.ordenes)
	}
	// Con la misma clave en otro paso, o con otro PDF, es otra operación.
	otroPaso := solicitudFirmaPrueba(2, borrador, firmado, "clave-firma-000000001")
	if _, err := s.Firmar(context.Background(), otroPaso); !errors.Is(err, ports.ErrClaveFirmaDocumentoUsada) {
		t.Fatalf("misma clave en otro paso: %v", err)
	}
	otroPDF := solicitudFirmaPrueba(1, borrador, append(append([]byte{}, borrador...), []byte(" firma2")...), "clave-firma-000000001")
	if _, err := s.Firmar(context.Background(), otroPDF); !errors.Is(err, ports.ErrClaveFirmaDocumentoUsada) {
		t.Fatalf("misma clave con otro PDF: %v", err)
	}
	// Una clave nueva sobre el paso ya firmado sigue sin estar pendiente.
	if _, err := s.Firmar(context.Background(), solicitudFirmaPrueba(1, borrador, firmado, "clave-firma-000000009")); !errors.Is(err, ErrPasoFirmaNoPendiente) {
		t.Fatalf("clave nueva sobre paso firmado: %v", err)
	}
}
