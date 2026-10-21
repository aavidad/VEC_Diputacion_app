package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmasv2"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// El selector sustituye solo la decisión por cargo: sin paso, denegado.
type selectorDescargaCargoPrueba struct {
	conPaso  bool
	salida   consultafirmasv2.SeleccionDescarga
	llamadas int
}

func (s *selectorDescargaCargoPrueba) SeleccionarConsultaDescarga(context.Context,
	consultafirmasv2.SolicitudDescarga) (consultafirmasv2.SeleccionDescarga, error) {
	s.llamadas++
	if !s.conPaso {
		return consultafirmasv2.SeleccionDescarga{}, ports.ErrFirmaDocumentoDenegada
	}
	return s.salida, nil
}

type historiaDescargaCargoPrueba struct {
	salida   consultafirmasv2.Resultado
	llamadas int
}

func (c *historiaDescargaCargoPrueba) Consultar(context.Context, consultafirmasv2.Solicitud) (consultafirmasv2.Resultado, error) {
	c.llamadas++
	return c.salida, nil
}

type pdfDescargaCargoPrueba struct {
	cuerpo   []byte
	llamadas int
}

func (p *pdfDescargaCargoPrueba) ObtenerPDFFirmaAnterior(_ context.Context,
	q ports.SolicitudPDFFirmaAnterior) (ports.PDFFirmaAnterior, error) {
	p.llamadas++
	return ports.PDFFirmaAnterior{Solicitud: q, Contenido: bytes.Clone(p.cuerpo)}, nil
}

type apuntesDescargaCargoPrueba struct {
	ordenes []vp.DatosOrdenIntentoAuditoria
}

func (a *apuntesDescargaCargoPrueba) AppendIntentoAuditoria(_ context.Context,
	o vp.OrdenIntentoAuditoria) (vp.AcuseIntentoAuditoria, error) {
	d, err := o.Datos()
	if err != nil {
		return vp.AcuseIntentoAuditoria{}, err
	}
	a.ordenes = append(a.ordenes, d)
	return vp.AcuseIntentoAuditoria{AuditoriaRef: "auditoria_descarga_cargo", Secuencia: int64(len(a.ordenes)),
		HuellaSHA256: strings.Repeat("c", 64), CorrelacionRef: d.Datos.CorrelacionRef,
		RegistradaEn: time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC)}, nil
}

// La fábrica nominal real decide si la descarga deja constancia: el Técnico,
// sin paso en la resolución, recibe 404 tras un apunte denegado y sin tocar
// historia ni PDF; Jefatura, con paso, recibe el PDF exacto sin apunte nuevo.
func TestFirmaNominalDescargaTecnicoAuditadoYJefaturaRecibePDF(t *testing.T) {
	e := nuevaCapsulaContextoFijadoPrueba(t, 0)
	p := e.peticion.Context().Value(clavePeticionNominalFirmaCT{}).(*peticionNominalFirmaCT)
	p.fuente.proceso = "vec-rrhh"
	pdf := []byte("%PDF-1.4\nresolucion-firmada")
	suma := sha256.Sum256(pdf)
	huella := hex.EncodeToString(suma[:])
	const expediente = "expediente:prueba"
	seleccion := consultafirmasv2.SeleccionDescarga{OrganizacionRef: "organizacion:central",
		Consulta: consultafirmasv2.Solicitud{ExpedienteRef: expediente, VersionExpediente: 7,
			Documento: "resolucion", PasoOrden: 1, ClaveIdempotencia: "consulta-descarga-000001",
			CatalogoHuella: strings.Repeat("a", 64), Via: ports.ViaFirmaCertificadoVEC}}
	historia := consultafirmasv2.Resultado{ExpedienteRef: expediente, Documento: "resolucion", VersionExpediente: 7,
		ExpedienteDocumentalRef: ports.ExpedienteDocumentalRef("organizacion:central", expediente),
		Firmas: []consultafirmasv2.Firma{{FirmaRef: "firma:ultima", ReciboRef: "recibo:firma:ultima",
			Documento: "resolucion", VersionExpediente: 6, Resultado: ctdomain.ResultadoFirmaFirmado,
			Custodiado:  &consultafirmasv2.Documento{Ref: "documento:firmado:ultimo", Version: 3, SHA256: huella},
			RevisionPDF: &consultafirmasv2.RevisionPDF{RevisionSHA256: huella, RevisionLongitud: uint64(len(pdf))}}}}
	const cuerpo = `{"expediente_ref":"expediente:prueba","version_observada":7,"documento":"resolucion","firma_ref":"firma:ultima"}`
	for _, caso := range []struct {
		cargo   string
		conPaso bool
		estado  int
	}{{"tecnico", false, http.StatusNotFound}, {"jefatura", true, http.StatusOK}} {
		t.Run(caso.cargo, func(t *testing.T) {
			selector := &selectorDescargaCargoPrueba{conPaso: caso.conPaso, salida: seleccion}
			lector := &historiaDescargaCargoPrueba{salida: historia}
			almacen := &pdfDescargaCargoPrueba{cuerpo: pdf}
			apuntes := &apuntesDescargaCargoPrueba{}
			servicio, err := consultafirmasv2.NuevaDescarga(selector, lector, almacen)
			if err != nil {
				t.Fatal(err)
			}
			h, err := httpinterno.NuevoManejadorDescargaFirmaR5V2(servicio, apuntes, p.fuente, p.fuente)
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(http.MethodPost, httpinterno.RutaDescargaFirmaR5V2, strings.NewReader(cuerpo))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Accept", "application/json")
			p.peticion = r
			ctx, err := vp.ConCorrelacionIncidenciasPeticion(context.WithValue(r.Context(), clavePeticionNominalFirmaCT{}, p))
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r.WithContext(ctx))
			if w.Code != caso.estado || selector.llamadas != 1 {
				t.Fatalf("descarga %s: HTTP %d, selector %d", caso.cargo, w.Code, selector.llamadas)
			}
			var respuesta struct {
				Data *struct {
					PDFBase64 string `json:"pdf_base64"`
					ReciboRef string `json:"recibo_ref"`
				} `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &respuesta); err != nil {
				t.Fatal(err)
			}
			if !caso.conPaso {
				if respuesta.Data != nil || lector.llamadas != 0 || almacen.llamadas != 0 || len(apuntes.ordenes) != 1 ||
					apuntes.ordenes[0].Datos.Resultado != core.ResultadoIntentoAuditoriaDenegado ||
					apuntes.ordenes[0].Datos.Accion != ports.AccionConsultarFirmasR5V2 ||
					apuntes.ordenes[0].Datos.RecursoRef != expediente {
					t.Fatalf("el rechazo del Técnico no quedó cerrado y auditado: historia %d, PDF %d, apuntes %d",
						lector.llamadas, almacen.llamadas, len(apuntes.ordenes))
				}
				return
			}
			if respuesta.Data == nil || len(apuntes.ordenes) != 0 || respuesta.Data.ReciboRef != "recibo:firma:ultima" {
				t.Fatal("Jefatura no recibió el documento con su recibo")
			}
			contenido, err := base64.StdEncoding.DecodeString(respuesta.Data.PDFBase64)
			if err != nil || !bytes.Equal(contenido, pdf) {
				t.Fatal("el PDF entregado a Jefatura no es el custodiado")
			}
		})
	}
}
