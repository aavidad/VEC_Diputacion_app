package postgresimportacionconvoca

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

func TestSerializacionActaEsExactaYRestaurable(t *testing.T) {
	acta := actaPostgreSQLPrueba(strings.Repeat("a", 64), 1)
	contenido, err := serializarActa(acta)
	if err != nil {
		t.Fatalf("serializar acta: %v", err)
	}
	restaurada, err := deserializarActa(contenido)
	if err != nil || restaurada.ActaRef != acta.ActaRef ||
		restaurada.RegistradaEn != acta.RegistradaEn {
		t.Fatalf("restaurar acta: %#v, %v", restaurada, err)
	}
	adulterada := bytes.Replace(
		contenido, []byte(`"acta_ref"`), []byte(`"campo_desconocido"`), 1,
	)
	if _, err := deserializarActa(adulterada); !errors.Is(err, ErrResultadoNoConfiable) {
		t.Fatalf("JSON abierto aceptado: %v", err)
	}
}

func TestFilasProtegidasRechazanLimitesHuellaYDuplicados(t *testing.T) {
	filas := []FilaStagingProtegida{filaProtegidaPrueba(2)}
	if err := validarFilasProtegidas(filas); err != nil {
		t.Fatalf("fila protegida valida: %v", err)
	}
	duplicadas := append(clonarFilasProtegidas(filas), filaProtegidaPrueba(2))
	if !errors.Is(validarFilasProtegidas(duplicadas), ErrMaterialNoConfiable) {
		t.Fatal("numero de fila duplicado aceptado")
	}
	excesiva := filaProtegidaPrueba(3)
	excesiva.ContenidoCifrado = make([]byte, maximoBytesCifradosPorFila+1)
	if !errors.Is(validarFilasProtegidas([]FilaStagingProtegida{excesiva}), ErrMaterialNoConfiable) {
		t.Fatal("cifrado excesivo aceptado")
	}
	filas[0].DerivacionDocumentoHMACSHA256 = make([]byte, 32)
	if !errors.Is(validarFilasProtegidas(filas), ErrMaterialNoConfiable) {
		t.Fatal("derivacion nula aceptada")
	}
}

func TestFilasProtegidasAdmiten100001ConPresupuestoAcotado(t *testing.T) {
	nonce := make([]byte, 12)
	cifrado := make([]byte, 16)
	derivacion := bytes.Repeat([]byte{1}, 32)
	atestacion := bytes.Repeat([]byte{2}, 32)
	filas := make([]FilaStagingProtegida, maximoFilasStaging)
	for i := range filas {
		filas[i] = FilaStagingProtegida{
			Numero: i + 2, EsquemaProteccion: EsquemaProteccionStagingV1,
			ClaveRef:           "kms:clave:convoca:prueba:v1",
			ClaveDerivacionRef: "kms:derivacion:convoca:prueba:v1",
			ClaveAtestacionRef: "kms:atestacion:convoca:prueba:v1",
			Nonce:              nonce, ContenidoCifrado: cifrado,
			DerivacionDocumentoHMACSHA256: derivacion,
			AtestacionFilaHMACSHA256:      atestacion,
		}
	}
	if err := validarFilasProtegidas(filas); err != nil {
		t.Fatalf("frontera de 100001 filas rechazada: %v", err)
	}
	contenido, err := serializarFilasProtegidas(filas)
	if err != nil || len(contenido) > maximoBytesJSONFilas {
		t.Fatalf("frontera serializada de 100001 filas rechazada: bytes=%d error=%v",
			len(contenido), err)
	}
	borrarBytes(contenido)
	filas = append(filas, FilaStagingProtegida{})
	if !errors.Is(validarFilasProtegidas(filas), ErrMaterialNoConfiable) {
		t.Fatal("frontera de 100002 filas aceptada")
	}
}

func TestFilasProtegidasAplicanPresupuestoTotalAntesDeSerializar(t *testing.T) {
	cifrado := make([]byte, maximoBytesCifradosPorFila)
	numeroFilas := maximoBytesProtegidosLote/maximoBytesCifradosPorFila + 2
	filas := make([]FilaStagingProtegida, numeroFilas)
	for i := range filas {
		filas[i] = FilaStagingProtegida{
			Numero: i + 2, EsquemaProteccion: EsquemaProteccionStagingV1,
			ClaveRef:           "kms:clave:convoca:prueba:v1",
			ClaveDerivacionRef: "kms:derivacion:convoca:prueba:v1",
			ClaveAtestacionRef: "kms:atestacion:convoca:prueba:v1",
			Nonce:              make([]byte, 12), ContenidoCifrado: cifrado,
			DerivacionDocumentoHMACSHA256: bytes.Repeat([]byte{1}, 32),
			AtestacionFilaHMACSHA256:      bytes.Repeat([]byte{2}, 32),
		}
	}
	if !errors.Is(validarFilasProtegidas(filas), ErrMaterialNoConfiable) {
		t.Fatal("presupuesto binario total excedido fue aceptado")
	}
}

