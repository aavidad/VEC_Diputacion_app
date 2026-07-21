package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestLecturaPostgreSQLConfirmaAntesDelKMSYNoRetieneTransaccion(t *testing.T) {
	fila := filaCifradaLecturaPostgreSQLPrueba(t)
	tx := &transaccionLecturaBorradorPostgreSQLPrueba{
		fila: filaLecturaBorradorPostgreSQLPrueba{fila: fila},
	}
	pool := &iniciadorLecturaBorradorPostgreSQLPrueba{tx: tx}
	descifrador := &descifradorBloqueadoLecturaPostgreSQLPrueba{
		tx: tx, iniciado: make(chan struct{}),
	}
	repositorio, err := nuevoRepositorioLecturaBorradoresPostgreSQL(pool, descifrador)
	if err != nil {
		t.Fatal(err)
	}
	contexto, capacidad := solicitudLecturaBorradorPostgreSQLPrueba(
		t, gobiernoconvocatorias.AccionConsultarBorradorGobernado,
	)
	solicitud := gobiernoconvocatorias.SolicitudDetalleBorradorGobernada{
		Contexto: contexto,
		Selector: puertosbolsa.SelectorVersionConvocatoriaExacta{
			ID: "proceso:bolsa:auxiliar-2026-1", Secuencia: 1,
		},
		Capacidad: capacidad,
	}
	ctx, cancelar := context.WithCancel(context.Background())
	hecho := make(chan error, 1)
	go func() {
		_, err := repositorio.ObtenerBorradorGobernado(ctx, solicitud)
		hecho <- err
	}()
	select {
	case <-descifrador.iniciado:
	case err := <-hecho:
		t.Fatalf("la lectura fallo antes del KMS: %v", err)
	case <-time.After(time.Second):
		t.Fatal("el KMS no fue invocado")
	}
	cancelar()
	if err := <-hecho; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion superior ocultada: %v", err)
	}
	if !descifrador.vioCommit || tx.confirmaciones != 1 || tx.reversiones != 0 ||
		pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite ||
		tx.configuraciones != 1 || !strings.Contains(tx.consulta, funcionObtenerBorradorPostgreSQL) {
		t.Fatalf("KMS dentro del snapshot: vio_commit=%v tx=%+v opciones=%+v", descifrador.vioCommit, tx, pool.opciones)
	}
}

func TestLecturaPostgreSQLAcotaKMSBloqueadoConPlazoLocal(t *testing.T) {
	descifrador := &descifradorBloqueadoLecturaPostgreSQLPrueba{}
	inicio := time.Now()
	_, err := descifrarBorradorConPlazo(
		context.Background(), descifrador,
		gobiernoconvocatorias.SolicitudDescifradoBorradorDurable{}, 20*time.Millisecond,
	)
	if !errors.Is(err, gobiernoconvocatorias.ErrOperacionBorradorEnCurso) ||
		time.Since(inicio) > time.Second {
		t.Fatalf("KMS bloqueado sin limite: err=%v duracion=%s", err, time.Since(inicio))
	}
}

func TestLecturaPostgreSQLFallaAntesDelPoolConCancelacionOCapacidadInvalida(t *testing.T) {
	pool := &iniciadorLecturaBorradorPostgreSQLPrueba{}
	descifrador := &descifradorBloqueadoLecturaPostgreSQLPrueba{}
	repositorio, err := nuevoRepositorioLecturaBorradoresPostgreSQL(pool, descifrador)
	if err != nil {
		t.Fatal(err)
	}
	selector := puertosbolsa.SelectorVersionConvocatoriaExacta{ID: "convocatoria-prueba", Secuencia: 1}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	_, err = repositorio.ObtenerBorradorGobernado(
		ctx, gobiernoconvocatorias.SolicitudDetalleBorradorGobernada{Selector: selector},
	)
	if !errors.Is(err, context.Canceled) || pool.inicios != 0 {
		t.Fatalf("cancelacion alcanzo PostgreSQL: err=%v inicios=%d", err, pool.inicios)
	}
	_, err = repositorio.ObtenerBorradorGobernado(
		context.Background(),
		gobiernoconvocatorias.SolicitudDetalleBorradorGobernada{Selector: selector},
	)
	if !errors.Is(err, gobiernoconvocatorias.ErrPreautorizacionLecturaBorrador) || pool.inicios != 0 {
		t.Fatalf("capacidad invalida alcanzo PostgreSQL: err=%v inicios=%d", err, pool.inicios)
	}
}

