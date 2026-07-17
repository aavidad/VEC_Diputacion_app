package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

type transaccionConsultaConvocatoriaPrueba struct {
	pgx.Tx
	fila           pgx.Row
	consulta       string
	argumentos     []any
	confirmaciones int
	reversiones    int
}

func (t *transaccionConsultaConvocatoriaPrueba) Exec(
	context.Context, string, ...any,
) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func (t *transaccionConsultaConvocatoriaPrueba) QueryRow(
	_ context.Context, consulta string, argumentos ...any,
) pgx.Row {
	t.consulta = consulta
	t.argumentos = make([]any, len(argumentos))
	for indice, argumento := range argumentos {
		if contenido, esBinario := argumento.([]byte); esBinario {
			t.argumentos[indice] = append([]byte(nil), contenido...)
			continue
		}
		t.argumentos[indice] = argumento
	}
	return t.fila
}

func (t *transaccionConsultaConvocatoriaPrueba) Commit(context.Context) error {
	t.confirmaciones++
	return nil
}

func (t *transaccionConsultaConvocatoriaPrueba) Rollback(context.Context) error {
	t.reversiones++
	return nil
}

func TestConsultaGobiernoConvocatoriasPostgreSQLRecuperaYConfirmaEvidenciaExacta(t *testing.T) {
	version := versionConsultaConvocatoriaPostgreSQLPrueba(t)
	solicitud := solicitudConsultaConvocatoriaPostgreSQLPrueba(t, version)
	datos, err := solicitud.Autorizacion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	versionCanonica, err := version.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	huellaVersion, err := version.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	tx := &transaccionConsultaConvocatoriaPrueba{fila: filaPostgreSQLBaremacionPrueba{valores: []any{
		"obtenida", versionCanonica, huellaVersion, nil, "",
		datos.Decision.DecisionRef, datos.HuellaDecisionSHA256,
		"atestacion:pdp:consulta:001", strings.Repeat("3", 64),
		"consumo:autorizacion:consulta:001", "auditoria:consulta:001", strings.Repeat("4", 64),
		solicitud.ConsultadaEn,
	}}}
	repositorio, err := nuevaConsultaGobiernoConvocatoriasPostgreSQL(
		iniciadorPostgreSQLBaremacionPrueba{tx: tx},
	)
	if err != nil {
		t.Fatal(err)
	}

	resultado, err := repositorio.ObtenerVersionExacta(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("recuperar version exacta: %v", err)
	}
	if resultado.ValidarPara(solicitud) != nil || resultado.Version.Referencia() != version.Referencia() {
		t.Fatal("el adaptador devolvio una evidencia distinta de la solicitada")
	}
	if tx.confirmaciones != 1 || tx.reversiones != 1 ||
		!strings.Contains(tx.consulta, funcionObtenerVersionExactaConvocatoriaV1) || len(tx.argumentos) != 4 {
		t.Fatalf("frontera transaccional inesperada: commits=%d rollbacks=%d consulta=%q argumentos=%d",
			tx.confirmaciones, tx.reversiones, tx.consulta, len(tx.argumentos))
	}
	operacion, valida := tx.argumentos[0].([]byte)
	if !valida || !json.Valid(operacion) || bytesContienenDatoPersonalConsulta(operacion) {
		t.Fatal("la operacion durable no es JSON minimizado")
	}
}

