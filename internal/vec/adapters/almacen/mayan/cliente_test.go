package mayan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func clientePrueba(t *testing.T, handler http.HandlerFunc) *Cliente {
	t.Helper()
	s := httptest.NewTLSServer(handler)
	t.Cleanup(s.Close)
	c, err := Nuevo(Configuracion{URLBase: s.URL, Token: "token-sintetico", FuenteID: "2", TipoDocumentoID: "3", ClienteHTTP: s.Client()})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func ordenPrueba(t *testing.T, contenido []byte) OrdenIngestaValidada {
	t.Helper()
	o, err := NuevaOrdenIngestaValidada("plan:synthetic-1", "scan:clean-1", digest(contenido), int64(len(contenido)), "application/pdf")
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func responderPaginas(w http.ResponseWriter, r *http.Request) bool {
	switch r.URL.Path {
	case "/api/v4/documents/5/versions/7/pages/":
		_, _ = io.WriteString(w, `{"count":1,"next":null,"results":[{"content_type":{"app_label":"documents","model":"documentfilepage"},"object_id":201}]}`)
	case "/api/v4/documents/5/files/6/pages/":
		_, _ = io.WriteString(w, `{"count":1,"next":null,"results":[{"id":201,"document_file_id":6}]}`)
	default:
		return false
	}
	return true
}

func TestIngestaVerificaOriginalYNoUsaRutaObsoleta(t *testing.T) {
	contenido := []byte("pdf-sintetico")
	var posts atomic.Int32
	c := clientePrueba(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Token token-sintetico" {
			t.Error("falta token")
		}
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v4/sources/2/actions/document_upload/execute/":
			posts.Add(1)
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Error(err)
			}
			var args map[string]any
			if err := json.Unmarshal([]byte(r.FormValue("arguments")), &args); err != nil || args["immediate_mode"] != true || args["document_type_id"] != float64(3) {
				t.Errorf("argumentos de carga inesperados: %v, %v", args, err)
			}
			f, _, err := r.FormFile("file")
			if err != nil {
				t.Error(err)
				return
			}
			defer f.Close()
			recibido, _ := io.ReadAll(f)
			if !bytes.Equal(recibido, contenido) {
				t.Error("bytes alterados")
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":5,"file_latest":null,"version_active":null}`)
		case "GET /api/v4/documents/5/":
			_, _ = io.WriteString(w, `{"id":5,"file_latest":{"id":6},"version_active":{"id":7}}`)
		case "GET /api/v4/documents/5/files/6/download/":
			_, _ = w.Write(contenido)
		default:
			if !responderPaginas(w, r) {
				t.Errorf("ruta inesperada %s %s", r.Method, r.URL.Path)
				http.NotFound(w, r)
			}
		}
	})
	doc, err := c.IngerirOriginal(context.Background(), ordenPrueba(t, contenido), bytes.NewReader(contenido))
	if err != nil || doc.DocumentoID != "5" || doc.ArchivoID != "6" || doc.VersionID != "7" || posts.Load() != 1 {
		t.Fatalf("documento=%+v err=%v posts=%d", doc, err, posts.Load())
	}
}

func TestIngestaAlteradaNoHacePOST(t *testing.T) {
	contenido := []byte("pdf-sintetico")
	var posts atomic.Int32
	c := clientePrueba(t, func(w http.ResponseWriter, r *http.Request) { posts.Add(1) })
	_, err := c.IngerirOriginal(context.Background(), ordenPrueba(t, contenido), strings.NewReader("contenido-distinto"))
	if !errors.Is(err, ErrIntegridad) || posts.Load() != 0 {
		t.Fatalf("err=%v posts=%d", err, posts.Load())
	}
}

