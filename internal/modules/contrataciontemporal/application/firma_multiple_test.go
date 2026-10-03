package application

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

// El fixture comprueba el consumidor del dictamen; no simula validación
// criptográfica ni acredita un documento firmado real.
func fixtureFirmaMultiple(n int) (docports.SolicitudVerificacionFirma, docports.VerificacionFirmasDocumento, []AntecedenteFirmaMultiple) {
	original := []byte("%PDF-1.7\noriginal sintetico\n")
	pdf := bytes.Clone(original)
	var firmas []docports.FirmaPDFVerificada
	var anteriores []AntecedenteFirmaMultiple
	for orden := 1; orden <= n; orden++ {
		pdf = append(pdf, []byte("\nrevision ")...)
		primera := len(pdf)
		pdf = append(pdf, []byte("<firma sintetica>")...)
		segunda := len(pdf)
		pdf = append(pdf, []byte("\ntrailer\n%%EOF\n")...)
		firmado := append(bytes.Clone(pdf[:primera]), pdf[segunda:]...)
		hash := sha256.Sum256(firmado)
		f := docports.FirmaPDFVerificada{
			Orden: orden, ByteRange: [4]uint64{0, uint64(primera), uint64(segunda), uint64(len(pdf) - segunda)},
			RevisionHuellaSHA256: huella(pdf), ContenidoFirmadoHuellaSHA256: hex.EncodeToString(hash[:]),
			RevisionLongitud: uint64(len(pdf)), CubreDocumentoCompletoHastaAqui: true,
			FirmanteRef: "firmante:" + string(rune('a'+orden)), CertificadoHuellaSHA256: huella([]byte{byte(orden)}),
			IntegridadEstado: "valida", CadenaEstado: "valida", CertificadoEstado: "vigente",
			RevocacionEstado: "vigente", SelloTiempoEstado: "no_presente", TipoFirma: "aprobacion",
			CambiosDesdeAnterior: docports.CambioFirmaPDF{Estado: "permitidos", Detalle: []string{"firma_anadida"}},
		}
		firmas = append(firmas, f)
		if orden < n {
			sufijo := string(rune('a' + orden))
			anteriores = append(anteriores, AntecedenteFirmaMultiple{PDFFirmado: bytes.Clone(pdf), Firma: ports.FirmaRegistrada{
				Documento: "informe_definitivo", PasoOrden: orden, Resultado: domain.ResultadoFirmaFirmado,
				Via: ports.ViaFirmaExternaPortafirmas, FirmaRef: "firma:" + sufijo, ReciboRef: "recibo:" + sufijo,
				OriginalRef: "ref:" + huella(original), OriginalVersion: 1, OriginalHuella: huella(original),
				FirmadoHuella: f.RevisionHuellaSHA256, FirmanteRef: f.FirmanteRef, CertificadoHuella: f.CertificadoHuellaSHA256,
				FirmantePrincipalAcreditado: true, DocumentoCustodiaRef: "custodia:" + sufijo, DocumentoCustodiaVersion: 1,
			}})
		}
	}
	zero := uint64(0)
	return docports.SolicitudVerificacionFirma{
			DocumentoID: "ref:" + huella(original), Version: 1, FormatoEsperado: "PAdES", HuellaOriginalSHA256: huella(original),
			ContenidoOriginal: original, ContenidoFirmado: pdf,
		}, docports.VerificacionFirmasDocumento{
			Estado: docports.EstadoVerificacionValida, Motivo: docports.MotivoFirmaVerificada, Formato: "PAdES", VinculoOriginal: true,
			HuellaOriginalSHA256: huella(original), HuellaFirmadoSHA256: huella(pdf), ComprobadoEn: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC),
			Firmas: firmas, CambiosPosteriores: docports.CambioPosteriorFirmaPDF{Estado: "ninguno", BytesNoFirmados: &zero},
		}, anteriores
}

func TestFirmaMultipleSeleccionaPasoExacto(t *testing.T) {
	for _, n := range []int{1, 2, domain.MaximoPasosCircuitoFirma} {
		s, v, anteriores := fixtureFirmaMultiple(n)
		f, err := ValidarFirmaMultipleParaPaso(s, v, "informe_definitivo", n, anteriores)
		if err != nil || f.Orden != n || f.FirmanteRef != v.Firmas[n-1].FirmanteRef {
			t.Fatalf("paso %d: %+v, %v", n, f, err)
		}
		f.CambiosDesdeAnterior.Detalle[0] = "alterado"
		if v.Firmas[n-1].CambiosDesdeAnterior.Detalle[0] != "firma_anadida" {
			t.Fatal("la selección conserva un alias mutable del dictamen")
		}
	}
}

