package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type circuitoFirmaPrueba struct{ err error }

func (c circuitoFirmaPrueba) CircuitoFirma(context.Context) (domain.CircuitoFirma, error) {
	return domain.CircuitoFirma{CatalogoRef: "vec.contratacion_temporal.circuito_firma:1", HuellaCatalogo: strings.Repeat("c", 64),
		Documentos: []domain.CircuitoFirmaDocumento{{Documento: "informe_definitivo", Etiqueta: "Informe definitivo", Pasos: []domain.PasoCircuitoFirma{
			{Orden: 1, Cargo: "Técnico", PerfilRef: "perfil:ct:tecnico", Accion: "firma", Devolucion: domain.DevolucionVuelveARedaccion, Referencia: "circuito:1:informe_definitivo.p1"},
			{Orden: 2, Cargo: "Jefatura", PerfilRef: "perfil:ct:jefatura", Accion: "firma", Devolucion: domain.DevolucionVuelvePasoAnterior, Referencia: "circuito:1:informe_definitivo.p2"},
		}}}}, c.err
}

type registroFirmaPrueba struct {
	firmas     []ports.FirmaRegistrada
	registrado []ports.MaterialFirmaDocumento
}

func (r *registroFirmaPrueba) RegistrarFirma(_ context.Context, m ports.MaterialFirmaDocumento, c ports.CapacidadFirmaDocumento) (ports.ReciboFirmaDocumento, error) {
	if ValidarCapacidadFirmaDocumento(c, m) != nil {
		return ports.ReciboFirmaDocumento{}, ports.ErrFirmaDocumentoDenegada
	}
	h, _ := m.HuellaSHA256()
	r.registrado = append(r.registrado, m)
	r.firmas = append(r.firmas, ports.FirmaRegistrada{Documento: m.Documento, Secuencia: m.Secuencia, CatalogoHuella: m.CatalogoHuella,
		PasoOrden: m.PasoOrden, Resultado: m.Resultado, MotivoDevolucion: m.MotivoDevolucion, OriginalHuella: m.OriginalHuella, FirmadoHuella: m.FirmadoHuella})
	return ports.ReciboFirmaDocumento{FirmaRef: "firma-ct:x", ReciboRef: "recibo-firma-ct:x", Secuencia: m.Secuencia, Resultado: m.Resultado,
		ExpedienteVersion: m.VersionExpediente, SolicitudHuella: h, RegistradaEn: instanteFirmaPrueba}, nil
}

func (r *registroFirmaPrueba) ConsultarFirmas(context.Context, string, string) ([]ports.FirmaRegistrada, error) {
	return append([]ports.FirmaRegistrada(nil), r.firmas...), nil
}

type autorizadorFirmaPrueba struct {
	denegar bool
	visto   []ports.MaterialFirmaDocumento
}

var instanteFirmaPrueba = time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)

func (a *autorizadorFirmaPrueba) AutorizarFirmaDocumento(_ context.Context, m ports.MaterialFirmaDocumento) (ports.CapacidadFirmaDocumento, error) {
	a.visto = append(a.visto, m)
	if a.denegar {
		return ports.CapacidadFirmaDocumento{}, errors.New("denegada")
	}
	return ports.TransportarMaterialFirmaDocumento(materialFirmaPrueba(m)), nil
}

func materialFirmaPrueba(m ports.MaterialFirmaDocumento) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	h := strings.Repeat("a", 64)
	r, err := RecursoFirmaDocumento(m)
	if err != nil {
		panic(err)
	}
	huellaContexto, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		panic(err)
	}
	hasta := instanteFirmaPrueba.Add(time.Minute)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision-firma-001", h, h, "contexto-001", h, ports.AccionFirmarDocumento, r.Referencia, huellaContexto, ports.AudienciaFirmaDocumentoV3, hasta.Add(-5*time.Second), hasta)
	if err != nil {
		panic(err)
	}
	k := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{7}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(k.Public())
	if err != nil {
		panic(err)
	}
	material, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{5}, vecports.TamanoMinimoCapacidadCanonicaV3), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("cose"), []byte("evidencia"), raiz)
	if err != nil {
		panic(err)
	}
	return material
}