func TestTimeoutPOSTIndeterminadoSinReenvio(t *testing.T) {
	contenido := []byte("pdf-sintetico")
	var posts atomic.Int32
	c, err := Nuevo(Configuracion{URLBase: "https://mayan.example.test", Token: "token-sintetico", FuenteID: "2", TipoDocumentoID: "3", ClienteHTTP: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		posts.Add(1)
		return nil, context.DeadlineExceeded
	})}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.IngerirOriginal(context.Background(), ordenPrueba(t, contenido), bytes.NewReader(contenido))
	if !errors.Is(err, ErrIndeterminado) || posts.Load() != 1 {
		t.Fatalf("err=%v posts=%d", err, posts.Load())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestLecturaAlteradaDeniegaYConservaID(t *testing.T) {
	contenido := []byte("pdf-sintetico")
	c := clientePrueba(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_, _ = io.WriteString(w, `{"id":5}`)
			return
		}
		if r.URL.Path == "/api/v4/documents/5/" {
			_, _ = io.WriteString(w, `{"id":5,"file_latest":{"id":6},"version_active":{"id":7}}`)
			return
		}
		if responderPaginas(w, r) {
			return
		}
		_, _ = io.WriteString(w, "pdf-adulterado")
	})
	doc, err := c.IngerirOriginal(context.Background(), ordenPrueba(t, contenido), bytes.NewReader(contenido))
	if !errors.Is(err, ErrIntegridad) || doc.DocumentoID != "5" || doc.ArchivoID != "6" {
		t.Fatalf("doc=%+v err=%v", doc, err)
	}
	if _, err := c.LeerOriginal(context.Background(), doc); !errors.Is(err, ErrIntegridad) {
		t.Fatalf("lectura=%v", err)
	}
}

func TestArchivoAsincronoSoloReintentaGET(t *testing.T) {
	contenido := []byte("pdf-sintetico")
	var posts, consultas atomic.Int32
	c := clientePrueba(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v4/sources/2/actions/document_upload/execute/":
			posts.Add(1)
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":5,"file_latest":null}`)
		case "GET /api/v4/documents/5/":
			if consultas.Add(1) == 1 {
				_, _ = io.WriteString(w, `{"id":5,"file_latest":null,"version_active":null}`)
			} else {
				_, _ = io.WriteString(w, `{"id":5,"file_latest":{"id":6},"version_active":{"id":7}}`)
			}
		case "GET /api/v4/documents/5/files/6/download/":
			_, _ = w.Write(contenido)
		default:
			if !responderPaginas(w, r) {
				t.Errorf("ruta inesperada %s %s", r.Method, r.URL.Path)
				http.NotFound(w, r)
			}
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	doc, err := c.IngerirOriginal(ctx, ordenPrueba(t, contenido), bytes.NewReader(contenido))
	if err != nil || doc.VersionID != "7" || posts.Load() != 1 || consultas.Load() != 3 {
		t.Fatalf("doc=%+v err=%v posts=%d consultas=%d", doc, err, posts.Load(), consultas.Load())
	}
}

func TestRedireccionNoExponeTokenNiReenviaOriginal(t *testing.T) {
	contenido := []byte("pdf-sintetico")
	var destinoLlamado atomic.Bool
	destino := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		destinoLlamado.Store(true)
	}))
	defer destino.Close()
	c := clientePrueba(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destino.URL+"/recibir", http.StatusTemporaryRedirect)
	})
	_, err := c.IngerirOriginal(context.Background(), ordenPrueba(t, contenido), bytes.NewReader(contenido))
	if !errors.Is(err, ErrIndeterminado) || destinoLlamado.Load() {
		t.Fatalf("err=%v destino=%v", err, destinoLlamado.Load())
	}
}

func TestConfiguracionNoExponeTokenEnFormatoNiJSON(t *testing.T) {
	cfg := Configuracion{URLBase: "https://mayan.example.test", Token: "secreto-sintetico", FuenteID: "2", TipoDocumentoID: "3"}
	serializada, err := json.Marshal(cfg)
	if err != nil || strings.Contains(fmt.Sprintf("%+v %#v %s", cfg, cfg, serializada), cfg.Token) {
		t.Fatalf("representacion insegura: %v", err)
	}
}

