// Package mayan adapta exclusivamente la ingestión de originales ya admitidos
// por VEC. La cuarentena, el antivirus, la autorización y el plan durable
// pertenecen al caso de uso y al almacén transitorio; Mayan no los sustituye.
package mayan

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrConfiguracion = errors.New("mayan: configuracion invalida")
	ErrOrden         = errors.New("mayan: orden de ingesta invalida")
	ErrIntegridad    = errors.New("mayan: original no coincide con la huella esperada")
	ErrIndeterminado = errors.New("mayan: efecto de ingesta indeterminado; reconciliar antes de reintentar")
	ErrRespuesta     = errors.New("mayan: respuesta no verificable")
	ErrNoDisponible  = errors.New("mayan: servicio no disponible")
)

const limiteRespuesta = 1 << 20
const limitePaginas = 1000

// OrdenIngestaValidada solo se construye en el lado confiable, tras validar
// contra las autoridades de VEC el recibo de cuarentena y el antivirus limpio.
// Este constructor valida la forma, pero no acredita por sí solo esos hechos.
type OrdenIngestaValidada struct {
	planRef, evidenciaRef, huella, mime string
	tamano                              int64
}

func NuevaOrdenIngestaValidada(planRef, evidenciaRef, huellaSHA256 string, tamano int64, mime string) (OrdenIngestaValidada, error) {
	if !referenciaValida(planRef) || !referenciaValida(evidenciaRef) || !huellaValida(huellaSHA256) ||
		tamano < 1 || tamano > 64<<20 || !mimeValido(mime) {
		return OrdenIngestaValidada{}, ErrOrden
	}
	return OrdenIngestaValidada{planRef: planRef, evidenciaRef: evidenciaRef, huella: strings.ToLower(huellaSHA256), tamano: tamano, mime: mime}, nil
}

// Documento guarda únicamente identificadores opacos de Mayan y propiedades
// verificadas del original. Nunca guarda una URL ni bytes en PostgreSQL.
type Documento struct {
	DocumentoID  string
	ArchivoID    string
	VersionID    string
	HuellaSHA256 string
	Tamano       int64
	MIME         string
}

// Configuracion se inyecta desde la composición, no desde una petición web.
// ClienteHTTP puede ser nil; se configura sin seguir redirecciones.
type Configuracion struct {
	URLBase         string
	Token           string
	FuenteID        string
	TipoDocumentoID string
	ClienteHTTP     *http.Client
}

func (Configuracion) String() string   { return "configuracion_mayan_redactada" }
func (Configuracion) GoString() string { return "configuracion_mayan_redactada" }
func (Configuracion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, "configuracion_mayan_redactada")
}
func (Configuracion) MarshalJSON() ([]byte, error) {
	return []byte(`"configuracion_mayan_redactada"`), nil
}
func (Configuracion) LogValue() slog.Value { return slog.StringValue("configuracion_mayan_redactada") }

type Cliente struct {
	base, token, fuenteID, tipoID string
	http                          *http.Client
}

func (*Cliente) String() string   { return "cliente_mayan_redactado" }
func (*Cliente) GoString() string { return "cliente_mayan_redactado" }
func (*Cliente) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, "cliente_mayan_redactado")
}
func (*Cliente) LogValue() slog.Value { return slog.StringValue("cliente_mayan_redactado") }

func Nuevo(c Configuracion) (*Cliente, error) {
	u, err := url.Parse(c.URLBase)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" ||
		!identificadorValido(c.FuenteID) || !identificadorValido(c.TipoDocumentoID) ||
		strings.TrimSpace(c.Token) == "" || strings.ContainsAny(c.Token, "\r\n") {
		return nil, ErrConfiguracion
	}
	h := c.ClienteHTTP
	if h == nil {
		h = &http.Client{Timeout: 30 * time.Second}
	}
	copia := *h
	copia.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if copia.Timeout <= 0 {
		copia.Timeout = 30 * time.Second
	}
	return &Cliente{base: strings.TrimRight(u.String(), "/"), token: c.Token, fuenteID: c.FuenteID, tipoID: c.TipoDocumentoID, http: &copia}, nil
}

