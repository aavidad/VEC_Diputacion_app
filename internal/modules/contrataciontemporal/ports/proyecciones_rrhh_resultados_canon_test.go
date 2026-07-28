package ports_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	huellaContenidoCuadroVacio  = "568056c5d1a9b0651d2bc85f7dcc6e6dc3a71b12ffe8f831cb7ea5ffd51aa0c4"
	huellaResultadoCuadroVacio  = "cb8ad45d7c31faa5100a249a840e66671b0de2319d23fc1c8878e56da7076ee0"
	huellaContenidoCuadroCursor = "acf18cd8e268f93f451f6ef6566e617c9128ba84c068b5b3124132f5f1ef5f07"
	huellaResultadoCuadroCursor = "e77c8c791e996bd33783716a65cf6ad36868ea03f916c368263ea1031d2a50be"
)

func TestCanonContenidoYResultadoCuadroVacioConservanVectoresDorados(
	t *testing.T,
) {
	t.Parallel()
	pagina := ports.PaginaCuadroRRHH{GeneradaEn: instantePuertosRRHH()}
	contenido, err := pagina.ExportarContenidoCanonicoParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	canonContenido := "" +
		"VEC-CT-CONTENIDO-CUADRO-RRHH-V1\n" +
		"27:2026-07-26T08:00:00.000000Z\n" +
		"1:0\n" +
		"1:0\n" +
		"0:\n"
	comprobarVectorCanonSQL(
		t, contenido, canonContenido, huellaContenidoCuadroVacio,
	)
	if contenido.Dominio() != ports.DominioCanonContenidoCuadroRRHH ||
		contenido.Version() != ports.VersionCanonContenidoResultadoRRHH {
		t.Fatalf(
			"identidad de contenido inestable: %s/v%d",
			contenido.Dominio(), contenido.Version(),
		)
	}

	resultado, err := contenido.ExportarResultadoCanonicoParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	canonResultado := "" +
		"VEC-CT-RESULTADO-CONSULTA-RRHH-V1\n" +
		"6:cuadro\n" +
		"27:2026-07-26T08:00:00.000000Z\n" +
		"1:0\n" +
		"64:" + huellaContenidoCuadroVacio + "\n" +
		"0:\n"
	comprobarVectorCanonSQL(
		t, resultado, canonResultado, huellaResultadoCuadroVacio,
	)
	if resultado.Dominio() != ports.DominioCanonResultadoConsultaRRHH ||
		resultado.Version() != ports.VersionCanonResultadoConsultaRRHH ||
		resultado.TipoConsulta() != "cuadro" ||
		!resultado.GeneradaEn().Equal(pagina.GeneradaEn) ||
		resultado.Total() != 0 ||
		resultado.ContenidoHuellaSHA256() != huellaContenidoCuadroVacio ||
		resultado.CursorHuellaSHA256() != "" {
		t.Fatalf("argumentos del resultado divergentes: %#v", resultado)
	}
}

func TestCanonContenidoCuadroCursorUsaSoloHuellaBinariaYOrdenCompleto(
	t *testing.T,
) {
	t.Parallel()
	cursor := cursorResultadoRRHHPrueba()
	pagina := paginaContenidoCuadroRRHHPrueba(cursor)
	contenido, err := pagina.ExportarContenidoCanonicoParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	if contenido.HuellaSHA256() != huellaContenidoCuadroCursor {
		t.Fatalf("huella de contenido inestable: %s", contenido.HuellaSHA256())
	}
	canon := contenido.BytesCanonicos()
	if len(canon) != 434 || bytes.Contains(canon, []byte(cursor)) {
		t.Fatalf("canon con tamaño inesperado o cursor claro: %d", len(canon))
	}
	materialCursor := bytes.Repeat([]byte{0xff}, sha256.Size)
	huellaCursor := sha256.Sum256(materialCursor)
	encuadreBinario := append([]byte("32:"), huellaCursor[:]...)
	encuadreBinario = append(encuadreBinario, '\n')
	if !bytes.Contains(canon, encuadreBinario) {
		t.Fatal("el canon no contiene la huella binaria encuadrada del cursor")
	}

	resultado, err := contenido.ExportarResultadoCanonicoParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	huellaRepresentacion := sha256.Sum256([]byte(cursor))
	if resultado.HuellaSHA256() != huellaResultadoCuadroCursor ||
		resultado.Total() != 1 ||
		resultado.CursorHuellaSHA256() != hex.EncodeToString(huellaCursor[:]) ||
		resultado.CursorHuellaSHA256() ==
			hex.EncodeToString(huellaRepresentacion[:]) ||
		bytes.Contains(resultado.BytesCanonicos(), []byte(cursor)) {
		t.Fatalf("resultado divergente o con cursor claro: %#v", resultado)
	}
}