func TestConstructoresSeparanImportacionYRecuperacion(t *testing.T) {
	if _, err := nuevoRepositorioPostgreSQL(nil, protectorNuloPrueba{}); !errors.Is(err, ErrRepositorioNoDisponible) {
		t.Fatalf("pool nulo aceptado: %v", err)
	}
	if _, err := nuevoRepositorioPostgreSQL(iniciadorNuloPrueba{}, nil); !errors.Is(err, ErrProtectorRequerido) {
		t.Fatalf("protector nulo aceptado: %v", err)
	}
	if _, err := nuevoRepositorioRecuperacionPostgreSQL(nil, protectorNuloPrueba{}); !errors.Is(err, ErrRepositorioNoDisponible) {
		t.Fatalf("recuperacion sin pool aceptada: %v", err)
	}
	if _, err := nuevoRepositorioRecuperacionPostgreSQL(iniciadorNuloPrueba{}, nil); !errors.Is(err, ErrProtectorRequerido) {
		t.Fatalf("recuperacion sin protector aceptada: %v", err)
	}
}

func TestRepositorioBorraMaterialParcialCuandoFallaElProtector(t *testing.T) {
	material := []FilaStagingProtegida{filaProtegidaPrueba(2)}
	protector := &protectorParcialConErrorPrueba{filas: material}
	repositorio, err := nuevoRepositorioPostgreSQL(
		iniciadorNuloPrueba{}, protector,
	)
	if err != nil {
		t.Fatalf("construir repositorio: %v", err)
	}
	lote := loteIntegracion(
		strings.Repeat("b", 64),
		time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC),
		1,
	)
	if _, _, err := repositorio.GuardarSiAusente(
		context.Background(), lote,
	); !errors.Is(err, ErrProteccionNoDisponible) {
		t.Fatalf("error del protector no propagado de forma saneada: %v", err)
	}
	for _, valor := range [][]byte{
		material[0].Nonce,
		material[0].ContenidoCifrado,
		material[0].DerivacionDocumentoHMACSHA256,
		material[0].AtestacionFilaHMACSHA256,
	} {
		for _, octeto := range valor {
			if octeto != 0 {
				t.Fatal("material parcial del protector permanecio en memoria")
			}
		}
	}
}

func filaProtegidaPrueba(numero int) FilaStagingProtegida {
	return FilaStagingProtegida{
		Numero: numero, EsquemaProteccion: EsquemaProteccionStagingV1,
		ClaveRef:           "kms:clave:convoca:prueba:v1",
		ClaveDerivacionRef: "kms:derivacion:convoca:prueba:v1",
		ClaveAtestacionRef: "kms:atestacion:convoca:prueba:v1",
		Nonce:              make([]byte, 12), ContenidoCifrado: make([]byte, 32),
		DerivacionDocumentoHMACSHA256: bytes.Repeat([]byte{1}, 32),
		AtestacionFilaHMACSHA256:      bytes.Repeat([]byte{2}, 32),
	}
}

func actaPostgreSQLPrueba(huella string, filas int) dominio.ActaImportacion {
	return dominio.ActaImportacion{
		ActaRef:             "acta:importacion-convoca:" + huella,
		ImportacionRef:      "importacion:convoca:" + huella,
		HuellaFicheroSHA256: huella, NombreFichero: "sintetico.xls",
		FicheroCustodiadoRef: "almacen:objeto:convoca:" + huella,
		ActorRef:             "actor:rrhh:prueba",
		RegistradaEn:         time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC),
		Esquema:              dominio.EsquemaResumenPersona,
		FilasLeidas:          filas, FilasAceptadas: filas,
		Procedencia: dominio.NuevaProcedenciaNoAutoritativa(),
	}
}

type iniciadorNuloPrueba struct{}

func (iniciadorNuloPrueba) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return nil, ErrRepositorioNoDisponible
}

type protectorNuloPrueba struct{}

func (protectorNuloPrueba) ProtegerStaging(
	context.Context,
	SolicitudProteccionStaging,
) (ResultadoProteccionStaging, error) {
	return ResultadoProteccionStaging{}, ErrProteccionNoDisponible
}

func (protectorNuloPrueba) RecuperarStaging(
	context.Context,
	SolicitudRecuperacionStaging,
) ([]dominio.FilaAceptada, error) {
	return nil, ErrProteccionNoDisponible
}

type protectorParcialConErrorPrueba struct {
	filas []FilaStagingProtegida
}

func (p *protectorParcialConErrorPrueba) ProtegerStaging(
	context.Context,
	SolicitudProteccionStaging,
) (ResultadoProteccionStaging, error) {
	return ResultadoProteccionStaging{Filas: p.filas},
		ErrProteccionNoDisponible
}

func (*protectorParcialConErrorPrueba) RecuperarStaging(
	context.Context,
	SolicitudRecuperacionStaging,
) ([]dominio.FilaAceptada, error) {
	return nil, ErrProteccionNoDisponible
}