// verificadorPrueba acredita una firma PAdES simulada: el firmado empieza
// por el original. Solo es un doble del puerto, no una verificación.
type verificadorPrueba struct {
	motivo docports.MotivoVerificacionFirma
}

func (v verificadorPrueba) VerificarMotivado(_ context.Context, s docports.SolicitudVerificacionFirma) (docports.VerificacionFirmaMotivada, error) {
	suma := sha256.Sum256(s.ContenidoFirmado)
	r := docports.ResultadoVerificacionFirma{Estado: v.motivo.EstadoAsociado(), HuellaOriginalSHA256: s.HuellaOriginalSHA256,
		HuellaFirmadoSHA256: hex.EncodeToString(suma[:]), SelloTiempoEstado: docports.SelloTiempoNoPresente, RevocacionEstado: docports.RevocacionNoComprobada}
	if v.motivo == docports.MotivoFirmaVerificada && bytes.HasPrefix(s.ContenidoFirmado, s.ContenidoOriginal) {
		r.VinculoOriginal, r.FirmanteRef, r.CertificadoHuellaSHA256, r.RevocacionEstado = true, "ref:"+strings.Repeat("f", 64), strings.Repeat("e", 64), docports.RevocacionVigente
	}
	return docports.VerificacionFirmaMotivada{Resultado: r, Motivo: v.motivo}, nil
}

func solicitudFirmaPrueba(orden int, original, firmado []byte, clave string) SolicitudFirmaDocumento {
	return SolicitudFirmaDocumento{OrganizacionRef: "organizacion:desarrollo:dipgra", ExpedienteRef: "expediente:ct:001", VersionExpediente: 7,
		Documento: "informe_definitivo", PasoOrden: orden, Resultado: domain.ResultadoFirmaFirmado, Original: original, Firmado: firmado, ClaveIdempotencia: clave}
}

func TestFirmaDocumentoCircuitoCompletoConCadena(t *testing.T) {
	registro, autorizador := &registroFirmaPrueba{}, &autorizadorFirmaPrueba{}
	s, err := NuevoServicioFirmaDocumento(circuitoFirmaPrueba{}, registro, autorizador, verificadorPrueba{motivo: docports.MotivoFirmaVerificada})
	if err != nil {
		t.Fatal(err)
	}
	borrador := []byte("%PDF-1.7 borrador")
	primera := append(append([]byte{}, borrador...), []byte(" firma1")...)
	r, err := s.Firmar(context.Background(), solicitudFirmaPrueba(1, borrador, primera, "clave-firma-000000001"))
	if err != nil || r.Recibo.Secuencia != 1 || r.Material.OriginalHuella != huella(borrador) || r.Material.FirmadoHuella != huella(primera) {
		t.Fatalf("paso 1: %+v %v", r, err)
	}
	// El paso 2 firma el mismo borrador que el paso 1, no otro documento.
	otro := []byte("%PDF-1.7 otro borrador")
	if _, err := s.Firmar(context.Background(), solicitudFirmaPrueba(2, otro, append(append([]byte{}, otro...), 'x'), "clave-firma-000000002")); !errors.Is(err, ports.ErrCadenaFirmaDocumentoRota) {
		t.Fatalf("cadena rota admitida: %v", err)
	}
	segunda := append(append([]byte{}, borrador...), []byte(" firma2")...)
	if r, err = s.Firmar(context.Background(), solicitudFirmaPrueba(2, borrador, segunda, "clave-firma-000000003")); err != nil || r.Recibo.Secuencia != 2 {
		t.Fatalf("paso 2: %v", err)
	}
	estado, err := s.Consultar(context.Background(), "organizacion:desarrollo:dipgra", "expediente:ct:001")
	if err != nil || len(estado.Documentos) != 1 || !estado.Documentos[0].Completo {
		t.Fatalf("circuito no completo: %+v %v", estado, err)
	}
	if _, err := s.Firmar(context.Background(), solicitudFirmaPrueba(2, borrador, append(segunda, 'y'), "clave-firma-000000004")); !errors.Is(err, ErrPasoFirmaNoPendiente) {
		t.Fatalf("firma sobre circuito completo: %v", err)
	}
}