func TestCanonContenidoCuadroMutaConCadaCampoYCardinalidad(
	t *testing.T,
) {
	t.Parallel()
	base := paginaContenidoCuadroRRHHPrueba("")
	exportacionBase, err := base.ExportarContenidoCanonicoParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	mutaciones := map[string]func(*ports.PaginaCuadroRRHH){
		"generada_en": func(p *ports.PaginaCuadroRRHH) {
			p.GeneradaEn = p.GeneradaEn.Add(time.Minute)
		},
		"expediente_ref": func(p *ports.PaginaCuadroRRHH) {
			p.Expedientes[0].ExpedienteRef = "expediente:rrhh:002"
		},
		"organizacion_ref": func(p *ports.PaginaCuadroRRHH) {
			p.Expedientes[0].OrganizacionRef = "organizacion:otra"
		},
		"numero_visible": func(p *ports.PaginaCuadroRRHH) {
			p.Expedientes[0].NumeroVisible = "2026/CT-002"
		},
		"version": func(p *ports.PaginaCuadroRRHH) {
			p.Expedientes[0].Version = 2
		},
		"flujo_ref": func(p *ports.PaginaCuadroRRHH) {
			p.Expedientes[0].FlujoRef = "flujo:rrhh:002"
		},
		"flujo_version": func(p *ports.PaginaCuadroRRHH) {
			p.Expedientes[0].FlujoVersion = 2
		},
		"flujo_huella": func(p *ports.PaginaCuadroRRHH) {
			p.Expedientes[0].FlujoHuella = strings.Repeat("b", 64)
		},
		"fase": func(p *ports.PaginaCuadroRRHH) {
			p.Expedientes[0].FaseClave = "analisis"
		},
		"estado": func(p *ports.PaginaCuadroRRHH) {
			p.Expedientes[0].EstadoClave = domain.EstadoCompletado
		},
		"centro": func(p *ports.PaginaCuadroRRHH) {
			p.Expedientes[0].CentroRef = "centro:rrhh:002"
		},
		"categoria": func(p *ports.PaginaCuadroRRHH) {
			p.Expedientes[0].CategoriaRef = "categoria:rrhh:002"
		},
		"modalidad": func(p *ports.PaginaCuadroRRHH) {
			p.Expedientes[0].ModalidadClave = "sustitucion"
		},
		"unidad": func(p *ports.PaginaCuadroRRHH) {
			p.Expedientes[0].UnidadRef = "unidad:rrhh:002"
		},
		"creado_en": func(p *ports.PaginaCuadroRRHH) {
			p.Expedientes[0].CreadoEn =
				p.Expedientes[0].CreadoEn.Add(-time.Minute)
		},
		"actualizado_en": func(p *ports.PaginaCuadroRRHH) {
			p.Expedientes[0].ActualizadoEn =
				p.Expedientes[0].ActualizadoEn.Add(time.Minute)
		},
		"hay_mas_cursor": func(p *ports.PaginaCuadroRRHH) {
			p.HayMas = true
			p.CursorSiguiente = cursorResultadoRRHHPrueba()
		},
		"cardinalidad": func(p *ports.PaginaCuadroRRHH) {
			segundo := p.Expedientes[0]
			segundo.ExpedienteRef = "expediente:rrhh:000"
			p.Expedientes = append(p.Expedientes, segundo)
		},
	}
	for nombre, mutar := range mutaciones {
		nombre, mutar := nombre, mutar
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			mutada := clonarPaginaContenidoCuadroRRHH(base)
			mutar(&mutada)
			exportacion, err := mutada.ExportarContenidoCanonicoParaSQL()
			if err != nil {
				t.Fatalf("mutación válida rechazada: %v", err)
			}
			if exportacion.HuellaSHA256() == exportacionBase.HuellaSHA256() {
				t.Fatal("la huella no ligó el campo mutado")
			}
		})
	}
}