func TestVersionAjenaConBytesIgualesQuedaIndeterminada(t *testing.T) {
	contenido := []byte("pdf-sintetico")
	var descargas atomic.Int32
	c := clientePrueba(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/sources/2/actions/document_upload/execute/":
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":5}`)
		case "/api/v4/documents/5/":
			_, _ = io.WriteString(w, `{"id":5,"file_latest":{"id":6},"version_active":{"id":7}}`)
		case "/api/v4/documents/5/versions/7/pages/":
			_, _ = io.WriteString(w, `{"count":1,"next":null,"results":[{"content_type":{"app_label":"documents","model":"documentfilepage"},"object_id":999}]}`)
		case "/api/v4/documents/5/files/6/pages/":
			_, _ = io.WriteString(w, `{"count":1,"next":null,"results":[{"id":201,"document_file_id":6}]}`)
		case "/api/v4/documents/5/files/6/download/":
			descargas.Add(1)
			_, _ = w.Write(contenido)
		default:
			http.NotFound(w, r)
		}
	})
	doc, err := c.IngerirOriginal(context.Background(), ordenPrueba(t, contenido), bytes.NewReader(contenido))
	if !errors.Is(err, ErrIndeterminado) || doc.DocumentoID != "5" || doc.VersionID != "" || descargas.Load() != 0 {
		t.Fatalf("doc=%+v err=%v descargas=%d", doc, err, descargas.Load())
	}
}

func TestCambioConcurrenteNoAdjudicaVersion(t *testing.T) {
	contenido := []byte("pdf-sintetico")
	var consultas atomic.Int32
	c := clientePrueba(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/sources/2/actions/document_upload/execute/":
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":5}`)
		case "/api/v4/documents/5/":
			if consultas.Add(1) == 1 {
				_, _ = io.WriteString(w, `{"id":5,"file_latest":{"id":6},"version_active":{"id":7}}`)
			} else {
				_, _ = io.WriteString(w, `{"id":5,"file_latest":{"id":8},"version_active":{"id":9}}`)
			}
		case "/api/v4/documents/5/files/6/download/":
			_, _ = w.Write(contenido)
		default:
			if !responderPaginas(w, r) {
				http.NotFound(w, r)
			}
		}
	})
	doc, err := c.IngerirOriginal(context.Background(), ordenPrueba(t, contenido), bytes.NewReader(contenido))
	if !errors.Is(err, ErrIndeterminado) || doc.DocumentoID != "5" || doc.ArchivoID != "" || doc.VersionID != "" || consultas.Load() != 2 {
		t.Fatalf("doc=%+v err=%v consultas=%d", doc, err, consultas.Load())
	}
}

func TestIDsConocidosNoSeSustituyenPorUltimos(t *testing.T) {
	contenido := []byte("pdf-sintetico")
	var consultasActual atomic.Int32
	c := clientePrueba(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/documents/5/" {
			consultasActual.Add(1)
			_, _ = io.WriteString(w, `{"id":5,"file_latest":{"id":8},"version_active":{"id":9}}`)
			return
		}
		if r.URL.Path == "/api/v4/documents/5/files/6/download/" {
			_, _ = w.Write(contenido)
			return
		}
		if !responderPaginas(w, r) {
			http.NotFound(w, r)
		}
	})
	doc := Documento{DocumentoID: "5", ArchivoID: "6", VersionID: "7", HuellaSHA256: digest(contenido), Tamano: int64(len(contenido)), MIME: "application/pdf"}
	recuperado, err := c.ReconciliarOriginal(context.Background(), doc)
	if err != nil || recuperado != doc || consultasActual.Load() != 0 {
		t.Fatalf("recuperado=%+v err=%v consultas=%d", recuperado, err, consultasActual.Load())
	}
}

