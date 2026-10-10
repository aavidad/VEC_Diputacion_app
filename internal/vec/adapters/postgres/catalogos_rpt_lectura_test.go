package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var descriptorRPTPrueba = ports.DescriptorCatalogoRPT{CatalogoID: "rpt.categorias", ModuloID: "organizacion"}

func autorizacionRPTPrueba(t *testing.T) ports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	ahora := time.Date(2026, 9, 30, 1, 0, 0, 0, time.UTC)
	h := strings.Repeat("a", 64)
	resumen, err := ports.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:rpt:prueba", h, h, "contexto:rpt:prueba", h,
		"vec.catalogos.categorias.listar_habilitadas", "rpt.categorias", h,
		"vec_catalogos_configurables.lectura_categorias.v1", ahora, ahora.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	if err != nil {
		t.Fatal(err)
	}
	material, err := ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		bytes.Repeat([]byte("x"), 512), resumen, []byte("d"), []byte("m"), []byte("c"), 1, 1,
		[]byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return material
}

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
	lista := listaRPTWire{AnclajePublicacion: jsonRPTPrueba(t, listaPublicacion(p1)),
		Items:         []json.RawMessage{jsonRPTPrueba(t, item(p2, e2)), jsonRPTPrueba(t, item(p1, e1))},
		Publicaciones: []json.RawMessage{jsonRPTPrueba(t, listaPublicacion(p2))},
		HayMas:        true, SiguienteCursor: &e1.Clave}
	consulta := ports.ConsultaCategoriasHabilitadasRPT{CatalogoID: "rpt.categorias", Limite: 100}
	r, err := decodificarListaRPT(jsonRPTPrueba(t, lista), consulta, descriptorRPTPrueba)
	if err != nil || !r.Encontrado || len(r.Categorias) != 2 || len(r.Publicaciones) != 2 ||
		r.Categorias[0].Publicacion.Version != 2 || r.Categorias[1].Publicacion.Version != 1 ||
		!r.HayMas || r.SiguienteCursor == nil || *r.SiguienteCursor != e1.Clave {
		t.Fatalf("lista mixta perdida: %+v %v", r, err)
	}
	ajeno := descriptorRPTPrueba
	ajeno.ModuloID = "otro_modulo"
	if _, err := decodificarListaRPT(jsonRPTPrueba(t, lista), consulta, ajeno); !errors.Is(err, ports.ErrLecturaRPTNoConfiable) {
		t.Fatalf("publicación de otro módulo aceptada: %v", err)
	}
	alterada := p1
	alterada.DocumentoCanonico += " "
	lista.AnclajePublicacion = jsonRPTPrueba(t, listaPublicacion(alterada))
	if _, err := decodificarListaRPT(jsonRPTPrueba(t, lista), consulta, descriptorRPTPrueba); !errors.Is(err, ports.ErrLecturaRPTNoConfiable) {
		t.Fatalf("documento con huella ajena aceptado: %v", err)
	}
	lista.AnclajePublicacion = jsonRPTPrueba(t, listaPublicacion(p1))
	falsa := item(p1, e1)
	falsa.Etiqueta = "Otra categoría"
	lista.Items[1] = jsonRPTPrueba(t, falsa)
	if _, err := decodificarListaRPT(jsonRPTPrueba(t, lista), consulta, descriptorRPTPrueba); !errors.Is(err, ports.ErrLecturaRPTNoConfiable) {
		t.Fatalf("entrada divergente aceptada: %v", err)
	}
}

