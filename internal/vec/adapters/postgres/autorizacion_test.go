package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type iniciadorPostgreSQLNulo struct{}

func (*iniciadorPostgreSQLNulo) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return nil, errors.New("no debe ejecutarse")
}

type iniciadorPostgreSQLConError struct{ err error }

func (i iniciadorPostgreSQLConError) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return nil, i.err
}

func TestNuevoAlmacenAutorizacionRechazaDependenciaNulaYTipada(t *testing.T) {
	t.Parallel()
	if _, err := nuevoAlmacenAutorizacion(nil); !errors.Is(err, domain.ErrConfiguracionAccesoInvalida) {
		t.Fatalf("nulo: %v", err)
	}
	var tipado *iniciadorPostgreSQLNulo
	if _, err := nuevoAlmacenAutorizacion(tipado); !errors.Is(err, domain.ErrConfiguracionAccesoInvalida) {
		t.Fatalf("nulo tipado: %v", err)
	}
}

func TestFuentePostgreSQLNoFiltraErrorNiCadenaDeConexion(t *testing.T) {
	t.Parallel()
	almacen, err := nuevoAlmacenAutorizacion(iniciadorPostgreSQLConError{
		err: errors.New("postgresql://usuario:secreto@servidor/vec SELECT dato_privado"),
	})
	if err != nil {
		t.Fatalf("crear almacen: %v", err)
	}
	_, err = almacen.ObtenerInstantaneaAutorizacion(context.Background(), "persona:1", "perfil:1")
	if !errors.Is(err, ports.ErrFuenteAutorizacionNoDisponible) {
		t.Fatalf("error cerrado esperado: %v", err)
	}
	if strings.Contains(err.Error(), "secreto") || strings.Contains(err.Error(), "SELECT") {
		t.Fatalf("el error filtra infraestructura: %q", err)
	}
}

func TestParsearRevisionAutorizacionCubreUint64SinNormalizar(t *testing.T) {
	t.Parallel()
	revision, err := parsearRevisionAutorizacion("18446744073709551615")
	if err != nil || revision != math.MaxUint64 {
		t.Fatalf("maximo uint64: revision=%d err=%v", revision, err)
	}
	for _, invalida := range []string{"", "0", " 1", "+1", "01x", "18446744073709551616"} {
		if _, err := parsearRevisionAutorizacion(invalida); err == nil {
			t.Fatalf("revision invalida aceptada: %q", invalida)
		}
	}
}

func TestDecodificarDocumentoPostgreSQLEsEstricto(t *testing.T) {
	t.Parallel()
	tipo := struct {
		Valor string `json:"valor"`
	}{}
	if err := decodificarDocumentoPostgreSQL([]byte(`{"valor":"uno"}`), &tipo); err != nil || tipo.Valor != "uno" {
		t.Fatalf("documento valido: %#v, %v", tipo, err)
	}
	for _, contenido := range []string{
		`{"valor":"uno","desconocido":true}`,
		`{"valor":"uno"} {"valor":"dos"}`,
		``,
	} {
		if err := decodificarDocumentoPostgreSQL([]byte(contenido), &tipo); err == nil {
			t.Fatalf("documento no estricto aceptado: %q", contenido)
		}
	}
	var destinoTipado *struct{ Valor string }
	if err := decodificarDocumentoPostgreSQL([]byte(`{"Valor":"uno"}`), destinoTipado); err == nil {
		t.Fatal("destino nulo tipado aceptado")
	}
}

func TestErroresPostgreSQLSeTraducenSinDetalles(t *testing.T) {
	t.Parallel()
	casos := []struct {
		codigo   string
		esperado error
	}{
		{"23505", ports.ErrVersionAutorizacionYaExiste},
		{"40001", ports.ErrInstantaneaAutorizacionObsoleta},
		{"40P01", ports.ErrInstantaneaAutorizacionObsoleta},
		{"55P03", ports.ErrInstantaneaAutorizacionObsoleta},
		{"XX000", ports.ErrRegistroDecisionNoDisponible},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.codigo, func(t *testing.T) {
			t.Parallel()
			err := errorRegistroAutorizacion(context.Background(), &pgconn.PgError{
				Code: caso.codigo, Message: "dsn=postgresql://usuario:secreto@servidor dato_privado",
			})
			if !errors.Is(err, caso.esperado) {
				t.Fatalf("traduccion: %v", err)
			}
			if strings.Contains(err.Error(), "secreto") || strings.Contains(err.Error(), "dato_privado") {
				t.Fatalf("detalle filtrado: %q", err)
			}
		})
	}
}