func TestListadoPostgreSQLRevalidaSelectorAntesDelPool(t *testing.T) {
	pool := &iniciadorLecturaBorradorPostgreSQLPrueba{}
	descifrador := &descifradorBloqueadoLecturaPostgreSQLPrueba{}
	repositorio, err := nuevoRepositorioLecturaBorradoresPostgreSQL(pool, descifrador)
	if err != nil {
		t.Fatal(err)
	}
	contexto, capacidad := solicitudLecturaBorradorPostgreSQLPrueba(
		t, gobiernoconvocatorias.AccionListarBorradoresGobernados,
	)
	for nombre, selector := range map[string]gobiernoconvocatorias.SelectorListaBorradores{
		"cursor no opaco": {Limite: 10, Cursor: "cursor-libre"},
		"texto enorme":    {Limite: 10, Texto: strings.Repeat("a", 181)},
		"categoria":       {Limite: 10, Categoria: "Auxiliar"},
	} {
		t.Run(nombre, func(t *testing.T) {
			_, err := repositorio.ListarBorradoresGobernados(
				context.Background(), gobiernoconvocatorias.SolicitudListadoBorradoresGobernada{
					Contexto: contexto, Selector: selector, Capacidad: capacidad,
				},
			)
			if !errors.Is(err, gobiernoconvocatorias.ErrLecturaBorradoresGobernadaInvalida) || pool.inicios != 0 {
				t.Fatalf("selector alcanzo pool: err=%v inicios=%d", err, pool.inicios)
			}
		})
	}
}

func TestDTOsLecturaPostgreSQLSonCerradosYSnakeCase(t *testing.T) {
	selector, err := json.Marshal(selectorListaBorradoresPostgreSQL{})
	if err != nil {
		t.Fatal(err)
	}
	lectura, err := json.Marshal(lecturaBorradorPostgreSQL{})
	if err != nil {
		t.Fatal(err)
	}
	exigirClavesJSONLecturaPostgreSQLPrueba(t, selector, []string{
		"categoria", "cursor", "limite", "texto",
	})
	exigirClavesJSONLecturaPostgreSQLPrueba(t, lectura, []string{
		"accion", "decision_ref", "huella_decision_sha256", "atestacion_ref",
		"atestacion_version", "estado_atestacion", "huella_atestacion_sha256",
		"recurso_ref", "organizacion_ref", "unidad_gestion_ref",
	})
}

func exigirClavesJSONLecturaPostgreSQLPrueba(t *testing.T, contenido []byte, esperadas []string) {
	t.Helper()
	var objeto map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &objeto); err != nil {
		t.Fatal(err)
	}
	if len(objeto) != len(esperadas) {
		t.Fatalf("DTO abierto o incompleto: %s", contenido)
	}
	for _, clave := range esperadas {
		if _, existe := objeto[clave]; !existe {
			t.Fatalf("falta clave %q en %s", clave, contenido)
		}
	}
}

func listaLecturaPostgreSQLPrueba(
	t *testing.T,
) (gobiernoconvocatorias.SelectorListaBorradores, listaBorradoresPersistida) {
	t.Helper()
	version := versionConsultaConvocatoriaPostgreSQLPrueba(t)
	estado, err := puertosbolsa.EstadoVersionConvocatoria(version)
	if err != nil {
		t.Fatal(err)
	}
	selector := gobiernoconvocatorias.SelectorListaBorradores{
		Limite: 10, Texto: "auxiliar", Categoria: "auxiliar_administrativo",
	}
	var lista listaBorradoresPersistida
	lista.Esquema = "vec.bolsa.borradores.lista.v1"
	lista.Selector = selectorListaBorradoresPostgreSQL{
		Categoria: selector.Categoria, Cursor: selector.Cursor,
		Limite: selector.Limite, Texto: selector.Texto,
	}
	lista.Paginacion.Limite, lista.Paginacion.Total = selector.Limite, 1
	lista.Capacidades = gobiernoconvocatorias.CapacidadesGlobalesBorrador{Consultar: true}
	lista.Elementos = []filaBorradorPersistida{{
		Estado: estado, ETag: `"` + strconv.Itoa(estado.Revision) + `-` + estado.HuellaEstadoSHA256 + `"`,
		CodigoVersionPublica: version.CodigoVersionPublica,
		IdentificadorPublico: version.Contenido.IdentificadorPublico,
		Titulo:               version.Contenido.Titulo, Tipo: version.Contenido.Tipo,
		Categorias:    append([]string(nil), version.Contenido.Categorias...),
		ExpedienteRef: version.ExpedienteRef,
		CreadaEn:      version.CreadaEn.UTC().Truncate(time.Microsecond),
		ActualizadaEn: version.CreadaEn.UTC().Truncate(time.Microsecond),
		NumeroPlazos:  len(version.Contenido.Plazos), NumeroRequisitos: len(version.Contenido.Requisitos),
		NumeroDocumentos: len(version.Configuracion.Documentos), NumeroAyudas: len(version.Contenido.Ayuda),
		Capacidades: gobiernoconvocatorias.CapacidadesFilaBorrador{Consultar: true, Actualizar: true},
	}}
	return selector, lista
}