func TestListaRPTVaciaConservaModuloGobernado(t *testing.T) {
	p, _ := publicacionRPTPrueba(t, 1, "categoria.uno")
	consulta := ports.ConsultaCategoriasHabilitadasRPT{CatalogoID: descriptorRPTPrueba.CatalogoID, Limite: 100}
	vacia := listaRPTWire{AnclajePublicacion: jsonRPTPrueba(t, map[string]any{
		"catalogo_id": p.CatalogoID, "version": p.Version, "huella_sha256": p.HuellaSHA256,
		"documento_canonico": p.DocumentoCanonico}),
		Items: []json.RawMessage{}, Publicaciones: []json.RawMessage{}}
	r, err := decodificarListaRPT(jsonRPTPrueba(t, vacia), consulta, descriptorRPTPrueba)
	if err != nil || !r.Encontrado || len(r.Categorias) != 0 {
		t.Fatalf("página vacía publicada rechazada: %+v %v", r, err)
	}
	ajeno := descriptorRPTPrueba
	ajeno.ModuloID = "otro_modulo"
	if _, err := decodificarListaRPT(jsonRPTPrueba(t, vacia), consulta, ajeno); !errors.Is(err, ports.ErrLecturaRPTNoConfiable) {
		t.Fatalf("módulo ajeno en página vacía aceptado: %v", err)
	}
}

func TestHistoricaRPTNoSustituyePublicacionPorControlActual(t *testing.T) {
	p, entrada := publicacionRPTPrueba(t, 1, "categoria.uno")
	actual := controlActualRPTWire{CatalogoID: p.CatalogoID, Version: 2,
		HuellaSHA256: p.HuellaSHA256, Revision: 3, Estado: "tombstone"}
	datos := publicacionHistoricaRPTWire{Publicacion: jsonRPTPrueba(t, p), Entrada: jsonRPTPrueba(t, entrada),
		ControlActual: jsonRPTPrueba(t, actual)}
	r, err := decodificarPublicacionHistoricaRPT(jsonRPTPrueba(t, datos), ports.ConsultaPublicacionCategoriaRPT{
		Referencia: p.referencia(), CategoriaID: entrada.Clave}, descriptorRPTPrueba)
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
	r, err := decodificarUsoCategoriaRPT(jsonRPTPrueba(t, uso), consulta, descriptorRPTPrueba)
	if err != nil || !r.Encontrado || r.Uso == nil || r.Uso.Estado != "reservado" {
		t.Fatalf("reserva previa perdida: %+v %v", r, err)
	}
	if _, err := decodificarUsoCategoriaRPT(jsonRPTPrueba(t, uso), ports.ConsultaUsoCategoriaRPT{
		Consumidor: uso.Consumidor, UsoRef: uso.UsoRef, ReservaReciboRef: "recibo:otro"}, descriptorRPTPrueba); !errors.Is(err, ports.ErrLecturaRPTNoConfiable) {
		t.Fatalf("recibo ajeno aceptado: %v", err)
	}
	uso.Estado, uso.Revision = "confirmado", 2
	uso.TerminalReciboRef = new(string)
	*uso.TerminalReciboRef = "recibo:terminal:uno"
	uso.TerminalEn = new(time.Time)
	*uso.TerminalEn = ahora.Add(time.Minute)
	if r, err := decodificarUsoCategoriaRPT(jsonRPTPrueba(t, uso), consulta, descriptorRPTPrueba); err != nil || r.Uso.TerminalReciboRef == nil {
		t.Fatalf("uso terminal perdido: %+v %v", r, err)
	}
}