func TestFirmaMultipleRechazaCadenaCriptograficaIncompleta(t *testing.T) {
	casos := []struct {
		nombre string
		cambia func(*docports.SolicitudVerificacionFirma, *docports.VerificacionFirmasDocumento)
	}{
		{"estado indeterminado", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Estado = docports.EstadoVerificacionIndeterminada
		}},
		{"motivo desconocido", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Motivo = "desconocido"
		}},
		{"formato ajeno", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Formato = "CAdES"
		}},
		{"sin comprobacion", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.ComprobadoEn = time.Time{}
		}},
		{"original desvinculado", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.VinculoOriginal = false
		}},
		{"hash original", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.HuellaOriginalSHA256 = huella([]byte("otro"))
		}},
		{"hash total", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.HuellaFirmadoSHA256 = huella([]byte("otro"))
		}},
		{"prefijo original", func(s *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			s.ContenidoOriginal[10] ^= 1
			s.HuellaOriginalSHA256 = huella(s.ContenidoOriginal)
			v.HuellaOriginalSHA256 = s.HuellaOriginalSHA256
		}},
		{"firma adicional", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas = append(v.Firmas, v.Firmas[1])
		}},
		{"firma ausente", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas = v.Firmas[1:]
		}},
		{"orden permutado", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[0], v.Firmas[1] = v.Firmas[1], v.Firmas[0]
		}},
		{"integridad anterior", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[0].IntegridadEstado = "parcial"
		}},
		{"cadena anterior", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[0].CadenaEstado = "no_comprobada"
		}},
		{"cert anterior", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[0].CertificadoEstado = "no_vigente"
		}},
		{"revocacion anterior", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[0].RevocacionEstado = "revocado"
		}},
		{"sello anterior", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[0].SelloTiempoEstado = "no_valido"
		}},
		{"cobertura parcial", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[0].CubreDocumentoCompletoHastaAqui = false
		}},
		{"sello documento", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[1].TipoFirma = "sello_tiempo_documento"
		}},
		{"certificacion", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[0].TipoFirma = "certificacion"
		}},
		{"DocMDP", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			p := 2
			v.Firmas[0].NivelDocMDP = &p
		}},
		{"cambio contenido", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[1].CambiosDesdeAnterior.Detalle = []string{"anotacion_anadida"}
		}},
		{"DSS", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[1].CambiosDesdeAnterior.Detalle = []string{"firma_anadida", "dss_anadido"}
		}},
		{"cambios desconocidos", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[1].CambiosDesdeAnterior.Estado = "no_comprobados"
		}},
		{"hash revision", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[0].RevisionHuellaSHA256 = huella([]byte("otra"))
		}},
		{"hash firmado", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[1].ContenidoFirmadoHuellaSHA256 = huella([]byte("otra"))
		}},
		{"rango inicio", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[1].ByteRange[0] = 1
		}},
		{"rango solapado", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[1].ByteRange[2] = v.Firmas[1].ByteRange[1]
		}},
		{"hueco excluye antecedente", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[1].ByteRange[1] = v.Firmas[0].RevisionLongitud - 1
		}},
		{"rango overflow", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[1].ByteRange[3] = math.MaxUint64
		}},
		{"rango fuera PDF", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[1].ByteRange[2] = math.MaxUint64
		}},
		{"revision fuera PDF", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.Firmas[1].RevisionLongitud = math.MaxUint64
		}},
		{"post sin evidencia", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.CambiosPosteriores.BytesNoFirmados = nil
		}},
		{"post bytes unsigned", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			*v.CambiosPosteriores.BytesNoFirmados = 1
		}},
		{"post cambio", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.CambiosPosteriores.Detalle = []string{"firma_anadida"}
		}},
		{"post desconocido", func(_ *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			v.CambiosPosteriores.Estado = "no_comprobados"
		}},
		{"cola no firmada", func(s *docports.SolicitudVerificacionFirma, v *docports.VerificacionFirmasDocumento) {
			s.ContenidoFirmado = append(s.ContenidoFirmado, 'x')
			v.HuellaFirmadoSHA256 = huella(s.ContenidoFirmado)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			s, v, anteriores := fixtureFirmaMultiple(2)
			caso.cambia(&s, &v)
			if _, err := ValidarFirmaMultipleParaPaso(s, v, "informe_definitivo", 2, anteriores); err == nil {
				t.Fatal("se aceptó una cadena incoherente")
			}
		})
	}
}

