package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type proveedorConsultaJustificantePrueba func(context.Context, ports.SolicitudResolverLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)

func (p proveedorConsultaJustificantePrueba) AutorizarConsultaJustificanteRespuestaRecibida(ctx context.Context, s ports.SolicitudResolverLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return p(ctx, s)
}

func justificanteConsultaPGPrueba(t *testing.T) (ports.SolicitudResolverLlamamiento, ports.JustificanteRespuestaRecibida) {
	t.Helper()
	_, seleccion, _ := materialesEjecucionSeleccionO6Prueba(t)
	seleccion.VersionExpediente = 6
	s := ports.SolicitudResolverLlamamiento{
		ClaveIdempotencia: "11111111-1111-4111-8111-111111111111", OrganizacionRef: seleccion.OrganizacionRef,
		ExpedienteRef: seleccion.ExpedienteRef, LlamamientoRef: seleccion.LlamamientoRef,
		ComunicacionRef: "comunicacion:sintetica", VersionEsperada: 2,
		Respuesta: ports.RespuestaLlamamientoAceptada, PruebaRespuestaRef: "justificante:sintetico",
	}
	j := ports.JustificanteRespuestaRecibida{Seleccion: seleccion, Respuesta: ports.RespuestaRecibidaRegistrada{
		Solicitud: ports.SolicitudRegistrarRespuestaRecibida{
			ClaveIdempotencia: "21111111-1111-4111-8111-111111111111", OrganizacionRef: s.OrganizacionRef,
			ExpedienteRef: s.ExpedienteRef, LlamamientoRef: s.LlamamientoRef, ComunicacionRef: s.ComunicacionRef,
			VersionComunicacionEsperada: 2, Respuesta: s.Respuesta, CorreoRef: "correo:sintetico", CorreoSHA256: strings.Repeat("a", 64),
			RecibidaEn: seleccion.ConfirmadaEn.Add(time.Hour),
		},
		JustificanteRef: s.PruebaRespuestaRef, ReciboRef: "recibo:respuesta", AuditoriaRef: "auditoria:respuesta",
		RegistradaEn: seleccion.ConfirmadaEn.Add(2 * time.Hour), Estado: ports.EstadoRespuestaRecibidaRegistrada,
	}}
	if err := j.ValidarPara(s); err != nil {
		t.Fatal(err)
	}
	return s, j
}