func TestReciboLecturaRPTExigeConsumoNuevoInclusoSinResultado(t *testing.T) {
	autorizacion := autorizacionRPTPrueba(t)
	resumen := autorizacion.ResumenCapacidad()
	base := reciboLecturaRPTWire{
		DecisionRef: resumen.DecisionRef(), EfectoRef: resumen.EfectoRef(),
		HuellaEfectoSHA256: resumen.EfectoHuellaSHA256(), ConsumoHuellaSHA256: strings.Repeat("b", 64),
		AuditoriaRef: "aud_v3_sintetica", ConsumidaEn: resumen.EmitidaEn().Add(time.Second),
		ConsumoNuevo: true, Encontrado: false, Datos: json.RawMessage("null"),
	}
	evidencia, encontrado, datos, err := decodificarReciboLecturaRPT(jsonRPTPrueba(t, base), autorizacion)
	if err != nil || encontrado || string(datos) != "null" || !evidencia.ConsumoNuevo || evidencia.AuditoriaRef == "" {
		t.Fatalf("lectura ausente perdió auditoría: %+v %t %q %v", evidencia, encontrado, datos, err)
	}
	base.ConsumoNuevo = false
	if _, _, _, err := decodificarReciboLecturaRPT(jsonRPTPrueba(t, base), autorizacion); !errors.Is(err, ports.ErrLecturaRPTNoConfiable) {
		t.Fatalf("replay sin consumo nuevo aceptado: %v", err)
	}
	base.ConsumoNuevo = true
	base.DecisionRef = "decision:ajena"
	if _, _, _, err := decodificarReciboLecturaRPT(jsonRPTPrueba(t, base), autorizacion); !errors.Is(err, ports.ErrLecturaRPTNoConfiable) {
		t.Fatalf("decisión ajena aceptada: %v", err)
	}
}

type filaLecturaRPTPrueba struct {
	respuesta []byte
	err       error
}

type filaHuellaRPTPrueba struct {
	huella string
	err    error
}

func (f filaHuellaRPTPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != 1 {
		return errors.New("huella con columnas inesperadas")
	}
	destino, ok := destinos[0].(*string)
	if !ok {
		return errors.New("huella sin destino textual")
	}
	*destino = f.huella
	return nil
}

func (f filaLecturaRPTPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != 1 {
		return errors.New("cantidad de columnas inesperada")
	}
	destino, ok := destinos[0].(*[]byte)
	if !ok {
		return errors.New("columna no binaria")
	}
	*destino = append([]byte(nil), f.respuesta...)
	return nil
}

type transaccionLecturaRPTPrueba struct {
	pgx.Tx
	respuesta        []byte
	errConsulta      error
	huellaMaterial   string
	materialCanonico string
	consultasCanon   int
	fachadas         int
	consulta         string
	argumentos       []any
	configurada      bool
	confirmada       bool
	revertida        bool
}

func (t *transaccionLecturaRPTPrueba) Exec(_ context.Context, consulta string, _ ...any) (pgconn.CommandTag, error) {
	t.configurada = consulta == configurarLecturaRPT &&
		strings.Contains(consulta, "set_config('timezone','UTC',true)") &&
		strings.Contains(consulta, "set_config('statement_timeout','15s',true)") &&
		strings.Contains(consulta, "set_config('idle_in_transaction_session_timeout','20s',true)")
	return pgconn.CommandTag{}, nil
}

func (t *transaccionLecturaRPTPrueba) QueryRow(_ context.Context, consulta string, argumentos ...any) pgx.Row {
	if consulta == consultaHuellaMaterialRPT {
		t.consultasCanon++
		if len(argumentos) == 1 {
			t.materialCanonico, _ = argumentos[0].(string)
		}
		huella := t.huellaMaterial
		if huella == "" {
			huella = strings.Repeat("a", 64)
		}
		return filaHuellaRPTPrueba{huella: huella}
	}
	t.fachadas++
	t.consulta = consulta
	t.argumentos = make([]any, len(argumentos))
	for i, argumento := range argumentos {
		if b, ok := argumento.([]byte); ok {
			t.argumentos[i] = append([]byte(nil), b...)
		} else {
			t.argumentos[i] = argumento
		}
	}
	return filaLecturaRPTPrueba{respuesta: t.respuesta, err: t.errConsulta}
}

func (t *transaccionLecturaRPTPrueba) Commit(context.Context) error {
	t.confirmada = true
	return nil
}

func (t *transaccionLecturaRPTPrueba) Rollback(context.Context) error {
	t.revertida = true
	return nil
}

type iniciadorLecturaRPTPrueba struct {
	tx       *transaccionLecturaRPTPrueba
	opciones pgx.TxOptions
	llamadas int
}

func (i *iniciadorLecturaRPTPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	i.llamadas++
	i.opciones = opciones
	return i.tx, nil
}

