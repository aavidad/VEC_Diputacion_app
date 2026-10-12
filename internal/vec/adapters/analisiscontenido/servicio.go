package analisiscontenido

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

type ConfiguracionServicio struct {
	ConectorID, ClamdSocket string
	Control                 *ClamdControlado
	Bwrap, QPDF, Prlimit    string
	IdentidadesCliente      []string // URI SAN exactas; CN no autoriza.
	MaxEdadFirmas           time.Duration
}

type Servicio struct {
	conectorID, clamdSocket string
	control                 *ClamdControlado
	qpdf                    configuracionQPDF
	identidades             map[string]struct{}
	maxEdadFirmas           time.Duration
	ocupacion               chan struct{}
}

func NuevoServicio(c ConfiguracionServicio) (*Servicio, error) {
	if !referenciaSegura(c.ConectorID, 128) || c.ClamdSocket == "" || c.Control == nil || c.Control.socket != c.ClamdSocket || !c.Control.vivo() ||
		c.Bwrap == "" || c.QPDF == "" || c.Prlimit == "" ||
		c.MaxEdadFirmas <= 0 || c.MaxEdadFirmas > 72*time.Hour || len(c.IdentidadesCliente) == 0 {
		return nil, ErrConfiguracion
	}
	identidades := make(map[string]struct{}, len(c.IdentidadesCliente))
	for _, id := range c.IdentidadesCliente {
		if !strings.HasPrefix(id, "spiffe://") || len(id) > 256 {
			return nil, ErrConfiguracion
		}
		identidades[id] = struct{}{}
	}
	for _, ruta := range []string{c.Bwrap, c.QPDF, c.Prlimit} {
		f, err := os.Stat(ruta)
		if err != nil || f.IsDir() || f.Mode()&0111 == 0 {
			return nil, ErrConfiguracion
		}
	}
	return &Servicio{conectorID: c.ConectorID, clamdSocket: c.ClamdSocket, control: c.Control, qpdf: configuracionQPDF{Bwrap: c.Bwrap, QPDF: c.QPDF, Prlimit: c.Prlimit}, identidades: identidades, maxEdadFirmas: c.MaxEdadFirmas, ocupacion: make(chan struct{}, 2)}, nil
}

// TLSConfig exige certificados de cliente verificados por una CA privada.
// Las claves y certificados viven fuera de Git y no se incluyen en errores.
func (s *Servicio) TLSConfig(rutaCertificado, rutaClave, rutaCA string) (*tls.Config, error) {
	if s == nil || rutaCertificado == "" || rutaClave == "" || rutaCA == "" {
		return nil, ErrConfiguracion
	}
	certPEM, err := leerArchivoTLS(rutaCertificado, false)
	if err != nil {
		return nil, ErrConfiguracion
	}
	clavePEM, err := leerArchivoTLS(rutaClave, true)
	if err != nil {
		return nil, ErrConfiguracion
	}
	cert, err := tls.X509KeyPair(certPEM, clavePEM)
	if err != nil {
		return nil, ErrConfiguracion
	}
	ca, err := leerArchivoTLS(rutaCA, false)
	if err != nil {
		return nil, ErrConfiguracion
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return nil, ErrConfiguracion
	}
	return &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cert}, ClientCAs: pool, ClientAuth: tls.RequireAndVerifyClientCert}, nil
}

func (s *Servicio) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s == nil || !s.identidadPermitida(r) {
			http.Error(w, "denegado", http.StatusForbidden)
			return
		}
		switch r.URL.Path {
		case "/v1/capacidades":
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if !s.ocupar(w) {
				return
			}
			defer s.liberar()
			s.capacidades(w, r)
		case "/v1/analisis":
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if !s.ocupar(w) {
				return
			}
			defer s.liberar()
			s.analisis(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

func (s *Servicio) ocupar(w http.ResponseWriter) bool {
	select {
	case s.ocupacion <- struct{}{}:
		return true
	default:
		http.Error(w, "ocupado", http.StatusServiceUnavailable)
		return false
	}
}
func (s *Servicio) liberar() { <-s.ocupacion }

func (s *Servicio) identidadPermitida(r *http.Request) bool {
	if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.PeerCertificates) == 0 {
		return false
	}
	identidades := r.TLS.PeerCertificates[0].URIs
	if len(identidades) != 1 {
		return false
	}
	_, permitida := s.identidades[identidades[0].String()]
	return permitida
}

func (s *Servicio) capacidades(w http.ResponseWriter, r *http.Request) {
	if !s.control.vivo() {
		http.Error(w, "no_disponible", http.StatusServiceUnavailable)
		return
	}
	if _, err := versionClamd(r.Context(), s.clamdSocket, time.Now().UTC(), s.maxEdadFirmas); err != nil || s.sondaQPDF(r.Context()) != nil || !s.control.vivo() {
		http.Error(w, "no_disponible", http.StatusServiceUnavailable)
		return
	}
	cap := ports.CapacidadesAnalizadorContenido{ConectorID: s.conectorID, VersionConector: 1, AnalisisEnFlujo: true, CanalAutenticado: true, CifradoEnTransito: true, IdentidadMutua: true, ActualizacionFirmas: true, DetectaMalware: true, DetectaContenidoActivo: true, TamanoMaximo: tamanoMaximoPDF}
	responderJSON(w, cap)
}

func (s *Servicio) sondaQPDF(ctx context.Context) error {
	codigo, deteccion, err := inspeccionarPDF(ctx, pdfSonda, s.qpdf)
	if err != nil || deteccion != nil || codigo != "PDF_PASIVO" {
		return errParserPDF
	}
	return nil
}

func (s *Servicio) analisis(w http.ResponseWriter, r *http.Request) {
	if !s.control.vivo() {
		http.Error(w, "no_disponible", http.StatusServiceUnavailable)
		return
	}
	ctx, cancelar := context.WithTimeout(r.Context(), 40*time.Second)
	defer cancelar()
	inicio := time.Now().UTC()
	r.Body = http.MaxBytesReader(w, r.Body, tamanoMaximoPDF+(1<<20))
	lector, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "carga_invalida", http.StatusBadRequest)
		return
	}
	metadatos, err := lector.NextPart()
	if err != nil || metadatos.FormName() != "metadatos" {
		http.Error(w, "carga_invalida", http.StatusBadRequest)
		return
	}
	bruto, err := io.ReadAll(io.LimitReader(metadatos, limiteMetadatos+1))
	if err != nil || len(bruto) > limiteMetadatos {
		http.Error(w, "carga_invalida", http.StatusBadRequest)
		return
	}
	var meta solicitudRemota
	decoder := json.NewDecoder(bytes.NewReader(bruto))
	decoder.DisallowUnknownFields()
	var extra any
	if decoder.Decode(&meta) != nil || decoder.Decode(&extra) != io.EOF || meta.validar() != nil {
		http.Error(w, "carga_invalida", http.StatusBadRequest)
		return
	}
	archivo, err := lector.NextPart()
	if err != nil || archivo.FormName() != "archivo" {
		http.Error(w, "carga_invalida", http.StatusBadRequest)
		return
	}
	contenido, err := io.ReadAll(io.LimitReader(archivo, meta.Tamano+1))
	if err != nil || int64(len(contenido)) != meta.Tamano || huella(contenido) != meta.HuellaSHA256 {
		http.Error(w, "contenido_invalido", http.StatusUnprocessableEntity)
		return
	}
	parteExtra, err := lector.NextPart()
	if err != io.EOF || parteExtra != nil {
		http.Error(w, "carga_invalida", http.StatusBadRequest)
		return
	}
	resultado := s.analizarBytes(ctx, meta, contenido, inicio)
	if resultado.Validar() != nil || (resultado.Estado == ports.EstadoAnalisisContenidoLimpio && !s.control.vivo()) {
		http.Error(w, "resultado_invalido", http.StatusServiceUnavailable)
		return
	}
	responderJSON(w, resultado)
}

