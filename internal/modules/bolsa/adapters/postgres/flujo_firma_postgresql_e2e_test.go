package postgres

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const (
	variableDSNFlujoFirmaPostgreSQLE2E      = "VEC_BOLSA_FIRMA_POSTGRES_E2E_DSN"
	variableDSNAdminFlujoFirmaPostgreSQLE2E = "VEC_BOLSA_FIRMA_POSTGRES_E2E_ADMIN_DSN"
	variableFaseFlujoFirmaPostgreSQLE2E     = "VEC_BOLSA_FIRMA_POSTGRES_E2E_FASE"
)

var claveEstadoFlujoFirmaPostgreSQLE2E = sha256.Sum256([]byte(
	"clave de prueba aislada para sellar la saga E2E de firma",
))

type verificadorFlujoFirmaPostgreSQLE2E struct{}

func (verificadorFlujoFirmaPostgreSQLE2E) VerificarEstadoFlujoFirmaBaremacion(
	_ context.Context,
	solicitud puertosbolsa.SolicitudVerificarEstadoFlujoFirmaBaremacion,
) error {
	if solicitud.Validar() != nil {
		return puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	contenido := solicitud.RepresentacionCanonica.Revelar()
	defer borrarBytesPostgreSQL(contenido)
	mac := hmac.New(sha256.New, claveEstadoFlujoFirmaPostgreSQLE2E[:])
	_, _ = mac.Write(contenido)
	esperado := "hmac-sha256:flujo_firma_postgresql_v1:" +
		hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(esperado), []byte(solicitud.SelloHMAC)) {
		return puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	return nil
}

func TestFlujoFirmaPostgreSQLE2E(t *testing.T) {
	dsn := os.Getenv(variableDSNFlujoFirmaPostgreSQLE2E)
	fase := os.Getenv(variableFaseFlujoFirmaPostgreSQLE2E)
	if dsn == "" || fase == "" {
		t.Skip("E2E PostgreSQL de firma no solicitado")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repositorio, err := NuevoRepositorioFlujosFirmaBaremacionPostgreSQL(
		pool,
		verificadorFlujoFirmaPostgreSQLE2E{},
		[]byte("0123456789abcdef0123456789abcdef"),
	)
	if err != nil {
		t.Fatal(err)
	}
	expediente := expedienteFlujoFirmaPostgreSQLE2E(t)
	consulta := puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion{
		FlujoRef:               expediente.FlujoRef,
		IndiceIdempotenciaHMAC: expediente.IndiceIdempotenciaHMAC,
		VinculoActorHMAC:       expediente.VinculoActorHMAC,
	}
	switch fase {
	case "crear":
		creado, err := repositorio.CrearORecuperarFlujoFirmaBaremacion(
			ctx,
			puertosbolsa.SolicitudCrearORecuperarFlujoFirmaBaremacion{
				Expediente: expediente,
			},
		)
		if err != nil || !creado.Creado {
			t.Fatalf("crear saga: resultado=%+v error=%v", creado, err)
		}
		recuperado, err := repositorio.CrearORecuperarFlujoFirmaBaremacion(
			ctx,
			puertosbolsa.SolicitudCrearORecuperarFlujoFirmaBaremacion{
				Expediente: expediente,
			},
		)
		if err != nil || recuperado.Creado {
			t.Fatalf("reconciliar creación: resultado=%+v error=%v", recuperado, err)
		}
	case "continuar":
		actual, err := repositorio.ObtenerFlujoFirmaBaremacion(ctx, consulta)
		if err != nil || actual.Version != 1 {
			t.Fatalf(
				"recuperar tras reinicio: resultado=%+v error=%v diagnostico=%v",
				actual,
				err,
				diagnosticarLecturaFlujoFirmaPostgreSQLE2E(
					ctx,
					pool,
					consulta,
				),
			)
		}
		primero := adquirirFlujoFirmaPostgreSQLE2E(
			t,
			ctx,
			repositorio,
			consulta,
			1,
			"worker-e2e-primero",
		)
		if _, err := repositorio.AdquirirArrendamientoFlujoFirmaBaremacion(
			ctx,
			puertosbolsa.SolicitudAdquirirArrendamientoFlujoFirmaBaremacion{
				Consulta: consulta, VersionEsperada: 1,
				PropietarioRef: "worker-e2e-segundo", Duracion: time.Minute,
			},
		); !errors.Is(err, puertosbolsa.ErrFlujoFirmaBaremacionOcupado) {
			t.Fatalf(
				"segundo arrendamiento concurrente no rechazado: %v diagnostico=%v",
				err,
				diagnosticarArrendamientoFlujoFirmaPostgreSQLE2E(
					ctx,
					pool,
					consulta,
				),
			)
		}
		if err := repositorio.LiberarArrendamientoFlujoFirmaBaremacion(
			ctx,
			puertosbolsa.SolicitudLiberarArrendamientoFlujoFirmaBaremacion{
				Arrendamiento: primero,
			},
		); err != nil {
			t.Fatal(err)
		}
		segundo := adquirirFlujoFirmaPostgreSQLE2E(
			t,
			ctx,
			repositorio,
			consulta,
			1,
			"worker-e2e-segundo",
		)
		if segundo.SecuenciaCercado <= primero.SecuenciaCercado {
			t.Fatal("el cercado no avanzó tras liberar y readquirir")
		}
		siguiente := siguienteFlujoFirmaPostgreSQLE2E(t, actual)
		guardado, err := repositorio.GuardarFlujoFirmaBaremacion(
			ctx,
			puertosbolsa.SolicitudGuardarFlujoFirmaBaremacion{
				VersionEsperada: 1,
				Arrendamiento:   segundo,
				Siguiente:       siguiente,
			},
		)
		if err != nil || guardado.Version != 2 {
			t.Fatalf(
				"guardar versión 2: resultado=%+v error=%v diagnostico=%v",
				guardado,
				err,
				diagnosticarTransicionFlujoFirmaPostgreSQLE2E(
					ctx,
					repositorio,
					segundo,
					actual,
					siguiente,
				),
			)
		}
		completado := completarPreparacionFlujoFirmaPostgreSQLE2E(
			t,
			guardado,
		)
		guardado, err = repositorio.GuardarFlujoFirmaBaremacion(
			ctx,
			puertosbolsa.SolicitudGuardarFlujoFirmaBaremacion{
				VersionEsperada: 2,
				Arrendamiento:   segundo,
				Siguiente:       completado,
			},
		)
		if err != nil || guardado.Version != 3 {
			t.Fatalf(
				"completar preparación de firma: resultado=%+v error=%v",
				guardado,
				err,
			)
		}
		if err := repositorio.LiberarArrendamientoFlujoFirmaBaremacion(
			ctx,
			puertosbolsa.SolicitudLiberarArrendamientoFlujoFirmaBaremacion{
				Arrendamiento: segundo,
			},
		); err != nil {
			t.Fatal(err)
		}
	case "verificar":
		actual, err := repositorio.ObtenerFlujoFirmaBaremacion(ctx, consulta)
		if err != nil || actual.Version != 3 ||
			len(actual.PuntosControl) != 1 ||
			actual.PuntosControl[0].Estado !=
				puertosbolsa.EstadoPuntoControlFirmaCompletado ||
			actual.ProyeccionLanzamiento == nil {
			t.Fatalf("estado no durable: resultado=%+v error=%v", actual, err)
		}
	default:
		t.Fatalf("fase E2E desconocida: %q", fase)
	}
}

func diagnosticarTransicionFlujoFirmaPostgreSQLE2E(
	ctx context.Context,
	repositorio *RepositorioFlujosFirmaBaremacionPostgreSQL,
	arrendamiento puertosbolsa.ArrendamientoFlujoFirmaBaremacion,
	anterior, siguiente puertosbolsa.ExpedienteFlujoFirmaBaremacion,
) error {
	dsn := os.Getenv(variableDSNAdminFlujoFirmaPostgreSQLE2E)
	if dsn == "" {
		return errors.New("DSN administrador de diagnóstico ausente")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	documentoAnterior, cifradoAnterior, err :=
		serializarExpedienteFlujoFirmaPostgreSQL(anterior)
	if err != nil {
		return err
	}
	defer borrarBytesPostgreSQL(documentoAnterior, cifradoAnterior)
	documentoSiguiente, cifradoSiguiente, err :=
		serializarExpedienteFlujoFirmaPostgreSQL(siguiente)
	if err != nil {
		return err
	}
	defer borrarBytesPostgreSQL(documentoSiguiente, cifradoSiguiente)
	var expedienteValido, transicionValida, instanteValido bool
	err = pool.QueryRow(ctx, `
		SELECT vec_bolsa_firma.expediente_valido($1, $2),
		       vec_bolsa_firma.transicion_valida($3, $1),
		       vec_bolsa_firma.instante_utc_valido($1 ->> 'actualizado_en')`,
		documentoSiguiente,
		cifradoSiguiente,
		documentoAnterior,
	).Scan(&expedienteValido, &transicionValida, &instanteValido)
	if err != nil {
		return err
	}
	operacion, err := serializarOperacionArrendamientoFlujoFirmaPostgreSQL(
		arrendamiento,
		anterior.Version,
	)
	if err != nil {
		return err
	}
	defer borrarBytesPostgreSQL(operacion)
	huellaToken, _, err := repositorio.operarHMACToken(
		arrendamiento.Token,
		nil,
	)
	if err != nil {
		return err
	}
	defer borrarBytesPostgreSQL(huellaToken)
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var resultado string
	var documentoPersistido, cifradoPersistido []byte
	err = tx.QueryRow(ctx, `
		SELECT resultado, expediente_documento, estado_cifrado
		  FROM vec_bolsa_firma.guardar_flujo_v1($1, $2, $3, $4)`,
		operacion,
		documentoSiguiente,
		cifradoSiguiente,
		huellaToken,
	).Scan(&resultado, &documentoPersistido, &cifradoPersistido)
	if err != nil {
		return err
	}
	persistido, errorDecodificacion := repositorio.decodificarYVerificar(
		ctx,
		documentoPersistido,
		cifradoPersistido,
	)
	return fmt.Errorf(
		"expediente_valido=%t transicion_valida=%t instante_valido=%t resultado=%s decodificacion=%v exacto=%t",
		expedienteValido,
		transicionValida,
		instanteValido,
		resultado,
		errorDecodificacion,
		expedientesFlujoFirmaPostgreSQLExactos(siguiente, persistido),
	)
}

func diagnosticarLecturaFlujoFirmaPostgreSQLE2E(
	ctx context.Context,
	pool *pgxpool.Pool,
	consulta puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion,
) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var documento, cifrado []byte
	return tx.QueryRow(ctx, `
		SELECT expediente_documento, estado_cifrado
		  FROM vec_bolsa_firma.obtener_flujo_v1($1, $2, $3)`,
		consulta.FlujoRef,
		consulta.IndiceIdempotenciaHMAC,
		consulta.VinculoActorHMAC,
	).Scan(&documento, &cifrado)
}

func diagnosticarArrendamientoFlujoFirmaPostgreSQLE2E(
	ctx context.Context,
	pool *pgxpool.Pool,
	consulta puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion,
) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	operacion, err := json.Marshal(operacionArrendamientoFlujoFirmaPostgreSQL{
		Esquema:  esquemaOperacionArrendamientoFlujoFirmaPostgreSQLV1,
		FlujoRef: consulta.FlujoRef, IndiceIdempotenciaHMAC: consulta.IndiceIdempotenciaHMAC,
		VinculoActorHMAC: consulta.VinculoActorHMAC, VersionEsperada: "1",
		PropietarioRef: "worker-e2e-diagnostico", DuracionMicrosegundos: "60000000",
	})
	if err != nil {
		return err
	}
	var resultado string
	var secuencia pgtype.Text
	var documento, cifrado []byte
	var expira pgtype.Timestamptz
	err = tx.QueryRow(ctx, `
		SELECT resultado, expediente_documento, estado_cifrado,
		       secuencia_cercado, expira_en
		  FROM vec_bolsa_firma.adquirir_arrendamiento_v1($1, $2)`,
		operacion,
		make([]byte, 32),
	).Scan(&resultado, &documento, &cifrado, &secuencia, &expira)
	if err != nil {
		return err
	}
	return fmt.Errorf("resultado SQL inesperado: %s", resultado)
}

func expedienteFlujoFirmaPostgreSQLE2E(
	t *testing.T,
) puertosbolsa.ExpedienteFlujoFirmaBaremacion {
	t.Helper()
	expediente := expedienteFlujoFirmaPostgreSQLPrueba(t)
	expediente.FlujoRef = "flujo-firma-postgresql-e2e-001"
	expediente.ProcesoRef = "proceso-firma-postgresql-e2e-001"
	expediente.SolicitudRef = "solicitud-firma-postgresql-e2e-001"
	expediente.BaremacionMeritoRef = "baremacion-firma-postgresql-e2e-001"
	expediente.DecisionRef = "decision-firma-postgresql-e2e-001"
	expediente.SelloEstadoHMAC = ""
	return sellarFlujoFirmaPostgreSQLE2E(t, expediente)
}

func siguienteFlujoFirmaPostgreSQLE2E(
	t *testing.T,
	anterior puertosbolsa.ExpedienteFlujoFirmaBaremacion,
) puertosbolsa.ExpedienteFlujoFirmaBaremacion {
	t.Helper()
	siguiente, err := anterior.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	siguiente.Version = 2
	siguiente.ActualizadoEn = time.Now().UTC().Add(-time.Second)
	siguiente.PuntosControl = append(
		siguiente.PuntosControl,
		puertosbolsa.PuntoControlFirmaBaremacion{
			Paso:                  puertosbolsa.PasoPrepararFirmaBaremacion,
			Estado:                puertosbolsa.EstadoPuntoControlFirmaDeclarado,
			EfectoRef:             "efecto-firma-postgresql-e2e-001",
			ClaveIdempotenciaHMAC: hmacFlujoFirmaPostgreSQLPrueba("7"),
			DeclaradoEn:           siguiente.ActualizadoEn,
		},
	)
	siguiente.SelloEstadoHMAC = ""
	return sellarFlujoFirmaPostgreSQLE2E(t, siguiente)
}

func completarPreparacionFlujoFirmaPostgreSQLE2E(
	t *testing.T,
	anterior puertosbolsa.ExpedienteFlujoFirmaBaremacion,
) puertosbolsa.ExpedienteFlujoFirmaBaremacion {
	t.Helper()
	siguiente, err := anterior.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	completadoEn := time.Now().UTC().Add(-500 * time.Millisecond)
	siguiente.Version = anterior.Version + 1
	siguiente.ActualizadoEn = completadoEn
	siguiente.Estado = puertosbolsa.EstadoExpedienteFirmaPendienteInteraccion
	siguiente.PuntosControl[0].Estado =
		puertosbolsa.EstadoPuntoControlFirmaCompletado
	siguiente.PuntosControl[0].ResultadoRef =
		"resultado-preparacion-postgresql-e2e-001"
	siguiente.PuntosControl[0].HuellaResultadoSHA256 =
		strings.Repeat("8", 64)
	siguiente.PuntosControl[0].CompletadoEn = completadoEn
	siguiente.ProyeccionLanzamiento =
		&puertosbolsa.ProyeccionLanzamientoFirmaBaremacion{
			FlujoRef:              siguiente.FlujoRef,
			SesionFirmaRef:        "sesion-firma-postgresql-e2e-001",
			LanzamientoRef:        "lanzamiento-firma-postgresql-e2e-001",
			CanalLanzamientoClave: "autofirma",
			PreparadaEn:           completadoEn,
			ExpiraEn:              completadoEn.Add(5 * time.Minute),
		}
	siguiente.SelloEstadoHMAC = ""
	return sellarFlujoFirmaPostgreSQLE2E(t, siguiente)
}

func sellarFlujoFirmaPostgreSQLE2E(
	t *testing.T,
	expediente puertosbolsa.ExpedienteFlujoFirmaBaremacion,
) puertosbolsa.ExpedienteFlujoFirmaBaremacion {
	t.Helper()
	preparado, canonica, err := expediente.PrepararSellado()
	if err != nil {
		t.Fatal(err)
	}
	contenido := canonica.Revelar()
	defer borrarBytesPostgreSQL(contenido)
	mac := hmac.New(sha256.New, claveEstadoFlujoFirmaPostgreSQLE2E[:])
	_, _ = mac.Write(contenido)
	sello := "hmac-sha256:flujo_firma_postgresql_v1:" +
		hex.EncodeToString(mac.Sum(nil))
	sellado, err := preparado.IncorporarSello(sello)
	if err != nil {
		t.Fatal(err)
	}
	return sellado
}

func adquirirFlujoFirmaPostgreSQLE2E(
	t *testing.T,
	ctx context.Context,
	repositorio *RepositorioFlujosFirmaBaremacionPostgreSQL,
	consulta puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion,
	version uint64,
	propietario string,
) puertosbolsa.ArrendamientoFlujoFirmaBaremacion {
	t.Helper()
	resultado, err := repositorio.AdquirirArrendamientoFlujoFirmaBaremacion(
		ctx,
		puertosbolsa.SolicitudAdquirirArrendamientoFlujoFirmaBaremacion{
			Consulta: consulta, VersionEsperada: version,
			PropietarioRef: propietario, Duracion: time.Minute,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado.Arrendamiento
}