func autorizacionYSolicitudRPTPrueba(t *testing.T, accion, tipo, referencia, consumidor string) (domain.SolicitudAutorizacionLigadaV3, ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) {
	t.Helper()
	escenario := nuevoEscenarioRegistroContextoActorV3PostgreSQLPrueba(t, true)
	d, err := escenario.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	d.Accion, d.Finalidad = accion, finalidadLecturaRPT
	ambitos := map[string]string{"catalogo_id": descriptorRPTPrueba.CatalogoID, "modulo_id": descriptorRPTPrueba.ModuloID}
	if consumidor != "" {
		ambitos["consumidor"] = consumidor
	}
	d.Recurso = domain.RecursoAutorizable{
		Referencia: referencia, ModuloID: descriptorRPTPrueba.ModuloID,
		Tipo: tipo, Ambitos: ambitos,
		Atributos: map[string]string{"material_sha256": strings.Repeat("a", 64)},
	}
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(d)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := d.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	resumen, err := ports.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:rpt:prueba", strings.Repeat("a", 64), strings.Repeat("b", 64),
		"contexto:rpt:prueba", strings.Repeat("c", 64), accion,
		d.Recurso.Referencia, huella, audienciaLecturaRPT, escenario.ahora, escenario.ahora.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	autorizacion, err := ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		bytes.Repeat([]byte("x"), 512), resumen, []byte("d"), []byte("m"), []byte("c"), 1, 1,
		[]byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return solicitud, autorizacion
}

func ordenListaRPTPrueba(t *testing.T) ports.OrdenCategoriasHabilitadasRPT {
	t.Helper()
	solicitud, autorizacion := autorizacionYSolicitudRPTPrueba(t, accionListarCategoriasRPT,
		tipoCatalogoRPT, descriptorRPTPrueba.CatalogoID, "")
	return ports.OrdenCategoriasHabilitadasRPT{Consulta: ports.ConsultaCategoriasHabilitadasRPT{
		CatalogoID: descriptorRPTPrueba.CatalogoID, Limite: 100}, Solicitud: solicitud, Autorizacion: autorizacion}
}