func TestCanonContenidoCuadroRechazaOrdenDuplicadosYEstadosImposibles(
	t *testing.T,
) {
	t.Parallel()
	base := paginaContenidoCuadroRRHHPrueba("")
	segundo := base.Expedientes[0]
	segundo.ExpedienteRef = "expediente:rrhh:000"
	base.Expedientes = append(base.Expedientes, segundo)

	casos := map[string]func(*ports.PaginaCuadroRRHH){
		"orden": func(p *ports.PaginaCuadroRRHH) {
			p.Expedientes[0], p.Expedientes[1] =
				p.Expedientes[1], p.Expedientes[0]
		},
		"duplicado": func(p *ports.PaginaCuadroRRHH) {
			p.Expedientes[1].ExpedienteRef = p.Expedientes[0].ExpedienteRef
		},
		"cursor_sin_hay_mas": func(p *ports.PaginaCuadroRRHH) {
			p.CursorSiguiente = cursorResultadoRRHHPrueba()
		},
		"hay_mas_sin_cursor": func(p *ports.PaginaCuadroRRHH) {
			p.HayMas = true
		},
		"actualizacion_futura": func(p *ports.PaginaCuadroRRHH) {
			p.Expedientes[0].ActualizadoEn = p.GeneradaEn.Add(time.Microsecond)
		},
	}
	for nombre, mutar := range casos {
		nombre, mutar := nombre, mutar
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			mutada := clonarPaginaContenidoCuadroRRHH(base)
			mutar(&mutada)
			if _, err := mutada.ExportarContenidoCanonicoParaSQL(); !errors.Is(
				err, ports.ErrResultadoConsultaRRHHNoConfiable,
			) {
				t.Fatalf("estado imposible aceptado: %v", err)
			}
		})
	}

	vaciaConCursor := ports.PaginaCuadroRRHH{
		GeneradaEn:      instantePuertosRRHH(),
		HayMas:          true,
		CursorSiguiente: cursorResultadoRRHHPrueba(),
	}
	if _, err := vaciaConCursor.ExportarContenidoCanonicoParaSQL(); !errors.Is(
		err, ports.ErrResultadoConsultaRRHHNoConfiable,
	) {
		t.Fatalf("cursor sobre resultado vacío aceptado: %v", err)
	}
}

func TestCanonesResultadoCuadroSonOpacosYCopiasDefensivas(t *testing.T) {
	t.Parallel()
	cursor := cursorResultadoRRHHPrueba()
	pagina := paginaContenidoCuadroRRHHPrueba(cursor)
	contenido, err := pagina.ExportarContenidoCanonicoParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := contenido.ExportarResultadoCanonicoParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	originalContenido := contenido.BytesCanonicos()
	originalResultado := resultado.BytesCanonicos()
	copia := contenido.BytesCanonicos()
	copia[0] ^= 0xff
	pagina.Expedientes[0].NumeroVisible = "2026/ALTERADO"
	pagina.CursorSiguiente = strings.Repeat("A", 43)
	if !bytes.Equal(contenido.BytesCanonicos(), originalContenido) ||
		!bytes.Equal(resultado.BytesCanonicos(), originalResultado) {
		t.Fatal("las exportaciones conservaron alias mutables")
	}

	for nombre, exportacion := range map[string]canonRRHHParaSQL{
		"contenido": contenido,
		"resultado": resultado,
	} {
		comprobarOpacidadCanonSQL(
			t, nombre, exportacion,
			[]string{cursor, "2026/CT-001", contenido.HuellaSHA256()},
		)
		for _, representacion := range []string{
			fmt.Sprintf("%v", exportacion),
			fmt.Sprintf("%+v", exportacion),
			fmt.Sprintf("%#v", exportacion),
		} {
			if strings.Contains(representacion, cursor) {
				t.Fatalf("%s filtró el cursor: %q", nombre, representacion)
			}
		}
	}

	var cero ports.ExportacionCanonicaContenidoCuadroRRHH
	if _, err := cero.ExportarResultadoCanonicoParaSQL(); !errors.Is(
		err, ports.ErrResultadoConsultaRRHHNoConfiable,
	) {
		t.Fatalf("exportación cero creó un resultado: %v", err)
	}
}

func paginaContenidoCuadroRRHHPrueba(
	cursor string,
) ports.PaginaCuadroRRHH {
	generadaEn := instantePuertosRRHH()
	resumen := ports.ResumenExpedienteRRHH{
		ExpedienteRef:   "expediente:rrhh:001",
		OrganizacionRef: "organizacion:diputacion-granada",
		NumeroVisible:   "2026/CT-001", Version: 1,
		FlujoRef: "flujo:rrhh:001", FlujoVersion: 1,
		FlujoHuella: strings.Repeat("a", 64),
		FaseClave:   "solicitud", EstadoClave: domain.EstadoEnCurso,
		CentroRef: "centro:rrhh:001", CategoriaRef: "categoria:rrhh:001",
		ModalidadClave: "interinidad", UnidadRef: "unidad:rrhh:001",
		CreadoEn:      generadaEn.Add(-time.Hour),
		ActualizadoEn: generadaEn.Add(-30 * time.Minute),
	}
	return ports.PaginaCuadroRRHH{
		GeneradaEn:  generadaEn,
		Expedientes: []ports.ResumenExpedienteRRHH{resumen},
		HayMas:      cursor != "", CursorSiguiente: cursor,
	}
}

func clonarPaginaContenidoCuadroRRHH(
	pagina ports.PaginaCuadroRRHH,
) ports.PaginaCuadroRRHH {
	pagina.Expedientes = append(
		[]ports.ResumenExpedienteRRHH(nil),
		pagina.Expedientes...,
	)
	return pagina
}

func cursorResultadoRRHHPrueba() string {
	return base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0xff}, 32))
}