func TestPaginasIncompletasNoConfirmanVersion(t *testing.T) {
	contenido := []byte("pdf-sintetico")
	casos := []struct{ nombre, respuesta string }{
		{"cero paginas", `{"count":0,"next":null,"results":[]}`},
		{"paginacion incompleta", `{"count":2,"next":null,"results":[{"content_type":{"app_label":"documents","model":"documentfilepage"},"object_id":201}]}`},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/api/v4/sources/2/actions/document_upload/execute/":
						w.WriteHeader(http.StatusCreated)
						_, _ = io.WriteString(w, `{"id":5}`)
					case "/api/v4/documents/5/":
						_, _ = io.WriteString(w, `{"id":5,"file_latest":{"id":6},"version_active":{"id":7}}`)
					case "/api/v4/documents/5/versions/7/pages/":
						_, _ = io.WriteString(w, caso.respuesta)
					case "/api/v4/documents/5/files/6/download/":
						_, _ = w.Write(contenido)
					default:
						if !responderPaginas(w, r) {
							http.NotFound(w, r)
						}
					}
				})
				// Las respuestas son síncronas: solo la espera de reconciliación
				// consume el plazo virtual, sin depender de TLS ni del ejecutor.
				c, err := Nuevo(Configuracion{URLBase: "https://mayan.example.test", Token: "token-sintetico", FuenteID: "2", TipoDocumentoID: "3", ClienteHTTP: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					w := httptest.NewRecorder()
					handler.ServeHTTP(w, r)
					return w.Result(), nil
				})}})
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Millisecond)
				defer cancelar()
				doc, err := c.IngerirOriginal(ctx, ordenPrueba(t, contenido), bytes.NewReader(contenido))
				if !errors.Is(err, ErrIndeterminado) || doc.DocumentoID != "5" || doc.VersionID != "" {
					t.Fatalf("doc=%+v err=%v", doc, err)
				}
			})
		})
	}
}

func TestPaginacionCompletaEnDosRespuestas(t *testing.T) {
	contenido := []byte("pdf-sintetico")
	var paginasVersion, paginasArchivo atomic.Int32
	c := clientePrueba(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/documents/5/files/6/download/" {
			_, _ = w.Write(contenido)
			return
		}
		if r.URL.Query().Get("page_size") != "100" {
			t.Errorf("tamaño de página inesperado: %s", r.URL.RawQuery)
		}
		pagina := r.URL.Query().Get("page")
		var resultados []map[string]any
		inicio, fin := 201, 301
		var siguiente any = "https://mayan.example.test/continuacion"
		if pagina == "2" {
			inicio, fin, siguiente = 301, 302, nil
		} else if pagina != "1" {
			t.Errorf("página inesperada: %s", pagina)
			http.NotFound(w, r)
			return
		}
		for id := inicio; id < fin; id++ {
			switch r.URL.Path {
			case "/api/v4/documents/5/versions/7/pages/":
				resultados = append(resultados, map[string]any{"content_type": map[string]string{"app_label": "documents", "model": "documentfilepage"}, "object_id": id})
			case "/api/v4/documents/5/files/6/pages/":
				resultados = append(resultados, map[string]any{"id": id, "document_file_id": 6})
			default:
				http.NotFound(w, r)
				return
			}
		}
		if strings.Contains(r.URL.Path, "/versions/") {
			paginasVersion.Add(1)
		} else {
			paginasArchivo.Add(1)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"count": 101, "next": siguiente, "results": resultados})
	})
	doc := Documento{DocumentoID: "5", ArchivoID: "6", VersionID: "7", HuellaSHA256: digest(contenido), Tamano: int64(len(contenido)), MIME: "application/pdf"}
	recuperado, err := c.ReconciliarOriginal(context.Background(), doc)
	if err != nil || recuperado != doc || paginasVersion.Load() != 2 || paginasArchivo.Load() != 2 {
		t.Fatalf("recuperado=%+v err=%v version=%d archivo=%d", recuperado, err, paginasVersion.Load(), paginasArchivo.Load())
	}
}