func TestLecturaRPTConsumeAD3EnTransaccionYConservaAuditoriaSiNoHayFilas(t *testing.T) {
	orden := ordenListaRPTPrueba(t)
	resumen := orden.Autorizacion.ResumenCapacidad()
	respuesta := reciboLecturaRPTWire{
		DecisionRef: resumen.DecisionRef(), EfectoRef: resumen.EfectoRef(),
		HuellaEfectoSHA256: resumen.EfectoHuellaSHA256(), ConsumoHuellaSHA256: strings.Repeat("d", 64),
		AuditoriaRef: "aud_v3_sintetica", ConsumidaEn: resumen.EmitidaEn().Add(time.Second),
		ConsumoNuevo: true, Encontrado: false, Datos: json.RawMessage("null"),
	}
	tx := &transaccionLecturaRPTPrueba{respuesta: jsonRPTPrueba(t, respuesta)}
	iniciador := &iniciadorLecturaRPTPrueba{tx: tx}
	lector, err := nuevoLectorCategoriasRPTPostgreSQL(iniciador, descriptorRPTPrueba)
	if err != nil {
		t.Fatal(err)
	}
	r, err := lector.ListarCategoriasHabilitadasRPT(context.Background(), orden)
	if err != nil || r.Encontrado || !r.Evidencia.ConsumoNuevo || !tx.confirmada || !tx.configurada ||
		iniciador.opciones.IsoLevel != pgx.Serializable || iniciador.opciones.AccessMode != pgx.ReadWrite ||
		tx.consulta != consultaListaRPT || len(tx.argumentos) != 11 || tx.consultasCanon != 1 || tx.fachadas != 1 {
		t.Fatalf("lectura sin efecto no conservó recibo: %+v %v tx=%+v", r, err, tx)
	}
	var material map[string]any
	if err := json.Unmarshal([]byte(tx.argumentos[0].(string)), &material); err != nil ||
		len(material) != 4 || material["catalogo_id"] != descriptorRPTPrueba.CatalogoID ||
		material["modulo_id"] != descriptorRPTPrueba.ModuloID || material["cursor_categoria_id"] != nil ||
		material["limite"] != float64(100) {
		t.Fatalf("material no exacto: %+v %v", material, err)
	}
	if tx.argumentos[5] != "1" || tx.argumentos[6] != "1" {
		t.Fatalf("versiones V3 no preservadas: %+v", tx.argumentos)
	}
	// Un recibo sin consumo nuevo revierte la transacción: no se presenta como lectura.
	respuesta.ConsumoNuevo = false
	tx2 := &transaccionLecturaRPTPrueba{respuesta: jsonRPTPrueba(t, respuesta)}
	lector2, _ := nuevoLectorCategoriasRPTPostgreSQL(&iniciadorLecturaRPTPrueba{tx: tx2}, descriptorRPTPrueba)
	if _, err := lector2.ListarCategoriasHabilitadasRPT(context.Background(), orden); !errors.Is(err, ports.ErrLecturaRPTNoConfiable) || tx2.confirmada || !tx2.revertida {
		t.Fatalf("recibo sin consumo nuevo aceptado: %v tx=%+v", err, tx2)
	}
	for _, caso := range []struct {
		codigo   string
		esperado error
	}{
		{"42501", ports.ErrLecturaRPTDenegada},
		{"54000", ports.ErrLecturaRPTPresupuesto},
	} {
		tx := &transaccionLecturaRPTPrueba{errConsulta: &pgconn.PgError{Code: caso.codigo}}
		lector, _ := nuevoLectorCategoriasRPTPostgreSQL(&iniciadorLecturaRPTPrueba{tx: tx}, descriptorRPTPrueba)
		if _, err := lector.ListarCategoriasHabilitadasRPT(context.Background(), orden); !errors.Is(err, caso.esperado) || tx.confirmada || !tx.revertida {
			t.Fatalf("SQLSTATE %s no cerró el efecto: %v tx=%+v", caso.codigo, err, tx)
		}
	}
}

func TestLectorRPTSinDescriptorConfiableDeniegaAntesDeConsultar(t *testing.T) {
	iniciador := &iniciadorLecturaRPTPrueba{tx: &transaccionLecturaRPTPrueba{}}
	if _, err := nuevoLectorCategoriasRPTPostgreSQL(iniciador, ports.DescriptorCatalogoRPT{}); !errors.Is(err, ports.ErrLecturaRPTDenegada) || iniciador.llamadas != 0 {
		t.Fatalf("descriptor ausente creó lector: %v", err)
	}
	if _, err := nuevoLectorCategoriasRPTPostgreSQL(nil, descriptorRPTPrueba); !errors.Is(err, ports.ErrLecturaRPTNoDisponible) {
		t.Fatalf("pool ausente creó lector: %v", err)
	}
}

func reciboAusenciaRPTPrueba(t *testing.T, autorizacion ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) []byte {
	t.Helper()
	z := autorizacion.ResumenCapacidad()
	return jsonRPTPrueba(t, reciboLecturaRPTWire{
		DecisionRef: z.DecisionRef(), EfectoRef: z.EfectoRef(),
		HuellaEfectoSHA256: z.EfectoHuellaSHA256(), ConsumoHuellaSHA256: strings.Repeat("d", 64),
		AuditoriaRef: "aud_v3_sintetica", ConsumidaEn: z.EmitidaEn().Add(time.Second),
		ConsumoNuevo: true, Encontrado: false, Datos: json.RawMessage("null"),
	})
}