func (s *Servicio) analizarBytes(ctx context.Context, meta solicitudRemota, contenido []byte, inicio time.Time) ports.ResultadoAnalisisContenido {
	resultado := ports.ResultadoAnalisisContenido{
		Objeto: meta.Objeto, ConectorAlmacenID: meta.ConectorAlmacenID, HuellaObjetoSHA256: meta.HuellaSHA256, TamanoObjeto: meta.Tamano,
		MIMEDeclarado: meta.MIME, MIMEDetectado: "application/octet-stream", ConectorAnalizadorID: s.conectorID, VersionConector: 1,
		Estado: ports.EstadoAnalisisContenidoNoConcluyente, CodigoResultado: "NO_CONCLUYENTE", BytesAnalizados: int64(len(contenido)),
		AnalisisIniciadoEn: inicio, CorrelacionRef: meta.CorrelacionRef, AutorizacionRef: meta.AutorizacionRef, Finalidad: meta.Finalidad, Clasificacion: meta.Clasificacion,
	}
	if bytes.HasPrefix(contenido, []byte("%PDF-")) {
		resultado.MIMEDetectado = "application/pdf"
	}
	info, estado, err := analizarClamd(ctx, s.clamdSocket, contenido, s.maxEdadFirmas, time.Now().UTC())
	if err != nil || !s.control.vivo() {
		resultado.CodigoResultado = "AV_NO_CONCLUYENTE"
	} else {
		resultado.MotorRef = "clamav"
		resultado.VersionMotor = info.motorVersion
		resultado.FirmasRef = info.firmasRef
		switch estado {
		case avMalicioso:
			resultado.Estado = ports.EstadoAnalisisContenidoMalicioso
			resultado.CodigoResultado = "MALWARE_DETECTADO"
			resultado.Detecciones = []ports.DeteccionContenido{{Clase: ports.ClaseDeteccionMalware, Codigo: "MALWARE_DETECTADO", FirmaRef: info.firmaDeteccion}}
		case avLimpio:
			if resultado.MIMEDetectado != "application/pdf" {
				resultado.CodigoResultado = "FORMATO_NO_PDF"
			} else {
				codigo, deteccion, errorPDF := inspeccionarPDF(ctx, contenido, s.qpdf)
				if errorPDF != nil {
					resultado.CodigoResultado = "PDF_NO_CONCLUYENTE"
				} else if deteccion != nil {
					resultado.Estado = ports.EstadoAnalisisContenidoSospechoso
					resultado.CodigoResultado = codigo
					resultado.Detecciones = []ports.DeteccionContenido{*deteccion}
				} else {
					resultado.Estado = ports.EstadoAnalisisContenidoLimpio
					resultado.CodigoResultado = "PDF_PASIVO"
				}
			}
		}
	}
	resultado.AnalisisCompletadoEn = time.Now().UTC()
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		resultado.Estado = ports.EstadoAnalisisContenidoError
		resultado.CodigoResultado = "EVIDENCIA_NO_DISPONIBLE"
	}
	resultado.EvidenciaRef = "analisis:" + hex.EncodeToString(nonce[:])
	bruto, _ := json.Marshal(resultado)
	suma := sha256.Sum256(bruto)
	resultado.HuellaEvidenciaSHA256 = hex.EncodeToString(suma[:])
	return resultado
}

func responderJSON(w http.ResponseWriter, valor any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(valor)
}