func TestFirmaDocumentoNuncaSinVerificacion(t *testing.T) {
	borrador := []byte("%PDF-1.7 borrador")
	firmado := append(append([]byte{}, borrador...), 'f')
	registro, autorizador := &registroFirmaPrueba{}, &autorizadorFirmaPrueba{}
	apagado, err := NuevoServicioFirmaDocumento(circuitoFirmaPrueba{}, registro, autorizador, nil)
	if err != nil || apagado.VerificacionDisponible() {
		t.Fatal("servicio sin verificador")
	}
	if _, err := apagado.Firmar(context.Background(), solicitudFirmaPrueba(1, borrador, firmado, "clave-firma-000000010")); !errors.Is(err, ErrVerificacionFirmaApagada) {
		t.Fatalf("firma sin verificador: %v", err)
	}
	for _, motivo := range []docports.MotivoVerificacionFirma{docports.MotivoValidadorNoDisponible, docports.MotivoIntegridadNoValida, docports.MotivoRevocacionNoAcreditada} {
		s, _ := NuevoServicioFirmaDocumento(circuitoFirmaPrueba{}, registro, autorizador, verificadorPrueba{motivo: motivo})
		var rechazo DictamenRechazado
		if _, err := s.Firmar(context.Background(), solicitudFirmaPrueba(1, borrador, firmado, "clave-firma-000000011")); !errors.As(err, &rechazo) || rechazo.Motivo != motivo {
			t.Fatalf("%s: %v", motivo, err)
		}
	}
	if len(registro.registrado) != 0 || len(autorizador.visto) != 0 {
		t.Fatal("una firma no verificada llegó a autorizarse o registrarse")
	}
}

func TestDevolucionFirmaDocumento(t *testing.T) {
	registro, autorizador := &registroFirmaPrueba{}, &autorizadorFirmaPrueba{}
	s, _ := NuevoServicioFirmaDocumento(circuitoFirmaPrueba{}, registro, autorizador, nil)
	sol := SolicitudFirmaDocumento{OrganizacionRef: "organizacion:desarrollo:dipgra", ExpedienteRef: "expediente:ct:001", VersionExpediente: 7,
		Documento: "informe_definitivo", PasoOrden: 1, Resultado: domain.ResultadoFirmaDevuelto, MotivoDevolucion: "Falta la fecha de efectos", ClaveIdempotencia: "clave-devolucion-00001"}
	r, err := s.Firmar(context.Background(), sol)
	if err != nil || r.Recibo.Resultado != domain.ResultadoFirmaDevuelto || r.Material.FirmadoHuella != "" {
		t.Fatalf("devolución: %+v %v", r, err)
	}
	sol.MotivoDevolucion = " "
	if _, err := s.Firmar(context.Background(), sol); !errors.Is(err, ports.ErrSolicitudFirmaDocumentoInvalida) {
		t.Fatal("devolución sin motivo admitida")
	}
	sol.MotivoDevolucion, sol.Original = "Motivo válido", []byte("x")
	if _, err := s.Firmar(context.Background(), sol); !errors.Is(err, ports.ErrSolicitudFirmaDocumentoInvalida) {
		t.Fatal("devolución con documento admitida")
	}
}