func TestRestaurarListaLecturaPostgreSQLAceptaCapacidadFuturaYRechazaTamper(t *testing.T) {
	selector, lista := listaLecturaPostgreSQLPrueba(t)
	valida, err := json.Marshal(lista)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := restaurarListaBorradores(valida, selector)
	if err != nil || len(resultado.Elementos) != 1 || !resultado.Elementos[0].Capacidades.Actualizar {
		t.Fatalf("lista valida rechazada: resultado=%+v err=%v", resultado, err)
	}
	casos := map[string]func(*listaBorradoresPersistida){
		"etag":     func(p *listaBorradoresPersistida) { p.Elementos[0].ETag = `"1-` + strings.Repeat("0", 64) + `"` },
		"contador": func(p *listaBorradoresPersistida) { p.Elementos[0].NumeroDocumentos = -1 },
		"total":    func(p *listaBorradoresPersistida) { p.Paginacion.Total = -1 },
		"instante": func(p *listaBorradoresPersistida) {
			p.Elementos[0].ActualizadaEn = time.Date(2026, 7, 18, 11, 0, 0, 0, time.FixedZone("CEST", 7200))
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			copia := lista
			copia.Elementos = append([]filaBorradorPersistida(nil), lista.Elementos...)
			mutar(&copia)
			contenido, err := json.Marshal(copia)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := restaurarListaBorradores(contenido, selector); !errors.Is(err, gobiernoconvocatorias.ErrResultadoBorradorInseguro) {
				t.Fatalf("alteracion aceptada: %v", err)
			}
		})
	}
}

func TestMetadatosLecturaPostgreSQLLigadosAlAgregado(t *testing.T) {
	version := versionConsultaConvocatoriaPostgreSQLPrueba(t)
	estado, err := puertosbolsa.EstadoVersionConvocatoria(version)
	if err != nil {
		t.Fatal(err)
	}
	selector := puertosbolsa.SelectorVersionConvocatoriaExacta{ID: version.ID, Secuencia: version.Secuencia}
	base := metadatosBorrador{
		Estado: estado, ETag: `"` + strconv.Itoa(estado.Revision) + `-` + estado.HuellaEstadoSHA256 + `"`,
		CodigoVersionPublica: version.CodigoVersionPublica,
		IdentificadorPublico: version.Contenido.IdentificadorPublico,
		ExpedienteRef:        version.ExpedienteRef,
	}
	base.Ambito.OrganizacionRef = version.AmbitoOrganizativo.OrganizacionRef()
	base.Ambito.UnidadGestionRef = version.AmbitoOrganizativo.UnidadGestionRef()
	if !base.coincide(version, selector) {
		t.Fatal("metadatos exactos rechazados")
	}
	alteraciones := []func(*metadatosBorrador){
		func(m *metadatosBorrador) { m.CodigoVersionPublica = "v-ajena" },
		func(m *metadatosBorrador) { m.IdentificadorPublico = "identificador-ajeno" },
		func(m *metadatosBorrador) { m.Ambito.OrganizacionRef = "org_otraorganizacion" },
		func(m *metadatosBorrador) { m.Ambito.UnidadGestionRef = "uni_otraunidadgestion" },
		func(m *metadatosBorrador) { m.ExpedienteRef = "expediente:ajeno" },
		func(m *metadatosBorrador) { m.ETag = `"1-` + strings.Repeat("0", 64) + `"` },
	}
	for indice, alterar := range alteraciones {
		copia := base
		alterar(&copia)
		if copia.coincide(version, selector) {
			t.Fatalf("metadato ajeno %d aceptado", indice)
		}
	}
}

func TestBuffersLecturaPostgreSQLSeBorran(t *testing.T) {
	fila := filaCifradaLecturaPostgreSQLPrueba(t)
	contenidos := [][]byte{fila.metadatos, fila.aad, fila.perfil, fila.atestacion, fila.procedencia, fila.envuelto, fila.nonce, fila.cifrado}
	fila.borrar()
	for indice, contenido := range contenidos {
		if !reflect.DeepEqual(contenido, make([]byte, len(contenido))) {
			t.Fatalf("buffer %d no borrado", indice)
		}
	}
}