func TestIdentificadorPostgreSQLSeguroNoCorrigeEntradas(t *testing.T) {
	t.Parallel()
	if !identificadorPostgreSQLSeguro("perfil:persona-1:bolsa", 512) {
		t.Fatal("referencia canonica rechazada")
	}
	for _, invalido := range []string{"", " perfil", "perfil ", "perfil\notro", "perfil otro", "perfil\x00otro"} {
		if identificadorPostgreSQLSeguro(invalido, 512) {
			t.Fatalf("identificador inseguro aceptado: %q", invalido)
		}
	}
}

func TestSerializarDecisionPostgreSQLMaterializaCatalogosVacios(t *testing.T) {
	t.Parallel()
	decision := decisionAutorizacionPostgreSQLPrueba(t)
	documento, err := serializarDecisionPostgreSQL(decision)
	if err != nil {
		t.Fatalf("serializar: %v", err)
	}
	var objeto map[string]any
	if err = json.Unmarshal(documento, &objeto); err != nil {
		t.Fatalf("decodificar: %v", err)
	}
	for _, clave := range []string{
		"politicas_evaluadas_refs", "politicas_evaluadas_huellas_sha256",
		"politicas_refs", "politicas_huellas_sha256",
		"campos_permitidos", "obligaciones",
	} {
		if _, existe := objeto[clave]; !existe {
			t.Fatalf("falta el catalogo vacio %q", clave)
		}
	}
	if _, existe := objeto["vinculo_autenticacion_actor"]; !existe {
		t.Fatal("falta el vinculo opaco de autenticacion y actor")
	}
	var reconstruida domain.DecisionAutorizacion
	if err = json.Unmarshal(documento, &reconstruida); !errors.Is(err, domain.ErrReconstruccionVinculoAutenticacionActorProhibida) {
		t.Fatalf("el adaptador reconstruyo el vinculo sin rehidratador autoritativo: %v", err)
	}
}

func TestSerializarDecisionPostgreSQLDerivaElPerfilCanonicoDeMicrosegundoFijo(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre       string
		microsegundo int
	}{
		{"cero_ceros_finales", 123456},
		{"un_cero_final", 123450},
		{"dos_ceros_finales", 123400},
		{"tres_ceros_finales", 123000},
		{"cuatro_ceros_finales", 120000},
		{"cinco_ceros_finales", 100000},
		{"seis_ceros_finales", 0},
		{"limite_inferior_no_cero", 1},
		{"limite_superior", 999999},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			ahora := time.Date(
				2026, time.July, 15, 10, 30, 0, caso.microsegundo*1_000, time.UTC,
			)
			decision := decisionAutorizacionPostgreSQLPruebaEn(t, ahora)
			// Registrar decisiones y materializar su canon no implica que estas
			// obligaciones ya sean consumibles por un PEP concreto.
			decision.Obligaciones = []string{"doble_control", "trazar_acceso"}
			if err := decision.ValidarEvidenciaInstantanea(); err != nil {
				t.Fatalf("decision con obligaciones valida: %v", err)
			}

			canonica, err := ports.RepresentacionCanonicaDecisionAutorizacionReforzadaV1(decision)
			if err != nil {
				t.Fatalf("representacion canonica: %v", err)
			}
			documento, err := serializarDecisionPostgreSQL(decision)
			if err != nil {
				t.Fatalf("serializar documento PostgreSQL: %v", err)
			}
			var objetoCanonico, objetoDocumento map[string]json.RawMessage
			if json.Unmarshal(canonica, &objetoCanonico) != nil ||
				json.Unmarshal(documento, &objetoDocumento) != nil {
				t.Fatal("el serializador produjo JSON no decodificable")
			}
			if len(objetoCanonico) != 30 || len(objetoDocumento) != 31 {
				t.Fatalf("numero de claves inesperado: canon=%d documento=%d", len(objetoCanonico), len(objetoDocumento))
			}
			for clave, valorCanonico := range objetoCanonico {
				switch clave {
				case "esquema", "politicas_evaluadas", "politicas_aplicables":
					continue
				}
				if !bytes.Equal(valorCanonico, objetoDocumento[clave]) {
					t.Fatalf("el campo %q diverge del canon: canon=%s documento=%s", clave, valorCanonico, objetoDocumento[clave])
				}
			}
			comprobarManifiestoCanonicoPostgreSQLPrueba(
				t, objetoCanonico["politicas_evaluadas"],
				objetoDocumento["politicas_evaluadas_refs"],
				objetoDocumento["politicas_evaluadas_huellas_sha256"],
			)
			comprobarManifiestoCanonicoPostgreSQLPrueba(
				t, objetoCanonico["politicas_aplicables"],
				objetoDocumento["politicas_refs"],
				objetoDocumento["politicas_huellas_sha256"],
			)

			const formato = "2006-01-02T15:04:05.000000Z"
			comprobarInstanteCanonicoPostgreSQLPrueba(
				t, objetoDocumento["emitida_en"], decision.EmitidaEn.Format(formato),
			)
			comprobarInstanteCanonicoPostgreSQLPrueba(
				t, objetoDocumento["valida_hasta"], decision.ValidaHasta.Format(formato),
			)
			var vinculo map[string]json.RawMessage
			if err := json.Unmarshal(objetoDocumento["vinculo_autenticacion_actor"], &vinculo); err != nil {
				t.Fatalf("vinculo canonico: %v", err)
			}
			datosVinculo, err := decision.VinculoAutenticacionActor.Datos()
			if err != nil {
				t.Fatalf("datos del vinculo: %v", err)
			}
			instantesVinculo := map[string]time.Time{
				"autenticacion_verificada_en": datosVinculo.AutenticacionVerificadaEn,
				"sesion_emitida_en":           datosVinculo.SesionEmitidaEn,
				"sesion_valida_hasta":         datosVinculo.SesionValidaHasta,
				"sesion_revalidada_en":        datosVinculo.SesionRevalidadaEn,
			}
			for clave, instante := range instantesVinculo {
				comprobarInstanteCanonicoPostgreSQLPrueba(
					t, vinculo[clave], instante.Format(formato),
				)
			}
		})
	}
}