func TestLecturasRPTHistoricaYUsoEnlazanMaterialYFuncionNominal(t *testing.T) {
	p, entrada := publicacionRPTPrueba(t, 1, "categoria.uno")
	historica := ports.ConsultaPublicacionCategoriaRPT{Referencia: p.referencia(), CategoriaID: entrada.Clave}
	s, a := autorizacionYSolicitudRPTPrueba(t, accionLeerPublicacionRPT,
		tipoCatalogoRPT, descriptorRPTPrueba.CatalogoID, "")
	tx := &transaccionLecturaRPTPrueba{respuesta: reciboAusenciaRPTPrueba(t, a)}
	lector, _ := nuevoLectorCategoriasRPTPostgreSQL(&iniciadorLecturaRPTPrueba{tx: tx}, descriptorRPTPrueba)
	r, err := lector.LeerPublicacionCategoriaRPT(context.Background(), ports.OrdenPublicacionCategoriaRPT{
		Consulta: historica, Solicitud: s, Autorizacion: a})
	if err != nil || r.Encontrado || !r.Evidencia.ConsumoNuevo || !tx.confirmada ||
		tx.consulta != consultaPublicacionRPT || len(tx.argumentos) != 11 || tx.consultasCanon != 1 || tx.fachadas != 1 {
		t.Fatalf("consulta histórica ajena: %+v %v tx=%+v", r, err, tx)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(tx.argumentos[0].(string)), &m); err != nil || len(m) != 5 ||
		m["catalogo_id"] != p.CatalogoID || m["modulo_id"] != descriptorRPTPrueba.ModuloID ||
		m["version"] != float64(p.Version) || m["huella_sha256"] != p.HuellaSHA256 ||
		m["categoria_id"] != entrada.Clave {
		t.Fatalf("material histórico no exacto: %+v %v", m, err)
	}
	uso := ports.ConsultaUsoCategoriaRPT{Consumidor: "contratacion_temporal",
		UsoRef: "uso:prueba:uno", ReservaReciboRef: "recibo:reserva:uno"}
	s, a = autorizacionYSolicitudRPTPrueba(t, accionConsultarUsoRPT, tipoUsoRPT, uso.UsoRef, uso.Consumidor)
	tx = &transaccionLecturaRPTPrueba{respuesta: reciboAusenciaRPTPrueba(t, a)}
	iniciador := &iniciadorLecturaRPTPrueba{tx: tx}
	lector, _ = nuevoLectorCategoriasRPTPostgreSQL(iniciador, descriptorRPTPrueba)
	ru, err := lector.ConsultarUsoCategoriaRPT(context.Background(), ports.OrdenUsoCategoriaRPT{
		Consulta: uso, Solicitud: s, Autorizacion: a})
	if err != nil || ru.Encontrado || !ru.Evidencia.ConsumoNuevo || !tx.confirmada ||
		tx.consulta != consultaUsoRPT || len(tx.argumentos) != 11 || tx.consultasCanon != 1 || tx.fachadas != 1 {
		t.Fatalf("consulta de uso ajena: %+v %v tx=%+v", ru, err, tx)
	}
	m = nil
	if err := json.Unmarshal([]byte(tx.argumentos[0].(string)), &m); err != nil || len(m) != 5 ||
		m["catalogo_id"] != descriptorRPTPrueba.CatalogoID || m["modulo_id"] != descriptorRPTPrueba.ModuloID ||
		m["consumidor"] != uso.Consumidor || m["uso_ref"] != uso.UsoRef ||
		m["reserva_recibo_ref"] != uso.ReservaReciboRef {
		t.Fatalf("material de uso no exacto: %+v %v", m, err)
	}
	d, err := s.Datos()
	if err != nil {
		t.Fatal(err)
	}
	d.Recurso.Ambitos["consumidor"] = "bolsa"
	ajena, err := domain.NuevaSolicitudAutorizacionLigadaV3(d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lector.ConsultarUsoCategoriaRPT(context.Background(), ports.OrdenUsoCategoriaRPT{
		Consulta: uso, Solicitud: ajena, Autorizacion: a}); !errors.Is(err, ports.ErrLecturaRPTDenegada) || iniciador.llamadas != 1 {
		t.Fatalf("ámbito de otro consumidor llegó a SQL: %v, llamadas=%d", err, iniciador.llamadas)
	}
}

func TestVersionHistoricaRPTRangoEnteroPositivoSQL117(t *testing.T) {
	p, entrada := publicacionRPTPrueba(t, 1, "categoria.uno")
	for _, version := range []int{1, maximoVersionMaterialRPT} {
		s, a := autorizacionYSolicitudRPTPrueba(t, accionLeerPublicacionRPT,
			tipoCatalogoRPT, descriptorRPTPrueba.CatalogoID, "")
		tx := &transaccionLecturaRPTPrueba{respuesta: reciboAusenciaRPTPrueba(t, a)}
		iniciador := &iniciadorLecturaRPTPrueba{tx: tx}
		lector, _ := nuevoLectorCategoriasRPTPostgreSQL(iniciador, descriptorRPTPrueba)
		consulta := ports.ConsultaPublicacionCategoriaRPT{Referencia: ports.ReferenciaPublicacionRPT{
			CatalogoID: p.CatalogoID, Version: version, HuellaSHA256: p.HuellaSHA256}, CategoriaID: entrada.Clave}
		r, err := lector.LeerPublicacionCategoriaRPT(context.Background(), ports.OrdenPublicacionCategoriaRPT{
			Consulta: consulta, Solicitud: s, Autorizacion: a})
		if err != nil || r.Encontrado || !tx.confirmada || iniciador.llamadas != 1 {
			t.Fatalf("versión %d válida no consultada: %+v %v", version, r, err)
		}
		var material map[string]any
		if err := json.Unmarshal([]byte(tx.argumentos[0].(string)), &material); err != nil ||
			material["version"] != float64(version) {
			t.Fatalf("versión %d no viajó como JSON number: %+v %v", version, material, err)
		}
		fueraDeRango := []int{0}
		if strconv.IntSize > 32 {
			demasiadoAlta := int64(maximoVersionMaterialRPT) + 1
			fueraDeRango = append(fueraDeRango, int(demasiadoAlta))
		}
		for _, fuera := range fueraDeRango {
			consulta.Referencia.Version = fuera
			if _, err := lector.LeerPublicacionCategoriaRPT(context.Background(), ports.OrdenPublicacionCategoriaRPT{
				Consulta: consulta, Solicitud: s, Autorizacion: a}); !errors.Is(err, ports.ErrLecturaRPTInvalida) || iniciador.llamadas != 1 {
				t.Fatalf("versión %d fuera de rango llegó a SQL: %v, llamadas=%d", fuera, err, iniciador.llamadas)
			}
		}
	}
}

func TestLecturasRPTRechazanMaterialAjenoAntesDeConsumirAD3(t *testing.T) {
	lista := ordenListaRPTPrueba(t)
	lista.Consulta.CursorCategoriaID = "categoria.dos"
	p, entrada := publicacionRPTPrueba(t, 1, "categoria.uno")
	sHistorica, aHistorica := autorizacionYSolicitudRPTPrueba(t, accionLeerPublicacionRPT,
		tipoCatalogoRPT, descriptorRPTPrueba.CatalogoID, "")
	historica := ports.OrdenPublicacionCategoriaRPT{
		Consulta: ports.ConsultaPublicacionCategoriaRPT{Referencia: ports.ReferenciaPublicacionRPT{
			CatalogoID: p.CatalogoID, Version: 2, HuellaSHA256: p.HuellaSHA256}, CategoriaID: entrada.Clave},
		Solicitud: sHistorica, Autorizacion: aHistorica,
	}
	sUso, aUso := autorizacionYSolicitudRPTPrueba(t, accionConsultarUsoRPT,
		tipoUsoRPT, "uso:prueba:uno", "contratacion_temporal")
	uso := ports.OrdenUsoCategoriaRPT{
		Consulta: ports.ConsultaUsoCategoriaRPT{Consumidor: "contratacion_temporal", UsoRef: "uso:prueba:uno",
			ReservaReciboRef: "recibo:otra-reserva"}, Solicitud: sUso, Autorizacion: aUso,
	}
	for _, caso := range []struct {
		nombre   string
		invocar  func(*LectorCategoriasRPTPostgreSQL) error
		material string
	}{
		{"lista", func(l *LectorCategoriasRPTPostgreSQL) error {
			_, err := l.ListarCategoriasHabilitadasRPT(context.Background(), lista)
			return err
		}, `"cursor_categoria_id":"categoria.dos"`},
		{"historica", func(l *LectorCategoriasRPTPostgreSQL) error {
			_, err := l.LeerPublicacionCategoriaRPT(context.Background(), historica)
			return err
		}, `"version":2`},
		{"uso", func(l *LectorCategoriasRPTPostgreSQL) error {
			_, err := l.ConsultarUsoCategoriaRPT(context.Background(), uso)
			return err
		}, `"reserva_recibo_ref":"recibo:otra-reserva"`},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &transaccionLecturaRPTPrueba{huellaMaterial: strings.Repeat("b", 64)}
			lector, err := nuevoLectorCategoriasRPTPostgreSQL(&iniciadorLecturaRPTPrueba{tx: tx}, descriptorRPTPrueba)
			if err != nil {
				t.Fatal(err)
			}
			if err := caso.invocar(lector); !errors.Is(err, ports.ErrLecturaRPTDenegada) ||
				tx.consultasCanon != 1 || tx.fachadas != 0 || tx.confirmada || !tx.revertida ||
				!strings.Contains(tx.materialCanonico, caso.material) {
				t.Fatalf("material ajeno alcanzó AD3: %v tx=%+v", err, tx)
			}
		})
	}
}