func TestFirmaMultipleConservaReciboYRevisionAnterior(t *testing.T) {
	casos := []struct {
		nombre string
		cambia func(*AntecedenteFirmaMultiple)
	}{
		{"otro documento", func(a *AntecedenteFirmaMultiple) { a.Firma.Documento = "resolucion" }},
		{"otro paso", func(a *AntecedenteFirmaMultiple) { a.Firma.PasoOrden = 2 }},
		{"original ref", func(a *AntecedenteFirmaMultiple) { a.Firma.OriginalRef = "original:otro" }},
		{"original version", func(a *AntecedenteFirmaMultiple) { a.Firma.OriginalVersion++ }},
		{"original huella", func(a *AntecedenteFirmaMultiple) { a.Firma.OriginalHuella = huella([]byte("otro")) }},
		{"certificado", func(a *AntecedenteFirmaMultiple) { a.Firma.CertificadoHuella = huella([]byte("otro")) }},
		{"firmante", func(a *AntecedenteFirmaMultiple) { a.Firma.FirmanteRef = "firmante:otro" }},
		{"recibo ausente", func(a *AntecedenteFirmaMultiple) { a.Firma.ReciboRef = "" }},
		{"firma ausente", func(a *AntecedenteFirmaMultiple) { a.Firma.FirmaRef = "" }},
		{"persona no acreditada", func(a *AntecedenteFirmaMultiple) { a.Firma.FirmantePrincipalAcreditado = false }},
		{"custodia ausente", func(a *AntecedenteFirmaMultiple) { a.Firma.DocumentoCustodiaRef = "" }},
		{"custodia version ausente", func(a *AntecedenteFirmaMultiple) { a.Firma.DocumentoCustodiaVersion = 0 }},
		{"PDF anterior alterado", func(a *AntecedenteFirmaMultiple) { a.PDFFirmado[0] ^= 1 }},
		{"PDF anterior truncado", func(a *AntecedenteFirmaMultiple) { a.PDFFirmado = a.PDFFirmado[:len(a.PDFFirmado)-1] }},
		{"firma legada", func(a *AntecedenteFirmaMultiple) { a.Firma.Via = "" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			s, v, anteriores := fixtureFirmaMultiple(2)
			caso.cambia(&anteriores[0])
			if _, err := ValidarFirmaMultipleParaPaso(s, v, "informe_definitivo", 2, anteriores); !errors.Is(err, ports.ErrCadenaFirmaDocumentoRota) {
				t.Fatalf("se esperaba cadena rota, se obtuvo %v", err)
			}
		})
	}
}

func TestFirmaMultipleCanonicoEstableYAntecedenteExacto(t *testing.T) {
	s, v, anteriores := fixtureFirmaMultiple(2)
	canon, err := CanonicoEvidenciaFirmasMultiple(s, v, "informe_definitivo", 2, anteriores)
	if err != nil {
		t.Fatal(err)
	}
	v.ComprobadoEn = v.ComprobadoEn.Add(time.Hour)
	replay, err := CanonicoEvidenciaFirmasMultiple(s, v, "informe_definitivo", 2, anteriores)
	if err != nil || !bytes.Equal(canon, replay) || bytes.Contains(canon, []byte("ComprobadoEn")) {
		t.Fatalf("la revalidación cambia el material: %v", err)
	}
	var cadena []json.RawMessage
	if err := json.Unmarshal(canon, &cadena); err != nil || len(cadena) != 2 {
		t.Fatalf("array no interpretable: %v", err)
	}
	s1, v1, _ := fixtureFirmaMultiple(1)
	canon1, err := CanonicoEvidenciaFirmasMultiple(s1, v1, "informe_definitivo", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	var cadena1 []json.RawMessage
	if err := json.Unmarshal(canon1, &cadena1); err != nil || !bytes.Equal(cadena[0], cadena1[0]) {
		t.Fatalf("la firma previa cambia al añadir el segundo paso: %v", err)
	}
}
