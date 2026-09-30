package postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func publicacionRPTPrueba(t *testing.T, version int, clave string) (publicacionRPTWire, domain.EntradaCatalogoConfigurable) {
	t.Helper()
	fecha := time.Date(2026, 9, 30, 1, 0, 0, 0, time.UTC)
	entrada := domain.EntradaCatalogoConfigurable{
		Clave: clave, Etiqueta: "Categoría sintética " + clave, Orden: version,
		VigenteDesde: fecha, Atributos: map[string]string{"origen": "prueba"},
	}
	catalogo := domain.CatalogoConfigurable{
		ID: "rpt.categorias", Version: version, Revision: 1, ModuloID: "organizacion",
		Nombre: "Categorías sintéticas", FuenteRef: "fuente:prueba:rpt",
		MotivoCreacion: "Prueba del lector de publicaciones", Entradas: []domain.EntradaCatalogoConfigurable{entrada},
		Estado: domain.EstadoCatalogoPublicado, CreadoPor: "actor:prueba:uno", CreadoEn: fecha,
		PublicadoPor: "actor:prueba:dos", PublicadoEn: fecha.Add(time.Minute),
		AprobacionRef: "aprobacion:prueba", MotivoPublicacion: "Prueba sintética",
	}
	if version > 1 {
		catalogo.VersionAnteriorRef = "rpt.categorias:1"
	}
	canonico, err := catalogo.ClonarCanonico()
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := json.Marshal(canonico)
	if err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(bytes)
	return publicacionRPTWire{
		CatalogoID: catalogo.ID, Version: version, HuellaSHA256: hex.EncodeToString(suma[:]),
		DocumentoCanonico: string(bytes), PublicadaEn: fecha.Add(time.Minute),
	}, entrada
}

func jsonRPTPrueba(t *testing.T, valor any) []byte {
	t.Helper()
	bytes, err := json.Marshal(valor)
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}

func TestListaRPTConservaVersionPorCategoriaYRechazaDocumentoOEntradaAlterados(t *testing.T) {
	p1, e1 := publicacionRPTPrueba(t, 1, "categoria.uno")
	p2, e2 := publicacionRPTPrueba(t, 2, "categoria.dos")
	item := func(p publicacionRPTWire, e domain.EntradaCatalogoConfigurable) categoriaRPTWire {
		return categoriaRPTWire{CategoriaID: e.Clave, CatalogoID: p.CatalogoID, Version: p.Version,
			HuellaSHA256: p.HuellaSHA256, Revision: int64(p.Version), Estado: "habilitada",
			Etiqueta: e.Etiqueta, Definicion: jsonRPTPrueba(t, e)}
	}
	listaPublicacion := func(p publicacionRPTWire) map[string]any {
		return map[string]any{"catalogo_id": p.CatalogoID, "version": p.Version,
			"huella_sha256": p.HuellaSHA256, "documento_canonico": p.DocumentoCanonico}
	}
	lista := listaRPTWire{Items: []json.RawMessage{jsonRPTPrueba(t, item(p2, e2)), jsonRPTPrueba(t, item(p1, e1))},
		Publicaciones: []json.RawMessage{jsonRPTPrueba(t, listaPublicacion(p1)), jsonRPTPrueba(t, listaPublicacion(p2))},
		HayMas:        true, SiguienteCursor: &e1.Clave}
	consulta := ports.ConsultaCategoriasHabilitadasRPT{CatalogoID: "rpt.categorias", Limite: 100}
	r, err := decodificarListaRPT(jsonRPTPrueba(t, lista), consulta)
	if err != nil || !r.Encontrado || len(r.Categorias) != 2 || len(r.Publicaciones) != 2 ||
		r.Categorias[0].Publicacion.Version != 2 || r.Categorias[1].Publicacion.Version != 1 ||
		!r.HayMas || r.SiguienteCursor == nil || *r.SiguienteCursor != e1.Clave {
		t.Fatalf("lista mixta perdida: %+v %v", r, err)
	}
	alterada := p1
	alterada.DocumentoCanonico += " "
	lista.Publicaciones[0] = jsonRPTPrueba(t, listaPublicacion(alterada))
	if _, err := decodificarListaRPT(jsonRPTPrueba(t, lista), consulta); !errors.Is(err, ports.ErrLecturaRPTNoConfiable) {
		t.Fatalf("documento con huella ajena aceptado: %v", err)
	}
	lista.Publicaciones[0] = jsonRPTPrueba(t, listaPublicacion(p1))
	falsa := item(p1, e1)
	falsa.Etiqueta = "Otra categoría"
	lista.Items[1] = jsonRPTPrueba(t, falsa)
	if _, err := decodificarListaRPT(jsonRPTPrueba(t, lista), consulta); !errors.Is(err, ports.ErrLecturaRPTNoConfiable) {
		t.Fatalf("entrada divergente aceptada: %v", err)
	}
}