// IngerirOriginal realiza exactamente un POST. Cualquier fallo después de
// empezar el envío, incluso timeout o respuesta inválida, queda indeterminado:
// el llamante conserva el plan y no repite el POST automáticamente.
func (c *Cliente) IngerirOriginal(ctx context.Context, orden OrdenIngestaValidada, contenido io.Reader) (Documento, error) {
	if c == nil || ctx == nil || contenido == nil || !orden.valida() {
		return Documento{}, ErrOrden
	}
	// Se coteja el original completo antes de permitir que llegue a Mayan.
	bytesOriginal, err := io.ReadAll(io.LimitReader(contenido, orden.tamano+1))
	if err != nil || int64(len(bytesOriginal)) != orden.tamano || digest(bytesOriginal) != orden.huella {
		return Documento{}, ErrIntegridad
	}
	var cuerpo bytes.Buffer
	m := multipart.NewWriter(&cuerpo)
	// SourceBackendActionSerializer acepta un único campo JSON `arguments`.
	// Los argumentos planos en multipart son ignorados por Mayan 4.12.2.
	if err := m.WriteField("arguments", `{"document_type_id":`+c.tipoID+`,"immediate_mode":true}`); err != nil {
		return Documento{}, ErrOrden
	}
	parte, err := m.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {`form-data; name="file"; filename="original.bin"`},
		"Content-Type":        {orden.mime},
	})
	if err != nil {
		return Documento{}, ErrOrden
	}
	if _, err = parte.Write(bytesOriginal); err != nil {
		return Documento{}, ErrOrden
	}
	if err = m.Close(); err != nil {
		return Documento{}, ErrOrden
	}
	endpoint := c.base + "/api/v4/sources/" + c.fuenteID + "/actions/document_upload/execute/"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &cuerpo)
	if err != nil {
		return Documento{}, ErrOrden
	}
	req.Header.Set("Authorization", "Token "+c.token)
	req.Header.Set("Content-Type", m.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return Documento{}, ErrIndeterminado
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Documento{}, ErrIndeterminado
	}
	var datos struct {
		ID json.Number `json:"id"`
	}
	if err := decodificarJSONLimitado(resp.Body, &datos); err != nil || !identificadorValido(string(datos.ID)) {
		return Documento{}, ErrIndeterminado
	}
	doc := Documento{DocumentoID: string(datos.ID), HuellaSHA256: orden.huella, Tamano: orden.tamano, MIME: orden.mime}
	doc, err = c.ReconciliarOriginal(ctx, doc)
	if err != nil {
		// El ID se devuelve para reconciliar; la custodia aún no está confirmada.
		if errors.Is(err, ErrIntegridad) {
			return doc, ErrIntegridad
		}
		return doc, ErrIndeterminado
	}
	return doc, nil
}

