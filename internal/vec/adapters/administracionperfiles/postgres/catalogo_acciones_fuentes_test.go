package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type filaCatalogoV2Prueba struct{ catalogo, paquete []byte }

func (f filaCatalogoV2Prueba) Scan(destinos ...any) error {
	if len(destinos) != 2 {
		return errors.New("la lectura no solicitó dos columnas")
	}
	*destinos[0].(*[]byte) = append([]byte(nil), f.catalogo...)
	*destinos[1].(*[]byte) = append([]byte(nil), f.paquete...)
	return nil
}

type poolCatalogoV2Prueba struct{ catalogo, paquete []byte }

func (p poolCatalogoV2Prueba) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return nil, errors.New("la fuente no escribe")
}

func (p poolCatalogoV2Prueba) QueryRow(_ context.Context, consulta string, _ ...any) pgx.Row {
	switch consulta {
	case acreditarFuenteCatalogoAccionesSQL:
		return filaFalsa{dato: true}
	case leerCatalogoAccionesSQL:
		return filaCatalogoV2Prueba{catalogo: p.catalogo, paquete: p.paquete}
	default:
		return filaFalsa{err: errors.New("fachada ajena")}
	}
}

func TestPreimagenFuenteV2VectorIndependiente(t *testing.T) {
	// Vector de bytes fijado independientemente para el contrato SQL/Go.
	const preimagen = `[{"referencia":"accion:sintetica","version":1,"fuente_ref":"fuente:modulo","fuente_version":2,"concesion":{"accion":"sintetico.consultar","modulo_id":"sintetico","tipo_recurso":"expediente","finalidades":["revision"],"garantia_minima":"alto"},"dimensiones_ambito":["unidad"],"clase_control":"consulta_auditada","vigente_desde":"2026-10-01T00:00:00Z","vigente_hasta":"0001-01-01T00:00:00Z"}]`
	const esperada = "5fdabf454a5f15834a80e70cef1aef929648805dc481119064206654bc4c94cb"
	entrada := domain.EntradaAccionAdministracionV1{
		Referencia: "accion:sintetica", Version: 1, FuenteRef: "fuente:modulo", FuenteVersion: 2,
		FuenteHuellaSHA256: strings.Repeat("a", 64),
		Concesion: domain.ConcesionRol{Accion: "sintetico.consultar", ModuloID: "sintetico",
			TipoRecurso: "expediente", Finalidades: []string{"revision"}, GarantiaMinima: domain.AuthAssuranceHigh},
		DimensionesAmbito: []string{"unidad"}, ClaseControl: "consulta_auditada",
		VigenteDesde: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	}
	canon, h, err := canonEntradasFuenteV2([]domain.EntradaAccionAdministracionV1{entrada})
	if err != nil || string(canon) != preimagen || len(canon) != 390 || h != esperada {
		t.Fatalf("preimagen Go difiere del vector SQL independiente: bytes=%d sha=%s err=%v", len(canon), h, err)
	}
}

func paqueteFuentePrueba(t *testing.T) ([]byte, domain.CatalogoAccionesAdministracionV1) {
	t.Helper()
	catalogoCanon, _ := catalogoAccionesPublicadoPrueba(t)
	var c domain.CatalogoAccionesAdministracionV1
	if err := json.Unmarshal(catalogoCanon, &c); err != nil {
		t.Fatal(err)
	}
	segunda := c.Entradas[0]
	segunda.Referencia = "accion:sintetica:segunda"
	segunda.Concesion.Accion = "sintetico.segunda"
	c.Entradas = append(c.Entradas, segunda)
	preimagen, h, err := canonEntradasFuenteV2(c.Entradas)
	if err != nil {
		t.Fatal(err)
	}
	fuente := fuentePaqueteCatalogoAccionesV2{ModuloID: "sintetico", Referencia: "fuente:modulo",
		Version: 2, HuellaSHA256: h, EntradasCanon: string(preimagen), Entradas: c.Entradas}
	for i := range fuente.Entradas {
		fuente.Entradas[i].FuenteHuellaSHA256 = h
	}
	c.Entradas = fuente.Entradas
	paquete := paqueteCatalogoAccionesV2{Esquema: esquemaPaqueteCatalogoAccionesV2,
		Referencia: c.FuenteRef, Version: c.FuenteVersion,
		Fuentes: []fuentePaqueteCatalogoAccionesV2{fuente}, Perfiles: c.Perfiles}
	b, err := json.Marshal(paquete)
	if err != nil {
		t.Fatal(err)
	}
	s := sha256.Sum256(b)
	c.FuenteHuellaSHA256 = hex.EncodeToString(s[:])
	if c.Validar() != nil {
		t.Fatal("fixture de catálogo inválida")
	}
	return b, c
}