func TestHistoricaRPTNoSustituyePublicacionPorControlActual(t *testing.T) {
	p, entrada := publicacionRPTPrueba(t, 1, "categoria.uno")
	actual := controlActualRPTWire{CatalogoID: p.CatalogoID, Version: 2,
		HuellaSHA256: p.HuellaSHA256, Revision: 3, Estado: "tombstone"}
	datos := publicacionHistoricaRPTWire{Publicacion: jsonRPTPrueba(t, p), Entrada: jsonRPTPrueba(t, entrada),
		ControlActual: jsonRPTPrueba(t, actual)}
	r, err := decodificarPublicacionHistoricaRPT(jsonRPTPrueba(t, datos), ports.ConsultaPublicacionCategoriaRPT{
		Referencia: p.referencia(), CategoriaID: entrada.Clave})
	if err != nil || !r.Encontrado || r.Publicacion == nil || r.ControlActual == nil ||
		r.Publicacion.Referencia.Version != 1 || r.ControlActual.Publicacion.Version != 2 ||
		r.ControlActual.Estado != "tombstone" || r.Entrada.Clave != entrada.Clave {
		t.Fatalf("historia/control confundidos: %+v %v", r, err)
	}
}

func TestUsoRPTExigeIdentidadYReciboDeReservaExactos(t *testing.T) {
	p, entrada := publicacionRPTPrueba(t, 1, "categoria.uno")
	ahora := time.Date(2026, 9, 30, 1, 0, 0, 0, time.UTC)
	uso := usoCategoriaRPTWire{Consumidor: "contratacion_temporal", UsoRef: "uso:prueba:uno",
		CategoriaID: entrada.Clave, CatalogoID: p.CatalogoID, Version: p.Version, HuellaSHA256: p.HuellaSHA256,
		Estado: "reservado", Revision: 1, ReservaReciboRef: "recibo:reserva:uno", ReservadoEn: ahora}
	consulta := ports.ConsultaUsoCategoriaRPT{Consumidor: uso.Consumidor, UsoRef: uso.UsoRef,
		ReservaReciboRef: uso.ReservaReciboRef}
	r, err := decodificarUsoCategoriaRPT(jsonRPTPrueba(t, uso), consulta)
	if err != nil || !r.Encontrado || r.Uso == nil || r.Uso.Estado != "reservado" {
		t.Fatalf("reserva previa perdida: %+v %v", r, err)
	}
	if _, err := decodificarUsoCategoriaRPT(jsonRPTPrueba(t, uso), ports.ConsultaUsoCategoriaRPT{
		Consumidor: uso.Consumidor, UsoRef: uso.UsoRef, ReservaReciboRef: "recibo:otro"}); !errors.Is(err, ports.ErrLecturaRPTNoConfiable) {
		t.Fatalf("recibo ajeno aceptado: %v", err)
	}
	uso.Estado, uso.Revision = "confirmado", 2
	uso.TerminalReciboRef = new(string)
	*uso.TerminalReciboRef = "recibo:terminal:uno"
	uso.TerminalEn = new(time.Time)
	*uso.TerminalEn = ahora.Add(time.Minute)
	if r, err := decodificarUsoCategoriaRPT(jsonRPTPrueba(t, uso), consulta); err != nil || r.Uso.TerminalReciboRef == nil {
		t.Fatalf("uso terminal perdido: %+v %v", r, err)
	}
}