// Las categorías publicadas con espacio de nombres («categoria:rpt:...»), que la
// base admite, se listan y se leen; antes el dominio las tenía por no confiables.
func TestListaRPTAdmiteClavesConDosPuntosComoLaBase(t *testing.T) {
	p, e := publicacionRPTPrueba(t, 1, "categoria:rpt:administrativo")
	lista := listaRPTWire{AnclajePublicacion: jsonRPTPrueba(t, map[string]any{
		"catalogo_id": p.CatalogoID, "version": p.Version, "huella_sha256": p.HuellaSHA256,
		"documento_canonico": p.DocumentoCanonico}),
		Items: []json.RawMessage{jsonRPTPrueba(t, categoriaRPTWire{CategoriaID: e.Clave, CatalogoID: p.CatalogoID,
			Version: p.Version, HuellaSHA256: p.HuellaSHA256, Revision: 1, Estado: "habilitada",
			Etiqueta: e.Etiqueta, Definicion: jsonRPTPrueba(t, e)})},
		Publicaciones: []json.RawMessage{}}
	consulta := ports.ConsultaCategoriasHabilitadasRPT{CatalogoID: descriptorRPTPrueba.CatalogoID, Limite: 100}
	r, err := decodificarListaRPT(jsonRPTPrueba(t, lista), consulta, descriptorRPTPrueba)
	if err != nil || len(r.Categorias) != 1 || r.Categorias[0].CategoriaID != "categoria:rpt:administrativo" {
		t.Fatalf("categoría con «:» rechazada: %+v %v", r, err)
	}
	historica := jsonRPTPrueba(t, publicacionHistoricaRPTWire{Publicacion: jsonRPTPrueba(t, p), Entrada: jsonRPTPrueba(t, e),
		ControlActual: json.RawMessage("null")})
	h, err := decodificarPublicacionHistoricaRPT(historica, ports.ConsultaPublicacionCategoriaRPT{Referencia: p.referencia(), CategoriaID: e.Clave}, descriptorRPTPrueba)
	if err != nil || !h.Encontrado || h.Entrada.Clave != e.Clave {
		t.Fatalf("publicación histórica con «:» rechazada: %+v %v", h, err)
	}
}