func TestConsultaGobiernoConvocatoriasPostgreSQLRechazaFilaManipuladaAntesDeCommit(t *testing.T) {
	version := versionConsultaConvocatoriaPostgreSQLPrueba(t)
	solicitud := solicitudConsultaConvocatoriaPostgreSQLPrueba(t, version)
	datos, _ := solicitud.Autorizacion.Datos()
	versionCanonica, _ := version.RepresentacionCanonica()
	huellaVersion, _ := version.HuellaSHA256()
	versionManipulada := append(append([]byte(nil), versionCanonica...), ' ')
	tx := &transaccionConsultaConvocatoriaPrueba{fila: filaPostgreSQLBaremacionPrueba{valores: []any{
		"obtenida", versionManipulada, huellaVersion, nil, "",
		datos.Decision.DecisionRef, datos.HuellaDecisionSHA256,
		"atestacion:pdp:consulta:001", strings.Repeat("3", 64),
		"consumo:autorizacion:consulta:001", "auditoria:consulta:001", strings.Repeat("4", 64),
		solicitud.ConsultadaEn,
	}}}
	repositorio, err := nuevaConsultaGobiernoConvocatoriasPostgreSQL(
		iniciadorPostgreSQLBaremacionPrueba{tx: tx},
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = repositorio.ObtenerVersionExacta(context.Background(), solicitud)
	if !errors.Is(err, puertosbolsa.ErrEvidenciaConsultaConvocatoriaNoConfiable) ||
		tx.confirmaciones != 0 || tx.reversiones != 1 {
		t.Fatalf("fila maleable no se rechazo de forma cerrada: err=%v commits=%d rollbacks=%d",
			err, tx.confirmaciones, tx.reversiones)
	}
}

func TestConsultaGobiernoConvocatoriasPostgreSQLRechazaAutorizacionDevueltaDistinta(t *testing.T) {
	version := versionConsultaConvocatoriaPostgreSQLPrueba(t)
	solicitud := solicitudConsultaConvocatoriaPostgreSQLPrueba(t, version)
	versionCanonica, _ := version.RepresentacionCanonica()
	huellaVersion, _ := version.HuellaSHA256()
	tx := &transaccionConsultaConvocatoriaPrueba{fila: filaPostgreSQLBaremacionPrueba{valores: []any{
		"obtenida", versionCanonica, huellaVersion, nil, "",
		"autorizacion:otra:001", strings.Repeat("9", 64),
		"atestacion:pdp:consulta:001", strings.Repeat("3", 64),
		"consumo:autorizacion:consulta:001", "auditoria:consulta:001", strings.Repeat("4", 64),
		solicitud.ConsultadaEn,
	}}}
	repositorio, err := nuevaConsultaGobiernoConvocatoriasPostgreSQL(
		iniciadorPostgreSQLBaremacionPrueba{tx: tx},
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = repositorio.ObtenerVersionExacta(context.Background(), solicitud)
	if !errors.Is(err, puertosbolsa.ErrEvidenciaConsultaConvocatoriaNoConfiable) ||
		tx.confirmaciones != 0 {
		t.Fatalf("sustitucion de autorizacion aceptada: %v", err)
	}
}

func TestConsultaGobiernoConvocatoriasPostgreSQLFallaCerradoSinFuenteOSolicitud(t *testing.T) {
	if _, err := nuevaConsultaGobiernoConvocatoriasPostgreSQL(nil); !errors.Is(
		err, puertosbolsa.ErrFuenteGobiernoConvocatoriasNoDisponible,
	) {
		t.Fatalf("constructor sin fuente: %v", err)
	}
	tx := &transaccionConsultaConvocatoriaPrueba{}
	repositorio, err := nuevaConsultaGobiernoConvocatoriasPostgreSQL(
		iniciadorPostgreSQLBaremacionPrueba{tx: tx},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repositorio.ObtenerVersionExacta(context.Background(), puertosbolsa.SolicitudConsultaVersionConvocatoriaAutorizada{}); !errors.Is(
		err, puertosbolsa.ErrConsultaGobiernoConvocatoriaInvalida,
	) || tx.consulta != "" {
		t.Fatalf("solicitud invalida alcanzo PostgreSQL: %v", err)
	}
}

func versionConsultaConvocatoriaPostgreSQLPrueba(
	t *testing.T,
) dominiobolsa.VersionConvocatoriaGobernada {
	t.Helper()
	huella := func(marca string) string { return strings.Repeat(marca, 64) }
	contenido := dominiobolsa.ContenidoPublicableConvocatoria{
		IdentificadorPublico: "auxiliar-administrativo-2026", Tipo: "bolsa_temporal",
		CatalogoCategorias: dominiobolsa.ReferenciaCatalogoCategorias{
			CatalogoID: "categorias-profesionales", CatalogoVersion: 1,
			CatalogoHuellaSHA256: huella("a"),
		},
		Categorias: []string{"auxiliar_administrativo"}, Titulo: "Bolsa temporal de auxiliares",
		Resumen:     "Convocatoria publica para la bolsa temporal.",
		Descripcion: "Proceso selectivo sujeto a bases firmadas y publicadas.",
		Plazos: []dominiobolsa.PlazoConvocatoria{{
			Referencia: "plazo:inscripcion", Tipo: "inscripcion", Titulo: "Inscripcion",
			Descripcion: "Plazo de presentacion.", AbreEn: instantePostgreSQLPrueba.Add(time.Hour),
			CierraEn: instantePostgreSQLPrueba.Add(30 * 24 * time.Hour),
		}},
		Requisitos: []dominiobolsa.RequisitoConvocatoria{{
			Referencia: "requisito:edad", Orden: 1, Titulo: "Edad",
			Descripcion: "Cumplir la edad exigida.", Obligatorio: true,
		}},
		Documentos: []dominiobolsa.DocumentoPublicableConvocatoria{{
			Referencia: "documento:bases", Tipo: "bases", Orden: 1, Titulo: "Bases",
			Descripcion: "Bases de la convocatoria.", Formato: "pdf", URL: "/bolsa/documentos/bases.pdf",
		}},
	}
	referencia := func(id string, marca string) dominiobolsa.ReferenciaConfiguracionConvocatoria {
		return dominiobolsa.ReferenciaConfiguracionConvocatoria{
			ID: id, Version: 1, HuellaContenidoSHA256: huella(marca),
		}
	}
	configuracion := dominiobolsa.ConfiguracionFijadaConvocatoria{
		Catalogos: referencia("catalogos:bolsa", "1"), Calendario: referencia("calendario:auxiliar", "2"),
		ReglasBaremacion: referencia("baremo:auxiliar", "3"),
		FlujoProceso:     referencia("convocatoria-bolsa", "4"), FlujoSolicitud: referencia("solicitud-bolsa", "5"),
		Documentos: []dominiobolsa.ReferenciaDocumentoOficialConvocatoria{{
			Rol: "bases", PublicacionRef: "documento:bases", DocumentoRef: "documento:logico:bases:001",
			VersionDocumento: 1, RepresentacionRef: "representacion:pdf:bases:001",
			HuellaContenidoSHA256: huella("6"), FirmaValidadaRef: "firma:validada:bases:001",
			ReciboCustodiaRef: "custodia:bases:001",
		}},
	}
	version, err := dominiobolsa.NuevaVersionConvocatoriaGobernada(
		dominiobolsa.DatosNuevaVersionConvocatoriaGobernada{
			ID: "proceso:bolsa:auxiliar-2026", CodigoVersionPublica: "v1",
			InstanciaFlujoRef: "instancia:flujo:convocatoria:001", Contenido: contenido,
			Configuracion: configuracion, ExpedienteRef: "expediente:seleccion:2026-001",
			Motivo: "Preparacion administrativa.", ActorID: "persona:tecnica:001",
			Instante: instantePostgreSQLPrueba.Add(-time.Hour),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return version
}

func solicitudConsultaConvocatoriaPostgreSQLPrueba(
	t *testing.T,
	version dominiobolsa.VersionConvocatoriaGobernada,
) puertosbolsa.SolicitudConsultaVersionConvocatoriaAutorizada {
	t.Helper()
	selector := puertosbolsa.SelectorVersionConvocatoriaExacta{ID: version.ID, Secuencia: version.Secuencia}
	recurso, err := puertosbolsa.RecursoAutorizableConsultaVersionConvocatoria(selector)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := pruebasvec.NuevoVinculoGenerico(instantePostgreSQLPrueba)
	if err != nil {
		t.Fatal(err)
	}
	datosVinculo, err := vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaCatalogo, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	decision := dominiovec.DecisionAutorizacion{
		DecisionRef: "autorizacion:consulta:convocatoria:001", Concedida: true, Codigo: "concedida",
		PrincipalID: datosVinculo.PrincipalID, PerfilActivoRef: datosVinculo.PerfilActivoRef,
		Accion: puertosbolsa.AccionConsultarVersionConvocatoria, RecursoRef: recurso.Referencia,
		ModuloID:                    puertosbolsa.ModuloGobiernoConvocatorias,
		TipoRecurso:                 puertosbolsa.TipoRecursoVersionConvocatoriaGobernada,
		ContextoRecursoHuellaSHA256: huellaRecurso,
		Finalidad:                   puertosbolsa.FinalidadConsultaInternaConvocatorias,
		CorrelacionRef:              "correlacion:convocatoria:001", VinculoAutenticacionActor: vinculo,
		AsignacionRef: "asignacion:rrhh:v1", AsignacionHuellaSHA256: strings.Repeat("7", 64),
		VersionRolRef: "rol:rrhh:v1", VersionRolHuellaSHA256: strings.Repeat("8", 64),
		ControlVigenciaVersionRolRef: "rol:rrhh:v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("9", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasHuellasSHA256: map[string]string{},
		GarantiaMinima:                  dominiovec.AuthAssuranceHigh, CamposPermitidos: []string{"version_convocatoria"},
		EmitidaEn:   instantePostgreSQLPrueba.Add(-time.Minute),
		ValidaHasta: instantePostgreSQLPrueba.Add(4 * time.Minute),
	}
	evidencia, err := puertosvec.NuevaEvidenciaUsoDecisionAutorizacion(decision, instantePostgreSQLPrueba)
	if err != nil {
		t.Fatal(err)
	}
	return puertosbolsa.SolicitudConsultaVersionConvocatoriaAutorizada{
		Selector: selector, Autorizacion: evidencia, ConsultadaEn: instantePostgreSQLPrueba,
	}
}

// Esta comprobacion evita que una futura ampliacion de la operacion de
// persistencia introduzca por accidente nombre, DNI, correo o telefono.
func bytesContienenDatoPersonalConsulta(contenido []byte) bool {
	texto := strings.ToLower(string(contenido))
	for _, termino := range []string{"dni", "nombre", "apellido", "correo", "telefono"} {
		if strings.Contains(texto, termino) {
			return true
		}
	}
	return false
}