// ReconciliarOriginal consulta el documento conocido tras un POST confirmado.
// Nunca repite la ingesta; espera de forma acotada el archivo creado en
// segundo plano por Mayan y verifica el original antes de confirmar custodia.
func (c *Cliente) ReconciliarOriginal(ctx context.Context, doc Documento) (Documento, error) {
	if c == nil || ctx == nil || !doc.validoParcial() {
		return Documento{}, ErrOrden
	}
	// Un par ya conocido es inmutable. No se sustituye por file_latest ni
	// version_active, que pueden cambiar tras otra carga concurrente.
	fijado := doc.valido()
	if (doc.ArchivoID == "") != (doc.VersionID == "") || (doc.ArchivoID != "" && !fijado) {
		return doc, ErrOrden
	}
	espera, cancelar := context.WithTimeout(ctx, 10*time.Second)
	defer cancelar()
	for {
		completo := doc
		var err error
		if !fijado {
			completo, err = c.consultarDocumento(espera, doc)
		}
		if err == nil {
			vinculoConfirmado := false
			err = c.verificarVinculoPaginas(espera, completo)
			if err == nil {
				vinculoConfirmado = true
				_, err = c.LeerOriginal(espera, completo)
				if err == nil && !fijado {
					// Si el puntero activo cambió durante la verificación, no
					// se adjudica la versión encontrada al POST inicial.
					var actual Documento
					actual, err = c.consultarDocumento(espera, doc)
					if err == nil && (actual.ArchivoID != completo.ArchivoID || actual.VersionID != completo.VersionID) {
						return doc, ErrIndeterminado
					}
				}
				if err == nil {
					return completo, nil
				}
			}
			// Un archivo alterado no se convertirá en válido con otra lectura.
			if errors.Is(err, ErrIntegridad) {
				if vinculoConfirmado {
					return completo, ErrIntegridad
				}
				if fijado {
					return doc, ErrIntegridad
				}
				return doc, ErrIndeterminado
			}
		}
		select {
		case <-espera.Done():
			return doc, ErrIndeterminado
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// verificarVinculoPaginas prueba la relación document file -> document
// version que Mayan 4.12.2 publica como DocumentVersionPage.content_object.
// Se exige el conjunto completo y no vacío; un PDF sin páginas aún no está
// listo. Otros formatos sin páginas quedan pendientes por diseño.
func (c *Cliente) verificarVinculoPaginas(ctx context.Context, doc Documento) error {
	versiones, err := c.listarPaginas(ctx, "/api/v4/documents/"+doc.DocumentoID+"/versions/"+doc.VersionID+"/pages/")
	if err != nil {
		return err
	}
	archivos, err := c.listarPaginas(ctx, "/api/v4/documents/"+doc.DocumentoID+"/files/"+doc.ArchivoID+"/pages/")
	if err != nil {
		return err
	}
	if len(versiones) == 0 || len(versiones) != len(archivos) {
		return ErrIntegridad
	}
	idsArchivo := make(map[string]struct{}, len(archivos))
	for _, bruto := range archivos {
		var pagina struct {
			ID             json.Number `json:"id"`
			DocumentFileID json.Number `json:"document_file_id"`
		}
		if json.Unmarshal(bruto, &pagina) != nil || !identificadorValido(string(pagina.ID)) || string(pagina.DocumentFileID) != doc.ArchivoID {
			return ErrRespuesta
		}
		id := string(pagina.ID)
		if _, existe := idsArchivo[id]; existe {
			return ErrRespuesta
		}
		idsArchivo[id] = struct{}{}
	}
	idsVersion := make(map[string]struct{}, len(versiones))
	for _, bruto := range versiones {
		var pagina struct {
			ContentType struct {
				AppLabel string `json:"app_label"`
				Model    string `json:"model"`
			} `json:"content_type"`
			ObjectID json.Number `json:"object_id"`
		}
		if json.Unmarshal(bruto, &pagina) != nil || pagina.ContentType.AppLabel != "documents" ||
			pagina.ContentType.Model != "documentfilepage" || !identificadorValido(string(pagina.ObjectID)) {
			return ErrRespuesta
		}
		id := string(pagina.ObjectID)
		if _, existe := idsVersion[id]; existe {
			return ErrRespuesta
		}
		idsVersion[id] = struct{}{}
		if _, existe := idsArchivo[id]; !existe {
			return ErrIntegridad
		}
	}
	return nil
}

func (c *Cliente) listarPaginas(ctx context.Context, ruta string) ([]json.RawMessage, error) {
	var todas []json.RawMessage
	cuentaEsperada := 0
	for numero := 1; numero <= limitePaginas/100; numero++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+ruta+"?page_size=100&page="+strconv.Itoa(numero), nil)
		if err != nil {
			return nil, ErrRespuesta
		}
		req.Header.Set("Authorization", "Token "+c.token)
		req.Header.Set("Accept", "application/json")
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, ErrNoDisponible
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, ErrNoDisponible
		}
		var pagina struct {
			Count   int               `json:"count"`
			Next    *string           `json:"next"`
			Results []json.RawMessage `json:"results"`
		}
		err = decodificarJSONLimitado(resp.Body, &pagina)
		resp.Body.Close()
		if numero == 1 {
			cuentaEsperada = pagina.Count
		}
		if err != nil || pagina.Count < 1 || pagina.Count > limitePaginas || len(pagina.Results) == 0 ||
			pagina.Count != cuentaEsperada || len(pagina.Results) > 100 || len(todas)+len(pagina.Results) > pagina.Count {
			return nil, ErrRespuesta
		}
		todas = append(todas, pagina.Results...)
		if pagina.Next == nil {
			if len(todas) != pagina.Count {
				return nil, ErrRespuesta
			}
			return todas, nil
		}
		if len(todas) == pagina.Count {
			return nil, ErrRespuesta
		}
	}
	return nil, ErrRespuesta
}