func TestSerializarDecisionPostgreSQLRechazaPrecisionSubmicrosegundo(t *testing.T) {
	t.Parallel()
	decision := decisionAutorizacionPostgreSQLPrueba(t)
	decision.EmitidaEn = decision.EmitidaEn.Add(time.Nanosecond)
	if _, err := serializarDecisionPostgreSQL(decision); !errors.Is(err, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida) {
		t.Fatalf("instante submicrosegundo no rechazado de forma cerrada: %v", err)
	}
}

func comprobarInstanteCanonicoPostgreSQLPrueba(
	t *testing.T,
	contenido json.RawMessage,
	esperado string,
) {
	t.Helper()
	var recibido string
	if err := json.Unmarshal(contenido, &recibido); err != nil || recibido != esperado {
		t.Fatalf("instante no canonico: recibido=%q esperado=%q err=%v", recibido, esperado, err)
	}
}

func comprobarManifiestoCanonicoPostgreSQLPrueba(
	t *testing.T,
	canonico, referenciasJSON, huellasJSON json.RawMessage,
) {
	t.Helper()
	var entradas []entradaManifiestoDecisionPostgreSQL
	var referencias []string
	var huellas map[string]string
	if json.Unmarshal(canonico, &entradas) != nil ||
		json.Unmarshal(referenciasJSON, &referencias) != nil ||
		json.Unmarshal(huellasJSON, &huellas) != nil {
		t.Fatal("manifiesto PostgreSQL no decodificable")
	}
	referenciasEsperadas := make([]string, 0, len(entradas))
	huellasEsperadas := make(map[string]string, len(entradas))
	for _, entrada := range entradas {
		referenciasEsperadas = append(referenciasEsperadas, entrada.Referencia)
		huellasEsperadas[entrada.Referencia] = entrada.HuellaSHA256
	}
	if !reflect.DeepEqual(referencias, referenciasEsperadas) || !reflect.DeepEqual(huellas, huellasEsperadas) {
		t.Fatalf("manifiesto divergente: refs=%v huellas=%v", referencias, huellas)
	}
}

type revalidadorAutenticacionActorPostgreSQLPrueba struct {
	solicitud     domain.SolicitudRevalidacionAutenticacionActorV1
	autenticacion domain.AutenticacionRevalidadaV1
}

func (r revalidadorAutenticacionActorPostgreSQLPrueba) RevalidarAutenticacionActorV1(
	ctx context.Context,
	solicitud domain.SolicitudRevalidacionAutenticacionActorV1,
) (domain.AutenticacionRevalidadaV1, error) {
	if ctx == nil {
		return domain.AutenticacionRevalidadaV1{}, domain.ErrAutenticacionRevalidadaInvalida
	}
	if err := ctx.Err(); err != nil {
		return domain.AutenticacionRevalidadaV1{}, err
	}
	if solicitud != r.solicitud {
		return domain.AutenticacionRevalidadaV1{}, domain.ErrAutenticacionRevalidadaInvalida
	}
	return r.autenticacion, nil
}

func decisionAutorizacionPostgreSQLPrueba(t *testing.T) domain.DecisionAutorizacion {
	t.Helper()
	return decisionAutorizacionPostgreSQLPruebaEn(
		t,
		time.Date(2026, time.July, 15, 10, 30, 0, 123_456_000, time.UTC),
	)
}