func TestFirmaDocumentoDenegadaYCircuitoAusente(t *testing.T) {
	borrador := []byte("%PDF-1.7 borrador")
	firmado := append(append([]byte{}, borrador...), 'f')
	registro := &registroFirmaPrueba{}
	s, _ := NuevoServicioFirmaDocumento(circuitoFirmaPrueba{}, registro, &autorizadorFirmaPrueba{denegar: true}, verificadorPrueba{motivo: docports.MotivoFirmaVerificada})
	if _, err := s.Firmar(context.Background(), solicitudFirmaPrueba(1, borrador, firmado, "clave-firma-000000020")); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || len(registro.registrado) != 0 {
		t.Fatalf("denegación: %v", err)
	}
	s, _ = NuevoServicioFirmaDocumento(circuitoFirmaPrueba{err: errors.New("sin catálogo")}, registro, &autorizadorFirmaPrueba{}, nil)
	if _, err := s.Consultar(context.Background(), "organizacion:desarrollo:dipgra", "expediente:ct:001"); !errors.Is(err, ErrCircuitoFirmaNoDisponible) {
		t.Fatalf("circuito ausente: %v", err)
	}
	if _, err := NuevoServicioFirmaDocumento(nil, registro, &autorizadorFirmaPrueba{}, nil); err == nil {
		t.Fatal("servicio sin circuito")
	}
}

func TestMaterialFirmaDocumentoCanonico(t *testing.T) {
	m := ports.MaterialFirmaDocumento{OrganizacionRef: "organizacion:desarrollo:dipgra", ExpedienteRef: "expediente:ct:001", VersionExpediente: 7,
		Documento: "informe_definitivo", CatalogoRef: "vec.contratacion_temporal.circuito_firma:1", CatalogoHuella: strings.Repeat("c", 64),
		PasoRef: "circuito:1:p1", PasoOrden: 1, Secuencia: 1, Resultado: domain.ResultadoFirmaDevuelto, MotivoDevolucion: "Falta <fecha>",
		ClaveIdempotencia: "clave-devolucion-00001"}
	c, err := m.Canonico()
	want := `{"OrganizacionRef":"organizacion:desarrollo:dipgra","ExpedienteRef":"expediente:ct:001","VersionExpediente":7,"Documento":"informe_definitivo","CatalogoRef":"vec.contratacion_temporal.circuito_firma:1","CatalogoHuella":"` + strings.Repeat("c", 64) + `","PasoRef":"circuito:1:p1","PasoOrden":1,"Secuencia":1,"Resultado":"devuelto","MotivoDevolucion":"Falta \u003cfecha\u003e","OriginalHuella":null,"FirmadoHuella":null,"CertificadoHuella":null,"FirmanteRef":null,"PoliticaVerificacion":null,"RevocacionEstado":null,"SelloTiempoEstado":null,"ClaveIdempotencia":"clave-devolucion-00001"}`
	if err != nil || string(c) != want {
		t.Fatalf("canónico:\n%s\n%v", c, err)
	}
	m.FirmadoHuella = strings.Repeat("1", 64)
	if m.Validar() == nil {
		t.Fatal("devolución con huella de firmado admitida")
	}
}

// CT118 recalcula en SQL la huella de contexto del recurso con una
// concatenación literal; debe coincidir con la del dominio V3.
func TestHuellaContextoRecursoFirmaCoincideConCT118(t *testing.T) {
	m := ports.MaterialFirmaDocumento{OrganizacionRef: "organizacion:desarrollo:dipgra", ExpedienteRef: "expediente:ct:001", VersionExpediente: 7,
		Documento: "informe_definitivo", CatalogoRef: "vec.contratacion_temporal.circuito_firma:1", CatalogoHuella: strings.Repeat("c", 64),
		PasoRef: "circuito:1:p1", PasoOrden: 1, Secuencia: 1, Resultado: domain.ResultadoFirmaDevuelto, MotivoDevolucion: "Falta la fecha",
		ClaveIdempotencia: "clave-devolucion-00001"}
	r, err := RecursoFirmaDocumento(m)
	if err != nil {
		t.Fatal(err)
	}
	dominio, err := r.HuellaContextoAutorizacionSHA256()
	h, _ := m.HuellaSHA256()
	sql := huella([]byte(`{"ambitos":{"organizacion_ref":"` + m.OrganizacionRef + `"},"atributos":{"material_sha256":"` + h + `"}}`))
	if err != nil || dominio != sql || r.Referencia != "operacion-firma-ct:clave-devolucion-00001" {
		t.Fatalf("huella de contexto divergente: %s %s %v", dominio, sql, err)
	}
}