func (c *Cliente) consultarDocumento(ctx context.Context, doc Documento) (Documento, error) {
	endpoint := c.base + "/api/v4/documents/" + doc.DocumentoID + "/"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return doc, ErrRespuesta
	}
	req.Header.Set("Authorization", "Token "+c.token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return doc, ErrNoDisponible
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return doc, ErrNoDisponible
	}
	var datos struct {
		ID         json.Number `json:"id"`
		FileLatest *struct {
			ID json.Number `json:"id"`
		} `json:"file_latest"`
		VersionActive *struct {
			ID json.Number `json:"id"`
		} `json:"version_active"`
	}
	if err := decodificarJSONLimitado(resp.Body, &datos); err != nil || string(datos.ID) != doc.DocumentoID {
		return doc, ErrRespuesta
	}
	if datos.FileLatest == nil || datos.VersionActive == nil {
		return doc, ErrNoDisponible
	}
	if !identificadorValido(string(datos.FileLatest.ID)) || !identificadorValido(string(datos.VersionActive.ID)) {
		return doc, ErrRespuesta
	}
	doc.ArchivoID, doc.VersionID = string(datos.FileLatest.ID), string(datos.VersionActive.ID)
	return doc, nil
}

func decodificarJSONLimitado(lector io.Reader, destino any) error {
	contenido, err := io.ReadAll(io.LimitReader(lector, limiteRespuesta+1))
	if err != nil || len(contenido) > limiteRespuesta || json.Unmarshal(contenido, destino) != nil {
		return ErrRespuesta
	}
	return nil
}

// LeerOriginal descarga el archivo fuente, nunca el PDF exportado de una
// versión, y coteja tamaño y SHA256 antes de exponer los bytes al llamante.
func (c *Cliente) LeerOriginal(ctx context.Context, doc Documento) ([]byte, error) {
	if c == nil || ctx == nil || !doc.valido() {
		return nil, ErrOrden
	}
	endpoint := c.base + "/api/v4/documents/" + doc.DocumentoID + "/files/" + doc.ArchivoID + "/download/"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, ErrOrden
	}
	req.Header.Set("Authorization", "Token "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, ErrNoDisponible
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, ErrNoDisponible
	}
	contenido, err := io.ReadAll(io.LimitReader(resp.Body, doc.Tamano+1))
	if err != nil {
		return nil, ErrNoDisponible
	}
	if int64(len(contenido)) != doc.Tamano || digest(contenido) != doc.HuellaSHA256 {
		return nil, ErrIntegridad
	}
	return contenido, nil
}

func (o OrdenIngestaValidada) valida() bool {
	return referenciaValida(o.planRef) && referenciaValida(o.evidenciaRef) && huellaValida(o.huella) &&
		o.tamano > 0 && o.tamano <= 64<<20 && mimeValido(o.mime)
}

func (d Documento) valido() bool {
	return d.validoParcial() && identificadorValido(d.ArchivoID) && identificadorValido(d.VersionID)
}

func (d Documento) validoParcial() bool {
	return identificadorValido(d.DocumentoID) && huellaValida(d.HuellaSHA256) && d.Tamano > 0 && d.Tamano <= 64<<20 && mimeValido(d.MIME)
}

func identificadorValido(s string) bool {
	n, err := strconv.ParseUint(s, 10, 64)
	return err == nil && n > 0 && fmt.Sprint(n) == s
}

func referenciaValida(s string) bool {
	if len(s) < 8 || len(s) > 256 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == ':' || r == '.') {
			return false
		}
	}
	return true
}

func huellaValida(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func mimeValido(s string) bool {
	if len(s) < 3 || len(s) > 127 || !strings.Contains(s, "/") {
		return false
	}
	mediaType, parametros, err := mime.ParseMediaType(s)
	return err == nil && mediaType == s && len(parametros) == 0
}

func digest(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}
