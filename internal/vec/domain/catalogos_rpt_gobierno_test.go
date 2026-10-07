package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func contenidoPublicarRPTPrueba(t *testing.T) ContenidoGobiernoCategoriaRPT {
	t.Helper()
	borrador := catalogoConfigurablePrueba()
	for i := range borrador.Entradas {
		borrador.Entradas[i].Atributos["estado"] = "habilitada"
	}
	publicado, err := borrador.Publicar("responsable-configuracion-2", "aprobacion-1", "Revision funcional superada", borrador.CreadoEn.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := json.Marshal(publicado)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(doc)
	return ContenidoGobiernoCategoriaRPT{
		Accion: AccionGobiernoCategoriaRPTPublicar, CatalogoID: publicado.ID,
		ModuloID: publicado.ModuloID, Version: publicado.Version,
		DocumentoCanonico: ptrGobiernoRPT(string(doc)), DocumentoHuellaSHA256: ptrGobiernoRPT(hex.EncodeToString(h[:])),
		PreimagenesControl:      map[string]PreimagenControlGobiernoCategoriaRPT{},
		PreimagenesHuellaSHA256: strings.Repeat("a", 64),
		MotivoRef:               "motivos:1:motivo_0123456789abcdef0123456789abcdef",
		FuenteRef:               publicado.FuenteRef,
	}
}

func ptrGobiernoRPT[T any](v T) *T { return &v }

func TestContenidoGobiernoCategoriaRPTExigeEditorSeparadoYDocumentoIntacto(t *testing.T) {
	c := contenidoPublicarRPTPrueba(t)
	if err := c.ValidarParaEditor("tecnico-configuracion-1"); err != nil {
		t.Fatal(err)
	}
	borrador := c
	borrador.DocumentoHuellaSHA256 = nil
	borrador.PreimagenesHuellaSHA256 = ""
	completado, err := borrador.PrepararBorradorParaEditor("tecnico-configuracion-1")
	if err != nil || completado.PreimagenesHuellaSHA256 != "" ||
		*completado.DocumentoHuellaSHA256 != *c.DocumentoHuellaSHA256 {
		t.Fatalf("borrador preparado incorrectamente: %v", err)
	}
	if err := c.ValidarParaEditor("responsable-configuracion-2"); !errors.Is(err, ErrGobiernoCategoriaRPTInvalido) {
		t.Fatalf("autopublicacion: %v", err)
	}
	alterado := c
	alterado.DocumentoCanonico = ptrGobiernoRPT(*c.DocumentoCanonico + " ")
	if err := alterado.ValidarParaEditor("tecnico-configuracion-1"); !errors.Is(err, ErrGobiernoCategoriaRPTInvalido) {
		t.Fatalf("huella de documento alterada: %v", err)
	}
}

func TestGobiernoCategoriaRPTRechazaDocumentoExcesivoAntesDeCalcularHuella(t *testing.T) {
	c := contenidoPublicarRPTPrueba(t)
	documento := strings.Repeat("x", maximoBytesCatalogo+1)
	c.DocumentoCanonico = &documento
	c.DocumentoHuellaSHA256 = nil
	c.PreimagenesHuellaSHA256 = ""
	if _, err := c.PrepararBorradorParaEditor("tecnico-configuracion-1"); !errors.Is(err, ErrGobiernoCategoriaRPTInvalido) ||
		c.DocumentoHuellaSHA256 != nil || *c.DocumentoCanonico != documento {
		t.Fatalf("documento excesivo alteró el borrador: %v", err)
	}
	if asignaciones := testing.AllocsPerRun(3, func() {
		_, _ = c.PrepararBorradorParaEditor("tecnico-configuracion-1")
	}); asignaciones != 0 {
		t.Fatalf("documento excesivo provocó asignaciones antes del rechazo: %v", asignaciones)
	}
}

func TestContenidoGobiernoCategoriaRPTDeshabilitarExigePreimagenExacta(t *testing.T) {
	c := contenidoDeshabilitarRPTPrueba(t)
	id, rev := *c.CategoriaID, *c.RevisionEsperada
	if err := c.ValidarParaEditor("tecnico-configuracion-1"); err != nil {
		t.Fatal(err)
	}
	c.PreimagenesControl[id] = PreimagenControlGobiernoCategoriaRPT{Version: c.Version - 1, Revision: rev - 1, HuellaSHA256: strings.Repeat("a", 64), Estado: "habilitada"}
	if err := c.ValidarParaEditor("tecnico-configuracion-1"); !errors.Is(err, ErrGobiernoCategoriaRPTInvalido) {
		t.Fatalf("CAS obsoleto: %v", err)
	}
}

func contenidoDeshabilitarRPTPrueba(t *testing.T) ContenidoGobiernoCategoriaRPT {
	t.Helper()
	c := contenidoPublicarRPTPrueba(t)
	var catalogo CatalogoConfigurable
	if err := json.Unmarshal([]byte(*c.DocumentoCanonico), &catalogo); err != nil {
		t.Fatal(err)
	}
	catalogo.Version = 2
	catalogo.VersionAnteriorRef = catalogo.ID + ":1"
	c.Version = 2
	for _, entrada := range catalogo.Entradas {
		c.PreimagenesControl[entrada.Clave] = PreimagenControlGobiernoCategoriaRPT{Version: 1, Revision: 7, HuellaSHA256: strings.Repeat("a", 64), Estado: "habilitada"}
	}
	catalogo.Entradas[0].Atributos["estado"] = "deshabilitada"
	c.CategoriaID = ptrGobiernoRPT(catalogo.Entradas[0].Clave)
	c.RevisionEsperada = ptrGobiernoRPT(int64(7))
	c.Accion = AccionGobiernoCategoriaRPTDeshabilitar
	doc, err := json.Marshal(catalogo)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(doc)
	c.DocumentoCanonico = ptrGobiernoRPT(string(doc))
	c.DocumentoHuellaSHA256 = ptrGobiernoRPT(hex.EncodeToString(h[:]))
	return c
}

func TestDeshabilitarRPTExigeVersionCompletaSinReactivarNiOmitir(t *testing.T) {
	for _, caso := range []string{"sin documento", "version antigua", "preimagen parcial", "categoria habilitada", "reactivacion"} {
		t.Run(caso, func(t *testing.T) {
			c := contenidoDeshabilitarRPTPrueba(t)
			var doc CatalogoConfigurable
			if err := json.Unmarshal([]byte(*c.DocumentoCanonico), &doc); err != nil {
				t.Fatal(err)
			}
			switch caso {
			case "sin documento":
				c.DocumentoCanonico = nil
			case "version antigua":
				p := c.PreimagenesControl[*c.CategoriaID]
				p.Version = c.Version
				c.PreimagenesControl[*c.CategoriaID] = p
			case "preimagen parcial":
				delete(c.PreimagenesControl, doc.Entradas[1].Clave)
			case "categoria habilitada":
				doc.Entradas[0].Atributos["estado"] = "habilitada"
			case "reactivacion":
				p := c.PreimagenesControl[doc.Entradas[1].Clave]
				p.Estado = "deshabilitada"
				c.PreimagenesControl[doc.Entradas[1].Clave] = p
			}
			if caso == "categoria habilitada" {
				b, _ := json.Marshal(doc)
				h := sha256.Sum256(b)
				c.DocumentoCanonico = ptrGobiernoRPT(string(b))
				c.DocumentoHuellaSHA256 = ptrGobiernoRPT(hex.EncodeToString(h[:]))
			}
			if !errors.Is(c.ValidarParaEditor("tecnico-configuracion-1"), ErrGobiernoCategoriaRPTInvalido) {
				t.Fatal("documento no conservador admitido")
			}
		})
	}
}