func TestPaqueteFuenteV2RecalculaAutoriaYOrden(t *testing.T) {
	b, c := paqueteFuentePrueba(t)
	if err := ValidarPaqueteCatalogoAccionesV2(b, c, c.FuenteRef, c.FuenteVersion, c.FuenteHuellaSHA256); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`\"referencia\"`)) {
		t.Fatal("entradas_canon no conserva escapes en paquete")
	}
	var p paqueteCatalogoAccionesV2
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatal(err)
	}
	// Todas las alteraciones se rehuellan hasta el paquete exterior. El fallo
	// depende de la autoria de la fuente o del censo, no del SHA exterior.
	casos := map[string]func(*paqueteCatalogoAccionesV2){
		"huella fuente inventada":    func(p *paqueteCatalogoAccionesV2) { p.Fuentes[0].HuellaSHA256 = strings.Repeat("f", 64) },
		"preimagen truncada":         func(p *paqueteCatalogoAccionesV2) { p.Fuentes[0].EntradasCanon = `[]` },
		"referencia fuente distinta": func(p *paqueteCatalogoAccionesV2) { p.Fuentes[0].Referencia = "fuente:ajena" },
		"módulo distinto":            func(p *paqueteCatalogoAccionesV2) { p.Fuentes[0].ModuloID = "ajeno" },
		"entrada omitida":            func(p *paqueteCatalogoAccionesV2) { p.Fuentes[0].Entradas = nil },
		"perfiles omitidos":          func(p *paqueteCatalogoAccionesV2) { p.Perfiles = nil },
		"unicode descriptor":         func(p *paqueteCatalogoAccionesV2) { p.Fuentes[0].Referencia = "fuente:ñ" },
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			otro := p
			otro.Fuentes = append([]fuentePaqueteCatalogoAccionesV2(nil), p.Fuentes...)
			cambiar(&otro)
			contenido, err := json.Marshal(otro)
			if err != nil {
				t.Fatal(err)
			}
			h := sha256.Sum256(contenido)
			alterado := c
			alterado.FuenteHuellaSHA256 = hex.EncodeToString(h[:])
			if err := ValidarPaqueteCatalogoAccionesV2(contenido, alterado,
				alterado.FuenteRef, alterado.FuenteVersion, alterado.FuenteHuellaSHA256); !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) {
				t.Fatalf("material alterado aceptado: %v", err)
			}
		})
	}
	for nombre, contenido := range map[string][]byte{
		"raíz extra":         append(append([]byte{}, b[:len(b)-1]...), []byte(`,"extra":"ajeno"}`)...),
		"raíz duplicada":     append([]byte(`{"esquema":"ajeno",`), b[1:]...),
		"fuente extra":       bytes.Replace(b, []byte(`"modulo_id":"sintetico"`), []byte(`"modulo_id":"sintetico","extra":true`), 1),
		"fuente duplicada":   bytes.Replace(b, []byte(`"modulo_id":"sintetico"`), []byte(`"modulo_id":"ajeno","modulo_id":"sintetico"`), 1),
		"entrada duplicada":  bytes.Replace(b, []byte(`"referencia":"accion:sintetica"`), []byte(`"referencia":"ajena","referencia":"accion:sintetica"`), 1),
		"escape equivalente": bytes.Replace(b, []byte(`"fuente:modulo"`), []byte(`"fuente:\u006dodulo"`), 1),
		"HTML escapado":      bytes.Replace(b, []byte(`"fuente:modulo"`), []byte(`"fuente:\u003cmodulo"`), 1),
	} {
		t.Run(nombre, func(t *testing.T) {
			if bytes.Equal(contenido, b) {
				t.Fatal("mutación de prueba no aplicada")
			}
			h := sha256.Sum256(contenido)
			alterado := c
			alterado.FuenteHuellaSHA256 = hex.EncodeToString(h[:])
			if err := ValidarPaqueteCatalogoAccionesV2(contenido, alterado,
				alterado.FuenteRef, alterado.FuenteVersion, alterado.FuenteHuellaSHA256); !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) {
				t.Fatalf("JSON no canónico aceptado: %v", err)
			}
		})
	}
	t.Run("orden alterado con huellas recalculadas", func(t *testing.T) {
		var otro paqueteCatalogoAccionesV2
		if err := json.Unmarshal(b, &otro); err != nil {
			t.Fatal(err)
		}
		fuente := &otro.Fuentes[0]
		fuente.Entradas[0], fuente.Entradas[1] = fuente.Entradas[1], fuente.Entradas[0]
		preimagen, h, err := canonEntradasFuenteV2(fuente.Entradas)
		if err != nil {
			t.Fatal(err)
		}
		fuente.EntradasCanon, fuente.HuellaSHA256 = string(preimagen), h
		for i := range fuente.Entradas {
			fuente.Entradas[i].FuenteHuellaSHA256 = h
		}
		contenido, err := json.Marshal(otro)
		if err != nil {
			t.Fatal(err)
		}
		s := sha256.Sum256(contenido)
		alterado := c
		alterado.Entradas = append([]domain.EntradaAccionAdministracionV1(nil), c.Entradas...)
		for i := range alterado.Entradas {
			alterado.Entradas[i].FuenteHuellaSHA256 = h
		}
		alterado.FuenteHuellaSHA256 = hex.EncodeToString(s[:])
		if err := ValidarPaqueteCatalogoAccionesV2(contenido, alterado,
			alterado.FuenteRef, alterado.FuenteVersion, alterado.FuenteHuellaSHA256); !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) {
			t.Fatalf("orden alterado aceptado: %v", err)
		}
	})
	t.Run("identidad fuente duplicada", func(t *testing.T) {
		var otro paqueteCatalogoAccionesV2
		if err := json.Unmarshal(b, &otro); err != nil {
			t.Fatal(err)
		}
		primera, segunda := otro.Fuentes[0], otro.Fuentes[0]
		primera.Entradas = primera.Entradas[:1]
		segunda.Entradas = segunda.Entradas[1:]
		alterado := c
		alterado.Entradas = append([]domain.EntradaAccionAdministracionV1(nil), c.Entradas...)
		for i, fuente := range []*fuentePaqueteCatalogoAccionesV2{&primera, &segunda} {
			preimagen, h, err := canonEntradasFuenteV2(fuente.Entradas)
			if err != nil {
				t.Fatal(err)
			}
			fuente.EntradasCanon, fuente.HuellaSHA256 = string(preimagen), h
			fuente.Entradas[0].FuenteHuellaSHA256 = h
			alterado.Entradas[i].FuenteHuellaSHA256 = h
		}
		otro.Fuentes = []fuentePaqueteCatalogoAccionesV2{primera, segunda}
		contenido, err := json.Marshal(otro)
		if err != nil {
			t.Fatal(err)
		}
		s := sha256.Sum256(contenido)
		alterado.FuenteHuellaSHA256 = hex.EncodeToString(s[:])
		if err := ValidarPaqueteCatalogoAccionesV2(contenido, alterado,
			alterado.FuenteRef, alterado.FuenteVersion, alterado.FuenteHuellaSHA256); !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) {
			t.Fatalf("descriptor duplicado aceptado: %v", err)
		}
	})
}

func TestFuenteCatalogoV2RequierePaqueteDeLaMismaLectura(t *testing.T) {
	paquete, c := paqueteFuentePrueba(t)
	catalogo, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	h, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre  string
		paquete []byte
		valido  bool
	}{
		{"v2 completa", paquete, true},
		{"paquete ausente", nil, false},
		{"paquete ajeno", []byte(`{}`), false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			fuente, err := nuevaFuenteCatalogoAcciones(context.Background(), poolCatalogoV2Prueba{catalogo: catalogo, paquete: caso.paquete})
			if err != nil {
				t.Fatal(err)
			}
			_, err = fuente.ObtenerCatalogoAccionesAdministracionV1(context.Background(), c.Referencia, c.Version, h)
			if caso.valido && err != nil || !caso.valido && !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) {
				t.Fatalf("fuente v2 no cerró lectura: %v", err)
			}
		})
	}
}