func decisionAutorizacionPostgreSQLPruebaEn(
	t *testing.T,
	ahora time.Time,
) domain.DecisionAutorizacion {
	t.Helper()
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_cuenta_postgresql_autorizacion_0001",
		Metodo:    domain.AuthMethodCertificate,
		Garantia:  domain.AuthAssuranceHigh,
	}
	instantanea := domain.InstantaneaContextoActor{
		VinculoRef:      "vca_contexto_postgresql_autorizacion_0001",
		VinculoVersion:  1,
		CuentaRef:       cuenta.CuentaRef,
		PersonaRef:      "per_persona_postgresql_autorizacion_0001",
		PersonaVersion:  1,
		PerfilActivoRef: "prf_perfil_postgresql_autorizacion_0001",
		PerfilVersion:   1,
		Estado:          domain.EstadoVinculoContextoActorActivo,
		VigenteDesde:    ahora.Add(-24 * time.Hour),
		VigenteHasta:    ahora.Add(24 * time.Hour),
	}
	actor, err := domain.NuevoContextoActor(cuenta, instantanea, ahora.Add(-time.Minute))
	if err != nil {
		t.Fatalf("actor de prueba: %v", err)
	}
	solicitudRevalidacion := domain.SolicitudRevalidacionAutenticacionActorV1{
		AutenticacionRef: "aut_autenticacion_postgresql_prueba_0001",
		SesionRef:        "ses_sesion_postgresql_autorizacion_0001",
	}
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef:             solicitudRevalidacion.AutenticacionRef,
		AutenticacionHuellaSHA256:    strings.Repeat("a", 64),
		AsercionRef:                  "ase_asercion_postgresql_autorizacion_0001",
		SesionRef:                    solicitudRevalidacion.SesionRef,
		ControlSesionRef:             "cse_control_postgresql_autorizacion_0001",
		ControlSesionRevision:        1,
		ControlSesionHuellaSHA256:    strings.Repeat("b", 64),
		CuentaRef:                    cuenta.CuentaRef,
		CuentaOrdinariaRef:           cuenta.CuentaRef,
		Superficie:                   domain.SuperficieAutenticacionExternaPersonalV1,
		MetodoObservado:              cuenta.Metodo,
		GarantiaObservada:            cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_politica_garantia_postgresql_0001",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("c", 64),
		AutenticacionVerificadaEn:    ahora.Add(-2 * time.Hour),
		SesionEmitidaEn:              ahora.Add(-90 * time.Minute),
		SesionRevalidadaEn:           ahora.Add(-2 * time.Minute),
		SesionValidaHasta:            ahora.Add(time.Hour),
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV1(
		context.Background(),
		revalidadorAutenticacionActorPostgreSQLPrueba{
			solicitud: solicitudRevalidacion, autenticacion: autenticacion,
		},
		solicitudRevalidacion,
		actor,
		ahora,
	)
	if err != nil {
		t.Fatalf("vinculo de prueba: %v", err)
	}
	recurso := domain.RecursoAutorizable{
		Referencia: "expediente:postgresql:0001",
		ModuloID:   "bolsa",
		Tipo:       "expediente",
		Ambitos:    map[string]string{"unidad": "seleccion"},
		Atributos:  map[string]string{},
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatalf("huella de recurso: %v", err)
	}
	huellaCatalogo, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(nil, nil)
	if err != nil {
		t.Fatalf("huella de catalogo vacio: %v", err)
	}
	decision := domain.DecisionAutorizacion{
		DecisionRef: "decision:postgresql:autorizacion:0001", Concedida: true, Codigo: "concedida",
		PrincipalID: actor.Principal.ID, PerfilActivoRef: actor.PerfilActivoRef,
		Accion: "bolsa.expediente.leer", RecursoRef: recurso.Referencia,
		ModuloID: recurso.ModuloID, TipoRecurso: recurso.Tipo, ContextoRecursoHuellaSHA256: huellaContexto,
		Finalidad: "gestion_bolsa", CorrelacionRef: "correlacion:postgresql:autorizacion:0001",
		VinculoAutenticacionActor: vinculo,
		AsignacionRef:             "asignacion:postgresql:autorizacion:0001", AsignacionHuellaSHA256: strings.Repeat("d", 64),
		VersionRolRef: "rol:postgresql:autorizacion:v1", VersionRolHuellaSHA256: strings.Repeat("e", 64),
		ControlVigenciaVersionRolRef: "rol:postgresql:autorizacion:v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("f", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		GarantiaMinima: domain.AuthAssuranceSubstantial,
		EmitidaEn:      ahora, ValidaHasta: ahora.Add(90 * time.Second),
	}
	if err := decision.ValidarEvidenciaInstantanea(); err != nil {
		t.Fatalf("decision de prueba: %v", err)
	}
	return decision
}