// Transporte deliberadamente no criptográfico, exclusivo de dobles de pgx.
// El consumidor SQL debe rechazarlo: no se exporta para dinámicas ni bootstrap.
func materialConsultaJustificantePrueba(t *testing.T, s ports.SolicitudResolverLlamamiento, accion, audiencia string, numero int) puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	recurso, err := RecursoConsultaJustificanteRespuestaRecibida(s)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	h := strings.Repeat("a", 64)
	ahora := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:unidad:"+strconv.Itoa(numero), h, h, "contexto:unidad", h,
		accion, s.ExpedienteRef, huella, audiencia, ahora, ahora.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)).Public())
	if err != nil {
		t.Fatal(err)
	}
	a, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("x", 512)), resumen,
		[]byte("{}"), []byte("{}"), []byte("{}"), 1, 1, []byte("unidad"), []byte("unidad"), []byte("unidad"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestConsultaJustificanteRespuestaRecibidaPGMaterialOchoCampos(t *testing.T) {
	s, _ := justificanteConsultaPGPrueba(t)
	r, err := RecursoConsultaJustificanteRespuestaRecibida(s)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(s)
	h := sha256.Sum256(b)
	if r.Referencia != s.ExpedienteRef || r.ModuloID != "contratacion_temporal" || r.Tipo != TipoRecursoConsultaJustificanteRespuestaRecibida ||
		len(r.Ambitos) != 1 || r.Ambitos["organizacion_ref"] != s.OrganizacionRef ||
		len(r.Atributos) != 1 || r.Atributos["material_sha256"] != hex.EncodeToString(h[:]) {
		t.Fatal("recurso no ligado a la solicitud directa")
	}
	tipo := reflect.TypeOf(s)
	if tipo.NumField() != 8 {
		t.Fatal("material debe conservar ocho campos")
	}
	for i := 0; i < tipo.NumField(); i++ {
		if tipo.Field(i).Tag != "" {
			t.Fatal("material con etiquetas")
		}
		otra := s
		campo := reflect.ValueOf(&otra).Elem().Field(i)
		if campo.Kind() == reflect.Uint64 {
			campo.SetUint(3)
			if _, err := RecursoConsultaJustificanteRespuestaRecibida(otra); !errors.Is(err, ports.ErrVersionRespuestaRecibidaEnConflicto) {
				t.Fatal(err)
			}
			continue
		}
		switch tipo.Field(i).Name {
		case "ClaveIdempotencia":
			campo.SetString("31111111-1111-4111-8111-111111111111")
		case "Respuesta":
			campo.SetString(string(ports.RespuestaLlamamientoRenunciada))
		default:
			campo.SetString(campo.String() + "otra")
		}
		nuevo, err := RecursoConsultaJustificanteRespuestaRecibida(otra)
		if err != nil || nuevo.Atributos["material_sha256"] == r.Atributos["material_sha256"] {
			t.Fatal("campo no ligado", tipo.Field(i).Name, err)
		}
	}
}

func TestConsultaJustificanteRespuestaRecibidaPGRecuperaConPermisoNuevo(t *testing.T) {
	s, original := justificanteConsultaPGPrueba(t)
	b, _ := json.Marshal(original)
	// +00:00 puede decodificarse con time.Local: conservar instantes y precisión.
	contenido := strings.ReplaceAll(string(b), "Z\"", "+00:00\"")
	tx := &transaccionEjecucionSeleccionO6Prueba{fila: filaEjecucionSeleccionO6Prueba{valores: []any{contenido}}}
	pool := &iniciadorEjecucionSeleccionO6Prueba{tx: tx}
	autorizaciones := 0
	proveedor := proveedorConsultaJustificantePrueba(func(_ context.Context, recibida ports.SolicitudResolverLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		if recibida != s {
			t.Fatal("proveedor recibió otro material")
		}
		autorizaciones++
		return materialConsultaJustificantePrueba(t, recibida, AccionConsultaJustificanteRespuestaRecibida, AudienciaRegistroComunicacionLlamamiento, autorizaciones), nil
	})
	l := &LectorJustificantesRespuestaRecibidaPostgreSQL{pool: pool, proveedor: proveedor}
	for intento := 0; intento < 2; intento++ {
		j, err := l.ConsultarJustificanteRespuestaRecibida(context.Background(), s)
		if err != nil || j != original {
			t.Fatal("recuperación divergente", err)
		}
	}
	if autorizaciones != 2 || pool.inicios != 2 || tx.confirmaciones != 2 ||
		pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite || len(tx.consultas) != 2 {
		t.Fatal("consulta sin consumo nuevo o transacción incorrecta")
	}
	esperado, _ := json.Marshal(s)
	for i, sql := range tx.consultas {
		if !strings.Contains(sql, "vec_contratacion_temporal.consultar_justificante_respuesta_recibida_rrhh_v1(") ||
			len(tx.argumentos[i]) != 11 || tx.argumentos[i][0] != string(esperado) || tx.argumentos[i][5] != int64(1) || tx.argumentos[i][6] != int64(1) {
			t.Fatal("fachada o argumentos divergentes")
		}
	}
}

func TestConsultaJustificanteRespuestaRecibidaPGFallaCerradoSinFiltrarNiReintentar(t *testing.T) {
	s, original := justificanteConsultaPGPrueba(t)
	b, _ := json.Marshal(original)
	for _, caso := range []string{"sin_material", "permiso_registro", "otra_audiencia", "otro_material", "denegada", "salida", "submicro", "sql", "commit"} {
		t.Run(caso, func(t *testing.T) {
			tx := &transaccionEjecucionSeleccionO6Prueba{fila: filaEjecucionSeleccionO6Prueba{valores: []any{string(b)}}}
			pool := &iniciadorEjecucionSeleccionO6Prueba{tx: tx}
			proveedor := proveedorConsultaJustificantePrueba(func(_ context.Context, recibida ports.SolicitudResolverLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
				accion, audiencia := AccionConsultaJustificanteRespuestaRecibida, AudienciaRegistroComunicacionLlamamiento
				switch caso {
				case "sin_material":
					return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
				case "denegada":
					return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrOperacionRespuestaRecibidaDenegada
				case "permiso_registro":
					accion = AccionRegistroRespuestaRecibida
				case "otra_audiencia":
					audiencia = "vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1"
				case "otro_material":
					recibida.PruebaRespuestaRef += "otro"
				}
				return materialConsultaJustificantePrueba(t, recibida, accion, audiencia, 1), nil
			})
			esperado := ports.ErrOperacionRespuestaRecibidaDenegada
			switch caso {
			case "salida", "submicro":
				m := original
				if caso == "salida" {
					m.Respuesta.JustificanteRef += "otro"
				} else {
					m.Seleccion.ConfirmadaEn = m.Seleccion.ConfirmadaEn.Add(time.Nanosecond)
				}
				contenido, _ := json.Marshal(m)
				tx.fila = filaEjecucionSeleccionO6Prueba{valores: []any{string(contenido)}}
				esperado = ports.ErrResultadoRespuestaRecibidaNoConfiable
			case "sql":
				tx.fila = filaEjecucionSeleccionO6Prueba{err: &pgconn.PgError{Code: "40001", Message: "dato privado"}}
				esperado = ports.ErrRespuestaRecibidaNoDisponible
			case "commit":
				tx.errCommit = errors.New("dato privado")
				esperado = ports.ErrRespuestaRecibidaNoDisponible
			}
			l := &LectorJustificantesRespuestaRecibidaPostgreSQL{pool: pool, proveedor: proveedor}
			j, err := l.ConsultarJustificanteRespuestaRecibida(context.Background(), s)
			if !errors.Is(err, esperado) || j != (ports.JustificanteRespuestaRecibida{}) || strings.Contains(err.Error(), "privado") || pool.inicios > 1 {
				t.Fatal("fallo expuso datos o repitió la consulta", err)
			}
			if esperado == ports.ErrOperacionRespuestaRecibidaDenegada && pool.inicios != 0 {
				t.Fatal("permiso incorrecto alcanzó SQL")
			}
			if pool.inicios == 1 && tx.reversiones != 1 {
				t.Fatal("sin cierre transaccional")
			}
			if caso != "commit" && tx.confirmaciones != 0 {
				t.Fatal("se confirmó un resultado rechazado")
			}
		})
	}
	var proveedor proveedorConsultaJustificantePrueba
	if l, err := NuevoLectorJustificantesRespuestaRecibidaPostgreSQL(&pgxpool.Pool{}, proveedor); l != nil || !errors.Is(err, ports.ErrRespuestaRecibidaNoDisponible) {
		t.Fatal("proveedor nulo admitido")
	}
	pool := &iniciadorEjecucionSeleccionO6Prueba{}
	proveedor = func(context.Context, ports.SolicitudResolverLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		t.Fatal("cancelación consultó proveedor")
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
	}
	l := &LectorJustificantesRespuestaRecibidaPostgreSQL{pool: pool, proveedor: proveedor}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := l.ConsultarJustificanteRespuestaRecibida(ctx, s); !errors.Is(err, context.Canceled) || pool.inicios != 0 {
		t.Fatal(err)
	}
}

func TestConsultaJustificanteRespuestaRecibidaPGErroresNominales(t *testing.T) {
	for codigo, esperado := range map[string]error{
		"P0570": ports.ErrSolicitudRespuestaRecibidaInvalida,
		"P0572": ports.ErrVersionRespuestaRecibidaEnConflicto,
		"P0573": ports.ErrOperacionRespuestaRecibidaDenegada,
		"P0574": ports.ErrRespuestaRecibidaNoDisponible,
		"42501": ports.ErrOperacionRespuestaRecibidaDenegada,
		"08006": ports.ErrRespuestaRecibidaNoDisponible,
	} {
		err := normalizarErrorConsultaJustificanteRespuestaRecibida(context.Background(), &pgconn.PgError{Code: codigo, Message: "dato privado"})
		if !errors.Is(err, esperado) || strings.Contains(err.Error(), "privado") {
			t.Fatal(codigo, err)
		}
	}
}
